<!--
  CanvasContainer - 画布容器
  提供标尺、缩放、平移、画布尺寸、DesignCanvas 挂载
-->
<template>
  <main
    class="canvas-container"
    ref="containerRef"
    @wheel="handleZoomWheel"
    @mousemove="handleRulerMouseMove"
    @mouseleave="handleRulerMouseLeave"
    @click="handleContainerClick"
  >
    <div
      v-if="showRuler"
      class="ruler-layer"
      :style="{ '--ruler-size': `${rulerInset}px` }"
    >
      <div class="ruler-zero" />
      <div class="ruler ruler-x" :style="rulerXStyle">
        <div
          class="ruler-crosshair-x"
          :style="{ left: `${pointerXOnRuler}px` }"
        />
        <span
          v-for="mark in rulerMarksX"
          :key="`x-${mark}`"
          class="ruler-label"
          :style="{ left: `${mark * zoom + translateX}px` }"
        >
          {{ mark }}
        </span>
      </div>
      <div class="ruler ruler-y" :style="rulerYStyle">
        <div
          class="ruler-crosshair-y"
          :style="{ top: `${pointerYOnRuler}px` }"
        />
        <span
          v-for="mark in rulerMarksY"
          :key="`y-${mark}`"
          class="ruler-label"
          :style="{ top: `${mark * zoom + translateY}px` }"
        >
          {{ mark }}
        </span>
      </div>
    </div>
    <div class="canvas-wrapper" ref="wrapperRef">
      <div
        class="canvas-scroll-content"
        :style="[scrollContentStyle, workbenchStyle]"
      >
        <div
          class="canvas"
          ref="canvasRef"
          :style="canvasStyle"
          @dragover="handleDragOver"
          @drop="handleDrop"
        >
          <teleport
            v-if="showInsertLine && insertLineStyle && insertLineBox"
            to="body"
          >
            <div
              class="canvas-insert-line"
              :class="insertLineStyle.orientation"
              :style="{
                left:
                  insertLineStyle.orientation === 'vertical'
                    ? insertLineBox.left + insertLineStyle.offset + 'px'
                    : insertLineBox.left + 'px',
                top:
                  insertLineStyle.orientation === 'horizontal'
                    ? insertLineBox.top + insertLineStyle.offset + 'px'
                    : insertLineBox.top + 'px',
                width:
                  insertLineStyle.orientation === 'vertical'
                    ? '2px'
                    : insertLineBox.width + 'px',
                height:
                  insertLineStyle.orientation === 'horizontal'
                    ? '2px'
                    : insertLineBox.height + 'px',
              }"
            />
          </teleport>
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

<script setup>
import {
  computed,
  onBeforeUnmount,
  onMounted,
  provide,
  ref,
  toRefs,
  watch,
} from "vue";
import { storeToRefs } from "pinia";
import { useEditorStore } from "@/stores/editor-store";
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
  showRuler: {
    type: Boolean,
    default: true,
  },
  viewResetToken: {
    type: Number,
    default: 0,
  },
});

const emit = defineEmits(["zoomChange"]);

const { width, height, zoom } = toRefs(props);

const containerRef = ref(null);
const wrapperRef = ref(null);
const canvasRef = ref(null);
const editorStore = useEditorStore();
const {
  doc,
  history,
  selection,
  pages,
  currentPageId,
  currentPage,
  docVersion,
} = storeToRefs(editorStore);
const dragState = useDragState();
const minorStep = 10;
const majorStep = 100;
const rulerMax = 5000;
const rulerSize = 18;
const defaultPageMarginX = 72;
const defaultPageMarginY = 52;
const containerSize = ref({ width: 0, height: 0 });
const pointerX = ref(-9999);
const pointerY = ref(-9999);
const isNodeTransforming = ref(false);
const rowInsertEdgeThreshold = 8;
const colInsertEdgeThreshold = 8;
const showInsertLine = ref(false);
const insertLineStyle = ref(null);
const insertLineBox = ref(null);
const rowInsertSnapshot = ref(null);
const layoutInsertSnapshot = ref(null);

/**
 * 拖拽结束时清理插入线
 */
watch(
  () => dragState.dragType,
  (value) => {
    if (!value) {
      showInsertLine.value = false;
      insertLineStyle.value = null;
      insertLineBox.value = null;
      rowInsertSnapshot.value = null;
      layoutInsertSnapshot.value = null;
    }
  },
);

/**
 * 处理鼠标移动，更新标尺指示线
 * @param {MouseEvent} event - 鼠标事件
 */
const handleRulerMouseMove = (event) => {
  if (!props.showRuler) return;
  if (isNodeTransforming.value) return;
  if (!containerRef.value) return;
  const rect = containerRef.value.getBoundingClientRect();
  pointerX.value = Math.min(Math.max(0, event.clientX - rect.left), rect.width);
  pointerY.value = Math.min(Math.max(0, event.clientY - rect.top), rect.height);
};

/**
 * 处理鼠标离开，隐藏标尺指示线
 */
const handleRulerMouseLeave = () => {
  if (!props.showRuler) return;
  if (isNodeTransforming.value) return;
  pointerX.value = -9999;
  pointerY.value = -9999;
};

/**
 * 处理组件拖拽/缩放的标尺指示
 * @param {CustomEvent} event - 自定义事件
 */
const handleNodeTransform = (event) => {
  if (!props.showRuler) return;
  if (!containerRef.value || !event?.detail) return;
  const rect = containerRef.value.getBoundingClientRect();
  const x = Number(event.detail.x) || 0;
  const y = Number(event.detail.y) || 0;
  pointerX.value = Math.min(
    Math.max(0, x * zoom.value + translateX.value + rulerInset.value),
    rect.width,
  );
  pointerY.value = Math.min(
    Math.max(0, y * zoom.value + translateY.value + rulerInset.value),
    rect.height,
  );
  isNodeTransforming.value = true;
};

/**
 * 结束组件拖拽/缩放的标尺指示
 */
