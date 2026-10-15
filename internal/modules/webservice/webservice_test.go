package webservice

import (
	"bufio"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	proxyproto "github.com/pires/go-proxyproto"

	"github.com/wangwanjie/gately/internal/config"
	"github.com/wangwanjie/gately/internal/logbuf"
	"github.com/wangwanjie/gately/internal/luckyimport"
	"github.com/wangwanjie/gately/internal/module"
)

func TestMatchAndRoute(t *testing.T) {
	cases := []struct {
		rule, host string
		want       int
	}{
		{"a.com", "a.com", 3},
		{"*.a.com", "x.a.com", 2},
		{"*.a.com", "x.y.a.com", 2},
		{"*.a.com", "a.com", 0},
		{"*", "anything", 1},
		{"a.com", "b.com", 0},
	}
	for _, c := range cases {
		if got := matchDomain(c.rule, c.host); got != c.want {
			t.Errorf("matchDomain(%q,%q)=%d want %d", c.rule, c.host, got, c.want)
		}
	}
	mk := func(key string, domains ...string) *siteHandler {
		return &siteHandler{site: Site{Key: key, Domains: domains}}
	}
	l := &listenerRunner{sites: []*siteHandler{mk("all", "*"), mk("wild", "*.a.com"), mk("deep", "*.x.a.com"), mk("exact", "www.a.com")}, deflt: mk("def")}
	for host, want := range map[string]string{"www.a.com:443": "exact", "y.x.a.com": "deep", "b.a.com": "wild", "other.org": "all"} {
		if got := l.route(host).site.Key; got != want {
			t.Errorf("route(%q)=%s want %s", host, got, want)
		}
	}
	l.sites = l.sites[1:]
	if l.route("other.org").site.Key != "def" {
		t.Error("无匹配时应使用默认站点")
	}
}

func TestValidate(t *testing.T) {
	ok := Listener{Port: 8080, Sites: []Site{{Domains: []string{" A.com "}, Targets: []string{"http://127.0.0.1:1"}}}}
	if err := ok.Validate(); err != nil {
		t.Fatal(err)
	}
	if ok.Network != "tcp" || ok.Sites[0].Type != TypeProxy || ok.Sites[0].Domains[0] != "a.com" || ok.Sites[0].Key == "" {
		t.Fatalf("默认值错误: %+v", ok)
	}
	bad := []Listener{
		{Port: 0},
		{Port: 80, Network: "udp"},
		{Port: 80, MinTLS: "2.0"},
		{Port: 80, Sites: []Site{{Targets: []string{"http://x"}}}},
		{Port: 80, Sites: []Site{{Domains: []string{"a.com"}, Targets: []string{"ftp://x"}}}},
		{Port: 80, Sites: []Site{{Domains: []string{"a.*.com"}, Targets: []string{"http://x"}}}},
		{Port: 80, Sites: []Site{{Domains: []string{"a.com"}, Type: TypeRedirect, RedirectTo: "/rel"}}},
		{Port: 80, Sites: []Site{{Domains: []string{"a.com"}, Type: TypeStatic, Root: "rel"}}},
		{Port: 80, Sites: []Site{{Domains: []string{"a.com"}, Targets: []string{"http://x"}, BasicAuth: true}}},
		{Port: 80, Sites: []Site{{Domains: []string{"a.com"}, Targets: []string{"http://x"}, IPMode: IPModeWhitelist, IPList: []string{"bad"}}}},
	}
	for i, l := range bad {
		if err := l.Validate(); err == nil {
			t.Errorf("第 %d 个非法配置未报错", i)
		}
	}
}

// startRunner 启动一个监听器并返回地址。
func startRunner(t *testing.T, l Listener, certs CertSource, trusted []string) (string, *listenerRunner) {
	t.Helper()
	ln, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	l.Port = ln.Addr().(*net.TCPAddr).Port
	ln.Close()
	l.ListenIP = "127.0.0.1"
	l.Network = "tcp4"
	if err := l.Validate(); err != nil {
		t.Fatal(err)
	}
	logs := logbuf.New(200, log.New(io.Discard, "", 0))
	r, err := newListenerRunner(l, logs.Logger("t"), newAuthenticator("secret"))
	if err != nil {
		t.Fatal(err)
	}
	if err := r.start(certs, trusted); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(r.stop)
	return fmt.Sprintf("127.0.0.1:%d", l.Port), r
}

