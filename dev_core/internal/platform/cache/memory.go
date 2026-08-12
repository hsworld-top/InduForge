package cache

import (
	"context"
	"errors"
	"sync"
	"time"
)

var ErrMiss = errors.New("缓存不存在或已过期")

type memoryItem struct {
	value     string
	expiresAt time.Time
}

type Memory struct {
	mu    sync.Mutex
	items map[string]memoryItem
	now   func() time.Time
}

func NewMemory() *Memory {
	return &Memory{items: make(map[string]memoryItem), now: time.Now}
}

func (m *Memory) Put(_ context.Context, key, value string, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items[key] = memoryItem{value: value, expiresAt: m.now().Add(ttl)}
	return nil
}

func (m *Memory) Get(_ context.Context, key string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	item, exists := m.items[key]
	if !exists || !m.now().Before(item.expiresAt) {
		delete(m.items, key)
		return "", ErrMiss
	}
	return item.value, nil
}

func (m *Memory) Take(_ context.Context, key string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	item, exists := m.items[key]
	delete(m.items, key)
	if !exists || !m.now().Before(item.expiresAt) {
		return "", ErrMiss
	}
	return item.value, nil
}

func (m *Memory) Delete(_ context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.items, key)
	return nil
}
