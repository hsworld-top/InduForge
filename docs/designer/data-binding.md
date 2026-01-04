# 数据绑定系统 - 完整指南

## 概述

InduForge Designer的数据绑定系统提供了强大的数据管理和绑定能力，支持与DataCenter深度集成，实现实时数据更新和可视化配置。

## 核心特性

### ✨ 多种数据源类型
- **DataCenter**: 与数据中心集成，支持查询和点位订阅
- **HTTP**: REST API调用，支持各种HTTP方法
- **Static**: 静态数据，适用于配置和字典
- **Computed**: 计算数据源，基于其他数据源计算

### 🔄 灵活的数据获取模式
- **Request**: 单次请求，按需加载
- **Poll**: 轮询模式，定时刷新
- **Subscription**: 订阅模式，实时推送

### 🎯 强大的数据绑定
- 支持组件属性绑定（props.*）
- 支持组件样式绑定（style.*）
- 表达式语法，支持复杂计算
- 实时预览绑定值

### 🛠️ 可视化配置
- 数据源配置面板
- 数据绑定配置面板
- 实时状态监控
- 错误提示和降级

## 快速开始

### 1. 安装依赖

数据绑定系统已集成到Designer中，无需额外安装。

### 2. 基本使用

```javascript
// 1. 在页面配置中添加数据源
{
  "dataSources": [
    {
      "id": "ds_devices",
      "type": "dataCenter",
      "config": {
        "sourceType": "query",
        "queryId": "query_devices"
      },
      "mode": "poll",
      "interval": 5000
    }
  ]
}

// 2. 在组件中绑定数据
{
  "id": "comp_table",
  "type": "Table",
  "bindings": {
    "props.data": "{{ data.ds_devices }}"
  }
}
```

### 3. 完整示例

参考本文档的架构与核心模块章节完成配置。

## 架构设计

### 系统架构（API模式 - 默认）

```
┌─────────────────────────────────────────────────────────────┐
│                    IDE (标签页容器)                          │
│  ┌──────────────────┐         ┌──────────────────┐          │
│  │   Designer Tab   │         │ DataCenter Tab   │          │
│  │                  │         │  (可选打开)       │          │
│  │  ┌────────────┐  │         │                  │          │
│  │  │DataSourceMgr│ │         │                  │          │
│  │  │     ↓       │ │         │                  │          │
│  │  │ 直接API调用 │ │         │                  │          │
│  │  └────────────┘  │         │                  │          │
│  └──────────────────┘         └──────────────────┘          │
└─────────────────────────────────────────────────────────────┘
                ↓ HTTP Request
┌─────────────────────────────────────────────────────────────┐
│                    Backend API (dev_core)                    │
│  - /api/v1/data/projects/:id/connections                    │
│  - /api/v1/data/projects/:id/queries                        │
│  - /api/v1/data/queries/:id/execute                         │
│  - /api/v1/data/.../execute-sql                             │
└─────────────────────────────────────────────────────────────┘
                ↓
┌─────────────────────────────────────────────────────────────┐
│                      Database                                │
└─────────────────────────────────────────────────────────────┘
```

**优势**：
- ✅ 不依赖DataCenter标签页
- ✅ Designer和DataCenter完全解耦
- ✅ 可以独立开发和测试
- ✅ 更简单的通信机制
- ✅ 更好的性能和可靠性

详细架构说明请参考 [数据绑定架构](./data-binding-architecture.md)

### 数据流

```
配置 → 注册 → 启动 → 获取数据 → 转换 → 缓存 → 绑定 → 更新组件
```

## 核心模块

### 1. DataCenterBridge
**文件**: `src/engine/datasource/DataCenterBridge.js`

iframe通信桥接，负责与DataCenter的消息通信。

**主要方法**:
- `init(iframeId)` - 初始化连接
- `executeQuery(projectId, queryId, parameters)` - 执行查询
- `executeSql(projectId, connectionId, sql, parameters)` - 执行SQL
- `subscribe(projectId, queryId, parameters, onData, onError, interval)` - 订阅数据

### 2. DataSourceManager
**文件**: `src/engine/datasource/DataSourceManager.js`

数据源管理器，负责数据源的生命周期管理。

**主要方法**:
- `register(dataSourceConfig)` - 注册数据源
- `start(dataSourceId)` - 启动数据源
- `stop(dataSourceId)` - 停止数据源
- `refresh(dataSourceId)` - 刷新数据源
- `getData(dataSourceId)` - 获取数据
- `getStatus(dataSourceId)` - 获取状态

