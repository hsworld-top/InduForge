# WebSocket 协议文档

本文档描述 InduForge 平台的 WebSocket 通信协议，基于 Socket.IO 实现。

## 1. 概述

### 1.1 技术选型

- **库**: Socket.IO
- **传输**: WebSocket / HTTP Long Polling（降级）
- **路径**: `/socket.io`
- **端口**: 与 HTTP 服务同端口（18101）

### 1.2 连接架构

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         WebSocket 连接架构                                   │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐       │
│  │   DataCenter    │     │    Designer     │     │  RuntimeEngine  │       │
│  │    (前端)       │     │    (前端)       │     │    (运行时)     │       │
│  └────────┬────────┘     └────────┬────────┘     └────────┬────────┘       │
│           │                       │                       │                 │
│           │   Socket.IO 连接      │                       │                 │
│           └───────────────────────┼───────────────────────┘                 │
│                                   │                                         │
│                                   ▼                                         │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                      SocketService (dev_core)                        │   │
│  │                                                                      │   │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐               │   │
│  │  │ 房间管理      │  │ 消息广播     │  │ 事件处理     │               │   │
│  │  │              │  │              │  │              │               │   │
│  │  │ project:xxx  │  │ mqtt:message │  │ subscribe   │               │   │
│  │  │ mqtt:sub:xxx │  │ tag:value    │  │ unsubscribe │               │   │
│  │  │ mqtt:tag:xxx │  │ status       │  │ disconnect  │               │   │
│  │  │ datapoint:xx │  │              │  │              │               │   │
│  │  └──────────────┘  └──────────────┘  └──────────────┘               │   │
│  │                                                                      │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

## 2. 连接建立

### 2.1 客户端连接

```javascript
import { io } from 'socket.io-client'

const socket = io('http://localhost:18101', {
  path: '/socket.io',
  transports: ['websocket', 'polling'],
  query: {
    projectId: 'proj_xxx', // 项目 ID（可选）
  },
})

// 连接成功
socket.on('connect', () => {
  console.log('Connected:', socket.id)
})

// 连接断开
socket.on('disconnect', (reason) => {
  console.log('Disconnected:', reason)
})

// 连接错误
socket.on('error', (error) => {
  console.error('Socket error:', error)
})
```

### 2.2 服务端配置

```javascript
const { Server } = require('socket.io')

const io = new Server(httpServer, {
  cors: {
    origin: process.env.CORS_ORIGINS || '*',
    methods: ['GET', 'POST'],
    credentials: true,
  },
  path: '/socket.io',
  transports: ['websocket', 'polling'],
})
```

## 3. 房间（Rooms）设计

### 3.1 房间类型

| 房间名格式                           | 用途               | 加入方式       |
| ------------------------------------ | ------------------ | -------------- |
| `project:{projectId}`                | 项目级广播         | 连接时自动加入 |
| `mqtt:subscription:{subscriptionId}` | MQTT 订阅消息      | 客户端订阅     |
| `mqtt:tag:{tagId}`                   | Tag 值更新         | 客户端订阅     |
| `datapoint:{projectId}:{path}`       | 数据点值更新       | 客户端订阅     |
| `node:{nodeId}`                      | 节点状态（待实现） | NodeAgent 连接 |
| `designer:{projectId}`               | Designer 协作事件  | Designer 进入  |

### 3.2 房间生命周期

```
连接建立
    │
    ├─ 自动加入 project:{projectId}（如果提供）
    │
    ├─ 客户端发送 subscribe 事件
    │       │
    │       └─ 加入对应房间
    │
    ├─ 客户端发送 unsubscribe 事件
    │       │
    │       └─ 离开对应房间
    │
    └─ 断开连接
            │
            └─ 自动离开所有房间
```

## 4. 事件定义

### 4.1 客户端 → 服务端事件

#### mqtt:subscribe

订阅 MQTT 消息。

