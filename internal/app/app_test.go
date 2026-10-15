package app

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"mime/multipart"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/wangwanjie/gately/internal/auth"
	"github.com/wangwanjie/gately/internal/module"
	"github.com/wangwanjie/gately/internal/modules/portforward"
)

// newTestServer 启动带端口转发模块的测试实例，并以已知密码登录。
func newTestServer(t *testing.T) (*httptest.Server, *http.Client) {
	t.Helper()
	dir := t.TempDir()
	a, err := New(Options{ConfigDir: dir, Modules: []module.Module{portforward.New()}})
	if err != nil {
		t.Fatal(err)
	}
	a.logger = log.New(io.Discard, "", 0)
	// 把随机初始密码替换成已知值，便于测试登录
	a.auth = auth.NewManager(auth.Account{}, func(auth.Account) error { return nil })
	if _, err := a.auth.EnsureInitialized(); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(a.Handler())
	t.Cleanup(srv.Close)
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}
	return srv, client
}

func doJSON(t *testing.T, c *http.Client, method, url string, body any) (int, []byte) {
	t.Helper()
	var r io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		r = bytes.NewReader(b)
	}
	req, _ := http.NewRequest(method, url, r)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, data
}

func TestAPIRequiresLogin(t *testing.T) {
	srv, c := newTestServer(t)
	if code, _ := doJSON(t, c, "GET", srv.URL+"/api/status", nil); code != http.StatusUnauthorized {
		t.Fatalf("未登录应返回 401，实际 %d", code)
	}
	if code, _ := doJSON(t, c, "GET", srv.URL+"/api/portforward/rules", nil); code != http.StatusUnauthorized {
		t.Fatalf("模块接口未登录应返回 401，实际 %d", code)
	}
	if code, _ := doJSON(t, c, "POST", srv.URL+"/api/login", map[string]string{"username": "admin", "password": "x"}); code != http.StatusUnauthorized {
		t.Fatalf("错误密码应返回 401，实际 %d", code)
	}
}

func TestPortForwardCRUDAndImport(t *testing.T) {
	dir := t.TempDir()
	pf := portforward.New()
	a, err := New(Options{ConfigDir: dir, Modules: []module.Module{pf}})
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

	if code, body := doJSON(t, c, "POST", srv.URL+"/api/login", map[string]string{"username": "admin", "password": "test-password"}); code != 200 {
		t.Fatalf("登录失败 %d %s", code, body)
	}

	port := freeTCPPort(t)
	rule := map[string]any{
		"name": "web", "enable": false, "protocols": []string{"tcp4"},
		"listenAddress": "127.0.0.1", "listenPorts": strconv.Itoa(port),
		"targetAddresses": []string{"127.0.0.1"}, "targetPorts": "80",
	}
	code, body := doJSON(t, c, "POST", srv.URL+"/api/portforward/rules", rule)
	if code != 200 {
		t.Fatalf("创建失败 %d %s", code, body)
	}
	var created portforward.Rule
	_ = json.Unmarshal(body, &created)
	if created.Key == "" {
		t.Fatal("应生成 key")
	}

	// 非法规则应被拒绝
	bad := map[string]any{"protocols": []string{"icmp"}, "listenPorts": "1", "targetAddresses": []string{"x"}}
	if code, _ := doJSON(t, c, "POST", srv.URL+"/api/portforward/rules", bad); code != http.StatusBadRequest {
		t.Fatalf("非法规则应返回 400，实际 %d", code)
	}

	// 启用后应处于运行状态
	if code, body := doJSON(t, c, "PUT", srv.URL+"/api/portforward/rules/"+created.Key+"/enable", map[string]bool{"enable": true}); code != 200 {
		t.Fatalf("启用失败 %d %s", code, body)
	}
	// 模块未经 Run 启动时没有运行上下文，这里只验证配置被持久化
	_, body = doJSON(t, c, "GET", srv.URL+"/api/portforward/rules", nil)
	var views []portforward.RuleView
	_ = json.Unmarshal(body, &views)
	if len(views) != 1 || !views[0].Enable {
		t.Fatalf("列表异常: %s", body)
	}

	// 导入明文 JSON 形式的 Lucky 配置（预览）
	lucky := `{"PortForwardsRuleList":[{"Name":"ssh","Enable":false,"ForwardTypes":["tcp"],"ListenPorts":"6022","TargetAddressList":["192.168.1.2"],"TargetPorts":"22","Options":{}}]}`
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, _ := mw.CreateFormFile("files", "lucky_portforward.lkcf")
	_, _ = fw.Write([]byte(lucky))
	_ = mw.Close()
	req, _ := http.NewRequest("POST", srv.URL+"/api/import/lucky?dryRun=1", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	resp, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	body, _ = io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != 200 || !bytes.Contains(body, []byte(`"imported":1`)) {
		t.Fatalf("导入预览异常 %d %s", resp.StatusCode, body)
	}

	if code, _ := doJSON(t, c, "DELETE", srv.URL+"/api/portforward/rules/"+created.Key, nil); code != 200 {
		t.Fatalf("删除失败 %d", code)
	}
}

// mustInitialPassword 重置为已知的初始密码并返回。
func mustInitialPassword(t *testing.T, a *App) string {
	t.Helper()
	a.auth = auth.NewManager(auth.Account{}, func(acc auth.Account) error { return a.store.Save(accountName, acc) })
	p, err := a.auth.EnsureInitialized()
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func freeTCPPort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port
}

func TestCrossOriginRejected(t *testing.T) {
	srv, c := newTestServer(t)
	req, _ := http.NewRequest("POST", srv.URL+"/api/login", strings.NewReader(`{"username":"a","password":"b"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://evil.example:8080")
	resp, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("跨源请求应被拒绝: %d", resp.StatusCode)
	}
	// 简单请求（text/plain）不能绕过 JSON 要求
	req, _ = http.NewRequest("POST", srv.URL+"/api/login", strings.NewReader(`{"username":"a","password":"b"}`))
	req.Header.Set("Content-Type", "text/plain")
	resp, _ = c.Do(req)
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("非 JSON 请求体应被拒绝: %d", resp.StatusCode)
	}
	// 同源请求放行
	req, _ = http.NewRequest("POST", srv.URL+"/api/logout", nil)
	req.Header.Set("Origin", srv.URL)
	resp, _ = c.Do(req)
	resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("同源请求应放行: %d", resp.StatusCode)
	}
}
