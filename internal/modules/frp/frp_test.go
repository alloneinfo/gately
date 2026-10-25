package frp

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/wangwanjie/gately/internal/config"
	"github.com/wangwanjie/gately/internal/logbuf"
	"github.com/wangwanjie/gately/internal/module"
)

func TestClientTOML(t *testing.T) {
	c := ClientSettings{ServerAddr: "frp.example.com", ServerPort: 7000, Token: `se"cret`, User: "me", Protocol: "quic"}
	got := ClientTOML(c, []Proxy{
		{Name: "ssh", Type: TypeTCP, Enable: true, LocalIP: "127.0.0.1", LocalPort: 22, RemotePort: 6022},
		{Name: "web", Type: TypeHTTP, Enable: true, LocalIP: "127.0.0.1", LocalPort: 80, CustomDomains: []string{"a.example.com", "b.example.com"}, Subdomain: "w"},
		{Name: "off", Type: TypeTCP, Enable: false, LocalIP: "127.0.0.1", LocalPort: 1, RemotePort: 1},
		{Name: "ssh", Type: TypeTCP, Enable: true, LocalIP: "127.0.0.1", LocalPort: 23, RemotePort: 6023},
		{Name: "sec", Type: TypeSTCP, Enable: true, LocalIP: "127.0.0.1", LocalPort: 3389, SecretKey: "k"},
	})
	for _, want := range []string{
		`serverAddr = "frp.example.com"`,
		`auth.token = "se\"cret"`,
		`transport.protocol = "quic"`,
		`user = "me"`,
		`remotePort = 6022`,
		`customDomains = ["a.example.com", "b.example.com"]`,
		`subdomain = "w"`,
		`secretKey = "k"`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("配置缺少 %s\n%s", want, got)
		}
	}
	if strings.Contains(got, `"off"`) || strings.Contains(got, "remotePort = 6023") {
		t.Errorf("停用与重名的代理不应写入\n%s", got)
	}
	// 顶层键必须出现在第一个 [[proxies]] 之前，否则会被解析为代理的字段
	if i := strings.Index(got, "[[proxies]]"); strings.Index(got, "auth.token") > i {
		t.Errorf("顶层键位置错误\n%s", got)
	}
}

func TestServerTOML(t *testing.T) {
	s := ServerSettings{BindPort: 7000, Token: "t", VhostHTTPPort: 8080, SubdomainHost: "example.com", AllowPorts: "6000-6100, 7500", DashboardPort: 7500, DashboardAddr: "127.0.0.1", DashboardUser: "a", DashboardPwd: "b"}
	got := ServerTOML(s)
	for _, want := range []string{
		"bindPort = 7000",
		"vhostHTTPPort = 8080",
		`subDomainHost = "example.com"`,
		`allowPorts = [{ start = 6000, end = 6100 }, { start = 7500, end = 7500 }]`,
		`webServer.port = 7500`,
		`webServer.password = "b"`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("配置缺少 %s\n%s", want, got)
		}
	}
	if strings.Contains(got, "vhostHTTPSPort") {
		t.Errorf("未设置的端口不应输出\n%s", got)
	}
}

func TestServerTOMLBindAddr(t *testing.T) {
	// 只限制连接端口时，代理端口仍须监听所有地址
	got := ServerTOML(ServerSettings{BindAddr: "127.0.0.1", BindPort: 7000})
	if !strings.Contains(got, `bindAddr = "127.0.0.1"`) || !strings.Contains(got, `proxyBindAddr = "0.0.0.0"`) {
		t.Errorf("监听地址输出错误\n%s", got)
	}
	got = ServerTOML(ServerSettings{BindAddr: "127.0.0.1", BindPort: 7000, ProxyBindAddr: "192.168.1.2"})
	if !strings.Contains(got, `proxyBindAddr = "192.168.1.2"`) {
		t.Errorf("代理监听地址输出错误\n%s", got)
	}
	got = ServerTOML(ServerSettings{BindPort: 7000})
	if strings.Contains(got, "bindAddr") || strings.Contains(got, "proxyBindAddr") {
		t.Errorf("未设置地址时不应输出\n%s", got)
	}
	for _, s := range []ServerSettings{{BindAddr: "localhost"}, {ProxyBindAddr: "1.2.3"}} {
		if err := s.Validate(); err == nil {
			t.Errorf("无效地址应报错: %+v", s)
		}
	}
	s := ServerSettings{BindAddr: " ::1 ", ProxyBindAddr: "::"}
	if err := s.Validate(); err != nil || s.BindAddr != "::1" {
		t.Errorf("IPv6 地址应通过: %v %+v", err, s)
	}
}

