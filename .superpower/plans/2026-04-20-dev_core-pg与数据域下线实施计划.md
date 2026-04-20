# dev_core PG 切换与数据域下线 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 `dev_core` 从 MySQL 切换到 PostgreSQL，并彻底下线 `dev_core` 内部数据域实现，统一由 `data_service` 提供平台侧数据域能力。

**Architecture:** 先补齐 `data_service` 契约与项目级快照接口，再切换 `datacenter`、`designer` 和 `dev_core` 的消费路径，确认不再依赖 `dev_core` 本地数据域表后，删除 `dev_core` 的数据域路由、模型与表结构，最后完成 `dev_core` 的 PostgreSQL + Sequelize 方言迁移。迁移过程中允许按 PostgreSQL 最佳实践优化表结构、索引和约束，但只围绕正式边界与性能问题展开。

**Tech Stack:** Node.js、Express、Sequelize、pg、Jest、Go、pgx/pgxpool、net/http、Go test、Vue 3、Vite、pnpm、Makefile

---

## File Structure

本次实施会涉及以下文件组，按职责拆分：

- `data_service/internal/http/router/router.go`
  负责补齐数据域正式 HTTP 路由，包含 connections / mqtt / datapoints / preview / compute 的对齐入口。
- `data_service/internal/http/handler/*.go`
  负责承接新增请求参数、响应结构和鉴权上下文。
- `data_service/internal/service/*.go`
  负责数据域正式业务逻辑，包括连接探查、MQTT 完整管理、数据点批量状态/取值、项目级快照。
- `data_service/internal/repository/*.go`
  负责 PostgreSQL 查询、索引命中友好的 SQL、快照读取与 upsert 查询。
- `data_service/internal/db/migrations/*.sql`
  负责数据域 PostgreSQL 表结构清理和性能增强索引。
- `data_service/internal/.../*_test.go`
  负责路由、服务、仓储层新增能力的单测 / 集成测试。

- `datacenter/src/api/data.api.js`
  统一数据工作台对 `data_service` 的正式接口调用。
- `datacenter/src/composables/useMqttConnection.js`
  对齐 MQTT 新契约，去掉对 `dev_core` 旧行为的隐式依赖。
- `designer/src/services/datacenterApi.ts`
  对齐数据点状态与批量值读取接口。
- `designer/src/data/DataService.ts`
  对齐 path/value 读取接口，去掉旧的 `/api/v1/datapoints/*` 假设。

- `dev_core/src/routes/v1/project.js`
  把工程导入导出中的数据域读取改成调用 `data_service` 快照接口。
- `dev_core/src/services/publishService.js`
  把发布打包、manifest 统计改成调用 `data_service` 快照接口。
- `dev_core/src/services/dataDomainClient.js`
  新增控制面到 `data_service` 的受控客户端，负责参数化 HTTP 调用、统一错误映射与超时。
- `dev_core/src/services/__tests__/dataDomainClient.test.js`
  覆盖 `dev_core` 调用 `data_service` 快照与读取接口的行为。

- `dev_core/src/routes/v1/index.js`
  删除 `/data` 路由挂载。
- `dev_core/src/routes/v1/data.js`
  历史数据域路由，最终删除。
- `dev_core/src/controllers/*data*`、`*mqtt*`
  历史数据域控制器，最终删除。
- `dev_core/src/services/dataConnectionService.js`
- `dev_core/src/services/dataQueryService.js`
- `dev_core/src/services/dataPointService.js`
- `dev_core/src/services/mqttService.js`
  历史数据域服务，最终删除。
- `dev_core/src/models/Data*.js`
  历史数据域 Sequelize 模型，最终删除。

- `dev_core/src/config/database.js`
  切换为 PostgreSQL + Sequelize。
- `dev_core/scripts/init-database.js`
  切换为 PostgreSQL 初始化脚本。
- `dev_core/database/init.sql`
  只保留控制面表，并改成 PostgreSQL 方言与索引设计。
- `dev_core/src/models/*.js`
  调整控制面模型字段类型 / `field` 映射 / 原生 SQL 兼容。

