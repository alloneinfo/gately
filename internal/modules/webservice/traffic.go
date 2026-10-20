package webservice

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/andybalholm/brotli"
)

// ---- 限流 ----

// bucket 是单个客户端的令牌桶与并发计数。
type bucket struct {
	tokens float64
	last   time.Time
	conns  int
}

// limiter 按客户端 IP 限制请求速率与并发数。
type limiter struct {
	mu       sync.Mutex
	rate     float64
	burst    float64
	maxConns int
	buckets  map[string]*bucket
	now      func() time.Time
}

func newLimiter(c RateLimit) *limiter {
	return &limiter{rate: c.Rate, burst: float64(c.Burst), maxConns: c.MaxConns, buckets: map[string]*bucket{}, now: time.Now}
}

// acquire 申请处理一个请求。成功时返回 release，调用方必须在请求结束后调用；
// 失败时返回建议的重试等待时间。
func (l *limiter) acquire(key string) (release func(), retry time.Duration, ok bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	if len(l.buckets) > 10000 {
		for k, b := range l.buckets {
			if b.conns == 0 && now.Sub(b.last) > 10*time.Minute {
				delete(l.buckets, k)
			}
		}
	}
	b := l.buckets[key]
	if b == nil {
		b = &bucket{tokens: l.burst, last: now}
		l.buckets[key] = b
	}
	if l.rate > 0 {
		b.tokens += now.Sub(b.last).Seconds() * l.rate
		if b.tokens > l.burst {
			b.tokens = l.burst
		}
		b.last = now
		if b.tokens < 1 {
			return nil, time.Duration((1 - b.tokens) / l.rate * float64(time.Second)), false
		}
	}
	if l.maxConns > 0 && b.conns >= l.maxConns {
		return nil, time.Second, false
	}
	if l.rate > 0 {
		b.tokens--
	}
	b.conns++
	return func() {
		l.mu.Lock()
		b.conns--
		l.mu.Unlock()
	}, 0, true
}

// ---- 重写 ----

type compiledRewrite struct {
	re       *regexp.Regexp
	to       string
	redirect bool
	code     int
}

func compileRewrites(rs []Rewrite) ([]compiledRewrite, error) {
	out := make([]compiledRewrite, 0, len(rs))
	for _, r := range rs {
		re, err := regexp.Compile(r.Pattern)
		if err != nil {
			return nil, err
		}
		out = append(out, compiledRewrite{re: re, to: r.To, redirect: r.Redirect, code: r.Code})
	}
	return out, nil
}

// redirectBeforeAuth 找到第一条匹配的重写规则，是跳转时直接写出跳转并返回 true；
// 是路径重写时不处理，留给认证之后的 applyRewrite，保持“命中第一条即停止”的顺序语义。
func (h *siteHandler) redirectBeforeAuth(w http.ResponseWriter, r *http.Request) bool {
	for _, rw := range h.rewrites {
		if rw.re.MatchString(r.URL.Path) {
			if rw.redirect {
				return h.applyRewrite(w, r)
			}
			return false
		}
	}
	return false
}

// applyRewrite 按顺序匹配重写规则。命中重定向时直接写出响应并返回 true；
// 命中重写时就地修改请求路径。
func (h *siteHandler) applyRewrite(w http.ResponseWriter, r *http.Request) bool {
	for _, rw := range h.rewrites {
		if !rw.re.MatchString(r.URL.Path) {
			continue
		}
		out := rw.re.ReplaceAllString(r.URL.Path, rw.to)
		if rw.redirect {
			// 跳转目标中的 {host} 替换为访客请求的域名，便于多个域名共用一条规则跳到各自的 HTTPS
			out = strings.ReplaceAll(out, hostPlaceholder, hostOnly(r.Host))
			if !strings.Contains(out, "?") && r.URL.RawQuery != "" {
				out += "?" + r.URL.RawQuery
			}
			http.Redirect(w, r, out, rw.code)
			return true
		}
		r.URL.Path = out
		r.URL.RawPath = ""
		return false
	}
	return false
}

// ---- 压缩 ----

const compressMinSize = 256

var compressibleTypes = []string{
	"text/", "application/json", "application/javascript", "application/xml", "application/xhtml",
	"application/rss", "application/atom", "application/wasm", "image/svg+xml", "application/x-javascript",
}

func compressible(contentType string) bool {
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

// negotiateEncoding 根据 Accept-Encoding 选择 br 或 gzip，不支持时返回空串。
func negotiateEncoding(accept string) string {
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
	if !compressible(ct) {
		return
	}
	if n, err := strconv.Atoi(h.Get("Content-Length")); err == nil && n < compressMinSize {
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

// compressHandler 对可压缩的响应启用 br / gzip。WebSocket 升级请求直接放行。
func compressHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		enc := negotiateEncoding(r.Header.Get("Accept-Encoding"))
		if enc == "" || strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
			next.ServeHTTP(w, r)
			return
		}
		cw := &compressWriter{ResponseWriter: w, req: r, enc: enc}
		defer cw.finish()
		next.ServeHTTP(cw, r)
	})
}

// retryAfterSeconds 把等待时间向上取整为秒。
func retryAfterSeconds(d time.Duration) string {
	s := int((d + time.Second - 1) / time.Second)
	if s < 1 {
		s = 1
	}
	return fmt.Sprint(s)
}

// limitKeyAddr 返回限流键。
func limitKeyAddr(a netip.Addr) string { return limitKey(a.String()) }
