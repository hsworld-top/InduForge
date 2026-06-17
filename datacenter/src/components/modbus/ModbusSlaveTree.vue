<template>
  <div class="modbus-slave-tree">
    <div class="modbus-slave-tree__toolbar">
      <button
        type="button"
        class="modbus-slave-tree__icon-action is-primary"
        title="新建从站"
        aria-label="新建从站"
        @click="$emit('create')"
      >
        <IconTablerPlus />
      </button>
      <button
        type="button"
        class="modbus-slave-tree__all"
        :class="{ 'is-active': !selectedSlaveId }"
        @click="$emit('select', '')"
      >
        <IconTablerHierarchy />
        <span>全部从站</span>
      </button>
    </div>
    <div class="modbus-slave-tree__list">
      <button
        v-for="slave in slaves"
        :key="slave.id"
        type="button"
        class="modbus-slave-tree__row"
        :class="{ 'is-active': selectedSlaveId === slave.id, 'is-disabled': !slave.enabled }"
        @click="$emit('select', slave.id)"
      >
        <span class="modbus-slave-tree__unit">{{ slave.unitId }}</span>
        <span class="modbus-slave-tree__name">{{ slave.name }}</span>
        <em>{{ slave.registerCount || 0 }}</em>
        <span class="modbus-slave-tree__actions">
          <button type="button" title="编辑从站" @click.stop="$emit('edit', slave)">
            <IconTablerPencil />
          </button>
          <button type="button" title="删除从站" @click.stop="$emit('delete', slave)">
            <IconTablerTrash />
          </button>
        </span>
      </button>
      <div v-if="slaves.length === 0" class="modbus-slave-tree__empty">暂无从站</div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { ModbusSlaveDevice } from './types'
import IconTablerHierarchy from '~icons/tabler/hierarchy'
import IconTablerPencil from '~icons/tabler/pencil'
import IconTablerPlus from '~icons/tabler/plus'
import IconTablerTrash from '~icons/tabler/trash'

defineProps<{
  slaves: ModbusSlaveDevice[]
  selectedSlaveId: string
}>()

defineEmits<{
  (event: 'select', slaveId: string): void
  (event: 'create'): void
  (event: 'edit', slave: ModbusSlaveDevice): void
  (event: 'delete', slave: ModbusSlaveDevice): void
}>()
</script>

<style scoped>
.modbus-slave-tree {
  min-height: 0;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.modbus-slave-tree__toolbar {
  display: flex;
  gap: 8px;
  align-items: center;
}

.modbus-slave-tree__icon-action,
.modbus-slave-tree__all,
.modbus-slave-tree__row .modbus-slave-tree__actions button {
  border: 1px solid var(--dc-border);
  background: var(--dc-surface);
  color: var(--dc-text);
  cursor: pointer;
}

.modbus-slave-tree__icon-action {
  width: 32px;
  height: 32px;
  border-radius: var(--dc-radius-sm);
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

.modbus-slave-tree__icon-action.is-primary {
  color: var(--dc-primary);
  border-color: rgba(37, 99, 235, 0.28);
}

.modbus-slave-tree__all {
  flex: 1;
  min-width: 0;
  height: 32px;
  border-radius: var(--dc-radius-sm);
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 0 10px;
  font-size: 13px;
}

.modbus-slave-tree__list {
  min-height: 0;
  overflow: auto;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.modbus-slave-tree__row {
  width: 100%;
  min-width: 0;
  height: 36px;
  border: 1px solid transparent;
  border-radius: var(--dc-radius-sm);
  background: transparent;
  color: var(--dc-text);
  display: grid;
  grid-template-columns: 34px minmax(0, 1fr) 26px auto;
  align-items: center;
  gap: 6px;
  padding: 0 6px;
  cursor: pointer;
  text-align: left;
}

.modbus-slave-tree__row:hover,
.modbus-slave-tree__row.is-active,
.modbus-slave-tree__all.is-active {
  border-color: rgba(37, 99, 235, 0.28);
  background: rgba(37, 99, 235, 0.08);
}

.modbus-slave-tree__row.is-disabled {
  color: var(--dc-text-muted);
}

.modbus-slave-tree__unit {
  width: 28px;
  height: 22px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: var(--dc-radius-xs);
  background: rgba(15, 23, 42, 0.06);
  font-size: 12px;
  font-weight: 700;
}

.modbus-slave-tree__name {
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
  font-size: 13px;
}

.modbus-slave-tree__row em {
  font-style: normal;
  color: var(--dc-text-muted);
  font-size: 12px;
}

.modbus-slave-tree__actions {
  display: none;
  align-items: center;
  gap: 4px;
}

.modbus-slave-tree__row:hover .modbus-slave-tree__actions {
  display: inline-flex;
}

.modbus-slave-tree__actions button {
  width: 24px;
  height: 24px;
  border-radius: var(--dc-radius-xs);
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

.modbus-slave-tree__empty {
  padding: 18px 8px;
  text-align: center;
  color: var(--dc-text-muted);
  font-size: 13px;
}
</style>
