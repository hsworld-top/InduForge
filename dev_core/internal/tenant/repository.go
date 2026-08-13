package tenant

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

type PostgreSQLRepository struct {
	pool    *pgxpool.Pool
	queries *dbsqlc.Queries
}

func NewPostgreSQLRepository(pool *pgxpool.Pool) *PostgreSQLRepository {
	return &PostgreSQLRepository{pool: pool, queries: dbsqlc.New(pool)}
}

func (r *PostgreSQLRepository) List(ctx context.Context, filter ListFilter) ([]Tenant, int64, error) {
	rows, err := r.queries.ListTenants(ctx, dbsqlc.ListTenantsParams{
		Keyword: filter.Keyword, Status: filter.Status,
		PageOffset: int32((filter.Page - 1) * filter.Limit), PageLimit: int32(filter.Limit),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("查询租户列表失败: %w", err)
	}
	total, err := r.queries.CountFilteredTenants(ctx, dbsqlc.CountFilteredTenantsParams{Keyword: filter.Keyword, Status: filter.Status})
	if err != nil {
		return nil, 0, fmt.Errorf("统计租户数量失败: %w", err)
	}
	result := make([]Tenant, 0, len(rows))
	for _, row := range rows {
		result = append(result, tenantFromListRow(row))
	}
	return result, total, nil
}

func (r *PostgreSQLRepository) Get(ctx context.Context, identifier string) (Tenant, error) {
	row, err := r.queries.GetTenantByIdentifier(ctx, identifier)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Tenant{}, ErrNotFound
		}
		return Tenant{}, fmt.Errorf("查询租户失败: %w", err)
	}
	return tenantFromGetRow(row), nil
}

func (r *PostgreSQLRepository) Create(ctx context.Context, item Tenant, adminUsername, adminPasswordHash string) (Tenant, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Tenant{}, fmt.Errorf("开始创建租户事务失败: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := r.queries.WithTx(tx)
	created, err := queries.CreateTenant(ctx, createTenantParams(item))
	if err != nil {
		return Tenant{}, mapConstraintError(err)
	}
	_, err = queries.CreateManagedUser(ctx, dbsqlc.CreateManagedUserParams{
		TenantID: created.ID, Username: adminUsername, PasswordHash: adminPasswordHash,
		Role: "SYSTEM_ADMIN", Status: "active", Preferences: []byte(`{}`),
	})
	if err != nil {
		return Tenant{}, fmt.Errorf("创建租户默认管理员失败: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Tenant{}, fmt.Errorf("提交创建租户事务失败: %w", err)
	}
	result := tenantFromModel(created)
	result.UserCount = 1
	return result, nil
}

func (r *PostgreSQLRepository) Update(ctx context.Context, item Tenant) (Tenant, error) {
	id, err := parseUUID(item.ID)
	if err != nil {
		return Tenant{}, ErrNotFound
	}
	params := updateTenantParams(item)
	params.TenantID = id
	updated, err := r.queries.UpdateTenant(ctx, params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Tenant{}, ErrNotFound
		}
		return Tenant{}, mapConstraintError(err)
	}
	return tenantFromModel(updated), nil
}

func (r *PostgreSQLRepository) Delete(ctx context.Context, tenantID string) error {
	id, err := parseUUID(tenantID)
	if err != nil {
		return ErrNotFound
	}
	rows, err := r.queries.DeleteTenant(ctx, id)
	if err != nil {
		return fmt.Errorf("删除租户失败: %w", err)
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgreSQLRepository) SetStatus(ctx context.Context, tenantID, status string) (Tenant, error) {
	id, err := parseUUID(tenantID)
	if err != nil {
		return Tenant{}, ErrNotFound
	}
	updated, err := r.queries.SetTenantStatus(ctx, dbsqlc.SetTenantStatusParams{Status: status, TenantID: id})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Tenant{}, ErrNotFound
		}
		return Tenant{}, fmt.Errorf("更新租户状态失败: %w", err)
	}
	return tenantFromModel(updated), nil
}

