package webservice

import (
	"bufio"
	"crypto/tls"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// 本文件覆盖替代 nginx 所需的能力：分路径后端、转发认证、HTTPS 后端 SNI、按域名跳转 HTTPS、静态站点 try_files 与热更新。

func TestLocations(t *testing.T) {
	main := namedBackend(t, "main")
	api := namedBackend(t, "api")
	var gotAuth string
	hdr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		_, _ = w.Write([]byte("main"))
	}))
	defer hdr.Close()
	l := Listener{Sites: []Site{
		{Enable: true, Domains: []string{"img.test"}, Targets: []string{main.URL},
			Locations: []Location{{Prefix: "/api/", Targets: []string{api.URL}}}},
		// 只在 /api/ 注入请求头，后端沿用站点后端
		{Enable: true, Domains: []string{"od.test"}, Targets: []string{hdr.URL},
			Locations: []Location{{Prefix: "/api/", RequestHeaders: []string{"Authorization: Bearer x"}}}},
	}}
	addr, _ := startRunner(t, l, nil, nil)
	for path, want := range map[string]string{"/": "main", "/api/convert": "api", "/api": "api", "/apix": "main", "/x/../api/y": "api"} {
		if _, body := get(t, http.DefaultClient, addr, "img.test", path, nil); body != want {
			t.Fatalf("%s 分到 %q，期望 %q", path, body, want)
		}
	}
	get(t, http.DefaultClient, addr, "od.test", "/api/run", nil)
	if gotAuth != "Bearer x" {
		t.Fatalf("/api/ 应注入请求头，实际 %q", gotAuth)
	}
	get(t, http.DefaultClient, addr, "od.test", "/index", nil)
	if gotAuth != "" {
		t.Fatalf("其他路径不应注入请求头，实际 %q", gotAuth)
	}

	bad := Site{Domains: []string{"a"}, Targets: []string{main.URL}, Locations: []Location{{Prefix: "api"}}}
	if bad.validate(false) == nil {
		t.Fatal("前缀不以 / 开头应校验失败")
	}
	bad.Locations = []Location{{Prefix: "/a", Targets: []string{"ftp://x"}}}
	if bad.validate(false) == nil {
		t.Fatal("非法后端应校验失败")
	}
}

// fakeAuthelia 模拟 Authelia 的 /api/authz/auth-request：带 session=ok Cookie 时通过并返回身份头，否则 401。
func fakeAuthelia(t *testing.T, seen *http.Header) *httptest.Server {
	t.Helper()
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*seen = r.Header.Clone()
		if c, err := r.Cookie("session"); err == nil && c.Value == "ok" {
			w.Header().Set("Remote-User", "alice")
			w.Header().Set("Remote-Groups", "admins")
			w.WriteHeader(http.StatusOK)
			return
		}
		if r.Header.Get("X-Test-Forbidden") != "" {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		w.WriteHeader(http.StatusUnauthorized)
	}))
	t.Cleanup(s.Close)
	return s
}

