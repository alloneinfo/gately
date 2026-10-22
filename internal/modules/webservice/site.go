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
	"sync"
	"sync/atomic"
	"time"

	proxyproto "github.com/pires/go-proxyproto"

	"github.com/wangwanjie/gately/internal/geoip"
	"github.com/wangwanjie/gately/internal/iplist"
	"github.com/wangwanjie/gately/internal/logbuf"
	"github.com/wangwanjie/gately/internal/webpage"
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
	pool     *pool
	// locations 为按路径前缀分流的子后端（最长前缀在前）
	locations []*locationHandler
	// fwd 为外部转发认证，未启用时为 nil
	fwd *forwardAuther
	// idleClosers 关闭各反向代理的空闲后端连接，站点被替换时调用
	idleClosers []func()
	limiter     *limiter
	cache       *respCache
	rewrites    []compiledRewrite
	metrics     *siteMetrics
	// rules 为预编译的路径规则（最长前缀在前），countries 为国家过滤集合
	rules     []compiledRule
	countries map[string]bool
}

type ctxKeyClientAddr struct{}

// clientAddr 返回请求的客户端地址（PROXY 协议处理后的真实地址）。
func clientAddr(r *http.Request) netip.AddrPort {
	ap, _ := netip.ParseAddrPort(r.RemoteAddr)
	return netip.AddrPortFrom(ap.Addr().Unmap(), ap.Port())
}

func newSiteHandler(s Site, log *logbuf.Logger, auth *authenticator, waf WAFProvider) (*siteHandler, error) {
	h := &siteHandler{site: s, stats: &SiteStats{}, log: log, auth: auth, metrics: newSiteMetrics()}
	var err error
	if h.rewrites, err = compileRewrites(s.Rewrites); err != nil {
		return nil, err
	}
	if s.RateLimit.Enable {
		h.limiter = newLimiter(s.RateLimit)
	}
	if h.ipList, err = parsePrefixes(s.IPList); err != nil {
		return nil, err
	}
	if h.rules, err = compileRules(s.Rules); err != nil {
		return nil, err
	}
	h.countries = map[string]bool{}
	for _, c := range s.Countries {
		h.countries[strings.ToUpper(c)] = true
	}
	if s.ForwardAuth.URL != "" {
		h.fwd = newForwardAuther(s.ForwardAuth)
	}
	switch s.Type {
	case TypeProxy:
		if h.pool, err = newPool(s, log); err != nil {
			return nil, err
		}
		h.handler = h.newProxy(s, h.pool)
		if len(s.Locations) > 0 {
			if err := h.buildLocations(); err != nil {
				return nil, err
			}
			h.handler = h.dispatchLocations(h.handler)
		}
		if s.Cache.Enable {
			h.cache = newRespCache(s.Cache)
			h.handler = h.cache.wrap(h.handler)
		}
	case TypeRedirect:
		h.handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// {host} 替换为访客请求的域名（不含端口），用于把 HTTP 统一跳到同域名的 HTTPS
			base := strings.ReplaceAll(s.RedirectTo, hostPlaceholder, hostOnly(r.Host))
			to := base
			if s.KeepPath {
				to = strings.TrimRight(base, "/") + r.URL.RequestURI()
			}
			applyHeaders(w.Header(), s.ResponseHeaders)
			http.Redirect(w, r, to, s.RedirectCode)
		})
	case TypeStatic:
		h.handler = h.newStatic(s)
	case TypeText:
		h.handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			applyHeaders(w.Header(), s.ResponseHeaders)
			w.Header().Set("Content-Type", s.ContentType)
			w.WriteHeader(s.StatusCode)
			_, _ = w.Write([]byte(s.Text))
		})
	}
	if s.Compress && s.Type != TypeRedirect {
		h.handler = compressHandler(h.handler)
	}
	h.pipeline = http.HandlerFunc(h.serveInner)
	// WAF 放在认证之前，登录页同样受保护
	if s.WAF != "" && waf != nil {
		h.pipeline = waf.Middleware(s.WAF, h.pipeline)
	}
	return h, nil
}

