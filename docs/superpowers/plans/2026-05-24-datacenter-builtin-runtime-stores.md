# 数据中心 IF 内置运行库多实例实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 将 `IF关系库`、`IF时序库`、`IF实时库`、`IF消息库` 实现为可创建多个实例的内置运行库连接，并提供连接级开发态工作台与 artifact 契约。

**架构：** 复用 `data_connections` 承载内置运行库实例；用户只填写名称，服务端生成不可变 `runtimeKey` 并据此派生 schema、key 前缀和 topic 前缀。关系库和时序库复用连接级 SQL 工作台 API；实时库和消息库新增连接级 key/topic/变量 API；artifact 使用集合结构输出运行态契约，不输出开发态资源和内部连接参数。

**技术栈：** Go、pgx、PostgreSQL、Redis、MQTT、Vue 3、TypeScript、pnpm workspace、Go test、datacenter typecheck/lint。

---

## 文件结构

- 修改：`data_service/internal/db/migrations/0001_core_tables.sql`
  - 扩展初始表约束，允许 `builtin.*` 类型和 `builtin` 分类；不创建同类型唯一索引。
- 修改：`data_service/internal/db/migrations/0023_builtin_runtime_stores.sql`
  - 升级已有环境的连接类型与分类约束；删除旧计划中的 `(project_id, type)` 唯一索引。
- 修改：`data_service/internal/db/migrations/0023_builtin_runtime_stores_down.sql`
  - 回退新增类型和分类约束。
- 修改：`data_service/tests/integration/migration_test.go`
  - 验证同项目可插入多个同类型 `builtin.*` 连接。
- 修改：`data_service/internal/service/builtin_runtime_store.go`
  - 生成 `runtimeKey`、清洗内部参数、派生连接级开发态资源名。
- 修改：`data_service/internal/service/builtin_runtime_store_test.go`
  - 覆盖 `runtimeKey`、多实例创建输入、内部参数清洗。
- 修改：`data_service/internal/service/connection_service.go`
  - 内置连接创建/更新规则：名称可改、类型和 `runtimeKey` 不可改、禁止内部参数写入。
- 修改：`data_service/internal/repository/connection_repository.go`
  - 保持按 `(project_id, id)` 读取；不再提供或使用按类型查重作为内置单例限制。
- 修改：`data_service/internal/service/builtin_runtime_service.go`
  - 所有内置开发态操作改为按 `connectionId` 定位具体实例。
- 修改：`data_service/internal/http/handler/builtin_runtime_handler.go`
  - 挂载连接级 SQL、实时 key、消息 topic/变量/publish API 的入参解析。
- 修改：`data_service/internal/http/router/router.go`
  - 使用 `/connections/{connectionId}/...` 连接级路由。
- 修改：`data_service/internal/repository/project_snapshot_repository.go`
  - `builtinStores` 从单例对象改为集合结构。
- 修改：`data_service/internal/service/project_snapshot_service.go`
  - 快照导入允许多实例 `builtin.*` 连接。
- 修改：`data_service/tests/integration/project_snapshot_artifact_test.go`
  - 验证 artifact 输出多个内置实例且不泄漏开发态资源。
- 修改：`datacenter/src/api/data.api.ts`
  - 新增/调整连接级内置运行库 API。
- 修改：`datacenter/src/api/schemas/access-source.schema.ts`
  - 纳入四类 `builtin.*` 类型。
- 修改：`datacenter/src/config/connectionTypes.ts`
  - 增加四类内置运行库的显示元数据。
- 修改：`datacenter/src/components/dialogs/ConnectionDialog.vue`
  - 新建内置运行库时只展示名称、说明和少量策略字段，不展示内部标识。
- 修改：`datacenter/src/components/access-source/AccessSourceWorkspace.vue`
  - 列表与卡片表达内置运行库实例。
- 修改：`datacenter/src/components/access-source/AccessSourceWorkbench.vue`
  - 将四类内置运行库分发到新的工作台。
- 修改/创建：`datacenter/src/components/access-source/workbench/BuiltinRelationWorkbench.vue`
  - 改为复用 SQL 工作台。
- 修改/创建：`datacenter/src/components/access-source/workbench/BuiltinTimeseriesWorkbench.vue`
  - 改为 SQL 工作台骨架加时序辅助入口。
- 修改/创建：`datacenter/src/components/access-source/workbench/BuiltinRealtimeWorkbench.vue`
  - 实时 key 定义与当前值测试工作台。
- 修改/创建：`datacenter/src/components/access-source/workbench/BuiltinMessageWorkbench.vue`
  - topic、变量与发布预览工作台。
- 修改/创建：`datacenter/src/components/access-source/workbench/builtin-store.ts`
  - 前端内置类型、文案和默认策略。

## 任务 1：迁移和连接模型允许同类型多实例

**文件：**
- 修改：`data_service/internal/db/migrations/0001_core_tables.sql`
- 修改：`data_service/internal/db/migrations/0023_builtin_runtime_stores.sql`
- 修改：`data_service/internal/db/migrations/0023_builtin_runtime_stores_down.sql`
- 修改：`data_service/tests/integration/migration_test.go`

- [ ] **步骤 1：编写失败的迁移测试**

在 `data_service/tests/integration/migration_test.go` 增加测试，确认同项目允许两个 `builtin.relation`：

```go
func TestBuiltinRuntimeStoreMigrationAllowsMultipleInstances(t *testing.T) {
	ctx := context.Background()
	fixture := newMigrationFixture(t)
	defer fixture.cleanup()

	migrator := setupMigrator(t, fixture.pool)
	if err := migrator.Up(ctx); err != nil {
		t.Fatalf("run migrations failed: %v", err)
	}

	projectID := "project-builtin-multi"
	userID := "user-builtin-multi"
	for index := 1; index <= 2; index++ {
		_, err := fixture.pool.Exec(ctx, `
			INSERT INTO data_connections (project_id, name, type, category, status, metadata, created_by, updated_by)
			VALUES ($1, $2, 'builtin.relation', 'builtin', 'connected', '{}'::jsonb, $3, $3)
		`, projectID, fmt.Sprintf("关系库 %d", index), userID)
		if err != nil {
			t.Fatalf("insert builtin relation %d failed: %v", index, err)
		}
	}
}
```

如果文件尚未导入 `fmt`，在 import 中加入：

```go
import (
	"fmt"
)
```

- [ ] **步骤 2：运行测试验证失败或确认旧索引冲突**

运行：

```powershell
cd data_service
go test ./tests/integration -run TestBuiltinRuntimeStoreMigrationAllowsMultipleInstances -count=1
```

预期：若旧唯一索引仍存在则 FAIL，错误包含 `data_connections_project_builtin_type_unique`；若当前库未包含旧索引但约束已允许 builtin 类型则 PASS，此时仍继续执行步骤 3 清理迁移定义。

- [ ] **步骤 3：调整迁移**

