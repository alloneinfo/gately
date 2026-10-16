package docker

// 容器文件管理。读取、下载、写入、新建目录、上传使用 Engine 的归档接口（/containers/{id}/archive 的
// GET / PUT / HEAD），容器停止时同样可用；重命名与删除没有对应的归档能力，通过 exec 执行 mv / rm -rf，
// 因此要求容器处于运行状态。
//
// 目录列表的取舍：GET archive 对目录会递归传输整个目录树，列出 / 或 /usr 这类大目录时无法只取第一层，
// 且 tar 是深度优先排列，第一层条目与深层条目交错，无法提前中止。所以运行中的容器优先用 exec 执行
// find + stat 只列第一层（busybox 与 GNU 都支持）；容器未运行，或容器内缺少 sh / find / stat（如 distroless 镜像）时，
// 回退到归档方案，并限制最多读取 archiveListMaxBytes 字节与 archiveListMaxEntries 个条目，超出部分被截断。
//
// 安全：所有路径必须是绝对路径并经过 path.Clean；解析归档时拒绝包含 ".." 的条目名；
// 上传的文件名只取最后一段。归档内容从不写入宿主机文件系统。

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/wangwanjie/gately/internal/httpx"
	"github.com/wangwanjie/gately/internal/logbuf"
)

const (
	archiveListMaxBytes   = 64 << 20 // 归档方案列目录时最多读取的字节数
	archiveListMaxEntries = 20000    // 列目录最多返回的条目数
	readFileLimit         = 1 << 20  // 文本读取上限 1MiB
	uploadMaxBytes        = 2 << 30  // 单次上传请求体上限
	fileOpTimeout         = 2 * time.Minute
)

func (m *Module) registerContainerFileRoutes(add func(pattern string, h http.HandlerFunc)) {
	add("GET /containers/{id}/files/list", m.handleFileList)
	add("GET /containers/{id}/files/read", m.handleFileRead)
	add("GET /containers/{id}/files/download", m.handleFileDownload)
	add("POST /containers/{id}/files/write", m.handleFileWrite)
	add("POST /containers/{id}/files/mkdir", m.handleFileMkdir)
	add("POST /containers/{id}/files/rename", m.handleFileRename)
	add("POST /containers/{id}/files/upload", m.handleFileUpload)
	add("DELETE /containers/{id}/files", m.handleFileDelete)
}

// ---------- 路径与名称安全 ----------

// cleanContainerPath 校验并规整容器内路径：必须是绝对路径，不含 NUL。
func cleanContainerPath(p string) (string, error) {
	if p == "" {
		return "", errors.New("路径不能为空")
	}
	if strings.ContainsRune(p, 0) {
		return "", errors.New("路径包含非法字符")
	}
	if !strings.HasPrefix(p, "/") {
		return "", errors.New("路径必须是容器内的绝对路径")
	}
	return path.Clean(p), nil
}

// safeTarRel 规整归档里的条目名，返回不带前导 ./ 与 / 的相对路径。名称含 ".." 段时返回 false，防止路径穿越。
func safeTarRel(name string) (string, bool) {
	name = strings.TrimPrefix(name, "./")
	name = strings.TrimLeft(name, "/")
	for _, seg := range strings.Split(name, "/") {
		if seg == ".." {
			return "", false
		}
	}
	name = path.Clean(name)
	if name == "." {
		return "", true
	}
	return name, true
}

// uploadFileName 取上传文件名的最后一段，拒绝空名与 . / ..。
func uploadFileName(name string) (string, error) {
	name = strings.ReplaceAll(name, "\\", "/")
	base := path.Base(name)
	if base == "" || base == "." || base == ".." || base == "/" || strings.ContainsRune(base, 0) {
		return "", fmt.Errorf("文件名 %q 无效", name)
	}
	return base, nil
}

// ---------- 归档接口封装 ----------

