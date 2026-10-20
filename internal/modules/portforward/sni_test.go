package portforward

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/binary"
	"io"
	"math/big"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"

	proxyproto "github.com/pires/go-proxyproto"
)

// captureClientHello 让 crypto/tls 客户端发出一个真实的 ClientHello 并返回原始字节。
func captureClientHello(t *testing.T, cfg *tls.Config) []byte {
	t.Helper()
	client, server := net.Pipe()
	defer server.Close()
	go func() {
		defer client.Close()
		_ = tls.Client(client, cfg).Handshake()
	}()
	_ = server.SetReadDeadline(time.Now().Add(3 * time.Second))
	raw, _, err := peekClientHello(server)
	if err != nil {
		t.Fatalf("读取 ClientHello 失败: %v", err)
	}
	return raw
}

func TestPeekClientHello(t *testing.T) {
	raw := captureClientHello(t, &tls.Config{ServerName: "VPN.Example.COM", NextProtos: []string{"h2", "http/1.1"}, InsecureSkipVerify: true})
	got, hello, err := peekClientHello(bytes.NewReader(append(append([]byte{}, raw...), "trailing"...)))
	if err != nil {
		t.Fatal(err)
	}
	if hello.ServerName != "vpn.example.com" {
		t.Fatalf("SNI = %q", hello.ServerName)
	}
	if strings.Join(hello.ALPN, ",") != "h2,http/1.1" {
		t.Fatalf("ALPN = %v", hello.ALPN)
	}
	// 只读到 ClientHello 结束，不能吞掉后续数据
	if !bytes.Equal(got, raw) {
		t.Fatalf("预读字节数 %d，期望 %d", len(got), len(raw))
	}

	// 不带 SNI（例如直接用 IP 访问）
	raw = captureClientHello(t, &tls.Config{InsecureSkipVerify: true})
	if _, hello, err = peekClientHello(bytes.NewReader(raw)); err != nil || hello.ServerName != "" {
		t.Fatalf("无 SNI: %q %v", hello.ServerName, err)
	}
}

// ClientHello 被拆成多条 TLS 记录时也要能拼接解析
func TestPeekClientHelloFragmented(t *testing.T) {
	raw := captureClientHello(t, &tls.Config{ServerName: "a.example.com", InsecureSkipVerify: true})
	msg := raw[tlsRecordHeaderLen:]
	var split []byte
	for len(msg) > 0 {
		n := min(len(msg), 100)
		hdr := []byte{0x16, 3, 1, 0, 0}
		binary.BigEndian.PutUint16(hdr[3:], uint16(n))
		split = append(split, hdr...)
		split = append(split, msg[:n]...)
		msg = msg[n:]
	}
	got, hello, err := peekClientHello(bytes.NewReader(split))
	if err != nil || hello.ServerName != "a.example.com" {
		t.Fatalf("分片解析失败: %q %v", hello.ServerName, err)
	}
	if !bytes.Equal(got, split) {
		t.Fatal("分片时预读字节应为全部记录")
	}
}

func TestPeekClientHelloNotTLS(t *testing.T) {
	got, _, err := peekClientHello(strings.NewReader("SSH-2.0-OpenSSH\r\n"))
	if err != errNotTLS || string(got) != "S" {
		t.Fatalf("非 TLS 应立即返回且只读 1 字节: %q %v", got, err)
	}
	// 截断的数据不能 panic
	raw := captureClientHello(t, &tls.Config{ServerName: "x.example.com", InsecureSkipVerify: true})
	for i := 0; i < len(raw); i += 7 {
		_, _, _ = peekClientHello(bytes.NewReader(raw[:i]))
	}
	// 篡改长度字段不能 panic
	for i := tlsRecordHeaderLen; i < len(raw); i++ {
		bad := append([]byte{}, raw...)
		bad[i] ^= 0xff
		_, _, _ = peekClientHello(bytes.NewReader(bad))
	}
}

func TestSNIRouterMatch(t *testing.T) {
	rule := Rule{
		Mode:            ModeSNI,
		Protocols:       []string{"tcp"},
		ListenPorts:     "443",
		TargetAddresses: []string{"127.0.0.9"},
		SNIRoutes: []SNIRoute{
			{Name: "exact", Domains: []string{"VPN.example.com."}, TargetAddresses: []string{"127.0.0.1"}},
			{Name: "wild", Domains: []string{"*.example.com"}, TargetAddresses: []string{"127.0.0.2"}},
			{Name: "deep", Domains: []string{"*.home.example.com"}, TargetAddresses: []string{"127.0.0.3"}},
			{Name: "any", Domains: []string{"*"}, TargetAddresses: []string{"127.0.0.4"}},
		},
	}
	if err := rule.Validate(); err != nil {
		t.Fatal(err)
	}
	rt := newSNIRouter(rule)
	cases := map[string]string{
		"vpn.example.com":      "exact",
		"a.example.com":        "wild",
		"a.b.example.com":      "wild",
		"nas.home.example.com": "deep",
		"example.com":          "any",
		"other.org":            "any",
		"":                     "",
	}
	for host, want := range cases {
		got := rt.match(host)
		if got == nil || got.name != want {
			t.Fatalf("%q 命中 %v，期望 %q", host, got, want)
		}
	}
	// 没有 SNI 时走默认目标
	if rt.match("") != rt.fallback {
		t.Fatal("无 SNI 应走默认目标")
	}
}

