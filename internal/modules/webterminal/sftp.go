package webterminal

import (
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"mime"
	"mime/multipart"
	"net/http"
	"path"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unicode/utf8"

	"github.com/pkg/sftp"

	"github.com/wangwanjie/gately/internal/httpx"
	"github.com/wangwanjie/gately/internal/module"
)

// previewLimit 是文本预览的大小上限。
const previewLimit = 1 << 20

// apiError 携带 HTTP 状态码的业务错误。
type apiError struct {
	status int
	msg    string
}

func (e *apiError) Error() string { return e.msg }

func itoa(i int) string { return strconv.Itoa(i) }

// isNotExist 同时识别本地错误与 SFTP 状态码错误。
func isNotExist(err error) bool {
	if errors.Is(err, fs.ErrNotExist) {
		return true
	}
	var se *sftp.StatusError
	return errors.As(err, &se) && se.Code == 2
}

func isPermission(err error) bool {
	if errors.Is(err, fs.ErrPermission) {
		return true
	}
	var se *sftp.StatusError
	return errors.As(err, &se) && se.Code == 3
}

// writeErr 把错误映射为 HTTP 响应：不存在 404、权限 403、过大 413、类型不支持 415、其余 500。
func writeErr(w http.ResponseWriter, err error) {
	var ae *apiError
	switch {
	case errors.As(err, &ae):
		httpx.Error(w, ae.status, "%s", ae.msg)
	case isNotExist(err):
		httpx.Error(w, http.StatusNotFound, "路径不存在")
	case isPermission(err):
		httpx.Error(w, http.StatusForbidden, "没有权限执行该操作")
	default:
		httpx.Error(w, http.StatusInternalServerError, "操作失败: %v", err)
	}
}

func badRequest(msg string) error { return &apiError{http.StatusBadRequest, msg} }

// cleanPath 校验并规范化客户端传入的绝对路径。
func cleanPath(p string) (string, error) {
	if strings.TrimSpace(p) == "" {
		return "", badRequest("路径不能为空")
	}
	if strings.ContainsRune(p, 0) {
		return "", badRequest("路径包含非法字符")
	}
	if !strings.HasPrefix(p, "/") {
		return "", badRequest("路径必须是以 / 开头的绝对路径")
	}
	return path.Clean(p), nil
}

// fileEntry 是目录列表中的一项。
type fileEntry struct {
	Name    string    `json:"name"`
	Path    string    `json:"path"`
	IsDir   bool      `json:"isDir"`
	IsLink  bool      `json:"isLink"`
	Size    int64     `json:"size"`
	Mode    string    `json:"mode"`
	ModeStr string    `json:"modeStr"`
	ModTime time.Time `json:"modTime"`
	Owner   string    `json:"owner,omitempty"`
}

// handler 是已通过门禁、已取得文件系统的处理函数。
type handler func(w http.ResponseWriter, r *http.Request, fsys fileSystem) error

// withFS 包装处理函数：门禁 -> 解析连接 -> 取得文件系统 -> 执行 -> 统一错误输出。
func (m *Module) withFS(h handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !m.requireAccess(w, r) {
			return
		}
		key := r.PathValue("key")
		m.mu.Lock()
		var conn *Connection
		for i := range m.cfg.Connections {
			if m.cfg.Connections[i].Key == key {
				c := m.cfg.Connections[i]
				conn = &c
				break
			}
		}
		allowLocal := m.cfg.Settings.AllowLocalShell
		m.mu.Unlock()
		if conn == nil {
			httpx.Error(w, http.StatusNotFound, "连接不存在")
			return
		}
		var fsys fileSystem
		switch conn.Type {
		case TypeLocal:
			// 本机文件系统等同于本机 Shell 的权限，沿用同一开关
			if !allowLocal {
				httpx.Error(w, http.StatusForbidden, "本机 Shell 未开启，无法管理本机文件")
				return
			}
			fsys = &localFS{workDir: conn.Local.WorkDir}
		case TypeSSH:
			client, release, err := m.acquireSSH(r.Context(), key)
			if err != nil {
				httpx.Error(w, http.StatusBadGateway, "连接 SSH 失败: %v", err)
				return
			}
			defer release()
			sc, err := sftp.NewClient(client)
			if err != nil {
				httpx.Error(w, http.StatusBadGateway, "打开 SFTP 子系统失败: %v", err)
				return
			}
			defer sc.Close()
			fsys = &sftpFS{c: sc}
		default:
			httpx.Error(w, http.StatusBadRequest, "该连接类型不支持文件管理")
			return
		}
		if err := h(w, r, fsys); err != nil {
			writeErr(w, err)
		}
	}
}

