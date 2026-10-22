package docker

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// 本文件是镜像、网络、卷与 Compose 功能使用的 Engine API 封装。

// escPath 按段转义镜像引用：保留斜杠，其余字符转义，避免 %2F 被部分 Docker 版本拒绝。
func escPath(ref string) string {
	parts := strings.Split(ref, "/")
	for i, p := range parts {
		parts[i] = url.PathEscape(p)
	}
	return strings.Join(parts, "/")
}

// rawRequest 发起请求并返回未读取的响应，状态码 >= 300 时转为 *APIError。调用方负责关闭响应体。
func (c *Client) rawRequest(ctx context.Context, method, path, contentType string, body io.Reader, headers map[string]string) (*http.Response, error) {
	base, err := c.endpoint(ctx)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, method, base+path, body)
	if err != nil {
		return nil, err
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		defer resp.Body.Close()
		var e struct {
			Message string `json:"message"`
		}
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
		_ = json.Unmarshal(b, &e)
		if e.Message == "" {
			e.Message = strings.TrimSpace(string(b))
		}
		return nil, &APIError{Status: resp.StatusCode, Message: e.Message}
	}
	return resp, nil
}

// jsonMessage 是 Docker 流式接口（拉取、推送、构建、加载）返回的一行 JSON。
type jsonMessage struct {
	Status      string `json:"status"`
	ID          string `json:"id"`
	Stream      string `json:"stream"`
	Error       string `json:"error"`
	ErrorDetail struct {
		Message string `json:"message"`
	} `json:"errorDetail"`
	ProgressDetail struct {
		Current int64 `json:"current"`
		Total   int64 `json:"total"`
	} `json:"progressDetail"`
}

// errText 返回消息里的错误文本，没有错误时为空。
func (m jsonMessage) errText() string {
	if m.Error != "" {
		return m.Error
	}
	return m.ErrorDetail.Message
}

// readJSONStream 逐条解码 JSON 流，fn 返回错误时终止；消息里带 error 字段时返回该错误。
func readJSONStream(r io.Reader, fn func(jsonMessage)) error {
	dec := json.NewDecoder(r)
	for {
		var msg jsonMessage
		if err := dec.Decode(&msg); err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
		if t := msg.errText(); t != "" {
			return errors.New(t)
		}
		if fn != nil {
			fn(msg)
		}
	}
}

// encodeAuth 把仓库认证信息编码为 X-Registry-Auth 头值。
func encodeAuth(server, username, password string) string {
	b, _ := json.Marshal(map[string]string{"username": username, "password": password, "serveraddress": server})
	return base64.URLEncoding.EncodeToString(b)
}

// emptyAuth 是无认证时的 X-Registry-Auth 值（推送接口要求必须带该头）。
const emptyAuth = "e30=" // base64("{}")

// ---------- 镜像 ----------

// imageSummary 是 /images/json 的列表项。
type imageSummary struct {
	ID          string   `json:"Id"`
	RepoTags    []string `json:"RepoTags"`
	RepoDigests []string `json:"RepoDigests"`
	Size        int64    `json:"Size"`
	Created     int64    `json:"Created"`
}

func (c *Client) resImages(ctx context.Context) ([]imageSummary, error) {
	var out []imageSummary
	err := c.do(ctx, http.MethodGet, "/images/json", nil, &out)
	return out, err
}

// imageInspect 是 /images/{id}/json 的常用字段。
type imageInspect struct {
	ID           string   `json:"Id"`
	RepoTags     []string `json:"RepoTags"`
	Size         int64    `json:"Size"`
	Created      string   `json:"Created"`
	Os           string   `json:"Os"`
	Architecture string   `json:"Architecture"`
	Author       string   `json:"Author"`
	Config       struct {
		Cmd          []string            `json:"Cmd"`
		Entrypoint   []string            `json:"Entrypoint"`
		Env          []string            `json:"Env"`
		ExposedPorts map[string]struct{} `json:"ExposedPorts"`
		WorkingDir   string              `json:"WorkingDir"`
		User         string              `json:"User"`
		Labels       map[string]string   `json:"Labels"`
	} `json:"Config"`
}

func (c *Client) resImageInspect(ctx context.Context, ref string) (imageInspect, error) {
	var out imageInspect
	err := c.do(ctx, http.MethodGet, "/images/"+escPath(ref)+"/json", nil, &out)
	return out, err
}

