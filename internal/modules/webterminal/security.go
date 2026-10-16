package webterminal

import (
	"net/http"
	"sync"
	"time"

	"github.com/wangwanjie/gately/internal/auth"
	"github.com/wangwanjie/gately/internal/httpx"
	"github.com/wangwanjie/gately/internal/logbuf"
)

// Security 是用户协议与二次验证配置。
type Security struct {
	AgreementAccepted bool `json:"agreementAccepted"`
	TwoFAEnable       bool `json:"twoFAEnable"`
	// TwoFAKey 为 TOTP 密钥，仅在服务端保存，接口不输出
	TwoFAKey string `json:"twoFAKey"`
}

const (
	// verifyTTL 是二次验证通过后的有效期
	verifyTTL = 30 * time.Minute
	// maxVerifyFails 连续失败达到该次数后锁定 verifyLockout
	maxVerifyFails = 5
	verifyLockout  = time.Minute
)

type failState struct {
	count int
	until time.Time
}

// twoFAState 保存各登录令牌的二次验证状态（仅内存）。
type twoFAState struct {
	mu       sync.Mutex
	verified map[string]time.Time
	fails    map[string]*failState
}

func newTwoFAState() *twoFAState {
	return &twoFAState{verified: map[string]time.Time{}, fails: map[string]*failState{}}
}

// accessKey 返回二次验证状态的键：优先登录令牌，没有令牌（其他认证方式）时退化为来源 IP。
func accessKey(r *http.Request) string {
	if t := auth.TokenFromRequest(r); t != "" {
		return "t:" + t
	}
	return "ip:" + httpx.ClientIP(r)
}

func (s *twoFAState) isVerified(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	exp, ok := s.verified[key]
	if !ok {
		return false
	}
	if time.Now().After(exp) {
		delete(s.verified, key)
		return false
	}
	return true
}

func (s *twoFAState) mark(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	// 顺带清理过期条目
	for k, exp := range s.verified {
		if now.After(exp) {
			delete(s.verified, k)
		}
	}
	s.verified[key] = now.Add(verifyTTL)
	delete(s.fails, key)
}

func (s *twoFAState) clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.verified = map[string]time.Time{}
	s.fails = map[string]*failState{}
}

// locked 判断是否处于失败锁定期。
func (s *twoFAState) locked(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	f := s.fails[key]
	return f != nil && time.Now().Before(f.until)
}

func (s *twoFAState) fail(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	f := s.fails[key]
	if f == nil {
		f = &failState{}
		s.fails[key] = f
	}
	f.count++
	if f.count >= maxVerifyFails {
		f.count = 0
		f.until = time.Now().Add(verifyLockout)
	}
}

// requireAccess 是终端与 SFTP 接口共用的门禁：协议未接受或需要二次验证时写入 403 并返回 false。
func (m *Module) requireAccess(w http.ResponseWriter, r *http.Request) bool {
	m.mu.Lock()
	sec := m.cfg.Security
	m.mu.Unlock()
	if !sec.AgreementAccepted {
		httpx.WriteJSON(w, http.StatusForbidden, map[string]any{"error": "请先阅读并接受 Web 终端用户协议", "needAgreement": true})
		return false
	}
	if sec.TwoFAEnable && sec.TwoFAKey != "" && !m.twofa.isVerified(accessKey(r)) {
		httpx.WriteJSON(w, http.StatusForbidden, map[string]any{"error": "需要先完成二次验证", "need2fa": true})
		return false
	}
	return true
}

type codeBody struct {
	Code string `json:"code"`
}

