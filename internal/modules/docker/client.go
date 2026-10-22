// Package docker 实现 Docker 管理模块：查看容器、启停、日志，以及检查镜像更新并按原配置重建容器。
//
// 直接调用 Docker Engine API（默认 /var/run/docker.sock），不依赖官方 SDK。
package docker

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

// apiVersion 为首选的 Engine API 版本，1.41 对应 Docker 20.10，兼容绝大多数部署。
// 实际版本在首次请求前协商：Docker 25 起最低版本逐步提高（Docker 29 要求至少 1.44），
// 服务端最低版本高于首选时改用服务端最低版本；服务端低于首选时改用服务端版本。
const apiVersion = "1.41"

// Client 是精简的 Docker Engine API 客户端。
type Client struct {
	hc *http.Client
	// root 为不带版本前缀的地址，例如 http://docker
	root string
	// mu 保护协商得到的 API 版本前缀 base（例如 http://docker/v1.44），协商成功前为空
	mu   sync.Mutex
	base string
	// network 与 addr 记录底层拨号信息（unix 套接字路径或 tcp 地址），连接劫持（终端）需要直接拨号
	network string
	addr    string
}

// NewClient 按地址创建客户端：unix:///var/run/docker.sock 或 tcp://host:2375。
func NewClient(host string) (*Client, error) {
	if host == "" {
		host = "unix:///var/run/docker.sock"
	}
	u, err := url.Parse(host)
	if err != nil {
		return nil, err
	}
	switch u.Scheme {
	case "unix":
		path := u.Path
		tr := &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			var d net.Dialer
			return d.DialContext(ctx, "unix", path)
		}}
		return &Client{hc: &http.Client{Transport: tr}, root: "http://docker", network: "unix", addr: path}, nil
	case "tcp", "http":
		return &Client{hc: &http.Client{}, root: "http://" + u.Host, network: "tcp", addr: u.Host}, nil
	}
	return nil, fmt.Errorf("不支持的 Docker 地址 %q", host)
}

// APIError 是 Docker 返回的错误。
type APIError struct {
	Status  int
	Message string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("Docker %d: %s", e.Status, e.Message)
}

// endpoint 返回带 API 版本前缀的地址，首次调用时与服务端协商版本；协商失败（Docker 未就绪等）不缓存，下次重试。
func (c *Client) endpoint(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.base != "" {
		return c.base, nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.root+"/version", nil)
	if err != nil {
		return "", err
	}
	resp, err := c.hc.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var v struct {
		APIVersion    string `json:"ApiVersion"`
		MinAPIVersion string `json:"MinAPIVersion"`
	}
	// 读不到版本（例如经过屏蔽了 /version 的套接字代理）时按首选版本请求，与协商前的行为一致
	if resp.StatusCode == http.StatusOK {
		_ = json.NewDecoder(resp.Body).Decode(&v)
	}
	c.base = c.root + "/v" + negotiateVersion(v.APIVersion, v.MinAPIVersion)
	return c.base, nil
}

// negotiateVersion 在服务端支持的范围 [min, max] 内选择最接近首选版本 apiVersion 的版本。
func negotiateVersion(max, min string) string {
	ver := apiVersion
	if min != "" && compareVersion(min, ver) > 0 {
		ver = min
	}
	if max != "" && compareVersion(max, ver) < 0 {
		ver = max
	}
	return ver
}

// compareVersion 比较形如 1.41 的版本号。
func compareVersion(a, b string) int {
	pa, pb := strings.Split(a, "."), strings.Split(b, ".")
	for i := 0; i < len(pa) || i < len(pb); i++ {
		var x, y int
		if i < len(pa) {
			x, _ = strconv.Atoi(pa[i])
		}
		if i < len(pb) {
			y, _ = strconv.Atoi(pb[i])
		}
		if x != y {
			if x < y {
				return -1
			}
			return 1
		}
	}
	return 0
}

func (c *Client) do(ctx context.Context, method, path string, body any, out any) error {
	base, err := c.endpoint(ctx)
	if err != nil {
		return err
	}
	var rd io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		rd = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, base+path, rd)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 && resp.StatusCode != http.StatusNotModified {
		var e struct {
			Message string `json:"message"`
		}
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
		_ = json.Unmarshal(b, &e)
		if e.Message == "" {
			e.Message = strings.TrimSpace(string(b))
		}
		return &APIError{Status: resp.StatusCode, Message: e.Message}
	}
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	return nil
}

