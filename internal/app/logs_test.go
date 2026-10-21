package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTightenConfigDir(t *testing.T) {
	dir := t.TempDir()
	write := func(rel string, mode os.FileMode) string {
		p := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("x"), mode); err != nil {
			t.Fatal(err)
		}
		return p
	}
	// 模拟旧版本在忽略创建权限的文件系统上留下的文件
	hist := write("history/frp/20261019T164942.550610455Z.json", 0o755)
	frpc := write("frp/frpc.toml", 0o755)
	audit := write("audit.log.1", 0o644)
	log := write("logs/ddns.log", 0o755)
	// bin/ 里是用户放的程序，必须保持可执行
	bin := write("bin/frpc", 0o755)
	if n := tightenConfigDir(dir); n == 0 {
		t.Fatal("应有改动")
	}
	for _, p := range []string{hist, frpc, audit, log} {
		st, _ := os.Stat(p)
		if st.Mode().Perm() != 0o600 {
			t.Fatalf("%s 权限 %o", p, st.Mode().Perm())
		}
	}
	for _, d := range []string{"history", "history/frp", "frp", "logs"} {
		st, _ := os.Stat(filepath.Join(dir, d))
		if st.Mode().Perm() != 0o700 {
			t.Fatalf("%s 权限 %o", d, st.Mode().Perm())
		}
	}
	if st, _ := os.Stat(bin); st.Mode().Perm() != 0o755 {
		t.Fatalf("bin/ 不应被改动，实际 %o", st.Mode().Perm())
	}
	if n := tightenConfigDir(dir); n != 0 {
		t.Fatalf("再次执行不应改动，实际 %d", n)
	}
}
