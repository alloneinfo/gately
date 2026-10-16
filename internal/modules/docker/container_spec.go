package docker

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net"
	"path"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
)

// 本文件负责 ContainerSpec（界面使用的简化配置）与 Docker Engine 容器配置之间的互转：
//   - specFromInspect：把 inspect 结果还原成 spec，用于编辑回填；
//   - applySpec：把 spec 覆盖到创建请求体上。创建时请求体为空，编辑时请求体来自旧容器，
//     spec 无法表达的字段（健康检查、日志驱动、ulimits、tmpfs 等）因此原样保留。

// SpecPort 是端口映射。
type SpecPort struct {
	HostIP    string `json:"hostIp"`
	Host      int    `json:"host"`
	Container int    `json:"container"`
	Proto     string `json:"proto"`
}

// SpecVolume 是数据卷或目录挂载。
type SpecVolume struct {
	Type     string `json:"type"` // bind / volume
	Source   string `json:"source"`
	Target   string `json:"target"`
	ReadOnly bool   `json:"readOnly"`
}

// SpecDevice 是设备映射。
type SpecDevice struct {
	Host      string `json:"host"`
	Container string `json:"container"`
	Perm      string `json:"perm"`
}

// ContainerSpec 是创建与编辑容器使用的配置。
type ContainerSpec struct {
	Name              string            `json:"name"`
	Image             string            `json:"image"`
	Pull              bool              `json:"pull"`
	AutoStart         bool              `json:"autoStart"`
	Command           string            `json:"command"`
	Entrypoint        string            `json:"entrypoint"`
	Env               []string          `json:"env"`
	Ports             []SpecPort        `json:"ports"`
	Volumes           []SpecVolume      `json:"volumes"`
	Network           string            `json:"network"`
	Networks          []string          `json:"networks"`
	IP                string            `json:"ip"`
	Aliases           []string          `json:"aliases"`
	Restart           string            `json:"restart"`
	RestartMaxRetries int               `json:"restartMaxRetries"`
	Privileged        bool              `json:"privileged"`
	CapAdd            []string          `json:"capAdd"`
	CapDrop           []string          `json:"capDrop"`
	Devices           []SpecDevice      `json:"devices"`
	User              string            `json:"user"`
	Workdir           string            `json:"workdir"`
	Hostname          string            `json:"hostname"`
	DNS               []string          `json:"dns"`
	Labels            map[string]string `json:"labels"`
	MemoryMB          int64             `json:"memoryMB"`
	CPUs              float64           `json:"cpus"`
	TTY               bool              `json:"tty"`
	StdinOpen         bool              `json:"stdinOpen"`
}

// ---------- shell 风格分词 ----------

// shellSplit 把命令行字符串按 shell 规则拆成参数：空白分隔，支持单引号、双引号与反斜杠转义。
func shellSplit(s string) ([]string, error) {
	var args []string
	var cur strings.Builder
	has := false // 当前参数是否已开始（空引号也算一个空参数）
	rs := []rune(s)
	flush := func() {
		if has {
			args = append(args, cur.String())
			cur.Reset()
			has = false
		}
	}
	for i := 0; i < len(rs); i++ {
		r := rs[i]
		switch {
		case r == ' ' || r == '\t' || r == '\n' || r == '\r':
			flush()
		case r == '\'':
			// 单引号内全部按字面处理
			has = true
			j := i + 1
			for j < len(rs) && rs[j] != '\'' {
				cur.WriteRune(rs[j])
				j++
			}
			if j >= len(rs) {
				return nil, errors.New("引号未闭合")
			}
			i = j
		case r == '"':
			has = true
			j := i + 1
			closed := false
			for j < len(rs) {
				c := rs[j]
				if c == '"' {
					closed = true
					break
				}
				if c == '\\' && j+1 < len(rs) {
					n := rs[j+1]
					// 双引号内只有这几个字符可以被转义，其余反斜杠保持字面
					if n == '"' || n == '\\' || n == '$' || n == '`' {
						cur.WriteRune(n)
						j += 2
						continue
					}
					if n == '\n' {
						j += 2
						continue
					}
				}
				cur.WriteRune(c)
				j++
			}
			if !closed {
				return nil, errors.New("引号未闭合")
			}
			i = j
		case r == '\\':
			if i+1 >= len(rs) {
				has = true
				cur.WriteRune('\\')
				break
			}
			i++
			if rs[i] == '\n' {
				// 行尾反斜杠表示续行
				break
			}
			has = true
			cur.WriteRune(rs[i])
		default:
			has = true
			cur.WriteRune(r)
		}
	}
	flush()
	return args, nil
}

