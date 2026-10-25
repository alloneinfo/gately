package fileusers

import (
	"testing"

	"golang.org/x/crypto/bcrypt"

	"github.com/wangwanjie/gately/internal/config"
	"github.com/wangwanjie/gately/internal/crud"
	"github.com/wangwanjie/gately/internal/mountfs"
)

func TestValidate(t *testing.T) {
	ok := []User{
		{Name: " alice ", Password: "p", Root: " /data/a "},
		{Name: "bob", Password: "p", Root: "/ignored", ReadOnly: true, Mounts: []mountfs.Mount{{Name: "m", Path: "/data/m"}}},
	}
	if err := Validate(ok); err != nil {
		t.Fatal(err)
	}
	if ok[0].Name != "alice" || ok[0].Root != "/data/a" || ok[1].Root != "" || ok[1].ReadOnly {
		t.Fatalf("规范化结果不符: %+v", ok)
	}
	bad := map[string][]User{
		"空用户名":   {{Name: " ", Password: "p", Root: "/a"}},
		"空密码":    {{Name: "a", Root: "/a"}},
		"含冒号":    {{Name: "a:b", Password: "p", Root: "/a"}},
		"大小写重复":  {{Name: "Ann", Password: "p", Root: "/a"}, {Name: "ann", Password: "p", Root: "/b"}},
		"相对路径":   {{Name: "a", Password: "p", Root: "data"}},
		"挂载名称无效": {{Name: "a", Password: "p", Mounts: []mountfs.Mount{{Name: "", Path: "/a"}}}},
	}
	for name, users := range bad {
		if err := Validate(users); err == nil {
			t.Errorf("%s 应校验失败", name)
		}
	}
}

func TestCheckPassword(t *testing.T) {
	h, _ := bcrypt.GenerateFromPassword([]byte("pw"), bcrypt.MinCost)
	if !CheckPassword("pw", "pw") || CheckPassword("pw", "bad") || !CheckPassword(string(h), "pw") || CheckPassword(string(h), "bad") {
		t.Fatal("密码校验结果错误")
	}
}

// TestSaveAndReload 验证保存时沿用脱敏占位符的原密码、写入 fileusers.json、通知订阅者，换配置目录时重新读取。
func TestSaveAndReload(t *testing.T) {
	dir := t.TempDir()
	st, _ := config.Open(dir)
	r := New()
	if err := r.Save(nil); err == nil {
		t.Fatal("未初始化时保存应失败")
	}
	if err := r.Init(st); err != nil {
		t.Fatal(err)
	}
	notified := 0
	cancel := r.Subscribe(func() { notified++ })
	if err := r.Save([]User{{Name: "alice", Password: "pa", Root: "/a"}, {Name: "bob", Password: "pb", Root: "/b"}}); err != nil {
		t.Fatal(err)
	}
	if notified != 1 {
		t.Fatalf("保存后应通知一次: %d", notified)
	}
	masked := r.Masked()
	if masked[0].Password != crud.SecretMask {
		t.Fatal("读取时密码应脱敏")
	}
	// 提交脱敏占位符表示不修改密码；alice 改名为 alice2 时按位置保留原密码
	masked[0].Name = "alice2"
	if err := r.Save(masked); err != nil {
		t.Fatal(err)
	}
	if u, ok := r.Lookup("alice2"); !ok || u.Password != "pa" {
		t.Fatalf("改名后应保留原密码: %+v", u)
	}
	if u, _ := r.Lookup("bob"); u.Password != "pb" {
		t.Fatal("未修改的密码应保留")
	}
	// 校验失败时不保存、不通知
	if err := r.Save([]User{{Name: "x", Password: "p", Root: "rel"}}); err == nil || notified != 2 || len(r.List()) != 2 {
		t.Fatalf("校验失败不应改动: %v %d", err, notified)
	}
	cancel()
	if err := r.Save(r.List()); err != nil || notified != 2 {
		t.Fatal("取消订阅后不应再通知")
	}

	// 同一配置目录再次 Init 不重新读取；新实例从文件读到保存的内容
	r2 := New()
	if err := r2.Init(st); err != nil {
		t.Fatal(err)
	}
	if u, ok := r2.Lookup("alice2"); !ok || u.Password != "pa" {
		t.Fatalf("应从 fileusers.json 读取: %+v", u)
	}
	other, _ := config.Open(t.TempDir())
	if err := r2.Init(other); err != nil || len(r2.List()) != 0 {
		t.Fatal("换配置目录后应重新读取")
	}
}