func (r *PostgreSQLRepository) ListNotes(ctx context.Context, tenantID string) ([]Note, error) {
	id, err := parseUUID(tenantID)
	if err != nil {
		return nil, ErrNotFound
	}
	rows, err := r.queries.ListDashboardNotes(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("查询便签失败: %w", err)
	}
	result := make([]Note, 0, len(rows))
	for _, row := range rows {
		result = append(result, Note{
			ID: uuidString(row.ID), TenantID: uuidString(row.TenantID), Content: row.Content,
			CreatedBy: uuidString(row.CreatedBy), UpdatedBy: uuidString(row.UpdatedBy),
			CreatedByUsername: row.CreatedByUsername, UpdatedByUsername: row.UpdatedByUsername,
			CreatedAt: row.CreatedAt.Time, UpdatedAt: row.UpdatedAt.Time,
		})
	}
	return result, nil
}

func (r *PostgreSQLRepository) CreateNote(ctx context.Context, tenantID, userID, content string) (Note, error) {
	tenantUUID, userUUID, err := parsePair(tenantID, userID)
	if err != nil {
		return Note{}, err
	}
	created, err := r.queries.CreateDashboardNote(ctx, dbsqlc.CreateDashboardNoteParams{TenantID: tenantUUID, Content: content, UserID: userUUID})
	if err != nil {
		return Note{}, fmt.Errorf("创建便签失败: %w", err)
	}
	return noteFromModel(created), nil
}

func (r *PostgreSQLRepository) UpdateNote(ctx context.Context, tenantID, userID, noteID, content string) (Note, error) {
	tenantUUID, userUUID, err := parsePair(tenantID, userID)
	if err != nil {
		return Note{}, err
	}
	noteUUID, err := parseUUID(noteID)
	if err != nil {
		return Note{}, ErrNoteNotFound
	}
	updated, err := r.queries.UpdateDashboardNote(ctx, dbsqlc.UpdateDashboardNoteParams{Content: content, UserID: userUUID, NoteID: noteUUID, TenantID: tenantUUID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Note{}, ErrNoteNotFound
		}
		return Note{}, fmt.Errorf("更新便签失败: %w", err)
	}
	return noteFromModel(updated), nil
}

func (r *PostgreSQLRepository) DeleteNote(ctx context.Context, tenantID, noteID string) error {
	tenantUUID, noteUUID, err := parsePair(tenantID, noteID)
	if err != nil {
		return ErrNoteNotFound
	}
	rows, err := r.queries.DeleteDashboardNote(ctx, dbsqlc.DeleteDashboardNoteParams{NoteID: noteUUID, TenantID: tenantUUID})
	if err != nil {
		return fmt.Errorf("删除便签失败: %w", err)
	}
	if rows == 0 {
		return ErrNoteNotFound
	}
	return nil
}

func createTenantParams(item Tenant) dbsqlc.CreateTenantParams {
	return dbsqlc.CreateTenantParams{
		Name: item.Name, Code: item.Code, Description: nullableText(item.Description), Status: item.Status,
		ContactEmail: nullableText(item.ContactEmail), ContactPhone: nullableText(item.ContactPhone),
		MaxUsers: item.MaxUsers, MaxProjects: item.MaxProjects, MaxStorage: item.MaxStorage,
		LogoObjectKey: nullableText(item.LogoObjectKey), LoginBackgroundObjectKey: nullableText(item.LoginBackgroundObjectKey),
		CompanyName: nullableText(item.CompanyName), CompanyAddress: nullableText(item.CompanyAddress),
		CompanyPhone: nullableText(item.CompanyPhone), CompanyWebsite: nullableText(item.CompanyWebsite),
		Settings: marshalSettings(item.Settings), ExpiresAt: nullableTime(item.ExpiresAt),
	}
}

func updateTenantParams(item Tenant) dbsqlc.UpdateTenantParams {
	created := createTenantParams(item)
	return dbsqlc.UpdateTenantParams{
		Name: created.Name, Code: created.Code, Description: created.Description, Status: created.Status,
		ContactEmail: created.ContactEmail, ContactPhone: created.ContactPhone, MaxUsers: created.MaxUsers,
		MaxProjects: created.MaxProjects, MaxStorage: created.MaxStorage, LogoObjectKey: created.LogoObjectKey,
		LoginBackgroundObjectKey: created.LoginBackgroundObjectKey, CompanyName: created.CompanyName,
		CompanyAddress: created.CompanyAddress, CompanyPhone: created.CompanyPhone,
		CompanyWebsite: created.CompanyWebsite, Settings: created.Settings, ExpiresAt: created.ExpiresAt,
	}
}

