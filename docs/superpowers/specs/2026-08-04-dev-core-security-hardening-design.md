# dev_core 安全与交付修复设计

**日期：** 2026-08-04  
**状态：** 已完成方案讨论，等待书面规格审阅  
**范围：** 修复 `dev_core` 现有实现的权限、安全、对象存储、实时通信、接口契约和计划文档问题

## 1. 背景

`dev_core` 已完成 Go 控制面基座、74 个 `dev_ide` REST 路由、最终数据库基线、认证、租户与用户、工程、运行权限、节点、发布部署、审计日志和实时事件的首轮重写。

现有实现通过了常规 Go 测试、Go vet、前端类型检查和 `dev_ide` 构建，但审查发现以下阻塞问题：

- 工程工作空间导出会跟随符号链接，存在控制面敏感文件泄露风险。
- 对象存储返回未签名的私有 S3 URL，租户图片和节点发布工件无法下载。
- 用户角色缺少层级限制，租户管理员可以提升为全局超级管理员。
- 工程、运行权限、节点和部署接口普遍只有登录校验，没有能力和工程可见性校验。
- 控制面和数据服务共用 `/socket.io`，生产代理把控制面事件转发到错误服务。
- Refresh Token 轮换、验证码和上传大小限制存在安全缺口。
- OpenAPI 和接口测试偏重路由存在性，不能约束实际请求结构和权限语义。
- code-server 规格仍保留已否决的 Node.js `dev_core` 代理方案。

本设计只处理上述问题，不实现 code-server、2D、3D 和工程上下文包。

## 2. 成功标准

完成后必须满足：

1. 用户不能通过工作空间符号链接读取控制面容器外部文件。
2. 租户图片和节点发布工件可以在 SeaweedFS 私有桶模式下正常下载。
3. 用户不能赋予超出自身管理边界的角色，不能修改自己的角色和状态。
4. 私有工程只对创建人和系统级管理员可见；分享工程按租户角色能力开放。
5. 所有工程、运行用户、节点、发布和部署写接口都有集中式能力校验。
6. 同一个 Refresh Token 在并发请求下最多成功轮换一次。
7. 登录验证码不能通过省略字段绕过。
8. 租户图片上传请求总大小严格限制为 16MB。
9. 控制面实时事件在开发、生产和离线拓扑中都连接到 `dev_core`。
10. OpenAPI 请求结构可以约束真实字段，关键安全路径有失败测试。
11. code-server 正式规格与已确认的开发态直连端口方案一致。

## 3. 集中式能力模型

### 3.1 能力定义

`auth` 包维护固定的角色能力矩阵，不引入数据库动态 RBAC。

能力至少包括：

- `tenant:manage`
- `user:manage`
- `project:read`
- `project:create`
- `project:write`
- `project:share`
- `project:delete`
- `runtime-access:manage`
- `release:publish`
- `deployment:execute`
- `deployment:operate`
- `node:read`
- `node:approve`
- `node:delete`
- `audit-log:read`
- `audit-log:delete`

角色能力边界：

- `SUPER_ADMIN`：跨租户平台管理和所有能力。
- `SYSTEM_ADMIN`：本租户内所有能力，不具备跨租户管理能力。
- `PROJECT_ADMIN`：工程管理、分享、删除、运行用户与角色、发布和部署能力。
- `OPS_ADMIN`：工程只读、节点审批与删除、部署执行与运行操作、审计日志查看。
- `USER_ADMIN`：只管理本租户用户。
- `DEVELOPER`：查看分享工程、创建工程、修改可访问工程、创建发布版本；不能审批节点或执行正式部署。
- `OPERATOR`：查看分享工程并操作已有部署；不能修改工程、创建版本或审批节点。
- `VIEWER`：只读访问分享工程。

### 3.2 工程可见性

沿用前端已经使用的枚举：

- `private`：私有工程。
- `internal`：租户内分享工程。

删除数据库基线中未使用的 `public` 枚举。

访问规则：

- 私有工程仅创建人、`SYSTEM_ADMIN`、`SUPER_ADMIN` 可以查看。
- 分享工程允许本租户具备 `project:read` 的用户查看。
- 修改工程必须同时满足工程可见性和 `project:write` 能力。
- 创建人被降级为只读角色后不能继续修改工程。
- 工程分享状态和工程删除只能由创建人或 `PROJECT_ADMIN`、`SYSTEM_ADMIN`、`SUPER_ADMIN` 操作，并且操作者仍需对应能力。
- `SUPER_ADMIN` 的跨租户访问必须由明确租户参数或租户管理入口触发，普通工程 ID 不绕过租户过滤。

工程列表在 PostgreSQL 中完成访问过滤：

```text
tenant_id = 当前租户
AND status <> deleted
AND (
  visibility = internal AND 当前角色具备 project:read
  OR created_by = 当前用户
  OR 当前角色为 SYSTEM_ADMIN/SUPER_ADMIN
)
```

