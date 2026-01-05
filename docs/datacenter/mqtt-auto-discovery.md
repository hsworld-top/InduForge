# MQTT 变量自动发现与批量导入

## 概述

在工业数据采集场景中，MQTT 变量的来源方式多样化，本功能支持以下三种变量获取方式：

| 方式                 | 场景                     | 特点               |
| -------------------- | ------------------------ | ------------------ |
| **MQTT 自动发现**    | 数据采集网关推送变量列表 | 实时发现、自动同步 |
| **RESTful API 拉取** | 第三方系统提供变量接口   | 主动获取、定时同步 |
| **CSV 文件导入**     | 离线变量清单             | 批量导入、手动维护 |

无论采用哪种方式获取变量定义，数据值的获取仍通过 MQTT Data Topic 订阅实现。

## 应用场景

### 典型数据流

```
┌─────────────────┐     Tag Topic        ┌─────────────────┐
│                 │ ───────────────────► │                 │
│   数据采集网关   │    (变量列表)         │   MQTT Broker   │
│   (PLC/SCADA)   │                      │                 │
│                 │     Data Topic       │                 │
│                 │ ───────────────────► │                 │
└─────────────────┘    (实时值)          └─────────────────┘
                                                │
                                                │ 订阅
                                                ▼
                                         ┌─────────────────┐
                                         │   InduForge     │
                                         │   DataCenter    │
                                         │                 │
                                         │  自动发现变量    │
                                         │  解析实时数据    │
                                         └─────────────────┘
```

### 数据格式示例

**Tag Topic 消息**（变量定义，推送一次或定期推送）：

```json
{
  "tags": [
    {
      "name": "温度传感器_01",
      "dataType": "number",
      "unit": "℃",
      "desc": "车间1温度"
    },
    {
      "name": "温度传感器_02",
      "dataType": "number",
      "unit": "℃",
      "desc": "车间2温度"
    },
    {
      "name": "压力传感器_01",
      "dataType": "number",
      "unit": "MPa",
      "desc": "管道压力"
    },
    { "name": "设备状态_01", "dataType": "boolean", "desc": "设备运行状态" }
  ]
}
```

**Data Topic 消息**（实时值，持续推送）：

```json
{
  "values": [
    { "N": "温度传感器_01", "V": 25.6, "T": 1704412800000, "Q": 192 },
    { "N": "温度传感器_02", "V": 26.1, "T": 1704412800000, "Q": 192 },
    { "N": "压力传感器_01", "V": 1.25, "T": 1704412800000, "Q": 192 },
    { "N": "设备状态_01", "V": true, "T": 1704412800000, "Q": 192 }
  ]
}
```

> **字段说明**：N = Name（变量名），V = Value（值），T = Timestamp（时间戳），Q = Quality（质量码）

## 功能设计

### 变量来源配置界面

在 MQTT 连接配置中，新增「变量来源」配置区域，支持三种来源方式的切换：

```
┌─ 变量来源配置 ─────────────────────────────────────────────────────────────────┐
│                                                                                 │
│  来源方式:  ● MQTT 自动发现   ○ RESTful API   ○ CSV 导入   ○ 手动创建         │
│                                                                                 │
└─────────────────────────────────────────────────────────────────────────────────┘
```

---

### 方式一：MQTT 自动发现

适用于数据采集网关通过 MQTT 主题推送变量列表的场景：

```
┌─ MQTT 连接配置 ──────────────────────────────────────────────────────────────┐
│                                                                               │
│  连接名称:  [生产数据采集                        ]                            │
│  Broker:    [mqtt://192.168.1.100:1883           ]                            │
│                                                                               │
│  ════════════════════════════════════════════════════════════════════════════ │
│                                                                               │
│  ▼ 自动发现配置                                                              │
│                                                                               │
│    ○ 禁用    ● 启用                                                          │
│                                                                               │
│    ┌─ 元数据主题（Tag Topic）───────────────────────────────────────────────┐│
│    │                                                                         ││
│    │  主题:        [system/tags                          ]                  ││
│    │  数据格式:    [JSON ▼]                                                  ││
│    │                                                                         ││
│    │  解析配置:                                                              ││
│    │    变量列表路径:   [$.tags                          ]                  ││
│    │    变量名字段:     [name                            ]                  ││
│    │    数据类型字段:   [dataType                        ]  (可选)          ││
│    │    单位字段:       [unit                            ]  (可选)          ││
│    │    描述字段:       [desc                            ]  (可选)          ││
│    │                                                                         ││
│    └─────────────────────────────────────────────────────────────────────────┘│
│                                                                               │
│    ┌─ 数据主题（Data Topic）────────────────────────────────────────────────┐│
│    │                                                                         ││
│    │  主题:        [system/data                          ]                  ││
│    │  数据格式:    [JSON ▼]                                                  ││
│    │                                                                         ││
│    │  解析配置:                                                              ││
│    │    值列表路径:     [$.values                        ]                  ││
│    │    变量名字段:     [N                               ]                  ││
│    │    值字段:         [V                               ]                  ││
│    │    时间戳字段:     [T                               ]  (可选)          ││
│    │    质量码字段:     [Q                               ]  (可选)          ││
│    │                                                                         ││
│    └─────────────────────────────────────────────────────────────────────────┘│
│                                                                               │
│    目标变量组:  [自动发现变量 ▼]  [+ 新建变量组]                             │
│                                                                               │
│                                          [测试解析]  [保存配置]              │
│                                                                               │
└───────────────────────────────────────────────────────────────────────────────┘
```

