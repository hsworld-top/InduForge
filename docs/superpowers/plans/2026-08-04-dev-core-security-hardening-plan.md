# dev_core 安全与交付修复实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 `executing-plans` 在当前会话逐任务实现。每个任务先补失败测试，再做最小实现并运行聚焦验证。根据仓库规则，本计划不自动创建 Git 提交。

**目标：** 修复 `dev_core` 当前权限、工程分享、工作空间、对象存储、认证、上传、实时通信和接口契约问题，使 Go 控制面具备可继续开发 Designer code-first 产品的安全基础。

**架构：** 在 `auth` 包集中维护固定角色能力矩阵，Service 层叠加工程可见性和所有权校验；SeaweedFS 保持私有，数据库只保存对象键，浏览器和节点使用短期签名 URL；工作空间导出拒绝符号链接和特殊文件并限制总量。控制面实时通信使用独立 Socket.IO path，OpenAPI 改为领域化请求响应 Schema。

**技术栈：** Go 1.26、Chi、pgx、sqlc、PostgreSQL、MinIO Go SDK、SeaweedFS S3、Socket.IO、Vue 3、TypeScript、Vitest、oapi-codegen。

---

## 文件职责

- `dev_core/internal/auth/authorization.go`：固定角色能力矩阵和统一授权函数。
- `dev_core/internal/auth/authorization_test.go`：角色能力和角色管理层级测试。
- `dev_core/internal/project/access.go`：工程可见性、所有权和能力组合校验。
- `dev_core/internal/project/workspace.go`：安全导出、目录排除和大小限制。
- `dev_core/internal/objectstore/minio.go`：对象引用、上传、签名下载和删除。
- `dev_core/internal/tenant/*`：租户图片对象键持久化和签名 URL 输出。
- `dev_core/internal/deployment/*`：发布清单对象引用和节点命令签名 URL。
- `dev_core/internal/auth/*`：验证码必填和 Refresh Token 单次轮换。
- `dev_core/internal/platform/api/*`、`dev_core/api/openapi.yaml`：真实接口契约。
- `dev_ide/src/utils/socket.ts`、`dev_ide/vite.config.ts`、`scripts/docker/edge/nginx.conf`：控制面实时路径。
- Designer code-server 规格与计划：纠正为开发态端口直连方案。

### 任务 1：建立集中式角色能力矩阵

**文件：**
- 创建：`dev_core/internal/auth/authorization.go`
- 创建：`dev_core/internal/auth/authorization_test.go`
- 修改：`dev_core/internal/platform/api/errors.go`

- [ ] **步骤 1：编写角色能力表驱动测试**

测试必须覆盖 `SUPER_ADMIN`、`SYSTEM_ADMIN`、`PROJECT_ADMIN`、`OPS_ADMIN`、`USER_ADMIN`、`DEVELOPER`、`OPERATOR`、`VIEWER`，并断言至少以下能力：

```go
func TestHasCapability(t *testing.T) {
    tests := []struct {
        role       string
        capability auth.Capability
        allowed    bool
    }{
        {"DEVELOPER", auth.CapabilityProjectWrite, true},
        {"DEVELOPER", auth.CapabilityDeploymentExecute, false},
        {"OPS_ADMIN", auth.CapabilityNodeApprove, true},
        {"USER_ADMIN", auth.CapabilityProjectRead, false},
        {"VIEWER", auth.CapabilityProjectRead, true},
        {"VIEWER", auth.CapabilityProjectWrite, false},
    }
    // 逐项断言。
}
```

- [ ] **步骤 2：运行测试验证失败**

运行：

```powershell
$env:GOSUMDB='sum.golang.org'
cd dev_core
go test ./internal/auth -run 'Capability|RoleAssignment'
```

预期：FAIL，缺少 `Capability` 和授权函数。

- [ ] **步骤 3：实现固定能力和角色层级**

提供以下接口：

```go
type Capability string

func HasCapability(role string, capability Capability) bool
func RequireCapability(user User, capability Capability) error
func CanAssignRole(actorRole, targetRole string) bool
func IsPlatformAdmin(role string) bool
```

`RequireCapability` 返回可被 Handler 映射为 `ErrorCodePermissionDenied` 的 `ErrPermissionDenied`。

- [ ] **步骤 4：运行聚焦测试**

运行：`go test ./internal/auth -run 'Capability|RoleAssignment'`

预期：PASS。

