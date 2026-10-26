package frp

import (
	"testing"

	"github.com/wangwanjie/gately/internal/binstall"
)

func TestAssetName(t *testing.T) {
	cases := map[binstall.Platform]string{
		{OS: "linux", Arch: "amd64"}:         "frp_0.71.1_linux_amd64.tar.gz",
		{OS: "linux", Arch: "arm", ARM: "7"}: "frp_0.71.1_linux_arm_hf.tar.gz",
		{OS: "linux", Arch: "arm", ARM: "6"}: "frp_0.71.1_linux_arm.tar.gz",
		{OS: "linux", Arch: "mipsle"}:        "frp_0.71.1_linux_mipsle.tar.gz",
		{OS: "darwin", Arch: "arm64"}:        "frp_0.71.1_darwin_arm64.tar.gz",
		{OS: "freebsd", Arch: "amd64"}:       "frp_0.71.1_freebsd_amd64.tar.gz",
		{OS: "windows", Arch: "amd64"}:       "frp_0.71.1_windows_amd64.zip",
	}
	for p, want := range cases {
		if got, err := assetName("0.71.1", p); err != nil || got != want {
			t.Errorf("%v: %q %v，期望 %q", p, got, err, want)
		}
	}
	for _, p := range []binstall.Platform{{OS: "linux", Arch: "386"}, {OS: "windows", Arch: "386"}, {OS: "darwin", Arch: "386"}} {
		if _, err := assetName("0.71.1", p); err == nil {
			t.Errorf("%v 上游没有提供，应报错", p)
		}
	}
}

func TestFrpVersion(t *testing.T) {
	if m := frpVersionRe.FindStringSubmatch("0.71.1\n"); m == nil || m[1] != "0.71.1" {
		t.Errorf("版本解析错误: %v", m)
	}
}
