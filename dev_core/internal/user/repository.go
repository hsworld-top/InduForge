package user

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	dbsqlc "github.com/indu-forge/dev_core/internal/platform/db/sqlc"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgreSQLRepository struct{ queries *dbsqlc.Queries }

func NewPostgreSQLRepository(pool *pgxpool.Pool) *PostgreSQLRepository {
	return &PostgreSQLRepository{queries: dbsqlc.New(pool)}
}

func (r *PostgreSQLRepository) List(ctx context.Context, tenantID string, filter ListFilter) ([]User, int64, error) {
	id, err := parseUUID(tenantID)
	if err != nil {
		return nil, 0, ErrNotFound
	}
	params := dbsqlc.ListUsersParams{TenantID: id, Keyword: filter.Keyword, Role: filter.Role, Status: filter.Status, PageOffset: int32((filter.Page - 1) * filter.Limit), PageLimit: int32(filter.Limit)}
	rows, err := r.queries.ListUsers(ctx, params)
	if err != nil {
		return nil, 0, fmt.Errorf("查询用户列表失败: %w", err)
	}
	total, err := r.queries.CountUsers(ctx, dbsqlc.CountUsersParams{TenantID: id, Keyword: filter.Keyword, Role: filter.Role, Status: filter.Status})
	if err != nil {
		return nil, 0, fmt.Errorf("统计用户数量失败: %w", err)
	}
	items := make([]User, 0, len(rows))
	for _, row := range rows {
		items = append(items, userFromListRow(row))
	}
	return items, total, nil
}

func (r *PostgreSQLRepository) Get(ctx context.Context, tenantID, userID string) (User, string, error) {
	tenantUUID, userUUID, err := parsePair(tenantID, userID)
	if err != nil {
		return User{}, "", ErrNotFound
	}
	row, err := r.queries.GetManagedUser(ctx, dbsqlc.GetManagedUserParams{UserID: userUUID, TenantID: tenantUUID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, "", ErrNotFound
		}
		return User{}, "", fmt.Errorf("查询用户失败: %w", err)
	}
	return userFromModel(row), row.PasswordHash, nil
}

func (r *PostgreSQLRepository) Create(ctx context.Context, item User, passwordHash string) (User, error) {
	tenantID, err := parseUUID(item.TenantID)
	if err != nil {
		return User{}, ErrNotFound
	}
	row, err := r.queries.CreateManagedUser(ctx, dbsqlc.CreateManagedUserParams{TenantID: tenantID, Username: item.Username, PasswordHash: passwordHash, Email: nullableText(item.Email), Phone: nullableText(item.Phone), FullName: nullableText(item.FullName), Avatar: nullableText(item.Avatar), Role: item.Role, Status: item.Status, Preferences: marshalMap(item.Preferences)})
	if err != nil {
		return User{}, mapConstraintError(err)
	}
	return userFromModel(row), nil
}

func (r *PostgreSQLRepository) Update(ctx context.Context, item User) (User, error) {
	tenantID, userID, err := parsePair(item.TenantID, item.ID)
	if err != nil {
		return User{}, ErrNotFound
	}
	row, err := r.queries.UpdateManagedUser(ctx, dbsqlc.UpdateManagedUserParams{Email: nullableText(item.Email), Phone: nullableText(item.Phone), FullName: nullableText(item.FullName), Avatar: nullableText(item.Avatar), Role: item.Role, Status: item.Status, Preferences: marshalMap(item.Preferences), UserID: userID, TenantID: tenantID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrNotFound
		}
		return User{}, mapConstraintError(err)
	}
	return userFromModel(row), nil
}

func (r *PostgreSQLRepository) UpdatePassword(ctx context.Context, tenantID, userID, passwordHash string) error {
	tenantUUID, userUUID, err := parsePair(tenantID, userID)
	if err != nil {
		return ErrNotFound
	}
	if _, err := r.queries.GetManagedUser(ctx, dbsqlc.GetManagedUserParams{UserID: userUUID, TenantID: tenantUUID}); err != nil {
		return ErrNotFound
	}
	return r.queries.UpdateAuthUserPassword(ctx, dbsqlc.UpdateAuthUserPasswordParams{PasswordHash: passwordHash, UserID: userUUID})
}

func (r *PostgreSQLRepository) Delete(ctx context.Context, tenantID, userID string) error {
	tenantUUID, userUUID, err := parsePair(tenantID, userID)
	if err != nil {
		return ErrNotFound
	}
	rows, err := r.queries.DeleteManagedUser(ctx, dbsqlc.DeleteManagedUserParams{UserID: userUUID, TenantID: tenantUUID})
	if err != nil {
		return fmt.Errorf("删除用户失败: %w", err)
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func userFromListRow(row dbsqlc.ListUsersRow) User {
	return User{ID: uuidString(row.ID), TenantID: uuidString(row.TenantID), Username: row.Username, Email: textString(row.Email), Phone: textString(row.Phone), FullName: textString(row.FullName), Avatar: textString(row.Avatar), Role: row.Role, Status: row.Status, Preferences: unmarshalMap(row.Preferences), LastLoginAt: timePointer(row.LastLoginAt), LastLoginIP: textString(row.LastLoginIp), PasswordChangedAt: timePointer(row.PasswordChangedAt), CreatedAt: row.CreatedAt.Time, UpdatedAt: row.UpdatedAt.Time}
}

func userFromModel(row dbsqlc.User) User {
	return User{ID: uuidString(row.ID), TenantID: uuidString(row.TenantID), Username: row.Username, Email: textString(row.Email), Phone: textString(row.Phone), FullName: textString(row.FullName), Avatar: textString(row.Avatar), Role: row.Role, Status: row.Status, Preferences: unmarshalMap(row.Preferences), LastLoginAt: timePointer(row.LastLoginAt), LastLoginIP: textString(row.LastLoginIp), PasswordChangedAt: timePointer(row.PasswordChangedAt), CreatedAt: row.CreatedAt.Time, UpdatedAt: row.UpdatedAt.Time}
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
func timePointer(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}
	result := value.Time
	return &result
}
func marshalMap(value map[string]any) []byte {
	if value == nil {
		return []byte(`{}`)
	}
	result, err := json.Marshal(value)
	if err != nil {
		return []byte(`{}`)
	}
	return result
}
func unmarshalMap(value []byte) map[string]any {
	result := map[string]any{}
	_ = json.Unmarshal(value, &result)
	return result
}
func mapConstraintError(err error) error {
	var pgError *pgconn.PgError
	if errors.As(err, &pgError) && pgError.Code == "23505" {
		return ErrAlreadyExists
	}
	return fmt.Errorf("保存用户失败: %w", err)
}
