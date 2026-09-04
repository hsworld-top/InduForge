package project

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/indu-forge/dev_core/internal/auth"
	dbsqlc "github.com/indu-forge/dev_core/internal/platform/db/sqlc"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgreSQLRepository struct {
	pool    *pgxpool.Pool
	queries *dbsqlc.Queries
}

func NewPostgreSQLRepository(pool *pgxpool.Pool) *PostgreSQLRepository {
	return &PostgreSQLRepository{pool: pool, queries: dbsqlc.New(pool)}
}

func (r *PostgreSQLRepository) List(ctx context.Context, tenantID string, filter ListFilter) ([]Project, int64, error) {
	tenantUUID, err := parseUUID(tenantID)
	if err != nil {
		return nil, 0, ErrNotFound
	}
	actorUUID, err := parseUUID(filter.ActorID)
	if err != nil {
		return nil, 0, ErrNotFound
	}
	params := dbsqlc.ListProjectsParams{TenantID: tenantUUID, IsPlatformAdmin: filter.IsPlatformAdmin, ActorID: actorUUID, CanReadShared: filter.CanReadShared, Keyword: filter.Keyword, Status: filter.Status, Visibility: filter.Visibility, GroupID: filter.GroupID, TagID: filter.TagID, PageOffset: int32((filter.Page - 1) * filter.Limit), PageLimit: int32(filter.Limit)}
	rows, err := r.queries.ListProjects(ctx, params)
	if err != nil {
		return nil, 0, fmt.Errorf("查询工程列表失败: %w", err)
	}
	total, err := r.queries.CountProjects(ctx, dbsqlc.CountProjectsParams{TenantID: tenantUUID, IsPlatformAdmin: filter.IsPlatformAdmin, ActorID: actorUUID, CanReadShared: filter.CanReadShared, Keyword: filter.Keyword, Status: filter.Status, Visibility: filter.Visibility, GroupID: filter.GroupID, TagID: filter.TagID})
	if err != nil {
		return nil, 0, fmt.Errorf("统计工程数量失败: %w", err)
	}
	items := make([]Project, 0, len(rows))
	for _, row := range rows {
		items = append(items, projectFromListRow(row))
	}
	if err := r.attachDeploymentSummaries(ctx, tenantID, items); err != nil {
		return nil, 0, fmt.Errorf("查询工程部署摘要失败: %w", err)
	}
	return items, total, nil
}

func (r *PostgreSQLRepository) Get(ctx context.Context, tenantID, projectID string) (Project, error) {
	tenantUUID, projectUUID, err := parsePair(tenantID, projectID)
	if err != nil {
		return Project{}, ErrNotFound
	}
	row, err := r.queries.GetProject(ctx, dbsqlc.GetProjectParams{ProjectID: projectUUID, TenantID: tenantUUID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Project{}, ErrNotFound
		}
		return Project{}, fmt.Errorf("查询工程失败: %w", err)
	}
	return projectFromModel(row), nil
}

