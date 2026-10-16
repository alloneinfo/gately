package storage

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// localBackend 把本机目录作为存储。使用 os.Root 限定访问范围，符号链接也无法越出根目录。
type localBackend struct {
	root *os.Root
}

func newLocal(dir string) (*localBackend, error) {
	if !filepath.IsAbs(dir) {
		return nil, errors.New("目录需为绝对路径")
	}
	r, err := os.OpenRoot(dir)
	if err != nil {
		return nil, err
	}
	return &localBackend{root: r}, nil
}

// rel 把挂载内路径转换为 os.Root 使用的相对路径。
func rel(p string) string {
	p = strings.TrimPrefix(cleanPath(p), "/")
	if p == "" {
		return "."
	}
	return p
}

func mapErr(err error) error {
	if errors.Is(err, fs.ErrNotExist) {
		return errNotFound
	}
	return err
}

func (l *localBackend) List(p string) ([]Entry, error) {
	d, err := l.root.Open(rel(p))
	if err != nil {
		return nil, mapErr(err)
	}
	defer d.Close()
	infos, err := d.ReadDir(-1)
	if err != nil {
		return nil, err
	}
	out := make([]Entry, 0, len(infos))
	for _, de := range infos {
		info, err := de.Info()
		if err != nil {
			continue
		}
		isDir := de.IsDir()
		// 指向目录的符号链接按目录展示
		if de.Type()&fs.ModeSymlink != 0 {
			if st, err := l.root.Stat(rel(p + "/" + de.Name())); err == nil {
				info, isDir = st, st.IsDir()
			}
		}
		out = append(out, Entry{Name: de.Name(), IsDir: isDir, Size: info.Size(), ModTime: info.ModTime()})
	}
	sortEntries(out)
	return out, nil
}

func sortEntries(es []Entry) {
	sort.Slice(es, func(i, j int) bool {
		if es[i].IsDir != es[j].IsDir {
			return es[i].IsDir
		}
		return strings.ToLower(es[i].Name) < strings.ToLower(es[j].Name)
	})
}

func (l *localBackend) Open(p string) (io.ReadCloser, int64, time.Time, error) {
	f, err := l.root.Open(rel(p))
	if err != nil {
		return nil, 0, time.Time{}, mapErr(err)
	}
	st, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, 0, time.Time{}, err
	}
	if st.IsDir() {
		f.Close()
		return nil, 0, time.Time{}, errors.New("目录无法下载")
	}
	return f, st.Size(), st.ModTime(), nil
}

func (l *localBackend) Put(p string, r io.Reader, _ int64) error {
	f, err := l.root.OpenFile(rel(p), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return mapErr(err)
	}
	if _, err := io.Copy(f, r); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

func (l *localBackend) Mkdir(p string) error { return mapErr(l.root.Mkdir(rel(p), 0o755)) }

func (l *localBackend) Remove(p string) error {
	if rel(p) == "." {
		return errors.New("不能删除挂载根目录")
	}
	return mapErr(l.root.RemoveAll(rel(p)))
}

func (l *localBackend) Rename(from, to string) error {
	if rel(from) == "." || rel(to) == "." {
		return errors.New("不能重命名挂载根目录")
	}
	return mapErr(l.root.Rename(rel(from), rel(to)))
}

func (l *localBackend) Close() error { return l.root.Close() }
