# 数据点改造实施说明

## 概述

本文档详细说明数据点功能的改造实施计划，包括前后端改造内容、实施步骤、界面设计等。

## 改造范围

### 涉及模块

| 模块           | 改造内容                                         |
| -------------- | ------------------------------------------------ |
| **dev_core**   | 数据点数据模型、CRUD API、自动生成逻辑、实时推送 |
| **datacenter** | 数据点管理界面、现有查询/变量界面集成数据点显示  |
| **designer**   | 数据点选择器、数据源配置支持数据点类型           |

### 不涉及模块

- dev_ide：无需改动
- nginx / docker：无需改动

## 实施阶段

### Phase 1：后端基础能力（预计 3-4 天）

#### 1.1 数据模型

**新增表：`data_points`**

```sql
CREATE TABLE data_points (
  id              VARCHAR(36) PRIMARY KEY,
  project_id      VARCHAR(36) NOT NULL,
  path            VARCHAR(255) NOT NULL,
  name            VARCHAR(100) NOT NULL,
  description     TEXT,

  -- 来源信息
  source_type     VARCHAR(50) NOT NULL,
  source_id       VARCHAR(36),
  source_config   JSON,

  -- 数据属性
  data_type       VARCHAR(20) NOT NULL,
  unit            VARCHAR(20),
  precision_num   INT,
  default_value   TEXT,

  -- 扩展属性
  min_value       DECIMAL(20,6),
  max_value       DECIMAL(20,6),
  alarm_low       DECIMAL(20,6),
  alarm_high      DECIMAL(20,6),
  tags            JSON,

  -- 刷新策略
  refresh_mode    VARCHAR(20) DEFAULT 'auto',
  refresh_interval INT,

  -- 状态
  status          VARCHAR(20) DEFAULT 'active',

  created_at      DATETIME NOT NULL,
  updated_at      DATETIME NOT NULL,

  UNIQUE KEY uk_project_path (project_id, path),
  INDEX idx_project_id (project_id),
  INDEX idx_source_type (source_type),
  INDEX idx_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

#### 1.2 后端服务

**新增文件**：

```
dev_core/
├── models/
│   └── DataPoint.js           # Sequelize 模型
├── services/
│   └── dataPointService.js    # 数据点业务逻辑
├── controllers/
│   └── dataPointController.js # 数据点控制器
└── routes/
    └── dataPoint.routes.js    # 数据点路由
```

**API 设计**：

| 方法   | 路由                                                     | 说明               |
| ------ | -------------------------------------------------------- | ------------------ |
| GET    | `/api/v1/data/projects/:projectId/datapoints`            | 获取数据点列表     |
| GET    | `/api/v1/data/projects/:projectId/datapoints/:id`        | 获取数据点详情     |
| PUT    | `/api/v1/data/projects/:projectId/datapoints/:id`        | 更新数据点扩展属性 |
| DELETE | `/api/v1/data/projects/:projectId/datapoints/:id`        | 删除失效数据点     |
| GET    | `/api/v1/data/projects/:projectId/datapoints/:id/usages` | 获取数据点使用情况 |

#### 1.3 自动生成逻辑

**查询保存时自动生成数据点**（一个查询对应一个数据点，值为查询结果对象）：

修改 `dev_core/services/queryService.js`：

```javascript
/**
 * 保存查询并自动生成数据点
 */
async saveQuery(projectId, queryData) {
  // 1. 保存查询
  const query = await this.queryRepository.save(queryData);

  // 2. 同步数据点（查询结果对象）
  await this.dataPointService.syncFromQuery(projectId, query);

  return query;
}
```

**MQTT 变量保存时自动生成数据点**：

修改 `dev_core/services/mqttTagService.js`：

```javascript
/**
 * 保存变量并自动生成数据点
 */
async saveTag(projectId, tagData) {
  // 1. 保存变量
  const tag = await this.tagRepository.save(tagData);

  // 2. 创建或更新数据点
  await this.dataPointService.syncFromMqttTag(projectId, tag);

  return tag;
}
```

**MQTT 订阅保存时自动生成数据点**（订阅本身作为数据点，值为消息对象）：

修改 `dev_core/controllers/mqttSubscriptionController.js`：

```javascript
/**
 * 创建订阅并自动生成数据点
 */
