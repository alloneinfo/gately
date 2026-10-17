package app

import (
	"encoding/json"
	"net/http"
	"reflect"
	"regexp"
	"sort"
	"strings"

	"github.com/wangwanjie/gately/internal/apitoken"
	"github.com/wangwanjie/gately/internal/httpx"
	"github.com/wangwanjie/gately/internal/version"
)

// 官网地址，写入 OpenAPI 的 externalDocs
const websiteURL = "https://gately.cn"

// endpoint 是文档中的一个接口。
type endpoint struct {
	Method  string
	Path    string
	Tag     string
	Summary string
	Desc    string
	// Scope 为所需权限；Denied 表示令牌不可访问
	Scope string
	// Req / Resp 为 schema（可能是 $ref），为 nil 表示无
	Req     map[string]any
	ReqFree bool
	Resp    map[string]any
	Query   []string
}

// docModel 是文档的中间表示，OpenAPI JSON 与 Markdown 都从它渲染。
type docModel struct {
	Endpoints []endpoint
	Schemas   map[string]map[string]any
	Tags      []string
	Domains   []scopeDomain
}

var pathParamRe = regexp.MustCompile(`\{([^}/]+)\}`)

// buildDoc 根据已登记的路由生成文档模型。
func (a *App) buildDoc() docModel {
	// 路由表来自处理器构建，重复构建只会刷新登记结果
	if len(a.routes) == 0 {
		a.Handler()
	}
	titles := map[string]string{}
	for _, e := range a.modules {
		titles[e.mod.ID()] = e.mod.Title()
	}
	schemas := newSchemaSet()
	var eps []endpoint
	for _, r := range a.routes {
		key := r.Method + " " + r.Path
		ep := endpoint{Method: r.Method, Path: r.Path, Scope: apitoken.Required(r.Method, r.Path)}
		ep.Tag = docTag(r, titles)
		// 日志接口支持 since 增量拉取
		if r.Method == http.MethodGet && (r.Path == "/api/logs" || (r.Module != "" && r.Path == "/api/"+r.Module+"/logs")) {
			ep.Query = []string{"since"}
		}
		d, hasDoc := a.routeDocs[key]
		if hasDoc {
			if d.Req != nil {
				ep.Req = schemas.of(reflect.TypeOf(d.Req))
			}
			if d.Resp != nil {
				ep.Resp = schemas.of(reflect.TypeOf(d.Resp))
			}
		}
		if c, ok := apiCatalog[key]; ok {
			ep.Summary, ep.Desc, ep.ReqFree = c.Summary, c.Desc, c.Body
		} else if c, ok := apiCatalog[genericKey(r)]; ok {
			ep.Summary, ep.Desc, ep.ReqFree = c.Summary, c.Desc, c.Body
		} else if hasDoc {
			ep.Summary = crudSummary(r, d.Summary)
		}
		if ep.Req != nil {
			ep.ReqFree = false
		}
		eps = append(eps, ep)
	}
	sort.SliceStable(eps, func(i, j int) bool {
		if eps[i].Tag != eps[j].Tag {
			return tagRank(eps[i].Tag) < tagRank(eps[j].Tag)
		}
		return false
	})
	var tags []string
	seen := map[string]bool{}
	for _, e := range eps {
		if !seen[e.Tag] {
			seen[e.Tag] = true
			tags = append(tags, e.Tag)
		}
	}
	return docModel{Endpoints: eps, Schemas: schemas.defs, Tags: tags, Domains: a.scopeDomains()}
}

// genericKey 把模块日志等通用接口映射到目录中的通用条目。
func genericKey(r routeInfo) string {
	if r.Module != "" && strings.HasSuffix(r.Path, "/"+"logs") && r.Path == "/api/"+r.Module+"/logs" {
		return r.Method + " /api/{module}/logs"
	}
	return ""
}

func docTag(r routeInfo, titles map[string]string) string {
	if r.Module != "" {
		return titles[r.Module]
	}
	switch {
	case strings.HasPrefix(r.Path, "/api/tokens"):
		return "API 令牌"
	case strings.HasPrefix(r.Path, "/api/account"):
		return "账号"
	case r.Path == "/api/import/lucky":
		return "配置导入"
	}
	return "系统"
}

// tagRank 让系统类接口排在功能模块之后，功能模块保持注册顺序。
func tagRank(tag string) int {
	switch tag {
	case "系统":
		return 1000
	case "配置导入":
		return 1001
	case "API 令牌":
		return 1002
	case "账号":
		return 1003
	}
	return 0
}

// crudSummary 由通用动作和集合名称拼出中文摘要，例如“获取 DDNS 任务列表”。
func crudSummary(r routeInfo, verb string) string {
	// 模块自定义的完整摘要原样使用，只有通用 CRUD 动作才拼接集合名称
	switch verb {
	case "列表", "新增", "修改", "删除", "调整排序", "启用或停用":
	default:
		return verb
	}
	rest := strings.TrimPrefix(r.Path, "/api/"+r.Module)
	coll := strings.TrimPrefix(rest, "/")
	coll, _, _ = strings.Cut(coll, "/")
	coll = strings.TrimSuffix(coll, "-order")
	noun := crudNouns[r.Module+"/"+coll]
	if noun == "" {
		noun = "条目"
	}
	switch verb {
	case "列表":
		return "获取" + noun + "列表"
	case "调整排序":
		return "调整" + noun + "排序"
	case "启用或停用":
		return "启用或停用" + noun
	}
	return verb + noun
}

