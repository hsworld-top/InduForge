# 设计中心编辑器主题与国际化同步设计

**日期**: 2026-04-20
**模块**: `designer`、`dev_ide`
**状态**: 已确认，待实现计划

## 1. 背景

当前 `designer` 作为设计中心编辑器嵌入在 `dev_ide` 中使用时，只具备有限的主题同步能力：

- `designer/src/main.ts` 仅监听 `THEME_UPDATE`
- `dev_ide` 打开设计中心时仅通过 URL 传递 `theme`
- 语言切换没有从 IDE 同步到设计中心
- 设计中心顶部工具栏中的主题与语言入口仍是占位逻辑

同时，本次改造必须明确隔离边界：

- 同步只作用于设计中心编辑器自身
- 画布内用户自己搭建的页面内容不跟随编辑器主题/语言变化
- 发布态与运行态字段语义不因本次编辑器 UI 改造而改变

## 2. 目标

本次首轮交付目标如下：

1. IDE 打开设计中心时，初始主题与语言同步到设计中心编辑器
2. IDE 运行中切换主题或语言时，设计中心编辑器实时同步
3. 设计中心单独打开时，优先读取本地缓存，没有则回退到 `light + zh`
4. 设计中心保留独立切换主题与语言的能力，但只影响设计中心自身
5. 编辑器壳层、面板、常用提示消息支持中英文切换
6. 画布中用户自己构建的页面区域不受影响

## 3. 非目标

本次不包含以下内容：

- 预览页或运行态页面的国际化改造
- 发布态 Schema 中 `theme`、`locale` 字段语义调整
- 画布内用户组件自动跟随编辑器主题
- 一次性清理所有历史低频硬编码文案
- 设计中心把本地切换反向同步回 IDE

## 4. 现状与问题

### 4.1 `designer` 现状

- `main.ts` 负责主题消息监听，但不管理语言消息
- `router/index.ts` 初始化时仅读取 `theme` URL 参数，不读取 `locale`
- `DesignerView.vue` 中的 `handleToggleTheme`、`handleToggleLocale` 仍为占位提示
- 项目尚未建立完整的 `vue-i18n` 编辑器文案体系，`package.json` 里也尚未引入 `vue-i18n`

### 4.2 `dev_ide` 现状

- `buildAppUrl` 仅传递 `theme`，未传递 `locale`
- `Dashboard.vue` 仅在主题变更时向嵌入 iframe 发出 `THEME_UPDATE`
- 语言变化时没有向设计中心发出运行中同步消息

### 4.3 风险

- 主题与语言初始化逻辑分散在 `main.ts`、`router`、组件内部，后续容易相互覆盖
- 设计器存在“编辑器 UI 文案”与“用户页面内容”同屏渲染区域，若边界控制不严，容易误伤用户页面

## 5. 方案选型

### 方案 A：纯宿主驱动

设计中心只接受 IDE 推送，不保留本地独立状态。

优点：

- 实现简单
- 同步路径单一

缺点：

- 无法满足“设计中心保留独立切换能力”
- 单独打开时行为不完整

### 方案 B：编辑器独立状态层 + IDE 增量同步

在 `designer` 内新增一层仅用于编辑器 UI 的偏好状态，管理 `theme` 与 `locale`。设计中心可独立切换，IDE 负责初始化与运行中同步。

优点：

- 满足嵌入和独立打开两种场景
- 可以严格限定作用域，只影响编辑器 UI
- 便于逐步扩展更多编辑器文案

缺点：

- 需要额外整理初始化与消息同步入口

### 方案 C：应用级全局主题/国际化

把整个 `designer` 全量接入应用级 i18n/theme，包括预览和运行态渲染。

优点：

- 长期扩展空间最大

缺点：

- 明显超出本次边界
- 易把编辑态与运行态耦合

### 结论

采用 **方案 B：编辑器独立状态层 + IDE 增量同步**。

## 6. 总体设计

### 6.1 核心原则

- 主题与语言只属于“编辑器 UI 偏好”，不属于页面模型
- 单一状态源负责 DOM 主题、Element Plus 语言包和本地存储同步
- 编辑器文案统一使用 `vue-i18n` 管理，避免继续扩散硬编码字符串
- IDE 只负责初始化值与运行中广播，不接管设计中心内部切换逻辑

### 6.2 设置来源优先级

