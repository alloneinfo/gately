package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/wangwanjie/gately/internal/crud"
	"github.com/wangwanjie/gately/internal/httpx"
	"github.com/wangwanjie/gately/internal/logbuf"
)

// 本文件实现系统信息、磁盘占用、清理与模块设置接口。

// secretMask 是设置接口返回的密码占位符，提交时收到该值表示保留原密码；与其他模块共用 crud.SecretMask，
// 便于界面识别并通过 GET /settings/secrets 取回明文。
const secretMask = crud.SecretMask

// legacySecretMask 是旧版本使用的占位符，升级前打开的页面提交它时同样保留原密码。
const legacySecretMask = "******"

func (m *Module) registerSystemRoutes(add func(pattern string, h http.HandlerFunc)) {
	add("GET /info", m.handleInfo)
	add("GET /disk-usage", m.handleDiskUsage)
	add("POST /prune", m.handlePrune)
	add("GET /settings", m.handleGetSettings)
	add("PUT /settings", m.handlePutSettings)
	// 查看仓库账号密码明文，键为 registry:<仓库地址>（与设置中保存的地址一致）
	add("GET /settings/secrets", crud.SecretsHandler(&m.mu, func() map[string]string {
		out := map[string]string{}
		for _, r := range m.settings.Registries {
			out["registry:"+r.Server] = r.Password
		}
		return out
	}))
}

// ---------- 系统信息 ----------

// InfoView 是 /info 的响应。
type InfoView struct {
	Version       string `json:"version"`
	APIVersion    string `json:"apiVersion"`
	OS            string `json:"os"`
	Arch          string `json:"arch"`
	Kernel        string `json:"kernel"`
	Name          string `json:"name"`
	CPUs          int    `json:"cpus"`
	MemTotal      int64  `json:"memTotal"`
	StorageDriver string `json:"storageDriver"`
	RootDir       string `json:"rootDir"`
	Containers    struct {
		Total   int `json:"total"`
		Running int `json:"running"`
		Paused  int `json:"paused"`
		Stopped int `json:"stopped"`
	} `json:"containers"`
	Images            int    `json:"images"`
	ComposeCli        bool   `json:"composeCli"`
	ComposeCliVersion string `json:"composeCliVersion"`
	// DaemonMirrors 是 Docker 守护进程自身的镜像加速地址（daemon.json 的 registry-mirrors），
	// 由 dockerd 拉取 Docker Hub 镜像时自动使用，与 Gately 设置中的回退加速地址相互独立
	DaemonMirrors []string `json:"daemonMirrors"`
}

func (m *Module) handleInfo(w http.ResponseWriter, r *http.Request) {
	c, ok := m.clientOrFail(w)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	var info struct {
		ServerVersion     string `json:"ServerVersion"`
		OSType            string `json:"OSType"`
		Architecture      string `json:"Architecture"`
		KernelVersion     string `json:"KernelVersion"`
		Name              string `json:"Name"`
		NCPU              int    `json:"NCPU"`
		MemTotal          int64  `json:"MemTotal"`
		Driver            string `json:"Driver"`
		DockerRootDir     string `json:"DockerRootDir"`
		Containers        int    `json:"Containers"`
		ContainersRunning int    `json:"ContainersRunning"`
		ContainersPaused  int    `json:"ContainersPaused"`
		ContainersStopped int    `json:"ContainersStopped"`
		Images            int    `json:"Images"`
		RegistryConfig    struct {
			Mirrors []string `json:"Mirrors"`
		} `json:"RegistryConfig"`
	}
	if err := c.do(ctx, http.MethodGet, "/info", nil, &info); err != nil {
		httpx.Error(w, http.StatusBadGateway, "无法连接 Docker：%v", err)
		return
	}
	// API 版本在 /version 里；失败时留空，不影响其它信息
	var ver struct {
		APIVersion string `json:"ApiVersion"`
	}
	_ = c.do(ctx, http.MethodGet, "/version", nil, &ver)

	v := InfoView{Version: info.ServerVersion, APIVersion: ver.APIVersion, OS: info.OSType, Arch: info.Architecture, Kernel: info.KernelVersion,
		Name: info.Name, CPUs: info.NCPU, MemTotal: info.MemTotal, StorageDriver: info.Driver, RootDir: info.DockerRootDir, Images: info.Images}
	v.Containers.Total, v.Containers.Running, v.Containers.Paused, v.Containers.Stopped =
		info.Containers, info.ContainersRunning, info.ContainersPaused, info.ContainersStopped
	v.DaemonMirrors = nonNil(info.RegistryConfig.Mirrors)
	v.ComposeCli, v.ComposeCliVersion = m.composeCLI()
	httpx.OK(w, v)
}

// composeInfo 是 compose 命令行探测结果的缓存。
type composeInfo struct {
	ok      bool
	version string
	at      time.Time
}

// composeCacheTTL 是探测结果缓存时长。
const composeCacheTTL = time.Minute

// composeProbe 探测 compose 命令行，测试中替换。
var composeProbe = probeComposeCLI

