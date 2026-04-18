<script setup lang="ts">
import type {
  MarqueeModifiers,
  MarqueeStartSource,
  OutsideMarqueeStartDetail,
} from "./interaction/marquee-interaction";
import { ElMessage } from "element-plus";
import { storeToRefs } from "pinia";
/**
 * DesignCanvas - 设计画布组件
 *
 * 职责：
 * - 渲染当前页面的组件树（NodeRenderer）
 * - 提供框选（marquee）多选、右键菜单、快捷键
 * - 处理组件拖拽放置与画布内排序
 * - 支持粘贴到鼠标位置、撤销与重做
 * - 无内容时显示空画布提示
 */
import { computed, inject, onBeforeUnmount, onMounted, provide, ref } from "vue";
import IconEpBottom from "~icons/ep/bottom";
import IconEpDelete from "~icons/ep/delete";
import IconEpPlus from "~icons/ep/plus";
import IconEpRefreshLeft from "~icons/ep/refresh-left";
import IconEpRefreshRight from "~icons/ep/refresh-right";
import IconEpTop from "~icons/ep/top";
import { isContainerType } from "@/editor-core/descriptors/registry";
import { createSelectableElement } from "@/editor-core/document/types";
import { resolvePlacement } from "@/editor-core/utils/placement-resolver";
import { useEditorStore } from "@/stores/editor-store";
import {
  buildAssetNodeProps,
  DESIGNER_ASSET_DRAG_MIME,
  parseAssetDragPayload,
  resolveAssetComponentType,
} from "@/ui/shared/helpers/asset-drag";
import { endDrag, useDragState } from "./composables/use-drag-state";
import { canvasZoomKey } from "./injection-keys";
import { createMarqueeClickGuard } from "./interaction/marquee-click-guard";
import {
  CANVAS_OUTSIDE_MARQUEE_START_EVENT,
  shouldClearSelectionOnMarqueeUp,
  shouldStartMarqueeFromCanvasPointerDown,
} from "./interaction/marquee-interaction";
import { collectMarqueeNodeIds } from "./interaction/marquee-selection";
import NodeRenderer from "./NodeRenderer.vue";

interface MarqueeState {
  active: boolean;
  moved: boolean;
  startX: number;
  startY: number;
  currentX: number;
  currentY: number;
  modifiers: { ctrl: boolean; meta: boolean; shift: boolean };
  startSource: MarqueeStartSource;
}

const editorStore = useEditorStore();
const {
  doc,
  currentPage,
  selection,
  history,
  docVersion,
  selectionVersion,
  error,
  canvasMousePos,
  hoveredNodeType,
} = storeToRefs(editorStore);
const canvasZoom = inject(canvasZoomKey, ref(1));
const dragState = useDragState();
const marqueeClickGuard = createMarqueeClickGuard();
const designCanvasRef = ref<HTMLElement | null>(null);

/** 当前页面根节点 ID */
const rootNodeId = computed(() => currentPage.value?.rootNodeId || "");
/** 框选状态：是否激活、是否移动、起止坐标、修饰键 */
const marquee = ref<MarqueeState>({
  active: false,
  moved: false,
  startX: 0,
  startY: 0,
  currentX: 0,
  currentY: 0,
  modifiers: { ctrl: false, meta: false, shift: false },
  startSource: "insideCanvas",
});
const marqueeStyle = computed(() => {
  const left = Math.min(marquee.value.startX, marquee.value.currentX);
  const top = Math.min(marquee.value.startY, marquee.value.currentY);
  const width = Math.abs(marquee.value.currentX - marquee.value.startX);
  const height = Math.abs(marquee.value.currentY - marquee.value.startY);
  return {
    left: `${left}px`,
    top: `${top}px`,
    width: `${width}px`,
    height: `${height}px`,
  };
});

const hasContent = computed(() => {
  void docVersion.value;
  if (!doc.value || !currentPage.value) return false;
  const root = doc.value.getNode(currentPage.value.rootNodeId);
  return (root?.children || []).length > 0;
});

/** 强制刷新画布（触发 `docVersion` 变更） */
function handleForceRefresh(): void {
  docVersion.value += 1;
}

