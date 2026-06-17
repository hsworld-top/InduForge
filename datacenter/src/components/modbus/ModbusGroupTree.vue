<template>
  <aside class="modbus-group-tree">
    <div class="modbus-group-tree__toolbar">
      <el-input v-model="keyword" size="small" placeholder="搜索寄存器组" clearable />
      <button
        type="button"
        class="modbus-group-tree__icon-action"
        title="新建寄存器组"
        aria-label="新建寄存器组"
        @click="$emit('create')"
      >
        <IconTablerFolderPlus />
      </button>
    </div>

    <button
      type="button"
      class="modbus-group-tree__item modbus-group-tree__all"
      :class="{ 'is-active': !selectedGroupId }"
      @click="$emit('select', '')"
      @contextmenu.prevent="openRootMenu"
    >
      <IconTablerStack2 />
      <span>全部变量</span>
    </button>

    <div class="modbus-group-tree__list">
      <div v-if="filteredGroups.length === 0" class="modbus-group-tree__empty">
        {{ keyword ? '没有匹配的寄存器组' : '暂无寄存器组，可右键全部变量新建' }}
      </div>
      <div v-for="group in filteredGroups" :key="group.id" class="modbus-group-tree__row">
        <button
          type="button"
          class="modbus-group-tree__item"
          :class="{ 'is-active': selectedGroupId === group.id }"
          :style="{ paddingLeft: `${12 + group.depth * 16}px` }"
          @click="$emit('select', group.id)"
          @contextmenu.prevent="openGroupMenu($event, group)"
        >
          <IconTablerFolder />
          <span>{{ group.name }}</span>
        </button>
      </div>
    </div>
    <Teleport to="body">
      <div
        v-if="menu.visible"
        class="modbus-group-tree__menu-mask"
        @click="closeMenu"
        @contextmenu.prevent="closeMenu"
      >
        <div
          class="modbus-group-tree__context-menu"
          :style="{ left: `${menu.x}px`, top: `${menu.y}px` }"
          @click.stop
        >
          <button type="button" @click="runMenuAction('create-child')">
            <IconTablerFolderPlus />
            <span>{{ menu.group ? '新建子分组' : '新建寄存器组' }}</span>
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
import type { ModbusRegisterGroup } from './types'
import IconTablerEdit from '~icons/tabler/edit'
import IconTablerFolder from '~icons/tabler/folder'
import IconTablerFolderPlus from '~icons/tabler/folder-plus'
import IconTablerStack2 from '~icons/tabler/stack-2'
import IconTablerTrash from '~icons/tabler/trash'

const props = defineProps<{
  groups: ModbusRegisterGroup[]
  selectedGroupId: string
}>()

const emit = defineEmits<{
  (event: 'select', groupId: string): void
  (event: 'create'): void
  (event: 'create-child', group: ModbusRegisterGroup | null): void
  (event: 'edit', group: ModbusRegisterGroup): void
  (event: 'delete', group: ModbusRegisterGroup): void
}>()

const keyword = ref('')
const menu = ref({
  visible: false,
  x: 0,
  y: 0,
  group: null as ModbusRegisterGroup | null,
})

const treeGroups = computed(() => {
  const children = new Map<string, ModbusRegisterGroup[]>()
  props.groups.forEach((group) => {
    const parent = group.parentId || ''
    children.set(parent, [...(children.get(parent) || []), group])
  })
  const walk = (parentId = '', depth = 0): Array<ModbusRegisterGroup & { depth: number }> =>
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

function openGroupMenu(event: MouseEvent, group: ModbusRegisterGroup) {
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
.modbus-group-tree {
  min-width: 0;
  min-height: 0;
  border-right: 1px solid var(--dc-border);
  background: var(--dc-surface-muted);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.modbus-group-tree__toolbar {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 30px;
  gap: 6px;
  padding: 8px;
  border-bottom: 1px solid var(--dc-border);
}

.modbus-group-tree__icon-action {
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

.modbus-group-tree__icon-action:hover {
  border-color: var(--dc-primary);
}

.modbus-group-tree__all {
  margin: 8px 8px 0;
}

.modbus-group-tree__item {
  width: 100%;
  min-height: 30px;
  border: 1px solid transparent;
  border-radius: var(--dc-radius-sm);
  background: transparent;
  color: var(--dc-text-secondary);
  display: grid;
  grid-template-columns: 18px minmax(0, 1fr);
  align-items: center;
  gap: 6px;
  padding: 3px 6px;
  text-align: left;
  cursor: pointer;
  font-size: 12px;
}

.modbus-group-tree__item > svg {
  width: 15px;
  height: 15px;
  color: var(--dc-primary);
}

.modbus-group-tree__item:hover {
  border-color: var(--dc-border);
}

.modbus-group-tree__item:hover,
.modbus-group-tree__item.is-active {
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
}

.modbus-group-tree__item span {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--dc-text);
  font-size: 13px;
  font-weight: 700;
}

.modbus-group-tree__item:hover span,
.modbus-group-tree__item.is-active span {
  color: var(--dc-primary);
}

.modbus-group-tree__list {
  min-height: 0;
  flex: 1;
  display: grid;
  align-content: start;
  gap: 2px;
  overflow: auto;
  padding: 8px 8px 12px;
}

.modbus-group-tree__empty {
  padding: 22px 8px;
  color: var(--dc-text-muted);
  font-size: 13px;
  text-align: center;
}

.modbus-group-tree__menu-mask {
  position: fixed;
  inset: 0;
  z-index: 2100;
}

.modbus-group-tree__context-menu {
  position: fixed;
  min-width: 138px;
  padding: 4px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
  box-shadow: var(--dc-shadow-surface);
}

.modbus-group-tree__context-menu button {
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

.modbus-group-tree__context-menu button:hover {
  background: var(--dc-surface-muted);
  color: var(--dc-primary);
}

.modbus-group-tree__context-menu button.is-danger:hover {
  background: var(--dc-danger-soft);
  color: var(--dc-danger);
}

.modbus-group-tree__context-menu svg {
  width: 15px;
  height: 15px;
  flex: 0 0 auto;
}
</style>
