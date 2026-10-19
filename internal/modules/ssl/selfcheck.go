package ssl

import (
	"context"
	"strconv"
	"time"

	"github.com/wangwanjie/gately/internal/module"
)

// SelfCheck 实现 module.SelfChecker：检查启用的证书是否已过期或即将过期。
func (m *Module) SelfCheck(_ context.Context, _ module.CheckEnv) []module.CheckResult {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []module.CheckResult
	now := time.Now()
	for i := range m.cfg.Certs {
		c := &m.cfg.Certs[i]
		if !c.Enable || c.CertPEM == "" {
			continue
		}
		info, err := certInfo(c.CertPEM)
		if err != nil {
			continue
		}
		days := int(info.NotAfter.Sub(now).Hours() / 24)
		params := map[string]string{"cert": label(c), "days": strconv.Itoa(days)}
		switch {
		case !info.NotAfter.After(now):
			out = append(out, module.CheckResult{ID: "cert.expired", Status: module.CheckFail, Params: params})
		case days < 14:
			out = append(out, module.CheckResult{ID: "cert.expiring", Status: module.CheckWarn, Params: params})
		default:
			out = append(out, module.CheckResult{ID: "cert.ok", Status: module.CheckOK, Params: params})
		}
	}
	return out
}
