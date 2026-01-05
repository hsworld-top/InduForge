# 数据绑定系统 - 完整指南

## 概述

InduForge Designer 的数据绑定系统提供了强大的数据管理和绑定能力。设计器通过绑定**数据点（DataPoint）**来获取数据中心管理的各类数据，实现组件与数据的动态关联。

## 核心架构

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                                                                             │
│   数据中心 (DataCenter)                     设计器 (Designer)               │
│   ┌───────────────────────┐                ┌───────────────────────┐       │
│   │                       │                │                       │       │
│   │  数据点目录            │   ─────────►  │  数据源配置            │       │
│   │  (统一管理)            │   HTTP/WS     │  (绑定数据点)          │       │
│   │                       │                │                       │       │
│   │  • db.xxx             │                │  引用数据点:           │       │
│   │  • mqtt.xxx           │                │  path: mqtt.EMQX.xxx  │       │
│   │  • calc.xxx           │                │  mode: subscription   │       │
│   │                       │                │                       │       │
│   └───────────────────────┘                └───────────────────────┘       │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

## 核心特性

### 📌 数据点绑定

设计器的数据配置核心是绑定**数据点**，数据点由数据中心统一管理：

- **db.\*** - 数据库查询字段（如 `db.生产库.设备统计.device_count`）
- **mqtt.\*** - MQTT 变量（如 `mqtt.EMQX.温度组.temperature`）
- **calc.\*** - 计算单元输出（如 `calc.功率计算.power`）
- **opcua.\*** - OPC UA 节点（规划中）
- **ws.\*** - WebSocket 变量（规划中）

### 🔄 灵活的数据获取模式

数据点支持三种获取模式：

| 模式             | 说明               | 适用场景             |
| ---------------- | ------------------ | -------------------- |
| **Request**      | 单次请求，按需加载 | 配置数据、初始化数据 |
| **Poll**         | 轮询模式，定时刷新 | 统计数据、报表数据   |
| **Subscription** | 订阅模式，实时推送 | 实时监控、设备状态   |

### 🎯 强大的数据绑定

- 支持组件属性绑定（`props.*`）
- 支持组件样式绑定（`style.*`）
- 表达式语法，支持复杂计算
- 实时预览绑定值

### 🛠️ 可视化配置

- 数据点选择器（树形浏览、搜索筛选）
- 数据绑定配置面板
- 实时状态监控
- 错误提示和降级

## 快速开始

### 1. 选择数据点

在设计器中打开数据源配置面板，点击「选择数据点」：

```
┌─ 选择数据点 ──────────────────────────────────────────┐
│                                                        │
│  [🔍 搜索...]                    [类型: 全部 ▼]        │
│                                                        │
│  ▼ 📡 MQTT 变量                                       │
│    │ ▼ EMQX                                           │
│    │   │ ▼ 温度传感器                                 │
│    │   │   ├─ temperature    25.5 ℃                  │
│    │   │   └─ humidity       60 %                     │
│                                                        │
│  ▼ 🗄️ 数据库查询                                      │
│    │ ▼ 生产库                                         │
│    │   │ ▼ 设备统计                                   │
│    │   │   ├─ device_count   150                      │
│    │   │   └─ online_count   142                      │
│                                                        │
│  已选择: mqtt.EMQX.温度传感器.temperature              │
│                                                        │
│                         [取消]    [确定]               │
└────────────────────────────────────────────────────────┘
```

### 2. 配置数据源

```json
{
  "dataSources": [
    {
      "id": "ds_temp",
      "path": "mqtt.EMQX.温度传感器.temperature",
      "dataType": "number",
      "mode": "subscription"
    }
  ]
}
```

### 3. 绑定到组件

```json
{
  "id": "gauge_temp",
  "type": "Gauge",
  "bindings": {
    "props.value": "{{ data.ds_temp }}",
    "props.unit": "℃",
    "props.color": "{{ data.ds_temp > 80 ? '#ff4d4f' : '#52c41a' }}"
  }
}
```

## 数据点路径格式

