package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/subtle"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"net/url"
	"strings"
	"time"
)

// TOTP 参数（RFC 6238，与常见验证器 App 兼容）
const (
	totpPeriod = 30
	totpDigits = 6
	// 允许前后各一个时间窗口，容忍时钟偏差
	totpSkew = 1
)

var b32 = base32.StdEncoding.WithPadding(base32.NoPadding)

// NewTOTPSecret 生成 160 位随机密钥（Base32 编码）。
func NewTOTPSecret() string {
	b := make([]byte, 20)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return b32.EncodeToString(b)
}

// totpCode 计算指定时间步的验证码。
func totpCode(secret string, step int64) (string, error) {
	key, err := b32.DecodeString(strings.ToUpper(strings.ReplaceAll(secret, " ", "")))
	if err != nil {
		return "", err
	}
	var msg [8]byte
	binary.BigEndian.PutUint64(msg[:], uint64(step))
	m := hmac.New(sha1.New, key)
	m.Write(msg[:])
	sum := m.Sum(nil)
	off := sum[len(sum)-1] & 0x0f
	v := binary.BigEndian.Uint32(sum[off:off+4]) & 0x7fffffff
	return fmt.Sprintf("%06d", v%1000000), nil
}

// ValidateTOTP 校验验证码（不防重放，用于绑定时确认验证器）。
func ValidateTOTP(secret, code string, now time.Time) bool {
	_, ok := matchTOTP(secret, code, now)
	return ok
}

// matchTOTP 校验验证码并返回匹配的时间步。
func matchTOTP(secret, code string, now time.Time) (int64, bool) {
	code = strings.TrimSpace(code)
	if len(code) != totpDigits {
		return 0, false
	}
	step := now.Unix() / totpPeriod
	for d := -totpSkew; d <= totpSkew; d++ {
		want, err := totpCode(secret, step+int64(d))
		if err != nil {
			return 0, false
		}
		if subtle.ConstantTimeCompare([]byte(want), []byte(code)) == 1 {
			return step + int64(d), true
		}
	}
	return 0, false
}

// TOTPURL 生成验证器 App 可扫描的 otpauth 地址。
func TOTPURL(secret, account string) string {
	v := url.Values{}
	v.Set("secret", secret)
	v.Set("issuer", "Gately")
	v.Set("digits", "6")
	v.Set("period", "30")
	return "otpauth://totp/" + url.PathEscape("Gately:"+account) + "?" + v.Encode()
}
