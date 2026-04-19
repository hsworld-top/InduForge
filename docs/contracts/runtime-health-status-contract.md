# Runtime 健康检查与状态协议

## 1. 文档定位
- 本文档定义 RuntimeEngine 对 NodeAgent 和平台暴露的最小健康与状态协议。
- 本协议的目标是支撑托管、探活、错误定位和运维展示。

## 2. 当前已实现
### 当前缺口
- RuntimeEngine 尚未实现，因此 `/health` 和 `/status` 目前没有事实接口。

## 3. 协议目标
- Runtime 必须提供轻量、稳定、低歧义的运维接口。
- NodeAgent 和 `dev_ide` 应基于该协议做探活和展示。

## 4. `GET /health`
### 目标
- 用于快速判断进程是否可服务。

### 成功响应
```json
{
  "success": true,
  "data": {
    "status": "UP"
  }
}
```

### 约束
- 不返回重业务数据。
- 响应应足够轻量，适合高频调用。

## 5. `GET /status`
### 目标
- 返回当前运行版本、项目、启动时间、运行状态和最后错误。

### 建议响应
```json
{
  "success": true,
  "data": {
    "runtimeStatus": "RUNNING",
    "projectId": "proj_xxx",
    "projectCode": "factory_dashboard",
    "version": "2026.03.06-001",
    "startedAt": "2026-03-06T10:05:00Z",
    "uptimeSeconds": 3600,
    "lastError": null
  }
}
```

## 6. 状态枚举
- `STARTING`
- `RUNNING`
- `DEGRADED`
- `FAILED`
- `STOPPED`

## 7. 字段说明
| 字段 | 必填 | 说明 |
| --- | --- | --- |
| `runtimeStatus` | 是 | 运行状态 |
| `projectId` | 是 | 工程 ID |
| `projectCode` | 是 | 工程编码 |
| `version` | 是 | 当前运行版本 |
| `startedAt` | 是 | 启动时间 |
| `uptimeSeconds` | 是 | 运行时长 |
| `lastError` | 否 | 最后错误摘要 |

## 8. 错误约定
- 若 Runtime 内部不可恢复，`runtimeStatus` 必须切到 `FAILED`。
- 若部分能力降级但页面仍可访问，可使用 `DEGRADED`。
- `lastError` 只放摘要，不直接塞入大段日志。

## 9. 平台消费建议
- NodeAgent 高频调用 `/health`。
- NodeAgent 低频调用 `/status`，并回传平台。
- `dev_ide` 展示 `runtimeStatus`、版本号、最后错误和启动时间。

## 10. 非目标
- 不定义指标采集协议。
- 不定义完整 tracing 或 profiling 接口。
- 不在首版引入复杂告警推送。

## 11. 关联文档
- [高层设计](../高层设计.md)
- [详细设计](../详细设计.md)
- [NodeAgent 与 RuntimeEngine 启动协议](./node-agent-runtime-protocol.md)
- [测试与质量策略](../测试与质量策略.md)
