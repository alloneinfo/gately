package webterminal

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"errors"
	"io"
	"io/fs"
	"net/http"
	"path"
	"strings"

	"github.com/wangwanjie/gately/internal/httpx"
)

// previewArchiveMax 是压缩包预览最多返回的条目数。
const previewArchiveMax = 10000

// 压缩格式
const (
	fmtZip   = "zip"
	fmtTar   = "tar"
	fmtTarGz = "tar.gz"
)

// archiveFormat 按文件名后缀识别可解压的格式，无法识别返回空串。
func archiveFormat(name string) string {
	n := strings.ToLower(name)
	switch {
	case strings.HasSuffix(n, ".zip"):
		return fmtZip
	case strings.HasSuffix(n, ".tar.gz"), strings.HasSuffix(n, ".tgz"):
		return fmtTarGz
	case strings.HasSuffix(n, ".tar"):
		return fmtTar
	}
	return ""
}

// archiveSuffix 返回压缩格式对应的后缀。
func archiveSuffix(format string) string {
	if format == fmtZip {
		return ".zip"
	}
	return ".tar.gz"
}

// skipSpecial 判断是否为设备、管道、套接字等不适合打包的特殊文件。
func skipSpecial(m fs.FileMode) bool {
	return m&(fs.ModeDevice|fs.ModeCharDevice|fs.ModeNamedPipe|fs.ModeSocket|fs.ModeIrregular) != 0
}

// archiveWriter 屏蔽 zip 与 tar 的写入差异。
type archiveWriter interface {
	// addDir 添加目录条目，name 不含结尾斜杠
	addDir(name string, fi fs.FileInfo) error
	addLink(name, target string, fi fs.FileInfo) error
	addFile(name string, fi fs.FileInfo, r io.Reader) error
	Close() error
}

type zipArchive struct{ w *zip.Writer }

func (z *zipArchive) addDir(name string, fi fs.FileInfo) error {
	h, err := zip.FileInfoHeader(fi)
	if err != nil {
		return err
	}
	h.Name = name + "/"
	h.Method = zip.Store
	_, err = z.w.CreateHeader(h)
	return err
}

func (z *zipArchive) addLink(name, target string, fi fs.FileInfo) error {
	h, err := zip.FileInfoHeader(fi)
	if err != nil {
		return err
	}
	h.Name = name
	h.Method = zip.Store
	w, err := z.w.CreateHeader(h)
	if err != nil {
		return err
	}
	_, err = io.WriteString(w, target)
	return err
}

func (z *zipArchive) addFile(name string, fi fs.FileInfo, r io.Reader) error {
	h, err := zip.FileInfoHeader(fi)
	if err != nil {
		return err
	}
	h.Name = name
	h.Method = zip.Deflate
	w, err := z.w.CreateHeader(h)
	if err != nil {
		return err
	}
	_, err = io.Copy(w, r)
	return err
}

func (z *zipArchive) Close() error { return z.w.Close() }

type tarArchive struct {
	tw *tar.Writer
	gz *gzip.Writer
}

func (t *tarArchive) header(name, link string, fi fs.FileInfo) (*tar.Header, error) {
	h, err := tar.FileInfoHeader(fi, link)
	if err != nil {
		return nil, err
	}
	h.Name = name
	// 不写入属主信息，避免解压时依赖来源机器的用户体系
	h.Uid, h.Gid, h.Uname, h.Gname = 0, 0, "", ""
	return h, nil
}

func (t *tarArchive) addDir(name string, fi fs.FileInfo) error {
	h, err := t.header(name+"/", "", fi)
	if err != nil {
		return err
	}
	return t.tw.WriteHeader(h)
}

func (t *tarArchive) addLink(name, target string, fi fs.FileInfo) error {
	h, err := t.header(name, target, fi)
	if err != nil {
		return err
	}
	return t.tw.WriteHeader(h)
}

func (t *tarArchive) addFile(name string, fi fs.FileInfo, r io.Reader) error {
	h, err := t.header(name, "", fi)
	if err != nil {
		return err
	}
	if err := t.tw.WriteHeader(h); err != nil {
		return err
	}
	// 以头部声明的大小为准，文件在读取期间变大时多余部分被截断
	_, err = io.Copy(t.tw, io.LimitReader(r, h.Size))
	return err
}

func (t *tarArchive) Close() error {
	if err := t.tw.Close(); err != nil {
		return err
	}
	return t.gz.Close()
}

