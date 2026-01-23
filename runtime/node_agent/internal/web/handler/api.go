package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gorilla/mux"
	"github.com/indu-forge/node_agent/internal/agent/orchestrator"
	"github.com/indu-forge/node_agent/internal/agent/store"
	"github.com/indu-forge/node_agent/internal/pkg/types"
	"github.com/indu-forge/node_agent/internal/pkg/logger"
	pkgConfig "github.com/indu-forge/node_agent/internal/pkg/config"
)

// APIHandler API 处理器
type APIHandler struct {
	orchestrator *orchestrator.Orchestrator
	store       *store.LocalStore
	logger      *logger.SimpleLogger
}

// NewAPIHandler 创建 API 处理器
func NewAPIHandler(orch *orchestrator.Orchestrator, st *store.LocalStore) *APIHandler {
	return &APIHandler{
		orchestrator: orch,
		store:        st,
		logger:       logger.GlobalLogger,
	}
}

// GetNodeInfo 获取节点信息
func (h *APIHandler) GetNodeInfo(w http.ResponseWriter, r *http.Request) {
	// 获取当前工作目录
	workDir, _ := os.Getwd()
	
	nodeInfo := types.NodeInfo{
		ID:           "node-001",
		Name:         "Node Agent",
		Version:      "1.0.0",
		ExecutorType: "process",
		WorkDir:      filepath.ToSlash(workDir), // 使用实际工作目录，转换为正斜杠
		CreatedAt:    now(),
		UpdatedAt:    now(),
	}

	h.jsonResponse(w, http.StatusOK, nodeInfo)
}

// ListProjects 列出项目
func (h *APIHandler) ListProjects(w http.ResponseWriter, r *http.Request) {
	projects, err := h.store.ListProjects()
	if err != nil {
		h.errorResponse(w, http.StatusInternalServerError, err)
		return
	}

	h.jsonResponse(w, http.StatusOK, projects)
}

// GetProject 获取项目详情
func (h *APIHandler) GetProject(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	projectID := vars["id"]

	project, err := h.orchestrator.GetProjectStatus(r.Context(), projectID)
	if err != nil {
		h.errorResponse(w, http.StatusNotFound, err)
		return
	}

	h.jsonResponse(w, http.StatusOK, project)
}

// DeployProject 部署项目
func (h *APIHandler) DeployProject(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	projectID := vars["id"]

	var req types.DeployRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.errorResponse(w, http.StatusBadRequest, err)
		return
	}

	req.ProjectID = projectID

	if err := h.orchestrator.Deploy(r.Context(), req); err != nil {
		h.errorResponse(w, http.StatusInternalServerError, err)
		return
	}

	h.jsonResponse(w, http.StatusOK, map[string]string{
		"status":  "success",
		"message": "部署成功",
	})
}

// StartProject 启动项目
func (h *APIHandler) StartProject(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	projectID := vars["id"]

	if err := h.orchestrator.Start(r.Context(), projectID); err != nil {
		h.errorResponse(w, http.StatusInternalServerError, err)
		return
	}

	h.jsonResponse(w, http.StatusOK, map[string]string{
		"status":  "success",
		"message": "启动成功",
	})
}

// StopProject 停止项目
func (h *APIHandler) StopProject(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	projectID := vars["id"]

	if err := h.orchestrator.Stop(r.Context(), projectID); err != nil {
		h.errorResponse(w, http.StatusInternalServerError, err)
		return
	}

	h.jsonResponse(w, http.StatusOK, map[string]string{
		"status":  "success",
		"message": "停止成功",
	})
}

// RestartProject 重启项目
func (h *APIHandler) RestartProject(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	projectID := vars["id"]

	if err := h.orchestrator.Restart(r.Context(), projectID); err != nil {
		h.errorResponse(w, http.StatusInternalServerError, err)
		return
	}

	h.jsonResponse(w, http.StatusOK, map[string]string{
		"status":  "success",
		"message": "重启成功",
	})
}

