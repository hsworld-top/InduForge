package runtimeaccess

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
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

func (r *PostgreSQLRepository) GetProjectAccess(ctx context.Context, tenantID, projectID string) (ProjectAccess, error) {
	tenantUUID, projectUUID, err := parsePair(tenantID, projectID)
	if err != nil {
		return ProjectAccess{}, ErrProjectNotFound
	}
	row, err := r.queries.GetProject(ctx, dbsqlc.GetProjectParams{ProjectID: projectUUID, TenantID: tenantUUID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ProjectAccess{}, ErrProjectNotFound
		}
		return ProjectAccess{}, err
	}
	return ProjectAccess{ID: uuidString(row.ID), TenantID: uuidString(row.TenantID), CreatedBy: uuidString(row.CreatedBy), Visibility: row.Visibility}, nil
}

func (r *PostgreSQLRepository) ListRoles(ctx context.Context, tenantID, projectID string) ([]Role, error) {
	tenantUUID, projectUUID, err := parsePair(tenantID, projectID)
	if err != nil {
		return nil, ErrRoleNotFound
	}
	rows, err := r.queries.ListRuntimeRoles(ctx, dbsqlc.ListRuntimeRolesParams{ProjectID: projectUUID, TenantID: tenantUUID})
	if err != nil {
		return nil, err
	}
	items := make([]Role, 0, len(rows))
	for _, row := range rows {
		items = append(items, roleFromList(row))
	}
	return items, nil
}
func (r *PostgreSQLRepository) GetRole(ctx context.Context, tenantID, projectID, roleID string) (Role, error) {
	tenantUUID, projectUUID, roleUUID, err := parseTriple(tenantID, projectID, roleID)
	if err != nil {
		return Role{}, ErrRoleNotFound
	}
	row, err := r.queries.GetRuntimeRole(ctx, dbsqlc.GetRuntimeRoleParams{RoleID: roleUUID, ProjectID: projectUUID, TenantID: tenantUUID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Role{}, ErrRoleNotFound
		}
		return Role{}, err
	}
	return roleFromModel(row), nil
}
func (r *PostgreSQLRepository) CreateRole(ctx context.Context, projectID, userID string, input RoleInput) (Role, error) {
	projectUUID, userUUID, err := parsePair(projectID, userID)
	if err != nil {
		return Role{}, err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Role{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := r.queries.WithTx(tx)
	row, err := q.CreateRuntimeRole(ctx, dbsqlc.CreateRuntimeRoleParams{ProjectID: projectUUID, Code: input.Code, Name: input.Name, Description: nullableText(input.Description), Status: input.Status, UserID: userUUID})
	if err != nil {
		return Role{}, mapConstraintError(err)
	}
	if err := replaceRoleGrants(ctx, q, projectUUID, row.ID, input.Capabilities); err != nil {
		return Role{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Role{}, err
	}
	item := roleFromModel(row)
	item.Capabilities = input.Capabilities
	return item, nil
}
func (r *PostgreSQLRepository) UpdateRole(ctx context.Context, projectID, roleID, userID string, input RoleInput) (Role, error) {
	projectUUID, roleUUID, userUUID, err := parseTriple(projectID, roleID, userID)
	if err != nil {
		return Role{}, ErrRoleNotFound
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Role{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := r.queries.WithTx(tx)
	row, err := q.UpdateRuntimeRole(ctx, dbsqlc.UpdateRuntimeRoleParams{Code: input.Code, Name: input.Name, Description: nullableText(input.Description), Status: input.Status, UserID: userUUID, RoleID: roleUUID, ProjectID: projectUUID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Role{}, ErrRoleNotFound
		}
		return Role{}, mapConstraintError(err)
	}
	// 未提交权限字段时只修改角色资料，显式空数组才清空权限。
	if input.Capabilities != nil {
		if err := replaceRoleGrants(ctx, q, projectUUID, roleUUID, input.Capabilities); err != nil {
			return Role{}, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return Role{}, err
	}
	item := roleFromModel(row)
	item.Capabilities = input.Capabilities
	return item, nil
}
func (r *PostgreSQLRepository) DeleteRole(ctx context.Context, tenantID, projectID, roleID string) error {
	if _, err := r.GetRole(ctx, tenantID, projectID, roleID); err != nil {
		return err
	}
	projectUUID, roleUUID, err := parsePair(projectID, roleID)
	if err != nil {
		return ErrRoleNotFound
	}
	rows, err := r.queries.DeleteRuntimeRole(ctx, dbsqlc.DeleteRuntimeRoleParams{RoleID: roleUUID, ProjectID: projectUUID})
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrBuiltin
	}
	return nil
}
func (r *PostgreSQLRepository) ListUsers(ctx context.Context, tenantID, projectID string) ([]RuntimeUser, error) {
	tenantUUID, projectUUID, err := parsePair(tenantID, projectID)
	if err != nil {
		return nil, ErrUserNotFound
	}
	rows, err := r.queries.ListRuntimeUsers(ctx, dbsqlc.ListRuntimeUsersParams{ProjectID: projectUUID, TenantID: tenantUUID})
	if err != nil {
		return nil, err
	}
	items := make([]RuntimeUser, 0, len(rows))
	for _, row := range rows {
		items = append(items, userFromList(row))
	}
	return items, nil
}
func (r *PostgreSQLRepository) GetUser(ctx context.Context, tenantID, projectID, userID string) (RuntimeUser, error) {
	tenantUUID, projectUUID, userUUID, err := parseTriple(tenantID, projectID, userID)
	if err != nil {
		return RuntimeUser{}, ErrUserNotFound
	}
	row, err := r.queries.GetRuntimeUser(ctx, dbsqlc.GetRuntimeUserParams{RuntimeUserID: userUUID, ProjectID: projectUUID, TenantID: tenantUUID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return RuntimeUser{}, ErrUserNotFound
		}
		return RuntimeUser{}, err
	}
	return userFromModel(row), nil
}
func (r *PostgreSQLRepository) CreateUser(ctx context.Context, projectID, userID string, input UserInput, passwordHash string) (RuntimeUser, error) {
	projectUUID, actorUUID, err := parsePair(projectID, userID)
	if err != nil {
		return RuntimeUser{}, err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return RuntimeUser{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := r.queries.WithTx(tx)
	row, err := q.CreateRuntimeUser(ctx, dbsqlc.CreateRuntimeUserParams{ProjectID: projectUUID, Username: input.Username, PasswordHash: passwordHash, DisplayName: nullableText(input.DisplayName), Email: nullableText(input.Email), Status: input.Status, UserID: actorUUID})
	if err != nil {
		return RuntimeUser{}, mapConstraintError(err)
	}
	if err := replaceUserRoles(ctx, q, row.ID, projectUUID, input.RoleIDs); err != nil {
		return RuntimeUser{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return RuntimeUser{}, err
	}
	item := userFromModel(row)
	return item, nil
}
func (r *PostgreSQLRepository) UpdateUserStatus(ctx context.Context, tenantID, projectID, userID, status, actorID string) (RuntimeUser, error) {
	if _, err := r.GetUser(ctx, tenantID, projectID, userID); err != nil {
		return RuntimeUser{}, err
	}
	projectUUID, userUUID, actorUUID, err := parseTriple(projectID, userID, actorID)
	if err != nil {
		return RuntimeUser{}, ErrUserNotFound
	}
	row, err := r.queries.UpdateRuntimeUserStatus(ctx, dbsqlc.UpdateRuntimeUserStatusParams{Status: status, UserID: actorUUID, RuntimeUserID: userUUID, ProjectID: projectUUID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return RuntimeUser{}, ErrUserNotFound
		}
		return RuntimeUser{}, err
	}
	return userFromModel(row), nil
}
func (r *PostgreSQLRepository) DeleteUser(ctx context.Context, tenantID, projectID, userID string) error {
	if _, err := r.GetUser(ctx, tenantID, projectID, userID); err != nil {
		return err
	}
	projectUUID, userUUID, err := parsePair(projectID, userID)
	if err != nil {
		return ErrUserNotFound
	}
	rows, err := r.queries.DeleteRuntimeUser(ctx, dbsqlc.DeleteRuntimeUserParams{RuntimeUserID: userUUID, ProjectID: projectUUID})
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrBuiltin
	}
	return nil
}
func (r *PostgreSQLRepository) ReplaceUserRoles(ctx context.Context, tenantID, projectID, userID string, roleIDs []string) error {
	if _, err := r.GetUser(ctx, tenantID, projectID, userID); err != nil {
		return err
	}
	projectUUID, userUUID, err := parsePair(projectID, userID)
	if err != nil {
		return ErrUserNotFound
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := r.queries.WithTx(tx)
	if err := replaceUserRoles(ctx, q, userUUID, projectUUID, roleIDs); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
func (r *PostgreSQLRepository) UpdateUserPassword(ctx context.Context, tenantID, projectID, userID, actorID, passwordHash string) error {
	if _, err := r.GetUser(ctx, tenantID, projectID, userID); err != nil {
		return err
	}
	projectUUID, userUUID, actorUUID, err := parseTriple(projectID, userID, actorID)
	if err != nil {
		return ErrUserNotFound
	}
	return r.queries.UpdateRuntimeUserPassword(ctx, dbsqlc.UpdateRuntimeUserPasswordParams{PasswordHash: passwordHash, UserID: actorUUID, RuntimeUserID: userUUID, ProjectID: projectUUID})
}
func replaceRoleGrants(ctx context.Context, q *dbsqlc.Queries, projectID, roleID pgtype.UUID, capabilities []string) error {
	if err := q.DeleteRuntimeRoleGrants(ctx, roleID); err != nil {
		return err
	}
	for _, capability := range capabilities {
		capability = strings.TrimSpace(capability)
		if capability == "" {
			continue
		}
		if err := q.CreateRuntimeRoleGrant(ctx, dbsqlc.CreateRuntimeRoleGrantParams{ProjectID: projectID, RoleID: roleID, Capability: capability}); err != nil {
			return err
		}
	}
	return nil
}
func replaceUserRoles(ctx context.Context, q *dbsqlc.Queries, userID, projectID pgtype.UUID, roleIDs []string) error {
	if err := q.DeleteRuntimeUserRoleBindings(ctx, userID); err != nil {
		return err
	}
	for _, roleID := range roleIDs {
		roleUUID, err := parseUUID(roleID)
		if err != nil {
			return ErrRoleNotFound
		}
		exists, err := q.CheckRuntimeRoleInProject(ctx, dbsqlc.CheckRuntimeRoleInProjectParams{RoleID: roleUUID, ProjectID: projectID})
		if err != nil || !exists {
			return ErrRoleNotFound
		}
		if err := q.CreateRuntimeUserRoleBinding(ctx, dbsqlc.CreateRuntimeUserRoleBindingParams{RuntimeUserID: userID, RoleID: roleUUID}); err != nil {
			return err
		}
	}
	return nil
}
func roleFromModel(row dbsqlc.ProjectRole) Role {
	return Role{ID: uuidString(row.ID), ProjectID: uuidString(row.ProjectID), Code: row.Code, Name: row.Name, Description: textString(row.Description), Status: row.Status, IsBuiltin: row.IsBuiltin, CreatedAt: row.CreatedAt.Time, UpdatedAt: row.UpdatedAt.Time}
}
func roleFromList(row dbsqlc.ListRuntimeRolesRow) Role {
	item := Role{ID: uuidString(row.ID), ProjectID: uuidString(row.ProjectID), Code: row.Code, Name: row.Name, Description: textString(row.Description), Status: row.Status, IsBuiltin: row.IsBuiltin, UserCount: row.UserCount, CreatedAt: row.CreatedAt.Time, UpdatedAt: row.UpdatedAt.Time}
	decodeJSONValue(row.Capabilities, &item.Capabilities)
	return item
}
func userFromModel(row dbsqlc.ProjectRuntimeUser) RuntimeUser {
	return RuntimeUser{ID: uuidString(row.ID), ProjectID: uuidString(row.ProjectID), Username: row.Username, DisplayName: textString(row.DisplayName), Email: textString(row.Email), Status: row.Status, IsBuiltinAdmin: row.IsBuiltinAdmin, LastLoginAt: timePointer(row.LastLoginAt), PasswordChangedAt: timePointer(row.PasswordChangedAt), CreatedAt: row.CreatedAt.Time, UpdatedAt: row.UpdatedAt.Time}
}
func userFromList(row dbsqlc.ListRuntimeUsersRow) RuntimeUser {
	item := RuntimeUser{ID: uuidString(row.ID), ProjectID: uuidString(row.ProjectID), Username: row.Username, DisplayName: textString(row.DisplayName), Email: textString(row.Email), Status: row.Status, IsBuiltinAdmin: row.IsBuiltinAdmin, LastLoginAt: timePointer(row.LastLoginAt), PasswordChangedAt: timePointer(row.PasswordChangedAt), CreatedAt: row.CreatedAt.Time, UpdatedAt: row.UpdatedAt.Time}
	decodeJSONValue(row.Roles, &item.Roles)
	return item
}
func decodeJSONValue(value any, target any) {
	if raw, ok := value.([]byte); ok {
		_ = json.Unmarshal(raw, target)
		return
	}
	raw, err := json.Marshal(value)
	if err == nil {
		_ = json.Unmarshal(raw, target)
	}
}
func parseUUID(value string) (pgtype.UUID, error) {
	parsed, err := uuid.Parse(value)
	if err != nil {
		return pgtype.UUID{}, err
	}
	return pgtype.UUID{Bytes: parsed, Valid: true}, nil
}
func parsePair(a, b string) (pgtype.UUID, pgtype.UUID, error) {
	x, err := parseUUID(a)
	if err != nil {
		return pgtype.UUID{}, pgtype.UUID{}, err
	}
	y, err := parseUUID(b)
	if err != nil {
		return pgtype.UUID{}, pgtype.UUID{}, err
	}
	return x, y, nil
}
func parseTriple(a, b, c string) (pgtype.UUID, pgtype.UUID, pgtype.UUID, error) {
	x, y, err := parsePair(a, b)
	if err != nil {
		return pgtype.UUID{}, pgtype.UUID{}, pgtype.UUID{}, err
	}
	z, err := parseUUID(c)
	if err != nil {
		return pgtype.UUID{}, pgtype.UUID{}, pgtype.UUID{}, err
	}
	return x, y, z, nil
}
func uuidString(v pgtype.UUID) string {
	if !v.Valid {
		return ""
	}
	return uuid.UUID(v.Bytes).String()
}
func textString(v pgtype.Text) string {
	if !v.Valid {
		return ""
	}
	return v.String
}
func nullableText(v string) pgtype.Text { return pgtype.Text{String: v, Valid: v != ""} }
func timePointer(v pgtype.Timestamptz) *time.Time {
	if !v.Valid {
		return nil
	}
	x := v.Time
	return &x
}
func mapConstraintError(err error) error {
	var pgError *pgconn.PgError
	if errors.As(err, &pgError) && pgError.Code == "23505" {
		return ErrAlreadyExists
	}
	return fmt.Errorf("保存运行权限失败: %w", err)
}

func (r *PostgreSQLRepository) PageRoles(ctx context.Context, tenantID, projectID string, query ListQuery) ([]Role, int64, error) {
	tenantUUID, projectUUID, err := parsePair(tenantID, projectID)
	if err != nil {
		return nil, 0, err
	}
	total, err := r.queries.CountRuntimeRoles(ctx, dbsqlc.CountRuntimeRolesParams{TenantID: tenantUUID, ProjectID: projectUUID, Keyword: query.Keyword})
	if err != nil {
		return nil, 0, err
	}
	rows, err := r.queries.PageRuntimeRoles(ctx, dbsqlc.PageRuntimeRolesParams{TenantID: tenantUUID, ProjectID: projectUUID, Keyword: query.Keyword, PageLimit: int32(query.Limit), PageOffset: int32((query.Page - 1) * query.Limit)})
	if err != nil {
		return nil, 0, err
	}
	items := make([]Role, 0, len(rows))
	for _, row := range rows {
		items = append(items, roleFromList(dbsqlc.ListRuntimeRolesRow(row)))
	}
	return items, total, nil
}

func (r *PostgreSQLRepository) PageUsers(ctx context.Context, tenantID, projectID string, query ListQuery) ([]RuntimeUser, int64, error) {
	tenantUUID, projectUUID, err := parsePair(tenantID, projectID)
	if err != nil {
		return nil, 0, err
	}
	total, err := r.queries.CountRuntimeUsers(ctx, dbsqlc.CountRuntimeUsersParams{TenantID: tenantUUID, ProjectID: projectUUID, Keyword: query.Keyword})
	if err != nil {
		return nil, 0, err
	}
	rows, err := r.queries.PageRuntimeUsers(ctx, dbsqlc.PageRuntimeUsersParams{TenantID: tenantUUID, ProjectID: projectUUID, Keyword: query.Keyword, PageLimit: int32(query.Limit), PageOffset: int32((query.Page - 1) * query.Limit)})
	if err != nil {
		return nil, 0, err
	}
	items := make([]RuntimeUser, 0, len(rows))
	for _, row := range rows {
		items = append(items, userFromList(dbsqlc.ListRuntimeUsersRow(row)))
	}
	return items, total, nil
}
