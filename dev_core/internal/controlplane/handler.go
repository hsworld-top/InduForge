package controlplane

import (
	"net/http"

	"github.com/indu-forge/dev_core/internal/auditlog"
	"github.com/indu-forge/dev_core/internal/auth"
	"github.com/indu-forge/dev_core/internal/codeworkspace"
	"github.com/indu-forge/dev_core/internal/contextpack"
	"github.com/indu-forge/dev_core/internal/deployment"
	"github.com/indu-forge/dev_core/internal/node"
	platformapi "github.com/indu-forge/dev_core/internal/platform/api"
	"github.com/indu-forge/dev_core/internal/project"
	"github.com/indu-forge/dev_core/internal/runtimeaccess"
	"github.com/indu-forge/dev_core/internal/tenant"
	"github.com/indu-forge/dev_core/internal/user"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

type Handler struct {
	platformapi.Unimplemented
	auth          *auth.Handler
	tenant        *tenant.Handler
	user          *user.Handler
	project       *project.Handler
	codeWorkspace *codeworkspace.Handler
	runtimeAccess *runtimeaccess.Handler
	node          *node.Handler
	deployment    *deployment.Handler
	contextPack   *contextpack.Handler
	auditLog      *auditlog.Handler
}

func (h *Handler) SetUserHandler(handler *user.Handler) {
	h.user = handler
}

func (h *Handler) SetProjectHandler(handler *project.Handler) { h.project = handler }

func (h *Handler) GetProjectAuthoringContext(w http.ResponseWriter, r *http.Request, projectID openapi_types.UUID) {
	h.project.GetProjectAuthoringContext(w, r, projectID.String())
}

func (h *Handler) SetCodeWorkspaceHandler(handler *codeworkspace.Handler) {
	h.codeWorkspace = handler
}

func (h *Handler) SetRuntimeAccessHandler(handler *runtimeaccess.Handler) {
	h.runtimeAccess = handler
}

func (h *Handler) SetNodeHandler(handler *node.Handler) { h.node = handler }

func (h *Handler) SetDeploymentHandler(handler *deployment.Handler) { h.deployment = handler }

func (h *Handler) SetContextPackHandler(handler *contextpack.Handler) { h.contextPack = handler }

func (h *Handler) GetCodeWorkspace(w http.ResponseWriter, r *http.Request, projectID string) {
	h.codeWorkspace.GetCodeWorkspace(w, r, projectID)
}

func (h *Handler) StartCodeWorkspace(w http.ResponseWriter, r *http.Request, projectID string) {
	h.codeWorkspace.StartCodeWorkspace(w, r, projectID)
}

func (h *Handler) StopCodeWorkspace(w http.ResponseWriter, r *http.Request, projectID string) {
	h.codeWorkspace.StopCodeWorkspace(w, r, projectID)
}

func (h *Handler) RebuildCodeWorkspace(w http.ResponseWriter, r *http.Request, projectID string) {
	h.codeWorkspace.RebuildCodeWorkspace(w, r, projectID)
}

func (h *Handler) SetAuditLogHandler(handler *auditlog.Handler) { h.auditLog = handler }

func NewHandler(authHandler *auth.Handler, tenantHandler ...*tenant.Handler) *Handler {
	handler := &Handler{auth: authHandler}
	if len(tenantHandler) > 0 {
		handler.tenant = tenantHandler[0]
	}
	return handler
}

func (h *Handler) ListTenants(w http.ResponseWriter, r *http.Request)  { h.tenant.ListTenants(w, r) }
func (h *Handler) CreateTenant(w http.ResponseWriter, r *http.Request) { h.tenant.CreateTenant(w, r) }
func (h *Handler) GetCurrentTenant(w http.ResponseWriter, r *http.Request) {
	h.tenant.GetCurrentTenant(w, r)
}
func (h *Handler) ListDashboardNotes(w http.ResponseWriter, r *http.Request) {
	h.tenant.ListDashboardNotes(w, r)
}
func (h *Handler) CreateDashboardNote(w http.ResponseWriter, r *http.Request) {
	h.tenant.CreateDashboardNote(w, r)
}
func (h *Handler) DeleteDashboardNote(w http.ResponseWriter, r *http.Request, noteID string) {
	h.tenant.DeleteDashboardNote(w, r, noteID)
}
func (h *Handler) UpdateDashboardNote(w http.ResponseWriter, r *http.Request, noteID string) {
	h.tenant.UpdateDashboardNote(w, r, noteID)
}
func (h *Handler) DeleteTenant(w http.ResponseWriter, r *http.Request, tenantID string) {
	h.tenant.DeleteTenant(w, r, tenantID)
}
func (h *Handler) GetTenant(w http.ResponseWriter, r *http.Request, tenantID string) {
	h.tenant.GetTenant(w, r, tenantID)
}
func (h *Handler) UpdateTenant(w http.ResponseWriter, r *http.Request, tenantID string) {
	h.tenant.UpdateTenant(w, r, tenantID)
}
func (h *Handler) ActivateTenant(w http.ResponseWriter, r *http.Request, tenantID string) {
	h.tenant.ActivateTenant(w, r, tenantID)
}
func (h *Handler) SuspendTenant(w http.ResponseWriter, r *http.Request, tenantID string) {
	h.tenant.SuspendTenant(w, r, tenantID)
}
func (h *Handler) UploadTenantAsset(w http.ResponseWriter, r *http.Request, tenantID string, params platformapi.UploadTenantAssetParams) {
	h.tenant.UploadTenantAsset(w, r, tenantID, params.Type)
}

func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request)  { h.user.ListUsers(w, r) }
func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) { h.user.CreateUser(w, r) }
func (h *Handler) DeleteUser(w http.ResponseWriter, r *http.Request, userID string) {
	h.user.DeleteUser(w, r, userID)
}
func (h *Handler) UpdateUser(w http.ResponseWriter, r *http.Request, userID string) {
	h.user.UpdateUser(w, r, userID)
}
func (h *Handler) ChangeUserPassword(w http.ResponseWriter, r *http.Request, userID string) {
	h.user.ChangeUserPassword(w, r, userID)
}

