<!--
  脚本面板：自定义脚本折叠块（树 + 拖拽）
-->
<script setup lang="ts">
import { useI18n } from "vue-i18n";
import IconEpEditPen from "~icons/ep/edit-pen";
import IconEpFolder from "~icons/ep/folder";

interface ScriptTreeNodeLike {
  id: string;
  label: string;
  type: string;
}

defineProps<{
  tree: ScriptTreeNodeLike[];
  allowDrop: (draggingNode: unknown, dropNode: unknown, dropType: unknown) => boolean;
  allowDrag: (draggingNode: unknown) => boolean;
  isSelected: (data: ScriptTreeNodeLike) => boolean;
}>();

const emit = defineEmits([
  "blankContextmenu",
  "nodeDblclick",
  "nodeContextmenu",
  "nodeDrop",
  "nodeClick",
]);

function handleNodeDblclick(data: ScriptTreeNodeLike) {
  emit("nodeDblclick", data);
}

function handleNodeContextmenu(event: MouseEvent, data: ScriptTreeNodeLike) {
  emit("nodeContextmenu", event, data);
}

function handleNodeDrop(draggingNode: unknown, dropNode: unknown, dropType: unknown) {
  emit("nodeDrop", draggingNode, dropNode, dropType);
}

function handleNodeClick(data: ScriptTreeNodeLike, event: MouseEvent) {
  emit("nodeClick", data, event);
}
const { t } = useI18n();
</script>

<template>
  <el-collapse-item name="custom">
    <template #title> {{ t("scriptPanel.sections.custom") }} </template>
    <div class="scripts-layout">
      <div class="scripts-list is-full" @contextmenu="emit('blankContextmenu', $event)">
        <el-tree
          :data="tree"
          node-key="id"
          :default-expand-all="true"
          highlight-current
          :expand-on-click-node="false"
          draggable
          :allow-drop="allowDrop"
          :allow-drag="allowDrag"
          @node-click="() => {}"
          @node-dblclick="handleNodeDblclick"
          @node-contextmenu="handleNodeContextmenu"
          @node-drop="handleNodeDrop"
        >
          <template #default="{ data }">
            <div
              class="tree-node"
              :class="[{ 'is-selected': isSelected(data) }, `node-${data.type}`]"
              @click.stop="(event) => handleNodeClick(data, event)"
              @dblclick.stop="handleNodeDblclick(data)"
            >
              <el-icon class="node-icon icon-custom">
                <IconEpFolder v-if="data.type === 'group'" />
                <IconEpEditPen v-else />
              </el-icon>
              <span class="node-label" :class="{ 'is-group': data.type === 'group' }">{{
                data.label
              }}</span>
            </div>
          </template>
        </el-tree>
      </div>
    </div>
  </el-collapse-item>
</template>

<style scoped>
.scripts-layout {
  display: flex;
  gap: 12px;
  flex: 1;
  padding: 6px 4px;
  box-sizing: border-box;
  min-height: 320px;
}

.scripts-list {
  width: 100%;
  border: 1px solid var(--designer-border-color);
  border-radius: 10px;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  background: var(--designer-shell-surface);
  box-shadow: var(--designer-shadow-panel);
  min-height: 320px;
  flex: 1;
}

.scripts-list.is-full {
  flex: 1;
}

:deep(.scripts-list .el-tree) {
  flex: 1;
  overflow: auto;
  padding: 10px 8px;
}

:deep(.scripts-list .el-tree-node__content) {
  height: 38px;
}

.tree-node {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  width: 100%;
  padding: 6px 8px;
  border-radius: 6px;
  transition: background-color 0.2s;
}

.node-icon {
  color: var(--designer-text-muted);
  flex-shrink: 0;
}

.node-group .node-icon {
  color: var(--designer-primary-text);
}

.node-item .node-icon.icon-custom {
  color: var(--designer-success-text);
}

.tree-node:hover {
  background: var(--designer-hover-surface);
}

.tree-node.is-selected {
  background: var(--designer-primary-soft);
  color: var(--designer-text-primary);
}

.node-label {
  font-size: 13px;
  color: var(--designer-text-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.node-label.is-group {
  font-weight: 600;
}
</style>
