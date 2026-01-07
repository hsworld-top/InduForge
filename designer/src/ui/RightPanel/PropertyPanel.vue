<template>
  <div class="flex flex-col gap-3">
    <div v-if="selectedNode">
      <el-descriptions :column="1" size="small" border>
        <el-descriptions-item label="ID">
          {{ selectedNode.id }}
        </el-descriptions-item>
        <el-descriptions-item label="类型">
          {{ selectedNode.type }}
        </el-descriptions-item>
        <el-descriptions-item label="名称">
          {{ selectedNode.label || "未命名" }}
        </el-descriptions-item>
      </el-descriptions>

      <el-divider />

      <div>
        <div class="text-xs text-gray-500 mb-2">Props</div>
        <pre class="text-xs bg-gray-50 dark:bg-gray-900 p-2 rounded">
{{ formattedProps }}
        </pre>
      </div>
    </div>
    <PageInspectorPanel v-else />
  </div>
</template>

<script setup>
import { computed, onBeforeUnmount, ref, watch } from "vue";
import { storeToRefs } from "pinia";
import { useEditorStore } from "@/stores/editor-store";
import PageInspectorPanel from "./PageInspectorPanel.vue";

const editorStore = useEditorStore();
const { doc, selection } = storeToRefs(editorStore);

const selectedNode = ref(null);
let unsubscribeSelection = null;

/**
 * 同步选中节点
 * @param {{ primary?: { id: string, kind: string } }} payload - 选中事件
 */
const syncSelectedNode = (payload) => {
  const primary = payload?.primary;
  if (primary?.kind === "node" && doc.value) {
    selectedNode.value = doc.value.getNode(primary.id);
  } else {
    selectedNode.value = null;
  }
};

/**
 * 订阅选中变化
 * @param {import('@/editor-core').SelectionModel | null} model - 选中模型
 */
const subscribeSelection = (model) => {
  if (!model) return;
  unsubscribeSelection = model.on("change", syncSelectedNode);
  const primary = model.getPrimaryElement();
  if (primary?.kind === "node" && doc.value) {
    selectedNode.value = doc.value.getNode(primary.id);
  } else {
    selectedNode.value = null;
  }
};

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
  { immediate: true }
);

onBeforeUnmount(() => {
  if (unsubscribeSelection) {
    unsubscribeSelection();
    unsubscribeSelection = null;
  }
});

/**
 * Props 格式化展示
 */
const formattedProps = computed(() => {
  if (!selectedNode.value) return "{}";
  return JSON.stringify(selectedNode.value.props || {}, null, 2);
});
</script>