/** 重置框选状态 */
function resetMarquee(): void {
  marquee.value.active = false;
  marquee.value.moved = false;
  marquee.value.startSource = "insideCanvas";
}

/**
 * 启动框选
 * @param {{
 *   clientX: number;
 *   clientY: number;
 *   modifiers: MarqueeModifiers;
 *   startSource: MarqueeStartSource;
 * }} payload - 框选起点参数
 */
function startMarquee(payload: {
  clientX: number;
  clientY: number;
  modifiers: MarqueeModifiers;
  startSource: MarqueeStartSource;
}): void {
  marquee.value.active = true;
  marquee.value.moved = false;
  marquee.value.startX = payload.clientX;
  marquee.value.startY = payload.clientY;
  marquee.value.currentX = payload.clientX;
  marquee.value.currentY = payload.clientY;
  marquee.value.modifiers = payload.modifiers;
  marquee.value.startSource = payload.startSource;
  document.addEventListener("pointermove", handleMarqueeMove);
  document.addEventListener("pointerup", handleMarqueeUp, { once: false });
}

/** 获取框选矩形的 left/top/right/bottom/width/height */
function getMarqueeRect(): {
  left: number;
  top: number;
  right: number;
  bottom: number;
  width: number;
  height: number;
} {
  const left = Math.min(marquee.value.startX, marquee.value.currentX);
  const top = Math.min(marquee.value.startY, marquee.value.currentY);
  const right = Math.max(marquee.value.startX, marquee.value.currentX);
  const bottom = Math.max(marquee.value.startY, marquee.value.currentY);
  return {
    left,
    top,
    right,
    bottom,
    width: right - left,
    height: bottom - top,
  };
}

/** 收集框选区域内相交的节点（支持单容器穿透） */
function collectIntersectedElements(): ReturnType<typeof createSelectableElement>[] {
  const rect = getMarqueeRect();
  if (rect.width < 2 && rect.height < 2) return [];
  const rootId = rootNodeId.value;
  if (!rootId || !doc.value) return [];
  const hitNodeIds = collectMarqueeNodeIds({
    rootId,
    marqueeRect: {
      left: rect.left,
      top: rect.top,
      right: rect.right,
      bottom: rect.bottom,
    },
    getNode: (id) => doc.value?.getNode?.(id),
    getRect: (id) => {
      const el = document.querySelector(`[data-node-id="${id}"]`);
      if (!el) return null;
      return el.getBoundingClientRect();
    },
    isContainer: (type) => isContainerType(type),
  });

  return hitNodeIds.map((id) => createSelectableElement("node", id));
}

/** 框选过程中更新当前坐标 */
function handleMarqueeMove(event: PointerEvent): void {
  if (!marquee.value.active) return;
  marquee.value.currentX = event.clientX;
  marquee.value.currentY = event.clientY;
  if (
    Math.abs(marquee.value.currentX - marquee.value.startX) > 3 ||
    Math.abs(marquee.value.currentY - marquee.value.startY) > 3
  ) {
    marquee.value.moved = true;
  }
}

/** 框选结束：应用选中结果或点击选中根节点 */
function handleMarqueeUp(): void {
  if (!marquee.value.active) return;
  const moved = marquee.value.moved;
  const modifiers = marquee.value.modifiers;
  const startSource = marquee.value.startSource;
  const rootId = rootNodeId.value;
  resetMarquee();
  document.removeEventListener("pointermove", handleMarqueeMove);
  document.removeEventListener("pointerup", handleMarqueeUp);

  const sel = selection.value;
  if (!sel) return;

  if (moved) {
    // 框选释放后浏览器通常会再派发一次 click，需要吞掉避免覆盖多选结果
    marqueeClickGuard.markShouldSuppressNextClick();
    const elements = collectIntersectedElements();
    if (elements.length) {
      if (modifiers.ctrl || modifiers.meta || modifiers.shift) {
        elements.forEach((el) => sel.addToSelection(el));
      } else {
        sel.selectMultiple(elements);
      }
      return;
    }
    if (!(modifiers.ctrl || modifiers.meta || modifiers.shift)) {
      sel.clearSelection();
    }
    return;
  }

  if (shouldClearSelectionOnMarqueeUp({ startSource, moved })) {
    sel.clearSelection();
    return;
  }

  if (!rootId) {
    sel.clearSelection();
    return;
  }
  const element = createSelectableElement("node", rootId);
  if (modifiers.shift) {
    sel.selectRange(element);
    return;
  }
  if (modifiers.meta || modifiers.ctrl) {
    sel.toggleSelect(element);
    return;
  }
  sel.select(element);
}

