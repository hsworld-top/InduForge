# 最佳实践指南

本文档汇总 Designer 开发中的最佳实践，包括性能优化、安全建议、命名规范等。

## 1. 性能优化

### 1.1 数据源优化

**合理设置轮询间隔**：

```json
// ❌ 不推荐：过于频繁的轮询
{
  "mode": "poll",
  "interval": 100
}

// ✅ 推荐：根据实际需求设置
{
  "mode": "poll",
  "interval": 5000
}

// ✅ 更推荐：使用订阅模式
{
  "mode": "subscription"
}
```

**启用数据缓存**：

```json
{
  "dataProviders": {
    "dc_main": {
      "cache": {
        "enabled": true,
        "ttl": 30000,
        "maxSize": 1000
      }
    }
  }
}
```

**按需订阅**：

```javascript
// ❌ 不推荐：订阅所有数据点
dataService.subscribeAll();

// ✅ 推荐：只订阅当前页面需要的数据点
dataService.subscribe(manifest.dataRequirements.datapoints);
```

### 1.2 组件优化

**避免深层嵌套**：

```json
// ❌ 不推荐：超过5层嵌套
{
  "children": [
    {
      "children": [
        {
          "children": [
            {
              "children": [
                { "children": [{ "children": [] }] }
              ]
            }
          ]
        }
      ]
    }
  ]
}

// ✅ 推荐：扁平化结构，使用布局容器
{
  "type": "GridContainer",
  "children": ["node_1", "node_2", "node_3"]
}
```

**大列表使用虚拟滚动**：

```json
{
  "type": "VirtualList",
  "props": {
    "itemHeight": 48,
    "overscan": 5,
    "data": "{{ $dp['db.query.设备列表'] }}"
  }
}
```

**复杂计算使用 computed 数据源**：

```json
// ❌ 不推荐：在绑定表达式中做复杂计算
{
  "bindings": {
    "data": {
      "kind": "expr",
      "expr": "{{ $array.filter($dp['devices'], d => d.status === 'active').map(d => ({ ...d, efficiency: d.output / d.target * 100 })).sort((a, b) => b.efficiency - a.efficiency) }}"
    }
  }
}

// ✅ 推荐：使用计算数据点或变量
{
  "bindings": {
    "data": {
      "kind": "datapoint",
      "path": "calc.设备效率排行"
    }
  }
}
```

### 1.3 动画优化

**使用 CSS 动画优先**：

```json
// ✅ 推荐：CSS transform 和 opacity（GPU 加速）
{
  "type": "rotate",
  "config": {
    "duration": 2000
  }
}

// ❌ 避免：频繁改变 width/height/margin
```

**离屏暂停**：

```json
{
  "animations": [
    {
      "pauseWhenHidden": true
    }
  ]
}
```

**限制同时运行的动画数量**：

```json
{
  "runtimeConfig": {
    "animation": {
      "maxConcurrent": 50
    }
  }
}
```

### 1.4 资源优化

**图片懒加载**：

```json
{
  "type": "Image",
  "props": {
    "lazy": true,
    "placeholder": "assets://images/placeholder.png"
  }
}
```

**资源压缩**：

- 图片：使用 WebP 格式，合适的尺寸
- 视频：使用 H.264/H.265，按需加载
- 图标：使用 SVG 或 Icon Font

**大文件外置**：

```json
{
  "assetsById": {
    "video_intro": {
      "type": "video",
      "storageMode": "external",
      "externalUrl": "https://cdn.example.com/videos/intro.mp4"
    }
  }
}
```

### 1.5 大规模数据点优化

当工程涉及上万甚至百万级数据点时，需要特别注意以下优化策略。

#### 1.5.1 数据点选择器

**虚拟滚动**：数据点列表必须使用虚拟滚动，避免渲染全部 DOM。

```typescript
// 使用 @vueuse/core 的 useVirtualList
import { useVirtualList } from "@vueuse/core";

const { list, containerProps, wrapperProps } = useVirtualList(
  datapoints, // 可能有 10 万条
  { itemHeight: 36 }
);
```

**分页加载 + 搜索**：不要一次性加载全部数据点。

```typescript
// ❌ 不推荐：全量加载
const allDatapoints = await api.getDatapoints();

// ✅ 推荐：分页 + 搜索
const loadDatapoints = async (params: {
  page: number;
  pageSize: number;
  keyword?: string;
  sourceType?: string;
}) => {
  return api.getDatapoints(params); // 服务端分页
};

// 搜索使用 debounce
const searchDatapoints = debounce(async (keyword: string) => {
  if (keyword.length < 2) return []; // 至少 2 字符
  return api.searchDatapoints({ keyword, limit: 100 });
}, 300);
```

**树形结构懒加载**：按层级展开加载。

