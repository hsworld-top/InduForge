# DataCenter 数据中心

DataCenter 是 InduForge 平台的数据管理模块，提供统一的数据库连接管理、MQTT 消息管理、SQL 查询和数据源配置功能。

## 核心功能

### 1. 数据点管理

数据点是 DataCenter 对所有数据源的统一抽象，提供标准化的数据访问接口。

#### 数据点类型

- **db.query**: 数据库查询结果对象
- **mqtt.tag**: MQTT 变量
- **mqtt.subscription**: MQTT 订阅消息对象
- **calc.output**: 计算单元输出（规划中）
- **opcua.node**: OPC UA 节点（规划中）
- **ws.tag**: WebSocket 变量（规划中）

#### 核心特性

- **自动生成**：保存查询/变量/订阅时自动创建对应数据点
- **统一路径**：`{类型}.{连接名}.{分组}.{名称}`，如 `mqtt.EMQX.温度组.temperature` 或 `mqtt.EMQX.设备订阅`
- **实时同步**：数据源变更时数据点自动更新
- **设计器集成**：设计器可直接枚举和使用数据点
- **失效管理**：支持查看失效数据点，提供单个或批量清理功能

详细设计请参考：

- [数据点方案设计](./datapoint-design.md)
- [数据点改造实施](./datapoint-implementation.md)

### 2. 连接管理

DataCenter 支持两大类连接：

#### 数据库连接（现有）

- **MySQL**: 5.7+ 版本，完整支持
- **PostgreSQL**: 12+ 版本，支持 Schema
- **SQL Server**: 2012+ 版本，支持 Schema

#### 消息连接

- **MQTT**: 已支持连接、订阅、消息查看、变量管理与实时推送
- **WebSocket**: 未来支持
- **OPC UA**: 未来支持

**通用功能**：

- 可视化连接配置和测试
- 连接状态实时监控（已连接/未连接/错误）
- 连接树形展示，支持展开/折叠
- 右键菜单快速操作（打开/断开/编辑/删除）

### 2. 查询管理

- 保存和管理 SQL 查询
- 查询列表展示和快速访问
- 支持参数化查询（多种参数类型）
- 查询结果分页展示
- 右键菜单管理查询（查看详情/打开/删除）

### 3. SQL 编辑器

- 基于 Monaco Editor 的代码编辑器
- SQL 语法高亮和自动补全
- SQL 美化功能
- 支持切换数据库连接
- 表选择器快速生成查询
- 参数面板支持动态参数输入

### 4. 表管理

- 表列表展示（显示行数和表类型）
- 双击表快速创建查询
- 表结构查看（字段、类型、约束等）
- 右键菜单操作（查看结构/查询数据）
- 支持表搜索和过滤

## 技术架构

### 前端技术栈

- **框架**: Vue 3 Composition API
- **UI 组件**: Element Plus
- **代码编辑器**: Monaco Editor
- **SQL 解析**: node-sql-parser
- **图标**: unplugin-icons (Tabler Icons)
- **样式**: Tailwind CSS
- **HTTP 客户端**: Axios

### 组件架构

采用模块化设计，按功能领域拆分：

- **连接组件**: 负责连接的展示和管理（ConnectionList、ConnectionItem、右键菜单）
- **数据库组件**: 按数据库类型拆分（MySQL、PostgreSQL、SQL Server），每种数据库包含 TableList 和 QueryEditor
- **对话框组件**: 统一的对话框管理（连接配置、连接详情、表结构查看）
- **共享组件**: 可复用的通用组件（StatusIndicator、MonacoEditor）

### 状态管理

- 使用 Composables 管理业务逻辑
- `useConnection`: 连接管理（CRUD、状态更新）
- `useConnectionStatus`: 连接状态管理
- `useMysql`/`usePostgres`/`useSqlServer`: 各数据库专用逻辑
- `useMqttSocket`: MQTT WebSocket 实时通信管理
- `useMqttConnection`: MQTT 连接管理
- `useMqttTagSync`: MQTT 变量同步管理

