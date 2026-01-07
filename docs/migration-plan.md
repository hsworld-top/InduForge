# InduForge 低代码平台迁移与扩展计划

> 基于 KingPortal 功能分析，结合 InduForge 现有架构的迁移方案

---

## ⚠️ 历史文档说明

**本文档为 2025 年编写的 KingPortal 迁移计划，部分内容已过时。**

2026年1月起，设计中心进入**全新重构阶段**，不再基于 KingPortal 迁移，而是采用全新架构设计。

### 当前开发方向

- **[高层设计](./高层设计.md)** - 平台整体架构设计
- **[Designer 重构计划](./designer/refactor/README.md)** - 设计中心重构详细计划

### 已废弃的规划

- Canvas 渲染引擎（Konva.js 纯 Canvas）→ 改为 DOM + Canvas 辅助
- 工业图形库迁移 → 暂缓，后续按需评估
- 原阶段规划 → 被新的重构里程碑替代

> 以下为原始迁移计划，保留作为历史参考。

---

## 一、现状分析

### 已完成的工作 ✅

1. **基础架构**
   - Vue 3 + Pinia + Vue Router
   - Element Plus UI 框架
   - Tailwind CSS 样式系统
   - 微前端架构（IDE + DataCenter + Designer）

2. **Designer 核心功能**
   - 三栏布局（页面树 + 画布 + 属性面板）
   - 基础拖拽系统（vue-draggable-plus）
   - 组件注册机制（registry）
   - 页面管理（创建、删除、重命名）
   - 组件选择和属性编辑
   - 画布缩放和标尺（vue3-sketch-ruler）
   - 撤销/重做预留接口

3. **DSL 设计**
   - 完整的 DSL 2.0 规范文档
   - Page Schema 结构定义
   - Component Schema 结构定义
   - DataSource、Action、Expression 系统设计

4. **后端支持**
   - Node.js + Express
   - 数据库设计（design_pages, design_components 等）

### 待实现的核心功能 🚧


从 KingPortal 需要迁移的功能：

1. **图形组件库**（5000+ 工业图元）
2. **Canvas 渲染引擎**（HT for Web / EaselJS）
3. **数据绑定系统**（表达式链接 + 触发器链接）
4. **动画系统**（闪烁、旋转、缩放等）
5. **图表组件**（ECharts 集成）
6. **UI 组件库**（50+ 组件）
7. **变量系统**（本地变量、工程变量、数据源变量）
8. **脚本编辑器**（Monaco Editor 集成）
9. **数据源管理**（实时订阅、轮询、HTTP）
10. **权限系统**（页面级、组件级、数据级）

---

## 二、技术栈对比与选型

### KingPortal 原技术栈
- jQuery + EasyUI + Element UI
- HT for Web（2D/3D 图形引擎）
- EaselJS（Canvas 图形库）
- ECharts（图表库）
- Ace Editor（代码编辑器）

### InduForge 现有技术栈
- Vue 3 + Pinia + Vue Router ✅
- Element Plus ✅
- Tailwind CSS ✅
- vue-draggable-plus ✅
- vue3-sketch-ruler ✅
- Moveable（组件拖拽调整）✅

### 需要新增的技术栈

| 功能模块 | 推荐方案 | 说明 |
|---------|---------|------|
| Canvas 渲染引擎 | **Konva.js** | 轻量级 Canvas 库，Vue 友好，替代 HT for Web |
| 3D 渲染 | **Three.js** | 成熟的 3D 库，替代 HT 3D |
| 图表库 | **ECharts** | 保持一致，功能强大 |
| 代码编辑器 | **Monaco Editor** | VS Code 同款，已在 DataCenter 使用 |
| 表达式解析 | **expr-eval** | 轻量级表达式解析库 |
| 动画库 | **GSAP** | 专业动画库，性能优秀 |

---

## 三、迁移策略

### 阶段一：完善基础设施（2周）

**目标**：完善现有 Designer 的基础功能

#### 1.1 组件库基础设施


```bash
# 安装依赖
cd InduForge/designer
pnpm add konva vue-konva echarts vue-echarts monaco-editor expr-eval gsap
```

