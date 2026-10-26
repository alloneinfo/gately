// Package frp 托管 frpc（内网穿透客户端）与 frps（服务端）：
// 在界面中维护连接与代理列表，生成配置文件并守护外部 frp 进程。
package frp

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/wangwanjie/gately/internal/binstall"
	"github.com/wangwanjie/gately/internal/config"
	"github.com/wangwanjie/gately/internal/crud"
	"github.com/wangwanjie/gately/internal/fsperm"
	"github.com/wangwanjie/gately/internal/httpx"
	"github.com/wangwanjie/gately/internal/logbuf"
	"github.com/wangwanjie/gately/internal/module"
	"github.com/wangwanjie/gately/internal/proc"
)

// ID 是模块标识。
const ID = "frp"

// role 描述 frp 的一个角色（客户端或服务端）及其托管进程。
type role struct {
	name string // frpc 或 frps
	sup  *proc.Supervisor
	err  string
	// version 缓存已探测的二进制版本，避免界面轮询时反复执行
	version     string
	versionPath string
}

// Module 是 frp 模块。
type Module struct {
	mu      sync.Mutex
	store   *config.Store
	logs    *logbuf.Buffer
	cfg     fileConfig
	client  *role
	server  *role
	started bool
	// httpClient 返回下载程序使用的 HTTP 客户端，由框架按“更新代理”设置注入
	httpClient func() *http.Client
	install    *binstall.Installer
}

// New 创建模块。
func New() *Module {
	return &Module{httpClient: func() *http.Client { return http.DefaultClient }}
}

// SetHTTPClient 实现 module.HTTPClientAware。
func (m *Module) SetHTTPClient(fn func() *http.Client) {
	if fn != nil {
		m.httpClient = fn
	}
}

// ID 实现 module.Module。
func (m *Module) ID() string { return ID }

// Title 实现 module.Module。
func (m *Module) Title() string { return "frp 内网穿透" }

// Init 实现 module.Module。
func (m *Module) Init(env module.Env) error {
	m.store, m.logs = env.Store, env.Logs
	m.client = &role{name: "frpc", sup: proc.New(m.logs.Logger("frpc"))}
	m.server = &role{name: "frps", sup: proc.New(m.logs.Logger("frps"))}
	m.install = m.newInstaller()
	if _, err := m.store.Load(ID, &m.cfg); err != nil {
		return err
	}
	m.cfg.fillDefaults()
	return nil
}

// fillDefaults 补齐未设置的默认值。
func (c *fileConfig) fillDefaults() {
	if c.Client.ServerPort == 0 {
		c.Client.ServerPort = 7000
	}
	if c.Server.BindPort == 0 {
		c.Server.BindPort = 7000
	}
	if c.Server.DashboardAddr == "" {
		c.Server.DashboardAddr = "127.0.0.1"
	}
}

// Start 实现 module.Module。
func (m *Module) Start(context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.started = true
	m.applyClientLocked()
	m.applyServerLocked()
	return nil
}

// Stop 实现 module.Module。
func (m *Module) Stop() error {
	m.mu.Lock()
	m.started = false
	m.mu.Unlock()
	m.client.sup.Stop()
	m.server.sup.Stop()
	return nil
}

func (m *Module) saveLocked() error { return m.store.Save(ID, m.cfg) }

// writeConfig 把配置写入 <配置目录>/frp/<file>，文件含令牌，权限 0600。
func (m *Module) writeConfig(file, content string) (string, error) {
	dir := filepath.Join(m.store.Dir(), "frp")
	if err := fsperm.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	path := filepath.Join(dir, file)
	tmp := path + ".tmp"
	if err := fsperm.WriteFile(tmp, []byte(content), 0o600); err != nil {
		return "", err
	}
	return path, os.Rename(tmp, path)
}

// launch 解析二进制、写配置并启动进程；调用方持有 mu。
func (m *Module) launch(r *role, binPath, file, content string) {
	r.sup.Stop()
	r.err = ""
	bin, err := proc.Resolve(binPath, m.store.Dir(), r.name)
	if err != nil {
		r.err = err.Error()
		m.logs.Logger(r.name).Errorf("%v", err)
		return
	}
	cfgPath, err := m.writeConfig(file, content)
	if err != nil {
		r.err = "写入配置文件失败: " + err.Error()
		m.logs.Logger(r.name).Errorf("%s", r.err)
		return
	}
	r.sup.Start(proc.Spec{Path: bin, Args: []string{"-c", cfgPath}})
}

