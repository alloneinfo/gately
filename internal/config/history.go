package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// historyDir 是配置目录下保存历史版本的子目录：history/<配置名>/<版本ID>.json。
const historyDir = "history"

// 版本 ID 即写入时刻（UTC），按字典序即时间序。
const versionLayout = "20060102T150405.000000000Z"

var validVersion = regexp.MustCompile(`^\d{8}T\d{6}\.\d{9}Z$`)

// ErrNoVersion 表示指定的历史版本不存在。
var ErrNoVersion = errors.New("历史版本不存在")

// Version 是某个配置文件的一个历史版本（被覆盖前的内容）。
type Version struct {
	ID   string    `json:"id"`
	Time time.Time `json:"time"`
	Size int64     `json:"size"`
}

type historyConfig struct {
	keep int
	skip map[string]bool
}

// EnableHistory 开启历史版本：每次覆盖配置文件前，把旧内容保存为一个版本，
// 每个配置最多保留 keep 个，skip 中的配置不记录（例如高频写入或含账号凭据的文件）。
func (s *Store) EnableHistory(keep int, skip ...string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if keep <= 0 {
		s.history = nil
		return
	}
	h := &historyConfig{keep: keep, skip: map[string]bool{}}
	for _, n := range skip {
		h.skip[n] = true
	}
	s.history = h
}

func (s *Store) versionDir(name string) string {
	return filepath.Join(s.dir, historyDir, name)
}

// snapshotLocked 在 rename 覆盖前，把现有内容保存为历史版本。内容未变化时不保存。
func (s *Store) snapshotLocked(name, p string, next []byte) {
	h := s.history
	if h == nil || h.skip[name] {
		return
	}
	old, err := os.ReadFile(p)
	if err != nil || string(old) == string(next) {
		return
	}
	dir := s.versionDir(name)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return
	}
	id := time.Now().UTC().Format(versionLayout)
	if err := os.WriteFile(filepath.Join(dir, id+".json"), old, 0o600); err != nil {
		return
	}
	// 超出保留数量时删除最旧的版本
	ids := s.versionIDsLocked(name)
	for len(ids) > h.keep {
		_ = os.Remove(filepath.Join(dir, ids[0]+".json"))
		ids = ids[1:]
	}
}

// versionIDsLocked 按时间升序返回某配置的版本 ID。
func (s *Store) versionIDsLocked(name string) []string {
	entries, err := os.ReadDir(s.versionDir(name))
	if err != nil {
		return nil
	}
	var ids []string
	for _, e := range entries {
		id := strings.TrimSuffix(e.Name(), ".json")
		if !e.IsDir() && id != e.Name() && validVersion.MatchString(id) {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	return ids
}

// HistoryNames 返回存在历史版本的配置名。
func (s *Store) HistoryNames() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	entries, err := os.ReadDir(filepath.Join(s.dir, historyDir))
	if err != nil {
		return nil
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() && validName.MatchString(e.Name()) && len(s.versionIDsLocked(e.Name())) > 0 {
			names = append(names, e.Name())
		}
	}
	return names
}

// Versions 返回某配置的历史版本，最新的在前。
func (s *Store) Versions(name string) ([]Version, error) {
	if !validName.MatchString(name) {
		return nil, fmt.Errorf("非法的配置名: %q", name)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	ids := s.versionIDsLocked(name)
	out := make([]Version, 0, len(ids))
	for i := len(ids) - 1; i >= 0; i-- {
		t, err := time.Parse(versionLayout, ids[i])
		if err != nil {
			continue
		}
		v := Version{ID: ids[i], Time: t}
		if st, err := os.Stat(filepath.Join(s.versionDir(name), ids[i]+".json")); err == nil {
			v.Size = st.Size()
		}
		out = append(out, v)
	}
	return out, nil
}

// ReadVersion 读取某个历史版本的内容。
func (s *Store) ReadVersion(name, id string) ([]byte, error) {
	if !validName.MatchString(name) || !validVersion.MatchString(id) {
		return nil, ErrNoVersion
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := os.ReadFile(filepath.Join(s.versionDir(name), id+".json"))
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrNoVersion
	}
	return data, err
}

// ReadRaw 读取配置文件的当前内容；文件不存在时返回 nil。
func (s *Store) ReadRaw(name string) ([]byte, error) {
	p, err := s.path(name)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := os.ReadFile(p)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	return data, err
}

// RestoreVersion 把配置回滚到某个历史版本。当前内容会先保存为新版本，因此回滚本身也可撤销。
func (s *Store) RestoreVersion(name, id string) error {
	data, err := s.ReadVersion(name, id)
	if err != nil {
		return err
	}
	return s.WriteRaw(name, data)
}
