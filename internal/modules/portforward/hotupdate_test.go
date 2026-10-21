package portforward

import (
	"bytes"
	"context"
	"io"
	"log"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/wangwanjie/gately/internal/config"
	"github.com/wangwanjie/gately/internal/logbuf"
	"github.com/wangwanjie/gately/internal/module"
)

// startedModule 创建并启动只含一条规则的模块。
func startedModule(t *testing.T, rule Rule) *Module {
	t.Helper()
	st, err := config.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	m := New()
	if err := m.Init(module.Env{Store: st, Logs: logbuf.New(100, log.New(io.Discard, "", 0))}); err != nil {
		t.Fatal(err)
	}
	if err := rule.Validate(); err != nil {
		t.Fatal(err)
	}
	m.cfg.Rules = []Rule{rule}
	if err := m.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = m.Stop() })
	return m
}

// editRule 模拟接口保存：替换规则后调用 restartLocked。
func editRule(t *testing.T, m *Module, rule Rule) {
	t.Helper()
	if err := rule.Validate(); err != nil {
		t.Fatal(err)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cfg.Rules[0] = rule
	m.restartLocked(rule.Key)
}

// echoOnce 发送 msg 并读回同样长度的数据。
func echoOnce(t *testing.T, c net.Conn, msg []byte) error {
	t.Helper()
	if _, err := c.Write(msg); err != nil {
		return err
	}
	buf := make([]byte, len(msg))
	_ = c.SetReadDeadline(time.Now().Add(3 * time.Second))
	if _, err := io.ReadFull(c, buf); err != nil {
		return err
	}
	if !bytes.Equal(buf, msg) {
		t.Fatalf("回显不一致: %q", buf)
	}
	return nil
}

func TestEditRuleKeepsConnections(t *testing.T) {
	echoPort, closeEcho := tcpEcho(t)
	defer closeEcho()
	listenPort := freePort(t, "tcp")
	rule := Rule{
		Key:             "k1",
		Name:            "测试",
		Enable:          true,
		Protocols:       []string{"tcp4"},
		ListenAddress:   "127.0.0.1",
		ListenPorts:     strconv.Itoa(listenPort),
		TargetAddresses: []string{"127.0.0.1"},
		TargetPorts:     strconv.Itoa(echoPort),
		Remark:          "备注",
	}
	m := startedModule(t, rule)
	addr := net.JoinHostPort("127.0.0.1", strconv.Itoa(listenPort))
	old, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer old.Close()
	if err := echoOnce(t, old, []byte("before")); err != nil {
		t.Fatal(err)
	}

	// 删除备注、改名并开启 PROXY 协议：监听参数不变，应热更新
	rule.Remark = ""
	rule.Name = "改名"
	rule.ProxyProtocol = "v1"
	editRule(t, m, rule)
	if err := echoOnce(t, old, []byte("after")); err != nil {
		t.Fatalf("热更新后已有连接被断开: %v", err)
	}
	// 新连接使用新配置：回显服务会把 PROXY v1 头原样返回
	fresh, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer fresh.Close()
	if _, err := fresh.Write([]byte("x")); err != nil {
		t.Fatal(err)
	}
	head := make([]byte, 6)
	_ = fresh.SetReadDeadline(time.Now().Add(3 * time.Second))
	if _, err := io.ReadFull(fresh, head); err != nil {
		t.Fatal(err)
	}
	if string(head) != "PROXY " {
		t.Fatalf("新连接未使用新配置，收到 %q", head)
	}
	m.mu.Lock()
	got := m.viewLocked(m.cfg.Rules[0])
	m.mu.Unlock()
	if !got.Running || got.Stats.TotalConns != 2 {
		t.Fatalf("热更新后状态或统计异常: %+v", got)
	}

	// 改监听端口必须重启：旧连接随旧监听一起关闭
	newPort := freePort(t, "tcp")
	rule.ListenPorts = strconv.Itoa(newPort)
	editRule(t, m, rule)
	if err := echoOnce(t, old, []byte("gone")); err == nil {
		t.Fatal("修改监听端口后旧连接仍然可用")
	}
	if _, err := net.Dial("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(newPort))); err != nil {
		t.Fatalf("新端口未监听: %v", err)
	}

	// 禁用规则时停止监听
	rule.Enable = false
	editRule(t, m, rule)
	m.mu.Lock()
	_, running := m.runners[rule.Key]
	m.mu.Unlock()
	if running {
		t.Fatal("禁用后规则仍在运行")
	}
}
