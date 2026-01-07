# 数据绑定系统 v2

本文档描述重构后的数据绑定系统，核心特性：三态数据隔离、数据点状态追踪、统一绑定结构。

## 1. 三态数据隔离

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           数据隔离三态模型                                   │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  设计态（Edit Mode）                                                        │
│  ├─ 不连接真实数据源                                                        │
│  ├─ 组件显示占位符或 designMock 数据                                        │
│  ├─ 数据点路径仅用于校验和自动补全                                          │
│  └─ 数据中心仅提供数据点目录（元数据），不返回实时值                        │
│                                                                             │
│  预览态（Preview Mode）                                                     │
│  ├─ 直接调用开发系统数据中心 API                                            │
│  ├─ 获取真实数据并渲染                                                      │
│  ├─ 使用开发环境的数据源配置                                                │
│  └─ 用于开发人员验证绑定是否正确                                            │
│                                                                             │
│  运行态（Runtime Mode）                                                     │
│  ├─ 独立运行，不依赖平台                                                    │
│  ├─ 通过 ConnectionProfile 连接节点侧数据源                                 │
│  ├─ 数据源 endpoint 由部署时注入                                            │
│  └─ 数据中心服务可部署在节点侧或远程                                        │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

**关键原则**：设计态只做"配置"，不做"连接"。开发时的数据只是开发时使用，运行时数据源完全独立配置。

## 2. Binding 结构

### 2.1 数据点绑定（推荐）

```json
{
  "text": {
    "kind": "datapoint",
    "provider": "dc_main",
    "datapointId": "dp_xxx",
    "path": "mqtt.EMQX.温度组.temperature",
    "transform": [
      { "op": "toFixed", "args": [1] },
      { "op": "suffix", "args": ["℃"] }
    ],
    "fallback": "--",
    "designMock": "25.5℃"
  }
}
```

| 字段        | 类型   | 说明                                 |
| ----------- | ------ | ------------------------------------ |
| kind        | string | 固定为 `datapoint`                   |
| provider    | string | 数据提供者别名（引用 dataProviders） |
| datapointId | string | 数据点 UUID（稳定标识，用于追踪）    |
| path        | string | 数据点路径（人类可读，用于显示）     |
| transform   | array  | 转换操作链                           |
| fallback    | any    | 数据无效时的默认值                   |
| designMock  | any    | 设计态显示的 mock 值                 |

### 2.2 变量绑定

```json
{
  "disabled": {
    "kind": "var",
    "scope": "page",
    "name": "isLoading",
    "transform": [],
    "fallback": false
  }
}
```

### 2.3 表达式绑定

```json
{
  "visible": {
    "kind": "expr",
    "expr": "{{ $dp['mqtt.EMQX.温度组.temperature'] > 30 }}",
    "fallback": true
  }
}
```

### 2.4 Transform 操作

| 操作    | 参数       | 说明       |
| ------- | ---------- | ---------- |
| toFixed | [digits]   | 保留小数位 |
| round   | []         | 四舍五入   |
| floor   | []         | 向下取整   |
| ceil    | []         | 向上取整   |
| prefix  | [str]      | 添加前缀   |
| suffix  | [str]      | 添加后缀   |
| map     | [mapping]  | 值映射     |
| format  | [template] | 格式化模板 |

## 3. 数据点状态机制

### 3.1 状态分类

| 状态    | 含义                       | 触发条件                     |
| ------- | -------------------------- | ---------------------------- |
| active  | 正常可用                   | 数据源正常，最近有数据       |
| invalid | 失效（需人工处理）         | 数据源删除/配置变更/解析失败 |
| unknown | 未知（引用了不存在的路径） | 手写路径/导入/表达式拼接     |

### 3.2 状态查询 API

```typescript
// 批量查询数据点状态
POST /api/v1/datapoints/status
Body: {
  projectId: string,
  refs: Array<{ path: string, providerAlias?: string }>
}
Response: Array<{
  path: string,
  status: 'active' | 'invalid' | 'unknown',
  status_reason?: string,
  last_seen_at?: string,
  dataType?: string
}>
```

### 3.3 DiagnosticsStore

```typescript
class DiagnosticsStore {
  // 状态缓存
  private statusCache: Map<string, DatapointStatus>;

  // 批量获取状态
  async fetchStatuses(paths: string[]): Promise<Map<string, DatapointStatus>>;

  // 获取单个状态
  getStatus(path: string): DatapointStatus | undefined;

  // 标记失效
  markInvalid(path: string, reason: string): void;

  // 获取影响的节点
  getAffectedNodes(path: string): NodeInfo[];

  // 订阅状态变化
  onStatusChange(
    handler: (path: string, status: DatapointStatus) => void
  ): void;
}
```