func (h *Handler) ListProjects(w http.ResponseWriter, r *http.Request) { h.project.ListProjects(w, r) }
func (h *Handler) CreateProject(w http.ResponseWriter, r *http.Request) {
	h.project.CreateProject(w, r)
}
func (h *Handler) ListProjectGroups(w http.ResponseWriter, r *http.Request) {
	h.project.ListProjectGroups(w, r)
}
func (h *Handler) CreateProjectGroup(w http.ResponseWriter, r *http.Request) {
	h.project.CreateProjectGroup(w, r)
}
func (h *Handler) DeleteProjectGroup(w http.ResponseWriter, r *http.Request, groupID string) {
	h.project.DeleteProjectGroup(w, r, groupID)
}
func (h *Handler) UpdateProjectGroup(w http.ResponseWriter, r *http.Request, groupID string) {
	h.project.UpdateProjectGroup(w, r, groupID)
}
func (h *Handler) ImportProject(w http.ResponseWriter, r *http.Request) {
	h.project.ImportProject(w, r)
}
func (h *Handler) ListProjectTags(w http.ResponseWriter, r *http.Request) {
	h.project.ListProjectTags(w, r)
}
func (h *Handler) CreateProjectTag(w http.ResponseWriter, r *http.Request) {
	h.project.CreateProjectTag(w, r)
}
func (h *Handler) DeleteProjectTag(w http.ResponseWriter, r *http.Request, tagID string) {
	h.project.DeleteProjectTag(w, r, tagID)
}
func (h *Handler) UpdateProjectTag(w http.ResponseWriter, r *http.Request, tagID string) {
	h.project.UpdateProjectTag(w, r, tagID)
}
func (h *Handler) DeleteProject(w http.ResponseWriter, r *http.Request, projectID string) {
	h.project.DeleteProject(w, r, projectID)
}
func (h *Handler) UpdateProject(w http.ResponseWriter, r *http.Request, projectID string) {
	h.project.UpdateProject(w, r, projectID)
}
func (h *Handler) GetProjectDeleteImpact(w http.ResponseWriter, r *http.Request, projectID string) {
	h.project.GetProjectDeleteImpact(w, r, projectID)
}
func (h *Handler) ExportProject(w http.ResponseWriter, r *http.Request, projectID string) {
	h.project.ExportProject(w, r, projectID)
}
func (h *Handler) SetProjectGroup(w http.ResponseWriter, r *http.Request, projectID string) {
	h.project.SetProjectGroup(w, r, projectID)
}
func (h *Handler) OperateProject(w http.ResponseWriter, r *http.Request, projectID, operation string) {
	h.project.OperateProject(w, r, projectID, operation)
}
func (h *Handler) ReplaceProjectTags(w http.ResponseWriter, r *http.Request, projectID string) {
	h.project.ReplaceProjectTags(w, r, projectID)
}

