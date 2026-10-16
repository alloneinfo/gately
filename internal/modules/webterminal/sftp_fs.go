package webterminal

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pkg/sftp"
)

// fileReader 是可随机读取的只读文件，os.File 与 sftp.File 均满足。
type fileReader interface {
	io.Reader
	io.Seeker
	io.ReaderAt
	io.Closer
}

// fileSystem 抽象文件管理所需的最小文件系统操作。
// 所有路径均为以 "/" 分隔的绝对路径，由实现负责转换为底层格式。
type fileSystem interface {
	// Home 返回默认起始目录
	Home() (string, error)
	Stat(p string) (fs.FileInfo, error)
	Lstat(p string) (fs.FileInfo, error)
	ReadDir(p string) ([]fs.FileInfo, error)
	Open(p string) (fileReader, error)
	// Create 创建或截断写入文件
	Create(p string) (io.WriteCloser, error)
	Mkdir(p string) error
	// Remove 删除文件或空目录
	Remove(p string) error
	Rename(oldPath, newPath string) error
	Chmod(p string, mode fs.FileMode) error
	Symlink(target, link string) error
	Readlink(p string) (string, error)
}

// localFS 操作服务器本机文件系统。
type localFS struct {
	workDir string
}

func (l *localFS) native(p string) string { return filepath.FromSlash(p) }

func (l *localFS) Home() (string, error) {
	if l.workDir != "" {
		return filepath.ToSlash(l.workDir), nil
	}
	h, err := os.UserHomeDir()
	if err != nil {
		return "/", nil
	}
	return filepath.ToSlash(h), nil
}
func (l *localFS) Stat(p string) (fs.FileInfo, error)  { return os.Stat(l.native(p)) }
func (l *localFS) Lstat(p string) (fs.FileInfo, error) { return os.Lstat(l.native(p)) }

func (l *localFS) ReadDir(p string) ([]fs.FileInfo, error) {
	entries, err := os.ReadDir(l.native(p))
	if err != nil {
		return nil, err
	}
	out := make([]fs.FileInfo, 0, len(entries))
	for _, e := range entries {
		fi, err := e.Info()
		if err != nil {
			// 列目录期间文件被删除等情况直接跳过
			continue
		}
		out = append(out, fi)
	}
	return out, nil
}

func (l *localFS) Open(p string) (fileReader, error) { return os.Open(l.native(p)) }

func (l *localFS) Create(p string) (io.WriteCloser, error) {
	return os.OpenFile(l.native(p), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
}
func (l *localFS) Mkdir(p string) error                   { return os.Mkdir(l.native(p), 0o755) }
func (l *localFS) Remove(p string) error                  { return os.Remove(l.native(p)) }
func (l *localFS) Rename(a, b string) error               { return os.Rename(l.native(a), l.native(b)) }
func (l *localFS) Chmod(p string, mode fs.FileMode) error { return os.Chmod(l.native(p), mode) }
func (l *localFS) Symlink(target, link string) error      { return os.Symlink(target, l.native(link)) }
func (l *localFS) Readlink(p string) (string, error)      { return os.Readlink(l.native(p)) }

// sftpFS 通过 SFTP 操作远端文件系统。
type sftpFS struct {
	c *sftp.Client
}

func (s *sftpFS) Home() (string, error)               { return s.c.Getwd() }
func (s *sftpFS) Stat(p string) (fs.FileInfo, error)  { return s.c.Stat(p) }
func (s *sftpFS) Lstat(p string) (fs.FileInfo, error) { return s.c.Lstat(p) }
func (s *sftpFS) ReadDir(p string) ([]fs.FileInfo, error) {
	return s.c.ReadDir(p)
}
func (s *sftpFS) Open(p string) (fileReader, error) { return s.c.Open(p) }
func (s *sftpFS) Create(p string) (io.WriteCloser, error) {
	return s.c.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_TRUNC)
}
func (s *sftpFS) Mkdir(p string) error                   { return s.c.Mkdir(p) }
func (s *sftpFS) Remove(p string) error                  { return s.c.Remove(p) }
func (s *sftpFS) Rename(a, b string) error               { return s.c.Rename(a, b) }
func (s *sftpFS) Chmod(p string, mode fs.FileMode) error { return s.c.Chmod(p, mode) }
func (s *sftpFS) Symlink(target, link string) error      { return s.c.Symlink(target, link) }
func (s *sftpFS) Readlink(p string) (string, error)      { return s.c.ReadLink(p) }

// ---- 基于 fileSystem 的通用辅助函数 ----

// exists 判断路径是否存在（不跟随符号链接，悬空链接也算存在）。
func exists(fsys fileSystem, p string) (bool, error) {
	_, err := fsys.Lstat(p)
	if err == nil {
		return true, nil
	}
	if isNotExist(err) {
		return false, nil
	}
	return false, err
}

