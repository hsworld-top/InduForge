# MQTT 实时数据源实现方案

本文档描述 DataCenter 中 MQTT 功能的实现方案，支持主题订阅和变量管理两种模式。

## 核心目标

1. **MQTT 连接管理**：配置和管理 MQTT Broker 连接
2. **主题订阅**：直接订阅 MQTT 主题，获取原始消息
3. **变量管理**：从 MQTT 消息中解析出变量（值、时间戳、质量戳）
4. **实时推送**：通过 WebSocket 推送实时数据到前端和 Designer
5. **Designer 集成**：支持两种数据绑定模式

## 数据源模式对比

### 主题订阅模式
- **绑定对象**：MQTT Topic（如 `sensor/+/data`）
- **数据格式**：原始消息（JSON/文本/二进制）
- **使用场景**：需要灵活处理复杂消息、自定义业务逻辑
- **配置位置**：在 Designer 中配置解析规则
- **优势**：灵活、可自定义处理
- **劣势**：每个组件需要独立解析

### 变量订阅模式
- **绑定对象**：Tag（变量，如 `temperature_sensor_01`）
- **数据格式**：已解析的值 + 时间戳 + 质量戳
- **使用场景**：常规数据展示、图表绑定
- **配置位置**：在 DataCenter 中预配置解析规则
- **优势**：简单直接、性能好、后端统一解析
- **劣势**：需要预先配置变量

## 数据模型设计

### 统一使用 data_tags 表

复用现有的 `data_tags` 和 `data_tag_groups` 表，支持 MQTT、OPC UA、Modbus 等多种协议。

**关键字段映射**：
- `address`：存储 MQTT Topic（支持通配符 `+` 和 `#`）
- `dataType`：数据类型（float、int32、string、bool、json 等）
- `accessMode`：read（订阅）、write（发布）、read_write
- `metadata`：JSON 字段存储 MQTT 特定配置

**metadata 结构示例**：
```json
{
  "protocol": "mqtt",
  "qos": 1,
  "retain": false,
  "parseRule": {
    "format": "json",
    "valuePath": "$.temperature",
    "timestampPath": "$.timestamp",
    "qualityPath": "$.quality"
  }
}
```

### 可选：data_mqtt_subscriptions 表

用于管理 Designer 中直接订阅的主题（原始消息模式）。

**字段**：
- `projectId`：所属工程
- `connectionId`：MQTT 连接
- `name`：订阅名称
- `topic`：MQTT 主题
- `qos`：QoS 等级
- `isEnabled`：是否启用

## 架构设计

### 后端架构

**协议层**：
- `BaseProtocol.js`：协议基类
- `MqttProtocol.js`：MQTT 协议实现（连接、订阅、发布、断开）

**服务层**：
- `mqttService.js`：MQTT 连接管理、消息分发
- `tagService.js`：统一的 Tag 管理服务
- `tagParserService.js`：消息解析引擎（支持 JSON、文本、二进制、脚本）

**控制器层**：
- `mqttConnectionController.js`：MQTT 连接 CRUD
- `tagController.js`：Tag CRUD
- `mqttSubscriptionController.js`：主题订阅管理

### 前端架构

**组件结构**：
- `mqtt/MqttConnectionForm.vue`：MQTT 连接配置表单
- `mqtt/MqttSubscriptionList.vue`：主题订阅列表
- `mqtt/MqttMessageViewer.vue`：实时消息查看器
- `tags/TagGroupTree.vue`：Tag 分组树
- `tags/TagList.vue`：Tag 列表
- `tags/TagEditor.vue`：Tag 编辑器（含 MQTT 解析规则配置）
- `tags/TagMonitor.vue`：实时 Tag 监控

**Composables**：
- `useMqttConnection.js`：MQTT 连接管理
- `useMqttSubscription.js`：主题订阅管理
- `useTag.js`：Tag 管理
- `useTagValue.js`：Tag 值订阅（WebSocket）
- `useMqttRealtime.js`：实时消息接收

## 消息解析引擎

### 支持的解析格式

#### 1. JSON 格式
使用 JSONPath 表达式提取值。

**配置示例**：
- 值路径：`$.temperature`
- 时间戳路径：`$.timestamp`
- 质量戳路径：`$.quality`

#### 2. 文本格式（分隔符）
使用分隔符和索引提取值。

**配置示例**：
- 分隔符：`,`
- 值索引：`1`
- 时间戳索引：`3`

#### 3. 二进制格式
按字节偏移和数据类型解析。

**配置示例**：
- 字节偏移：`0`
- 数据类型：`float32`
- 字节序：`big-endian`

#### 4. 自定义脚本
使用 JavaScript 脚本自定义解析逻辑（沙箱环境）。

**脚本示例**：
```javascript
function parse(message) {
  const data = JSON.parse(message);
  return {
    value: data.temp * 0.1 + 32,
    timestamp: data.ts,
    quality: data.temp > 0 ? 'good' : 'bad'
  };
}
```

## 数据回写功能

### 回写场景

MQTT 和 OPC UA 等协议需要支持数据回写，用于控制设备或发送指令。

**典型场景**：
- 控制设备开关（发送控制指令）
- 设置设备参数（修改配置值）
- 发送报警确认（确认消息）
- 远程控制（执行命令）