// pathStat 是 HEAD archive 返回的路径信息。
type pathStat struct {
	Name       string      `json:"name"`
	Size       int64       `json:"size"`
	Mode       os.FileMode `json:"mode"`
	Mtime      time.Time   `json:"mtime"`
	LinkTarget string      `json:"linkTarget"`
}

// fileStat 读取容器内路径的信息，路径不存在时返回 404 的 *APIError。
func (c *Client) fileStat(ctx context.Context, id, p string) (pathStat, error) {
	resp, err := c.rawRequest(ctx, http.MethodHead, "/containers/"+url.PathEscape(id)+"/archive?path="+url.QueryEscape(p), "", nil, nil)
	if err != nil {
		return pathStat{}, err
	}
	defer resp.Body.Close()
	raw := resp.Header.Get("X-Docker-Container-Path-Stat")
	if raw == "" {
		return pathStat{}, errors.New("Docker 未返回路径信息")
	}
	b, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return pathStat{}, err
	}
	var st pathStat
	if err := json.Unmarshal(b, &st); err != nil {
		return pathStat{}, err
	}
	return st, nil
}

// fileResolve 跟随符号链接，返回最终路径及其信息（最多 8 层）。
func (c *Client) fileResolve(ctx context.Context, id, p string) (string, pathStat, error) {
	for i := 0; i < 8; i++ {
		st, err := c.fileStat(ctx, id, p)
		if err != nil {
			return p, st, err
		}
		if st.Mode&os.ModeSymlink == 0 || st.LinkTarget == "" {
			return p, st, nil
		}
		target := st.LinkTarget
		if !strings.HasPrefix(target, "/") {
			target = path.Join(path.Dir(p), target)
		}
		p = path.Clean(target)
	}
	return p, pathStat{}, errors.New("符号链接层级过深")
}

func (c *Client) fileArchiveGet(ctx context.Context, id, p string) (io.ReadCloser, error) {
	resp, err := c.rawRequest(ctx, http.MethodGet, "/containers/"+url.PathEscape(id)+"/archive?path="+url.QueryEscape(p), "", nil, nil)
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}

// fileArchivePut 把 tar 流解压到容器内的目录 dir。
func (c *Client) fileArchivePut(ctx context.Context, id, dir string, body io.Reader) error {
	q := url.Values{"path": {dir}, "noOverwriteDirNonDir": {"1"}}
	resp, err := c.rawRequest(ctx, http.MethodPut, "/containers/"+url.PathEscape(id)+"/archive?"+q.Encode(), "application/x-tar", body, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	return nil
}

// fileOwner 读取已有文件的属主，只解析 tar 头后立即断开，不会传输整个文件。
func (c *Client) fileOwner(ctx context.Context, id, p string) (uid, gid int, ok bool) {
	cctx, cancel := context.WithCancel(ctx)
	defer cancel()
	body, err := c.fileArchiveGet(cctx, id, p)
	if err != nil {
		return 0, 0, false
	}
	defer body.Close()
	hdr, err := tar.NewReader(body).Next()
	if err != nil {
		return 0, 0, false
	}
	return hdr.Uid, hdr.Gid, true
}

// putFile 把数据写入容器内 dir/name：文件已存在时保留其权限与属主，新文件为 0644，属主 root。
func (c *Client) putFile(ctx context.Context, id, dir, name string, size int64, r io.Reader) error {
	full := path.Join(dir, name)
	mode, uid, gid := int64(0o644), 0, 0
	if rp, st, err := c.fileResolve(ctx, id, full); err == nil {
		if st.Mode.IsDir() {
			return fmt.Errorf("%s 是目录", full)
		}
		mode = int64(st.Mode.Perm())
		if u, g, ok := c.fileOwner(ctx, id, rp); ok {
			uid, gid = u, g
		}
		// 目标是符号链接时写入链接指向的真实文件
		dir, name = path.Dir(rp), path.Base(rp)
	}
	pr, pw := io.Pipe()
	go func() {
		tw := tar.NewWriter(pw)
		err := tw.WriteHeader(&tar.Header{Typeflag: tar.TypeReg, Name: name, Mode: mode, Uid: uid, Gid: gid, Size: size, ModTime: time.Now()})
		if err == nil {
			_, err = io.Copy(tw, r)
		}
		if err == nil {
			err = tw.Close()
		}
		pw.CloseWithError(err)
	}()
	err := c.fileArchivePut(ctx, id, dir, pr)
	pr.Close()
	return err
}

// ---------- 列目录 ----------

// FileEntry 是目录列表中的一项。
type FileEntry struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	Dir      bool   `json:"dir"`
	Size     int64  `json:"size"`
	Mode     string `json:"mode"`     // 形如 drwxr-xr-x
	Modified string `json:"modified"` // RFC3339
	Link     string `json:"link,omitempty"`
}

