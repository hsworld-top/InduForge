package project_test

import (
	"errors"
	"testing"

	"github.com/indu-forge/dev_core/internal/auth"
	"github.com/indu-forge/dev_core/internal/project"
)

func TestProjectVisibilityAccess(t *testing.T) {
	privateProject := project.Project{TenantID: "tenant-1", CreatedBy: "owner-1", Visibility: "private"}
	sharedProject := project.Project{TenantID: "tenant-1", CreatedBy: "owner-1", Visibility: "internal"}

	tests := []struct {
		name    string
		actor   auth.User
		item    project.Project
		allowed bool
	}{
		{name: "私有工程创建人可访问", actor: auth.User{ID: "owner-1", TenantID: "tenant-1", Role: "PROJECT_ADMIN"}, item: privateProject, allowed: true},
		{name: "同租户其他开发人员不能访问私有工程", actor: auth.User{ID: "developer-2", TenantID: "tenant-1", Role: "PROJECT_ADMIN"}, item: privateProject, allowed: false},
		{name: "系统管理员可访问私有工程", actor: auth.User{ID: "admin-1", TenantID: "tenant-1", Role: "SYSTEM_ADMIN"}, item: privateProject, allowed: true},
		{name: "运维人员可访问分享工程", actor: auth.User{ID: "viewer-1", TenantID: "tenant-1", Role: "OPS_ADMIN"}, item: sharedProject, allowed: true},
		{name: "用户管理员不能访问分享工程", actor: auth.User{ID: "user-admin-1", TenantID: "tenant-1", Role: "USER_ADMIN"}, item: sharedProject, allowed: false},
		{name: "其他租户不能访问分享工程", actor: auth.User{ID: "admin-2", TenantID: "tenant-2", Role: "SYSTEM_ADMIN"}, item: sharedProject, allowed: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := project.CanRead(test.actor, test.item); got != test.allowed {
				t.Fatalf("CanRead() = %v, want %v", got, test.allowed)
			}
		})
	}
}

func TestProjectCapabilityRequiresVisibilityAndRole(t *testing.T) {
	item := project.Project{TenantID: "tenant-1", CreatedBy: "owner-1", Visibility: "internal"}
	if err := project.RequireCapability(auth.User{ID: "developer-2", TenantID: "tenant-1", Role: "PROJECT_ADMIN"}, item, auth.CapabilityProjectWrite); err != nil {
		t.Fatalf("分享工程应允许开发人员修改: %v", err)
	}
	err := project.RequireCapability(auth.User{ID: "viewer-1", TenantID: "tenant-1", Role: "OPS_ADMIN"}, item, auth.CapabilityProjectWrite)
	if err != nil {
		t.Fatalf("运维人员应具备工程编辑能力: %v", err)
	}
}

func TestProjectOwnerOrAdminRestriction(t *testing.T) {
	item := project.Project{TenantID: "tenant-1", CreatedBy: "owner-1", Visibility: "internal"}
	if err := project.RequireOwnerOrAdmin(auth.User{ID: "owner-1", TenantID: "tenant-1", Role: "PROJECT_ADMIN"}, item, auth.CapabilityProjectDelete); err != nil {
		t.Fatalf("创建人应可删除自己的工程: %v", err)
	}
	if err := project.RequireOwnerOrAdmin(auth.User{ID: "admin-1", TenantID: "tenant-1", Role: "SYSTEM_ADMIN"}, item, auth.CapabilityProjectDelete); err != nil {
		t.Fatalf("管理员应可删除分享工程: %v", err)
	}
	err := project.RequireOwnerOrAdmin(auth.User{ID: "developer-2", TenantID: "tenant-1", Role: "PROJECT_ADMIN"}, item, auth.CapabilityProjectDelete)
	if !errors.Is(err, auth.ErrPermissionDenied) {
		t.Fatalf("非创建人的开发人员不能删除分享工程: %v", err)
	}
}
