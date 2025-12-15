# DataCenter 阶段五开发计划 - MQTT 数据回写功能

## 目标概述

实现 MQTT 数据回写和消息发布功能，支持用户自定义消息格式模板，实现变量回写和自定义消息发布两种模式。

**预计时间**：1-2 周  
**优先级**：P1（核心功能）  
**前置条件**：阶段 1-3 已完成（MQTT 连接、主题订阅、变量管理）

---

## 核心功能

### 1. 变量回写模式
- 直接修改变量值，系统根据用户配置的模板自动构造消息并发布
- 适用场景：简单的值修改、标准化控制指令、单个变量更新

### 2. 消息发布模式
- 用户自定义完整的消息内容，灵活控制发布的数据格式
- 适用场景：复杂控制指令、批量参数设置、多变量联动

### 3. 用户自定义消息模板
- 支持任意格式：JSON、文本、XML、CSV、自定义协议、二进制等
- 模板变量替换：`{{value}}`、`{{timestamp}}`、`{{user}}` 等
- 变量格式化：`{{value:json}}`、`{{value:hex}}`、`{{value:int}}` 等
- 实时预览：显示替换变量后的实际消息

---

## 后端开发任务

### 1. 数据模型扩展

#### 1.1 扩展 data_tags 表的 metadata 字段
**文件位置**：`dev_core/src/models/DataTag.js`

**任务**：
- [ ] 在 Tag 模型中添加 `writeConfig` 的 JSON Schema 验证
- [ ] 确保 `metadata.writeConfig` 结构正确

**writeConfig 结构**：
```json
{
  "writeConfig": {
    "topic": "device/001/control",
    "qos": 1,
    "retain": false,
    "payloadTemplate": "{\"cmd\":\"write\",\"val\":{{value}},\"ts\":{{timestamp}}}",
    "validation": {
      "min": 0,
      "max": 100,
      "required": true
    },
    "confirmation": true
  }
}
```

---

### 2. 模板引擎实现

#### 2.1 创建 TemplateEngine 工具类
**文件位置**：`dev_core/src/utils/templateEngine.js`

**任务**：
- [ ] 实现变量替换功能
- [ ] 支持内置变量：`{{value}}`、`{{timestamp}}`、`{{datetime}}`、`{{user}}`、`{{userId}}`、`{{tagName}}`、`{{tagId}}`、`{{oldValue}}`
- [ ] 支持变量格式化：`{{value:json}}`、`{{value:hex}}`、`{{value:int}}`、`{{timestamp:iso}}`
- [ ] 实现模板验证（检查语法错误）
- [ ] 实现模板预览功能

**核心方法**：
```javascript
class TemplateEngine {
  // 渲染模板
  static render(template, variables) { }
  
  // 验证模板语法
  static validate(template) { }
  
  // 格式化变量值
  static formatValue(value, format) { }
  
  // 获取可用变量列表
  static getAvailableVariables() { }
}
```

---

### 3. 服务层实现

#### 3.1 扩展 tagService
**文件位置**：`dev_core/src/services/tagService.js`

**任务**：
- [ ] 实现 `writeTagValue(tagId, value, userId)` 方法
- [ ] 实现值验证（min/max/required）
- [ ] 实现消息模板渲染
- [ ] 调用 MqttProtocol.publish() 发布消息
- [ ] 记录回写日志

**实现思路**：
```javascript
async writeTagValue(tagId, value, userId) {
  // 1. 获取 Tag 配置
  const tag = await DataTag.findByPk(tagId);
  
  // 2. 验证访问模式
  if (tag.accessMode === 'read') {
    throw new AppError('TAG_READ_ONLY');
  }
  
  // 3. 验证值
  this.validateValue(value, tag.metadata.writeConfig.validation);
  
  // 4. 渲染消息模板
  const payload = TemplateEngine.render(
    tag.metadata.writeConfig.payloadTemplate,
    { value, timestamp: Date.now(), user: userId, ... }
  );
  
  // 5. 发布消息
  const protocol = await this.getMqttProtocol(tag.connectionId);
  await protocol.publish(
    tag.metadata.writeConfig.topic,
    payload,
    { qos: tag.metadata.writeConfig.qos }
  );
  
  // 6. 记录日志
  await this.logWriteOperation(tag, value, payload);
  
  return { success: true, payload };
}
```

#### 3.2 扩展 mqttService
**文件位置**：`dev_core/src/services/mqttService.js`

