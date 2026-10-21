package i18n

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
)

// RequestLang 返回请求期望的语言：优先 Accept-Language，没有时使用服务端语言。
func RequestLang(r *http.Request) string {
	if l := FromAcceptLanguage(r.Header.Get("Accept-Language")); l != "" {
		return l
	}
	return ServerLang()
}

// TrDeep 递归翻译 JSON 值（map / slice / string）中的字符串；minRunes 为参与翻译的最短字符串长度，
// 用来避免把用户自填的短文本（如备注“测试”）误翻成其他语言。
func TrDeep(lang string, v any, minRunes int) any {
	switch x := v.(type) {
	case string:
		if len([]rune(x)) < minRunes {
			return x
		}
		return Tr(lang, x)
	case []any:
		for i := range x {
			x[i] = TrDeep(lang, x[i], minRunes)
		}
		return x
	case map[string]any:
		for k, e := range x {
			x[k] = TrDeep(lang, e, minRunes)
		}
		return x
	}
	return v
}

// fullTranslatePaths 返回 true 的接口数据全部由程序生成（状态、日志、文档等），所有字符串都翻译。
func fullTranslate(path string) bool {
	switch {
	case path == "/api/status", path == "/api/logs", path == "/api/openapi.json",
		strings.HasSuffix(path, "/logs"), strings.HasPrefix(path, "/api/events"),
		strings.HasSuffix(path, "/history"), strings.Contains(path, "/tasks"),
		strings.HasPrefix(path, "/api/tokens/scopes"), strings.HasSuffix(path, "/providers"):
		return true
	}
	return false
}

// skipTranslate 返回 true 的接口内容来自用户文件或外部程序，不做翻译。
func skipTranslate(path string) bool {
	for _, p := range []string{"/files", "/content", "/download", "/backup", "/terminal", "/sftp", "/stats/", "/containers/"} {
		if strings.Contains(path, p) && !strings.HasSuffix(path, "/logs") {
			return true
		}
	}
	return false
}

// Middleware 按请求语言翻译 JSON 响应：错误响应（状态码 >= 400）中的说明，以及程序生成的状态、日志、文档数据。
// 语言为简体中文时不做任何处理。
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lang := RequestLang(r)
		if lang == SourceLang || !strings.HasPrefix(r.URL.Path, "/api/") {
			next.ServeHTTP(w, r)
			return
		}
		tw := &transWriter{ResponseWriter: w, lang: lang, path: r.URL.Path, get: r.Method == http.MethodGet}
		next.ServeHTTP(tw, r)
		tw.finish()
	})
}

type transWriter struct {
	http.ResponseWriter
	lang   string
	path   string
	get    bool
	status int
	buf    bytes.Buffer
	// capture 为 true 时响应体被缓存，结束后翻译再输出
	capture bool
	decided bool
}

func (t *transWriter) decide(status int) {
	if t.decided {
		return
	}
	t.decided = true
	t.status = status
	ct := t.Header().Get("Content-Type")
	isJSON := strings.Contains(ct, "json")
	isText := strings.HasPrefix(ct, "text/plain")
	switch {
	case status >= 400 && (isJSON || isText):
		t.capture = true
	case status == http.StatusOK && isJSON && t.get && !skipTranslate(t.path):
		t.capture = true
	}
}

func (t *transWriter) WriteHeader(status int) {
	t.decide(status)
	if t.capture {
		return
	}
	t.ResponseWriter.WriteHeader(status)
}

func (t *transWriter) Write(p []byte) (int, error) {
	if !t.decided {
		t.decide(http.StatusOK)
	}
	if t.capture {
		return t.buf.Write(p)
	}
	return t.ResponseWriter.Write(p)
}

func (t *transWriter) Flush() {
	if t.capture {
		return
	}
	if f, ok := t.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (t *transWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h, ok := t.ResponseWriter.(http.Hijacker); ok {
		return h.Hijack()
	}
	return nil, nil, errors.New("hijack not supported")
}

func (t *transWriter) Unwrap() http.ResponseWriter { return t.ResponseWriter }

func (t *transWriter) ReadFrom(r io.Reader) (int64, error) {
	return io.Copy(struct{ io.Writer }{t}, r)
}

// finish 翻译缓存的响应体并输出。
func (t *transWriter) finish() {
	if !t.capture {
		return
	}
	body := t.buf.Bytes()
	ct := t.Header().Get("Content-Type")
	if strings.Contains(ct, "json") {
		var v any
		if err := json.Unmarshal(body, &v); err == nil {
			min := 4
			if t.status >= 400 || fullTranslate(t.path) {
				min = 0
			}
			v = TrDeep(t.lang, v, min)
			var out bytes.Buffer
			enc := json.NewEncoder(&out)
			if err := enc.Encode(v); err == nil {
				body = out.Bytes()
			}
		}
	} else {
		body = []byte(Tr(t.lang, strings.TrimRight(string(body), "\n")) + "\n")
	}
	t.Header().Del("Content-Length")
	t.ResponseWriter.WriteHeader(t.status)
	_, _ = t.ResponseWriter.Write(body)
}

// Localize 把程序生成的结果（如导入预览、部署结果）转为按请求语言翻译后的 JSON 值。
// 中间件只翻译 GET 响应与错误响应，POST 接口返回的说明文字需要用它显式翻译。
func Localize(r *http.Request, v any) any {
	lang := RequestLang(r)
	if lang == SourceLang {
		return v
	}
	b, err := json.Marshal(v)
	if err != nil {
		return v
	}
	var x any
	if err := json.Unmarshal(b, &x); err != nil {
		return v
	}
	return TrDeep(lang, x, 0)
}