### 统一标签页系统

- **数据点标签页**：展示所有数据点，支持按类型/状态筛选、搜索和失效清理
- **表列表标签页**：展示数据库表列表，支持双击快速创建查询
- **查询标签页**：动态创建，包含 SQL 编辑器和结果展示，支持参数化查询
- **MQTT 订阅列表标签页**：管理 MQTT 订阅，支持创建、编辑、启停
- **MQTT 消息查看器标签页**：实时查看订阅消息
- **MQTT 变量管理标签页**：左右布局，左侧变量列表，右侧变量实时监控
- **计算单元标签页**：规划中
- **报警单元标签页**：规划中
- **标签页状态管理**：SQL 内容、执行结果、参数配置、修改状态等
- **标签页滚轮支持**：鼠标滚轮快速切换标签页

## 用户界面

### 布局结构

```
┌─────────────────────────────────────────────────────────────┐
│  左侧面板          │        右侧统一标签页区域              │
│                   │                                         │
│  [基础配置]       │  ┌─ 标签页 ───────────────────────┐   │
│  ○ 数据点        │  │ 数据点 │ 表列表 │ 查询1 │ ...  │   │
│                   │  └──────────────────────────────────┘   │
│  [连接管理]       │                                         │
│  [新建] [刷新]    │  ┌─ 查询编辑器 ─────────────────────┐  │
│  [搜索...]        │  │ [连接] [表] [运行] [保存] [美化] │  │
│  ● 连接1 (MySQL)  │  │                                   │  │
│  ├─ 查询 (蓝色)   │  │ SELECT * FROM devices            │  │
│  │  ├─ 设备查询  │  │ WHERE status = ?                 │  │
│  │  └─ 报警统计  │  │                                   │  │
│  └─ 表 (绿色)     │  └───────────────────────────────────┘  │
│     ├─ devices    │                                         │
│     └─ alarms     │  ┌─ 参数面板 ───────────────────────┐  │
│                   │  │ param1: [1] (number)             │  │
│  ● MQTT连接       │  └───────────────────────────────────┘  │
│  └─ 订阅 (紫色)   │                                         │
│     └─ 设备状态  │  ┌─ 查询结果 ───────────────────────┐  │
│                   │  │ 100 行，执行时间: 45ms           │  │
│  [处理逻辑]       │  │ ┌──────┬──────┬────────┐         │  │
│  ○ 计算单元      │  │ │ id   │ name │ status │         │  │
│  ○ 报警单元      │  │ ├──────┼──────┼────────┤         │  │
│                   │  │ │ 1    │ ...  │ 1      │         │  │
│                   │  │ └──────┴──────┴────────┘         │  │
│                   │  │ [分页: 1/5]                      │  │
│                   │  └───────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

### 视觉设计

- **查询区域**: 蓝色主题（`blue-50`/`blue-700`），表示已保存的查询
- **表区域**: 绿色主题（`green-50`/`green-700`），表示数据库表
- **Hover 效果**: 背景高亮 + 轻微右移动画（`hover:translate-x-0.5`）
- **状态指示**: 连接状态用颜色标识（绿色=已连接，红色=错误，灰色=未连接/断开）
- **图标系统**: 使用 Tabler Icons 提供一致的视觉语言

## 工作流程

### 创建和使用连接

1. 点击左上角"新建连接"按钮
2. 选择数据库类型（MySQL/PostgreSQL/SQL Server）
3. 填写连接信息（主机、端口、数据库、用户名、密码）
4. 点击"测试连接"确保配置正确
5. 保存连接
6. 双击连接进行连接测试并展开，自动加载表和查询列表
7. 再次双击已展开的连接可折叠

### 执行 SQL 查询

1. **快速查询**: 双击表自动创建查询（生成 `SELECT * FROM table LIMIT 100`）
2. **新建查询**: 在表列表标签页点击"新建查询"按钮
3. 在 SQL 编辑器中编写 SQL 语句
4. 可选：切换数据库连接或使用表选择器
5. 可选：配置查询参数（使用 `?` 占位符）
6. 点击"运行"执行查询
7. 查看结果（支持分页浏览）
8. 点击"保存"保存查询（输入查询名称）

### 管理已保存的查询

1. 查询保存后显示在左侧树的"查询"区域（蓝色）
2. 双击查询打开编辑
3. 修改后可重新保存或另存为新查询
4. 右键查询显示菜单（查看详情/打开/删除）
5. 删除查询会自动关闭对应的标签页

### 查看表结构

1. 右键点击表
2. 选择"查看表结构"
3. 查看字段信息（名称、类型、长度、是否为空、默认值、注释等）
4. 查看索引信息（索引名、类型、字段）

## 连接类型支持情况

### 数据库连接

#### MySQL

- ✅ 连接管理
- ✅ 表列表展示（含行数）
- ✅ 表结构查看
- ✅ SQL 查询编辑器
- ✅ 参数化查询
- ✅ SQL 美化
- ✅ 查询保存和管理

#### PostgreSQL

- ✅ 连接管理
- ✅ 表列表展示（含行数）
- ✅ 表结构查看
- ✅ SQL 查询编辑器
- ✅ 参数化查询
- ✅ SQL 美化
- ✅ 查询保存和管理
- ✅ Schema 支持

#### SQL Server

- ✅ 连接管理
- ✅ 表列表展示（含行数）
- ✅ 表结构查看
- ✅ SQL 查询编辑器
- ✅ 参数化查询
- ✅ SQL 美化
- ✅ 查询保存和管理
- ✅ Schema 支持

### 消息连接

#### MQTT

- ✅ 连接管理（创建/编辑/删除/测试/启停/状态监控）
- ✅ 主题订阅管理（列表/创建/编辑/删除/启停/消息实时查看）
- ✅ 变量组与变量管理（分组/CRUD/排序/启停/值解析）
- ✅ 变量实时监控（实时值展示、更新时间、质量状态）
- ✅ 实时数据推送（Socket.IO）
- ✅ 数据点自动生成（MQTT 变量/订阅自动创建对应数据点）
- ⏳ [变量自动发现与批量导入](./mqtt-auto-discovery.md)（MQTT/API/CSV 多种来源，规划中）
- ⏳ 数据回写功能（规划）

#### WebSocket

- ⏳ 未来支持

#### OPC UA

- ⏳ 未来支持

## 扩展性

### 添加新数据库类型

1. 在 `components/database/` 下创建新目录（如 `oracle/`）
2. 实现 `TableList.vue` 和 `QueryEditor.vue` 组件
3. 在 `composables/database/` 下创建专用逻辑（如 `useOracle.js`）
4. 在 `config/connectionTypes.js` 中注册新类型
5. 在 `components/connection/forms/` 下创建连接表单（如需要）
6. 在 `DataCenterNew.vue` 中添加对应的组件引用

### 自定义 SQL 方言

使用 `node-sql-parser` 支持不同的 SQL 方言：

- MySQL: `mysql`
- PostgreSQL: `postgresql`
- SQL Server: `transactsql`

## 性能优化（大规模数据点）

当数据点数量达到上万甚至百万级时，需要考虑以下优化策略。

> **说明**：以下为设计时预留的优化方案，可根据实际业务规模按需实施。

### 数据库层优化

#### 索引优化

```sql
-- 复合索引（项目 + 来源类型 + 路径前缀）
CREATE INDEX idx_dp_project_source_path ON data_points(project_id, source_type, path(100));

