//go:build linux || darwin

package smb

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/wangwanjie/gately/internal/banlist"
	"github.com/wangwanjie/gately/internal/gosamba/config"
	"github.com/wangwanjie/gately/internal/gosamba/discovery"
	"github.com/wangwanjie/gately/internal/gosamba/parent"
	"github.com/wangwanjie/gately/internal/gosamba/transport"
	"github.com/wangwanjie/gately/internal/gosamba/userdb"
	"github.com/wangwanjie/gately/internal/logbuf"
)

// supported 表示当前系统可以运行 SMB 服务。
const supported = true

// durableTimeout 是断线后可续用的文件句柄保留时长，与 gosamba 默认值一致。
const durableTimeout = 60 * time.Second

// server 是一次启动的 SMB 服务，持有监听与全部连接。
type server struct {
	ln     net.Listener
	cancel context.CancelFunc
	done   chan struct{}
}

// startServer 按配置快照监听并开始服务。
func startServer(s Settings, logs *logbuf.Buffer) (*server, error) {
	log := slog.New(&logHandler{logs: logs, debug: os.Getenv(debugEnv) != ""})
	uid, gid := os.Getuid(), os.Getgid()
	users := make([]config.UserConfig, 0, len(s.Users))
	for _, u := range s.Users {
		shares := []config.ShareConfig{}
		for _, sh := range u.Shares() {
			// 目录不存在时只记录警告，不影响其他共享；客户端访问该共享时会收到错误
			if st, err := os.Stat(sh.Path); err != nil || !st.IsDir() {
				logs.Add(logbuf.LevelWarn, "", fmt.Sprintf("用户 %s 的共享 %s 目录不可用: %s", u.Name, sh.Name, sh.Path))
			}
			shares = append(shares, config.ShareConfig{Name: sh.Name, Path: sh.Path, ReadOnly: sh.ReadOnly})
		}
		users = append(users, config.UserConfig{
			Name:   u.Name,
			NTHash: userdb.NTHash(u.Password),
			// 文件操作以 Gately 进程身份进行，不切换系统用户
			SystemUID:   uid,
			SystemGID:   gid,
			AllowShares: []string{"*"},
			Shares:      shares,
		})
	}
	opts := parent.ConnOptions{
		RequireEncryption: s.RequireEncryption,
		RequireSigning:    true,
		MaxIOSize:         8 << 20,
		Users:             users,
		// 全局共享为空：每个用户只能看到并连接自己的共享
		Shares:         []config.ShareConfig{},
		DurableTimeout: durableTimeout,
	}

	ln, err := net.Listen("tcp", net.JoinHostPort(s.ListenIP, strconv.Itoa(s.Port)))
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	srv := &server{ln: ln, cancel: cancel, done: make(chan struct{})}

	// 断线续用的句柄表与会话索引在整个服务内共享，客户端换新连接后仍能找回原来的句柄与会话
	durable := parent.NewDurableTable()
	durable.StartSweeper(ctx, 30*time.Second)
	sessions := parent.NewSessionIndex()
	l := &parent.Listener{
		Log:      log,
		MaxFrame: transport.MaxFrameSize,
		Handler: func(ctx context.Context, c net.Conn, lg *slog.Logger, maxFrame uint32) {
			o := opts
			o.Durable = durable
			o.Sessions = sessions
			parent.ServeConn(ctx, c, lg, maxFrame, o)
		},
	}
	if s.MDNS {
		advertise(ctx, ln, log, logs)
	}
	go func() {
		defer close(srv.done)
		// 被封禁的地址在 Accept 时直接断开
		if err := l.Serve(ctx, banlist.Default.Listener(ln)); err != nil && !errors.Is(err, net.ErrClosed) {
			logs.Add(logbuf.LevelWarn, "", "SMB 服务退出: "+err.Error())
		}
	}()
	return srv, nil
}

// advertise 通过 mDNS 广播 _smb._tcp 服务，失败只记录日志。
func advertise(ctx context.Context, ln net.Listener, log *slog.Logger, logs *logbuf.Buffer) {
	port := ln.Addr().(*net.TCPAddr).Port
	host, _ := os.Hostname()
	instance, host := mdnsNames(host)
	closer, err := discovery.Advertise(ctx, instance, host, port, log)
	if err != nil {
		logs.Add(logbuf.LevelWarn, "", "Bonjour 广播失败: "+err.Error())
		return
	}
	go func() {
		<-ctx.Done()
		_ = closer.Close()
	}()
}