// registerSFTP 注册 /sftp/{key}/... 文件管理接口。
func (m *Module) registerSFTP(r module.Router) {
	r.HandleFunc("GET /sftp/{key}/list", m.withFS(handleList))
	r.HandleFunc("GET /sftp/{key}/read", m.withFS(handleRead))
	r.HandleFunc("POST /sftp/{key}/write", m.withFS(handleWrite))
	r.HandleFunc("POST /sftp/{key}/upload", m.withFS(handleUpload))
	r.HandleFunc("POST /sftp/{key}/mkdir", m.withFS(handleMkdir))
	r.HandleFunc("POST /sftp/{key}/touch", m.withFS(handleTouch))
	r.HandleFunc("POST /sftp/{key}/rename", m.withFS(handleRename))
	r.HandleFunc("POST /sftp/{key}/copy", m.withFS(handleCopy))
	r.HandleFunc("POST /sftp/{key}/chmod", m.withFS(handleChmod))
	r.HandleFunc("DELETE /sftp/{key}/remove", m.withFS(handleRemove))
	r.HandleFunc("POST /sftp/{key}/compress", m.withFS(handleCompress))
	r.HandleFunc("POST /sftp/{key}/decompress", m.withFS(handleDecompress))
	r.HandleFunc("GET /sftp/{key}/preview-archive", m.withFS(handlePreviewArchive))
}

func handleList(w http.ResponseWriter, r *http.Request, fsys fileSystem) error {
	p := r.URL.Query().Get("path")
	if strings.TrimSpace(p) == "" {
		home, err := fsys.Home()
		if err != nil {
			return err
		}
		p = home
	}
	p, err := cleanPath(p)
	if err != nil {
		return err
	}
	fi, err := fsys.Stat(p)
	if err != nil {
		return err
	}
	if !fi.IsDir() {
		return badRequest("路径不是目录")
	}
	infos, err := fsys.ReadDir(p)
	if err != nil {
		return err
	}
	entries := make([]fileEntry, 0, len(infos))
	for _, info := range infos {
		e := fileEntry{
			Name:    info.Name(),
			Path:    path.Join(p, info.Name()),
			IsDir:   info.IsDir(),
			Size:    info.Size(),
			Mode:    strconv.FormatUint(uint64(info.Mode().Perm()), 8),
			ModeStr: info.Mode().String(),
			ModTime: info.ModTime(),
			Owner:   ownerOf(info),
		}
		if info.Mode()&fs.ModeSymlink != 0 {
			e.IsLink = true
			// 指向目录的链接按目录展示，悬空链接保持为文件
			if target, err := fsys.Stat(e.Path); err == nil {
				e.IsDir = target.IsDir()
				e.Size = target.Size()
			}
		}
		entries = append(entries, e)
	}
	sortEntries(entries)
	httpx.OK(w, map[string]any{"path": p, "parent": path.Dir(p), "entries": entries})
	return nil
}

// ownerOf 返回条目属主：SFTP 返回数字 uid，本机返回用户名。
func ownerOf(fi fs.FileInfo) string {
	if st, ok := fi.Sys().(*sftp.FileStat); ok {
		return strconv.FormatUint(uint64(st.UID), 10)
	}
	return localOwner(fi)
}

