# 变量系统（Vars System）

变量系统用于管理页面内部状态和跨页面共享数据，与数据点（DataPoint）形成互补。

## 1. 概述

### 1.1 变量 vs 数据点

| 特性     | 数据点（DataPoint）    | 变量（Vars）                |
| -------- | ---------------------- | --------------------------- |
| 来源     | 数据中心（外部数据源） | 页面/工程内部定义           |
| 生命周期 | 跨页面持久化           | 页面级或会话级              |
| 典型用途 | 设备数据、查询结果     | 筛选条件、表单状态、UI 状态 |
| 可写性   | 只读（除 writeTag）    | 可读写                      |
| 实时性   | 实时订阅               | 本地状态                    |

### 1.2 变量分类

| 类型     | 作用域   | 生命周期              | 典型用途               |
| -------- | -------- | --------------------- | ---------------------- |
| 页面变量 | 当前页面 | 页面 mount → unmount  | 表单状态、临时筛选条件 |
| 全局变量 | 整个工程 | 应用启动 → 关闭       | 用户偏好、全局筛选条件 |
| 会话变量 | 整个工程 | 登录 → 登出（持久化） | 用户设置、购物车       |

## 2. Schema 设计

### 2.1 工程级变量声明

```json
{
  "vars": {
    "global": {
      "currentShift": {
        "type": "string",
        "default": "day",
        "enum": ["day", "night"],
        "label": "当前班次",
        "persistent": false
      },
      "selectedLine": {
        "type": "string",
        "default": "",
        "label": "选中产线"
      },
      "userPreferences": {
        "type": "object",
        "default": {},
        "persistent": true,
        "storageKey": "user_prefs"
      }
    }
  }
}
```

### 2.2 页面级变量声明

```json
{
  "pagesById": {
    "page_home": {
      "vars": {
        "searchKeyword": {
          "type": "string",
          "default": ""
        },
        "filterStatus": {
          "type": "string",
          "default": "all",
          "enum": ["all", "running", "stopped", "error"]
        },
        "selectedRows": {
          "type": "array",
          "default": [],
          "items": { "type": "string" }
        },
        "dialogVisible": {
          "type": "boolean",
          "default": false
        }
      }
    }
  }
}
```

### 2.3 变量类型定义

```typescript
interface VarDefinition {
  type: "string" | "number" | "boolean" | "array" | "object";
  default: any; // 默认值（必填）
  label?: string; // 显示名称
  description?: string; // 描述

  // 约束
  enum?: any[]; // 枚举值
  min?: number; // 最小值（number）
  max?: number; // 最大值（number）
  minLength?: number; // 最小长度（string/array）
  maxLength?: number; // 最大长度（string/array）
  items?: { type: string }; // 数组元素类型

  // 持久化
  persistent?: boolean; // 是否持久化到 localStorage
  storageKey?: string; // localStorage key（默认 var_{name}）
}
```

## 3. 绑定语法

### 3.1 在 Binding 中引用变量

```json
{
  "bindings": {
    "visible": {
      "kind": "var",
      "scope": "page",
      "name": "dialogVisible"
    },
    "value": {
      "kind": "var",
      "scope": "global",
      "name": "selectedLine"
    }
  }
}
```

### 3.2 在表达式中引用变量

```json
{
  "bindings": {
    "text": {
      "kind": "expr",
      "expr": "{{ $vars.searchKeyword ? '搜索: ' + $vars.searchKeyword : '请输入关键词' }}"
    },
    "disabled": {
      "kind": "expr",
      "expr": "{{ $vars.selectedRows.length === 0 }}"
    }
  }
}
```

### 3.3 上下文变量

| 变量           | 说明             | 示例                   |
| -------------- | ---------------- | ---------------------- |
| `$vars`        | 当前页面变量     | `$vars.searchKeyword`  |
| `$global`      | 全局变量         | `$global.currentShift` |
| `$vars.{name}` | 直接访问页面变量 | `$vars.filterStatus`   |

## 4. 动作中操作变量

### 4.1 setVar 动作

```json
{
  "events": {
    "onClick": [
      {
        "type": "setVar",
        "scope": "page",
        "name": "dialogVisible",
        "value": true
      }
    ]
  }
}
```