type ctxKeyUpstream struct{}

// upstreamStatus 返回后端状态，非代理站点为空。
func (h *siteHandler) upstreamStatus() []UpstreamStatus {
	if h.pool == nil {
		return nil
	}
	return h.pool.status()
}

// startBackground 启动站点的后台任务（主动健康检查）。
func (h *siteHandler) startBackground(ctx context.Context) {
	if h.pool != nil {
		h.pool.startChecks(ctx)
	}
	for _, l := range h.locations {
		if l.pool != nil {
			l.pool.startChecks(ctx)
		}
	}
}

// closeIdle 关闭站点各反向代理的空闲后端连接。
func (h *siteHandler) closeIdle() {
	for _, f := range h.idleClosers {
		f()
	}
}

// inherit 在热更新站点时沿用旧处理器的统计数据，避免修改配置后计数与分钟曲线清零。
func (h *siteHandler) inherit(old *siteHandler) {
	h.stats = old.stats
	h.metrics = old.metrics
}

// proxyBufferPool 复用反向代理的拷贝缓冲区，避免每个请求分配 32 KB 带来的 GC 压力。
type proxyBufferPool struct{ p sync.Pool }

func (b *proxyBufferPool) Get() []byte {
	if v, ok := b.p.Get().(*[]byte); ok {
		return *v
	}
	return make([]byte, 32*1024)
}

func (b *proxyBufferPool) Put(buf []byte) {
	if cap(buf) < 32*1024 {
		return
	}
	buf = buf[:32*1024]
	b.p.Put(&buf)
}

var sharedBufferPool = &proxyBufferPool{}

// sharedSessionCache 缓存到 HTTPS 后端的 TLS 会话，新连接可以复用会话，省去完整握手。
var sharedSessionCache = tls.NewLRUClientSessionCache(512)