func (h *Handler) RefreshWorkspaceContext(w http.ResponseWriter, r *http.Request, projectID string) {
	h.contextPack.RefreshWorkspaceContext(w, r, projectID)
}

func (h *Handler) ListRuntimeRoles(w http.ResponseWriter, r *http.Request, projectID string) {
	h.runtimeAccess.ListRuntimeRoles(w, r, projectID)
}
func (h *Handler) CreateRuntimeRole(w http.ResponseWriter, r *http.Request, projectID string) {
	h.runtimeAccess.CreateRuntimeRole(w, r, projectID)
}
func (h *Handler) UpdateRuntimeRole(w http.ResponseWriter, r *http.Request, projectID, roleID string) {
	h.runtimeAccess.UpdateRuntimeRole(w, r, projectID, roleID)
}
func (h *Handler) DeleteRuntimeRole(w http.ResponseWriter, r *http.Request, projectID, roleID string) {
	h.runtimeAccess.DeleteRuntimeRole(w, r, projectID, roleID)
}
func (h *Handler) ListRuntimeUsers(w http.ResponseWriter, r *http.Request, projectID string) {
	h.runtimeAccess.ListRuntimeUsers(w, r, projectID)
}
func (h *Handler) CreateRuntimeUser(w http.ResponseWriter, r *http.Request, projectID string) {
	h.runtimeAccess.CreateRuntimeUser(w, r, projectID)
}
func (h *Handler) UpdateRuntimeUserStatus(w http.ResponseWriter, r *http.Request, projectID, userID string) {
	h.runtimeAccess.UpdateRuntimeUserStatus(w, r, projectID, userID)
}
func (h *Handler) DeleteRuntimeUser(w http.ResponseWriter, r *http.Request, projectID, userID string) {
	h.runtimeAccess.DeleteRuntimeUser(w, r, projectID, userID)
}
func (h *Handler) ReplaceRuntimeUserRoles(w http.ResponseWriter, r *http.Request, projectID, userID string) {
	h.runtimeAccess.ReplaceRuntimeUserRoles(w, r, projectID, userID)
}
func (h *Handler) ResetRuntimeUserPassword(w http.ResponseWriter, r *http.Request, projectID, userID string) {
	h.runtimeAccess.ResetRuntimeUserPassword(w, r, projectID, userID)
}

func (h *Handler) ListNodes(w http.ResponseWriter, r *http.Request) { h.node.ListNodes(w, r) }
func (h *Handler) ApproveNode(w http.ResponseWriter, r *http.Request, nodeID string) {
	h.node.ApproveNode(w, r, nodeID)
}
func (h *Handler) RejectNode(w http.ResponseWriter, r *http.Request, nodeID string) {
	h.node.RejectNode(w, r, nodeID)
}
func (h *Handler) DeleteNode(w http.ResponseWriter, r *http.Request, nodeID string) {
	h.node.DeleteNode(w, r, nodeID)
}

