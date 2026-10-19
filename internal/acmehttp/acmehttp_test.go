package acmehttp

import (
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

func TestServe(t *testing.T) {
	Put("tok", "tok.auth")
	defer Delete("tok")
	rec := httptest.NewRecorder()
	if !Serve(rec, httptest.NewRequest("GET", PathPrefix+"tok", nil)) || rec.Body.String() != "tok.auth" {
		t.Fatalf("应返回密钥授权: %q", rec.Body.String())
	}
	if Serve(httptest.NewRecorder(), httptest.NewRequest("GET", PathPrefix+"other", nil)) {
		t.Fatal("未登记的 token 不应应答")
	}
	if Serve(httptest.NewRecorder(), httptest.NewRequest("GET", "/index.html", nil)) {
		t.Fatal("其他路径不应应答")
	}
}

func TestProviderStandalone(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()

	p := &Provider{Port: port}
	if err := p.Present("example.com", "t1", "t1.key"); err != nil {
		t.Fatal(err)
	}
	resp, err := http.Get("http://127.0.0.1:" + strconv.Itoa(port) + PathPrefix + "t1")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if string(body) != "t1.key" {
		t.Fatalf("body = %q", body)
	}
	_ = p.CleanUp("example.com", "t1", "t1.key")
	if _, err := http.Get("http://127.0.0.1:" + strconv.Itoa(port) + PathPrefix + "t1"); err == nil {
		t.Fatal("清理后临时服务应已关闭")
	}
}

func TestProviderPortInUse(t *testing.T) {
	ln, _ := net.Listen("tcp", ":0")
	defer ln.Close()
	p := &Provider{Port: ln.Addr().(*net.TCPAddr).Port}
	if err := p.Present("example.com", "t2", "k"); err != nil {
		t.Fatalf("端口被占用时应假定由其他服务应答: %v", err)
	}
	_ = p.CleanUp("example.com", "t2", "k")
}
