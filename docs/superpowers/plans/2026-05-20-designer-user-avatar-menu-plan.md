# 用户头像系统组件实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 新增用户头像系统组件，支持当前运行用户展示、可配置菜单、语言子菜单、页面运行态主题子菜单和退出登录。

**架构：** 按现有物料体系新增 `UserAvatarMenu` manifest、descriptor、renderer。运行态用户复用 `editor-store.runtimeUsers/selectedPreviewRuntimeUserId`，页面主题新增 `projectRuntimeTheme` 状态并写入预览/画布根容器属性。属性配置先以结构化 JSON/对象属性承载菜单模板，渲染器内部完成默认项归一化与内置动作。

**技术栈：** Vue 3、Pinia、Element Plus、Vitest、现有 designer 物料注册体系。

---

### 任务 1：运行态主题状态

**文件：**

- 修改：`designer/src/stores/editor-store.ts`
- 修改：`designer/src/stores/editor-store.types.ts`

- [x] 新增 `projectRuntimeTheme = ref<'light' | 'dark'>('light')`。
- [x] 新增 `setProjectRuntimeTheme(theme)`，只接受 `light/dark`。
- [x] store return 暴露 `projectRuntimeTheme` 和 `setProjectRuntimeTheme`。

### 任务 2：用户头像物料

**文件：**

- 创建：`designer/src/materials/UserAvatarMenu/manifest.ts`
- 创建：`designer/src/materials/UserAvatarMenu/index.ts`
- 创建：`designer/src/editor-core/descriptors/user-avatar-menu.ts`
- 创建：`designer/src/materials/UserAvatarMenu/UserAvatarMenuRenderer.vue`
- 修改：`designer/src/materials/index.ts`
- 修改：`designer/src/materials/manifests/index.ts`

- [x] manifest 默认 props 包含展示样式、显示用户名、显示副标题、头像尺寸、菜单项。
- [x] descriptor 使用 custom renderer，默认大小 170x44。
- [x] renderer 读取当前运行用户、语言列表、运行态主题。
- [x] 内置菜单项支持 profile/script、locale、theme、logout。

### 任务 3：系统组件面板接入

**文件：**

- 修改：`designer/src/ui/editors/page/sidebar-panels/left/ComponentPanel.vue`
- 修改：`designer/src/ui/editors/page/sidebar-panels/left/ComponentPanel.system.test.ts`

- [x] 系统组件固定包含 `UserAvatarMenu`。
- [x] `LanguageSwitcher` 仍按国际化开关显示。
- [x] 增加用户头像物料预览。
- [x] 更新测试覆盖显示和搜索。

### 任务 4：运行态主题注入

**文件：**

- 修改：`designer/src/ui/editors/page/canvas/CanvasContainer.vue`
- 修改：`designer/src/ui/editors/page/preview/PreviewView.vue`

- [x] 页面运行容器写入 `data-runtime-theme`。
- [x] 注入基础 CSS 变量，浅色/深色只影响用户页面区域。

### 任务 5：属性与交互验证

**文件：**

- 修改/新增测试：相关组件测试。

- [x] 运行 `pnpm --dir designer exec vue-tsc --noEmit --pretty false`。
- [x] 运行相关 Vitest。
- [x] 浏览器打开 `http://localhost:18601/dashboard` 验证系统组件、拖入、预览菜单、语言切换、主题切换。
