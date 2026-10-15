package dnsprovider

import "testing"

func TestSplitFQDN(t *testing.T) {
	cases := map[string][2]string{
		"example.com":                     {"@", "example.com"},
		"a.b.example.com":                 {"a.b", "example.com"},
		"www.example.com.cn":              {"www", "example.com.cn"},
		"example.com.cn":                  {"@", "example.com.cn"},
		"x:sub.example.co.uk":             {"x", "sub.example.co.uk"},
		"*.Example.com.":                  {"*", "example.com"},
		"_acme-challenge.a.example.co.uk": {"_acme-challenge.a", "example.co.uk"},
	}
	for in, want := range cases {
		s, z := SplitFQDN(in)
		if s != want[0] || z != want[1] {
			t.Errorf("SplitFQDN(%q)=%q,%q want %v", in, s, z, want)
		}
	}
	if s, ok := SubForZone("_acme-challenge.www.example.com.", "example.com"); !ok || s != "_acme-challenge.www" {
		t.Fatalf("SubForZone 错误: %q %v", s, ok)
	}
	if _, ok := SubForZone("a.other.com", "example.com"); ok {
		t.Fatal("不属于该主域名应返回 false")
	}
}
