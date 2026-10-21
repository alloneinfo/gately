package auth

import (
	"testing"
)

func TestDefaultPasswordFlag(t *testing.T) {
	m, pass := newTestManager(t)
	if pass != DefaultPassword || !m.UsingDefaultPassword() {
		t.Fatalf("全新安装应使用默认密码并标记: %q %v", pass, m.UsingDefaultPassword())
	}
	if _, _, err := m.Login("1.1.1.1", "admin", DefaultPassword); err != nil {
		t.Fatal(err)
	}
	if err := m.ChangeCredentials(DefaultPassword, "admin", "new-password-1"); err != nil {
		t.Fatal(err)
	}
	if m.UsingDefaultPassword() {
		t.Fatal("修改密码后应清除默认密码标记")
	}

	// 指定初始密码时不返回也不标记
	m2 := NewManager(Account{}, func(Account) error { return nil })
	if err := m2.SetBootstrapPassword("my-initial-pw"); err != nil {
		t.Fatal(err)
	}
	if p, err := m2.EnsureInitialized(); err != nil || p != "" || m2.UsingDefaultPassword() {
		t.Fatalf("指定初始密码: %q %v %v", p, err, m2.UsingDefaultPassword())
	}
}

func TestResetRequestRoundTrip(t *testing.T) {
	dir := t.TempDir()
	m, _ := newTestManager(t)
	if err := m.ChangeCredentials(DefaultPassword, "root", "forgotten-pw"); err != nil {
		t.Fatal(err)
	}
	if err := m.SetTOTP("forgotten-pw", "", NewTOTPSecret()); err != nil {
		t.Fatal(err)
	}
	old, _, err := m.Login("2.2.2.2", "root", "forgotten-pw", "000000")
	if err == nil || old != "" {
		t.Fatal("启用两步验证后错误验证码不应登录成功")
	}
	tok, _ := m.Issue()

	// 先后提交重置密码与关闭两步验证，两个请求应合并
	req, err := NewPasswordReset("")
	if err != nil {
		t.Fatal(err)
	}
	if err := WriteResetRequest(dir, req); err != nil {
		t.Fatal(err)
	}
	if err := WriteResetRequest(dir, ResetRequest{DisableTOTP: true}); err != nil {
		t.Fatal(err)
	}
	if !ResetPending(dir) {
		t.Fatal("应有待处理的请求")
	}
	got, err := TakeResetRequest(dir)
	if err != nil || got == nil {
		t.Fatalf("读取请求: %v %v", got, err)
	}
	if ResetPending(dir) {
		t.Fatal("读取后应删除请求文件")
	}
	if !got.DisableTOTP || got.PasswordHash == "" || !got.DefaultPassword {
		t.Fatalf("请求未合并: %+v", got)
	}

	// 先把该 IP 锁定，重置后应解除
	for range 6 {
		_, _, _ = m.Login("3.3.3.3", "root", "wrong")
	}
	if err := m.ApplyReset(*got); err != nil {
		t.Fatal(err)
	}
	if _, ok := m.Verify(tok); ok {
		t.Fatal("重置后旧会话应失效")
	}
	if m.TOTPEnabled() || !m.UsingDefaultPassword() {
		t.Fatal("应关闭两步验证并标记默认密码")
	}
	if _, _, err := m.Login("3.3.3.3", "root", DefaultPassword); err != nil {
		t.Fatalf("重置后应能用默认密码直接登录（用户名不变）: %v", err)
	}

	if req, err := TakeResetRequest(dir); req != nil || err != nil {
		t.Fatalf("没有请求时应返回 nil: %v %v", req, err)
	}
	if err := m.ApplyReset(ResetRequest{PasswordHash: "plain"}); err == nil {
		t.Fatal("非 bcrypt 哈希应拒绝")
	}
	if _, err := NewPasswordReset("short"); err == nil {
		t.Fatal("过短的新密码应拒绝")
	}
}
