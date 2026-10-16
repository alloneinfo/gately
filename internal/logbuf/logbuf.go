// Package logbuf 提供带序号的内存环形日志缓冲，供前端增量拉取。
package logbuf

import (
	"fmt"
	"log"
	"sync"
	"time"
)

// 日志级别
const (
	LevelDebug = "debug"
	LevelInfo  = "info"
	LevelWarn  = "warn"
	LevelError = "error"
)

// Entry 是一条日志。Seq 单调递增，前端用它做增量拉取。
type Entry struct {
	Seq     uint64    `json:"seq"`
	Time    time.Time `json:"time"`
	Level   string    `json:"level"`
	Source  string    `json:"source,omitempty"`
	Message string    `json:"message"`
}

// Buffer 是固定容量的环形缓冲，写满后覆盖最旧的日志。
type Buffer struct {
	mu      sync.Mutex
	entries []Entry
	start   int // 最旧一条在 entries 中的下标
	count   int
	nextSeq uint64
	// echo 不为 nil 时同时输出到标准日志，便于 docker logs 查看
	echo *log.Logger
}

// New 创建容量为 size 的缓冲。
func New(size int, echo *log.Logger) *Buffer {
	if size <= 0 {
		size = 500
	}
	return &Buffer{entries: make([]Entry, size), nextSeq: 1, echo: echo}
}

// Add 追加一条日志。
func (b *Buffer) Add(level, source, msg string) {
	b.mu.Lock()
	e := Entry{Seq: b.nextSeq, Time: time.Now(), Level: level, Source: source, Message: msg}
	b.nextSeq++
	idx := (b.start + b.count) % len(b.entries)
	b.entries[idx] = e
	if b.count < len(b.entries) {
		b.count++
	} else {
		b.start = (b.start + 1) % len(b.entries)
	}
	b.mu.Unlock()
	if b.echo != nil {
		if source != "" {
			b.echo.Printf("[%s] [%s] %s", level, source, msg)
		} else {
			b.echo.Printf("[%s] %s", level, msg)
		}
	}
}

// Since 返回序号大于 seq 的全部日志（按时间顺序）。seq 为 0 时返回全部。
func (b *Buffer) Since(seq uint64) []Entry {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]Entry, 0, b.count)
	for i := 0; i < b.count; i++ {
		e := b.entries[(b.start+i)%len(b.entries)]
		if e.Seq > seq {
			out = append(out, e)
		}
	}
	return out
}

// Clear 清空缓冲，序号继续递增，保证前端增量拉取不会错乱。
func (b *Buffer) Clear() {
	b.mu.Lock()
	b.start = 0
	b.count = 0
	b.mu.Unlock()
}

// Logger 是绑定了来源标签的便捷写入器。
type Logger struct {
	buf    *Buffer
	source string
}

// Logger 返回带来源标签的写入器，例如每条转发规则一个。
func (b *Buffer) Logger(source string) *Logger {
	return &Logger{buf: b, source: source}
}

// Debugf 写入调试日志。
func (l *Logger) Debugf(format string, args ...any) {
	l.buf.Add(LevelDebug, l.source, fmt.Sprintf(format, args...))
}

// Infof 写入普通日志。
func (l *Logger) Infof(format string, args ...any) {
	l.buf.Add(LevelInfo, l.source, fmt.Sprintf(format, args...))
}

// Warnf 写入警告日志。
func (l *Logger) Warnf(format string, args ...any) {
	l.buf.Add(LevelWarn, l.source, fmt.Sprintf(format, args...))
}

// Errorf 写入错误日志。
func (l *Logger) Errorf(format string, args ...any) {
	l.buf.Add(LevelError, l.source, fmt.Sprintf(format, args...))
}

// Resize 修改缓冲容量，保留最新的日志。
func (b *Buffer) Resize(size int) {
	if size <= 0 {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if size == len(b.entries) {
		return
	}
	keep := min(b.count, size)
	next := make([]Entry, size)
	for i := 0; i < keep; i++ {
		next[i] = b.entries[(b.start+b.count-keep+i)%len(b.entries)]
	}
	b.entries, b.start, b.count = next, 0, keep
}
