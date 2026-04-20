# data_service 后续实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在已完成 `dev_core` 数据域下线和 `data_service` 控制面迁移的基础上，补齐剩余运行态与开发态缺口，使 `data_service` 从“管理面可用”推进到“开发态链路完整可用”。

**Architecture:** 当前 `data_service` 已完成 PostgreSQL 存储、控制面 API、项目快照、查询执行、数据点管理、MQTT 管理和 preview session 基础能力，但 MQTT 实时值、协议运行态、preview 读/订阅链路仍存在降级实现。后续计划按“先补主链路、再补实时链路、最后做性能和治理”的顺序推进，避免一边扩协议一边留下核心链路未闭环。

**Tech Stack:** Go、pgx/PostgreSQL、Redis、JWT、HTTP API、测试容器集成测试

---

## 当前完成度基线

### 已完成

1. `data_service` 已完成 PostgreSQL 迁移与基础服务装配。
2. 连接、查询、数据点、MQTT 管理、项目快照、preview session、compute、协议配置入库接口均已落地。
3. `dev_core` 已切换到调用 `data_service` 承接数据域主链路，且本地数据域实现已下线。
4. 当前验证已通过：
   - `go test ./...`
   - `go build ./...`

### 当前缺口

1. MQTT 连接测试、启停、状态仍主要是配置校验和状态字段流转，不是真正 broker runtime。
2. MQTT Tag 当前值接口仍返回 `defaultValue` 降级结果，不是真实订阅值。
3. preview 仅完成 session create/heartbeat/delete，未形成完整 read/subscribe/unsubscribe 开发态链路。
4. 数据点 `usages` 仍为空实现，写值仍是“更新默认值”兼容方案。
5. protocol wave1/wave2 当前主要是配置入库，不是完整运行态接入。
6. 查询执行对外部库采用“每次执行临时建池”，功能可用但性能保守。

## 文件边界

### 现有核心入口

- `D:\SVNCode\indu-forge\data_service\internal\app\server.go`
- `D:\SVNCode\indu-forge\data_service\internal\http\router\router.go`
- `D:\SVNCode\indu-forge\data_service\internal\service\connection_service.go`
- `D:\SVNCode\indu-forge\data_service\internal\service\query_service.go`
- `D:\SVNCode\indu-forge\data_service\internal\service\datapoint_service.go`
- `D:\SVNCode\indu-forge\data_service\internal\service\datapoint_management.go`
- `D:\SVNCode\indu-forge\data_service\internal\service\mqtt_service.go`
- `D:\SVNCode\indu-forge\data_service\internal\service\mqtt_management.go`
- `D:\SVNCode\indu-forge\data_service\internal\service\preview_session_service.go`
- `D:\SVNCode\indu-forge\data_service\internal\service\protocol_wave1_service.go`
- `D:\SVNCode\indu-forge\data_service\internal\service\protocol_wave2_service.go`
- `D:\SVNCode\indu-forge\data_service\internal\repository\*.go`
- `D:\SVNCode\indu-forge\data_service\internal\db\migrations\*.sql`
- `D:\SVNCode\indu-forge\data_service\tests\integration\*.go`

### 建议新增文件

- `D:\SVNCode\indu-forge\data_service\internal\service\mqtt_runtime.go`
  负责 MQTT 真实连接、订阅、消息缓存与状态维护，避免继续把运行态逻辑挤在 `mqtt_management.go` 里。
- `D:\SVNCode\indu-forge\data_service\internal\service\preview_runtime.go`
  负责 preview 会话内 read/subscribe/unsubscribe 的运行态协调。
- `D:\SVNCode\indu-forge\data_service\internal\service\external_pool_cache.go`
  负责外部关系库连接池缓存，解决 query/connection 每次临时建池问题。

## 优先级总览

1. **P0：补齐 MQTT 真实运行态，打通 Tag 实时值。**
2. **P0：补齐 preview 读/订阅链路，恢复设计器和数据中心开发态完整体验。**
3. **P1：把数据点写值、usages 从兼容实现提升到可用实现。**
4. **P1：把 query/connection 对外部库访问改成池缓存。**
5. **P2：把 protocol wave1/wave2 从“配置面”推进到“最小运行态”。**

### Task 1: MQTT 运行态闭环

**Files:**
- Create: `D:\SVNCode\indu-forge\data_service\internal\service\mqtt_runtime.go`
- Modify: `D:\SVNCode\indu-forge\data_service\internal\service\mqtt_service.go`
- Modify: `D:\SVNCode\indu-forge\data_service\internal\service\mqtt_management.go`
- Modify: `D:\SVNCode\indu-forge\data_service\internal\app\server.go`
- Test: `D:\SVNCode\indu-forge\data_service\tests\integration\mqtt_api_test.go`

