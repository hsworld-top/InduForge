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
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	mux.HandleFunc("/ready", func(w http.ResponseWriter, _ *http.Request) {
		if !s.Ready() {
			http.Error(w, "not ready", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	return mux
}
