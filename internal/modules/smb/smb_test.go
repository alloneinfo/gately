package smb

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/wangwanjie/gately/internal/config"
	"github.com/wangwanjie/gately/internal/crud"
	"github.com/wangwanjie/gately/internal/fileusers"
	"github.com/wangwanjie/gately/internal/logbuf"
	"github.com/wangwanjie/gately/internal/module"
	"github.com/wangwanjie/gately/internal/mountfs"
)

func TestSettingsValidate(t *testing.T) {
	s := Settings{}
	if err := s.Validate(); err != nil || s.Port != DefaultPort {
		t.Fatalf("默认值不正确: %v %+v", err, s)
	}
	cases := []struct {
		name string
		s    Settings
		want string
	}{
		{"启用但无用户", Settings{Enable: true}, "至少添加一个用户"},
		{"端口越界", Settings{Port: 70000}, "端口无效"},
		{"监听地址", Settings{ListenIP: "abc"}, "监听地址无效"},
		{"空密码", Settings{Users: []User{{Name: "a", Root: "/data"}}}, "不能为空"},
		{"相对路径", Settings{Users: []User{{Name: "a", Password: "p", Root: "data"}}}, "绝对路径"},
		{"用户名非法字符", Settings{Users: []User{{Name: `a\b`, Password: "p", Root: "/data"}}}, "无效"},
		{"用户名保留", Settings{Users: []User{{Name: "IPC$", Password: "p", Root: "/data"}}}, "无效"},
		{"用户名大小写重复", Settings{Users: []User{
			{Name: "Alice", Password: "p", Root: "/data"},
			{Name: "alice", Password: "p", Root: "/data"},
		}}, "重复"},
		{"bcrypt 密码", Settings{Users: []User{{Name: "a", Password: "$2a$10$" + strings.Repeat("x", 53), Root: "/data"}}}, "bcrypt"},
		{"共享名非法字符", Settings{Users: []User{{Name: "a", Password: "p", Mounts: []mountfs.Mount{{Name: "x:y", Path: "/data"}}}}}, "共享名"},
		{"共享名大小写重复", Settings{Users: []User{{Name: "a", Password: "p", Mounts: []mountfs.Mount{
			{Name: "Media", Path: "/data/a"},
			{Name: "media", Path: "/data/b"},
		}}}}, "重复"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := c.s.Validate()
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("期望包含 %q 的错误，得到 %v", c.want, err)
			}
		})
	}

	// 多目录模式会清空单目录字段
	s = Settings{Users: []User{{Name: "a", Password: "p", Root: "/x", ReadOnly: true, Mounts: []mountfs.Mount{{Name: "m", Path: "/data"}}}}}
	if err := s.Validate(); err != nil || s.Users[0].Root != "" || s.Users[0].ReadOnly {
		t.Fatalf("多目录规范化不正确: %v %+v", err, s.Users[0])
	}
}

func TestUserShares(t *testing.T) {
	u := User{Name: "alice", Root: "/data", ReadOnly: true}
	if got := u.Shares(); len(got) != 1 || got[0] != (Share{Name: "alice", Path: "/data", ReadOnly: true}) {
		t.Fatalf("单目录共享不正确: %+v", got)
	}
	u.Mounts = []mountfs.Mount{{Name: "a", Path: "/x"}, {Name: "b", Path: "/y", ReadOnly: true}}
	if got := u.Shares(); len(got) != 2 || got[1] != (Share{Name: "b", Path: "/y", ReadOnly: true}) {
		t.Fatalf("多目录共享不正确: %+v", got)
	}
}

func TestValidName(t *testing.T) {
	for _, n := range []string{"media", "照片", "my share", "a-b_c.d", "x$"} {
		if !validName(n) {
			t.Errorf("%q 应合法", n)
		}
	}
	for _, n := range []string{"", "a/b", "a*b", "ipc$", ".hidden", "end.", "a\tb", strings.Repeat("x", 81)} {
		if validName(n) {
			t.Errorf("%q 应非法", n)
		}
	}
}

