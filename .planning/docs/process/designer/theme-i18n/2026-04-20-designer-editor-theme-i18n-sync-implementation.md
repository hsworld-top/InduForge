# 设计中心编辑器主题与国际化同步 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 让 `dev_ide` 与 `designer` 在编辑器层完成主题和语言同步，并让 `designer` 通过 `vue-i18n` 管理编辑器文案，同时保证画布内用户页面不受影响。

**Architecture:** 在 `designer` 内新增一层专用于编辑器 UI 的状态与国际化入口，统一管理 `theme`、`locale`、DOM 主题应用、Element Plus locale 和 `vue-i18n` locale。`dev_ide` 继续作为宿主，负责 URL 初始传参与运行中 `postMessage` 广播；`designer` 负责消费宿主消息并保留本地独立切换能力。

**Tech Stack:** Vue 3、TypeScript、Pinia、Element Plus、`vue-i18n`、Vitest、Node `assert`

---

## 文件结构与职责

### `designer`

- Modify: `designer/package.json`
  - 新增 `vue-i18n` 依赖，保持与 `dev_ide` 一致的国际化实现路线。
- Create: `designer/src/i18n/index.ts`
  - 创建 `vue-i18n` 实例，导出 `messages`、`i18n`、locale 解析工具。
- Create: `designer/src/i18n/messages/zh.ts`
  - 中文编辑器文案字典，首轮覆盖壳层与高频面板文案。
- Create: `designer/src/i18n/messages/en.ts`
  - 英文编辑器文案字典，与中文键集保持一致。
- Create: `designer/src/stores/editor-ui-store.ts`
  - 编辑器 UI 状态单一入口，负责 `theme`、`locale`、本地缓存、DOM 应用。
- Create: `designer/src/stores/editor-ui-store.test.ts`
  - 验证 URL/本地缓存回退、主题应用、语言应用、非法输入降级。
- Modify: `designer/src/constants/index.ts`
  - 如有必要，补充 locale 常量或消息类型常量。
- Modify: `designer/src/utils/storage.ts`
  - 补充 `getTheme/getLanguage/setTheme/setLanguage` 便捷方法。
- Modify: `designer/src/utils/storage.test.ts`
  - 为新增存储方法补测试。
- Modify: `designer/src/main.ts`
  - 注册 `vue-i18n`，统一接入主题/语言消息监听。
- Create: `designer/src/router/runtime-settings.ts`
  - 提取 URL 主题/语言解析逻辑，保证可测。
- Modify: `designer/src/router/index.ts`
  - 调用 runtime helper，减少路由文件内的副作用逻辑。
- Modify: `designer/src/App.vue`
  - 用 `ElConfigProvider` 和 `vue-i18n` locale 绑定包裹 `router-view`。
- Modify: `designer/src/ui/shell/DesignerView.vue`
  - 接入 `useI18n` 和编辑器 UI 状态，替换高频消息和主题/语言切换逻辑。
- Modify: `designer/src/ui/shell/TopToolbar/TopToolbar.vue`
  - 替换高频硬编码文案，接入 `useI18n`。
- Modify: `designer/src/ui/shell/el-message-compat.ts`
  - 若需要，补充可接收翻译后字符串的兼容封装。
- Create: `designer/src/ui/shell/TopToolbar/TopToolbar.i18n.test.ts`
  - 覆盖壳层高频文案切换与事件行为。

### `dev_ide`

- Modify: `dev_ide/src/utils/appUrl.js`
  - 生成设计中心链接时增加 `locale` 参数。
- Create: `dev_ide/src/views/dashboard/embedded-sync.js`
  - 提取 iframe 主题/语言广播逻辑，便于 Node 断言测试。
- Modify: `dev_ide/src/views/Dashboard.vue`
  - 补充 `syncEmbeddedLocale` 并与现有主题广播并行。
- Modify: `dev_ide/tests/run-tests.js`
  - 增加 appUrl 构造与嵌入消息同步的 Node 断言测试。