**文件结构扩展**：
```
designer/src/
├── registry/
│   ├── components/
│   │   ├── basic/          # 基础组件（Text, Image, Shape）
│   │   ├── ui/             # UI 组件（Button, Input, Table）
│   │   ├── charts/         # 图表组件（Line, Bar, Pie）
│   │   ├── industrial/     # 工业组件（Motor, Pump, Valve）
│   │   └── layout/         # 布局容器（Panel, Grid, Flex）
│   └── index.js
├── engine/
│   ├── canvas/             # Canvas 渲染引擎
│   │   ├── KonvaRenderer.js
│   │   ├── ComponentRenderer.js
│   │   └── SelectionManager.js
│   ├── binding/            # 数据绑定引擎
│   │   ├── ExpressionEngine.js
│   │   ├── DataBinder.js
│   │   └── TriggerManager.js
│   └── animation/          # 动画引擎
│       ├── AnimationManager.js
│       └── presets.js
├── components/
│   ├── canvas/
│   │   ├── DesignCanvas.vue      # 主画布（已有）
│   │   ├── CanvasRuler.vue       # 标尺（已有）
│   │   ├── ComponentWrapper.vue  # 组件包装器（新增）
│   │   └── SelectionBox.vue      # 选择框（新增）
│   ├── panels/
│   │   ├── PageTree.vue          # 页面树（已有）
│   │   ├── ComponentTree.vue     # 组件树（已有）
│   │   ├── ComponentLibrary.vue  # 组件库（已有）
│   │   ├── PropertyPanel.vue     # 属性面板（已有）
│   │   ├── DataBindingPanel.vue  # 数据绑定面板（新增）
│   │   ├── EventPanel.vue        # 事件面板（新增）
│   │   └── AnimationPanel.vue    # 动画面板（新增）
│   └── editors/
│       ├── ExpressionEditor.vue  # 表达式编辑器（新增）
│       ├── ScriptEditor.vue      # 脚本编辑器（新增）
│       └── StyleEditor.vue       # 样式编辑器（新增）
└── composables/
    ├── useCanvas.js              # 画布操作（已有）
    ├── useSelection.js           # 选择操作（已有）
    ├── useDragDrop.js            # 拖拽操作（已有）
    ├── useHistory.js             # 历史记录（新增）
    ├── useDataBinding.js         # 数据绑定（新增）
    └── useAnimation.js           # 动画控制（新增）
```

#### 1.2 完善组件注册机制

**扩展 `registry/index.js`**：
- 支持组件分类和搜索
- 支持组件预览缩略图
- 支持组件属性 Schema 定义
- 支持组件事件定义

#### 1.3 实现历史记录（撤销/重做）

**创建 `composables/useHistory.js`**：
```javascript
import { ref, computed } from 'vue'

export function useHistory(maxSize = 50) {
  const history = ref([])
  const currentIndex = ref(-1)
  
  const canUndo = computed(() => currentIndex.value > 0)
  const canRedo = computed(() => currentIndex.value < history.value.length - 1)
  
  function push(state) {
    // 删除当前索引之后的历史
    history.value = history.value.slice(0, currentIndex.value + 1)
    // 添加新状态
    history.value.push(JSON.parse(JSON.stringify(state)))
    // 限制历史记录大小
    if (history.value.length > maxSize) {
      history.value.shift()
    } else {
      currentIndex.value++
    }
  }
  
  function undo() {
    if (canUndo.value) {
      currentIndex.value--
      return history.value[currentIndex.value]
    }
  }
  
  function redo() {
    if (canRedo.value) {
      currentIndex.value++
      return history.value[currentIndex.value]
    }
  }
  
  return { canUndo, canRedo, push, undo, redo }
}
```

---

### 阶段二：Canvas 渲染引擎（3周）🚧

**目标**：实现基于 Konva.js 的 Canvas 渲染引擎

**当前状态**：基础渲染已完成，待实现高级功能

**待实现功能**：
- [ ] 容器组件支持子组件拖入
- [ ] 组件右键菜单（复制、删除、插入）
- [ ] 组件层级管理
- [ ] 组件对齐辅助线
- [ ] 组件吸附功能

#### 2.1 安装和配置 Konva

```bash
pnpm add konva vue-konva
```

#### 2.2 创建 Canvas 渲染引擎

**`engine/canvas/KonvaRenderer.js`**：


```javascript
import Konva from 'konva'

export class KonvaRenderer {
  constructor(container, config) {
    this.stage = new Konva.Stage({
      container,
      width: config.width,
      height: config.height,
    })
    
    this.layer = new Konva.Layer()
    this.stage.add(this.layer)
    
    this.components = new Map()
  }
  
  renderComponent(componentSchema) {
    const { type, id, style, props } = componentSchema
    
    // 根据组件类型创建 Konva 节点
    let node
    switch (type) {
      case 'Rectangle':
        node = new Konva.Rect({
          x: style.left,
          y: style.top,
          width: style.width,
          height: style.height,
          fill: props.fill,
          stroke: props.stroke,
          strokeWidth: props.strokeWidth,
        })
        break
      case 'Text':
        node = new Konva.Text({
          x: style.left,
          y: style.top,
          text: props.content,
          fontSize: props.fontSize,
          fill: props.color,
        })
        break
      // ... 其他组件类型
    }
    
    if (node) {
      node.id(id)
      this.components.set(id, node)
      this.layer.add(node)
    }
  }
  
  updateComponent(id, updates) {
    const node = this.components.get(id)
    if (node) {
      node.setAttrs(updates)
      this.layer.batchDraw()
    }
  }
  
  removeComponent(id) {
    const node = this.components.get(id)
    if (node) {
      node.destroy()
      this.components.delete(id)
      this.layer.batchDraw()
    }
  }
  
  clear() {
    this.layer.destroyChildren()
    this.components.clear()
  }
}
```

#### 2.3 重构 DesignCanvas.vue

使用 Konva 替代原有的 DOM 渲染方式，支持：
- 组件拖拽
- 组件缩放
- 组件旋转
- 多选
- 对齐辅助线

---

### 阶段三：基础组件库（3周）✅

**目标**：实现 30+ 基础组件

**状态**：✅ 已完成核心组件（23个），建立完整架构

**完成时间**：2024-12-08

**目录结构**：
```
registry/
├── layout/      # 布局组件（4个）
├── basic/       # 基础组件（7个）
├── ui/          # Element组件（7个）
├── charts/      # 图表组件（5个）
├── industrial/  # 工业组件（预留）
└── index.js     # 注册中心
```