### 回写模式

#### 模式 1：变量回写

直接修改变量的值，系统自动构造消息并发布到对应的 Topic。

**适用场景**：
- 简单的值修改
- 标准化的控制指令
- 单个变量更新

**工作流程**：
```
Designer 组件
  ↓ 触发回写
HTTP POST /api/v1/data/tags/:tagId/write
  ↓ 传递新值
TagService.writeValue()
  ↓ 根据回写配置构造消息
MqttProtocol.publish()
  ↓ 发布到 MQTT Broker
设备接收并执行
```

**Tag 回写配置**：
```json
{
  "name": "device_switch",
  "address": "device/001/control",
  "accessMode": "read_write",
  "metadata": {
    "protocol": "mqtt",
    "writeConfig": {
      "topic": "device/001/control",
      "qos": 1,
      "retain": false,
      "format": "json",
      "template": {
        "command": "set_switch",
        "value": "{{value}}",
        "timestamp": "{{timestamp}}"
      }
    }
  }
}
```

**回写消息示例**：
```json
{
  "command": "set_switch",
  "value": 1,
  "timestamp": 1702886400000
}
```

#### 模式 2：消息发布

自定义完整的消息内容，灵活控制发布的数据格式。

**适用场景**：
- 复杂的控制指令
- 批量参数设置
- 自定义消息格式
- 多变量联动

**工作流程**：
```
Designer 组件
  ↓ 触发发布
HTTP POST /api/v1/data/mqtt/publish
  ↓ 传递完整消息
MqttService.publish()
  ↓ 发布到 MQTT Broker
设备接收并执行
```

**发布配置**：
```json
{
  "connectionId": "mqtt-conn-uuid",
  "topic": "device/001/batch-control",
  "qos": 1,
  "retain": false,
  "payload": {
    "command": "batch_set",
    "parameters": {
      "temperature": 25,
      "humidity": 60,
      "mode": "auto"
    },
    "timestamp": 1702886400000
  }
}
```

### 回写配置

#### Tag 回写配置结构

在 Tag 的 `metadata.writeConfig` 中配置回写规则：

```json
{
  "writeConfig": {
    "topic": "device/001/control",
    "qos": 1,
    "retain": false,
    "payloadTemplate": "{\"command\":\"set_value\",\"value\":{{value}},\"timestamp\":{{timestamp}}}",
    "validation": {
      "min": 0,
      "max": 100,
      "required": true
    },
    "confirmation": true
  }
}
```

**配置说明**：
- `topic`：发布的 MQTT 主题（必填）
- `qos`：QoS 等级（0/1/2）
- `retain`：是否保留消息
- `payloadTemplate`：**用户自定义的消息模板**（必填）
- `validation`：值验证规则（可选）
- `confirmation`：是否需要确认（可选）

**重要说明**：
- `payloadTemplate` 是**完全由用户自定义**的字符串模板
- 用户可以定义任意格式的消息（JSON、文本、XML 等）
- 系统只负责替换模板中的变量占位符
- 不限制消息结构和字段

#### 模板变量

系统支持以下内置变量，用户可以在模板中使用：

| 变量 | 说明 | 示例值 |
|------|------|--------|
| `{{value}}` | 回写的值 | `25.5` |
| `{{timestamp}}` | 当前时间戳（毫秒） | `1702886400000` |
| `{{datetime}}` | 当前日期时间（ISO 8601） | `"2024-12-15T10:30:45.123Z"` |
| `{{user}}` | 操作用户名 | `"admin"` |
| `{{userId}}` | 操作用户 ID | `"user-uuid"` |
| `{{tagName}}` | 变量名 | `"temperature_setpoint"` |
| `{{tagId}}` | 变量 ID | `"tag-uuid"` |
| `{{oldValue}}` | 原值 | `20` |

**变量格式化**（可选）：
- `{{value:json}}` - JSON 格式化（字符串加引号）
- `{{value:hex}}` - 十六进制格式
- `{{value:int}}` - 整数格式
- `{{timestamp:iso}}` - ISO 8601 日期格式

#### 用户自定义格式示例

**示例 1：简单 JSON 格式**
```json
{
  "payloadTemplate": "{\"cmd\":\"write\",\"val\":{{value}}}"
}
```
发布消息：`{"cmd":"write","val":25.5}`

**示例 2：完整 JSON 格式**
```json
{
  "payloadTemplate": "{\"command\":\"set_value\",\"deviceId\":\"device001\",\"parameter\":\"temperature\",\"value\":{{value}},\"timestamp\":{{timestamp}},\"user\":\"{{user}}\"}"
}
```
发布消息：
```json
{
  "command": "set_value",
  "deviceId": "device001",
  "parameter": "temperature",
  "value": 25.5,
  "timestamp": 1702886400000,
  "user": "admin"
}
```

**示例 3：文本格式（CSV）**
```json
{
  "payloadTemplate": "SET,{{tagName}},{{value}},{{timestamp}}"
}
```
发布消息：`SET,temperature_setpoint,25.5,1702886400000`

**示例 4：文本格式（键值对）**
```json
{
  "payloadTemplate": "cmd=write&tag={{tagName}}&value={{value}}&user={{user}}"
}
```
发布消息：`cmd=write&tag=temperature_setpoint&value=25.5&user=admin`

