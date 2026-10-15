package waf

import (
	"encoding/json"
	"fmt"

	"github.com/wangwanjie/gately/internal/luckyimport"
)

type luckyCoraza struct {
	Key                    string
	Name                   string
	Enable                 bool
	InboundScoreThreshold  int
	OutboundScoreThreshold int
	CorazaWAFConfigList    []struct {
		Enable bool
		Remark string
		Type   string
		Param  string
	}
}

// ImportLucky 实现 luckyimport.Importer。沿用 Lucky 的实例 key，使 Web 服务站点的引用保持有效。
func (m *Module) ImportLucky(doc luckyimport.Document, dryRun bool) (*luckyimport.Report, error) {
	raw, ok := doc["CorazaList"]
	if !ok {
		return nil, nil
	}
	var src []luckyCoraza
	if err := json.Unmarshal(raw, &src); err != nil {
		return nil, err
	}
	rep := &luckyimport.Report{Module: ID, Title: m.Title(), Found: len(src)}
	var out []Instance
	for _, lc := range src {
		in := Instance{Key: lc.Key, Name: lc.Name, Enable: lc.Enable, InboundThreshold: lc.InboundScoreThreshold, OutboundThreshold: lc.OutboundScoreThreshold}
		for _, c := range lc.CorazaWAFConfigList {
			rs := RuleSet{Enable: c.Enable, Remark: c.Remark, Param: c.Param, Type: RuleCustom}
			if c.Type == "OWASPCoreRuleset" {
				rs.Type = RuleCRS
			}
			in.RuleSets = append(in.RuleSets, rs)
		}
		if err := in.Validate(); err != nil {
			rep.Warnings = append(rep.Warnings, fmt.Sprintf("%s: 跳过，%v", lc.Name, err))
			continue
		}
		out = append(out, in)
	}
	rep.Imported = len(out)
	if dryRun {
		return rep, nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	old := m.cfg.Instances
	existing := map[string]bool{}
	for _, in := range m.cfg.Instances {
		existing[in.Key] = true
	}
	for _, in := range out {
		if existing[in.Key] {
			rep.Warnings = append(rep.Warnings, in.Name+": 已存在相同 key 的实例，跳过")
			rep.Imported--
			continue
		}
		m.cfg.Instances = append(m.cfg.Instances, in)
	}
	if err := m.saveLocked(); err != nil {
		m.cfg.Instances = old
		return nil, err
	}
	for _, in := range m.cfg.Instances {
		m.buildLocked(in.Key)
	}
	return rep, nil
}