### 3. MessageHandler
**文件**: `datacenter/src/utils/messageHandler.js`

DataCenter端的消息处理器，处理来自Designer的请求。

**支持的操作**:
- GET_CONNECTIONS - 获取连接列表
- GET_QUERIES - 获取查询列表
- EXECUTE_QUERY - 执行查询
- EXECUTE_SQL - 执行SQL
- SUBSCRIBE - 订阅数据
- UNSUBSCRIBE - 取消订阅

### 4. DataSourcePanel
**文件**: `src/components/panels/DataSourcePanel.vue`

数据源配置面板，提供可视化的数据源管理界面。

**功能**:
- 添加/编辑/删除数据源
- 查看数据源状态
- 刷新数据源
- 配置数据源参数

### 5. DataBindingPanel
**文件**: `src/components/panels/DataBindingPanel.vue`

数据绑定配置面板，提供可视化的数据绑定配置。

**功能**:
- 添加/删除绑定
- 编辑表达式
- 实时预览绑定值
- 快速选择数据源

### 6. VariablePanel
**文件**: `src/components/panels/VariablePanel.vue`

变量管理面板，用于维护页面变量。

**功能**:
- 新增/编辑/删除变量，编辑时会回填变量信息并更新原变量
- 变量重命名会同步更新页面事件、数据源配置与组件绑定引用

## API文档

### DataSourceConfig

```typescript
interface DataSourceConfig {
  id: string                    // 数据源ID
  type: 'dataCenter' | 'http' | 'static' | 'computed'
  config: {
    // DataCenter配置
    sourceType?: 'query' | 'tags'
    queryId?: string
    connectionId?: string
    tags?: string[]
    parameters?: Record<string, any>
    
    // HTTP配置
    url?: string
    method?: 'GET' | 'POST' | 'PUT' | 'DELETE'
    headers?: Record<string, string>
    params?: Record<string, any>
    data?: any
    
    // Static配置
    data?: any
    
    // Computed配置
    dependencies?: string[]
    compute?: string
  }
  mode?: 'request' | 'poll' | 'subscription'
  interval?: number             // 轮询间隔（毫秒）
  transformer?: string          // 数据转换函数
  errorHandler?: string         // 错误处理函数
  options?: {
    autoStart?: boolean         // 自动启动
    retryOnError?: boolean      // 错误重试
    retryCount?: number         // 重试次数
    retryInterval?: number      // 重试间隔
  }
}
```

### Binding

```typescript
interface Binding {
  [path: string]: string        // 属性路径 -> 表达式
}

// 示例
{
  "props.value": "{{ data.ds_temp.value }}",
  "style.color": "{{ data.ds_temp.value > 80 ? '#ff4d4f' : '#52c41a' }}"
}
```

### Expression Context

```typescript
interface ExpressionContext {
  vars: Record<string, any>     // 页面变量
  data: Record<string, any>     // 数据源数据
  props: Record<string, any>    // 组件属性
  $user: {                      // 用户信息
    id: string
    name: string
    role: string
    permissions: string[]
  }
  $route: {                     // 路由信息
    params: Record<string, any>
    query: Record<string, any>
  }
  $env: {                       // 环境变量
    API_BASE: string
    MODE: string
  }
  $global: Record<string, any>  // 全局变量
}
```

## 表达式语法

### 基本语法

```javascript
// 访问数据源
{{ data.ds_devices }}

// 访问变量
{{ vars.filterStatus }}

// 访问属性
{{ props.title }}

// 条件表达式
{{ data.ds_temp.value > 80 ? '高温' : '正常' }}

// 数组操作
{{ data.ds_devices.filter(d => d.status === 1) }}

// 对象操作
{{ data.ds_device.name + ' - ' + data.ds_device.status }}

// 函数调用
{{ Math.round(data.ds_temp.value * 100) / 100 }}
```

### 内置函数

```javascript
// 格式化
{{ $format.number(value, 2) }}
{{ $format.date(date, 'YYYY-MM-DD') }}

// 数组操作
{{ $array.sum(arr, 'field') }}
{{ $array.avg(arr, 'field') }}
{{ $array.max(arr, 'field') }}

// 条件判断
{{ $if(condition, trueVal, falseVal) }}
```

## 使用场景

### 场景1: 实时监控大屏