var shellSafe = regexp.MustCompile(`^[A-Za-z0-9_@%+=:,./-]+$`)

// shellJoin 是 shellSplit 的逆操作：需要时用单引号包裹参数。
func shellJoin(args []string) string {
	parts := make([]string, 0, len(args))
	for _, a := range args {
		switch {
		case a == "":
			parts = append(parts, "''")
		case shellSafe.MatchString(a):
			parts = append(parts, a)
		default:
			parts = append(parts, "'"+strings.ReplaceAll(a, "'", `'\''`)+"'")
		}
	}
	return strings.Join(parts, " ")
}

// ---------- inspect -> spec ----------

// strList 兼容 Engine 返回的字符串数组、单个字符串与 null。
type strList []string

func (l *strList) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		*l = nil
		return nil
	}
	var arr []string
	if err := json.Unmarshal(b, &arr); err == nil {
		*l = arr
		return nil
	}
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	if s == "" {
		*l = nil
		return nil
	}
	*l = strList{s}
	return nil
}

// imageDefaults 是镜像自带的默认配置，用于在回填 spec 时去掉“继承自镜像”的部分。
type imageDefaults struct {
	Cmd        strList           `json:"Cmd"`
	Entrypoint strList           `json:"Entrypoint"`
	Env        []string          `json:"Env"`
	Labels     map[string]string `json:"Labels"`
	User       string            `json:"User"`
	WorkingDir string            `json:"WorkingDir"`
}

// inspectConfig 是 inspect 里 Config 的常用字段。
type inspectConfig struct {
	Image      string            `json:"Image"`
	Cmd        strList           `json:"Cmd"`
	Entrypoint strList           `json:"Entrypoint"`
	Env        []string          `json:"Env"`
	User       string            `json:"User"`
	WorkingDir string            `json:"WorkingDir"`
	Hostname   string            `json:"Hostname"`
	Labels     map[string]string `json:"Labels"`
	Tty        bool              `json:"Tty"`
	OpenStdin  bool              `json:"OpenStdin"`
}

// inspectHost 是 inspect 里 HostConfig 的常用字段。
type inspectHost struct {
	PortBindings map[string][]struct {
		HostIP   string `json:"HostIp"`
		HostPort string `json:"HostPort"`
	} `json:"PortBindings"`
	RestartPolicy struct {
		Name              string `json:"Name"`
		MaximumRetryCount int    `json:"MaximumRetryCount"`
	} `json:"RestartPolicy"`
	Privileged bool     `json:"Privileged"`
	CapAdd     []string `json:"CapAdd"`
	CapDrop    []string `json:"CapDrop"`
	Devices    []struct {
		PathOnHost        string `json:"PathOnHost"`
		PathInContainer   string `json:"PathInContainer"`
		CgroupPermissions string `json:"CgroupPermissions"`
	} `json:"Devices"`
	DNS         []string `json:"Dns"`
	Memory      int64    `json:"Memory"`
	NanoCPUs    int64    `json:"NanoCpus"`
	NetworkMode string   `json:"NetworkMode"`
}

// inspectMount 是 inspect 里的一条挂载。
type inspectMount struct {
	Type        string `json:"Type"`
	Name        string `json:"Name"`
	Source      string `json:"Source"`
	Destination string `json:"Destination"`
	RW          bool   `json:"RW"`
}

// inspectEndpoint 是 inspect 里 NetworkSettings.Networks 的一项。
type inspectEndpoint struct {
	Aliases    []string       `json:"Aliases"`
	IPAMConfig map[string]any `json:"IPAMConfig"`
	Links      []string       `json:"Links"`
	IPAddress  string         `json:"IPAddress"`
	Gateway    string         `json:"Gateway"`
	MacAddress string         `json:"MacAddress"`
}

func decodeField(info map[string]json.RawMessage, key string, out any) {
	if raw, ok := info[key]; ok && len(raw) > 0 {
		_ = json.Unmarshal(raw, out)
	}
}

// networkMode 把 Engine 的 NetworkMode 转成界面使用的网络名（default 即 bridge）。
func networkMode(mode string) string {
	if mode == "default" {
		return "bridge"
	}
	return mode
}

