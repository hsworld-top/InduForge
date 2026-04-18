<script setup lang="ts">
import { ElMessage } from "element-plus";
import { storeToRefs } from "pinia";
/**
 * DesignCanvas - 璁捐鐢诲竷缁勪欢
 *
 * 鑱岃矗锛?
 * - 娓叉煋褰撳墠椤甸潰鐨勭粍浠舵爲锛圢odeRenderer锛?
 * - 妗嗛€夛紙marquee锛夊閫夈€佸彸閿彍鍗曘€佸揩鎹烽敭
 * - 鎷栨嫿鏀剧疆缁勪欢銆佺敾甯冨唴鎷栨嫿鎺掑簭
 * - 绮樿创鍒伴紶鏍囦綅缃€佹挙閿€/閲嶅仛
 * - 绌虹敾甯冨崰浣嶆彁绀?
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
import { eventToCanvasPosition } from "@/editor-core/utils/placement-utils";
import { resolvePlacement } from "@/editor-core/utils/placement-resolver";
import { useEditorStore } from "@/stores/editor-store";
import {
  DESIGNER_ASSET_DRAG_MIME,
  buildAssetNodeProps,
  parseAssetDragPayload,
  resolveAssetComponentType,
} from "@/ui/shared/utils/asset-drag";
import { endDrag, useDragState } from "./composables/use-drag-state";
import { canvasZoomKey } from "./injection-keys";
import { createMarqueeClickGuard } from "./services/marquee-click-guard";
import {
  CANVAS_OUTSIDE_MARQUEE_START_EVENT,
  shouldClearSelectionOnMarqueeUp,
  shouldStartMarqueeFromCanvasPointerDown,
  type MarqueeModifiers,
  type MarqueeStartSource,
  type OutsideMarqueeStartDetail,
} from "./services/marquee-interaction";
import { collectMarqueeNodeIds } from "./services/marquee-selection";
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

/** 褰撳墠椤甸潰鏍硅妭鐐?ID */
const rootNodeId = computed(() => currentPage.value?.rootNodeId || "");
/** 妗嗛€夌姸鎬侊細鏄惁婵€娲汇€佹槸鍚︾Щ鍔ㄣ€佽捣姝㈠潗鏍囥€佷慨楗伴敭 */
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

/** 寮哄埗鍒锋柊鐢诲竷锛堣Е鍙?docVersion 鍙樻洿锛?*/
function handleForceRefresh(): void {
  docVersion.value += 1;
}

/** 閲嶇疆妗嗛€夌姸鎬?*/
function resetMarquee(): void {
  marquee.value.active = false;
  marquee.value.moved = false;
  marquee.value.startSource = "insideCanvas";
}

/**
 * 鍚姩妗嗛€夈€?
 * @param {{
 *   clientX: number;
 *   clientY: number;
 *   modifiers: MarqueeModifiers;
 *   startSource: MarqueeStartSource;
 * }} payload - 妗嗛€夎捣鐐瑰弬鏁?
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

/** 鑾峰彇妗嗛€夌煩褰㈢殑 left/top/right/bottom/width/height */
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

/** 鏀堕泦妗嗛€夊尯鍩熷唴鐩镐氦鐨勮妭鐐癸紙鏀寔鍗曞鍣ㄧ┛閫忥級 */
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

/** 妗嗛€夎繃绋嬩腑鏇存柊褰撳墠鍧愭爣 */
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

/** 妗嗛€夌粨鏉燂細搴旂敤閫変腑缁撴灉鎴栫偣鍑婚€変腑鏍硅妭鐐?*/
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
    // 妗嗛€夐噴鏀惧悗娴忚鍣ㄩ€氬父浼氬啀娲惧彂涓€娆?click锛岄渶瑕佸悶鎺夐伩鍏嶈鐩栧閫夌粨鏋溿€?
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
 * 鍏ㄥ眬 click 鎹曡幏锛氭秷璐规閫夊悗鐨勨€滆ˉ鍙?click鈥濓紝閬垮厤鎶婃閫夌粨鏋滄敼鍐欐垚鍗曢€?娓呯┖銆?
 * @param {MouseEvent} event - 榧犳爣浜嬩欢
 */
function handleDocumentClickCapture(event: MouseEvent): void {
  if (!marqueeClickGuard.consumeShouldSuppressNextClick()) return;
  event.preventDefault();
  event.stopPropagation();
}

