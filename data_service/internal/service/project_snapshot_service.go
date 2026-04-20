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

// GetArtifact 基于项目快照生成 Phase 1 产物。
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
// 说明：Phase 1 在 snapshot 层只接受正式协议范围，避免通过导入入口重新把 Phase 2 预留协议写回库内。
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
	return s.repository.ReplaceProjectData(ctx, projectID, actorID, snapshot)
}

func validateSnapshotConnectionType(connectionType string) error {
	switch strings.TrimSpace(strings.ToLower(connectionType)) {
	case "relational", "mqtt", "kafka", "http", "websocket", "redis":
		return nil
	case "opcua":
		return newPhaseBoundaryProtocolError("OPC UA")
	case "modbus":
		return newPhaseBoundaryProtocolError("Modbus")
	case "s7":
		return newPhaseBoundaryProtocolError("S7")
	case "tdengine":
		return newPhaseBoundaryProtocolError("TDengine")
	case "opcda":
		return newPhaseBoundaryProtocolError("OPC DA")
	default:
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "快照连接类型不受支持")
	}
}
