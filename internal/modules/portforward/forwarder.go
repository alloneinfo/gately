package portforward

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	proxyproto "github.com/pires/go-proxyproto"

	"github.com/wangwanjie/gately/internal/banlist"
	"github.com/wangwanjie/gately/internal/logbuf"
	"github.com/wangwanjie/gately/internal/netpipe"
)

// Stats 是规则的运行统计，所有字段使用原子操作。
type Stats struct {
	ActiveConns atomic.Int64
	TotalConns  atomic.Int64
	BytesIn     atomic.Int64 // 客户端 -> 目标
	BytesOut    atomic.Int64 // 目标 -> 客户端
	// SNIRejected 是 SNI 未命中且没有默认目标而被断开的连接数
	SNIRejected atomic.Int64
}

// StatsSnapshot 是统计的快照，用于 API 输出。
type StatsSnapshot struct {
	ActiveConns int64 `json:"activeConns"`
	TotalConns  int64 `json:"totalConns"`
	BytesIn     int64 `json:"bytesIn"`
	BytesOut    int64 `json:"bytesOut"`
	// 以下仅 SNI 分流规则输出：RouteConns 与规则的 sniRoutes 一一对应，为各路由累计连接数
	RouteConns   []int64 `json:"routeConns,omitempty"`
	DefaultConns int64   `json:"defaultConns,omitempty"`
	SNIRejected  int64   `json:"sniRejected,omitempty"`
}

func (s *Stats) snapshot() StatsSnapshot {
	return StatsSnapshot{
		ActiveConns: s.ActiveConns.Load(),
		TotalConns:  s.TotalConns.Load(),
		BytesIn:     s.BytesIn.Load(),
		BytesOut:    s.BytesOut.Load(),
		SNIRejected: s.SNIRejected.Load(),
	}
}

// snapshot 输出规则统计，SNI 规则附带各路由的命中数。
func (r *runner) snapshot() StatsSnapshot {
	snap := r.stats.snapshot()
	if sni := r.current().sni; sni != nil {
		snap.RouteConns = make([]int64, len(sni.targets))
		for i, t := range sni.targets {
			snap.RouteConns[i] = t.conns.Load()
		}
		if sni.fallback != nil {
			snap.DefaultConns = sni.fallback.conns.Load()
		}
	}
	return snap
}

// targetPicker 在多个目标地址间轮询，连接失败时依次尝试下一个。
type targetPicker struct {
	hosts []string
	next  atomic.Uint64
}

// order 返回本次尝试的目标顺序：从轮询位置开始的完整一圈。
func (p *targetPicker) order() []string {
	n := len(p.hosts)
	start := int(p.next.Add(1)-1) % n
	out := make([]string, n)
	for i := 0; i < n; i++ {
		out[i] = p.hosts[(start+i)%n]
	}
	return out
}

// route 是规则运行时可热替换的部分：目标地址、SNI 路由表、PROXY 协议、日志名与全局参数。
// 每个新连接开始时读取一次当前 route，替换后新连接立即使用新配置，已建立的连接不受影响。
type route struct {
	rule     Rule
	settings Settings
	log      *logbuf.Logger
	picker   *targetPicker
	// sni 为 SNI 分流路由表，仅 SNI 模式的规则存在
	sni *sniRouter
}

func newRoute(rule Rule, settings Settings, log *logbuf.Logger) *route {
	rt := &route{
		rule:     rule,
		settings: settings,
		log:      log,
		picker:   &targetPicker{hosts: rule.TargetAddresses},
	}
	if rule.Mode == ModeSNI {
		rt.sni = newSNIRouter(rule)
	}
	return rt
}

// runner 负责一条规则的全部监听器。
type runner struct {
	// rule、settings 为启动监听时的配置，决定监听的协议、端口映射与连接上限；运行中只读
	rule     Rule
	settings Settings
	stats    *Stats
	routes   atomic.Pointer[route]

	cancel  context.CancelFunc
	wg      sync.WaitGroup
	mu      sync.Mutex
	closers []io.Closer
	// listenErrors 记录启动失败的监听项，供界面展示
	listenErrors []string
}

func newRunner(rule Rule, settings Settings, log *logbuf.Logger) *runner {
	r := &runner{rule: rule, settings: settings, stats: &Stats{}}
	r.routes.Store(newRoute(rule, settings, log))
	return r
}

// current 返回当前生效的 route。
func (r *runner) current() *route {
	return r.routes.Load()
}

