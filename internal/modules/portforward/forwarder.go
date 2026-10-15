package portforward

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/wangwanjie/gately/internal/logbuf"
	"github.com/wangwanjie/gately/internal/netpipe"
)

// Stats 是规则的运行统计，所有字段使用原子操作。
type Stats struct {
	ActiveConns atomic.Int64
	TotalConns  atomic.Int64
	BytesIn     atomic.Int64 // 客户端 -> 目标
	BytesOut    atomic.Int64 // 目标 -> 客户端
}

// StatsSnapshot 是统计的快照，用于 API 输出。
type StatsSnapshot struct {
	ActiveConns int64 `json:"activeConns"`
	TotalConns  int64 `json:"totalConns"`
	BytesIn     int64 `json:"bytesIn"`
	BytesOut    int64 `json:"bytesOut"`
}

func (s *Stats) snapshot() StatsSnapshot {
	return StatsSnapshot{
		ActiveConns: s.ActiveConns.Load(),
		TotalConns:  s.TotalConns.Load(),
		BytesIn:     s.BytesIn.Load(),
		BytesOut:    s.BytesOut.Load(),
	}
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

// runner 负责一条规则的全部监听器。
type runner struct {
	rule     Rule
	settings Settings
	log      *logbuf.Logger
	stats    *Stats
	picker   *targetPicker

	cancel  context.CancelFunc
	wg      sync.WaitGroup
	mu      sync.Mutex
	closers []io.Closer
	// listenErrors 记录启动失败的监听项，供界面展示
	listenErrors []string
}

func newRunner(rule Rule, settings Settings, log *logbuf.Logger) *runner {
	return &runner{
		rule:     rule,
		settings: settings,
		log:      log,
		stats:    &Stats{},
		picker:   &targetPicker{hosts: rule.TargetAddresses},
	}
}

// start 为每个 协议×端口 建立监听。部分端口失败时其余端口照常工作。
func (r *runner) start(parent context.Context) error {
	pairs, err := MapPorts(r.rule.ListenPorts, r.rule.TargetPorts)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(parent)
	r.cancel = cancel

	maxConns := r.rule.MaxConnections
	if maxConns <= 0 {
		maxConns = r.settings.DefaultMaxConnections
	}
	udpTimeout := time.Duration(r.rule.UDPSessionTimeout) * time.Second
	if udpTimeout <= 0 {
		udpTimeout = time.Duration(r.settings.DefaultUDPSessionTimeout) * time.Second
	}

	okCount := 0
	for _, proto := range r.rule.Protocols {
		for _, pair := range pairs {
			addr := net.JoinHostPort(r.rule.ListenAddress, strconv.Itoa(pair.Listen))
			var startErr error
			if strings.HasPrefix(proto, "tcp") {
				startErr = r.startTCP(ctx, proto, addr, pair.Target, maxConns)
			} else {
				startErr = r.startUDP(ctx, proto, addr, pair.Target, udpTimeout)
			}
			if startErr != nil {
				msg := fmt.Sprintf("%s %s 监听失败: %v", proto, addr, startErr)
				r.listenErrors = append(r.listenErrors, msg)
				r.log.Errorf("%s", msg)
				continue
			}
			okCount++
			r.log.Infof("%s %s -> %s:%d 已启动", proto, addr, strings.Join(r.rule.TargetAddresses, "|"), pair.Target)
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

func (r *runner) dialTarget(network string, port int) (net.Conn, error) {
	timeout := time.Duration(r.settings.DialTimeout) * time.Second
	var lastErr error
	for _, host := range r.picker.order() {
		conn, err := net.DialTimeout(network, net.JoinHostPort(host, strconv.Itoa(port)), timeout)
		if err == nil {
			return conn, nil
		}
		lastErr = err
	}
	return nil, lastErr
}

func (r *runner) startTCP(ctx context.Context, network, addr string, targetPort, maxConns int) error {
	var lc net.ListenConfig
	ln, err := lc.Listen(ctx, network, addr)
	if err != nil {
		return err
	}
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
				r.log.Warnf("accept 失败: %v", err)
				// 避免 fd 耗尽等错误下空转
				time.Sleep(100 * time.Millisecond)
				continue
			}
			select {
			case sem <- struct{}{}:
			default:
				r.log.Warnf("%s 超过最大连接数 %d，拒绝 %s", addr, maxConns, conn.RemoteAddr())
				conn.Close()
				continue
			}
			r.wg.Add(1)
			go func() {
				defer r.wg.Done()
				defer func() { <-sem }()
				r.handleTCP(ctx, conn, targetPort)
			}()
		}
	}()
	return nil
}

func (r *runner) handleTCP(ctx context.Context, client net.Conn, targetPort int) {
	defer client.Close()
	r.stats.ActiveConns.Add(1)
	r.stats.TotalConns.Add(1)
	defer r.stats.ActiveConns.Add(-1)

	target, err := r.dialTarget("tcp", targetPort)
	if err != nil {
		r.log.Warnf("%s 连接目标失败: %v", client.RemoteAddr(), err)
		return
	}
	defer target.Close()
	r.log.Debugf("%s -> %s 已连接", client.RemoteAddr(), target.RemoteAddr())

	in, out := netpipe.Join(ctx, client, target)
	r.stats.BytesIn.Add(in)
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
				r.log.Warnf("UDP 读取失败: %v", err)
				continue
			}
			key := clientAddr.String()
			mu.Lock()
			sess := sessions[key]
			if sess == nil {
				if len(sessions) >= r.settings.MaxUDPSessions {
					mu.Unlock()
					r.log.Warnf("%s UDP 会话数达到上限 %d，丢弃 %s 的数据包", addr, r.settings.MaxUDPSessions, key)
					continue
				}
				conn, err := r.dialTarget("udp", targetPort)
				if err != nil {
					mu.Unlock()
					r.log.Warnf("%s 建立 UDP 会话失败: %v", key, err)
					continue
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