func TestSNIRuleValidate(t *testing.T) {
	base := func() Rule {
		return Rule{Mode: ModeSNI, Protocols: []string{"tcp"}, ListenPorts: "443",
			SNIRoutes: []SNIRoute{{Domains: []string{"a.example.com"}, TargetAddresses: []string{"127.0.0.1"}, TargetPort: 8443, ProxyProtocol: "v2"}}}
	}
	r := base()
	if err := r.Validate(); err != nil {
		t.Fatalf("合法规则被拒绝: %v", err)
	}
	bad := map[string]func(*Rule){
		"UDP": func(r *Rule) { r.Protocols = []string{"tcp", "udp"} },
		"重复域名": func(r *Rule) {
			r.SNIRoutes = append(r.SNIRoutes, SNIRoute{Domains: []string{"A.example.com"}, TargetAddresses: []string{"x"}})
		},
		"无域名":     func(r *Rule) { r.SNIRoutes[0].Domains = []string{" "} },
		"无目标":     func(r *Rule) { r.SNIRoutes[0].TargetAddresses = nil },
		"错误通配":    func(r *Rule) { r.SNIRoutes[0].Domains = []string{"a.*.com"} },
		"端口越界":    func(r *Rule) { r.SNIRoutes[0].TargetPort = 70000 },
		"PROXY版本": func(r *Rule) { r.SNIRoutes[0].ProxyProtocol = "v3" },
		"空规则":     func(r *Rule) { r.SNIRoutes = nil },
		"未知模式":    func(r *Rule) { r.Mode = "http" },
	}
	for name, mutate := range bad {
		r := base()
		mutate(&r)
		if r.Validate() == nil {
			t.Fatalf("%s: 应校验失败", name)
		}
	}
	// 只有默认目标也合法，等同于普通转发
	r = base()
	r.SNIRoutes = nil
	r.TargetAddresses = []string{"127.0.0.1"}
	if err := r.Validate(); err != nil {
		t.Fatal(err)
	}
}

// selfSignedTLS 启动一个 TLS 回显服务，握手后先写出 name 再回显，用于确认连接被分到哪个后端。
// acceptProxy 为 true 时在 TLS 之下解析 PROXY 协议，并把看到的客户端地址发到 seen。
func selfSignedTLS(t *testing.T, name string, acceptProxy bool, seen chan<- string) int {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tpl := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: name},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour), DNSNames: []string{"*"}}
	der, err := x509.CreateCertificate(rand.Reader, tpl, tpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	cert := tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}
	var ln net.Listener
	ln, err = net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	port := ln.Addr().(*net.TCPAddr).Port
	if acceptProxy {
		ln = &proxyproto.Listener{Listener: ln}
	}
	tln := tls.NewListener(ln, &tls.Config{Certificates: []tls.Certificate{cert}})
	go func() {
		for {
			c, err := tln.Accept()
			if err != nil {
				return
			}
			go func() {
				defer c.Close()
				if err := c.(*tls.Conn).Handshake(); err != nil {
					return
				}
				if seen != nil {
					seen <- c.RemoteAddr().String()
				}
				_, _ = c.Write([]byte(name + "\n"))
				_, _ = io.Copy(c, c)
			}()
		}
	}()
	return port
}