**任务**：
- [ ] 实现 `publishMessage(connectionId, topic, payload, options)` 方法
- [ ] 支持自定义消息发布
- [ ] 记录发布历史

---

### 4. 控制器实现

#### 4.1 创建 tagWriteController
**文件位置**：`dev_core/src/controllers/tagWriteController.js`

**任务**：
- [ ] 实现变量回写 API
- [ ] 实现消息发布 API
- [ ] 实现回写确认逻辑
- [ ] 错误处理

**API 方法**：
```javascript
// 变量回写
async writeTag(req, res, next) {
  const { tagId } = req.params;
  const { value, confirmation } = req.body;
  const userId = req.user.id;
  
  // 实现回写逻辑
}

// 消息发布
async publishMessage(req, res, next) {
  const { connectionId, topic, qos, retain, payload } = req.body;
  
  // 实现发布逻辑
}
```

---

### 5. API 路由

#### 5.1 添加回写相关路由
**文件位置**：`dev_core/src/routes/data.js`

**任务**：
- [ ] `POST /api/v1/data/tags/:tagId/write` - 变量回写
- [ ] `POST /api/v1/data/mqtt/publish` - 消息发布
- [ ] `GET /api/v1/data/tags/:tagId/write-history` - 回写历史（可选）

---

### 6. 日志记录

#### 6.1 创建回写日志表（可选）
**文件位置**：`dev_core/database/init.sql`

**表结构**：
```sql
CREATE TABLE data_tag_write_logs (
  id VARCHAR(36) PRIMARY KEY,
  tag_id VARCHAR(36) NOT NULL,
  user_id VARCHAR(36) NOT NULL,
  old_value TEXT,
  new_value TEXT,
  payload TEXT,
  topic VARCHAR(255),
  status VARCHAR(20),
  error_message TEXT,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (tag_id) REFERENCES data_tags(id)
);
```

---

## 前端开发任务

### 1. Tag 编辑器扩展

#### 1.1 扩展 TagEditor 组件
**文件位置**：`datacenter/src/components/tags/TagEditor.vue`

**任务**：
- [ ] 添加"回写配置"面板
- [ ] 实现访问模式选择（read/write/read_write）
- [ ] 实现回写配置表单
  - 回写主题
  - QoS 选择
  - Retain 选项
  - 消息模板编辑器
  - 值验证配置
  - 确认选项
- [ ] 实现模板编辑器
  - 多行文本输入
  - 语法高亮（可选）
  - 可用变量列表
  - 快速插入变量按钮
- [ ] 实现模板示例
  - JSON 格式示例
  - 文本格式示例
  - XML 格式示例
  - 自定义格式示例
- [ ] 实现实时预览
  - 输入测试值
  - 显示渲染后的消息
- [ ] 实现测试发布功能
  - 使用测试值实际发布消息
  - 显示发布结果

**UI 布局参考**：
```
┌─ 变量回写配置 ─────────────────────────────────┐
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
│ [插入变量 ▼] [模板示例 ▼]                      │
│                                                  │
│ 可用变量：{{value}}, {{timestamp}}, {{user}}   │
│                                                  │
│ 值验证（可选）                                   │
│ ├─ 最小值: [0]  最大值: [100]                  │
│ └─ 必填: [✓]                                    │
│                                                  │
│ 测试回写                                         │
│ ├─ 测试值: [25.5]                               │
│ └─ 预览消息:                                     │
│   {"cmd":"write","val":25.5,"ts":1702886400000} │
│                                                  │
│ [测试发布] [保存配置]                            │
└──────────────────────────────────────────────────┘
```

---

### 2. 消息发布组件

#### 2.1 创建 MqttPublisher 组件
**文件位置**：`datacenter/src/components/mqtt/MqttPublisher.vue`

**任务**：
- [ ] 创建消息发布界面
- [ ] 实现连接选择
- [ ] 实现主题输入
- [ ] 实现 QoS 和 Retain 选项
- [ ] 实现消息内容编辑器
  - 多行文本输入
  - JSON 格式化按钮
  - 语法高亮（可选）
- [ ] 实现发布按钮
- [ ] 实现发布历史记录
  - 显示最近发布的消息
  - 支持重新发布
  - 支持清空历史

