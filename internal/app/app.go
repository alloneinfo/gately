// Package app 负责组装配置、认证、模块与管理 HTTP 服务。
package app

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"mime/multipart"
	"net"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/wangwanjie/gately/internal/auth"
	"github.com/wangwanjie/gately/internal/config"
	"github.com/wangwanjie/gately/internal/httpx"
	"github.com/wangwanjie/gately/internal/logbuf"
	"github.com/wangwanjie/gately/internal/luckyimport"
	"github.com/wangwanjie/gately/internal/module"
	"github.com/wangwanjie/gately/internal/sysstat"
	"github.com/wangwanjie/gately/internal/version"
)

// 基础配置与账号配置的文件名
const (
	settingsName = "gately"
	accountName  = "account"
)

// DefaultListen 是管理界面默认监听地址，避开 Lucky 的 16601 以便迁移期间并存。
const DefaultListen = ":16680"

// Options 是启动参数。
type Options struct {
	ConfigDir string
	// Listen 非空时覆盖配置文件中的监听地址
	Listen string
	// Web 是嵌入的前端文件系统，可为 nil
	Web fs.FS
	// Modules 是要启用的功能模块
	Modules []module.Module
	// Certs 为管理界面 HTTPS 提供证书，可为 nil
	Certs CertProvider
}

type moduleEntry struct {
	mod  module.Module
	logs *logbuf.Buffer
}

// App 是一个运行中的 Gately 实例。
type App struct {
	opts       Options
	store      *config.Store
	settingsMu sync.RWMutex
	settings   Settings
	// fileListen 为配置文件中的监听地址（命令行 -listen 覆盖时用于回写）
	fileListen string
	// restartPending 表示有设置需要重启后生效；httpsReady 表示 HTTPS 管理服务已启动
	restartPending atomic.Bool
	httpsReady     atomic.Bool
	restartCh      chan struct{}
	auth           *auth.Manager
	sysLogs        *logbuf.Buffer
	modules        []*moduleEntry
	startTime      time.Time
	// hostStats 采集概览页展示的主机信息，保存上一次采样用于计算使用率与速率
	hostStats *sysstat.Collector
	logger    *log.Logger
	// pendingTOTP 是生成后尚未确认的两步验证密钥
	pendingMu   sync.Mutex
	pendingTOTP string
}

// New 加载配置并初始化所有模块，但不启动网络服务。
func New(opts Options) (*App, error) {
	store, err := config.Open(opts.ConfigDir)
	if err != nil {
		return nil, err
	}
	logger := log.New(os.Stderr, "", log.LstdFlags)
	a := &App{
		opts:      opts,
		store:     store,
		settings:  Settings{Listen: DefaultListen},
		sysLogs:   logbuf.New(defaultLogLines, logger),
		startTime: time.Now(),
		hostStats: sysstat.New(opts.ConfigDir),
		logger:    logger,
		restartCh: make(chan struct{}, 1),
	}
	if _, err := store.Load(settingsName, &a.settings); err != nil {
		return nil, err
	}
	a.settings.normalize()
	a.fileListen = a.settings.Listen
	if opts.Listen != "" {
		a.settings.Listen = opts.Listen
	}

	var acc auth.Account
	if _, err := store.Load(accountName, &acc); err != nil {
		return nil, err
	}
	a.auth = auth.NewManager(acc, func(acc auth.Account) error {
		return store.Save(accountName, acc)
	})
	generated, err := a.auth.EnsureInitialized()
	if err != nil {
		return nil, fmt.Errorf("初始化管理员账号失败: %w", err)
	}
	if generated != "" {
		// 初始密码只在首次启动时打印一次，登录后应立即修改
		a.logger.Printf("==== 已创建管理员账号 admin，初始密码: %s ，请登录后立即修改 ====", generated)
	}

	seen := map[string]bool{}
	for _, m := range opts.Modules {
		if seen[m.ID()] || m.ID() == settingsName || m.ID() == accountName {
			return nil, fmt.Errorf("模块 ID 冲突: %s", m.ID())
		}
		seen[m.ID()] = true
		entry := &moduleEntry{mod: m, logs: logbuf.New(defaultLogLines, logger)}
		if err := m.Init(module.Env{Store: store, Logs: entry.logs}); err != nil {
			return nil, fmt.Errorf("模块 %s 初始化失败: %w", m.ID(), err)
		}
		a.modules = append(a.modules, entry)
	}
	a.applyRuntime()
	return a, nil
}