### 4.2 动态值设置

```json
{
  "type": "setVar",
  "scope": "page",
  "name": "selectedRows",
  "value": {
    "kind": "expr",
    "expr": "{{ [...$vars.selectedRows, $event.row.id] }}"
  }
}
```

### 4.3 批量设置

```json
{
  "type": "setVars",
  "vars": [
    { "scope": "page", "name": "searchKeyword", "value": "" },
    { "scope": "page", "name": "filterStatus", "value": "all" },
    { "scope": "page", "name": "selectedRows", "value": [] }
  ]
}
```

### 4.4 重置变量

```json
{
  "type": "resetVar",
  "scope": "page",
  "name": "searchKeyword"
}

// 或重置所有页面变量
{
  "type": "resetVars",
  "scope": "page"
}
```

## 5. 运行时实现

### 5.1 VarsStore 设计

```typescript
class VarsStore {
  private globalVars: Map<string, any> = new Map();
  private pageVars: Map<string, Map<string, any>> = new Map();
  private definitions: VarsDefinitions;

  constructor(definitions: VarsDefinitions) {
    this.definitions = definitions;
    this.initGlobalVars();
  }

  // 初始化全局变量
  private initGlobalVars(): void {
    for (const [name, def] of Object.entries(this.definitions.global)) {
      let value = def.default;

      // 从 localStorage 恢复持久化变量
      if (def.persistent) {
        const stored = localStorage.getItem(def.storageKey || `var_${name}`);
        if (stored !== null) {
          value = JSON.parse(stored);
        }
      }

      this.globalVars.set(name, value);
    }
  }

  // 初始化页面变量
  initPageVars(
    pageId: string,
    definitions: Record<string, VarDefinition>
  ): void {
    const vars = new Map<string, any>();
    for (const [name, def] of Object.entries(definitions)) {
      vars.set(name, def.default);
    }
    this.pageVars.set(pageId, vars);
  }

  // 清理页面变量
  clearPageVars(pageId: string): void {
    this.pageVars.delete(pageId);
  }

  // 获取变量
  get(scope: "global" | "page", name: string, pageId?: string): any {
    if (scope === "global") {
      return this.globalVars.get(name);
    } else {
      return this.pageVars.get(pageId!)?.get(name);
    }
  }

  // 设置变量
  set(
    scope: "global" | "page",
    name: string,
    value: any,
    pageId?: string
  ): void {
    if (scope === "global") {
      this.globalVars.set(name, value);

      // 持久化
      const def = this.definitions.global[name];
      if (def?.persistent) {
        localStorage.setItem(
          def.storageKey || `var_${name}`,
          JSON.stringify(value)
        );
      }
    } else {
      this.pageVars.get(pageId!)?.set(name, value);
    }

    // 触发变更事件
    this.emit("change", { scope, name, value, pageId });
  }

  // 重置变量
  reset(scope: "global" | "page", name: string, pageId?: string): void {
    const def =
      scope === "global"
        ? this.definitions.global[name]
        : this.definitions.pages[pageId!]?.[name];

    if (def) {
      this.set(scope, name, def.default, pageId);
    }
  }

  // 获取上下文（用于表达式求值）
  getContext(pageId: string): VarsContext {
    const pageVars = this.pageVars.get(pageId) || new Map();
    return {
      $vars: Object.fromEntries(pageVars),
      $global: Object.fromEntries(this.globalVars),
    };
  }
}
```

### 5.2 与组件集成

```typescript
// Vue 组合式 API
function useVar(name: string, scope: "page" | "global" = "page") {
  const varsStore = inject("varsStore");
  const pageId = inject("currentPageId");

  const value = computed({
    get: () => varsStore.get(scope, name, pageId),
    set: (val) => varsStore.set(scope, name, val, pageId),
  });

  return value;
}

// 使用示例
const searchKeyword = useVar("searchKeyword");
const currentShift = useVar("currentShift", "global");
```

## 6. Designer 支持

### 6.1 变量管理面板