### 3.4 UI 展示

**数据点选择器**：

```
┌─ 选择数据点 ──────────────────────────────────────────┐
│                                                        │
│  ▼ 📡 MQTT 变量                                       │
│    │ ▼ EMQX                                           │
│    │   │ ▼ 温度传感器                                 │
│    │   │   ├─ 🟢 temperature    25.5 ℃               │
│    │   │   └─ 🔴 humidity       (已失效)              │
│                                                        │
│  ▼ 🗄️ 数据库查询                                      │
│    │ ▼ 生产库                                         │
│    │   │ ▼ 设备统计                                   │
│    │   │   ├─ 🟢 device_count   150                   │
│    │   │   └─ 🟡 online_count   (未知)                │
│                                                        │
└────────────────────────────────────────────────────────┘

🟢 active    🔴 invalid    🟡 unknown
```

**属性面板**：

```
┌─ 绑定配置 ──────────────────────────────────────────┐
│                                                      │
│  属性: text                                          │
│  类型: 数据点绑定                                    │
│                                                      │
│  数据点: mqtt.EMQX.温度组.temperature               │
│          🔴 失效 - 数据源已删除                      │
│             [查看影响] [选择其他]                    │
│                                                      │
│  转换: toFixed(1) → suffix("℃")                     │
│  降级值: "--"                                        │
│  设计态值: "25.5℃"                                  │
│                                                      │
└──────────────────────────────────────────────────────┘
```

## 4. 预览态数据连接

### 4.1 架构

预览态**直接调用开发系统的数据中心 API**，无需 iframe 或 Bridge 通信：

```
┌─────────────────────────────────────────────────────────────────┐
│                     Designer (开发环境)                          │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  设计态                          预览态                          │
│  ┌───────────────┐              ┌───────────────┐               │
│  │ DesignRenderer │              │RuntimeRenderer│               │
│  │               │              │               │               │
│  │  显示 Mock 值  │    切换模式   │  显示真实数据  │               │
│  │               │  ─────────▶  │               │               │
│  └───────────────┘              └───────┬───────┘               │
│                                         │                       │
│                                         ▼                       │
│                            ┌───────────────────────┐            │
│                            │   DataService (API)   │            │
│                            │                       │            │
│                            │  HTTP + WebSocket     │            │
│                            └───────────┬───────────┘            │
│                                        │                        │
└────────────────────────────────────────┼────────────────────────┘
                                         │
                                         ▼
┌─────────────────────────────────────────────────────────────────┐
│                     dev_core (后端)                              │
├─────────────────────────────────────────────────────────────────┤
│  /api/v1/datapoints/subscribe      (WebSocket 订阅)             │
│  /api/v1/datapoints/:path/value    (获取单值)                   │
│  /api/v1/queries/:id/execute       (执行查询)                   │
│  /api/v1/datapoints/status         (批量获取状态)               │
└─────────────────────────────────────────────────────────────────┘
```

### 4.2 DataService 实现

```typescript
class DataService {
  private socket: Socket | null = null;
  private baseUrl: string;
  private cache: Map<string, any> = new Map();
  private subscribers: Map<string, Set<(value: any) => void>> = new Map();

  constructor(baseUrl: string = "/api/v1") {
    this.baseUrl = baseUrl;
  }

  // 连接数据中心
  async connect(): Promise<void> {
    this.socket = io("/datapoint", {
      transports: ["websocket"],
    });

    this.socket.on("connect", () => {
      console.log("DataService connected");
    });

    this.socket.on("datapoint:value", (data: { path: string; value: any }) => {
      this.cache.set(data.path, data.value);
      this.notifySubscribers(data.path, data.value);
    });
  }

  // 订阅数据点
  subscribe(path: string, callback: (value: any) => void): () => void {
    // 发送订阅请求
    if (!this.subscribers.has(path)) {
      this.subscribers.set(path, new Set());
      this.socket?.emit("datapoint:subscribe", { path });
    }

    // 注册回调
    this.subscribers.get(path)!.add(callback);

    // 立即返回缓存值
    if (this.cache.has(path)) {
      callback(this.cache.get(path));
    }

    // 返回取消订阅函数
    return () => {
      this.subscribers.get(path)?.delete(callback);
      if (this.subscribers.get(path)?.size === 0) {
        this.socket?.emit("datapoint:unsubscribe", { path });
        this.subscribers.delete(path);
      }
    };
  }

  // 批量订阅
  subscribeMany(paths: string[]): void {
    paths.forEach((path) => {
      if (!this.subscribers.has(path)) {
        this.subscribers.set(path, new Set());
      }
    });
    this.socket?.emit("datapoint:subscribe:batch", { paths });
  }

  // 获取数据点当前值
  async getValue(path: string): Promise<any> {
    if (this.cache.has(path)) {
      return this.cache.get(path);
    }
    const response = await fetch(
      `${this.baseUrl}/datapoints/${encodeURIComponent(path)}/value`
    );
    const data = await response.json();
    this.cache.set(path, data.value);
    return data.value;
  }

  // 执行查询
  async executeQuery(queryId: string, params?: object): Promise<any> {
    const response = await fetch(`${this.baseUrl}/queries/${queryId}/execute`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(params),
    });
    return response.json();
  }

  // 断开连接
  disconnect(): void {
    this.socket?.disconnect();
    this.socket = null;
    this.subscribers.clear();
    this.cache.clear();
  }

  private notifySubscribers(path: string, value: any): void {
    this.subscribers.get(path)?.forEach((cb) => cb(value));
  }
}
```

