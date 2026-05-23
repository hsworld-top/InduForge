<!--
  CanvasContainer - 画布容器
  提供标尺、缩放、平移、画布尺寸、DesignCanvas 挂载
-->
<script setup lang="ts">
import type {
  CanvasInsertLineBox,
  CanvasInsertLineStyle,
  CanvasLayoutInsertTarget,
  CanvasRowInsertTarget,
  DesignCanvasPageConfig,
} from './canvas-internal.types'
import type { ComponentNode } from '@/editor-core/document/types'
import { storeToRefs } from 'pinia'
import {
  computed,
  nextTick,
  onBeforeUnmount,
  onMounted,
  provide,
  ref,
  toRef,
  toRefs,
  watch,
} from 'vue'
import {
  canAcceptChildByDescriptor,
  getDescriptor,
  isContainerType,
} from '@/editor-core/descriptors/registry'
import { componentRegistry } from '@/editor-core/registry/component-registry'
import { useEditorStore } from '@/stores/editor-store'
import {
  CANVAS_COL_INSERT_EDGE_THRESHOLD,
  CANVAS_DEFAULT_PAGE_MARGIN_X,
  CANVAS_DEFAULT_PAGE_MARGIN_Y,
  CANVAS_ROW_INSERT_EDGE_THRESHOLD,
  CANVAS_RULER_MAJOR_STEP,
  CANVAS_RULER_MAX,
  CANVAS_RULER_MINOR_STEP,
  CANVAS_RULER_SIZE,
} from './canvas-container-constants'
import CanvasInsertLineOverlay from './CanvasInsertLineOverlay.vue'
import CanvasRulerLayer from './CanvasRulerLayer.vue'
import { useCanvasRulerPointer } from './composables/use-canvas-ruler-pointer'
import {
  type CanvasViewportZoomAnchor,
  useCanvasViewportPlacement,
} from './composables/use-canvas-viewport-placement'
import { useCanvasZoomWheel } from './composables/use-canvas-zoom-wheel'
import { endDrag, useDragState } from './composables/use-drag-state'
import DesignCanvas from './DesignCanvas.vue'
import { canvasSnapEnabledKey, canvasZoomKey } from './injection-keys'
import PageStyleInjector from './PageStyleInjector'
import { buildDesignerPageDomId } from './style-config-css'
import {
  CANVAS_OUTSIDE_MARQUEE_START_EVENT,
  type OutsideMarqueeStartDetail,
} from './interaction/marquee-interaction'

type CanvasContainerHost = HTMLElement & {
  __rulerObserver?: ResizeObserver | null
}

const NODE_POINTER_DRAG_FREEZE_START_EVENT = 'designer:node-pointer-drag-freeze-start'
const NODE_POINTER_DRAG_FREEZE_END_EVENT = 'designer:node-pointer-drag-freeze-end'

interface DropTargetResolution {
  nodeId: string
  element: HTMLElement | null
}

interface CanvasContentBounds {
  minX: number
  minY: number
  maxX: number
  maxY: number
}

const props = defineProps({
  width: {
    type: Number,
    default: 1920,
  },
  height: {
    type: Number,
    default: 1080,
  },
  zoom: {
    type: Number,
    default: 1,
  },
  showRuler: {
    type: Boolean,
    default: true,
  },
  showGrid: {
    type: Boolean,
    default: true,
  },
  enableSnap: {
    type: Boolean,
    default: true,
  },
  viewResetToken: {
    type: Number,
    default: 0,
  },
})

const emit = defineEmits(['zoomChange'])

const { width, height, zoom } = toRefs(props)

const containerRef = ref<HTMLElement | null>(null)
const wrapperRef = ref<HTMLElement | null>(null)
const canvasRef = ref<HTMLElement | null>(null)
const zoomAnchor = ref<CanvasViewportZoomAnchor | null>(null)
const editorStore = useEditorStore()
const { doc, selection, pages, currentPageId, currentPage, docVersion, projectRuntimeTheme } =
  storeToRefs(editorStore)
const dragState = useDragState()
const minorStep = CANVAS_RULER_MINOR_STEP
const majorStep = CANVAS_RULER_MAJOR_STEP
const rulerMax = CANVAS_RULER_MAX
const rulerSize = CANVAS_RULER_SIZE
const defaultPageMarginX = CANVAS_DEFAULT_PAGE_MARGIN_X
const defaultPageMarginY = CANVAS_DEFAULT_PAGE_MARGIN_Y
const containerSize = ref({ width: 0, height: 0 })
const translateX = ref(0)
const translateY = ref(0)
const rulerInset = computed(() => (props.showRuler ? rulerSize : 0))
const {
  pointerX,
  pointerY,
  handleRulerMouseMove,
  handleRulerMouseLeave,
  handleNodeTransform,
  handleNodeTransformEnd,
} = useCanvasRulerPointer({
  showRuler: toRef(props, 'showRuler'),
  containerRef,
  zoom,
  translateX,
  translateY,
  rulerInset,
})
const rowInsertEdgeThreshold = CANVAS_ROW_INSERT_EDGE_THRESHOLD
const colInsertEdgeThreshold = CANVAS_COL_INSERT_EDGE_THRESHOLD
const showInsertLine = ref(false)
const insertLineStyle = ref<CanvasInsertLineStyle | null>(null)
const insertLineBox = ref<CanvasInsertLineBox | null>(null)
const rowInsertSnapshot = ref<CanvasRowInsertTarget | null>(null)
const layoutInsertSnapshot = ref<CanvasLayoutInsertTarget | null>(null)

function captureWheelZoomAnchor(event: WheelEvent): void {
  const wrapper = wrapperRef.value
  const canvas = canvasRef.value
  if (!wrapper || !canvas || !zoom.value) {
    zoomAnchor.value = null
    return
  }
  const wrapperRect = wrapper.getBoundingClientRect()
  const canvasRect = canvas.getBoundingClientRect()
  zoomAnchor.value = {
    viewportX: event.clientX - wrapperRect.left,
    viewportY: event.clientY - wrapperRect.top,
    canvasX: (event.clientX - canvasRect.left) / zoom.value,
    canvasY: (event.clientY - canvasRect.top) / zoom.value,
  }
}

const { handleZoomWheel } = useCanvasZoomWheel({
  zoom,
  onZoomChange: (next, event) => {
    captureWheelZoomAnchor(event)
    emit('zoomChange', next)
  },
})

/**
 * 拖拽结束时清理插入线
 */
watch(
  () => dragState.dragType,
  (value) => {
    if (!value) {
      showInsertLine.value = false
      insertLineStyle.value = null
      insertLineBox.value = null
      rowInsertSnapshot.value = null
      layoutInsertSnapshot.value = null
    }
  },
)

function handleGlobalDragOver(event: DragEvent) {
  if (!dragState.dragType) return
  if (!containerRef.value) return
  const t = event.target
  if (!(t instanceof Node) || !containerRef.value.contains(t)) return
  event.preventDefault()
}

