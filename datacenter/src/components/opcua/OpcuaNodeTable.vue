<template>
  <div class="opcua-table">
    <el-table
      class="opcua-table__grid"
      :data="nodes"
      :loading="loading"
      height="100%"
      size="small"
      row-key="id"
      highlight-current-row
      :current-row-key="selectedNodeId"
      @row-click="(row) => $emit('select', row)"
      @row-contextmenu="(row, _column, event) => $emit('row-contextmenu', event, row)"
      @sort-change="(payload) => $emit('sort-change', payload)"
    >
      <el-table-column
        label="变量名"
        min-width="170"
        show-overflow-tooltip
        prop="name"
        sortable="custom"
      >
        <template #default="{ row }">
          <div class="opcua-table__name">
            <strong>{{ row.name }}</strong>
          </div>
        </template>
      </el-table-column>
      <el-table-column
        label="NodeId"
        min-width="250"
        show-overflow-tooltip
        prop="nodeId"
        sortable="custom"
      >
        <template #default="{ row }">
          <code class="opcua-table__node-id">{{ row.nodeId }}</code>
        </template>
      </el-table-column>
      <el-table-column label="类型" width="104" prop="dataType" sortable="custom">
        <template #default="{ row }">
          <span class="opcua-table__type">{{ row.dataType }}</span>
        </template>
      </el-table-column>
      <el-table-column prop="datapointPath" label="数据点" min-width="210" show-overflow-tooltip>
        <template #default="{ row }">
          <span v-if="row.datapointPath" class="opcua-table__datapoint">{{
            row.datapointPath
          }}</span>
          <span v-else class="opcua-table__muted">待生成</span>
        </template>
      </el-table-column>
      <el-table-column label="最近值" min-width="110" show-overflow-tooltip>
        <template #default="{ row }">{{ formatValue(row.lastValue) }}</template>
      </el-table-column>
      <el-table-column label="质量" width="86">
        <template #default="{ row }">
          <el-tag
            class="opcua-table__tag"
            size="small"
            :type="isGoodQuality(row.quality) ? 'success' : 'info'"
          >
            {{ formatQuality(row.quality) }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="状态" width="86" prop="status" sortable="custom">
        <template #default="{ row }">
          <el-tag
            class="opcua-table__tag"
            size="small"
            :type="row.status === 'active' ? 'success' : 'warning'"
          >
            {{ row.status || 'active' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="160" fixed="right">
        <template #default="{ row }">
          <div class="opcua-table__actions">
            <el-tooltip content="查看详情" placement="top">
              <button
                type="button"
                class="opcua-table__icon-action"
                @click.stop="$emit('detail', row)"
              >
                <IconTablerEye />
              </button>
            </el-tooltip>
            <el-tooltip content="复制为新变量" placement="top">
              <button
                type="button"
                class="opcua-table__icon-action"
                @click.stop="$emit('duplicate', row)"
              >
                <IconTablerCopy />
              </button>
            </el-tooltip>
            <el-tooltip content="编辑" placement="top">
              <button
                type="button"
                class="opcua-table__icon-action"
                @click.stop="$emit('edit', row)"
              >
                <IconTablerEdit />
              </button>
            </el-tooltip>
            <el-tooltip content="删除" placement="top">
              <button
                type="button"
                class="opcua-table__icon-action is-danger"
                @click.stop="$emit('delete', row)"
              >
                <IconTablerTrash />
              </button>
            </el-tooltip>
          </div>
        </template>
      </el-table-column>
    </el-table>
    <div class="opcua-table__pagination">
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
import IconTablerCopy from '~icons/tabler/copy'
import IconTablerEdit from '~icons/tabler/edit'
import IconTablerEye from '~icons/tabler/eye'
import IconTablerTrash from '~icons/tabler/trash'
import type { OpcuaNode } from './types'

defineProps<{
  nodes: OpcuaNode[]
  loading: boolean
  selectedNodeId: string
  page: number
  pageSize: number
  total: number
}>()

defineEmits<{
  (event: 'select', node: OpcuaNode): void
  (event: 'detail', node: OpcuaNode): void
  (event: 'duplicate', node: OpcuaNode): void
  (event: 'edit', node: OpcuaNode): void
  (event: 'delete', node: OpcuaNode): void
  (event: 'row-contextmenu', mouseEvent: MouseEvent, node: OpcuaNode): void
  (event: 'page-change', page: number): void
  (event: 'page-size-change', pageSize: number): void
  (event: 'sort-change', payload: { prop?: string; order?: string | null }): void
}>()

const formatValue = (value: unknown) => {
  if (value === null || value === undefined || value === '') return '-'
  if (typeof value === 'object') return JSON.stringify(value)
  return String(value)
}

const isGoodQuality = (quality?: string) => String(quality || '').toLowerCase() === 'good'

const formatQuality = (quality?: string) => {
  if (!quality) return 'unknown'
  if (isGoodQuality(quality)) return 'Good'
  return quality
}
</script>

<style scoped>
.opcua-table {
  height: 100%;
  min-height: 0;
  display: flex;
  flex-direction: column;
  background: var(--dc-surface-raised);
}

.opcua-table__grid {
  flex: 1;
  min-height: 0;
}

.opcua-table :deep(.el-table) {
  --el-table-border-color: var(--dc-border);
  --el-table-header-bg-color: var(--dc-surface-subtle);
  --el-table-header-text-color: var(--dc-text-secondary);
  color: var(--dc-text);
  font-size: 12px;
}

.opcua-table :deep(.el-table__header th) {
  font-weight: 700;
}

.opcua-table :deep(.el-table__row) {
  cursor: pointer;
}

.opcua-table__name {
  min-width: 0;
  display: grid;
  gap: 2px;
}

.opcua-table__name strong {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  color: var(--dc-text);
  font-size: 13px;
  line-height: 17px;
  white-space: nowrap;
}

.opcua-table__node-id {
  max-width: 100%;
  display: inline-block;
  padding: 2px 6px;
  border-radius: var(--dc-radius-xs);
  background: var(--dc-surface-muted);
  color: var(--dc-text-secondary);
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: 11px;
  line-height: 16px;
  overflow: hidden;
  text-overflow: ellipsis;
  vertical-align: middle;
  white-space: nowrap;
}

.opcua-table__type {
  font-weight: 700;
  color: var(--dc-text-secondary);
}

.opcua-table__datapoint {
  color: var(--dc-primary);
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: 11px;
}

.opcua-table__tag {
  min-width: 62px;
  justify-content: center;
}

.opcua-table__actions {
  display: inline-flex;
  align-items: center;
  gap: 4px;
}

.opcua-table__icon-action {
  width: 26px;
  height: 26px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
  color: var(--dc-text-secondary);
  cursor: pointer;
}

.opcua-table__icon-action:hover {
  border-color: var(--dc-primary);
  color: var(--dc-primary);
}

.opcua-table__icon-action.is-danger:hover {
  border-color: var(--dc-danger);
  color: var(--dc-danger);
}

.opcua-table__icon-action svg {
  width: 14px;
  height: 14px;
}

.opcua-table__muted {
  color: var(--dc-text-muted);
}

.opcua-table__pagination {
  min-height: 42px;
  display: flex;
  align-items: center;
  justify-content: flex-end;
  padding: 6px 12px;
  border-top: 1px solid var(--dc-border);
  background: var(--dc-surface-raised);
}
</style>