### 任务 2：实现私有与分享工程访问过滤

**文件：**
- 创建：`dev_core/internal/project/access.go`
- 创建：`dev_core/internal/project/access_test.go`
- 修改：`dev_core/db/schema/core-schema.sql`
- 修改：`dev_core/db/queries/project.sql`
- 修改：`dev_core/internal/project/repository.go`
- 修改：`dev_core/internal/project/service.go`
- 修改：`dev_core/internal/project/handler.go`
- 修改：`dev_core/internal/project/handler_test.go`
- 生成：`dev_core/internal/platform/db/sqlc/project.sql.go`

- [ ] **步骤 1：编写工程访问失败测试**

覆盖：

- 私有工程创建人可读写。
- 同租户其他 `DEVELOPER` 不能读取私有工程。
- `SYSTEM_ADMIN` 可读取私有工程。
- 分享工程 `VIEWER` 可读不可写。
- 分享工程 `DEVELOPER` 可写，但只有创建人或项目管理员可切换分享和删除。
- `USER_ADMIN` 看不到分享工程。

- [ ] **步骤 2：运行工程测试验证失败**

运行：`go test ./internal/project -run 'Access|Visibility|Permission'`

预期：FAIL，当前列表和 Service 没有访问过滤。

- [ ] **步骤 3：收紧数据库最终基线**

将工程可见性约束改为：

```sql
visibility text NOT NULL DEFAULT 'private'
  CHECK (visibility IN ('private', 'internal'))
```

`ListProjects` 和 `CountProjects` 增加参数：

```sql
AND (
  sqlc.arg(is_platform_admin)::boolean
  OR p.created_by = sqlc.arg(actor_id)
  OR (
    sqlc.arg(can_read_shared)::boolean
    AND p.visibility = 'internal'
  )
)
```

- [ ] **步骤 4：生成 sqlc 代码**

运行：

```powershell
cd dev_core
go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.30.0 generate
```

预期：生成代码成功且没有手工修改生成文件。

- [ ] **步骤 5：实现工程访问函数**

```go
func CanRead(actor auth.User, item Project) bool
func RequireCapability(actor auth.User, item Project, capability auth.Capability) error
func RequireOwnerOrAdmin(actor auth.User, item Project, capability auth.Capability) error
```

Service 在创建、更新、删除、生命周期、导入导出、标签和分组绑定前执行对应校验。工程列表直接传递访问参数到 PostgreSQL。

- [ ] **步骤 6：运行工程与契约测试**

运行：

```powershell
go test ./internal/project ./tests/contract
```

预期：PASS。

### 任务 3：给运行权限、节点、发布和部署补齐授权

**文件：**
- 修改：`dev_core/internal/runtimeaccess/service.go`
- 修改：`dev_core/internal/runtimeaccess/handler.go`
- 修改：`dev_core/internal/runtimeaccess/handler_test.go`
- 修改：`dev_core/internal/node/service.go`
- 修改：`dev_core/internal/node/handler.go`
- 修改：`dev_core/internal/node/handler_test.go`
- 修改：`dev_core/internal/deployment/service.go`
- 修改：`dev_core/internal/deployment/handler.go`
- 修改：`dev_core/internal/deployment/handler_test.go`
- 修改：`dev_core/internal/auditlog/handler.go`

- [ ] **步骤 1：编写各领域权限拒绝测试**

至少断言：

- `VIEWER` 不能创建运行角色、重置运行用户密码。
- `DEVELOPER` 不能审批或删除节点。
- `DEVELOPER` 可以发布版本但不能执行正式部署。
- `OPERATOR` 只能操作已有部署，不能创建部署。
- `OPS_ADMIN` 可以审批节点、创建部署和操作部署，不能修改工程。
- `USER_ADMIN` 不能读取节点、部署和审计日志。

- [ ] **步骤 2：运行测试验证失败**

运行：

```powershell
go test ./internal/runtimeaccess ./internal/node ./internal/deployment ./internal/auditlog -run 'Permission|Capability'
```

预期：FAIL。

- [ ] **步骤 3：在 Service 层执行能力校验**

- 运行用户与角色写操作使用 `runtime-access:manage`。
- 节点列表使用 `node:read`，审批使用 `node:approve`，删除使用 `node:delete`。
- 发布版本使用 `release:publish`。
- 创建或回滚部署使用 `deployment:execute`。
- 启动、停止、重启、移除使用 `deployment:operate`。
- 审计日志查询和清理分别使用 `audit-log:read`、`audit-log:delete`。

