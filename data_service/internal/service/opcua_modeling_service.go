package service

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/indu-forge/data_service/internal/repository"
)

var opcuaCodeSanitizer = regexp.MustCompile(`[^a-zA-Z0-9_]+`)

// OpcuaNodeGroup 表示前端工作台使用的 OPC UA 变量组。
type OpcuaNodeGroup struct {
	ID           string    `json:"id"`
	ProjectID    string    `json:"projectId"`
	ConnectionID string    `json:"connectionId"`
	ParentID     *string   `json:"parentId"`
	Name         string    `json:"name"`
	Description  *string   `json:"description"`
	SortOrder    int       `json:"sortOrder"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// OpcuaNode 表示前端工作台使用的 OPC UA 变量。
type OpcuaNode struct {
	ID              string     `json:"id"`
	ProjectID       string     `json:"projectId"`
	ConnectionID    string     `json:"connectionId"`
	GroupID         *string    `json:"groupId"`
	Name            string     `json:"name"`
	Code            string     `json:"code"`
	NodeID          string     `json:"nodeId"`
	BrowseName      *string    `json:"browseName"`
	DisplayName     *string    `json:"displayName"`
	DataType        string     `json:"dataType"`
	Unit            *string    `json:"unit"`
	SamplingMS      int        `json:"samplingMs"`
	Deadband        *float64   `json:"deadband"`
	AccessLevel     string     `json:"accessLevel"`
	Description     *string    `json:"description"`
	SortOrder       int        `json:"sortOrder"`
	Status          string     `json:"status"`
	DataPointID     *string    `json:"datapointId"`
	DataPointPath   *string    `json:"datapointPath"`
	DataPointStatus *string    `json:"datapointStatus"`
	LastValue       any        `json:"lastValue,omitempty"`
	Quality         string     `json:"quality"`
	LastUpdatedAt   *time.Time `json:"lastUpdatedAt,omitempty"`
	CreatedAt       time.Time  `json:"createdAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
}

// OpcuaValidationIssue 表示 OPC UA 建模校验问题。
type OpcuaValidationIssue struct {
	Severity string  `json:"severity"`
	Code     string  `json:"code"`
	GroupID  *string `json:"groupId,omitempty"`
	NodeID   *string `json:"nodeId,omitempty"`
	NodeName string  `json:"nodeName,omitempty"`
	Message  string  `json:"message"`
}

// OpcuaValidationResult 表示 OPC UA 建模校验结果。
type OpcuaValidationResult struct {
	Valid  bool                   `json:"valid"`
	Issues []OpcuaValidationIssue `json:"issues"`
}

// CreateOpcuaNodeGroupInput 描述创建变量组输入。
type CreateOpcuaNodeGroupInput struct {
	ParentID    *string
	Name        string
	Description *string
	SortOrder   int
}

// UpdateOpcuaNodeGroupInput 描述更新变量组输入。
type UpdateOpcuaNodeGroupInput struct {
	ParentID    *string
	HasParentID bool
	Name        string
	Description *string
	SortOrder   int
}

// CreateOpcuaNodeInput 描述创建变量输入。
type CreateOpcuaNodeInput struct {
	GroupID     *string
	Name        string
	Code        string
	NodeID      string
	BrowseName  *string
	DisplayName *string
	DataType    string
	Unit        *string
	SamplingMS  *int
	Deadband    *float64
	AccessLevel string
	Description *string
	SortOrder   int
}

// UpdateOpcuaNodeInput 描述更新变量输入。
type UpdateOpcuaNodeInput struct {
	GroupID     *string
	HasGroupID  bool
	Name        string
	Code        string
	NodeID      string
	BrowseName  *string
	DisplayName *string
	DataType    string
	Unit        *string
	SamplingMS  *int
	Deadband    *float64
	AccessLevel string
	Description *string
	SortOrder   int
	Status      string
}

// ImportOpcuaNodeInput 描述批量导入的单个节点输入。
type ImportOpcuaNodeInput struct {
	Name        string   `json:"name"`
	Code        string   `json:"code"`
	NodeID      string   `json:"nodeId"`
	BrowseName  *string  `json:"browseName"`
	DisplayName *string  `json:"displayName"`
	DataType    string   `json:"dataType"`
	Unit        *string  `json:"unit"`
	SamplingMS  *int     `json:"samplingMs"`
	Deadband    *float64 `json:"deadband"`
	Description *string  `json:"description"`
}