// addTree 把 src 递归加入归档，归档内路径以 name 为根；skip 为归档文件自身，避免自我打包。
func addTree(fsys fileSystem, aw archiveWriter, src, name, skip string) error {
	if src == skip {
		return nil
	}
	fi, err := fsys.Lstat(src)
	if err != nil {
		return err
	}
	switch {
	case skipSpecial(fi.Mode()):
		return nil
	case fi.Mode()&fs.ModeSymlink != 0:
		target, err := fsys.Readlink(src)
		if err != nil {
			return err
		}
		return aw.addLink(name, target, fi)
	case fi.IsDir():
		if err := aw.addDir(name, fi); err != nil {
			return err
		}
		children, err := fsys.ReadDir(src)
		if err != nil {
			return err
		}
		for _, c := range children {
			if err := addTree(fsys, aw, path.Join(src, c.Name()), name+"/"+c.Name(), skip); err != nil {
				return err
			}
		}
		return nil
	}
	f, err := fsys.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()
	return aw.addFile(name, fi, f)
}

func handleCompress(w http.ResponseWriter, r *http.Request, fsys fileSystem) error {
	var req struct {
		Paths  []string `json:"paths"`
		Dest   string   `json:"dest"`
		Format string   `json:"format"`
	}
	if err := httpx.ReadJSON(r, &req); err != nil {
		return badRequest(err.Error())
	}
	if len(req.Paths) == 0 {
		return badRequest("未选择要压缩的项目")
	}
	format := strings.ToLower(strings.TrimSpace(req.Format))
	switch format {
	case "":
		format = fmtZip
	case fmtZip, fmtTarGz:
	case "tgz":
		format = fmtTarGz
	default:
		return badRequest("不支持的压缩格式，仅支持 zip 与 tar.gz")
	}
	srcs := make([]string, 0, len(req.Paths))
	seen := map[string]bool{}
	for _, raw := range req.Paths {
		p, err := cleanPath(raw)
		if err != nil {
			return err
		}
		if p == "/" {
			return badRequest("不能压缩根目录")
		}
		if seen[path.Base(p)] {
			return badRequest("所选项目存在同名条目: " + path.Base(p))
		}
		seen[path.Base(p)] = true
		srcs = append(srcs, p)
	}
	dest, err := cleanPath(req.Dest)
	if err != nil {
		return err
	}
	// dest 是已存在的目录，或以 / 结尾时，在其中按首个项目名自动命名
	if fi, err := fsys.Stat(dest); err == nil && fi.IsDir() || strings.HasSuffix(req.Dest, "/") {
		if err := mkdirAll(fsys, dest); err != nil {
			return err
		}
		base := path.Base(srcs[0])
		if len(srcs) > 1 {
			base = path.Base(path.Dir(srcs[0]))
			if base == "/" || base == "." {
				base = "archive"
			}
		}
		dest = path.Join(dest, base+archiveSuffix(format))
	} else if err := mkdirAll(fsys, path.Dir(dest)); err != nil {
		return err
	}
	if ok, err := exists(fsys, dest); err != nil {
		return err
	} else if ok {
		return &apiError{http.StatusConflict, "目标压缩包已存在: " + dest}
	}

	out, err := fsys.Create(dest)
	if err != nil {
		return err
	}
	var aw archiveWriter
	if format == fmtZip {
		aw = &zipArchive{w: zip.NewWriter(out)}
	} else {
		gz := gzip.NewWriter(out)
		aw = &tarArchive{tw: tar.NewWriter(gz), gz: gz}
	}
	for _, src := range srcs {
		if err = addTree(fsys, aw, src, path.Base(src), dest); err != nil {
			break
		}
	}
	if cerr := aw.Close(); err == nil {
		err = cerr
	}
	if cerr := out.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		_ = fsys.Remove(dest)
		return err
	}
	httpx.OK(w, map[string]any{"ok": true, "path": dest})
	return nil
}

// archiveEntry 是归档中的一个条目，内容通过 open 按需读取。
type archiveEntry struct {
	name   string
	size   int64
	isDir  bool
	isFile bool // 普通文件；符号链接、硬链接等其余类型为 false
	mode   fs.FileMode
	open   func() (io.Reader, error)
}

// walkArchive 依次遍历归档条目，fn 返回 errStopWalk 可提前结束。
func walkArchive(fsys fileSystem, archivePath string, fn func(e *archiveEntry) error) error {
	format := archiveFormat(archivePath)
	if format == "" {
		return &apiError{http.StatusUnsupportedMediaType, "不支持的压缩包格式，仅支持 zip、tar、tar.gz、tgz"}
	}
	fi, err := fsys.Stat(archivePath)
	if err != nil {
		return err
	}
	if fi.IsDir() {
		return badRequest("路径是目录")
	}
	f, err := fsys.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()

	if format == fmtZip {
		zr, err := zip.NewReader(f, fi.Size())
		if err != nil {
			return badRequest("无法解析 zip 文件: " + err.Error())
		}
		for _, zf := range zr.File {
			zf := zf
			mode := zf.Mode()
			e := &archiveEntry{
				name:   zf.Name,
				size:   int64(zf.UncompressedSize64),
				isDir:  zf.FileInfo().IsDir(),
				isFile: mode.IsRegular(),
				mode:   mode,
				open: func() (io.Reader, error) {
					rc, err := zf.Open()
					return rc, err
				},
			}
			if err := fn(e); err != nil {
				if errors.Is(err, errStopWalk) {
					return nil
				}
				return err
			}
		}
		return nil
	}

	var src io.Reader = f
	if format == fmtTarGz {
		gz, err := gzip.NewReader(f)
		if err != nil {
			return badRequest("无法解析 gzip 文件: " + err.Error())
		}
		defer gz.Close()
		src = gz
	}
	tr := tar.NewReader(src)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return badRequest("无法解析 tar 文件: " + err.Error())
		}
		e := &archiveEntry{
			name:   h.Name,
			size:   h.Size,
			isDir:  h.Typeflag == tar.TypeDir,
			isFile: h.Typeflag == tar.TypeReg,
			mode:   fs.FileMode(h.Mode),
			open:   func() (io.Reader, error) { return tr, nil },
		}
		if err := fn(e); err != nil {
			if errors.Is(err, errStopWalk) {
				return nil
			}
			return err
		}
	}
}

