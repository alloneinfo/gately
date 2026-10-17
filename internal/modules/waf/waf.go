// Package waf 实现基于 Coraza 与 OWASP 核心规则集（CRS）的 Web 应用防火墙，
// 以实例形式供 Web 服务的站点引用。
package waf

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"github.com/wangwanjie/gately/internal/events"
	"io/fs"
	"net"
	"net/http"
	"path"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	coreruleset "github.com/corazawaf/coraza-coreruleset/v4"
	"github.com/corazawaf/coraza/v3"
	txhttp "github.com/corazawaf/coraza/v3/http"
	"github.com/corazawaf/coraza/v3/types"

	"github.com/wangwanjie/gately/internal/config"
	"github.com/wangwanjie/gately/internal/crud"
	"github.com/wangwanjie/gately/internal/httpx"
	"github.com/wangwanjie/gately/internal/logbuf"
	"github.com/wangwanjie/gately/internal/module"
)

// ID 是模块标识。
const ID = "waf"

// 规则集类型
const (
	RuleCRS    = "crs"
	RuleCustom = "custom"
)

// 工作模式
const (
	ModeBlock  = "block"
	ModeDetect = "detect"
)

// RuleSet 是实例中的一组规则。
type RuleSet struct {
	Enable bool   `json:"enable"`
	Remark string `json:"remark"`
	Type   string `json:"type"`
	// Param：CRS 类型为规则文件名列表（每行一个，留空表示全部）；自定义类型为 SecLang 指令
	Param string `json:"param"`
}

// Instance 是一个 WAF 实例。
type Instance struct {
	Key    string `json:"key"`
	Name   string `json:"name"`
	Enable bool   `json:"enable"`
	Mode   string `json:"mode"`
	// Paranoia 为 CRS 偏执等级 1~4，越高越严格、误报越多
	Paranoia          int       `json:"paranoia"`
	InboundThreshold  int       `json:"inboundThreshold"`
	OutboundThreshold int       `json:"outboundThreshold"`
	RuleSets          []RuleSet `json:"ruleSets"`
	Remark            string    `json:"remark"`
}

type fileConfig struct {
	Instances []Instance `json:"instances"`
}

// CRSFiles 返回内置 CRS 的规则文件列表。
func CRSFiles() []string {
	entries, _ := fs.ReadDir(coreruleset.FS, "@owasp_crs")
	var out []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".conf") {
			out = append(out, e.Name())
		}
	}
	return out
}

// Validate 校验实例并补全默认值。
func (in *Instance) Validate() error {
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		return errors.New("名称不能为空")
	}
	switch in.Mode {
	case "":
		in.Mode = ModeBlock
	case ModeBlock, ModeDetect:
	default:
		return fmt.Errorf("未知模式 %q", in.Mode)
	}
	if in.Paranoia == 0 {
		in.Paranoia = 1
	}
	if in.Paranoia < 1 || in.Paranoia > 4 {
		return errors.New("偏执等级应为 1~4")
	}
	if in.InboundThreshold <= 0 {
		in.InboundThreshold = 5
	}
	if in.OutboundThreshold <= 0 {
		in.OutboundThreshold = 4
	}
	known := map[string]bool{}
	for _, f := range CRSFiles() {
		known[f] = true
	}
	for i := range in.RuleSets {
		rs := &in.RuleSets[i]
		switch rs.Type {
		case "", RuleCRS:
			rs.Type = RuleCRS
			for _, f := range splitLines(rs.Param) {
				if !known[f] {
					return fmt.Errorf("未知的 CRS 规则文件 %q", f)
				}
			}
		case RuleCustom:
		default:
			return fmt.Errorf("未知的规则类型 %q", rs.Type)
		}
	}
	_, err := in.directives()
	if err != nil {
		return err
	}
	// 试编译一次，尽早发现自定义规则的语法错误
	if _, err := compile(*in, nil); err != nil {
		return fmt.Errorf("规则编译失败: %w", err)
	}
	return nil
}

func splitLines(s string) []string {
	var out []string
	for _, l := range strings.Split(s, "\n") {
		if l = strings.TrimSpace(l); l != "" {
			out = append(out, l)
		}
	}
	return out
}

