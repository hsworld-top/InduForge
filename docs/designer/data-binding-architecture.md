# 数据绑定系统架构说明

## 架构模式

InduForge的数据绑定系统支持两种架构模式，以适应不同的部署场景：

### 模式1: API模式（默认，推荐）

**适用场景**：
- IDE使用标签页方式打开Designer和DataCenter
- Designer和DataCenter独立开发和部署
- 需要更好的解耦和独立性

**架构图**：
```
┌─────────────────────────────────────────────────────────────┐
│                    IDE (标签页容器)                          │
│  ┌──────────────────┐         ┌──────────────────┐          │
│  │   Designer Tab   │         │ DataCenter Tab   │          │
│  │                  │         │                  │          │
│  │  DataSourceMgr   │         │  (可选打开)       │          │
│  │       ↓          │         │                  │          │
│  │   直接API调用     │         │                  │          │
│  └──────────────────┘         └──────────────────┘          │
└─────────────────────────────────────────────────────────────┘
                ↓ HTTP Request
┌─────────────────────────────────────────────────────────────┐
│                    Backend API (dev_core)                    │
│  ┌────────────────────────────────────────────────────────┐ │
│  │  /api/v1/data/projects/:id/connections                 │ │
│  │  /api/v1/data/projects/:id/queries                     │ │
│  │  /api/v1/data/queries/:id/execute                      │ │
│  │  /api/v1/data/projects/:id/connections/:id/execute-sql │ │
│  └────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
                ↓
┌─────────────────────────────────────────────────────────────┐
│                      Database                                │
│  MySQL / PostgreSQL / SQL Server                             │
└─────────────────────────────────────────────────────────────┘
```

**优点**：
- ✅ 不依赖DataCenter标签页是否打开
- ✅ Designer和DataCenter完全解耦
- ✅ 可以独立开发和测试
- ✅ 更简单的通信机制
- ✅ 更好的错误处理和重试
- ✅ 支持标准的HTTP缓存

**缺点**：
- ❌ 需要后端API支持
- ❌ 无法利用DataCenter的UI功能（如查询编辑器）

### 模式2: Bridge模式（可选）

**适用场景**：
- Designer通过iframe嵌入DataCenter
- 需要利用DataCenter的UI功能
- 需要更紧密的集成

**架构图**：
```
┌─────────────────────────────────────────────────────────────┐
│                      Designer                                │
│  ┌────────────────────────────────────────────────────────┐ │
│  │              DataSourceManager                          │ │
│  │                    ↓                                    │ │
│  │            DataCenterBridge                             │ │
│  │                    ↓                                    │ │
│  │              postMessage                                │ │
│  └────────────────────────────────────────────────────────┘ │
│  ┌────────────────────────────────────────────────────────┐ │
│  │         <iframe id="datacenter-iframe">                 │ │
│  │                                                          │ │
│  │              DataCenter                                  │ │
│  │         ┌──────────────────┐                            │ │
│  │         │ MessageHandler   │                            │ │
│  │         │       ↓          │                            │ │
│  │         │   Data API       │                            │ │
│  │         └──────────────────┘                            │ │
│  └────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
                ↓ HTTP Request
┌─────────────────────────────────────────────────────────────┐
│                    Backend API                               │
└─────────────────────────────────────────────────────────────┘
```

**优点**：
- ✅ 可以利用DataCenter的UI功能
- ✅ 更紧密的集成
- ✅ 可以共享DataCenter的状态

**缺点**：
- ❌ 依赖DataCenter iframe加载完成
- ❌ 更复杂的通信机制
- ❌ 跨域和安全性问题
- ❌ 调试更困难

## 默认配置

系统默认使用**API模式**，因为它更适合标签页架构，且不依赖iframe。

```javascript
// 在 store/design.js 中
async initDataSourceManager(options = {}) {
  // 默认使用API模式
  const mode = options.mode || 'api'
  dataSourceManager = new DataSourceManager(this, { mode })
  // ...
}
```

## 切换模式

如果需要使用Bridge模式（例如，Designer通过iframe嵌入DataCenter），可以在初始化时指定：

```javascript
// 使用Bridge模式
await store.initDataSourceManager({ 
  mode: 'bridge',
  iframeId: 'datacenter-iframe'
})
```

## API模式实现细节

### 1. 数据源管理器

```javascript
// DataSourceManager.js
export class DataSourceManager {
  constructor(store, options = {}) {
    this.mode = options.mode || 'api' // 默认API模式
    // ...
  }
  
  // 执行查询
  async executeQueryAPI(projectId, queryId, parameters) {
    if (this.mode === 'bridge') {
      return await this.bridge.executeQuery(projectId, queryId, parameters)
    } else {
      // 直接调用后端API
      const response = await request({
        url: `/data/queries/${queryId}/execute`,
        method: 'post',
        data: { parameters }
      })
      return response.data || response
    }
  }
}
```

### 2. 请求工具

使用Designer的统一请求工具（`utils/request.js`），自动处理：
- 认证token
- 租户ID
- 错误处理
- 请求重试

```javascript
import request from '@/utils/request'

// 执行查询
const response = await request({
  url: `/data/queries/${queryId}/execute`,
  method: 'post',
  data: { parameters }
})
```

