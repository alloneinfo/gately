package auth

import (
	"errors"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func newTestManager(t *testing.T) (*Manager, string) {
	t.Helper()
	m := NewManager(Account{}, func(Account) error { return nil })
	pass, err := m.EnsureInitialized()
	if err != nil {
		t.Fatal(err)
	}
	if pass == "" {
		t.Fatal("首次初始化应返回默认密码")
	}
	return m, pass
}

func TestLoginAndVerify(t *testing.T) {
	m, pass := newTestManager(t)
	token, _, err := m.Login("1.1.1.1", "admin", pass)
	if err != nil {
		t.Fatal(err)
	}
	if user, ok := m.Verify(token); !ok || user != "admin" {
		t.Fatalf("令牌校验失败: %v %v", user, ok)
	}
	if _, ok := m.Verify(token + "x"); ok {
		t.Fatal("篡改的令牌应无效")
	}
	// 过期
	m.now = func() time.Time { return time.Now().Add(100 * time.Hour) }
	if _, ok := m.Verify(token); ok {
		t.Fatal("过期令牌应无效")
	}
}

func TestLockoutAfterFailures(t *testing.T) {
	m, pass := newTestManager(t)
	for i := 0; i < maxFailures; i++ {
		if _, _, err := m.Login("2.2.2.2", "admin", "wrong"); !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("第 %d 次应为凭据错误: %v", i, err)
		}
	}
	if _, _, err := m.Login("2.2.2.2", "admin", pass); !errors.Is(err, ErrLocked) {
		t.Fatalf("应被锁定: %v", err)
	}
	// 其他 IP 不受影响
	if _, _, err := m.Login("3.3.3.3", "admin", pass); err != nil {
		t.Fatal(err)
	}
}

func TestChangeCredentialsInvalidatesTokens(t *testing.T) {
	m, pass := newTestManager(t)
	token, _, err := m.Login("1.1.1.1", "admin", pass)
	if err != nil {
		t.Fatal(err)
	}
	if err := m.ChangeCredentials("bad", "root", "new-password"); err == nil {
		t.Fatal("原密码错误时应拒绝")
	}
	if err := m.ChangeCredentials(pass, "root", "new-password"); err != nil {
		t.Fatal(err)
	}
	if _, ok := m.Verify(token); ok {
		t.Fatal("修改密码后旧令牌应失效")
	}
	if _, _, err := m.Login("1.1.1.1", "root", "new-password"); err != nil {
		t.Fatal(err)
	}
}

func TestConcurrentLoginLimited(t *testing.T) {
	m, _ := newTestManager(t)
	var wg sync.WaitGroup
	var attempts atomic.Int64
	for i := 0; i < 30; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, _, err := m.Login("2001:db8::"+strconv.Itoa(i+1), "admin", "wrong"); errors.Is(err, ErrInvalidCredentials) {
				attempts.Add(1)
			}
		}()
	}
	wg.Wait()
	// 同一 /64 内的并发请求共享计数，实际校验次数不超过上限
	if n := attempts.Load(); n > maxFailures {
		t.Fatalf("并发绕过了限流：实际尝试 %d 次", n)
	}
}
