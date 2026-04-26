package handler

import (
	"net/http"

	"github.com/indu-forge/data_service/internal/http/middleware"
	"github.com/indu-forge/data_service/internal/http/response"
	"github.com/indu-forge/data_service/internal/service"
)

// ContractCheckHandler 提供项目数据域契约 dry-run 检查入口。
type ContractCheckHandler struct {
	service *service.ContractCheckService
}

func NewContractCheckHandler(contractCheckService *service.ContractCheckService) *ContractCheckHandler {
	return &ContractCheckHandler{service: contractCheckService}
}

func (h *ContractCheckHandler) Run(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var request struct {
		Scope      string `json:"scope"`
		ObjectType string `json:"objectType"`
		ObjectID   string `json:"objectId"`
	}
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}
	result, err := h.service.Run(r.Context(), claims, r.PathValue("projectId"), service.ContractCheckInput{
		Scope:      request.Scope,
		ObjectType: request.ObjectType,
		ObjectID:   request.ObjectID,
	})
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func (h *ContractCheckHandler) Latest(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	result, err := h.service.Latest(r.Context(), claims, r.PathValue("projectId"))
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func (h *ContractCheckHandler) Runs(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
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
	result, err := h.service.ListRuns(r.Context(), claims, r.PathValue("projectId"), page, pageSize)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}
