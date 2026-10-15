// Package netpipe 提供两个连接之间的双向转发。
package netpipe

import (
	"context"
	"io"
	"net"
)

// Join 在 a 与 b 之间双向复制数据，直到两个方向都结束或 ctx 取消。
// 返回 a→b 与 b→a 的字节数。调用方负责在返回后关闭连接。
func Join(ctx context.Context, a, b net.Conn) (aToB, bToA int64) {
	// ctx 取消时主动关闭两端，使阻塞的读写返回
	stop := context.AfterFunc(ctx, func() {
		a.Close()
		b.Close()
	})
	defer stop()

	done := make(chan struct{}, 2)
	go func() {
		aToB, _ = io.Copy(b, a)
		closeWrite(b)
		done <- struct{}{}
	}()
	go func() {
		bToA, _ = io.Copy(a, b)
		closeWrite(a)
		done <- struct{}{}
	}()
	<-done
	<-done
	return aToB, bToA
}

// closeWrite 半关闭写方向，让对端读到 EOF，同时保留另一方向的传输。
func closeWrite(c net.Conn) {
	if cw, ok := c.(interface{ CloseWrite() error }); ok {
		_ = cw.CloseWrite()
		return
	}
	_ = c.Close()
}
