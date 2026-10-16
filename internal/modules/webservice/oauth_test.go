package webservice

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/wangwanjie/gately/internal/logbuf"
	"github.com/wangwanjie/gately/internal/thirdauth"
)

// fakeIdP 模拟 GitHub 与 OIDC 身份提供方：授权码直接映射到用户。
func fakeIdP(t *testing.T, users map[string]map[string]any) *httptest.Server {
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			_ = json.NewEncoder(w).Encode(map[string]string{
				"authorization_endpoint": srv.URL + "/authorize", "token_endpoint": srv.URL + "/token", "userinfo_endpoint": srv.URL + "/user",
			})
		case "/token":
			_ = r.ParseForm()
			if r.Form.Get("client_secret") != "sec" {
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "bad_client"})
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]string{"access_token": "tok-" + r.Form.Get("code")})
		case "/user":
			code := strings.TrimPrefix(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "), "tok-")
			_ = json.NewEncoder(w).Encode(users[code])
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestOAuthFlow(t *testing.T) {
	idp := fakeIdP(t, map[string]map[string]any{
		"alice": {"login": "alice"},
		"bob":   {"login": "bob"},
		"carol": {"email": "Carol@Example.com", "email_verified": true, "sub": "123"},
		"eve":   {"email": "eve@example.com", "email_verified": false, "sub": "999"},
	})
	old := thirdauth.Endpoints
	thirdauth.Endpoints.GitHub.Authorize, thirdauth.Endpoints.GitHub.Token, thirdauth.Endpoints.GitHub.User = idp.URL+"/authorize", idp.URL+"/token", idp.URL+"/user"
	defer func() { thirdauth.Endpoints = old }()

	ln, _ := net.Listen("tcp4", "127.0.0.1:0")
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()
	l := Listener{ListenIP: "127.0.0.1", Network: "tcp4", Port: port, Sites: []Site{
		{Enable: true, Key: "p", Domains: []string{"app.test"}, Type: TypeText, Text: "secret area", WebAuth: true,
			OAuthUsers: []string{"gh:alice", "google:carol@example.com"}},
	}, Default: Site{Enable: true, Type: TypeText, Text: "auth host"}}
	if err := l.Validate(); err != nil {
		t.Fatal(err)
	}
	auth := newAuthenticator("secret")
	auth.setOAuth(oauthEnv{
		providers: map[string]OAuthProvider{
			"gh":     {Key: "gh", Name: "GitHub", Type: thirdauth.TypeGitHub, ClientID: "id", ClientSecret: "sec"},
			"google": {Key: "google", Name: "Google", Type: thirdauth.TypeOIDC, ClientID: "id", ClientSecret: "sec", Issuer: idp.URL},
		},
		callbackBase: fmt.Sprintf("http://auth.test:%d", port),
	})
	r, err := newListenerRunner(l, logbuf.New(50, log.New(io.Discard, "", 0)).Logger("t"), auth, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := r.start(nil, nil); err != nil {
		t.Fatal(err)
	}
	defer r.stop()
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	c := noRedirect()
	// 所有请求都发往 127.0.0.1，Cookie 罐可携带登录发起时写入的 nonce
	c.Jar, _ = cookiejar.New(nil)

	// 以指定 Host 访问绝对地址（把 Location 中的域名映射到本地监听）
	follow := func(raw string) *http.Response {
		u, _ := url.Parse(raw)
		req, _ := http.NewRequest("GET", "http://"+addr+u.RequestURI(), nil)
		req.Host = u.Host
		resp, err := c.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		return resp
	}
	var lastTicket string
	// authorize 完成登录页 → 身份提供方 → 回调，返回带票据的回跳地址
	authorize := func(provider, code string) string {
		_, body := get(t, c, addr, "app.test", authPath+"/login", nil)
		if !strings.Contains(body, "使用 GitHub 登录") || !strings.Contains(body, "使用 Google 登录") {
			t.Fatalf("登录页应显示第三方登录入口: %s", body)
		}
		resp, _ := get(t, c, addr, "app.test", oauthStart+"?p="+provider+"&next=/dash", nil)
		loc, _ := url.Parse(resp.Header.Get("Location"))
		if !strings.HasPrefix(loc.String(), idp.URL+"/authorize") || loc.Query().Get("redirect_uri") != fmt.Sprintf("http://auth.test:%d%s", port, oauthCallback) {
			t.Fatalf("应跳转到身份提供方并携带回调地址: %s", loc)
		}
		// 身份提供方带着授权码回调认证域名
		cb := follow(fmt.Sprintf("http://auth.test:%d%s?code=%s&state=%s", port, oauthCallback, code, url.QueryEscape(loc.Query().Get("state"))))
		if cb.StatusCode != 302 {
			t.Fatalf("回调处理失败: %d", cb.StatusCode)
		}
		return cb.Header.Get("Location")
	}
	login := func(provider, code string) *http.Response {
		lastTicket = authorize(provider, code)
		return follow(lastTicket)
	}

	fin := login("gh", "alice")
	if fin.StatusCode != 302 || fin.Header.Get("Location") != "/dash" || len(fin.Cookies()) == 0 {
		t.Fatalf("允许的 GitHub 账号应登录成功: %d %s", fin.StatusCode, fin.Header.Get("Location"))
	}
	req, _ := http.NewRequest("GET", "http://"+addr+"/dash", nil)
	req.Host = "app.test"
	req.AddCookie(fin.Cookies()[0])
	resp, _ := c.Do(req)
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if string(body) != "secret area" {
		t.Fatalf("登录后应能访问: %d %q", resp.StatusCode, body)
	}

	// 票据只能使用一次
	if again := follow(lastTicket); again.StatusCode != 400 {
		t.Fatalf("重放票据应被拒绝: %d", again.StatusCode)
	}
	// 没有发起登录时写入的 nonce Cookie（例如票据被注入到其他浏览器）应被拒绝
	saved := c.Jar
	ticketURL := authorize("gh", "alice")
	c.Jar, _ = cookiejar.New(nil)
	if other := follow(ticketURL); other.StatusCode != 400 {
		t.Fatalf("其他浏览器使用票据应被拒绝: %d", other.StatusCode)
	}
	c.Jar = saved
	if fin := follow(ticketURL); fin.StatusCode != 302 {
		t.Fatalf("发起登录的浏览器应能使用票据: %d", fin.StatusCode)
	}

	if fin := login("gh", "bob"); fin.StatusCode != 403 {
		t.Fatalf("不在允许列表的账号应被拒绝: %d", fin.StatusCode)
	}
	if fin := login("google", "carol"); fin.StatusCode != 302 {
		t.Fatalf("OIDC 邮箱（大小写不敏感）应登录成功: %d", fin.StatusCode)
	}
	if fin := login("google", "eve"); fin.StatusCode != 403 {
		t.Fatalf("未验证邮箱应退回使用 sub 且被拒绝: %d", fin.StatusCode)
	}

	// 回调只在认证域名上生效
	if resp, _ := get(t, c, addr, "app.test", oauthCallback+"?code=alice&state=x", nil); resp.StatusCode == 302 {
		t.Fatal("非认证域名不应处理回调")
	}
	// 伪造的票据无效
	if resp, _ := get(t, c, addr, "app.test", oauthFinish+"?ticket=forged.sig", nil); resp.StatusCode != 400 {
		t.Fatalf("伪造票据应被拒绝: %d", resp.StatusCode)
	}
}
