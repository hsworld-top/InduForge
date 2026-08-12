package project

import "github.com/indu-forge/dev_core/internal/auth"

func CanRead(actor auth.User, item Project) bool {
	if actor.TenantID == "" || actor.TenantID != item.TenantID {
		return false
	}
	if auth.IsPlatformAdmin(actor.Role) || actor.ID == item.CreatedBy {
		return true
	}
	return item.Visibility == "internal" && auth.HasCapability(actor.Role, auth.CapabilityProjectRead)
}

func RequireCapability(actor auth.User, item Project, capability auth.Capability) error {
	if !CanRead(actor, item) {
		return auth.ErrPermissionDenied
	}
	return auth.RequireCapability(actor, capability)
}

func RequireOwnerOrAdmin(actor auth.User, item Project, capability auth.Capability) error {
	if actor.TenantID == "" || actor.TenantID != item.TenantID {
		return auth.ErrPermissionDenied
	}
	if err := auth.RequireCapability(actor, capability); err != nil {
		return err
	}
	if actor.ID == item.CreatedBy || auth.IsPlatformAdmin(actor.Role) || actor.Role == "PROJECT_ADMIN" {
		return nil
	}
	return auth.ErrPermissionDenied
}