在 `data_service/internal/db/migrations/0023_builtin_runtime_stores.sql` 中确保没有创建唯一索引，并显式清理旧索引：

```sql
-- 扩展 data_connections 支持 IF 内置运行库，多实例模型不限制同项目同类型数量。
ALTER TABLE data_connections DROP CONSTRAINT IF EXISTS data_connections_type_check;
ALTER TABLE data_connections
    ADD CONSTRAINT data_connections_type_check
    CHECK (type IN (
        'relational', 'mqtt', 'websocket', 'opcua', 'modbus', 'http', 's7',
        'kafka', 'redis', 'tdengine',
        'builtin.relation', 'builtin.timeseries', 'builtin.realtime', 'builtin.message'
    ));

ALTER TABLE data_connections DROP CONSTRAINT IF EXISTS data_connections_category_check;
ALTER TABLE data_connections
    ADD CONSTRAINT data_connections_category_check
    CHECK (category IN ('database', 'message', 'protocol', 'api', 'builtin'));

DROP INDEX IF EXISTS data_connections_project_builtin_type_unique;
```

在 `data_service/internal/db/migrations/0001_core_tables.sql` 确保 `type` 和 `category` CHECK 包含四类 `builtin.*` 和 `builtin` 分类，且不创建 `data_connections_project_builtin_type_unique`。

在 `data_service/internal/db/migrations/0023_builtin_runtime_stores_down.sql` 保留：

```sql
DROP INDEX IF EXISTS data_connections_project_builtin_type_unique;
```

并恢复到不含 `builtin.*` 的连接类型约束。

- [ ] **步骤 4：运行迁移测试验证通过**

运行：

```powershell
cd data_service
go test ./tests/integration -run TestBuiltinRuntimeStoreMigrationAllowsMultipleInstances -count=1
```

预期：PASS。

- [ ] **步骤 5：Commit**

```powershell
git add data_service/internal/db/migrations/0001_core_tables.sql data_service/internal/db/migrations/0023_builtin_runtime_stores.sql data_service/internal/db/migrations/0023_builtin_runtime_stores_down.sql data_service/tests/integration/migration_test.go
git commit -m "feat(data_service): 允许内置运行库多实例连接"
```

## 任务 2：服务端生成不可变 runtimeKey

**文件：**
- 修改：`data_service/internal/service/builtin_runtime_store.go`
- 修改：`data_service/internal/service/builtin_runtime_store_test.go`
- 修改：`data_service/internal/service/connection_service.go`

- [ ] **步骤 1：编写 runtimeKey 单测**

在 `data_service/internal/service/builtin_runtime_store_test.go` 增加：

```go
func TestNormalizeBuiltinStoreCreateInputGeneratesRuntimeKey(t *testing.T) {
	input := CreateConnectionInput{
		Name: "订单库",
		Type: "builtin.relation",
		Config: map[string]any{
			"runtimeKey": "user_supplied",
			"devSchema": "p_leak",
			"host": "127.0.0.1",
		},
	}

	result, err := normalizeBuiltinStoreCreateInput("project-1", input)
	if err != nil {
		t.Fatalf("normalize builtin input failed: %v", err)
	}

	runtimeKey, _ := result.Config["runtimeKey"].(string)
	if !strings.HasPrefix(runtimeKey, "rel_") {
		t.Fatalf("expected generated rel_ runtimeKey, got %q", runtimeKey)
	}
	if runtimeKey == "user_supplied" {
		t.Fatal("runtimeKey must be generated by service")
	}
	if result.Config["devSchema"] == "p_leak" {
		t.Fatal("devSchema must not keep user supplied value")
	}
	if _, ok := result.Config["host"]; ok {
		t.Fatal("builtin config must drop internal host")
	}
}
```

如果文件尚未导入 `strings`，补充：

```go
import "strings"
```

- [ ] **步骤 2：运行测试验证失败**

运行：

```powershell
cd data_service
go test ./internal/service -run TestNormalizeBuiltinStoreCreateInputGeneratesRuntimeKey -count=1
```

预期：FAIL，通常是 `runtimeKey` 未生成或保留了用户输入。

- [ ] **步骤 3：实现 runtimeKey 生成和内部字段清洗**

在 `data_service/internal/service/builtin_runtime_store.go` 增加：

```go
func newBuiltinRuntimeKey(connectionType string) string {
	prefix := map[string]string{
		"builtin.relation":   "rel",
		"builtin.timeseries": "ts",
		"builtin.realtime":   "rt",
		"builtin.message":    "msg",
	}[strings.TrimSpace(strings.ToLower(connectionType))]
	if prefix == "" {
		prefix = "store"
	}
	return fmt.Sprintf("%s_%s", prefix, randomHex(3))
}

func deriveBuiltinConnectionSchema(projectID string, runtimeKey string) string {
	normalizedProject := normalizeBuiltinIdentifier(projectID, "unknown")
	normalizedKey := normalizeBuiltinIdentifier(runtimeKey, "store")
	return fmt.Sprintf("p_%s_%s", normalizedProject, normalizedKey)
}
```

扩展内部参数黑名单，确保用户输入不会覆盖系统字段：

```go
var unsafeBuiltinConfigKeys = map[string]struct{}{
	"host":       {},
	"port":       {},
	"username":   {},
	"password":   {},
	"broker":     {},
	"brokerurl":  {},
	"database":   {},
	"db":         {},
	"token":      {},
	"secret":     {},
	"runtimekey": {},
	"devschema":  {},
	"namespace":  {},
	"topicprefix": {},
}
```

在 `normalizeBuiltinStoreCreateInput` 中先生成：

```go
runtimeKey := newBuiltinRuntimeKey(connectionType)
config["runtimeKey"] = runtimeKey
```

四类配置改为：

```go
case "builtin.relation":
	config["store"] = "relation"
	config["devSchema"] = deriveBuiltinConnectionSchema(projectID, runtimeKey)
	config["runtimeSchema"] = runtimeKey
	config["ddlVersion"] = builtinStoreDDLVersion
case "builtin.timeseries":
	config["store"] = "timeseries"
	config["devSchema"] = deriveBuiltinConnectionSchema(projectID, runtimeKey)
	config["runtimeSchema"] = runtimeKey
	config["retentionDays"] = intFromAny(config["retentionDays"], 30)
	config["ddlVersion"] = builtinStoreDDLVersion
case "builtin.realtime":
	config["store"] = "realtime"
	config["namespace"] = runtimeKey
	config["defaultTtlSeconds"] = intFromAny(config["defaultTtlSeconds"], 300)
case "builtin.message":
	config["store"] = "message"
	config["topicPrefix"] = runtimeKey
```

- [ ] **步骤 4：禁止更新时改动 runtimeKey 和 type**

在 `data_service/internal/service/connection_service.go` 的更新逻辑中，读取原连接后加入：