### 配置数据模型

```typescript
interface MqttAutoDiscoveryConfig {
  enabled: boolean; // 是否启用自动发现

  // 元数据主题配置
  tagTopic: {
    topic: string; // 主题名称
    format: "json" | "xml"; // 数据格式
    listPath: string; // 变量列表路径 (JSONPath/XPath)
    fields: {
      name: string; // 变量名字段
      dataType?: string; // 数据类型字段
      unit?: string; // 单位字段
      description?: string; // 描述字段
      [key: string]: string; // 其他扩展字段
    };
  };

  // 数据主题配置
  dataTopic: {
    topic: string; // 主题名称
    format: "json" | "xml"; // 数据格式
    listPath: string; // 值列表路径
    fields: {
      name: string; // 变量名字段 (对应 N)
      value: string; // 值字段 (对应 V)
      timestamp?: string; // 时间戳字段 (对应 T)
      quality?: string; // 质量码字段 (对应 Q)
    };
  };

  targetGroupId: string; // 目标变量组 ID
}
```

### 数据库表扩展

在 `data_mqtt_configs` 表增加自动发现配置字段：

```sql
ALTER TABLE data_mqtt_configs ADD COLUMN auto_discovery_config JSON;
```

或新建独立表：

```sql
CREATE TABLE data_mqtt_auto_discovery (
  id              VARCHAR(36) PRIMARY KEY,
  config_id       VARCHAR(36) NOT NULL,      -- 关联 MQTT 连接
  enabled         BOOLEAN DEFAULT false,

  -- 元数据主题配置
  tag_topic       VARCHAR(255) NOT NULL,
  tag_format      VARCHAR(20) DEFAULT 'json',
  tag_list_path   VARCHAR(255),
  tag_fields      JSON,

  -- 数据主题配置
  data_topic      VARCHAR(255) NOT NULL,
  data_format     VARCHAR(20) DEFAULT 'json',
  data_list_path  VARCHAR(255),
  data_fields     JSON,

  -- 目标变量组
  target_group_id VARCHAR(36),

  created_at      DATETIME NOT NULL,
  updated_at      DATETIME NOT NULL,

  INDEX idx_config_id (config_id)
);
```

---

### 方式二：RESTful API 拉取

适用于变量清单由第三方系统（如 SCADA、MES、设备管理平台）通过 HTTP 接口提供的场景：