const handleNodeTransformEnd = () => {
  isNodeTransforming.value = false;
};

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
  handleDropWithType(event, componentType);
};
const handleGlobalMouseUp = (event) => {
  if (!dragState.dragType) return;
  if (!containerRef.value) {
    endDrag();
    showInsertLine.value = false;
    insertLineStyle.value = null;
    insertLineBox.value = null;
    return;
  }

  if (!containerRef.value.contains(event.target)) {
    endDrag();
    showInsertLine.value = false;
    insertLineStyle.value = null;
    insertLineBox.value = null;
    return;
  }

  const componentType = dragState.dragType;
  handleDropWithType(event, componentType);
};

// 向子组件提供当前缩放比例，用于拖拽落点换算
provide("canvasZoom", zoom);

const rootNodeId = computed(() => currentPage.value?.rootNodeId || "");
const currentPageSnapshot = computed(() => {
  const page = pages.value.find((item) => item.id === currentPageId.value);
  return page || currentPage.value || null;
});
const translateX = ref(0);
const translateY = ref(0);
const rulerInset = computed(() => (props.showRuler ? rulerSize : 0));
const showWorkbenchGrid = computed(() =>
  Boolean(currentPageSnapshot.value?.config?.showGrid),
);
const pointerXOnRuler = computed(() =>
  Math.max(0, pointerX.value - rulerInset.value),
);
const pointerYOnRuler = computed(() =>
  Math.max(0, pointerY.value - rulerInset.value),
);

/**
 * 应用页面优先的默认落点
 * 页面不贴左上，并在可视区有安全边距，首屏视觉更聚焦页面
 */
const applyDefaultPagePlacement = () => {
  const viewportWidth = Math.max(
    0,
    (containerSize.value.width || 0) - rulerInset.value,
  );
  const viewportHeight = Math.max(
    0,
    (containerSize.value.height || 0) - rulerInset.value,
  );
  if (!viewportWidth || !viewportHeight) return;

  const scaledWidth = width.value * zoom.value;
  const scaledHeight = height.value * zoom.value;
  const nextTranslateX =
    scaledWidth + defaultPageMarginX * 2 <= viewportWidth
      ? Math.round(
          defaultPageMarginX +
            (viewportWidth - scaledWidth - defaultPageMarginX * 2) / 2,
        )
      : Math.max(0, Math.round((viewportWidth - scaledWidth) / 2));
  const nextTranslateY =
    scaledHeight + defaultPageMarginY * 2 <= viewportHeight
      ? Math.round(
          defaultPageMarginY +
            (viewportHeight - scaledHeight - defaultPageMarginY * 2) / 2,
        )
      : Math.max(0, Math.round((viewportHeight - scaledHeight) / 2));

  translateX.value = nextTranslateX;
  translateY.value = nextTranslateY;
};

/**
 * 缩放时保持视口中心对应画布点稳定，避免缩放后页面跳角落
 * @param {number} prevZoom - 旧缩放值
 * @param {number} nextZoom - 新缩放值
 */
const keepViewportCenterStableOnZoom = (prevZoom, nextZoom) => {
  const viewportWidth = Math.max(
    0,
    (containerSize.value.width || 0) - rulerInset.value,
  );
  const viewportHeight = Math.max(
    0,
    (containerSize.value.height || 0) - rulerInset.value,
  );
  if (!viewportWidth || !viewportHeight) return;
  if (!prevZoom || !nextZoom || prevZoom === nextZoom) return;

  const centerX = rulerInset.value + viewportWidth / 2;
  const centerY = rulerInset.value + viewportHeight / 2;
  const canvasX = (centerX - rulerInset.value - translateX.value) / prevZoom;
  const canvasY = (centerY - rulerInset.value - translateY.value) / prevZoom;

  translateX.value = Math.round(
    centerX - rulerInset.value - canvasX * nextZoom,
  );
  translateY.value = Math.round(
    centerY - rulerInset.value - canvasY * nextZoom,
  );
};

const workbenchStyle = computed(() => {
  const alpha = props.showRuler ? 0.04 : 0.03;
  const style = {
    backgroundColor: "#eef2f7",
    backgroundImage: "none",
  };
  if (!showWorkbenchGrid.value) return style;
  style.backgroundImage = `linear-gradient(rgba(100,116,139,${alpha}) 1px, transparent 1px), linear-gradient(90deg, rgba(100,116,139,${alpha}) 1px, transparent 1px)`;
  style.backgroundSize = "24px 24px";
  style.backgroundPosition = "0 0";
  return style;
});

watch(
  () => zoom.value,
  (nextZoom, prevZoom) => {
    if (!Number.isFinite(nextZoom) || !Number.isFinite(prevZoom)) return;
    keepViewportCenterStableOnZoom(prevZoom, nextZoom);
  },
);

watch(
  () => rulerInset.value,
  (nextInset, prevInset) => {
    if (!Number.isFinite(nextInset) || !Number.isFinite(prevInset)) return;
    const delta = prevInset - nextInset;
    translateX.value += delta;
    translateY.value += delta;
  },
);

watch(
  [
    () => rootNodeId.value,
    () => width.value,
    () => height.value,
    () => props.viewResetToken,
  ],
  () => {
    applyDefaultPagePlacement();
  },
  { immediate: true },
);

watch(
  () => [containerSize.value.width, containerSize.value.height],
  () => {
    applyDefaultPagePlacement();
  },
);

