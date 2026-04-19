# 国际化与主题系统

本文档描述 Designer 的国际化（i18n）和主题（Theme）系统设计。

## 1. 国际化系统（i18n）

### 1.1 设计目标

- 用户可在设计器中配置页面的多语言资源
- 支持静态文本、组件属性、提示信息的国际化
- 运行时可动态切换语言
- 支持常用语言（中文、英文等），可扩展

### 1.2 Schema 设计

#### 工程级国际化配置

```json
{
  "i18n": {
    "defaultLocale": "zh-CN",
    "supportedLocales": ["zh-CN", "en-US", "ja-JP"],
    "fallbackLocale": "zh-CN",

    "resources": {
      "zh-CN": {
        "common.confirm": "确认",
        "common.cancel": "取消",
        "common.save": "保存",
        "common.delete": "删除",
        "page.home.title": "首页",
        "page.home.welcome": "欢迎使用系统",
        "component.btn_stop.text": "急停",
        "component.btn_stop.confirmTitle": "确认执行急停？"
      },
      "en-US": {
        "common.confirm": "Confirm",
        "common.cancel": "Cancel",
        "common.save": "Save",
        "common.delete": "Delete",
        "page.home.title": "Home",
        "page.home.welcome": "Welcome to the system",
        "component.btn_stop.text": "E-Stop",
        "component.btn_stop.confirmTitle": "Confirm emergency stop?"
      }
    }
  }
}
```

#### 页面级国际化资源（可覆盖/扩展工程级）

```json
{
  "pagesById": {
    "page_device_list": {
      "id": "page_device_list",
      "name": "设备列表",
      "i18n": {
        "zh-CN": {
          "title": "设备列表",
          "filter.all": "全部",
          "filter.online": "在线",
          "filter.offline": "离线"
        },
        "en-US": {
          "title": "Device List",
          "filter.all": "All",
          "filter.online": "Online",
          "filter.offline": "Offline"
        }
      }
    }
  }
}
```

### 1.3 组件属性国际化

#### 方式一：直接使用 i18n key

```json
{
  "id": "node_title",
  "type": "Text",
  "props": {
    "text": { "$i18n": "page.home.welcome" }
  }
}
```

#### 方式二：内联多语言对象

```json
{
  "id": "node_title",
  "type": "Text",
  "props": {
    "text": {
      "$i18n": {
        "zh-CN": "欢迎使用",
        "en-US": "Welcome"
      }
    }
  }
}
```

### 1.4 Designer UI - 国际化配置面板

```
┌─ 国际化配置 ────────────────────────────────────────────────────────────┐
│                                                                          │
│  支持语言: [+ 添加语言]                                                  │
│  ┌──────────────────────────────────────────────────────────────────┐   │
│  │ ☑ 简体中文 (zh-CN) [默认]    ☑ English (en-US)    ☐ 日本語      │   │
│  └──────────────────────────────────────────────────────────────────┘   │
│                                                                          │
│  资源管理:                                                               │
│  ┌──────────────────────────────────────────────────────────────────┐   │
│  │ Key                    │ 简体中文        │ English                │   │
│  ├────────────────────────┼─────────────────┼────────────────────────┤   │
│  │ common.confirm         │ 确认            │ Confirm                │   │
│  │ common.cancel          │ 取消            │ Cancel                 │   │
│  │ page.home.title        │ 首页            │ Home                   │   │
│  │ page.home.welcome      │ 欢迎使用系统    │ Welcome to the system  │   │
│  └──────────────────────────────────────────────────────────────────┘   │
│                                                                          │
│  [导入 JSON]  [导出 JSON]  [批量翻译]                                   │
│                                                                          │
└──────────────────────────────────────────────────────────────────────────┘
```

### 1.5 属性面板集成

在组件属性面板中，文本类属性旁显示国际化按钮：

```
┌─ 属性配置 ──────────────────────────────────────────┐
│                                                      │
│  文本: [ 欢迎使用系统 ] [🌐]                        │
│                          ↑                          │
│                    点击配置多语言                    │
│                                                      │
│  ┌─ 多语言配置 ─────────────────────────┐           │
│  │                                       │           │
│  │  ○ 使用 i18n key: [ page.home.welcome ▼ ]        │
│  │  ● 内联配置:                          │           │
│  │    简体中文: [ 欢迎使用系统 ]        │           │
│  │    English:  [ Welcome to the system ]│           │
│  │                                       │           │
│  │               [确定]  [取消]          │           │
│  └───────────────────────────────────────┘           │
└──────────────────────────────────────────────────────┘
```

