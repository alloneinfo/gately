package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/wangwanjie/gately/internal/config"
)

// applyForbidden 是不允许通过声明式配置覆盖的配置名：账号凭据与令牌应当只通过登录会话管理。
var applyForbidden = map[string]bool{accountName: true, "tokens": true}

// applyConfigDir 把 dir 下的 <名称>.json 逐个写入配置目录，内容相同的文件不会重写。
// 写入前会校验是合法 JSON；被覆盖的旧内容保存在配置历史中，可在界面回滚。
func applyConfigDir(store *config.Store, dir string) (int, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, err
	}
	applied := 0
	for _, e := range entries {
		name, ok := strings.CutSuffix(e.Name(), ".json")
		if !ok || e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		if applyForbidden[name] {
			return applied, fmt.Errorf("%s.json 不允许通过声明式配置覆盖", name)
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return applied, err
		}
		if !json.Valid(data) {
			return applied, fmt.Errorf("%s 不是合法的 JSON", e.Name())
		}
		if !bytes.HasSuffix(data, []byte("\n")) {
			data = append(data, '\n')
		}
		cur, err := store.ReadRaw(name)
		if err != nil {
			return applied, err
		}
		if bytes.Equal(cur, data) {
			continue
		}
		if err := store.WriteRaw(name, data); err != nil {
			return applied, fmt.Errorf("写入 %s 失败: %w", e.Name(), err)
		}
		applied++
	}
	return applied, nil
}