设计中心启动时，编辑器 UI 设置按以下顺序解析：

1. URL 参数
2. 本地 `localStorage`
3. 默认值 `theme=light`、`locale=zh`

设计中心运行中，状态来源有两类：

- IDE `postMessage` 推送更新
- 设计中心顶部工具栏内的本地手动切换

后写入者覆盖当前编辑器状态，但只作用于设计中心自身。

## 7. 架构拆分

### 7.1 `designer` 新增编辑器 UI 状态模块

建议新增轻量状态模块，例如：

- `designer/src/stores/editor-ui-store.ts`

职责：

- 保存当前 `theme`
- 保存当前 `locale`
- 从 URL / `localStorage` 初始化
- 更新并持久化设置
- 把主题应用到 `document.documentElement`
- 暴露 Element Plus 所需的 locale 对象

建议最小接口：

```ts
type EditorTheme = "light" | "dark";
type EditorLocale = "zh" | "en";

interface EditorUiState {
  theme: EditorTheme;
  locale: EditorLocale;
}

interface EditorUiStore {
  theme: Ref<EditorTheme>;
  locale: Ref<EditorLocale>;
  initFromRuntime(input?: Partial<EditorUiState>): void;
  setTheme(theme: EditorTheme): void;
  setLocale(locale: EditorLocale): void;
  applyThemeToDom(theme?: EditorTheme): void;
  getElementLocale(): ElementPlusLocale;
}
```

### 7.2 根部件接入语言包

在 `designer` 根部件层接入 `vue-i18n` 与 `ElConfigProvider`，让应用文案和 Element Plus 组件语言包都跟随编辑器 `locale` 变化。

建议位置：

- `App.vue` 包裹 `router-view`
- `main.ts` 注册 `vue-i18n` 实例

作用：

- 编辑器壳层、面板、提示消息走统一翻译入口
- 对话框、下拉框、分页、日期等 Element Plus 组件文案统一切换

### 7.3 主题作用域

主题只应用于编辑器壳层与编辑器面板：

- 顶部工具栏
- 左右工具轨与停靠面板
- Element Plus 组件
- Monaco 编辑器主题
- 编辑器状态栏、空态、提示信息

以下区域不接入该主题状态：

- 画布内用户页面自身样式
- 用户组件配置出的文字内容
- 导出后的运行态页面

## 8. 同步协议设计

### 8.1 URL 初始同步

`dev_ide` 打开设计中心时，URL 追加：

- `theme`
- `locale`

示例：

```text
/designer/?pid=xxx&tenant=xxx&token=xxx&refreshToken=xxx&theme=dark&locale=en&type=app
```

### 8.2 运行中消息同步

约定两类消息：

```ts
type ThemeUpdateMessage = {
  type: "THEME_UPDATE";
  theme: "light" | "dark";
};

type LocaleUpdateMessage = {
  type: "LOCALE_UPDATE";
  locale: "zh" | "en";
};
```

`designer` 统一在一个入口处理消息，不再让主题与语言分别散落在不同文件中。

### 8.3 宿主与子应用关系

- IDE 负责广播当前主题与语言
- 设计中心负责消费并更新本地编辑器 UI 状态
- 设计中心本地切换不回推 IDE

## 9. 详细改造点

### 9.1 `dev_ide`

#### `src/utils/appUrl.js`

- 为 `designer` 打开地址补充 `locale`
- 读取来源与 IDE 当前语言一致

#### `src/views/Dashboard.vue`

- 在现有 `syncEmbeddedTheme` 基础上增加 `syncEmbeddedLocale`
- 监听 `locale` 变化，向嵌入的设计中心 iframe 广播 `LOCALE_UPDATE`

### 9.2 `designer`

#### `src/router/index.ts`

- `syncRuntimeSettings()` 补充对 `locale` URL 参数的解析与持久化
- 保持 URL 消费后从地址栏移除，避免刷新后污染链接

#### `src/main.ts`

- 从“只监听主题”改为“统一初始化编辑器 UI 设置 + 监听主题/语言消息”
- 避免直接在入口文件里重复拼装 DOM 写入逻辑，改为调用编辑器 UI 状态模块
- 注册 `vue-i18n` 实例，并与编辑器 UI `locale` 状态联动

#### `src/App.vue`

