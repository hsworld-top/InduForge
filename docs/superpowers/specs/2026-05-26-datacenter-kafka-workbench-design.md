# 数据中心 Kafka 工作台设计

> 日期：2026-05-26  
> 范围：`datacenter/` Kafka 接入源专属工作台、`data_service/` Kafka 开发态预览与字段映射接口。  
> 目标：让 Kafka 接入源从通用协议预览升级为家族化工作台，支持一个接入源管理多个 Topic，完整提供 Topic 管理、短时抓样、字段映射和数据点建模闭环。

## 1. 定位

Kafka 工作台是开发态配置和前期预览工具，不是长期运行态消费器。

开发态负责：

- 管理 Kafka 接入源下的多个 Topic 映射。
- 测试 Kafka 配置和 Topic 可读性。
- 使用短时 reader 抓取样本消息。
- 从样本中推断字段路径，并创建 Kafka 字段数据点。
- 记录预览摘要和诊断信息，辅助建模验收。

开发态不负责：

- 长期消费任务。
- 正式 offset 提交、断点续消或消费组运维。
- 正式历史入库。
- 触发运行态计算、报警或工作流。
- 替代节点侧或运行态采集器能力。

## 2. 当前现状

当前仓库中 Kafka 已有基础配置和短时预览能力：

- `data_kafka_configs` 保存单连接的 brokers、topic、consumerGroup、startPosition 和 options。
- `/api/v1/data/projects/{projectId}/kafka/configs` 可创建 Kafka 配置。
- 统一协议预览已支持 `kafka-go` 临时 reader，按 `limit` 和 `timeoutMs` 抓取样本。
- `datacenter` 当前把 `kafka/http/websocket` 统一渲染到 `ProtocolWorkbenchPanel.vue`。

现状缺口：

- Kafka 仍是通用协议面板，缺少独立工作台心智。
- 一个接入源只暴露单个默认 topic，不能管理多个业务 Topic。
- 只有抓样，没有 Topic 映射、字段映射和数据点建模闭环。
- UI 未对齐 MQTT 工作台的左树、中间页签、右侧详情结构。

## 3. 核心心智

Kafka 工作台与 MQTT 工作台保持家族化，但不硬套 MQTT 术语。

映射关系：

- Kafka 接入源 ≈ MQTT Broker。
- Kafka Topic 映射 ≈ MQTT 订阅。
- Kafka 字段映射 ≈ MQTT Tag。

一个 Kafka 接入源表示一个 Kafka 集群或接入入口，可以管理多个项目内 Topic 映射。Topic 映射是项目配置，不等同于创建或修改 Kafka Broker 上的真实 Topic。

## 4. 成功标准

- Kafka 接入源打开后进入独立 Kafka 工作台，不再使用通用协议面板。
- 一个 Kafka 接入源可维护多个 Topic 映射，并支持分组管理。
- 左侧 explorer、连接状态按钮、页签、中间消息流、右侧 inspector 与 MQTT 工作台家族化。
- 选中 Topic 映射后可按 partition、offset 策略执行短时抓样。
- 抓样结果展示 key、value、headers、partition、offset、timestamp。
- JSON 样本可推断字段路径，用户可勾选字段创建 `kafka.field` 数据点。
- 离开工作台、切换接入源或停止预览后，不保留长期 consumer。
- 预览只保存摘要记录，不保存原始消息。
- `pnpm --dir datacenter build` 通过；如修改 `data_service`，运行最贴近变更面的 Go 测试。

## 5. 不做范围

- 不实现长期 Kafka 消费任务。
- 不实现正式 offset 提交、消费组 rebalance 管理或 lag 运维大盘。
- 不创建、删除或修改真实 Kafka Topic。
- 不支持 Kafka 消息写入或生产测试。
- 不支持 Avro、Protobuf、Schema Registry 的完整解析链路。
- 不把 Kafka 字段映射抽象成跨所有协议的通用建模框架。

## 6. 信息架构

Kafka 工作台采用左中右结构：