// get 以指定 Host 访问。
func get(t *testing.T, c *http.Client, addr, host, path string, hdr map[string]string) (*http.Response, string) {
	t.Helper()
	req, _ := http.NewRequest("GET", "http://"+addr+path, nil)
	req.Host = host
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	resp, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	return resp, string(b)
}

func noRedirect() *http.Client {
	return &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
}

func TestSiteTypesAndHeaders(t *testing.T) {
	var gotHost, gotXFF, gotCustom string
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHost, gotXFF, gotCustom = r.Host, r.Header.Get("X-Forwarded-For"), r.Header.Get("X-Custom")
		w.Header().Set("Server", "backend")
		_, _ = w.Write([]byte("backend:" + r.URL.Path))
	}))
	defer backend.Close()
	root := t.TempDir()
	_ = os.WriteFile(filepath.Join(root, "index.html"), []byte("static-index"), 0o644)
	_ = os.MkdirAll(filepath.Join(root, "sub"), 0o755)
	_ = os.WriteFile(filepath.Join(root, "sub", "a.txt"), []byte("a"), 0o644)

	l := Listener{
		Sites: []Site{
			{Enable: true, Domains: []string{"app.test"}, Targets: []string{backend.URL}, ForwardedHeaders: true,
				RequestHeaders: []string{"X-Custom: yes"}, ResponseHeaders: []string{"Server:", "X-Gately: 1"}},
			{Enable: true, Domains: []string{"host.test"}, Targets: []string{backend.URL}, UseTargetHost: true},
			{Enable: true, Domains: []string{"go.test"}, Type: TypeRedirect, RedirectTo: "https://example.com/base", KeepPath: true, RedirectCode: 301},
			{Enable: true, Domains: []string{"files.test"}, Type: TypeStatic, Root: root},
			{Enable: true, Domains: []string{"txt.test"}, Type: TypeText, Text: "maintenance", StatusCode: 503},
			{Enable: false, Domains: []string{"off.test"}, Type: TypeText, Text: "x"},
		},
		Default: Site{Enable: true, Type: TypeText, Text: "default", StatusCode: 404},
	}
	addr, _ := startRunner(t, l, nil, nil)
	c := noRedirect()

	resp, body := get(t, c, addr, "app.test", "/hello", nil)
	if body != "backend:/hello" || gotHost != "app.test" || gotXFF != "127.0.0.1" || gotCustom != "yes" {
		t.Fatalf("反向代理异常: body=%q host=%q xff=%q custom=%q", body, gotHost, gotXFF, gotCustom)
	}
	if resp.Header.Get("Server") != "" || resp.Header.Get("X-Gately") != "1" {
		t.Fatalf("响应头改写失败: %v", resp.Header)
	}
	get(t, c, addr, "host.test", "/", nil)
	if gotHost != strings.TrimPrefix(backend.URL, "http://") {
		t.Fatalf("UseTargetHost 时 Host 应为后端地址，实际 %q", gotHost)
	}
	if gotXFF != "" {
		t.Fatalf("未开启转发头时不应添加 X-Forwarded-For: %q", gotXFF)
	}
	resp, _ = get(t, c, addr, "go.test", "/x?y=1", nil)
	if resp.StatusCode != 301 || resp.Header.Get("Location") != "https://example.com/base/x?y=1" {
		t.Fatalf("重定向异常: %d %s", resp.StatusCode, resp.Header.Get("Location"))
	}
	if _, body = get(t, c, addr, "files.test", "/", nil); body != "static-index" {
		t.Fatalf("静态首页异常: %q", body)
	}
	if resp, _ = get(t, c, addr, "files.test", "/sub/", nil); resp.StatusCode != 404 {
		t.Fatalf("关闭目录列表时目录应返回 404，实际 %d", resp.StatusCode)
	}
	if resp, body = get(t, c, addr, "txt.test", "/", nil); resp.StatusCode != 503 || body != "maintenance" {
		t.Fatalf("固定文本异常: %d %q", resp.StatusCode, body)
	}
	if _, body = get(t, c, addr, "off.test", "/", nil); body != "default" {
		t.Fatalf("停用站点应落到默认站点: %q", body)
	}
}

