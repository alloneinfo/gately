package webservice

import (
	"bufio"
	"container/list"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/wangwanjie/gately/internal/httpcompress"
)

// maxCacheEntry 是单个缓存条目的上限，同时不超过总容量的 1/4。
const maxCacheEntry = 4 << 20

type cacheEntry struct {
	key     string
	status  int
	header  http.Header
	body    []byte
	expires time.Time
	stored  time.Time
}

// respCache 是按字节容量限制的 LRU 内存缓存。
type respCache struct {
	mu    sync.Mutex
	max   int64
	size  int64
	ttl   time.Duration
	ll    *list.List
	items map[string]*list.Element
	now   func() time.Time
}

func newRespCache(c CacheConfig) *respCache {
	return &respCache{max: int64(c.MaxMB) << 20, ttl: time.Duration(c.TTL) * time.Second, ll: list.New(), items: map[string]*list.Element{}, now: time.Now}
}

func (c *respCache) get(key string) *cacheEntry {
	c.mu.Lock()
	defer c.mu.Unlock()
	el, ok := c.items[key]
	if !ok {
		return nil
	}
	e := el.Value.(*cacheEntry)
	if c.now().After(e.expires) {
		c.removeLocked(el)
		return nil
	}
	c.ll.MoveToFront(el)
	return e
}

func (c *respCache) removeLocked(el *list.Element) {
	e := el.Value.(*cacheEntry)
	c.ll.Remove(el)
	delete(c.items, e.key)
	c.size -= int64(len(e.body))
}

func (c *respCache) put(e *cacheEntry) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if old, ok := c.items[e.key]; ok {
		c.removeLocked(old)
	}
	c.items[e.key] = c.ll.PushFront(e)
	c.size += int64(len(e.body))
	for c.size > c.max && c.ll.Len() > 0 {
		c.removeLocked(c.ll.Back())
	}
}

// requestCacheable 判断请求是否可以走缓存。带凭据或 Cookie 的请求内容可能因人而异，一律不缓存。
func requestCacheable(r *http.Request) bool {
	if r.Method != http.MethodGet {
		return false
	}
	if r.Header.Get("Authorization") != "" || r.Header.Get("Cookie") != "" || r.Header.Get("Range") != "" {
		return false
	}
	if strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
		return false
	}
	return !strings.Contains(strings.ToLower(r.Header.Get("Cache-Control")), "no-cache")
}

func cacheKey(r *http.Request) string {
	key := r.Host + "|" + r.URL.RequestURI() + "|" + httpcompress.Negotiate(r.Header.Get("Accept-Encoding")) + "|" + r.Header.Get("Accept-Encoding")
	// 按用户分流或传递用户名时，不同后端、不同用户的响应分开缓存
	if id, ok := r.Context().Value(ctxKeyIdentity{}).(requestIdentity); ok {
		key += "|" + id.cacheTag
	}
	return key
}

// freshness 根据响应头计算可缓存时长。ok 为 false 表示不可缓存。
func freshness(h http.Header, fallback time.Duration) (time.Duration, bool) {
	if h.Get("Set-Cookie") != "" || strings.TrimSpace(h.Get("Vary")) == "*" {
		return 0, false
	}
	var maxAge, sMax = -1, -1
	for _, d := range strings.Split(strings.ToLower(h.Get("Cache-Control")), ",") {
		d = strings.TrimSpace(d)
		switch {
		case d == "no-store" || d == "private" || d == "no-cache":
			return 0, false
		case strings.HasPrefix(d, "s-maxage="):
			if v, err := strconv.Atoi(strings.TrimPrefix(d, "s-maxage=")); err == nil {
				sMax = v
			}
		case strings.HasPrefix(d, "max-age="):
			if v, err := strconv.Atoi(strings.TrimPrefix(d, "max-age=")); err == nil {
				maxAge = v
			}
		}
	}
	switch {
	case sMax >= 0:
		return time.Duration(sMax) * time.Second, sMax > 0
	case maxAge >= 0:
		return time.Duration(maxAge) * time.Second, maxAge > 0
	}
	return fallback, fallback > 0
}

// captureWriter 在透传响应的同时收集内容，用于写入缓存。
type captureWriter struct {
	http.ResponseWriter
	limit   int
	status  int
	body    []byte
	over    bool
	started bool
}

func (c *captureWriter) WriteHeader(code int) {
	if !c.started {
		c.status = code
		c.started = true
	}
	c.ResponseWriter.WriteHeader(code)
}

func (c *captureWriter) Write(p []byte) (int, error) {
	if !c.started {
		c.status, c.started = http.StatusOK, true
	}
	if !c.over {
		if len(c.body)+len(p) > c.limit {
			c.over, c.body = true, nil
		} else {
			c.body = append(c.body, p...)
		}
	}
	return c.ResponseWriter.Write(p)
}

func (c *captureWriter) Flush() { _ = http.NewResponseController(c.ResponseWriter).Flush() }

func (c *captureWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	c.over = true
	return http.NewResponseController(c.ResponseWriter).Hijack()
}

func (c *captureWriter) Unwrap() http.ResponseWriter { return c.ResponseWriter }

// hopHeaders 不应进入缓存。
var hopHeaders = []string{"Connection", "Keep-Alive", "Proxy-Connection", "Transfer-Encoding", "Upgrade", "Date", "Set-Cookie"}

// wrap 返回带缓存的处理器。
func (c *respCache) wrap(next http.Handler) http.Handler {
	entryLimit := int(c.max / 4)
	if entryLimit > maxCacheEntry {
		entryLimit = maxCacheEntry
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !requestCacheable(r) {
			next.ServeHTTP(w, r)
			return
		}
		key := cacheKey(r)
		if e := c.get(key); e != nil {
			h := w.Header()
			for k, v := range e.header {
				h[k] = append([]string(nil), v...)
			}
			h.Set("X-Cache", "HIT")
			h.Set("Age", strconv.Itoa(int(c.now().Sub(e.stored).Seconds())))
			w.WriteHeader(e.status)
			_, _ = w.Write(e.body)
			return
		}
		cw := &captureWriter{ResponseWriter: w, limit: entryLimit}
		w.Header().Set("X-Cache", "MISS")
		next.ServeHTTP(cw, r)
		if cw.over || cw.status != http.StatusOK || len(cw.body) == 0 {
			return
		}
		h := w.Header()
		if strings.HasPrefix(strings.ToLower(h.Get("Content-Type")), "text/event-stream") {
			return
		}
		ttl, ok := freshness(h, c.ttl)
		if !ok {
			return
		}
		stored := h.Clone()
		stored.Del("X-Cache")
		stored.Del("Age")
		for _, k := range hopHeaders {
			stored.Del(k)
		}
		now := c.now()
		c.put(&cacheEntry{key: key, status: cw.status, header: stored, body: cw.body, stored: now, expires: now.Add(ttl)})
	})
}
