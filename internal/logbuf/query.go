package logbuf

import (
	"bytes"
	"context"
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
	return b.QueryContext(context.Background(), after, before, limit, f)
}

// QueryContext 与 Query 相同，但查找日志文件时会定期检查 ctx：请求被取消（例如前端已换了过滤条件）后
// 立即停止扫描，返回已找到的部分结果。
func (b *Buffer) QueryContext(ctx context.Context, after, before uint64, limit int, f *Filter) []Entry {
	b.mu.Lock()
	// 从新到旧扫描缓冲，倒序收集候选日志。每次写日志都要拿 b.mu，
	// 有过滤条件时只在锁内复制序号范围内的日志，匹配放到锁外做，避免拖慢写入
	var cand []Entry
	filtered := !f.Empty()
	for i := b.count - 1; i >= 0; i-- {
		e := b.at(i)
		if before > 0 && e.Seq >= before {
			continue
		}
		if e.Seq <= after {
			break
		}
		cand = append(cand, e)
		if !filtered && limit > 0 && len(cand) >= limit {
			// 没有过滤条件时每条都命中，凑够即可停止
			break
		}
	}
	// oldest 为缓冲中最早一条的序号，文件里只取比它更早的日志，避免与缓冲重复
	oldest := b.nextSeq
	if b.count > 0 {
		oldest = b.at(0).Seq
	}
	sink := b.sink
	b.mu.Unlock()

	rev := cand
	if filtered {
		// 原地过滤：命中的日志依次前移，仍保持从新到旧的顺序
		rev = cand[:0]
		for i := range cand {
			if f.Match(&cand[i]) {
				rev = append(rev, cand[i])
				if limit > 0 && len(rev) >= limit {
					break
				}
			}
		}
	}

	if limit > 0 && len(rev) < limit && sink != nil && after+1 < oldest {
		upper := oldest
		if before > 0 && before < upper {
			upper = before
		}
		rev = append(rev, sink.search(ctx, after, upper, limit-len(rev), f)...)
	}
	// 翻转为时间顺序
	for i, j := 0, len(rev)-1; i < j; i, j = i+1, j-1 {
		rev[i], rev[j] = rev[j], rev[i]
	}
	return rev
}

// snapFile 是查找开始时打开的一个日志文件及当时的大小。
type snapFile struct {
	file *os.File
	size int64
}

// snapshot 在 ioMu 内先把待写盘的日志同步写入文件，再打开当前文件与 .1 历史文件并记下大小，按从新到旧排列。
//
// 先写盘是为了让刚被挤出内存缓冲、还在 pending 里等待批量写盘的日志也能被查到，否则“加载更早的日志”会漏掉这一段。
// 之后的扫描不再持有 ioMu，不阻塞写盘：轮转是把当前文件重命名为 .1、清空是删除文件，
// 在 Unix 上已打开的句柄仍指向原来的内容；新追加的内容在记下的大小之后，不会被读到半行。
// Windows 上文件被打开时重命名或删除可能失败，写盘逻辑会容忍（留到下次轮转），扫描很快结束后即恢复。
// 文件不存在时跳过；其他打开失败时不再继续打开更早的文件，避免结果中间缺一段。调用方负责关闭返回的文件。
func (s *fileSink) snapshot() []snapFile {
	s.ioMu.Lock()
	defer s.ioMu.Unlock()
	s.flushLocked()
	var out []snapFile
	for _, p := range []string{s.path, s.path + ".1"} {
		file, err := os.Open(p)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			break
		}
		st, err := file.Stat()
		if err != nil {
			_ = file.Close()
			break
		}
		out = append(out, snapFile{file: file, size: st.Size()})
	}
	return out
}

// search 从新到旧在日志文件中查找序号在 (after, before) 之间且满足条件的日志，最多 n 条，按倒序返回。
// ctx 被取消或读取出错时停止，返回已找到的部分。
func (s *fileSink) search(ctx context.Context, after, before uint64, n int, f *Filter) []Entry {
	files := s.snapshot()
	defer func() {
		for _, sf := range files {
			_ = sf.file.Close()
		}
	}()
	return searchFiles(ctx, files, after, before, n, f)
}

// searchFiles 依次（从新到旧）扫描已打开的日志文件，不涉及任何锁。
func searchFiles(ctx context.Context, files []snapFile, after, before uint64, n int, f *Filter) []Entry {
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
	for _, sf := range files {
		done, err := scanBackward(ctx, sf.file, sf.size, visit)
		if err != nil || !done {
			// 读取出错时不再继续读更早的文件，否则结果中间会缺一段
			break
		}
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

// scanBackward 从 file 的 size 处逐行向前读取，对每个非空行调用 fn，fn 返回 false 时停止。
// 返回值表示是否读完了整个文件，调用方据此决定是否继续读更早的文件；
// 读取出错或 ctx 被取消时返回错误，调用方应停止，避免跳过一段继续读更早的文件。
func scanBackward(ctx context.Context, file *os.File, size int64, fn func(line []byte) bool) (bool, error) {
	off := size
	// carry 为上一块开头尚未凑成整行的部分
	var carry []byte
	for off > 0 {
		// 每读一块检查一次是否已取消
		if err := ctx.Err(); err != nil {
			return false, err
		}
		step := min(int64(tailChunk), off)
		off -= step
		chunk := make([]byte, step, int(step)+len(carry))
		// ReadAt 读不满时一定返回错误（包括 io.EOF，说明文件比记下的大小短），都按读取失败处理
		if n, err := file.ReadAt(chunk, off); n < len(chunk) {
			if err == nil {
				err = io.ErrUnexpectedEOF
			}
			return false, err
		}
		data := append(chunk, carry...)
		// 从后往前切出完整的行；块开头的残行留到下一轮
		for {
			i := bytes.LastIndexByte(data, '\n')
			if i < 0 {
				break
			}
			if line := data[i+1:]; len(line) > 0 && !fn(line) {
				return false, nil
			}
			data = data[:i]
		}
		carry = data
	}
	if len(carry) > 0 && !fn(carry) {
		return false, nil
	}
	return true, nil
}
