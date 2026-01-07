<template>
  <div class="flex flex-col gap-2">
    <el-tree
      v-if="outlineData.length"
      :data="outlineData"
      node-key="id"
      highlight-current
      :current-node-key="selectedNodeId"
      @node-click="handleSelectNode"
    />
    <div v-else class="text-sm text-gray-400 text-center py-6">
      暂无组件大纲
    </div>
  </div>
</template>

<script setup>
import { computed, onBeforeUnmount, ref, watch } from "vue";
import { storeToRefs } from "pinia";
import { useEditorStore } from "@/stores/editor-store";

const editorStore = useEditorStore();
const { doc, currentPageId, selection } = storeToRefs(editorStore);

const selectedNodeId = ref("");
let unsubscribeSelection = null;

/**
 * 构建组件大纲树
 * @param {import('@/editor-core').DocumentModel} document - 文档模型
 * @param {string} rootNodeId - 根节点 ID
 * @returns {Array<{id: string, label: string, children?: Array}>}
 */
const buildOutlineTree = (document, rootNodeId) => {
  const rootNode = document.getNode(rootNodeId);
  if (!rootNode) return [];

  const buildChildren = (node) =>
    (node.children || []).map((childId) => {
      const child = document.getNode(childId);
      if (!child) return null;
      return {
        id: child.id,
        label: child.label || child.type || child.id,
        children: buildChildren(child),
      };
    }).filter(Boolean);

  return [
    {
      id: rootNode.id,
      label: rootNode.label || rootNode.type || rootNode.id,
      children: buildChildren(rootNode),
    },
  ];
};

/**
 * 组件大纲树数据
 */
const outlineData = computed(() => {
  if (!doc.value || !currentPageId.value) return [];
  const page = doc.value.getPage(currentPageId.value);
  if (!page) return [];
  return buildOutlineTree(doc.value, page.rootNodeId);
});

/**
 * 选中节点
 * @param {{ id: string }} node - 点击的节点
 */
const handleSelectNode = (node) => {
  selection.value?.select({ kind: "node", id: node.id });
};

/**
 * 同步选中状态
 * @param {{ primary?: { id: string, kind: string } }} payload - 选中事件
 */
const syncSelection = (payload) => {
  const primary = payload?.primary;
  selectedNodeId.value = primary?.kind === "node" ? primary.id : "";
};

/**
 * 订阅选中变化
 * @param {import('@/editor-core').SelectionModel | null} model - 选中模型
 */
const subscribeSelection = (model) => {
  if (!model) return;
  unsubscribeSelection = model.on("change", syncSelection);
  const primary = model.getPrimaryElement();
  selectedNodeId.value = primary?.kind === "node" ? primary.id : "";
};

watch(
  () => selection.value,
  (model, prevModel) => {
    if (unsubscribeSelection) {
      unsubscribeSelection();
      unsubscribeSelection = null;
    }
    if (prevModel?.removeAllListeners) {
      // 保持现有监听由订阅函数负责清理
    }
    if (model) {
      subscribeSelection(model);
    }
  },
  { immediate: true }
);

onBeforeUnmount(() => {
  if (unsubscribeSelection) {
    unsubscribeSelection();
    unsubscribeSelection = null;
  }
});
</script>
