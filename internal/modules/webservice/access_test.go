package webservice

import (
	"bytes"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/maxmind/mmdbwriter"
	"github.com/maxmind/mmdbwriter/mmdbtype"

	"github.com/wangwanjie/gately/internal/geoip"
)

func okBackend(t *testing.T) *httptest.Server {
	t.Helper()
	b := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("ok:" + r.URL.Path)) }))
	t.Cleanup(b.Close)
	return b
}

func TestShareLinkFlow(t *testing.T) {
	b := okBackend(t)
	l := Listener{Sites: []Site{{
		Enable: true, Domains: []string{"s.test"}, Targets: []string{b.URL}, BasicAuth: true, Users: []string{"alice:secret"},
		ShareLinks: []ShareLink{
			{Key: "a", Name: "全站", Token: "aaaaaaaaaaaaaaaaaaaa"},
			{Key: "once", Name: "一次", Token: "bbbbbbbbbbbbbbbbbbbb", MaxUses: 1},
			{Key: "old", Name: "过期", Token: "cccccccccccccccccccc", ExpiresAt: time.Now().Add(-time.Hour)},
			{Key: "docs", Name: "文档", Token: "dddddddddddddddddddd", PathPrefix: "/docs"},
		},
	}}}
	addr, _ := startRunner(t, l, nil, nil)
	c := noRedirect()

	if resp, _ := get(t, c, addr, "s.test", "/", nil); resp.StatusCode != 401 {
		t.Fatalf("未认证应 401，实际 %d", resp.StatusCode)
	}
	// 兑换：302 到去掉令牌的地址，并下发 Cookie
	resp, _ := get(t, c, addr, "s.test", "/page?x=1&_gs=aaaaaaaaaaaaaaaaaaaa", nil)
	if resp.StatusCode != 302 || resp.Header.Get("Location") != "/page?x=1" || len(resp.Cookies()) == 0 {
		t.Fatalf("兑换失败: %d %s", resp.StatusCode, resp.Header.Get("Location"))
	}
	cookie := resp.Cookies()[0]
	req, _ := http.NewRequest("GET", "http://"+addr+"/page", nil)
	req.Host = "s.test"
	req.AddCookie(cookie)
	r2, err := c.Do(req)
	if err != nil || r2.StatusCode != 200 {
		t.Fatalf("持有分享 Cookie 应放行: %v %v", r2, err)
	}
	r2.Body.Close()

	// 错误令牌与过期链接都回到认证
	for _, tok := range []string{"zzzzzzzzzzzzzzzzzzzz", "cccccccccccccccccccc"} {
		if resp, _ := get(t, c, addr, "s.test", "/?_gs="+tok, nil); resp.StatusCode != 401 {
			t.Fatalf("令牌 %s 应无效，实际 %d", tok, resp.StatusCode)
		}
	}
	// 次数限制
	if resp, _ := get(t, c, addr, "s.test", "/?_gs=bbbbbbbbbbbbbbbbbbbb", nil); resp.StatusCode != 302 {
		t.Fatalf("首次兑换应成功，实际 %d", resp.StatusCode)
	}
	if resp, _ := get(t, c, addr, "s.test", "/?_gs=bbbbbbbbbbbbbbbbbbbb", nil); resp.StatusCode != 401 {
		t.Fatalf("超过次数应无效，实际 %d", resp.StatusCode)
	}
	// 路径限制：在 / 兑换 docs 链接失败，在 /docs 兑换成功且 Cookie 不能访问其他路径
	if resp, _ := get(t, c, addr, "s.test", "/?_gs=dddddddddddddddddddd", nil); resp.StatusCode != 401 {
		t.Fatalf("路径不在范围内应无效，实际 %d", resp.StatusCode)
	}
	resp, _ = get(t, c, addr, "s.test", "/docs/a?_gs=dddddddddddddddddddd", nil)
	if resp.StatusCode != 302 || len(resp.Cookies()) == 0 {
		t.Fatalf("docs 链接兑换失败: %d", resp.StatusCode)
	}
	for path, want := range map[string]int{"/docs/a": 200, "/docs": 200, "/other": 401, "/docsx": 401} {
		req, _ := http.NewRequest("GET", "http://"+addr+path, nil)
		req.Host = "s.test"
		req.AddCookie(resp.Cookies()[0])
		r3, err := c.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		r3.Body.Close()
		if r3.StatusCode != want {
			t.Errorf("docs Cookie 访问 %s 应 %d，实际 %d", path, want, r3.StatusCode)
		}
	}
}

