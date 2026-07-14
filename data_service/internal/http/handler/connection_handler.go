package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/indu-forge/data_service/internal/auth"
	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/indu-forge/data_service/internal/http/middleware"
	"github.com/indu-forge/data_service/internal/http/response"
	"github.com/indu-forge/data_service/internal/repository"
	"github.com/indu-forge/data_service/internal/service"
)

// ConnectionHandler 负责承接 connections 领域的 HTTP 请求。
type ConnectionHandler struct {
	service *service.ConnectionService
}

// NewConnectionHandler 创建连接处理器。
func NewConnectionHandler(connectionService *service.ConnectionService) *ConnectionHandler {
	return &ConnectionHandler{service: connectionService}
}

// List 返回项目连接列表。
func (h *ConnectionHandler) List(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}

	query := r.URL.Query()
	isPagedQuery := query.Has("page") || query.Has("pageSize") || query.Has("limit") || query.Has("search") || query.Has("typeGroup")
	if isPagedQuery {
		page, parseErr := parseOptionalInt(query.Get("page"), 1, "page")
		if parseErr != nil {
			return parseErr
		}
		pageSize, parseErr := parseOptionalInt(firstNonEmpty(query.Get("pageSize"), query.Get("limit")), 10, "pageSize")
		if parseErr != nil {
			return parseErr
		}
		if pageSize > 100 {
			pageSize = 100
		}
		connections, total, listErr := h.service.ListConnectionsPage(r.Context(), r.PathValue("projectId"), claims.TenantID, repository.ConnectionListFilter{
			Page: page, PageSize: pageSize, Search: query.Get("search"), TypeGroup: query.Get("typeGroup"),
		})
		if listErr != nil {
			return normalizeRepresentativeHandlerError(listErr)
		}
		totalPages := 0
		if total > 0 {
			totalPages = (total + pageSize - 1) / pageSize
		}
		response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]any{
			"list":       connections,
			"pagination": map[string]int{"page": page, "pageSize": pageSize, "total": total, "totalPages": totalPages},
		})
		return nil
	}

	connections, err := h.service.ListConnections(r.Context(), r.PathValue("projectId"), claims.TenantID)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}

	response.WriteSuccess(w, middleware.RequestID(r.Context()), connections)
	return nil
}

// Get 返回单个项目连接，用于分页列表外的工作台深链接恢复。
func (h *ConnectionHandler) Get(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	connection, err := h.service.GetConnection(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), claims.TenantID)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), connection)
	return nil
}

// Create 创建项目连接。
func (h *ConnectionHandler) Create(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}

	var request struct {
		Name   string         `json:"name"`
		Type   string         `json:"type"`
		Status string         `json:"status"`
		Config map[string]any `json:"config"`
	}
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}

	connection, err := h.service.CreateConnection(r.Context(), r.PathValue("projectId"), claims.TenantID, claims.UserID, service.CreateConnectionInput{
		Name:   request.Name,
		Type:   request.Type,
		Status: request.Status,
		Config: request.Config,
	})
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}

	response.WriteSuccess(w, middleware.RequestID(r.Context()), connection)
	return nil
}

// Update 更新项目连接。
func (h *ConnectionHandler) Update(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}

	var raw map[string]json.RawMessage
	if err := decodeJSONBody(r, &raw); err != nil {
		return err
	}
	if err := validateUpdatePayloadKeys(raw); err != nil {
		return err
	}

	input := service.UpdateConnectionInput{}
	if value, ok := raw["name"]; ok {
		var name string
		if err := json.Unmarshal(value, &name); err != nil {
			return apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "name 字段格式无效", err)
		}
		input.Name = &name
	}
	if value, ok := raw["type"]; ok {
		var connectionType string
		if err := json.Unmarshal(value, &connectionType); err != nil {
			return apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "type 字段格式无效", err)
		}
		input.Type = &connectionType
	}
	if value, ok := raw["status"]; ok {
		var status string
		if err := json.Unmarshal(value, &status); err != nil {
			return apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "status 字段格式无效", err)
		}
		input.Status = &status
	}
	if value, ok := raw["config"]; ok {
		var config map[string]any
		if err := json.Unmarshal(value, &config); err != nil {
			return apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "config 字段格式无效", err)
		}
		input.Config = config
		input.HasConfig = true
	}

	connection, err := h.service.UpdateConnection(
		r.Context(),
		r.PathValue("projectId"),
		r.PathValue("connectionId"),
		claims.TenantID,
		claims.UserID,
		input,
	)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}

	response.WriteSuccess(w, middleware.RequestID(r.Context()), connection)
	return nil
}

