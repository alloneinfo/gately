package geoip

import (
	"bytes"
	"net"
	"net/netip"
	"os"
	"path/filepath"
	"testing"

	"github.com/maxmind/mmdbwriter"
	"github.com/maxmind/mmdbwriter/mmdbtype"
)

// buildDB 生成含两个网段的测试库：1.0.0.0/24 → AU，2a00:1450::/32 → JP（使用 registered_country 回退）。
func buildDB(t *testing.T) []byte {
	t.Helper()
	w, err := mmdbwriter.New(mmdbwriter.Options{DatabaseType: "Test-Country", RecordSize: 24})
	if err != nil {
		t.Fatal(err)
	}
	_, n1, _ := net.ParseCIDR("1.0.0.0/24")
	if err := w.Insert(n1, mmdbtype.Map{"country": mmdbtype.Map{"iso_code": mmdbtype.String("au")}}); err != nil {
		t.Fatal(err)
	}
	_, n2, _ := net.ParseCIDR("2a00:1450::/32")
	if err := w.Insert(n2, mmdbtype.Map{"registered_country": mmdbtype.Map{"iso_code": mmdbtype.String("JP")}}); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if _, err := w.WriteTo(&buf); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestLookupAndFallback(t *testing.T) {
	path := filepath.Join(t.TempDir(), "c.mmdb")
	if _, err := Save(path, bytes.NewReader(buildDB(t))); err != nil {
		t.Fatal(err)
	}
	d := &DB{}
	if d.Country(netip.MustParseAddr("1.0.0.5")) != "" {
		t.Fatal("未加载时应返回空")
	}
	if err := d.Load(path); err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	cases := map[string]string{
		"1.0.0.5":        "AU",
		"::ffff:1.0.0.9": "AU",
		"2a00:1450::1":   "JP",
		"8.8.8.8":        "",
	}
	for ip, want := range cases {
		if got := d.Country(netip.MustParseAddr(ip)); got != want {
			t.Errorf("%s: got %q want %q", ip, got, want)
		}
	}
	if info := d.Info(); !info.Loaded || info.Type != "Test-Country" {
		t.Fatalf("Info 不符: %+v", info)
	}
}

func TestSaveRejectsInvalidAndKeepsOld(t *testing.T) {
	path := filepath.Join(t.TempDir(), "c.mmdb")
	good := buildDB(t)
	if _, err := Save(path, bytes.NewReader(good)); err != nil {
		t.Fatal(err)
	}
	if _, err := Save(path, bytes.NewReader([]byte("garbage"))); err == nil {
		t.Fatal("无效文件应被拒绝")
	}
	data, _ := os.ReadFile(path)
	if !bytes.Equal(data, good) {
		t.Fatal("校验失败不应覆盖旧文件")
	}
}

func TestIsLocal(t *testing.T) {
	for ip, want := range map[string]bool{"127.0.0.1": true, "192.168.1.1": true, "fd00::1": true, "8.8.8.8": false, "::ffff:10.0.0.1": true} {
		if IsLocal(netip.MustParseAddr(ip)) != want {
			t.Errorf("IsLocal(%s) != %v", ip, want)
		}
	}
}
