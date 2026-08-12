# dev_core 面向 dev_ide 的 Go 重写实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）跟踪进度。

**目标：** 新建 Go 模块 `dev_core`，实现当前 `dev_ide` 实际调用的全部控制面接口，并为每个接口提供 HTTP 接口级测试；Designer 低代码相关能力不迁移。

**架构：** 使用 Chi 组织 `/api/v1`，OpenAPI 与 `oapi-codegen` 维护 REST 契约，`pgx/v5 + sqlc` 访问 PostgreSQL。业务按认证、租户、用户、工程、运行权限、节点、部署发布和日志拆包；接口测试注入领域 Fake，Repository 使用 PostgreSQL 集成测试。

**技术栈：** Go 1.26、Chi、OpenAPI、oapi-codegen、pgx/v5、sqlc、go-redis/v9、minio-go/v7、Moby Go Client、Argon2id、golang-jwt/jwt/v5、log/slog、Testcontainers Go。

---

## 范围

### 保留并重写

- 认证、租户、平台用户。
- 工程、标签、分组、生命周期和导入导出。
- 工程运行用户、运行角色和权限绑定。
- 节点注册、审批、心跳、状态和节点管理。
- 应用版本、开发部署、版本部署、节点命令和回滚。
- 日志列表、详情、导出、清理、统计和最近活动。
- `dev_ide` 使用的 `ops:*` Socket.IO 运维事件。

### 删除

- `design_pages`、`design_project_settings`、`design_asset_folders`、`design_assets`。
- `dev_core/src/dsl/`、`/api/v1/design/**` 和 `/api/v1/pages/**`。
- 旧页面 Schema、页面锁、低代码素材库和旧页面发布逻辑。

### 接口覆盖标准

- 从 `dev_ide/src` 按 HTTP 方法和参数化路径形状提取 74 个唯一 REST 操作，全部进入 OpenAPI。
- 每个 REST 操作至少有一个成功 HTTP 接口测试。
- 认证、权限、参数和不存在资源具有代表性失败测试。
- Node Agent 使用的注册、心跳和部署状态接口也实现并测试。
- 建立覆盖守卫：`dev_ide` 新增 Axios 调用但未增加 OpenAPI 和 HTTP 测试时，Go 测试必须失败。

---

## 目标目录

```text
dev_core/
├─ cmd/dev_core/main.go
├─ api/openapi.yaml
├─ db/schema/core-schema.sql
├─ db/queries/*.sql
├─ internal/app/
├─ internal/config/
├─ internal/platform/{api,db,cache}/
├─ internal/{auth,tenant,user,project,runtimeaccess,node,deployment,auditlog}/
├─ internal/{objectstore,realtime,worker}/
├─ tests/contract/
├─ sqlc.yaml
├─ go.mod
└─ Dockerfile
```

---

### 任务 1：Go 服务基座

**文件：**
- 创建：`dev_core/go.mod`
- 创建：`dev_core/cmd/dev_core/main.go`
- 创建：`dev_core/internal/config/config.go`
- 创建：`dev_core/internal/app/app.go`
- 创建：`dev_core/internal/platform/api/{response,errors,middleware}.go`
- 测试：`dev_core/internal/app/app_test.go`
- 修改：`package.json`

- [ ] 先写 `/health`、请求 ID、统一响应和 panic recovery 的失败测试。
- [ ] 实现最小 Chi Server，固定 `/api/v1` 和 `:18101`。
- [ ] 根 `package.json` 增加 `go:test:core` 并纳入 `go:test`。
- [ ] 运行 `cd dev_core && go test ./internal/app`，预期 PASS。

统一响应：

```go
type Envelope struct {
    Code int `json:"code"`
    Msg string `json:"msg"`
    Data any `json:"data"`
    ReqID string `json:"reqId"`
}
```

### 任务 2：最终数据库基线与 sqlc

**文件：**
- 创建：`dev_core/db/schema/core-schema.sql`
- 创建：`dev_core/sqlc.yaml`
- 创建：`dev_core/internal/platform/db/{pool,bootstrap}.go`
- 测试：`dev_core/internal/platform/db/bootstrap_test.go`
- 生成：`dev_core/internal/platform/db/sqlc/*.go`

- [ ] 建立 snake_case 最终表：租户、便签、用户、刷新令牌、工程、标签、分组、运行用户、运行角色、授权、审计日志、应用版本、节点、节点部署和节点命令。
- [ ] 编写空库初始化、非空库禁止修改、旧低代码表不存在测试。
- [ ] 固定查询使用 sqlc，动态分页搜索使用参数化 pgx。
- [ ] 运行 `cd dev_core && go test ./internal/platform/db/...`。

### 任务 3：OpenAPI、代码生成与覆盖守卫

**文件：**
- 创建：`dev_core/api/openapi.yaml`
- 创建：`dev_core/internal/platform/api/generate.go`
- 生成：`dev_core/internal/platform/api/generated.go`
- 创建：`dev_core/tests/contract/{endpoint_inventory,dev_ide_endpoints_test}.go`

- [ ] 录入全部 74 个 dev_ide REST operationId。
- [ ] 使用固定版本 `oapi-codegen` 生成 Chi Server 接口和类型。
- [ ] 扫描 `dev_ide/src` Axios 调用并与 OpenAPI 比较。
- [ ] 维护接口测试登记表并验证每个 operationId 有测试。
- [ ] 运行 `cd dev_core && go test ./tests/contract`。