// OpcuaPreviewResult 表示开发态辅助预览结果。
type OpcuaPreviewResult struct {
	Values      []OpcuaNode `json:"values"`
	Diagnostics []string    `json:"diagnostics"`
}

// OpcuaModelingService 承载 OPC UA 点位建模业务规则。
type OpcuaModelingService struct {
	repository  *repository.OpcuaModelingRepository
	connections *repository.ConnectionRepository
	datapoints  *repository.DataPointRepository
}

// NewOpcuaModelingService 创建 OPC UA 建模服务。
func NewOpcuaModelingService(repo *repository.OpcuaModelingRepository, connections *repository.ConnectionRepository, datapoints *repository.DataPointRepository) *OpcuaModelingService {
	return &OpcuaModelingService{repository: repo, connections: connections, datapoints: datapoints}
}

// ListGroups 返回变量组。
func (s *OpcuaModelingService) ListGroups(ctx context.Context, projectID, connectionID string) ([]OpcuaNodeGroup, error) {
	if err := s.validateProjectConnection(ctx, projectID, connectionID); err != nil {
		return nil, err
	}
	records, err := s.repository.ListGroups(ctx, projectID, connectionID)
	if err != nil {
		return nil, err
	}
	result := make([]OpcuaNodeGroup, 0, len(records))
	for _, record := range records {
		result = append(result, toOpcuaNodeGroup(record))
	}
	return result, nil
}

// CreateGroup 创建变量组。
func (s *OpcuaModelingService) CreateGroup(ctx context.Context, projectID, connectionID, userID string, input CreateOpcuaNodeGroupInput) (*OpcuaNodeGroup, error) {
	if err := s.validateProjectConnectionAndUser(ctx, projectID, connectionID, userID); err != nil {
		return nil, err
	}
	name, err := normalizeOpcuaRequiredText(input.Name, "变量组名称不能为空")
	if err != nil {
		return nil, err
	}
	record, err := s.repository.CreateGroup(ctx, repository.CreateOpcuaNodeGroupParams{
		ProjectID:    projectID,
		ConnectionID: connectionID,
		ParentID:     normalizeOptionalText(input.ParentID),
		Name:         name,
		Description:  normalizeOptionalText(input.Description),
		SortOrder:    input.SortOrder,
		UserID:       userID,
	})
	if err != nil {
		return nil, err
	}
	result := toOpcuaNodeGroup(*record)
	return &result, nil
}

// UpdateGroup 更新变量组。
func (s *OpcuaModelingService) UpdateGroup(ctx context.Context, projectID, connectionID, groupID, userID string, input UpdateOpcuaNodeGroupInput) (*OpcuaNodeGroup, error) {
	if err := s.validateProjectConnectionAndUser(ctx, projectID, connectionID, userID); err != nil {
		return nil, err
	}
	name, err := normalizeOpcuaRequiredText(input.Name, "变量组名称不能为空")
	if err != nil {
		return nil, err
	}
	record, err := s.repository.UpdateGroup(ctx, repository.UpdateOpcuaNodeGroupParams{
		ID:           groupID,
		ProjectID:    projectID,
		ConnectionID: connectionID,
		ParentID:     normalizeOptionalText(input.ParentID),
		HasParentID:  input.HasParentID,
		Name:         name,
		Description:  normalizeOptionalText(input.Description),
		SortOrder:    input.SortOrder,
		UserID:       userID,
	})
	if err != nil {
		return nil, err
	}
	result := toOpcuaNodeGroup(*record)
	return &result, nil
}

// DeleteGroup 删除变量组。
func (s *OpcuaModelingService) DeleteGroup(ctx context.Context, projectID, connectionID, groupID, userID string) error {
	if err := s.validateProjectConnectionAndUser(ctx, projectID, connectionID, userID); err != nil {
		return err
	}
	return s.repository.DeleteGroup(ctx, projectID, connectionID, groupID, userID)
}

// ListNodes 返回变量列表。
func (s *OpcuaModelingService) ListNodes(ctx context.Context, projectID, connectionID string, groupID *string) ([]OpcuaNode, error) {
	if err := s.validateProjectConnection(ctx, projectID, connectionID); err != nil {
		return nil, err
	}
	records, err := s.repository.ListNodes(ctx, projectID, connectionID, normalizeOptionalText(groupID))
	if err != nil {
		return nil, err
	}
	result := make([]OpcuaNode, 0, len(records))
	for _, record := range records {
		result = append(result, toOpcuaNode(record))
	}
	return result, nil
}

