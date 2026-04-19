# PageInspector 三分区改造 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将页面属性栏重构为“基本/视觉/运行”三分区，仅保留已确认的页面级配置，并统一运行配置语义。

**Architecture:** 把“页面配置归一化”和“分区字段清单”从 SFC 中抽离为可测试的纯函数模块，先用 Vitest 写回归测试，再在 `PageInspectorPanel.vue` 接入。模板层只负责渲染，配置字段增删和兼容映射都走 helper，避免后续继续在 SFC 内堆逻辑。

**Tech Stack:** Vue 3 + TypeScript + Pinia + Element Plus + Vitest

---

## File Structure

- Create: `designer/src/ui/editors/page/panels/right/page-inspector-config.ts`
  - 职责：窗口类型兼容映射、页面配置 patch 构建（只输出允许字段）。
- Create: `designer/src/ui/editors/page/panels/right/page-inspector-sections.ts`
  - 职责：页面属性分区字段清单与运行约束开关逻辑。
- Create: `designer/src/ui/editors/page/panels/right/page-inspector-config.test.ts`
  - 职责：验证配置 patch 输出、兼容映射与“剔除字段”规则。
- Create: `designer/src/ui/editors/page/panels/right/page-inspector-sections.test.ts`
  - 职责：验证分区顺序、字段集合、运行区联动规则。
- Modify: `designer/src/ui/editors/page/panels/right/PageInspectorPanel.vue`
  - 职责：接入三分区与 helper，移除不需要字段，保留权限配置预留入口。

### Task 1: 配置归一化 helper（TDD）

**Files:**
- Create: `designer/src/ui/editors/page/panels/right/page-inspector-config.ts`
- Create: `designer/src/ui/editors/page/panels/right/page-inspector-config.test.ts`

- [ ] **Step 1: 写失败测试（窗口类型映射 + 允许字段白名单）**

```ts
// designer/src/ui/editors/page/panels/right/page-inspector-config.test.ts
import { describe, expect, it } from "vitest";
import { buildPageConfigPatch, normalizeWindowStyle } from "./page-inspector-config";

describe("page-inspector-config", () => {
  it("将 legacy normal 映射为 replace", () => {
    expect(normalizeWindowStyle("normal")).toBe("replace");
    expect(normalizeWindowStyle("popup")).toBe("popup");
    expect(normalizeWindowStyle("cover")).toBe("cover");
    expect(normalizeWindowStyle("replace")).toBe("replace");
  });

  it("只输出允许的页面配置字段", () => {
    const patch = buildPageConfigPatch({
      description: "desc",
      width: 1920,
      height: 1080,
      autoFit: true,
      lockAspectRatio: true,
      enableMinSize: true,
      windowStyle: "cover",
      permissionDesc: "0item",
      backgroundKind: "color",
      backgroundValue: "#ffffff",
    });

    expect(patch).toMatchObject({
      description: "desc",
      width: 1920,
      height: 1080,
      autoFit: true,
      lockAspectRatio: true,
      enableMinSize: true,
      windowStyle: "cover",
      permissionDesc: "0item",
      background: { kind: "color", value: "#ffffff" },
    });

    expect("showGrid" in patch).toBe(false);
    expect("enableSnap" in patch).toBe(false);
    expect("fontAutoFit" in patch).toBe(false);
    expect("windowWidth" in patch).toBe(false);
    expect("windowHeight" in patch).toBe(false);
    expect("x" in patch).toBe(false);
    expect("y" in patch).toBe(false);
  });
});
```

- [ ] **Step 2: 运行测试确认失败**

Run: `pnpm --dir designer exec vitest --run src/ui/editors/page/panels/right/page-inspector-config.test.ts`  
Expected: FAIL，提示 `Cannot find module './page-inspector-config'`。

- [ ] **Step 3: 最小实现 helper**