### 任务 4：认证

**文件：**
- 创建：`dev_core/db/queries/auth.sql`
- 创建：`dev_core/internal/auth/{handler,service,repository,password,token}.go`
- 测试：`dev_core/internal/auth/handler_test.go`

- [ ] 测试并实现验证码、登录、刷新、登出、当前用户、修改密码和登录配置。
- [ ] 使用 Argon2id PHC 密码哈希。
- [ ] Access Token 使用 JWT v5/HS256；Refresh Token 使用随机值、数据库哈希和轮换事务。
- [ ] 运行 `cd dev_core && go test ./internal/auth ./tests/contract -run 'Auth|auth'`。

### 任务 5：租户和用户

**文件：**
- 创建：`dev_core/db/queries/{tenant,user}.sql`
- 创建：`dev_core/internal/tenant/{handler,service,repository}.go`
- 创建：`dev_core/internal/user/{handler,service,repository}.go`
- 测试：`dev_core/internal/{tenant,user}/handler_test.go`

- [ ] 测试并实现租户列表、详情、当前租户、便签 CRUD、创建、更新、删除、启用、停用和上传。
- [ ] 测试并实现用户列表、创建、更新、修改密码和删除。
- [ ] 列表分页、搜索和排序全部在 PostgreSQL 完成。
- [ ] 上传流式写入 SeaweedFS，不把完整文件读入内存。

### 任务 6：工程、标签、分组和运行权限

**文件：**
- 创建：`dev_core/db/queries/{project,runtime_access}.sql`
- 创建：`dev_core/internal/project/{handler,service,repository,archive}.go`
- 创建：`dev_core/internal/runtimeaccess/{handler,service,repository}.go`
- 测试：`dev_core/internal/{project,runtimeaccess}/handler_test.go`

- [ ] 测试并实现工程列表、标签 CRUD、分组 CRUD、工程标签、工程分组、创建、更新、删除、删除影响、生命周期操作和导入导出。
- [ ] 创建工程时在同一事务建立默认运行管理员、默认角色和授权。
- [ ] 测试并实现运行用户与运行角色的 10 个接口。
- [ ] 导入导出只包含控制面数据和工作空间，不包含旧页面 Schema。

### 任务 7：节点、发布和部署

**文件：**
- 创建：`dev_core/db/queries/{node,deployment}.sql`
- 创建：`dev_core/internal/node/{handler,service,repository}.go`
- 创建：`dev_core/internal/deployment/{handler,service,repository,artifact}.go`
- 测试：`dev_core/internal/{node,deployment}/handler_test.go`

- [ ] 测试并实现 dev_ide 节点列表、审批、拒绝和删除。
- [ ] 测试并实现 Node Agent 注册、审批查询、心跳、离线和部署状态回报。
- [ ] 测试并实现版本创建/列表/删除、开发部署、版本部署、节点启动/停止/重启/删除、回滚、工程节点和节点历史。
- [ ] 初版版本从工程工作空间生成排除开发缓存的 SHA-256 压缩包并上传 SeaweedFS。

### 任务 8：日志、运维事件和基础设施

**文件：**
- 创建：`dev_core/db/queries/log.sql`
- 创建：`dev_core/internal/auditlog/{handler,service,repository}.go`
- 创建：`dev_core/internal/realtime/socket.go`
- 创建：`dev_core/internal/objectstore/{store,minio}.go`
- 创建：`dev_core/internal/platform/cache/redis.go`
- 创建：`dev_core/internal/worker/worker.go`
- 测试：对应 `_test.go`

- [ ] 测试并实现日志列表、详情、CSV 导出、清理、统计和最近活动。
- [ ] 审计日志禁止记录密码、Token、节点密钥和文件内容。
- [ ] 仅保留 `ops:subscribe`、节点指标/状态、工程指标、部署状态和待审批事件。
- [ ] Redis 只保存验证码、Token 吊销和在线 TTL；持久任务状态落 PostgreSQL。
- [ ] 后台 Worker 支持优雅停止和重启恢复。

### 任务 9：容器化与最终验证

**文件：**
- 创建：`dev_core/Dockerfile`
- 修改：`.env.development.example`
- 修改：`.env.production.example`
- 修改：`docs/03-模块设计/dev_core/README.md`
- 修改：`docs/04-契约与规范/统一REST接口规范与清单.md`

- [ ] 使用固定 Go 1.26 构建镜像，运行镜像只包含二进制、OpenAPI 和数据库基线。
- [ ] 不切换或删除旧 `dev_core`，直到全部接口和 dev_ide 验证通过。
- [ ] 运行 `gofmt -w dev_core`、`go test ./...`、`go vet ./...`。
- [ ] 运行 `pnpm --dir dev_ide test`、`pnpm --dir dev_ide typecheck`、`pnpm --dir dev_ide build`。
- [ ] 运行根 `pnpm go:test` 和 `pnpm typecheck`。

## 完成审计

1. 每个 `dev_ide` Axios REST 调用都有 OpenAPI operation 和成功 HTTP 测试。
2. `dev_ide` 使用的全部 `ops:*` 事件可连接并接收。
3. 最终 SQL 不包含旧低代码表。
4. `dev_core` 不引用旧 Node.js 源码。
5. Go 测试、vet、dev_ide 测试、类型检查和构建全部通过。
6. 在证据完整前不删除或替换旧 `dev_core`。
