package auth

import (
	"errors"
	"strings"
	"testing"
	"time"
)

// RFC 6238 附录 B 的 SHA1 测试向量（密钥为 ASCII "12345678901234567890"，取 8 位中的后 6 位）
func TestTOTPRFCVector(t *testing.T) {
	secret := b32.EncodeToString([]byte("12345678901234567890"))
	cases := map[int64]string{59: "287082", 1111111109: "081804", 1234567890: "005924", 2000000000: "279037"}
	for ts, want := range cases {
		got, err := totpCode(secret, ts/totpPeriod)
		if err != nil || got != want {
			t.Errorf("t=%d got %s want %s", ts, got, want)
		}
		if !ValidateTOTP(secret, want, time.Unix(ts, 0)) {
			t.Errorf("t=%d 校验失败", ts)
		}
	}
	if ValidateTOTP(secret, "000000", time.Unix(59, 0)) {
		t.Error("错误验证码不应通过")
	}
	if !strings.HasPrefix(TOTPURL(secret, "admin"), "otpauth://totp/Gately:admin?") {
		t.Error("otpauth 地址格式错误")
	}
}

func TestLoginWithTOTP(t *testing.T) {
	m, pass := newTestManager(t)
	secret := NewTOTPSecret()
	if err := m.SetTOTP("wrong", secret); err == nil {
		t.Fatal("密码错误时不应启用")
	}
	if err := m.SetTOTP(pass, secret); err != nil {
		t.Fatal(err)
	}
	if _, _, err := m.Login("1.1.1.1", "admin", pass); !errors.Is(err, ErrTOTPRequired) {
		t.Fatalf("应要求验证码: %v", err)
	}
	if _, _, err := m.Login("1.1.1.1", "admin", pass, "123456"); err == nil {
		t.Fatal("错误验证码应失败")
	}
	code, _ := totpCode(secret, time.Now().Unix()/totpPeriod)
	if _, _, err := m.Login("1.1.1.1", "admin", pass, code); err != nil {
		t.Fatal(err)
	}
	if err := m.SetTOTP(pass, ""); err != nil || m.TOTPEnabled() {
		t.Fatal("关闭两步验证失败")
	}
}
