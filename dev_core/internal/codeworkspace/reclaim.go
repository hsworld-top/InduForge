package codeworkspace

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

const WorkspaceIdleTimeout = 30 * time.Minute

type workspaceActivity struct {
	last     time.Time
	requests int
}
type activityRegistry struct {
	sync.Mutex
	projects map[string]*workspaceActivity
}
type reclaimEngine interface {
	RunningProjects(context.Context) ([]string, error)
	WorkspaceBusy(context.Context, string) (bool, error)
}

// 只统计经鉴权的工作区流量；长连接存续期间视为使用中，关闭后才开始空闲计时。
func (s *Service) beginActivity(id string) func() {
	s.activity.Lock()
	if s.activity.projects == nil {
		s.activity.projects = map[string]*workspaceActivity{}
	}
	item := s.activity.projects[id]
	if item == nil {
		item = &workspaceActivity{}
		s.activity.projects[id] = item
	}
	item.last = s.now()
	item.requests++
	s.activity.Unlock()
	return func() { s.activity.Lock(); item.last = s.now(); item.requests--; s.activity.Unlock() }
}

// 中心重启后重新观察完整空闲窗口；探测失败不删除，防止误伤仍在执行的任务。
func (s *Service) ReclaimIdle(ctx context.Context) error {
	engine, ok := s.engine.(reclaimEngine)
	if !ok {
		return nil
	}
	ids, err := engine.RunningProjects(ctx)
	if err != nil {
		return err
	}
	for _, id := range ids {
		s.activity.Lock()
		if s.activity.projects == nil {
			s.activity.projects = map[string]*workspaceActivity{}
		}
		item := s.activity.projects[id]
		if item == nil {
			s.activity.projects[id] = &workspaceActivity{last: s.now()}
			s.activity.Unlock()
			continue
		}
		eligible := item.requests == 0 && s.now().Sub(item.last) >= WorkspaceIdleTimeout
		last := item.last
		s.activity.Unlock()
		if !eligible {
			continue
		}
		busy, probeErr := engine.WorkspaceBusy(ctx, id)
		s.activity.Lock()
		// 探测期间重新访问，或任务仍运行时，重新计算完整空闲窗口。
		if probeErr != nil || busy {
			item.last = s.now()
			s.activity.Unlock()
			continue
		}
		if item.requests != 0 || !item.last.Equal(last) {
			s.activity.Unlock()
			continue
		}
		err = s.engine.Stop(ctx, containerName(id))
		item.last = s.now()
		s.activity.Unlock()
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) RunIdleReclaimer(ctx context.Context, logger *slog.Logger) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			checkCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
			if err := s.ReclaimIdle(checkCtx); err != nil {
				logger.Warn("开发环境空闲回收检查失败", "error", err)
			}
			cancel()
		}
	}
}
