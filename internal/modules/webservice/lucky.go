package webservice

import (
	"encoding/json"
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/wangwanjie/gately/internal/crud"
	"github.com/wangwanjie/gately/internal/luckyimport"
)

// flexStrings 兼容 Lucky 中同一字段时而是字符串、时而是字符串数组的情况。
type flexStrings []string

func (f *flexStrings) UnmarshalJSON(b []byte) error {
	var arr []string
	if err := json.Unmarshal(b, &arr); err == nil {
		*f = arr
		return nil
	}
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	if s != "" {
		*f = []string{s}
	}
	return nil
}

// luckySite 对应 Lucky 反向代理子规则（ProxyList 元素）与默认规则。
type luckySite struct {
	Key                        string
	Enable                     bool
	Remark                     string
	Domains                    []string
	Locations                  []string
	WebServiceType             string
	LocationInsecureSkipVerify bool
	UseTargetHost              bool
	DisableKeepAlives          bool
	AddRemoteIPToHeader        bool
	AddProtoToHeader           bool
	EnableBasicAuth            bool
	BasicAuthUser              string
	BasicAuthPasswd            string
	BasicAuthUserList          flexStrings
	SafeIPMode                 string
	EnableAccessLog            bool
	EnableCrossDomain          bool
	CorazaWAFInstance          string
	CustomOutputText           string
	OtherParams                map[string]any
}

type luckyListener struct {
	RuleName      string
	Enable        bool
	ListenIP      string
	ListenPort    int
	Network       string
	EnableTLS     bool
	Http3         bool
	TLSMinVersion int
	DefaultProxy  luckySite
	ProxyList     []luckySite
}

// luckyUsers 解析用户列表。Lucky 某些版本会把整段 "user:pass" 文本按字符存成数组，这里先还原再拆分。
func luckyUsers(ls luckySite) []string {
	list := ls.BasicAuthUserList
	allChars := len(list) > 1
	for _, s := range list {
		if len([]rune(s)) != 1 {
			allChars = false
			break
		}
	}
	if allChars {
		list = []string{strings.Join(list, "")}
	}
	var out []string
	for _, item := range list {
		for _, u := range strings.FieldsFunc(item, func(r rune) bool { return r == '\n' || r == ',' }) {
			if u = strings.TrimSpace(u); strings.Contains(u, ":") {
				out = append(out, u)
			}
		}
	}
	if ls.BasicAuthUser != "" && ls.BasicAuthPasswd != "" {
		out = append(out, ls.BasicAuthUser+":"+ls.BasicAuthPasswd)
	}
	return out
}

// convertSite 把 Lucky 子规则转换为站点，返回不支持特性的说明。
func convertSite(ls luckySite) (Site, []string) {
	var notes []string
	s := Site{
		Name:               ls.Remark,
		Enable:             ls.Enable,
		Domains:            ls.Domains,
		Targets:            ls.Locations,
		InsecureSkipVerify: ls.LocationInsecureSkipVerify,
		UseTargetHost:      ls.UseTargetHost,
		DisableKeepAlives:  ls.DisableKeepAlives,
		ForwardedHeaders:   ls.AddRemoteIPToHeader || ls.AddProtoToHeader,
		AccessLog:          ls.EnableAccessLog,
		CORS:               ls.EnableCrossDomain,
		// 保留 Lucky 的 WAF 实例 key，WAF 模块导入时沿用相同 key
		WAF: ls.CorazaWAFInstance,
	}
	if v, ok := ls.OtherParams["ProxyProtocolV2"].(bool); ok {
		s.SendProxyProtocol = v
	}
	webAuth, _ := ls.OtherParams["WebAuth"].(bool)
	s.WebAuth = webAuth
	s.BasicAuth = ls.EnableBasicAuth
	if s.WebAuth || s.BasicAuth {
		s.Users = luckyUsers(ls)
		// 只依赖第三方登录、没有站点用户时无法等价迁移；为避免去掉认证后暴露站点，停用该站点
		if len(s.Users) == 0 {
			s.WebAuth, s.BasicAuth, s.Enable = false, false, false
			notes = append(notes, "认证停用")
		}
	}
	switch ls.WebServiceType {
	case "reverseproxy", "":
		s.Type = TypeProxy
	case "redirect", "url":
		s.Type = TypeRedirect
		if len(ls.Locations) > 0 {
			s.RedirectTo = ls.Locations[0]
		}
		s.KeepPath = true
	case "textoutput":
		s.Type = TypeText
		s.Text = ls.CustomOutputText
	default:
		notes = append(notes, fmt.Sprintf("站点类型 %q 暂不支持", ls.WebServiceType))
	}
	if ls.SafeIPMode == "whitelist" {
		notes = append(notes, "原规则使用全局 IP 白名单，迁移后未启用访问控制，请注意暴露面")
	}
	if oauth, _ := ls.OtherParams["AllowAllThirdUsers"].(bool); oauth {
		notes = append(notes, "第三方登录")
	}
	return s, notes
}