```typescript
// 首次只加载顶层（连接列表）
const connections = await api.getConnections();

// 展开时加载子节点
const loadChildren = async (node: TreeNode) => {
  if (node.type === "connection") {
    return api.getDatapointGroups(node.id);
  }
  if (node.type === "group") {
    return api.getDatapointsByGroup(node.id, { limit: 100 });
  }
};
```

#### 1.5.2 WebSocket 订阅优化

**前缀订阅**：避免逐个订阅数据点。

```typescript
// ❌ 不推荐：逐个订阅
paths.forEach((path) => {
  socket.emit("datapoint:subscribe", { path });
});

// ✅ 推荐：批量订阅
socket.emit("datapoint:subscribe", {
  projectId,
  paths, // 一次性传入
});

// ✅ 更推荐：前缀订阅（如果后端支持）
socket.emit("datapoint:subscribe", {
  projectId,
  prefixes: ["mqtt.EMQX.温度组.*"], // 通配符
});
```

**批量值更新**：合并处理推送的值。

```typescript
// 值缓存 Map
const valueCache = reactive(new Map<string, DataPointValue>());

// 监听批量推送
socket.on(
  "datapoint:batch",
  (data: { timestamp: number; values: Record<string, any> }) => {
    // 批量更新，只触发一次响应式更新
    const batch = new Map(valueCache);
    Object.entries(data.values).forEach(([path, value]) => {
      batch.set(path, { value, timestamp: data.timestamp });
    });
    // 替换整个 Map
    valueCache.clear();
    batch.forEach((v, k) => valueCache.set(k, v));
  }
);
```

**页面级订阅管理**：离开页面时取消订阅。

```typescript
// 在页面组件中
const subscriptions = new Set<string>();

onMounted(() => {
  // 收集当前页面需要的数据点
  const paths = collectPageDatapoints(pageSchema);
  paths.forEach((p) => subscriptions.add(p));

  // 订阅
  socket.emit("datapoint:subscribe", { projectId, paths: [...subscriptions] });
});

onUnmounted(() => {
  // 取消订阅
  socket.emit("datapoint:unsubscribe", {
    projectId,
    paths: [...subscriptions],
  });
  subscriptions.clear();
});
```

#### 1.5.3 绑定校验优化

**增量校验**：只校验变更的部分。

```typescript
// ❌ 不推荐：每次保存全量校验
const validateAll = async (doc: DocumentModel) => {
  const allBindings = collectAllBindings(doc); // 可能有上万条
  return validateBindings(allBindings);
};

// ✅ 推荐：增量校验
const validateChanged = async (changedNodes: string[], doc: DocumentModel) => {
  const changedBindings = changedNodes.flatMap((nodeId) =>
    collectNodeBindings(doc, nodeId)
  );
  return validateBindings(changedBindings);
};
```

**后台校验**：使用 Web Worker 或后端校验。

```typescript
// 大规模校验放到 Web Worker
const validationWorker = new Worker("/workers/validation.js");

validationWorker.postMessage({
  type: "validate",
  bindings: collectAllBindings(doc),
});

validationWorker.onmessage = (e) => {
  if (e.data.type === "result") {
    updateDiagnostics(e.data.errors, e.data.warnings);
  }
};
```

#### 1.5.4 运行时优化

**数据点分级**：按更新频率分级处理。

```typescript
// 数据点分级策略
enum DataPointTier {
  HOT = "hot", // 高频（<1s）- 实时监控
  WARM = "warm", // 中频（1-10s）- 常规数据
  COLD = "cold", // 低频（>10s）- 配置状态
}

// 运行时按级别差异化处理
class TieredSubscriptionManager {
  // HOT: 实时推送，内存缓存
  // WARM: 100ms 聚合推送，Redis 缓存
  // COLD: 按需拉取
}
```

**值变化检测**：避免无意义的更新。

```typescript
// 只有值真正变化时才更新
const updateValue = (path: string, newValue: any) => {
  const current = valueCache.get(path);

  // 深度比较或使用 deadband
  if (isEqual(current?.value, newValue)) {
    return; // 值未变化，跳过
  }

  // 数值类型使用死区过滤
  if (typeof newValue === "number" && typeof current?.value === "number") {
    const deadband = getDeadband(path);
    if (Math.abs(newValue - current.value) < deadband) {
      return; // 在死区内，跳过
    }
  }

  valueCache.set(path, { value: newValue, timestamp: Date.now() });
};
```

#### 1.5.5 设计态性能建议

| 场景         | 数量级     | 建议              |
| ------------ | ---------- | ----------------- |
| 数据点选择器 | > 1000     | 必须分页 + 搜索   |
| 数据点树     | > 5000     | 懒加载 + 虚拟滚动 |
| 绑定校验     | > 500 节点 | 增量校验          |
| 诊断面板     | > 100 问题 | 分组 + 分页       |
| 大纲树       | > 200 节点 | 虚拟滚动          |