```
{来源类型}.{连接名}.{分组名}.{数据点名}
```

**路径示例**：

| 路径                               | 说明                                             |
| ---------------------------------- | ------------------------------------------------ |
| `db.生产库.设备统计.device_count`  | 数据库查询「设备统计」的 `device_count` 字段     |
| `mqtt.EMQX.温度传感器.temperature` | MQTT 连接下「温度传感器」组的 `temperature` 变量 |
| `calc.功率计算.power`              | 计算单元「功率计算」的 `power` 输出              |

## 数据源配置

### DataSourceConfig

```typescript
interface DataSourceConfig {
  id: string; // 数据源 ID，用于绑定引用
  path: string; // 数据点路径
  dataType: "number" | "string" | "boolean" | "object" | "array"; // 数据类型
  mode: "request" | "poll" | "subscription"; // 获取模式
  interval?: number; // 轮询间隔（毫秒），mode=poll 时有效
  transformer?: string; // 数据转换脚本（可选）
  errorHandler?: string; // 错误处理函数
  options?: {
    autoStart?: boolean; // 页面加载时自动启动
    retryOnError?: boolean; // 错误时重试
    retryCount?: number; // 重试次数
    retryInterval?: number; // 重试间隔
  };
}
```

### 数据类型说明

| 数据类型  | 说明       | 处理方式                                          |
| --------- | ---------- | ------------------------------------------------- |
| `number`  | 数值类型   | 直接使用，支持数学运算                            |
| `string`  | 字符串类型 | 直接使用，支持字符串操作                          |
| `boolean` | 布尔类型   | 直接使用，支持条件判断                            |
| `object`  | 对象类型   | 可通过 `transformer` 脚本解析提取字段             |
| `array`   | 数组类型   | 可通过 `transformer` 脚本处理或直接绑定到列表组件 |

### 数据转换脚本（transformer）

对于 `object` 或 `array` 类型的数据点，可以编写转换脚本提取或处理数据：

```javascript
// 示例 1：从对象中提取字段
{
  "transformer": "(data) => data.value"
}

// 示例 2：从对象中提取多个字段
{
  "transformer": "(data) => ({ temp: data.temperature, hum: data.humidity })"
}

// 示例 3：处理数组，提取第一个元素
{
  "transformer": "(data) => data[0]"
}

// 示例 4：数组过滤
{
  "transformer": "(data) => data.filter(item => item.status === 'online')"
}

// 示例 5：数组聚合计算
{
  "transformer": "(data) => data.reduce((sum, item) => sum + item.value, 0)"
}
```

### 获取模式详解

#### Request 模式（单次请求）

适用于不常变化的数据，如配置、字典等。

```json
{
  "id": "ds_config",
  "path": "db.系统库.系统配置.site_name",
  "dataType": "string",
  "mode": "request"
}
```

#### Poll 模式（定时轮询）

适用于需要定期更新的数据，如统计数据。

```json
{
  "id": "ds_stats",
  "path": "db.生产库.设备统计.online_count",
  "dataType": "number",
  "mode": "poll",
  "interval": 5000
}
```

#### Subscription 模式（实时订阅）

适用于实时数据，通过 Socket.IO 推送。

```json
{
  "id": "ds_realtime",
  "path": "mqtt.EMQX.温度传感器.temperature",
  "dataType": "number",
  "mode": "subscription"
}
```

## 数据绑定

### 绑定语法

使用 `{{ expression }}` 语法将数据绑定到组件属性：

```json
{
  "bindings": {
    "props.value": "{{ data.ds_temp }}",
    "props.title": "{{ '当前温度: ' + data.ds_temp + '℃' }}",
    "style.color": "{{ data.ds_temp > 80 ? '#ff4d4f' : '#52c41a' }}",
    "style.display": "{{ data.ds_temp > 0 ? 'block' : 'none' }}"
  }
}
```

### 表达式上下文

