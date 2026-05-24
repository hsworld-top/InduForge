package service

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/indu-forge/data_service/internal/repository"
)

const (
	WorkbenchScopeQuery = "query"
	WorkbenchScopeTable = "table"
)

// WorkbenchObjectGroup 是前端 SQL 工作台展示查询/表分组的对象。
type WorkbenchObjectGroup struct {
	ID           string    `json:"id"`
	ProjectID    string    `json:"projectId"`
	ConnectionID string    `json:"connectionId"`
	Scope        string    `json:"scope"`
	Name         string    `json:"name"`
	SortOrder    int       `json:"sortOrder"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// TableGroupMember 表示真实表名与工作台展示分组的关系。
type TableGroupMember struct {
	TableName string `json:"tableName"`
	GroupID   string `json:"groupId"`
}

// WorkbenchGroupService 负责 SQL 工作台对象分组的校验与持久化。
type WorkbenchGroupService struct {
	repository  *repository.WorkbenchGroupRepository
	connections *repository.ConnectionRepository
}

func NewWorkbenchGroupService(repo *repository.WorkbenchGroupRepository, connectionRepo *repository.ConnectionRepository) *WorkbenchGroupService {
	return &WorkbenchGroupService{repository: repo, connections: connectionRepo}
}

func (s *WorkbenchGroupService) ListGroups(ctx context.Context, projectID, connectionID, scope string) ([]WorkbenchObjectGroup, error) {
	scope, err := normalizeWorkbenchScope(scope)
	if err != nil {
		return nil, err
	}
	if err := s.ensureConnection(ctx, projectID, connectionID); err != nil {
		return nil, err
	}
	records, err := s.repository.ListGroups(ctx, projectID, connectionID, scope)
	if err != nil {
		return nil, err
	}
	groups := make([]WorkbenchObjectGroup, 0, len(records))
	for _, record := range records {
		groups = append(groups, toWorkbenchGroup(record))
	}
	return groups, nil
}

func (s *WorkbenchGroupService) CreateGroup(ctx context.Context, projectID, connectionID, scope, userID, name string) (*WorkbenchObjectGroup, error) {
	scope, err := normalizeWorkbenchScope(scope)
	if err != nil {
		return nil, err
	}
	name, err = normalizeWorkbenchGroupName(name)
	if err != nil {
		return nil, err
	}
	if err := validateUserID(userID); err != nil {
		return nil, err
	}
	if err := s.ensureConnection(ctx, projectID, connectionID); err != nil {
		return nil, err
	}
	record, err := s.repository.CreateGroup(ctx, repository.CreateWorkbenchGroupParams{
		ProjectID:    projectID,
		ConnectionID: connectionID,
		Scope:        scope,
		Name:         name,
		UserID:       userID,
	})
	if err != nil {
		return nil, err
	}
	group := toWorkbenchGroup(*record)
	return &group, nil
}

func (s *WorkbenchGroupService) UpdateGroup(ctx context.Context, projectID, groupID, userID, name string) (*WorkbenchObjectGroup, error) {
	if err := validateProjectID(projectID); err != nil {
		return nil, err
	}
	if err := validateWorkbenchGroupID(groupID); err != nil {
		return nil, err
	}
	if err := validateUserID(userID); err != nil {
		return nil, err
	}
	name, err := normalizeWorkbenchGroupName(name)
	if err != nil {
		return nil, err
	}
	record, err := s.repository.UpdateGroup(ctx, repository.UpdateWorkbenchGroupParams{
		ProjectID: projectID,
		ID:        groupID,
		Name:      name,
		UserID:    userID,
	})
	if err != nil {
		return nil, err
	}
	group := toWorkbenchGroup(*record)
	return &group, nil
}

func (s *WorkbenchGroupService) DeleteGroup(ctx context.Context, projectID, groupID string) error {
	if err := validateProjectID(projectID); err != nil {
		return err
	}
	if err := validateWorkbenchGroupID(groupID); err != nil {
		return err
	}
	return s.repository.DeleteGroup(ctx, projectID, groupID)
}

func (s *WorkbenchGroupService) MoveQuery(ctx context.Context, projectID, queryID, userID string, groupID *string) error {
	if err := validateProjectID(projectID); err != nil {
		return err
	}
	if err := validateQueryID(queryID); err != nil {
		return err
	}
	if err := validateUserID(userID); err != nil {
		return err
	}
	normalizedGroupID, err := s.normalizeOptionalGroup(ctx, projectID, groupID, WorkbenchScopeQuery)
	if err != nil {
		return err
	}
	return s.repository.MoveQuery(ctx, projectID, queryID, normalizedGroupID, userID)
}

func (s *WorkbenchGroupService) ListTableMembers(ctx context.Context, projectID, connectionID string) ([]TableGroupMember, error) {
	if err := s.ensureConnection(ctx, projectID, connectionID); err != nil {
		return nil, err
	}
	records, err := s.repository.ListTableMembers(ctx, projectID, connectionID)
	if err != nil {
		return nil, err
	}
	members := make([]TableGroupMember, 0, len(records))
	for _, record := range records {
		members = append(members, TableGroupMember{TableName: record.TableName, GroupID: record.GroupID})
	}
	return members, nil
}

func (s *WorkbenchGroupService) MoveTable(ctx context.Context, projectID, connectionID, userID, tableName string, groupID *string) error {
	if err := s.ensureConnection(ctx, projectID, connectionID); err != nil {
		return err
	}
	if err := validateUserID(userID); err != nil {
		return err
	}
	tableName = strings.TrimSpace(tableName)
	if tableName == "" {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "表名不能为空")
	}
	if len([]rune(tableName)) > 255 {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "表名长度不能超过 255 个字符")
	}
	normalizedGroupID, err := s.normalizeOptionalGroup(ctx, projectID, groupID, WorkbenchScopeTable)
	if err != nil {
		return err
	}
	return s.repository.MoveTable(ctx, repository.MoveTableGroupParams{
		ProjectID:    projectID,
		ConnectionID: connectionID,
		TableName:    tableName,
		GroupID:      normalizedGroupID,
		UserID:       userID,
	})
}

func (s *WorkbenchGroupService) RenameTableMember(ctx context.Context, projectID, connectionID, oldName, newName, userID string) error {
	if err := s.ensureConnection(ctx, projectID, connectionID); err != nil {
		return err
	}
	if err := validateUserID(userID); err != nil {
		return err
	}
	oldName = strings.TrimSpace(oldName)
	newName = strings.TrimSpace(newName)
	if oldName == "" || newName == "" {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "表名不能为空")
	}
	return s.repository.RenameTableMember(ctx, projectID, connectionID, oldName, newName, userID)
}

func (s *WorkbenchGroupService) DeleteTableMember(ctx context.Context, projectID, connectionID, tableName string) error {
	if err := s.ensureConnection(ctx, projectID, connectionID); err != nil {
		return err
	}
	tableName = strings.TrimSpace(tableName)
	if tableName == "" {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "表名不能为空")
	}
	return s.repository.DeleteTableMember(ctx, projectID, connectionID, tableName)
}

func (s *WorkbenchGroupService) normalizeOptionalGroup(ctx context.Context, projectID string, groupID *string, scope string) (*string, error) {
	if groupID == nil || strings.TrimSpace(*groupID) == "" {
		return nil, nil
	}
	normalized := strings.TrimSpace(*groupID)
	if err := validateWorkbenchGroupID(normalized); err != nil {
		return nil, err
	}
	group, err := s.repository.GetGroup(ctx, projectID, normalized)
	if err != nil {
		return nil, err
	}
	if group.Scope != scope {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "分组类型不匹配")
	}
	return &normalized, nil
}

func (s *WorkbenchGroupService) ensureConnection(ctx context.Context, projectID, connectionID string) error {
	if err := validateProjectID(projectID); err != nil {
		return err
	}
	if err := validateConnectionID(connectionID); err != nil {
		return err
	}
	_, err := s.connections.GetByProjectAndID(ctx, projectID, connectionID)
	return err
}

func normalizeWorkbenchScope(scope string) (string, error) {
	scope = strings.TrimSpace(strings.ToLower(scope))
	if scope != WorkbenchScopeQuery && scope != WorkbenchScopeTable {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "分组类型仅支持查询或表")
	}
	return scope, nil
}

func normalizeWorkbenchGroupName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "分组名称不能为空")
	}
	if len([]rune(name)) > 100 {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "分组名称长度不能超过 100 个字符")
	}
	return name, nil
}

func validateWorkbenchGroupID(groupID string) error {
	if _, err := uuid.Parse(strings.TrimSpace(groupID)); err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "groupId 格式无效", err)
	}
	return nil
}

func toWorkbenchGroup(record repository.WorkbenchObjectGroupRecord) WorkbenchObjectGroup {
	return WorkbenchObjectGroup{
		ID:           record.ID,
		ProjectID:    record.ProjectID,
		ConnectionID: record.ConnectionID,
		Scope:        record.Scope,
		Name:         record.Name,
		SortOrder:    record.SortOrder,
		CreatedAt:    record.CreatedAt,
		UpdatedAt:    record.UpdatedAt,
	}
}