## 2. 安全建议

### 2.1 表达式安全

**表达式沙箱限制**：

```javascript
// ❌ 禁止的表达式
{
  {
    eval("malicious code");
  }
}
{
  {
    window.location = "evil.com";
  }
}
{
  {
    new Function("alert(1)")();
  }
}

// ✅ 允许的表达式
{
  {
    $dp["device.temp"] > 80 ? "过热" : "正常";
  }
}
{
  {
    $format.number(value, 2);
  }
}
```

**自定义脚本限制**：

```json
{
  "runtimeConfig": {
    "security": {
      "scriptSandbox": true,
      "scriptTimeout": 5000,
      "allowedGlobals": ["Math", "JSON", "Date"]
    }
  }
}
```

### 2.2 权限控制

**最小权限原则**：

```json
{
  "permissions": {
    "visible": { "allowRoles": ["admin", "operator", "viewer"] },
    "enable": { "allowRoles": ["admin", "operator"] }
  },
  "events": {
    "click": [
      {
        "type": "writeTag",
        "permissions": { "allowRoles": ["admin"] }
      }
    ]
  }
}
```

**敏感操作二次确认**：

```json
{
  "type": "writeTag",
  "config": {
    "path": "device.emergency_stop",
    "value": 1,
    "confirm": true,
    "confirmConfig": {
      "title": "紧急停机确认",
      "content": "此操作将停止所有设备，确定执行吗？",
      "type": "warning"
    }
  }
}
```

### 2.3 数据验证

**所有用户输入必须验证**：

```json
{
  "validation": {
    "rules": [
      { "required": true },
      { "type": "number" },
      { "min": 0, "max": 100 }
    ]
  }
}
```

**写入前验证**：

```json
{
  "events": {
    "click": [
      {
        "type": "condition",
        "config": {
          "if": "{{ $vars.page.inputValue >= 0 && $vars.page.inputValue <= 100 }}",
          "then": [
            {
              "type": "writeTag",
              "config": {
                "path": "device.setpoint",
                "value": "{{ $vars.page.inputValue }}"
              }
            }
          ],
          "else": [
            {
              "type": "notify",
              "config": { "type": "error", "message": "输入值超出范围" }
            }
          ]
        }
      }
    ]
  }
}
```

### 2.4 通信安全

**API 调用安全**：

```typescript
// ✅ 推荐：使用统一的 request 封装
const response = await request.get("/api/v1/datapoints/value", {
  params: { path },
});

// ✅ 推荐：Token 从 Storage 统一读取
const token = localStorage.getItem("access_token");
request.defaults.headers.common["Authorization"] = `Bearer ${token}`;
```

**WebSocket 连接管理**：

```typescript
// ✅ 推荐：使用重连机制
const socket = io("/datapoint", {
  transports: ["websocket"],
  reconnection: true,
  reconnectionAttempts: 5,
  reconnectionDelay: 1000,
});

// ✅ 推荐：监听连接状态
socket.on("connect_error", (error) => {
  console.error("DataService connection error:", error);
  notifyUser("数据连接失败，请检查网络");
});
```

## 3. 命名规范

### 3.1 ID 命名

| 类型       | 前缀   | 示例                         |
| ---------- | ------ | ---------------------------- |
| 页面       | page\_ | page_monitor, page_login     |
| 节点       | node\_ | node_header, node_chart_temp |
| 数据点     | dp\_   | dp_temp_001                  |
| 变量       | -      | isLoading, selectedId        |
| 动作       | act\_  | act_submit, act_refresh      |
| 动画       | anim\_ | anim_rotate, anim_flash      |
| 自定义组件 | cc\_   | cc_standard_pump_v1          |
| 资源       | 按类型 | img_logo, icon_motor         |

### 3.2 路径命名

**页面路径**：

```
/                    # 首页
/login               # 登录页
/monitor             # 监控页
/monitor/:deviceId   # 设备详情（带参数）
/settings            # 设置页
/admin               # 管理页
```

**数据点路径**：

```
mqtt.{连接名}.{分组}.{标签名}
db.query.{查询名}
calc.{计算单元}.{输出名}
```

### 3.3 变量命名

**页面变量**：

```javascript
// 布尔值：is/has/can/should 前缀
isLoading;
isEditing;
hasError;
canSubmit;

// 选中/当前：selected/current 前缀
selectedId;
selectedDevice;
currentTab;
currentPage;

// 列表/集合：复数形式
devices;
selectedIds;
filteredItems;

// 表单数据：formData 或具体名称
formData;
loginForm;
settingsForm;
```