function handleGlobalDrop(event: DragEvent) {
  if (!dragState.dragType) return
  if (!canvasRef.value || !containerRef.value) return
  const t = event.target
  if (!(t instanceof Node) || !containerRef.value.contains(t)) return
  event.preventDefault()

  const componentType = dragState.dragType
  handleDropWithType(event, componentType)
}
function handleGlobalMouseUp(event: MouseEvent) {
  if (!dragState.dragType) return
  if (!containerRef.value) {
    endDrag()
    showInsertLine.value = false
    insertLineStyle.value = null
    insertLineBox.value = null
    return
  }

  const t = event.target
  if (!(t instanceof Node) || !containerRef.value.contains(t)) {
    endDrag()
    showInsertLine.value = false
    insertLineStyle.value = null
    insertLineBox.value = null
    return
  }

  const componentType = dragState.dragType
  handleDropWithType(event, componentType)
}

/**
 * 处理工作台灰区 pointerdown，桥接到画布内部框选。
 * @param {PointerEvent} event - 指针事件
 */
function handleWrapperPointerDownCapture(event: PointerEvent): void {
  if (event.pointerType === 'mouse' && event.button !== 0) return
  if (!(event.target instanceof Node)) return
  if (canvasRef.value?.contains(event.target)) return
  const detail: OutsideMarqueeStartDetail = {
    clientX: event.clientX,
    clientY: event.clientY,
    modifiers: {
      ctrl: Boolean(event.ctrlKey),
      meta: Boolean(event.metaKey),
      shift: Boolean(event.shiftKey),
    },
  }
  window.dispatchEvent(new CustomEvent(CANVAS_OUTSIDE_MARQUEE_START_EVENT, { detail }))
}

// 向子组件提供当前缩放比例，用于拖拽落点换算（InjectionKey 便于 TS/Volar 推断）
provide(canvasZoomKey, zoom)
// 向节点拖拽逻辑提供吸附开关，保持顶部工具栏与画布行为一致。
provide(canvasSnapEnabledKey, toRef(props, 'enableSnap'))

const rootNodeId = computed(() => currentPage.value?.rootNodeId || '')
const currentPageSnapshot = computed(() => {
  const page = pages.value.find((item) => item.id === currentPageId.value)
  return page || currentPage.value || null
})
const pageStyleConfig = computed(() => String(currentPageSnapshot.value?.config?.styleConfig || ''))
const pageDomId = computed(() =>
  currentPageId.value ? buildDesignerPageDomId(currentPageId.value) : undefined,
)
const showWorkbenchGrid = computed(() => props.showGrid)
const pointerXOnRuler = computed(() => Math.max(0, pointerX.value - rulerInset.value))
const pointerYOnRuler = computed(() => Math.max(0, pointerY.value - rulerInset.value))

function parseFiniteNumber(value: unknown): number | null {
  if (typeof value === 'number' && Number.isFinite(value)) return value
  if (typeof value !== 'string') return null
  const normalized = value.trim().replace(/px$/i, '')
  if (!normalized) return null
  const parsed = Number(normalized)
  return Number.isFinite(parsed) ? parsed : null
}

function resolveAbsoluteBounds(node: ComponentNode): CanvasContentBounds | null {
  const abs = node.absolutePos || node.layoutItem?.free?.abs || null
  const isAbsolute = node.positioning === 'absolute' || node.layoutItem?.free?.mode === 'abs'
  if (!isAbsolute && !abs) return null
  const style = node.style || {}
  const x = parseFiniteNumber(abs?.x) ?? parseFiniteNumber(style.left) ?? 0
  const y = parseFiniteNumber(abs?.y) ?? parseFiniteNumber(style.top) ?? 0
  const w = Math.max(1, parseFiniteNumber(abs?.w) ?? parseFiniteNumber(style.width) ?? 120)
  const h = Math.max(1, parseFiniteNumber(abs?.h) ?? parseFiniteNumber(style.height) ?? 40)
  return {
    minX: x,
    minY: y,
    maxX: x + w,
    maxY: y + h,
  }
}

const canvasContentBounds = computed<CanvasContentBounds>(() => {
  void docVersion.value
  const base: CanvasContentBounds = {
    minX: 0,
    minY: 0,
    maxX: width.value,
    maxY: height.value,
  }
  const model = doc.value
  const rootNode = rootNodeId.value ? model?.getNode(rootNodeId.value) : null
  if (!model || !rootNode) return base
  for (const childId of rootNode.children || []) {
    const child = model.getNode(childId)
    if (!child) continue
    const bounds = resolveAbsoluteBounds(child)
    if (!bounds) continue
    base.minX = Math.min(base.minX, bounds.minX)
    base.minY = Math.min(base.minY, bounds.minY)
    base.maxX = Math.max(base.maxX, bounds.maxX)
    base.maxY = Math.max(base.maxY, bounds.maxY)
  }
  return base
})

const canvasOverflowOrigin = ref({ left: 0, top: 0 })

watch(
  () => [canvasContentBounds.value.minX, canvasContentBounds.value.minY] as const,
  ([minX, minY]) => {
    const requiredLeft = Math.max(0, -minX)
    const requiredTop = Math.max(0, -minY)
    const nextLeft = Math.max(canvasOverflowOrigin.value.left, requiredLeft)
    const nextTop = Math.max(canvasOverflowOrigin.value.top, requiredTop)
    if (
      nextLeft === canvasOverflowOrigin.value.left &&
      nextTop === canvasOverflowOrigin.value.top
    ) {
      return
    }
    // 编辑态滚动世界只自动扩张，不随拖拽自动收缩，避免远距离拖动时画布原点回弹。
    canvasOverflowOrigin.value = { left: nextLeft, top: nextTop }
  },
  { immediate: true },
)

const canvasOverflowOffset = computed(() => ({
  left: Math.ceil(canvasOverflowOrigin.value.left * zoom.value),
  top: Math.ceil(canvasOverflowOrigin.value.top * zoom.value),
}))

const frozenCanvasOverflowOffset = ref<{ left: number; top: number } | null>(null)
const activeCanvasOverflowOffset = computed(
  () => frozenCanvasOverflowOffset.value || canvasOverflowOffset.value,
)

function freezeCanvasOverflowOffset(): void {
  if (frozenCanvasOverflowOffset.value) return
  frozenCanvasOverflowOffset.value = { ...canvasOverflowOffset.value }
}

function releaseCanvasOverflowOffset(): void {
  frozenCanvasOverflowOffset.value = null
}

watch(
  () => dragState.dragType,
  (dragType) => {
    if (dragType) {
      freezeCanvasOverflowOffset()
      return
    }
    if (!dragType && frozenCanvasOverflowOffset.value) {
      releaseCanvasOverflowOffset()
    }
  },
  { flush: 'sync' },
)

watch(
  () => [activeCanvasOverflowOffset.value.left, activeCanvasOverflowOffset.value.top] as const,
  ([nextLeft, nextTop], [prevLeft, prevTop]) => {
    const deltaLeft = nextLeft - prevLeft
    const deltaTop = nextTop - prevTop
    if (deltaLeft <= 0 && deltaTop <= 0) return
    // 只在滚动世界向左/上扩张时补偿滚动；收缩不自动补偿，避免拖回页面时按钮被拉回顶部。
    void nextTick(() => {
      const wrapper = wrapperRef.value
      if (!wrapper) return
      if (deltaLeft > 0) wrapper.scrollLeft += deltaLeft
      if (deltaTop > 0) wrapper.scrollTop += deltaTop
    })
  },
)

/**
 * 插入节点（拖入场景）：禁止自动选中新建节点
 * @param {string} type - 组件类型
 * @param {string} parentId - 父节点 ID
 * @param {number | undefined} index - 插入索引
 * @param {{ dropPosition?: { x: number; y: number } }} [options] - 插入参数
 * @returns {import('@/editor-core').ComponentNode | null}
 */