func TestParseAllowPorts(t *testing.T) {
	for _, bad := range []string{"abc", "0-10", "100-50", "1-70000", "6000-"} {
		if _, err := ParseAllowPorts(bad); err == nil {
			t.Errorf("%q 应被拒绝", bad)
		}
	}
	if r, err := ParseAllowPorts("80, 6000-6100"); err != nil || len(r) != 2 {
		t.Errorf("解析失败: %v %v", r, err)
	}
}

func TestProxyValidate(t *testing.T) {
	ok := Proxy{Name: "a", Type: TypeTCP, LocalPort: 22, RemotePort: 6022}
	if err := ok.Validate(); err != nil || ok.LocalIP != "127.0.0.1" {
		t.Errorf("合法配置被拒绝或未填默认值: %v %+v", err, ok)
	}
	bad := []Proxy{
		{Type: TypeTCP, LocalPort: 22, RemotePort: 1},
		{Name: "a", Type: "ftp", LocalPort: 22},
		{Name: "a", Type: TypeTCP, LocalPort: 0, RemotePort: 1},
		{Name: "a", Type: TypeTCP, LocalPort: 22},
		{Name: "a", Type: TypeHTTP, LocalPort: 80},
		{Name: "a", Type: TypeSTCP, LocalPort: 80},
	}
	for i, p := range bad {
		if err := p.Validate(); err == nil {
			t.Errorf("用例 %d 应被拒绝: %+v", i, p)
		}
	}
}

func TestServerValidate(t *testing.T) {
	s := ServerSettings{DashboardPort: 7500}
	if err := s.Validate(); err == nil {
		t.Error("启用控制台但没有凭据应被拒绝")
	}
	s = ServerSettings{}
	if err := s.Validate(); err != nil || s.BindPort != 7000 || s.DashboardAddr != "127.0.0.1" {
		t.Errorf("默认值错误: %v %+v", err, s)
	}
}

// TestModuleRunsFakeFrpc 用假的 frpc 脚本验证配置写入、进程启动与日志采集。
func TestModuleRunsFakeFrpc(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("依赖 sh")
	}
	dir := t.TempDir()
	bin := filepath.Join(dir, "frpc")
	script := "#!/bin/sh\necho \"2026-01-01 00:00:00.000 [I] [client] using $2\"\ncat \"$2\" | head -1\nexec sleep 30\n"
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	store, err := config.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	logs := logbuf.New(100, nil)
	m := New()
	if err := m.Init(module.Env{Store: store, Logs: logs}); err != nil {
		t.Fatal(err)
	}
	m.cfg.Client = ClientSettings{Enable: true, BinPath: bin, ServerAddr: "example.com", ServerPort: 7000}
	m.cfg.Proxies = []Proxy{{Key: "k", Name: "ssh", Type: TypeTCP, Enable: true, LocalIP: "127.0.0.1", LocalPort: 22, RemotePort: 6022}}
	if err := m.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer m.Stop()

	deadline := time.Now().Add(5 * time.Second)
	for !m.client.sup.State().Running && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if !m.client.sup.State().Running {
		t.Fatalf("frpc 未运行: %+v", m.client.sup.State())
	}
	data, err := os.ReadFile(filepath.Join(dir, "frp", "frpc.toml"))
	if err != nil || !strings.Contains(string(data), `serverAddr = "example.com"`) {
		t.Fatalf("配置文件错误: %v %s", err, data)
	}
	if st, _ := os.Stat(filepath.Join(dir, "frp", "frpc.toml")); st.Mode().Perm() != 0o600 {
		t.Errorf("配置文件含令牌，权限应为 0600，实际 %v", st.Mode().Perm())
	}
	var found bool
	for time.Now().Before(deadline) && !found {
		for _, e := range logs.Since(0) {
			if strings.Contains(e.Message, "serverAddr") {
				found = true
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !found {
		t.Errorf("未采集到进程输出: %+v", logs.Since(0))
	}
	// 停用后进程应被停止
	m.mu.Lock()
	m.cfg.Client.Enable = false
	m.applyClientLocked()
	m.mu.Unlock()
	if m.client.sup.State().Running {
		t.Error("停用后 frpc 仍在运行")
	}
}
