# Implementation Plan

- [x] 1. 后端基础设施






  - [x] 1.1 创建 DesignPage 数据库模型

    - 创建 `dev_core/src/models/DesignPage.js`
    - 定义字段：id, projectId, parentId, name, type, schemaContent, sortOrder, lockedBy, lockedAt, createdBy, updatedBy
    - 添加索引和外键关系
    - 在 models/index.js 中注册模型
    - _Requirements: 7.1, 7.2, 7.3_



  - [ ] 1.2 创建设计服务层
    - 创建 `dev_core/src/services/designService.js`
    - 实现 getPages(projectId) - 获取项目页面列表
    - 实现 getPage(pageId) - 获取单个页面 Schema
    - 实现 createPage(projectId, data) - 创建页面
    - 实现 updatePage(pageId, schema) - 更新页面
    - 实现 deletePage(pageId) - 删除页面（含文件夹保护）
    - _Requirements: 7.1, 7.2, 7.3, 7.4, 7.5, 7.6_

  - [ ]* 1.3 编写属性测试：文件夹删除保护
    - **Property 15: Folder Deletion Protection**
    - **Validates: Requirements 7.6**

  - [ ]* 1.4 编写属性测试：UUID 生成
    - **Property 14: UUID Generation**


    - **Validates: Requirements 7.3**

  - [ ] 1.5 创建设计控制器和路由
    - 创建 `dev_core/src/controllers/designController.js`
    - 创建 `dev_core/src/routes/design.js`
    - 实现 RESTful API 端点：GET/POST/PUT/DELETE
    - 集成 Schema 验证
    - _Requirements: 7.1, 7.2, 7.3, 7.4, 7.5_

  - [ ]* 1.6 编写属性测试：API 页面列表响应
    - **Property 13: API Page List Response**
    - **Validates: Requirements 7.1**

  - [ ]* 1.7 编写属性测试：Schema 验证
    - **Property 11: Schema Validation**
    - **Validates: Requirements 6.3**

- [ ] 2. Checkpoint - 确保所有测试通过
  - Ensure all tests pass, ask the user if questions arise.

- [ ] 3. 前端状态管理
  - [ ] 3.1 创建 Design Store
    - 创建 `designer/src/store/design.js`
    - 实现状态：projectId, pages, currentPageId, currentPage, selectedComponentId, isDirty
    - 实现 actions：loadProject, loadPage, savePage, createPage, deletePage, renamePage
    - 实现组件操作：selectComponent, updateComponent, addComponent, removeComponent, moveComponent
    - _Requirements: 1.1, 1.2, 1.3, 1.4, 1.5, 6.1, 6.2, 6.5_

  - [ ]* 3.2 编写属性测试：脏状态跟踪
    - **Property 10: Dirty State Tracking**
    - **Validates: Requirements 6.1**

  - [ ]* 3.3 编写属性测试：属性更新变更
    - **Property 7: Prop Update Mutation**
    - **Validates: Requirements 4.2, 4.3**

  - [ ] 3.4 创建 Design API 模块
    - 创建 `designer/src/api/design.api.js`
    - 实现 getPages, getPage, createPage, updatePage, deletePage
    - _Requirements: 7.1, 7.2, 7.3, 7.4, 7.5_

- [ ] 4. 画布核心功能
  - [ ] 4.1 创建 useCanvas Composable
    - 创建 `designer/src/composables/useCanvas.js`
    - 实现 calculateScale(viewportSize, canvasSize, scaleMode)
    - 实现 snapToGrid(position, gridSize)
    - 实现 screenToCanvas/canvasToScreen 坐标转换
    - _Requirements: 2.3, 2.4, 3.6_

  - [ ]* 4.2 编写属性测试：画布缩放计算
    - **Property 2: Canvas Scale Calculation**
    - **Validates: Requirements 2.4**

  - [ ]* 4.3 编写属性测试：网格吸附计算
    - **Property 3: Grid Snapping Calculation**
    - **Validates: Requirements 3.6**

  - [ ] 4.4 创建 useSelection Composable
    - 创建 `designer/src/composables/useSelection.js`
    - 实现 select, deselect, isSelected
    - _Requirements: 3.1, 3.2_

  - [ ] 4.5 创建 useDragDrop Composable
    - 创建 `designer/src/composables/useDragDrop.js`
    - 实现拖拽位置计算
    - 实现调整大小计算
    - _Requirements: 3.4, 3.5_

  - [ ]* 4.6 编写属性测试：拖拽位置更新
    - **Property 4: Drag Position Update**
    - **Validates: Requirements 3.4**

  - [ ]* 4.7 编写属性测试：调整大小更新
    - **Property 5: Resize Dimension Update**
    - **Validates: Requirements 3.5**

- [ ] 5. Checkpoint - 确保所有测试通过
  - Ensure all tests pass, ask the user if questions arise.