```typescript
interface ExpressionContext {
  data: Record<string, any>; // 数据源数据（通过 data.{数据源ID} 访问）
  vars: Record<string, any>; // 页面变量
  props: Record<string, any>; // 组件属性
  $user: {
    // 用户信息
    id: string;
    name: string;
    role: string;
  };
  $route: {
    // 路由信息
    params: Record<string, any>;
    query: Record<string, any>;
  };
}
```

### 内置函数

```javascript
// 格式化
{
  {
    $format.number(data.ds_value, 2);
  }
} // 数字格式化，保留 2 位小数
{
  {
    $format.date(data.ds_time, "YYYY-MM-DD");
  }
} // 日期格式化

// 数组操作
{
  {
    $array.sum(data.ds_list, "value");
  }
} // 求和
{
  {
    $array.avg(data.ds_list, "value");
  }
} // 平均值
{
  {
    $array.max(data.ds_list, "value");
  }
} // 最大值

// 条件判断
{
  {
    $if(data.ds_status > 0, "正常", "异常");
  }
}
```

## 使用场景

### 场景 1: 实时监控大屏

```json
{
  "dataSources": [
    {
      "id": "ds_temp",
      "path": "mqtt.EMQX.车间1.temperature",
      "dataType": "number",
      "mode": "subscription"
    },
    {
      "id": "ds_humidity",
      "path": "mqtt.EMQX.车间1.humidity",
      "dataType": "number",
      "mode": "subscription"
    },
    {
      "id": "ds_power",
      "path": "calc.功率计算.power",
      "dataType": "number",
      "mode": "subscription"
    }
  ],
  "components": [
    {
      "id": "gauge_temp",
      "type": "Gauge",
      "bindings": {
        "props.value": "{{ data.ds_temp }}",
        "props.unit": "℃",
        "props.color": "{{ data.ds_temp > 35 ? '#ff4d4f' : '#52c41a' }}"
      }
    },
    {
      "id": "text_power",
      "type": "Text",
      "bindings": {
        "props.content": "{{ '实时功率: ' + $format.number(data.ds_power, 1) + ' kW' }}"
      }
    }
  ]
}
```

### 场景 2: 数据报表

```json
{
  "dataSources": [
    {
      "id": "ds_device_count",
      "path": "db.生产库.设备统计.device_count",
      "dataType": "number",
      "mode": "poll",
      "interval": 10000
    },
    {
      "id": "ds_online_count",
      "path": "db.生产库.设备统计.online_count",
      "dataType": "number",
      "mode": "poll",
      "interval": 10000
    }
  ],
  "components": [
    {
      "id": "card_total",
      "type": "StatCard",
      "bindings": {
        "props.title": "设备总数",
        "props.value": "{{ data.ds_device_count }}",
        "props.suffix": "台"
      }
    },
    {
      "id": "card_online",
      "type": "StatCard",
      "bindings": {
        "props.title": "在线设备",
        "props.value": "{{ data.ds_online_count }}",
        "props.suffix": "台",
        "props.trend": "{{ (data.ds_online_count / data.ds_device_count * 100).toFixed(1) + '%' }}"
      }
    }
  ]
}
```

### 场景 3: 条件显示

```json
{
  "dataSources": [
    {
      "id": "ds_alarm",
      "path": "mqtt.EMQX.报警系统.has_alarm",
      "dataType": "boolean",
      "mode": "subscription"
    }
  ],
  "components": [
    {
      "id": "alarm_indicator",
      "type": "Indicator",
      "bindings": {
        "style.display": "{{ data.ds_alarm ? 'block' : 'none' }}",
        "style.backgroundColor": "#ff4d4f",
        "props.text": "报警中"
      }
    }
  ]
}
```

### 场景 4: 对象类型数据处理

当数据点返回的是 JSON 对象时，可以通过 `transformer` 脚本解析：

```json
{
  "dataSources": [
    {
      "id": "ds_sensor_raw",
      "path": "mqtt.EMQX.传感器.raw_data",
      "dataType": "object",
      "mode": "subscription",
      "transformer": "(data) => ({ temp: data.temperature, hum: data.humidity, ts: data.timestamp })"
    }
  ],
  "components": [
    {
      "id": "text_temp",
      "type": "Text",
      "bindings": {
        "props.content": "{{ '温度: ' + data.ds_sensor_raw.temp + '℃' }}"
      }
    },
    {
      "id": "text_humidity",
      "type": "Text",
      "bindings": {
        "props.content": "{{ '湿度: ' + data.ds_sensor_raw.hum + '%' }}"
      }
    }
  ]
}
```

