package service

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"
	apperrors "github.com/indu-forge/data_service/internal/errors"
)

// ProjectTenantBindingStore 定义项目租户绑定的最小持久化能力，便于隔离业务规则测试。
type ProjectTenantBindingStore interface {
	BindIfUnbound(ctx context.Context, projectID, tenantID string, authoringEpoch int64) (boundTenantID string, boundEpoch int64, created bool, err error)
	Get(ctx context.Context, projectID, tenantID string) (authoringEpoch int64, err error)
}

// ProjectTenantBindingService 保证项目只能首次绑定一个租户。
type ProjectTenantBindingService struct {
	store ProjectTenantBindingStore
}

// NewProjectTenantBindingService 创建项目租户绑定服务。
func NewProjectTenantBindingService(store ProjectTenantBindingStore) *ProjectTenantBindingService {
	return &ProjectTenantBindingService{store: store}
}

// Bind 创建项目租户归属；相同绑定可重试，任何改绑请求均返回冲突。
func (s *ProjectTenantBindingService) Bind(ctx context.Context, projectID, tenantID, authoringEpoch string) (created bool, err error) {
	if _, parseErr := uuid.Parse(strings.TrimSpace(projectID)); parseErr != nil {
		return false, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "projectId 格式无效", parseErr)
	}
	if _, parseErr := uuid.Parse(strings.TrimSpace(tenantID)); parseErr != nil {
		return false, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "tenantId 格式无效", parseErr)
	}

	epoch, parseErr := ParseAuthoringEpoch(authoringEpoch)
	if parseErr != nil {
		return false, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "authoringEpoch 格式无效")
	}
	boundTenantID, boundEpoch, created, err := s.store.BindIfUnbound(ctx, projectID, tenantID, epoch)
	if err != nil {
		return false, err
	}
	if boundTenantID != tenantID {
		// 不能向调用方暴露当前租户标识，避免内部接口错误被横向枚举利用。
		return false, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusConflict, "项目已绑定其他租户")
	}
	if boundEpoch != epoch {
		return false, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusConflict, "工程开发态代次与中心不一致")
	}
	return created, nil
}

func (s *ProjectTenantBindingService) Get(ctx context.Context, projectID, tenantID string) (string, error) {
	if _, err := uuid.Parse(strings.TrimSpace(projectID)); err != nil {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "projectId 格式无效")
	}
	if _, err := uuid.Parse(strings.TrimSpace(tenantID)); err != nil {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "tenantId 格式无效")
	}
	epoch, err := s.store.Get(ctx, projectID, tenantID)
	if err != nil {
		return "", err
	}
	return FormatAuthoringEpoch(epoch), nil
}
