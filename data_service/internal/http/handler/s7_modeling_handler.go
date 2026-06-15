package handler

import (
	"net/http"

	"github.com/indu-forge/data_service/internal/http/middleware"
	"github.com/indu-forge/data_service/internal/http/response"
	"github.com/indu-forge/data_service/internal/service"
)

// S7ModelingHandler 暴露 S7 变量建模工作台接口。
type S7ModelingHandler struct {
	service *service.S7ModelingService
}

// NewS7ModelingHandler 创建 S7 建模接口处理器。
func NewS7ModelingHandler(service *service.S7ModelingService) *S7ModelingHandler {
	return &S7ModelingHandler{service: service}
}

func (h *S7ModelingHandler) GetProfile(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}
	result, err := h.service.GetProfile(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"))
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func (h *S7ModelingHandler) UpsertProfile(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var request s7ProfileRequest
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}
	result, err := h.service.UpsertProfile(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), claims.UserID, request.toInput())
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func (h *S7ModelingHandler) ListGroups(w http.ResponseWriter, r *http.Request) error {
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

func (h *S7ModelingHandler) CreateGroup(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var request s7GroupRequest
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}
	result, err := h.service.CreateGroup(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), claims.UserID, service.CreateS7VariableGroupInput{
		ParentID: request.ParentID, Name: request.Name, Code: request.Code, Description: request.Description, SortOrder: request.SortOrder,
	})
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func (h *S7ModelingHandler) UpdateGroup(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var request s7GroupRequest
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}
	result, err := h.service.UpdateGroup(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), r.PathValue("groupId"), claims.UserID, service.UpdateS7VariableGroupInput{
		ParentID: request.ParentID, HasParentID: request.HasParentID, Name: request.Name, Code: request.Code, Description: request.Description, SortOrder: request.SortOrder,
	})
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func (h *S7ModelingHandler) DeleteGroup(w http.ResponseWriter, r *http.Request) error {
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

func (h *S7ModelingHandler) ListVariables(w http.ResponseWriter, r *http.Request) error {
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
	result, err := h.service.ListVariablesPage(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), groupID, page, pageSize)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func (h *S7ModelingHandler) CreateVariable(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var request s7VariableRequest
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}
	result, err := h.service.CreateVariable(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), claims.UserID, request.toCreateInput())
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func (h *S7ModelingHandler) BatchImportVariables(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var request struct {
		GroupID   *string                         `json:"groupId"`
		Variables []service.ImportS7VariableInput `json:"variables"`
	}
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}
	result, err := h.service.BatchImportVariables(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), claims.UserID, request.GroupID, request.Variables)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]any{"list": result})
	return nil
}

func (h *S7ModelingHandler) UpdateVariable(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var request s7VariableRequest
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}
	result, err := h.service.UpdateVariable(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), r.PathValue("variableId"), claims.UserID, request.toUpdateInput())
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func (h *S7ModelingHandler) DeleteVariable(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	if err := h.service.DeleteVariable(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), r.PathValue("variableId"), claims.UserID); err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]bool{"deleted": true})
	return nil
}

func (h *S7ModelingHandler) ValidateModel(w http.ResponseWriter, r *http.Request) error {
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

func (h *S7ModelingHandler) PreviewVariables(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}
	var request struct {
		GroupID *string `json:"groupId"`
	}
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}
	result, err := h.service.PreviewVariables(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), request.GroupID)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func (h *S7ModelingHandler) EstimateReadPlans(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}
	var groupID *string
	if value := r.URL.Query().Get("groupId"); value != "" {
		groupID = &value
	}
	result, err := h.service.EstimateReadPlans(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), groupID)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

type s7ProfileRequest struct {
	PlcFamily            string         `json:"plcFamily"`
	CommunicationMode    string         `json:"communicationMode"`
	Host                 string         `json:"host"`
	Port                 *int           `json:"port"`
	Rack                 *int           `json:"rack"`
	Slot                 *int           `json:"slot"`
	LocalTSAP            *string        `json:"localTsap"`
	RemoteTSAP           *string        `json:"remoteTsap"`
	PollIntervalMS       *int           `json:"pollIntervalMs"`
	ConnectTimeoutMS     *int           `json:"connectTimeoutMs"`
	ReadTimeoutMS        *int           `json:"readTimeoutMs"`
	PDUSize              *int           `json:"pduSize"`
	MaxReadBytes         *int           `json:"maxReadBytes"`
	MaxGapBytes          *int           `json:"maxGapBytes"`
	MaxConcurrentReads   *int           `json:"maxConcurrentReads"`
	ByteOrder            string         `json:"byteOrder"`
	WordOrder            string         `json:"wordOrder"`
	OptimizedBlockAccess bool           `json:"optimizedBlockAccess"`
	AllowAbsoluteAddress bool           `json:"allowAbsoluteAddress"`
	AllowSymbolAddress   bool           `json:"allowSymbolAddress"`
	SupportedAreas       []string       `json:"supportedAreas"`
	Options              map[string]any `json:"options"`
}