function insertNodeWithoutSelection(
  type: string,
  parentId: string,
  index: number | undefined,
  options: { dropPosition?: { x: number; y: number } } = {},
): ComponentNode | null {
  return editorStore.insertNode(type, parentId, index, {
    ...options,
    autoSelectInserted: false,
  })
}

useCanvasViewportPlacement({
  containerSize,
  rulerInset,
  width,
  height,
  zoom,
  translateX,
  translateY,
  canvasOverflowOffset: activeCanvasOverflowOffset,
  wrapperRef,
  zoomAnchor,
  defaultPageMarginX,
  defaultPageMarginY,
  rootNodeId,
  viewResetToken: toRef(props, 'viewResetToken'),
})

const workbenchStyle = computed((): Record<string, string> => {
  const alpha = props.showRuler ? 0.04 : 0.03
  const style: Record<string, string> = {
    backgroundColor: 'var(--designer-group-surface)',
    backgroundImage: 'none',
  }
  if (!showWorkbenchGrid.value) return style
  style.backgroundImage = `linear-gradient(rgba(100,116,139,${alpha}) 1px, transparent 1px), linear-gradient(90deg, rgba(100,116,139,${alpha}) 1px, transparent 1px)`
  style.backgroundSize = '24px 24px'
  style.backgroundPosition = '0 0'
  return style
})

const canvasStyle = computed((): Record<string, string> => {
  void docVersion.value
  const config = (currentPageSnapshot.value?.config || {}) as DesignCanvasPageConfig
  const background = config.background || null
  const showGrid = props.showGrid
  const style: Record<string, string> = {
    width: `${width.value}px`,
    height: `${height.value}px`,
    transform: `translate(${translateX.value + rulerInset.value + activeCanvasOverflowOffset.value.left}px, ${
      translateY.value + rulerInset.value + activeCanvasOverflowOffset.value.top
    }px) scale(${zoom.value})`,
    backgroundColor: 'var(--designer-shell-surface)',
    border: '1px solid rgba(148, 163, 184, 0.45)',
    boxShadow: '0 0 0 1px rgba(255,255,255,0.85) inset, 0 10px 26px rgba(15, 23, 42, 0.08)',
  }

  if (background?.kind === 'color') {
    style.backgroundColor = background.value || '#ffffff'
  } else if (background?.kind === 'image') {
    style.backgroundImage = `url(${background.value || ''})`
    style.backgroundSize = 'cover'
    style.backgroundRepeat = 'no-repeat'
    style.backgroundPosition = 'center'
  } else if (background?.kind === 'gradient') {
    style.backgroundImage = background.value || ''
    style.backgroundSize = 'cover'
    style.backgroundRepeat = 'no-repeat'
    style.backgroundPosition = 'center'
  }

  if (showGrid) {
    const minorStepSize = 12
    const majorStepSize = 48
    const gridLayer = `
      linear-gradient(rgba(71, 85, 105, 0.12) 1px, transparent 1px),
      linear-gradient(90deg, rgba(71, 85, 105, 0.12) 1px, transparent 1px),
      linear-gradient(rgba(71, 85, 105, 0.2) 1px, transparent 1px),
      linear-gradient(90deg, rgba(71, 85, 105, 0.2) 1px, transparent 1px)
    `
    if (style.backgroundImage) {
      style.backgroundImage = `${gridLayer}, ${style.backgroundImage}`
      style.backgroundSize = `${minorStepSize}px ${minorStepSize}px, ${minorStepSize}px ${minorStepSize}px, ${majorStepSize}px ${majorStepSize}px, ${majorStepSize}px ${majorStepSize}px, ${style.backgroundSize || 'cover'}`
      style.backgroundRepeat = `repeat, repeat, repeat, repeat, ${style.backgroundRepeat || 'no-repeat'}`
      style.backgroundPosition = `0 0, 0 0, 0 0, 0 0, ${style.backgroundPosition || 'center'}`
    } else {
      style.backgroundImage = gridLayer
      style.backgroundSize = `${minorStepSize}px ${minorStepSize}px, ${minorStepSize}px ${minorStepSize}px, ${majorStepSize}px ${majorStepSize}px, ${majorStepSize}px ${majorStepSize}px`
    }
  }

  return style
})

/**
 * 计算滚动内容尺寸，确保缩放后能触发滚动条
 */
const scrollContentStyle = computed(() => {
  const bounds = canvasContentBounds.value
  const scaledWidth = Math.max(width.value, bounds.maxX) * zoom.value
  const scaledHeight = Math.max(height.value, bounds.maxY) * zoom.value
  const viewportWidth = Math.max(0, (containerSize.value.width || 0) - rulerInset.value)
  const viewportHeight = Math.max(0, (containerSize.value.height || 0) - rulerInset.value)
  // 右/下编辑扩展区保持“可编辑但不过度”，避免滚动后空白区域喧宾夺主
  const workspaceExtraRight = Math.max(24, Math.min(68, Math.round(viewportWidth * 0.09)))
  const workspaceExtraBottom = Math.max(28, Math.min(76, Math.round(viewportHeight * 0.1)))
  const coverageX = scaledWidth / Math.max(1, viewportWidth)
  const coverageY = scaledHeight / Math.max(1, viewportHeight)
  const pageStartX = rulerInset.value + translateX.value + activeCanvasOverflowOffset.value.left
  const pageStartY = rulerInset.value + translateY.value + activeCanvasOverflowOffset.value.top
  const offsetX = Math.max(rulerInset.value, pageStartX)
  const offsetY = Math.max(rulerInset.value, pageStartY)
  const baseWidth = Math.ceil(scaledWidth + offsetX)
  const baseHeight = Math.ceil(scaledHeight + offsetY)
  const minWidth = containerSize.value.width || 0
  const minHeight = containerSize.value.height || 0
  // 当页面已经完整落在当前视口内时，不再追加右/下扩展区，避免出现“适配后右侧灰条”
  const effectiveExtraRight =
    baseWidth <= minWidth
      ? 0
      : coverageX >= 1.6
        ? 0
        : coverageX >= 1.2
          ? Math.round(workspaceExtraRight * 0.25)
          : workspaceExtraRight
  const effectiveExtraBottom =
    baseHeight <= minHeight
      ? 0
      : coverageY >= 1.6
        ? 0
        : coverageY >= 1.2
          ? Math.round(workspaceExtraBottom * 0.28)
          : workspaceExtraBottom
  // 当存在页面左/上方的负坐标节点时，需要保留足够的滚动范围回到页面本体位置。
  const originScrollReserveWidth = minWidth + activeCanvasOverflowOffset.value.left
  const originScrollReserveHeight = minHeight + activeCanvasOverflowOffset.value.top
  return {
    width: `${Math.max(minWidth, baseWidth + effectiveExtraRight, originScrollReserveWidth)}px`,
    height: `${Math.max(minHeight, baseHeight + effectiveExtraBottom, originScrollReserveHeight)}px`,
  }
})

const rulerXStyle = computed(() => {
  const minor = minorStep * zoom.value
  const major = majorStep * zoom.value
  return {
    '--ruler-size': `${rulerInset.value}px`,
    '--ruler-minor': `${minor}px`,
    '--ruler-major': `${major}px`,
    '--ruler-offset': `${translateX.value}px`,
  }
})