- `D:\SVNCode\indu-forge\.env`
- `D:\SVNCode\indu-forge\.env_example`
- `dev_core/database/README.md`
- `docs/backend/database-init.md`
  更新 PostgreSQL 配置口径和初始化说明。

### Task 1: 补齐 data_service 正式契约

**Files:**
- Modify: `D:\SVNCode\indu-forge\data_service\internal\http\router\router.go`
- Modify: `D:\SVNCode\indu-forge\data_service\internal\http\handler\connection_handler.go`
- Modify: `D:\SVNCode\indu-forge\data_service\internal\http\handler\datapoint_handler.go`
- Modify: `D:\SVNCode\indu-forge\data_service\internal\http\handler\mqtt_handler.go`
- Create: `D:\SVNCode\indu-forge\data_service\internal\http\handler\project_snapshot_handler.go`
- Modify: `D:\SVNCode\indu-forge\data_service\internal\service\connection_service.go`
- Modify: `D:\SVNCode\indu-forge\data_service\internal\service\datapoint_service.go`
- Modify: `D:\SVNCode\indu-forge\data_service\internal\service\mqtt_service.go`
- Create: `D:\SVNCode\indu-forge\data_service\internal\service\project_snapshot_service.go`
- Modify: `D:\SVNCode\indu-forge\data_service\internal\repository\connection_repository.go`
- Modify: `D:\SVNCode\indu-forge\data_service\internal\repository\datapoint_repository.go`
- Modify: `D:\SVNCode\indu-forge\data_service\internal\repository\mqtt_repository.go`
- Create: `D:\SVNCode\indu-forge\data_service\internal\repository\project_snapshot_repository.go`
- Test: `D:\SVNCode\indu-forge\data_service\internal\http\handler\connection_handler_test.go`
- Test: `D:\SVNCode\indu-forge\data_service\internal\http\handler\datapoint_handler_test.go`
- Test: `D:\SVNCode\indu-forge\data_service\internal\http\handler\mqtt_handler_test.go`
- Test: `D:\SVNCode\indu-forge\data_service\internal\http\handler\project_snapshot_handler_test.go`

- [ ] **Step 1: 写 connection / datapoint / snapshot 缺口的失败测试**

```go
func TestConnectionHandler_TestConnection(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/data/projects/p1/connections/test", strings.NewReader(`{
	  "type":"relational",
	  "config":{"dbType":"postgresql","host":"127.0.0.1","port":5432}
	}`))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	handler := NewConnectionHandler(&service.ConnectionService{})

	err := handler.TestConnection(rec, req)
	if err == nil {
		t.Fatalf("expected missing implementation error before feature is added")
	}
}

func TestDataPointHandler_GetStatuses(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/data/projects/p1/datapoints/status", strings.NewReader(`{
	  "datapointIds":["dp-1","dp-2"]
	}`))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	handler := NewDataPointHandler(&service.DataPointService{})

	err := handler.GetStatuses(rec, req)
	if err == nil {
		t.Fatalf("expected missing implementation error before feature is added")
	}
}
```

- [ ] **Step 2: 运行 Go 单测，确认按预期失败**

Run: `go test ./internal/http/handler -run "TestConnectionHandler_TestConnection|TestDataPointHandler_GetStatuses" -count=1`  
Expected: FAIL，提示 `TestConnection` / `GetStatuses` / `GetSnapshot` 尚未实现或未注册。

- [ ] **Step 3: 最小实现 router / handler / service / repository 骨架**

```go
// router.go
mux.Handle(
	"POST /api/v1/data/projects/{projectId}/connections/test",
	middleware.Authenticate(opts.jwtValidator)(
		middleware.RequireCapability("project:read")(
			middleware.ErrorHandler(opts.connectionHandler.TestConnection),
		),
	),
)

mux.Handle(
	"POST /api/v1/data/projects/{projectId}/datapoints/status",
	middleware.Authenticate(opts.jwtValidator)(
		middleware.RequireCapability("project:read")(
			middleware.ErrorHandler(opts.dataPointHandler.GetStatuses),
		),
	),
)

mux.Handle(
	"GET /api/v1/data/projects/{projectId}/snapshot",
	middleware.Authenticate(opts.jwtValidator)(
		middleware.RequireCapability("project:read")(
			middleware.ErrorHandler(opts.projectSnapshotHandler.Get),
		),
	),
)
```