```go
if isBuiltinStoreType(current.Type) {
	if input.Type != "" && strings.TrimSpace(strings.ToLower(input.Type)) != current.Type {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "内置运行库类型不允许修改")
	}
	nextConfig := sanitizeBuiltinConfig(input.Config)
	currentRuntimeKey := builtinConfigString(current.Config, "runtimeKey", "")
	if currentRuntimeKey != "" {
		nextConfig["runtimeKey"] = currentRuntimeKey
	}
	for _, key := range []string{"devSchema", "runtimeSchema", "namespace", "topicPrefix", "store"} {
		if value, ok := current.Config[key]; ok {
			nextConfig[key] = value
		}
	}
	input.Config = nextConfig
}
```

如果当前更新函数没有 `current.Config` 结构，使用已有连接记录的 `Config` 或 `Metadata` 字段转换；不要新增按名称推导逻辑。

- [ ] **步骤 5：运行服务测试**

运行：

```powershell
cd data_service
go test ./internal/service -run 'BuiltinStore|NormalizeBuiltin' -count=1
```

预期：PASS。

- [ ] **步骤 6：Commit**

```powershell
git add data_service/internal/service/builtin_runtime_store.go data_service/internal/service/builtin_runtime_store_test.go data_service/internal/service/connection_service.go
git commit -m "feat(data_service): 生成内置运行库系统标识"
```

## 任务 3：关系库和时序库接入连接级 SQL API

**文件：**
- 修改：`data_service/internal/service/builtin_runtime_service.go`
- 修改：`data_service/internal/service/builtin_runtime_service_test.go`
- 修改：`data_service/internal/service/connection_service.go`
- 修改：`data_service/internal/http/router/router.go`

- [ ] **步骤 1：编写连接级 SQL 测试**

在 `data_service/internal/service/builtin_runtime_service_test.go` 增加测试，直接验证 schema 推导不再只看项目和类型：

```go
func TestBuiltinSQLUsesConnectionRuntimeKey(t *testing.T) {
	connection := Connection{
		ID:        "conn-1",
		ProjectID: "project-1",
		Type:      "builtin.relation",
		Config: map[string]any{
			"runtimeKey": "rel_abc123",
			"devSchema":  "p_project_1_rel_abc123",
		},
	}

	schema, store, err := builtinSQLContextFromConnection(connection)
	if err != nil {
		t.Fatalf("resolve builtin sql context failed: %v", err)
	}
	if store != "relation" {
		t.Fatalf("expected relation store, got %q", store)
	}
	if schema != "p_project_1_rel_abc123" {
		t.Fatalf("expected connection schema, got %q", schema)
	}
}
```

- [ ] **步骤 2：运行测试验证失败**

运行：

```powershell
cd data_service
go test ./internal/service -run TestBuiltinSQLUsesConnectionRuntimeKey -count=1
```

预期：FAIL，错误包含 `undefined: builtinSQLContextFromConnection`。

- [ ] **步骤 3：实现连接级 SQL 上下文**

在 `data_service/internal/service/builtin_runtime_service.go` 增加：

```go
func builtinSQLContextFromConnection(connection Connection) (string, string, error) {
	store := ""
	switch connection.Type {
	case "builtin.relation":
		store = "relation"
	case "builtin.timeseries":
		store = "timeseries"
	default:
		return "", "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "连接不是内置 SQL 运行库")
	}
	schema := builtinConfigString(connection.Config, "devSchema", "")
	if schema == "" {
		runtimeKey := builtinConfigString(connection.Config, "runtimeKey", "")
		if runtimeKey == "" {
			return "", "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "内置运行库缺少系统标识")
		}
		schema = deriveBuiltinConnectionSchema(connection.ProjectID, runtimeKey)
	}
	return schema, store, nil
}
```

将内置 SQL 执行服务改造为接收 `Connection` 或 `connectionId` 解析后的连接，不再只接收 `Store` 字符串。保留核心执行函数：

```go
func (s *BuiltinRuntimeService) ExecuteSQLInSchema(ctx context.Context, schemaName string, sqlText string, parameters []any, limit int) (*BuiltinSQLExecuteResult, error) {
	// 复用现有 ExecuteSQL 中的校验、事务、SET LOCAL search_path、Query/Exec 逻辑。
}
```

原项目级 `ExecuteSQL` 可临时保留给兼容测试，但连接级入口必须调用 `ExecuteSQLInSchema`。

- [ ] **步骤 4：在连接服务 SQL API 中分发 builtin 类型**

找到 `data_service/internal/service/connection_service.go` 中现有 `ExecuteSQL`、`GetConnectionTables`、`GetTableStructure` 相关函数。在每个函数开始处读取连接后加入：

```go
if connection.Type == "builtin.relation" || connection.Type == "builtin.timeseries" {
	schemaName, _, err := builtinSQLContextFromConnection(connection)
	if err != nil {
		return nil, err
	}
	return s.builtinRuntime.ExecuteSQLInSchema(ctx, schemaName, sql, parameters, 500)
}
```

表列表使用连接级 schema 查询：

```sql
SELECT table_name
FROM information_schema.tables
WHERE table_schema = $1 AND table_type = 'BASE TABLE'
ORDER BY table_name
```

表结构使用连接级 schema 查询：

```sql
SELECT column_name, data_type, is_nullable, column_default
FROM information_schema.columns
WHERE table_schema = $1 AND table_name = $2
ORDER BY ordinal_position
```

这些查询只由服务端传入 `schemaName`，用户不能提供 schema。

- [ ] **步骤 5：保留并标记旧项目级 API**

在 `data_service/internal/http/router/router.go` 中，旧的：

```go
POST /api/v1/data/projects/{projectId}/builtin/relation/sql/execute
POST /api/v1/data/projects/{projectId}/builtin/timeseries/query
```

不再作为前端调用路径。若为了过渡保留，handler 必须返回明确错误：

```go
return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "请使用连接级内置运行库 API")
```

前端计划中不得再调用旧路径。

- [ ] **步骤 6：运行后端测试**

运行：

```powershell
cd data_service
go test ./internal/service -run 'BuiltinSQL|Connection.*SQL|Table' -count=1
```

预期：PASS。

- [ ] **步骤 7：Commit**

```powershell
git add data_service/internal/service/builtin_runtime_service.go data_service/internal/service/builtin_runtime_service_test.go data_service/internal/service/connection_service.go data_service/internal/http/router/router.go
git commit -m "feat(data_service): 接入内置 SQL 连接级执行"
```

## 任务 4：实时库连接级 key 定义与当前值 API

**文件：**
- 修改：`data_service/internal/service/builtin_runtime_service.go`
- 修改：`data_service/internal/service/builtin_runtime_service_test.go`
- 修改：`data_service/internal/http/handler/builtin_runtime_handler.go`
- 修改：`data_service/internal/http/router/router.go`
- 修改：`data_service/internal/db/migrations/0023_builtin_runtime_stores.sql`

- [ ] **步骤 1：编写 key 前缀测试**

在 `data_service/internal/service/builtin_runtime_service_test.go` 增加：

