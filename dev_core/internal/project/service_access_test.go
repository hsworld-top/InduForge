package project_test

import (
	"context"
	"errors"
	"testing"

	"github.com/indu-forge/dev_core/internal/auth"
	"github.com/indu-forge/dev_core/internal/project"
)

func TestServiceFiltersPrivateAndSharedProjects(t *testing.T) {
	repository := &fakeRepository{
		projects: map[string]project.Project{
			"private-owned": {ID: "private-owned", TenantID: "tenant-1", CreatedBy: "developer-1", Visibility: "private"},
			"private-other": {ID: "private-other", TenantID: "tenant-1", CreatedBy: "developer-2", Visibility: "private"},
			"shared-other":  {ID: "shared-other", TenantID: "tenant-1", CreatedBy: "developer-2", Visibility: "internal"},
		},
	}
	service := project.NewService(repository, &fakeWorkspace{}, "admin123")
	service.SetTenantBindingEnsurer(noopTenantBindingEnsurer{})
	actor := auth.User{ID: "developer-1", TenantID: "tenant-1", Role: "DEVELOPER"}

	items, total, err := service.List(context.Background(), actor, project.ListFilter{Page: 1, Limit: 20})
	if err != nil {
		t.Fatalf("查询工程列表失败: %v", err)
	}
	if total != 2 || len(items) != 2 {
		t.Fatalf("开发人员只应看到自己的私有工程和分享工程: total=%d items=%#v", total, items)
	}
}

func TestServiceRejectsPrivateProjectForOtherDeveloper(t *testing.T) {
	repository := &fakeRepository{projects: map[string]project.Project{
		"private-project": {ID: "private-project", TenantID: "tenant-1", CreatedBy: "owner-1", Visibility: "private"},
	}}
	service := project.NewService(repository, &fakeWorkspace{}, "admin123")
	service.SetTenantBindingEnsurer(noopTenantBindingEnsurer{})
	actor := auth.User{ID: "developer-2", TenantID: "tenant-1", Role: "DEVELOPER"}

	_, err := service.Get(context.Background(), actor, "private-project")
	if !errors.Is(err, auth.ErrPermissionDenied) {
		t.Fatalf("其他开发人员访问私有工程应失败: %v", err)
	}
}

func TestServiceSharedProjectWriteAndOwnerOnlyDelete(t *testing.T) {
	repository := &fakeRepository{projects: map[string]project.Project{
		"shared-project": {ID: "shared-project", TenantID: "tenant-1", CreatedBy: "owner-1", Visibility: "internal", Name: "原名称"},
	}}
	service := project.NewService(repository, &fakeWorkspace{files: map[string]map[string]string{"shared-project": {}}}, "admin123")
	service.SetTenantBindingEnsurer(noopTenantBindingEnsurer{})
	developer := auth.User{ID: "developer-2", TenantID: "tenant-1", Role: "DEVELOPER"}
	newName := "更新名称"

	updated, err := service.Update(context.Background(), developer, "shared-project", project.ProjectInput{Name: &newName})
	if err != nil || updated.Name != newName {
		t.Fatalf("开发人员应可修改分享工程: item=%#v err=%v", updated, err)
	}
	if err := service.Delete(context.Background(), developer, "shared-project"); !errors.Is(err, auth.ErrPermissionDenied) {
		t.Fatalf("非创建人不能删除分享工程: %v", err)
	}
}

func TestServiceCreateAlwaysStartsPrivate(t *testing.T) {
	repository := &fakeRepository{projects: map[string]project.Project{}}
	service := project.NewService(repository, &fakeWorkspace{}, "admin123")
	service.SetTenantBindingEnsurer(noopTenantBindingEnsurer{})
	actor := auth.User{ID: "developer-1", TenantID: "tenant-1", Username: "开发人员", Role: "DEVELOPER"}
	name := "新工程"
	requestedVisibility := "internal"

	created, err := service.Create(context.Background(), actor, project.ProjectInput{Name: &name, Visibility: &requestedVisibility})
	if err != nil {
		t.Fatalf("创建工程失败: %v", err)
	}
	if created.Visibility != "private" {
		t.Fatalf("新工程必须从私有状态开始: %q", created.Visibility)
	}
}