```go
// project_snapshot_service.go
type ProjectSnapshot struct {
	Connections       []repository.ConnectionRecord   `json:"connections"`
	Queries           []repository.QueryRecord        `json:"queries"`
	DataPoints        []repository.DataPointRecord    `json:"datapoints"`
	MqttSubscriptions []repository.MqttSubscription   `json:"mqttSubscriptions"`
}

func (s *ProjectSnapshotService) Get(ctx context.Context, projectID string) (*ProjectSnapshot, error) {
	return s.repository.GetByProject(ctx, projectID)
}
```

- [ ] **Step 4: 为缺口能力补 PostgreSQL 友好的实现与参数化 SQL**

```go
const listDataPointStatusesSQL = `
SELECT id, status, updated_at
FROM data_points
WHERE project_id = $1
  AND id = ANY($2::uuid[])
`

func (r *DataPointRepository) ListStatuses(ctx context.Context, projectID string, ids []string) ([]DataPointStatus, error) {
	rows, err := r.pool.Query(ctx, listDataPointStatusesSQL, projectID, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	// scan ...
	return result, nil
}
```

- [ ] **Step 5: 运行处理器与服务测试，确认通过**

Run: `go test ./internal/http/handler ./internal/service ./internal/repository -count=1`  
Expected: PASS

- [ ] **Step 6: 运行 data_service 全量验证**

Run: `make -C data_service test`  
Expected: `ok` 或 `PASS`，无新增失败用例。

- [ ] **Step 7: Commit**

```bash
git add data_service/internal/http/router/router.go data_service/internal/http/handler data_service/internal/service data_service/internal/repository
git commit -m "feat(data_service): 补齐正式数据域契约"
```

### Task 2: 优化 data_service PostgreSQL 表结构与索引

**Files:**
- Modify: `D:\SVNCode\indu-forge\data_service\internal\db\migrations\0001_core_tables.sql`
- Modify: `D:\SVNCode\indu-forge\data_service\internal\db\migrations\0002_mqtt_tables.sql`
- Modify: `D:\SVNCode\indu-forge\data_service\internal\db\migrations\0005_preview_sessions.sql`
- Create: `D:\SVNCode\indu-forge\data_service\internal\db\migrations\0006_performance_indexes.sql`
- Create: `D:\SVNCode\indu-forge\data_service\internal\db\migrations\0006_performance_indexes_down.sql`
- Test: `D:\SVNCode\indu-forge\data_service\tests\integration\migration_test.go`

- [ ] **Step 1: 先写迁移集成测试，锁定目标索引和字段语义**

```go
func TestMigrationAddsProjectPathUniqueIndex(t *testing.T) {
	db := openTestDB(t)
	mustApplyMigrations(t, db)

	var exists bool
	err := db.QueryRow(context.Background(), `
		SELECT EXISTS (
			SELECT 1
			FROM pg_indexes
			WHERE schemaname = 'public'
			  AND tablename = 'data_points'
			  AND indexname = 'data_points_project_path_idx'
		)
	`).Scan(&exists)
	require.NoError(t, err)
	require.True(t, exists)
}
```

- [ ] **Step 2: 运行迁移测试，确认先失败**

Run: `go test ./tests/integration -run TestMigrationAddsProjectPathUniqueIndex -count=1`  
Expected: FAIL，提示索引不存在。

- [ ] **Step 3: 新增性能迁移，只补正式访问路径索引**

```sql
CREATE INDEX IF NOT EXISTS data_connections_project_type_idx
ON data_connections (project_id, type);

CREATE INDEX IF NOT EXISTS data_connections_project_status_idx
ON data_connections (project_id, status);

CREATE UNIQUE INDEX IF NOT EXISTS data_points_project_path_idx
ON data_points (project_id, path);

CREATE INDEX IF NOT EXISTS data_preview_sessions_project_user_status_idx
ON data_preview_sessions (project_id, user_id, status);
```

- [ ] **Step 4: 运行迁移测试，确认通过**

Run: `go test ./tests/integration -run TestMigrationAddsProjectPathUniqueIndex -count=1`  
Expected: PASS