// composeCLI 返回 compose 命令行是否可用及其版本，结果缓存 1 分钟。
func (m *Module) composeCLI() (bool, string) {
	m.composeMu.Lock()
	defer m.composeMu.Unlock()
	if !m.composeCache.at.IsZero() && time.Since(m.composeCache.at) < composeCacheTTL {
		return m.composeCache.ok, m.composeCache.version
	}
	ok, ver := composeProbe()
	m.composeCache = composeInfo{ok: ok, version: ver, at: time.Now()}
	return ok, ver
}

// probeComposeCLI 复用 compose 模块的命令行查找（依次尝试 docker compose 与 docker-compose，运行 version --short），整体最多 6 秒。
func probeComposeCLI() (bool, string) {
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()
	cli, err := findComposeCLI(ctx)
	if err != nil {
		return false, ""
	}
	v := strings.TrimPrefix(cli.version, "v")
	if i := strings.IndexByte(v, '\n'); i >= 0 {
		v = strings.TrimSpace(v[:i])
	}
	return true, v
}

// ---------- 磁盘占用 ----------

// UsageItem 是一类资源的占用。
type UsageItem struct {
	Count       int   `json:"count"`
	Size        int64 `json:"size"`
	Reclaimable int64 `json:"reclaimable"`
}

// DiskUsage 是 /disk-usage 的响应。
type DiskUsage struct {
	Images     UsageItem `json:"images"`
	Containers UsageItem `json:"containers"`
	Volumes    UsageItem `json:"volumes"`
	BuildCache UsageItem `json:"buildCache"`
}

// dfResponse 是 /system/df 的响应子集。
type dfResponse struct {
	LayersSize int64 `json:"LayersSize"`
	Images     []struct {
		Size       int64 `json:"Size"`
		Containers int64 `json:"Containers"`
	} `json:"Images"`
	Containers []struct {
		SizeRw int64  `json:"SizeRw"`
		State  string `json:"State"`
	} `json:"Containers"`
	Volumes []struct {
		UsageData *struct {
			Size     int64 `json:"Size"`
			RefCount int64 `json:"RefCount"`
		} `json:"UsageData"`
	} `json:"Volumes"`
	BuildCache []struct {
		Size  int64 `json:"Size"`
		InUse bool  `json:"InUse"`
	} `json:"BuildCache"`
}

// pos 把 Docker 用 -1 表示的“未知大小”当作 0 累加。
func pos(n int64) int64 {
	if n < 0 {
		return 0
	}
	return n
}

// diskUsageFrom 汇总 /system/df：大小为 -1 的条目不计入；可回收量按“未被使用”的条目统计。
func diskUsageFrom(df dfResponse) DiskUsage {
	var u DiskUsage
	u.Images.Count = len(df.Images)
	var imgSum int64
	for _, im := range df.Images {
		imgSum += pos(im.Size)
		if im.Containers == 0 {
			u.Images.Reclaimable += pos(im.Size)
		}
	}
	// LayersSize 去掉了镜像间共享层的重复统计，比逐个相加更准确
	u.Images.Size = imgSum
	if df.LayersSize > 0 {
		u.Images.Size = df.LayersSize
	}
	if u.Images.Reclaimable > u.Images.Size {
		u.Images.Reclaimable = u.Images.Size
	}
	u.Containers.Count = len(df.Containers)
	for _, ct := range df.Containers {
		u.Containers.Size += pos(ct.SizeRw)
		if ct.State != "running" && ct.State != "paused" && ct.State != "restarting" {
			u.Containers.Reclaimable += pos(ct.SizeRw)
		}
	}
	u.Volumes.Count = len(df.Volumes)
	for _, v := range df.Volumes {
		if v.UsageData == nil {
			continue
		}
		u.Volumes.Size += pos(v.UsageData.Size)
		if v.UsageData.RefCount == 0 {
			u.Volumes.Reclaimable += pos(v.UsageData.Size)
		}
	}
	u.BuildCache.Count = len(df.BuildCache)
	for _, b := range df.BuildCache {
		u.BuildCache.Size += pos(b.Size)
		if !b.InUse {
			u.BuildCache.Reclaimable += pos(b.Size)
		}
	}
	return u
}

func (m *Module) handleDiskUsage(w http.ResponseWriter, r *http.Request) {
	c, ok := m.clientOrFail(w)
	if !ok {
		return
	}
	// 统计磁盘占用在大量镜像与卷时较慢
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()
	var df dfResponse
	if err := c.do(ctx, http.MethodGet, "/system/df", nil, &df); err != nil {
		writeCtrErr(w, err)
		return
	}
	httpx.OK(w, diskUsageFrom(df))
}

// ---------- 清理 ----------