不得查询租户全部工程后在浏览器或 Go 内存中二次过滤。

### 3.3 服务校验边界

Handler 只负责认证和解析请求，Service 负责最终授权，避免内部调用绕过权限。

统一提供：

- `HasCapability(role, capability)`：判断平台角色能力。
- `RequireCapability(actor, capability)`：返回统一权限错误。
- `CanReadProject(actor, project)`：检查租户、可见性和创建人。
- `RequireProjectCapability(actor, project, capability)`：同时检查可见性和能力。
- `RequireProjectOwnerOrAdmin(actor, project, capability)`：用于分享和删除。

节点管理是租户级能力，不依赖工程可见性。部署和发布必须先读取工程，再执行工程访问和能力校验。

### 3.4 用户角色层级

- `USER_ADMIN` 只能创建和管理 `DEVELOPER`、`OPERATOR`、`VIEWER`。
- `SYSTEM_ADMIN` 可以管理除 `SUPER_ADMIN` 外的本租户角色。
- 只有 `SUPER_ADMIN` 可以赋予或撤销 `SUPER_ADMIN`。
- 任何用户都不能通过用户管理接口修改自己的角色或状态。
- 用户创建和更新必须在 Service 层校验角色值和角色层级，不能依赖数据库 CHECK 约束返回错误。
- 修改用户密码后撤销该用户全部 Refresh Token。

## 4. 工作空间安全与资源限制

### 4.1 文件类型

工作空间导出和发布快照只接受普通文件：

- 使用 `DirEntry.Type()`、`Info().Mode()` 或等价方式识别符号链接。
- 遇到符号链接立即返回明确业务错误，不调用 `os.ReadFile` 跟随目标。
- 拒绝命名管道、Socket、设备文件和其他非普通文件。
- 工作空间根目录必须经过绝对路径和边界校验。

### 4.2 排除目录

导出和发布排除：

- `node_modules`
- `dist`
- `.git`
- `.pnpm-store`
- `.vite`
- `code-server-data`
- `code-server-config`
- IDE 缓存目录
- `.induforge/context` 生成上下文目录

### 4.3 大小限制

- 单个普通文件最大 32MB。
- 单次工程导出或发布源文件总量最大 512MB。
- 统计原始文件大小，不以 Base64 或 ZIP 后大小代替。
- 超限立即停止遍历并返回统一业务错误。

本轮保留现有内存构建方式，但通过总量限制避免无界内存占用。后续正式构建容器阶段再改为流式快照和流式压缩。

## 5. 私有对象存储

### 5.1 对象引用

对象存储接口拆分为：

- `Put`：上传并返回 `bucket`、`objectKey`、大小和内容类型，不返回永久 URL。
- `PresignGet`：按对象键生成带过期时间的签名下载 URL。
- `Delete`：删除对象，用于后续资源清理。

SeaweedFS S3 桶保持私有，不增加匿名读策略，不通过 Nginx 暴露 S3 管理入口。

### 5.2 租户图片

数据库字段调整为：

- `logo_object_key`
- `login_background_object_key`

对外 JSON 仍返回 `logoUrl` 和 `loginBackgroundUrl`，但值由 Service 在响应时动态生成。

- 已登录租户接口生成短期签名 URL。
- `/api/v1/auth/config` 根据 `tenantCode` 查询租户并生成登录 Logo 和背景签名 URL，满足登录前展示。
- 签名 URL 只用于读取，不暴露访问密钥。

### 5.3 发布工件

`application_versions` 已有 `artifact_bucket` 和 `artifact_key`，继续作为持久化来源。

发布清单只保存：

- `artifactBucket`
- `artifactKey`
- `artifactHash`
- `artifactSize`
- `sourceHash`

不保存签名 URL。

创建节点部署命令时，根据 `artifact_bucket + artifact_key` 生成有效期 30 分钟的签名 URL，并写入命令 Payload。节点仍使用普通 HTTPS/HTTP GET 下载，不持有 S3 密钥。

签名过期且命令尚未执行时，由控制面重试 Worker 重新生成 URL 后再下发，不复用过期 URL。

## 6. 认证与上传安全

### 6.1 Refresh Token

轮换必须具备单次消费语义：

- 在事务内使用 `SELECT ... FOR UPDATE` 锁定旧 Token，或使用条件 `UPDATE ... RETURNING` 原子撤销。
- 只有成功撤销旧 Token 后才创建替换 Token，或在同一事务中保证失败回滚。
- 更新影响 0 行返回刷新令牌无效，不返回成功。
- 增加两个并发刷新请求只有一个成功的数据库集成测试。

### 6.2 验证码

- 登录请求必须同时提供 `captchaKey` 和 `captchaCode`。
- 缺少任一字段返回验证码错误。
- 验证码使用原子 Take，一次验证后立即失效。
- 用户名或密码错误时验证码也已经消费，避免重复尝试。

### 6.3 租户图片上传

