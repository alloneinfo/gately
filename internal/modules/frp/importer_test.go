package frp

import (
	"slices"
	"strings"
	"testing"

	"github.com/wangwanjie/gately/internal/config"
	"github.com/wangwanjie/gately/internal/logbuf"
	"github.com/wangwanjie/gately/internal/module"
)

// 旧版 frpc.ini：含 [common]、注释、多种代理类型与 frp 不支持的字段
const legacyClient = `[common]
server_addr = 203.0.113.10
server_port = 17000
privilege_token = a&*^b#c
authentication_method = token
tls_enable = true
# log_level = erro

# 公网通过 ssh 访问内部服务器
[pve-ssh]
type = tcp
local_ip = 192.168.18.18
local_port = 22
remote_port = 6022

[pve]
type = https
local_ip = 192.168.18.18
local_port = 8006
subdomain = pve
use_encryption = true

[gitea]
type = http
local_ip = 192.168.18.8
local_port = 8887
custom_domains = gitea.example.com, git.example.com

[sec]
type = stcp
sk = abc
local_port = 3389

[range:ports]
type = tcp
local_port = 6000-6006
remote_port = 6000-6006

[bad-plugin]
type = tcp
plugin = http_proxy
remote_port = 7000

[pve-ssh]
type = tcp
local_port = 23
remote_port = 6023
`

// 旧版语法放进 .toml 文件，值带引号
const legacyServer = `[common]
bind_port = 17000
vhost_http_port = 8082
vhost_https_port = 8083
dashboard_port = 7500
dashboard_user = "admin"
dashboard_pwd = "p@ss#1"
authentication_method = "token"
tls_enable = true
privilege_token = "&*^&**&**^%&&*(&*454789)"
max_pool_count = 5
log_max_days = 3
allow_ports = 6000-6100, 7000
subdomain_host = "stun.example.com"`

func TestImportLegacyClient(t *testing.T) {
	res, err := ParseImport(RoleClient, legacyClient)
	if err != nil {
		t.Fatal(err)
	}
	c := res.Client
	// 值中的 # 不是注释，旧版语法只有整行注释
	if c.ServerAddr != "203.0.113.10" || c.ServerPort != 17000 || c.Token != "a&*^b#c" {
		t.Errorf("连接参数错误: %+v", c)
	}
	var names []string
	for _, p := range res.Proxies {
		names = append(names, p.Name)
	}
	if !slices.Equal(names, []string{"pve-ssh", "pve", "gitea", "sec"}) {
		t.Fatalf("代理列表错误: %v", names)
	}
	byName := map[string]Proxy{}
	for _, p := range res.Proxies {
		byName[p.Name] = p
		if !p.Enable {
			t.Errorf("导入的代理应启用: %s", p.Name)
		}
	}
	if p := byName["pve-ssh"]; p.RemotePort != 6022 || p.LocalIP != "192.168.18.18" || p.LocalPort != 22 {
		t.Errorf("tcp 代理错误: %+v", p)
	}
	if p := byName["pve"]; p.Type != TypeHTTPS || p.Subdomain != "pve" {
		t.Errorf("子域名代理错误: %+v", p)
	}
	if p := byName["gitea"]; !slices.Equal(p.CustomDomains, []string{"gitea.example.com", "git.example.com"}) {
		t.Errorf("自定义域名错误: %+v", p)
	}
	if p := byName["sec"]; p.SecretKey != "abc" || p.LocalIP != "127.0.0.1" {
		t.Errorf("stcp 代理错误: %+v", p)
	}
	all := strings.Join(res.Warnings, "\n")
	for _, want := range []string{"range:ports", "已跳过代理 bad-plugin", "pve-ssh 重复", "use_encryption（1 个代理）"} {
		if !strings.Contains(all, want) {
			t.Errorf("提示缺少 %q:\n%s", want, all)
		}
	}
	if strings.Contains(all, "tls_enable") || strings.Contains(all, "authentication_method") {
		t.Errorf("已处理的字段不应提示:\n%s", all)
	}
}

func TestImportLegacyServer(t *testing.T) {
	res, err := ParseImport(RoleServer, legacyServer)
	if err != nil {
		t.Fatal(err)
	}
	s := res.Server
	if s.BindPort != 17000 || s.VhostHTTPPort != 8082 || s.VhostHTTPSPort != 8083 || s.SubdomainHost != "stun.example.com" {
		t.Errorf("端口或域名错误: %+v", s)
	}
	if s.Token != "&*^&**&**^%&&*(&*454789)" || s.DashboardPwd != "p@ss#1" || s.DashboardUser != "admin" {
		t.Errorf("凭据错误: %+v", s)
	}
	if s.AllowPorts != "6000-6100,7000" {
		t.Errorf("允许端口错误: %q", s.AllowPorts)
	}
	// 旧版控制台默认监听所有网卡，导入时保留并提示
	if s.DashboardAddr != "0.0.0.0" {
		t.Errorf("旧版控制台监听地址应为 0.0.0.0: %q", s.DashboardAddr)
	}
	all := strings.Join(res.Warnings, "\n")
	for _, want := range []string{"监听所有网卡", "max_pool_count", "log_max_days"} {
		if !strings.Contains(all, want) {
			t.Errorf("提示缺少 %q:\n%s", want, all)
		}
	}
}

