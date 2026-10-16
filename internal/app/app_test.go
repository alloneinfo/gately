package app

import (
	"archive/zip"
	"bytes"
	"context"
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
	"time"

	"github.com/wangwanjie/gately/internal/auth"
	"github.com/wangwanjie/gately/internal/module"
	"github.com/wangwanjie/gately/internal/modules/portforward"
	"github.com/wangwanjie/gately/internal/modules/ssl"
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

// loginApp 创建带端口转发模块的实例并以已知密码登录，返回实例、服务地址和客户端。
func loginApp(t *testing.T) (*App, *httptest.Server, *http.Client) {
	t.Helper()
	dir := t.TempDir()
	a, err := New(Options{ConfigDir: dir, Modules: []module.Module{portforward.New()}})
	if err != nil {
		t.Fatal(err)
	}
	a.logger = log.New(io.Discard, "", 0)
	if err := a.auth.ChangeCredentials(mustInitialPassword(t, a), "admin", "test-password"); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(a.Handler())
	t.Cleanup(srv.Close)
	jar, _ := cookiejar.New(nil)
	c := &http.Client{Jar: jar, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	if code, body := doJSON(t, c, "POST", srv.URL+"/api/login", map[string]string{"username": "admin", "password": "test-password"}); code != 200 {
		t.Fatalf("登录失败 %d %s", code, body)
	}
	return a, srv, c
}

func uploadFiles(t *testing.T, c *http.Client, url, field string, files map[string][]byte) (int, []byte) {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	for name, data := range files {
		fw, _ := mw.CreateFormFile(field, name)
		_, _ = fw.Write(data)
	}
	_ = mw.Close()
	req, _ := http.NewRequest("POST", url, &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	resp, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, body
}

func zipOf(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, content := range files {
		w, _ := zw.Create(name)
		_, _ = w.Write([]byte(content))
	}
	_ = zw.Close()
	return buf.Bytes()
}

func TestImportZipKeepEnabledAndDedupe(t *testing.T) {
	_, srv, c := loginApp(t)
	port := freeTCPPort(t)
	pf := `{"PortForwardsRuleList":[{"Name":"web","Enable":true,"ForwardTypes":["tcp4"],"ListenAddress":"127.0.0.1","ListenPorts":"` + strconv.Itoa(port) + `","TargetAddressList":["127.0.0.1"],"TargetPorts":"80","Options":{}}]}`
	base := `{"AdminWebListenPort":16601,"AllowInternetaccess":true,"MaxConsecutiveLoginFailures":8,"TimeZone":"Asia/Shanghai","SafeURL":"secret"}`
	archive := zipOf(t, map[string]string{
		"lucky_conf/lucky_portforward.lkcf": pf,
		"lucky_conf/lucky_base.lkcf":        base,
		"lucky_conf/readme.txt":             "ignored",
		"__MACOSX/._lucky_base.lkcf":        "junk",
	})
	code, body := uploadFiles(t, c, srv.URL+"/api/import/lucky?enable=1", "files", map[string][]byte{"backup.zip": archive})
	if code != 200 || !bytes.Contains(body, []byte(`"imported":1`)) || bytes.Contains(body, []byte(`"error"`)) {
		t.Fatalf("zip 导入失败 %d %s", code, body)
	}
	_, body = doJSON(t, c, "GET", srv.URL+"/api/portforward/rules", nil)
	var views []portforward.RuleView
	_ = json.Unmarshal(body, &views)
	if len(views) != 1 || !views[0].Enable {
		t.Fatalf("开启“保留启用状态”后规则应为启用: %s", body)
	}
	// 基础设置：登录失败上限与时区被迁移，管理端口与安全入口不迁移
	_, body = doJSON(t, c, "GET", srv.URL+"/api/system/settings", nil)
	var s settingsView
	_ = json.Unmarshal(body, &s)
	if s.MaxLoginFailures != 8 || s.TimeZone != "Asia/Shanghai" || s.SafePath != "" || s.Listen != DefaultListen {
		t.Fatalf("基础设置迁移错误: %s", body)
	}
	// 重复导入应跳过同名规则
	code, body = uploadFiles(t, c, srv.URL+"/api/import/lucky", "files", map[string][]byte{"lucky_portforward.lkcf": []byte(pf)})
	if code != 200 || !bytes.Contains(body, []byte("已存在同名条目")) {
		t.Fatalf("重复导入应跳过: %s", body)
	}
	_, body = doJSON(t, c, "GET", srv.URL+"/api/portforward/rules", nil)
	_ = json.Unmarshal(body, &views)
	if len(views) != 1 {
		t.Fatalf("重复导入不应产生重复规则: %d", len(views))
	}
	// 压缩包中没有配置文件时报错
	_, body = uploadFiles(t, c, srv.URL+"/api/import/lucky", "files", map[string][]byte{"empty.zip": zipOf(t, map[string]string{"a.txt": "x"})})
	if !bytes.Contains(body, []byte("没有找到")) {
		t.Fatalf("空压缩包应提示: %s", body)
	}
}

func TestSettingsSafePathAndToken(t *testing.T) {
	_, srv, c := loginApp(t)
	_, body := doJSON(t, c, "GET", srv.URL+"/api/system/settings", nil)
	var s settingsView
	_ = json.Unmarshal(body, &s)
	s.SafePath = "my-entry"
	s.APIToken = "0123456789abcdef0123"
	s.SessionHours = 24
	if code, body := doJSON(t, c, "PUT", srv.URL+"/api/system/settings", s); code != 200 {
		t.Fatalf("保存设置失败 %d %s", code, body)
	}
	_, body = doJSON(t, c, "GET", srv.URL+"/api/system/settings", nil)
	_ = json.Unmarshal(body, &s)
	if s.SafePath != "/my-entry" || s.APIToken != "********" || s.SessionHours != 24 {
		t.Fatalf("设置未生效或令牌未脱敏: %s", body)
	}

	// 新浏览器：未经过安全入口时首页与登录接口均返回 404
	jar, _ := cookiejar.New(nil)
	fresh := &http.Client{Jar: jar, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	if code, _ := doJSON(t, fresh, "GET", srv.URL+"/", nil); code != 404 {
		t.Fatalf("未经过安全入口应返回 404，实际 %d", code)
	}
	if code, _ := doJSON(t, fresh, "POST", srv.URL+"/api/login", map[string]string{"username": "admin", "password": "test-password"}); code != 404 {
		t.Fatalf("未经过安全入口时登录接口应返回 404，实际 %d", code)
	}
	if code, _ := doJSON(t, fresh, "GET", srv.URL+"/my-entry", nil); code != 302 {
		t.Fatalf("访问安全入口应跳转，实际 %d", code)
	}
	if code, _ := doJSON(t, fresh, "POST", srv.URL+"/api/login", map[string]string{"username": "admin", "password": "test-password"}); code != 200 {
		t.Fatalf("经过安全入口后应能登录，实际 %d", code)
	}

	// API 令牌可直接调用接口（不需要安全入口与会话）
	req, _ := http.NewRequest("GET", srv.URL+"/api/system/info", nil)
	req.Header.Set("Authorization", "Bearer 0123456789abcdef0123")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("API 令牌应可访问，实际 %d", resp.StatusCode)
	}
	req.Header.Set("Authorization", "Bearer wrong-token-wrong-token")
	resp, _ = http.DefaultClient.Do(req)
	resp.Body.Close()
	if resp.StatusCode != 404 {
		t.Fatalf("错误令牌应被安全入口拦截，实际 %d", resp.StatusCode)
	}

	// 非法设置被拒绝
	s.SafePath = "/api"
	if code, _ := doJSON(t, c, "PUT", srv.URL+"/api/system/settings", s); code != 400 {
		t.Fatalf("非法安全入口应被拒绝，实际 %d", code)
	}
}

func TestBackupRestore(t *testing.T) {
	a, srv, c := loginApp(t)
	rule := map[string]any{"name": "web", "enable": false, "protocols": []string{"tcp4"}, "listenAddress": "127.0.0.1",
		"listenPorts": strconv.Itoa(freeTCPPort(t)), "targetAddresses": []string{"127.0.0.1"}, "targetPorts": "80"}
	if code, body := doJSON(t, c, "POST", srv.URL+"/api/portforward/rules", rule); code != 200 {
		t.Fatalf("%d %s", code, body)
	}
	resp, err := c.Get(srv.URL + "/api/system/backup")
	if err != nil {
		t.Fatal(err)
	}
	backup, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	zr, err := zip.NewReader(bytes.NewReader(backup), int64(len(backup)))
	if err != nil {
		t.Fatal(err)
	}
	names := map[string]bool{}
	for _, f := range zr.File {
		names[f.Name] = true
	}
	if !names["account.json"] || !names["portforward.json"] {
		t.Fatalf("备份内容不完整: %v", names)
	}
	// 恢复会触发重启信号
	code, body := uploadFiles(t, c, srv.URL+"/api/system/restore", "file", map[string][]byte{"b.zip": backup})
	if code != 200 || !bytes.Contains(body, []byte(`"restored"`)) {
		t.Fatalf("恢复失败 %d %s", code, body)
	}
	select {
	case <-a.restartCh:
	case <-time.After(3 * time.Second):
		t.Fatal("恢复后应请求重启")
	}
	// 缺少账号文件的压缩包被拒绝
	code, _ = uploadFiles(t, c, srv.URL+"/api/system/restore", "file", map[string][]byte{"b.zip": zipOf(t, map[string]string{"x.json": "{}"})})
	if code != 400 {
		t.Fatalf("非 Gately 备份应被拒绝，实际 %d", code)
	}
}

func TestIsPrivate(t *testing.T) {
	for ip, want := range map[string]bool{"192.168.1.2": true, "10.0.0.1": true, "100.64.0.3": true, "127.0.0.1": true, "fd00::1": true, "8.8.8.8": false, "2408:8000::1": false} {
		if isPrivate(ip) != want {
			t.Errorf("isPrivate(%s) != %v", ip, want)
		}
	}
}

// 重启时应先停止全部模块（含有后台协程的证书模块），再重新执行进程
func TestRunRestartStopsModules(t *testing.T) {
	called := make(chan struct{})
	old := restartProcess
	restartProcess = func() { close(called) }
	defer func() { restartProcess = old }()
	certs := ssl.New()
	a, err := New(Options{ConfigDir: t.TempDir(), Listen: "127.0.0.1:" + strconv.Itoa(freeTCPPort(t)), Modules: []module.Module{certs, portforward.New()}})
	if err != nil {
		t.Fatal(err)
	}
	a.logger = log.New(io.Discard, "", 0)
	done := make(chan error, 1)
	go func() { done <- a.Run(context.Background()) }()
	time.Sleep(300 * time.Millisecond)
	a.scheduleRestart()
	select {
	case <-called:
	case <-time.After(15 * time.Second):
		t.Fatal("重启卡住：模块未能停止")
	}
	<-done
}