// sortEntries 目录在前，其余按名称排序。
func sortEntries(list []FileEntry) {
	sort.SliceStable(list, func(i, j int) bool {
		if list[i].Dir != list[j].Dir {
			return list[i].Dir
		}
		return strings.ToLower(list[i].Name) < strings.ToLower(list[j].Name)
	})
}

// listScript 只列出 $1 下的第一层条目。F 行为 stat 信息，L 行为符号链接目标，D 行表示链接指向目录。
const listScript = `cd "$1" 2>/dev/null || exit 3
find . -mindepth 1 -maxdepth 1 -exec stat -c 'F|%F|%s|%a|%Y|%n' {} +
find . -mindepth 1 -maxdepth 1 -type l -exec sh -c 'for f; do printf "L|%s|%s\n" "$f" "$(readlink "$f")"; if [ -d "$f" ]; then printf "D|%s\n" "$f"; fi; done' sh {} +
exit 0`

// modeFromStat 由 stat 的文件类型文本和八进制权限生成 os.FileMode。
func modeFromStat(typ, octal string) os.FileMode {
	perm, _ := strconv.ParseUint(octal, 8, 32)
	mode := os.FileMode(perm & 0o777)
	if perm&0o4000 != 0 {
		mode |= os.ModeSetuid
	}
	if perm&0o2000 != 0 {
		mode |= os.ModeSetgid
	}
	if perm&0o1000 != 0 {
		mode |= os.ModeSticky
	}
	switch {
	case typ == "directory":
		mode |= os.ModeDir
	case typ == "symbolic link":
		mode |= os.ModeSymlink
	case strings.Contains(typ, "fifo"):
		mode |= os.ModeNamedPipe
	case strings.Contains(typ, "socket"):
		mode |= os.ModeSocket
	case strings.Contains(typ, "character"):
		mode |= os.ModeDevice | os.ModeCharDevice
	case strings.Contains(typ, "block"):
		mode |= os.ModeDevice
	}
	return mode
}

// parseListOutput 解析 listScript 的输出。
func parseListOutput(dir string, out []byte) []FileEntry {
	links := map[string]string{}
	linkDirs := map[string]bool{}
	var files [][]string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimRight(line, "\r")
		switch {
		case strings.HasPrefix(line, "F|"):
			parts := strings.SplitN(line, "|", 6)
			if len(parts) == 6 {
				files = append(files, parts)
			}
		case strings.HasPrefix(line, "L|"):
			parts := strings.SplitN(line, "|", 3)
			if len(parts) == 3 {
				links[strings.TrimPrefix(parts[1], "./")] = parts[2]
			}
		case strings.HasPrefix(line, "D|"):
			linkDirs[strings.TrimPrefix(line[2:], "./")] = true
		}
	}
	list := make([]FileEntry, 0, len(files))
	for _, p := range files {
		name := strings.TrimPrefix(p[5], "./")
		if name == "" || name == "." {
			continue
		}
		size, _ := strconv.ParseInt(p[2], 10, 64)
		mtime, _ := strconv.ParseInt(p[4], 10, 64)
		mode := modeFromStat(p[1], p[3])
		e := FileEntry{Name: name, Path: path.Join(dir, name), Dir: mode.IsDir() || linkDirs[name], Size: size, Mode: mode.String(), Modified: time.Unix(mtime, 0).UTC().Format(time.RFC3339)}
		if target, ok := links[name]; ok {
			e.Link = target
		}
		list = append(list, e)
		if len(list) >= archiveListMaxEntries {
			break
		}
	}
	sortEntries(list)
	return list
}