```typescript
// 请求
socket.emit('mqtt:subscribe', {
  subscriptionId: string;  // 订阅 ID
});

// 示例
socket.emit('mqtt:subscribe', {
  subscriptionId: 'sub_xxx'
});
```

#### mqtt:unsubscribe

取消订阅 MQTT 消息。

```typescript
// 请求
socket.emit('mqtt:unsubscribe', {
  subscriptionId: string;
});
```

#### mqtt:tag:subscribe

订阅 Tag 值更新。

```typescript
// 请求
socket.emit('mqtt:tag:subscribe', {
  tagId: string;
});

// 示例
socket.emit('mqtt:tag:subscribe', {
  tagId: 'tag_xxx'
});
```

#### mqtt:tag:unsubscribe

取消订阅 Tag 值更新。

```typescript
// 请求
socket.emit('mqtt:tag:unsubscribe', {
  tagId: string;
});
```

#### datapoint:subscribe

订阅数据点值更新。

```typescript
// 请求
socket.emit('datapoint:subscribe', {
  projectId: string;
  paths: string[];  // 数据点路径数组
});

// 示例
socket.emit('datapoint:subscribe', {
  projectId: 'proj_xxx',
  paths: [
    'mqtt.EMQX.温度组.temperature',
    'mqtt.EMQX.温度组.humidity',
    'db.MySQL生产库.实时数据.pressure'
  ]
});
```

#### datapoint:unsubscribe

取消订阅数据点值更新。

```typescript
// 请求
socket.emit('datapoint:unsubscribe', {
  projectId: string;
  paths: string[];
});
```

#### designer:join

Designer 加入工程（用于接收协作事件）。

```typescript
// 请求
socket.emit('designer:join', {
  projectId: string;
});

// 服务端处理：加入 designer:{projectId} 房间
```

#### designer:leave

Designer 离开工程。

```typescript
// 请求
socket.emit('designer:leave', {
  projectId: string;
});
```

### 4.2 服务端 → 客户端事件

#### mqtt:message

MQTT 消息推送。

```typescript
// 数据结构
interface MqttMessageEvent {
  subscriptionId: string
  message: {
    topic: string
    payload: string | object
    qos: 0 | 1 | 2
    timestamp: number
  }
}

// 示例
socket.on('mqtt:message', (data) => {
  console.log('MQTT Message:', data)
  // {
  //   subscriptionId: 'sub_xxx',
  //   message: {
  //     topic: 'sensors/temperature',
  //     payload: { value: 25.5, unit: '°C' },
  //     qos: 0,
  //     timestamp: 1704614400000
  //   }
  // }
})
```

#### mqtt:connection:status

MQTT 连接状态变化。

```typescript
// 数据结构
interface ConnectionStatusEvent {
  connectionId: string
  status: 'connected' | 'disconnected' | 'connecting' | 'error'
  timestamp: number
  error?: string
}

// 示例
socket.on('mqtt:connection:status', (data) => {
  console.log('Connection Status:', data)
  // {
  //   connectionId: 'conn_xxx',
  //   status: 'connected',
  //   timestamp: 1704614400000
  // }
})
```

#### mqtt:subscription:status

MQTT 订阅状态变化。

```typescript
// 数据结构
interface SubscriptionStatusEvent {
  subscriptionId: string
  status: 'active' | 'inactive' | 'error'
  timestamp: number
  error?: string
}

// 示例
socket.on('mqtt:subscription:status', (data) => {
  console.log('Subscription Status:', data)
})
```

#### mqtt:tag:value

Tag 值更新推送。

```typescript
// 数据结构
interface TagValueEvent {
  tagId: string
  value: any
  quality: 'good' | 'bad' | 'uncertain'
  timestamp: number
  source?: string
}

// 示例
socket.on('mqtt:tag:value', (data) => {
  console.log('Tag Value:', data)
  // {
  //   tagId: 'tag_xxx',
  //   value: 25.5,
  //   quality: 'good',
  //   timestamp: 1704614400000
  // }
})
```

#### datapoint:value

数据点值更新推送（待实现）。

