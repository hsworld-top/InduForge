package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
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
		CenterURL     string `json:"centerUrl"`
		AccessToken   string `json:"accessToken"`
		NodeName      string `json:"nodeName"`
		NodeDesc      string `json:"nodeDescription"`
		IPAddress     string `json:"ipAddress"`
		Port          int    `json:"port"`
		AgentVersion  string `json:"agentVersion"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.errorResponse(w, http.StatusBadRequest, err)
		return
	}
	if req.CenterURL == "" || req.AccessToken == "" || req.NodeName == "" {
		h.errorResponse(w, http.StatusBadRequest, fmt.Errorf("centerUrl/accessToken/nodeName 不能为空"))
		return
	}

	targetURL, err := buildCenterURL(req.CenterURL, "/api/v1/nodes/register")
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, err)
		return
	}

	payload := map[string]interface{}{
		"name":         req.NodeName,
		"description":  req.NodeDesc,
		"agentVersion": req.AgentVersion,
		"ipAddress":    req.IPAddress,
		"port":         req.Port,
	}

	body, _ := json.Marshal(payload)
	headers := map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", req.AccessToken),
	}
	h.proxyRequest(w, http.MethodPost, targetURL, body, headers)
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
