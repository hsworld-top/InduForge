package auth

import "strings"

type Capability string

const (
	CapabilityTenantManage        Capability = "tenant:manage"
	CapabilityUserManage          Capability = "user:manage"
	CapabilityProjectRead         Capability = "project:read"
	CapabilityProjectCreate       Capability = "project:create"
	CapabilityProjectWrite        Capability = "project:write"
	CapabilityProjectShare        Capability = "project:share"
	CapabilityProjectDelete       Capability = "project:delete"
	CapabilityRuntimeAccessManage Capability = "runtime-access:manage"
	CapabilityReleasePublish      Capability = "release:publish"
	CapabilityDeploymentExecute   Capability = "deployment:execute"
	CapabilityDeploymentOperate   Capability = "deployment:operate"
	CapabilityNodeRead            Capability = "node:read"
	CapabilityNodeApprove         Capability = "node:approve"
	CapabilityNodeDelete          Capability = "node:delete"
	CapabilityEnvironmentRead     Capability = "environment:read"
	CapabilityEnvironmentManage   Capability = "environment:manage"
	CapabilityAuditLogRead        Capability = "audit-log:read"
	CapabilityAuditLogDelete      Capability = "audit-log:delete"
)

var allCapabilities = capabilitySet(
	CapabilityUserManage,
	CapabilityProjectRead,
	CapabilityProjectCreate,
	CapabilityProjectWrite,
	CapabilityProjectShare,
	CapabilityProjectDelete,
	CapabilityRuntimeAccessManage,
	CapabilityReleasePublish,
	CapabilityDeploymentExecute,
	CapabilityDeploymentOperate,
	CapabilityNodeRead,
	CapabilityNodeApprove,
	CapabilityNodeDelete,
	CapabilityEnvironmentRead,
	CapabilityEnvironmentManage,
	CapabilityAuditLogRead,
	CapabilityAuditLogDelete,
)

var roleCapabilities = map[string]map[Capability]struct{}{
	"SUPER_ADMIN":  capabilitySet(CapabilityTenantManage),
	"SYSTEM_ADMIN": allCapabilities,
	"PROJECT_ADMIN": capabilitySet(
		CapabilityProjectRead,
		CapabilityProjectCreate,
		CapabilityProjectWrite,
		CapabilityProjectShare,
		CapabilityProjectDelete,
		CapabilityRuntimeAccessManage,
		CapabilityReleasePublish,
		CapabilityDeploymentExecute,
		CapabilityDeploymentOperate,
		CapabilityNodeRead,
		CapabilityEnvironmentRead,
		CapabilityAuditLogRead,
	),
	"OPS_ADMIN": capabilitySet(
		CapabilityProjectRead,
		CapabilityProjectCreate,
		CapabilityProjectWrite,
		CapabilityProjectShare,
		CapabilityProjectDelete,
		CapabilityRuntimeAccessManage,
		CapabilityReleasePublish,
		CapabilityDeploymentExecute,
		CapabilityDeploymentOperate,
		CapabilityNodeRead,
		CapabilityNodeApprove,
		CapabilityNodeDelete,
		CapabilityEnvironmentRead,
		CapabilityEnvironmentManage,
		CapabilityAuditLogRead,
	),
}

func HasCapability(role string, capability Capability) bool {
	capabilities, ok := roleCapabilities[normalizeRole(role)]
	if !ok {
		return false
	}
	_, ok = capabilities[capability]
	return ok
}

func RequireCapability(user User, capability Capability) error {
	if !HasCapability(user.Role, capability) {
		return ErrPermissionDenied
	}
	return nil
}

func CanAssignRole(actorRole, targetRole string) bool {
	actorRole = normalizeRole(actorRole)
	targetRole = normalizeRole(targetRole)
	if _, exists := roleCapabilities[targetRole]; !exists {
		return false
	}
	switch actorRole {
	case "SUPER_ADMIN":
		return true
	case "SYSTEM_ADMIN":
		return targetRole != "SUPER_ADMIN"
	default:
		return false
	}
}

func IsTenantAdministrator(role string) bool {
	role = normalizeRole(role)
	return role == "SYSTEM_ADMIN"
}

func capabilitySet(capabilities ...Capability) map[Capability]struct{} {
	result := make(map[Capability]struct{}, len(capabilities))
	for _, capability := range capabilities {
		result[capability] = struct{}{}
	}
	return result
}

func normalizeRole(role string) string {
	return strings.ToUpper(strings.TrimSpace(role))
}