async createSubscription(req, res, next) {
  // ...创建订阅...
  await this.dataPointService.syncFromMqttSubscription(
    projectId,
    subscription.id
  );
}
```

**数据点同步服务**：

```javascript
// dev_core/services/dataPointService.js

class DataPointService {
  /**
   * 从查询同步数据点
   */
  async syncFromQuery(projectId, query) {
    const connection = await this.getConnection(query.connectionId);
    const basePath = `db.${connection.name}.${query.name}`;

    // 获取现有数据点
    const existing = await this.getBySource("db.query", query.id);

    if (existing.length > 0) {
      await this.update(existing[0].id, {
        path: basePath,
        name: query.name,
        dataType: "object",
        sourceConfig: { mode: "result" },
      });
    } else {
      await this.create({
        projectId,
        path: basePath,
        name: query.name,
        sourceType: "db.query",
        sourceId: query.id,
        sourceConfig: { mode: "result" },
        dataType: "object",
      });
    }
  }

  /**
   * 从 MQTT 变量同步数据点
   */
  async syncFromMqttTag(projectId, tag) {
    const connection = await this.getMqttConnection(tag.configId);
    const group = await this.getMqttTagGroup(tag.groupId);
    const path = `mqtt.${connection.name}.${group.name}.${tag.name}`;

    const existingPoint = await this.getByPath(projectId, path);

    if (existingPoint) {
      // 更新
      await this.update(existingPoint.id, {
        name: tag.displayName || tag.name,
        dataType: tag.dataType,
      });
    } else {
      // 创建
      await this.create({
        projectId,
        path,
        name: tag.displayName || tag.name,
        sourceType: "mqtt.tag",
        sourceId: tag.id,
        dataType: tag.dataType,
      });
    }
  }
}
```

#### 1.4 实时推送

**Socket.IO 事件**：

```javascript
// 订阅数据点
socket.on("datapoint:subscribe", async ({ projectId, paths }) => {
  // 加入对应房间
  paths.forEach((path) => {
    socket.join(`datapoint:${projectId}:${path}`);
  });
});

// 取消订阅
socket.on("datapoint:unsubscribe", async ({ projectId, paths }) => {
  paths.forEach((path) => {
    socket.leave(`datapoint:${projectId}:${path}`);
  });
});

// 值变化时推送
// 在 MQTT 消息处理、查询刷新、计算完成时调用
function emitDataPointValue(projectId, path, value, timestamp) {
  io.to(`datapoint:${projectId}:${path}`).emit("datapoint:value", {
    path,
    value,
    timestamp,
    quality: "good",
  });
}
```

### Phase 2：数据中心前端改造（预计 4-5 天）

#### 2.1 新增数据点管理界面

**文件结构**：

```
datacenter/src/
├── views/
│   └── datapoint/
│       └── DataPointList.vue       # 数据点列表主界面
├── components/
│   └── datapoint/
│       ├── DataPointTable.vue      # 数据点表格
│       ├── DataPointDetail.vue     # 数据点详情面板
│       └── DataPointFilter.vue     # 筛选组件
├── api/
│   └── datapoint.api.js            # 数据点 API
└── composables/
    └── useDataPoint.js             # 数据点业务逻辑
