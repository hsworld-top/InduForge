# Data Service (PG) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在仓库根目录落地独立 Go `data_service`，完整承接并扩展数据中心设计能力（协议矩阵、preview、compute、alarm、runtime_inapp、发布契约），并保持 datacenter 可联调。

**Architecture:** 采用 Go 单体服务的分层结构（HTTP Handler -> Service -> Repository），统一 `ApiResponse/AppError/ErrorCodes`，数据层固定 PostgreSQL + Redis，执行层包含 Node/Python 子进程沙盒与 CEL 报警规则引擎。前端继续由 `datacenter` 承载最小页面，Nginx 仅做路由分流，不经过 `dev_core` 转发数据请求。

**Tech Stack:** Go 1.24、gorilla/mux、pgx、go-redis、golang-jwt、cel-go、Node/Python 子进程、PostgreSQL、Redis、Vue3 + Element Plus

---

## Scope Check（范围拆解）

该规格覆盖多个独立子域（协议接入、preview 会话、compute、alarm、通知、发布契约、前端最小页、网关分流）。
为保持可执行性，本计划按 11 里程碑拆为 14 个任务（每个任务可独立提交），并通过“主阻塞链 + 并行链”推进：

1. 主阻塞链：M1 -> M2 -> M3 -> M4 -> M5 -> M6 -> M7 -> M8 -> M11
2. 并行链：M9 在 M5 后并行，M10 依赖 M5+M9，M11 依赖 M8+M10

## File Structure（先锁定边界）

### 新建目录（data_service）

1. `data_service/cmd/main.go`：服务入口。
2. `data_service/internal/app/server.go`：应用装配（配置、路由、依赖注入）。
3. `data_service/internal/config/config.go`：环境变量加载与校验。
4. `data_service/internal/http/router/router.go`：统一路由注册。
5. `data_service/internal/http/response/api_response.go`：统一响应结构。
6. `data_service/internal/errors/{app_error.go,error_codes.go}`：业务错误模型。
7. `data_service/internal/http/middleware/*`：requestId、鉴权、能力校验、错误恢复。
8. `data_service/internal/db/postgres/*`：连接池、事务、迁移执行。
9. `data_service/internal/db/migrations/*.sql`：全量 PG 表迁移脚本。
10. `data_service/internal/repository/*`：按领域拆仓储。
11. `data_service/internal/service/*`：按领域拆业务逻辑。
12. `data_service/internal/engine/compute/*`：JS/Python 执行器。
13. `data_service/internal/engine/alarm/*`：CEL 引擎、状态机、事件。
14. `data_service/internal/notify/runtime_inapp/*`：站内通知投递。
15. `data_service/internal/publish/contract/*`：发布契约导出。
16. `data_service/tests/{unit,integration,e2e}/*`：测试。
17. `data_service/Makefile`：build/test/lint/migrate 命令入口。

### 现有目录改动

1. `datacenter/src/api/data.api.js`：新增 compute/alarm/preview/runtime_inapp API。
2. `datacenter/src/views/DataCenterNew.vue`：接入最小页入口（替换“规划中”占位）。
3. `datacenter/src/components/compute/*`：计算最小页面。
4. `datacenter/src/components/alarm/*`：报警最小页面。
5. `datacenter/src/components/preview/*`：会话最小页面。
6. `datacenter/src/components/notification/*`：runtime_inapp 最小页面。
7. `nginx/nginx.conf`：`/api/v1/data/` 分流到 `data_service`。
8. `.env_example`：新增 `DATA_SERVICE_PORT/VITE_DATA_SERVICE_URL` 说明。
9. `docs/datacenter/*`：接口与联调文档更新。

---

### Task 1: 初始化 data_service 基础骨架（M1） ✅（已完成）

**Files:**
- Create: `data_service/go.mod`
- Create: `data_service/Makefile`
- Create: `data_service/cmd/main.go`
- Create: `data_service/internal/app/server.go`
- Create: `data_service/internal/config/config.go`
- Create: `data_service/internal/http/router/router.go`
- Create: `data_service/internal/http/handler/health_handler.go`
- Test: `data_service/internal/http/handler/health_handler_test.go`

- [x] **Step 1: 写健康检查失败测试**

```go
package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthHandler_ReturnsHealthy(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()

	NewHealthHandler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if rr.Body.String() != "healthy\n" {
		t.Fatalf("expected healthy body, got %q", rr.Body.String())
	}
}
```

