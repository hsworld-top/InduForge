package hostd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const TimeSyncSchemaVersion = "induforge.time-sync-plan.v1"

type TimeSyncPlan struct {
	SchemaVersion  string   `json:"schemaVersion"`
	Generation     int64    `json:"generation"`
	NodeID         string   `json:"nodeId"`
	Role           string   `json:"role"`
	CenterIP       string   `json:"centerIp"`
	AllowedClients []string `json:"allowedClients,omitempty"`
}

type TimeSyncState struct {
	SchemaVersion string  `json:"schemaVersion"`
	Generation    int64   `json:"generation"`
	NodeID        string  `json:"nodeId"`
	Role          string  `json:"role"`
	Source        string  `json:"source"`
	Status        string  `json:"status"`
	OffsetMillis  float64 `json:"offsetMillis"`
	Message       string  `json:"message,omitempty"`
	CheckedAt     string  `json:"checkedAt"`
}

func (plan TimeSyncPlan) Validate() error {
	if plan.SchemaVersion != TimeSyncSchemaVersion || plan.Generation < 1 || !uuidPattern.MatchString(strings.ToLower(plan.NodeID)) {
		return errors.New("时间同步计划元数据无效")
	}
	center := net.ParseIP(strings.TrimSpace(plan.CenterIP))
	if center == nil || center.IsUnspecified() || center.IsLoopback() || center.IsMulticast() {
		return errors.New("中心时间源地址无效")
	}
	if plan.Role != "center" && plan.Role != "client" {
		return errors.New("时间同步节点角色无效")
	}
	seen := map[string]struct{}{}
	for _, value := range plan.AllowedClients {
		ip := net.ParseIP(strings.TrimSpace(value))
		if ip == nil || ip.IsUnspecified() || ip.IsLoopback() || ip.IsMulticast() {
			return errors.New("允许同步时间的节点地址无效")
		}
		if _, exists := seen[ip.String()]; exists {
			return errors.New("允许同步时间的节点地址重复")
		}
		seen[ip.String()] = struct{}{}
	}
	if plan.Role == "client" && len(plan.AllowedClients) != 0 {
		return errors.New("工作节点不能提供时间服务")
	}
	return nil
}

