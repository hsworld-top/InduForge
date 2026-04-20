package service

import (
	"context"
	"net/http"

	"github.com/indu-forge/data_service/internal/repository"
	apperrors "github.com/indu-forge/data_service/internal/errors"
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

// Replace 用快照内容覆盖项目数据域数据。
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
	}
	return s.repository.ReplaceProjectData(ctx, projectID, actorID, snapshot)
}