- [x] **Step 2: 运行测试确认失败**

Run: `go test ./internal/http/handler -run TestHealthHandler_ReturnsHealthy -v`  
Expected: FAIL，提示 `NewHealthHandler` 未定义。

- [x] **Step 3: 实现最小健康路由与入口**

```go
package handler

import "net/http"

func NewHealthHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("healthy\n"))
	})
}
```

- [x] **Step 4: 运行目标测试与全量编译**

Run: `go test ./internal/http/handler -run TestHealthHandler_ReturnsHealthy -v`  
Expected: PASS

Run: `go test ./...`  
Expected: PASS（当前仅基础包）。

- [x] **Step 5: 提交**

```bash
git add data_service
git commit -m "feat(product): 初始化 data_service 基础骨架与健康检查"
```

### Task 2: 统一 ApiResponse + AppError + ErrorCodes（M1） ✅（已完成）

**Files:**
- Create: `data_service/internal/http/response/api_response.go`
- Create: `data_service/internal/errors/app_error.go`
- Create: `data_service/internal/errors/error_codes.go`
- Create: `data_service/internal/http/middleware/request_id.go`
- Create: `data_service/internal/http/middleware/error_handler.go`
- Test: `data_service/internal/http/response/api_response_test.go`
- Test: `data_service/internal/http/middleware/request_id_test.go`

- [x] **Step 1: 写 ApiResponse 与 requestId 失败测试**

```go
func TestSuccessResponse_ContainsRequestID(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-ID", "rid-1")

	response.WriteSuccess(rr, req, map[string]string{"ok": "1"})

	if !strings.Contains(rr.Body.String(), "\"requestId\":\"rid-1\"") {
		t.Fatalf("requestId not found: %s", rr.Body.String())
	}
}
```

- [x] **Step 2: 运行测试确认失败**

Run: `go test ./internal/http/response -run TestSuccessResponse_ContainsRequestID -v`  
Expected: FAIL，`WriteSuccess` 未定义。

- [x] **Step 3: 实现响应模型与中间件**

```go
type ApiResponse struct {
	Success   bool   `json:"success"`
	ErrorCode string `json:"errorCode"`
	Message   string `json:"message"`
	RequestID string `json:"requestId"`
	Data      any    `json:"data"`
}
```

```go
func WriteSuccess(w http.ResponseWriter, r *http.Request, data any) {
	resp := ApiResponse{Success: true, ErrorCode: "", Message: "OK", RequestID: requestIDFromContext(r.Context()), Data: data}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}
```

- [x] **Step 4: 运行测试**

Run: `go test ./internal/http/response ./internal/http/middleware -v`  
Expected: PASS

- [x] **Step 5: 提交**

```bash
git add data_service/internal/http data_service/internal/errors
git commit -m "feat(product): 新增统一 ApiResponse 与错误处理中间件"
```

### Task 3: JWT 验签与 capability 校验基线（M2） ✅（已完成）

**Files:**
- Create: `data_service/internal/auth/jwt_validator.go`
- Create: `data_service/internal/auth/claims.go`
- Create: `data_service/internal/http/middleware/authenticate.go`
- Create: `data_service/internal/http/middleware/capability_guard.go`
- Test: `data_service/internal/http/middleware/authenticate_test.go`
- Test: `data_service/internal/http/middleware/capability_guard_test.go`

- [x] **Step 1: 写鉴权失败测试**

```go
func TestAuthenticate_RejectsInvalidToken(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/data/projects/p1/connections", nil)
	req.Header.Set("Authorization", "Bearer bad-token")
	rr := httptest.NewRecorder()

	h := middleware.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}
```

- [x] **Step 2: 运行测试确认失败**

Run: `go test ./internal/http/middleware -run TestAuthenticate_RejectsInvalidToken -v`  
Expected: FAIL，`Authenticate` 未定义。

- [x] **Step 3: 实现 JWT + capability guard**

```go
type Claims struct {
	UserID        string   `json:"userId"`
	TenantID      string   `json:"tenantId"`
	ProjectIDs    []string `json:"projectIds"`
	Capabilities  []string `json:"capabilities"`
	jwt.RegisteredClaims
}
```

```go
func RequireCapability(required string) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := auth.ClaimsFromContext(r.Context())
			if !contains(claims.Capabilities, required) {
				response.WriteAppError(w, r, errors.NewPermissionDenied("缺少能力点: "+required))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
```

