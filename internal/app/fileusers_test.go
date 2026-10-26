package app

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/wangwanjie/gately/internal/config"
	"github.com/wangwanjie/gately/internal/module"
	"github.com/wangwanjie/gately/internal/modules/ftp"
	"github.com/wangwanjie/gately/internal/modules/webdav"
)

// TestFileUsersAPI 验证 /api/fileusers 读写共享用户、列出未关闭的文件服务，令牌需要 fileusers 权限，旧接口仍可读写同一份列表。
func TestFileUsersAPI(t *testing.T) {
	dir := t.TempDir()
	st, err := config.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	// 关闭 FTP 模块，共享用户接口不再列出它
	if err := st.Save(settingsName, map[string]any{"disabledModules": []string{"ftp"}}); err != nil {
		t.Fatal(err)
	}
	if err := st.Save("webdav", map[string]any{"port": 18800, "users": []map[string]any{{"name": "alice", "password": "p", "root": "/tmp"}}}); err != nil {
		t.Fatal(err)
	}
	a, err := New(Options{ConfigDir: dir, Modules: []module.Module{webdav.New(nil), ftp.New(nil)}})
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
	c := &http.Client{Jar: jar}
	if code, body := doJSON(t, c, "POST", srv.URL+"/api/login", map[string]string{"username": "admin", "password": "test-password"}); code != 200 {
		t.Fatalf("登录失败 %d %s", code, body)
	}

	type reply struct {
		Users []struct {
			Name     string `json:"name"`
			Password string `json:"password"`
		} `json:"users"`
		Services []struct {
			ID          string   `json:"id"`
			SharedUsers bool     `json:"sharedUsers"`
			LocalUsers  []string `json:"localUsers"`
		} `json:"services"`
	}
	users := []map[string]any{{"name": "bob", "password": "secret", "root": "/tmp"}}
	code, body := doJSON(t, c, "PUT", srv.URL+"/api/fileusers", map[string]any{"users": users})
	if code != 200 {
		t.Fatalf("保存共享用户失败 %d %s", code, body)
	}
	var r reply
	if err := json.Unmarshal(body, &r); err != nil {
		t.Fatal(err)
	}
	if len(r.Users) != 1 || r.Users[0].Name != "bob" || r.Users[0].Password == "secret" {
		t.Fatalf("应返回脱敏后的共享用户: %s", body)
	}
	if len(r.Services) != 1 || r.Services[0].ID != "webdav" || len(r.Services[0].LocalUsers) != 1 || r.Services[0].LocalUsers[0] != "alice" {
		t.Fatalf("应只列出未关闭的 WebDAV 及其本地用户: %s", body)
	}

	// 旧接口读写的是同一份列表
	code, body = doJSON(t, c, "GET", srv.URL+"/api/webdav/shared-users", nil)
	if err := json.Unmarshal(body, &r); code != 200 || err != nil || len(r.Users) != 1 || r.Users[0].Name != "bob" {
		t.Fatalf("旧接口应读到同一份共享用户 %d %s", code, body)
	}

	// 令牌：fileusers:write 可以保存，只有 webdav:write 不行
	_, ok := createToken(t, srv.URL, c, map[string]any{"name": "fu", "scopes": []string{"fileusers:write"}})
	if code, out := tokenRequest(t, srv.URL, "PUT", "/api/fileusers", ok, map[string]any{"users": users}); code != 200 {
		t.Fatalf("fileusers:write 应能保存 %d %s", code, out)
	}
	_, ro := createToken(t, srv.URL, c, map[string]any{"name": "ro", "scopes": []string{"fileusers:read"}})
	if code, out := tokenRequest(t, srv.URL, "GET", "/api/fileusers", ro, nil); code != 200 {
		t.Fatalf("fileusers:read 应能读取 %d %s", code, out)
	}
	if code, _ := tokenRequest(t, srv.URL, "PUT", "/api/fileusers", ro, map[string]any{"users": users}); code != http.StatusForbidden {
		t.Fatalf("fileusers:read 不能保存，实际 %d", code)
	}
	// 查看密码明文需要 fileusers:write，结果按用户名索引
	if code, out := doJSON(t, c, "GET", srv.URL+"/api/fileusers/secrets", nil); code != 200 || strings.TrimSpace(string(out)) != `{"bob":"secret"}` {
		t.Fatalf("应按用户名返回共享用户密码明文 %d %s", code, out)
	}
	if code, out := tokenRequest(t, srv.URL, "GET", "/api/fileusers/secrets", ok, nil); code != 200 {
		t.Fatalf("fileusers:write 应能查看密码明文 %d %s", code, out)
	}
	if code, _ := tokenRequest(t, srv.URL, "GET", "/api/fileusers/secrets", ro, nil); code != http.StatusForbidden {
		t.Fatalf("fileusers:read 不能查看密码明文，实际 %d", code)
	}
	_, wd := createToken(t, srv.URL, c, map[string]any{"name": "wd", "scopes": []string{"webdav:write"}})
	if code, _ := tokenRequest(t, srv.URL, "PUT", "/api/fileusers", wd, map[string]any{"users": users}); code != http.StatusForbidden {
		t.Fatalf("只有 webdav:write 不能保存，实际 %d", code)
	}
}
