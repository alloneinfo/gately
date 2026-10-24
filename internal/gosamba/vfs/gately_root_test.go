// Gately 新增：根目录为 "/" 的共享与同名前缀目录的越界判断测试。

package vfs

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// TestResolve_SlashRoot 验证共享根目录为 "/" 时子路径都能正常解析。
// 旧实现拼出的前缀是 "//"，导致所有子路径都被当成越界。
func TestResolve_SlashRoot(t *testing.T) {
	cases := []struct {
		smb, want string
	}{
		{"", "/"},
		{"/", "/"},
		{"etc", "/etc"},
		{"etc\\hosts", "/etc/hosts"},
		{"/usr/bin", "/usr/bin"},
		// 根目录之上没有更高层级，".." 被 Clean 收敛回 "/" 下，仍在共享内。
		{"../etc", "/etc"},
	}
	for _, tc := range cases {
		got, err := Resolve("/", tc.smb)
		if err != nil {
			t.Errorf("Resolve(\"/\", %q) error: %v", tc.smb, err)
			continue
		}
		if got != tc.want {
			t.Errorf("Resolve(\"/\", %q) = %q, want %q", tc.smb, got, tc.want)
		}
	}
}

// TestResolveSecure_SlashRoot 验证带符号链接检查的解析在根目录为 "/" 时也可用，
// 包括 macOS 上 /etc 这类指向 /private/etc 的符号链接。
func TestResolveSecure_SlashRoot(t *testing.T) {
	dir := t.TempDir()
	name := filepath.Join(dir, "f.txt")
	if err := os.WriteFile(name, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{"etc", name, filepath.Join(dir, "not-exist")} {
		got, err := ResolveSecure("/", p)
		if err != nil {
			t.Errorf("ResolveSecure(\"/\", %q) error: %v", p, err)
			continue
		}
		if got != filepath.Clean("/"+p) {
			t.Errorf("ResolveSecure(\"/\", %q) = %q", p, got)
		}
	}
	got, err := ResolveSecureNorm("/", "etc", false)
	if err != nil || got != "/etc" {
		t.Errorf("ResolveSecureNorm(\"/\", \"etc\") = %q, %v", got, err)
	}
	if _, err := ResolveLink("/", name); err != nil {
		t.Errorf("ResolveLink(\"/\", %q) error: %v", name, err)
	}
}

// TestResolve_PrefixSibling 验证普通根目录的越界保护没有被放宽：
// "/srv" 不能访问同名前缀的 "/srv2"，".." 也不能跳出。
func TestResolve_PrefixSibling(t *testing.T) {
	for _, p := range []string{"../srv2", "../srv2/x", "..", "../etc/passwd", "a/../../srv2"} {
		if _, err := Resolve("/srv", p); !errors.Is(err, ErrTraversal) {
			t.Errorf("Resolve(\"/srv\", %q) err=%v, want ErrTraversal", p, err)
		}
	}
	if got, err := Resolve("/srv", "x/../y"); err != nil || got != "/srv/y" {
		t.Errorf("Resolve(\"/srv\", \"x/../y\") = %q, %v", got, err)
	}
	if !underRoot("/srv", "/srv") || underRoot("/srv", "/srv2") || !underRoot("/", "/srv2") {
		t.Error("underRoot 前缀判断错误")
	}
}
