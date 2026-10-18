// Package crud 为"有序规则列表"类配置提供通用的增删改查、启停与排序接口。
//
// 各模块只需提供取 key/启用字段的访问器、校验函数和变更回调，即可获得一致的 REST 接口：
//
//	GET    /<path>               列表
//	POST   /<path>               新增
//	PUT    /<path>/{key}         修改
//	DELETE /<path>/{key}         删除
//	PUT    /<path>/{key}/enable  启停，body {"enable": bool}
//	PUT    /<path>-order         排序，body {"keys": [...]}
package crud

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"slices"
	"sync"

	"github.com/wangwanjie/gately/internal/httpx"
	"github.com/wangwanjie/gately/internal/module"
)

// Collection 管理一个 []T 列表。Mu 与模块共享，回调在持锁状态下执行。
type Collection[T any] struct {
	Mu    *sync.Mutex
	Items *[]T
	// Save 持久化整个模块配置
	Save func() error
	// Key 与 Enabled 返回字段指针，用于读写
	Key     func(*T) *string
	Enabled func(*T) *bool
	// Validate 校验并规范化输入
	Validate func(*T) error
	// Merge 在修改时合并旧值，例如保留被脱敏的密钥；可为 nil
	Merge func(old, updated *T)
	// View 把条目转换为输出结构（附加运行状态、脱敏）；可为 nil
	View func(*T) any
	// Changed 在条目新增、修改、启停后调用，用于重启运行实例
	Changed func(key string)
	// Removed 在条目删除后调用
	Removed func(key string)
	// BeforeRemove 在删除前调用（持锁），返回错误时拒绝删除，例如条目仍被引用；可为 nil
	BeforeRemove func(it *T) error
}

// NewKey 生成随机 key。
func NewKey() string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func (c *Collection[T]) index(key string) int {
	return slices.IndexFunc(*c.Items, func(it T) bool { return *c.Key(&it) == key })
}

// Find 返回 key 对应条目的指针，调用方需持有 Mu。
func (c *Collection[T]) Find(key string) *T {
	i := c.index(key)
	if i < 0 {
		return nil
	}
	return &(*c.Items)[i]
}

func (c *Collection[T]) view(it *T) any {
	if c.View != nil {
		return c.View(it)
	}
	return it
}

// Register 在 path 下注册全部接口。
func (c *Collection[T]) Register(r module.Router, path string) {
	r.HandleFunc("GET "+path, c.list)
	r.HandleFunc("POST "+path, c.create)
	r.HandleFunc("PUT "+path+"/{key}", c.update)
	r.HandleFunc("DELETE "+path+"/{key}", c.remove)
	r.HandleFunc("PUT "+path+"/{key}/enable", c.enable)
	r.HandleFunc("PUT "+path+"-order", c.order)

	// 登记接口文档：列表返回条目数组，新增与修改的请求体为条目本身
	var item T
	module.DocOf(r, "GET "+path, module.Doc{Summary: "列表", Resp: []T{}})
	module.DocOf(r, "POST "+path, module.Doc{Summary: "新增", Req: &item, Resp: &item})
	module.DocOf(r, "PUT "+path+"/{key}", module.Doc{Summary: "修改", Req: &item, Resp: &item})
	module.DocOf(r, "DELETE "+path+"/{key}", module.Doc{Summary: "删除"})
	module.DocOf(r, "PUT "+path+"/{key}/enable", module.Doc{Summary: "启用或停用", Req: &enableRequest{}})
	module.DocOf(r, "PUT "+path+"-order", module.Doc{Summary: "调整排序", Req: &orderRequest{}})
}

func (c *Collection[T]) list(w http.ResponseWriter, r *http.Request) {
	c.Mu.Lock()
	defer c.Mu.Unlock()
	out := make([]any, 0, len(*c.Items))
	for i := range *c.Items {
		out = append(out, c.view(&(*c.Items)[i]))
	}
	httpx.OK(w, out)
}

func (c *Collection[T]) create(w http.ResponseWriter, r *http.Request) {
	var it T
	if err := httpx.ReadJSON(r, &it); err != nil {
		httpx.Error(w, http.StatusBadRequest, "%v", err)
		return
	}
	if err := c.Validate(&it); err != nil {
		httpx.Error(w, http.StatusBadRequest, "%v", err)
		return
	}
	c.Mu.Lock()
	defer c.Mu.Unlock()
	key := NewKey()
	*c.Key(&it) = key
	*c.Items = append(*c.Items, it)
	if err := c.Save(); err != nil {
		*c.Items = (*c.Items)[:len(*c.Items)-1]
		httpx.Error(w, http.StatusInternalServerError, "保存失败: %v", err)
		return
	}
	if c.Changed != nil {
		c.Changed(key)
	}
	httpx.OK(w, c.view(c.Find(key)))
}