// effectiveLimits 返回规则实际使用的每端口最大连接数与 UDP 会话超时（规则未设置时取全局默认值）。
func effectiveLimits(rule Rule, settings Settings) (maxConns int, udpTimeout time.Duration) {
	maxConns = rule.MaxConnections
	if maxConns <= 0 {
		maxConns = settings.DefaultMaxConnections
	}
	udpTimeout = time.Duration(rule.UDPSessionTimeout) * time.Second
	if udpTimeout <= 0 {
		udpTimeout = time.Duration(settings.DefaultUDPSessionTimeout) * time.Second
	}
	return maxConns, udpTimeout
}

// canUpdate 判断新配置能否在不重新监听的情况下生效：监听协议、地址、端口映射、转发模式与连接上限都不变，
// 且当前所有端口都在正常监听（有端口监听失败时需要重启以便重试）。
func (r *runner) canUpdate(rule Rule, settings Settings) bool {
	if len(r.listenErrors) > 0 {
		return false
	}
	old := r.rule
	if !slices.Equal(old.Protocols, rule.Protocols) || old.ListenAddress != rule.ListenAddress ||
		old.ListenPorts != rule.ListenPorts || old.TargetPorts != rule.TargetPorts || old.Mode != rule.Mode {
		return false
	}
	oldMax, oldUDP := effectiveLimits(old, r.settings)
	newMax, newUDP := effectiveLimits(rule, settings)
	return oldMax == newMax && oldUDP == newUDP
}

// update 热替换目标地址、SNI 路由表等配置，监听器与已建立的连接保持不断。调用前需用 canUpdate 确认。
func (r *runner) update(rule Rule, settings Settings, log *logbuf.Logger) {
	r.routes.Store(newRoute(rule, settings, log))
	log.Infof("配置已更新，监听未中断")
}

// start 为每个 协议×端口 建立监听。部分端口失败时其余端口照常工作。
func (r *runner) start(parent context.Context) error {
	pairs, err := MapPorts(r.rule.ListenPorts, r.rule.TargetPorts)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(parent)
	r.cancel = cancel
	log := r.current().log

	maxConns, udpTimeout := effectiveLimits(r.rule, r.settings)

	okCount := 0
	for _, proto := range r.rule.Protocols {
		for _, pair := range pairs {
			addr := net.JoinHostPort(r.rule.ListenAddress, strconv.Itoa(pair.Listen))
			var startErr error
			if strings.HasPrefix(proto, "tcp") {
				startErr = r.startTCP(ctx, proto, addr, pair, maxConns)
			} else {
				startErr = r.startUDP(ctx, proto, addr, pair.Target, udpTimeout)
			}
			if startErr != nil {
				msg := fmt.Sprintf("%s %s 监听失败: %v", proto, addr, startErr)
				r.listenErrors = append(r.listenErrors, msg)
				log.Errorf("%s", msg)
				continue
			}
			okCount++
			if r.rule.Mode == ModeSNI {
				log.Infof("%s %s 已启动（SNI 分流，%d 条路由）", proto, addr, len(r.rule.SNIRoutes))
			} else {
				log.Infof("%s %s -> %s:%d 已启动", proto, addr, strings.Join(r.rule.TargetAddresses, "|"), pair.Target)
			}
		}
	}
	if okCount == 0 {
		r.stop()
		return errors.New("所有端口监听失败")
	}
	return nil
}

func (r *runner) track(c io.Closer) {
	r.mu.Lock()
	r.closers = append(r.closers, c)
	r.mu.Unlock()
}

// stop 关闭全部监听器与连接，并等待协程退出。
func (r *runner) stop() {
	if r.cancel != nil {
		r.cancel()
	}
	r.mu.Lock()
	for _, c := range r.closers {
		_ = c.Close()
	}
	r.closers = nil
	r.mu.Unlock()
	r.wg.Wait()
}

func (rt *route) dialTarget(network string, port int) (net.Conn, error) {
	timeout := time.Duration(rt.settings.DialTimeout) * time.Second
	var lastErr error
	for _, host := range rt.picker.order() {
		conn, err := net.DialTimeout(network, net.JoinHostPort(host, strconv.Itoa(port)), timeout)
		if err == nil {
			return conn, nil
		}
		lastErr = err
	}
	return nil, lastErr
}

