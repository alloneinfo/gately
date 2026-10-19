package ddns

import (
	"context"
	"net"
	"strings"
	"time"

	"github.com/wangwanjie/gately/internal/module"
)

// SelfCheck 实现 module.SelfChecker：检查使用动态地址变量的记录，其域名当前解析结果是否等于本机出口公网地址。
func (m *Module) SelfCheck(ctx context.Context, env module.CheckEnv) []module.CheckResult {
	type target struct{ fqdn, typ, want string }
	var targets []target
	m.mu.Lock()
	for _, t := range m.cfg.Tasks {
		if !t.Enable {
			continue
		}
		for _, rec := range t.Records {
			if !rec.Enable {
				continue
			}
			fqdn := rec.Zone
			if rec.Sub != "" && rec.Sub != "@" {
				fqdn = rec.Sub + "." + rec.Zone
			}
			switch {
			case rec.Type == "A" && strings.Contains(rec.Value, "{ipv4Addr}") && env.PublicIPv4 != "":
				targets = append(targets, target{fqdn, "ip4", env.PublicIPv4})
			case rec.Type == "AAAA" && strings.Contains(rec.Value, "{ipv6Addr}") && env.PublicIPv6 != "":
				targets = append(targets, target{fqdn, "ip6", env.PublicIPv6})
			}
		}
	}
	m.mu.Unlock()

	var out []module.CheckResult
	seen := map[string]bool{}
	for _, tg := range targets {
		key := tg.fqdn + "|" + tg.typ
		if seen[key] {
			continue
		}
		seen[key] = true
		lctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		addrs, err := net.DefaultResolver.LookupIP(lctx, tg.typ, tg.fqdn)
		cancel()
		params := map[string]string{"domain": tg.fqdn, "expected": tg.want}
		if err != nil || len(addrs) == 0 {
			out = append(out, module.CheckResult{ID: "ddns.unresolved", Status: module.CheckWarn, Params: params})
			continue
		}
		var got []string
		match := false
		for _, a := range addrs {
			got = append(got, a.String())
			if a.String() == tg.want {
				match = true
			}
		}
		params["resolved"] = strings.Join(got, ", ")
		if match {
			out = append(out, module.CheckResult{ID: "ddns.match", Status: module.CheckOK, Params: params})
		} else {
			out = append(out, module.CheckResult{ID: "ddns.mismatch", Status: module.CheckWarn, Params: params})
		}
	}
	return out
}