func (m docModel) buildOpenAPI() map[string]any {
	paths := map[string]any{}
	for _, e := range m.Endpoints {
		item, _ := paths[e.Path].(map[string]any)
		if item == nil {
			item = map[string]any{}
			paths[e.Path] = item
		}
		item[strings.ToLower(e.Method)] = m.operation(e)
	}
	var tags []map[string]any
	for _, t := range m.Tags {
		tags = append(tags, map[string]any{"name": t})
	}
	schemas := map[string]any{"Error": map[string]any{
		"type":       "object",
		"properties": map[string]any{"error": map[string]any{"type": "string", "description": "错误说明"}},
	}}
	for k, v := range m.Schemas {
		schemas[k] = v
	}
	var scopes []string
	for _, d := range m.Domains {
		scopes = append(scopes, d.ID)
	}
	return map[string]any{
		"openapi": "3.0.3",
		"info": map[string]any{
			"title":       "Gately 管理 API",
			"version":     version.Version,
			"description": apiIntro,
		},
		"externalDocs": map[string]any{"description": "Gately 官网", "url": websiteURL},
		"servers": []map[string]any{{
			"url":         "http://localhost:16680",
			"description": "默认管理地址，请替换为你的 Gately 地址",
		}},
		"tags":  tags,
		"paths": paths,
		"components": map[string]any{
			"securitySchemes": map[string]any{
				"bearerAuth": map[string]any{
					"type": "http", "scheme": "bearer",
					"description": "在“API 令牌”页创建令牌，请求头携带 Authorization: Bearer gtl_xxxx",
				},
			},
			"schemas": schemas,
		},
		"x-scope-domains": scopes,
	}
}

func (m docModel) operation(e endpoint) map[string]any {
	op := map[string]any{"tags": []string{e.Tag}, "summary": e.Summary}
	desc := e.Desc
	switch e.Scope {
	case apitoken.Denied:
		desc += "\n\n**仅登录会话可用，API 令牌无法访问此接口。**"
	case apitoken.Any:
		desc += "\n\n任何有效的 API 令牌均可访问，无需特定权限。"
	default:
		desc += "\n\n所需权限：`" + e.Scope + "`。"
	}
	op["description"] = strings.TrimSpace(desc)
	op["x-required-scope"] = e.Scope
	op["x-token-allowed"] = e.Scope != apitoken.Denied
	if e.Scope != apitoken.Denied {
		op["security"] = []map[string][]string{{"bearerAuth": {}}}
	}
	var params []map[string]any
	for _, mt := range pathParamRe.FindAllStringSubmatch(e.Path, -1) {
		params = append(params, map[string]any{"name": mt[1], "in": "path", "required": true, "schema": map[string]any{"type": "string"}})
	}
	for _, q := range e.Query {
		params = append(params, map[string]any{"name": q, "in": "query", "schema": map[string]any{"type": "integer"},
			"description": "增量拉取：只返回序号大于该值的日志"})
	}
	if len(params) > 0 {
		op["parameters"] = params
	}
	if e.Req != nil || e.ReqFree {
		body := e.Req
		if body == nil {
			body = map[string]any{"type": "object"}
		}
		op["requestBody"] = map[string]any{"content": map[string]any{"application/json": map[string]any{"schema": body}}}
	}
	ok := map[string]any{"description": "成功"}
	if e.Resp != nil {
		ok["content"] = map[string]any{"application/json": map[string]any{"schema": e.Resp}}
	}
	errResp := func(desc string) map[string]any {
		return map[string]any{"description": desc, "content": map[string]any{"application/json": map[string]any{
			"schema": map[string]any{"$ref": "#/components/schemas/Error"}}}}
	}
	responses := map[string]any{"200": ok, "400": errResp("请求参数错误"), "401": errResp("令牌无效、已过期或未登录")}
	if e.Scope != apitoken.Any {
		responses["403"] = errResp("令牌缺少所需权限")
	}
	op["responses"] = responses
	return op
}

func (a *App) handleOpenAPI(w http.ResponseWriter, r *http.Request) {
	httpx.OK(w, a.buildDoc().buildOpenAPI())
}

// APIDocs 返回 OpenAPI JSON 与 Markdown 文档，供命令行导出静态文件。
func (a *App) APIDocs() (openapi []byte, markdown []byte, err error) {
	m := a.buildDoc()
	openapi, err = json.MarshalIndent(m.buildOpenAPI(), "", "  ")
	if err != nil {
		return nil, nil, err
	}
	return append(openapi, '\n'), []byte(m.markdown()), nil
}