func tenantFromListRow(row dbsqlc.ListTenantsRow) Tenant {
	return tenantFromFields(row.ID, row.Name, row.Code, row.Description, row.Status, row.ContactEmail, row.ContactPhone, row.MaxUsers, row.MaxProjects, row.MaxStorage, row.UsedStorage, row.LogoObjectKey, row.LoginBackgroundObjectKey, row.CompanyName, row.CompanyAddress, row.CompanyPhone, row.CompanyWebsite, row.Settings, row.ExpiresAt, row.CreatedAt, row.UpdatedAt, row.UserCount, row.ProjectCount)
}

func tenantFromGetRow(row dbsqlc.GetTenantByIdentifierRow) Tenant {
	return tenantFromFields(row.ID, row.Name, row.Code, row.Description, row.Status, row.ContactEmail, row.ContactPhone, row.MaxUsers, row.MaxProjects, row.MaxStorage, row.UsedStorage, row.LogoObjectKey, row.LoginBackgroundObjectKey, row.CompanyName, row.CompanyAddress, row.CompanyPhone, row.CompanyWebsite, row.Settings, row.ExpiresAt, row.CreatedAt, row.UpdatedAt, row.UserCount, row.ProjectCount)
}

func tenantFromModel(row dbsqlc.Tenant) Tenant {
	return tenantFromFields(row.ID, row.Name, row.Code, row.Description, row.Status, row.ContactEmail, row.ContactPhone, row.MaxUsers, row.MaxProjects, row.MaxStorage, row.UsedStorage, row.LogoObjectKey, row.LoginBackgroundObjectKey, row.CompanyName, row.CompanyAddress, row.CompanyPhone, row.CompanyWebsite, row.Settings, row.ExpiresAt, row.CreatedAt, row.UpdatedAt, 0, 0)
}

func tenantFromFields(id pgtype.UUID, name, code string, description pgtype.Text, status string, contactEmail, contactPhone pgtype.Text, maxUsers, maxProjects int32, maxStorage, usedStorage int64, logoObjectKey, backgroundObjectKey, companyName, companyAddress, companyPhone, companyWebsite pgtype.Text, settings []byte, expiresAt, createdAt, updatedAt pgtype.Timestamptz, userCount, projectCount int64) Tenant {
	var expires *time.Time
	if expiresAt.Valid {
		value := expiresAt.Time
		expires = &value
	}
	return Tenant{ID: uuidString(id), Name: name, Code: code, Description: textString(description), Status: status,
		ContactEmail: textString(contactEmail), ContactPhone: textString(contactPhone), MaxUsers: maxUsers,
		MaxProjects: maxProjects, MaxStorage: maxStorage, UsedStorage: usedStorage, LogoObjectKey: textString(logoObjectKey),
		LoginBackgroundObjectKey: textString(backgroundObjectKey), CompanyName: textString(companyName), CompanyAddress: textString(companyAddress),
		CompanyPhone: textString(companyPhone), CompanyWebsite: textString(companyWebsite), Settings: unmarshalSettings(settings),
		ExpiresAt: expires, CreatedAt: createdAt.Time, UpdatedAt: updatedAt.Time, UserCount: userCount, ProjectCount: projectCount}
}

func noteFromModel(row dbsqlc.TenantDashboardNote) Note {
	return Note{ID: uuidString(row.ID), TenantID: uuidString(row.TenantID), Content: row.Content, CreatedBy: uuidString(row.CreatedBy), UpdatedBy: uuidString(row.UpdatedBy), CreatedAt: row.CreatedAt.Time, UpdatedAt: row.UpdatedAt.Time}
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

func nullableTime(value *time.Time) pgtype.Timestamptz {
	if value == nil {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: *value, Valid: true}
}

func marshalSettings(value map[string]any) []byte {
	if value == nil {
		return []byte(`{}`)
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return []byte(`{}`)
	}
	return encoded
}

func unmarshalSettings(value []byte) map[string]any {
	result := map[string]any{}
	_ = json.Unmarshal(value, &result)
	return result
}

func mapConstraintError(err error) error {
	var pgError *pgconn.PgError
	if errors.As(err, &pgError) && pgError.Code == "23505" {
		return ErrAlreadyExists
	}
	return fmt.Errorf("保存租户失败: %w", err)
}