// Delete 删除项目连接。
func (h *ConnectionHandler) Delete(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}

	if err := h.service.DeleteConnection(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId")); err != nil {
		return normalizeRepresentativeHandlerError(err)
	}

	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]bool{"deleted": true})
	return nil
}

// TestConnection 测试外部关系库连接。
func (h *ConnectionHandler) TestConnection(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}

	var request struct {
		Type   string         `json:"type"`
		Config map[string]any `json:"config"`
	}
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}

	result, err := h.service.TestConnection(r.Context(), r.PathValue("projectId"), service.CreateConnectionInput{
		Type:   request.Type,
		Config: request.Config,
	})
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}

	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

// UpdateStatus 更新连接状态。
func (h *ConnectionHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}

	var request struct {
		Status string `json:"status"`
	}
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}

	result, err := h.service.UpdateConnectionStatus(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), request.Status)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}

	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

// UpdateOrder 持久化接入源卡片展示顺序。
func (h *ConnectionHandler) UpdateOrder(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}

	var request struct {
		ConnectionIDs []string `json:"connectionIds"`
	}
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}

	if err := h.service.UpdateConnectionOrder(r.Context(), r.PathValue("projectId"), request.ConnectionIDs); err != nil {
		return normalizeRepresentativeHandlerError(err)
	}

	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]bool{"saved": true})
	return nil
}

// ListTables 返回连接下的表列表。
func (h *ConnectionHandler) ListTables(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}

	tables, err := h.service.ListTables(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"))
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}

	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]any{"tables": tables})
	return nil
}

// CreateTable 按结构化表设计创建表。
func (h *ConnectionHandler) CreateTable(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}

	var request struct {
		Name       string                                     `json:"name"`
		Kind       string                                     `json:"kind"`
		Columns    []service.CreateRelationalTableColumnInput `json:"columns"`
		Indexes    []service.CreateRelationalTableIndexInput  `json:"indexes"`
		Timeseries *service.CreateRelationalTimeseriesInput   `json:"timeseries"`
	}
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}

	result, err := h.service.CreateTable(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), service.CreateRelationalTableInput{
		Name:       request.Name,
		Kind:       request.Kind,
		Columns:    request.Columns,
		Indexes:    request.Indexes,
		Timeseries: request.Timeseries,
	})
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}

	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

// RenameTable 重命名物理表，并同步工作台表分组映射。
func (h *ConnectionHandler) RenameTable(w http.ResponseWriter, r *http.Request) error {
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
	if err := h.service.RenameTable(
		r.Context(),
		r.PathValue("projectId"),
		r.PathValue("connectionId"),
		r.PathValue("tableName"),
		request.Name,
		claims.UserID,
	); err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]bool{"renamed": true})
	return nil
}

// DeleteTable 删除物理表，并清理工作台表分组映射。
func (h *ConnectionHandler) DeleteTable(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}
	if err := h.service.DeleteTable(
		r.Context(),
		r.PathValue("projectId"),
		r.PathValue("connectionId"),
		r.PathValue("tableName"),
	); err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]bool{"deleted": true})
	return nil
}

// GetTableStructure 返回表结构。
func (h *ConnectionHandler) GetTableStructure(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}

	result, err := h.service.GetTableStructure(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), r.PathValue("tableName"))
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}

	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

// UpdateTableStructure 修改内置 IF 表结构。
func (h *ConnectionHandler) UpdateTableStructure(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}

	var request struct {
		Columns []service.CreateRelationalTableColumnInput `json:"columns"`
		Indexes []service.CreateRelationalTableIndexInput  `json:"indexes"`
	}
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}

	result, err := h.service.UpdateTableStructure(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), r.PathValue("tableName"), service.UpdateRelationalTableInput{
		Columns: request.Columns,
		Indexes: request.Indexes,
	})
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}

	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

// GetTableData 返回表数据预览。
func (h *ConnectionHandler) GetTableData(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}

	page, err := parseOptionalInt(r.URL.Query().Get("page"), 1, "page")
	if err != nil {
		return err
	}
	limit, err := parseOptionalInt(firstNonEmpty(r.URL.Query().Get("limit"), r.URL.Query().Get("pageSize")), 100, "limit")
	if err != nil {
		return err
	}

	result, err := h.service.GetTableData(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), r.PathValue("tableName"), page, limit)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}

	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

