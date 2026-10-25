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