const canvasStyle = computed(() => {
  docVersion.value;
  const config = currentPageSnapshot.value?.config || {};
  const background = config.background || null;
  const showGrid = Boolean(config.showGrid);
  const style = {
    width: `${width.value}px`,
    height: `${height.value}px`,
    transform: `translate(${translateX.value + rulerInset.value}px, ${
      translateY.value + rulerInset.value
    }px) scale(${zoom.value})`,
    backgroundColor: "#ffffff",
    border: "1px solid rgba(148, 163, 184, 0.45)",
    boxShadow:
      "0 0 0 1px rgba(255,255,255,0.85) inset, 0 10px 26px rgba(15, 23, 42, 0.08)",
  };

  if (background?.kind === "color") {
    style.backgroundColor = background.value || "#ffffff";
  } else if (background?.kind === "image") {
    style.backgroundImage = `url(${background.value || ""})`;
    style.backgroundSize = "cover";
    style.backgroundRepeat = "no-repeat";
    style.backgroundPosition = "center";
  } else if (background?.kind === "gradient") {
    style.backgroundImage = background.value || "";
    style.backgroundSize = "cover";
    style.backgroundRepeat = "no-repeat";
    style.backgroundPosition = "center";
  }

  if (showGrid) {
    const minorStepSize = 12;
    const majorStepSize = 48;
    const gridLayer = `
      linear-gradient(rgba(71, 85, 105, 0.12) 1px, transparent 1px),
      linear-gradient(90deg, rgba(71, 85, 105, 0.12) 1px, transparent 1px),
      linear-gradient(rgba(71, 85, 105, 0.2) 1px, transparent 1px),
      linear-gradient(90deg, rgba(71, 85, 105, 0.2) 1px, transparent 1px)
    `;
    if (style.backgroundImage) {
      style.backgroundImage = `${gridLayer}, ${style.backgroundImage}`;
      style.backgroundSize = `${minorStepSize}px ${minorStepSize}px, ${minorStepSize}px ${minorStepSize}px, ${majorStepSize}px ${majorStepSize}px, ${majorStepSize}px ${majorStepSize}px, ${style.backgroundSize || "cover"}`;
      style.backgroundRepeat = `repeat, repeat, repeat, repeat, ${style.backgroundRepeat || "no-repeat"}`;
      style.backgroundPosition = `0 0, 0 0, 0 0, 0 0, ${style.backgroundPosition || "center"}`;
    } else {
      style.backgroundImage = gridLayer;
      style.backgroundSize = `${minorStepSize}px ${minorStepSize}px, ${minorStepSize}px ${minorStepSize}px, ${majorStepSize}px ${majorStepSize}px, ${majorStepSize}px ${majorStepSize}px`;
    }
  }

  return style;
});

/**
 * 计算滚动内容尺寸，确保缩放后能触发滚动条
 */
const scrollContentStyle = computed(() => {
  const scaledWidth = width.value * zoom.value;
  const scaledHeight = height.value * zoom.value;
  const viewportWidth = Math.max(
    0,
    (containerSize.value.width || 0) - rulerInset.value,
  );
  const viewportHeight = Math.max(
    0,
    (containerSize.value.height || 0) - rulerInset.value,
  );
  // 右/下编辑扩展区保持“可编辑但不过度”，避免滚动后空白区域喧宾夺主
  const workspaceExtraRight = Math.max(
    24,
    Math.min(68, Math.round(viewportWidth * 0.09)),
  );
  const workspaceExtraBottom = Math.max(
    28,
    Math.min(76, Math.round(viewportHeight * 0.1)),
  );
  const coverageX = scaledWidth / Math.max(1, viewportWidth);
  const coverageY = scaledHeight / Math.max(1, viewportHeight);
  const pageStartX = rulerInset.value + translateX.value;
  const pageStartY = rulerInset.value + translateY.value;
  const offsetX = Math.max(rulerInset.value, pageStartX);
  const offsetY = Math.max(rulerInset.value, pageStartY);
  const baseWidth = Math.ceil(scaledWidth + offsetX);
  const baseHeight = Math.ceil(scaledHeight + offsetY);
  const minWidth = containerSize.value.width || 0;
  const minHeight = containerSize.value.height || 0;
  // 当页面已经完整落在当前视口内时，不再追加右/下扩展区，避免出现“适配后右侧灰条”
  const effectiveExtraRight =
    baseWidth <= minWidth
      ? 0
      : coverageX >= 1.6
        ? 0
        : coverageX >= 1.2
          ? Math.round(workspaceExtraRight * 0.25)
          : workspaceExtraRight;
  const effectiveExtraBottom =
    baseHeight <= minHeight
      ? 0
      : coverageY >= 1.6
        ? 0
        : coverageY >= 1.2
          ? Math.round(workspaceExtraBottom * 0.28)
          : workspaceExtraBottom;
  return {
    width: `${Math.max(minWidth, baseWidth + effectiveExtraRight)}px`,
    height: `${Math.max(minHeight, baseHeight + effectiveExtraBottom)}px`,
  };
});

const rulerXStyle = computed(() => {
  const minor = minorStep * zoom.value;
  const major = majorStep * zoom.value;
  return {
    "--ruler-size": `${rulerInset.value}px`,
    "--ruler-minor": `${minor}px`,
    "--ruler-major": `${major}px`,
    "--ruler-offset": `${translateX.value}px`,
  };
});

const rulerYStyle = computed(() => {
  const minor = minorStep * zoom.value;
  const major = majorStep * zoom.value;
  return {
    "--ruler-size": `${rulerInset.value}px`,
    "--ruler-minor": `${minor}px`,
    "--ruler-major": `${major}px`,
    "--ruler-offset": `${translateY.value}px`,
  };
});

const rulerMarksX = computed(() => {
  const marks = [];
  const max = rulerMax;
  for (let value = 0; value <= max; value += majorStep) {
    const pos = value * zoom.value + translateX.value + rulerInset.value;
    if (pos < -majorStep || pos > containerSize.value.width) continue;
    marks.push(value);
  }
  return marks;
});

