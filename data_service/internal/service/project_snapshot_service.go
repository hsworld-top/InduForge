package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/indu-forge/data_service/internal/repository"
)

// ProjectSnapshotService 承载项目级数据域快照读写。
type ProjectSnapshotService struct {
	repository *repository.ProjectSnapshotRepository
	now        func() time.Time
	schemaRoot string
}

// BuildCollectorArtifact 从权威快照生成 collector 运行定义；请求中的 sourceSnapshot
// 只能证明调用方读取的版本，绝不参与点位、连接或 secret 的生成。
func (s *ProjectSnapshotService) BuildCollectorArtifact(ctx context.Context, projectID, tenantID, artifactID string, revision int64, sourceSnapshot json.RawMessage) (map[string]any, error) {
	if err := validateProjectID(projectID); err != nil || revision < 1 || len(sourceSnapshot) == 0 {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "采集工件请求无效")
	}
	current, err := s.GetArtifact(ctx, projectID, tenantID)
	if err != nil {
		return nil, err
	}
	var provided map[string]any
	if json.Unmarshal(sourceSnapshot, &provided) != nil || provided["projectId"] != projectID {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "采集工件快照无效")
	}
	currentRaw, _ := json.Marshal(current)
	var currentMap map[string]any
	_ = json.Unmarshal(currentRaw, &currentMap)
	delete(currentMap, "generatedAt") // 构建时钟不是快照版本，不能造成伪冲突。
	delete(provided, "generatedAt")
	currentCanonical, _ := json.Marshal(currentMap)
	providedRaw, _ := json.Marshal(provided)
	if string(currentCanonical) != string(providedRaw) {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusConflict, "采集工件快照已变化")
	}
	snapshot, err := s.repository.GetByProject(ctx, projectID, tenantID)
	if err != nil {
		return nil, err
	}
	artifact, err := repository.BuildCollectorRuntimeArtifactV1(s.schemaRoot, projectID, artifactID, revision, "1.0.0", snapshot)
	if err != nil {
		return nil, err
	}
	raw, err := json.Marshal(artifact)
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(raw)
	return map[string]any{"schemaVersion": artifact.SchemaVersion, "projectId": projectID, "artifactRevision": revision, "sha256": "sha256:" + hex.EncodeToString(sum[:]), "size": len(raw), "artifact": json.RawMessage(raw)}, nil
}

// NewProjectSnapshotService 创建快照服务。
func NewProjectSnapshotService(repo *repository.ProjectSnapshotRepository) *ProjectSnapshotService {
	return NewProjectSnapshotServiceWithClock(repo, time.Now)
}

// NewProjectSnapshotServiceWithClock 允许发布调用方在测试和可重放构建中注入时钟。
func NewProjectSnapshotServiceWithClock(repo *repository.ProjectSnapshotRepository, now func() time.Time) *ProjectSnapshotService {
	return NewProjectSnapshotServiceWithClockAndSchemaRoot(repo, now, "")
}

// NewProjectSnapshotServiceWithClockAndSchemaRoot 同时注入时钟和受控 runtime schema 目录。
func NewProjectSnapshotServiceWithClockAndSchemaRoot(repo *repository.ProjectSnapshotRepository, now func() time.Time, schemaRoot string) *ProjectSnapshotService {
	if now == nil {
		now = time.Now
	}
	if strings.TrimSpace(schemaRoot) != "" {
		schemaRoot = filepath.Clean(schemaRoot)
	}
	return &ProjectSnapshotService{repository: repo, now: now, schemaRoot: schemaRoot}
}

// Get 读取项目快照。
func (s *ProjectSnapshotService) Get(ctx context.Context, projectID, tenantID string) (*repository.ProjectSnapshot, error) {
	if err := validateProjectID(projectID); err != nil {
		return nil, err
	}
	return s.repository.GetByProject(ctx, projectID, tenantID)
}

// GetArtifact 基于项目快照生成数据域发布产物。
func (s *ProjectSnapshotService) GetArtifact(ctx context.Context, projectID, tenantID string) (*repository.RuntimeProjectArtifactV1, error) {
	if err := validateProjectID(projectID); err != nil {
		return nil, err
	}
	snapshot, err := s.repository.GetByProject(ctx, projectID, tenantID)
	if err != nil {
		return nil, err
	}
	return repository.BuildRuntimeProjectArtifactV1(s.schemaRoot, projectID, snapshot, s.now().UTC())
}

// Replace 用快照内容覆盖项目数据域数据。
// 说明：snapshot/artifact 现在承载平台侧配置契约；工业协议仍只落配置，不在 data_service 内启动采集会话。
func (s *ProjectSnapshotService) Replace(ctx context.Context, projectID, actorID string, snapshot repository.ProjectSnapshot) error {
	if err := validateProjectID(projectID); err != nil {
		return err
	}
	if err := validateUserID(actorID); err != nil {
		return err
	}
	for _, record := range snapshot.Connections {
		if record.Type == "" || record.Name == "" {
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "快照连接数据不完整")
		}
		if err := validateSnapshotConnectionType(record.Type); err != nil {
			return err
		}
	}

	normalizedSnapshot := normalizeProjectSnapshot(snapshot)
	for index := range normalizedSnapshot.DataPoints {
		attributes, err := normalizeDataPointAttributeDefaults(normalizedSnapshot.DataPoints[index].AttributeDefaults)
		if err != nil {
			return err
		}
		normalizedSnapshot.DataPoints[index].AttributeDefaults = attributes
	}
	if err := validateSnapshotAlarmItems(normalizedSnapshot); err != nil {
		return err
	}
	return s.repository.ReplaceProjectData(ctx, projectID, actorID, normalizedSnapshot)
}

