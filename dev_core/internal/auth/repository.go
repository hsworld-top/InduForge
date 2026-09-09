package auth

import (
	"context"
	"encoding/json"
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
	var count int64
	err := r.pool.QueryRow(ctx, `SELECT count(*) FROM tenants WHERE status='active' AND initialized AND (expires_at IS NULL OR expires_at > now())`).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("统计活跃租户失败: %w", err)
	}
	return count, nil
}

func (r *PostgreSQLRepository) FindLoginUser(ctx context.Context, username, tenantCode string, platform bool) (User, error) {
	if !platform {
		if tenantCode == "" {
			return User{}, ErrTenantRequired
		}
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

func (r *PostgreSQLRepository) UpdatePassword(ctx context.Context, userID, passwordHash string, credentialVersion int64) error {
	id, err := parseUUID(userID)
	if err != nil {
		return err
	}
	rows, err := r.queries.UpdateAuthUserPassword(ctx, dbsqlc.UpdateAuthUserPasswordParams{PasswordHash: passwordHash, UserID: id, CredentialVersion: credentialVersion})
	if err != nil {
		return err
	}
	if rows != 1 {
		return ErrUnauthorized
	}
	return nil
}

func (r *PostgreSQLRepository) CreateRefreshToken(ctx context.Context, token RefreshToken) error {
	tenantID, err := optionalUUID(token.TenantID)
	if err != nil {
		return err
	}
	userID, err := parseUUID(token.UserID)
	if err != nil {
		return err
	}
	_, err = r.queries.CreateRefreshToken(ctx, dbsqlc.CreateRefreshTokenParams{
		RememberMe: token.RememberMe, CredentialVersion: token.CredentialVersion,
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
		RememberMe: row.RememberMe, CredentialVersion: row.CredentialVersion,
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
	tenantID, err := optionalUUID(replacement.TenantID)
	if err != nil {
		return err
	}
	userID, err := parseUUID(replacement.UserID)
	if err != nil {
		return err
	}
	replacementID, err := queries.CreateRefreshToken(ctx, dbsqlc.CreateRefreshTokenParams{
		RememberMe: replacement.RememberMe, CredentialVersion: replacement.CredentialVersion,
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
	brand := TenantBranding{ID: uuidString(row.ID), Code: row.Code, Name: row.Name, LogoObjectKey: textString(row.LogoObjectKey), LoginBackgroundObjectKey: textString(row.LoginBackgroundObjectKey)}
	var raw []byte
	if err := r.pool.QueryRow(ctx, `SELECT COALESCE(settings->'captchaBackgrounds','[]'::jsonb) FROM tenants WHERE id=$1`, row.ID).Scan(&raw); err != nil {
		return TenantBranding{}, err
	}
	_ = json.Unmarshal(raw, &brand.CaptchaBackgrounds)
	return brand, nil
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
	return User{MustChangePassword: row.MustChangePassword, CredentialVersion: row.CredentialVersion, TenantInitialized: row.TenantInitialized, ID: uuidString(row.ID), TenantID: uuidString(row.TenantID), Username: row.Username, PasswordHash: row.PasswordHash, Email: textString(row.Email), FullName: textString(row.FullName), Avatar: textString(row.Avatar), Role: row.Role, Status: row.Status, TenantCode: row.TenantCode, TenantName: row.TenantName, TenantStatus: row.TenantStatus, LogoObjectKey: textString(row.LogoObjectKey), LoginBackgroundObjectKey: textString(row.LoginBackgroundObjectKey)}
}

func userFromUsernameRow(row dbsqlc.FindAuthUsersByUsernameRow) User {
	return User{MustChangePassword: row.MustChangePassword, CredentialVersion: row.CredentialVersion, TenantInitialized: row.TenantInitialized, ID: uuidString(row.ID), TenantID: uuidString(row.TenantID), Username: row.Username, PasswordHash: row.PasswordHash, Email: textString(row.Email), FullName: textString(row.FullName), Avatar: textString(row.Avatar), Role: row.Role, Status: row.Status, TenantCode: row.TenantCode, TenantName: row.TenantName, TenantStatus: row.TenantStatus, LogoObjectKey: textString(row.LogoObjectKey), LoginBackgroundObjectKey: textString(row.LoginBackgroundObjectKey)}
}

func userFromIDRow(row dbsqlc.GetAuthUserByIDRow) User {
	return User{MustChangePassword: row.MustChangePassword, CredentialVersion: row.CredentialVersion, TenantInitialized: row.TenantInitialized, ID: uuidString(row.ID), TenantID: uuidString(row.TenantID), Username: row.Username, PasswordHash: row.PasswordHash, Email: textString(row.Email), FullName: textString(row.FullName), Avatar: textString(row.Avatar), Role: row.Role, Status: row.Status, TenantCode: row.TenantCode, TenantName: row.TenantName, TenantStatus: row.TenantStatus, LogoObjectKey: textString(row.LogoObjectKey), LoginBackgroundObjectKey: textString(row.LoginBackgroundObjectKey)}
}

func optionalUUID(value string) (pgtype.UUID, error) {
	if value == "" {
		return pgtype.UUID{}, nil
	}
	return parseUUID(value)
}

func (r *PostgreSQLRepository) ListLoginTenants(ctx context.Context, keyword, code string, page, limit int) ([]TenantBranding, int64, error) {
	var total int64
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM tenants WHERE status='active' AND initialized AND (expires_at IS NULL OR expires_at > now()) AND ($2='' OR code=$2) AND ($1='' OR name ILIKE '%'||$1||'%' OR code ILIKE '%'||$1||'%')`, keyword, code).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.pool.Query(ctx, `SELECT id::text,code,name,COALESCE(logo_object_key,''),COALESCE(login_background_object_key,'') FROM tenants WHERE status='active' AND initialized AND (expires_at IS NULL OR expires_at > now()) AND ($2='' OR code=$2) AND ($1='' OR name ILIKE '%'||$1||'%' OR code ILIKE '%'||$1||'%') ORDER BY is_default DESC,name,id LIMIT $3 OFFSET $4`, keyword, code, limit, (page-1)*limit)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := []TenantBranding{}
	for rows.Next() {
		var item TenantBranding
		if err := rows.Scan(&item.ID, &item.Code, &item.Name, &item.LogoObjectKey, &item.LoginBackgroundObjectKey); err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func (r *PostgreSQLRepository) UpdateProfile(ctx context.Context, userID, fullName, email string) error {
	id, err := parseUUID(userID)
	if err != nil {
		return ErrUnauthorized
	}
	rows, err := r.queries.UpdateAuthUserProfile(ctx, dbsqlc.UpdateAuthUserProfileParams{
		UserID: id, FullName: pgtype.Text{String: fullName, Valid: fullName != ""}, Email: pgtype.Text{String: email, Valid: email != ""},
	})
	if err != nil {
		return err
	}
	if rows != 1 {
		return ErrUnauthorized
	}
	return nil
}

// 返回被替换的对象键，行锁确保并发上传只清理实际被替换的头像。
func (r *PostgreSQLRepository) UpdateAvatar(ctx context.Context, userID, objectKey string) (string, error) {
	id, err := parseUUID(userID)
	if err != nil {
		return "", err
	}
	old, err := r.queries.UpdateAuthUserAvatar(ctx, dbsqlc.UpdateAuthUserAvatarParams{UserID: id, Avatar: pgtype.Text{String: objectKey, Valid: objectKey != ""}})
	if err != nil {
		return "", mapNotFound(err, ErrNotFound)
	}
	return textString(old), nil
}