func TestForwardAuth(t *testing.T) {
	var authSeen http.Header
	authz := fakeAuthelia(t, &authSeen)
	var gotUser, gotPath string
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUser, gotPath = r.Header.Get("Remote-User"), r.URL.Path
		_, _ = w.Write([]byte("ok"))
	}))
	defer backend.Close()
	l := Listener{Sites: []Site{{
		Enable: true, Domains: []string{"code.test"}, Targets: []string{backend.URL},
		ForwardAuth: ForwardAuth{URL: authz.URL + "/api/authz/auth-request", LoginURL: "https://auth.test/?rd={url}&rm={method}"},
	}, {
		// headscale 场景：只有 /admin/ 需要认证，其余路径公开
		Enable: true, Domains: []string{"hs.test"}, Targets: []string{backend.URL}, AuthOptional: true,
		ForwardAuth: ForwardAuth{URL: authz.URL + "/api/authz/auth-request"},
		Rules:       []PathRule{{Prefix: "/admin/", Auth: RuleAuthOn}},
	}}}
	addr, _ := startRunner(t, l, nil, nil)

	// 未登录：跳到登录页并带上原始地址
	resp, _ := get(t, noRedirect(), addr, "code.test", "/edit?x=1", map[string]string{"Remote-User": "mallory"})
	if resp.StatusCode != http.StatusFound {
		t.Fatalf("未登录应 302，实际 %d", resp.StatusCode)
	}
	loc := resp.Header.Get("Location")
	if !strings.HasPrefix(loc, "https://auth.test/?rd=") || !strings.Contains(loc, url.QueryEscape("http://code.test/edit?x=1")) || !strings.HasSuffix(loc, "&rm=GET") {
		t.Fatalf("登录跳转地址错误: %s", loc)
	}
	if authSeen.Get("X-Original-URL") != "http://code.test/edit?x=1" || authSeen.Get("X-Original-Method") != "GET" || authSeen.Get("X-Forwarded-For") != "127.0.0.1" {
		t.Fatalf("认证子请求缺少原始信息: %v", authSeen)
	}
	if authSeen.Get("Remote-User") != "" {
		t.Fatal("访客自带的身份头不应传给认证服务")
	}

	// 已登录：身份头来自认证服务，访客伪造的被覆盖
	_, body := get(t, noRedirect(), addr, "code.test", "/edit", map[string]string{"Cookie": "session=ok", "Remote-User": "mallory"})
	if body != "ok" || gotUser != "alice" {
		t.Fatalf("已登录应放行并带上身份头: body=%q user=%q", body, gotUser)
	}

	// 403 等其他结果原样返回
	resp, _ = get(t, noRedirect(), addr, "code.test", "/", map[string]string{"X-Test-Forbidden": "1"})
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("认证服务的 403 应原样返回，实际 %d", resp.StatusCode)
	}

	// 未配置登录地址时 401 原样返回
	resp, _ = get(t, noRedirect(), addr, "hs.test", "/admin/", nil)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("/admin/ 未登录应 401，实际 %d", resp.StatusCode)
	}
	// 公开路径不经认证，但访客伪造的身份头仍要删除
	_, body = get(t, noRedirect(), addr, "hs.test", "/ts2021", map[string]string{"Remote-User": "mallory"})
	if body != "ok" || gotPath != "/ts2021" || gotUser != "" {
		t.Fatalf("公开路径处理错误: body=%q path=%q user=%q", body, gotPath, gotUser)
	}

	// 跳转类重写先于认证：根路径直接跳到 /app/，不要求登录
	l2 := Listener{Sites: []Site{{Enable: true, Domains: []string{"sb.test"}, Targets: []string{backend.URL},
		ForwardAuth: ForwardAuth{URL: authz.URL + "/api/authz/auth-request"},
		Rewrites:    []Rewrite{{Pattern: "^/$", To: "/app/", Redirect: true, Code: 302}}}}}
	addr2, _ := startRunner(t, l2, nil, nil)
	if resp, _ := get(t, noRedirect(), addr2, "sb.test", "/", nil); resp.StatusCode != 302 || resp.Header.Get("Location") != "/app/" {
		t.Fatalf("根路径应先跳转: %d %s", resp.StatusCode, resp.Header.Get("Location"))
	}
	if resp, _ := get(t, noRedirect(), addr2, "sb.test", "/app/", nil); resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("/app/ 仍需认证，实际 %d", resp.StatusCode)
	}

	// 认证服务不可用时返回 502
	authz.Close()
	resp, _ = get(t, noRedirect(), addr, "code.test", "/", map[string]string{"Cookie": "session=ok"})
	if resp.StatusCode != http.StatusBadGateway {
		t.Fatalf("认证服务不可用应 502，实际 %d", resp.StatusCode)
	}

	bad := Site{Domains: []string{"a"}, Targets: []string{backend.URL}, BasicAuth: true, Users: []string{"a:b"}, ForwardAuth: ForwardAuth{URL: authz.URL}}
	if bad.validate(false) == nil {
		t.Fatal("转发认证与基本认证同时启用应校验失败")
	}
}

