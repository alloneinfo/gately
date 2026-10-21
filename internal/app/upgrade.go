package app

import (
	"archive/tar"
	"bufio"
	"bytes"
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
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/wangwanjie/gately/internal/httpx"
	"github.com/wangwanjie/gately/internal/logbuf"
	"github.com/wangwanjie/gately/internal/version"
)

// ---- 在线升级 ----
//
// 只在 Linux 二进制部署（systemd、PVE LXC、OpenWrt 等）中开放：下载与本机平台对应的发布包，
// 核对 checksums.txt 中的 SHA-256，试运行新二进制确认能在本机执行并读出版本号，
// 再原地替换可执行文件并以相同参数重启。旧版本保留为同目录下的 <文件名>.old，便于手动回滚。
// Docker 镜像与飞牛 fnOS 安装包分别由镜像更新与应用中心负责升级，不提供在线升级。

// 不支持在线升级的原因，前端据此显示对应的升级方式
const (
	upgradeReasonOS       = "os"       // 非 Linux 系统
	upgradeReasonDocker   = "docker"   // 运行在 Docker / Podman 容器中
	upgradeReasonFnOS     = "fnos"     // 飞牛 fnOS 安装包
	upgradeReasonReadonly = "readonly" // 可执行文件所在目录不可写
)

const (
	// maxUpgradeArchive 为发布压缩包的大小上限，maxUpgradeBinary 为解压后二进制的大小上限
	maxUpgradeArchive = 128 << 20
	maxUpgradeBinary  = 256 << 20
	// upgradeTimeout 为一次在线升级（查询版本、下载、校验）的总时长上限
	upgradeTimeout = 10 * time.Minute
)

// 以下变量在测试中可替换
var (
	upgradeGOOS       = runtime.GOOS
	upgradeGOARCH     = runtime.GOARCH
	upgradeExecutable = os.Executable
	// containerMarkers 存在任一文件即视为运行在 Docker / Podman 容器中。
	// 这两个文件由容器运行时在创建容器时生成，PVE 用 OCI 镜像创建的 LXC 中不存在，仍可在线升级
	containerMarkers = []string{"/.dockerenv", "/run/.containerenv"}
)

// upgradeReasonText 是拒绝在线升级时返回的说明。
var upgradeReasonText = map[string]string{
	upgradeReasonOS:       "在线升级只支持 Linux，请下载新版本后手动替换程序",
	upgradeReasonDocker:   "运行在容器中，请拉取新镜像后重建容器",
	upgradeReasonFnOS:     "飞牛 fnOS 安装包请在应用中心手动安装新版 .fpk",
	upgradeReasonReadonly: "程序所在目录不可写，无法在线升级",
}

// upgradeSupport 描述当前部署方式能否在线升级。
type upgradeSupport struct {
	Supported bool   `json:"supported"`
	Reason    string `json:"reason,omitempty"`
	// Executable 为将被替换的可执行文件路径
	Executable string `json:"executable,omitempty"`
}

func (a *App) upgradeSupport() upgradeSupport {
	if upgradeGOOS != "linux" {
		return upgradeSupport{Reason: upgradeReasonOS}
	}
	for _, m := range containerMarkers {
		if fileExists(m) {
			return upgradeSupport{Reason: upgradeReasonDocker}
		}
	}
	exe, err := upgradeExecutable()
	if err != nil {
		return upgradeSupport{Reason: upgradeReasonReadonly}
	}
	// 可执行文件是符号链接时替换链接指向的真实文件
	if real, err := filepath.EvalSymlinks(exe); err == nil {
		exe = real
	}
	// 飞牛安装包以统一网关方式运行，程序位于 /var/apps/<应用>/ 下，运行时带有 TRIM_APPDEST 环境变量
	if a.opts.GatewaySocket != "" || os.Getenv("TRIM_APPDEST") != "" || strings.HasPrefix(exe, "/var/apps/") {
		return upgradeSupport{Reason: upgradeReasonFnOS, Executable: exe}
	}
	if !dirWritable(filepath.Dir(exe)) {
		return upgradeSupport{Reason: upgradeReasonReadonly, Executable: exe}
	}
	return upgradeSupport{Supported: true, Executable: exe}
}

// dirWritable 通过创建临时文件判断目录是否可写，比检查权限位更能反映只读挂载等情况。
func dirWritable(dir string) bool {
	f, err := os.CreateTemp(dir, ".gately-wtest-*")
	if err != nil {
		return false
	}
	name := f.Name()
	f.Close()
	os.Remove(name)
	return true
}

