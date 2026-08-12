package tenant

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

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

func (h *Handler) ListTenants(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireSuperAdmin(w, r); !ok {
		return
	}
	page, limit := pagination(r)
	items, total, err := h.service.List(r.Context(), ListFilter{Keyword: strings.TrimSpace(r.URL.Query().Get("keyword")), Status: strings.TrimSpace(r.URL.Query().Get("status")), Page: page, Limit: limit})
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	list := make([]map[string]any, 0, len(items))
	for _, item := range items {
		list = append(list, tenantPayload(item))
	}
	platformapi.WriteSuccess(w, r, map[string]any{"list": list, "pagination": paginationPayload(page, limit, total)})
}

func (h *Handler) GetTenant(w http.ResponseWriter, r *http.Request, tenantID string) {
	if _, ok := h.requireSuperAdmin(w, r); !ok {
		return
	}
	item, err := h.service.Get(r.Context(), tenantID)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	platformapi.WriteSuccess(w, r, tenantPayload(item))
}

func (h *Handler) GetCurrentTenant(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	item, err := h.service.Get(r.Context(), user.TenantID)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	platformapi.WriteSuccess(w, r, tenantPayload(item))
}

func (h *Handler) CreateTenant(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireSuperAdmin(w, r); !ok {
		return
	}
	input, err := decodeTenantInput(r)
	if err != nil {
		h.writeInvalid(w, r, err)
		return
	}
	item, err := h.service.Create(r.Context(), input)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	platformapi.WriteSuccess(w, r, tenantPayload(item))
}

func (h *Handler) UpdateTenant(w http.ResponseWriter, r *http.Request, tenantID string) {
	if _, ok := h.requireSuperAdmin(w, r); !ok {
		return
	}
	input, err := decodeTenantInput(r)
	if err != nil {
		h.writeInvalid(w, r, err)
		return
	}
	item, err := h.service.Update(r.Context(), tenantID, input)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	platformapi.WriteSuccess(w, r, tenantPayload(item))
}

func (h *Handler) DeleteTenant(w http.ResponseWriter, r *http.Request, tenantID string) {
	user, ok := h.requireSuperAdmin(w, r)
	if !ok {
		return
	}
	item, err := h.service.Get(r.Context(), tenantID)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	if item.ID == user.TenantID {
		h.writeInvalid(w, r, errors.New("不能删除当前登录租户"))
		return
	}
	if err := h.service.Delete(r.Context(), tenantID); err != nil {
		h.writeError(w, r, err)
		return
	}
	platformapi.WriteSuccess(w, r, map[string]any{})
}

func (h *Handler) ActivateTenant(w http.ResponseWriter, r *http.Request, tenantID string) {
	h.setStatus(w, r, tenantID, "active")
}

func (h *Handler) SuspendTenant(w http.ResponseWriter, r *http.Request, tenantID string) {
	h.setStatus(w, r, tenantID, "suspended")
}

func (h *Handler) ListDashboardNotes(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	items, err := h.service.ListNotes(r.Context(), user.TenantID)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	list := make([]map[string]any, 0, len(items))
	for _, item := range items {
		list = append(list, notePayload(item))
	}
	platformapi.WriteSuccess(w, r, list)
}

func (h *Handler) CreateDashboardNote(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	content, err := decodeNote(r)
	if err != nil {
		h.writeInvalid(w, r, err)
		return
	}
	item, err := h.service.CreateNote(r.Context(), user.TenantID, user.ID, content)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	platformapi.WriteSuccess(w, r, notePayload(item))
}

func (h *Handler) UpdateDashboardNote(w http.ResponseWriter, r *http.Request, noteID string) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	content, err := decodeNote(r)
	if err != nil {
		h.writeInvalid(w, r, err)
		return
	}
	item, err := h.service.UpdateNote(r.Context(), user.TenantID, user.ID, noteID, content)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	platformapi.WriteSuccess(w, r, notePayload(item))
}

func (h *Handler) DeleteDashboardNote(w http.ResponseWriter, r *http.Request, noteID string) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	if err := h.service.DeleteNote(r.Context(), user.TenantID, noteID); err != nil {
		h.writeError(w, r, err)
		return
	}
	platformapi.WriteSuccess(w, r, map[string]any{})
}

