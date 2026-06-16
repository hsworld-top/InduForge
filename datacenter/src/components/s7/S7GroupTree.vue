<template>
  <aside class="s7-group-tree">
    <div class="s7-group-tree__toolbar">
      <el-input v-model="keyword" size="small" placeholder="搜索变量组" clearable />
      <button
        type="button"
        class="s7-group-tree__icon-action"
        title="新建变量组"
        aria-label="新建变量组"
        @click="$emit('create')"
      >
        <IconTablerFolderPlus />
      </button>
    </div>
    <button
      type="button"
      class="s7-group-tree__item s7-group-tree__all"
      :class="{ 'is-active': !selectedGroupId }"
      @click="$emit('select', '')"
      @contextmenu.prevent="openRootMenu"
    >
      <IconTablerStack2 />
      <span>全部变量</span>
      <em>{{ !selectedGroupId ? total : '' }}</em>
    </button>
    <div class="s7-group-tree__list">
      <div v-if="filteredGroups.length === 0" class="s7-group-tree__empty">
        {{ keyword ? '没有匹配的变量组' : '暂无变量组，可右键全部变量新建' }}
      </div>
      <div v-for="group in filteredGroups" :key="group.id" class="s7-group-tree__row">
        <button
          type="button"
          class="s7-group-tree__item"
          :class="{ 'is-active': selectedGroupId === group.id }"
          :style="{ paddingLeft: `${12 + group.depth * 16}px` }"
          @click="$emit('select', group.id)"
          @contextmenu.prevent="openGroupMenu($event, group)"
        >
          <IconTablerFolder />
          <span>{{ group.name }}</span>
          <em>{{ selectedGroupId === group.id ? total : '' }}</em>
        </button>
        <div class="s7-group-tree__actions">
          <el-tooltip content="编辑变量组" placement="top">
            <el-button text size="small" :icon="IconTablerEdit" @click="$emit('edit', group)" />
          </el-tooltip>
          <el-tooltip content="删除变量组" placement="top">
            <el-button text size="small" :icon="IconTablerTrash" @click="$emit('delete', group)" />
          </el-tooltip>
        </div>
      </div>
    </div>
    <Teleport to="body">
      <div
        v-if="menu.visible"
        class="s7-group-tree__menu-mask"
        @click="closeMenu"
        @contextmenu.prevent="closeMenu"
      >
        <div
          class="s7-group-tree__context-menu"
          :style="{ left: `${menu.x}px`, top: `${menu.y}px` }"
          @click.stop
        >
          <button type="button" @click="runMenuAction('create-child')">
            <IconTablerFolderPlus />
            <span>{{ menu.group ? '新建子分组' : '新建变量组' }}</span>
          </button>
          <button v-if="menu.group" type="button" @click="runMenuAction('edit')">
            <IconTablerEdit />
            <span>重命名</span>
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
import type { S7VariableGroup } from './types'
import IconTablerEdit from '~icons/tabler/edit'
import IconTablerFolder from '~icons/tabler/folder'
import IconTablerFolderPlus from '~icons/tabler/folder-plus'
import IconTablerStack2 from '~icons/tabler/stack-2'
import IconTablerTrash from '~icons/tabler/trash'

const props = defineProps<{
  groups: S7VariableGroup[]
  selectedGroupId: string
  total?: number
}>()

const emit = defineEmits<{
  (event: 'select', groupId: string): void
  (event: 'create'): void
  (event: 'create-child', group: S7VariableGroup | null): void
  (event: 'edit', group: S7VariableGroup): void
  (event: 'delete', group: S7VariableGroup): void
}>()

const keyword = ref('')
const menu = ref({
  visible: false,
  x: 0,
  y: 0,
  group: null as S7VariableGroup | null,
})
const treeGroups = computed(() => {
  const children = new Map<string, S7VariableGroup[]>()
  props.groups.forEach((group) => {
    const parent = group.parentId || ''
    children.set(parent, [...(children.get(parent) || []), group])
  })
  const walk = (parentId = '', depth = 0): Array<S7VariableGroup & { depth: number }> =>
    (children.get(parentId) || []).flatMap((group) => [
      { ...group, depth },
      ...walk(group.id, depth + 1),
    ])
  return walk()
})
const filteredGroups = computed(() => {
  const text = keyword.value.trim().toLowerCase()
  if (!text) return treeGroups.value
  return treeGroups.value.filter((group) =>
    [group.name, group.code].some((value) => value.toLowerCase().includes(text)),
  )
})

