package handler

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/indu-forge/node_agent/internal/agent/network"
	pkgConfig "github.com/indu-forge/node_agent/internal/pkg/config"
	"github.com/indu-forge/node_agent/internal/pkg/types"
)

// heartbeatController 负责运行时启停中心心跳。
type heartbeatController struct {
	mu     sync.Mutex
	cancel context.CancelFunc
}

// centerCommandDeduper 中心指令去重器，避免同一条指令在连续心跳中重复执行。
type centerCommandDeduper struct {
	mu   sync.Mutex
	ttl  time.Duration
	seen map[string]time.Time
}

func newHeartbeatController() *heartbeatController {
	return &heartbeatController{}
}

func newCenterCommandDeduper(ttl time.Duration) *centerCommandDeduper {
	return &centerCommandDeduper{
		ttl:  ttl,
		seen: make(map[string]time.Time),
	}
}

func (d *centerCommandDeduper) IsDuplicate(commandKey string) bool {
	if d == nil || strings.TrimSpace(commandKey) == "" {
		return false
	}

	now := time.Now()
	d.mu.Lock()
	defer d.mu.Unlock()

	for key, at := range d.seen {
		if now.Sub(at) > d.ttl {
			delete(d.seen, key)
		}
	}

	if at, ok := d.seen[commandKey]; ok && now.Sub(at) <= d.ttl {
		return true
	}
	d.seen[commandKey] = now
	return false
}

func (d *centerCommandDeduper) Reset() {
	if d == nil {
		return
	}
	d.mu.Lock()
	d.seen = make(map[string]time.Time)
	d.mu.Unlock()
}

