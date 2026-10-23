package logbuf

import (
	"fmt"
	"testing"
)

// seqs 提取日志序号，便于断言。
func seqs(es []Entry) []uint64 {
	out := make([]uint64, len(es))
	for i, e := range es {
		out[i] = e.Seq
	}
	return out
}

func TestRange(t *testing.T) {
	// 容量 5，写入 8 条后环形缓冲已回绕，保留序号 4..8
	b := New(5, nil)
	for i := 0; i < 8; i++ {
		b.Add(LevelInfo, "", fmt.Sprint(i))
	}
	cases := []struct {
		after, before uint64
		limit         int
		want          string
	}{
		{0, 0, 0, "[4 5 6 7 8]"},
		{0, 0, 2, "[7 8]"},
		{6, 0, 0, "[7 8]"},
		{6, 0, 1, "[8]"},
		{8, 0, 0, "[]"},
		{0, 7, 0, "[4 5 6]"},
		{0, 7, 2, "[5 6]"},
		{0, 4, 0, "[]"},
		{4, 7, 0, "[5 6]"},
	}
	for _, c := range cases {
		got := fmt.Sprint(seqs(b.Range(c.after, c.before, c.limit)))
		if got != c.want {
			t.Errorf("Range(%d, %d, %d) = %s，期望 %s", c.after, c.before, c.limit, got, c.want)
		}
	}
	if got := fmt.Sprint(seqs(b.Since(5))); got != "[6 7 8]" {
		t.Errorf("Since(5) = %s", got)
	}
}
