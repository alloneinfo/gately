package dnsprovider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

// 阿里云官方文档"签名机制"中的示例
func TestAliSignOfficialExample(t *testing.T) {
	q := aliSign(map[string]string{
		"Format":           "XML",
		"AccessKeyId":      "testid",
		"Action":           "DescribeDomainRecords",
		"SignatureMethod":  "HMAC-SHA1",
		"DomainName":       "example.com",
		"SignatureNonce":   "f59ed6a9-83fc-473b-9cc6-99c95df3856e",
		"SignatureVersion": "1.0",
		"Version":          "2015-01-09",
		"Timestamp":        "2016-03-24T16:41:54Z",
	}, "testsecret")
	v, _ := url.ParseQuery(q)
	if got := v.Get("Signature"); got != "uRpHwaSEt3J+6KQD//svCh/x+pI=" {
		t.Fatalf("签名与官方示例不一致: %s", got)
	}
}

func TestAliDNSUpsert(t *testing.T) {
	var actions []string
	existing := ""
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("Signature") == "" || q.Get("AccessKeyId") != "ak" {
			w.WriteHeader(400)
			_, _ = w.Write([]byte(`{"Code":"SignatureDoesNotMatch","Message":"bad"}`))
			return
		}
		action := q.Get("Action")
		actions = append(actions, action)
		switch action {
		case "DescribeSubDomainRecords":
			if q.Get("SubDomain") != "www.example.com" || q.Get("Type") != "A" {
				t.Errorf("查询参数错误: %v", q)
			}
			if existing == "" {
				_, _ = w.Write([]byte(`{"TotalCount":0,"DomainRecords":{"Record":[]}}`))
				return
			}
			_, _ = w.Write([]byte(`{"TotalCount":1,"DomainRecords":{"Record":[{"RecordId":"r1","RR":"www","Type":"A","Value":"` + existing + `","TTL":600,"Line":"default"}]}}`))
		case "AddDomainRecord":
			if q.Get("DomainName") != "example.com" || q.Get("RR") != "www" || q.Get("Value") != "1.1.1.1" {
				t.Errorf("新增参数错误: %v", q)
			}
			existing = q.Get("Value")
			_, _ = w.Write([]byte(`{"RecordId":"r1"}`))
		case "UpdateDomainRecord":
			if q.Get("RecordId") != "r1" {
				t.Errorf("更新参数错误: %v", q)
			}
			existing = q.Get("Value")
			_, _ = w.Write([]byte(`{"RecordId":"r1"}`))
		}
	}))
	defer srv.Close()

	p, err := New("alidns", Credentials{ID: "ak", Secret: "sk"}, srv.Client())
	if err != nil {
		t.Fatal(err)
	}
	p.(*aliDNS).endpoint = srv.URL + "/"
	p.(*aliDNS).now = func() time.Time { return time.Unix(1700000000, 0) }
	rec := Record{Zone: "example.com", Sub: "www", Type: "A", Value: "1.1.1.1"}
	steps := []struct {
		value, want string
	}{{"1.1.1.1", ActionCreated}, {"1.1.1.1", ActionUnchanged}, {"2.2.2.2", ActionUpdated}}
	for _, s := range steps {
		rec.Value = s.value
		got, err := p.Upsert(context.Background(), rec)
		if err != nil {
			t.Fatal(err)
		}
		if got != s.want {
			t.Fatalf("value=%s got %s want %s", s.value, got, s.want)
		}
	}
	if strings.Join(actions, ",") != "DescribeSubDomainRecords,AddDomainRecord,DescribeSubDomainRecords,DescribeSubDomainRecords,UpdateDomainRecord" {
		t.Fatalf("调用序列错误: %v", actions)
	}

	bad, _ := New("alidns", Credentials{ID: "wrong", Secret: "sk"}, srv.Client())
	bad.(*aliDNS).endpoint = srv.URL + "/"
	if _, err := bad.Upsert(context.Background(), rec); err == nil || !strings.Contains(err.Error(), "SignatureDoesNotMatch") {
		t.Fatalf("应返回阿里云错误码: %v", err)
	}
}
