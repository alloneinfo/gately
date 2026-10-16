package docker

import (
	"net/http"
	"sort"
	"strings"

	"github.com/wangwanjie/gately/internal/httpx"
)

// builtinNetworks 是 Docker 内置网络，不可删除。
var builtinNetworks = map[string]bool{"bridge": true, "host": true, "none": true}

// NetworkContainer 是接入网络的容器。
type NetworkContainer struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	IP   string `json:"ip"`
}

// NetworkView 是网络列表项。
type NetworkView struct {
	ID         string             `json:"id"`
	Name       string             `json:"name"`
	Driver     string             `json:"driver"`
	Scope      string             `json:"scope"`
	Subnet     string             `json:"subnet"`
	Gateway    string             `json:"gateway"`
	Internal   bool               `json:"internal"`
	IPv6       bool               `json:"ipv6"`
	Builtin    bool               `json:"builtin"`
	Containers []NetworkContainer `json:"containers"`
}

// buildNetworkViews 合并网络与容器信息：容器所属网络取自各容器的 NetworkSettings。
func buildNetworkViews(nets []networkInfo, cts []resContainer) []NetworkView {
	byName := map[string][]NetworkContainer{}
	for _, ct := range cts {
		for netName, ep := range ct.NetworkSettings.Networks {
			byName[netName] = append(byName[netName], NetworkContainer{ID: ct.ID, Name: ct.name(), IP: ep.IPAddress})
		}
	}
	out := make([]NetworkView, 0, len(nets))
	for _, n := range nets {
		v := NetworkView{ID: n.ID, Name: n.Name, Driver: n.Driver, Scope: n.Scope, Internal: n.Internal, IPv6: n.EnableIPv6,
			Builtin: builtinNetworks[n.Name], Containers: byName[n.Name]}
		if v.Containers == nil {
			v.Containers = []NetworkContainer{}
		}
		sort.Slice(v.Containers, func(i, j int) bool { return v.Containers[i].Name < v.Containers[j].Name })
		if len(n.IPAM.Config) > 0 {
			v.Subnet, v.Gateway = n.IPAM.Config[0].Subnet, n.IPAM.Config[0].Gateway
		}
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Builtin != out[j].Builtin {
			return out[i].Builtin
		}
		return out[i].Name < out[j].Name
	})
	return out
}

func (m *Module) handleNetworkList(w http.ResponseWriter, r *http.Request) {
	c, ok := m.withClient(w)
	if !ok {
		return
	}
	nets, err := c.resNetworks(r.Context())
	if err != nil {
		writeDockerErr(w, err)
		return
	}
	cts, err := c.resContainers(r.Context())
	if err != nil {
		writeDockerErr(w, err)
		return
	}
	httpx.OK(w, buildNetworkViews(nets, cts))
}

func (m *Module) handleNetworkCreate(w http.ResponseWriter, r *http.Request) {
	c, ok := m.withClient(w)
	if !ok {
		return
	}
	var req struct {
		Name       string `json:"name"`
		Driver     string `json:"driver"`
		Subnet     string `json:"subnet"`
		Gateway    string `json:"gateway"`
		Internal   bool   `json:"internal"`
		Attachable bool   `json:"attachable"`
		IPv6       bool   `json:"ipv6"`
	}
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "%v", err)
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		httpx.Error(w, http.StatusBadRequest, "网络名称不能为空")
		return
	}
	if req.Driver == "" {
		req.Driver = "bridge"
	}
	if req.Gateway != "" && req.Subnet == "" {
		httpx.Error(w, http.StatusBadRequest, "指定网关时必须同时指定子网")
		return
	}
	body := map[string]any{"Name": req.Name, "Driver": req.Driver, "Internal": req.Internal, "Attachable": req.Attachable, "EnableIPv6": req.IPv6}
	if req.Subnet != "" {
		cfg := map[string]string{"Subnet": req.Subnet}
		if req.Gateway != "" {
			cfg["Gateway"] = req.Gateway
		}
		body["IPAM"] = map[string]any{"Driver": "default", "Config": []map[string]string{cfg}}
	}
	id, err := c.resNetworkCreate(r.Context(), body)
	if err != nil {
		writeDockerErr(w, err)
		return
	}
	m.logInfo("创建网络 %s", req.Name)
	httpx.OK(w, map[string]string{"id": id})
}

func (m *Module) handleNetworkDelete(w http.ResponseWriter, r *http.Request) {
	c, ok := m.withClient(w)
	if !ok {
		return
	}
	id := r.PathValue("id")
	n, err := c.resNetworkInspect(r.Context(), id)
	if err != nil {
		writeDockerErr(w, err)
		return
	}
	if builtinNetworks[n.Name] {
		httpx.Error(w, http.StatusBadRequest, "内置网络 %s 不能删除", n.Name)
		return
	}
	if err := c.resNetworkRemove(r.Context(), n.ID); err != nil {
		writeDockerErr(w, err)
		return
	}
	m.logInfo("删除网络 %s", n.Name)
	httpx.OK(w, map[string]bool{"ok": true})
}

func (m *Module) handleNetworkConnect(w http.ResponseWriter, r *http.Request) {
	c, ok := m.withClient(w)
	if !ok {
		return
	}
	var req struct {
		Container string   `json:"container"`
		Aliases   []string `json:"aliases"`
		IP        string   `json:"ip"`
	}
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "%v", err)
		return
	}
	if strings.TrimSpace(req.Container) == "" {
		httpx.Error(w, http.StatusBadRequest, "容器不能为空")
		return
	}
	ep := map[string]any{}
	if len(req.Aliases) > 0 {
		ep["Aliases"] = req.Aliases
	}
	if req.IP != "" {
		key := "IPv4Address"
		if strings.Contains(req.IP, ":") {
			key = "IPv6Address"
		}
		ep["IPAMConfig"] = map[string]string{key: req.IP}
	}
	if err := c.ConnectNetwork(r.Context(), r.PathValue("id"), req.Container, ep); err != nil {
		writeDockerErr(w, err)
		return
	}
	m.logInfo("容器 %s 连接网络 %s", shortID(req.Container), shortID(r.PathValue("id")))
	httpx.OK(w, map[string]bool{"ok": true})
}

func (m *Module) handleNetworkDisconnect(w http.ResponseWriter, r *http.Request) {
	c, ok := m.withClient(w)
	if !ok {
		return
	}
	var req struct {
		Container string `json:"container"`
		Force     bool   `json:"force"`
	}
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "%v", err)
		return
	}
	if strings.TrimSpace(req.Container) == "" {
		httpx.Error(w, http.StatusBadRequest, "容器不能为空")
		return
	}
	if err := c.resNetworkDisconnect(r.Context(), r.PathValue("id"), req.Container, req.Force); err != nil {
		writeDockerErr(w, err)
		return
	}
	m.logInfo("容器 %s 断开网络 %s", shortID(req.Container), shortID(r.PathValue("id")))
	httpx.OK(w, map[string]bool{"ok": true})
}