- 用 `I18nProvider` 语义组织应用级国际化入口，并通过 `ElConfigProvider` 包裹应用
- `vue-i18n` 当前 locale 与 Element Plus locale 都从编辑器 UI 状态模块读取

#### `src/ui/shell/DesignerView.vue`

- 把 `handleToggleTheme()` 从占位提示改为真实切换逻辑
- 把 `handleToggleLocale()` 从占位提示改为真实切换逻辑
- 面板标题、工具轨标签、常见操作消息逐步改为使用翻译字典

#### `src/ui/shared/widgets/base/MonacoEditor.vue`

- 保持现有主题跟随机制
- 显式接入编辑器主题状态，避免仅依赖 DOM 猜测
- 不处理编辑器 UI 文案国际化，仅处理编辑器主题

## 10. 国际化范围

### 10.1 首轮覆盖范围

首轮优先覆盖高频编辑器区域：

- 顶部工具栏操作项
- 左右工具轨标签
- 左右停靠面板标题
- 常用成功/失败/警告消息
- 主要空态文案

### 10.2 首轮暂缓范围

以下文案允许后续增量补齐：

- 低频异常提示
- 深层嵌套面板中的历史硬编码说明文字
- 与运行态语义重叠的配置描述文案

## 11. 边界隔离规则

为防止误把编辑器设置扩散到用户页面，需遵守以下规则：

1. 编辑器 UI 状态不得写入页面 Schema
2. 组件 `props` 中的用户文本不得自动翻译
3. 画布节点渲染不得读取编辑器 `locale` 作为业务渲染依据
4. 编辑器 `theme` 不得覆盖用户页面定义的背景、颜色、字体
5. 发布态 `theme/locale` 字段继续保留工程/运行态语义，不复用编辑器 UI 状态

## 12. 失败与降级处理

### 12.1 非法 URL 参数

- `theme` 非 `light/dark` 时回退本地值
- `locale` 非 `zh/en` 时回退本地值

### 12.2 非法消息

- 非协议消息直接忽略
- 字段不合法直接忽略，不抛异常

### 12.3 本地存储异常

- 读取失败时回退默认值
- 写入失败时不影响当前界面即时切换

## 13. 测试策略

### 13.1 静态验证

- `pnpm --dir designer typecheck`
- `pnpm --dir designer test`
- `pnpm --dir designer build`

### 13.2 行为验证

1. 从 IDE 打开设计中心，确认初始主题和语言继承 IDE
2. IDE 中切换主题，设计中心编辑器壳层和面板同步，画布用户页面不变
3. IDE 中切换语言，设计中心编辑器文案同步，画布用户页面不变
4. 单独打开设计中心，确认读取本地缓存
5. 在设计中心中手动切换主题/语言，只影响设计中心自身

## 14. 实施顺序

建议按以下顺序进入实现：

1. 统一编辑器 UI 状态入口
2. 打通 IDE URL 与 `postMessage` 双通道同步
3. 接入 Element Plus locale
4. 替换设计器主链路高频文案
5. 验证“画布用户页面不受影响”这一关键边界

## 15. 实现约束补充

为减少后续实现计划的分歧，补充以下落地约束：

### 15.1 翻译字典组织

首轮建议采用集中式字典文件：

- `designer/src/i18n/index.ts`
- `designer/src/i18n/messages/zh.ts`
- `designer/src/i18n/messages/en.ts`

技术约束：

- 国际化模块统一采用 `vue-i18n`
- 参照 `dev_ide/src/lang/index.js` 的消息组织方式，但只保留设计中心编辑器所需子集
- 采用组合式 API 模式，避免引入 legacy 配置

组织规则：

- 页面模型、运行态字段名不进入翻译字典
- 键名以编辑器 UI 语义组织，例如 `toolbar.save`、`rail.pages`、`message.saveSuccess`
- 首轮不拆分到每个面板单独文件，避免早期过度切碎

### 15.2 首轮文案覆盖清单

首轮必须覆盖以下区域：

- `TopToolbar.vue` 中的主要操作项与下拉项
- `DesignerView.vue` 中的通用 `ElMessage` 提示
- 左右工具轨标签
- 左右停靠面板标题
- 页面管理主链路相关空态和常用按钮文案

以下区域允许第二轮再补：

- 深层低频配置说明
- 复杂对话框的长段说明文本
- 与运行态配置耦合较重的专业文案
