package webservice

import (
	"errors"
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"path/filepath"
	"strings"
)

// 站点类型
const (
	TypeProxy    = "proxy"
	TypeRedirect = "redirect"
	TypeStatic   = "static"
	TypeText     = "text"
)

// 访问控制模式
const (
	IPModeOff       = ""
	IPModeWhitelist = "whitelist"
	IPModeBlacklist = "blacklist"
)

// Site 是监听器下按域名匹配的一个站点。
type Site struct {
	Key    string `json:"key"`
	Name   string `json:"name"`
	Enable bool   `json:"enable"`
	// Domains 支持精确域名、*.example.com 通配（匹配任意级子域名）与 *（匹配所有）
	Domains []string `json:"domains"`
	Type    string   `json:"type"`

	// 反向代理
	Targets            []string `json:"targets"`
	InsecureSkipVerify bool     `json:"insecureSkipVerify"`
	// UseTargetHost 为 true 时 Host 头使用目标地址，否则保留客户端请求的 Host
	UseTargetHost     bool `json:"useTargetHost"`
	DisableKeepAlives bool `json:"disableKeepAlives"`
	// SendProxyProtocol 向后端发送 PROXY v2 头传递真实客户端 IP（需要后端支持）
	SendProxyProtocol bool `json:"sendProxyProtocol"`
	// ForwardedHeaders 添加 X-Forwarded-For/Proto/Host
	ForwardedHeaders bool `json:"forwardedHeaders"`

	// 重定向：RedirectTo 为目标地址前缀，KeepPath 时附加原路径与查询串
	RedirectTo   string `json:"redirectTo"`
	RedirectCode int    `json:"redirectCode"`
	KeepPath     bool   `json:"keepPath"`

	// 静态文件
	Root    string `json:"root"`
	ListDir bool   `json:"listDir"`

	// 固定文本
	Text        string `json:"text"`
	ContentType string `json:"contentType"`
	StatusCode  int    `json:"statusCode"`

	// 认证：Users 每项为 用户名:密码（密码可为 bcrypt 哈希）
	BasicAuth bool     `json:"basicAuth"`
	WebAuth   bool     `json:"webAuth"`
	Users     []string `json:"users"`
	// OAuthUsers 为允许通过第三方登录访问的身份，格式 提供方key:用户名（或邮箱），提供方key:* 表示该提供方任意用户
	OAuthUsers []string `json:"oauthUsers"`

	IPMode string   `json:"ipMode"`
	IPList []string `json:"ipList"`

	// RequestHeaders / ResponseHeaders 每项为 "Key: Value"，Value 为空表示删除该头
	RequestHeaders  []string `json:"requestHeaders"`
	ResponseHeaders []string `json:"responseHeaders"`
	CORS            bool     `json:"cors"`
	AccessLog       bool     `json:"accessLog"`
	// WAF 为引用的 WAF 实例 key，留空不检查；实例不存在或停用时放行
	WAF    string `json:"waf"`
	Remark string `json:"remark"`
}

// Listener 是一个监听端口及其站点。
type Listener struct {
	Key    string `json:"key"`
	Name   string `json:"name"`
	Enable bool   `json:"enable"`
	// Network 为 tcp（双栈）、tcp4、tcp6
	Network  string `json:"network"`
	ListenIP string `json:"listenIp"`
	Port     int    `json:"port"`
	TLS      bool   `json:"tls"`
	// MinTLS 为 1.0 / 1.1 / 1.2 / 1.3
	MinTLS string `json:"minTls"`
	// AcceptProxyProtocol 接收上游（如 haproxy、frp）发送的 PROXY 协议头，仅信任全局设置中的来源
	AcceptProxyProtocol bool   `json:"acceptProxyProtocol"`
	Sites               []Site `json:"sites"`
	// Default 在没有站点匹配时使用；未启用时返回 404
	Default Site   `json:"default"`
	Remark  string `json:"remark"`
}

// Settings 是模块级配置。
type Settings struct {
	Enable bool `json:"enable"`
	// TrustedProxies 为可信的 PROXY 协议来源（IP 或 CIDR）
	TrustedProxies []string `json:"trustedProxies"`
	// AuthSecret 用于签名网页认证 Cookie，首次启动自动生成
	AuthSecret string `json:"authSecret"`
	// OAuthProviders 为自建的第三方登录配置
	OAuthProviders []OAuthProvider `json:"oauthProviders"`
	// OAuthCallbackBase 为接收第三方登录回调的站点地址（需由 Gately 的某个监听提供），例如 https://auth.example.com
	OAuthCallbackBase string `json:"oauthCallbackBase"`
}

type fileConfig struct {
	Settings  Settings   `json:"settings"`
	Listeners []Listener `json:"listeners"`
}

func cleanList(in []string) []string {
	var out []string
	for _, v := range in {
		if v = strings.TrimSpace(v); v != "" {
			out = append(out, v)
		}
	}
	return out
}

// parsePrefixes 解析 IP 或 CIDR 列表。
func parsePrefixes(list []string) ([]netip.Prefix, error) {
	var out []netip.Prefix
	for _, s := range list {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if strings.Contains(s, "/") {
			p, err := netip.ParsePrefix(s)
			if err != nil {
				return nil, fmt.Errorf("无效的网段 %q", s)
			}
			out = append(out, p.Masked())
			continue
		}
		a, err := netip.ParseAddr(s)
		if err != nil {
			return nil, fmt.Errorf("无效的 IP %q", s)
		}
		out = append(out, netip.PrefixFrom(a, a.BitLen()))
	}
	return out, nil
}

