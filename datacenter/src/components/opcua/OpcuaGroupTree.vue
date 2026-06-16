<template>
  <aside class="opcua-groups">
    <div class="opcua-groups__toolbar">
      <el-input v-model="keyword" size="small" placeholder="搜索变量组" clearable />
      <button
        type="button"
        class="opcua-groups__icon-action"
        title="新建变量组"
        aria-label="新建变量组"
        @click="$emit('create')"
      >
        <IconTablerFolderPlus />
      </button>
    </div>

    <div class="opcua-groups__tree">
      <button
        type="button"
        class="opcua-groups__root"
        :class="{ 'is-active': selectedGroupId === '' }"
        @click="$emit('select', '')"
        @contextmenu.prevent.stop="openRootMenu"
      >
        <IconTablerStack2 class="opcua-groups__root-icon" />
        <span class="opcua-groups__root-name">全部变量</span>
        <span class="opcua-groups__root-count">{{ total || 0 }}</span>
      </button>

      <template v-if="filteredGroups.length > 0">
        <OpcuaGroupTreeBranch
          v-for="group in filteredGroups"
          :key="group.id"
          :node="group"
          :selected-group-id="selectedGroupId"
          @select="$emit('select', $event)"
          @group-contextmenu="openGroupMenu"
        />
      </template>
      <div v-else class="opcua-groups__empty">
        {{ keyword ? '没有匹配的变量组' : '暂无变量组，可右键全部变量新建' }}
      </div>
    </div>
    <Teleport to="body">
      <div
        v-if="menu.visible"
        class="opcua-groups__menu-mask"
        @click="closeMenu"
        @contextmenu.prevent="closeMenu"
      >
        <div
          class="opcua-groups__context-menu"
          :style="{ left: `${menu.x}px`, top: `${menu.y}px` }"
          @click.stop
        >
          <button type="button" @click="runMenuAction('create-child')">
            <IconTablerFolderPlus />
            <span>{{ menu.group ? '新建子分组' : '新建变量组' }}</span>
          </button>
          <button v-if="menu.group" type="button" @click="runMenuAction('edit')">
            <IconTablerPencil />
            <span>编辑分组</span>
          </button>
          <button
            v-if="menu.group"
            type="button"
            class="is-danger"
            @click="runMenuAction('delete')"
          >
            <IconTablerTrash />
            <span>删除</span>
          </button>
        </div>
      </div>
    </Teleport>
  </aside>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import IconTablerFolderPlus from '~icons/tabler/folder-plus'
import IconTablerPencil from '~icons/tabler/pencil'
import IconTablerStack2 from '~icons/tabler/stack-2'
import IconTablerTrash from '~icons/tabler/trash'
import OpcuaGroupTreeBranch from './OpcuaGroupTreeBranch.vue'
import type { OpcuaNodeGroup } from './types'

type OpcuaGroupTreeNode = OpcuaNodeGroup & {
  children: OpcuaGroupTreeNode[]
}

const props = defineProps<{
  groups: OpcuaNodeGroup[]
  selectedGroupId: string
  total?: number
}>()

const emit = defineEmits<{
  (event: 'select', groupId: string): void
  (event: 'create'): void
  (event: 'create-child', group: OpcuaNodeGroup | null): void
  (event: 'edit', group: OpcuaNodeGroup): void
  (event: 'delete', group: OpcuaNodeGroup): void
}>()

const keyword = ref('')
const menu = ref({
  visible: false,
  x: 0,
  y: 0,
  group: null as OpcuaNodeGroup | null,
})

const treeGroups = computed(() => {
  const nodes = new Map<string, OpcuaGroupTreeNode>()
  props.groups.forEach((group) => {
    nodes.set(group.id, { ...group, children: [] })
  })
  const roots: OpcuaGroupTreeNode[] = []
  nodes.forEach((node) => {
    const parent = node.parentId ? nodes.get(node.parentId) : null
    if (parent) parent.children.push(node)
    else roots.push(node)
  })
  const sortNodes = (items: OpcuaGroupTreeNode[]) => {
    items.sort(
      (left, right) =>
        (left.sortOrder || 0) - (right.sortOrder || 0) || left.name.localeCompare(right.name),
    )
    items.forEach((item) => sortNodes(item.children))
  }
  sortNodes(roots)
  return roots
})

