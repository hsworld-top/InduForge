package handler

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/indu-forge/node_agent/internal/agent/orchestrator"
	"github.com/indu-forge/node_agent/internal/agent/store"
	pkgConfig "github.com/indu-forge/node_agent/internal/pkg/config"
	"github.com/indu-forge/node_agent/internal/pkg/logger"
	"github.com/indu-forge/node_agent/internal/pkg/types"
	pkgUtils "github.com/indu-forge/node_agent/internal/pkg/utils"
	"gopkg.in/yaml.v3"
)

const errorCodeProjectManagedByCenter = "PROJECT_MANAGED_BY_CENTER"

// APIHandler API 处理器
type APIHandler struct {
	orchestrator   *orchestrator.Orchestrator
	store          *store.LocalStore
	logger         *logger.SimpleLogger
	bootstrapStore *BootstrapStore
	heartbeatCtrl  *heartbeatController
}

// NewAPIHandler 创建 API 处理器
func NewAPIHandler(orch *orchestrator.Orchestrator, st *store.LocalStore) *APIHandler {
	bootstrapStore, err := NewBootstrapStore("")
	if err != nil {
		logger.GlobalLogger.Warn("初始化 bootstrap 状态存储失败，使用内存默认状态", "error", err)
		bootstrapStore = &BootstrapStore{
			filePath: defaultBootstrapStatePath,
			state: BootstrapState{
				Status: BootstrapUninitialized,
			},
		}
	}

	h := &APIHandler{
		orchestrator:   orch,
		store:          st,
		logger:         logger.GlobalLogger,
		bootstrapStore: bootstrapStore,
		heartbeatCtrl:  newHeartbeatController(),
	}

	h.reconcileBootstrapFromConfig()
	return h
}

// GetNodeInfo 获取节点信息
func (h *APIHandler) GetNodeInfo(w http.ResponseWriter, r *http.Request) {
	// 获取当前工作目录
	workDir, _ := os.Getwd()
	configPath := pkgConfig.GetConfigPath()
	nodeMode, _ := pkgConfig.GetMode(configPath)
	machineID := pkgUtils.GetMachineID()

	nodeInfo := types.NodeInfo{
		ID:           machineID,
		MachineID:    machineID,
		Name:         "Node Agent",
		Version:      "1.0.0",
		Mode:         string(nodeMode),
		ExecutorType: "process",
		WorkDir:      filepath.ToSlash(workDir), // 使用实际工作目录，转换为正斜杠
		CreatedAt:    now(),
		UpdatedAt:    now(),
	}

	h.jsonResponse(w, http.StatusOK, nodeInfo)
}

// ListProjects 列出项目
func (h *APIHandler) ListProjects(w http.ResponseWriter, r *http.Request) {
	if !h.requireBootstrapReady(w) {
		return
	}

	projects, err := h.store.ListProjects()
	if err != nil {
		h.errorResponse(w, http.StatusInternalServerError, err)
		return
	}
	for _, project := range projects {
		if project.Source == "" {
			project.Source = types.ProjectSourceLocal
		}
	}

	h.jsonResponse(w, http.StatusOK, projects)
}

// GetProject 获取项目详情
func (h *APIHandler) GetProject(w http.ResponseWriter, r *http.Request) {
	if !h.requireBootstrapReady(w) {
		return
	}

	vars := mux.Vars(r)
	projectID := vars["id"]

	project, err := h.orchestrator.GetProjectStatus(r.Context(), projectID)
	if err != nil {
		h.errorResponse(w, http.StatusNotFound, err)
		return
	}
	if project.Source == "" {
		project.Source = types.ProjectSourceLocal
	}

	h.jsonResponse(w, http.StatusOK, project)
}