type OpcuaNodeListResult struct {
	Nodes      []OpcuaNode                `json:"list"`
	Pagination ProtocolModelingPagination `json:"pagination"`
}

// OpcuaNodeListFilter 是 OPC UA 变量列表的查询条件。
type OpcuaNodeListFilter struct {
	GroupID     *string
	Search      string
	QuickFilter string
	NodeID      string
	BrowseName  string
	AccessLevel string
	DataType    string
	SortBy      string
	SortOrder   string
	Page        int
	PageSize    int
}

// ListNodesPage 返回当前分组下的一页变量，分页条件只影响列表展示，不影响预览和校验等全量流程。
func (s *OpcuaModelingService) ListNodesPage(ctx context.Context, projectID, connectionID string, filter OpcuaNodeListFilter) (*OpcuaNodeListResult, error) {
	if err := s.validateProjectConnection(ctx, projectID, connectionID); err != nil {
		return nil, err
	}
	page, pageSize := normalizePageAndSize(filter.Page, filter.PageSize, 1, 100)
	records, total, err := s.repository.ListNodesPage(ctx, projectID, connectionID, repository.OpcuaNodeListFilter{
		GroupID:     normalizeOptionalText(filter.GroupID),
		Search:      strings.TrimSpace(filter.Search),
		QuickFilter: strings.TrimSpace(filter.QuickFilter),
		NodeID:      strings.TrimSpace(filter.NodeID),
		BrowseName:  strings.TrimSpace(filter.BrowseName),
		AccessLevel: strings.TrimSpace(filter.AccessLevel),
		DataType:    strings.TrimSpace(filter.DataType),
		SortBy:      strings.TrimSpace(filter.SortBy),
		SortOrder:   strings.TrimSpace(filter.SortOrder),
	}, page, pageSize)
	if err != nil {
		return nil, err
	}
	nodes := make([]OpcuaNode, 0, len(records))
	for _, record := range records {
		nodes = append(nodes, toOpcuaNode(record))
	}
	return &OpcuaNodeListResult{
		Nodes:      nodes,
		Pagination: newProtocolModelingPagination(page, pageSize, total),
	}, nil
}

// CreateNode 创建变量并同步数据点。
func (s *OpcuaModelingService) CreateNode(ctx context.Context, projectID, connectionID, userID string, input CreateOpcuaNodeInput) (*OpcuaNode, error) {
	if err := s.validateProjectConnectionAndUser(ctx, projectID, connectionID, userID); err != nil {
		return nil, err
	}
	params, err := s.normalizeCreateNodeInput(projectID, connectionID, userID, input)
	if err != nil {
		return nil, err
	}
	record, err := s.repository.CreateNode(ctx, params)
	if err != nil {
		return nil, err
	}
	if err := s.syncNodeDatapoint(ctx, *record, userID); err != nil {
		return nil, err
	}
	loaded, err := s.repository.GetNode(ctx, projectID, record.ID)
	if err != nil {
		return nil, err
	}
	result := toOpcuaNode(*loaded)
	return &result, nil
}

// BatchImportNodes 批量导入变量，变量名可由前端确认后传入。
func (s *OpcuaModelingService) BatchImportNodes(ctx context.Context, projectID, connectionID, userID string, groupID *string, nodes []ImportOpcuaNodeInput) ([]OpcuaNode, error) {
	if len(nodes) == 0 {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "导入变量不能为空")
	}
	result := make([]OpcuaNode, 0, len(nodes))
	for index, item := range nodes {
		name := strings.TrimSpace(item.Name)
		if name == "" {
			name = normalizeOpcuaNodeName(valueOrDefault(item.DisplayName, nil), valueOrDefault(item.BrowseName, nil), item.NodeID)
		}
		input := CreateOpcuaNodeInput{
			GroupID:     groupID,
			Name:        name,
			Code:        item.Code,
			NodeID:      item.NodeID,
			BrowseName:  item.BrowseName,
			DisplayName: item.DisplayName,
			DataType:    item.DataType,
			Unit:        item.Unit,
			SamplingMS:  item.SamplingMS,
			Deadband:    item.Deadband,
			AccessLevel: "Read",
			Description: item.Description,
			SortOrder:   index,
		}
		created, err := s.CreateNode(ctx, projectID, connectionID, userID, input)
		if err != nil {
			return nil, err
		}
		result = append(result, *created)
	}
	return result, nil
}

