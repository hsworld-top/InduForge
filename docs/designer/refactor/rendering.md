# 渲染架构（Rendering Architecture）

本文档描述 Designer 的渲染架构，包括设计态渲染和运行态渲染的同构设计。

## 1. 架构概览

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                            渲染架构                                          │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                        Schema (project.json)                         │   │
│  └──────────────────────────────────┬──────────────────────────────────┘   │
│                                     │                                       │
│                    ┌────────────────┴────────────────┐                     │
│                    ▼                                 ▼                      │
│  ┌─────────────────────────────┐  ┌─────────────────────────────┐         │
│  │      设计态渲染器            │  │      运行态渲染器            │         │
│  │   (DesignRenderer)          │  │   (RuntimeRenderer)          │         │
│  │                             │  │                              │         │
│  │  ┌───────────────────────┐  │  │  ┌────────────────────────┐ │         │
│  │  │   RuntimeRenderer     │  │  │  │   Canvas Layer         │ │         │
│  │  │   (复用运行时渲染)     │  │  │  │   (Konva.js)           │ │         │
│  │  └───────────────────────┘  │  │  └────────────────────────┘ │         │
│  │  ┌───────────────────────┐  │  │  ┌────────────────────────┐ │         │
│  │  │   DesignOverlay       │  │  │  │   DOM Layer            │ │         │
│  │  │   (设计态叠加层)       │  │  │  │   (Vue Components)     │ │         │
│  │  └───────────────────────┘  │  │  └────────────────────────┘ │         │
│  └─────────────────────────────┘  └─────────────────────────────┘         │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

## 2. 分层渲染

### 2.1 层级结构

```
┌─────────────────────────────────────────────────────────────────┐
│  Overlay Layer (z-index: 100)  - 设计态专用                     │
│  • 选择框和控制点（缩放、旋转）                                  │
│  • 对齐辅助线和插入线                                           │
│  • 拖拽预览和框选矩形                                           │
│  • 悬停高亮                                                     │
├─────────────────────────────────────────────────────────────────┤
│  DOM Layer (z-index: 10)  - 交互组件                            │
│  • Vue 组件（Button, Input, Table, Chart...）                   │
│  • CSS 布局（Flexbox, Grid）                                    │
│  • 完整事件支持                                                 │
├─────────────────────────────────────────────────────────────────┤
│  Canvas Layer (z-index: 1)  - 图形绘制                          │
│  • 基础图形（Line, Rect, Circle, Polygon）                      │
│  • 管道（Pipe，带流动动画）                                     │
│  • 工业符号（Symbol）                                           │
│  • 文字标注（Text）                                             │
└─────────────────────────────────────────────────────────────────┘
```

### 2.2 层级 z-index 分配

| 层级           | z-index | 内容                           |
| -------------- | ------- | ------------------------------ |
| Canvas Layer   | 1       | Konva Stage                    |
| DOM Layer      | 10      | Vue 组件容器                   |
| Overlay Layer  | 100     | 选择框、辅助线                 |
| Dialog Layer   | 200     | 弹窗、抽屉                     |
| Tooltip Layer  | 300     | 工具提示、下拉菜单             |

## 3. RuntimeRenderer（运行态渲染器）

### 3.1 核心职责

- 递归渲染 Schema 中的节点树
- 解析数据绑定，订阅数据源
- 处理条件渲染、循环渲染
- 渲染 Canvas 图形和 DOM 组件

### 3.2 组件结构

```typescript
// RuntimeRenderer.vue
<template>
  <div class="runtime-renderer" :style="containerStyle">
    <!-- Canvas 层 -->
    <CanvasRenderer
      :graphics="currentPage.graphicsIds"
      :graphicsById="graphicsById"
      :symbolsById="symbolsById"
      :bindings="resolvedBindings"
    />
    
    <!-- DOM 层 -->
    <div class="dom-layer">
      <NodeRenderer
        v-for="nodeId in currentPage.childrenIds"
        :key="nodeId"
        :nodeId="nodeId"
        :nodesById="nodesById"
      />
    </div>
  </div>
</template>
```

### 3.3 NodeRenderer（节点渲染器）