// directives 生成实例的 SecLang 指令。
func (in *Instance) directives() (string, error) {
	var b strings.Builder
	b.WriteString("Include @coraza.conf-recommended\n")
	if in.Mode == ModeDetect {
		b.WriteString("SecRuleEngine DetectionOnly\n")
	} else {
		b.WriteString("SecRuleEngine On\n")
	}
	// 响应体检查开销较大，只检查请求
	b.WriteString("SecResponseBodyAccess Off\n")
	hasCRS := false
	for _, rs := range in.RuleSets {
		if rs.Enable && rs.Type == RuleCRS {
			hasCRS = true
		}
	}
	if hasCRS {
		b.WriteString("Include @crs-setup.conf.example\n")
		fmt.Fprintf(&b, "SecAction \"id:900000,phase:1,pass,t:none,nolog,setvar:tx.blocking_paranoia_level=%d\"\n", in.Paranoia)
		fmt.Fprintf(&b, "SecAction \"id:900110,phase:1,pass,t:none,nolog,setvar:tx.inbound_anomaly_score_threshold=%d,setvar:tx.outbound_anomaly_score_threshold=%d\"\n",
			in.InboundThreshold, in.OutboundThreshold)
	}
	included := map[string]bool{}
	include := func(f string) {
		if !included[f] {
			included[f] = true
			b.WriteString("Include @owasp_crs/" + f + "\n")
		}
	}
	for _, rs := range in.RuleSets {
		if !rs.Enable {
			continue
		}
		if rs.Type == RuleCustom {
			b.WriteString(rs.Param + "\n")
			continue
		}
		files := splitLines(rs.Param)
		if len(files) == 0 {
			files = CRSFiles()
		}
		// 初始化与阻断判定规则是 CRS 工作的前提，按需自动补上并保持正确顺序
		include("REQUEST-901-INITIALIZATION.conf")
		for _, f := range files {
			if !strings.HasSuffix(f, "BLOCKING-EVALUATION.conf") && !strings.HasPrefix(f, "RESPONSE-980") {
				include(f)
			}
		}
		include("REQUEST-949-BLOCKING-EVALUATION.conf")
		for _, f := range files {
			include(f)
		}
	}
	return b.String(), nil
}

// compile 编译实例，onMatch 在规则命中时回调。
func compile(in Instance, onMatch func(types.MatchedRule)) (coraza.WAF, error) {
	d, err := in.directives()
	if err != nil {
		return nil, err
	}
	cfg := coraza.NewWAFConfig().WithRootFS(coreruleset.FS).WithDirectives(d)
	if onMatch != nil {
		cfg = cfg.WithErrorCallback(onMatch)
	}
	return coraza.NewWAF(cfg)
}

type compiled struct {
	waf     coraza.WAF
	enabled bool
	name    string
	blocked atomic.Int64
	matched atomic.Int64
}

// Module 是 WAF 模块。
type Module struct {
	mu     sync.Mutex
	store  *config.Store
	logs   *logbuf.Buffer
	cfg    fileConfig
	state  sync.Map // key -> *compiled
	errors map[string]string
}

// New 创建模块。
func New() *Module { return &Module{errors: map[string]string{}} }

// ID 实现 module.Module。
func (m *Module) ID() string { return ID }

// Title 实现 module.Module。
func (m *Module) Title() string { return "WAF 防火墙" }

// Init 实现 module.Module。
func (m *Module) Init(env module.Env) error {
	m.store, m.logs = env.Store, env.Logs
	if _, err := m.store.Load(ID, &m.cfg); err != nil {
		return err
	}
	for _, in := range m.cfg.Instances {
		m.buildLocked(in.Key)
	}
	return nil
}

// Start 实现 module.Module。
func (m *Module) Start(context.Context) error { return nil }

// Stop 实现 module.Module。
func (m *Module) Stop() error { return nil }

func (m *Module) saveLocked() error { return m.store.Save(ID, m.cfg) }

// buildLocked 重新编译实例；编译失败时保留错误并让引用该实例的站点直接放行。
func (m *Module) buildLocked(key string) {
	var inst *Instance
	for i := range m.cfg.Instances {
		if m.cfg.Instances[i].Key == key {
			inst = &m.cfg.Instances[i]
		}
	}
	if inst == nil {
		m.state.Delete(key)
		delete(m.errors, key)
		return
	}
	log := m.logs.Logger(inst.Name)
	c := &compiled{enabled: inst.Enable, name: inst.Name}
	if old, ok := m.state.Load(key); ok {
		c.blocked.Store(old.(*compiled).blocked.Load())
		c.matched.Store(old.(*compiled).matched.Load())
	}
	w, err := compile(*inst, func(r types.MatchedRule) {
		c.matched.Add(1)
		// 只记录有明确消息的规则命中，过滤 CRS 内部的计分规则
		if r.Message() != "" {
			log.Warnf("%s", r.ErrorLog())
		}
	})
	if err != nil {
		m.errors[key] = err.Error()
		log.Errorf("编译失败: %v", err)
		m.state.Delete(key)
		return
	}
	delete(m.errors, key)
	c.waf = w
	m.state.Store(key, c)
}

