# 设计器工程全局脚本面板实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 将左侧脚本面板优化为工程全局脚本配置中心，并抽出工程脚本和页面脚本共用的脚本编辑弹窗。

**架构：** 保留现有 `globalScripts` 数据结构和运行时消费逻辑。新增通用脚本编辑弹窗，将 `ScriptVarsSystemScriptDialog.vue` 与 `ScriptVarsPanel.vue` 内部编辑弹窗的重复结构收敛到统一组件。左侧全局脚本面板只负责工程级脚本，右侧高级面板保持页面级入口并通过作用域文案区分。

**技术栈：** Vue 3、TypeScript、Pinia、Element Plus、Monaco Editor、Vitest、vue-tsc。

---

## 文件结构

- 修改：`designer/src/ui/shared/tool-panels/ScriptVarsPanel.vue`
  - 调整左侧工程全局脚本面板标题、区域展示、系统脚本单例交互。
  - 使用通用脚本编辑弹窗。

- 创建：`designer/src/ui/shared/tool-panels/ScriptEditorDialog.vue`
  - 通用脚本编辑弹窗。
  - 支持工程全局和当前页面作用域文案。
  - 承载 Monaco、辅助插入侧栏、保存、关闭确认。

- 修改：`designer/src/ui/shared/tool-panels/ScriptVarsSystemScriptDialog.vue`
  - 删除重复编辑器主体或改为薄封装。
  - 复用 `ScriptEditorDialog.vue`。

- 修改：`designer/src/ui/shared/tool-panels/ScriptVarsSystemSection.vue`
  - 系统脚本区域改为固定条目，不提供新增入口。
  - 标题和条目文案体现工程全局语义。

- 修改：`designer/src/ui/shared/tool-panels/ScriptVarsTimersSection.vue`
  - 区域标题改为全局定时器。
  - 标题右侧保留新增入口。

- 修改：`designer/src/ui/shared/tool-panels/ScriptVarsVariableChangesSection.vue`
  - 区域标题改为全局变量监听。
  - 标题右侧保留新增入口。

- 修改：`designer/src/ui/shared/tool-panels/ScriptVarsCustomSection.vue`
  - 区域标题改为全局自定义函数。
  - 标题右侧保留新增入口。

- 修改：`designer/src/ui/editors/page/sidebar-panels/right/AdvancedPanel.vue`
  - 保持页面级脚本入口语义。
  - 文案明确为当前页面脚本。

- 修改：`designer/src/ui/editors/page/sidebar-panels/right/EventPanel.vue`
  - 页面级脚本弹窗切换到 `ScriptEditorDialog.vue`。
  - 保存仍写当前页面生命周期配置。

- 修改：`designer/src/i18n/messages/zh.ts`
  - 增加工程全局脚本、当前页面脚本、作用域标签、区域标题文案。

- 修改：`designer/src/i18n/messages/en.ts`
  - 同步英文文案。

- 测试：`designer/src/ui/shared/tool-panels/ScriptVarsDialogs.i18n.test.ts`
  - 覆盖通用脚本弹窗作用域文案。

- 新增测试：`designer/src/ui/shared/tool-panels/ScriptEditorDialog.test.ts`
  - 覆盖保存、取消、未保存关闭确认、作用域显示。

- 新增测试：`designer/src/ui/shared/tool-panels/ScriptVarsPanel.global.test.ts`
  - 覆盖系统脚本没有新增入口，其他三类有新增入口。

---

### 任务 1：新增通用脚本编辑弹窗

**文件：**

- 创建：`designer/src/ui/shared/tool-panels/ScriptEditorDialog.vue`
- 新增测试：`designer/src/ui/shared/tool-panels/ScriptEditorDialog.test.ts`

- [ ] **步骤 1：编写失败测试**

测试内容：

```ts
import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import ScriptEditorDialog from './ScriptEditorDialog.vue'

vi.mock('@/ui/shared/widgets/base/monaco-editor-async', () => ({
  default: {
    props: ['modelValue'],
    emits: ['update:modelValue'],
    template: `<textarea class="mock-monaco" :value="modelValue" @input="$emit('update:modelValue', ($event.target as HTMLTextAreaElement).value)" />`,
  },
}))

describe('ScriptEditorDialog', () => {
  it('显示工程全局作用域并保存代码', async () => {
    const wrapper = mount(ScriptEditorDialog, {
      props: {
        modelValue: true,
        code: 'console.log(1)',
        scope: 'global',
        kind: 'custom',
        title: 'formatUser',
        completions: [],
      },
      global: {
        stubs: ['el-dialog', 'el-button', 'el-tooltip', 'el-input', 'el-tree', 'el-icon'],
      },
    })

    expect(wrapper.text()).toContain('工程全局')
    await wrapper.find('.mock-monaco').setValue('console.log(2)')
    await wrapper.find('[data-test="script-editor-save"]').trigger('click')
    expect(wrapper.emitted('save')?.[0]).toEqual(['console.log(2)'])
  })
})
```

