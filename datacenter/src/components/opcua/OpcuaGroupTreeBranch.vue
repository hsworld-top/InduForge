<template>
  <section class="opcua-group-tree-branch">
    <button
      type="button"
      class="opcua-group-tree-branch__group"
      :class="{ 'is-active': selectedGroupId === node.id }"
      @click="$emit('select', node.id)"
      @dblclick="expanded = !expanded"
      @contextmenu.prevent.stop="$emit('groupContextmenu', $event, node)"
    >
      <IconTablerChevronRight
        class="opcua-group-tree-branch__chevron"
        :class="{ 'is-open': expanded }"
        @click.stop="expanded = !expanded"
      />
      <IconTablerFolderOpen v-if="expanded" class="opcua-group-tree-branch__icon" />
      <IconTablerFolder v-else class="opcua-group-tree-branch__icon" />
      <el-tooltip :content="node.name" placement="top" :show-after="400">
        <span class="opcua-group-tree-branch__group-name">{{ node.name }}</span>
      </el-tooltip>
    </button>

    <div v-if="expanded && node.children.length > 0" class="opcua-group-tree-branch__children">
      <OpcuaGroupTreeBranch
        v-for="child in node.children"
        :key="child.id"
        :node="child"
        :selected-group-id="selectedGroupId"
        @select="$emit('select', $event)"
        @group-contextmenu="(event, group) => $emit('groupContextmenu', event, group)"
      />
    </div>
  </section>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import IconTablerChevronRight from '~icons/tabler/chevron-right'
import IconTablerFolder from '~icons/tabler/folder'
import IconTablerFolderOpen from '~icons/tabler/folder-open'
import type { OpcuaNodeGroup } from './types'

defineOptions({ name: 'OpcuaGroupTreeBranch' })

type OpcuaGroupTreeNode = OpcuaNodeGroup & {
  children: OpcuaGroupTreeNode[]
}

defineProps<{
  node: OpcuaGroupTreeNode
  selectedGroupId: string
}>()

defineEmits<{
  (event: 'select', groupId: string): void
  (event: 'groupContextmenu', mouseEvent: MouseEvent, group: OpcuaGroupTreeNode): void
}>()

const expanded = ref(true)
</script>

<style scoped>
.opcua-group-tree-branch {
  display: grid;
  gap: 2px;
}

.opcua-group-tree-branch__group {
  width: 100%;
  min-height: 28px;
  display: grid;
  grid-template-columns: 16px 18px minmax(0, 1fr);
  align-items: center;
  gap: 5px;
  padding: 0 6px;
  border: 1px solid transparent;
  border-radius: var(--dc-radius-sm);
  background: transparent;
  color: var(--dc-text-secondary);
  font-size: 13px;
  font-weight: 700;
  text-align: left;
}

.opcua-group-tree-branch__group:hover {
  border-color: var(--dc-border);
  background: var(--dc-surface-muted);
  color: var(--dc-text);
}

.opcua-group-tree-branch__group.is-active {
  border-color: rgba(29, 78, 216, 0.28);
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
}

.opcua-group-tree-branch__chevron {
  width: 14px;
  height: 14px;
  transform: rotate(0deg);
  transition: transform 0.16s ease;
}

.opcua-group-tree-branch__chevron.is-open {
  transform: rotate(90deg);
}

.opcua-group-tree-branch__icon {
  width: 16px;
  height: 16px;
  color: var(--dc-primary);
}

.opcua-group-tree-branch__group-name {
  min-width: 0;
  overflow: hidden;
  color: var(--dc-text);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.opcua-group-tree-branch__group.is-active .opcua-group-tree-branch__group-name {
  color: var(--dc-primary);
}

.opcua-group-tree-branch__children {
  display: grid;
  gap: 2px;
  margin-left: 10px;
  padding-left: 6px;
}
</style>