// Handler 返回完整的 HTTP 处理器（API + 前端）。
func (a *App) Handler() http.Handler {
	mux := http.NewServeMux()
	protected := http.NewServeMux()

	mux.HandleFunc("POST /api/login", a.handleLogin)
	mux.HandleFunc("POST /api/logout", a.handleLogout)

	protected.HandleFunc("GET /api/session", a.handleSession)
	protected.HandleFunc("GET /api/status", a.handleStatus)
	protected.HandleFunc("PUT /api/account", a.handleAccount)
	protected.HandleFunc("GET /api/account/totp", a.handleTOTPStatus)
	protected.HandleFunc("POST /api/account/totp/setup", a.handleTOTPSetup)
	protected.HandleFunc("POST /api/account/totp/enable", a.handleTOTPEnable)
	protected.HandleFunc("POST /api/account/totp/disable", a.handleTOTPDisable)
	protected.HandleFunc("GET /api/logs", a.logsHandler(a.sysLogs))
	protected.HandleFunc("POST /api/import/lucky", a.handleLuckyImport)
	protected.HandleFunc("GET /api/system/settings", a.handleGetSettings)
	protected.HandleFunc("PUT /api/system/settings", a.handlePutSettings)
	protected.HandleFunc("POST /api/system/token", a.handleGenerateToken)
	protected.HandleFunc("GET /api/system/info", a.handleSystemInfo)
	protected.HandleFunc("GET /api/system/update", a.handleCheckUpdate)
	protected.HandleFunc("GET /api/system/backup", a.handleBackup)
	protected.HandleFunc("POST /api/system/restore", a.handleRestore)
	protected.HandleFunc("POST /api/system/restart", a.handleRestart)
	for _, e := range a.modules {
		prefix := "/api/" + e.mod.ID()
		protected.HandleFunc("GET "+prefix+"/logs", a.logsHandler(e.logs))
		protected.HandleFunc("DELETE "+prefix+"/logs", func(w http.ResponseWriter, r *http.Request) {
			e.logs.Clear()
			httpx.OK(w, map[string]bool{"ok": true})
		})
		e.mod.Routes(prefixRouter{mux: protected, prefix: prefix})
	}
	mux.Handle("/api/", a.requireAuth(protected))
	mux.Handle("/", a.staticHandler())
	return securityHeaders(a.accessControl(httpx.SameOrigin(mux)))
}

// Run 启动所有模块和管理服务，阻塞直到 ctx 取消。收到重启请求时停止全部服务后重新执行自身。
func (a *App) Run(parent context.Context) error {
	// 模块使用独立的可取消上下文：重启时先取消，确保后台协程退出后 Stop 不会一直等待
	ctx, cancelModules := context.WithCancel(parent)
	defer cancelModules()
	for _, e := range a.modules {
		if err := e.mod.Start(ctx); err != nil {
			// 单个模块失败不影响其他模块，错误写入日志供界面查看
			e.logs.Add(logbuf.LevelError, "", "模块启动失败: "+err.Error())
		}
	}
	s := a.settingsSnapshot()
	handler := a.Handler()
	servers := []*http.Server{{Addr: s.Listen, Handler: handler, ReadHeaderTimeout: 10 * time.Second}}
	errCh := make(chan error, 2)
	go func() {
		a.sysLogs.Add(logbuf.LevelInfo, "", "管理界面监听 "+s.Listen)
		errCh <- servers[0].ListenAndServe()
	}()
	if s.HTTPSListen != "" && a.opts.Certs != nil {
		tlsSrv := &http.Server{
			Addr: s.HTTPSListen, Handler: handler, ReadHeaderTimeout: 10 * time.Second,
			TLSConfig: &tls.Config{GetCertificate: a.opts.Certs.GetCertificate, MinVersion: tls.VersionTLS12},
		}
		servers = append(servers, tlsSrv)
		ln, err := net.Listen("tcp", s.HTTPSListen)
		if err != nil {
			// HTTPS 失败不影响 HTTP 管理界面
			a.sysLogs.Add(logbuf.LevelError, "", "HTTPS 管理界面监听失败: "+err.Error())
		} else {
			a.httpsReady.Store(true)
			a.sysLogs.Add(logbuf.LevelInfo, "", "HTTPS 管理界面监听 "+s.HTTPSListen)
			go func() {
				if err := tlsSrv.ServeTLS(ln, "", ""); err != nil && !errors.Is(err, http.ErrServerClosed) {
					a.sysLogs.Add(logbuf.LevelError, "", "HTTPS 管理界面退出: "+err.Error())
				}
			}()
		}
	}

	restart := false
	select {
	case <-ctx.Done():
	case <-a.restartCh:
		restart = true
	case err := <-errCh:
		if !errors.Is(err, http.ErrServerClosed) {
			for _, srv := range servers[1:] {
				_ = srv.Close()
			}
			cancelModules()
			a.stopModules()
			return err
		}
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	for _, srv := range servers {
		_ = srv.Shutdown(shutdownCtx)
	}
	cancelModules()
	a.stopModules()
	if restart {
		a.logger.Printf("正在重启…")
		restartProcess()
	}
	return nil
}

// requireAuth 在会话校验之外接受 API 令牌。
func (a *App) requireAuth(next http.Handler) http.Handler {
	session := a.auth.Require(next)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if a.apiTokenOK(r) {
			next.ServeHTTP(w, r.WithContext(auth.WithUser(r.Context(), "api-token")))
			return
		}
		session.ServeHTTP(w, r)
	})
}

