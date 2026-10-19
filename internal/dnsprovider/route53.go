package dnsprovider

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

// Amazon Route 53。ID 为 Access Key ID，Secret 为 Secret Access Key；可选填托管区域 ID，留空按域名查找。
// 请求使用 AWS Signature V4 签名（服务 route53，区域固定 us-east-1）。
func init() {
	register(Meta{
		Name:        "route53",
		Title:       "Amazon Route 53",
		IDLabel:     "Access Key ID",
		SecretLabel: "Secret Access Key",
		Extra: []Field{
			{Key: "hostedZoneId", Label: "托管区域 ID", Placeholder: "可选，如 Z0123456789ABC；留空按域名自动查找"},
		},
		DocURL:       "https://console.aws.amazon.com/iam/home#/security_credentials",
		SupportsACME: true,
	}, func(c Credentials, hc *http.Client) (Provider, error) {
		if err := requireCreds(c, true, true); err != nil {
			return nil, err
		}
		return &route53{ak: c.ID, sk: c.Secret, zoneID: strings.TrimPrefix(strings.TrimSpace(c.Extra["hostedZoneId"]), "/hostedzone/"), hc: hc, endpoint: "https://route53.amazonaws.com"}, nil
	})
}

type route53 struct {
	ak, sk, zoneID string
	hc             *http.Client
	endpoint       string
}

const r53Version = "/2013-04-01"

// signV4 计算 AWS Signature V4 的 Authorization 头。headers 的键须为小写，且包含 host 与 x-amz-date。
func signV4(method, path, query string, headers map[string]string, payload []byte, ak, sk, region, service string, t time.Time) string {
	names := make([]string, 0, len(headers))
	for k := range headers {
		names = append(names, k)
	}
	sort.Strings(names)
	var canonHeaders strings.Builder
	for _, k := range names {
		canonHeaders.WriteString(k + ":" + strings.TrimSpace(headers[k]) + "\n")
	}
	signed := strings.Join(names, ";")
	sum := sha256.Sum256(payload)
	canonical := strings.Join([]string{method, path, query, canonHeaders.String(), signed, hex.EncodeToString(sum[:])}, "\n")

	date := t.UTC().Format("20060102")
	scope := date + "/" + region + "/" + service + "/aws4_request"
	csum := sha256.Sum256([]byte(canonical))
	toSign := "AWS4-HMAC-SHA256\n" + t.UTC().Format("20060102T150405Z") + "\n" + scope + "\n" + hex.EncodeToString(csum[:])

	mac := func(key []byte, msg string) []byte {
		h := hmac.New(sha256.New, key)
		h.Write([]byte(msg))
		return h.Sum(nil)
	}
	key := mac([]byte("AWS4"+sk), date)
	key = mac(key, region)
	key = mac(key, service)
	key = mac(key, "aws4_request")
	sig := hex.EncodeToString(mac(key, toSign))
	return fmt.Sprintf("AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s", ak, scope, signed, sig)
}

// canonicalQuery 按 SigV4 规则生成规范查询串：参数名排序，名和值都按 RFC 3986 编码。
func canonicalQuery(q url.Values) string {
	keys := make([]string, 0, len(q))
	for k := range q {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var parts []string
	for _, k := range keys {
		for _, v := range q[k] {
			parts = append(parts, awsEscape(k)+"="+awsEscape(v))
		}
	}
	return strings.Join(parts, "&")
}

func awsEscape(s string) string {
	return strings.ReplaceAll(url.QueryEscape(s), "+", "%20")
}

type r53Error struct {
	Code    string `xml:"Error>Code"`
	Message string `xml:"Error>Message"`
}

func (r *route53) call(ctx context.Context, method, path string, q url.Values, body []byte, out any) error {
	u, err := url.Parse(r.endpoint)
	if err != nil {
		return err
	}
	cq := canonicalQuery(q)
	full := r.endpoint + path
	if cq != "" {
		full += "?" + cq
	}
	req, err := ctxReq(ctx, method, full, bytes.NewReader(body))
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	sum := sha256.Sum256(body)
	hdr := map[string]string{
		"host":                 u.Host,
		"x-amz-content-sha256": hex.EncodeToString(sum[:]),
		"x-amz-date":           now.Format("20060102T150405Z"),
	}
	req.Header.Set("X-Amz-Date", hdr["x-amz-date"])
	req.Header.Set("X-Amz-Content-Sha256", hdr["x-amz-content-sha256"])
	req.Header.Set("Authorization", signV4(method, path, cq, hdr, body, r.ak, r.sk, "us-east-1", "route53", now))
	if len(body) > 0 {
		req.Header.Set("Content-Type", "text/xml")
	}
	resp, err := r.hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var e r53Error
		if xml.Unmarshal(raw, &e) == nil && e.Code != "" {
			return fmt.Errorf("Route 53: %s: %s", e.Code, e.Message)
		}
		return fmt.Errorf("Route 53: %w", &HTTPError{Status: resp.StatusCode, Body: string(raw)})
	}
	if out != nil {
		if err := xml.Unmarshal(raw, out); err != nil {
			return fmt.Errorf("Route 53: 解析响应失败: %w", err)
		}
	}
	return nil
}

