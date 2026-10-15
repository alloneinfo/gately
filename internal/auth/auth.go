// Package auth 实现管理员账号、会话令牌与登录限流。
//
// 令牌格式：base64url(payload) + "." + base64url(HMAC-SHA256(payload))。
// 修改密码时轮换签名密钥，使所有旧会话立即失效。
package auth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// CookieName 是会话 Cookie 名称。
const CookieName = "gately_token"

// 登录限流参数：窗口期内连续失败达到上限后锁定该 IP。
const (
	maxFailures  = 5
	failWindow   = 15 * time.Minute
	lockDuration = 15 * time.Minute
)

var (
	// ErrInvalidCredentials 表示用户名或密码错误。
	ErrInvalidCredentials = errors.New("用户名或密码错误")
	// ErrLocked 表示该 IP 因多次失败被临时锁定。
	ErrLocked = errors.New("登录失败次数过多，请稍后再试")
	// ErrTOTPRequired 表示需要输入两步验证码。
	ErrTOTPRequired = errors.New("请输入两步验证码")
)

// Account 是持久化的管理员账号信息。
type Account struct {
	Username      string `json:"username"`
	PasswordHash  string `json:"passwordHash"`
	SessionSecret string `json:"sessionSecret"`
	TokenTTLHours int    `json:"tokenTTLHours"`
	// TOTPSecret 非空表示已启用两步验证
	TOTPSecret string `json:"totpSecret,omitempty"`
}

type failState struct {
	count       int
	first       time.Time
	lockedUntil time.Time
}

// Manager 管理账号校验和会话。
type Manager struct {
	mu    sync.RWMutex
	acc   Account
	save  func(Account) error
	fails map[string]*failState
	now   func() time.Time
}

// NewManager 用已加载的账号创建管理器，save 用于持久化账号变更。
func NewManager(acc Account, save func(Account) error) *Manager {
	return &Manager{acc: acc, save: save, fails: map[string]*failState{}, now: time.Now}
}

// EnsureInitialized 在首次启动时创建 admin 账号与随机密码。
// 返回值 generated 非空表示新生成的初始密码，调用方需要展示给用户。
func (m *Manager) EnsureInitialized() (generated string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	changed := false
	if m.acc.SessionSecret == "" {
		m.acc.SessionSecret = randomString(32)
		changed = true
	}
	if m.acc.TokenTTLHours <= 0 {
		m.acc.TokenTTLHours = 72
		changed = true
	}
	if m.acc.Username == "" || m.acc.PasswordHash == "" {
		generated = randomString(12)
		hash, err := bcrypt.GenerateFromPassword([]byte(generated), bcrypt.DefaultCost)
		if err != nil {
			return "", err
		}
		m.acc.Username = "admin"
		m.acc.PasswordHash = string(hash)
		changed = true
	}
	if changed {
		if err := m.save(m.acc); err != nil {
			return "", err
		}
	}
	return generated, nil
}

// Username 返回当前管理员用户名。
func (m *Manager) Username() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.acc.Username
}

// Login 校验凭据并签发令牌。启用两步验证时 code 必填。
func (m *Manager) Login(ip, username, password string, code ...string) (string, time.Time, error) {
	if m.isLocked(ip) {
		return "", time.Time{}, ErrLocked
	}
	m.mu.RLock()
	acc := m.acc
	m.mu.RUnlock()

	userOK := subtle.ConstantTimeCompare([]byte(username), []byte(acc.Username)) == 1
	// 无论用户名是否正确都执行一次 bcrypt，避免通过耗时判断用户名
	passErr := bcrypt.CompareHashAndPassword([]byte(acc.PasswordHash), []byte(password))
	if !userOK || passErr != nil {
		m.recordFailure(ip)
		return "", time.Time{}, ErrInvalidCredentials
	}
	if acc.TOTPSecret != "" {
		c := ""
		if len(code) > 0 {
			c = code[0]
		}
		if c == "" {
			return "", time.Time{}, ErrTOTPRequired
		}
		if !ValidateTOTP(acc.TOTPSecret, c, m.now()) {
			m.recordFailure(ip)
			return "", time.Time{}, errors.New("两步验证码错误")
		}
	}
	m.clearFailures(ip)
	exp := m.now().Add(time.Duration(acc.TokenTTLHours) * time.Hour)
	return m.sign(acc, exp), exp, nil
}

