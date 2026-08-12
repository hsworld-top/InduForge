package project

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

func (h *Handler) ListProjects(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	page, limit := pagination(r)
	items, total, err := h.service.List(r.Context(), actor, ListFilter{Keyword: firstQuery(r, "name", "keyword"), Status: r.URL.Query().Get("status"), Visibility: firstQuery(r, "visibility"), GroupID: firstQuery(r, "groupId", "group"), TagID: firstQuery(r, "tagId", "tag"), Page: page, Limit: limit})
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	list := make([]map[string]any, 0, len(items))
	for _, item := range items {
		list = append(list, projectResponse(item))
	}
	platformapi.WriteSuccess(w, r, map[string]any{"list": list, "pagination": paginationPayload(page, limit, total)})
}
func (h *Handler) CreateProject(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	input, err := decodeProjectInput(r)
	if err != nil {
		h.writeInvalid(w, r, err)
		return
	}
	item, err := h.service.Create(r.Context(), actor, input)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	platformapi.WriteSuccess(w, r, projectResponse(item))
}
func (h *Handler) UpdateProject(w http.ResponseWriter, r *http.Request, projectID string) {
	actor, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	input, err := decodeProjectInput(r)
	if err != nil {
		h.writeInvalid(w, r, err)
		return
	}
	item, err := h.service.Update(r.Context(), actor, projectID, input)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	platformapi.WriteSuccess(w, r, projectResponse(item))
}
func (h *Handler) DeleteProject(w http.ResponseWriter, r *http.Request, projectID string) {
	actor, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	if err := h.service.Delete(r.Context(), actor, projectID); err != nil {
		h.writeError(w, r, err)
		return
	}
	platformapi.WriteSuccess(w, r, map[string]any{})
}
func (h *Handler) GetProjectDeleteImpact(w http.ResponseWriter, r *http.Request, projectID string) {
	actor, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	impact, err := h.service.DeleteImpact(r.Context(), actor, projectID)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	platformapi.WriteSuccess(w, r, map[string]any{"runtimeUserCount": impact.RuntimeUserCount, "runtimeRoleCount": impact.RuntimeRoleCount, "deploymentCount": impact.DeploymentCount, "versionCount": impact.VersionCount, "hasImpact": impact.RuntimeUserCount+impact.RuntimeRoleCount+impact.DeploymentCount+impact.VersionCount > 0})
}
func (h *Handler) OperateProject(w http.ResponseWriter, r *http.Request, projectID, operation string) {
	actor, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	item, err := h.service.Operate(r.Context(), actor, projectID, operation)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	platformapi.WriteSuccess(w, r, projectResponse(item))
}
func (h *Handler) ExportProject(w http.ResponseWriter, r *http.Request, projectID string) {
	actor, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	payload, err := h.service.Export(r.Context(), actor, projectID)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment; filename=project-"+projectID+".json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(raw)
}
func (h *Handler) ImportProject(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	var body struct {
		Name    string         `json:"name"`
		Payload map[string]any `json:"payload"`
	}
	if err := decodeJSON(r, &body); err != nil {
		h.writeInvalid(w, r, err)
		return
	}
	item, err := h.service.Import(r.Context(), actor, body.Name, body.Payload)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	platformapi.WriteSuccess(w, r, projectResponse(item))
}

