// Package reuseport 提供可复用端口的监听与拨号，使同一本地端口既能监听入站连接，
// 又能主动连接 STUN 服务器与保活服务器（STUN 穿透的关键）。
package reuseport

import (
	"context"
	"net"
	"time"
)

// ListenConfig 返回设置了端口复用选项的 ListenConfig。
func ListenConfig() net.ListenConfig {
	return net.ListenConfig{Control: control}
}

// Listen 以端口复用方式监听 TCP。
func Listen(ctx context.Context, network, addr string) (net.Listener, error) {
	lc := ListenConfig()
	return lc.Listen(ctx, network, addr)
}

// ListenPacket 以端口复用方式监听 UDP。
func ListenPacket(ctx context.Context, network, addr string) (net.PacketConn, error) {
	lc := ListenConfig()
	return lc.ListenPacket(ctx, network, addr)
}

// Dial 从指定本地地址发起连接，本地端口可与已有监听共享。
func Dial(ctx context.Context, network string, local net.Addr, remote string, timeout time.Duration) (net.Conn, error) {
	d := net.Dialer{LocalAddr: local, Timeout: timeout, Control: control}
	return d.DialContext(ctx, network, remote)
}