// listViaExec 用 exec 列出运行中容器的目录。notFound 表示目录不存在或无权进入。
func (c *Client) listViaExec(ctx context.Context, id, dir string) (list []FileEntry, notFound bool, err error) {
	so, _, code, err := c.execRun(ctx, id, []string{"sh", "-c", listScript, "sh", dir}, "")
	if err != nil {
		return nil, false, err
	}
	if code == 3 {
		return nil, true, nil
	}
	if code != 0 && len(so) == 0 {
		return nil, false, fmt.Errorf("容器内列目录失败（退出码 %d）", code)
	}
	return parseListOutput(dir, so), false, nil
}

// listViaArchive 通过归档接口列出目录第一层，最多读取 archiveListMaxBytes 字节，超出即截断。
func (c *Client) listViaArchive(ctx context.Context, id, dir string) ([]FileEntry, error) {
	rp, st, err := c.fileResolve(ctx, id, dir)
	if err != nil {
		return nil, err
	}
	if !st.Mode.IsDir() {
		return nil, fmt.Errorf("%s 不是目录", dir)
	}
	cctx, cancel := context.WithCancel(ctx)
	defer cancel()
	body, err := c.fileArchiveGet(cctx, id, rp)
	if err != nil {
		return nil, err
	}
	defer body.Close()
	return entriesFromTar(io.LimitReader(body, archiveListMaxBytes), rp, dir), nil
}

// entriesFromTar 从目录的 tar 流中取出第一层条目；读到流末尾、出错或达到条目上限时停止。
// root 为被归档的目录（决定条目名前缀），dir 为展示用路径。
func entriesFromTar(r io.Reader, root, dir string) []FileEntry {
	tr := tar.NewReader(r)
	prefix := ""
	if root != "/" {
		prefix = path.Base(root)
	}
	var list []FileEntry
	for {
		hdr, err := tr.Next()
		if err != nil {
			break
		}
		rel, ok := safeTarRel(hdr.Name)
		if !ok || rel == "" {
			continue
		}
		if prefix != "" {
			switch {
			case rel == prefix:
				continue
			case strings.HasPrefix(rel, prefix+"/"):
				rel = rel[len(prefix)+1:]
			}
		}
		if strings.Contains(rel, "/") {
			continue
		}
		e := FileEntry{Name: rel, Path: path.Join(dir, rel), Dir: hdr.Typeflag == tar.TypeDir, Size: hdr.Size, Mode: hdr.FileInfo().Mode().String(), Modified: hdr.ModTime.UTC().Format(time.RFC3339)}
		if hdr.Typeflag == tar.TypeSymlink {
			e.Link = hdr.Linkname
		}
		list = append(list, e)
		if len(list) >= archiveListMaxEntries {
			break
		}
	}
	sortEntries(list)
	return list
}

// isRunning 判断容器是否在运行。
func isRunning(ctx context.Context, c *Client, id string) (bool, error) {
	info, err := c.Inspect(ctx, id)
	if err != nil {
		return false, err
	}
	var st struct {
		Running bool `json:"Running"`
		Paused  bool `json:"Paused"`
	}
	decodeField(info, "State", &st)
	return st.Running && !st.Paused, nil
}

