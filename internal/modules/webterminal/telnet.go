package webterminal

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Telnet 协议常量（RFC 854 及相关选项）
const (
	telIAC  = 255
	telDONT = 254
	telDO   = 253
	telWONT = 252
	telWILL = 251
	telSB   = 250
	telSE   = 240

	optEcho  = 1
	optSGA   = 3
	optTType = 24
	optNAWS  = 31
)

// TelnetConfig 是 Telnet 连接参数。
type TelnetConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	// Timeout 为连接超时（秒）
	Timeout int   `json:"timeout"`
	Proxy   Proxy `json:"proxy"`
}

// telnetSession 实现 session：在原始 TCP 连接上完成 IAC 选项协商，
// 对上层只暴露净化后的终端数据流。
type telnetSession struct {
	conn net.Conn
	pr   *io.PipeReader
	pw   *io.PipeWriter

	// wmu 保护对 conn 的写入
	wmu sync.Mutex

	// mu 保护选项协商状态与窗口大小
	mu         sync.Mutex
	us, him    map[byte]bool
	cols, rows int

	// remoteEcho 表示对端接管回显（字符模式），用于决定回车的发送形式
	remoteEcho atomic.Bool

	// 登录提示自动填充
	user, pass         string
	userSent, passSent bool
	tail               []byte

	closeOnce sync.Once
}

func (t *telnetSession) Read(p []byte) (int, error) { return t.pr.Read(p) }

// eol 返回回车键应发送的字节：字符模式遵循 NVT 规范发送 CR NUL，行模式发送 CR LF。
func (t *telnetSession) eol() []byte {
	if t.remoteEcho.Load() {
		return []byte{'\r', 0}
	}
	return []byte{'\r', '\n'}
}

// Write 把浏览器输入编码为 NVT 数据：转义 IAC，并转换回车。
func (t *telnetSession) Write(p []byte) (int, error) {
	out := make([]byte, 0, len(p)+8)
	for i := 0; i < len(p); i++ {
		b := p[i]
		switch {
		case b == telIAC:
			out = append(out, telIAC, telIAC)
		case b == '\r':
			if i+1 < len(p) && p[i+1] == '\n' {
				out = append(out, '\r', '\n')
				i++
			} else {
				out = append(out, t.eol()...)
			}
		default:
			out = append(out, b)
		}
	}
	if err := t.writeRaw(out); err != nil {
		return 0, err
	}
	return len(p), nil
}

func (t *telnetSession) writeRaw(b []byte) error {
	t.wmu.Lock()
	defer t.wmu.Unlock()
	_, err := t.conn.Write(b)
	return err
}

// Resize 更新窗口大小；对端启用 NAWS 后立即通知。
func (t *telnetSession) Resize(cols, rows int) error {
	t.mu.Lock()
	t.cols, t.rows = cols, rows
	enabled := t.us[optNAWS]
	t.mu.Unlock()
	if enabled {
		return t.sendNAWS()
	}
	return nil
}

func (t *telnetSession) Close() error {
	t.closeOnce.Do(func() {
		_ = t.conn.Close()
		_ = t.pw.CloseWithError(io.EOF)
	})
	return nil
}

func (t *telnetSession) sendCmd(cmd, opt byte) {
	_ = t.writeRaw([]byte{telIAC, cmd, opt})
}

func escapeIAC(b []byte) []byte {
	return bytes.ReplaceAll(b, []byte{telIAC}, []byte{telIAC, telIAC})
}

func (t *telnetSession) sendNAWS() error {
	t.mu.Lock()
	c, r := t.cols, t.rows
	t.mu.Unlock()
	data := []byte{byte(c >> 8), byte(c), byte(r >> 8), byte(r)}
	msg := []byte{telIAC, telSB, optNAWS}
	msg = append(msg, escapeIAC(data)...)
	msg = append(msg, telIAC, telSE)
	return t.writeRaw(msg)
}

// handleOption 处理 WILL / WONT / DO / DONT 协商，只在状态变化时应答，避免协商死循环。
func (t *telnetSession) handleOption(cmd, opt byte) {
	t.mu.Lock()
	var reply byte
	sendNAWS := false
	switch cmd {
	case telDO:
		switch opt {
		case optTType, optNAWS, optSGA:
			if !t.us[opt] {
				t.us[opt] = true
				reply = telWILL
			}
			sendNAWS = opt == optNAWS
		default:
			reply = telWONT
		}
	case telDONT:
		if t.us[opt] {
			t.us[opt] = false
			reply = telWONT
		}
	case telWILL:
		switch opt {
		case optEcho, optSGA:
			if !t.him[opt] {
				t.him[opt] = true
				reply = telDO
			}
			if opt == optEcho {
				t.remoteEcho.Store(true)
			}
		default:
			reply = telDONT
		}
	case telWONT:
		if t.him[opt] {
			t.him[opt] = false
			reply = telDONT
		}
		if opt == optEcho {
			t.remoteEcho.Store(false)
		}
	}
	t.mu.Unlock()
	if reply != 0 {
		t.sendCmd(reply, opt)
	}
	if sendNAWS {
		_ = t.sendNAWS()
	}
}