```text
┌──────── 左侧：Kafka 接入源 / Topic ────────┬──────── 中间：Topic 工作区 ─────────┬──── 右侧：详情/诊断 ────┐
│ ‹ 返回                                      │ [消息预览] [字段映射] [消费位点]      │ 接入源摘要              │
│ Kafka 事件总线                              │                                      │ brokers                │
│ 状态：[连接] / 已连接 hover 断开             │ 消息预览工具栏：                     │ 安全协议 / SASL / TLS    │
│ brokers: kafka-01:9092,kafka-02:9092       │ [Topic] [Partition] [Offset模式]     │ preview group           │
│                                             │ [获取样本] [监听30s] [清空] [搜索]    │                         │
│ Topic 管理                                  │                                      │ 当前 Topic              │
│ [搜索 Topic] [+ 新建 Topic 映射] [刷新]      │ ┌────────────────────────────────┐   │ partitions: 6           │
│                                             │ │ partition 0 · offset 18392      │   │ latest offset: 18420     │
│ ▸ 设备遥测                                  │ │ key: line-1                     │   │ 本次样本: 100            │
│   device.telemetry                          │ │ value: {                        │   │                         │
│ ▸ 报警事件                                  │ │   "temperature": 72.5,          │   │ 样本结构                │
│   alarm.events                              │ │   "status": "RUNNING"           │   │ JSON · 12字段            │
│ ▸ 生产订单                                  │ │ }                              │   │                         │
│   production.order                          │ └────────────────────────────────┘   │ 快捷操作                │
│                                             │                                      │ [创建字段映射]           │
│ 分组                                        │ 字段映射页：                         │ [复制样本]              │
│  默认分组                                   │ temperature  number → 数据点          │ [查看错误]              │
│  产线 A                                     │ status       string → 数据点          │                         │
└─────────────────────────────────────────────┴──────────────────────────────────────┴─────────────────────────┘
```

### 6.1 左侧 explorer

左侧职责：

- 放置 `WorkbenchSourceHeader`，展示返回、名称、连接态、连接和断开操作。
- 展示 Kafka brokers、默认安全协议和默认 preview group。
- 展示 Topic 分组树和 Topic 映射。
- 提供 Topic 搜索、新建映射、新建分组、刷新元数据入口。

Topic 分组只是项目内管理结构，不映射 Kafka Broker 层级。

### 6.2 中间工作区

中间按 Topic 映射打开页签：

- `Topic 管理`：默认页签，管理分组与 Topic 映射。
- `{topic} / 消息预览`：展示短时抓样结果。
- `{topic} / 字段映射`：从最近样本推断字段并创建数据点。

页签 id：

- Topic 管理：`kafka-topics-${connection.id}`
- 消息预览：`kafka-preview-${mapping.id}`
- 字段映射：`kafka-fields-${mapping.id}`

### 6.3 右侧 inspector

右侧职责：

- 展示接入源摘要：brokers、安全协议、认证模式。
- 展示当前 Topic 摘要：partition 数、最近 offset、最近抓样数量。
- 展示样本结构：JSON、String、Binary、字段数。
- 展示最近错误：连接失败、鉴权失败、Topic 不存在、读取超时。
- 提供快捷操作：创建字段映射、复制样本、刷新 Topic 元数据。

## 7. 领域模型

### 7.1 Topic 映射

Topic 映射是项目内配置，用于声明当前 Kafka 接入源下需要管理的业务 Topic。

字段：

```json
{
  "id": "mapping-id",
  "projectId": "project-id",
  "connectionId": "kafka-connection-id",
  "groupId": "group-id",
  "name": "设备遥测",
  "topic": "device.telemetry",
  "description": "产线设备遥测事件",
  "partitionMode": "all",
  "partition": null,
  "startPosition": "latest",
  "decode": "json",
  "sampleLimit": 100,
  "timeoutMs": 5000,
  "createdAt": "2026-05-26 10:00:00",
  "updatedAt": "2026-05-26 10:00:00"
}
```

字段约束：

