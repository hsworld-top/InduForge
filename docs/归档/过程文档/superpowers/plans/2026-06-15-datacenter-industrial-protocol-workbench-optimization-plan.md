# 数据中心工业协议工作台优化实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 落地工业协议工作台优化的第一条完整主干：存储策略独立菜单、统一数据点归档策略 API、协议工作台存储摘要与冗余/契约摘要入口，并保持后端与前端可编译。

**架构：** 后端在 `data_service` 新增存储策略领域模型、迁移、Repository、Service、Handler 和 Router，策略面向统一数据点并通过目标接入源能力筛选。前端在 `datacenter` 新增“存储策略”一级模块，提供策略列表、编辑抽屉、绑定规则、预估摘要和存储目标能力展示，同时在数据中心模块导航中接入。协议工作台本轮不做长期监控和写值，只补“存储策略摘要/运行契约提示/冗余摘要”的可视化骨架。

**技术栈：** Go、PostgreSQL 兼容 SQL、Vue 3、TypeScript、Element Plus、Pinia/Vite、pnpm workspace。

---

## 文件结构

- 创建：`data_service/internal/db/migrations/0052_data_storage_policies.sql`，新增存储策略表、绑定表和索引。
- 创建：`data_service/internal/db/migrations/0052_data_storage_policies_down.sql`，回滚存储策略表。
- 创建：`data_service/internal/repository/storage_policy_repository.go`，封装存储策略、绑定、目标能力查询和预估所需数据访问。
- 创建：`data_service/internal/service/storage_policy_service.go`，封装策略校验、能力筛选、静态/动态绑定、预估和目标摘要。
- 创建：`data_service/internal/http/handler/storage_policy_handler.go`，提供 `/api/v1/data-storage-policies` REST API。
- 修改：`data_service/internal/http/router/router.go`，挂载存储策略路由。
- 修改：`data_service/internal/app/server.go`，组装存储策略 Repository、Service、Handler。
- 创建：`datacenter/src/api/storage-policy.api.ts`，封装前端存储策略 API 和类型。
- 创建：`datacenter/src/views/storage-policy/StoragePolicyWorkspace.vue`，新增存储策略工作台页面。
- 修改：`datacenter/src/router/route-config.ts`，注册 `storage-policy` 模块路由。
- 修改：`datacenter/src/config/datacenterModules.ts`，新增数据中心菜单项。
- 修改：`datacenter/src/views/DataCenterNew.vue`，允许切换到 `storage-policy` 模块。
- 修改：`datacenter/tests/router-config.test.ts`、`datacenter/tests/datacenter-modules.test.ts`，覆盖新增模块。

## 任务 1：后端数据结构与领域 API

**文件：**

- 创建：`data_service/internal/db/migrations/0052_data_storage_policies.sql`
- 创建：`data_service/internal/db/migrations/0052_data_storage_policies_down.sql`
- 创建：`data_service/internal/repository/storage_policy_repository.go`
- 创建：`data_service/internal/service/storage_policy_service.go`

- [ ] **步骤 1：新增数据库迁移**

```sql
CREATE TABLE IF NOT EXISTS data_storage_policies (...);
CREATE TABLE IF NOT EXISTS data_storage_policy_bindings (...);
```

验证：`cd data_service && go test ./tests/integration -run TestMigrations -count=1` 应通过。

- [ ] **步骤 2：新增 Repository**

```go
type StoragePolicyRepository struct { db *sql.DB }
func (r *StoragePolicyRepository) ListPolicies(ctx context.Context, projectID string, q StoragePolicyListQuery) ([]StoragePolicy, int64, error)
```

验证：`gofmt -w data_service/internal/repository/storage_policy_repository.go` 无格式错误。

- [ ] **步骤 3：新增 Service**

```go
type StoragePolicyService struct { repo *repository.StoragePolicyRepository }
func (s *StoragePolicyService) CreatePolicy(ctx context.Context, input StoragePolicyInput) (*StoragePolicyDetail, error)
```

