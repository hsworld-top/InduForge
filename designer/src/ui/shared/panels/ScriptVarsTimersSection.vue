<!--
  脚本面板：定时器折叠块（树 + 拖拽）
-->
<script setup>
import IconEpFolder from "~icons/ep/folder";
import IconEpTimer from "~icons/ep/timer";

defineProps({
  tree: { type: Array, required: true },
  allowDrop: { type: Function, required: true },
  allowDrag: { type: Function, required: true },
  isSelected: { type: Function, required: true },
});

const emit = defineEmits([
  "blankContextmenu",
  "nodeDblclick",
  "nodeContextmenu",
  "nodeDrop",
  "nodeClick",
]);
</script>

<template>
  <el-collapse-item name="timers">
    <template #title> 定时器 </template>
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
          @node-dblclick="(data) => emit('nodeDblclick', data)"
          @node-contextmenu="(event, data) => emit('nodeContextmenu', event, data)"
          @node-drop="
            (draggingNode, dropNode, dropType) => emit('nodeDrop', draggingNode, dropNode, dropType)
          "
        >
          <template #default="{ data }">
            <div
              class="tree-node"
              :class="[{ 'is-selected': isSelected(data) }, `node-${data.type}`]"
              @click.stop="(event) => emit('nodeClick', data, event)"
              @dblclick.stop="emit('nodeDblclick', data)"
            >
              <el-icon class="node-icon icon-timer">
                <IconEpFolder v-if="data.type === 'group'" />
                <IconEpTimer v-else />
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
  border: 1px solid #e4e7ed;
  border-radius: 10px;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  background: #fff;
  box-shadow: 0 6px 18px rgba(15, 23, 42, 0.06);
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
  color: #94a3b8;
  flex-shrink: 0;
}

.node-group .node-icon {
  color: #3b82f6;
}

.node-item .node-icon.icon-timer {
  color: #f59e0b;
}

.tree-node:hover {
  background: #f5f7fa;
}

.tree-node.is-selected {
  background: #e8f3ff;
  color: #303133;
}

.node-label {
  font-size: 13px;
  color: #303133;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.node-label.is-group {
  font-weight: 600;
}
</style>