```
┌─ RESTful API 配置 ─────────────────────────────────────────────────────────────┐
│                                                                                 │
│  接口地址:  [https://scada.example.com/api/tags     ]                          │
│  请求方式:  [GET ▼]                                                             │
│                                                                                 │
│  ┌─ 认证配置（可选）─────────────────────────────────────────────────────────┐ │
│  │                                                                            │ │
│  │  认证方式:  ○ 无   ● Basic Auth   ○ Bearer Token   ○ API Key              │ │
│  │  用户名:    [admin                               ]                         │ │
│  │  密码:      [********                            ]                         │ │
│  │                                                                            │ │
│  └────────────────────────────────────────────────────────────────────────────┘ │
│                                                                                 │
│  ┌─ 请求头（可选）───────────────────────────────────────────────────────────┐ │
│  │  [+ 添加请求头]                                                            │ │
│  │  X-Tenant-Id:  [tenant-001                       ]                         │ │
│  └────────────────────────────────────────────────────────────────────────────┘ │
│                                                                                 │
│  ┌─ 响应解析 ────────────────────────────────────────────────────────────────┐ │
│  │                                                                            │ │
│  │  数据格式:        [JSON ▼]                                                  │ │
│  │  变量列表路径:    [$.data.tags                   ]                         │ │
│  │  变量名字段:      [tagName                       ]                         │ │
│  │  数据类型字段:    [dataType                      ]  (可选)                 │ │
│  │  单位字段:        [unit                          ]  (可选)                 │ │
│  │  描述字段:        [description                   ]  (可选)                 │ │
│  │                                                                            │ │
│  └────────────────────────────────────────────────────────────────────────────┘ │
│                                                                                 │
│  ┌─ 同步策略 ────────────────────────────────────────────────────────────────┐ │
│  │                                                                            │ │
│  │  ○ 手动同步    ● 定时同步                                                  │ │
│  │  同步周期:     [每天 ▼]  时间: [02:00]                                     │ │
│  │  □ 连接启动时自动同步                                                      │ │
│  │                                                                            │ │
│  └────────────────────────────────────────────────────────────────────────────┘ │
│                                                                                 │
│  目标变量组:  [API同步变量 ▼]  [+ 新建变量组]                                  │
│                                                                                 │
│                                    [测试连接]  [立即同步]  [保存配置]          │
│                                                                                 │
└─────────────────────────────────────────────────────────────────────────────────┘
```

#### RESTful API 响应示例

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "tags": [
      {
        "tagName": "PLC01.温度",
        "dataType": "float",
        "unit": "℃",
        "description": "1#炉温度"
      },
      {
        "tagName": "PLC01.压力",
        "dataType": "float",
        "unit": "MPa",
        "description": "1#炉压力"
      },
      {
        "tagName": "PLC01.状态",
        "dataType": "int",
        "description": "1#炉运行状态"
      }
    ],
    "total": 3
  }
}
```

#### API 配置数据模型

```typescript
interface MqttApiDiscoveryConfig {
  enabled: boolean;

  // API 配置
  api: {
    url: string; // 接口地址
    method: "GET" | "POST"; // 请求方式
    auth?: {
      type: "none" | "basic" | "bearer" | "apikey";
      username?: string;
      password?: string;
      token?: string;
      apiKey?: string;
      apiKeyHeader?: string; // API Key 的 Header 名称
    };
    headers?: Record<string, string>; // 自定义请求头
    body?: string; // POST 请求体（JSON 字符串）
  };

  // 响应解析
  response: {
    format: "json" | "xml";
    listPath: string; // 变量列表路径
    fields: {
      name: string;
      dataType?: string;
      unit?: string;
      description?: string;
    };
  };

  // 同步策略
  sync: {
    mode: "manual" | "scheduled";
    cron?: string; // Cron 表达式
    syncOnStart?: boolean; // 连接启动时同步
  };