// imageHistoryItem 是 /images/{id}/history 的一层。
type imageHistoryItem struct {
	Created   int64  `json:"Created"`
	CreatedBy string `json:"CreatedBy"`
	Size      int64  `json:"Size"`
}

func (c *Client) resImageHistory(ctx context.Context, ref string) ([]imageHistoryItem, error) {
	var out []imageHistoryItem
	err := c.do(ctx, http.MethodGet, "/images/"+escPath(ref)+"/history", nil, &out)
	return out, err
}

// resImageTag 给镜像增加标签。
func (c *Client) resImageTag(ctx context.Context, ref, repo, tag string) error {
	q := url.Values{"repo": {repo}}
	if tag != "" {
		q.Set("tag", tag)
	}
	return c.do(ctx, http.MethodPost, "/images/"+escPath(ref)+"/tag?"+q.Encode(), nil, nil)
}

// resImageRemove 删除镜像或标签，返回 Docker 报告的 Untagged / Deleted 条目。
func (c *Client) resImageRemove(ctx context.Context, ref string, force bool) ([]map[string]string, error) {
	path := "/images/" + escPath(ref)
	if force {
		path += "?force=1"
	}
	var out []map[string]string
	err := c.do(ctx, http.MethodDelete, path, nil, &out)
	return out, err
}

// imageSearchResult 是 Docker Hub 搜索结果。
type imageSearchResult struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Stars       int    `json:"stars"`
	Official    bool   `json:"official"`
}

func (c *Client) resImageSearch(ctx context.Context, term string, limit int) ([]imageSearchResult, error) {
	var raw []struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		StarCount   int    `json:"star_count"`
		IsOfficial  bool   `json:"is_official"`
	}
	q := url.Values{"term": {term}, "limit": {strconv.Itoa(limit)}}
	if err := c.do(ctx, http.MethodGet, "/images/search?"+q.Encode(), nil, &raw); err != nil {
		return nil, err
	}
	out := make([]imageSearchResult, 0, len(raw))
	for _, r := range raw {
		out = append(out, imageSearchResult{Name: r.Name, Description: r.Description, Stars: r.StarCount, Official: r.IsOfficial})
	}
	return out, nil
}

// resImagePull 拉取镜像（带认证），每条进度消息回调 onMsg。
func (c *Client) resImagePull(ctx context.Context, ref, auth string, onMsg func(jsonMessage)) error {
	image, tag := splitRef(ref)
	q := url.Values{"fromImage": {image}}
	if tag != "" {
		q.Set("tag", tag)
	}
	headers := map[string]string{}
	if auth != "" {
		headers["X-Registry-Auth"] = auth
	}
	resp, err := c.rawRequest(ctx, http.MethodPost, "/images/create?"+q.Encode(), "", nil, headers)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return readJSONStream(resp.Body, onMsg)
}

// resImagePush 推送镜像。
func (c *Client) resImagePush(ctx context.Context, ref, auth string, onMsg func(jsonMessage)) error {
	image, tag := splitRef(ref)
	path := "/images/" + escPath(image) + "/push"
	if tag != "" {
		path += "?tag=" + url.QueryEscape(tag)
	}
	if auth == "" {
		auth = emptyAuth
	}
	resp, err := c.rawRequest(ctx, http.MethodPost, path, "", nil, map[string]string{"X-Registry-Auth": auth})
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return readJSONStream(resp.Body, onMsg)
}

