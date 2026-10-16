package webdav

import (
	"encoding/json"

	"github.com/wangwanjie/gately/internal/luckyimport"
)

// ImportLucky 实现 luckyimport.Importer，迁移监听参数与用户（导入后保持停用）。
func (m *Module) ImportLucky(doc luckyimport.Document, opts luckyimport.Options) (*luckyimport.Report, error) {
	raw, ok := doc["WebDavConf"]
	if !ok {
		return nil, nil
	}
	var src struct {
		Enable     bool
		ListenIP   string
		ListenPort int
		TLSEnable  bool
		Users      []struct {
			UserName string
			Password string
			Path     string
			ReadOnly bool
		}
	}
	if err := json.Unmarshal(raw, &src); err != nil {
		return nil, err
	}
	s := Settings{Enable: src.Enable && opts.KeepEnabled, ListenIP: src.ListenIP, Port: src.ListenPort, TLS: src.TLSEnable}
	for _, u := range src.Users {
		s.Users = append(s.Users, User{Name: u.UserName, Password: u.Password, Root: u.Path, ReadOnly: u.ReadOnly})
	}
	rep := &luckyimport.Report{Module: ID, Title: m.Title(), Found: 1}
	if err := s.Validate(); err != nil {
		rep.Warnings = append(rep.Warnings, "跳过，"+err.Error())
		return rep, nil
	}
	rep.Imported = 1
	if !s.Enable {
		rep.Warnings = append(rep.Warnings, "WebDAV 设置已导入并保持停用")
	}
	if opts.DryRun {
		return rep, nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.store.Save(ID, s); err != nil {
		return nil, err
	}
	m.settings = s
	m.restartLocked()
	return rep, nil
}
