//go:build linux || darwin

package smb

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"github.com/wangwanjie/gately/internal/fileusers"
	"github.com/wangwanjie/gately/internal/mountfs"
)

// TestSharedUsersLogin 验证开启共享用户后共享用户可登录 SMB，共享名按其目录配置生成；
// 与本地用户同名（不区分大小写）、bcrypt 密码或用户名不合法的共享用户无法登录；修改共享用户后服务自动重新加载。
func TestSharedUsersLogin(t *testing.T) {
	alice, team, pub := t.TempDir(), t.TempDir(), t.TempDir()
	_ = os.WriteFile(filepath.Join(pub, "pub.txt"), []byte("pub"), 0o644)
	h, _ := bcrypt.GenerateFromPassword([]byte("ph"), bcrypt.MinCost)
	// startTestModule 内会初始化共享用户的配置目录，这里先启动再保存共享用户
	m, port := startTestModule(t, Settings{
		Users:       []User{{Name: "local", Password: "lp", Root: t.TempDir()}},
		SharedUsers: true,
	})
	if err := fileusers.Default.Save([]fileusers.User{
		{Name: "alice", Password: "pa", Root: alice},
		{Name: "bob", Password: "pb", Mounts: []mountfs.Mount{{Name: "team", Path: team}, {Name: "pub", Path: pub, ReadOnly: true}}},
		{Name: "hashed", Password: string(h), Root: pub},
		{Name: "bad*name", Password: "px", Root: pub},
		{Name: "LOCAL", Password: "shared", Root: pub},
	}); err != nil {
		t.Fatal(err)
	}
	sess, err := dial(t, port, "alice", "pa")
	if err != nil {
		t.Fatalf("alice 登录失败: %v", err)
	}
	if names, _ := sess.ListSharenames(); !slices.Contains(names, "alice") {
		t.Fatalf("单目录共享用户的共享名应为用户名: %v", names)
	}
	fs, err := sess.Mount("alice")
	if err != nil {
		t.Fatal(err)
	}
	if err := fs.WriteFile("a.txt", []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	fs.Umount()
	if _, err := os.Stat(filepath.Join(alice, "a.txt")); err != nil {
		t.Fatal("应写入 alice 的目录")
	}
	bob, err := dial(t, port, "bob", "pb")
	if err != nil {
		t.Fatalf("bob 登录失败: %v", err)
	}
	if names, _ := bob.ListSharenames(); !slices.Contains(names, "team") || !slices.Contains(names, "pub") {
		t.Fatalf("多目录共享用户的共享列表不符: %v", names)
	}
	pubFs, err := bob.Mount("pub")
	if err != nil {
		t.Fatal(err)
	}
	if err := pubFs.WriteFile("x.txt", []byte("x"), 0o644); err == nil {
		t.Fatal("只读共享不应能写入")
	}
	pubFs.Umount()
	for _, c := range []struct{ user, pass, why string }{
		{"alice", "bad", "错误密码"},
		{"hashed", "ph", "bcrypt 密码"},
		{"bad*name", "px", "用户名不合法"},
		{"local", "shared", "与本地用户同名"},
	} {
		if _, err := dial(t, port, c.user, c.pass); err == nil {
			t.Errorf("%s的共享用户 %s 不应能登录", c.why, c.user)
		}
	}
	if !hasLog(m, "共享用户 hashed 的密码是 bcrypt 哈希") || !hasLog(m, "共享用户 bad*name 的用户名不能用于 SMB") {
		t.Fatal("无法用于 SMB 的共享用户应写入运行日志")
	}
	if _, err := dial(t, port, "local", "lp"); err != nil {
		t.Fatalf("本地用户应正常登录: %v", err)
	}

	// 修改共享用户后服务同步重新加载：旧密码失效、新密码可用
	users := fileusers.Default.List()
	users[0].Password = "pa2"
	if err := fileusers.Default.Save(users); err != nil {
		t.Fatal(err)
	}
	if !hasLog(m, "共享用户已修改") {
		t.Fatal("应重新加载 SMB 服务")
	}
	if _, err := dial(t, port, "alice", "pa2"); err != nil {
		t.Fatalf("新密码应能登录: %v", err)
	}
	if _, err := dial(t, port, "alice", "pa"); err == nil {
		t.Fatal("旧密码应失效")
	}

	// 保存内容不影响 SMB 账号时不重启
	m.mu.Lock()
	srv := m.srv
	m.mu.Unlock()
	if err := fileusers.Default.Save(fileusers.Default.List()); err != nil {
		t.Fatal(err)
	}
	m.mu.Lock()
	same := m.srv == srv
	m.mu.Unlock()
	if !same {
		t.Fatal("账号未变化时不应重启服务")
	}
}

func TestBuildAccountsIssues(t *testing.T) {
	shared := []fileusers.User{
		{Name: "ok", Password: "p", Root: "/a"},
		{Name: "Local", Password: "p", Root: "/a"},
		{Name: "dupshare", Password: "p", Mounts: []mountfs.Mount{{Name: "Docs", Path: "/a"}, {Name: "docs", Path: "/b"}}},
	}
	s := Settings{Users: []User{{Name: "local", Password: "p", Root: "/l"}}}
	if accounts, issues := buildAccounts(s, shared); len(accounts) != 1 || issues != nil {
		t.Fatal("未开启共享用户时只有本地用户")
	}
	s.SharedUsers = true
	accounts, issues := buildAccounts(s, shared)
	if len(accounts) != 2 || accounts[1].Name != "ok" || !accounts[1].Shared {
		t.Fatalf("账号不符: %+v", accounts)
	}
	if len(issues) != 1 || issues[0].Name != "dupshare" || issues[0].Reason != IssueShare {
		t.Fatalf("问题列表不符: %+v", issues)
	}
}
