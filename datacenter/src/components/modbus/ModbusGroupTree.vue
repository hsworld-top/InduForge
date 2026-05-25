<template>
  <aside class="modbus-group-tree">
    <div class="modbus-group-tree__head">
      <strong>寄存器组</strong>
      <span>{{ groups.length }} 组</span>
    </div>
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
    >
      <IconTablerStack2 />
      <span>全部变量</span>
      <em>{{ registers.length }}</em>
    </button>

    <div class="modbus-group-tree__list">
      <div v-if="filteredGroups.length === 0" class="modbus-group-tree__empty">
        {{ keyword ? '没有匹配的寄存器组' : '暂无寄存器组' }}
      </div>
      <div v-for="group in filteredGroups" :key="group.id" class="modbus-group-tree__row">
        <button
          type="button"
          class="modbus-group-tree__item"
          :class="{ 'is-active': selectedGroupId === group.id }"
          :style="{ paddingLeft: `${12 + group.depth * 16}px` }"
          @click="$emit('select', group.id)"
        >
          <IconTablerFolder />
          <span>{{ group.name }}</span>
          <em>{{ countByGroup[group.id] || 0 }}</em>
        </button>
        <div class="modbus-group-tree__actions">
          <el-button text size="small" :icon="IconTablerEdit" @click="$emit('edit', group)" />
          <el-button text size="small" :icon="IconTablerTrash" @click="$emit('delete', group)" />
        </div>
      </div>
    </div>
  </aside>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import type { ModbusRegister, ModbusRegisterGroup } from './types'
import IconTablerEdit from '~icons/tabler/edit'
import IconTablerFolder from '~icons/tabler/folder'
import IconTablerFolderPlus from '~icons/tabler/folder-plus'
import IconTablerStack2 from '~icons/tabler/stack-2'
import IconTablerTrash from '~icons/tabler/trash'

const props = defineProps<{
  groups: ModbusRegisterGroup[]
  registers: ModbusRegister[]
  selectedGroupId: string
}>()

defineEmits<{
  (event: 'select', groupId: string): void
  (event: 'create'): void
  (event: 'edit', group: ModbusRegisterGroup): void
  (event: 'delete', group: ModbusRegisterGroup): void
}>()

const keyword = ref('')

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
const countByGroup = computed(() => {
  const result: Record<string, number> = {}
  for (const item of props.registers) {
    if (!item.groupId) continue
    result[item.groupId] = (result[item.groupId] || 0) + 1
  }
  return result
})
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

.modbus-group-tree__head {
  height: 38px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 0 10px 0 12px;
  border-bottom: 1px solid var(--dc-border);
  color: var(--dc-text);
}

.modbus-group-tree__head strong {
  font-size: 13px;
}

.modbus-group-tree__head span {
  color: var(--dc-text-muted);
  font-size: 12px;
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
  grid-template-columns: 18px minmax(0, 1fr) auto;
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

.modbus-group-tree__item em {
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

.modbus-group-tree__list {
  min-height: 0;
  flex: 1;
  display: grid;
  align-content: start;
  gap: 2px;
  overflow: auto;
  padding: 8px 8px 12px;
}

.modbus-group-tree__row {
  position: relative;
}

.modbus-group-tree__actions {
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

.modbus-group-tree__row:hover .modbus-group-tree__actions {
  display: flex;
}

.modbus-group-tree__empty {
  padding: 22px 8px;
  color: var(--dc-text-muted);
  font-size: 13px;
  text-align: center;
}
</style>