// UpdateNode 更新变量并同步数据点。
func (s *OpcuaModelingService) UpdateNode(ctx context.Context, projectID, connectionID, nodeID, userID string, input UpdateOpcuaNodeInput) (*OpcuaNode, error) {
	if err := s.validateProjectConnectionAndUser(ctx, projectID, connectionID, userID); err != nil {
		return nil, err
	}
	current, err := s.repository.GetNode(ctx, projectID, nodeID)
	if err != nil {
		return nil, err
	}
	if current.ConnectionID != connectionID {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "OPC UA 变量不存在")
	}
	params, err := s.normalizeUpdateNodeInput(projectID, userID, *current, input)
	if err != nil {
		return nil, err
	}
	record, err := s.repository.UpdateNode(ctx, params)
	if err != nil {
		return nil, err
	}
	if err := s.syncNodeDatapoint(ctx, *record, userID); err != nil {
		return nil, err
	}
	loaded, err := s.repository.GetNode(ctx, projectID, record.ID)
	if err != nil {
		return nil, err
	}
	result := toOpcuaNode(*loaded)
	return &result, nil
}

// DeleteNode 删除变量并将数据点标记失效。
func (s *OpcuaModelingService) DeleteNode(ctx context.Context, projectID, connectionID, nodeID, userID string) error {
	if err := s.validateProjectConnectionAndUser(ctx, projectID, connectionID, userID); err != nil {
		return err
	}
	current, err := s.repository.GetNode(ctx, projectID, nodeID)
	if err != nil {
		return err
	}
	if current.ConnectionID != connectionID {
		return apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "OPC UA 变量不存在")
	}
	if err := s.repository.DeleteNode(ctx, projectID, connectionID, nodeID); err != nil {
		return err
	}
	_, _ = s.datapoints.MarkInvalidBySource(ctx, projectID, "opcua.node", nodeID, stringPtr(userID))
	return nil
}

// ValidateModel 校验当前连接下的 OPC UA 建模结果。
func (s *OpcuaModelingService) ValidateModel(ctx context.Context, projectID, connectionID string) (*OpcuaValidationResult, error) {
	nodes, err := s.ListNodes(ctx, projectID, connectionID, nil)
	if err != nil {
		return nil, err
	}
	issues := make([]OpcuaValidationIssue, 0)
	codeSeen := map[string]string{}
	nodeSeen := map[string]string{}
	for _, node := range nodes {
		nodeID := node.ID
		if strings.TrimSpace(node.Name) == "" {
			issues = append(issues, opcuaNodeIssue("error", "name.empty", nodeID, node.Name, "变量名不能为空"))
		}
		if strings.TrimSpace(node.Code) == "" {
			issues = append(issues, opcuaNodeIssue("error", "code.empty", nodeID, node.Name, "变量标识符不能为空"))
		}
		if previous, ok := codeSeen[node.Code]; ok && previous != node.ID {
			issues = append(issues, opcuaNodeIssue("error", "code.duplicate", nodeID, node.Name, "变量标识符重复"))
		}
		codeSeen[node.Code] = node.ID
		if strings.TrimSpace(node.NodeID) == "" {
			issues = append(issues, opcuaNodeIssue("error", "nodeId.empty", nodeID, node.Name, "NodeId 不能为空"))
		}
		if previous, ok := nodeSeen[node.NodeID]; ok && previous != node.ID {
			issues = append(issues, opcuaNodeIssue("error", "nodeId.duplicate", nodeID, node.Name, "NodeId 重复"))
		}
		nodeSeen[node.NodeID] = node.ID
		if strings.TrimSpace(node.DataType) == "" {
			issues = append(issues, opcuaNodeIssue("error", "dataType.empty", nodeID, node.Name, "数据类型不能为空"))
		}
		if node.SamplingMS <= 0 {
			issues = append(issues, opcuaNodeIssue("error", "sampling.invalid", nodeID, node.Name, "采样周期必须大于 0"))
		}
		if node.DataPointPath == nil || strings.TrimSpace(*node.DataPointPath) == "" {
			issues = append(issues, opcuaNodeIssue("error", "datapoint.missing", nodeID, node.Name, "数据点未生成"))
		}
		if node.DataPointStatus != nil && *node.DataPointStatus == "invalid" {
			issues = append(issues, opcuaNodeIssue("warning", "datapoint.invalid", nodeID, node.Name, "数据点已失效"))
		}
	}
	return &OpcuaValidationResult{Valid: len(issues) == 0, Issues: issues}, nil
}

