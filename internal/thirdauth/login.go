package thirdauth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/wangwanjie/gately/internal/crud"
)

// LoginConfig 是管理界面的第三方登录设置。
type LoginConfig struct {
	Enable bool `json:"enable"`
	// CallbackBase 为管理界面的外部访问地址（须与浏览器打开管理界面使用的地址一致），例如 https://gately.example.com
	CallbackBase string     `json:"callbackBase"`
	Providers    []Provider `json:"providers"`
	// Allowed 为允许登录管理界面的身份，格式见 ParseEntry
	Allowed []string `json:"allowed"`
	// SkipTOTP 为真时第三方登录跳过两步验证（对应 Lucky 的“第三方登录跳过两步验证”）
	SkipTOTP bool `json:"skipTotp"`
}

// CallbackPath 是管理界面接收回调的路径。
const CallbackPath = "/api/login/oauth/callback"

// StartPath 是管理界面发起第三方登录的路径。
const StartPath = "/api/login/oauth/start"

// Normalize 清理空白项。
func (c *LoginConfig) Normalize() {
	c.CallbackBase = strings.TrimRight(strings.TrimSpace(c.CallbackBase), "/")
	var allowed []string
	for _, a := range c.Allowed {
		if a = strings.TrimSpace(a); a != "" {
			allowed = append(allowed, a)
		}
	}
	c.Allowed = allowed
}

// Validate 校验配置，old 为已保存的配置，用于还原被脱敏的 Client Secret。
func (c *LoginConfig) Validate(old LoginConfig) error {
	c.Normalize()
	oldSecrets := map[string]string{}
	for _, p := range old.Providers {
		oldSecrets[p.Key] = p.ClientSecret
	}
	keys := map[string]bool{}
	for i := range c.Providers {
		p := &c.Providers[i]
		p.Key = strings.ToLower(strings.TrimSpace(p.Key))
		crud.KeepSecret(oldSecrets[p.Key], &p.ClientSecret)
		if err := p.Validate(); err != nil {
			return err
		}
		if keys[p.Key] {
			return fmt.Errorf("第三方登录标识 %s 重复", p.Key)
		}
		keys[p.Key] = true
	}
	if c.CallbackBase != "" {
		if u, err := url.Parse(c.CallbackBase); err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.Path != "" {
			return errors.New("回调基础地址需为不带路径的 http(s) 地址，例如 https://gately.example.com")
		}
	}
	if err := ValidateEntries(c.Allowed); err != nil {
		return err
	}
	if c.Enable {
		if c.CallbackBase == "" || len(c.Providers) == 0 || len(c.Allowed) == 0 {
			return errors.New("启用第三方登录需要填写回调基础地址、至少一个登录方式和允许登录的身份")
		}
	}
	return nil
}

// Masked 返回 Client Secret 脱敏后的副本，用于 API 输出。
func (c LoginConfig) Masked() LoginConfig {
	c.Providers = append([]Provider{}, c.Providers...)
	for i := range c.Providers {
		c.Providers[i].ClientSecret = crud.Mask(c.Providers[i].ClientSecret)
	}
	return c
}

// Secrets 返回各登录方式的 Client Secret 明文，按登录方式标识（Key）索引，供管理界面“显示密码”使用。
func (c LoginConfig) Secrets() map[string]string {
	out := make(map[string]string, len(c.Providers))
	for _, p := range c.Providers {
		out[p.Key] = p.ClientSecret
	}
	return out
}

// ProviderMap 以 Key 索引提供方。
func (c LoginConfig) ProviderMap() map[string]Provider {
	m := make(map[string]Provider, len(c.Providers))
	for _, p := range c.Providers {
		m[p.Key] = p
	}
	return m
}

// CallbackURL 返回应在提供方登记的回调地址。
func (c LoginConfig) CallbackURL() string {
	return c.CallbackBase + CallbackPath
}

// ---- 带签名的短期令牌（OAuth state） ----

// Sign 生成带过期时间的签名数据。
func Sign(secret []byte, kind string, v any, ttl time.Duration) string {
	body, _ := json.Marshal(map[string]any{"k": kind, "v": v, "e": time.Now().Add(ttl).Unix()})
	p := base64.RawURLEncoding.EncodeToString(body)
	m := hmac.New(sha256.New, secret)
	m.Write([]byte(p))
	return p + "." + hex.EncodeToString(m.Sum(nil))
}

// Verify 校验签名、类型与有效期，并把内容解码到 out。
func Verify(secret []byte, kind, token string, out any) error {
	_, err := VerifyExp(secret, kind, token, out)
	return err
}

// VerifyExp 与 Verify 相同，另外返回过期时间（Unix 秒），用于推算签发时间。
func VerifyExp(secret []byte, kind, token string, out any) (int64, error) {
	p, sig, ok := strings.Cut(token, ".")
	if !ok {
		return 0, errors.New("格式错误")
	}
	m := hmac.New(sha256.New, secret)
	m.Write([]byte(p))
	if !hmac.Equal([]byte(sig), []byte(hex.EncodeToString(m.Sum(nil)))) {
		return 0, errors.New("签名无效")
	}
	raw, err := base64.RawURLEncoding.DecodeString(p)
	if err != nil {
		return 0, err
	}
	var env struct {
		K string          `json:"k"`
		V json.RawMessage `json:"v"`
		E int64           `json:"e"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return 0, err
	}
	if env.K != kind {
		return 0, errors.New("类型不匹配")
	}
	if time.Now().Unix() > env.E {
		return 0, errors.New("已过期，请重新登录")
	}
	return env.E, json.Unmarshal(env.V, out)
}

// RandomNonce 生成随机串，用于把登录流程绑定到发起登录的浏览器。
func RandomNonce() string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
