# Canvas 渲染引擎

## 概述

设计中心使用基于 Konva.js 的 Canvas 渲染引擎，提供高性能的组件渲染和交互能力。

## 核心模块

### 1. KonvaRenderer (渲染器)

负责将组件 DSL 转换为 Konva 节点并渲染到 Canvas。

**主要功能**:
- 组件渲染
- 样式应用
- 事件绑定
- 节点更新和删除

**支持的组件类型**:
- Rectangle (矩形)
- Circle (圆形)
- Text (文本)
- Image (图片)
- Line (线条)
- Ellipse (椭圆)

### 2. SelectionManager (选择管理器)

管理组件的选择状态和变换操作。

**主要功能**:
- 单选和多选
- 框选
- 拖拽移动
- 缩放调整
- 旋转
- 对齐和分布

### 3. GuideLineManager (辅助线管理器)

提供智能对齐和吸附功能。

**主要功能**:
- 实时对齐检测
- 智能吸附
- 可视化辅助线
- 可配置吸附阈值

### 4. CanvasEngine (统一引擎)

整合所有模块，提供统一的 API 接口。

## 使用示例

### 初始化引擎

```javascript
import { CanvasEngine } from '@/engine/canvas'

const canvasEngine = new CanvasEngine(containerElement, {
  width: 1920,
  height: 1080
})
```

### 渲染组件

```javascript
// 渲染单个组件
canvasEngine.renderComponent({
  id: 'rect_001',
  type: 'Rectangle',
  style: {
    left: 100,
    top: 100,
    width: 200,
    height: 150
  },
  props: {
    fill: '#409EFF',
    stroke: '#303133',
    strokeWidth: 2
  }
})

// 批量渲染
canvasEngine.renderComponents([
  { id: 'rect_001', type: 'Rectangle', ... },
  { id: 'circle_001', type: 'Circle', ... }
])
```

### 选择和操作

```javascript
// 选中组件
canvasEngine.selectComponents('rect_001')

// 多选
canvasEngine.selectComponents(['rect_001', 'circle_001'])

// 获取选中的组件
const selectedIds = canvasEngine.getSelectedIds()

// 删除选中的组件
const deletedIds = canvasEngine.deleteSelected()
```

### 对齐和分布

```javascript
// 左对齐
canvasEngine.alignSelected('left')

// 水平居中
canvasEngine.alignSelected('center-h')

// 水平分布
canvasEngine.distributeSelected('horizontal')
```

### 监听事件

```javascript
// 监听选择变化
canvasEngine.on('selection:change', ({ ids }) => {
  console.log('Selected:', ids)
})

// 监听组件更新
canvasEngine.on('component:update', ({ id, updates }) => {
  console.log('Component updated:', id, updates)
})

// 监听组件点击
canvasEngine.on('component:click', ({ id }) => {
  console.log('Component clicked:', id)
})
```

## 架构设计

### Konva 层级结构

```
Stage (舞台)
  └─ Layer (图层)
       ├─ mainLayer (主图层) - 渲染组件
       └─ selectionLayer (选择层) - 渲染选择框、辅助线

每个组件对应一个 Konva.Node:
- Konva.Rect (矩形)
- Konva.Circle (圆形)
- Konva.Text (文本)
- Konva.Image (图片)
- Konva.Line (线条)
- Konva.Ellipse (椭圆)
```

### 事件流

```
Konva 事件 → CanvasEngine → Store → Vue 组件

例如：拖拽事件
1. Konva.Node.on('dragend')
2. CanvasEngine.emitEvent('component:update')
3. Store.updateComponent()
4. Vue 响应式更新
5. Store.saveHistory()
```

## 性能优化

### 已实现的优化

1. **批量绘制**: 使用 `layer.batchDraw()` 减少重绘次数
2. **事件委托**: 在 Layer 级别处理事件
3. **按需更新**: 只更新变化的属性
4. **离屏渲染**: Konva 自动优化

### 待优化项

1. **增量渲染**: 当前是全量重渲染，需要改为增量更新
2. **虚拟滚动**: 大量组件时使用虚拟滚动
3. **离屏 Canvas**: 复杂组件使用离屏 Canvas
4. **Web Worker**: 将计算密集型任务移到 Worker