var errStopWalk = errors.New("stop walk")

// safeEntryPath 把归档条目名映射到 dest 之内，含 ".." 的条目整体拒绝（防 zip-slip）。
// 返回空串表示条目指向目录自身，应忽略。
func safeEntryPath(dest, name string) (string, error) {
	name = strings.ReplaceAll(name, "\\", "/")
	for _, seg := range strings.Split(name, "/") {
		if seg == ".." {
			return "", badRequest("压缩包包含非法路径: " + name)
		}
	}
	c := path.Clean("/" + name)
	if c == "/" {
		return "", nil
	}
	target := path.Join(dest, c)
	if !isWithin(dest, target) {
		return "", badRequest("压缩包包含非法路径: " + name)
	}
	return target, nil
}

func handleDecompress(w http.ResponseWriter, r *http.Request, fsys fileSystem) error {
	var req struct {
		Path string `json:"path"`
		Dest string `json:"dest"`
	}
	if err := httpx.ReadJSON(r, &req); err != nil {
		return badRequest(err.Error())
	}
	src, err := cleanPath(req.Path)
	if err != nil {
		return err
	}
	// dest 为空时解压到压缩包所在目录
	dest := path.Dir(src)
	if strings.TrimSpace(req.Dest) != "" {
		if dest, err = cleanPath(req.Dest); err != nil {
			return err
		}
	}
	if err := mkdirAll(fsys, dest); err != nil {
		return err
	}
	count := 0
	err = walkArchive(fsys, src, func(e *archiveEntry) error {
		target, err := safeEntryPath(dest, e.name)
		if err != nil {
			return err
		}
		if target == "" {
			return nil
		}
		if e.isDir {
			return mkdirAll(fsys, target)
		}
		// 符号链接、硬链接等条目会让后续写入逃出目标目录，一律跳过
		if !e.isFile {
			return nil
		}
		if err := mkdirAll(fsys, path.Dir(target)); err != nil {
			return err
		}
		if fi, err := fsys.Lstat(target); err == nil {
			if fi.IsDir() {
				return &apiError{http.StatusConflict, "已存在同名目录: " + target}
			}
			// 目标是符号链接时先删除，避免写穿到链接指向的位置
			if fi.Mode()&fs.ModeSymlink != 0 {
				if err := fsys.Remove(target); err != nil {
					return err
				}
			}
		}
		rd, err := e.open()
		if err != nil {
			return err
		}
		if c, ok := rd.(io.Closer); ok {
			defer c.Close()
		}
		out, err := fsys.Create(target)
		if err != nil {
			return err
		}
		_, err = io.Copy(out, rd)
		if cerr := out.Close(); err == nil {
			err = cerr
		}
		if err != nil {
			_ = fsys.Remove(target)
			return err
		}
		// 仅保留 rwx 权限位，丢弃 setuid 等特殊位
		if perm := e.mode.Perm(); perm != 0 {
			_ = fsys.Chmod(target, perm)
		}
		count++
		return nil
	})
	if err != nil {
		return err
	}
	httpx.OK(w, map[string]any{"ok": true, "dest": dest, "count": count})
	return nil
}

func handlePreviewArchive(w http.ResponseWriter, r *http.Request, fsys fileSystem) error {
	p, err := cleanPath(r.URL.Query().Get("path"))
	if err != nil {
		return err
	}
	type item struct {
		Name  string `json:"name"`
		Size  int64  `json:"size"`
		IsDir bool   `json:"isDir"`
	}
	list := []item{}
	err = walkArchive(fsys, p, func(e *archiveEntry) error {
		if len(list) >= previewArchiveMax {
			return errStopWalk
		}
		list = append(list, item{Name: strings.TrimSuffix(e.name, "/"), Size: e.size, IsDir: e.isDir})
		return nil
	})
	if err != nil {
		return err
	}
	httpx.OK(w, list)
	return nil
}
