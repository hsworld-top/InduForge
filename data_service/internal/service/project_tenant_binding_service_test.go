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
	bindings map[string]string
}

func (s *memoryProjectTenantBindingStore) BindIfUnbound(_ context.Context, projectID, tenantID string) (string, bool, error) {
	if existing, ok := s.bindings[projectID]; ok {
		return existing, false, nil
	}
	s.bindings[projectID] = tenantID
	return tenantID, true, nil
}

func TestProjectTenantBindingService_BindsIdempotentlyAndRejectsRebinding(t *testing.T) {
	projectID := uuid.NewString()
	tenantID := uuid.NewString()
	otherTenantID := uuid.NewString()
	service := NewProjectTenantBindingService(&memoryProjectTenantBindingStore{bindings: map[string]string{}})

	created, err := service.Bind(context.Background(), projectID, tenantID)
	if err != nil || !created {
		t.Fatalf("expected initial bind to be created, created=%v err=%v", created, err)
	}
	created, err = service.Bind(context.Background(), projectID, tenantID)
	if err != nil || created {
		t.Fatalf("expected same tenant bind to be idempotent, created=%v err=%v", created, err)
	}
	_, err = service.Bind(context.Background(), projectID, otherTenantID)
	var appErr *apperrors.AppError
	if !errors.As(err, &appErr) || appErr.StatusCode != http.StatusConflict {
		t.Fatalf("expected conflict, got %v", err)
	}
	if appErr.Message == otherTenantID || appErr.Message == tenantID {
		t.Fatalf("conflict must not disclose tenant IDs: %q", appErr.Message)
	}
}

func TestProjectTenantBindingService_RejectsInvalidUUID(t *testing.T) {
	service := NewProjectTenantBindingService(&memoryProjectTenantBindingStore{bindings: map[string]string{}})
	if _, err := service.Bind(context.Background(), "not-a-uuid", uuid.NewString()); err == nil {
		t.Fatal("expected invalid project UUID to be rejected")
	}
	if _, err := service.Bind(context.Background(), uuid.NewString(), "not-a-uuid"); err == nil {
		t.Fatal("expected invalid tenant UUID to be rejected")
	}
}