```typescript
// 数据结构
interface DataPointValueEvent {
  projectId: string
  path: string
  value: any
  dataType: string
  quality: 'good' | 'bad' | 'uncertain'
  timestamp: number
}

// 示例
socket.on('datapoint:value', (data) => {
  console.log('DataPoint Value:', data)
  // {
  //   projectId: 'proj_xxx',
  //   path: 'mqtt.EMQX.温度组.temperature',
  //   value: 25.5,
  //   dataType: 'number',
  //   quality: 'good',
  //   timestamp: 1704614400000
  // }
})
```

### 4.3 Designer 协作事件

#### page:lock:changed

页面锁状态变化广播（广播到 `designer:{projectId}` 房间）。

```typescript
// 数据结构
interface PageLockChangedEvent {
  pageId: string
  pageName: string
  locked: boolean
  lockedBy?: string // 用户 ID
  lockedByName?: string // 用户姓名
  lockedAt?: number // 锁定时间戳
}

// 示例
socket.on('page:lock:changed', (data) => {
  console.log('Page Lock Changed:', data)
  // {
  //   pageId: 'page_xxx',
  //   pageName: '首页',
  //   locked: true,
  //   lockedBy: 'user_001',
  //   lockedByName: '张三',
  //   lockedAt: 1704614400000
  // }

  if (data.locked && data.lockedBy !== currentUserId) {
    // 提示用户页面已被锁定
    showNotification(`页面 "${data.pageName}" 正在被 ${data.lockedByName} 编辑`)
  }
})
```

#### page:lock:force_release

页面锁被强制释放通知（发送给原锁定者）。

```typescript
// 数据结构
interface PageLockForceReleaseEvent {
  pageId: string
  pageName: string
  releasedBy: string // 执行强制释放的管理员 ID
  releasedByName: string
  reason: 'timeout' | 'admin_force' | 'user_logout'
}

// 示例
socket.on('page:lock:force_release', (data) => {
  console.log('Page Lock Force Released:', data)
  // 提示用户并切换到只读模式
  showWarning(`你在页面 "${data.pageName}" 的编辑权限已被释放，原因：${data.reason}`)
  setReadonlyMode(true)
})
```

#### designer:user:presence

用户在线状态变化（可选功能，用于显示谁在查看工程）。

```typescript
// 数据结构
interface UserPresenceEvent {
  projectId: string
  users: {
    userId: string
    userName: string
    avatar?: string
    currentPageId?: string
    status: 'viewing' | 'editing'
  }[]
}

// 示例
socket.on('designer:user:presence', (data) => {
  console.log('User Presence:', data)
  // 更新在线用户列表（可在 UI 上显示头像）
  updateOnlineUsers(data.users)
})
```

## 5. 节点通信协议（待实现）

### 5.1 NodeAgent → 服务端事件

#### node:register

节点注册。

```typescript
socket.emit('node:register', {
  registrationCode: string;  // 注册码
  nodeName: string;
  system: {
    os: string;
    arch: string;
    nodeAgentVersion: string;
  };
});
```

#### node:heartbeat

心跳上报。

```typescript
socket.emit('node:heartbeat', {
  nodeId: string;
  status: NodeStatus;  // 详见 publish-pipeline.md
});
```

#### node:deploy:result

部署结果上报。

```typescript
socket.emit('node:deploy:result', {
  deploymentId: string;
  success: boolean;
  error?: string;
  startedAt?: number;
});
```

#### node:log

日志上报。

```typescript
socket.emit('node:log', {
  nodeId: string;
  projectId?: string;
  level: 'debug' | 'info' | 'warn' | 'error';
  message: string;
  meta?: object;
  timestamp: number;
});
```

### 5.2 服务端 → NodeAgent 事件

#### node:deploy

部署指令。