func TestIPFilterAndBasicAuth(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("ok")) }))
	defer backend.Close()
	l := Listener{Sites: []Site{
		{Enable: true, Domains: []string{"white.test"}, Targets: []string{backend.URL}, IPMode: IPModeWhitelist, IPList: []string{"10.0.0.0/8"}},
		{Enable: true, Domains: []string{"black.test"}, Targets: []string{backend.URL}, IPMode: IPModeBlacklist, IPList: []string{"10.0.0.0/8"}},
		{Enable: true, Domains: []string{"auth.test"}, Targets: []string{backend.URL}, BasicAuth: true, Users: []string{"alice:secret"}},
	}}
	addr, _ := startRunner(t, l, nil, nil)
	c := http.DefaultClient
	if resp, _ := get(t, c, addr, "white.test", "/", nil); resp.StatusCode != 403 {
		t.Fatalf("不在白名单应返回 403，实际 %d", resp.StatusCode)
	}
	if resp, _ := get(t, c, addr, "black.test", "/", nil); resp.StatusCode != 200 {
		t.Fatalf("不在黑名单应放行，实际 %d", resp.StatusCode)
	}
	if resp, _ := get(t, c, addr, "auth.test", "/", nil); resp.StatusCode != 401 || resp.Header.Get("WWW-Authenticate") == "" {
		t.Fatalf("未认证应返回 401，实际 %d", resp.StatusCode)
	}
	req, _ := http.NewRequest("GET", "http://"+addr+"/", nil)
	req.Host = "auth.test"
	req.SetBasicAuth("alice", "secret")
	resp, err := c.Do(req)
	if err != nil || resp.StatusCode != 200 {
		t.Fatalf("正确凭据应放行: %v %v", resp, err)
	}
	resp.Body.Close()
}

func TestWebAuthFlow(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("secret page " + r.URL.Path)) }))
	defer backend.Close()
	l := Listener{Sites: []Site{{Enable: true, Key: "w", Domains: []string{"web.test"}, Targets: []string{backend.URL}, WebAuth: true, Users: []string{"bob:pw"}}}}
	addr, _ := startRunner(t, l, nil, nil)
	jar, _ := cookiejar.New(nil)
	c := &http.Client{Jar: jar, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	browser := map[string]string{"User-Agent": "Mozilla/5.0", "Accept": "text/html"}

	resp, _ := get(t, c, addr, "web.test", "/dash?a=1", browser)
	if resp.StatusCode != 302 || !strings.HasPrefix(resp.Header.Get("Location"), authPath+"/login?next=") {
		t.Fatalf("浏览器未登录应跳转登录页: %d %s", resp.StatusCode, resp.Header.Get("Location"))
	}
	// 非浏览器回退到基本认证
	if resp, _ := get(t, c, addr, "web.test", "/", map[string]string{"User-Agent": "curl/8"}); resp.StatusCode != 401 {
		t.Fatalf("非浏览器应回退到基本认证: %d", resp.StatusCode)
	}

	login := func(user, pass, next string) *http.Response {
		form := url.Values{"username": {user}, "password": {pass}, "next": {next}}
		req, _ := http.NewRequest("POST", "http://"+addr+authPath+"/login", strings.NewReader(form.Encode()))
		req.Host = "web.test"
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp, err := c.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		return resp
	}
	if resp := login("bob", "wrong", "/dash"); resp.StatusCode != 401 {
		t.Fatalf("错误密码应返回 401: %d", resp.StatusCode)
	}
	// 开放重定向防护
	if resp := login("bob", "pw", "//evil.com"); resp.Header.Get("Location") != "/" {
		t.Fatalf("应拒绝站外跳转: %s", resp.Header.Get("Location"))
	}
	resp = login("bob", "pw", "/dash?a=1")
	if resp.StatusCode != 302 || resp.Header.Get("Location") != "/dash?a=1" {
		t.Fatalf("登录后应跳回原地址: %d %s", resp.StatusCode, resp.Header.Get("Location"))
	}
	cookies := resp.Cookies()
	if len(cookies) == 0 {
		t.Fatal("登录后应设置 Cookie")
	}
	req, _ := http.NewRequest("GET", "http://"+addr+"/dash", nil)
	req.Host = "web.test"
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.AddCookie(cookies[0])
	r2, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(r2.Body)
	r2.Body.Close()
	if string(body) != "secret page /dash" {
		t.Fatalf("携带会话应能访问: %d %q", r2.StatusCode, body)
	}
	// 会话绑定域名：换 Host 不可用
	req.Host = "other.test"
	l2 := &authenticator{secret: []byte("secret")}
	if l2.validCookie(req, Site{Key: "w", Users: []string{"bob:pw"}}) {
		t.Fatal("会话不应在其他域名生效")
	}
}

