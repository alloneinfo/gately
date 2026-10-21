package iplist

import (
	"net/netip"
	"strings"
	"testing"
)

func TestParseAndContains(t *testing.T) {
	text := `# 广东省 IP 列表
1.2.3.0/24
1.2.4.0/24 相邻网段，应与上一行合并
10.0.0.1
192.168.1.10-192.168.1.20 ; 范围
::ffff:8.8.8.0/120
2001:db8::/32
2001:db8:1::/48 // 被上一行覆盖
not-an-ip
300.1.1.1/24
`
	set, skipped, err := Parse(strings.NewReader(text))
	if err != nil {
		t.Fatal(err)
	}
	if skipped != 2 || set.Len() != 7 {
		t.Fatalf("条目统计错误: len=%d skipped=%d", set.Len(), skipped)
	}
	if len(set.v4) != 4 || len(set.v6) != 1 {
		t.Fatalf("区间应被合并: v4=%v v6=%v", set.v4, set.v6)
	}
	in := []string{"1.2.3.0", "1.2.4.255", "10.0.0.1", "192.168.1.15", "192.168.1.20", "8.8.8.8", "::ffff:1.2.3.9", "2001:db8:ffff::1"}
	out := []string{"1.2.2.255", "1.2.5.0", "10.0.0.2", "192.168.1.21", "8.8.9.0", "2001:db9::", "::1", "0.0.0.0"}
	for _, s := range in {
		if !set.Contains(netip.MustParseAddr(s)) {
			t.Errorf("%s 应在列表中", s)
		}
	}
	for _, s := range out {
		if set.Contains(netip.MustParseAddr(s)) {
			t.Errorf("%s 不应在列表中", s)
		}
	}
	var nilSet *Set
	if nilSet.Contains(netip.MustParseAddr("1.2.3.4")) || nilSet.Len() != 0 {
		t.Fatal("空列表不应包含任何地址")
	}
}

func TestParseEmpty(t *testing.T) {
	if _, _, err := Parse(strings.NewReader("# 只有注释\n\n<html>")); err == nil {
		t.Fatal("没有有效条目时应报错，避免把网页错误页当成空列表")
	}
}

func TestRegistry(t *testing.T) {
	r := &Registry{}
	set, _, _ := Parse(strings.NewReader("1.1.1.1"))
	r.Store("a", set)
	if got, ok := r.Get("a"); !ok || !got.Contains(netip.MustParseAddr("1.1.1.1")) {
		t.Fatal("应能取回列表")
	}
	r.Store("a", nil)
	if _, ok := r.Get("a"); ok {
		t.Fatal("存入 nil 应删除列表")
	}
}

func BenchmarkContains(b *testing.B) {
	var sb strings.Builder
	for i := 0; i < 8000; i++ {
		sb.WriteString(netip.AddrFrom4([4]byte{byte(i >> 8), byte(i), 0, 0}).String() + "/24\n")
	}
	set, _, _ := Parse(strings.NewReader(sb.String()))
	a := netip.MustParseAddr("20.100.0.1")
	for b.Loop() {
		set.Contains(a)
	}
}
