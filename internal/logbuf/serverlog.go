package logbuf

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
)

// serverLogThrottle 为同类服务器错误的最短记录间隔，避免浏览器重试或扫描刷屏
var serverLogThrottle = time.Minute

// serverLogMaxKeys 限制节流表大小，防止大量不同 SNI 等内容撑大内存
const serverLogMaxKeys = 256

// ServerErrorLog 返回可设置为 http.Server.ErrorLog 的标准日志器，
// 把 net/http 内部错误（如 TLS 握手失败）写入本模块日志，同时保留 docker logs 输出。
func (l *Logger) ServerErrorLog() *log.Logger {
	return log.New(&serverLogWriter{l: l, seen: map[string]*throttleState{}}, "", 0)
}

type throttleState struct {
	last       time.Time
	suppressed int
}

type serverLogWriter struct {
	l    *Logger
	mu   sync.Mutex
	seen map[string]*throttleState
}

func (w *serverLogWriter) Write(p []byte) (int, error) {
	msg := strings.TrimSpace(string(p))
	level, text, key := classifyServerLog(msg)
	if level == LevelDebug {
		// 客户端断开、超时等常见噪声只记调试级别，不做节流
		w.l.buf.Add(level, w.l.source, text)
		return len(p), nil
	}
	w.mu.Lock()
	now := time.Now()
	st, ok := w.seen[key]
	if ok && now.Sub(st.last) < serverLogThrottle {
		st.suppressed++
		w.mu.Unlock()
		return len(p), nil
	}
	if !ok {
		if len(w.seen) >= serverLogMaxKeys {
			w.seen = map[string]*throttleState{}
		}
		st = &throttleState{}
		w.seen[key] = st
	}
	if st.suppressed > 0 {
		text += fmt.Sprintf("（此前 %s 内同类错误另有 %d 次）", serverLogThrottle, st.suppressed)
	}
	st.last, st.suppressed = now, 0
	w.mu.Unlock()
	w.l.buf.Add(level, w.l.source, text)
	return len(p), nil
}

// classifyServerLog 解析 net/http 的错误日志，返回级别、展示文本与节流键。
func classifyServerLog(msg string) (level, text, key string) {
	const tlsPrefix = "http: TLS handshake error from "
	rest, ok := strings.CutPrefix(msg, tlsPrefix)
	if !ok {
		return LevelWarn, msg, msg
	}
	// 格式为 "<地址>: <原因>"，IPv6 地址带方括号，按第一个 ": " 切分即可
	addr, reason, found := strings.Cut(rest, ": ")
	if !found {
		return LevelWarn, "TLS 握手失败: " + rest, rest
	}
	text = fmt.Sprintf("TLS 握手失败（来自 %s）: %s", addr, reason)
	lower := strings.ToLower(reason)
	if lower == "eof" || strings.Contains(lower, "connection reset") || strings.Contains(lower, "i/o timeout") || strings.Contains(lower, "broken pipe") {
		return LevelDebug, text, ""
	}
	// 节流键去掉来源地址，同一原因（如缺少某域名证书）只按原因合并
	return LevelWarn, text, "tls:" + reason
}
