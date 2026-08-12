package user

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
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

func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.requireManager(w, r)
	if !ok {
		return
	}
	page, limit := pagination(r)
	keyword := strings.TrimSpace(r.URL.Query().Get("username"))
	if keyword == "" {
		keyword = strings.TrimSpace(r.URL.Query().Get("keyword"))
	}
	items, total, err := h.service.List(r.Context(), actor.TenantID, ListFilter{Keyword: keyword, Role: strings.TrimSpace(r.URL.Query().Get("role")), Status: strings.TrimSpace(r.URL.Query().Get("status")), Page: page, Limit: limit})
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	list := make([]map[string]any, 0, len(items))
	for _, item := range items {
		list = append(list, userPayload(item))
	}
	platformapi.WriteSuccess(w, r, map[string]any{"list": list, "pagination": paginationPayload(page, limit, total)})
}

func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.requireManager(w, r)
	if !ok {
		return
	}
	input, err := decodeInput(r)
	if err != nil {
		h.writeInvalid(w, r, err)
		return
	}
	item, err := h.service.Create(r.Context(), actor, input)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	platformapi.WriteSuccess(w, r, userPayload(item))
}

func (h *Handler) UpdateUser(w http.ResponseWriter, r *http.Request, userID string) {
	actor, ok := h.requireManager(w, r)
	if !ok {
		return
	}
	input, err := decodeInput(r)
	if err != nil {
		h.writeInvalid(w, r, err)
		return
	}
	item, err := h.service.Update(r.Context(), actor, userID, input)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	platformapi.WriteSuccess(w, r, userPayload(item))
}

func (h *Handler) ChangeUserPassword(w http.ResponseWriter, r *http.Request, userID string) {
	actor, ok := h.requireManager(w, r)
	if !ok {
		return
	}
	var body struct {
		NewPassword string `json:"newPassword"`
	}
	if err := decodeJSON(r, &body); err != nil {
		h.writeInvalid(w, r, err)
		return
	}
	if err := h.service.UpdatePassword(r.Context(), actor, userID, body.NewPassword); err != nil {
		h.writeError(w, r, err)
		return
	}
	platformapi.WriteSuccess(w, r, map[string]any{})
}

func (h *Handler) DeleteUser(w http.ResponseWriter, r *http.Request, userID string) {
	actor, ok := h.requireManager(w, r)
	if !ok {
		return
	}
	if err := h.service.Delete(r.Context(), actor, userID); err != nil {
		h.writeError(w, r, err)
		return
	}
	platformapi.WriteSuccess(w, r, map[string]any{})
}

func (h *Handler) requireManager(w http.ResponseWriter, r *http.Request) (auth.User, bool) {
	actor, ok := auth.UserFromContext(r.Context())
	if !ok {
		parts := strings.Fields(r.Header.Get("Authorization"))
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			platformapi.WriteError(w, r, http.StatusUnauthorized, platformapi.ErrorCodeTokenRequired, "缺少访问令牌")
			return auth.User{}, false
		}
		var err error
		actor, err = h.authService.Authenticate(r.Context(), parts[1])
		if err != nil {
			platformapi.WriteError(w, r, http.StatusUnauthorized, platformapi.ErrorCodeTokenInvalid, "访问令牌无效或已过期")
			return auth.User{}, false
		}
	}
	if err := auth.RequireCapability(actor, auth.CapabilityUserManage); err != nil {
		platformapi.WriteError(w, r, http.StatusOK, platformapi.ErrorCodePermissionDenied, "无权执行当前操作")
		return auth.User{}, false
	}
	return actor, true
}

func (h *Handler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, auth.ErrPermissionDenied):
		platformapi.WriteError(w, r, http.StatusOK, platformapi.ErrorCodePermissionDenied, err.Error())
	case errors.Is(err, ErrNotFound):
		platformapi.WriteError(w, r, http.StatusOK, platformapi.ErrorCodeUserNotFound, err.Error())
	case errors.Is(err, ErrAlreadyExists):
		platformapi.WriteError(w, r, http.StatusOK, platformapi.ErrorCodeUsernameExists, err.Error())
	case errors.Is(err, ErrDeleteSelf), errors.Is(err, ErrModifySelf):
		platformapi.WriteError(w, r, http.StatusOK, platformapi.ErrorCodeInvalidRequest, err.Error())
	default:
		if strings.Contains(err.Error(), "密码") || strings.Contains(err.Error(), "用户名") {
			h.writeInvalid(w, r, err)
			return
		}
		platformapi.WriteError(w, r, http.StatusInternalServerError, platformapi.ErrorCodeInternal, "系统内部错误")
	}
}

func (h *Handler) writeInvalid(w http.ResponseWriter, r *http.Request, err error) {
	platformapi.WriteError(w, r, http.StatusOK, platformapi.ErrorCodeInvalidRequest, err.Error())
}

func decodeInput(r *http.Request) (Input, error) {
	var body struct {
		TenantID    *string        `json:"tenantId"`
		Username    *string        `json:"username"`
		Password    *string        `json:"password"`
		Email       *string        `json:"email"`
		Phone       *string        `json:"phone"`
		FullName    *string        `json:"fullName"`
		Avatar      *string        `json:"avatar"`
		Role        *string        `json:"role"`
		Status      *string        `json:"status"`
		Preferences map[string]any `json:"preferences"`
	}
	if err := decodeJSON(r, &body); err != nil {
		return Input{}, err
	}
	return Input{TenantID: body.TenantID, Username: body.Username, Password: body.Password, Email: body.Email, Phone: body.Phone, FullName: body.FullName, Avatar: body.Avatar, Role: body.Role, Status: body.Status, Preferences: body.Preferences}, nil
}

func decodeJSON(r *http.Request, target any) error {
	return json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(target)
}
func pagination(r *http.Request) (int, int) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	return normalizePage(page, limit)
}
func paginationPayload(page, limit int, total int64) map[string]any {
	pages := int64(0)
	if total > 0 {
		pages = (total + int64(limit) - 1) / int64(limit)
	}
	return map[string]any{"page": page, "limit": limit, "total": total, "pages": pages}
}
func userPayload(item User) map[string]any {
	payload := map[string]any{"id": item.ID, "tenantId": item.TenantID, "username": item.Username, "email": item.Email, "phone": item.Phone, "fullName": item.FullName, "avatar": item.Avatar, "role": item.Role, "status": item.Status, "preferences": item.Preferences, "lastLoginIp": item.LastLoginIP, "createdAt": item.CreatedAt.Format(timeFormat), "updatedAt": item.UpdatedAt.Format(timeFormat)}
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
