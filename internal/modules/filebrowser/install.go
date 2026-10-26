package filebrowser

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"runtime/debug"
	"strings"
	"sync/atomic"
	"time"

	"github.com/wangwanjie/gately/internal/fsperm"
	"github.com/wangwanjie/gately/internal/httpx"
	"github.com/wangwanjie/gately/internal/module"
	"github.com/wangwanjie/gately/internal/proc"
)

// defaultReleaseAPI 返回 FileBrowser Quantum 最新的正式版（GitHub 只把 stable 标为 latest，beta 不会出现在这里）。
const defaultReleaseAPI = "https://api.github.com/repos/gtsteffaniak/filebrowser/releases/latest"

const (
	// maxBinarySize 是下载程序的大小上限，正常约 45 MB
	maxBinarySize = 256 << 20
	// installTimeout 是一次安装（查询版本、下载、校验）的总时限
	installTimeout = 15 * time.Minute
)

// 安装阶段。
const (
	stageChecking    = "checking"
	stageDownloading = "downloading"
	stageVerifying   = "verifying"
	stageDone        = "done"
	stageFailed      = "failed"
)

// installState 是一键安装的进度，由 GET /install 返回。
type installState struct {
	Running bool   `json:"running"`
	Stage   string `json:"stage"`
	// Version 为正在安装或已安装的版本
	Version    string `json:"version"`
	Downloaded int64  `json:"downloaded"`
	Total      int64  `json:"total"`
	Error      string `json:"error"`
}

// exeName 在 Windows 下为可执行文件名加上 .exe。
func exeName(name string) string {
	if runtime.GOOS == "windows" {
		return name + ".exe"
	}
	return name
}

// goarm 返回 32 位 ARM 构建的 GOARM 版本（5、6、7），未记录时按 7 处理。
func goarm() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, s := range info.Settings {
			if s.Key == "GOARM" && s.Value != "" {
				return strings.SplitN(s.Value, ",", 2)[0]
			}
		}
	}
	return "7"
}

// assetName 返回上游发布中对应平台的文件名。FileBrowser Quantum 直接发布可执行文件，不打包。
func assetName(goos, goarch, arm string) (string, error) {
	switch goos + "/" + goarch {
	case "linux/amd64", "linux/arm64", "linux/riscv64", "darwin/amd64", "darwin/arm64", "freebsd/amd64":
		return goos + "-" + goarch + "-filebrowser", nil
	case "linux/arm":
		switch arm {
		case "6", "7":
			return "linux-armv" + arm + "-filebrowser", nil
		}
		return "", fmt.Errorf("FileBrowser Quantum 没有提供 ARMv%s 的程序", arm)
	case "windows/amd64":
		return "filebrowser.exe", nil
	}
	return "", fmt.Errorf("FileBrowser Quantum 没有提供 %s/%s 的程序", goos, goarch)
}

type releaseAsset struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
	Size int64  `json:"size"`
	// Digest 为 GitHub 记录的文件摘要，格式 sha256:<十六进制>
	Digest string `json:"digest"`
}

type release struct {
	TagName string         `json:"tag_name"`
	Assets  []releaseAsset `json:"assets"`
}

var (
	quantumVersionRe = regexp.MustCompile(`Version\s*:\s*(\S+)`)
	legacyVersionRe  = regexp.MustCompile(`File Browser (v\S+)`)
)

// probeVersion 执行 `filebrowser version` 返回版本，以及程序是否为 FileBrowser Quantum。
// 原版 filebrowser 输出 “File Browser v2.x/提交”，Quantum 输出多行信息，其中有 “Version : v2.0.2-stable”。
func probeVersion(bin string) (string, bool) {
	out, _ := proc.Output(5*time.Second, bin, []string{"version"}, nil)
	text := string(out)
	if strings.Contains(text, "Quantum") {
		if m := quantumVersionRe.FindStringSubmatch(text); m != nil {
			return m[1], true
		}
		return "", true
	}
	if m := legacyVersionRe.FindStringSubmatch(text); m != nil {
		return m[1], false
	}
	return "", false
}

func (m *Module) registerInstall(r module.Router) {
	r.HandleFunc("GET /install", func(w http.ResponseWriter, req *http.Request) {
		m.mu.Lock()
		defer m.mu.Unlock()
		httpx.OK(w, m.install)
	})
	r.HandleFunc("POST /install", func(w http.ResponseWriter, req *http.Request) {
		m.mu.Lock()
		defer m.mu.Unlock()
		if !m.install.Running {
			m.install = installState{Running: true, Stage: stageChecking}
			go m.runInstall()
		}
		httpx.OK(w, m.install)
	})
}

// setInstall 在持有 mu 的情况下更新安装进度。
func (m *Module) setInstall(fn func(st *installState)) {
	m.mu.Lock()
	fn(&m.install)
	m.mu.Unlock()
}

