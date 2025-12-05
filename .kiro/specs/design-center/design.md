# Design Document: Design Center (Phase 1)

## Overview

设计中心是 InduForge 低代码平台的可视化页面设计模块。第一阶段实现核心编辑器框架，包括页面管理、画布渲染、组件交互、属性编辑和基础组件库。

### 技术栈
- **前端**: Vue 3 + Pinia + Element Plus + Tailwind CSS
- **后端**: Express + Sequelize (复用 dev_core)
- **数据格式**: JSON (DSL Schema)

### 架构概览

```
┌─────────────────────────────────────────────────────────────────┐
│                        Design Center                             │
├─────────────┬─────────────────────────────┬────────────────────┤
│  Page Tree  │         Canvas              │  Property Panel    │
│  (左侧)     │         (中间)               │  (右侧)            │
├─────────────┼─────────────────────────────┼────────────────────┤
│  Component  │                             │  Style Editor      │
│  Tree       │                             │                    │
├─────────────┤                             ├────────────────────┤
│  Component  │                             │  Props Editor      │
│  Library    │                             │                    │
└─────────────┴─────────────────────────────┴────────────────────┘
```

## Architecture

### 前端架构

```
designer/src/
├── api/                    # API 调用层
│   └── design.api.js       # 设计中心 API
├── components/             # 通用组件
│   ├── canvas/             # 画布相关组件
│   │   ├── DesignCanvas.vue
│   │   ├── CanvasComponent.vue
│   │   └── SelectionOverlay.vue
│   ├── panels/             # 面板组件
│   │   ├── PageTree.vue
│   │   ├── ComponentTree.vue
│   │   ├── ComponentLibrary.vue
│   │   └── PropertyPanel.vue
│   └── editors/            # 属性编辑器
│       ├── StyleEditor.vue
│       └── PropsEditor.vue
├── composables/            # 组合式函数
│   ├── useCanvas.js        # 画布逻辑
│   ├── useSelection.js     # 选择逻辑
│   └── useDragDrop.js      # 拖拽逻辑
├── store/                  # Pinia 状态管理
│   ├── design.js           # 设计状态
│   └── page.js             # 页面状态
├── registry/               # 组件注册
│   ├── index.js            # 组件注册表
│   └── components/         # 基础组件定义
│       ├── Container.js
│       ├── Text.js
│       ├── Button.js
│       ├── Image.js
│       └── Input.js
└── views/
    └── DesignCenter.vue    # 主视图
```

### 后端架构

```
dev_core/src/
├── models/
│   └── DesignPage.js       # 页面模型
├── services/
│   └── designService.js    # 设计服务
├── controllers/
│   └── designController.js # 设计控制器
└── routes/
    └── design.js           # 设计路由
```

### 数据流

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│   User       │────▶│   Store      │────▶│   Canvas     │
│   Action     │     │   (Pinia)    │     │   Render     │
└──────────────┘     └──────────────┘     └──────────────┘
       │                    │                    │
       │                    ▼                    │
       │             ┌──────────────┐            │
       └────────────▶│   API        │◀───────────┘
                     │   (Backend)  │
                     └──────────────┘
```

## Components and Interfaces

### 1. Store 接口

#### DesignStore (store/design.js)

```javascript
// State
{
  projectId: string | null,      // 当前项目 ID
  pages: Page[],                 // 页面列表
  currentPageId: string | null,  // 当前页面 ID
  currentPage: PageSchema | null,// 当前页面 Schema
  selectedComponentId: string | null, // 选中组件 ID
  isDirty: boolean,              // 是否有未保存更改
  clipboard: ComponentSchema | null, // 剪贴板
}

// Actions
loadProject(projectId: string): Promise<void>
loadPage(pageId: string): Promise<void>
savePage(): Promise<void>
createPage(name: string, parentId?: string): Promise<Page>
deletePage(pageId: string): Promise<void>
renamePage(pageId: string, name: string): Promise<void>

selectComponent(componentId: string | null): void
updateComponent(componentId: string, updates: Partial<ComponentSchema>): void
addComponent(component: ComponentSchema, parentId?: string): void
removeComponent(componentId: string): void
moveComponent(componentId: string, targetParentId: string, index: number): void
```

### 2. API 接口

#### Design API (api/design.api.js)

```javascript
// 页面管理
getPages(projectId: string): Promise<Page[]>
getPage(pageId: string): Promise<PageSchema>
createPage(projectId: string, data: CreatePageDTO): Promise<Page>
updatePage(pageId: string, schema: PageSchema): Promise<void>
deletePage(pageId: string): Promise<void>
```

### 3. 组件注册接口

#### ComponentRegistry (registry/index.js)

```javascript
interface ComponentDefinition {
  type: string;           // 组件类型标识
  name: string;           // 显示名称
  category: string;       // 分类
  icon: string;           // 图标
  defaultProps: object;   // 默认属性
  defaultStyle: object;   // 默认样式
  propsSchema: object;    // 属性 Schema (用于生成表单)
  render: (props, style) => VNode; // 渲染函数
}