## Task 1: 建立 `designer` 的 `vue-i18n` 与编辑器 UI 状态基础

**Files:**
- Modify: `designer/package.json`
- Create: `designer/src/i18n/index.ts`
- Create: `designer/src/i18n/messages/zh.ts`
- Create: `designer/src/i18n/messages/en.ts`
- Create: `designer/src/stores/editor-ui-store.ts`
- Create: `designer/src/stores/editor-ui-store.test.ts`
- Modify: `designer/src/utils/storage.ts`
- Modify: `designer/src/utils/storage.test.ts`

- [ ] **Step 1: 先写 `editor-ui-store` 的失败测试**

```ts
// designer/src/stores/editor-ui-store.test.ts
import { afterEach, describe, expect, it, vi } from "vitest";
import { createEditorUiStore } from "./editor-ui-store";

const store: Record<string, string> = {};

function stubStorage() {
  vi.stubGlobal("localStorage", {
    getItem: (key: string) => store[key] ?? null,
    setItem: (key: string, value: string) => {
      store[key] = value;
    },
    removeItem: (key: string) => {
      delete store[key];
    },
    clear: () => {
      Object.keys(store).forEach((key) => delete store[key]);
    },
  });
}

describe("editor-ui-store", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
    document.documentElement.className = "";
    document.documentElement.removeAttribute("data-theme");
    Object.keys(store).forEach((key) => delete store[key]);
  });

  it("优先使用 URL 中的 theme 和 locale", () => {
    stubStorage();
    store.theme = JSON.stringify("light");
    store.language = JSON.stringify("zh");

    const ui = createEditorUiStore();
    ui.initFromRuntime({ theme: "dark", locale: "en" });

    expect(ui.theme.value).toBe("dark");
    expect(ui.locale.value).toBe("en");
  });

  it("非法输入会回退到本地缓存和默认值", () => {
    stubStorage();
    store.theme = JSON.stringify("dark");

    const ui = createEditorUiStore();
    ui.initFromRuntime({ theme: "solarized" as never, locale: "jp" as never });

    expect(ui.theme.value).toBe("dark");
    expect(ui.locale.value).toBe("zh");
  });

  it("切换主题时同步更新 DOM 与本地缓存", () => {
    stubStorage();

    const ui = createEditorUiStore();
    ui.initFromRuntime();
    ui.setTheme("dark");

    expect(document.documentElement.classList.contains("dark")).toBe(true);
    expect(document.documentElement.getAttribute("data-theme")).toBe("dark");
    expect(store.theme).toBe(JSON.stringify("dark"));
  });
});
```

- [ ] **Step 2: 运行测试，确认当前失败**

Run: `pnpm --dir designer exec vitest --run src/stores/editor-ui-store.test.ts src/utils/storage.test.ts`

Expected: FAIL，提示 `Cannot find module './editor-ui-store'` 或缺少 `getTheme/getLanguage` 等方法。

- [ ] **Step 3: 最小实现 `storage` 便捷方法与 `editor-ui-store`**

```ts
// designer/src/utils/storage.ts
static getTheme(): "light" | "dark" {
  const value = this.get(STORAGE_KEYS.THEME, "light");
  return value === "dark" ? "dark" : "light";
}

static setTheme(theme: "light" | "dark"): void {
  this.set(STORAGE_KEYS.THEME, theme);
}

static getLanguage(): "zh" | "en" {
  const value = this.get(STORAGE_KEYS.LANGUAGE, "zh");
  return value === "en" ? "en" : "zh";
}

static setLanguage(language: "zh" | "en"): void {
  this.set(STORAGE_KEYS.LANGUAGE, language);
}
```