// HTTPS 后端按 SNI 分流（frp vhost_https）：开启后 SNI 为访客域名，且不同域名的连接互不复用
func TestUpstreamSNIFromHost(t *testing.T) {
	sniSeen := make(chan string, 8)
	backend := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(r.TLS.ServerName + "|" + r.Host))
	}))
	backend.TLS = &tls.Config{GetConfigForClient: func(h *tls.ClientHelloInfo) (*tls.Config, error) {
		sniSeen <- h.ServerName
		return nil, nil
	}}
	backend.StartTLS()
	defer backend.Close()
	l := Listener{Sites: []Site{{
		Enable: true, Domains: []string{"*.home.test"}, Targets: []string{backend.URL}, InsecureSkipVerify: true, UpstreamSNIFromHost: true,
	}, {
		Enable: true, Domains: []string{"plain.test"}, Targets: []string{backend.URL}, InsecureSkipVerify: true,
	}}}
	addr, _ := startRunner(t, l, nil, nil)
	for _, host := range []string{"nas.home.test", "pve.home.test", "nas.home.test"} {
		_, body := get(t, http.DefaultClient, addr, host, "/", nil)
		if body != host+"|"+host {
			t.Fatalf("%s 的 SNI/Host 错误: %q", host, body)
		}
	}
	// nas 第二次请求复用了连接，只应有两次握手，且分别是两个域名
	got := map[string]bool{}
	for i := 0; i < 2; i++ {
		got[<-sniSeen] = true
	}
	if !got["nas.home.test"] || !got["pve.home.test"] {
		t.Fatalf("握手 SNI 错误: %v", got)
	}
	// 未开启时沿用后端地址（IP 不发送 SNI）
	if _, body := get(t, http.DefaultClient, addr, "plain.test", "/", nil); body != "|plain.test" {
		t.Fatalf("未开启时不应发送访客域名作为 SNI: %q", body)
	}
}

func TestRedirectHostPlaceholder(t *testing.T) {
	l := Listener{Default: Site{Enable: true, Type: TypeRedirect, RedirectTo: "https://{host}", KeepPath: true, RedirectCode: 301}}
	addr, _ := startRunner(t, l, nil, nil)
	resp, _ := get(t, noRedirect(), addr, "blog.test:80", "/a/b?c=1", nil)
	if resp.StatusCode != 301 || resp.Header.Get("Location") != "https://blog.test/a/b?c=1" {
		t.Fatalf("跳转错误: %d %s", resp.StatusCode, resp.Header.Get("Location"))
	}
}

// 端口 80 上保留外部 ACME 客户端（acme.sh webroot）的验证目录，其余路径按域名跳到 HTTPS
func TestRewriteHostPlaceholderWithACMEWebroot(t *testing.T) {
	root := t.TempDir()
	_ = os.MkdirAll(filepath.Join(root, ".well-known", "acme-challenge"), 0o755)
	_ = os.WriteFile(filepath.Join(root, ".well-known", "acme-challenge", "tok"), []byte("tok.key"), 0o644)
	l := Listener{Sites: []Site{{Enable: true, Domains: []string{"g.test", "doc.g.test"}, Type: TypeStatic, Root: root, Rewrites: []Rewrite{
		{Pattern: `^/(\.well-known/acme-challenge/.*)$`, To: "/$1"},
		{Pattern: `^.*$`, To: "https://{host}$0", Redirect: true, Code: 301},
	}}}}
	addr, _ := startRunner(t, l, nil, nil)
	if resp, body := get(t, noRedirect(), addr, "g.test", "/.well-known/acme-challenge/tok", nil); resp.StatusCode != 200 || body != "tok.key" {
		t.Fatalf("验证文件应直接返回: %d %q", resp.StatusCode, body)
	}
	resp, _ := get(t, noRedirect(), addr, "doc.g.test", "/x?y=1", nil)
	if resp.StatusCode != 301 || resp.Header.Get("Location") != "https://doc.g.test/x?y=1" {
		t.Fatalf("跳转错误: %d %s", resp.StatusCode, resp.Header.Get("Location"))
	}
}

