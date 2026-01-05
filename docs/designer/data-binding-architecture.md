# 数据绑定系统架构说明

## 架构概述

InduForge 的数据绑定系统以**数据点（DataPoint）**为核心，设计器通过绑定数据点来获取数据中心管理的各类数据。

### 核心架构图

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              IDE (标签页容器)                                │
│                                                                             │
│  ┌─ Designer Tab ─────────────────────────────────────────────────────┐    │
│  │                                                                     │    │
│  │   ┌─────────────────┐    ┌─────────────────┐    ┌──────────────┐  │    │
│  │   │ DataSourcePanel │───►│DataSourceManager│───►│  DataBinder  │  │    │
│  │   │ (数据源配置)     │    │ (数据源管理器)   │    │ (属性绑定)   │  │    │
│  │   └─────────────────┘    └────────┬────────┘    └──────────────┘  │    │
│  │                                   │                                │    │
│  │   ┌─────────────────┐             │                                │    │
│  │   │DataPointSelector│             │                                │    │
│  │   │ (数据点选择器)   │             │                                │    │
│  │   └─────────────────┘             │                                │    │
│  │                                   │                                │    │
│  └───────────────────────────────────┼────────────────────────────────┘    │
│                                      │                                      │
│  ┌─ DataCenter Tab (可选打开) ───────┼───────────────────────────────┐    │
│  │                                   │                                │    │
│  │   数据点管理、连接配置、查询管理   │                                │    │
│  │                                   │                                │    │
│  └───────────────────────────────────┼────────────────────────────────┘    │
│                                      │                                      │
└──────────────────────────────────────┼──────────────────────────────────────┘
                                       │
                          ┌────────────┴────────────┐
                          │     HTTP / WebSocket    │
                          └────────────┬────────────┘
                                       │
┌──────────────────────────────────────┼──────────────────────────────────────┐
│                           Backend API (dev_core)                             │
│  ┌───────────────────────────────────┼────────────────────────────────────┐ │
│  │                                   │                                     │ │
│  │  数据点 API                       │  Socket.IO 推送                     │ │
│  │  • GET  /datapoints              │  • datapoint:subscribe              │ │
│  │  • GET  /datapoints/:id/value    │  • datapoint:value                  │ │
│  │                                   │  • datapoint:unsubscribe            │ │
│  │                                   │                                     │ │
│  └───────────────────────────────────┴────────────────────────────────────┘ │
│                                      │                                       │
│  ┌───────────────────────────────────┼────────────────────────────────────┐ │
│  │                           DataPointService                              │ │
│  │                                   │                                     │ │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐               │ │
│  │  │ DB 查询  │  │MQTT 变量 │  │ 计算输出  │  │ OPC UA   │               │ │
│  │  │ db.*     │  │ mqtt.*   │  │ calc.*   │  │ opcua.*  │               │ │
│  │  └──────────┘  └──────────┘  └──────────┘  └──────────┘               │ │
│  └────────────────────────────────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────────────────────────────────┘
```

## 数据流

### 完整数据流

```
1. 用户在数据点选择器中选择数据点
   ↓
2. DataSourcePanel 配置数据源（数据点路径 + 获取模式）
   ↓
3. DataSourceManager.register() 注册数据源
   ↓
4. DataSourceManager.start() 启动数据源
   ↓
5. 根据 mode 执行不同策略：
   • request → HTTP 单次请求
   • poll    → HTTP 定时轮询
   • subscription → Socket.IO 订阅
   ↓
6. Backend DataPointService 获取数据点值
   ↓
7. 返回数据到 DataSourceManager
   ↓
8. DataSourceManager.updateData() 更新 Store.dataSources
   ↓
9. DataBinder 监听变化，解析 {{ expression }}
   ↓
10. 更新组件属性，触发 Vue 响应式渲染
```

### 三种获取模式的实现

#### Request 模式

```
用户配置 → DataSourceManager.start()
                    ↓
            HTTP GET /datapoints/:path/value
                    ↓
            返回数据 → updateData()
                    ↓
            完成（不再请求）
```

#### Poll 模式

```
用户配置 → DataSourceManager.start()
                    ↓
            HTTP GET /datapoints/:path/value
                    ↓
            返回数据 → updateData()
                    ↓
            setInterval(interval)
                    ↓
            循环请求... (直到 stop())
