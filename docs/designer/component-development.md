# 组件开发指南

> **重要**：本文档正在重构中。新的组件开发规范请参阅 [Schema 设计](./refactor/schema-design.md)。

## 概述

InduForge Designer 采用 **Component Manifest** 规范定义组件，支持自动生成属性面板、校验 props、声明事件等。

## 新版组件规范

### Component Manifest 结构

```typescript
interface ComponentManifest {
  type: string; // 组件类型标识（唯一）
  title: string; // 组件显示名称
  category: string; // 组件分类
  icon?: string; // 组件图标

  propsSchema: {
    // 属性 Schema（自动生成属性面板）
    [key: string]: {
      label: string;
      type: "string" | "number" | "boolean" | "color" | "enum" | "object";
      default?: any;
      enumValues?: string[];
      description?: string;
    };
  };

  events: Array<{
    // 事件定义
    name: string;
    title: string;
    allowedActions?: string[];
  }>;

  layoutCaps?: {
    // 布局能力
    canContain?: boolean;
    allowedParentLayouts?: string[];
  };
}
```

### 示例：按钮组件

```typescript
const ButtonManifest: ComponentManifest = {
  type: "Button",
  title: "按钮",
  category: "ui",
  icon: "button",

  propsSchema: {
    text: {
      label: "按钮文字",
      type: "string",
      default: "按钮",
    },
    type: {
      label: "类型",
      type: "enum",
      enumValues: ["default", "primary", "success", "warning", "danger"],
      default: "default",
    },
    size: {
      label: "尺寸",
      type: "enum",
      enumValues: ["small", "default", "large"],
      default: "default",
    },
    disabled: {
      label: "禁用",
      type: "boolean",
      default: false,
    },
  },

  events: [
    {
      name: "click",
      title: "点击",
      allowedActions: ["navigate", "setVar", "callApi"],
    },
    { name: "dblclick", title: "双击" },
  ],

  layoutCaps: {
    canContain: false,
    allowedParentLayouts: ["flex", "free", "grid"],
  },
};
```

### 示例：主题切换组件

```typescript
const ThemeSwitcherManifest: ComponentManifest = {
  type: "ThemeSwitcher",
  title: "主题切换",
  category: "system",

  propsSchema: {
    mode: {
      label: "切换模式",
      type: "enum",
      enumValues: ["dropdown", "toggle", "icons"],
      default: "dropdown",
    },
    showLabel: {
      label: "显示标签",
      type: "boolean",
      default: true,
    },
  },

  events: [
    {
      name: "change",
      title: "主题切换时",
      allowedActions: ["setVar", "notify"],
    },
  ],
};
```

## 数据绑定

组件属性支持多种绑定方式：

```json
{
  "bindings": {
    "text": {
      "kind": "datapoint",
      "provider": "dc_main",
      "path": "mqtt.EMQX.温度组.temperature",
      "transform": "value => value.toFixed(1) + '℃'",
      "designMock": "25.5℃"
    }
  }
}
```

详见 [数据绑定 v2](./refactor/data-binding-v2.md)。

## 国际化支持

组件属性支持 `$i18n` 语法：

```json
{
  "props": {
    "text": { "$i18n": "button.confirm" }
  }
}
```

详见 [国际化与主题](./refactor/i18n-theme.md)。

## 相关文档

- **[Schema 设计](./refactor/schema-design.md)** - 规范化工程 Schema
- **[数据绑定 v2](./refactor/data-binding-v2.md)** - 三态隔离、Binding 结构
- **[布局系统](./refactor/layout-system.md)** - Flex/Free/Grid 布局
- **[国际化与主题](./refactor/i18n-theme.md)** - i18n、主题切换

---

**版本**: 3.0.0-alpha  
**最后更新**: 2026-01
