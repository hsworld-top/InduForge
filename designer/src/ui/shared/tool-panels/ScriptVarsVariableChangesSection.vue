<!--
  脚本面板：变量改变折叠块（树 + 拖拽）
-->
<script setup lang="ts">
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import IconEpEditPen from "~icons/ep/edit-pen";
import IconEpFolder from "~icons/ep/folder";
import IconEpPlus from "~icons/ep/plus";
import IconEpRefresh from "~icons/ep/refresh";

interface ScriptTreeNodeLike {
  id: string;
  label: string;
  type: string;
}

const props = defineProps<{
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
  "createScript",
]);
const { t } = useI18n();

const itemCount = computed(() => countItems(props.tree));

function countItems(nodes: ScriptTreeNodeLike[]): number {
  return nodes.reduce((sum, node: any) => {
    if (node.type === "item") return sum + 1;
    return sum + countItems(node.children || []);
  }, 0);
}

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
</script>

<template>
  <el-collapse-item name="variableChanges">
    <template #title>
      <div class="section-title">
        <span>{{ t("scriptPanel.sections.globalVariableChanges") }}</span>
        <span class="section-count">{{ itemCount }}</span>
        <el-tooltip :content="t('scriptPanel.actions.create')" placement="top">
          <el-button class="section-action" size="small" text circle @click.stop="emit('createScript')">
            <IconEpPlus />
          </el-button>
        </el-tooltip>
      </div>
    </template>
    <div class="scripts-layout">
      <div class="scripts-list is-full" @contextmenu="emit('blankContextmenu', $event)">
        <div v-if="itemCount === 0" class="empty-state">
          <div class="empty-text">{{ t("scriptPanel.empty.variableChanges") }}</div>
        </div>
        <el-tree
          v-else
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
              <el-icon class="node-icon icon-change">
                <IconEpFolder v-if="data.type === 'group'" />
                <IconEpRefresh v-else />
              </el-icon>
              <span class="node-label" :class="{ 'is-group': data.type === 'group' }">{{
                data.label
              }}</span>
              <el-tooltip v-if="data.type === 'item'" :content="t('scriptPanel.actions.edit')" placement="top">
                <el-button class="row-action" size="small" text circle @click.stop="handleNodeDblclick(data)">
                  <IconEpEditPen />
                </el-button>
              </el-tooltip>
            </div>
          </template>
        </el-tree>
      </div>
    </div>
  </el-collapse-item>
</template>

<style scoped>
.scripts-layout {
  display: block;
  padding: 4px 4px 8px;
  box-sizing: border-box;
}

.scripts-list {
  width: 100%;
  border: 0;
  border-radius: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  background: transparent;
  min-height: 0;
}

.scripts-list.is-full {
  flex: 1;
}

:deep(.scripts-list .el-tree) {
  overflow: auto;
  padding: 0;
  background: transparent;
}

:deep(.scripts-list .el-tree-node__content) {
  height: 32px;
  background: transparent;
}

:deep(.scripts-list .el-tree-node__content:hover) {
  background: transparent;
}

:deep(.scripts-list .el-tree-node__expand-icon.is-leaf) {
  display: none;
}

:deep(.scripts-list .el-tree-node__expand-icon) {
  margin-right: 2px;
}

.section-title {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
}

.section-count {
  min-width: 18px;
  height: 18px;
  padding: 0 6px;
  border-radius: 999px;
  background: color-mix(in srgb, var(--designer-border-color) 55%, white);
  color: var(--designer-text-secondary);
  font-size: 12px;
  line-height: 18px;
  text-align: center;
  font-weight: 500;
}

.section-action {
  margin-left: auto;
  color: var(--designer-text-muted);
  opacity: 0;
}

.section-title:hover .section-action {
  color: var(--designer-primary-text);
  opacity: 1;
}

.empty-state {
  display: flex;
  align-items: center;
  justify-content: space-between;
  min-height: 40px;
  padding: 0 8px;
  color: var(--designer-text-muted);
}

.empty-text {
  font-size: 13px;
}

.tree-node {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  width: 100%;
  height: 32px;
  padding: 0 8px;
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

.node-item .node-icon.icon-change {
  color: #8b5cf6;
}

.tree-node:hover {
  background: var(--designer-hover-surface);
}

.tree-node.is-selected {
  background: var(--designer-primary-soft);
  color: var(--designer-text-primary);
}

.node-label {
  flex: 1;
  min-width: 0;
  font-size: 13px;
  color: var(--designer-text-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.node-label.is-group {
  font-weight: 600;
}

.row-action {
  opacity: 0;
  color: var(--designer-primary-text);
}

.tree-node:hover .row-action,
.tree-node.is-selected .row-action {
  opacity: 1;
}
</style>
