package docker

import (
	"bufio"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/wangwanjie/gately/internal/httpx"
)

// helperImage 是卷导入导出使用的辅助容器镜像。
const helperImage = "alpine:latest"

// VolumeRef 是使用卷的容器。
type VolumeRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// VolumeView 是卷列表项。
type VolumeView struct {
	Name       string            `json:"name"`
	Driver     string            `json:"driver"`
	Mountpoint string            `json:"mountpoint"`
	Created    string            `json:"created"`
	Size       int64             `json:"size"`
	RefCount   int               `json:"refCount"`
	UsedBy     []VolumeRef       `json:"usedBy"`
	Labels     map[string]string `json:"labels"`
}

// buildVolumeViews 合并卷、用量（可能为空）与容器挂载反查。
func buildVolumeViews(vols []volumeInfo, usage map[string]volumeUsage, cts []resContainer) []VolumeView {
	usedBy := map[string][]VolumeRef{}
	for _, ct := range cts {
		seen := map[string]bool{}
		for _, mt := range ct.Mounts {
			if mt.Type == "volume" && mt.Name != "" && !seen[mt.Name] {
				seen[mt.Name] = true
				usedBy[mt.Name] = append(usedBy[mt.Name], VolumeRef{ID: ct.ID, Name: ct.name()})
			}
		}
	}
	out := make([]VolumeView, 0, len(vols))
	for _, v := range vols {
		view := VolumeView{Name: v.Name, Driver: v.Driver, Mountpoint: v.Mountpoint, Created: v.CreatedAt, Size: -1,
			UsedBy: usedBy[v.Name], Labels: v.Labels}
		if view.UsedBy == nil {
			view.UsedBy = []VolumeRef{}
		}
		if view.Labels == nil {
			view.Labels = map[string]string{}
		}
		sort.Slice(view.UsedBy, func(i, j int) bool { return view.UsedBy[i].Name < view.UsedBy[j].Name })
		view.RefCount = len(view.UsedBy)
		if u, ok := usage[v.Name]; ok {
			view.Size = u.Size
			if u.RefCount >= 0 && u.RefCount > view.RefCount {
				view.RefCount = u.RefCount
			}
		}
		out = append(out, view)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func (m *Module) handleVolumeList(w http.ResponseWriter, r *http.Request) {
	c, ok := m.withClient(w)
	if !ok {
		return
	}
	vols, err := c.resVolumes(r.Context())
	if err != nil {
		writeDockerErr(w, err)
		return
	}
	cts, err := c.resContainers(r.Context())
	if err != nil {
		writeDockerErr(w, err)
		return
	}
	// 用量信息获取失败（如 /system/df 较慢或被禁用）时，大小显示为未知
	usage, _ := c.resVolumeUsage(r.Context())
	httpx.OK(w, buildVolumeViews(vols, usage, cts))
}

func (m *Module) handleVolumeCreate(w http.ResponseWriter, r *http.Request) {
	c, ok := m.withClient(w)
	if !ok {
		return
	}
	var req struct {
		Name   string `json:"name"`
		Driver string `json:"driver"`
	}
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "%v", err)
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		httpx.Error(w, http.StatusBadRequest, "卷名称不能为空")
		return
	}
	if req.Driver == "" {
		req.Driver = "local"
	}
	if err := c.resVolumeCreate(r.Context(), req.Name, req.Driver); err != nil {
		writeDockerErr(w, err)
		return
	}
	m.logInfo("创建卷 %s", req.Name)
	httpx.OK(w, map[string]bool{"ok": true})
}

func (m *Module) handleVolumeDelete(w http.ResponseWriter, r *http.Request) {
	c, ok := m.withClient(w)
	if !ok {
		return
	}
	name := r.PathValue("name")
	if err := c.resVolumeRemove(r.Context(), name, r.URL.Query().Get("force") == "1"); err != nil {
		writeDockerErr(w, err)
		return
	}
	m.logInfo("删除卷 %s", name)
	httpx.OK(w, map[string]bool{"ok": true})
}