const rulerMarksY = computed(() => {
  const marks = [];
  const max = rulerMax;
  for (let value = 0; value <= max; value += majorStep) {
    const pos = value * zoom.value + translateY.value + rulerInset.value;
    if (pos < -majorStep || pos > containerSize.value.height) continue;
    marks.push(value);
  }
  return marks;
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
  componentType = componentType || fallbackType;
  if (!componentType) {
    showInsertLine.value = false;
    insertLineStyle.value = null;
    insertLineBox.value = null;
    rowInsertSnapshot.value = null;
    layoutInsertSnapshot.value = null;
    return;
  }

  const rowInsertTarget =
    componentType !== "ElCol" ? resolveRowInsertTarget(event) : null;
  if (rowInsertTarget?.insertLine && rowInsertTarget?.lineBox) {
    showInsertLine.value = true;
    insertLineStyle.value = rowInsertTarget.insertLine;
    insertLineBox.value = rowInsertTarget.lineBox;
    rowInsertSnapshot.value = rowInsertTarget;
    layoutInsertSnapshot.value = null;
    return;
  }

  const layoutInsertTarget =
    componentType !== "ElLayoutRow" ? resolveLayoutInsertTarget(event) : null;
  if (layoutInsertTarget?.insertLine && layoutInsertTarget?.lineBox) {
    showInsertLine.value = true;
    insertLineStyle.value = layoutInsertTarget.insertLine;
    insertLineBox.value = layoutInsertTarget.lineBox;
    layoutInsertSnapshot.value = layoutInsertTarget;
    rowInsertSnapshot.value = null;
    return;
  }

  showInsertLine.value = false;
  insertLineStyle.value = null;
  insertLineBox.value = null;
  rowInsertSnapshot.value = null;
  layoutInsertSnapshot.value = null;
};

/**
 * 在 ElLayout 内插入组件
 * @param {import('@/editor-core').ComponentNode} layoutNode - 布局节点
 * @param {string} componentType - 组件类型
 */
const insertIntoElLayout = (layoutNode, componentType) => {
  if (!layoutNode) return;
  const rowIds = (layoutNode.children || []).filter((childId) => {
    const childNode = doc.value?.getNode?.(childId);
    return childNode?.type === "ElLayoutRow";
  });
  let rowId = rowIds[0];
  if (!rowId) {
    const rowNode = editorStore.insertNode("ElLayoutRow", layoutNode.id, 0);
    if (!rowNode) return;
    const latestLayout = doc.value?.getNode?.(layoutNode.id);
    const rowCount = (latestLayout?.children || []).filter((childId) => {
      const childNode = doc.value?.getNode?.(childId);
      return childNode?.type === "ElLayoutRow";
    }).length;
    editorStore.updateNode(layoutNode.id, {
      props: {
        ...(latestLayout?.props || layoutNode.props || {}),
        rows: Math.max(1, rowCount),
      },
    });
    editorStore.updateNode(rowNode.id, {
      props: { ...(rowNode.props || {}), columns: 1 },
    });
    rowId = rowNode.id;
  }
  const rowNode = doc.value?.getNode?.(rowId);
  if (!rowNode) return;
  insertIntoElLayoutRow(rowNode, componentType);
};

/**
 * 在 ElLayoutRow 内插入组件
 * @param {import('@/editor-core').ComponentNode} rowNode - 行节点
 * @param {string} componentType - 组件类型
 */
const insertIntoElLayoutRow = (rowNode, componentType) => {
  if (!rowNode) return;
  const colIds = (rowNode.children || []).filter((childId) => {
    const childNode = doc.value?.getNode?.(childId);
    return childNode?.type === "ElCol";
  });
  let colId = colIds[0];
  if (!colId) {
    const colNode = editorStore.insertNode("ElCol", rowNode.id, 0);
    colId = colNode?.id || "";
    if (colId) {
      const latestRow = doc.value?.getNode?.(rowNode.id);
      const colCount = (latestRow?.children || []).filter((childId) => {
        const childNode = doc.value?.getNode?.(childId);
        return childNode?.type === "ElCol";
      }).length;
      editorStore.updateNode(rowNode.id, {
        props: {
          ...(latestRow?.props || rowNode.props || {}),
          columns: Math.max(1, colCount),
        },
      });
    }
  }
  if (!colId) return;
  editorStore.insertNode(componentType, colId);
};

/**
 * 解析 ElLayout 行插入目标（靠近上下边缘）
 * @param {DragEvent} event - 拖拽事件
 * @returns {{ layoutNode: import('@/editor-core').ComponentNode, index: number } | null}
 */
const resolveLayoutInsertTarget = (event) => {
  if (!doc.value) return null;
  const hitList = document.elementsFromPoint(event.clientX, event.clientY);
  for (const hit of hitList) {
    if (!(hit instanceof Element)) continue;
    const layoutElement = hit.closest?.(
      '[data-node-type="ElLayout"][data-node-id]',
    );
    if (!layoutElement) continue;
    const layoutId = layoutElement.getAttribute("data-node-id");
    const layoutNode = layoutId ? doc.value.getNode?.(layoutId) : null;
    if (!layoutNode || layoutNode.type !== "ElLayout") continue;
    const rowIds = (layoutNode.children || []).filter((childId) => {
      const childNode = doc.value?.getNode?.(childId);
      return childNode?.type === "ElLayoutRow";
    });
    if (rowIds.length === 0) {
      return { layoutNode, index: 0 };
    }
    const pointY = event.clientY;
    for (let i = 0; i < rowIds.length; i += 1) {
      const rowId = rowIds[i];
      const rowElement = layoutElement.querySelector(
        `[data-node-id="${rowId}"]`,
      );
      if (!rowElement) continue;
      const rect = rowElement.getBoundingClientRect?.();
      if (!rect) continue;
      if (Math.abs(pointY - rect.top) <= rowInsertEdgeThreshold) {
        const layoutRect = layoutElement.getBoundingClientRect?.();
        if (!layoutRect) return { layoutNode, index: i };
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
            orientation: "horizontal",
            offset: Math.max(0, rect.top - layoutRect.top),
          },
        };
      }
      if (Math.abs(pointY - rect.bottom) <= rowInsertEdgeThreshold) {
        const layoutRect = layoutElement.getBoundingClientRect?.();
        if (!layoutRect) return { layoutNode, index: i + 1 };
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
            orientation: "horizontal",
            offset: Math.max(0, rect.bottom - layoutRect.top),
          },
        };
      }
    }
  }
  return null;
};

/**
 * 解析 ElLayoutRow 列插入目标（靠近左右边缘）
 * @param {DragEvent} event - 拖拽事件
 * @returns {{ rowNode: import('@/editor-core').ComponentNode, index: number } | null}
 */
