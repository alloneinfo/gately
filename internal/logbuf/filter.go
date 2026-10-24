package logbuf

import (
	"net/netip"
	"strconv"
	"strings"
	"time"
)

// 常用的结构化字段名。写日志时附带这些字段，前端可以按字段精确过滤。
const (
	FieldIP     = "ip"
	FieldHost   = "host"
	FieldSite   = "site"
	FieldStatus = "status"
	FieldMethod = "method"
	FieldPath   = "path"
	FieldRule   = "rule"
	FieldUser   = "user"
)

// Fields 是一条日志附带的结构化字段，例如访问日志的客户端 IP、域名与状态码。
type Fields map[string]string

// Filter 是日志过滤条件，零值表示不过滤。各条件之间为“且”的关系。
type Filter struct {
	// MinLevel 为最低级别，例如 warn 只保留警告与错误
	MinLevel string
	// Source 为来源标签，不区分大小写精确匹配
	Source string
	// Terms 为关键词（已转小写），消息、来源与字段值中必须全部包含；Excludes 为排除词，包含任意一个即不匹配
	Terms    []string
	Excludes []string
	// IP 为客户端地址条件，命中任意一项即匹配
	IP *IPMatcher
	// Host 为站点或域名（已转小写）的子串
	Host string
	// Status 为状态码区间，命中任意一个即匹配；只匹配带 status 字段的日志
	Status []StatusRange
	// From、To 为时间范围（含两端），零值表示不限
	From, To time.Time
	// Translate 不为 nil 时，关键词与来源也和译文比较：接口按请求语言翻译日志后返回，
	// 用户输入的是界面上看到的译文，日志里保存的则是服务端语言的原文
	Translate func(string) string
}

// Empty 判断是否没有任何过滤条件。
func (f *Filter) Empty() bool {
	return f == nil || (f.MinLevel == "" && f.Source == "" && len(f.Terms) == 0 && len(f.Excludes) == 0 &&
		f.IP == nil && f.Host == "" && len(f.Status) == 0 && f.From.IsZero() && f.To.IsZero())
}

var levelOrder = map[string]int{LevelDebug: 0, LevelInfo: 1, LevelWarn: 2, LevelError: 3}

// ValidLevel 判断是否为已知的日志级别。
func ValidLevel(level string) bool {
	_, ok := levelOrder[level]
	return ok
}

// Match 判断一条日志是否满足全部条件。
func (f *Filter) Match(e *Entry) bool {
	if f.Empty() {
		return true
	}
	if f.MinLevel != "" && levelOrder[e.Level] < levelOrder[f.MinLevel] {
		return false
	}
	if f.Source != "" && !strings.EqualFold(e.Source, f.Source) && (f.Translate == nil || !strings.EqualFold(f.Translate(e.Source), f.Source)) {
		return false
	}
	if !f.From.IsZero() && e.Time.Before(f.From) {
		return false
	}
	if !f.To.IsZero() && e.Time.After(f.To) {
		return false
	}
	if len(f.Status) > 0 && !f.matchStatus(e) {
		return false
	}
	if f.Host != "" && !f.matchHost(e) {
		return false
	}
	if f.IP != nil && !f.IP.matchEntry(e) {
		return false
	}
	if len(f.Terms) > 0 || len(f.Excludes) > 0 {
		text := f.searchText(e)
		for _, t := range f.Terms {
			if !strings.Contains(text, t) {
				return false
			}
		}
		for _, t := range f.Excludes {
			if strings.Contains(text, t) {
				return false
			}
		}
	}
	return true
}

// searchText 返回关键词检索的文本：来源、消息与字段值（需要时附上来源与消息的译文），统一转小写。
func (f *Filter) searchText(e *Entry) string {
	var b strings.Builder
	b.Grow(len(e.Source) + len(e.Message) + 32)
	b.WriteString(e.Source)
	b.WriteByte('\n')
	b.WriteString(e.Message)
	if f.Translate != nil {
		// 译文与原文相同时（例如纯英文的访问日志）不重复拼接
		if tr := f.Translate(e.Source); tr != e.Source {
			b.WriteByte('\n')
			b.WriteString(tr)
		}
		if tr := f.Translate(e.Message); tr != e.Message {
			b.WriteByte('\n')
			b.WriteString(tr)
		}
	}
	for _, v := range e.Fields {
		b.WriteByte('\n')
		b.WriteString(v)
	}
	return strings.ToLower(b.String())
}

func (f *Filter) matchStatus(e *Entry) bool {
	code, err := strconv.Atoi(e.Fields[FieldStatus])
	if err != nil {
		return false
	}
	for _, r := range f.Status {
		if code >= r.Min && code <= r.Max {
			return true
		}
	}
	return false
}

// matchHost 优先匹配 host 与 site 字段；日志没有这两个字段时退回到在消息中查找。
func (f *Filter) matchHost(e *Entry) bool {
	host, hasHost := e.Fields[FieldHost]
	site, hasSite := e.Fields[FieldSite]
	if !hasHost && !hasSite {
		return strings.Contains(strings.ToLower(e.Message), f.Host)
	}
	return (hasHost && strings.Contains(strings.ToLower(host), f.Host)) || (hasSite && strings.Contains(strings.ToLower(site), f.Host))
}

// StatusRange 是闭区间 [Min, Max] 的状态码范围。
type StatusRange struct {
	Min, Max int
}

