package provisioner

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/indu-forge/runtime-engine/internal/store/postgres"
	"github.com/jackc/pgx/v5"
)

var safePGIdentifier = regexp.MustCompile(`^[a-z][a-z0-9_]{0,62}$`)
var runtimeDatabaseIdentifier = regexp.MustCompile(`^ifrt_[0-9a-f]{16}$`)

// PostgresBootstrapCredentials 仅供 initContainer 读取。maintenanceDsn 允许创建
// 尚不存在的目标库，运行容器不会挂载此文件。
type PostgresBootstrapCredentials struct {
	SchemaVersion  string `json:"schemaVersion"`
	MaintenanceDSN string `json:"maintenanceDsn"`
	Database       string `json:"database"`
	Schema         string `json:"schema"`
	Username       string `json:"username"`
	Password       string `json:"password"`
}

// DBAdmin 将 PostgreSQL 管理操作收敛为可替换边界，禁止任何实现回传 DSN 或密码。
type DBAdmin interface {
	DatabaseExists(context.Context, string) (bool, error)
	CreateDatabase(context.Context, string, string) error
	EnsureRole(context.Context, string, string) error
	InitializeSchema(context.Context, string, string, string) error
	Close()
}

type pgxAdmin struct {
	maintenance    *pgx.Conn
	maintenanceDSN string
}

func decodePostgresBootstrap(raw []byte) (PostgresBootstrapCredentials, error) {
	var credentials PostgresBootstrapCredentials
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if err := d.Decode(&credentials); err != nil || d.Decode(&struct{}{}) != io.EOF {
		return PostgresBootstrapCredentials{}, errors.New("PostgreSQL bootstrap 凭据格式非法")
	}
	if credentials.SchemaVersion != "postgres-bootstrap.v1" || strings.TrimSpace(credentials.MaintenanceDSN) == "" || strings.TrimSpace(credentials.Password) == "" || !validPGIdentifier(credentials.Database) || !validPGIdentifier(credentials.Schema) || !validPGIdentifier(credentials.Username) {
		return PostgresBootstrapCredentials{}, errors.New("PostgreSQL bootstrap 凭据非法")
	}
	return credentials, nil
}

func validPGIdentifier(value string) bool   { return safePGIdentifier.MatchString(value) }
func quotePGIdentifier(value string) string { return pgx.Identifier{value}.Sanitize() }

func openDBAdmin(ctx context.Context, credentials PostgresBootstrapCredentials) (DBAdmin, error) {
	conn, err := pgx.Connect(ctx, credentials.MaintenanceDSN)
	if err != nil {
		return nil, errors.New("PostgreSQL maintenance 连接失败")
	}
	return &pgxAdmin{maintenance: conn, maintenanceDSN: credentials.MaintenanceDSN}, nil
}