// validateProxyURL 校验代理地址，支持 http、https 与 socks5。
func validateProxyURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https" && u.Scheme != "socks5" && u.Scheme != "socks5h") {
		return fmt.Errorf("更新代理地址需为 http://、https:// 或 socks5:// 开头的地址，例如 http://127.0.0.1:7890")
	}
	return nil
}

// updateClient 返回访问 GitHub 检查与下载新版本使用的客户端：设置了更新代理时走该代理，
// 否则沿用默认客户端（遵循 HTTPS_PROXY 等环境变量）。
func (a *App) updateClient() *http.Client {
	p := a.settingsSnapshot().UpdateProxy
	if p == "" {
		return http.DefaultClient
	}
	u, err := url.Parse(p)
	if err != nil {
		// 保存设置时已校验，这里仅防御异常配置
		return http.DefaultClient
	}
	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.Proxy = http.ProxyURL(u)
	return &http.Client{Transport: tr}
}

// releaseAPI 是检查更新使用的地址，只在用户点击“检查更新”或执行在线升级时访问。
var releaseAPI = "https://api.github.com/repos/wangwanjie/gately/releases/latest"

type releaseAsset struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
	Size int64  `json:"size"`
}

type releaseInfo struct {
	TagName     string         `json:"tag_name"`
	HTMLURL     string         `json:"html_url"`
	Body        string         `json:"body"`
	PublishedAt time.Time      `json:"published_at"`
	Assets      []releaseAsset `json:"assets"`
}

func (r *releaseInfo) asset(name string) *releaseAsset {
	for i := range r.Assets {
		if r.Assets[i].Name == name {
			return &r.Assets[i]
		}
	}
	return nil
}

// fetchLatestRelease 查询 GitHub 上的最新正式版本。
func fetchLatestRelease(ctx context.Context, client *http.Client) (*releaseInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, releaseAPI, nil)
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
	var rel releaseInfo
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&rel); err != nil {
		return nil, fmt.Errorf("解析版本信息失败: %v", err)
	}
	return &rel, nil
}

// downloadBytes 下载文件到内存，超过 limit 字节视为异常。
func downloadBytes(ctx context.Context, client *http.Client, rawURL string, limit int64) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("下载 %s 失败: %v", urlBase(rawURL), err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("下载 %s 失败: %s", urlBase(rawURL), resp.Status)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return nil, fmt.Errorf("下载 %s 失败: %v", urlBase(rawURL), err)
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("%s 超过 %d MB，已中止", urlBase(rawURL), limit>>20)
	}
	return data, nil
}

// urlBase 返回 URL 路径的最后一段，用于错误提示。
func urlBase(rawURL string) string {
	if u, err := url.Parse(rawURL); err == nil {
		return filepath.Base(u.Path)
	}
	return rawURL
}

// checksumFor 从 checksums.txt（sha256sum 输出格式）中找出指定文件的 SHA-256。
func checksumFor(sums []byte, name string) (string, bool) {
	sc := bufio.NewScanner(bytes.NewReader(sums))
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) == 2 && strings.TrimPrefix(fields[1], "*") == name {
			return strings.ToLower(fields[0]), true
		}
	}
	return "", false
}

// extractBinary 从升级包中取出 gately 可执行文件：支持发布用的 tar.gz，也接受直接上传的二进制。
func extractBinary(data []byte) ([]byte, error) {
	if !bytes.HasPrefix(data, []byte{0x1f, 0x8b}) {
		// 非 gzip 视为二进制本身，能否在本机运行由后续试运行确认
		return data, nil
	}
	zr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("升级包不是有效的 tar.gz: %v", err)
	}
	tr := tar.NewReader(zr)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("升级包不是有效的 tar.gz: %v", err)
		}
		if h.Typeflag != tar.TypeReg || filepath.Base(h.Name) != "gately" {
			continue
		}
		bin, err := io.ReadAll(io.LimitReader(tr, maxUpgradeBinary+1))
		if err != nil {
			return nil, fmt.Errorf("解压升级包失败: %v", err)
		}
		if len(bin) > maxUpgradeBinary {
			return nil, errors.New("升级包中的程序过大，已中止")
		}
		return bin, nil
	}
	return nil, errors.New("升级包中没有 gately 可执行文件")
}