// mkdirAll 递归创建目录，已存在的目录视为成功，路径上存在普通文件则报错。
func mkdirAll(fsys fileSystem, p string) error {
	fi, err := fsys.Stat(p)
	if err == nil {
		if fi.IsDir() {
			return nil
		}
		return &apiError{409, "路径已存在且不是目录: " + p}
	}
	if !isNotExist(err) {
		return err
	}
	if p == "/" || p == "." {
		return err
	}
	if err := mkdirAll(fsys, path.Dir(p)); err != nil {
		return err
	}
	if err := fsys.Mkdir(p); err != nil {
		// 并发创建时再确认一次
		if fi, e := fsys.Stat(p); e == nil && fi.IsDir() {
			return nil
		}
		return err
	}
	return nil
}

// removeAll 递归删除，符号链接只删除链接本身。
func removeAll(fsys fileSystem, p string) error {
	fi, err := fsys.Lstat(p)
	if err != nil {
		return err
	}
	if fi.IsDir() {
		children, err := fsys.ReadDir(p)
		if err != nil {
			return err
		}
		for _, c := range children {
			if err := removeAll(fsys, path.Join(p, c.Name())); err != nil {
				return err
			}
		}
	}
	return fsys.Remove(p)
}

// chmodAll 递归修改权限，不跟随符号链接。
func chmodAll(fsys fileSystem, p string, mode fs.FileMode, recursive bool) error {
	fi, err := fsys.Lstat(p)
	if err != nil {
		return err
	}
	if fi.Mode()&fs.ModeSymlink != 0 {
		if recursive {
			return nil
		}
		return fsys.Chmod(p, mode)
	}
	if err := fsys.Chmod(p, mode); err != nil {
		return err
	}
	if recursive && fi.IsDir() {
		children, err := fsys.ReadDir(p)
		if err != nil {
			return err
		}
		for _, c := range children {
			if err := chmodAll(fsys, path.Join(p, c.Name()), mode, true); err != nil {
				return err
			}
		}
	}
	return nil
}

// copyTree 递归复制 src 到 dst（dst 必须不存在），符号链接按链接复制。
func copyTree(fsys fileSystem, src, dst string) error {
	fi, err := fsys.Lstat(src)
	if err != nil {
		return err
	}
	switch {
	case fi.Mode()&fs.ModeSymlink != 0:
		target, err := fsys.Readlink(src)
		if err != nil {
			return err
		}
		return fsys.Symlink(target, dst)
	case fi.IsDir():
		if err := fsys.Mkdir(dst); err != nil {
			return err
		}
		children, err := fsys.ReadDir(src)
		if err != nil {
			return err
		}
		for _, c := range children {
			if err := copyTree(fsys, path.Join(src, c.Name()), path.Join(dst, c.Name())); err != nil {
				return err
			}
		}
		// 子项写完后再同步目录权限，避免只读目录影响写入
		_ = fsys.Chmod(dst, fi.Mode().Perm())
		return nil
	case fi.Mode().IsRegular():
		return copyFile(fsys, src, dst, fi.Mode().Perm())
	}
	// 设备、管道、套接字等特殊文件不复制
	return nil
}

func copyFile(fsys fileSystem, src, dst string, perm fs.FileMode) error {
	in, err := fsys.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := fsys.Create(dst)
	if err != nil {
		return err
	}
	_, err = io.Copy(out, in)
	if cerr := out.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		_ = fsys.Remove(dst)
		return err
	}
	_ = fsys.Chmod(dst, perm)
	return nil
}

// uniqueName 在 dir 下为 name 选一个不冲突的名称：name、"name 副本"、"name 副本 2"……
func uniqueName(fsys fileSystem, dir, name string, isDir bool) (string, error) {
	ext := ""
	base := name
	if !isDir {
		ext = path.Ext(name)
		if ext != name {
			base = strings.TrimSuffix(name, ext)
		} else {
			ext = ""
		}
	}
	candidate := name
	for i := 1; ; i++ {
		ok, err := exists(fsys, path.Join(dir, candidate))
		if err != nil {
			return "", err
		}
		if !ok {
			return candidate, nil
		}
		if i == 1 {
			candidate = base + " 副本" + ext
		} else {
			candidate = base + " 副本 " + itoa(i) + ext
		}
		if i > 10000 {
			return "", errors.New("无法生成不冲突的名称")
		}
	}
}

// isWithin 判断 p 是否等于 root 或位于 root 之下。
func isWithin(root, p string) bool {
	if root == "/" {
		return true
	}
	return p == root || strings.HasPrefix(p, root+"/")
}

// sortEntries 目录在前，同类按名称（忽略大小写）排序。
func sortEntries(list []fileEntry) {
	sort.SliceStable(list, func(i, j int) bool {
		if list[i].IsDir != list[j].IsDir {
			return list[i].IsDir
		}
		a, b := strings.ToLower(list[i].Name), strings.ToLower(list[j].Name)
		if a != b {
			return a < b
		}
		return list[i].Name < list[j].Name
	})
}