func (m *Module) applyClientLocked() {
	if !m.started {
		return
	}
	if !m.cfg.Client.Enable {
		m.client.sup.Stop()
		m.client.err = ""
		return
	}
	// frpc 不允许代理重名，重名的后续条目会被忽略并提示
	seen := map[string]bool{}
	for _, p := range m.cfg.Proxies {
		if !p.Enable {
			continue
		}
		if seen[p.Name] {
			m.logs.Logger("frpc").Warnf("代理名称 %q 重复，已忽略后面的同名代理", p.Name)
		}
		seen[p.Name] = true
	}
	m.launch(m.client, m.cfg.Client.BinPath, "frpc.toml", ClientTOML(m.cfg.Client, m.cfg.Proxies))
}

func (m *Module) applyServerLocked() {
	if !m.started {
		return
	}
	if !m.cfg.Server.Enable {
		m.server.sup.Stop()
		m.server.err = ""
		return
	}
	m.launch(m.server, m.cfg.Server.BinPath, "frps.toml", ServerTOML(m.cfg.Server))
}

// roleView 是界面展示的角色状态。
type roleView struct {
	Settings any        `json:"settings"`
	State    proc.State `json:"state"`
	Binary   string     `json:"binary"`
	Version  string     `json:"version"`
	Error    string     `json:"error"`
}

func (m *Module) viewLocked(r *role, binPath string, settings any) roleView {
	v := roleView{Settings: settings, State: r.sup.State(), Error: r.err}
	bin, err := proc.Resolve(binPath, m.store.Dir(), r.name)
	if err != nil {
		if v.Error == "" {
			v.Error = err.Error()
		}
		return v
	}
	v.Binary = bin
	if r.versionPath != bin {
		r.versionPath, r.version = bin, proc.Version(bin, "--version")
	}
	v.Version = r.version
	if v.State.Error != "" {
		v.Error = v.State.Error
	}
	return v
}

