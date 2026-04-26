<!--
  OutlineTree - 大纲树
  展示当前页面的组件层级结构，支持选中、显隐、锁定、上下移动、删除
-->
<script setup lang="ts">
import { ElMessage, ElMessageBox } from "element-plus";
import { storeToRefs } from "pinia";
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue";
import IconEpArrowDown from "~icons/ep/arrow-down";
import IconEpArrowUp from "~icons/ep/arrow-up";
import IconEpDelete from "~icons/ep/delete";
import IconEpHide from "~icons/ep/hide";
import IconEpLock from "~icons/ep/lock";
import IconEpUnlock from "~icons/ep/unlock";
import IconEpView from "~icons/ep/view";
import { useEditorStore } from "@/stores/editor-store";

interface OutlineNodeLike {
  id: string;
  label: string;
  hidden: boolean;
  locked: boolean;
  isRoot: boolean;
  children?: OutlineNodeLike[];
}

interface TreeContextMenuData {
  data: OutlineNodeLike;
}

function showSuccessMessage(message: string): void {
  ElMessage.success(message as never);
}

function showInfoMessage(message: string): void {
  ElMessage.info(message as never);
}

function showErrorMessage(message: string): void {
  ElMessage.error(message as never);
}

const editorStore = useEditorStore();
const { doc, currentPageId, selection, docVersion } = storeToRefs(editorStore);

const selectedNodeId = ref("");
const contextMenuVisible = ref(false);
const contextMenuNode = ref<OutlineNodeLike | null>(null);
const contextMenuPoint = ref({ x: 0, y: 0 });
let unsubscribeSelection: (() => void) | null = null;

const contextMenuVirtualRef = {
  getBoundingClientRect: () => {
    const { x, y } = contextMenuPoint.value;
    return {
      width: 0,
      height: 0,
      top: y,
      bottom: y,
      left: x,
      right: x,
    };
  },
};

/**
 * 构建组件大纲树（隐藏根节点，直接展示子节点）
 * @param {import('@/editor-core').DocumentModel} document - 文档模型
 * @param {string} rootNodeId - 根节点 ID
 * @returns {Array<{id: string, label: string, hidden: boolean, locked: boolean, isRoot: boolean, children?: Array}>}
 */
function buildOutlineTree(
  document: { getNode: (id: string) => any },
  rootNodeId: string,
): OutlineNodeLike[] {
  const rootNode = document.getNode(rootNodeId);
  if (!rootNode) return [];

  const buildChildren = (node: any): OutlineNodeLike[] =>
    (node.children || [])
      .map((childId: string) => {
        const child = document.getNode(childId);
        if (!child) return null;
        return {
          id: child.id,
          label: child.label || child.type || child.id,
          hidden: child.hidden || false,
          locked: child.locked || false,
          isRoot: false,
          children: buildChildren(child),
        };
      })
      .filter(Boolean);

  return buildChildren(rootNode);
}

/**
 * 组件大纲树数据
 */
const outlineData = computed(() => {
  void docVersion.value;
  if (!doc.value || !currentPageId.value) return [];
  const page = doc.value.getPage(currentPageId.value);
  if (!page) return [];
  return buildOutlineTree(doc.value, page.rootNodeId);
});

/**
 * 选中节点
 * @param {{ id: string }} node - 点击的节点
 */
function handleSelectNode(node: OutlineNodeLike): void {
  selection.value?.select({ kind: "node", id: node.id });
}

/**
 * 同步选中状态
 * @param {{ primary?: { id: string, kind: string } }} payload - 选中事件
 */
function syncSelection(
  payload: { primary?: { id: string; kind: string } } | null | undefined,
): void {
  const primary = payload?.primary;
  selectedNodeId.value = primary?.kind === "node" ? primary.id || "" : "";
}

/**
 * 订阅选中变化
 * @param {import('@/editor-core').SelectionModel | null} model - 选中模型
 */
function subscribeSelection(model: any): void {
  if (!model) return;
  unsubscribeSelection = model.on("change", syncSelection);
  const primary = model.getPrimaryElement();
  selectedNodeId.value = primary?.kind === "node" ? primary.id || "" : "";
}

watch(
  () => selection.value,
  (model) => {
    if (unsubscribeSelection) {
      unsubscribeSelection();
      unsubscribeSelection = null;
    }
    if (model) {
      subscribeSelection(model);
    }
  },
  { immediate: true },
);

