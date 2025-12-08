# 设计中心 (Designer) 文档

## 概述

设计中心是 InduForge 平台的核心模块，提供可视化的页面设计功能。通过拖拽式操作，用户可以快速构建工业应用界面，无需编写代码。

## 核心功能

### 1. 可视化设计
- 拖拽式组件布局
- 实时预览
- 所见即所得
- 多种画布缩放模式

### 2. 组件库
- **基础组件**: 矩形、圆形、文本、图片等
- **UI 组件**: 按钮、输入框、表格、图表等
- **工业组件**: 电机、泵、阀门、管道等
- **自定义组件**: 支持组件封装和复用

### 3. 数据绑定
- 表达式绑定
- 实时数据订阅
- 多数据源支持
- 数据转换和计算

### 4. Canvas 渲染
- 基于 Konva.js 的高性能渲染
- 支持大量组件
- 流畅的交互体验
- 智能对齐和吸附

### 5. 历史记录
- 撤销/重做
- 操作历史追踪
- 快照管理

## 技术架构

```
Designer
├── engine/                 # 核心引擎
│   ├── canvas/            # Canvas 渲染引擎
│   ├── binding/           # 数据绑定引擎
│   ├── animation/         # 动画引擎
│   └── datasource/        # 数据源管理
├── registry/              # 组件注册系统
│   └── components/        # 组件定义
├── components/            # Vue 组件
│   ├── canvas/           # 画布组件
│   └── panels/           # 面板组件
├── store/                # 状态管理
└── views/                # 页面视图
```

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

## 文档导航

### 核心功能文档
- **[数据绑定系统](./data-binding.md)** - 完整的数据绑定指南
  - 数据源配置
  - 表达式语法
  - 实时订阅
  - 使用示例

- **[数据绑定架构](./data-binding-architecture.md)** - 架构设计说明
  - API 模式 vs Bridge 模式
  - 通信机制
  - 性能对比
  - 最佳实践

- **[Canvas 渲染引擎](./canvas-engine.md)** - Konva 渲染引擎
  - 渲染原理
  - 组件渲染
  - 选择和变换
  - 对齐和吸附

- **[组件开发指南](./component-development.md)** - 自定义组件开发
  - 组件结构
  - 属性定义
  - 事件处理
  - 注册和使用

### 开发历程
- **[开发历程](./development-history.md)** - 各阶段开发总结
  - 阶段一: 基础设施
  - 阶段二: Canvas 渲染引擎
  - 阶段三: 组件库扩展
  - 阶段四: 数据绑定系统

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
{{ data.ds_device.temperature }}

// 条件表达式
{{ data.ds_device.status == 1 ? '运行' : '停止' }}

// 函数调用
{{ format(data.ds_device.temperature, 1) }}
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
  type: 'MyComponent',
  name: '我的组件',
  category: '自定义',
  defaultProps: {
    title: 'Hello'
  },
  propsSchema: {
    title: {
      type: 'text',
      label: '标题'
    }
  }
}
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

更多问题请参考 [数据绑定系统 - 常见问题](./data-binding.md#常见问题)。

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

---

**版本**: 2.0.0  
**最后更新**: 2025-12-08