```
┌─────────────────────────────────────────────────────────────────┐
│  变量管理                                           [+ 添加]    │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  📁 全局变量                                                    │
│  ├─ currentShift (string) = "day"                              │
│  ├─ selectedLine (string) = ""                                 │
│  └─ userPreferences (object) 🔒 持久化                         │
│                                                                 │
│  📁 页面变量 - 首页                                             │
│  ├─ searchKeyword (string) = ""                                │
│  ├─ filterStatus (string) = "all"                              │
│  ├─ selectedRows (array) = []                                  │
│  └─ dialogVisible (boolean) = false                            │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### 6.4 工程变量管理（设计器实现）

工程变量在设计器内以树形结构展示，支持分组与批量操作：

- 分组层级：支持 1-5 层嵌套分组。
- 右键菜单：新建分组/新增变量/快速添加/复制/粘贴/移动/删除。
- 双击与拖拽：双击关闭菜单；拖拽变量/分组可调整层级。
- 多选：`Ctrl/⌘ + 左键` 多选，仅允许移动/复制/删除。
- 映射区分：映射变量与非映射变量使用不同图标区分。

### 6.5 快速添加数据中心变量

“快速添加”基于数据中心的数据源与字段批量创建工程变量：

- 选择数据源（连接）
- 支持字段搜索、前缀/后缀、批量替换
- 变量自动写入映射信息（`source.type = "dataCenter"`）

### 6.6 导入导出

工程变量支持导入/导出，便于批量维护：

- 导出：CSV / XLSX / JSON
- 导入：CSV / XLSX / JSON
- JSON 支持导入 `{ definitions, groups }` 的完整结构

### 6.7 数据持久化

工程变量保存到 `design_project_settings.globalVariables`，并同步写入
`projects.variables` 以兼容旧接口。

### 6.2 变量选择器

在绑定配置中提供变量选择器：

```
┌─────────────────────────────────────────────────────────────────┐
│  绑定类型: [变量 ▼]                                             │
│                                                                 │
│  作用域: ○ 页面变量  ● 全局变量                                 │
│                                                                 │
│  选择变量:                                                      │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │ 🔍 搜索变量...                                          │   │
│  ├─────────────────────────────────────────────────────────┤   │
│  │ ○ currentShift (string) - 当前班次                      │   │
│  │ ● selectedLine (string) - 选中产线                      │   │
│  │ ○ userPreferences (object) - 用户偏好                   │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### 6.3 设计态预览

设计态可以模拟变量值进行预览：

```
┌─────────────────────────────────────────────────────────────────┐
│  变量调试                                           [重置全部]  │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  currentShift:    [day     ▼]                                  │
│  selectedLine:    [Line_A    ]                                 │
│  dialogVisible:   [✓]                                          │
│  selectedRows:    [编辑 JSON...]                               │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

## 7. 典型场景

### 7.1 表单状态管理

```json
{
  "vars": {
    "formData": {
      "type": "object",
      "default": {
        "name": "",
        "email": "",
        "phone": ""
      }
    },
    "formErrors": {
      "type": "object",
      "default": {}
    },
    "submitting": {
      "type": "boolean",
      "default": false
    }
  }
}
```

### 7.2 筛选条件联动

```json
{
  "vars": {
    "dateRange": { "type": "array", "default": [] },
    "department": { "type": "string", "default": "" },
    "status": { "type": "string", "default": "all" }
  }
}
```

查询时使用：

```json
{
  "type": "callApi",
  "api": "getRecords",
  "params": {
    "startDate": "{{ $vars.dateRange[0] }}",
    "endDate": "{{ $vars.dateRange[1] }}",
    "department": "{{ $vars.department }}",
    "status": "{{ $vars.status }}"
  }
}
```

### 7.3 多步骤向导

```json
{
  "vars": {
    "currentStep": { "type": "number", "default": 1 },
    "step1Data": { "type": "object", "default": {} },
    "step2Data": { "type": "object", "default": {} },
    "step3Data": { "type": "object", "default": {} }
  }
}
```

---

**相关文档**：

- [数据绑定 v2](./data-binding-v2.md)
- [表达式引擎](./expression-engine.md)
- [动作系统](./action-system.md)
- [Schema 设计](./schema-design.md)
