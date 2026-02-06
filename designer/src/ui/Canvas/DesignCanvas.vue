<template>
  <div
    class="design-canvas"
    tabindex="0"
    @pointerdown.capture="handleCanvasPointerDown"
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
        :style="{ left: contextMenuX + 'px', top: contextMenuY + 'px' }"
        @click="handleContextMenuClick"
      >
        <div v-if="hasSelection" class="menu-item" @click="handleDelete">
          <IconEpDelete />
          <span>删除</span>
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
          <span>置于顶层</span>
          <span class="shortcut">Ctrl+Shift+]</span>
        </div>
        <div v-if="hasSelection" class="menu-item" @click="handleMoveToBottom">
          <IconEpBottom />
          <span>置于底层</span>
          <span class="shortcut">Ctrl+Shift+[</span>
        </div>
        <div v-if="hasSelection" class="menu-divider"></div>
        <div
          v-if="isElColSelected"
          class="menu-item"
          @click="handleInsertColLeft"
        >
          <IconEpPlus />
          <span>左侧新加一列</span>
        </div>
        <div
          v-if="isElColSelected"
          class="menu-item"
          @click="handleInsertColRight"
        >
          <IconEpPlus />
          <span>右侧新加一列</span>
        </div>
        <div v-if="isElColSelected" class="menu-divider"></div>
        <div
          v-if="isElLayoutRowSelected"
          class="menu-item"
          @click="handleInsertRowUp"
        >
          <IconEpPlus />
          <span>上方新加一行</span>
        </div>
        <div
          v-if="isElLayoutRowSelected"
          class="menu-item"
          @click="handleInsertRowDown"
        >
          <IconEpPlus />
          <span>下方新加一行</span>
        </div>
        <div v-if="isElLayoutRowSelected" class="menu-divider"></div>
        <div
          class="menu-item"
          @click="handleUndo"
          :class="{ disabled: !canUndo }"
        >
          <IconEpRefreshLeft />
          <span>撤销</span>
          <span class="shortcut">Ctrl+Z</span>
        </div>
        <div
          class="menu-item"
          @click="handleRedo"
          :class="{ disabled: !canRedo }"
        >
          <IconEpRefreshRight />
          <span>重做</span>
          <span class="shortcut">Ctrl+Y</span>
        </div>
      </div>
    </teleport>
  </div>
</template>

<script setup>
import { computed, ref, onMounted, onBeforeUnmount, provide, inject } from "vue";
import { storeToRefs } from "pinia";
import { useEditorStore } from "@/stores/editor-store";
import { createSelectableElement } from "@/editor-core";
import { ElMessage } from "element-plus";
import NodeRenderer from "./NodeRenderer.vue";
import IconEpPlus from "~icons/ep/plus";
import IconEpDelete from "~icons/ep/delete";
import IconEpTop from "~icons/ep/top";
import IconEpBottom from "~icons/ep/bottom";
import IconEpRefreshLeft from "~icons/ep/refresh-left";
import IconEpRefreshRight from "~icons/ep/refresh-right";
import { useDragState, endDrag } from "./use-drag-state";

const editorStore = useEditorStore();
const { doc, currentPage, selection, history, docVersion, selectionVersion, error } =
  storeToRefs(editorStore);
const canvasZoom = inject("canvasZoom", ref(1));
const dragState = useDragState();

const rootNodeId = computed(() => currentPage.value?.rootNodeId || "");

const hasContent = computed(() => {
  docVersion.value;
  if (!doc.value || !currentPage.value) return false;
  const root = doc.value.getNode(currentPage.value.rootNodeId);
  return (root?.children || []).length > 0;
});

const handleForceRefresh = () => {
  docVersion.value += 1;
};

/**
 * 兜底处理画布点击选中，避免组件内部阻止冒泡导致无法选中
 * @param {PointerEvent | MouseEvent} event - 鼠标事件
 */
const handleCanvasPointerDown = (event) => {
  if (!selection.value) return;
  if (event.pointerType === "mouse" && event.button !== 0) return;
  if (!(event.target instanceof Element)) {
    closeContextMenu();
    selection.value?.clearSelection();
    return;
  }

  closeContextMenu();

  const nodeElement = event.target.closest(".designer-node");
  if (nodeElement && !nodeElement.classList.contains("is-root")) {
    return;
  }

  const rootId = rootNodeId.value;
  if (rootId) {
    const element = createSelectableElement("node", rootId);
    if (event.shiftKey) {
      selection.value.selectRange(element);
      return;
    }
    if (event.metaKey || event.ctrlKey) {
      selection.value.toggleSelect(element);
      return;
    }
    selection.value.select(element);
    return;
  }

  selection.value?.clearSelection();
};

// ✅ 右键菜单状态
const contextMenuVisible = ref(false);
const contextMenuX = ref(0);
const contextMenuY = ref(0);

const hasSelection = computed(() => {
  selectionVersion.value;
  return selection.value?.getSelectedElements?.().length > 0;
});

const isElColSelected = computed(() => {
  selectionVersion.value;
  const primary = selection.value?.getPrimarySelection?.();
  if (primary?.type === "ElCol") return true;
  const selectedNodes = selection.value?.getSelectedNodes?.() || [];
  return selectedNodes.some((node) => node?.type === "ElCol");
});
const isElLayoutRowSelected = computed(() => {
  selectionVersion.value;
  const primary = selection.value?.getPrimarySelection?.();
  if (primary?.type === "ElLayoutRow") return true;
  const selectedNodes = selection.value?.getSelectedNodes?.() || [];
  return selectedNodes.some((node) => node?.type === "ElLayoutRow");
});

const canUndo = computed(() => history.value?.canUndo?.() || false);
const canRedo = computed(() => history.value?.canRedo?.() || false);

/**
 * 处理画布空白处放置组件
 * @param {DragEvent} event - 拖拽事件
 */
const handleCanvasDrop = (event) => {
  if (!currentPage.value?.rootNodeId) return;

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

  const rect = event.currentTarget.getBoundingClientRect();
  const zoomValue = Number(canvasZoom?.value) || 1;
  const offsetX = (event.clientX - rect.left) / zoomValue;
  const offsetY = (event.clientY - rect.top) / zoomValue;

  const insertIntoElLayout = (layoutId) => {
    const layoutNode = doc.value?.getNode?.(layoutId);
    if (!layoutNode || layoutNode.type !== "ElLayout") return false;
    const rowCount = (layoutNode.children || []).filter((childId) => {
      const childNode = doc.value?.getNode?.(childId);
      return childNode?.type === "ElLayoutRow";
    }).length;
    const rowNode = editorStore.insertNode("ElLayoutRow", layoutId, rowCount);
    if (!rowNode) return false;

    const latestLayout = doc.value?.getNode?.(layoutId);
    const nextRows = Math.max(1, rowCount + 1);
    if ((latestLayout?.props?.rows || 0) !== nextRows) {
      editorStore.updateNode(layoutId, {
        props: { ...(latestLayout?.props || layoutNode.props || {}), rows: nextRows },
      });
    }

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
      if (!colNode) return false;
      colId = colNode.id;
    }
    const inserted = editorStore.insertNode(componentType, colId);
    return Boolean(inserted);
  };

  const resolveLayoutFromPoint = () => {
    const layoutElements = Array.from(
      document.querySelectorAll('[data-node-type="ElLayout"][data-node-id]')
    );
    if (!layoutElements.length) return null;
    const pointX = event.clientX;
    const pointY = event.clientY;
    let candidate = null;
    let minDistance = Number.POSITIVE_INFINITY;
    for (const element of layoutElements) {
      const rect = element.getBoundingClientRect();
      const withinX = pointX >= rect.left && pointX <= rect.right;
      const withinY = pointY >= rect.top && pointY <= rect.bottom + 24;
      if (!withinX || !withinY) continue;
      const distance = Math.max(0, pointY - rect.bottom);
      if (distance < minDistance) {
        minDistance = distance;
        candidate = element;
      }
    }
    if (!candidate) return null;
    return candidate.getAttribute("data-node-id");
  };

  let inserted = editorStore.insertNode(
    componentType,
    currentPage.value.rootNodeId,
    undefined,
    {
      dropPosition: {
        x: Math.max(0, Math.round(offsetX)),
        y: Math.max(0, Math.round(offsetY)),
      },
    }
  );
  if (!inserted) {
    const layoutId = resolveLayoutFromPoint();
    if (layoutId) {
      inserted = insertIntoElLayout(layoutId);
    }
  }
  endDrag();
  if (inserted) {
    event.stopPropagation();
    return;
  }
  const message = error.value || "插入失败：页面未就绪或处于只读状态";
  ElMessage.warning(message);
};