### 4.3 预览模式切换

```typescript
// stores/designerStore.ts
const useDesignerStore = defineStore("designer", {
  state: () => ({
    mode: "edit" as "edit" | "preview",
    dataService: null as DataService | null,
  }),

  actions: {
    async enterPreview() {
      this.dataService = new DataService();
      await this.dataService.connect();

      // 收集当前页面所有数据点依赖
      const paths = this.collectDatapointPaths();
      this.dataService.subscribeMany(paths);

      this.mode = "preview";
    },

    exitPreview() {
      this.dataService?.disconnect();
      this.dataService = null;
      this.mode = "edit";
    },
  },
});
```

## 5. 表达式引擎

### 5.1 语法

```javascript
// 访问数据点
{
  {
    $dp["mqtt.EMQX.温度组.temperature"];
  }
}

// 访问变量
{
  {
    $vars.page.selectedId;
  }
}
{
  {
    $vars.global.currentUser.name;
  }
}

// 条件表达式
{
  {
    $dp["device.status"] === 1 ? "运行" : "停止";
  }
}

// 内置函数
{
  {
    format($dp["device.temperature"], "%.1f℃");
  }
}
{
  {
    dateFormat($dp["device.updateTime"], "YYYY-MM-DD HH:mm:ss");
  }
}
```

### 5.2 上下文

```typescript
interface ExpressionContext {
  $dp: Record<string, any>; // 数据点值
  $vars: {
    page: Record<string, any>; // 页面变量
    global: Record<string, any>; // 全局变量
  };
  $props: Record<string, any>; // 组件属性
  $event: any; // 事件对象（仅在事件处理中）
}
```

### 5.3 内置函数

| 函数       | 说明       | 示例                           |
| ---------- | ---------- | ------------------------------ |
| format     | 数值格式化 | `format(123.456, '%.2f')`      |
| dateFormat | 日期格式化 | `dateFormat(ts, 'YYYY-MM-DD')` |
| ifNull     | 空值替换   | `ifNull(value, '--')`          |
| round      | 四舍五入   | `round(3.14159, 2)`            |
| abs        | 绝对值     | `abs(-10)`                     |
| min/max    | 最值       | `min(a, b, c)`                 |
| clamp      | 限制范围   | `clamp(value, 0, 100)`         |

## 6. 实现步骤

### Step 1：定义类型

```typescript
// data/types.ts
export interface Binding { ... }
export interface DatapointBinding { ... }
export interface DiagnosticsInfo { ... }
```

### Step 2：实现 DiagnosticsStore

```typescript
// data/diagnosticsStore.ts
// 1. 状态缓存管理
// 2. 批量查询 API
// 3. 实时推送订阅
```

### Step 3：实现 ExpressionEngine

```typescript
// data/expressionEngine.ts
// 1. 表达式解析
// 2. 上下文构建
// 3. 内置函数注册
```

### Step 4：实现 DataService

```typescript
// data/dataService.ts
// 1. WebSocket 连接管理
// 2. 订阅管理
// 3. 缓存与通知
```

### Step 5：实现 MockDataProvider

```typescript
// data/mockDataProvider.ts
// 1. 读取 designMock 值
// 2. 类型推断默认值
// 3. 组件默认值
```

### Step 6：UI 集成

```typescript
// 1. 数据点选择器添加状态显示
// 2. 属性面板添加诊断信息
// 3. 发布前校验面板
```

## 7. 测试要点

- [ ] 三态切换正确性
- [ ] 数据点状态查询与缓存
- [ ] WebSocket 连接与订阅
- [ ] 表达式解析正确性
- [ ] Transform 操作链
- [ ] 失效数据点 UI 展示
- [ ] 发布校验规则

---

**相关文档**：

- [Schema 设计](./schema-design.md)
- [变量系统](./vars-system.md)
- [运行时引擎](./runtime-engine.md)