// fileErr 把归档接口的错误转为更易懂的提示。
func fileErr(w http.ResponseWriter, err error) {
	var ae *APIError
	if errors.As(err, &ae) && ae.Status == http.StatusNotFound {
		httpx.Error(w, http.StatusNotFound, "路径不存在: %s", ae.Message)
		return
	}
	writeCtrErr(w, err)
}

// fileReq 校验路径参数并取得客户端。
func (m *Module) fileReq(w http.ResponseWriter, raw string) (*Client, string, bool) {
	p, err := cleanContainerPath(raw)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "%v", err)
		return nil, "", false
	}
	c, ok := m.clientOrFail(w)
	if !ok {
		return nil, "", false
	}
	return c, p, true
}

func (m *Module) handleFileList(w http.ResponseWriter, r *http.Request) {
	raw := r.URL.Query().Get("path")
	if raw == "" {
		raw = "/"
	}
	c, dir, ok := m.fileReq(w, raw)
	if !ok {
		return
	}
	id := r.PathValue("id")
	ctx, cancel := context.WithTimeout(r.Context(), fileOpTimeout)
	defer cancel()
	running, err := isRunning(ctx, c, id)
	if err != nil {
		fileErr(w, err)
		return
	}
	if running {
		list, notFound, err := c.listViaExec(ctx, id, dir)
		if notFound {
			httpx.Error(w, http.StatusNotFound, "目录不存在或无权访问: %s", dir)
			return
		}
		if err == nil {
			httpx.OK(w, nonNilEntries(list))
			return
		}
		// 容器内缺少 sh / find / stat 等工具时回退到归档方案
	}
	list, err := c.listViaArchive(ctx, id, dir)
	if err != nil {
		fileErr(w, err)
		return
	}
	httpx.OK(w, nonNilEntries(list))
}

func nonNilEntries(l []FileEntry) []FileEntry {
	if l == nil {
		return []FileEntry{}
	}
	return l
}

// ---------- 读取与下载 ----------

// trimPartialRune 去掉被截断的末尾不完整 UTF-8 字符。
func trimPartialRune(b []byte) []byte {
	for i := 0; i < 3 && len(b) > 0 && !utf8.Valid(b); i++ {
		b = b[:len(b)-1]
	}
	return b
}

func (m *Module) handleFileRead(w http.ResponseWriter, r *http.Request) {
	c, p, ok := m.fileReq(w, r.URL.Query().Get("path"))
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), fileOpTimeout)
	defer cancel()
	rp, st, err := c.fileResolve(ctx, r.PathValue("id"), p)
	if err != nil {
		fileErr(w, err)
		return
	}
	if !st.Mode.IsRegular() {
		httpx.Error(w, http.StatusBadRequest, "%s 不是普通文件", p)
		return
	}
	body, err := c.fileArchiveGet(ctx, r.PathValue("id"), rp)
	if err != nil {
		fileErr(w, err)
		return
	}
	defer body.Close()
	tr := tar.NewReader(body)
	hdr, err := tr.Next()
	if err != nil {
		httpx.Error(w, http.StatusBadGateway, "读取归档失败: %v", err)
		return
	}
	data, err := io.ReadAll(io.LimitReader(tr, readFileLimit+1))
	if err != nil {
		httpx.Error(w, http.StatusBadGateway, "读取文件失败: %v", err)
		return
	}
	truncated := len(data) > readFileLimit
	if truncated {
		data = trimPartialRune(data[:readFileLimit])
	}
	if bytes.IndexByte(data, 0) >= 0 || !utf8.Valid(data) {
		httpx.Error(w, http.StatusBadRequest, "该文件是二进制文件，不支持文本查看，请下载")
		return
	}
	httpx.OK(w, map[string]any{"content": string(data), "size": hdr.Size, "truncated": truncated})
}

// contentDisposition 生成兼容中文名的下载头。
func contentDisposition(name string) string {
	return "attachment; filename*=UTF-8''" + url.PathEscape(name)
}

