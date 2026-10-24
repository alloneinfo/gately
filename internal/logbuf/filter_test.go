package logbuf

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParseKeywords(t *testing.T) {
	cases := []struct {
		in             string
		terms, exclude string
	}{
		{"", "[]", "[]"},
		{"  Login  失败 ", "[login 失败]", "[]"},
		{`"connection refused" -debug`, "[connection refused]", "[debug]"},
		{`-"a b" c`, "[c]", "[a b]"},
		{"- x", "[x]", "[]"},
		{"a-b　c", "[a-b c]", "[]"},
	}
	for _, c := range cases {
		terms, ex := ParseKeywords(c.in)
		if got := fmt.Sprint(terms); got != c.terms {
			t.Errorf("ParseKeywords(%q) terms = %s，期望 %s", c.in, got, c.terms)
		}
		if got := fmt.Sprint(ex); got != c.exclude {
			t.Errorf("ParseKeywords(%q) excludes = %s，期望 %s", c.in, got, c.exclude)
		}
	}
}

func TestParseStatus(t *testing.T) {
	ok := map[string]string{
		"404":      "[{404 404}]",
		"4xx, 502": "[{400 499} {502 502}]",
		"500-599":  "[{500 599}]",
		"2XX，3xx":  "[{200 299} {300 399}]",
		"":         "[]",
	}
	for in, want := range ok {
		got, valid := ParseStatus(in)
		if !valid || fmt.Sprint(got) != want {
			t.Errorf("ParseStatus(%q) = %v %v，期望 %s", in, got, valid, want)
		}
	}
	for _, in := range []string{"abc", "6xx", "99", "500-400", "1000", "4x"} {
		if _, valid := ParseStatus(in); valid {
			t.Errorf("ParseStatus(%q) 应当无效", in)
		}
	}
}

func TestIPMatcher(t *testing.T) {
	m := ParseIP("10.0.0.0/8, 2001:db8::1, 192.168.")
	cases := map[string]bool{
		"10.1.2.3":          true,
		"10.1.2.3:8080":     true,
		"[2001:db8::1]:443": true,
		"::ffff:10.0.0.1":   true,
		"192.168.1.20":      true,
		"172.16.0.1":        false,
		"2001:db8::2":       false,
	}
	for in, want := range cases {
		if got := m.match(in); got != want {
			t.Errorf("match(%q) = %v，期望 %v", in, got, want)
		}
	}
	if ParseIP(" , ") != nil {
		t.Error("没有有效项时应返回 nil")
	}
}

func TestAddrTokens(t *testing.T) {
	got := addrTokens("来自 1.2.3.4:5678 的登录失败，封禁 [2001:db8::1]:22 与 10.0.0.0/24。face")
	joined := strings.Join(got, " ")
	for _, s := range []string{"1.2.3.4:5678", "[2001:db8::1]:22", "10.0.0.0"} {
		if !strings.Contains(joined, s) {
			t.Errorf("addrTokens 结果 %q 缺少 %s", joined, s)
		}
	}
	m := ParseIP("2001:db8::1")
	e := Entry{Message: "封禁 [2001:db8::1]:22"}
	if !m.matchEntry(&e) {
		t.Error("应能从消息中提取带端口的 IPv6 地址")
	}
}