/**
 * 显示右键菜单（从 NodeRenderer 触发）
 * @param {MouseEvent} event - 鼠标事件
 * @param {boolean} forceShow - 是否强制显示（组件已在 NodeRenderer 中被选中）
 */
const showContextMenu = (event, forceShow = true) => {
  // ✅ 如果菜单已经显示，再次右键则关闭
  if (contextMenuVisible.value) {
    closeContextMenu();
    return;
  }

  // 只有选中组件时才显示右键菜单
  // forceShow 为 true 时表示组件已从 NodeRenderer 选中
  if (!forceShow && !hasSelection.value) {
    return;
  }

  contextMenuX.value = event.clientX;
  contextMenuY.value = event.clientY;
  contextMenuVisible.value = true;
};

/**
 * 关闭右键菜单
 */
const closeContextMenu = () => {
  contextMenuVisible.value = false;
};

// ✅ 提供右键菜单显示函数给子组件
provide("showContextMenu", showContextMenu);

/**
 * 处理菜单项点击
 */
const handleContextMenuClick = () => {
  closeContextMenu();
};

/**
 * 删除选中节点
 */
const handleDelete = () => {
  editorStore.removeSelectedNodes();
  closeContextMenu();
};

/**
 * 上移一层
 */
const handleMoveUp = () => {
  editorStore.moveNodeUp();
  closeContextMenu();
};

