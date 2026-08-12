package runtimeaccess

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/indu-forge/dev_core/internal/auth"
	platformapi "github.com/indu-forge/dev_core/internal/platform/api"
)

const timeFormat = "2006-01-02 15:04:05"

type Handler struct {
	service     *Service
	authService *auth.Service
}

func NewHandler(service *Service, authService *auth.Service) *Handler {
	return &Handler{service: service, authService: authService}
}
func (h *Handler) ListRuntimeRoles(w http.ResponseWriter, r *http.Request, projectID string) {
	actor, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	items, err := h.service.ListRoles(r.Context(), actor, projectID)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	list := make([]map[string]any, 0, len(items))
	for _, item := range items {
		list = append(list, rolePayload(item))
	}
	platformapi.WriteSuccess(w, r, list)
}
func (h *Handler) CreateRuntimeRole(w http.ResponseWriter, r *http.Request, projectID string) {
	actor, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	var input RoleInput
	if err := decodeJSON(r, &input); err != nil {
		h.invalid(w, r, err)
		return
	}
	item, err := h.service.CreateRole(r.Context(), actor, projectID, input)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	platformapi.WriteSuccess(w, r, rolePayload(item))
}
func (h *Handler) UpdateRuntimeRole(w http.ResponseWriter, r *http.Request, projectID, roleID string) {
	actor, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	var input RoleInput
	if err := decodeJSON(r, &input); err != nil {
		h.invalid(w, r, err)
		return
	}
	item, err := h.service.UpdateRole(r.Context(), actor, projectID, roleID, input)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	platformapi.WriteSuccess(w, r, rolePayload(item))
}
func (h *Handler) DeleteRuntimeRole(w http.ResponseWriter, r *http.Request, projectID, roleID string) {
	actor, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	if err := h.service.DeleteRole(r.Context(), actor, projectID, roleID); err != nil {
		h.writeError(w, r, err)
		return
	}
	platformapi.WriteSuccess(w, r, map[string]any{})
}
func (h *Handler) ListRuntimeUsers(w http.ResponseWriter, r *http.Request, projectID string) {
	actor, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	items, err := h.service.ListUsers(r.Context(), actor, projectID)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	list := make([]map[string]any, 0, len(items))
	for _, item := range items {
		list = append(list, userPayload(item))
	}
	platformapi.WriteSuccess(w, r, list)
}
func (h *Handler) CreateRuntimeUser(w http.ResponseWriter, r *http.Request, projectID string) {
	actor, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	var body struct {
		Username        string   `json:"username"`
		InitialPassword string   `json:"initialPassword"`
		Password        string   `json:"password"`
		DisplayName     string   `json:"displayName"`
		Email           string   `json:"email"`
		Status          string   `json:"status"`
		RoleIDs         []string `json:"roleIds"`
	}
	if err := decodeJSON(r, &body); err != nil {
		h.invalid(w, r, err)
		return
	}
	password := body.InitialPassword
	if password == "" {
		password = body.Password
	}
	item, err := h.service.CreateUser(r.Context(), actor, projectID, UserInput{Username: body.Username, Password: password, DisplayName: body.DisplayName, Email: body.Email, Status: body.Status, RoleIDs: body.RoleIDs})
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	platformapi.WriteSuccess(w, r, userPayload(item))
}
func (h *Handler) UpdateRuntimeUserStatus(w http.ResponseWriter, r *http.Request, projectID, userID string) {
	actor, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	var body struct {
		Status string `json:"status"`
	}
	if err := decodeJSON(r, &body); err != nil {
		h.invalid(w, r, err)
		return
	}
	item, err := h.service.UpdateUserStatus(r.Context(), actor, projectID, userID, body.Status)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	platformapi.WriteSuccess(w, r, userPayload(item))
}
func (h *Handler) DeleteRuntimeUser(w http.ResponseWriter, r *http.Request, projectID, userID string) {
	actor, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	if err := h.service.DeleteUser(r.Context(), actor, projectID, userID); err != nil {
		h.writeError(w, r, err)
		return
	}
	platformapi.WriteSuccess(w, r, map[string]any{})
}
func (h *Handler) ReplaceRuntimeUserRoles(w http.ResponseWriter, r *http.Request, projectID, userID string) {
	actor, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	var body struct {
		RoleIDs []string `json:"roleIds"`
	}
	if err := decodeJSON(r, &body); err != nil {
		h.invalid(w, r, err)
		return
	}
	if err := h.service.ReplaceUserRoles(r.Context(), actor, projectID, userID, body.RoleIDs); err != nil {
		h.writeError(w, r, err)
		return
	}
	platformapi.WriteSuccess(w, r, map[string]any{"roleIds": body.RoleIDs})
}
func (h *Handler) ResetRuntimeUserPassword(w http.ResponseWriter, r *http.Request, projectID, userID string) {
	actor, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	var body struct {
		NewPassword string `json:"newPassword"`
	}
	if err := decodeJSON(r, &body); err != nil {
		h.invalid(w, r, err)
		return
	}
	if err := h.service.ResetPassword(r.Context(), actor, projectID, userID, body.NewPassword); err != nil {
		h.writeError(w, r, err)
		return
	}
	platformapi.WriteSuccess(w, r, map[string]any{})
}
func (h *Handler) requireUser(w http.ResponseWriter, r *http.Request) (auth.User, bool) {
	if actor, ok := auth.UserFromContext(r.Context()); ok {
		return actor, true
	}
	parts := strings.Fields(r.Header.Get("Authorization"))
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		platformapi.WriteError(w, r, http.StatusUnauthorized, platformapi.ErrorCodeTokenRequired, "缺少访问令牌")
		return auth.User{}, false
	}
	actor, err := h.authService.Authenticate(r.Context(), parts[1])
	if err != nil {
		platformapi.WriteError(w, r, http.StatusUnauthorized, platformapi.ErrorCodeTokenInvalid, "访问令牌无效或已过期")
		return auth.User{}, false
	}
	return actor, true
}
func (h *Handler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, auth.ErrPermissionDenied):
		platformapi.WriteError(w, r, http.StatusOK, platformapi.ErrorCodePermissionDenied, err.Error())
	case errors.Is(err, ErrUserNotFound), errors.Is(err, ErrRoleNotFound), errors.Is(err, ErrProjectNotFound):
		platformapi.WriteError(w, r, http.StatusOK, platformapi.ErrorCodeNotFound, err.Error())
	case errors.Is(err, ErrAlreadyExists):
		platformapi.WriteError(w, r, http.StatusOK, platformapi.ErrorCodeAlreadyExists, err.Error())
	case errors.Is(err, ErrBuiltin):
		platformapi.WriteError(w, r, http.StatusOK, platformapi.ErrorCodeInvalidRequest, err.Error())
	default:
		if strings.Contains(err.Error(), "密码") || strings.Contains(err.Error(), "不能为空") || strings.Contains(err.Error(), "状态无效") {
			h.invalid(w, r, err)
			return
		}
		platformapi.WriteError(w, r, http.StatusInternalServerError, platformapi.ErrorCodeInternal, "系统内部错误")
	}
}
func (h *Handler) invalid(w http.ResponseWriter, r *http.Request, err error) {
	platformapi.WriteError(w, r, http.StatusOK, platformapi.ErrorCodeInvalidRequest, err.Error())
}
func decodeJSON(r *http.Request, target any) error {
	return json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(target)
}
func rolePayload(item Role) map[string]any {
	return map[string]any{"id": item.ID, "projectId": item.ProjectID, "code": item.Code, "name": item.Name, "description": item.Description, "status": item.Status, "isBuiltin": item.IsBuiltin, "isSystem": item.IsBuiltin, "capabilities": item.Capabilities, "permissions": item.Capabilities, "userCount": item.UserCount, "createdAt": item.CreatedAt.Format(timeFormat), "updatedAt": item.UpdatedAt.Format(timeFormat)}
}
func userPayload(item RuntimeUser) map[string]any {
	roles := make([]map[string]any, 0, len(item.Roles))
	roleIDs := make([]string, 0, len(item.Roles))
	for _, role := range item.Roles {
		roles = append(roles, rolePayload(role))
		roleIDs = append(roleIDs, role.ID)
	}
	payload := map[string]any{"id": item.ID, "projectId": item.ProjectID, "username": item.Username, "displayName": item.DisplayName, "email": item.Email, "status": item.Status, "isBuiltinAdmin": item.IsBuiltinAdmin, "isDefaultAdmin": item.IsBuiltinAdmin, "roles": roles, "roleIds": roleIDs, "createdAt": item.CreatedAt.Format(timeFormat), "updatedAt": item.UpdatedAt.Format(timeFormat)}
	if item.LastLoginAt != nil {
		payload["lastLoginAt"] = item.LastLoginAt.Format(timeFormat)
	} else {
		payload["lastLoginAt"] = nil
	}
	if item.PasswordChangedAt != nil {
		payload["passwordChangedAt"] = item.PasswordChangedAt.Format(timeFormat)
	} else {
		payload["passwordChangedAt"] = nil
	}
	return payload
}
