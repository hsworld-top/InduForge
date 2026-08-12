package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	dbsqlc "github.com/indu-forge/dev_core/internal/platform/db/sqlc"
	"github.com/jackc/pgx/v5"
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

func (r *PostgreSQLRepository) CountActiveTenants(ctx context.Context) (int64, error) {
	count, err := r.queries.CountFilteredTenants(ctx, dbsqlc.CountFilteredTenantsParams{Status: "active"})
	if err != nil {
		return 0, fmt.Errorf("统计活跃租户失败: %w", err)
	}
	return count, nil
}

func (r *PostgreSQLRepository) FindLoginUser(ctx context.Context, username, tenantCode string) (User, error) {
	if tenantCode != "" {
		row, err := r.queries.FindAuthUserByTenantCode(ctx, dbsqlc.FindAuthUserByTenantCodeParams{Username: username, TenantCode: tenantCode})
		if err != nil {
			return User{}, mapNotFound(err, ErrInvalidCredentials)
		}
		return userFromTenantRow(row), nil
	}

	rows, err := r.queries.FindAuthUsersByUsername(ctx, username)
	if err != nil {
		return User{}, fmt.Errorf("查询登录用户失败: %w", err)
	}
	if len(rows) == 0 {
		return User{}, ErrInvalidCredentials
	}
	if len(rows) > 1 {
		return User{}, ErrTenantRequired
	}
	return userFromUsernameRow(rows[0]), nil
}

func (r *PostgreSQLRepository) GetUser(ctx context.Context, userID string) (User, error) {
	id, err := parseUUID(userID)
	if err != nil {
		return User{}, ErrNotFound
	}
	row, err := r.queries.GetAuthUserByID(ctx, id)
	if err != nil {
		return User{}, mapNotFound(err, ErrNotFound)
	}
	return userFromIDRow(row), nil
}

func (r *PostgreSQLRepository) UpdateLogin(ctx context.Context, userID, loginIP string) error {
	id, err := parseUUID(userID)
	if err != nil {
		return err
	}
	return r.queries.UpdateAuthUserLogin(ctx, dbsqlc.UpdateAuthUserLoginParams{
		LoginIp: pgtype.Text{String: loginIP, Valid: loginIP != ""},
		UserID:  id,
	})
}

func (r *PostgreSQLRepository) UpdatePassword(ctx context.Context, userID, passwordHash string) error {
	id, err := parseUUID(userID)
	if err != nil {
		return err
	}
	return r.queries.UpdateAuthUserPassword(ctx, dbsqlc.UpdateAuthUserPasswordParams{PasswordHash: passwordHash, UserID: id})
}

func (r *PostgreSQLRepository) CreateRefreshToken(ctx context.Context, token RefreshToken) error {
	tenantID, err := parseUUID(token.TenantID)
	if err != nil {
		return err
	}
	userID, err := parseUUID(token.UserID)
	if err != nil {
		return err
	}
	_, err = r.queries.CreateRefreshToken(ctx, dbsqlc.CreateRefreshTokenParams{
		TenantID:  tenantID,
		UserID:    userID,
		TokenHash: token.Hash,
		ExpiresAt: pgtype.Timestamptz{Time: token.ExpiresAt, Valid: true},
	})
	return err
}

func (r *PostgreSQLRepository) GetRefreshToken(ctx context.Context, tokenHash string) (RefreshToken, error) {
	row, err := r.queries.GetRefreshTokenByHash(ctx, tokenHash)
	if err != nil {
		return RefreshToken{}, mapNotFound(err, ErrInvalidRefresh)
	}
	return RefreshToken{
		ID:        uuidString(row.ID),
		TenantID:  uuidString(row.TenantID),
		UserID:    uuidString(row.UserID),
		Hash:      row.TokenHash,
		ExpiresAt: row.ExpiresAt.Time,
		Revoked:   row.RevokedAt.Valid,
	}, nil
}