```ts
// designer/src/ui/editors/page/panels/right/page-inspector-config.ts
export type WindowStyle = "popup" | "cover" | "replace";

export interface PageInspectorConfigInput {
  description: string;
  width: number;
  height: number;
  autoFit: boolean;
  lockAspectRatio: boolean;
  enableMinSize: boolean;
  windowStyle: WindowStyle;
  permissionDesc: string;
  backgroundKind: "color" | "image" | "gradient";
  backgroundValue: string;
}

export function normalizeWindowStyle(value: unknown): WindowStyle {
  if (value === "normal") return "replace";
  if (value === "popup" || value === "cover" || value === "replace") return value;
  return "cover";
}

function toPositiveInt(value: unknown, fallback: number): number {
  const next = Number(value);
  return Number.isFinite(next) && next > 0 ? Math.round(next) : fallback;
}

export function buildPageConfigPatch(input: PageInspectorConfigInput) {
  return {
    description: String(input.description || ""),
    width: toPositiveInt(input.width, 1920),
    height: toPositiveInt(input.height, 1080),
    autoFit: Boolean(input.autoFit),
    lockAspectRatio: Boolean(input.lockAspectRatio),
    enableMinSize: Boolean(input.enableMinSize),
    windowStyle: normalizeWindowStyle(input.windowStyle),
    permissionDesc: String(input.permissionDesc || "0item"),
    background: {
      kind: input.backgroundKind,
      value: String(input.backgroundValue || "#ffffff"),
    },
  };
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `pnpm --dir designer exec vitest --run src/ui/editors/page/panels/right/page-inspector-config.test.ts`  
Expected: PASS。

- [ ] **Step 5: 提交**

```bash
git add designer/src/ui/editors/page/panels/right/page-inspector-config.ts designer/src/ui/editors/page/panels/right/page-inspector-config.test.ts
git commit -m "refactor(designer): 提取页面属性配置归一化逻辑"
```

### Task 2: 分区结构 helper（TDD）

**Files:**
- Create: `designer/src/ui/editors/page/panels/right/page-inspector-sections.ts`
- Create: `designer/src/ui/editors/page/panels/right/page-inspector-sections.test.ts`

- [ ] **Step 1: 写失败测试（分区字段与顺序）**

```ts
// designer/src/ui/editors/page/panels/right/page-inspector-sections.test.ts
import { describe, expect, it } from "vitest";
import { PAGE_INSPECTOR_SECTIONS, isRuntimeConstraintDisabled } from "./page-inspector-sections";

describe("page-inspector-sections", () => {
  it("分区顺序固定为 基本/视觉/运行", () => {
    expect(PAGE_INSPECTOR_SECTIONS.map((s) => s.key)).toEqual(["basic", "visual", "runtime"]);
  });

  it("运行区包含窗口类型与缩放约束，不包含字体自适应", () => {
    const runtime = PAGE_INSPECTOR_SECTIONS.find((s) => s.key === "runtime");
    expect(runtime?.fields).toEqual(["windowStyle", "autoFit", "lockAspectRatio", "enableMinSize"]);
    expect(runtime?.fields.includes("fontAutoFit" as never)).toBe(false);
  });

  it("autoFit 关闭时运行约束禁用", () => {
    expect(isRuntimeConstraintDisabled(false)).toBe(true);
    expect(isRuntimeConstraintDisabled(true)).toBe(false);
  });
});
```

- [ ] **Step 2: 运行测试确认失败**

Run: `pnpm --dir designer exec vitest --run src/ui/editors/page/panels/right/page-inspector-sections.test.ts`  
Expected: FAIL，提示模块不存在。

- [ ] **Step 3: 最小实现 helper**

```ts
// designer/src/ui/editors/page/panels/right/page-inspector-sections.ts
export type SectionKey = "basic" | "visual" | "runtime";

export interface InspectorSection {
  key: SectionKey;
  title: string;
  fields: string[];
}

export const PAGE_INSPECTOR_SECTIONS: InspectorSection[] = [
  { key: "basic", title: "基本", fields: ["name", "description", "pageType", "path", "permission"] },
  { key: "visual", title: "视觉", fields: ["width", "height", "backgroundKind", "backgroundValue"] },
  {
    key: "runtime",
    title: "运行",
    fields: ["windowStyle", "autoFit", "lockAspectRatio", "enableMinSize"],
  },
];

