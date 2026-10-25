// Package fileusers 保存 WebDAV、FTP、SMB 共用的文件服务共享用户。
// 共享用户与各服务自己的本地用户结构一致（用户名、密码、目录或多个目录），只定义一次，
// 在服务中打开「使用共享用户」后即可登录；与本地用户同名时只按本地用户校验。
// 列表保存在配置目录的 fileusers.json，可在任一文件服务的接口中读取和修改。
package fileusers

import (
	"crypto/subtle"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"golang.org/x/crypto/bcrypt"

	"github.com/wangwanjie/gately/internal/config"
	"github.com/wangwanjie/gately/internal/crud"
	"github.com/wangwanjie/gately/internal/httpx"
	"github.com/wangwanjie/gately/internal/module"
	"github.com/wangwanjie/gately/internal/mountfs"
)

// ID 是共享用户的配置名（fileusers.json）。
const ID = "fileusers"

// User 是一个文件服务共享用户，字段与各文件服务的本地用户一致。
type User struct {
	Name string `json:"name"`
	// Password 为明文或 bcrypt 哈希；SMB 的 NTLM 认证需要明文，bcrypt 哈希的用户无法登录 SMB
	Password string `json:"password"`
	// Root 为该用户可访问的目录（绝对路径），ReadOnly 作用于该目录；Mounts 非空时二者不再使用
	Root     string `json:"root"`
	ReadOnly bool   `json:"readOnly"`
	// Mounts 为多个目录，每个目录可单独设为只读
	Mounts []mountfs.Mount `json:"mounts,omitempty"`
}

// file 是 fileusers.json 的结构，外面包一层便于以后增加字段。
type file struct {
	Users []User `json:"users"`
}

// Validate 校验并规范化共享用户。用户名不区分大小写不可重复（SMB 按不区分大小写匹配用户名），
// 也不能包含冒号（HTTP 基本认证以冒号分隔用户名与密码）。
func Validate(users []User) error {
	seen := map[string]bool{}
	for i := range users {
		u := &users[i]
		u.Name = strings.TrimSpace(u.Name)
		if u.Name == "" || u.Password == "" {
			return errors.New("共享用户的用户名和密码不能为空")
		}
		if strings.Contains(u.Name, ":") {
			return fmt.Errorf("共享用户名 %q 不能包含冒号", u.Name)
		}
		key := strings.ToLower(u.Name)
		if seen[key] {
			return fmt.Errorf("共享用户 %s 重复（不区分大小写）", u.Name)
		}
		seen[key] = true
		if len(u.Mounts) > 0 {
			// 多目录模式下单目录字段清空，避免两套配置并存
			u.Root, u.ReadOnly = "", false
			if err := mountfs.Validate(u.Mounts); err != nil {
				return fmt.Errorf("共享用户 %s: %w", u.Name, err)
			}
			continue
		}
		u.Root = strings.TrimSpace(u.Root)
		if !filepath.IsAbs(u.Root) {
			return fmt.Errorf("共享用户 %s 的目录需为绝对路径", u.Name)
		}
	}
	return nil
}

// CheckPassword 校验密码：stored 为 bcrypt 哈希时按哈希比较，否则按明文常量时间比较。
func CheckPassword(stored, given string) bool {
	if strings.HasPrefix(stored, "$2") {
		return bcrypt.CompareHashAndPassword([]byte(stored), []byte(given)) == nil
	}
	return subtle.ConstantTimeCompare([]byte(stored), []byte(given)) == 1
}

// Registry 保存当前的共享用户，并在修改后通知各文件服务。
type Registry struct {
	mu    sync.RWMutex
	store *config.Store
	users []User
	// saveMu 串行化保存，保证写文件与内存列表的顺序一致
	saveMu sync.Mutex

	subMu  sync.Mutex
	subs   map[int]func()
	nextID int
}

// New 创建空的注册表。
func New() *Registry { return &Registry{subs: map[int]func(){}} }

