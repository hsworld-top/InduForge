# Requirements Document

## Introduction

设计中心是 InduForge 低代码平台的核心模块，提供可视化页面设计能力。用户可以通过拖拽组件、配置属性、绑定数据源等方式快速构建工业应用页面。设计中心需要支持 DSL 规范定义的所有功能，包括页面管理、组件编辑、数据源配置、动作系统、权限控制等。

本规范采用分阶段开发策略，第一阶段聚焦于核心页面管理和基础编辑器框架。

## Glossary

- **DSL (Domain Specific Language)**: 领域特定语言，用于描述页面结构的 JSON Schema
- **Page Schema**: 页面的完整 DSL 结构，包含 meta、config、variables、dataSources、components、permissions 等节点
- **Component Schema**: 组件的 DSL 结构，包含 id、type、style、props、bindings、events、children 等属性
- **Design Center**: 设计中心，可视化页面设计工具
- **Canvas**: 画布，页面设计的可视化编辑区域
- **Component Tree**: 组件树，展示页面组件层级结构的面板
- **Property Panel**: 属性面板，用于编辑选中组件属性的面板
- **Page Tree**: 页面树，展示项目中所有页面的层级结构

## Requirements

### Requirement 1: 页面管理

**User Story:** As a developer, I want to manage pages within a project, so that I can organize and structure my application's UI.

#### Acceptance Criteria

1. WHEN a user opens the design center with a project ID THEN the Design Center SHALL display the page tree showing all pages in the project
2. WHEN a user clicks the "create page" button THEN the Design Center SHALL create a new page with default Page Schema and add it to the page tree
3. WHEN a user selects a page from the page tree THEN the Design Center SHALL load the page's schema and display it in the canvas
4. WHEN a user renames a page THEN the Design Center SHALL update the page's meta.name field and persist the change
5. WHEN a user deletes a page THEN the Design Center SHALL remove the page from the database after confirmation
6. WHEN a user creates a folder in the page tree THEN the Design Center SHALL create a page entry with type "folder" to organize pages hierarchically

### Requirement 2: 画布渲染

**User Story:** As a developer, I want to see a visual representation of my page design, so that I can understand how the page will look at runtime.

#### Acceptance Criteria

1. WHEN a page schema is loaded THEN the Canvas SHALL render all components according to their style properties (position, size, zIndex)
2. WHEN a component has absolute positioning THEN the Canvas SHALL position the component at the specified left/top coordinates
3. WHEN the page config specifies width and height THEN the Canvas SHALL display a bounded design area with those dimensions
4. WHEN the page config specifies scaleMode as "fit" THEN the Canvas SHALL scale the design area to fit within the viewport while maintaining aspect ratio
5. WHEN a component is selected THEN the Canvas SHALL display selection handles around the component

### Requirement 3: 组件选择与基础交互

**User Story:** As a developer, I want to select and manipulate components on the canvas, so that I can edit their properties and layout.

#### Acceptance Criteria

1. WHEN a user clicks on a component in the canvas THEN the Design Center SHALL select that component and highlight it
2. WHEN a user clicks on empty canvas area THEN the Design Center SHALL deselect all components
3. WHEN a component is selected THEN the Property Panel SHALL display the component's editable properties
4. WHEN a user drags a selected component THEN the Canvas SHALL update the component's left/top style properties
5. WHEN a user resizes a component using handles THEN the Canvas SHALL update the component's width/height style properties
6. WHEN snapToGrid is enabled in page config THEN the Canvas SHALL snap component positions to the grid during drag/resize

### Requirement 4: 属性面板

**User Story:** As a developer, I want to edit component properties through a form interface, so that I can configure component behavior without writing code.

#### Acceptance Criteria

1. WHEN a component is selected THEN the Property Panel SHALL display form fields for all props defined in the component schema
2. WHEN a user modifies a prop value in the Property Panel THEN the Design Center SHALL update the component's props in the schema
3. WHEN a user modifies a style value in the Property Panel THEN the Design Center SHALL update the component's style in the schema and re-render the canvas
4. WHEN a prop type is boolean THEN the Property Panel SHALL render a switch/checkbox control
5. WHEN a prop type is number THEN the Property Panel SHALL render a numeric input control
6. WHEN a prop type is string THEN the Property Panel SHALL render a text input control
7. WHEN a prop has enum options THEN the Property Panel SHALL render a select/dropdown control

### Requirement 5: 组件树面板

**User Story:** As a developer, I want to see the hierarchical structure of components, so that I can navigate and manage complex page layouts.

#### Acceptance Criteria

1. WHEN a page is loaded THEN the Component Tree SHALL display all components in a hierarchical tree structure
2. WHEN a user clicks a component in the tree THEN the Design Center SHALL select that component in the canvas
3. WHEN a user drags a component in the tree THEN the Design Center SHALL reorder or reparent the component in the schema
4. WHEN a component has children THEN the Component Tree SHALL display them as nested items under the parent
5. WHEN a component is locked THEN the Component Tree SHALL display a lock icon and prevent selection/editing

### Requirement 6: Schema 持久化

**User Story:** As a developer, I want my design changes to be saved automatically, so that I don't lose my work.

#### Acceptance Criteria

1. WHEN a user makes changes to the page schema THEN the Design Center SHALL mark the page as having unsaved changes
2. WHEN a user clicks the save button THEN the Design Center SHALL persist the current schema to the backend API
3. WHEN saving a page schema THEN the Backend SHALL validate the schema against DSL validators before storing
4. WHEN schema validation fails THEN the Backend SHALL return validation errors and the Design Center SHALL display them to the user
5. WHEN a page is saved successfully THEN the Design Center SHALL clear the unsaved changes indicator
6. WHEN serializing a Page Schema to JSON THEN the Design Center SHALL produce valid JSON that can be parsed back to an equivalent schema
7. WHEN parsing a Page Schema from JSON THEN the Design Center SHALL reconstruct the original schema structure

### Requirement 7: 后端 API

**User Story:** As a developer, I want backend APIs to manage design data, so that the design center can persist and retrieve page schemas.

#### Acceptance Criteria

1. WHEN the Design Center requests page list THEN the Backend SHALL return all pages for the specified project with id, name, type, and parentId
2. WHEN the Design Center requests a page by ID THEN the Backend SHALL return the complete Page Schema
3. WHEN the Design Center creates a new page THEN the Backend SHALL generate a UUID, create the page record, and return the new page data
4. WHEN the Design Center updates a page THEN the Backend SHALL validate and store the updated schema
5. WHEN the Design Center deletes a page THEN the Backend SHALL remove the page record and return success status
6. WHEN a page has child pages (folder) THEN the Backend SHALL prevent deletion until children are moved or deleted

### Requirement 8: 基础组件库

**User Story:** As a developer, I want a library of basic components, so that I can quickly build page layouts.

#### Acceptance Criteria

1. WHEN the Design Center loads THEN the Component Library SHALL display available components grouped by category
2. WHEN a user drags a component from the library to the canvas THEN the Design Center SHALL create a new component instance with default props
3. THE Design Center SHALL provide a Container component that supports flex/grid layout for children
4. THE Design Center SHALL provide a Text component for displaying static or bound text content
5. THE Design Center SHALL provide a Button component with click event support
6. THE Design Center SHALL provide an Image component for displaying images from assets or URLs
7. THE Design Center SHALL provide an Input component for text input with validation support

