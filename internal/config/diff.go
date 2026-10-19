package config

import (
	"strconv"
	"strings"
)

// DiffLine 是行级差异中的一行。Op 为 " "（相同）、"+"（新增）、"-"（删除）、"@"（省略的未变更行数）。
type DiffLine struct {
	Op   string `json:"op"`
	Text string `json:"text"`
}

// 差异计算的上限：超过时退化为"整段替换"，避免超大文件占满 CPU 与内存。
const maxDiffCells = 4_000_000

// 变更行前后保留的上下文行数
const diffContext = 3

// Diff 对两段文本做行级比较，返回带上下文的差异。相同的长段落折叠为一行 "@"。
func Diff(oldText, newText string) []DiffLine {
	a := splitLines(oldText)
	b := splitLines(newText)
	// 去掉公共前后缀，缩小需要比较的区间
	pre := 0
	for pre < len(a) && pre < len(b) && a[pre] == b[pre] {
		pre++
	}
	suf := 0
	for suf < len(a)-pre && suf < len(b)-pre && a[len(a)-1-suf] == b[len(b)-1-suf] {
		suf++
	}
	midA := a[pre : len(a)-suf]
	midB := b[pre : len(b)-suf]

	var mid []DiffLine
	if len(midA)*len(midB) > maxDiffCells {
		for _, l := range midA {
			mid = append(mid, DiffLine{"-", l})
		}
		for _, l := range midB {
			mid = append(mid, DiffLine{"+", l})
		}
	} else {
		mid = lcsDiff(midA, midB)
	}

	all := make([]DiffLine, 0, pre+len(mid)+suf)
	for _, l := range a[:pre] {
		all = append(all, DiffLine{" ", l})
	}
	all = append(all, mid...)
	for _, l := range a[len(a)-suf:] {
		all = append(all, DiffLine{" ", l})
	}
	return collapse(all)
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(strings.TrimSuffix(s, "\n"), "\n")
}

// lcsDiff 用最长公共子序列求差异。
func lcsDiff(a, b []string) []DiffLine {
	n, m := len(a), len(b)
	dp := make([][]int32, n+1)
	for i := range dp {
		dp[i] = make([]int32, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if a[i] == b[j] {
				dp[i][j] = dp[i+1][j+1] + 1
			} else if dp[i+1][j] >= dp[i][j+1] {
				dp[i][j] = dp[i+1][j]
			} else {
				dp[i][j] = dp[i][j+1]
			}
		}
	}
	var out []DiffLine
	i, j := 0, 0
	for i < n && j < m {
		switch {
		case a[i] == b[j]:
			out = append(out, DiffLine{" ", a[i]})
			i++
			j++
		case dp[i+1][j] >= dp[i][j+1]:
			out = append(out, DiffLine{"-", a[i]})
			i++
		default:
			out = append(out, DiffLine{"+", b[j]})
			j++
		}
	}
	for ; i < n; i++ {
		out = append(out, DiffLine{"-", a[i]})
	}
	for ; j < m; j++ {
		out = append(out, DiffLine{"+", b[j]})
	}
	return out
}

// collapse 只保留变更行及其上下文，其余相同行折叠成一条 "@" 记录（Text 为被省略的行数）。
func collapse(lines []DiffLine) []DiffLine {
	keep := make([]bool, len(lines))
	for i, l := range lines {
		if l.Op == " " {
			continue
		}
		for k := i - diffContext; k <= i+diffContext; k++ {
			if k >= 0 && k < len(lines) {
				keep[k] = true
			}
		}
	}
	out := []DiffLine{}
	changed := false
	for _, l := range lines {
		if l.Op != " " {
			changed = true
			break
		}
	}
	if !changed {
		return out
	}
	skipped := 0
	flush := func() {
		if skipped > 0 {
			out = append(out, DiffLine{"@", strconv.Itoa(skipped)})
			skipped = 0
		}
	}
	for i, l := range lines {
		if keep[i] {
			flush()
			out = append(out, l)
		} else {
			skipped++
		}
	}
	flush()
	return out
}