// registerSecurity 注册安全相关接口。
func (m *Module) registerSecurity(r interface {
	HandleFunc(string, http.HandlerFunc)
}) {
	r.HandleFunc("GET /security", func(w http.ResponseWriter, req *http.Request) {
		m.mu.Lock()
		sec := m.cfg.Security
		m.mu.Unlock()
		httpx.OK(w, map[string]bool{
			"agreementAccepted": sec.AgreementAccepted,
			"twoFAEnable":       sec.TwoFAEnable,
			"verified":          m.twofa.isVerified(accessKey(req)),
		})
	})
	r.HandleFunc("POST /security/agreement", func(w http.ResponseWriter, req *http.Request) {
		m.mu.Lock()
		defer m.mu.Unlock()
		old := m.cfg.Security
		m.cfg.Security.AgreementAccepted = true
		if err := m.saveLocked(); err != nil {
			m.cfg.Security = old
			httpx.Error(w, http.StatusInternalServerError, "保存失败: %v", err)
			return
		}
		m.logs.Add(logbuf.LevelInfo, "", "已接受 Web 终端用户协议")
		httpx.OK(w, map[string]bool{"ok": true})
	})
	r.HandleFunc("POST /security/2fa/setup", func(w http.ResponseWriter, req *http.Request) {
		m.mu.Lock()
		enabled := m.cfg.Security.TwoFAEnable
		m.mu.Unlock()
		if enabled {
			httpx.Error(w, http.StatusBadRequest, "二次验证已启用，请先关闭后再重新绑定")
			return
		}
		secret := auth.NewTOTPSecret()
		httpx.OK(w, map[string]string{"secret": secret, "url": auth.TOTPURL(secret, "WebTerminal")})
	})
	r.HandleFunc("POST /security/2fa/enable", func(w http.ResponseWriter, req *http.Request) {
		var body struct {
			Secret string `json:"secret"`
			Code   string `json:"code"`
		}
		if err := httpx.ReadJSON(req, &body); err != nil {
			httpx.Error(w, http.StatusBadRequest, "%v", err)
			return
		}
		key := accessKey(req)
		if m.twofa.locked(key) {
			httpx.Error(w, http.StatusTooManyRequests, "验证失败次数过多，请稍后再试")
			return
		}
		if body.Secret == "" || !auth.ValidateTOTP(body.Secret, body.Code, time.Now()) {
			m.twofa.fail(key)
			httpx.Error(w, http.StatusBadRequest, "验证码错误")
			return
		}
		m.mu.Lock()
		defer m.mu.Unlock()
		old := m.cfg.Security
		m.cfg.Security.TwoFAEnable = true
		m.cfg.Security.TwoFAKey = body.Secret
		if err := m.saveLocked(); err != nil {
			m.cfg.Security = old
			httpx.Error(w, http.StatusInternalServerError, "保存失败: %v", err)
			return
		}
		// 刚证明过持有验证器，当前令牌直接视为已验证
		m.twofa.mark(key)
		m.logs.Add(logbuf.LevelInfo, "", "已启用 Web 终端二次验证")
		httpx.OK(w, map[string]bool{"ok": true})
	})
	r.HandleFunc("POST /security/2fa/disable", func(w http.ResponseWriter, req *http.Request) {
		var body codeBody
		if err := httpx.ReadJSON(req, &body); err != nil {
			httpx.Error(w, http.StatusBadRequest, "%v", err)
			return
		}
		key := accessKey(req)
		if m.twofa.locked(key) {
			httpx.Error(w, http.StatusTooManyRequests, "验证失败次数过多，请稍后再试")
			return
		}
		m.mu.Lock()
		defer m.mu.Unlock()
		sec := m.cfg.Security
		if !sec.TwoFAEnable {
			httpx.OK(w, map[string]bool{"ok": true})
			return
		}
		if !auth.ValidateTOTP(sec.TwoFAKey, body.Code, time.Now()) {
			m.twofa.fail(key)
			httpx.Error(w, http.StatusBadRequest, "验证码错误")
			return
		}
		m.cfg.Security.TwoFAEnable = false
		m.cfg.Security.TwoFAKey = ""
		if err := m.saveLocked(); err != nil {
			m.cfg.Security = sec
			httpx.Error(w, http.StatusInternalServerError, "保存失败: %v", err)
			return
		}
		m.twofa.clear()
		m.logs.Add(logbuf.LevelInfo, "", "已关闭 Web 终端二次验证")
		httpx.OK(w, map[string]bool{"ok": true})
	})
	r.HandleFunc("POST /security/verify2fa", func(w http.ResponseWriter, req *http.Request) {
		var body codeBody
		if err := httpx.ReadJSON(req, &body); err != nil {
			httpx.Error(w, http.StatusBadRequest, "%v", err)
			return
		}
		key := accessKey(req)
		if m.twofa.locked(key) {
			httpx.Error(w, http.StatusTooManyRequests, "验证失败次数过多，请稍后再试")
			return
		}
		m.mu.Lock()
		sec := m.cfg.Security
		m.mu.Unlock()
		if !sec.TwoFAEnable || sec.TwoFAKey == "" {
			httpx.OK(w, map[string]bool{"ok": true})
			return
		}
		if !auth.ValidateTOTP(sec.TwoFAKey, body.Code, time.Now()) {
			m.twofa.fail(key)
			httpx.Error(w, http.StatusBadRequest, "验证码错误")
			return
		}
		m.twofa.mark(key)
		httpx.OK(w, map[string]bool{"ok": true})
	})
	r.HandleFunc("GET /security/check2fa", func(w http.ResponseWriter, req *http.Request) {
		m.mu.Lock()
		sec := m.cfg.Security
		m.mu.Unlock()
		need := sec.TwoFAEnable && sec.TwoFAKey != "" && !m.twofa.isVerified(accessKey(req))
		httpx.OK(w, map[string]bool{"need": need})
	})
}
