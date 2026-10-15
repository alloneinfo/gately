package docker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/wangwanjie/gately/internal/config"
	"github.com/wangwanjie/gately/internal/httpx"
	"github.com/wangwanjie/gately/internal/logbuf"
	"github.com/wangwanjie/gately/internal/module"
)

// ID 是模块标识。
const ID = "docker"

// Settings 是模块配置。
type Settings struct {
	// Host 为 Docker 地址，默认 unix:///var/run/docker.sock
	Host string `json:"host"`
}

// UpdateInfo 是镜像更新检查结果。
type UpdateInfo struct {
	Checked   time.Time `json:"checked"`
	HasUpdate bool      `json:"hasUpdate"`
	Error     string    `json:"error,omitempty"`
}

// Module 是 Docker 管理模块。
type Module struct {
	mu       sync.Mutex
	store    *config.Store
	logs     *logbuf.Buffer
	settings Settings
	updates  map[string]UpdateInfo // 容器 ID -> 检查结果
	busy     map[string]bool
}

// New 创建模块。
func New() *Module {
	return &Module{updates: map[string]UpdateInfo{}, busy: map[string]bool{}}
}

// ID 实现 module.Module。
func (m *Module) ID() string { return ID }

// Title 实现 module.Module。
func (m *Module) Title() string { return "Docker 管理" }

// Init 实现 module.Module。
func (m *Module) Init(env module.Env) error {
	m.store, m.logs = env.Store, env.Logs
	_, err := m.store.Load(ID, &m.settings)
	return err
}

// Start 实现 module.Module。
func (m *Module) Start(context.Context) error { return nil }

// Stop 实现 module.Module。
func (m *Module) Stop() error { return nil }

func (m *Module) client() (*Client, error) {
	m.mu.Lock()
	host := m.settings.Host
	m.mu.Unlock()
	return NewClient(host)
}

// ContainerView 是界面展示的容器信息。
type ContainerView struct {
	ID      string            `json:"id"`
	Name    string            `json:"name"`
	Image   string            `json:"image"`
	State   string            `json:"state"`
	Status  string            `json:"status"`
	Created int64             `json:"created"`
	Ports   []string          `json:"ports"`
	Project string            `json:"project,omitempty"`
	Update  *UpdateInfo       `json:"update,omitempty"`
	Busy    bool              `json:"busy"`
	Labels  map[string]string `json:"-"`
}

// Routes 实现 module.Module。
func (m *Module) Routes(r module.Router) {
	r.HandleFunc("GET /containers", m.handleList)
	r.HandleFunc("POST /containers/{id}/{action}", m.handleAction)
	r.HandleFunc("GET /containers/{id}/logs", m.handleLogs)
	r.HandleFunc("POST /containers/{id}/check", m.handleCheck)
	r.HandleFunc("POST /containers/{id}/upgrade", m.handleUpgrade)
	r.HandleFunc("GET /settings", func(w http.ResponseWriter, req *http.Request) {
		m.mu.Lock()
		defer m.mu.Unlock()
		httpx.OK(w, m.settings)
	})
	r.HandleFunc("PUT /settings", func(w http.ResponseWriter, req *http.Request) {
		var s Settings
		if err := httpx.ReadJSON(req, &s); err != nil {
			httpx.Error(w, http.StatusBadRequest, "%v", err)
			return
		}
		if _, err := NewClient(s.Host); err != nil {
			httpx.Error(w, http.StatusBadRequest, "%v", err)
			return
		}
		m.mu.Lock()
		defer m.mu.Unlock()
		if err := m.store.Save(ID, s); err != nil {
			httpx.Error(w, http.StatusInternalServerError, "%v", err)
			return
		}
		m.settings = s
		httpx.OK(w, s)
	})
}