```ts
// designer/src/stores/editor-ui-store.ts
import { computed, ref } from "vue";
import en from "element-plus/es/locale/lang/en";
import zhCn from "element-plus/es/locale/lang/zh-cn";
import { STORAGE_KEYS } from "@/constants";
import { Storage } from "@/utils/storage";

export type EditorTheme = "light" | "dark";
export type EditorLocale = "zh" | "en";

const normalizeTheme = (value: unknown): EditorTheme | null =>
  value === "dark" || value === "light" ? value : null;

const normalizeLocale = (value: unknown): EditorLocale | null =>
  value === "en" || value === "zh" ? value : null;

export function createEditorUiStore() {
  const theme = ref<EditorTheme>("light");
  const locale = ref<EditorLocale>("zh");

  const applyThemeToDom = (nextTheme = theme.value) => {
    document.documentElement.classList.toggle("dark", nextTheme === "dark");
    document.documentElement.setAttribute("data-theme", nextTheme);
  };

  const initFromRuntime = (input?: Partial<{ theme: EditorTheme; locale: EditorLocale }>) => {
    theme.value = normalizeTheme(input?.theme) ?? Storage.getTheme();
    locale.value = normalizeLocale(input?.locale) ?? Storage.getLanguage();
    applyThemeToDom(theme.value);
    Storage.set(STORAGE_KEYS.THEME, theme.value);
    Storage.set(STORAGE_KEYS.LANGUAGE, locale.value);
  };

  const setTheme = (nextTheme: EditorTheme) => {
    theme.value = nextTheme;
    applyThemeToDom(nextTheme);
    Storage.setTheme(nextTheme);
  };

  const setLocale = (nextLocale: EditorLocale) => {
    locale.value = nextLocale;
    Storage.setLanguage(nextLocale);
  };

  const elementLocale = computed(() => (locale.value === "en" ? en : zhCn));

  return { theme, locale, initFromRuntime, setTheme, setLocale, applyThemeToDom, elementLocale };
}
```

- [ ] **Step 4: 建立 `vue-i18n` 入口与首轮字典**

```ts
// designer/src/i18n/messages/zh.ts
export const zhMessages = {
  toolbar: {
    preview: "预览",
    pagePreview: "页面预览",
    appPreview: "应用预览",
    save: "保存",
    saveSettings: "保存设置",
    toggleTheme: "主题设置",
    toggleLocale: "中英文切换",
  },
  shell: {
    loadingTitle: "设计器加载中",
    loadingSubtitle: "正在准备画布与资源...",
    untitledPage: "未命名页面",
  },
  message: {
    noUndo: "没有可撤销的操作",
    noRedo: "没有可重做的操作",
    saveSuccess: "保存成功",
  },
} as const;
```

```ts
// designer/src/i18n/messages/en.ts
export const enMessages = {
  toolbar: {
    preview: "Preview",
    pagePreview: "Page Preview",
    appPreview: "App Preview",
    save: "Save",
    saveSettings: "Save Settings",
    toggleTheme: "Theme",
    toggleLocale: "Switch Language",
  },
  shell: {
    loadingTitle: "Designer is loading",
    loadingSubtitle: "Preparing canvas and assets...",
    untitledPage: "Untitled Page",
  },
  message: {
    noUndo: "Nothing to undo",
    noRedo: "Nothing to redo",
    saveSuccess: "Saved",
  },
} as const;
```

```ts
// designer/src/i18n/index.ts
import { createI18n } from "vue-i18n";
import { enMessages } from "./messages/en";
import { zhMessages } from "./messages/zh";

export const messages = { zh: zhMessages, en: enMessages };

export const i18n = createI18n({
  legacy: false,
  locale: "zh",
  fallbackLocale: "zh",
  messages,
});
```

- [ ] **Step 5: 安装依赖并再次运行测试**

Run: `pnpm --dir designer add vue-i18n`

Run: `pnpm --dir designer exec vitest --run src/stores/editor-ui-store.test.ts src/utils/storage.test.ts`

Expected: PASS，输出 `2 passed` 或更多通过项。

- [ ] **Step 6: 提交这一批基础设施改动**

