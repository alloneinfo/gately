// Package portforward 实现 TCP/UDP 端口转发模块。
package portforward

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"slices"
	"sync"

	"github.com/wangwanjie/gately/internal/config"
	"github.com/wangwanjie/gately/internal/httpx"
	"github.com/wangwanjie/gately/internal/logbuf"
	"github.com/wangwanjie/gately/internal/module"
)

// ID 是模块标识。
const ID = "portforward"

// Module 是端口转发模块。
type Module struct {
	mu      sync.Mutex
	store   *config.Store
	logs    *logbuf.Buffer
	cfg     fileConfig
	ctx     context.Context
	runners map[string]*runner
	// errors 记录规则级启动错误（规则 key -> 错误信息）
	errors map[string]string
}

// New 创建模块实例。
func New() *Module {
	return &Module{runners: map[string]*runner{}, errors: map[string]string{}}
}

// ID 实现 module.Module。
func (m *Module) ID() string { return ID }

// Title 实现 module.Module。
func (m *Module) Title() string { return "端口转发" }

// Init 实现 module.Module。
func (m *Module) Init(env module.Env) error {
	m.store = env.Store
	m.logs = env.Logs
	m.cfg = fileConfig{Settings: defaultSettings()}
	if _, err := m.store.Load(ID, &m.cfg); err != nil {
		return err
	}
	m.cfg.Settings.normalize()
	return nil
}

// Start 实现 module.Module。
func (m *Module) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ctx = ctx
	if !m.cfg.Settings.Enable {
		m.logs.Add(logbuf.LevelInfo, "", "端口转发模块已禁用")
		return nil
	}
	for _, rule := range m.cfg.Rules {
		if rule.Enable {
			m.startRuleLocked(rule)
		}
	}
	return nil
}

// Stop 实现 module.Module。
func (m *Module) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for key := range m.runners {
		m.stopRuleLocked(key)
	}
	return nil
}

func (m *Module) startRuleLocked(rule Rule) {
	if m.ctx == nil {
		return
	}
	r := newRunner(rule, m.cfg.Settings, m.logs.Logger(ruleLabel(rule)))
	if err := r.start(m.ctx); err != nil {
		m.errors[rule.Key] = err.Error()
		return
	}
	delete(m.errors, rule.Key)
	if len(r.listenErrors) > 0 {
		m.errors[rule.Key] = fmt.Sprintf("%d 个监听失败", len(r.listenErrors))
	}
	m.runners[rule.Key] = r
}

func (m *Module) stopRuleLocked(key string) {
	if r, ok := m.runners[key]; ok {
		r.stop()
		delete(m.runners, key)
		r.log.Infof("已停止")
	}
	delete(m.errors, key)
}

// restartLocked 按当前配置重启一条规则（规则不存在或已禁用时只停止）。
func (m *Module) restartLocked(key string) {
	m.stopRuleLocked(key)
	if !m.cfg.Settings.Enable {
		return
	}
	for _, rule := range m.cfg.Rules {
		if rule.Key == key && rule.Enable {
			m.startRuleLocked(rule)
		}
	}
}

func (m *Module) saveLocked() error {
	return m.store.Save(ID, m.cfg)
}

func ruleLabel(r Rule) string {
	if r.Name != "" {
		return r.Name
	}
	return r.ListenPorts
}

func newKey() string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func (m *Module) indexLocked(key string) int {
	return slices.IndexFunc(m.cfg.Rules, func(r Rule) bool { return r.Key == key })
}

// RuleView 是带运行状态的规则，用于 API 输出。
type RuleView struct {
	Rule
	Running bool          `json:"running"`
	Error   string        `json:"error,omitempty"`
	Details []string      `json:"details,omitempty"`
	Stats   StatsSnapshot `json:"stats"`
}

func (m *Module) views() []RuleView {
	out := make([]RuleView, 0, len(m.cfg.Rules))
	for _, rule := range m.cfg.Rules {
		v := RuleView{Rule: rule, Error: m.errors[rule.Key]}
		if r, ok := m.runners[rule.Key]; ok {
			v.Running = true
			v.Stats = r.stats.snapshot()
			v.Details = r.listenErrors
		}
		out = append(out, v)
	}
	return out
}

// Routes 实现 module.Module。
func (m *Module) Routes(r module.Router) {
	r.HandleFunc("GET /rules", m.handleList)
	r.HandleFunc("POST /rules", m.handleCreate)
	r.HandleFunc("PUT /rules/{key}", m.handleUpdate)
	r.HandleFunc("DELETE /rules/{key}", m.handleDelete)
	r.HandleFunc("PUT /rules/{key}/enable", m.handleEnable)
	r.HandleFunc("PUT /order", m.handleOrder)
	r.HandleFunc("GET /settings", m.handleGetSettings)
	r.HandleFunc("PUT /settings", m.handlePutSettings)
}

func (m *Module) handleList(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()
	httpx.OK(w, m.views())
}