/**
 * 下移一层
 */
const handleMoveDown = () => {
  editorStore.moveNodeDown();
  closeContextMenu();
};

/**
 * 置于顶层
 */
const handleMoveToTop = () => {
  editorStore.moveNodeToTop();
  closeContextMenu();
};

/**
 * 置于底层
 */
const handleMoveToBottom = () => {
  editorStore.moveNodeToBottom();
  closeContextMenu();
};

/**
 * �������һ��
 */
const handleInsertColLeft = () => {
  editorStore.insertElColLeft();
  closeContextMenu();
};

/**
 * �Ҳ�����һ��
 */
const handleInsertColRight = () => {
  editorStore.insertElColRight();
  closeContextMenu();
};

/**
 * 在选中行上方插入一行
 */
const handleInsertRowUp = () => {
  editorStore.insertElLayoutRowUp();
  closeContextMenu();
};

/**
 * 在选中行下方插入一行
 */
const handleInsertRowDown = () => {
  editorStore.insertElLayoutRowDown();
  closeContextMenu();
};

/**
 * 撤销
 */
const handleUndo = () => {
  if (canUndo.value) {
    editorStore.undo();
  }
  closeContextMenu();
};

/**
 * 重做
 */
const handleRedo = () => {
  if (canRedo.value) {
    editorStore.redo();
  }
  closeContextMenu();
};

// ✅ 左键点击外部关闭菜单
const handleClickOutside = (event) => {
  if (contextMenuVisible.value) {
    closeContextMenu();
  }
};

onMounted(() => {
  document.addEventListener("click", handleClickOutside);
  window.addEventListener("designer:force-refresh", handleForceRefresh);
});

onBeforeUnmount(() => {
  document.removeEventListener("click", handleClickOutside);
  window.removeEventListener("designer:force-refresh", handleForceRefresh);
});

/**
 * 处理键盘事件
 * @param {KeyboardEvent} event - 键盘事件
 */
const handleKeyDown = (event) => {
  // Delete / Backspace 删除选中节点
  if (event.key === "Delete" || event.key === "Backspace") {
    event.preventDefault();
    editorStore.removeSelectedNodes();
    return;
  }

  // Ctrl+Z / Cmd+Z 撤销
  if (
    (event.ctrlKey || event.metaKey) &&
    event.key === "z" &&
    !event.shiftKey
  ) {
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
  if (
    (event.ctrlKey || event.metaKey) &&
    event.key === "]" &&
    !event.shiftKey
  ) {
    event.preventDefault();
    editorStore.moveNodeUp();
    return;
  }

  // Ctrl+[ 下移图层
  if (
    (event.ctrlKey || event.metaKey) &&
    event.key === "[" &&
    !event.shiftKey
  ) {
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

  // Escape 清除选中
  if (event.key === "Escape") {
    selection.value?.clearSelection();
    return;
  }
};
</script>

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

/* ✅ 右键菜单样式 */
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
</style>
