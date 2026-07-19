package handler

import (
	"net/http"
	"strings"

	"github.com/indu-forge/data_service/internal/http/middleware"
	"github.com/indu-forge/data_service/internal/http/response"
	"github.com/indu-forge/data_service/internal/repository"
	"github.com/indu-forge/data_service/internal/service"
)

type CollectorPointHandler struct {
	service *service.CollectorPointService
}

func NewCollectorPointHandler(pointService *service.CollectorPointService) *CollectorPointHandler {
	return &CollectorPointHandler{service: pointService}
}

func (h *CollectorPointHandler) ListGroups(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}
	var parentID *string
	if value := strings.TrimSpace(r.URL.Query().Get("parentId")); value != "" {
		parentID = &value
	}
	result, err := h.service.ListPointGroups(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), parentID)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}
func (h *CollectorPointHandler) CreateGroup(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}
	var input struct {
		ParentID  *string        `json:"parentId"`
		Name      string         `json:"name"`
		SortOrder int            `json:"sortOrder"`
		Metadata  map[string]any `json:"metadata"`
	}
	if err := decodeJSONBody(r, &input); err != nil {
		return err
	}
	result, err := h.service.CreatePointGroup(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), input.ParentID, input.Name, input.SortOrder, input.Metadata)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}
func (h *CollectorPointHandler) UpdateGroup(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}
	var input struct {
		Name string `json:"name"`
	}
	if err := decodeJSONBody(r, &input); err != nil {
		return err
	}
	result, err := h.service.UpdatePointGroup(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), r.PathValue("groupId"), input.Name)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func (h *CollectorPointHandler) DeleteGroup(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}
	if err := h.service.DeletePointGroup(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), r.PathValue("groupId")); err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]bool{"deleted": true})
	return nil
}

func (h *CollectorPointHandler) ListPoints(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}
	query := r.URL.Query()
	page, err := parseOptionalInt(query.Get("page"), 1, "page")
	if err != nil {
		return err
	}
	pageSize, err := parseOptionalInt(firstNonEmpty(query.Get("pageSize"), query.Get("limit")), 20, "pageSize")
	if err != nil {
		return err
	}
	enabled, err := parseOptionalBool(query.Get("enabled"))
	if err != nil {
		return err
	}
	var groupID *string
	if value := strings.TrimSpace(query.Get("groupId")); value != "" {
		groupID = &value
	}
	result, err := h.service.ListPoints(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), repository.CollectorPointListFilter{Page: page, PageSize: pageSize, Search: query.Get("search"), GroupID: groupID, Enabled: enabled, DataType: query.Get("dataType"), SortBy: query.Get("sortBy"), SortOrder: query.Get("sortOrder")})
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}
func (h *CollectorPointHandler) CheckAddresses(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}
	var input struct {
		Addresses []map[string]any `json:"addresses"`
	}
	if err := decodeJSONBody(r, &input); err != nil {
		return err
	}
	indexes, err := h.service.FindExistingPointAddressIndexes(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), input.Addresses)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]any{"indexes": indexes})
	return nil
}

func (h *CollectorPointHandler) CreateBatch(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var input struct {
		Points []service.CreateCollectorPointInput `json:"points"`
	}
	if err := decodeJSONBody(r, &input); err != nil {
		return err
	}
	result, err := h.service.CreatePointsBatch(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), claims.UserID, input.Points)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]any{"list": result})
	return nil
}
func (h *CollectorPointHandler) UpdateBatch(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var input struct {
		Points []service.UpdateCollectorPointInput `json:"points"`
	}
	if err := decodeJSONBody(r, &input); err != nil {
		return err
	}
	result, err := h.service.UpdatePointsBatch(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), claims.UserID, input.Points)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]any{"list": result})
	return nil
}
func (h *CollectorPointHandler) DeleteBatch(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}
	var input struct {
		PointIDs []string `json:"pointIds"`
	}
	if err := decodeJSONBody(r, &input); err != nil {
		return err
	}
	if err := h.service.DeletePointsBatch(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), input.PointIDs); err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]bool{"deleted": true})
	return nil
}
func (h *CollectorPointHandler) MoveBatch(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}
	var input struct {
		PointIDs []string `json:"pointIds"`
		GroupID  *string  `json:"groupId"`
	}
	if err := decodeJSONBody(r, &input); err != nil {
		return err
	}
	if err := h.service.MovePointsBatch(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), input.GroupID, input.PointIDs); err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]bool{"moved": true})
	return nil
}