function openRootMenu(event: MouseEvent) {
  menu.value = { visible: true, x: event.clientX, y: event.clientY, group: null }
}

function openGroupMenu(event: MouseEvent, group: S7VariableGroup) {
  menu.value = { visible: true, x: event.clientX, y: event.clientY, group }
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
.s7-group-tree {
  min-width: 0;
  min-height: 0;
  border-right: 1px solid var(--dc-border);
  background: var(--dc-surface-muted);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
.s7-group-tree__empty {
  color: var(--dc-text-muted);
  font-size: 12px;
}
.s7-group-tree__toolbar {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 30px;
  gap: 6px;
  padding: 8px;
  border-bottom: 1px solid var(--dc-border);
}
.s7-group-tree__icon-action {
  width: 30px;
  height: 30px;
  border: 1px solid color-mix(in oklch, var(--dc-primary) 40%, var(--dc-border));
  border-radius: var(--dc-radius-sm);
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
  display: inline-flex;
  align-items: center;
  justify-content: center;
}
.s7-group-tree__all {
  margin: 8px 8px 0;
}
.s7-group-tree__item {
  width: 100%;
  min-height: 30px;
  border: 1px solid transparent;
  border-radius: var(--dc-radius-sm);
  background: transparent;
  display: grid;
  grid-template-columns: 18px minmax(0, 1fr) auto;
  align-items: center;
  gap: 6px;
  padding: 3px 6px;
  color: var(--dc-text-secondary);
  text-align: left;
  cursor: pointer;
  font-size: 12px;
}
.s7-group-tree__item > svg {
  width: 15px;
  height: 15px;
  color: var(--dc-primary);
}
.s7-group-tree__item:hover,
.s7-group-tree__item.is-active {
  border-color: var(--dc-border);
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
}
.s7-group-tree__item span {
  min-width: 0;
  overflow: hidden;
  color: var(--dc-text);
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 13px;
  font-weight: 700;
}
.s7-group-tree__item em {
  min-width: 22px;
  height: 18px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 0 6px;
  border: 1px solid var(--dc-border);
  border-radius: 999px;
  background: var(--dc-surface-raised);
  color: var(--dc-text-muted);
  font-style: normal;
  font-size: 11px;
  font-weight: 700;
}
.s7-group-tree__list {
  min-height: 0;
  flex: 1;
  display: grid;
  align-content: start;
  gap: 2px;
  overflow: auto;
  padding: 8px 8px 12px;
}
.s7-group-tree__row {
  position: relative;
}
.s7-group-tree__actions {
  position: absolute;
  right: 3px;
  top: 1px;
  display: none;
  align-items: center;
  height: 28px;
  padding-left: 4px;
  border-radius: var(--dc-radius-sm);
  background: var(--dc-primary-soft);
}
.s7-group-tree__row:hover .s7-group-tree__actions {
  display: flex;
}
.s7-group-tree__empty {
  padding: 22px 8px;
  text-align: center;
}

.s7-group-tree__menu-mask {
  position: fixed;
  inset: 0;
  z-index: 2100;
}

.s7-group-tree__context-menu {
  position: fixed;
  min-width: 138px;
  padding: 4px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
  box-shadow: var(--dc-shadow-surface);
}

.s7-group-tree__context-menu button {
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

.s7-group-tree__context-menu button:hover {
  background: var(--dc-surface-muted);
  color: var(--dc-primary);
}

.s7-group-tree__context-menu button.is-danger:hover {
  background: var(--dc-danger-soft);
  color: var(--dc-danger);
}

.s7-group-tree__context-menu svg {
  width: 15px;
  height: 15px;
  flex: 0 0 auto;
}
</style>
