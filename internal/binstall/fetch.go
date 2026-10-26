package binstall

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/wangwanjie/gately/internal/fsperm"
)

// GitHubAPI 是 GitHub API 地址，测试中可替换。
var GitHubAPI = "https://api.github.com"

// GitHubAsset 是 GitHub 发布中的一个文件。
type GitHubAsset struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
	Size int64  `json:"size"`
	// Digest 为 GitHub 记录的文件摘要，格式 sha256:<十六进制>，2025 年 6 月前上传的文件没有
	Digest string `json:"digest"`
}

// GitHubRelease 是 GitHub 的一个发布。
type GitHubRelease struct {
	TagName string        `json:"tag_name"`
	Body    string        `json:"body"`
	Assets  []GitHubAsset `json:"assets"`
}

// Find 返回名为 name 的文件。
func (r *GitHubRelease) Find(name string) *GitHubAsset {
	for i := range r.Assets {
		if r.Assets[i].Name == name {
			return &r.Assets[i]
		}
	}
	return nil
}

// FetchGitHubRelease 查询 repo（owner/name）的发布：version 为空时取最新正式版，
// 否则按标签查找，标签带不带前缀 v 都会尝试。
func FetchGitHubRelease(ctx context.Context, client *http.Client, repo, version string) (*GitHubRelease, error) {
	if version == "" {
		return getRelease(ctx, client, GitHubAPI+"/repos/"+repo+"/releases/latest")
	}
	tags := []string{version}
	if strings.HasPrefix(version, "v") {
		tags = append(tags, strings.TrimPrefix(version, "v"))
	} else {
		tags = append(tags, "v"+version)
	}
	var lastErr error
	for _, tag := range tags {
		rel, err := getRelease(ctx, client, GitHubAPI+"/repos/"+repo+"/releases/tags/"+url.PathEscape(tag))
		if err == nil {
			return rel, nil
		}
		lastErr = err
		if !errors.Is(err, errNotFound) {
			return nil, err
		}
	}
	if errors.Is(lastErr, errNotFound) {
		return nil, fmt.Errorf("没有找到版本 %s", version)
	}
	return nil, lastErr
}

var errNotFound = errors.New("not found")

func getRelease(ctx context.Context, client *http.Client, rawURL string) (*GitHubRelease, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("无法访问 GitHub: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, errNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub 返回 %s", resp.Status)
	}
	var rel GitHubRelease
	if err := json.NewDecoder(io.LimitReader(resp.Body, 8<<20)).Decode(&rel); err != nil {
		return nil, fmt.Errorf("解析版本信息失败: %v", err)
	}
	if rel.TagName == "" {
		return nil, errors.New("解析版本信息失败: 缺少版本号")
	}
	return &rel, nil
}

var hexSumRe = regexp.MustCompile(`\b[0-9a-fA-F]{64}\b`)

// FindChecksum 在校验文本中查找 name 的 SHA-256，兼容 “<摘要>  <文件名>” 与 “<文件名>: <摘要>” 两种写法。
func FindChecksum(text, name string) string {
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		fields := strings.FieldsFunc(line, func(r rune) bool { return r == ' ' || r == '\t' || r == ':' || r == '*' })
		if !slices.Contains(fields, name) {
			continue
		}
		if m := hexSumRe.FindString(line); m != "" {
			return strings.ToLower(m)
		}
	}
	return ""
}

// GitHubAssetRelease 把 GitHub 发布中名为 name 的文件转换为 Release：摘要优先取 GitHub 记录的 digest，
// 没有时依次在名为 checksumFiles 的校验文件与发布说明中查找。
func GitHubAssetRelease(ctx context.Context, client *http.Client, rel *GitHubRelease, version, name string, checksumFiles ...string) (*Release, error) {
	a := rel.Find(name)
	if a == nil {
		return nil, fmt.Errorf("版本 %s 中没有 %s", rel.TagName, name)
	}
	out := &Release{Version: version, Asset: Asset{Name: a.Name, URL: a.URL, Size: a.Size}}
	if sum, ok := strings.CutPrefix(strings.ToLower(a.Digest), "sha256:"); ok && len(sum) == 64 {
		out.Asset.SHA256 = sum
		return out, nil
	}
	for _, cf := range checksumFiles {
		if c := rel.Find(cf); c != nil {
			text, err := fetchText(ctx, client, c.URL)
			if err != nil {
				return nil, err
			}
			if sum := FindChecksum(text, name); sum != "" {
				out.Asset.SHA256 = sum
				return out, nil
			}
		}
	}
	out.Asset.SHA256 = FindChecksum(rel.Body, name)
	return out, nil
}

