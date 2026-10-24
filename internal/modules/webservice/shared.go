package webservice

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/wangwanjie/gately/internal/auth"
	"github.com/wangwanjie/gately/internal/crud"
)

// AuthUser 是可在多个站点共用的认证用户。站点开启“使用共享用户”后，网页认证与基本认证都接受这些用户，
// 设置了 TOTP 的用户在网页登录时还需输入动态验证码（相当于 Authelia 的 two_factor 策略）。
type AuthUser struct {
	Name string `json:"name"`
	// Password 保存 bcrypt 哈希；接口提交明文时自动哈希，读取时脱敏
	Password string `json:"password"`
	// TOTPSecret 为 Base32 密钥，留空表示不启用二次验证；读取时脱敏
	TOTPSecret string `json:"totpSecret"`
	Remark     string `json:"remark"`
	// Email、DisplayName、Groups 为可选的身份信息，供内置 OIDC 身份提供方作为 email、name、groups 声明提供给应用
	Email       string   `json:"email,omitempty"`
	DisplayName string   `json:"displayName,omitempty"`
	Groups      []string `json:"groups,omitempty"`
}

// ssoCookie 是共享用户跨站登录态的 Cookie 名称。
const ssoCookie = "gately_sso"

// sharedEnv 是共享用户与单点登录域的运行时副本。
type sharedEnv struct {
	users map[string]AuthUser
	// domain 为单点登录 Cookie 域，空表示每个站点单独登录
	domain string
}

type sharedStore struct {
	mu  sync.RWMutex
	env sharedEnv
}

func (s *sharedStore) set(users []AuthUser, domain string) {
	m := make(map[string]AuthUser, len(users))
	for _, u := range users {
		m[u.Name] = u
	}
	s.mu.Lock()
	s.env = sharedEnv{users: m, domain: strings.ToLower(strings.Trim(domain, ". "))}
	s.mu.Unlock()
}

func (s *sharedStore) get() sharedEnv {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.env
}

// ssoFor 返回 host 可使用的单点登录 Cookie 域；host 不在该域下时返回空。
func (e sharedEnv) ssoFor(host string) string {
	if e.domain == "" {
		return ""
	}
	if host == e.domain || strings.HasSuffix(host, "."+e.domain) {
		return e.domain
	}
	return ""
}

// checkPassword 校验共享用户的密码（常量时间比较用户名，未命中时也做一次哈希比较，避免通过耗时探测用户名）。
func (e sharedEnv) checkPassword(name, pass string) (AuthUser, bool) {
	u, ok := e.users[name]
	if !ok || subtle.ConstantTimeCompare([]byte(u.Name), []byte(name)) != 1 {
		_ = bcrypt.CompareHashAndPassword(dummyHash(), []byte(pass))
		return AuthUser{}, false
	}
	return u, bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(pass)) == nil
}

// dummyHash 是用户不存在时用于比较的哈希，使耗时与真实用户一致。
var dummyHash = sync.OnceValue(func() []byte {
	h, _ := bcrypt.GenerateFromPassword([]byte("gately-dummy"), bcrypt.DefaultCost)
	return h
})

// sharedCred 返回共享用户凭据指纹，修改密码或 TOTP 后旧会话立即失效。
func (a *authenticator) sharedCred(u AuthUser) string {
	m := hmac.New(sha256.New, a.secret)
	m.Write([]byte("shared|" + u.Name + "|" + u.Password + "|" + u.TOTPSecret))
	return hex.EncodeToString(m.Sum(nil)[:8])
}

// loginShared 用共享用户登录：校验密码与 TOTP，返回给访客的错误提示（空表示成功）。
// needCode 为 true 表示密码正确，但该用户启用了 TOTP 且尚未提交验证码，应进入验证码步骤。
func (a *authenticator) loginShared(name, pass, code string) (u AuthUser, msg string, needCode bool) {
	env := a.shared.get()
	u, ok := env.checkPassword(name, pass)
	if !ok {
		return AuthUser{}, "用户名或密码错误", false
	}
	if u.TOTPSecret == "" {
		return u, "", false
	}
	if strings.TrimSpace(code) == "" {
		return u, "请输入动态验证码", true
	}
	if msg := a.checkTOTP(u, code); msg != "" {
		return AuthUser{}, msg, false
	}
	return u, "", false
}

// checkTOTP 校验共享用户的动态验证码，返回给访客的错误提示（空表示通过）。
func (a *authenticator) checkTOTP(u AuthUser, code string) string {
	code = strings.TrimSpace(code)
	if code == "" {
		return "请输入动态验证码"
	}
	if !auth.ValidateTOTP(u.TOTPSecret, code, time.Now()) {
		return "动态验证码错误"
	}
	// 同一验证码在有效期内只能使用一次，防止被截获后重放
	if !a.consumeOnce("totp|"+u.Name+"|"+code, 2*time.Minute) {
		return "动态验证码已使用，请等待下一个"
	}
	return ""
}