// mdnsNames 由系统主机名生成 Bonjour 服务实例名与主机名：主机名只取第一段，
// DNS 标签不超过 63 字节，实例名形如「主机名 (Gately)」，便于与 NAS 自带的 SMB 服务区分。
func mdnsNames(hostname string) (instance, host string) {
	host, _, _ = strings.Cut(hostname, ".")
	if host == "" {
		host = "gately"
	}
	host = truncateBytes(host, 63)
	return truncateBytes(host, 63-len(" (Gately)")) + " (Gately)", host
}

// truncateBytes 按字节截断字符串，不截断多字节字符。
func truncateBytes(s string, n int) string {
	if len(s) <= n {
		return s
	}
	for n > 0 && !utf8.RuneStart(s[n]) {
		n--
	}
	return s[:n]
}

// Addr 返回实际监听地址。
func (s *server) Addr() string { return s.ln.Addr().String() }

// Close 关闭监听与全部连接，最多等待 5 秒让连接收尾。
func (s *server) Close() {
	s.cancel()
	select {
	case <-s.done:
	case <-time.After(5 * time.Second):
	}
}

// failedUser 匹配 gosamba 认证失败错误中的用户名。
var failedUser = regexp.MustCompile(`user ("(?:[^"\\]|\\.)*")`)

// logHandler 把 gosamba 的结构化日志转换为模块运行日志：
// 登录成功与认证失败改写为易读的中文记录，其余只保留警告与错误，连接建立、协商等调试信息丢弃。
type logHandler struct {
	logs  *logbuf.Buffer
	attrs []slog.Attr
	// debug 为 true 时原样记录 gosamba 的全部日志（含调试级别），用于排查客户端兼容问题
	debug bool
}

// debugEnv 为非空时开启 SMB 协议调试日志。
const debugEnv = "GATELY_SMB_DEBUG"

func (h *logHandler) Enabled(_ context.Context, l slog.Level) bool {
	return h.debug || l >= slog.LevelInfo
}

func (h *logHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &logHandler{logs: h.logs, attrs: append(append([]slog.Attr{}, h.attrs...), attrs...), debug: h.debug}
}

// WithGroup 不区分分组，属性仍按名称读取。
func (h *logHandler) WithGroup(string) slog.Handler { return h }

func (h *logHandler) Handle(_ context.Context, r slog.Record) error {
	attrs := map[string]string{}
	var extra []string
	collect := func(a slog.Attr) bool {
		v := a.Value.Resolve().String()
		attrs[a.Key] = v
		if a.Key != "remote" {
			extra = append(extra, a.Key+"="+v)
		}
		return true
	}
	for _, a := range h.attrs {
		collect(a)
	}
	r.Attrs(collect)
	remote := attrs["remote"]
	if h.debug {
		h.logs.Add(logbuf.LevelDebug, "", strings.TrimSpace(remote+" "+r.Message+" "+strings.Join(extra, " ")))
	}
	switch r.Message {
	case "session authenticated":
		h.logs.Add(logbuf.LevelInfo, "", fmt.Sprintf("%s 登录（用户 %q）", remote, attrs["smb_user"]))
		return nil
	case "auth failed":
		// 错误形如 unknown user "x" 或 user "x": ...，取出用户名，与 FTP 的认证失败记录格式一致
		if m := failedUser.FindStringSubmatch(attrs["err"]); m != nil {
			if name, err := strconv.Unquote(m[1]); err == nil {
				h.logs.Add(logbuf.LevelWarn, "", fmt.Sprintf("%s 认证失败（用户 %q）", remote, name))
				return nil
			}
		}
		h.logs.Add(logbuf.LevelWarn, "", fmt.Sprintf("%s 认证失败: %s", remote, attrs["err"]))
		return nil
	case "session-setup error":
		// 认证失败已由 auth failed 记录，这里只是同一错误的重复
		if strings.Contains(attrs["err"], "authentication failed") || strings.Contains(attrs["err"], "unknown user") {
			return nil
		}
	case "delete-on-close failed":
		// 访达删除文件夹时会先对非空目录设置关闭时删除，随后逐个删除内容，属于正常现象
		if strings.Contains(attrs["err"], "directory not empty") {
			return nil
		}
	case "negotiate failed":
		// 客户端常同时尝试多个地址，再关掉多余的连接，协商前断开属于正常现象
		if strings.HasSuffix(attrs["err"], "EOF") || strings.Contains(attrs["err"], "connection reset") {
			return nil
		}
	}
	if r.Level < slog.LevelWarn {
		return nil
	}
	level := logbuf.LevelWarn
	if r.Level >= slog.LevelError {
		level = logbuf.LevelError
	}
	msg := r.Message
	if len(extra) > 0 {
		msg += " " + strings.Join(extra, " ")
	}
	if remote != "" {
		msg = remote + " " + msg
	}
	h.logs.Add(level, "", msg)
	return nil
}
