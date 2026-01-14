<template>
  <main
    class="canvas-container"
    ref="containerRef"
    @wheel="handleZoomWheel"
    @click="handleContainerClick"
  >
    <div class="canvas-wrapper" ref="wrapperRef">
      <div
        class="canvas"
        ref="canvasRef"
        :style="canvasStyle"
        @dragover="handleDragOver"
        @drop="handleDrop"
      >
        <div class="absolute inset-0 pointer-events-none">
          <slot name="canvas-layer" />
        </div>
        <div class="absolute inset-0">
          <DesignCanvas />
        </div>
      </div>
    </div>
    <div class="canvas-statusbar">
      <span>画布: {{ width }} × {{ height }}</span>
      <el-divider direction="vertical" />
      <span>缩放: {{ Math.round(zoom * 100) }}%</span>
      <el-tooltip content="重置缩放">
        <el-button size="small" text @click="handleZoomReset">
          <IconEpRefresh />
        </el-button>
      </el-tooltip>
    </div>
  </main>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, provide, ref, toRefs } from "vue";
import { storeToRefs } from "pinia";
import { useEditorStore } from "@/stores/editor-store";
import IconEpRefresh from "~icons/ep/refresh";
import { componentRegistry } from "@/editor-core";
import { useDragState, endDrag } from "./use-drag-state";
import DesignCanvas from "./DesignCanvas.vue";

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
});

const emit = defineEmits(["zoomChange"]);

const { width, height, zoom } = toRefs(props);

const containerRef = ref(null);
const wrapperRef = ref(null);
const canvasRef = ref(null);
const editorStore = useEditorStore();
const { doc, history, selection, currentPage } = storeToRefs(editorStore);
const dragState = useDragState();

const handleGlobalDragOver = (event) => {
  if (!dragState.dragType) return;
  if (!containerRef.value) return;
  if (!containerRef.value.contains(event.target)) return;
  event.preventDefault();
};

const handleGlobalDrop = (event) => {
  if (!dragState.dragType) return;
  if (!canvasRef.value || !containerRef.value) return;
  if (!containerRef.value.contains(event.target)) return;
  event.preventDefault();

  const componentType = dragState.dragType;
  const target = resolveDropTarget(event, componentType);
  if (!target.nodeId || !target.element) return;

  const { x, y } = calcDropOffset(event, target.element);
  insertNode(componentType, target.nodeId, x, y);
  endDrag();
};
const handleGlobalMouseUp = (event) => {
  if (!dragState.dragType) return;
  if (!containerRef.value) {
    endDrag();
    return;
  }

  if (!containerRef.value.contains(event.target)) {
    endDrag();
    return;
  }

  const componentType = dragState.dragType;
  const target = resolveDropTarget(event, componentType);
  if (!target.nodeId || !target.element) {
    endDrag();
    return;
  }

  const { x, y } = calcDropOffset(event, target.element);
  insertNode(componentType, target.nodeId, x, y);
  endDrag();
};

// 向子组件提供当前缩放比例，用于拖拽落点换�?
provide("canvasZoom", zoom);

const rootNodeId = computed(() => currentPage.value?.rootNodeId || "");
const translateX = ref(0);
const translateY = ref(0);

const canvasStyle = computed(() => {
  const style = {
    width: `${width.value}px`,
    height: `${height.value}px`,
    transform: `translate(${translateX.value}px, ${translateY.value}px) scale(${zoom.value})`,
  };

  // 显示网格
  if (currentPage.value?.config?.showGrid) {
    const gridSize = 10;
    style.backgroundImage = `
      linear-gradient(rgba(0, 0, 0, 0.08) 1px, transparent 1px),
      linear-gradient(90deg, rgba(0, 0, 0, 0.08) 1px, transparent 1px)
    `;
    style.backgroundSize = `${gridSize}px ${gridSize}px`;
  }

  return style;
});

/**
 * 处理拖拽经过
 * @param {DragEvent} event - 拖拽事件
 */
const handleDragOver = (event) => {
  event.preventDefault();
  if (event.dataTransfer) {
    event.dataTransfer.dropEffect = "copy";
  }
};

/**
 * 处理拖拽放置
 * @param {DragEvent} event - 拖拽事件
 */
const handleDrop = (event) => {
  event.preventDefault();
  if (!canvasRef.value) return;

  const payload =
    event.dataTransfer?.getData("application/x-designer-component") ||
    event.dataTransfer?.getData("text/plain");
  const fallbackType = dragState.dragType || "";

  let componentType = "";
  if (payload) {
    try {
      const parsed = JSON.parse(payload);
      componentType = parsed.type || "";
    } catch (error) {
      componentType = payload;
    }
  }

  if (!componentType) {
    componentType = fallbackType;
  }
  if (!componentType) return;

  const target = resolveDropTarget(event, componentType);
  if (!target.nodeId || !target.element) return;

  const { x, y } = calcDropOffset(event, target.element);
  insertNode(componentType, target.nodeId, x, y);
  endDrag();
};