const filteredGroups = computed(() => {
  const text = keyword.value.trim().toLowerCase()
  if (!text) return treeGroups.value
  const filterNode = (group: OpcuaGroupTreeNode): OpcuaGroupTreeNode | null => {
    const children = group.children
      .map(filterNode)
      .filter((child): child is OpcuaGroupTreeNode => Boolean(child))
    const selfMatches = group.name.toLowerCase().includes(text)
    if (selfMatches || children.length > 0) {
      return { ...group, children: selfMatches ? group.children : children }
    }
    return null
  }
  return treeGroups.value
    .map(filterNode)
    .filter((group): group is OpcuaGroupTreeNode => Boolean(group))
})

function openRootMenu(event: MouseEvent) {
  menu.value = {
    visible: true,
    x: Math.min(event.clientX, window.innerWidth - 180),
    y: Math.min(event.clientY, window.innerHeight - 92),
    group: null,
  }
}

function openGroupMenu(event: MouseEvent, group: OpcuaGroupTreeNode) {
  menu.value = {
    visible: true,
    x: Math.min(event.clientX, window.innerWidth - 180),
    y: Math.min(event.clientY, window.innerHeight - 126),
    group,
  }
}

function closeMenu() {
  menu.value.visible = false
}

function runMenuAction(action: 'create-child' | 'edit' | 'delete') {
  const group = menu.value.group
  closeMenu()
  if (action === 'create-child') {
    emit('create-child', group)
    return
  }
  if (!group) return
  emit(action, group)
}
</script>

<style scoped>
.opcua-groups {
  min-width: 0;
  border-right: 1px solid var(--dc-border);
  background: var(--dc-surface-muted);
  display: flex;
  flex-direction: column;
  min-height: 0;
  overflow: hidden;
}

.opcua-groups__toolbar {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 30px;
  gap: 6px;
  padding: 8px;
  border-bottom: 1px solid var(--dc-border);
}

.opcua-groups__icon-action {
  width: 30px;
  height: 30px;
  border: 1px solid color-mix(in oklch, var(--dc-primary) 40%, var(--dc-border));
  border-radius: var(--dc-radius-sm);
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
}

.opcua-groups__icon-action:hover {
  border-color: var(--dc-primary);
}

.opcua-groups__tree {
  min-height: 0;
  flex: 1;
  display: grid;
  align-content: start;
  gap: 2px;
  overflow: auto;
  padding: 8px 8px 12px;
}

.opcua-groups__root {
  width: 100%;
  min-height: 32px;
  display: grid;
  grid-template-columns: 18px minmax(0, 1fr) auto;
  align-items: center;
  gap: 6px;
  padding: 0 6px;
  border: 1px solid transparent;
  border-radius: var(--dc-radius-sm);
  background: transparent;
  color: var(--dc-text-secondary);
  font-size: 13px;
  font-weight: 700;
  text-align: left;
}

.opcua-groups__root:hover {
  border-color: var(--dc-border);
  background: var(--dc-surface-muted);
  color: var(--dc-text);
}

.opcua-groups__root.is-active {
  border-color: rgba(29, 78, 216, 0.28);
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
}

.opcua-groups__root-icon {
  width: 16px;
  height: 16px;
  color: var(--dc-primary);
}

.opcua-groups__root-name {
  min-width: 0;
  overflow: hidden;
  color: var(--dc-text);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.opcua-groups__root.is-active .opcua-groups__root-name {
  color: var(--dc-primary);
}

.opcua-groups__root-count {
  min-width: 20px;
  padding: 2px 6px;
  border-radius: 999px;
  background: var(--dc-surface-muted);
  color: var(--dc-text-muted);
  font-size: 11px;
  text-align: center;
}

.opcua-groups__empty {
  padding: 22px 8px;
  color: var(--dc-text-muted);
  font-size: 13px;
  text-align: center;
}

.opcua-groups__menu-mask {
  position: fixed;
  inset: 0;
  z-index: 2100;
}

.opcua-groups__context-menu {
  position: fixed;
  min-width: 138px;
  padding: 4px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
  box-shadow: var(--dc-shadow-surface);
}

.opcua-groups__context-menu button {
  width: 100%;
  height: 30px;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 0 8px;
  border: 0;
  border-radius: var(--dc-radius-sm);
  background: transparent;
  color: var(--dc-text-secondary);
  font-size: 13px;
  text-align: left;
}

.opcua-groups__context-menu button:hover {
  background: var(--dc-surface-muted);
  color: var(--dc-primary);
}

.opcua-groups__context-menu button.is-danger:hover {
  background: var(--dc-danger-soft);
  color: var(--dc-danger);
}

.opcua-groups__context-menu svg {
  width: 15px;
  height: 15px;
  flex: 0 0 auto;
}
</style>