```bash
git add designer/package.json designer/pnpm-lock.yaml designer/src/i18n/index.ts designer/src/i18n/messages/zh.ts designer/src/i18n/messages/en.ts designer/src/stores/editor-ui-store.ts designer/src/stores/editor-ui-store.test.ts designer/src/utils/storage.ts designer/src/utils/storage.test.ts
git commit -m "feat(designer): 建立编辑器主题与国际化基础设施"
```

## Task 2: 打通 `designer` 的启动、URL、消息与根部件接入

**Files:**
- Modify: `designer/src/main.ts`
- Create: `designer/src/router/runtime-settings.ts`
- Modify: `designer/src/router/index.ts`
- Modify: `designer/src/App.vue`
- Test: `designer/src/stores/editor-ui-store.test.ts`

- [ ] **Step 1: 补一组运行时同步测试，先让它失败**

```ts
// 追加到 designer/src/stores/editor-ui-store.test.ts
it("收到宿主消息时可更新语言", () => {
  stubStorage();
  const ui = createEditorUiStore();
  ui.initFromRuntime();

  ui.setLocale("en");

  expect(ui.locale.value).toBe("en");
  expect(store.language).toBe(JSON.stringify("en"));
});
```

```ts
// 新增到 designer/src/stores/editor-ui-store.test.ts 或独立 helper test
import { parseRuntimeSettings } from "@/router/runtime-settings";

it("解析 URL 时会消费 locale 参数", () => {
  const url = new URL("http://localhost/designer/?theme=dark&locale=en&pid=p1");
  const result = parseRuntimeSettings(url, { theme: "light", locale: "zh" });

  expect(result.theme).toBe("dark");
  expect(result.locale).toBe("en");
});
```

- [ ] **Step 2: 运行针对性测试，确认失败**

Run: `pnpm --dir designer exec vitest --run src/stores/editor-ui-store.test.ts`

Expected: FAIL，提示 `parseRuntimeSettings` 未导出或 URL 中 `locale` 未生效。

- [ ] **Step 3: 在 `main.ts`、`router`、`App.vue` 接入统一入口**

```ts
// designer/src/main.ts
import { i18n } from "./i18n";
import { createEditorUiStore } from "@/stores/editor-ui-store";

const editorUi = createEditorUiStore();
app.provide("editorUi", editorUi);
app.use(i18n);

function handleRuntimeMessage(event: MessageEvent) {
  const data = event.data as { type?: string; theme?: string; locale?: string };
  if (data.type === "THEME_UPDATE" && (data.theme === "light" || data.theme === "dark")) {
    editorUi.setTheme(data.theme);
  }
  if (data.type === "LOCALE_UPDATE" && (data.locale === "zh" || data.locale === "en")) {
    editorUi.setLocale(data.locale);
    i18n.global.locale.value = data.locale;
  }
}
```

```ts
// designer/src/router/runtime-settings.ts
export function parseRuntimeSettings(url: URL, fallback: { theme: string; locale: string }) {
  const params = url.searchParams;
  const runtimeTheme = params.get("theme");
  const runtimeLocale = params.get("locale");

  return {
    theme: runtimeTheme === "dark" || runtimeTheme === "light" ? runtimeTheme : fallback.theme,
    locale: runtimeLocale === "en" || runtimeLocale === "zh" ? runtimeLocale : fallback.locale,
  };
}
```

```vue
<!-- designer/src/App.vue -->
<script setup lang="ts">
import { computed, inject, watch } from "vue";
import { ElConfigProvider } from "element-plus";
import { useI18n } from "vue-i18n";
import { createEditorUiStore } from "@/stores/editor-ui-store";

const editorUi = inject<ReturnType<typeof createEditorUiStore>>("editorUi")!;
const { locale } = useI18n();

const elementLocale = computed(() => editorUi.elementLocale.value);

watch(
  () => editorUi.locale.value,
  (value) => {
    locale.value = value;
  },
  { immediate: true },
);
</script>

<template>
  <el-config-provider :locale="elementLocale">
    <router-view />
    <!-- 保留原有 loading -->
  </el-config-provider>
</template>
```