func TestStaticCleanURLs(t *testing.T) {
	root := t.TempDir()
	write := func(name, body string) {
		p := filepath.Join(root, name)
		_ = os.MkdirAll(filepath.Dir(p), 0o755)
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("index.html", "home")
	write("guide.html", "guide")
	write("docs/index.html", "docs")
	write("assets/app.js", "js")
	write("404.html", "not found page")
	l := Listener{Sites: []Site{{Enable: true, Domains: []string{"g.test"}, Type: TypeStatic, Root: root, CleanURLs: true, NotFoundPage: "404.html",
		ResponseHeaders: []string{"Strict-Transport-Security: max-age=63072000"}}}}
	addr, _ := startRunner(t, l, nil, nil)
	cases := []struct {
		path, body string
		code       int
	}{
		{"/", "home", 200},
		{"/guide", "guide", 200},
		{"/docs", "docs", 200},
		{"/docs/", "docs", 200},
		{"/assets/app.js", "js", 200},
		{"/missing", "not found page", 404},
		{"/assets/", "not found page", 404},
	}
	for _, c := range cases {
		resp, body := get(t, noRedirect(), addr, "g.test", c.path, nil)
		if resp.StatusCode != c.code || body != c.body {
			t.Fatalf("%s: %d %q，期望 %d %q", c.path, resp.StatusCode, body, c.code, c.body)
		}
		if resp.Header.Get("Strict-Transport-Security") == "" {
			t.Fatalf("%s: 静态站点应带上响应头", c.path)
		}
	}
}

// 修改站点只替换站点表：监听不断开，已建立的长连接保持，新请求立即使用新配置
func TestHotReloadKeepsConnections(t *testing.T) {
	a := namedBackend(t, "a")
	b := namedBackend(t, "b")
	// 长连接后端：建立后持续回显
	echoLn, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer echoLn.Close()
	go func() {
		for {
			c, err := echoLn.Accept()
			if err != nil {
				return
			}
			go func() {
				defer c.Close()
				br := bufio.NewReader(c)
				if _, err := http.ReadRequest(br); err != nil {
					return
				}
				_, _ = c.Write([]byte("HTTP/1.1 101 Switching Protocols\r\nConnection: Upgrade\r\nUpgrade: echo\r\n\r\n"))
				buf := make([]byte, 64)
				for {
					n, err := br.Read(buf)
					if err != nil {
						return
					}
					_, _ = c.Write(buf[:n])
				}
			}()
		}
	}()
	cfg := Listener{Key: "l1", Sites: []Site{
		{Key: "s1", Enable: true, Domains: []string{"x.test"}, Targets: []string{a.URL}},
		{Key: "ws", Enable: true, Domains: []string{"ws.test"}, Targets: []string{"http://" + echoLn.Addr().String()}},
	}}
	addr, r := startRunner(t, cfg, nil, nil)

	ws, err := net.Dial("tcp4", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer ws.Close()
	_, _ = ws.Write([]byte("GET / HTTP/1.1\r\nHost: ws.test\r\nConnection: Upgrade\r\nUpgrade: echo\r\n\r\n"))
	wr := bufio.NewReader(ws)
	_ = ws.SetDeadline(time.Now().Add(5 * time.Second))
	resp, err := http.ReadResponse(wr, nil)
	if err != nil || resp.StatusCode != http.StatusSwitchingProtocols {
		t.Fatalf("升级失败: %v %v", resp, err)
	}

	if _, body := get(t, http.DefaultClient, addr, "x.test", "/", nil); body != "a" {
		t.Fatalf("初始应到 a，实际 %q", body)
	}
	before := r.current().sites[0].stats.Requests.Load()
	cfg = r.cfg
	cfg.Sites[0].Targets = []string{b.URL}
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
	if err := r.reload(cfg, nil); err != nil {
		t.Fatal(err)
	}
	if _, body := get(t, http.DefaultClient, addr, "x.test", "/", nil); body != "b" {
		t.Fatalf("热更新后应到 b，实际 %q", body)
	}
	if got := r.current().sites[0].stats.Requests.Load(); got != before+1 {
		t.Fatalf("热更新后统计应延续: %d -> %d", before, got)
	}
	// 热更新前建立的升级连接仍可用
	_, _ = ws.Write([]byte("still-alive"))
	buf := make([]byte, len("still-alive"))
	if _, err := ioReadFull(wr, buf); err != nil || string(buf) != "still-alive" {
		t.Fatalf("热更新不应断开已有连接: %q %v", buf, err)
	}
}

func ioReadFull(r *bufio.Reader, buf []byte) (int, error) {
	n := 0
	for n < len(buf) {
		m, err := r.Read(buf[n:])
		n += m
		if err != nil {
			return n, err
		}
	}
	return n, nil
}