- [ ] **步骤 2：运行测试验证失败**

运行：

```powershell
pnpm --dir designer exec vitest --run src/ui/shared/tool-panels/ScriptEditorDialog.test.ts
```

预期：失败，提示组件不存在。

- [ ] **步骤 3：实现最小组件**

创建 `ScriptEditorDialog.vue`，包含：

- `modelValue`
- `code`
- `scope`
- `kind`
- `title`
- `completions`
- `save` 事件
- `update:modelValue`
- `update:code`

保存按钮加 `data-test="script-editor-save"`。

- [ ] **步骤 4：运行测试验证通过**

运行：

```powershell
pnpm --dir designer exec vitest --run src/ui/shared/tool-panels/ScriptEditorDialog.test.ts
```

预期：通过。

---

### 任务 2：通用弹窗接入系统脚本

**文件：**

- 修改：`designer/src/ui/shared/tool-panels/ScriptVarsPanel.vue`
- 修改：`designer/src/ui/shared/tool-panels/ScriptVarsSystemScriptDialog.vue`
- 测试：`designer/src/ui/shared/tool-panels/ScriptVarsDialogs.i18n.test.ts`

- [ ] **步骤 1：编写失败测试**

在 `ScriptVarsDialogs.i18n.test.ts` 增加断言：

```ts
expect(wrapper.text()).toContain('工程全局')
expect(wrapper.text()).toContain('系统启动')
```

- [ ] **步骤 2：运行测试验证失败**

运行：

```powershell
pnpm --dir designer exec vitest --run src/ui/shared/tool-panels/ScriptVarsDialogs.i18n.test.ts
```

预期：失败，当前弹窗未显示工程全局作用域。

- [ ] **步骤 3：接入通用弹窗**

把系统脚本编辑弹窗改为使用：

```vue
<ScriptEditorDialog
  v-model="systemEditorVisible"
  v-model:code="systemCode"
  scope="global"
  :kind="selectedSystemKey === 'startup' ? 'systemStartup' : 'systemShutdown'"
  :title="selectedSystemLabel"
  :completions="jsCompletions"
  @save="saveSystemScript"
/>
```

- [ ] **步骤 4：运行测试验证通过**

运行：

```powershell
pnpm --dir designer exec vitest --run src/ui/shared/tool-panels/ScriptVarsDialogs.i18n.test.ts
```

预期：通过。

---

### 任务 3：优化工程全局脚本面板信息架构

**文件：**

- 修改：`designer/src/ui/shared/tool-panels/ScriptVarsPanel.vue`
- 修改：`designer/src/ui/shared/tool-panels/ScriptVarsSystemSection.vue`
- 修改：`designer/src/ui/shared/tool-panels/ScriptVarsTimersSection.vue`
- 修改：`designer/src/ui/shared/tool-panels/ScriptVarsVariableChangesSection.vue`
- 修改：`designer/src/ui/shared/tool-panels/ScriptVarsCustomSection.vue`
- 新增测试：`designer/src/ui/shared/tool-panels/ScriptVarsPanel.global.test.ts`

- [ ] **步骤 1：编写失败测试**

测试断言：

```ts
expect(wrapper.text()).toContain('工程全局脚本')
expect(wrapper.text()).toContain('系统脚本')
expect(wrapper.text()).toContain('全局定时器')
expect(wrapper.text()).toContain('全局变量监听')
expect(wrapper.text()).toContain('全局自定义函数')
expect(wrapper.find('[data-test="system-add"]').exists()).toBe(false)
expect(wrapper.find('[data-test="timer-add"]').exists()).toBe(true)
expect(wrapper.find('[data-test="variable-change-add"]').exists()).toBe(true)
expect(wrapper.find('[data-test="custom-script-add"]').exists()).toBe(true)
```

