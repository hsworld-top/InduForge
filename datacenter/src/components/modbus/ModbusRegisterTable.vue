<template>
  <div class="modbus-register-table">
    <el-table
      ref="tableRef"
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
      @selection-change="handleSelectionChange"
      @sort-change="(payload) => $emit('sort-change', payload)"
    >
      <el-table-column type="selection" width="44" reserve-selection />
      <el-table-column label="变量" min-width="190" prop="name" sortable="custom">
        <template #default="{ row }">
          <span class="modbus-register-table__text">{{ row.name }}</span>
        </template>
      </el-table-column>
      <el-table-column label="分组" min-width="120">
        <template #default="{ row }">
          <span class="modbus-register-table__path">{{ formatGroup(row.groupId) }}</span>
        </template>
      </el-table-column>
      <el-table-column label="数据点" min-width="170">
        <template #default="{ row }">
          <span class="modbus-register-table__path">{{ row.datapointPath || '-' }}</span>
        </template>
      </el-table-column>
      <el-table-column label="最近值" width="116">
        <template #default="{ row }">
          <span class="modbus-register-table__text">{{ formatValue(row.lastValue) }}</span>
        </template>
      </el-table-column>
      <el-table-column label="质量" width="92">
        <template #default="{ row }">
          <el-tag size="small" :type="qualityTagType(row.quality)">
            {{ formatQuality(row.quality) }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="从站" width="84" prop="unitId" sortable="custom">
        <template #default="{ row }">
          <span class="modbus-register-table__text">{{ row.unitId }}</span>
        </template>
      </el-table-column>
      <el-table-column
        label="地址"
        min-width="172"
        prop="address"
        sortable="custom"
        show-overflow-tooltip
      >
        <template #default="{ row }">
          <span class="modbus-register-table__text">{{
            `${formatArea(row.area)} ${row.address} / 协议 ${row.protocolAddress}`
          }}</span>
        </template>
      </el-table-column>
      <el-table-column label="类型" width="88" prop="dataType" sortable="custom">
        <template #default="{ row }">
          <span class="modbus-register-table__text">{{ row.dataType }}</span>
        </template>
      </el-table-column>
      <el-table-column label="采集周期(ms)" width="128" prop="pollIntervalMs" sortable="custom">
        <template #default="{ row }">
          <span class="modbus-register-table__text">{{ row.pollIntervalMs }}</span>
        </template>
      </el-table-column>
      <el-table-column label="读写权限" width="96">
        <template #default="{ row }">
          <span class="modbus-register-table__text">{{ formatAccessLevel(row.accessLevel) }}</span>
        </template>
      </el-table-column>
      <el-table-column label="启用状态" width="96" prop="status" sortable="custom">
        <template #default="{ row }">
          <el-tag size="small" :type="row.status === 'active' ? 'success' : 'info'">
            {{ formatStatus(row.status) }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="132" fixed="right">
        <template #default="{ row }">
          <div class="modbus-register-table__actions">
            <el-tooltip content="查看详情" placement="top">
              <button type="button" aria-label="查看详情" @click.stop="$emit('detail', row)">
                <IconTablerEye />
              </button>
            </el-tooltip>
            <el-tooltip content="复制为新变量" placement="top">
              <button type="button" aria-label="复制为新变量" @click.stop="$emit('duplicate', row)">
                <IconTablerCopy />
              </button>
            </el-tooltip>
            <el-tooltip content="编辑" placement="top">
              <button type="button" aria-label="编辑" @click.stop="$emit('edit', row)">
                <IconTablerPencil />
              </button>
            </el-tooltip>
            <el-tooltip content="删除" placement="top">
              <button
                type="button"
                class="is-danger"
                aria-label="删除"
                @click.stop="$emit('delete', row)"
              >
                <IconTablerTrash />
              </button>
            </el-tooltip>
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
        size="small"
        @current-change="$emit('page-change', $event)"
        @size-change="$emit('page-size-change', $event)"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { nextTick, ref, watch } from 'vue'
import type { TableInstance } from 'element-plus'
import type { ModbusRegister } from './types'
import IconTablerCopy from '~icons/tabler/copy'
import IconTablerEye from '~icons/tabler/eye'
import IconTablerPencil from '~icons/tabler/pencil'
import IconTablerTrash from '~icons/tabler/trash'

const props = defineProps<{
  registers: ModbusRegister[]
  loading?: boolean
  selectedRegisterId?: string
  selectedIds?: string[]
  page: number
  pageSize: number
  total: number
  groupFormatter?: (groupId?: string | null) => string
}>()

const emit = defineEmits<{
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

const tableRef = ref<TableInstance | null>(null)
const syncingSelection = ref(false)

watch(
  () => [props.registers, props.selectedIds] as const,
  () => {
    void syncSelection()
  },
  { deep: true },
)

async function syncSelection() {
  await nextTick()
  const table = tableRef.value
  if (!table) return
  syncingSelection.value = true
  table.clearSelection()
  const selected = new Set(props.selectedIds || [])
  props.registers.forEach((register) => {
    if (selected.has(register.id)) {
      table.toggleRowSelection(register, true)
    }
  })
  await nextTick()
  syncingSelection.value = false
}

function handleSelectionChange(rows: ModbusRegister[]) {
  if (syncingSelection.value) return
  emit('selection-change', rows || [])
}

function clearSelection() {
  tableRef.value?.clearSelection()
}

defineExpose({ clearSelection, syncSelection })

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

const formatAccessLevel = (accessLevel?: string) => {
  const normalized = String(accessLevel || '').toLowerCase()
  return normalized.includes('write') ? '读写' : '只读'
}

const formatQuality = (quality?: string) => {
  const normalized = String(quality || '').toLowerCase()
  if (normalized === 'good') return 'Good'
  if (normalized === 'bad') return 'Bad'
  return 'Unknown'
}

const qualityTagType = (quality?: string) => {
  const normalized = String(quality || '').toLowerCase()
  if (normalized === 'good') return 'success'
  if (normalized === 'bad') return 'danger'
  return 'info'
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

.modbus-register-table__grid :deep(.cell) {
  word-break: normal;
}

.modbus-register-table__grid :deep(th .cell) {
  white-space: nowrap;
}

.modbus-register-table__grid :deep(.modbus-register-table__row--selected td) {
  background: color-mix(in oklch, var(--dc-primary) 8%, var(--dc-surface-raised));
}

.modbus-register-table__text,
.modbus-register-table__path {
  display: inline-block;
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.modbus-register-table__text {
  color: var(--dc-text);
  font-size: 13px;
}

.modbus-register-table__text.is-muted,
.modbus-register-table__path {
  color: var(--dc-text-muted);
}

.modbus-register-table__path {
  font-size: 12px;
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

.modbus-register-table__actions button.is-danger:hover {
  border-color: color-mix(in oklch, var(--dc-danger) 34%, var(--dc-border));
  color: var(--dc-danger);
}

.modbus-register-table__actions svg {
  width: 14px;
  height: 14px;
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