// probeBinary 以 -version 试运行新二进制，确认它能在本机执行（架构正确、文件完整）并返回其版本号。
// 使用 -version 参数：即使是不认识该参数的旧版本也只会报错退出，不会启动第二个服务实例。
func probeBinary(ctx context.Context, path string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, path, "-version").Output()
	if err != nil {
		return "", fmt.Errorf("新版本无法在本机运行，可能与本机架构不符或文件已损坏: %v", err)
	}
	var v struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(bytes.TrimSpace(out), &v); err != nil || v.Version == "" {
		return "", errors.New("无法读取新版本的版本号，可能不是 Gately 程序")
	}
	return v.Version, nil
}

// installBinary 把新二进制写入可执行文件所在目录，试运行确认可用后替换原文件。
// 原文件以硬链接（不支持时复制）保留为 <文件名>.old；新文件经同目录 rename 原子替换，
// 运行中的进程不受影响，随后的重启会执行新文件。
func installBinary(ctx context.Context, exe string, bin []byte) (string, error) {
	dir := filepath.Dir(exe)
	mode := os.FileMode(0o755)
	if fi, err := os.Stat(exe); err == nil {
		mode = fi.Mode().Perm() | 0o111
	}
	f, err := os.CreateTemp(dir, ".gately-upgrade-*")
	if err != nil {
		return "", fmt.Errorf("无法在 %s 中写入新版本: %v", dir, err)
	}
	tmp := f.Name()
	// 任一步骤失败都删除临时文件
	installed := false
	defer func() {
		if !installed {
			os.Remove(tmp)
		}
	}()
	_, err = f.Write(bin)
	if err == nil {
		err = f.Sync()
	}
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		if errors.Is(err, syscall.ENOSPC) {
			return "", fmt.Errorf("%s 所在磁盘空间不足，升级需要约 %d MB 可用空间", dir, len(bin)>>20+1)
		}
		return "", fmt.Errorf("写入新版本失败: %v", err)
	}
	if err := os.Chmod(tmp, mode); err != nil {
		return "", fmt.Errorf("写入新版本失败: %v", err)
	}
	ver, err := probeBinary(ctx, tmp)
	if err != nil {
		return "", err
	}
	old := exe + ".old"
	if err := os.Remove(old); err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("无法删除上次保留的旧版本 %s: %v", old, err)
	}
	if err := os.Link(exe, old); err != nil {
		if err := copyFile(exe, old, mode); err != nil {
			os.Remove(old)
			return "", fmt.Errorf("备份当前版本到 %s 失败: %v", old, err)
		}
	}
	if err := os.Rename(tmp, exe); err != nil {
		return "", fmt.Errorf("替换 %s 失败: %v", exe, err)
	}
	installed = true
	return ver, nil
}

func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

// beginUpgrade 检查能否升级并占用升级锁；返回 false 时已写出错误响应。
func (a *App) beginUpgrade(w http.ResponseWriter) (upgradeSupport, bool) {
	sup := a.upgradeSupport()
	if !sup.Supported {
		httpx.Error(w, http.StatusConflict, "%s", upgradeReasonText[sup.Reason])
		return sup, false
	}
	if !a.upgrading.CompareAndSwap(false, true) {
		httpx.Error(w, http.StatusConflict, "正在升级中，请稍候")
		return sup, false
	}
	return sup, true
}

// finishUpgrade 记录日志、返回结果并安排重启。
func (a *App) finishUpgrade(w http.ResponseWriter, exe, to string) {
	a.sysLogs.Add(logbuf.LevelWarn, "system", fmt.Sprintf("已升级 %s → %s，旧版本保留为 %s，正在重启", version.Version, to, exe+".old"))
	httpx.OK(w, map[string]any{"from": version.Version, "to": to, "executable": exe, "previous": exe + ".old"})
	a.scheduleRestart()
}