```typescript
// NodeRenderer.vue
<template>
  <!-- 条件渲染 -->
  <template v-if="shouldRender">
    <!-- 循环渲染 -->
    <template v-if="node.loop">
      <component
        v-for="(item, index) in loopData"
        :key="getLoopKey(item, index)"
        :is="getComponent(node.type)"
        v-bind="resolvedProps"
        :style="resolvedStyle"
        @[event]="handleEvent"
      >
        <!-- 递归渲染子节点 -->
        <NodeRenderer
          v-for="childId in node.children"
          :key="childId"
          :nodeId="childId"
          :loopContext="{ item, index }"
        />
      </component>
    </template>
    
    <!-- 普通渲染 -->
    <template v-else>
      <component
        :is="getComponent(node.type)"
        v-bind="resolvedProps"
        :style="resolvedStyle"
        v-on="eventHandlers"
      >
        <NodeRenderer
          v-for="childId in node.children"
          :key="childId"
          :nodeId="childId"
        />
      </component>
    </template>
  </template>
</template>

<script setup>
const { node, loopContext } = defineProps(['nodeId', 'loopContext']);

// 解析绑定
const resolvedProps = computed(() => {
  return resolveBindings(node.props, node.bindings, {
    $item: loopContext?.item,
    $index: loopContext?.index,
  });
});

// 条件渲染
const shouldRender = computed(() => {
  if (!node.conditions?.visible) return true;
  return evaluateCondition(node.conditions.visible);
});

// 权限检查
const hasPermission = computed(() => {
  return checkPermission(node.permissions?.visible);
});
</script>
```

### 3.4 CanvasRenderer（Canvas 渲染器）

```typescript
// CanvasRenderer.vue
<template>
  <div ref="containerRef" class="canvas-layer"></div>
</template>

<script setup>
import Konva from 'konva';

const containerRef = ref(null);
const stage = ref(null);
const layer = ref(null);

onMounted(() => {
  // 初始化 Konva Stage
  stage.value = new Konva.Stage({
    container: containerRef.value,
    width: props.width,
    height: props.height,
  });
  
  layer.value = new Konva.Layer();
  stage.value.add(layer.value);
  
  // 渲染图形
  renderGraphics();
});

function renderGraphics() {
  for (const graphicId of props.graphics) {
    const graphic = props.graphicsById[graphicId];
    const shape = createKonvaShape(graphic);
    layer.value.add(shape);
    
    // 设置绑定
    setupBindings(shape, graphic.bindings);
  }
  layer.value.draw();
}

function createKonvaShape(graphic) {
  switch (graphic.type) {
    case 'Canvas.Line':
      return new Konva.Line({
        points: graphic.props.points.flat(),
        stroke: graphic.props.stroke,
        strokeWidth: graphic.props.strokeWidth,
      });
    case 'Canvas.Rect':
      return new Konva.Rect({
        x: graphic.props.x,
        y: graphic.props.y,
        width: graphic.props.width,
        height: graphic.props.height,
        fill: graphic.props.fill,
        stroke: graphic.props.stroke,
      });
    case 'Canvas.Pipe':
      return createPipe(graphic);
    // ...
  }
}
</script>
```

## 4. DesignRenderer（设计态渲染器）

### 4.1 核心职责

- 复用 RuntimeRenderer 渲染实际内容
- 叠加设计态交互层（DesignOverlay）
- 拦截事件，阻止组件响应
- 支持拖拽、选择、缩放等设计操作

### 4.2 组件结构

```typescript
// DesignRenderer.vue
<template>
  <div class="design-renderer" @click.capture="handleClick">
    <!-- 运行态渲染（只展示，不响应事件） -->
    <RuntimeRenderer
      :schema="schema"
      :mode="'design'"
      class="runtime-content"
    />
    
    <!-- 设计态叠加层 -->
    <DesignOverlay
      :selectedElements="selectedElements"
      :hoveredElement="hoveredElement"
      :dropTarget="dropTarget"
      :alignmentGuides="alignmentGuides"
      @resize="handleResize"
      @rotate="handleRotate"
    />
  </div>
</template>

<script setup>
// 拦截点击事件
function handleClick(e) {
  e.stopPropagation();
  e.preventDefault();
  
  // 识别点击的元素
  const element = findElementAtPoint(e.clientX, e.clientY);
  if (element) {
    selectionModel.select(element);
  }
}
</script>

<style scoped>
.runtime-content {
  pointer-events: none; /* 阻止运行态组件响应事件 */
}

.runtime-content :deep(*) {
  pointer-events: auto; /* 恢复，由外层捕获 */
  cursor: default;
}
</style>
```