func TestPathRules(t *testing.T) {
	b := okBackend(t)
	l := Listener{Sites: []Site{
		{Enable: true, Domains: []string{"opt.test"}, Targets: []string{b.URL}, BasicAuth: true, Users: []string{"a:b"}, AuthOptional: true,
			Rules: []PathRule{{Prefix: "/admin", Auth: RuleAuthOn}}},
		{Enable: true, Domains: []string{"pub.test"}, Targets: []string{b.URL}, BasicAuth: true, Users: []string{"a:b"},
			Rules: []PathRule{{Prefix: "/health", Auth: RuleAuthOff}, {Prefix: "/internal", IPMode: IPModeWhitelist, IPList: []string{"10.0.0.0/8"}}}},
	}}
	addr, _ := startRunner(t, l, nil, nil)
	c := noRedirect()
	cases := []struct {
		host, path string
		want       int
	}{
		{"opt.test", "/", 200},
		{"opt.test", "/admin", 401},
		{"opt.test", "/admin/users", 401},
		{"opt.test", "/administrator", 200},
		{"opt.test", "/public/../admin", 401},
		{"opt.test", "//admin", 401},
		{"pub.test", "/", 401},
		{"pub.test", "/health", 200},
		{"pub.test", "/health/x", 200},
		{"pub.test", "/healthz", 401},
		{"pub.test", "/internal/x", 403},
	}
	for _, tc := range cases {
		if resp, _ := get(t, c, addr, tc.host, tc.path, nil); resp.StatusCode != tc.want {
			t.Errorf("%s%s 应 %d，实际 %d", tc.host, tc.path, tc.want, resp.StatusCode)
		}
	}
}

func TestAutoBan(t *testing.T) {
	b := okBackend(t)
	l := Listener{Sites: []Site{{Enable: true, Domains: []string{"b.test"}, Targets: []string{b.URL}, BasicAuth: true, Users: []string{"alice:secret"}}}}
	addr, r := startRunner(t, l, nil, nil)
	r.auth.setBanSeconds(60)
	c := noRedirect()
	bad := func() int {
		req, _ := http.NewRequest("GET", "http://"+addr+"/", nil)
		req.Host = "b.test"
		req.SetBasicAuth("alice", "wrong")
		resp, err := c.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		return resp.StatusCode
	}
	for i := 0; i < maxFailures; i++ {
		if code := bad(); code != 401 {
			t.Fatalf("第 %d 次失败应 401，实际 %d", i+1, code)
		}
	}
	// 达到上限后连正确凭据也被拒绝（403 来自封禁）
	req, _ := http.NewRequest("GET", "http://"+addr+"/", nil)
	req.Host = "b.test"
	req.SetBasicAuth("alice", "secret")
	resp, _ := c.Do(req)
	resp.Body.Close()
	if resp.StatusCode != 403 {
		t.Fatalf("被封禁后应 403，实际 %d", resp.StatusCode)
	}
	bans := r.auth.listBans()
	if len(bans) != 1 || bans[0].Key != "127.0.0.1" || bans[0].Manual {
		t.Fatalf("封禁列表不符: %+v", bans)
	}
	r.auth.unban("127.0.0.1")
	if len(r.auth.listBans()) != 0 {
		t.Fatal("解除后列表应为空")
	}
}

func TestBanDisabledByDefaultAndExpiry(t *testing.T) {
	a := newAuthenticator("s")
	for i := 0; i < maxFailures; i++ {
		a.release("1.1.1.1", false)
	}
	if a.banned(netip.MustParseAddr("1.1.1.1")) {
		t.Fatal("未开启自动封禁时不应封禁")
	}
	a.ban("2001:db8::5", time.Millisecond, "t", true)
	time.Sleep(5 * time.Millisecond)
	if a.banned(netip.MustParseAddr("2001:db8::9")) {
		t.Fatal("封禁过期后应解除")
	}
	a.ban("2001:db8::5", time.Minute, "t", true)
	if !a.banned(netip.MustParseAddr("2001:db8::ffff")) {
		t.Fatal("IPv6 应按 /64 聚合封禁")
	}
}