```

#### Subscription 模式

```
用户配置 → DataSourceManager.start()
                    ↓
            socket.emit('datapoint:subscribe', { paths: [path] })
                    ↓
            socket.on('datapoint:value', callback)
                    ↓
            实时接收推送 → updateData()
                    ↓
            持续监听... (直到 stop() 调用 unsubscribe)
```

## 核心模块

### 1. DataSourceManager

**文件**: `designer/src/engine/datasource/DataSourceManager.js`

数据源管理器，负责数据源的生命周期和数据获取。

```javascript
class DataSourceManager {
  constructor(store, options = {}) {
    this.store = store;
    this.dataSources = new Map();
    this.subscriptions = new Map();
    this.socket = null;
  }

  /**
   * 注册数据源
   */
  register(config) {
    this.dataSources.set(config.id, {
      config,
      status: "idle",
      data: null,
      error: null,
    });
  }

  /**
   * 启动数据源
   */
  async start(id) {
    const ds = this.dataSources.get(id);
    const { config } = ds;

    switch (config.mode) {
      case "request":
        await this.fetchDataPoint(config);
        break;
      case "poll":
        await this.startPolling(config);
        break;
      case "subscription":
        this.subscribeDataPoint(config);
        break;
    }
  }

  /**
   * 获取数据点值（HTTP）
   */
  async fetchDataPoint(config) {
    const { path } = config.config;
    const response = await request.get(
      `/data/projects/${this.projectId}/datapoints/value`,
      { params: { path } }
    );
    this.updateData(config.id, response.data.value);
  }

  /**
   * 订阅数据点（Socket.IO）
   */
  subscribeDataPoint(config) {
    const { path } = config.config;

    this.socket.emit("datapoint:subscribe", {
      projectId: this.projectId,
      paths: [path],
    });

    const handler = (data) => {
      if (data.path === path) {
        this.updateData(config.id, data.value);
      }
    };

    this.socket.on("datapoint:value", handler);
    this.subscriptions.set(config.id, { path, handler });
  }

  /**
   * 停止数据源
   */
  stop(id) {
    const sub = this.subscriptions.get(id);
    if (sub) {
      this.socket.emit("datapoint:unsubscribe", {
        projectId: this.projectId,
        paths: [sub.path],
      });
      this.socket.off("datapoint:value", sub.handler);
      this.subscriptions.delete(id);
    }
  }
}
```

### 2. DataPointSelector

**文件**: `designer/src/components/panels/DataPointSelector.vue`

数据点选择器组件，提供可视化的数据点浏览和选择。

**功能**：

- 树形展示数据点目录
- 按类型分组（数据库、MQTT、计算等）
- 搜索和筛选
- 显示当前值和数据类型
- 快速复制路径

### 3. DataSourcePanel

**文件**: `designer/src/components/panels/DataSourcePanel.vue`

数据源配置面板，配置页面级数据源。

**功能**：

- 添加/编辑/删除数据源
- 选择数据点
- 配置获取模式和间隔
- 查看数据源状态

### 4. DataBindingPanel

**文件**: `designer/src/components/panels/DataBindingPanel.vue`

数据绑定配置面板，将数据绑定到组件属性。

**功能**：

- 添加/删除绑定
- 编辑绑定表达式
- 实时预览绑定值
- 快速选择数据源

## 后端 API

### 数据点 API

| 方法 | 路由 | 说明 |
|------|------|------|
| GET | `/api/v1/data/projects/:projectId/datapoints` | 获取数据点列表 |
| GET | `/api/v1/data/projects/:projectId/datapoints/value` | 获取数据点值（支持批量） |

### Socket.IO 事件

| 事件 | 方向 | 说明 |
|------|------|------|
| `datapoint:subscribe` | Client → Server | 订阅数据点 |
| `datapoint:unsubscribe` | Client → Server | 取消订阅 |
| `datapoint:value` | Server → Client | 推送数据点值变化 |

### 请求示例

```javascript
// 获取数据点列表
GET /api/v1/data/projects/xxx/datapoints?type=mqtt.tag&status=active

// 获取单个数据点值
GET /api/v1/data/projects/xxx/datapoints/value?path=mqtt.EMQX.温度组.temperature

