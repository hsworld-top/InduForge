# Runtime API V1 契约

## 1. 边界

Runtime API 是发布工程的动态 HTTP/WebSocket 接口，必须只监听同一物理节点的 loopback 地址。浏览器通过 Project Gateway 同源访问；Gateway 代理时注入：

- `X-InduForge-Deployment-Id`
- `X-InduForge-Project-Id`

除 `/health` 与 `/api/v1/status` 外，所有 Runtime API HTTP/WS 路径都会核对这两个值；不匹配返回 `403 / code=40301`。调用成功的响应均为 `{code,msg,data,reqId}`，只有 `code=0` 成功。

## 2. 会话

| 方法和路径                       | 当前行为                                                                           |
| -------------------------------- | ---------------------------------------------------------------------------------- |
| `POST /api/v1/runtime/session`   | 用 Bearer token 建立 HttpOnly、SameSite 会话 Cookie，返回 subject、roles、过期时间 |
| `GET /api/v1/runtime/session`    | 返回当前会话身份                                                                   |
| `DELETE /api/v1/runtime/session` | 撤销当前会话                                                                       |

无有效 Bearer token 或会话 Cookie 时返回 `401 / code=40101`。token 只用于建立会话；发布前端应由 Runtime SDK 使用同源 Cookie，不应保存 PostgreSQL、NATS 或部署 Secret。

## 3. 当前只读数据面

| 方法和路径                                  | 当前行为                                              |
| ------------------------------------------- | ----------------------------------------------------- |
| `GET /api/v1/runtime/catalog`               | 返回 Runtime Artifact 的点位、计算和报警目录          |
| `GET /api/v1/runtime/points/{path}`         | 返回当前点位值；不存在或无运行值为 `404 / code=40401` |
| `GET /api/v1/runtime/points/{path}/history` | 返回受限时间窗口历史数据                              |
| `GET /api/v1/runtime/alarms/current`        | 返回当前报警                                          |
| `GET /api/v1/runtime/computes`              | 返回计算单元目录                                      |
| `GET /ws/v1/points`                         | 经会话认证后订阅 NATS 实时点位事件                    |

`POST /api/v1/runtime/points/{path}/write` 与 `POST /api/v1/runtime/computes/{id}/run` 已保留但当前明确返回 `501 / code=50031`。报警确认、设备写入、发布、任意动作及人工计算执行没有冻结请求/审计/结果契约，不能在 SDK 或 Gateway 中伪造成功。

## 4. 运行依赖和安全文件

启动参数要求 Runtime Artifact、PostgreSQL DSN secret、token secret、NATS URL/credentials，以及 deployment/project/account/site/node/version 身份。NATS credentials、PostgreSQL secret 必须是普通文件、不能为符号链接、不得被组或其他用户读取。Runtime API 启动时连接 PostgreSQL 和 NATS；任一不可用会使启动失败。

本 API 不自行下载 Release、不向中心查询对象存储、不创建 PostgreSQL schema 或 NATS 资源；这些由受控发布/运维流程提供。

## 5. 健康状态的当前边界

`GET /health` 和 `GET /api/v1/status` 已实现，并会探测 PostgreSQL/NATS。`/api/v1/status` 已输出 [Runtime 健康检查与状态协议](./runtime-health-status-contract.md) 冻结的 `runtime-health-status.v1` 身份、进程和业务新鲜度字段；仍须在真实安装环境中以共享 JSON Schema 校验实际响应，不能只以 HTTP 200 作为契约验收证据。

当 PostgreSQL 探测失败时，`/health` 返回 HTTP `503`、`code=50031`、
`msg=运行依赖不可用：PostgreSQL` 和 `status=DOWN`；健康时返回 HTTP `200`、`code=0`、
`msg=ok` 和 `status=UP`。调用方仍应同时检查 HTTP 状态与统一包络，不得只以单一字段判定依赖健康。

## 6. SDK 绑定

`@induforge/runtime-sdk` 的 `createHttpRuntime()` 默认同源请求 `/api/v1/runtime`、连接 `/ws/v1/points`；它会把服务端的失败包络原样返回，不会把未实现的写动作转换为成功。详情见 [Runtime SDK README](../../../runtime/web-sdk/README.md)。

## 7. 关联文档

- [工程运行微服务设计](../../02-系统设计/工程运行微服务设计.md)
- [NodeAgent 与运行系统协议](./node-agent-runtime-protocol.md)
- [Runtime 健康检查与状态协议](./runtime-health-status-contract.md)