**示例 5：XML 格式**
```json
{
  "payloadTemplate": "<command><type>set_value</type><tag>{{tagName}}</tag><value>{{value}}</value><timestamp>{{timestamp}}</timestamp></command>"
}
```
发布消息：
```xml
<command>
  <type>set_value</type>
  <tag>temperature_setpoint</tag>
  <value>25.5</value>
  <timestamp>1702886400000</timestamp>
</command>
```

**示例 6：自定义协议格式**
```json
{
  "payloadTemplate": "W|{{tagName}}|{{value}}|{{timestamp}}|END"
}
```
发布消息：`W|temperature_setpoint|25.5|1702886400000|END`

**示例 7：嵌套 JSON 格式**
```json
{
  "payloadTemplate": "{\"header\":{\"cmd\":\"write\",\"user\":\"{{user}}\"},\"body\":{\"tag\":\"{{tagName}}\",\"value\":{{value}},\"old\":{{oldValue}}}}"
}
```
发布消息：
```json
{
  "header": {
    "cmd": "write",
    "user": "admin"
  },
  "body": {
    "tag": "temperature_setpoint",
    "value": 25.5,
    "old": 20
  }
}
```

**示例 8：二进制格式（十六进制字符串）**
```json
{
  "payloadTemplate": "01{{value:hex}}FF"
}
```
发布消息：`0119FF`（假设 value=25，十六进制为 19）

### UI 设计

#### Tag 编辑器 - 回写配置

```
┌─ 变量回写配置 ─────────────────────────────────┐
│                                                  │
│ 访问模式: [读写 ▼]                              │
│                                                  │
│ 回写配置                                         │
│ ├─ 回写主题: [device/001/control]              │
│ ├─ QoS: [1 ▼]                                   │
│ └─ Retain: [ ]                                  │
│                                                  │
│ 消息模板（用户自定义）                           │
│ ┌──────────────────────────────────────────┐   │
│ │ {"cmd":"write","val":{{value}},          │   │
│ │  "ts":{{timestamp}},"user":"{{user}}"}   │   │
│ └──────────────────────────────────────────┘   │
│                                                  │
│ 可用变量：                                       │
│ • {{value}} - 回写的值                          │
│ • {{timestamp}} - 时间戳                        │
│ • {{datetime}} - 日期时间                       │
│ • {{user}} - 操作用户                           │
│ • {{tagName}} - 变量名                          │
│ • {{oldValue}} - 原值                           │
│                                                  │
│ 模板示例：                                       │
│ [JSON] [文本] [XML] [自定义]                    │
│                                                  │
│ 值验证（可选）                                   │
│ ├─ 最小值: [0]                                  │
│ ├─ 最大值: [100]                                │
│ └─ 必填: [✓]                                    │
│                                                  │
│ 其他选项                                         │
│ └─ 需要确认: [✓]                                │
│                                                  │
│ 测试回写                                         │
│ ├─ 测试值: [25.5]                               │
│ └─ 预览消息:                                     │
│   ┌──────────────────────────────────────────┐ │
│   │ {"cmd":"write","val":25.5,               │ │
│   │  "ts":1702886400000,"user":"admin"}      │ │
│   └──────────────────────────────────────────┘ │
│                                                  │
│ [测试发布] [保存配置]                            │
└──────────────────────────────────────────────────┘
```

**UI 功能说明**：
- **消息模板**：多行文本编辑器，用户可以输入任意格式的模板
- **可用变量**：显示所有可用的模板变量及说明
- **模板示例**：提供常用格式的快速模板（点击插入）
- **预览消息**：实时预览替换变量后的实际消息
- **测试发布**：使用测试值实际发布消息到 MQTT Broker

#### 消息发布界面

```
┌─ MQTT 消息发布 ─────────────────────┐
│                                      │
│ 连接: [MQTT Broker 1 ▼]            │
│ 主题: [device/001/control]          │
│ QoS: [1 ▼]  Retain: [ ]            │
│                                      │
│ 消息内容                             │
│ ┌────────────────────────────────┐ │
│ │ {                              │ │
│ │   "command": "set_switch",     │ │
│ │   "value": 1,                  │ │
│ │   "timestamp": 1702886400000   │ │
│ │ }                              │ │
│ └────────────────────────────────┘ │
│                                      │
│ [格式化] [清空]                      │
│                                      │
│ [发布消息]                           │
│                                      │
│ 发布历史                             │
│ ┌────────────────────────────────┐ │
│ │ 2024-12-15 10:30:45            │ │
│ │ Topic: device/001/control      │ │
│ │ 成功                            │ │
│ └────────────────────────────────┘ │
└──────────────────────────────────────┘
```

### Designer 集成

#### 变量回写

**组件触发回写**：
```json
{
  "type": "Button",
  "props": {
    "text": "开启设备"
  },
  "events": {
    "click": {
      "type": "writeTag",
      "config": {
        "tagName": "device_switch",
        "value": 1,
        "confirmation": true,
        "confirmMessage": "确定要开启设备吗？"
      }
    }
  }
}
```