- `topic` 必填，保存真实 Kafka Topic 名称。
- `partitionMode` 支持 `all`、`single`。
- `startPosition` 支持 `latest`、`earliest`、`offset`。
- `decode` 支持 `json`、`string`、`binary`。
- `sampleLimit` 默认 100，最大 1000。
- `timeoutMs` 默认 5000，最大 30000。

### 7.2 Topic 分组

Topic 分组用于左侧树管理，行为对齐 MQTT 订阅分组。

字段：

```json
{
  "id": "group-id",
  "projectId": "project-id",
  "connectionId": "kafka-connection-id",
  "parentId": null,
  "name": "产线 A",
  "sortOrder": 10
}
```

### 7.3 字段映射

字段映射描述从 Kafka 消息中提取一个字段并同步成数据点。

字段：

```json
{
  "id": "field-id",
  "projectId": "project-id",
  "connectionId": "kafka-connection-id",
  "topicMappingId": "mapping-id",
  "name": "temperature",
  "valuePath": "temperature",
  "keyPath": "",
  "dataType": "number",
  "enabled": true,
  "description": "设备温度"
}
```

字段映射同步生成数据点：

```json
{
  "sourceType": "kafka.field",
  "sourceId": "kafka-connection-id",
  "sourceConfig": {
    "topicMappingId": "mapping-id",
    "topic": "device.telemetry",
    "partition": null,
    "decode": "json",
    "keyPath": "",
    "valuePath": "temperature"
  }
}
```

### 7.4 预览样本

预览样本是内存态开发数据，只在本次工作台会话中展示。服务端只保存预览摘要，不保存原始消息。

样本结构：

```json
{
  "id": "connection-topic-partition-offset",
  "topic": "device.telemetry",
  "partition": 0,
  "offset": 18392,
  "key": "line-1",
  "headers": {
    "traceId": "abc"
  },
  "value": {
    "temperature": 72.5,
    "status": "RUNNING"
  },
  "rawPayload": "{\"temperature\":72.5,\"status\":\"RUNNING\"}",
  "decode": "json",
  "timestamp": "2026-05-26 10:10:10"
}
```

字段说明：

- `value` 是按 `decode` 解析后的消息体。
- `rawPayload` 只返回给当前预览响应，不写入接入源记录。
- `headers` 需要过滤敏感字段。
- `id` 用于前端列表稳定渲染，不需要入库。

## 8. 后端接口

Kafka 专属工作台新增接口，不继续把 Topic 管理和字段映射堆在通用 `protocols/{connectionId}/preview` 上。

Topic 元数据：

```text
GET /api/v1/data/projects/{projectId}/kafka/{connectionId}/topics
```

返回 broker 上可见 Topic 摘要。若集群禁用元数据列举或数量过大，接口可返回空列表和诊断，前端仍允许手动创建 Topic 映射。

Topic 分组：

```text
GET    /api/v1/data/projects/{projectId}/kafka/{connectionId}/topic-groups
POST   /api/v1/data/projects/{projectId}/kafka/{connectionId}/topic-groups
PUT    /api/v1/data/projects/{projectId}/kafka/topic-groups/{groupId}
DELETE /api/v1/data/projects/{projectId}/kafka/topic-groups/{groupId}
```

Topic 映射：

```text
GET    /api/v1/data/projects/{projectId}/kafka/{connectionId}/topic-mappings
POST   /api/v1/data/projects/{projectId}/kafka/{connectionId}/topic-mappings
GET    /api/v1/data/projects/{projectId}/kafka/topic-mappings/{mappingId}
PUT    /api/v1/data/projects/{projectId}/kafka/topic-mappings/{mappingId}
DELETE /api/v1/data/projects/{projectId}/kafka/topic-mappings/{mappingId}
```

短时预览：

```text
POST /api/v1/data/projects/{projectId}/kafka/{connectionId}/preview
POST /api/v1/data/projects/{projectId}/kafka/topic-mappings/{mappingId}/preview
```

预览请求：