// DeployProject 部署项目
func (h *APIHandler) DeployProject(w http.ResponseWriter, r *http.Request) {
	if !h.requireBootstrapReady(w) {
		return
	}

	vars := mux.Vars(r)
	projectID := vars["id"]

	var req types.DeployRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.errorResponse(w, http.StatusBadRequest, err)
		return
	}

	req.ProjectID = projectID
	controlSource := h.getControlSource(r)
	if controlSource == types.ProjectSourceLocal {
		// 本地部署入口强制创建 local 工程，避免伪造中心来源。
		req.Source = types.ProjectSourceLocal
	} else {
		if req.Source == "" {
			req.Source = types.ProjectSourceCenter
		}
	}
	if err := h.ensureProjectMutable(r, projectID); err != nil {
		h.errorResponseWithCode(w, http.StatusForbidden, err, errorCodeProjectManagedByCenter)
		return
	}

	if err := h.orchestrator.Deploy(r.Context(), req); err != nil {
		h.errorResponse(w, http.StatusInternalServerError, err)
		return
	}

	h.jsonResponse(w, http.StatusOK, map[string]string{
		"status":  "success",
		"message": "部署成功",
	})
}

// DeployProjectByIFP 通过上传 IFP 包自动解析工程信息并部署。
func (h *APIHandler) DeployProjectByIFP(w http.ResponseWriter, r *http.Request) {
	if !h.requireBootstrapReady(w) {
		return
	}

	if err := r.ParseMultipartForm(256 << 20); err != nil {
		h.errorResponse(w, http.StatusBadRequest, fmt.Errorf("解析上传请求失败: %w", err))
		return
	}

	file, fileHeader, err := r.FormFile("ifpFile")
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, errors.New("请上传 ifpFile 文件"))
		return
	}
	defer file.Close()

	if !strings.HasSuffix(strings.ToLower(fileHeader.Filename), ".ifp") {
		h.errorResponse(w, http.StatusBadRequest, errors.New("仅支持 .ifp 文件"))
		return
	}

	tempPath, err := saveUploadedFile(file, fileHeader)
	if err != nil {
		h.errorResponse(w, http.StatusInternalServerError, fmt.Errorf("保存上传文件失败: %w", err))
		return
	}
	defer os.Remove(tempPath)

	projectID, version, err := parseProjectMetaFromIFP(tempPath, fileHeader.Filename)
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, err)
		return
	}

	autoStart := false
	if rawAutoStart := strings.TrimSpace(r.FormValue("autoStart")); rawAutoStart != "" {
		autoStart, _ = strconv.ParseBool(rawAutoStart)
	}

	req := types.DeployRequest{
		ProjectID:  projectID,
		Version:    version,
		IFPPackage: tempPath,
		Source:     types.ProjectSourceLocal,
		AutoStart:  autoStart,
	}

	if err := h.orchestrator.Deploy(r.Context(), req); err != nil {
		h.errorResponse(w, http.StatusInternalServerError, err)
		return
	}

	h.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"status":    "success",
		"message":   "部署成功",
		"projectId": projectID,
		"version":   version,
	})
}