func (m *Module) handleCreate(w http.ResponseWriter, r *http.Request) {
	var rule Rule
	if err := httpx.ReadJSON(r, &rule); err != nil {
		httpx.Error(w, http.StatusBadRequest, "%v", err)
		return
	}
	if err := rule.Validate(); err != nil {
		httpx.Error(w, http.StatusBadRequest, "%v", err)
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	rule.Key = newKey()
	m.cfg.Rules = append(m.cfg.Rules, rule)
	if err := m.saveLocked(); err != nil {
		m.cfg.Rules = m.cfg.Rules[:len(m.cfg.Rules)-1]
		httpx.Error(w, http.StatusInternalServerError, "保存失败: %v", err)
		return
	}
	m.restartLocked(rule.Key)
	httpx.OK(w, rule)
}

func (m *Module) handleUpdate(w http.ResponseWriter, r *http.Request) {
	var rule Rule
	if err := httpx.ReadJSON(r, &rule); err != nil {
		httpx.Error(w, http.StatusBadRequest, "%v", err)
		return
	}
	if err := rule.Validate(); err != nil {
		httpx.Error(w, http.StatusBadRequest, "%v", err)
		return
	}
	key := r.PathValue("key")
	m.mu.Lock()
	defer m.mu.Unlock()
	i := m.indexLocked(key)
	if i < 0 {
		httpx.Error(w, http.StatusNotFound, "规则不存在")
		return
	}
	old := m.cfg.Rules[i]
	rule.Key = key
	m.cfg.Rules[i] = rule
	if err := m.saveLocked(); err != nil {
		m.cfg.Rules[i] = old
		httpx.Error(w, http.StatusInternalServerError, "保存失败: %v", err)
		return
	}
	m.restartLocked(key)
	httpx.OK(w, rule)
}

func (m *Module) handleDelete(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	m.mu.Lock()
	defer m.mu.Unlock()
	i := m.indexLocked(key)
	if i < 0 {
		httpx.Error(w, http.StatusNotFound, "规则不存在")
		return
	}
	old := slices.Clone(m.cfg.Rules)
	m.cfg.Rules = slices.Delete(m.cfg.Rules, i, i+1)
	if err := m.saveLocked(); err != nil {
		m.cfg.Rules = old
		httpx.Error(w, http.StatusInternalServerError, "保存失败: %v", err)
		return
	}
	m.stopRuleLocked(key)
	httpx.OK(w, map[string]bool{"ok": true})
}

type enableRequest struct {
	Enable bool `json:"enable"`
}

func (m *Module) handleEnable(w http.ResponseWriter, r *http.Request) {
	var req enableRequest
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "%v", err)
		return
	}
	key := r.PathValue("key")
	m.mu.Lock()
	defer m.mu.Unlock()
	i := m.indexLocked(key)
	if i < 0 {
		httpx.Error(w, http.StatusNotFound, "规则不存在")
		return
	}
	m.cfg.Rules[i].Enable = req.Enable
	if err := m.saveLocked(); err != nil {
		m.cfg.Rules[i].Enable = !req.Enable
		httpx.Error(w, http.StatusInternalServerError, "保存失败: %v", err)
		return
	}
	m.restartLocked(key)
	httpx.OK(w, map[string]bool{"ok": true})
}

type orderRequest struct {
	Keys []string `json:"keys"`
}

func (m *Module) handleOrder(w http.ResponseWriter, r *http.Request) {
	var req orderRequest
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "%v", err)
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(req.Keys) != len(m.cfg.Rules) {
		httpx.Error(w, http.StatusBadRequest, "排序列表与规则数量不一致")
		return
	}
	reordered := make([]Rule, 0, len(req.Keys))
	for _, k := range req.Keys {
		i := m.indexLocked(k)
		if i < 0 {
			httpx.Error(w, http.StatusBadRequest, "未知规则 %s", k)
			return
		}
		reordered = append(reordered, m.cfg.Rules[i])
	}
	old := m.cfg.Rules
	m.cfg.Rules = reordered
	if err := m.saveLocked(); err != nil {
		m.cfg.Rules = old
		httpx.Error(w, http.StatusInternalServerError, "保存失败: %v", err)
		return
	}
	httpx.OK(w, map[string]bool{"ok": true})
}

func (m *Module) handleGetSettings(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()
	httpx.OK(w, m.cfg.Settings)
}

func (m *Module) handlePutSettings(w http.ResponseWriter, r *http.Request) {
	var s Settings
	if err := httpx.ReadJSON(r, &s); err != nil {
		httpx.Error(w, http.StatusBadRequest, "%v", err)
		return
	}
	s.normalize()
	m.mu.Lock()
	defer m.mu.Unlock()
	old := m.cfg.Settings
	m.cfg.Settings = s
	if err := m.saveLocked(); err != nil {
		m.cfg.Settings = old
		httpx.Error(w, http.StatusInternalServerError, "保存失败: %v", err)
		return
	}
	// 全局参数影响所有规则，整体重启
	for key := range m.runners {
		m.stopRuleLocked(key)
	}
	if s.Enable {
		for _, rule := range m.cfg.Rules {
			if rule.Enable {
				m.startRuleLocked(rule)
			}
		}
	}
	httpx.OK(w, s)
}

// ImportRules 追加导入规则（由 Lucky 导入器调用），返回导入数量。
func (m *Module) ImportRules(rules []Rule) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	old := m.cfg.Rules
	added := 0
	for _, rule := range rules {
		if err := rule.Validate(); err != nil {
			continue
		}
		rule.Key = newKey()
		m.cfg.Rules = append(m.cfg.Rules, rule)
		added++
	}
	if err := m.saveLocked(); err != nil {
		m.cfg.Rules = old
		return 0, err
	}
	for _, rule := range m.cfg.Rules[len(old):] {
		m.restartLocked(rule.Key)
	}
	return added, nil
}