```json
{
  "topic": "device.telemetry",
  "partitionMode": "all",
  "partition": null,
  "startPosition": "latest",
  "offset": null,
  "limit": 100,
  "timeoutMs": 5000,
  "decode": "json"
}
```

预览响应：

```json
{
  "status": "ok",
  "topic": "device.telemetry",
  "samples": [
    {
      "topic": "device.telemetry",
      "partition": 0,
      "offset": 18392,
      "key": "line-1",
      "headers": { "traceId": "abc" },
      "value": { "temperature": 72.5, "status": "RUNNING" },
      "rawPayload": "{\"temperature\":72.5,\"status\":\"RUNNING\"}",
      "timestamp": "2026-05-26 10:10:10"
    }
  ],
  "schema": {
    "decode": "json",
    "fields": [
      { "path": "temperature", "type": "number" },
      { "path": "status", "type": "string" }
    ]
  },
  "diagnostics": {
    "sampleCount": 1,
    "durationMs": 210,
    "truncated": false
  }
}
```

字段映射：

```text
GET    /api/v1/data/projects/{projectId}/kafka/topic-mappings/{mappingId}/fields
POST   /api/v1/data/projects/{projectId}/kafka/topic-mappings/{mappingId}/fields
POST   /api/v1/data/projects/{projectId}/kafka/topic-mappings/{mappingId}/fields/batch
PUT    /api/v1/data/projects/{projectId}/kafka/fields/{fieldId}
DELETE /api/v1/data/projects/{projectId}/kafka/fields/{fieldId}
PATCH  /api/v1/data/projects/{projectId}/kafka/fields/{fieldId}/toggle
```

## 9. 数据表

新增表：

- `data_kafka_topic_groups`
- `data_kafka_topic_mappings`
- `data_kafka_fields`

`data_kafka_topic_mappings` 与 `data_connections` 多对一。`data_kafka_fields` 与 topic mapping 多对一，并同步 `data_points(source_type='kafka.field')`。

### 9.1 `data_kafka_topic_groups`

保存 Kafka 工作台左侧 Topic 分组。

核心字段：

- `id uuid primary key`
- `project_id uuid not null`
- `connection_id uuid not null`
- `parent_id uuid null`
- `name text not null`
- `sort_order integer not null default 0`
- `created_at timestamptz not null default now()`
- `updated_at timestamptz not null default now()`

约束：

- `connection_id` 级联引用 `data_connections(id)`。
- `parent_id` 级联引用本表 `id`，删除分组时子分组和 Topic 映射回到根目录，或要求前端确认后由服务端置空。
- 同一连接同级分组名称唯一。

### 9.2 `data_kafka_topic_mappings`

保存项目内 Kafka Topic 映射。

核心字段：

- `id uuid primary key`
- `project_id uuid not null`
- `connection_id uuid not null`
- `group_id uuid null`
- `name text not null`
- `topic text not null`
- `description text not null default ''`
- `partition_mode text not null default 'all'`
- `partition integer null`
- `start_position text not null default 'latest'`
- `decode text not null default 'json'`
- `sample_limit integer not null default 100`
- `timeout_ms integer not null default 5000`
- `sort_order integer not null default 0`
- `created_by uuid not null`
- `updated_by uuid not null`
- `created_at timestamptz not null default now()`
- `updated_at timestamptz not null default now()`

约束：

- `partition_mode in ('all', 'single')`
- `start_position in ('latest', 'earliest', 'offset')`
- `decode in ('json', 'string', 'binary')`
- `sample_limit between 1 and 1000`
- `timeout_ms between 1000 and 30000`
- `partition_mode = 'single'` 时 `partition` 必须非空且大于等于 0。

### 9.3 `data_kafka_fields`

保存 Kafka 字段映射，并驱动数据点同步。

核心字段：

- `id uuid primary key`
- `project_id uuid not null`
- `connection_id uuid not null`
- `topic_mapping_id uuid not null`
- `name text not null`
- `value_path text not null`
- `key_path text not null default ''`
- `data_type text not null`
- `enabled boolean not null default true`
- `description text not null default ''`
- `sort_order integer not null default 0`
- `created_by uuid not null`
- `updated_by uuid not null`
- `created_at timestamptz not null default now()`
- `updated_at timestamptz not null default now()`

