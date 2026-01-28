package handler

import (
	"github.com/gorilla/mux"
	webstatic "github.com/indu-forge/node_agent/internal/web/static"
)

// RegisterRoutes 注册路由
func (h *APIHandler) RegisterRoutes() *mux.Router {
	router := mux.NewRouter()

	// 节点信息（放在最前，确保 API 路由优先级更高）
	router.HandleFunc("/api/v1/node/info", h.GetNodeInfo).Methods("GET")
	router.HandleFunc("/api/v1/node/status", h.StatusCheck).Methods("GET")

	// 项目管理
	projects := router.PathPrefix("/api/v1/projects").Subrouter()
	projects.HandleFunc("", h.ListProjects).Methods("GET")
	projects.HandleFunc("/{id}", h.GetProject).Methods("GET")
	projects.HandleFunc("/{id}/deploy", h.DeployProject).Methods("POST")
	projects.HandleFunc("/{id}/start", h.StartProject).Methods("POST")
	projects.HandleFunc("/{id}/stop", h.StopProject).Methods("POST")
	projects.HandleFunc("/{id}/restart", h.RestartProject).Methods("POST")
	projects.HandleFunc("/{id}/rollback", h.RollbackProject).Methods("POST")
	projects.HandleFunc("/{id}/logs", h.GetLogs).Methods("GET")

	// 连接配置
	router.HandleFunc("/api/v1/profile", h.GetProfile).Methods("GET")
	router.HandleFunc("/api/v1/profile", h.SaveProfile).Methods("POST")

	// 配置管理
	router.HandleFunc("/api/v1/config/save", h.SaveConfig).Methods("POST")
	router.HandleFunc("/api/v1/config/service", h.GetServiceConfig).Methods("GET")

	// 运维中心代理
	router.HandleFunc("/api/v1/center/login", h.CenterLogin).Methods("POST")
	router.HandleFunc("/api/v1/center/register", h.CenterRegister).Methods("POST")
	router.HandleFunc("/api/v1/center/approval-status", h.CenterApprovalStatus).Methods("GET")
	router.HandleFunc("/api/v1/center/health", h.CenterHealth).Methods("GET")

	// 健康检查
	router.HandleFunc("/health", h.HealthCheck).Methods("GET")

	// 静态文件服务 - Web 管理界面（放在最后，作为默认路由）
	router.PathPrefix("/").Handler(newSPAHandler(webstatic.DistFS(), "index.html"))

	return router
}
