package crud

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

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

// muxRouter 把 http.ServeMux 适配为 module.Router。
type muxRouter struct{ *http.ServeMux }

func (r muxRouter) HandleFunc(pattern string, h http.HandlerFunc) { r.ServeMux.HandleFunc(pattern, h) }

type secretItem struct {
	Key   string
	On    bool
	Token string
}

// 提供 Secrets 时注册查看明文接口，返回指定条目的明文；条目不存在时返回 404。
func TestCollectionSecrets(t *testing.T) {
	var mu sync.Mutex
	items := []secretItem{{Key: "a", Token: "t0k"}}
	c := &Collection[secretItem]{
		Mu:       &mu,
		Items:    &items,
		Save:     func() error { return nil },
		Key:      func(it *secretItem) *string { return &it.Key },
		Enabled:  func(it *secretItem) *bool { return &it.On },
		Validate: func(*secretItem) error { return nil },
		Secrets:  func(it *secretItem) map[string]string { return map[string]string{"token": it.Token} },
	}
	mux := http.NewServeMux()
	c.Register(muxRouter{mux}, "/items")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("GET", "/items/a/secrets", nil))
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), `"token":"t0k"`) {
		t.Fatalf("应返回明文: %d %s", rec.Code, rec.Body.String())
	}
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("GET", "/items/x/secrets", nil))
	if rec.Code != 404 {
		t.Fatalf("条目不存在应返回 404，实际 %d", rec.Code)
	}
	// 未提供 Secrets 时不注册该接口
	c.Secrets = nil
	mux = http.NewServeMux()
	c.Register(muxRouter{mux}, "/items")
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("GET", "/items/a/secrets", nil))
	if rec.Code != 404 && rec.Code != 405 {
		t.Fatalf("未提供 Secrets 时不应注册接口，实际 %d", rec.Code)
	}
}