// validateSnapshotAlarmItems 保证快照不能绕过普通 API 的报警语义校验。
func validateSnapshotAlarmItems(snapshot repository.ProjectSnapshot) error {
	allowedSeverities, severityOrder := defaultAlarmSeverityPolicy()
	if snapshot.AlarmSettings != nil && len(snapshot.AlarmSettings.SeverityDefinitions) > 0 {
		allowedSeverities, severityOrder = map[string]bool{}, map[string]int{}
		for _, definition := range snapshot.AlarmSettings.SeverityDefinitions {
			allowedSeverities[definition.Key] = true
			severityOrder[definition.Key] = definition.SortOrder
		}
	}
	dataTypes := make(map[string]string, len(snapshot.DataPoints))
	for _, datapoint := range snapshot.DataPoints {
		dataTypes[datapoint.ID] = datapoint.DataType
	}
	for _, record := range snapshot.AlarmItems {
		input := SaveAlarmItemInput{DatapointID: valueString(record.DatapointID), DisplayName: record.DisplayName, Mode: record.Mode, EvaluationMode: record.EvaluationMode, DerivedExpression: record.DerivedExpression}
		for _, condition := range record.Conditions {
			input.Conditions = append(input.Conditions, AlarmCondition{ID: condition.ID, Kind: condition.Kind, Operator: condition.Operator, Label: condition.Label, Severity: condition.Severity, Params: condition.Params, TriggerDelayMS: condition.TriggerDelayMS, ClearDelayMS: condition.ClearDelayMS, Deadband: condition.Deadband})
		}
		if record.Mode == "point" {
			if record.DatapointID == nil {
				return badAlarm("快照中的普通报警必须关联一个数据点")
			}
			dataType, exists := dataTypes[*record.DatapointID]
			if !exists {
				return badAlarm("快照报警引用了不存在的数据点")
			}
			category := alarmDataCategory(dataType)
			if record.EvaluationMode == "single" && len(input.Conditions) != 1 {
				return badAlarm("快照普通单条件报警必须且只能有一个条件")
			}
			if _, err := normalizeConfigurationConditionsWithPolicy(input.Conditions, category, record.EvaluationMode, false, allowedSeverities, severityOrder); err != nil {
				return err
			}
		} else if record.Mode == "derived" {
			if len(record.Inputs) < 2 || strings.TrimSpace(record.DerivedExpression) == "" || record.EvaluationMode != "single" || len(input.Conditions) != 1 {
				return badAlarm("快照组合报警结构不完整")
			}
			seenPoints, seenKeys := map[string]bool{}, map[string]bool{}
			for _, item := range record.Inputs {
				if _, exists := dataTypes[item.DatapointID]; !exists || seenPoints[item.DatapointID] || !alarmInputKeyPattern.MatchString(strings.TrimSpace(item.InputKey)) || seenKeys[item.InputKey] {
					return badAlarm("快照组合报警输入点或别名不合法")
				}
				seenPoints[item.DatapointID], seenKeys[item.InputKey] = true, true
				input.Inputs = append(input.Inputs, AlarmItemInput{DatapointID: item.DatapointID, InputKey: item.InputKey, DataType: dataTypes[item.DatapointID]})
			}
			if err := validateDerivedAlarmExpression(record.DerivedExpression, seenKeys); err != nil {
				return err
			}
			if _, err := normalizeConfigurationConditionsWithPolicy(input.Conditions, "", "single", true, allowedSeverities, severityOrder); err != nil {
				return err
			}
			if err := validateDerivedAlarmExpressionTypes(record.DerivedExpression, input.Inputs, input.Conditions[0]); err != nil {
				return err
			}
		} else {
			return badAlarm("快照报警模式不受支持")
		}
		fingerprint, err := alarmTriggerFingerprint(input)
		if err != nil {
			return err
		}
		if record.TriggerFingerprint != fingerprint || record.NameKey != normalizeAlarmName(record.DisplayName) {
			return badAlarm("快照报警名称键或触发指纹不一致")
		}
	}
	return nil
}

func valueString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

// normalizeProjectSnapshot 在快照导入前补齐数据点运行态权限默认值。
// 关键边界：当 HTTP payload 明确传入 runtimePermissions 时保持原值；仅对完全缺省的场景补成 inherit=true。
func normalizeProjectSnapshot(snapshot repository.ProjectSnapshot) repository.ProjectSnapshot {
	if len(snapshot.DataPoints) == 0 {
		return snapshot
	}

	normalized := snapshot
	normalized.DataPoints = append([]repository.DataPointRecord(nil), snapshot.DataPoints...)
	for index := range normalized.DataPoints {
		if normalized.DataPoints[index].RuntimePermissionsDefined {
			continue
		}
		normalized.DataPoints[index].RuntimePermissions = repository.DefaultDataPointRuntimePermissions()
	}
	return normalized
}

func validateSnapshotConnectionType(connectionType string) error {
	switch strings.TrimSpace(strings.ToLower(connectionType)) {
	case "relational", "mqtt", "kafka", "http", "websocket", "redis", "tdengine",
		"builtin.relation", "builtin.timeseries", "builtin.realtime", "builtin.message":
		return nil
	case "opcda":
		return newPhaseBoundaryProtocolError("OPC DA")
	default:
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "快照连接类型不受支持")
	}
}
