package logbuf

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"strconv"
)

// Query 返回序号在 (after, before) 之间、满足过滤条件的日志（按时间顺序），before 为 0 表示不限。
// limit 大于 0 时只保留其中最新的 limit 条；这时如果内存缓冲里凑不够，且开启了落盘，
// 会继续从日志文件（含 .1 历史文件）中查找比缓冲更早的日志，让过滤与“加载更早的日志”能覆盖文件里保留的全部历史。
// limit 为 0 时只查内存缓冲，避免一次返回整份日志文件。
func (b *Buffer) Query(after, before uint64, limit int, f *Filter) []Entry {
	b.mu.Lock()
	// 从新到旧扫描缓冲，倒序收集
	var rev []Entry
	for i := b.count - 1; i >= 0; i-- {
		e := b.at(i)
		if before > 0 && e.Seq >= before {
			continue
		}
		if e.Seq <= after {
			break
		}
		if f.Match(&e) {
			rev = append(rev, e)
			if limit > 0 && len(rev) >= limit {
				break
			}
		}
	}
	// oldest 为缓冲中最早一条的序号，文件里只取比它更早的日志，避免与缓冲重复
	oldest := b.nextSeq
	if b.count > 0 {
		oldest = b.at(0).Seq
	}
	sink := b.sink
	b.mu.Unlock()

	if limit > 0 && len(rev) < limit && sink != nil && after+1 < oldest {
		upper := oldest
		if before > 0 && before < upper {
			upper = before
		}
		rev = append(rev, sink.search(after, upper, limit-len(rev), f)...)
	}
	// 翻转为时间顺序
	for i, j := 0, len(rev)-1; i < j; i, j = i+1, j-1 {
		rev[i], rev[j] = rev[j], rev[i]
	}
	return rev
}

// search 从新到旧在日志文件中查找序号在 (after, before) 之间且满足条件的日志，最多 n 条，按倒序返回。
func (s *fileSink) search(after, before uint64, n int, f *Filter) []Entry {
	// 持有 ioMu，查找期间不会发生轮转或清空；新日志仍先进入内存，不受影响
	s.ioMu.Lock()
	defer s.ioMu.Unlock()
	var out []Entry
	pre := newPrefilter(f)
	visit := func(line []byte) bool {
		seq, ok := lineSeq(line)
		if !ok || seq >= before {
			return true
		}
		if seq <= after {
			// 文件按序号递增写入，再往前只会更小
			return false
		}
		if !pre.ok(line) {
			return true
		}
		var e Entry
		if json.Unmarshal(line, &e) != nil || e.Seq == 0 || !f.Match(&e) {
			return true
		}
		out = append(out, e)
		return len(out) < n
	}
	if scanBackward(s.path, visit) {
		scanBackward(s.path+".1", visit)
	}
	return out
}

// lineSeq 从一行 JSON 开头快速读出序号（Entry 序列化时 seq 是第一个字段），不必完整解析。
func lineSeq(line []byte) (uint64, bool) {
	const prefix = `{"seq":`
	if !bytes.HasPrefix(line, []byte(prefix)) {
		return 0, false
	}
	rest := line[len(prefix):]
	end := bytes.IndexByte(rest, ',')
	if end <= 0 {
		return 0, false
	}
	v, err := strconv.ParseUint(string(rest[:end]), 10, 64)
	return v, err == nil
}

// prefilter 在完整解析 JSON 之前，用原始字节排除明显不含关键词的行。
// JSON 序列化会转义引号、反斜杠与 <>&，含这些字符的关键词不做预过滤；需要与译文比较时也不做。
type prefilter struct {
	terms [][]byte
}

func newPrefilter(f *Filter) prefilter {
	var p prefilter
	if f == nil || f.Translate != nil {
		// 需要与译文比较时，原文里不一定包含关键词，不能预过滤
		return p
	}
	for _, t := range f.Terms {
		if !bytes.ContainsAny([]byte(t), "\"\\<>&") {
			p.terms = append(p.terms, []byte(t))
		}
	}
	return p
}

func (p prefilter) ok(line []byte) bool {
	if len(p.terms) == 0 {
		return true
	}
	lower := bytes.ToLower(line)
	for _, t := range p.terms {
		if !bytes.Contains(lower, t) {
			return false
		}
	}
	return true
}

// scanBackward 从文件末尾逐行向前读取，对每个非空行调用 fn，fn 返回 false 时停止。
// 返回值表示是否读完了整个文件（文件不存在也视为读完），调用方据此决定是否继续读更早的文件。
func scanBackward(path string, fn func(line []byte) bool) bool {
	file, err := os.Open(path)
	if err != nil {
		return true
	}
	defer file.Close()
	st, err := file.Stat()
	if err != nil {
		return true
	}
	off := st.Size()
	// carry 为上一块开头尚未凑成整行的部分
	var carry []byte
	for off > 0 {
		step := min(int64(tailChunk), off)
		off -= step
		chunk := make([]byte, step, int(step)+len(carry))
		if _, err := file.ReadAt(chunk, off); err != nil && !errors.Is(err, io.EOF) {
			return true
		}
		data := append(chunk, carry...)
		// 从后往前切出完整的行；块开头的残行留到下一轮
		for {
			i := bytes.LastIndexByte(data, '\n')
			if i < 0 {
				break
			}
			if line := data[i+1:]; len(line) > 0 && !fn(line) {
				return false
			}
			data = data[:i]
		}
		carry = data
	}
	if len(carry) > 0 && !fn(carry) {
		return false
	}
	return true
}
