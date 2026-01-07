# 设计中心开发历程

本文档记录了设计中心各个阶段的开发历程和成果。

> **重要**：2026 年 1 月起，设计中心进入**重构阶段**，采用全新架构。
> 详见 [重构计划](./refactor/README.md)。

---

## 🔄 重构阶段（2026 年 1 月 - 进行中）

### 重构目标

- 采用 **编辑器内核 + 命令系统** 架构，支撑撤销重做、差量保存
- 实现 **三态数据隔离**（设计态/预览态/运行态）
- 支持 **多端适配**（PC/BigScreen/Tablet/Phone）
- 提供 **完整发布流水线**，输出可独立运行的工程制品（.ifp）
- 支持 **国际化** 和 **主题切换**

### 新架构文档

- [重构计划](./refactor/README.md) - 完整重构计划与里程碑
- [编辑器内核](./refactor/editor-core.md) - DocumentModel、Command、History
- [Schema 设计](./refactor/schema-design.md) - 规范化工程 Schema（v2）
- [数据绑定 v2](./refactor/data-binding-v2.md) - 三态隔离、数据点绑定
- [布局系统](./refactor/layout-system.md) - Flex/Free/Grid、Constraints
- [发布流水线](./refactor/publish-pipeline.md) - 校验、编译、打包、部署
- [运行时引擎](./refactor/runtime-engine.md) - DataService、Watchdog
- [国际化与主题](./refactor/i18n-theme.md) - i18n、主题切换组件

### 里程碑

| 里程碑 | 阶段         | 状态      |
| ------ | ------------ | --------- |
| M1     | 编辑器内核   | 🚧 进行中 |
| M2     | 数据绑定     | ⏳ 待开始 |
| M3     | 发布运行     | ⏳ 待开始 |
| M4     | 国际化与主题 | ⏳ 待开始 |

---

## 历史阶段（已归档）

> 以下为重构前的开发历程记录，部分实现可能已被新架构替代。

## 阶段一：基础设施（2 周）

### 完成时间

2025-11-24

### 核心成果

#### 1. 表达式引擎

- 支持 `{{ expression }}` 语法
- 内置函数：format、if、sum、avg、max、min 等
- 自定义函数注册
- 单元测试覆盖

#### 2. 数据绑定管理器

- 组件属性与数据源绑定
- 响应式数据更新
- 支持嵌套属性绑定
- 上下文构建

#### 3. 动画管理器

- 基于 GSAP 的动画系统
- 支持 8 种动画类型
- 条件触发支持
- 动画预设配置

#### 4. 历史记录系统

- 撤销/重做功能
- 快照模式
- 可配置历史记录大小

#### 5. 组件注册系统

- 组件分类索引
- 组件标签索引
- 组件搜索功能
- 4 个基础组件

### 技术亮点

- 模块化设计
- 可扩展架构
- 类型安全
- 性能优化

详细内容：内部记录未纳入仓库文档。

---

## 阶段二：Canvas 渲染引擎（3 周）

### 完成时间

2025-12-01

### 核心成果

#### 1. Konva 渲染引擎

- 基于 Konva.js 的高性能渲染
- 支持 6 种基础图形
- 组件渲染、更新、删除
- 样式应用和事件绑定

#### 2. 选择管理器

- 单选和多选
- 框选功能
- Transformer 集成
- 对齐和分布功能

#### 3. 辅助线管理器

- 实时对齐检测
- 智能吸附
- 可视化辅助线
- 可配置吸附阈值

#### 4. DesignCanvas 组件重构

- 集成 Konva 渲染引擎
- 响应式数据绑定
- 自动渲染组件
- 历史记录集成

### 技术亮点

- 高性能 Canvas 渲染
- 直观的交互体验
- 模块化架构设计
- 易于扩展和维护

详细内容：内部记录未纳入仓库文档。

---

## 阶段三：组件库扩展（计划中）

### 预计时间

3 周

### 计划内容

#### Week 1: UI 组件（10 个）

- Button、Input、Select、Switch、Slider
- Progress、Badge、Tag、Tooltip、Alert

#### Week 2: 数据展示组件（10 个）

- Table、Tree、Tabs、Collapse、Timeline
- Card、List、Descriptions、Empty、Result

#### Week 3: 图表组件（10 个）

- LineChart、BarChart、PieChart、GaugeChart、RadarChart
- ScatterChart、HeatmapChart、TreeChart、SankeyChart、FunnelChart

---

## 阶段四：数据绑定系统（4 周）

### 完成时间

2025-12-08

### 核心成果

#### 1. DataCenter 通信桥接

- iframe postMessage 通信机制
- 握手协议和就绪检测
- 请求-响应模式
- 订阅-推送模式

#### 2. 数据源管理器