-- 状态索引
CREATE INDEX idx_dp_project_status ON data_points(project_id, status);

-- 路径 Hash 索引（精确匹配场景）
ALTER TABLE data_points ADD COLUMN path_hash CHAR(32) GENERATED ALWAYS AS (MD5(path)) STORED;
CREATE INDEX idx_dp_path_hash ON data_points(project_id, path_hash);
```

#### 分表策略

当单表数据量过大时，可按项目分表：

```
data_points_{projectId}
├── data_points_proj_001
├── data_points_proj_002
└── ...
```

#### 查询优化

```typescript
// ❌ 不推荐：全量查询
SELECT * FROM data_points WHERE project_id = ?;

// ✅ 推荐：分页 + 条件筛选
SELECT * FROM data_points
WHERE project_id = ?
  AND source_type = ?
  AND path LIKE ?
ORDER BY path
LIMIT ? OFFSET ?;

// ✅ 推荐：使用 path_hash 精确查询
SELECT * FROM data_points
WHERE project_id = ? AND path_hash = MD5(?);
```

### 缓存层优化

#### Redis 缓存架构

```
┌─────────────────────────────────────────────────────────────────┐
│                       缓存层设计                                 │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌──────────────────┐  ┌──────────────────┐  ┌────────────────┐│
│  │  值缓存 (HOT)    │  │  状态缓存 (WARM) │  │  元数据缓存    ││
│  │                  │  │                  │  │                ││
│  │  Key: dp:v:{path}│  │  Key: dp:s:{path}│  │  Key: dp:m:{id}││
│  │  TTL: 60s        │  │  TTL: 300s       │  │  TTL: 3600s    ││
│  │  数据: 最新值    │  │  数据: 连接状态  │  │  数据: 定义    ││
│  │                  │  │  quality/error   │  │  path/dataType ││
│  └──────────────────┘  └──────────────────┘  └────────────────┘│
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