// handleUpgrade 在线升级到 GitHub 上的最新版本。
func (a *App) handleUpgrade(w http.ResponseWriter, r *http.Request) {
	sup, ok := a.beginUpgrade(w)
	if !ok {
		return
	}
	defer a.upgrading.Store(false)
	ctx, cancel := context.WithTimeout(r.Context(), upgradeTimeout)
	defer cancel()
	client := a.updateClient()
	rel, err := fetchLatestRelease(ctx, client)
	if err != nil {
		httpx.Error(w, http.StatusBadGateway, "%v", err)
		return
	}
	if !versionNewer(rel.TagName, version.Version) {
		httpx.Error(w, http.StatusConflict, "当前版本 %s 已是最新，无需升级", version.Version)
		return
	}
	name := fmt.Sprintf("gately_%s_%s_%s.tar.gz", rel.TagName, upgradeGOOS, upgradeGOARCH)
	pkg, sums := rel.asset(name), rel.asset("checksums.txt")
	if pkg == nil || sums == nil {
		httpx.Error(w, http.StatusBadGateway, "版本 %s 中缺少 %s 或 checksums.txt", rel.TagName, name)
		return
	}
	sumData, err := downloadBytes(ctx, client, sums.URL, 1<<20)
	if err != nil {
		httpx.Error(w, http.StatusBadGateway, "%v", err)
		return
	}
	want, ok := checksumFor(sumData, name)
	if !ok {
		httpx.Error(w, http.StatusBadGateway, "checksums.txt 中没有 %s 的校验值", name)
		return
	}
	data, err := downloadBytes(ctx, client, pkg.URL, maxUpgradeArchive)
	if err != nil {
		httpx.Error(w, http.StatusBadGateway, "%v", err)
		return
	}
	sum := sha256.Sum256(data)
	if got := hex.EncodeToString(sum[:]); got != want {
		httpx.Error(w, http.StatusBadGateway, "%s 的 SHA-256 校验失败，下载内容可能被篡改或不完整", name)
		return
	}
	bin, err := extractBinary(data)
	if err != nil {
		httpx.Error(w, http.StatusBadGateway, "%v", err)
		return
	}
	to, err := installBinary(ctx, sup.Executable, bin)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "%v", err)
		return
	}
	a.finishUpgrade(w, sup.Executable, to)
}

// handleUpgradeUpload 用上传的升级包（发布页的 tar.gz 或解压后的 gately 二进制）升级。
// 不限制版本高低，可用于离线环境或回退到旧版本；能否在本机运行由试运行确认。
func (a *App) handleUpgradeUpload(w http.ResponseWriter, r *http.Request) {
	sup, ok := a.beginUpgrade(w)
	if !ok {
		return
	}
	defer a.upgrading.Store(false)
	r.Body = http.MaxBytesReader(w, r.Body, maxUpgradeArchive+(1<<20))
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		httpx.Error(w, http.StatusBadRequest, "读取上传文件失败: %v", err)
		return
	}
	fhs := r.MultipartForm.File["file"]
	if len(fhs) != 1 {
		httpx.Error(w, http.StatusBadRequest, "请上传一个升级包")
		return
	}
	f, err := fhs[0].Open()
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "读取上传文件失败: %v", err)
		return
	}
	data, err := io.ReadAll(io.LimitReader(f, maxUpgradeArchive+1))
	f.Close()
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "读取上传文件失败: %v", err)
		return
	}
	if len(data) > maxUpgradeArchive {
		httpx.Error(w, http.StatusBadRequest, "升级包超过 %d MB", maxUpgradeArchive>>20)
		return
	}
	bin, err := extractBinary(data)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "%v", err)
		return
	}
	to, err := installBinary(r.Context(), sup.Executable, bin)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "%v", err)
		return
	}
	a.finishUpgrade(w, sup.Executable, to)
}

// versionNewer 判断版本 a 是否比 b 新。版本格式为 v主.次.修订[-预发布]，无法解析（如 dev）时返回 false。
func versionNewer(a, b string) bool {
	va, okA := parseVersion(a)
	vb, okB := parseVersion(b)
	if !okA || !okB {
		return false
	}
	for i := 0; i < 3; i++ {
		if va.nums[i] != vb.nums[i] {
			return va.nums[i] > vb.nums[i]
		}
	}
	// 主版本号相同时，正式版比预发布版新；两者都是预发布版时按字符串比较
	switch {
	case va.pre == vb.pre:
		return false
	case va.pre == "":
		return true
	case vb.pre == "":
		return false
	}
	return va.pre > vb.pre
}

type parsedVersion struct {
	nums [3]int
	pre  string
}

func parseVersion(s string) (parsedVersion, bool) {
	var v parsedVersion
	s = strings.TrimPrefix(strings.TrimSpace(s), "v")
	if i := strings.IndexByte(s, '+'); i >= 0 {
		s = s[:i]
	}
	if i := strings.IndexByte(s, '-'); i >= 0 {
		s, v.pre = s[:i], s[i+1:]
	}
	parts := strings.Split(s, ".")
	if len(parts) != 3 {
		return v, false
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return v, false
		}
		v.nums[i] = n
	}
	return v, true
}