// Middleware 返回经过指定实例检查的处理器。实例不存在、停用或编译失败时直接放行，
// 每次请求时读取最新实例，修改 WAF 配置后无需重启 Web 服务。
func (m *Module) Middleware(key string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		v, ok := m.state.Load(key)
		if !ok {
			next.ServeHTTP(w, r)
			return
		}
		c := v.(*compiled)
		if !c.enabled || c.waf == nil {
			next.ServeHTTP(w, r)
			return
		}
		rec := &blockRecorder{ResponseWriter: w}
		txhttp.WrapHandler(c.waf, next).ServeHTTP(rec, r)
		if rec.blocked {
			c.blocked.Add(1)
			events.Publish(events.Event{
				Type:    events.WAFBlocked,
				Source:  c.name,
				Level:   events.LevelWarn,
				Title:   "WAF 拦截了请求",
				Message: r.Method + " " + r.Host + r.URL.Path,
				Fields:  map[string]string{"ip": r.RemoteAddr, "host": r.Host, "path": r.URL.Path},
			})
		}
	})
}

// blockRecorder 识别 WAF 拦截（下游未写入即返回 403）。
type blockRecorder struct {
	http.ResponseWriter
	blocked bool
	wrote   bool
}

func (b *blockRecorder) WriteHeader(code int) {
	if !b.wrote && code == http.StatusForbidden {
		b.blocked = true
	}
	b.wrote = true
	b.ResponseWriter.WriteHeader(code)
}

func (b *blockRecorder) Write(p []byte) (int, error) {
	b.wrote = true
	return b.ResponseWriter.Write(p)
}

// Unwrap 让 WebSocket 升级等能访问底层连接。
func (b *blockRecorder) Unwrap() http.ResponseWriter { return b.ResponseWriter }

// Flush 透传刷新。Coraza 用类型断言判断下游能力，必须显式实现，否则流式响应会被缓冲。
func (b *blockRecorder) Flush() {
	_ = http.NewResponseController(b.ResponseWriter).Flush()
}

// Hijack 透传连接劫持，使经过 WAF 的站点仍能升级 WebSocket。
func (b *blockRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	b.wrote = true
	return http.NewResponseController(b.ResponseWriter).Hijack()
}

// InstanceInfo 是供其他模块选择实例的简要信息。
type InstanceInfo struct {
	Key  string `json:"key"`
	Name string `json:"name"`
}

// Instances 返回实例列表。
func (m *Module) Instances() []InstanceInfo {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]InstanceInfo, 0, len(m.cfg.Instances))
	for _, in := range m.cfg.Instances {
		out = append(out, InstanceInfo{Key: in.Key, Name: in.Name})
	}
	return out
}

// InstanceView 是带统计的实例。
type InstanceView struct {
	Instance
	Error   string `json:"error,omitempty"`
	Blocked int64  `json:"blocked"`
	Matched int64  `json:"matched"`
}

func (m *Module) viewLocked(in *Instance) any {
	v := InstanceView{Instance: *in, Error: m.errors[in.Key]}
	if c, ok := m.state.Load(in.Key); ok {
		v.Blocked = c.(*compiled).blocked.Load()
		v.Matched = c.(*compiled).matched.Load()
	}
	return v
}

// Routes 实现 module.Module。
func (m *Module) Routes(r module.Router) {
	insts := &crud.Collection[Instance]{
		Mu:       &m.mu,
		Items:    &m.cfg.Instances,
		Save:     m.saveLocked,
		Key:      func(in *Instance) *string { return &in.Key },
		Enabled:  func(in *Instance) *bool { return &in.Enable },
		Validate: func(in *Instance) error { return in.Validate() },
		View:     m.viewLocked,
		Changed:  m.buildLocked,
		Removed:  m.buildLocked,
	}
	insts.Register(r, "/instances")
	r.HandleFunc("GET /crs", func(w http.ResponseWriter, req *http.Request) {
		httpx.OK(w, CRSFiles())
	})
	r.HandleFunc("GET /crs/{file}", func(w http.ResponseWriter, req *http.Request) {
		name := path.Base(req.PathValue("file"))
		data, err := fs.ReadFile(coreruleset.FS, "@owasp_crs/"+name)
		if err != nil {
			httpx.Error(w, http.StatusNotFound, "规则文件不存在")
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("Content-Length", strconv.Itoa(len(data)))
		_, _ = w.Write(data)
	})
}
