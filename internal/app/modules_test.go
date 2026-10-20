package app

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/wangwanjie/gately/internal/module"
	"github.com/wangwanjie/gately/internal/modules/portforward"
	"github.com/wangwanjie/gately/internal/modules/wol"
)

// startModulesApp 在 dir 上启动带端口转发与网络唤醒模块的实例并登录；测试中假定网络唤醒依赖端口转发。
// 同一 dir 再次调用相当于重启，用于验证开关在重启后生效。
func startModulesApp(t *testing.T, dir string) (*App, *httptest.Server, *http.Client) {
	t.Helper()
	a, err := New(Options{
		ConfigDir:       dir,
		Modules:         []module.Module{portforward.New(), wol.New()},
		Requires:        map[string][]string{wol.ID: {portforward.ID}},
		InitialPassword: "test-password",
	})
	if err != nil {
		t.Fatal(err)
	}
	a.logger = log.New(io.Discard, "", 0)
	srv := httptest.NewServer(a.Handler())
	t.Cleanup(srv.Close)
	jar, _ := cookiejar.New(nil)
	c := &http.Client{Jar: jar}
	if code, body := doJSON(t, c, "POST", srv.URL+"/api/login", map[string]string{"username": "admin", "password": "test-password"}); code != 200 {
		t.Fatalf("登录失败 %d %s", code, body)
	}
	return a, srv, c
}

func TestDisableModules(t *testing.T) {
	dir := t.TempDir()
	_, srv, c := startModulesApp(t, dir)
	_, body := doJSON(t, c, "GET", srv.URL+"/api/system/settings", nil)
	var s settingsView
	_ = json.Unmarshal(body, &s)
	if len(s.Modules) != 2 || len(s.DisabledModules) != 0 {
		t.Fatalf("默认应全部开启并列出模块: %s", body)
	}

	// 被依赖的模块不能关闭，未知模块也被拒绝
	s.DisabledModules = []string{portforward.ID}
	if code, body := doJSON(t, c, "PUT", srv.URL+"/api/system/settings", s); code != http.StatusBadRequest {
		t.Fatalf("关闭被依赖的模块应失败: %d %s", code, body)
	}
	s.DisabledModules = []string{"nope"}
	if code, _ := doJSON(t, c, "PUT", srv.URL+"/api/system/settings", s); code != http.StatusBadRequest {
		t.Fatalf("未知模块应被拒绝: %d", code)
	}

	// 依赖方与被依赖方一起关闭是允许的，保存后提示重启，当前进程内仍可使用
	s.DisabledModules = []string{wol.ID, portforward.ID}
	code, body := doJSON(t, c, "PUT", srv.URL+"/api/system/settings", s)
	if code != 200 {
		t.Fatalf("关闭模块失败 %d %s", code, body)
	}
	_ = json.Unmarshal(body, &s)
	if !s.RestartRequired {
		t.Fatalf("修改模块开关应提示重启: %s", body)
	}
	if code, _ := doJSON(t, c, "GET", srv.URL+"/api/wol/devices", nil); code != 200 {
		t.Fatalf("重启前模块应仍可访问: %d", code)
	}

	// 模拟重启：关闭的模块不再提供接口，状态与会话中标记为关闭，接口文档中不再出现
	a2, srv2, c2 := startModulesApp(t, dir)
	code, body = doJSON(t, c2, "GET", srv2.URL+"/api/wol/devices", nil)
	if code != http.StatusForbidden {
		t.Fatalf("关闭的模块应返回 403: %d %s", code, body)
	}
	if code, _ := doJSON(t, c2, "GET", srv2.URL+"/api/portforward/logs", nil); code != http.StatusForbidden {
		t.Fatalf("关闭模块的日志接口也应不可用: %d", code)
	}
	_, body = doJSON(t, c2, "GET", srv2.URL+"/api/session", nil)
	if !bytes.Contains(body, []byte(`"disabledModules":["portforward","wol"]`)) {
		t.Fatalf("会话应返回关闭的模块: %s", body)
	}
	_, body = doJSON(t, c2, "GET", srv2.URL+"/api/status", nil)
	var st struct {
		Modules []moduleInfo `json:"modules"`
	}
	_ = json.Unmarshal(body, &st)
	if len(st.Modules) != 2 || st.Modules[0].Enabled || st.Modules[1].Enabled {
		t.Fatalf("状态中应标记模块关闭: %s", body)
	}
	for _, r := range a2.routes {
		if r.Module == wol.ID || r.Module == portforward.ID {
			t.Fatalf("关闭模块的接口不应出现在路由表: %+v", r)
		}
	}

	// 关闭的模块不导入 Lucky 配置，只报告未导入
	lucky := `{"PortForwardsRuleList":[{"Name":"ssh","Enable":false,"ForwardTypes":["tcp"],"ListenPorts":"6022","TargetAddressList":["192.168.1.2"],"TargetPorts":"22","Options":{}}]}`
	code, body = uploadFiles(t, c2, srv2.URL+"/api/import/lucky", "files", map[string][]byte{"lucky_portforward.lkcf": []byte(lucky)})
	if code != 200 || !bytes.Contains(body, []byte(`"found":1,"imported":0`)) {
		t.Fatalf("关闭的模块应跳过导入: %d %s", code, body)
	}
}

// 配置文件中关闭了仍被依赖的模块时，启动时保持该模块开启
func TestDisabledModuleKeptForDependents(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, settingsName+".json"), []byte(`{"disabledModules":["portforward"]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	a, _, _ := startModulesApp(t, dir)
	if ids := a.disabledModuleIDs(); len(ids) != 0 {
		t.Fatalf("被依赖的模块应保持开启: %v", ids)
	}
}
