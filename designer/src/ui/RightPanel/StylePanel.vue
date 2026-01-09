<template>
  <div class="flex flex-col gap-3 style-panel">
    <template v-if="hasSelection">
      <el-collapse v-model="activeNames">
        <!-- 基础样式 -->
        <el-collapse-item title="基础样式" name="basic">
          <SizeEditor
            :model-value="currentStyle"
            @update:model-value="handleStyleChange"
          />
          <el-divider style="margin: 12px 0" />
          <BackgroundEditor
            :model-value="currentStyle"
            @update:model-value="handleStyleChange"
          />
          <el-divider style="margin: 12px 0" />
          <BorderEditor
            :model-value="currentStyle"
            @update:model-value="handleStyleChange"
          />
        </el-collapse-item>

        <!-- 布局样式 -->
        <el-collapse-item title="布局样式" name="layout">
          <PositionEditor
            :model-value="currentStyle"
            @update:model-value="handleStyleChange"
          />
          <el-divider style="margin: 12px 0" />
          <SpacingEditor
            title="内边距"
            prefix="padding"
            :model-value="currentStyle"
            @update:model-value="handleStyleChange"
          />
          <el-divider style="margin: 12px 0" />
          <SpacingEditor
            title="外边距"
            prefix="margin"
            :model-value="currentStyle"
            @update:model-value="handleStyleChange"
          />
        </el-collapse-item>
      </el-collapse>
    </template>
    <div v-else class="text-sm text-gray-400 text-center py-6">
      请选择组件
    </div>
  </div>
</template>

<script setup>
/**
 * 样式面板
 * 提供基础样式和布局样式的编辑功能
 */
import { computed, ref, watch } from "vue";
import { storeToRefs } from "pinia";
import { useEditorStore } from "@/stores/editor-store";
import SizeEditor from "./StylePanel/SizeEditor.vue";
import SpacingEditor from "./StylePanel/SpacingEditor.vue";
import BackgroundEditor from "./StylePanel/BackgroundEditor.vue";
import BorderEditor from "./StylePanel/BorderEditor.vue";
import PositionEditor from "./StylePanel/PositionEditor.vue";

const editorStore = useEditorStore();
const { doc, selection } = storeToRefs(editorStore);

const activeNames = ref(["basic", "layout"]);

/**
 * 获取当前选中的节点
 */
const selectedNode = computed(() => {
  const primary = selection.value?.getPrimaryElement();
  if (!primary || primary.kind !== "node") return null;
  return doc.value?.getNode(primary.id) || null;
});

/**
 * 是否有选中元素
 */
const hasSelection = computed(() => selectedNode.value !== null);

/**
 * 当前样式
 */
const currentStyle = computed(() => {
  if (!selectedNode.value) return {};
  return selectedNode.value.style || {};
});

/**
 * 处理样式变更
 * @param {Object} newStyle - 新样式对象
 */
const handleStyleChange = (newStyle) => {
  if (!selectedNode.value) return;
  editorStore.updateNode(selectedNode.value.id, { style: newStyle });
};
</script>

<style scoped>
.style-panel {
  padding: 8px;
}

.style-panel :deep(.el-collapse-item__header) {
  font-size: 13px;
  font-weight: 500;
}

.style-panel :deep(.el-collapse-item__content) {
  padding: 12px 8px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}
</style>