// 新版 TOML：Gately 自己生成的配置应能原样导回
func TestImportRoundTrip(t *testing.T) {
	c := ClientSettings{ServerAddr: "frp.example.com", ServerPort: 7000, Token: `se"cret`, User: "me", Protocol: "quic"}
	proxies := []Proxy{
		{Name: "ssh", Type: TypeTCP, Enable: true, LocalIP: "10.0.0.2", LocalPort: 22, RemotePort: 6022},
		{Name: "web", Type: TypeHTTP, Enable: true, LocalIP: "127.0.0.1", LocalPort: 80, CustomDomains: []string{"a.example.com", "b.example.com"}, Subdomain: "w"},
		{Name: "sec", Type: TypeSTCP, Enable: true, LocalIP: "127.0.0.1", LocalPort: 3389, SecretKey: "k"},
	}
	res, err := ParseImport(RoleClient, ClientTOML(c, proxies))
	if err != nil {
		t.Fatal(err)
	}
	if *res.Client != c {
		t.Errorf("客户端设置不一致: %+v", *res.Client)
	}
	if len(res.Proxies) != 3 || res.Proxies[1].Subdomain != "w" || !slices.Equal(res.Proxies[1].CustomDomains, proxies[1].CustomDomains) || res.Proxies[2].SecretKey != "k" {
		t.Errorf("代理不一致: %+v", res.Proxies)
	}
	if len(res.Warnings) != 0 {
		t.Errorf("回导不应有提示: %v", res.Warnings)
	}

	s := ServerSettings{BindPort: 7000, Token: "t", VhostHTTPPort: 8080, SubdomainHost: "example.com", AllowPorts: "6000-6100,7500", DashboardPort: 7500, DashboardAddr: "127.0.0.1", DashboardUser: "a", DashboardPwd: "b"}
	sres, err := ParseImport(RoleServer, ServerTOML(s))
	if err != nil {
		t.Fatal(err)
	}
	if *sres.Server != s {
		t.Errorf("服务端设置不一致: %+v", *sres.Server)
	}
}

func TestImportMultilineAndTables(t *testing.T) {
	res, err := ParseImport(RoleServer, `bindPort = 7000 # 注释
allowPorts = [
  { start = 2000, end = 3000 },
  { single = 3001 }, # 单端口
]
[auth]
token = "x # y"
[webServer]
addr = "127.0.0.1"
port = 7500
user = "u"
password = "p"
`)
	if err != nil {
		t.Fatal(err)
	}
	s := res.Server
	if s.AllowPorts != "2000-3000,3001" || s.Token != "x # y" || s.DashboardPort != 7500 || s.DashboardUser != "u" {
		t.Errorf("解析错误: %+v", s)
	}
}

func TestImportRejectsWrongRoleAndEmpty(t *testing.T) {
	if _, err := ParseImport(RoleClient, legacyServer); err == nil || !strings.Contains(err.Error(), "服务端") {
		t.Errorf("把 frps 配置导入客户端应被拒绝: %v", err)
	}
	if _, err := ParseImport(RoleServer, legacyClient); err == nil || !strings.Contains(err.Error(), "客户端") {
		t.Errorf("把 frpc 配置导入服务端应被拒绝: %v", err)
	}
	if _, err := ParseImport(RoleClient, "  \n"); err == nil {
		t.Error("空内容应被拒绝")
	}
	if _, err := ParseImport("x", "a = 1"); err == nil {
		t.Error("未知角色应被拒绝")
	}
	if _, err := ParseImport(RoleClient, "serverAddr = \"a\"\nnot a pair"); err == nil {
		t.Error("无法解析的行应报错")
	}
}

func TestApplyImport(t *testing.T) {
	store, err := config.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	m := New()
	if err := m.Init(module.Env{Store: store, Logs: logbuf.New(100, nil)}); err != nil {
		t.Fatal(err)
	}
	m.cfg.Client = ClientSettings{Enable: false, BinPath: "/opt/frpc", ServerAddr: "old"}
	m.cfg.Proxies = []Proxy{{Key: "k1", Name: "pve-ssh", Type: TypeTCP, Enable: false, LocalIP: "1.1.1.1", LocalPort: 1, RemotePort: 1}}

	res, err := ParseImport(RoleClient, legacyClient)
	if err != nil {
		t.Fatal(err)
	}
	out := &ImportResponse{Warnings: res.Warnings}
	m.mu.Lock()
	err = m.applyImportLocked(res, ProxyAppend, out)
	m.mu.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	// 追加：同名代理保持原样，其余 3 条新增
	if out.Added != 3 || len(m.cfg.Proxies) != 4 || m.cfg.Proxies[0].LocalIP != "1.1.1.1" {
		t.Errorf("追加结果错误: added=%d %+v", out.Added, m.cfg.Proxies)
	}
	if m.cfg.Client.ServerAddr != "203.0.113.10" || m.cfg.Client.BinPath != "/opt/frpc" || m.cfg.Client.Enable {
		t.Errorf("应覆盖连接参数并沿用启用状态与路径: %+v", m.cfg.Client)
	}
	for _, p := range m.cfg.Proxies[1:] {
		if p.Key == "" {
			t.Errorf("新增代理缺少 key: %+v", p)
		}
	}

	// 替换：全部以导入为准
	out = &ImportResponse{}
	m.mu.Lock()
	err = m.applyImportLocked(res, ProxyReplace, out)
	m.mu.Unlock()
	if err != nil || out.Added != 4 || len(m.cfg.Proxies) != 4 || m.cfg.Proxies[0].LocalIP != "192.168.18.18" {
		t.Errorf("替换结果错误: %v added=%d %+v", err, out.Added, m.cfg.Proxies)
	}
}
