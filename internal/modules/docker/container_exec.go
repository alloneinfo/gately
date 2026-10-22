package docker

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/coder/websocket"
)

// 本文件包含容器内执行命令的能力：
//   - 非交互执行（文件管理的 ls/mv/rm 使用）：创建 exec，读取多路复用输出并取得退出码；
//   - 交互终端：通过 WebSocket 与容器 TTY 双向转发，Engine 连接用 Upgrade: tcp 劫持为原始字节流。

func (m *Module) registerExecRoutes(add func(pattern string, h http.HandlerFunc)) {
	add("GET /containers/{id}/exec", m.handleExecWS)
}

// ---------- 多路复用输出 ----------

// demuxSplit 把 Docker 多路复用流拆成 stdout 与 stderr；不是该格式时全部视为 stdout。
func demuxSplit(data []byte) (stdout, stderr []byte) {
	rest := data
	var so, se bytes.Buffer
	for len(rest) >= 8 {
		stream := rest[0]
		if (stream != 0 && stream != 1 && stream != 2) || rest[1] != 0 || rest[2] != 0 || rest[3] != 0 {
			return data, nil
		}
		n := int(binary.BigEndian.Uint32(rest[4:8]))
		if n < 0 || 8+n > len(rest) {
			return data, nil
		}
		switch stream {
		case 2:
			se.Write(rest[8 : 8+n])
		default:
			so.Write(rest[8 : 8+n])
		}
		rest = rest[8+n:]
	}
	if len(rest) > 0 {
		return data, nil
	}
	return so.Bytes(), se.Bytes()
}

// ---------- Engine exec 接口 ----------

// execCreate 创建 exec 实例，返回其 ID。
func (c *Client) execCreate(ctx context.Context, containerID string, opts map[string]any) (string, error) {
	var out struct {
		ID string `json:"Id"`
	}
	err := c.do(ctx, http.MethodPost, "/containers/"+url.PathEscape(containerID)+"/exec", opts, &out)
	return out.ID, err
}

// execResize 调整 exec 的终端大小。
func (c *Client) execResize(ctx context.Context, execID string, cols, rows int) error {
	q := url.Values{"h": {strconv.Itoa(rows)}, "w": {strconv.Itoa(cols)}}
	return c.do(ctx, http.MethodPost, "/exec/"+url.PathEscape(execID)+"/resize?"+q.Encode(), nil, nil)
}

// execExitCode 读取 exec 的退出码；进程仍在运行时 running 为 true。
func (c *Client) execExitCode(ctx context.Context, execID string) (code int, running bool, err error) {
	var out struct {
		Running  bool `json:"Running"`
		ExitCode int  `json:"ExitCode"`
	}
	err = c.do(ctx, http.MethodGet, "/exec/"+url.PathEscape(execID)+"/json", nil, &out)
	return out.ExitCode, out.Running, err
}

// execOutputLimit 是非交互执行读取输出的上限。
const execOutputLimit = 16 << 20

// execRun 在容器内执行命令并等待结束，返回 stdout、stderr 与退出码。
func (c *Client) execRun(ctx context.Context, containerID string, cmd []string, user string) (stdout, stderr []byte, code int, err error) {
	opts := map[string]any{"AttachStdout": true, "AttachStderr": true, "Tty": false, "Cmd": cmd}
	if user != "" {
		opts["User"] = user
	}
	execID, err := c.execCreate(ctx, containerID, opts)
	if err != nil {
		return nil, nil, 0, err
	}
	payload, _ := json.Marshal(map[string]any{"Detach": false, "Tty": false})
	resp, err := c.rawRequest(ctx, http.MethodPost, "/exec/"+url.PathEscape(execID)+"/start", "application/json", bytes.NewReader(payload), nil)
	if err != nil {
		return nil, nil, 0, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, execOutputLimit))
	if err != nil {
		return nil, nil, 0, err
	}
	stdout, stderr = demuxSplit(data)
	// 输出流结束后退出码可能稍后才写入，短暂轮询
	for i := 0; i < 20; i++ {
		var running bool
		code, running, err = c.execExitCode(ctx, execID)
		if err != nil {
			return stdout, stderr, 0, err
		}
		if !running {
			return stdout, stderr, code, nil
		}
		select {
		case <-ctx.Done():
			return stdout, stderr, 0, ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
	}
	return stdout, stderr, code, errors.New("命令未在预期时间内结束")
}

// execOK 执行命令，退出码非 0 时把 stderr 作为错误返回。
func (c *Client) execOK(ctx context.Context, containerID string, cmd []string) ([]byte, error) {
	so, se, code, err := c.execRun(ctx, containerID, cmd, "")
	if err != nil {
		return nil, err
	}
	if code != 0 {
		msg := strings.TrimSpace(string(se))
		if msg == "" {
			msg = strings.TrimSpace(string(so))
		}
		if msg == "" {
			msg = fmt.Sprintf("退出码 %d", code)
		}
		return so, errors.New(msg)
	}
	return so, nil
}

// ---------- 连接劫持 ----------