**表达式回写**：
```json
{
  "type": "Slider",
  "props": {
    "value": "{{ $tags.temperature_setpoint.value }}"
  },
  "events": {
    "change": {
      "type": "writeTag",
      "config": {
        "tagName": "temperature_setpoint",
        "value": "{{ $event.value }}"
      }
    }
  }
}
```

#### 消息发布

**批量控制**：
```json
{
  "type": "Button",
  "props": {
    "text": "批量设置"
  },
  "events": {
    "click": {
      "type": "publishMqtt",
      "config": {
        "connectionId": "mqtt-conn-uuid",
        "topic": "device/batch-control",
        "payload": {
          "command": "batch_set",
          "devices": ["device001", "device002"],
          "parameters": {
            "mode": "auto",
            "temperature": "{{ $vars.targetTemp }}"
          }
        }
      }
    }
  }
}
```

## 实时数据推送

### Socket.IO 通信协议

使用 Socket.IO 进行实时双向通信，统一管理所有实时数据订阅。

#### 客户端订阅

**订阅主题**：
```javascript
// 订阅 MQTT 主题
socket.emit('subscribe_mqtt_topic', {
  subscriptionId: 'mqtt-sub-uuid'
});

// 订阅变量
socket.emit('subscribe_tags', {
  connectionId: 'mqtt-conn-uuid',
  tagNames: ['temperature_sensor_01', 'humidity_sensor_01']
});
```

**取消订阅**：
```javascript
socket.emit('unsubscribe_mqtt_topic', {
  subscriptionId: 'mqtt-sub-uuid'
});

socket.emit('unsubscribe_tags', {
  tagNames: ['temperature_sensor_01']
});
```

#### 服务端推送

**主题消息推送**：
```javascript
// 事件名：mqtt_topic_message
socket.on('mqtt_topic_message', (data) => {
  // data 结构
  {
    "subscriptionId": "uuid",
    "topic": "sensor/device001/data",
    "payload": "{\"temperature\":25.5}",
    "qos": 1,
    "retain": false,
    "timestamp": 1702886400000
  }
});
```

**变量值推送**：
```javascript
// 事件名：tag_value_update
socket.on('tag_value_update', (data) => {
  // data 结构
  {
    "connectionId": "uuid",
    "tags": [
      {
        "id": "uuid",
        "name": "temperature_sensor_01",
        "value": 25.5,
        "timestamp": 1702886400000,
        "quality": "good",
        "unit": "°C"
      }
    ]
  }
});
```

#### 连接管理

**认证**：
```javascript
// 连接时携带 token
const socket = io('http://localhost:9099', {
  auth: {
    token: 'jwt-token'
  }
});
```

**房间管理**：
- 每个项目一个房间：`project:{projectId}`
- 每个连接一个房间：`connection:{connectionId}`
- 每个订阅一个房间：`subscription:{subscriptionId}`

## Designer 集成

### 数据源架构统一

Designer 和 DataCenter 共用同一个 `dev_core` 后端，数据源配置保持一致的架构。

#### 现有数据源类型

**关系型数据库查询**：
```json
{
  "id": "ds_device_list",
  "type": "dataCenter",
  "config": {
    "sourceType": "query",
    "queryId": "query-uuid",  // 保存的查询 ID
    "parameters": {
      "status": 1
    }
  }
}
```

**访问方式**：HTTP 请求
- `GET /api/v1/data/queries/:queryId/execute`
- 传递参数，返回查询结果

#### MQTT 数据源类型

**主题订阅模式**：
```json
{
  "id": "ds_mqtt_raw",
  "type": "dataCenter",
  "config": {
    "sourceType": "mqtt_topic",
    "connectionId": "mqtt-conn-uuid",
    "subscriptionId": "mqtt-sub-uuid",  // 保存的主题订阅 ID
    "mode": "realtime"
  }
}
```

**访问方式**：
- **查询**（HTTP）：`GET /api/v1/data/mqtt/subscriptions/:subscriptionId/latest` - 获取最新消息
- **订阅**（Socket.IO）：`socket.emit('subscribe_mqtt_topic', { subscriptionId })` - 实时接收消息

**变量订阅模式**：
```json
{
  "id": "ds_sensors",
  "type": "dataCenter",
  "config": {
    "sourceType": "mqtt_tags",
    "connectionId": "mqtt-conn-uuid",
    "tagNames": ["temperature_sensor_01", "humidity_sensor_01"],  // 变量名数组
    "mode": "realtime"
  }
}
```

**访问方式**：
- **查询**（HTTP）：`POST /api/v1/data/tags/values` - 批量获取变量值
- **订阅**（Socket.IO）：`socket.emit('subscribe_tags', { tagNames })` - 实时接收变量更新

### 组件绑定示例

#### 主题订阅模式
```json
{
  "type": "Text",
  "props": {
    "text": "{{ JSON.parse($ds_mqtt_raw.payload).temperature }}°C"
  }
}
```

#### 变量订阅模式
```json
{
  "type": "Text",
  "props": {
    "text": "{{ $ds_sensors.temperature_sensor_01.value }}°C",
    "color": "{{ $ds_sensors.temperature_sensor_01.quality === 'good' ? '#00ff00' : '#ff0000' }}"
  }
}
```

### Designer 数据源选择器

**关系型数据库**：
1. 选择连接
2. 枚举该连接下的已保存查询列表
3. 选择查询（通过 queryId）

