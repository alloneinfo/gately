package webservice

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/wangwanjie/gately/internal/banlist"
	"github.com/wangwanjie/gately/internal/i18n"
	"github.com/wangwanjie/gately/internal/thirdauth"
)

// 网页认证使用的保留路径与会话时长
const (
	authPath     = "/__gately_auth"
	sessionTTL   = 7 * 24 * time.Hour
	maxFailures  = 10
	failureReset = 15 * time.Minute
)

// authenticator 处理站点的基本认证与网页认证（登录页 + 签名 Cookie）。
type authenticator struct {
	secret []byte
	mu     sync.Mutex
	fails  map[string]*failCount
	// used 记录已使用的一次性令牌（OAuth state 与票据）及其过期时间
	used map[string]int64
	// oauth 为第三方登录配置，设置变更时整体替换
	oauthMu sync.RWMutex
	oauth   oauthEnv

	// bans 为共享封禁表；banMu 保护自动封禁时长；shareUses 记录分享链接已兑换次数（受 mu 保护）
	bans      *banlist.Store
	banMu     sync.Mutex
	autoBan   time.Duration
	shareUses map[string]int

	// shared 为共享认证用户与单点登录域，设置变更时整体替换
	shared sharedStore
}

func (a *authenticator) setOAuth(env oauthEnv) {
	a.oauthMu.Lock()
	a.oauth = env
	a.oauthMu.Unlock()
}

func (a *authenticator) oauthEnv() oauthEnv {
	a.oauthMu.RLock()
	defer a.oauthMu.RUnlock()
	return a.oauth
}

type failCount struct {
	n        int
	inflight int
	first    time.Time
}

func newAuthenticator(secret string) *authenticator {
	return &authenticator{secret: []byte(secret), fails: map[string]*failCount{}, used: map[string]int64{},
		bans: banlist.Default, shareUses: map[string]int{}}
}

// verifyUser 校验用户名密码，密码支持明文（兼容 Lucky）与 bcrypt 哈希。
func verifyUser(users []string, name, pass string) bool {
	ok := false
	for _, u := range users {
		un, up, _ := strings.Cut(u, ":")
		if subtle.ConstantTimeCompare([]byte(un), []byte(name)) != 1 {
			continue
		}
		if strings.HasPrefix(up, "$2") {
			ok = bcrypt.CompareHashAndPassword([]byte(up), []byte(pass)) == nil
		} else {
			ok = subtle.ConstantTimeCompare([]byte(up), []byte(pass)) == 1
		}
		if ok {
			return true
		}
	}
	return false
}

// limitKey 返回限流键：IPv6 按 /64 聚合，避免攻击者轮换同一网段内的地址绕过限制。
func limitKey(ip string) string {
	a, err := netip.ParseAddr(ip)
	if err != nil {
		return ip
	}
	a = a.Unmap()
	if a.Is6() {
		p, _ := a.Prefix(64)
		return p.String()
	}
	return a.String()
}

// reserve 为一次密码校验占位；失败次数加上进行中的校验达到上限时拒绝，防止并发绕过。
func (a *authenticator) reserve(key string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	now := time.Now()
	// 表过大时清理过期项，防止大量来源地址撑大内存
	if len(a.fails) > 4096 {
		for k, f := range a.fails {
			if f.inflight == 0 && now.Sub(f.first) > failureReset {
				delete(a.fails, k)
			}
		}
	}
	f := a.fails[key]
	if f == nil || (f.inflight == 0 && now.Sub(f.first) > failureReset) {
		f = &failCount{first: now}
		a.fails[key] = f
	}
	if f.n+f.inflight >= maxFailures {
		return false
	}
	f.inflight++
	return true
}

// release 结束占位并记录结果；成功不清零计数，避免交替使用正确密码重置限流。
func (a *authenticator) release(key string, ok bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	f := a.fails[key]
	if f == nil {
		return
	}
	f.inflight--
	if !ok {
		f.n++
		// 恰好达到上限时触发一次自动封禁
		if f.n == maxFailures {
			a.autoBanFor(key)
		}
	}
}

