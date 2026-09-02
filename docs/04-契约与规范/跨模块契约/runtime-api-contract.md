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
| `GET /ws/v1/alarms`                         | 经会话认证后订阅当前 deployment 的 `alarm.event` 变更 |

`POST /api/v1/runtime/points/{path}/write` 仅适用于 `manual.input` 且当前会话角色通过 Artifact `runtimePermissions.write` 的数据点。Runtime API 在 PostgreSQL 中以 deployment binding 下发的 `manualOwner/manualEpoch` 复核 `runtime-api` producer fence 并原子分配 sequence，再只向本 deployment Account 的 `data.raw.<pointId>` 发布冻结的 `data.raw.v1` 事件；它不直接篡改 `point_current`。Engine 的 writer、compute 与 alarm 消费同一事件，浏览器只能以其后续投影/WS 观察结果。

`POST /api/v1/runtime/alarms/{id}/acknowledge` 仅允许 `admin` 或 `operator` 会话角色，body 必须含 `expectedVersion`（大于 0）和不超过 2048 字符的可选 `comment`。Runtime API 在同一 PostgreSQL 事务中以 `(deployment_id, alarm_item_id)` 锁定状态、复核版本与活动实例，更新活动实例的 `ackedAt`/`acknowledgedBy` 与状态版本，并插入不可变的 `alarm_ack_audit`。版本过期或报警已清除返回 HTTP `409 / code=40001`；该接口只作用于 Gateway 已绑定的 deployment。

`GET /ws/v1/alarms` 的首帧必须为 `{"action":"subscribe"}`，成功回 `{"type":"subscribed"}`；后续以 `{"type":"alarm","data":...}` 推送经过 deployment/account 过滤的 `alarm.event.v1` `alarm-transition`。SDK 默认在非主动断开时有限重连，调用方仍可通过关闭函数终止订阅。

`POST /api/v1/runtime/computes/{id}/run` 仍返回 `501 / code=50031`；设备写入、发布及其他动作在其各自受控命令/审计契约冻结前不得由 SDK 或 Gateway 伪造成功。

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
