package app

import (
	"context"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/coder/websocket"

	"github.com/wangwanjie/gately/internal/logbuf"
	"github.com/wangwanjie/gately/internal/module"
	"github.com/wangwanjie/gately/internal/modules/portforward"
)

func TestPrefixCookiePath(t *testing.T) {
	cases := map[string]string{
		"gately_token=abc; Path=/; HttpOnly":                   "gately_token=abc; Path=/app/gately; HttpOnly",
		"n=1; Path=/api/login/oauth; Max-Age=600":              "n=1; Path=/app/gately/api/login/oauth; Max-Age=600",
		"Path=/; Path=/":                                       "Path=/; Path=/app/gately",
		"x=1; HttpOnly":                                        "x=1; HttpOnly",
		"x=1; path=/; Secure":                                  "x=1; Path=/app/gately; Secure",
		"x=1; Domain=example.com; Path=relative; SameSite=Lax": "x=1; Domain=example.com; Path=relative; SameSite=Lax",
	}
	for in, want := range cases {
		if got := prefixCookiePath(in, "/app/gately"); got != want {
			t.Errorf("prefixCookiePath(%q) = %q，期望 %q", in, got, want)
		}
	}
}

func TestNormalizeGatewayPrefix(t *testing.T) {
	for in, want := range map[string]string{"": "", "/": "", "/app/gately/": "/app/gately", " /x ": "/x"} {
		got, err := normalizeGatewayPrefix(in)
		if err != nil || got != want {
			t.Errorf("normalizeGatewayPrefix(%q) = %q, %v，期望 %q", in, got, err, want)
		}
	}
	for _, bad := range []string{"app", "/a?b", `/a"b`} {
		if _, err := normalizeGatewayPrefix(bad); err == nil {
			t.Errorf("normalizeGatewayPrefix(%q) 应报错", bad)
		}
	}
}

func TestListenUnix(t *testing.T) {
	// macOS 的 Socket 路径上限较短，使用短目录
	dir, err := os.MkdirTemp("", "gw")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	path := filepath.Join(dir, "g.sock")
	ln, err := listenUnix(path)
	if err != nil {
		t.Fatal(err)
	}
	// 模拟异常退出遗留的 Socket 文件：关闭时不删除
	ln.(*net.UnixListener).SetUnlinkOnClose(false)
	_ = ln.Close()
	ln, err = listenUnix(path)
	if err != nil {
		t.Fatalf("遗留的 Socket 文件应被替换: %v", err)
	}
	if fi, _ := os.Stat(path); fi.Mode().Perm() != 0o660 {
		t.Fatalf("Socket 权限应为 0660，实际 %v", fi.Mode().Perm())
	}
	_ = ln.Close()

	regular := filepath.Join(dir, "file")
	_ = os.WriteFile(regular, []byte("x"), 0o600)
	if _, err := listenUnix(regular); err == nil {
		t.Fatal("同名普通文件不应被覆盖")
	}
}

// newGatewayApp 创建带最小前端文件的实例，返回实例、直接访问的服务和经网关访问的服务。
func newGatewayApp(t *testing.T) (*App, *httptest.Server, *httptest.Server) {
	t.Helper()
	web := fstest.MapFS{
		"index.html":  {Data: []byte(`<!doctype html><html><head><title>Gately</title><script type="module" src="./assets/a.js"></script></head><body></body></html>`)},
		"assets/a.js": {Data: []byte("console.log(1)")},
		"favicon.svg": {Data: []byte("<svg/>")},
	}
	a, err := New(Options{ConfigDir: t.TempDir(), Web: web, Modules: []module.Module{portforward.New()}})
	if err != nil {
		t.Fatal(err)
	}
	a.logger = log.New(io.Discard, "", 0)
	if err := a.auth.ChangeCredentials(mustInitialPassword(t, a), "admin", "test-password"); err != nil {
		t.Fatal(err)
	}
	handler := a.Handler()
	direct := httptest.NewServer(handler)
	t.Cleanup(direct.Close)
	gw := httptest.NewServer(a.gatewayHandler("/app/gately", handler))
	t.Cleanup(gw.Close)
	return a, direct, gw
}