const resolveRowInsertTarget = (event) => {
  if (!doc.value) return null;
  const primaryHit = document.elementFromPoint(event.clientX, event.clientY);
  if (primaryHit instanceof Element) {
    const rowElement = primaryHit.closest?.(
      '[data-node-type="ElLayoutRow"][data-node-id]',
    );
    if (rowElement) {
      const rowId = rowElement.getAttribute("data-node-id");
      const rowNode = rowId ? doc.value?.getNode?.(rowId) : null;
      const rowRect = rowElement.getBoundingClientRect?.();
      if (rowNode?.type === "ElLayoutRow" && rowRect) {
        const nearLeft = event.clientX - rowRect.left <= colInsertEdgeThreshold;
        const nearRight =
          rowRect.right - event.clientX <= colInsertEdgeThreshold;
        if (nearLeft || nearRight) {
          const colIds = (rowNode.children || []).filter((childId) => {
            const childNode = doc.value?.getNode?.(childId);
            return childNode?.type === "ElCol";
          });
          const index = nearLeft ? 0 : colIds.length;
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
              orientation: "vertical",
              offset: Math.max(
                0,
                (nearLeft ? rowRect.left : rowRect.right) - rowRect.left,
              ),
            },
          };
        }
      }
    }
  }
  const hitList = document.elementsFromPoint(event.clientX, event.clientY);
  for (const hit of hitList) {
    if (!(hit instanceof Element)) continue;
    const colElement = hit.closest?.('[data-node-type="ElCol"][data-node-id]');
    if (!colElement) continue;
    const colId = colElement.getAttribute("data-node-id");
    const colNode = colId ? doc.value.getNode?.(colId) : null;
    if (!colNode) continue;
    const rowNode = doc.value.getParent?.(colNode.id);
    if (!rowNode || rowNode.type !== "ElLayoutRow") continue;
    const colRect = colElement.getBoundingClientRect?.();
    if (!colRect) continue;
    const rowElement =
      colElement.closest?.(`[data-node-id="${rowNode.id}"]`) ||
      document.querySelector(`[data-node-id="${rowNode.id}"]`);
    const rowRect = rowElement?.getBoundingClientRect?.();
    if (rowRect) {
      const nearRowLeft =
        event.clientX - rowRect.left <= colInsertEdgeThreshold;
      const nearRowRight =
        rowRect.right - event.clientX <= colInsertEdgeThreshold;
      if (nearRowLeft || nearRowRight) {
        const colIds = (rowNode.children || []).filter((childId) => {
          const childNode = doc.value?.getNode?.(childId);
          return childNode?.type === "ElCol";
        });
        const index = nearRowLeft ? 0 : colIds.length;
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
            orientation: "vertical",
            offset: Math.max(
              0,
              (nearRowLeft ? rowRect.left : rowRect.right) - rowRect.left,
            ),
          },
        };
      }
    }
    const nearLeft = event.clientX - colRect.left <= colInsertEdgeThreshold;
    const nearRight = colRect.right - event.clientX <= colInsertEdgeThreshold;
    if (!nearLeft && !nearRight) continue;
    const colIds = (rowNode.children || []).filter((childId) => {
      const childNode = doc.value?.getNode?.(childId);
      return childNode?.type === "ElCol";
    });
    const currentIndex = colIds.indexOf(colNode.id);
    if (currentIndex === -1) continue;
    const index = nearLeft ? currentIndex : currentIndex + 1;
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
            orientation: "vertical",
            offset: Math.max(
              0,
              (nearLeft ? colRect.left : colRect.right) - rowRect.left,
            ),
          }
        : null,
    };
  }
  for (const hit of hitList) {
    if (!(hit instanceof Element)) continue;
    const rowElement = hit.closest?.(
      '[data-node-type="ElLayoutRow"][data-node-id]',
    );
    if (!rowElement) continue;
    const rowId = rowElement.getAttribute("data-node-id");
    const rowNode = rowId ? doc.value?.getNode?.(rowId) : null;
    if (!rowNode || rowNode.type !== "ElLayoutRow") continue;
    const rowRect = rowElement.getBoundingClientRect?.();
    if (!rowRect) continue;
    const nearLeft = event.clientX - rowRect.left <= colInsertEdgeThreshold;
    const nearRight = rowRect.right - event.clientX <= colInsertEdgeThreshold;
    if (!nearLeft && !nearRight) continue;
    const colIds = (rowNode.children || []).filter((childId) => {
      const childNode = doc.value?.getNode?.(childId);
      return childNode?.type === "ElCol";
    });
    const index = nearLeft ? 0 : colIds.length;
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
        orientation: "vertical",
        offset: Math.max(
          0,
          (nearLeft ? rowRect.left : rowRect.right) - rowRect.left,
        ),
      },
    };
  }
  return null;
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

  handleDropWithType(event, componentType);
};

/**
 * 统一处理拖拽放置逻辑
 * @param {DragEvent} event - 拖拽事件
 * @param {string} componentType - 组件类型
 */