func (h *Handler) ListProjectVersions(w http.ResponseWriter, r *http.Request, projectID string) {
	h.deployment.ListProjectVersions(w, r, projectID)
}
func (h *Handler) PublishProjectVersion(w http.ResponseWriter, r *http.Request, projectID string) {
	h.deployment.PublishProjectVersion(w, r, projectID)
}
func (h *Handler) DeletePublishedVersion(w http.ResponseWriter, r *http.Request, versionID string) {
	h.deployment.DeletePublishedVersion(w, r, versionID)
}
func (h *Handler) RestoreVersionDevelopment(w http.ResponseWriter, r *http.Request, versionID openapi_types.UUID) {
	h.deployment.RestoreVersionDevelopment(w, r, versionID.String())
}
func (h *Handler) GetRestoreDevelopmentTask(w http.ResponseWriter, r *http.Request, taskID openapi_types.UUID) {
	h.deployment.GetRestoreDevelopmentTask(w, r, taskID.String())
}
func (h *Handler) ListProjectDeploymentNodes(w http.ResponseWriter, r *http.Request, projectID string) {
	h.deployment.ListProjectDeploymentNodes(w, r, projectID)
}
func (h *Handler) DeployPublishedVersion(w http.ResponseWriter, r *http.Request, versionID string) {
	h.deployment.DeployPublishedVersion(w, r, versionID)
}
func (h *Handler) DeployProjectDevelopment(w http.ResponseWriter, r *http.Request, projectID string) {
	h.deployment.DeployProjectDevelopment(w, r, projectID)
}
func (h *Handler) StartNodeDeployment(w http.ResponseWriter, r *http.Request, deploymentID string) {
	h.deployment.StartNodeDeployment(w, r, deploymentID)
}
func (h *Handler) StopNodeDeployment(w http.ResponseWriter, r *http.Request, deploymentID string) {
	h.deployment.StopNodeDeployment(w, r, deploymentID)
}
func (h *Handler) RestartNodeDeployment(w http.ResponseWriter, r *http.Request, deploymentID string) {
	h.deployment.RestartNodeDeployment(w, r, deploymentID)
}
func (h *Handler) DeleteNodeDeployment(w http.ResponseWriter, r *http.Request, deploymentID string) {
	h.deployment.DeleteNodeDeployment(w, r, deploymentID)
}
func (h *Handler) RollbackDeployment(w http.ResponseWriter, r *http.Request, versionID string) {
	h.deployment.RollbackDeployment(w, r, versionID)
}

func (h *Handler) ListAuditLogs(w http.ResponseWriter, r *http.Request) {
	h.auditLog.ListAuditLogs(w, r)
}
func (h *Handler) GetAuditLog(w http.ResponseWriter, r *http.Request, logID string) {
	h.auditLog.GetAuditLog(w, r, logID)
}
func (h *Handler) DeleteAuditLogs(w http.ResponseWriter, r *http.Request) {
	h.auditLog.DeleteAuditLogs(w, r)
}
func (h *Handler) ExportAuditLogs(w http.ResponseWriter, r *http.Request) {
	h.auditLog.ExportAuditLogs(w, r)
}
func (h *Handler) GetAuditLogStats(w http.ResponseWriter, r *http.Request) {
	h.auditLog.GetAuditLogStats(w, r)
}
func (h *Handler) ListRecentActivities(w http.ResponseWriter, r *http.Request) {
	h.auditLog.ListRecentActivities(w, r)
}

func (h *Handler) GetAuthCaptcha(w http.ResponseWriter, r *http.Request, params platformapi.GetAuthCaptchaParams) {
	h.auth.GetAuthCaptcha(w, r, params)
}

func (h *Handler) GetAuthConfig(w http.ResponseWriter, r *http.Request) {
	h.auth.GetAuthConfig(w, r)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	h.auth.Login(w, r)
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	h.auth.Logout(w, r)
}

func (h *Handler) GetCurrentAuthUser(w http.ResponseWriter, r *http.Request) {
	h.auth.GetCurrentAuthUser(w, r)
}

func (h *Handler) ChangeAuthPassword(w http.ResponseWriter, r *http.Request) {
	h.auth.ChangeAuthPassword(w, r)
}

func (h *Handler) RefreshAuthToken(w http.ResponseWriter, r *http.Request) {
	h.auth.RefreshAuthToken(w, r)
}

func (h *Handler) ListAuthTenants(w http.ResponseWriter, r *http.Request, params platformapi.ListAuthTenantsParams) {
	h.auth.ListAuthTenants(w, r, params)
}
func (h *Handler) InitializeTenant(w http.ResponseWriter, r *http.Request, tenantID string) {
	h.tenant.InitializeTenant(w, r, tenantID)
}
func (h *Handler) ResetTenantAdminPassword(w http.ResponseWriter, r *http.Request, tenantID string) {
	h.tenant.ResetTenantAdminPassword(w, r, tenantID)
}

func (h *Handler) UpdateCurrentAuthUser(w http.ResponseWriter, r *http.Request) {
	h.auth.UpdateCurrentAuthUser(w, r)
}

func (h *Handler) UploadCurrentUserAvatar(w http.ResponseWriter, r *http.Request) {
	h.auth.UploadCurrentUserAvatar(w, r)
}
func (h *Handler) GetCurrentUserAvatar(w http.ResponseWriter, r *http.Request) {
	h.auth.GetCurrentUserAvatar(w, r)
}

func (h *Handler) DeleteCurrentUserAvatar(w http.ResponseWriter, r *http.Request) {
	h.auth.DeleteCurrentUserAvatar(w, r)
}