- [x] **Step 4: 跑鉴权测试**

Run: `go test ./internal/http/middleware -v`  
Expected: PASS

- [x] **Step 5: 提交**

```bash
git add data_service/internal/auth data_service/internal/http/middleware
git commit -m "feat(product): 实现 JWT 验签与 capability 授权基线"
```

### Task 4: PostgreSQL 迁移框架与核心兼容表（M3） ✅（已完成）

**Files:**
- Create: `data_service/internal/db/postgres/pool.go`
- Create: `data_service/internal/db/migrate/migrator.go`
- Create: `data_service/internal/db/migrations/0001_core_tables.sql`
- Create: `data_service/internal/db/migrations/0001_core_tables_down.sql`
- Test: `data_service/tests/integration/migration_test.go`

- [x] **Step 1: 写 migration 集成失败测试**

```go
func TestMigrateUp_CreatesCoreTables(t *testing.T) {
	db := testutil.NewPostgres(t)
	if err := migrate.Up(db.DSN()); err != nil {
		t.Fatalf("migrate up failed: %v", err)
	}
	testutil.AssertTableExists(t, db, "data_connections")
	testutil.AssertTableExists(t, db, "data_points")
}
```

- [x] **Step 2: 运行测试确认失败**

Run: `go test ./tests/integration -run TestMigrateUp_CreatesCoreTables -v`  
Expected: FAIL，migration 执行器不存在。

- [x] **Step 3: 按 design-postgres-tables 规则实现核心表**

```sql
CREATE TABLE data_connections (
  id uuid PRIMARY KEY,
  project_id uuid NOT NULL,
  tenant_id uuid NOT NULL,
  name varchar(128) NOT NULL,
  type varchar(32) NOT NULL,
  status varchar(32) NOT NULL DEFAULT 'unknown',
  config jsonb NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE(project_id, name)
);
CREATE INDEX idx_data_connections_project_type ON data_connections(project_id, type);
```

- [x] **Step 4: 跑 migration 与索引检查**

Run: `go test ./tests/integration -run TestMigrateUp_CreatesCoreTables -v`  
Expected: PASS

Run: `go test ./tests/integration -run TestMigrationIndexes -v`  
Expected: PASS

- [x] **Step 5: 提交**

```bash
git add data_service/internal/db data_service/tests/integration
git commit -m "feat(product): 建立 PG 迁移框架与核心兼容表"
```

### Task 5: Connections 领域接口承接（M4） ✅（已完成）

**Files:**
- Create: `data_service/internal/repository/connection_repository.go`
- Create: `data_service/internal/service/connection_service.go`
- Create: `data_service/internal/http/handler/connection_handler.go`
- Modify: `data_service/internal/http/router/router.go`
- Test: `data_service/tests/integration/connection_api_test.go`

- [x] **Step 1: 写 Connections CRUD 失败用例**

```go
func TestConnectionsCRUD(t *testing.T) {
	srv := testserver.New(t)
	created := srv.MustCreateConnection(t, "p1", map[string]any{"name": "pg-main", "type": "relational"})
	srv.MustUpdateConnection(t, "p1", created.ID, map[string]any{"name": "pg-main-2"})
	list := srv.MustListConnections(t, "p1")
	if len(list) != 1 { t.Fatalf("expected 1, got %d", len(list)) }
}
```

- [x] **Step 2: 跑测试确认失败**

Run: `go test ./tests/integration -run TestConnectionsCRUD -v`  
Expected: FAIL，路由未注册。

- [x] **Step 3: 实现参数化查询与 Handler**

```go
const insertSQL = `
INSERT INTO data_connections (id, project_id, tenant_id, name, type, status, config)
VALUES ($1,$2,$3,$4,$5,$6,$7)
RETURNING id, project_id, tenant_id, name, type, status, config, created_at, updated_at`
```

```go
r.HandleFunc("/api/v1/data/projects/{projectId}/connections", h.List).Methods(http.MethodGet)
r.HandleFunc("/api/v1/data/projects/{projectId}/connections", h.Create).Methods(http.MethodPost)
r.HandleFunc("/api/v1/data/projects/{projectId}/connections/{connectionId}", h.Update).Methods(http.MethodPut)
r.HandleFunc("/api/v1/data/projects/{projectId}/connections/{connectionId}", h.Delete).Methods(http.MethodDelete)
```

- [x] **Step 4: 跑 connections 集成测试**