export function isRuntimeConstraintDisabled(autoFit: boolean): boolean {
  return !autoFit;
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `pnpm --dir designer exec vitest --run src/ui/editors/page/panels/right/page-inspector-sections.test.ts`  
Expected: PASS。

- [ ] **Step 5: 提交**

```bash
git add designer/src/ui/editors/page/panels/right/page-inspector-sections.ts designer/src/ui/editors/page/panels/right/page-inspector-sections.test.ts
git commit -m "test(designer): 增加页面属性分区与运行约束测试"
```

### Task 3: PageInspectorPanel 三分区模板改造

**Files:**
- Modify: `designer/src/ui/editors/page/panels/right/PageInspectorPanel.vue`

- [ ] **Step 1: 写失败测试（配置输出不再包含字体自适应）**

```ts
// 追加到 page-inspector-config.test.ts
it("不再持久化字体自适应开关", () => {
  const patch = buildPageConfigPatch({
    description: "",
    width: 1920,
    height: 1080,
    autoFit: true,
    lockAspectRatio: false,
    enableMinSize: false,
    windowStyle: "cover",
    permissionDesc: "0item",
    backgroundKind: "color",
    backgroundValue: "#ffffff",
  });
  expect("fontAutoFit" in patch).toBe(false);
});
```

- [ ] **Step 2: 运行测试确认失败（当前组件仍展示字体自适应）**

Run: `pnpm --dir designer typecheck`  
Expected: 当前不会因 UI 失败，但后续改造会引入未使用字段/类型错误，作为本任务前置校验基线。

- [ ] **Step 3: 最小实现（接入 helper + 调整分区字段）**

```ts
// PageInspectorPanel.vue script 关键片段
import { buildPageConfigPatch, normalizeWindowStyle } from "./page-inspector-config";
import { isRuntimeConstraintDisabled } from "./page-inspector-sections";

const runtimeConstraintDisabled = computed(() => isRuntimeConstraintDisabled(form.autoFit));

form.windowStyle = normalizeWindowStyle(page?.config?.windowStyle);

function handleConfigUpdate(): void {
  if (!currentPage.value) return;
  const nextConfig = {
    ...(currentPage.value.config || {}),
    ...buildPageConfigPatch({
      description: form.description,
      width: form.width,
      height: form.height,
      autoFit: form.autoFit,
      lockAspectRatio: form.lockAspectRatio,
      enableMinSize: form.enableMinSize,
      windowStyle: form.windowStyle,
      permissionDesc: form.permissionDesc,
      backgroundKind: form.backgroundKind,
      backgroundValue: form.backgroundValue,
    }),
  };
  editorStore.updateCurrentPage({ config: nextConfig });
}
```

```vue
<!-- PageInspectorPanel.vue template 关键片段 -->
<div class="section-title">运行</div>
<div class="page-prop-item">
  <div class="page-prop-label">窗口类型</div>
  <div class="page-prop-editor">
    <el-select v-model="form.windowStyle" size="small" @change="handleConfigUpdate">
      <el-option label="弹出式" value="popup" />
      <el-option label="覆盖式" value="cover" />
      <el-option label="替换式" value="replace" />
    </el-select>
  </div>
</div>
<div class="page-prop-item page-prop-item--switch">
  <div class="page-prop-label">锁定宽高比</div>
  <div class="page-prop-editor page-prop-editor-switch">
    <el-switch v-model="form.lockAspectRatio" :disabled="runtimeConstraintDisabled" @change="handleConfigUpdate" />
  </div>
</div>
<div class="page-prop-item page-prop-item--switch">
  <div class="page-prop-label">最小尺寸</div>
  <div class="page-prop-editor page-prop-editor-switch">
    <el-switch v-model="form.enableMinSize" :disabled="runtimeConstraintDisabled" @change="handleConfigUpdate" />
  </div>
</div>
<div class="page-prop-item page-prop-item--hint">权限配置（即将支持）</div>
```

- [ ] **Step 4: 运行测试与类型检查**

Run:
- `pnpm --dir designer exec vitest --run src/ui/editors/page/panels/right/page-inspector-config.test.ts src/ui/editors/page/panels/right/page-inspector-sections.test.ts`
- `pnpm --dir designer typecheck`

Expected: 全部 PASS。

- [ ] **Step 5: 提交**

```bash
git add designer/src/ui/editors/page/panels/right/PageInspectorPanel.vue
git commit -m "refactor(designer): 页面属性重构为基本视觉运行三分区"
```

### Task 4: 回归验证与收尾

**Files:**
- Modify (if needed): `designer/src/ui/editors/page/panels/right/PageInspectorPanel.vue`

- [ ] **Step 1: 手工回归清单（本地 9091）**

- 打开 `http://localhost:9091/designer/?pid=90e1748e-27c0-43da-a212-f3167011a540&tenant=550e8400-e29b-41d4-a716-446655440000&type=app`
- 验证“基本/视觉/运行”三分区标题与字段顺序。
- 验证不再出现：`窗口大小`、`位置XY`、`showGrid`、`enableSnap`、`字体自适应`。
- 验证运行区：`autoFit=关` 时 `锁定宽高比/最小尺寸` 置灰。
- 验证窗口类型下拉仅有：`弹出式/覆盖式/替换式`。

- [ ] **Step 2: 运行全量测试**

Run: `pnpm --dir designer test`  
Expected: PASS。

- [ ] **Step 3: 运行格式与类型最终检查**

Run: `pnpm --dir designer typecheck`  
Expected: PASS。

- [ ] **Step 4: 最终提交**

```bash
git add designer/src/ui/editors/page/panels/right/page-inspector-config.ts designer/src/ui/editors/page/panels/right/page-inspector-sections.ts designer/src/ui/editors/page/panels/right/page-inspector-config.test.ts designer/src/ui/editors/page/panels/right/page-inspector-sections.test.ts designer/src/ui/editors/page/panels/right/PageInspectorPanel.vue
git commit -m "feat(designer): 完成页面属性栏三分区改造与运行配置收敛"
```

- [ ] **Step 5: 输出变更说明**

在交付说明中明确：
- 页面属性已按“基本/视觉/运行”分区。
- 运行区中 `锁定宽高比/最小尺寸` 依赖 `自适应`。
- 权限配置仍为预留入口。

## Self-Review

- 规格覆盖检查：已覆盖三分区、保留字段、移除字段、窗口类型三枚举、权限预留、运行区联动。
- 占位扫描：计划中无 TBD/TODO/“后续实现”式占位步骤。
- 命名一致性：`windowStyle`、`autoFit`、`lockAspectRatio`、`enableMinSize` 在任务间保持一致。