// consumeOnce 标记一次性令牌已使用，重复使用返回 false。
func (a *authenticator) consumeOnce(token string, ttl time.Duration) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	now := time.Now().Unix()
	for k, exp := range a.used {
		if exp < now {
			delete(a.used, k)
		}
	}
	if _, seen := a.used[token]; seen {
		return false
	}
	a.used[token] = now + int64(ttl/time.Second) + 1
	return true
}

// cookieName 按 监听器/站点 区分 Cookie，避免同域名下不同站点或端口共享登录态。
func cookieName(s Site) string {
	scope := s.scope
	if scope == "" {
		scope = s.Key
	}
	sum := sha256.Sum256([]byte(scope))
	return "gately_site_" + hex.EncodeToString(sum[:6])
}

// siteSession 是网页认证会话 Cookie 的内容。
type siteSession struct {
	User  string `json:"u"`
	Host  string `json:"h"`
	Scope string `json:"s"`
	// Cred 为密码用户的凭据指纹，修改密码后旧会话失效
	Cred string `json:"c,omitempty"`
	// Aliases 为第三方登录身份的其他标识，校验允许列表时与 User 一起匹配
	Aliases []string `json:"a,omitempty"`
	// Shared 表示共享认证用户的会话（Cred 为共享用户凭据指纹，Scope 为 sso 时跨站有效）
	Shared bool `json:"sh,omitempty"`
}

// credVersion 返回用户当前凭据的指纹（HMAC，Cookie 中不暴露密码哈希）。
func (a *authenticator) credVersion(users []string, name string) string {
	for _, u := range users {
		if un, _, _ := strings.Cut(u, ":"); un == name {
			m := hmac.New(sha256.New, a.secret)
			m.Write([]byte("cred|" + u))
			return hex.EncodeToString(m.Sum(nil)[:8])
		}
	}
	return ""
}

// issueSession 写出站点会话 Cookie。
func (a *authenticator) issueSession(w http.ResponseWriter, r *http.Request, s Site, user string, aliases ...string) {
	exp := time.Now().Add(sessionTTL)
	sess := siteSession{User: user, Host: hostOnly(r.Host), Scope: s.scope, Cred: a.credVersion(s.Users, user), Aliases: aliases}
	http.SetCookie(w, &http.Cookie{
		Name: cookieName(s), Value: a.signBlob("session", sess, sessionTTL),
		Path: "/", Expires: exp, HttpOnly: true, Secure: r.TLS != nil, SameSite: http.SameSiteLaxMode,
	})
}

// validCookie 校验会话 Cookie；用户被删除或密码修改后会话立即失效。
func (a *authenticator) validCookie(r *http.Request, s Site) bool {
	c, err := r.Cookie(cookieName(s))
	if err != nil {
		return false
	}
	var sess siteSession
	if err := a.verifyBlob("session", c.Value, &sess); err != nil {
		return false
	}
	if sess.Shared {
		return s.SharedUsers && a.validShared(r, s, &sess)
	}
	if sess.Host != hostOnly(r.Host) || sess.Scope != s.scope {
		return false
	}
	for _, u := range s.Users {
		if name, _, _ := strings.Cut(u, ":"); name == sess.User {
			return sess.Cred != "" && hmac.Equal([]byte(sess.Cred), []byte(a.credVersion(s.Users, sess.User)))
		}
	}
	// 第三方登录的会话用户为 提供方:标识，允许列表变化后立即失效
	if provider, id, ok := strings.Cut(sess.User, ":"); ok && sess.Cred == "" {
		return identityAllowed(s, thirdauth.Identity{Provider: provider, Primary: id, Aliases: sess.Aliases})
	}
	return false
}

// isBrowser 粗略判断是否为浏览器访问，非浏览器回退到基本认证（与 Lucky 行为一致）。
func isBrowser(r *http.Request) bool {
	ua := r.UserAgent()
	return strings.Contains(ua, "Mozilla/") && strings.Contains(r.Header.Get("Accept"), "text/html")
}

