package webservice

import (
	"bufio"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/wangwanjie/gately/internal/logbuf"
)

// TestCloseSite 验证「断开连接」类型的默认站点对未匹配域名不返回任何响应，匹配的站点照常响应，
// 访问日志记为 444 并带上 Referer 与 User-Agent。
func TestCloseSite(t *testing.T) {
	logs := logbuf.New(50, log.New(io.Discard, "", 0))
	l := Listener{
		Sites:   []Site{{Enable: true, Domains: []string{"app.test"}, Type: TypeText, Text: "ok"}},
		Default: Site{Enable: true, Type: TypeClose, AccessLog: true, Compress: true},
	}
	addr, _ := startRunnerLogs(t, l, nil, nil, logs)

	if resp, body := get(t, noRedirect(), addr, "app.test", "/", nil); resp.StatusCode != 200 || body != "ok" {
		t.Fatalf("匹配的站点应正常响应: %d %q", resp.StatusCode, body)
	}

	// 用原始连接发请求，确认服务端一个字节都不写就断开
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
	_, _ = io.WriteString(conn, "GET /wp-login.php HTTP/1.1\r\nHost: scan.test\r\nReferer: http://r.test/\r\nUser-Agent: Scanner/1.0\r\n\r\n")
	b, err := io.ReadAll(bufio.NewReader(conn))
	if len(b) != 0 || err != nil {
		t.Fatalf("未匹配域名应直接断开连接，收到 %q（%v）", b, err)
	}

	req, _ := http.NewRequest("GET", "http://"+addr+"/", nil)
	req.Host = "other.test"
	if resp, err := http.DefaultClient.Do(req); err == nil {
		resp.Body.Close()
		t.Fatalf("客户端应收到连接错误，得到状态码 %d", resp.StatusCode)
	}

	got := logs.Query(0, 0, 0, &logbuf.Filter{Status: []logbuf.StatusRange{{Min: statusClosed, Max: statusClosed}}})
	// Go 客户端遇到连接被关闭会自动重试幂等请求，所以至少 2 条
	if len(got) < 2 {
		t.Fatalf("应至少有 2 条 444 访问日志: %+v", got)
	}
	if msg := got[0].Message; !strings.Contains(msg, `444`) || !strings.HasSuffix(msg, `"http://r.test/" "Scanner/1.0"`) {
		t.Fatalf("访问日志应带状态码、Referer 与 User-Agent: %q", msg)
	}
}

// TestAccessLogHeaders 验证访问日志末尾附带 Referer 与 User-Agent，缺失时记为 "-"，控制字符被转义。
func TestAccessLogHeaders(t *testing.T) {
	logs := logbuf.New(50, log.New(io.Discard, "", 0))
	l := Listener{Sites: []Site{{Enable: true, Domains: []string{"app.test"}, Type: TypeText, Text: "ok", AccessLog: true}}}
	addr, _ := startRunnerLogs(t, l, nil, nil, logs)
	get(t, noRedirect(), addr, "app.test", "/a", map[string]string{"User-Agent": "Mozilla/5.0 (iPhone)"})
	got := logs.Query(0, 0, 0, &logbuf.Filter{Host: "app.test"})
	if len(got) != 1 || !strings.HasSuffix(got[0].Message, `"-" "Mozilla/5.0 (iPhone)"`) {
		t.Fatalf("访问日志格式错误: %+v", got)
	}
	if orDash("") != "-" || orDash("x") != "x" {
		t.Fatal("orDash 错误")
	}
}

// TestImportNginx444 验证 nginx 的 return 444 导入为断开连接，不再改成 403。
func TestImportNginx444(t *testing.T) {
	rep, err := importNginx(`http { server { listen 80 default_server; server_name _; return 444; } }`, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, pl, s := findSite(rep, "默认站点")
	if pl == nil || !pl.Default || s.Type != TypeClose || len(pl.Warnings) != 0 {
		t.Fatalf("return 444 应导入为断开连接的默认站点: %+v %+v", pl, s)
	}
}

// wrapWAF 模拟 Coraza 的中间件：把 ResponseWriter 包成不带 Unwrap 的结构后再交给下一层。
type wrapWAF struct{ calls int }

func (f *wrapWAF) Middleware(_ string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.calls++
		next.ServeHTTP(struct{ http.ResponseWriter }{w}, r)
	})
}

