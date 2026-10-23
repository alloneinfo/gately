package systz

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParsePOSIX(t *testing.T) {
	cases := map[string]int{
		"CST-8":        8 * 3600,
		"UTC0":         0,
		"EST5":         -5 * 3600,
		"<+0545>-5:45": 5*3600 + 45*60,
		"IST-5:30":     5*3600 + 30*60,
	}
	for s, want := range cases {
		loc := parsePOSIX(s)
		if loc == nil {
			t.Errorf("parsePOSIX(%q) = nil", s)
			continue
		}
		if _, off := time.Now().In(loc).Zone(); off != want {
			t.Errorf("parsePOSIX(%q) 偏移 %d，期望 %d", s, off, want)
		}
	}
	// 带夏令时规则或格式错误的字符串不处理
	for _, s := range []string{"CET-1CEST,M3.5.0,M10.5.0/3", "", "8", "CS-8", "CST-8:xx"} {
		if parsePOSIX(s) != nil {
			t.Errorf("parsePOSIX(%q) 应返回 nil", s)
		}
	}
}

func TestDetect(t *testing.T) {
	dir := t.TempDir()
	uciSystemPath = filepath.Join(dir, "system")
	posixTZPath = filepath.Join(dir, "TZ")
	t.Cleanup(func() {
		uciSystemPath, posixTZPath = "/etc/config/system", "/etc/TZ"
	})
	os.WriteFile(posixTZPath, []byte("CST-8\n"), 0o644)

	// 有 zonename 时使用 IANA 名称（依赖内嵌或系统的时区数据）
	os.WriteFile(uciSystemPath, []byte("config system\n\toption hostname 'ImmortalWrt'\n\toption zonename 'Asia/Shanghai'\n\toption timezone 'CST-8'\n\nconfig timeserver 'ntp'\n"), 0o644)
	if loc := detect(); loc == nil || loc.String() != "Asia/Shanghai" {
		t.Fatalf("detect() = %v，期望 Asia/Shanghai", loc)
	}

	// 只有 /etc/TZ 时按固定偏移处理
	os.WriteFile(uciSystemPath, []byte("config system\n\toption hostname 'ImmortalWrt'\n"), 0o644)
	loc := detect()
	if loc == nil {
		t.Fatal("detect() = nil")
	}
	if _, off := time.Now().In(loc).Zone(); off != 8*3600 {
		t.Fatalf("偏移 %d，期望 28800", off)
	}
}
