# Layout Components Registration

## Overview

This directory contains all layout component definitions and their Vue implementations for the InduForge Designer.

**Task**: Task 2.3 - 重构布局组件注册（Container, Row/Col, Flex, Grid, CenterLayout）

## Components

### 1. Container (`Container.js` / `Container.vue`)
- **Type**: `Container`
- **Name**: 盒子容器
- **Description**: 布局容器，支持 Flex/Grid/Block 三种布局模式
- **Features**:
  - Supports 3 layout modes: Flex, Grid, Block
  - Configurable flex properties (direction, justify, align, wrap)
  - Configurable grid properties (template columns/rows, auto flow)
  - Customizable appearance (background, border, shadow)

### 2. Row (`Row.js` / `Row.vue`)
- **Type**: `Row`
- **Name**: 行列容器
- **Description**: 栅格行容器，宽度 100%，配合 Col 组件使用
- **Features**:
  - 24-grid system support
  - Horizontal and vertical gutter
  - Justify and align options
  - Auto wrap support

### 3. Col (`Col.js` / `Col.vue`)
- **Type**: `Col`
- **Name**: 栅格布局
- **Description**: 栅格列组件，配合 Row 使用，基于 24 栅格系统
- **Features**:
  - Span (1-24) for width calculation
  - Offset for left spacing
  - Push/Pull for positioning
  - Cannot be dragged freely in canvas (only within Row)

### 4. FlexLayout (`FlexLayout.js` / `Flex.vue`)
- **Type**: `FlexLayout`
- **Name**: 弹性容器
- **Description**: 高级 Flex 布局容器，支持完整的 Flexbox 属性
- **Features**:
  - Full Flexbox property support
  - Direction (row, column, reverse)
  - Justify content and align items
  - Wrap options
  - Gap spacing

### 5. Grid (`Grid.js` / `Grid.vue`)
- **Type**: `Grid`
- **Name**: 栅格布局
- **Description**: CSS Grid 布局容器，支持完整的 Grid 属性
- **Features**:
  - Grid template columns and rows
  - Grid auto flow
  - Justify and align items/content
  - Row and column gap
  - Full CSS Grid support

### 6. CenterLayout (`CenterLayout.js` / `CenterLayout.vue`)
- **Type**: `CenterLayout`
- **Name**: 全宽居中
- **Description**: 居中布局容器，子组件自动水平垂直居中
- **Features**:
  - Automatic horizontal and vertical centering
  - Gap spacing for multiple children
  - Simplified Flexbox preset

## Registration

All layout components are registered through the ComponentFactory:

```javascript
import layoutComponents from './layout/index.js'
import { registerAllComponentsSync } from './index.js'

// Register all layout components
registerAllComponentsSync(layoutComponents)
```

## Component Definition Structure

Each component definition includes:

- `type`: Unique component identifier
- `name`: Display name (Chinese)
- `category`: Component category (布局组件)
- `icon`: Icon identifier
- `component`: Vue component reference
- `container`: Boolean flag (all layout components are containers)
- `defaultProps`: Default property values
- `defaultStyle`: Default style values
- `propsSchema`: Property schema for the property panel
- `eventsSchema`: Available events
- `version`: Component version

## Testing

All layout components have comprehensive tests in `__tests__/layoutRegistration.test.js`:

- Component registration verification
- Component retrieval by type
- Component instance creation
- Property schema validation
- Vue component reference verification
- Category and search functionality

Run tests:
```bash
npm test layoutRegistration.test.js
```

## Usage Example

```javascript
import { createComponentInstance } from '@/registry'

// Create a Container instance
const container = createComponentInstance('Container', {
  props: {
    layout: 'flex',
    flexDirection: 'row',
    gap: 20
  },
  style: {
    left: 100,
    top: 100,
    width: 400,
    height: 300
  }
})

// Create a Grid instance
const grid = createComponentInstance('Grid', {
  props: {
    gridTemplateColumns: 'repeat(4, 1fr)',
    gap: 15
  }
})
```

## Requirements Satisfied

✅ Requirement 2: DOM 布局容器渲染
- All layout containers use Vue components
- CSS native layout (Flexbox, Grid, Block)
- True WYSIWYG rendering

✅ Requirement 12: 组件注册机制
- Unified component registration through ComponentFactory
- Component metadata storage
- Dynamic component loading

## Files Modified/Created

### Created:
- `Grid.js` - Grid component definition
- `__tests__/layoutRegistration.test.js` - Comprehensive test suite
- `README.md` - This documentation

### Modified:
- `index.js` - Updated to export all 6 layout components
- `Container.js` - Added Vue component reference
- `Row.js` - Added Vue component reference
- `Col.js` - Added Vue component reference
- `FlexLayout.js` - Added Vue component reference
- `CenterLayout.js` - Added Vue component reference
- `../index.js` - Added `registerAllComponentsSync` function

## Next Steps

The following tasks remain for complete layout component implementation:

1. Task 2.4: 实现动态组件渲染 (Integrate with DomRenderer)
2. Task 3.1-3.8: Implement layout component features (drop zones, style conversion, etc.)
3. Task 4.3: 实现拖拽到容器 (Drag and drop into containers)

## Notes

- All layout components are marked as `container: true`
- Row and Col work together as a 24-grid system
- Container supports 3 layout modes (Flex/Grid/Block)
- Grid component provides full CSS Grid capabilities
- CenterLayout is a simplified preset for centering content
