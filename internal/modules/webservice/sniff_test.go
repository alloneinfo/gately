package webservice

import (
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	proxyproto "github.com/pires/go-proxyproto"
)

func plainHTTPListener(mode string) Listener {
	return Listener{TLS: true, PlainHTTP: mode, Sites: []Site{{Enable: true, Domains: []string{"secure.test"}, Type: TypeText, Text: "mixed ok"}}}
}

func tlsClient() *http.Client {
	return &http.Client{
		Transport:     &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true, ServerName: "secure.test"}, ForceAttemptHTTP2: true},
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
}

// doReq 以 Host secure.test 访问，返回响应与正文。
func doReq(t *testing.T, c *http.Client, method, url string) (*http.Response, string) {
	t.Helper()
	req, _ := http.NewRequest(method, url, nil)
	req.Host = "secure.test"
	resp, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp, string(b)
}

func TestPlainHTTPServe(t *testing.T) {
	addr, _ := startRunner(t, plainHTTPListener(PlainHTTPServe), fakeCerts{selfSigned(t, "secure.test")}, nil)
	c := tlsClient()
	// 同一端口：HTTPS（含 HTTP/2 协商）与明文 HTTP 都能访问到站点
	resp, body := doReq(t, c, "GET", "https://"+addr+"/")
	if body != "mixed ok" || resp.TLS == nil || resp.ProtoMajor != 2 {
		t.Fatalf("HTTPS 访问异常: %q proto=%s", body, resp.Proto)
	}
	resp, body = doReq(t, c, "GET", "http://"+addr+"/")
	if body != "mixed ok" || resp.TLS != nil || resp.StatusCode != 200 {
		t.Fatalf("明文 HTTP 访问异常: %d %q", resp.StatusCode, body)
	}
}

func TestPlainHTTPRedirect(t *testing.T) {
	addr, _ := startRunner(t, plainHTTPListener(PlainHTTPRedirect), fakeCerts{selfSigned(t, "secure.test")}, nil)
	c := tlsClient()
	resp, _ := doReq(t, c, "GET", "http://"+addr+"/a/b?x=1")
	if resp.StatusCode != http.StatusMovedPermanently || resp.Header.Get("Location") != "https://secure.test/a/b?x=1" {
		t.Fatalf("GET 跳转异常: %d %s", resp.StatusCode, resp.Header.Get("Location"))
	}
	resp, _ = doReq(t, c, "POST", "http://"+addr+"/p")
	if resp.StatusCode != http.StatusPermanentRedirect {
		t.Fatalf("POST 应使用 308: %d", resp.StatusCode)
	}
	// Host 带端口时跳转保留端口
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = conn.Write([]byte("GET /q HTTP/1.1\r\nHost: secure.test:8443\r\nConnection: close\r\n\r\n"))
	raw, _ := io.ReadAll(conn)
	conn.Close()
	if !strings.Contains(string(raw), "Location: https://secure.test:8443/q") {
		t.Fatalf("跳转未保留端口: %q", raw)
	}
	// HTTPS 照常提供站点
	resp, body := doReq(t, c, "GET", "https://"+addr+"/")
	if body != "mixed ok" || resp.TLS == nil {
		t.Fatalf("HTTPS 访问异常: %q", body)
	}
}

func TestPlainHTTPOff(t *testing.T) {
	addr, _ := startRunner(t, plainHTTPListener(""), fakeCerts{selfSigned(t, "secure.test")}, nil)
	req, _ := http.NewRequest("GET", "http://"+addr+"/", nil)
	req.Host = "secure.test"
	resp, err := http.DefaultClient.Do(req)
	if err == nil {
		resp.Body.Close()
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("未开启时明文 HTTP 不应访问到站点: %d", resp.StatusCode)
		}
	}
}

func TestPlainHTTPValidate(t *testing.T) {
	l := Listener{Port: 80, PlainHTTP: PlainHTTPServe}
	if err := l.Validate(); err != nil || l.PlainHTTP != "" {
		t.Fatalf("未开启 HTTPS 时应清空选项: %v %q", err, l.PlainHTTP)
	}
	l = Listener{Port: 443, TLS: true, PlainHTTP: "bogus"}
	if err := l.Validate(); err == nil {
		t.Fatal("非法取值应报错")
	}
}

// 迟迟不发数据的连接不影响其他连接接入，超时后被关闭
func TestSniffSlowClient(t *testing.T) {
	old := sniffTimeout
	sniffTimeout = 300 * time.Millisecond
	t.Cleanup(func() { sniffTimeout = old })
	addr, _ := startRunner(t, plainHTTPListener(PlainHTTPServe), fakeCerts{selfSigned(t, "secure.test")}, nil)
	idle, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer idle.Close()
	if _, body := doReq(t, tlsClient(), "GET", "http://"+addr+"/"); body != "mixed ok" {
		t.Fatalf("空闲连接阻塞了其他连接: %q", body)
	}
	_ = idle.SetReadDeadline(time.Now().Add(3 * time.Second))
	if _, err := idle.Read(make([]byte, 1)); err == nil {
		t.Fatal("空闲连接应在超时后被关闭")
	} else if ne, ok := err.(net.Error); ok && ne.Timeout() {
		t.Fatal("空闲连接未被服务端关闭")
	}
}

// PROXY 协议头在协议识别之前解析，HTTPS 与明文 HTTP 都能拿到真实来源
func TestSniffWithProxyProtocol(t *testing.T) {
	l := plainHTTPListener(PlainHTTPServe)
	l.AcceptProxyProtocol = true
	l.Sites[0].Type = TypeText
	addr, _ := startRunner(t, l, fakeCerts{selfSigned(t, "secure.test")}, []string{"127.0.0.1"})
	send := func(useTLS bool) string {
		c, err := net.Dial("tcp", addr)
		if err != nil {
			t.Fatal(err)
		}
		defer c.Close()
		h := proxyproto.HeaderProxyFromAddrs(2, &net.TCPAddr{IP: net.ParseIP("203.0.113.9"), Port: 1}, c.RemoteAddr())
		_, _ = h.WriteTo(c)
		var rw net.Conn = c
		if useTLS {
			tc := tls.Client(c, &tls.Config{InsecureSkipVerify: true, ServerName: "secure.test"})
			if err := tc.Handshake(); err != nil {
				t.Fatal(err)
			}
			rw = tc
		}
		_, _ = rw.Write([]byte("GET / HTTP/1.1\r\nHost: secure.test\r\nConnection: close\r\n\r\n"))
		b, _ := io.ReadAll(rw)
		return string(b)
	}
	for _, useTLS := range []bool{true, false} {
		if out := send(useTLS); !strings.Contains(out, "mixed ok") {
			t.Fatalf("tls=%v 经 PROXY 协议访问失败: %q", useTLS, out)
		}
	}
}
