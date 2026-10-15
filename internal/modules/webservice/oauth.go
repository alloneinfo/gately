package webservice

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// 第三方登录类型
const (
	OAuthGitHub = "github"
	OAuthOIDC   = "oidc"
)

// OAuthProvider 是一个自建的第三方登录配置（使用用户自己申请的 OAuth 应用，不经过任何中转服务器）。
type OAuthProvider struct {
	// Key 同时作为身份前缀，例如 key 为 github 时身份写作 github:用户名
	Key          string `json:"key"`
	Name         string `json:"name"`
	Type         string `json:"type"`
	ClientID     string `json:"clientId"`
	ClientSecret string `json:"clientSecret"`
	// Issuer 为 OIDC 签发者地址，例如 https://accounts.google.com
	Issuer string `json:"issuer"`
}

// 第三方登录相关的保留路径
const (
	oauthStart    = authPath + "/oauth/start"
	oauthCallback = authPath + "/oauth/callback"
	oauthFinish   = authPath + "/oauth/finish"
	// oauthNonceCookie 把登录流程绑定到发起登录的浏览器
	oauthNonceCookie = "gately_oauth_nonce"
	stateTTL         = 10 * time.Minute
	ticketTTL        = 2 * time.Minute
)

// GitHub 端点，测试时可替换
var githubEndpoints = struct{ Authorize, Token, User string }{
	Authorize: "https://github.com/login/oauth/authorize",
	Token:     "https://github.com/login/oauth/access_token",
	User:      "https://api.github.com/user",
}

func (p *OAuthProvider) validate() error {
	p.Key = strings.ToLower(strings.TrimSpace(p.Key))
	if p.Key == "" || strings.ContainsAny(p.Key, ": /") {
		return errors.New("第三方登录的标识不能为空，且不能包含冒号、空格或斜杠")
	}
	if p.ClientID == "" || p.ClientSecret == "" {
		return fmt.Errorf("第三方登录 %s 缺少 Client ID 或 Secret", p.Key)
	}
	switch p.Type {
	case OAuthGitHub:
	case OAuthOIDC:
		if !strings.HasPrefix(p.Issuer, "https://") {
			return fmt.Errorf("第三方登录 %s 的 Issuer 需为 https 地址", p.Key)
		}
	default:
		return fmt.Errorf("未知的第三方登录类型 %q", p.Type)
	}
	return nil
}

// oidcConfig 是 OIDC 发现文档中用到的端点。
type oidcConfig struct {
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	UserinfoEndpoint      string `json:"userinfo_endpoint"`
}

var (
	oidcCacheMu sync.Mutex
	oidcCache   = map[string]oidcConfig{}
)

func discoverOIDC(ctx context.Context, issuer string) (oidcConfig, error) {
	oidcCacheMu.Lock()
	c, ok := oidcCache[issuer]
	oidcCacheMu.Unlock()
	if ok {
		return c, nil
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(issuer, "/")+"/.well-known/openid-configuration", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return c, err
	}
	defer resp.Body.Close()
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&c); err != nil {
		return c, err
	}
	if c.AuthorizationEndpoint == "" || c.TokenEndpoint == "" || c.UserinfoEndpoint == "" {
		return c, errors.New("OIDC 发现文档不完整")
	}
	oidcCacheMu.Lock()
	oidcCache[issuer] = c
	oidcCacheMu.Unlock()
	return c, nil
}

// authorizeURL 生成跳转到身份提供方的地址。
func (p *OAuthProvider) authorizeURL(ctx context.Context, redirect, state string) (string, error) {
	v := url.Values{"client_id": {p.ClientID}, "redirect_uri": {redirect}, "state": {state}}
	switch p.Type {
	case OAuthGitHub:
		v.Set("scope", "read:user user:email")
		return githubEndpoints.Authorize + "?" + v.Encode(), nil
	default:
		c, err := discoverOIDC(ctx, p.Issuer)
		if err != nil {
			return "", err
		}
		v.Set("response_type", "code")
		v.Set("scope", "openid email profile")
		return c.AuthorizationEndpoint + "?" + v.Encode(), nil
	}
}