// TestKeepPasswords 验证提交占位符密码时恢复原密码：按去空白后的用户名匹配，改名时按位置匹配。
func TestKeepPasswords(t *testing.T) {
	old := []User{{Name: "alice", Password: "pa"}, {Name: "bob", Password: "pb"}}
	mask := crud.SecretMask

	// 用户名带空白也能匹配
	got := []User{{Name: " alice ", Password: mask}, {Name: "bob", Password: "new"}}
	keepPasswords(old, got)
	if got[0].Password != "pa" || got[1].Password != "new" {
		t.Fatalf("按名称恢复失败: %+v", got)
	}

	// 改名：同一位置的原用户已不在新列表中，沿用其密码
	got = []User{{Name: "alice", Password: mask}, {Name: "carol", Password: mask}}
	keepPasswords(old, got)
	if got[0].Password != "pa" || got[1].Password != "pb" {
		t.Fatalf("改名后应保留原密码: %+v", got)
	}

	// 调换顺序：按名称匹配，不按位置
	got = []User{{Name: "bob", Password: mask}, {Name: "alice", Password: mask}}
	keepPasswords(old, got)
	if got[0].Password != "pb" || got[1].Password != "pa" {
		t.Fatalf("调换顺序后密码错位: %+v", got)
	}

	// 删除一个用户后另一个改名：位置已错开，不能拿到被删用户的密码，清空后由校验报错
	got = []User{{Name: "dave", Password: mask}}
	keepPasswords(old, got)
	if got[0].Password != "" {
		t.Fatalf("无法确定原用户时不应沿用其他用户的密码: %+v", got)
	}

	// 新增用户提交占位符：没有原密码，清空
	got = []User{{Name: "alice", Password: mask}, {Name: "bob", Password: mask}, {Name: "eve", Password: mask}}
	keepPasswords(old, got)
	if got[2].Password != "" {
		t.Fatalf("新用户不应得到占位符密码: %+v", got)
	}
}

type muxRouter struct{ *http.ServeMux }

func (r muxRouter) HandleFunc(p string, h http.HandlerFunc) { r.ServeMux.HandleFunc(p, h) }

// TestSharedUsersUsable 验证只靠共享用户启用 SMB 时按实际能用于 SMB 的账号计算：
// 共享用户全部不能用于 SMB 时不能启用，也不能把共享用户改成全部不可用；与本地用户同名的共享用户不列为问题。
func TestSharedUsersUsable(t *testing.T) {
	st, _ := config.Open(t.TempDir())
	m := New()
	if err := m.Init(module.Env{Store: st, Logs: logbuf.New(20, log.New(io.Discard, "", 0))}); err != nil {
		t.Fatal(err)
	}
	if err := fileusers.Default.Save([]fileusers.User{{Name: "bad*name", Password: "p", Root: "/a"}}); err != nil {
		t.Fatal(err)
	}
	s := Settings{Enable: true, SharedUsers: true}
	if err := s.Validate(); err == nil {
		t.Fatal("共享用户都不能用于 SMB 时不应允许启用")
	}
	if err := fileusers.Default.Save([]fileusers.User{{Name: "alice", Password: "p", Root: "/a"}, {Name: "Local", Password: "p", Root: "/a"}}); err != nil {
		t.Fatal(err)
	}
	if err := s.Validate(); err != nil {
		t.Fatal(err)
	}
	// 不启动服务，只设置配置，验证保存前的检查
	m.mu.Lock()
	m.settings = s
	m.mu.Unlock()
	if err := fileusers.Default.Save([]fileusers.User{{Name: "bad*name", Password: "p", Root: "/a"}}); err == nil {
		t.Fatal("只靠共享用户启用 SMB 时不应允许把共享用户改成都不可用")
	}

	m.mu.Lock()
	m.settings = Settings{Users: []User{{Name: "local", Password: "lp", Root: "/l"}}}
	m.mu.Unlock()
	if err := fileusers.Default.Save([]fileusers.User{{Name: "bad*name", Password: "p", Root: "/a"}, {Name: "LOCAL", Password: "p", Root: "/a"}}); err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	m.Routes(muxRouter{mux})
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("GET", "/shared-users", nil))
	var out struct {
		Issues []Issue `json:"issues"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if len(out.Issues) != 1 || out.Issues[0].Name != "bad*name" {
		t.Fatalf("只应列出不能用于 SMB 的共享用户，与本地用户同名的不列出: %s", rec.Body.String())
	}
}

// TestSettingsSecrets 验证查看本地用户密码明文的接口按用户名返回原始密码，而设置接口仍返回占位符。
func TestSettingsSecrets(t *testing.T) {
	st, _ := config.Open(t.TempDir())
	m := New()
	if err := m.Init(module.Env{Store: st, Logs: logbuf.New(20, log.New(io.Discard, "", 0))}); err != nil {
		t.Fatal(err)
	}
	m.mu.Lock()
	m.settings = Settings{Users: []User{{Name: "alice", Password: "p1", Root: "/a"}, {Name: "bob", Password: "p2", Root: "/b"}}}
	m.mu.Unlock()
	mux := http.NewServeMux()
	m.Routes(muxRouter{mux})
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("GET", "/settings/secrets", nil))
	var out map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusOK || len(out) != 2 || out["alice"] != "p1" || out["bob"] != "p2" {
		t.Fatalf("应按用户名返回密码明文: %d %s", rec.Code, rec.Body.String())
	}
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("GET", "/settings", nil))
	if strings.Contains(rec.Body.String(), "p1") {
		t.Fatalf("设置接口不应返回密码明文: %s", rec.Body.String())
	}
}