**使用方法**：
```javascript
// 注册所有组件
import { registerAllComponents } from '@/registry/components'
registerAllComponents()

// 或按需注册
import { registerLayoutComponents, registerBasicComponents } from '@/registry/components'
registerLayoutComponents()
registerBasicComponents()

// 使用组件
import { getComponent, createComponentInstance } from '@/registry'
const button = getComponent('Button')
const instance = createComponentInstance('Button', {
  props: { text: '确定' },
  style: { left: 100, top: 100 }
})
```

#### 3.1 基础图形组件（7种）✅

**`registry/basic/`**：
- ✅ Rectangle（矩形）
- ✅ Circle（圆形）
- ✅ Ellipse（椭圆）
- ✅ Line（直线）
- ✅ Polygon（多边形）
- ✅ Text（文本）
- ✅ Image（图片）

#### 3.2 布局组件（5种）✅

**`registry/layout/`** - 5 大布局元素：

1. **盒子容器** (Container) - 通用布局容器，支持 Flex/Grid/Block
2. **行列容器** (Row + Col) - 24 栅格系统
3. **弹性容器** (FlexLayout) - 完整 Flexbox 属性配置
4. **全宽居中** (CenterLayout) - 子组件自动居中
5. **栅格布局** - Row + Col 的栅格模式

**组件列表**：
- ✅ Container（盒子容器）- 蓝色 #409EFF
- ✅ FlexLayout（弹性容器）- 橙色 #E6A23C
- ✅ CenterLayout（全宽居中）- 绿色 #67C23A
- ✅ Row（栅格行）- 红色 #F56C6C
- ✅ Col（栅格列）- 灰色 #909399

**完成时间**：2024-12-08

**功能特性**：
- ✅ 所有容器支持子组件嵌套
- ✅ 支持拖拽、调整尺寸（Col 除外）
- ✅ 完整的样式属性配置（背景、边框、圆角、阴影、内外边距）
- ✅ 不同容器类型显示不同颜色和图标
- ✅ 支持拖放区域可视化
- ✅ 编辑器交互支持（hover 高亮、组件树显示）

**文档**：
- `docs/layout-containers-guide.md` - 完整使用指南

#### 3.3 Element UI 组件（7种）✅

**`registry/ui/`**：
- ✅ Button（按钮）
- ✅ Input（输入框）
- ✅ Select（选择器）
- ✅ Switch（开关）
- ✅ Slider（滑块）
- ✅ Progress（进度条）
- ✅ Table（表格）

#### 3.4 图表组件（5种）✅

**`registry/charts/`**：
- ✅ LineChart（折线图）
- ✅ BarChart（柱状图）
- ✅ PieChart（饼图）
- ✅ GaugeChart（仪表盘）
- ✅ ScatterChart（散点图）

#### 3.5 待扩展组件

**基础图形**（待添加）：
- Polyline（折线）
- SVG（SVG 图形）
- Path（路径）

**组件定义示例**：
```javascript
// registry/basic/Rectangle.js
export default {
  type: 'Rectangle',
  name: '矩形',
  category: '基础组件',
  icon: 'crop-square',
  thumbnail: null,
  tags: ['图形', '矩形', '方形', 'shape'],
  description: '矩形图形，支持填充色、边框、圆角等属性',
  defaultProps: {
    fill: '#409EFF',
    stroke: '#303133',
    strokeWidth: 1,
    cornerRadius: 0,
    opacity: 1
  },
  defaultStyle: {
    position: 'absolute',
    left: 100,
    top: 100,
    width: 120,
    height: 80,
    zIndex: 1
  },
  propsSchema: {
    fill: { 
      type: 'color', 
      label: '填充颜色',
      group: '外观',
      default: '#409EFF'
    },
    stroke: { 
      type: 'color', 
      label: '边框颜色',
      group: '外观',
      default: '#303133'
    },
    strokeWidth: { 
      type: 'number', 
      label: '边框宽度',
      group: '外观',
      min: 0, 
      max: 20,
      unit: 'px',
      default: 1
    },
    cornerRadius: { 
      type: 'number', 
      label: '圆角',
      group: '外观',
      min: 0, 
      max: 100,
      unit: 'px',
      default: 0
    }
  },
  eventsSchema: {
    click: { label: '点击', description: '鼠标点击时触发' }
  },
  container: false,
  version: '1.0.0'
}
```

**待扩展 UI 组件**：
- Tree（树形控件）
- Tabs（标签页）
- Dialog（对话框）
- DatePicker（日期选择器）
- TimePicker（时间选择器）
- Upload（上传）
- Form（表单）
- ... 等

**待扩展图表组件**：
- RadarChart（雷达图）
- HeatmapChart（热力图）
- TreeChart（树图）
- SankeyChart（桑基图）
- FunnelChart（漏斗图）
- ... 等

---

### 阶段四：数据绑定系统（4周）

**目标**：实现表达式绑定和数据源订阅

#### 4.1 表达式引擎