func (a *App) stopModules() {
	// 逆序停止，和启动顺序对称
	for i := len(a.modules) - 1; i >= 0; i-- {
		if err := a.modules[i].mod.Stop(); err != nil {
			a.logger.Printf("停止模块 %s 失败: %v", a.modules[i].mod.ID(), err)
		}
	}
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	// Code 为两步验证码，未启用时忽略
	Code string `json:"code"`
}

func (a *App) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "%v", err)
		return
	}
	ip := httpx.ClientIP(r)
	token, exp, err := a.auth.Login(ip, req.Username, req.Password, req.Code)
	if errors.Is(err, auth.ErrTOTPRequired) {
		// 密码正确但缺少验证码，提示前端显示验证码输入框
		httpx.WriteJSON(w, http.StatusUnauthorized, map[string]any{"error": err.Error(), "totpRequired": true})
		return
	}
	if err != nil {
		a.sysLogs.Add(logbuf.LevelWarn, "auth", fmt.Sprintf("来自 %s 的登录失败: %v", ip, err))
		status := http.StatusUnauthorized
		if errors.Is(err, auth.ErrLocked) {
			status = http.StatusTooManyRequests
		}
		httpx.Error(w, status, "%v", err)
		return
	}
	a.sysLogs.Add(logbuf.LevelInfo, "auth", fmt.Sprintf("%s 从 %s 登录", req.Username, ip))
	http.SetCookie(w, &http.Cookie{
		Name:     auth.CookieName,
		Value:    token,
		Path:     "/",
		Expires:  exp,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   r.TLS != nil,
	})
	httpx.OK(w, map[string]any{"username": req.Username, "expires": exp})
}

func (a *App) handleLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{Name: auth.CookieName, Value: "", Path: "/", MaxAge: -1, HttpOnly: true, SameSite: http.SameSiteStrictMode})
	httpx.OK(w, map[string]bool{"ok": true})
}

func (a *App) handleSession(w http.ResponseWriter, r *http.Request) {
	httpx.OK(w, map[string]string{"username": auth.UserFromContext(r.Context())})
}

type moduleInfo struct {
	ID      string               `json:"id"`
	Title   string               `json:"title"`
	Summary []module.SummaryItem `json:"summary,omitempty"`
}

func (a *App) handleStatus(w http.ResponseWriter, r *http.Request) {
	mods := make([]moduleInfo, 0, len(a.modules))
	for _, e := range a.modules {
		info := moduleInfo{ID: e.mod.ID(), Title: e.mod.Title()}
		if sm, ok := e.mod.(module.Summarizer); ok {
			info.Summary = sm.Summary()
		}
		mods = append(mods, info)
	}
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	httpx.OK(w, map[string]any{
		"version":    version.Version,
		"commit":     version.Commit,
		"buildDate":  version.BuildDate,
		"goVersion":  runtime.Version(),
		"os":         runtime.GOOS,
		"arch":       runtime.GOARCH,
		"startTime":  a.startTime,
		"goroutines": runtime.NumGoroutine(),
		"memAlloc":   mem.Alloc,
		"modules":    mods,
		"timeZone":   zoneLabel(),
		"configDir":  a.opts.ConfigDir,
		"host":       a.hostStats.Snapshot(),
	})
}

