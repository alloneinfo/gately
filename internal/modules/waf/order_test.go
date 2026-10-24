package waf

import (
	"strings"
	"testing"
)

// sqliTarget 是会被 CRS 判为 SQL 注入的请求
const sqliTarget = "/api?id=1%27%20OR%20%271%27%3D%271%27%20UNION%20SELECT%20password%20FROM%20users--"

// orderedInstance 按 customFirst 决定自定义规则组排在 CRS 规则组之前还是之后。
func orderedInstance(key, rule string, customFirst bool) Instance {
	crs := RuleSet{Enable: true, Type: RuleCRS}
	cus := RuleSet{Enable: true, Type: RuleCustom, Param: rule}
	rs := []RuleSet{crs, cus}
	if customFirst {
		rs = []RuleSet{cus, crs}
	}
	return Instance{Key: key, Name: key, Enable: true, Paranoia: 1, InboundThreshold: 5, OutboundThreshold: 4, RuleSets: rs}
}

// 生成的指令须保持规则组在列表中的顺序，界面上的上移/下移才有意义。
func TestDirectivesFollowRuleSetOrder(t *testing.T) {
	rule := `SecRule REQUEST_URI "@beginsWith /api" "id:10002,phase:2,allow"`
	for _, first := range []bool{true, false} {
		in := orderedInstance("o", rule, first)
		d, err := in.directives()
		if err != nil {
			t.Fatal(err)
		}
		custom := strings.Index(d, "id:10002")
		crs := strings.Index(d, "Include @owasp_crs/REQUEST-901-INITIALIZATION.conf")
		if custom < 0 || crs < 0 {
			t.Fatalf("指令缺少自定义规则或 CRS: custom=%d crs=%d", custom, crs)
		}
		if (custom < crs) != first {
			t.Fatalf("customFirst=%v 时自定义规则位置错误: custom=%d crs=%d", first, custom, crs)
		}
	}
}

// 阶段 2 的放行规则只有排在 CRS 之前才生效，排在之后时 CRS 的 949110 已先拦截。
func TestAllowRuleMustPrecedeCRS(t *testing.T) {
	rule := `SecRule REQUEST_URI "@beginsWith /api" "id:10002,phase:2,allow"`
	m := newModule(t, orderedInstance("before", rule, true), orderedInstance("after", rule, false))
	if c := status(t, m.Middleware("before", ok), sqliTarget); c != 200 {
		t.Fatalf("放行规则在 CRS 之前应放行: %d", c)
	}
	if c := status(t, m.Middleware("after", ok), sqliTarget); c != 403 {
		t.Fatalf("放行规则在 CRS 之后应被 CRS 拦截: %d", c)
	}
}

// 拦截规则放在 CRS 前后都应生效，与界面提示一致。
func TestDenyRuleWorksInAnyPosition(t *testing.T) {
	rule := `SecRule REQUEST_URI "@beginsWith /admin" "id:10001,phase:1,deny,status:403"`
	m := newModule(t, orderedInstance("before", rule, true), orderedInstance("after", rule, false))
	for _, key := range []string{"before", "after"} {
		h := m.Middleware(key, ok)
		if c := status(t, h, "/admin/x"); c != 403 {
			t.Fatalf("%s: 拦截规则应生效: %d", key, c)
		}
		if c := status(t, h, "/index.html"); c != 200 {
			t.Fatalf("%s: 正常请求不应被拦截: %d", key, c)
		}
	}
}

// SecRuleRemoveById 等配置期指令只作用于已加载的规则：排在 CRS 之后才能移除 CRS 规则，排在之前不起作用。
// 与界面提示一致：运行期的 allow、ctl: 放在 CRS 之前，配置期的移除/改目标指令放在 CRS 之后。
func TestRemoveByIDMustFollowCRS(t *testing.T) {
	rule := `SecRuleRemoveById 949110`
	m := newModule(t, orderedInstance("before", rule, true), orderedInstance("after", rule, false))
	if c := status(t, m.Middleware("before", ok), sqliTarget); c != 403 {
		t.Fatalf("移除指令在 CRS 之前不应生效: %d", c)
	}
	if c := status(t, m.Middleware("after", ok), sqliTarget); c != 200 {
		t.Fatalf("移除指令在 CRS 之后应生效: %d", c)
	}
}