func (r *route53) hostedZone(ctx context.Context, zone string) (string, error) {
	if r.zoneID != "" {
		return r.zoneID, nil
	}
	var out struct {
		Zones []struct {
			ID   string `xml:"Id"`
			Name string `xml:"Name"`
		} `xml:"HostedZones>HostedZone"`
	}
	q := url.Values{"dnsname": {zone + "."}, "maxitems": {"5"}}
	if err := r.call(ctx, http.MethodGet, r53Version+"/hostedzonesbyname", q, nil, &out); err != nil {
		return "", err
	}
	for _, z := range out.Zones {
		if strings.EqualFold(strings.TrimSuffix(z.Name, "."), zone) {
			return strings.TrimPrefix(z.ID, "/hostedzone/"), nil
		}
	}
	return "", fmt.Errorf("Route 53 账号下未找到域名 %s", zone)
}

type r53Set struct {
	Name   string   `xml:"Name"`
	Type   string   `xml:"Type"`
	TTL    int      `xml:"TTL"`
	Values []string `xml:"ResourceRecords>ResourceRecord>Value"`
}

// getSet 读取同名同类型的记录集，不存在时返回 nil。
func (r *route53) getSet(ctx context.Context, zid, fqdn, typ string) (*r53Set, error) {
	var out struct {
		Sets []r53Set `xml:"ResourceRecordSets>ResourceRecordSet"`
	}
	q := url.Values{"name": {fqdn + "."}, "type": {typ}, "maxitems": {"1"}}
	if err := r.call(ctx, http.MethodGet, r53Version+"/hostedzone/"+zid+"/rrset", q, nil, &out); err != nil {
		return nil, err
	}
	for i := range out.Sets {
		s := out.Sets[i]
		// 通配符在 Route 53 中以 \052 返回
		name := strings.ReplaceAll(strings.TrimSuffix(s.Name, "."), `\052`, "*")
		if strings.EqualFold(name, fqdn) && s.Type == typ {
			return &s, nil
		}
	}
	return nil, nil
}

func xmlEscape(s string) string {
	var b bytes.Buffer
	_ = xml.EscapeText(&b, []byte(s))
	return b.String()
}

// change 提交一次 UPSERT / DELETE。
func (r *route53) change(ctx context.Context, zid, action, fqdn, typ string, ttl int, values []string) error {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?><ChangeResourceRecordSetsRequest xmlns="https://route53.amazonaws.com/doc/2013-04-01/"><ChangeBatch><Changes><Change>`)
	b.WriteString("<Action>" + action + "</Action><ResourceRecordSet>")
	b.WriteString("<Name>" + xmlEscape(fqdn+".") + "</Name><Type>" + typ + "</Type>")
	b.WriteString(fmt.Sprintf("<TTL>%d</TTL><ResourceRecords>", ttl))
	for _, v := range values {
		b.WriteString("<ResourceRecord><Value>" + xmlEscape(v) + "</Value></ResourceRecord>")
	}
	b.WriteString("</ResourceRecords></ResourceRecordSet></Change></Changes></ChangeBatch></ChangeResourceRecordSetsRequest>")
	return r.call(ctx, http.MethodPost, r53Version+"/hostedzone/"+zid+"/rrset", nil, []byte(b.String()), nil)
}

func (r *route53) Upsert(ctx context.Context, rec Record) (string, error) {
	zid, err := r.hostedZone(ctx, rec.Zone)
	if err != nil {
		return "", err
	}
	fqdn := rec.FQDN()
	cur, err := r.getSet(ctx, zid, fqdn, rec.Type)
	if err != nil {
		return "", err
	}
	ttl := rec.TTL
	if ttl <= 0 {
		ttl = 300
	}
	if cur != nil && len(cur.Values) == 1 && cur.Values[0] == rec.Value && (rec.TTL <= 0 || cur.TTL == ttl) {
		return ActionUnchanged, nil
	}
	if err := r.change(ctx, zid, "UPSERT", fqdn, rec.Type, ttl, []string{rec.Value}); err != nil {
		return "", err
	}
	if cur == nil {
		return ActionCreated, nil
	}
	return ActionUpdated, nil
}

func (r *route53) AddTXT(ctx context.Context, zone, sub, value string) error {
	zid, err := r.hostedZone(ctx, zone)
	if err != nil {
		return err
	}
	fqdn := Record{Zone: zone, Sub: sub}.FQDN()
	cur, err := r.getSet(ctx, zid, fqdn, "TXT")
	if err != nil {
		return err
	}
	vals := []string{`"` + value + `"`}
	if cur != nil {
		vals = append(cur.Values, vals...)
	}
	return r.change(ctx, zid, "UPSERT", fqdn, "TXT", 60, vals)
}

func (r *route53) DeleteTXT(ctx context.Context, zone, sub, value string) error {
	zid, err := r.hostedZone(ctx, zone)
	if err != nil {
		return err
	}
	fqdn := Record{Zone: zone, Sub: sub}.FQDN()
	cur, err := r.getSet(ctx, zid, fqdn, "TXT")
	if err != nil || cur == nil {
		return err
	}
	var keep []string
	for _, v := range cur.Values {
		if unquote(v) != value {
			keep = append(keep, v)
		}
	}
	if len(keep) == len(cur.Values) {
		return nil
	}
	if len(keep) == 0 {
		// DELETE 需要与现有记录集完全一致
		return r.change(ctx, zid, "DELETE", fqdn, "TXT", cur.TTL, cur.Values)
	}
	return r.change(ctx, zid, "UPSERT", fqdn, "TXT", cur.TTL, keep)
}

var _ TXTProvider = (*route53)(nil)
