# RuntimeEngine

构建与静态校验：`make build test vet`。二进制不会创建 NATS stream/consumer、PostgreSQL 表或迁移；运维必须在部署前显式安装 [schema.sql](internal/store/postgres/schema.sql) 并配置现有拓扑。

生产启动示例：

```sh
runtime-engine --production \
  --config /run/runtime/runtime-engine-config.json \
  --config-root /run/runtime \
  --index /run/runtime/site-index.json \
  --listen 127.0.0.1:17802
```

`--production` 默认开启；它要求只读配置/secret 文件并拒绝 NATS `none` 认证。站点索引使用 `collector-runtime-index.v1`，资源可内嵌，secret 必须是索引同目录下的相对普通文件：

本机物理节点由 NodeAgent 使用 `runtime-engine.config.v2` 启动：`executionForm` 必须为 `native-linux`，`artifactMount.source` 必须为 `native-release`，并提供本节点的 `nodeId`。`artifactMount.mountPath` 必须是绝对、规范化、无符号链接的只读 release 根；Loader 会校验根及每一个 Artifact 的实际只读挂载、路径边界与原始字节 SHA-256。v1 仍仅表示 `k3s-workload` / `release-pvc`，不会被自动解释为本机进程。

```json
{"schemaVersion":"collector-runtime-index.v1","resources":{"site-resource://site-a/nats":{"url":"nats://nats:4222","accountId":"account-a"}},"secrets":{"secret://site-a/deployment/runtime-postgres":"runtime-postgres.json"}}
```

secret 文件应由运行时挂载、仅服务账号可读且不可写，例如 PostgreSQL：`{"schemaVersion":"postgres-dsn.v1","dsn":"postgres://USER:PASSWORD@HOST/DB?sslmode=require"}`。不要把 index 或 status 输出写入日志。

`GET /health` 在 RUNNING 且 HEALTHY/DEGRADED 时返回 `200` 和 `UP`；其余生命周期返回 `503` 与 `DOWN`。`GET /api/v1/status` 始终返回状态包络。SIGTERM/SIGINT 进入 STOPPING，最多 20 秒排空；超时以 FAILED 退出，绝不声称已排空。
