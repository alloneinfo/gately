package waf

import (
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/wangwanjie/gately/internal/config"
	"github.com/wangwanjie/gately/internal/logbuf"
	"github.com/wangwanjie/gately/internal/luckyimport"
	"github.com/wangwanjie/gately/internal/module"
)

func newModule(t *testing.T, insts ...Instance) *Module {
	t.Helper()
	st, _ := config.Open(t.TempDir())
	m := New()
	if err := m.Init(module.Env{Store: st, Logs: logbuf.New(200, log.New(io.Discard, "", 0))}); err != nil {
		t.Fatal(err)
	}
	for _, in := range insts {
		if err := in.Validate(); err != nil {
			t.Fatal(err)
		}
		m.cfg.Instances = append(m.cfg.Instances, in)
		m.buildLocked(in.Key)
	}
	return m
}

func status(t *testing.T, h http.Handler, target string) int {
	t.Helper()
	req := httptest.NewRequest("GET", target, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Accept", "text/html")
	req.RemoteAddr = "203.0.113.7:4000"
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec.Code
}

var ok = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("ok")) })

func TestCRSBlocksAttacks(t *testing.T) {
	m := newModule(t, Instance{Key: "full", Name: "full", Enable: true, RuleSets: []RuleSet{{Enable: true, Type: RuleCRS}}})
	h := m.Middleware("full", ok)
	if c := status(t, h, "/index.html?page=2"); c != 200 {
		t.Fatalf("正常请求不应被拦截: %d", c)
	}
	attacks := []string{
		"/?file=" + url.QueryEscape("../../../../etc/passwd"),
		"/?id=" + url.QueryEscape("1' OR '1'='1' UNION SELECT password FROM users--"),
		"/?q=" + url.QueryEscape("<script>alert(document.cookie)</script>"),
	}
	for _, a := range attacks {
		if c := status(t, h, a); c != 403 {
			t.Errorf("攻击请求应被拦截 %s: %d", a, c)
		}
	}
	v := m.viewLocked(&m.cfg.Instances[0]).(InstanceView)
	if v.Blocked != int64(len(attacks)) {
		t.Fatalf("拦截计数错误: %d", v.Blocked)
	}
	logs := m.logs.Since(0)
	if len(logs) == 0 || !strings.Contains(logs[0].Message, "203.0.113.7") {
		t.Fatalf("应记录命中日志并包含客户端 IP: %v", logs)
	}
}

func TestLuckyStyleSubset(t *testing.T) {
	// 与 Lucky 中 waf-main 相同：只启用 LFI 规则
	m := newModule(t, Instance{Key: "lfi", Name: "lfi", Enable: true, InboundThreshold: 6, OutboundThreshold: 4,
		RuleSets: []RuleSet{{Enable: true, Type: RuleCRS, Param: "REQUEST-901-INITIALIZATION.conf\nREQUEST-930-APPLICATION-ATTACK-LFI.conf\nREQUEST-949-BLOCKING-EVALUATION.conf"}}})
	h := m.Middleware("lfi", ok)
	if c := status(t, h, "/?f="+url.QueryEscape("../../../etc/shadow")); c != 403 {
		t.Fatalf("LFI 应被拦截: %d", c)
	}
	// 未启用 SQL 注入规则，不拦截
	if c := status(t, h, "/?id="+url.QueryEscape("1' UNION SELECT 1--")); c != 200 {
		t.Fatalf("未启用的规则不应拦截: %d", c)
	}
}

func TestDetectModeCustomRuleAndPassThrough(t *testing.T) {
	m := newModule(t,
		Instance{Key: "det", Name: "det", Enable: true, Mode: ModeDetect, RuleSets: []RuleSet{{Enable: true, Type: RuleCRS}}},
		Instance{Key: "cus", Name: "cus", Enable: true, RuleSets: []RuleSet{{Enable: true, Type: RuleCustom,
			Param: `SecRule REQUEST_URI "@beginsWith /admin" "id:10001,phase:1,deny,status:403,msg:'禁止访问后台'"`}}},
		Instance{Key: "off", Name: "off", Enable: false, RuleSets: []RuleSet{{Enable: true, Type: RuleCustom, Param: `SecRule REQUEST_URI "@rx ." "id:10002,phase:1,deny,status:403"`}}},
	)
	if c := status(t, m.Middleware("det", ok), "/?f="+url.QueryEscape("../../etc/passwd")); c != 200 {
		t.Fatalf("仅检测模式不应拦截: %d", c)
	}
	if v := m.viewLocked(&m.cfg.Instances[0]).(InstanceView); v.Matched == 0 {
		t.Fatal("仅检测模式应记录命中")
	}
	if c := status(t, m.Middleware("cus", ok), "/admin/login"); c != 403 {
		t.Fatalf("自定义规则应拦截: %d", c)
	}
	if c := status(t, m.Middleware("off", ok), "/x"); c != 200 {
		t.Fatalf("停用实例应放行: %d", c)
	}
	if c := status(t, m.Middleware("missing", ok), "/x"); c != 200 {
		t.Fatalf("不存在的实例应放行: %d", c)
	}
}

func TestValidate(t *testing.T) {
	bad := []Instance{
		{Name: ""},
		{Name: "a", Mode: "x"},
		{Name: "a", Paranoia: 5},
		{Name: "a", RuleSets: []RuleSet{{Enable: true, Type: RuleCRS, Param: "NOPE.conf"}}},
		{Name: "a", RuleSets: []RuleSet{{Enable: true, Type: RuleCustom, Param: "SecRule BROKEN"}}},
	}
	for i, in := range bad {
		if err := in.Validate(); err == nil {
			t.Errorf("第 %d 个非法实例未报错", i)
		}
	}
	if len(CRSFiles()) < 20 {
		t.Fatal("内置 CRS 规则文件数量异常")
	}
}

// TestImportRealSample 仅在设置 GATELY_LUCKY_WAF_SAMPLE 时运行。
func TestImportRealSample(t *testing.T) {
	path := os.Getenv("GATELY_LUCKY_WAF_SAMPLE")
	if path == "" {
		t.Skip("未设置 GATELY_LUCKY_WAF_SAMPLE")
	}
	data, _ := os.ReadFile(path)
	doc, err := luckyimport.Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	m := newModule(t)
	rep, err := m.ImportLucky(doc, false)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("found=%d imported=%d %v", rep.Found, rep.Imported, rep.Warnings)
	if rep.Imported != rep.Found {
		t.Fatal("有实例未能导入")
	}
	for _, in := range m.cfg.Instances {
		h := m.Middleware(in.Key, ok)
		t.Logf("%s(%s) LFI 拦截=%v", in.Name, in.Key, status(t, h, "/?f="+url.QueryEscape("../../etc/passwd")) == 403)
	}
}
