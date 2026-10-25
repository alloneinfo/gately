package webdav

import (
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/wangwanjie/gately/internal/config"
	"github.com/wangwanjie/gately/internal/fileusers"
	"github.com/wangwanjie/gately/internal/logbuf"
	"github.com/wangwanjie/gately/internal/module"
	"github.com/wangwanjie/gately/internal/mountfs"
)

// TestWebDAVSharedUsers 验证开启共享用户后共享用户按自己的目录登录，本地同名用户优先，修改共享用户立即生效，关闭开关后无法登录。
func TestWebDAVSharedUsers(t *testing.T) {
	alice, pub, local := t.TempDir(), t.TempDir(), t.TempDir()
	_ = os.WriteFile(filepath.Join(pub, "pub.txt"), []byte("pub"), 0o644)
	ln, _ := net.Listen("tcp4", "127.0.0.1:0")
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()
	st, _ := config.Open(t.TempDir())
	m := New(nil)
	if err := m.Init(module.Env{Store: st, Logs: logbuf.New(50, log.New(io.Discard, "", 0))}); err != nil {
		t.Fatal(err)
	}
	if err := fileusers.Default.Save([]fileusers.User{
		{Name: "alice", Password: "pa", Root: alice},
		{Name: "bob", Password: "pb", Mounts: []mountfs.Mount{{Name: "pub", Path: pub, ReadOnly: true}}},
		{Name: "dup", Password: "shared", Root: pub},
	}); err != nil {
		t.Fatal(err)
	}
	m.settings = Settings{Enable: true, ListenIP: "127.0.0.1", Port: port, SharedUsers: true,
		Users: []User{{Name: "dup", Password: "local", Root: local}}}
	if err := m.settings.Validate(); err != nil {
		t.Fatal(err)
	}
	m.restartLocked()
	defer m.Stop()
	base := "http://127.0.0.1:" + strconv.Itoa(port)
	do := func(method, path, user, pass, body string) int {
		req, _ := http.NewRequest(method, base+path, strings.NewReader(body))
		req.SetBasicAuth(user, pass)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		return resp.StatusCode
	}
	if c := do("PUT", "/a.txt", "alice", "pa", "a"); c != 201 {
		t.Fatalf("共享用户 alice 写入失败: %d", c)
	}
	if _, err := os.Stat(filepath.Join(alice, "a.txt")); err != nil {
		t.Fatal("应写入 alice 的目录")
	}
	if c := do("GET", "/", "alice", "bad", ""); c != 401 {
		t.Fatalf("错误密码应 401: %d", c)
	}
	if c := do("GET", "/pub/pub.txt", "bob", "pb", ""); c != 200 {
		t.Fatalf("bob 应能读取多目录: %d", c)
	}
	if c := do("PUT", "/pub/x.txt", "bob", "pb", "x"); c == 201 {
		t.Fatal("只读目录不应能写入")
	}
	// 与本地用户同名时只认本地用户
	if c := do("GET", "/", "dup", "shared", ""); c != 401 {
		t.Fatalf("同名时共享用户密码不应通过: %d", c)
	}
	if c := do("PUT", "/l.txt", "dup", "local", "l"); c != 201 {
		t.Fatalf("本地用户应正常: %d", c)
	}

	// 修改共享用户立即生效：alice 改密码并换目录，旧密码失效，新目录生效；删除 bob 后无法登录
	newDir := t.TempDir()
	if err := fileusers.Default.Save([]fileusers.User{{Name: "alice", Password: "pa2", Root: newDir}}); err != nil {
		t.Fatal(err)
	}
	if c := do("GET", "/", "alice", "pa", ""); c != 401 {
		t.Fatalf("旧密码应失效: %d", c)
	}
	if c := do("PUT", "/n.txt", "alice", "pa2", "n"); c != 201 {
		t.Fatalf("新密码应可写入: %d", c)
	}
	if _, err := os.Stat(filepath.Join(newDir, "n.txt")); err != nil {
		t.Fatal("应写入新目录")
	}
	if c := do("GET", "/", "bob", "pb", ""); c != 401 {
		t.Fatalf("删除的共享用户应无法登录: %d", c)
	}

	// 关闭开关后共享用户无法登录
	m.mu.Lock()
	m.settings.SharedUsers = false
	m.restartLocked()
	m.mu.Unlock()
	if c := do("GET", "/", "alice", "pa2", ""); c != 401 {
		t.Fatalf("关闭开关后应拒绝: %d", c)
	}
}

func TestSharedUsersEnableValidate(t *testing.T) {
	st, _ := config.Open(t.TempDir())
	if err := fileusers.Default.Init(st); err != nil {
		t.Fatal(err)
	}
	s := Settings{Enable: true, SharedUsers: true}
	if err := s.Validate(); err == nil {
		t.Fatal("没有本地用户也没有共享用户时应报错")
	}
	if err := fileusers.Default.Save([]fileusers.User{{Name: "a", Password: "p", Root: "/data"}}); err != nil {
		t.Fatal(err)
	}
	if err := s.Validate(); err != nil {
		t.Fatalf("有共享用户时可以只用共享用户启用: %v", err)
	}
	s.SharedUsers = false
	if err := s.Validate(); err == nil {
		t.Fatal("关闭共享用户且没有本地用户时应报错")
	}
}

// TestWebDAVSharedUsersKeepOne 验证只靠共享用户启用 WebDAV 时不允许删光共享用户，有本地用户时可以。
func TestWebDAVSharedUsersKeepOne(t *testing.T) {
	st, _ := config.Open(t.TempDir())
	m := New(nil)
	if err := m.Init(module.Env{Store: st, Logs: logbuf.New(50, log.New(io.Discard, "", 0))}); err != nil {
		t.Fatal(err)
	}
	if err := fileusers.Default.Save([]fileusers.User{{Name: "alice", Password: "pa", Root: t.TempDir()}}); err != nil {
		t.Fatal(err)
	}
	m.mu.Lock()
	m.settings = Settings{Enable: true, SharedUsers: true}
	m.mu.Unlock()
	if err := fileusers.Default.Save(nil); err == nil {
		t.Fatal("只靠共享用户启用时不应允许删光共享用户")
	}
	m.mu.Lock()
	m.settings.Users = []User{{Name: "local", Password: "lp", Root: t.TempDir()}}
	m.mu.Unlock()
	if err := fileusers.Default.Save(nil); err != nil {
		t.Fatalf("有本地用户时应允许删光共享用户: %v", err)
	}
}
