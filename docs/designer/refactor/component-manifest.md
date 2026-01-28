# 组件清单（Component Manifest）

Component Manifest 是组件的元数据描述，用于自动生成属性面板、校验 props、拖拽预览等。

## 1. 概述

### 1.1 作用

| 功能             | 说明                       |
| ---------------- | -------------------------- |
| 属性面板自动生成 | 根据 manifest 生成表单控件 |
| 类型校验         | 校验 props 类型和约束      |
| 拖拽预览         | 显示组件缩略图和名称       |
| 布局约束         | 定义组件可放置的容器类型   |
| 事件声明         | 列出组件支持的事件         |
| 文档生成         | 自动生成组件 API 文档      |

### 1.2 设计原则

- **声明式**：用 JSON 描述，无需编写代码
- **类型安全**：基于 JSON Schema 的类型系统
- **可扩展**：支持自定义编辑器和校验器
- **国际化**：标签支持 i18n

## 2. Manifest 结构

### 2.1 完整示例

```json
{
  "name": "Button",
  "displayName": "按钮",
  "category": "基础组件",
  "icon": "mdi:button-cursor",
  "description": "用于触发操作的按钮组件",

  "props": {
    "text": {
      "type": "string",
      "displayName": "按钮文本",
      "default": "按钮",
      "bindable": true,
      "i18n": true
    },
    "type": {
      "type": "string",
      "displayName": "按钮类型",
      "default": "primary",
      "enum": ["primary", "success", "warning", "danger", "info", "text"],
      "enumLabels": ["主要", "成功", "警告", "危险", "信息", "文本"]
    },
    "size": {
      "type": "string",
      "displayName": "尺寸",
      "default": "default",
      "enum": ["large", "default", "small"]
    },
    "disabled": {
      "type": "boolean",
      "displayName": "禁用",
      "default": false,
      "bindable": true
    },
    "loading": {
      "type": "boolean",
      "displayName": "加载中",
      "default": false,
      "bindable": true
    },
    "icon": {
      "type": "string",
      "displayName": "图标",
      "editor": "icon-picker"
    }
  },

  "events": {
    "onClick": {
      "displayName": "点击时",
      "description": "按钮被点击时触发",
      "payload": {}
    }
  },

  "styles": {
    "width": { "default": "auto" },
    "height": { "default": "auto" },
    "margin": { "default": "0" },
    "padding": { "default": "0" }
  },

  "layout": {
    "allowedParents": ["FlexContainer", "FreeContainer", "GridContainer"],
    "isContainer": false
  },

  "preview": {
    "width": 80,
    "height": 32
  }
}
```

### 2.2 字段说明

#### 基础信息

| 字段        | 类型   | 必填 | 说明            |
| ----------- | ------ | ---- | --------------- |
| name        | string | ✅   | 组件唯一标识    |
| displayName | string | ✅   | 显示名称        |
| category    | string | ✅   | 分类            |
| icon        | string | -    | 图标（iconify） |
| description | string | -    | 描述            |
| version     | string | -    | 版本号          |

#### Props 定义

```typescript
interface PropDefinition {
  type: PropType;
  displayName: string;
  description?: string;
  default?: any;

  // 约束
  required?: boolean;
  enum?: any[];
  enumLabels?: string[];
  min?: number;
  max?: number;
  minLength?: number;
  maxLength?: number;
  pattern?: string;

  // 绑定支持
  bindable?: boolean;
  i18n?: boolean;

  // 编辑器
  editor?: string;
  editorProps?: Record<string, any>;

  // 分组
  group?: string;

  // 条件显示
  visibleWhen?: string;
}

type PropType =
  | "string"
  | "number"
  | "boolean"
  | "array"
  | "object"
  | "color"
  | "date"
  | "icon"
  | "asset";
```

#### Events 定义

```typescript
interface EventDefinition {
  displayName: string;
  description?: string;
  payload?: Record<string, PayloadField>;
}

interface PayloadField {
  type: string;
  description?: string;
}
```

#### Layout 定义

```typescript
interface LayoutConfig {
  // 允许放入的父容器类型
  allowedParents?: string[];

  // 是否为容器组件
  isContainer?: boolean;

  // 容器配置（仅 isContainer=true）
  container?: {
    acceptsChildren?: string[]; // 允许的子组件类型
    maxChildren?: number; // 最大子组件数
    layoutType?: "flex" | "free" | "grid";
  };
}
```

