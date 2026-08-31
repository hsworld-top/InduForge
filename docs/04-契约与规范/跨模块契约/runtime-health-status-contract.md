# Runtime 健康检查与状态协议

## 1. 范围与权威来源

本协议对应 [runtime-health-status.schema.json](../../../contracts/runtime/runtime-health-status.schema.json)，用于一个工程单节点部署中的 Project Gateway、Runtime API、RuntimeEngine 和 Industrial Collector。它不定义中心的节点在线推断、主机指标、发布状态或未来分布式调度。

所有 HTTP 响应使用 `code`、`msg`、`data`、`reqId` 包络；只有 `code=0` 是成功。NodeAgent 探测的健康 URL 必须由本地 `ServiceConfig` 明确声明并限制为回环地址，中心不得把任意探测 URL 或命令传给 Agent。Runtime API 的 PostgreSQL 故障健康分支当前仍错误地在 HTTP `503` 中返回 `code=0`，该实现差异见第 5 节，不能据此放宽本协议。

## 2. `GET /health`

`GET /health` 是轻量进程探测，成功返回：

```json
{
  "code": 0,
  "msg": "ok",
  "data": { "status": "UP", "observedAt": "2026-08-31T10:00:00Z" },
  "reqId": "..."
}
```

RuntimeEngine 仅在 `RUNNING` 且 `HEALTHY`/`DEGRADED` 时返回 `200/UP`；停止、排空或失败返回 `503/DOWN`。Gateway 当前自身可服务时返回 `200/UP`，并额外给出 `upstreamStatus`；其 `/api/v1/status` 会将 Runtime API 不可用表为 `DEGRADED`。`/health` 不能代替完整状态或业务新鲜度。

## 3. `GET /api/v1/status` 的冻结模型

schema 的根对象必须是状态数据（再由 HTTP 包络承载），基础必填字段为：

| 字段                                                  | 约束                                                                                          |
| ----------------------------------------------------- | --------------------------------------------------------------------------------------------- |
| `schemaVersion`                                       | 固定为 `runtime-health-status.v1`                                                             |
| `componentRole`                                       | `project-gateway`、`runtime-api`、`runtime-engine`、`industrial-collector`、`compute-sandbox` |
| `siteId`、`executionForm`                             | 必填；执行形态为 `k3s-workload`、`native-linux` 或 `native-windows`                           |
| `lifecycleState`、`healthState`                       | 必填枚举，见下节                                                                              |
| `version`、`startedAt`、`uptimeSeconds`、`observedAt` | 必填；时间均为 RFC3339 UTC                                                                    |

`project-gateway`、`runtime-api`、`runtime-engine`、`industrial-collector` 还必须给出 `deploymentId`、`accountId` 和 `businessFreshness`。原生 Collector 必须给出 `nodeId`、`processId`、`collector` 详情，且只能为 native 执行形态；原生 Linux 的 Gateway 与 RuntimeEngine 也必须给出 `nodeId`、`processId`。非 Collector 不能携带 `collector` 对象。

`businessFreshness` 的字段固定为 `state`、`lastBusinessEventAt`、`evaluatedAt`；不得用自由字段名替换。Collector 详情的 assignment、binding、artifact、ownership 与 WAL 指标也以 schema 为准。

## 4. 枚举和语义

- 生命周期：`STARTING`、`RUNNING`、`STOPPING`、`STOPPED`、`FAILED`。
- 健康：`HEALTHY`、`DEGRADED`、`UNAVAILABLE`、`MAINTENANCE`、`UNKNOWN`。
- `FRESH`、`STALE`、`EXPIRED`、`UNKNOWN` 属于 `businessFreshness.state`；它表示业务事件证据的新鲜度，不等于进程是否存活。

`FAILED` 应同时报告 `UNAVAILABLE`；可服务但有依赖或业务降级时使用 `DEGRADED` 与稳定的 `reasonCode`。`lastError` 只放短摘要，不得放 Secret、DSN、token、完整日志或 Artifact 内容。

## 5. 当前实现符合度与缺口

| 组件                 | 已实现事实                                             | 与冻结 schema 的缺口                                                                                                                                                                                        |
| -------------------- | ------------------------------------------------------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| RuntimeEngine        | `/health`、`/api/v1/status`；native v2 会报告 node/PID | 当前最接近 schema 的实现；实际部署仍依赖只读 release 挂载和外部依赖配置                                                                                                                                     |
| Project Gateway      | `/health`、聚合 `/api/v1/status`                       | status 携带 schema、deployment/account/node/process 身份和 `businessFreshness`；Runtime API 不可用时为 `DEGRADED`                                                                                           |
| Runtime API          | `/health`、`/api/v1/status`，并探测 PostgreSQL/NATS    | status 携带 schema、deployment/account/node/process 身份；业务新鲜度使用冻结的 `lastBusinessEventAt`/`evaluatedAt` 字段，依赖异常时返回明确 reasonCode；PostgreSQL 故障时 `/health` 返回 `503 / code=50031` |
| Industrial Collector | 包构建和 NodeAgent 组托管已接入                        | 本仓库本轮未完成与本 schema 的真实进程端到端验证                                                                                                                                                            |

NodeAgent 使用回环 `/health` 做进程托管；中心采集完整状态时使用 `/api/v1/status`。当前 Gateway、Runtime API 与 RuntimeEngine 已按冻结字段输出状态，但仍需在真实安装验收中把响应送入共享 JSON Schema 校验器，不能只以 HTTP 200 作为契约验收结论。

## 6. 关联文档

- [NodeAgent 与运行系统协议](./node-agent-runtime-protocol.md)
- [Runtime API 契约](./runtime-api-contract.md)
- [工程运行系统架构](../../02-系统设计/工程运行系统架构.md)