Run: `go test ./tests/integration -run TestConnectionsCRUD -v`  
Expected: PASS

- [x] **Step 5: 提交**

```bash
git add data_service/internal/repository data_service/internal/service data_service/internal/http/handler data_service/tests/integration
git commit -m "feat(product): 承接 connections 领域接口"
```

### Task 6: Queries + Datapoints 接口承接（M5） ✅（已完成）

**Files:**
- Create: `data_service/internal/repository/query_repository.go`
- Create: `data_service/internal/repository/datapoint_repository.go`
- Create: `data_service/internal/service/query_service.go`
- Create: `data_service/internal/service/datapoint_service.go`
- Create: `data_service/internal/http/handler/query_handler.go`
- Create: `data_service/internal/http/handler/datapoint_handler.go`
- Modify: `data_service/internal/http/router/router.go`
- Test: `data_service/tests/integration/query_datapoint_api_test.go`

- [x] **Step 1: 写查询执行与数据点用例（失败）**

```go
func TestExecuteQueryAndDataPointValue(t *testing.T) {
	srv := testserver.New(t)
	queryID := srv.MustCreateQuery(t, "p1", "SELECT 1 AS v")
	ret := srv.MustExecuteQuery(t, queryID)
	if ret.RowCount != 1 { t.Fatalf("expected row_count=1, got %d", ret.RowCount) }
}
```

- [x] **Step 2: 运行失败验证**

Run: `go test ./tests/integration -run TestExecuteQueryAndDataPointValue -v`  
Expected: FAIL。

- [x] **Step 3: 实现 query/datapoint handler 与 service**

```go
r.HandleFunc("/api/v1/data/projects/{projectId}/queries", qh.List).Methods(http.MethodGet)
r.HandleFunc("/api/v1/data/projects/{projectId}/queries", qh.Create).Methods(http.MethodPost)
r.HandleFunc("/api/v1/data/queries/{id}/execute", qh.Execute).Methods(http.MethodPost)
r.HandleFunc("/api/v1/data/projects/{projectId}/datapoints", dh.List).Methods(http.MethodGet)
r.HandleFunc("/api/v1/data/projects/{projectId}/datapoints/{id}", dh.Update).Methods(http.MethodPut)
```

- [x] **Step 4: 跑集成测试**

Run: `go test ./tests/integration -run TestExecuteQueryAndDataPointValue -v`  
Expected: PASS

Run: `go test ./tests/integration -run TestDataPointBatchDelete -v`  
Expected: PASS

- [x] **Step 5: 提交**

```bash
git add data_service/internal/repository data_service/internal/service data_service/internal/http/handler data_service/tests/integration
git commit -m "feat(product): 承接 queries 与 datapoints 接口"
```

### Task 7: MQTT 全链路承接（M6） ✅（已完成）

**Files:**
- Create: `data_service/internal/repository/mqtt_repository.go`
- Create: `data_service/internal/service/mqtt_service.go`
- Create: `data_service/internal/http/handler/mqtt_handler.go`
- Create: `data_service/internal/db/migrations/0002_mqtt_tables.sql`
- Test: `data_service/tests/integration/mqtt_api_test.go`

- [x] **Step 1: 写 MQTT 生命周期失败用例**

```go
func TestMqttConnectionLifecycle(t *testing.T) {
	srv := testserver.New(t)
	conn := srv.MustCreateMqttConnection(t, "p1")
	srv.MustStartMqttConnection(t, "p1", conn.ID)
	status := srv.MustGetMqttStatus(t, "p1", conn.ID)
	if status != "connected" { t.Fatalf("expected connected, got %s", status) }
}
```

- [x] **Step 2: 运行失败验证**

Run: `go test ./tests/integration -run TestMqttConnectionLifecycle -v`  
Expected: FAIL。

- [x] **Step 3: 实现 MQTT 仓储与接口**

```go
r.HandleFunc("/api/v1/data/projects/{projectId}/mqtt/connections", mh.CreateConnection).Methods(http.MethodPost)
r.HandleFunc("/api/v1/data/projects/{projectId}/mqtt/connections/{connectionId}/start", mh.StartConnection).Methods(http.MethodPost)
r.HandleFunc("/api/v1/data/projects/{projectId}/mqtt/subscriptions/{subscriptionId}/messages", mh.ListMessages).Methods(http.MethodGet)
```

- [x] **Step 4: 运行 MQTT 用例**

