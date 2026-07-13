# 采集调试代理协议

## 1. 适用范围

本协议定义数据中心、`data_service` 与采集调试代理之间的开发态短链交互。采集调试代理是用户可选安装的 Windows 托盘程序，不承担工程长期采集、WAL 或运行态 JetStream 发布。

公开协议只使用工业协议和设备能力名称，不暴露底层商业 SDK、供应商、授权方式和内部实现版本。

## 2. Agent 生命周期

```text
未注册 → 使用一次性注册码注册 → 已注册
已注册 → 启动心跳和任务领取 → 在线
在线 → 用户断开中心 → 已断开
在线/已断开 → 托盘退出 → 离线
```

- 关闭状态窗口只隐藏到托盘，不改变在线状态。
- 只有托盘“断开中心”或“退出”才停止任务领取。
- 中心以最后心跳时间判断在线状态，不信任 Agent 上报的客户端时间。

## 3. Agent 能力信封

```json
{
  "agentId": "550e8400-e29b-41d4-a716-446655440000",
  "name": "开发电脑",
  "os": "windows",
  "arch": "x64",
  "version": "0.1.0",
  "protocols": [
    {
      "protocolType": "opcua",
      "capabilityVersion": "1.0",
      "operations": ["connection.test", "opcua.browse", "opcua.read"]
    }
  ]
}
```

约束：

- `protocolType` 是平台稳定标识，不使用 SDK 或供应商名称。
- `operations` 只能包含 Agent 实际可执行的标准操作。
- Agent 版本用于安装包和协议兼容判断，不等同于底层驱动版本。

## 4. 任务请求信封

```json
{
  "taskId": "550e8400-e29b-41d4-a716-446655440010",
  "operation": "opcua.browse",
  "deadlineAt": "2026-07-13 12:00:00",
  "connection": {
    "protocolType": "opcua",
    "endpointUrl": "opc.tcp://127.0.0.1:18540/induforge/sim",
    "securityMode": "None",
    "securityPolicy": "None",
    "authentication": {
      "type": "anonymous"
    }
  },
  "input": {
    "parentNodeId": "ns=0;i=85",
    "maxDepth": 1
  }
}
```

### 4.1 第一阶段操作

| 操作              | 输入                         | 输出                     |
| ----------------- | ---------------------------- | ------------------------ |
| `connection.test` | 空对象                       | 连接耗时、服务端信息     |
| `opcua.browse`    | `parentNodeId`、`maxDepth=1` | 指定父节点的直接子节点   |
| `opcua.read`      | `nodeIds`                    | 批量数据值、质量和时间戳 |

第一阶段不支持长期 Subscription 和 Write。

## 5. 任务结果信封

```json
{
  "taskId": "550e8400-e29b-41d4-a716-446655440010",
  "status": "succeeded",
  "startedAt": "2026-07-13 11:59:58",
  "finishedAt": "2026-07-13 11:59:59",
  "result": {},
  "error": null,
  "diagnostics": []
}
```

任务终态：

- `succeeded`
- `failed`
- `cancelled`
- `expired`

失败结构：

```json
{
  "code": "OPCUA_BAD_CERTIFICATE",
  "message": "服务端证书未受信任",
  "retryable": false
}
```

## 6. OPC UA 节点

```json
{
  "nodeId": "ns=2;s=Plant01.Pressure",
  "browseName": "2:Pressure",
  "displayName": "压力",
  "nodeClass": "variable",
  "dataType": "Double",
  "hasChildren": false
}
```

`nodeClass` 第一阶段使用：

- `object`
- `variable`
- `method`
- `other`

## 7. OPC UA 数据值

```json
{
  "nodeId": "ns=2;s=Plant01.Pressure",
  "value": 0.62,
  "dataType": "Double",
  "quality": "Good",
  "sourceTimestamp": "2026-07-13 12:00:01",
  "serverTimestamp": "2026-07-13 12:00:01"
}
```

- `quality` 保留 OPC UA 状态码的可读名称。
- `value` 使用 JSON 可表达的标准类型；不可直接序列化的值返回字符串表示和诊断信息。
- Read 必须使用单任务批量读取，前端不得逐点创建任务。

## 8. 认证与敏感信息

- 一次性注册码默认 10 分钟过期且只能使用一次。
- Agent Token 仅在注册成功时返回一次，中心只保存 SHA-256 哈希。
- Windows Agent 使用 DPAPI CurrentUser 范围保存 Token。
- 用户名、密码和证书只随单个任务下发，不进入 Agent 普通日志。
- Agent 只能领取明确分配给自己的任务。
- 普通用户错误不得包含 SDK 名称、授权信息、DLL 路径和内部堆栈。

## 9. 版本兼容

- Agent 请求必须携带 Agent 版本和 `capabilityVersion`。
- 第一阶段协议版本固定为 `1.0`。
- 不增加旧结构兼容分支；协议变更直接同步中心、Agent 和数据中心。