// newTransport 为一个后端创建连接池。每个后端独立一个 Transport：
// 开启“使用访问域名作为 SNI”时请求 URL 的主机名会换成访问域名，独立的连接池保证连接不会被串用到别的后端。
func newTransport(s Site, up *upstream) *http.Transport {
	dialer := &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}
	// 实际拨号地址始终是后端配置的地址，与请求 URL 中的主机名无关
	target := up.url.Host
	if up.url.Port() == "" {
		port := "80"
		if up.url.Scheme == "https" {
			port = "443"
		}
		target = net.JoinHostPort(up.url.Hostname(), port)
	}
	return &http.Transport{
		Proxy:             nil,
		TLSClientConfig:   &tls.Config{InsecureSkipVerify: s.InsecureSkipVerify, ClientSessionCache: sharedSessionCache},
		ForceAttemptHTTP2: !s.SendProxyProtocol,
		// 本机后端的并发可能很高，空闲连接上限放宽到与 nginx keepalive 常用值相当，减少突发时的重复建连
		MaxIdleConnsPerHost:   256,
		IdleConnTimeout:       90 * time.Second,
		ResponseHeaderTimeout: 0,
		// 发送 PROXY 头时每个连接只对应一个客户端，必须禁用连接复用
		DisableKeepAlives: s.DisableKeepAlives || s.SendProxyProtocol,
		DialContext: func(ctx context.Context, network, _ string) (net.Conn, error) {
			c, err := dialer.DialContext(ctx, network, target)
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
}

// upstreamRoundTripper 按请求选中的后端使用对应的连接池。
type upstreamRoundTripper map[*upstream]*http.Transport

func (rt upstreamRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	up, _ := req.Context().Value(ctxKeyUpstream{}).(*upstream)
	return rt[up].RoundTrip(req)
}

// newProxy 创建反向代理处理器，s 为生效的站点配置（分路径规则会传入合并后的配置），p 为其后端集合。
func (h *siteHandler) newProxy(s Site, p *pool) http.Handler {
	rt := upstreamRoundTripper{}
	for _, up := range p.ups {
		tr := newTransport(s, up)
		rt[up] = tr
		h.idleClosers = append(h.idleClosers, tr.CloseIdleConnections)
	}
	rp := &httputil.ReverseProxy{
		Transport:  rt,
		BufferPool: sharedBufferPool,
		Rewrite: func(pr *httputil.ProxyRequest) {
			up, _ := pr.In.Context().Value(ctxKeyUpstream{}).(*upstream)
			pr.SetURL(up.url)
			if !s.UseTargetHost {
				// SetURL 会改写 Host，这里恢复客户端请求的 Host
				pr.Out.Host = pr.In.Host
			}
			if s.UpstreamSNIFromHost && up.url.Scheme == "https" {
				// 把 URL 主机名换成访问域名，TLS 握手的 SNI 与证书校验都使用它；拨号地址仍是后端地址（见 newTransport）
				if host := hostOnly(pr.In.Host); host != "" && net.ParseIP(host) == nil {
					port := up.url.Port()
					if port == "" {
						port = "443"
					}
					pr.Out.URL.Host = net.JoinHostPort(host, port)
				}
			}
			// 客户端自带的 X-Real-IP 不可信，只在开启转发头时由 Gately 写入
			pr.Out.Header.Del("X-Real-Ip")
			if s.ForwardedHeaders {
				pr.SetXForwarded()
				pr.Out.Header.Set("X-Real-Ip", clientAddr(pr.In).Addr().String())
				if port := localPort(pr.In); port != "" {
					pr.Out.Header.Set("X-Forwarded-Port", port)
				}
			}
			// 站点会话 Cookie 只用于 Gately 自身认证，不转发给后端
			stripCookie(pr.Out, cookieName(s), oauthNonceCookie, ssoCookie)
			applyHeaders(pr.Out.Header, s.RequestHeaders)
		},
		ModifyResponse: func(resp *http.Response) error {
			applyHeaders(resp.Header, s.ResponseHeaders)
			if up, _ := resp.Request.Context().Value(ctxKeyUpstream{}).(*upstream); up != nil {
				p.ok(up)
			}
			return nil
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			h.stats.Errors.Add(1)
			// 客户端中途断开不是后端的错
			if up, _ := r.Context().Value(ctxKeyUpstream{}).(*upstream); up != nil && r.Context().Err() == nil {
				p.fail(up, err)
			}
			h.log.Warnf("%s %s%s 后端错误: %v", r.Method, r.Host, r.URL.Path, err)
			writeErrorPage(w, r, http.StatusBadGateway, "无法连接到后端服务", "站点的后端服务没有响应，可能未启动或网络不可达。请稍后重试。")
		},
		// 流式响应（SSE 等）及时刷新
		FlushInterval: -1,
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		client := clientAddr(r)
		up := p.pick(client.Addr())
		up.conns.Add(1)
		defer up.conns.Add(-1)
		ctx := context.WithValue(r.Context(), ctxKeyClientAddr{}, client)
		ctx = context.WithValue(ctx, ctxKeyUpstream{}, up)
		rp.ServeHTTP(w, r.WithContext(ctx))
	})
}

// localPort 返回请求到达的本地端口。经 PROXY 协议转入时为原始目的端口。
func localPort(r *http.Request) string {
	if a, ok := r.Context().Value(http.LocalAddrContextKey).(net.Addr); ok {
		if _, port, err := net.SplitHostPort(a.String()); err == nil {
			return port
		}
	}
	return ""
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
// 命中的路径规则设置了 IP 模式时，以规则的名单为准。
// 引用的订阅 IP 列表尚未下载成功时视为空列表：白名单因此拒绝访问，黑名单不额外拦截。
func (h *siteHandler) allowedIP(a netip.Addr, rule *compiledRule) bool {
	mode, list, sets := h.site.IPMode, h.ipList, h.site.IPSets
	if rule != nil && rule.ipMode != IPModeOff {
		mode, list, sets = rule.ipMode, rule.ipList, rule.ipSets
	}
	if mode == IPModeOff {
		return true
	}
	in := false
	for _, p := range list {
		if p.Contains(a) {
			in = true
			break
		}
	}
	for _, key := range sets {
		if in {
			break
		}
		if set, ok := iplist.Default.Get(key); ok {
			in = set.Contains(a)
		}
	}
	if mode == IPModeWhitelist {
		return in
	}
	return !in
}

// statusRecorder 记录响应状态码，用于访问日志。
type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int64
}

