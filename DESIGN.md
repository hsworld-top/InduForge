---
name: InduForge
description: 工业应用低代码开发平台的产品工作台视觉系统
colors:
  primary: '#1d4ed8'
  primary-hover: '#1e40af'
  accent: '#0ea5a5'
  success: '#22c55e'
  warning: '#f59e0b'
  danger: '#ef4444'
  background: '#f6f5f2'
  surface: '#ffffff'
  surface-muted: '#eef2f7'
  text-primary: '#0f172a'
  text-secondary: '#475569'
  text-muted: '#94a3b8'
  border: '#d7dde7'
typography:
  headline:
    fontFamily: 'Source Han Sans SC, PingFang SC, Microsoft YaHei, Helvetica Neue, sans-serif'
    fontSize: '20px'
    fontWeight: 700
    lineHeight: 1.4
    letterSpacing: '0'
  title:
    fontFamily: 'Source Han Sans SC, PingFang SC, Microsoft YaHei, Helvetica Neue, sans-serif'
    fontSize: '16px'
    fontWeight: 700
    lineHeight: 1.5
    letterSpacing: '0'
  body:
    fontFamily: 'Source Han Sans SC, PingFang SC, Microsoft YaHei, Helvetica Neue, sans-serif'
    fontSize: '14px'
    fontWeight: 400
    lineHeight: 1.5
    letterSpacing: '0'
  label:
    fontFamily: 'Source Han Sans SC, PingFang SC, Microsoft YaHei, Helvetica Neue, sans-serif'
    fontSize: '13px'
    fontWeight: 500
    lineHeight: 1.4
    letterSpacing: '0'
  mono:
    fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, Liberation Mono, Courier New, monospace'
    fontSize: '12px'
    fontWeight: 400
    lineHeight: 1.4
    letterSpacing: '0'
rounded:
  sm: '8px'
  md: '12px'
  lg: '18px'
  xl: '24px'
spacing:
  xs: '4px'
  sm: '8px'
  md: '12px'
  lg: '16px'
  xl: '24px'
components:
  icon-button:
    backgroundColor: '{colors.surface}'
    textColor: '{colors.text-secondary}'
    rounded: '{rounded.md}'
    width: '32px'
    height: '32px'
  button-primary:
    backgroundColor: '{colors.primary}'
    textColor: '{colors.surface}'
    rounded: '{rounded.md}'
    padding: '0 12px'
    height: '32px'
  card:
    backgroundColor: '{colors.surface}'
    textColor: '{colors.text-primary}'
    rounded: '{rounded.md}'
    padding: '14px'
  input:
    backgroundColor: '{colors.surface-muted}'
    textColor: '{colors.text-primary}'
    rounded: '{rounded.md}'
    padding: '6px 12px'
    height: '32px'
---

# Design System: InduForge

## 1. Overview

**Creative North Star: "可信工业工作台"**

InduForge 的界面是产品型工作台，不是展示型网站。视觉系统服务于平台管理、工程设计、数据接入、发布部署和节点运维等连续任务，因此默认采用克制的浅色工作台、清晰的状态表达、稳定的组件结构和适中的信息密度。

系统应像可靠的工业控制台：专业、安静、可扫描。蓝色是主要操作和选中语言，青色只作为少量辅助强调；中性色承担大部分界面面积。卡片、表格、筛选、分页和弹窗都应使用熟悉的产品工具模式，让用户专注于工程状态和操作结果。

**Key Characteristics:**

- 专业克制：低饱和中性色承载界面，主色只用于操作和状态。
- 工具感明确：导航、工具栏、筛选、表格、卡片和弹窗使用熟悉结构。
- 信息密度适中：一屏内承载足够多状态，长文本可截断，不牺牲扫描效率。
- 轻量层次：通过边框、弱背景、半透明面板和细阴影建立层级。
- 宿主一致：`dev_ide`、`designer`、`datacenter` 共享工作台视觉语言。

## 2. Colors

InduForge 使用“克制中性色 + 单主色 + 少量辅助强调”的产品色彩策略。颜色表达任务和状态，不做装饰铺陈。

### Primary

- **Workbench Blue** (`primary`): 用于主操作、当前选中、链接、聚焦边框和关键入口。
- **Workbench Blue Hover** (`primary-hover`): 用于主操作 hover 和激活反馈。

### Secondary

- **Industrial Teal** (`accent`): 用于主渐变的第二端和少量辅助强调，不能成为大面积主题色。

### Tertiary

- **Operational Green** (`success`): 成功、运行中、可用。
- **Review Amber** (`warning`): 部署中、待处理、需注意。
- **Failure Red** (`danger`): 删除、错误、失败和危险操作。

### Neutral