/**
 * 判断组件是否为布局容器类型
 * @param {string} type - 组件类型
 * @returns {boolean}
 */
const isLayoutContainerType = (type) => {
  return [
    "FlexContainer",
    "GridContainer",
    "FreeContainer",
    "ResponsiveLayout",
    "ColumnLayout1",
    "ColumnLayout2",
    "ColumnLayout4",
  ].includes(type);
};

/**
 * 插入组件节点
 * @param {string} type - 组件类型
 * @param {string} parentId - 父节�?ID
 * @param {number} x - X 坐标
 * @param {number} y - Y 坐标
 */
const insertNode = (type, parentId, x, y) => {
  if (!parentId) return;
  const parentNode = doc.value?.getNode(parentId);
  const insertIndex = parentNode?.children?.length ?? 0;
  editorStore.insertNode(type, parentId, insertIndex, {
    dropPosition: { x, y },
  });
};

/**
 * 处理画布缩放
 * @param {WheelEvent} event - 滚轮事件
 */
const handleZoomWheel = (event) => {
  if (!event.ctrlKey) return;
  event.preventDefault();

  const rect =
    wrapperRef.value?.getBoundingClientRect() ||
    containerRef.value?.getBoundingClientRect();
  if (!rect) return;

  const pointerX = Math.min(
    Math.max(0, event.clientX - rect.left),
    rect.width
  );
  const pointerY = Math.min(
    Math.max(0, event.clientY - rect.top),
    rect.height
  );

  const step = 0.1;
  const direction = event.deltaY > 0 ? -1 : 1;
  const nextZoom = Math.min(5, Math.max(0.1, zoom.value + step * direction));
  const currentZoom = zoom.value;
  if (nextZoom === currentZoom) return;

  const worldX = (pointerX - translateX.value) / currentZoom;
  const worldY = (pointerY - translateY.value) / currentZoom;

  translateX.value = pointerX - worldX * nextZoom;
  translateY.value = pointerY - worldY * nextZoom;

  emit("zoomChange", Number(nextZoom.toFixed(2)));
};

/**
 * 重置缩放比例
 */
const handleZoomReset = () => {
  translateX.value = 0;
  translateY.value = 0;
  emit("zoomChange", 1);
};
/**
 * \u70b9\u51fb\u753b\u5e03\u5916\u90e8\u7a7a\u767d\u533a\u57df\u65f6\u663e\u793a\u9875\u9762\u4fe1\u606f
 * @param {MouseEvent} event - \u9f20\u6807\u4e8b\u4ef6
 */
const handleContainerClick = (event) => {
  const target = event.target;
  if (canvasRef.value && canvasRef.value.contains(target)) {
    return;
  }
  selection.value?.clearSelection();
};


/**
 * 构建布局配置
 * @param {import('@/editor-core').ComponentNode | null} parentNode - 父节�? * @param {{x: number, y: number, width: number, height: number}} dropInfo - 放置信息
 * @returns {import('@/editor-core').LayoutItem | null}
 */
const buildLayoutItem = (parentNode, dropInfo) => {
  if (!parentNode) {
    return buildFreeLayoutItem(dropInfo);
  }

  if (
    parentNode.type === "GridContainer" ||
    parentNode.type === "ColumnLayout1" ||
    parentNode.type === "ColumnLayout2" ||
    parentNode.type === "ColumnLayout4"
  ) {
    return buildGridLayoutItem(parentNode);
  }

  if (parentNode.type === "FlexContainer" || parentNode.type === "ResponsiveLayout") {
    return buildFlexLayoutItem();
  }

  if (parentNode.type === "FreeContainer") {
    return buildFreeLayoutItem(dropInfo);
  }

  return buildFreeLayoutItem(dropInfo);
};

/**
 * 构建自由布局配置
 * @param {{x: number, y: number, width: number, height: number}} dropInfo - 放置信息
 * @returns {import('@/editor-core').LayoutItem}
 */
const buildFreeLayoutItem = (dropInfo) => {
  return {
    free: {
      mode: "abs",
      abs: {
        x: Math.max(0, Math.round(dropInfo.x)),
        y: Math.max(0, Math.round(dropInfo.y)),
        w: dropInfo.width,
        h: dropInfo.height,
        z: 1,
      },
    },
  };
};

/**
 * 构建 Flex 布局配置
 * @returns {import('@/editor-core').LayoutItem}
 */
const buildFlexLayoutItem = () => {
  return {
    flex: {
      grow: 0,
      shrink: 0,
      basis: "auto",
    },
  };
};

/**
 * 构建 Grid 布局配置
 * @param {import('@/editor-core').ComponentNode} parentNode - 父节�? * @returns {import('@/editor-core').LayoutItem}
 */