- [ ] **Step 4: 运行测试和类型检查**

Run: `pnpm --dir designer exec vitest --run src/stores/editor-ui-store.test.ts src/utils/storage.test.ts`

Expected: PASS

Run: `pnpm --dir designer typecheck`

Expected: PASS

- [ ] **Step 5: 提交运行时接入**

```bash
git add designer/src/main.ts designer/src/router/runtime-settings.ts designer/src/router/index.ts designer/src/App.vue designer/src/stores/editor-ui-store.test.ts
git commit -m "feat(designer): 接入编辑器主题与语言运行时同步"
```

## Task 3: 打通 `dev_ide` 到 `designer` 的 URL 与广播链路

**Files:**
- Modify: `dev_ide/src/utils/appUrl.js`
- Create: `dev_ide/src/views/dashboard/embedded-sync.js`
- Modify: `dev_ide/src/views/Dashboard.vue`
- Modify: `dev_ide/tests/run-tests.js`

- [ ] **Step 1: 先补 `dev_ide` 断言测试**

```js
// 追加到 dev_ide/tests/run-tests.js
import { buildAppUrl } from '../src/utils/appUrl.js'
import { Storage } from '../src/utils/storage.js'
import { syncEmbeddedLocale } from '../src/views/dashboard/embedded-sync.js'

run('设计中心地址：会带上 theme 和 locale', () => {
  const originalStorage = {
    getToken: Storage.getToken,
    getRefreshToken: Storage.getRefreshToken,
    getTheme: Storage.getTheme,
    getLanguage: Storage.getLanguage,
  }

  Storage.getToken = () => 't1'
  Storage.getRefreshToken = () => 'rt1'
  Storage.getTheme = () => 'dark'
  Storage.getLanguage = () => 'en'

  const url = buildAppUrl('designer', { id: 'p1', tenantId: 'tenant-1' })

  assert.equal(url.includes('theme=dark'), true)
  assert.equal(url.includes('locale=en'), true)

  Object.assign(Storage, originalStorage)
})

run('Dashboard 广播：会向嵌入 iframe 发送语言同步消息', () => {
  const messages = []
  const iframes = [
    { contentWindow: { postMessage: (payload) => messages.push(payload) } },
  ]

  syncEmbeddedLocale(iframes, 'en')

  assert.deepEqual(messages[0], { type: 'LOCALE_UPDATE', locale: 'en' })
})
```

- [ ] **Step 2: 运行 `dev_ide` 测试，确认先失败**

Run: `pnpm --dir dev_ide test`

Expected: FAIL，提示 `Storage.getLanguage` 不存在或 `buildAppUrl` 未包含 `locale`。

- [ ] **Step 3: 实现 URL 和广播同步**

```js
// dev_ide/src/views/dashboard/embedded-sync.js
export const syncEmbeddedTheme = (iframes, theme) => {
  iframes.forEach((iframe) => {
    iframe.contentWindow?.postMessage({ type: 'THEME_UPDATE', theme }, '*')
  })
}

export const syncEmbeddedLocale = (iframes, locale) => {
  iframes.forEach((iframe) => {
    iframe.contentWindow?.postMessage({ type: 'LOCALE_UPDATE', locale }, '*')
  })
}
```

```js
// dev_ide/src/utils/appUrl.js
const theme = Storage.getTheme();
const locale = Storage.getLanguage?.();

if (theme) params.set("theme", theme);
if (locale) params.set("locale", locale);
```

```js
// dev_ide/src/views/Dashboard.vue
import { syncEmbeddedLocale, syncEmbeddedTheme } from '@/views/dashboard/embedded-sync.js'

watch(isDark, (nextIsDark) => {
  const theme = nextIsDark ? 'dark' : 'light'
  syncEmbeddedTheme(document.querySelectorAll('iframe.embedded-iframe'), theme)
})

watch(locale, (nextLocale) => {
  syncEmbeddedLocale(document.querySelectorAll('iframe.embedded-iframe'), nextLocale)
  tabs.value = tabs.value.map((tab) => {
    if (tab.titleKey) {
      return { ...tab, title: t(tab.titleKey) }
    }
    return tab
  })
})
```

