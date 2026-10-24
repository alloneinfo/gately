package crud

import "testing"

type secretUser struct {
	Name, Password string
}

func keep(old, updated []secretUser) []secretUser {
	KeepSecrets(old, updated, func(u *secretUser) string { return u.Name }, func(u *secretUser) *string { return &u.Password })
	return updated
}

// 按名称保留、改名时按位置保留、调换顺序与删除时不取错、新密码原样保存。
func TestKeepSecrets(t *testing.T) {
	old := []secretUser{{"alice", "a1"}, {"bob", "b1"}}
	got := keep(old, []secretUser{{" bob ", SecretMask}, {"alice", SecretMask}})
	if got[0].Password != "b1" || got[1].Password != "a1" {
		t.Fatalf("调换顺序后应按名称保留: %+v", got)
	}
	got = keep(old, []secretUser{{"alice2", SecretMask}, {"bob", "new"}})
	if got[0].Password != "a1" || got[1].Password != "new" {
		t.Fatalf("改名后应按位置保留原密码: %+v", got)
	}
	got = keep(old, []secretUser{{"carol", SecretMask}})
	if got[0].Password != "" {
		t.Fatalf("数量变化时不应按位置猜测: %+v", got)
	}
	got = keep(old, []secretUser{{"bob2", SecretMask}, {"alice", SecretMask}})
	if got[0].Password != "" || got[1].Password != "a1" {
		t.Fatalf("同位置原用户仍存在时不应取它的密码: %+v", got)
	}
}
