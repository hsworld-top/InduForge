package service

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/google/uuid"
	apperrors "github.com/indu-forge/data_service/internal/errors"
)

type memoryProjectTenantBindingStore struct {
	bindings map[string]struct {
		tenant string
		epoch  int64
	}
}

func (s *memoryProjectTenantBindingStore) BindIfUnbound(_ context.Context, projectID, tenantID string, epoch int64) (string, int64, bool, error) {
	if existing, ok := s.bindings[projectID]; ok {
		return existing.tenant, existing.epoch, false, nil
	}
	s.bindings[projectID] = struct {
		tenant string
		epoch  int64
	}{tenantID, epoch}
	return tenantID, epoch, true, nil
}

func (s *memoryProjectTenantBindingStore) Get(_ context.Context, projectID, tenantID string) (int64, error) {
	item, ok := s.bindings[projectID]
	if !ok || item.tenant != tenantID {
		return 0, errors.New("not found")
	}
	return item.epoch, nil
}

func TestProjectTenantBindingService_BindsIdempotentlyAndRejectsRebinding(t *testing.T) {
	projectID := uuid.NewString()
	tenantID := uuid.NewString()
	otherTenantID := uuid.NewString()
	service := NewProjectTenantBindingService(&memoryProjectTenantBindingStore{bindings: map[string]struct {
		tenant string
		epoch  int64
	}{}})

	created, err := service.Bind(context.Background(), projectID, tenantID, "epoch-1")
	if err != nil || !created {
		t.Fatalf("expected initial bind to be created, created=%v err=%v", created, err)
	}
	created, err = service.Bind(context.Background(), projectID, tenantID, "epoch-1")
	if err != nil || created {
		t.Fatalf("expected same tenant bind to be idempotent, created=%v err=%v", created, err)
	}
	if _, err = service.Bind(context.Background(), projectID, tenantID, "epoch-2"); err == nil {
		t.Fatal("existing binding must reject a different epoch")
	}
	if epoch, getErr := service.Get(context.Background(), projectID, tenantID); getErr != nil || epoch != "epoch-1" {
		t.Fatalf("unexpected binding context: epoch=%q err=%v", epoch, getErr)
	}
	_, err = service.Bind(context.Background(), projectID, otherTenantID, "epoch-1")
	var appErr *apperrors.AppError
	if !errors.As(err, &appErr) || appErr.StatusCode != http.StatusConflict {
		t.Fatalf("expected conflict, got %v", err)
	}
	if appErr.Message == otherTenantID || appErr.Message == tenantID {
		t.Fatalf("conflict must not disclose tenant IDs: %q", appErr.Message)
	}
}

func TestProjectTenantBindingService_RejectsInvalidUUID(t *testing.T) {
	service := NewProjectTenantBindingService(&memoryProjectTenantBindingStore{bindings: map[string]struct {
		tenant string
		epoch  int64
	}{}})
	if _, err := service.Bind(context.Background(), "not-a-uuid", uuid.NewString(), "epoch-1"); err == nil {
		t.Fatal("expected invalid project UUID to be rejected")
	}
	if _, err := service.Bind(context.Background(), uuid.NewString(), "not-a-uuid", "epoch-1"); err == nil {
		t.Fatal("expected invalid tenant UUID to be rejected")
	}
}
