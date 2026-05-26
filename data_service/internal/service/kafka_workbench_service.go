package service

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/indu-forge/data_service/internal/repository"
)

const (
	defaultKafkaSampleLimit = 100
	defaultKafkaTimeoutMS   = 5000
	maxKafkaSampleLimit     = 1000
	maxKafkaTimeoutMS       = 30000
)

var (
	allowedKafkaPartitionModes  = map[string]struct{}{"all": {}, "single": {}}
	allowedKafkaDecodes         = map[string]struct{}{"json": {}, "string": {}, "binary": {}}
	allowedKafkaWorkbenchStarts = map[string]struct{}{"latest": {}, "earliest": {}, "offset": {}}
)

// KafkaTopicGroup 表示 Kafka 工作台左侧 Topic 树分组。
type KafkaTopicGroup struct {
	ID           string    `json:"id"`
	ProjectID    string    `json:"projectId"`
	ConnectionID string    `json:"connectionId"`
	ParentID     *string   `json:"parentId"`
	Name         string    `json:"name"`
	SortOrder    int       `json:"sortOrder"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// KafkaTopicMapping 表示一个项目内 Topic 映射配置，不代表真实 Kafka Topic 生命周期。
type KafkaTopicMapping struct {
	ID            string    `json:"id"`
	ProjectID     string    `json:"projectId"`
	ConnectionID  string    `json:"connectionId"`
	GroupID       *string   `json:"groupId"`
	Name          string    `json:"name"`
	Topic         string    `json:"topic"`
	Description   string    `json:"description"`
	PartitionMode string    `json:"partitionMode"`
	Partition     *int      `json:"partition"`
	StartPosition string    `json:"startPosition"`
	Decode        string    `json:"decode"`
	SampleLimit   int       `json:"sampleLimit"`
	TimeoutMS     int       `json:"timeoutMs"`
	SortOrder     int       `json:"sortOrder"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

// KafkaField 表示 Kafka 消息字段与平台数据点之间的映射。
type KafkaField struct {
	ID             string    `json:"id"`
	ProjectID      string    `json:"projectId"`
	ConnectionID   string    `json:"connectionId"`
	TopicMappingID string    `json:"topicMappingId"`
	Name           string    `json:"name"`
	ValuePath      string    `json:"valuePath"`
	KeyPath        string    `json:"keyPath"`
	DataType       string    `json:"dataType"`
	Enabled        bool      `json:"enabled"`
	Description    string    `json:"description"`
	SortOrder      int       `json:"sortOrder"`
	SourceType     string    `json:"sourceType"`
	DataPointID    string    `json:"dataPointId"`
	DataPointPath  string    `json:"dataPointPath"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

// KafkaPreviewInput 描述一次开发态短时抓样请求。
type KafkaPreviewInput struct {
	Limit     int    `json:"limit"`
	TimeoutMS int    `json:"timeoutMs"`
	Decode    string `json:"decode"`
	Offset    *int64 `json:"offset"`
	Partition *int   `json:"partition"`
}

// KafkaSchemaField 是从样本 value 中递归推断出的字段候选。
type KafkaSchemaField struct {
	Path string `json:"path"`
	Type string `json:"type"`
}

// KafkaPreviewResult 是 Kafka 工作台预览响应，保留 adapter 原始诊断并额外展开字段 schema。
type KafkaPreviewResult struct {
	Protocol     string             `json:"protocol"`
	ConnectionID string             `json:"connectionId"`
	Topic        string             `json:"topic"`
	Status       string             `json:"status"`
	Schema       []KafkaSchemaField `json:"schema"`
	Samples      []any              `json:"samples"`
	Diagnostics  map[string]any     `json:"diagnostics"`
	DurationMS   int64              `json:"durationMs"`
	Truncated    bool               `json:"truncated"`
}

// CreateKafkaTopicGroupInput 描述创建 Topic 分组的输入。
type CreateKafkaTopicGroupInput struct {
	ParentID  *string `json:"parentId"`
	Name      string  `json:"name"`
	SortOrder int     `json:"sortOrder"`
}

// UpdateKafkaTopicGroupInput 描述更新 Topic 分组的输入。
type UpdateKafkaTopicGroupInput struct {
	ParentID  *string `json:"parentId"`
	HasParent bool    `json:"-"`
	Name      string  `json:"name"`
	SortOrder int     `json:"sortOrder"`
}

// CreateKafkaTopicMappingInput 描述创建 Topic 映射的输入。
type CreateKafkaTopicMappingInput struct {
	GroupID       *string `json:"groupId"`
	Name          string  `json:"name"`
	Topic         string  `json:"topic"`
	Description   string  `json:"description"`
	PartitionMode string  `json:"partitionMode"`
	Partition     *int    `json:"partition"`
	StartPosition string  `json:"startPosition"`
	Decode        string  `json:"decode"`
	SampleLimit   int     `json:"sampleLimit"`
	TimeoutMS     int     `json:"timeoutMs"`
	SortOrder     int     `json:"sortOrder"`
}

// UpdateKafkaTopicMappingInput 描述更新 Topic 映射的输入。
type UpdateKafkaTopicMappingInput struct {
	GroupID       *string `json:"groupId"`
	HasGroupID    bool    `json:"-"`
	Name          string  `json:"name"`
	Topic         string  `json:"topic"`
	Description   string  `json:"description"`
	PartitionMode string  `json:"partitionMode"`
	Partition     *int    `json:"partition"`
	StartPosition string  `json:"startPosition"`
	Decode        string  `json:"decode"`
	SampleLimit   int     `json:"sampleLimit"`
	TimeoutMS     int     `json:"timeoutMs"`
	SortOrder     int     `json:"sortOrder"`
}

// CreateKafkaFieldInput 描述创建字段映射的输入。
type CreateKafkaFieldInput struct {
	Name        string `json:"name"`
	ValuePath   string `json:"valuePath"`
	KeyPath     string `json:"keyPath"`
	DataType    string `json:"dataType"`
	Enabled     bool   `json:"enabled"`
	Description string `json:"description"`
	SortOrder   int    `json:"sortOrder"`
}

// UpdateKafkaFieldInput 描述更新字段映射的输入。
type UpdateKafkaFieldInput struct {
	Name        string `json:"name"`
	ValuePath   string `json:"valuePath"`
	KeyPath     string `json:"keyPath"`
	DataType    string `json:"dataType"`
	Enabled     bool   `json:"enabled"`
	Description string `json:"description"`
	SortOrder   int    `json:"sortOrder"`
}

// KafkaWorkbenchService 承载 Kafka 工作台开发态配置、预览和字段建模规则。
type KafkaWorkbenchService struct {
	repository *repository.KafkaWorkbenchRepository
	preview    KafkaPreviewAdapter
}

// NewKafkaWorkbenchService 创建 Kafka 工作台服务。
func NewKafkaWorkbenchService(repo *repository.KafkaWorkbenchRepository) *KafkaWorkbenchService {
	return &KafkaWorkbenchService{repository: repo, preview: KafkaPreviewAdapter{}}
}

// ListTopicGroups 返回连接下 Topic 分组。
func (s *KafkaWorkbenchService) ListTopicGroups(ctx context.Context, projectID, connectionID string) ([]KafkaTopicGroup, error) {
	if err := validateProjectAndConnection(projectID, connectionID); err != nil {
		return nil, err
	}
	records, err := s.repository.ListTopicGroups(ctx, projectID, connectionID)
	if err != nil {
		return nil, err
	}
	result := make([]KafkaTopicGroup, 0, len(records))
	for _, record := range records {
		result = append(result, toKafkaTopicGroup(record))
	}
	return result, nil
}

// CreateTopicGroup 创建 Topic 分组。
func (s *KafkaWorkbenchService) CreateTopicGroup(ctx context.Context, projectID, connectionID, userID string, input CreateKafkaTopicGroupInput) (*KafkaTopicGroup, error) {
	if err := validateProjectConnectionAndUser(projectID, connectionID, userID); err != nil {
		return nil, err
	}
	name, err := normalizeKafkaRequiredText(input.Name, 100, "Topic 分组名称不能为空")
	if err != nil {
		return nil, err
	}
	parentID, err := s.normalizeKafkaGroupParent(ctx, projectID, connectionID, input.ParentID, "")
	if err != nil {
		return nil, err
	}
	record, err := s.repository.CreateTopicGroup(ctx, repository.CreateKafkaTopicGroupParams{
		ProjectID:    projectID,
		ConnectionID: connectionID,
		ParentID:     parentID,
		Name:         name,
		SortOrder:    input.SortOrder,
		UserID:       userID,
	})
	if err != nil {
		return nil, err
	}
	result := toKafkaTopicGroup(*record)
	return &result, nil
}

// UpdateTopicGroup 更新 Topic 分组。
func (s *KafkaWorkbenchService) UpdateTopicGroup(ctx context.Context, projectID, groupID, userID string, input UpdateKafkaTopicGroupInput) (*KafkaTopicGroup, error) {
	if err := validateProjectAndUser(projectID, userID); err != nil {
		return nil, err
	}
	if err := validateUUIDText(groupID, "groupId 格式无效"); err != nil {
		return nil, err
	}
	current, err := s.repository.GetTopicGroup(ctx, projectID, groupID)
	if err != nil {
		return nil, err
	}
	name, err := normalizeKafkaRequiredText(fallbackTrimmed(input.Name, current.Name), 100, "Topic 分组名称不能为空")
	if err != nil {
		return nil, err
	}
	parentID := cloneOptionalString(current.ParentID)
	if input.HasParent {
		parentID, err = s.normalizeKafkaGroupParent(ctx, projectID, current.ConnectionID, input.ParentID, current.ID)
		if err != nil {
			return nil, err
		}
	}
	sortOrder := input.SortOrder
	if sortOrder == 0 && current.SortOrder != 0 {
		sortOrder = current.SortOrder
	}
	record, err := s.repository.UpdateTopicGroup(ctx, repository.UpdateKafkaTopicGroupParams{
		ProjectID: projectID,
		GroupID:   groupID,
		ParentID:  parentID,
		HasParent: true,
		Name:      name,
		SortOrder: sortOrder,
		UserID:    userID,
	})
	if err != nil {
		return nil, err
	}
	result := toKafkaTopicGroup(*record)
	return &result, nil
}

// DeleteTopicGroup 删除 Topic 分组。
func (s *KafkaWorkbenchService) DeleteTopicGroup(ctx context.Context, projectID, groupID string) error {
	if err := validateProjectID(projectID); err != nil {
		return err
	}
	if err := validateUUIDText(groupID, "groupId 格式无效"); err != nil {
		return err
	}
	return s.repository.DeleteTopicGroup(ctx, projectID, groupID)
}

// ListTopicMappings 返回连接下 Topic 映射。
func (s *KafkaWorkbenchService) ListTopicMappings(ctx context.Context, projectID, connectionID string) ([]KafkaTopicMapping, error) {
	if err := validateProjectAndConnection(projectID, connectionID); err != nil {
		return nil, err
	}
	records, err := s.repository.ListTopicMappings(ctx, projectID, connectionID)
	if err != nil {
		return nil, err
	}
	result := make([]KafkaTopicMapping, 0, len(records))
	for _, record := range records {
		result = append(result, toKafkaTopicMapping(record))
	}
	return result, nil
}

// GetTopicMapping 返回单个 Topic 映射。
func (s *KafkaWorkbenchService) GetTopicMapping(ctx context.Context, projectID, mappingID string) (*KafkaTopicMapping, error) {
	if err := validateProjectID(projectID); err != nil {
		return nil, err
	}
	if err := validateUUIDText(mappingID, "mappingId 格式无效"); err != nil {
		return nil, err
	}
	record, err := s.repository.GetTopicMapping(ctx, projectID, mappingID)
	if err != nil {
		return nil, err
	}
	result := toKafkaTopicMapping(*record)
	return &result, nil
}

// CreateTopicMapping 创建 Topic 映射。
func (s *KafkaWorkbenchService) CreateTopicMapping(ctx context.Context, projectID, connectionID, userID string, input CreateKafkaTopicMappingInput) (*KafkaTopicMapping, error) {
	if err := validateProjectConnectionAndUser(projectID, connectionID, userID); err != nil {
		return nil, err
	}
	params, err := s.normalizeCreateTopicMapping(ctx, projectID, connectionID, userID, input)
	if err != nil {
		return nil, err
	}
	record, err := s.repository.CreateTopicMapping(ctx, params)
	if err != nil {
		return nil, err
	}
	result := toKafkaTopicMapping(*record)
	return &result, nil
}

// UpdateTopicMapping 更新 Topic 映射。
func (s *KafkaWorkbenchService) UpdateTopicMapping(ctx context.Context, projectID, mappingID, userID string, input UpdateKafkaTopicMappingInput) (*KafkaTopicMapping, error) {
	if err := validateProjectAndUser(projectID, userID); err != nil {
		return nil, err
	}
	if err := validateUUIDText(mappingID, "mappingId 格式无效"); err != nil {
		return nil, err
	}
	current, err := s.repository.GetTopicMapping(ctx, projectID, mappingID)
	if err != nil {
		return nil, err
	}
	params, err := s.normalizeUpdateTopicMapping(ctx, projectID, userID, *current, input)
	if err != nil {
		return nil, err
	}
	record, err := s.repository.UpdateTopicMapping(ctx, params)
	if err != nil {
		return nil, err
	}
	result := toKafkaTopicMapping(*record)
	return &result, nil
}

// DeleteTopicMapping 删除 Topic 映射，并由仓储标记字段数据点失效。
func (s *KafkaWorkbenchService) DeleteTopicMapping(ctx context.Context, projectID, mappingID string) error {
	if err := validateProjectID(projectID); err != nil {
		return err
	}
	if err := validateUUIDText(mappingID, "mappingId 格式无效"); err != nil {
		return err
	}
	return s.repository.DeleteTopicMapping(ctx, projectID, mappingID)
}

// ListFields 返回 Topic 映射下字段映射。
func (s *KafkaWorkbenchService) ListFields(ctx context.Context, projectID, mappingID string) ([]KafkaField, error) {
	if err := validateProjectID(projectID); err != nil {
		return nil, err
	}
	if err := validateUUIDText(mappingID, "mappingId 格式无效"); err != nil {
		return nil, err
	}
	records, err := s.repository.ListFields(ctx, projectID, mappingID)
	if err != nil {
		return nil, err
	}
	result := make([]KafkaField, 0, len(records))
	for _, record := range records {
		result = append(result, toKafkaField(record))
	}
	return result, nil
}

// CreateField 创建字段映射并同步 kafka.field 数据点。
func (s *KafkaWorkbenchService) CreateField(ctx context.Context, projectID, mappingID, userID string, input CreateKafkaFieldInput) (*KafkaField, error) {
	if err := validateProjectAndUser(projectID, userID); err != nil {
		return nil, err
	}
	if err := validateUUIDText(mappingID, "mappingId 格式无效"); err != nil {
		return nil, err
	}
	mapping, err := s.repository.GetTopicMapping(ctx, projectID, mappingID)
	if err != nil {
		return nil, err
	}
	params, err := s.normalizeCreateField(projectID, userID, *mapping, input)
	if err != nil {
		return nil, err
	}
	record, err := s.repository.CreateFieldWithDataPoint(ctx, params)
	if err != nil {
		return nil, err
	}
	result := toKafkaField(*record)
	return &result, nil
}

// CreateFieldsBatch 批量创建字段映射，遇到任一字段非法时停止并返回错误。
func (s *KafkaWorkbenchService) CreateFieldsBatch(ctx context.Context, projectID, mappingID, userID string, inputs []CreateKafkaFieldInput) ([]KafkaField, error) {
	if len(inputs) == 0 {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "字段映射不能为空")
	}
	result := make([]KafkaField, 0, len(inputs))
	for index, input := range inputs {
		if input.SortOrder == 0 {
			input.SortOrder = index
		}
		field, err := s.CreateField(ctx, projectID, mappingID, userID, input)
		if err != nil {
			return nil, err
		}
		result = append(result, *field)
	}
	return result, nil
}

// UpdateField 更新字段映射并同步 kafka.field 数据点。
func (s *KafkaWorkbenchService) UpdateField(ctx context.Context, projectID, fieldID, userID string, input UpdateKafkaFieldInput) (*KafkaField, error) {
	if err := validateProjectAndUser(projectID, userID); err != nil {
		return nil, err
	}
	if err := validateUUIDText(fieldID, "fieldId 格式无效"); err != nil {
		return nil, err
	}
	current, err := s.repository.GetField(ctx, projectID, fieldID)
	if err != nil {
		return nil, err
	}
	params, err := s.normalizeUpdateField(ctx, projectID, userID, *current, input)
	if err != nil {
		return nil, err
	}
	record, err := s.repository.UpdateFieldWithDataPoint(ctx, params)
	if err != nil {
		return nil, err
	}
	result := toKafkaField(*record)
	return &result, nil
}

// ToggleField 切换字段映射启停状态，同步更新生成数据点状态。
func (s *KafkaWorkbenchService) ToggleField(ctx context.Context, projectID, fieldID, userID string, enabled bool) (*KafkaField, error) {
	if err := validateProjectAndUser(projectID, userID); err != nil {
		return nil, err
	}
	if err := validateUUIDText(fieldID, "fieldId 格式无效"); err != nil {
		return nil, err
	}
	current, err := s.repository.GetField(ctx, projectID, fieldID)
	if err != nil {
		return nil, err
	}
	return s.UpdateField(ctx, projectID, fieldID, userID, UpdateKafkaFieldInput{
		Name:        current.Name,
		ValuePath:   current.ValuePath,
		KeyPath:     current.KeyPath,
		DataType:    current.DataType,
		Enabled:     enabled,
		Description: current.Description,
		SortOrder:   current.SortOrder,
	})
}

// DeleteField 删除字段映射，并由仓储标记生成数据点失效。
func (s *KafkaWorkbenchService) DeleteField(ctx context.Context, projectID, fieldID string) error {
	if err := validateProjectID(projectID); err != nil {
		return err
	}
	if err := validateUUIDText(fieldID, "fieldId 格式无效"); err != nil {
		return err
	}
	return s.repository.DeleteFieldWithDataPoint(ctx, projectID, fieldID)
}

// PreviewConnection 对 Kafka 接入源默认 Topic 执行一次短时预览。
func (s *KafkaWorkbenchService) PreviewConnection(ctx context.Context, projectID, connectionID string, input KafkaPreviewInput) (*KafkaPreviewResult, error) {
	if err := validateProjectAndConnection(projectID, connectionID); err != nil {
		return nil, err
	}
	connection, err := s.repository.GetPreviewConnection(ctx, projectID, connectionID)
	if err != nil {
		return nil, err
	}
	result, previewErr := s.previewKafka(ctx, *connection, input, nil)
	_ = s.recordKafkaPreview(ctx, connection.ProjectID, connection.ID, result, previewErr)
	return result, previewErr
}

// PreviewTopicMapping 按 Topic 映射配置执行一次短时预览。
func (s *KafkaWorkbenchService) PreviewTopicMapping(ctx context.Context, projectID, mappingID string, input KafkaPreviewInput) (*KafkaPreviewResult, error) {
	if err := validateProjectID(projectID); err != nil {
		return nil, err
	}
	if err := validateUUIDText(mappingID, "mappingId 格式无效"); err != nil {
		return nil, err
	}
	mapping, err := s.repository.GetTopicMapping(ctx, projectID, mappingID)
	if err != nil {
		return nil, err
	}
	connection, err := s.repository.GetPreviewConnection(ctx, projectID, mapping.ConnectionID)
	if err != nil {
		return nil, err
	}
	overrides := map[string]any{
		"topic":         mapping.Topic,
		"partitionMode": mapping.PartitionMode,
		"partition":     mapping.Partition,
		"startPosition": mapping.StartPosition,
		"decode":        mapping.Decode,
	}
	if input.Limit <= 0 {
		input.Limit = mapping.SampleLimit
	}
	if input.TimeoutMS <= 0 {
		input.TimeoutMS = mapping.TimeoutMS
	}
	if input.Decode == "" {
		input.Decode = mapping.Decode
	}
	if input.Partition == nil {
		input.Partition = cloneOptionalInt(mapping.Partition)
	}
	result, previewErr := s.previewKafka(ctx, *connection, input, overrides)
	_ = s.recordKafkaPreview(ctx, mapping.ProjectID, mapping.ConnectionID, result, previewErr)
	return result, previewErr
}

func (s *KafkaWorkbenchService) previewKafka(ctx context.Context, connection repository.ProtocolPreviewConnectionRecord, input KafkaPreviewInput, overrides map[string]any) (*KafkaPreviewResult, error) {
	limit, timeout := normalizeKafkaPreviewLimits(input)
	options := cloneMap(overrides)
	if input.Decode != "" {
		options["decode"] = normalizeKafkaDecode(input.Decode)
	}
	if input.Offset != nil {
		options["offset"] = *input.Offset
		options["startPosition"] = "offset"
	}
	if input.Partition != nil {
		options["partitionMode"] = "single"
		options["partition"] = *input.Partition
	}
	previewCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	raw, err := s.preview.Preview(previewCtx, ProtocolPreviewAdapterInput{
		Connection: connection,
		Limit:      limit,
		Timeout:    timeout,
		Options:    options,
	})
	if err != nil {
		failed := failedProtocolPreviewResult("kafka", connection.ID, time.Now(), err)
		return toKafkaPreviewResult(failed), err
	}
	return toKafkaPreviewResult(raw), nil
}

func (s *KafkaWorkbenchService) recordKafkaPreview(ctx context.Context, projectID, connectionID string, result *KafkaPreviewResult, previewErr error) error {
	if result == nil {
		return nil
	}
	errorSummary := ""
	if previewErr != nil {
		errorSummary = previewErr.Error()
	}
	return s.repository.CreatePreviewRecord(ctx, repository.CreateAccessSourceRecordParams{
		ProjectID:    projectID,
		ConnectionID: connectionID,
		RecordType:   "kafka.preview",
		Title:        "执行 Kafka 短时预览",
		Status:       result.Status,
		Protocol:     result.Protocol,
		DurationMS:   result.DurationMS,
		SampleCount:  len(result.Samples),
		Truncated:    result.Truncated,
		ErrorSummary: errorSummary,
		Detail: map[string]any{
			"topic":       result.Topic,
			"durationMs":  result.DurationMS,
			"sampleCount": len(result.Samples),
			"truncated":   result.Truncated,
		},
	})
}

func (s *KafkaWorkbenchService) normalizeCreateTopicMapping(ctx context.Context, projectID, connectionID, userID string, input CreateKafkaTopicMappingInput) (repository.CreateKafkaTopicMappingParams, error) {
	name, err := normalizeKafkaRequiredText(input.Name, 100, "Topic 映射名称不能为空")
	if err != nil {
		return repository.CreateKafkaTopicMappingParams{}, err
	}
	topic, err := normalizeKafkaRequiredText(input.Topic, 500, "Topic 不能为空")
	if err != nil {
		return repository.CreateKafkaTopicMappingParams{}, err
	}
	partitionMode, partition, err := normalizeKafkaPartition(input.PartitionMode, input.Partition)
	if err != nil {
		return repository.CreateKafkaTopicMappingParams{}, err
	}
	startPosition, err := normalizeKafkaStartPosition(input.StartPosition)
	if err != nil {
		return repository.CreateKafkaTopicMappingParams{}, err
	}
	decode := normalizeKafkaDecode(input.Decode)
	groupID, err := s.normalizeKafkaMappingGroup(ctx, projectID, connectionID, input.GroupID)
	if err != nil {
		return repository.CreateKafkaTopicMappingParams{}, err
	}
	return repository.CreateKafkaTopicMappingParams{
		ProjectID:     projectID,
		ConnectionID:  connectionID,
		GroupID:       groupID,
		Name:          name,
		Topic:         topic,
		Description:   strings.TrimSpace(input.Description),
		PartitionMode: partitionMode,
		Partition:     partition,
		StartPosition: startPosition,
		Decode:        decode,
		SampleLimit:   normalizeKafkaBoundedInt(input.SampleLimit, defaultKafkaSampleLimit, 1, maxKafkaSampleLimit),
		TimeoutMS:     normalizeKafkaBoundedInt(input.TimeoutMS, defaultKafkaTimeoutMS, 1000, maxKafkaTimeoutMS),
		SortOrder:     input.SortOrder,
		UserID:        userID,
	}, nil
}

func (s *KafkaWorkbenchService) normalizeUpdateTopicMapping(ctx context.Context, projectID, userID string, current repository.KafkaTopicMappingRecord, input UpdateKafkaTopicMappingInput) (repository.UpdateKafkaTopicMappingParams, error) {
	name := fallbackTrimmed(input.Name, current.Name)
	topic := fallbackTrimmed(input.Topic, current.Topic)
	partitionMode := fallbackTrimmed(input.PartitionMode, current.PartitionMode)
	partition := input.Partition
	if partition == nil {
		partition = cloneOptionalInt(current.Partition)
	}
	normalizedMode, normalizedPartition, err := normalizeKafkaPartition(partitionMode, partition)
	if err != nil {
		return repository.UpdateKafkaTopicMappingParams{}, err
	}
	startPosition, err := normalizeKafkaStartPosition(fallbackTrimmed(input.StartPosition, current.StartPosition))
	if err != nil {
		return repository.UpdateKafkaTopicMappingParams{}, err
	}
	groupID := cloneOptionalString(current.GroupID)
	if input.HasGroupID {
		groupID, err = s.normalizeKafkaMappingGroup(ctx, projectID, current.ConnectionID, input.GroupID)
		if err != nil {
			return repository.UpdateKafkaTopicMappingParams{}, err
		}
	}
	return repository.UpdateKafkaTopicMappingParams{
		ID:            current.ID,
		ProjectID:     projectID,
		GroupID:       groupID,
		HasGroupID:    true,
		Name:          name,
		Topic:         topic,
		Description:   strings.TrimSpace(input.Description),
		PartitionMode: normalizedMode,
		Partition:     normalizedPartition,
		StartPosition: startPosition,
		Decode:        normalizeKafkaDecode(fallbackTrimmed(input.Decode, current.Decode)),
		SampleLimit:   normalizeKafkaBoundedInt(input.SampleLimit, current.SampleLimit, 1, maxKafkaSampleLimit),
		TimeoutMS:     normalizeKafkaBoundedInt(input.TimeoutMS, current.TimeoutMS, 1000, maxKafkaTimeoutMS),
		SortOrder:     input.SortOrder,
		UserID:        userID,
	}, nil
}

func (s *KafkaWorkbenchService) normalizeCreateField(projectID, userID string, mapping repository.KafkaTopicMappingRecord, input CreateKafkaFieldInput) (repository.CreateKafkaFieldParams, error) {
	name, valuePath, dataType, err := normalizeKafkaFieldCore(input.Name, input.ValuePath, input.DataType)
	if err != nil {
		return repository.CreateKafkaFieldParams{}, err
	}
	sourceConfig := kafkaFieldSourceConfig(mapping, valuePath, input.KeyPath)
	return repository.CreateKafkaFieldParams{
		ProjectID:      projectID,
		ConnectionID:   mapping.ConnectionID,
		TopicMappingID: mapping.ID,
		Name:           name,
		ValuePath:      valuePath,
		KeyPath:        strings.TrimSpace(input.KeyPath),
		DataType:       dataType,
		Enabled:        input.Enabled,
		Description:    strings.TrimSpace(input.Description),
		SortOrder:      input.SortOrder,
		DataPointPath:  buildKafkaDataPointPath(mapping.Name, name),
		SourceConfig:   sourceConfig,
		UserID:         userID,
	}, nil
}

func (s *KafkaWorkbenchService) normalizeUpdateField(ctx context.Context, projectID, userID string, current repository.KafkaFieldRecord, input UpdateKafkaFieldInput) (repository.UpdateKafkaFieldParams, error) {
	mapping, err := s.repository.GetTopicMapping(ctx, projectID, current.TopicMappingID)
	if err != nil {
		return repository.UpdateKafkaFieldParams{}, err
	}
	name, valuePath, dataType, err := normalizeKafkaFieldCore(fallbackTrimmed(input.Name, current.Name), fallbackTrimmed(input.ValuePath, current.ValuePath), fallbackTrimmed(input.DataType, current.DataType))
	if err != nil {
		return repository.UpdateKafkaFieldParams{}, err
	}
	keyPath := input.KeyPath
	if strings.TrimSpace(keyPath) == "" {
		keyPath = current.KeyPath
	}
	sourceConfig := kafkaFieldSourceConfig(*mapping, valuePath, keyPath)
	return repository.UpdateKafkaFieldParams{
		ID:            current.ID,
		ProjectID:     projectID,
		Name:          name,
		ValuePath:     valuePath,
		KeyPath:       strings.TrimSpace(keyPath),
		DataType:      dataType,
		Enabled:       input.Enabled,
		Description:   strings.TrimSpace(input.Description),
		SortOrder:     input.SortOrder,
		DataPointPath: buildKafkaDataPointPath(mapping.Name, name),
		SourceConfig:  sourceConfig,
		UserID:        userID,
	}, nil
}

func (s *KafkaWorkbenchService) normalizeKafkaMappingGroup(ctx context.Context, projectID, connectionID string, groupID *string) (*string, error) {
	groupID = normalizeOptionalText(groupID)
	if groupID == nil {
		return nil, nil
	}
	groups, err := s.repository.ListTopicGroups(ctx, projectID, connectionID)
	if err != nil {
		return nil, err
	}
	for _, group := range groups {
		if group.ID == *groupID {
			return groupID, nil
		}
	}
	return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "Topic 分组不属于当前 Kafka 接入源")
}

func (s *KafkaWorkbenchService) normalizeKafkaGroupParent(ctx context.Context, projectID, connectionID string, parentID *string, currentID string) (*string, error) {
	parentID = normalizeOptionalText(parentID)
	if parentID == nil {
		return nil, nil
	}
	groups, err := s.repository.ListTopicGroups(ctx, projectID, connectionID)
	if err != nil {
		return nil, err
	}
	if currentID != "" && isKafkaTopicGroupDescendant(groups, currentID, *parentID) {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "Topic 分组不能移动到自身或子分组下")
	}
	for _, group := range groups {
		if group.ID == *parentID {
			return parentID, nil
		}
	}
	return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "父级 Topic 分组不存在")
}

func validateProjectAndConnection(projectID, connectionID string) error {
	if err := validateProjectID(projectID); err != nil {
		return err
	}
	return validateConnectionID(connectionID)
}

func validateProjectAndUser(projectID, userID string) error {
	if err := validateProjectID(projectID); err != nil {
		return err
	}
	return validateUserID(userID)
}

func validateProjectConnectionAndUser(projectID, connectionID, userID string) error {
	if err := validateProjectAndConnection(projectID, connectionID); err != nil {
		return err
	}
	return validateUserID(userID)
}

func validateUUIDText(value, message string) error {
	if _, err := normalizeUUID(value); err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, message, err)
	}
	return nil
}

func normalizeUUID(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if _, err := uuid.Parse(trimmed); err != nil {
		return "", err
	}
	return trimmed, nil
}

func normalizeKafkaRequiredText(value string, maxLen int, emptyMessage string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, emptyMessage)
	}
	if len([]rune(trimmed)) > maxLen {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, fmt.Sprintf("长度不能超过 %d 个字符", maxLen))
	}
	return trimmed, nil
}

func normalizeKafkaPartition(mode string, partition *int) (string, *int, error) {
	normalizedMode := strings.ToLower(strings.TrimSpace(mode))
	if normalizedMode == "" {
		normalizedMode = "all"
	}
	if _, ok := allowedKafkaPartitionModes[normalizedMode]; !ok {
		return "", nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "partitionMode 不合法")
	}
	if normalizedMode == "single" {
		if partition == nil || *partition < 0 {
			return "", nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "单分区预览必须指定非负 partition")
		}
		return normalizedMode, partition, nil
	}
	return normalizedMode, nil, nil
}

func normalizeKafkaStartPosition(value string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "" {
		normalized = "latest"
	}
	if _, ok := allowedKafkaWorkbenchStarts[normalized]; !ok {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "startPosition 不合法")
	}
	return normalized, nil
}

func normalizeKafkaDecode(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if _, ok := allowedKafkaDecodes[normalized]; ok {
		return normalized
	}
	return "json"
}

func normalizeKafkaBoundedInt(value, fallback, min, max int) int {
	if value <= 0 {
		value = fallback
	}
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func normalizeKafkaPreviewLimits(input KafkaPreviewInput) (int, time.Duration) {
	limit := normalizeKafkaBoundedInt(input.Limit, defaultKafkaSampleLimit, 1, maxKafkaSampleLimit)
	timeoutMS := normalizeKafkaBoundedInt(input.TimeoutMS, defaultKafkaTimeoutMS, 1000, maxKafkaTimeoutMS)
	return limit, time.Duration(timeoutMS) * time.Millisecond
}

func normalizeKafkaFieldCore(name, valuePath, dataType string) (string, string, string, error) {
	normalizedName, err := normalizeKafkaRequiredText(name, 100, "字段名称不能为空")
	if err != nil {
		return "", "", "", err
	}
	normalizedPath, err := normalizeKafkaRequiredText(valuePath, 500, "字段路径不能为空")
	if err != nil {
		return "", "", "", err
	}
	normalizedType := strings.ToLower(strings.TrimSpace(dataType))
	if normalizedType == "" {
		normalizedType = "string"
	}
	return normalizedName, normalizedPath, normalizedType, nil
}

func kafkaFieldSourceConfig(mapping repository.KafkaTopicMappingRecord, valuePath, keyPath string) map[string]any {
	// sourceConfig 描述运行态未来如何解析消息；当前工作台只写配置，不启动长期消费。
	return map[string]any{
		"connectionId":   mapping.ConnectionID,
		"topicMappingId": mapping.ID,
		"topic":          mapping.Topic,
		"valuePath":      valuePath,
		"keyPath":        strings.TrimSpace(keyPath),
		"decode":         mapping.Decode,
		"partitionMode":  mapping.PartitionMode,
		"partition":      mapping.Partition,
	}
}

func buildKafkaDataPointPath(mappingName, fieldName string) string {
	return "kafka." + normalizeDatapointSegment(mappingName) + "." + normalizeDatapointSegment(fieldName)
}

func toKafkaTopicGroup(record repository.KafkaTopicGroupRecord) KafkaTopicGroup {
	return KafkaTopicGroup{
		ID:           record.ID,
		ProjectID:    record.ProjectID,
		ConnectionID: record.ConnectionID,
		ParentID:     cloneOptionalString(record.ParentID),
		Name:         record.Name,
		SortOrder:    record.SortOrder,
		CreatedAt:    record.CreatedAt,
		UpdatedAt:    record.UpdatedAt,
	}
}

func toKafkaTopicMapping(record repository.KafkaTopicMappingRecord) KafkaTopicMapping {
	return KafkaTopicMapping{
		ID:            record.ID,
		ProjectID:     record.ProjectID,
		ConnectionID:  record.ConnectionID,
		GroupID:       cloneOptionalString(record.GroupID),
		Name:          record.Name,
		Topic:         record.Topic,
		Description:   record.Description,
		PartitionMode: record.PartitionMode,
		Partition:     cloneOptionalInt(record.Partition),
		StartPosition: record.StartPosition,
		Decode:        record.Decode,
		SampleLimit:   record.SampleLimit,
		TimeoutMS:     record.TimeoutMS,
		SortOrder:     record.SortOrder,
		CreatedAt:     record.CreatedAt,
		UpdatedAt:     record.UpdatedAt,
	}
}

func toKafkaField(record repository.KafkaFieldRecord) KafkaField {
	return KafkaField{
		ID:             record.ID,
		ProjectID:      record.ProjectID,
		ConnectionID:   record.ConnectionID,
		TopicMappingID: record.TopicMappingID,
		Name:           record.Name,
		ValuePath:      record.ValuePath,
		KeyPath:        record.KeyPath,
		DataType:       record.DataType,
		Enabled:        record.Enabled,
		Description:    record.Description,
		SortOrder:      record.SortOrder,
		SourceType:     "kafka.field",
		DataPointID:    kafkaStringValue(record.DataPointID),
		DataPointPath:  kafkaStringValue(record.DataPointPath),
		CreatedAt:      record.CreatedAt,
		UpdatedAt:      record.UpdatedAt,
	}
}

func toKafkaPreviewResult(result *ProtocolPreviewResult) *KafkaPreviewResult {
	if result == nil {
		return &KafkaPreviewResult{Protocol: "kafka", Status: "ok", Samples: []any{}, Diagnostics: map[string]any{}, Schema: []KafkaSchemaField{}}
	}
	topic := ""
	if result.Diagnostics != nil {
		topic = strings.TrimSpace(toString(result.Diagnostics["topic"]))
	}
	return &KafkaPreviewResult{
		Protocol:     result.Protocol,
		ConnectionID: result.ConnectionID,
		Topic:        topic,
		Status:       result.Status,
		Schema:       inferKafkaSchemaFields(result.Samples),
		Samples:      result.Samples,
		Diagnostics:  cloneMap(result.Diagnostics),
		DurationMS:   result.DurationMS,
		Truncated:    result.Truncated,
	}
}

func inferKafkaSchemaFields(samples []any) []KafkaSchemaField {
	fields := make([]KafkaSchemaField, 0)
	seen := map[string]struct{}{}
	var visit func(prefix string, value any)
	visit = func(prefix string, value any) {
		switch typed := value.(type) {
		case map[string]any:
			for key, child := range typed {
				next := key
				if prefix != "" {
					next = prefix + "." + key
				}
				visit(next, child)
			}
		case []any:
			if prefix != "" {
				if _, ok := seen[prefix]; !ok {
					seen[prefix] = struct{}{}
					fields = append(fields, KafkaSchemaField{Path: prefix, Type: "array"})
				}
			}
			for _, child := range typed {
				visit(prefix, child)
			}
		default:
			if prefix == "" {
				return
			}
			if _, ok := seen[prefix]; ok {
				return
			}
			seen[prefix] = struct{}{}
			fields = append(fields, KafkaSchemaField{Path: prefix, Type: kafkaSchemaValueType(value)})
		}
	}
	for _, sample := range samples {
		if mapped, ok := sample.(map[string]any); ok {
			visit("", mapped["value"])
		} else {
			visit("", sample)
		}
	}
	return fields
}

func kafkaSchemaValueType(value any) string {
	switch value.(type) {
	case float64, float32, int, int64, int32:
		return "number"
	case bool:
		return "boolean"
	case map[string]any:
		return "object"
	case []any:
		return "array"
	case nil:
		return "null"
	default:
		return "string"
	}
}

func isKafkaTopicGroupDescendant(groups []repository.KafkaTopicGroupRecord, groupID, possibleDescendantID string) bool {
	byID := make(map[string]repository.KafkaTopicGroupRecord, len(groups))
	for _, group := range groups {
		byID[group.ID] = group
	}
	currentID := possibleDescendantID
	visited := map[string]struct{}{}
	for currentID != "" {
		if currentID == groupID {
			return true
		}
		if _, ok := visited[currentID]; ok {
			return false
		}
		visited[currentID] = struct{}{}
		current, ok := byID[currentID]
		if !ok || current.ParentID == nil {
			return false
		}
		currentID = *current.ParentID
	}
	return false
}

func kafkaStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}