// 端到端：真实 TLS 握手经 SNI 分流透传到不同后端，各后端的 PROXY 协议设置互不影响
func TestSNIForward(t *testing.T) {
	seenA := make(chan string, 4)
	portA := selfSignedTLS(t, "anylink", true, seenA)
	portB := selfSignedTLS(t, "nginx", false, nil)
	portDefault := selfSignedTLS(t, "ocserv", false, nil)
	listenPort := freePort(t, "tcp")

	rule := Rule{
		Mode:            ModeSNI,
		Protocols:       []string{"tcp4"},
		ListenAddress:   "127.0.0.1",
		ListenPorts:     strconv.Itoa(listenPort),
		TargetAddresses: []string{"127.0.0.1"},
		TargetPorts:     strconv.Itoa(portDefault),
		SNIRoutes: []SNIRoute{
			{Name: "anylink", Domains: []string{"vpn.example.com"}, TargetAddresses: []string{"127.0.0.1"}, TargetPort: portA, ProxyProtocol: "v2"},
			{Name: "nginx", Domains: []string{"*.example.com"}, TargetAddresses: []string{"127.0.0.1"}, TargetPort: portB},
		},
	}
	if err := rule.Validate(); err != nil {
		t.Fatal(err)
	}
	r := testRunner(rule)
	if err := r.start(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer r.stop()

	addr := net.JoinHostPort("127.0.0.1", strconv.Itoa(listenPort))
	dial := func(sni string) string {
		t.Helper()
		conn, err := tls.DialWithDialer(&net.Dialer{Timeout: 3 * time.Second}, "tcp4", addr, &tls.Config{ServerName: sni, InsecureSkipVerify: true})
		if err != nil {
			t.Fatalf("%s 握手失败: %v", sni, err)
		}
		defer conn.Close()
		_ = conn.SetDeadline(time.Now().Add(3 * time.Second))
		line := make([]byte, 64)
		n, err := conn.Read(line)
		if err != nil {
			t.Fatalf("%s 读取失败: %v", sni, err)
		}
		// 握手后的数据双向透传
		payload := bytes.Repeat([]byte("x"), 256*1024)
		go func() { _, _ = conn.Write(payload) }()
		echo := make([]byte, len(payload))
		if _, err := io.ReadFull(conn, echo); err != nil || !bytes.Equal(echo, payload) {
			t.Fatalf("%s 回显失败: %v", sni, err)
		}
		return strings.TrimSpace(string(line[:n]))
	}
	if got := dial("vpn.example.com"); got != "anylink" {
		t.Fatalf("vpn.example.com 分到 %s", got)
	}
	select {
	case got := <-seenA:
		if !strings.HasPrefix(got, "127.0.0.1:") {
			t.Fatalf("PROXY 协议传递的客户端地址错误: %s", got)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("后端未收到连接")
	}
	if got := dial("Gitea.Example.com"); got != "nginx" {
		t.Fatalf("通配域名分到 %s", got)
	}
	if got := dial("other.org"); got != "ocserv" {
		t.Fatalf("未命中应走默认目标，实际 %s", got)
	}

	// 等统计落地后检查各路由命中数
	deadline := time.Now().Add(2 * time.Second)
	for r.stats.ActiveConns.Load() != 0 && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	s := r.snapshot()
	if len(s.RouteConns) != 2 || s.RouteConns[0] != 1 || s.RouteConns[1] != 1 || s.DefaultConns != 1 || s.TotalConns != 3 {
		t.Fatalf("统计错误: %+v", s)
	}
}

// 非 TLS 客户端：先发数据的立即走默认目标；不发数据的等待超时后走默认目标；没有默认目标时断开
func TestSNIForwardNonTLS(t *testing.T) {
	old := sniPeekTimeout
	sniPeekTimeout = 300 * time.Millisecond
	defer func() { sniPeekTimeout = old }()

	// 默认目标是一个先发言的服务，模拟 SSH
	ln, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go func() {
				defer c.Close()
				_, _ = c.Write([]byte("BANNER\n"))
				_, _ = io.Copy(c, c)
			}()
		}
	}()
	listenPort := freePort(t, "tcp")
	rule := Rule{Mode: ModeSNI, Protocols: []string{"tcp4"}, ListenAddress: "127.0.0.1", ListenPorts: strconv.Itoa(listenPort),
		TargetAddresses: []string{"127.0.0.1"}, TargetPorts: strconv.Itoa(ln.Addr().(*net.TCPAddr).Port),
		SNIRoutes: []SNIRoute{{Domains: []string{"a.example.com"}, TargetAddresses: []string{"127.0.0.1"}, TargetPort: 1}}}
	if err := rule.Validate(); err != nil {
		t.Fatal(err)
	}
	r := testRunner(rule)
	if err := r.start(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer r.stop()
	addr := net.JoinHostPort("127.0.0.1", strconv.Itoa(listenPort))

	// 客户端先发言：首字节不是 TLS，预读的数据也要完整送达
	conn, err := net.Dial("tcp4", addr)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = conn.Write([]byte("ping"))
	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	buf := make([]byte, len("BANNER\nping"))
	if _, err := io.ReadFull(conn, buf); err != nil || string(buf) != "BANNER\nping" {
		t.Fatalf("非 TLS 转发失败: %q %v", buf, err)
	}
	conn.Close()

	// 服务端先发言：等待超时后转到默认目标
	conn, err = net.Dial("tcp4", addr)
	if err != nil {
		t.Fatal(err)
	}
	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	buf = make([]byte, len("BANNER\n"))
	if _, err := io.ReadFull(conn, buf); err != nil || string(buf) != "BANNER\n" {
		t.Fatalf("超时后应转到默认目标: %q %v", buf, err)
	}
	conn.Close()

	// 没有默认目标：未命中直接断开
	r.stop()
	rule.TargetAddresses = nil
	r = testRunner(rule)
	if err := r.start(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer r.stop()
	conn, err = net.Dial("tcp4", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	_, _ = conn.Write([]byte("ping"))
	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	if n, err := conn.Read(make([]byte, 8)); err == nil {
		t.Fatalf("应断开连接，实际读到 %d 字节", n)
	}
	time.Sleep(50 * time.Millisecond)
	if r.stats.SNIRejected.Load() != 1 {
		t.Fatalf("拒绝计数 = %d", r.stats.SNIRejected.Load())
	}
}