```json
{
  "dataSources": [
    {
      "id": "ds_realtime_data",
      "type": "dataCenter",
      "config": {
        "sourceType": "tags",
        "connectionId": "conn_plc_01",
        "tags": ["temp", "pressure", "flow"]
      },
      "mode": "subscription",
      "interval": 1000
    }
  ],
  "components": [
    {
      "id": "comp_temp",
      "type": "Gauge",
      "bindings": {
        "props.value": "{{ data.ds_realtime_data.temp }}",
        "props.color": "{{ data.ds_realtime_data.temp > 80 ? '#ff4d4f' : '#52c41a' }}"
      }
    }
  ]
}
```

### 场景2: 数据报表

```json
{
  "dataSources": [
    {
      "id": "ds_report",
      "type": "dataCenter",
      "config": {
        "sourceType": "query",
        "queryId": "query_daily_report",
        "parameters": {
          "date": "{{ vars.selectedDate }}"
        }
      },
      "mode": "request"
    }
  ],
  "components": [
    {
      "id": "comp_table",
      "type": "Table",
      "bindings": {
        "props.data": "{{ data.ds_report }}"
      }
    }
  ]
}
```

### 场景3: 表单应用

```json
{
  "dataSources": [
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
  ],
  "components": [
    {
      "id": "comp_select",
      "type": "Select",
      "bindings": {
        "props.options": "{{ data.ds_options }}"
      }
    }
  ]
}
```

## 性能优化

### 1. 合理设置轮询间隔
```javascript
// 不推荐：间隔太短
{ "interval": 500 }

// 推荐：根据实际需求设置
{ "interval": 5000 }  // 5秒
```

### 2. 使用数据转换器
```javascript
// 减少数据量
{
  "transformer": "(data) => data.slice(0, 100)"
}

// 提取需要的字段
{
  "transformer": "(data) => data.map(d => ({ id: d.id, name: d.name }))"
}
```

### 3. 使用计算数据源
```javascript
// 缓存计算结果
{
  "id": "ds_summary",
  "type": "computed",
  "config": {
    "dependencies": ["ds_devices"],
    "compute": "(sources) => ({ total: sources.ds_devices.length })"
  }
}
```

### 4. 避免复杂表达式
```javascript
// 不推荐：在绑定中进行复杂计算
{
  "bindings": {
    "props.value": "{{ data.ds_devices.filter(d => d.status === 1).map(d => d.value).reduce((a, b) => a + b, 0) }}"
  }
}

// 推荐：使用计算数据源
{
  "dataSources": [
    {
      "id": "ds_total",
      "type": "computed",
      "config": {
        "dependencies": ["ds_devices"],
        "compute": "(sources) => sources.ds_devices.filter(d => d.status === 1).map(d => d.value).reduce((a, b) => a + b, 0)"
      }
    }
  ],
  "bindings": {
    "props.value": "{{ data.ds_total }}"
  }
}
```

## 调试指南

### 1. 查看数据源状态

```javascript
// 在浏览器控制台
const store = useDesignStore()
const dataSources = Array.from(store.dataSourceManager.dataSources.values())
console.log(dataSources)
```

### 2. 查看数据源数据

```javascript
console.log(store.dataSources)
```

### 3. 手动刷新数据源

```javascript
await store.dataSourceManager.refresh('ds_devices')
```

### 4. 监听消息通信

```javascript
window.addEventListener('message', (event) => {
  console.log('Message:', event.data)
})
```

## 常见问题

### Q: 数据源状态一直是"加载中"？
A: 检查DataCenter是否加载完成，查询ID是否存在，网络请求是否成功。

### Q: 数据绑定不生效？
A: 检查表达式语法，确认数据源状态为"就绪"，查看实时预览。

### Q: 轮询不工作？
A: 确认模式为"轮询"，检查间隔设置，手动刷新测试。

### Q: 性能问题？
A: 增加轮询间隔，使用数据转换器，优化表达式，使用计算数据源。

## 文档索引

- [数据绑定架构](./data-binding-architecture.md)
- [设计中心概述](./README.md)
- [DSL 设计规范](../dsl-design.md)
- [迁移计划](../migration-plan.md)

## 贡献指南

欢迎贡献代码和文档！请遵循以下规范：

1. 代码风格：遵循ESLint规则
2. 提交信息：使用语义化提交信息
3. 测试：添加单元测试
4. 文档：更新相关文档

## 许可证

ISC

## 联系方式

如有问题或建议，请联系开发团队。