func testCountryDB(t *testing.T) {
	t.Helper()
	w, err := mmdbwriter.New(mmdbwriter.Options{DatabaseType: "Test", RecordSize: 24})
	if err != nil {
		t.Fatal(err)
	}
	for cidr, code := range map[string]string{"1.0.0.0/24": "AU", "114.114.114.0/24": "CN"} {
		_, n, _ := net.ParseCIDR(cidr)
		if err := w.Insert(n, mmdbtype.Map{"country": mmdbtype.Map{"iso_code": mmdbtype.String(code)}}); err != nil {
			t.Fatal(err)
		}
	}
	var buf bytes.Buffer
	if _, err := w.WriteTo(&buf); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "c.mmdb")
	if _, err := geoip.Save(path, &buf); err != nil {
		t.Fatal(err)
	}
	if err := geoip.Default.Load(path); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(geoip.Default.Close)
}

func TestCountryFilter(t *testing.T) {
	mk := func(mode string, codes ...string) *siteHandler {
		h := &siteHandler{site: Site{CountryMode: mode, Countries: codes}, countries: map[string]bool{}}
		for _, c := range codes {
			h.countries[c] = true
		}
		return h
	}
	ip := netip.MustParseAddr
	allow, deny := mk(CountryAllow, "CN"), mk(CountryDeny, "AU")
	// 数据库未加载时放行
	if !allow.countryAllowed(ip("1.0.0.1")) {
		t.Fatal("未加载数据库应放行")
	}
	testCountryDB(t)
	cases := []struct {
		h    *siteHandler
		ip   string
		want bool
	}{
		{allow, "114.114.114.114", true},
		{allow, "1.0.0.1", false},
		{allow, "8.8.8.8", false},
		{allow, "192.168.1.5", true},
		{deny, "1.0.0.1", false},
		{deny, "114.114.114.114", true},
		{deny, "8.8.8.8", true},
		{mk(CountryOff), "1.0.0.1", true},
	}
	for _, c := range cases {
		if got := c.h.countryAllowed(ip(c.ip)); got != c.want {
			t.Errorf("%s mode=%s: got %v want %v", c.ip, c.h.site.CountryMode, got, c.want)
		}
	}
}

func TestValidateAccess(t *testing.T) {
	base := func() Site {
		return Site{Domains: []string{"a.com"}, Targets: []string{"http://127.0.0.1:1"}}
	}
	bad := map[string]func(*Site){
		"国家模式缺代码":  func(s *Site) { s.CountryMode = CountryAllow },
		"国家代码无效":   func(s *Site) { s.CountryMode = CountryDeny; s.Countries = []string{"CHN"} },
		"规则无斜杠":    func(s *Site) { s.Rules = []PathRule{{Prefix: "admin"}} },
		"规则重复":     func(s *Site) { s.Rules = []PathRule{{Prefix: "/a"}, {Prefix: "/a/"}} },
		"规则认证无效":   func(s *Site) { s.Rules = []PathRule{{Prefix: "/a", Auth: "x"}} },
		"分享链接无认证":  func(s *Site) { s.ShareLinks = []ShareLink{{Name: "x"}} },
		"仅路径认证无认证": func(s *Site) { s.AuthOptional = true },
	}
	for name, mut := range bad {
		s := base()
		mut(&s)
		if err := s.validate(false); err == nil {
			t.Errorf("%s: 应校验失败", name)
		}
	}
	s := base()
	s.BasicAuth, s.Users = true, []string{"a:b"}
	s.CountryMode, s.Countries = CountryAllow, []string{" cn ", "us", "CN"}
	s.Rules = []PathRule{{Prefix: "/admin/", Auth: RuleAuthOn}}
	s.ShareLinks = []ShareLink{{Name: " 临时 "}, {Name: "b", Key: "dup"}, {Name: "c", Key: "dup"}}
	if err := s.validate(false); err != nil {
		t.Fatal(err)
	}
	if strings.Join(s.Countries, ",") != "CN,US" || s.Rules[0].Prefix != "/admin" {
		t.Fatalf("规范化结果不符: %v %v", s.Countries, s.Rules)
	}
	l := s.ShareLinks
	if l[0].Name != "临时" || len(l[0].Token) < 16 || l[0].Key == "" || l[1].Key == l[2].Key || l[1].Token == l[2].Token {
		t.Fatalf("分享链接应补全 key/token 且不重复: %+v", l)
	}
}
