package luckyimport

import (
	"encoding/json"
	"fmt"
	"path"
	"strings"

	"github.com/wangwanjie/gately/internal/mountfs"
)

// 本文件解析 Lucky「网络存储」相关配置：存储管理列表，以及 FTP / WebDAV 共用的用户与挂载列表结构。

// Lucky 存储类型。
const (
	StoreLocal     = "local"
	StoreWebDAV    = "webdav"
	StoreAliyunpan = "aliyunpan"
)

// StorageItem 是 Lucky 存储管理（lucky_storagemanagement.lkcf）中的一项。
type StorageItem struct {
	Key      string
	Remark   string
	Type     string
	Enable   bool
	Writable bool
	Params   struct {
		LocalPath          string
		WebDAVURL          string
		WebDAVUser         string
		WebDAVPasswd       string
		WebDAVAnonymous    bool
		InsecureSkipVerify bool
		// Proxy 为 http / https / socks5，非空时经代理访问 WebDAV
		Proxy     string
		ProxyAddr string
	}
	// SystemMount 把存储挂载到系统目录（依赖 rclone / FUSE）
	SystemMount struct {
		Enable     bool
		MountType  string
		MountPoint string
	}
}

// StorageList 返回存储管理列表；doc 不是存储管理配置时 ok 为 false。
// 存储管理配置固定带有 AliyunpanConf 字段，以此与同样使用 List 字段的 frp、cloudflared 配置区分。
func StorageList(doc Document) (items []StorageItem, ok bool, err error) {
	if _, has := doc["AliyunpanConf"]; !has {
		return nil, false, nil
	}
	raw, has := doc["List"]
	if !has {
		return nil, true, nil
	}
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, true, err
	}
	return items, true, nil
}

// ListTypes 返回 doc 中 List 各项的 Type 字段，用于区分共用 List 字段的配置文件。
func ListTypes(doc Document) []string {
	raw, ok := doc["List"]
	if !ok {
		return nil
	}
	var items []struct{ Type string }
	if json.Unmarshal(raw, &items) != nil {
		return nil
	}
	types := make([]string, len(items))
	for i, it := range items {
		types[i] = it.Type
	}
	return types
}

// AllTypes 判断 types 非空且每一项都属于 allowed。
func AllTypes(types []string, allowed ...string) bool {
	if len(types) == 0 {
		return false
	}
	for _, t := range types {
		ok := false
		for _, a := range allowed {
			if t == a {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}
	return true
}

// luckyFileUser 是 Lucky FTP / WebDAV 的用户。同一用户名可以出现多次，每次带不同的挂载列表，
// Lucky 会把它们合并为一个用户，登录后根目录列出全部挂载。
type luckyFileUser struct {
	Username  string
	Password  string
	MountList []struct {
		// Type 为 local（本地目录，Param 为路径）、store（存储管理，Param 为存储 Key）或 rclone
		Type        string
		Param       string
		DisplayName string
		Writable    bool
	}
}

// FileUser 是转换后的 FTP / WebDAV 用户，挂载列表已通过 mountfs.Validate。
type FileUser struct {
	Name     string
	Password string
	Mounts   []mountfs.Mount
}

// ConvertFileUsers 把 Lucky 的用户列表转换为多目录用户。opts.ResolveStore 用于解析 store 类型的挂载。
func ConvertFileUsers(raw json.RawMessage, opts Options) ([]FileUser, []string, error) {
	var src []luckyFileUser
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &src); err != nil {
			return nil, nil, err
		}
	}
	var users []FileUser
	var warnings []string
	index := map[string]int{}
	for _, lu := range src {
		name := strings.TrimSpace(lu.Username)
		if name == "" || lu.Password == "" {
			warnings = append(warnings, "跳过用户名或密码为空的用户")
			continue
		}
		i, seen := index[name]
		if seen && users[i].Password != lu.Password {
			warnings = append(warnings, fmt.Sprintf("用户 %s 出现多次且密码不同，仅保留第一个", name))
			continue
		}
		if !seen {
			i = len(users)
			index[name] = i
			users = append(users, FileUser{Name: name, Password: lu.Password})
		}
		u := &users[i]
		for _, ml := range lu.MountList {
			dir, why := resolveMount(ml.Type, ml.Param, opts)
			if why != "" {
				warnings = append(warnings, fmt.Sprintf("用户 %s 的挂载 %s: %s，已跳过", name, firstNonEmpty(ml.DisplayName, ml.Param), why))
				continue
			}
			u.Mounts = append(u.Mounts, mountfs.Mount{
				Name:     uniqueName(u.Mounts, mountName(ml.DisplayName, dir)),
				Path:     dir,
				ReadOnly: !ml.Writable,
			})
		}
	}
	// 没有任何可用挂载的用户无法登录后访问文件，不导入
	out := users[:0]
	for _, u := range users {
		if err := mountfs.Validate(u.Mounts); err != nil {
			warnings = append(warnings, fmt.Sprintf("用户 %s: %v，已跳过", u.Name, err))
			continue
		}
		if len(u.Mounts) == 0 {
			warnings = append(warnings, fmt.Sprintf("用户 %s 没有可迁移的目录，已跳过", u.Name))
			continue
		}
		out = append(out, u)
	}
	return out, warnings, nil
}

// resolveMount 返回挂载对应的本地目录；无法迁移时返回原因。
func resolveMount(typ, param string, opts Options) (dir, why string) {
	switch typ {
	case "local":
		return strings.TrimSpace(param), ""
	case "store":
		if opts.ResolveStore == nil {
			return "", "引用了存储管理中的存储，请同时导入 lucky_storagemanagement.lkcf"
		}
		return opts.ResolveStore(param)
	case "rclone":
		return "", "rclone 网盘挂载暂不支持"
	default:
		return "", fmt.Sprintf("不支持的挂载类型 %q", typ)
	}
}

// mountName 优先使用 Lucky 中的显示名称，为空时取目录名。
func mountName(display, dir string) string {
	name := strings.TrimSpace(display)
	if name == "" {
		name = path.Base(strings.ReplaceAll(dir, "\\", "/"))
	}
	// 名称会成为虚拟根目录下的目录名，不能包含路径分隔符
	name = strings.NewReplacer("/", "_", "\\", "_").Replace(name)
	if name == "" || name == "." || name == ".." {
		name = "dir"
	}
	return name
}

// uniqueName 在名称重复时追加序号。
func uniqueName(mounts []mountfs.Mount, name string) string {
	taken := map[string]bool{}
	for _, mt := range mounts {
		taken[mt.Name] = true
	}
	if !taken[name] {
		return name
	}
	for n := 2; ; n++ {
		candidate := fmt.Sprintf("%s-%d", name, n)
		if !taken[candidate] {
			return candidate
		}
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// StoreResolver 根据存储管理列表生成 ResolveStore：本地存储返回目录，其他类型返回原因。
// fallback 在 items 中找不到 Key 时调用，可为 nil。
func StoreResolver(items map[string]StorageItem, fallback func(key string) (string, bool)) func(string) (string, string) {
	return func(key string) (string, string) {
		if it, ok := items[key]; ok {
			if it.Type != StoreLocal {
				return "", fmt.Sprintf("引用的存储 %s 类型为 %s，仅支持本地目录", it.Remark, it.Type)
			}
			return it.Params.LocalPath, ""
		}
		if fallback != nil {
			if dir, ok := fallback(key); ok {
				return dir, ""
			}
		}
		return "", "引用了存储管理中的存储，请同时导入 lucky_storagemanagement.lkcf"
	}
}