// Routes 实现 module.Module。
func (m *Module) Routes(r module.Router) {
	proxies := &crud.Collection[Proxy]{
		Mu:       &m.mu,
		Items:    &m.cfg.Proxies,
		Save:     m.saveLocked,
		Key:      func(p *Proxy) *string { return &p.Key },
		Enabled:  func(p *Proxy) *bool { return &p.Enable },
		Validate: func(p *Proxy) error { return p.Validate() },
		Merge:    func(old, updated *Proxy) { crud.KeepSecret(old.SecretKey, &updated.SecretKey) },
		View:     func(p *Proxy) any { v := *p; v.SecretKey = crud.Mask(v.SecretKey); return v },
		Changed:  func(string) { m.applyClientLocked() },
		Removed:  func(string) { m.applyClientLocked() },
	}
	proxies.Register(r, "/proxies")
	m.registerImport(r)
	m.install.Register(r)
	r.HandleFunc("POST /reset", func(w http.ResponseWriter, req *http.Request) {
		m.mu.Lock()
		defer m.mu.Unlock()
		if err := m.resetLocked(); err != nil {
			httpx.Error(w, http.StatusInternalServerError, "%v", err)
			return
		}
		httpx.OK(w, map[string]bool{"ok": true})
	})

	r.HandleFunc("GET /client", func(w http.ResponseWriter, req *http.Request) {
		m.mu.Lock()
		defer m.mu.Unlock()
		s := m.cfg.Client
		s.Token = crud.Mask(s.Token)
		httpx.OK(w, m.viewLocked(m.client, m.cfg.Client.BinPath, s))
	})
	r.HandleFunc("PUT /client", func(w http.ResponseWriter, req *http.Request) {
		var s ClientSettings
		if err := httpx.ReadJSON(req, &s); err != nil {
			httpx.Error(w, http.StatusBadRequest, "%v", err)
			return
		}
		m.mu.Lock()
		defer m.mu.Unlock()
		crud.KeepSecret(m.cfg.Client.Token, &s.Token)
		if err := s.Validate(); err != nil {
			httpx.Error(w, http.StatusBadRequest, "%v", err)
			return
		}
		old := m.cfg.Client
		m.cfg.Client = s
		if err := m.saveLocked(); err != nil {
			m.cfg.Client = old
			httpx.Error(w, http.StatusInternalServerError, "保存失败: %v", err)
			return
		}
		m.applyClientLocked()
		httpx.OK(w, map[string]bool{"ok": true})
	})
	r.HandleFunc("GET /server", func(w http.ResponseWriter, req *http.Request) {
		m.mu.Lock()
		defer m.mu.Unlock()
		s := m.cfg.Server
		s.Token, s.DashboardPwd = crud.Mask(s.Token), crud.Mask(s.DashboardPwd)
		httpx.OK(w, m.viewLocked(m.server, m.cfg.Server.BinPath, s))
	})
	r.HandleFunc("PUT /server", func(w http.ResponseWriter, req *http.Request) {
		var s ServerSettings
		if err := httpx.ReadJSON(req, &s); err != nil {
			httpx.Error(w, http.StatusBadRequest, "%v", err)
			return
		}
		m.mu.Lock()
		defer m.mu.Unlock()
		crud.KeepSecret(m.cfg.Server.Token, &s.Token)
		crud.KeepSecret(m.cfg.Server.DashboardPwd, &s.DashboardPwd)
		if err := s.Validate(); err != nil {
			httpx.Error(w, http.StatusBadRequest, "%v", err)
			return
		}
		old := m.cfg.Server
		m.cfg.Server = s
		if err := m.saveLocked(); err != nil {
			m.cfg.Server = old
			httpx.Error(w, http.StatusInternalServerError, "保存失败: %v", err)
			return
		}
		m.applyServerLocked()
		httpx.OK(w, map[string]bool{"ok": true})
	})
}

// Summary 实现 module.Summarizer。
func (m *Module) Summary() []module.SummaryItem {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []module.SummaryItem
	if m.cfg.Client.Enable {
		out = append(out, roleSummary("客户端", m.client))
		enabled := 0
		for _, p := range m.cfg.Proxies {
			if p.Enable {
				enabled++
			}
		}
		out = append(out, module.SummaryItem{Label: "代理", Value: fmt.Sprintf("%d / %d", enabled, len(m.cfg.Proxies))})
	}
	if m.cfg.Server.Enable {
		out = append(out, roleSummary("服务端", m.server))
	}
	if len(out) == 0 {
		out = append(out, module.SummaryItem{Label: "状态", Value: "未启用", Level: "info"})
	}
	return out
}

func roleSummary(label string, r *role) module.SummaryItem {
	st := r.sup.State()
	switch {
	case st.Running:
		return module.SummaryItem{Label: label, Value: "运行中", Level: "ok"}
	case r.err != "" || st.Error != "":
		return module.SummaryItem{Label: label, Value: "启动失败", Level: "error"}
	default:
		return module.SummaryItem{Label: label, Value: "未运行", Level: "warn"}
	}
}

// resetLocked 恢复默认配置：停止 frpc 与 frps，清空连接、代理与安装设置，删除生成的配置文件。
// 旧配置留在配置历史中，可以恢复；安装的程序不受影响。调用方持有 mu。
func (m *Module) resetLocked() error {
	var cfg fileConfig
	cfg.fillDefaults()
	old := m.cfg
	m.cfg = cfg
	if err := m.saveLocked(); err != nil {
		m.cfg = old
		return fmt.Errorf("保存失败: %v", err)
	}
	// 新配置均未启用，apply 会停止进程
	m.applyClientLocked()
	m.applyServerLocked()
	m.client.err, m.server.err = "", ""
	if err := os.RemoveAll(filepath.Join(m.store.Dir(), "frp")); err != nil {
		return fmt.Errorf("删除生成的配置文件失败: %v", err)
	}
	m.logs.Logger("frp").Infof("已恢复默认配置")
	return nil
}
