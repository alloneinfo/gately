package app

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/wangwanjie/gately/internal/module"
	"github.com/wangwanjie/gately/internal/modules/webservice"
	"github.com/wangwanjie/gately/internal/thirdauth"
)

// fakeProviders 模拟 GitHub：授权码即用户名。
func fakeGitHub(t *testing.T) {
	t.Helper()
	idp := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			_ = r.ParseForm()
			_ = json.NewEncoder(w).Encode(map[string]string{"access_token": "tok-" + r.Form.Get("code")})
		case "/user":
			name := strings.TrimPrefix(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "), "tok-")
			id := int64(99)
			if name == "VanJay" {
				id = 15172899
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"login": name, "id": id})
		}
	}))
	old := thirdauth.Endpoints
	thirdauth.Endpoints.GitHub.Authorize, thirdauth.Endpoints.GitHub.Token, thirdauth.Endpoints.GitHub.User = idp.URL+"/authorize", idp.URL+"/token", idp.URL+"/user"
	t.Cleanup(func() {
		thirdauth.Endpoints = old
		idp.Close()
	})
}

func TestAdminThirdLogin(t *testing.T) {
	fakeGitHub(t)
	_, srv, admin := loginApp(t)
	_, body := doJSON(t, admin, "GET", srv.URL+"/api/system/settings", nil)
	var s settingsView
	_ = json.Unmarshal(body, &s)
	s.ThirdLogin = thirdauth.LoginConfig{
		Enable: true, CallbackBase: srv.URL + "/",
		Providers: []thirdauth.Provider{{Key: "github", Name: "GitHub", Type: "github", ClientID: "id", ClientSecret: "sec"}},
		// 按数字 ID 授权，用户名放在备注里
		Allowed: []string{"github:15172899 # VanJay"},
	}
	if code, body := doJSON(t, admin, "PUT", srv.URL+"/api/system/settings", s); code != 200 {
		t.Fatalf("保存失败 %d %s", code, body)
	}
	// 再次保存时脱敏的 Secret 应还原为原值
	_, body = doJSON(t, admin, "GET", srv.URL+"/api/system/settings", nil)
	_ = json.Unmarshal(body, &s)
	if s.ThirdLogin.Providers[0].ClientSecret != "********" || s.ThirdLogin.CallbackBase != srv.URL {
		t.Fatalf("Secret 应脱敏、回调地址应去掉末尾斜杠: %s", body)
	}
	if code, body := doJSON(t, admin, "PUT", srv.URL+"/api/system/settings", s); code != 200 {
		t.Fatalf("回写脱敏设置失败 %d %s", code, body)
	}
	// 查看 Client Secret 明文按登录方式标识返回原值
	if code, body := doJSON(t, admin, "GET", srv.URL+"/api/system/settings/secrets", nil); code != 200 || strings.TrimSpace(string(body)) != `{"github":"sec"}` {
		t.Fatalf("应返回 Client Secret 明文 %d %s", code, body)
	}

	newBrowser := func() *http.Client {
		jar, _ := cookiejar.New(nil)
		return &http.Client{Jar: jar, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	}
	// 未登录也能取得登录方式
	b := newBrowser()
	_, body = doJSON(t, b, "GET", srv.URL+"/api/login/providers", nil)
	if !strings.Contains(string(body), `"key":"github"`) {
		t.Fatalf("应列出 GitHub 登录: %s", body)
	}
	// start 返回 302 并携带 state
	start := func(c *http.Client) string {
		resp, err := c.Get(srv.URL + thirdauth.StartPath + "?p=github")
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		loc, _ := url.Parse(resp.Header.Get("Location"))
		if resp.StatusCode != 302 || loc.Query().Get("redirect_uri") != srv.URL+thirdauth.CallbackPath {
			t.Fatalf("应跳转身份提供方: %d %s", resp.StatusCode, loc)
		}
		return loc.Query().Get("state")
	}
	callback := func(c *http.Client, code, state string) *http.Response {
		resp, err := c.Get(srv.URL + thirdauth.CallbackPath + "?code=" + code + "&state=" + url.QueryEscape(state))
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		return resp
	}

	// 允许的账号（按数字 ID 匹配）登录成功并获得会话
	state := start(b)
	if resp := callback(b, "VanJay", state); resp.Header.Get("Location") != "/" {
		t.Fatalf("登录应跳转首页: %d %s", resp.StatusCode, resp.Header.Get("Location"))
	}
	if code, body := doJSON(t, b, "GET", srv.URL+"/api/session", nil); code != 200 {
		t.Fatalf("第三方登录后应有会话 %d %s", code, body)
	}
	// state 不可重放
	if resp := callback(b, "VanJay", state); !strings.HasPrefix(resp.Header.Get("Location"), "/#/login?oauthError=") {
		t.Fatalf("重放 state 应失败: %s", resp.Header.Get("Location"))
	}
	// 同一用户名但数字 ID 不同的账号被拒绝，并提示完整标识
	b2 := newBrowser()
	resp := callback(b2, "mallory", start(b2))
	// 错误原因放在 hash 路由的查询串里
	_, rawQuery, _ := strings.Cut(resp.Header.Get("Location"), "?")
	if msg, _ := url.QueryUnescape(strings.TrimPrefix(rawQuery, "oauthError=")); !strings.Contains(msg, "github:mallory") {
		t.Fatalf("未授权账号应被拒绝并提示标识: %s", resp.Header.Get("Location"))
	}
	if code, _ := doJSON(t, b2, "GET", srv.URL+"/api/session", nil); code != 401 {
		t.Fatalf("被拒绝的账号不应有会话: %d", code)
	}
	// 其他浏览器使用 state（没有发起登录时写入的 nonce）被拒绝
	victim := newBrowser()
	st := start(newBrowser())
	if resp := callback(victim, "VanJay", st); !strings.Contains(resp.Header.Get("Location"), "oauthError") {
		t.Fatalf("其他浏览器使用 state 应被拒绝: %s", resp.Header.Get("Location"))
	}
	// 停用后不再提供登录方式
	s.ThirdLogin.Enable = false
	doJSON(t, admin, "PUT", srv.URL+"/api/system/settings", s)
	if _, body := doJSON(t, b, "GET", srv.URL+"/api/login/providers", nil); strings.Contains(string(body), "github") {
		t.Fatalf("停用后不应列出登录方式: %s", body)
	}
}

func TestAdminThirdLoginTOTP(t *testing.T) {
	fakeGitHub(t)
	a, srv, _ := loginApp(t)
	// 启用两步验证后，未勾选“跳过两步验证”时第三方登录被拒绝
	a.settingsMu.Lock()
	a.settings.ThirdLogin = thirdauth.LoginConfig{Enable: true, CallbackBase: srv.URL,
		Providers: []thirdauth.Provider{{Key: "github", Type: "github", ClientID: "id", ClientSecret: "sec"}}, Allowed: []string{"github:*"}}
	a.settingsMu.Unlock()
	secret := "JBSWY3DPEHPK3PXP"
	if err := a.auth.SetTOTP("test-password", "", secret); err != nil {
		t.Fatal(err)
	}
	jar, _ := cookiejar.New(nil)
	b := &http.Client{Jar: jar, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	resp, _ := b.Get(srv.URL + thirdauth.StartPath + "?p=github")
	resp.Body.Close()
	loc, _ := url.Parse(resp.Header.Get("Location"))
	resp, _ = b.Get(srv.URL + thirdauth.CallbackPath + "?code=VanJay&state=" + url.QueryEscape(loc.Query().Get("state")))
	resp.Body.Close()
	if !strings.Contains(resp.Header.Get("Location"), "oauthError") {
		t.Fatalf("启用两步验证时应拒绝: %s", resp.Header.Get("Location"))
	}
}

// TestImportLuckyThirdParty 验证 Lucky 第三方身份表、站点引用与基础设置的迁移。
func TestImportLuckyThirdParty(t *testing.T) {
	dir := t.TempDir()
	ws := webservice.New(nil, nil)
	a, err := New(Options{ConfigDir: dir, Modules: []module.Module{ws}})
	if err != nil {
		t.Fatal(err)
	}
	a.logger = log.New(io.Discard, "", 0)
	if err := a.auth.ChangeCredentials(mustInitialPassword(t, a), "admin", "test-password"); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(a.Handler())
	defer srv.Close()
	jar, _ := cookiejar.New(nil)
	c := &http.Client{Jar: jar}
	doJSON(t, c, "POST", srv.URL+"/api/login", map[string]string{"username": "admin", "password": "test-password"})

	identities := `{"List":[
	 {"Key":"k1","Enable":true,"Type":"github","ID":"15172899","Name":"VanJay"},
	 {"Key":"k2","Enable":true,"Type":"qq","ID":"UID_ABC","Name":"VanJay"},
	 {"Key":"k3","Enable":true,"Type":"google","ID":"1146","Name":"万杰","EMail":"a@b.c"},
	 {"Key":"k4","Enable":false,"Type":"github","ID":"7","Name":"off"}],"Config":null,"ConfVer":1}`
	rp := `{"ReverseProxyRuleList":[{"RuleName":"r","Enable":true,"ListenPort":4434,"Network":"tcp4",
	  "DefaultProxy":{"Locations":["http://127.0.0.1:1"],"WebServiceType":"reverseproxy","OtherParams":{"WebAuth":true,"AllowThirdUserList":["k1","k2","k3"]}},
	  "ProxyList":[
	   {"Enable":true,"Domains":["d.example.com"],"Locations":["http://127.0.0.1:2"],"WebServiceType":"reverseproxy","BasicAuthUserList":["bob:pw"],"OtherParams":{"WebAuth":true,"AllowAllThirdUsers":true}},
	   {"Enable":true,"Domains":["e.example.com"],"Locations":["http://127.0.0.1:3"],"WebServiceType":"reverseproxy","OtherParams":{"WebAuth":true,"AllowThirdUserList":["gone"]}}
	  ]}]}`
	base := `{"AdminWebListenPort":16601,"EnableThirdAuthLogin":true,"AllowAllThirdAuthUsers":false,"ThirdAuthLoginUserList":["k1","k3"],"ThirdAuthLoginSkipTwoFA":true}`
	// 文件顺序不影响结果：身份表总是先于站点与基础设置处理
	code, body := uploadFiles(t, c, srv.URL+"/api/import/lucky?enable=1", "files", map[string][]byte{
		"lucky_base.lkcf": []byte(base), "lucky_reverseproxy.lkcf": []byte(rp), "lucky_thirdPartyAuthManager.lkcf": []byte(identities),
	})
	if code != 200 || strings.Contains(string(body), `"error"`) {
		t.Fatalf("导入失败 %d %s", code, body)
	}

	_, body = doJSON(t, c, "GET", srv.URL+"/api/system/settings", nil)
	var s settingsView
	_ = json.Unmarshal(body, &s)
	want := []string{"github:15172899 # VanJay", "google:1146 # 万杰 a@b.c"}
	if strings.Join(s.ThirdLogin.Allowed, "|") != strings.Join(want, "|") || !s.ThirdLogin.SkipTOTP || s.ThirdLogin.Enable {
		t.Fatalf("管理后台第三方登录迁移错误（不应自动启用）: %+v", s.ThirdLogin)
	}

	_, body = doJSON(t, c, "GET", srv.URL+"/api/webservice/listeners", nil)
	var views []struct {
		Default webservice.Site
		Sites   []webservice.Site
	}
	if err := json.Unmarshal(body, &views); err != nil || len(views) != 1 {
		t.Fatalf("监听规则: %v %s", err, body)
	}
	v := views[0]
	if !v.Default.WebAuth || !v.Default.Enable || len(v.Default.OAuthUsers) != 3 || v.Default.OAuthUsers[0] != "github:15172899 # VanJay" || v.Default.OAuthUsers[1] != "qq:UID_ABC # VanJay" {
		t.Fatalf("仅第三方登录的默认站点应保留认证并带上身份: %+v", v.Default)
	}
	// AllowAllThirdUsers 展开为全部启用的身份（停用的不含）
	if len(v.Sites[0].OAuthUsers) != 3 || len(v.Sites[0].Users) != 1 {
		t.Fatalf("允许全部第三方用户: %+v", v.Sites[0])
	}
	// 引用的身份找不到且没有站点用户：停用，避免去掉认证后暴露
	if v.Sites[1].Enable || v.Sites[1].WebAuth {
		t.Fatalf("无可用认证的站点应停用: %+v", v.Sites[1])
	}
}