func (h *Handler) UploadTenantAsset(w http.ResponseWriter, r *http.Request, tenantID string, assetType string) {
	if _, ok := h.requireSuperAdmin(w, r); !ok {
		return
	}
	item, err := h.service.Get(r.Context(), tenantID)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16<<20)
	if err := r.ParseMultipartForm(16 << 20); err != nil {
		h.writeInvalid(w, r, errors.New("上传文件不能超过 16MB"))
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		h.writeInvalid(w, r, errors.New("缺少上传文件"))
		return
	}
	defer file.Close()
	head := make([]byte, 512)
	read, readErr := io.ReadFull(file, head)
	if readErr != nil && !errors.Is(readErr, io.EOF) && !errors.Is(readErr, io.ErrUnexpectedEOF) {
		h.writeInvalid(w, r, errors.New("读取上传文件失败"))
		return
	}
	head = head[:read]
	contentType := http.DetectContentType(head)
	if strings.EqualFold(filepath.Ext(header.Filename), ".svg") && bytes.HasPrefix(bytes.TrimSpace(head), []byte("<svg")) {
		contentType = "image/svg+xml"
	}
	asset, err := h.service.Upload(r.Context(), item.ID, assetType, header.Filename, contentType, io.MultiReader(bytes.NewReader(head), file), header.Size)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	input := Input{}
	if assetType == "logo" {
		input.LogoObjectKey = &asset.ObjectKey
	} else {
		input.LoginBackgroundObjectKey = &asset.ObjectKey
	}
	if _, err := h.service.Update(r.Context(), item.ID, input); err != nil {
		h.writeError(w, r, err)
		return
	}
	platformapi.WriteSuccess(w, r, map[string]any{"fileUrl": asset.URL})
}

func (h *Handler) setStatus(w http.ResponseWriter, r *http.Request, tenantID, status string) {
	if _, ok := h.requireSuperAdmin(w, r); !ok {
		return
	}
	item, err := h.service.SetStatus(r.Context(), tenantID, status)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	platformapi.WriteSuccess(w, r, tenantPayload(item))
}

func (h *Handler) requireUser(w http.ResponseWriter, r *http.Request) (auth.User, bool) {
	if user, ok := auth.UserFromContext(r.Context()); ok {
		return user, true
	}
	parts := strings.Fields(r.Header.Get("Authorization"))
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		platformapi.WriteError(w, r, http.StatusUnauthorized, platformapi.ErrorCodeTokenRequired, "缺少访问令牌")
		return auth.User{}, false
	}
	user, err := h.authService.Authenticate(r.Context(), parts[1])
	if err != nil {
		platformapi.WriteError(w, r, http.StatusUnauthorized, platformapi.ErrorCodeTokenInvalid, "访问令牌无效或已过期")
		return auth.User{}, false
	}
	return user, true
}

func (h *Handler) requireSuperAdmin(w http.ResponseWriter, r *http.Request) (auth.User, bool) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return auth.User{}, false
	}
	if user.Role != "SUPER_ADMIN" {
		platformapi.WriteError(w, r, http.StatusOK, platformapi.ErrorCodePermissionDenied, "无权执行当前操作")
		return auth.User{}, false
	}
	return user, true
}

func (h *Handler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		platformapi.WriteError(w, r, http.StatusOK, platformapi.ErrorCodeTenantNotFound, err.Error())
	case errors.Is(err, ErrAlreadyExists):
		platformapi.WriteError(w, r, http.StatusOK, platformapi.ErrorCodeTenantCodeExists, err.Error())
	case errors.Is(err, ErrNoteNotFound):
		platformapi.WriteError(w, r, http.StatusOK, platformapi.ErrorCodeNotFound, err.Error())
	default:
		if strings.Contains(err.Error(), "不能为空") || strings.Contains(err.Error(), "不能超过") || strings.Contains(err.Error(), "仅支持") {
			h.writeInvalid(w, r, err)
			return
		}
		platformapi.WriteError(w, r, http.StatusInternalServerError, platformapi.ErrorCodeInternal, "系统内部错误")
	}
}

