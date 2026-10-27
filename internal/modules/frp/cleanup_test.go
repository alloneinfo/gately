package frp

import (
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

// 启动时删除不属于当前实例的配置文件（旧版 frpc.toml / frps.toml、恢复历史后遗留的实例文件），
// 停用的实例不保留含令牌的配置文件；无关文件不受影响。
func TestStartCleansStaleConfigFiles(t *testing.T) {
	dir := t.TempDir()
	m, _, _ := newTestModule(t, dir)
	m.cfg.Clients = []Client{{Key: "c1", Name: "家里", ClientSettings: ClientSettings{ServerAddr: "a", ServerPort: 7000}}}
	m.cfg.Servers = []Server{{Key: "s1", Name: "frps", ServerSettings: ServerSettings{BindPort: 7000}}}
	frpDir := filepath.Join(dir, "frp")
	if err := os.MkdirAll(frpDir, 0o700); err != nil {
		t.Fatal(err)
	}
	stale := []string{"frpc.toml", "frps.toml", "frpc-old.toml", "frps-gone.toml.tmp", "frpc-c1.toml"}
	for _, name := range append(stale, "notes.txt") {
		if err := os.WriteFile(filepath.Join(frpDir, name), []byte("token = \"x\""), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := m.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = m.Stop() })
	for _, name := range stale {
		if _, err := os.Stat(filepath.Join(frpDir, name)); !os.IsNotExist(err) {
			t.Errorf("%s 应被删除: %v", name, err)
		}
	}
	if _, err := os.Stat(filepath.Join(frpDir, "notes.txt")); err != nil {
		t.Errorf("无关文件不应被删除: %v", err)
	}
}

// 载入时忽略所属 frpc 实例不存在的代理；删除 frpc 实例时与其代理在同一次保存中删除。
func TestOrphanProxies(t *testing.T) {
	dir := t.TempDir()
	cfg := `{"clients": [{"key": "c1", "name": "家里", "serverAddr": "a", "serverPort": 7000}],
  "proxies": [
    {"key": "p1", "client": "c1", "name": "ssh", "type": "tcp", "localIp": "127.0.0.1", "localPort": 22, "remotePort": 6022},
    {"key": "p2", "client": "gone", "name": "web", "type": "tcp", "localIp": "127.0.0.1", "localPort": 80, "remotePort": 6080}
  ]}`
	if err := os.WriteFile(filepath.Join(dir, "frp.json"), []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}
	m, srv, _ := newTestModule(t, dir)
	if len(m.cfg.Proxies) != 1 || m.cfg.Proxies[0].Key != "p1" {
		t.Fatalf("应忽略孤儿代理: %+v", m.cfg.Proxies)
	}
	if code := doJSON(t, "DELETE", srv.URL+"/api/frp/clients/c1", nil, nil); code != http.StatusOK {
		t.Fatalf("删除实例失败: %d", code)
	}
	m2, _, _ := newTestModule(t, dir)
	if len(m2.cfg.Clients) != 0 || len(m2.cfg.Proxies) != 0 {
		t.Fatalf("实例与其代理应一起删除并保存: %+v", m2.cfg)
	}
}

// 导入到已有实例时，导入后的设置也要通过校验；失败时配置不变。
func TestApplyImportValidatesTarget(t *testing.T) {
	m, _, _ := newTestModule(t, t.TempDir())
	m.cfg.Clients = []Client{{Key: "c1", Name: "家里", Enable: true, ClientSettings: ClientSettings{ServerAddr: "old", ServerPort: 7000}}}
	res, err := ParseImport(RoleClient, "serverPort = 7000\n")
	if err != nil {
		t.Fatal(err)
	}
	out := &ImportResponse{}
	m.mu.Lock()
	err = m.applyImportLocked(res, ImportRequest{Target: "c1"}, out)
	m.mu.Unlock()
	if !errors.Is(err, errInvalidImport) {
		t.Fatalf("缺少服务端地址时应拒绝导入到已启用实例: %v", err)
	}
	if m.cfg.Clients[0].ServerAddr != "old" {
		t.Fatalf("导入失败时配置不应改变: %+v", m.cfg.Clients[0])
	}
}
