package sceneasset

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestResolveViewerSessionRefreshesIdleExpiryWithoutExceedingAbsoluteExpiry(t *testing.T) {
	now := time.Now().UTC()
	absoluteExpiry := now.Add(5 * time.Minute)
	encoded, err := json.Marshal(ViewerSession{
		ID: "viewer-1", ExpiresAt: now.Add(time.Minute), AbsoluteExpiresAt: absoluteExpiry,
	})
	if err != nil {
		t.Fatalf("序列化 Viewer 会话失败: %v", err)
	}
	store := &recordingViewerSessionStore{value: string(encoded)}
	service := NewService(nil, nil, nil, store, nil)

	resolved, err := service.ResolveViewerSession(context.Background(), "viewer-1")
	if err != nil {
		t.Fatalf("解析 Viewer 会话失败: %v", err)
	}
	if !resolved.ExpiresAt.Equal(absoluteExpiry) {
		t.Fatalf("空闲续期越过或未达到绝对期限: got %s, want %s", resolved.ExpiresAt, absoluteExpiry)
	}
	if store.ttl <= 0 || store.ttl > 5*time.Minute {
		t.Fatalf("Redis TTL 未限制在绝对期限内: %s", store.ttl)
	}
}

func TestResolveViewerSessionDeletesAbsolutelyExpiredSession(t *testing.T) {
	now := time.Now().UTC()
	encoded, err := json.Marshal(ViewerSession{
		ID: "viewer-1", ExpiresAt: now.Add(time.Minute), AbsoluteExpiresAt: now.Add(-time.Second),
	})
	if err != nil {
		t.Fatalf("序列化 Viewer 会话失败: %v", err)
	}
	store := &recordingViewerSessionStore{value: string(encoded)}
	service := NewService(nil, nil, nil, store, nil)

	_, err = service.ResolveViewerSession(context.Background(), "viewer-1")
	if !errors.Is(err, ErrSessionExpired) {
		t.Fatalf("绝对过期会话未被拒绝: %v", err)
	}
	if !store.deleted {
		t.Fatal("绝对过期会话未从短期存储删除")
	}
}

func TestOpenViewerFileRejectsPathTraversalBeforeRevisionLookup(t *testing.T) {
	now := time.Now().UTC()
	encoded, err := json.Marshal(ViewerSession{
		ID: "viewer-1", ExpiresAt: now.Add(time.Minute), AbsoluteExpiresAt: now.Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("序列化 Viewer 会话失败: %v", err)
	}
	service := NewService(nil, nil, nil, &recordingViewerSessionStore{value: string(encoded)}, nil)

	_, err = service.OpenViewerFile(context.Background(), "viewer-1", "../secret.json")
	if !errors.Is(err, ErrInvalidPath) {
		t.Fatalf("Viewer 路径逃逸未被拒绝: %v", err)
	}
}

type recordingViewerSessionStore struct {
	value   string
	ttl     time.Duration
	deleted bool
}

func (s *recordingViewerSessionStore) PutSceneSession(_ context.Context, _ string, value string, ttl time.Duration) error {
	s.value = value
	s.ttl = ttl
	return nil
}

func (s *recordingViewerSessionStore) GetSceneSession(context.Context, string) (string, error) {
	return s.value, nil
}

func (s *recordingViewerSessionStore) DeleteSceneSession(context.Context, string) error {
	s.deleted = true
	return nil
}