// RollbackProject 回滚项目
func (h *APIHandler) RollbackProject(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	projectID := vars["id"]
	version := r.URL.Query().Get("version")

	if version == "" {
		h.errorResponse(w, http.StatusBadRequest, errors.New("version 参数不能为空"))
		return
	}

	if err := h.orchestrator.Rollback(r.Context(), projectID, version); err != nil {
		h.errorResponse(w, http.StatusInternalServerError, err)
		return
	}

	h.jsonResponse(w, http.StatusOK, map[string]string{
		"status":  "success",
		"message": "回滚成功",
	})
}

// GetProfile 获取连接配置
func (h *APIHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	projectID := r.URL.Query().Get("projectId")
	if projectID == "" {
		h.errorResponse(w, http.StatusBadRequest, errors.New("projectId 参数不能为空"))
		return
	}

	profile, err := h.store.GetConnectionProfile(projectID)
	if err != nil {
		h.errorResponse(w, http.StatusNotFound, err)
		return
	}

	h.jsonResponse(w, http.StatusOK, profile)
}

// SaveProfile 保存连接配置
func (h *APIHandler) SaveProfile(w http.ResponseWriter, r *http.Request) {
	var req types.ConnectionProfile
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.errorResponse(w, http.StatusBadRequest, err)
		return
	}

	projectID := r.URL.Query().Get("projectId")
	if projectID == "" {
		h.errorResponse(w, http.StatusBadRequest, errors.New("projectId 参数不能为空"))
		return
	}

	if err := h.store.SaveConnectionProfile(projectID, req); err != nil {
		h.errorResponse(w, http.StatusInternalServerError, err)
		return
	}

	h.jsonResponse(w, http.StatusOK, map[string]string{
		"status":  "success",
		"message": "保存成功",
	})
}

// HealthCheck 健康检查
func (h *APIHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
		"time":   time.Now().Format(time.RFC3339),
	})
}

// StatusCheck 状态检查
func (h *APIHandler) StatusCheck(w http.ResponseWriter, r *http.Request) {
	// 获取项目状态
	projects, err := h.store.ListProjects()
	if err != nil {
		h.errorResponse(w, http.StatusInternalServerError, err)
		return
	}

	status := map[string]interface{}{
		"node": map[string]interface{}{
			"id":       "node-001",
			"status":   "healthy",
			"projects": len(projects),
		},
		"projects": projects,
	}

	h.jsonResponse(w, http.StatusOK, status)
}

// GetLogs 获取日志
func (h *APIHandler) GetLogs(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	_ = vars["id"] // 获取 projectID，实际未使用

	// 简化实现，返回示例日志
	logs := []map[string]string{
		{
			"time":    time.Now().Format("2006-01-02 15:04:05"),
			"level":   "INFO",
			"message": "项目启动成功",
		},
	}

	h.jsonResponse(w, http.StatusOK, logs)
}

// jsonResponse JSON 响应
func (h *APIHandler) jsonResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

// errorResponse 错误响应
func (h *APIHandler) errorResponse(w http.ResponseWriter, statusCode int, err error) {
	h.logger.Error("API 错误", "status", statusCode, "error", err)
	h.jsonResponse(w, statusCode, map[string]string{
		"error": err.Error(),
	})
}

// SaveConfig 保存配置
func (h *APIHandler) SaveConfig(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Mode              string `json:"mode"`
		CenterURL         string `json:"centerUrl"`
		NodeID            string `json:"nodeId"`
		RegistrationToken string `json:"registrationToken"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.errorResponse(w, http.StatusBadRequest, err)
		return
	}

	// 仅在线模式需要保存配置
	if req.Mode == "online" {
		configPath := pkgConfig.GetConfigPath()
		
		onlineConfig := pkgConfig.OnlineConfig{
			CenterURL:         req.CenterURL,
			NodeID:            req.NodeID,
			RegistrationToken: req.RegistrationToken,
		}

		if err := pkgConfig.SaveOnlineConfig(configPath, onlineConfig); err != nil {
			h.errorResponse(w, http.StatusInternalServerError, err)
			return
		}

		h.logger.Info("在线模式配置已保存", "nodeId", req.NodeID)
	}

	h.jsonResponse(w, http.StatusOK, map[string]string{
		"status":  "success",
		"message": "配置保存成功",
	})
}

// now 获取当前时间
func now() time.Time {
	return time.Now()
}
