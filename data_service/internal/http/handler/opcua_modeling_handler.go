package handler

import (
	"net/http"

	"github.com/indu-forge/data_service/internal/http/middleware"
	"github.com/indu-forge/data_service/internal/http/response"
	"github.com/indu-forge/data_service/internal/service"
)

// OpcuaModelingHandler 暴露 OPC UA 点位建模工作台接口。
type OpcuaModelingHandler struct {
	service *service.OpcuaModelingService
}

// NewOpcuaModelingHandler 创建 OPC UA 点位建模接口处理器。
func NewOpcuaModelingHandler(service *service.OpcuaModelingService) *OpcuaModelingHandler {
	return &OpcuaModelingHandler{service: service}
}

// ListGroups 返回当前 OPC UA 接入源下的变量组。
func (h *OpcuaModelingHandler) ListGroups(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}
	result, err := h.service.ListGroups(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"))
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]any{"list": result})
	return nil
}

// CreateGroup 创建变量组。
func (h *OpcuaModelingHandler) CreateGroup(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var request struct {
		ParentID    *string `json:"parentId"`
		Name        string  `json:"name"`
		Description *string `json:"description"`
		SortOrder   int     `json:"sortOrder"`
	}
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}
	result, err := h.service.CreateGroup(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), claims.UserID, service.CreateOpcuaNodeGroupInput{
		ParentID:    request.ParentID,
		Name:        request.Name,
		Description: request.Description,
		SortOrder:   request.SortOrder,
	})
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

// UpdateGroup 更新变量组。
func (h *OpcuaModelingHandler) UpdateGroup(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var request struct {
		ParentID    *string `json:"parentId"`
		HasParentID bool    `json:"hasParentId"`
		Name        string  `json:"name"`
		Description *string `json:"description"`
		SortOrder   int     `json:"sortOrder"`
	}
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}
	result, err := h.service.UpdateGroup(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), r.PathValue("groupId"), claims.UserID, service.UpdateOpcuaNodeGroupInput{
		ParentID:    request.ParentID,
		HasParentID: request.HasParentID,
		Name:        request.Name,
		Description: request.Description,
		SortOrder:   request.SortOrder,
	})
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

// DeleteGroup 删除变量组，组内变量保留并移动到未分组。
func (h *OpcuaModelingHandler) DeleteGroup(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	if err := h.service.DeleteGroup(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), r.PathValue("groupId"), claims.UserID); err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]bool{"deleted": true})
	return nil
}

// ListNodes 返回变量列表。
func (h *OpcuaModelingHandler) ListNodes(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}
	query := r.URL.Query()
	var groupID *string
	if value := query.Get("groupId"); value != "" {
		groupID = &value
	}
	page, err := parseOptionalInt(query.Get("page"), 1, "page")
	if err != nil {
		return err
	}
	pageSize, err := parseOptionalInt(firstNonEmpty(query.Get("pageSize"), query.Get("limit")), 20, "pageSize")
	if err != nil {
		return err
	}
	result, err := h.service.ListNodesPage(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), groupID, page, pageSize)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

// CreateNode 创建变量并自动同步数据点。
func (h *OpcuaModelingHandler) CreateNode(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var request opcuaNodeRequest
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}
	result, err := h.service.CreateNode(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), claims.UserID, request.toCreateInput())
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

// BatchImportNodes 批量导入已确认的 OPC UA 变量。
func (h *OpcuaModelingHandler) BatchImportNodes(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var request struct {
		GroupID *string                        `json:"groupId"`
		Nodes   []service.ImportOpcuaNodeInput `json:"nodes"`
	}
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}
	result, err := h.service.BatchImportNodes(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), claims.UserID, request.GroupID, request.Nodes)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]any{"list": result})
	return nil
}

// UpdateNode 更新变量并同步数据点。
func (h *OpcuaModelingHandler) UpdateNode(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var request opcuaNodeRequest
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}
	result, err := h.service.UpdateNode(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), r.PathValue("nodeId"), claims.UserID, request.toUpdateInput())
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

// DeleteNode 删除变量并将关联数据点标记失效。
func (h *OpcuaModelingHandler) DeleteNode(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	if err := h.service.DeleteNode(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), r.PathValue("nodeId"), claims.UserID); err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]bool{"deleted": true})
	return nil
}

// ValidateModel 执行建模校验。
func (h *OpcuaModelingHandler) ValidateModel(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}
	result, err := h.service.ValidateModel(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"))
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

// PreviewNodes 返回辅助预览数据。真实采集由运行态执行。
func (h *OpcuaModelingHandler) PreviewNodes(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}
	var request struct {
		GroupID *string `json:"groupId"`
	}
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}
	result, err := h.service.PreviewNodes(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), request.GroupID)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

type opcuaNodeRequest struct {
	GroupID     *string  `json:"groupId"`
	HasGroupID  bool     `json:"hasGroupId"`
	Name        string   `json:"name"`
	Code        string   `json:"code"`
	NodeID      string   `json:"nodeId"`
	BrowseName  *string  `json:"browseName"`
	DisplayName *string  `json:"displayName"`
	DataType    string   `json:"dataType"`
	Unit        *string  `json:"unit"`
	SamplingMS  *int     `json:"samplingMs"`
	Deadband    *float64 `json:"deadband"`
	AccessLevel string   `json:"accessLevel"`
	Description *string  `json:"description"`
	SortOrder   int      `json:"sortOrder"`
	Status      string   `json:"status"`
}

func (r opcuaNodeRequest) toCreateInput() service.CreateOpcuaNodeInput {
	return service.CreateOpcuaNodeInput{
		GroupID:     r.GroupID,
		Name:        r.Name,
		Code:        r.Code,
		NodeID:      r.NodeID,
		BrowseName:  r.BrowseName,
		DisplayName: r.DisplayName,
		DataType:    r.DataType,
		Unit:        r.Unit,
		SamplingMS:  r.SamplingMS,
		Deadband:    r.Deadband,
		AccessLevel: r.AccessLevel,
		Description: r.Description,
		SortOrder:   r.SortOrder,
	}
}

func (r opcuaNodeRequest) toUpdateInput() service.UpdateOpcuaNodeInput {
	return service.UpdateOpcuaNodeInput{
		GroupID:     r.GroupID,
		HasGroupID:  r.HasGroupID,
		Name:        r.Name,
		Code:        r.Code,
		NodeID:      r.NodeID,
		BrowseName:  r.BrowseName,
		DisplayName: r.DisplayName,
		DataType:    r.DataType,
		Unit:        r.Unit,
		SamplingMS:  r.SamplingMS,
		Deadband:    r.Deadband,
		AccessLevel: r.AccessLevel,
		Description: r.Description,
		SortOrder:   r.SortOrder,
		Status:      r.Status,
	}
}
