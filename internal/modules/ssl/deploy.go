package ssl

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/wangwanjie/gately/internal/events"
	"github.com/wangwanjie/gately/internal/httpx"
	"github.com/wangwanjie/gately/internal/i18n"
	"github.com/wangwanjie/gately/internal/script"
)

// 部署目标类型
const (
	// DeploySSH 经 Web 终端中的 SSH 连接把证书上传到远程主机，再执行命令
	DeploySSH = "ssh"
	// DeployDocker 重启本机 Docker 容器，让挂载了映射目录的服务加载新证书
	DeployDocker = "docker"
)

// 部署参数，可在测试中调整
var (
	deployTimeout = 2 * time.Minute
	deployBackoff = time.Hour
)

// Deploy 是证书内容变化后自动执行的部署目标。
type Deploy struct {
	Enable bool   `json:"enable"`
	Type   string `json:"type"`
	Remark string `json:"remark"`
	// Connection 为 Web 终端中 SSH 连接的 key，复用其地址、认证、代理与已记录的主机密钥
	Connection string `json:"connection"`
	// Dir 为远程主机上的证书目录（绝对路径），文件名与证书映射相同：<备注>.crt/.key/.pem/_issuerCertificate.crt
	Dir string `json:"dir"`
	// Command 为上传后在远程主机执行的命令，可使用 $certName、$certDir、$certFile、$keyFile
	Command string `json:"command"`
	// Containers 为要重启的本机容器名称或 ID
	Containers []string `json:"containers"`
	// DeployedHash 为最近一次成功部署时证书与目标设置的摘要，与当前不同则需要重新部署
	DeployedHash string    `json:"deployedHash,omitempty"`
	DeployedAt   time.Time `json:"deployedAt,omitzero"`
}

// DeployState 是部署目标的运行状态（不持久化），与 Cert.Deploys 按下标对应。
type DeployState struct {
	Running   bool      `json:"running"`
	LastError string    `json:"lastError,omitempty"`
	NextRetry time.Time `json:"nextRetry,omitzero"`
}

// RemoteFile 是部署时上传的文件。
type RemoteFile struct {
	Name string
	Data []byte
	Mode os.FileMode
}

// SSHTarget 是可选的 SSH 连接。
type SSHTarget struct {
	Key     string `json:"key"`
	Name    string `json:"name"`
	Host    string `json:"host"`
	Trusted bool   `json:"trusted"`
}

// Deployers 由主程序注入，连接 Web 终端与 Docker 模块，避免证书模块直接依赖它们。
type Deployers struct {
	// Push 经 SSH 连接上传文件并执行命令，返回命令输出
	Push func(ctx context.Context, conn, dir string, files []RemoteFile, command string) (string, error)
	// Restart 重启本机容器
	Restart func(ctx context.Context, names []string) error
	// SSHTargets 列出可用的 SSH 连接
	SSHTargets func() []SSHTarget
	// Containers 列出本机容器名称
	Containers func(ctx context.Context) ([]string, error)
}

// SetDeployers 注入部署能力，需在 Start 之前调用。
func (m *Module) SetDeployers(d Deployers) {
	m.mu.Lock()
	m.deployers = d
	m.mu.Unlock()
}

// validate 校验部署目标并整理字段。
func (d *Deploy) validate(remark string) error {
	d.Remark = strings.TrimSpace(d.Remark)
	switch d.Type {
	case DeploySSH:
		if remark == "" {
			return errors.New("部署到 SSH 主机时备注不能为空（用作文件名）")
		}
		if strings.TrimSpace(d.Connection) == "" {
			return errors.New("请选择部署使用的 SSH 连接")
		}
		d.Dir = strings.TrimSpace(d.Dir)
		if !path.IsAbs(d.Dir) {
			return errors.New("远程证书目录需为绝对路径")
		}
		d.Dir = path.Clean(d.Dir)
		d.Containers = nil
	case DeployDocker:
		var names []string
		for _, n := range d.Containers {
			if n = strings.TrimSpace(n); n != "" && !strings.ContainsAny(n, "/ ") {
				names = append(names, n)
			}
		}
		if len(names) == 0 {
			return errors.New("请选择要重启的容器")
		}
		d.Containers = names
		d.Connection, d.Dir, d.Command = "", "", ""
	default:
		return fmt.Errorf("未知的部署方式 %q", d.Type)
	}
	return nil
}