func (m *Module) handleFileDownload(w http.ResponseWriter, r *http.Request) {
	c, p, ok := m.fileReq(w, r.URL.Query().Get("path"))
	if !ok {
		return
	}
	id := r.PathValue("id")
	// 下载可能很大，只受请求本身取消的约束
	rp, st, err := c.fileResolve(r.Context(), id, p)
	if err != nil {
		fileErr(w, err)
		return
	}
	body, err := c.fileArchiveGet(r.Context(), id, rp)
	if err != nil {
		fileErr(w, err)
		return
	}
	defer body.Close()
	name := path.Base(p)
	if st.Mode.IsDir() {
		if p == "/" {
			name = "root"
		}
		// 目录：Engine 返回的 tar 直接压成 tar.gz
		w.Header().Set("Content-Type", "application/gzip")
		w.Header().Set("Content-Disposition", contentDisposition(name+".tar.gz"))
		gz := gzip.NewWriter(w)
		_, _ = io.Copy(gz, body)
		_ = gz.Close()
		return
	}
	tr := tar.NewReader(body)
	hdr, err := tr.Next()
	if err != nil {
		httpx.Error(w, http.StatusBadGateway, "读取归档失败: %v", err)
		return
	}
	if hdr.Typeflag != tar.TypeReg {
		httpx.Error(w, http.StatusBadRequest, "%s 不是普通文件", p)
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", contentDisposition(name))
	w.Header().Set("Content-Length", strconv.FormatInt(hdr.Size, 10))
	_, _ = io.Copy(w, tr)
}

// ---------- 写入、新建、上传 ----------

func (m *Module) handleFileWrite(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "%v", err)
		return
	}
	c, p, ok := m.fileReq(w, req.Path)
	if !ok {
		return
	}
	if p == "/" {
		httpx.Error(w, http.StatusBadRequest, "不能写入根目录")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), fileOpTimeout)
	defer cancel()
	if err := c.putFile(ctx, r.PathValue("id"), path.Dir(p), path.Base(p), int64(len(req.Content)), strings.NewReader(req.Content)); err != nil {
		fileErr(w, err)
		return
	}
	m.logs.Add(logbuf.LevelInfo, "", fmt.Sprintf("容器 %s 写入文件 %s", shortID(r.PathValue("id")), p))
	httpx.OK(w, map[string]bool{"ok": true})
}

func (m *Module) handleFileMkdir(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path string `json:"path"`
	}
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "%v", err)
		return
	}
	c, p, ok := m.fileReq(w, req.Path)
	if !ok {
		return
	}
	if p == "/" {
		httpx.Error(w, http.StatusBadRequest, "根目录已存在")
		return
	}
	id := r.PathValue("id")
	ctx, cancel := context.WithTimeout(r.Context(), fileOpTimeout)
	defer cancel()
	if _, err := c.fileStat(ctx, id, p); err == nil {
		httpx.Error(w, http.StatusConflict, "%s 已存在", p)
		return
	}
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	if err := tw.WriteHeader(&tar.Header{Typeflag: tar.TypeDir, Name: path.Base(p) + "/", Mode: 0o755, ModTime: time.Now()}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "%v", err)
		return
	}
	_ = tw.Close()
	if err := c.fileArchivePut(ctx, id, path.Dir(p), &buf); err != nil {
		fileErr(w, err)
		return
	}
	httpx.OK(w, map[string]bool{"ok": true})
}

