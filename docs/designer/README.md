# 设计中心 (Designer) 文档

## 概述

设计中心是 InduForge 平台的核心模块，提供可视化的页面设计功能。通过拖拽式操作，用户可以快速构建工业应用界面，无需编写代码。

> **🚀 重构计划**：设计中心正在重构，详见 [Designer 重构计划](./refactor/README.md)

## 核心功能

### 1. 可视化设计

- 拖拽式组件布局
- 实时预览
- 所见即所得
- 多种画布缩放模式

### 2. 组件库

- **基础组件**: 按钮、视频、图片、开关、多选组、下拉框、单选组
- **UI 组件**: 输入框、表格、图表、标签、对话框等
- **布局组件**: Container、Flex、Grid、Row/Col、CenterLayout
- **自定义组件**: 支持组件封装和复用

### 2.5 绘图工具 ✨ 新增

- **基础图形**: 直线、矩形、圆形、椭圆、多边形、路径
- **管道**: 带流动动画的管道，支持流速/方向绑定
- **符号库**: 预设工业符号（泵、阀门、储罐、换热器等）
- **文字标注**: Canvas 层文字，支持数据绑定

### 3. 数据绑定

- 表达式绑定
- 实时数据订阅
- 多数据源支持
- 数据转换和计算
- 直接调用数据中心 API（预览/运行时）

### 4. 选中与辅助线

- 多选支持（单选/多选/框选/全选）
- 选择框和控制点（缩放、旋转）
- 智能对齐辅助线
- 自动吸附功能

### 5. 历史记录

- 撤销/重做（Ctrl+Z/Y）
- 操作历史追踪（最多 50 条）
- 事务支持（批量操作合并）

### 6. 高级功能

- 画布缩放（10%-500%，Ctrl+滚轮/Plus/Minus/0）
- 完整的快捷键系统
- 批量操作（移动、删除、复制）

### 7. 多人开发支持 ✨ 新增

- **页面级编辑锁**：同一页面同时只能有一人编辑，防止冲突
- **锁冲突提示**：尝试编辑已锁定页面时显示当前编辑者
- **只读模式**：无法获取锁时自动进入只读模式
- **锁超时释放**：30 分钟无操作或断连自动释放
- **在线状态感知**：页面树显示其他人正在编辑的页面

