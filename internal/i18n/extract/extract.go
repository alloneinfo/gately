// Package extract 扫描 Go 源码，收集所有含中文的字符串字面量，作为运行时翻译的键。
package extract

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var han = regexp.MustCompile(`\p{Han}`)

// Literals 返回 roots 下所有非测试 Go 文件里含中文的字符串字面量（去重、排序）。
// 跳过 i18n 包自身，避免文档示例被当作键。
func Literals(roots ...string) ([]string, error) {
	set := map[string]struct{}{}
	for _, root := range roots {
		err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if filepath.Base(p) == "i18n" || filepath.Base(p) == "node_modules" {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
				return nil
			}
			f, err := parser.ParseFile(token.NewFileSet(), p, nil, 0)
			if err != nil {
				return err
			}
			ast.Inspect(f, func(n ast.Node) bool {
				bl, ok := n.(*ast.BasicLit)
				if !ok || bl.Kind != token.STRING {
					return true
				}
				if s, err := strconv.Unquote(bl.Value); err == nil && han.MatchString(s) {
					set[s] = struct{}{}
				}
				return true
			})
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	out := make([]string, 0, len(set))
	for s := range set {
		out = append(out, s)
	}
	sort.Strings(out)
	return out, nil
}