func (m *Module) handleList(w http.ResponseWriter, r *http.Request) {
	c, err := m.client()
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "%v", err)
		return
	}
	list, err := c.Containers(r.Context())
	if err != nil {
		httpx.Error(w, http.StatusBadGateway, "无法连接 Docker：%v（容器部署时需挂载 /var/run/docker.sock）", err)
		return
	}
	m.mu.Lock()
	out := make([]ContainerView, 0, len(list))
	for _, ct := range list {
		v := ContainerView{ID: ct.ID, Name: ct.Name(), Image: ct.Image, State: ct.State, Status: ct.Status, Created: ct.Created,
			Project: ct.Labels["com.docker.compose.project"], Busy: m.busy[ct.ID]}
		if u, ok := m.updates[ct.ID]; ok {
			u := u
			v.Update = &u
		}
		seen := map[string]bool{}
		for _, p := range ct.Ports {
			s := strconv.Itoa(p.PrivatePort) + "/" + p.Type
			if p.PublicPort > 0 {
				s = strconv.Itoa(p.PublicPort) + "→" + s
			}
			if !seen[s] {
				seen[s] = true
				v.Ports = append(v.Ports, s)
			}
		}
		out = append(out, v)
	}
	m.mu.Unlock()
	sort.Slice(out, func(i, j int) bool {
		if out[i].Project != out[j].Project {
			return out[i].Project < out[j].Project
		}
		return out[i].Name < out[j].Name
	})
	httpx.OK(w, out)
}

func (m *Module) handleAction(w http.ResponseWriter, r *http.Request) {
	c, err := m.client()
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "%v", err)
		return
	}
	id, action := r.PathValue("id"), r.PathValue("action")
	if err := c.Action(r.Context(), id, action); err != nil {
		httpx.Error(w, http.StatusBadGateway, "%v", err)
		return
	}
	m.logs.Add(logbuf.LevelInfo, "", fmt.Sprintf("容器 %s 执行 %s", shortID(id), action))
	httpx.OK(w, map[string]bool{"ok": true})
}

func (m *Module) handleLogs(w http.ResponseWriter, r *http.Request) {
	c, err := m.client()
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "%v", err)
		return
	}
	tail, _ := strconv.Atoi(r.URL.Query().Get("tail"))
	if tail <= 0 || tail > 5000 {
		tail = 300
	}
	text, err := c.Logs(r.Context(), r.PathValue("id"), tail)
	if err != nil {
		httpx.Error(w, http.StatusBadGateway, "%v", err)
		return
	}
	httpx.OK(w, map[string]string{"logs": text})
}

func shortID(id string) string {
	if len(id) > 12 {
		return id[:12]
	}
	return id
}

// containerImage 从 inspect 结果中取镜像引用与当前镜像 ID。
func containerImage(info map[string]json.RawMessage) (ref, imageID, name string, err error) {
	var cfg struct {
		Image string `json:"Image"`
	}
	if err = json.Unmarshal(info["Config"], &cfg); err != nil {
		return
	}
	_ = json.Unmarshal(info["Image"], &imageID)
	_ = json.Unmarshal(info["Name"], &name)
	ref = cfg.Image
	if ref == "" || len(ref) == 71 && ref[:7] == "sha256:" {
		err = errors.New("容器使用镜像 ID 创建，无法检查更新")
	}
	return
}

// checkUpdate 拉取最新镜像并与容器当前镜像比较。
func (m *Module) checkUpdate(ctx context.Context, c *Client, id string) (bool, error) {
	info, err := c.Inspect(ctx, id)
	if err != nil {
		return false, err
	}
	ref, cur, _, err := containerImage(info)
	if err != nil {
		return false, err
	}
	if err := c.Pull(ctx, ref); err != nil {
		return false, fmt.Errorf("拉取 %s 失败: %w", ref, err)
	}
	latest, err := c.ImageID(ctx, ref)
	if err != nil {
		return false, err
	}
	return latest != "" && latest != cur, nil
}

func (m *Module) handleCheck(w http.ResponseWriter, r *http.Request) {
	c, err := m.client()
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "%v", err)
		return
	}
	id := r.PathValue("id")
	ctx, cancel := context.WithTimeout(r.Context(), waitTimeout)
	defer cancel()
	has, err := m.checkUpdate(ctx, c, id)
	info := UpdateInfo{Checked: time.Now(), HasUpdate: has}
	if err != nil {
		info.Error = err.Error()
	}
	m.mu.Lock()
	m.updates[id] = info
	m.mu.Unlock()
	httpx.OK(w, info)
}

