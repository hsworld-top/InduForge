package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/indu-forge/data_service/internal/repository"
)

var (
	s7CodeSanitizer       = regexp.MustCompile(`[^a-zA-Z0-9_]+`)
	s7DBAddressPattern    = regexp.MustCompile(`(?i)^DB(\d+)\.DB([XBWDL])(\d+)(?:\.(\d))?$`)
	s7AreaAddressPattern  = regexp.MustCompile(`(?i)^([MIQ])([BWD]?)(\d+)(?:\.(\d))?$`)
	s7SupportedDataTypes  = map[string]struct{}{"Bool": {}, "Byte": {}, "Word": {}, "DWord": {}, "Int": {}, "DInt": {}, "Real": {}, "DateTime": {}, "String": {}}
	s7SupportedAreaLookup = map[string]struct{}{"DB": {}, "M": {}, "I": {}, "Q": {}, "T": {}, "C": {}}
)

// S7Profile 表示前端工作台使用的 PLC 档案。
type S7Profile struct {
	ID                   string         `json:"id"`
	ProjectID            string         `json:"projectId"`
	ConnectionID         string         `json:"connectionId"`
	PlcFamily            string         `json:"plcFamily"`
	CommunicationMode    string         `json:"communicationMode"`
	Host                 string         `json:"host"`
	Port                 int            `json:"port"`
	Rack                 int            `json:"rack"`
	Slot                 int            `json:"slot"`
	LocalTSAP            *string        `json:"localTsap"`
	RemoteTSAP           *string        `json:"remoteTsap"`
	PollIntervalMS       int            `json:"pollIntervalMs"`
	ConnectTimeoutMS     int            `json:"connectTimeoutMs"`
	ReadTimeoutMS        int            `json:"readTimeoutMs"`
	PDUSize              *int           `json:"pduSize"`
	MaxReadBytes         *int           `json:"maxReadBytes"`
	MaxGapBytes          int            `json:"maxGapBytes"`
	MaxConcurrentReads   int            `json:"maxConcurrentReads"`
	ByteOrder            string         `json:"byteOrder"`
	WordOrder            string         `json:"wordOrder"`
	OptimizedBlockAccess bool           `json:"optimizedBlockAccess"`
	AllowAbsoluteAddress bool           `json:"allowAbsoluteAddress"`
	AllowSymbolAddress   bool           `json:"allowSymbolAddress"`
	SupportedAreas       []string       `json:"supportedAreas"`
	Options              map[string]any `json:"options"`
	Configured           bool           `json:"configured"`
}

