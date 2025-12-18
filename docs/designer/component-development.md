# 组件开发指南

## 概述

本指南介绍如何在 InduForge 设计中心开发自定义组件。

## 组件结构

一个完整的组件定义包含以下部分：

```javascript
export default {
  // 基本信息
  type: 'MyComponent',           // 组件类型标识（唯一）
  name: '我的组件',               // 组件显示名称
  category: '自定义',             // 组件分类
  icon: 'star',                  // 组件图标
  tags: ['custom', '自定义'],     // 组件标签
  description: '这是一个自定义组件', // 组件描述
  
  // 默认属性
  defaultProps: {
    title: 'Hello',
    color: '#409EFF'
  },
  
  // 默认样式
  defaultStyle: {
    position: 'absolute',
    left: 0,
    top: 0,
    width: 100,
    height: 100
  },
  
  // 属性 Schema
  propsSchema: {
    title: {
      type: 'text',
      label: '标题',
      default: 'Hello',
      description: '组件标题'
    },
    color: {
      type: 'color',
      label: '颜色',
      default: '#409EFF'
    }
  },
  
  // 事件 Schema
  eventsSchema: {
    click: {
      label: '点击',
      description: '组件被点击时触发'
    }
  }
}
```

## 开发步骤

### 1. 创建组件定义文件

在 `registry/components/` 下创建组件文件：

```bash
# 基础组件
registry/components/basic/MyBasicComponent.js

# UI 组件
registry/components/ui/MyUIComponent.js

# 图表组件
registry/components/charts/MyChartComponent.js

# 工业组件
registry/components/industrial/MyIndustrialComponent.js
```

### 2. 定义组件 Schema

```javascript
// registry/components/custom/MyComponent.js
export default {
  type: 'MyComponent',
  name: '我的组件',
  category: '自定义',
  icon: 'star',
  tags: ['custom'],
  
  defaultProps: {
    title: 'Hello',
    subtitle: 'World',
    color: '#409EFF',
    fontSize: 16
  },
  
  defaultStyle: {
    position: 'absolute',
    left: 0,
    top: 0,
    width: 200,
    height: 100
  },
  
  propsSchema: {
    title: {
      type: 'text',
      label: '标题',
      default: 'Hello'
    },
    subtitle: {
      type: 'text',
      label: '副标题',
      default: 'World'
    },
    color: {
      type: 'color',
      label: '颜色',
      default: '#409EFF'
    },
    fontSize: {
      type: 'number',
      label: '字体大小',
      default: 16,
      min: 12,
      max: 48
    }
  },
  
  eventsSchema: {
    click: {
      label: '点击',
      description: '组件被点击时触发'
    },
    dblclick: {
      label: '双击',
      description: '组件被双击时触发'
    }
  }
}
```

### 3. 注册组件

在 `registry/components/index.js` 中导入并导出：

```javascript
// registry/components/index.js
import MyComponent from './custom/MyComponent'

export default [
  // ... 其他组件
  MyComponent
]
```

### 4. 在应用启动时注册

在 `main.js` 或 `DesignCenter.vue` 中注册：

```javascript
import { registerComponents } from '@/registry'
import customComponents from '@/registry/components'

// 注册所有组件
registerComponents(customComponents)
```

## 属性类型

### 基本类型

```javascript
propsSchema: {
  // 文本
  text: {
    type: 'text',
    label: '文本',
    default: ''
  },
  
  // 数字
  number: {
    type: 'number',
    label: '数字',
    default: 0,
    min: 0,
    max: 100,
    step: 1
  },
  
  // 布尔值
  boolean: {
    type: 'boolean',
    label: '布尔值',
    default: false
  },
  
  // 颜色
  color: {
    type: 'color',
    label: '颜色',
    default: '#409EFF'
  },
  
  // 选择
  select: {
    type: 'select',
    label: '选择',
    default: 'option1',
    options: [
      { label: '选项1', value: 'option1' },
      { label: '选项2', value: 'option2' }
    ]
  },
  
  // 多选
  multiSelect: {
    type: 'multiSelect',
    label: '多选',
    default: [],
    options: [
      { label: '选项1', value: 'option1' },
      { label: '选项2', value: 'option2' }
    ]
  },
  
  // 对象
  object: {
    type: 'object',
    label: '对象',
    default: {}
  },
  
  // 数组
  array: {
    type: 'array',
    label: '数组',
    default: []
  }
}
```

