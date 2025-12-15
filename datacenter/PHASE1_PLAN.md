# DataCenter 阶段一开发计划

## 目标概述

实现 MQTT 连接的基础管理功能，包括连接的创建、配置、测试和状态管理。

**预计时间**：1-2 周  
**优先级**：P0（基础功能）

---

## 后端开发任务

### 1. 数据模型完善

#### 1.1 确认 data_mqtt_configs 表结构
**文件位置**：`dev_core/database/init.sql`

**检查项**：
- [ ] 表结构是否完整（brokerUrl, protocol, clientId, username, password 等）
- [ ] 外键约束是否正确（关联 data_connections 表）
- [ ] 索引是否合理

**如需修改**：
- 更新 `init.sql` 文件
- 创建数据库迁移脚本

#### 1.2 创建 Sequelize 模型
**文件位置**：`dev_core/src/models/DataMqttConfig.js`

**任务**：
- [ ] 创建 `DataMqttConfig` 模型
- [ ] 定义字段和数据类型
- [ ] 配置关联关系（belongsTo DataConnection）
- [ ] 在 `models/index.js` 中注册模型

**参考**：`DataRelationalConfig.js`

---

### 2. 协议实现

#### 2.1 实现 MqttProtocol 类
**文件位置**：`dev_core/src/services/protocols/MqttProtocol.js`

**依赖安装**：
```bash
cd dev_core
pnpm add mqtt
```

**实现方法**：
- [ ] `constructor(config)` - 初始化配置
- [ ] `async connect()` - 连接到 MQTT Broker
- [ ] `async disconnect()` - 断开连接
- [ ] `async testConnection()` - 测试连接
- [ ] 错误处理和日志记录

**关键点**：
- 使用 `mqtt.js` 库
- 支持 mqtt/mqtts/ws/wss 协议
- 处理连接超时
- 处理认证失败

**示例结构**：
```javascript
const mqtt = require('mqtt');
const BaseProtocol = require('./BaseProtocol');

class MqttProtocol extends BaseProtocol {
  constructor(config) {
    super(config);
    this.client = null;
  }

  async connect() {
    // 实现连接逻辑
  }

  async testConnection() {
    // 实现测试逻辑
  }

  async disconnect() {
    // 实现断开逻辑
  }
}

module.exports = MqttProtocol;
```

---

### 3. 服务层实现

#### 3.1 扩展 dataConnectionService
**文件位置**：`dev_core/src/services/dataConnectionService.js`

**任务**：
- [ ] 在 `testConnection` 方法中添加 MQTT 类型支持
- [ ] 实现 `testMqttConnection(config)` 方法
- [ ] 处理 MQTT 特定的错误

**实现思路**：
```javascript
async testConnection(type, config) {
  if (type === 'relational') {
    return await this.testRelationalConnection(config);
  } else if (type === 'mqtt') {
    return await this.testMqttConnection(config);
  }
  // ...
}

async testMqttConnection(config) {
  const protocol = new MqttProtocol(config);
  try {
    await protocol.testConnection();
    return { success: true, message: 'MQTT 连接测试成功' };
  } catch (error) {
    throw new AppError(ErrorCodes.CONNECTION_FAILED, 400, {
      message: `MQTT 连接失败: ${error.message}`
    });
  } finally {
    await protocol.disconnect();
  }
}
```

#### 3.2 创建 MqttConnectionService（可选）
**文件位置**：`dev_core/src/services/mqttConnectionService.js`

**任务**：
- [ ] 管理活动的 MQTT 连接
- [ ] 连接池管理
- [ ] 自动重连机制
- [ ] 健康检查

**说明**：此服务用于管理长连接，阶段一可以先实现基础版本

---

### 4. 控制器实现

#### 4.1 扩展 dataConnectionController
**文件位置**：`dev_core/src/controllers/dataConnectionController.js`

**任务**：
- [ ] 确保 MQTT 连接的 CRUD 操作正常工作
- [ ] 在创建连接时保存 MQTT 配置到 `data_mqtt_configs` 表
- [ ] 在更新连接时同步更新 MQTT 配置
- [ ] 在删除连接时级联删除 MQTT 配置