## 交互功能

### 拖拽和吸附

```
1. 用户拖拽组件
   ↓
2. GuideLineManager 检测对齐
   ↓
3. 显示红色辅助线
   ↓
4. 自动吸附到对齐位置
   ↓
5. 拖拽结束，隐藏辅助线
   ↓
6. 触发 component:update 事件
   ↓
7. Store 更新组件位置
   ↓
8. 保存历史记录
```

### 框选多选

```
1. 用户在空白处按下鼠标
   ↓
2. 创建选择框（蓝色半透明矩形）
   ↓
3. 拖动鼠标，选择框跟随
   ↓
4. 释放鼠标
   ↓
5. 检测选择框内的组件
   ↓
6. 选中所有相交的组件
   ↓
7. 显示 Transformer（8个控制点）
   ↓
8. 可以批量拖拽、缩放、旋转
```

### 对齐和分布

**对齐操作**:
- 左对齐: 所有组件左边缘对齐到最左边的组件
- 右对齐: 所有组件右边缘对齐到最右边的组件
- 顶对齐: 所有组件顶边对齐到最上面的组件
- 底对齐: 所有组件底边对齐到最下面的组件
- 水平居中: 所有组件水平中心对齐
- 垂直居中: 所有组件垂直中心对齐

**分布操作**:
- 水平分布: 组件在水平方向上均匀分布
- 垂直分布: 组件在垂直方向上均匀分布

## API 文档

### CanvasEngine

#### 构造函数

```javascript
new CanvasEngine(container, config)
```

**参数**:
- `container`: DOM 元素，Canvas 容器
- `config`: 配置对象
  - `width`: 画布宽度
  - `height`: 画布高度

#### 方法

**渲染相关**:
- `renderComponent(componentSchema)` - 渲染单个组件
- `renderComponents(componentSchemas)` - 批量渲染组件
- `updateComponent(id, updates)` - 更新组件
- `removeComponent(id)` - 删除组件
- `clear()` - 清空画布

**选择相关**:
- `selectComponents(ids)` - 选中组件
- `getSelectedIds()` - 获取选中的组件 ID
- `clearSelection()` - 清空选择
- `deleteSelected()` - 删除选中的组件

**对齐相关**:
- `alignSelected(type)` - 对齐选中的组件
  - type: 'left' | 'right' | 'top' | 'bottom' | 'center-h' | 'center-v'
- `distributeSelected(type)` - 分布选中的组件
  - type: 'horizontal' | 'vertical'

**缩放相关**:
- `setScale(scale)` - 设置缩放比例
- `getScale()` - 获取缩放比例

**导出相关**:
- `toDataURL(options)` - 导出为图片
  - options.mimeType: 'image/png' | 'image/jpeg'
  - options.quality: 0-1

**事件相关**:
- `on(event, handler)` - 监听事件
- `off(event, handler)` - 取消监听
- `emit(event, data)` - 触发事件

#### 事件

- `selection:change` - 选择变化
- `component:update` - 组件更新
- `component:click` - 组件点击
- `component:dragstart` - 拖拽开始
- `component:dragmove` - 拖拽中
- `component:dragend` - 拖拽结束
- `transform:end` - 变换结束

## 常见问题

### Q: 如何提高渲染性能？
A: 
1. 减少组件数量
2. 使用批量渲染
3. 避免频繁更新
4. 使用虚拟滚动

### Q: 如何自定义组件渲染？
A: 在 KonvaRenderer 中添加新的组件类型处理逻辑。

### Q: 如何调试 Canvas 渲染？
A: 使用 Konva DevTools 或在浏览器控制台查看 Konva 对象。

### Q: 如何导出高清图片？
A: 使用 `toDataURL({ quality: 1 })` 并设置合适的 mimeType。

## 相关资源

- [Konva.js 官方文档](https://konvajs.org/)
- [Konva.js API 文档](https://konvajs.org/api/Konva.html)
- [阶段二完成报告](../../designer/PHASE2_COMPLETED.md)
- [阶段二总结](../../designer/PHASE2_SUMMARY.md)

---

**版本**: 2.0.0  
**最后更新**: 2025-12-08