// StartProject 启动项目
func (h *APIHandler) StartProject(w http.ResponseWriter, r *http.Request) {
	if !h.requireBootstrapReady(w) {
		return
	}

	vars := mux.Vars(r)
	projectID := vars["id"]
	if err := h.ensureProjectMutable(r, projectID); err != nil {
		h.errorResponseWithCode(w, http.StatusForbidden, err, errorCodeProjectManagedByCenter)
		return
	}

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
	if !h.requireBootstrapReady(w) {
		return
	}

	vars := mux.Vars(r)
	projectID := vars["id"]
	if err := h.ensureProjectMutable(r, projectID); err != nil {
		h.errorResponseWithCode(w, http.StatusForbidden, err, errorCodeProjectManagedByCenter)
		return
	}

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
	if !h.requireBootstrapReady(w) {
		return
	}

	vars := mux.Vars(r)
	projectID := vars["id"]
	if err := h.ensureProjectMutable(r, projectID); err != nil {
		h.errorResponseWithCode(w, http.StatusForbidden, err, errorCodeProjectManagedByCenter)
		return
	}

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
	if err := h.ensureProjectMutable(r, projectID); err != nil {
		h.errorResponseWithCode(w, http.StatusForbidden, err, errorCodeProjectManagedByCenter)
		return
	}
	if !h.requireBootstrapReady(w) {
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
	if !h.requireBootstrapReady(w) {
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
	if !h.requireBootstrapReady(w) {
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

	machineID := pkgUtils.GetMachineID()
	status := map[string]interface{}{
		"node": map[string]interface{}{
			"id":       machineID,
			"machineId": machineID,
			"status":   "healthy",
			"projects": len(projects),
		},
		"projects": projects,
	}
	for _, project := range projects {
		if project.Source == "" {
			project.Source = types.ProjectSourceLocal
		}
	}

	h.jsonResponse(w, http.StatusOK, status)
}

// GetLogs 获取日志
func (h *APIHandler) GetLogs(w http.ResponseWriter, r *http.Request) {
	if !h.requireBootstrapReady(w) {
		return
	}

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

// errorResponseWithCode 带业务错误码的错误响应。
func (h *APIHandler) errorResponseWithCode(w http.ResponseWriter, statusCode int, err error, code string) {
	h.logger.Error("API 错误", "status", statusCode, "error", err, "code", code)
	h.jsonResponse(w, statusCode, map[string]string{
		"error": err.Error(),
		"code":  code,
	})
}

// SaveConfig 保存配置
func (h *APIHandler) SaveConfig(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Mode              string `json:"mode"`
		CenterURL         string `json:"centerUrl"`
		NodeID            string `json:"nodeId"`
		RegistrationToken string `json:"registrationToken"`
		NodeName          string `json:"nodeName"`
		NodeDescription   string `json:"nodeDescription"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.errorResponse(w, http.StatusBadRequest, err)
		return
	}

	configPath := pkgConfig.GetConfigPath()

	if req.Mode == "online" {
		onlineConfig := pkgConfig.OnlineConfig{
			CenterURL:         req.CenterURL,
			NodeID:            req.NodeID,
			RegistrationToken: req.RegistrationToken,
		}

		if err := pkgConfig.SaveOnlineConfig(configPath, onlineConfig); err != nil {
			h.errorResponse(w, http.StatusInternalServerError, err)
			return
		}
		h.startOnlineHeartbeat(req.CenterURL, req.NodeID, req.RegistrationToken)

		h.logger.Info("在线模式配置已保存", "nodeId", req.NodeID)
		if h.bootstrapStore != nil {
			_ = h.bootstrapStore.Update(func(state *BootstrapState) {
				state.Mode = "online"
				state.CenterURL = req.CenterURL
				state.NodeID = req.NodeID
				if req.CenterURL != "" && req.NodeID != "" && req.RegistrationToken != "" {
					state.Status = BootstrapRegisteredPending
				} else {
					state.Status = BootstrapCenterConfigured
				}
				state.LastError = ""
			})
		}
	} else if req.Mode == "offline" {
		if err := pkgConfig.SaveOfflineConfig(configPath, req.NodeName); err != nil {
			h.errorResponse(w, http.StatusInternalServerError, err)
			return
		}
		h.stopOnlineHeartbeat()
		h.logger.Info("离线模式配置已保存", "nodeName", req.NodeName)
		if h.bootstrapStore != nil {
			_ = h.bootstrapStore.Update(func(state *BootstrapState) {
				state.Mode = "offline"
				state.NodeName = req.NodeName
				if req.NodeName != "" {
					state.Status = BootstrapReady
				} else {
					state.Status = BootstrapUninitialized
				}
				state.LastError = ""
			})
		}
	}

	h.jsonResponse(w, http.StatusOK, map[string]string{
		"status":  "success",
		"message": "配置保存成功",
	})
}

// GetServiceConfig 获取服务配置
func (h *APIHandler) GetServiceConfig(w http.ResponseWriter, r *http.Request) {
	configPath := pkgConfig.GetConfigPath()

	listenConfig, err := pkgConfig.GetListenConfig(configPath)
	if err != nil {
		h.errorResponse(w, http.StatusInternalServerError, err)
		return
	}

	// 读取完整配置以返回在线/离线信息
	var config struct {
		Agent struct {
			ID     string                 `yaml:"id"`
			Mode   pkgConfig.NodeMode     `yaml:"mode"`
			Online pkgConfig.OnlineConfig `yaml:"online"`
		} `yaml:"agent"`
	}

	configData, err := os.ReadFile(configPath)
	if err != nil {
		h.errorResponse(w, http.StatusInternalServerError, err)
		return
	}
	if err := yaml.Unmarshal(configData, &config); err != nil {
		h.errorResponse(w, http.StatusInternalServerError, err)
		return
	}

	response := map[string]interface{}{
		"listen": map[string]interface{}{
			"host": listenConfig.Host,
			"port": listenConfig.Port,
		},
		"mode":     string(config.Agent.Mode),
		"nodeName": config.Agent.ID,
		"online": map[string]interface{}{
			"centerUrl":         config.Agent.Online.CenterURL,
			"nodeId":            config.Agent.Online.NodeID,
			"registrationToken": config.Agent.Online.RegistrationToken,
		},
	}
	if h.bootstrapStore != nil {
		bootstrapState := h.bootstrapStore.Get()
		response["bootstrap"] = map[string]interface{}{
			"status":        bootstrapState.Status,
			"isInitialized": bootstrapState.Status == BootstrapReady,
		}
	}

	h.jsonResponse(w, http.StatusOK, response)
}

// requireBootstrapReady 校验初始化是否完成。
func (h *APIHandler) requireBootstrapReady(w http.ResponseWriter) bool {
	if h.bootstrapStore == nil {
		return true
	}
	if h.bootstrapStore.IsReady() {
		return true
	}

	h.jsonResponse(w, http.StatusLocked, map[string]interface{}{
		"error": "节点尚未完成初始化，请先完成初始化向导",
		"code":  "NODE_NOT_INITIALIZED",
	})
	return false
}

// getControlSource 获取控制来源。
func (h *APIHandler) getControlSource(r *http.Request) string {
	source := strings.TrimSpace(strings.ToLower(r.Header.Get("X-Control-Source")))
	if source == "" {
		source = strings.TrimSpace(strings.ToLower(r.URL.Query().Get("controlSource")))
	}
	if source == types.ProjectSourceCenter {
		return types.ProjectSourceCenter
	}
	return types.ProjectSourceLocal
}

// ensureProjectMutable 校验项目是否允许本地变更。
func (h *APIHandler) ensureProjectMutable(r *http.Request, projectID string) error {
	project, err := h.store.GetProject(projectID)
	if err != nil {
		// 项目不存在时由后续流程处理。
		return nil
	}
	if project.Source == "" {
		project.Source = types.ProjectSourceLocal
	}
	if project.Source == types.ProjectSourceCenter && h.getControlSource(r) != types.ProjectSourceCenter {
		return fmt.Errorf("该工程由运维中心托管，请在运维中心执行状态变更")
	}
	return nil
}

// reconcileBootstrapFromConfig 从现有配置文件恢复 bootstrap 状态。
func (h *APIHandler) reconcileBootstrapFromConfig() {
	if h.bootstrapStore == nil {
		return
	}

	configPath := pkgConfig.GetConfigPath()
	mode, err := pkgConfig.GetMode(configPath)
	if err != nil {
		return
	}

	if mode == pkgConfig.ModeOnline {
		onlineConfig, err := pkgConfig.GetOnlineConfig(configPath)
		if err != nil {
			h.stopOnlineHeartbeat()
			return
		}
		if onlineConfig.CenterURL != "" && onlineConfig.NodeID != "" && onlineConfig.RegistrationToken != "" {
			h.startOnlineHeartbeat(onlineConfig.CenterURL, onlineConfig.NodeID, onlineConfig.RegistrationToken)
			_ = h.bootstrapStore.Update(func(state *BootstrapState) {
				if state.Status == BootstrapReady {
					return
				}
				state.Status = BootstrapReady
				state.Mode = "online"
				state.CenterURL = onlineConfig.CenterURL
				state.NodeID = onlineConfig.NodeID
				state.LastError = ""
			})
		} else {
			h.stopOnlineHeartbeat()
		}
		return
	}

	if mode == pkgConfig.ModeOffline {
		h.stopOnlineHeartbeat()
		nodeName := ""
		var config struct {
			Agent struct {
				ID string `yaml:"id"`
			} `yaml:"agent"`
		}
		if content, err := os.ReadFile(configPath); err == nil {
			_ = yaml.Unmarshal(content, &config)
			nodeName = config.Agent.ID
		}

		_ = h.bootstrapStore.Update(func(state *BootstrapState) {
			if state.Status == BootstrapReady || nodeName == "" {
				return
			}
			state.Status = BootstrapReady
			state.Mode = "offline"
			state.NodeName = nodeName
			state.LastError = ""
		})
	}
}

// now 获取当前时间
func now() time.Time {
	return time.Now()
}

func saveUploadedFile(file multipart.File, fileHeader *multipart.FileHeader) (string, error) {
	baseName := strings.TrimSuffix(filepath.Base(fileHeader.Filename), filepath.Ext(fileHeader.Filename))
	if baseName == "" {
		baseName = "package"
	}
	tempFile, err := os.CreateTemp("", fmt.Sprintf("node-agent-%s-*.ifp", baseName))
	if err != nil {
		return "", err
	}
	defer tempFile.Close()

	if _, err := io.Copy(tempFile, file); err != nil {
		return "", err
	}
	return tempFile.Name(), nil
}

func parseProjectMetaFromIFP(ifpPath string, filename string) (string, string, error) {
	reader, err := zip.OpenReader(ifpPath)
	if err != nil {
		return "", "", errors.New("IFP 文件格式无效，无法读取压缩包")
	}
	defer reader.Close()

	candidateFiles := []string{
		"manifest.json",
		"metadata.json",
		"project.json",
		"ifp.json",
		"meta/manifest.json",
		"meta/metadata.json",
		"META-INF/manifest.json",
	}

	projectID := ""
	version := ""
	for _, name := range candidateFiles {
		meta, ok := readJSONFromZip(reader.File, name)
		if !ok {
			continue
		}
		if projectID == "" {
			projectID = firstString(meta,
				"projectId", "project_id", "projectID", "id", "name",
			)
			if projectID == "" {
				projectID = nestedFirstString(meta, "project", "projectId", "id", "name")
			}
		}
		if version == "" {
			version = firstString(meta,
				"version", "projectVersion", "appVersion",
			)
			if version == "" {
				version = nestedFirstString(meta, "project", "version")
			}
		}
		if projectID != "" && version != "" {
			break
		}
	}

	if projectID == "" || version == "" {
		inferredProjectID, inferredVersion := inferMetaFromFilename(filename)
		if projectID == "" {
			projectID = inferredProjectID
		}
		if version == "" {
			version = inferredVersion
		}
	}

	projectID = strings.TrimSpace(projectID)
	version = strings.TrimSpace(version)
	if projectID == "" || version == "" {
		return "", "", errors.New("无法从 IFP 包中解析 projectId/version，请检查包内元数据")
	}
	return projectID, version, nil
}

func readJSONFromZip(files []*zip.File, targetName string) (map[string]interface{}, bool) {
	normalizedTarget := strings.ToLower(strings.TrimSpace(targetName))
	for _, file := range files {
		if strings.ToLower(strings.TrimSpace(file.Name)) != normalizedTarget {
			continue
		}
		fileReader, err := file.Open()
		if err != nil {
			return nil, false
		}
		defer fileReader.Close()

		var meta map[string]interface{}
		if err := json.NewDecoder(fileReader).Decode(&meta); err != nil {
			return nil, false
		}
		return meta, true
	}
	return nil, false
}

func firstString(data map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if raw, ok := data[key]; ok {
			if value, ok := raw.(string); ok {
				value = strings.TrimSpace(value)
				if value != "" {
					return value
				}
			}
		}
	}
	return ""
}

func nestedFirstString(data map[string]interface{}, parentKey string, keys ...string) string {
	rawParent, ok := data[parentKey]
	if !ok {
		return ""
	}
	parentMap, ok := rawParent.(map[string]interface{})
	if !ok {
		return ""
	}
	return firstString(parentMap, keys...)
}

func inferMetaFromFilename(filename string) (string, string) {
	name := strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))
	name = strings.TrimSpace(name)
	if name == "" {
		return "", ""
	}
	parts := strings.Split(name, "-")
	if len(parts) >= 2 {
		projectID := strings.Join(parts[:len(parts)-1], "-")
		version := parts[len(parts)-1]
		return strings.TrimSpace(projectID), strings.TrimSpace(version)
	}
	return name, ""
}
