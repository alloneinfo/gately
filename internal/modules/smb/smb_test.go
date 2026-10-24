package smb

import (
	"strings"
	"testing"

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