/**
 * 鍏滃簳澶勭悊鐢诲竷鐐瑰嚮閫変腑锛岄伩鍏嶇粍浠跺唴閮ㄩ樆姝㈠啋娉″鑷存棤娉曢€変腑
 * @param {PointerEvent | MouseEvent} event - 榧犳爣浜嬩欢
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
  // 鍚姩妗嗛€夊悗闃绘柇浜嬩欢涓嬪彂鍒拌妭鐐瑰眰锛岄伩鍏嶈妭鐐规嫋鎷介€昏緫鎶㈠崰鍚屼竴娆?pointer 搴忓垪
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
 * 澶勭悊宸ヤ綔鍙扮伆鍖鸿Е鍙戠殑妗嗛€夊紑濮嬩簨浠躲€?
 * @param {Event} event - 鑷畾涔変簨浠?
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
 * 鏇存柊鐢诲竷涓婄殑榧犳爣鍧愭爣锛屼互鍙婃偓鍋滆妭鐐圭被鍨?
 * @param {PointerEvent} event - 鎸囬拡浜嬩欢
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
  // 妫€娴嬫偓鍋滆妭鐐圭被鍨嬶紙渚涘簳閮ㄧ姸鎬佹爮灞曠ず锛?
  const hitEl = document.elementFromPoint(event.clientX, event.clientY);
  const nodeEl = hitEl?.closest?.("[data-node-type]");
  hoveredNodeType.value = nodeEl?.getAttribute?.("data-node-type") || "";
}

/**
 * 榧犳爣绂诲紑鐢诲竷鏃舵竻绌哄潗鏍囦笌鎮仠淇℃伅
 */
function handleCanvasPointerLeave(): void {
  canvasMousePos.value = null;
  hoveredNodeType.value = "";
}