// noEndpointMode 判断网络模式是否不支持端口映射、别名与加入其它网络。
func noEndpointMode(mode string) bool {
	return mode == "host" || mode == "none" || strings.HasPrefix(mode, "container:")
}

// specFromInspect 把 inspect 结果还原成 ContainerSpec。img 为镜像默认配置，可为空。
func specFromInspect(info map[string]json.RawMessage, img *imageDefaults) ContainerSpec {
	var cfg inspectConfig
	var host inspectHost
	var mounts []inspectMount
	var ns struct {
		Networks map[string]inspectEndpoint `json:"Networks"`
	}
	var id, name string
	decodeField(info, "Config", &cfg)
	decodeField(info, "HostConfig", &host)
	decodeField(info, "Mounts", &mounts)
	decodeField(info, "NetworkSettings", &ns)
	decodeField(info, "Id", &id)
	decodeField(info, "Name", &name)

	spec := ContainerSpec{
		Name:              trimSlash(name),
		Image:             cfg.Image,
		Env:               append([]string{}, cfg.Env...),
		Ports:             []SpecPort{},
		Volumes:           []SpecVolume{},
		Networks:          []string{},
		Aliases:           []string{},
		Restart:           host.RestartPolicy.Name,
		RestartMaxRetries: host.RestartPolicy.MaximumRetryCount,
		Privileged:        host.Privileged,
		CapAdd:            nonNil(host.CapAdd),
		CapDrop:           nonNil(host.CapDrop),
		Devices:           []SpecDevice{},
		User:              cfg.User,
		Workdir:           cfg.WorkingDir,
		Hostname:          cfg.Hostname,
		DNS:               nonNil(host.DNS),
		Labels:            map[string]string{},
		TTY:               cfg.Tty,
		StdinOpen:         cfg.OpenStdin,
		Command:           shellJoin(cfg.Cmd),
		Entrypoint:        shellJoin(cfg.Entrypoint),
	}
	for k, v := range cfg.Labels {
		spec.Labels[k] = v
	}
	if spec.Restart == "" {
		spec.Restart = "no"
	}
	// 默认主机名取自容器 ID，不属于用户配置
	if spec.Hostname != "" && id != "" && strings.HasPrefix(id, spec.Hostname) {
		spec.Hostname = ""
	}
	if host.Memory > 0 {
		spec.MemoryMB = host.Memory >> 20
		if spec.MemoryMB == 0 {
			spec.MemoryMB = 1
		}
	}
	if host.NanoCPUs > 0 {
		spec.CPUs = float64(host.NanoCPUs) / 1e9
	}
	// 去掉继承自镜像的默认值，编辑时留空即表示沿用镜像
	if img != nil {
		if slices.Equal(cfg.Cmd, img.Cmd) {
			spec.Command = ""
		}
		if slices.Equal(cfg.Entrypoint, img.Entrypoint) {
			spec.Entrypoint = ""
		}
		if len(img.Env) > 0 {
			inherited := map[string]bool{}
			for _, e := range img.Env {
				inherited[e] = true
			}
			spec.Env = slices.DeleteFunc(spec.Env, func(e string) bool { return inherited[e] })
		}
		for k, v := range img.Labels {
			if cur, ok := spec.Labels[k]; ok && cur == v {
				delete(spec.Labels, k)
			}
		}
		if spec.User == img.User {
			spec.User = ""
		}
		if spec.Workdir == img.WorkingDir {
			spec.Workdir = ""
		}
	}

	// 端口
	keys := make([]string, 0, len(host.PortBindings))
	for k := range host.PortBindings {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		portStr, proto, _ := strings.Cut(k, "/")
		cport, err := strconv.Atoi(portStr)
		if err != nil {
			continue
		}
		if proto == "" {
			proto = "tcp"
		}
		for _, b := range host.PortBindings[k] {
			hp, _ := strconv.Atoi(b.HostPort)
			spec.Ports = append(spec.Ports, SpecPort{HostIP: b.HostIP, Host: hp, Container: cport, Proto: proto})
		}
	}
	sort.SliceStable(spec.Ports, func(i, j int) bool {
		if spec.Ports[i].Container != spec.Ports[j].Container {
			return spec.Ports[i].Container < spec.Ports[j].Container
		}
		return spec.Ports[i].Proto < spec.Ports[j].Proto
	})

	// 挂载：只还原 bind 与 volume 两类，tmpfs 等保持原配置不动
	for _, m := range mounts {
		switch m.Type {
		case "bind":
			spec.Volumes = append(spec.Volumes, SpecVolume{Type: "bind", Source: m.Source, Target: m.Destination, ReadOnly: !m.RW})
		case "volume":
			spec.Volumes = append(spec.Volumes, SpecVolume{Type: "volume", Source: m.Name, Target: m.Destination, ReadOnly: !m.RW})
		}
	}
	sort.SliceStable(spec.Volumes, func(i, j int) bool { return spec.Volumes[i].Target < spec.Volumes[j].Target })

	for _, d := range host.Devices {
		spec.Devices = append(spec.Devices, SpecDevice{Host: d.PathOnHost, Container: d.PathInContainer, Perm: d.CgroupPermissions})
	}

	// 网络：主网络取 NetworkMode，其余为额外网络
	names := make([]string, 0, len(ns.Networks))
	for n := range ns.Networks {
		names = append(names, n)
	}
	sort.Strings(names)
	primary := networkMode(host.NetworkMode)
	if primary == "" && len(names) > 0 {
		primary = names[0]
	}
	spec.Network = primary
	for _, n := range names {
		if n != primary && n != host.NetworkMode {
			spec.Networks = append(spec.Networks, n)
		}
	}
	ep, ok := ns.Networks[primary]
	if !ok {
		ep, ok = ns.Networks[host.NetworkMode]
	}
	if ok {
		for _, a := range ep.Aliases {
			// 容器 ID 前缀是 Docker 自动加的别名
			if id != "" && strings.HasPrefix(id, a) && len(a) >= 12 {
				continue
			}
			spec.Aliases = append(spec.Aliases, a)
		}
		if v, _ := ep.IPAMConfig["IPv4Address"].(string); v != "" {
			spec.IP = v
		} else if v, _ := ep.IPAMConfig["IPv6Address"].(string); v != "" {
			spec.IP = v
		}
	}
	return spec
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return append([]string{}, s...)
}