验证：`cd data_service && go test ./internal/service -run StoragePolicy -count=1` 可运行；若暂无单测则至少 `go test ./internal/service -run TestNonExisting -count=1` 编译通过。

## 任务 2：后端 HTTP 接口与组装

**文件：**

- 创建：`data_service/internal/http/handler/storage_policy_handler.go`
- 修改：`data_service/internal/http/router/router.go`
- 修改：`data_service/internal/app/server.go`

- [ ] **步骤 1：新增 Handler**

```go
func (h *StoragePolicyHandler) ListPolicies(c *gin.Context)
func (h *StoragePolicyHandler) CreatePolicy(c *gin.Context)
func (h *StoragePolicyHandler) EstimatePolicy(c *gin.Context)
```

验证：`gofmt -w data_service/internal/http/handler/storage_policy_handler.go` 无格式错误。

- [ ] **步骤 2：挂载路由**

```go
api.GET("/data-storage-policies", storagePolicyHandler.ListPolicies)
api.POST("/data-storage-policies", storagePolicyHandler.CreatePolicy)
```

验证：`cd data_service && go test ./...` 后端全量测试通过或明确记录外部依赖导致的失败。

## 任务 3：前端存储策略模块

**文件：**

- 创建：`datacenter/src/api/storage-policy.api.ts`
- 创建：`datacenter/src/views/storage-policy/StoragePolicyWorkspace.vue`
- 修改：`datacenter/src/router/route-config.ts`
- 修改：`datacenter/src/config/datacenterModules.ts`
- 修改：`datacenter/src/views/DataCenterNew.vue`

- [ ] **步骤 1：新增 API 类型与请求封装**

```ts
export async function listStoragePolicies(
  params: StoragePolicyListParams,
): Promise<StoragePolicyListResponse>
```

验证：`pnpm --filter datacenter typecheck` 不出现新增类型错误。

- [ ] **步骤 2：新增工作台页面**

页面包含策略列表、目标能力摘要、绑定模式、写入模式、保留策略、预估信息、编辑抽屉和目标连接筛选。界面保持运维工具的密度和克制风格。

验证：`pnpm --filter datacenter typecheck` 与 `pnpm --filter datacenter build` 通过。

- [ ] **步骤 3：接入数据中心导航**

```ts
{ key: 'storage-policy', label: '存储策略', routeName: 'data-storage-policy' }
```

验证：`pnpm --filter datacenter test -- router-config datacenter-modules` 通过。

## 任务 4：协议工作台摘要联动

**文件：**

- 修改：优先复用现有 OPC UA、Modbus、S7 工作台 Inspector 或共享组件文件。

- [ ] **步骤 1：定位现有协议工作台摘要组件**

使用 `rg "存储|storage|Inspector|contract|redundancy" datacenter/src/views datacenter/src/components` 找到最小改动点。

- [ ] **步骤 2：补充只读摘要**

摘要只展示“实时当前值默认 IF 实时库”“历史归档命中策略数量/未配置”“设备冗余摘要”“采集冗余摘要”，并提供跳转到存储策略模块，不提供写值、报警、计算快捷入口。

验证：`pnpm --filter datacenter typecheck` 通过。

## 任务 5：最终验证、提交与推送

**文件：**

- 修改：只包含本计划涉及文件。

- [ ] **步骤 1：执行格式化**

运行：`gofmt -w data_service/internal/repository/storage_policy_repository.go data_service/internal/service/storage_policy_service.go data_service/internal/http/handler/storage_policy_handler.go data_service/internal/http/router/router.go data_service/internal/app/server.go`

- [ ] **步骤 2：执行验证**

运行：

```bash
pnpm --filter datacenter typecheck
pnpm --filter datacenter build
pnpm --filter datacenter test -- router-config datacenter-modules
cd data_service && go test ./...
```

- [ ] **步骤 3：提交并推送**

运行：

```bash
git add docs/superpowers/plans/2026-06-15-datacenter-industrial-protocol-workbench-optimization-plan.md data_service datacenter
git commit -m "feat(datacenter): 落地工业协议存储策略工作台"
git push origin master
```