// gatewayRequest 模拟 fnOS 网关转发的请求：Host 不带端口，浏览器的 Origin 带端口。
func gatewayRequest(t *testing.T, c *http.Client, method, url, body, realIP string) *http.Response {
	t.Helper()
	var r io.Reader
	if body != "" {
		r = strings.NewReader(body)
	}
	req, _ := http.NewRequest(method, url, r)
	req.Host = "nas.example.com"
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", "https://nas.example.com:4433")
	}
	if realIP != "" {
		req.Header.Set("X-Real-IP", realIP)
	}
	resp, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { resp.Body.Close() })
	return resp
}

func TestGatewayServesPrefixedUI(t *testing.T) {
	a, direct, gw := newGatewayApp(t)
	// 开启安全入口：直接访问需要先经过入口，经网关访问不受影响
	a.settingsMu.Lock()
	a.settings.SafePath = "/entry"
	a.settingsMu.Unlock()

	jar, _ := cookiejar.New(nil)
	c := &http.Client{Jar: jar, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}

	resp := gatewayRequest(t, c, "GET", gw.URL+"/app/gately", "", "")
	if resp.StatusCode != http.StatusMovedPermanently || resp.Header.Get("Location") != "/app/gately/" {
		t.Fatalf("前缀不带斜杠时应跳转到 /app/gately/，实际 %d %q", resp.StatusCode, resp.Header.Get("Location"))
	}

	resp = gatewayRequest(t, c, "GET", gw.URL+"/app/gately/", "", "")
	page, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 || !strings.Contains(string(page), `<meta name="gately-base" content="/app/gately" />`) {
		t.Fatalf("经网关打开首页应注入路径前缀，实际 %d %s", resp.StatusCode, page)
	}
	if !strings.Contains(string(page), `src="/app/gately/assets/a.js"`) {
		t.Fatalf("资源路径应改写为带前缀的绝对路径: %s", page)
	}
	if resp = gatewayRequest(t, c, "GET", gw.URL+"/app/gately/assets/a.js", "", ""); resp.StatusCode != 200 {
		t.Fatalf("静态资源应可访问，实际 %d", resp.StatusCode)
	}

	// 直接访问的行为不变：仍需安全入口，首页不注入前缀，资源路径与以往一致
	resp, err := http.Get(direct.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("直接访问未经过安全入口应返回 404，实际 %d", resp.StatusCode)
	}
	a.settingsMu.Lock()
	a.settings.SafePath = ""
	a.settingsMu.Unlock()
	resp, err = http.Get(direct.URL + "/some/deep/path")
	if err != nil {
		t.Fatal(err)
	}
	page, _ = io.ReadAll(resp.Body)
	resp.Body.Close()
	if strings.Contains(string(page), "gately-base") || !strings.Contains(string(page), `src="/assets/a.js"`) {
		t.Fatalf("直接访问的首页应与以往一致: %s", page)
	}
	a.settingsMu.Lock()
	a.settings.SafePath = "/entry"
	a.settingsMu.Unlock()

	// 登录：Origin 带端口也能通过跨源校验，会话 Cookie 限定在前缀路径下
	resp = gatewayRequest(t, c, "POST", gw.URL+"/app/gately/api/login", `{"username":"admin","password":"test-password"}`, "")
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("经网关登录失败 %d %s", resp.StatusCode, body)
	}
	if sc := resp.Header.Get("Set-Cookie"); !strings.Contains(sc, "Path=/app/gately;") && !strings.HasSuffix(sc, "Path=/app/gately") {
		t.Fatalf("会话 Cookie 路径应带前缀: %q", sc)
	}
	if resp = gatewayRequest(t, c, "GET", gw.URL+"/app/gately/api/session", "", ""); resp.StatusCode != 200 {
		t.Fatalf("登录后应能访问接口，实际 %d", resp.StatusCode)
	}
	// 跨源请求仍被拒绝
	req, _ := http.NewRequest("POST", gw.URL+"/app/gately/api/logout", nil)
	req.Host = "nas.example.com"
	req.Header.Set("Origin", "https://evil.example.com")
	if resp, err = c.Do(req); err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("经网关的跨源请求应被拒绝，实际 %d", resp.StatusCode)
	}
}

