<template>
  <el-table
    class="modbus-register-table"
    :data="registers"
    :loading="loading"
    height="100%"
    row-key="id"
    highlight-current-row
    :row-class-name="rowClassName"
    empty-text="暂无 Modbus 变量"
    @row-click="(row) => $emit('select', row)"
  >
    <el-table-column label="变量名" min-width="150">
      <template #default="{ row }">
        <div class="modbus-register-table__name">
          <strong>{{ row.name }}</strong>
          <span>{{ row.code }}</span>
        </div>
      </template>
    </el-table-column>
    <el-table-column label="从站" width="76">
      <template #default="{ row }">
        <span class="modbus-register-table__unit">{{ row.unitId }}</span>
      </template>
    </el-table-column>
    <el-table-column label="区域" min-width="128">
      <template #default="{ row }">
        <span class="modbus-register-table__area" :data-area="row.area">{{ formatArea(row.area) }}</span>
      </template>
    </el-table-column>
    <el-table-column label="地址" width="126">
      <template #default="{ row }">
        <div class="modbus-register-table__address">
          <strong>{{ row.address }}</strong>
          <span>协议 {{ row.protocolAddress }}</span>
        </div>
      </template>
    </el-table-column>
    <el-table-column label="类型" width="92">
      <template #default="{ row }">
        <span class="modbus-register-table__type">{{ row.dataType }}</span>
      </template>
    </el-table-column>
    <el-table-column label="字节序" width="86">
      <template #default="{ row }">{{ row.byteOrder || '-' }}</template>
    </el-table-column>
    <el-table-column label="数据点" min-width="170">
      <template #default="{ row }">
        <span class="modbus-register-table__path">{{ row.datapointPath || '-' }}</span>
      </template>
    </el-table-column>
    <el-table-column label="最近值" width="90">
      <template #default="{ row }">{{ row.lastValue ?? '-' }}</template>
    </el-table-column>
    <el-table-column label="质量" width="84">
      <template #default="{ row }">
        <el-tag size="small" :type="row.quality === 'Good' ? 'success' : 'info'">{{ row.quality || 'unknown' }}</el-tag>
      </template>
    </el-table-column>
    <el-table-column label="状态" width="86">
      <template #default="{ row }">
        <el-tag size="small" :type="row.status === 'active' ? 'success' : 'info'">{{ row.status }}</el-tag>
      </template>
    </el-table-column>
    <el-table-column label="操作" width="92" fixed="right">
      <template #default="{ row }">
        <div class="modbus-register-table__actions">
          <button type="button" title="编辑" aria-label="编辑" @click.stop="$emit('edit', row)">
            <IconTablerPencil />
          </button>
          <button type="button" title="删除" aria-label="删除" @click.stop="$emit('delete', row)">
            <IconTablerTrash />
          </button>
        </div>
      </template>
    </el-table-column>
  </el-table>
</template>

<script setup lang="ts">
import type { ModbusRegister } from './types'
import IconTablerPencil from '~icons/tabler/pencil'
import IconTablerTrash from '~icons/tabler/trash'

const props = defineProps<{
  registers: ModbusRegister[]
  loading?: boolean
  selectedRegisterId?: string
}>()

defineEmits<{
  (event: 'select', register: ModbusRegister): void
  (event: 'edit', register: ModbusRegister): void
  (event: 'delete', register: ModbusRegister): void
}>()

const formatArea = (area: string) => {
  const map: Record<string, string> = {
    coil: 'Coil',
    discrete_input: 'Discrete Input',
    input_register: 'Input Register',
    holding_register: 'Holding Register',
  }
  return map[area] || area
}

const rowClassName = ({ row }: { row: ModbusRegister }) =>
  row.id === props.selectedRegisterId ? 'modbus-register-table__row--selected' : ''
</script>

<style scoped>
.modbus-register-table {
  flex: 1;
}

.modbus-register-table :deep(.el-table__row) {
  cursor: pointer;
}

.modbus-register-table :deep(.modbus-register-table__row--selected td) {
  background: color-mix(in oklch, var(--dc-primary) 8%, var(--dc-surface-raised));
}

.modbus-register-table__name {
  min-width: 0;
  display: grid;
  gap: 2px;
}

.modbus-register-table__name strong {
  overflow: hidden;
  color: var(--dc-text);
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 12px;
}

.modbus-register-table__name span,
.modbus-register-table__path {
  overflow: hidden;
  color: var(--dc-text-muted);
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 11px;
}

.modbus-register-table__unit,
.modbus-register-table__type,
.modbus-register-table__area {
  min-height: 22px;
  display: inline-flex;
  align-items: center;
  padding: 0 7px;
  border: 1px solid var(--dc-border);
  border-radius: 999px;
  background: var(--dc-surface-subtle);
  color: var(--dc-text-secondary);
  font-size: 11px;
  font-weight: 700;
}

.modbus-register-table__unit {
  min-width: 30px;
  justify-content: center;
  color: var(--dc-primary);
}

.modbus-register-table__area[data-area='holding_register'] {
  border-color: color-mix(in oklch, var(--dc-primary) 26%, var(--dc-border));
}

.modbus-register-table__area[data-area='coil'] {
  border-color: color-mix(in oklch, #16a34a 32%, var(--dc-border));
}

.modbus-register-table__address {
  display: grid;
  gap: 1px;
}

.modbus-register-table__address strong {
  color: var(--dc-text);
  font-size: 12px;
  line-height: 16px;
}

.modbus-register-table__address span {
  color: var(--dc-text-muted);
  font-size: 11px;
}

.modbus-register-table__actions {
  display: flex;
  align-items: center;
  gap: 4px;
}

.modbus-register-table__actions button {
  width: 24px;
  height: 24px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
  color: var(--dc-text-secondary);
}

.modbus-register-table__actions button:hover {
  border-color: color-mix(in oklch, var(--dc-primary) 28%, var(--dc-border));
  color: var(--dc-primary);
}
</style>