### 高级类型

```javascript
propsSchema: {
  // 表达式
  expression: {
    type: 'expression',
    label: '表达式',
    default: '{{ data.ds_xxx }}'
  },
  
  // 数据源
  dataSource: {
    type: 'dataSource',
    label: '数据源',
    default: ''
  },
  
  // 图片
  image: {
    type: 'image',
    label: '图片',
    default: ''
  },
  
  // 图标
  icon: {
    type: 'icon',
    label: '图标',
    default: ''
  }
}
```

## 组件示例

### 示例1: 简单文本组件

```javascript
export default {
  type: 'SimpleText',
  name: '简单文本',
  category: '基础',
  icon: 'text',
  
  defaultProps: {
    content: 'Hello World',
    color: '#303133',
    fontSize: 14
  },
  
  defaultStyle: {
    position: 'absolute',
    left: 0,
    top: 0,
    width: 100,
    height: 30
  },
  
  propsSchema: {
    content: {
      type: 'text',
      label: '内容',
      default: 'Hello World'
    },
    color: {
      type: 'color',
      label: '颜色',
      default: '#303133'
    },
    fontSize: {
      type: 'number',
      label: '字体大小',
      default: 14,
      min: 12,
      max: 48
    }
  }
}
```

### 示例2: 状态指示器

```javascript
export default {
  type: 'StatusIndicator',
  name: '状态指示器',
  category: '工业',
  icon: 'circle',
  
  defaultProps: {
    status: 0,
    labels: ['停止', '运行', '故障'],
    colors: ['#909399', '#67C23A', '#F56C6C']
  },
  
  defaultStyle: {
    position: 'absolute',
    left: 0,
    top: 0,
    width: 80,
    height: 80
  },
  
  propsSchema: {
    status: {
      type: 'select',
      label: '状态',
      default: 0,
      options: [
        { label: '停止', value: 0 },
        { label: '运行', value: 1 },
        { label: '故障', value: 2 }
      ]
    },
    labels: {
      type: 'array',
      label: '标签',
      default: ['停止', '运行', '故障']
    },
    colors: {
      type: 'array',
      label: '颜色',
      default: ['#909399', '#67C23A', '#F56C6C']
    }
  },
  
  eventsSchema: {
    click: {
      label: '点击',
      description: '状态指示器被点击时触发'
    }
  }
}
```

### 示例3: 数据表格

```javascript
export default {
  type: 'DataTable',
  name: '数据表格',
  category: 'UI',
  icon: 'table',
  
  defaultProps: {
    columns: [
      { prop: 'name', label: '名称' },
      { prop: 'value', label: '值' }
    ],
    data: [],
    stripe: true,
    border: true
  },
  
  defaultStyle: {
    position: 'absolute',
    left: 0,
    top: 0,
    width: 400,
    height: 300
  },
  
  propsSchema: {
    columns: {
      type: 'array',
      label: '列配置',
      default: []
    },
    data: {
      type: 'expression',
      label: '数据',
      default: '{{ data.ds_table }}'
    },
    stripe: {
      type: 'boolean',
      label: '斑马纹',
      default: true
    },
    border: {
      type: 'boolean',
      label: '边框',
      default: true
    }
  },
  
  eventsSchema: {
    rowClick: {
      label: '行点击',
      description: '表格行被点击时触发'
    }
  }
}
```

## 最佳实践

### 1. 命名规范

- 组件类型使用 PascalCase: `MyComponent`
- 组件文件名与类型一致: `MyComponent.js`
- 属性名使用 camelCase: `fontSize`

### 2. 默认值

- 为所有属性提供合理的默认值
- 默认值应该是最常用的配置
- 避免使用 null 或 undefined

### 3. 属性分组

将相关属性分组，提高可维护性：