func (s *statusRecorder) Write(p []byte) (int, error) {
	n, err := s.ResponseWriter.Write(p)
	s.bytes += int64(n)
	return n, err
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
		h.metrics.record(client.Addr(), rec.status, rec.bytes, time.Now())
		if h.site.AccessLog {
			uri := r.URL.RequestURI()
			// 认证路径的查询串含票据、授权码等敏感信息，不写入日志
			if strings.HasPrefix(r.URL.Path, authPath) {
				uri = r.URL.Path
			}
			h.log.Infof("%s %s %s%s %d %dms", client.Addr(), r.Method, r.Host, uri, rec.status, time.Since(start).Milliseconds())
		}
	}()

	if h.auth.banned(client.Addr()) {
		http.Error(rec, "403 Forbidden", http.StatusForbidden)
		return
	}
	if !h.allowedIP(client.Addr(), h.matchRule(r.URL.Path)) {
		h.log.Warnf("拒绝 %s 访问 %s（访问控制）", client.Addr(), r.Host)
		http.Error(rec, "403 Forbidden", http.StatusForbidden)
		return
	}
	if !h.countryAllowed(client.Addr()) {
		h.log.Warnf("拒绝 %s 访问 %s（国家过滤：%s）", client.Addr(), r.Host, geoip.Default.Country(client.Addr()))
		http.Error(rec, "403 Forbidden", http.StatusForbidden)
		return
	}
	if h.limiter != nil {
		release, retry, ok := h.limiter.acquire(limitKeyAddr(client.Addr()))
		if !ok {
			rec.Header().Set("Retry-After", retryAfterSeconds(retry))
			http.Error(rec, "429 Too Many Requests", http.StatusTooManyRequests)
			return
		}
		defer release()
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
	// 登录页等认证保留路径始终交给认证器处理，其余路径按规则决定是否需要认证
	if h.site.WebAuth && strings.HasPrefix(r.URL.Path, authPath) {
		h.auth.check(rec, r, h.site)
		return
	}
	if h.fwd != nil {
		// 无论该路径是否需要认证，都不允许访客自带身份头
		h.fwd.stripIdentity(r)
	}
	// 跳转类重写不返回任何内容，先于认证执行，与 nginx 中 location = / { return 302 ...; } 不经认证的行为一致
	if len(h.rewrites) > 0 && h.redirectBeforeAuth(rec, r) {
		return
	}
	if h.needAuth(h.matchRule(r.URL.Path)) {
		handled, allowed := h.tryShare(rec, r)
		if handled {
			return
		}
		if !allowed {
			if h.fwd != nil {
				if !h.fwd.check(rec, r, h) {
					return
				}
			} else if !h.auth.check(rec, r, h.site) {
				return
			}
		}
	}
	if len(h.rewrites) > 0 && h.applyRewrite(rec, r) {
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

// writeErrorPage 浏览器访问时输出样式化的错误页，其他客户端输出纯文本。
func writeErrorPage(w http.ResponseWriter, r *http.Request, code int, title, detail string) {
	webpage.Error(w, r, webpage.Problem{Code: code, Title: title, Detail: detail})
}

// writeNotFound 输出 404：浏览器看到错误页，其他客户端与 http.NotFound 一致。
func writeNotFound(w http.ResponseWriter, r *http.Request, title, detail string) {
	if !webpage.WantsHTML(r) {
		http.NotFound(w, r)
		return
	}
	webpage.Error(w, r, webpage.Problem{Code: http.StatusNotFound, Title: title, Detail: detail, Home: r.URL.Path != "/"})
}