### 场景 5: 数组类型数据处理

当数据点返回的是数组时，可以直接绑定到列表组件或通过 `transformer` 处理：

```json
{
  "dataSources": [
    {
      "id": "ds_device_list",
      "path": "db.生产库.设备列表.all_devices",
      "dataType": "array",
      "mode": "poll",
      "interval": 30000
    },
    {
      "id": "ds_online_devices",
      "path": "db.生产库.设备列表.all_devices",
      "dataType": "array",
      "mode": "poll",
      "interval": 30000,
      "transformer": "(data) => data.filter(d => d.status === 'online')"
    }
  ],
  "components": [
    {
      "id": "table_devices",
      "type": "Table",
      "bindings": {
        "props.data": "{{ data.ds_device_list }}"
      }
    },
    {
      "id": "text_online_count",
      "type": "Text",
      "bindings": {
        "props.content": "{{ '在线设备: ' + data.ds_online_devices.length + ' 台' }}"
      }
    }
  ]
}
```

## 辅助数据源类型

除了数据点外，设计器还支持以下辅助数据源类型：

### Static（静态数据）

用于配置固定数据，如下拉选项：

```json
{
  "id": "ds_options",
  "type": "static",
  "config": {
    "data": [
      { "value": 1, "label": "选项1" },
      { "value": 2, "label": "选项2" }
    ]
  }
}
```

### Computed（计算数据源）

基于其他数据源进行二次计算：

```json
{
  "id": "ds_summary",
  "type": "computed",
  "config": {
    "dependencies": ["ds_device_count", "ds_online_count"],
    "compute": "(sources) => ({ onlineRate: (sources.ds_online_count / sources.ds_device_count * 100).toFixed(1) })"
  }
}
```

## 性能优化

### 1. 合理选择获取模式

```javascript
// 实时数据 → subscription
{ "mode": "subscription" }

// 统计数据 → poll（间隔不要太短）
{ "mode": "poll", "interval": 10000 }

// 配置数据 → request
{ "mode": "request" }
```

### 2. 使用 Computed 缓存计算

```json
{
  "id": "ds_filtered",
  "type": "computed",
  "config": {
    "dependencies": ["ds_list"],
    "compute": "(sources) => sources.ds_list.filter(d => d.status === 1)"
  }
}
```

### 3. 避免复杂表达式

```javascript
// ❌ 不推荐：在绑定中进行复杂计算
"props.value": "{{ data.ds_list.filter(d => d.status === 1).reduce((a, b) => a + b.value, 0) }}"

// ✅ 推荐：使用 Computed 数据源
"props.value": "{{ data.ds_computed_total }}"
```

## 调试指南

### 查看数据源状态

打开浏览器控制台：

```javascript
// 查看所有数据源
const store = useDesignStore();
console.log(store.dataSources);

// 查看特定数据源
console.log(store.dataSources.ds_temp);
```

### 手动刷新数据源

```javascript
await store.dataSourceManager.refresh("ds_temp");
```

## 常见问题

### Q: 数据源状态一直是"加载中"？

A: 检查数据点路径是否正确，数据中心是否已创建该数据点。

### Q: 实时数据不更新？

A: 确认 `mode` 设置为 `subscription`，检查 Socket.IO 连接状态。

### Q: 如何获取多个数据点？

A: 为每个数据点创建单独的数据源，通过不同的 `id` 引用。

## 文档索引

- [数据绑定架构](./data-binding-architecture.md)
- [数据点方案设计](../datacenter/datapoint-design.md)
- [数据点改造实施](../datacenter/datapoint-implementation.md)
- [DSL 设计规范](../dsl-design.md)
- [设计中心概述](./README.md)
