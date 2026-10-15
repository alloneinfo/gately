package stun

import (
	"encoding/binary"
	"net"
	"net/netip"
	"sync"
)

// Server 是最小的 STUN Binding 服务端，同时监听 TCP 与 UDP。
type Server struct {
	TCP     net.Listener
	UDP     net.PacketConn
	mu      sync.Mutex
	mapAddr func(observed netip.AddrPort) netip.AddrPort
	wg      sync.WaitGroup
}

// SetMapAddr 设置返回给客户端地址的改写函数，测试中用来模拟 NAT 映射变化。
func (s *Server) SetMapAddr(f func(observed netip.AddrPort) netip.AddrPort) {
	s.mu.Lock()
	s.mapAddr = f
	s.mu.Unlock()
}

// ListenServer 在 addr 上同时启动 TCP 与 UDP 服务。
func ListenServer(addr string) (*Server, error) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	pc, err := net.ListenPacket("udp", ln.Addr().String())
	if err != nil {
		ln.Close()
		return nil, err
	}
	s := &Server{TCP: ln, UDP: pc}
	s.wg.Add(2)
	go s.serveTCP()
	go s.serveUDP()
	return s, nil
}

// Addr 返回服务地址（TCP 与 UDP 端口相同）。
func (s *Server) Addr() string {
	return s.TCP.Addr().String()
}

// Close 关闭服务。
func (s *Server) Close() {
	s.TCP.Close()
	s.UDP.Close()
	s.wg.Wait()
}

func (s *Server) mapped(a net.Addr) netip.AddrPort {
	ap, _ := netip.ParseAddrPort(a.String())
	ap = netip.AddrPortFrom(ap.Addr().Unmap(), ap.Port())
	s.mu.Lock()
	f := s.mapAddr
	s.mu.Unlock()
	if f != nil {
		return f(ap)
	}
	return ap
}

func (s *Server) serveTCP() {
	defer s.wg.Done()
	for {
		c, err := s.TCP.Accept()
		if err != nil {
			return
		}
		go func() {
			defer c.Close()
			for {
				msg, err := ReadMessage(c)
				if err != nil {
					return
				}
				if binary.BigEndian.Uint16(msg[0:]) != typeBindingRequest {
					return
				}
				var id TxID
				copy(id[:], msg[8:20])
				if _, err := c.Write(EncodeBindingResponse(id, s.mapped(c.RemoteAddr()))); err != nil {
					return
				}
			}
		}()
	}
}

func (s *Server) serveUDP() {
	defer s.wg.Done()
	buf := make([]byte, maxMessageSize)
	for {
		n, addr, err := s.UDP.ReadFrom(buf)
		if err != nil {
			return
		}
		msg := buf[:n]
		if !IsMessage(msg) || binary.BigEndian.Uint16(msg[0:]) != typeBindingRequest {
			continue
		}
		var id TxID
		copy(id[:], msg[8:20])
		_, _ = s.UDP.WriteTo(EncodeBindingResponse(id, s.mapped(addr)), addr)
	}
}