  targetGroupId: string;
}
```

---

### 方式三：CSV 文件导入

适用于离线变量清单、Excel 导出的点表、或需要批量维护变量的场景：

```
┌─ CSV 导入 ─────────────────────────────────────────────────────────────────────┐
│                                                                                 │
│  ┌─────────────────────────────────────────────────────────────────────────┐   │
│  │                                                                          │   │
│  │                     📂 点击或拖拽文件到此处上传                          │   │
│  │                                                                          │   │
│  │                     支持 .csv, .xlsx, .xls 格式                         │   │
│  │                     文件大小不超过 10MB                                  │   │
│  │                                                                          │   │
│  └─────────────────────────────────────────────────────────────────────────┘   │
│                                                                                 │
│  [📥 下载模板]  [📖 查看导入说明]                                              │
│                                                                                 │
│  ════════════════════════════════════════════════════════════════════════════  │
│                                                                                 │
│  ┌─ 字段映射 ────────────────────────────────────────────────────────────────┐ │
│  │                                                                            │ │
│  │  文件列名              →    系统字段                                       │ │
│  │  ─────────────────────────────────────────────────────                     │ │
│  │  [TagName        ▼]   →    变量名（必填）                                  │ │
│  │  [DataType       ▼]   →    数据类型                                        │ │
│  │  [Unit           ▼]   →    单位                                            │ │
│  │  [Description    ▼]   →    描述                                            │ │
│  │  [MinValue       ▼]   →    最小值                                          │ │
│  │  [MaxValue       ▼]   →    最大值                                          │ │
│  │                                                                            │ │
│  └────────────────────────────────────────────────────────────────────────────┘ │
│                                                                                 │
│  ┌─ 导入预览 ────────────────────────────────────────────────────────────────┐ │
│  │                                                                            │ │
│  │  │ 变量名      │ 数据类型 │ 单位 │ 描述       │ 状态      │                │ │
│  │  ├─────────────┼──────────┼──────┼────────────┼───────────┤                │ │
│  │  │ 温度_01     │ number   │ ℃   │ 车间温度   │ ✅ 新增   │                │ │
│  │  │ 温度_02     │ number   │ ℃   │ 仓库温度   │ ✅ 新增   │                │ │
│  │  │ 压力_01     │ number   │ MPa  │ 管道压力   │ 🔄 更新   │                │ │
│  │  │ 状态_01     │ boolean  │ -    │ 设备状态   │ ⚠️ 冲突   │                │ │
│  │  │ 无效变量    │ -        │ -    │ -          │ ❌ 跳过   │                │ │
│  │                                                                            │ │
│  │  共 5 条记录: 新增 2, 更新 1, 冲突 1, 跳过 1                               │ │
│  │                                                                            │ │
│  └────────────────────────────────────────────────────────────────────────────┘ │
│                                                                                 │
│  导入模式:  ● 合并（保留现有，新增/更新）   ○ 覆盖（清空后全量导入）           │
│                                                                                 │
│  目标变量组:  [导入变量 ▼]  [+ 新建变量组]                                     │
│                                                                                 │
│                                              [取消]  [确认导入 (4条)]          │
│                                                                                 │
└─────────────────────────────────────────────────────────────────────────────────┘
```

#### CSV 模板格式

```csv
TagName,DataType,Unit,Description,MinValue,MaxValue
温度传感器_01,number,℃,车间1温度,0,100
温度传感器_02,number,℃,车间2温度,0,100
压力传感器_01,number,MPa,管道压力,0,10
设备状态_01,boolean,,设备运行状态,,
```

#### 导入状态说明

| 状态      | 说明                                           |
| --------- | ---------------------------------------------- |
| ✅ 新增   | 变量不存在，将创建新变量                       |
| 🔄 更新   | 变量已存在，属性有变化，将更新                 |
| ⚠️ 冲突   | 变量已存在但属性冲突（如数据类型不同），需确认 |
| ❌ 跳过   | 数据无效（如变量名为空），跳过导入             |
| ➖ 无变化 | 变量已存在且属性相同，无需操作                 |

#### 导入配置数据模型

```typescript
interface CsvImportConfig {
  // 字段映射
  fieldMapping: {
    name: string; // 变量名列
    dataType?: string; // 数据类型列
    unit?: string; // 单位列
    description?: string; // 描述列
    minValue?: string; // 最小值列
    maxValue?: string; // 最大值列
  };

  // 导入选项
  options: {
    mode: "merge" | "overwrite"; // 合并 / 覆盖
    skipInvalid: boolean; // 跳过无效行
    updateExisting: boolean; // 更新已存在的变量
  };

  targetGroupId: string;
}
```

---

### 数据主题配置（通用）

无论采用哪种变量来源方式，数据值的获取都通过 MQTT Data Topic 实现：

```
┌─ 数据主题配置（所有来源方式通用）──────────────────────────────────────────────┐
│                                                                                 │
│  主题:        [system/data                          ]                          │
│  数据格式:    [JSON ▼]                                                          │
│                                                                                 │
│  解析配置:                                                                      │
│    值列表路径:     [$.values                        ]                          │
│    变量名字段:     [N                               ]                          │
│    值字段:         [V                               ]                          │
│    时间戳字段:     [T                               ]  (可选)                  │
│    质量码字段:     [Q                               ]  (可选)                  │
│                                                                                 │
│                                          [测试解析]  [保存配置]                │
│                                                                                 │
└─────────────────────────────────────────────────────────────────────────────────┘
```

## 处理流程

### 1. 变量自动发现流程

```
                    订阅 Tag Topic
                          │
                          ▼
                    收到元数据消息
                          │
                          ▼
              ┌───────────────────────┐
              │  解析变量列表          │
              │  (根据 listPath)       │
              └───────────────────────┘
                          │
                          ▼
              ┌───────────────────────┐
              │  遍历每个变量定义      │
              └───────────────────────┘
                          │
            ┌─────────────┼─────────────┐
            ▼             ▼             ▼
        变量已存在?   变量不存在?   变量被删除?
            │             │             │
            ▼             ▼             ▼
        更新属性      创建变量      标记失效
            │             │             │
            └─────────────┼─────────────┘
                          │
                          ▼
              ┌───────────────────────┐
              │  同步数据点            │
              │  (自动生成/更新)       │
              └───────────────────────┘
