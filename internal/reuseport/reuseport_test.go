package reuseport

import (
	"context"
	"io"
	"net"
	"testing"
	"time"
)

// 同一本地端口：一边监听，一边主动连接远端
func TestListenAndDialSamePort(t *testing.T) {
	remote, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer remote.Close()
	go func() {
		c, err := remote.Accept()
		if err == nil {
			_, _ = io.Copy(c, c)
			c.Close()
		}
	}()

	ln, err := Listen(context.Background(), "tcp4", "0.0.0.0:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	port := ln.Addr().(*net.TCPAddr).Port

	c, err := Dial(context.Background(), "tcp4", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: port}, remote.Addr().String(), 3*time.Second)
	if err != nil {
		t.Fatalf("复用端口拨号失败: %v", err)
	}
	defer c.Close()
	if c.LocalAddr().(*net.TCPAddr).Port != port {
		t.Fatalf("本地端口 %d，期望 %d", c.LocalAddr().(*net.TCPAddr).Port, port)
	}
	_, _ = c.Write([]byte("hi"))
	buf := make([]byte, 2)
	_ = c.SetReadDeadline(time.Now().Add(3 * time.Second))
	if _, err := io.ReadFull(c, buf); err != nil || string(buf) != "hi" {
		t.Fatalf("回显失败: %q %v", buf, err)
	}
}
