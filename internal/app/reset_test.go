package app

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/wangwanjie/gately/internal/auth"
)

// TestAccountResetFromCLI 模拟 gately reset-password：写入请求文件后由运行中的实例应用。
func TestAccountResetFromCLI(t *testing.T) {
	a, srv, c := loginApp(t)
	if _, body := doJSON(t, c, "GET", srv.URL+"/api/session", nil); !bytes.Contains(body, []byte(`"defaultPassword":false`)) {
		t.Fatalf("已修改密码时不应提示默认密码: %s", body)
	}

	req, err := auth.NewPasswordReset("")
	if err != nil {
		t.Fatal(err)
	}
	if err := auth.WriteResetRequest(a.opts.ConfigDir, req); err != nil {
		t.Fatal(err)
	}
	a.applyAccountReset()
	if auth.ResetPending(a.opts.ConfigDir) {
		t.Fatal("应用后应删除请求文件")
	}

	if code, _ := doJSON(t, c, "GET", srv.URL+"/api/session", nil); code != http.StatusUnauthorized {
		t.Fatalf("重置后旧会话应失效: %d", code)
	}
	if code, body := doJSON(t, c, "POST", srv.URL+"/api/login", map[string]string{"username": "admin", "password": auth.DefaultPassword}); code != 200 {
		t.Fatalf("重置后应能用默认密码登录: %d %s", code, body)
	}
	if _, body := doJSON(t, c, "GET", srv.URL+"/api/session", nil); !bytes.Contains(body, []byte(`"defaultPassword":true`)) {
		t.Fatalf("重置为默认密码后应提示修改: %s", body)
	}
}