Run: `go test ./tests/integration -run TestMqttConnectionLifecycle -v`  
Expected: PASS

- [x] **Step 5: 提交**

```bash
git add data_service/internal/repository data_service/internal/service data_service/internal/http/handler data_service/internal/db/migrations data_service/tests/integration
git commit -m "feat(product): 承接 MQTT 全链路接口与数据表"
```

### Task 8: 协议矩阵第一波（M7） ✅（已完成）

**Files:**
- Create: `data_service/internal/protocol/kafka/*`
- Create: `data_service/internal/protocol/httpx/*`
- Create: `data_service/internal/protocol/websocketx/*`
- Create: `data_service/internal/protocol/redisx/*`
- Create: `data_service/internal/http/handler/protocol_wave1_handler.go`
- Create: `data_service/internal/db/migrations/0003_protocol_wave1.sql`
- Test: `data_service/tests/integration/protocol_wave1_test.go`

- [x] **Step 1: 写 wave1 协议失败用例**

```go
func TestKafkaSourcePreview(t *testing.T) {
	srv := testserver.New(t)
	cfg := srv.MustCreateKafkaConfig(t, "p1")
	rows := srv.MustPreviewKafkaTopic(t, "p1", cfg.ID)
	if len(rows) == 0 { t.Fatalf("expected preview rows") }
}
```

- [x] **Step 2: 运行失败验证**

Run: `go test ./tests/integration -run TestKafkaSourcePreview -v`  
Expected: FAIL。

- [x] **Step 3: 实现 wave1 协议 handler 与仓储**

```go
r.HandleFunc("/api/v1/data/projects/{projectId}/kafka/configs", h.CreateKafkaConfig).Methods(http.MethodPost)
r.HandleFunc("/api/v1/data/projects/{projectId}/http/configs", h.CreateHTTPConfig).Methods(http.MethodPost)
r.HandleFunc("/api/v1/data/projects/{projectId}/websocket/configs", h.CreateWebSocketConfig).Methods(http.MethodPost)
r.HandleFunc("/api/v1/data/projects/{projectId}/redis/configs", h.CreateRedisConfig).Methods(http.MethodPost)
```

- [x] **Step 4: 跑 wave1 集成测试**

Run: `go test ./tests/integration -run TestProtocolWave1 -v`  
Expected: PASS

- [x] **Step 5: 提交**

```bash
git add data_service/internal/protocol data_service/internal/http/handler data_service/internal/db/migrations data_service/tests/integration
git commit -m "feat(product): 落地协议矩阵第一波（kafka/http/ws/redis）"
```

### Task 9: 协议矩阵第二波 + OPC DA 契约（M8）

**Files:**
- Create: `data_service/internal/protocol/opcua/*`
- Create: `data_service/internal/protocol/s7/*`
- Create: `data_service/internal/protocol/modbus/*`
- Create: `data_service/internal/protocol/tdengine/*`
- Create: `data_service/internal/protocol/opcda_contract/*`
- Create: `data_service/internal/http/handler/protocol_wave2_handler.go`
- Create: `data_service/internal/db/migrations/0004_protocol_wave2.sql`
- Test: `data_service/tests/integration/protocol_wave2_test.go`

- [x] **Step 1: 写 wave2 协议失败用例**

```go
func TestOpcDaContractValidation(t *testing.T) {
	srv := testserver.New(t)
	err := srv.ValidateOpcDaContract(map[string]any{"itemPath": "", "samplingMs": 1000})
	if err == nil { t.Fatalf("expected validation error") }
}
```

- [x] **Step 2: 运行失败验证**

Run: `go test ./tests/integration -run TestOpcDaContractValidation -v`  
Expected: FAIL。

- [x] **Step 3: 实现 wave2 协议路由与模型**

```go
r.HandleFunc("/api/v1/data/projects/{projectId}/opcua/configs", h.CreateOpcuaConfig).Methods(http.MethodPost)
r.HandleFunc("/api/v1/data/projects/{projectId}/s7/configs", h.CreateS7Config).Methods(http.MethodPost)
r.HandleFunc("/api/v1/data/projects/{projectId}/modbus/configs", h.CreateModbusConfig).Methods(http.MethodPost)
r.HandleFunc("/api/v1/data/projects/{projectId}/opcda/contracts/validate", h.ValidateOpcDaContract).Methods(http.MethodPost)
```