/**
 * 全局 click 捕获：消费框选后的“补发 click”，避免把框选结果改成单选或清空
 * @param {MouseEvent} event - 鼠标事件
 */
function handleDocumentClickCapture(event: MouseEvent): void {
  if (!marqueeClickGuard.consumeShouldSuppressNextClick()) return;
  event.preventDefault();
  event.stopPropagation();
}

/**
 * 兜底处理画布点击选中，避免组件内部阻止冒泡导致无法选中
 * @param {PointerEvent | MouseEvent} event - 鼠标事件
 */
function handleCanvasPointerDown(event: PointerEvent): void {
  marqueeClickGuard.reset();
  if (!selection.value) return;
  if (event.pointerType === "mouse" && event.button !== 0) return;
  if (event.currentTarget instanceof HTMLElement) {
    event.currentTarget.focus({ preventScroll: true });
  }
  if (!(event.target instanceof Element)) {
    closeContextMenu();
    selection.value?.clearSelection();
    return;
  }

  closeContextMenu();

  const nodeElement = event.target.closest(".designer-node");
  const shouldStartMarquee = shouldStartMarqueeFromCanvasPointerDown({
    hasNodeElement: Boolean(nodeElement),
    isRootNode: Boolean(nodeElement?.classList.contains("is-root")),
  });
  if (!shouldStartMarquee) {
    return;
  }
  // 启动框选后阻断事件下发到节点层，避免节点拖拽逻辑抢占同一次 pointer 序列
  event.preventDefault();
  event.stopPropagation();
  startMarquee({
    clientX: event.clientX,
    clientY: event.clientY,
    modifiers: {
      ctrl: Boolean(event.ctrlKey),
      meta: Boolean(event.metaKey),
      shift: Boolean(event.shiftKey),
    },
    startSource: "insideCanvas",
  });
}

/**
 * 处理工作台灰区触发的框选开始事件
 * @param {Event} event - 自定义事件
 */
function handleOutsideMarqueeStart(event: Event): void {
  marqueeClickGuard.reset();
  if (!selection.value) return;
  const customEvent = event as CustomEvent<OutsideMarqueeStartDetail>;
  const detail = customEvent.detail;
  if (!detail) return;
  closeContextMenu();
  designCanvasRef.value?.focus?.({ preventScroll: true });
  startMarquee({
    clientX: detail.clientX,
    clientY: detail.clientY,
    modifiers: detail.modifiers,
    startSource: "outsideCanvas",
  });
}

/**
 * 更新画布上的鼠标坐标，以及悬停节点类型
 * @param {PointerEvent} event - 指针事件
 */
function handleCanvasPointerMove(event: PointerEvent): void {
  const el = event.currentTarget;
  if (!el || !(el instanceof Element)) return;
  const rect = el.getBoundingClientRect();
  const zoomValue = Number(canvasZoom?.value) || 1;
  canvasMousePos.value = {
    x: (event.clientX - rect.left) / zoomValue,
    y: (event.clientY - rect.top) / zoomValue,
  };
  // 检测悬停节点类型（用于底部状态栏显示）
  const hitEl = document.elementFromPoint(event.clientX, event.clientY);
  const nodeEl = hitEl?.closest?.("[data-node-type]");
  hoveredNodeType.value = nodeEl?.getAttribute?.("data-node-type") || "";
}

/**
 * 鼠标离开画布时清空坐标与悬停信息
 */
function handleCanvasPointerLeave(): void {
  canvasMousePos.value = null;
  hoveredNodeType.value = "";
}

/** 右键菜单显示状态与坐标 */
const contextMenuVisible = ref(false);
const contextMenuX = ref(0);
const contextMenuY = ref(0);

const hasSelection = computed(() => {
  void selectionVersion.value;
  return (selection.value?.getSelectedElements?.() ?? []).length > 0;
});

