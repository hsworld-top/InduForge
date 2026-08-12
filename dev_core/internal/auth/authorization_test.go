package auth_test

import (
	"errors"
	"testing"

	"github.com/indu-forge/dev_core/internal/auth"
)

func TestHasCapability(t *testing.T) {
	tests := []struct {
		name       string
		role       string
		capability auth.Capability
		allowed    bool
	}{
		{name: "超级管理员拥有全部能力", role: "SUPER_ADMIN", capability: auth.CapabilityTenantManage, allowed: true},
		{name: "系统管理员拥有租户内全部能力", role: "SYSTEM_ADMIN", capability: auth.CapabilityProjectDelete, allowed: true},
		{name: "工程管理员可以管理运行权限", role: "PROJECT_ADMIN", capability: auth.CapabilityRuntimeAccessManage, allowed: true},
		{name: "运维管理员可以审批节点", role: "OPS_ADMIN", capability: auth.CapabilityNodeApprove, allowed: true},
		{name: "用户管理员不能读取工程", role: "USER_ADMIN", capability: auth.CapabilityProjectRead, allowed: false},
		{name: "开发人员可以修改工程", role: "DEVELOPER", capability: auth.CapabilityProjectWrite, allowed: true},
		{name: "开发人员不能执行正式部署", role: "DEVELOPER", capability: auth.CapabilityDeploymentExecute, allowed: false},
		{name: "操作员可以操作已有部署", role: "OPERATOR", capability: auth.CapabilityDeploymentOperate, allowed: true},
		{name: "观察员只能读取工程", role: "VIEWER", capability: auth.CapabilityProjectRead, allowed: true},
		{name: "观察员不能修改工程", role: "VIEWER", capability: auth.CapabilityProjectWrite, allowed: false},
		{name: "未知角色没有能力", role: "UNKNOWN", capability: auth.CapabilityProjectRead, allowed: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := auth.HasCapability(test.role, test.capability); got != test.allowed {
				t.Fatalf("HasCapability(%q, %q) = %v, want %v", test.role, test.capability, got, test.allowed)
			}
		})
	}
}

func TestRequireCapabilityReturnsPermissionError(t *testing.T) {
	user := auth.User{Role: "VIEWER"}
	err := auth.RequireCapability(user, auth.CapabilityProjectWrite)
	if !errors.Is(err, auth.ErrPermissionDenied) {
		t.Fatalf("权限不足应返回 ErrPermissionDenied，got %v", err)
	}
}

func TestCanAssignRole(t *testing.T) {
	tests := []struct {
		actorRole  string
		targetRole string
		allowed    bool
	}{
		{actorRole: "USER_ADMIN", targetRole: "DEVELOPER", allowed: true},
		{actorRole: "USER_ADMIN", targetRole: "OPERATOR", allowed: true},
		{actorRole: "USER_ADMIN", targetRole: "VIEWER", allowed: true},
		{actorRole: "USER_ADMIN", targetRole: "PROJECT_ADMIN", allowed: false},
		{actorRole: "SYSTEM_ADMIN", targetRole: "OPS_ADMIN", allowed: true},
		{actorRole: "SYSTEM_ADMIN", targetRole: "SUPER_ADMIN", allowed: false},
		{actorRole: "SUPER_ADMIN", targetRole: "SUPER_ADMIN", allowed: true},
		{actorRole: "PROJECT_ADMIN", targetRole: "VIEWER", allowed: false},
		{actorRole: "UNKNOWN", targetRole: "VIEWER", allowed: false},
	}

	for _, test := range tests {
		name := test.actorRole + "_to_" + test.targetRole
		t.Run(name, func(t *testing.T) {
			if got := auth.CanAssignRole(test.actorRole, test.targetRole); got != test.allowed {
				t.Fatalf("CanAssignRole(%q, %q) = %v, want %v", test.actorRole, test.targetRole, got, test.allowed)
			}
		})
	}
}

func TestIsPlatformAdmin(t *testing.T) {
	if !auth.IsPlatformAdmin("SUPER_ADMIN") || !auth.IsPlatformAdmin("SYSTEM_ADMIN") {
		t.Fatal("SUPER_ADMIN 和 SYSTEM_ADMIN 应为平台管理员")
	}
	if auth.IsPlatformAdmin("PROJECT_ADMIN") {
		t.Fatal("PROJECT_ADMIN 不应绕过工程可见性")
	}
}