const rulerYStyle = computed(() => {
  const minor = minorStep * zoom.value
  const major = majorStep * zoom.value
  return {
    '--ruler-size': `${rulerInset.value}px`,
    '--ruler-minor': `${minor}px`,
    '--ruler-major': `${major}px`,
    '--ruler-offset': `${translateY.value}px`,
  }
})

const rulerMarksX = computed(() => {
  const marks = []
  const max = rulerMax
  for (let value = 0; value <= max; value += majorStep) {
    const pos = value * zoom.value + translateX.value + rulerInset.value
    if (pos < -majorStep || pos > containerSize.value.width) continue
    marks.push(value)
  }
  return marks
})

const rulerMarksY = computed(() => {
  const marks = []
  const max = rulerMax
  for (let value = 0; value <= max; value += majorStep) {
    const pos = value * zoom.value + translateY.value + rulerInset.value
    if (pos < -majorStep || pos > containerSize.value.height) continue
    marks.push(value)
  }
  return marks
})

/**
 * 处理拖拽经过
 * @param {DragEvent} event - 拖拽事件
 */
function handleDragOver(event: DragEvent) {
  event.preventDefault()
  if (event.dataTransfer) {
    event.dataTransfer.dropEffect = 'copy'
  }
  const payload =
    event.dataTransfer?.getData('application/x-designer-component') ||
    event.dataTransfer?.getData('text/plain')
  const fallbackType = dragState.dragType || ''
  let componentType = ''
  if (payload) {
    try {
      const parsed = JSON.parse(payload)
      componentType = parsed.type || ''
    } catch {
      componentType = payload
    }
  }
  componentType = componentType || fallbackType
  if (!componentType) {
    showInsertLine.value = false
    insertLineStyle.value = null
    insertLineBox.value = null
    rowInsertSnapshot.value = null
    layoutInsertSnapshot.value = null
    return
  }

  const rowInsertTarget = componentType !== 'ElCol' ? resolveRowInsertTarget(event) : null
  if (rowInsertTarget?.insertLine && rowInsertTarget?.lineBox) {
    showInsertLine.value = true
    insertLineStyle.value = rowInsertTarget.insertLine
    insertLineBox.value = rowInsertTarget.lineBox
    rowInsertSnapshot.value = rowInsertTarget
    layoutInsertSnapshot.value = null
    return
  }

  const layoutInsertTarget =
    componentType !== 'ElLayoutRow' ? resolveLayoutInsertTarget(event) : null
  if (
    layoutInsertTarget &&
    'insertLine' in layoutInsertTarget &&
    layoutInsertTarget.insertLine &&
    'lineBox' in layoutInsertTarget &&
    layoutInsertTarget.lineBox
  ) {
    const withLine = layoutInsertTarget as Extract<
      CanvasLayoutInsertTarget,
      { lineBox: CanvasInsertLineBox; insertLine: CanvasInsertLineStyle }
    >
    showInsertLine.value = true
    insertLineStyle.value = withLine.insertLine
    insertLineBox.value = withLine.lineBox
    layoutInsertSnapshot.value = layoutInsertTarget
    rowInsertSnapshot.value = null
    return
  }

  showInsertLine.value = false
  insertLineStyle.value = null
  insertLineBox.value = null
  rowInsertSnapshot.value = null
  layoutInsertSnapshot.value = null
}

/**
 * 在 ElLayout 内插入组件
 * @param {import('@/editor-core').ComponentNode} layoutNode - 布局节点
 * @param {string} componentType - 组件类型
 */
function insertIntoElLayout(layoutNode: ComponentNode, componentType: string) {
  if (!layoutNode) return
  const rowIds = (layoutNode.children || []).filter((childId) => {
    const childNode = doc.value?.getNode?.(childId)
    return childNode?.type === 'ElLayoutRow'
  })
  let rowId = rowIds[0]
  if (!rowId) {
    const rowNode = insertNodeWithoutSelection('ElLayoutRow', layoutNode.id, 0)
    if (!rowNode) return
    const latestLayout = doc.value?.getNode?.(layoutNode.id)
    const rowCount = (latestLayout?.children || []).filter((childId) => {
      const childNode = doc.value?.getNode?.(childId)
      return childNode?.type === 'ElLayoutRow'
    }).length
    editorStore.updateNode(layoutNode.id, {
      props: {
        ...(latestLayout?.props || layoutNode.props || {}),
        rows: Math.max(1, rowCount),
      },
    })
    editorStore.updateNode(rowNode.id, {
      props: { ...(rowNode.props || {}), columns: 1 },
    })
    rowId = rowNode.id
  }
  const rowNode = doc.value?.getNode?.(rowId)
  if (!rowNode) return
  insertIntoElLayoutRow(rowNode, componentType)
}

/**
 * 在 ElLayoutRow 内插入组件
 * @param {import('@/editor-core').ComponentNode} rowNode - 行节点
 * @param {string} componentType - 组件类型
 */
function insertIntoElLayoutRow(rowNode: ComponentNode, componentType: string) {
  if (!rowNode) return
  const colIds = (rowNode.children || []).filter((childId) => {
    const childNode = doc.value?.getNode?.(childId)
    return childNode?.type === 'ElCol'
  })
  let colId = colIds[0]
  if (!colId) {
    const colNode = insertNodeWithoutSelection('ElCol', rowNode.id, 0)
    colId = colNode?.id || ''
    if (colId) {
      const latestRow = doc.value?.getNode?.(rowNode.id)
      const colCount = (latestRow?.children || []).filter((childId) => {
        const childNode = doc.value?.getNode?.(childId)
        return childNode?.type === 'ElCol'
      }).length
      editorStore.updateNode(rowNode.id, {
        props: {
          ...(latestRow?.props || rowNode.props || {}),
          columns: Math.max(1, colCount),
        },
      })
    }
  }
  if (!colId) return
  insertNodeWithoutSelection(componentType, colId, undefined)
}

/**
 * 解析 ElLayout 行插入目标（靠近上下边缘）
 * @param {DragEvent} event - 拖拽事件
 * @returns {{ layoutNode: import('@/editor-core').ComponentNode, index: number } | null}
 */