// ensureHelperImage 确保辅助镜像存在，缺失时先拉取。
func (m *Module) ensureHelperImage(ctx context.Context, c *Client) error {
	id, err := c.ImageID(ctx, helperImage)
	if err != nil {
		return err
	}
	if id != "" {
		return nil
	}
	if err := m.pullImageWithMirrors(ctx, c, helperImage, nil); err != nil {
		return fmt.Errorf("辅助镜像 %s 不存在且拉取失败: %w", helperImage, err)
	}
	return nil
}

// withHelper 创建挂载了卷的辅助容器并在 fn 结束后删除（使用独立 context，避免请求取消后遗留容器）。
func (m *Module) withHelper(ctx context.Context, c *Client, volume string, readOnly bool, fn func(id string) error) error {
	if err := m.ensureHelperImage(ctx, c); err != nil {
		return err
	}
	id, err := c.resCreateHelper(ctx, helperImage, volume, readOnly)
	if err != nil {
		return err
	}
	defer func() {
		cctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := c.resRemoveContainer(cctx, id); err != nil {
			m.logWarn("删除卷辅助容器 %s 失败，请手动清理: %v", shortID(id), err)
		}
	}()
	return fn(id)
}

func (m *Module) handleVolumeExport(w http.ResponseWriter, r *http.Request) {
	c, ok := m.withClient(w)
	if !ok {
		return
	}
	name := r.PathValue("name")
	// 卷必须存在，否则 Docker 会在创建辅助容器时静默新建空卷
	if _, err := c.resVolumeInspect(r.Context(), name); err != nil {
		writeDockerErr(w, err)
		return
	}
	wrote := false
	err := m.withHelper(r.Context(), c, name, true, func(id string) error {
		// 读取 /vol/. 得到卷内容本身（不带顶层目录）。归档接口对未启动的容器同样有效，无需执行命令。
		src, err := c.resArchiveGet(r.Context(), id, "/vol/.")
		if err != nil {
			return err
		}
		defer src.Close()
		w.Header().Set("Content-Type", "application/gzip")
		w.Header().Set("Content-Disposition", `attachment; filename="`+safeFilename(name)+`.tar.gz"`)
		wrote = true
		gz := gzip.NewWriter(w)
		if _, err := io.Copy(gz, src); err != nil {
			return err
		}
		return gz.Close()
	})
	if err != nil {
		if wrote {
			// 响应头已发出，只能记录日志
			m.logWarn("导出卷 %s 中断: %v", name, err)
			return
		}
		writeDockerErr(w, err)
		return
	}
	m.logInfo("导出卷 %s", name)
}

func (m *Module) handleVolumeImport(w http.ResponseWriter, r *http.Request) {
	c, ok := m.withClient(w)
	if !ok {
		return
	}
	name := r.PathValue("name")
	// 请求体可能很大，只做流式处理：按魔数判断是否 gzip，是则边读边解压
	br := bufio.NewReaderSize(r.Body, 64<<10)
	var src io.Reader = br
	if magic, err := br.Peek(2); err == nil && magic[0] == 0x1f && magic[1] == 0x8b {
		gz, err := gzip.NewReader(br)
		if err != nil {
			httpx.Error(w, http.StatusBadRequest, "压缩包格式错误: %v", err)
			return
		}
		defer gz.Close()
		src = gz
	} else if err != nil {
		httpx.Error(w, http.StatusBadRequest, "请求体为空或无法读取")
		return
	}
	// 卷不存在则创建
	if _, err := c.resVolumeInspect(r.Context(), name); err != nil {
		if !isNotFound(err) {
			writeDockerErr(w, err)
			return
		}
		if err := c.resVolumeCreate(r.Context(), name, "local"); err != nil {
			writeDockerErr(w, err)
			return
		}
	}
	err := m.withHelper(r.Context(), c, name, false, func(id string) error {
		return c.resArchivePut(r.Context(), id, "/vol", src)
	})
	if err != nil {
		writeDockerErr(w, err)
		return
	}
	m.logInfo("导入卷 %s", name)
	httpx.OK(w, map[string]bool{"ok": true})
}
