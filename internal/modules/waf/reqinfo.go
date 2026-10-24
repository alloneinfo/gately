package waf

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"net/netip"
	"strconv"
	"strings"
	"sync"

	"github.com/corazawaf/coraza/v3"
	"github.com/corazawaf/coraza/v3/experimental"
	"github.com/corazawaf/coraza/v3/types"

	"github.com/wangwanjie/gately/internal/logbuf"
)

// Coraza 的规则命中回调只拿得到事务 ID，拿不到请求本身（命中信息里的 hostname 也是空的）。
// 为了让命中日志带上域名等字段，Middleware 为每个请求生成事务 ID 并登记请求信息，
// 包装后的 WAF 在创建事务时使用这个 ID，回调里再按 ID 查回请求信息。

// reqInfo 是一次请求在命中日志中需要的信息。
type reqInfo struct {
	id   string
	host string
	ip   string
}

type ctxKeyReqInfo struct{}

// pending 保存处理中的请求：事务 ID -> *reqInfo，请求结束后删除。
var pending sync.Map

// trackRequest 登记请求信息并放入上下文，返回登记后的请求与结束时调用的清理函数。
func trackRequest(r *http.Request) (*http.Request, func()) {
	var b [12]byte
	_, _ = rand.Read(b[:])
	info := &reqInfo{id: hex.EncodeToString(b[:]), host: hostOnly(r.Host)}
	if ap, err := netip.ParseAddrPort(r.RemoteAddr); err == nil {
		info.ip = ap.Addr().Unmap().String()
	}
	pending.Store(info.id, info)
	return r.WithContext(context.WithValue(r.Context(), ctxKeyReqInfo{}, info)), func() { pending.Delete(info.id) }
}

// matchFields 返回规则命中日志的结构化字段。
func matchFields(r types.MatchedRule) logbuf.Fields {
	f := logbuf.Fields{logbuf.FieldIP: r.ClientIPAddress(), logbuf.FieldPath: r.URI()}
	if id := r.Rule().ID(); id > 0 {
		f[logbuf.FieldRule] = strconv.Itoa(id)
	}
	if v, ok := pending.Load(r.TransactionID()); ok {
		info := v.(*reqInfo)
		f[logbuf.FieldHost] = info.host
		if f[logbuf.FieldIP] == "" {
			f[logbuf.FieldIP] = info.ip
		}
	}
	return f
}

// trackedWAF 包装 coraza.WAF：创建事务时使用 Middleware 登记的事务 ID。
type trackedWAF struct {
	coraza.WAF
}

// NewTransactionWithOptions 实现 experimental.WAFWithOptions，txhttp.WrapHandler 会优先调用它并传入请求上下文。
func (t trackedWAF) NewTransactionWithOptions(o experimental.Options) types.Transaction {
	if info, ok := o.Context.Value(ctxKeyReqInfo{}).(*reqInfo); ok && o.ID == "" {
		o.ID = info.id
	}
	if w, ok := t.WAF.(experimental.WAFWithOptions); ok {
		return w.NewTransactionWithOptions(o)
	}
	return t.WAF.NewTransaction()
}

// hostOnly 去掉 Host 中的端口并转小写，与 Web 服务的写法一致。
func hostOnly(h string) string {
	h = strings.ToLower(h)
	if i := strings.LastIndexByte(h, ':'); i > 0 && !strings.HasSuffix(h, "]") {
		return strings.Trim(h[:i], "[]")
	}
	return strings.Trim(h, "[]")
}