function resolveLayoutInsertTarget(event: DragEvent | MouseEvent): CanvasLayoutInsertTarget | null {
  if (!doc.value) return null
  const hitList = document.elementsFromPoint(event.clientX, event.clientY)
  for (const hit of hitList) {
    if (!(hit instanceof Element)) continue
    const layoutElement = hit.closest?.('[data-node-type="ElLayout"][data-node-id]')
    if (!layoutElement) continue
    const layoutId = layoutElement.getAttribute('data-node-id')
    const layoutNode = layoutId ? doc.value.getNode?.(layoutId) : null
    if (!layoutNode || layoutNode.type !== 'ElLayout') continue
    const rowIds = (layoutNode.children || []).filter((childId) => {
      const childNode = doc.value?.getNode?.(childId)
      return childNode?.type === 'ElLayoutRow'
    })
    if (rowIds.length === 0) {
      return { layoutNode, index: 0 }
    }
    const pointY = event.clientY
    for (let i = 0; i < rowIds.length; i += 1) {
      const rowId = rowIds[i]
      const rowElement = layoutElement.querySelector(`[data-node-id="${rowId}"]`)
      if (!rowElement) continue
      const rect = rowElement.getBoundingClientRect?.()
      if (!rect) continue
      if (Math.abs(pointY - rect.top) <= rowInsertEdgeThreshold) {
        const layoutRect = layoutElement.getBoundingClientRect?.()
        if (!layoutRect) return { layoutNode, index: i }
        return {
          layoutNode,
          index: i,
          lineBox: {
            left: layoutRect.left,
            top: layoutRect.top,
            width: layoutRect.width,
            height: layoutRect.height,
          },
          insertLine: {
            orientation: 'horizontal',
            offset: Math.max(0, rect.top - layoutRect.top),
          },
        }
      }
      if (Math.abs(pointY - rect.bottom) <= rowInsertEdgeThreshold) {
        const layoutRect = layoutElement.getBoundingClientRect?.()
        if (!layoutRect) return { layoutNode, index: i + 1 }
        return {
          layoutNode,
          index: i + 1,
          lineBox: {
            left: layoutRect.left,
            top: layoutRect.top,
            width: layoutRect.width,
            height: layoutRect.height,
          },
          insertLine: {
            orientation: 'horizontal',
            offset: Math.max(0, rect.bottom - layoutRect.top),
          },
        }
      }
    }
  }
  return null
}

/**
 * 解析 ElLayoutRow 列插入目标（靠近左右边缘）
 * @param {DragEvent} event - 拖拽事件
 * @returns {{ rowNode: import('@/editor-core').ComponentNode, index: number } | null}
 */
function resolveRowInsertTarget(event: DragEvent | MouseEvent): CanvasRowInsertTarget | null {
  if (!doc.value) return null
  const primaryHit = document.elementFromPoint(event.clientX, event.clientY)
  if (primaryHit instanceof Element) {
    const rowElement = primaryHit.closest?.('[data-node-type="ElLayoutRow"][data-node-id]')
    if (rowElement) {
      const rowId = rowElement.getAttribute('data-node-id')
      const rowNode = rowId ? doc.value?.getNode?.(rowId) : null
      const rowRect = rowElement.getBoundingClientRect?.()
      if (rowNode?.type === 'ElLayoutRow' && rowRect) {
        const nearLeft = event.clientX - rowRect.left <= colInsertEdgeThreshold
        const nearRight = rowRect.right - event.clientX <= colInsertEdgeThreshold
        if (nearLeft || nearRight) {
          const colIds = (rowNode.children || []).filter((childId) => {
            const childNode = doc.value?.getNode?.(childId)
            return childNode?.type === 'ElCol'
          })
          const index = nearLeft ? 0 : colIds.length
          return {
            rowNode,
            index,
            lineBox: {
              left: rowRect.left,
              top: rowRect.top,
              width: rowRect.width,
              height: rowRect.height,
            },
            insertLine: {
              orientation: 'vertical',
              offset: Math.max(0, (nearLeft ? rowRect.left : rowRect.right) - rowRect.left),
            },
          }
        }
      }
    }
  }
  const hitList = document.elementsFromPoint(event.clientX, event.clientY)
  for (const hit of hitList) {
    if (!(hit instanceof Element)) continue
    const colElement = hit.closest?.('[data-node-type="ElCol"][data-node-id]')
    if (!colElement) continue
    const colId = colElement.getAttribute('data-node-id')
    const colNode = colId ? doc.value.getNode?.(colId) : null
    if (!colNode) continue
    const rowNode = doc.value.getParent?.(colNode.id)
    if (!rowNode || rowNode.type !== 'ElLayoutRow') continue
    const colRect = colElement.getBoundingClientRect?.()
    if (!colRect) continue
    const rowElement =
      colElement.closest?.(`[data-node-id="${rowNode.id}"]`) ||
      document.querySelector(`[data-node-id="${rowNode.id}"]`)
    const rowRect = rowElement?.getBoundingClientRect?.()
    if (rowRect) {
      const nearRowLeft = event.clientX - rowRect.left <= colInsertEdgeThreshold
      const nearRowRight = rowRect.right - event.clientX <= colInsertEdgeThreshold
      if (nearRowLeft || nearRowRight) {
        const colIds = (rowNode.children || []).filter((childId) => {
          const childNode = doc.value?.getNode?.(childId)
          return childNode?.type === 'ElCol'
        })
        const index = nearRowLeft ? 0 : colIds.length
        return {
          rowNode,
          index,
          lineBox: {
            left: rowRect.left,
            top: rowRect.top,
            width: rowRect.width,
            height: rowRect.height,
          },
          insertLine: {
            orientation: 'vertical',
            offset: Math.max(0, (nearRowLeft ? rowRect.left : rowRect.right) - rowRect.left),
          },
        }
      }
    }
    const nearLeft = event.clientX - colRect.left <= colInsertEdgeThreshold
    const nearRight = colRect.right - event.clientX <= colInsertEdgeThreshold
    if (!nearLeft && !nearRight) continue
    const colIds = (rowNode.children || []).filter((childId) => {
      const childNode = doc.value?.getNode?.(childId)
      return childNode?.type === 'ElCol'
    })
    const currentIndex = colIds.indexOf(colNode.id)
    if (currentIndex === -1) continue
    const index = nearLeft ? currentIndex : currentIndex + 1
    return {
      rowNode,
      index,
      lineBox: rowRect
        ? {
            left: rowRect.left,
            top: rowRect.top,
            width: rowRect.width,
            height: rowRect.height,
          }
        : null,
      insertLine: rowRect
        ? {
            orientation: 'vertical',
            offset: Math.max(0, (nearLeft ? colRect.left : colRect.right) - rowRect.left),
          }
        : null,
    }
  }
  for (const hit of hitList) {
    if (!(hit instanceof Element)) continue
    const rowElement = hit.closest?.('[data-node-type="ElLayoutRow"][data-node-id]')
    if (!rowElement) continue
    const rowId = rowElement.getAttribute('data-node-id')
    const rowNode = rowId ? doc.value?.getNode?.(rowId) : null
    if (!rowNode || rowNode.type !== 'ElLayoutRow') continue
    const rowRect = rowElement.getBoundingClientRect?.()
    if (!rowRect) continue
    const nearLeft = event.clientX - rowRect.left <= colInsertEdgeThreshold
    const nearRight = rowRect.right - event.clientX <= colInsertEdgeThreshold
    if (!nearLeft && !nearRight) continue
    const colIds = (rowNode.children || []).filter((childId) => {
      const childNode = doc.value?.getNode?.(childId)
      return childNode?.type === 'ElCol'
    })
    const index = nearLeft ? 0 : colIds.length
    return {
      rowNode,
      index,
      lineBox: {
        left: rowRect.left,
        top: rowRect.top,
        width: rowRect.width,
        height: rowRect.height,
      },
      insertLine: {
        orientation: 'vertical',
        offset: Math.max(0, (nearLeft ? rowRect.left : rowRect.right) - rowRect.left),
      },
    }
  }
  return null
}

/**
 * 处理拖拽放置
 * @param {DragEvent} event - 拖拽事件
 */
function handleDrop(event: DragEvent) {
  event.preventDefault()
  if (!canvasRef.value) return

  const payload =
    event.dataTransfer?.getData('application/x-designer-component') ||
    event.dataTransfer?.getData('text/plain')
  const fallbackType = dragState.dragType || ''

  let componentType = ''
  if (payload) {
    try {
      const parsed = JSON.parse(payload)
      componentType = parsed.type || ''
    } catch {
      componentType = payload
    }
  }

  if (!componentType) {
    componentType = fallbackType
  }
  if (!componentType) return

  handleDropWithType(event, componentType)
}

