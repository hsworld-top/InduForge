package handler

import (
	"encoding/csv"
	"encoding/json"
	"net/http"
	"strconv"

	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/indu-forge/data_service/internal/http/middleware"
	"github.com/indu-forge/data_service/internal/http/response"
	"github.com/indu-forge/data_service/internal/repository"
	"github.com/indu-forge/data_service/internal/service"
)

type CollectorHandler struct{ service *service.CollectorService }

func NewCollectorHandler(collectorService *service.CollectorService) *CollectorHandler {
	return &CollectorHandler{service: collectorService}
}

func (h *CollectorHandler) GetConnectionDiagnostic(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}
	result, err := h.service.GetConnectionDiagnostic(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"))
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func (h *CollectorHandler) ExportConnectionDiagnostic(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}
	result, err := h.service.GetConnectionDiagnostic(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"))
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="collector-diagnostic.csv"`)
	w.Header().Set("X-Collector-Point-Count", strconv.Itoa(result.PointCount))
	if _, err := w.Write([]byte{0xEF, 0xBB, 0xBF}); err != nil {
		return err
	}
	writer := csv.NewWriter(w)
	if err := writer.Write([]string{"点位ID", "点位名称", "地址", "错误码", "错误信息", "最近尝试时间"}); err != nil {
		return err
	}
	for _, point := range result.FailedPoints {
		if err := writer.Write([]string{point.PointID, point.Name, point.AddressText, point.ErrorCode, point.ErrorMessage, point.AttemptedAt.Format("2006-01-02 15:04:05Z07:00")}); err != nil {
			return err
		}
	}
	if len(result.FailedPoints) == 0 {
		if err := writer.Write([]string{"", "暂无失败点位", "", "", "", ""}); err != nil {
			return err
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return err
	}
	return nil
}

func (h *CollectorHandler) ListConnections(w http.ResponseWriter, r *http.Request) error {
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

	result, err := h.service.ListConnections(r.Context(), r.PathValue("projectId"), repository.CollectorConnectionListFilter{Page: page, PageSize: pageSize, Search: query.Get("search"), ProtocolFamily: query.Get("protocolFamily"), DriverID: query.Get("driverId"), SortBy: query.Get("sortBy"), SortOrder: query.Get("sortOrder")})
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func (h *CollectorHandler) CreateConnection(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var input service.CreateCollectorConnectionInput
	if err := decodeJSONBody(r, &input); err != nil {
		return err
	}
	result, err := h.service.CreateConnection(r.Context(), r.PathValue("projectId"), claims.UserID, input)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func (h *CollectorHandler) GetConnection(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}
	result, err := h.service.GetConnection(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"))
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func (h *CollectorHandler) UpdateConnection(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var raw map[string]json.RawMessage
	if err := decodeJSONBody(r, &raw); err != nil {
		return err
	}
	allowed := map[string]struct{}{"name": {}, "config": {}, "metadata": {}, "secrets": {}, "isEnabled": {}, "defaultAcquisition": {}}
	for key := range raw {
		if _, ok := allowed[key]; !ok {
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "更新请求包含不支持的字段: "+key)
		}
	}
	input := service.UpdateCollectorConnectionInput{}
	if value, ok := raw["name"]; ok {
		var parsed string
		if err := json.Unmarshal(value, &parsed); err != nil {
			return invalidCollectorField("name", err)
		}
		input.Name = &parsed
	}

	if value, ok := raw["config"]; ok {
		if err := json.Unmarshal(value, &input.Config); err != nil {
			return invalidCollectorField("config", err)
		}
		input.HasConfig = true
	}
	if value, ok := raw["metadata"]; ok {
		if err := json.Unmarshal(value, &input.Metadata); err != nil {
			return invalidCollectorField("metadata", err)
		}
		input.HasMetadata = true
	}
	if value, ok := raw["secrets"]; ok {
		if err := json.Unmarshal(value, &input.Secrets); err != nil {
			return invalidCollectorField("secrets", err)
		}
	}
	if value, ok := raw["isEnabled"]; ok {
		var parsed bool
		if err := json.Unmarshal(value, &parsed); err != nil {
			return invalidCollectorField("isEnabled", err)
		}
		input.IsEnabled = &parsed
	}
	if value, ok := raw["defaultAcquisition"]; ok {
		if err := json.Unmarshal(value, &input.DefaultAcquisition); err != nil {
			return invalidCollectorField("defaultAcquisition", err)
		}
		input.HasDefaultAcquisition = true
	}
	result, err := h.service.UpdateConnection(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), claims.UserID, input)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func (h *CollectorHandler) DeleteConnection(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}
	if err := h.service.DeleteConnection(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId")); err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]bool{"deleted": true})
	return nil
}

func (h *CollectorHandler) DeleteConnectionImpact(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}
	impact, err := h.service.GetConnectionDeleteImpact(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"))
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]any{
		"scopeType": impact.ScopeType, "scopeId": impact.ScopeID, "name": impact.Name,
		"canDelete":           impact.CanDelete(),
		"generatedDatapoints": map[string]any{"count": impact.GeneratedPointCount, "action": "mark_invalid"},
		"ownedResources":      impact.OwnedResources, "blockingUsages": impact.BlockingUsages,
	})
	return nil
}

func invalidCollectorField(field string, err error) error {
	return apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, field+" 字段格式无效", err)
}