// Validate 校验监听器及其站点。
func (l *Listener) Validate() error {
	l.Name = strings.TrimSpace(l.Name)
	switch l.Network {
	case "":
		l.Network = "tcp"
	case "tcp", "tcp4", "tcp6":
	default:
		return fmt.Errorf("不支持的网络类型 %q", l.Network)
	}
	l.ListenIP = strings.Trim(strings.TrimSpace(l.ListenIP), "[]")
	if l.ListenIP != "" && net.ParseIP(l.ListenIP) == nil {
		return fmt.Errorf("监听地址 %q 无效", l.ListenIP)
	}
	if l.Port < 1 || l.Port > 65535 {
		return errors.New("监听端口无效")
	}
	switch l.MinTLS {
	case "", "1.0", "1.1", "1.2", "1.3":
	default:
		return fmt.Errorf("不支持的 TLS 版本 %q", l.MinTLS)
	}
	keys := map[string]bool{}
	for i := range l.Sites {
		s := &l.Sites[i]
		if err := s.validate(false); err != nil {
			label := s.Name
			if label == "" && len(s.Domains) > 0 {
				label = s.Domains[0]
			}
			return fmt.Errorf("站点 %s: %w", label, err)
		}
		if s.Key == "" || keys[s.Key] {
			s.Key = fmt.Sprintf("s%d", i+1)
			for keys[s.Key] {
				s.Key += "_"
			}
		}
		keys[s.Key] = true
	}
	if l.Default.Enable {
		if err := l.Default.validate(true); err != nil {
			return fmt.Errorf("默认站点: %w", err)
		}
	}
	l.Default.Key = "default"
	return nil
}

func (s *Site) validate(isDefault bool) error {
	s.Name = strings.TrimSpace(s.Name)
	var domains []string
	for _, d := range s.Domains {
		d = strings.ToLower(strings.Trim(strings.TrimSpace(d), "."))
		if d == "" {
			continue
		}
		if d != "*" && (strings.Count(d, "*") > 1 || (strings.Contains(d, "*") && !strings.HasPrefix(d, "*."))) {
			return fmt.Errorf("域名 %q 格式错误", d)
		}
		domains = append(domains, d)
	}
	s.Domains = domains
	if !isDefault && len(s.Domains) == 0 {
		return errors.New("至少填写一个域名")
	}
	switch s.Type {
	case "", TypeProxy:
		s.Type = TypeProxy
		s.Targets = cleanList(s.Targets)
		if len(s.Targets) == 0 {
			return errors.New("请填写后端地址")
		}
		for _, t := range s.Targets {
			u, err := url.Parse(t)
			if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
				return fmt.Errorf("后端地址 %q 需为 http(s)://主机:端口", t)
			}
		}
	case TypeRedirect:
		u, err := url.Parse(strings.TrimSpace(s.RedirectTo))
		if err != nil || u.Scheme == "" || u.Host == "" {
			return errors.New("重定向地址需为完整 URL")
		}
		switch s.RedirectCode {
		case 0:
			s.RedirectCode = 302
		case 301, 302, 307, 308:
		default:
			return errors.New("重定向状态码只能是 301/302/307/308")
		}
	case TypeStatic:
		if !filepath.IsAbs(s.Root) {
			return errors.New("静态文件目录需为绝对路径")
		}
	case TypeText:
		if s.StatusCode == 0 {
			s.StatusCode = 200
		}
		if s.StatusCode < 100 || s.StatusCode > 599 {
			return errors.New("状态码无效")
		}
		if s.ContentType == "" {
			s.ContentType = "text/plain; charset=utf-8"
		}
	default:
		return fmt.Errorf("未知的站点类型 %q", s.Type)
	}
	s.Users = cleanList(s.Users)
	s.OAuthUsers = cleanList(s.OAuthUsers)
	if s.BasicAuth && len(s.Users) == 0 {
		return errors.New("启用基本认证时至少添加一个用户")
	}
	if s.WebAuth && len(s.Users) == 0 && len(s.OAuthUsers) == 0 {
		return errors.New("启用网页认证时至少添加一个用户或第三方登录身份")
	}
	for _, u := range s.OAuthUsers {
		if p, id, ok := strings.Cut(u, ":"); !ok || p == "" || id == "" {
			return fmt.Errorf("第三方身份格式应为 提供方:用户名，实际 %q", u)
		}
	}
	for _, u := range s.Users {
		if name, pass, ok := strings.Cut(u, ":"); !ok || name == "" || pass == "" {
			return fmt.Errorf("用户格式应为 用户名:密码，实际 %q", u)
		}
	}
	switch s.IPMode {
	case IPModeOff, IPModeWhitelist, IPModeBlacklist:
	default:
		return fmt.Errorf("未知的访问控制模式 %q", s.IPMode)
	}
	if _, err := parsePrefixes(s.IPList); err != nil {
		return err
	}
	for _, h := range append(append([]string{}, s.RequestHeaders...), s.ResponseHeaders...) {
		if k, _, ok := strings.Cut(h, ":"); !ok || strings.TrimSpace(k) == "" {
			return fmt.Errorf("请求头格式应为 Key: Value，实际 %q", h)
		}
	}
	return nil
}

// matchDomain 判断 host 是否匹配规则。返回匹配优先级：精确 3、通配 2、全部 1，不匹配 0。
func matchDomain(rule, host string) int {
	switch {
	case rule == host:
		return 3
	case rule == "*":
		return 1
	case strings.HasPrefix(rule, "*.") && strings.HasSuffix(host, rule[1:]):
		return 2
	}
	return 0
}
