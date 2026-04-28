package service

import (
	"context"
	"net/http"
	"strings"
	"time"

	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/indu-forge/data_service/internal/repository"
)

// ProjectSnapshotService 承载项目级数据域快照读写。
type ProjectSnapshotService struct {
	repository *repository.ProjectSnapshotRepository
}

// NewProjectSnapshotService 创建快照服务。
func NewProjectSnapshotService(repo *repository.ProjectSnapshotRepository) *ProjectSnapshotService {
	return &ProjectSnapshotService{repository: repo}
}

// Get 读取项目快照。
func (s *ProjectSnapshotService) Get(ctx context.Context, projectID string) (*repository.ProjectSnapshot, error) {
	if err := validateProjectID(projectID); err != nil {
		return nil, err
	}
	return s.repository.GetByProject(ctx, projectID)
}

// GetArtifact 基于项目快照生成数据域发布产物。
func (s *ProjectSnapshotService) GetArtifact(ctx context.Context, projectID string) (*repository.ProjectArtifactV1, error) {
	if err := validateProjectID(projectID); err != nil {
		return nil, err
	}
	snapshot, err := s.repository.GetByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return repository.BuildProjectArtifactV1(projectID, snapshot, time.Now().UTC()), nil
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
	return s.repository.ReplaceProjectData(ctx, projectID, actorID, normalizedSnapshot)
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
	case "relational", "mqtt", "kafka", "http", "websocket", "redis", "opcua", "modbus", "s7", "tdengine":
		return nil
	case "opcda":
		return newPhaseBoundaryProtocolError("OPC DA")
	default:
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "快照连接类型不受支持")
	}
}