// Ping 检查 Docker 是否可用。
func (c *Client) Ping(ctx context.Context) error {
	return c.do(ctx, http.MethodGet, "/_ping", nil, nil)
}

// Container 是容器列表项。
type Container struct {
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
	} `json:"Mounts"`
	NetworkSettings struct {
		Networks map[string]json.RawMessage `json:"Networks"`
	} `json:"NetworkSettings"`
}

// Name 返回不带斜杠的容器名。
func (c Container) Name() string {
	if len(c.Names) == 0 {
		return c.ID[:12]
	}
	return strings.TrimPrefix(c.Names[0], "/")
}

// Containers 列出全部容器。
func (c *Client) Containers(ctx context.Context) ([]Container, error) {
	var out []Container
	err := c.do(ctx, http.MethodGet, "/containers/json?all=1", nil, &out)
	return out, err
}

// Action 执行 start/stop/restart/pause/unpause/kill。
func (c *Client) Action(ctx context.Context, id, action string) error {
	switch action {
	case "start", "stop", "restart", "pause", "unpause", "kill":
	default:
		return fmt.Errorf("不支持的操作 %q", action)
	}
	return c.do(ctx, http.MethodPost, "/containers/"+url.PathEscape(id)+"/"+action, nil, nil)
}

// Logs 返回容器最近的日志。非 TTY 容器的输出是带 8 字节头的多路复用流，这里解析为纯文本。
func (c *Client) Logs(ctx context.Context, id string, tail int) (string, error) {
	return c.LogsWith(ctx, id, tail, true)
}

// LogsWith 与 Logs 相同，timestamps 控制是否带时间戳。
func (c *Client) LogsWith(ctx context.Context, id string, tail int, timestamps bool) (string, error) {
	q := url.Values{"stdout": {"1"}, "stderr": {"1"}, "tail": {fmt.Sprint(tail)}}
	if timestamps {
		q.Set("timestamps", "1")
	}
	base, err := c.endpoint(ctx)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/containers/"+url.PathEscape(id)+"/logs?"+q.Encode(), nil)
	if err != nil {
		return "", err
	}
	resp, err := c.hc.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", &APIError{Status: resp.StatusCode, Message: string(b)}
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return "", err
	}
	return demux(data), nil
}

// demux 解析 Docker 多路复用日志流；不是该格式时原样返回。
func demux(data []byte) string {
	var out strings.Builder
	rest := data
	for len(rest) >= 8 {
		stream := rest[0]
		if (stream != 0 && stream != 1 && stream != 2) || rest[1] != 0 || rest[2] != 0 || rest[3] != 0 {
			return string(data)
		}
		n := int(binary.BigEndian.Uint32(rest[4:8]))
		if 8+n > len(rest) {
			return string(data)
		}
		out.Write(rest[8 : 8+n])
		rest = rest[8+n:]
	}
	if len(rest) > 0 {
		return string(data)
	}
	return out.String()
}

// FollowLogs 持续读取容器从现在开始产生的日志，把标准输出与标准错误按块交给 onData，直到 ctx 取消或日志流结束。
// 非 TTY 容器的输出带 8 字节的多路复用头，这里按第一个块的格式判断并去掉。
func (c *Client) FollowLogs(ctx context.Context, id string, onData func([]byte)) error {
	q := url.Values{"stdout": {"1"}, "stderr": {"1"}, "follow": {"1"}, "tail": {"0"}, "since": {fmt.Sprint(time.Now().Unix())}}
	base, err := c.endpoint(ctx)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/containers/"+url.PathEscape(id)+"/logs?"+q.Encode(), nil)
	if err != nil {
		return err
	}
	resp, err := c.hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		var e struct {
			Message string `json:"message"`
		}
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		if json.Unmarshal(b, &e) != nil || e.Message == "" {
			e.Message = strings.TrimSpace(string(b))
		}
		return &APIError{Status: resp.StatusCode, Message: e.Message}
	}
	rd := bufio.NewReaderSize(resp.Body, 64<<10)
	head, err := rd.Peek(8)
	if err != nil {
		if errors.Is(err, io.EOF) && len(head) > 0 {
			onData(head)
		}
		return err
	}
	multiplexed := (head[0] == 0 || head[0] == 1 || head[0] == 2) && head[1] == 0 && head[2] == 0 && head[3] == 0
	if !multiplexed {
		buf := make([]byte, 32<<10)
		for {
			n, err := rd.Read(buf)
			if n > 0 {
				onData(buf[:n])
			}
			if err != nil {
				return err
			}
		}
	}
	hdr := make([]byte, 8)
	for {
		if _, err := io.ReadFull(rd, hdr); err != nil {
			return err
		}
		n := int(binary.BigEndian.Uint32(hdr[4:8]))
		if n > 1<<20 {
			return fmt.Errorf("日志块过大（%d 字节）", n)
		}
		chunk := make([]byte, n)
		if _, err := io.ReadFull(rd, chunk); err != nil {
			return err
		}
		onData(chunk)
	}
}