func (m *Module) handleUpgrade(w http.ResponseWriter, r *http.Request) {
	c, err := m.client()
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "%v", err)
		return
	}
	id := r.PathValue("id")
	m.mu.Lock()
	if m.busy[id] {
		m.mu.Unlock()
		httpx.Error(w, http.StatusConflict, "正在升级")
		return
	}
	m.busy[id] = true
	m.mu.Unlock()
	defer func() {
		m.mu.Lock()
		delete(m.busy, id)
		m.mu.Unlock()
	}()
	ctx, cancel := context.WithTimeout(context.Background(), waitTimeout)
	defer cancel()
	newID, err := m.upgrade(ctx, c, id)
	if err != nil {
		httpx.Error(w, http.StatusBadGateway, "%v", err)
		return
	}
	m.mu.Lock()
	delete(m.updates, id)
	m.mu.Unlock()
	httpx.OK(w, map[string]string{"id": newID})
}

// upgrade 拉取最新镜像并按原配置重建容器；新容器启动失败时恢复旧容器。
// ctx 只约束检查与拉取阶段；停止旧容器之后的每一步使用独立超时，保证恢复流程不因调用方取消而中断。
func (m *Module) upgrade(ctx context.Context, c *Client, id string) (string, error) {
	info, err := c.Inspect(ctx, id)
	if err != nil {
		return "", err
	}
	ref, _, name, err := containerImage(info)
	if err != nil {
		return "", err
	}
	name = trimSlash(name)
	log := m.logs.Logger(name)
	log.Infof("开始升级，拉取 %s", ref)
	if err := c.Pull(ctx, ref); err != nil {
		return "", fmt.Errorf("拉取镜像失败: %w", err)
	}
	body, extraNets, err := recreateBody(info)
	if err != nil {
		return "", err
	}
	var state struct {
		Running bool `json:"Running"`
	}
	_ = json.Unmarshal(info["State"], &state)

	// step 为单个容器操作提供独立超时
	step := func(fn func(context.Context) error) error {
		sctx, cancel := context.WithTimeout(context.Background(), stepTimeout)
		defer cancel()
		return fn(sctx)
	}
	backup := name + "_gately_old_" + time.Now().Format("20060102150405")
	if state.Running {
		if err := step(func(c2 context.Context) error { return c.Action(c2, id, "stop") }); err != nil {
			return "", fmt.Errorf("停止旧容器失败: %w", err)
		}
	}
	if err := step(func(c2 context.Context) error { return c.Rename(c2, id, backup) }); err != nil {
		if state.Running {
			_ = step(func(c2 context.Context) error { return c.Action(c2, id, "start") })
		}
		return "", fmt.Errorf("重命名旧容器失败: %w", err)
	}
	restore := func(cause error) error {
		log.Errorf("升级失败，恢复旧容器: %v", cause)
		if err := step(func(c2 context.Context) error { return c.Rename(c2, id, name) }); err != nil {
			log.Errorf("恢复旧容器名称失败，请手动处理 %s: %v", backup, err)
		}
		if state.Running {
			if err := step(func(c2 context.Context) error { return c.Action(c2, id, "start") }); err != nil {
				log.Errorf("重新启动旧容器失败: %v", err)
			}
		}
		return cause
	}
	var newID string
	if err := step(func(c2 context.Context) error {
		var err error
		newID, err = c.Create(c2, name, body)
		return err
	}); err != nil {
		return "", restore(fmt.Errorf("创建新容器失败: %w", err))
	}
	discard := func(cause error) (string, error) {
		_ = step(func(c2 context.Context) error { return c.Remove(c2, newID) })
		return "", restore(cause)
	}
	// 旧版 Docker 创建时只接受一个网络，其余网络在创建后逐个连接
	for _, n := range extraNets {
		if err := step(func(c2 context.Context) error { return c.ConnectNetwork(c2, n.name, newID, n.endpoint) }); err != nil {
			return discard(fmt.Errorf("连接网络 %s 失败: %w", n.name, err))
		}
	}
	if state.Running {
		if err := step(func(c2 context.Context) error { return c.Action(c2, newID, "start") }); err != nil {
			return discard(fmt.Errorf("启动新容器失败: %w", err))
		}
	}
	if err := step(func(c2 context.Context) error { return c.Remove(c2, id) }); err != nil {
		log.Warnf("删除旧容器 %s 失败，请手动清理: %v", backup, err)
	}
	log.Infof("升级完成，新容器 %s", shortID(newID))
	return newID, nil
}