// ChangeCredentials 修改用户名和密码，需要提供旧密码；成功后旧令牌全部失效。
func (m *Manager) ChangeCredentials(oldPassword, newUsername, newPassword string) error {
	newUsername = strings.TrimSpace(newUsername)
	if newUsername == "" {
		return errors.New("用户名不能为空")
	}
	if len(newPassword) < 8 {
		return errors.New("新密码至少 8 位")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if bcrypt.CompareHashAndPassword([]byte(m.acc.PasswordHash), []byte(oldPassword)) != nil {
		return errors.New("原密码错误")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	next := m.acc
	next.Username = newUsername
	next.PasswordHash = string(hash)
	next.SessionSecret = randomString(32)
	if err := m.save(next); err != nil {
		return err
	}
	m.acc = next
	return nil
}

type payload struct {
	User string `json:"u"`
	Exp  int64  `json:"e"`
}

func (m *Manager) sign(acc Account, exp time.Time) string {
	body, _ := json.Marshal(payload{User: acc.Username, Exp: exp.Unix()})
	p := base64.RawURLEncoding.EncodeToString(body)
	mac := hmac.New(sha256.New, []byte(acc.SessionSecret))
	mac.Write([]byte(p))
	return p + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// Verify 校验令牌，返回用户名。
func (m *Manager) Verify(token string) (string, bool) {
	p, sig, found := strings.Cut(token, ".")
	if !found {
		return "", false
	}
	m.mu.RLock()
	acc := m.acc
	m.mu.RUnlock()
	mac := hmac.New(sha256.New, []byte(acc.SessionSecret))
	mac.Write([]byte(p))
	want := mac.Sum(nil)
	got, err := base64.RawURLEncoding.DecodeString(sig)
	if err != nil || !hmac.Equal(got, want) {
		return "", false
	}
	raw, err := base64.RawURLEncoding.DecodeString(p)
	if err != nil {
		return "", false
	}
	var pl payload
	if err := json.Unmarshal(raw, &pl); err != nil {
		return "", false
	}
	if pl.User != acc.Username || m.now().Unix() > pl.Exp {
		return "", false
	}
	return pl.User, true
}

// TokenFromRequest 从 Authorization 头或 Cookie 中取令牌。
func TokenFromRequest(r *http.Request) string {
	if h := r.Header.Get("Authorization"); strings.HasPrefix(h, "Bearer ") {
		return strings.TrimPrefix(h, "Bearer ")
	}
	if c, err := r.Cookie(CookieName); err == nil {
		return c.Value
	}
	return ""
}

type ctxKey struct{}

// UserFromContext 返回中间件写入的用户名。
func UserFromContext(ctx context.Context) string {
	u, _ := ctx.Value(ctxKey{}).(string)
	return u
}

// Require 是要求登录的中间件。
func (m *Manager) Require(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := m.Verify(TokenFromRequest(r))
		if !ok {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"未登录或登录已过期"}` + "\n"))
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, user)))
	})
}

func (m *Manager) isLocked(ip string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	st := m.fails[ip]
	return st != nil && m.now().Before(st.lockedUntil)
}

func (m *Manager) recordFailure(ip string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := m.now()
	st := m.fails[ip]
	if st == nil || now.Sub(st.first) > failWindow {
		st = &failState{first: now}
		m.fails[ip] = st
	}
	st.count++
	if st.count >= maxFailures {
		st.lockedUntil = now.Add(lockDuration)
		st.count = 0
		st.first = now
	}
	// 顺带清理过期记录，防止 map 无限增长
	for k, v := range m.fails {
		if now.Sub(v.first) > failWindow && now.After(v.lockedUntil) {
			delete(m.fails, k)
		}
	}
}

func (m *Manager) clearFailures(ip string) {
	m.mu.Lock()
	delete(m.fails, ip)
	m.mu.Unlock()
}

func randomString(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(b)[:n]
}

// TOTPEnabled 返回是否已启用两步验证。
func (m *Manager) TOTPEnabled() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.acc.TOTPSecret != ""
}

// SetTOTP 启用或关闭两步验证。secret 为空表示关闭；需要提供当前密码。
func (m *Manager) SetTOTP(password, secret string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if bcrypt.CompareHashAndPassword([]byte(m.acc.PasswordHash), []byte(password)) != nil {
		return errors.New("密码错误")
	}
	next := m.acc
	next.TOTPSecret = secret
	if err := m.save(next); err != nil {
		return err
	}
	m.acc = next
	return nil
}