> 详细设计见 [编辑器内核 - 页面编辑锁](./refactor/editor-core.md#9-页面编辑锁pagelockmanager)

## 技术架构

### 混合渲染架构 (Canvas + DOM 双层) ✨ 核心创新

设计中心采用创新的混合渲染架构，完美结合 Canvas 和 DOM 的优势：

- **Canvas Layer (z-index: 1)**: 绘制 2D 工艺流程图
  - 基础图形（线段、矩形、圆形、多边形）
  - 管道（带流动动画）
  - 工业符号（泵、阀门、储罐等）
  - 文字标注
  - 高性能绑定大量图形
- **DOM Layer (z-index: 10)**: 渲染交互组件
  - Vue 组件 + CSS 原生布局（Flexbox/Grid）
  - 完整的事件支持和交互
  - 表单、表格、图表等复杂组件
- **Overlay Layer (z-index: 100)**: 设计态辅助
  - 选择框和控制点（缩放、旋转）
  - 对齐辅助线和插入线
  - 拖拽预览和框选矩形

**统一数据绑定**：无论 Canvas 图形还是 DOM 组件，都支持相同的数据绑定、事件配置和动画系统。

```
Designer/
├── engine/                      # 核心引擎
│   ├── canvas/                 # Canvas 辅助渲染（Konva）
│   │   ├── SelectionBox.js         # 选择框和控制点
│   │   ├── AlignmentGuides.js      # 对齐辅助线和吸附
│   │   ├── SelectionRect.js        # 框选矩形
│   │   ├── InsertLine.js           # 插入线
│   │   └── DragPreview.js          # 拖拽预览
│   ├── binding/                # 数据绑定引擎
│   ├── animation/              # 动画引擎（GSAP）
│   └── datasource/             # 数据源管理
│
├── registry/                    # 组件注册系统
│   ├── ComponentFactory.js         # 组件工厂（核心）
│   ├── layout/                     # 布局组件（DOM）
│   │   ├── Container.vue              # 容器（Flex/Grid/Block）
│   │   ├── Row.vue / Col.vue          # 24栅格系统
│   │   ├── Flex.vue                   # Flexbox布局
│   │   ├── Grid.vue                   # Grid布局
│   │   └── CenterLayout.vue           # 居中布局
│   ├── basic/                      # 基础组件
│   ├── ui/                         # UI组件
│   └── charts/                     # 图表组件
│
├── components/                  # Vue 组件
│   ├── canvas/                     # 画布组件
│   │   ├── DesignCanvas.vue           # 主画布（混合架构核心）
│   │   ├── DomRenderer.vue            # DOM层递归渲染器
│   │   ├── CanvasAuxiliary.vue        # Canvas层辅助功能
│   │   └── ComponentWrapper.vue       # 组件包装器（拖拽/选中）
│   ├── panels/                     # 面板组件
│   │   ├── PropertyPanel.vue          # 属性面板（支持多选）
│   │   ├── ComponentLibrary.vue       # 组件库
│   │   └── ComponentTree.vue          # 组件树
│   └── editors/                    # 属性编辑器
│       ├── PositionEditor.vue         # 位置与尺寸
│       ├── SpacingEditor.vue          # 间距（Margin/Padding）
│       ├── TransformEditor.vue        # 变换（旋转/缩放/透明度）
│       ├── FlexEditor.vue             # Flexbox属性
│       ├── GridEditor.vue             # Grid属性
│       └── TextComponentEditor.vue    # 文本组件属性
│
├── composables/                 # 组合式函数
│   ├── useCoordinateSync.js        # 坐标系统同步
│   ├── useCanvas.js                # Canvas状态管理
│   ├── useZoom.js                  # 缩放功能（10%-500%）
│   ├── useHistory.js               # 历史记录（撤销/重做）
│   └── useDragDrop.js              # 拖拽功能
│
├── utils/                       # 工具函数
│   ├── styleConverter.js           # DSL样式→CSS转换
│   └── dropZoneCalculator.js       # 插入位置计算
│
├── store/                       # 状态管理（Pinia）
│   └── design.js                   # 设计画布状态
│
└── views/                       # 页面视图
    └── DesignCenter.vue            # 设计中心主页面
```

### 架构优势

1. **性能优化**: DOM 渲染组件，Canvas 渲染辅助图形，各司其职
   - DOM Layer: 利用浏览器原生渲染和事件处理
   - Canvas Layer: 高性能绘制辅助图形（Konva.js）
2. **原生布局**: 使用 CSS Flexbox/Grid，布局更精确可靠
   - 支持任意嵌套的容器结构
   - 完整的响应式布局能力
   - 无需手动计算布局位置
3. **易于维护**: Vue 组件开发，代码结构清晰
   - 组件化架构，每个组件职责单一
   - 完整的 TypeScript 类型支持（可选）
   - 测试覆盖率高（Vitest + Property-based Testing）
4. **坐标同步**: 自动同步 DOM 和 Canvas 坐标系统

   - ResizeObserver 监听尺寸变化
   - 实时更新选择框和辅助线位置
   - 支持缩放和滚动同步

5. **用户体验**: 完整的交互功能
   - 拖拽系统（库 → 画布、画布内、容器内、排序）
   - 多选支持（单选/Ctrl+点击/框选/Ctrl+A）
   - 撤销/重做（Ctrl+Z/Y）
   - 画布缩放（Ctrl+滚轮/Plus/Minus/0）
   - 智能对齐和自动吸附

## 快速开始

### 启动设计中心

```bash
cd InduForge/designer
pnpm install
pnpm dev
```

访问: http://localhost:9093/designer/

### 创建第一个页面

1. 点击"新建页面"
2. 输入页面名称
3. 从组件库拖拽组件到画布
4. 配置组件属性
5. 保存页面

### 快捷键列表 ⌨️

**选择操作**：

- `单击` - 选中组件
- `Ctrl + 单击` - 多选/取消选中
- `Ctrl + A` - 全选
- `Esc` - 取消选择

**编辑操作**：

- `Ctrl + C` - 复制
- `Ctrl + V` - 粘贴
- `Ctrl + D` - 复制并粘贴
- `Delete / Backspace` - 删除

**历史操作**：

- `Ctrl + Z` - 撤销
- `Ctrl + Y` - 重做
- `Ctrl + Shift + Z` - 重做（Mac 风格）

**缩放操作**：

- `Ctrl + 滚轮` - 以鼠标位置缩放画布
- `Ctrl + Plus` - 放大
- `Ctrl + Minus` - 缩小
- `Ctrl + 0` - 重置缩放（100%）

**拖拽操作**：

- `拖拽组件` - 从组件库拖到画布
- `拖拽画布内组件` - 移动位置或改变父容器
- `拖拽到容器` - 添加为容器子组件

## 文档导航

### 🚀 重构文档（当前设计）

**核心架构**：

- **[重构计划](./refactor/README.md)** - 完整重构计划与里程碑
- **[编辑器内核](./refactor/editor-core.md)** - DocumentModel、Command、History、Selection
- **[Schema 设计](./refactor/schema-design.md)** - 规范化工程 Schema（v2）、循环、插槽、生命周期

**数据系统**：

- **[数据绑定 v2](./refactor/data-binding-v2.md)** - 三态隔离、Binding 结构、数据点状态
- **[表达式引擎](./refactor/expression-engine.md)** - 上下文变量、内置函数、工业计算

**动作与动画**：

- **[动作系统](./refactor/action-system.md)** - 完整动作类型、条件分支、循环、并行
- **[动画系统](./refactor/animation-system.md)** - 状态驱动动画、工业场景动画
- **[验证系统](./refactor/validation-system.md)** - 表单验证规则、异步验证

**布局与渲染**：

- **[布局系统](./refactor/layout-system.md)** - Flex/Free/Grid、Constraints 约束
- **[设计态交互](./refactor/design-interaction.md)** - 工具栏、属性面板、预览功能

**发布与运行**：

- **[发布流水线](./refactor/publish-pipeline.md)** - 校验、编译、打包、快照
- **[运行时引擎](./refactor/runtime-engine.md)** - DataService、资源生命周期、Watchdog

**国际化与主题**：

- **[国际化与主题](./refactor/i18n-theme.md)** - i18n 资源、主题系统、切换组件

**开发指南**：

- **[最佳实践](./refactor/best-practices.md)** - 性能优化、安全建议、命名规范

### 历史文档

- **[组件开发指南](./component-development.md)** - 组件开发参考
- **[开发历程](./development-history.md)** - 各阶段开发记录

### 开发历程

- **[开发历程](./development-history.md)** - 各阶段开发总结

## 核心概念

### DSL (Domain Specific Language)

设计中心使用 JSON 格式的 DSL 来描述页面结构：

```json
{
  "meta": {
    "id": "page_001",
    "name": "监控大屏"
  },
  "config": {
    "width": 1920,
    "height": 1080
  },
  "variables": {
    "deviceId": { "type": "string", "default": "" }
  },
  "dataSources": [
    {
      "id": "ds_device",
      "type": "dataCenter",
      "config": {
        "sourceType": "query",
        "queryId": "query_device_status"
      }
    }
  ],
  "components": [
    {
      "id": "comp_001",
      "type": "Text",
      "bindings": {
        "props.content": "{{ data.ds_device.name }}"
      }
    }
  ]
}
```

详细的 DSL 规范请参考 [DSL 设计规范](../dsl-design.md)。

### 组件系统

组件是设计中心的基本单元，每个组件包含：

- **类型 (type)**: 组件的类型标识
- **属性 (props)**: 组件的配置属性
- **样式 (style)**: 组件的样式定义
- **绑定 (bindings)**: 数据绑定配置
- **事件 (events)**: 事件处理配置

### 数据源

数据源提供组件所需的数据，支持多种类型：

- **dataCenter**: 数据中心查询/点位订阅
- **http**: REST API 调用
- **static**: 静态数据
- **computed**: 计算数据源

### 表达式系统

使用 `{{ expression }}` 语法进行数据绑定：

```javascript
// 访问数据源
{
  {
    data.ds_device.temperature;
  }
}

// 条件表达式
{
  {
    data.ds_device.status == 1 ? "运行" : "停止";
  }
}

// 函数调用
{
  {
    format(data.ds_device.temperature, 1);
  }
}
```

## 开发指南

### 添加自定义组件

1. 在 `registry/components/` 下创建组件定义
2. 定义组件的 Schema
3. 注册组件
4. 在组件库中使用

示例：

```javascript
// registry/components/custom/MyComponent.js
export default {
  type: "MyComponent",
  name: "我的组件",
  category: "自定义",
  defaultProps: {
    title: "Hello",
  },
  propsSchema: {
    title: {
      type: "text",
      label: "标题",
    },
  },
};
```

### 配置数据源

1. 打开数据源面板
2. 点击"添加数据源"
3. 选择数据源类型
4. 配置数据源参数
5. 保存

### 绑定数据到组件

1. 选择组件
2. 打开数据绑定面板
3. 添加绑定
4. 输入表达式
5. 实时预览

## 常见问题

### Q: 如何调试数据绑定？

A: 在数据绑定面板中查看实时预览值，或在浏览器控制台查看 `store.dataSources`。

### Q: 组件不显示怎么办？

A: 检查组件的样式配置，确保位置和大小正确。

### Q: 数据源状态一直是"加载中"？

A: 检查后端 API 是否正常，查询 ID 是否存在。

### Q: 如何提高性能？

A: 减少轮询频率，使用数据转换器减少数据量，优化表达式。

更多问题请参考 [数据绑定 v2](./refactor/data-binding-v2.md) 或 [最佳实践](./refactor/best-practices.md)。

## 相关资源

- [DSL 设计规范](../dsl-design.md)
- [数据中心文档](../datacenter/README.md)
- [后端 API 文档](../backend/README.md)
- [Konva.js 文档](https://konvajs.org/)
- [Vue 3 文档](https://vuejs.org/)

## 贡献指南

欢迎贡献代码和文档！请遵循：

1. 代码规范: ESLint + Prettier
2. 提交规范: 语义化提交信息
3. 测试: 添加单元测试
4. 文档: 更新相关文档

## 最新更新 (2026-01-06)

### 🚀 重构计划发布

设计中心正在进行大规模重构，采用全新架构：

- **编辑器内核**：DocumentModel + Command/History 命令系统
- **三态数据隔离**：设计态/预览态/运行态数据分离
- **多端适配**：支持 PC/BigScreen/Mobile 多视图
- **发布流水线**：完整的校验、编译、打包、快照流程

详见 [Designer 重构计划](./refactor/README.md)

### 历史版本

✅ **Phase 1-3**：混合渲染架构（DOM + Canvas 双层）

- 实现 DesignCanvas、DomRenderer、CanvasAuxiliary 组件
- 实现 Container、Flex、Grid、Row/Col 布局组件
- 实现 ComponentFactory 组件工厂

### 重构后待实现

详见 [重构计划 - 里程碑](./refactor/README.md#重构阶段)

---

**版本**: 3.0.0-alpha  
**最后更新**: 2026-01-06