// runInstall 下载最新稳定版并替换配置目录 bin 下的程序，完成后重启已启用的实例。
func (m *Module) runInstall() {
	ctx, cancel := context.WithTimeout(context.Background(), installTimeout)
	defer cancel()
	log := m.logs.Logger("filebrowser")
	ver, err := m.downloadLatest(ctx)
	if err != nil {
		log.Errorf("安装 FileBrowser Quantum 失败: %v", err)
		m.setInstall(func(st *installState) {
			st.Running, st.Stage, st.Error = false, stageFailed, err.Error()
		})
		return
	}
	log.Infof("已安装 FileBrowser Quantum %s 到 %s", ver, m.localBin())
	m.mu.Lock()
	m.install.Running, m.install.Stage, m.install.Version = false, stageDone, ver
	// 让设置页重新探测版本，并用新程序重启实例
	m.versionPath = ""
	m.restartAllLocked()
	m.mu.Unlock()
}

// downloadLatest 查询最新稳定版、下载并校验 SHA-256，安装到配置目录 bin 下，返回版本号。
func (m *Module) downloadLatest(ctx context.Context) (string, error) {
	name, err := assetName(runtime.GOOS, runtime.GOARCH, goarm())
	if err != nil {
		return "", err
	}
	client := m.client()
	rel, err := m.fetchRelease(ctx, client)
	if err != nil {
		return "", err
	}
	var asset *releaseAsset
	for k := range rel.Assets {
		if rel.Assets[k].Name == name {
			asset = &rel.Assets[k]
		}
	}
	if asset == nil {
		return "", fmt.Errorf("版本 %s 中没有 %s", rel.TagName, name)
	}
	want, ok := strings.CutPrefix(strings.ToLower(asset.Digest), "sha256:")
	if !ok || len(want) != sha256.Size*2 {
		return "", fmt.Errorf("版本 %s 的 %s 缺少 SHA-256 校验值，已中止", rel.TagName, name)
	}
	m.setInstall(func(st *installState) {
		st.Stage, st.Version, st.Total = stageDownloading, rel.TagName, asset.Size
	})

	target := m.localBin()
	if err := fsperm.MkdirAll(filepath.Dir(target), 0o700); err != nil {
		return "", fmt.Errorf("创建目录失败: %v", err)
	}
	// 先下载到同目录的临时文件，校验通过后再改名替换，运行中的实例不受影响
	tmp := target + ".download"
	defer os.Remove(tmp)
	sum, err := downloadTo(ctx, client, asset.URL, tmp, func(done int64) {
		m.setInstall(func(st *installState) { st.Downloaded = done })
	})
	if err != nil {
		return "", err
	}
	m.setInstall(func(st *installState) { st.Stage = stageVerifying })
	if sum != want {
		return "", fmt.Errorf("%s 的 SHA-256 校验失败，下载内容可能被篡改或不完整", name)
	}
	if err := os.Chmod(tmp, 0o755); err != nil {
		return "", err
	}
	if _, quantum := probeVersion(tmp); !quantum {
		return "", fmt.Errorf("下载的程序无法在本机运行")
	}
	if err := os.Rename(tmp, target); err != nil {
		return "", fmt.Errorf("安装失败: %v", err)
	}
	return rel.TagName, nil
}

// fetchRelease 查询最新正式版的发布信息。
func (m *Module) fetchRelease(ctx context.Context, client *http.Client) (*release, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, m.releaseAPI, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("无法访问 GitHub: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub 返回 %s", resp.Status)
	}
	var rel release
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&rel); err != nil {
		return nil, fmt.Errorf("解析版本信息失败: %v", err)
	}
	if rel.TagName == "" {
		return nil, errors.New("解析版本信息失败: 缺少版本号")
	}
	return &rel, nil
}

// countWriter 统计写入的字节数并回调，用于报告下载进度。
type countWriter struct {
	n    atomic.Int64
	tick func(n int64)
}

func (c *countWriter) Write(p []byte) (int, error) {
	c.tick(c.n.Add(int64(len(p))))
	return len(p), nil
}

// downloadTo 把 rawURL 下载到 path，返回内容的 SHA-256（十六进制）。超过 maxBinarySize 视为异常。
func downloadTo(ctx context.Context, client *http.Client, rawURL, path string, progress func(done int64)) (string, error) {
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
	f, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return "", err
	}
	h := sha256.New()
	n, err := io.Copy(io.MultiWriter(f, h, &countWriter{tick: progress}), io.LimitReader(resp.Body, maxBinarySize+1))
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return "", fmt.Errorf("下载失败: %v", err)
	}
	if n > maxBinarySize {
		return "", fmt.Errorf("文件超过 %d MB，已中止", maxBinarySize>>20)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