// exchange 用授权码换取访问令牌并查询身份，返回身份标识（GitHub 为登录名，OIDC 为邮箱，缺失时为 sub）。
func (p *OAuthProvider) exchange(ctx context.Context, code, redirect string) (string, error) {
	tokenURL, userURL := githubEndpoints.Token, githubEndpoints.User
	if p.Type == OAuthOIDC {
		c, err := discoverOIDC(ctx, p.Issuer)
		if err != nil {
			return "", err
		}
		tokenURL, userURL = c.TokenEndpoint, c.UserinfoEndpoint
	}
	form := url.Values{"client_id": {p.ClientID}, "client_secret": {p.ClientSecret}, "code": {code}, "redirect_uri": {redirect}, "grant_type": {"authorization_code"}}
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	var tok struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
		Desc        string `json:"error_description"`
	}
	err = json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&tok)
	resp.Body.Close()
	if err != nil {
		return "", fmt.Errorf("解析令牌响应失败: %w", err)
	}
	if tok.AccessToken == "" {
		return "", fmt.Errorf("获取令牌失败: %s %s", tok.Error, tok.Desc)
	}
	req, _ = http.NewRequestWithContext(ctx, http.MethodGet, userURL, nil)
	req.Header.Set("Authorization", "Bearer "+tok.AccessToken)
	req.Header.Set("Accept", "application/json")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var u struct {
		Login         string          `json:"login"`
		Email         string          `json:"email"`
		EmailVerified json.RawMessage `json:"email_verified"`
		Sub           string          `json:"sub"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&u); err != nil {
		return "", fmt.Errorf("解析用户信息失败: %w", err)
	}
	if p.Type == OAuthGitHub {
		if u.Login == "" {
			return "", errors.New("未获取到 GitHub 用户名")
		}
		return u.Login, nil
	}
	// 未验证的邮箱不可信，退回使用 sub
	// 部分提供方把 email_verified 编码为字符串，只有明确为 true 时才采用邮箱
	verified := string(u.EmailVerified) == "true" || string(u.EmailVerified) == `"true"`
	if u.Email != "" && verified {
		return strings.ToLower(u.Email), nil
	}
	if u.Sub == "" {
		return "", errors.New("未获取到用户标识")
	}
	return u.Sub, nil
}

// signBlob 生成带过期时间的签名数据，用于 state 与回跳票据。
func (a *authenticator) signBlob(kind string, v any, ttl time.Duration) string {
	body, _ := json.Marshal(map[string]any{"k": kind, "v": v, "e": time.Now().Add(ttl).Unix()})
	p := base64.RawURLEncoding.EncodeToString(body)
	m := hmac.New(sha256.New, a.secret)
	m.Write([]byte(p))
	return p + "." + hex.EncodeToString(m.Sum(nil))
}

func (a *authenticator) verifyBlob(kind, token string, out any) error {
	p, sig, ok := strings.Cut(token, ".")
	if !ok {
		return errors.New("格式错误")
	}
	m := hmac.New(sha256.New, a.secret)
	m.Write([]byte(p))
	if !hmac.Equal([]byte(sig), []byte(hex.EncodeToString(m.Sum(nil)))) {
		return errors.New("签名无效")
	}
	raw, err := base64.RawURLEncoding.DecodeString(p)
	if err != nil {
		return err
	}
	var env struct {
		K string          `json:"k"`
		V json.RawMessage `json:"v"`
		E int64           `json:"e"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return err
	}
	if env.K != kind {
		return errors.New("类型不匹配")
	}
	if time.Now().Unix() > env.E {
		return errors.New("已过期，请重新登录")
	}
	return json.Unmarshal(env.V, out)
}

// oauthState 在跳转身份提供方时携带，回调时用于定位原站点。
type oauthState struct {
	Provider string `json:"p"`
	Origin   string `json:"o"` // 原站点的 scheme://host
	Scope    string `json:"s"`
	Next     string `json:"n"`
	Nonce    string `json:"r"`
}

// oauthTicket 由认证域名签发，原站点凭它建立会话。
type oauthTicket struct {
	Identity string `json:"i"`
	Host     string `json:"h"`
	Scope    string `json:"s"`
	Next     string `json:"n"`
	Nonce    string `json:"r"`
}

// oauthEnv 是处理第三方登录需要的全局配置。
type oauthEnv struct {
	providers map[string]OAuthProvider
	// callbackBase 为认证域名的外部地址，例如 https://auth.example.com:4434
	callbackBase string
}

func (e oauthEnv) callbackURL() string {
	return strings.TrimRight(e.callbackBase, "/") + oauthCallback
}

func requestOrigin(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return scheme + "://" + r.Host
}

func randomNonce() string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// identityAllowed 判断身份是否在站点允许列表中。列表项格式：提供方key:标识，或 提供方key:* 表示该提供方的所有用户。
func identityAllowed(s Site, provider, identity string) bool {
	for _, allowed := range s.OAuthUsers {
		p, id, ok := strings.Cut(strings.TrimSpace(allowed), ":")
		if !ok || !strings.EqualFold(p, provider) {
			continue
		}
		if id == "*" || strings.EqualFold(id, identity) {
			return true
		}
	}
	return false
}