```go
func TestDeriveBuiltinRealtimeKeyUsesRuntimeKey(t *testing.T) {
	key, err := deriveBuiltinRealtimeKey("ifdev", "project-1", "rt_a1b2c3", "device/line1/status")
	if err != nil {
		t.Fatalf("derive realtime key failed: %v", err)
	}
	expected := "ifdev:project-1:rt_a1b2c3:device/line1/status"
	if key != expected {
		t.Fatalf("expected %q, got %q", expected, key)
	}
}
```

- [ ] **步骤 2：运行测试验证失败**

运行：

```powershell
cd data_service
go test ./internal/service -run TestDeriveBuiltinRealtimeKeyUsesRuntimeKey -count=1
```

预期：FAIL，旧函数还接受 group/category。

- [ ] **步骤 3：改造实时 key 推导**

将 `deriveBuiltinRealtimeKey` 签名改为：

```go
func deriveBuiltinRealtimeKey(prefix, projectID, runtimeKey, key string) (string, error) {
	runtimeKey = strings.Trim(strings.TrimSpace(runtimeKey), ":")
	if runtimeKey == "" {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "实时库缺少系统标识")
	}
	key = strings.Trim(strings.TrimSpace(key), "/")
	if key == "" || strings.Contains(key, "..") {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "实时库 key 不合法")
	}
	return fmt.Sprintf("%s:%s:%s:%s", strings.Trim(prefix, ":"), projectID, runtimeKey, key), nil
}
```

`SetRealtimeKey`、`GetRealtimeKey`、`DeleteRealtimeKey` 改为接收 `Connection` 或 `runtimeKey`，不再接收 `group`。

- [ ] **步骤 4：增加 key 定义存储表**

在 `data_service/internal/db/migrations/0023_builtin_runtime_stores.sql` 增加：

```sql
CREATE TABLE IF NOT EXISTS data_builtin_realtime_keys (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id text NOT NULL,
    connection_id uuid NOT NULL,
    key_path text NOT NULL,
    value_type text NOT NULL DEFAULT 'json',
    default_ttl_seconds integer NOT NULL DEFAULT 300,
    description text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (project_id, connection_id, key_path),
    FOREIGN KEY (connection_id) REFERENCES data_connections (id) ON DELETE CASCADE
);
```

在 down 迁移中增加：

```sql
DROP TABLE IF EXISTS data_builtin_realtime_keys;
```

- [ ] **步骤 5：挂载连接级 realtime API**

在 `data_service/internal/http/router/router.go` 挂载：

```go
mux.Handle("GET /api/v1/data/projects/{projectId}/connections/{connectionId}/realtime/keys", read(opts.builtinRuntimeHandler.ListRealtimeKeys))
mux.Handle("POST /api/v1/data/projects/{projectId}/connections/{connectionId}/realtime/keys", write(opts.builtinRuntimeHandler.SetRealtimeKey))
mux.Handle("GET /api/v1/data/projects/{projectId}/connections/{connectionId}/realtime/keys/{key}", read(opts.builtinRuntimeHandler.GetRealtimeKey))
mux.Handle("DELETE /api/v1/data/projects/{projectId}/connections/{connectionId}/realtime/keys/{key}", write(opts.builtinRuntimeHandler.DeleteRealtimeKey))
```

handler 必须通过 `projectId + connectionId` 读取连接，并校验 `connection.Type == "builtin.realtime"` 后再调用服务。

- [ ] **步骤 6：运行实时库测试**

运行：

```powershell
cd data_service
go test ./internal/service -run 'BuiltinRealtime|DeriveBuiltinRealtimeKey' -count=1
```

预期：PASS。

- [ ] **步骤 7：Commit**

```powershell
git add data_service/internal/service/builtin_runtime_service.go data_service/internal/service/builtin_runtime_service_test.go data_service/internal/http/handler/builtin_runtime_handler.go data_service/internal/http/router/router.go data_service/internal/db/migrations/0023_builtin_runtime_stores.sql data_service/internal/db/migrations/0023_builtin_runtime_stores_down.sql
git commit -m "feat(data_service): 增加内置实时库连接级 key API"
```

## 任务 5：消息库 topic、变量与发布 API

**文件：**
- 修改：`data_service/internal/service/builtin_runtime_service.go`
- 修改：`data_service/internal/service/builtin_runtime_service_test.go`
- 修改：`data_service/internal/http/handler/builtin_runtime_handler.go`
- 修改：`data_service/internal/http/router/router.go`
- 修改：`data_service/internal/db/migrations/0023_builtin_runtime_stores.sql`

- [ ] **步骤 1：编写 topic 前缀测试**

在 `data_service/internal/service/builtin_runtime_service_test.go` 增加：

```go
func TestDeriveBuiltinMessageTopicUsesRuntimeKey(t *testing.T) {
	topic, err := deriveBuiltinMessageTopic("ifdev", "project-1", "msg_a1b2c3", "device/1/telemetry")
	if err != nil {
		t.Fatalf("derive message topic failed: %v", err)
	}
	expected := "ifdev/project-1/msg_a1b2c3/device/1/telemetry"
	if topic != expected {
		t.Fatalf("expected %q, got %q", expected, topic)
	}
}
```

- [ ] **步骤 2：运行测试验证失败**

运行：

```powershell
cd data_service
go test ./internal/service -run TestDeriveBuiltinMessageTopicUsesRuntimeKey -count=1
```

预期：FAIL，旧函数缺少 `runtimeKey` 参数。

- [ ] **步骤 3：增加 topic 与变量存储表**

在 `data_service/internal/db/migrations/0023_builtin_runtime_stores.sql` 增加：

```sql
CREATE TABLE IF NOT EXISTS data_builtin_message_topics (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id text NOT NULL,
    connection_id uuid NOT NULL,
    topic text NOT NULL,
    name text NOT NULL,
    description text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (project_id, connection_id, topic),
    FOREIGN KEY (connection_id) REFERENCES data_connections (id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS data_builtin_message_variables (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id text NOT NULL,
    topic_id uuid NOT NULL,
    name text NOT NULL,
    payload_path text NOT NULL,
    value_type text NOT NULL DEFAULT 'string',
    unit text NOT NULL DEFAULT '',
    description text NOT NULL DEFAULT '',
    create_datapoint boolean NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (project_id, topic_id, name),
    FOREIGN KEY (topic_id) REFERENCES data_builtin_message_topics (id) ON DELETE CASCADE
);
```

在 down 迁移中增加：

```sql
DROP TABLE IF EXISTS data_builtin_message_variables;
DROP TABLE IF EXISTS data_builtin_message_topics;
```

- [ ] **步骤 4：改造消息 topic 推导和发布**

将 `deriveBuiltinMessageTopic` 签名改为：

