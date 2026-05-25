# OPC UA / Modbus 开发态会话与工作台布局设计

> 日期：2026-05-25  
> 范围：`datacenter/` OPC UA 与 Modbus 工作台布局、`data_service/` 开发态短时会话接口。  
> 目标：让 OPC UA 与 Modbus 工作台具备和 MQTT 工作台一致的开发态连接心智，连接后支持在线浏览、读取、短时订阅或轮询、变量预览，同时保持运行态长期采集仍由节点侧负责。

## 1. 定位

OPC UA / Modbus 工作台的连接能力是开发态短时会话，不是运行态采集器。

开发态会话负责：

- 验证连接配置是否可用。
- 在会话内进行 OPC UA 地址空间浏览、节点读取和短时订阅。
- 在会话内进行 Modbus 寄存器读取、批量读取和短时轮询。
- 为变量导入、变量预览、建模校验和读取计划估算提供现场反馈。
- 在用户断开、页面离开或会话超时后释放连接与监听资源。

开发态会话不负责：

- 发布后的长期采集任务。
- 节点侧断网缓存、补传和长期归档。
- 运行态报警、计算和存储触发。
- 替代 `runtime_data_service` 或 `node_agent` 的运行时职责。

## 2. 现状

当前 MQTT 工作台已经具备开发态连接 / 断开心智：

- 左侧 explorer 内部放置接入源 Header。
- Header 内有连接状态按钮。
- 连接后支持消息监听、变量预览和发布测试。

当前 OPC UA 与 Modbus 工作台仍是：

- 外层全宽 Header + 下方左中右布局。
- Header 内放置测试连接、导入、新建、预览、校验、刷新等操作。
- `PreviewNodes` / `PreviewRegisters` 是辅助预览，不代表持续会话。
- OPC UA / Modbus 没有和 MQTT 等价的开发态会话 ID、连接状态和断开动作。

因此需要同时调整交互布局和后端会话能力，避免前端出现假连接状态。

## 3. 工作台布局

OPC UA 与 Modbus 工作台统一改为只有左中右三栏：

```text
┌──────────── 左侧：接入源 + 分组 ────────────┬──────────── 中间：变量表 ────────────┬──── 右侧：详情 ────┐
│ 返回 / 名称 / Endpoint / 连接状态           │ 当前分组标题 + 上下文操作             │ 变量/分组详情       │
│ [连接] / [断开]                             │ [导入变量] [新建变量] [变量预览]       │ 校验问题            │
│                                            │ [建模校验] [刷新]                     │ 数据点信息          │
│ 分组树                                      │ 搜索变量 + 变量表                      │ 读取计划/诊断       │
└────────────────────────────────────────────┴──────────────────────────────────────┴───────────────────┘
```

左侧职责：

- 放置 `WorkbenchSourceHeader`，视觉结构对齐 MQTT 工作台。
- 展示返回入口、接入源名称、端点、默认从站或安全策略等连接级信息。
- 展示 `连接 / 断开` 按钮和连接状态。
- 展示分组树。

中间职责：

- 展示当前分组名称。
- 展示当前分组变量数量、数据点数量、问题数量。
- 放置变量级和分组级操作：导入变量、新建变量、变量预览、建模校验、刷新。
- Modbus 额外放置读取计划估算入口。
- 展示搜索框和变量表。

右侧职责：

- 展示当前分组或变量详情。
- 展示数据点同步信息。
- 展示当前上下文校验问题。
- Modbus 展示当前分组读取计划摘要。

## 4. 操作作用域

操作作用域应跟随当前上下文，减少误操作。

导入变量：

- 已选分组时导入到当前分组。
- 未选分组时导入到未分组。
- 按钮提示文案显示实际目标，例如“导入到「熔炼炉01」”。

新建变量：

- 默认分组为当前选中分组。
- 未选分组时默认未分组。

变量预览：

- 已选分组时只预览当前分组变量。
- 未选分组时预览全部变量。
- 若后续支持单变量预览，可以从表格行操作进入，不改变顶部按钮的分组级语义。

建模校验：

- 后端可以继续执行全量校验。
- 前端按当前上下文展示结果：选中变量时只显示该变量问题；未选变量但选中分组时只显示该分组内问题；未选分组时显示全部问题。
- 校验抽屉需要显示当前筛选范围，避免用户误以为全局模型已全部通过。

读取计划估算：

- 仅 Modbus 使用。
- 已选分组时估算当前分组。
- 未选分组时估算全部寄存器。

## 5. OPC UA 开发态会话

OPC UA 会话用于开发态短时浏览和预览。

能力：

- 连接 / 断开。
- 浏览地址空间。
- 搜索节点。
- 读取单个或批量节点当前值。
- 对当前分组变量短时订阅。
- 返回值、质量码、SourceTimestamp、ServerTimestamp 和错误信息。
- 导入变量时从浏览树选择节点。

建议接口：

```text
POST   /api/v1/data/projects/{projectId}/opcua/{connectionId}/sessions
DELETE /api/v1/data/projects/{projectId}/opcua/{connectionId}/sessions/{sessionId}
GET    /api/v1/data/projects/{projectId}/opcua/{connectionId}/sessions/{sessionId}/browse
POST   /api/v1/data/projects/{projectId}/opcua/{connectionId}/sessions/{sessionId}/read
POST   /api/v1/data/projects/{projectId}/opcua/{connectionId}/sessions/{sessionId}/subscribe
DELETE /api/v1/data/projects/{projectId}/opcua/{connectionId}/sessions/{sessionId}/subscribe
```