func TestWebSocketUpgrade(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Upgrade") != "websocket" {
			http.Error(w, "need upgrade", 400)
			return
		}
		conn, brw, err := http.NewResponseController(w).Hijack()
		if err != nil {
			return
		}
		defer conn.Close()
		_, _ = brw.WriteString("HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\n\r\n")
		_ = brw.Flush()
		line, _ := brw.ReadString('\n')
		_, _ = brw.WriteString("echo:" + line)
		_ = brw.Flush()
	}))
	defer backend.Close()
	addr, _ := startRunner(t, Listener{Sites: []Site{{Enable: true, Domains: []string{"ws.test"}, Targets: []string{backend.URL}}}}, nil, nil)
	c, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	_, _ = c.Write([]byte("GET /ws HTTP/1.1\r\nHost: ws.test\r\nUpgrade: websocket\r\nConnection: Upgrade\r\n\r\n"))
	br := bufio.NewReader(c)
	resp, err := http.ReadResponse(br, nil)
	if err != nil || resp.StatusCode != 101 {
		t.Fatalf("升级失败: %v %v", resp, err)
	}
	_, _ = c.Write([]byte("hi\n"))
	_ = c.SetReadDeadline(time.Now().Add(3 * time.Second))
	line, _ := br.ReadString('\n')
	if line != "echo:hi\n" {
		t.Fatalf("升级后数据转发失败: %q", line)
	}
}

type fakeCerts struct{ cert *tls.Certificate }

func (f fakeCerts) GetCertificate(h *tls.ClientHelloInfo) (*tls.Certificate, error) {
	if h.ServerName == "secure.test" {
		return f.cert, nil
	}
	return nil, fmt.Errorf("no cert")
}

func selfSigned(t *testing.T, name string) *tls.Certificate {
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	tmpl := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: name}, DNSNames: []string{name},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour)}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	return &tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}
}

func TestTLSListener(t *testing.T) {
	addr, _ := startRunner(t, Listener{TLS: true, MinTLS: "1.2", Sites: []Site{{Enable: true, Domains: []string{"secure.test"}, Type: TypeText, Text: "tls ok"}}},
		fakeCerts{selfSigned(t, "secure.test")}, nil)
	client := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true, ServerName: "secure.test"}}}
	req, _ := http.NewRequest("GET", "https://"+addr+"/", nil)
	req.Host = "secure.test"
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if string(body) != "tls ok" || resp.TLS == nil {
		t.Fatalf("HTTPS 访问异常: %q", body)
	}
	bad := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true, ServerName: "unknown.test"}}}
	if _, err := bad.Get("https://" + addr + "/"); err == nil {
		t.Fatal("没有证书的域名握手应失败")
	}
}

func TestProxyProtocol(t *testing.T) {
	// 后端用 PROXY 协议监听，记录收到的真实客户端地址
	raw, _ := net.Listen("tcp4", "127.0.0.1:0")
	pl := &proxyproto.Listener{Listener: raw}
	var mu sync.Mutex
	var seen []string
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		seen = append(seen, r.RemoteAddr)
		mu.Unlock()
		_, _ = w.Write([]byte("pp"))
	})}
	go srv.Serve(pl)
	defer srv.Close()

	l := Listener{
		AcceptProxyProtocol: true,
		Sites:               []Site{{Enable: true, Domains: []string{"pp.test"}, Targets: []string{"http://" + raw.Addr().String()}, SendProxyProtocol: true}},
	}
	addr, _ := startRunner(t, l, nil, []string{"127.0.0.1"})

	// 模拟 haproxy：先发 PROXY v2 头声明真实客户端 203.0.113.5
	sendVia := func(src string) string {
		c, err := net.Dial("tcp", addr)
		if err != nil {
			t.Fatal(err)
		}
		defer c.Close()
		h := proxyproto.HeaderProxyFromAddrs(2, &net.TCPAddr{IP: net.ParseIP(src), Port: 4321}, c.RemoteAddr())
		_, _ = h.WriteTo(c)
		_, _ = c.Write([]byte("GET / HTTP/1.1\r\nHost: pp.test\r\nConnection: close\r\n\r\n"))
		b, _ := io.ReadAll(c)
		return string(b)
	}
	if out := sendVia("203.0.113.5"); !strings.Contains(out, "pp") {
		t.Fatalf("请求失败: %s", out)
	}
	mu.Lock()
	got := append([]string{}, seen...)
	mu.Unlock()
	if len(got) != 1 || !strings.HasPrefix(got[0], "203.0.113.5:") {
		t.Fatalf("后端应收到真实客户端地址 203.0.113.5，实际 %v", got)
	}

	// 不可信来源的 PROXY 头会被忽略
	addr2, _ := startRunner(t, l, nil, []string{"10.0.0.1"})
	addr = addr2
	sendVia("198.51.100.9")
	mu.Lock()
	defer mu.Unlock()
	if len(seen) != 2 || !strings.HasPrefix(seen[1], "127.0.0.1:") {
		t.Fatalf("不可信来源的 PROXY 头应被忽略，后端看到 %v", seen)
	}
}