**MQTT 主题订阅**：
1. 选择 MQTT 连接
2. 枚举该连接下的主题订阅列表
3. 选择主题订阅（通过 subscriptionId）

**MQTT 变量订阅**：
1. 选择 MQTT 连接
2. 枚举该连接下的变量列表（按分组展示）
3. 多选变量（通过 tagName 数组）

## UI 设计

### 左侧连接树

```
[消息连接]
● MQTT Broker 1
  ├─ 主题订阅 (蓝色)
  │  ├─ 设备状态 (sensor/+/status)
  │  └─ 报警信息 (alarm/#)
  │
  └─ 变量 (绿色)
     ├─ 传感器组
     │  ├─ temperature_sensor_01
     │  ├─ humidity_sensor_01
     │  └─ pressure_sensor_01
     └─ 设备组
        └─ device_status_01
```

### 右侧标签页

#### 标签页 1：主题订阅管理
- 订阅列表（主题、QoS、消息速率、最后消息时间）
- 新建订阅按钮
- 查看消息、编辑、删除操作

#### 标签页 2：变量管理
- 变量列表（按分组展示）
- 显示当前值、时间戳、质量戳
- 新建变量、批量导入按钮
- 编辑、测试、删除操作

#### 标签页 3：实时消息监控
- 实时显示接收的 MQTT 消息
- 支持暂停、清空、导出
- 显示主题、QoS、Retain、Payload

#### 标签页 4：变量编辑器
- 基础信息（名称、显示名称、数据类型）
- MQTT 配置（Topic、QoS、访问模式）
- 解析规则配置（格式、路径、脚本）
- 测试解析功能

## 实施计划

### 阶段 1：基础连接管理（1-2 周）

**目标**：实现 MQTT 连接的创建、配置、测试和管理

**后端任务**：
- 完善 `data_mqtt_configs` 表结构
- 实现 `MqttProtocol` 基础功能（连接、断开、测试）
- 实现 MQTT 连接 CRUD API
  - `POST /api/v1/data/projects/:projectId/connections`
  - `GET /api/v1/data/projects/:projectId/connections?type=mqtt`
  - `PUT /api/v1/data/projects/:projectId/connections/:id`
  - `DELETE /api/v1/data/projects/:projectId/connections/:id`
  - `POST /api/v1/data/projects/:projectId/connections/test`
- 实现连接状态管理和健康检查
- 实现连接启动/停止功能

**前端任务**：
- 在 `connectionTypes.js` 添加 MQTT 配置
- 创建 `MqttConnectionForm` 组件
  - Broker 地址、端口、协议配置
  - 客户端 ID、用户名、密码
  - QoS、Clean Session、Keep Alive 等选项
  - SSL/TLS 配置（可选）
- 在连接列表中支持 MQTT 类型显示
- 实现连接测试 UI
- 实现连接状态实时更新

**验收标准**：
- 可以创建、编辑、删除 MQTT 连接
- 可以测试连接是否成功
- 连接状态实时显示（已连接/未连接/错误）
- 可以手动启动/停止连接

### 阶段 2：主题订阅管理（1-2 周）

**目标**：实现 MQTT 主题的订阅和原始消息接收

**后端任务**：
- 创建 `data_mqtt_subscriptions` 表
- 实现主题订阅功能（支持通配符 `+` 和 `#`）
- 实现消息接收和缓存
- 实现主题订阅 CRUD API
  - `POST /api/v1/data/projects/:projectId/mqtt/subscriptions`
  - `GET /api/v1/data/projects/:projectId/mqtt/subscriptions?connectionId=uuid`
  - `PUT /api/v1/data/mqtt/subscriptions/:id`
  - `DELETE /api/v1/data/mqtt/subscriptions/:id`
  - `GET /api/v1/data/mqtt/subscriptions/:id/latest` - 获取最新消息
- 实现 Socket.IO 基础框架
  - 认证中间件
  - 房间管理
  - 订阅/取消订阅事件处理
- 实现原始消息推送

**前端任务**：
- 创建 `MqttSubscriptionList` 组件
  - 显示订阅列表（主题、QoS、消息速率）
  - 新建、编辑、删除订阅
- 创建 `MqttMessageViewer` 组件
  - 实时消息展示
  - 支持暂停、清空、导出
  - 消息格式化（JSON 高亮）
- 实现 Socket.IO 客户端连接
- 实现订阅管理和消息接收

**验收标准**：
- 可以创建、编辑、删除主题订阅
- 可以通过 HTTP 获取最新消息
- 可以通过 Socket.IO 实时接收消息
- 消息显示主题、QoS、Payload、时间戳等信息

### 阶段 3：变量管理与解析（2-3 周）

**目标**：实现变量（Tag）的创建、配置和消息解析

**后端任务**：
- 实现 Tag CRUD API（复用现有 `data_tags` 表）
  - `POST /api/v1/data/projects/:projectId/tags`
  - `GET /api/v1/data/projects/:projectId/tags?connectionId=uuid`
  - `PUT /api/v1/data/tags/:id`
  - `DELETE /api/v1/data/tags/:id`
  - `POST /api/v1/data/tags/values` - 批量获取变量值
  - `GET /api/v1/data/tags/:id/value` - 获取单个变量值
