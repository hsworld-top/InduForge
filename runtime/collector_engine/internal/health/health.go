package health

import (
	"net/http"
	"sync"
)

type State struct {
	mu    sync.RWMutex
	ready bool
}

func (s *State) SetReady(v bool) { s.mu.Lock(); s.ready = v; s.mu.Unlock() }
func (s *State) Ready() bool     { s.mu.RLock(); defer s.mu.RUnlock(); return s.ready }
func (s *State) Handler() http.Handler {
	mux := http.NewServeMux()
	// 存活探针只确认进程仍能响应，采集链路异常不应触发 Pod 重启。
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	// 就绪探针反映采集链路状态；未就绪时从 Service 端点摘除实例。
	mux.HandleFunc("/ready", func(w http.ResponseWriter, _ *http.Request) {
		if !s.Ready() {
			http.Error(w, "not ready", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	return mux
}
