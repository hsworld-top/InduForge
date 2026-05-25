<template>
  <aside class="modbus-group-tree">
    <div class="modbus-group-tree__head">
      <strong>寄存器组</strong>
      <button type="button" title="新建寄存器组" aria-label="新建寄存器组" @click="$emit('create')">
        <IconTablerPlus />
      </button>
    </div>
    <el-input v-model="keyword" size="small" placeholder="搜索寄存器组" clearable />
    <div class="modbus-group-tree__list">
      <div
        role="button"
        tabindex="0"
        type="button"
        class="modbus-group-tree__item"
        :class="{ 'is-active': !selectedGroupId }"
        @click="$emit('select', '')"
      >
        <IconTablerStack2 />
        <span>全部变量</span>
        <em>{{ registers.length }}</em>
        <i></i>
        <i></i>
      </div>
      <div
        v-for="group in filteredGroups"
        :key="group.id"
        role="button"
        tabindex="0"
        class="modbus-group-tree__item"
        :class="{ 'is-active': selectedGroupId === group.id }"
        @click="$emit('select', group.id)"
      >
        <IconTablerFolder />
        <span>{{ group.name }}</span>
        <em>{{ countByGroup[group.id] || 0 }}</em>
        <button type="button" class="modbus-group-tree__action" title="编辑" aria-label="编辑" @click.stop="$emit('edit', group)">
          <IconTablerPencil />
        </button>
        <button type="button" class="modbus-group-tree__action" title="删除" aria-label="删除" @click.stop="$emit('delete', group)">
          <IconTablerTrash />
        </button>
      </div>
      <div v-if="filteredGroups.length === 0" class="modbus-group-tree__empty">没有匹配的寄存器组</div>
    </div>
  </aside>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import type { ModbusRegister, ModbusRegisterGroup } from './types'
import IconTablerFolder from '~icons/tabler/folder'
import IconTablerPencil from '~icons/tabler/pencil'
import IconTablerPlus from '~icons/tabler/plus'
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
const filteredGroups = computed(() => {
  const text = keyword.value.trim().toLowerCase()
  if (!text) return props.groups
  return props.groups.filter((group) => group.name.toLowerCase().includes(text))
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
  padding: 12px 10px;
  border-right: 1px solid var(--dc-border);
  background:
    linear-gradient(180deg, color-mix(in oklch, var(--dc-surface-subtle) 88%, var(--dc-primary) 12%), var(--dc-surface-subtle)),
    var(--dc-surface-subtle);
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.modbus-group-tree__head {
  height: 28px;
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.modbus-group-tree__head strong {
  color: var(--dc-text);
  font-size: 13px;
}

.modbus-group-tree button {
  border: 0;
  background: transparent;
  color: inherit;
}

.modbus-group-tree__head button {
  width: 28px;
  height: 28px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
  color: var(--dc-text-secondary);
}

.modbus-group-tree__list {
  min-height: 0;
  overflow: auto;
  display: grid;
  gap: 4px;
}

.modbus-group-tree__item {
  width: 100%;
  min-height: 34px;
  padding: 0 7px;
  display: grid;
  grid-template-columns: 18px minmax(0, 1fr) auto 22px 22px;
  align-items: center;
  gap: 6px;
  border: 1px solid transparent;
  border-radius: var(--dc-radius-sm);
  color: var(--dc-text-secondary);
  text-align: left;
  cursor: pointer;
}

.modbus-group-tree__item:hover {
  border-color: color-mix(in oklch, var(--dc-primary) 16%, var(--dc-border));
  background: color-mix(in oklch, var(--dc-surface-raised) 86%, var(--dc-primary) 14%);
  color: var(--dc-text);
}

.modbus-group-tree__item.is-active {
  border-color: color-mix(in oklch, var(--dc-primary) 34%, var(--dc-border));
  background: var(--dc-surface-raised);
  color: var(--dc-text);
  box-shadow: inset 3px 0 0 var(--dc-primary);
}

.modbus-group-tree__item span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 12px;
}

.modbus-group-tree__item em {
  min-width: 22px;
  height: 18px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 999px;
  background: color-mix(in oklch, var(--dc-surface-raised) 68%, var(--dc-border) 32%);
  font-style: normal;
  color: var(--dc-text-muted);
  font-size: 11px;
}

.modbus-group-tree__action {
  width: 22px;
  height: 22px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: var(--dc-radius-sm);
  color: var(--dc-text-muted);
  opacity: 0;
}

.modbus-group-tree__item:hover .modbus-group-tree__action,
.modbus-group-tree__item.is-active .modbus-group-tree__action {
  opacity: 1;
}

.modbus-group-tree__action:hover {
  background: color-mix(in oklch, var(--dc-primary) 10%, var(--dc-surface-raised));
  color: var(--dc-primary);
}

.modbus-group-tree__empty {
  padding: 18px 8px;
  color: var(--dc-text-muted);
  font-size: 12px;
  text-align: center;
}
</style>
