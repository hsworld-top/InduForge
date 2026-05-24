package handler

import (
	"encoding/json"
	"net/http"

	"github.com/indu-forge/data_service/internal/http/middleware"
	"github.com/indu-forge/data_service/internal/http/response"
	"github.com/indu-forge/data_service/internal/service"
)

// WorkbenchGroupHandler 承接 SQL 工作台查询/表分组请求。
type WorkbenchGroupHandler struct {
	service *service.WorkbenchGroupService
}

func NewWorkbenchGroupHandler(groupService *service.WorkbenchGroupService) *WorkbenchGroupHandler {
	return &WorkbenchGroupHandler{service: groupService}
}

func (h *WorkbenchGroupHandler) ListGroups(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}
	result, err := h.service.ListGroups(
		r.Context(),
		r.PathValue("projectId"),
		r.PathValue("connectionId"),
		r.URL.Query().Get("scope"),
	)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]any{"groups": result})
	return nil
}

func (h *WorkbenchGroupHandler) CreateGroup(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var request struct {
		Scope string `json:"scope"`
		Name  string `json:"name"`
	}
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}
	result, err := h.service.CreateGroup(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), request.Scope, claims.UserID, request.Name)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]any{"group": result})
	return nil
}

func (h *WorkbenchGroupHandler) UpdateGroup(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var request struct {
		Name string `json:"name"`
	}
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}
	result, err := h.service.UpdateGroup(r.Context(), r.PathValue("projectId"), r.PathValue("groupId"), claims.UserID, request.Name)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]any{"group": result})
	return nil
}

func (h *WorkbenchGroupHandler) DeleteGroup(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}
	if err := h.service.DeleteGroup(r.Context(), r.PathValue("projectId"), r.PathValue("groupId")); err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]bool{"deleted": true})
	return nil
}

func (h *WorkbenchGroupHandler) MoveQuery(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	groupID, err := parseOptionalGroupID(r)
	if err != nil {
		return err
	}
	if err := h.service.MoveQuery(r.Context(), r.PathValue("projectId"), r.PathValue("queryId"), claims.UserID, groupID); err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]bool{"moved": true})
	return nil
}

func (h *WorkbenchGroupHandler) ListTableMembers(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}
	result, err := h.service.ListTableMembers(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"))
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]any{"members": result})
	return nil
}

func (h *WorkbenchGroupHandler) MoveTable(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	groupID, err := parseOptionalGroupID(r)
	if err != nil {
		return err
	}
	if err := h.service.MoveTable(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), claims.UserID, r.PathValue("tableName"), groupID); err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]bool{"moved": true})
	return nil
}

func parseOptionalGroupID(r *http.Request) (*string, error) {
	var raw map[string]json.RawMessage
	if err := decodeJSONBody(r, &raw); err != nil {
		return nil, err
	}
	value, ok := raw["groupId"]
	if !ok || string(value) == "null" {
		return nil, nil
	}
	var groupID string
	if err := json.Unmarshal(value, &groupID); err != nil {
		return nil, err
	}
	return &groupID, nil
}
