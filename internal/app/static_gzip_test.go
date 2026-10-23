package app

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/wangwanjie/gately/internal/module"
	"github.com/wangwanjie/gately/internal/modules/portforward"
)

func gz(t *testing.T, b []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	_, _ = w.Write(b)
	_ = w.Close()
	return buf.Bytes()
}

// 静态资源有预压缩文件时直接返回 .gz；首页等动态内容由压缩中间件实时压缩。
func TestStaticCompression(t *testing.T) {
	js := []byte(strings.Repeat("console.log('gately');\n", 200))
	index := `<!doctype html><html><head><title>Gately</title><script type="module" src="./assets/a.js"></script></head><body>` +
		strings.Repeat("<!-- padding -->", 50) + `</body></html>`
	web := fstest.MapFS{
		"index.html":     {Data: []byte(index)},
		"assets/a.js":    {Data: js},
		"assets/a.js.gz": {Data: gz(t, js)},
	}
	a, err := New(Options{ConfigDir: t.TempDir(), Web: web, Modules: []module.Module{portforward.New()}})
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(a.Handler())
	defer srv.Close()

	get := func(path, accept string) *http.Response {
		req, _ := http.NewRequest(http.MethodGet, srv.URL+path, nil)
		if accept != "" {
			req.Header.Set("Accept-Encoding", accept)
		}
		// 关闭 Transport 自动解压，才能看到原始的 Content-Encoding
		resp, err := (&http.Transport{DisableCompression: true}).RoundTrip(req)
		if err != nil {
			t.Fatal(err)
		}
		return resp
	}

	// 接受 gzip：返回预压缩文件，类型按原文件
	resp := get("/assets/a.js", "gzip, deflate, br")
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.Header.Get("Content-Encoding") != "gzip" || !strings.Contains(resp.Header.Get("Content-Type"), "javascript") {
		t.Fatalf("预压缩资源响应头不对: %v", resp.Header)
	}
	if !bytes.Equal(body, web["assets/a.js.gz"].Data) {
		t.Fatal("应原样返回预压缩文件")
	}
	if !strings.Contains(resp.Header.Get("Cache-Control"), "immutable") {
		t.Fatal("assets 应保留长期缓存头")
	}

	// 不接受压缩：返回原文件
	resp = get("/assets/a.js", "")
	body, _ = io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.Header.Get("Content-Encoding") != "" || !bytes.Equal(body, js) {
		t.Fatal("不接受压缩时应返回原文件")
	}

	// 首页由中间件实时压缩
	resp = get("/", "gzip")
	body, _ = io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.Header.Get("Content-Encoding") != "gzip" {
		t.Fatalf("首页应被压缩: %v", resp.Header)
	}
	zr, err := gzip.NewReader(bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	plain, _ := io.ReadAll(zr)
	if !strings.Contains(string(plain), "<title>Gately</title>") {
		t.Fatal("解压后的首页内容不对")
	}
}

func TestAcceptsGzip(t *testing.T) {
	for accept, want := range map[string]bool{
		"":                   false,
		"br":                 false,
		"gzip":               true,
		"br, gzip;q=0.8":     true,
		"gzip;q=0":           false,
		"*":                  true,
		"deflate, GZIP":      true,
		"identity, gzip;q=0": false,
	} {
		if got := acceptsGzip(accept); got != want {
			t.Errorf("acceptsGzip(%q) = %v，期望 %v", accept, got, want)
		}
	}
}
