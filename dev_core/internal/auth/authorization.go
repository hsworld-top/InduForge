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
	CapabilityAuditLogRead        Capability = "audit-log:read"
	CapabilityAuditLogDelete      Capability = "audit-log:delete"
)

var allCapabilities = capabilitySet(
	CapabilityTenantManage,
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
	CapabilityAuditLogRead,
	CapabilityAuditLogDelete,
)

var roleCapabilities = map[string]map[Capability]struct{}{
	"SUPER_ADMIN":  allCapabilities,
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
		CapabilityAuditLogRead,
	),
	"OPS_ADMIN": capabilitySet(
		CapabilityProjectRead,
		CapabilityDeploymentExecute,
		CapabilityDeploymentOperate,
		CapabilityNodeRead,
		CapabilityNodeApprove,
		CapabilityNodeDelete,
		CapabilityAuditLogRead,
	),
	"USER_ADMIN": capabilitySet(
		CapabilityUserManage,
	),
	"DEVELOPER": capabilitySet(
		CapabilityProjectRead,
		CapabilityProjectCreate,
		CapabilityProjectWrite,
		CapabilityProjectShare,
		CapabilityProjectDelete,
		CapabilityReleasePublish,
	),
	"OPERATOR": capabilitySet(
		CapabilityProjectRead,
		CapabilityDeploymentOperate,
	),
	"VIEWER": capabilitySet(
		CapabilityProjectRead,
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
	case "USER_ADMIN":
		return targetRole == "DEVELOPER" || targetRole == "OPERATOR" || targetRole == "VIEWER"
	default:
		return false
	}
}

func IsPlatformAdmin(role string) bool {
	role = normalizeRole(role)
	return role == "SUPER_ADMIN" || role == "SYSTEM_ADMIN"
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