func trimSlash(s string) string {
	for len(s) > 0 && s[0] == '/' {
		s = s[1:]
	}
	return s
}

// networkAttach 是创建后需要追加连接的网络。
type networkAttach struct {
	name     string
	endpoint map[string]any
}

// recreateBody 用旧容器的配置构造创建请求：Config 与 HostConfig 原样保留，网络只保留别名与固定 IP 等用户配置。
// 多网络容器只在创建请求中放主网络，其余网络通过返回值在创建后连接。
func recreateBody(info map[string]json.RawMessage) (map[string]any, []networkAttach, error) {
	var cfg map[string]any
	if err := json.Unmarshal(info["Config"], &cfg); err != nil {
		return nil, nil, err
	}
	// 主机名默认取自容器 ID，沿用会导致新容器使用旧 ID 作为主机名
	if host, _ := cfg["Hostname"].(string); host != "" {
		var id string
		_ = json.Unmarshal(info["Id"], &id)
		if len(id) >= len(host) && id[:len(host)] == host {
			delete(cfg, "Hostname")
		}
	}
	var hostCfg map[string]any
	if err := json.Unmarshal(info["HostConfig"], &hostCfg); err != nil {
		return nil, nil, err
	}
	keepAnonymousVolumes(info, hostCfg)
	var ns struct {
		Networks map[string]struct {
			Aliases    []string       `json:"Aliases"`
			IPAMConfig map[string]any `json:"IPAMConfig"`
			Links      []string       `json:"Links"`
		} `json:"Networks"`
	}
	_ = json.Unmarshal(info["NetworkSettings"], &ns)
	names := make([]string, 0, len(ns.Networks))
	for n := range ns.Networks {
		names = append(names, n)
	}
	sort.Strings(names)
	// 主网络优先取 NetworkMode 指定的网络
	if mode, _ := hostCfg["NetworkMode"].(string); mode != "" {
		for i, n := range names {
			if n == mode {
				names[0], names[i] = names[i], names[0]
				break
			}
		}
	}
	endpoints := map[string]any{}
	var extra []networkAttach
	for i, netName := range names {
		n := ns.Networks[netName]
		ep := map[string]any{}
		if len(n.Aliases) > 0 {
			ep["Aliases"] = n.Aliases
		}
		if len(n.IPAMConfig) > 0 {
			ep["IPAMConfig"] = n.IPAMConfig
		}
		if len(n.Links) > 0 {
			ep["Links"] = n.Links
		}
		if i == 0 {
			endpoints[netName] = ep
		} else {
			extra = append(extra, networkAttach{name: netName, endpoint: ep})
		}
	}
	body := cfg
	body["HostConfig"] = hostCfg
	body["NetworkingConfig"] = map[string]any{"EndpointsConfig": endpoints}
	return body, extra, nil
}

// keepAnonymousVolumes 把镜像声明、未显式挂载的匿名卷加入 Binds，新容器沿用原数据卷，避免升级后数据“消失”。
func keepAnonymousVolumes(info map[string]json.RawMessage, hostCfg map[string]any) {
	var mounts []struct {
		Type        string `json:"Type"`
		Name        string `json:"Name"`
		Destination string `json:"Destination"`
		RW          bool   `json:"RW"`
	}
	_ = json.Unmarshal(info["Mounts"], &mounts)
	covered := map[string]bool{}
	var binds []any
	if b, ok := hostCfg["Binds"].([]any); ok {
		binds = b
	}
	for _, b := range binds {
		if s, ok := b.(string); ok {
			parts := strings.Split(s, ":")
			if len(parts) >= 2 {
				covered[parts[1]] = true
			}
		}
	}
	if ms, ok := hostCfg["Mounts"].([]any); ok {
		for _, m := range ms {
			if mm, ok := m.(map[string]any); ok {
				if t, _ := mm["Target"].(string); t != "" {
					covered[t] = true
				}
			}
		}
	}
	added := false
	for _, m := range mounts {
		if m.Type != "volume" || m.Name == "" || covered[m.Destination] {
			continue
		}
		bind := m.Name + ":" + m.Destination
		if !m.RW {
			bind += ":ro"
		}
		binds = append(binds, bind)
		covered[m.Destination] = true
		added = true
	}
	if added {
		hostCfg["Binds"] = binds
	}
}