- 支持 4 种数据源类型
- 3 种数据获取模式
- 数据转换器和错误处理器
- 响应式数据缓存

#### 3. API 模式支持

- 直接调用后端 API
- 不依赖 DataCenter 标签页
- 适用于标签页架构
- 自动降级机制

#### 4. 可视化配置面板

- 数据源配置面板
- 数据绑定配置面板
- 实时状态监控
- 错误提示和降级

### 技术亮点

- 单模式支持（API）
- 完全解耦的架构
- 可视化配置界面
- 完善的文档体系

详细内容：内部记录未纳入仓库文档。

---

## 阶段更新：组件渲染可视化与交互修复

### 完成时间

2025-12-15

### 核心成果

- 组件渲染落地：为 UI / 基础 / 图表组件补充实际 Vue 渲染实现并在 registry 绑定，拖入画布即可看到 Element Plus 控件、图片占位与图表占位。
- 样式隔离：新增 `styleHelpers` 去除定位相关样式，避免与 ComponentWrapper 的定位叠加；新增图标解析 helper 支持 Element Plus Icons。
- 交互修复：组件点击切换到捕获阶段，避免子组件阻止冒泡导致无法选中；移除 canvas-layer 强制 z-index，减少遮挡问题。
- 拖拽修复：容器自身拖动时不再被子级 drop 逻辑拦截，画布可正常移动已有组件。
- 拖拽兼容：画布内拖动组件同时写入 `application/json` 数据，DesignCanvas 可统一解析拖拽来源并更新位置。
- 拖拽判定优化：拖拽负载包含 `source` 字段时优先作为画布内移动处理，避免误判为新增组件。
- 拖拽兜底：当 dataTransfer 为空但已有选中组件时，按照指针位置移动选中组件，避免浏览器丢失拖拽数据导致无法拖动。
- 拖拽 MIME 兼容：画布内拖动同时写入 `text/plain`，DesignCanvas 增加 text/plain 读取兜底，提升跨浏览器稳定性。
- 拖拽光标：Canvas dragenter/dragover 设置 dropEffect 为 move，避免出现禁用光标。
- 拖拽落点兜底：监听子组件 dragend，按指针位置更新组件样式，即便 drop 事件丢失也能完成拖动。
- 容器移动适配：画布内拖动容器时将百分比宽度转换为像素宽度，确保落点与指针一致。
- 拖拽偏移：拖拽时记录指针相对组件左上角的偏移，在落点时扣除偏移，防止组件跟随鼠标时“乱跑”。
- 拖拽体验：拖拽开始停止事件冒泡、强制 dropEffect=move，并禁用组件内文字选择，避免局部区域无法触发拖拽。
- 定位精度：ComponentWrapper 计算样式时不再将 0 判定为 falsy，确保左上角坐标与宽高为 0/数字时正常应用。
- 拖拽事件捕获：dragstart/drag/dragend 改为捕获阶段并阻止默认，确保嵌套容器内任意区域都能触发拖拽。
- 回退阻止默认：dragstart/drag/dragend 不再 prevent default，仅在捕获阶段 stop 事件，恢复浏览器拖拽流程。
- 容器放置提示：dragover 显式设置 dropEffect（copy/move），避免拖入容器时被误判为禁止。
- 组件进容器：支持画布内已存在组件移动到容器，移动前重置为相对定位并调用 moveComponent 追加到容器末尾。
- 组件进容器优化：画布内组件入容器改为“删除旧节点 + 复制为相对定位再插入”，防止原节点引用导致移动失败。
- 组件进容器回退：改用 updateComponent 调整为相对定位 + moveComponent 直接移动原节点，避免复制导致的状态丢失。
- 组件进容器兜底：画布内移动缺少 id 时回退使用当前选中组件，提升拖入容器的鲁棒性。
- 组件进容器稳定化：画布内移动改为“查找原组件 → 改相对定位 →remove→add”路径，确保真正插入目标容器。
- 拖拽定位容器：记录 hover 容器 id，canvas drop/dragend 兜底移入 hover 容器。
- 组件进容器回滚：再次改为相对定位 + moveComponent 路径，避免重建节点导致状态丢失。
- 组件进容器再次回滚：克隆为相对定位、删除原节点再 add，确保一定写入容器末尾。
- 组件进容器再优化：新增 detachComponentById，先拆下原节点再相对定位 add 到容器，保证引用正确。
- 按钮样式配置校验：按钮属性面板新增样式配置对话框，支持 key:value 逐行解析并校验格式（缺冒号/空属性提示错误），样式写入 styleConfig 供渲染层生效。
- 样式配置编辑器：按钮样式配置弹窗嵌入 Monaco Editor（CSS 高亮/补全/折叠/校验），引入官方样式并配置 CSS 校验；监听系统/IDE 深浅色自动切换 `vs/vs-dark`；保存前按 key:value 规则校验，错误提示并阻断提交，合规后写入 styleConfig 应用于按钮。
- 容器兜底落点：DesignCanvas 在 hover 容器时直接按容器逻辑落点处理（即使容器 drop 未触发），并统一相对定位后 moveComponent，修复画布内组件拖入容器异常。