- [x] **Step 4: 运行 wave2 用例**

Run: `go test ./tests/integration -run TestProtocolWave2 -v`  
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add data_service/internal/protocol data_service/internal/http/handler data_service/internal/db/migrations data_service/tests/integration
git commit -m "feat(product): 落地协议矩阵第二波与 OPC DA 契约"
```

### Task 10: Preview 会话域（M9）

**Files:**
- Create: `data_service/internal/service/preview_session_service.go`
- Create: `data_service/internal/repository/preview_session_repository.go`
- Create: `data_service/internal/http/handler/preview_handler.go`
- Create: `data_service/internal/cache/redis_client.go`
- Create: `data_service/internal/db/migrations/0005_preview_sessions.sql`
- Test: `data_service/tests/integration/preview_session_test.go`

- [ ] **Step 1: 写 preview 滑动过期失败测试**

```go
func TestPreviewSessionSlidingTTL(t *testing.T) {
	srv := testserver.New(t)
	s := srv.MustCreatePreviewSession(t, "p1")
	srv.MustHeartbeatSession(t, s.ID)
	ttl := srv.MustGetSessionTTL(t, s.ID)
	if ttl <= 0 { t.Fatalf("expected ttl > 0") }
}
```

- [ ] **Step 2: 运行失败验证**

Run: `go test ./tests/integration -run TestPreviewSessionSlidingTTL -v`  
Expected: FAIL。

- [ ] **Step 3: 实现 preview 接口与 Redis key 规范**

```go
const (
	SessionKeyFmt       = "preview:session:%s"
	SessionConnKeyFmt   = "preview:session:%s:connections"
	SessionSubKeyFmt    = "preview:session:%s:subscriptions"
	SessionTTL          = 30 * time.Minute
)
```

```go
r.HandleFunc("/api/v1/data/projects/{projectId}/preview/sessions", h.Create).Methods(http.MethodPost)
r.HandleFunc("/api/v1/data/preview/sessions/{sessionId}/heartbeat", h.Heartbeat).Methods(http.MethodPost)
r.HandleFunc("/api/v1/data/preview/sessions/{sessionId}", h.Delete).Methods(http.MethodDelete)
```

- [ ] **Step 4: 运行 preview 用例**

Run: `go test ./tests/integration -run TestPreviewSession -v`  
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add data_service/internal/service data_service/internal/repository data_service/internal/http/handler data_service/internal/cache data_service/internal/db/migrations data_service/tests/integration
git commit -m "feat(product): 实现 preview 会话与 Redis 滑动过期机制"
```

### Task 11: Compute 引擎（Node/Python 沙盒）+ 最小前端（M10）

**Files:**
- Create: `data_service/internal/engine/compute/node_runner.go`
- Create: `data_service/internal/engine/compute/python_runner.go`
- Create: `data_service/internal/engine/compute/scheduler.go`
- Create: `data_service/internal/service/compute_service.go`
- Create: `data_service/internal/http/handler/compute_handler.go`
- Create: `data_service/internal/db/migrations/0006_compute_tables.sql`
- Modify: `datacenter/src/api/data.api.js`
- Create: `datacenter/src/components/compute/ComputeUnitPanel.vue`
- Modify: `datacenter/src/views/DataCenterNew.vue`
- Test: `data_service/tests/integration/compute_test.go`

- [ ] **Step 1: 写 compute 失败测试（JS/Python/超时）**

```go
func TestComputeRunTimeout(t *testing.T) {
	srv := testserver.New(t)
	unit := srv.MustCreateComputeUnit(t, "p1", "js", "while(true){}")
	err := srv.RunComputeUnit(t, unit.ID)
	if !errors.Is(err, compute.ErrTimeout) {
		t.Fatalf("expected timeout error, got %v", err)
	}
}
```

- [ ] **Step 2: 运行失败验证**

Run: `go test ./tests/integration -run TestComputeRunTimeout -v`  
Expected: FAIL。

- [ ] **Step 3: 实现子进程沙盒与 compute API**

```go
cmd := exec.CommandContext(ctx, nodeBinaryPath, "-e", script)
cmd.Env = []string{"NODE_ENV=production"}
cmd.Dir = runtimeDir
```

```go
r.HandleFunc("/api/v1/data/projects/{projectId}/compute-units", h.Create).Methods(http.MethodPost)
r.HandleFunc("/api/v1/data/projects/{projectId}/compute-units/{id}/run", h.Run).Methods(http.MethodPost)
r.HandleFunc("/api/v1/data/projects/{projectId}/compute-units/{id}/debug", h.Debug).Methods(http.MethodPost)
```

