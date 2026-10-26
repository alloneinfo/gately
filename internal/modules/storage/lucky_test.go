package storage

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/wangwanjie/gately/internal/config"
	"github.com/wangwanjie/gately/internal/logbuf"
	"github.com/wangwanjie/gately/internal/luckyimport"
	"github.com/wangwanjie/gately/internal/module"
)

func TestImportLuckyStorage(t *testing.T) {
	st, _ := config.Open(t.TempDir())
	m := New()
	if err := m.Init(module.Env{Store: st, Logs: logbuf.New(20, log.New(io.Discard, "", 0))}); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	raw, _ := json.Marshal([]map[string]any{
		{"Key": "k1", "Remark": "share", "Enable": true, "Type": "local", "Writable": false, "Params": map[string]any{"LocalPath": dir},
			"SystemMount": map[string]any{"Enable": true, "MountPoint": "/mnt/x"}},
		{"Key": "k2", "Remark": "nas", "Enable": true, "Type": "webdav", "Writable": true,
			"Params": map[string]any{"WebDAVURL": "https://dav.example.com/", "WebDAVUser": "u", "WebDAVPasswd": "p", "InsecureSkipVerify": true, "Proxy": "socks5"}},
		{"Key": "k3", "Remark": "pan", "Enable": true, "Type": "aliyunpan", "Params": map[string]any{"AliyunpanRefreshToken": "r"}},
		{"Key": "k4", "Remark": "gone", "Enable": true, "Type": "local", "Params": map[string]any{"LocalPath": "/no/such/dir"}},
	})
	doc := luckyimport.Document{"List": raw, "AliyunpanConf": json.RawMessage(`{}`)}
	rep, err := m.ImportLucky(doc, luckyimport.Options{})
	if err != nil || rep == nil {
		t.Fatalf("导入失败: %v %v", rep, err)
	}
	if rep.Found != 4 || rep.Imported != 2 {
		t.Fatalf("统计错误: %+v", rep)
	}
	local, dav := m.cfg.Mounts[0], m.cfg.Mounts[1]
	if local.Key != "k1" || local.Name != "share" || local.Path != dir || !local.ReadOnly || local.Enable {
		t.Fatalf("本地存储映射错误: %+v", local)
	}
	if dav.URL != "https://dav.example.com/" || dav.Username != "u" || dav.Password != "p" || !dav.SkipVerify || dav.ReadOnly {
		t.Fatalf("WebDAV 存储映射错误: %+v", dav)
	}
	if p, ok := m.MountPath("k1"); !ok || p != dir {
		t.Fatal("应能按 Lucky 存储 Key 查到本地目录")
	}
	joined := strings.Join(rep.Warnings, "\n")
	for _, want := range []string{"系统目录", "socks5", "阿里云盘", "gone: 跳过"} {
		if !strings.Contains(joined, want) {
			t.Errorf("缺少提示 %q:\n%s", want, joined)
		}
	}
	if rep, _ := m.ImportLucky(doc, luckyimport.Options{}); rep.Imported != 0 {
		t.Fatal("重复导入应跳过同名存储")
	}
	if rep, _ := m.ImportLucky(luckyimport.Document{"List": raw}, luckyimport.Options{}); rep != nil {
		t.Fatal("缺少 AliyunpanConf 的配置不应被识别为存储管理")
	}
}

type secretsRouter struct{ mux *http.ServeMux }

func (tr secretsRouter) HandleFunc(p string, h http.HandlerFunc) {
	method, path, _ := strings.Cut(p, " ")
	tr.mux.HandleFunc(method+" /api/storage"+path, h)
}

// TestMountSecrets 验证查看存储密码明文接口返回真实密码，挂载不存在时返回 404。
func TestMountSecrets(t *testing.T) {
	st, _ := config.Open(t.TempDir())
	m := New()
	if err := m.Init(module.Env{Store: st, Logs: logbuf.New(20, log.New(io.Discard, "", 0))}); err != nil {
		t.Fatal(err)
	}
	m.cfg.Mounts = []Mount{{Key: "k", Name: "nas", Type: "webdav", Password: "pw"}}
	mux := http.NewServeMux()
	m.Routes(secretsRouter{mux})
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("GET", "/api/storage/mounts/k/secrets", nil))
	var got map[string]string
	_ = json.Unmarshal(rec.Body.Bytes(), &got)
	if rec.Code != 200 || got["password"] != "pw" {
		t.Fatalf("应返回密码明文: %d %s", rec.Code, rec.Body.String())
	}
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("GET", "/api/storage/mounts/missing/secrets", nil))
	if rec.Code != 404 {
		t.Fatalf("挂载不存在应返回 404，实际 %d", rec.Code)
	}
}