```

#### 2.2 数据点列表界面

```
┌─ 数据点管理 ─────────────────────────────────────────────────────────────────┐
│                                                                               │
│  💡 数据点由查询、变量、计算单元自动生成，您可以在此查看和管理。              │
│                                                                               │
│  ┌─ 工具栏 ───────────────────────────────────────────────────────────────┐  │
│  │ [🔍 搜索路径或名称...]        [类型: 全部 ▼]  [状态: 全部 ▼]  [导出]    │  │
│  └─────────────────────────────────────────────────────────────────────────┘  │
│                                                                               │
│  ┌─ 按来源分组展示 ──────────────────────────────────────────────────────┐   │
│  │                                                                         │  │
│  │  ▼ 🗄️ 数据库查询 (23)                                                  │  │
│  │    │                                                                    │  │
│  │    │  路径                              名称        值      来源        │  │
│  │    ├─ db.生产库.设备统计.device_count   设备总数    150     设备统计    │  │
│  │    ├─ db.生产库.设备统计.online_count   在线数量    142     设备统计    │  │
│  │    └─ ...                                                               │  │
│  │                                                                         │  │
│  │  ▼ 📡 MQTT 变量 (48)                                                   │  │
│  │    │                                                                    │  │
│  │    │  路径                              名称        值      来源        │  │
│  │    ├─ mqtt.EMQX.温度组.temperature      车间温度    25.5℃  温度变量    │  │
│  │    └─ ...                                                               │  │
│  │                                                                         │  │
│  │  ▼ 📐 计算输出 (15)                                                    │  │
│  │    ├─ calc.功率计算.power               实时功率    3351W   功率计算    │  │
│  │    └─ ...                                                               │  │
│  │                                                                         │  │
│  │  ▼ ⚠️ 已失效 (2)                                                       │  │
│  │    ├─ db.生产库.旧查询.old_field        (已删除)    -       [清理]      │  │
│  │    └─ ...                                                               │  │
│  │                                                                         │  │
│  └─────────────────────────────────────────────────────────────────────────┘  │
│                                                                               │
│  共 86 个活跃数据点，2 个已失效                                               │
│                                                                               │
└───────────────────────────────────────────────────────────────────────────────┘
```

#### 2.3 左侧树增加数据点节点

修改 `datacenter/src/views/DataCenterNew.vue`：

```vue
<template>
  <!-- 左侧面板 -->
  <div class="left-panel">
    <!-- 新建按钮 -->
    <el-dropdown>
      <el-button>新建 ▼</el-button>
      <template #dropdown>
        <el-dropdown-menu>
          <el-dropdown-item>数据库连接</el-dropdown-item>
          <el-dropdown-item>MQTT 连接</el-dropdown-item>
          <el-dropdown-item divided>计算单元</el-dropdown-item>
        </el-dropdown-menu>
      </template>
    </el-dropdown>

    <!-- 数据点入口 (新增) -->
    <div class="section-header" @click="openDataPointList">
      <IconPoint class="icon" />
      <span>数据点</span>
      <span class="count">({{ datapointCount }})</span>
    </div>

    <el-divider />

    <!-- 连接树 (保留现有) -->
    <ConnectionTree />
  </div>
</template>
```

#### 2.4 查询编辑器集成数据点显示

修改查询编辑器，在查询结果下方显示自动生成的数据点（每个查询一条）：

```vue
<!-- QueryEditor.vue 新增部分 -->
<template>
  <!-- ... 现有 SQL 编辑器 ... -->

  <!-- 查询结果 -->
  <div class="query-result">
    <!-- ... 现有结果表格 ... -->
  </div>

  <!-- 自动生成的数据点 (新增) -->
  <div class="auto-datapoints" v-if="savedQueryId && datapoints.length">
    <div class="section-title">
      <IconPoint />
      <span>自动生成的数据点</span>
    </div>
    <el-table :data="datapoints" size="small">
      <el-table-column prop="name" label="名称" width="140" />
      <el-table-column prop="path" label="数据点路径">
        <template #default="{ row }">
          <span class="path">{{ row.path }}</span>
          <el-button link size="small" @click="copyPath(row.path)">
            <IconCopy />
          </el-button>
        </template>
      </el-table-column>
      <el-table-column prop="dataType" label="数据类型" width="100" />
    </el-table>
  </div>
</template>
```

#### 2.5 MQTT 变量列表集成数据点状态

修改变量列表，增加数据点状态列：

```vue
<!-- MqttTagList.vue 修改 -->
<el-table :data="tags">
  <el-table-column prop="name" label="变量名" />
  <el-table-column prop="value" label="当前值" />
  <el-table-column prop="dataType" label="数据类型" />
  
  <!-- 新增：数据点状态 -->
  <el-table-column label="数据点" width="150">
    <template #default="{ row }">
      <div v-if="row.datapointPath" class="datapoint-status">
        <IconPoint class="icon active" />
        <span class="path">{{ row.datapointPath }}</span>
      </div>
      <span v-else class="no-datapoint">-</span>
    </template>
  </el-table-column>