### 4.3 DesignOverlay（设计态叠加层）

```typescript
// DesignOverlay.vue
<template>
  <div class="design-overlay">
    <!-- 选择框 -->
    <SelectionBox
      v-for="element in selectedElements"
      :key="element.id"
      :element="element"
      :showHandles="selectedElements.length === 1"
      @resize="emit('resize', $event)"
      @rotate="emit('rotate', $event)"
    />
    
    <!-- 悬停高亮 -->
    <HoverHighlight
      v-if="hoveredElement && !isSelected(hoveredElement)"
      :element="hoveredElement"
    />
    
    <!-- 拖拽目标高亮 -->
    <DropTargetHighlight
      v-if="dropTarget"
      :target="dropTarget"
    />
    
    <!-- 对齐辅助线 -->
    <AlignmentGuides :guides="alignmentGuides" />
    
    <!-- 框选矩形 -->
    <MarqueeSelection
      v-if="isMarqueeSelecting"
      :start="marqueeStart"
      :end="marqueeEnd"
    />
    
    <!-- 空容器占位符 -->
    <EmptyPlaceholder
      v-for="container in emptyContainers"
      :key="container.id"
      :container="container"
    />
  </div>
</template>
```

### 4.4 SelectionBox（选择框）

```typescript
// SelectionBox.vue
<template>
  <div
    class="selection-box"
    :class="{ locked: element.locked }"
    :style="boxStyle"
  >
    <!-- 边框 -->
    <div class="border"></div>
    
    <!-- 缩放控制点（8个） -->
    <template v-if="showHandles && !element.locked">
      <div
        v-for="handle in resizeHandles"
        :key="handle.position"
        class="resize-handle"
        :class="handle.position"
        @mousedown="startResize(handle)"
      />
    </template>
    
    <!-- 旋转控制点 -->
    <div
      v-if="showHandles && !element.locked"
      class="rotate-handle"
      @mousedown="startRotate"
    />
  </div>
</template>

<style scoped>
.selection-box {
  position: absolute;
  pointer-events: none;
}

.border {
  position: absolute;
  inset: 0;
  border: 2px solid var(--color-primary);
}

.selection-box.locked .border {
  border-style: dashed;
  border-color: var(--color-warning);
}

.resize-handle {
  position: absolute;
  width: 8px;
  height: 8px;
  background: white;
  border: 1px solid var(--color-primary);
  pointer-events: auto;
  cursor: nwse-resize;
}

.resize-handle.nw { top: -4px; left: -4px; cursor: nwse-resize; }
.resize-handle.ne { top: -4px; right: -4px; cursor: nesw-resize; }
.resize-handle.sw { bottom: -4px; left: -4px; cursor: nesw-resize; }
.resize-handle.se { bottom: -4px; right: -4px; cursor: nwse-resize; }
.resize-handle.n { top: -4px; left: 50%; cursor: ns-resize; }
.resize-handle.s { bottom: -4px; left: 50%; cursor: ns-resize; }
.resize-handle.w { left: -4px; top: 50%; cursor: ew-resize; }
.resize-handle.e { right: -4px; top: 50%; cursor: ew-resize; }

.rotate-handle {
  position: absolute;
  top: -24px;
  left: 50%;
  transform: translateX(-50%);
  width: 12px;
  height: 12px;
  border-radius: 50%;
  background: white;
  border: 1px solid var(--color-primary);
  pointer-events: auto;
  cursor: grab;
}
</style>
```

## 5. 数据绑定解析

### 5.1 BindingResolver

