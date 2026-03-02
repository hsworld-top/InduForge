package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/indu-forge/node_agent/internal/pkg/autostart"
	pkgConfig "github.com/indu-forge/node_agent/internal/pkg/config"
)

// GetBootstrapStatus 获取初始化状态。
func (h *APIHandler) GetBootstrapStatus(w http.ResponseWriter, r *http.Request) {
	state := h.bootstrapStore.Get()
	centerBound := state.CenterURL != "" && state.NodeID != ""
	response := map[string]interface{}{
		"status":        state.Status,
		"mode":          state.Mode,
		"centerUrl":     state.CenterURL,
		"nodeId":        state.NodeID,
		"nodeName":      state.NodeName,
		"centerBound":   centerBound,
		"autoStart":     state.AutoStartEnabled,
		"lastError":     state.LastError,
		"updatedAt":     state.UpdatedAt,
		"isInitialized": state.Status == BootstrapReady,
	}
	h.jsonResponse(w, http.StatusOK, response)
}

// CompleteBootstrap 完成初始化并切换为 READY。
func (h *APIHandler) CompleteBootstrap(w http.ResponseWriter, r *http.Request) {
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

	mode := req.Mode
	if mode == "" {
		mode = "local"
	}
	if mode != "online" && mode != "offline" && mode != "local" {
		h.errorResponse(w, http.StatusBadRequest, fmt.Errorf("mode 仅支持 online/offline/local"))
		return
	}

	// 完成初始化时同步落盘运行模式配置，避免重启丢失。
	configPath := pkgConfig.GetConfigPath()
	if mode == "online" {
		if req.CenterURL == "" || req.NodeID == "" || req.RegistrationToken == "" {
			h.errorResponse(w, http.StatusBadRequest, fmt.Errorf("在线模式需提供 centerUrl/nodeId/registrationToken"))
			return
		}
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
	} else {
		if req.NodeName == "" {
			h.errorResponse(w, http.StatusBadRequest, fmt.Errorf("离线模式需提供 nodeName"))
			return
		}
		if err := pkgConfig.SaveOfflineConfig(configPath, req.NodeName); err != nil {
			h.errorResponse(w, http.StatusInternalServerError, err)
			return
		}
		h.stopOnlineHeartbeat()
	}

	if err := h.bootstrapStore.Update(func(state *BootstrapState) {
		state.Mode = mode
		state.CenterURL = req.CenterURL
		state.NodeID = req.NodeID
		state.NodeName = req.NodeName
		state.Status = BootstrapReady
		state.LastError = ""
	}); err != nil {
		h.errorResponse(w, http.StatusInternalServerError, err)
		return
	}

	h.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"status":  "success",
		"message": "初始化完成",
	})
}

// SaveAutoStart 保存开机自启动设置。
func (h *APIHandler) SaveAutoStart(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Enabled bool `json:"enabled"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.errorResponse(w, http.StatusBadRequest, err)
		return
	}

	exePath, err := os.Executable()
	if err != nil {
		h.errorResponse(w, http.StatusInternalServerError, err)
		return
	}

	cfg := autostart.Config{
		ExePath:    exePath,
		Name:       "node_agent",
		Args:       []string{},
		WorkingDir: filepath.Dir(exePath),
	}

	starter := autostart.NewAutoStarter()
	if req.Enabled {
		if err := starter.Enable(cfg); err != nil {
			h.errorResponse(w, http.StatusInternalServerError, err)
			return
		}
	} else {
		if err := starter.Disable(cfg); err != nil {
			h.errorResponse(w, http.StatusInternalServerError, err)
			return
		}
	}

	if err := h.bootstrapStore.Update(func(state *BootstrapState) {
		state.AutoStartEnabled = req.Enabled
	}); err != nil {
		h.errorResponse(w, http.StatusInternalServerError, err)
		return
	}

	h.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"status":  "success",
		"message": "开机自启动配置已更新",
	})
}
