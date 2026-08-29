# Runtime 健康检查与状态协议

## 1. 文档定位

- 本文档定义节点侧运行组件对 NodeAgent、site-controller 和平台暴露的最小健康与状态协议。
- 适用组件包括 `runtime-api`、`runtime-engine` 各逻辑角色、工程入口、Windows/Linux 原生
  Collector 子进程以及后续纳入工程运行栈的长期运行服务。
- 本协议的目标是支撑托管、探活、错误定位和运维展示。
- 本协议只描述单组件健康，不承担主机资源、K3s、队列和长期指标传输；完整遥测见
  [平台运维体系设计](../../06-运维与安全/平台运维体系设计.md)。

## 2. 当前已实现

### 当前缺口

- `runtime-api`、工程静态入口和 `runtime-engine` 尚未实现，因此 `/health` 和
  `/api/v1/status` 目前没有事实接口。

## 3. 协议目标

- Runtime 必须提供轻量、稳定、低歧义的运维接口。
- NodeAgent、site-controller 和中心运维界面应基于该协议做探活、聚合和展示。

## 4. `GET /health`

### 目标

- 用于快速判断进程是否可服务。

### 成功响应

```json
{
  "code": 0,
  "msg": "ok",
  "data": {
    "status": "UP",
    "observedAt": "2026-08-30T10:00:00Z"
  },
  "reqId": "req_xxx"
}
```

### 约束

- 不返回重业务数据。
- 响应应足够轻量，适合高频调用。

## 5. `GET /api/v1/status`

### 目标

- 返回当前运行版本、项目、组件角色、启动时间、运行状态和最后错误。

### 建议响应

```json
{
  "code": 0,
  "msg": "ok",
  "data": {
    "lifecycleState": "RUNNING",
    "healthState": "HEALTHY",
    "componentRole": "runtime-api",
    "deploymentId": "deployment-project-a-prod",
    "projectId": "proj_xxx",
    "projectCode": "factory_dashboard",
    "version": "2026.03.06-001",
    "deploymentStage": "production",
    "executionForm": "k3s-workload",
    "siteId": "site-a",
    "nodeId": "node-01",
    "startedAt": "2026-03-06T10:05:00Z",
    "uptimeSeconds": 3600,
    "processId": 1234,
    "upstreamStatus": "CONNECTED",
    "walBacklog": 0,
    "lastError": null,
    "reasonCode": null,
    "observedAt": "2026-08-30T10:00:00Z"
  },
  "reqId": "req_xxx"
}
```

## 6. 状态枚举

生命周期状态：

- `STARTING`
- `RUNNING`
- `STOPPING`
- `STOPPED`
- `FAILED`

健康状态：

- `HEALTHY`
- `DEGRADED`
- `UNAVAILABLE`
- `MAINTENANCE`
- `UNKNOWN`

中心根据 `observedAt` 和 `receivedAt` 另行计算 `FRESH / STALE / EXPIRED`。状态过期只表示中心
没有新证据，不等同于节点或组件已经宕机。

## 7. 字段说明

| 字段             | 必填 | 说明                                                                 |
| ---------------- | ---- | -------------------------------------------------------------------- |
| `lifecycleState` | 是   | 进程或组件生命周期状态                                               |
| `healthState`    | 是   | 当前健康状态                                                         |
| `componentRole`  | 是   | 组件角色                                                             |
| `deploymentId`   | 否   | ProjectDeployment/原生 Collector 的稳定部署实例 ID                    |
| `projectId`      | 否   | 工程级组件的工程 ID；站点/主机组件不填                               |
| `projectCode`    | 否   | 工程编码                                                             |
| `version`        | 是   | 当前运行版本                                                         |
| `deploymentStage` | 否  | development、staging 或 production                                   |
| `executionForm`  | 是   | `k3s-workload`、`native-linux` 或 `native-windows`                    |
| `siteId`         | 是   | 当前运行站点 ID                                                      |
| `nodeId`         | 否   | 原生进程所在节点；Pod 可返回当前调度节点                              |
| `startedAt`      | 是   | 启动时间                                                             |
| `uptimeSeconds`  | 是   | 运行时长                                                             |
| `processId`      | 否   | 原生进程 PID，容器模式可不返回                                       |
| `upstreamStatus` | 否   | 采集节点上游连接状态                                                 |
| `walBacklog`     | 否   | 采集节点本地 WAL 待补投条数或批次数                                  |
| `lastError`      | 否   | 最后错误摘要                                                         |
| `reasonCode`     | 否   | 结构化降级或故障原因                                                 |
| `observedAt`     | 是   | 组件生成本次状态的时间                                               |

## 8. 错误约定

- 若进程内部不可恢复，`lifecycleState` 切到 `FAILED`，`healthState` 切到 `UNAVAILABLE`。
- 若部分能力降级但核心能力仍可用，使用 `DEGRADED` 并返回 `reasonCode`。
- `lastError` 只放摘要，不直接塞入大段日志。
- 所有 HTTP 响应遵循 `code/msg/data/reqId`；只有 `code = 0` 表示接口成功。

## 9. 平台消费建议

- NodeAgent 负责原生进程的 `/health` 与 `/api/v1/status`。
- site-controller 负责 K3s 工作负载的探活、聚合与回传。
- 平台展示健康状态、状态新鲜度、版本、最后错误、原因码和启动时间。

## 10. 非目标

- 不定义指标采集协议；主机、K3s、基础服务、工程和 Collector 指标遵循平台运维体系。
- 不定义完整 tracing 或 profiling 接口。
- 不在首版引入复杂告警推送。

## 11. 关联文档

- [平台系统架构](../../01-产品与架构/平台系统架构.md)
- [系统设计文档](../../02-系统设计/README.md)
- [NodeAgent 与工程运行栈启动协议](./node-agent-runtime-protocol.md)
- [测试与质量策略](../../05-研发与交付/测试与质量策略.md)