**关键代码位置**：
- `createConnection` 方法
- `updateConnection` 方法
- `testConnection` 方法

---

### 5. API 路由

#### 5.1 确认现有路由
**文件位置**：`dev_core/src/routes/data.js`

**检查项**：
- [ ] `POST /api/v1/data/projects/:projectId/connections` - 创建连接
- [ ] `GET /api/v1/data/projects/:projectId/connections` - 获取连接列表
- [ ] `PUT /api/v1/data/projects/:projectId/connections/:id` - 更新连接
- [ ] `DELETE /api/v1/data/projects/:projectId/connections/:id` - 删除连接
- [ ] `POST /api/v1/data/projects/:projectId/connections/test` - 测试连接

**说明**：这些路由应该已经存在，需要确保支持 MQTT 类型

---

### 6. 错误处理

#### 6.1 添加 MQTT 相关错误码
**文件位置**：`dev_core/src/constants/errorCodes.js`

**任务**：
- [ ] 添加 MQTT 连接失败错误码
- [ ] 添加 MQTT 认证失败错误码
- [ ] 添加 MQTT 超时错误码

**示例**：
```javascript
MQTT_CONNECTION_FAILED: 'MQTT_CONNECTION_FAILED',
MQTT_AUTH_FAILED: 'MQTT_AUTH_FAILED',
MQTT_TIMEOUT: 'MQTT_TIMEOUT',
```

---

## 前端开发任务

### 1. 连接类型配置

#### 1.1 添加 MQTT 配置
**文件位置**：`datacenter/src/config/connectionTypes.js`

**任务**：
- [ ] 在 `CONNECTION_TYPES` 中添加 `mqtt` 配置
- [ ] 设置 `category: 'message'`（消息连接分类）
- [ ] 定义默认配置（broker、port、protocol 等）
- [ ] 定义表单组件名称

**连接分类说明**：
- `category: 'database'` - 数据库连接（MySQL、PostgreSQL、SQL Server）
- `category: 'message'` - 消息连接（MQTT、WebSocket、OPC UA）

**示例结构**：
```javascript
mqtt: {
  label: 'MQTT',
  category: 'message',  // 消息连接分类
  icon: 'Message',
  defaultPort: 1883,
  formComponent: 'MqttConnectionForm',
  defaultConfig: {
    brokerUrl: 'localhost',
    port: 1883,
    protocol: 'mqtt',
    clientId: '',
    username: '',
    password: '',
    keepalive: 60,
    cleanSession: true,
    reconnectPeriod: 5000,
    connectTimeout: 30000,
    qos: 0
  }
}
```

---

### 2. 连接表单组件

#### 2.1 创建 MqttConnectionForm 组件
**文件位置**：`datacenter/src/components/connection/forms/MqttConnectionForm.vue`

**任务**：
- [ ] 创建表单组件
- [ ] 实现表单字段
  - Broker 地址（必填）
  - 端口号（必填，默认 1883）
  - 协议选择（mqtt/mqtts/ws/wss）
  - 客户端 ID（可选，自动生成）
  - 用户名（可选）
  - 密码（可选）
  - Keep Alive（默认 60 秒）
  - Clean Session（默认 true）
  - 重连间隔（默认 5000 ms）
  - 连接超时（默认 30000 ms）
  - 默认 QoS（0/1/2）
- [ ] 实现表单验证
- [ ] 实现高级选项折叠面板（可选配置）

**UI 布局**：
```
┌─ MQTT 连接配置 ─────────────────────┐
│                                      │
│ 基础配置                             │
│ ├─ Broker 地址: [localhost      ]  │
│ ├─ 端口: [1883]  协议: [mqtt ▼]    │
│ ├─ 客户端 ID: [自动生成]           │
│ ├─ 用户名: [          ]            │
│ └─ 密码: [          ]              │
│                                      │
│ [展开高级选项 ▼]                    │
│                                      │
│ 高级配置                             │
│ ├─ Keep Alive: [60] 秒             │
│ ├─ Clean Session: [✓]              │
│ ├─ 重连间隔: [5000] ms             │
│ ├─ 连接超时: [30000] ms            │
│ └─ 默认 QoS: [0 ▼]                 │
│                                      │
│ [测试连接] [取消] [保存]            │
└──────────────────────────────────────┘
```