- 实现 Tag 分组 API
  - `POST /api/v1/data/projects/:projectId/tag-groups`
  - `GET /api/v1/data/projects/:projectId/tag-groups?connectionId=uuid`
- 实现消息解析引擎（支持 JSON、文本、二进制、脚本）
  - JSONPath 解析器
  - 文本分隔符解析器
  - 二进制解析器
  - JavaScript 脚本解析器（vm2 沙箱）
- 实现 Tag 订阅功能
  - 自动订阅启用的 Tag
  - 消息解析和值更新
  - 值缓存管理
- 实现 Socket.IO Tag 值推送
  - `subscribe_tags` 事件处理
  - `tag_value_update` 事件推送

**前端任务**：
- 创建 `TagGroupTree` 组件（树形结构展示分组）
- 创建 `TagList` 组件
  - 显示变量列表（名称、类型、当前值、时间戳、质量戳）
  - 支持按分组过滤
- 创建 `TagEditor` 组件
  - 基础信息配置
  - MQTT Topic 配置
  - 解析规则配置（多种格式）
  - 测试解析功能
- 创建解析规则编辑器
  - JSON 格式：JSONPath 配置
  - 文本格式：分隔符和索引配置
  - 脚本格式：Monaco Editor 编辑器
- 创建 `TagMonitor` 组件（实时监控）
- 实现 Socket.IO Tag 订阅

**验收标准**：
- 可以创建、编辑、删除 Tag 和分组
- 可以配置多种解析规则（JSON、文本、脚本）
- 可以测试解析规则是否正确
- 可以通过 HTTP 批量查询 Tag 值
- 可以通过 Socket.IO 实时接收 Tag 值更新
- 可以实时查看 Tag 的当前值、时间戳、质量戳

### 阶段 4：Designer 集成（1-2 周）

**目标**：在 Designer 中支持 MQTT 数据源绑定

**后端任务**：
- 实现 Socket.IO 服务器
- 实现订阅管理（房间、认证、权限）
- 实现 MQTT 消息转发到 Socket.IO
- 实现 Tag 值变化推送
- 实现 HTTP API（获取最新值、批量查询）

**Designer 任务**：
- 扩展数据源配置（支持 `mqtt_topic` 和 `mqtt_tags`）
  - 在 `dataCenter` 类型下添加 `sourceType`
  - 支持 `query`（现有）、`mqtt_topic`（新增）、`mqtt_tags`（新增）
- 实现数据源选择器 UI
  - 枚举 MQTT 连接列表
  - 根据 sourceType 显示不同的选择器
  - `mqtt_topic`：枚举主题订阅列表，单选
  - `mqtt_tags`：枚举变量列表（按分组），多选
- 实现 Socket.IO 客户端连接
  - 连接管理（连接、断开、重连）
  - 认证（携带 JWT token）
- 实现订阅管理
  - 页面加载时自动订阅
  - 页面卸载时自动取消订阅
  - 订阅去重
- 实现数据接收和状态更新
  - 接收 `mqtt_topic_message` 事件
  - 接收 `tag_value_update` 事件
  - 更新数据源状态
  - 触发组件重新渲染
- 实现表达式绑定
  - 支持访问主题消息：`$ds_mqtt_raw.payload`
  - 支持访问变量值：`$ds_sensors.temperature_sensor_01.value`
  - 支持访问时间戳和质量戳
- 实现混合模式（HTTP + Socket.IO）
  - 初始加载使用 HTTP 获取当前值
  - 后续更新使用 Socket.IO 实时推送
- 实现历史数据缓存（可选）
  - 缓存最近 N 条数据
  - 支持图表展示历史趋势

**验收标准**：
- Designer 开发态可以选择 MQTT 连接
- 可以枚举并选择主题订阅或变量
- 组件可以绑定 MQTT 数据源
- 运行态页面加载时通过 HTTP 获取初始数据
- 运行态通过 Socket.IO 实时接收数据更新
- 组件可以通过表达式访问数据（值、时间戳、质量戳）
- 页面卸载时正确清理订阅

### 阶段 5：数据回写功能（1-2 周）

**目标**：实现 MQTT 数据回写和消息发布功能

**后端任务**：
- 扩展 Tag 模型支持回写配置（`metadata.writeConfig`）
- 实现 Tag 回写 API
  - `POST /api/v1/data/tags/:tagId/write` - 变量回写
  - `POST /api/v1/data/mqtt/publish` - 消息发布
- 实现消息模板引擎
  - 支持用户自定义模板（任意格式）
  - 支持变量替换（`{{variable}}`）
  - 支持变量格式化（`{{value:json}}`、`{{value:hex}}` 等）
  - 模板验证（检查语法错误）
- 实现值验证（最小值、最大值、必填等）
- 实现回写确认机制
- 实现回写日志记录
- 实现 MqttProtocol.publish() 方法

**前端任务**：
- 扩展 `TagEditor` 组件
  - 添加回写配置面板
  - 消息模板编辑器（多行文本输入）
  - 可用变量列表和说明
  - 模板示例快速插入
  - 实时消息预览（替换变量后的结果）
  - 值验证配置
  - 测试回写功能（实际发布测试消息）
- 创建 `MqttPublisher` 组件
  - 消息发布界面
  - 消息格式化（JSON 美化）
  - 发布历史记录