func (m *Module) handleFileUpload(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, uploadMaxBytes)
	// 超过 32MiB 的部分由标准库写入临时文件，不占内存
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		httpx.Error(w, http.StatusBadRequest, "解析上传内容失败: %v", err)
		return
	}
	defer r.MultipartForm.RemoveAll()
	var dirRaw string
	if v := r.MultipartForm.Value["path"]; len(v) > 0 {
		dirRaw = v[0]
	}
	c, dir, ok := m.fileReq(w, dirRaw)
	if !ok {
		return
	}
	files := r.MultipartForm.File["file"]
	if len(files) == 0 {
		httpx.Error(w, http.StatusBadRequest, "没有上传文件")
		return
	}
	names := make([]string, 0, len(files))
	for _, fh := range files {
		name, err := uploadFileName(fh.Filename)
		if err != nil {
			httpx.Error(w, http.StatusBadRequest, "%v", err)
			return
		}
		names = append(names, name)
	}
	id := r.PathValue("id")
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Minute)
	defer cancel()
	// 目标目录必须存在且是目录
	rd, st, err := c.fileResolve(ctx, id, dir)
	if err != nil {
		fileErr(w, err)
		return
	}
	if !st.Mode.IsDir() {
		httpx.Error(w, http.StatusBadRequest, "%s 不是目录", dir)
		return
	}
	for i, fh := range files {
		if err := uploadOne(ctx, c, id, rd, names[i], fh); err != nil {
			fileErr(w, fmt.Errorf("上传 %s 失败: %w", names[i], err))
			return
		}
	}
	m.logs.Add(logbuf.LevelInfo, "", fmt.Sprintf("容器 %s 上传 %d 个文件到 %s", shortID(id), len(files), dir))
	httpx.OK(w, map[string]any{"ok": true, "count": len(files)})
}

func uploadOne(ctx context.Context, c *Client, id, dir, name string, fh *multipart.FileHeader) error {
	f, err := fh.Open()
	if err != nil {
		return err
	}
	defer f.Close()
	return c.putFile(ctx, id, dir, name, fh.Size, f)
}

// ---------- 重命名与删除（exec） ----------

func (m *Module) execFileOp(w http.ResponseWriter, r *http.Request, cmd []string, desc string) bool {
	c, ok := m.clientOrFail(w)
	if !ok {
		return false
	}
	id := r.PathValue("id")
	ctx, cancel := context.WithTimeout(r.Context(), fileOpTimeout)
	defer cancel()
	running, err := isRunning(ctx, c, id)
	if err != nil {
		writeCtrErr(w, err)
		return false
	}
	if !running {
		httpx.Error(w, http.StatusConflict, "容器未运行，无法%s（该操作需要在容器内执行命令，请先启动容器）", desc)
		return false
	}
	if _, err := c.execOK(ctx, id, cmd); err != nil {
		httpx.Error(w, http.StatusBadGateway, "%s失败: %v", desc, err)
		return false
	}
	return true
}

func (m *Module) handleFileRename(w http.ResponseWriter, r *http.Request) {
	var req struct {
		From string `json:"from"`
		To   string `json:"to"`
	}
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "%v", err)
		return
	}
	from, err1 := cleanContainerPath(req.From)
	to, err2 := cleanContainerPath(req.To)
	if err := errors.Join(err1, err2); err != nil {
		httpx.Error(w, http.StatusBadRequest, "%v", err)
		return
	}
	if from == "/" || to == "/" {
		httpx.Error(w, http.StatusBadRequest, "不能重命名根目录")
		return
	}
	if !m.execFileOp(w, r, []string{"mv", from, to}, "重命名") {
		return
	}
	m.logs.Add(logbuf.LevelInfo, "", fmt.Sprintf("容器 %s 重命名 %s -> %s", shortID(r.PathValue("id")), from, to))
	httpx.OK(w, map[string]bool{"ok": true})
}

func (m *Module) handleFileDelete(w http.ResponseWriter, r *http.Request) {
	p, err := cleanContainerPath(r.URL.Query().Get("path"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "%v", err)
		return
	}
	if p == "/" {
		httpx.Error(w, http.StatusBadRequest, "不能删除根目录")
		return
	}
	if !m.execFileOp(w, r, []string{"rm", "-rf", p}, "删除") {
		return
	}
	m.logs.Add(logbuf.LevelInfo, "", fmt.Sprintf("容器 %s 删除 %s", shortID(r.PathValue("id")), p))
	httpx.OK(w, map[string]bool{"ok": true})
}