// PreviewNodes 返回开发态辅助预览占位结果。真实采集由运行态执行。
func (s *OpcuaModelingService) PreviewNodes(ctx context.Context, projectID, connectionID string, groupID *string) (*OpcuaPreviewResult, error) {
	nodes, err := s.ListNodes(ctx, projectID, connectionID, groupID)
	if err != nil {
		return nil, err
	}
	return &OpcuaPreviewResult{
		Values: nodes,
		Diagnostics: []string{
			"OPC UA 真实采集由运行态执行；当前预览返回已建模变量用于核对 NodeId、类型和数据点。",
		},
	}, nil
}

func (s *OpcuaModelingService) normalizeCreateNodeInput(projectID, connectionID, userID string, input CreateOpcuaNodeInput) (repository.CreateOpcuaNodeParams, error) {
	nodeID, err := normalizeOpcuaRequiredText(input.NodeID, "NodeId 不能为空")
	if err != nil {
		return repository.CreateOpcuaNodeParams{}, err
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		name = normalizeOpcuaNodeName(valueOrDefault(input.DisplayName, nil), valueOrDefault(input.BrowseName, nil), nodeID)
	}
	name, err = normalizeOpcuaRequiredText(name, "变量名不能为空")
	if err != nil {
		return repository.CreateOpcuaNodeParams{}, err
	}
	code := normalizeOpcuaNodeCode(input.Code, name, nodeID)
	dataType, err := normalizeOpcuaRequiredText(input.DataType, "数据类型不能为空")
	if err != nil {
		return repository.CreateOpcuaNodeParams{}, err
	}
	samplingMS, err := normalizePositiveInt(input.SamplingMS, 1000, "采样周期必须大于 0")
	if err != nil {
		return repository.CreateOpcuaNodeParams{}, err
	}
	accessLevel := normalizeOpcuaAccessLevel(input.AccessLevel)
	return repository.CreateOpcuaNodeParams{
		ProjectID:    projectID,
		ConnectionID: connectionID,
		GroupID:      normalizeOptionalText(input.GroupID),
		Name:         name,
		Code:         code,
		NodeID:       nodeID,
		BrowseName:   normalizeOptionalText(input.BrowseName),
		DisplayName:  normalizeOptionalText(input.DisplayName),
		DataType:     dataType,
		Unit:         normalizeOptionalText(input.Unit),
		SamplingMS:   samplingMS,
		Deadband:     input.Deadband,
		AccessLevel:  accessLevel,
		Description:  normalizeOptionalText(input.Description),
		SortOrder:    input.SortOrder,
		UserID:       userID,
	}, nil
}

func (s *OpcuaModelingService) normalizeUpdateNodeInput(projectID, userID string, current repository.OpcuaNodeRecord, input UpdateOpcuaNodeInput) (repository.UpdateOpcuaNodeParams, error) {
	nodeID := firstNonEmpty(input.NodeID, current.NodeID)
	name := firstNonEmpty(input.Name, current.Name)
	code := firstNonEmpty(input.Code, current.Code)
	dataType := firstNonEmpty(input.DataType, current.DataType)
	status := firstNonEmpty(input.Status, current.Status)
	if _, ok := allowedDataPointStatuses[status]; !ok {
		return repository.UpdateOpcuaNodeParams{}, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "变量状态不合法")
	}
	samplingInput := input.SamplingMS
	if samplingInput == nil {
		samplingInput = &current.SamplingMS
	}
	samplingMS, err := normalizePositiveInt(samplingInput, 1000, "采样周期必须大于 0")
	if err != nil {
		return repository.UpdateOpcuaNodeParams{}, err
	}
	return repository.UpdateOpcuaNodeParams{
		ID:           current.ID,
		ProjectID:    projectID,
		ConnectionID: current.ConnectionID,
		GroupID:      normalizeOptionalText(input.GroupID),
		HasGroupID:   input.HasGroupID,
		Name:         strings.TrimSpace(name),
		Code:         normalizeOpcuaNodeCode(code, name, nodeID),
		NodeID:       strings.TrimSpace(nodeID),
		BrowseName:   optionalTextOrCurrent(input.BrowseName, current.BrowseName),
		DisplayName:  optionalTextOrCurrent(input.DisplayName, current.DisplayName),
		DataType:     strings.TrimSpace(dataType),
		Unit:         optionalTextOrCurrent(input.Unit, current.Unit),
		SamplingMS:   samplingMS,
		Deadband:     optionalFloatOrCurrent(input.Deadband, current.Deadband),
		AccessLevel:  normalizeOpcuaAccessLevel(firstNonEmpty(input.AccessLevel, current.AccessLevel)),
		Description:  optionalTextOrCurrent(input.Description, current.Description),
		SortOrder:    input.SortOrder,
		Status:       status,
		UserID:       userID,
	}, nil
}

