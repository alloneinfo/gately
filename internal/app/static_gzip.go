package app

import (
	"io"
	"mime"
	"net/http"
	"path"
	"strings"
	"time"
)

// servePrecompressed 在客户端接受 gzip 且构建时生成了同名 .gz 文件时直接返回压缩版本，省去每次请求的实时压缩。
// 返回 false 表示没有可用的预压缩文件，由调用方按原文件处理。
func (a *App) servePrecompressed(w http.ResponseWriter, r *http.Request, name string) bool {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		return false
	}
	// 范围请求按原文件处理，避免压缩内容与 Range 偏移不一致
	if r.Header.Get("Range") != "" || !acceptsGzip(r.Header.Get("Accept-Encoding")) {
		return false
	}
	f, err := a.opts.Web.Open(name + ".gz")
	if err != nil {
		return false
	}
	defer f.Close()
	rs, ok := f.(io.ReadSeeker)
	if !ok {
		return false
	}
	h := w.Header()
	// Content-Type 按原文件扩展名设置，否则 ServeContent 会按 .gz 识别
	ct := mime.TypeByExtension(path.Ext(name))
	if ct == "" {
		ct = "application/octet-stream"
	}
	h.Set("Content-Type", ct)
	h.Set("Content-Encoding", "gzip")
	h.Add("Vary", "Accept-Encoding")
	// 内嵌文件没有修改时间，传零值即不输出 Last-Modified
	http.ServeContent(w, r, name, time.Time{}, rs)
	return true
}

// acceptsGzip 判断 Accept-Encoding 是否接受 gzip（含 *），q=0 视为拒绝。
func acceptsGzip(accept string) bool {
	for _, part := range strings.Split(accept, ",") {
		enc, params, _ := strings.Cut(strings.TrimSpace(part), ";")
		enc = strings.ToLower(strings.TrimSpace(enc))
		if enc != "gzip" && enc != "*" {
			continue
		}
		q := strings.ReplaceAll(strings.TrimSpace(params), " ", "")
		if q == "q=0" || q == "q=0.0" || q == "q=0.00" || q == "q=0.000" {
			return false
		}
		return true
	}
	return false
}