#### 缓存实现

```typescript
class DataPointCacheService {
  private redis: Redis;

  // 获取数据点值（缓存优先）
  async getValue(projectId: string, path: string): Promise<any> {
    const cacheKey = `dp:v:${projectId}:${path}`;

    // 1. 尝试从缓存获取
    const cached = await this.redis.get(cacheKey);
    if (cached) {
      return JSON.parse(cached);
    }

    // 2. 缓存未命中，从源获取
    const value = await this.fetchFromSource(projectId, path);

    // 3. 写入缓存
    await this.redis.setex(cacheKey, 60, JSON.stringify(value));

    return value;
  }

  // 批量获取（减少 Redis 往返）
  async getValues(
    projectId: string,
    paths: string[]
  ): Promise<Map<string, any>> {
    const keys = paths.map((p) => `dp:v:${projectId}:${p}`);
    const values = await this.redis.mget(keys);

    const result = new Map();
    const missing: string[] = [];

    values.forEach((v, i) => {
      if (v) {
        result.set(paths[i], JSON.parse(v));
      } else {
        missing.push(paths[i]);
      }
    });

    // 批量获取缺失的
    if (missing.length > 0) {
      const fetched = await this.fetchFromSourceBatch(projectId, missing);
      // ... 写入缓存和结果
    }

    return result;
  }
}
```

### WebSocket 推送优化

#### 批量聚合推送

```typescript
// 服务端：聚合推送器
class BatchPushService {
  private buffer = new Map<string, Map<string, any>>();
  private flushInterval = 100; // 100ms 聚合

  constructor() {
    setInterval(() => this.flush(), this.flushInterval);
  }

  // 缓冲单个值更新
  push(projectId: string, path: string, value: any): void {
    if (!this.buffer.has(projectId)) {
      this.buffer.set(projectId, new Map());
    }
    this.buffer.get(projectId)!.set(path, value);
  }

  // 定时批量推送
  private flush(): void {
    for (const [projectId, values] of this.buffer) {
      if (values.size === 0) continue;

      const payload = {
        timestamp: Date.now(),
        values: Object.fromEntries(values),
      };

      // 推送到项目房间
      socketService.io
        .to(`project:${projectId}`)
        .emit("datapoint:batch", payload);

      values.clear();
    }
  }
}
```