部署和运行权限 Service 必须读取工程并调用任务 2 的工程访问校验。

- [ ] **步骤 4：运行领域测试**

运行：

```powershell
go test ./internal/runtimeaccess ./internal/node ./internal/deployment ./internal/auditlog
```

预期：PASS。

### 任务 4：修复用户角色提权和凭据撤销

**文件：**
- 修改：`dev_core/internal/user/service.go`
- 修改：`dev_core/internal/user/handler.go`
- 修改：`dev_core/internal/user/handler_test.go`
- 修改：`dev_core/cmd/dev_core/main.go`

- [ ] **步骤 1：编写角色层级和自修改测试**

覆盖：

- `USER_ADMIN` 创建 `PROJECT_ADMIN`、`SYSTEM_ADMIN`、`SUPER_ADMIN` 均失败。
- `SYSTEM_ADMIN` 创建或更新 `SUPER_ADMIN` 失败。
- `SUPER_ADMIN` 可以赋予 `SUPER_ADMIN`。
- 任意用户修改自己的角色或状态失败。
- 修改密码后调用 Refresh Token 全量撤销接口。

- [ ] **步骤 2：运行测试验证失败**

运行：`go test ./internal/user -run 'Role|Self|Password'`

预期：FAIL。

- [ ] **步骤 3：实现 Service 校验和认证协作接口**

`user.Service` 增加 Token 撤销依赖：

```go
type TokenRevoker interface {
    RevokeUserRefreshTokens(context.Context, string) error
}

func NewService(repository Repository, revoker TokenRevoker) *Service
```

更新角色前调用 `auth.CanAssignRole`，更新自身时拒绝 `Role` 和 `Status` 字段。密码修改成功后撤销用户全部 Refresh Token。

- [ ] **步骤 4：运行用户测试**

运行：`go test ./internal/user`

预期：PASS。

### 任务 5：加固工作空间导出

**文件：**
- 修改：`dev_core/internal/project/workspace.go`
- 创建：`dev_core/internal/project/workspace_test.go`
- 修改：`dev_core/internal/deployment/handler_test.go`

- [ ] **步骤 1：编写符号链接、特殊文件和大小限制测试**

测试常量：

```go
const maxWorkspaceFileSize int64 = 32 << 20
const maxWorkspaceTotalSize int64 = 512 << 20
```

测试必须包含：

- 普通文件可导出。
- 文件符号链接被拒绝且目标内容不出现在结果中。
- 符号链接目录被拒绝。
- 超过 32MB 的单文件被拒绝。
- 累计超过 512MB 时被拒绝。
- `node_modules`、`dist`、`.git`、`.vite`、`code-server-data`、`code-server-config`、`.induforge/context` 被排除。

Windows 无法创建符号链接时测试使用 `t.Skip`，Linux CI 必须执行。

- [ ] **步骤 2：运行测试验证失败**

运行：`go test ./internal/project -run Workspace`

预期：FAIL。

- [ ] **步骤 3：实现安全遍历**

遍历时先检查：

```go
if entry.Type()&os.ModeSymlink != 0 {
    return fmt.Errorf("工作空间包含不允许的符号链接: %s", relative)
}
info, err := entry.Info()
if err != nil || !info.Mode().IsRegular() {
    return fmt.Errorf("工作空间包含非普通文件: %s", relative)
}
```

读取前累计 `info.Size()`，超限立即停止。核心逻辑添加中文注释说明安全原因。

- [ ] **步骤 4：运行工作空间和发布测试**

运行：`go test ./internal/project ./internal/deployment`

预期：PASS。

### 任务 6：重构对象存储为对象引用和签名 URL

**文件：**
- 修改：`dev_core/internal/objectstore/minio.go`
- 创建：`dev_core/internal/objectstore/minio_test.go`
- 修改：`dev_core/internal/config/config.go`
- 修改：`.env.development.example`
- 修改：`.env.production.example`

- [ ] **步骤 1：编写对象存储接口测试**

定义：

```go
type ObjectRef struct {
    Bucket     string
    Key        string
    Size       int64
    ContentType string
}

func (store *MinIO) Put(...) (ObjectRef, error)
func (store *MinIO) PresignGet(ctx context.Context, key string, ttl time.Duration) (string, error)
func (store *MinIO) Delete(ctx context.Context, key string) error
```