func (r *runner) startTCP(ctx context.Context, network, addr string, pair PortPair, maxConns int) error {
	var lc net.ListenConfig
	ln, err := lc.Listen(ctx, network, addr)
	if err != nil {
		return err
	}
	// 被封禁地址的连接在 Accept 时直接关闭
	ln = banlist.Default.Listener(ln)
	r.track(ln)
	sem := make(chan struct{}, maxConns)
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		for {
			conn, err := ln.Accept()
			if err != nil {
				if ctx.Err() != nil || errors.Is(err, net.ErrClosed) {
					return
				}
				r.current().log.Warnf("accept 失败: %v", err)
				// 避免 fd 耗尽等错误下空转
				time.Sleep(100 * time.Millisecond)
				continue
			}
			select {
			case sem <- struct{}{}:
			default:
				r.current().log.Warnf("%s 超过最大连接数 %d，拒绝 %s", addr, maxConns, conn.RemoteAddr())
				conn.Close()
				continue
			}
			r.wg.Add(1)
			go func() {
				defer r.wg.Done()
				defer func() { <-sem }()
				// 每个连接固定使用建立时的配置，热更新只影响之后的新连接
				rt := r.current()
				if rt.sni != nil {
					r.handleSNI(ctx, rt, conn, pair)
				} else {
					r.handleTCP(ctx, rt, conn, pair.Target)
				}
			}()
		}
	}()
	return nil
}

func (r *runner) handleTCP(ctx context.Context, rt *route, client net.Conn, targetPort int) {
	defer client.Close()
	r.stats.ActiveConns.Add(1)
	r.stats.TotalConns.Add(1)
	defer r.stats.ActiveConns.Add(-1)

	target, err := rt.dialTarget("tcp", targetPort)
	if err != nil {
		rt.log.Warnf("%s 连接目标失败: %v", client.RemoteAddr(), err)
		return
	}
	defer target.Close()
	if err := writePrefix(target, client, rt.rule.ProxyProtocol, nil); err != nil {
		rt.log.Warnf("%s 发送 PROXY 协议头失败: %v", client.RemoteAddr(), err)
		return
	}
	rt.log.Debugf("%s -> %s 已连接", client.RemoteAddr(), target.RemoteAddr())

	in, out := netpipe.Join(ctx, client, target)
	r.stats.BytesIn.Add(in)
	r.stats.BytesOut.Add(out)
}

// writePrefix 在双向转发开始前写入 PROXY 协议头（version 非空时）与已预读的客户端数据，合并为一次写入以减少小包。
func writePrefix(target, client net.Conn, version string, peeked []byte) error {
	var buf []byte
	if version != "" {
		// 源地址为真实客户端，目的地址为本规则的监听地址
		v := byte(2)
		if version == "v1" {
			v = 1
		}
		header, err := proxyproto.HeaderProxyFromAddrs(v, client.RemoteAddr(), client.LocalAddr()).Format()
		if err != nil {
			return err
		}
		buf = header
	}
	buf = append(buf, peeked...)
	if len(buf) == 0 {
		return nil
	}
	_, err := target.Write(buf)
	return err
}

// handleSNI 预读 ClientHello，按 SNI 选择目标后透传整个连接（TLS 不终止）。
// 与 haproxy 的 inspect-delay + req_ssl_sni 行为一致：非 TLS、无 SNI、解析失败或超时未收到 ClientHello 时交给默认目标。
func (r *runner) handleSNI(ctx context.Context, rt *route, client net.Conn, pair PortPair) {
	defer client.Close()
	r.stats.ActiveConns.Add(1)
	r.stats.TotalConns.Add(1)
	defer r.stats.ActiveConns.Add(-1)

	_ = client.SetReadDeadline(time.Now().Add(sniPeekTimeout))
	peeked, hello, err := peekClientHello(client)
	_ = client.SetReadDeadline(time.Time{})
	var ne net.Error
	switch {
	case err == nil:
	case errors.As(err, &ne) && ne.Timeout():
		// 客户端迟迟未发送 ClientHello（例如服务端先发言的协议），交给默认目标
	case errors.As(err, &ne), errors.Is(err, io.EOF), errors.Is(err, io.ErrUnexpectedEOF):
		// 客户端在发完 ClientHello 前断开
		return
	default:
		rt.log.Debugf("%s 未解析到 SNI: %v", client.RemoteAddr(), err)
	}

	target := rt.sni.match(hello.ServerName)
	if target == nil {
		r.stats.SNIRejected.Add(1)
		rt.log.Debugf("%s SNI %q 未命中路由且未设置默认目标，已断开", client.RemoteAddr(), hello.ServerName)
		return
	}
	target.conns.Add(1)
	port := target.port
	if target == rt.sni.fallback {
		port = pair.Target
	} else if port == 0 {
		port = pair.Listen
	}
	conn, err := dialSNITarget(target, port, time.Duration(rt.settings.DialTimeout)*time.Second)
	if err != nil {
		rt.log.Warnf("%s SNI %q 连接目标失败: %v", client.RemoteAddr(), hello.ServerName, err)
		return
	}
	defer conn.Close()
	if err := writePrefix(conn, client, target.proxyProtocol, peeked); err != nil {
		rt.log.Warnf("%s 写入目标失败: %v", client.RemoteAddr(), err)
		return
	}
	rt.log.Debugf("%s SNI %q ALPN %v -> %s", client.RemoteAddr(), hello.ServerName, hello.ALPN, conn.RemoteAddr())

	// 预读的数据已直接写给目标，这里传入原始连接，Linux 上 io.Copy 可走 splice 零拷贝
	in, out := netpipe.Join(ctx, client, conn)
	r.stats.BytesIn.Add(in + int64(len(peeked)))
	r.stats.BytesOut.Add(out)
}

