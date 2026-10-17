package storage

import (
	"bytes"
	"strings"
)

// MountInfo 是供其他模块（如备份）选择挂载点的简要信息。
type MountInfo struct {
	Key      string `json:"key"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	Enable   bool   `json:"enable"`
	ReadOnly bool   `json:"readOnly"`
}

// MountList 返回全部挂载点。
func (m *Module) MountList() []MountInfo {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]MountInfo, 0, len(m.cfg.Mounts))
	for _, mt := range m.cfg.Mounts {
		out = append(out, MountInfo{Key: mt.Key, Name: mt.Name, Type: mt.Type, Enable: mt.Enable, ReadOnly: mt.ReadOnly})
	}
	return out
}

// PutFile 把 data 写入挂载点 dir 目录下的 name 文件，目录不存在时逐级创建。
func (m *Module) PutFile(mount, dir, name string, data []byte) error {
	b, err := m.open(mount, true)
	if err != nil {
		return err
	}
	dir = cleanPath(dir)
	// 逐级创建目录；已存在时的错误忽略，真正的失败会在写入时暴露
	cur := ""
	for _, seg := range strings.Split(strings.Trim(dir, "/"), "/") {
		if seg == "" {
			continue
		}
		cur += "/" + seg
		_ = b.Mkdir(cur)
	}
	return b.Put(cleanPath(dir+"/"+name), bytes.NewReader(data), int64(len(data)))
}

// ListNames 返回挂载点 dir 目录下的文件名（不含子目录）。目录不存在时返回空列表。
func (m *Module) ListNames(mount, dir string) ([]string, error) {
	b, err := m.open(mount, false)
	if err != nil {
		return nil, err
	}
	entries, err := b.List(cleanPath(dir))
	if err != nil {
		if err == errNotFound {
			return nil, nil
		}
		return nil, err
	}
	var out []string
	for _, e := range entries {
		if !e.IsDir {
			out = append(out, e.Name)
		}
	}
	return out, nil
}

// RemoveFile 删除挂载点内的文件。
func (m *Module) RemoveFile(mount, p string) error {
	b, err := m.open(mount, true)
	if err != nil {
		return err
	}
	return b.Remove(cleanPath(p))
}