// Inspect 返回容器详情的原始 JSON，重建容器时需要完整保留未知字段。
func (c *Client) Inspect(ctx context.Context, id string) (map[string]json.RawMessage, error) {
	var out map[string]json.RawMessage
	err := c.do(ctx, http.MethodGet, "/containers/"+url.PathEscape(id)+"/json", nil, &out)
	return out, err
}

// ImageID 返回本地镜像 ID；镜像不存在时返回空字符串。
func (c *Client) ImageID(ctx context.Context, ref string) (string, error) {
	var out struct {
		ID string `json:"Id"`
	}
	err := c.do(ctx, http.MethodGet, "/images/"+url.PathEscape(ref)+"/json", nil, &out)
	var ae *APIError
	if errors.As(err, &ae) && ae.Status == http.StatusNotFound {
		return "", nil
	}
	return out.ID, err
}

// Pull 拉取镜像。返回的进度流逐行解析，遇到错误消息时返回错误。
func (c *Client) Pull(ctx context.Context, ref string) error {
	image, tag := splitRef(ref)
	q := url.Values{"fromImage": {image}, "tag": {tag}}
	base, err := c.endpoint(ctx)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/images/create?"+q.Encode(), nil)
	if err != nil {
		return err
	}
	resp, err := c.hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return &APIError{Status: resp.StatusCode, Message: strings.TrimSpace(string(b))}
	}
	sc := bufio.NewScanner(resp.Body)
	sc.Buffer(make([]byte, 64<<10), 1<<20)
	for sc.Scan() {
		var msg struct {
			Error string `json:"error"`
		}
		if json.Unmarshal(sc.Bytes(), &msg) == nil && msg.Error != "" {
			return errors.New(msg.Error)
		}
	}
	return sc.Err()
}

// splitRef 把镜像引用拆成名称与标签，没有标签时为 latest；带摘要的引用原样返回。
func splitRef(ref string) (string, string) {
	if strings.Contains(ref, "@") {
		return ref, ""
	}
	slash := strings.LastIndexByte(ref, '/')
	colon := strings.LastIndexByte(ref, ':')
	if colon > slash {
		return ref[:colon], ref[colon+1:]
	}
	return ref, "latest"
}

// Rename 重命名容器。
func (c *Client) Rename(ctx context.Context, id, name string) error {
	return c.do(ctx, http.MethodPost, "/containers/"+url.PathEscape(id)+"/rename?name="+url.QueryEscape(name), nil, nil)
}

// Remove 删除容器（保留数据卷）。
func (c *Client) Remove(ctx context.Context, id string) error {
	return c.RemoveVolumes(ctx, id, false)
}

// RemoveVolumes 强制删除容器，volumes 为 true 时同时删除匿名卷。
func (c *Client) RemoveVolumes(ctx context.Context, id string, volumes bool) error {
	path := "/containers/" + url.PathEscape(id) + "?force=1"
	if volumes {
		path += "&v=1"
	}
	return c.do(ctx, http.MethodDelete, path, nil, nil)
}

// Create 创建容器，返回新容器 ID。
func (c *Client) Create(ctx context.Context, name string, body any) (string, error) {
	var out struct {
		ID string `json:"Id"`
	}
	err := c.do(ctx, http.MethodPost, "/containers/create?name="+url.QueryEscape(name), body, &out)
	return out.ID, err
}

// waitTimeout 是检查更新与拉取镜像的总超时（大镜像拉取较慢）。
var waitTimeout = 30 * time.Minute

// stepTimeout 是重建容器时单个操作的超时。
var stepTimeout = 2 * time.Minute

// ConnectNetwork 把容器连接到网络。
func (c *Client) ConnectNetwork(ctx context.Context, network, id string, endpoint map[string]any) error {
	return c.do(ctx, http.MethodPost, "/networks/"+url.PathEscape(network)+"/connect", map[string]any{"Container": id, "EndpointConfig": endpoint}, nil)
}
