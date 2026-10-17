package webservice

import (
	"html/template"
	"io"
	"net/http"
	"sync"

	"github.com/wangwanjie/gately/internal/i18n"
)

// 站点访客看到的登录页与错误页按访客的 Accept-Language 本地化，模板按语言解析后缓存。
var localizedTmpls sync.Map

// visitorLang 返回访客偏好的语言，请求没有语言偏好时使用服务端语言。
func visitorLang(r *http.Request) string {
	return i18n.RequestLang(r)
}

// executeLocalized 用目标语言的模板渲染；译文模板解析失败时退回中文原版，保证页面始终可用。
func executeLocalized(w io.Writer, name, src, lang string, data any) error {
	key := name + "|" + lang
	if v, ok := localizedTmpls.Load(key); ok {
		return v.(*template.Template).Execute(w, data)
	}
	t, err := template.New(name).Parse(i18n.Tr(lang, src))
	if err != nil {
		t = template.Must(template.New(name).Parse(src))
	}
	localizedTmpls.Store(key, t)
	return t.Execute(w, data)
}