const isElColSelected = computed(() => {
  void selectionVersion.value;
  const primary = selection.value?.getPrimarySelection?.();
  if (primary?.type === "ElCol") return true;
  const selectedNodes = selection.value?.getSelectedNodes?.() || [];
  return selectedNodes.some((node) => node?.type === "ElCol");
});
const isElLayoutRowSelected = computed(() => {
  void selectionVersion.value;
  const primary = selection.value?.getPrimarySelection?.();
  if (primary?.type === "ElLayoutRow") return true;
  const selectedNodes = selection.value?.getSelectedNodes?.() || [];
  return selectedNodes.some((node) => node?.type === "ElLayoutRow");
});

const canUndo = computed(() => history.value?.canUndo?.() || false);
const canRedo = computed(() => history.value?.canRedo?.() || false);

/**
 * 处理画布空白区域放置组件
 * @param {DragEvent} event - 拖拽事件
 */
function handleCanvasDrop(event: DragEvent): void {
  if (!currentPage.value?.rootNodeId) return;
  const assetPayload = parseAssetDragPayload(event.dataTransfer?.getData(DESIGNER_ASSET_DRAG_MIME));

  const payload =
    event.dataTransfer?.getData("application/x-designer-component") ||
    event.dataTransfer?.getData("application/x-designer-node") ||
    event.dataTransfer?.getData("text/plain");
  const fallbackType = dragState.dragType || "";
  const assetComponentType = assetPayload ? resolveAssetComponentType(assetPayload) : "";

  /**
   * 将资源拖拽 payload 写入新建节点属性。
   * @param {ReturnType<typeof editorStore.insertNode>} insertedNode - 新建节点
   * @param {string} insertedType - 新建节点类型
   */
  const applyAssetPayloadToInsertedNode = (
    insertedNode: ReturnType<typeof editorStore.insertNode>,
    insertedType: string,
  ): void => {
    if (!insertedNode || !assetPayload) return;
    const componentType =
      insertedType === "Image" ? "Image" : insertedType === "Video" ? "Video" : "DownloadLink";
    const patchProps = buildAssetNodeProps(assetPayload, componentType);
    if (!patchProps || Object.keys(patchProps).length === 0) return;
    editorStore.updateNode(insertedNode.id, {
      props: { ...(insertedNode.props || {}), ...patchProps },
    });
  };

  let componentType = "";
  if (payload) {
    try {
      const parsed = JSON.parse(payload);
      componentType = parsed.type || "";
    } catch {
      componentType = payload;
    }
  }

  if (!componentType) {
    componentType = fallbackType;
  }
  if (!componentType) {
    componentType = assetComponentType;
  }
  if (!componentType) return;

  const canvasEl = event.currentTarget;
  if (!(canvasEl instanceof HTMLElement)) return;

  const zoomValue = Number(canvasZoom?.value) || 1;

  // 使用 placementResolver 统一解析放置目标
  const resolution = resolvePlacement(
    event,
    canvasEl,
    doc.value!,
    { rootNodeId: currentPage.value.rootNodeId },
    zoomValue,
  );

  const { parentId, index, dropPosition, containerType } = resolution;

  // 特殊处理 ElLayout 容器的自动插入行/列逻辑（保留原有规则）
  const insertIntoElLayout = (layoutId: string): boolean => {
    const layoutNode = doc.value?.getNode?.(layoutId);
    if (!layoutNode || layoutNode.type !== "ElLayout") return false;
    const rowCount = (layoutNode.children || []).filter((childId: string) => {
      const childNode = doc.value?.getNode?.(childId);
      return childNode?.type === "ElLayoutRow";
    }).length;
    const rowNode = editorStore.insertNode("ElLayoutRow", layoutId, rowCount, {
      autoSelectInserted: false,
    });
    if (!rowNode) return false;

    const latestLayout = doc.value?.getNode?.(layoutId);
    const nextRows = Math.max(1, rowCount + 1);
    const prevRows = latestLayout?.props?.rows as number | undefined;
    if ((prevRows || 0) !== nextRows) {
      editorStore.updateNode(layoutId, {
        props: {
          ...(latestLayout?.props || layoutNode.props || {}),
          rows: nextRows,
        },
      });
    }

    editorStore.updateNode(rowNode.id, {
      props: { ...(rowNode.props || {}), columns: 1 },
    });
    const latestRow = doc.value?.getNode?.(rowNode.id);
    const colIds = (latestRow?.children || []).filter((childId: string) => {
      const childNode = doc.value?.getNode?.(childId);
      return childNode?.type === "ElCol";
    });
    let colId = colIds[0];
    if (!colId) {
      const colNode = editorStore.insertNode("ElCol", rowNode.id, 0, {
        autoSelectInserted: false,
      });
      if (!colNode) return false;
      colId = colNode.id;
    }
    const inserted = editorStore.insertNode(componentType, colId, undefined, {
      autoSelectInserted: false,
    });
    applyAssetPayloadToInsertedNode(inserted, componentType);
    return Boolean(inserted);
  };

  let didInsert = false;

  // ElLayout 需要特殊处理：自动创建行/列
  if (containerType === "flex" && parentId) {
    const parentNode = doc.value?.getNode?.(parentId);
    if (parentNode?.type === "ElLayout") {
      didInsert = insertIntoElLayout(parentId);
    }
  }

  // 标准放置逻辑：使用 placementResolver 返回的 parentId、index、dropPosition
  if (!didInsert && parentId) {
    // 校验 insertIndex 有效性（D-05: append fallback）
    const parentNode = doc.value?.getNode?.(parentId);
    const siblings = parentNode?.children || [];
    const validIndex =
      index < 0 || index > siblings.length || !siblings[index] ? siblings.length : index;

    // 构建插入参数
    const insertOptions: {
      dropPosition?: { x: number; y: number };
      autoSelectInserted: false;
    } = { autoSelectInserted: false };
    if (dropPosition) {
      insertOptions.dropPosition = {
        x: Math.max(0, Math.round(dropPosition.x)),
        y: Math.max(0, Math.round(dropPosition.y)),
      };
    }

    const inserted = editorStore.insertNode(componentType, parentId, validIndex, insertOptions);
    applyAssetPayloadToInsertedNode(inserted, componentType);
    didInsert = Boolean(inserted);
  }

  endDrag();
  if (didInsert) {
    event.stopPropagation();
    return;
  }
  const message = String(error.value ?? "插入失败：页面未就绪或处于只读状态");
  ElMessage.warning(message as never);
}