- **Workbench Canvas** (`background`): 工作台页面底色。
- **Panel Surface** (`surface`): 面板、表格、按钮和普通卡片底色。
- **Muted Control Surface** (`surface-muted`): 输入框、弱容器、编号块和低权重区域。
- **Primary Ink** (`text-primary`): 标题、项目名、关键数据。
- **Secondary Ink** (`text-secondary`): 正文、描述、说明。
- **Muted Ink** (`text-muted`): 时间、占位、辅助标签。
- **Soft Border** (`border`): 标准边框和输入框边界。

### Named Rules

**The Task Color Rule.** 颜色只用于表达任务优先级、选择和状态，禁止把主色、危险色或渐变当作普通装饰。

**The One Host Rule.** 子应用必须继承宿主主题，不允许 `designer`、`datacenter` 各自发展一套冲突的全局配色。

## 3. Typography

**Display Font:** Source Han Sans SC / PingFang SC / Microsoft YaHei  
**Body Font:** Source Han Sans SC / PingFang SC / Microsoft YaHei  
**Label/Mono Font:** ui-monospace / SFMono-Regular / Menlo / Consolas

**Character:** 字体系统应保持中文产品工具的清晰和稳定。常规界面使用单一无衬线族，时间、版本号、日志和技术标识使用等宽字体增强扫描效率。

### Hierarchy

- **Headline** (700, 20px, 1.4): 仪表盘欢迎语、页面重点标题。
- **Title** (700, 16px, 1.5): 工具栏标题、面板标题。
- **Body** (400, 14px, 1.5): 常规正文、描述、表格内容。
- **Label** (500, 13px, 1.4): 筛选、按钮、列表项、控件文案。
- **Meta** (400, 11-12px, 1.4): 标签、更新时间、状态辅助文本。
- **Mono** (400, 10-12px, 1.4): 日志时间、版本号、技术标识。

### Named Rules

**The Product Type Rule.** 产品界面不使用展示字体、负字距或随视口变化的流式字号；层级来自字号、字重、颜色和位置。

## 4. Elevation

InduForge 使用轻量层次，而不是厚重阴影。默认界面通过边框、弱背景和间距表达结构；阴影只用于工具栏、浮层、hover 卡片和高优先级弹出内容。

### Shadow Vocabulary

- **Surface Shadow** (`0 2px 6px rgba(15, 23, 42, 0.08)`): 工具栏、普通按钮、轻面板。
- **Popover Shadow** (`0 12px 24px rgba(15, 23, 42, 0.12)`): 筛选面板、下拉浮层。
- **High Overlay Shadow** (`0 20px 40px rgba(15, 23, 42, 0.18)`): 高优先级弹层。
- **Card Hover Lift** (`0 8px 18px rgba(15, 23, 42, 0.10)`): 卡片 hover 状态。

### Named Rules

**The Flat Until Needed Rule.** 静态内容默认轻边框和弱背景，只有 hover、focus、弹出和主操作可以获得更明显的抬升。

## 5. Components

### Buttons

- **Shape:** 轻微圆角，图标按钮和普通按钮使用中圆角 (`12px`)，表格操作按钮可使用 `6-8px`。
- **Primary:** 主操作使用蓝色或主渐变，尺寸通常为 `32px` 高。每个工具栏只保留一个最强主操作。
- **Hover / Focus:** hover 可轻微上移 `1px` 或改变背景；focus 使用主色边框和弱主色 ring。
- **Icon Buttons:** 高频工具按钮使用 `32px * 32px`，必须有 tooltip 或 `aria-label`；表格操作列使用 Icon Action Button 的 `24px * 24px` 行内小尺寸变体。
- **Danger:** 删除和危险操作使用危险色 hover，不使用常态大面积红底。

### Chips

- **Style:** 运行节点、标签和状态说明使用小尺寸 chip 或组件库 tag，文字通常为 `11-12px`。
- **State:** 状态 chip 必须配合文本，不能只依赖颜色。

### Cards / Containers

- **Corner Style:** 普通卡片使用 `12px`，主内容区和工具栏使用 `18px`。
- **Background:** 工作台内容面板使用半透明白或深色面板，普通业务卡片使用白底或深蓝灰底。
- **Shadow Strategy:** 默认轻阴影，hover 才允许更明显阴影和 `translateY(-2px)` 到 `translateY(-3px)`。
- **Border:** 使用低透明边框，选中态使用主色边框。
- **Internal Padding:** 项目卡片约 `14px`，内容滚动区约 `16px`。

### Inputs / Fields

- **Style:** 输入框使用弱背景、标准边框、中圆角和 `32px` 左右高度。
- **Focus:** 聚焦时使用主色边框和 `0 0 0 3px` 主色弱 ring。
- **Error / Disabled:** 错误使用危险色和明确文案，禁用态降低透明度并禁止 pointer。

### Navigation

- **Sidebar:** 宿主侧边栏宽度约 `240px`，菜单项高度 `40px`，激活态使用蓝色弱底和蓝色文本。
- **Tabs:** 标签页高度约 `36px`，激活态使用弱蓝底。子应用嵌入时不重复全局导航。
- **Mobile:** 窄屏下侧边栏以抽屉形式出现，点击菜单后收起。

