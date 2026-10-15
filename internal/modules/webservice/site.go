package webservice

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/netip"
	"net/url"
	"strings"
	"sync/atomic"
	"time"

	proxyproto "github.com/pires/go-proxyproto"

	"github.com/wangwanjie/gately/internal/logbuf"
)

// SiteStats 是站点的请求统计。
type SiteStats struct {
	Requests atomic.Int64
	Active   atomic.Int64
	Errors   atomic.Int64
}

// WAFProvider 提供按实例检查请求的中间件（由 WAF 模块实现）。
type WAFProvider interface {
	Middleware(key string, next http.Handler) http.Handler
}

// siteHandler 是一个站点编译后的处理器。
type siteHandler struct {
	site    Site
	ipList  []netip.Prefix
	stats   *SiteStats
	log     *logbuf.Logger
	auth    *authenticator
	handler http.Handler
	// pipeline 为 WAF（可选）→ 跨域 → 认证 → 站点处理器
	pipeline http.Handler
	targets  []*url.URL
	next     atomic.Uint64
}

type ctxKeyClientAddr struct{}

// clientAddr 返回请求的客户端地址（PROXY 协议处理后的真实地址）。
func clientAddr(r *http.Request) netip.AddrPort {
	ap, _ := netip.ParseAddrPort(r.RemoteAddr)
	return netip.AddrPortFrom(ap.Addr().Unmap(), ap.Port())
}

func newSiteHandler(s Site, log *logbuf.Logger, auth *authenticator, waf WAFProvider) (*siteHandler, error) {
	h := &siteHandler{site: s, stats: &SiteStats{}, log: log, auth: auth}
	var err error
	if h.ipList, err = parsePrefixes(s.IPList); err != nil {
		return nil, err
	}
	switch s.Type {
	case TypeProxy:
		for _, t := range s.Targets {
			u, err := url.Parse(t)
			if err != nil {
				return nil, err
			}
			h.targets = append(h.targets, u)
		}
		h.handler = h.newProxy()
	case TypeRedirect:
		h.handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			to := strings.TrimRight(s.RedirectTo, "/")
			if s.KeepPath {
				to += r.URL.RequestURI()
			} else {
				to = s.RedirectTo
			}
			http.Redirect(w, r, to, s.RedirectCode)
		})
	case TypeStatic:
		fs := http.FileServer(http.Dir(s.Root))
		h.handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 关闭目录列表时，目录请求只允许存在 index.html
			if !s.ListDir && strings.HasSuffix(r.URL.Path, "/") {
				if _, err := http.Dir(s.Root).Open(r.URL.Path + "index.html"); err != nil {
					http.NotFound(w, r)
					return
				}
			}
			fs.ServeHTTP(w, r)
		})
	case TypeText:
		h.handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", s.ContentType)
			w.WriteHeader(s.StatusCode)
			_, _ = w.Write([]byte(s.Text))
		})
	}
	h.pipeline = http.HandlerFunc(h.serveInner)
	// WAF 放在认证之前，登录页同样受保护
	if s.WAF != "" && waf != nil {
		h.pipeline = waf.Middleware(s.WAF, h.pipeline)
	}
	return h, nil
}

// pickTarget 轮询选择后端。
func (h *siteHandler) pickTarget() *url.URL {
	n := h.next.Add(1) - 1
	return h.targets[int(n%uint64(len(h.targets)))]
}

