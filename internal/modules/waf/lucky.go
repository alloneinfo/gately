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
func (m *Module) ImportLucky(doc luckyimport.Document, opts luckyimport.Options) (*luckyimport.Report, error) {
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
	m.mu.Lock()
	defer m.mu.Unlock()
	old := m.cfg.Instances
	existing := map[string]bool{}
	for _, in := range m.cfg.Instances {
		existing[in.Key] = true
	}
	var fresh []Instance
	for _, in := range out {
		if existing[in.Key] {
			rep.Warnings = append(rep.Warnings, luckyimport.Skipped(in.Name))
			continue
		}
		// WAF 实例只在被站点引用时生效，保留原启用状态
		fresh = append(fresh, in)
	}
	rep.Imported = len(fresh)
	if opts.DryRun {
		return rep, nil
	}
	m.cfg.Instances = append(m.cfg.Instances, fresh...)
	if err := m.saveLocked(); err != nil {
		m.cfg.Instances = old
		return nil, err
	}
	for _, in := range m.cfg.Instances {
		m.buildLocked(in.Key)
	}
	return rep, nil
}
