package handler

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/indu-forge/node_agent/internal/agent/network"
	pkgConfig "github.com/indu-forge/node_agent/internal/pkg/config"
)

// heartbeatController 负责运行时启停中心心跳。
type heartbeatController struct {
	mu     sync.Mutex
	cancel context.CancelFunc
}

func newHeartbeatController() *heartbeatController {
	return &heartbeatController{}
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