/**
 * 统一处理拖拽放置逻辑
 * @param {DragEvent} event - 拖拽事件
 * @param {string} componentType - 组件类型
 */
function handleDropWithType(event: DragEvent | MouseEvent, componentType: string) {
  showInsertLine.value = false
  insertLineStyle.value = null
  insertLineBox.value = null
  const cachedRowInsert = rowInsertSnapshot.value
  const cachedLayoutInsert = layoutInsertSnapshot.value
  rowInsertSnapshot.value = null
  layoutInsertSnapshot.value = null
  const layoutInsertTarget =
    componentType !== 'ElLayoutRow' ? cachedLayoutInsert || resolveLayoutInsertTarget(event) : null
  if (layoutInsertTarget) {
    const rowNode = insertNodeWithoutSelection(
      'ElLayoutRow',
      layoutInsertTarget.layoutNode.id,
      layoutInsertTarget.index,
    )
    if (rowNode) {
      const latestLayout = doc.value?.getNode?.(layoutInsertTarget.layoutNode.id)
      const rowCount = (latestLayout?.children || []).filter((childId) => {
        const childNode = doc.value?.getNode?.(childId)
        return childNode?.type === 'ElLayoutRow'
      }).length
      editorStore.updateNode(layoutInsertTarget.layoutNode.id, {
        props: {
          ...(latestLayout?.props || layoutInsertTarget.layoutNode.props || {}),
          rows: Math.max(1, rowCount),
        },
      })
      editorStore.updateNode(rowNode.id, {
        props: { ...(rowNode.props || {}), columns: 1 },
      })
      const latestRow = doc.value?.getNode?.(rowNode.id)
      const colIds = (latestRow?.children || []).filter((childId) => {
        const childNode = doc.value?.getNode?.(childId)
        return childNode?.type === 'ElCol'
      })
      let colId = colIds[0]
      if (!colId) {
        const colNode = insertNodeWithoutSelection('ElCol', rowNode.id, 0)
        colId = colNode?.id || ''
      }
      if (colId) {
        insertNodeWithoutSelection(componentType, colId, undefined)
      }
    }
    endDrag()
    return
  }

  const rowInsertTarget =
    componentType !== 'ElCol' ? cachedRowInsert || resolveRowInsertTarget(event) : null
  if (rowInsertTarget) {
    const colNode = insertNodeWithoutSelection(
      'ElCol',
      rowInsertTarget.rowNode.id,
      rowInsertTarget.index,
    )
    if (colNode) {
      const latestRow = doc.value?.getNode?.(rowInsertTarget.rowNode.id)
      const colCount = (latestRow?.children || []).filter((childId) => {
        const childNode = doc.value?.getNode?.(childId)
        return childNode?.type === 'ElCol'
      }).length
      editorStore.updateNode(rowInsertTarget.rowNode.id, {
        props: {
          ...(latestRow?.props || rowInsertTarget.rowNode.props || {}),
          columns: Math.max(1, colCount),
        },
      })
      insertNodeWithoutSelection(componentType, colNode.id, undefined)
    }
    endDrag()
    return
  }

  const target = resolveDropTarget(event, componentType)
  if (!target.nodeId || !target.element) {
    endDrag()
    return
  }
  const targetNode = doc.value?.getNode?.(target.nodeId)
  if (targetNode?.type === 'ElCol' && componentType !== 'ElCol') {
    if ((targetNode.children || []).length > 0) {
      const rowNode = doc.value?.getParent?.(targetNode.id)
      if (rowNode?.type === 'ElLayoutRow') {
        const rowInsertTarget =
          componentType !== 'ElCol' ? cachedRowInsert || resolveRowInsertTarget(event) : null
        const colIds = (rowNode.children || []).filter((childId) => {
          const childNode = doc.value?.getNode?.(childId)
          return childNode?.type === 'ElCol'
        })
        const currentIndex = Math.max(0, colIds.indexOf(targetNode.id))
        let insertIndex = currentIndex + 1
        if (rowInsertTarget?.rowNode?.id === rowNode.id) {
          insertIndex = rowInsertTarget.index
        }
        const colNode = insertNodeWithoutSelection('ElCol', rowNode.id, insertIndex)
        if (colNode) {
          const latestRow = doc.value?.getNode?.(rowNode.id)
          const colCount = (latestRow?.children || []).filter((childId) => {
            const childNode = doc.value?.getNode?.(childId)
            return childNode?.type === 'ElCol'
          }).length
          editorStore.updateNode(rowNode.id, {
            props: {
              ...(latestRow?.props || rowNode.props || {}),
              columns: Math.max(1, colCount),
            },
          })
          insertNodeWithoutSelection(componentType, colNode.id, undefined)
        }
        endDrag()
        return
      }
    }
    const rowNode = doc.value?.getParent?.(targetNode.id)
    if (rowNode?.type === 'ElLayoutRow') {
      const rowElement = document.querySelector(`[data-node-id="${rowNode.id}"]`)
      const rowRect = rowElement?.getBoundingClientRect?.()
      if (rowRect) {
        const nearLeft = event.clientX - rowRect.left <= colInsertEdgeThreshold
        const nearRight = rowRect.right - event.clientX <= colInsertEdgeThreshold
        if (nearLeft || nearRight) {
          const colIds = (rowNode.children || []).filter((childId) => {
            const childNode = doc.value?.getNode?.(childId)
            return childNode?.type === 'ElCol'
          })
          const insertIndex = nearLeft ? 0 : colIds.length
          const colNode = insertNodeWithoutSelection('ElCol', rowNode.id, insertIndex)
          if (colNode) {
            const latestRow = doc.value?.getNode?.(rowNode.id)
            const colCount = (latestRow?.children || []).filter((childId) => {
              const childNode = doc.value?.getNode?.(childId)
              return childNode?.type === 'ElCol'
            }).length
            editorStore.updateNode(rowNode.id, {
              props: {
                ...(latestRow?.props || rowNode.props || {}),
                columns: Math.max(1, colCount),
              },
            })
            insertNodeWithoutSelection(componentType, colNode.id, undefined)
          }
          endDrag()
          return
        }
      }
    }
  }
  if (targetNode?.type === 'ElLayout') {
    const nearestCol = resolveNearestElCol(event)
    if (nearestCol) {
      if ((nearestCol.children || []).length > 0) {
        const rowNode = doc.value?.getParent?.(nearestCol.id)
        if (rowNode?.type === 'ElLayoutRow') {
          const rowInsertTarget =
            componentType !== 'ElCol' ? cachedRowInsert || resolveRowInsertTarget(event) : null
          let insertIndex = (rowNode.children || []).length
          if (rowInsertTarget?.rowNode?.id === rowNode.id) {
            insertIndex = rowInsertTarget.index
          } else {
            const colIds = (rowNode.children || []).filter((childId) => {
              const childNode = doc.value?.getNode?.(childId)
              return childNode?.type === 'ElCol'
            })
            const currentIndex = Math.max(0, colIds.indexOf(nearestCol.id))
            insertIndex = currentIndex + 1
          }
          const colNode = insertNodeWithoutSelection('ElCol', rowNode.id, insertIndex)
          if (colNode) {
            const latestRow = doc.value?.getNode?.(rowNode.id)
            const colCount = (latestRow?.children || []).filter((childId) => {
              const childNode = doc.value?.getNode?.(childId)
              return childNode?.type === 'ElCol'
            }).length
            editorStore.updateNode(rowNode.id, {
              props: {
                ...(latestRow?.props || rowNode.props || {}),
                columns: Math.max(1, colCount),
              },
            })
            insertNodeWithoutSelection(componentType, colNode.id, undefined)
          }
          endDrag()
          return
        }
      } else {
        insertNodeWithoutSelection(componentType, nearestCol.id, undefined)
        endDrag()
        return
      }
    }
    const rowTarget = resolveLayoutRowByPoint(targetNode, target.element, event)
    if (rowTarget) {
      insertIntoElLayoutRow(rowTarget, componentType)
      endDrag()
      return
    }
    insertIntoElLayout(targetNode, componentType)
    endDrag()
    return
  }
  if (targetNode?.type === 'ElLayoutRow' && componentType !== 'ElCol') {
    const nearestCol = resolveNearestElCol(event)
    if (nearestCol) {
      if ((nearestCol.children || []).length > 0) {
        const rowInsertTarget =
          componentType !== 'ElCol' ? cachedRowInsert || resolveRowInsertTarget(event) : null
        let insertIndex = (targetNode.children || []).length
        if (rowInsertTarget?.rowNode?.id === targetNode.id) {
          insertIndex = rowInsertTarget.index
        } else {
          const colIds = (targetNode.children || []).filter((childId) => {
            const childNode = doc.value?.getNode?.(childId)
            return childNode?.type === 'ElCol'
          })
          const currentIndex = Math.max(0, colIds.indexOf(nearestCol.id))
          insertIndex = currentIndex + 1
        }
        const colNode = insertNodeWithoutSelection('ElCol', targetNode.id, insertIndex)
        if (colNode) {
          const latestRow = doc.value?.getNode?.(targetNode.id)
          const colCount = (latestRow?.children || []).filter((childId) => {
            const childNode = doc.value?.getNode?.(childId)
            return childNode?.type === 'ElCol'
          }).length
          editorStore.updateNode(targetNode.id, {
            props: {
              ...(latestRow?.props || targetNode.props || {}),
              columns: Math.max(1, colCount),
            },
          })
          insertNodeWithoutSelection(componentType, colNode.id, undefined)
        }
        endDrag()
        return
      }
      insertNodeWithoutSelection(componentType, nearestCol.id, undefined)
      endDrag()
      return
    }
    insertIntoElLayoutRow(targetNode, componentType)
    endDrag()
    return
  }

  const { x, y } = calcDropOffset(event, target.element)
  insertNode(componentType, target.nodeId, x, y)
  endDrag()
}

