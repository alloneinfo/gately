// Package webservice 实现 Web 服务模块：按域名把请求分发到反向代理、重定向、静态文件或固定文本，
// 支持 HTTPS（证书来自证书模块）、PROXY 协议、基本认证、网页认证与 IP 访问控制。
package webservice

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/wangwanjie/gately/internal/config"
	"github.com/wangwanjie/gately/internal/crud"
	"github.com/wangwanjie/gately/internal/geoip"
	"github.com/wangwanjie/gately/internal/httpx"
	"github.com/wangwanjie/gately/internal/logbuf"
	"github.com/wangwanjie/gately/internal/module"
)

// ID 是模块标识。
const ID = "webservice"

// Module 是 Web 服务模块。
type Module struct {
	mu      sync.Mutex
	store   *config.Store
	logs    *logbuf.Buffer
	cfg     fileConfig
	ctx     context.Context
	certs   CertSource
	waf     WAFProvider
	auth    *authenticator
	runners map[string]*listenerRunner
	errors  map[string]string
}

// New 创建模块。certs 为证书来源，可为 nil（此时不能启用 HTTPS）；waf 可为 nil（站点的 WAF 设置不生效）。
func New(certs CertSource, waf WAFProvider) *Module {
	return &Module{certs: certs, waf: waf, runners: map[string]*listenerRunner{}, errors: map[string]string{}}
}

// ID 实现 module.Module。
func (m *Module) ID() string { return ID }

// Title 实现 module.Module。
func (m *Module) Title() string { return "Web 服务" }

// Init 实现 module.Module。
func (m *Module) Init(env module.Env) error {
	m.store = env.Store
	m.logs = env.Logs
	m.cfg = fileConfig{Settings: Settings{Enable: true}}
	if _, err := m.store.Load(ID, &m.cfg); err != nil {
		return err
	}
	if m.cfg.Settings.AuthSecret == "" {
		b := make([]byte, 32)
		_, _ = rand.Read(b)
		m.cfg.Settings.AuthSecret = hex.EncodeToString(b)
		if err := m.saveLocked(); err != nil {
			return err
		}
	}
	m.auth = newAuthenticator(m.cfg.Settings.AuthSecret)
	m.auth.setBanSeconds(m.cfg.Settings.AutoBanSeconds)
	m.applyOAuthLocked()
	// 已导入过 GeoIP 数据库时随模块加载，失败只记录日志
	if _, err := os.Stat(m.geoipPath()); err == nil {
		if err := geoip.Default.Load(m.geoipPath()); err != nil {
			m.logs.Add(logbuf.LevelWarn, "", err.Error())
		}
	}
	return nil
}

// applyOAuthLocked 把第三方登录设置同步给认证器。
func (m *Module) applyOAuthLocked() {
	env := oauthEnv{providers: map[string]OAuthProvider{}, callbackBase: strings.TrimRight(m.cfg.Settings.OAuthCallbackBase, "/")}
	for _, p := range m.cfg.Settings.OAuthProviders {
		env.providers[p.Key] = p
	}
	m.auth.setOAuth(env)
}

// Start 实现 module.Module。
func (m *Module) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ctx = ctx
	if !m.cfg.Settings.Enable {
		m.logs.Add(logbuf.LevelInfo, "", "Web 服务模块已禁用")
		return nil
	}
	for _, l := range m.cfg.Listeners {
		if l.Enable {
			m.startLocked(l)
		}
	}
	return nil
}

// Stop 实现 module.Module。
func (m *Module) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for key := range m.runners {
		m.stopLocked(key)
	}
	return nil
}

func (m *Module) startLocked(l Listener) {
	if m.ctx == nil {
		return
	}
	label := l.Name
	if label == "" {
		label = ":" + strconv.Itoa(l.Port)
	}
	r, err := newListenerRunner(l, m.logs.Logger(label), m.auth, m.waf)
	if err == nil {
		err = r.start(m.certs, m.cfg.Settings.TrustedProxies)
	}
	if err != nil {
		m.errors[l.Key] = err.Error()
		m.logs.Logger(label).Errorf("启动失败: %v", err)
		return
	}
	delete(m.errors, l.Key)
	m.runners[l.Key] = r
}