/**
 * 显示右键菜单（从 NodeRenderer 触发）
 * @param {MouseEvent} event - 鼠标事件
 * @param {boolean} forceShow - 是否强制显示（组件已在 NodeRenderer 中被选中）
 */
function showContextMenu(event: MouseEvent, forceShow = true): void {
  // 如果菜单已经显示，再次右键则关闭
  if (contextMenuVisible.value) {
    closeContextMenu();
    return;
  }

  // 只有选中组件时才显示右键菜单
  // forceShow 为 true 时表示组件已由 NodeRenderer 选中
  if (!forceShow && !hasSelection.value) {
    return;
  }

  contextMenuX.value = event.clientX;
  contextMenuY.value = event.clientY;
  contextMenuVisible.value = true;
}

/**
 * 关闭右键菜单
 */
function closeContextMenu(): void {
  contextMenuVisible.value = false;
}

provide("showContextMenu", showContextMenu);
/**
 * 处理菜单项点击
 */
function handleContextMenuClick(): void {
  closeContextMenu();
}

/**
 * 删除选中节点
 */
function handleDelete() {
  editorStore.removeSelectedNodes();
  closeContextMenu();
}

/**
 * 上移一层
 */
function handleMoveUp() {
  editorStore.moveNodeUp();
  closeContextMenu();
}

/**
 * 下移一层
 */
function handleMoveDown() {
  editorStore.moveNodeDown();
  closeContextMenu();
}

/**
 * 置于顶层
 */
function handleMoveToTop() {
  editorStore.moveNodeToTop();
  closeContextMenu();
}

/**
 * 置于底层
 */
function handleMoveToBottom() {
  editorStore.moveNodeToBottom();
  closeContextMenu();
}

/**
 * 在选中列左侧插入一列
 */
function handleInsertColLeft() {
  editorStore.insertElColLeft();
  closeContextMenu();
}

/**
 * 在选中列右侧插入一列
 */