// check 执行认证，返回 true 表示放行；否则已写出响应。
func (a *authenticator) check(w http.ResponseWriter, r *http.Request, s Site) bool {
	ip := clientAddr(r).Addr().String()
	if s.WebAuth {
		switch r.URL.Path {
		case oauthStart:
			a.handleOAuthStart(w, r, s, a.oauthEnv())
			return false
		case oauthFinish:
			a.handleOAuthFinish(w, r, s)
			return false
		}
		if strings.HasPrefix(r.URL.Path, authPath) {
			a.handleLogin(w, r, s, ip)
			return false
		}
		if a.validCookie(r, s) || (s.SharedUsers && a.validSSOCookie(r, s)) {
			return true
		}
		if isBrowser(r) {
			target := authPath + "/login?next=" + url.QueryEscape(r.URL.RequestURI())
			http.Redirect(w, r, target, http.StatusFound)
			return false
		}
	}
	// 基本认证（网页认证对非浏览器的回退）；只配置了第三方身份时非浏览器无法登录
	if len(s.Users) == 0 && !s.SharedUsers {
		http.Error(w, "401 Unauthorized", http.StatusUnauthorized)
		return false
	}
	if name, pass, ok := r.BasicAuth(); ok {
		key := limitKey(ip)
		if !a.reserve(key) {
			http.Error(w, "429 Too Many Requests", http.StatusTooManyRequests)
			return false
		}
		valid := verifyUser(s.Users, name, pass)
		if !valid && s.SharedUsers {
			// 基本认证无法输入动态验证码，启用了 TOTP 的共享用户只能通过网页登录
			if u, ok := a.shared.get().checkPassword(name, pass); ok && u.TOTPSecret == "" {
				valid = true
			}
		}
		a.release(key, valid)
		if valid {
			return true
		}
	}
	w.Header().Set("WWW-Authenticate", `Basic realm="Gately", charset="UTF-8"`)
	http.Error(w, "401 Unauthorized", http.StatusUnauthorized)
	return false
}

const loginSrc = `<!doctype html>
<html lang="zh-CN"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>登录 - {{.Host}}</title>
<style>
body{margin:0;min-height:100vh;display:flex;align-items:center;justify-content:center;background:#f2f5f4;font-family:-apple-system,BlinkMacSystemFont,"PingFang SC","Microsoft YaHei",sans-serif}
form{background:#fff;padding:28px 32px;border-radius:10px;box-shadow:0 2px 16px rgba(0,0,0,.08);width:300px}
h1{font-size:18px;margin:0 0 4px}p{margin:0 0 18px;color:#888;font-size:13px}
input{width:100%;box-sizing:border-box;padding:10px;margin-bottom:12px;border:1px solid #ccd;border-radius:6px;font-size:14px}
button{width:100%;padding:10px;border:0;border-radius:6px;background:#0e7c66;color:#fff;font-size:15px;cursor:pointer}
.err{color:#d33;font-size:13px;margin-bottom:10px}
.oauth{display:block;text-align:center;margin-top:10px;padding:9px;border:1px solid #ccd;border-radius:6px;color:#333;text-decoration:none;font-size:14px}
</style></head><body>
<form method="post" action="{{.Action}}">
<h1>访问 {{.Host}}</h1><p>此站点需要登录</p>
{{if .Error}}<div class="err">{{.Error}}</div>{{end}}
{{if .HasPassword}}
<input name="username" placeholder="用户名" autocomplete="username" required autofocus>
<input name="password" type="password" placeholder="密码" autocomplete="current-password" required>
{{if .HasTOTP}}<input name="code" placeholder="动态验证码（未启用可留空）" inputmode="numeric" autocomplete="one-time-code" maxlength="6">{{end}}
<input type="hidden" name="next" value="{{.Next}}">
<button type="submit">登录</button>
{{end}}
{{range .Providers}}<a class="oauth" href="{{$.StartURL}}?p={{.Key}}&next={{$.Next}}">使用 {{.Name}} 登录</a>{{end}}
</form></body></html>`

