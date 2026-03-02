package handler

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// BootstrapStatus 初始化状态。
type BootstrapStatus string

const (
	BootstrapUninitialized          BootstrapStatus = "UNINITIALIZED"
	BootstrapCenterConfigured       BootstrapStatus = "CENTER_CONFIGURED"
	BootstrapRegisteredPending      BootstrapStatus = "REGISTERED_PENDING_APPROVAL"
	BootstrapReady                  BootstrapStatus = "READY"
	BootstrapError                  BootstrapStatus = "ERROR"
	defaultBootstrapStatePath                       = "./data/bootstrap.json"
)

// BootstrapState 初始化状态快照。
type BootstrapState struct {
	Status           BootstrapStatus `json:"status"`
	Mode             string          `json:"mode,omitempty"`
	CenterURL        string          `json:"centerUrl,omitempty"`
	NodeID           string          `json:"nodeId,omitempty"`
	NodeName         string          `json:"nodeName,omitempty"`
	LastError        string          `json:"lastError,omitempty"`
	AutoStartEnabled bool            `json:"autoStartEnabled"`
	UpdatedAt        string          `json:"updatedAt"`
}

// BootstrapStore 初始化状态存储。
type BootstrapStore struct {
	mu       sync.RWMutex
	filePath string
	state    BootstrapState
}

// NewBootstrapStore 创建初始化状态存储。
func NewBootstrapStore(filePath string) (*BootstrapStore, error) {
	if filePath == "" {
		filePath = defaultBootstrapStatePath
	}

	store := &BootstrapStore{
		filePath: filePath,
		state: BootstrapState{
			Status:    BootstrapUninitialized,
			UpdatedAt: time.Now().Format(time.RFC3339),
		},
	}

	if err := store.load(); err != nil {
		return nil, err
	}

	return store, nil
}

// Get 获取状态快照。
func (s *BootstrapStore) Get() BootstrapState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state
}

// Update 更新状态并持久化。
func (s *BootstrapStore) Update(updateFn func(state *BootstrapState)) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	updateFn(&s.state)
	s.state.UpdatedAt = time.Now().Format(time.RFC3339)

	return s.saveLocked()
}

// IsReady 判断是否已初始化完成。
func (s *BootstrapStore) IsReady() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state.Status == BootstrapReady
}

// load 加载状态文件。
func (s *BootstrapStore) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	content, err := os.ReadFile(s.filePath)
	if os.IsNotExist(err) {
		return s.saveLocked()
	}
	if err != nil {
		return err
	}

	var loaded BootstrapState
	if err := json.Unmarshal(content, &loaded); err != nil {
		return err
	}

	if loaded.Status == "" {
		loaded.Status = BootstrapUninitialized
	}
	if loaded.UpdatedAt == "" {
		loaded.UpdatedAt = time.Now().Format(time.RFC3339)
	}

	s.state = loaded
	return nil
}

// saveLocked 持久化状态（调用方需持有写锁）。
func (s *BootstrapStore) saveLocked() error {
	if err := os.MkdirAll(filepath.Dir(s.filePath), 0755); err != nil {
		return err
	}

	content, err := json.MarshalIndent(s.state, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.filePath, content, 0644)
}

