package tailscale

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/wangwanjie/gately/internal/binstall"
)

// TestResolve 用本地服务模拟 pkgs.tailscale.com，验证最新版与指定版本的地址和摘要。
func TestResolve(t *testing.T) {
	const sum = "512f613ed7d04fc7282f7dc5bd76aa105aae28168fbb9357eb749ebcbc29f0ae"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/stable/" && r.URL.RawQuery == "mode=json":
			_, _ = w.Write([]byte(`{"TarballsVersion":"1.104.2","Tarballs":{"amd64":"tailscale_1.104.2_amd64.tgz"}}`))
		case strings.HasSuffix(r.URL.Path, "_amd64.tgz.sha256") && !strings.Contains(r.URL.Path, "9.9.9"):
			_, _ = w.Write([]byte(strings.ToUpper(sum) + "\n"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	old := pkgsURL
	pkgsURL = srv.URL + "/stable"
	defer func() { pkgsURL = old }()

	spec := installSpec()
	ctx := context.Background()
	amd64 := binstall.Platform{OS: "linux", Arch: "amd64"}
	rel, err := spec.Resolve(ctx, http.DefaultClient, amd64, "")
	if err != nil || rel.Version != "1.104.2" || rel.Asset.URL != pkgsURL+"/tailscale_1.104.2_amd64.tgz" || rel.Asset.SHA256 != sum {
		t.Fatalf("最新版解析错误: %+v %v", rel, err)
	}
	if rel, err := spec.Resolve(ctx, http.DefaultClient, amd64, "v1.90.2"); err != nil || rel.Asset.Name != "tailscale_1.90.2_amd64.tgz" {
		t.Errorf("指定版本解析错误: %+v %v", rel, err)
	}
	if _, err := spec.Resolve(ctx, http.DefaultClient, amd64, "9.9.9"); err == nil {
		t.Error("不存在的版本应报错")
	}
	if _, err := spec.Resolve(ctx, http.DefaultClient, amd64, "1.90.2; rm"); err == nil {
		t.Error("非法版本号应报错")
	}
	for _, p := range []binstall.Platform{{OS: "darwin", Arch: "arm64"}, {OS: "windows", Arch: "amd64"}, {OS: "linux", Arch: "loong64"}} {
		if err := spec.Check(p); err == nil {
			t.Errorf("%v 应提示不支持", p)
		}
	}
}