- [ ] **Step 4: 再跑 `dev_ide` 测试**

Run: `pnpm --dir dev_ide test`

Expected: PASS，输出 `ALL TESTS PASSED`。

- [ ] **Step 5: 提交宿主同步链路**

```bash
git add dev_ide/src/utils/appUrl.js dev_ide/src/views/dashboard/embedded-sync.js dev_ide/src/views/Dashboard.vue dev_ide/tests/run-tests.js
git commit -m "feat(dev_ide): 同步设计中心主题与语言参数"
```

## Task 4: 迁移 `designer` 壳层高频文案并启用本地切换

**Files:**
- Modify: `designer/src/ui/shell/DesignerView.vue`
- Modify: `designer/src/ui/shell/TopToolbar/TopToolbar.vue`
- Create: `designer/src/ui/shell/TopToolbar/TopToolbar.i18n.test.ts`
- Modify: `designer/src/i18n/messages/zh.ts`
- Modify: `designer/src/i18n/messages/en.ts`

- [ ] **Step 1: 先写壳层国际化测试**

```ts
// designer/src/ui/shell/TopToolbar/TopToolbar.i18n.test.ts
import { mount } from "@vue/test-utils";
import { describe, expect, it } from "vitest";
import { createI18n } from "vue-i18n";
import TopToolbar from "./TopToolbar.vue";
import { enMessages } from "@/i18n/messages/en";
import { zhMessages } from "@/i18n/messages/zh";

function mountToolbar(locale: "zh" | "en") {
  const i18n = createI18n({
    legacy: false,
    locale,
    messages: { zh: zhMessages, en: enMessages },
  });

  return mount(TopToolbar, {
    global: {
      plugins: [i18n],
      stubs: ["el-button", "el-dropdown", "el-dropdown-menu", "el-dropdown-item", "el-tooltip"],
    },
    props: {
      pageName: "Demo",
    },
  });
}

describe("TopToolbar i18n", () => {
  it("中文状态下显示预览与保存", () => {
    const wrapper = mountToolbar("zh");
    expect(wrapper.text()).toContain("预览");
    expect(wrapper.text()).toContain("保存");
  });

  it("英文状态下显示 Preview 与 Save", () => {
    const wrapper = mountToolbar("en");
    expect(wrapper.text()).toContain("Preview");
    expect(wrapper.text()).toContain("Save");
  });
});
```

- [ ] **Step 2: 运行测试，确认当前失败**

Run: `pnpm --dir designer exec vitest --run src/ui/shell/TopToolbar/TopToolbar.i18n.test.ts`

Expected: FAIL，因 `TopToolbar.vue` 仍使用硬编码文案且未注入 `useI18n`。

- [ ] **Step 3: 在 `TopToolbar` 与 `DesignerView` 中替换高频文案并接入本地切换**

```ts
// designer/src/ui/shell/TopToolbar/TopToolbar.vue
import { useI18n } from "vue-i18n";

const { t } = useI18n();

const saveStatusText = computed(() => {
  if (props.isSaving) return t("toolbar.saveSaving");
  if (props.isDirty) return t("toolbar.saveDirty");
  return t("toolbar.saveSaved");
});
```

```vue
<!-- designer/src/ui/shell/TopToolbar/TopToolbar.vue -->
<span class="split-action__text">
  <IconLucidePlay />
  <span>{{ t("toolbar.preview") }}</span>
</span>
```

