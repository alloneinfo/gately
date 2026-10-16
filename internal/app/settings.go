package app

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"math/rand/v2"
	"net"
	"net/http"
	"net/netip"
	"runtime"
	"runtime/debug"
	"strings"
	"time"

	"github.com/wangwanjie/gately/internal/auth"
	"github.com/wangwanjie/gately/internal/crud"
	"github.com/wangwanjie/gately/internal/httpx"
	"github.com/wangwanjie/gately/internal/logbuf"
)

// Settings 是全局设置（settings.json）。字段对齐 Lucky 的基础设置中对 Gately 有意义的部分。
type Settings struct {
	// Listen 为管理界面 HTTP 监听地址，例如 :16680
	Listen string `json:"listen"`
	// HTTPSListen 非空时额外以 HTTPS 提供管理界面，证书由证书模块按域名提供
	HTTPSListen string `json:"httpsListen"`
	// ForceHTTPS 把 HTTP 访问重定向到 HTTPS（仅当证书模块有该域名的证书时）
	ForceHTTPS bool `json:"forceHTTPS"`
	// SafePath 为安全入口：设置后需先访问该路径才能打开登录页，其他路径返回 404
	SafePath string `json:"safePath"`
	// LANOnly 只允许内网地址（含 Tailscale 等 CGNAT 地址）访问管理界面
	LANOnly bool `json:"lanOnly"`
	// MaxLoginFailures / LockMinutes 为登录限流参数
	MaxLoginFailures int `json:"maxLoginFailures"`
	LockMinutes      int `json:"lockMinutes"`
	// APIToken 非空时允许脚本以 Authorization: Bearer <token> 调用管理 API
	APIToken string `json:"apiToken"`
	// TimeZone 为 IANA 时区名，例如 Asia/Shanghai；为空使用系统时区
	TimeZone string `json:"timeZone"`
	// DNSServers 为自定义 DNS 服务器，影响所有出站域名解析（DDNS、Webhook、ACME 等）
	DNSServers []string `json:"dnsServers"`
	// LogLines 为每个模块保留的日志条数
	LogLines int `json:"logLines"`
	// GCPercent / MaxProcs 为运行时调优参数，0 表示默认
	GCPercent int `json:"gcPercent"`
	MaxProcs  int `json:"maxProcs"`
}

// 默认值
const (
	defaultLogLines = 2000
	maxLogLines     = 20000
)

func (s *Settings) normalize() {
	if s.Listen == "" {
		s.Listen = DefaultListen
	}
	if s.LogLines <= 0 {
		s.LogLines = defaultLogLines
	}
	s.SafePath = strings.TrimSpace(s.SafePath)
	if s.SafePath != "" && !strings.HasPrefix(s.SafePath, "/") {
		s.SafePath = "/" + s.SafePath
	}
	s.SafePath = strings.TrimRight(s.SafePath, "/")
	var dns []string
	for _, d := range s.DNSServers {
		if d = strings.TrimSpace(d); d != "" {
			dns = append(dns, d)
		}
	}
	s.DNSServers = dns
}

// Validate 校验设置。
func (s *Settings) Validate() error {
	s.normalize()
	for _, addr := range []string{s.Listen, s.HTTPSListen} {
		if addr == "" {
			continue
		}
		if _, port, err := net.SplitHostPort(addr); err != nil || port == "" {
			return fmt.Errorf("监听地址 %q 格式错误，应为 :端口 或 IP:端口", addr)
		}
	}
	if s.HTTPSListen != "" && s.HTTPSListen == s.Listen {
		return errors.New("HTTPS 监听地址不能与 HTTP 相同")
	}
	if s.ForceHTTPS && s.HTTPSListen == "" {
		return errors.New("强制 HTTPS 需要先设置 HTTPS 监听地址")
	}
	if s.SafePath != "" {
		if s.SafePath == "/api" || strings.HasPrefix(s.SafePath, "/api/") || strings.HasPrefix(s.SafePath, "/assets") || strings.ContainsAny(s.SafePath, " ?#") {
			return errors.New("安全入口不能以 /api、/assets 开头，且不能包含空格、? 或 #")
		}
		if len(s.SafePath) < 4 {
			return errors.New("安全入口至少 3 个字符")
		}
	}
	if s.MaxLoginFailures < 0 || s.MaxLoginFailures > 100 {
		return errors.New("连续登录失败上限应在 1~100 之间")
	}
	if s.LockMinutes < 0 || s.LockMinutes > 7*24*60 {
		return errors.New("锁定时长应在 1 分钟到 7 天之间")
	}
	if s.APIToken != "" && s.APIToken != crud.SecretMask && len(s.APIToken) < 16 {
		return errors.New("API 令牌至少 16 个字符")
	}
	if s.TimeZone != "" {
		if _, err := time.LoadLocation(s.TimeZone); err != nil {
			return fmt.Errorf("时区 %q 无效", s.TimeZone)
		}
	}
	for _, d := range s.DNSServers {
		if _, err := netip.ParseAddr(d); err != nil {
			if _, err := netip.ParseAddrPort(d); err != nil {
				return fmt.Errorf("DNS 服务器 %q 应为 IP 或 IP:端口", d)
			}
		}
	}
	if s.LogLines > maxLogLines {
		return fmt.Errorf("日志条数不能超过 %d", maxLogLines)
	}
	if s.GCPercent < 0 || s.GCPercent > 1000 || s.MaxProcs < 0 || s.MaxProcs > 1024 {
		return errors.New("运行时参数超出范围")
	}
	return nil
}

