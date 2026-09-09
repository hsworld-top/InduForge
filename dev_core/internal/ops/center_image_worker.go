package ops

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/indu-forge/dev_core/internal/imagecatalog"
	"sync"
	"time"
)

type centerImageJob struct {
	running bool
	ready   map[string]time.Time
	retryAt time.Time
	failure error
}
type centerImageWorkers struct {
	sync.Mutex
	ctx   context.Context
	slots chan struct{}
	nodes map[string]*centerImageJob
}

// 中心内置记录可能属于不同组织，但共享一个物理宿主；只运行一个导入任务以限制磁盘峰值。
func (r *PostgreSQLRepository) SetImageWorkerContext(ctx context.Context) {
	r.centerImageWorkers = &centerImageWorkers{ctx: ctx, slots: make(chan struct{}, 1), nodes: map[string]*centerImageJob{}}
}
func (r *PostgreSQLRepository) ensureCenterImagesAsync(n Node, artifacts []imagecatalog.Artifact) error {
	w := r.centerImageWorkers
	if w == nil {
		return fmt.Errorf("中心镜像工作器尚未启动")
	}
	w.Lock()
	defer w.Unlock()
	if e := w.ctx.Err(); e != nil {
		return e
	}
	j := w.nodes[n.ID]
	if j == nil {
		j = &centerImageJob{ready: map[string]time.Time{}}
		w.nodes[n.ID] = j
	}
	now := time.Now()
	all := true
	for _, a := range artifacts {
		if now.Sub(j.ready[a.SHA256]) > time.Minute {
			all = false
			break
		}
	}
	if all {
		return nil
	}
	if j.running {
		return &imagePreparationPending{message: "中心正在准备运行资源"}
	}
	if j.failure != nil && now.Before(j.retryAt) {
		return j.failure
	}
	select {
	case w.slots <- struct{}{}:
	default:
		return &imagePreparationPending{message: "等待中心运行资源下载任务"}
	}
	j.running = true
	j.failure = nil
	// 使用主进程生命周期，不继承短暂HTTP/调和请求的context；每次请求都立即返回pending。
	go func() {
		ctx, cancel := context.WithTimeout(w.ctx, 30*time.Minute)
		defer cancel()
		notify := func(state ImageState) { r.recordCenterImageState(ctx, n, state) }
		notify(ImageState{Status: "downloading", Message: "正在准备中心运行资源"})
		err := r.prepareCenterImagesWithProgress(ctx, artifacts, notify)
		state := ImageState{Status: "ready", Message: "中心运行资源已就绪"}
		if err != nil {
			state.Status = "failed"
			state.Message = err.Error()
		} else {
			for _, a := range artifacts {
				state.Artifacts = append(state.Artifacts, a.SHA256)
			}
		}
		notify(state)
		w.Lock()
		j.running = false
		j.failure = err
		j.retryAt = time.Now().Add(30 * time.Second)
		if err == nil {
			for _, a := range artifacts {
				j.ready[a.SHA256] = time.Now()
			}
		}
		for hash, at := range j.ready {
			if time.Since(at) > time.Hour {
				delete(j.ready, hash)
			}
		}
		<-w.slots
		w.Unlock()
	}()
	return &imagePreparationPending{message: "中心正在准备运行资源"}
}
func (r *PostgreSQLRepository) recordCenterImageState(ctx context.Context, n Node, state ImageState) {
	if r.pool == nil {
		return
	}
	b, _ := json.Marshal(state)
	_, err := r.pool.Exec(ctx, `UPDATE host_nodes SET resource_summary=resource_summary||jsonb_build_object('imageState',$2::jsonb),updated_at=now() WHERE id=$1 AND node_source='built_in'`, n.ID, b)
	if err == nil {
		r.publish(n.TenantID, []string{"nodes", "environments", "deployments"}, []string{n.ID}, state.Status == "ready" || state.Status == "failed")
	}
}