```ts
// designer/src/ui/shell/DesignerView.vue
import { useI18n } from "vue-i18n";
import { inject } from "vue";

const { t, locale } = useI18n();
const editorUi = inject("editorUi") as ReturnType<typeof createEditorUiStore>;

function handleToggleTheme() {
  editorUi.setTheme(editorUi.theme.value === "dark" ? "light" : "dark");
}

function handleToggleLocale() {
  const nextLocale = editorUi.locale.value === "zh" ? "en" : "zh";
  editorUi.setLocale(nextLocale);
  locale.value = nextLocale;
}

function handleUndo() {
  if (!editorStore.undo()) {
    ElMessage.info(t("message.noUndo"));
  }
}
```

- [ ] **Step 4: 扩展字典并跑组件测试**

```ts
// designer/src/i18n/messages/zh.ts
toolbar: {
  preview: "预览",
  save: "保存",
  saveSaving: "保存中...",
  saveDirty: "未保存",
  saveSaved: "已保存",
  exportPage: "导出页面",
  localeToggle: "中英文切换",
}
```

```ts
// designer/src/i18n/messages/en.ts
toolbar: {
  preview: "Preview",
  save: "Save",
  saveSaving: "Saving...",
  saveDirty: "Unsaved",
  saveSaved: "Saved",
  exportPage: "Export Page",
  localeToggle: "Switch Language",
}
```

Run: `pnpm --dir designer exec vitest --run src/ui/shell/TopToolbar/TopToolbar.i18n.test.ts src/stores/editor-ui-store.test.ts`

Expected: PASS

- [ ] **Step 5: 跑完整静态校验**

Run: `pnpm --dir designer typecheck`

Expected: PASS

Run: `pnpm --dir designer test`

Expected: PASS

- [ ] **Step 6: 提交壳层文案与本地切换能力**

```bash
git add designer/src/ui/shell/DesignerView.vue designer/src/ui/shell/TopToolbar/TopToolbar.vue designer/src/ui/shell/TopToolbar/TopToolbar.i18n.test.ts designer/src/i18n/messages/zh.ts designer/src/i18n/messages/en.ts
git commit -m "feat(designer): 接入编辑器壳层国际化与本地切换"
```

## Task 5: 完整回归与手工边界验证

**Files:**
- Modify: `docs/superpowers/specs/2026-04-20-designer-editor-theme-i18n-sync-design.md`（如实现与规格有微调时补充）

- [ ] **Step 1: 跑两侧最终命令**

Run: `pnpm --dir designer typecheck`

Expected: PASS

Run: `pnpm --dir designer test`

Expected: PASS

Run: `pnpm --dir designer build`

Expected: PASS

Run: `pnpm --dir dev_ide test`

Expected: PASS with `ALL TESTS PASSED`

- [ ] **Step 2: 做嵌入场景手工验证**

```text
1. 在 IDE 中打开设计中心
2. 确认设计中心初始语言和主题与 IDE 一致
3. 在 IDE 中切换主题，确认设计器壳层变更，画布内用户页面不变
4. 在 IDE 中切换语言，确认设计器壳层与高频面板文案变更，画布内用户页面不变
```

- [ ] **Step 3: 做独立打开与本地覆盖验证**

```text
1. 单独打开 /designer/
2. 确认读取 localStorage 的 theme 与 language
3. 在设计中心里切换主题和语言
4. 刷新页面，确认设置保留
5. 返回 IDE，确认 IDE 自身不被设计器本地切换反向影响
```

- [ ] **Step 4: 汇总验证结果并修正文档差异**

```md
- 若手工验证与 spec 一致：不改 spec
- 若出现实现上合理但与 spec 略有差异的细节：只补充 `docs/superpowers/specs/2026-04-20-designer-editor-theme-i18n-sync-design.md` 的对应条目
```

- [ ] **Step 5: 提交最终验证与收尾**

```bash
git diff --quiet -- docs/superpowers/specs/2026-04-20-designer-editor-theme-i18n-sync-design.md || (
  git add docs/superpowers/specs/2026-04-20-designer-editor-theme-i18n-sync-design.md &&
  git commit -m "test(designer): 同步主题与国际化最终验证"
)
```
