package webservice

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"

	proxyproto "github.com/pires/go-proxyproto"
	"github.com/quic-go/quic-go/http3"

	"github.com/wangwanjie/gately/internal/acmehttp"
	"github.com/wangwanjie/gately/internal/logbuf"
)

// CertSource 提供按 SNI 选择证书的能力（由证书模块实现）。
type CertSource interface {
	GetCertificate(*tls.ClientHelloInfo) (*tls.Certificate, error)
}

var tlsVersions = map[string]uint16{
	"1.0": tls.VersionTLS10, "1.1": tls.VersionTLS11, "1.2": tls.VersionTLS12, "1.3": tls.VersionTLS13,
}

// listenerRunner 运行一个监听器。
type listenerRunner struct {
	cfg  Listener
	auth *authenticator
	log  *logbuf.Logger
	srv  *http.Server
	// h3 为 HTTP/3（QUIC）服务，仅在监听器开启 HTTP3 时存在
	h3 *http3.Server
	// altSvc 为通告 HTTP/3 的响应头值，未开启时为空
	altSvc string
	// cancel 结束所有请求上下文，用于关闭已升级（WebSocket）的连接
	cancel  context.CancelFunc
	sites   []*siteHandler
	deflt   *siteHandler
	startAt time.Time
	err     string
}

// route 按 Host 选择站点：精确匹配优先，其次最长通配，最后 *。
func (l *listenerRunner) route(host string) *siteHandler {
	host = hostOnly(host)
	var best *siteHandler
	bestScore, bestLen := 0, 0
	for _, s := range l.sites {
		for _, d := range s.site.Domains {
			score := matchDomain(d, host)
			if score == 0 {
				continue
			}
			if score > bestScore || (score == bestScore && len(d) > bestLen) {
				best, bestScore, bestLen = s, score, len(d)
			}
		}
	}
	if best != nil {
		return best
	}
	return l.deflt
}

func (l *listenerRunner) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 通过 TCP 访问时告知浏览器可改用 HTTP/3；HTTP/3 请求本身无需再通告
	if l.altSvc != "" && r.ProtoMajor < 3 {
		w.Header().Set("Alt-Svc", l.altSvc)
	}
	// ACME HTTP-01 验证请求优先应答（仅在签发进行中登记过 token 时命中）
	if r.TLS == nil && acmehttp.Serve(w, r) {
		return
	}
	// 第三方登录回调只在配置的认证域名上处理，与站点路由无关
	if r.URL.Path == oauthCallback && l.auth != nil {
		env := l.auth.oauthEnv()
		if u, err := url.Parse(env.callbackBase); err == nil && env.callbackBase != "" && hostOnly(u.Host) == hostOnly(r.Host) {
			l.auth.handleOAuthCallback(w, r, env)
			return
		}
	}
	s := l.route(r.Host)
	if s == nil {
		http.NotFound(w, r)
		return
	}
	s.ServeHTTP(w, r)
}

func newListenerRunner(cfg Listener, log *logbuf.Logger, auth *authenticator, waf WAFProvider) (*listenerRunner, error) {
	l := &listenerRunner{cfg: cfg, log: log, auth: auth}
	for _, s := range cfg.Sites {
		if !s.Enable {
			continue
		}
		s.scope = cfg.Key + "/" + s.Key
		h, err := newSiteHandler(s, log, auth, waf)
		if err != nil {
			return nil, err
		}
		l.sites = append(l.sites, h)
	}
	if cfg.Default.Enable {
		d := cfg.Default
		d.scope = cfg.Key + "/" + d.Key
		h, err := newSiteHandler(d, log, auth, waf)
		if err != nil {
			return nil, err
		}
		l.deflt = h
	}
	return l, nil
}