// restartFields 返回需要重启才能生效的字段是否变化。
func restartRequired(old, cur Settings) bool {
	return old.Listen != cur.Listen || old.HTTPSListen != cur.HTTPSListen
}

// applyRuntime 应用无需重启即可生效的设置。
func (a *App) applyRuntime() {
	s := a.settingsSnapshot()
	a.auth.SetLimits(s.MaxLoginFailures, time.Duration(s.LockMinutes)*time.Minute)
	if s.TimeZone != "" {
		if loc, err := time.LoadLocation(s.TimeZone); err == nil {
			time.Local = loc
		}
	}
	setResolver(s.DNSServers)
	a.sysLogs.Resize(s.LogLines)
	for _, e := range a.modules {
		e.logs.Resize(s.LogLines)
	}
	if s.GCPercent > 0 {
		debug.SetGCPercent(s.GCPercent)
	} else {
		debug.SetGCPercent(100)
	}
	if s.MaxProcs > 0 {
		runtime.GOMAXPROCS(s.MaxProcs)
	}
}

// setResolver 让所有出站解析使用指定的 DNS 服务器；为空时恢复系统默认。
func setResolver(servers []string) {
	if len(servers) == 0 {
		net.DefaultResolver = &net.Resolver{}
		return
	}
	addrs := make([]string, 0, len(servers))
	for _, s := range servers {
		if ap, err := netip.ParseAddrPort(s); err == nil {
			addrs = append(addrs, ap.String())
		} else {
			addrs = append(addrs, net.JoinHostPort(s, "53"))
		}
	}
	d := &net.Dialer{Timeout: 5 * time.Second}
	net.DefaultResolver = &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, _ string) (net.Conn, error) {
			// 随机选择起点，失败时依次尝试其他服务器
			start := rand.IntN(len(addrs))
			var lastErr error
			for i := range addrs {
				c, err := d.DialContext(ctx, network, addrs[(start+i)%len(addrs)])
				if err == nil {
					return c, nil
				}
				lastErr = err
			}
			return nil, lastErr
		},
	}
}

func (a *App) settingsSnapshot() Settings {
	a.settingsMu.RLock()
	defer a.settingsMu.RUnlock()
	return a.settings
}

// settingsView 是返回给前端的设置，API 令牌脱敏。
type settingsView struct {
	Settings
	// SessionHours 为登录会话有效期，保存在账号中
	SessionHours int `json:"sessionHours"`
	// RestartRequired 表示有修改需要重启后生效
	RestartRequired bool `json:"restartRequired"`
}

func (a *App) viewSettings() settingsView {
	s := a.settingsSnapshot()
	if s.APIToken != "" {
		s.APIToken = crud.SecretMask
	}
	return settingsView{Settings: s, SessionHours: a.auth.TokenTTLHours(), RestartRequired: a.restartPending.Load()}
}

func (a *App) handleGetSettings(w http.ResponseWriter, r *http.Request) {
	httpx.OK(w, a.viewSettings())
}