const handleDropWithType = (event, componentType) => {
  showInsertLine.value = false;
  insertLineStyle.value = null;
  insertLineBox.value = null;
  const cachedRowInsert = rowInsertSnapshot.value;
  const cachedLayoutInsert = layoutInsertSnapshot.value;
  rowInsertSnapshot.value = null;
  layoutInsertSnapshot.value = null;
  const layoutInsertTarget =
    componentType !== "ElLayoutRow"
      ? cachedLayoutInsert || resolveLayoutInsertTarget(event)
      : null;
  if (layoutInsertTarget) {
    const rowNode = editorStore.insertNode(
      "ElLayoutRow",
      layoutInsertTarget.layoutNode.id,
      layoutInsertTarget.index,
    );
    if (rowNode) {
      const latestLayout = doc.value?.getNode?.(
        layoutInsertTarget.layoutNode.id,
      );
      const rowCount = (latestLayout?.children || []).filter((childId) => {
        const childNode = doc.value?.getNode?.(childId);
        return childNode?.type === "ElLayoutRow";
      }).length;
      editorStore.updateNode(layoutInsertTarget.layoutNode.id, {
        props: {
          ...(latestLayout?.props || layoutInsertTarget.layoutNode.props || {}),
          rows: Math.max(1, rowCount),
        },
      });
      editorStore.updateNode(rowNode.id, {
        props: { ...(rowNode.props || {}), columns: 1 },
      });
      const latestRow = doc.value?.getNode?.(rowNode.id);
      const colIds = (latestRow?.children || []).filter((childId) => {
        const childNode = doc.value?.getNode?.(childId);
        return childNode?.type === "ElCol";
      });
      let colId = colIds[0];
      if (!colId) {
        const colNode = editorStore.insertNode("ElCol", rowNode.id, 0);
        colId = colNode?.id || "";
      }
      if (colId) {
        editorStore.insertNode(componentType, colId);
      }
    }
    endDrag();
    return;
  }

  const rowInsertTarget =
    componentType !== "ElCol"
      ? cachedRowInsert || resolveRowInsertTarget(event)
      : null;
  if (rowInsertTarget) {
    const colNode = editorStore.insertNode(
      "ElCol",
      rowInsertTarget.rowNode.id,
      rowInsertTarget.index,
    );
    if (colNode) {
      const latestRow = doc.value?.getNode?.(rowInsertTarget.rowNode.id);
      const colCount = (latestRow?.children || []).filter((childId) => {
        const childNode = doc.value?.getNode?.(childId);
        return childNode?.type === "ElCol";
      }).length;
      editorStore.updateNode(rowInsertTarget.rowNode.id, {
        props: {
          ...(latestRow?.props || rowInsertTarget.rowNode.props || {}),
          columns: Math.max(1, colCount),
        },
      });
      editorStore.insertNode(componentType, colNode.id);
    }
    endDrag();
    return;
  }

  const target = resolveDropTarget(event, componentType);
  if (!target.nodeId || !target.element) {
    endDrag();
    return;
  }
  const targetNode = doc.value?.getNode?.(target.nodeId);
  if (targetNode?.type === "ElCol" && componentType !== "ElCol") {
    if ((targetNode.children || []).length > 0) {
      const rowNode = doc.value?.getParent?.(targetNode.id);
      if (rowNode?.type === "ElLayoutRow") {
        const rowInsertTarget =
          componentType !== "ElCol"
            ? cachedRowInsert || resolveRowInsertTarget(event)
            : null;
        const colIds = (rowNode.children || []).filter((childId) => {
          const childNode = doc.value?.getNode?.(childId);
          return childNode?.type === "ElCol";
        });
        const currentIndex = Math.max(0, colIds.indexOf(targetNode.id));
        let insertIndex = currentIndex + 1;
        if (rowInsertTarget?.rowNode?.id === rowNode.id) {
          insertIndex = rowInsertTarget.index;
        }
        const colNode = editorStore.insertNode(
          "ElCol",
          rowNode.id,
          insertIndex,
        );
        if (colNode) {
          const latestRow = doc.value?.getNode?.(rowNode.id);
          const colCount = (latestRow?.children || []).filter((childId) => {
            const childNode = doc.value?.getNode?.(childId);
            return childNode?.type === "ElCol";
          }).length;
          editorStore.updateNode(rowNode.id, {
            props: {
              ...(latestRow?.props || rowNode.props || {}),
              columns: Math.max(1, colCount),
            },
          });
          editorStore.insertNode(componentType, colNode.id);
        }
        endDrag();
        return;
      }
    }
    const rowNode = doc.value?.getParent?.(targetNode.id);
    if (rowNode?.type === "ElLayoutRow") {
      const rowElement = document.querySelector(
        `[data-node-id="${rowNode.id}"]`,
      );
      const rowRect = rowElement?.getBoundingClientRect?.();
      if (rowRect) {
        const nearLeft = event.clientX - rowRect.left <= colInsertEdgeThreshold;
        const nearRight =
          rowRect.right - event.clientX <= colInsertEdgeThreshold;
        if (nearLeft || nearRight) {
          const colIds = (rowNode.children || []).filter((childId) => {
            const childNode = doc.value?.getNode?.(childId);
            return childNode?.type === "ElCol";
          });
          const insertIndex = nearLeft ? 0 : colIds.length;
          const colNode = editorStore.insertNode(
            "ElCol",
            rowNode.id,
            insertIndex,
          );
          if (colNode) {
            const latestRow = doc.value?.getNode?.(rowNode.id);
            const colCount = (latestRow?.children || []).filter((childId) => {
              const childNode = doc.value?.getNode?.(childId);
              return childNode?.type === "ElCol";
            }).length;
            editorStore.updateNode(rowNode.id, {
              props: {
                ...(latestRow?.props || rowNode.props || {}),
                columns: Math.max(1, colCount),
              },
            });
            editorStore.insertNode(componentType, colNode.id);
          }
          endDrag();
          return;
        }
      }
    }
  }
  if (targetNode?.type === "ElLayout") {
    const nearestCol = resolveNearestElCol(event);
    if (nearestCol) {
      if ((nearestCol.children || []).length > 0) {
        const rowNode = doc.value?.getParent?.(nearestCol.id);
        if (rowNode?.type === "ElLayoutRow") {
          const rowInsertTarget =
            componentType !== "ElCol"
              ? cachedRowInsert || resolveRowInsertTarget(event)
              : null;
          let insertIndex = (rowNode.children || []).length;
          if (rowInsertTarget?.rowNode?.id === rowNode.id) {
            insertIndex = rowInsertTarget.index;
          } else {
            const colIds = (rowNode.children || []).filter((childId) => {
              const childNode = doc.value?.getNode?.(childId);
              return childNode?.type === "ElCol";
            });
            const currentIndex = Math.max(0, colIds.indexOf(nearestCol.id));
            insertIndex = currentIndex + 1;
          }
          const colNode = editorStore.insertNode(
            "ElCol",
            rowNode.id,
            insertIndex,
          );
          if (colNode) {
            const latestRow = doc.value?.getNode?.(rowNode.id);
            const colCount = (latestRow?.children || []).filter((childId) => {
              const childNode = doc.value?.getNode?.(childId);
              return childNode?.type === "ElCol";
            }).length;
            editorStore.updateNode(rowNode.id, {
              props: {
                ...(latestRow?.props || rowNode.props || {}),
                columns: Math.max(1, colCount),
              },
            });
            editorStore.insertNode(componentType, colNode.id);
          }
          endDrag();
          return;
        }
      } else {
        editorStore.insertNode(componentType, nearestCol.id);
        endDrag();
        return;
      }
    }
    const rowTarget = resolveLayoutRowByPoint(
      targetNode,
      target.element,
      event,
    );
    if (rowTarget) {
      insertIntoElLayoutRow(rowTarget, componentType);
      endDrag();
      return;
    }
    insertIntoElLayout(targetNode, componentType);
    endDrag();
    return;
  }
  if (targetNode?.type === "ElLayoutRow" && componentType !== "ElCol") {
    const nearestCol = resolveNearestElCol(event);
    if (nearestCol) {
      if ((nearestCol.children || []).length > 0) {
        const rowInsertTarget =
          componentType !== "ElCol"
            ? cachedRowInsert || resolveRowInsertTarget(event)
            : null;
        let insertIndex = (targetNode.children || []).length;
        if (rowInsertTarget?.rowNode?.id === targetNode.id) {
          insertIndex = rowInsertTarget.index;
        } else {
          const colIds = (targetNode.children || []).filter((childId) => {
            const childNode = doc.value?.getNode?.(childId);
            return childNode?.type === "ElCol";
          });
          const currentIndex = Math.max(0, colIds.indexOf(nearestCol.id));
          insertIndex = currentIndex + 1;
        }
        const colNode = editorStore.insertNode(
          "ElCol",
          targetNode.id,
          insertIndex,
        );
        if (colNode) {
          const latestRow = doc.value?.getNode?.(targetNode.id);
          const colCount = (latestRow?.children || []).filter((childId) => {
            const childNode = doc.value?.getNode?.(childId);
            return childNode?.type === "ElCol";
          }).length;
          editorStore.updateNode(targetNode.id, {
            props: {
              ...(latestRow?.props || targetNode.props || {}),
              columns: Math.max(1, colCount),
            },
          });
          editorStore.insertNode(componentType, colNode.id);
        }
        endDrag();
        return;
      }
      editorStore.insertNode(componentType, nearestCol.id);
      endDrag();
      return;
    }
    insertIntoElLayoutRow(targetNode, componentType);
    endDrag();
    return;
  }

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
    "ElContainer",
    "ElLayout",
    "ElLayoutRow",
  ].includes(type);
};

