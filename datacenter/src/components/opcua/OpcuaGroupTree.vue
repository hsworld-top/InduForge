<template>
  <aside class="opcua-groups">
    <div class="opcua-groups__head">
      <strong>变量组</strong>
      <span>{{ groups.length }} 组</span>
    </div>
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

    <button
      type="button"
      class="opcua-groups__item opcua-groups__all"
      :class="{ 'is-active': selectedGroupId === '' }"
      @click="$emit('select', '')"
      @contextmenu.prevent="openRootMenu"
    >
      <IconTablerStack2 />
      <span>全部变量</span>
      <em class="opcua-groups__count">{{ selectedGroupId === '' ? total : '' }}</em>
    </button>

    <div class="opcua-groups__tree">
      <div v-if="filteredGroups.length === 0" class="opcua-groups__empty">
        {{ keyword ? '没有匹配的变量组' : '暂无变量组' }}
      </div>
      <div v-for="group in filteredGroups" :key="group.id" class="opcua-groups__row">
        <button
          type="button"
          class="opcua-groups__item"
          :class="{ 'is-active': selectedGroupId === group.id }"
          :style="{ paddingLeft: `${12 + group.depth * 16}px` }"
          @click="$emit('select', group.id)"
          @contextmenu.prevent="openGroupMenu($event, group)"
        >
          <IconTablerFolder />
          <span>{{ group.name }}</span>
          <em class="opcua-groups__count">{{ selectedGroupId === group.id ? total : '' }}</em>
        </button>
        <div class="opcua-groups__actions">
          <el-button text size="small" :icon="IconTablerEdit" @click="$emit('edit', group)" />
          <el-button text size="small" :icon="IconTablerTrash" @click="$emit('delete', group)" />
        </div>
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
import IconTablerEdit from '~icons/tabler/edit'
import IconTablerFolder from '~icons/tabler/folder'
import IconTablerFolderPlus from '~icons/tabler/folder-plus'
import IconTablerStack2 from '~icons/tabler/stack-2'
import IconTablerTrash from '~icons/tabler/trash'
import type { OpcuaNodeGroup } from './types'

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
  const children = new Map<string, OpcuaNodeGroup[]>()
  props.groups.forEach((group) => {
    const parent = group.parentId || ''
    children.set(parent, [...(children.get(parent) || []), group])
  })
  const walk = (parentId = '', depth = 0): Array<OpcuaNodeGroup & { depth: number }> =>
    (children.get(parentId) || []).flatMap((group) => [
      { ...group, depth },
      ...walk(group.id, depth + 1),
    ])
  return walk()
})

const filteredGroups = computed(() => {
  const text = keyword.value.trim().toLowerCase()
  if (!text) return treeGroups.value
  return treeGroups.value.filter((group) => group.name.toLowerCase().includes(text))
})

function openRootMenu(event: MouseEvent) {
  menu.value = { visible: true, x: event.clientX, y: event.clientY, group: null }
}

function openGroupMenu(event: MouseEvent, group: OpcuaNodeGroup) {
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
.opcua-groups {
  min-width: 0;
  border-right: 1px solid var(--dc-border);
  background: var(--dc-surface-muted);
  display: flex;
  flex-direction: column;
  min-height: 0;
  overflow: hidden;
}

.opcua-groups__head {
  height: 38px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 0 10px 0 12px;
  border-bottom: 1px solid var(--dc-border);
  color: var(--dc-text);
}

.opcua-groups__head strong {
  font-size: 13px;
}

.opcua-groups__head span {
  color: var(--dc-text-muted);
  font-size: 12px;
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

.opcua-groups__all {
  margin: 8px 8px 0;
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

.opcua-groups__row {
  position: relative;
}

.opcua-groups__item {
  width: 100%;
  min-height: 30px;
  border: 0;
  border: 1px solid transparent;
  border-radius: var(--dc-radius-sm);
  background: transparent;
  color: var(--dc-text-secondary);
  display: grid;
  grid-template-columns: 18px minmax(0, 1fr) auto;
  align-items: center;
  gap: 6px;
  padding: 3px 6px;
  text-align: left;
  cursor: pointer;
  font-size: 12px;
}

.opcua-groups__item > svg {
  width: 15px;
  height: 15px;
  color: var(--dc-primary);
}

.opcua-groups__item span {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--dc-text);
  font-size: 13px;
  font-weight: 700;
}

.opcua-groups__count {
  min-width: 22px;
  height: 18px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 0 6px;
  border-radius: 999px;
  background: var(--dc-surface-raised);
  border: 1px solid var(--dc-border);
  font-style: normal;
  color: var(--dc-text-muted);
  font-size: 11px;
  font-weight: 700;
}

.opcua-groups__item:hover {
  border-color: var(--dc-border);
}

.opcua-groups__item:hover,
.opcua-groups__item.is-active {
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
}

.opcua-groups__item:hover span,
.opcua-groups__item.is-active span {
  color: var(--dc-primary);
}

.opcua-groups__actions {
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

.opcua-groups__row:hover .opcua-groups__actions {
  display: flex;
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
