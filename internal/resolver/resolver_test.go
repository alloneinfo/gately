package resolver

import (
	"context"
	"encoding/binary"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestParse(t *testing.T) {
	cases := map[string]Server{
		"223.5.5.5":                 {Addr: "223.5.5.5:53"},
		"1.1.1.1:5353":              {Addr: "1.1.1.1:5353"},
		"2400:3200::1":              {Addr: "[2400:3200::1]:53"},
		"doh@223.5.5.5":             {URL: "https://223.5.5.5/dns-query"},
		"doh@dns.alidns.com/x":      {URL: "https://dns.alidns.com/x"},
		"https://doh.pub/dns-query": {URL: "https://doh.pub/dns-query"},
	}
	for in, want := range cases {
		got, err := Parse(in)
		if err != nil || got != want {
			t.Errorf("Parse(%q) = %+v, %v", in, got, err)
		}
	}
	for _, bad := range []string{"", "dns.example.com", "tls@1.1.1.1"} {
		if _, err := Parse(bad); err == nil {
			t.Errorf("Parse(%q) 应报错", bad)
		}
	}
}

// buildAnswer 构造一个 A 记录响应。
func buildAnswer(q []byte, ip net.IP) []byte {
	// 只保留报头与问题部分（Go 会附带 EDNS 记录）
	end := 12
	for q[end] != 0 {
		end += int(q[end]) + 1
	}
	end += 5
	resp := append([]byte(nil), q[:end]...)
	resp[2] |= 0x80                          // QR
	binary.BigEndian.PutUint16(resp[6:], 1)  // ANCOUNT
	binary.BigEndian.PutUint16(resp[10:], 0) // ARCOUNT
	ans := []byte{0xc0, 0x0c, 0, 1, 0, 1, 0, 0, 0, 60, 0, 4}
	return append(append(resp, ans...), ip.To4()...)
}

func TestDoHResolve(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "application/dns-message" {
			w.WriteHeader(400)
			return
		}
		q, _ := io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/dns-message")
		_, _ = w.Write(buildAnswer(q, net.IPv4(203, 0, 113, 7)))
	}))
	defer srv.Close()
	old := bootstrap
	bootstrap = srv.Client()
	defer func() { bootstrap = old }()

	// 第一个服务器不可用，应自动切换到第二个
	r := New([]Server{{Addr: "127.0.0.1:1"}, {URL: srv.URL + "/dns-query"}})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	ips, err := r.LookupIP(ctx, "ip4", "example.test")
	if err != nil || len(ips) != 1 || !ips[0].Equal(net.IPv4(203, 0, 113, 7)) {
		t.Fatalf("DoH 解析失败: %v %v", ips, err)
	}
}

// 真实 DoH 服务器（需要网络）：GATELY_NET_TEST=1
func TestRealDoH(t *testing.T) {
	if os.Getenv("GATELY_NET_TEST") == "" {
		t.Skip("设置 GATELY_NET_TEST=1 启用")
	}
	srv, _ := Parse("doh@223.5.5.5")
	r := New([]Server{srv})
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	ips, err := r.LookupHost(ctx, "www.aliyun.com")
	if err != nil || len(ips) == 0 {
		t.Fatalf("%v %v", ips, err)
	}
	t.Log(ips)
}