**UI 布局**：
```
┌─ MQTT 消息发布 ─────────────────────┐
│ 连接: [MQTT Broker 1 ▼]            │
│ 主题: [device/001/control]          │
│ QoS: [1 ▼]  Retain: [ ]            │
│                                      │
│ 消息内容                             │
│ ┌────────────────────────────────┐ │
│ │ {                              │ │
│ │   "command": "set_switch",     │ │
│ │   "value": 1                   │ │
│ │ }                              │ │
│ └────────────────────────────────┘ │
│ [格式化] [清空]                      │
│                                      │
│ [发布消息]                           │
│                                      │
│ 发布历史                             │
│ • 2024-12-15 10:30:45 - 成功        │
│ • 2024-12-15 10:25:12 - 成功        │
│ [清空历史]                           │
└──────────────────────────────────────┘
```

---

### 3. Composables

#### 3.1 创建 useTagWrite composable
**文件位置**：`datacenter/src/composables/useTagWrite.js`

**任务**：
- [ ] 实现 `writeTag(tagId, value)` 方法
- [ ] 实现 `publishMessage(config)` 方法
- [ ] 实现 `previewTemplate(template, variables)` 方法
- [ ] 实现 `testPublish(tagId, testValue)` 方法
- [ ] 错误处理和消息提示

---

### 4. API 客户端

#### 4.1 扩展 data.api.js
**文件位置**：`datacenter/src/api/data.api.js`

**任务**：
- [ ] 添加 `writeTag(tagId, value, confirmation)` 方法
- [ ] 添加 `publishMqttMessage(config)` 方法
- [ ] 添加 `previewTemplate(template, variables)` 方法（可选）

---

## Designer 集成任务

### 1. 动作类型扩展

#### 1.1 添加 writeTag 动作类型
**文件位置**：`designer/src/engine/actions/`

**任务**：
- [ ] 实现 `writeTag` 动作处理器
- [ ] 支持表达式计算回写值
- [ ] 支持确认对话框
- [ ] 错误处理和提示

**配置示例**：
```json
{
  "type": "writeTag",
  "config": {
    "tagName": "temperature_setpoint",
    "value": "{{ $event.value }}",
    "confirmation": true,
    "confirmMessage": "确定要修改温度设定值吗？"
  }
}
```

#### 1.2 添加 publishMqtt 动作类型
**文件位置**：`designer/src/engine/actions/`

**任务**：
- [ ] 实现 `publishMqtt` 动作处理器
- [ ] 支持表达式计算 payload
- [ ] 支持确认对话框
- [ ] 错误处理和提示

**配置示例**：
```json
{
  "type": "publishMqtt",
  "config": {
    "connectionId": "mqtt-conn-uuid",
    "topic": "device/batch-control",
    "payload": {
      "command": "batch_set",
      "temperature": "{{ $vars.targetTemp }}"
    }
  }
}
```

---

### 2. 组件事件配置

#### 2.1 扩展事件配置面板
**文件位置**：`designer/src/components/property-panel/EventConfig.vue`

**任务**：
- [ ] 在动作类型下拉列表中添加 `writeTag` 和 `publishMqtt`
- [ ] 实现 `writeTag` 配置表单
  - Tag 选择器
  - 值表达式编辑器
  - 确认选项
- [ ] 实现 `publishMqtt` 配置表单
  - 连接选择器
  - 主题输入
  - Payload 编辑器（支持表达式）

---

## 测试任务

### 1. 后端测试

#### 1.1 单元测试
**文件位置**：`dev_core/src/utils/__tests__/templateEngine.test.js`

**测试用例**：
- [ ] 测试变量替换
- [ ] 测试变量格式化
- [ ] 测试模板验证
- [ ] 测试边界情况

**文件位置**：`dev_core/src/services/__tests__/tagService.test.js`

**测试用例**：
- [ ] 测试变量回写成功
- [ ] 测试值验证失败
- [ ] 测试只读变量回写失败
- [ ] 测试模板渲染

#### 1.2 集成测试
**测试场景**：
- [ ] 创建带回写配置的 Tag
- [ ] 回写变量值
- [ ] 发布自定义消息
- [ ] 查看回写历史

---

### 2. 前端测试

#### 2.1 组件测试
**测试组件**：
- [ ] TagEditor 回写配置面板
- [ ] MqttPublisher 消息发布
- [ ] 模板预览功能

#### 2.2 端到端测试
**测试流程**：
- [ ] 配置 Tag 回写规则
- [ ] 测试模板预览
- [ ] 测试发布消息
- [ ] 在 Designer 中配置回写动作
- [ ] 在运行态触发回写
- [ ] 验证消息发布成功