**参考组件**：
- `datacenter/src/components/connection/forms/` 目录下的其他表单组件

---

### 3. 连接对话框扩展

#### 3.1 更新 ConnectionDialog 组件
**文件位置**：`datacenter/src/components/dialogs/ConnectionDialog.vue`

**任务**：
- [ ] 在连接类型选择中添加 MQTT 选项
- [ ] 支持按分类显示连接类型（数据库连接 / 消息连接）
- [ ] 根据选择的类型动态加载对应的表单组件
- [ ] 确保 MQTT 表单数据正确提交

**关键代码**：
```vue
<template>
  <el-dialog>
    <!-- 连接类型选择（按分类分组） -->
    <el-select v-model="form.type">
      <el-option-group label="数据库连接">
        <el-option label="MySQL" value="mysql" />
        <el-option label="PostgreSQL" value="postgresql" />
        <el-option label="SQL Server" value="sqlserver" />
      </el-option-group>
      <el-option-group label="消息连接">
        <el-option label="MQTT" value="mqtt" />
      </el-option-group>
    </el-select>
    
    <!-- 动态表单组件 -->
    <component
      :is="currentFormComponent"
      v-model="form.config"
    />
  </el-dialog>
</template>
```

---

### 4. 连接列表显示

#### 4.1 更新 ConnectionList 组件
**文件位置**：`datacenter/src/components/connection/ConnectionList.vue`

**任务**：
- [ ] 支持按分类显示连接（数据库连接 / 消息连接）
- [ ] 为 MQTT 连接显示特定的图标（Message 图标）
- [ ] 显示 MQTT 连接状态（已连接/未连接/错误）
- [ ] 支持 MQTT 连接的右键菜单操作
- [ ] 在连接树中添加分类标题（[数据库连接] / [消息连接]）

**UI 结构**：
```
[新建连接]

[数据库连接]
● MySQL 连接 1
● PostgreSQL 连接 2

[消息连接]
● MQTT Broker 1
```

**说明**：现有的 ConnectionList 应该已经支持多种连接类型，需要添加分类显示功能

---

### 5. 连接状态管理

#### 5.1 更新 useConnection composable
**文件位置**：`datacenter/src/composables/useConnection.js`

**任务**：
- [ ] 确保 MQTT 连接的 CRUD 操作正常工作
- [ ] 实现 MQTT 连接测试功能
- [ ] 处理 MQTT 特定的错误消息

**说明**：现有的 useConnection 应该已经支持多种连接类型，只需确认 MQTT 类型能正常工作

---

### 6. API 客户端

#### 6.1 确认 data.api.js
**文件位置**：`datacenter/src/api/data.api.js`

**检查项**：
- [ ] `createConnection` 方法支持 MQTT 配置
- [ ] `testConnection` 方法支持 MQTT 类型
- [ ] `updateConnection` 方法支持 MQTT 配置
- [ ] 错误处理正确

**说明**：现有的 API 客户端应该已经支持多种连接类型

---

## 测试任务

### 1. 后端测试

#### 1.1 单元测试
**文件位置**：`dev_core/src/services/protocols/__tests__/MqttProtocol.test.js`

**测试用例**：
- [ ] 测试连接成功场景
- [ ] 测试连接失败场景（错误的 Broker 地址）
- [ ] 测试认证失败场景（错误的用户名/密码）
- [ ] 测试连接超时场景
- [ ] 测试断开连接

#### 1.2 集成测试
**测试场景**：
- [ ] 创建 MQTT 连接
- [ ] 测试 MQTT 连接
- [ ] 更新 MQTT 连接
- [ ] 删除 MQTT 连接

---

### 2. 前端测试

#### 2.1 组件测试
**测试组件**：
- [ ] MqttConnectionForm 表单验证
- [ ] MqttConnectionForm 数据提交

#### 2.2 端到端测试
**测试流程**：
- [ ] 打开 DataCenter
- [ ] 点击"新建连接"
- [ ] 选择 MQTT 类型
- [ ] 填写连接信息
- [ ] 点击"测试连接"
- [ ] 保存连接
- [ ] 查看连接列表
- [ ] 编辑连接
- [ ] 删除连接

