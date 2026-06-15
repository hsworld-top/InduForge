<template>
  <div class="s7-variable-table">
    <el-table
      class="s7-variable-table__grid"
      :data="variables"
      :loading="loading"
      height="100%"
      row-key="id"
      highlight-current-row
      :row-class-name="rowClassName"
      empty-text="暂无 S7 变量"
      @row-click="(row) => $emit('select', row)"
      @row-contextmenu="(row, _column, event) => $emit('row-contextmenu', event, row)"
      @sort-change="(payload) => $emit('sort-change', payload)"
    >
      <el-table-column label="变量名" min-width="150" prop="name" sortable="custom">
        <template #default="{ row }">
          <div class="s7-variable-table__name">
            <strong :title="row.name">{{ row.name }}</strong>
            <span :title="row.code">{{ row.code }}</span>
          </div>
        </template>
      </el-table-column>
      <el-table-column label="地址" min-width="138" prop="normalizedAddress" sortable="custom">
        <template #default="{ row }">
          <div class="s7-variable-table__address">
            <strong :title="row.normalizedAddress || row.addressText">{{
              row.normalizedAddress || row.addressText
            }}</strong>
            <span :title="`${row.area} ${formatByteRange(row)}`"
              >{{ row.area }} {{ formatByteRange(row) }}</span
            >
          </div>
        </template>
      </el-table-column>
      <el-table-column label="类型" width="84" prop="dataType" sortable="custom">
        <template #default="{ row }">
          <span class="s7-variable-table__pill">{{ row.dataType }}</span>
        </template>
      </el-table-column>
      <el-table-column label="字节范围" width="108">
        <template #default="{ row }">{{ formatByteRange(row) }}</template>
      </el-table-column>
      <el-table-column label="数据点" min-width="180">
        <template #default="{ row }">
          <span class="s7-variable-table__path" :title="row.datapointPath || ''">{{
            row.datapointPath || '-'
          }}</span>
        </template>
      </el-table-column>
      <el-table-column label="最近值" width="96">
        <template #default="{ row }"
          ><span class="s7-variable-table__value" :title="formatValue(row.lastValue)">{{
            formatValue(row.lastValue)
          }}</span></template
        >
      </el-table-column>
      <el-table-column label="质量" width="86">
        <template #default="{ row }">
          <el-tag
            size="small"
            :type="row.quality === 'Good' || row.quality === 'good' ? 'success' : 'info'"
            >{{ row.quality || 'unknown' }}</el-tag
          >
        </template>
      </el-table-column>
      <el-table-column label="周期" width="82" prop="pollIntervalMs" sortable="custom">
        <template #default="{ row }">{{ row.pollIntervalMs }}ms</template>
      </el-table-column>
      <el-table-column label="权限" width="92" prop="accessLevel" sortable="custom">
        <template #default="{ row }">
          <el-tag size="small" :type="row.accessLevel === 'Read' ? 'info' : 'warning'">
            {{ row.accessLevel || 'Read' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="状态" width="86" prop="status" sortable="custom">
        <template #default="{ row }">
          <el-tag size="small" :type="row.status === 'active' ? 'success' : 'info'">{{
            row.status
          }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="92" fixed="right">
        <template #default="{ row }">
          <div class="s7-variable-table__actions">
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
    <div class="s7-variable-table__pagination">
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
import type { S7Variable } from './types'
import IconTablerPencil from '~icons/tabler/pencil'
import IconTablerTrash from '~icons/tabler/trash'

const props = defineProps<{
  variables: S7Variable[]
  loading?: boolean
  selectedVariableId?: string
  page: number
  pageSize: number
  total: number
}>()

defineEmits<{
  (event: 'select', variable: S7Variable): void
  (event: 'edit', variable: S7Variable): void
  (event: 'delete', variable: S7Variable): void
  (event: 'row-contextmenu', mouseEvent: MouseEvent, variable: S7Variable): void
  (event: 'page-change', page: number): void
  (event: 'page-size-change', pageSize: number): void
  (event: 'sort-change', payload: { prop?: string; order?: string | null }): void
}>()

const formatByteRange = (row: S7Variable) => {
  const end = row.byteOffset + row.readLength - 1
  const db =
    row.area === 'DB' && row.dbNumber !== null && row.dbNumber !== undefined
      ? `DB${row.dbNumber} `
      : ''
  return `${db}${row.byteOffset}-${end}`
}
const formatValue = (value: unknown) => {
  if (value === null || value === undefined || value === '') return '-'
  if (typeof value === 'object') return JSON.stringify(value)
  return String(value)
}
const rowClassName = ({ row }: { row: S7Variable }) =>
  row.id === props.selectedVariableId ? 's7-variable-table__row--selected' : ''
</script>

<style scoped>
.s7-variable-table {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
}
.s7-variable-table__grid {
  flex: 1;
  min-height: 0;
}
.s7-variable-table__grid :deep(.el-table__row) {
  cursor: pointer;
}
.s7-variable-table__grid :deep(.s7-variable-table__row--selected td) {
  background: color-mix(in oklch, var(--dc-primary) 8%, var(--dc-surface-raised));
}
.s7-variable-table__name,
.s7-variable-table__address {
  min-width: 0;
  display: grid;
  gap: 2px;
}
.s7-variable-table__name strong,
.s7-variable-table__address strong {
  overflow: hidden;
  color: var(--dc-text);
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 12px;
}
.s7-variable-table__name span,
.s7-variable-table__address span,
.s7-variable-table__path,
.s7-variable-table__value {
  display: block;
  overflow: hidden;
  color: var(--dc-text-muted);
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 11px;
}
.s7-variable-table__pill {
  min-height: 22px;
  display: inline-flex;
  align-items: center;
  padding: 0 7px;
  border: 1px solid var(--dc-border);
  border-radius: 999px;
  background: var(--dc-surface-subtle);
  color: var(--dc-primary);
  font-size: 11px;
  font-weight: 700;
}
.s7-variable-table__actions {
  display: flex;
  align-items: center;
  gap: 4px;
}
.s7-variable-table__actions button {
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
.s7-variable-table__actions button:hover {
  border-color: color-mix(in oklch, var(--dc-primary) 28%, var(--dc-border));
  color: var(--dc-primary);
}
.s7-variable-table__pagination {
  min-height: 42px;
  display: flex;
  align-items: center;
  justify-content: flex-end;
  padding: 6px 12px;
  border-top: 1px solid var(--dc-border);
  background: var(--dc-surface-raised);
}
</style>
