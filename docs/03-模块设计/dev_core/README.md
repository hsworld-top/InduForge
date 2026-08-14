# dev_core 控制面

## 模块定位

`dev_core` 是使用 Go 实现的平台控制面后端，面向 `dev_ide` 提供租户、用户、工程、运行权限、节点、发布部署、审计日志和实时事件能力。旧 Node.js `dev_core` 已移除。

## 技术基线

- HTTP：Chi + OpenAPI + oapi-codegen，统一前缀 `/api/v1`，健康检查 `/health`。
- 数据库：PostgreSQL + `pgx/v5` + sqlc，只维护从零创建最终结构的建库基线。
- 缓存：Redis，用于登录滑块挑战、Token 吊销和节点在线 TTL。
- 对象存储：SeaweedFS S3，设计资源与 `.ifp` 工件使用独立桶。
- 实时事件：Socket.IO v4 协议，路径 `/socket.io`，按认证用户的租户隔离房间。
- 工程源码：`CODE_WORKSPACE_ROOT/{projectId}/workspace`。`dev_core` 只创建空目录，客户侧工程进入开发容器后从四套官方 Vue/React Vite 模板中首次选择。

## 能力边界

- 保留 `dev_ide` 当前使用的 74 个控制面 REST 操作，契约源位于 `dev_core/api/openapi.yaml`。
- 支持 Node Agent 注册、审批查询、心跳、离线和部署状态回报。
- 发布时构建工程 Vite 源码，聚合前端静态产物、HT 运行资源、数据与采集配置，生成不可变 Release 并上传对象存储。
- 审计中间件不读取请求体，不记录密码、Token、节点密钥或上传文件内容。
- 不维护与工程源码并行的页面配置模型，也不承担浏览器端页面渲染。
- Designer 通过工程工作空间、数据点上下文包以及 2D/3D 场景公开契约与控制面协作。

## 实时事件

- `ops:node:pending`
- `ops:node:metrics`
- `ops:node:status`
- `ops:project:metrics`
- `ops:deploy:status`

客户端使用 Access Token 建立连接，并发送 `ops:subscribe`。服务端忽略客户端声明的租户归属，只允许加入 Token 对应租户的房间。

## 开发验证

```powershell
$env:GOSUMDB='sum.golang.org'
cd dev_core
gofmt -w .
go test ./...
go vet ./...
```

根目录统一验证入口为 `pnpm test:core`、`pnpm lint:core` 与 `pnpm typecheck`。

生产或离线安装使用镜像内的 `/app/dev_core init` 命令，只允许在空库中创建最终结构、默认租户、超级管理员和默认租户管理员；已有业务表时不会自动迁移或修补。