onBeforeUnmount(() => {
  if (unsubscribeSelection) {
    unsubscribeSelection();
    unsubscribeSelection = null;
  }
  document.removeEventListener("click", handleGlobalClick);
});

/**
 * 全局点击处理,关闭右键菜单
 */
function handleGlobalClick(): void {
  contextMenuVisible.value = false;
}

onMounted(() => {
  document.addEventListener("click", handleGlobalClick);
});

/**
 * 切换显示/隐藏
 * @param {string} nodeId - 节点 ID
 */
function toggleVisibility(nodeId: string): void {
  if (editorStore.toggleNodeVisibility(nodeId)) {
    const node = doc.value?.getNode(nodeId);
    showSuccessMessage(node?.hidden ? "已隐藏" : "已显示");
  }
}

/**
 * 切换锁定/解锁
 * @param {string} nodeId - 节点 ID
 */
function toggleLock(nodeId: string): void {
  if (editorStore.toggleNodeLock(nodeId)) {
    const node = doc.value?.getNode(nodeId);
    showSuccessMessage(node?.locked ? "已锁定" : "已解锁");
  }
}

/**
 * 上移图层
 * @param {string} nodeId - 节点 ID
 */
function moveUp(nodeId: string): void {
  if (editorStore.moveNodeUp(nodeId)) {
    showSuccessMessage("已上移");
  } else {
    showInfoMessage("已在最上层");
  }
}

/**
 * 下移图层
 * @param {string} nodeId - 节点 ID
 */
function moveDown(nodeId: string): void {
  if (editorStore.moveNodeDown(nodeId)) {
    showSuccessMessage("已下移");
  } else {
    showInfoMessage("已在最下层");
  }
}

/**
 * 处理右键菜单
 * @param {MouseEvent} event - 鼠标事件
 * @param {*} nodeData - 树节点数据
 * @param {*} node - 树节点对象
 */
function handleContextMenu(
  event: MouseEvent,
  nodeData: TreeContextMenuData | null | undefined,
): void {
  event.preventDefault();
  if (!nodeData?.data || nodeData.data.isRoot) return; // 根节点不显示菜单

  contextMenuNode.value = nodeData.data;
  contextMenuPoint.value = { x: event.clientX, y: event.clientY };
  contextMenuVisible.value = true;
}

/**
 * 右键菜单:切换显示/隐藏
 */
function handleToggleVisibility(): void {
  if (!contextMenuNode.value) return;
  toggleVisibility(contextMenuNode.value.id);
  contextMenuVisible.value = false;
}

/**
 * 右键菜单:切换锁定/解锁
 */
function handleToggleLock(): void {
  if (!contextMenuNode.value) return;
  toggleLock(contextMenuNode.value.id);
  contextMenuVisible.value = false;
}

/**
 * 右键菜单:上移图层
 */
function handleMoveUp(): void {
  if (!contextMenuNode.value) return;
  moveUp(contextMenuNode.value.id);
  contextMenuVisible.value = false;
}

/**
 * 右键菜单:下移图层
 */
function handleMoveDown(): void {
  if (!contextMenuNode.value) return;
  moveDown(contextMenuNode.value.id);
  contextMenuVisible.value = false;
}

/**
 * 右键菜单:置顶
 */
function handleMoveToTop(): void {
  if (!contextMenuNode.value) return;
  if (editorStore.moveNodeToTop(contextMenuNode.value.id)) {
    showSuccessMessage("已置顶");
  }
  contextMenuVisible.value = false;
}

/**
 * 右键菜单:置底
 */
function handleMoveToBottom(): void {
  if (!contextMenuNode.value) return;
  if (editorStore.moveNodeToBottom(contextMenuNode.value.id)) {
    showSuccessMessage("已置底");
  }
  contextMenuVisible.value = false;
}

/**
 * 删除节点
 * @param {string} nodeId - 节点 ID
 */
function deleteNode(nodeId: string): void {
  if (!nodeId) return;

  const node = doc.value?.getNode(nodeId);
  const nodeName = node?.label || node?.type || "节点";

  ElMessageBox.confirm(`确定要删除"${nodeName}"吗？此操作不可撤销。`, "删除确认", {
    confirmButtonText: "确定",
    cancelButtonText: "取消",
    type: "warning",
  })
    .then(() => {
      if (editorStore.removeNode(nodeId)) {
        showSuccessMessage("删除成功");
      } else {
        showErrorMessage("删除失败");
      }
    })
    .catch(() => {
      // 用户取消
    });
}