</el-table>
```

### Phase 3：设计器集成（预计 3-4 天）

#### 3.1 数据点选择器组件

**新增文件**：`designer/src/ui/shared/tool-panels/DataPointSelector.vue`

```vue
<template>
  <el-dialog title="选择数据点" v-model="visible" width="700px">
    <!-- 搜索和筛选 -->
    <div class="toolbar">
      <el-input
        v-model="searchText"
        placeholder="搜索数据点路径或名称..."
        clearable
      />
      <el-select v-model="filterType" placeholder="类型">
        <el-option label="全部" value="" />
        <el-option label="数据库查询" value="db.query" />
        <el-option label="MQTT 变量" value="mqtt.tag" />
        <el-option label="计算输出" value="calc.output" />
      </el-select>
    </div>

    <!-- 数据点树 -->
    <el-tree
      :data="treeData"
      :props="{ label: 'name', children: 'children' }"
      @node-click="handleSelect"
      highlight-current
    >
      <template #default="{ node, data }">
        <div class="tree-node">
          <component :is="getIcon(data.type)" class="icon" />
          <span class="name">{{ data.name }}</span>
          <span class="path" v-if="data.path">{{ data.path }}</span>
          <span class="value" v-if="data.value !== undefined">
            {{ data.value }}
          </span>
        </div>
      </template>
    </el-tree>

    <!-- 选中预览 -->
    <div class="selected-preview" v-if="selectedPoint">
      <div class="label">已选择：</div>
      <div class="path">{{ selectedPoint.path }}</div>
      <div class="info">
        <span>类型：{{ selectedPoint.dataType }}</span>
        <span v-if="selectedPoint.unit">单位：{{ selectedPoint.unit }}</span>
      </div>
    </div>

    <template #footer>
      <el-button @click="visible = false">取消</el-button>
      <el-button type="primary" @click="confirm" :disabled="!selectedPoint">
        确定
      </el-button>
    </template>
  </el-dialog>
</template>
```

#### 3.2 数据源配置支持数据点

修改 `DataSourcePanel.vue`，新增数据点类型数据源：

```vue
<!-- 数据源类型选择 -->
<el-form-item label="数据源类型">
  <el-select v-model="form.type">
    <el-option label="数据点" value="datapoint" />  <!-- 新增 -->
    <el-option label="HTTP 请求" value="http" />
    <el-option label="静态数据" value="static" />
    <el-option label="计算数据" value="computed" />
  </el-select>
</el-form-item>

<!-- 数据点配置 (新增) -->
<template v-if="form.type === 'datapoint'">
  <el-form-item label="数据点">
    <div class="datapoint-selector">
      <el-input
        v-model="form.config.path"
        placeholder="点击选择数据点"
        readonly
      />
      <el-button @click="showDataPointSelector">选择</el-button>
    </div>
  </el-form-item>

  <el-form-item label="获取模式">
    <el-radio-group v-model="form.mode">
      <el-radio label="request">单次请求</el-radio>
      <el-radio label="poll">定时轮询</el-radio>
      <el-radio label="subscription">实时订阅</el-radio>
    </el-radio-group>
  </el-form-item>
</template>
```

#### 3.3 DataSourceManager 支持数据点

修改 `designer/src/data/DataService.ts`：

```javascript
class DataSourceManager {
  // 新增数据点获取方法
  async fetchDataPoint(config) {
    const { path } = config;

    // 调用后端 API 获取数据点值
    const response = await api.get(
      `/data/projects/${this.projectId}/datapoints/value`,
      { params: { path } }
    );

    return response.data.value;
  }

  // 新增数据点订阅方法
  subscribeDataPoint(config, onData, onError) {
    const { path } = config;

    // 通过 Socket.IO 订阅
    this.socket.emit("datapoint:subscribe", {
      projectId: this.projectId,
      paths: [path],
    });

    const handler = (data) => {
      if (data.path === path) {
        onData(data.value);
      }
    };

    this.socket.on("datapoint:value", handler);

    // 返回取消订阅函数
    return () => {
      this.socket.emit("datapoint:unsubscribe", {
        projectId: this.projectId,
        paths: [path],
      });
      this.socket.off("datapoint:value", handler);
    };
  }
}
```

### Phase 4：计算单元集成（预计 2-3 天）

#### 4.1 计算单元输入选择数据点

计算单元的输入参数配置使用数据点选择器：

```vue
<!-- CalcUnitEditor.vue -->
<template>
  <div class="calc-unit-editor">
    <!-- 输入参数配置 -->
    <div class="input-section">
      <div class="section-title">输入参数</div>
      <div v-for="(input, index) in inputs" :key="index" class="input-item">
        <el-input v-model="input.name" placeholder="参数名" />
        <el-input
          v-model="input.datapoint"
          placeholder="选择数据点"
          readonly
          @click="selectDataPoint(index)"
        />
        <el-button @click="removeInput(index)">删除</el-button>
      </div>
      <el-button @click="addInput">+ 添加输入</el-button>
    </div>

    <!-- 脚本编辑器 -->
    <div class="script-section">
      <div class="section-title">计算脚本</div>
      <MonacoEditor v-model="script" language="javascript" :height="300" />
    </div>

    <!-- 输出变量（自动解析） -->
    <div class="output-section">
      <div class="section-title">输出变量（自动生成数据点）</div>
      <div v-for="output in parsedOutputs" :key="output" class="output-item">
        <IconPoint />
        <span>{{ output }}</span>
        <span class="path">→ calc.{{ unitName }}.{{ output }}</span>
      </div>
    </div>
  </div>