- [ ] **Step 4: 运行后端与前端最小验证**

Run: `go test ./tests/integration -run TestCompute -v`  
Expected: PASS

Run: `pnpm --dir datacenter build`  
Expected: BUILD SUCCESS（compute 面板可编译）。

- [ ] **Step 5: 提交**

```bash
git add data_service datacenter/src/api/data.api.js datacenter/src/components/compute datacenter/src/views/DataCenterNew.vue
git commit -m "feat(product): 实现 compute 引擎与最小前端页面"
```

### Task 12: Alarm + runtime_inapp + 发布契约 + 最小前端（M11）

**Files:**
- Create: `data_service/internal/engine/alarm/cel_engine.go`
- Create: `data_service/internal/engine/alarm/state_machine.go`
- Create: `data_service/internal/service/alarm_service.go`
- Create: `data_service/internal/notify/runtime_inapp/dispatcher.go`
- Create: `data_service/internal/publish/contract/builder.go`
- Create: `data_service/internal/http/handler/alarm_handler.go`
- Create: `data_service/internal/http/handler/notify_handler.go`
- Create: `data_service/internal/http/handler/publish_contract_handler.go`
- Create: `data_service/internal/db/migrations/0007_alarm_notify_tables.sql`
- Modify: `datacenter/src/api/data.api.js`
- Create: `datacenter/src/components/alarm/AlarmPanel.vue`
- Create: `datacenter/src/components/notification/RuntimeInappPanel.vue`
- Modify: `datacenter/src/views/DataCenterNew.vue`
- Test: `data_service/tests/integration/alarm_notify_contract_test.go`

- [ ] **Step 1: 写报警状态机与通知失败测试**

```go
func TestAlarmStateTransition(t *testing.T) {
	sm := alarm.NewStateMachine()
	state := sm.Apply(alarm.StateNormal, alarm.EventRaised)
	if state != alarm.StateActive { t.Fatalf("expected active, got %s", state) }
}
```

```go
func TestRuntimeInappIdempotency(t *testing.T) {
	d := runtimeinapp.NewDeduper()
	key := "event1-policy1-channel1-user1"
	if !d.Accept(key) { t.Fatalf("first delivery must pass") }
	if d.Accept(key) { t.Fatalf("duplicate delivery must be rejected") }
}
```

- [ ] **Step 2: 运行失败验证**

Run: `go test ./tests/integration -run TestAlarmStateTransition -v`  
Expected: FAIL。

- [ ] **Step 3: 实现 CEL + 状态机 + 通知 + 发布契约 API**

```go
env, _ := cel.NewEnv(
	cel.Variable("value", cel.DoubleType),
	cel.Variable("threshold", cel.DoubleType),
)
ast, issues := env.Compile(expr)
if issues != nil && issues.Err() != nil {
	return errors.NewValidation("CEL 编译失败", issues.Err())
}
```

```go
r.HandleFunc("/api/v1/data/projects/{projectId}/alarm-rules", h.CreateRule).Methods(http.MethodPost)
r.HandleFunc("/api/v1/data/projects/{projectId}/alarm-events/{id}/ack", h.Ack).Methods(http.MethodPost)
r.HandleFunc("/api/v1/data/projects/{projectId}/alarm-events/{id}/silence", h.Silence).Methods(http.MethodPost)
r.HandleFunc("/api/v1/data/projects/{projectId}/publish/contracts", p.Export).Methods(http.MethodGet)
```

- [ ] **Step 4: 跑后端与前端验证**

Run: `go test ./tests/integration -run TestAlarmNotifyContract -v`  
Expected: PASS

Run: `pnpm --dir datacenter build`  
Expected: BUILD SUCCESS（alarm/runtime_inapp 面板可编译）。

- [ ] **Step 5: 提交**

```bash
git add data_service datacenter/src/api/data.api.js datacenter/src/components/alarm datacenter/src/components/notification datacenter/src/views/DataCenterNew.vue
git commit -m "feat(product): 实现 alarm/runtime_inapp/发布契约闭环"
```

### Task 13: Nginx 分流、环境变量、联调脚本（上线准备）