### 影响范围

- 前端：`designer/src/components/canvas/*`、`designer/src/registry/ui/*`、`designer/src/registry/basic/Image.*`、`designer/src/registry/charts/*`。
- 文档：补充本阶段更新记录。
- [数据绑定系统](./data-binding.md)
- [数据绑定架构](./data-binding-architecture.md)

---

## 后续规划

### 阶段五：工业图形库（4 周）

- 迁移 KingPortal 的 5000+ 工业图元
- 图元数据转换
- 图形组件渲染
- 图形库面板

### 阶段六：运行时引擎（4 周）

- 页面渲染引擎
- 事件系统
- 动作执行器
- 权限控制

### 阶段七：开发工具（2 周）

- 调试面板
- 数据监控
- 性能分析
- 错误追踪

---

## 技术演进

### 架构演进

**阶段一**: 基础设施

```
Vue 3 + Pinia + Element Plus
  ↓
表达式引擎 + 数据绑定 + 动画系统
```

**阶段二**: Canvas 渲染

```
基础设施
  ↓
Konva.js + 选择管理 + 辅助线
```

**阶段四**: 数据绑定

```
Canvas 渲染
  ↓
DataCenter 集成 + API 模式
```

### 技术栈演进

| 阶段   | 新增技术    | 用途        |
| ------ | ----------- | ----------- |
| 阶段一 | GSAP        | 动画系统    |
| 阶段一 | expr-eval   | 表达式解析  |
| 阶段二 | Konva.js    | Canvas 渲染 |
| 阶段四 | postMessage | 跨应用通信  |

### 代码统计

| 阶段     | 新增文件 | 代码行数  | 测试用例 |
| -------- | -------- | --------- | -------- |
| 阶段一   | 15+      | 2000+     | 45       |
| 阶段二   | 5        | 1500+     | -        |
| 阶段四   | 7        | 2000+     | -        |
| **总计** | **27+**  | **5500+** | **45+**  |

---

## 经验总结

### 成功经验

1. **模块化设计**: 清晰的职责划分，易于维护和扩展
2. **渐进式开发**: 分阶段实现，每个阶段都有可交付成果
3. **文档先行**: 完善的文档体系，降低学习成本
4. **测试驱动**: 核心功能有测试覆盖，保证质量

### 遇到的挑战

1. **性能优化**: 大量组件渲染时的性能问题
2. **跨应用通信**: 标签页架构下的通信机制
3. **架构调整**: 从 Bridge 模式到 API 模式的转变
4. **文档维护**: 保持文档与代码同步

### 改进方向

1. **增量渲染**: 优化组件更新机制
2. **类型定义**: 添加 TypeScript 类型
3. **单元测试**: 提高测试覆盖率
4. **性能监控**: 添加性能分析工具

---

## 相关资源

### 文档

- [DSL 设计规范](../dsl-design.md)
- [迁移计划](../migration-plan.md)
- [数据绑定系统](./data-binding.md)
- [Canvas 渲染引擎](./canvas-engine.md)

### 代码

- [表达式引擎](../../designer/src/engine/binding/ExpressionEngine.js)
- [数据绑定管理器](../../designer/src/engine/binding/DataBinder.js)
- [Konva 渲染器](../../designer/src/engine/canvas/KonvaRenderer.js)
- [数据源管理器](../../designer/src/engine/datasource/DataSourceManager.js)

---

**版本**: 2.0.0  
**最后更新**: 2025-12-26

## 更新记录