```

### 2. 数据解析流程

```
                    订阅 Data Topic
                          │
                          ▼
                    收到数据消息
                          │
                          ▼
              ┌───────────────────────┐
              │  解析值列表            │
              │  (根据 listPath)       │
              └───────────────────────┘
                          │
                          ▼
              ┌───────────────────────┐
              │  遍历每个数据项        │
              │  { N, V, T, Q }        │
              └───────────────────────┘
                          │
                          ▼
              ┌───────────────────────┐
              │  根据 N 查找变量       │
              └───────────────────────┘
                          │
            ┌─────────────┴─────────────┐
            ▼                           ▼
        找到变量                    未找到变量
            │                           │
            ▼                           ▼
        更新变量值                  记录日志/忽略
        (V, T, Q)
            │
            ▼
        推送 Socket.IO
        mqtt:tag:value
            │
            ▼
        更新数据点值
        datapoint:value
```

### 3. RESTful API 同步流程

```
              ┌───────────────────────┐
              │  触发同步             │
              │  (手动/定时/启动)     │
              └───────────────────────┘
                          │
                          ▼
              ┌───────────────────────┐
              │  发送 HTTP 请求       │
              │  (带认证信息)         │
              └───────────────────────┘
                          │
            ┌─────────────┴─────────────┐
            ▼                           ▼
        请求成功                    请求失败
            │                           │
            ▼                           ▼
        解析响应                    记录错误
        提取变量列表                重试/告警
            │
            ▼
        ┌───────────────────────┐
        │  对比现有变量          │
        │  计算差异              │
        └───────────────────────┘
            │
            ▼
        ┌───────────────────────┐
        │  批量创建/更新/失效    │
        │  变量                 │
        └───────────────────────┘
            │
            ▼
        ┌───────────────────────┐
        │  同步数据点            │
        └───────────────────────┘
            │
            ▼
        ┌───────────────────────┐
        │  记录同步日志          │
        │  发送同步通知          │
        └───────────────────────┘
```

### 4. CSV 导入流程

```
              ┌───────────────────────┐
              │  上传文件             │
              │  (.csv/.xlsx/.xls)    │
              └───────────────────────┘
                          │
                          ▼
              ┌───────────────────────┐
              │  解析文件内容         │
              │  提取列头与数据行     │
              └───────────────────────┘
                          │
                          ▼
              ┌───────────────────────┐
              │  用户配置字段映射     │
              └───────────────────────┘
                          │
                          ▼
              ┌───────────────────────┐
              │  预览校验             │
              │  标记每行状态         │
              │  (新增/更新/冲突/跳过) │
              └───────────────────────┘
                          │
                          ▼
              ┌───────────────────────┐
              │  用户确认导入         │
              │  选择导入模式         │
              └───────────────────────┘
                          │
                          ▼
              ┌───────────────────────┐
              │  执行批量导入         │
              │  创建/更新变量        │
              └───────────────────────┘
                          │
                          ▼
              ┌───────────────────────┐
              │  同步数据点           │
              └───────────────────────┘
                          │
                          ▼
              ┌───────────────────────┐
              │  返回导入结果         │
              │  (成功/失败/跳过数量) │
              └───────────────────────┘
```

## 后端实现要点

### MqttAutoDiscoveryService

```javascript
class MqttAutoDiscoveryService {
  /**
   * 处理 Tag Topic 消息 - 自动发现变量
   */
  async handleTagMessage(configId, message) {
    const config = await this.getAutoDiscoveryConfig(configId);
    if (!config.enabled) return;

    // 解析变量列表
    const tags = this.extractList(message, config.tagTopic);

    for (const tagDef of tags) {
      const tagName = tagDef[config.tagTopic.fields.name];
      const existingTag = await this.findTagByName(
        config.targetGroupId,
        tagName
      );

      if (existingTag) {
        // 更新现有变量
        await this.updateTag(existingTag.id, {
          dataType: tagDef[config.tagTopic.fields.dataType],
          unit: tagDef[config.tagTopic.fields.unit],
          description: tagDef[config.tagTopic.fields.description],
        });
      } else {
        // 创建新变量
        await this.createTag({
          groupId: config.targetGroupId,
          name: tagName,
          dataType: tagDef[config.tagTopic.fields.dataType] || "string",
          unit: tagDef[config.tagTopic.fields.unit],
          description: tagDef[config.tagTopic.fields.description],
          source: "auto_discovery", // 标记来源
        });
      }
    }

    // 同步数据点
    await this.dataPointService.syncFromTagGroup(config.targetGroupId);
  }