// udpSession 是一个客户端地址对应的目标连接。
type udpSession struct {
	conn       net.Conn
	lastActive atomic.Int64 // UnixNano
}

func (r *runner) startUDP(ctx context.Context, network, addr string, targetPort int, idle time.Duration) error {
	var lc net.ListenConfig
	pc, err := lc.ListenPacket(ctx, network, addr)
	if err != nil {
		return err
	}
	r.track(pc)

	var (
		mu       sync.Mutex
		sessions = map[string]*udpSession{}
	)
	closeAll := func() {
		mu.Lock()
		for k, s := range sessions {
			s.conn.Close()
			delete(sessions, k)
		}
		mu.Unlock()
	}

	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		defer closeAll()
		buf := make([]byte, 64*1024)
		for {
			n, clientAddr, err := pc.ReadFrom(buf)
			if err != nil {
				if ctx.Err() != nil || errors.Is(err, net.ErrClosed) {
					return
				}
				r.current().log.Warnf("UDP 读取失败: %v", err)
				continue
			}
			// 丢弃被封禁地址的数据包（已有会话随空闲超时自然结束）
			if banlist.Default.BannedAddr(clientAddr) {
				continue
			}
			key := clientAddr.String()
			// 新会话使用当前配置的目标与上限，已有会话保持原目标
			rt := r.current()
			mu.Lock()
			sess := sessions[key]
			if sess == nil {
				if len(sessions) >= rt.settings.MaxUDPSessions {
					mu.Unlock()
					rt.log.Warnf("%s UDP 会话数达到上限 %d，丢弃 %s 的数据包", addr, rt.settings.MaxUDPSessions, key)
					continue
				}
				// 目标为域名时拨号需要解析，放到锁外，避免阻塞会话清理
				mu.Unlock()
				conn, err := rt.dialTarget("udp", targetPort)
				if err != nil {
					rt.log.Warnf("%s 建立 UDP 会话失败: %v", key, err)
					continue
				}
				mu.Lock()
				if ctx.Err() != nil {
					mu.Unlock()
					conn.Close()
					return
				}
				sess = &udpSession{conn: conn}
				sess.lastActive.Store(time.Now().UnixNano())
				sessions[key] = sess
				r.stats.ActiveConns.Add(1)
				r.stats.TotalConns.Add(1)
				r.wg.Add(1)
				go r.pumpUDP(pc, clientAddr, sess, idle, func() {
					mu.Lock()
					if sessions[key] == sess {
						delete(sessions, key)
					}
					mu.Unlock()
				})
			}
			mu.Unlock()

			sess.lastActive.Store(time.Now().UnixNano())
			if _, err := sess.conn.Write(buf[:n]); err == nil {
				r.stats.BytesIn.Add(int64(n))
			}
		}
	}()
	return nil
}

// pumpUDP 把目标的回包发回客户端，空闲超时后关闭会话。
func (r *runner) pumpUDP(pc net.PacketConn, client net.Addr, sess *udpSession, idle time.Duration, remove func()) {
	defer r.wg.Done()
	defer r.stats.ActiveConns.Add(-1)
	defer remove()
	defer sess.conn.Close()
	buf := make([]byte, 64*1024)
	for {
		_ = sess.conn.SetReadDeadline(time.Now().Add(idle))
		n, err := sess.conn.Read(buf)
		if err != nil {
			var ne net.Error
			if errors.As(err, &ne) && ne.Timeout() {
				// 读超时期间客户端仍有发送则继续等待
				last := time.Unix(0, sess.lastActive.Load())
				if time.Since(last) < idle {
					continue
				}
			}
			return
		}
		sess.lastActive.Store(time.Now().UnixNano())
		if _, err := pc.WriteTo(buf[:n], client); err != nil {
			return
		}
		r.stats.BytesOut.Add(int64(n))
	}
}
