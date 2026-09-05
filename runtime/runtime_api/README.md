# Runtime API

Runtime API 是发布后工程的唯一动态 HTTP/WebSocket 边界。它运行在物理节点回环地址上，只能由同节点 Project Gateway 代理，浏览器不会直接访问 PostgreSQL、NATS、中心对象存储或现场 Secret。

当前 API 已实现以下能力；经 Project Gateway 可访问的范围见下文：

- `/health`、`/api/v1/status` 统一包络及真实 PostgreSQL/NATS 状态；
- 读取 `runtime-project-artifact.v1`，以 Release 中的 path/UUID 目录约束查询；
- 当前值、受限时间窗口历史值、报警状态和计算单元只读目录；
- NATS `data.raw.>` / `data.computed.>` 实时事件身份校验与 `/ws/v1/points` 订阅；
- SHA-256 Bearer Secret 换取 HttpOnly SameSite 运行会话；
- 按 Artifact 权限写入 `manual.input` 数据点、按角色确认报警，以及人工计算命令的幂等审计入队、发布和状态查询；
- 所有工程数据请求同时校验 Gateway 注入的 deployment/project 身份。

人工计算要求启用的手工触发计算单元及 `admin` 或 `operator` 角色，成功响应表示命令已受理，执行结果需查询命令状态。`manual.input` 写入是工程人工数据事件，不等同于工业设备写入；未实现的设备动作仍需先明确受控命令与审计契约。

当前 Project Gateway 只开放白名单中的只读查询和实时订阅，使用服务端持有的 `viewer` token，移除客户端 Authorization；创建会话、人工写入、报警确认、人工计算及其他未开放的运行接口返回 `403 / code=40301`。API 内部已有动作实现，不代表发布页面已经能执行这些动作。

中心已维护工程用户、密码哈希、角色与权限，但当前部署只交付 `deployment-user / viewer` token；工程用户名密码登录、账号禁用及角色权限发布尚未接通。目标是通过受控运行身份发布和同源会话完成这些能力，并在真实 Gateway 路径验证授权、拒绝与审计，不能通过绕过入口或暴露部署 Secret 开放操作。详细边界见 [Runtime API 契约](../../docs/04-契约与规范/跨模块契约/runtime-api-contract.md)。

```sh
runtime-api \
  --listen 127.0.0.1:17801 \
  --artifact /var/lib/induforge-node/releases/release-1/runtime/runtime-project.json \
  --postgres-secret /var/lib/induforge-node/secrets/deployment-1/postgres.json \
  --token-secret /var/lib/induforge-node/secrets/deployment-1/runtime-api-tokens.json \
  --nats-url nats://127.0.0.1:4222 \
  --nats-credentials /var/lib/induforge-node/secrets/deployment-1/nats.creds \
  --deployment-id deployment-1 \
  --project-id 11111111-1111-4111-8111-111111111111 \
  --account-id account-1 \
  --site-id site-1 \
  --node-id node-1 \
  --version release-1 \
  --execution-form native-linux
```

Project Gateway 只把允许的运行请求转发到该回环端口，并注入 `X-InduForge-Deployment-Id`、`X-InduForge-Project-Id`；后续开放登录和动作接口仍须保持这一身份边界。按变更影响在本模块选择 `make build`、`make test` 或 `make vet` 验证。