  /**
   * 处理 Data Topic 消息 - 更新变量值
   */
  async handleDataMessage(configId, message) {
    const config = await this.getAutoDiscoveryConfig(configId);
    if (!config.enabled) return;

    // 解析值列表
    const values = this.extractList(message, config.dataTopic);

    for (const item of values) {
      const tagName = item[config.dataTopic.fields.name];
      const value = item[config.dataTopic.fields.value];
      const timestamp = item[config.dataTopic.fields.timestamp];
      const quality = item[config.dataTopic.fields.quality];

      // 查找变量并更新值
      const tag = await this.findTagByName(config.targetGroupId, tagName);
      if (tag) {
        await this.updateTagValue(tag.id, { value, timestamp, quality });

        // 推送实时值
        this.socketService.emitTagValue(tag.id, value, timestamp, quality);
        this.socketService.emitDataPointValue(
          tag.datapointPath,
          value,
          timestamp
        );
      }
    }
  }

  /**
   * 根据配置提取列表数据
   */
  extractList(message, topicConfig) {
    if (topicConfig.format === "json") {
      return JSONPath.query(JSON.parse(message), topicConfig.listPath);
    }
    // 其他格式处理...
  }
}
```

### MqttApiSyncService

```javascript
class MqttApiSyncService {
  /**
   * 从 RESTful API 同步变量列表
   */
  async syncFromApi(configId) {
    const config = await this.getApiSyncConfig(configId);

    // 构建请求
    const requestConfig = {
      method: config.api.method,
      url: config.api.url,
      headers: { ...config.api.headers },
    };

    // 添加认证
    if (config.api.auth?.type === "basic") {
      requestConfig.auth = {
        username: config.api.auth.username,
        password: config.api.auth.password,
      };
    } else if (config.api.auth?.type === "bearer") {
      requestConfig.headers[
        "Authorization"
      ] = `Bearer ${config.api.auth.token}`;
    } else if (config.api.auth?.type === "apikey") {
      requestConfig.headers[config.api.auth.apiKeyHeader || "X-API-Key"] =
        config.api.auth.apiKey;
    }

    // 发送请求
    const response = await axios(requestConfig);

    // 解析响应
    const tags = JSONPath.query(response.data, config.response.listPath);

    // 同步变量
    const result = await this.syncTags(
      config.targetGroupId,
      tags,
      config.response.fields
    );

    // 记录同步日志
    await this.logSyncResult(configId, result);

    return result;
  }

  /**
   * 同步变量（对比差异，批量操作）
   */
  async syncTags(groupId, remoteTags, fields) {
    const existingTags = await this.getTagsByGroup(groupId);
    const existingMap = new Map(existingTags.map((t) => [t.name, t]));

    const result = { created: 0, updated: 0, invalidated: 0 };
    const remoteNames = new Set();

    for (const tagDef of remoteTags) {
      const name = tagDef[fields.name];
      remoteNames.add(name);

      const existing = existingMap.get(name);
      if (existing) {
        // 更新
        await this.updateTag(existing.id, {
          dataType: tagDef[fields.dataType],
          unit: tagDef[fields.unit],
          description: tagDef[fields.description],
        });
        result.updated++;
      } else {
        // 创建
        await this.createTag({
          groupId,
          name,
          dataType: tagDef[fields.dataType] || "string",
          unit: tagDef[fields.unit],
          description: tagDef[fields.description],
          source: "api_sync",
        });
        result.created++;
      }
    }

    // 标记已删除的变量为失效
    for (const [name, tag] of existingMap) {
      if (!remoteNames.has(name) && tag.source === "api_sync") {
        await this.invalidateTag(tag.id);
        result.invalidated++;
      }
    }

    // 同步数据点
    await this.dataPointService.syncFromTagGroup(groupId);

    return result;
  }
}
```

### MqttCsvImportService

```javascript
class MqttCsvImportService {
  /**
   * 解析上传的文件
   */
  async parseFile(file) {
    const ext = path.extname(file.originalname).toLowerCase();

    if (ext === ".csv") {
      return this.parseCsv(file.buffer);
    } else if ([".xlsx", ".xls"].includes(ext)) {
      return this.parseExcel(file.buffer);
    }

    throw new AppError("不支持的文件格式", ErrorCodes.INVALID_INPUT);
  }

