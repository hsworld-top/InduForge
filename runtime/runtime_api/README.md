# Runtime API

Runtime API 是发布后工程的唯一动态 HTTP/WebSocket 边界。它运行在物理节点回环地址上，只能由同节点 Project Gateway 代理，浏览器不会直接访问 PostgreSQL、NATS、中心对象存储或现场 Secret。

运行态对象库通过 `GET /api/v1/runtime/assets`、`GET /api/v1/runtime/assets/{assetId}` 和 `GET /api/v1/runtime/assets/{assetId}/content` 提供。内容接口支持 Range 与 ETag，响应只包含工程 `assetId` 和公开元数据，不返回对象存储 key。运行制品固定把 `object-library/` 作为对象库根目录，并由 `runtime-assets.v1` 的 `manifest.json` 描述资源；部署工作负载通过 `--asset-root /work/runtime-api-artifact/object-library` 显式挂载。Runtime API 启动时会校验版本、重复 ID、文件路径和文件大小，发现缺失资源直接失败，避免发布后才暴露问题。未配置目录时接口会返回明确的“运行态对象库未配置”，不会降级为任意文件读取。

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

运行用户可以作为 Argon2id 摘要写入 `runtime-api-tokens.v1` secret 的 `users` 数组，Runtime API 通过 `POST /api/v1/runtime/session` 的用户名密码建立同源会话；发布调和会从工程运行用户、角色和授权表生成该受限快照，密码明文不会进入 Artifact 或上下文。详细边界见 [Runtime API 契约](../../docs/04-契约与规范/跨模块契约/runtime-api-contract.md)。

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