```go
func deriveBuiltinMessageTopic(prefix, projectID, runtimeKey, topic string) (string, error) {
	runtimeKey = strings.Trim(strings.TrimSpace(runtimeKey), "/")
	if runtimeKey == "" {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "消息库缺少系统标识")
	}
	topic = strings.Trim(strings.TrimSpace(topic), "/")
	if topic == "" || strings.Contains(topic, "..") || strings.Contains(topic, "#") || strings.Contains(topic, "+") {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "消息 topic 不合法")
	}
	return fmt.Sprintf("%s/%s/%s/%s", strings.Trim(prefix, "/"), projectID, runtimeKey, topic), nil
}
```

`PublishMessage` 通过连接的 `config.runtimeKey` 派生完整 topic。

- [ ] **步骤 5：挂载连接级 message API**

在 `router.go` 挂载：

```go
mux.Handle("GET /api/v1/data/projects/{projectId}/connections/{connectionId}/message/topics", read(opts.builtinRuntimeHandler.ListMessageTopics))
mux.Handle("POST /api/v1/data/projects/{projectId}/connections/{connectionId}/message/topics", write(opts.builtinRuntimeHandler.CreateMessageTopic))
mux.Handle("POST /api/v1/data/projects/{projectId}/connections/{connectionId}/message/publish", write(opts.builtinRuntimeHandler.PublishMessage))
mux.Handle("GET /api/v1/data/projects/{projectId}/connections/{connectionId}/message/topics/{topicId}/variables", read(opts.builtinRuntimeHandler.ListMessageVariables))
mux.Handle("POST /api/v1/data/projects/{projectId}/connections/{connectionId}/message/topics/{topicId}/variables", write(opts.builtinRuntimeHandler.CreateMessageVariable))
```

handler 必须校验 `connection.Type == "builtin.message"`。

- [ ] **步骤 6：运行消息库测试**

运行：

```powershell
cd data_service
go test ./internal/service -run 'BuiltinMessage|DeriveBuiltinMessageTopic' -count=1
```

预期：PASS。

- [ ] **步骤 7：Commit**

```powershell
git add data_service/internal/service/builtin_runtime_service.go data_service/internal/service/builtin_runtime_service_test.go data_service/internal/http/handler/builtin_runtime_handler.go data_service/internal/http/router/router.go data_service/internal/db/migrations/0023_builtin_runtime_stores.sql data_service/internal/db/migrations/0023_builtin_runtime_stores_down.sql
git commit -m "feat(data_service): 增加内置消息库 topic 变量 API"
```

## 任务 6：artifact 输出集合结构 builtinStores

**文件：**
- 修改：`data_service/internal/repository/project_snapshot_repository.go`
- 修改：`data_service/internal/service/project_snapshot_service.go`
- 修改：`data_service/tests/integration/project_snapshot_artifact_test.go`

- [ ] **步骤 1：编写 artifact 多实例测试**

在 `data_service/tests/integration/project_snapshot_artifact_test.go` 中增加断言：同一项目插入两个关系库后 artifact 中有两个 `relations` 项。

```go
if len(artifact.BuiltinStores.Relations) != 2 {
	t.Fatalf("expected 2 relation stores, got %d", len(artifact.BuiltinStores.Relations))
}
for _, store := range artifact.BuiltinStores.Relations {
	if store.RuntimeKey == "" {
		t.Fatal("relation store runtimeKey must not be empty")
	}
	if strings.HasPrefix(store.Schema, "p_") {
		t.Fatalf("artifact leaked dev schema %q", store.Schema)
	}
}
rawArtifact, err := json.Marshal(artifact)
if err != nil {
	t.Fatalf("marshal artifact failed: %v", err)
}
for _, forbidden := range []string{"if_dev_data", "p_project_", "18379", "18883", "password", "broker"} {
	if strings.Contains(string(rawArtifact), forbidden) {
		t.Fatalf("artifact leaked internal value %q: %s", forbidden, string(rawArtifact))
	}
}
```

- [ ] **步骤 2：运行测试验证失败**

运行：

```powershell
cd data_service
go test ./tests/integration -run TestProjectArtifactV1Contract -count=1
```

预期：FAIL，旧结构没有 `Relations` 集合或只输出单例。

- [ ] **步骤 3：调整 artifact 类型**

在 `project_snapshot_repository.go` 定义：

```go
type ArtifactBuiltinStoresPayload struct {
	Relations      []ArtifactBuiltinRelationStore   `json:"relations"`
	Timeseries     []ArtifactBuiltinTimeseriesStore `json:"timeseries"`
	RealtimeSpaces []ArtifactBuiltinRealtimeStore   `json:"realtimeSpaces"`
	MessageSpaces  []ArtifactBuiltinMessageStore    `json:"messageSpaces"`
}

type ArtifactBuiltinRelationStore struct {
	ID         string   `json:"id"`
	RuntimeKey string   `json:"runtimeKey"`
	Name       string   `json:"name"`
	Schema     string   `json:"schema"`
	DDLVersion string   `json:"ddlVersion"`
	Queries    []string `json:"queries"`
}

type ArtifactBuiltinTimeseriesStore struct {
	ID            string   `json:"id"`
	RuntimeKey    string   `json:"runtimeKey"`
	Name          string   `json:"name"`
	Schema        string   `json:"schema"`
	RetentionDays int      `json:"retentionDays"`
	DDLVersion    string   `json:"ddlVersion"`
	Tables        []string `json:"tables"`
}

type ArtifactBuiltinRealtimeStore struct {
	ID                string   `json:"id"`
	RuntimeKey        string   `json:"runtimeKey"`
	Name              string   `json:"name"`
	Namespace         string   `json:"namespace"`
	DefaultTtlSeconds int      `json:"defaultTtlSeconds"`
	Keys              []string `json:"keys"`
}

type ArtifactBuiltinMessageStore struct {
	ID          string   `json:"id"`
	RuntimeKey  string   `json:"runtimeKey"`
	Name        string   `json:"name"`
	TopicPrefix string   `json:"topicPrefix"`
	Topics      []string `json:"topics"`
	Variables   []string `json:"variables"`
	Bindings    []string `json:"bindings"`
}
```

- [ ] **步骤 4：从连接记录构建集合**

在 `buildBuiltinStores` 中按连接追加：