测试签名 URL 包含签名参数且不暴露 Access Key、Secret Key 原文。

- [ ] **步骤 2：运行测试验证失败**

运行：`go test ./internal/objectstore`

预期：FAIL。

- [ ] **步骤 3：实现对象引用和签名下载**

使用 MinIO SDK `PresignedGetObject`；删除 `publicURL` 拼接逻辑和 `IF_OBJECT_STORE_PUBLIC_URL` 配置。默认租户图片签名 TTL 1 小时，部署工件 TTL 30 分钟。

- [ ] **步骤 4：运行对象存储测试**

运行：`go test ./internal/objectstore`

预期：PASS。

### 任务 7：租户图片改存对象键并严格限制上传

**文件：**
- 修改：`dev_core/db/schema/core-schema.sql`
- 修改：`dev_core/db/queries/tenant_user.sql`
- 修改：`dev_core/internal/tenant/service.go`
- 修改：`dev_core/internal/tenant/repository.go`
- 修改：`dev_core/internal/tenant/handler.go`
- 修改：`dev_core/internal/tenant/handler_test.go`
- 修改：`dev_core/internal/auth/models.go`
- 修改：`dev_core/internal/auth/repository.go`
- 修改：`dev_core/internal/auth/service.go`
- 修改：`dev_core/internal/auth/handler_test.go`
- 生成：`dev_core/internal/platform/db/sqlc/tenant_user.sql.go`
- 生成：`dev_core/internal/platform/db/sqlc/auth.sql.go`

- [ ] **步骤 1：编写对象键和上传安全测试**

覆盖：

- 数据库模型保存 `logoObjectKey`、`loginBackgroundObjectKey`。
- 租户 JSON 动态返回签名 URL。
- `/auth/config?tenantCode=...` 返回登录 Logo 和背景签名 URL。
- 请求体超过 16MB 被拒绝，不产生临时大文件。
- 扩展名、检测 MIME 和允许类型不一致时被拒绝。
- SVG 只允许 Logo，不允许背景。

- [ ] **步骤 2：运行测试验证失败**

运行：`go test ./internal/tenant ./internal/auth -run 'Upload|Asset|Config'`

预期：FAIL。

- [ ] **步骤 3：修改最终数据库基线和查询**

将：

```sql
logo_url text,
login_background_url text
```

替换为：

```sql
logo_object_key text,
login_background_object_key text
```

更新所有查询别名和模型映射，不保留旧字段兼容。

- [ ] **步骤 4：生成 sqlc 代码**

运行：`go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.30.0 generate`

预期：PASS。

- [ ] **步骤 5：实现签名 URL 输出和上传校验**

Handler 使用：

```go
r.Body = http.MaxBytesReader(w, r.Body, 16<<20)
```

读取文件头后使用 `http.DetectContentType`，并校验允许扩展名。Service 上传后持久化对象键，响应前调用 `PresignGet`。

- [ ] **步骤 6：运行租户和认证测试**

运行：`go test ./internal/tenant ./internal/auth`

预期：PASS。

### 任务 8：发布清单和节点命令使用短期签名工件 URL

**文件：**
- 修改：`dev_core/internal/deployment/artifact.go`
- 修改：`dev_core/internal/deployment/service.go`
- 修改：`dev_core/internal/deployment/repository.go`
- 修改：`dev_core/internal/deployment/handler_test.go`
- 修改：`dev_core/internal/worker/worker.go`
- 修改：`dev_core/internal/worker/worker_test.go`

- [ ] **步骤 1：编写发布清单和命令测试**

断言：

- 版本清单包含 `artifactBucket`、`artifactKey`、`artifactHash`、`artifactSize`。
- 版本清单不包含 `artifactUrl`。
- 创建部署时调用 `PresignGet` 并把签名 URL 写入节点命令。
- 签名失败时事务不创建部分部署记录。
- Worker 重发部署命令时重新生成签名 URL。

- [ ] **步骤 2：运行测试验证失败**

运行：`go test ./internal/deployment ./internal/worker`

预期：FAIL。

- [ ] **步骤 3：实现对象引用发布和命令签名**

`deployment.Store` 接口改为：

```go
type Store interface {
    Put(context.Context, string, io.Reader, int64, string) (objectstore.ObjectRef, error)
    PresignGet(context.Context, string, time.Duration) (string, error)
}
```