// resImageLoad 把 docker save 的 tar 流式写入 Docker，body 不会被整体读入内存。
func (c *Client) resImageLoad(ctx context.Context, body io.Reader, onMsg func(jsonMessage)) error {
	resp, err := c.rawRequest(ctx, http.MethodPost, "/images/load?quiet=0", "application/x-tar", body, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return readJSONStream(resp.Body, onMsg)
}

// resImageSave 返回 docker save 的 tar 流，调用方负责关闭。
func (c *Client) resImageSave(ctx context.Context, ref string) (io.ReadCloser, error) {
	resp, err := c.rawRequest(ctx, http.MethodGet, "/images/"+escPath(ref)+"/get", "", nil, nil)
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}

// resImageBuild 构建镜像。contextTar 为空时使用 query 里的 remote 参数。
func (c *Client) resImageBuild(ctx context.Context, query url.Values, contextTar io.Reader, auth string, onMsg func(jsonMessage)) error {
	ct := ""
	if contextTar != nil {
		ct = "application/x-tar"
	}
	headers := map[string]string{}
	if auth != "" {
		headers["X-Registry-Config"] = auth
	}
	resp, err := c.rawRequest(ctx, http.MethodPost, "/build?"+query.Encode(), ct, contextTar, headers)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return readJSONStream(resp.Body, onMsg)
}

// ---------- 容器引用（镜像计数、卷与网络反查） ----------

// resContainer 是带挂载与网络信息的容器列表项。
type resContainer struct {
	ID      string            `json:"Id"`
	Names   []string          `json:"Names"`
	Image   string            `json:"Image"`
	ImageID string            `json:"ImageID"`
	State   string            `json:"State"`
	Status  string            `json:"Status"`
	Created int64             `json:"Created"`
	Labels  map[string]string `json:"Labels"`
	Ports   []struct {
		IP          string `json:"IP"`
		PrivatePort int    `json:"PrivatePort"`
		PublicPort  int    `json:"PublicPort"`
		Type        string `json:"Type"`
	} `json:"Ports"`
	Mounts []struct {
		Type string `json:"Type"`
		Name string `json:"Name"`
	} `json:"Mounts"`
	NetworkSettings struct {
		Networks map[string]struct {
			NetworkID string `json:"NetworkID"`
			IPAddress string `json:"IPAddress"`
		} `json:"Networks"`
	} `json:"NetworkSettings"`
}

// name 返回不带斜杠的容器名。
func (rc resContainer) name() string {
	if len(rc.Names) == 0 {
		return shortID(rc.ID)
	}
	return strings.TrimPrefix(rc.Names[0], "/")
}

// resContainers 列出容器（含已停止），可按标签过滤。
func (c *Client) resContainers(ctx context.Context, labelFilters ...string) ([]resContainer, error) {
	path := "/containers/json?all=1"
	if len(labelFilters) > 0 {
		path += "&filters=" + url.QueryEscape(labelFilter(labelFilters...))
	}
	var out []resContainer
	err := c.do(ctx, http.MethodGet, path, nil, &out)
	return out, err
}

// labelFilter 生成 Docker 的 label 过滤参数 JSON。
func labelFilter(labels ...string) string {
	b, _ := json.Marshal(map[string][]string{"label": labels})
	return string(b)
}

// ---------- 网络 ----------

// networkInfo 是 /networks 的一项。
type networkInfo struct {
	ID         string `json:"Id"`
	Name       string `json:"Name"`
	Driver     string `json:"Driver"`
	Scope      string `json:"Scope"`
	EnableIPv6 bool   `json:"EnableIPv6"`
	Internal   bool   `json:"Internal"`
	IPAM       struct {
		Config []struct {
			Subnet  string `json:"Subnet"`
			Gateway string `json:"Gateway"`
		} `json:"Config"`
	} `json:"IPAM"`
}

func (c *Client) resNetworks(ctx context.Context, labelFilters ...string) ([]networkInfo, error) {
	path := "/networks"
	if len(labelFilters) > 0 {
		path += "?filters=" + url.QueryEscape(labelFilter(labelFilters...))
	}
	var out []networkInfo
	err := c.do(ctx, http.MethodGet, path, nil, &out)
	return out, err
}

func (c *Client) resNetworkInspect(ctx context.Context, id string) (networkInfo, error) {
	var out networkInfo
	err := c.do(ctx, http.MethodGet, "/networks/"+url.PathEscape(id), nil, &out)
	return out, err
}

func (c *Client) resNetworkCreate(ctx context.Context, body map[string]any) (string, error) {
	var out struct {
		ID string `json:"Id"`
	}
	err := c.do(ctx, http.MethodPost, "/networks/create", body, &out)
	return out.ID, err
}

func (c *Client) resNetworkRemove(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, "/networks/"+url.PathEscape(id), nil, nil)
}

func (c *Client) resNetworkDisconnect(ctx context.Context, network, container string, force bool) error {
	return c.do(ctx, http.MethodPost, "/networks/"+url.PathEscape(network)+"/disconnect",
		map[string]any{"Container": container, "Force": force}, nil)
}

// ---------- 卷 ----------

// volumeInfo 是 /volumes 的一项。
type volumeInfo struct {
	Name       string            `json:"Name"`
	Driver     string            `json:"Driver"`
	Mountpoint string            `json:"Mountpoint"`
	CreatedAt  string            `json:"CreatedAt"`
	Labels     map[string]string `json:"Labels"`
}