const buildGridLayoutItem = (parentNode) => {
  const columns = resolveGridCount(parentNode.props?.columns);
  const colCount = Math.max(1, columns);
  const index = parentNode.children?.length ?? 0;
  const row = Math.floor(index / colCount) + 1;
  const col = (index % colCount) + 1;

  return {
    grid: {
      row,
      col,
      rowSpan: 1,
      colSpan: 1,
    },
  };
};

/**
 * 解析 Grid 列数
 * @param {string | number | undefined} value - 列配�? * @returns {number}
 */
const resolveGridCount = (value) => {
  if (typeof value === "number" && Number.isFinite(value)) {
    return Math.max(1, Math.floor(value));
  }

  if (typeof value === "string") {
    const repeatMatch = value.match(/repeat\((\d+)/i);
    if (repeatMatch) {
      const count = Number(repeatMatch[1]);
      if (Number.isFinite(count)) return Math.max(1, Math.floor(count));
    }
    const tokens = value.trim().split(/\s+/).filter(Boolean);
    if (tokens.length > 0) return tokens.length;
  }

  return 1;
};

/**
 * 解析拖拽落点目标容器
 * @param {DragEvent} event - 拖拽事件
 * @param {string} componentType - 组件类型
 * @returns {{ nodeId: string, element: HTMLElement } | { nodeId: string, element: HTMLElement | null }}
 */
const resolveDropTarget = (event, componentType) => {
  if (!doc.value) {
    return { nodeId: rootNodeId.value, element: canvasRef.value };
  }

  const hit = document.elementFromPoint(event.clientX, event.clientY);
  let current = hit;

  while (current && current !== canvasRef.value) {
    const nodeId = current.dataset?.nodeId;
    if (nodeId && isContainerNode(nodeId) && canAcceptChild(nodeId, componentType)) {
      return { nodeId, element: current };
    }
    current = current.parentElement;
  }

  return { nodeId: rootNodeId.value, element: canvasRef.value };
};

/**
 * 判断节点是否为容�? * @param {string} nodeId - 节点 ID
 * @returns {boolean}
 */
const isContainerNode = (nodeId) => {
  const node = doc.value?.getNode(nodeId);
  if (!node) return false;
  const manifest = componentRegistry.get(node.type);
  return Boolean(manifest?.isContainer);
};

/**
 * 判断容器是否允许子组�? * @param {string} parentId - 父节�?ID
 * @param {string} childType - 子组件类�? * @returns {boolean}
 */
const canAcceptChild = (parentId, childType) => {
  const node = doc.value?.getNode(parentId);
  if (!node) return false;
  const manifest = componentRegistry.get(node.type);
  const allowed = manifest?.allowedChildren;
  if (!Array.isArray(allowed) || allowed.length === 0) return true;
  return allowed.includes(childType);
};

/**
 * 计算落点相对坐标
 * @param {DragEvent} event - 拖拽事件
 * @param {HTMLElement} element - 目标元素
 * @returns {{ x: number, y: number }}
 */
const calcDropOffset = (event, element) => {
  const rect = element.getBoundingClientRect();
  const offsetX = (event.clientX - rect.left) / zoom.value;
  const offsetY = (event.clientY - rect.top) / zoom.value;
  return {
    x: Math.max(0, Math.round(offsetX)),
    y: Math.max(0, Math.round(offsetY)),
  };
};

/**
 * 获取默认尺寸
 * @param {string} type - 组件类型
 * @param {Object | undefined} manifest - 组件清单
 * @returns {{width: number, height: number}}
 */
const resolveDefaultSize = (type, manifest) => {
  if (manifest?.defaultSize) {
    return {
      width: manifest.defaultSize.width || 120,
      height: manifest.defaultSize.height || 32,
    };
  }

  const sizeMap = {
    FlexContainer: { width: 360, height: 200 },
    FreeContainer: { width: 360, height: 200 },
    GridContainer: { width: 360, height: 200 },
    Text: { width: 120, height: 32 },
    Button: { width: 120, height: 36 },
  };

  return sizeMap[type] || { width: 160, height: 80 };
};
onMounted(() => {
  window.addEventListener("dragover", handleGlobalDragOver);
  window.addEventListener("drop", handleGlobalDrop);
  window.addEventListener("mouseup", handleGlobalMouseUp);
});

onBeforeUnmount(() => {
  window.removeEventListener("dragover", handleGlobalDragOver);
  window.removeEventListener("drop", handleGlobalDrop);
  window.removeEventListener("mouseup", handleGlobalMouseUp);
});
</script>

<style scoped>
.canvas {
  position: relative;
  transform-origin: 0 0;
  background-image: linear-gradient(
      rgba(0, 0, 0, 0.05) 1px,
      transparent 1px
    ),
    linear-gradient(90deg, rgba(0, 0, 0, 0.05) 1px, transparent 1px);
  background-size: 10px 10px;
}

.dark .canvas {
  background-image: linear-gradient(
      rgba(255, 255, 255, 0.05) 1px,
      transparent 1px
    ),
    linear-gradient(90deg, rgba(255, 255, 255, 0.05) 1px, transparent 1px);
}
</style>












