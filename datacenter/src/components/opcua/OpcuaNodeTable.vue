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
    >
      <el-table-column label="变量名" min-width="170" show-overflow-tooltip>
        <template #default="{ row }">
          <div class="opcua-table__name">
            <strong>{{ row.name }}</strong>
            <span>{{ row.code }}</span>
          </div>
        </template>
      </el-table-column>
      <el-table-column label="NodeId" min-width="250" show-overflow-tooltip>
        <template #default="{ row }">
          <code class="opcua-table__node-id">{{ row.nodeId }}</code>
        </template>
      </el-table-column>
      <el-table-column label="类型" width="104">
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
            :type="row.quality === 'good' ? 'success' : 'info'"
          >
            {{ row.quality || 'unknown' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="状态" width="86">
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
      <el-table-column label="操作" width="112" fixed="right">
        <template #default="{ row }">
          <div class="opcua-table__actions">
            <el-button text size="small" :icon="IconTablerEdit" @click.stop="$emit('edit', row)" />
            <el-button
              text
              size="small"
              :icon="IconTablerTrash"
              @click.stop="$emit('delete', row)"
            />
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
import IconTablerEdit from '~icons/tabler/edit'
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
  (event: 'edit', node: OpcuaNode): void
  (event: 'delete', node: OpcuaNode): void
  (event: 'page-change', page: number): void
  (event: 'page-size-change', pageSize: number): void
}>()

const formatValue = (value: unknown) => {
  if (value === null || value === undefined || value === '') return '-'
  if (typeof value === 'object') return JSON.stringify(value)
  return String(value)
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

.opcua-table__name span {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  color: var(--dc-text-muted);
  font-size: 11px;
  line-height: 14px;
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
  gap: 2px;
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
