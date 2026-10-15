package portforward

import (
	"encoding/json"
	"fmt"

	"github.com/wangwanjie/gately/internal/luckyimport"
)

// luckyRule 对应 Lucky 端口转发规则中 Gately 关心的字段（lucky_portforward 配置）。
type luckyRule struct {
	Name              string
	Enable            bool
	ForwardTypes      []string
	ListenAddress     string
	ListenPorts       string
	TargetAddressList []string
	TargetPorts       string
	Options           struct {
		SingleProxyMaxTCPConnections int
		UDPSessionTimeout            int
		SafeMode                     string
		TCPListenTLS                 bool
		TCPRelayTLS                  bool
		TCPStreamEncryptionSource    bool
		TCPStreamEncryptionAccept    bool
		SinglePortSpeedLimit         bool
		RuleSpeedLimit               bool
		UDPPacketSourceEncryption    bool
		UDPPacketAcceptEncryption    bool
	}
}

// convertLuckyRules 把 Lucky 规则转换为 Gately 规则，并列出无法迁移的选项。
func convertLuckyRules(raw json.RawMessage) ([]Rule, []string, error) {
	var src []luckyRule
	if err := json.Unmarshal(raw, &src); err != nil {
		return nil, nil, err
	}
	var (
		out      []Rule
		warnings []string
	)
	for i, lr := range src {
		label := lr.Name
		if label == "" {
			label = fmt.Sprintf("第 %d 条(%s)", i+1, lr.ListenPorts)
		}
		rule := Rule{
			Name:            lr.Name,
			Enable:          lr.Enable,
			Protocols:       lr.ForwardTypes,
			ListenAddress:   lr.ListenAddress,
			ListenPorts:     lr.ListenPorts,
			TargetAddresses: lr.TargetAddressList,
			TargetPorts:     lr.TargetPorts,
			MaxConnections:  lr.Options.SingleProxyMaxTCPConnections,
		}
		// Lucky 的 UDPSessionTimeout 单位为毫秒（界面默认 30000），Gately 使用秒
		if lr.Options.UDPSessionTimeout > 0 {
			rule.UDPSessionTimeout = max(1, lr.Options.UDPSessionTimeout/1000)
		}
		o := lr.Options
		if o.TCPListenTLS || o.TCPRelayTLS || o.TCPStreamEncryptionSource || o.TCPStreamEncryptionAccept || o.UDPPacketSourceEncryption || o.UDPPacketAcceptEncryption {
			warnings = append(warnings, label+": TLS/流加密选项暂不支持，已按明文转发导入")
		}
		if o.SinglePortSpeedLimit || o.RuleSpeedLimit {
			warnings = append(warnings, label+": 限速选项暂不支持，已忽略")
		}
		if o.SafeMode == "whitelist" {
			warnings = append(warnings, label+": 原规则启用了白名单模式，IP 过滤将在后续版本支持，请注意暴露面")
		}
		if err := rule.Validate(); err != nil {
			warnings = append(warnings, label+": 跳过，"+err.Error())
			continue
		}
		// 导入后一律停用：避免与仍在运行的 Lucky 争用端口，或在测试机上意外开放端口
		rule.Enable = false
		out = append(out, rule)
	}
	if len(out) > 0 {
		warnings = append(warnings, "导入的规则默认停用：请先停止 Lucky 中对应的规则，再在 Gately 中启用")
	}
	return out, warnings, nil
}

// ImportLucky 实现 luckyimport.Importer。
func (m *Module) ImportLucky(doc luckyimport.Document, dryRun bool) (*luckyimport.Report, error) {
	raw, ok := doc["PortForwardsRuleList"]
	if !ok {
		return nil, nil
	}
	rules, warnings, err := convertLuckyRules(raw)
	if err != nil {
		return nil, err
	}
	var total []json.RawMessage
	_ = json.Unmarshal(raw, &total)
	rep := &luckyimport.Report{Module: ID, Title: m.Title(), Found: len(total), Warnings: warnings}
	if dryRun {
		rep.Imported = len(rules)
		return rep, nil
	}
	n, err := m.ImportRules(rules)
	if err != nil {
		return nil, err
	}
	rep.Imported = n
	return rep, nil
}
