package storage

import (
	"crypto/tls"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
)

// davBackend 是远程 WebDAV 的最小客户端，覆盖 PROPFIND / GET / PUT / MKCOL / DELETE / MOVE。
type davBackend struct {
	base     *url.URL
	user     string
	password string
	client   *http.Client
}

func newDAV(rawURL, user, password string, insecure bool) (*davBackend, error) {
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return nil, errors.New("WebDAV 地址需以 http:// 或 https:// 开头")
	}
	u.Path = strings.TrimRight(u.Path, "/")
	tr := http.DefaultTransport.(*http.Transport).Clone()
	if insecure {
		tr.TLSClientConfig = insecureTLS()
	}
	// 不设整体超时，以免大文件传输被中断；连接与响应头由 Transport 默认值约束
	return &davBackend{base: u, user: user, password: password, client: &http.Client{Transport: tr}}, nil
}

// endpoint 返回挂载内路径对应的完整地址。
func (d *davBackend) endpoint(p string) string {
	u := *d.base
	u.Path = d.base.Path + cleanPath(p)
	if u.Path == "" {
		u.Path = "/"
	}
	return u.String()
}

func (d *davBackend) do(method, p string, body io.Reader, hdr map[string]string) (*http.Response, error) {
	req, err := http.NewRequest(method, d.endpoint(p), body)
	if err != nil {
		return nil, err
	}
	if d.user != "" {
		req.SetBasicAuth(d.user, d.password)
	}
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	return d.client.Do(req)
}

func statusErr(resp *http.Response) error {
	switch resp.StatusCode {
	case http.StatusNotFound:
		return errNotFound
	case http.StatusUnauthorized:
		return errors.New("远程认证失败")
	case http.StatusForbidden:
		return errors.New("远程拒绝访问")
	}
	return fmt.Errorf("远程返回 %s", resp.Status)
}

type propfindResp struct {
	Responses []struct {
		Href     string `xml:"href"`
		Propstat []struct {
			Prop struct {
				IsDir    *struct{} `xml:"resourcetype>collection"`
				Length   string    `xml:"getcontentlength"`
				Modified string    `xml:"getlastmodified"`
			} `xml:"prop"`
		} `xml:"propstat"`
	} `xml:"response"`
}

func (d *davBackend) List(p string) ([]Entry, error) {
	resp, err := d.do("PROPFIND", p, strings.NewReader(`<?xml version="1.0"?><propfind xmlns="DAV:"><prop><resourcetype/><getcontentlength/><getlastmodified/></prop></propfind>`),
		map[string]string{"Depth": "1", "Content-Type": "application/xml"})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusMultiStatus {
		return nil, statusErr(resp)
	}
	var pr propfindResp
	if err := xml.NewDecoder(io.LimitReader(resp.Body, 32<<20)).Decode(&pr); err != nil {
		return nil, fmt.Errorf("解析远程响应失败: %w", err)
	}
	self := strings.TrimRight(d.base.Path+cleanPath(p), "/")
	var out []Entry
	for _, r := range pr.Responses {
		href := r.Href
		if u, err := url.Parse(href); err == nil {
			href = u.Path
		}
		href = strings.TrimRight(href, "/")
		// 跳过目录自身
		if href == self || href == "" && self == "" {
			continue
		}
		e := Entry{Name: path.Base(href)}
		for _, ps := range r.Propstat {
			if ps.Prop.IsDir != nil {
				e.IsDir = true
			}
			if ps.Prop.Length != "" {
				_, _ = fmt.Sscan(ps.Prop.Length, &e.Size)
			}
			if t, err := http.ParseTime(ps.Prop.Modified); err == nil {
				e.ModTime = t
			}
		}
		out = append(out, e)
	}
	sortEntries(out)
	return out, nil
}

func (d *davBackend) Open(p string) (io.ReadCloser, int64, time.Time, error) {
	resp, err := d.do(http.MethodGet, p, nil, nil)
	if err != nil {
		return nil, 0, time.Time{}, err
	}
	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		return nil, 0, time.Time{}, statusErr(resp)
	}
	mod, _ := http.ParseTime(resp.Header.Get("Last-Modified"))
	return resp.Body, resp.ContentLength, mod, nil
}

func (d *davBackend) simple(method, p string, body io.Reader, hdr map[string]string, ok ...int) error {
	resp, err := d.do(method, p, body, hdr)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<20))
	for _, c := range ok {
		if resp.StatusCode == c {
			return nil
		}
	}
	return statusErr(resp)
}

func (d *davBackend) Put(p string, r io.Reader, size int64) error {
	// 多数 WebDAV 服务端不接受分块上传，必须带 Content-Length；大小未知时无法上传
	if size < 0 {
		return errors.New("远程存储上传需要已知文件大小")
	}
	req, err := http.NewRequest(http.MethodPut, d.endpoint(p), r)
	if err != nil {
		return err
	}
	req.ContentLength = size
	if d.user != "" {
		req.SetBasicAuth(d.user, d.password)
	}
	resp, err := d.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<20))
	switch resp.StatusCode {
	case http.StatusOK, http.StatusCreated, http.StatusNoContent:
		return nil
	}
	return statusErr(resp)
}

func (d *davBackend) Mkdir(p string) error {
	return d.simple("MKCOL", p, nil, nil, http.StatusCreated)
}

func (d *davBackend) Remove(p string) error {
	if cleanPath(p) == "/" {
		return errors.New("不能删除挂载根目录")
	}
	return d.simple(http.MethodDelete, p, nil, nil, http.StatusOK, http.StatusNoContent)
}

func (d *davBackend) Rename(from, to string) error {
	return d.simple("MOVE", from, nil, map[string]string{"Destination": d.endpoint(to), "Overwrite": "F"},
		http.StatusCreated, http.StatusNoContent, http.StatusOK)
}

func (d *davBackend) Close() error {
	d.client.CloseIdleConnections()
	return nil
}

// insecureTLS 返回跳过证书校验的配置，仅在用户为该挂载显式勾选时使用（自签名证书的内网 NAS）。
func insecureTLS() *tls.Config {
	return &tls.Config{InsecureSkipVerify: true}
}