// deployHash 计算证书内容与部署目标设置的摘要，任何一方变化都会触发重新部署。
func deployHash(c *Cert, d *Deploy) string {
	h := sha256.New()
	for _, s := range []string{c.CertPEM, c.KeyPEM, c.IssuerPEM, c.fileBase(), d.Type, d.Connection, d.Dir, d.Command, strings.Join(d.Containers, "\n")} {
		h.Write([]byte(s))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}

// deployJob 是一次部署所需的快照，在锁外执行。
type deployJob struct {
	certKey string
	index   int
	hash    string
	target  Deploy
	label   string
	files   []RemoteFile
	vars    map[string]string
}

func (m *Module) deployStateLocked(key string, i int) *DeployState {
	st := m.stateLocked(key)
	for len(st.Deploys) <= i {
		st.Deploys = append(st.Deploys, DeployState{})
	}
	return &st.Deploys[i]
}

// newDeployJobLocked 生成部署快照。
func newDeployJobLocked(c *Cert, i int) deployJob {
	d := c.Deploys[i]
	job := deployJob{certKey: c.Key, index: i, hash: deployHash(c, &d), target: d, label: label(c)}
	if d.Type == DeploySSH {
		for name, data := range mappingFiles(c) {
			mode := os.FileMode(0o644)
			if strings.HasSuffix(name, ".key") {
				mode = 0o600
			}
			job.files = append(job.files, RemoteFile{Name: name, Data: data, Mode: mode})
		}
		job.vars = map[string]string{
			"certName": c.fileBase(),
			"certDir":  d.Dir,
			"certFile": path.Join(d.Dir, c.fileBase()+".crt"),
			"keyFile":  path.Join(d.Dir, c.fileBase()+".key"),
		}
	}
	return job
}

// scheduleDeploysLocked 在后台执行证书内容或目标设置已变化的部署；失败的目标在退避时间内不重试。
func (m *Module) scheduleDeploysLocked(c *Cert) {
	if !c.Enable || c.CertPEM == "" || c.KeyPEM == "" || m.ctx == nil {
		return
	}
	now := time.Now()
	for i := range c.Deploys {
		d := &c.Deploys[i]
		if !d.Enable || d.DeployedHash == deployHash(c, d) {
			continue
		}
		ds := m.deployStateLocked(c.Key, i)
		if ds.Running || now.Before(ds.NextRetry) {
			continue
		}
		ds.Running = true
		job := newDeployJobLocked(c, i)
		ctx := m.ctx
		m.wg.Add(1)
		go func() {
			defer m.wg.Done()
			m.finishDeploy(job, m.runDeploy(ctx, job))
		}()
	}
}

// runDeploy 执行一次部署，不持有锁。
func (m *Module) runDeploy(parent context.Context, job deployJob) error {
	m.mu.Lock()
	dep := m.deployers
	m.mu.Unlock()
	ctx, cancel := context.WithTimeout(parent, deployTimeout)
	defer cancel()
	log := m.logs.Logger(job.label)
	switch job.target.Type {
	case DeploySSH:
		if dep.Push == nil {
			return errors.New("当前环境不支持经 SSH 部署")
		}
		out, err := dep.Push(ctx, job.target.Connection, job.target.Dir, job.files, remoteCommand(job.target.Command, job.vars))
		if err != nil {
			if s := strings.TrimSpace(out); s != "" {
				return fmt.Errorf("%w：%s", err, script.Summarize([]byte(s), 500))
			}
			return err
		}
		log.Infof("证书已部署到 %s %s %s", sshTargetName(dep, job.target.Connection), job.target.Dir, script.Summarize([]byte(out), 1000))
	case DeployDocker:
		if dep.Restart == nil {
			return errors.New("当前环境不支持重启容器")
		}
		if err := dep.Restart(ctx, job.target.Containers); err != nil {
			return err
		}
		log.Infof("证书已更新，已重启容器 %s", strings.Join(job.target.Containers, ", "))
	}
	return nil
}

// finishDeploy 记录部署结果；部署期间证书或目标被修改时只更新状态，由新配置再次触发。
func (m *Module) finishDeploy(job deployJob, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	c := m.findLocked(job.certKey)
	if c == nil {
		return
	}
	ds := m.deployStateLocked(job.certKey, job.index)
	ds.Running = false
	current := job.index < len(c.Deploys) && deployHash(c, &c.Deploys[job.index]) == job.hash
	if err != nil {
		ds.LastError = err.Error()
		ds.NextRetry = time.Now().Add(deployBackoff)
		m.logs.Logger(job.label).Errorf("证书部署失败（%s）: %v", deployName(&job.target), err)
		events.Publish(events.Event{
			Type:    events.SSLDeployFailed,
			Source:  job.label,
			Level:   events.LevelError,
			Title:   "证书部署失败：" + job.label,
			Message: deployName(&job.target) + "：" + err.Error(),
		})
		return
	}
	ds.LastError = ""
	ds.NextRetry = time.Time{}
	if !current {
		return
	}
	c.Deploys[job.index].DeployedHash = job.hash
	c.Deploys[job.index].DeployedAt = time.Now()
	if err := m.saveLocked(); err != nil {
		m.logs.Logger(job.label).Errorf("保存部署记录失败: %v", err)
	}
}

// remoteCommand 在命令前以 Shell 变量赋值的方式提供证书路径。远程 sshd 通常不接受客户端传入的环境变量，
// 因此不使用 Setenv；变量值统一用单引号转义，不会被远程 Shell 解释。
func remoteCommand(command string, vars map[string]string) string {
	if strings.TrimSpace(command) == "" {
		return ""
	}
	var b strings.Builder
	for _, k := range []string{"certName", "certDir", "certFile", "keyFile"} {
		fmt.Fprintf(&b, "%s=%s; ", k, shellQuote(vars[k]))
	}
	b.WriteString(command)
	return b.String()
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// deployName 返回部署目标的简短描述，用于日志与通知。
func deployName(d *Deploy) string {
	if d.Remark != "" {
		return d.Remark
	}
	if d.Type == DeployDocker {
		return "Docker " + strings.Join(d.Containers, ", ")
	}
	return "SSH " + d.Dir
}

func sshTargetName(dep Deployers, key string) string {
	if dep.SSHTargets != nil {
		for _, t := range dep.SSHTargets() {
			if t.Key == key {
				return t.Name
			}
		}
	}
	return key
}

// deployResult 是手动部署接口返回的单个目标结果。
type deployResult struct {
	Index int    `json:"index"`
	Name  string `json:"name"`
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

// handleDeploy 立即部署证书到全部启用的目标（忽略是否已部署与失败退避），同步返回每个目标的结果。
func (m *Module) handleDeploy(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	c := m.findLocked(r.PathValue("key"))
	if c == nil {
		m.mu.Unlock()
		httpx.Error(w, http.StatusNotFound, "证书不存在")
		return
	}
	if c.CertPEM == "" || c.KeyPEM == "" {
		m.mu.Unlock()
		httpx.Error(w, http.StatusBadRequest, "证书尚未签发，暂无可部署的内容")
		return
	}
	var jobs []deployJob
	for i := range c.Deploys {
		if !c.Deploys[i].Enable {
			continue
		}
		ds := m.deployStateLocked(c.Key, i)
		if ds.Running {
			continue
		}
		ds.Running = true
		jobs = append(jobs, newDeployJobLocked(c, i))
	}
	m.mu.Unlock()
	if len(jobs) == 0 {
		httpx.Error(w, http.StatusBadRequest, "没有可执行的部署目标（未启用或正在部署）")
		return
	}
	results := make([]deployResult, 0, len(jobs))
	for _, job := range jobs {
		err := m.runDeploy(r.Context(), job)
		m.finishDeploy(job, err)
		res := deployResult{Index: job.index, Name: deployName(&job.target), OK: err == nil}
		if err != nil {
			res.Error = err.Error()
		}
		results = append(results, res)
	}
	httpx.OK(w, i18n.Localize(r, results))
}

// handleDeployTargets 返回可选的 SSH 连接与本机容器，任一来源不可用时返回空列表并附带原因。
func (m *Module) handleDeployTargets(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	dep := m.deployers
	m.mu.Unlock()
	out := map[string]any{"ssh": []SSHTarget{}, "containers": []string{}}
	if dep.SSHTargets != nil {
		out["ssh"] = dep.SSHTargets()
	}
	if dep.Containers != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		if names, err := dep.Containers(ctx); err != nil {
			out["containersError"] = err.Error()
		} else {
			out["containers"] = names
		}
	}
	httpx.OK(w, out)
}

// keepDeployRecords 编辑证书时保留部署记录：目标设置未变的部署沿用已部署的摘要，避免每次保存都重新部署。
func keepDeployRecords(old, updated *Cert) {
	for i := range updated.Deploys {
		u := &updated.Deploys[i]
		u.DeployedHash, u.DeployedAt = "", time.Time{}
		for j := range old.Deploys {
			o := &old.Deploys[j]
			if o.Type == u.Type && o.Connection == u.Connection && o.Dir == u.Dir && o.Command == u.Command &&
				strings.Join(o.Containers, "\n") == strings.Join(u.Containers, "\n") {
				u.DeployedHash, u.DeployedAt = o.DeployedHash, o.DeployedAt
				break
			}
		}
	}
}

// deployIndexLabel 用于校验错误信息中指出第几个部署目标。
func deployIndexLabel(i int) string { return strconv.Itoa(i + 1) }