func (h *APIHandler) startOnlineHeartbeat(centerURL string, nodeID string, registrationToken string) {
	if h.heartbeatCtrl == nil {
		return
	}
	if strings.TrimSpace(centerURL) == "" || strings.TrimSpace(nodeID) == "" || strings.TrimSpace(registrationToken) == "" {
		h.stopOnlineHeartbeat()
		return
	}

	endpoint := fmt.Sprintf("%s/api/v1/nodes", strings.TrimRight(centerURL, "/"))
	heartbeat := network.NewHeartbeatSender(
		network.HeartbeatConfig{
			Enabled:  true,
			Interval: 10 * time.Second,
			Endpoint: endpoint,
		},
		nodeID,
		registrationToken,
	)
	heartbeat.SetCommandHandler(h.buildCenterCommandHandler(centerURL, nodeID, registrationToken))

	h.heartbeatCtrl.mu.Lock()
	if h.heartbeatCtrl.cancel != nil {
		h.heartbeatCtrl.cancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	h.heartbeatCtrl.cancel = cancel
	h.heartbeatCtrl.mu.Unlock()

	go heartbeat.Start(ctx)
	h.logger.Info("已启用运维中心心跳", "centerUrl", centerURL, "nodeId", nodeID)
}

func (h *APIHandler) stopOnlineHeartbeat() {
	if h.heartbeatCtrl == nil {
		return
	}
	h.heartbeatCtrl.mu.Lock()
	cancel := h.heartbeatCtrl.cancel
	h.heartbeatCtrl.cancel = nil
	h.heartbeatCtrl.mu.Unlock()

	if cancel != nil {
		cancel()
		h.logger.Info("已停止运维中心心跳")
	}
	if h.commandDeduper != nil {
		h.commandDeduper.Reset()
	}
}

func (h *APIHandler) syncOnlineHeartbeatFromConfig(configPath string) {
	mode, err := pkgConfig.GetMode(configPath)
	if err != nil {
		h.stopOnlineHeartbeat()
		return
	}
	if mode != pkgConfig.ModeOnline {
		h.stopOnlineHeartbeat()
		return
	}

	onlineConfig, err := pkgConfig.GetOnlineConfig(configPath)
	if err != nil || onlineConfig == nil {
		h.stopOnlineHeartbeat()
		return
	}
	h.startOnlineHeartbeat(onlineConfig.CenterURL, onlineConfig.NodeID, onlineConfig.RegistrationToken)
}

// buildCenterCommandHandler 构建中心指令处理器。
func (h *APIHandler) buildCenterCommandHandler(centerURL string, nodeID string, registrationToken string) func(context.Context, types.PendingCommand) error {
	return func(ctx context.Context, command types.PendingCommand) error {
		return h.handleCenterCommand(ctx, centerURL, nodeID, registrationToken, command)
	}
}

// handleCenterCommand 执行中心下发指令并回传执行结果。
func (h *APIHandler) handleCenterCommand(
	ctx context.Context,
	centerURL string,
	nodeID string,
	registrationToken string,
	command types.PendingCommand,
) (err error) {
	startedAt := time.Now()
	commandType := strings.ToLower(strings.TrimSpace(command.Type))
	if commandType == "" {
		return fmt.Errorf("无效指令类型")
	}

	payload := command.Payload
	commandKey := buildCenterCommandKey(commandType, payload)
	projectID := toString(payload["projectId"])
	nodeDeploymentID := toString(payload["deploymentId"])

	defer func() {
		success := err == nil
		errorMessage := ""
		if err != nil {
			errorMessage = err.Error()
		}
		h.logger.Info("中心指令执行审计", "type", commandType, "commandKey", commandKey, "nodeId", nodeID, "projectId", projectID, "deploymentId", nodeDeploymentID, "success", success, "error", errorMessage, "elapsedMs", time.Since(startedAt).Milliseconds(), "centerUrl", centerURL)
	}()

	if h.commandDeduper != nil && h.commandDeduper.IsDuplicate(commandKey) {
		h.logger.Info("忽略重复中心指令", "type", commandType, "key", commandKey)
		return nil
	}

	switch commandType {
	case "start":
		if projectID == "" {
			return fmt.Errorf("start 指令缺少 projectId")
		}
		if err := h.orchestrator.Start(ctx, projectID); err != nil {
			return h.reportDeploymentStatus(centerURL, nodeID, registrationToken, nodeDeploymentID, "error", fmt.Sprintf("启动失败: %v", err))
		}
		if nodeDeploymentID != "" {
			return h.reportDeploymentStatus(centerURL, nodeID, registrationToken, nodeDeploymentID, "running", "启动成功")
		}
		return nil
	case "stop":
		if projectID == "" {
			return fmt.Errorf("stop 指令缺少 projectId")
		}
		if err := h.orchestrator.Stop(ctx, projectID); err != nil {
			return h.reportDeploymentStatus(centerURL, nodeID, registrationToken, nodeDeploymentID, "error", fmt.Sprintf("停止失败: %v", err))
		}
		if nodeDeploymentID != "" {
			return h.reportDeploymentStatus(centerURL, nodeID, registrationToken, nodeDeploymentID, "stopped", "停止成功")
		}
		return nil
	case "restart":
		if projectID == "" {
			return fmt.Errorf("restart 指令缺少 projectId")
		}
		if err := h.orchestrator.Restart(ctx, projectID); err != nil {
			return h.reportDeploymentStatus(centerURL, nodeID, registrationToken, nodeDeploymentID, "error", fmt.Sprintf("重启失败: %v", err))
		}
		if nodeDeploymentID != "" {
			return h.reportDeploymentStatus(centerURL, nodeID, registrationToken, nodeDeploymentID, "running", "重启成功")
		}
		return nil
	case "deploy":
		if nodeDeploymentID == "" {
			return fmt.Errorf("deploy 指令缺少 deploymentId")
		}
		if projectID == "" {
			return h.reportDeploymentStatus(centerURL, nodeID, registrationToken, nodeDeploymentID, "error", "deploy 指令缺少 projectId")
		}
		artifactURL := toString(payload["artifactUrl"])
		artifactHash := strings.ToLower(strings.TrimSpace(toString(payload["artifactHash"])))
		version := toString(payload["version"])
		if version == "" {
			version = "dev"
		}
		if artifactURL == "" {
			return h.reportDeploymentStatus(centerURL, nodeID, registrationToken, nodeDeploymentID, "error", "deploy 指令缺少 artifactUrl")
		}

		if err := h.reportDeploymentStatus(centerURL, nodeID, registrationToken, nodeDeploymentID, "deploying", "开始拉取目标版本包"); err != nil {
			h.logger.Warn("回传 deploying 状态失败", "error", err)
		}

		localArtifactPath, actualHash, err := h.fetchCenterArtifact(ctx, projectID, version, artifactURL)
		if err != nil {
			return h.reportDeploymentStatus(centerURL, nodeID, registrationToken, nodeDeploymentID, "error", fmt.Sprintf("拉取目标版本包失败: %v", err))
		}
		h.logger.Info("目标版本包下载完成", "projectId", projectID, "version", version, "path", localArtifactPath, "sha256", actualHash)
		if artifactHash != "" && actualHash != artifactHash {
			return h.reportDeploymentStatus(centerURL, nodeID, registrationToken, nodeDeploymentID, "error", fmt.Sprintf("包完整性校验失败: expected=%s actual=%s", artifactHash, actualHash))
		}

		if err := h.executeCenterDeploy(ctx, projectID, version, localArtifactPath); err != nil {
			return h.reportDeploymentStatus(centerURL, nodeID, registrationToken, nodeDeploymentID, "error", fmt.Sprintf("部署后启动失败: %v", err))
		}
		return h.reportDeploymentStatus(centerURL, nodeID, registrationToken, nodeDeploymentID, "running", "部署指令执行完成")
	case "rollback":
		if nodeDeploymentID == "" {
			return fmt.Errorf("rollback 指令缺少 deploymentId")
		}
		if projectID == "" {
			return h.reportDeploymentStatus(centerURL, nodeID, registrationToken, nodeDeploymentID, "error", "rollback 指令缺少 projectId")
		}
		artifactURL := toString(payload["artifactUrl"])
		artifactHash := strings.ToLower(strings.TrimSpace(toString(payload["artifactHash"])))
		version := toString(payload["version"])
		if version == "" {
			return h.reportDeploymentStatus(centerURL, nodeID, registrationToken, nodeDeploymentID, "error", "rollback 指令缺少 version")
		}
		if artifactURL == "" {
			return h.reportDeploymentStatus(centerURL, nodeID, registrationToken, nodeDeploymentID, "error", "rollback 指令缺少 artifactUrl")
		}

		if err := h.reportDeploymentStatus(centerURL, nodeID, registrationToken, nodeDeploymentID, "rolling_back", "开始拉取回滚目标版本包"); err != nil {
			h.logger.Warn("回传 rolling_back 状态失败", "error", err)
		}

		localArtifactPath, actualHash, err := h.fetchCenterArtifact(ctx, projectID, version, artifactURL)
		if err != nil {
			return h.reportDeploymentStatus(centerURL, nodeID, registrationToken, nodeDeploymentID, "error", fmt.Sprintf("拉取回滚目标版本包失败: %v", err))
		}
		h.logger.Info("回滚目标版本包下载完成", "projectId", projectID, "version", version, "path", localArtifactPath, "sha256", actualHash)
		if artifactHash != "" && actualHash != artifactHash {
			return h.reportDeploymentStatus(centerURL, nodeID, registrationToken, nodeDeploymentID, "error", fmt.Sprintf("包完整性校验失败: expected=%s actual=%s", artifactHash, actualHash))
		}

		if err := h.executeCenterDeploy(ctx, projectID, version, localArtifactPath); err != nil {
			return h.reportDeploymentStatus(centerURL, nodeID, registrationToken, nodeDeploymentID, "error", fmt.Sprintf("回滚后启动失败: %v", err))
		}
		return h.reportDeploymentStatus(centerURL, nodeID, registrationToken, nodeDeploymentID, "running", "回滚指令执行完成")
	default:
		return fmt.Errorf("暂不支持的指令类型: %s", commandType)
	}
}

// executeCenterDeploy 以中心托管来源执行部署并自动启动。
func (h *APIHandler) executeCenterDeploy(ctx context.Context, projectID string, version string, localArtifactPath string) error {
	req := types.DeployRequest{
		ProjectID:  projectID,
		Version:    version,
		IFPPackage: localArtifactPath,
		Source:     types.ProjectSourceCenter,
		AutoStart:  true,
	}
	return h.orchestrator.Deploy(ctx, req)
}

// reportDeploymentStatus 向中心回传部署状态。
func (h *APIHandler) reportDeploymentStatus(
	centerURL string,
	nodeID string,
	registrationToken string,
	deploymentID string,
	status string,
	message string,
) error {
	if strings.TrimSpace(deploymentID) == "" {
		return nil
	}

	endpoint := fmt.Sprintf("%s/api/v1/nodes/%s/deployment-status", strings.TrimRight(centerURL, "/"), nodeID)
	body := map[string]interface{}{
		"deploymentId": deploymentID,
		"status":       status,
		"message":      message,
	}
	now := time.Now().UTC().Format(time.RFC3339)
	if status == "running" {
		body["startedAt"] = now
	}
	if status == "stopped" {
		body["stoppedAt"] = now
	}
	if status == "error" {
		body["error"] = map[string]interface{}{
			"message": message,
		}
	}
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Registration-Token", registrationToken)

	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("回传部署状态失败: status=%d", resp.StatusCode)
	}
	return nil
}