约束：

- `data_type` 复用现有数据点类型枚举。
- 同一 Topic 映射下 `value_path` 唯一。
- 删除字段映射时，同步删除或标记失效对应 `kafka.field` 数据点，具体行为保持与现有 MQTT Tag 数据点同步策略一致。

索引：

- `data_kafka_topic_mappings(project_id, connection_id, topic)` 唯一。
- `data_kafka_fields(project_id, topic_mapping_id, value_path)` 唯一。
- `data_kafka_fields(project_id, connection_id)` 普通索引。

## 10. 前端结构

新增：

- `datacenter/src/components/access-source/workbench/KafkaWorkbenchPanel.vue`
- `datacenter/src/components/kafka/KafkaWorkbench.vue`
- `datacenter/src/components/kafka/KafkaTopicTreeBranch.vue`
- `datacenter/src/components/kafka/KafkaTopicMappingDialog.vue`
- `datacenter/src/components/kafka/KafkaFieldMappingPanel.vue`
- `datacenter/src/components/kafka/KafkaPreviewPanel.vue`
- `datacenter/src/components/kafka/kafkaTopicTreeModel.ts`

复用：

- `WorkbenchSourceHeader`
- `WorkbenchStreamToolbar`
- `WorkbenchStreamMessageList`
- `WorkbenchStatusPill`
- `DcDialog`
- `getApiErrorMessage`

`AccessSourceWorkbench.vue` 中 `type === 'kafka'` 改为渲染 `KafkaWorkbenchPanel`。

### 10.1 Kafka 工作台组件职责

`KafkaWorkbenchPanel.vue`：

- 作为接入源工作台分发壳，接收 `projectId`、`connection` 和 `back` 事件。
- 挂载 `KafkaWorkbench.vue`。

`KafkaWorkbench.vue`：

- 维护连接状态、Topic 分组树、Topic 映射、页签和当前选中上下文。
- 统一处理工作台卸载、切换接入源和关闭页签时的预览清理。
- 负责打开 Topic 管理、消息预览、字段映射和详情弹窗。

`KafkaPreviewPanel.vue`：

- 接收 Topic 映射和预览参数。
- 调用 Kafka 预览接口。
- 通过 `WorkbenchStreamToolbar` 和 `WorkbenchStreamMessageList` 展示消息。
- 把最近样本和 schema 回传给工作台，供字段映射使用。

`KafkaFieldMappingPanel.vue`：

- 展示已存在字段映射。
- 展示从最近样本推断出的候选字段。
- 支持创建、批量创建、编辑、启停和删除字段映射。
- 操作成功后刷新数据点同步状态。

`KafkaTopicMappingDialog.vue`：

- 创建和编辑 Topic 映射。
- 支持手动输入 Topic，也支持从 broker 元数据发现结果中选择 Topic。
- 校验 partition、offset、decode、limit、timeout 等字段。

## 11. 数据流

### 11.1 打开工作台

1. 用户从接入源卡片点击 Kafka 工作台。
2. `AccessSourceWorkbench` 渲染 `KafkaWorkbenchPanel`。
3. `KafkaWorkbench` 加载 Topic 分组与 Topic 映射。
4. 默认打开 `Topic 管理` 页签。

### 11.2 短时预览

1. 用户选择 Topic 映射并点击“消息预览”。
2. 工作台根据映射生成预览参数。
3. 后端创建临时 reader，读取有限样本。
4. 后端关闭 reader，不提交正式 offset。
5. 前端展示消息流和 schema。
6. 后端写入不含原始消息的预览摘要记录。

### 11.3 字段映射

1. 用户在消息预览得到 JSON 样本。
2. 前端展示后端返回或本地推断的字段路径。
3. 用户勾选字段并确认创建。
4. 后端写入 `data_kafka_fields`。
5. 后端同步生成或更新 `data_points(source_type='kafka.field')`。
6. 数据点工作区可按 `sourceId` 或 `sourceType` 查看 Kafka 字段数据点。