func (r s7ProfileRequest) toInput() service.UpsertS7ProfileInput {
	return service.UpsertS7ProfileInput{
		PlcFamily:            r.PlcFamily,
		CommunicationMode:    r.CommunicationMode,
		Host:                 r.Host,
		Port:                 r.Port,
		Rack:                 r.Rack,
		Slot:                 r.Slot,
		LocalTSAP:            r.LocalTSAP,
		RemoteTSAP:           r.RemoteTSAP,
		PollIntervalMS:       r.PollIntervalMS,
		ConnectTimeoutMS:     r.ConnectTimeoutMS,
		ReadTimeoutMS:        r.ReadTimeoutMS,
		PDUSize:              r.PDUSize,
		MaxReadBytes:         r.MaxReadBytes,
		MaxGapBytes:          r.MaxGapBytes,
		MaxConcurrentReads:   r.MaxConcurrentReads,
		ByteOrder:            r.ByteOrder,
		WordOrder:            r.WordOrder,
		OptimizedBlockAccess: r.OptimizedBlockAccess,
		AllowAbsoluteAddress: r.AllowAbsoluteAddress,
		AllowSymbolAddress:   r.AllowSymbolAddress,
		SupportedAreas:       r.SupportedAreas,
		Options:              r.Options,
	}
}

type s7GroupRequest struct {
	ParentID    *string `json:"parentId"`
	HasParentID bool    `json:"hasParentId"`
	Name        string  `json:"name"`
	Code        string  `json:"code"`
	Description *string `json:"description"`
	SortOrder   int     `json:"sortOrder"`
}

type s7VariableRequest struct {
	GroupID        *string        `json:"groupId"`
	HasGroupID     bool           `json:"hasGroupId"`
	Name           string         `json:"name"`
	Code           string         `json:"code"`
	Description    *string        `json:"description"`
	AddressText    string         `json:"addressText"`
	DataType       string         `json:"dataType"`
	Length         *int           `json:"length"`
	ArrayLength    *int           `json:"arrayLength"`
	ByteOrder      string         `json:"byteOrder"`
	WordOrder      string         `json:"wordOrder"`
	Scale          *float64       `json:"scale"`
	Offset         *float64       `json:"offset"`
	Unit           *string        `json:"unit"`
	PollIntervalMS *int           `json:"pollIntervalMs"`
	AccessLevel    string         `json:"accessLevel"`
	QualityRule    map[string]any `json:"qualityRule"`
	Metadata       map[string]any `json:"metadata"`
	SortOrder      int            `json:"sortOrder"`
	Status         string         `json:"status"`
}

func (r s7VariableRequest) toCreateInput() service.CreateS7VariableInput {
	return service.CreateS7VariableInput{
		GroupID: r.GroupID, Name: r.Name, Code: r.Code, Description: r.Description, AddressText: r.AddressText,
		DataType: r.DataType, Length: r.Length, ArrayLength: r.ArrayLength, ByteOrder: r.ByteOrder,
		WordOrder: r.WordOrder, Scale: r.Scale, Offset: r.Offset, Unit: r.Unit,
		PollIntervalMS: r.PollIntervalMS, AccessLevel: r.AccessLevel, QualityRule: r.QualityRule,
		Metadata: r.Metadata, SortOrder: r.SortOrder,
	}
}

func (r s7VariableRequest) toUpdateInput() service.UpdateS7VariableInput {
	return service.UpdateS7VariableInput{
		GroupID: r.GroupID, HasGroupID: r.HasGroupID, Name: r.Name, Code: r.Code, Description: r.Description, AddressText: r.AddressText,
		DataType: r.DataType, Length: r.Length, ArrayLength: r.ArrayLength, ByteOrder: r.ByteOrder,
		WordOrder: r.WordOrder, Scale: r.Scale, Offset: r.Offset, Unit: r.Unit,
		PollIntervalMS: r.PollIntervalMS, AccessLevel: r.AccessLevel, QualityRule: r.QualityRule,
		Metadata: r.Metadata, SortOrder: r.SortOrder, Status: r.Status,
	}
}