// issueSharedSession 写出共享用户会话。配置了单点登录域且站点在该域下时写到父域，其余站点免登录。
func (a *authenticator) issueSharedSession(w http.ResponseWriter, r *http.Request, s Site, u AuthUser) {
	host := hostOnly(r.Host)
	sess := siteSession{User: u.Name, Shared: true, Cred: a.sharedCred(u)}
	c := &http.Cookie{Path: "/", Expires: time.Now().Add(sessionTTL), HttpOnly: true, Secure: r.TLS != nil, SameSite: http.SameSiteLaxMode}
	if domain := a.shared.get().ssoFor(host); domain != "" {
		sess.Scope = "sso"
		c.Name, c.Domain = ssoCookie, domain
	} else {
		sess.Host, sess.Scope = host, s.scope
		c.Name = cookieName(s)
	}
	c.Value = a.signBlob("session", sess, sessionTTL)
	http.SetCookie(w, c)
}

// validShared 校验共享用户会话：单点登录 Cookie 或本站点 Cookie。
func (a *authenticator) validShared(r *http.Request, s Site, sess *siteSession) bool {
	env := a.shared.get()
	u, ok := env.users[sess.User]
	if !ok || !hmac.Equal([]byte(sess.Cred), []byte(a.sharedCred(u))) {
		return false
	}
	host := hostOnly(r.Host)
	if sess.Scope == "sso" {
		return env.ssoFor(host) != ""
	}
	return sess.Host == host && sess.Scope == s.scope
}

// validSSOCookie 检查单点登录 Cookie（站点开启共享用户时）。
func (a *authenticator) validSSOCookie(r *http.Request, s Site) bool {
	c, err := r.Cookie(ssoCookie)
	if err != nil {
		return false
	}
	var sess siteSession
	if err := a.verifyBlob("session", c.Value, &sess); err != nil || !sess.Shared {
		return false
	}
	return a.validShared(r, s, &sess)
}

// normalizeAuthUsers 校验共享用户并处理密码与 TOTP：明文密码哈希保存，脱敏占位符恢复旧值。
func normalizeAuthUsers(users []AuthUser, old []AuthUser) ([]AuthUser, error) {
	prev := map[string]AuthUser{}
	for _, u := range old {
		prev[u.Name] = u
	}
	seen := map[string]bool{}
	out := make([]AuthUser, 0, len(users))
	for _, u := range users {
		u.Name = strings.TrimSpace(u.Name)
		if u.Name == "" || strings.ContainsAny(u.Name, ": ") {
			return nil, fmt.Errorf("共享用户名 %q 无效（不能为空或包含空格、冒号）", u.Name)
		}
		if seen[u.Name] {
			return nil, fmt.Errorf("共享用户 %s 重复", u.Name)
		}
		seen[u.Name] = true
		o := prev[u.Name]
		switch {
		case u.Password == "" || u.Password == crud.SecretMask:
			// 未修改密码
			u.Password = o.Password
		case strings.HasPrefix(u.Password, "$2"):
			if _, err := bcrypt.Cost([]byte(u.Password)); err != nil {
				return nil, fmt.Errorf("共享用户 %s 的密码哈希无效", u.Name)
			}
		default:
			h, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
			if err != nil {
				return nil, err
			}
			u.Password = string(h)
		}
		if u.Password == "" {
			return nil, fmt.Errorf("请为共享用户 %s 设置密码", u.Name)
		}
		crud.KeepSecret(o.TOTPSecret, &u.TOTPSecret)
		u.TOTPSecret = strings.ToUpper(strings.ReplaceAll(u.TOTPSecret, " ", ""))
		if u.TOTPSecret != "" && !auth.ValidTOTPSecret(u.TOTPSecret) {
			return nil, fmt.Errorf("共享用户 %s 的 TOTP 密钥无效", u.Name)
		}
		u.Remark = strings.TrimSpace(u.Remark)
		u.Email = strings.TrimSpace(u.Email)
		if u.Email != "" && (!strings.Contains(u.Email, "@") || strings.ContainsAny(u.Email, " \t<>")) {
			return nil, fmt.Errorf("共享用户 %s 的邮箱 %q 无效", u.Name, u.Email)
		}
		u.DisplayName = strings.TrimSpace(u.DisplayName)
		u.Groups = uniqueList(cleanList(u.Groups))
		for _, g := range u.Groups {
			if strings.ContainsAny(g, " \t,") {
				return nil, fmt.Errorf("共享用户 %s 的分组 %q 无效（不能包含空格或逗号）", u.Name, g)
			}
		}
		if len(u.Groups) == 0 {
			u.Groups = nil
		}
		out = append(out, u)
	}
	return out, nil
}

// maskAuthUsers 返回脱敏后的共享用户，用于接口输出。
func maskAuthUsers(users []AuthUser) []AuthUser {
	out := make([]AuthUser, len(users))
	for i, u := range users {
		u.Password = crud.Mask(u.Password)
		u.TOTPSecret = crud.Mask(u.TOTPSecret)
		out[i] = u
	}
	return out
}

// validateSSODomain 校验单点登录域：必须是至少两段的域名。
func validateSSODomain(d string) (string, error) {
	d = strings.ToLower(strings.Trim(strings.TrimSpace(d), "."))
	if d == "" {
		return "", nil
	}
	if !strings.Contains(d, ".") || strings.ContainsAny(d, " /:*") {
		return "", errors.New("单点登录域需为域名，例如 example.com")
	}
	return d, nil
}