- [ ] **Step 1: 先补 MQTT 真实运行态的失败用例**

目标：补一个集成测试，覆盖“测试连接真实失败/成功、启动后状态反映 runtime、Tag 值来自实际消息而非 defaultValue”。

- [ ] **Step 2: 跑集成测试，确认现状失败点**

Run: `go test ./tests/integration -run TestMqtt -count=1`

Expected:
- 连接测试仍只是校验参数，不能反映真实 broker 可达性
- Tag 值接口仍返回默认值

- [ ] **Step 3: 新增 MQTT runtime 管理组件**

实现要点：
- 为连接建立内存级 runtime registry
- 启动连接时真实建立 broker 连接
- 订阅启用项时接收消息并缓存最近值
- 停止连接时清理 runtime 资源

- [ ] **Step 4: 把 MQTT 管理接口接到 runtime**

实现要点：
- `TestConnectionConfig` 改为真实连接测试
- `StartConnection`/`StopConnection`/`GetConnectionStatus` 改为读取 runtime 状态
- `GetTagValue`/`GetTagValues` 优先返回最新订阅值，降级时才回退默认值

- [ ] **Step 5: 再跑 MQTT 集成测试**

Run: `go test ./tests/integration -run TestMqtt -count=1`

Expected: PASS

- [ ] **Step 6: 提交**

```bash
git add data_service/internal/service data_service/internal/app data_service/tests/integration
git commit -m "feat(data_service): 补齐mqtt运行态链路"
```

### Task 2: Preview 开发态链路补齐

**Files:**
- Create: `D:\SVNCode\indu-forge\data_service\internal\service\preview_runtime.go`
- Modify: `D:\SVNCode\indu-forge\data_service\internal\http\router\router.go`
- Modify: `D:\SVNCode\indu-forge\data_service\internal\http\handler\preview_handler.go`
- Modify: `D:\SVNCode\indu-forge\data_service\internal\service\preview_session_service.go`
- Test: `D:\SVNCode\indu-forge\data_service\tests\integration\preview_session_test.go`

- [ ] **Step 1: 先补 preview read/subscribe/unsubscribe 的失败用例**

目标：覆盖会话内读取数据点、订阅 MQTT/数据点、取消订阅、过期或关闭后拒绝操作。

- [ ] **Step 2: 跑 preview 集成测试，确认缺口**

Run: `go test ./tests/integration -run TestPreview -count=1`

Expected:
- 当前只有 create/heartbeat/delete
- 缺少 read/subscribe/unsubscribe 路由和处理逻辑

- [ ] **Step 3: 新增 preview runtime 协调层**

实现要点：
- 按 session 维护读/订阅上下文
- 对 MQTT 与数据点读取做统一入口
- 会话关闭/过期时清理订阅资源

- [ ] **Step 4: 补齐 preview handler 和 router**

新增接口：
- `POST /api/v1/data/preview/sessions/{sessionId}/read`
- `POST /api/v1/data/preview/sessions/{sessionId}/subscribe`
- `POST /api/v1/data/preview/sessions/{sessionId}/unsubscribe`

- [ ] **Step 5: 再跑 preview 集成测试**

Run: `go test ./tests/integration -run TestPreview -count=1`

Expected: PASS

- [ ] **Step 6: 提交**

```bash
git add data_service/internal/http data_service/internal/service data_service/tests/integration
git commit -m "feat(data_service): 补齐preview开发态链路"
```

### Task 3: 数据点兼容实现转正

**Files:**
- Modify: `D:\SVNCode\indu-forge\data_service\internal\service\datapoint_management.go`
- Modify: `D:\SVNCode\indu-forge\data_service\internal\repository\datapoint_repository.go`
- Modify: `D:\SVNCode\indu-forge\data_service\internal\repository\query_repository.go`
- Test: `D:\SVNCode\indu-forge\data_service\tests\integration\query_datapoint_api_test.go`

- [ ] **Step 1: 先补 usages 与 writeValue 的失败用例**

目标：
- `usages` 至少返回 query/source/project snapshot 等直接引用
- `writeValue` 对可写类型做明确边界，不再隐式等于“改默认值”

- [ ] **Step 2: 跑数据点集成测试，确认失败**

Run: `go test ./tests/integration -run TestQueryAndDataPoint -count=1`

Expected:
- `usages` 仍为空数组
- `writeValue` 语义与真实写入不一致

- [ ] **Step 3: 实现数据点 usages**

实现要点：
- 先统计 `db.query`、`mqtt.tag`、`mqtt.subscription`、snapshot 直接引用
- 返回最小可用结构，不追求一次性覆盖所有隐式引用