  /**
   * 预览导入结果
   */
  async previewImport(groupId, data, fieldMapping) {
    const existingTags = await this.getTagsByGroup(groupId);
    const existingMap = new Map(existingTags.map((t) => [t.name, t]));

    const preview = [];

    for (const row of data) {
      const name = row[fieldMapping.name];

      // 校验必填字段
      if (!name || name.trim() === "") {
        preview.push({ ...row, _status: "skip", _reason: "变量名为空" });
        continue;
      }

      const existing = existingMap.get(name);

      if (!existing) {
        preview.push({ ...row, _status: "create" });
      } else {
        // 检查是否有变化
        const hasChange =
          row[fieldMapping.dataType] !== existing.dataType ||
          row[fieldMapping.unit] !== existing.unit ||
          row[fieldMapping.description] !== existing.description;

        if (hasChange) {
          // 检查数据类型冲突
          if (
            row[fieldMapping.dataType] &&
            row[fieldMapping.dataType] !== existing.dataType &&
            existing.value !== null
          ) {
            preview.push({
              ...row,
              _status: "conflict",
              _reason: "数据类型不同",
            });
          } else {
            preview.push({ ...row, _status: "update" });
          }
        } else {
          preview.push({ ...row, _status: "unchanged" });
        }
      }
    }

    return preview;
  }