会话创建响应：

```json
{
  "sessionId": "dev-session-id",
  "status": "connected",
  "connectedAt": "2026-05-25 18:30:00",
  "endpoint": "opc.tcp://127.0.0.1:4840",
  "diagnostics": []
}
```

节点读取响应：

```json
{
  "values": [
    {
      "nodeId": "ns=2;s=Line1.Furnace01.Temp",
      "value": 720.5,
      "dataType": "Double",
      "quality": "Good",
      "sourceTimestamp": "2026-05-25 18:30:10",
      "serverTimestamp": "2026-05-25 18:30:10",
      "error": null
    }
  ],
  "diagnostics": []
}
```

订阅结果可以先通过轮询接口返回最近值；后续若接入 WebSocket 或 SSE，应保持响应对象结构一致。

## 6. Modbus 开发态会话

Modbus 会话用于开发态短时读取和轮询。

能力：

- 连接 / 断开。
- 读取单个寄存器。
- 批量读取当前分组寄存器。
- 当前分组短时轮询。
- 返回原始值、解析值、采集时间和错误信息。
- 获取当前分组读取计划估算。

建议接口：

```text
POST   /api/v1/data/projects/{projectId}/modbus/{connectionId}/sessions
DELETE /api/v1/data/projects/{projectId}/modbus/{connectionId}/sessions/{sessionId}
POST   /api/v1/data/projects/{projectId}/modbus/{connectionId}/sessions/{sessionId}/read
POST   /api/v1/data/projects/{projectId}/modbus/{connectionId}/sessions/{sessionId}/poll
DELETE /api/v1/data/projects/{projectId}/modbus/{connectionId}/sessions/{sessionId}/poll
GET    /api/v1/data/projects/{projectId}/modbus/{connectionId}/read-plan
```

寄存器读取响应：

```json
{
  "values": [
    {
      "registerId": "register-id",
      "slaveId": 1,
      "area": "holding",
      "address": 40001,
      "rawValue": [180, 65],
      "value": 23.5,
      "dataType": "float32",
      "timestamp": "2026-05-25 18:30:10",
      "error": null
    }
  ],
  "diagnostics": []
}
```

轮询只用于当前工作台预览，不写入运行态存储，不触发正式报警或计算。

## 7. 前端状态机

每个协议工作台维护独立开发态会话状态：

```text
idle -> connecting -> connected -> disconnecting -> idle
idle -> connecting -> error
connected -> error
```

状态含义：

- `idle`：未连接，可以编辑建模数据，不能在线浏览或实时预览。
- `connecting`：正在创建会话，禁用重复连接。
- `connected`：可浏览、读取、订阅或轮询。
- `disconnecting`：正在释放会话，禁用会话动作。
- `error`：连接失败或会话异常，允许重新连接。

页面离开、切换接入源或组件卸载时，需要主动断开会话。断开失败不阻止页面离开，但需要记录诊断信息。

## 8. 后端会话生命周期

会话生命周期规则：

- 会话 ID 由 `data_service` 生成。
- 会话绑定 `projectId`、`connectionId` 和当前用户。
- 会话只允许当前用户访问。
- 会话有空闲超时，建议默认 10 分钟。
- 会话总时长建议默认 60 分钟，到期自动释放。
- 同一用户同一连接可以复用一个活跃会话，或创建新会话前关闭旧会话。
- 服务停止、连接异常或超时后，会话状态变为 `error` 或自动清理。

资源清理：

- OPC UA 断开时关闭订阅和客户端连接。
- Modbus 断开时停止轮询并关闭 TCP/串口资源。
- 后端需要避免工作台关闭后遗留长时间会话。

## 9. 数据与安全

开发态会话需要遵守现有项目和用户边界：

- 所有接口必须校验项目访问权限。
- 所有连接配置读取必须限定当前项目。
- OPC UA / Modbus 写入能力不在本设计范围内。
- 预览读取不写正式历史库。
- 预览读取不触发正式报警、计算和工作流。
- 诊断信息可返回连接失败、认证失败、NodeId 不存在、寄存器读取失败等原因，但不能泄露密码和敏感证书内容。

## 10. 与数据点建模的关系

开发态会话只增强建模体验，不改变数据点抽象。

OPC UA：

- 浏览树选中的节点可以导入为 `data_opcua_nodes`。
- `data_opcua_nodes` 继续自动同步 `data_points(source_type='opcua.node')`。
- 预览值通过 `nodeId` 或已建模变量 ID 读取。

Modbus：

- 手工创建或批量导入寄存器定义。
- 寄存器定义继续自动同步 `data_points(source_type='modbus.register')`。
- 预览值通过寄存器定义读取和解析。

统一数据点仍然是页面、计算、报警、发布契约的消费对象。

## 11. 验证

静态验证：

- `pnpm --filter datacenter typecheck`
- `pnpm --filter datacenter build`
- `go test ./...` 或后端最贴近会话能力的 `data_service` 测试

手动验收：

- OPC UA 工作台无全宽 Header，左侧 Header 与分组树同列。
- Modbus 工作台无全宽 Header，左侧 Header 与分组树同列。
- 未连接时在线浏览、短时订阅或轮询入口不可用。
- 点击连接后显示已连接状态，断开后回到未连接状态。
- OPC UA 连接后可以浏览节点、读取当前值并导入变量。
- Modbus 连接后可以读取当前分组寄存器并短时轮询。
- 变量预览只针对当前分组。
- 建模校验按当前变量、当前分组或全部范围展示。
- 离开工作台时会话被释放。