func (m *Manager) ApplyTimeSync(ctx context.Context, plan TimeSyncPlan) (TimeSyncState, error) {
	if err := plan.Validate(); err != nil {
		return TimeSyncState{}, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, err := os.Stat(m.cfg.Chronyc); err != nil {
		state := failedTimeSyncState(plan, "节点安装包未正确安装 Chrony")
		_ = m.saveTimeSyncPlan(plan)
		return state, fmt.Errorf("Chrony 客户端不可用: %w", err)
	}
	config := plan.RenderChronyConfig()
	existing, _ := os.ReadFile(m.cfg.ChronyConfig)
	configChanged := string(existing) != config
	if configChanged {
		if err := ensureRealDirectory(filepath.Dir(m.cfg.ChronyConfig), 0755); err != nil {
			return TimeSyncState{}, err
		}
		if err := writeAtomic(m.cfg.ChronyConfig, []byte(config), 0644); err != nil {
			return TimeSyncState{}, fmt.Errorf("写入 Chrony 固定配置失败: %w", err)
		}
	}
	dropIn := plan.RenderChronyServiceDropIn()
	existingDropIn, _ := os.ReadFile(m.cfg.ChronyDropIn)
	if string(existingDropIn) != dropIn {
		if err := ensureRealDirectory(filepath.Dir(m.cfg.ChronyDropIn), 0755); err != nil {
			return TimeSyncState{}, err
		}
		if err := writeAtomic(m.cfg.ChronyDropIn, []byte(dropIn), 0644); err != nil {
			return TimeSyncState{}, fmt.Errorf("写入 Chrony 服务约束失败: %w", err)
		}
		configChanged = true
		if output, err := m.cfg.Runner.Run(ctx, m.cfg.Systemctl, "daemon-reload"); err != nil {
			return failedTimeSyncState(plan, sanitizedCommandMessage(output, err)), commandError("重新加载 Chrony 服务约束", output, err)
		}
	}
	activeOutput, activeErr := m.cfg.Runner.Run(ctx, m.cfg.Systemctl, "is-active", "chrony.service")
	if activeErr != nil || strings.TrimSpace(string(activeOutput)) != "active" {
		_, _ = m.cfg.Runner.Run(ctx, m.cfg.Systemctl, "disable", "--now", "systemd-timesyncd.service")
		if output, err := m.cfg.Runner.Run(ctx, m.cfg.Systemctl, "enable", "--now", "chrony.service"); err != nil {
			return failedTimeSyncState(plan, sanitizedCommandMessage(output, err)), commandError("启动节点时间同步", output, err)
		}
		configChanged = false
	}
	if configChanged {
		if output, err := m.cfg.Runner.Run(ctx, m.cfg.Systemctl, "restart", "chrony.service"); err != nil {
			return failedTimeSyncState(plan, sanitizedCommandMessage(output, err)), commandError("重新加载节点时间同步", output, err)
		}
	}
	if err := m.saveTimeSyncPlan(plan); err != nil {
		return TimeSyncState{}, err
	}
	state := m.observeTimeSync(ctx, plan)
	if plan.Role == "client" && state.OffsetMillis > 30000 {
		// 大幅偏差若只依赖 slew 可能持续数分钟，期间告警、采集和事件时间线均不可信。
		// 只允许工作节点立即跳变；中心节点由 -x 保证永远不被平台修改。
		if output, err := m.cfg.Runner.Run(ctx, m.cfg.Chronyc, "makestep"); err != nil {
			state.Status = "failed"
			state.Message = "工作节点大幅时间偏差自动修复失败: " + sanitizedCommandMessage(output, err)
			return state, nil
		}
		state = m.observeTimeSync(ctx, plan)
		if state.Status == "synchronized" {
			state.Message = "检测到超过 30 秒的偏差，已立即与中心时间对齐"
		}
	}
	return state, nil
}

func (m *Manager) TimeSyncStatus(ctx context.Context) (TimeSyncState, error) {
	plan, exists, err := m.loadTimeSyncPlan()
	if err != nil {
		return TimeSyncState{}, err
	}
	if !exists {
		return TimeSyncState{SchemaVersion: "induforge.time-sync-state.v1", Status: "unconfigured", CheckedAt: time.Now().UTC().Format(time.RFC3339)}, nil
	}
	return m.observeTimeSync(ctx, plan), nil
}

func (plan TimeSyncPlan) RenderChronyConfig() string {
	lines := []string{
		"# Managed by InduForge. User maintains the center system clock.",
		"driftfile /var/lib/chrony/chrony.drift",
		"logdir /var/log/chrony",
	}
	if plan.Role == "center" {
		lines = append(lines, "local stratum 10", "bindaddress "+plan.CenterIP)
		for _, client := range plan.AllowedClients {
			lines = append(lines, "allow "+client)
		}
	} else {
		// 工作节点在任何一次重新获得中心样本后都允许修正超过 1 秒的偏差。
		// 这覆盖离线重连和用户调整中心时间；中心进程自身由 -x 禁止调钟。
		// 限制采样周期，避免虚拟机来宾时钟异常漂移时 Chrony 将轮询放大到
		// 数百秒，导致节点事件时间再次偏离中心。物理机也沿用同一稳定上限。
		lines = append(lines, "server "+plan.CenterIP+" iburst prefer minpoll 2 maxpoll 4", "makestep 1.0 -1", "rtcsync")
	}
	return strings.Join(lines, "\n") + "\n"
}

// 中心节点使用 -x 明确禁止 Chrony 调整系统时钟；平台只发布中心现有时间。
func (plan TimeSyncPlan) RenderChronyServiceDropIn() string {
	options := "-F 1"
	if plan.Role == "center" {
		options += " -x"
	}
	return "[Service]\nExecStart=\nExecStart=!/usr/lib/systemd/scripts/chronyd-starter.sh " + options + "\n"
}

func (m *Manager) observeTimeSync(ctx context.Context, plan TimeSyncPlan) TimeSyncState {
	state := TimeSyncState{SchemaVersion: "induforge.time-sync-state.v1", Generation: plan.Generation, NodeID: plan.NodeID, Role: plan.Role, Source: plan.CenterIP, Status: "failed", CheckedAt: time.Now().UTC().Format(time.RFC3339)}
	if plan.Role == "center" {
		state.Source = "center-system-clock"
		if output, err := m.cfg.Runner.Run(ctx, m.cfg.Systemctl, "is-active", "chrony.service"); err == nil && strings.TrimSpace(string(output)) == "active" {
			state.Status, state.Message = "synchronized", ""
		} else {
			state.Message = "中心时间服务未运行"
		}
		return state
	}
	output, err := m.cfg.Runner.Run(ctx, m.cfg.Chronyc, "-n", "tracking")
	if err != nil {
		state.Message = sanitizedCommandMessage(output, err)
		return state
	}
	offsetSeconds, lastOffsetSeconds, leapNormal, systemOffsetFound := 0.0, 0.0, false, false
	for _, line := range strings.Split(string(output), "\n") {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key, value := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
		if key == "System time" {
			fields := strings.Fields(value)
			if len(fields) > 0 {
				if parsed, parseErr := strconv.ParseFloat(fields[0], 64); parseErr == nil {
					offsetSeconds, systemOffsetFound = parsed, true
				}
			}
		}
		if key == "Last offset" {
			fields := strings.Fields(value)
			if len(fields) > 0 {
				lastOffsetSeconds, _ = strconv.ParseFloat(fields[0], 64)
			}
		}
		if key == "Leap status" && strings.EqualFold(value, "Normal") {
			leapNormal = true
		}
	}
	if !systemOffsetFound {
		offsetSeconds = lastOffsetSeconds
	}
	state.OffsetMillis = math.Abs(offsetSeconds * 1000)
	if !leapNormal {
		state.Message = "尚未与中心时间源完成同步"
	} else if state.OffsetMillis > 30000 {
		state.Message = "与中心时间偏差超过 30 秒"
	} else if state.OffsetMillis > 5000 {
		state.Status, state.Message = "adjusting", "与中心时间偏差超过 5 秒，正在平滑校准"
	} else {
		state.Status = "synchronized"
	}
	return state
}

func failedTimeSyncState(plan TimeSyncPlan, message string) TimeSyncState {
	return TimeSyncState{SchemaVersion: "induforge.time-sync-state.v1", Generation: plan.Generation, NodeID: plan.NodeID, Role: plan.Role, Source: plan.CenterIP, Status: "failed", Message: message, CheckedAt: time.Now().UTC().Format(time.RFC3339)}
}

func (m *Manager) timeSyncPlanPath() string {
	return filepath.Join(m.cfg.StateDir, "time-sync-plan.json")
}

func (m *Manager) saveTimeSyncPlan(plan TimeSyncPlan) error {
	raw, err := json.Marshal(plan)
	if err != nil {
		return err
	}
	return writeAtomic(m.timeSyncPlanPath(), raw, 0600)
}

func (m *Manager) loadTimeSyncPlan() (TimeSyncPlan, bool, error) {
	raw, err := os.ReadFile(m.timeSyncPlanPath())
	if os.IsNotExist(err) {
		return TimeSyncPlan{}, false, nil
	}
	if err != nil {
		return TimeSyncPlan{}, false, err
	}
	var plan TimeSyncPlan
	if err := json.Unmarshal(raw, &plan); err != nil {
		return TimeSyncPlan{}, false, err
	}
	return plan, true, nil
}