- [ ] **Step 5: 全量运行 data_service 构建与测试**

Run: `make -C data_service test && make -C data_service build`  
Expected: 测试通过，构建成功。

- [ ] **Step 6: Commit**

```bash
git add data_service/internal/db/migrations data_service/tests/integration/migration_test.go
git commit -m "perf(data_service): 优化 PostgreSQL 索引与约束"
```

### Task 3: 切换 datacenter 与 designer 到 data_service 正式契约

**Files:**
- Modify: `D:\SVNCode\indu-forge\datacenter\src\api\data.api.js`
- Modify: `D:\SVNCode\indu-forge\datacenter\src\composables\useMqttConnection.js`
- Modify: `D:\SVNCode\indu-forge\designer\src\services\datacenterApi.ts`
- Modify: `D:\SVNCode\indu-forge\designer\src\data\DataService.ts`
- Modify: `D:\SVNCode\indu-forge\designer\src\data\DiagnosticsStore.ts`
- Test: `D:\SVNCode\indu-forge\datacenter\src\api\__tests__\data.api.test.js`
- Test: `D:\SVNCode\indu-forge\designer\src\data\__tests__\DataService.test.ts`

- [ ] **Step 1: 为前端 API 契约变更写失败测试**

```ts
it("uses data_service datapoint status endpoint", async () => {
  await datacenterApi.getDatapointStatus("p1", ["dp1"]);
  expect(request.post).toHaveBeenCalledWith(
    "/data/projects/p1/datapoints/status",
    { datapointIds: ["dp1"] }
  );
});

it("fetches datapoint values from /api/v1/data path instead of legacy /api/v1/datapoints", async () => {
  const service = new DataService({ baseUrl: "http://localhost:19602" });
  global.fetch = vi.fn().mockResolvedValue(okJson({ success: true, data: [] })) as any;

  await service.fetchValues(["dp.path"]);

  expect(global.fetch).toHaveBeenCalledWith(
    "http://localhost:19602/api/v1/data/datapoints/values",
    expect.any(Object)
  );
});
```

- [ ] **Step 2: 运行前端测试，确认失败**

Run: `pnpm --dir datacenter test -- --runInBand data.api`  
Expected: FAIL，接口路径仍指向旧实现。  

Run: `pnpm --dir designer test -- --runInBand DataService`  
Expected: FAIL，仍请求旧 `/api/v1/datapoints/*`。

- [ ] **Step 3: 最小改动前端接口路径与响应适配**

```ts
// designer/src/data/DataService.ts
const response = await fetch(`${this._baseUrl}/api/v1/data/datapoints/values`, {
  method: "POST",
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify({ paths }),
});
```

```ts
// designer/src/services/datacenterApi.ts
return request.post(`/data/projects/${projectId}/datapoints/status`, {
  datapointIds,
});
```

- [ ] **Step 4: 运行 datacenter / designer 测试**

Run: `pnpm --dir datacenter test`  
Expected: PASS  