```go
runtimeKey := stringFromMap(connection.Config, "runtimeKey", "")
if runtimeKey == "" {
	return
}
switch connection.Type {
case "builtin.relation":
	payload.Relations = append(payload.Relations, ArtifactBuiltinRelationStore{
		ID:         connection.ID,
		RuntimeKey: runtimeKey,
		Name:       connection.Name,
		Schema:     runtimeKey,
		DDLVersion: stringFromMap(connection.Config, "ddlVersion", builtinStoreDDLVersion),
		Queries:    []string{},
	})
case "builtin.timeseries":
	payload.Timeseries = append(payload.Timeseries, ArtifactBuiltinTimeseriesStore{
		ID:            connection.ID,
		RuntimeKey:    runtimeKey,
		Name:          connection.Name,
		Schema:        runtimeKey,
		RetentionDays: intFromMap(connection.Config, "retentionDays", 30),
		DDLVersion:    stringFromMap(connection.Config, "ddlVersion", builtinStoreDDLVersion),
		Tables:        []string{},
	})
case "builtin.realtime":
	payload.RealtimeSpaces = append(payload.RealtimeSpaces, ArtifactBuiltinRealtimeStore{
		ID:                connection.ID,
		RuntimeKey:        runtimeKey,
		Name:              connection.Name,
		Namespace:         runtimeKey,
		DefaultTtlSeconds: intFromMap(connection.Config, "defaultTtlSeconds", 300),
		Keys:              []string{},
	})
case "builtin.message":
	payload.MessageSpaces = append(payload.MessageSpaces, ArtifactBuiltinMessageStore{
		ID:          connection.ID,
		RuntimeKey:  runtimeKey,
		Name:        connection.Name,
		TopicPrefix: runtimeKey,
		Topics:      []string{},
		Variables:   []string{},
		Bindings:    []string{},
	})
}
```

- [ ] **步骤 5：运行 artifact 测试**

运行：

```powershell
cd data_service
go test ./internal/repository ./internal/service ./tests/integration -run 'Artifact|Snapshot|Builtin' -count=1
```

预期：PASS。

- [ ] **步骤 6：Commit**

```powershell
git add data_service/internal/repository/project_snapshot_repository.go data_service/internal/service/project_snapshot_service.go data_service/tests/integration/project_snapshot_artifact_test.go
git commit -m "feat(data_service): 输出内置运行库多实例契约"
```

## 任务 7：前端类型、API 和新建弹窗

**文件：**
- 修改：`datacenter/src/api/schemas/access-source.schema.ts`
- 修改：`datacenter/src/config/connectionTypes.ts`
- 修改：`datacenter/src/api/data.api.ts`
- 修改/创建：`datacenter/src/components/access-source/workbench/builtin-store.ts`
- 修改：`datacenter/src/components/dialogs/ConnectionDialog.vue`

- [ ] **步骤 1：定义内置运行库前端元数据**

在 `builtin-store.ts` 写入：

```ts
export type BuiltinStoreType =
  | 'builtin.relation'
  | 'builtin.timeseries'
  | 'builtin.realtime'
  | 'builtin.message'

export const BUILTIN_STORE_TYPES: Array<{
  type: BuiltinStoreType
  name: string
  description: string
  defaultConfig: Record<string, unknown>
}> = [
  {
    type: 'builtin.relation',
    name: 'IF关系库',
    description: '业务表、表单数据和普通 SQL',
    defaultConfig: { allowDdlTest: true },
  },
  {
    type: 'builtin.timeseries',
    name: 'IF时序库',
    description: '数据点历史、趋势和聚合',
    defaultConfig: { retentionDays: 30, timeField: 'ts' },
  },
  {
    type: 'builtin.realtime',
    name: 'IF实时库',
    description: '当前值、短期状态和缓存',
    defaultConfig: { defaultTtlSeconds: 300 },
  },
  {
    type: 'builtin.message',
    name: 'IF消息库',
    description: '设备消息、模拟数据和订阅',
    defaultConfig: { sampleTopic: 'mock-data', samplePayload: { value: 1 } },
  },
]

export const isBuiltinStoreType = (type?: string): type is BuiltinStoreType =>
  BUILTIN_STORE_TYPES.some((item) => item.type === type)

export const getBuiltinStoreMeta = (type?: string) =>
  BUILTIN_STORE_TYPES.find((item) => item.type === type)
```

- [ ] **步骤 2：调整连接级 API**

在 `datacenter/src/api/data.api.ts` 增加：

```ts
export const getBuiltinRealtimeKeys = (projectId, connectionId, params = {}) =>
  request({
    url: `/data/projects/${projectId}/connections/${connectionId}/realtime/keys`,
    method: 'get',
    params,
  })

export const setBuiltinRealtimeKey = (projectId, connectionId, data) =>
  request({
    url: `/data/projects/${projectId}/connections/${connectionId}/realtime/keys`,
    method: 'post',
    data,
  })

export const getBuiltinRealtimeKey = (projectId, connectionId, key) =>
  request({
    url: `/data/projects/${projectId}/connections/${connectionId}/realtime/keys/${encodeURIComponent(key)}`,
    method: 'get',
  })

export const deleteBuiltinRealtimeKey = (projectId, connectionId, key) =>
  request({
    url: `/data/projects/${projectId}/connections/${connectionId}/realtime/keys/${encodeURIComponent(key)}`,
    method: 'delete',
  })

export const getBuiltinMessageTopics = (projectId, connectionId) =>
  request({
    url: `/data/projects/${projectId}/connections/${connectionId}/message/topics`,
    method: 'get',
  })

export const createBuiltinMessageTopic = (projectId, connectionId, data) =>
  request({
    url: `/data/projects/${projectId}/connections/${connectionId}/message/topics`,
    method: 'post',
    data,
  })

export const publishBuiltinMessage = (projectId, connectionId, data) =>
  request({
    url: `/data/projects/${projectId}/connections/${connectionId}/message/publish`,
    method: 'post',
    data,
  })

export const getBuiltinMessageVariables = (projectId, connectionId, topicId) =>
  request({
    url: `/data/projects/${projectId}/connections/${connectionId}/message/topics/${topicId}/variables`,
    method: 'get',
  })

export const createBuiltinMessageVariable = (projectId, connectionId, topicId, data) =>
  request({
    url: `/data/projects/${projectId}/connections/${connectionId}/message/topics/${topicId}/variables`,
    method: 'post',
    data,
  })
```

移除前端对旧项目级 `/builtin/relation`、`/builtin/timeseries`、`/builtin/realtime`、`/builtin/message` API 的调用。

- [ ] **步骤 3：扩展 schema 和 connectionTypes**

在 `datacenter/src/api/schemas/access-source.schema.ts` 的接入源类型 union 中加入：

```ts
'builtin.relation',
'builtin.timeseries',
'builtin.realtime',
'builtin.message',
```

在 `datacenter/src/config/connectionTypes.ts` 增加四类 `builtin` 分类配置，显示名分别为 `IF关系库`、`IF时序库`、`IF实时库`、`IF消息库`。

- [ ] **步骤 4：调整新建弹窗**

在 `ConnectionDialog.vue` 中，内置类型选择后表单只允许：

```ts
const builtinPayload = {
  name: form.name,
  type: form.type,
  config: {
    description: form.description,
    ...selectedBuiltinMeta.defaultConfig,
    ...visiblePolicyFields,
  },
}
```

模板中不得出现 `runtimeKey`、`devSchema`、`host`、`port`、`username`、`password`、`broker`、`database`、`db` 输入。

- [ ] **步骤 5：运行前端检查**

运行：

```powershell
pnpm --filter datacenter typecheck
pnpm --filter datacenter lint
```

