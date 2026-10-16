package webservice

import (
	"context"
	"crypto/tls"
	"fmt"
	"html/template"
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
				f, err := http.Dir(s.Root).Open(r.URL.Path + "index.html")
				if err != nil {
					http.NotFound(w, r)
					return
				}
				f.Close()
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
			// 客户端自带的 X-Real-IP 不可信，只在开启转发头时由 Gately 写入
			pr.Out.Header.Del("X-Real-Ip")
			if s.ForwardedHeaders {
				pr.SetXForwarded()
				pr.Out.Header.Set("X-Real-Ip", clientAddr(pr.In).Addr().String())
			}
			// 站点会话 Cookie 只用于 Gately 自身认证，不转发给后端
			stripCookie(pr.Out, cookieName(s), oauthNonceCookie)
			applyHeaders(pr.Out.Header, s.RequestHeaders)
		},
		ModifyResponse: func(resp *http.Response) error {
			applyHeaders(resp.Header, s.ResponseHeaders)
			return nil
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			h.stats.Errors.Add(1)
			h.log.Warnf("%s %s%s 后端错误: %v", r.Method, r.Host, r.URL.Path, err)
			writeErrorPage(w, r, http.StatusBadGateway, "无法连接到后端服务", "站点的后端服务没有响应，可能未启动或网络不可达。请稍后重试。")
		},
		// 流式响应（SSE 等）及时刷新
		FlushInterval: -1,
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), ctxKeyClientAddr{}, clientAddr(r))
		rp.ServeHTTP(w, r.WithContext(ctx))
	})
}

// stripCookie 从请求中删除指定名称的 Cookie。
func stripCookie(r *http.Request, names ...string) {
	cookies := r.Cookies()
	if len(cookies) == 0 {
		return
	}
	var kept []string
	for _, c := range cookies {
		drop := false
		for _, n := range names {
			if c.Name == n {
				drop = true
				break
			}
		}
		if !drop {
			kept = append(kept, c.Name+"="+c.Value)
		}
	}
	if len(kept) == len(cookies) {
		return
	}
	r.Header.Del("Cookie")
	if len(kept) > 0 {
		r.Header.Set("Cookie", strings.Join(kept, "; "))
	}
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
			uri := r.URL.RequestURI()
			// 认证路径的查询串含票据、授权码等敏感信息，不写入日志
			if strings.HasPrefix(r.URL.Path, authPath) {
				uri = r.URL.Path
			}
			h.log.Infof("%s %s %s%s %d %dms", client.Addr(), r.Method, r.Host, uri, rec.status, time.Since(start).Milliseconds())
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
		if origin := r.Header.Get("Origin"); origin != "" && h.corsAllowed(origin) {
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

// corsAllowed 判断是否向该来源开放带凭据的跨域访问。
// 站点开启认证时只允许站点自身域名作为来源，否则任意网页都能借用户的登录态读取内容。
func (h *siteHandler) corsAllowed(origin string) bool {
	if !h.site.BasicAuth && !h.site.WebAuth {
		return true
	}
	u, err := url.Parse(origin)
	if err != nil || u.Host == "" {
		return false
	}
	host := hostOnly(u.Host)
	for _, d := range h.site.Domains {
		if d != "*" && matchDomain(d, host) > 0 {
			return true
		}
	}
	return false
}

// snapshot 返回统计快照。
func (s *SiteStats) snapshot() map[string]int64 {
	return map[string]int64{"requests": s.Requests.Load(), "active": s.Active.Load(), "errors": s.Errors.Load()}
}

func (h *siteHandler) String() string {
	return fmt.Sprintf("%s %v", h.site.Type, h.site.Domains)
}

var errorPageTmpl = template.Must(template.New("err").Parse(`<!doctype html>
<html lang="zh-CN"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>{{.Code}} {{.Title}}</title>
<style>
body{margin:0;min-height:100vh;display:flex;align-items:center;justify-content:center;background:#f2f5f4;color:#333;font-family:-apple-system,BlinkMacSystemFont,"PingFang SC","Microsoft YaHei",sans-serif}
.box{background:#fff;padding:32px 36px;border-radius:12px;box-shadow:0 2px 16px rgba(0,0,0,.06);max-width:420px;margin:16px}
.code{font-size:44px;font-weight:600;color:#0e7c66;margin:0}h1{font-size:18px;margin:8px 0}p{color:#777;font-size:14px;line-height:1.6;margin:0}
.host{margin-top:16px;font-size:12px;color:#aaa;word-break:break-all}
@media (prefers-color-scheme:dark){body{background:#141817;color:#ddd}.box{background:#1d2221;box-shadow:none}p{color:#999}}
</style></head><body><div class="box"><p class="code">{{.Code}}</p><h1>{{.Title}}</h1><p>{{.Detail}}</p><div class="host">{{.Host}} · Gately</div></div></body></html>`))

// writeErrorPage 浏览器访问时输出简洁的错误页，其他客户端输出纯文本。
func writeErrorPage(w http.ResponseWriter, r *http.Request, code int, title, detail string) {
	if !strings.Contains(r.Header.Get("Accept"), "text/html") {
		http.Error(w, fmt.Sprintf("%d %s", code, http.StatusText(code)), code)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(code)
	_ = errorPageTmpl.Execute(w, map[string]any{"Code": code, "Title": title, "Detail": detail, "Host": r.Host})
}