/**
 * 插入组件节点
 * @param {string} type - 组件类型
 * @param {string} parentId - 父节点ID
 * @param {number} x - X 坐标
 * @param {number} y - Y 坐标
 */
function insertNode(type: string, parentId: string, x: number, y: number) {
  if (!parentId) return
  const parentNode = doc.value?.getNode(parentId)
  const insertIndex = parentNode?.children?.length ?? 0
  insertNodeWithoutSelection(type, parentId, insertIndex, {
    dropPosition: { x, y },
  })
}

/**
 * \u70b9\u51fb\u753b\u5e03\u5916\u90e8\u7a7a\u767d\u533a\u57df\u65f6\u663e\u793a\u9875\u9762\u4fe1\u606f
 * @param {MouseEvent} event - \u9f20\u6807\u4e8b\u4ef6
 */
function handleContainerClick(event: MouseEvent) {
  const target = event.target
  if (canvasRef.value && target instanceof Node && canvasRef.value.contains(target)) {
    return
  }
  selection.value?.clearSelection()
}

/**
 * 解析拖拽落点目标容器
 * @param {DragEvent} event - 拖拽事件
 * @param {string} componentType - 组件类型
 * @returns {{ nodeId: string, element: HTMLElement } | { nodeId: string, element: HTMLElement | null }}
 */
function resolveDropTarget(
  event: DragEvent | MouseEvent,
  componentType: string,
): DropTargetResolution {
  if (!doc.value) {
    return { nodeId: rootNodeId.value, element: canvasRef.value }
  }

  const hitList = document.elementsFromPoint(event.clientX, event.clientY)
  for (const hit of hitList) {
    if (!(hit instanceof Element)) continue
    const nodeElement = hit.closest?.('[data-node-id][data-node-type]')
    if (!(nodeElement instanceof HTMLElement)) continue
    const nodeId = nodeElement.getAttribute('data-node-id')
    if (!nodeId) continue
    const node = doc.value?.getNode?.(nodeId)
    if (!node) continue
    if (node.type === 'ElCol') {
      return { nodeId, element: nodeElement }
    }
    if (node.type === 'ElLayoutRow' || node.type === 'ElLayout') {
      return { nodeId, element: nodeElement }
    }
  }

  const hit = document.elementFromPoint(event.clientX, event.clientY)
  let current = hit

  while (current && current !== canvasRef.value) {
    const htmlCurrent = current instanceof HTMLElement ? current : null
    const nodeId = htmlCurrent?.dataset?.nodeId
    if (nodeId && isContainerNode(nodeId) && canAcceptChild(nodeId, componentType)) {
      return { nodeId, element: htmlCurrent }
    }
    current = current.parentElement
  }

  return { nodeId: rootNodeId.value, element: canvasRef.value }
}

/**
 * 解析鼠标下最近的 ElCol
 * @param {DragEvent} event - 拖拽事件
 * @returns {import('@/editor-core').ComponentNode | null}
 */
function resolveNearestElCol(event: DragEvent | MouseEvent): ComponentNode | null {
  if (!doc.value) return null
  const hitList = document.elementsFromPoint(event.clientX, event.clientY)
  for (const hit of hitList) {
    if (!(hit instanceof Element)) continue
    const colElement = hit.closest?.('[data-node-type="ElCol"][data-node-id]')
    if (!colElement) continue
    const colId = colElement.getAttribute('data-node-id')
    if (!colId) continue
    const colNode = doc.value?.getNode?.(colId)
    if (colNode?.type === 'ElCol') return colNode
  }
  return null
}

/**
 * 根据鼠标位置解析 ElLayout 内最接近的行
 * @param {import('@/editor-core').ComponentNode} layoutNode - 布局节点
 * @param {HTMLElement | null} layoutElement - 布局元素
 * @param {DragEvent} event - 拖拽事件
 * @returns {import('@/editor-core').ComponentNode | null}
 */
function resolveLayoutRowByPoint(
  layoutNode: ComponentNode,
  layoutElement: HTMLElement | null,

  event: DragEvent | MouseEvent,
): ComponentNode | null {
  if (!doc.value || !layoutNode || layoutNode.type !== 'ElLayout') return null
  if (!layoutElement) return null
  const rowIds = (layoutNode.children || []).filter((childId: string) => {
    const childNode = doc.value?.getNode?.(childId)
    return childNode?.type === 'ElLayoutRow'
  })
  let bestRow = null
  let bestDistance = Number.POSITIVE_INFINITY
  for (const rowId of rowIds) {
    const rowElement = layoutElement.querySelector(`[data-node-id="${rowId}"]`)
    if (!rowElement) continue
    const rect = rowElement.getBoundingClientRect?.()
    if (!rect) continue
    if (event.clientY >= rect.top && event.clientY <= rect.bottom) {
      return doc.value?.getNode?.(rowId) || null
    }
    const distance = Math.min(
      Math.abs(event.clientY - rect.top),
      Math.abs(event.clientY - rect.bottom),
    )
    if (distance < bestDistance) {
      bestDistance = distance
      bestRow = doc.value?.getNode?.(rowId) || null
    }
  }
  return bestRow
}