### Workbench Toolbar

工作台工具栏是页面任务控制中心。左侧放搜索、视图切换、筛选和排序；右侧放新增、刷新、导入、设置和批量操作。常态高度紧凑，允许窄屏换行。

### Project Card

项目卡片是 `dev_ide` 的标志性工作台组件。结构包含项目名、可见性、描述、运行模式、部署状态、节点 chip、标签、更新时间和操作按钮。卡片最小高度约 `200px`，长文本必须截断。

### Pill Button

- **Style:** 数据中心 v2 的药丸筛选 / 排序按钮使用圆角 `12px`、高度 `28px`、padding `0 10px`、字体 `13px`。数据中心 v2 子应用以 `--dc-*` 局部 token 为准。
- **State:** 支持 default / hover / active。active 使用 `--dc-primary-soft` 弱底和主色文字，hover 使用弱底或边框增强。
- **Size:** 可选前置 `14px` icon + 文本，图标和文字保持紧凑间距。

### Status Badge

- **Style:** 状态徽标用于 success / warning / danger / info / muted 五类状态，必须使用文本 + 颜色双重表达，禁止仅颜色。
- **State:** 不同状态使用对应语义色和弱背景，禁用或未知状态使用 muted。
- **Size:** 高度 `22px`，圆角 `6px`，字体 `12px`，内容保持单行。

### Icon Action Button

- **Style:** 行内操作按钮用于表格操作列，作为工具栏 `32px * 32px` 图标按钮的紧凑变体。
- **State:** 支持 default / hover / disabled / danger。危险操作仅在 hover 或确认流程中强化危险色。
- **Size:** 行内小尺寸为 `24px * 24px`，图标通常为 `14px`，必须有 tooltip 或 `aria-label`。

### Drawer

- **Style:** 数据中心 v2 抽屉默认宽度 `480px`，最大 `720px`，支持拖拽调整宽度。数据中心 v2 子应用以 `--dc-*` 局部 token 为准。
- **Header:** 头部包含 title 字号标题、操作图标群和关闭按钮，标题与操作区保持清晰分组。
- **Content:** 内容区可滚动，padding `16px`。优先单页滚动，Tab 仅在内容分量大且互斥时使用。
- **State:** 钉住模式去掉边阴影，嵌入主区右侧成为第三栏，不再表现为浮层。

### Bulk Action Bar

- **Style:** 批量操作浮动条在列表选中 `>= 1` 行时出现，浮在底部分页栏上方。
- **Content:** 内容包含选中计数、主要批量动作和清空选择。
- **State:** 无选中项时隐藏；批量动作需要按权限、加载和禁用状态给出明确反馈。

### Link Chip

- **Style:** 跨模块跳转芯片使用圆角 `8px`、高度 `24px`、字体 `12px`，用于关联对象的轻量入口。
- **Content:** 内部包含模块图标、对象名和跳转箭头，长对象名需要截断。
- **State:** hover 增加弱底色；点击后切换模块并打开目标对象详情。

### Empty State

- **Style:** 空状态基于 `el-empty` 包装，保持文案简洁，并可附带一个辅助动作。
- **Content:** 文案说明当前为空的业务原因；辅助动作通常为创建或刷新。
- **State:** 权限不足、能力未启用和筛选无结果应使用不同文案，不混用通用空状态。

### Loading State

- **Style:** 列表加载使用 `v-loading`，详情区域使用骨架屏，骨架屏按需引入。
- **State:** 加载态不应清空已有上下文；刷新列表时保持分页和筛选位置稳定。
- **Size:** 骨架屏尺寸应贴合详情布局，避免加载完成后产生明显跳动。

## 6. Do's and Don'ts

### Do:

- **Do** 直接进入可操作工作台，首屏放任务控件和业务内容。
- **Do** 使用 `--ck-*` 工作台 token 作为新管理界面的主视觉来源。
- **Do** 保持工具栏、内容区、分页区三段式骨架。
- **Do** 用文本、标签或图标配合颜色表达状态。
- **Do** 为加载、空、错误、禁用、权限不足和批量选择状态补齐界面反馈。
- **Do** 在 `designer` 和 `datacenter` 中继承宿主主题、语言和工作台控件语言。

### Don't:

- **Don't** 做营销型 SaaS 落地页式首屏，禁止大面积 hero、装饰插画和空泛价值主张。
- **Don't** 做炫技型暗色大屏，禁止霓虹、过饱和渐变、发光边框和无意义动效。
- **Don't** 让子应用各自建立冲突的全局视觉语言。
- **Don't** 把每个业务模块都做成同尺寸卡片网格，表格、双栏、画布和面板应按任务选择。
- **Don't** 使用颜色作为唯一状态表达。
- **Don't** 为普通信息块添加夸张阴影、玻璃模糊或过大圆角。