---

## 验收标准

### 功能验收

- [ ] 可以创建 MQTT 连接（填写 Broker、端口、认证信息）
- [ ] 可以测试 MQTT 连接（成功/失败提示）
- [ ] 可以编辑 MQTT 连接
- [ ] 可以删除 MQTT 连接
- [ ] 连接列表正确显示 MQTT 连接
- [ ] 连接状态正确显示（已连接/未连接/错误）
- [ ] 错误提示友好且准确

### 性能验收

- [ ] 连接测试响应时间 < 5 秒
- [ ] 连接创建响应时间 < 1 秒
- [ ] 连接列表加载时间 < 2 秒

### 安全验收

- [ ] MQTT 密码加密存储
- [ ] API 请求需要认证
- [ ] 错误信息不泄露敏感数据

---

## 技术风险

### 1. MQTT Broker 连接问题
**风险**：不同的 MQTT Broker 可能有不同的配置要求

**应对**：
- 支持多种协议（mqtt/mqtts/ws/wss）
- 提供详细的错误信息
- 提供连接测试功能

### 2. 密码加密存储
**风险**：密码需要加密存储，但测试连接时需要解密

**应对**：
- 使用 AES-256-CBC 加密
- 密钥存储在环境变量中
- 参考现有的 DataRelationalConfig 实现

### 3. 连接超时处理
**风险**：MQTT 连接可能因为网络问题超时

**应对**：
- 设置合理的超时时间（默认 30 秒）
- 提供清晰的超时错误提示
- 支持用户自定义超时时间

---

## 开发顺序建议

### Week 1

**Day 1-2：后端基础**
1. 创建 DataMqttConfig 模型
2. 实现 MqttProtocol 基础功能
3. 扩展 dataConnectionService

**Day 3-4：前端基础**
1. 添加 MQTT 连接类型配置
2. 创建 MqttConnectionForm 组件
3. 更新 ConnectionDialog

**Day 5：集成测试**
1. 后端单元测试
2. 前端组件测试
3. 端到端测试

### Week 2

**Day 1-2：优化和修复**
1. 修复测试中发现的问题
2. 优化用户体验
3. 完善错误处理

**Day 3-4：文档和部署**
1. 更新 API 文档
2. 更新用户文档
3. 准备演示环境

**Day 5：验收和交付**
1. 功能验收
2. 性能测试
3. 代码审查

---

## 依赖和前置条件

### 开发环境
- [ ] Node.js >= 18.0.0
- [ ] pnpm >= 8.0.0
- [ ] MySQL 数据库（用于存储连接配置）

### 测试环境
- [ ] MQTT Broker（推荐使用 Mosquitto 或 EMQX）
- [ ] 测试账号和权限

### 第三方库
- [ ] `mqtt` - MQTT 客户端库
- [ ] `crypto` - 密码加密（Node.js 内置）

---

## 参考资料

### 内部文档
- [MQTT 实现方案](../docs/datacenter/mqtt-implementation.md)
- [连接管理文档](../docs/datacenter/connections.md)
- [数据库实现概述](../docs/datacenter/database-implementation.md)

### 外部资源
- [MQTT.js 文档](https://github.com/mqttjs/MQTT.js)
- [MQTT 协议规范](https://mqtt.org/mqtt-specification/)
- [Mosquitto 文档](https://mosquitto.org/documentation/)

---

## 问题和决策记录

### Q1: MQTT 密码如何加密存储？
**决策**：使用 AES-256-CBC 加密，参考 DataRelationalConfig 的实现

### Q2: 是否需要支持 MQTT 5.0？
**决策**：阶段一先支持 MQTT 3.1.1，后续根据需求支持 5.0

### Q3: 客户端 ID 如何生成？
**决策**：默认自动生成（`induforge_${timestamp}_${random}`），用户也可以自定义

### Q4: 是否需要支持 SSL/TLS？
**决策**：阶段一先支持基础连接，SSL/TLS 在后续阶段实现

---

**文档版本**：1.0.0  
**创建日期**：2025-12-15  
**最后更新**：2025-12-15  
**负责人**：开发团队