func toString(value interface{}) string {
	if value == nil {
		return ""
	}
	if text, ok := value.(string); ok {
		return strings.TrimSpace(text)
	}
	return strings.TrimSpace(fmt.Sprintf("%v", value))
}

// fetchCenterArtifact 下载中心下发的部署包并返回本地路径与 SHA256。
func (h *APIHandler) fetchCenterArtifact(ctx context.Context, projectID string, version string, artifactURL string) (string, string, error) {
	parsedURL, err := url.Parse(strings.TrimSpace(artifactURL))
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return "", "", fmt.Errorf("artifactUrl 无效")
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, parsedURL.String(), nil)
	if err != nil {
		return "", "", err
	}

	client := &http.Client{Timeout: 2 * time.Minute}
	response, err := client.Do(request)
	if err != nil {
		return "", "", err
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusBadRequest {
		return "", "", fmt.Errorf("下载失败: status=%d", response.StatusCode)
	}

	baseDir := filepath.Join("data", "projects", projectID, "artifacts", version)
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return "", "", err
	}

	ext := strings.ToLower(filepath.Ext(parsedURL.Path))
	if ext == "" {
		ext = ".ifp"
	}
	targetPath := filepath.Join(baseDir, "package"+ext)
	tempPath := targetPath + ".download"

	file, err := os.Create(tempPath)
	if err != nil {
		return "", "", err
	}

	hasher := sha256.New()
	writer := io.MultiWriter(file, hasher)
	_, copyErr := io.Copy(writer, response.Body)
	closeErr := file.Close()
	if copyErr != nil {
		_ = os.Remove(tempPath)
		return "", "", copyErr
	}
	if closeErr != nil {
		_ = os.Remove(tempPath)
		return "", "", closeErr
	}
	if err := os.Rename(tempPath, targetPath); err != nil {
		_ = os.Remove(tempPath)
		return "", "", err
	}

	return targetPath, fmt.Sprintf("%x", hasher.Sum(nil)), nil
}

func buildCenterCommandKey(commandType string, payload map[string]interface{}) string {
	commandID := toString(payload["commandId"])
	if commandID != "" {
		return fmt.Sprintf("%s#%s", commandType, commandID)
	}

	deploymentID := toString(payload["deploymentId"])
	projectID := toString(payload["projectId"])
	version := toString(payload["version"])
	if deploymentID == "" && projectID == "" {
		return ""
	}
	return fmt.Sprintf("%s|%s|%s|%s", commandType, deploymentID, projectID, version)
}
