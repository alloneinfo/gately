// Package httpcompress 为 HTTP 响应提供 br / gzip 压缩，供 Web 服务站点与管理端共用。
package httpcompress

import (
	"bufio"
	"compress/gzip"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/andybalholm/brotli"
)

const MinSize = 256

var compressibleTypes = []string{
	"text/", "application/json", "application/javascript", "application/xml", "application/xhtml",
	"application/rss", "application/atom", "application/wasm", "image/svg+xml", "application/x-javascript",
}

func Compressible(contentType string) bool {
	ct := strings.ToLower(contentType)
	if strings.HasPrefix(ct, "text/event-stream") {
		return false
	}
	for _, p := range compressibleTypes {
		if strings.HasPrefix(ct, p) {
			return true
		}
	}
	return false
}

// Negotiate 根据 Accept-Encoding 选择 br 或 gzip，不支持时返回空串。
func Negotiate(accept string) string {
	var br, gz bool
	for _, part := range strings.Split(accept, ",") {
		name, params, _ := strings.Cut(strings.TrimSpace(part), ";")
		name = strings.ToLower(strings.TrimSpace(name))
		if q := strings.TrimSpace(params); strings.HasPrefix(q, "q=") {
			if v, err := strconv.ParseFloat(strings.TrimPrefix(q, "q="), 64); err == nil && v == 0 {
				continue
			}
		}
		switch name {
		case "br":
			br = true
		case "gzip", "*":
			gz = true
		}
	}
	switch {
	case br:
		return "br"
	case gz:
		return "gzip"
	}
	return ""
}

// compressWriter 在首次写出响应头时决定是否压缩。
type compressWriter struct {
	http.ResponseWriter
	req     *http.Request
	enc     string
	w       io.WriteCloser
	decided bool
}

func (c *compressWriter) decide(status int, sniff []byte) {
	if c.decided {
		return
	}
	c.decided = true
	h := c.Header()
	if h.Get("Content-Encoding") != "" || h.Get("Content-Range") != "" || c.req.Method == http.MethodHead ||
		status < 200 || status == http.StatusNoContent || status == http.StatusNotModified || status == http.StatusSwitchingProtocols {
		return
	}
	ct := h.Get("Content-Type")
	if ct == "" && len(sniff) > 0 {
		ct = http.DetectContentType(sniff)
	}
	if !Compressible(ct) {
		return
	}
	if n, err := strconv.Atoi(h.Get("Content-Length")); err == nil && n < MinSize {
		return
	}
	h.Del("Content-Length")
	h.Set("Content-Encoding", c.enc)
	h.Add("Vary", "Accept-Encoding")
	// 压缩后内容变化，强校验 ETag 不再适用
	if et := h.Get("Etag"); et != "" && !strings.HasPrefix(et, "W/") {
		h.Set("Etag", "W/"+et)
	}
}

func (c *compressWriter) WriteHeader(code int) {
	c.decide(code, nil)
	c.ResponseWriter.WriteHeader(code)
}

func (c *compressWriter) Write(p []byte) (int, error) {
	if !c.decided {
		c.decide(http.StatusOK, p)
		c.ResponseWriter.WriteHeader(http.StatusOK)
	}
	if c.Header().Get("Content-Encoding") != c.enc {
		return c.ResponseWriter.Write(p)
	}
	if c.w == nil {
		// 压缩器的内部缓冲较大（brotli 约数百 KB），从池中复用，避免每个响应重新分配
		if c.enc == "br" {
			bw := brotliPool.Get().(*brotli.Writer)
			bw.Reset(c.ResponseWriter)
			c.w = bw
		} else {
			gz := gzipPool.Get().(*gzip.Writer)
			gz.Reset(c.ResponseWriter)
			c.w = gz
		}
	}
	return c.w.Write(p)
}

var (
	brotliPool = sync.Pool{New: func() any { return brotli.NewWriterLevel(nil, 4) }}
	gzipPool   = sync.Pool{New: func() any {
		gz, _ := gzip.NewWriterLevel(nil, gzip.DefaultCompression)
		return gz
	}}
)

func (c *compressWriter) Flush() {
	if f, ok := c.w.(interface{ Flush() error }); ok {
		_ = f.Flush()
	}
	_ = http.NewResponseController(c.ResponseWriter).Flush()
}

func (c *compressWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return http.NewResponseController(c.ResponseWriter).Hijack()
}

func (c *compressWriter) Unwrap() http.ResponseWriter { return c.ResponseWriter }

func (c *compressWriter) finish() {
	if c.w == nil {
		return
	}
	_ = c.w.Close()
	// Close 之后放回池中，下次使用前会 Reset 到新的输出
	switch w := c.w.(type) {
	case *brotli.Writer:
		brotliPool.Put(w)
	case *gzip.Writer:
		gzipPool.Put(w)
	}
	c.w = nil
}

// Handler 对可压缩的响应启用 br / gzip。WebSocket 升级请求直接放行。
func Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		enc := Negotiate(r.Header.Get("Accept-Encoding"))
		if enc == "" || strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
			next.ServeHTTP(w, r)
			return
		}
		cw := &compressWriter{ResponseWriter: w, req: r, enc: enc}
		defer cw.finish()
		next.ServeHTTP(cw, r)
	})
}
