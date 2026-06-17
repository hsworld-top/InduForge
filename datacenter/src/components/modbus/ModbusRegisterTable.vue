<template>
  <div class="modbus-register-table">
    <el-table
      class="modbus-register-table__grid"
      :data="registers"
      :loading="loading"
      height="100%"
      row-key="id"
      highlight-current-row
      :row-class-name="rowClassName"
      empty-text="暂无 Modbus 变量"
      @row-click="(row) => $emit('select', row)"
      @row-contextmenu="(row, _column, event) => $emit('row-contextmenu', event, row)"
      @selection-change="(rows) => $emit('selection-change', rows)"
      @sort-change="(payload) => $emit('sort-change', payload)"
    >
      <el-table-column type="selection" width="44" />
      <el-table-column label="变量" min-width="190" prop="name" sortable="custom">
        <template #default="{ row }">
          <div class="modbus-register-table__name">
            <strong>{{ row.name }}</strong>
            <span>{{ formatGroup(row.groupId) }}</span>
          </div>
        </template>
      </el-table-column>
      <el-table-column label="从站" width="72" prop="unitId" sortable="custom">
        <template #default="{ row }">
          <span class="modbus-register-table__unit">{{ row.unitId }}</span>
        </template>
      </el-table-column>
      <el-table-column label="地址" min-width="172" prop="address" sortable="custom">
        <template #default="{ row }">
          <div class="modbus-register-table__address">
            <span class="modbus-register-table__area" :data-area="row.area">{{
              formatArea(row.area)
            }}</span>
            <strong>{{ row.address }}</strong>
            <span>协议 {{ row.protocolAddress }}</span>
          </div>
        </template>
      </el-table-column>
      <el-table-column label="类型" width="88" prop="dataType" sortable="custom">
        <template #default="{ row }">
          <span class="modbus-register-table__type">{{ row.dataType }}</span>
        </template>
      </el-table-column>
      <el-table-column label="采集" width="112" prop="pollIntervalMs" sortable="custom">
        <template #default="{ row }">
          <div class="modbus-register-table__collect">
            <strong>{{ row.pollIntervalMs }}ms</strong>
            <span :class="{ 'is-muted': row.status !== 'active' }">{{
              formatStatus(row.status)
            }}</span>
          </div>
        </template>
      </el-table-column>
      <el-table-column label="最近值" width="116">
        <template #default="{ row }">
          <div class="modbus-register-table__value">
            <strong>{{ formatValue(row.lastValue) }}</strong>
            <span>{{ formatQuality(row.quality) }}</span>
          </div>
        </template>
      </el-table-column>
    </el-table>
    <div class="modbus-register-table__pagination">
      <el-pagination
        :current-page="page"
        :page-size="pageSize"
        :page-sizes="[20, 50, 100]"
        :total="total"
        background
        layout="total, sizes, prev, pager, next, jumper"
        small
        @current-change="$emit('page-change', $event)"
        @size-change="$emit('page-size-change', $event)"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import type { ModbusRegister } from './types'

const props = defineProps<{
  registers: ModbusRegister[]
  loading?: boolean
  selectedRegisterId?: string
  page: number
  pageSize: number
  total: number
  groupFormatter?: (groupId?: string | null) => string
}>()

defineEmits<{
  (event: 'select', register: ModbusRegister): void
  (event: 'detail', register: ModbusRegister): void
  (event: 'duplicate', register: ModbusRegister): void
  (event: 'edit', register: ModbusRegister): void
  (event: 'delete', register: ModbusRegister): void
  (event: 'row-contextmenu', mouseEvent: MouseEvent, register: ModbusRegister): void
  (event: 'selection-change', rows: ModbusRegister[]): void
  (event: 'page-change', page: number): void
  (event: 'page-size-change', pageSize: number): void
  (event: 'sort-change', payload: { prop?: string; order?: string | null }): void
}>()

const formatArea = (area: string) => {
  const map: Record<string, string> = {
    coil: '线圈',
    discrete_input: '离散输入',
    input_register: '输入寄存器',
    holding_register: '保持寄存器',
  }
  return map[area] || area
}

const formatStatus = (status?: string) => (status === 'active' ? '启用' : '停用')

const formatQuality = (quality?: string) => {
  if (quality === 'Good') return '质量正常'
  if (quality === 'Bad') return '质量异常'
  return '暂无质量'
}

const formatValue = (value: unknown) => {
  if (value === null || value === undefined || value === '') return '-'
  if (typeof value === 'object') return JSON.stringify(value)
  return String(value)
}

const rowClassName = ({ row }: { row: ModbusRegister }) =>
  row.id === props.selectedRegisterId ? 'modbus-register-table__row--selected' : ''

const formatGroup = (groupId?: string | null) => props.groupFormatter?.(groupId) || '未分组'
</script>

<style scoped>
.modbus-register-table {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
}

.modbus-register-table__grid {
  flex: 1;
  min-height: 0;
}

.modbus-register-table__grid :deep(.el-table__row) {
  cursor: pointer;
}

.modbus-register-table__grid :deep(.modbus-register-table__row--selected td) {
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
  font-size: 13px;
  line-height: 18px;
}

.modbus-register-table__name span {
  overflow: hidden;
  color: var(--dc-text-muted);
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 11px;
}

.modbus-register-table__unit,
.modbus-register-table__type {
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

.modbus-register-table__area {
  min-width: 58px;
  color: var(--dc-primary);
  font-size: 12px;
  font-weight: 700;
}

.modbus-register-table__unit {
  min-width: 30px;
  justify-content: center;
  color: var(--dc-primary);
}

.modbus-register-table__area[data-area='holding_register'] {
  color: var(--dc-primary);
}

.modbus-register-table__area[data-area='coil'] {
  color: #15803d;
}

.modbus-register-table__address {
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 6px;
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

.modbus-register-table__collect,
.modbus-register-table__value {
  display: grid;
  gap: 1px;
}

.modbus-register-table__collect strong,
.modbus-register-table__value strong {
  overflow: hidden;
  color: var(--dc-text);
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 12px;
  line-height: 16px;
}

.modbus-register-table__collect span,
.modbus-register-table__value span {
  color: var(--dc-success);
  font-size: 11px;
}

.modbus-register-table__collect span.is-muted,
.modbus-register-table__value span {
  color: var(--dc-text-muted);
}

.modbus-register-table__pagination {
  min-height: 42px;
  display: flex;
  align-items: center;
  justify-content: flex-end;
  padding: 6px 12px;
  border-top: 1px solid var(--dc-border);
  background: var(--dc-surface-raised);
}
</style>