发布持久化对象键；部署和 Worker 发命令前生成 30 分钟签名 URL。不得把签名 URL写回版本 Manifest。

- [ ] **步骤 4：运行发布、Worker 和 Node Agent 测试**

运行：

```powershell
go test ./internal/deployment ./internal/worker
cd ../runtime/node_agent
go test ./internal/web/handler
```

预期：PASS。

### 任务 9：修复 Refresh Token 和验证码

**文件：**
- 修改：`dev_core/db/queries/auth.sql`
- 修改：`dev_core/internal/auth/repository.go`
- 修改：`dev_core/internal/auth/service.go`
- 修改：`dev_core/internal/auth/handler_test.go`
- 创建：`dev_core/internal/auth/repository_integration_test.go`
- 生成：`dev_core/internal/platform/db/sqlc/auth.sql.go`

- [ ] **步骤 1：编写验证码必填和轮换测试**

- 登录缺少验证码 Key 或 Code 均返回验证码错误。
- 同一验证码只能使用一次。
- Refresh Token 第二次轮换返回无效。
- 数据库可用时，两个并发轮换请求只有一个成功。

- [ ] **步骤 2：运行测试验证失败**

运行：`go test ./internal/auth -run 'Captcha|Refresh|Rotate'`

预期：FAIL。

- [ ] **步骤 3：实现原子轮换**

SQL 使用条件更新返回旧 Token：

```sql
-- name: ConsumeRefreshToken :one
UPDATE refresh_tokens
SET revoked_at = now(), last_used_at = now()
WHERE token_hash = sqlc.arg(token_hash)
  AND revoked_at IS NULL
  AND expires_at > now()
RETURNING id, tenant_id, user_id;
```

在同一事务内成功消费后创建替换 Token，再更新 `replaced_by_token_id`。影响 0 行映射为 `ErrInvalidRefresh`。

- [ ] **步骤 4：验证码改为无条件校验**

`Login` 在查询用户前始终调用 `verifyCaptcha`，缺失字段直接失败。

- [ ] **步骤 5：生成代码并运行认证测试**

运行：

```powershell
go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.30.0 generate
go test ./internal/auth
```

预期：PASS。

### 任务 10：拆分控制面 Socket.IO 路径

**文件：**
- 修改：`dev_core/cmd/dev_core/main.go`
- 修改：`dev_core/internal/realtime/server.go`
- 修改：`dev_ide/src/utils/socket.ts`
- 修改：`dev_ide/vite.config.ts`
- 修改：`dev_ide/tests/unit/vite-proxy.test.ts`
- 创建：`dev_ide/tests/unit/socket-path.test.ts`
- 修改：`scripts/docker/edge/nginx.conf`

- [ ] **步骤 1：编写路径配置失败测试**

断言：

- dev_ide 控制面 Socket path 为 `/control-socket.io`。
- Vite `/control-socket.io` 指向 `VITE_API_URL`，`/socket.io` 指向 `VITE_DATA_SERVICE_URL`。
- Nginx `/control-socket.io` 指向 `control:18101` 并有 Upgrade 头。

- [ ] **步骤 2：运行前端测试验证失败**

运行：`pnpm --filter ide test -- --run tests/unit/vite-proxy.test.ts tests/unit/socket-path.test.ts`

预期：FAIL。

- [ ] **步骤 3：修改后端、前端和 Nginx 路径**

Go 只挂载：

```go
router.Handle("/control-socket.io", realtimeServer.Handler())
router.Handle("/control-socket.io/*", realtimeServer.Handler())
```

dev_ide Socket 客户端设置 `path: '/control-socket.io'`。

- [ ] **步骤 4：运行前端测试和类型检查**

运行：

```powershell
pnpm --filter ide test -- --run
pnpm --filter ide typecheck
```

预期：PASS。

### 任务 11：收紧 OpenAPI 并纠正文档

**文件：**
- 修改：`dev_core/api/openapi.yaml`
- 生成：`dev_core/internal/platform/api/generated.go`
- 修改：`dev_core/tests/contract/dev_ide_endpoints_test.go`
- 修改：`dev_core/tests/contract/endpoint_inventory.go`
- 修改：`docs/superpowers/specs/2026-08-03-designer-code-first-product-design.md`
- 修改：`docs/superpowers/plans/2026-08-03-designer-code-server-foundation-plan.md`
- 修改：`docs/superpowers/plans/2026-08-03-designer-code-server-roadmap.md`
- 修改：`docs/superpowers/plans/2026-08-03-dev-core-dev-ide-api-plan.md`