func (h *siteHandler) newProxy() http.Handler {
	s := h.site
	dialer := &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}
	tr := &http.Transport{
		Proxy:                 nil,
		TLSClientConfig:       &tls.Config{InsecureSkipVerify: s.InsecureSkipVerify},
		ForceAttemptHTTP2:     !s.SendProxyProtocol,
		MaxIdleConnsPerHost:   32,
		IdleConnTimeout:       90 * time.Second,
		ResponseHeaderTimeout: 0,
		// 发送 PROXY 头时每个连接只对应一个客户端，必须禁用连接复用
		DisableKeepAlives: s.DisableKeepAlives || s.SendProxyProtocol,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			c, err := dialer.DialContext(ctx, network, addr)
			if err != nil || !s.SendProxyProtocol {
				return c, err
			}
			src, _ := ctx.Value(ctxKeyClientAddr{}).(netip.AddrPort)
			if !src.IsValid() {
				return c, nil
			}
			header := proxyproto.HeaderProxyFromAddrs(2, net.TCPAddrFromAddrPort(src), c.LocalAddr())
			if _, err := header.WriteTo(c); err != nil {
				c.Close()
				return nil, err
			}
			return c, nil
		},
	}
	rp := &httputil.ReverseProxy{
		Transport: tr,
		Rewrite: func(pr *httputil.ProxyRequest) {
			target := h.pickTarget()
			pr.SetURL(target)
			if !s.UseTargetHost {
				// SetURL 会改写 Host，这里恢复客户端请求的 Host
				pr.Out.Host = pr.In.Host
			}
			if s.ForwardedHeaders {
				pr.SetXForwarded()
			}
			applyHeaders(pr.Out.Header, s.RequestHeaders)
		},
		ModifyResponse: func(resp *http.Response) error {
			applyHeaders(resp.Header, s.ResponseHeaders)
			return nil
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			h.stats.Errors.Add(1)
			h.log.Warnf("%s %s%s 后端错误: %v", r.Method, r.Host, r.URL.Path, err)
			http.Error(w, "502 Bad Gateway", http.StatusBadGateway)
		},
		// 流式响应（SSE 等）及时刷新
		FlushInterval: -1,
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), ctxKeyClientAddr{}, clientAddr(r))
		rp.ServeHTTP(w, r.WithContext(ctx))
	})
}

// applyHeaders 按 "Key: Value" 设置头，值为空时删除。
func applyHeaders(h http.Header, list []string) {
	for _, line := range list {
		k, v, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		k, v = strings.TrimSpace(k), strings.TrimSpace(v)
		if v == "" {
			h.Del(k)
		} else {
			h.Set(k, v)
		}
	}
}

// allowedIP 按访问控制模式判断客户端是否允许访问。
func (h *siteHandler) allowedIP(a netip.Addr) bool {
	if h.site.IPMode == IPModeOff {
		return true
	}
	in := false
	for _, p := range h.ipList {
		if p.Contains(a) {
			in = true
			break
		}
	}
	if h.site.IPMode == IPModeWhitelist {
		return in
	}
	return !in
}

// statusRecorder 记录响应状态码，用于访问日志。
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

// Unwrap 让 http.ResponseController 能访问底层连接（WebSocket 升级等需要）。
func (s *statusRecorder) Unwrap() http.ResponseWriter {
	return s.ResponseWriter
}

func (h *siteHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.stats.Requests.Add(1)
	h.stats.Active.Add(1)
	defer h.stats.Active.Add(-1)
	start := time.Now()
	client := clientAddr(r)
	rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

	defer func() {
		if h.site.AccessLog {
			h.log.Infof("%s %s %s%s %d %dms", client.Addr(), r.Method, r.Host, r.URL.RequestURI(), rec.status, time.Since(start).Milliseconds())
		}
	}()

	if !h.allowedIP(client.Addr()) {
		h.log.Warnf("拒绝 %s 访问 %s（访问控制）", client.Addr(), r.Host)
		http.Error(rec, "403 Forbidden", http.StatusForbidden)
		return
	}
	h.pipeline.ServeHTTP(rec, r)
}

// serveInner 处理跨域、认证并交给站点处理器。
func (h *siteHandler) serveInner(rec http.ResponseWriter, r *http.Request) {
	if h.site.CORS {
		if origin := r.Header.Get("Origin"); origin != "" {
			rec.Header().Set("Access-Control-Allow-Origin", origin)
			rec.Header().Set("Access-Control-Allow-Credentials", "true")
			rec.Header().Set("Vary", "Origin")
			if r.Method == http.MethodOptions {
				rec.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
				if req := r.Header.Get("Access-Control-Request-Headers"); req != "" {
					rec.Header().Set("Access-Control-Allow-Headers", req)
				}
				rec.WriteHeader(http.StatusNoContent)
				return
			}
		}
	}
	if (h.site.BasicAuth || h.site.WebAuth) && !h.auth.check(rec, r, h.site) {
		return
	}
	h.handler.ServeHTTP(rec, r)
}

// snapshot 返回统计快照。
func (s *SiteStats) snapshot() map[string]int64 {
	return map[string]int64{"requests": s.Requests.Load(), "active": s.Active.Load(), "errors": s.Errors.Load()}
}

func (h *siteHandler) String() string {
	return fmt.Sprintf("%s %v", h.site.Type, h.site.Domains)
}