func convertLuckyListeners(raw json.RawMessage) ([]Listener, []string, int, error) {
	var src []luckyListener
	if err := json.Unmarshal(raw, &src); err != nil {
		return nil, nil, 0, err
	}
	var (
		out   []Listener
		warns []string
	)
	minTLS := map[int]string{0: "1.0", 1: "1.1", 2: "1.2", 3: "1.3"}
	for i, ll := range src {
		label := ll.RuleName
		if label == "" {
			label = fmt.Sprintf("第 %d 个监听规则", i+1)
		}
		l := Listener{
			Name:     ll.RuleName,
			Enable:   false,
			Network:  ll.Network,
			ListenIP: ll.ListenIP,
			Port:     ll.ListenPort,
			TLS:      ll.EnableTLS,
			MinTLS:   minTLS[ll.TLSMinVersion],
		}
		if ll.Http3 {
			warns = append(warns, label+": HTTP/3 暂不支持，已忽略")
		}
		// 汇总各站点的不支持特性，避免逐条重复告警
		featureSites := map[string][]string{}
		for j, ls := range ll.ProxyList {
			site, notes := convertSite(ls)
			name := site.Name
			if name == "" && len(site.Domains) > 0 {
				name = site.Domains[0]
			}
			if name == "" {
				name = fmt.Sprintf("站点 %d", j+1)
			}
			skip := false
			for _, n := range notes {
				if strings.HasPrefix(n, "站点类型") {
					warns = append(warns, label+"/"+name+": "+n+"，已跳过")
					skip = true
				} else {
					featureSites[n] = append(featureSites[n], name)
				}
			}
			if !skip {
				l.Sites = append(l.Sites, site)
			}
		}
		if def, notes := convertSite(ll.DefaultProxy); len(def.Targets) > 0 || def.Type != TypeProxy {
			// 默认站点没有单独的启用开关，认证无法迁移时保持停用
			def.Enable = !slices.Contains(notes, "认证停用")
			def.Domains = nil
			l.Default = def
			for _, n := range notes {
				featureSites[n] = append(featureSites[n], "默认站点")
			}
		}
		keys := make([]string, 0, len(featureSites))
		for k := range featureSites {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			msg := k
			switch k {
			case "第三方登录":
				msg = "第三方登录原本经过 Lucky 官方中转服务器，迁移后需在全局设置中配置自己的 GitHub/OIDC 应用并填写允许的身份；在此之前仅能使用站点用户登录"
			case "认证停用":
				msg = "这些站点只配置了 Lucky 中转的第三方登录、没有站点用户，为避免去掉认证后暴露，已停用（在全局设置中配置自己的 GitHub/OIDC 登录或添加用户后再启用）"
			}
			warns = append(warns, fmt.Sprintf("%s: %s：%s", label, msg, strings.Join(featureSites[k], "、")))
		}
		if err := l.Validate(); err != nil {
			warns = append(warns, label+": 跳过，"+err.Error())
			continue
		}
		out = append(out, l)
	}
	if len(out) > 0 {
		warns = append(warns, "站点引用的 WAF 实例需同时导入 lucky_coraza.lkcf；引用不存在的实例时不做检查（与 Lucky 行为一致）")
		warns = append(warns, "导入的监听规则默认停用：请先停止 Lucky 中对应的规则，再在 Gately 中启用；HTTPS 站点需要先在证书模块中启用对应证书")
	}
	return out, warns, len(src), nil
}

// ImportLucky 实现 luckyimport.Importer，同时识别 lucky_base 中的 PROXY 协议可信来源。
func (m *Module) ImportLucky(doc luckyimport.Document, dryRun bool) (*luckyimport.Report, error) {
	if raw, ok := doc["ProxyProtocolTrustedCIDRs"]; ok && !dryRun {
		var cidrs any
		_ = json.Unmarshal(raw, &cidrs)
		var list []string
		switch v := cidrs.(type) {
		case string:
			list = strings.FieldsFunc(v, func(r rune) bool { return r == '\n' || r == ',' })
		case []any:
			for _, x := range v {
				if s, ok := x.(string); ok {
					list = append(list, s)
				}
			}
		}
		if list = cleanList(list); len(list) > 0 {
			if _, err := parsePrefixes(list); err == nil {
				m.mu.Lock()
				m.cfg.Settings.TrustedProxies = list
				_ = m.saveLocked()
				m.mu.Unlock()
			}
		}
	}
	raw, ok := doc["ReverseProxyRuleList"]
	if !ok {
		return nil, nil
	}
	listeners, warns, found, err := convertLuckyListeners(raw)
	if err != nil {
		return nil, err
	}
	rep := &luckyimport.Report{Module: ID, Title: m.Title(), Found: found, Imported: len(listeners), Warnings: warns}
	if dryRun {
		return rep, nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	old := m.cfg.Listeners
	for _, l := range listeners {
		l.Key = crud.NewKey()
		m.cfg.Listeners = append(m.cfg.Listeners, l)
	}
	if err := m.saveLocked(); err != nil {
		m.cfg.Listeners = old
		return nil, err
	}
	return rep, nil
}