// ParseStatus 解析状态码条件，多项以逗号或空格分隔，每项可以是 404、4xx 或 500-599。
func ParseStatus(s string) ([]StatusRange, bool) {
	var out []StatusRange
	for _, item := range splitList(s) {
		item = strings.ToLower(item)
		var r StatusRange
		switch {
		case len(item) == 3 && item[1:] == "xx" && item[0] >= '1' && item[0] <= '5':
			r.Min = int(item[0]-'0') * 100
			r.Max = r.Min + 99
		case strings.Contains(item, "-"):
			lo, hi, _ := strings.Cut(item, "-")
			a, err1 := strconv.Atoi(lo)
			b, err2 := strconv.Atoi(hi)
			if err1 != nil || err2 != nil || a > b {
				return nil, false
			}
			r = StatusRange{a, b}
		default:
			n, err := strconv.Atoi(item)
			if err != nil {
				return nil, false
			}
			r = StatusRange{n, n}
		}
		if r.Min < 100 || r.Max > 599 {
			return nil, false
		}
		out = append(out, r)
	}
	return out, true
}

// IPMatcher 匹配客户端地址：可以是完整地址、CIDR 网段，或尚未输完整的地址片段（按子串匹配）。
type IPMatcher struct {
	prefixes []netip.Prefix
	texts    []string
}

// ParseIP 解析地址条件，多项以逗号或空格分隔。没有任何有效项时返回 nil。
func ParseIP(s string) *IPMatcher {
	m := &IPMatcher{}
	for _, item := range splitList(s) {
		if p, err := netip.ParsePrefix(item); err == nil {
			m.prefixes = append(m.prefixes, p.Masked())
		} else if a, err := netip.ParseAddr(item); err == nil {
			a = a.Unmap()
			m.prefixes = append(m.prefixes, netip.PrefixFrom(a, a.BitLen()))
		} else {
			m.texts = append(m.texts, strings.ToLower(item))
		}
	}
	if len(m.prefixes) == 0 && len(m.texts) == 0 {
		return nil
	}
	return m
}

// matchEntry 优先使用 ip 字段；日志没有该字段时，从消息中提取所有像地址的片段逐个比较。
func (m *IPMatcher) matchEntry(e *Entry) bool {
	if ip, ok := e.Fields[FieldIP]; ok {
		return m.match(ip)
	}
	for _, tok := range addrTokens(e.Message) {
		if m.match(tok) {
			return true
		}
	}
	return false
}

// match 判断一个地址字符串（可带端口）是否命中。
func (m *IPMatcher) match(s string) bool {
	a, ok := parseAddr(s)
	if ok {
		for _, p := range m.prefixes {
			if p.Contains(a) {
				return true
			}
		}
		s = a.String()
	}
	s = strings.ToLower(s)
	for _, t := range m.texts {
		if strings.Contains(s, t) {
			return true
		}
	}
	return false
}

// parseAddr 解析地址，兼容 1.2.3.4:80、[::1]:80 这类带端口的写法。
func parseAddr(s string) (netip.Addr, bool) {
	if a, err := netip.ParseAddr(s); err == nil {
		return a.Unmap(), true
	}
	if ap, err := netip.ParseAddrPort(s); err == nil {
		return ap.Addr().Unmap(), true
	}
	return netip.Addr{}, false
}

// addrTokens 从消息中切出可能是地址的片段：由十六进制字符、点、冒号、方括号组成且至少含一个点或冒号。
func addrTokens(msg string) []string {
	var out []string
	start := -1
	flush := func(end int) {
		if start >= 0 {
			// 去掉句末的点、冒号；保留方括号，[v6]:端口 交给 parseAddr 解析
			tok := strings.Trim(msg[start:end], ".:")
			if strings.HasPrefix(tok, "[") && strings.HasSuffix(tok, "]") {
				tok = tok[1 : len(tok)-1]
			}
			if strings.ContainsAny(tok, ".:") {
				out = append(out, tok)
			}
			start = -1
		}
	}
	for i := 0; i < len(msg); i++ {
		c := msg[i]
		isAddr := (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F') || c == '.' || c == ':' || c == '[' || c == ']'
		if isAddr {
			if start < 0 {
				start = i
			}
			continue
		}
		flush(i)
	}
	flush(len(msg))
	return out
}

// ParseKeywords 把关键词拆成包含词与排除词：以空格分隔，双引号包住的短语视为一个词，前缀 - 表示排除。
func ParseKeywords(s string) (terms, excludes []string) {
	var cur strings.Builder
	quoted, neg, has := false, false, false
	emit := func() {
		if has && cur.Len() > 0 {
			w := strings.ToLower(cur.String())
			if neg {
				excludes = append(excludes, w)
			} else {
				terms = append(terms, w)
			}
		}
		cur.Reset()
		quoted, neg, has = false, false, false
	}
	for _, r := range s {
		switch {
		case r == '"':
			if quoted {
				emit()
			} else {
				quoted, has = true, true
			}
		case !quoted && (r == ' ' || r == '\t' || r == '　'):
			emit()
		case r == '-' && !quoted && !has:
			// 词首的 - 表示排除；单独一个 - 不构成词
			neg, has = true, true
		default:
			cur.WriteRune(r)
			has = true
		}
	}
	emit()
	return terms, excludes
}

// splitList 按逗号、中文逗号与空白拆分并去掉空项。
func splitList(s string) []string {
	return strings.FieldsFunc(s, func(r rune) bool {
		return r == ',' || r == '，' || r == ' ' || r == '\t' || r == ';'
	})
}
