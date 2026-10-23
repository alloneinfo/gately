// Package sysstat 采集概览页展示的主机与进程信息：CPU、内存、磁盘、网卡与实时速率。
//
// 只读取本机信息（Linux 读取 /proc），不访问任何外部地址。速率与使用率通过相邻两次采样的差值计算，
// 因此首次调用会额外采样一次。不支持的平台上对应字段留空，由前端显示为“-”。
package sysstat

import (
	"net"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Snapshot 是一次采集的结果。
type Snapshot struct {
	Hostname    string      `json:"hostname"`
	OS          string      `json:"os"`
	Platform    string      `json:"platform,omitempty"`
	Kernel      string      `json:"kernel,omitempty"`
	Arch        string      `json:"arch"`
	InContainer bool        `json:"inContainer"`
	BootTime    int64       `json:"bootTime,omitempty"`
	CPU         CPU         `json:"cpu"`
	Memory      *Memory     `json:"memory,omitempty"`
	Disk        *Disk       `json:"disk,omitempty"`
	Interfaces  []Interface `json:"interfaces"`
	Process     Process     `json:"process"`
	SampledAt   time.Time   `json:"sampledAt"`
}

// CPU 描述处理器信息。Usage 为整机使用率百分比，未知时为 nil。
type CPU struct {
	Model string    `json:"model,omitempty"`
	Cores int       `json:"cores"`
	Usage *float64  `json:"usage,omitempty"`
	Load  []float64 `json:"load,omitempty"`
}

// Memory 描述物理内存与交换分区，单位字节。
type Memory struct {
	Total     uint64 `json:"total"`
	Used      uint64 `json:"used"`
	Available uint64 `json:"available"`
	SwapTotal uint64 `json:"swapTotal"`
	SwapUsed  uint64 `json:"swapUsed"`
}

// Disk 描述配置目录所在分区的容量，单位字节。
type Disk struct {
	Path  string `json:"path"`
	Total uint64 `json:"total"`
	Used  uint64 `json:"used"`
	Free  uint64 `json:"free"`
}

// Interface 描述一个网卡。速率单位为字节/秒，未知时为 nil。
type Interface struct {
	Name    string   `json:"name"`
	MAC     string   `json:"mac,omitempty"`
	MTU     int      `json:"mtu"`
	IPv4    []string `json:"ipv4"`
	IPv6    []string `json:"ipv6"`
	RxBytes uint64   `json:"rxBytes"`
	TxBytes uint64   `json:"txBytes"`
	RxRate  *float64 `json:"rxRate,omitempty"`
	TxRate  *float64 `json:"txRate,omitempty"`
}

// Process 描述 Gately 进程自身的资源占用。
type Process struct {
	PID        int      `json:"pid"`
	RSS        uint64   `json:"rss,omitempty"`
	CPU        *float64 `json:"cpu,omitempty"`
	Threads    int      `json:"threads,omitempty"`
	FDs        int      `json:"fds,omitempty"`
	Goroutines int      `json:"goroutines"`
	HeapAlloc  uint64   `json:"heapAlloc"`
	HeapSys    uint64   `json:"heapSys"`
	NumGC      uint32   `json:"numGC"`
}

// rawSample 是计算差值所需的原始计数器。
type rawSample struct {
	at        time.Time
	cpuTotal  uint64
	cpuIdle   uint64
	procTicks uint64
	ifBytes   map[string][2]uint64
	ok        bool
}

// Collector 保存上一次采样，用于计算使用率与速率。并发安全。
type Collector struct {
	mu   sync.Mutex
	dir  string
	last rawSample
}

// New 创建采集器，dir 为统计磁盘容量的目录（一般为配置目录）。
func New(dir string) *Collector {
	return &Collector{dir: dir}
}

// 采样间隔过短时差值噪声大，过长时数据已不代表“当前”，此时补采一次
const (
	minInterval = 300 * time.Millisecond
	maxInterval = 60 * time.Second
)

// Snapshot 采集一次主机信息。
func (c *Collector) Snapshot() Snapshot {
	c.mu.Lock()
	defer c.mu.Unlock()

	cur := readRaw()
	prev := c.last
	if cur.ok && (!prev.ok || cur.at.Sub(prev.at) > maxInterval) {
		// 没有可用的上一次采样：短暂等待后再采一次，保证首屏就有使用率与速率
		prev = cur
		time.Sleep(minInterval)
		cur = readRaw()
	}

	s := Snapshot{
		OS:          runtime.GOOS,
		Arch:        runtime.GOARCH,
		InContainer: inContainer(),
		SampledAt:   time.Now(),
	}
	s.Hostname, _ = os.Hostname()
	s.CPU.Cores = runtime.NumCPU()
	fillPlatform(&s)
	if c.dir != "" {
		s.Disk = diskUsage(c.dir)
	}
	s.Interfaces = interfaces(cur, prev)
	fillProcess(&s.Process)

	if cur.ok && prev.ok {
		dt := cur.at.Sub(prev.at).Seconds()
		if dt > 0 && cur.cpuTotal > prev.cpuTotal {
			total := float64(cur.cpuTotal - prev.cpuTotal)
			idle := float64(cur.cpuIdle - prev.cpuIdle)
			usage := clampPercent((total - idle) / total * 100)
			s.CPU.Usage = &usage
		}
		if dt > 0 && cur.procTicks >= prev.procTicks {
			// 进程 CPU 按整机核数归一化，与整机使用率口径一致
			p := clampPercent(float64(cur.procTicks-prev.procTicks) / clockTicks / dt / float64(s.CPU.Cores) * 100)
			s.Process.CPU = &p
		}
	}
	// 两次请求间隔太近时保留更早的基准，下一次差值的时间跨度足够长、结果更平稳
	if cur.ok && (!c.last.ok || cur.at.Sub(c.last.at) >= minInterval) {
		c.last = cur
	}
	return s
}

func clampPercent(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 100 {
		return 100
	}
	return v
}

// virtualPrefixes 是容器与虚拟化产生的网卡前缀，数量多且对用户意义不大，概览中隐藏。
// Docker 自定义网络的网桥另由 isDockerBridge 识别：OpenWrt 的 br-lan、br-wan 同样以 br- 开头，是真实的局域网口。
var virtualPrefixes = []string{"veth", "docker", "virbr", "cni", "flannel", "vnet", "tap", "dummy", "kube", "cali", "lxc"}

func isVirtual(name string) bool {
	if isDockerBridge(name) {
		return true
	}
	for _, p := range virtualPrefixes {
		if strings.HasPrefix(name, p) {
			return true
		}
	}
	return false
}

// isDockerBridge 判断网卡是否为 Docker 自定义网络的网桥，其名称固定为 br- 加网络 ID 的前 12 位十六进制。
func isDockerBridge(name string) bool {
	id, ok := strings.CutPrefix(name, "br-")
	if !ok || len(id) != 12 {
		return false
	}
	for _, c := range id {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}

// interfaces 列出处于启用状态、配置了地址的网卡，并附带流量计数与速率。
func interfaces(cur, prev rawSample) []Interface {
	list, err := net.Interfaces()
	if err != nil {
		return []Interface{}
	}
	dt := cur.at.Sub(prev.at).Seconds()
	out := []Interface{}
	for _, ifc := range list {
		if ifc.Flags&net.FlagUp == 0 || ifc.Flags&net.FlagLoopback != 0 || isVirtual(ifc.Name) {
			continue
		}
		it := Interface{Name: ifc.Name, MAC: ifc.HardwareAddr.String(), MTU: ifc.MTU, IPv4: []string{}, IPv6: []string{}}
		addrs, _ := ifc.Addrs()
		routable := false
		for _, a := range addrs {
			ipn, ok := a.(*net.IPNet)
			if !ok {
				continue
			}
			ones, _ := ipn.Mask.Size()
			if !ipn.IP.IsLinkLocalUnicast() {
				routable = true
			}
			if ip4 := ipn.IP.To4(); ip4 != nil {
				it.IPv4 = append(it.IPv4, ip4.String()+"/"+strconv.Itoa(ones))
			} else {
				it.IPv6 = append(it.IPv6, ipn.IP.String()+"/"+strconv.Itoa(ones))
			}
		}
		// 只有链路本地地址的网卡（如 macOS 的 awdl0、未拨号的接口）不展示
		if !routable {
			continue
		}
		if b, ok := cur.ifBytes[ifc.Name]; ok {
			it.RxBytes, it.TxBytes = b[0], b[1]
			if p, ok := prev.ifBytes[ifc.Name]; ok && dt > 0 && b[0] >= p[0] && b[1] >= p[1] {
				rx := float64(b[0]-p[0]) / dt
				tx := float64(b[1]-p[1]) / dt
				it.RxRate, it.TxRate = &rx, &tx
			}
		}
		out = append(out, it)
	}
	// 有 IPv4 的网卡排在前面，其次按名称排序，结果稳定便于前端展示
	sort.SliceStable(out, func(i, j int) bool {
		hi, hj := len(out[i].IPv4) > 0, len(out[j].IPv4) > 0
		if hi != hj {
			return hi
		}
		return out[i].Name < out[j].Name
	})
	return out
}

// fillProcess 填充 Go 运行时信息，平台相关字段由 fillProcessOS 补充。
func fillProcess(p *Process) {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	p.PID = os.Getpid()
	p.Goroutines = runtime.NumGoroutine()
	p.HeapAlloc = ms.HeapAlloc
	p.HeapSys = ms.Sys
	p.NumGC = ms.NumGC
	fillProcessOS(p)
	if p.RSS == 0 {
		// 无法读取常驻内存时，用向系统申请的内存近似
		p.RSS = ms.Sys
	}
}

func inContainer() bool {
	for _, f := range []string{"/.dockerenv", "/run/.containerenv"} {
		if _, err := os.Stat(f); err == nil {
			return true
		}
	}
	return false
}
