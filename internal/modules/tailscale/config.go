package tailscale

import (
	"errors"
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
)

// Settings 是 Tailscale 配置。
type Settings struct {
	Enable bool `json:"enable"`
	// ManageDaemon 为 true 时由 Gately 运行 tailscaled；为 false 时使用系统已有的 tailscaled，只负责 tailscale up 与状态展示
	ManageDaemon   bool   `json:"manageDaemon"`
	TailscaledPath string `json:"tailscaledPath"`
	TailscalePath  string `json:"tailscalePath"`
	// StateDir 保存登录状态，留空为配置目录下的 tailscale 子目录
	StateDir string `json:"stateDir"`
	// Socket 为 tailscaled 的控制套接字，托管时留空为 StateDir 下的 tailscaled.sock，外部守护进程留空使用系统默认
	Socket string `json:"socket"`
	// Userspace 使用用户态网络，无需 TUN 设备与特权，适合容器；此时本机服务经 Socks5 访问其他节点
	Userspace bool `json:"userspace"`
	// Socks5 为用户态网络下的 SOCKS5 代理监听地址，例如 localhost:1055
	Socks5 string `json:"socks5"`
	// AuthKey 为认证密钥，留空时需要在状态面板中打开登录链接完成认证
	AuthKey     string `json:"authKey"`
	Hostname    string `json:"hostname"`
	LoginServer string `json:"loginServer"`
	// AdvertiseRoutes 是向 tailnet 发布的子网路由（CIDR）
	AdvertiseRoutes   []string `json:"advertiseRoutes"`
	AdvertiseExitNode bool     `json:"advertiseExitNode"`
	AcceptRoutes      bool     `json:"acceptRoutes"`
	AcceptDNS         bool     `json:"acceptDns"`
	// ExtraUpArgs 追加到 tailscale up 之后，以空白分隔
	ExtraUpArgs string `json:"extraUpArgs"`
}

// DefaultSettings 是首次使用的默认配置：托管 tailscaled 并使用用户态网络。
func DefaultSettings() Settings {
	return Settings{ManageDaemon: true, Userspace: true}
}

var hostnameRe = regexp.MustCompile(`^[A-Za-z0-9]([A-Za-z0-9-]{0,61}[A-Za-z0-9])?$`)

// Validate 校验并规范化配置。
func (s *Settings) Validate() error {
	s.TailscaledPath = strings.TrimSpace(s.TailscaledPath)
	s.TailscalePath = strings.TrimSpace(s.TailscalePath)
	s.StateDir = strings.TrimSpace(s.StateDir)
	s.Socket = strings.TrimSpace(s.Socket)
	s.Socks5 = strings.TrimSpace(s.Socks5)
	s.AuthKey = strings.TrimSpace(s.AuthKey)
	s.Hostname = strings.TrimSpace(s.Hostname)
	s.LoginServer = strings.TrimSpace(s.LoginServer)
	s.ExtraUpArgs = strings.TrimSpace(s.ExtraUpArgs)
	if s.StateDir != "" && !filepath.IsAbs(s.StateDir) {
		return errors.New("状态目录需为绝对路径")
	}
	if s.Socks5 != "" {
		if _, _, err := net.SplitHostPort(s.Socks5); err != nil {
			return errors.New("SOCKS5 地址格式应为 主机:端口")
		}
	}
	if s.Hostname != "" && !hostnameRe.MatchString(s.Hostname) {
		return errors.New("主机名只能包含字母、数字和连字符")
	}
	if s.LoginServer != "" {
		u, err := url.Parse(s.LoginServer)
		if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
			return errors.New("控制服务器地址格式应为 https://headscale.example.com")
		}
	}
	var routes []string
	for _, r := range s.AdvertiseRoutes {
		if r = strings.TrimSpace(r); r == "" {
			continue
		}
		p, err := netip.ParsePrefix(r)
		if err != nil {
			return fmt.Errorf("子网路由 %q 不是有效的 CIDR", r)
		}
		routes = append(routes, p.Masked().String())
	}
	s.AdvertiseRoutes = routes
	return nil
}

// stateDir 返回实际使用的状态目录。
func (s *Settings) stateDir(configDir string) string {
	if s.StateDir != "" {
		return s.StateDir
	}
	return filepath.Join(configDir, "tailscale")
}

// socket 返回实际使用的控制套接字路径，空串表示使用 tailscale 命令的系统默认值。
func (s *Settings) socket(configDir string) string {
	if s.Socket != "" {
		return s.Socket
	}
	if s.ManageDaemon {
		return filepath.Join(s.stateDir(configDir), "tailscaled.sock")
	}
	return ""
}

// DaemonArgs 生成 tailscaled 的命令行参数。
func (s *Settings) DaemonArgs(configDir string) []string {
	args := []string{"--statedir=" + s.stateDir(configDir), "--socket=" + s.socket(configDir)}
	if s.Userspace {
		args = append(args, "--tun=userspace-networking")
	}
	if s.Socks5 != "" {
		args = append(args, "--socks5-server="+s.Socks5)
	}
	return args
}

// CLIArgs 生成 tailscale 命令的全局参数。
func (s *Settings) CLIArgs(configDir string) []string {
	if sock := s.socket(configDir); sock != "" {
		return []string{"--socket=" + sock}
	}
	return nil
}

// UpArgs 生成 tailscale up 的参数。使用 --reset 让界面配置成为唯一的事实来源，
// 否则 tailscale 会要求列出所有非默认参数。
func (s *Settings) UpArgs() []string {
	args := []string{"up", "--reset", fmt.Sprintf("--accept-routes=%t", s.AcceptRoutes), fmt.Sprintf("--accept-dns=%t", s.AcceptDNS)}
	if s.AuthKey != "" {
		args = append(args, "--auth-key="+s.AuthKey)
	}
	if s.Hostname != "" {
		args = append(args, "--hostname="+s.Hostname)
	}
	if s.LoginServer != "" {
		args = append(args, "--login-server="+s.LoginServer)
	}
	if len(s.AdvertiseRoutes) > 0 {
		args = append(args, "--advertise-routes="+strings.Join(s.AdvertiseRoutes, ","))
	}
	if s.AdvertiseExitNode {
		args = append(args, "--advertise-exit-node")
	}
	return append(args, strings.Fields(s.ExtraUpArgs)...)
}