func (r *PostgreSQLRepository) RotateRefreshToken(ctx context.Context, oldTokenHash string, replacement RefreshToken) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("开始刷新令牌轮换事务失败: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	queries := r.queries.WithTx(tx)
	oldToken, err := queries.ConsumeRefreshToken(ctx, oldTokenHash)
	if err != nil {
		return ErrInvalidRefresh
	}
	tenantID, err := parseUUID(replacement.TenantID)
	if err != nil {
		return err
	}
	userID, err := parseUUID(replacement.UserID)
	if err != nil {
		return err
	}
	replacementID, err := queries.CreateRefreshToken(ctx, dbsqlc.CreateRefreshTokenParams{
		TenantID:  tenantID,
		UserID:    userID,
		TokenHash: replacement.Hash,
		ExpiresAt: pgtype.Timestamptz{Time: replacement.ExpiresAt, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("保存替换刷新令牌失败: %w", err)
	}
	if oldToken.TenantID != tenantID || oldToken.UserID != userID {
		return ErrInvalidRefresh
	}
	if err := queries.SetRefreshTokenReplacement(ctx, dbsqlc.SetRefreshTokenReplacementParams{ReplacedByTokenID: replacementID, TokenID: oldToken.ID}); err != nil {
		return fmt.Errorf("撤销旧刷新令牌失败: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("提交刷新令牌轮换事务失败: %w", err)
	}
	return nil
}

func (r *PostgreSQLRepository) RevokeRefreshToken(ctx context.Context, tokenHash string) error {
	return r.queries.RevokeRefreshToken(ctx, tokenHash)
}

func (r *PostgreSQLRepository) RevokeUserRefreshTokens(ctx context.Context, userID string) error {
	id, err := parseUUID(userID)
	if err != nil {
		return err
	}
	return r.queries.RevokeUserRefreshTokens(ctx, id)
}

func (r *PostgreSQLRepository) GetTenantBranding(ctx context.Context, tenantCode string) (TenantBranding, error) {
	row, err := r.queries.GetTenantBrandingByCode(ctx, tenantCode)
	if err != nil {
		return TenantBranding{}, mapNotFound(err, ErrNotFound)
	}
	return TenantBranding{ID: uuidString(row.ID), Code: row.Code, Name: row.Name, LogoObjectKey: textString(row.LogoObjectKey), LoginBackgroundObjectKey: textString(row.LoginBackgroundObjectKey)}, nil
}

func mapNotFound(err, target error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return target
	}
	return err
}

func parseUUID(value string) (pgtype.UUID, error) {
	parsed, err := uuid.Parse(value)
	if err != nil {
		return pgtype.UUID{}, fmt.Errorf("UUID 无效: %w", err)
	}
	return pgtype.UUID{Bytes: parsed, Valid: true}, nil
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

func userFromTenantRow(row dbsqlc.FindAuthUserByTenantCodeRow) User {
	return User{ID: uuidString(row.ID), TenantID: uuidString(row.TenantID), Username: row.Username, PasswordHash: row.PasswordHash, Email: textString(row.Email), FullName: textString(row.FullName), Role: row.Role, Status: row.Status, TenantCode: row.TenantCode, TenantName: row.TenantName, TenantStatus: row.TenantStatus, LogoObjectKey: textString(row.LogoObjectKey), LoginBackgroundObjectKey: textString(row.LoginBackgroundObjectKey)}
}

func userFromUsernameRow(row dbsqlc.FindAuthUsersByUsernameRow) User {
	return User{ID: uuidString(row.ID), TenantID: uuidString(row.TenantID), Username: row.Username, PasswordHash: row.PasswordHash, Email: textString(row.Email), FullName: textString(row.FullName), Role: row.Role, Status: row.Status, TenantCode: row.TenantCode, TenantName: row.TenantName, TenantStatus: row.TenantStatus, LogoObjectKey: textString(row.LogoObjectKey), LoginBackgroundObjectKey: textString(row.LoginBackgroundObjectKey)}
}

func userFromIDRow(row dbsqlc.GetAuthUserByIDRow) User {
	return User{ID: uuidString(row.ID), TenantID: uuidString(row.TenantID), Username: row.Username, PasswordHash: row.PasswordHash, Email: textString(row.Email), FullName: textString(row.FullName), Role: row.Role, Status: row.Status, TenantCode: row.TenantCode, TenantName: row.TenantName, TenantStatus: row.TenantStatus, LogoObjectKey: textString(row.LogoObjectKey), LoginBackgroundObjectKey: textString(row.LoginBackgroundObjectKey)}
}