预期：typecheck PASS；lint PASS 或仅保留仓库既有 warning。

- [ ] **步骤 6：Commit**

```powershell
git add datacenter/src/api/schemas/access-source.schema.ts datacenter/src/config/connectionTypes.ts datacenter/src/api/data.api.ts datacenter/src/components/access-source/workbench/builtin-store.ts datacenter/src/components/dialogs/ConnectionDialog.vue
git commit -m "feat(datacenter): 增加内置运行库多实例入口"
```

## 任务 8：关系库和时序库复用 SQL 工作台

**文件：**
- 修改：`datacenter/src/components/access-source/AccessSourceWorkbench.vue`
- 修改/创建：`datacenter/src/components/access-source/workbench/BuiltinRelationWorkbench.vue`
- 修改/创建：`datacenter/src/components/access-source/workbench/BuiltinTimeseriesWorkbench.vue`

- [ ] **步骤 1：关系库直接复用 SQL 工作台**

将 `BuiltinRelationWorkbench.vue` 改为：

```vue
<template>
  <SqlQueryWorkbench :project-id="projectId" :connection="sqlConnection" @back="$emit('back')" />
</template>

<script setup lang="ts">
import { computed } from 'vue'
import SqlQueryWorkbench from '@/components/database/SqlQueryWorkbench.vue'

const props = defineProps<{
  connection: Record<string, any>
  projectId: string
}>()

defineEmits<{
  (event: 'back'): void
}>()

const sqlConnection = computed(() => ({
  ...props.connection,
  type: 'builtin.relation',
  relationalConfig: {
    dbType: 'postgresql',
    database: props.connection.name || 'IF关系库',
    schema: props.connection.config?.runtimeKey || 'system',
  },
}))
</script>
```

- [ ] **步骤 2：时序库复用 SQL 工作台**

将 `BuiltinTimeseriesWorkbench.vue` 改为同样复用 `SqlQueryWorkbench`，但 `dbTypeLabel` 通过连接类型显示不一定支持。若 `SqlQueryWorkbench` 只显示 `PostgreSQL`，本阶段接受；右侧增强入口在任务 10 补。

```vue
<template>
  <SqlQueryWorkbench :project-id="projectId" :connection="sqlConnection" @back="$emit('back')" />
</template>

<script setup lang="ts">
import { computed } from 'vue'
import SqlQueryWorkbench from '@/components/database/SqlQueryWorkbench.vue'

const props = defineProps<{
  connection: Record<string, any>
  projectId: string
}>()

defineEmits<{
  (event: 'back'): void
}>()

const sqlConnection = computed(() => ({
  ...props.connection,
  type: 'builtin.timeseries',
  relationalConfig: {
    dbType: 'postgresql',
    database: props.connection.name || 'IF时序库',
    schema: props.connection.config?.runtimeKey || 'system',
  },
}))
</script>
```

- [ ] **步骤 3：工作台分发**

在 `AccessSourceWorkbench.vue` 中确保：

```ts
if (type === 'builtin.relation') return BuiltinRelationWorkbench
if (type === 'builtin.timeseries') return BuiltinTimeseriesWorkbench
```

- [ ] **步骤 4：运行前端检查**

运行：

```powershell
pnpm --filter datacenter typecheck
pnpm --filter datacenter lint
```

预期：PASS。

- [ ] **步骤 5：Commit**

```powershell
git add datacenter/src/components/access-source/AccessSourceWorkbench.vue datacenter/src/components/access-source/workbench/BuiltinRelationWorkbench.vue datacenter/src/components/access-source/workbench/BuiltinTimeseriesWorkbench.vue
git commit -m "feat(datacenter): 复用 SQL 工作台承载内置 SQL 库"
```

## 任务 9：实时库工作台参考 Redis 管理体验

**文件：**
- 修改/创建：`datacenter/src/components/access-source/workbench/BuiltinRealtimeWorkbench.vue`
- 修改：`datacenter/src/components/access-source/AccessSourceWorkbench.vue`

- [ ] **步骤 1：实现实时库工作台布局**

`BuiltinRealtimeWorkbench.vue` 使用三栏布局：

```vue
<template>
  <section class="builtin-realtime">
    <header class="builtin-realtime__topbar">
      <button type="button" class="builtin-realtime__back" @click="$emit('back')">
        <IconTablerArrowLeft />
      </button>
      <div class="builtin-realtime__identity">
        <span>IF实时库</span>
        <strong>{{ connection.name || '未命名实时库' }}</strong>
      </div>
      <el-input v-model="pattern" size="small" placeholder="key 前缀" @keyup.enter="loadKeys" />
      <el-button size="small" :loading="loadingKeys" @click="loadKeys">刷新</el-button>
    </header>

    <main class="builtin-realtime__main">
      <section class="builtin-realtime__keys">
        <div class="builtin-realtime__pane-head">
          <strong>Key 定义</strong>
          <el-input v-model="keyword" size="small" clearable placeholder="过滤 key" />
        </div>
        <button
          v-for="item in filteredKeys"
          :key="item.key"
          type="button"
          class="builtin-realtime__row"
          :class="{ 'is-selected': selectedKey === item.key }"
          @click="selectKey(item.key)"
        >
          <span>{{ item.key }}</span>
          <em>{{ item.type || 'json' }}</em>
          <em>{{ formatTTL(item.ttl) }}</em>
        </button>
      </section>

      <aside class="builtin-realtime__detail">
        <el-input v-model="form.key" size="small" placeholder="device/line1/status" />
        <el-input v-model="form.valueText" type="textarea" :rows="8" spellcheck="false" />
        <el-input-number v-model="form.ttlSeconds" :min="0" :max="86400" />
        <div class="builtin-realtime__actions">
          <el-button type="primary" :loading="saving" @click="setKey">写入</el-button>
          <el-button :loading="loadingValue" @click="getKey">读取</el-button>
          <el-button :loading="deleting" @click="deleteKey">删除</el-button>
        </div>
        <pre>{{ currentValueText }}</pre>
      </aside>
    </main>
  </section>
</template>
```

- [ ] **步骤 2：接入连接级 API**

脚本中调用：

```ts
await dataAPI.getBuiltinRealtimeKeys(props.projectId, props.connection.id, {
  pattern: pattern.value || '*',
})
await dataAPI.setBuiltinRealtimeKey(props.projectId, props.connection.id, {
  key: form.key,
  value,
  ttlSeconds: form.ttlSeconds,
})
await dataAPI.getBuiltinRealtimeKey(props.projectId, props.connection.id, form.key)
await dataAPI.deleteBuiltinRealtimeKey(props.projectId, props.connection.id, form.key)
```

`valueText` 优先按 JSON 解析，解析失败则作为字符串写入。

- [ ] **步骤 3：工作台分发**

在 `AccessSourceWorkbench.vue` 中确保：

```ts
if (type === 'builtin.realtime') return BuiltinRealtimeWorkbench
```