```typescript
socket.on('node:deploy', (data) => {
  // {
  //   deploymentId: 'dep_xxx',
  //   ifpUrl: 'https://registry.example.com/projects/xxx/v1.0.0.ifp',
  //   config: {
  //     port: 8080,
  //     uiTarget: 'bigscreen',
  //     defaultLocale: 'zh-CN',
  //     defaultTheme: 'dark'
  //   }
  // }
})
```

#### node:stop

停止工程。

```typescript
socket.on('node:stop', (data) => {
  // { projectId: 'proj_xxx' }
})
```

#### node:restart

重启工程。

```typescript
socket.on('node:restart', (data) => {
  // { projectId: 'proj_xxx' }
})
```

#### node:rollback

回滚版本。

```typescript
socket.on('node:rollback', (data) => {
  // { projectId: 'proj_xxx', version: 'v1.1.0' }
})
```

## 6. 使用示例

### 6.1 DataCenter 订阅 MQTT 消息

```javascript
import { io } from 'socket.io-client'

class MqttSocketManager {
  constructor(projectId) {
    this.socket = io('http://localhost:18101', {
      query: { projectId },
    })
    this.messageHandlers = new Map()
  }

  // 订阅订阅消息
  subscribeSubscription(subscriptionId, handler) {
    this.socket.emit('mqtt:subscribe', { subscriptionId })

    const key = `subscription:${subscriptionId}`
    this.messageHandlers.set(key, handler)

    return () => {
      this.socket.emit('mqtt:unsubscribe', { subscriptionId })
      this.messageHandlers.delete(key)
    }
  }

  // 订阅 Tag 值
  subscribeTag(tagId, handler) {
    this.socket.emit('mqtt:tag:subscribe', { tagId })

    const key = `tag:${tagId}`
    this.messageHandlers.set(key, handler)

    return () => {
      this.socket.emit('mqtt:tag:unsubscribe', { tagId })
      this.messageHandlers.delete(key)
    }
  }

  // 初始化监听器
  init() {
    this.socket.on('mqtt:message', (data) => {
      const key = `subscription:${data.subscriptionId}`
      const handler = this.messageHandlers.get(key)
      if (handler) handler(data.message)
    })

    this.socket.on('mqtt:tag:value', (data) => {
      const key = `tag:${data.tagId}`
      const handler = this.messageHandlers.get(key)
      if (handler) handler(data)
    })

    this.socket.on('mqtt:connection:status', (data) => {
      console.log('Connection status changed:', data)
    })
  }

  disconnect() {
    this.socket.disconnect()
  }
}
```

### 6.2 Designer 预览订阅数据点

```javascript
class PreviewDataService {
  constructor(projectId) {
    this.projectId = projectId
    this.socket = io('http://localhost:18101', {
      query: { projectId },
    })
    this.subscribers = new Map()
  }

  // 订阅数据点
  subscribe(path, callback) {
    const paths = Array.isArray(path) ? path : [path]

    this.socket.emit('datapoint:subscribe', {
      projectId: this.projectId,
      paths,
    })

    paths.forEach((p) => {
      if (!this.subscribers.has(p)) {
        this.subscribers.set(p, new Set())
      }
      this.subscribers.get(p).add(callback)
    })

    // 返回取消订阅函数
    return () => {
      paths.forEach((p) => {
        const subs = this.subscribers.get(p)
        if (subs) {
          subs.delete(callback)
          if (subs.size === 0) {
            this.subscribers.delete(p)
          }
        }
      })

      this.socket.emit('datapoint:unsubscribe', {
        projectId: this.projectId,
        paths,
      })
    }
  }

  init() {
    this.socket.on('datapoint:value', (data) => {
      const subs = this.subscribers.get(data.path)
      if (subs) {
        subs.forEach((callback) => callback(data.value, data))
      }
    })
  }
}
```

### 6.3 NodeAgent 连接管理