// pruneEndpoint 返回清理目标对应的 Engine 路径。
func pruneEndpoint(target string, all bool) (string, bool) {
	switch target {
	case "containers":
		return "/containers/prune", true
	case "images":
		// dangling=false 表示清理全部未被使用的镜像，否则只清理悬空镜像
		filter, _ := json.Marshal(map[string][]string{"dangling": {fmt.Sprint(!all)}})
		return "/images/prune?filters=" + url.QueryEscape(string(filter)), true
	case "networks":
		return "/networks/prune", true
	case "volumes":
		if all {
			// 新版 Docker（API 1.42+）默认只清理匿名卷，all=true 才包含命名卷；旧版忽略该过滤器
			filter, _ := json.Marshal(map[string][]string{"all": {"true"}})
			return "/volumes/prune?filters=" + url.QueryEscape(string(filter)), true
		}
		return "/volumes/prune", true
	case "buildcache":
		if all {
			return "/build/prune?all=1", true
		}
		return "/build/prune", true
	}
	return "", false
}

// parsePruneResult 统计 *Deleted 数组的条目数与 SpaceReclaimed。
func parsePruneResult(raw map[string]json.RawMessage) (deleted int, space int64) {
	for k, v := range raw {
		if strings.HasSuffix(k, "Deleted") {
			var arr []json.RawMessage
			if json.Unmarshal(v, &arr) == nil {
				deleted += len(arr)
			}
		}
	}
	_ = json.Unmarshal(raw["SpaceReclaimed"], &space)
	return
}

func (m *Module) handlePrune(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Target string `json:"target"`
		All    bool   `json:"all"`
	}
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "%v", err)
		return
	}
	endpoint, ok := pruneEndpoint(req.Target, req.All)
	if !ok {
		httpx.Error(w, http.StatusBadRequest, "清理目标 %q 无效，可选 containers / images / networks / volumes / buildcache", req.Target)
		return
	}
	c, ok := m.clientOrFail(w)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Minute)
	defer cancel()
	var raw map[string]json.RawMessage
	if err := c.do(ctx, http.MethodPost, endpoint, nil, &raw); err != nil {
		writeCtrErr(w, err)
		return
	}
	deleted, space := parsePruneResult(raw)
	m.logs.Add(logbuf.LevelInfo, "", fmt.Sprintf("清理 %s：删除 %d 项，释放 %d 字节", req.Target, deleted, space))
	httpx.OK(w, map[string]int64{"deleted": int64(deleted), "spaceReclaimed": space})
}

// ---------- 设置 ----------

// maskSettings 返回密码被掩码的设置副本。
func maskSettings(s Settings) Settings {
	out := s
	out.Mirrors = nonNil(s.Mirrors)
	out.Registries = make([]Registry, 0, len(s.Registries))
	for _, r := range s.Registries {
		if r.Password != "" {
			r.Password = secretMask
		}
		out.Registries = append(out.Registries, r)
	}
	return out
}

// mergeSettings 规整提交的设置：镜像加速地址去空去重；仓库按 server 去重（同一 server 后出现的覆盖先出现的，位置不变）；
// 密码为掩码时沿用 old 中同一 server 的原密码。
func mergeSettings(old, in Settings) Settings {
	out := in
	out.Host = strings.TrimSpace(in.Host)
	out.Mirrors = []string{}
	seen := map[string]bool{}
	for _, mir := range in.Mirrors {
		mir = strings.TrimSpace(mir)
		if mir == "" || seen[mir] {
			continue
		}
		seen[mir] = true
		out.Mirrors = append(out.Mirrors, mir)
	}
	oldPass := map[string]string{}
	for _, r := range old.Registries {
		oldPass[normalizeRegistry(r.Server)] = r.Password
	}
	out.Registries = []Registry{}
	index := map[string]int{}
	for _, r := range in.Registries {
		r.Server = strings.TrimSpace(r.Server)
		r.Username = strings.TrimSpace(r.Username)
		if r.Server == "" {
			continue
		}
		key := normalizeRegistry(r.Server)
		if r.Password == secretMask || r.Password == legacySecretMask {
			r.Password = oldPass[key]
		}
		if i, ok := index[key]; ok {
			out.Registries[i] = r
			continue
		}
		index[key] = len(out.Registries)
		out.Registries = append(out.Registries, r)
	}
	if out.AutoCheckHours < 0 {
		out.AutoCheckHours = 0
	}
	return out
}

func (m *Module) handleGetSettings(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	s := m.settings
	m.mu.Unlock()
	httpx.OK(w, maskSettings(s))
}

func (m *Module) handlePutSettings(w http.ResponseWriter, r *http.Request) {
	var in Settings
	if err := httpx.ReadJSON(r, &in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "%v", err)
		return
	}
	if _, err := NewClient(strings.TrimSpace(in.Host)); err != nil {
		httpx.Error(w, http.StatusBadRequest, "%v", err)
		return
	}
	m.mu.Lock()
	s := mergeSettings(m.settings, in)
	if err := m.store.Save(ID, s); err != nil {
		m.mu.Unlock()
		httpx.Error(w, http.StatusInternalServerError, "%v", err)
		return
	}
	m.settings = s
	m.mu.Unlock()
	// 自动检查间隔可能变化
	m.restartAutoCheck()
	m.logs.Add(logbuf.LevelInfo, "", fmt.Sprintf("设置已修改：镜像加速 %d 个，仓库账号 %d 个，自动检查间隔 %d 小时", len(s.Mirrors), len(s.Registries), s.AutoCheckHours))
	httpx.OK(w, maskSettings(s))
}