// hijack 向 Engine 发起带 Upgrade: tcp 的请求，成功后返回原始连接。
// br 可能已缓冲了握手之后的数据，读取必须经过它。
func (c *Client) hijack(ctx context.Context, method, path string, body any) (net.Conn, *bufio.Reader, error) {
	if c.network == "" || c.addr == "" {
		return nil, nil, errors.New("客户端缺少拨号信息")
	}
	base, err := c.endpoint(ctx)
	if err != nil {
		return nil, nil, err
	}
	var d net.Dialer
	conn, err := d.DialContext(ctx, c.network, c.addr)
	if err != nil {
		return nil, nil, err
	}
	payload, err := json.Marshal(body)
	if err != nil {
		conn.Close()
		return nil, nil, err
	}
	req, err := http.NewRequestWithContext(ctx, method, base+path, bytes.NewReader(payload))
	if err != nil {
		conn.Close()
		return nil, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Upgrade", "tcp")
	// 握手阶段设置截止时间，成功后清除
	_ = conn.SetDeadline(time.Now().Add(15 * time.Second))
	if err := req.Write(conn); err != nil {
		conn.Close()
		return nil, nil, err
	}
	br := bufio.NewReader(conn)
	resp, err := http.ReadResponse(br, req)
	if err != nil {
		conn.Close()
		return nil, nil, err
	}
	if resp.StatusCode != http.StatusSwitchingProtocols && resp.StatusCode != http.StatusOK {
		defer conn.Close()
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
		var e struct {
			Message string `json:"message"`
		}
		_ = json.Unmarshal(b, &e)
		if e.Message == "" {
			e.Message = strings.TrimSpace(string(b))
		}
		return nil, nil, &APIError{Status: resp.StatusCode, Message: e.Message}
	}
	_ = conn.SetDeadline(time.Time{})
	return conn, br, nil
}

// ---------- WebSocket 终端 ----------

// execClientMsg 是浏览器发来的消息：t=i 输入，t=r 调整窗口大小（与 Web 终端协议一致）。
type execClientMsg struct {
	T string `json:"t"`
	D string `json:"d"`
	C int    `json:"c"`
	R int    `json:"r"`
}

func (m *Module) handleExecWS(w http.ResponseWriter, r *http.Request) {
	ws, err := websocket.Accept(w, r, nil)
	if err != nil {
		return
	}
	ws.SetReadLimit(1 << 20)
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	defer ws.Close(websocket.StatusNormalClosure, "")

	// 错误提示以文本消息 {"d":"..."} 发送，终端输出始终是二进制消息
	notice := func(msg string) {
		b, _ := json.Marshal(map[string]string{"d": msg})
		_ = ws.Write(ctx, websocket.MessageText, b)
	}
	c, err := m.client()
	if err != nil {
		notice("连接 Docker 失败: " + err.Error())
		return
	}
	id := r.PathValue("id")
	q := r.URL.Query()
	info, err := c.Inspect(ctx, id)
	if err != nil {
		notice("获取容器信息失败: " + err.Error())
		return
	}
	var state struct {
		Running bool `json:"Running"`
		Paused  bool `json:"Paused"`
	}
	var name string
	decodeField(info, "State", &state)
	decodeField(info, "Name", &name)
	name = trimSlash(name)
	if !state.Running || state.Paused {
		notice("容器未运行，无法打开终端")
		return
	}
	cmdStr := strings.TrimSpace(q.Get("cmd"))
	if cmdStr == "" {
		cmdStr = "/bin/sh"
	}
	cmd, err := shellSplit(cmdStr)
	if err != nil || len(cmd) == 0 {
		notice("命令格式错误")
		return
	}
	cols, _ := strconv.Atoi(q.Get("cols"))
	rows, _ := strconv.Atoi(q.Get("rows"))
	if cols <= 0 || rows <= 0 {
		cols, rows = 80, 24
	}
	opts := map[string]any{
		"AttachStdin": true, "AttachStdout": true, "AttachStderr": true, "Tty": true,
		"Cmd": cmd, "Env": []string{"TERM=xterm-256color"},
	}
	if user := strings.TrimSpace(q.Get("user")); user != "" {
		opts["User"] = user
	}
	log := m.logs.Logger(name)
	execID, err := c.execCreate(ctx, id, opts)
	if err != nil {
		log.Warnf("创建终端失败: %v", err)
		notice("创建终端失败: " + err.Error())
		return
	}
	conn, br, err := c.hijack(ctx, http.MethodPost, "/exec/"+url.PathEscape(execID)+"/start", map[string]any{"Detach": false, "Tty": true})
	if err != nil {
		log.Warnf("启动终端失败: %v", err)
		notice("启动终端失败: " + err.Error())
		return
	}
	defer conn.Close()
	// 先调整大小再开始转发，避免首屏按默认尺寸排版
	rctx, rcancel := context.WithTimeout(ctx, 5*time.Second)
	_ = c.execResize(rctx, execID, cols, rows)
	rcancel()
	log.Infof("终端会话开始（来自 %s，命令 %s）", r.RemoteAddr, cmdStr)
	started := time.Now()

	// 容器输出 -> 浏览器
	go func() {
		defer cancel()
		buf := make([]byte, 32*1024)
		for {
			n, err := br.Read(buf)
			if n > 0 {
				if werr := ws.Write(ctx, websocket.MessageBinary, buf[:n]); werr != nil {
					return
				}
			}
			if err != nil {
				return
			}
		}
	}()
	// 连接上下文结束（进程退出或浏览器断开）时关闭容器侧连接，让上面的读取返回
	go func() {
		<-ctx.Done()
		conn.Close()
	}()
	// 浏览器输入 -> 容器
	for {
		_, data, err := ws.Read(ctx)
		if err != nil {
			break
		}
		var msg execClientMsg
		if json.Unmarshal(data, &msg) != nil {
			continue
		}
		switch msg.T {
		case "i":
			if _, err := conn.Write([]byte(msg.D)); err != nil {
				cancel()
			}
		case "r":
			if msg.C > 0 && msg.R > 0 {
				rc, rcl := context.WithTimeout(context.Background(), 5*time.Second)
				_ = c.execResize(rc, execID, msg.C, msg.R)
				rcl()
			}
		}
		if ctx.Err() != nil {
			break
		}
	}
	log.Infof("终端会话结束，持续 %s", time.Since(started).Round(time.Second))
}