func TestFilterMatch(t *testing.T) {
	now := time.Now()
	access := Entry{Time: now, Level: LevelInfo, Source: "HTTPS :443", Message: "1.2.3.4 GET a.example.com/x 404 3ms",
		Fields: Fields{FieldIP: "1.2.3.4", FieldHost: "a.example.com", FieldSite: "博客", FieldStatus: "404"}}
	login := Entry{Time: now.Add(-time.Hour), Level: LevelWarn, Source: "auth", Message: "来自 5.6.7.8 的登录失败: 密码错误"}

	cases := []struct {
		name   string
		f      Filter
		access bool
		login  bool
	}{
		{"空条件", Filter{}, true, true},
		{"最低级别", Filter{MinLevel: LevelWarn}, false, true},
		{"来源不区分大小写", Filter{Source: "AUTH"}, false, true},
		{"关键词全部包含", Filter{Terms: []string{"get", "404"}}, true, false},
		{"关键词匹配字段值", Filter{Terms: []string{"博客"}}, true, false},
		{"排除词", Filter{Excludes: []string{"登录"}}, true, false},
		{"IP 字段", Filter{IP: ParseIP("1.2.3.4")}, true, false},
		{"IP 从消息提取", Filter{IP: ParseIP("5.6.0.0/16")}, false, true},
		{"站点名称", Filter{Host: "博客"}, true, false},
		{"域名片段", Filter{Host: "example"}, true, false},
		{"无字段时域名在消息中查找", Filter{Host: "5.6.7"}, false, true},
		{"状态码只匹配带字段的日志", Filter{Status: []StatusRange{{400, 499}}}, true, false},
		{"状态码不命中", Filter{Status: []StatusRange{{500, 599}}}, false, false},
		{"开始时间", Filter{From: now.Add(-time.Minute)}, true, false},
		{"结束时间", Filter{To: now.Add(-time.Minute)}, false, true},
	}
	for _, c := range cases {
		if got := c.f.Match(&access); got != c.access {
			t.Errorf("%s: 访问日志匹配 = %v，期望 %v", c.name, got, c.access)
		}
		if got := c.f.Match(&login); got != c.login {
			t.Errorf("%s: 登录日志匹配 = %v，期望 %v", c.name, got, c.login)
		}
	}
	var nilFilter *Filter
	if !nilFilter.Match(&login) {
		t.Error("nil 过滤条件应匹配全部")
	}
}

func TestFilterTranslate(t *testing.T) {
	tr := func(s string) string {
		return strings.NewReplacer("登录失败", "login failed", "入侵防护", "Intrusion").Replace(s)
	}
	e := Entry{Level: LevelWarn, Source: "入侵防护", Message: "来自 5.6.7.8 的登录失败"}
	f := Filter{Terms: []string{"login failed"}, Translate: tr}
	if !f.Match(&e) {
		t.Error("关键词应能匹配译文")
	}
	f = Filter{Terms: []string{"登录失败"}, Translate: tr}
	if !f.Match(&e) {
		t.Error("关键词仍应能匹配原文")
	}
	f = Filter{Source: "intrusion", Translate: tr}
	if !f.Match(&e) {
		t.Error("来源应能匹配译文")
	}
}

func TestQueryFilter(t *testing.T) {
	b := New(100, nil)
	for i := 1; i <= 30; i++ {
		l := b.Logger("r")
		if i%3 == 0 {
			l.With(Fields{FieldIP: "10.0.0.1"}).Warnf("第 %d 条", i)
		} else {
			l.Infof("第 %d 条", i)
		}
	}
	f := &Filter{MinLevel: LevelWarn}
	if got := fmt.Sprint(seqs(b.Query(0, 0, 0, f))); got != "[3 6 9 12 15 18 21 24 27 30]" {
		t.Errorf("按级别过滤 = %s", got)
	}
	// limit 取最新的若干条，before 向前翻页，since 增量拉取
	if got := fmt.Sprint(seqs(b.Query(0, 0, 3, f))); got != "[24 27 30]" {
		t.Errorf("limit = %s", got)
	}
	if got := fmt.Sprint(seqs(b.Query(0, 24, 3, f))); got != "[15 18 21]" {
		t.Errorf("before = %s", got)
	}
	if got := fmt.Sprint(seqs(b.Query(25, 0, 0, f))); got != "[27 30]" {
		t.Errorf("since = %s", got)
	}
	if got := fmt.Sprint(seqs(b.Query(0, 0, 2, &Filter{IP: ParseIP("10.0.0.1")}))); got != "[27 30]" {
		t.Errorf("按 IP 过滤 = %s", got)
	}
	if got := b.Query(0, 0, 0, nil); len(got) != 30 {
		t.Errorf("无过滤条件应返回全部，得到 %d 条", len(got))
	}
}