func (h *Handler) ListProjectTags(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	items, err := h.service.ListTags(r.Context(), actor, r.URL.Query().Get("keyword"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	list := make([]map[string]any, 0, len(items))
	for _, item := range items {
		list = append(list, tagResponse(item))
	}
	platformapi.WriteSuccess(w, r, list)
}
func (h *Handler) CreateProjectTag(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	input, err := decodeTagInput(r)
	if err != nil {
		h.writeInvalid(w, r, err)
		return
	}
	item, err := h.service.CreateTag(r.Context(), actor, input)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	platformapi.WriteSuccess(w, r, tagResponse(item))
}
func (h *Handler) UpdateProjectTag(w http.ResponseWriter, r *http.Request, tagID string) {
	actor, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	input, err := decodeTagInput(r)
	if err != nil {
		h.writeInvalid(w, r, err)
		return
	}
	item, err := h.service.UpdateTag(r.Context(), actor, tagID, input)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	platformapi.WriteSuccess(w, r, tagResponse(item))
}
func (h *Handler) DeleteProjectTag(w http.ResponseWriter, r *http.Request, tagID string) {
	actor, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	if err := h.service.DeleteTag(r.Context(), actor, tagID); err != nil {
		h.writeError(w, r, err)
		return
	}
	platformapi.WriteSuccess(w, r, map[string]any{})
}
func (h *Handler) ReplaceProjectTags(w http.ResponseWriter, r *http.Request, projectID string) {
	actor, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	var body struct {
		TagIDs []string `json:"tagIds"`
	}
	if err := decodeJSON(r, &body); err != nil {
		h.writeInvalid(w, r, err)
		return
	}
	if err := h.service.ReplaceTags(r.Context(), actor, projectID, body.TagIDs); err != nil {
		h.writeError(w, r, err)
		return
	}
	platformapi.WriteSuccess(w, r, map[string]any{"tagIds": body.TagIDs})
}

func (h *Handler) ListProjectGroups(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	items, err := h.service.ListGroups(r.Context(), actor, r.URL.Query().Get("keyword"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	list := make([]map[string]any, 0, len(items))
	for _, item := range items {
		list = append(list, groupResponse(item))
	}
	platformapi.WriteSuccess(w, r, list)
}
func (h *Handler) CreateProjectGroup(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	input, err := decodeGroupInput(r)
	if err != nil {
		h.writeInvalid(w, r, err)
		return
	}
	item, err := h.service.CreateGroup(r.Context(), actor, input)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	platformapi.WriteSuccess(w, r, groupResponse(item))
}
func (h *Handler) UpdateProjectGroup(w http.ResponseWriter, r *http.Request, groupID string) {
	actor, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	input, err := decodeGroupInput(r)
	if err != nil {
		h.writeInvalid(w, r, err)
		return
	}
	item, err := h.service.UpdateGroup(r.Context(), actor, groupID, input)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	platformapi.WriteSuccess(w, r, groupResponse(item))
}
func (h *Handler) DeleteProjectGroup(w http.ResponseWriter, r *http.Request, groupID string) {
	actor, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	if err := h.service.DeleteGroup(r.Context(), actor, groupID); err != nil {
		h.writeError(w, r, err)
		return
	}
	platformapi.WriteSuccess(w, r, map[string]any{})
}
func (h *Handler) SetProjectGroup(w http.ResponseWriter, r *http.Request, projectID string) {
	actor, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	var body struct {
		GroupID *string `json:"groupId"`
	}
	if err := decodeJSON(r, &body); err != nil {
		h.writeInvalid(w, r, err)
		return
	}
	if err := h.service.SetGroup(r.Context(), actor, projectID, body.GroupID); err != nil {
		h.writeError(w, r, err)
		return
	}
	platformapi.WriteSuccess(w, r, map[string]any{"groupId": body.GroupID})
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
	case errors.Is(err, ErrNotFound):
		platformapi.WriteError(w, r, http.StatusOK, platformapi.ErrorCodeProjectNotFound, err.Error())
	case errors.Is(err, ErrAlreadyExists):
		platformapi.WriteError(w, r, http.StatusOK, platformapi.ErrorCodeProjectCodeExists, err.Error())
	case errors.Is(err, ErrTagNotFound), errors.Is(err, ErrGroupNotFound):
		platformapi.WriteError(w, r, http.StatusOK, platformapi.ErrorCodeNotFound, err.Error())
	case errors.Is(err, auth.ErrPermissionDenied):
		platformapi.WriteError(w, r, http.StatusOK, platformapi.ErrorCodePermissionDenied, err.Error())
	default:
		if strings.Contains(err.Error(), "不能为空") || strings.Contains(err.Error(), "不支持") {
			h.writeInvalid(w, r, err)
			return
		}
		platformapi.WriteError(w, r, http.StatusInternalServerError, platformapi.ErrorCodeInternal, "系统内部错误")
	}
}
func (h *Handler) writeInvalid(w http.ResponseWriter, r *http.Request, err error) {
	platformapi.WriteError(w, r, http.StatusOK, platformapi.ErrorCodeInvalidRequest, err.Error())
}
func decodeJSON(r *http.Request, target any) error {
	return json.NewDecoder(io.LimitReader(r.Body, 2<<20)).Decode(target)
}
func decodeProjectInput(r *http.Request) (ProjectInput, error) {
	var body struct {
		Name        *string `json:"name"`
		Code        *string `json:"code"`
		Description *string `json:"description"`
		Icon        *string `json:"icon"`
		Visibility  *string `json:"visibility"`
	}
	if err := decodeJSON(r, &body); err != nil {
		return ProjectInput{}, err
	}
	return ProjectInput{Name: body.Name, Code: body.Code, Description: body.Description, Icon: body.Icon, Visibility: body.Visibility}, nil
}
func decodeTagInput(r *http.Request) (TagInput, error) {
	var body struct {
		Name        *string `json:"name"`
		Color       *string `json:"color"`
		Description *string `json:"description"`
		SortOrder   *int32  `json:"sortOrder"`
	}
	if err := decodeJSON(r, &body); err != nil {
		return TagInput{}, err
	}
	return TagInput{Name: body.Name, Color: body.Color, Description: body.Description, SortOrder: body.SortOrder}, nil
}
func decodeGroupInput(r *http.Request) (GroupInput, error) {
	var body struct {
		Name        *string `json:"name"`
		Description *string `json:"description"`
		SortOrder   *int32  `json:"sortOrder"`
	}
	if err := decodeJSON(r, &body); err != nil {
		return GroupInput{}, err
	}
	return GroupInput{Name: body.Name, Description: body.Description, SortOrder: body.SortOrder}, nil
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
func firstQuery(r *http.Request, names ...string) string {
	for _, name := range names {
		if values := r.URL.Query()[name]; len(values) > 0 && strings.TrimSpace(values[0]) != "" {
			return strings.TrimSpace(values[0])
		}
	}
	return ""
}
func projectResponse(item Project) map[string]any {
	tags := make([]map[string]any, 0, len(item.Tags))
	for _, tag := range item.Tags {
		tags = append(tags, tagResponse(tag))
	}
	payload := map[string]any{"id": item.ID, "tenantId": item.TenantID, "name": item.Name, "code": item.Code, "description": item.Description, "icon": item.Icon, "status": item.Status, "visibility": item.Visibility, "createdBy": item.CreatedBy, "createdByName": item.CreatedByName, "tags": tags, "createdAt": item.CreatedAt.Format(timeFormat), "updatedAt": item.UpdatedAt.Format(timeFormat)}
	if item.Group != nil {
		payload["group"] = groupResponse(*item.Group)
		payload["groupId"] = item.Group.ID
	} else {
		payload["group"] = nil
		payload["groupId"] = nil
	}
	return payload
}
func tagResponse(item Tag) map[string]any {
	return map[string]any{"id": item.ID, "name": item.Name, "color": item.Color, "description": item.Description, "sortOrder": item.SortOrder, "projectCount": item.ProjectCount, "createdAt": item.CreatedAt.Format(timeFormat), "updatedAt": item.UpdatedAt.Format(timeFormat)}
}
func groupResponse(item Group) map[string]any {
	return map[string]any{"id": item.ID, "name": item.Name, "description": item.Description, "sortOrder": item.SortOrder, "projectCount": item.ProjectCount, "createdAt": item.CreatedAt.Format(timeFormat), "updatedAt": item.UpdatedAt.Format(timeFormat)}
}
