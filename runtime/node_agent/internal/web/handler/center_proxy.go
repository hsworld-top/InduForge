package handler

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	pkgUtils "github.com/indu-forge/node_agent/internal/pkg/utils"
)

// CenterLogin 代理运维中心登录请求
func (h *APIHandler) CenterLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CenterURL  string `json:"centerUrl"`
		Username   string `json:"username"`
		Password   string `json:"password"`
		TenantCode string `json:"tenantCode"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.errorResponse(w, http.StatusBadRequest, err)
		return
	}
	if req.CenterURL == "" || req.Username == "" || req.Password == "" {
		h.errorResponse(w, http.StatusBadRequest, fmt.Errorf("centerUrl/username/password 不能为空"))
		return
	}

	targetURL, err := buildCenterURL(req.CenterURL, "/api/v1/auth/login")
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, err)
		return
	}

	payload := map[string]string{
		"username": req.Username,
		"password": req.Password,
	}
	if req.TenantCode != "" {
		payload["tenantCode"] = req.TenantCode
	}

	body, _ := json.Marshal(payload)
	h.proxyRequest(w, http.MethodPost, targetURL, body, nil)
}

// CenterRegister 代理运维中心节点注册请求
func (h *APIHandler) CenterRegister(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CenterURL    string `json:"centerUrl"`
		Username     string `json:"username"`
		Password     string `json:"password"`
		NodeName     string `json:"nodeName"`
		NodeDesc     string `json:"nodeDescription"`
		IPAddress    string `json:"ipAddress"`
		Port         int    `json:"port"`
		AgentVersion string `json:"agentVersion"`
		TenantCode   string `json:"tenantCode"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.errorResponse(w, http.StatusBadRequest, err)
		return
	}
	if req.CenterURL == "" || req.Username == "" || req.Password == "" || req.NodeName == "" {
		h.errorResponse(w, http.StatusBadRequest, fmt.Errorf("centerUrl/username/password/nodeName 不能为空"))
		return
	}

	targetURL, err := buildCenterURL(req.CenterURL, "/api/v1/node-register/register-with-auth")
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, err)
		return
	}

	payload := map[string]interface{}{
		"username":        req.Username,
		"password":        req.Password,
		"tenantCode":      req.TenantCode,
		"nodeId":          buildCenterNodeID(pkgUtils.GetMachineID()),
		"nodeName":        req.NodeName,
		"nodeDescription": req.NodeDesc,
		"agentVersion":    req.AgentVersion,
		"ipAddress":       req.IPAddress,
		"port":            req.Port,
		"mode":            "online",
	}

	body, _ := json.Marshal(payload)
	h.proxyRequest(w, http.MethodPost, targetURL, body, nil)
}

// CenterApprovalStatus 代理运维中心审批状态查询
func (h *APIHandler) CenterApprovalStatus(w http.ResponseWriter, r *http.Request) {
	centerURL := r.URL.Query().Get("centerUrl")
	nodeId := r.URL.Query().Get("nodeId")
	if centerURL == "" || nodeId == "" {
		h.errorResponse(w, http.StatusBadRequest, fmt.Errorf("centerUrl/nodeId 不能为空"))
		return
	}

	targetURL, err := buildCenterURL(centerURL, "/api/v1/node-register/"+nodeId+"/approval-status")
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, err)
		return
	}

	h.proxyRequest(w, http.MethodGet, targetURL, nil, nil)
}

// CenterHealth 代理运维中心健康检查
func (h *APIHandler) CenterHealth(w http.ResponseWriter, r *http.Request) {
	centerURL := r.URL.Query().Get("centerUrl")
	if centerURL == "" {
		h.errorResponse(w, http.StatusBadRequest, fmt.Errorf("centerUrl 不能为空"))
		return
	}

	targetURL, err := buildCenterURL(centerURL, "/health")
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, err)
		return
	}

	h.proxyRequest(w, http.MethodGet, targetURL, nil, nil)
}

// proxyRequest 转发请求并回写响应
func (h *APIHandler) proxyRequest(w http.ResponseWriter, method, targetURL string, body []byte, headers map[string]string) {
	var bodyReader io.Reader
	if len(body) > 0 {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequest(method, targetURL, bodyReader)
	if err != nil {
		h.errorResponse(w, http.StatusInternalServerError, err)
		return
	}
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		h.errorResponse(w, http.StatusBadGateway, err)
		return
	}
	defer resp.Body.Close()

	contentType := resp.Header.Get("Content-Type")
	if contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}

	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

// buildCenterURL 拼接运维中心 URL
func buildCenterURL(baseURL, path string) (string, error) {
	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("centerUrl 无效")
	}

	ref, err := url.Parse(path)
	if err != nil {
		return "", err
	}

	return parsed.ResolveReference(ref).String(), nil
}

// buildCenterNodeID 规范化用于运维中心注册的 nodeId（最长 36 位）。
func buildCenterNodeID(machineID string) string {
	id := strings.TrimSpace(machineID)
	if id == "" {
		return ""
	}
	id = strings.TrimPrefix(id, "node-")
	if len(id) <= 36 {
		return id
	}

	// 若机器码超长，回退为基于机器码哈希的稳定 UUID，确保同一机器恒定。
	sum := sha256.Sum256([]byte(id))
	hexText := hex.EncodeToString(sum[:16])
	return fmt.Sprintf("%s-%s-%s-%s-%s", hexText[0:8], hexText[8:12], hexText[12:16], hexText[16:20], hexText[20:32])
}