/**
 * 右键菜单:删除
 */
function handleDelete(): void {
  if (!contextMenuNode.value) return;
  deleteNode(contextMenuNode.value.id);
  contextMenuVisible.value = false;
}
</script>

<template>
  <div class="flex flex-col gap-2 outline-tree-root">
    <el-tree
      v-if="outlineData.length"
      :data="outlineData"
      node-key="id"
      highlight-current
      :current-node-key="selectedNodeId"
      @node-click="handleSelectNode"
      @node-contextmenu="handleContextMenu"
    >
      <template #default="{ data }">
        <div class="outline-node" :class="{ 'is-hidden': data.hidden, 'is-locked': data.locked }">
          <el-button
            v-if="!data.isRoot"
            size="small"
            text
            class="visibility-btn"
            @click.stop="toggleVisibility(data.id)"
          >
            <IconEpView v-if="!data.hidden" />
            <IconEpHide v-else />
          </el-button>
          <el-button
            v-if="!data.isRoot"
            size="small"
            text
            class="lock-btn"
            @click.stop="toggleLock(data.id)"
          >
            <IconEpUnlock v-if="!data.locked" />
            <IconEpLock v-else />
          </el-button>
          <span class="node-label">{{ data.label }}</span>
          <div v-if="!data.isRoot" class="layer-actions">
            <el-button size="small" text @click.stop="moveUp(data.id)">
              <IconEpArrowUp />
            </el-button>
            <el-button size="small" text @click.stop="moveDown(data.id)">
              <IconEpArrowDown />
            </el-button>
            <el-button size="small" text class="delete-btn" @click.stop="deleteNode(data.id)">
              <IconEpDelete />
            </el-button>
          </div>
        </div>
      </template>
    </el-tree>
    <div v-else class="text-sm text-gray-400 text-center py-6">暂无组件大纲</div>
  </div>

  <!-- 右键菜单 -->
  <el-popover
    v-model:visible="contextMenuVisible"
    :virtual-ref="contextMenuVirtualRef"
    virtual-triggering
    trigger="manual"
    placement="bottom-start"
    popper-class="outline-context-menu"
  >
    <div v-if="contextMenuNode" class="context-menu" @click.stop>
      <el-button size="small" text @click="handleToggleVisibility">
        {{ contextMenuNode.hidden ? "显示" : "隐藏" }}
      </el-button>
      <el-button size="small" text @click="handleToggleLock">
        {{ contextMenuNode.locked ? "解锁" : "锁定" }}
      </el-button>
      <el-divider style="margin: 4px 0" />
      <el-button size="small" text @click="handleMoveUp">上移图层</el-button>
      <el-button size="small" text @click="handleMoveDown">下移图层</el-button>
      <el-button size="small" text @click="handleMoveToTop">置顶</el-button>
      <el-button size="small" text @click="handleMoveToBottom">置底</el-button>
      <el-divider style="margin: 4px 0" />
      <el-button size="small" text class="delete-btn" @click="handleDelete">删除</el-button>
    </div>
  </el-popover>
</template>

<style scoped>
.outline-tree-root :deep(.el-tree-node__content) {
  padding-right: 8px;
}

.outline-node {
  display: flex;
  align-items: center;
  gap: 4px;
  width: 100%;
  padding: 2px 0;
}

.outline-node.is-hidden {
  opacity: 0.5;
}

.outline-node.is-locked .node-label::after {
  content: "🔒";
  margin-left: 4px;
  font-size: 10px;
}

.node-label {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.visibility-btn,
.lock-btn {
  padding: 2px;
  height: auto;
  opacity: 0.5;
  transition: opacity 0.2s;
}

.visibility-btn:hover,
.lock-btn:hover {
  opacity: 1;
}

.layer-actions {
  display: flex;
  gap: 2px;
  opacity: 0;
  transition: opacity 0.2s;
}

.outline-node:hover .layer-actions {
  opacity: 1;
}

.layer-actions .el-button {
  padding: 2px;
  height: auto;
}

.delete-btn {
  color: var(--designer-danger-text);
}

.delete-btn:hover {
  color: var(--designer-danger-text);
  opacity: 1;
}

.outline-context-menu {
  padding: 6px 8px;
}

.context-menu {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 120px;
}

.context-menu .el-button {
  justify-content: flex-start;
}
</style>