/** 鍙抽敭鑿滃崟鏄剧ず鐘舵€佷笌鍧愭爣 */
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
 * 澶勭悊鐢诲竷绌虹櫧澶勬斁缃粍浠?
 * @param {DragEvent} event - 鎷栨嫿浜嬩欢
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
   * 灏嗚祫婧愭嫋鎷?payload 鍐欏叆鏂板缓鑺傜偣灞炴€с€?
   * @param {ReturnType<typeof editorStore.insertNode>} insertedNode - 鏂板缓鑺傜偣
   * @param {string} insertedType - 鏂板缓鑺傜偣绫诲瀷
   */
  const applyAssetPayloadToInsertedNode = (
    insertedNode: ReturnType<typeof editorStore.insertNode>,
    insertedType: string,
  ): void => {
    if (!insertedNode || !assetPayload) return;
    const componentType =
      insertedType === "Image"
        ? "Image"
        : insertedType === "Video"
          ? "Video"
          : "DownloadLink";
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

  // 浣跨敤 placementResolver 缁熶竴澶勭悊鏀剧疆瑙ｆ瀽
  const resolution = resolvePlacement(
    event,
    canvasEl,
    doc.value!,
    { rootNodeId: currentPage.value.rootNodeId },
    zoomValue,
  );

  const { parentId, index, dropPosition, containerType } = resolution;

  // 鐗规畩澶勭悊 ElLayout 瀹瑰櫒鐨勮嚜鍔ㄦ彃鍏ヨ/鍒楅€昏緫锛堜繚鐣欏師鏈変笟鍔¤鍒欙級
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

  // ElLayout 闇€瑕佺壒娈婂鐞嗭細鑷姩鍒涘缓琛?鍒?
  if (containerType === "flex" && parentId) {
    const parentNode = doc.value?.getNode?.(parentId);
    if (parentNode?.type === "ElLayout") {
      didInsert = insertIntoElLayout(parentId);
    }
  }

  // 鏍囧噯鏀剧疆閫昏緫锛氫娇鐢?placementResolver 杩斿洖鐨?parentId銆乮ndex銆乨ropPosition
  if (!didInsert && parentId) {
    // 楠岃瘉 insertIndex 鏈夋晥鎬э紙D-05: append fallback锛?
    const parentNode = doc.value?.getNode?.(parentId);
    const siblings = parentNode?.children || [];
    const validIndex = index < 0 || index > siblings.length || !siblings[index]
      ? siblings.length
      : index;

    // 鏋勫缓鎻掑叆閫夐」
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
 * 鏄剧ず鍙抽敭鑿滃崟锛堜粠 NodeRenderer 瑙﹀彂锛?
 * @param {MouseEvent} event - 榧犳爣浜嬩欢
 * @param {boolean} forceShow - 鏄惁寮哄埗鏄剧ず锛堢粍浠跺凡鍦?NodeRenderer 涓閫変腑锛?
 */
function showContextMenu(event: MouseEvent, forceShow = true): void {
  // 濡傛灉鑿滃崟宸茬粡鏄剧ず锛屽啀娆″彸閿垯鍏抽棴
  if (contextMenuVisible.value) {
    closeContextMenu();
    return;
  }

  // 鍙湁閫変腑缁勪欢鏃舵墠鏄剧ず鍙抽敭鑿滃崟
  // forceShow 涓?true 鏃惰〃绀虹粍浠跺凡浠?NodeRenderer 閫変腑
  if (!forceShow && !hasSelection.value) {
    return;
  }

  contextMenuX.value = event.clientX;
  contextMenuY.value = event.clientY;
  contextMenuVisible.value = true;
}

/**
 * 鍏抽棴鍙抽敭鑿滃崟
 */
function closeContextMenu(): void {
  contextMenuVisible.value = false;
}

provide("showContextMenu", showContextMenu);
/**
 * 澶勭悊鑿滃崟椤圭偣鍑?
 */
function handleContextMenuClick(): void {
  closeContextMenu();
}

/**
 * 鍒犻櫎閫変腑鑺傜偣
 */
function handleDelete() {
  editorStore.removeSelectedNodes();
  closeContextMenu();
}

/**
 * 涓婄Щ涓€灞?
 */
function handleMoveUp() {
  editorStore.moveNodeUp();
  closeContextMenu();
}

/**
 * 涓嬬Щ涓€灞?
 */
function handleMoveDown() {
  editorStore.moveNodeDown();
  closeContextMenu();
}

/**
 * 缃簬椤跺眰
 */
function handleMoveToTop() {
  editorStore.moveNodeToTop();
  closeContextMenu();
}

/**
 * 缃簬搴曞眰
 */
function handleMoveToBottom() {
  editorStore.moveNodeToBottom();
  closeContextMenu();
}

/**
 * 鍦ㄩ€変腑鍒楀乏渚ф彃鍏ヤ竴鍒?
 */
function handleInsertColLeft() {
  editorStore.insertElColLeft();
  closeContextMenu();
}

/**
 * 鍦ㄩ€変腑鍒楀彸渚ф彃鍏ヤ竴鍒?
 */
function handleInsertColRight() {
  editorStore.insertElColRight();
  closeContextMenu();
}

/**
 * 鍦ㄩ€変腑琛屼笂鏂规彃鍏ヤ竴琛?
 */
function handleInsertRowUp() {
  editorStore.insertElLayoutRowUp();
  closeContextMenu();
}

/**
 * 鍦ㄩ€変腑琛屼笅鏂规彃鍏ヤ竴琛?
 */
function handleInsertRowDown() {
  editorStore.insertElLayoutRowDown();
  closeContextMenu();
}

/**
 * 鎾ら攢
 */
function handleUndo() {
  if (canUndo.value) {
    editorStore.undo();
  }
  closeContextMenu();
}

/**
 * 閲嶅仛
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
 * 鍒ゆ柇閿洏浜嬩欢鏄惁鏉ヨ嚜鍙紪杈戣緭鍏ヤ笂涓嬫枃锛岄伩鍏嶈鍒犺緭鍏ュ唴瀹?
 * @param {EventTarget | null} target - 浜嬩欢鐩爣
 * @returns {boolean} 鏄惁鍙紪杈戜笂涓嬫枃
 */
function isEditableInputTarget(target: EventTarget | null): boolean {
  if (!(target instanceof HTMLElement)) return false;
  if (target.isContentEditable) return true;
  return Boolean(target.closest('input, textarea, select, [contenteditable="true"]'));
}

/**
 * 鍏ㄥ眬閿洏鍏滃簳鐩戝惉锛屼繚璇佺敾甯冨け鐒︽椂蹇嵎閿粛鍙敤
 * @param {KeyboardEvent} event - 閿洏浜嬩欢
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
 * 澶勭悊閿洏浜嬩欢
 * @param {KeyboardEvent} event - 閿洏浜嬩欢
 */
function handleKeyDown(event: KeyboardEvent): void {
  // 鏂瑰悜閿Щ鍔ㄩ€変腑鑺傜偣锛孲hift 寰皟 1px
  if (["ArrowUp", "ArrowDown", "ArrowLeft", "ArrowRight"].includes(event.key)) {
    event.preventDefault();
    const step = event.shiftKey ? 1 : 10;
    const dx = event.key === "ArrowLeft" ? -step : event.key === "ArrowRight" ? step : 0;
    const dy = event.key === "ArrowUp" ? -step : event.key === "ArrowDown" ? step : 0;
    editorStore.moveSelectedByDelta(dx, dy);
    return;
  }

  // Delete / Backspace 鍒犻櫎閫変腑鑺傜偣
  if (event.key === "Delete" || event.key === "Backspace") {
    event.preventDefault();
    editorStore.removeSelectedNodes();
    return;
  }

  // Ctrl+Z / Cmd+Z 鎾ら攢
  if ((event.ctrlKey || event.metaKey) && event.key === "z" && !event.shiftKey) {
    event.preventDefault();
    editorStore.undo();
    return;
  }

  // Ctrl+Shift+Z / Cmd+Shift+Z 閲嶅仛
  if ((event.ctrlKey || event.metaKey) && event.key === "z" && event.shiftKey) {
    event.preventDefault();
    editorStore.redo();
    return;
  }

  // Ctrl+Y / Cmd+Y 閲嶅仛
  if ((event.ctrlKey || event.metaKey) && event.key === "y") {
    event.preventDefault();
    editorStore.redo();
    return;
  }

  // Ctrl+] 涓婄Щ鍥惧眰
  if ((event.ctrlKey || event.metaKey) && event.key === "]" && !event.shiftKey) {
    event.preventDefault();
    editorStore.moveNodeUp();
    return;
  }

  // Ctrl+[ 涓嬬Щ鍥惧眰
  if ((event.ctrlKey || event.metaKey) && event.key === "[" && !event.shiftKey) {
    event.preventDefault();
    editorStore.moveNodeDown();
    return;
  }

  // Ctrl+Shift+] 缃《
  if ((event.ctrlKey || event.metaKey) && event.key === "]" && event.shiftKey) {
    event.preventDefault();
    editorStore.moveNodeToTop();
    return;
  }

  // Ctrl+Shift+[ 缃簳
  if ((event.ctrlKey || event.metaKey) && event.key === "[" && event.shiftKey) {
    event.preventDefault();
    editorStore.moveNodeToBottom();
    return;
  }

  // Ctrl+H 鍒囨崲鏄剧ず/闅愯棌
  if ((event.ctrlKey || event.metaKey) && event.key === "h") {
    event.preventDefault();
    const primary = selection.value?.getPrimaryElement();
    if (primary && primary.kind === "node") {
      editorStore.toggleNodeVisibility(primary.id);
    }
    return;
  }

  // Ctrl+L 鍒囨崲閿佸畾/瑙ｉ攣
  if ((event.ctrlKey || event.metaKey) && event.key === "l") {
    event.preventDefault();
    const primary = selection.value?.getPrimaryElement();
    if (primary && primary.kind === "node") {
      editorStore.toggleNodeLock(primary.id);
    }
    return;
  }

  // Ctrl+C 澶嶅埗
  if ((event.ctrlKey || event.metaKey) && event.key === "c" && !event.shiftKey) {
    event.preventDefault();
    editorStore.copyNodes();
    return;
  }

  // Ctrl+V 绮樿创锛堢矘璐村埌榧犳爣浣嶇疆锛?
  if ((event.ctrlKey || event.metaKey) && event.key === "v" && !event.shiftKey) {
    event.preventDefault();
    editorStore.pasteNodes(canvasMousePos.value ?? undefined);
    return;
  }

  // Ctrl+D 澶嶅埗鍏冪礌
  if ((event.ctrlKey || event.metaKey) && event.key === "d") {
    event.preventDefault();
    editorStore.duplicateNodes();
    return;
  }

  // Escape 娓呴櫎閫変腑
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
      <p>浠庡乏渚ф嫋鎷界粍浠跺埌鐢诲竷</p>
    </div>

    <!-- 鍙抽敭鑿滃崟 -->
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

/* 鉁?鍙抽敭鑿滃崟鏍峰紡 */
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