**`engine/binding/ExpressionEngine.js`**：
```javascript
import { Parser } from 'expr-eval'

export class ExpressionEngine {
  constructor() {
    this.parser = new Parser()
    this.context = {}
  }
  
  setContext(context) {
    this.context = context
  }
  
  evaluate(expression) {
    try {
      // 移除 {{ }} 包裹
      const cleanExpr = expression.replace(/^\{\{|\}\}$/g, '').trim()
      return this.parser.evaluate(cleanExpr, this.context)
    } catch (error) {
      console.error('Expression evaluation error:', error)
      return null
    }
  }
  
  // 内置函数
  registerFunction(name, fn) {
    this.parser.functions[name] = fn
  }
}

// 注册内置函数
const engine = new ExpressionEngine()
engine.registerFunction('format', (value, decimals) => {
  return Number(value).toFixed(decimals)
})
engine.registerFunction('if', (condition, trueVal, falseVal) => {
  return condition ? trueVal : falseVal
})

export default engine
```

#### 4.2 数据绑定管理器

**`engine/binding/DataBinder.js`**：
```javascript
import { watch } from 'vue'
import expressionEngine from './ExpressionEngine'

export class DataBinder {
  constructor(store) {
    this.store = store
    this.watchers = new Map()
  }
  
  bind(componentId, bindings) {
    // bindings: { 'props.value': '{{ data.ds_temp.value }}' }
    Object.entries(bindings).forEach(([path, expression]) => {
      const watcher = watch(
        () => this.evaluateExpression(expression),
        (newValue) => {
          this.updateComponentProperty(componentId, path, newValue)
        },
        { immediate: true }
      )
      
      const key = `${componentId}:${path}`
      this.watchers.set(key, watcher)
    })
  }
  
  unbind(componentId) {
    for (const [key, watcher] of this.watchers.entries()) {
      if (key.startsWith(componentId + ':')) {
        watcher()
        this.watchers.delete(key)
      }
    }
  }
  
  evaluateExpression(expression) {
    // 构建上下文
    const context = {
      vars: this.store.currentPage?.variables || {},
      data: this.store.dataSources || {},
      $user: this.store.user || {},
    }
    
    expressionEngine.setContext(context)
    return expressionEngine.evaluate(expression)
  }
  
  updateComponentProperty(componentId, path, value) {
    // path: 'props.value' or 'style.color'
    const [section, key] = path.split('.')
    this.store.updateComponent(componentId, {
      [section]: { [key]: value }
    })
  }
}
```

#### 4.3 数据源管理

**扩展 `store/design.js`**：
```javascript
// 添加数据源状态
state: () => ({
  // ... 现有状态
  dataSources: {},        // 数据源数据
  dataSourceConfigs: [],  // 数据源配置
})

// 添加数据源操作
actions: {
  async startDataSource(dataSourceId) {
    const config = this.dataSourceConfigs.find(ds => ds.id === dataSourceId)
    if (!config) return
    
    if (config.type === 'dataCenter' && config.config.mode === 'subscription') {
      // WebSocket 订阅
      this.subscribeDataCenter(config)
    } else if (config.mode === 'poll') {
      // 轮询
      this.pollDataSource(config)
    }
  },
  
  subscribeDataCenter(config) {
    const ws = new WebSocket('ws://localhost:9099/ws/data')
    ws.onopen = () => {
      ws.send(JSON.stringify({
        type: 'subscribe',
        tags: config.config.tags
      }))
    }
    ws.onmessage = (event) => {
      const data = JSON.parse(event.data)
      this.dataSources[config.id] = data
    }
  }
}
```

---

### 阶段五：工业图形库（4周）

**目标**：迁移 KingPortal 的 5000+ 工业图元

#### 5.1 图元数据迁移

从 KingPortal 的 `Predefined/graphicGroup/` 提取 JSON 配置：

**迁移脚本**：
```javascript
// scripts/migrate-graphics.js
const fs = require('fs')
const path = require('path')

const sourceDir = '../KingPortal/extension/Predefined/graphicGroup'
const targetDir = './public/graphics'

// 读取所有图形组
const groups = fs.readdirSync(sourceDir)

groups.forEach(group => {
  const groupPath = path.join(sourceDir, group)
  const files = fs.readdirSync(groupPath)
  
  files.forEach(file => {
    if (file.endsWith('.json')) {
      const content = fs.readFileSync(path.join(groupPath, file), 'utf-8')
      const graphic = JSON.parse(content)
      
      // 转换为 InduForge 格式
      const converted = convertGraphic(graphic)
      
      // 保存
      const targetPath = path.join(targetDir, group, file)
      fs.mkdirSync(path.dirname(targetPath), { recursive: true })
      fs.writeFileSync(targetPath, JSON.stringify(converted, null, 2))
    }
  })
})

function convertGraphic(graphic) {
  // 转换逻辑：KingPortal 格式 -> InduForge DSL 格式
  return {
    type: 'IndustrialGraphic',
    name: graphic.name,
    category: graphic.category,
    thumbnail: graphic.thumbnail,
    components: convertComponents(graphic.children)
  }
}
```

#### 5.2 图形组件渲染

**`registry/components/industrial/IndustrialGraphic.js`**：
```javascript
export default {
  type: 'IndustrialGraphic',
  name: '工业图元',
  category: '工业组件',
  render(props, context) {
    // 渲染复合图形
    const { graphicId } = props
    const graphic = context.getGraphic(graphicId)
    
    return graphic.components.map(comp => {
      return context.renderComponent(comp)
    })
  }
}
```

