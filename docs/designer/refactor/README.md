# Designer 重构计划

本目录包含设计中心重构的详细技术文档，按阶段组织，可作为开发引导。

## 重构目标

- 采用 **编辑器内核 + 命令系统** 架构，支撑撤销重做、差量保存、协作
- 实现 **三态数据隔离**（设计态/预览态/运行态）
- 支持 **Canvas + DOM 双层架构**，绘制 2D 工艺流程图 + 交互组件
- 支持 **多端适配**（PC/BigScreen/Mobile）
- 提供 **完整发布流水线**，输出可独立运行的工程制品（.ifp）

> **关于发布与部署**：
>
> - **发布入口**：在 **dev_ide** 的工程卡片上操作（不在 Designer 内部）
> - **发布内容**：Designer Schema + DataCenter 配置（连接、查询、数据点）
> - **部署管理**：在 dev_ide 运维管理界面部署到节点
> - **运行时数据**：通过 ConnectionProfile 连接节点侧数据源
>
> 详见 [发布流水线](./publish-pipeline.md) 和 [运行时引擎](./runtime-engine.md)

## 重构阶段

### 第一阶段：编辑器内核（基础能力）

**预计周期**：2-3 周

| 模块            | 文档                                   | 验收标准                        |
| --------------- | -------------------------------------- | ------------------------------- |
| Schema v2       | [schema-design.md](./schema-design.md) | 规范化 pagesById/nodesById 结构 |
| DocumentModel   | [editor-core.md](./editor-core.md)     | CRUD 节点、索引查询             |
| Command/History | [editor-core.md](./editor-core.md)     | 撤销重做正常工作                |
| SelectionModel  | [editor-core.md](./editor-core.md)     | 单选/多选/hover                 |
| Serializer      | [editor-core.md](./editor-core.md)     | 导入导出、版本迁移              |

**里程碑 M1**：能拖组件到画布、撤销重做、保存加载

### 第二阶段：数据绑定（与 DataCenter 对接）

**预计周期**：2-3 周

| 模块           | 文档                                       | 验收标准                      |
| -------------- | ------------------------------------------ | ----------------------------- |
| Binding 结构   | [data-binding-v2.md](./data-binding-v2.md) | provider + datapointId + path |
| 数据点状态     | [data-binding-v2.md](./data-binding-v2.md) | 失效检测与诊断展示            |
| 预览态数据连接 | [data-binding-v2.md](./data-binding-v2.md) | 直接调用数据中心 API          |
| 变量系统       | [vars-system.md](./vars-system.md)         | 页面级/全局变量               |
| 表达式引擎     | [data-binding-v2.md](./data-binding-v2.md) | {{ }} 语法解析                |

**里程碑 M2**：能配置数据绑定、预览看到真实数据

### 第三阶段：布局与交互

**预计周期**：2-3 周

| 模块                   | 文档                                             | 验收标准             |
| ---------------------- | ------------------------------------------------ | -------------------- |
| FreeLayout Constraints | [layout-system.md](./layout-system.md)           | 约束布局运行时计算   |
| Canvas 绘图层          | [design-interaction.md](./design-interaction.md) | 绘制线/矩形/圆/管道  |
| 符号库                 | [schema-design.md](./schema-design.md)           | 预设工业符号拖入     |
| 空容器 Placeholder     | [design-interaction.md](./design-interaction.md) | 设计态占位符渲染     |
| 拖拽高亮               | [design-interaction.md](./design-interaction.md) | Drop target 视觉反馈 |
| 属性面板               | [design-interaction.md](./design-interaction.md) | 自动生成、绑定配置   |

**里程碑 M3**：能绘制工艺流程图（Canvas 图形 + DOM 组件）、配置管道流动动画

### 第四阶段：发布与运行（闭环交付）

**预计周期**：2-3 周

| 模块            | 文档                                         | 验收标准                    |
| --------------- | -------------------------------------------- | --------------------------- |
| 发布流水线      | [publish-pipeline.md](./publish-pipeline.md) | validate → compile → bundle |
| AssetNormalizer | [publish-pipeline.md](./publish-pipeline.md) | 资源分层打包                |
| 工程快照        | [publish-pipeline.md](./publish-pipeline.md) | 完整快照与回滚              |
| RuntimeEngine   | [runtime-engine.md](./runtime-engine.md)     | 独立运行、数据订阅          |
| 资源生命周期    | [runtime-engine.md](./runtime-engine.md)     | DisposableScope、Watchdog   |

**里程碑 M3**：能发布 .ifp、节点能加载并运行、数据正常显示

### 第五阶段：国际化与主题

**预计周期**：1-2 周

| 模块         | 文档                             | 验收标准                      |
| ------------ | -------------------------------- | ----------------------------- |
| 国际化       | [i18n-theme.md](./i18n-theme.md) | i18n 资源配置、$i18n 属性解析 |
| 主题系统     | [i18n-theme.md](./i18n-theme.md) | CSS 变量、主题定义            |
| 语言切换组件 | [i18n-theme.md](./i18n-theme.md) | LocaleSwitcher 运行时切换     |
| 主题切换组件 | [i18n-theme.md](./i18n-theme.md) | ThemeSwitcher 运行时切换      |

**里程碑 M4**：能配置多语言资源、能通过组件切换语言和主题

### 第六阶段：多端与高级特性

**预计周期**：1-2 周

