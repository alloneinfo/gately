package webservice

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// userBackend 返回 名称|X-Remote-User 请求头，便于同时检查分到哪个后端与传递的用户名。
func userBackend(t *testing.T, name string) *httptest.Server {
	t.Helper()
	b := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "max-age=60")
		_, _ = w.Write([]byte(name + "|" + r.Header.Get("X-Remote-User")))
	}))
	t.Cleanup(b.Close)
	return b
}

func basicHeader(user, pass string) map[string]string {
	return map[string]string{"Authorization": "Basic " + base64.StdEncoding.EncodeToString([]byte(user+":"+pass))}
}

// TestUserRoutesBasicAuth 覆盖基本认证下按用户分流、与分路径规则的优先级、用户名请求头与防伪造、公开路径。
func TestUserRoutesBasicAuth(t *testing.T) {
	main, admin, vip, api := userBackend(t, "main"), userBackend(t, "admin"), userBackend(t, "vip"), userBackend(t, "api")
	l := Listener{Sites: []Site{{
		Enable: true, Domains: []string{"app.test"}, Targets: []string{main.URL},
		BasicAuth: true, Users: []string{"admin:pw", "vip_user:pw", "bob:pw"},
		AuthOptional: true, Rules: []PathRule{{Prefix: "/", Auth: RuleAuthOn}, {Prefix: "/public/", Auth: RuleAuthOff}},
		Locations: []Location{
			{Prefix: "/api/", Targets: []string{api.URL}},
			// 只追加请求头的路径沿用后端，按用户分流时应改用该用户的后端
			{Prefix: "/hdr/", RequestHeaders: []string{"X-Loc: 1"}},
		},
		UserRoutes: []UserRoute{{Users: []string{"admin"}, Targets: []string{admin.URL}}, {Users: []string{"vip_user"}, Targets: []string{vip.URL}}},
		UserHeader: "x-remote-user",
	}}}
	addr, _ := startRunner(t, l, nil, nil)
	cases := []struct {
		user, path, want string
	}{
		{"admin", "/", "admin|admin"},
		{"vip_user", "/", "vip|vip_user"},
		{"bob", "/", "main|bob"},
		{"admin", "/api/x", "api|admin"},
		{"admin", "/hdr/x", "admin|admin"},
		{"bob", "/hdr/x", "main|bob"},
	}
	for _, c := range cases {
		hdr := basicHeader(c.user, "pw")
		hdr["X-Remote-User"] = "mallory"
		if _, body := get(t, http.DefaultClient, addr, "app.test", c.path, hdr); body != c.want {
			t.Fatalf("%s 访问 %s 得到 %q，期望 %q", c.user, c.path, body, c.want)
		}
	}
	// 公开路径的未登录访客使用站点后端，伪造的用户名头被删除
	if _, body := get(t, http.DefaultClient, addr, "app.test", "/public/a", map[string]string{"X-Remote-User": "admin"}); body != "main|" {
		t.Fatalf("公开路径得到 %q", body)
	}
	// 密码错误仍需认证
	if resp, _ := get(t, http.DefaultClient, addr, "app.test", "/", basicHeader("admin", "bad")); resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("密码错误应 401，实际 %d", resp.StatusCode)
	}
}

// TestUserRoutesWebSession 覆盖网页登录会话（含第三方登录的别名）按用户分流，以及公开路径上已登录访客的分流。
func TestUserRoutesWebSession(t *testing.T) {
	main, admin := userBackend(t, "main"), userBackend(t, "admin")
	l := Listener{Sites: []Site{{
		Enable: true, Domains: []string{"web.test"}, Targets: []string{main.URL},
		WebAuth: true, OAuthUsers: []string{"github:alice"},
		AuthOptional: true, Rules: []PathRule{{Prefix: "/private/", Auth: RuleAuthOn}},
		UserRoutes: []UserRoute{{Users: []string{"github:alice"}, Targets: []string{admin.URL}}},
	}}}
	addr, rn := startRunner(t, l, nil, nil)
	h := rn.route("web.test")
	// 第三方登录会话的主标识为稳定 ID，用户名作为别名
	req := httptest.NewRequest("GET", "http://web.test/", nil)
	rec := httptest.NewRecorder()
	h.auth.issueSession(rec, req, h.site, "github:1001", "alice")
	cookie := rec.Result().Cookies()[0]
	hdr := map[string]string{"Cookie": cookie.Name + "=" + cookie.Value}
	for _, path := range []string{"/private/x", "/"} {
		if _, body := get(t, noRedirect(), addr, "web.test", path, hdr); !strings.HasPrefix(body, "admin|") {
			t.Fatalf("已登录访问 %s 得到 %q，期望分到 admin", path, body)
		}
	}
	if _, body := get(t, noRedirect(), addr, "web.test", "/", nil); !strings.HasPrefix(body, "main|") {
		t.Fatalf("未登录访问公开路径得到 %q", body)
	}
}