- [ ] 6. 画布组件
  - [ ] 6.1 创建 DesignCanvas 组件
    - 创建 `designer/src/components/canvas/DesignCanvas.vue`
    - 实现画布容器，支持缩放和平移
    - 渲染 page config 指定的宽高和背景
    - _Requirements: 2.3, 2.4_

  - [ ] 6.2 创建 CanvasComponent 组件
    - 创建 `designer/src/components/canvas/CanvasComponent.vue`
    - 根据 component schema 渲染组件
    - 应用 style 属性（position, left, top, width, height, zIndex）
    - 递归渲染 children
    - _Requirements: 2.1, 2.2_

  - [ ]* 6.3 编写属性测试：组件样式渲染
    - **Property 1: Component Style Rendering**
    - **Validates: Requirements 2.1, 2.2**

  - [ ] 6.4 创建 SelectionOverlay 组件
    - 创建 `designer/src/components/canvas/SelectionOverlay.vue`
    - 显示选中组件的边框和调整手柄
    - 处理拖拽和调整大小事件
    - _Requirements: 2.5, 3.4, 3.5_

- [ ] 7. 组件注册系统
  - [ ] 7.1 创建组件注册表
    - 创建 `designer/src/registry/index.js`
    - 实现 registerComponent, getComponent, getAllComponents, getComponentsByCategory
    - _Requirements: 8.1_

  - [ ] 7.2 实现基础组件定义
    - 创建 `designer/src/registry/components/Container.js`
    - 创建 `designer/src/registry/components/Text.js`
    - 创建 `designer/src/registry/components/Button.js`
    - 创建 `designer/src/registry/components/Image.js`
    - 创建 `designer/src/registry/components/Input.js`
    - 每个组件定义 type, name, category, icon, defaultProps, defaultStyle, propsSchema
    - _Requirements: 8.3, 8.4, 8.5, 8.6, 8.7_

  - [ ]* 7.3 编写属性测试：组件实例化默认值
    - **Property 16: Component Instantiation Defaults**
    - **Validates: Requirements 8.2**

- [ ] 8. 面板组件
  - [ ] 8.1 创建 PageTree 组件
    - 创建 `designer/src/components/panels/PageTree.vue`
    - 显示项目页面层级结构
    - 支持创建、重命名、删除页面
    - 支持创建文件夹
    - _Requirements: 1.1, 1.2, 1.4, 1.5, 1.6_

  - [ ] 8.2 创建 ComponentTree 组件
    - 创建 `designer/src/components/panels/ComponentTree.vue`
    - 显示当前页面组件层级
    - 支持点击选中组件
    - 支持拖拽重排序
    - 显示锁定状态
    - _Requirements: 5.1, 5.2, 5.3, 5.4, 5.5_

  - [ ]* 8.3 编写属性测试：组件树结构
    - **Property 8: Component Tree Structure**
    - **Validates: Requirements 5.1**

  - [ ]* 8.4 编写属性测试：锁定组件保护
    - **Property 9: Locked Component Protection**
    - **Validates: Requirements 5.5**

  - [ ] 8.5 创建 ComponentLibrary 组件
    - 创建 `designer/src/components/panels/ComponentLibrary.vue`
    - 按分类显示可用组件
    - 支持拖拽到画布
    - _Requirements: 8.1, 8.2_

  - [ ] 8.6 创建 PropertyPanel 组件
    - 创建 `designer/src/components/panels/PropertyPanel.vue`
    - 显示选中组件的属性编辑器
    - 包含 StyleEditor 和 PropsEditor
    - _Requirements: 3.3, 4.1_

  - [ ]* 8.7 编写属性测试：控件类型选择
    - **Property 6: Control Type Selection**
    - **Validates: Requirements 4.4, 4.5, 4.6, 4.7**

- [ ] 9. Checkpoint - 确保所有测试通过
  - Ensure all tests pass, ask the user if questions arise.

- [ ] 10. 属性编辑器
  - [ ] 10.1 创建 StyleEditor 组件
    - 创建 `designer/src/components/editors/StyleEditor.vue`
    - 编辑 position, left, top, width, height, zIndex 等样式
    - _Requirements: 4.3_

  - [ ] 10.2 创建 PropsEditor 组件
    - 创建 `designer/src/components/editors/PropsEditor.vue`
    - 根据 propsSchema 动态生成表单
    - 支持 boolean/number/string/enum 类型
    - _Requirements: 4.1, 4.2, 4.4, 4.5, 4.6, 4.7_

- [ ] 11. 主视图集成
  - [ ] 11.1 重构 DesignCenter.vue
    - 替换 iframe 为自研设计器
    - 集成三栏布局：PageTree + ComponentTree + ComponentLibrary | Canvas | PropertyPanel
    - 添加工具栏：保存、撤销、重做按钮
    - _Requirements: 1.1, 1.3, 6.2_

  - [ ] 11.2 实现保存功能
    - 集成 savePage action
    - 显示保存状态和错误信息
    - _Requirements: 6.2, 6.4, 6.5_

- [ ] 12. Schema 序列化测试
  - [ ]* 12.1 编写属性测试：Schema Round-Trip
    - **Property 12: Schema Round-Trip**
    - **Validates: Requirements 6.6, 6.7**

- [ ] 13. Final Checkpoint - 确保所有测试通过
  - Ensure all tests pass, ask the user if questions arise.