- 实现回写 API 调用
- 实现确认对话框

**Designer 任务**：
- 实现 `writeTag` 动作类型
- 实现 `publishMqtt` 动作类型
- 支持组件事件触发回写
- 支持表达式计算回写值
- 实现回写确认 UI

**验收标准**：
- 可以配置 Tag 回写规则
- 可以通过 API 回写变量值
- 可以通过 API 发布自定义消息
- Designer 中可以配置回写动作
- 回写前可以显示确认对话框
- 回写操作有日志记录

### 阶段 6：优化与扩展（1 周）

**目标**：性能优化和功能扩展

**优化任务**：
- 消息频率限制（防止过载）
- 连接池管理
- 消息缓存和历史数据
- 错误处理和重连机制
- 性能监控和日志

**扩展功能**：
- 批量导入 Tag
- Tag 模板功能
- 消息统计和分析
- 告警规则集成
- 数据导出功能
- 回写权限控制
- 回写审计日志

## API 设计

### HTTP API（查询模式）

#### 关系型数据库查询
```
GET /api/v1/data/queries/:queryId/execute
POST /api/v1/data/queries/:queryId/execute
```

**请求参数**：
```json
{
  "parameters": {
    "status": 1,
    "startDate": "2025-01-01"
  }
}
```

**响应**：
```json
{
  "success": true,
  "data": {
    "columns": ["id", "name", "status"],
    "rows": [[1, "Device1", 1], [2, "Device2", 1]],
    "rowCount": 2
  }
}
```

#### MQTT 主题最新消息
```
GET /api/v1/data/mqtt/subscriptions/:subscriptionId/latest
```

**响应**：
```json
{
  "success": true,
  "data": {
    "subscriptionId": "uuid",
    "topic": "sensor/device001/data",
    "payload": "{\"temperature\":25.5}",
    "qos": 1,
    "timestamp": 1702886400000
  }
}
```

#### MQTT 变量批量查询
```
POST /api/v1/data/tags/values
```

**请求参数**：
```json
{
  "connectionId": "mqtt-conn-uuid",
  "tagNames": ["temperature_sensor_01", "humidity_sensor_01"]
}
```

**响应**：
```json
{
  "success": true,
  "data": {
    "tags": [
      {
        "name": "temperature_sensor_01",
        "value": 25.5,
        "timestamp": 1702886400000,
        "quality": "good",
        "unit": "°C"
      },
      {
        "name": "humidity_sensor_01",
        "value": 60,
        "timestamp": 1702886400000,
        "quality": "good",
        "unit": "%"
      }
    ]
  }
}
```

#### MQTT 变量回写
```
POST /api/v1/data/tags/:tagId/write
```

**请求参数**：
```json
{
  "value": 30,
  "confirmation": true
}
```

**响应**：
```json
{
  "success": true,
  "data": {
    "tagId": "uuid",
    "tagName": "temperature_setpoint",
    "oldValue": 25,
    "newValue": 30,
    "timestamp": 1702886400000,
    "publishedTopic": "device/001/control",
    "publishedPayload": "{\"command\":\"set_value\",\"value\":30}"
  }
}
```

#### MQTT 消息发布
```
POST /api/v1/data/mqtt/publish
```

**请求参数**：
```json
{
  "connectionId": "mqtt-conn-uuid",
  "topic": "device/001/control",
  "qos": 1,
  "retain": false,
  "payload": {
    "command": "set_switch",
    "value": 1,
    "timestamp": 1702886400000
  }
}
```

**响应**：
```json
{
  "success": true,
  "data": {
    "topic": "device/001/control",
    "qos": 1,
    "timestamp": 1702886400000,
    "messageId": "uuid"
  }
}
```

#### 枚举 API（Designer 数据源选择器）

**枚举连接列表**：
```
GET /api/v1/data/projects/:projectId/connections?type=mqtt
```

**枚举主题订阅列表**：
```
GET /api/v1/data/projects/:projectId/mqtt/subscriptions?connectionId=uuid
```

**枚举变量列表**：
```
GET /api/v1/data/projects/:projectId/tags?connectionId=uuid
```

**枚举变量分组**：
```
GET /api/v1/data/projects/:projectId/tag-groups?connectionId=uuid
```

### Socket.IO 事件

#### 客户端发送事件

| 事件名 | 参数 | 说明 |
|--------|------|------|
| `subscribe_mqtt_topic` | `{ subscriptionId }` | 订阅 MQTT 主题 |
| `unsubscribe_mqtt_topic` | `{ subscriptionId }` | 取消订阅主题 |
| `subscribe_tags` | `{ connectionId, tagNames[] }` | 订阅变量 |
| `unsubscribe_tags` | `{ tagNames[] }` | 取消订阅变量 |
| `subscribe_connection` | `{ connectionId }` | 订阅连接下所有数据 |
| `unsubscribe_connection` | `{ connectionId }` | 取消订阅连接 |

#### 服务端推送事件

| 事件名 | 数据结构 | 说明 |
|--------|---------|------|
| `mqtt_topic_message` | `{ subscriptionId, topic, payload, qos, timestamp }` | MQTT 主题消息 |
| `tag_value_update` | `{ connectionId, tags[] }` | 变量值更新 |
| `connection_status` | `{ connectionId, status }` | 连接状态变化 |
| `error` | `{ code, message }` | 错误信息 |