### 3. 后端API

后端需要提供以下API端点：

```
GET    /api/v1/data/projects/:projectId/connections
POST   /api/v1/data/projects/:projectId/connections
GET    /api/v1/data/projects/:projectId/queries
POST   /api/v1/data/projects/:projectId/queries
POST   /api/v1/data/queries/:queryId/execute
POST   /api/v1/data/projects/:projectId/connections/:connectionId/execute-sql
GET    /api/v1/data/projects/:projectId/connections/:connectionId/tables
GET    /api/v1/data/projects/:projectId/connections/:connectionId/tables/:tableName/data
```

## 数据流对比

### API模式数据流

```
1. 用户配置数据源
   ↓
2. DataSourcePanel → Store → DataSourceManager.register()
   ↓
3. DataSourceManager.start()
   ↓
4. DataSourceManager.executeQueryAPI() → request() → Backend API
   ↓
5. Backend → Database → 返回数据
   ↓
6. DataSourceManager.updateData() → Store.dataSources
   ↓
7. DataBinder监听变化 → 更新组件属性
   ↓
8. Component响应式更新
```

### Bridge模式数据流

```
1. 用户配置数据源
   ↓
2. DataSourcePanel → Store → DataSourceManager.register()
   ↓
3. DataSourceManager.start()
   ↓
4. DataCenterBridge.request() → postMessage → iframe
   ↓
5. MessageHandler.handleRequest() → DataAPI → Backend
   ↓
6. Backend → Database → 返回数据
   ↓
7. MessageHandler → postMessage → DataCenterBridge
   ↓
8. DataSourceManager.updateData() → Store.dataSources
   ↓
9. DataBinder监听变化 → 更新组件属性
   ↓
10. Component响应式更新
```

## 性能对比

| 指标 | API模式 | Bridge模式 |
|------|---------|-----------|
| 请求延迟 | 低（直接HTTP） | 中（postMessage + HTTP） |
| 通信开销 | 小 | 大（消息序列化） |
| 错误处理 | 简单 | 复杂 |
| 调试难度 | 低 | 高 |
| 依赖性 | 仅后端 | 后端 + iframe |

## 兼容性

### API模式兼容性
- ✅ 所有现代浏览器
- ✅ 标签页架构
- ✅ iframe架构（如果需要）
- ✅ 独立部署

### Bridge模式兼容性
- ✅ 所有现代浏览器（支持postMessage）
- ❌ 标签页架构（无法通信）
- ✅ iframe架构
- ❌ 独立部署（需要同域）

## 迁移指南

### 从Bridge模式迁移到API模式

1. **更新初始化代码**：
```javascript
// 旧代码（Bridge模式）
await store.initDataSourceManager({ 
  mode: 'bridge',
  iframeId: 'datacenter-iframe'
})

// 新代码（API模式）
await store.initDataSourceManager() // 默认API模式
```

2. **移除iframe依赖**：
```html
<!-- 可以移除DataCenter iframe -->
<!-- <iframe id="datacenter-iframe" src="/datacenter"></iframe> -->
```

3. **确保后端API可用**：
确保后端提供所有必需的API端点。

4. **测试数据源**：
测试所有数据源是否正常工作。

### 从API模式迁移到Bridge模式

1. **添加iframe**：
```html
<iframe id="datacenter-iframe" src="/datacenter"></iframe>
```

2. **更新初始化代码**：
```javascript
await store.initDataSourceManager({ 
  mode: 'bridge',
  iframeId: 'datacenter-iframe'
})
```

3. **初始化MessageHandler**：
在DataCenter的main.js中：
```javascript
import { initMessageHandler } from './utils/messageHandler'
initMessageHandler()
```

## 最佳实践

### 1. 使用API模式（推荐）

对于标签页架构，始终使用API模式：
```javascript
// 默认配置，无需额外参数
await store.initDataSourceManager()
```

### 2. 错误处理

API模式提供更好的错误处理：
```javascript
{
  "errorHandler": "(error) => ({ value: 0, status: 'error', message: error.message })"
}
```

### 3. 缓存策略

利用HTTP缓存减少请求：
```javascript
// 使用较长的轮询间隔
{
  "mode": "poll",
  "interval": 10000 // 10秒
}
```

### 4. 降级策略

如果Bridge模式初始化失败，自动降级到API模式：
```javascript
async initDataSourceManager(options = {}) {
  const mode = options.mode || 'api'
  dataSourceManager = new DataSourceManager(this, { mode })
  
  const success = await dataSourceManager.init(options.iframeId)
  if (!success && mode === 'bridge') {
    // 降级到API模式
    dataSourceManager = new DataSourceManager(this, { mode: 'api' })
    await dataSourceManager.init()
  }
}
```

## 总结

- **API模式**是默认和推荐的模式，适用于标签页架构
- **Bridge模式**是可选模式，适用于iframe架构
- 两种模式可以无缝切换，无需修改数据源配置
- API模式提供更好的性能、可靠性和可维护性

## 相关文档

- [快速开始指南](./PHASE4_QUICKSTART.md)
- [完整指南](./DATA_BINDING_README.md)
- [阶段四完成报告](./PHASE4_COMPLETED.md)