| 模块       | 文档                                           | 验收标准                 |
| ---------- | ---------------------------------------------- | ------------------------ |
| Multi-View | [multi-view.md](./multi-view.md)               | 同路由多视图、运行时选路 |
| 权限系统   | [permissions.md](./permissions.md)             | 组件权限 + 动作权限      |
| 动作系统   | [action-system.md](./action-system.md)         | 完整动作类型、控制流     |
| 动画系统   | [animation-system.md](./animation-system.md)   | 状态动画、工业场景动画   |
| 验证系统   | [validation-system.md](./validation-system.md) | 表单验证、规则配置       |

## 文档索引

### 核心架构

- [编辑器内核](./editor-core.md) - DocumentModel、Command、History、Selection
- [Schema 设计](./schema-design.md) - 规范化工程 Schema（v2）、循环渲染、插槽、生命周期
- [组件清单](./component-manifest.md) - Component Manifest 规范

### 数据系统

- [数据绑定 v2](./data-binding-v2.md) - 三态隔离、Binding 结构、数据点状态
- [表达式引擎](./expression-engine.md) - 上下文变量、内置函数、工业计算
- [变量系统](./vars-system.md) - 页面级/全局变量

### 布局与渲染

- [布局系统](./layout-system.md) - Flex/Free/Grid、Constraints 约束
- [渲染架构](./rendering.md) - 设计态/运行态同构渲染
- [设计态交互](./design-interaction.md) - 工具栏、属性面板、Canvas 绘图、预览功能
- [Canvas 图形](./schema-design.md#6-canvas-图形节点) - 工艺流程图、管道、符号库

### 动作与动画

- [动作系统](./action-system.md) - 完整动作类型、条件分支、循环、并行
- [动画系统](./animation-system.md) - 状态驱动动画、工业场景动画
- [验证系统](./validation-system.md) - 表单验证规则、异步验证

### 发布与运行

- [发布流水线](./publish-pipeline.md) - 校验、编译、打包、快照
- [运行时引擎](./runtime-engine.md) - DataService、资源生命周期、Watchdog

### 多端与权限

- [多端适配](./multi-view.md) - Multi-View 模型
- [权限系统](./permissions.md) - 组件权限 + 动作权限

### 国际化与主题

- [国际化与主题](./i18n-theme.md) - i18n 资源管理、主题系统、切换组件

### 开发指南

- [最佳实践](./best-practices.md) - 性能优化、安全建议、命名规范、调试技巧

## 目录结构（重构后）

```
designer/
  src/
    editor-core/           # 编辑器内核
      document/
        types.ts           # Schema v2 类型定义
        documentModel.ts   # 规范化文档模型
        serializer.ts      # 导入导出
        migrations.ts      # 版本迁移
      commands/
        Command.ts         # 命令接口
        history.ts         # 撤销重做栈
        nodeCommands.ts    # 节点操作命令
        pageCommands.ts    # 页面操作命令
        bindingCommands.ts # 绑定操作命令
        varCommands.ts     # 变量操作命令
      selection/
        selectionModel.ts  # 选中状态管理
      validate/
        validator.ts       # Schema 校验
        bindingValidator.ts# 绑定校验（含数据点状态）
      registry/
        componentRegistry.ts
        manifests/

    data/                  # 数据层
      dataService.ts       # 数据服务
      datapointRegistry.ts # 数据点注册表
      mockDataProvider.ts  # Mock 数据（设计态）
      dataService.ts       # 数据服务（预览态 API 调用）
      expressionEngine.ts  # 表达式引擎
      varsStore.ts         # 变量状态管理
      diagnosticsStore.ts  # 诊断信息（数据点状态）

    renderer/              # 渲染层
      runtimeRenderer/
        renderNode.ts
        componentAdapters/
        bindingResolver.ts
      designRenderer/
        DesignWrapper.vue
        overlays/
        placeholderRenderer.ts
      canvasRenderer/       # Canvas 绘图层
        CanvasLayer.vue
        graphics/
          Line.ts
          Rect.ts
          Circle.ts
          Pipe.ts           # 管道（带流动动画）
          Text.ts
          Symbol.ts         # 符号引用
        pipeAnimation.ts    # 管道流动动画
        symbolRegistry.ts   # 符号库

    interaction/           # 交互层
      dnd/
      resize/
      align/
      constraints/         # 约束布局交互
      drawing/             # 绘图交互
        drawingToolManager.ts
        tools/
          LineTool.ts
          RectTool.ts
          CircleTool.ts
          PolygonTool.ts
          PipeTool.ts
          TextTool.ts

    services/              # 服务层
      projectApi.ts
      datacenterApi.ts
      publishPipeline.ts
      assetNormalizer.ts

    ui/                    # UI 层
      LeftPanel/
        DrawingTools/       # 绘图工具面板
        ComponentPanel/     # 组件面板
        SymbolLibrary/      # 符号库面板
        PageTree/           # 页面树
        OutlineTree/        # 大纲树
        DatapointPanel/     # 数据点面板
      Canvas/               # 画布区域
        CanvasContainer.vue
        CanvasLayer.vue     # Canvas 绘图层
        DOMLayer.vue        # DOM 组件层
      RightPanel/
        PropertyPanel/      # 属性面板
        StylePanel/         # 样式面板
        EventPanel/         # 事件面板
        BindingPanel/       # 绑定面板
      TopToolbar/           # 顶部工具栏
        SelectTools/        # 选择工具
        DrawingTools/       # 绘图工具
      DatapointPicker/
      DiagnosticsPanel/
```

## 相关文档

- [高层设计](../../高层设计.md) - 全链路架构设计
- [数据中心](../../datacenter/README.md) - DataCenter 文档
- [数据库设计](../../高层设计.md#9-数据库设计当前实现) - 表结构设计

---

**版本**: 1.0.0  
**创建日期**: 2026-01-06