#### 5.3 图形库面板

**`components/panels/GraphicLibrary.vue`**：
- 分类展示（三维按钮、阀门管道、泵电机等）
- 搜索和过滤
- 预览缩略图
- 拖拽到画布

---

### 阶段六：动画系统（2周）

**目标**：实现组件动画效果

#### 6.1 动画管理器

**`engine/animation/AnimationManager.js`**：
```javascript
import gsap from 'gsap'

export class AnimationManager {
  constructor() {
    this.animations = new Map()
  }
  
  play(componentId, animationConfig) {
    const { type, config, condition } = animationConfig
    
    // 检查条件
    if (condition && !this.evaluateCondition(condition)) {
      return
    }
    
    const target = document.getElementById(componentId)
    if (!target) return
    
    let animation
    switch (type) {
      case 'rotate':
        animation = gsap.to(target, {
          rotation: 360,
          duration: config.duration / 1000,
          repeat: config.iterations === 'infinite' ? -1 : config.iterations,
          ease: config.easing || 'linear'
        })
        break
      case 'flash':
        animation = gsap.to(target, {
          opacity: 0,
          duration: config.duration / 1000 / 2,
          repeat: config.iterations === 'infinite' ? -1 : config.iterations * 2,
          yoyo: true
        })
        break
      // ... 其他动画类型
    }
    
    this.animations.set(componentId, animation)
  }
  
  stop(componentId) {
    const animation = this.animations.get(componentId)
    if (animation) {
      animation.kill()
      this.animations.delete(componentId)
    }
  }
}
```

#### 6.2 动画预设

**`engine/animation/presets.js`**：
```javascript
export const animationPresets = {
  rotate: {
    name: '旋转',
    config: {
      duration: 2000,
      iterations: 'infinite',
      direction: 'normal',
      easing: 'linear'
    }
  },
  flash: {
    name: '闪烁',
    config: {
      duration: 500,
      iterations: 'infinite',
      colors: ['#ff0000', 'transparent']
    }
  },
  pulse: {
    name: '脉冲',
    config: {
      duration: 1000,
      iterations: 'infinite',
      scale: 1.1
    }
  }
}
```

---

### 阶段七：脚本编辑器（2周）

**目标**：集成 Monaco Editor

#### 7.1 Monaco Editor 集成

```bash
pnpm add monaco-editor
```

**`components/editors/ScriptEditor.vue`**：
```vue
<template>
  <div ref="editorRef" class="script-editor"></div>
</template>

<script setup>
import { ref, onMounted, watch } from 'vue'
import * as monaco from 'monaco-editor'

const props = defineProps({
  modelValue: String,
  language: { type: String, default: 'javascript' }
})

const emit = defineEmits(['update:modelValue'])

const editorRef = ref(null)
let editor = null

onMounted(() => {
  editor = monaco.editor.create(editorRef.value, {
    value: props.modelValue || '',
    language: props.language,
    theme: 'vs-dark',
    minimap: { enabled: false },
    automaticLayout: true
  })
  
  editor.onDidChangeModelContent(() => {
    emit('update:modelValue', editor.getValue())
  })
})

watch(() => props.modelValue, (newValue) => {
  if (editor && newValue !== editor.getValue()) {
    editor.setValue(newValue || '')
  }
})
</script>

<style scoped>
.script-editor {
  width: 100%;
  height: 400px;
}
</style>
```

#### 7.2 脚本执行沙箱

**`engine/script/ScriptRunner.js`**：
```javascript
export class ScriptRunner {
  constructor(context) {
    this.context = context
  }
  
  run(code) {
    try {
      // 创建沙箱环境
      const sandbox = {
        vars: this.context.vars,
        data: this.context.data,
        $api: this.context.api,
        console: {
          log: (...args) => console.log('[Script]', ...args)
        }
      }
      
      // 使用 Function 构造函数执行代码
      const fn = new Function(...Object.keys(sandbox), code)
      return fn(...Object.values(sandbox))
    } catch (error) {
      console.error('Script execution error:', error)
      throw error
    }
  }
}
```

---

### 阶段四：高级交互功能（2-3周）✅

**目标**：实现容器嵌套、右键菜单等高级交互功能

**状态**：✅ 已完成核心功能

**完成时间**：2024-12-08

**已实现功能**：
- ✅ 右键菜单组件（ContextMenu.vue）
- ✅ 快捷键支持（useKeyboard.js）
- ✅ 组件层级管理（置顶、置底、上移、下移）
- ✅ 对齐辅助线系统（AlignmentGuides.js）
- ✅ 容器组件基础结构（支持拖放区域标记）
- ✅ 复制/粘贴/删除功能
- ✅ 右键菜单事件集成

**待完善功能**：
- ⏳ 容器组件子组件拖入逻辑（需要完善拖放处理）
- ⏳ 组件在前面/后面插入功能

**实现文件**：
- `components/canvas/ContextMenu.vue` - 右键菜单组件
- `composables/useKeyboard.js` - 快捷键处理
- `engine/canvas/AlignmentGuides.js` - 对齐辅助线
- `store/design.js` - 层级管理方法
- `engine/canvas/KonvaRenderer.js` - 容器组件渲染
- `engine/canvas/CanvasEngine.js` - 对齐辅助线集成