Run: `pnpm --dir designer test`  
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add datacenter/src/api/data.api.js datacenter/src/composables/useMqttConnection.js designer/src/services/datacenterApi.ts designer/src/data
git commit -m "refactor(frontend): 对齐 data_service 正式契约"
```

### Task 4: 把 dev_core 导入导出与发布改为依赖 data_service 快照

**Files:**
- Create: `D:\SVNCode\indu-forge\dev_core\src\services\dataDomainClient.js`
- Modify: `D:\SVNCode\indu-forge\dev_core\src\routes\v1\project.js`
- Modify: `D:\SVNCode\indu-forge\dev_core\src\services\publishService.js`
- Test: `D:\SVNCode\indu-forge\dev_core\src\services\__tests__\dataDomainClient.test.js`
- Test: `D:\SVNCode\indu-forge\dev_core\src\services\__tests__\publishService.snapshot.test.js`
- Test: `D:\SVNCode\indu-forge\dev_core\src\routes\v1\__tests__\project.export-import.test.js`

- [ ] **Step 1: 写失败测试，锁定 dev_core 不再读本地数据域模型**

```js
test("export project reads datacenter snapshot from data_service", async () => {
  const client = { getProjectSnapshot: jest.fn().mockResolvedValue({ connections: [], queries: [], datapoints: [] }) };
  const zip = await exportProjectWithClient({ projectId: "p1", dataDomainClient: client });
  expect(client.getProjectSnapshot).toHaveBeenCalledWith("p1");
  expect(zip).toContain("datacenter/connections.json");
});
```

- [ ] **Step 2: 运行 dev_core 相关测试，确认失败**

Run: `pnpm --dir dev_core test -- project.export-import publishService.snapshot`  
Expected: FAIL，仍直接依赖 `DataConnection` / `DataQuery` / `DataPoint`。

- [ ] **Step 3: 新增 dataDomainClient，并替换导出 / 导入 / 发布中的数据域读取**

```js
class DataDomainClient {
  async getProjectSnapshot(projectId, token) {
    const response = await fetch(`${this.baseUrl}/api/v1/data/projects/${projectId}/snapshot`, {
      headers: token ? { Authorization: `Bearer ${token}` } : {},
    });
    if (!response.ok) {
      throw new Error(`data_service snapshot request failed: ${response.status}`);
    }
    const payload = await response.json();
    return payload.data;
  }
}
```

- [ ] **Step 4: 运行 dev_core 测试**

Run: `pnpm --dir dev_core test -- --runInBand project.export-import publishService.snapshot dataDomainClient`  
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add dev_core/src/services/dataDomainClient.js dev_core/src/routes/v1/project.js dev_core/src/services/publishService.js dev_core/src/services/__tests__ dev_core/src/routes/v1/__tests__
git commit -m "refactor(dev_core): 改用 data_service 数据域快照"
```

### Task 5: 删除 dev_core 历史数据域实现

**Files:**
- Modify: `D:\SVNCode\indu-forge\dev_core\src\routes\v1\index.js`
- Delete: `D:\SVNCode\indu-forge\dev_core\src\routes\v1\data.js`
- Delete: `D:\SVNCode\indu-forge\dev_core\src\controllers\dataPointController.js`
- Delete: `D:\SVNCode\indu-forge\dev_core\src\controllers\mqttConnectionController.js`
- Delete: `D:\SVNCode\indu-forge\dev_core\src\controllers\mqttSubscriptionController.js`
- Delete: `D:\SVNCode\indu-forge\dev_core\src\controllers\mqttTagController.js`
- Delete: `D:\SVNCode\indu-forge\dev_core\src\controllers\mqttTagGroupController.js`
- Delete: `D:\SVNCode\indu-forge\dev_core\src\services\dataConnectionService.js`
- Delete: `D:\SVNCode\indu-forge\dev_core\src\services\dataQueryService.js`
- Delete: `D:\SVNCode\indu-forge\dev_core\src\services\dataPointService.js`
- Delete: `D:\SVNCode\indu-forge\dev_core\src\services\mqttService.js`
- Delete: `D:\SVNCode\indu-forge\dev_core\src\models\DataConnection.js`
- Delete: `D:\SVNCode\indu-forge\dev_core\src\models\DataRelationalConfig.js`
- Delete: `D:\SVNCode\indu-forge\dev_core\src\models\DataQuery.js`
- Delete: `D:\SVNCode\indu-forge\dev_core\src\models\DataQueryLog.js`
- Delete: `D:\SVNCode\indu-forge\dev_core\src\models\DataPoint.js`
- Delete: `D:\SVNCode\indu-forge\dev_core\src\models\DataMqttConfig.js`
- Delete: `D:\SVNCode\indu-forge\dev_core\src\models\DataMqttSubscription.js`
- Delete: `D:\SVNCode\indu-forge\dev_core\src\models\DataMqttTagGroup.js`
- Delete: `D:\SVNCode\indu-forge\dev_core\src\models\DataMqttTag.js`
- Modify: `D:\SVNCode\indu-forge\dev_core\src\models\index.js`
- Test: `D:\SVNCode\indu-forge\dev_core\src\app.test.js`

- [ ] **Step 1: 写路由挂载失败测试**

```js
test("v1 router no longer mounts /data routes", async () => {
  const app = buildApp();
  const response = await request(app).get("/api/v1/data/projects/p1/connections");
  expect(response.status).toBe(404);
});
```

