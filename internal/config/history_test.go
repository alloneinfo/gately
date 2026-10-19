package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHistorySnapshotAndRestore(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	s.EnableHistory(3, "account")
	for _, v := range []string{"v1", "v2", "v2", "v3", "v4", "v5"} {
		if err := s.WriteRaw("demo", []byte(v)); err != nil {
			t.Fatal(err)
		}
	}
	// 首次写入没有旧内容；重复内容不产生版本；最多保留 3 个
	vs, _ := s.Versions("demo")
	if len(vs) != 3 {
		t.Fatalf("应保留 3 个版本，实际 %d", len(vs))
	}
	first, _ := s.ReadVersion("demo", vs[0].ID)
	last, _ := s.ReadVersion("demo", vs[2].ID)
	if string(first) != "v4" || string(last) != "v2" {
		t.Fatalf("版本内容/顺序不符: %q %q", first, last)
	}
	// 回滚到 v3，当前的 v5 变成新版本，回滚可撤销
	if err := s.RestoreVersion("demo", vs[1].ID); err != nil {
		t.Fatal(err)
	}
	cur, _ := s.ReadRaw("demo")
	if string(cur) != "v3" {
		t.Fatalf("回滚后内容 = %q", cur)
	}
	vs, _ = s.Versions("demo")
	top, _ := s.ReadVersion("demo", vs[0].ID)
	if string(top) != "v5" {
		t.Fatalf("回滚前的内容应保存为最新版本，实际 %q", top)
	}
	// 跳过名单与非法 ID
	_ = s.WriteRaw("account", []byte("a"))
	_ = s.WriteRaw("account", []byte("b"))
	if vs, _ := s.Versions("account"); len(vs) != 0 {
		t.Fatal("account 不应记录历史")
	}
	if _, err := s.ReadVersion("demo", "../../etc/passwd"); err != ErrNoVersion {
		t.Fatalf("非法 ID 应被拒绝: %v", err)
	}
	// 历史目录不应出现在顶层 *.json 中
	if m, _ := filepath.Glob(filepath.Join(s.Dir(), "*.json")); len(m) != 2 {
		t.Fatalf("顶层配置文件数量异常: %v", m)
	}
	if names := s.HistoryNames(); len(names) != 1 || names[0] != "demo" {
		t.Fatalf("HistoryNames = %v", names)
	}
	_ = os.RemoveAll(s.Dir())
}

func TestDiff(t *testing.T) {
	var old, nw []string
	for i := 0; i < 30; i++ {
		old = append(old, "line")
		nw = append(nw, "line")
	}
	nw[15] = "changed"
	d := Diff(strings.Join(old, "\n")+"\n", strings.Join(nw, "\n")+"\n")
	var plus, minus, skip int
	for _, l := range d {
		switch l.Op {
		case "+":
			plus++
		case "-":
			minus++
		case "@":
			skip++
		}
	}
	if plus != 1 || minus != 1 || skip != 2 {
		t.Fatalf("差异不符预期: %+v", d)
	}
	if len(Diff("a\n", "a\n")) != 0 {
		t.Fatal("相同内容应无差异")
	}
}
