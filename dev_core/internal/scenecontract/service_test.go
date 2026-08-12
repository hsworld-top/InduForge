package scenecontract

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/indu-forge/dev_core/internal/auth"
	"github.com/indu-forge/dev_core/internal/project"
)

type fakeProjects struct{ item project.Project }

func (f fakeProjects) Get(context.Context, string, string) (project.Project, error) {
	return f.item, nil
}

func TestPutListAndDeleteContract(t *testing.T) {
	workspace := t.TempDir()
	service := NewService(fakeProjects{item: project.Project{ID: "project-1", TenantID: "tenant-1", WorkspacePath: workspace, Visibility: "private", CreatedBy: "developer-1"}})
	actor := auth.User{ID: "developer-1", TenantID: "tenant-1", Role: "DEVELOPER"}

	created, err := service.Put(context.Background(), actor, "project-1", "2d", "line-overview", Contract{
		Name:          "产线总览",
		EmbedMode:     "both",
		Inputs:        []Member{{Name: "lineId", Schema: map[string]any{"type": "string"}}},
		DatapointRefs: []string{"line.speed", "line.speed", "line.status"},
	})
	if err != nil {
		t.Fatalf("Put returned error: %v", err)
	}
	if created.ContractVersion == "" || len(created.DatapointRefs) != 2 {
		t.Fatalf("unexpected created contract: %#v", created)
	}

	snapshot, err := service.List(context.Background(), actor, "project-1")
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(snapshot.Contracts) != 1 || snapshot.ContractVersion == "" {
		t.Fatalf("unexpected snapshot: %#v", snapshot)
	}

	if err := service.Delete(context.Background(), actor, "project-1", "2d", "line-overview"); err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(workspace, ".induforge", "scenes", "2d", "line-overview.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("contract file still exists: %v", err)
	}
}

func TestPutRejectsInvalidContract(t *testing.T) {
	workspace := t.TempDir()
	service := NewService(fakeProjects{item: project.Project{ID: "project-1", TenantID: "tenant-1", WorkspacePath: workspace, Visibility: "private", CreatedBy: "developer-1"}})
	actor := auth.User{ID: "developer-1", TenantID: "tenant-1", Role: "DEVELOPER"}
	_, err := service.Put(context.Background(), actor, "project-1", "2d", "../escape", Contract{Name: "非法场景"})
	if !errors.Is(err, ErrInvalidContract) {
		t.Fatalf("expected ErrInvalidContract, got %v", err)
	}
}