func (m *Module) stopLocked(key string) {
	if r, ok := m.runners[key]; ok {
		r.stop()
		delete(m.runners, key)
	}
	delete(m.errors, key)
}

func (m *Module) restartLocked(key string) {
	m.stopLocked(key)
	if !m.cfg.Settings.Enable {
		return
	}
	for _, l := range m.cfg.Listeners {
		if l.Key == key && l.Enable {
			m.startLocked(l)
		}
	}
}

func (m *Module) saveLocked() error {
	return m.store.Save(ID, m.cfg)
}

// ListenerView 是带运行状态的监听器。
type ListenerView struct {
	Listener
	Running bool                        `json:"running"`
	Error   string                      `json:"error,omitempty"`
	Stats   map[string]map[string]int64 `json:"stats"`
	// Upstreams 为各站点的后端健康状态，键为站点 key
	Upstreams map[string][]UpstreamStatus `json:"upstreams,omitempty"`
}

func (m *Module) viewLocked(l *Listener) any {
	v := ListenerView{Listener: *l, Error: m.errors[l.Key], Stats: map[string]map[string]int64{}}
	if r, ok := m.runners[l.Key]; ok {
		v.Running = true
		for _, s := range r.sites {
			v.Stats[s.site.Key] = s.stats.snapshot()
			if st := s.upstreamStatus(); st != nil {
				if v.Upstreams == nil {
					v.Upstreams = map[string][]UpstreamStatus{}
				}
				v.Upstreams[s.site.Key] = st
			}
		}
		if r.deflt != nil {
			v.Stats["default"] = r.deflt.stats.snapshot()
		}
	}
	return v
}

// Routes 实现 module.Module。
func (m *Module) Routes(r module.Router) {
	listeners := &crud.Collection[Listener]{
		Mu:       &m.mu,
		Items:    &m.cfg.Listeners,
		Save:     m.saveLocked,
		Key:      func(l *Listener) *string { return &l.Key },
		Enabled:  func(l *Listener) *bool { return &l.Enable },
		Validate: func(l *Listener) error { return l.Validate() },
		View:     m.viewLocked,
		Changed:  m.restartLocked,
		Removed:  m.stopLocked,
	}
	listeners.Register(r, "/listeners")
	r.HandleFunc("GET /settings", func(w http.ResponseWriter, req *http.Request) {
		m.mu.Lock()
		defer m.mu.Unlock()
		s := m.cfg.Settings
		s.AuthSecret = ""
		s.OAuthProviders = append([]OAuthProvider{}, s.OAuthProviders...)
		for i := range s.OAuthProviders {
			s.OAuthProviders[i].ClientSecret = crud.Mask(s.OAuthProviders[i].ClientSecret)
		}
		httpx.OK(w, s)
	})
	r.HandleFunc("PUT /settings", m.handlePutSettings)
	r.HandleFunc("GET /stats", m.handleStats)
	r.HandleFunc("GET /metrics", m.handleMetrics)
	module.DocOf(r, "GET /stats", module.Doc{Summary: "站点分钟级统计（?listener=&site=&minutes=）", Resp: &SiteStatsView{}})
	module.DocOf(r, "GET /metrics", module.Doc{Summary: "Prometheus 文本格式指标"})
	r.HandleFunc("GET /bans", m.handleListBans)
	r.HandleFunc("POST /bans", m.handleAddBan)
	r.HandleFunc("DELETE /bans", m.handleRemoveBan)
	r.HandleFunc("GET /geoip", m.handleGeoIPInfo)
	r.HandleFunc("POST /geoip/upload", m.handleGeoIPUpload)
	r.HandleFunc("POST /geoip/download", m.handleGeoIPDownload)
	r.HandleFunc("DELETE /geoip", m.handleGeoIPDelete)
	module.DocOf(r, "GET /bans", module.Doc{Summary: "当前被封禁的 IP", Resp: []BanInfo{}})
	module.DocOf(r, "POST /bans", module.Doc{Summary: "手动封禁 IP", Req: &banRequest{}})
	module.DocOf(r, "DELETE /bans", module.Doc{Summary: "解除封禁（body 为 {\"key\": 地址}）", Req: &banRequest{}})
	module.DocOf(r, "GET /geoip", module.Doc{Summary: "GeoIP 数据库状态", Resp: &geoip.Info{}})
	module.DocOf(r, "POST /geoip/upload", module.Doc{Summary: "上传 GeoIP 数据库（multipart 字段 file，.mmdb）"})
	module.DocOf(r, "DELETE /geoip", module.Doc{Summary: "移除 GeoIP 数据库", Resp: &geoip.Info{}})
	module.DocOf(r, "POST /geoip/download", module.Doc{Summary: "从地址下载 GeoIP 数据库（.mmdb）", Req: &geoipDownload{}})
}