  /**
   * 执行导入
   */
  async executeImport(groupId, data, fieldMapping, options) {
    const result = { created: 0, updated: 0, skipped: 0, failed: 0 };

    // 覆盖模式：先清空现有变量
    if (options.mode === "overwrite") {
      await this.clearTagGroup(groupId);
    }

    for (const row of data) {
      const name = row[fieldMapping.name];

      if (!name || name.trim() === "") {
        result.skipped++;
        continue;
      }

      try {
        const existing = await this.findTagByName(groupId, name);

        if (existing && options.updateExisting) {
          await this.updateTag(existing.id, {
            dataType: row[fieldMapping.dataType] || existing.dataType,
            unit: row[fieldMapping.unit],
            description: row[fieldMapping.description],
            minValue: row[fieldMapping.minValue],
            maxValue: row[fieldMapping.maxValue],
          });
          result.updated++;
        } else if (!existing) {
          await this.createTag({
            groupId,
            name,
            dataType: row[fieldMapping.dataType] || "string",
            unit: row[fieldMapping.unit],
            description: row[fieldMapping.description],
            minValue: row[fieldMapping.minValue],
            maxValue: row[fieldMapping.maxValue],
            source: "csv_import",
          });
          result.created++;
        } else {
          result.skipped++;
        }
      } catch (error) {
        result.failed++;
      }
    }

    // 同步数据点
    await this.dataPointService.syncFromTagGroup(groupId);

    return result;
  }
}
```

## 前端界面设计

### 变量列表显示来源标记

变量列表中显示每个变量的来源方式：

```
┌─ 变量管理 ─────────────────────────────────────────────────────────────────────┐
│                                                                                 │
│  [+ 手动新建]  [🔄 同步]  [📥 导入CSV]  [📤 导出]                              │
│                                                                                 │
│  ┌───────────────────────────────────────────────────────────────────────────┐ │
│  │ 启用  变量名          当前值    类型     单位  来源          数据点       │ │
│  ├───────────────────────────────────────────────────────────────────────────┤ │
│  │ [✓]  温度传感器_01   25.6      number   ℃    📡 MQTT发现   📌 已创建   │ │
│  │ [✓]  温度传感器_02   26.1      number   ℃    📡 MQTT发现   📌 已创建   │ │
│  │ [✓]  PLC01.压力      1.25      number   MPa  🌐 API同步    📌 已创建   │ │
│  │ [✓]  设备状态_01     true      boolean  -    📄 CSV导入    📌 已创建   │ │
│  │ [✓]  手动变量_01     100       number   -    ✏️ 手动创建    📌 已创建   │ │
│  │ [✗]  失效变量_01     -         number   -    📡 MQTT发现   ⚠️ 已失效   │ │
│  └───────────────────────────────────────────────────────────────────────────┘ │
│                                                                                 │
│  📡 MQTT发现: 3 个   🌐 API同步: 1 个   📄 CSV导入: 1 个   ✏️ 手动: 1 个       │
│                                                                                 │
└─────────────────────────────────────────────────────────────────────────────────┘
```

#### 来源类型标记

| 图标 | 来源类型  | 说明                         |
| ---- | --------- | ---------------------------- |
| 📡   | MQTT 发现 | 通过 MQTT Tag Topic 自动发现 |
| 🌐   | API 同步  | 通过 RESTful API 同步        |
| 📄   | CSV 导入  | 通过 CSV/Excel 文件导入      |
| ✏️   | 手动创建  | 用户手动创建                 |

### 测试解析功能

```
┌─ 测试解析 ────────────────────────────────────────────────────────────────────┐
│                                                                                │
│  ┌─ 输入测试消息 ──────────────────────────────────────────────────────────┐ │
│  │ {                                                                        │ │
│  │   "tags": [                                                              │ │
│  │     { "name": "temp_01", "dataType": "number", "unit": "℃" },           │ │
│  │     { "name": "temp_02", "dataType": "number", "unit": "℃" }            │ │
│  │   ]                                                                      │ │
│  │ }                                                                        │ │
│  └──────────────────────────────────────────────────────────────────────────┘ │
│                                                                                │
│                                              [解析测试]                        │
│                                                                                │
│  ┌─ 解析结果 ──────────────────────────────────────────────────────────────┐ │
│  │                                                                          │ │
│  │  ✅ 成功解析 2 个变量:                                                   │ │
│  │                                                                          │ │
│  │  │ 变量名    │ 数据类型 │ 单位 │                                        │ │
│  │  ├───────────┼──────────┼──────┤                                        │ │
│  │  │ temp_01   │ number   │ ℃   │                                        │ │
│  │  │ temp_02   │ number   │ ℃   │                                        │ │
│  │                                                                          │ │
│  └──────────────────────────────────────────────────────────────────────────┘ │
│                                                                                │
└────────────────────────────────────────────────────────────────────────────────┘
```

## 常见数据格式支持

### 格式 1: 标准 NVTQ 格式

```json
{
  "values": [{ "N": "tag1", "V": 25.6, "T": 1704412800000, "Q": 192 }]
}
```

配置：

```json
{
  "listPath": "$.values",
  "fields": { "name": "N", "value": "V", "timestamp": "T", "quality": "Q" }
}
```

### 格式 2: 扁平键值对

```json
{
  "tag1": 25.6,
  "tag2": 26.1,
  "tag3": true
}
```

配置（使用脚本解析）：

```json
{
  "listPath": "$",
  "parseMode": "script",
  "script": "Object.entries(data).map(([k, v]) => ({ N: k, V: v }))"
}
```

### 格式 3: 嵌套结构

```json
{
  "device": {
    "sensors": {
      "temperature": { "value": 25.6, "unit": "℃" },
      "humidity": { "value": 60, "unit": "%" }
    }
  }
}
```

配置（使用脚本解析）：

```json
{
  "parseMode": "script",
  "script": "Object.entries(data.device.sensors).map(([k, v]) => ({ N: k, V: v.value }))"
}
```

## 与数据点的集成

无论采用哪种来源方式，创建的变量都会自动生成对应的数据点：

```
变量来源                                     自动生成数据点
─────────────────────────────────────────────────────────────────────
📡 MQTT 发现: 温度传感器_01        →    mqtt.生产连接.自动发现组.温度传感器_01
🌐 API 同步:  PLC01.压力           →    mqtt.生产连接.API同步组.PLC01.压力
📄 CSV 导入:  设备状态_01          →    mqtt.生产连接.导入组.设备状态_01
✏️ 手动创建:  自定义变量           →    mqtt.生产连接.手动组.自定义变量
```

设计器可通过数据点统一接口使用所有变量数据。

## 实施计划

### 阶段一：基础功能

| 任务              | 说明                               | 预计时间 |
| ----------------- | ---------------------------------- | -------- |
| 数据模型扩展      | 新增 `source` 字段、自动发现配置表 | 1 天     |
| MQTT 自动发现后端 | Tag/Data Topic 处理服务            | 2-3 天   |
| MQTT 自动发现前端 | 配置界面、测试解析功能             | 2 天     |

### 阶段二：扩展功能

| 任务         | 说明                      | 预计时间 |
| ------------ | ------------------------- | -------- |
| API 同步后端 | HTTP 请求、认证、定时任务 | 2 天     |
| API 同步前端 | 配置界面、同步日志        | 1.5 天   |
| CSV 导入后端 | 文件解析、预览、批量导入  | 2 天     |
| CSV 导入前端 | 上传、字段映射、预览确认  | 1.5 天   |

### 阶段三：整合优化

| 任务         | 说明                   | 预计时间 |
| ------------ | ---------------------- | -------- |
| 统一来源标记 | 变量列表显示来源、统计 | 0.5 天   |
| 数据点集成   | 所有来源自动生成数据点 | 1 天     |
| 测试与优化   | 各场景测试、性能优化   | 1.5 天   |

**总计：约 15-16 天**

## 相关文档

- [MQTT 实现说明](./mqtt-implementation.md)
- [数据点方案设计](./datapoint-design.md)
- [数据中心概述](./README.md)
