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
	"net/netip"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// CookieName 是会话 Cookie 名称。
const CookieName = "gately_token"

// 登录限流默认参数：窗口期内连续失败达到上限后锁定该 IP。上限与锁定时长可在设置中调整。
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
	count int
	// inflight 为正在校验中的请求数，校验前预占，防止并发请求绕过次数限制
	inflight    int
	first       time.Time
	lockedUntil time.Time
}

// accountKey 用于账号设置类操作（修改密码、两步验证）的失败计数。
const accountKey = "__account__"

// bcryptSem 限制同时进行的 bcrypt 校验数量，避免未认证请求打满 CPU。
var bcryptSem = make(chan struct{}, 4)

// limitKey 返回限流键：IPv6 按 /64 聚合，避免攻击者在一个网段内轮换地址。
func limitKey(ip string) string {
	a, err := netip.ParseAddr(ip)
	if err != nil {
		return ip
	}
	a = a.Unmap()
	if a.Is6() {
		return netip.PrefixFrom(a, 64).Masked().String()
	}
	return a.String()
}

// Manager 管理账号校验和会话。
type Manager struct {
	mu    sync.RWMutex
	acc   Account
	save  func(Account) error
	fails map[string]*failState
	now   func() time.Time
	// lastTOTPStep 为最近一次使用的验证码时间步，防止验证码在有效期内被重放
	lastTOTPStep int64
	// maxFail / lockDur 为当前生效的限流参数
	maxFail int
	lockDur time.Duration
}

// SetLimits 设置连续失败上限与锁定时长，非正数表示使用默认值。
func (m *Manager) SetLimits(maxFail int, lock time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if maxFail <= 0 {
		maxFail = maxFailures
	}
	if lock <= 0 {
		lock = lockDuration
	}
	m.maxFail, m.lockDur = maxFail, lock
}

// TokenTTLHours 返回会话有效期（小时）。
func (m *Manager) TokenTTLHours() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.acc.TokenTTLHours
}

// SetTokenTTL 修改会话有效期并持久化，已签发的会话不受影响。
func (m *Manager) SetTokenTTL(hours int) error {
	if hours < 1 || hours > 24*365 {
		return errors.New("会话有效期应在 1 小时到 365 天之间")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.acc.TokenTTLHours == hours {
		return nil
	}
	acc := m.acc
	acc.TokenTTLHours = hours
	if err := m.save(acc); err != nil {
		return err
	}
	m.acc = acc
	return nil
}

// NewManager 用已加载的账号创建管理器，save 用于持久化账号变更。
func NewManager(acc Account, save func(Account) error) *Manager {
	return &Manager{acc: acc, save: save, fails: map[string]*failState{}, now: time.Now, maxFail: maxFailures, lockDur: lockDuration}
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
	key := limitKey(ip)
	if !m.reserve(key) {
		return "", time.Time{}, ErrLocked
	}
	ok := false
	defer func() { m.release(key, ok) }()

	m.mu.RLock()
	acc := m.acc
	m.mu.RUnlock()

	userOK := subtle.ConstantTimeCompare([]byte(username), []byte(acc.Username)) == 1
	// 无论用户名是否正确都执行一次 bcrypt，避免通过耗时判断用户名
	passErr := checkPassword(acc.PasswordHash, password)
	if !userOK || passErr != nil {
		return "", time.Time{}, ErrInvalidCredentials
	}
	if acc.TOTPSecret != "" {
		c := ""
		if len(code) > 0 {
			c = code[0]
		}
		if c == "" {
			// 密码正确但缺验证码，不计为失败
			ok = true
			return "", time.Time{}, ErrTOTPRequired
		}
		if !m.consumeTOTP(acc.TOTPSecret, c) {
			return "", time.Time{}, errors.New("两步验证码错误")
		}
	}
	ok = true
	exp := m.now().Add(time.Duration(acc.TokenTTLHours) * time.Hour)
	return m.sign(acc, exp), exp, nil
}

// checkPassword 在全局并发上限内执行 bcrypt 比较。
func checkPassword(hash, password string) error {
	bcryptSem <- struct{}{}
	defer func() { <-bcryptSem }()
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

// consumeTOTP 校验验证码，每个时间步只能使用一次。
func (m *Manager) consumeTOTP(secret, code string) bool {
	step, ok := matchTOTP(secret, code, m.now())
	if !ok {
		return false
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if step <= m.lastTOTPStep {
		return false
	}
	m.lastTOTPStep = step
	return true
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
	if err := m.verifyAccountPassword(oldPassword); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
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

// WithUser 把用户名写入上下文（API 令牌等其他认证方式使用）。
func WithUser(ctx context.Context, user string) context.Context {
	return context.WithValue(ctx, ctxKey{}, user)
}

// EntrySecret 返回用于签名安全入口 Cookie 的密钥。
func (m *Manager) EntrySecret() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.acc.SessionSecret
}

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

// reserve 检查是否锁定并预占一次尝试；返回 false 表示应拒绝。
func (m *Manager) reserve(key string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := m.now()
	st := m.fails[key]
	if st == nil || (now.Sub(st.first) > failWindow && st.inflight == 0) {
		if st != nil && now.Before(st.lockedUntil) {
			return false
		}
		st = &failState{first: now}
		m.fails[key] = st
	}
	if now.Before(st.lockedUntil) || st.count+st.inflight >= m.maxFail {
		return false
	}
	st.inflight++
	return true
}

// release 结束一次尝试：成功时清除记录，失败时累计并在达到上限后锁定。
func (m *Manager) release(key string, success bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	st := m.fails[key]
	if st == nil {
		return
	}
	st.inflight--
	now := m.now()
	if success {
		if st.inflight == 0 {
			delete(m.fails, key)
		}
		return
	}
	st.count++
	if st.count >= m.maxFail {
		st.lockedUntil = now.Add(m.lockDur)
		st.count = 0
		st.first = now
	}
	// 顺带清理过期记录，防止 map 无限增长
	for k, v := range m.fails {
		if v.inflight == 0 && now.Sub(v.first) > failWindow && now.After(v.lockedUntil) {
			delete(m.fails, k)
		}
	}
}

// verifyAccountPassword 校验账号设置操作中的当前密码，失败次数与登录一样受限。
func (m *Manager) verifyAccountPassword(password string) error {
	if !m.reserve(accountKey) {
		return ErrLocked
	}
	m.mu.RLock()
	hash := m.acc.PasswordHash
	m.mu.RUnlock()
	err := checkPassword(hash, password)
	m.release(accountKey, err == nil)
	if err != nil {
		return errors.New("原密码错误")
	}
	return nil
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

// SetTOTP 启用或关闭两步验证。secret 为空表示关闭，关闭时还需提供当前验证码。
// 成功后轮换会话密钥，使此前签发的令牌全部失效。
func (m *Manager) SetTOTP(password, code, secret string) error {
	if err := m.verifyAccountPassword(password); err != nil {
		return err
	}
	m.mu.RLock()
	current := m.acc.TOTPSecret
	m.mu.RUnlock()
	if secret == "" && current != "" && !m.consumeTOTP(current, code) {
		return errors.New("两步验证码错误")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	next := m.acc
	next.TOTPSecret = secret
	next.SessionSecret = randomString(32)
	if err := m.save(next); err != nil {
		return err
	}
	m.acc = next
	return nil
}