// ExecuteSQL 执行只读 SQL。
func (h *ConnectionHandler) ExecuteSQL(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}

	var request struct {
		SQL        string `json:"sql"`
		Parameters []any  `json:"parameters"`
	}
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}

	result, err := h.service.ExecuteSQL(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), request.SQL, request.Parameters)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}

	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

// ListRedisKeys 返回 Redis 工作台 key 列表。
func (h *ConnectionHandler) ListRedisKeys(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}
	limit, err := parseOptionalInt(r.URL.Query().Get("limit"), 200, "limit")
	if err != nil {
		return err
	}
	result, err := h.service.ListRedisKeys(
		r.Context(),
		r.PathValue("projectId"),
		r.PathValue("connectionId"),
		r.URL.Query().Get("pattern"),
		limit,
	)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

// GetRedisValue 返回 Redis 工作台单 key 值。
func (h *ConnectionHandler) GetRedisValue(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}
	result, err := h.service.GetRedisValue(
		r.Context(),
		r.PathValue("projectId"),
		r.PathValue("connectionId"),
		r.URL.Query().Get("key"),
	)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

// ExecuteRedisCommand 执行 Redis 工作台受限只读命令。
func (h *ConnectionHandler) ExecuteRedisCommand(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}
	var request struct {
		Command string   `json:"command"`
		Args    []string `json:"args"`
	}
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}
	result, err := h.service.ExecuteRedisCommand(
		r.Context(),
		r.PathValue("projectId"),
		r.PathValue("connectionId"),
		request.Command,
		request.Args,
	)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func requireClaims(r *http.Request) (*auth.Claims, error) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeAuthTokenRequired, http.StatusUnauthorized, "请先完成认证")
	}
	return claims, nil
}

// normalizeRepresentativeHandlerError 统一代表 handler 的业务失败边界：
// 1. 业务失败（部分 BAD_REQUEST / NOT_FOUND / 权限类）统一返回 HTTP 200 + 非 0 code。
// 2. 技术异常继续保留 4xx/5xx，避免认证、解码、下游故障被压成 200。
func normalizeRepresentativeHandlerError(err error) error {
	if err == nil {
		return nil
	}

	var appErr *apperrors.AppError
	if !errors.As(err, &appErr) || appErr == nil {
		return err
	}
	if !isRepresentativeBusinessFailure(appErr) {
		return err
	}
	if appErr.StatusCode == http.StatusOK {
		return err
	}

	if appErr.Err != nil {
		return apperrors.WrapAppError(appErr.Code, http.StatusOK, appErr.Message, appErr.Err)
	}
	return apperrors.NewAppError(appErr.Code, http.StatusOK, appErr.Message)
}

func isRepresentativeBusinessFailure(appErr *apperrors.AppError) bool {
	if appErr == nil {
		return false
	}
	if hasWrappedAppErrorCause(appErr) {
		return false
	}

	switch appErr.Code {
	case apperrors.ErrorCodeBadRequest:
		return appErr.StatusCode == http.StatusBadRequest
	case apperrors.ErrorCodeNotFound:
		return appErr.StatusCode == http.StatusNotFound
	case apperrors.ErrorCodePermissionInsufficient, apperrors.ErrorCodePermissionProjectMismatch:
		return appErr.StatusCode == http.StatusForbidden
	case apperrors.ErrorCodeAuthTokenRequired, apperrors.ErrorCodeAuthTokenInvalid, apperrors.ErrorCodeAuthSecretRequired:
		return false
	case apperrors.ErrorCodeInternal:
		return false
	default:
		return false
	}
}

func hasWrappedAppErrorCause(appErr *apperrors.AppError) bool {
	if appErr == nil {
		return false
	}
	return errors.Unwrap(appErr) != nil
}

func decodeJSONBody(r *http.Request, target any) error {
	if r.Body == nil {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "请求体不能为空")
	}

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "请求体 JSON 格式无效", err)
	}
	if decoder.More() {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "请求体只能包含一个 JSON 对象")
	}
	return nil
}

func validateUpdatePayloadKeys(raw map[string]json.RawMessage) error {
	allowedFields := map[string]struct{}{
		"name":   {},
		"type":   {},
		"status": {},
		"config": {},
	}

	for field := range raw {
		if _, ok := allowedFields[field]; ok {
			continue
		}
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "更新请求包含不支持的字段: "+field)
	}

	return nil
}