func TestGatewayClientIPAndLANOnly(t *testing.T) {
	a, _, gw := newGatewayApp(t)
	a.settingsMu.Lock()
	a.settings.LANOnly = true
	a.settingsMu.Unlock()
	c := &http.Client{}
	if resp := gatewayRequest(t, c, "GET", gw.URL+"/app/gately/", "", "203.0.113.9"); resp.StatusCode != http.StatusForbidden {
		t.Fatalf("仅内网访问时外网地址应被拒绝，实际 %d", resp.StatusCode)
	}
	if resp := gatewayRequest(t, c, "GET", gw.URL+"/app/gately/", "", "192.168.1.5"); resp.StatusCode != 200 {
		t.Fatalf("仅内网访问时内网地址应放行，实际 %d", resp.StatusCode)
	}
	// fnOS 网关的 X-Forwarded-For 固定为 127.0.0.1：缺少 X-Real-IP 时不能因此被当作内网放行
	req, _ := http.NewRequest("GET", gw.URL+"/app/gately/", nil)
	req.Header.Set("X-Forwarded-For", "127.0.0.1")
	resp, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("只有回环转发地址时应视为未知地址并拒绝，实际 %d", resp.StatusCode)
	}
}

func TestGatewayWriterRewritesRedirects(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/x" {
			t.Errorf("前缀应被去掉，实际路径 %q", r.URL.Path)
		}
		if p, ok := gatewayPrefix(r); !ok || p != "/app/gately" {
			t.Errorf("请求上下文应标记网关前缀，实际 %q %v", p, ok)
		}
		http.SetCookie(w, &http.Cookie{Name: "n", Value: "1", Path: "/api/login/oauth"})
		http.Redirect(w, r, "/#/login?e=1", http.StatusFound)
	})
	a := &App{sysLogs: logbuf.New(10, log.New(io.Discard, "", 0))}
	h := a.gatewayHandler("/app/gately", next)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/app/gately/api/x", nil))
	if loc := rec.Header().Get("Location"); loc != "/app/gately/#/login?e=1" {
		t.Fatalf("站内重定向应补前缀，实际 %q", loc)
	}
	if sc := rec.Header().Get("Set-Cookie"); !strings.Contains(sc, "Path=/app/gately/api/login/oauth") {
		t.Fatalf("Cookie 路径应补前缀，实际 %q", sc)
	}

	// 网关已去掉前缀时同样可用
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/api/x", nil))
	if rec.Code != http.StatusFound {
		t.Fatalf("不带前缀的请求也应处理，实际 %d", rec.Code)
	}
}

func TestGatewayWebSocket(t *testing.T) {
	// 网关包装的 ResponseWriter 需要支持 Hijack，WebSocket 才能升级
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ws, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer ws.CloseNow()
		typ, msg, err := ws.Read(r.Context())
		if err == nil {
			_ = ws.Write(r.Context(), typ, append([]byte("echo:"), msg...))
		}
	})
	a := &App{sysLogs: logbuf.New(10, log.New(io.Discard, "", 0))}
	srv := httptest.NewServer(a.gatewayHandler("/app/gately", next))
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ws, _, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(srv.URL, "http")+"/app/gately/api/ws", nil)
	if err != nil {
		t.Fatalf("经网关建立 WebSocket 失败: %v", err)
	}
	defer ws.CloseNow()
	if err := ws.Write(ctx, websocket.MessageText, []byte("hi")); err != nil {
		t.Fatal(err)
	}
	if _, msg, err := ws.Read(ctx); err != nil || string(msg) != "echo:hi" {
		t.Fatalf("WebSocket 回显异常: %q %v", msg, err)
	}
}
