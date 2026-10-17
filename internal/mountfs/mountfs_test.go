package mountfs

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/spf13/afero"
)

func testMounts(t *testing.T) ([]Mount, string, string) {
	rw, ro := t.TempDir(), t.TempDir()
	if err := os.WriteFile(filepath.Join(ro, "readme.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	mounts := []Mount{{Name: "data", Path: rw}, {Name: "docs", Path: ro, ReadOnly: true}}
	if err := Validate(mounts); err != nil {
		t.Fatal(err)
	}
	return mounts, rw, ro
}

func TestValidate(t *testing.T) {
	cases := [][]Mount{
		{{Name: "", Path: "/tmp"}},
		{{Name: "a/b", Path: "/tmp"}},
		{{Name: "..", Path: "/tmp"}},
		{{Name: "a", Path: "rel"}},
		{{Name: "a", Path: "/tmp"}, {Name: "a", Path: "/var"}},
	}
	for i, c := range cases {
		if Validate(c) == nil {
			t.Errorf("第 %d 组应校验失败: %+v", i, c)
		}
	}
}

func TestAfero(t *testing.T) {
	mounts, rw, _ := testMounts(t)
	f := Afero(mounts)

	root, err := f.Open("/")
	if err != nil {
		t.Fatal(err)
	}
	names, err := root.Readdirnames(-1)
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(names)
	if len(names) != 2 || names[0] != "data" || names[1] != "docs" {
		t.Fatalf("根目录应列出挂载名称，实际 %v", names)
	}
	// 分页读取在末尾返回 io.EOF
	_, _ = root.Seek(0, io.SeekStart)
	if page, err := root.Readdir(1); err != nil || len(page) != 1 {
		t.Fatalf("分页读取失败: %v %v", page, err)
	}
	_, _ = root.Readdir(1)
	if _, err := root.Readdir(1); err != io.EOF {
		t.Fatalf("读完后应返回 io.EOF，实际 %v", err)
	}

	if err := afero.WriteFile(f, "/data/a.txt", []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(rw, "a.txt")); err != nil {
		t.Fatal("文件应写入挂载目录:", err)
	}
	if b, err := afero.ReadFile(f, "/docs/readme.txt"); err != nil || string(b) != "hello" {
		t.Fatalf("读取只读挂载失败: %q %v", b, err)
	}
	if err := afero.WriteFile(f, "/docs/b.txt", []byte("x"), 0o644); err == nil {
		t.Fatal("只读挂载应拒绝写入")
	}
	for _, op := range []func() error{
		func() error { return f.Mkdir("/new", 0o755) },
		func() error { return f.Remove("/data") },
		func() error { return f.Rename("/data/a.txt", "/docs/a.txt") },
		func() error { _, err := f.Create("/root.txt"); return err },
	} {
		if err := op(); !errors.Is(err, fs.ErrPermission) {
			t.Fatal("根目录、挂载目录本身与跨挂载移动应被拒绝")
		}
	}
	if err := f.Rename("/data/a.txt", "/data/c.txt"); err != nil {
		t.Fatal("同一挂载内移动应成功:", err)
	}
	if fi, err := f.Stat("/docs"); err != nil || fi.Name() != "docs" || !fi.IsDir() {
		t.Fatalf("挂载目录信息应使用挂载名称: %v %v", fi, err)
	}
	if _, err := f.Stat("/missing/x"); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("不存在的挂载应返回 ErrNotExist，实际 %v", err)
	}
	// 路径中的 .. 不能越出挂载目录
	if _, err := f.Stat("/data/../../etc/passwd"); err == nil {
		t.Fatal("越界路径不应可访问")
	}
}

func TestWebDAV(t *testing.T) {
	mounts, rw, _ := testMounts(t)
	d := WebDAV(mounts)
	ctx := context.Background()

	root, err := d.OpenFile(ctx, "/", os.O_RDONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	infos, err := root.Readdir(0)
	if err != nil || len(infos) != 2 {
		t.Fatalf("根目录应列出 2 个挂载: %v %v", infos, err)
	}
	if err := d.Mkdir(ctx, "/data/sub", 0o755); err != nil {
		t.Fatal(err)
	}
	if st, err := os.Stat(filepath.Join(rw, "sub")); err != nil || !st.IsDir() {
		t.Fatal("目录应创建在挂载目录中")
	}
	if err := d.Mkdir(ctx, "/docs/sub", 0o755); err == nil {
		t.Fatal("只读挂载应拒绝创建目录")
	}
	if err := d.RemoveAll(ctx, "/data"); err == nil {
		t.Fatal("不应允许删除挂载目录本身")
	}
	if _, err := d.OpenFile(ctx, "/x.txt", os.O_CREATE|os.O_WRONLY, 0o644); err == nil {
		t.Fatal("根目录应拒绝写入")
	}
	f, err := d.OpenFile(ctx, "/docs", os.O_RDONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	if fi, _ := f.Stat(); fi.Name() != "docs" {
		t.Fatalf("挂载目录句柄应返回挂载名称，实际 %s", fi.Name())
	}
}
