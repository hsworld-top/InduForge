# Runtime API

Runtime API 是发布后工程的唯一动态 HTTP/WebSocket 边界。它运行在物理节点回环地址上，只能由同节点 Project Gateway 代理，浏览器不会直接访问 PostgreSQL、NATS、中心对象存储或现场 Secret。

首个正式 `edge-single` 切片已实现：

- `/health`、`/api/v1/status` 统一包络及真实 PostgreSQL/NATS 状态；
- 读取 `runtime-project-artifact.v1`，以 Release 中的 path/UUID 目录约束查询；
- 当前值、受限时间窗口历史值、报警状态和计算单元只读目录；
- NATS `data.raw.>` / `data.computed.>` 实时事件身份校验与 `/ws/v1/points` 订阅；
- SHA-256 Bearer Secret 换取 HttpOnly SameSite 运行会话；
- 所有工程数据请求同时校验 Gateway 注入的 deployment/project 身份。

当前 Runtime V1 尚未冻结设备写入与人工计算命令的可审计请求/结果协议，因此对应端点明确返回 `501 / code=50031`，不会返回伪成功。

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

Project Gateway 必须把 `/api/v1/runtime/*` 和 `/ws/*` 转发到该回环端口，并注入 `X-InduForge-Deployment-Id`、`X-InduForge-Project-Id`。验证：`make build test vet`。