function handleInsertColRight() {
  editorStore.insertElColRight();
  closeContextMenu();
}

/**
 * 在选中行上方插入一行
 */
function handleInsertRowUp() {
  editorStore.insertElLayoutRowUp();
  closeContextMenu();
}

/**
 * 在选中行下方插入一行
 */
function handleInsertRowDown() {
  editorStore.insertElLayoutRowDown();
  closeContextMenu();
}

/**
 * 撤销
 */
function handleUndo() {
  if (canUndo.value) {
    editorStore.undo();
  }
  closeContextMenu();
}

/**
 * 重做
 */
function handleRedo() {
  if (canRedo.value) {
    editorStore.redo();
  }
  closeContextMenu();
}

function handleClickOutside(_event: MouseEvent): void {
  if (contextMenuVisible.value) {
    closeContextMenu();
  }
}

/**
 * 判断键盘事件是否来自可编辑输入上下文，避免误删输入内容
 * @param {EventTarget | null} target - 事件目标
 * @returns {boolean} 是否可编辑上下文
 */
function isEditableInputTarget(target: EventTarget | null): boolean {
  if (!(target instanceof HTMLElement)) return false;
  if (target.isContentEditable) return true;
  return Boolean(target.closest('input, textarea, select, [contenteditable="true"]'));
}

/**
 * 全局键盘兜底监听，保证画布失焦时快捷键仍可用
 * @param {KeyboardEvent} event - 键盘事件
 */
function handleGlobalKeyDown(event: KeyboardEvent): void {
  if (isEditableInputTarget(event.target)) return;
  if (event.key === "Delete" || event.key === "Backspace") {
    const selectedElements = selection.value?.getSelectedElements?.() || [];
    const hasSelectedNode = selectedElements.some((item) => item.kind === "node");
    if (!hasSelectedNode) return;
    event.preventDefault();
    event.stopPropagation();
    editorStore.removeSelectedNodes();
    return;
  }
  if (event.defaultPrevented) return;
  handleKeyDown(event);
}

onMounted(() => {
  document.addEventListener("click", handleClickOutside);
  document.addEventListener("click", handleDocumentClickCapture, true);
  window.addEventListener(
    CANVAS_OUTSIDE_MARQUEE_START_EVENT,
    handleOutsideMarqueeStart as EventListener,
  );
  window.addEventListener("designer:force-refresh", handleForceRefresh);
  window.addEventListener("keydown", handleGlobalKeyDown, true);
});

onBeforeUnmount(() => {
  document.removeEventListener("click", handleClickOutside);
  document.removeEventListener("click", handleDocumentClickCapture, true);
  window.removeEventListener(
    CANVAS_OUTSIDE_MARQUEE_START_EVENT,
    handleOutsideMarqueeStart as EventListener,
  );
  window.removeEventListener("designer:force-refresh", handleForceRefresh);
  window.removeEventListener("keydown", handleGlobalKeyDown, true);
  document.removeEventListener("pointermove", handleMarqueeMove);
  document.removeEventListener("pointerup", handleMarqueeUp);
});

/**
 * 处理键盘事件
 * @param {KeyboardEvent} event - 键盘事件
 */