// handleOAuthStart 在站点域名上发起第三方登录。
func (a *authenticator) handleOAuthStart(w http.ResponseWriter, r *http.Request, s Site, env oauthEnv) {
	pk := r.URL.Query().Get("p")
	p, ok := env.providers[pk]
	if !ok || env.callbackBase == "" {
		http.Error(w, "第三方登录未配置", http.StatusBadRequest)
		return
	}
	nonce := randomNonce()
	state := a.signBlob("state", oauthState{Provider: pk, Origin: requestOrigin(r), Scope: s.scope, Next: safeNext(r.URL.Query().Get("next")), Nonce: nonce}, stateTTL)
	target, err := p.authorizeURL(r.Context(), env.callbackURL(), state)
	if err != nil {
		http.Error(w, "无法连接身份提供方: "+err.Error(), http.StatusBadGateway)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name: oauthNonceCookie, Value: nonce, Path: authPath, MaxAge: int(stateTTL / time.Second),
		HttpOnly: true, Secure: r.TLS != nil, SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, target, http.StatusFound)
}

// handleOAuthCallback 在认证域名上处理身份提供方的回调，然后带票据跳回原站点。
func (a *authenticator) handleOAuthCallback(w http.ResponseWriter, r *http.Request, env oauthEnv) {
	var st oauthState
	rawState := r.URL.Query().Get("state")
	if err := a.verifyBlob("state", rawState, &st); err != nil {
		http.Error(w, "登录状态无效: "+err.Error(), http.StatusBadRequest)
		return
	}
	if !a.consumeOnce("state|"+rawState, stateTTL) {
		http.Error(w, "登录状态已使用，请重新登录", http.StatusBadRequest)
		return
	}
	p, ok := env.providers[st.Provider]
	if !ok {
		http.Error(w, "第三方登录配置已变化，请重试", http.StatusBadRequest)
		return
	}
	if e := r.URL.Query().Get("error"); e != "" {
		http.Error(w, "身份提供方拒绝登录: "+e, http.StatusForbidden)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	id, err := p.exchange(ctx, r.URL.Query().Get("code"), env.callbackURL())
	if err != nil {
		http.Error(w, "登录失败: "+err.Error(), http.StatusBadGateway)
		return
	}
	u, err := url.Parse(st.Origin)
	if err != nil {
		http.Error(w, "原站点地址无效", http.StatusBadRequest)
		return
	}
	ticket := a.signBlob("ticket", oauthTicket{Identity: st.Provider + ":" + id, Host: hostOnly(u.Host), Scope: st.Scope, Next: st.Next, Nonce: st.Nonce}, ticketTTL)
	http.Redirect(w, r, st.Origin+oauthFinish+"?ticket="+url.QueryEscape(ticket), http.StatusFound)
}

// handleOAuthFinish 在原站点上校验票据与允许列表，建立会话。
func (a *authenticator) handleOAuthFinish(w http.ResponseWriter, r *http.Request, s Site) {
	var t oauthTicket
	raw := r.URL.Query().Get("ticket")
	if err := a.verifyBlob("ticket", raw, &t); err != nil {
		http.Error(w, "登录票据无效: "+err.Error(), http.StatusBadRequest)
		return
	}
	if t.Host != hostOnly(r.Host) || t.Scope != s.scope {
		http.Error(w, "登录票据与站点不匹配", http.StatusBadRequest)
		return
	}
	// 票据必须由发起登录的同一浏览器使用，防止把他人的登录结果注入受害者浏览器
	c, err := r.Cookie(oauthNonceCookie)
	if err != nil || !hmac.Equal([]byte(c.Value), []byte(t.Nonce)) {
		http.Error(w, "登录状态与当前浏览器不匹配，请重新登录", http.StatusBadRequest)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: oauthNonceCookie, Value: "", Path: authPath, MaxAge: -1})
	if !a.consumeOnce("ticket|"+raw, ticketTTL) {
		http.Error(w, "登录票据已使用", http.StatusBadRequest)
		return
	}
	provider, identity, _ := strings.Cut(t.Identity, ":")
	if !identityAllowed(s, provider, identity) {
		http.Error(w, fmt.Sprintf("账号 %s 没有访问此站点的权限", t.Identity), http.StatusForbidden)
		return
	}
	a.issueSession(w, r, s, t.Identity)
	http.Redirect(w, r, safeNext(t.Next), http.StatusFound)
}