func (c *Collection[T]) update(w http.ResponseWriter, r *http.Request) {
	var it T
	if err := httpx.ReadJSON(r, &it); err != nil {
		httpx.Error(w, http.StatusBadRequest, "%v", err)
		return
	}
	key := r.PathValue("key")
	c.Mu.Lock()
	defer c.Mu.Unlock()
	i := c.index(key)
	if i < 0 {
		httpx.Error(w, http.StatusNotFound, "条目不存在")
		return
	}
	old := (*c.Items)[i]
	if c.Merge != nil {
		c.Merge(&old, &it)
	}
	if err := c.Validate(&it); err != nil {
		httpx.Error(w, http.StatusBadRequest, "%v", err)
		return
	}
	*c.Key(&it) = key
	(*c.Items)[i] = it
	if err := c.Save(); err != nil {
		(*c.Items)[i] = old
		httpx.Error(w, http.StatusInternalServerError, "保存失败: %v", err)
		return
	}
	if c.Changed != nil {
		c.Changed(key)
	}
	httpx.OK(w, c.view(&(*c.Items)[i]))
}

func (c *Collection[T]) remove(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	c.Mu.Lock()
	defer c.Mu.Unlock()
	i := c.index(key)
	if i < 0 {
		httpx.Error(w, http.StatusNotFound, "条目不存在")
		return
	}
	if c.BeforeRemove != nil {
		if err := c.BeforeRemove(&(*c.Items)[i]); err != nil {
			httpx.Error(w, http.StatusConflict, "%v", err)
			return
		}
	}
	old := slices.Clone(*c.Items)
	*c.Items = slices.Delete(*c.Items, i, i+1)
	if err := c.Save(); err != nil {
		*c.Items = old
		httpx.Error(w, http.StatusInternalServerError, "保存失败: %v", err)
		return
	}
	if c.Removed != nil {
		c.Removed(key)
	}
	httpx.OK(w, map[string]bool{"ok": true})
}

type enableRequest struct {
	Enable bool `json:"enable"`
}

func (c *Collection[T]) enable(w http.ResponseWriter, r *http.Request) {
	var req enableRequest
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "%v", err)
		return
	}
	key := r.PathValue("key")
	c.Mu.Lock()
	defer c.Mu.Unlock()
	it := c.Find(key)
	if it == nil {
		httpx.Error(w, http.StatusNotFound, "条目不存在")
		return
	}
	en := c.Enabled(it)
	prev := *en
	*en = req.Enable
	if err := c.Save(); err != nil {
		*en = prev
		httpx.Error(w, http.StatusInternalServerError, "保存失败: %v", err)
		return
	}
	if c.Changed != nil {
		c.Changed(key)
	}
	httpx.OK(w, map[string]bool{"ok": true})
}

type orderRequest struct {
	Keys []string `json:"keys"`
}

func (c *Collection[T]) order(w http.ResponseWriter, r *http.Request) {
	var req orderRequest
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "%v", err)
		return
	}
	c.Mu.Lock()
	defer c.Mu.Unlock()
	if len(req.Keys) != len(*c.Items) {
		httpx.Error(w, http.StatusBadRequest, "排序列表与条目数量不一致")
		return
	}
	reordered := make([]T, 0, len(req.Keys))
	seen := map[string]bool{}
	for _, k := range req.Keys {
		i := c.index(k)
		if i < 0 || seen[k] {
			httpx.Error(w, http.StatusBadRequest, "排序列表包含未知或重复条目 %s", k)
			return
		}
		seen[k] = true
		reordered = append(reordered, (*c.Items)[i])
	}
	old := *c.Items
	*c.Items = reordered
	if err := c.Save(); err != nil {
		*c.Items = old
		httpx.Error(w, http.StatusInternalServerError, "保存失败: %v", err)
		return
	}
	httpx.OK(w, map[string]bool{"ok": true})
}

// SecretMask 是 API 输出中替代真实密钥的占位符。客户端原样提交时服务端保留旧值。
const SecretMask = "********"

// Mask 返回脱敏后的值。
func Mask(s string) string {
	if s == "" {
		return ""
	}
	return SecretMask
}

// KeepSecret 若新值为占位符则恢复旧值。
func KeepSecret(old string, updated *string) {
	if *updated == SecretMask {
		*updated = old
	}
}
