# Designer UI 优化实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 Designer 模块的 CSS 变量逐步替换为 DESIGN.md 设计规范定义的值，保留现有结构，采用渐进式优化。

**Architecture:** 在 `main.css` 中重映射 `--designer-*` CSS 变量为目标值（颜色、圆角、间距、字体、阴影），保持组件结构不变，同步更新 dark mode 变量。

**Tech Stack:** CSS (CSS Variables), Tailwind CSS, Vue 3

---

## 文件结构

- **Modify**: `designer/src/assets/styles/main.css` — CSS 变量重映射

---

## 实施阶段

### Task 1: 重映射 CSS 变量（light mode）

**Files:**
- Modify: `designer/src/assets/styles/main.css:7-71` (`:root` 块)

- [ ] **Step 1: 更新颜色变量**

将 `:root` 中的颜色变量替换为 DESIGN.md 规范值：

```css
:root {
  /* 背景与表面 */
  --designer-shell-bg: #f6f5f2;
  --designer-shell-surface: #ffffff;
  --designer-surface-elevated: #ffffff;
  --designer-group-surface: #eef2f7;
  --designer-chip-surface: #eef2f7;

  /* 边框 */
  --designer-border-color: #d7dde7;
  --designer-border-soft: #eef2f7;
  --designer-border-strong: #b8c0d0;

  /* 文本 */
  --designer-text-primary: #0f172a;
  --designer-text-regular: #0f172a;
  --designer-text-secondary: #475569;
  --designer-text-muted: #94a3b8;

  /* 交互状态 */
  --designer-hover-surface: #f6f5f2;
  --designer-active-surface: #eef2f7;

  /* 主色 */
  --designer-primary: #1d4ed8;
  --designer-primary-border: #1d4ed8;
  --designer-primary-text: #1d4ed8;
  --designer-primary-soft: #eef2f7;

  /* 状态色 */
  --designer-success-surface: #f0fdf4;
  --designer-success-text: #22c55e;
  --designer-warning-surface: #fffbeb;
  --designer-warning-text: #f59e0b;
  --designer-info-surface: #eff6ff;
  --designer-info-text: #0ea5a5;

  /* 阴影 */
  --designer-shadow-panel: 0 2px 6px rgba(15, 23, 42, 0.08);
  --designer-shadow-popover: 0 12px 24px rgba(15, 23, 42, 0.12);
}
```

- [ ] **Step 2: 更新圆角变量**

```css
  /* 圆角系统 */
  --designer-radius-sm: 8px;
  --designer-radius-md: 12px;
  --designer-radius-lg: 18px;
```

- [ ] **Step 3: 更新间距变量**

```css
  /* 间距系统 */
  --designer-gap-2xs: 4px;
  --designer-gap-xs: 4px;
  --designer-gap-sm: 8px;
  --designer-gap-md: 12px;
  --designer-gap-lg: 16px;
  --designer-gap-xl: 24px;
```

- [ ] **Step 4: 更新字体族**

将 `html` 和 `body` 的字体族从 Inter 改为：
```css
html {
  font-family: "Source Han Sans SC", "PingFang SC", "Microsoft YaHei", "Helvetica Neue", sans-serif;
}

body {
  @apply bg-gray-50 dark:bg-gray-900 text-gray-900 dark:text-gray-100;
  font-family: "Source Han Sans SC", "PingFang SC", "Microsoft YaHei", "Helvetica Neue", sans-serif;
}
```

- [ ] **Step 5: 提交变更**

```bash
cd designer
git add src/assets/styles/main.css
git commit -m "refactor(designer): 更新 CSS 变量为设计系统规范值 (Phase 1)"
```

---

### Task 2: 更新 Dark Mode 变量

**Files:**
- Modify: `designer/src/assets/styles/main.css:118-149` (`.dark` 块)

- [ ] **Step 1: 更新 dark mode 颜色变量**

将 `.dark` 块中的变量替换为 DESIGN.md 规范值（深色模式）：

```css
.dark {
  --designer-shell-bg: #0f172a;
  --designer-shell-surface: #111827;
  --designer-surface-elevated: #1f2937;
  --designer-group-surface: #1a2437;
  --designer-chip-surface: #1f2937;
  --designer-border-color: #273449;
  --designer-border-soft: #1e293b;
  --designer-border-strong: #334155;
  --designer-text-primary: #f8fafc;
  --designer-text-regular: #e5e7eb;
  --designer-text-secondary: #cbd5e1;
  --designer-text-muted: #94a3b8;
  --designer-hover-surface: #1e293b;
  --designer-active-surface: rgba(29, 78, 216, 0.18);
  --designer-primary: #60a5fa;
  --designer-primary-border: #3b82f6;
  --designer-primary-text: #93c5fd;
  --designer-primary-soft: rgba(59, 130, 246, 0.14);
  --designer-success-surface: rgba(34, 197, 94, 0.1);
  --designer-success-text: #86efac;
  --designer-warning-surface: rgba(245, 158, 11, 0.12);
  --designer-warning-text: #fcd34d;
  --designer-info-surface: rgba(59, 130, 246, 0.12);
  --designer-info-text: #38bdf8;
  --designer-shadow-panel: 0 8px 18px rgba(2, 6, 23, 0.32);
  --designer-shadow-popover: 0 16px 32px rgba(2, 6, 23, 0.42);
}
```

- [ ] **Step 2: 提交变更**

```bash
cd designer
git add src/assets/styles/main.css
git commit -m "refactor(designer): 更新 dark mode CSS 变量 (Phase 2)"
```

---

### Task 3: 验证与微调

**Files:**
- 检查: `designer/src/ui/shell/TopToolbar.vue`
- 检查: `designer/src/ui/shell/DockPanel/`
- 检查: `designer/src/ui/shell/ToolRail/`

- [ ] **Step 1: 启动 Designer 开发服务器**

```bash
cd designer
pnpm dev
```

打开浏览器验证以下组件的渲染效果：
- 工具栏（TopToolbar）- 按钮、标题、图标
- 面板（DockPanel）- 圆角、边框、阴影
- 工具栏（ToolRail）- 图标按钮、hover 状态
- 输入框、卡片、chip 等组件

- [ ] **Step 2: 如有异常，进行微调**

记录发现的问题并在 `main.css` 中微调。

- [ ] **Step 3: 提交最终变更**

```bash
cd designer
git add src/assets/styles/main.css
git commit -m "refactor(designer): 完成 CSS 变量优化并验证 UI 渲染"
```

---

## 验证清单

- [ ] 主色调从 `#2563eb` 变为 `#1d4ed8`
- [ ] 背景色从 `#f5f5f5` 变为 `#f6f5f2`
- [ ] 圆角从 4/6/8px 变为 8/12/18px
- [ ] 间距统一为 4/8/12/16/24px
- [ ] 字体族从 Inter 变为 Source Han Sans SC 等
- [ ] Dark mode 变量同步更新
- [ ] 所有 UI 组件渲染正常，无布局异常