func TestConvertLucky(t *testing.T) {
	chars := strings.Split("bob:pw,amy:x1", "")
	raw, _ := json.Marshal([]map[string]any{{
		"RuleName": "main", "Enable": true, "ListenPort": 4434, "Network": "tcp6", "EnableTLS": true, "TLSMinVersion": 1, "Http3": true,
		"DefaultProxy": map[string]any{"Locations": []string{"http://192.168.1.8:8898"}, "WebServiceType": "reverseproxy"},
		"ProxyList": []map[string]any{
			{"Enable": true, "Domains": []string{"a.example.com"}, "Locations": []string{"https://192.168.1.9:5001"}, "WebServiceType": "reverseproxy",
				"LocationInsecureSkipVerify": true, "DisableKeepAlives": true, "CorazaWAFInstance": "main", "EnableAccessLog": true,
				"OtherParams": map[string]any{"ProxyProtocolV2": true}},
			{"Enable": true, "Domains": []string{"b.example.com"}, "Locations": []string{"http://192.168.1.10:80"}, "WebServiceType": "reverseproxy",
				"BasicAuthUserList": chars, "OtherParams": map[string]any{"WebAuth": true, "AllowAllThirdUsers": true}},
			{"Enable": true, "Domains": []string{"c.example.com"}, "WebServiceType": "fileServer"},
		},
	}})
	ls, warns, found, err := convertLuckyListeners(raw)
	if err != nil {
		t.Fatal(err)
	}
	if found != 1 || len(ls) != 1 {
		t.Fatalf("found=%d imported=%d %v", found, len(ls), warns)
	}
	l := ls[0]
	if l.Enable || l.Network != "tcp6" || !l.TLS || l.MinTLS != "1.1" || len(l.Sites) != 2 || !l.Default.Enable || l.Default.Targets[0] != "http://192.168.1.8:8898" {
		t.Fatalf("监听器映射错误: %+v", l)
	}
	a, b := l.Sites[0], l.Sites[1]
	if !a.SendProxyProtocol || !a.InsecureSkipVerify || !a.DisableKeepAlives || !a.AccessLog {
		t.Fatalf("站点 a 映射错误: %+v", a)
	}
	if !b.WebAuth || len(b.Users) != 2 || b.Users[0] != "bob:pw" || b.Users[1] != "amy:x1" {
		t.Fatalf("按字符存储的用户列表应还原: %+v", b.Users)
	}
	joined := strings.Join(warns, "\n")
	for _, w := range []string{"HTTP/3", "fileServer", "WAF", "第三方登录", "默认停用"} {
		if !strings.Contains(joined, w) {
			t.Errorf("缺少告警 %q:\n%s", w, joined)
		}
	}
}

// TestImportRealSample 仅在设置 GATELY_LUCKY_WEB_SAMPLE 时运行。
func TestImportRealSample(t *testing.T) {
	path := os.Getenv("GATELY_LUCKY_WEB_SAMPLE")
	if path == "" {
		t.Skip("未设置 GATELY_LUCKY_WEB_SAMPLE")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := luckyimport.Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	st, _ := config.Open(t.TempDir())
	m := New(nil)
	if err := m.Init(module.Env{Store: st, Logs: logbuf.New(10, log.New(io.Discard, "", 0))}); err != nil {
		t.Fatal(err)
	}
	rep, err := m.ImportLucky(doc, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, w := range rep.Warnings {
		t.Log("warning:", w)
	}
	t.Logf("found=%d imported=%d", rep.Found, rep.Imported)
	for _, l := range m.cfg.Listeners {
		t.Logf("%s :%d tls=%v sites=%d default=%v", l.Name, l.Port, l.TLS, len(l.Sites), l.Default.Enable)
	}
	if rep.Imported != rep.Found {
		t.Fatal("有监听规则未能导入")
	}
}