func (m *Module) handlePutSettings(w http.ResponseWriter, r *http.Request) {
	var s Settings
	if err := httpx.ReadJSON(r, &s); err != nil {
		httpx.Error(w, http.StatusBadRequest, "%v", err)
		return
	}
	if _, err := parsePrefixes(s.TrustedProxies); err != nil {
		httpx.Error(w, http.StatusBadRequest, "%v", err)
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	old := m.cfg.Settings
	s.AuthSecret = old.AuthSecret
	// 保留被脱敏的 Client Secret
	oldSecrets := map[string]string{}
	for _, p := range old.OAuthProviders {
		oldSecrets[p.Key] = p.ClientSecret
	}
	keys := map[string]bool{}
	for i := range s.OAuthProviders {
		p := &s.OAuthProviders[i]
		p.Key = strings.ToLower(strings.TrimSpace(p.Key))
		crud.KeepSecret(oldSecrets[p.Key], &p.ClientSecret)
		if err := p.Validate(); err != nil {
			httpx.Error(w, http.StatusBadRequest, "%v", err)
			return
		}
		if keys[p.Key] {
			httpx.Error(w, http.StatusBadRequest, "第三方登录标识 %s 重复", p.Key)
			return
		}
		keys[p.Key] = true
	}
	s.OAuthCallbackBase = strings.TrimSpace(s.OAuthCallbackBase)
	if s.OAuthCallbackBase != "" {
		if u, err := url.Parse(s.OAuthCallbackBase); err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			httpx.Error(w, http.StatusBadRequest, "回调地址需为完整的 http(s) 地址")
			return
		}
	}
	s.TrustedProxies = cleanList(s.TrustedProxies)
	if s.AutoBanSeconds < 0 || time.Duration(s.AutoBanSeconds)*time.Second > maxBanDuration {
		httpx.Error(w, http.StatusBadRequest, "自动封禁时长需在 0 到 30 天之间")
		return
	}
	m.cfg.Settings = s
	m.auth.setBanSeconds(s.AutoBanSeconds)
	if err := m.saveLocked(); err != nil {
		m.cfg.Settings = old
		httpx.Error(w, http.StatusInternalServerError, "保存失败: %v", err)
		return
	}
	m.applyOAuthLocked()
	for key := range m.runners {
		m.stopLocked(key)
	}
	if s.Enable {
		for _, l := range m.cfg.Listeners {
			if l.Enable {
				m.startLocked(l)
			}
		}
	}
	s.AuthSecret = ""
	for i := range s.OAuthProviders {
		s.OAuthProviders[i].ClientSecret = crud.Mask(s.OAuthProviders[i].ClientSecret)
	}
	httpx.OK(w, s)
}