```typescript
class BindingResolver {
  constructor(
    private dataService: DataService,
    private varsStore: VarsStore,
    private expressionEngine: ExpressionEngine
  ) {}

  /**
   * 解析绑定，返回实际值
   */
  resolve(
    props: Record<string, any>,
    bindings: Record<string, Binding>,
    context: EvalContext
  ): Record<string, any> {
    const resolved = { ...props };

    for (const [key, binding] of Object.entries(bindings)) {
      resolved[key] = this.resolveBinding(binding, context);
    }

    return resolved;
  }

  private resolveBinding(binding: Binding, context: EvalContext): any {
    switch (binding.kind) {
      case 'datapoint':
        return this.resolveDatapoint(binding, context);
      case 'var':
        return this.resolveVar(binding, context);
      case 'expr':
        return this.resolveExpression(binding, context);
      default:
        return binding.designMock ?? null;
    }
  }

  private resolveDatapoint(binding: DatapointBinding, context: EvalContext): any {
    const value = this.dataService.getValue(binding.path);
    
    if (value === undefined || value === null) {
      return binding.fallback ?? binding.designMock;
    }

    // 应用转换
    if (binding.transform) {
      return this.applyTransform(value, binding.transform);
    }

    return value;
  }

  private resolveVar(binding: VarBinding, context: EvalContext): any {
    return this.varsStore.get(binding.scope, binding.name, context.pageId);
  }

  private resolveExpression(binding: ExprBinding, context: EvalContext): any {
    return this.expressionEngine.evaluate(binding.expr, {
      ...context,
      $vars: this.varsStore.getContext(context.pageId).$vars,
      $global: this.varsStore.getContext(context.pageId).$global,
      $dp: (path: string) => this.dataService.getValue(path),
    });
  }
}
```

### 5.2 响应式更新

```typescript
// 使用 Vue 的响应式系统
function useResolvedBindings(node: ComponentNode) {
  const bindingResolver = inject('bindingResolver');
  const context = inject('evalContext');

  // 创建响应式绑定值
  const resolved = reactive<Record<string, any>>({});

  // 监听数据变化
  watchEffect(() => {
    const newResolved = bindingResolver.resolve(
      node.props,
      node.bindings,
      context.value
    );
    
    Object.assign(resolved, newResolved);
  });

  return resolved;
}
```

## 6. 性能优化

### 6.1 虚拟化渲染

对于大量图形元素，使用视口裁剪：

```typescript
function renderVisibleGraphics(viewport: Rect) {
  const visibleGraphics = graphics.filter(g => 
    intersects(getBounds(g), viewport)
  );
  
  // 只渲染可见图形
  visibleGraphics.forEach(g => renderGraphic(g));
}
```

### 6.2 批量更新

```typescript
// 合并多个绑定更新
const batchUpdate = useDebounceFn((updates: BindingUpdate[]) => {
  layer.value.batchDraw(() => {
    updates.forEach(({ shapeId, prop, value }) => {
      const shape = shapeMap.get(shapeId);
      shape?.setAttr(prop, value);
    });
  });
}, 16); // 60fps
```

### 6.3 离屏渲染

```typescript
// 复杂符号使用离屏 Canvas 缓存
function cacheSymbol(symbol: SymbolDef): HTMLCanvasElement {
  const offscreen = document.createElement('canvas');
  const ctx = offscreen.getContext('2d');
  
  // 渲染符号到离屏 Canvas
  renderSymbolToContext(symbol, ctx);
  
  return offscreen;
}
```

## 7. 模式切换

### 7.1 渲染模式

| 模式    | RuntimeRenderer | DesignOverlay | 事件响应 | 数据来源      |
| ------- | --------------- | ------------- | -------- | ------------- |
| design  | ✅              | ✅            | 设计操作 | designMock    |
| preview | ✅              | ❌            | 正常响应 | DataCenter API |
| runtime | ✅              | ❌            | 正常响应 | ConnectionProfile |

### 7.2 模式切换代码

```typescript
// App.vue
<template>
  <DesignRenderer v-if="mode === 'design'" :schema="schema" />
  <RuntimeRenderer v-else :schema="schema" :mode="mode" />
</template>

<script setup>
const mode = ref<'design' | 'preview' | 'runtime'>('design');

// 预览按钮
function enterPreview() {
  mode.value = 'preview';
}

// 返回设计
function exitPreview() {
  mode.value = 'design';
}
</script>
```

---

**相关文档**：

- [设计态交互](./design-interaction.md)
- [运行时引擎](./runtime-engine.md)
- [Schema 设计](./schema-design.md)
- [数据绑定 v2](./data-binding-v2.md)