#### 4.1 容器组件支持

**功能需求**：
- 容器组件（Container, Row, Col）可以接收子组件拖入
- 子组件在容器内部可以自由布局
- 支持多层嵌套
- 容器边界高亮显示

**实现方案**：
```javascript
// 1. 在 KonvaRenderer 中为容器组件添加拖放区域
createContainer(schema) {
  const group = new Konva.Group({
    x: style.left,
    y: style.top,
    draggable: true,
  })
  
  // 容器背景
  const bg = new Konva.Rect({
    width: style.width,
    height: style.height,
    fill: 'transparent',
    stroke: '#409EFF',
    strokeWidth: 2,
    dash: [10, 5],
  })
  
  // 拖放区域
  const dropZone = new Konva.Rect({
    width: style.width,
    height: style.height,
    fill: 'rgba(64, 158, 255, 0.1)',
    visible: false, // 默认隐藏
  })
  
  group.add(bg)
  group.add(dropZone)
  
  // 监听拖放事件
  group.on('dragover', () => {
    dropZone.visible(true)
  })
  
  group.on('dragleave', () => {
    dropZone.visible(false)
  })
  
  group.on('drop', (e) => {
    // 处理子组件拖入
    const childComponent = e.dataTransfer.getData('component')
    this.addChildToContainer(group.id(), childComponent)
  })
  
  return group
}

// 2. 在 DesignCanvas 中处理容器拖放
function handleDropToContainer(containerId, component) {
  // 计算相对于容器的位置
  const container = canvasEngine.getComponent(containerId)
  const relativePos = {
    left: component.style.left - container.style.left,
    top: component.style.top - container.style.top,
  }
  
  // 添加为子组件
  designStore.addComponent(component, containerId)
}
```

#### 4.2 右键菜单功能

**功能需求**：
- 右键点击组件显示上下文菜单
- 支持复制、删除、插入等操作
- 支持层级调整（置顶、置底、上移、下移）

**实现方案**：
```javascript
// 1. 创建右键菜单组件
// components/canvas/ContextMenu.vue
<template>
  <div 
    v-show="visible"
    class="context-menu"
    :style="{ left: x + 'px', top: y + 'px' }"
  >
    <div class="menu-item" @click="handleCopy">
      <el-icon><DocumentCopy /></el-icon>
      <span>复制</span>
      <span class="shortcut">Ctrl+C</span>
    </div>
    <div class="menu-item" @click="handleDelete">
      <el-icon><Delete /></el-icon>
      <span>删除</span>
      <span class="shortcut">Delete</span>
    </div>
    <div class="menu-divider"></div>
    <div class="menu-item" @click="handleBringToFront">
      <el-icon><Top /></el-icon>
      <span>置于顶层</span>
    </div>
    <div class="menu-item" @click="handleSendToBack">
      <el-icon><Bottom /></el-icon>
      <span>置于底层</span>
    </div>
    <div class="menu-divider"></div>
    <div class="menu-item" @click="handleInsertBefore">
      <el-icon><Plus /></el-icon>
      <span>在前面插入</span>
    </div>
    <div class="menu-item" @click="handleInsertAfter">
      <el-icon><Plus /></el-icon>
      <span>在后面插入</span>
    </div>
  </div>
</template>

// 2. 在 CanvasEngine 中集成右键菜单
canvasEngine.on('component:contextmenu', ({ id, x, y }) => {
  showContextMenu(id, x, y)
})
```

#### 4.3 组件层级管理

**功能需求**：
- 支持调整组件的 z-index
- 支持置顶、置底、上移、下移操作
- 在组件树中显示层级关系

**实现方案**：
```javascript
// store/design.js
actions: {
  // 置于顶层
  bringToFront(componentId) {
    const component = this.getComponentById(componentId)
    if (!component) return
    
    const siblings = this.getSiblings(componentId)
    const maxZIndex = Math.max(...siblings.map(c => c.style.zIndex || 0))
    
    this.updateComponent(componentId, {
      style: { zIndex: maxZIndex + 1 }
    })
  },
  
  // 置于底层
  sendToBack(componentId) {
    const component = this.getComponentById(componentId)
    if (!component) return
    
    this.updateComponent(componentId, {
      style: { zIndex: 0 }
    })
    
    // 调整其他组件的 zIndex
    const siblings = this.getSiblings(componentId)
    siblings.forEach(sibling => {
      if (sibling.id !== componentId && sibling.style.zIndex > 0) {
        this.updateComponent(sibling.id, {
          style: { zIndex: sibling.style.zIndex + 1 }
        })
      }
    })
  },
  
  // 上移一层
  moveUp(componentId) {
    const component = this.getComponentById(componentId)
    if (!component) return
    
    const currentZIndex = component.style.zIndex || 0
    this.updateComponent(componentId, {
      style: { zIndex: currentZIndex + 1 }
    })
  },
  
  // 下移一层
  moveDown(componentId) {
    const component = this.getComponentById(componentId)
    if (!component) return
    
    const currentZIndex = component.style.zIndex || 0
    if (currentZIndex > 0) {
      this.updateComponent(componentId, {
        style: { zIndex: currentZIndex - 1 }
      })
    }
  }
}
```

#### 4.4 组件对齐辅助线

