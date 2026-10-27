package ssl

import (
	"errors"
	"testing"
)

// TestDNSSolverLocate 覆盖主域名留空时按 SOA 区域定位验证记录、查询失败时的回退与显式主域名。
func TestDNSSolverLocate(t *testing.T) {
	orig := findZone
	defer func() { findZone = orig }()
	zones := map[string]string{
		// 子域名单独托管（NS 委派）在另一个服务商
		"_acme-challenge.web.test.com":     "web.test.com",
		"_acme-challenge.a.example.com.cn": "example.com.cn",
	}
	findZone = func(fqdn string) (string, error) {
		if z, ok := zones[fqdn]; ok {
			return z, nil
		}
		return "", errors.New("no soa")
	}
	cases := []struct {
		solverZone, domain, zone, sub string
	}{
		{"", "web.test.com", "web.test.com", "_acme-challenge"},
		{"", "a.example.com.cn", "example.com.cn", "_acme-challenge.a"},
		// SOA 查询失败时按域名段数推断
		{"", "nas.other.com", "other.com", "_acme-challenge.nas"},
		// 显式指定主域名时不查询
		{"test.com", "web.test.com", "test.com", "_acme-challenge.web"},
	}
	for _, c := range cases {
		s := &dnsSolver{zone: c.solverZone}
		zone, sub, _, err := s.locate(c.domain, "key")
		if err != nil {
			t.Fatalf("%s: %v", c.domain, err)
		}
		if zone != c.zone || sub != c.sub {
			t.Fatalf("%s: 得到 %s / %s，期望 %s / %s", c.domain, zone, sub, c.zone, c.sub)
		}
	}
}