// Socket.IO 订阅
socket.emit("datapoint:subscribe", {
  projectId: "xxx",
  paths: ["mqtt.EMQX.温度组.temperature", "mqtt.EMQX.温度组.humidity"],
});

// 接收推送
socket.on("datapoint:value", (data) => {
  // { path: 'mqtt.EMQX.温度组.temperature', value: 25.6, timestamp: 1704412800000 }
});
```

## 配置说明

### 数据源配置结构

```typescript
interface DataSourceConfig {
  id: string; // 数据源 ID
  path: string; // 数据点路径
  dataType: "number" | "string" | "boolean" | "object" | "array"; // 数据类型
  mode: "request" | "poll" | "subscription"; // 获取模式
  interval?: number; // 轮询间隔（mode=poll 时）
  transformer?: string; // 数据转换脚本（object/array 类型时常用）
  errorHandler?: string; // 错误处理函数
}

// 辅助数据源类型
interface StaticDataSource {
  id: string;
  type: "static";
  data: any;
}

interface ComputedDataSource {
  id: string;
  type: "computed";
  dependencies: string[];
  compute: string;
}
```

### 数据类型说明

| 数据类型  | 说明       | 处理方式                         |
| --------- | ---------- | -------------------------------- |
| `number`  | 数值类型   | 直接使用，支持数学运算           |
| `string`  | 字符串类型 | 直接使用，支持字符串操作         |
| `boolean` | 布尔类型   | 直接使用，支持条件判断           |
| `object`  | 对象类型   | 可通过 transformer 脚本解析字段  |
| `array`   | 数组类型   | 可通过 transformer 处理或直接绑定列表组件 |

### 绑定配置结构

```typescript
interface ComponentBindings {
  [propertyPath: string]: string; // 属性路径 → 表达式
}

// 示例
{
  "props.value": "{{ data.ds_temp }}",
  "props.title": "{{ '温度: ' + data.ds_temp }}",
  "style.color": "{{ data.ds_temp > 80 ? 'red' : 'green' }}"
}
```

## 最佳实践

### 1. 选择合适的获取模式

```javascript
// 实时数据（变化频繁）
{ "mode": "subscription" }

// 统计数据（变化较慢）
{ "mode": "poll", "interval": 10000 }

// 配置数据（几乎不变）
{ "mode": "request" }
```

### 2. 合理设置轮询间隔

```javascript
// ❌ 间隔太短，增加服务器压力
{ "interval": 500 }

// ✅ 根据数据变化频率设置
{ "interval": 5000 }  // 统计数据
{ "interval": 30000 } // 配置数据
```

### 3. 使用 Computed 优化性能

```javascript
// 将复杂计算抽取为 computed 数据源
{
  "id": "ds_summary",
  "type": "computed",
  "config": {
    "dependencies": ["ds_list"],
    "compute": "(sources) => sources.ds_list.filter(d => d.active).length"
  }
}
```

### 4. 错误处理

```javascript
{
  "errorHandler": "(error) => ({ value: 0, error: error.message })"
}
```

## 与数据中心的关系

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                                                                             │
│   数据中心（DataCenter）                    设计器（Designer）              │
│   ════════════════════                    ════════════════════              │
│                                                                             │
│   负责：                                   负责：                           │
│   • 连接管理（数据库/MQTT/OPC UA）         • 页面设计                       │
│   • 查询/变量配置                          • 数据源配置（绑定数据点）        │
│   • 数据点自动生成                         • 组件属性绑定                   │
│   • 数据点统一管理                         • 预览与运行                     │
│                                                                             │
│   ┌─────────────────────┐                ┌─────────────────────┐           │
│   │                     │                │                     │           │
│   │  创建查询 ──────────┼───► 自动生成 ──┼───► 绑定数据点      │           │
│   │  创建变量           │    数据点      │                     │           │
│   │                     │                │                     │           │
│   └─────────────────────┘                └─────────────────────┘           │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

**职责分离**：

- **数据中心**：负责数据的「采集、解析、存储、推送」
- **设计器**：负责数据的「绑定、展示、交互」

**数据点**是两者之间的桥梁，由数据中心自动生成和管理，供设计器引用使用。

## 相关文档

- [数据绑定完整指南](./data-binding.md)
- [数据点方案设计](../datacenter/datapoint-design.md)
- [数据点改造实施](../datacenter/datapoint-implementation.md)
- [设计中心概述](./README.md)
