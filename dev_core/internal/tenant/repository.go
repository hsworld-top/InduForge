package tenant

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"runtime"
	"time"

	"github.com/google/uuid"
	platformdb "github.com/indu-forge/dev_core/internal/platform/db"
	dbsqlc "github.com/indu-forge/dev_core/internal/platform/db/sqlc"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgreSQLRepository struct {
	pool          *pgxpool.Pool
	queries       *dbsqlc.Queries
	demoWorkspace interface {
		Initialize(string) (string, error)
		Remove(string) error
	}
	runtimeConfig platformdb.BuiltInRuntimeConfig
}

func NewPostgreSQLRepository(pool *pgxpool.Pool) *PostgreSQLRepository {
	return &PostgreSQLRepository{pool: pool, queries: dbsqlc.New(pool), runtimeConfig: platformdb.BuiltInRuntimeConfig{NodeName: "induframe-center", Architecture: runtime.GOARCH, K3sVersion: "v1.36.4+k3s1", K3sAPIPort: 6443}}
}

func (r *PostgreSQLRepository) SetBuiltInRuntimeConfig(config platformdb.BuiltInRuntimeConfig) {
	r.runtimeConfig = config
}

func (r *PostgreSQLRepository) SetDemoWorkspace(workspace interface {
	Initialize(string) (string, error)
	Remove(string) error
}) {
	r.demoWorkspace = workspace
}
func (r *PostgreSQLRepository) createDemoShell(ctx context.Context, tx pgx.Tx, tenantID, actorID string) (func(), error) {
	if r.demoWorkspace == nil {
		return func() {}, fmt.Errorf("示例工程工作空间未配置")
	}
	projectID := uuid.NewString()
	path, err := r.demoWorkspace.Initialize(projectID)
	if err != nil {
		return func() {}, err
	}
	cleanup := func() { _ = r.demoWorkspace.Remove(path) }
	if err := platformdb.CreateDemoShell(ctx, tx, tenantID, actorID, projectID, path); err != nil {
		cleanup()
		return func() {}, err
	}
	return cleanup, nil
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
		item := tenantFromListRow(row)

		result = append(result, item)
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
	item := tenantFromGetRow(row)

	return item, nil
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
	admin, err := queries.CreateManagedUser(ctx, dbsqlc.CreateManagedUserParams{
		TenantID: created.ID, Username: adminUsername, PasswordHash: adminPasswordHash,
		Role: "SYSTEM_ADMIN", Status: "active", Preferences: []byte(`{}`),
	})
	if err != nil {
		return Tenant{}, fmt.Errorf("创建租户默认管理员失败: %w", err)
	}
	if _, _, err := platformdb.EnsureBuiltInRuntime(ctx, tx, uuidString(created.ID), uuidString(admin.ID), r.runtimeConfig); err != nil {
		return Tenant{}, err
	}
	cleanupDemo, err := r.createDemoShell(ctx, tx, uuidString(created.ID), uuidString(admin.ID))
	if err != nil {
		return Tenant{}, err
	}
	committed := false
	defer func() {
		if !committed {
			cleanupDemo()
		}
	}()
	if _, err := tx.Exec(ctx, `UPDATE tenants SET initialized=true, admin_user_id=$2 WHERE id=$1`, created.ID, admin.ID); err != nil {
		return Tenant{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Tenant{}, fmt.Errorf("提交创建租户事务失败: %w", err)
	}
	committed = true
	result := tenantFromModel(created)
	result.UserCount = 1
	result.Initialized = true
	result.AdminUsername = adminUsername
	result.AdminUserID = uuidString(admin.ID)
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
	result := tenantFromFields(row.ID, row.Name, row.Code, row.Description, row.Status, row.ContactEmail, row.ContactPhone, row.MaxUsers, row.MaxProjects, row.MaxStorage, row.UsedStorage, row.LogoObjectKey, row.LoginBackgroundObjectKey, row.CompanyName, row.CompanyAddress, row.CompanyPhone, row.CompanyWebsite, row.Settings, row.ExpiresAt, row.CreatedAt, row.UpdatedAt, row.UserCount, row.ProjectCount)
	result.Initialized, result.IsDefault, result.AdminUserID = row.Initialized, row.IsDefault, uuidString(row.AdminUserID)
	result.AdminUsername = row.AdminUsername
	return result
}

func tenantFromGetRow(row dbsqlc.GetTenantByIdentifierRow) Tenant {
	result := tenantFromFields(row.ID, row.Name, row.Code, row.Description, row.Status, row.ContactEmail, row.ContactPhone, row.MaxUsers, row.MaxProjects, row.MaxStorage, row.UsedStorage, row.LogoObjectKey, row.LoginBackgroundObjectKey, row.CompanyName, row.CompanyAddress, row.CompanyPhone, row.CompanyWebsite, row.Settings, row.ExpiresAt, row.CreatedAt, row.UpdatedAt, row.UserCount, row.ProjectCount)
	result.Initialized, result.IsDefault, result.AdminUserID = row.Initialized, row.IsDefault, uuidString(row.AdminUserID)
	result.AdminUsername = row.AdminUsername
	return result
}

func tenantFromModel(row dbsqlc.Tenant) Tenant {
	result := tenantFromFields(row.ID, row.Name, row.Code, row.Description, row.Status, row.ContactEmail, row.ContactPhone, row.MaxUsers, row.MaxProjects, row.MaxStorage, row.UsedStorage, row.LogoObjectKey, row.LoginBackgroundObjectKey, row.CompanyName, row.CompanyAddress, row.CompanyPhone, row.CompanyWebsite, row.Settings, row.ExpiresAt, row.CreatedAt, row.UpdatedAt, 0, 0)
	result.Initialized, result.IsDefault, result.AdminUserID = row.Initialized, row.IsDefault, uuidString(row.AdminUserID)
	return result
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
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	var pgError *pgconn.PgError
	if errors.As(err, &pgError) && pgError.Code == "23505" {
		return ErrAlreadyExists
	}
	return fmt.Errorf("保存租户失败: %w", err)
}

// 锁定租户行串行化初始化；重试已成功的请求不会替换管理员或密码。
func (r *PostgreSQLRepository) Initialize(ctx context.Context, identifier, username, hash string) (Tenant, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Tenant{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	cleanupDemo := func() {}
	committed := false
	defer func() {
		if !committed {
			cleanupDemo()
		}
	}()
	var id string
	var initialized bool
	if err := tx.QueryRow(ctx, `SELECT id::text, initialized FROM tenants WHERE id::text=$1 OR lower(code)=lower($1) FOR UPDATE`, identifier).Scan(&id, &initialized); err != nil {
		return Tenant{}, mapConstraintError(err)
	}
	if !initialized {
		var userID string
		if err := tx.QueryRow(ctx, `INSERT INTO users(tenant_id,username,password_hash,role,status) VALUES($1,$2,$3,'SYSTEM_ADMIN','active') RETURNING id::text`, id, username, hash).Scan(&userID); err != nil {
			return Tenant{}, mapConstraintError(err)
		}
		if _, _, err := platformdb.EnsureBuiltInRuntime(ctx, tx, id, userID, r.runtimeConfig); err != nil {
			return Tenant{}, err
		}
		cleanupDemo, err = r.createDemoShell(ctx, tx, id, userID)
		if err != nil {
			return Tenant{}, err
		}
		if _, err := tx.Exec(ctx, `UPDATE tenants SET initialized=true,admin_user_id=$2,updated_at=now() WHERE id=$1`, id, userID); err != nil {
			return Tenant{}, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return Tenant{}, err
	}
	committed = true
	return r.Get(ctx, id)
}

// 密码版本和刷新会话在同一事务更新，旧 access token 也立即因版本不匹配失效。
func (r *PostgreSQLRepository) ResetAdminPassword(ctx context.Context, identifier, hash string) (Tenant, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Tenant{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var id, userID string
	if err := tx.QueryRow(ctx, `SELECT id::text,admin_user_id::text FROM tenants WHERE (id::text=$1 OR lower(code)=lower($1)) AND initialized FOR UPDATE`, identifier).Scan(&id, &userID); err != nil {
		return Tenant{}, ErrNotFound
	}
	result, err := tx.Exec(ctx, `UPDATE users SET password_hash=$3,must_change_password=true,credential_version=credential_version+1,password_changed_at=now(),updated_at=now() WHERE id=$1 AND tenant_id=$2 AND role='SYSTEM_ADMIN'`, userID, id, hash)
	if err != nil {
		return Tenant{}, err
	}
	if result.RowsAffected() != 1 {
		return Tenant{}, ErrNotFound
	}
	if _, err := tx.Exec(ctx, `UPDATE refresh_tokens SET revoked_at=COALESCE(revoked_at,now()) WHERE user_id=$1`, userID); err != nil {
		return Tenant{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Tenant{}, err
	}
	return r.Get(ctx, id)
}
