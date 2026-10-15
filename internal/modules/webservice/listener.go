package webservice

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"

	proxyproto "github.com/pires/go-proxyproto"

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
	cfg     Listener
	auth    *authenticator
	log     *logbuf.Logger
	srv     *http.Server
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
		h, err := newSiteHandler(s, log, auth, waf)
		if err != nil {
			return nil, err
		}
		l.sites = append(l.sites, h)
	}
	if cfg.Default.Enable {
		h, err := newSiteHandler(cfg.Default, log, auth, waf)
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
		// 只有可信来源发来的 PROXY 头才会被采用，其他来源的头被忽略
		policy, err := proxyproto.ConnLaxWhiteListPolicy(trusted)
		if err != nil {
			ln.Close()
			return err
		}
		ln = &proxyproto.Listener{Listener: ln, ConnPolicy: policy, ReadHeaderTimeout: 5 * time.Second}
	}
	l.srv = &http.Server{
		Handler:           l,
		ReadHeaderTimeout: 30 * time.Second,
		IdleTimeout:       120 * time.Second,
		ErrorLog:          nil,
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
	}
	l.startAt = time.Now()
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

func (l *listenerRunner) stop() {
	if l.srv == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := l.srv.Shutdown(ctx); err != nil {
		_ = l.srv.Close()
	}
}
