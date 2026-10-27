package webterminal

import (
	"net"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/wangwanjie/gately/internal/auth"
)

// TestCredentialPreset 覆盖认证预设：脱敏输出、连接引用预设完成认证、默认用户名、被引用时拒绝删除。
func TestCredentialPreset(t *testing.T) {
	keyPEM, pub := newKey(t)
	addr, _ := startSSHServer(t, pub)
	host, port, _ := net.SplitHostPort(addr)
	pn, _ := strconv.Atoi(port)
	m, srv := newModule(t)

	code, out := call(t, srv, "POST", "/credentials", map[string]any{"name": "nas key", "authType": "key", "username": "root", "privateKey": keyPEM})
	if code != http.StatusOK {
		t.Fatalf("新增预设失败: %d %v", code, out)
	}
	if out["privateKey"] != "********" {
		t.Fatalf("预设私钥应脱敏输出: %v", out["privateKey"])
	}
	credKey, _ := out["key"].(string)

	// 无效私钥应被拒绝
	if code, _ := call(t, srv, "POST", "/credentials", map[string]any{"name": "bad", "authType": "key", "privateKey": "garbage"}); code != http.StatusBadRequest {
		t.Fatalf("无效私钥应返回 400，实际 %d", code)
	}

	// 连接未填用户名，使用预设中的默认用户名；内联密码会被清空
	code, out = call(t, srv, "POST", "/connections", map[string]any{"name": "nas", "type": "ssh",
		"ssh": map[string]any{"host": host, "port": pn, "authType": AuthCredential, "credentialKey": credKey, "password": "leftover"}})
	if code != http.StatusOK {
		t.Fatalf("新增连接失败: %d %v", code, out)
	}
	connKey, _ := out["key"].(string)
	m.mu.Lock()
	inline := m.cfg.Connections[0].SSH.Password
	m.mu.Unlock()
	if inline != "" {
		t.Fatal("使用预设时不应保留连接内的密码")
	}

	code, out = call(t, srv, "POST", "/connections/"+connKey+"/test", nil)
	if code != http.StatusOK || out["ok"] != true {
		t.Fatalf("经预设认证的连接测试应成功: %d %v", code, out)
	}

	code, out = call(t, srv, "DELETE", "/credentials/"+credKey, nil)
	if code != http.StatusConflict || !strings.Contains(out["error"].(string), "nas") {
		t.Fatalf("被引用的预设应拒绝删除并列出连接: %d %v", code, out)
	}

	// 删除连接后预设可删除；之后引用失效的连接报错
	m.mu.Lock()
	saved := m.cfg.Connections[0]
	m.mu.Unlock()
	if code, _ := call(t, srv, "DELETE", "/connections/"+connKey, nil); code != http.StatusOK {
		t.Fatalf("删除连接失败: %d", code)
	}
	if code, _ := call(t, srv, "DELETE", "/credentials/"+credKey, nil); code != http.StatusOK {
		t.Fatalf("删除预设失败: %d", code)
	}
	if _, _, err := m.dialSSH(t.Context(), &saved); err == nil || !strings.Contains(err.Error(), "认证预设不存在") {
		t.Fatalf("预设缺失时应提示: %v", err)
	}
}

// TestSecretsEndpoints 验证连接与认证预设的查看明文接口返回被脱敏字段的真实值，条目不存在时返回 404。
func TestSecretsEndpoints(t *testing.T) {
	m, srv := newModule(t)
	m.mu.Lock()
	m.cfg.Connections = []Connection{{Key: "c", Name: "nas", Type: "ssh"}}
	m.cfg.Connections[0].SSH.Password = "sp"
	m.cfg.Connections[0].SSH.Proxy.Password = "pp"
	m.cfg.Connections[0].Telnet.Password = "tp"
	m.cfg.Credentials = []Credential{{Key: "k", Name: "cred", AuthType: "password", Password: "cp", Passphrase: "ph"}}
	m.mu.Unlock()

	code, out := call(t, srv, "GET", "/connections/c/secrets", nil)
	if code != http.StatusOK || out["ssh.password"] != "sp" || out["ssh.proxy.password"] != "pp" || out["telnet.password"] != "tp" {
		t.Fatalf("连接应返回密码明文: %d %v", code, out)
	}
	code, out = call(t, srv, "GET", "/credentials/k/secrets", nil)
	if code != http.StatusOK || out["password"] != "cp" || out["passphrase"] != "ph" {
		t.Fatalf("认证预设应返回密码明文: %d %v", code, out)
	}
	if code, _ := call(t, srv, "GET", "/connections/missing/secrets", nil); code != http.StatusNotFound {
		t.Fatalf("连接不存在应返回 404，实际 %d", code)
	}
	if code, _ := call(t, srv, "GET", "/credentials/missing/secrets", nil); code != http.StatusNotFound {
		t.Fatalf("认证预设不存在应返回 404，实际 %d", code)
	}
}

// TestSecretsRequireTwoFA 验证开启终端二次验证后，未验证的会话不能取回连接与预设的密码明文，验证后可以。
func TestSecretsRequireTwoFA(t *testing.T) {
	m, srv := newModule(t)
	m.mu.Lock()
	m.cfg.Connections = []Connection{{Key: "c", Name: "nas", Type: "ssh"}}
	m.cfg.Connections[0].SSH.Password = "sp"
	m.cfg.Credentials = []Credential{{Key: "k", Name: "cred", AuthType: "password", Password: "cp"}}
	m.cfg.Security.TwoFAEnable = true
	m.cfg.Security.TwoFAKey = auth.NewTOTPSecret()
	m.mu.Unlock()

	for _, p := range []string{"/connections/c/secrets", "/credentials/k/secrets"} {
		code, out := call(t, srv, "GET", p, nil)
		if code != http.StatusForbidden || out["need2fa"] != true {
			t.Fatalf("%s 未通过二次验证应返回 403: %d %v", p, code, out)
		}
	}
	if code, _ := call(t, srv, "POST", "/security/verify2fa", map[string]string{"code": totpNow(m.cfg.Security.TwoFAKey)}); code != http.StatusOK {
		t.Fatalf("二次验证失败: %d", code)
	}
	if code, out := call(t, srv, "GET", "/connections/c/secrets", nil); code != http.StatusOK || out["ssh.password"] != "sp" {
		t.Fatalf("通过二次验证后应返回明文: %d %v", code, out)
	}
}