func (s *OpcuaModelingService) syncNodeDatapoint(ctx context.Context, node repository.OpcuaNodeRecord, userID string) error {
	connection, err := s.connections.GetByProjectAndID(ctx, node.ProjectID, node.ConnectionID)
	if err != nil {
		return err
	}
	basePath := "opcua." + normalizeDatapointSegment(connection.Name)
	groups, _ := s.repository.ListGroups(ctx, node.ProjectID, node.ConnectionID)
	groupPath := s.groupPathSegment(groups, node.GroupID)
	if groupPath != "" {
		basePath += "." + groupPath
	}
	basePath += "." + normalizeDatapointSegment(node.Code)
	path := s.allocateDataPointPath(ctx, node.ProjectID, basePath, node.ID, "opcua.node")
	sourceID := node.ID
	_, err = s.datapoints.UpsertBySource(ctx, repository.CreateDataPointParams{
		ProjectID:    node.ProjectID,
		UserID:       stringPtr(userID),
		Path:         path,
		Name:         node.Name,
		Description:  cloneOptionalString(node.Description),
		SourceType:   "opcua.node",
		SourceID:     &sourceID,
		SourceConfig: map[string]any{"connectionId": node.ConnectionID, "nodeId": node.NodeID, "samplingMs": node.SamplingMS, "deadband": node.Deadband, "accessLevel": node.AccessLevel},
		DataType:     normalizeOpcuaDataPointType(node.DataType),
		Unit:         cloneOptionalString(node.Unit),
		Tags:         []any{},
		RefreshMode:  "subscription",
		Status:       "active",
		DisplayOrder: &node.SortOrder,
	})
	return err
}

func (s *OpcuaModelingService) allocateDataPointPath(ctx context.Context, projectID, basePath, sourceID, sourceType string) string {
	candidate := strings.TrimSpace(basePath)
	if candidate == "" {
		candidate = "opcua.unnamed"
	}
	for index := 2; index < 100; index++ {
		record, err := s.datapoints.GetByProjectAndPath(ctx, projectID, candidate)
		if err != nil || record == nil {
			return candidate
		}
		if record.SourceType == sourceType && record.SourceID != nil && *record.SourceID == sourceID {
			return candidate
		}
		candidate = fmt.Sprintf("%s_%d", basePath, index)
	}
	return basePath + "_" + sourceID[:8]
}

func (s *OpcuaModelingService) groupPathSegment(groups []repository.OpcuaNodeGroupRecord, groupID *string) string {
	if groupID == nil || *groupID == "" {
		return ""
	}
	byID := make(map[string]repository.OpcuaNodeGroupRecord, len(groups))
	for _, group := range groups {
		byID[group.ID] = group
	}
	segments := make([]string, 0)
	currentID := *groupID
	visited := map[string]struct{}{}
	for currentID != "" {
		if _, ok := visited[currentID]; ok {
			break
		}
		visited[currentID] = struct{}{}
		group, ok := byID[currentID]
		if !ok {
			break
		}
		segments = append([]string{normalizeDatapointSegment(group.Name)}, segments...)
		if group.ParentID == nil {
			break
		}
		currentID = *group.ParentID
	}
	return strings.Join(segments, ".")
}

func (s *OpcuaModelingService) validateProjectConnection(ctx context.Context, projectID, connectionID string) error {
	if err := validateProjectID(projectID); err != nil {
		return err
	}
	connection, err := s.connections.GetByProjectAndID(ctx, projectID, connectionID)
	if err != nil {
		return err
	}
	if connection.Type != "opcua" {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "接入源不是 OPC UA 类型")
	}
	return nil
}