---

## 验收标准

### 功能验收

- [ ] 可以配置 Tag 回写规则（主题、模板、验证）
- [ ] 可以使用任意格式的消息模板（JSON、文本、XML 等）
- [ ] 可以插入模板变量（{{value}}、{{timestamp}} 等）
- [ ] 可以实时预览渲染后的消息
- [ ] 可以测试发布消息到 MQTT Broker
- [ ] 可以通过 API 回写变量值
- [ ] 可以通过 API 发布自定义消息
- [ ] Designer 中可以配置回写动作
- [ ] 回写前可以显示确认对话框
- [ ] 回写操作有日志记录
- [ ] 错误提示友好且准确

### 性能验收

- [ ] 模板渲染时间 < 10ms
- [ ] 回写 API 响应时间 < 500ms
- [ ] 消息发布响应时间 < 500ms

### 安全验收

- [ ] 回写操作需要认证
- [ ] 回写操作有权限控制
- [ ] 模板变量不允许执行任意代码
- [ ] 回写日志记录完整

---

## 技术风险

### 1. 模板安全性
**风险**：用户自定义模板可能包含恶意代码

**应对**：
- 模板引擎只支持简单的变量替换，不支持代码执行
- 对模板进行语法验证
- 限制模板大小（如 10KB）

### 2. 消息格式兼容性
**风险**：不同设备可能需要不同的消息格式

**应对**：
- 提供丰富的模板示例（8+ 种格式）
- 支持完全自定义的模板
- 提供实时预览功能

### 3. 回写确认机制
**风险**：误操作可能导致设备故障

**应对**：
- 支持回写前确认
- 支持值验证（min/max/required）
- 记录完整的回写日志
- 支持回写权限控制

---

## 开发顺序建议

### Week 1

**Day 1-2：后端基础**
1. 实现 TemplateEngine 工具类
2. 扩展 tagService 实现回写功能
3. 创建 tagWriteController
4. 添加 API 路由

**Day 3-4：前端基础**
1. 扩展 TagEditor 组件（回写配置面板）
2. 实现模板编辑器
3. 实现实时预览功能
4. 创建 MqttPublisher 组件

**Day 5：集成测试**
1. 后端单元测试
2. 前端组件测试
3. API 集成测试

### Week 2

**Day 1-2：Designer 集成**
1. 实现 writeTag 动作类型
2. 实现 publishMqtt 动作类型
3. 扩展事件配置面板

**Day 3：优化和修复**
1. 修复测试中发现的问题
2. 优化用户体验
3. 完善错误处理

**Day 4：文档和部署**
1. 更新 API 文档
2. 更新用户文档
3. 准备演示环境

**Day 5：验收和交付**
1. 功能验收
2. 性能测试
3. 代码审查

---

## 依赖和前置条件

### 前置阶段
- [ ] 阶段 1：MQTT 连接管理（已完成）
- [ ] 阶段 2：主题订阅管理（已完成）
- [ ] 阶段 3：变量管理与解析（已完成）

### 开发环境
- [ ] Node.js >= 18.0.0
- [ ] pnpm >= 8.0.0
- [ ] MQTT Broker（用于测试）

### 测试环境
- [ ] 可回写的 MQTT 设备或模拟器
- [ ] 测试账号和权限

---

## 参考资料

### 内部文档
- [MQTT 实现方案](../docs/datacenter/mqtt-implementation.md)
- [阶段一开发计划](./PHASE1_PLAN.md)

### 外部资源
- [MQTT.js 文档](https://github.com/mqttjs/MQTT.js)
- [模板引擎设计模式](https://en.wikipedia.org/wiki/Template_processor)

---

## 问题和决策记录

### Q1: 模板变量如何转义？
**决策**：使用 `{{` 和 `}}` 作为变量标记，如需输出字面量 `{{`，使用 `\{{`

### Q2: 是否支持条件判断和循环？
**决策**：阶段五不支持，只支持简单的变量替换和格式化

### Q3: 回写日志保留多久？
**决策**：默认保留 30 天，可配置

### Q4: 是否需要回写队列？
**决策**：阶段五不实现，直接发布。后续可以考虑添加队列机制

---

**文档版本**：1.0.0  
**创建日期**：2025-12-15  
**最后更新**：2025-12-15  
**负责人**：开发团队