#### 差值推送

```typescript
// 只推送变化的值
class DeltaPushService {
  private lastValues = new Map<string, any>();

  shouldPush(path: string, newValue: any): boolean {
    const lastValue = this.lastValues.get(path);

    // 首次推送
    if (lastValue === undefined) {
      this.lastValues.set(path, newValue);
      return true;
    }

    // 值相同，跳过
    if (isEqual(lastValue, newValue)) {
      return false;
    }

    // 数值类型：死区过滤
    if (typeof newValue === "number" && typeof lastValue === "number") {
      const deadband = this.getDeadband(path);
      if (Math.abs(newValue - lastValue) < deadband) {
        return false;
      }
    }

    this.lastValues.set(path, newValue);
    return true;
  }
}
```

#### 订阅聚合

```typescript
// 支持前缀订阅，减少订阅数量
class SubscriptionManager {
  // 按前缀聚合
  private prefixSubscriptions = new Map<string, Set<string>>(); // prefix -> socketIds

  // 订阅前缀
  subscribePrefixes(socketId: string, prefixes: string[]): void {
    prefixes.forEach((prefix) => {
      if (!this.prefixSubscriptions.has(prefix)) {
        this.prefixSubscriptions.set(prefix, new Set());
      }
      this.prefixSubscriptions.get(prefix)!.add(socketId);
    });
  }

  // 获取匹配的订阅者
  getSubscribers(path: string): Set<string> {
    const subscribers = new Set<string>();

    for (const [prefix, sockets] of this.prefixSubscriptions) {
      if (path.startsWith(prefix.replace("*", ""))) {
        sockets.forEach((s) => subscribers.add(s));
      }
    }

    return subscribers;
  }
}
```

### 数据点分级

```typescript
// 按更新频率和重要性分级
enum DataPointTier {
  HOT = "hot", // 高频（<1s）- 关键监控值
  WARM = "warm", // 中频（1-10s）- 常规数据
  COLD = "cold", // 低频（>10s）- 配置/状态
}

// 数据点定义扩展
interface DataPoint {
  // ... 其他字段
  tier?: DataPointTier;
  updateInterval?: number; // 最小更新间隔 (ms)
  deadband?: number; // 数值死区
}

// 按级别差异化处理
class TieredDataService {
  async getValue(path: string): Promise<any> {
    const meta = await this.getMeta(path);

    switch (meta.tier) {
      case "hot":
        return this.memoryCache.get(path);
      case "warm":
        return this.redisCache.get(path);
      case "cold":
        return this.database.get(path);
    }
  }
}
```

### 容量规划参考

| 数据点规模 | 推荐配置           | 关键优化            |
| ---------- | ------------------ | ------------------- |
| < 1 万     | 单机 4C8G          | 基础索引            |
| 1-10 万    | 单机 8C16G + Redis | 缓存 + 批量推送     |
| 10-50 万   | 集群 2-3 节点      | 分表 + 订阅聚合     |
| 50-100 万  | 集群 + 专用缓存    | 数据分级 + 差值推送 |
| > 100 万   | Kafka + 时序数据库 | 专业方案            |

### 实施优先级

1. **立即可做**（低成本高收益）：

   - 数据库索引优化
   - WebSocket 批量推送（100ms 聚合）

2. **短期实施**（中等复杂度）：

   - Redis 值缓存层
   - 前缀订阅机制
   - 差值推送

3. **长期规划**（需要架构调整）：
   - 数据点分级
   - 分表策略
   - 时序数据库（如需历史数据）

## 相关文档

- [数据点方案设计](./datapoint-design.md)
- [数据点改造实施](./datapoint-implementation.md)
- [连接管理详细说明](./connections.md)
- [查询管理详细说明](./queries.md)
- [数据库实现概述](./database-implementation.md)
- [MQTT 实现说明](./mqtt-implementation.md)
- [WebSocket 协议](../backend/websocket.md)