func (h *Handler) writeInvalid(w http.ResponseWriter, r *http.Request, err error) {
	platformapi.WriteError(w, r, http.StatusOK, platformapi.ErrorCodeInvalidRequest, err.Error())
}

func decodeTenantInput(r *http.Request) (Input, error) {
	var body struct {
		Name           *string        `json:"name"`
		Code           *string        `json:"code"`
		Description    *string        `json:"description"`
		Status         *string        `json:"status"`
		ContactEmail   *string        `json:"contactEmail"`
		ContactPhone   *string        `json:"contactPhone"`
		MaxUsers       *int32         `json:"maxUsers"`
		MaxProjects    *int32         `json:"maxProjects"`
		MaxStorage     *int64         `json:"maxStorage"`
		CompanyName    *string        `json:"companyName"`
		CompanyAddress *string        `json:"companyAddress"`
		CompanyPhone   *string        `json:"companyPhone"`
		CompanyWebsite *string        `json:"companyWebsite"`
		Settings       map[string]any `json:"settings"`
		ExpiresAt      *string        `json:"expiresAt"`
	}
	if err := decodeJSON(r, &body); err != nil {
		return Input{}, err
	}
	input := Input{Name: body.Name, Code: body.Code, Description: body.Description, Status: body.Status, ContactEmail: body.ContactEmail, ContactPhone: body.ContactPhone, MaxUsers: body.MaxUsers, MaxProjects: body.MaxProjects, MaxStorage: body.MaxStorage, CompanyName: body.CompanyName, CompanyAddress: body.CompanyAddress, CompanyPhone: body.CompanyPhone, CompanyWebsite: body.CompanyWebsite, Settings: body.Settings}
	if body.ExpiresAt != nil && strings.TrimSpace(*body.ExpiresAt) != "" {
		parsed, err := time.Parse(time.RFC3339, *body.ExpiresAt)
		if err != nil {
			return Input{}, fmt.Errorf("expiresAt 格式错误")
		}
		input.ExpiresAt = &parsed
	}
	return input, nil
}

func decodeNote(r *http.Request) (string, error) {
	var body struct {
		Content string `json:"content"`
	}
	if err := decodeJSON(r, &body); err != nil {
		return "", err
	}
	return body.Content, nil
}

func decodeJSON(r *http.Request, target any) error {
	decoder := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	if err := decoder.Decode(target); err != nil {
		return err
	}
	return nil
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

func tenantPayload(item Tenant) map[string]any {
	payload := map[string]any{"id": item.ID, "name": item.Name, "code": item.Code, "description": item.Description, "status": item.Status, "contactEmail": item.ContactEmail, "contactPhone": item.ContactPhone, "maxUsers": item.MaxUsers, "maxProjects": item.MaxProjects, "maxStorage": item.MaxStorage, "usedStorage": item.UsedStorage, "logoUrl": item.LogoURL, "loginBackgroundUrl": item.LoginBackgroundURL, "companyName": item.CompanyName, "companyAddress": item.CompanyAddress, "companyPhone": item.CompanyPhone, "companyWebsite": item.CompanyWebsite, "settings": item.Settings, "userCount": item.UserCount, "projectCount": item.ProjectCount, "createdAt": item.CreatedAt.Format(timeFormat), "updatedAt": item.UpdatedAt.Format(timeFormat)}
	if item.ExpiresAt != nil {
		payload["expiresAt"] = item.ExpiresAt.Format(timeFormat)
	} else {
		payload["expiresAt"] = nil
	}
	return payload
}

func notePayload(item Note) map[string]any {
	return map[string]any{"id": item.ID, "tenantId": item.TenantID, "content": item.Content, "createdBy": item.CreatedBy, "updatedBy": item.UpdatedBy, "createdByUsername": item.CreatedByUsername, "updatedByUsername": item.UpdatedByUsername, "createdAt": item.CreatedAt.Format(timeFormat), "updatedAt": item.UpdatedAt.Format(timeFormat)}
}
