package ops

import (
	"fmt"
	"reflect"
	"time"
)

// ActivateWorkload 接收节点本地产生的启动配置，不能直接接收中心提供的命令或路径。
// 配置和进程按 serviceId 隔离；候选版本验证通过后才停旧进程，失败则恢复旧配置。
func (s *Supervisor) ActivateWorkload(id string, group ServiceGroup, generation int64, services []ServiceConfig, restart bool) (ProcessStatus, error) {
	s.opMu.Lock()
	defer s.opMu.Unlock()
	if err := validateWorkload(id, group); err != nil {
		return ProcessStatus{}, err
	}
	if len(services) == 0 {
		return ProcessStatus{}, fmt.Errorf("部署启动配置为空")
	}
	candidates := make([]ServiceConfig, len(services))
	seen := map[string]bool{}
	for i, service := range services {
		if service.Group != group || seen[service.Component] {
			return ProcessStatus{}, fmt.Errorf("部署服务类型或组件重复")
		}
		seen[service.Component] = true
		installed := false
		for _, template := range s.installed[group] {
			if template.Component == service.Component {
				installed = true
			}
		}
		if !installed {
			return ProcessStatus{}, fmt.Errorf("节点未安装组件 %s", service.Component)
		}
		if service.Current == "" {
			service.Current = "current"
		}
		if service.HealthTimeout <= 0 {
			service.HealthTimeout = 10 * time.Second
		}
		if service.DrainTimeout <= 0 {
			service.DrainTimeout = 20 * time.Second
		}
		if err := validateServiceConfig(service); err != nil {
			return ProcessStatus{}, err
		}
		if _, _, err := resolveReleaseService(service); err != nil {
			return ProcessStatus{}, err
		}
		service.Arguments = append([]string(nil), service.Arguments...)
		if service.Environment != nil {
			values := make(map[string]string, len(service.Environment))
			for key, value := range service.Environment {
				values[key] = value
			}
			service.Environment = values
		}
		candidates[i] = service
	}
	s.mu.Lock()
	previous := s.workloads[id]
	oldStatus := s.statusLocked(id)
	if len(previous) == 0 && oldStatus.State == "running" {
		previous = s.services[group]
	}
	if generation < oldStatus.Generation {
		s.mu.Unlock()
		return oldStatus, nil
	}
	same := reflect.DeepEqual(previous, candidates)
	if same && oldStatus.State == "running" && oldStatus.Generation == generation {
		s.mu.Unlock()
		return oldStatus, nil
	}
	s.mu.Unlock()
	if !same || restart {
		if _, err := s.stop(id, oldStatus.Generation); err != nil {
			return ProcessStatus{}, err
		}
	}
	s.mu.Lock()
	s.workloads[id] = candidates
	if s.processes[id] == nil {
		s.processes[id] = &managedProcess{status: ProcessStatus{WorkloadID: id, Role: group, State: "stopped", LogPath: s.logPath(id)}}
	}
	s.processes[id].status.Role = group
	err := s.saveLocked()
	s.mu.Unlock()
	var status ProcessStatus
	if err == nil {
		status, err = s.start(id, group, generation)
	}
	if err == nil {
		return status, nil
	}
	if len(previous) > 0 && oldStatus.State == "running" {
		s.mu.Lock()
		s.workloads[id] = previous
		s.mu.Unlock()
		restored, restoreErr := s.start(id, group, oldStatus.Generation)
		if restoreErr != nil {
			return ProcessStatus{}, fmt.Errorf("候选采集启动失败且旧版本恢复失败: %w", restoreErr)
		}
		return restored, fmt.Errorf("候选采集启动失败，已恢复旧版本: %w", err)
	}
	return status, err
}

// servicesForWorkload 只在 opMu 或 mu 持有期间调用，部署配置优先于静态配置。
func (s *Supervisor) servicesForWorkload(id string, group ServiceGroup) ([]ServiceConfig, bool) {
	if configured := s.workloads[id]; len(configured) > 0 {
		return configured, configured[0].Group == group
	}
	configured, ok := s.services[group]
	return configured, ok
}

type persistedProcess struct {
	ProcessStatus
	Services []ServiceConfig `json:"services,omitempty"`
}