// safeNext 只允许站内相对路径，防止开放重定向。
func safeNext(next string) string {
	if next == "" || next[0] != '/' || strings.ContainsRune(next, '\\') {
		return "/"
	}
	if len(next) > 1 && next[1] == '/' {
		return "/"
	}
	for _, c := range next {
		if c < 0x20 || c == 0x7f {
			return "/"
		}
	}
	u, err := url.Parse(next)
	if err != nil || u.Scheme != "" || u.Host != "" || u.User != nil || strings.HasPrefix(u.Path, authPath) {
		return "/"
	}
	return next
}

// loginProvider 是登录页展示的第三方登录入口。
type loginProvider struct{ Key, Name string }

// siteProviders 返回站点允许、且已配置的第三方登录提供方。
func (a *authenticator) siteProviders(s Site) []loginProvider {
	env := a.oauthEnv()
	if env.callbackBase == "" {
		return nil
	}
	var out []loginProvider
	for _, p := range thirdauth.Providers(s.OAuthUsers, env.providers) {
		out = append(out, loginProvider{Key: p.Key, Name: p.DisplayName()})
	}
	return out
}

func (a *authenticator) handleLogin(w http.ResponseWriter, r *http.Request, s Site, ip string) {
	host := hostOnly(r.Host)
	data := map[string]any{
		"Host": host, "Action": authPath + "/login", "Next": safeNext(r.URL.Query().Get("next")),
		"HasPassword": len(s.Users) > 0 || s.SharedUsers, "Providers": a.siteProviders(s), "StartURL": oauthStart,
		"HasTOTP": s.SharedUsers && a.shared.get().hasTOTP(),
	}
	if r.URL.Path == authPath+"/logout" {
		http.SetCookie(w, &http.Cookie{Name: cookieName(s), Value: "", Path: "/", MaxAge: -1})
		if domain := a.shared.get().ssoFor(host); domain != "" && s.SharedUsers {
			// 单点登录的会话写在父域，退出时一并清除
			http.SetCookie(w, &http.Cookie{Name: ssoCookie, Value: "", Path: "/", Domain: domain, MaxAge: -1})
		}
		http.Redirect(w, r, authPath+"/login", http.StatusFound)
		return
	}
	if r.Method == http.MethodPost {
		_ = r.ParseForm()
		data["Next"] = safeNext(r.PostForm.Get("next"))
		next, _ := data["Next"].(string)
		key := limitKey(ip)
		if !a.reserve(key) {
			data["Error"] = "失败次数过多，请稍后再试"
		} else {
			user := r.PostForm.Get("username")
			pass := r.PostForm.Get("password")
			valid := len(s.Users) > 0 && verifyUser(s.Users, user, pass)
			msg := "用户名或密码错误"
			var shared AuthUser
			if !valid && s.SharedUsers {
				var errMsg string
				shared, errMsg = a.loginShared(user, pass, r.PostForm.Get("code"))
				valid = errMsg == ""
				if !valid {
					msg = errMsg
				}
			}
			a.release(key, valid)
			if valid {
				if shared.Name != "" {
					a.issueSharedSession(w, r, s, shared)
				} else {
					a.issueSession(w, r, s, user)
				}
				http.Redirect(w, r, next, http.StatusFound)
				return
			}
			data["Error"] = msg
		}
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	if _, failed := data["Error"]; failed {
		w.WriteHeader(http.StatusUnauthorized)
	}
	lang := visitorLang(r)
	if msg, ok := data["Error"].(string); ok {
		data["Error"] = i18n.Tr(lang, msg)
	}
	_ = executeLocalized(w, "login", loginSrc, lang, data)
}

// hostOnly 去掉 Host 中的端口。
func hostOnly(h string) string {
	h = strings.ToLower(h)
	if i := strings.LastIndexByte(h, ':'); i > 0 && !strings.HasSuffix(h, "]") {
		return strings.Trim(h[:i], "[]")
	}
	return strings.Trim(h, "[]")
}