// TestCloseSiteSkipsPipeline 验证断开连接的站点在访问控制、WAF 与认证之前直接断开：
// 不在白名单的地址也不会收到 403，WAF 包装了 ResponseWriter 时访问日志与统计仍记为 444；
// 其余设置不生效也不校验，开着网页认证但没有用户也能保存。
func TestCloseSiteSkipsPipeline(t *testing.T) {
	logs := logbuf.New(50, log.New(io.Discard, "", 0))
	fw := &wrapWAF{}
	l := Listener{Default: Site{Enable: true, Type: TypeClose, AccessLog: true, WAF: "k", CORS: true, WebAuth: true,
		IPMode: IPModeWhitelist, IPList: []string{"10.0.0.1"}}}
	ln, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	l.Port = ln.Addr().(*net.TCPAddr).Port
	ln.Close()
	l.ListenIP, l.Network = "127.0.0.1", "tcp4"
	if err := l.Validate(); err != nil {
		t.Fatal(err)
	}
	r, err := newListenerRunner(l, logs.Logger("t"), newAuthenticator("s"), fw)
	if err != nil {
		t.Fatal(err)
	}
	if err := r.start(nil, nil); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(r.stop)

	conn, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
	_, _ = io.WriteString(conn, "OPTIONS / HTTP/1.1\r\nHost: scan.test\r\nOrigin: http://o.test\r\nAccess-Control-Request-Method: GET\r\n\r\n")
	if b, err := io.ReadAll(bufio.NewReader(conn)); len(b) != 0 || err != nil {
		t.Fatalf("不在白名单的地址也应直接断开，收到 %q（%v）", b, err)
	}
	if fw.calls != 0 {
		t.Fatalf("断开连接的站点不应经过 WAF，调用了 %d 次", fw.calls)
	}
	got := logs.Query(0, 0, 0, &logbuf.Filter{Status: []logbuf.StatusRange{{Min: statusClosed, Max: statusClosed}}})
	if len(got) != 1 {
		t.Fatalf("应有 1 条 444 访问日志: %+v", got)
	}
}

// TestAccessLogClip 验证访问日志中过长的 URI、Referer 与 User-Agent 被截断，且不截断多字节字符。
func TestAccessLogClip(t *testing.T) {
	if clip("abc", 3) != "abc" || clip("abcd", 3) != "abc…" {
		t.Fatal("按字节截断错误")
	}
	// "中" 占 3 字节，截到 4 字节时只能保留一个完整的字
	if got := clip("中中", 4); got != "中…" {
		t.Fatalf("截断了多字节字符: %q", got)
	}
	logs := logbuf.New(50, log.New(io.Discard, "", 0))
	l := Listener{Sites: []Site{{Enable: true, Domains: []string{"app.test"}, Type: TypeText, Text: "ok", AccessLog: true}}}
	addr, _ := startRunnerLogs(t, l, nil, nil, logs)
	get(t, noRedirect(), addr, "app.test", "/"+strings.Repeat("p", 5000), map[string]string{"User-Agent": strings.Repeat("u", 5000)})
	got := logs.Query(0, 0, 0, &logbuf.Filter{Host: "app.test"})
	if len(got) != 1 || len(got[0].Message) > maxLogURI+2*maxLogHeader+200 || !strings.Contains(got[0].Message, "u…\"") {
		t.Fatalf("过长的字段应被截断: %d 字节", len(got[0].Message))
	}
}

// TestImportNginx444Locations 验证 server 级 return 444 的站点整体断开连接，location 不导入并给出提示。
func TestImportNginx444Locations(t *testing.T) {
	rep, err := importNginx(`http { server { listen 80; server_name a.test; return 444;
		location / { proxy_pass http://127.0.0.1:8080; }
		location /api { proxy_pass http://127.0.0.1:9090; } } }`, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, pl, s := findSite(rep, "a.test")
	if pl == nil || s.Type != TypeClose || len(s.Locations) != 0 || len(pl.Warnings) != 2 {
		t.Fatalf("server 级 return 444 应导入为断开连接且不导入 location: %+v %+v", pl, s)
	}
	rep, err = importNginx(`http { server { listen 80; server_name b.test;
		location / { return 444; }
		location /api { proxy_pass http://127.0.0.1:9090; } } }`, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, pl, s = findSite(rep, "b.test")
	if pl == nil || s.Type != TypeClose || len(s.Locations) != 0 || len(pl.Warnings) != 1 {
		t.Fatalf("location / 的 return 444 应导入为断开连接并提示其他 location: %+v %+v", pl, s)
	}
}