registerComponent(definition: ComponentDefinition): void
getComponent(type: string): ComponentDefinition | null
getAllComponents(): ComponentDefinition[]
getComponentsByCategory(): Record<string, ComponentDefinition[]>
```

### 4. Canvas 接口

#### useCanvas Composable

```javascript
interface CanvasState {
  scale: number;          // 缩放比例
  offset: { x: number, y: number }; // 画布偏移
  gridSize: number;       // 网格大小
  snapToGrid: boolean;    // 是否吸附网格
}

useCanvas(): {
  canvasState: Ref<CanvasState>,
  calculateScale(viewportSize, canvasSize, scaleMode): number,
  snapToGrid(position: Position, gridSize: number): Position,
  screenToCanvas(screenPos: Position): Position,
  canvasToScreen(canvasPos: Position): Position,
}
```

### 5. Selection 接口

#### useSelection Composable

```javascript
useSelection(): {
  selectedId: Ref<string | null>,
  selectedComponent: ComputedRef<ComponentSchema | null>,
  select(componentId: string): void,
  deselect(): void,
  isSelected(componentId: string): boolean,
}
```

## Data Models

### 1. DesignPage 数据库模型

```javascript
// dev_core/src/models/DesignPage.js
{
  id: UUID (PK),
  projectId: UUID (FK -> projects.id),
  parentId: UUID (FK -> design_pages.id, nullable), // 父页面/文件夹
  name: STRING(200),
  type: ENUM('page', 'folder', 'dialog'),
  schemaContent: JSON,    // 完整 Page Schema
  sortOrder: INTEGER,     // 排序顺序
  lockedBy: UUID (FK -> users.id, nullable), // 编辑锁
  lockedAt: DATE,
  createdBy: UUID (FK -> users.id),
  updatedBy: UUID (FK -> users.id),
  createdAt: DATE,
  updatedAt: DATE,
}
```

### 2. Page Schema (DSL)

```javascript
// 简化的 Page Schema 结构
{
  version: "2.0.0",
  meta: {
    id: string,
    name: string,
    description?: string,
  },
  config: {
    width: number,
    height: number,
    scaleMode: 'fit' | 'fill' | 'fixed',
    backgroundColor: string,
    gridSize: number,
    snapToGrid: boolean,
    theme: 'light' | 'dark',
  },
  variables: Record<string, VariableDefinition>,
  dataSources: DataSourceSchema[],
  components: ComponentSchema[],
  permissions: PermissionsSchema,
}
```

### 3. Component Schema

```javascript
{
  id: string,
  type: string,
  label: string,
  locked: boolean,
  visible: boolean,
  style: {
    position: 'absolute' | 'relative',
    left: number,
    top: number,
    width: number,
    height: number,
    zIndex: number,
    // ... 其他样式属性
  },
  props: Record<string, any>,
  bindings: Record<string, string>,
  events: Record<string, ActionSchema[]>,
  animations: AnimationSchema[],
  children: ComponentSchema[],
}
```

## Correctness Properties

*A property is a characteristic or behavior that should hold true across all valid executions of a system-essentially, a formal statement about what the system should do. Properties serve as the bridge between human-readable specifications and machine-verifiable correctness guarantees.*

### Property 1: Component Style Rendering
*For any* valid component schema with style properties (left, top, width, height, zIndex), the rendered component element SHALL have CSS styles matching those values.
**Validates: Requirements 2.1, 2.2**

### Property 2: Canvas Scale Calculation
*For any* viewport dimensions and canvas dimensions with scaleMode "fit", the calculated scale factor SHALL ensure the canvas fits within the viewport while maintaining aspect ratio (scale = min(viewportWidth/canvasWidth, viewportHeight/canvasHeight)).
**Validates: Requirements 2.4**

### Property 3: Grid Snapping Calculation
*For any* position (x, y) and grid size g > 0, the snapped position SHALL be (round(x/g)*g, round(y/g)*g).
**Validates: Requirements 3.6**

### Property 4: Drag Position Update
*For any* component with initial position (x, y) and drag delta (dx, dy), the resulting position SHALL be (x + dx, y + dy) before grid snapping.
**Validates: Requirements 3.4**

### Property 5: Resize Dimension Update
*For any* component with initial dimensions (w, h) and resize delta (dw, dh), the resulting dimensions SHALL be (max(minWidth, w + dw), max(minHeight, h + dh)).
**Validates: Requirements 3.5**

### Property 6: Control Type Selection
*For any* prop definition with type T, the Property Panel SHALL render: switch for boolean, number input for number, text input for string, select for enum.
**Validates: Requirements 4.4, 4.5, 4.6, 4.7**

### Property 7: Prop Update Mutation
*For any* component and prop update (key, value), the resulting schema SHALL have component.props[key] === value with all other props unchanged.
**Validates: Requirements 4.2, 4.3**

### Property 8: Component Tree Structure
*For any* page schema with nested components, the component tree SHALL display a tree structure where each component's children appear as nested items under their parent.
**Validates: Requirements 5.1**

### Property 9: Locked Component Protection
*For any* component with locked === true, selection and editing operations SHALL be prevented.
**Validates: Requirements 5.5**

### Property 10: Dirty State Tracking
*For any* schema modification operation, the isDirty flag SHALL be set to true.
**Validates: Requirements 6.1**

### Property 11: Schema Validation
*For any* page schema submitted to the backend, the validator SHALL return errors for missing required fields or invalid enum values.
**Validates: Requirements 6.3**

### Property 12: Schema Round-Trip
*For any* valid Page Schema, serializing to JSON and parsing back SHALL produce an equivalent schema (JSON.parse(JSON.stringify(schema)) deep equals schema).
**Validates: Requirements 6.6, 6.7**

### Property 13: API Page List Response
*For any* project with pages, the API response SHALL include all pages with id, name, type, and parentId fields.
**Validates: Requirements 7.1**

### Property 14: UUID Generation
*For any* page creation request, the backend SHALL generate a valid UUID v4 for the page id.
**Validates: Requirements 7.3**

### Property 15: Folder Deletion Protection
*For any* folder page with child pages, deletion SHALL fail with an error indicating children must be removed first.
**Validates: Requirements 7.6**

### Property 16: Component Instantiation Defaults
*For any* component type in the registry, creating a new instance SHALL produce a component with all defaultProps and defaultStyle values applied.
**Validates: Requirements 8.2**

## Error Handling

### 前端错误处理

1. **API 错误**: 使用 Element Plus Message 组件显示错误信息
2. **验证错误**: 在 Property Panel 中显示字段级错误
3. **保存冲突**: 检测并发编辑，提示用户刷新或覆盖

### 后端错误处理

1. **验证错误**: 返回 400 状态码和详细错误列表
2. **未找到**: 返回 404 状态码
3. **权限错误**: 返回 403 状态码
4. **服务器错误**: 返回 500 状态码，记录日志

### 错误码定义

```javascript
const DesignErrorCodes = {
  PAGE_NOT_FOUND: 'DESIGN_001',
  SCHEMA_VALIDATION_FAILED: 'DESIGN_002',
  PAGE_LOCKED: 'DESIGN_003',
  FOLDER_NOT_EMPTY: 'DESIGN_004',
  INVALID_PARENT: 'DESIGN_005',
};
```

## Testing Strategy

### 单元测试

使用 Vitest 进行单元测试：

1. **Store 测试**: 测试状态管理逻辑
2. **Composable 测试**: 测试 useCanvas、useSelection 等
3. **Validator 测试**: 测试 DSL 验证器
4. **API 测试**: 测试后端 API 端点

### 属性测试

使用 fast-check 进行属性测试：

1. **Schema 验证属性**: 验证各种 schema 结构
2. **计算属性**: 验证缩放、网格吸附等计算
3. **状态变更属性**: 验证状态变更的正确性
4. **Round-trip 属性**: 验证序列化/反序列化

### 测试配置

```javascript
// vitest.config.js
{
  test: {
    environment: 'jsdom',
    globals: true,
    coverage: {
      reporter: ['text', 'json', 'html'],
    },
  },
}
```

### 属性测试示例

```javascript
// 每个属性测试运行至少 100 次迭代
import { fc } from 'fast-check';

// Property 3: Grid Snapping
test('grid snapping produces multiples of grid size', () => {
  fc.assert(
    fc.property(
      fc.integer({ min: 0, max: 2000 }),
      fc.integer({ min: 0, max: 2000 }),
      fc.integer({ min: 1, max: 100 }),
      (x, y, gridSize) => {
        const snapped = snapToGrid({ x, y }, gridSize);
        return snapped.x % gridSize === 0 && snapped.y % gridSize === 0;
      }
    ),
    { numRuns: 100 }
  );
});
```
