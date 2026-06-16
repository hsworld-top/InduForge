<template>
  <DcDialog
    v-model="visible"
    title="变量预览"
    width="920px"
    body-max-height="calc(100vh - 180px)"
    class="opcua-preview-dialog"
  >
    <div class="opcua-preview">
      <div class="opcua-preview__summary">
        <span>{{ scopeLabel || '全部变量' }}</span>
        <strong>{{ nodes.length }} 个变量</strong>
      </div>

      <div v-if="diagnostics.length" class="opcua-preview__diagnostics">
        <span v-for="item in diagnostics" :key="item">{{ item }}</span>
      </div>

      <el-table
        class="opcua-preview__table"
        :data="pagedNodes"
        height="420"
        size="small"
        empty-text="当前范围暂无预览值"
      >
        <el-table-column prop="nodeId" label="NodeId" min-width="260" show-overflow-tooltip />
        <el-table-column label="当前值" min-width="140" show-overflow-tooltip>
          <template #default="{ row }">{{ formatValue(row.value) }}</template>
        </el-table-column>
        <el-table-column prop="dataType" label="类型" width="110" />
        <el-table-column label="质量" width="96">
          <template #default="{ row }">
            <el-tag size="small" :type="isGoodQuality(row.quality) ? 'success' : 'info'">
              {{ formatQuality(row.quality) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="设备时间" min-width="160" show-overflow-tooltip>
          <template #default="{ row }">{{ formatTime(row.sourceTimestamp) }}</template>
        </el-table-column>
        <el-table-column label="服务时间" min-width="160" show-overflow-tooltip>
          <template #default="{ row }">{{ formatTime(row.serverTimestamp) }}</template>
        </el-table-column>
        <el-table-column label="诊断" min-width="130" show-overflow-tooltip>
          <template #default="{ row }">{{ row.error || '-' }}</template>
        </el-table-column>
      </el-table>

      <div class="opcua-preview__pagination">
        <el-pagination
          v-model:current-page="page"
          v-model:page-size="pageSize"
          :page-sizes="[10, 20, 50, 100]"
          :total="nodes.length"
          background
          layout="total, sizes, prev, pager, next"
          small
        />
      </div>
    </div>
  </DcDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import dayjs from 'dayjs'
import { TIME_FORMAT } from '@/constants'
import DcDialog from '@/components/shared/DcDialog.vue'
import type { OpcuaReadValue } from './types'

const props = defineProps<{
  modelValue: boolean
  nodes: OpcuaReadValue[]
  diagnostics: string[]
  scopeLabel?: string
}>()

const emit = defineEmits<{
  (event: 'update:modelValue', value: boolean): void
}>()

const page = ref(1)
const pageSize = ref(10)

const visible = computed({
  get: () => props.modelValue,
  set: (value: boolean) => emit('update:modelValue', value),
})

const pagedNodes = computed(() => {
  const start = (page.value - 1) * pageSize.value
  return props.nodes.slice(start, start + pageSize.value)
})

watch(
  () => [props.modelValue, props.nodes.length],
  () => {
    page.value = 1
  },
)

const formatValue = (value: unknown) => {
  if (value === null || value === undefined || value === '') return '-'
  if (typeof value === 'object') return JSON.stringify(value)
  return String(value)
}

const isGoodQuality = (quality?: string) => String(quality || '').toLowerCase() === 'good'

const formatQuality = (quality?: string) => {
  if (!quality) return '未知'
  if (isGoodQuality(quality)) return '正常'
  return quality
}

const formatTime = (value?: string) => {
  if (!value) return '-'
  const parsed = dayjs(value)
  return parsed.isValid() ? parsed.format(TIME_FORMAT) : value
}
</script>

<style scoped>
.opcua-preview {
  min-height: 0;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.opcua-preview__summary {
  min-height: 34px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 8px 10px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-subtle);
}

.opcua-preview__summary span,
.opcua-preview__summary strong {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.opcua-preview__summary span {
  color: var(--dc-text-secondary);
  font-size: 12px;
}

.opcua-preview__summary strong {
  color: var(--dc-text);
  font-size: 13px;
}

.opcua-preview__diagnostics {
  display: grid;
  gap: 4px;
  padding: 8px 10px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-muted);
  color: var(--dc-text-muted);
  font-size: 12px;
}

.opcua-preview__table {
  width: 100%;
}

.opcua-preview__table :deep(.el-table__header th) {
  background: var(--dc-surface-subtle);
  color: var(--dc-text-secondary);
  font-weight: 700;
}

.opcua-preview__pagination {
  display: flex;
  justify-content: flex-end;
  padding-top: 2px;
}

:global(.opcua-preview-dialog .el-dialog__body) {
  overflow: hidden;
}
</style>
