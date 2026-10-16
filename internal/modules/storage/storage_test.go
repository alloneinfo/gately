package storage

import (
	"io"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	xwebdav "golang.org/x/net/webdav"
)

// exercise 对任意后端执行完整的增删改查流程。
func exercise(t *testing.T, b backend) {
	t.Helper()
	if err := b.Mkdir("/docs"); err != nil {
		t.Fatal("mkdir:", err)
	}
	body := "hello"
	if err := b.Put("/docs/a.txt", strings.NewReader(body), int64(len(body))); err != nil {
		t.Fatal("put:", err)
	}
	es, err := b.List("/")
	if err != nil || len(es) != 1 || es[0].Name != "docs" || !es[0].IsDir {
		t.Fatalf("根目录列表不正确: %v %+v", err, es)
	}
	es, err = b.List("/docs")
	if err != nil || len(es) != 1 || es[0].Name != "a.txt" || es[0].IsDir || es[0].Size != 5 {
		t.Fatalf("子目录列表不正确: %v %+v", err, es)
	}
	rc, size, _, err := b.Open("/docs/a.txt")
	if err != nil {
		t.Fatal("open:", err)
	}
	got, _ := io.ReadAll(rc)
	rc.Close()
	if string(got) != body || size != 5 {
		t.Fatalf("内容不一致: %q %d", got, size)
	}
	if err := b.Rename("/docs/a.txt", "/docs/b.txt"); err != nil {
		t.Fatal("rename:", err)
	}
	if _, _, _, err := b.Open("/docs/a.txt"); err != errNotFound {
		t.Fatalf("旧文件应不存在: %v", err)
	}
	if err := b.Remove("/docs"); err != nil {
		t.Fatal("remove:", err)
	}
	if es, _ := b.List("/"); len(es) != 0 {
		t.Fatalf("删除后应为空: %+v", es)
	}
	if err := b.Remove("/"); err == nil {
		t.Fatal("删除根目录应被拒绝")
	}
}

func TestLocalBackend(t *testing.T) {
	root := t.TempDir()
	b, err := newLocal(root)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	exercise(t, b)

	// 路径穿越与符号链接都不能越出根目录
	outside := t.TempDir()
	_ = os.WriteFile(filepath.Join(outside, "secret"), []byte("x"), 0o644)
	if _, _, _, err := b.Open("/../../" + filepath.Base(outside) + "/secret"); err == nil {
		t.Fatal("路径穿越应失败")
	}
	_ = os.Symlink(outside, filepath.Join(root, "link"))
	if _, _, _, err := b.Open("/link/secret"); err == nil {
		t.Fatal("符号链接越界应失败")
	}
}

func TestDAVBackend(t *testing.T) {
	dir := t.TempDir()
	srv := httptest.NewServer(&xwebdav.Handler{FileSystem: xwebdav.Dir(dir), LockSystem: xwebdav.NewMemLS()})
	defer srv.Close()
	b, err := newDAV(srv.URL+"/dav", "", "", false)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	// 根路径 /dav 需先存在
	_ = os.MkdirAll(filepath.Join(dir, "dav"), 0o755)
	exercise(t, b)
}

func TestMountValidate(t *testing.T) {
	m := Mount{Name: "x", Type: TypeLocal, Path: "relative"}
	if err := m.Validate(); err == nil {
		t.Fatal("相对路径应报错")
	}
	m = Mount{Name: "x", Type: TypeLocal, Path: t.TempDir()}
	if err := m.Validate(); err != nil {
		t.Fatal(err)
	}
	m = Mount{Name: "x", Type: TypeWebDAV, URL: "ftp://h"}
	if err := m.Validate(); err == nil {
		t.Fatal("非 http 地址应报错")
	}
}