## 3. 属性编辑器

### 3.1 内置编辑器

| 编辑器       | 适用类型     | 说明         |
| ------------ | ------------ | ------------ |
| text-input   | string       | 单行文本输入 |
| textarea     | string       | 多行文本输入 |
| number-input | number       | 数字输入     |
| slider       | number       | 滑块         |
| switch       | boolean      | 开关         |
| checkbox     | boolean      | 复选框       |
| select       | string       | 下拉选择     |
| radio        | string       | 单选组       |
| color-picker | color        | 颜色选择器   |
| icon-picker  | icon         | 图标选择器   |
| asset-picker | asset        | 资源选择器   |
| date-picker  | date         | 日期选择器   |
| json-editor  | object/array | JSON 编辑器  |
| expression   | any          | 表达式编辑器 |
| binding      | any          | 绑定配置器   |

### 3.2 自定义编辑器

```typescript
// 注册自定义编辑器
registerEditor('my-editor', {
  component: MyEditorComponent,
  // 值转换
  serialize: (value) => JSON.stringify(value),
  deserialize: (raw) => JSON.parse(raw),
});

// 在 manifest 中使用
{
  "props": {
    "config": {
      "type": "object",
      "editor": "my-editor",
      "editorProps": {
        "mode": "advanced"
      }
    }
  }
}
```

### 3.3 条件显示

```json
{
  "props": {
    "mode": {
      "type": "string",
      "enum": ["simple", "advanced"]
    },
    "advancedConfig": {
      "type": "object",
      "visibleWhen": "props.mode === 'advanced'"
    }
  }
}
```

## 4. 属性分组

### 4.1 分组定义

```json
{
  "propGroups": [
    { "name": "basic", "displayName": "基础", "order": 1 },
    { "name": "appearance", "displayName": "外观", "order": 2 },
    { "name": "behavior", "displayName": "行为", "order": 3 },
    { "name": "advanced", "displayName": "高级", "order": 4, "collapsed": true }
  ],

  "props": {
    "text": { "group": "basic", ... },
    "type": { "group": "appearance", ... },
    "disabled": { "group": "behavior", ... }
  }
}
```

### 4.2 属性面板渲染

```
┌─────────────────────────────────────────────────────────────────┐
│  按钮属性                                                       │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ▼ 基础                                                        │
│    按钮文本:  [按钮      ]  [🔗]                               │
│                                                                 │
│  ▼ 外观                                                        │
│    按钮类型:  [主要   ▼]                                       │
│    尺寸:      [默认   ▼]                                       │
│    图标:      [选择图标...]                                    │
│                                                                 │
│  ▼ 行为                                                        │
│    禁用:      [ ]  [🔗]                                        │
│    加载中:    [ ]  [🔗]                                        │
│                                                                 │
│  ▶ 高级 (点击展开)                                             │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

## 5. 组件注册

### 5.1 ComponentRegistry

```typescript
class ComponentRegistry {
  private components: Map<string, RegisteredComponent> = new Map();

  /**
   * 注册组件
   */
  register(manifest: ComponentManifest, component: Component): void {
    // 校验 manifest
    this.validateManifest(manifest);

    this.components.set(manifest.name, {
      manifest,
      component,
    });
  }

  /**
   * 获取组件
   */
  get(name: string): RegisteredComponent | undefined {
    return this.components.get(name);
  }

  /**
   * 获取组件列表（按分类）
   */
  getByCategory(): Map<string, RegisteredComponent[]> {
    const result = new Map<string, RegisteredComponent[]>();

    for (const comp of this.components.values()) {
      const category = comp.manifest.category;
      if (!result.has(category)) {
        result.set(category, []);
      }
      result.get(category)!.push(comp);
    }

    return result;
  }

  /**
   * 创建默认节点
   */
  createDefaultNode(name: string): ComponentNode {
    const { manifest } = this.get(name)!;

    return {
      id: generateId(),
      type: name,
      props: this.getDefaultProps(manifest.props),
      style: this.getDefaultStyles(manifest.styles),
      bindings: {},
      events: {},
      children: [],
    };
  }