/**
 * 插入组件节点
 * @param {string} type - 组件类型
 * @param {string} parentId - 父节点ID
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

  const step = 0.1;
  const direction = event.deltaY > 0 ? -1 : 1;
  const nextZoom = Math.min(5, Math.max(0.1, zoom.value + step * direction));
  if (nextZoom === zoom.value) return;

  emit("zoomChange", Number(nextZoom.toFixed(2)));
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
 * @param {import('@/editor-core').ComponentNode | null} parentNode - 父节点 * @param {{x: number, y: number, width: number, height: number}} dropInfo - 放置信息
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

  if (
    parentNode.type === "FlexContainer" ||
    parentNode.type === "ResponsiveLayout" ||
    parentNode.type === "ElContainer" ||
    parentNode.type === "ElLayout" ||
    parentNode.type === "ElLayoutRow" ||
    parentNode.type === "ElHeader" ||
    parentNode.type === "ElAside" ||
    parentNode.type === "ElMain" ||
    parentNode.type === "ElFooter" ||
    parentNode.type === "ElCol"
  ) {
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
 * @param {import('@/editor-core').ComponentNode} parentNode - 父节点 * @returns {import('@/editor-core').LayoutItem}
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
 * @param {string | number | undefined} value - 列配置 * @returns {number}
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

  const hitList = document.elementsFromPoint(event.clientX, event.clientY);
  for (const hit of hitList) {
    if (!(hit instanceof Element)) continue;
    const nodeElement = hit.closest?.("[data-node-id][data-node-type]");
    if (!nodeElement) continue;
    const nodeId = nodeElement.getAttribute("data-node-id");
    if (!nodeId) continue;
    const node = doc.value?.getNode?.(nodeId);
    if (!node) continue;
    if (node.type === "ElCol") {
      return { nodeId, element: nodeElement };
    }
    if (node.type === "ElLayoutRow" || node.type === "ElLayout") {
      return { nodeId, element: nodeElement };
    }
  }

  const hit = document.elementFromPoint(event.clientX, event.clientY);
  let current = hit;

  while (current && current !== canvasRef.value) {
    const nodeId = current.dataset?.nodeId;
    if (
      nodeId &&
      isContainerNode(nodeId) &&
      canAcceptChild(nodeId, componentType)
    ) {
      return { nodeId, element: current };
    }
    current = current.parentElement;
  }

  return { nodeId: rootNodeId.value, element: canvasRef.value };
};

/**
 * 解析鼠标下最近的 ElCol
 * @param {DragEvent} event - 拖拽事件
 * @returns {import('@/editor-core').ComponentNode | null}
 */
const resolveNearestElCol = (event) => {
  if (!doc.value) return null;
  const hitList = document.elementsFromPoint(event.clientX, event.clientY);
  for (const hit of hitList) {
    if (!(hit instanceof Element)) continue;
    const colElement = hit.closest?.('[data-node-type="ElCol"][data-node-id]');
    if (!colElement) continue;
    const colId = colElement.getAttribute("data-node-id");
    if (!colId) continue;
    const colNode = doc.value?.getNode?.(colId);
    if (colNode?.type === "ElCol") return colNode;
  }
  return null;
};

/**
 * 根据鼠标位置解析 ElLayout 内最接近的行
 * @param {import('@/editor-core').ComponentNode} layoutNode - 布局节点
 * @param {HTMLElement | null} layoutElement - 布局元素
 * @param {DragEvent} event - 拖拽事件
 * @returns {import('@/editor-core').ComponentNode | null}
 */
const resolveLayoutRowByPoint = (layoutNode, layoutElement, event) => {
  if (!doc.value || !layoutNode || layoutNode.type !== "ElLayout") return null;
  if (!layoutElement) return null;
  const rowIds = (layoutNode.children || []).filter((childId) => {
    const childNode = doc.value?.getNode?.(childId);
    return childNode?.type === "ElLayoutRow";
  });
  let bestRow = null;
  let bestDistance = Number.POSITIVE_INFINITY;
  for (const rowId of rowIds) {
    const rowElement = layoutElement.querySelector(`[data-node-id="${rowId}"]`);
    if (!rowElement) continue;
    const rect = rowElement.getBoundingClientRect?.();
    if (!rect) continue;
    if (event.clientY >= rect.top && event.clientY <= rect.bottom) {
      return doc.value?.getNode?.(rowId) || null;
    }
    const distance = Math.min(
      Math.abs(event.clientY - rect.top),
      Math.abs(event.clientY - rect.bottom),
    );
    if (distance < bestDistance) {
      bestDistance = distance;
      bestRow = doc.value?.getNode?.(rowId) || null;
    }
  }
  return bestRow;
};