function handleKeyDown(event: KeyboardEvent): void {
  // 方向键移动选中节点，Shift 微调 1px
  if (["ArrowUp", "ArrowDown", "ArrowLeft", "ArrowRight"].includes(event.key)) {
    event.preventDefault();
    const step = event.shiftKey ? 1 : 10;
    const dx = event.key === "ArrowLeft" ? -step : event.key === "ArrowRight" ? step : 0;
    const dy = event.key === "ArrowUp" ? -step : event.key === "ArrowDown" ? step : 0;
    editorStore.moveSelectedByDelta(dx, dy);
    return;
  }

  // Delete / Backspace 删除选中节点
  if (event.key === "Delete" || event.key === "Backspace") {
    event.preventDefault();
    editorStore.removeSelectedNodes();
    return;
  }

  // Ctrl+Z / Cmd+Z 撤销
  if ((event.ctrlKey || event.metaKey) && event.key === "z" && !event.shiftKey) {
    event.preventDefault();
    editorStore.undo();
    return;
  }

  // Ctrl+Shift+Z / Cmd+Shift+Z 重做
  if ((event.ctrlKey || event.metaKey) && event.key === "z" && event.shiftKey) {
    event.preventDefault();
    editorStore.redo();
    return;
  }

  // Ctrl+Y / Cmd+Y 重做
  if ((event.ctrlKey || event.metaKey) && event.key === "y") {
    event.preventDefault();
    editorStore.redo();
    return;
  }

  // Ctrl+] 上移图层
  if ((event.ctrlKey || event.metaKey) && event.key === "]" && !event.shiftKey) {
    event.preventDefault();
    editorStore.moveNodeUp();
    return;
  }

  // Ctrl+[ 下移图层
  if ((event.ctrlKey || event.metaKey) && event.key === "[" && !event.shiftKey) {
    event.preventDefault();
    editorStore.moveNodeDown();
    return;
  }

  // Ctrl+Shift+] 置顶
  if ((event.ctrlKey || event.metaKey) && event.key === "]" && event.shiftKey) {
    event.preventDefault();
    editorStore.moveNodeToTop();
    return;
  }

  // Ctrl+Shift+[ 置底
  if ((event.ctrlKey || event.metaKey) && event.key === "[" && event.shiftKey) {
    event.preventDefault();
    editorStore.moveNodeToBottom();
    return;
  }

  // Ctrl+H 切换显示/隐藏
  if ((event.ctrlKey || event.metaKey) && event.key === "h") {
    event.preventDefault();
    const primary = selection.value?.getPrimaryElement();
    if (primary && primary.kind === "node") {
      editorStore.toggleNodeVisibility(primary.id);
    }
    return;
  }

  // Ctrl+L 切换锁定/解锁
  if ((event.ctrlKey || event.metaKey) && event.key === "l") {
    event.preventDefault();
    const primary = selection.value?.getPrimaryElement();
    if (primary && primary.kind === "node") {
      editorStore.toggleNodeLock(primary.id);
    }
    return;
  }

  // Ctrl+C 复制
  if ((event.ctrlKey || event.metaKey) && event.key === "c" && !event.shiftKey) {
    event.preventDefault();
    editorStore.copyNodes();
    return;
  }

  // Ctrl+V 粘贴（粘贴到鼠标位置）
  if ((event.ctrlKey || event.metaKey) && event.key === "v" && !event.shiftKey) {
    event.preventDefault();
    editorStore.pasteNodes(canvasMousePos.value ?? undefined);
    return;
  }

  // Ctrl+D 复制元素
  if ((event.ctrlKey || event.metaKey) && event.key === "d") {
    event.preventDefault();
    editorStore.duplicateNodes();
    return;
  }

  // Escape 清除选中
  if (event.key === "Escape") {
    selection.value?.clearSelection();
  }
}
</script>

