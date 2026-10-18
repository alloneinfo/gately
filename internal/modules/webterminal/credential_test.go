package webterminal

import (
	"net"
	"net/http"
	"strconv"
	"strings"
	"testing"
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
