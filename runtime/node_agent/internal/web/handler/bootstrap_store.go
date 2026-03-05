package handler

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// BootstrapStatus 初始化状态。
type BootstrapStatus string

const (
	BootstrapUninitialized       BootstrapStatus = "UNINITIALIZED"
	BootstrapCenterConfigured    BootstrapStatus = "CENTER_CONFIGURED"
	BootstrapRegisteredPending   BootstrapStatus = "REGISTERED_PENDING_APPROVAL"
	BootstrapReady               BootstrapStatus = "READY"
	BootstrapError               BootstrapStatus = "ERROR"
	defaultBootstrapStateDataDir                 = "./data"
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

// BootstrapStore 初始化状态存储（SQLite）。
type BootstrapStore struct {
	mu      sync.RWMutex
	dataDir string
	db      *sql.DB
	state   BootstrapState
}

// NewBootstrapStore 创建初始化状态存储。
func NewBootstrapStore(dataDir string) (*BootstrapStore, error) {
	if dataDir == "" {
		dataDir = defaultBootstrapStateDataDir
	}

	store := &BootstrapStore{
		dataDir: dataDir,
		state: BootstrapState{
			Status:    BootstrapUninitialized,
			UpdatedAt: time.Now().Format(time.RFC3339),
		},
	}
	if err := store.initSQLite(); err != nil {
		return nil, err
	}
	if err := store.load(); err != nil {
		_ = store.Close()
		return nil, err
	}
	return store, nil
}

// Close 关闭连接。
func (s *BootstrapStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db == nil {
		return nil
	}
	err := s.db.Close()
	s.db = nil
	return err
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

func (s *BootstrapStore) initSQLite() error {
	if err := os.MkdirAll(s.dataDir, 0755); err != nil {
		return err
	}
	dbPath := filepath.Join(s.dataDir, "node_agent.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return err
	}
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS bootstrap_state (
		id INTEGER PRIMARY KEY CHECK (id = 1),
		state_json TEXT NOT NULL,
		updated_at TEXT NOT NULL
	);`); err != nil {
		_ = db.Close()
		return err
	}
	s.db = db
	return nil
}

// load 加载状态。
func (s *BootstrapStore) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var (
		rawState  string
		updatedAt string
	)
	err := s.db.QueryRow(`SELECT state_json, updated_at FROM bootstrap_state WHERE id = 1`).Scan(&rawState, &updatedAt)
	if err == sql.ErrNoRows {
		return s.saveLocked()
	}
	if err != nil {
		return err
	}

	var loaded BootstrapState
	if err := json.Unmarshal([]byte(rawState), &loaded); err != nil {
		return err
	}
	if loaded.Status == "" {
		loaded.Status = BootstrapUninitialized
	}
	if loaded.UpdatedAt == "" {
		loaded.UpdatedAt = updatedAt
	}
	if loaded.UpdatedAt == "" {
		loaded.UpdatedAt = time.Now().Format(time.RFC3339)
	}

	s.state = loaded
	return nil
}

// saveLocked 持久化状态（调用方需持有写锁）。
func (s *BootstrapStore) saveLocked() error {
	content, err := json.Marshal(s.state)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(
		`INSERT INTO bootstrap_state (id, state_json, updated_at)
		 VALUES (1, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET
		   state_json=excluded.state_json,
		   updated_at=excluded.updated_at`,
		string(content),
		s.state.UpdatedAt,
	)
	return err
}
