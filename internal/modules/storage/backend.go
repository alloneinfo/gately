package storage

import (
	"errors"
	"io"
	"path"
	"strings"
	"time"
)

// Entry 是目录列表中的一项。
type Entry struct {
	Name    string    `json:"name"`
	IsDir   bool      `json:"isDir"`
	Size    int64     `json:"size"`
	ModTime time.Time `json:"modTime"`
}

// backend 抽象一种存储（本地目录或远程 WebDAV）。所有 path 参数均为以 / 开头的挂载内相对路径。
type backend interface {
	List(p string) ([]Entry, error)
	// Open 打开文件用于下载，返回内容、大小与修改时间
	Open(p string) (rc io.ReadCloser, size int64, mod time.Time, err error)
	// Put 写入文件（覆盖已有文件），size 为内容长度，未知时为 -1
	Put(p string, r io.Reader, size int64) error
	Mkdir(p string) error
	Remove(p string) error
	Rename(from, to string) error
	Close() error
}

// errNotFound 表示路径不存在。
var errNotFound = errors.New("路径不存在")

// cleanPath 规范化挂载内路径：统一以 / 开头，消除 .. 与重复分隔符，使其无法越出挂载根。
func cleanPath(p string) string {
	p = strings.ReplaceAll(p, "\\", "/")
	return path.Clean("/" + p)
}