```javascript
propsSchema: {
  // 基本属性
  title: { ... },
  subtitle: { ... },
  
  // 样式属性
  color: { ... },
  fontSize: { ... },
  
  // 数据属性
  dataSource: { ... },
  dataField: { ... }
}
```

### 4. 文档注释

为组件添加详细的描述和使用说明：

```javascript
export default {
  type: 'MyComponent',
  name: '我的组件',
  description: '这是一个自定义组件，用于显示...',
  
  propsSchema: {
    title: {
      type: 'text',
      label: '标题',
      description: '组件的主标题，支持表达式绑定'
    }
  }
}
```

### 5. 验证

添加属性验证，确保数据正确：

```javascript
propsSchema: {
  fontSize: {
    type: 'number',
    label: '字体大小',
    default: 14,
    min: 12,
    max: 48,
    validator: (value) => {
      return value >= 12 && value <= 48
    }
  }
}
```

## 调试技巧

### 1. 查看组件注册

```javascript
import { getComponent, getAllComponents } from '@/registry'

// 获取单个组件
const component = getComponent('MyComponent')
console.log(component)

// 获取所有组件
const allComponents = getAllComponents()
console.log(allComponents)
```

### 2. 查看组件实例

```javascript
// 在浏览器控制台
const store = useDesignStore()
const component = store.components.find(c => c.id === 'comp_001')
console.log(component)
```

### 3. 调试渲染

在 KonvaRenderer 中添加日志：

```javascript
renderComponent(componentSchema) {
  console.log('Rendering component:', componentSchema)
  // ...
}
```

## 常见问题

### Q: 如何支持嵌套组件？
A: 在组件定义中添加 `children` 属性，并在渲染时递归处理。

### Q: 如何支持插槽？
A: 在组件定义中添加 `slots` 属性，定义插槽名称和内容。

### Q: 如何支持自定义渲染？
A: 在组件定义中添加 `render` 方法，自定义渲染逻辑。

### Q: 如何支持动画？
A: 使用动画系统，在组件定义中添加 `animations` 配置。

## Button 组件（Element Plus）

- 类型：`Button`，定义于 `src/registry/ui/Button.js`
- 支持扩展配置：
  - `styleConfig`：
    - 支持 CSS JSON 或逐行 `key: value` 文本。
    - 支持直接粘贴标准 CSS 代码块（含 `{}`），会注入到 `<head>` 生效，例如：
      ```css
      #myBtn {
        color: red;
      }
      #myBtn span {
        color: red;
      }
      ```
    - 若 CSS 中包含 `#xxx` 且未单独设置 `domId`，会自动提取 `xxx` 赋给按钮 DOM，便于对子级写选择器；也可在 JSON/逐行写法用 `domId: xxx` 显式指定。
  - `advancedConfig`：JSON 或对象字面量，优先级高于属性面板字段，可覆盖按钮属性（`text`、`type`、`size`、`disabled`、`loading`、`plain`、`round`、`circle`、`icon`、`styleConfig` 等）。
  - 其他属性：新增 `domId` 字段，用于显式设置按钮根 DOM 的 `id`（优先级高于 `styleConfig` 自动提取）。
  - 配置弹窗：样式配置 / 详细配置弹窗支持拖拽，可移动到任意位置。
  - `eventsSchema`：开放 Element Plus Button 的常用事件（click、mousedown、mouseup、mouseenter、mouseleave、focus、blur），可在“连接”面板中绑定。

示例（advancedConfig）：

```json
{
  "text": "确认",
  "type": "primary",
  "size": "large",
  "disabled": false,
  "icon": "CircleCheck",
  "styleConfig": {
    "color": "#fff",
    "background": "#409eff",
    "borderRadius": "6px"
  }
}
```

> 解析顺序：属性面板字段 < advancedConfig。请避免在 advancedConfig 中写危险语句，仅使用 JSON/对象字面量。

## 相关资源

- [DSL 设计规范](../dsl-design.md)
- [组件注册系统](../../designer/registry/index.js)
- [基础组件示例](../../designer/registry/components/basic/)
- [Vue 3 文档](https://vuejs.org/)

---

**版本**: 2.0.1  
**最后更新**: 2025-12-18