```javascript
class NodeAgentSocket {
  constructor(nodeId, token) {
    this.nodeId = nodeId
    this.socket = io('http://dev-server:18101', {
      auth: { token },
      query: { nodeId },
    })
  }

  // 发送心跳
  sendHeartbeat(status) {
    this.socket.emit('node:heartbeat', {
      nodeId: this.nodeId,
      status,
    })
  }

  // 上报部署结果
  reportDeployResult(deploymentId, success, error) {
    this.socket.emit('node:deploy:result', {
      deploymentId,
      success,
      error,
      startedAt: success ? Date.now() : undefined,
    })
  }

  // 上报日志
  log(level, message, meta) {
    this.socket.emit('node:log', {
      nodeId: this.nodeId,
      level,
      message,
      meta,
      timestamp: Date.now(),
    })
  }

  init() {
    // 监听部署指令
    this.socket.on('node:deploy', async (data) => {
      try {
        await this.handleDeploy(data)
        this.reportDeployResult(data.deploymentId, true)
      } catch (error) {
        this.reportDeployResult(data.deploymentId, false, error.message)
      }
    })

    // 监听停止指令
    this.socket.on('node:stop', async (data) => {
      await this.handleStop(data.projectId)
    })

    // 监听重启指令
    this.socket.on('node:restart', async (data) => {
      await this.handleRestart(data.projectId)
    })

    // 监听回滚指令
    this.socket.on('node:rollback', async (data) => {
      await this.handleRollback(data.projectId, data.version)
    })
  }
}
```

## 7. 错误处理

### 7.1 连接错误

```javascript
socket.on('connect_error', (error) => {
  console.error('Connection error:', error.message)
  // 重连逻辑由 Socket.IO 自动处理
})

socket.on('reconnect', (attemptNumber) => {
  console.log('Reconnected after', attemptNumber, 'attempts')
})

socket.on('reconnect_error', (error) => {
  console.error('Reconnection error:', error.message)
})

socket.on('reconnect_failed', () => {
  console.error('Reconnection failed')
  // 通知用户或执行降级策略
})
```

### 7.2 业务错误

```javascript
// 服务端发送错误事件
socket.emit('error', {
  code: 'SUBSCRIPTION_NOT_FOUND',
  message: '订阅不存在',
  details: { subscriptionId: 'sub_xxx' },
})

// 客户端处理
socket.on('error', (error) => {
  console.error('Server error:', error)
  // 根据 error.code 处理不同错误
})
```

## 8. 安全考虑

### 8.1 认证

目前 WebSocket 连接不强制认证，建议后续增强：

```javascript
// 客户端携带 Token
const socket = io('http://localhost:18101', {
  auth: {
    token: localStorage.getItem('token'),
  },
})

// 服务端验证
io.use((socket, next) => {
  const token = socket.handshake.auth.token
  if (verifyToken(token)) {
    next()
  } else {
    next(new Error('Authentication failed'))
  }
})
```

### 8.2 房间访问控制

```javascript
// 验证项目访问权限
socket.on('mqtt:subscribe', async (data) => {
  const userId = socket.userId
  const hasAccess = await checkProjectAccess(userId, data.projectId)

  if (!hasAccess) {
    socket.emit('error', { code: 'ACCESS_DENIED' })
    return
  }

  socket.join(`mqtt:subscription:${data.subscriptionId}`)
})
```

## 9. 性能优化

### 9.1 消息压缩

```javascript
const io = new Server(httpServer, {
  perMessageDeflate: {
    threshold: 1024, // 只压缩大于 1KB 的消息
  },
})
```

### 9.2 消息节流

```javascript
// 服务端节流广播
const throttledBroadcast = throttle((room, event, data) => {
  io.to(room).emit(event, data)
}, 100) // 最小间隔 100ms
```

### 9.3 批量订阅

```javascript
// 客户端批量订阅
socket.emit('datapoint:subscribe', {
  projectId: 'proj_xxx',
  paths: paths, // 一次性订阅多个路径
})
```

---

**相关文档**：

- [后端 API](./README.md)
- [数据中心概览](../datacenter/README.md)
- [设计器概览](../designer/README.md)
- [详细设计](../详细设计.md)

---

**版本**: 1.0.0  
**创建日期**: 2026-01-07