**全局变量**：

```javascript
// 用户信息
currentUser;

// 应用状态
theme;
locale;
```

### 3.4 事件命名

```javascript
// 组件事件
click;
dblclick;
change;
input;
focus;
blur;

// 自定义事件：on 前缀 + 动词
onSelect;
onConfirm;
onCancel;
onRefresh;
```

## 4. 项目结构建议

### 4.1 页面组织

```
pages/
├── home/               # 首页模块
│   ├── page_home       # PC 视图
│   └── page_home_bs    # 大屏视图
├── monitor/            # 监控模块
│   ├── page_overview   # 总览
│   └── page_detail     # 详情
├── settings/           # 设置模块
└── admin/              # 管理模块
```

### 4.2 自定义组件组织

```
customComponents/
├── basic/              # 基础组件
│   ├── cc_status_badge
│   └── cc_data_card
├── industrial/         # 工业组件
│   ├── cc_motor
│   ├── cc_pump
│   └── cc_valve
└── chart/              # 图表组件
    ├── cc_realtime_chart
    └── cc_gauge_chart
```

### 4.3 资源组织

```
assets/
├── images/
│   ├── bg/             # 背景图
│   ├── icons/          # 图标
│   └── logos/          # Logo
├── fonts/              # 字体
└── videos/             # 视频（建议外置）
```

## 5. 常见问题与解决方案

### 5.1 数据不更新

**检查清单**：

1. 数据点路径是否正确
2. 数据源连接状态
3. 订阅是否成功
4. 绑定表达式是否正确

**调试方法**：

```json
{
  "bindings": {
    "text": {
      "kind": "expr",
      "expr": "{{ JSON.stringify($dp['mqtt.EMQX.温度组.temperature']) }}"
    }
  }
}
```

### 5.2 动画不播放

**检查清单**：

1. 触发条件是否满足
2. condition 表达式是否正确
3. 动画配置是否完整
4. 是否被其他动画覆盖

### 5.3 权限不生效

**检查清单**：

1. 用户角色是否正确
2. allowRoles/denyRoles 配置
3. 条件表达式是否正确
4. 父组件是否有权限限制

### 5.4 内存泄漏

**预防措施**：

1. 使用 DisposableScope 管理订阅
2. 组件卸载时清理定时器
3. 避免在循环中创建订阅
4. 使用引用计数管理共享资源

**检测方法**：

```typescript
// 开发模式下启用内存监控
runtimeConfig.debug.memoryMonitor = true;
```

## 6. 版本迁移指南

### 6.1 迁移脚本模板

```typescript
// migrations/1.0.0_to_2.0.0.ts
export function migrate(oldSchema: SchemaV1): SchemaV2 {
  const newSchema: SchemaV2 = {
    schemaVersion: 2,
    project: oldSchema.project,
    // ...基本字段复制
  };

  // 迁移 pages
  newSchema.pagesById = {};
  for (const page of oldSchema.pages) {
    newSchema.pagesById[page.id] = migratePage(page);
  }

  // 迁移 nodes
  newSchema.nodesById = {};
  for (const comp of oldSchema.components) {
    newSchema.nodesById[comp.id] = migrateComponent(comp);
  }

  return newSchema;
}
```

### 6.2 兼容性检查

```typescript
function validateMigration(oldSchema: any, newSchema: any): ValidationResult {
  const issues: Issue[] = [];

  // 检查页面数量
  if (Object.keys(newSchema.pagesById).length !== oldSchema.pages.length) {
    issues.push({ type: "warning", message: "页面数量不匹配" });
  }

  // 检查数据绑定完整性
  // ...

  return { valid: issues.length === 0, issues };
}
```

## 7. 调试技巧

### 7.1 开发模式配置

```json
{
  "runtimeConfig": {
    "debug": {
      "enabled": true,
      "logLevel": "debug",
      "showBindingValues": true,
      "showAnimationState": true,
      "highlightUpdates": true
    }
  }
}
```

### 7.2 控制台命令

```javascript
// 查看当前数据点值
__designer__.dp.getAll();

// 查看变量状态
__designer__.vars.getAll();

// 手动触发数据更新
__designer__.dp.setValue("mqtt.EMQX.温度组.temperature", 25.5);

// 查看组件状态
__designer__.nodes.get("node_temp_display");
```

### 7.3 性能分析

```javascript
// 启用性能追踪
__designer__.performance.startTrace();

// 导出性能报告
__designer__.performance.exportReport();
```

---

**相关文档**：

- [Schema 设计](./schema-design.md)
- [动作系统](./action-system.md)
- [动画系统](./animation-system.md)
- [表达式引擎](./expression-engine.md)
- [验证系统](./validation-system.md)