func handleRead(w http.ResponseWriter, r *http.Request, fsys fileSystem) error {
	p, err := cleanPath(r.URL.Query().Get("path"))
	if err != nil {
		return err
	}
	fi, err := fsys.Stat(p)
	if err != nil {
		return err
	}
	if fi.IsDir() {
		return badRequest("路径是目录")
	}
	download := r.URL.Query().Get("download") == "1"
	if !download && fi.Size() > previewLimit {
		return &apiError{http.StatusRequestEntityTooLarge, "文件超过 1MB，请下载后查看"}
	}
	f, err := fsys.Open(p)
	if err != nil {
		return err
	}
	defer f.Close()

	if download {
		name := path.Base(p)
		h := w.Header()
		h.Set("Content-Type", "application/octet-stream")
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": name}))
		// ServeContent 基于 Seek 提供 Range 与 If-Modified-Since 支持
		http.ServeContent(w, r, name, fi.ModTime(), f)
		return nil
	}

	data, err := io.ReadAll(io.LimitReader(f, previewLimit+1))
	if err != nil {
		return err
	}
	if len(data) > previewLimit {
		return &apiError{http.StatusRequestEntityTooLarge, "文件超过 1MB，请下载后查看"}
	}
	if isBinary(data) {
		return &apiError{http.StatusUnsupportedMediaType, "二进制文件不支持预览，请下载后查看"}
	}
	httpx.OK(w, map[string]any{
		"path": p, "name": path.Base(p), "size": len(data),
		"modTime": fi.ModTime(), "content": string(data),
	})
	return nil
}

// isBinary 含 NUL 字节或不是合法 UTF-8 时视为二进制。
func isBinary(data []byte) bool {
	for _, b := range data {
		if b == 0 {
			return true
		}
	}
	return !utf8.Valid(data)
}

func handleWrite(w http.ResponseWriter, r *http.Request, fsys fileSystem) error {
	var req struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := httpx.ReadJSON(r, &req); err != nil {
		return badRequest(err.Error())
	}
	p, err := cleanPath(req.Path)
	if err != nil {
		return err
	}
	if fi, err := fsys.Stat(p); err == nil && fi.IsDir() {
		return badRequest("目标是目录，无法写入")
	}
	out, err := fsys.Create(p)
	if err != nil {
		return err
	}
	_, err = io.WriteString(out, req.Content)
	if cerr := out.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return err
	}
	httpx.OK(w, map[string]any{"ok": true, "path": p, "size": len(req.Content)})
	return nil
}

// safeRelPath 把客户端给出的相对路径限制在目标目录内：统一分隔符并拒绝 ".." 片段。
func safeRelPath(rel string) (string, error) {
	rel = strings.ReplaceAll(rel, "\\", "/")
	if strings.ContainsRune(rel, 0) {
		return "", badRequest("文件名包含非法字符")
	}
	parts := []string{}
	for _, seg := range strings.Split(rel, "/") {
		switch seg {
		case "", ".":
		case "..":
			return "", badRequest("文件路径不允许包含 ..")
		default:
			parts = append(parts, seg)
		}
	}
	if len(parts) == 0 {
		return "", badRequest("文件名为空")
	}
	return strings.Join(parts, "/"), nil
}

// rawFilename 读取 part 未经 Base 处理的 filename，保留浏览器上传目录时携带的相对路径。
func rawFilename(part *multipart.Part) string {
	_, params, err := mime.ParseMediaType(part.Header.Get("Content-Disposition"))
	if err != nil {
		return part.FileName()
	}
	return params["filename"]
}

