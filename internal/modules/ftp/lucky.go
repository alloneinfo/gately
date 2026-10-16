package ftp

import (
	"encoding/json"

	"github.com/wangwanjie/gately/internal/luckyimport"
)

// ImportLucky 实现 luckyimport.Importer，迁移监听与被动端口等服务参数（导入后保持停用）。
// Lucky 配置中的用户列表结构未在样本中确认，不做猜测性导入，需在界面中重新添加用户。
func (m *Module) ImportLucky(doc luckyimport.Document, opts luckyimport.Options) (*luckyimport.Report, error) {
	raw, ok := doc["Conf"]
	if !ok {
		return nil, nil
	}
	var src struct {
		Port              int
		PassivePortStart  int
		PassivePortEnd    int
		TLSRequired       int
		IdleTimeout       int
		ConnectionTimeout int
		DisableActiveMode bool
		Users             json.RawMessage
	}
	if err := json.Unmarshal(raw, &src); err != nil {
		return nil, err
	}
	// 其他模块的配置也可能带有 Conf 键，缺少 FTP 特有字段时视为非 FTP 配置
	var probe map[string]json.RawMessage
	if err := json.Unmarshal(raw, &probe); err != nil {
		return nil, err
	}
	if _, has := probe["PassivePortStart"]; !has {
		return nil, nil
	}
	s := Settings{
		Port:              src.Port,
		PassiveStart:      src.PassivePortStart,
		PassiveEnd:        src.PassivePortEnd,
		TLSMode:           src.TLSRequired,
		IdleTimeout:       src.IdleTimeout,
		ConnectionTimeout: src.ConnectionTimeout,
		DisableActiveMode: src.DisableActiveMode,
	}
	rep := &luckyimport.Report{Module: ID, Title: m.Title(), Found: 1}
	if err := s.Validate(); err != nil {
		rep.Warnings = append(rep.Warnings, "跳过，"+err.Error())
		return rep, nil
	}
	rep.Imported = 1
	rep.Warnings = append(rep.Warnings, "FTP 服务参数已导入并保持停用，用户需在界面中重新添加")
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