**Files:**
- Modify: `nginx/nginx.conf`
- Modify: `.env_example`
- Create: `data_service/scripts/check-nginx.ps1`
- Create: `data_service/scripts/smoke-test.ps1`
- Create: `docs/datacenter/data-service-debug-runbook.md`

- [ ] **Step 1: 写分流规则失败检查脚本**

```powershell
$cfg = Get-Content -Raw -LiteralPath "nginx/nginx.conf"
if ($cfg -notmatch "location /api/v1/data/") { throw "missing data_service route" }
if ($cfg -notmatch "upstream data_service") { throw "missing data_service upstream" }
```

- [ ] **Step 2: 运行检查确认失败**

Run: `pwsh -File data_service/scripts/check-nginx.ps1`  
Expected: FAIL，提示缺少 `upstream data_service`。

- [ ] **Step 3: 实现分流与 runbook**

```nginx
upstream data_service {
    server localhost:19099;
    keepalive 32;
}

location /api/v1/data/ {
    proxy_pass http://data_service;
    proxy_http_version 1.1;
    proxy_set_header Host $host;
    proxy_set_header X-Request-ID $request_id;
}
```

- [ ] **Step 4: 执行 smoke test**

Run: `pwsh -File data_service/scripts/smoke-test.ps1`  
Expected: `/health` 与 `/api/v1/data/...` 返回符合 `ApiResponse`。

- [ ] **Step 5: 提交**

```bash
git add nginx/nginx.conf .env_example data_service/scripts docs/datacenter/data-service-debug-runbook.md
git commit -m "chore(product): 完成 data_service 分流与联调脚本"
```

### Task 14: 全链路回归与验收报告（交付闸门）

**Files:**
- Create: `data_service/tests/e2e/full_chain_test.go`
- Create: `docs/datacenter/data-service-uat-checklist.md`
- Create: `docs/datacenter/data-service-risk-log.md`

- [ ] **Step 1: 写 E2E 失败用例（协议->compute->alarm->通知）**

```go
func TestFullChain_DataToAlarmToRuntimeInapp(t *testing.T) {
	env := e2e.NewEnv(t)
	env.EmitDataPoint("p1", "sensor.temp", 120)
	env.RunCompute("p1", "calc.high_temp")
	env.EvalAlarm("p1", "rule.high_temp")
	msg := env.PullRuntimeInapp("p1")
	if msg == nil { t.Fatal("expected runtime_inapp message") }
}
```

- [ ] **Step 2: 运行确认失败**

Run: `go test ./tests/e2e -run TestFullChain_DataToAlarmToRuntimeInapp -v`  
Expected: FAIL（在闭环未完整前）。

- [ ] **Step 3: 补齐缺口并更新验收清单**

```markdown
- [x] Connections/Queries/Datapoints 语义兼容
- [x] MQTT + wave1 + wave2 协议联调样例
- [x] Preview 30 分钟滑动过期
- [x] Compute JS/Python 超时与资源限制
- [x] Alarm 状态机 + CEL + runtime_inapp 幂等
- [x] 发布契约导出与 Node 契约校验
```

- [ ] **Step 4: 执行最终回归**

Run: `go test ./...`  
Expected: PASS

Run: `pnpm --dir datacenter build`  
Expected: BUILD SUCCESS

- [ ] **Step 5: 提交**

```bash
git add data_service/tests docs/datacenter
git commit -m "test(product): 完成 data_service 全链路回归与验收文档"
```

---

## 技能执行闸门（必须打勾）

- [ ] 每次落库变更前执行 `design-postgres-tables` 审核（字段类型、约束、索引、查询路径映射）。
- [ ] 报警域涉及空间能力时执行 `design-postgis-tables` 评估（本期默认仅文档声明）。
- [ ] 每个里程碑收尾执行 `supabase-postgres-best-practices` 复审并记录结果。

## 里程碑出口标准（统一）

- [ ] 代码与测试通过。
- [ ] datacenter 最小联调通过。
- [ ] 接口文档更新完成。
- [ ] 风险清单更新完成。
- [ ] SQL/索引复审记录归档。

## Spec Self-Review（本计划自检结果）

1. Spec coverage：已映射 M1-M11 全部能力点，包含协议两波、preview、compute、alarm、runtime_inapp、发布契约。
2. Placeholder scan：已移除 TBD/TODO/“后续再补”类语句。
3. Type consistency：`ApiResponse`、`AppError`、`ErrorCodes`、JWT capability、CEL 与子进程沙盒命名保持一致。
