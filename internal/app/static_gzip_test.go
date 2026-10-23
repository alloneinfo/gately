package app

import (
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math/rand"
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

	// 只接受 gzip（经 HTTP 访问的浏览器）：同样原样返回预压缩文件，不能被中间件再压一次
	resp = get("/assets/a.js", "gzip, deflate")
	body, _ = io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.Header.Get("Content-Encoding") != "gzip" || !bytes.Equal(body, web["assets/a.js.gz"].Data) {
		t.Fatal("只接受 gzip 时应原样返回预压缩文件")
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
		"*;q=0, gzip":        true,
		"gzip;q=0, *":        false,
	} {
		if got := acceptsGzip(accept); got != want {
			t.Errorf("acceptsGzip(%q) = %v，期望 %v", accept, got, want)
		}
	}
}

// 构建产物只嵌入 .gz 时：接受 gzip 的请求原样返回 .gz，其余请求（含 Range）由服务端解压后返回。
func TestStaticGzipOnly(t *testing.T) {
	// 混入伪随机内容，让 .gz 超过 net/http 的写缓冲（约 2 KB），才能覆盖 Content-Length 的情形
	var sb strings.Builder
	rnd := rand.New(rand.NewSource(1))
	for i := 0; i < 600; i++ {
		fmt.Fprintf(&sb, ".g%x{color:#%06x}\n", rnd.Int63(), rnd.Intn(1<<24))
	}
	css := []byte(sb.String())
	web := fstest.MapFS{
		"index.html":      {Data: []byte(`<!doctype html><html><head></head><body></body></html>`)},
		"assets/b.css.gz": {Data: gz(t, css)},
	}
	a, err := New(Options{ConfigDir: t.TempDir(), Web: web, Modules: []module.Module{portforward.New()}})
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(a.Handler())
	defer srv.Close()

	get := func(accept, rng string) (*http.Response, []byte) {
		req, _ := http.NewRequest(http.MethodGet, srv.URL+"/assets/b.css", nil)
		if accept != "" {
			req.Header.Set("Accept-Encoding", accept)
		}
		if rng != "" {
			req.Header.Set("Range", rng)
		}
		resp, err := (&http.Transport{DisableCompression: true}).RoundTrip(req)
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return resp, body
	}

	resp, body := get("gzip", "")
	if resp.StatusCode != http.StatusOK || resp.Header.Get("Content-Encoding") != "gzip" || !bytes.Equal(body, web["assets/b.css.gz"].Data) {
		t.Fatalf("接受 gzip 时应返回 .gz: %d %v", resp.StatusCode, resp.Header)
	}

	// 预压缩响应超过写缓冲时也要带 Content-Length，不能退化为 chunked
	if resp.ContentLength != int64(len(web["assets/b.css.gz"].Data)) || len(resp.TransferEncoding) != 0 {
		t.Fatalf("预压缩响应应带 Content-Length: %d %v", resp.ContentLength, resp.TransferEncoding)
	}
	if !strings.Contains(resp.Header.Get("Vary"), "Accept-Encoding") {
		t.Fatalf("预压缩响应应带 Vary: %v", resp.Header)
	}

	resp, body = get("", "")
	// 回退到解压内容的响应同样要带 Vary，避免共享缓存把未压缩版本发给所有人
	if !strings.Contains(resp.Header.Get("Vary"), "Accept-Encoding") {
		t.Fatalf("解压回退的响应应带 Vary: %v", resp.Header)
	}
	if resp.StatusCode != http.StatusOK || resp.Header.Get("Content-Encoding") != "" || !bytes.Equal(body, css) {
		t.Fatalf("不接受 gzip 时应返回解压内容: %d %v", resp.StatusCode, resp.Header)
	}
	if !strings.Contains(resp.Header.Get("Content-Type"), "text/css") {
		t.Fatalf("Content-Type 应按原文件扩展名: %v", resp.Header)
	}

	resp, body = get("gzip", "bytes=0-9")
	if resp.StatusCode != http.StatusPartialContent || !bytes.Equal(body, css[:10]) {
		t.Fatalf("Range 请求应返回解压后的片段: %d %q", resp.StatusCode, body)
	}

	info, err := fs.Stat(gzipOnlyFS{fsys: web}, "assets/b.css")
	if err != nil || info.Size() != int64(len(css)) {
		t.Fatalf("Stat 应返回原始长度: %v %v", info, err)
	}
	if _, err := fs.Stat(gzipOnlyFS{fsys: web}, "assets/none.css"); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("不存在的文件应返回 ErrNotExist: %v", err)
	}
}
