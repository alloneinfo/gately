package httpcompress

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// 上游已经是 gzip 且客户端只接受 gzip 时应原样透传，不能再压一层。
func TestKeepUpstreamEncoding(t *testing.T) {
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	_, _ = zw.Write([]byte(strings.Repeat("hello gately ", 200)))
	_ = zw.Close()
	pre := buf.Bytes()

	h := Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("Content-Encoding", "gzip")
		_, _ = w.Write(pre)
	}))
	for _, accept := range []string{"gzip", "gzip, deflate", "br, gzip"} {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Accept-Encoding", accept)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Header().Get("Content-Encoding") != "gzip" || !bytes.Equal(rec.Body.Bytes(), pre) {
			t.Fatalf("Accept-Encoding %q：上游 gzip 应原样返回，实际 %d 字节", accept, rec.Body.Len())
		}
	}

	// 普通响应仍正常压缩
	plain := strings.Repeat("hello gately ", 200)
	h = Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = io.WriteString(w, plain)
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Header().Get("Content-Encoding") != "gzip" {
		t.Fatal("普通响应应被压缩")
	}
	zr, err := gzip.NewReader(rec.Body)
	if err != nil {
		t.Fatal(err)
	}
	got, _ := io.ReadAll(zr)
	if string(got) != plain {
		t.Fatal("解压内容不对")
	}
}

// 先 Flush 再写正文时，发出的响应头必须已带 Content-Encoding，与随后写入的压缩数据一致。
func TestFlushBeforeWrite(t *testing.T) {
	body := strings.Repeat(`{"k":"v"}`, 100)
	h := Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.(http.Flusher).Flush()
		_, _ = io.WriteString(w, body)
	}))
	srv := httptest.NewServer(h)
	defer srv.Close()
	req, _ := http.NewRequest(http.MethodGet, srv.URL, nil)
	req.Header.Set("Accept-Encoding", "gzip")
	resp, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.Header.Get("Content-Encoding") != "gzip" {
		t.Fatalf("Flush 后的响应头缺少 Content-Encoding: %v", resp.Header)
	}
	zr, err := gzip.NewReader(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	got, _ := io.ReadAll(zr)
	if string(got) != body {
		t.Fatal("解压内容不对")
	}
}

// 已决定压缩但处理函数没有写正文时，应输出合法的空 gzip 流。
func TestEmptyCompressedBody(t *testing.T) {
	h := Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Header().Get("Content-Encoding") != "gzip" {
		t.Fatal("应声明 gzip")
	}
	zr, err := gzip.NewReader(rec.Body)
	if err != nil {
		t.Fatalf("空正文应是合法的 gzip 流: %v", err)
	}
	if got, err := io.ReadAll(zr); err != nil || len(got) != 0 {
		t.Fatalf("解压结果应为空: %q %v", got, err)
	}
}