- [ ] **Step 2: 运行测试，确认当前失败**

Run: `pnpm --dir dev_core test -- --runInBand app.test`  
Expected: FAIL，`/api/v1/data` 仍被挂载。

- [ ] **Step 3: 删除历史数据域路由、服务、控制器、模型与索引引用**

```js
// routes/v1/index.js
// 删除这一行
// router.use("/data", dataRoutes);

module.exports = { buildV1Router };
```

```js
// models/index.js
// 删除历史 Data* 模型注册与关联，只保留控制面模型导出
module.exports = {
  sequelize,
  Sequelize,
  Tenant,
  User,
  Project,
  DesignPage,
  Deployment,
  Node,
  NodeDeployment,
  NodeCommand,
  Log,
};
```

- [ ] **Step 4: 运行 dev_core 全量测试**

Run: `pnpm --dir dev_core test`  
Expected: PASS，且不再有对历史 Data* 模型的 require 失败。

- [ ] **Step 5: Commit**

```bash
git add dev_core/src/routes/v1/index.js dev_core/src/models/index.js
git rm dev_core/src/routes/v1/data.js dev_core/src/controllers/dataPointController.js dev_core/src/controllers/mqttConnectionController.js dev_core/src/controllers/mqttSubscriptionController.js dev_core/src/controllers/mqttTagController.js dev_core/src/controllers/mqttTagGroupController.js dev_core/src/services/dataConnectionService.js dev_core/src/services/dataQueryService.js dev_core/src/services/dataPointService.js dev_core/src/services/mqttService.js dev_core/src/models/DataConnection.js dev_core/src/models/DataRelationalConfig.js dev_core/src/models/DataQuery.js dev_core/src/models/DataQueryLog.js dev_core/src/models/DataPoint.js dev_core/src/models/DataMqttConfig.js dev_core/src/models/DataMqttSubscription.js dev_core/src/models/DataMqttTagGroup.js dev_core/src/models/DataMqttTag.js
git commit -m "refactor(dev_core): 下线历史数据域实现"
```

### Task 6: dev_core 切 PostgreSQL 并清理控制面表结构

**Files:**
- Modify: `D:\SVNCode\indu-forge\dev_core\src\config\database.js`
- Modify: `D:\SVNCode\indu-forge\dev_core\scripts\init-database.js`
- Modify: `D:\SVNCode\indu-forge\dev_core\database\init.sql`
- Modify: `D:\SVNCode\indu-forge\dev_core\src\models\Tenant.js`
- Modify: `D:\SVNCode\indu-forge\dev_core\src\models\User.js`
- Modify: `D:\SVNCode\indu-forge\dev_core\src\models\Project.js`
- Modify: `D:\SVNCode\indu-forge\dev_core\src\models\Deployment.js`
- Modify: `D:\SVNCode\indu-forge\dev_core\src\models\Node.js`
- Modify: `D:\SVNCode\indu-forge\dev_core\src\models\NodeDeployment.js`
- Modify: `D:\SVNCode\indu-forge\dev_core\src\models\NodeCommand.js`
- Modify: `D:\SVNCode\indu-forge\dev_core\src\routes\v1\project.js`
- Modify: `D:\SVNCode\indu-forge\dev_core\src\services\designService.js`
- Test: `D:\SVNCode\indu-forge\dev_core\src\config\__tests__\database.pg.test.js`
- Test: `D:\SVNCode\indu-forge\dev_core\src\services\__tests__\designService.pg.test.js`

- [ ] **Step 1: 写 PostgreSQL 方言失败测试**

```js
test("database config uses postgres dialect", () => {
  process.env.DB_PORT = "5432";
  jest.resetModules();
  const { sequelize } = require("../database");
  expect(sequelize.getDialect()).toBe("postgres");
});

test("project settings upsert uses postgres ON CONFLICT", async () => {
  const sql = buildProjectSettingsUpsertSQL();
  expect(sql).toContain("ON CONFLICT");
  expect(sql).not.toContain("ON DUPLICATE KEY");
});
```

- [ ] **Step 2: 运行测试，确认先失败**