- 画布事件面板扩展：新增“变量改变/定时器”可添加条目，提供添加/删除按钮与“添加事件”弹窗，保存后按列表展示并支持配置脚本；弹窗表单与输入框比例同步调整并补充默认变量选项。
- 画布事件面板：未选中组件时在“连接”显示创建时/存在时/关闭时事件，脚本保存到页面 events；点击画布空白区清空选择以回到画布事件视图。
- Button 组件 DOM id 调整：实际渲染的 id 现在使用随机生成，避免多个按钮重复；样式配置中的 `domId` 通过 `data-dom-id` 作为选择器占位，继续可以使用 domId 来配置样式。
- 右侧面板 Tabs 合并为组件树/属性/样式/连接，PropertyPanel 通过 initialTab 与 showTabs 复用内容。
- 连接（事件）配置改为配置按钮 + 编辑器弹窗，支持在 Monaco 中编写组件事件脚本。
- Element Plus 组件全集接入：新增通用包装组件并批量注册官方组件（如 Link、Row/Col、Menu、Tabs、Dialog、Upload 等），支持直接拖入画布并通过 JSON 属性或示例 Slot 查看效果。
- 组件展示优化：Element Plus 组件在面板中统一使用中文名称；对话框/抽屉示例取消遮罩与 body 附着，避免拖入后遮挡画布；组件属性默认以 JSON 字符串存储，属性面板编辑更稳定。
- Element Plus 组件类型去重：新增组件的类型前缀统一改为 `El*`，避免与已有基础/布局组件（如 Row、Text、Link）冲突导致属性面板空白或错配。
- 修复动态组件渲染：Element Plus 自动包装使用 `resolveDynamicComponent`，避免在属性面板切换或选择组件时出现 `resolveComponent can only be used in render()` 警告并导致组件空白。
- Element Plus 表单类组件规范化（批次一）：为输入框/计数器/选择器/级联/时间日期/开关/滑块/取色器/评分/穿梭框/上传等补充独立属性表单（非 JSON），预设示例数据与完整事件元数据，保持与按钮/输入框一致的配置体验。
- Element Plus 表单类组件拆分（批次一）：去除通用包装，新增独立 `.vue` + `.js`（ElInput、ElInputNumber、ElSelect、ElCascader、ElTimePicker、ElTimeSelect、ElDatePicker、ElColorPicker、ElSwitch、ElSlider、ElRate、ElTransfer、ElUpload），按按钮规范注册到 UI registry，属性/事件面板与现有组件一致。
- Element Plus 布局/基础组件拆分（批次二部分）：新增行/列/容器/头部/侧边栏/主体/底部、间距、分割线、卡片、面包屑的独立 `.vue` + `.js`，并从自动注册中排除；UI registry 按按钮规范注册，避免重复注册和属性冲突。
- Element Plus 导航组件拆分（批次二继续）：新增选项卡、步骤条、菜单、下拉菜单、分页的独立 `.vue` + `.js`，注册到 UI registry 并从自动注册中过滤，属性/事件按官方映射并提供示例配置。
- Element Plus 反馈/展示/数据组件拆分（批次三）：新增徽章、头像、标签、提示、树、时间线、折叠面板、走马灯、描述列表、结果、骨架屏、图片、日历、单选组、多选组、气泡/文字提示、气泡确认、对话框、抽屉等的独立 `.vue` + `.js`，全部按按钮规范注册并从自动注册中过滤，提供示例数据和事件映射，属性面板为结构化表单。
- 属性面板 JSON 配置体验调整：JSON 类型属性由输入框改为“配置/已配置”按钮 + 弹窗编辑；组件属性标签移除“(JSON)”标识，避免冗余提示。
- 画布拖拽交互完善：组件渲染层补齐 dragstart/drag/dragend 透传，组件节点按锁定状态设置 draggable。
- 拖拽预览与落点对齐：记录鼠标相对组件偏移用于预览/落点计算，dragend 兜底仅在有效画布坐标时生效，drop 后不再二次移动。
- 光标与边框提示优化：拖拽时显示虚线轮廓，按 effectAllowed 自动设置 dropEffect（copy/move），避免出现“禁止”光标。
- 画布事件面板分组显示：变量改变/定时器按分组标题展示，分组标题样式与工具条一致（背景/边框/圆角/内边距）。
- 连接页事件按钮调整：移除“事件”标题行配置按钮，保留每个事件项配置按钮并统一最小宽度。
- 组件树同名区分：同名组件显示序号后缀（如 Button1、Button2），序号紧跟名称。
- 容器内拖拽修复：落点判定优先指针位置，避免误判锁回容器；内层组件拖拽不再被父容器拦截，可拖出到画布根级。
- 变量面板编辑修复：编辑弹窗回填变量信息，保存时更新原变量并支持改名同步（组件绑定/事件/数据源配置），历史记录按新增/编辑区分。
- 弹窗不再修改 body 宽度：对话框、抽屉、确认弹窗统一关闭滚动锁定，避免 iframe/画布宽度被滚动条补偿影响。
- 右侧面板 Tabs 细节优化：组件树/属性/样式/连接的 `el-tabs__item` padding 统一为 `0 17px`，变量标签单独增加 `margin-right: 10px`。
- 画布缩放交互升级：支持 Ctrl/Cmd + 滚轮缩放画布，缩放中心保持并与缩放比例显示同步。
- 组件缩放能力补齐：选中组件显示 8 个拖拽控制点，拖动可调整尺寸，支持最小尺寸限制并在结束时记录历史。
- Canvas 辅助层事件补齐：SelectionBox 透传 resize/rotate 回调，辅助层 resize 逻辑与历史记录完善。