---

## 2. 主题系统（Theme）

### 2.1 设计目标

- 支持浅色/深色/自定义主题
- 用户可在设计器中定义主题变量
- 提供主题切换组件，运行时可切换
- 组件样式自动响应主题变化

### 2.2 Schema 设计

#### 工程级主题配置

```json
{
  "themes": {
    "defaultTheme": "light",
    "supportedThemes": ["light", "dark", "industrial"],

    "definitions": {
      "light": {
        "name": "浅色主题",
        "variables": {
          "--color-primary": "#409EFF",
          "--color-success": "#67C23A",
          "--color-warning": "#E6A23C",
          "--color-danger": "#F56C6C",
          "--color-info": "#909399",

          "--bg-color": "#FFFFFF",
          "--bg-color-page": "#F5F7FA",
          "--bg-color-overlay": "#FFFFFF",

          "--text-color-primary": "#303133",
          "--text-color-regular": "#606266",
          "--text-color-secondary": "#909399",
          "--text-color-placeholder": "#C0C4CC",

          "--border-color": "#DCDFE6",
          "--border-color-light": "#E4E7ED",

          "--shadow-base": "0 2px 4px rgba(0, 0, 0, 0.12)",
          "--shadow-light": "0 2px 12px rgba(0, 0, 0, 0.1)"
        }
      },
      "dark": {
        "name": "深色主题",
        "variables": {
          "--color-primary": "#409EFF",
          "--color-success": "#67C23A",
          "--color-warning": "#E6A23C",
          "--color-danger": "#F56C6C",
          "--color-info": "#909399",

          "--bg-color": "#141414",
          "--bg-color-page": "#0A0A0A",
          "--bg-color-overlay": "#1D1D1D",

          "--text-color-primary": "#E5EAF3",
          "--text-color-regular": "#CFD3DC",
          "--text-color-secondary": "#A3A6AD",
          "--text-color-placeholder": "#8D9095",

          "--border-color": "#4C4D4F",
          "--border-color-light": "#414243",

          "--shadow-base": "0 2px 4px rgba(0, 0, 0, 0.5)",
          "--shadow-light": "0 2px 12px rgba(0, 0, 0, 0.4)"
        }
      },
      "industrial": {
        "name": "工业风格",
        "variables": {
          "--color-primary": "#00D4AA",
          "--color-success": "#52C41A",
          "--color-warning": "#FAAD14",
          "--color-danger": "#FF4D4F",

          "--bg-color": "#0D1117",
          "--bg-color-page": "#010409",
          "--bg-color-overlay": "#161B22",

          "--text-color-primary": "#C9D1D9",
          "--text-color-regular": "#8B949E"
        }
      }
    }
  }
}
```

### 2.3 组件样式引用主题变量

```json
{
  "id": "node_card",
  "type": "Card",
  "style": {
    "backgroundColor": "var(--bg-color-overlay)",
    "color": "var(--text-color-primary)",
    "borderColor": "var(--border-color)",
    "boxShadow": "var(--shadow-base)"
  }
}
```

### 2.4 主题切换组件（ThemeSwitcher）

#### 组件 Manifest

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
    size: {
      label: "尺寸",
      type: "enum",
      enumValues: ["small", "default", "large"],
      default: "default",
    },
  },

  events: [
    {
      name: "change",
      title: "主题切换时",
      allowedActions: ["setVar", "notify"],
    },
  ],

  layoutCaps: {
    allowedParentLayouts: ["free", "flex", "grid"],
  },
};
```

#### 组件渲染

```vue
<template>
  <!-- 下拉模式 -->
  <el-dropdown v-if="props.mode === 'dropdown'" @command="handleThemeChange">
    <span class="theme-trigger">
      <i :class="currentThemeIcon" />
      <span v-if="props.showLabel">{{ currentThemeName }}</span>
    </span>
    <template #dropdown>
      <el-dropdown-menu>
        <el-dropdown-item
          v-for="theme in availableThemes"
          :key="theme.id"
          :command="theme.id"
          :class="{ active: theme.id === currentTheme }"
        >
          <i :class="theme.icon" />
          {{ theme.name }}
        </el-dropdown-item>
      </el-dropdown-menu>
    </template>
  </el-dropdown>

  <!-- 切换模式（仅 light/dark） -->
  <el-switch
    v-else-if="props.mode === 'toggle'"
    :model-value="currentTheme === 'dark'"
    @change="handleToggle"
    :active-icon="Moon"
    :inactive-icon="Sunny"
  />

  <!-- 图标模式 -->
  <div v-else class="theme-icons">
    <button
      v-for="theme in availableThemes"
      :key="theme.id"
      :class="{ active: theme.id === currentTheme }"
      @click="handleThemeChange(theme.id)"
    >
      <i :class="theme.icon" />
    </button>
  </div>