// zoneLabel 返回时区名称；未显式设置时区（显示为 Local）时改用缩写与 UTC 偏移，例如 CST UTC+08:00。
func zoneLabel() string {
	name, offset := time.Now().Zone()
	sign := "+"
	if offset < 0 {
		sign = "-"
		offset = -offset
	}
	utc := fmt.Sprintf("UTC%s%02d:%02d", sign, offset/3600, offset%3600/60)
	if loc := time.Local.String(); loc != "Local" {
		return loc + " (" + utc + ")"
	}
	return name + " " + utc
}

type accountRequest struct {
	OldPassword string `json:"oldPassword"`
	Username    string `json:"username"`
	NewPassword string `json:"newPassword"`
}

func (a *App) handleAccount(w http.ResponseWriter, r *http.Request) {
	var req accountRequest
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "%v", err)
		return
	}
	if err := a.auth.ChangeCredentials(req.OldPassword, req.Username, req.NewPassword); err != nil {
		httpx.Error(w, http.StatusBadRequest, "%v", err)
		return
	}
	a.sysLogs.Add(logbuf.LevelInfo, "auth", "管理员账号已修改，所有会话已失效")
	httpx.OK(w, map[string]bool{"ok": true})
}

func (a *App) handleTOTPStatus(w http.ResponseWriter, r *http.Request) {
	httpx.OK(w, map[string]bool{"enabled": a.auth.TOTPEnabled()})
}

// handleTOTPSetup 生成待确认的密钥，确认验证码后才真正启用。
func (a *App) handleTOTPSetup(w http.ResponseWriter, r *http.Request) {
	secret := auth.NewTOTPSecret()
	a.pendingMu.Lock()
	a.pendingTOTP = secret
	a.pendingMu.Unlock()
	httpx.OK(w, map[string]string{"secret": secret, "url": auth.TOTPURL(secret, a.auth.Username())})
}

type totpRequest struct {
	Password string `json:"password"`
	Code     string `json:"code"`
}

func (a *App) handleTOTPEnable(w http.ResponseWriter, r *http.Request) {
	var req totpRequest
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "%v", err)
		return
	}
	a.pendingMu.Lock()
	secret := a.pendingTOTP
	a.pendingMu.Unlock()
	if secret == "" {
		httpx.Error(w, http.StatusBadRequest, "请先生成密钥")
		return
	}
	if !auth.ValidateTOTP(secret, req.Code, time.Now()) {
		httpx.Error(w, http.StatusBadRequest, "验证码错误，请确认验证器时间同步")
		return
	}
	if err := a.auth.SetTOTP(req.Password, "", secret); err != nil {
		httpx.Error(w, http.StatusBadRequest, "%v", err)
		return
	}
	a.pendingMu.Lock()
	a.pendingTOTP = ""
	a.pendingMu.Unlock()
	a.sysLogs.Add(logbuf.LevelInfo, "auth", "已启用两步验证")
	httpx.OK(w, map[string]bool{"ok": true})
}

func (a *App) handleTOTPDisable(w http.ResponseWriter, r *http.Request) {
	var req totpRequest
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "%v", err)
		return
	}
	if err := a.auth.SetTOTP(req.Password, req.Code, ""); err != nil {
		httpx.Error(w, http.StatusBadRequest, "%v", err)
		return
	}
	a.sysLogs.Add(logbuf.LevelWarn, "auth", "已关闭两步验证")
	httpx.OK(w, map[string]bool{"ok": true})
}