// TestQueryFile 验证内存缓冲凑不够时会继续查找日志文件（含 .1 历史文件），且不与缓冲重复。
func TestQueryFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "w.log")
	b := New(5, nil)
	// 文件上限很小，写入过程中会轮转出 .1
	if err := b.Persist(path, 1000); err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= 20; i++ {
		lv := LevelInfo
		if i%2 == 0 {
			lv = LevelError
		}
		b.Add(lv, "", fmt.Sprintf("第 %d 条日志", i))
		if i%4 == 0 {
			// 分批写盘，触发轮转
			b.sink.flush()
		}
	}
	b.sink.flush()
	defer b.ClosePersist()

	// 缓冲只有 16..20，更早的要从文件找
	f := &Filter{MinLevel: LevelError}
	got := seqs(b.Query(0, 0, 4, f))
	if fmt.Sprint(got) != "[14 16 18 20]" {
		t.Errorf("跨缓冲与文件查询 = %v", got)
	}
	// 加载更早：before 落在文件范围内
	got = seqs(b.Query(0, 14, 2, f))
	if fmt.Sprint(got) != "[10 12]" {
		t.Errorf("从文件加载更早的日志 = %v", got)
	}
	// 关键词预过滤：只在文件中出现的那条
	got = seqs(b.Query(0, 0, 10, &Filter{Terms: []string{"第 13 条"}}))
	if fmt.Sprint(got) != "[13]" {
		t.Errorf("文件中按关键词查找 = %v", got)
	}
	// 不带过滤条件时，加载更早的日志也能翻到文件里，结果按序号递增且不重复
	all := seqs(b.Query(0, 0, 100, nil))
	for i := 1; i < len(all); i++ {
		if all[i] <= all[i-1] {
			t.Fatalf("结果应按序号递增且不重复: %v", all)
		}
	}
	if fmt.Sprint(all) != "[9 10 11 12 13 14 15 16 17 18 19 20]" {
		t.Errorf("应包含文件中的历史日志: %v", all)
	}
	// since 落在缓冲范围内时只查缓冲
	if got := fmt.Sprint(seqs(b.Query(18, 0, 10, nil))); got != "[19 20]" {
		t.Errorf("增量拉取 = %s", got)
	}
	// limit 为 0 时不查文件
	if got := b.Query(0, 0, 0, nil); len(got) != 5 {
		t.Errorf("limit 为 0 时只返回缓冲，得到 %d 条", len(got))
	}
}

func TestScanBackward(t *testing.T) {
	path := filepath.Join(t.TempDir(), "x.log")
	var sb strings.Builder
	n := 5000
	for i := 0; i < n; i++ {
		fmt.Fprintf(&sb, `{"seq":%d,"message":"%s"}`+"\n", i+1, strings.Repeat("x", i%50))
	}
	if err := os.WriteFile(path, []byte(sb.String()), 0o600); err != nil {
		t.Fatal(err)
	}
	var got []uint64
	done := scanBackward(path, func(line []byte) bool {
		s, ok := lineSeq(line)
		if !ok {
			t.Fatalf("无法解析行 %q", line)
		}
		got = append(got, s)
		return true
	})
	if !done || len(got) != n || got[0] != uint64(n) || got[n-1] != 1 {
		t.Fatalf("倒序读取结果不对: done=%v 共 %d 条", done, len(got))
	}
	for i := 1; i < n; i++ {
		if got[i] != got[i-1]-1 {
			t.Fatalf("第 %d 行顺序错误: %d 之后是 %d", i, got[i-1], got[i])
		}
	}
	// 提前停止
	count := 0
	if scanBackward(path, func([]byte) bool { count++; return count < 3 }) || count != 3 {
		t.Errorf("提前停止失败，count=%d", count)
	}
}