**功能需求**：
- 拖动组件时显示对齐辅助线
- 支持与其他组件对齐（左、右、上、下、中心）
- 支持与画布边界对齐

**实现方案**：
```javascript
// engine/canvas/AlignmentGuides.js
export class AlignmentGuides {
  constructor(stage, layer) {
    this.stage = stage
    this.layer = layer
    this.guides = []
    this.threshold = 5 // 对齐阈值（像素）
  }
  
  // 显示辅助线
  show(movingNode, allNodes) {
    this.clear()
    
    const movingBox = movingNode.getClientRect()
    
    allNodes.forEach(node => {
      if (node === movingNode) return
      
      const box = node.getClientRect()
      
      // 检查左对齐
      if (Math.abs(movingBox.x - box.x) < this.threshold) {
        this.drawVerticalLine(box.x)
        movingNode.x(box.x)
      }
      
      // 检查右对齐
      if (Math.abs(movingBox.x + movingBox.width - (box.x + box.width)) < this.threshold) {
        this.drawVerticalLine(box.x + box.width)
        movingNode.x(box.x + box.width - movingBox.width)
      }
      
      // 检查顶部对齐
      if (Math.abs(movingBox.y - box.y) < this.threshold) {
        this.drawHorizontalLine(box.y)
        movingNode.y(box.y)
      }
      
      // 检查底部对齐
      if (Math.abs(movingBox.y + movingBox.height - (box.y + box.height)) < this.threshold) {
        this.drawHorizontalLine(box.y + box.height)
        movingNode.y(box.y + box.height - movingBox.height)
      }
      
      // 检查中心对齐
      const movingCenterX = movingBox.x + movingBox.width / 2
      const centerX = box.x + box.width / 2
      if (Math.abs(movingCenterX - centerX) < this.threshold) {
        this.drawVerticalLine(centerX)
        movingNode.x(centerX - movingBox.width / 2)
      }
    })
  }
  
  // 绘制垂直辅助线
  drawVerticalLine(x) {
    const line = new Konva.Line({
      points: [x, 0, x, this.stage.height()],
      stroke: '#FF4081',
      strokeWidth: 1,
      dash: [4, 4],
    })
    this.guides.push(line)
    this.layer.add(line)
  }
  
  // 绘制水平辅助线
  drawHorizontalLine(y) {
    const line = new Konva.Line({
      points: [0, y, this.stage.width(), y],
      stroke: '#FF4081',
      strokeWidth: 1,
      dash: [4, 4],
    })
    this.guides.push(line)
    this.layer.add(line)
  }
  
  // 清除辅助线
  clear() {
    this.guides.forEach(guide => guide.destroy())
    this.guides = []
  }
}
```

#### 4.5 快捷键支持

**功能需求**：
- Ctrl+C: 复制
- Ctrl+V: 粘贴
- Delete: 删除
- Ctrl+Z: 撤销
- Ctrl+Y: 重做
- Ctrl+D: 复制并粘贴

**实现方案**：
```javascript
// composables/useKeyboard.js
export function useKeyboard() {
  const designStore = useDesignStore()
  
  onMounted(() => {
    document.addEventListener('keydown', handleKeyDown)
  })
  
  onUnmounted(() => {
    document.removeEventListener('keydown', handleKeyDown)
  })
  
  function handleKeyDown(e) {
    // 忽略输入框中的按键
    if (e.target.tagName === 'INPUT' || e.target.tagName === 'TEXTAREA') {
      return
    }
    
    // Ctrl+C: 复制
    if (e.ctrlKey && e.key === 'c') {
      e.preventDefault()
      designStore.copySelectedComponent()
    }
    
    // Ctrl+V: 粘贴
    if (e.ctrlKey && e.key === 'v') {
      e.preventDefault()
      designStore.pasteComponent()
    }
    
    // Delete: 删除
    if (e.key === 'Delete') {
      e.preventDefault()
      designStore.deleteSelectedComponent()
    }
    
    // Ctrl+Z: 撤销
    if (e.ctrlKey && e.key === 'z') {
      e.preventDefault()
      designStore.undo()
    }
    
    // Ctrl+Y: 重做
    if (e.ctrlKey && e.key === 'y') {
      e.preventDefault()
      designStore.redo()
    }
    
    // Ctrl+D: 复制并粘贴
    if (e.ctrlKey && e.key === 'd') {
      e.preventDefault()
      designStore.duplicateSelectedComponent()
    }
  }
}
```

---

## 四、关键技术实现

### 4.1 组件拖拽和调整

使用 Moveable 库实现组件的拖拽、缩放、旋转：

```vue
<template>
  <Moveable
    :target="selectedElement"
    :draggable="true"
    :resizable="true"
    :rotatable="true"
    @drag="handleDrag"
    @resize="handleResize"
    @rotate="handleRotate"
  />
</template>
```

### 4.2 对齐辅助线

使用 vue3-sketch-ruler 实现标尺和辅助线：

```vue
<template>
  <SketchRule
    :scale="scale"
    :width="canvasWidth"
    :height="canvasHeight"
    :start-x="scrollLeft"
    :start-y="scrollTop"
    @guide-line-change="handleGuideLineChange"
  />
</template>
```

### 4.3 实时数据订阅

使用 WebSocket 实现实时数据订阅：