/**
 * 判断节点是否为容器 * @param {string} nodeId - 节点 ID
 * @returns {boolean}
 */
function isContainerNode(nodeId: string): boolean {
  const node = doc.value?.getNode(nodeId)
  if (!node) return false
  return isContainerType(node.type)
}

/**
 * 判断容器是否允许子组件 * @param {string} parentId - 父节点ID
 * @param {string} childType - 子组件类型 * @returns {boolean}
 */
function canAcceptChild(parentId: string, childType: string): boolean {
  const node = doc.value?.getNode(parentId)
  if (!node) return false
  // 优先从 descriptor 读取（新架构组件）
  const currentChildCount = (node.children || []).length
  const descriptor = getDescriptor(node.type)
  if (descriptor) {
    // 如果已注册 descriptor，使用 descriptor 的判断结果
    return canAcceptChildByDescriptor(node.type, childType, currentChildCount)
  }
  const manifest = componentRegistry.get(node.type)
  const allowed = manifest?.allowedChildren
  if (!Array.isArray(allowed) || allowed.length === 0) return false
  return allowed.includes(childType)
}

/**
 * 计算落点相对坐标
 * @param {DragEvent} event - 拖拽事件
 * @param {HTMLElement} element - 目标元素
 * @returns {{ x: number, y: number }}
 */
function calcDropOffset(
  event: DragEvent | MouseEvent,
  element: HTMLElement,
): { x: number; y: number } {
  const rect = element.getBoundingClientRect()
  const offsetX = (event.clientX - rect.left) / zoom.value
  const offsetY = (event.clientY - rect.top) / zoom.value
  return {
    x: Math.max(0, Math.round(offsetX)),
    y: Math.max(0, Math.round(offsetY)),
  }
}

/**
 * 获取默认尺寸
 * @param {string} type - 组件类型
 * @param {Object | undefined} manifest - 组件清单
 * @returns {{width: number, height: number}}
 */
onMounted(() => {
  window.addEventListener('dragover', handleGlobalDragOver)
  window.addEventListener('drop', handleGlobalDrop)
  window.addEventListener('mouseup', handleGlobalMouseUp)
  window.addEventListener('designer:node-transform', handleNodeTransform)
  window.addEventListener('designer:node-transform-end', handleNodeTransformEnd)
  window.addEventListener(NODE_POINTER_DRAG_FREEZE_START_EVENT, freezeCanvasOverflowOffset)
  window.addEventListener(NODE_POINTER_DRAG_FREEZE_END_EVENT, releaseCanvasOverflowOffset)
  if (containerRef.value && typeof ResizeObserver !== 'undefined') {
    const host = containerRef.value as CanvasContainerHost
    const observer = new ResizeObserver((entries) => {
      const entry = entries[0]
      if (!entry) return
      const { width: w, height: h } = entry.contentRect
      containerSize.value = { width: w, height: h }
    })
    observer.observe(host)
    host.__rulerObserver = observer
  }
})

onBeforeUnmount(() => {
  window.removeEventListener('dragover', handleGlobalDragOver)
  window.removeEventListener('drop', handleGlobalDrop)
  window.removeEventListener('mouseup', handleGlobalMouseUp)
  window.removeEventListener('designer:node-transform', handleNodeTransform)
  window.removeEventListener('designer:node-transform-end', handleNodeTransformEnd)
  window.removeEventListener(NODE_POINTER_DRAG_FREEZE_START_EVENT, freezeCanvasOverflowOffset)
  window.removeEventListener(NODE_POINTER_DRAG_FREEZE_END_EVENT, releaseCanvasOverflowOffset)
  const host = containerRef.value as CanvasContainerHost | null
  if (host?.__rulerObserver) {
    host.__rulerObserver.disconnect()
    host.__rulerObserver = null
  }
})
</script>

<template>
  <main
    ref="containerRef"
    class="canvas-container"
    @wheel="handleZoomWheel"
    @mousemove="handleRulerMouseMove"
    @mouseleave="handleRulerMouseLeave"
    @click="handleContainerClick"
  >
    <CanvasRulerLayer
      v-if="showRuler"
      :ruler-inset="rulerInset"
      :ruler-x-style="rulerXStyle"
      :ruler-y-style="rulerYStyle"
      :ruler-marks-x="rulerMarksX"
      :ruler-marks-y="rulerMarksY"
      :pointer-x-on-ruler="pointerXOnRuler"
      :pointer-y-on-ruler="pointerYOnRuler"
      :zoom="zoom"
      :translate-x="translateX"
      :translate-y="translateY"
    />
    <div
      ref="wrapperRef"
      class="canvas-wrapper"
      @pointerdown.capture="handleWrapperPointerDownCapture"
    >
      <div class="canvas-scroll-content" :style="[scrollContentStyle, workbenchStyle]">
        <div
          ref="canvasRef"
          class="canvas"
          :id="pageDomId"
          :style="canvasStyle"
          :data-page-style-root="currentPageId || undefined"
          :data-page-dom-id="pageDomId"
          :data-runtime-theme="projectRuntimeTheme"
          @dragover="handleDragOver"
          @drop="handleDrop"
        >
          <PageStyleInjector :css="pageStyleConfig" :page-id="currentPageId || ''" />
          <CanvasInsertLineOverlay
            :show="showInsertLine"
            :line-style="insertLineStyle"
            :line-box="insertLineBox"
          />
          <div class="absolute inset-0 pointer-events-none">
            <slot name="canvas-layer" />
          </div>
          <div class="absolute inset-0">
            <DesignCanvas />
          </div>
        </div>
      </div>
    </div>
  </main>
</template>

<style scoped>
.canvas-container {
  position: relative;
  height: 100%;
  min-height: 0;
}

.canvas-wrapper {
  position: relative;
  display: block !important;
  width: 100%;
  height: 100%;
  min-height: 0;
  overflow: auto;
  padding: 0 !important;
  align-items: flex-start !important;
  justify-content: flex-start !important;
}

.canvas-scroll-content {
  position: relative;
  min-width: 100%;
  min-height: 100%;
}

.canvas {
  position: relative;
  transform-origin: 0 0;
}

.canvas[data-runtime-theme='light'] {
  --runtime-bg-color: #ffffff;
  --runtime-surface-color: #ffffff;
  --runtime-text-color: #1f2937;
  --runtime-text-muted-color: #667085;
  --runtime-border-color: #d0d5dd;
  --runtime-primary-color: #1677ff;
}

.canvas[data-runtime-theme='dark'] {
  --runtime-bg-color: #111827;
  --runtime-surface-color: #1f2937;
  --runtime-text-color: #f9fafb;
  --runtime-text-muted-color: #98a2b3;
  --runtime-border-color: #344054;
  --runtime-primary-color: #60a5fa;
}
</style>
