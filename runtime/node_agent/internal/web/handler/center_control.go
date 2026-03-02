package handler

import "net/http"

// withCenterControlSource 在请求中注入中心控制来源标记。
func (h *APIHandler) withCenterControlSource(r *http.Request) *http.Request {
	cloned := r.Clone(r.Context())
	cloned.Header.Set("X-Control-Source", "center")
	return cloned
}

// CenterDeployProject 中心来源部署项目入口。
func (h *APIHandler) CenterDeployProject(w http.ResponseWriter, r *http.Request) {
	h.DeployProject(w, h.withCenterControlSource(r))
}

// CenterStartProject 中心来源启动项目入口。
func (h *APIHandler) CenterStartProject(w http.ResponseWriter, r *http.Request) {
	h.StartProject(w, h.withCenterControlSource(r))
}

// CenterStopProject 中心来源停止项目入口。
func (h *APIHandler) CenterStopProject(w http.ResponseWriter, r *http.Request) {
	h.StopProject(w, h.withCenterControlSource(r))
}

// CenterRestartProject 中心来源重启项目入口。
func (h *APIHandler) CenterRestartProject(w http.ResponseWriter, r *http.Request) {
	h.RestartProject(w, h.withCenterControlSource(r))
}

// CenterRollbackProject 中心来源回滚项目入口。
func (h *APIHandler) CenterRollbackProject(w http.ResponseWriter, r *http.Request) {
	h.RollbackProject(w, h.withCenterControlSource(r))
}
