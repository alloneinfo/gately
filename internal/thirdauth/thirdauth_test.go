package thirdauth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestAllowed(t *testing.T) {
	entries := []string{"github:15172899 # VanJay", "Google:A@B.c", "qq:*"}
	cases := []struct {
		id   Identity
		want bool
	}{
		{Identity{Provider: "github", Primary: "renamed", Aliases: []string{"15172899"}}, true},
		{Identity{Provider: "github", Primary: "VanJay"}, false},
		{Identity{Provider: "google", Primary: "a@b.c"}, true},
		{Identity{Provider: "google", Primary: "sub1", Aliases: []string{}}, false},
		{Identity{Provider: "qq", Primary: "anything"}, true},
		{Identity{Provider: "weibo", Primary: "x"}, false},
	}
	for _, c := range cases {
		if got := Allowed(entries, c.id); got != c.want {
			t.Errorf("%+v: got %v want %v", c.id, got, c.want)
		}
	}
	if err := ValidateEntries([]string{"github"}); err == nil {
		t.Error("缺少标识应校验失败")
	}
	if p, v, ok := ParseEntry("qq:UID_X  # 备注"); !ok || p != "qq" || v != "UID_X" {
		t.Errorf("备注应被忽略: %q %q %v", p, v, ok)
	}
}

func TestLuckyResolve(t *testing.T) {
	ids := LuckyIdentities{List: []LuckyIdentity{
		{Key: "a", Enable: true, Type: "GitHub", ID: "1", Name: "me"},
		{Key: "b", Enable: false, Type: "qq", ID: "2"},
		{Key: "c", Enable: true, Type: "google", ID: "3", Name: "n", EMail: "e@x.y"},
	}}
	got, missing := ids.Resolve([]string{"a", "b", "zzz", "c"}, false)
	if strings.Join(got, "|") != "github:1 # me|google:3 # n e@x.y" || missing != 1 {
		t.Errorf("按 Key 解析: %q missing=%d", got, missing)
	}
	all, _ := ids.Resolve(nil, true)
	if len(all) != 2 {
		t.Errorf("全部身份应只含启用项: %q", all)
	}
}

func TestQQExchange(t *testing.T) {
	var gotRedirect string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			gotRedirect = r.URL.Query().Get("redirect_uri")
			if r.URL.Query().Get("client_secret") != "sec" || r.URL.Query().Get("code") != "c1" {
				_ = json.NewEncoder(w).Encode(map[string]any{"error": 100010, "error_description": "bad"})
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "AT", "expires_in": 7776000})
		case "/me":
			if r.URL.Query().Get("access_token") != "AT" {
				_ = json.NewEncoder(w).Encode(map[string]any{"error": 100016})
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]string{"client_id": "id", "openid": "UID_ABC"})
		}
	}))
	defer srv.Close()
	old := Endpoints
	Endpoints.QQ.Authorize, Endpoints.QQ.Token, Endpoints.QQ.Me = srv.URL+"/authorize", srv.URL+"/token", srv.URL+"/me"
	defer func() { Endpoints = old }()

	p := Provider{Key: "qq", Type: TypeQQ, ClientID: "id", ClientSecret: "sec"}
	if err := p.Validate(); err != nil {
		t.Fatal(err)
	}
	au, err := p.AuthorizeURL(context.Background(), "https://x/cb", "st")
	if err != nil {
		t.Fatal(err)
	}
	u, _ := url.Parse(au)
	if u.Query().Get("response_type") != "code" || u.Query().Get("scope") != "get_user_info" || u.Query().Get("state") != "st" {
		t.Errorf("QQ 授权地址参数错误: %s", au)
	}
	id, err := p.Exchange(context.Background(), "c1", "https://x/cb")
	if err != nil || id.String() != "qq:UID_ABC" || gotRedirect != "https://x/cb" {
		t.Fatalf("QQ 换取身份: %+v %v", id, err)
	}
	if _, err := p.Exchange(context.Background(), "bad", "https://x/cb"); err == nil {
		t.Error("授权码错误应失败")
	}
}

func TestGoogleDefaultsIssuer(t *testing.T) {
	p := Provider{Key: "google", Type: TypeGoogle, ClientID: "i", ClientSecret: "s"}
	if err := p.Validate(); err != nil || p.Issuer != GoogleIssuer {
		t.Errorf("Google 应自动使用固定 Issuer: %v %q", err, p.Issuer)
	}
}