// start 开始监听。trusted 为可信的 PROXY 协议来源。
func (l *listenerRunner) start(certs CertSource, trusted []string) error {
	addr := net.JoinHostPort(l.cfg.ListenIP, strconv.Itoa(l.cfg.Port))
	ln, err := net.Listen(l.cfg.Network, addr)
	if err != nil {
		return err
	}
	if l.cfg.AcceptProxyProtocol {
		if len(trusted) == 0 {
			ln.Close()
			return errors.New("已开启接收 PROXY 协议，但未配置可信来源")
		}
		// 只有可信来源发来的 PROXY 头才会被采用，其他来源的头被忽略
		policy, err := proxyproto.ConnLaxWhiteListPolicy(trusted)
		if err != nil {
			ln.Close()
			return err
		}
		ln = &proxyproto.Listener{Listener: ln, ConnPolicy: policy, ReadHeaderTimeout: 5 * time.Second}
	}
	baseCtx, cancel := context.WithCancel(context.Background())
	l.cancel = cancel
	l.srv = &http.Server{
		Handler:           l,
		BaseContext:       func(net.Listener) context.Context { return baseCtx },
		ReadHeaderTimeout: 30 * time.Second,
		IdleTimeout:       120 * time.Second,
		// net/http 内部错误（TLS 握手失败、缺少证书等）写入本监听器的运行日志
		ErrorLog: l.log.ServerErrorLog(),
	}
	if l.cfg.TLS {
		if certs == nil {
			ln.Close()
			return errors.New("证书模块不可用")
		}
		min := tlsVersions[l.cfg.MinTLS]
		if min == 0 {
			min = tls.VersionTLS12
		}
		l.srv.TLSConfig = &tls.Config{GetCertificate: certs.GetCertificate, MinVersion: min, NextProtos: []string{"h2", "http/1.1"}}
		ln = tls.NewListener(ln, l.srv.TLSConfig)
		if l.cfg.HTTP3 {
			if err := l.startHTTP3(addr); err != nil {
				// HTTP/3 失败不影响 HTTP/1.1 与 HTTP/2 服务
				l.log.Errorf("HTTP/3 监听 %s 失败: %v", addr, err)
			}
		}
	}
	l.startAt = time.Now()
	for _, s := range l.sites {
		s.startBackground(baseCtx)
	}
	if l.deflt != nil {
		l.deflt.startBackground(baseCtx)
	}
	go func() {
		if err := l.srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			l.log.Errorf("监听 %s 退出: %v", addr, err)
		}
	}()
	scheme := "http"
	if l.cfg.TLS {
		scheme = "https"
	}
	l.log.Infof("%s 监听 %s（%d 个站点）", scheme, addr, len(l.sites))
	return nil
}

// startHTTP3 在与 TCP 相同的地址与端口号上启动 UDP（QUIC）监听。
func (l *listenerRunner) startHTTP3(addr string) error {
	network := map[string]string{"tcp": "udp", "tcp4": "udp4", "tcp6": "udp6"}[l.cfg.Network]
	pc, err := net.ListenPacket(network, addr)
	if err != nil {
		return err
	}
	l.h3 = &http3.Server{
		Handler:     l,
		TLSConfig:   http3.ConfigureTLSConfig(l.srv.TLSConfig),
		IdleTimeout: 120 * time.Second,
	}
	l.altSvc = fmt.Sprintf(`h3=":%d"; ma=86400`, l.cfg.Port)
	go func() {
		if err := l.h3.Serve(pc); err != nil && !errors.Is(err, http.ErrServerClosed) {
			l.log.Errorf("HTTP/3 监听 %s 退出: %v", addr, err)
		}
	}()
	l.log.Infof("HTTP/3 监听 %s（UDP）", addr)
	return nil
}

func (l *listenerRunner) stop() {
	if l.srv == nil {
		return
	}
	if l.h3 != nil {
		_ = l.h3.Close()
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := l.srv.Shutdown(ctx); err != nil {
		_ = l.srv.Close()
	}
	// Shutdown 不管理已劫持的连接；取消请求上下文后反向代理会关闭升级后的连接
	l.cancel()
}