- [ ] **Step 4: 收紧 writeValue 能力边界**

实现要点：
- 对 `manual`/明确可写来源允许写入缓存或默认值
- 对不可写来源返回明确错误，不再静默兼容

- [ ] **Step 5: 再跑数据点测试**

Run: `go test ./tests/integration -run TestQueryAndDataPoint -count=1`

Expected: PASS

- [ ] **Step 6: 提交**

```bash
git add data_service/internal/service data_service/internal/repository data_service/tests/integration
git commit -m "feat(data_service): 补齐数据点使用关系与写值边界"
```

### Task 4: 外部库连接池缓存

**Files:**
- Create: `D:\SVNCode\indu-forge\data_service\internal\service\external_pool_cache.go`
- Modify: `D:\SVNCode\indu-forge\data_service\internal\service\query_service.go`
- Modify: `D:\SVNCode\indu-forge\data_service\internal\service\connection_service.go`
- Modify: `D:\SVNCode\indu-forge\data_service\internal\app\server.go`
- Test: `D:\SVNCode\indu-forge\data_service\tests\integration\connection_api_test.go`
- Test: `D:\SVNCode\indu-forge\data_service\tests\integration\query_datapoint_api_test.go`

- [ ] **Step 1: 先补性能路径回归测试或统计断言**

目标：验证重复执行 query/list tables 时复用同类外部连接，而不是每次新建。

- [ ] **Step 2: 跑连接/查询测试确认现状**

Run:
- `go test ./tests/integration -run TestConnection -count=1`
- `go test ./tests/integration -run TestQueryAndDataPoint -count=1`

- [ ] **Step 3: 新增连接池缓存组件**

实现要点：
- 以 `databaseUrl + searchPath` 作为缓存键
- 控制 TTL 和失效
- 服务关闭时统一释放

- [ ] **Step 4: 接入 connection/query 服务**

实现要点：
- `TestConnection`、`ListTables`、`GetTableStructure`、`GetTableData`、`ExecuteSQL`
- `executeRecord`

- [ ] **Step 5: 回归测试**

Run: `go test ./...`

Expected: PASS

- [ ] **Step 6: 提交**

```bash
git add data_service/internal/app data_service/internal/service data_service/tests/integration
git commit -m "perf(data_service): 复用外部关系库连接池"
```

### Task 5: 协议运行态最小闭环

**Files:**
- Modify: `D:\SVNCode\indu-forge\data_service\internal\service\protocol_wave1_service.go`
- Modify: `D:\SVNCode\indu-forge\data_service\internal\service\protocol_wave2_service.go`
- Modify: `D:\SVNCode\indu-forge\data_service\internal\http\handler\protocol_wave1_handler.go`
- Modify: `D:\SVNCode\indu-forge\data_service\internal\http\handler\protocol_wave2_handler.go`
- Test: `D:\SVNCode\indu-forge\data_service\tests\integration\protocol_wave1_test.go`
- Test: `D:\SVNCode\indu-forge\data_service\tests\integration\protocol_wave2_test.go`

- [ ] **Step 1: 补协议域失败用例**

目标：区分“仅配置入库”与“最小可验证运行态”，至少补：
- Kafka topic preview 真实边界
- OPCDA contract 校验不再恒等返回 valid
- 其他协议保留最小探活/参数校验边界

- [ ] **Step 2: 跑协议集成测试确认失败点**

Run:
- `go test ./tests/integration -run TestProtocolWave1 -count=1`
- `go test ./tests/integration -run TestProtocolWave2 -count=1`

- [ ] **Step 3: 先补最小运行态或明确错误语义**

实现要点：
- 能真实探活的先做探活
- 不能真实接入的，返回明确“不支持/未启用适配器”，不要继续伪装成功

- [ ] **Step 4: 再跑协议测试**

Run:
- `go test ./tests/integration -run TestProtocolWave1 -count=1`
- `go test ./tests/integration -run TestProtocolWave2 -count=1`

Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add data_service/internal/service data_service/internal/http/handler data_service/tests/integration
git commit -m "feat(data_service): 收紧协议配置与运行态边界"
```

## 收尾验证

- [ ] `go test ./...`
- [ ] `go build ./...`
- [ ] 联调验证 `dev_core` -> `data_service` 的 snapshot、query、datapoint、mqtt 主链路

## 结果判断标准

1. `data_service` 不再只提供控制面 CRUD，而具备最小可用的开发态运行链路。
2. MQTT 不再依赖 `defaultValue` 伪装实时值。
3. preview 不再只有 session 外壳，而能承接 read/subscribe/unsubscribe。
4. 数据点写值、usages、查询执行性能具备明确边界和可验证行为。
5. protocol wave1/wave2 至少不再“假成功”。

