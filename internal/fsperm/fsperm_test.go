package fsperm

import (
	"os"
	"path/filepath"
	"testing"
)

func perm(t *testing.T, p string) os.FileMode {
	t.Helper()
	st, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	return st.Mode().Perm()
}

func TestWriteFileAndMkdirAllChmod(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "a", "b")
	// 预先以宽松权限创建，模拟文件系统忽略创建权限的情况
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	f := filepath.Join(dir, "x.json")
	if err := os.WriteFile(f, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := WriteFile(f, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := perm(t, dir); got != 0o700 {
		t.Fatalf("目录权限 %o", got)
	}
	if got := perm(t, f); got != 0o600 {
		t.Fatalf("文件权限 %o", got)
	}
}

func TestTighten(t *testing.T) {
	root := filepath.Join(t.TempDir(), "history")
	sub := filepath.Join(root, "frp")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	loose := filepath.Join(sub, "1.json")
	tight := filepath.Join(sub, "2.json")
	if err := os.WriteFile(loose, nil, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(tight, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	// 只有所有者权限但带执行位的文件也要改成 0600
	ownerExec := filepath.Join(sub, "3.json")
	if err := os.WriteFile(ownerExec, nil, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if n := Tighten(root); n != 4 {
		t.Fatalf("应改动 2 个目录和 2 个文件，实际 %d", n)
	}
	for p, want := range map[string]os.FileMode{root: 0o700, sub: 0o700, loose: 0o600, tight: 0o600, ownerExec: 0o600} {
		if got := perm(t, p); got != want {
			t.Fatalf("%s 权限 %o，期望 %o", p, got, want)
		}
	}
	if n := Tighten(filepath.Join(root, "missing")); n != 0 {
		t.Fatalf("不存在的路径不应改动，实际 %d", n)
	}
}
