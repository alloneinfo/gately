package webservice

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"html/template"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
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
	// oauth 为第三方登录配置，设置变更时整体替换
	oauthMu sync.RWMutex
	oauth   oauthEnv
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
	n     int
	first time.Time
}

func newAuthenticator(secret string) *authenticator {
	return &authenticator{secret: []byte(secret), fails: map[string]*failCount{}}
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

func (a *authenticator) locked(ip string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	f := a.fails[ip]
	if f == nil {
		return false
	}
	if time.Since(f.first) > failureReset {
		delete(a.fails, ip)
		return false
	}
	return f.n >= maxFailures
}

func (a *authenticator) fail(ip string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	f := a.fails[ip]
	if f == nil || time.Since(f.first) > failureReset {
		f = &failCount{first: time.Now()}
		a.fails[ip] = f
	}
	f.n++
}

// cookieName 按站点区分 Cookie，避免同域名下不同站点共享登录态。
func cookieName(siteKey string) string {
	return "gately_site_" + siteKey
}

func (a *authenticator) sign(user, host string, exp int64) string {
	payload := user + "|" + host + "|" + strconv.FormatInt(exp, 10)
	m := hmac.New(sha256.New, a.secret)
	m.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString([]byte(payload)) + "." + hex.EncodeToString(m.Sum(nil))
}

// validCookie 校验会话 Cookie；用户被删除后会话立即失效。
func (a *authenticator) validCookie(r *http.Request, s Site) bool {
	c, err := r.Cookie(cookieName(s.Key))
	if err != nil {
		return false
	}
	p, sig, ok := strings.Cut(c.Value, ".")
	if !ok {
		return false
	}
	raw, err := base64.RawURLEncoding.DecodeString(p)
	if err != nil {
		return false
	}
	m := hmac.New(sha256.New, a.secret)
	m.Write(raw)
	want := hex.EncodeToString(m.Sum(nil))
	if subtle.ConstantTimeCompare([]byte(sig), []byte(want)) != 1 {
		return false
	}
	parts := strings.Split(string(raw), "|")
	if len(parts) != 3 || parts[1] != hostOnly(r.Host) {
		return false
	}
	exp, _ := strconv.ParseInt(parts[2], 10, 64)
	if time.Now().Unix() > exp {
		return false
	}
	for _, u := range s.Users {
		if name, _, _ := strings.Cut(u, ":"); name == parts[0] {
			return true
		}
	}
	// 第三方登录的会话用户为 提供方:标识，允许列表变化后立即失效
	if provider, id, ok := strings.Cut(parts[0], ":"); ok {
		return identityAllowed(s, provider, id)
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
		if a.validCookie(r, s) {
			return true
		}
		if isBrowser(r) {
			target := authPath + "/login?next=" + url.QueryEscape(r.URL.RequestURI())
			http.Redirect(w, r, target, http.StatusFound)
			return false
		}
	}
	// 基本认证（网页认证对非浏览器的回退）；只配置了第三方身份时非浏览器无法登录
	if len(s.Users) == 0 {
		http.Error(w, "401 Unauthorized", http.StatusUnauthorized)
		return false
	}
	if a.locked(ip) {
		http.Error(w, "429 Too Many Requests", http.StatusTooManyRequests)
		return false
	}
	if name, pass, ok := r.BasicAuth(); ok {
		if verifyUser(s.Users, name, pass) {
			return true
		}
		a.fail(ip)
	}
	w.Header().Set("WWW-Authenticate", `Basic realm="Gately", charset="UTF-8"`)
	http.Error(w, "401 Unauthorized", http.StatusUnauthorized)
	return false
}

var loginTmpl = template.Must(template.New("login").Parse(`<!doctype html>
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
<input type="hidden" name="next" value="{{.Next}}">
<button type="submit">登录</button>
{{end}}
{{range .Providers}}<a class="oauth" href="{{$.StartURL}}?p={{.Key}}&next={{$.Next}}">使用 {{.Name}} 登录</a>{{end}}
</form></body></html>`))

// safeNext 只允许站内相对路径，防止开放重定向。
func safeNext(next string) string {
	if next == "" || !strings.HasPrefix(next, "/") || strings.HasPrefix(next, "//") || strings.HasPrefix(next, authPath) {
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
	seen := map[string]bool{}
	var out []loginProvider
	for _, u := range s.OAuthUsers {
		key, _, _ := strings.Cut(u, ":")
		key = strings.ToLower(key)
		if p, ok := env.providers[key]; ok && !seen[key] {
			seen[key] = true
			name := p.Name
			if name == "" {
				name = p.Key
			}
			out = append(out, loginProvider{Key: key, Name: name})
		}
	}
	return out
}

func (a *authenticator) handleLogin(w http.ResponseWriter, r *http.Request, s Site, ip string) {
	host := hostOnly(r.Host)
	data := map[string]any{
		"Host": host, "Action": authPath + "/login", "Next": safeNext(r.URL.Query().Get("next")),
		"HasPassword": len(s.Users) > 0, "Providers": a.siteProviders(s), "StartURL": oauthStart,
	}
	if r.URL.Path == authPath+"/logout" {
		http.SetCookie(w, &http.Cookie{Name: cookieName(s.Key), Value: "", Path: "/", MaxAge: -1})
		http.Redirect(w, r, authPath+"/login", http.StatusFound)
		return
	}
	if r.Method == http.MethodPost {
		_ = r.ParseForm()
		data["Next"] = safeNext(r.PostForm.Get("next"))
		next, _ := data["Next"].(string)
		switch {
		case a.locked(ip):
			data["Error"] = "失败次数过多，请稍后再试"
		case verifyUser(s.Users, r.PostForm.Get("username"), r.PostForm.Get("password")):
			exp := time.Now().Add(sessionTTL)
			http.SetCookie(w, &http.Cookie{
				Name: cookieName(s.Key), Value: a.sign(r.PostForm.Get("username"), host, exp.Unix()),
				Path: "/", Expires: exp, HttpOnly: true, Secure: r.TLS != nil, SameSite: http.SameSiteLaxMode,
			})
			http.Redirect(w, r, next, http.StatusFound)
			return
		default:
			a.fail(ip)
			data["Error"] = "用户名或密码错误"
		}
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	if _, failed := data["Error"]; failed {
		w.WriteHeader(http.StatusUnauthorized)
	}
	_ = loginTmpl.Execute(w, data)
}

// hostOnly 去掉 Host 中的端口。
func hostOnly(h string) string {
	h = strings.ToLower(h)
	if i := strings.LastIndexByte(h, ':'); i > 0 && !strings.HasSuffix(h, "]") {
		return strings.Trim(h[:i], "[]")
	}
	return strings.Trim(h, "[]")
}