</template>

<script setup>
import { useTheme } from "@/composables/useTheme";

const { currentTheme, availableThemes, setTheme } = useTheme();

function handleThemeChange(themeId) {
  setTheme(themeId);
  emit("change", themeId);
}

function handleToggle(isDark) {
  setTheme(isDark ? "dark" : "light");
}
</script>
```

### 2.5 Designer UI - 主题配置面板

```
┌─ 主题配置 ────────────────────────────────────────────────────────────┐
│                                                                        │
│  默认主题: [ 浅色主题 ▼ ]                                             │
│                                                                        │
│  主题列表:                                                             │
│  ┌────────────────────────────────────────────────────────────────┐   │
│  │ ☑ 浅色主题 (light) [默认]  [编辑]                              │   │
│  │ ☑ 深色主题 (dark)          [编辑]                              │   │
│  │ ☑ 工业风格 (industrial)    [编辑]                              │   │
│  │                                                                 │   │
│  │ [+ 新建主题]                                                   │   │
│  └────────────────────────────────────────────────────────────────┘   │
│                                                                        │
│  ─────────────────────────────────────────────────────────────────────│
│                                                                        │
│  编辑主题: 浅色主题                                                    │
│  ┌────────────────────────────────────────────────────────────────┐   │
│  │ 颜色变量:                                                       │   │
│  │ --color-primary     [#409EFF] [🎨]                             │   │
│  │ --color-success     [#67C23A] [🎨]                             │   │
│  │ --color-warning     [#E6A23C] [🎨]                             │   │
│  │ --color-danger      [#F56C6C] [🎨]                             │   │
│  │                                                                 │   │
│  │ 背景变量:                                                       │   │
│  │ --bg-color          [#FFFFFF] [🎨]                             │   │
│  │ --bg-color-page     [#F5F7FA] [🎨]                             │   │
│  │                                                                 │   │
│  │ 文字变量:                                                       │   │
│  │ --text-color-primary [#303133] [🎨]                            │   │
│  └────────────────────────────────────────────────────────────────┘   │
│                                                                        │
│  预览: [浅色] [深色] [工业风格]                                       │
│                                                                        │
└────────────────────────────────────────────────────────────────────────┘
```

---

## 3. 运行时实现

### 3.1 i18n Service

```typescript
class I18nService {
  private currentLocale: Ref<string>;
  private resources: Map<string, Record<string, string>>;

  constructor(config: I18nConfig) {
    this.currentLocale = ref(config.defaultLocale);
    this.resources = new Map();

    // 加载资源
    for (const [locale, res] of Object.entries(config.resources)) {
      this.resources.set(locale, res);
    }
  }

  // 获取翻译
  t(key: string, params?: Record<string, any>): string {
    const resource = this.resources.get(this.currentLocale.value);
    let text = resource?.[key] ?? key;

    // 参数替换 {name} -> value
    if (params) {
      for (const [k, v] of Object.entries(params)) {
        text = text.replace(new RegExp(`\\{${k}\\}`, "g"), String(v));
      }
    }

    return text;
  }

  // 切换语言
  setLocale(locale: string): void {
    if (this.resources.has(locale)) {
      this.currentLocale.value = locale;
      document.documentElement.lang = locale;
      // 持久化
      localStorage.setItem("locale", locale);
    }
  }

  // 解析 $i18n 属性
  resolveI18nProp(prop: any): string {
    if (!prop || typeof prop !== "object" || !("$i18n" in prop)) {
      return prop;
    }

    const i18nValue = prop.$i18n;

    // key 引用
    if (typeof i18nValue === "string") {
      return this.t(i18nValue);
    }

    // 内联对象
    if (typeof i18nValue === "object") {
      return (
        i18nValue[this.currentLocale.value] ??
        i18nValue[this.fallbackLocale] ??
        Object.values(i18nValue)[0]
      );
    }

    return prop;
  }
}
```

### 3.2 Theme Service

```typescript
class ThemeService {
  private currentTheme: Ref<string>;
  private themes: Map<string, ThemeDefinition>;

  constructor(config: ThemeConfig) {
    this.currentTheme = ref(config.defaultTheme);
    this.themes = new Map();

    for (const [id, def] of Object.entries(config.definitions)) {
      this.themes.set(id, def);
    }

    // 应用初始主题
    this.applyTheme(this.currentTheme.value);
  }

  // 切换主题
  setTheme(themeId: string): void {
    if (!this.themes.has(themeId)) return;

    this.currentTheme.value = themeId;
    this.applyTheme(themeId);

    // 持久化
    localStorage.setItem("theme", themeId);
  }

  // 应用主题（设置 CSS 变量）
  private applyTheme(themeId: string): void {
    const theme = this.themes.get(themeId);
    if (!theme) return;

    const root = document.documentElement;

    for (const [key, value] of Object.entries(theme.variables)) {
      root.style.setProperty(key, value);
    }

    // 设置 data-theme 属性（供 CSS 选择器使用）
    root.setAttribute("data-theme", themeId);
  }

  // 获取当前主题
  get theme(): string {
    return this.currentTheme.value;
  }

  // 获取可用主题列表
  get availableThemes(): Array<{ id: string; name: string }> {
    return Array.from(this.themes.entries()).map(([id, def]) => ({
      id,
      name: def.name,
    }));
  }
}
```

### 3.3 语言切换组件（LocaleSwitcher）

```typescript
const LocaleSwitcherManifest: ComponentManifest = {
  type: "LocaleSwitcher",
  title: "语言切换",
  category: "system",

  propsSchema: {
    mode: {
      label: "显示模式",
      type: "enum",
      enumValues: ["dropdown", "flags", "text"],
      default: "dropdown",
    },
    showNativeName: {
      label: "显示本地名称",
      type: "boolean",
      default: true,
    },
  },

  events: [
    {
      name: "change",
      title: "语言切换时",
      allowedActions: ["setVar", "notify"],
    },
  ],
};
```

---

## 4. IFP 打包

国际化和主题配置作为工程 Schema 的一部分打包进 IFP：

```
project-v1.0.0.ifp
├── manifest.json
├── project.json          # 包含 i18n 和 themes 配置
│   ├── i18n: {...}
│   └── themes: {...}
├── datacenter.json
└── assets/
```

---

## 5. 实现步骤

### 国际化

1. **Schema 扩展** - 添加 i18n 配置结构
2. **I18nService** - 实现翻译、切换、资源管理
3. **Designer UI** - 国际化配置面板
4. **属性面板集成** - 文本属性的多语言配置
5. **LocaleSwitcher 组件** - 语言切换组件
6. **RuntimeRenderer 集成** - 解析 $i18n 属性

### 主题

1. **Schema 扩展** - 添加 themes 配置结构
2. **ThemeService** - 实现主题切换、CSS 变量应用
3. **Designer UI** - 主题配置面板
4. **预置主题** - light/dark/industrial
5. **ThemeSwitcher 组件** - 主题切换组件
6. **组件库适配** - 确保组件样式使用主题变量

---

## 6. 测试要点

### 国际化

- [ ] i18n key 引用正确解析
- [ ] 内联多语言对象正确解析
- [ ] 语言切换后 UI 更新
- [ ] 参数替换正确
- [ ] fallback 语言生效
- [ ] 页面级资源覆盖工程级

### 主题

- [ ] 主题切换后 CSS 变量更新
- [ ] 组件样式响应主题变化
- [ ] 自定义主题保存和加载
- [ ] 主题持久化（localStorage）
- [ ] data-theme 属性正确设置

---

**相关文档**：

- [Schema 设计](./schema-design.md)
- [运行时引擎](./runtime-engine.md)
- [组件开发指南](../component-development.md)