## 数据流设计

### 查询模式（HTTP）

**关系型数据库查询流程**：
```
Designer 运行时
  ↓ HTTP GET
/api/v1/data/queries/:queryId/execute
  ↓
QueryController
  ↓
QueryService.execute()
  ↓
DatabaseDriver (MySQL/PostgreSQL/SQL Server)
  ↓
返回查询结果
```

**MQTT 变量查询流程**：
```
Designer 运行时
  ↓ HTTP POST
/api/v1/data/tags/values
  ↓
TagController
  ↓
TagService.getValues()
  ↓
从缓存/内存中获取最新值
  ↓
返回变量值
```

### 订阅模式（Socket.IO）

**MQTT 实时数据流**：
```
MQTT Broker
  ↓ 发布消息
MqttProtocol.client.on('message')
  ↓
MqttService.handleMessage()
  ↓ 解析消息
TagParserService.parse()
  ↓ 更新缓存
TagValueCache.set()
  ↓ 推送到 Socket.IO
Socket.IO.to('tag:temperature').emit('tag_value_update')
  ↓
Designer 运行时接收
  ↓
更新组件状态
```

**订阅管理流程**：
```
Designer 运行时
  ↓ Socket.IO
socket.emit('subscribe_tags', { tagNames })
  ↓
SocketService.handleSubscribe()
  ↓ 加入房间
socket.join('tag:temperature_sensor_01')
  ↓ 返回当前值
socket.emit('tag_value_update', currentValues)
  ↓
持续接收实时更新
```

### Designer 数据源配置流程

**配置阶段（开发态）**：
```
1. 用户在 Designer 中添加数据源
2. 选择数据源类型（dataCenter）
3. 选择源类型（query/mqtt_topic/mqtt_tags）
4. 枚举可用资源：
   - query: 枚举已保存的查询列表
   - mqtt_topic: 枚举主题订阅列表
   - mqtt_tags: 枚举变量列表（按分组）
5. 选择资源（queryId/subscriptionId/tagNames）
6. 保存到页面 DSL
```

**运行阶段（运行态）**：
```
1. 页面加载，解析 DSL
2. 初始化数据源：
   - query: HTTP 请求获取数据
   - mqtt_topic: Socket.IO 订阅 + HTTP 获取最新值
   - mqtt_tags: Socket.IO 订阅 + HTTP 获取当前值
3. 建立 Socket.IO 连接（如果有实时数据源）
4. 发送订阅请求
5. 接收实时数据更新
6. 更新组件状态和 UI
7. 页面卸载时取消订阅
```

## 技术细节

### 消息解析引擎

**实现思路**：
- 使用 `jsonpath-plus` 库处理 JSON 格式
- 使用字符串分割处理文本格式
- 使用 Buffer 操作处理二进制格式
- 使用 `vm2` 沙箱执行自定义脚本

**安全考虑**：
- 脚本执行使用沙箱环境（vm2）
- 限制脚本执行时间（1 秒超时）
- 限制脚本访问权限（无文件系统、网络访问）

### 实时推送

**实现思路**：
- 使用 Socket.IO 进行双向通信
- 后端维护客户端连接池和订阅关系
- 使用房间（Room）机制管理订阅
- 消息分发采用发布-订阅模式
- 支持按项目、连接、订阅、Tag 过滤推送

**Socket.IO 房间设计**：
- `project:{projectId}`：项目级广播
- `connection:{connectionId}`：连接级广播
- `subscription:{subscriptionId}`：主题订阅
- `tag:{tagName}`：单个变量订阅

**性能优化**：
- 消息批量推送（减少网络开销）
- 消息频率限制（防抖/节流）
- 客户端断线重连机制（Socket.IO 自动重连）
- 心跳检测保持连接（Socket.IO 内置）
- 订阅去重（同一客户端多次订阅同一资源）

### 数据存储

**实时数据**：
- 存储在内存中（Redis 缓存）
- 保留最近 N 条消息
- 支持按时间范围查询

**历史数据**（可选）：
- 存储在时序数据库（InfluxDB/TimescaleDB）
- 支持长期数据分析
- 支持数据聚合和降采样

## 扩展性

### 支持其他协议

统一的 Tag 架构可以轻松扩展到其他协议：

**OPC UA**：
- `address`：NodeId（如 `ns=2;s=Device1.Temperature`）
- `metadata.protocol`：`opcua`
- 实现 `OpcuaProtocol` 类

**Modbus**：
- `address`：寄存器地址（如 `40001`）
- `metadata.protocol`：`modbus`
- 实现 `ModbusProtocol` 类

### 跨协议数据整合

Designer 中可以混合使用不同协议的 Tag：

```json
{
  "dataSources": [
    {
      "id": "ds_all_sensors",
      "type": "tags",
      "config": {
        "tags": [
          "mqtt_temperature",
          "opcua_pressure",
          "modbus_flow"
        ]
      }
    }
  ]
}
```

## 相关文档

- [连接管理](./connections.md)
- [查询管理](./queries.md)
- [数据库实现概述](./database-implementation.md)

---

**版本**: 1.0.0  
**最后更新**: 2025-12-15