func (r *PostgreSQLRepository) Create(ctx context.Context, item Project, actor auth.User, runtimeAdminHash string) (Project, error) {
	projectUUID, tenantUUID, err := parsePair(item.ID, item.TenantID)
	if err != nil {
		return Project{}, err
	}
	userUUID, err := parseUUID(actor.ID)
	if err != nil {
		return Project{}, err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Project{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := r.queries.WithTx(tx)
	created, err := queries.CreateProject(ctx, dbsqlc.CreateProjectParams{ProjectID: projectUUID, TenantID: tenantUUID, Name: item.Name, Code: item.Code, Description: nullableText(item.Description), Icon: nullableText(item.Icon), WorkspacePath: item.WorkspacePath, Visibility: item.Visibility, UserID: userUUID})
	if err != nil {
		return Project{}, mapConstraintError(err)
	}
	roleID, runtimeUserID := uuid.NewString(), uuid.NewString()
	if _, err := tx.Exec(ctx, `INSERT INTO project_roles (id, project_id, code, name, description, status, is_builtin, created_by) VALUES ($1,$2,'ADMIN','管理员','工程内置管理员','active',true,$3)`, roleID, item.ID, actor.ID); err != nil {
		return Project{}, fmt.Errorf("创建默认运行角色失败: %w", err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO project_role_grants (project_id, role_id, capability, effect) VALUES ($1,$2,'project.*','allow')`, item.ID, roleID); err != nil {
		return Project{}, fmt.Errorf("创建默认运行授权失败: %w", err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO project_runtime_users (id, project_id, username, password_hash, display_name, status, is_builtin_admin, created_by) VALUES ($1,$2,'admin',$3,'管理员','active',true,$4)`, runtimeUserID, item.ID, runtimeAdminHash, actor.ID); err != nil {
		return Project{}, fmt.Errorf("创建默认运行用户失败: %w", err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO project_user_role_bindings (runtime_user_id, role_id) VALUES ($1,$2)`, runtimeUserID, roleID); err != nil {
		return Project{}, fmt.Errorf("绑定默认运行角色失败: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Project{}, err
	}
	return projectFromModel(created), nil
}

func (r *PostgreSQLRepository) Update(ctx context.Context, item Project, actor auth.User) (Project, error) {
	projectUUID, tenantUUID, err := parsePair(item.ID, item.TenantID)
	if err != nil {
		return Project{}, ErrNotFound
	}
	userUUID, err := parseUUID(actor.ID)
	if err != nil {
		return Project{}, err
	}
	row, err := r.queries.UpdateProject(ctx, dbsqlc.UpdateProjectParams{Name: item.Name, Description: nullableText(item.Description), Icon: nullableText(item.Icon), Visibility: item.Visibility, UserID: userUUID, ProjectID: projectUUID, TenantID: tenantUUID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Project{}, ErrNotFound
		}
		return Project{}, mapConstraintError(err)
	}
	return projectFromModel(row), nil
}

func (r *PostgreSQLRepository) SetStatus(ctx context.Context, tenantID, projectID, status, userID string) (Project, error) {
	tenantUUID, projectUUID, err := parsePair(tenantID, projectID)
	if err != nil {
		return Project{}, ErrNotFound
	}
	userUUID, err := parseUUID(userID)
	if err != nil {
		return Project{}, err
	}
	row, err := r.queries.SetProjectLifecycleStatus(ctx, dbsqlc.SetProjectLifecycleStatusParams{Status: status, UserID: userUUID, ProjectID: projectUUID, TenantID: tenantUUID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Project{}, ErrNotFound
		}
		return Project{}, err
	}
	return projectFromModel(row), nil
}

func (r *PostgreSQLRepository) Delete(ctx context.Context, tenantID, projectID, userID string) error {
	tenantUUID, projectUUID, err := parsePair(tenantID, projectID)
	if err != nil {
		return ErrNotFound
	}
	userUUID, err := parseUUID(userID)
	if err != nil {
		return err
	}
	rows, err := r.queries.SoftDeleteProject(ctx, dbsqlc.SoftDeleteProjectParams{UserID: userUUID, ProjectID: projectUUID, TenantID: tenantUUID})
	if err != nil {
		return fmt.Errorf("删除工程失败: %w", err)
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgreSQLRepository) DeleteImpact(ctx context.Context, tenantID, projectID string) (DeleteImpact, error) {
	if _, err := r.Get(ctx, tenantID, projectID); err != nil {
		return DeleteImpact{}, err
	}
	projectUUID, _ := parseUUID(projectID)
	row, err := r.queries.GetProjectDeleteImpact(ctx, projectUUID)
	if err != nil {
		return DeleteImpact{}, err
	}
	return DeleteImpact{RuntimeUserCount: row.RuntimeUserCount, RuntimeRoleCount: row.RuntimeRoleCount, DeploymentCount: row.DeploymentCount, VersionCount: row.VersionCount}, nil
}

func (r *PostgreSQLRepository) ListTags(ctx context.Context, tenantID, keyword string) ([]Tag, error) {
	tenantUUID, err := parseUUID(tenantID)
	if err != nil {
		return nil, ErrNotFound
	}
	rows, err := r.queries.ListProjectTags(ctx, dbsqlc.ListProjectTagsParams{TenantID: tenantUUID, Keyword: keyword})
	if err != nil {
		return nil, err
	}
	items := make([]Tag, 0, len(rows))
	for _, row := range rows {
		items = append(items, tagFromListRow(row))
	}
	return items, nil
}
func (r *PostgreSQLRepository) GetTag(ctx context.Context, tenantID, tagID string) (Tag, error) {
	tenantUUID, tagUUID, err := parsePair(tenantID, tagID)
	if err != nil {
		return Tag{}, ErrTagNotFound
	}
	row, err := r.queries.GetProjectTag(ctx, dbsqlc.GetProjectTagParams{TagID: tagUUID, TenantID: tenantUUID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Tag{}, ErrTagNotFound
		}
		return Tag{}, err
	}
	return tagFromModel(row), nil
}
func (r *PostgreSQLRepository) CreateTag(ctx context.Context, tenantID, userID string, item Tag) (Tag, error) {
	tenantUUID, userUUID, err := parsePair(tenantID, userID)
	if err != nil {
		return Tag{}, err
	}
	row, err := r.queries.CreateProjectTag(ctx, dbsqlc.CreateProjectTagParams{TenantID: tenantUUID, Name: item.Name, Color: nullableText(item.Color), Description: nullableText(item.Description), SortOrder: item.SortOrder, UserID: userUUID})
	if err != nil {
		return Tag{}, mapConstraintError(err)
	}
	return tagFromModel(row), nil
}
func (r *PostgreSQLRepository) UpdateTag(ctx context.Context, tenantID string, item Tag) (Tag, error) {
	tenantUUID, tagUUID, err := parsePair(tenantID, item.ID)
	if err != nil {
		return Tag{}, ErrTagNotFound
	}
	row, err := r.queries.UpdateProjectTag(ctx, dbsqlc.UpdateProjectTagParams{Name: item.Name, Color: nullableText(item.Color), Description: nullableText(item.Description), SortOrder: item.SortOrder, TagID: tagUUID, TenantID: tenantUUID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Tag{}, ErrTagNotFound
		}
		return Tag{}, mapConstraintError(err)
	}
	return tagFromModel(row), nil
}
func (r *PostgreSQLRepository) DeleteTag(ctx context.Context, tenantID, tagID string) error {
	tenantUUID, tagUUID, err := parsePair(tenantID, tagID)
	if err != nil {
		return ErrTagNotFound
	}
	rows, err := r.queries.DeleteProjectTag(ctx, dbsqlc.DeleteProjectTagParams{TagID: tagUUID, TenantID: tenantUUID})
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrTagNotFound
	}
	return nil
}

func (r *PostgreSQLRepository) ReplaceTags(ctx context.Context, tenantID, projectID string, tagIDs []string) error {
	project, err := r.Get(ctx, tenantID, projectID)
	if err != nil {
		return err
	}
	projectUUID, _ := parseUUID(project.ID)
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := r.queries.WithTx(tx)
	if err := queries.DeleteProjectTagBindings(ctx, projectUUID); err != nil {
		return err
	}
	for _, tagID := range tagIDs {
		tag, err := r.GetTag(ctx, tenantID, tagID)
		if err != nil {
			return err
		}
		tagUUID, _ := parseUUID(tag.ID)
		if err := queries.CreateProjectTagBinding(ctx, dbsqlc.CreateProjectTagBindingParams{ProjectID: projectUUID, TagID: tagUUID}); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *PostgreSQLRepository) ListGroups(ctx context.Context, tenantID, keyword string) ([]Group, error) {
	tenantUUID, err := parseUUID(tenantID)
	if err != nil {
		return nil, ErrNotFound
	}
	rows, err := r.queries.ListProjectGroups(ctx, dbsqlc.ListProjectGroupsParams{TenantID: tenantUUID, Keyword: keyword})
	if err != nil {
		return nil, err
	}
	items := make([]Group, 0, len(rows))
	for _, row := range rows {
		items = append(items, groupFromListRow(row))
	}
	return items, nil
}
func (r *PostgreSQLRepository) GetGroup(ctx context.Context, tenantID, groupID string) (Group, error) {
	tenantUUID, groupUUID, err := parsePair(tenantID, groupID)
	if err != nil {
		return Group{}, ErrGroupNotFound
	}
	row, err := r.queries.GetProjectGroup(ctx, dbsqlc.GetProjectGroupParams{GroupID: groupUUID, TenantID: tenantUUID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Group{}, ErrGroupNotFound
		}
		return Group{}, err
	}
	return groupFromModel(row), nil
}
func (r *PostgreSQLRepository) CreateGroup(ctx context.Context, tenantID, userID string, item Group) (Group, error) {
	tenantUUID, userUUID, err := parsePair(tenantID, userID)
	if err != nil {
		return Group{}, err
	}
	row, err := r.queries.CreateProjectGroup(ctx, dbsqlc.CreateProjectGroupParams{TenantID: tenantUUID, Name: item.Name, Description: nullableText(item.Description), SortOrder: item.SortOrder, UserID: userUUID})
	if err != nil {
		return Group{}, mapConstraintError(err)
	}
	return groupFromModel(row), nil
}
func (r *PostgreSQLRepository) UpdateGroup(ctx context.Context, tenantID string, item Group) (Group, error) {
	tenantUUID, groupUUID, err := parsePair(tenantID, item.ID)
	if err != nil {
		return Group{}, ErrGroupNotFound
	}
	row, err := r.queries.UpdateProjectGroup(ctx, dbsqlc.UpdateProjectGroupParams{Name: item.Name, Description: nullableText(item.Description), SortOrder: item.SortOrder, GroupID: groupUUID, TenantID: tenantUUID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Group{}, ErrGroupNotFound
		}
		return Group{}, mapConstraintError(err)
	}
	return groupFromModel(row), nil
}
func (r *PostgreSQLRepository) DeleteGroup(ctx context.Context, tenantID, groupID string) error {
	tenantUUID, groupUUID, err := parsePair(tenantID, groupID)
	if err != nil {
		return ErrGroupNotFound
	}
	rows, err := r.queries.DeleteProjectGroup(ctx, dbsqlc.DeleteProjectGroupParams{GroupID: groupUUID, TenantID: tenantUUID})
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrGroupNotFound
	}
	return nil
}
func (r *PostgreSQLRepository) SetGroup(ctx context.Context, tenantID, projectID string, groupID *string) error {
	project, err := r.Get(ctx, tenantID, projectID)
	if err != nil {
		return err
	}
	projectUUID, _ := parseUUID(project.ID)
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := r.queries.WithTx(tx)
	if err := queries.DeleteProjectGroupBinding(ctx, projectUUID); err != nil {
		return err
	}
	if groupID != nil && *groupID != "" {
		group, err := r.GetGroup(ctx, tenantID, *groupID)
		if err != nil {
			return err
		}
		groupUUID, _ := parseUUID(group.ID)
		if err := queries.CreateProjectGroupBinding(ctx, dbsqlc.CreateProjectGroupBindingParams{ProjectID: projectUUID, GroupID: groupUUID}); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func projectFromModel(row dbsqlc.Project) Project {
	return Project{ID: uuidString(row.ID), TenantID: uuidString(row.TenantID), Name: row.Name, Code: row.Code, Description: textString(row.Description), Icon: textString(row.Icon), WorkspacePath: row.WorkspacePath, Status: row.Status, Visibility: row.Visibility, AuthoringEpoch: row.AuthoringEpoch, CreatedBy: uuidString(row.CreatedBy), UpdatedBy: uuidString(row.UpdatedBy), CreatedAt: row.CreatedAt.Time, UpdatedAt: row.UpdatedAt.Time}
}
func projectFromListRow(row dbsqlc.ListProjectsRow) Project {
	item := Project{ID: uuidString(row.ID), TenantID: uuidString(row.TenantID), Name: row.Name, Code: row.Code, Description: textString(row.Description), Icon: textString(row.Icon), WorkspacePath: row.WorkspacePath, Status: row.Status, Visibility: row.Visibility, AuthoringEpoch: row.AuthoringEpoch, CreatedBy: uuidString(row.CreatedBy), CreatedByName: row.CreatedByName, UpdatedBy: uuidString(row.UpdatedBy), CreatedAt: row.CreatedAt.Time, UpdatedAt: row.UpdatedAt.Time}
	if row.GroupID.Valid {
		item.Group = &Group{ID: uuidString(row.GroupID), Name: textString(row.GroupName)}
	}
	decodeTags(row.Tags, &item.Tags)
	return item
}

func decodeTags(value any, target *[]Tag) {
	if raw, ok := value.([]byte); ok {
		_ = json.Unmarshal(raw, target)
		return
	}
	raw, err := json.Marshal(value)
	if err == nil {
		_ = json.Unmarshal(raw, target)
	}
}
func tagFromModel(row dbsqlc.ProjectTag) Tag {
	return Tag{ID: uuidString(row.ID), TenantID: uuidString(row.TenantID), Name: row.Name, Color: textString(row.Color), Description: textString(row.Description), SortOrder: row.SortOrder, CreatedAt: row.CreatedAt.Time, UpdatedAt: row.UpdatedAt.Time}
}
func tagFromListRow(row dbsqlc.ListProjectTagsRow) Tag {
	return Tag{ID: uuidString(row.ID), TenantID: uuidString(row.TenantID), Name: row.Name, Color: textString(row.Color), Description: textString(row.Description), SortOrder: row.SortOrder, ProjectCount: row.ProjectCount, CreatedAt: row.CreatedAt.Time, UpdatedAt: row.UpdatedAt.Time}
}
func groupFromModel(row dbsqlc.ProjectGroup) Group {
	return Group{ID: uuidString(row.ID), TenantID: uuidString(row.TenantID), Name: row.Name, Description: textString(row.Description), SortOrder: row.SortOrder, CreatedAt: row.CreatedAt.Time, UpdatedAt: row.UpdatedAt.Time}
}
func groupFromListRow(row dbsqlc.ListProjectGroupsRow) Group {
	return Group{ID: uuidString(row.ID), TenantID: uuidString(row.TenantID), Name: row.Name, Description: textString(row.Description), SortOrder: row.SortOrder, ProjectCount: row.ProjectCount, CreatedAt: row.CreatedAt.Time, UpdatedAt: row.UpdatedAt.Time}
}
func parseUUID(value string) (pgtype.UUID, error) {
	parsed, err := uuid.Parse(value)
	if err != nil {
		return pgtype.UUID{}, err
	}
	return pgtype.UUID{Bytes: parsed, Valid: true}, nil
}
func parsePair(first, second string) (pgtype.UUID, pgtype.UUID, error) {
	left, err := parseUUID(first)
	if err != nil {
		return pgtype.UUID{}, pgtype.UUID{}, err
	}
	right, err := parseUUID(second)
	if err != nil {
		return pgtype.UUID{}, pgtype.UUID{}, err
	}
	return left, right, nil
}
func uuidString(value pgtype.UUID) string {
	if !value.Valid {
		return ""
	}
	return uuid.UUID(value.Bytes).String()
}
func textString(value pgtype.Text) string {
	if !value.Valid {
		return ""
	}
	return value.String
}
func nullableText(value string) pgtype.Text { return pgtype.Text{String: value, Valid: value != ""} }
func mapConstraintError(err error) error {
	var pgError *pgconn.PgError
	if errors.As(err, &pgError) && pgError.Code == "23505" {
		return ErrAlreadyExists
	}
	if errors.As(err, &pgError) && pgError.Code == "55000" {
		return ErrAuthoringBusy
	}
	return fmt.Errorf("保存工程资源失败: %w", err)
}