func (c *Client) resVolumes(ctx context.Context, labelFilters ...string) ([]volumeInfo, error) {
	path := "/volumes"
	if len(labelFilters) > 0 {
		path += "?filters=" + url.QueryEscape(labelFilter(labelFilters...))
	}
	var out struct {
		Volumes []volumeInfo `json:"Volumes"`
	}
	err := c.do(ctx, http.MethodGet, path, nil, &out)
	return out.Volumes, err
}

// resVolumeUsage 从 /system/df 取各卷的大小与引用数；未知时不在结果中。
func (c *Client) resVolumeUsage(ctx context.Context) (map[string]volumeUsage, error) {
	var out struct {
		Volumes []struct {
			Name      string `json:"Name"`
			UsageData *struct {
				Size     int64 `json:"Size"`
				RefCount int   `json:"RefCount"`
			} `json:"UsageData"`
		} `json:"Volumes"`
	}
	if err := c.do(ctx, http.MethodGet, "/system/df", nil, &out); err != nil {
		return nil, err
	}
	res := map[string]volumeUsage{}
	for _, v := range out.Volumes {
		if v.UsageData != nil {
			res[v.Name] = volumeUsage{Size: v.UsageData.Size, RefCount: v.UsageData.RefCount}
		}
	}
	return res, nil
}

// volumeUsage 是 /system/df 里的卷用量。
type volumeUsage struct {
	Size     int64
	RefCount int
}

func (c *Client) resVolumeCreate(ctx context.Context, name, driver string) error {
	return c.do(ctx, http.MethodPost, "/volumes/create", map[string]any{"Name": name, "Driver": driver}, nil)
}

func (c *Client) resVolumeRemove(ctx context.Context, name string, force bool) error {
	path := "/volumes/" + url.PathEscape(name)
	if force {
		path += "?force=1"
	}
	return c.do(ctx, http.MethodDelete, path, nil, nil)
}

// ---------- 辅助容器（卷导入导出） ----------

// resCreateHelper 创建一个不会启动的辅助容器，把卷挂载到 /vol。
func (c *Client) resCreateHelper(ctx context.Context, image, volume string, readOnly bool) (string, error) {
	bind := volume + ":/vol"
	if readOnly {
		bind += ":ro"
	}
	body := map[string]any{
		"Image":           image,
		"Cmd":             []string{"true"},
		"NetworkDisabled": true,
		"Labels":          map[string]string{"gately.helper": "volume"},
		"HostConfig":      map[string]any{"Binds": []string{bind}},
	}
	var out struct {
		ID string `json:"Id"`
	}
	if err := c.do(ctx, http.MethodPost, "/containers/create", body, &out); err != nil {
		return "", err
	}
	return out.ID, nil
}

// resRemoveContainer 强制删除容器。
func (c *Client) resRemoveContainer(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, "/containers/"+url.PathEscape(id)+"?force=1", nil, nil)
}

// resArchiveGet 以 tar 流读取容器内路径，调用方负责关闭。
func (c *Client) resArchiveGet(ctx context.Context, id, path string) (io.ReadCloser, error) {
	resp, err := c.rawRequest(ctx, http.MethodGet, "/containers/"+url.PathEscape(id)+"/archive?path="+url.QueryEscape(path), "", nil, nil)
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}

// resArchivePut 把 tar 流解压到容器内目录。
func (c *Client) resArchivePut(ctx context.Context, id, path string, body io.Reader) error {
	resp, err := c.rawRequest(ctx, http.MethodPut, "/containers/"+url.PathEscape(id)+"/archive?path="+url.QueryEscape(path), "application/x-tar", body, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	return nil
}

// ---------- Compose（Engine 侧操作） ----------

// resContainerOp 对容器执行 start / stop / restart / pause / unpause。
func (c *Client) resContainerOp(ctx context.Context, id, op string) error {
	switch op {
	case "start", "stop", "restart", "pause", "unpause":
	default:
		return fmt.Errorf("不支持的操作 %q", op)
	}
	return c.do(ctx, http.MethodPost, "/containers/"+url.PathEscape(id)+"/"+op, nil, nil)
}

// isNotFound 判断是否为 Docker 返回的 404。
func isNotFound(err error) bool {
	var ae *APIError
	return errors.As(err, &ae) && ae.Status == http.StatusNotFound
}

// resVolumeInspect 查询卷是否存在。
func (c *Client) resVolumeInspect(ctx context.Context, name string) (volumeInfo, error) {
	var out volumeInfo
	err := c.do(ctx, http.MethodGet, "/volumes/"+url.PathEscape(name), nil, &out)
	return out, err
}