// handleLuckyImport 接收 multipart 上传的 .lkcf 文件或 Lucky 导出的整站配置 zip（字段名 files），
// 分发给实现了 luckyimport.Importer 的模块。?dryRun=1 时只预览不写入，?enable=1 时保留 Lucky 中的启用状态。
func (a *App) handleLuckyImport(w http.ResponseWriter, r *http.Request) {
	// 限制总请求体与文件数量，防止超大上传占用内存与磁盘
	r.Body = http.MaxBytesReader(w, r.Body, 64<<20)
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		httpx.Error(w, http.StatusBadRequest, "读取上传文件失败: %v", err)
		return
	}
	opts := luckyimport.Options{DryRun: r.URL.Query().Get("dryRun") == "1", KeepEnabled: r.URL.Query().Get("enable") == "1"}
	type fileResult struct {
		File    string                `json:"file"`
		Error   string                `json:"error,omitempty"`
		Reports []*luckyimport.Report `json:"reports"`
	}
	results := []fileResult{}
	if len(r.MultipartForm.File["files"]) > 64 {
		httpx.Error(w, http.StatusBadRequest, "一次最多导入 64 个文件")
		return
	}
	var entries []luckyimport.Entry
	for _, fh := range r.MultipartForm.File["files"] {
		data, err := readUpload(fh)
		if err == nil && luckyimport.IsArchive(data) {
			var sub []luckyimport.Entry
			if sub, err = luckyimport.ExpandArchive(fh.Filename, data); err == nil {
				entries = append(entries, sub...)
				continue
			}
		}
		if err != nil {
			results = append(results, fileResult{File: fh.Filename, Error: err.Error(), Reports: []*luckyimport.Report{}})
			continue
		}
		entries = append(entries, luckyimport.Entry{Name: fh.Filename, Data: data})
	}
	for _, en := range entries {
		res := fileResult{File: en.Name, Reports: []*luckyimport.Report{}}
		doc, err := luckyimport.ParseAny(en.Data)
		if err == nil {
			for _, e := range a.modules {
				imp, ok := e.mod.(luckyimport.Importer)
				if !ok {
					continue
				}
				rep, ierr := imp.ImportLucky(doc, opts)
				if ierr != nil {
					err = fmt.Errorf("%s: %w", e.mod.Title(), ierr)
					break
				}
				if rep != nil {
					res.Reports = append(res.Reports, rep)
					if !opts.DryRun {
						e.logs.Add(logbuf.LevelInfo, "import", fmt.Sprintf("从 Lucky 导入 %d/%d 条", rep.Imported, rep.Found))
					}
				}
			}
			if err == nil {
				if rep := a.importLuckyBase(doc, opts); rep != nil {
					res.Reports = append(res.Reports, rep)
				}
			}
		}
		if err != nil {
			res.Error = err.Error()
		}
		results = append(results, res)
	}
	httpx.OK(w, results)
}

func readUpload(fh *multipart.FileHeader) ([]byte, error) {
	f, err := fh.Open()
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(io.LimitReader(f, 32<<20))
}

// logsHandler 支持 ?since=<seq> 增量拉取。
func (a *App) logsHandler(buf *logbuf.Buffer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var since uint64
		if s := r.URL.Query().Get("since"); s != "" {
			v, err := strconv.ParseUint(s, 10, 64)
			if err != nil {
				httpx.Error(w, http.StatusBadRequest, "since 参数无效")
				return
			}
			since = v
		}
		httpx.OK(w, buf.Since(since))
	}
}

// staticHandler 提供前端单页应用，未知路径回退到 index.html。
func (a *App) staticHandler() http.Handler {
	if a.opts.Web == nil {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "前端未构建：请在 web 目录执行 pnpm install && pnpm build 后重新编译", http.StatusNotFound)
		})
	}
	files := http.FileServer(http.FS(a.opts.Web))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/")
		if name != "" {
			if _, err := fs.Stat(a.opts.Web, name); err == nil {
				// 带哈希的构建产物可长期缓存
				if strings.HasPrefix(name, "assets/") {
					w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
				}
				files.ServeHTTP(w, r)
				return
			}
		}
		index, err := fs.ReadFile(a.opts.Web, "index.html")
		if err != nil {
			http.Error(w, "前端未构建：请在 web 目录执行 pnpm install && pnpm build 后重新编译", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		_, _ = w.Write(index)
	})
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "same-origin")
		next.ServeHTTP(w, r)
	})
}

// prefixRouter 把模块的相对路由挂到 /api/<ID> 前缀下。
type prefixRouter struct {
	mux    *http.ServeMux
	prefix string
}

func (p prefixRouter) HandleFunc(pattern string, h http.HandlerFunc) {
	method, path, found := strings.Cut(pattern, " ")
	if !found {
		p.mux.HandleFunc(p.prefix+pattern, h)
		return
	}
	p.mux.HandleFunc(method+" "+p.prefix+path, h)
}