Run: `pnpm --dir dev_core test -- --runInBand database.pg designService.pg`  
Expected: FAIL，仍使用 MySQL 方言。

- [ ] **Step 3: 改造 PostgreSQL 连接、初始化与 Sequelize 字段映射**

```js
const sequelize = new Sequelize(
  process.env.DB_NAME || "tenant_management",
  process.env.DB_USER || "postgres",
  process.env.DB_PASSWORD || "",
  {
    host: process.env.DB_HOST || "127.0.0.1",
    port: Number(process.env.DB_PORT || 5432),
    dialect: "postgres",
    logging: false,
    pool: {
      max: Number(process.env.DB_MAX_CONNECTIONS || 30),
      min: 2,
      idle: 30000,
      acquire: 10000,
    },
    dialectOptions: {
      application_name: "indu-forge-dev_core",
    },
  }
);
```

```sql
CREATE TABLE IF NOT EXISTS projects (
  id uuid PRIMARY KEY,
  tenant_id uuid NOT NULL REFERENCES tenants(id),
  name text NOT NULL,
  status text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'archived', 'deleted')),
  project_variables jsonb,
  entry_config jsonb,
  created_at timestamptz NOT NULL,
  updated_at timestamptz NOT NULL
);

CREATE INDEX IF NOT EXISTS projects_tenant_status_idx
ON projects (tenant_id, status);
```

- [ ] **Step 4: 改写原始 SQL 为 PostgreSQL 语法**

```js
const PROJECT_SETTINGS_UPSERT_SQL = `
INSERT INTO design_project_settings (
  project_id, schema_version, global_variables, global_scripts, updated_by, updated_at
) VALUES ($1, $2, $3::jsonb, $4::jsonb, $5, $6)
ON CONFLICT (project_id) DO UPDATE SET
  global_variables = EXCLUDED.global_variables,
  global_scripts = EXCLUDED.global_scripts,
  updated_by = EXCLUDED.updated_by,
  updated_at = EXCLUDED.updated_at
`;
```

- [ ] **Step 5: 更新根环境与文档**

```env
DB_HOST=172.21.242.174
DB_PORT=5432
DB_NAME=tenant_management
DB_USER=postgres
DB_PASSWORD=postgres
DATA_SERVICE_DATABASE_URL=postgres://postgres:postgres@172.21.242.174:5432/data_service?sslmode=disable
```

- [ ] **Step 6: 运行初始化与测试**

Run: `pnpm --dir dev_core db:init`  
Expected: 成功创建 PostgreSQL 控制面表与初始数据。  

Run: `pnpm --dir dev_core test`  
Expected: PASS  

Run: `make -C data_service test && make -C data_service build`  
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add .env .env_example dev_core/src/config/database.js dev_core/scripts/init-database.js dev_core/database/init.sql dev_core/src/models dev_core/src/routes/v1/project.js dev_core/src/services/designService.js dev_core/database/README.md docs/backend/database-init.md
git commit -m "feat(dev_core): 切换 PostgreSQL 控制面数据库"
```

## Self-Review

- 规格覆盖检查：
  - `data_service` 正式接管 connections / mqtt / datapoints / preview / compute：由 Task 1、Task 2 覆盖。
  - `datacenter`、`designer` 不再依赖 `dev_core` 数据域：由 Task 3 覆盖。
  - `dev_core` 导入导出 / 发布改用项目快照：由 Task 4 覆盖。
  - `dev_core` 彻底下线 `/api/v1/data` 与历史数据域表：由 Task 5 覆盖。
  - `dev_core` 切 PostgreSQL、保留 Sequelize、顺手做索引与结构优化：由 Task 6 覆盖。
- Placeholder 扫描：
  - 计划中没有 `TODO` / `TBD` / “后续再补” 之类占位语句。
  - 每个任务都给出了目标文件、测试入口、命令和最小代码骨架。
- 类型与命名一致性：
  - 全文统一使用 `data_service` 项目快照 `snapshot` 概念。
  - `dev_core` 统一保留 Sequelize，未混入第二套 ORM。
  - PostgreSQL 结构统一采用 `snake_case` 物理字段，JavaScript 侧通过模型映射承接。