- [ ] **步骤 2：运行测试验证失败**

运行：

```powershell
pnpm --dir designer exec vitest --run src/ui/shared/tool-panels/ScriptVarsPanel.global.test.ts
```

预期：失败，当前没有这些测试标识和文案。

- [ ] **步骤 3：调整面板结构**

在 `ScriptVarsPanel.vue` 顶部增加标题区：

```vue
<div class="global-scripts__header">
  <div class="global-scripts__title">{{ t("scriptPanel.global.title") }}</div>
  <div class="global-scripts__desc">{{ t("scriptPanel.global.description") }}</div>
</div>
```

系统脚本区只保留固定两项。

其他三类区块标题右侧保留新增按钮，并加 `data-test`。

- [ ] **步骤 4：运行测试验证通过**

运行：

```powershell
pnpm --dir designer exec vitest --run src/ui/shared/tool-panels/ScriptVarsPanel.global.test.ts
```

预期：通过。

---

### 任务 4：高级面板页面脚本语义区分

**文件：**

- 修改：`designer/src/ui/editors/page/sidebar-panels/right/AdvancedPanel.vue`
- 修改：`designer/src/ui/editors/page/sidebar-panels/right/EventPanel.vue`
- 测试：新增或扩展右侧高级面板测试

- [ ] **步骤 1：编写失败测试**

断言高级面板出现：

```ts
expect(wrapper.text()).toContain('当前页面脚本')
```

并且不出现：

```ts
expect(wrapper.text()).not.toContain('工程全局脚本')
```

- [ ] **步骤 2：运行测试验证失败**

运行：

```powershell
pnpm --dir designer exec vitest --run src/ui/editors/page/sidebar-panels/right/AdvancedPanel*.test.ts
```

预期：失败或没有匹配测试，需要新增测试文件。

- [ ] **步骤 3：接入文案和通用弹窗**

高级面板标题改为“当前页面脚本”。

页面级脚本弹窗调用：

```vue
<ScriptEditorDialog
  v-model="pageScriptEditorVisible"
  v-model:code="pageScriptCode"
  scope="page"
  :kind="pageScriptKind"
  :title="pageScriptTitle"
  :completions="jsCompletions"
  @save="savePageScript"
/>
```

- [ ] **步骤 4：运行测试验证通过**

运行：

```powershell
pnpm --dir designer exec vitest --run src/ui/editors/page/sidebar-panels/right/AdvancedPanel*.test.ts
```

预期：通过。

---

### 任务 5：文案与类型检查

**文件：**

- 修改：`designer/src/i18n/messages/zh.ts`
- 修改：`designer/src/i18n/messages/en.ts`

- [ ] **步骤 1：补充文案**

新增中文：

```ts
scriptPanel: {
  global: {
    title: "工程全局脚本",
    description: "对整个工程预览和运行态生效",
    scope: "工程全局",
  },
  page: {
    title: "当前页面脚本",
    description: "仅对当前页面生效",
    scope: "当前页面",
  },
}
```

新增英文对应：

```ts
global: {
  title: "Project Global Scripts",
  description: "Applies to the whole project in preview and runtime",
  scope: "Project Global",
},
page: {
  title: "Current Page Scripts",
  description: "Applies only to the current page",
  scope: "Current Page",
},
```

- [ ] **步骤 2：运行相关测试**

运行：

```powershell
pnpm --dir designer exec vitest --run src/ui/shared/tool-panels/ScriptVarsDialogs.i18n.test.ts src/ui/shared/tool-panels/ScriptEditorDialog.test.ts src/ui/shared/tool-panels/ScriptVarsPanel.global.test.ts
```

预期：全部通过。

- [ ] **步骤 3：运行类型检查**

运行：

```powershell
pnpm --dir designer exec vue-tsc --noEmit --pretty false
```

预期：无输出，退出码 0。

---

## 验证清单

- 打开 `http://localhost:18601/dashboard`。
- 进入设计中心。
- 打开左侧脚本面板。
- 面板标题显示“工程全局脚本”。
- 系统脚本只显示系统启动、系统关闭，不显示新增。
- 全局定时器、全局变量监听、全局自定义函数显示新增入口。
- 打开系统启动脚本，弹窗显示“工程全局”。
- 打开高级面板页面脚本，弹窗显示“当前页面”。
- 保存脚本后刷新页面，配置仍保留。
- 预览页面，现有脚本执行逻辑不回退。