func handleUpload(w http.ResponseWriter, r *http.Request, fsys fileSystem) error {
	mr, err := r.MultipartReader()
	if err != nil {
		return badRequest("请求必须是 multipart/form-data")
	}
	var dest string
	var rels []string
	var uploaded []string
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return badRequest("读取上传数据失败: " + err.Error())
		}
		if part.FileName() == "" {
			val, _ := io.ReadAll(io.LimitReader(part, 64<<10))
			switch part.FormName() {
			case "path":
				dest = strings.TrimSpace(string(val))
			case "relativePaths":
				s := strings.TrimSpace(string(val))
				if strings.HasPrefix(s, "[") {
					var arr []string
					if json.Unmarshal([]byte(s), &arr) == nil {
						rels = append(rels, arr...)
						continue
					}
				}
				rels = append(rels, s)
			}
			continue
		}
		if dest == "" {
			return badRequest("path 字段必须位于 files 之前")
		}
		if len(uploaded) == 0 {
			d, err := cleanPath(dest)
			if err != nil {
				return err
			}
			dest = d
			fi, err := fsys.Stat(dest)
			if err != nil {
				return err
			}
			if !fi.IsDir() {
				return badRequest("上传目标不是目录")
			}
		}
		rel := rawFilename(part)
		if len(rels) > 0 && rels[0] != "" {
			rel = rels[0]
		}
		if len(rels) > 0 {
			rels = rels[1:]
		}
		rel, err = safeRelPath(rel)
		if err != nil {
			return err
		}
		target := path.Join(dest, rel)
		if err := mkdirAll(fsys, path.Dir(target)); err != nil {
			return err
		}
		if fi, err := fsys.Stat(target); err == nil && fi.IsDir() {
			return &apiError{http.StatusConflict, "已存在同名目录: " + rel}
		}
		out, err := fsys.Create(target)
		if err != nil {
			return err
		}
		_, err = io.Copy(out, part)
		if cerr := out.Close(); err == nil {
			err = cerr
		}
		if err != nil {
			_ = fsys.Remove(target)
			return err
		}
		uploaded = append(uploaded, target)
	}
	if len(uploaded) == 0 {
		return badRequest("没有收到文件")
	}
	httpx.OK(w, map[string]any{"ok": true, "count": len(uploaded), "files": uploaded})
	return nil
}

type pathReq struct {
	Path string `json:"path"`
}

func handleMkdir(w http.ResponseWriter, r *http.Request, fsys fileSystem) error {
	var req pathReq
	if err := httpx.ReadJSON(r, &req); err != nil {
		return badRequest(err.Error())
	}
	p, err := cleanPath(req.Path)
	if err != nil {
		return err
	}
	if ok, err := exists(fsys, p); err != nil {
		return err
	} else if ok {
		return &apiError{http.StatusConflict, "目标已存在"}
	}
	if err := mkdirAll(fsys, p); err != nil {
		return err
	}
	httpx.OK(w, map[string]any{"ok": true, "path": p})
	return nil
}

