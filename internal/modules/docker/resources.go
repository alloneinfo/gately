package docker

import "net/http"

// registerResourceRoutes 注册镜像、网络、卷与 Compose 接口。
func (m *Module) registerResourceRoutes(add func(pattern string, h http.HandlerFunc)) {
	// 镜像
	add("GET /images", m.handleImageList)
	add("POST /images/pull", m.handleImagePull)
	add("POST /images/push", m.handleImagePush)
	add("POST /images/search", m.handleImageSearch)
	add("POST /images/load", m.handleImageLoad)
	add("POST /images/build", m.handleImageBuild)
	add("GET /images/{id}", m.handleImageDetail)
	add("POST /images/{id}/tag", m.handleImageTag)
	add("DELETE /images/{id}", m.handleImageDelete)
	add("GET /images/{id}/export", m.handleImageExport)

	// 网络
	add("GET /networks", m.handleNetworkList)
	add("POST /networks", m.handleNetworkCreate)
	add("DELETE /networks/{id}", m.handleNetworkDelete)
	add("POST /networks/{id}/connect", m.handleNetworkConnect)
	add("POST /networks/{id}/disconnect", m.handleNetworkDisconnect)

	// 卷
	add("GET /volumes", m.handleVolumeList)
	add("POST /volumes", m.handleVolumeCreate)
	add("DELETE /volumes/{name}", m.handleVolumeDelete)
	add("GET /volumes/{name}/export", m.handleVolumeExport)
	add("POST /volumes/{name}/import", m.handleVolumeImport)

	// Compose
	add("GET /compose/projects", m.handleComposeList)
	add("POST /compose/projects", m.handleComposeCreate)
	add("POST /compose/{project}/up", m.handleComposeUp)
	add("POST /compose/{project}/pull", m.handleComposePull)
	add("POST /compose/{project}/{action}", m.handleComposeAction)
	add("GET /compose/{project}/config", m.handleComposeConfigGet)
	add("PUT /compose/{project}/config", m.handleComposeConfigPut)
	add("GET /compose/{project}/logs", m.handleComposeLogs)
	add("DELETE /compose/{project}", m.handleComposeDelete)
}