// S7VariableGroup 表示 S7 变量组。
type S7VariableGroup struct {
	ID           string    `json:"id"`
	ProjectID    string    `json:"projectId"`
	ConnectionID string    `json:"connectionId"`
	ParentID     *string   `json:"parentId"`
	Name         string    `json:"name"`
	Code         string    `json:"code"`
	Description  *string   `json:"description"`
	SortOrder    int       `json:"sortOrder"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// S7Variable 表示 S7 变量地址表中的一行。
type S7Variable struct {
	ID                string     `json:"id"`
	ProjectID         string     `json:"projectId"`
	ConnectionID      string     `json:"connectionId"`
	GroupID           *string    `json:"groupId"`
	Name              string     `json:"name"`
	Code              string     `json:"code"`
	Description       *string    `json:"description"`
	Area              string     `json:"area"`
	DBNumber          *int       `json:"dbNumber"`
	ByteOffset        int        `json:"byteOffset"`
	BitOffset         *int       `json:"bitOffset"`
	AddressText       string     `json:"addressText"`
	NormalizedAddress string     `json:"normalizedAddress"`
	AddressType       string     `json:"addressType"`
	ReadLength        int        `json:"readLength"`
	DataType          string     `json:"dataType"`
	Length            *int       `json:"length"`
	ArrayLength       *int       `json:"arrayLength"`
	ByteOrder         string     `json:"byteOrder"`
	WordOrder         string     `json:"wordOrder"`
	Scale             float64    `json:"scale"`
	Offset            float64    `json:"offset"`
	Unit              *string    `json:"unit"`
	PollIntervalMS    int        `json:"pollIntervalMs"`
	LastValue         any        `json:"lastValue,omitempty"`
	Quality           string     `json:"quality"`
	LastUpdatedAt     *time.Time `json:"lastUpdatedAt,omitempty"`
	SortOrder         int        `json:"sortOrder"`
	Status            string     `json:"status"`
	DataPointID       *string    `json:"datapointId"`
	DataPointPath     *string    `json:"datapointPath"`
	DataPointStatus   *string    `json:"datapointStatus"`
	CreatedAt         time.Time  `json:"createdAt"`
	UpdatedAt         time.Time  `json:"updatedAt"`
}

type UpsertS7ProfileInput struct {
	PlcFamily            string
	CommunicationMode    string
	Host                 string
	Port                 *int
	Rack                 *int
	Slot                 *int
	LocalTSAP            *string
	RemoteTSAP           *string
	PollIntervalMS       *int
	ConnectTimeoutMS     *int
	ReadTimeoutMS        *int
	PDUSize              *int
	MaxReadBytes         *int
	MaxGapBytes          *int
	MaxConcurrentReads   *int
	ByteOrder            string
	WordOrder            string
	OptimizedBlockAccess bool
	AllowAbsoluteAddress bool
	AllowSymbolAddress   bool
	SupportedAreas       []string
	Options              map[string]any
}

type CreateS7VariableGroupInput struct {
	ParentID    *string
	Name        string
	Code        string
	Description *string
	SortOrder   int
}

type UpdateS7VariableGroupInput struct {
	ParentID    *string
	HasParentID bool
	Name        string
	Code        string
	Description *string
	SortOrder   int
}

type CreateS7VariableInput struct {
	GroupID        *string
	Name           string
	Code           string
	Description    *string
	AddressText    string
	DataType       string
	Length         *int
	ArrayLength    *int
	ByteOrder      string
	WordOrder      string
	Scale          *float64
	Offset         *float64
	Unit           *string
	PollIntervalMS *int
	QualityRule    map[string]any
	Metadata       map[string]any
	SortOrder      int
}

type UpdateS7VariableInput struct {
	GroupID        *string
	HasGroupID     bool
	Name           string
	Code           string
	Description    *string
	AddressText    string
	DataType       string
	Length         *int
	ArrayLength    *int
	ByteOrder      string
	WordOrder      string
	Scale          *float64
	Offset         *float64
	Unit           *string
	PollIntervalMS *int
	QualityRule    map[string]any
	Metadata       map[string]any
	SortOrder      int
	Status         string
}

type ImportS7VariableInput struct {
	Name           string         `json:"name"`
	Code           string         `json:"code"`
	GroupID        *string        `json:"groupId"`
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
	Description    *string        `json:"description"`
	Metadata       map[string]any `json:"metadata"`
}

type S7ParsedAddress struct {
	Area              string
	DBNumber          *int
	ByteOffset        int
	BitOffset         *int
	AddressType       string
	NormalizedAddress string
}

type S7ReadPlan struct {
	ID             string   `json:"id"`
	ConnectionID   string   `json:"connectionId"`
	Area           string   `json:"area"`
	DBNumber       *int     `json:"dbNumber,omitempty"`
	StartByte      int      `json:"startByte"`
	EndByte        int      `json:"endByte"`
	ReadLength     int      `json:"readLength"`
	PollIntervalMS int      `json:"pollIntervalMs"`
	VariableIDs    []string `json:"variableIds"`
	VariableCount  int      `json:"variableCount"`
	MaxGapBytes    int      `json:"maxGapBytes"`
	ReadMode       string   `json:"readMode"`
	DisplayRange   string   `json:"displayRange"`
	ReadsPerSecond float64  `json:"readsPerSecond"`
}

type S7ReadPlanEstimate struct {
	VariableCount        int          `json:"variableCount"`
	BlockCount           int          `json:"blockCount"`
	TotalReadBytes       int          `json:"totalReadBytes"`
	ReadsPerSecond       float64      `json:"readsPerSecond"`
	EstimatedCycleMS     int          `json:"estimatedCycleMs"`
	LargestBlockBytes    int          `json:"largestBlockBytes"`
	FragmentedGroupCount int          `json:"fragmentedGroupCount"`
	Plans                []S7ReadPlan `json:"plans"`
	Diagnostics          []string     `json:"diagnostics"`
}

type S7ValidationIssue struct {
	Severity     string  `json:"severity"`
	Code         string  `json:"code"`
	GroupID      *string `json:"groupId,omitempty"`
	VariableID   *string `json:"variableId,omitempty"`
	VariableName string  `json:"variableName,omitempty"`
	Message      string  `json:"message"`
}

type S7ValidationResult struct {
	Valid  bool                `json:"valid"`
	Issues []S7ValidationIssue `json:"issues"`
}

type S7PreviewResult struct {
	Values      []S7Variable `json:"values"`
	Diagnostics []string     `json:"diagnostics"`
}

// ProtocolDevS7Variable 是 S7 开发态会话读取需要的最小变量投影。
type ProtocolDevS7Variable struct {
	ID                string
	Code              string
	Name              string
	GroupID           *string
	Area              string
	DBNumber          *int
	ByteOffset        int
	BitOffset         *int
	AddressText       string
	NormalizedAddress string
	ReadLength        int
	DataType          string
	Scale             float64
	Offset            float64
	Unit              *string
}

type S7ModelingService struct {
	repository  *repository.S7ModelingRepository
	connections *repository.ConnectionRepository
	datapoints  *repository.DataPointRepository
}

func NewS7ModelingService(repo *repository.S7ModelingRepository, connections *repository.ConnectionRepository, datapoints *repository.DataPointRepository) *S7ModelingService {
	return &S7ModelingService{repository: repo, connections: connections, datapoints: datapoints}
}

func (s *S7ModelingService) GetProfile(ctx context.Context, projectID, connectionID string) (*S7Profile, error) {
	connection, err := s.validateAndLoadConnection(ctx, projectID, connectionID)
	if err != nil {
		return nil, err
	}
	record, err := s.repository.GetProfile(ctx, projectID, connectionID)
	if err != nil {
		return nil, err
	}
	if record == nil {
		profile := defaultS7ProfileForFamily("S7 Compatible", *connection)
		profile.ProjectID = projectID
		profile.ConnectionID = connectionID
		return &profile, nil
	}
	result := toS7Profile(*record)
	return &result, nil
}

func (s *S7ModelingService) UpsertProfile(ctx context.Context, projectID, connectionID, userID string, input UpsertS7ProfileInput) (*S7Profile, error) {
	connection, err := s.validateAndLoadConnectionWithUser(ctx, projectID, connectionID, userID)
	if err != nil {
		return nil, err
	}
	params, err := s.normalizeProfileInput(*connection, userID, input)
	if err != nil {
		return nil, err
	}
	record, err := s.repository.UpsertProfile(ctx, params)
	if err != nil {
		return nil, err
	}
	result := toS7Profile(*record)
	return &result, nil
}

func (s *S7ModelingService) ListGroups(ctx context.Context, projectID, connectionID string) ([]S7VariableGroup, error) {
	if _, err := s.validateAndLoadConnection(ctx, projectID, connectionID); err != nil {
		return nil, err
	}
	records, err := s.repository.ListGroups(ctx, projectID, connectionID)
	if err != nil {
		return nil, err
	}
	result := make([]S7VariableGroup, 0, len(records))
	for _, record := range records {
		result = append(result, toS7VariableGroup(record))
	}
	return result, nil
}

func (s *S7ModelingService) CreateGroup(ctx context.Context, projectID, connectionID, userID string, input CreateS7VariableGroupInput) (*S7VariableGroup, error) {
	if _, err := s.validateAndLoadConnectionWithUser(ctx, projectID, connectionID, userID); err != nil {
		return nil, err
	}
	name, err := normalizeS7RequiredText(input.Name, "变量组名称不能为空")
	if err != nil {
		return nil, err
	}
	record, err := s.repository.CreateGroup(ctx, repository.CreateS7VariableGroupParams{
		ProjectID:    projectID,
		ConnectionID: connectionID,
		ParentID:     normalizeOptionalText(input.ParentID),
		Name:         name,
		Code:         normalizeS7Code(input.Code, name, "group"),
		Description:  normalizeOptionalText(input.Description),
		SortOrder:    input.SortOrder,
		UserID:       userID,
	})
	if err != nil {
		return nil, err
	}
	result := toS7VariableGroup(*record)
	return &result, nil
}

func (s *S7ModelingService) UpdateGroup(ctx context.Context, projectID, connectionID, groupID, userID string, input UpdateS7VariableGroupInput) (*S7VariableGroup, error) {
	if _, err := s.validateAndLoadConnectionWithUser(ctx, projectID, connectionID, userID); err != nil {
		return nil, err
	}
	name, err := normalizeS7RequiredText(input.Name, "变量组名称不能为空")
	if err != nil {
		return nil, err
	}
	record, err := s.repository.UpdateGroup(ctx, repository.UpdateS7VariableGroupParams{
		ID:           groupID,
		ProjectID:    projectID,
		ConnectionID: connectionID,
		ParentID:     normalizeOptionalText(input.ParentID),
		HasParentID:  input.HasParentID,
		Name:         name,
		Code:         normalizeS7Code(input.Code, name, "group"),
		Description:  normalizeOptionalText(input.Description),
		SortOrder:    input.SortOrder,
		UserID:       userID,
	})
	if err != nil {
		return nil, err
	}
	result := toS7VariableGroup(*record)
	return &result, nil
}

func (s *S7ModelingService) DeleteGroup(ctx context.Context, projectID, connectionID, groupID, userID string) error {
	if _, err := s.validateAndLoadConnectionWithUser(ctx, projectID, connectionID, userID); err != nil {
		return err
	}
	return s.repository.DeleteGroup(ctx, projectID, connectionID, groupID, userID)
}

func (s *S7ModelingService) ListVariables(ctx context.Context, projectID, connectionID string, groupID *string) ([]S7Variable, error) {
	if _, err := s.validateAndLoadConnection(ctx, projectID, connectionID); err != nil {
		return nil, err
	}
	records, err := s.repository.ListVariables(ctx, projectID, connectionID, normalizeOptionalText(groupID))
	if err != nil {
		return nil, err
	}
	result := make([]S7Variable, 0, len(records))
	for _, record := range records {
		result = append(result, toS7Variable(record))
	}
	return result, nil
}

type S7VariableListResult struct {
	Variables  []S7Variable               `json:"list"`
	Pagination ProtocolModelingPagination `json:"pagination"`
}

// ListVariablesPage 返回当前分组下的一页变量，分页条件只影响列表展示，不影响预览和校验等全量流程。
func (s *S7ModelingService) ListVariablesPage(ctx context.Context, projectID, connectionID string, groupID *string, page, pageSize int) (*S7VariableListResult, error) {
	if _, err := s.validateAndLoadConnection(ctx, projectID, connectionID); err != nil {
		return nil, err
	}
	page, pageSize = normalizePageAndSize(page, pageSize, 1, 100)
	records, total, err := s.repository.ListVariablesPage(ctx, projectID, connectionID, normalizeOptionalText(groupID), page, pageSize)
	if err != nil {
		return nil, err
	}
	variables := make([]S7Variable, 0, len(records))
	for _, record := range records {
		variables = append(variables, toS7Variable(record))
	}
	return &S7VariableListResult{
		Variables:  variables,
		Pagination: newProtocolModelingPagination(page, pageSize, total),
	}, nil
}

func (s *S7ModelingService) CreateVariable(ctx context.Context, projectID, connectionID, userID string, input CreateS7VariableInput) (*S7Variable, error) {
	if _, err := s.validateAndLoadConnectionWithUser(ctx, projectID, connectionID, userID); err != nil {
		return nil, err
	}
	params, err := s.normalizeCreateVariableInput(ctx, projectID, connectionID, userID, input)
	if err != nil {
		return nil, err
	}
	record, err := s.repository.CreateVariable(ctx, params)
	if err != nil {
		return nil, err
	}
	if err := s.syncVariableDatapoint(ctx, *record, userID); err != nil {
		return nil, err
	}
	loaded, err := s.repository.GetVariable(ctx, projectID, connectionID, record.ID)
	if err != nil {
		return nil, err
	}
	result := toS7Variable(*loaded)
	return &result, nil
}

func (s *S7ModelingService) BatchImportVariables(ctx context.Context, projectID, connectionID, userID string, groupID *string, variables []ImportS7VariableInput) ([]S7Variable, error) {
	if len(variables) == 0 {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "导入变量不能为空")
	}
	result := make([]S7Variable, 0, len(variables))
	for index, item := range variables {
		targetGroupID := groupID
		if item.GroupID != nil {
			targetGroupID = item.GroupID
		}
		name := strings.TrimSpace(item.Name)
		if name == "" {
			name = strings.TrimSpace(item.AddressText)
		}
		created, err := s.CreateVariable(ctx, projectID, connectionID, userID, CreateS7VariableInput{
			GroupID:        targetGroupID,
			Name:           name,
			Code:           item.Code,
			Description:    item.Description,
			AddressText:    item.AddressText,
			DataType:       item.DataType,
			Length:         item.Length,
			ArrayLength:    item.ArrayLength,
			ByteOrder:      item.ByteOrder,
			WordOrder:      item.WordOrder,
			Scale:          item.Scale,
			Offset:         item.Offset,
			Unit:           item.Unit,
			PollIntervalMS: item.PollIntervalMS,
			Metadata:       item.Metadata,
			SortOrder:      index,
		})
		if err != nil {
			return nil, err
		}
		result = append(result, *created)
	}
	return result, nil
}

func (s *S7ModelingService) UpdateVariable(ctx context.Context, projectID, connectionID, variableID, userID string, input UpdateS7VariableInput) (*S7Variable, error) {
	if _, err := s.validateAndLoadConnectionWithUser(ctx, projectID, connectionID, userID); err != nil {
		return nil, err
	}
	current, err := s.repository.GetVariable(ctx, projectID, connectionID, variableID)
	if err != nil {
		return nil, err
	}
	params, err := s.normalizeUpdateVariableInput(projectID, userID, *current, input)
	if err != nil {
		return nil, err
	}
	record, err := s.repository.UpdateVariable(ctx, params)
	if err != nil {
		return nil, err
	}
	if err := s.syncVariableDatapoint(ctx, *record, userID); err != nil {
		return nil, err
	}
	loaded, err := s.repository.GetVariable(ctx, projectID, connectionID, record.ID)
	if err != nil {
		return nil, err
	}
	result := toS7Variable(*loaded)
	return &result, nil
}

func (s *S7ModelingService) DeleteVariable(ctx context.Context, projectID, connectionID, variableID, userID string) error {
	if _, err := s.validateAndLoadConnectionWithUser(ctx, projectID, connectionID, userID); err != nil {
		return err
	}
	if err := s.repository.DeleteVariable(ctx, projectID, connectionID, variableID); err != nil {
		return err
	}
	_, _ = s.datapoints.MarkInvalidBySource(ctx, projectID, "s7.variable", variableID, stringPtr(userID))
	return nil
}

func (s *S7ModelingService) UpdateVariableLastValue(ctx context.Context, projectID, connectionID, variableID, userID string, value any, quality string) error {
	return s.repository.UpdateVariableLastValue(ctx, repository.UpdateS7VariableLastValueParams{
		ProjectID:    projectID,
		ConnectionID: connectionID,
		VariableID:   variableID,
		LastValue:    value,
		Quality:      firstNonEmpty(quality, "good"),
		UserID:       userID,
	})
}

func (s *S7ModelingService) ValidateModel(ctx context.Context, projectID, connectionID string) (*S7ValidationResult, error) {
	profile, err := s.GetProfile(ctx, projectID, connectionID)
	if err != nil {
		return nil, err
	}
	variables, err := s.ListVariables(ctx, projectID, connectionID, nil)
	if err != nil {
		return nil, err
	}
	groups, err := s.ListGroups(ctx, projectID, connectionID)
	if err != nil {
		return nil, err
	}

	issues := make([]S7ValidationIssue, 0)
	if !profile.Configured {
		issues = append(issues, s7Issue("error", "profile.missing", "", "", "PLC 类型和连接能力尚未确认"))
	}
	groupIDs := map[string]struct{}{}
	for _, group := range groups {
		groupIDs[group.ID] = struct{}{}
	}
	supportedAreas := stringSet(profile.SupportedAreas)
	codeSeen := map[string]string{}
	type rangeItem struct {
		variable S7Variable
		start    int
		end      int
	}
	ranges := map[string][]rangeItem{}
	for _, variable := range variables {
		if strings.TrimSpace(variable.Name) == "" {
			issues = append(issues, s7VariableIssue("error", "name.empty", variable, "变量名不能为空"))
		}
		if strings.TrimSpace(variable.Code) == "" {
			issues = append(issues, s7VariableIssue("error", "code.empty", variable, "变量标识符不能为空"))
		}
		if previous, ok := codeSeen[variable.Code]; ok && previous != variable.ID {
			issues = append(issues, s7VariableIssue("error", "code.duplicate", variable, "变量标识符重复"))
		}
		codeSeen[variable.Code] = variable.ID
		if variable.GroupID != nil {
			if _, ok := groupIDs[*variable.GroupID]; !ok {
				issues = append(issues, s7VariableIssue("error", "group.missing", variable, "变量组不存在"))
			}
		}
		parsed, parseErr := ParseS7Address(variable.AddressText, variable.DataType)
		if parseErr != nil {
			issues = append(issues, s7VariableIssue("error", "address.invalid", variable, parseErr.Error()))
		} else if parsed.NormalizedAddress != variable.NormalizedAddress {
			issues = append(issues, s7VariableIssue("warning", "address.normalized", variable, "标准地址与原始地址不一致，建议重新保存"))
		}
		if variable.Area == "DB" && variable.DBNumber == nil {
			issues = append(issues, s7VariableIssue("error", "address.dbNumberMissing", variable, "DB 区地址必须包含 DB 号"))
		}
		if strings.EqualFold(variable.DataType, "Bool") && variable.BitOffset == nil {
			issues = append(issues, s7VariableIssue("error", "address.bitMissing", variable, "Bool 变量必须包含 bit 偏移"))
		}
		if strings.EqualFold(variable.DataType, "String") && variable.Length == nil {
			issues = append(issues, s7VariableIssue("error", "type.stringLengthMissing", variable, "String 变量必须填写长度"))
		}
		if variable.ByteOffset < 0 {
			issues = append(issues, s7VariableIssue("error", "address.byteOffset", variable, "字节偏移不能小于 0"))
		}
		if variable.PollIntervalMS <= 0 {
			issues = append(issues, s7VariableIssue("error", "poll.invalid", variable, "采集周期必须大于 0"))
		}
		if _, ok := supportedAreas[variable.Area]; !ok {
			issues = append(issues, s7VariableIssue("error", "profile.areaUnsupported", variable, "PLC 档案不支持该地址区"))
		}
		if !s7AddressTypeMatchesDataType(variable.AddressType, variable.DataType) {
			issues = append(issues, s7VariableIssue("error", "type.addressMismatch", variable, "数据类型与地址宽度不匹配"))
		}
		if variable.DataPointPath == nil || strings.TrimSpace(*variable.DataPointPath) == "" {
			issues = append(issues, s7VariableIssue("error", "datapoint.missing", variable, "数据点未生成"))
		}
		if variable.DataPointStatus != nil && *variable.DataPointStatus == "invalid" {
			issues = append(issues, s7VariableIssue("warning", "datapoint.invalid", variable, "数据点已失效"))
		}
		if profile.OptimizedBlockAccess && variable.Area == "DB" {
			issues = append(issues, s7VariableIssue("warning", "profile.optimizedDB", variable, "S7-1200/1500 开启优化 DB 访问时，绝对地址可能无法读取"))
		}
		key := fmt.Sprintf("%s|%s", variable.Area, optionalIntKey(variable.DBNumber))
		ranges[key] = append(ranges[key], rangeItem{variable: variable, start: variable.ByteOffset, end: variable.ByteOffset + variable.ReadLength - 1})
	}
	for _, items := range ranges {
		sort.Slice(items, func(i, j int) bool { return items[i].start < items[j].start })
		for index := 1; index < len(items); index++ {
			if items[index].start <= items[index-1].end {
				issues = append(issues, s7VariableIssue("warning", "address.overlap", items[index].variable, "地址范围与另一个变量重叠"))
			}
		}
	}
	estimate := BuildS7ReadPlanEstimate(*profile, variables)
	if estimate.FragmentedGroupCount > 8 {
		issues = append(issues, s7Issue("warning", "readPlan.fragmented", "", "", "读取块碎片较多，建议整理地址连续性或调整采集周期"))
	}
	return &S7ValidationResult{Valid: len(issues) == 0, Issues: issues}, nil
}

func (s *S7ModelingService) PreviewVariables(ctx context.Context, projectID, connectionID string, groupID *string) (*S7PreviewResult, error) {
	variables, err := s.ListVariables(ctx, projectID, connectionID, groupID)
	if err != nil {
		return nil, err
	}
	profile, err := s.GetProfile(ctx, projectID, connectionID)
	if err != nil {
		return nil, err
	}
	estimate := BuildS7ReadPlanEstimate(*profile, variables)
	return &S7PreviewResult{
		Values: variables,
		Diagnostics: []string{
			fmt.Sprintf("S7 真实采集由运行态执行；当前预览返回 %d 个已建模变量用于核对地址、类型和数据点。", len(variables)),
			fmt.Sprintf("当前范围预计合并为 %d 个读取块，预计 %.2f reads/s。", estimate.BlockCount, estimate.ReadsPerSecond),
		},
	}, nil
}

func (s *S7ModelingService) EstimateReadPlans(ctx context.Context, projectID, connectionID string, groupID *string) (*S7ReadPlanEstimate, error) {
	profile, err := s.GetProfile(ctx, projectID, connectionID)
	if err != nil {
		return nil, err
	}
	variables, err := s.ListVariables(ctx, projectID, connectionID, groupID)
	if err != nil {
		return nil, err
	}
	estimate := BuildS7ReadPlanEstimate(*profile, variables)
	return &estimate, nil
}

func (s *S7ModelingService) ListDevSessionS7Variables(ctx context.Context, projectID, connectionID string, groupID *string) ([]ProtocolDevS7Variable, error) {
	variables, err := s.ListVariables(ctx, projectID, connectionID, groupID)
	if err != nil {
		return nil, err
	}
	result := make([]ProtocolDevS7Variable, 0, len(variables))
	for _, variable := range variables {
		result = append(result, ProtocolDevS7Variable{
			ID:                variable.ID,
			Code:              variable.Code,
			Name:              variable.Name,
			GroupID:           variable.GroupID,
			Area:              variable.Area,
			DBNumber:          variable.DBNumber,
			ByteOffset:        variable.ByteOffset,
			BitOffset:         variable.BitOffset,
			AddressText:       variable.AddressText,
			NormalizedAddress: variable.NormalizedAddress,
			ReadLength:        variable.ReadLength,
			DataType:          variable.DataType,
			Scale:             variable.Scale,
			Offset:            variable.Offset,
			Unit:              variable.Unit,
		})
	}
	return result, nil
}

func (s *S7ModelingService) EstimateDevSessionS7ReadPlan(ctx context.Context, projectID, connectionID string, groupID *string) (*S7ReadPlanEstimate, error) {
	return s.EstimateReadPlans(ctx, projectID, connectionID, groupID)
}

func (s *S7ModelingService) normalizeProfileInput(connection repository.ConnectionRecord, userID string, input UpsertS7ProfileInput) (repository.UpsertS7ProfileParams, error) {
	family := firstNonEmpty(input.PlcFamily, "S7 Compatible")
	defaults := defaultS7ProfileForFamily(family, connection)
	host := firstNonEmpty(input.Host, defaults.Host)
	if host == "" {
		return repository.UpsertS7ProfileParams{}, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "PLC 地址不能为空")
	}
	supportedAreas := normalizeS7SupportedAreas(input.SupportedAreas)
	if len(supportedAreas) == 0 {
		supportedAreas = defaults.SupportedAreas
	}
	return repository.UpsertS7ProfileParams{
		ProjectID:            connection.ProjectID,
		ConnectionID:         connection.ID,
		PlcFamily:            family,
		CommunicationMode:    firstNonEmpty(input.CommunicationMode, defaults.CommunicationMode),
		Host:                 host,
		Port:                 intOrDefault(input.Port, defaults.Port),
		Rack:                 intOrDefault(input.Rack, defaults.Rack),
		Slot:                 intOrDefault(input.Slot, defaults.Slot),
		LocalTSAP:            normalizeOptionalText(input.LocalTSAP),
		RemoteTSAP:           normalizeOptionalText(input.RemoteTSAP),
		PollIntervalMS:       intOrDefault(input.PollIntervalMS, defaults.PollIntervalMS),
		ConnectTimeoutMS:     intOrDefault(input.ConnectTimeoutMS, defaults.ConnectTimeoutMS),
		ReadTimeoutMS:        intOrDefault(input.ReadTimeoutMS, defaults.ReadTimeoutMS),
		PDUSize:              input.PDUSize,
		MaxReadBytes:         input.MaxReadBytes,
		MaxGapBytes:          intOrDefault(input.MaxGapBytes, defaults.MaxGapBytes),
		MaxConcurrentReads:   intOrDefault(input.MaxConcurrentReads, defaults.MaxConcurrentReads),
		ByteOrder:            normalizeS7Endian(firstNonEmpty(input.ByteOrder, defaults.ByteOrder)),
		WordOrder:            normalizeS7Endian(firstNonEmpty(input.WordOrder, defaults.WordOrder)),
		OptimizedBlockAccess: input.OptimizedBlockAccess,
		AllowAbsoluteAddress: input.AllowAbsoluteAddress || defaults.AllowAbsoluteAddress,
		AllowSymbolAddress:   input.AllowSymbolAddress,
		SupportedAreas:       stringsToAnySlice(supportedAreas),
		Options:              normalizeMap(input.Options),
		UserID:               userID,
	}, nil
}

func (s *S7ModelingService) normalizeCreateVariableInput(ctx context.Context, projectID, connectionID, userID string, input CreateS7VariableInput) (repository.CreateS7VariableParams, error) {
	name, err := normalizeS7RequiredText(input.Name, "变量名不能为空")
	if err != nil {
		return repository.CreateS7VariableParams{}, err
	}
	dataType := normalizeS7DataType(input.DataType)
	parsed, err := ParseS7Address(input.AddressText, dataType)
	if err != nil {
		return repository.CreateS7VariableParams{}, err
	}
	readLength, err := s7ReadLengthForDataType(dataType, input.Length, input.ArrayLength)
	if err != nil {
		return repository.CreateS7VariableParams{}, err
	}
	return repository.CreateS7VariableParams{
		ProjectID:         projectID,
		ConnectionID:      connectionID,
		GroupID:           normalizeOptionalText(input.GroupID),
		Name:              name,
		Code:              normalizeS7Code(input.Code, name, parsed.NormalizedAddress),
		Description:       normalizeOptionalText(input.Description),
		Area:              parsed.Area,
		DBNumber:          parsed.DBNumber,
		ByteOffset:        parsed.ByteOffset,
		BitOffset:         parsed.BitOffset,
		AddressText:       strings.TrimSpace(input.AddressText),
		NormalizedAddress: parsed.NormalizedAddress,
		AddressType:       parsed.AddressType,
		ReadLength:        readLength,
		DataType:          dataType,
		Length:            input.Length,
		ArrayLength:       input.ArrayLength,
		ByteOrder:         normalizeS7Endian(input.ByteOrder),
		WordOrder:         normalizeS7Endian(input.WordOrder),
		Scale:             floatOrDefault(input.Scale, 1),
		Offset:            floatOrDefault(input.Offset, 0),
		Unit:              normalizeOptionalText(input.Unit),
		PollIntervalMS:    intOrDefault(input.PollIntervalMS, 1000),
		QualityRule:       normalizeMap(input.QualityRule),
		Metadata:          normalizeMap(input.Metadata),
		SortOrder:         input.SortOrder,
		UserID:            userID,
	}, nil
}

func (s *S7ModelingService) normalizeUpdateVariableInput(projectID, userID string, current repository.S7VariableRecord, input UpdateS7VariableInput) (repository.UpdateS7VariableParams, error) {
	name := firstNonEmpty(input.Name, current.Name)
	dataType := normalizeS7DataType(firstNonEmpty(input.DataType, current.DataType))
	addressText := firstNonEmpty(input.AddressText, current.AddressText)
	parsed, err := ParseS7Address(addressText, dataType)
	if err != nil {
		return repository.UpdateS7VariableParams{}, err
	}
	length := optionalIntOrCurrent(input.Length, current.Length)
	arrayLength := optionalIntOrCurrent(input.ArrayLength, current.ArrayLength)
	readLength, err := s7ReadLengthForDataType(dataType, length, arrayLength)
	if err != nil {
		return repository.UpdateS7VariableParams{}, err
	}
	status := firstNonEmpty(input.Status, current.Status)
	if _, ok := allowedDataPointStatuses[status]; !ok {
		return repository.UpdateS7VariableParams{}, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "变量状态不合法")
	}
	return repository.UpdateS7VariableParams{
		ID:                current.ID,
		ProjectID:         projectID,
		ConnectionID:      current.ConnectionID,
		GroupID:           normalizeOptionalText(input.GroupID),
		HasGroupID:        input.HasGroupID,
		Name:              strings.TrimSpace(name),
		Code:              normalizeS7Code(firstNonEmpty(input.Code, current.Code), name, parsed.NormalizedAddress),
		Description:       optionalTextOrCurrent(input.Description, current.Description),
		Area:              parsed.Area,
		DBNumber:          parsed.DBNumber,
		ByteOffset:        parsed.ByteOffset,
		BitOffset:         parsed.BitOffset,
		AddressText:       strings.TrimSpace(addressText),
		NormalizedAddress: parsed.NormalizedAddress,
		AddressType:       parsed.AddressType,
		ReadLength:        readLength,
		DataType:          dataType,
		Length:            length,
		ArrayLength:       arrayLength,
		ByteOrder:         normalizeS7Endian(firstNonEmpty(input.ByteOrder, current.ByteOrder)),
		WordOrder:         normalizeS7Endian(firstNonEmpty(input.WordOrder, current.WordOrder)),
		Scale:             floatOrDefault(input.Scale, current.Scale),
		Offset:            floatOrDefault(input.Offset, current.Offset),
		Unit:              optionalTextOrCurrent(input.Unit, current.Unit),
		PollIntervalMS:    intOrDefault(input.PollIntervalMS, current.PollIntervalMS),
		QualityRule:       normalizeMap(input.QualityRule),
		Metadata:          normalizeMap(input.Metadata),
		SortOrder:         input.SortOrder,
		Status:            status,
		UserID:            userID,
	}, nil
}

func (s *S7ModelingService) syncVariableDatapoint(ctx context.Context, variable repository.S7VariableRecord, userID string) error {
	connection, err := s.connections.GetByProjectAndID(ctx, variable.ProjectID, variable.ConnectionID)
	if err != nil {
		return err
	}
	basePath := "s7." + normalizeDatapointSegment(connection.Name)
	groups, _ := s.repository.ListGroups(ctx, variable.ProjectID, variable.ConnectionID)
	groupPath := s.groupPathSegment(groups, variable.GroupID)
	if groupPath != "" {
		basePath += "." + groupPath
	}
	basePath += "." + normalizeDatapointSegment(variable.Code)
	path := s.allocateDataPointPath(ctx, variable.ProjectID, basePath, variable.ID, "s7.variable")
	sourceID := variable.ID
	refreshInterval := variable.PollIntervalMS
	_, err = s.datapoints.UpsertBySource(ctx, repository.CreateDataPointParams{
		ProjectID:   variable.ProjectID,
		UserID:      stringPtr(userID),
		Path:        path,
		Name:        variable.Name,
		Description: cloneOptionalString(variable.Description),
		SourceType:  "s7.variable",
		SourceID:    &sourceID,
		SourceConfig: map[string]any{
			"connectionId":       variable.ConnectionID,
			"area":               variable.Area,
			"dbNumber":           variable.DBNumber,
			"byteOffset":         variable.ByteOffset,
			"bitOffset":          variable.BitOffset,
			"addressText":        variable.AddressText,
			"normalizedAddress":  variable.NormalizedAddress,
			"readLength":         variable.ReadLength,
			"dataType":           variable.DataType,
			"byteOrder":          variable.ByteOrder,
			"wordOrder":          variable.WordOrder,
			"scale":              variable.Scale,
			"offset":             variable.Offset,
			"pollIntervalMs":     variable.PollIntervalMS,
			"runtimeOwnerHint":   "readPlan",
			"optimizedDBWarning": variable.Area == "DB",
		},
		DataType:          normalizeS7DataPointType(variable.DataType),
		Unit:              cloneOptionalString(variable.Unit),
		Tags:              []any{},
		RefreshMode:       "auto",
		RefreshIntervalMS: &refreshInterval,
		Status:            "active",
		DisplayOrder:      &variable.SortOrder,
	})
	return err
}

func (s *S7ModelingService) allocateDataPointPath(ctx context.Context, projectID, basePath, sourceID, sourceType string) string {
	candidate := strings.TrimSpace(basePath)
	if candidate == "" {
		candidate = "s7.unnamed"
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

func (s *S7ModelingService) groupPathSegment(groups []repository.S7VariableGroupRecord, groupID *string) string {
	if groupID == nil || *groupID == "" {
		return ""
	}
	byID := make(map[string]repository.S7VariableGroupRecord, len(groups))
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
		segments = append([]string{normalizeDatapointSegment(group.Code)}, segments...)
		if group.ParentID == nil {
			break
		}
		currentID = *group.ParentID
	}
	return strings.Join(segments, ".")
}

func (s *S7ModelingService) validateAndLoadConnection(ctx context.Context, projectID, connectionID string) (*repository.ConnectionRecord, error) {
	if err := validateProjectID(projectID); err != nil {
		return nil, err
	}
	connection, err := s.connections.GetByProjectAndID(ctx, projectID, connectionID)
	if err != nil {
		return nil, err
	}
	if connection.Type != "s7" {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "接入源不是 S7 类型")
	}
	return connection, nil
}

func (s *S7ModelingService) validateAndLoadConnectionWithUser(ctx context.Context, projectID, connectionID, userID string) (*repository.ConnectionRecord, error) {
	if err := validateUserID(userID); err != nil {
		return nil, err
	}
	return s.validateAndLoadConnection(ctx, projectID, connectionID)
}

func defaultS7ProfileForFamily(family string, connection repository.ConnectionRecord) S7Profile {
	profile := S7Profile{
		ProjectID:            connection.ProjectID,
		ConnectionID:         connection.ID,
		PlcFamily:            firstNonEmpty(family, "S7 Compatible"),
		CommunicationMode:    "rack_slot",
		Host:                 firstNonEmpty(parseS7ConfigString(connection.Config["host"]), parseS7ConfigString(connection.Config["ip"])),
		Port:                 intOrDefault(parseS7ConfigIntPtr(connection.Config["port"]), 102),
		Rack:                 0,
		Slot:                 1,
		PollIntervalMS:       1000,
		ConnectTimeoutMS:     3000,
		ReadTimeoutMS:        3000,
		MaxGapBytes:          8,
		MaxConcurrentReads:   1,
		ByteOrder:            "big_endian",
		WordOrder:            "big_endian",
		AllowAbsoluteAddress: true,
		SupportedAreas:       []string{"DB", "M", "I", "Q"},
		Options:              map[string]any{},
		Configured:           false,
	}
	switch profile.PlcFamily {
	case "S7-200", "S7-200 SMART":
		profile.CommunicationMode = "tsap"
	case "S7-300", "S7-400":
		profile.Slot = 2
	case "S7-1200", "S7-1500":
		profile.Slot = 1
		profile.MaxGapBytes = 16
		profile.OptimizedBlockAccess = true
	}
	return profile
}

func ParseS7Address(addressText string, dataType string) (S7ParsedAddress, error) {
	text := strings.ToUpper(strings.TrimSpace(addressText))
	if text == "" {
		return S7ParsedAddress{}, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "S7 地址不能为空")
	}
	if matches := s7DBAddressPattern.FindStringSubmatch(text); len(matches) > 0 {
		dbNumber, _ := strconv.Atoi(matches[1])
		addressType := "DB" + matches[2]
		byteOffset, _ := strconv.Atoi(matches[3])
		bitOffset := parseOptionalS7Bit(matches[4])
		normalized, err := normalizeS7ParsedAddress("DB", &dbNumber, addressType, byteOffset, bitOffset, dataType)
		if err != nil {
			return S7ParsedAddress{}, err
		}
		return S7ParsedAddress{Area: "DB", DBNumber: &dbNumber, ByteOffset: byteOffset, BitOffset: bitOffset, AddressType: addressType, NormalizedAddress: normalized}, nil
	}
	if matches := s7AreaAddressPattern.FindStringSubmatch(text); len(matches) > 0 {
		area := strings.ToUpper(matches[1])
		width := strings.ToUpper(matches[2])
		if width == "" && matches[4] != "" {
			width = "X"
		}
		addressType := area + width
		byteOffset, _ := strconv.Atoi(matches[3])
		bitOffset := parseOptionalS7Bit(matches[4])
		normalized, err := normalizeS7ParsedAddress(area, nil, addressType, byteOffset, bitOffset, dataType)
		if err != nil {
			return S7ParsedAddress{}, err
		}
		return S7ParsedAddress{Area: area, ByteOffset: byteOffset, BitOffset: bitOffset, AddressType: addressType, NormalizedAddress: normalized}, nil
	}
	return S7ParsedAddress{}, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "S7 地址格式不支持")
}

func normalizeS7ParsedAddress(area string, dbNumber *int, addressType string, byteOffset int, bitOffset *int, dataType string) (string, error) {
	if bitOffset != nil && (*bitOffset < 0 || *bitOffset > 7) {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "bit 偏移必须在 0-7 之间")
	}
	if strings.HasSuffix(addressType, "X") || strings.EqualFold(dataType, "Bool") {
		if bitOffset == nil {
			return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "Bool 地址必须包含 bit 偏移")
		}
		if area == "DB" && dbNumber != nil {
			return fmt.Sprintf("DB%d.DBX%d.%d", *dbNumber, byteOffset, *bitOffset), nil
		}
		return fmt.Sprintf("%s%d.%d", area, byteOffset, *bitOffset), nil
	}
	if area == "DB" && dbNumber != nil {
		return fmt.Sprintf("DB%d.%s%d", *dbNumber, addressType, byteOffset), nil
	}
	return fmt.Sprintf("%s%d", addressType, byteOffset), nil
}

func s7ReadLengthForDataType(dataType string, length *int, arrayLength *int) (int, error) {
	normalized := normalizeS7DataType(dataType)
	elementLength := 1
	switch normalized {
	case "Bool", "Byte":
		elementLength = 1
	case "Word", "Int":
		elementLength = 2
	case "DWord", "DInt", "Real":
		elementLength = 4
	case "DateTime":
		elementLength = 8
	case "String":
		if length == nil || *length <= 0 {
			return 0, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "String 变量必须填写长度")
		}
		elementLength = 2 + *length
	default:
		return 0, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "S7 数据类型不支持")
	}
	count := intOrDefault(arrayLength, 1)
	if count <= 0 {
		return 0, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "数组长度必须大于 0")
	}
	return elementLength * count, nil
}

func BuildS7ReadPlanEstimate(profile S7Profile, variables []S7Variable) S7ReadPlanEstimate {
	active := make([]S7Variable, 0, len(variables))
	for _, variable := range variables {
		if variable.Status != "active" {
			continue
		}
		active = append(active, variable)
	}
	maxBytes := intOrDefault(profile.MaxReadBytes, 240)
	if maxBytes <= 0 {
		maxBytes = 240
	}
	maxGap := profile.MaxGapBytes
	if maxGap < 0 {
		maxGap = 0
	}
	grouped := map[string][]S7Variable{}
	for _, variable := range active {
		key := strings.Join([]string{variable.ConnectionID, variable.Area, optionalIntKey(variable.DBNumber), strconv.Itoa(variable.PollIntervalMS), s7ReadMode(variable)}, "|")
		grouped[key] = append(grouped[key], variable)
	}
	plans := make([]S7ReadPlan, 0)
	fragmentedGroups := 0
	for _, items := range grouped {
		sort.Slice(items, func(i, j int) bool { return items[i].ByteOffset < items[j].ByteOffset })
		initialPlanCount := len(plans)
		var current *S7ReadPlan
		for _, item := range items {
			start := item.ByteOffset
			end := item.ByteOffset + item.ReadLength - 1
			if current == nil || start > current.EndByte+maxGap || end-current.StartByte+1 > maxBytes {
				plan := newS7ReadPlan(item, start, end, maxGap)
				current = &plan
				plans = append(plans, plan)
				continue
			}
			if end > current.EndByte {
				current.EndByte = end
				current.ReadLength = current.EndByte - current.StartByte + 1
				current.DisplayRange = formatS7ReadRange(current.Area, current.DBNumber, current.StartByte, current.EndByte)
			}
			current.VariableIDs = append(current.VariableIDs, item.ID)
			current.VariableCount++
			plans[len(plans)-1] = *current
		}
		if len(plans)-initialPlanCount > 1 {
			fragmentedGroups++
		}
	}
	sort.Slice(plans, func(i, j int) bool {
		if plans[i].Area != plans[j].Area {
			return plans[i].Area < plans[j].Area
		}
		if optionalIntKey(plans[i].DBNumber) != optionalIntKey(plans[j].DBNumber) {
			return optionalIntKey(plans[i].DBNumber) < optionalIntKey(plans[j].DBNumber)
		}
		return plans[i].StartByte < plans[j].StartByte
	})
	totalBytes := 0
	largest := 0
	readsPerSecond := 0.0
	for index := range plans {
		plans[index].ID = fmt.Sprintf("s7-read-plan-%03d", index+1)
		plans[index].ReadsPerSecond = 1000 / float64(plans[index].PollIntervalMS)
		totalBytes += plans[index].ReadLength
		if plans[index].ReadLength > largest {
			largest = plans[index].ReadLength
		}
		readsPerSecond += plans[index].ReadsPerSecond
	}
	diagnostics := []string{fmt.Sprintf("你建了 %d 个 S7 变量，运行态预计合并为 %d 个读取块。", len(active), len(plans))}
	if readsPerSecond > 50 {
		diagnostics = append(diagnostics, "预计 reads/s 较高，建议调大采集周期或整理地址连续性。")
	}
	if profile.OptimizedBlockAccess {
		diagnostics = append(diagnostics, "当前 PLC 档案标记了优化 DB 访问风险，请确认 DB 块允许绝对地址读取。")
	}
	estimatedCycle := profile.PollIntervalMS
	if len(plans) > 0 {
		estimatedCycle = plans[0].PollIntervalMS
	}
	return S7ReadPlanEstimate{
		VariableCount:        len(active),
		BlockCount:           len(plans),
		TotalReadBytes:       totalBytes,
		ReadsPerSecond:       readsPerSecond,
		EstimatedCycleMS:     estimatedCycle,
		LargestBlockBytes:    largest,
		FragmentedGroupCount: fragmentedGroups,
		Plans:                plans,
		Diagnostics:          diagnostics,
	}
}

func newS7ReadPlan(item S7Variable, start, end, maxGap int) S7ReadPlan {
	return S7ReadPlan{
		ConnectionID:   item.ConnectionID,
		Area:           item.Area,
		DBNumber:       cloneOptionalInt(item.DBNumber),
		StartByte:      start,
		EndByte:        end,
		ReadLength:     end - start + 1,
		PollIntervalMS: item.PollIntervalMS,
		VariableIDs:    []string{item.ID},
		VariableCount:  1,
		MaxGapBytes:    maxGap,
		ReadMode:       s7ReadMode(item),
		DisplayRange:   formatS7ReadRange(item.Area, item.DBNumber, start, end),
	}
}

func normalizeS7RequiredText(value string, message string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, message)
	}
	return trimmed, nil
}

func normalizeS7Code(values ...string) string {
	source := firstNonEmpty(values...)
	source = strings.TrimSpace(strings.ToLower(source))
	normalized := s7CodeSanitizer.ReplaceAllString(source, "_")
	normalized = strings.Trim(normalized, "_")
	if normalized == "" {
		return "variable"
	}
	return normalized
}

func normalizeS7DataType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "bool", "boolean":
		return "Bool"
	case "byte", "uint8":
		return "Byte"
	case "word", "uint16":
		return "Word"
	case "dword", "uint32":
		return "DWord"
	case "int", "int16":
		return "Int"
	case "dint", "int32":
		return "DInt"
	case "real", "float", "float32":
		return "Real"
	case "datetime", "date_time":
		return "DateTime"
	case "string":
		return "String"
	default:
		trimmed := strings.TrimSpace(value)
		if _, ok := s7SupportedDataTypes[trimmed]; ok {
			return trimmed
		}
		return "Real"
	}
}

func normalizeS7DataPointType(dataType string) string {
	switch normalizeS7DataType(dataType) {
	case "Bool":
		return "boolean"
	case "String", "DateTime":
		return "string"
	default:
		return "number"
	}
}

func normalizeS7Endian(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "little", "little_endian":
		return "little_endian"
	default:
		return "big_endian"
	}
}

func normalizeS7SupportedAreas(values []string) []string {
	result := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		area := strings.ToUpper(strings.TrimSpace(value))
		if _, ok := s7SupportedAreaLookup[area]; !ok {
			continue
		}
		if _, ok := seen[area]; ok {
			continue
		}
		seen[area] = struct{}{}
		result = append(result, area)
	}
	return result
}

func stringsToAnySlice(values []string) []any {
	result := make([]any, 0, len(values))
	for _, value := range values {
		result = append(result, value)
	}
	return result
}

func normalizeMap(value map[string]any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	return value
}

func parseS7ConfigString(value any) string {
	if typed, ok := value.(string); ok {
		return strings.TrimSpace(typed)
	}
	return ""
}

func parseS7ConfigIntPtr(value any) *int {
	switch typed := value.(type) {
	case int:
		return &typed
	case int32:
		result := int(typed)
		return &result
	case int64:
		result := int(typed)
		return &result
	case float64:
		result := int(typed)
		return &result
	default:
		return nil
	}
}

func parseOptionalS7Bit(value string) *int {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parsed, _ := strconv.Atoi(value)
	return &parsed
}

func s7AddressTypeMatchesDataType(addressType string, dataType string) bool {
	addressType = strings.ToUpper(strings.TrimSpace(addressType))
	switch normalizeS7DataType(dataType) {
	case "Bool":
		return strings.HasSuffix(addressType, "X")
	case "Byte", "String":
		return strings.HasSuffix(addressType, "B")
	case "Word", "Int":
		return strings.HasSuffix(addressType, "W")
	case "DWord", "DInt", "Real", "DateTime":
		return strings.HasSuffix(addressType, "D") || strings.HasSuffix(addressType, "L")
	default:
		return true
	}
}

func s7ReadMode(variable S7Variable) string {
	if variable.Area == "DB" {
		return "db"
	}
	return "area"
}

func formatS7ReadRange(area string, dbNumber *int, start int, end int) string {
	prefix := area
	if area == "DB" && dbNumber != nil {
		prefix = fmt.Sprintf("DB%d", *dbNumber)
	}
	if start == end {
		return fmt.Sprintf("%s %d", prefix, start)
	}
	return fmt.Sprintf("%s %d-%d", prefix, start, end)
}

func optionalIntKey(value *int) string {
	if value == nil {
		return ""
	}
	return strconv.Itoa(*value)
}

func s7Issue(severity, code, variableID, variableName, message string) S7ValidationIssue {
	issue := S7ValidationIssue{Severity: severity, Code: code, VariableName: variableName, Message: message}
	if variableID != "" {
		issue.VariableID = &variableID
	}
	return issue
}

func s7VariableIssue(severity, code string, variable S7Variable, message string) S7ValidationIssue {
	return S7ValidationIssue{Severity: severity, Code: code, GroupID: cloneOptionalString(variable.GroupID), VariableID: &variable.ID, VariableName: variable.Name, Message: message}
}

func toS7Profile(record repository.S7ProfileRecord) S7Profile {
	return S7Profile{
		ID:                   record.ID,
		ProjectID:            record.ProjectID,
		ConnectionID:         record.ConnectionID,
		PlcFamily:            record.PlcFamily,
		CommunicationMode:    record.CommunicationMode,
		Host:                 record.Host,
		Port:                 record.Port,
		Rack:                 record.Rack,
		Slot:                 record.Slot,
		LocalTSAP:            cloneOptionalString(record.LocalTSAP),
		RemoteTSAP:           cloneOptionalString(record.RemoteTSAP),
		PollIntervalMS:       record.PollIntervalMS,
		ConnectTimeoutMS:     record.ConnectTimeoutMS,
		ReadTimeoutMS:        record.ReadTimeoutMS,
		PDUSize:              cloneOptionalInt(record.PDUSize),
		MaxReadBytes:         cloneOptionalInt(record.MaxReadBytes),
		MaxGapBytes:          record.MaxGapBytes,
		MaxConcurrentReads:   record.MaxConcurrentReads,
		ByteOrder:            record.ByteOrder,
		WordOrder:            record.WordOrder,
		OptimizedBlockAccess: record.OptimizedBlockAccess,
		AllowAbsoluteAddress: record.AllowAbsoluteAddress,
		AllowSymbolAddress:   record.AllowSymbolAddress,
		SupportedAreas:       bytesToStringSlice(record.SupportedAreas),
		Options:              bytesToMap(record.Options),
		Configured:           true,
	}
}

func toS7VariableGroup(record repository.S7VariableGroupRecord) S7VariableGroup {
	return S7VariableGroup{
		ID:           record.ID,
		ProjectID:    record.ProjectID,
		ConnectionID: record.ConnectionID,
		ParentID:     cloneOptionalString(record.ParentID),
		Name:         record.Name,
		Code:         record.Code,
		Description:  cloneOptionalString(record.Description),
		SortOrder:    record.SortOrder,
		CreatedAt:    record.CreatedAt,
		UpdatedAt:    record.UpdatedAt,
	}
}

func toS7Variable(record repository.S7VariableRecord) S7Variable {
	return S7Variable{
		ID:                record.ID,
		ProjectID:         record.ProjectID,
		ConnectionID:      record.ConnectionID,
		GroupID:           cloneOptionalString(record.GroupID),
		Name:              record.Name,
		Code:              record.Code,
		Description:       cloneOptionalString(record.Description),
		Area:              record.Area,
		DBNumber:          cloneOptionalInt(record.DBNumber),
		ByteOffset:        record.ByteOffset,
		BitOffset:         cloneOptionalInt(record.BitOffset),
		AddressText:       record.AddressText,
		NormalizedAddress: record.NormalizedAddress,
		AddressType:       record.AddressType,
		ReadLength:        record.ReadLength,
		DataType:          record.DataType,
		Length:            cloneOptionalInt(record.Length),
		ArrayLength:       cloneOptionalInt(record.ArrayLength),
		ByteOrder:         record.ByteOrder,
		WordOrder:         record.WordOrder,
		Scale:             record.Scale,
		Offset:            record.Offset,
		Unit:              cloneOptionalString(record.Unit),
		PollIntervalMS:    record.PollIntervalMS,
		LastValue:         bytesToAny(record.LastValue),
		Quality:           record.Quality,
		LastUpdatedAt:     cloneOptionalTime(record.LastUpdatedAt),
		SortOrder:         record.SortOrder,
		Status:            record.Status,
		DataPointID:       cloneOptionalString(record.DataPointID),
		DataPointPath:     cloneOptionalString(record.DataPointPath),
		DataPointStatus:   cloneOptionalString(record.DataPointStatus),
		CreatedAt:         record.CreatedAt,
		UpdatedAt:         record.UpdatedAt,
	}
}

func bytesToStringSlice(payload []byte) []string {
	if len(payload) == 0 {
		return []string{}
	}
	var result []string
	if err := json.Unmarshal(payload, &result); err != nil || result == nil {
		return []string{}
	}
	return result
}

func bytesToMap(payload []byte) map[string]any {
	if len(payload) == 0 {
		return map[string]any{}
	}
	var result map[string]any
	if err := json.Unmarshal(payload, &result); err != nil || result == nil {
		return map[string]any{}
	}
	return result
}

func bytesToAny(payload []byte) any {
	if len(payload) == 0 {
		return nil
	}
	var result any
	if err := json.Unmarshal(payload, &result); err != nil {
		return nil
	}
	return result
}

func cloneOptionalTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}
