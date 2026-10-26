package cloudflared

import (
	"testing"

	"github.com/wangwanjie/gately/internal/binstall"
)

func TestAssetName(t *testing.T) {
	cases := map[binstall.Platform]string{
		{OS: "linux", Arch: "amd64"}:         "cloudflared-linux-amd64",
		{OS: "linux", Arch: "arm", ARM: "7"}: "cloudflared-linux-armhf",
		{OS: "linux", Arch: "arm", ARM: "6"}: "cloudflared-linux-arm",
		{OS: "darwin", Arch: "arm64"}:        "cloudflared-darwin-arm64.tgz",
		{OS: "windows", Arch: "amd64"}:       "cloudflared-windows-amd64.exe",
	}
	for p, want := range cases {
		if got, err := assetName(p); err != nil || got != want {
			t.Errorf("%v: %q %v，期望 %q", p, got, err, want)
		}
	}
	for _, p := range []binstall.Platform{{OS: "linux", Arch: "mipsle"}, {OS: "freebsd", Arch: "amd64"}, {OS: "windows", Arch: "arm64"}} {
		if _, err := assetName(p); err == nil {
			t.Errorf("%v 上游没有提供，应报错", p)
		}
	}
	if m := cloudflaredVersionRe.FindStringSubmatch("cloudflared version 2026.10.3 (built 2026-10-21-1200 UTC)"); m == nil || m[1] != "2026.10.3" {
		t.Errorf("版本解析错误: %v", m)
	}
}
