package webterminal

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPushFiles(t *testing.T) {
	env := newSSHEnv(t)
	m := env.m
	ctx := context.Background()
	dir := filepath.Join(env.root, "certs", "nas")
	files := []PushFile{{Name: "a.crt", Data: []byte("cert\n"), Mode: 0o644}, {Name: "a.key", Data: []byte("key\n"), Mode: 0o600}}
	if _, err := m.Push(ctx, "s", dir, files, ""); err == nil || !strings.Contains(err.Error(), "主机密钥") {
		t.Fatalf("未记录主机密钥时应拒绝推送: %v", err)
	}
	// 模拟在 Web 终端中连接过一次：首次连接记录主机密钥
	_, release, err := m.acquireSSH(ctx, "s")
	if err != nil {
		t.Fatal(err)
	}
	release()
	if ts := m.SSHTargets(); len(ts) != 1 || !ts[0].Trusted || ts[0].Name != "nas" {
		t.Fatalf("SSH 目标列表错误: %+v", ts)
	}
	if _, err := m.Push(ctx, "s", dir, files, ""); err != nil {
		t.Fatal(err)
	}
	// 再推一次覆盖已有文件
	files[0].Data = []byte("cert2\n")
	if _, err := m.Push(ctx, "s", dir, files, ""); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "a.crt")); string(b) != "cert2\n" {
		t.Fatalf("证书内容错误: %q", b)
	}
	fi, err := os.Stat(filepath.Join(dir, "a.key"))
	if err != nil || fi.Mode().Perm() != 0o600 {
		t.Fatalf("私钥权限应为 0600: %v %v", err, fi)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 2 {
		t.Fatalf("不应残留临时文件: %v", entries)
	}
	if _, err := m.Push(ctx, "missing", dir, files, ""); err == nil {
		t.Fatal("不存在的连接应报错")
	}
}
