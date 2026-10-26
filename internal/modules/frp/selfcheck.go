package frp

import (
	"context"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/wangwanjie/gately/internal/module"
)

// conflictWindow 是检查 frp 日志的时间范围；frpc 会定期重试启动失败的代理，冲突持续存在时总能在近期日志中看到。
const conflictWindow = 30 * time.Minute

var (
	// reProxyExists 匹配同名代理已被其他客户端注册，frpc 与 frps 的日志都会出现
	reProxyExists = regexp.MustCompile(`proxy \[([^\]]+)\] already exists`)
	// reStartError 匹配 frpc 启动代理失败的原因，reNewProxyError 匹配 frps 拒绝注册代理的原因，只关心域名与端口冲突
	reStartError    = regexp.MustCompile(`\[([^\]]+)\] start error: (.*)`)
	reNewProxyError = regexp.MustCompile(`new proxy \[([^\]]+)\] type \[[^\]]*\] error: (.*)`)
)

// conflictKeywords 是视为“与其他客户端冲突”的启动错误
var conflictKeywords = []string{"router config conflict", "port already used", "port unavailable"}

// SelfCheck 实现 module.SelfChecker：检查本机配置中的重名代理，以及近期日志中 frps 报告的代理名、域名或端口冲突。
// 多台设备复制同一份 frpc 配置连到同一个 frps 时，后连上的设备会注册失败，对应的网站表现为超时，很难直接看出原因。
func (m *Module) SelfCheck(_ context.Context, _ module.CheckEnv) []module.CheckResult {
	m.mu.Lock()
	var dup []string
	clientOn, serverOn := false, false
	enabled := 0
	for _, c := range m.cfg.Clients {
		if !c.Enable {
			continue
		}
		clientOn = true
		// frpc 不允许同一实例内代理重名；不同实例连接不同服务端，可以重名
		names := map[string]int{}
		for _, p := range m.cfg.Proxies {
			if p.Client == c.Key && p.Enable {
				names[p.Name]++
				enabled++
			}
		}
		for n, cnt := range names {
			if cnt > 1 {
				if len(m.cfg.Clients) > 1 {
					n = c.Name + " / " + n
				}
				dup = append(dup, n)
			}
		}
	}
	for _, s := range m.cfg.Servers {
		serverOn = serverOn || s.Enable
	}
	m.mu.Unlock()
	if !clientOn && !serverOn {
		return nil
	}
	var out []module.CheckResult
	sort.Strings(dup)
	for _, n := range dup {
		out = append(out, module.CheckResult{ID: "frp.duplicate", Status: module.CheckWarn, Params: map[string]string{"proxy": n}})
	}
	conflicts := m.recentConflicts(time.Now().Add(-conflictWindow))
	for _, c := range conflicts {
		out = append(out, module.CheckResult{ID: "frp.conflict", Status: module.CheckWarn, Params: map[string]string{"role": c.role, "proxy": c.proxy, "reason": c.reason}})
	}
	if clientOn && len(out) == 0 {
		out = append(out, module.CheckResult{ID: "frp.ok", Status: module.CheckOK, Params: map[string]string{"count": strconv.Itoa(enabled)}})
	}
	return out
}

type frpConflict struct {
	role, proxy, reason string
}

// recentConflicts 从 since 之后的 frpc / frps 日志中找出冲突的代理。同一个代理在 frpc 与 frps 的日志里都会出现，只报告一次。
func (m *Module) recentConflicts(since time.Time) []frpConflict {
	var out []frpConflict
	seen := map[string]bool{}
	add := func(role, proxy, reason string) {
		if !seen[proxy] {
			seen[proxy] = true
			out = append(out, frpConflict{role, proxy, reason})
		}
	}
	for _, e := range m.logs.Since(0) {
		// 日志来源为 frpc:<实例名> / frps:<实例名>，旧版本为 frpc / frps
		role, _, _ := strings.Cut(e.Source, ":")
		if e.Time.Before(since) || (role != binClient && role != binServer) {
			continue
		}
		if mm := reProxyExists.FindStringSubmatch(e.Message); mm != nil {
			add(role, mm[1], "proxy already exists")
			continue
		}
		mm := reStartError.FindStringSubmatch(e.Message)
		if role == binServer {
			// Gately 作为 frps 时，两台远程设备之间的端口、域名冲突只出现在 frps 日志里
			mm = reNewProxyError.FindStringSubmatch(e.Message)
		}
		if mm != nil {
			reason := strings.TrimSpace(mm[2])
			for _, k := range conflictKeywords {
				if strings.Contains(reason, k) {
					add(role, mm[1], reason)
					break
				}
			}
		}
	}
	return out
}