- [ ] **步骤 1：增加 OpenAPI 守卫测试**

守卫必须失败于：

- 任意 requestBody 仍引用 `GenericPayload`。
- 74 个 operationId 数量改变。
- 写接口缺少 requestBody Schema。

AST HTTP 测试扫描的描述改为“路由请求证据”，不再称为完整语义覆盖。

- [ ] **步骤 2：运行契约测试验证失败**

运行：`go test ./tests/contract`

预期：FAIL，存在 `GenericPayload`。

- [ ] **步骤 3：定义领域化 Schema**

为认证、租户、用户、工程、运行访问、节点、发布、部署和日志写接口定义实际请求对象。统一响应保留 `code`、`msg`、`data`、`reqId`，列表响应明确 `list` 与 `pagination`。

- [ ] **步骤 4：重新生成 OpenAPI 代码**

运行：

```powershell
cd dev_core/internal/platform/api
go generate
```

预期：生成成功。

- [ ] **步骤 5：纠正 code-server 和完成状态文档**

删除或改写 Node.js `dev_core`、Dockerode、`http-proxy`、一次性代理票据和控制面反向代理要求。明确本阶段由 `dev_core` 后续创建容器并返回开发态端口，Designer iframe 直连。

在 Go 重写计划中勾选已真实完成的基础任务；安全修复相关任务在验证通过后再勾选。

- [ ] **步骤 6：运行契约和文档扫描**

运行：

```powershell
go test ./tests/contract
rg -n "Dockerode|http-proxy|codeWorkspaceProxyService|经过 `dev_core`.*代理" docs/superpowers
```

预期：契约 PASS；正式 code-server 规格与计划中不存在旧代理实现要求。

### 任务 12：调整开发数据库并执行最终验证

**文件：**
- 修改：仅当前 WSL/Docker 开发数据库，不新增迁移文件
- 检查：全部本轮修改文件

- [ ] **步骤 1：直接调整当前开发数据库**

数据库可用时执行：

```sql
UPDATE projects SET visibility = 'internal' WHERE visibility = 'public';
ALTER TABLE projects DROP CONSTRAINT projects_visibility_check;
ALTER TABLE projects ADD CONSTRAINT projects_visibility_check
  CHECK (visibility IN ('private', 'internal'));
ALTER TABLE tenants RENAME COLUMN logo_url TO logo_object_key;
ALTER TABLE tenants RENAME COLUMN login_background_url TO login_background_object_key;
```

执行前查询实际约束名；SQL 只用于当前开发数据库，不保存为迁移脚本。

- [ ] **步骤 2：验证空库最终基线**

运行：`go test ./internal/platform/db`

预期：空库基线包含最终字段和约束，不包含 ALTER、DROP、UPDATE、DELETE 迁移逻辑。

- [ ] **步骤 3：执行 Go 全量测试与静态检查**

运行：

```powershell
$env:GOSUMDB='sum.golang.org'
pnpm go:test
pnpm go:vet
```

预期：PASS。

- [ ] **步骤 4：执行前端测试、类型检查和构建**

运行：

```powershell
pnpm typecheck
pnpm --filter ide test -- --run
pnpm --filter ide build
```

预期：PASS；允许既有 Rollup chunk 警告，不允许 TypeScript 或测试错误。

- [ ] **步骤 5：验证格式和差异**

运行：

```powershell
gofmt -w dev_core
git diff --check
git status --short
```

只格式化 `dev_core`，不运行会自动修改其他模块的根 `pnpm lint`。

- [ ] **步骤 6：Docker 可用时验证 Compose**

运行：

```powershell
docker compose --env-file .env -f scripts/docker/docker-compose.dev.yml config --quiet
docker compose --env-file .env -f scripts/docker/docker-compose.prod.yml config --quiet
docker compose --env-file .env -f scripts/docker/docker-compose.offline.yml config --quiet
```

预期：PASS。Docker daemon 不可用时明确记录为未验证项。

- [ ] **步骤 7：最终复审**

重新检查：

- 没有持久化签名 URL。
- 没有 Service 绕过能力校验。
- 没有工作空间符号链接读取路径。
- 没有 `GenericPayload` 请求体。
- 没有旧 code-server 代理实施要求。
- `dev_core` 新文件全部纳入最终变更清单。