// fetchText 读取不超过 1 MB 的文本文件。
func fetchText(ctx context.Context, client *http.Client, rawURL string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("下载失败: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("下载 %s 失败: %s", path.Base(req.URL.Path), resp.Status)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	return string(data), err
}

// FetchText 供 Spec.Resolve 读取校验文件、版本列表等小文件。
func FetchText(ctx context.Context, client *http.Client, rawURL string) (string, error) {
	return fetchText(ctx, client, rawURL)
}

// countWriter 统计写入的字节数并回调，用于报告下载进度。
type countWriter struct {
	n    int64
	tick func(n int64)
}

func (c *countWriter) Write(p []byte) (int, error) {
	c.n += int64(len(p))
	c.tick(c.n)
	return len(p), nil
}

// download 把 rawURL 下载到 dst，返回内容的 SHA-256（十六进制）。size 在收到响应头时以 Content-Length 回调（未知为 -1）。
func download(ctx context.Context, client *http.Client, rawURL, dst string, size, progress func(n int64)) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("下载失败: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("下载失败: %s", resp.Status)
	}
	size(resp.ContentLength)
	f, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return "", err
	}
	h := sha256.New()
	n, err := io.Copy(io.MultiWriter(f, h, &countWriter{tick: progress}), io.LimitReader(resp.Body, maxDownloadSize+1))
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return "", fmt.Errorf("下载失败: %v", err)
	}
	if n > maxDownloadSize {
		return "", fmt.Errorf("文件超过 %d MB，已中止", maxDownloadSize>>20)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// extract 从下载的文件中取出 files 列出的可执行文件放到 dir，权限 0755。
// 压缩包内按文件名匹配（忽略所在目录）；不是压缩包时下载的文件本身就是唯一的程序。
func extract(src, assetName, dir string, files []string) error {
	if err := fsperm.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	want := map[string]bool{}
	for _, f := range files {
		want[ExeName(f)] = true
	}
	lower := strings.ToLower(assetName)
	var err error
	switch {
	case strings.HasSuffix(lower, ".tar.gz") || strings.HasSuffix(lower, ".tgz"):
		err = extractTarGz(src, dir, want)
	case strings.HasSuffix(lower, ".zip"):
		err = extractZip(src, dir, want)
	default:
		if len(files) != 1 {
			return fmt.Errorf("%s 不是压缩包，无法取出 %d 个程序", assetName, len(files))
		}
		err = os.Rename(src, filepath.Join(dir, ExeName(files[0])))
	}
	if err != nil {
		return err
	}
	for _, f := range files {
		p := filepath.Join(dir, ExeName(f))
		if _, err := os.Stat(p); err != nil {
			return fmt.Errorf("%s 中没有 %s", assetName, ExeName(f))
		}
		if err := os.Chmod(p, 0o755); err != nil {
			return err
		}
	}
	return nil
}

// writeEntry 把压缩包中的一个文件写到 dir/name，限制大小。
func writeEntry(r io.Reader, dir, name string) error {
	f, err := os.OpenFile(filepath.Join(dir, name), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o700)
	if err != nil {
		return err
	}
	n, err := io.Copy(f, io.LimitReader(r, maxDownloadSize+1))
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err == nil && n > maxDownloadSize {
		err = fmt.Errorf("%s 超过 %d MB，已中止", name, maxDownloadSize>>20)
	}
	if err != nil {
		return fmt.Errorf("解压失败: %v", err)
	}
	return nil
}

func extractTarGz(src, dir string, want map[string]bool) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("解压失败: %v", err)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("解压失败: %v", err)
		}
		name := path.Base(h.Name)
		if h.Typeflag != tar.TypeReg || !want[name] {
			continue
		}
		if err := writeEntry(tr, dir, name); err != nil {
			return err
		}
		delete(want, name)
	}
}

func extractZip(src, dir string, want map[string]bool) error {
	zr, err := zip.OpenReader(src)
	if err != nil {
		return fmt.Errorf("解压失败: %v", err)
	}
	defer zr.Close()
	for _, zf := range zr.File {
		name := path.Base(zf.Name)
		if zf.FileInfo().IsDir() || !want[name] {
			continue
		}
		rc, err := zf.Open()
		if err != nil {
			return fmt.Errorf("解压失败: %v", err)
		}
		err = writeEntry(rc, dir, name)
		rc.Close()
		if err != nil {
			return err
		}
		delete(want, name)
	}
	return nil
}