func (a *pgxAdmin) DatabaseExists(ctx context.Context, name string) (bool, error) {
	var exists bool
	err := a.maintenance.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname=$1)`, name).Scan(&exists)
	return exists, err
}
func (a *pgxAdmin) CreateDatabase(ctx context.Context, name, owner string) error {
	_, err := a.maintenance.Exec(ctx, "CREATE DATABASE "+quotePGIdentifier(name)+" OWNER "+quotePGIdentifier(owner))
	return err
}
func (a *pgxAdmin) EnsureRole(ctx context.Context, role, password string) error {
	var exists bool
	if err := a.maintenance.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM pg_roles WHERE rolname=$1)`, role).Scan(&exists); err != nil {
		return err
	}
	operation := "CREATE ROLE"
	if exists {
		operation = "ALTER ROLE"
	}
	// CREATE/ALTER ROLE 属 utility statement，PostgreSQL 不接受其中的绑定参数。
	// 先由服务端 format(%I/%L) 参数化生成完整语句，再执行该受信结果。
	var statement string
	if err := a.maintenance.QueryRow(ctx, "SELECT format('"+operation+" %I LOGIN PASSWORD %L', $1::text, $2::text)", role, password).Scan(&statement); err != nil {
		return err
	}
	_, err := a.maintenance.Exec(ctx, statement)
	return err
}
func (a *pgxAdmin) InitializeSchema(ctx context.Context, database, schema, role string) error {
	cfg, err := pgx.ParseConfig(a.maintenanceDSN)
	if err != nil {
		return errors.New("PostgreSQL runtime 连接参数非法")
	}
	cfg.Database = database
	conn, err := pgx.ConnectConfig(ctx, cfg)
	if err != nil {
		return errors.New("PostgreSQL runtime 连接失败")
	}
	defer conn.Close(ctx)
	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, "induforge-runtime-state/"+database+"/"+schema); err != nil {
		return err
	}
	var owner string
	err = tx.QueryRow(ctx, `SELECT r.rolname FROM pg_namespace n JOIN pg_roles r ON r.oid=n.nspowner WHERE n.nspname=$1`, schema).Scan(&owner)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	if owner != "" && owner != role {
		return errors.New("schema owner conflict")
	}
	if owner == "" {
		if _, err = tx.Exec(ctx, "CREATE SCHEMA "+quotePGIdentifier(schema)+" AUTHORIZATION "+quotePGIdentifier(role)); err != nil {
			return err
		}
		// schema.sql 是唯一从零基线。替换仅作用于嵌入的受信 SQL 中固定 schema
		// 名称，不接受外部 SQL，且事务失败会完整回滚，绝不执行 ALTER/迁移。
		sql := strings.ReplaceAll(postgres.SchemaSQL, "runtime_engine.", schema+".")
		sql = strings.Replace(sql, "CREATE SCHEMA runtime_engine;", "", 1)
		if _, err = tx.Exec(ctx, "SET LOCAL search_path TO "+quotePGIdentifier(schema)); err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, sql); err != nil {
			return err
		}
	} else {
		var version string
		if err = tx.QueryRow(ctx, "SELECT version FROM "+quotePGIdentifier(schema)+".schema_meta").Scan(&version); err != nil || version != "runtime_engine.v1" {
			return errors.New("runtime schema conflict")
		}
	}
	for _, statement := range []string{
		"GRANT USAGE ON SCHEMA " + quotePGIdentifier(schema) + " TO " + quotePGIdentifier(role),
		"GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA " + quotePGIdentifier(schema) + " TO " + quotePGIdentifier(role),
		"GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA " + quotePGIdentifier(schema) + " TO " + quotePGIdentifier(role),
	} {
		if _, err = tx.Exec(ctx, statement); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
func (a *pgxAdmin) Close() {
	if a != nil && a.maintenance != nil {
		_ = a.maintenance.Close(context.Background())
	}
}

// ProvisionState 使用 maintenance DB 完成库、role 和项目 schema 的唯一基线。
func ProvisionState(ctx context.Context, input Input, raw []byte) error {
	credentials, err := decodePostgresBootstrap(raw)
	if err != nil {
		return err
	}
	timeout, cancel := context.WithTimeout(ctx, provisionTimeout)
	defer cancel()
	admin, err := openDBAdmin(timeout, credentials)
	if err != nil {
		return err
	}
	defer admin.Close()
	return ProvisionStateWithAdmin(timeout, input, credentials, admin)
}

func ProvisionStateWithAdmin(ctx context.Context, input Input, credentials PostgresBootstrapCredentials, admin DBAdmin) error {
	if admin == nil || !runtimeDatabaseIdentifier.MatchString(credentials.Database) || credentials.Database != expectedRuntimeDatabase(input.Binding.ProjectID, input.Binding.SiteID) || credentials.Schema != "runtime_engine" || !validPGIdentifier(credentials.Username) || credentials.Password == "" || input.Binding.StateStore.Schema != credentials.Schema {
		return errors.New("PostgreSQL provision 输入非法")
	}
	if err := admin.EnsureRole(ctx, credentials.Username, credentials.Password); err != nil {
		return stateError("配置 role", credentials.Username)
	}
	exists, err := admin.DatabaseExists(ctx, credentials.Database)
	if err != nil {
		return stateError("检查 database", credentials.Database)
	}
	if !exists && admin.CreateDatabase(ctx, credentials.Database, credentials.Username) != nil {
		return stateError("创建 database", credentials.Database)
	}
	if err := admin.InitializeSchema(ctx, credentials.Database, credentials.Schema, credentials.Username); err != nil {
		return stateError("初始化 schema", credentials.Schema)
	}
	return nil
}
func stateError(stage, name string) error { return fmt.Errorf("PostgreSQL %s失败: %s", stage, name) }

func expectedRuntimeDatabase(projectID, environmentID string) string {
	sum := sha256.Sum256([]byte(projectID + "\x00" + environmentID))
	return fmt.Sprintf("ifrt_%x", sum[:8])
}