</template>
```

#### 4.2 计算单元保存时生成数据点

```javascript
// dev_core/services/calcUnitService.js

async saveCalcUnit(projectId, unitData) {
  // 1. 保存计算单元
  const unit = await this.calcUnitRepository.save(unitData);

  // 2. 解析输出变量
  const outputs = this.parseOutputVariables(unitData.script);

  // 3. 同步数据点
  await this.dataPointService.syncFromCalcUnit(projectId, unit, outputs);

  return unit;
}

/**
 * 解析脚本中的输出变量
 * 匹配 output.xxx = 的模式
 */
parseOutputVariables(script) {
  const regex = /output\.(\w+)\s*=/g;
  const outputs = new Set();
  let match;
  while ((match = regex.exec(script)) !== null) {
    outputs.add(match[1]);
  }
  return Array.from(outputs);
}
```

## 界面交互设计

### 查询编辑器改造

```
┌─ 查询: 设备统计 ─────────────────────────────────────────────────────────────┐
│                                                                               │
│  ┌─────────────────────────────────────────────────────────────────────┐     │
│  │ SELECT                                                               │     │
│  │   COUNT(*) as device_count,                                         │     │
│  │   SUM(CASE WHEN status='online' THEN 1 ELSE 0 END) as online_count  │     │
│  │ FROM devices                                                         │     │
│  └─────────────────────────────────────────────────────────────────────┘     │
│                                                                               │
│  [▶ 执行]  [💾 保存]                      刷新间隔: [5000] ms                │
│                                                                               │
│  ═══════════════════════════════════════════════════════════════════════════ │
│                                                                               │
│  ┌─ 查询结果 ─────────────────────────────────────────────────────────────┐ │
│  │  device_count    online_count                                          │ │
│  │  150             142                                                   │ │
│  └─────────────────────────────────────────────────────────────────────────┘ │
│                                                                               │
│  ┌─ 自动生成的数据点 📌 ──────────────────────────────────────────────────┐ │
│  │                                                                         │ │
│  │  名称          数据点路径                               数据类型  [复制]│ │
│  │  ─────────────────────────────────────────────────────────────────────  │ │
│  │  设备统计      db.生产库.设备统计                       object   📋   │ │
│  │                                                                         │ │
│  └─────────────────────────────────────────────────────────────────────────┘ │
│                                                                               │
└───────────────────────────────────────────────────────────────────────────────┘
```

### MQTT 变量保存提示

```
┌─ 新建变量 ────────────────────────────────────────────────────────────────────┐
│                                                                                │
│  所属变量组:  [温度传感器 ▼]                                                   │
│                                                                                │
│  变量名称:    [temperature                              ]                     │
│  显示名称:    [车间温度                                  ]                     │
│  数据类型:    [number ▼]                                                       │
│  单位:        [℃                  ]                                           │
│                                                                                │
│  ═══════════════════════════════════════════════════════════════════════════  │
│                                                                                │
│  💡 保存后将自动创建数据点：                                                   │
│                                                                                │
│     📌 mqtt.EMQX.温度传感器.temperature                                        │
│                                                                                │
│     此数据点可在设计器中直接使用                                               │
│                                                                                │
│                                            [取消]    [保存]                    │
│                                                                                │
└────────────────────────────────────────────────────────────────────────────────┘
```

### 设计器数据点选择器

```
┌─ 选择数据点 ──────────────────────────────────────────────────────────────────┐
│                                                                                │
│  ┌─ 搜索和筛选 ───────────────────────────────────────────────────────────┐  │
│  │ [🔍 搜索数据点路径或名称...]                    [类型: 全部 ▼]          │  │
│  └─────────────────────────────────────────────────────────────────────────┘  │
│                                                                                │
│  ┌─ 数据点树 ─────────────────────────────────────────────────────────────┐  │
│  │                                                                         │  │
│  │  ▼ 🗄️ 数据库查询                                                       │  │
│  │    │ ▼ 生产库                                                          │  │
│  │    │   │ ▼ 设备统计                                                    │  │
│  │    │   │   ├─ device_count        150                                  │  │
│  │    │   │   └─ online_count        142                                  │  │
│  │    │   └─ 报警统计                                                     │  │
│  │    │       └─ ...                                                      │  │
│  │                                                                         │  │
│  │  ▼ 📡 MQTT 变量                                                        │  │
│  │    │ ▼ EMQX                                                            │  │
│  │    │   │ ▼ 温度传感器                                                  │  │
│  │    │   │   ├─ temperature  ●      25.5 ℃                               │  │
│  │    │   │   └─ humidity            60 %                                 │  │
│  │    │   └─ 压力传感器                                                   │  │
│  │    │       └─ ...                                                      │  │
│  │                                                                         │  │
│  │  ▼ 📐 计算输出                                                         │  │
│  │    │ ▼ 功率计算                                                        │  │
│  │    │   ├─ power                   3351 W                               │  │
│  │    │   └─ efficiency              85.5 %                               │  │
│  │                                                                         │  │
│  └─────────────────────────────────────────────────────────────────────────┘  │
│                                                                                │
│  ┌─ 已选择 ───────────────────────────────────────────────────────────────┐  │
│  │  📌 mqtt.EMQX.温度传感器.temperature                                    │  │
│  │  类型: number    单位: ℃    当前值: 25.5                                │  │
│  └─────────────────────────────────────────────────────────────────────────┘  │
│                                                                                │
│                                            [取消]    [确定]                    │
│                                                                                │
└────────────────────────────────────────────────────────────────────────────────┘
```

## 实施时间线

| 阶段    | 任务             | 预计时间 | 依赖             |
| ------- | ---------------- | -------- | ---------------- |
| Phase 1 | 后端基础能力     | 3-4 天   | -                |
| Phase 2 | 数据中心前端改造 | 4-5 天   | Phase 1          |
| Phase 3 | 设计器集成       | 3-4 天   | Phase 1, Phase 2 |
| Phase 4 | 计算单元集成     | 2-3 天   | Phase 1, Phase 3 |
| 测试    | 集成测试和修复   | 2-3 天   | 全部             |

**总计：约 14-19 天**

## 测试要点

### 功能测试

| 测试项     | 验证内容                     |
| ---------- | ---------------------------- |
| 查询保存   | 保存查询后自动生成数据点     |
| 查询修改   | 修改查询后数据点同步更新     |
| 查询删除   | 删除查询后数据点标记失效     |
| 变量创建   | 创建变量后自动生成数据点     |
| 变量重命名 | 重命名变量后数据点路径更新   |
| 数据点列表 | 按类型分组显示，支持搜索筛选 |
| 设计器选择 | 能够浏览和选择数据点         |
| 实时订阅   | 数据点值变化实时推送到设计器 |

### 边界测试

| 测试项     | 验证内容                     |
| ---------- | ---------------------------- |
| 重复路径   | 同名查询/变量的处理          |
| 特殊字符   | 名称包含特殊字符时的路径生成 |
| 大量数据点 | 1000+ 数据点的性能           |
| 并发操作   | 多用户同时操作的一致性       |

## 回滚方案

如需回滚：

1. 数据库：删除 `data_points` 表
2. 后端：回退相关服务代码
3. 前端：回退相关组件代码
4. 配置：恢复原有配置

数据点功能对现有功能无破坏性依赖，回滚不影响现有查询、变量等功能。

## 相关文档

- [数据点方案设计](./datapoint-design.md)
- [数据中心概述](./README.md)
- [数据绑定系统](../designer/data-binding.md)