func handleTouch(w http.ResponseWriter, r *http.Request, fsys fileSystem) error {
	var req pathReq
	if err := httpx.ReadJSON(r, &req); err != nil {
		return badRequest(err.Error())
	}
	p, err := cleanPath(req.Path)
	if err != nil {
		return err
	}
	if ok, err := exists(fsys, p); err != nil {
		return err
	} else if ok {
		return &apiError{http.StatusConflict, "目标已存在"}
	}
	if err := mkdirAll(fsys, path.Dir(p)); err != nil {
		return err
	}
	out, err := fsys.Create(p)
	if err != nil {
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	httpx.OK(w, map[string]any{"ok": true, "path": p})
	return nil
}

func handleRename(w http.ResponseWriter, r *http.Request, fsys fileSystem) error {
	var req struct {
		From string `json:"from"`
		To   string `json:"to"`
	}
	if err := httpx.ReadJSON(r, &req); err != nil {
		return badRequest(err.Error())
	}
	from, err := cleanPath(req.From)
	if err != nil {
		return err
	}
	to, err := cleanPath(req.To)
	if err != nil {
		return err
	}
	if from == "/" {
		return &apiError{http.StatusForbidden, "禁止重命名根目录"}
	}
	if from == to {
		httpx.OK(w, map[string]any{"ok": true, "path": to})
		return nil
	}
	if isWithin(from, to) {
		return badRequest("不能把目录移动到自身内部")
	}
	if _, err := fsys.Lstat(from); err != nil {
		return err
	}
	if ok, err := exists(fsys, to); err != nil {
		return err
	} else if ok {
		return &apiError{http.StatusConflict, "目标已存在"}
	}
	if err := fsys.Rename(from, to); err != nil {
		if !errors.Is(err, syscall.EXDEV) {
			return err
		}
		// 跨设备移动：先复制再删除源
		if err := copyTree(fsys, from, to); err != nil {
			return err
		}
		if err := removeAll(fsys, from); err != nil {
			return err
		}
	}
	httpx.OK(w, map[string]any{"ok": true, "path": to})
	return nil
}

func handleCopy(w http.ResponseWriter, r *http.Request, fsys fileSystem) error {
	var req struct {
		From []string `json:"from"`
		To   string   `json:"to"`
	}
	if err := httpx.ReadJSON(r, &req); err != nil {
		return badRequest(err.Error())
	}
	if len(req.From) == 0 {
		return badRequest("未选择要复制的项目")
	}
	to, err := cleanPath(req.To)
	if err != nil {
		return err
	}
	fi, err := fsys.Stat(to)
	if err != nil {
		return err
	}
	if !fi.IsDir() {
		return badRequest("复制目标不是目录")
	}
	copied := make([]string, 0, len(req.From))
	for _, f := range req.From {
		src, err := cleanPath(f)
		if err != nil {
			return err
		}
		if src == "/" {
			return badRequest("不能复制根目录")
		}
		sfi, err := fsys.Lstat(src)
		if err != nil {
			return err
		}
		if sfi.IsDir() && isWithin(src, to) {
			return badRequest("不能把目录复制到自身内部: " + src)
		}
		// 目标已存在时自动改名为 "name 副本"、"name 副本 2" 等
		name, err := uniqueName(fsys, to, path.Base(src), sfi.IsDir())
		if err != nil {
			return err
		}
		dst := path.Join(to, name)
		if err := copyTree(fsys, src, dst); err != nil {
			return err
		}
		copied = append(copied, dst)
	}
	httpx.OK(w, map[string]any{"ok": true, "paths": copied})
	return nil
}

func handleChmod(w http.ResponseWriter, r *http.Request, fsys fileSystem) error {
	var req struct {
		Path      string `json:"path"`
		Mode      string `json:"mode"`
		Recursive bool   `json:"recursive"`
	}
	if err := httpx.ReadJSON(r, &req); err != nil {
		return badRequest(err.Error())
	}
	p, err := cleanPath(req.Path)
	if err != nil {
		return err
	}
	n, err := strconv.ParseUint(strings.TrimSpace(req.Mode), 8, 32)
	if err != nil || n > 0o7777 {
		return badRequest("权限必须是八进制数字，例如 755")
	}
	if err := chmodAll(fsys, p, fs.FileMode(n), req.Recursive); err != nil {
		return err
	}
	httpx.OK(w, map[string]any{"ok": true})
	return nil
}

func handleRemove(w http.ResponseWriter, r *http.Request, fsys fileSystem) error {
	var req struct {
		Paths []string `json:"paths"`
	}
	if err := httpx.ReadJSON(r, &req); err != nil {
		return badRequest(err.Error())
	}
	if len(req.Paths) == 0 {
		return badRequest("未选择要删除的项目")
	}
	paths := make([]string, 0, len(req.Paths))
	for _, raw := range req.Paths {
		p, err := cleanPath(raw)
		if err != nil {
			return err
		}
		if p == "/" {
			return &apiError{http.StatusForbidden, "禁止删除根目录"}
		}
		paths = append(paths, p)
	}
	for _, p := range paths {
		if err := removeAll(fsys, p); err != nil {
			return err
		}
	}
	httpx.OK(w, map[string]any{"ok": true, "count": len(paths)})
	return nil
}
