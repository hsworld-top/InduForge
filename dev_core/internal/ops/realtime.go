package ops

import (
	"crypto/sha256"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/indu-forge/dev_core/internal/auth"
	platformapi "github.com/indu-forge/dev_core/internal/platform/api"
)

// ChangePublisher 的内容只能是失效范围，不接受任意业务payload，从类型边界杜绝凭据泄露。
type ChangePublisher interface {
	PublishChange(string, []string, []string, bool)
}

type changeFingerprint struct {
	digest [32]byte
	at     time.Time
}
type changeObserver struct {
	mu              sync.Mutex
	values          map[string]changeFingerprint
	freshnessCursor string
}

func (h *Handler) SetEvents(events ChangePublisher)              { h.changes = events }
func (r *PostgreSQLRepository) SetEvents(events ChangePublisher) { r.events = events }

func (r *PostgreSQLRepository) publish(tenant string, topics, ids []string, terminal bool) {
	if r.events != nil {
		r.events.PublishChange(tenant, topics, ids, terminal)
	}
}

// observedChange 只比较已提交的安全投影；指纹缓存有界且30分钟过期，历史实体不会无限增长。
func (r *PostgreSQLRepository) observedChange(tenant string, topics, ids []string, terminal bool, state any) {
	if r.events == nil {
		return
	}
	raw, err := json.Marshal(state)
	if err != nil {
		return
	}
	key := tenant + "/" + strings.Join(topics, ",") + "/" + strings.Join(ids, ",")
	digest := sha256.Sum256(raw)
	now := time.Now()
	r.observer.mu.Lock()
	if r.observer.values == nil {
		r.observer.values = map[string]changeFingerprint{}
	}
	old, exists := r.observer.values[key]
	if exists && old.digest == digest && now.Sub(old.at) < 30*time.Minute {
		r.observer.mu.Unlock()
		return
	}
	if len(r.observer.values) >= 4096 {
		for k, item := range r.observer.values {
			if now.Sub(item.at) >= 30*time.Minute {
				delete(r.observer.values, k)
			}
		}
		if len(r.observer.values) >= 4096 {
			for k := range r.observer.values {
				delete(r.observer.values, k)
				break
			}
		}
	}
	r.observer.values[key] = changeFingerprint{digest, now}
	r.observer.mu.Unlock()
	r.publish(tenant, topics, ids, terminal)
}

// writeSuccess 仅在业务成功、仓储事务已返回后发布结构/操作失效。读取和Agent命令不触发。
// 使用broad通知使新增/删除/成员变更同时更新分页total及当前展开详情，不从响应复制任何secret。
func (h *Handler) writeSuccess(w http.ResponseWriter, request *http.Request, data any) {
	platformapi.WriteSuccess(w, request, data)
	if h.changes == nil || request.Method == http.MethodGet || strings.Contains(request.URL.Path, "/agent/") {
		return
	}
	actor, ok := auth.UserFromContext(request.Context())
	if !ok {
		return
	}
	var topics []string
	switch {
	case strings.Contains(request.URL.Path, "/project-deployments"):
		topics = []string{"deployments", "environments", "events"}
	case strings.Contains(request.URL.Path, "/runtime-environments"):
		topics = []string{"environments", "nodes", "events"}
	case strings.Contains(request.URL.Path, "/nodes"), strings.Contains(request.URL.Path, "/node-enrollments"):
		topics = []string{"nodes", "environments", "events"}
	default:
		return
	}
	h.changes.PublishChange(actor.TenantID, topics, nil, true)
}