func (s *OpcuaModelingService) validateProjectConnectionAndUser(ctx context.Context, projectID, connectionID, userID string) error {
	if err := validateUserID(userID); err != nil {
		return err
	}
	return s.validateProjectConnection(ctx, projectID, connectionID)
}

func normalizeOpcuaRequiredText(value string, message string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, message)
	}
	return trimmed, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func optionalTextOrCurrent(next *string, current *string) *string {
	if next == nil {
		return cloneOptionalString(current)
	}
	return normalizeOptionalText(next)
}

func optionalFloatOrCurrent(next *float64, current *float64) *float64 {
	if next == nil {
		return cloneOptionalFloat64(current)
	}
	return cloneOptionalFloat64(next)
}

func normalizeOpcuaNodeName(displayName, browseName, nodeID string) string {
	if strings.TrimSpace(displayName) != "" {
		return strings.TrimSpace(displayName)
	}
	if strings.TrimSpace(browseName) != "" {
		return strings.TrimSpace(browseName)
	}
	parts := strings.FieldsFunc(nodeID, func(r rune) bool {
		return r == '.' || r == '/' || r == ';' || r == '=' || r == ':'
	})
	if len(parts) == 0 {
		return strings.TrimSpace(nodeID)
	}
	return parts[len(parts)-1]
}

func normalizeOpcuaNodeCode(code, name, nodeID string) string {
	source := firstNonEmpty(code, name, normalizeOpcuaNodeName("", "", nodeID))
	source = strings.TrimSpace(strings.ToLower(source))
	normalized := opcuaCodeSanitizer.ReplaceAllString(source, "_")
	normalized = strings.Trim(normalized, "_")
	if normalized == "" {
		return "node"
	}
	return normalized
}

func normalizeOpcuaAccessLevel(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "write":
		return "Write"
	case "readwrite", "read_write":
		return "ReadWrite"
	default:
		return "Read"
	}
}

func normalizeOpcuaDataPointType(dataType string) string {
	switch strings.ToLower(strings.TrimSpace(dataType)) {
	case "double", "float", "number", "int", "int16", "int32", "int64", "uint16", "uint32", "uint64":
		return "number"
	case "bool", "boolean":
		return "boolean"
	case "object", "extensionobject":
		return "object"
	case "array":
		return "array"
	default:
		return "string"
	}
}

func opcuaNodeIssue(severity, code, nodeID, nodeName, message string) OpcuaValidationIssue {
	return OpcuaValidationIssue{Severity: severity, Code: code, NodeID: &nodeID, NodeName: nodeName, Message: message}
}

func toOpcuaNodeGroup(record repository.OpcuaNodeGroupRecord) OpcuaNodeGroup {
	return OpcuaNodeGroup{
		ID:           record.ID,
		ProjectID:    record.ProjectID,
		ConnectionID: record.ConnectionID,
		ParentID:     cloneOptionalString(record.ParentID),
		Name:         record.Name,
		Description:  cloneOptionalString(record.Description),
		SortOrder:    record.SortOrder,
		CreatedAt:    record.CreatedAt,
		UpdatedAt:    record.UpdatedAt,
	}
}

func toOpcuaNode(record repository.OpcuaNodeRecord) OpcuaNode {
	return OpcuaNode{
		ID:              record.ID,
		ProjectID:       record.ProjectID,
		ConnectionID:    record.ConnectionID,
		GroupID:         cloneOptionalString(record.GroupID),
		Name:            record.Name,
		Code:            record.Code,
		NodeID:          record.NodeID,
		BrowseName:      cloneOptionalString(record.BrowseName),
		DisplayName:     cloneOptionalString(record.DisplayName),
		DataType:        record.DataType,
		Unit:            cloneOptionalString(record.Unit),
		SamplingMS:      record.SamplingMS,
		Deadband:        cloneOptionalFloat64(record.Deadband),
		AccessLevel:     record.AccessLevel,
		Description:     cloneOptionalString(record.Description),
		SortOrder:       record.SortOrder,
		Status:          record.Status,
		DataPointID:     cloneOptionalString(record.DataPointID),
		DataPointPath:   cloneOptionalString(record.DataPointPath),
		DataPointStatus: cloneOptionalString(record.DataPointStatus),
		Quality:         "unknown",
		CreatedAt:       record.CreatedAt,
		UpdatedAt:       record.UpdatedAt,
	}
}