  private getDefaultProps(
    propDefs: Record<string, PropDefinition>
  ): Record<string, any> {
    const props: Record<string, any> = {};
    for (const [key, def] of Object.entries(propDefs)) {
      if (def.default !== undefined) {
        props[key] = def.default;
      }
    }
    return props;
  }
}
```

### 5.2 批量注册

```typescript
// 自动扫描注册
const manifests = import.meta.glob("./components/*/manifest.json", {
  eager: true,
});
const components = import.meta.glob("./components/*/index.vue", {
  eager: true,
});

for (const [path, manifest] of Object.entries(manifests)) {
  const componentPath = path.replace("manifest.json", "index.vue");
  const component = components[componentPath];

  registry.register(manifest as ComponentManifest, component as Component);
}
```

## 6. 内置组件分类

### 6.1 基础组件

| 组件   | name    | 说明     |
| ------ | ------- | -------- |
| 按钮   | Button  | 触发操作 |
| 文本   | Text    | 显示文本 |
| 图片   | Image   | 显示图片 |
| 图标   | Icon    | 显示图标 |
| 链接   | Link    | 超链接   |
| 分隔线 | Divider | 分隔线   |

### 6.2 表单组件

| 组件     | name        | 说明       |
| -------- | ----------- | ---------- |
| 输入框   | Input       | 文本输入   |
| 数字输入 | InputNumber | 数字输入   |
| 选择器   | Select      | 下拉选择   |
| 单选组   | RadioGroup  | 单选按钮组 |
| 复选框   | Checkbox    | 复选框     |
| 开关     | Switch      | 开关切换   |
| 日期选择 | DatePicker  | 日期选择   |
| 时间选择 | TimePicker  | 时间选择   |
| 滑块     | Slider      | 滑块输入   |

### 6.3 数据展示

| 组件   | name      | 说明     |
| ------ | --------- | -------- |
| 表格   | Table     | 数据表格 |
| 列表   | List      | 列表展示 |
| 树形   | Tree      | 树形结构 |
| 进度条 | Progress  | 进度展示 |
| 数值   | Statistic | 统计数值 |
| 徽章   | Badge     | 徽章标记 |

### 6.4 布局组件

| 组件      | name          | 说明       |
| --------- | ------------- | ---------- |
| Flex 容器 | FlexContainer | 弹性布局   |
| Free 容器 | FreeContainer | 自由定位   |
| Grid 容器 | GridContainer | 网格布局   |
| 卡片      | Card          | 卡片容器   |
| 折叠面板  | Collapse      | 可折叠面板 |
| 标签页    | Tabs          | 标签页容器 |

### 6.5 反馈组件

| 组件 | name    | 说明     |
| ---- | ------- | -------- |
| 弹窗 | Dialog  | 对话框   |
| 抽屉 | Drawer  | 侧边抽屉 |
| 消息 | Message | 消息提示 |
| 通知 | Notify  | 通知提醒 |
| 加载 | Loading | 加载状态 |

### 6.6 图表组件

| 组件   | name      | 说明   |
| ------ | --------- | ------ |
| 折线图 | LineChart | 折线图 |
| 柱状图 | BarChart  | 柱状图 |
| 饼图   | PieChart  | 饼图   |
| 仪表盘 | Gauge     | 仪表盘 |
| 散点图 | Scatter   | 散点图 |

---

**相关文档**：

- [Schema 设计](./schema-design.md)
- [设计态交互](./design-interaction.md)
- [渲染架构](./rendering.md)

#### 6.6.1 ECharts 组件方法
适用：图表（EChart）组件。

- `setOption(option, notMergeOrOpts, lazyUpdate, silent, replaceMerge)`
  - `notMergeOrOpts` 支持两种写法：
    - `true/false`：`true` 表示替换旧配置（清空后重绘），`false` 表示增量更新。
    - `{ notMerge, lazyUpdate, silent, replaceMerge }`：与 ECharts 官方参数一致。
- `echarts(method, ...args)`：调用任意 ECharts 实例方法（如 `dispatchAction`）。

示例：
```js
// 替换整个 option（清空旧图）
components.图表1.setOption(option, true);

// 增量更新
components.图表1.setOption(option, false);

// 传 opts（与 ECharts 官网一致）
components.图表1.setOption(option, { notMerge: true, lazyUpdate: false });

// 调用任意 ECharts 方法
components.图表1.echarts("dispatchAction", { type: "highlight", seriesIndex: 0 });
```