func (a *App) handlePutSettings(w http.ResponseWriter, r *http.Request) {
	var req settingsView
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "%v", err)
		return
	}
	next := req.Settings
	if err := next.Validate(); err != nil {
		httpx.Error(w, http.StatusBadRequest, "%v", err)
		return
	}
	if req.SessionHours != 0 {
		if err := a.auth.SetTokenTTL(req.SessionHours); err != nil {
			httpx.Error(w, http.StatusBadRequest, "%v", err)
			return
		}
	}
	a.settingsMu.Lock()
	old := a.settings
	if next.APIToken == crud.SecretMask {
		next.APIToken = old.APIToken
	}
	// 命令行 -listen 覆盖时不改写配置文件中的地址
	if a.opts.Listen != "" {
		next.Listen = a.fileListen
	}
	if err := a.store.Save(settingsName, next); err != nil {
		a.settingsMu.Unlock()
		httpx.Error(w, http.StatusInternalServerError, "保存失败: %v", err)
		return
	}
	if a.opts.Listen != "" {
		a.fileListen = next.Listen
		next.Listen = old.Listen
	}
	a.settings = next
	a.settingsMu.Unlock()
	if restartRequired(old, next) {
		a.restartPending.Store(true)
	}
	a.applyRuntime()
	a.sysLogs.Add(logbuf.LevelInfo, "settings", "全局设置已更新")
	httpx.OK(w, a.viewSettings())
}

// handleGenerateToken 生成一个新的随机 API 令牌（不保存，由前端填入后提交）。
func (a *App) handleGenerateToken(w http.ResponseWriter, r *http.Request) {
	httpx.OK(w, map[string]string{"token": randomHex(24)})
}

// ---- 访问控制中间件 ----

// entryCookie 记录已通过安全入口的浏览器。
const entryCookie = "gately_entry"

func (a *App) entryValue(path string) string {
	m := hmac.New(sha256.New, []byte(a.auth.EntrySecret()))
	m.Write([]byte("entry|" + path))
	return hex.EncodeToString(m.Sum(nil))[:32]
}

// isPrivate 判断地址是否属于内网（含回环、链路本地、CGNAT 与 IPv6 ULA）。
func isPrivate(ip string) bool {
	a, err := netip.ParseAddr(ip)
	if err != nil {
		return false
	}
	a = a.Unmap()
	if a.IsLoopback() || a.IsPrivate() || a.IsLinkLocalUnicast() {
		return true
	}
	return netip.MustParsePrefix("100.64.0.0/10").Contains(a)
}

// apiTokenOK 判断请求是否携带有效的 API 令牌。
func (a *App) apiTokenOK(r *http.Request) bool {
	tok := a.settingsSnapshot().APIToken
	if tok == "" {
		return false
	}
	h := r.Header.Get("Authorization")
	if !strings.HasPrefix(h, "Bearer ") {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(strings.TrimPrefix(h, "Bearer ")), []byte(tok)) == 1
}

// accessControl 依次执行：内网限制、强制 HTTPS、安全入口。
func (a *App) accessControl(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s := a.settingsSnapshot()
		if s.LANOnly && !isPrivate(httpx.ClientIP(r)) {
			http.Error(w, "403 Forbidden", http.StatusForbidden)
			return
		}
		if s.ForceHTTPS && r.TLS == nil && a.httpsReady.Load() && a.hasCert(r.Host) {
			_, port, _ := net.SplitHostPort(s.HTTPSListen)
			target := "https://" + hostOnly(r.Host)
			if port != "443" {
				target += ":" + port
			}
			http.Redirect(w, r, target+r.URL.RequestURI(), http.StatusFound)
			return
		}
		if s.SafePath != "" {
			if r.URL.Path == s.SafePath || r.URL.Path == s.SafePath+"/" {
				http.SetCookie(w, &http.Cookie{Name: entryCookie, Value: a.entryValue(s.SafePath), Path: "/", HttpOnly: true,
					SameSite: http.SameSiteLaxMode, Secure: r.TLS != nil, MaxAge: 365 * 24 * 3600})
				http.Redirect(w, r, "/", http.StatusFound)
				return
			}
			c, err := r.Cookie(entryCookie)
			passed := err == nil && hmac.Equal([]byte(c.Value), []byte(a.entryValue(s.SafePath)))
			if !passed && !a.apiTokenOK(r) {
				// 已登录的会话（例如修改安全入口之前登录）仍可访问
				if _, ok := a.auth.Verify(auth.TokenFromRequest(r)); !ok {
					http.NotFound(w, r)
					return
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}

// hasCert 判断证书模块是否有该域名的证书。
func (a *App) hasCert(host string) bool {
	if a.opts.Certs == nil {
		return false
	}
	return a.opts.Certs.HasCertificate(hostOnly(host))
}

func hostOnly(h string) string {
	if host, _, err := net.SplitHostPort(h); err == nil {
		return host
	}
	return strings.Trim(h, "[]")
}
