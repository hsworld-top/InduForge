# RuntimeEngine V1

构建与静态校验：`make build test vet`。二进制不会创建 NATS stream/consumer、PostgreSQL 表或迁移；运维必须在部署前显式安装 [schema.sql](internal/store/postgres/schema.sql) 并配置现有拓扑。

生产启动示例：

```sh
runtime-engine --production \
  --config /run/runtime/runtime-engine-config.json \
  --config-root /run/runtime \
  --index /run/runtime/site-index.json \
  --listen :8080
```

`--production` 默认开启；它要求只读配置/secret 文件并拒绝 NATS `none` 认证。站点索引使用 `collector-runtime-index.v1`，资源可内嵌，secret 必须是索引同目录下的相对普通文件：

```json
{"schemaVersion":"collector-runtime-index.v1","resources":{"site-resource://site-a/nats":{"url":"nats://nats:4222","accountId":"account-a"}},"secrets":{"secret://site-a/deployment/runtime-postgres":"runtime-postgres.json"}}
```

secret 文件应由运行时挂载、仅服务账号可读且不可写，例如 PostgreSQL：`{"schemaVersion":"postgres-dsn.v1","dsn":"postgres://USER:PASSWORD@HOST/DB?sslmode=require"}`。不要把 index 或 status 输出写入日志。

`GET /health` 在 RUNNING 且 HEALTHY/DEGRADED 时返回 `200` 和 `UP`；其余生命周期返回 `503` 与 `DOWN`。`GET /api/v1/status` 始终返回状态包络。SIGTERM/SIGINT 进入 STOPPING，最多 20 秒排空；超时以 FAILED 退出，绝不声称已排空。