<template>
  <div
    ref="designCanvasRef"
    class="design-canvas"
    tabindex="0"
    @pointerdown.capture="handleCanvasPointerDown"
    @pointermove="handleCanvasPointerMove"
    @pointerleave="handleCanvasPointerLeave"
    @keydown="handleKeyDown"
    @dragover.prevent
    @drop.prevent="handleCanvasDrop"
  >
    <NodeRenderer v-if="rootNodeId" :node-id="rootNodeId" :is-root="true" />
    <div v-if="!hasContent" class="empty-placeholder">
      <IconEpPlus class="text-5xl mb-4" />
      <p>从左侧拖拽组件到画布</p>
    </div>

    <!-- 右键菜单 -->
    <teleport to="body">
      <div
        v-if="contextMenuVisible"
        class="context-menu"
        :style="{ left: `${contextMenuX}px`, top: `${contextMenuY}px` }"
        @click="handleContextMenuClick"
      >
        <div v-if="hasSelection" class="menu-item" @click="handleDelete">
          <IconEpDelete />
          <span>鍒犻櫎</span>
          <span class="shortcut">Del</span>
        </div>
        <div v-if="hasSelection" class="menu-item" @click="handleMoveUp">
          <IconEpTop />
          <span>上移一层</span>
          <span class="shortcut">Ctrl+]</span>
        </div>
        <div v-if="hasSelection" class="menu-item" @click="handleMoveDown">
          <IconEpBottom />
          <span>下移一层</span>
          <span class="shortcut">Ctrl+[</span>
        </div>
        <div v-if="hasSelection" class="menu-item" @click="handleMoveToTop">
          <IconEpTop />
          <span>缃簬椤跺眰</span>
          <span class="shortcut">Ctrl+Shift+]</span>
        </div>
        <div v-if="hasSelection" class="menu-item" @click="handleMoveToBottom">
          <IconEpBottom />
          <span>缃簬搴曞眰</span>
          <span class="shortcut">Ctrl+Shift+[</span>
        </div>
        <div v-if="hasSelection" class="menu-divider"></div>
        <div v-if="isElColSelected" class="menu-item" @click="handleInsertColLeft">
          <IconEpPlus />
          <span>左侧新增一列</span>
        </div>
        <div v-if="isElColSelected" class="menu-item" @click="handleInsertColRight">
          <IconEpPlus />
          <span>右侧新增一列</span>
        </div>
        <div v-if="isElColSelected" class="menu-divider"></div>
        <div v-if="isElLayoutRowSelected" class="menu-item" @click="handleInsertRowUp">
          <IconEpPlus />
          <span>上方新增一行</span>
        </div>
        <div v-if="isElLayoutRowSelected" class="menu-item" @click="handleInsertRowDown">
          <IconEpPlus />
          <span>下方新增一行</span>
        </div>
        <div v-if="isElLayoutRowSelected" class="menu-divider"></div>
        <div class="menu-item" :class="{ disabled: !canUndo }" @click="handleUndo">
          <IconEpRefreshLeft />
          <span>鎾ら攢</span>
          <span class="shortcut">Ctrl+Z</span>
        </div>
        <div class="menu-item" :class="{ disabled: !canRedo }" @click="handleRedo">
          <IconEpRefreshRight />
          <span>閲嶅仛</span>
          <span class="shortcut">Ctrl+Y</span>
        </div>
      </div>
    </teleport>

    <teleport to="body">
      <div v-if="marquee.active" class="marquee-selection" :style="marqueeStyle" />
    </teleport>
  </div>
</template>

<style scoped>
.design-canvas {
  position: relative;
  width: 100%;
  height: 100%;
  outline: none;
}

.design-canvas:focus {
  outline: none;
}

.empty-placeholder {
  position: absolute;
  inset: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  color: #9ca3af;
  pointer-events: none;
  transition: all 0.2s ease;
}

.empty-placeholder.drag-over {
  background-color: rgba(59, 130, 246, 0.1);
  border: 2px dashed #3b82f6;
  color: #3b82f6;
}

/* 右键菜单样式 */
.context-menu {
  position: fixed;
  background: white;
  border: 1px solid #e4e7ed;
  border-radius: 6px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
  padding: 4px 0;
  min-width: 180px;
  z-index: 9999;
  user-select: none;
}

.dark .context-menu {
  background: #1a1a1a;
  border-color: #3a3a3a;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.5);
}

.menu-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px;
  font-size: 13px;
  color: #303133;
  cursor: pointer;
  transition: background-color 0.2s;
}

.dark .menu-item {
  color: #e4e7ed;
}

.menu-item:hover:not(.disabled) {
  background-color: #f5f7fa;
}

.dark .menu-item:hover:not(.disabled) {
  background-color: #2a2a2a;
}

.menu-item.disabled {
  color: #c0c4cc;
  cursor: not-allowed;
  opacity: 0.5;
}

.menu-item svg {
  width: 14px;
  height: 14px;
  flex-shrink: 0;
}

.menu-item span:first-of-type {
  flex: 1;
}

.menu-item .shortcut {
  font-size: 11px;
  color: #909399;
  margin-left: auto;
}

.dark .menu-item .shortcut {
  color: #606266;
}

.menu-divider {
  height: 1px;
  background-color: #e4e7ed;
  margin: 4px 0;
}

.dark .menu-divider {
  background-color: #3a3a3a;
}

.marquee-selection {
  position: fixed;
  border: 1px solid rgba(59, 130, 246, 0.9);
  background: rgba(59, 130, 246, 0.16);
  pointer-events: none;
  z-index: 9998;
}
</style>