// handleSubnegotiation 处理子协商：对端请求终端类型时回复 xterm-256color。
func (t *telnetSession) handleSubnegotiation(data []byte) {
	if len(data) >= 2 && data[0] == optTType && data[1] == 1 {
		msg := []byte{telIAC, telSB, optTType, 0}
		msg = append(msg, "xterm-256color"...)
		msg = append(msg, telIAC, telSE)
		_ = t.writeRaw(msg)
	}
}

// 登录提示的识别后缀（统一小写、去除尾部空白后比较）
var (
	userPrompts = []string{"login:", "username:", "user name:", "user:", "login as:", "login name:", "用户名:", "用户名：", "账号:", "账号："}
	passPrompts = []string{"password:", "passcode:", "密码:", "密码："}
)

// autoLogin 根据输出末尾的提示自动填入用户名与密码，各只尝试一次，避免密码错误时反复提交导致账号锁定。
func (t *telnetSession) autoLogin(out []byte) {
	if (t.user == "" || t.userSent) && (t.pass == "" || t.passSent) {
		return
	}
	t.tail = append(t.tail, out...)
	if len(t.tail) > 256 {
		t.tail = t.tail[len(t.tail)-256:]
	}
	text := strings.ToLower(strings.TrimRight(string(t.tail), " \t\r\n\x00"))
	if !t.userSent && t.user != "" {
		for _, p := range userPrompts {
			if strings.HasSuffix(text, p) {
				t.userSent = true
				t.tail = nil
				_ = t.writeRaw(append(escapeIAC([]byte(t.user)), t.eol()...))
				return
			}
		}
	}
	if !t.passSent && t.pass != "" {
		for _, p := range passPrompts {
			if strings.HasSuffix(text, p) {
				t.passSent = true
				t.tail = nil
				_ = t.writeRaw(append(escapeIAC([]byte(t.pass)), t.eol()...))
				return
			}
		}
	}
}

// readLoop 解析 IAC 序列，把数据字节写入管道。
func (t *telnetSession) readLoop(r io.Reader) {
	br := bufio.NewReaderSize(r, 16*1024)
	out := make([]byte, 0, 4096)
	flush := func() error {
		if len(out) == 0 {
			return nil
		}
		t.autoLogin(out)
		_, err := t.pw.Write(out)
		out = out[:0]
		return err
	}
	prevCR := false
	for {
		if br.Buffered() == 0 {
			if err := flush(); err != nil {
				return
			}
		}
		b, err := br.ReadByte()
		if err != nil {
			_ = flush()
			_ = t.pw.CloseWithError(io.EOF)
			return
		}
		if b != telIAC {
			// NVT 中 CR 后的 NUL 表示单独的回车，丢弃该 NUL
			if b == 0 && prevCR {
				prevCR = false
				continue
			}
			prevCR = b == '\r'
			out = append(out, b)
			if len(out) >= 4096 {
				if err := flush(); err != nil {
					return
				}
			}
			continue
		}
		prevCR = false
		cmd, err := br.ReadByte()
		if err != nil {
			_ = t.pw.CloseWithError(io.EOF)
			return
		}
		switch cmd {
		case telIAC:
			out = append(out, telIAC)
		case telDO, telDONT, telWILL, telWONT:
			opt, err := br.ReadByte()
			if err != nil {
				_ = t.pw.CloseWithError(io.EOF)
				return
			}
			t.handleOption(cmd, opt)
		case telSB:
			var data []byte
			for {
				c, err := br.ReadByte()
				if err != nil {
					_ = t.pw.CloseWithError(io.EOF)
					return
				}
				if c != telIAC {
					data = append(data, c)
					continue
				}
				n, err := br.ReadByte()
				if err != nil {
					_ = t.pw.CloseWithError(io.EOF)
					return
				}
				if n == telSE {
					break
				}
				if n == telIAC {
					data = append(data, telIAC)
				}
			}
			t.handleSubnegotiation(data)
		default:
			// NOP、GA 等无需处理的命令直接忽略
		}
	}
}

// startTelnet 建立 Telnet 连接（可经代理）并启动协商。
func startTelnet(ctx context.Context, c *Connection, cols, rows int) (session, error) {
	t := c.Telnet
	nc, err := dialTelnet(ctx, &t)
	if err != nil {
		return nil, err
	}
	pr, pw := io.Pipe()
	s := &telnetSession{
		conn: nc, pr: pr, pw: pw,
		us: map[byte]bool{}, him: map[byte]bool{},
		cols: cols, rows: rows,
		user: t.Username, pass: t.Password,
	}
	// 主动声明能力：终端类型、窗口大小、抑制 GA；协商状态同步记录避免重复应答
	s.mu.Lock()
	s.us[optTType], s.us[optNAWS], s.us[optSGA], s.him[optSGA] = true, true, true, true
	s.mu.Unlock()
	s.sendCmd(telWILL, optTType)
	s.sendCmd(telWILL, optNAWS)
	s.sendCmd(telWILL, optSGA)
	s.sendCmd(telDO, optSGA)
	go s.readLoop(nc)
	return s, nil
}

func telnetTimeout(t *TelnetConfig) time.Duration {
	if t.Timeout <= 0 {
		return 15 * time.Second
	}
	return time.Duration(t.Timeout) * time.Second
}

func dialTelnet(ctx context.Context, t *TelnetConfig) (net.Conn, error) {
	addr := net.JoinHostPort(t.Host, strconv.Itoa(t.Port))
	return dialTarget(ctx, t.Proxy, addr, telnetTimeout(t))
}
