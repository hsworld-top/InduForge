package handler

import (
	"net/http"

	"github.com/indu-forge/data_service/internal/http/middleware"
	"github.com/indu-forge/data_service/internal/http/response"
	"github.com/indu-forge/data_service/internal/service"
)

type CollectorCatalogHandler struct {
	service *service.CollectorCatalogService
}

func NewCollectorCatalogHandler(catalogService *service.CollectorCatalogService) *CollectorCatalogHandler {
	return &CollectorCatalogHandler{service: catalogService}
}

func (h *CollectorCatalogHandler) List(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}
	page, err := parseOptionalInt(r.URL.Query().Get("page"), 1, "page")
	if err != nil {
		return err
	}
	pageSize, err := parseOptionalInt(firstNonEmpty(r.URL.Query().Get("pageSize"), r.URL.Query().Get("limit")), 20, "pageSize")
	if err != nil {
		return err
	}
	query := r.URL.Query()
	result, err := h.service.List(service.ListCollectorDriversInput{
		Page: page, PageSize: pageSize, Search: query.Get("search"),
		ProtocolFamily: query.Get("protocolFamily"), Category: query.Get("category"), Transport: query.Get("transport"),
	})
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func (h *CollectorCatalogHandler) Get(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}
	result, err := h.service.Get(r.PathValue("driverId"))
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}