- [ ] **步骤 4：运行前端检查**

运行：

```powershell
pnpm --filter datacenter typecheck
pnpm --filter datacenter lint
```

预期：PASS。

- [ ] **步骤 5：Commit**

```powershell
git add datacenter/src/components/access-source/workbench/BuiltinRealtimeWorkbench.vue datacenter/src/components/access-source/AccessSourceWorkbench.vue
git commit -m "feat(datacenter): 完善内置实时库 key 工作台"
```

## 任务 10：消息库工作台参考 MQTT topic/变量体验

**文件：**
- 修改/创建：`datacenter/src/components/access-source/workbench/BuiltinMessageWorkbench.vue`
- 修改：`datacenter/src/components/access-source/AccessSourceWorkbench.vue`

- [ ] **步骤 1：实现 topic 和变量布局**

`BuiltinMessageWorkbench.vue` 使用左侧 topic，右侧变量和发布测试：

```vue
<template>
  <section class="builtin-message">
    <aside class="builtin-message__topics">
      <header>
        <button type="button" @click="$emit('back')">
          <IconTablerArrowLeft />
        </button>
        <strong>{{ connection.name || 'IF消息库' }}</strong>
        <el-button size="small" @click="createTopic">新建 Topic</el-button>
      </header>
      <button
        v-for="topic in topics"
        :key="topic.id"
        type="button"
        :class="{ 'is-active': selectedTopic?.id === topic.id }"
        @click="selectedTopic = topic; loadVariables()"
      >
        <span>{{ topic.name || topic.topic }}</span>
        <small>{{ topic.topic }}</small>
      </button>
    </aside>

    <main class="builtin-message__main">
      <section class="builtin-message__variables">
        <div class="builtin-message__pane-head">
          <strong>变量</strong>
          <el-button size="small" :disabled="!selectedTopic" @click="createVariable">新增变量</el-button>
        </div>
        <el-table :data="variables" size="small" border height="100%">
          <el-table-column prop="name" label="变量" min-width="140" />
          <el-table-column prop="payloadPath" label="Payload Path" min-width="180" />
          <el-table-column prop="valueType" label="类型" width="100" />
          <el-table-column prop="unit" label="单位" width="80" />
        </el-table>
      </section>

      <section class="builtin-message__publisher">
        <div class="builtin-message__pane-head">
          <strong>发布测试</strong>
          <el-button type="primary" size="small" :loading="publishing" @click="publish">发布</el-button>
        </div>
        <el-input v-model="publishForm.topic" size="small" placeholder="mock-data" />
        <el-input v-model="publishForm.payloadText" type="textarea" :rows="10" spellcheck="false" />
        <pre>{{ publishResultText }}</pre>
      </section>
    </main>
  </section>
</template>
```

- [ ] **步骤 2：接入连接级 API**

脚本调用：

```ts
await dataAPI.getBuiltinMessageTopics(props.projectId, props.connection.id)
await dataAPI.createBuiltinMessageTopic(props.projectId, props.connection.id, {
  topic: publishForm.topic,
  name: publishForm.topic,
})
await dataAPI.getBuiltinMessageVariables(props.projectId, props.connection.id, selectedTopic.value.id)
await dataAPI.createBuiltinMessageVariable(props.projectId, props.connection.id, selectedTopic.value.id, {
  name: variableForm.name,
  payloadPath: variableForm.payloadPath,
  valueType: variableForm.valueType,
  unit: variableForm.unit,
})
await dataAPI.publishBuiltinMessage(props.projectId, props.connection.id, {
  topic: publishForm.topic,
  qos: 0,
  payload,
})
```

Payload JSON 解析失败时使用 `ElMessage.error('Payload 必须是合法 JSON')` 并中止发布。

- [ ] **步骤 3：工作台分发**

在 `AccessSourceWorkbench.vue` 中确保：

```ts
if (type === 'builtin.message') return BuiltinMessageWorkbench
```

- [ ] **步骤 4：运行前端检查**

运行：

```powershell
pnpm --filter datacenter typecheck
pnpm --filter datacenter lint
```

预期：PASS。

- [ ] **步骤 5：Commit**

```powershell
git add datacenter/src/components/access-source/workbench/BuiltinMessageWorkbench.vue datacenter/src/components/access-source/AccessSourceWorkbench.vue
git commit -m "feat(datacenter): 完善内置消息库 topic 变量工作台"
```

## 任务 11：端到端验证和浏览器验收

**文件：**
- 根据验证结果只修改本计划相关文件。

- [ ] **步骤 1：后端完整验证**

运行：

```powershell
cd data_service
go test ./...
go build ./...
```

预期：PASS。

- [ ] **步骤 2：前端完整验证**

运行：

```powershell
pnpm --filter datacenter typecheck
pnpm --filter datacenter lint
pnpm --filter datacenter build
```

预期：typecheck PASS，lint PASS 或仅仓库既有 warning，build PASS。

- [ ] **步骤 3：浏览器验收**

打开数据中心 test1 工程并执行：

1. 新建两个 `IF关系库`，名称分别为“订单库”和“配方库”，确认都创建成功。
2. 打开“订单库”，执行 `create table if not exists orders(id int primary key);`，再刷新表列表确认出现 `orders`。
3. 打开“配方库”，确认表列表不出现 `orders`。
4. 新建一个 `IF时序库`，执行时序样例表创建和查询。
5. 新建一个 `IF实时库`，写入 `device/line1/status`，读取后确认 TTL 和 value 可见。
6. 新建一个 `IF消息库`，创建 `mock-data` topic，新增变量 `value`，发布 JSON payload。
7. 导出或请求 artifact，确认 `builtinStores.relations` 至少有两个元素，且响应不包含 `if_dev_data`、`p_<projectId>`、`18379`、`18883`、`password`、`broker`。

- [ ] **步骤 4：Commit 验收修复**

```powershell
git status --short
git add data_service datacenter docs/superpowers
git commit -m "fix(datacenter): 完成内置运行库多实例验收"
```

## 自检

- 规格覆盖度：计划覆盖了多实例连接、系统内置标识、名称可改、连接级 SQL/realtime/message API、四类工作台、artifact 集合结构、对象库排除、开发态数据不发布。
- 占位符扫描：计划未使用“待定”“TODO”“类似任务”等占位表达；每个任务包含文件、代码形态、验证命令和预期结果。
- 类型一致性：后端和前端统一使用 `builtin.relation`、`builtin.timeseries`、`builtin.realtime`、`builtin.message`；不可变标识统一为 `runtimeKey`；artifact 集合字段统一为 `relations`、`timeseries`、`realtimeSpaces`、`messageSpaces`。
- 验证路径：后端以 `go test ./...`、`go build ./...` 收口；前端以 `pnpm --filter datacenter typecheck`、`pnpm --filter datacenter lint`、`pnpm --filter datacenter build` 收口；浏览器验收覆盖多实例隔离和 artifact 泄漏检查。