### 11.4 连接状态

1. 用户点击左侧 Header 的“连接”。
2. 前端调用 Kafka 连接测试或元数据接口验证 brokers 可达。
3. 成功后工作台进入 `connected`，允许执行 Topic 元数据刷新和短时预览。
4. 用户点击“断开”时，前端清空当前短时预览任务状态，后端没有长期连接需要释放。
5. 页面卸载时执行同样清理逻辑。

状态机：

```text
idle -> connecting -> connected
idle -> connecting -> error
connected -> previewing -> connected
connected -> disconnecting -> idle
connected -> error
error -> connecting
```

### 11.5 删除 Topic 映射

1. 用户删除 Topic 映射。
2. 前端提示该映射下字段和数据点会受影响。
3. 用户确认后，后端删除 Topic 映射、字段映射，并按数据点同步策略清理对应 `kafka.field` 数据点。
4. 前端关闭该映射相关消息预览和字段映射页签。
5. 左侧树刷新。

## 12. 完整交付范围

本设计作为一次完整功能交付，不拆分阶段。交付完成时应同时具备以下能力：

- Kafka 接入源进入专属工作台。
- Topic 分组管理完整可用。
- Topic 映射创建、编辑、删除、移动分组完整可用。
- Broker Topic 元数据发现可用；不可用时支持手动 Topic 映射。
- Kafka 连接测试和工作台连接状态可用。
- Topic 短时预览可按 partition、offset 策略抓样。
- 消息流展示 key、value、headers、partition、offset、timestamp。
- 最近样本 schema 推断可用。
- 字段映射 CRUD、启停和批量创建可用。
- 字段映射同步 `kafka.field` 数据点。
- 接入源记录只保存预览摘要，不保存原始消息。
- 切换接入源、关闭页签和卸载工作台时不会遗留消费任务。

## 13. 错误处理

- Kafka 配置缺失 brokers：阻止连接和预览，提示配置不完整。
- Topic 不存在：预览失败，保留 Topic 映射，右侧展示诊断。
- 鉴权失败：隐藏敏感认证字段，只展示认证失败摘要。
- 预览超时：返回已读取样本；若没有样本，展示空态和超时诊断。
- JSON 解析失败：按 string 显示 raw payload，字段映射页提示当前样本无法推断 JSON 字段。
- 字段路径重复：复用已有数据点或提示字段已存在，避免重复创建。
- 删除 Topic 映射：同步删除字段映射或要求用户确认影响，删除后关闭相关页签。

## 14. 验证

静态验证：

- `pnpm --dir datacenter build`
- 修改前端类型或 store 时运行相关 `vitest`
- 修改 `data_service` 时运行 Kafka 预览、字段映射、数据点同步相关 Go 测试

手动验收：

- Kafka 接入源进入独立工作台。
- 一个接入源下能创建多个 Topic 映射。
- Topic 映射能分组、重命名、删除。
- 选中 Topic 后能短时获取样本。
- 样本能展示 key、value、headers、partition、offset、timestamp。
- JSON 样本能推断字段路径。
- 勾选字段能创建 Kafka 字段数据点。
- 离开工作台后没有持续消费行为。

## 15. 风险与控制

- Kafka Topic 数量可能很大。控制方式：Topic 元数据只做发现入口，项目内工作台只管理用户加入的 Topic 映射。
- 用户可能误解字段数据点等于已长期采集。控制方式：工作台文案和数据点详情明确标注“开发态建模，运行态采集另行发布生效”。
- Kafka 安全配置差异较多。控制方式：复用连接配置中的 options，不扩展完整安全配置编辑器。
- JSON 字段推断可能不稳定。控制方式：字段映射保存显式 `valuePath` 和 `dataType`，允许用户手动调整。
- 预览读取 latest 时可能无消息。控制方式：提供 earliest 和指定 offset 选项，并展示超时诊断。