- 在读取 Multipart 前使用 `http.MaxBytesReader` 将整个请求限制为 16MB。
- 解析后再次检查文件头大小和实际读取大小。
- Logo 允许 PNG、JPEG、WebP、SVG。
- 登录背景允许 PNG、JPEG、WebP。
- 扩展名和检测到的 MIME 必须匹配。
- 超限、类型不支持和内容不匹配使用明确业务错误，不返回内部错误。

## 7. 实时通信路径

- `dev_core` Socket.IO 路径改为 `/control-socket.io`。
- `dev_ide` 控制面实时客户端使用 `/control-socket.io`。
- Nginx 将 `/control-socket.io` 转发到 `control:18101`，开启 WebSocket Upgrade。
- data_service 继续使用 `/socket.io`，Nginx 继续将其转发到 `data:18102`。
- Vite 开发代理同时保留两个明确路径，禁止使用一个泛化路径覆盖两个服务。

必须测试开发代理和生产 Nginx 配置中的目标服务及 Upgrade 头。

## 8. OpenAPI 与测试

### 8.1 OpenAPI

- 删除写接口使用的 `GenericPayload`。
- 为认证、租户、用户、工程、运行用户与角色、节点、发布和部署定义实际请求 Schema。
- 响应至少区分列表、详情、分页和批量操作结果。
- 继续保持统一外层字段 `code`、`msg`、`data`、`reqId`。
- 重新生成 oapi-codegen 文件，并保持 74 个现有 operationId 不变。

### 8.2 必须新增的测试

- 每个角色的能力矩阵表驱动测试。
- 私有工程创建人、管理员、同租户其他用户的访问测试。
- 分享工程只读、修改、发布和部署权限测试。
- `USER_ADMIN` 和 `SYSTEM_ADMIN` 越级赋予角色测试。
- 自修改角色和状态拒绝测试。
- 工作空间符号链接和特殊文件拒绝测试。
- 单文件与总量超限测试。
- 私有对象上传后签名 URL 测试。
- 发布清单不包含签名 URL 测试。
- 节点命令包含短期签名 URL 测试。
- Refresh Token 并发单次消费测试。
- 缺少验证码字段的登录拒绝测试。
- Multipart 总请求大小和 MIME/扩展名测试。
- `/control-socket.io` 路由和 Nginx 目标测试。

现有 AST 路由扫描测试继续作为“接口未遗漏”守卫，但不得再描述为完整语义覆盖。

## 9. code-server 文档纠正

正式规格和路线图统一修改为：

- `dev_core` 使用 Docker Engine API 创建、启动、停止和查询一个工程一个共享 code-server 容器。
- `dev_core` 运行在容器内时，通过明确挂载的 Docker Socket 控制宿主 Docker。
- 开发阶段为工程容器分配宿主机端口，并由 API 返回当前端口、状态和 iframe URL。
- Designer 决定用户是否可以看到和进入代码中心，并通过 iframe 直接打开返回地址。
- code-server HTTP、静态资源和 WebSocket 不经过 `dev_core` 反向代理。
- 本阶段不实现一次性代理票据、Cookie 代理会话、Dockerode 或 `http-proxy`。
- code-server 目录挂载、源码产物和 IDE 状态边界维持既有产品设计。

本轮只修正文档，不实现上述容器管理能力。

## 10. 数据库影响

代码库只维护最终建库基线，不新增迁移脚本。

基线调整：

- `projects.visibility` 只允许 `private`、`internal`。
- `tenants.logo_url` 改为 `logo_object_key`。
- `tenants.login_background_url` 改为 `login_background_object_key`。

当前开发数据库按项目规则直接执行 SQL：

- 将历史 `public` 工程调整为 `internal`。
- 替换 `projects.visibility` CHECK 约束。
- 重命名租户图片字段。

服务启动时不得包含 ALTER、数据回填或旧字段兼容逻辑。

## 11. 不在本轮范围

- code-server 容器管理实现。
- Designer 三入口 UI 重构。
- 2D、3D 编辑器。
- 工程上下文包和数据点分片文档。
- 固定 Designer SDK。
- 正式构建容器和流式大工程构建。
- 数据库动态 RBAC 和逐用户工程成员选择。

## 12. 最终验证

至少执行：

```powershell
$env:GOSUMDB='sum.golang.org'
pnpm go:test
pnpm go:vet
pnpm typecheck
pnpm --filter ide test -- --run
pnpm --filter ide build
```

Docker 可用时执行：

```powershell
docker compose --env-file .env -f scripts/docker/docker-compose.dev.yml config --quiet
docker compose --env-file .env -f scripts/docker/docker-compose.prod.yml config --quiet
docker compose --env-file .env -f scripts/docker/docker-compose.offline.yml config --quiet
```

还必须人工或集成验证：

- 私有与分享工程的角色访问矩阵。
- SeaweedFS 私有桶下租户图片和远程节点工件下载。
- 两个并发 Refresh Token 请求只有一个成功。
- code-server 相关文档不再出现控制面代理实现要求。