```javascript
// composables/useDataSource.js
export function useDataSource(config) {
  const data = ref(null)
  let ws = null
  
  function connect() {
    ws = new WebSocket(config.url)
    
    ws.onopen = () => {
      ws.send(JSON.stringify({
        type: 'subscribe',
        tags: config.tags
      }))
    }
    
    ws.onmessage = (event) => {
      data.value = JSON.parse(event.data)
    }
    
    ws.onerror = (error) => {
      console.error('WebSocket error:', error)
    }
  }
  
  function disconnect() {
    if (ws) {
      ws.close()
      ws = null
    }
  }
  
  onMounted(connect)
  onUnmounted(disconnect)
  
  return { data }
}
```

---

## 五、性能优化

### 5.1 虚拟滚动

对于大量组件的列表（如组件库、页面树），使用虚拟滚动：

```bash
pnpm add vue-virtual-scroller
```

### 5.2 Canvas 性能优化

- 使用离屏 Canvas 预渲染
- 按需渲染可视区域内的组件
- 使用 requestAnimationFrame 优化动画
- 组件缓存和复用

### 5.3 数据绑定优化

- 使用 computed 缓存计算结果
- 避免深层响应式对象
- 使用 shallowRef 优化大对象

---

## 六、测试策略

### 6.1 单元测试

使用 Vitest 进行单元测试：

```javascript
// registry/__tests__/registry.test.js
import { describe, it, expect } from 'vitest'
import { registerComponent, getComponent } from '../index'

describe('Component Registry', () => {
  it('should register a component', () => {
    const definition = {
      type: 'TestComponent',
      name: 'Test',
      category: 'Test'
    }
    registerComponent(definition)
    expect(getComponent('TestComponent')).toEqual(definition)
  })
})
```

### 6.2 集成测试

使用 @vue/test-utils 进行组件测试：

```javascript
// components/__tests__/DesignCanvas.test.js
import { mount } from '@vue/test-utils'
import DesignCanvas from '../canvas/DesignCanvas.vue'

describe('DesignCanvas', () => {
  it('should render components', () => {
    const wrapper = mount(DesignCanvas, {
      props: {
        components: [
          { id: '1', type: 'Rectangle', style: { left: 0, top: 0 } }
        ]
      }
    })
    expect(wrapper.find('.canvas-component').exists()).toBe(true)
  })
})
```

---

## 七、部署和发布

### 7.1 构建优化

```javascript
// vite.config.js
export default {
  build: {
    rollupOptions: {
      output: {
        manualChunks: {
          'vendor': ['vue', 'vue-router', 'pinia'],
          'ui': ['element-plus'],
          'canvas': ['konva', 'vue-konva'],
          'charts': ['echarts', 'vue-echarts']
        }
      }
    }
  }
}
```

### 7.2 CDN 加速

将静态资源（图形库、图标）上传到 CDN：

```javascript
// vite.config.js
export default {
  base: process.env.NODE_ENV === 'production' 
    ? 'https://cdn.example.com/designer/' 
    : '/'
}
```

---

## 八、时间规划

| 阶段 | 任务 | 工期 | 人力 |
|------|------|------|------|
| 阶段一 | 完善基础设施 | 2周 | 1人 |
| 阶段二 | Canvas 渲染引擎 | 3周 | 2人 |
| 阶段三 | 基础组件库 | 3周 | 2人 |
| 阶段四 | 数据绑定系统 | 4周 | 2人 |
| 阶段五 | 工业图形库 | 4周 | 1人 |
| 阶段六 | 动画系统 | 2周 | 1人 |
| 阶段七 | 脚本编辑器 | 2周 | 1人 |
| **总计** | | **20周** | **2-3人** |

---

## 九、风险和挑战

### 9.1 技术风险

1. **Canvas 性能**：大量组件渲染可能导致性能问题
   - 解决方案：虚拟渲染、离屏 Canvas、Web Worker

2. **数据绑定复杂度**：表达式解析和依赖追踪
   - 解决方案：使用成熟的表达式库，限制表达式复杂度

3. **浏览器兼容性**：Canvas、WebSocket 等特性
   - 解决方案：使用 polyfill，明确支持的浏览器版本

### 9.2 业务风险

1. **图形库迁移**：5000+ 图元的数据转换
   - 解决方案：编写自动化迁移脚本，分批验证

2. **用户习惯**：从 KingPortal 迁移的用户适应新界面
   - 解决方案：保持相似的交互方式，提供迁移指南

---

## 十、后续规划

### 10.1 短期目标（3个月）

- 完成基础组件库和 Canvas 引擎
- 实现数据绑定和动画系统
- 迁移核心工业图形库（1000+）

### 10.2 中期目标（6个月）

- 完成全部工业图形库迁移
- 实现协同编辑功能
- 性能优化和稳定性提升

### 10.3 长期目标（1年）

- 支持自定义组件开发
- 提供组件市场
- 支持移动端设计器
- AI 辅助设计功能

---

## 附录：参考资料

- [Konva.js 文档](https://konvajs.org/)
- [ECharts 文档](https://echarts.apache.org/)
- [Monaco Editor 文档](https://microsoft.github.io/monaco-editor/)
- [GSAP 文档](https://greensock.com/gsap/)
- [Vue 3 文档](https://vuejs.org/)
- [Pinia 文档](https://pinia.vuejs.org/)