/**
 * 判断节点是否为容器 * @param {string} nodeId - 节点 ID
 * @returns {boolean}
 */
const isContainerNode = (nodeId) => {
  const node = doc.value?.getNode(nodeId);
  if (!node) return false;
  const manifest = componentRegistry.get(node.type);
  return Boolean(manifest?.isContainer);
};

/**
 * 判断容器是否允许子组件 * @param {string} parentId - 父节点ID
 * @param {string} childType - 子组件类型 * @returns {boolean}
 */
const canAcceptChild = (parentId, childType) => {
  const node = doc.value?.getNode(parentId);
  if (!node) return false;
  if (node.type === "ElLayout") {
    return childType === "ElLayoutRow";
  }
  if (node.type === "ElLayoutRow") {
    return childType === "ElCol";
  }
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
    ElContainer: { width: 360, height: 240 },
    ElLayout: { width: 360, height: 200 },
    Text: { width: 120, height: 32 },
    Button: { width: 120, height: 36 },
  };

  return sizeMap[type] || { width: 160, height: 80 };
};
onMounted(() => {
  window.addEventListener("dragover", handleGlobalDragOver);
  window.addEventListener("drop", handleGlobalDrop);
  window.addEventListener("mouseup", handleGlobalMouseUp);
  window.addEventListener("designer:node-transform", handleNodeTransform);
  window.addEventListener(
    "designer:node-transform-end",
    handleNodeTransformEnd,
  );
  if (containerRef.value && typeof ResizeObserver !== "undefined") {
    const observer = new ResizeObserver((entries) => {
      const entry = entries[0];
      if (!entry) return;
      const { width: w, height: h } = entry.contentRect;
      containerSize.value = { width: w, height: h };
    });
    observer.observe(containerRef.value);
    containerRef.value.__rulerObserver = observer;
  }
});

onBeforeUnmount(() => {
  window.removeEventListener("dragover", handleGlobalDragOver);
  window.removeEventListener("drop", handleGlobalDrop);
  window.removeEventListener("mouseup", handleGlobalMouseUp);
  window.removeEventListener("designer:node-transform", handleNodeTransform);
  window.removeEventListener(
    "designer:node-transform-end",
    handleNodeTransformEnd,
  );
  if (containerRef.value?.__rulerObserver) {
    containerRef.value.__rulerObserver.disconnect();
    containerRef.value.__rulerObserver = null;
  }
});
</script>

<style scoped>
.canvas-container {
  position: relative;
  height: 100%;
  min-height: 0;
}

.canvas-wrapper {
  position: relative;
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

.dark .canvas {
}
.ruler-layer {
  position: absolute;
  inset: 0;
  pointer-events: none;
  z-index: 5;
  --ruler-size: 18px;
}
.canvas-insert-line {
  position: absolute;
  background: #ef4444;
  pointer-events: none;
  z-index: 9999;
  transition: all 0.1s ease;
}

.canvas-insert-line.horizontal {
  height: 2px;
  left: 0;
  right: 0;
}

.canvas-insert-line.vertical {
  width: 2px;
  top: 0;
  bottom: 0;
}

.ruler-crosshair-x {
  position: absolute;
  top: 0;
  height: 18px;
  width: 2px;
  background: #409eff;
  opacity: 0.9;
  z-index: 3;
}

.ruler-crosshair-y {
  position: absolute;
  left: 0;
  width: 18px;
  height: 2px;
  background: #409eff;
  opacity: 0.9;
  z-index: 3;
}

.ruler-zero {
  position: absolute;
  left: 0;
  top: 0;
  width: var(--ruler-size);
  height: var(--ruler-size);
  box-sizing: border-box;
  background: #f5f7fa;
  border-right: 1px solid #dcdfe6;
  border-bottom: 1px solid #dcdfe6;
  z-index: 3;
}

.ruler {
  position: absolute;
  color: #606266;
  font-size: 10px;
  background-color: #f5f7fa;
  border-color: #dcdfe6;
  overflow: hidden;
}

.ruler-x {
  left: var(--ruler-size);
  top: 0;
  height: 18px;
  right: 0;
  border-bottom: 1px solid #dcdfe6;
  z-index: 2;
}

.ruler-y {
  left: 0;
  top: var(--ruler-size);
  width: 18px;
  bottom: 0;
  border-right: 1px solid #dcdfe6;
  z-index: 1;
}

.ruler-x::before,
.ruler-x::after,
.ruler-y::before,
.ruler-y::after {
  content: "";
  position: absolute;
  pointer-events: none;
}

.ruler-x::before {
  left: 0;
  right: 0;
  bottom: 0;
  height: 6px;
  background-image: linear-gradient(to right, #c0c4cc 1px, transparent 1px);
  background-size: var(--ruler-minor, 10px) 100%;
  background-position: var(--ruler-offset, 0) 0;
}

.ruler-x::after {
  left: 0;
  right: 0;
  bottom: 0;
  height: 12px;
  background-image: linear-gradient(to right, #909399 1px, transparent 1px);
  background-size: var(--ruler-major, 100px) 100%;
  background-position: var(--ruler-offset, 0) 0;
}

.ruler-y::before {
  top: 0;
  bottom: 0;
  right: 0;
  width: 6px;
  background-image: linear-gradient(to bottom, #c0c4cc 1px, transparent 1px);
  background-size: 100% var(--ruler-minor, 10px);
  background-position: 0 var(--ruler-offset, 0);
}

.ruler-y::after {
  top: 0;
  bottom: 0;
  right: 0;
  width: 12px;
  background-image: linear-gradient(to bottom, #909399 1px, transparent 1px);
  background-size: 100% var(--ruler-major, 100px);
  background-position: 0 var(--ruler-offset, 0);
}

.ruler-label {
  position: absolute;
  padding: 2px 2px 0 2px;
  line-height: 1;
  white-space: nowrap;
}

.ruler-y .ruler-label {
  transform: rotate(-90deg);
  transform-origin: left top;
  left: 2px;
}
</style>