// ---------- spec -> 创建请求 ----------

var containerNameRe = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]*$`)

// validateSpec 校验 spec 里与 Engine 无关的基本约束，返回明确的中文错误。
func validateSpec(spec *ContainerSpec) error {
	spec.Image = strings.TrimSpace(spec.Image)
	spec.Name = strings.TrimSpace(spec.Name)
	if spec.Image == "" {
		return errors.New("镜像不能为空")
	}
	if spec.Name != "" && !containerNameRe.MatchString(spec.Name) {
		return fmt.Errorf("容器名称 %q 无效，只能包含字母、数字、下划线、点和连字符，且以字母或数字开头", spec.Name)
	}
	switch spec.Restart {
	case "", "no", "always", "unless-stopped", "on-failure":
	default:
		return fmt.Errorf("重启策略 %q 无效，可选 no / always / unless-stopped / on-failure", spec.Restart)
	}
	if spec.RestartMaxRetries < 0 {
		return errors.New("最大重启次数不能为负数")
	}
	if spec.MemoryMB < 0 || spec.CPUs < 0 || math.IsNaN(spec.CPUs) || math.IsInf(spec.CPUs, 0) {
		return errors.New("内存与 CPU 限制不能为负数")
	}
	if spec.IP != "" && net.ParseIP(spec.IP) == nil {
		return fmt.Errorf("固定 IP %q 格式无效", spec.IP)
	}
	return nil
}

// applySpec 把 spec 覆盖到请求体 body 上（含 HostConfig 与 NetworkingConfig），返回创建后需要追加连接的网络。
// info 为旧容器的 inspect 结果（创建新容器时为 nil），用于沿用其中无法用 spec 表达的细节。
func applySpec(spec ContainerSpec, body map[string]any, info map[string]json.RawMessage) ([]networkAttach, error) {
	if err := validateSpec(&spec); err != nil {
		return nil, err
	}
	hostCfg, _ := body["HostConfig"].(map[string]any)
	if hostCfg == nil {
		hostCfg = map[string]any{}
		body["HostConfig"] = hostCfg
	}
	setOrDelete := func(m map[string]any, key string, v any, empty bool) {
		if empty {
			delete(m, key)
			return
		}
		m[key] = v
	}

	body["Image"] = spec.Image
	// 命令与入口点：留空表示使用镜像默认
	cmd, err := shellSplit(spec.Command)
	if err != nil {
		return nil, fmt.Errorf("命令格式错误: %w", err)
	}
	setOrDelete(body, "Cmd", cmd, len(cmd) == 0)
	ep, err := shellSplit(spec.Entrypoint)
	if err != nil {
		return nil, fmt.Errorf("入口点格式错误: %w", err)
	}
	setOrDelete(body, "Entrypoint", ep, len(ep) == 0)
	setOrDelete(body, "Env", spec.Env, len(spec.Env) == 0)
	setOrDelete(body, "User", spec.User, spec.User == "")
	setOrDelete(body, "WorkingDir", spec.Workdir, spec.Workdir == "")
	setOrDelete(body, "Hostname", spec.Hostname, spec.Hostname == "")
	setOrDelete(body, "Labels", spec.Labels, len(spec.Labels) == 0)
	body["Tty"] = spec.TTY
	body["OpenStdin"] = spec.StdinOpen

	// 网络模式先确定，端口与别名依赖它
	mode := networkMode(spec.Network)
	if mode == "" {
		if old, _ := hostCfg["NetworkMode"].(string); old != "" {
			mode = networkMode(old)
		}
	}
	noEndpoint := noEndpointMode(mode)
	if noEndpoint {
		switch {
		case len(spec.Ports) > 0:
			return nil, fmt.Errorf("网络模式 %s 不支持端口映射，请移除端口或改用其它网络", mode)
		case len(spec.Aliases) > 0:
			return nil, fmt.Errorf("网络模式 %s 不支持网络别名", mode)
		case spec.IP != "":
			return nil, fmt.Errorf("网络模式 %s 不支持固定 IP", mode)
		case len(spec.Networks) > 0:
			return nil, fmt.Errorf("网络模式 %s 不能再加入其它网络", mode)
		}
	}

	if err := applyPorts(spec.Ports, body, hostCfg); err != nil {
		return nil, err
	}
	if err := applyVolumes(spec.Volumes, body, hostCfg); err != nil {
		return nil, err
	}

	// 重启策略
	restart := spec.Restart
	if restart == "" {
		restart = "no"
	}
	rp := map[string]any{"Name": restart}
	if restart == "on-failure" {
		rp["MaximumRetryCount"] = spec.RestartMaxRetries
	}
	hostCfg["RestartPolicy"] = rp
	hostCfg["Privileged"] = spec.Privileged
	setOrDelete(hostCfg, "CapAdd", spec.CapAdd, len(spec.CapAdd) == 0)
	setOrDelete(hostCfg, "CapDrop", spec.CapDrop, len(spec.CapDrop) == 0)
	setOrDelete(hostCfg, "Dns", spec.DNS, len(spec.DNS) == 0)

	devices := make([]map[string]string, 0, len(spec.Devices))
	for _, d := range spec.Devices {
		if d.Host == "" {
			return nil, errors.New("设备映射的宿主机路径不能为空")
		}
		cpath := d.Container
		if cpath == "" {
			cpath = d.Host
		}
		perm := d.Perm
		if perm == "" {
			perm = "rwm"
		}
		devices = append(devices, map[string]string{"PathOnHost": d.Host, "PathInContainer": cpath, "CgroupPermissions": perm})
	}
	setOrDelete(hostCfg, "Devices", devices, len(devices) == 0)

	applyResources(spec, hostCfg)
	return applyNetworks(spec, mode, noEndpoint, body, hostCfg, info)
}

// applyResources 设置内存与 CPU 限制。数值与旧配置换算后一致时保留旧值，避免编辑一次就改变精确的字节数。
func applyResources(spec ContainerSpec, hostCfg map[string]any) {
	oldMem := int64(anyToFloat(hostCfg["Memory"]))
	newMem := spec.MemoryMB << 20
	if spec.MemoryMB > 0 && oldMem > 0 && max(oldMem>>20, 1) == spec.MemoryMB {
		newMem = oldMem
	}
	if newMem != oldMem {
		// 内存上限变化后沿用旧的 swap 上限可能小于内存而被拒绝，交给 Docker 取默认值
		hostCfg["MemorySwap"] = 0
	}
	hostCfg["Memory"] = newMem

	oldCPU := int64(anyToFloat(hostCfg["NanoCpus"]))
	newCPU := int64(math.Round(spec.CPUs * 1e9))
	if oldCPU > 0 && math.Abs(float64(oldCPU)/1e9-spec.CPUs) < 1e-6 {
		newCPU = oldCPU
	}
	hostCfg["NanoCpus"] = newCPU
}

func anyToFloat(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case int64:
		return float64(n)
	case int:
		return float64(n)
	}
	return 0
}

// applyPorts 生成 ExposedPorts 与 PortBindings。
func applyPorts(ports []SpecPort, body, hostCfg map[string]any) error {
	exposed, _ := body["ExposedPorts"].(map[string]any)
	if exposed == nil {
		exposed = map[string]any{}
	}
	bindings := map[string][]map[string]string{}
	seen := map[string]bool{}
	for _, p := range ports {
		if p.Container < 1 || p.Container > 65535 {
			return fmt.Errorf("容器端口 %d 无效", p.Container)
		}
		if p.Host < 0 || p.Host > 65535 {
			return fmt.Errorf("宿主机端口 %d 无效", p.Host)
		}
		proto := strings.ToLower(p.Proto)
		if proto == "" {
			proto = "tcp"
		}
		if proto != "tcp" && proto != "udp" && proto != "sctp" {
			return fmt.Errorf("端口协议 %q 无效，可选 tcp / udp / sctp", p.Proto)
		}
		if p.HostIP != "" && net.ParseIP(p.HostIP) == nil {
			return fmt.Errorf("监听地址 %q 格式无效", p.HostIP)
		}
		if p.Host > 0 {
			key := fmt.Sprintf("%s|%d|%s", p.HostIP, p.Host, proto)
			if seen[key] {
				return fmt.Errorf("宿主机端口 %d/%s 重复映射", p.Host, proto)
			}
			seen[key] = true
		}
		key := fmt.Sprintf("%d/%s", p.Container, proto)
		exposed[key] = map[string]any{}
		hostPort := ""
		if p.Host > 0 {
			hostPort = strconv.Itoa(p.Host)
		}
		bindings[key] = append(bindings[key], map[string]string{"HostIp": p.HostIP, "HostPort": hostPort})
	}
	if len(exposed) > 0 {
		body["ExposedPorts"] = exposed
	}
	if len(bindings) == 0 {
		delete(hostCfg, "PortBindings")
		return nil
	}
	hostCfg["PortBindings"] = bindings
	return nil
}

// bindParts 拆分 Binds 条目 src:dst[:opts]。
func bindParts(b string) (src, dst, opts string) {
	parts := strings.SplitN(b, ":", 3)
	switch len(parts) {
	case 1:
		return "", parts[0], ""
	case 2:
		return parts[0], parts[1], ""
	}
	return parts[0], parts[1], parts[2]
}

func hasOpt(opts, want string) bool {
	return slices.Contains(strings.Split(opts, ","), want)
}

// applyVolumes 把 spec 里的挂载写入 Binds。旧容器上与 spec 一致的挂载（含 :z 之类的附加选项）原样沿用，
// 通过 HostConfig.Mounts 声明的挂载在目标仍存在时保留，已移除的挂载随之删除。
func applyVolumes(vols []SpecVolume, body, hostCfg map[string]any) error {
	oldBinds := map[string]string{} // 目标路径 -> 旧 Binds 条目
	if bs, ok := hostCfg["Binds"].([]any); ok {
		for _, b := range bs {
			if s, ok := b.(string); ok {
				_, dst, _ := bindParts(s)
				oldBinds[dst] = s
			}
		}
	}
	// HostConfig.Mounts 中 bind / volume 以外的类型（tmpfs 等）无法用 spec 表达，原样保留
	coveredByMounts := map[string]bool{}
	var keepMounts []any
	if ms, ok := hostCfg["Mounts"].([]any); ok {
		for _, m := range ms {
			mm, ok := m.(map[string]any)
			if !ok {
				continue
			}
			typ, _ := mm["Type"].(string)
			target, _ := mm["Target"].(string)
			if typ != "bind" && typ != "volume" {
				keepMounts = append(keepMounts, m)
				continue
			}
			for _, v := range vols {
				ro, _ := mm["ReadOnly"].(bool)
				src, _ := mm["Source"].(string)
				if v.Target == target && v.Source == src && v.ReadOnly == ro {
					keepMounts = append(keepMounts, m)
					coveredByMounts[target] = true
					break
				}
			}
		}
	}

	var binds []string
	anonymous := map[string]any{}
	seenTarget := map[string]bool{}
	for _, v := range vols {
		if !path.IsAbs(v.Target) {
			return fmt.Errorf("挂载目标 %q 必须是绝对路径", v.Target)
		}
		target := path.Clean(v.Target)
		if seenTarget[target] {
			return fmt.Errorf("挂载目标 %s 重复", target)
		}
		seenTarget[target] = true
		typ := v.Type
		if typ == "" {
			if strings.HasPrefix(v.Source, "/") {
				typ = "bind"
			} else {
				typ = "volume"
			}
		}
		switch typ {
		case "bind":
			if !strings.HasPrefix(v.Source, "/") {
				return fmt.Errorf("目录挂载的宿主机路径 %q 必须是绝对路径", v.Source)
			}
		case "volume":
			if v.Source == "" {
				// 未命名的卷是匿名卷，由 Docker 在启动时创建
				anonymous[target] = map[string]any{}
				continue
			}
		default:
			return fmt.Errorf("挂载类型 %q 无效，可选 bind / volume", v.Type)
		}
		if strings.Contains(v.Source, ":") || strings.Contains(target, ":") {
			return fmt.Errorf("挂载 %s:%s 的路径不能包含冒号", v.Source, v.Target)
		}
		if coveredByMounts[target] {
			continue
		}
		entry := v.Source + ":" + target
		if v.ReadOnly {
			entry += ":ro"
		}
		if old, ok := oldBinds[target]; ok {
			src, _, opts := bindParts(old)
			if src == v.Source && hasOpt(opts, "ro") == v.ReadOnly {
				entry = old
			}
		}
		binds = append(binds, entry)
	}
	if len(binds) == 0 {
		delete(hostCfg, "Binds")
	} else {
		hostCfg["Binds"] = binds
	}
	if len(keepMounts) == 0 {
		delete(hostCfg, "Mounts")
	} else {
		hostCfg["Mounts"] = keepMounts
	}
	if len(anonymous) > 0 {
		vm, _ := body["Volumes"].(map[string]any)
		if vm == nil {
			vm = map[string]any{}
		}
		for k, v := range anonymous {
			vm[k] = v
		}
		body["Volumes"] = vm
	}
	return nil
}

// applyNetworks 设置主网络与创建后追加连接的其它网络。
func applyNetworks(spec ContainerSpec, mode string, noEndpoint bool, body, hostCfg map[string]any, info map[string]json.RawMessage) ([]networkAttach, error) {
	var oldNets struct {
		Networks map[string]inspectEndpoint `json:"Networks"`
	}
	if info != nil {
		decodeField(info, "NetworkSettings", &oldNets)
	}
	endpoints := map[string]any{}
	var extra []networkAttach
	if noEndpoint {
		hostCfg["NetworkMode"] = mode
		body["NetworkingConfig"] = map[string]any{"EndpointsConfig": endpoints}
		return nil, nil
	}
	if mode == "" && (len(spec.Aliases) > 0 || spec.IP != "") {
		// 别名与固定 IP 需要明确的主网络
		mode = "bridge"
	}
	if mode != "" {
		hostCfg["NetworkMode"] = mode
		ep := map[string]any{}
		if len(spec.Aliases) > 0 {
			ep["Aliases"] = spec.Aliases
		}
		if spec.IP != "" {
			key := "IPv4Address"
			if ip := net.ParseIP(spec.IP); ip != nil && ip.To4() == nil {
				key = "IPv6Address"
			}
			ep["IPAMConfig"] = map[string]any{key: spec.IP}
		}
		endpoints[mode] = ep
	}
	seen := map[string]bool{mode: true, "": true}
	for _, n := range spec.Networks {
		n = strings.TrimSpace(n)
		if seen[n] {
			continue
		}
		seen[n] = true
		if noEndpointMode(n) {
			return nil, fmt.Errorf("不能把网络 %s 作为额外网络加入", n)
		}
		ep := map[string]any{}
		if old, ok := oldNets.Networks[n]; ok {
			if len(old.Aliases) > 0 {
				ep["Aliases"] = old.Aliases
			}
			if len(old.IPAMConfig) > 0 {
				ep["IPAMConfig"] = old.IPAMConfig
			}
			if len(old.Links) > 0 {
				ep["Links"] = old.Links
			}
		}
		extra = append(extra, networkAttach{name: n, endpoint: ep})
	}
	body["NetworkingConfig"] = map[string]any{"EndpointsConfig": endpoints}
	return extra, nil
}