// TestUserRoutesForwardAuth 覆盖转发认证返回的 Remote-User 参与分流，访客伪造的不生效。
func TestUserRoutesForwardAuth(t *testing.T) {
	var seen http.Header
	authz := fakeAuthelia(t, &seen)
	main, admin := userBackend(t, "main"), userBackend(t, "admin")
	l := Listener{Sites: []Site{{
		Enable: true, Domains: []string{"fa.test"}, Targets: []string{main.URL},
		ForwardAuth: ForwardAuth{URL: authz.URL},
		UserRoutes:  []UserRoute{{Users: []string{"alice"}, Targets: []string{admin.URL}}, {Users: []string{"mallory"}, Targets: []string{main.URL}}},
	}}}
	addr, _ := startRunner(t, l, nil, nil)
	if _, body := get(t, noRedirect(), addr, "fa.test", "/", map[string]string{"Cookie": "session=ok", "Remote-User": "mallory"}); !strings.HasPrefix(body, "admin|") {
		t.Fatalf("转发认证用户 alice 应分到 admin，得到 %q", body)
	}
}

func TestValidateUserRoutes(t *testing.T) {
	base := func() Site {
		return Site{Domains: []string{"a.test"}, Targets: []string{"http://127.0.0.1:1"}, BasicAuth: true, Users: []string{"a:b"}}
	}
	ok := base()
	ok.UserRoutes = []UserRoute{{Users: []string{" a ", ""}, Targets: []string{"http://127.0.0.1:2"}}}
	ok.UserHeader = " x-remote-user "
	if err := ok.validate(false); err != nil {
		t.Fatal(err)
	}
	if ok.UserHeader != "X-Remote-User" || len(ok.UserRoutes[0].Users) != 1 || ok.UserRoutes[0].Users[0] != "a" {
		t.Fatalf("整理结果错误: %+v", ok)
	}
	bad := map[string]func(s *Site){
		"未启用认证": func(s *Site) {
			s.BasicAuth = false
			s.UserRoutes = []UserRoute{{Users: []string{"a"}, Targets: []string{"http://x"}}}
		},
		"缺少用户": func(s *Site) { s.UserRoutes = []UserRoute{{Targets: []string{"http://x"}}} },
		"缺少后端": func(s *Site) { s.UserRoutes = []UserRoute{{Users: []string{"a"}}} },
		"非法后端": func(s *Site) { s.UserRoutes = []UserRoute{{Users: []string{"a"}, Targets: []string{"ftp://x"}}} },
		"用户重复": func(s *Site) {
			s.UserRoutes = []UserRoute{{Users: []string{"a"}, Targets: []string{"http://x"}}, {Users: []string{"a"}, Targets: []string{"http://y"}}}
		},
		"非法请求头": func(s *Site) { s.UserHeader = "X User" },
		"禁用请求头": func(s *Site) { s.UserHeader = "host" },
		"非反向代理": func(s *Site) {
			s.Type, s.Root = TypeStatic, "/tmp"
			s.UserHeader = "X-User"
		},
	}
	for name, mod := range bad {
		s := base()
		mod(&s)
		if err := s.validate(false); err == nil {
			t.Errorf("%s 应校验失败", name)
		}
	}
}

// TestUserRouteCacheKey 覆盖响应缓存键按分流规则与传递的用户名区分。
// 带 Cookie 或 Authorization 的请求本就不缓存，这里防的是转发认证按其他方式（如来源 IP）放行的情况
func TestUserRouteCacheKey(t *testing.T) {
	h := &siteHandler{site: Site{UserHeader: "X-Remote-User"}, userRoutes: []*userRouteHandler{{users: map[string]bool{"admin": true}}}}
	key := func(names ...string) string {
		r := h.withIdentity(httptest.NewRequest("GET", "http://a.test/x", nil), names)
		return cacheKey(r)
	}
	if key("admin") == key("bob") || key("bob") == key("carol") || key() == key("bob") {
		t.Fatal("不同用户的缓存键应不同")
	}
	h.site.UserHeader = ""
	if key("bob") != key("carol") || key("admin") == key("bob") {
		t.Fatal("未传递用户名时缓存键只按分流规则区分")
	}
}