// Default 是进程内唯一的注册表，由各文件服务在 Init 时以同一个配置目录初始化。
var Default = New()

// Init 从配置目录读取共享用户。多个模块以同一个配置目录调用时只读取一次；换了配置目录（测试中）时重新读取。
func (r *Registry) Init(store *config.Store) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.store == store {
		return nil
	}
	var f file
	if _, err := store.Load(ID, &f); err != nil {
		return err
	}
	r.store, r.users = store, f.Users
	return nil
}

// List 返回全部共享用户的副本。
func (r *Registry) List() []User {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return clone(r.users)
}

// Lookup 按用户名（区分大小写）查找共享用户。
func (r *Registry) Lookup(name string) (User, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, u := range r.users {
		if u.Name == name {
			u.Mounts = slices.Clone(u.Mounts)
			return u, true
		}
	}
	return User{}, false
}

// Save 校验并保存共享用户，仍为脱敏占位符的密码沿用原值，保存后同步通知订阅者。
func (r *Registry) Save(users []User) error {
	r.saveMu.Lock()
	defer r.saveMu.Unlock()
	r.mu.RLock()
	store, old := r.store, clone(r.users)
	r.mu.RUnlock()
	if store == nil {
		return errors.New("共享用户尚未初始化")
	}
	users = clone(users)
	// 用户没有固定 ID，改名后仍按位置保留原密码，见 crud.KeepSecrets
	crud.KeepSecrets(old, users, func(u *User) string { return u.Name }, func(u *User) *string { return &u.Password })
	if err := Validate(users); err != nil {
		return err
	}
	if err := store.Save(ID, file{Users: users}); err != nil {
		return err
	}
	r.mu.Lock()
	r.users = users
	r.mu.Unlock()
	r.notify()
	return nil
}

// Subscribe 注册共享用户变化时的回调，返回取消函数。回调在保存请求中同步执行，不能持有调用方的锁。
func (r *Registry) Subscribe(fn func()) (cancel func()) {
	r.subMu.Lock()
	id := r.nextID
	r.nextID++
	r.subs[id] = fn
	r.subMu.Unlock()
	return func() {
		r.subMu.Lock()
		delete(r.subs, id)
		r.subMu.Unlock()
	}
}

func (r *Registry) notify() {
	r.subMu.Lock()
	subs := make([]func(), 0, len(r.subs))
	for _, fn := range r.subs {
		subs = append(subs, fn)
	}
	r.subMu.Unlock()
	for _, fn := range subs {
		fn()
	}
}

// Masked 返回密码脱敏后的共享用户，供接口返回。
func (r *Registry) Masked() []User {
	users := r.List()
	for i := range users {
		users[i].Password = crud.Mask(users[i].Password)
	}
	return users
}

// Routes 注册读取与修改共享用户的接口；extra 为各服务附加在读取结果中的信息（如 SMB 无法使用的用户），可为空。
func (r *Registry) Routes(rt module.Router, extra func(map[string]any)) {
	reply := func(w http.ResponseWriter) {
		out := map[string]any{"users": r.Masked()}
		if extra != nil {
			extra(out)
		}
		httpx.OK(w, out)
	}
	rt.HandleFunc("GET /shared-users", func(w http.ResponseWriter, _ *http.Request) {
		reply(w)
	})
	rt.HandleFunc("PUT /shared-users", func(w http.ResponseWriter, req *http.Request) {
		var body struct {
			Users []User `json:"users"`
		}
		if err := httpx.ReadJSON(req, &body); err != nil {
			httpx.Error(w, http.StatusBadRequest, "%v", err)
			return
		}
		if err := r.Save(body.Users); err != nil {
			httpx.Error(w, http.StatusBadRequest, "%v", err)
			return
		}
		reply(w)
	})
}

func clone(users []User) []User {
	out := make([]User, len(users))
	for i, u := range users {
		u.Mounts = slices.Clone(u.Mounts)
		out[i] = u
	}
	return out
}
