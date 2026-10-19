package httpx

import (
	"net"
	"net/http"
	"net/netip"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// 管理端口常放在本机反向代理（nginx、haproxy、Gately Web 服务）之后。此时 TCP 对端是代理自己，
// 真实客户端地址只能从代理写入的请求头中取得。只有对端是可信代理时才读取这些请求头，
// 否则任何人都能伪造来源地址，绕过"仅内网访问"与登录限流。

// proxyTrust 是当前生效的代理信任配置。
type proxyTrust struct {
	enable bool
	extra  []netip.Prefix
}

var trust atomic.Pointer[proxyTrust]

func init() {
	trust.Store(&proxyTrust{enable: true})
}

// SetProxyTrust 设置是否信任本机代理传来的客户端地址，extra 为额外信任的代理地址段。
func SetProxyTrust(enable bool, extra []netip.Prefix) {
	trust.Store(&proxyTrust{enable: enable, extra: extra})
}

// ParsePrefixes 把 IP 或 CIDR 列表解析为地址段，单个 IP 视为 /32 或 /128。
func ParsePrefixes(list []string) ([]netip.Prefix, error) {
	var out []netip.Prefix
	for _, s := range list {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if p, err := netip.ParsePrefix(s); err == nil {
			out = append(out, p.Masked())
			continue
		}
		a, err := netip.ParseAddr(strings.Trim(s, "[]"))
		if err != nil {
			return nil, err
		}
		a = a.Unmap()
		out = append(out, netip.PrefixFrom(a, a.BitLen()))
	}
	return out, nil
}

// PeerIP 返回 TCP 对端地址，不读取任何请求头。
func PeerIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return strings.Trim(host, "[]")
}

// ClientIP 返回请求的客户端地址。对端为可信代理时，依次从 X-Real-IP、X-Forwarded-For（自右向左跳过可信代理）中取得真实地址；
// 否则返回对端地址。
func ClientIP(r *http.Request) string {
	peer := PeerIP(r)
	t := trust.Load()
	if !t.enable || !trustedProxy(peer, t) {
		return peer
	}
	if v := strings.TrimSpace(r.Header.Get("X-Real-IP")); v != "" {
		if a, err := netip.ParseAddr(v); err == nil {
			return a.Unmap().String()
		}
	}
	// X-Forwarded-For 最右侧由最近的代理追加；继续向左只跳过同样可信的代理，第一个不可信的地址即客户端
	var hops []string
	for _, h := range r.Header.Values("X-Forwarded-For") {
		for _, part := range strings.Split(h, ",") {
			if part = strings.TrimSpace(part); part != "" {
				hops = append(hops, part)
			}
		}
	}
	for i := len(hops) - 1; i >= 0; i-- {
		a, err := netip.ParseAddr(strings.Trim(hops[i], "[]"))
		if err != nil {
			// 格式异常时不再向左信任，保守地返回对端地址
			return peer
		}
		ip := a.Unmap().String()
		if !trustedProxy(ip, t) {
			return ip
		}
	}
	return peer
}

// trustedProxy 判断地址是否为可信代理：回环地址、本机网卡地址或额外配置的地址段。
func trustedProxy(ip string, t *proxyTrust) bool {
	a, err := netip.ParseAddr(ip)
	if err != nil {
		return false
	}
	a = a.Unmap()
	if a.IsLoopback() {
		return true
	}
	for _, p := range t.extra {
		if p.Contains(a) {
			return true
		}
	}
	return localAddrs().has(a)
}

// 本机地址缓存：网卡地址会变化（DHCP、IPv6 前缀轮换），定期刷新。
type addrSet struct {
	at    time.Time
	addrs map[netip.Addr]bool
}

func (s *addrSet) has(a netip.Addr) bool {
	return s != nil && s.addrs[a]
}

var (
	localMu    sync.Mutex
	localCache *addrSet
	// interfaceAddrs 允许测试替换本机地址来源
	interfaceAddrs = net.InterfaceAddrs
)

const localAddrTTL = 30 * time.Second

func localAddrs() *addrSet {
	localMu.Lock()
	defer localMu.Unlock()
	if localCache != nil && time.Since(localCache.at) < localAddrTTL {
		return localCache
	}
	set := &addrSet{at: time.Now(), addrs: map[netip.Addr]bool{}}
	if addrs, err := interfaceAddrs(); err == nil {
		for _, a := range addrs {
			if ipn, ok := a.(*net.IPNet); ok {
				if ip, ok := netip.AddrFromSlice(ipn.IP); ok {
					set.addrs[ip.Unmap()] = true
				}
			}
		}
	}
	localCache = set
	return set
}
