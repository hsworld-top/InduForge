<template>
  <DcDialog v-model="visible" :title="title" width="800px" class="datapoint-picker">
    <section class="datapoint-picker__panel">
      <header class="datapoint-picker__head">
        <div class="datapoint-picker__summary">
          <strong>可选数据点</strong>
          <span>共 {{ total }} 条</span>
          <span v-if="selected.size" class="is-selected">已选择 {{ selected.size }} 个</span>
        </div>
        <div class="datapoint-picker__filters">
          <el-input v-model="search" clearable placeholder="搜索名称或路径" @keyup.enter="reload">
            <template #prefix><IconTablerSearch /></template>
          </el-input>
          <el-select v-model="status" clearable placeholder="全部状态" @change="reload">
            <el-option label="正常" value="active" />
            <el-option label="停用" value="inactive" />
            <el-option label="失效" value="invalid" />
            <el-option label="异常" value="error" />
          </el-select>
        </div>
      </header>

      <div v-loading="loading" class="datapoint-picker__table-scroll">
        <el-table
          :data="options"
          row-key="id"
          height="360"
          :row-class-name="rowClassName"
          @row-click="togglePoint"
        >
          <el-table-column width="44">
            <template #default="{ row }">
              <el-checkbox
                :model-value="selected.has(String(row.id))"
                :disabled="isPointDisabled(row)"
                :title="pointDisabledReason(row)"
                @click.stop
                @change="togglePoint(row)"
              />
            </template>
          </el-table-column>
          <el-table-column label="名称" min-width="260" show-overflow-tooltip>
            <template #default="{ row }">
              <div class="datapoint-picker__name">
                <strong>{{ row.name || row.path }}</strong>
                <span>{{ row.path }}</span>
                <small v-if="pointDisabledReason(row)">{{ pointDisabledReason(row) }}</small>
              </div>
            </template>
          </el-table-column>
          <el-table-column prop="dataType" label="类型" width="120" />
          <el-table-column label="状态" width="90">
            <template #default="{ row }">
              <span :class="{ 'is-invalid-status': isInvalidPoint(row) }">
                {{ statusLabel(row.status) }}
              </span>
            </template>
          </el-table-column>
        </el-table>
      </div>

      <DataCenterPagination
        :page="page"
        :page-size="pageSize"
        :total="total"
        :total-pages="totalPages"
        :page-size-options="[12, 20, 50]"
        @change="changePage"
      />
    </section>

    <template #footer>
      <div class="datapoint-picker__footer">
        <slot name="warning" :points="selectedPoints" />
        <span class="datapoint-picker__footer-spacer" />
        <button type="button" class="dc-button" @click="visible = false">取消</button>
        <button
          type="button"
          class="dc-button dc-button--primary"
          :disabled="!selected.size || selectionDisabled"
          @click="apply"
        >
          {{ confirmText }}{{ selected.size ? `（${selected.size}）` : '' }}
        </button>
      </div>
    </template>
  </DcDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import IconTablerSearch from '~icons/tabler/search'
import { ElMessage } from 'element-plus'
import DcDialog from '@/components/shared/DcDialog.vue'
import DataCenterPagination from '@/components/shared/DataCenterPagination.vue'
import { getDatapoints } from '@/api/datapoint.api'
import type { Datapoint } from '@/api/schemas/datapoint.schema'
import { getApiErrorMessage } from '@/utils/request'

export type DatapointPickerSelection = Pick<Datapoint, 'id' | 'path' | 'name'> &
  Partial<Pick<Datapoint, 'dataType' | 'status' | 'sourceType'>>

const props = withDefaults(
  defineProps<{
    modelValue: boolean
    projectId: string
    title?: string
    confirmText?: string
    multiple?: boolean
    disabled?: boolean
    validateSelection?: (points: DatapointPickerSelection[]) => boolean
    disabledReason?: (point: DatapointPickerSelection) => string
    initialSelection?: DatapointPickerSelection[]
  }>(),
  {
    title: '选择数据点',
    confirmText: '确定',
    multiple: true,
    disabled: false,
    initialSelection: () => [],
  },
)

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  apply: [points: DatapointPickerSelection[]]
  select: [point: DatapointPickerSelection]
}>()
const visible = computed({
  get: () => props.modelValue,
  set: (value: boolean) => emit('update:modelValue', value),
})
const options = ref<Datapoint[]>([])
const selected = ref(new Map<string, DatapointPickerSelection>())
const loading = ref(false)
const search = ref('')
const status = ref('')
const page = ref(1)
const pageSize = ref(12)
const total = ref(0)
const invalidPointHint = '该数据点已失效，仅可在数据点模块查看或清理'
const totalPages = computed(() => (total.value > 0 ? Math.ceil(total.value / pageSize.value) : 0))
const selectedPoints = computed(() => [...selected.value.values()])
const selectionDisabled = computed(
  () =>
    props.disabled ||
    (props.validateSelection ? !props.validateSelection(selectedPoints.value) : false),
)

watch(
  () => props.modelValue,
  (opened) => {
    if (!opened) return
    selected.value = new Map(
      props.initialSelection.map((point) => [String(point.id), { ...point, id: String(point.id) }]),
    )
    page.value = 1
    void loadOptions()
  },
)

async function loadOptions() {
  loading.value = true
  try {
    const result = await getDatapoints(props.projectId, {
      page: page.value,
      pageSize: pageSize.value,
      search: search.value || undefined,
      status: status.value || undefined,
    })
    options.value = result.list
    // 编辑已有配置时，使用列表返回的实时状态覆盖旧选择，避免失效点继续被提交。
    const nextSelected = new Map(selected.value)
    result.list.forEach((point) => {
      const id = String(point.id)
      if (nextSelected.has(id)) nextSelected.set(id, { ...nextSelected.get(id), ...point, id })
    })
    selected.value = nextSelected
    total.value = result.pagination.total || 0
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '加载数据点失败'))
  } finally {
    loading.value = false
  }
}
function reload() {
  page.value = 1
  void loadOptions()
}
function changePage(next: { page: number; pageSize: number }) {
  page.value = next.page
  pageSize.value = next.pageSize
  void loadOptions()
}
function togglePoint(point: DatapointPickerSelection) {
  const reason = pointDisabledReason(point)
  if (reason) {
    ElMessage.warning(reason)
    return
  }
  const id = String(point.id)
  const next = new Map(selected.value)
  if (next.has(id)) next.delete(id)
  else {
    if (!props.multiple) next.clear()
    next.set(id, { ...point, id })
  }
  selected.value = next
}
function rowClassName({ row }: { row: Datapoint }) {
  return [
    selected.value.has(String(row.id)) ? 'is-selected' : '',
    isPointDisabled(row) ? 'is-invalid' : '',
  ]
    .filter(Boolean)
    .join(' ')
}
function apply() {
  const points = selectedPoints.value.map((point) => ({ ...point }))
  const disabled = points.find((point) => pointDisabledReason(point))
  if (disabled) {
    ElMessage.warning(pointDisabledReason(disabled))
    return
  }
  if (!points.length || selectionDisabled.value) return
  emit('apply', points)
  if (!props.multiple) emit('select', points[0])
  visible.value = false
}

function isInvalidPoint(point: Pick<DatapointPickerSelection, 'status'>) {
  return point.status === 'invalid'
}

function pointDisabledReason(point: DatapointPickerSelection) {
  if (isInvalidPoint(point)) return invalidPointHint
  return props.disabledReason?.(point) || ''
}

function isPointDisabled(point: DatapointPickerSelection) {
  return Boolean(pointDisabledReason(point))
}

function statusLabel(value?: string) {
  return (
    { active: '正常', inactive: '停用', invalid: '失效', error: '异常' }[value || ''] ||
    value ||
    '-'
  )
}
</script>

<style scoped>
.datapoint-picker__panel {
  min-width: 0;
  overflow: hidden;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-md);
  background: var(--dc-surface);
}
.datapoint-picker__head {
  min-height: 58px;
  padding: 10px 12px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  border-bottom: 1px solid var(--dc-border);
}
.datapoint-picker__summary {
  display: flex;
  align-items: baseline;
  gap: 8px;
  white-space: nowrap;
}
.datapoint-picker__summary span {
  color: var(--dc-text-secondary);
  font-size: 12px;
}
.datapoint-picker__summary .is-selected {
  color: var(--dc-primary);
  font-weight: 700;
}
.datapoint-picker__filters {
  display: grid;
  grid-template-columns: minmax(170px, 1fr) 120px;
  gap: 8px;
  width: min(410px, 70%);
}
.datapoint-picker__table-scroll {
  min-width: 0;
  overflow-x: auto;
}
.datapoint-picker__name {
  display: grid;
  min-width: 0;
}
.datapoint-picker__name strong,
.datapoint-picker__name span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.datapoint-picker__name span {
  color: var(--dc-text-secondary);
  font-size: 11px;
}
.datapoint-picker__name small {
  color: var(--dc-text-muted);
  font-size: 11px;
}
.datapoint-picker__table-scroll :deep(.el-table__row.is-invalid > td.el-table__cell),
.datapoint-picker__table-scroll :deep(.el-table__row.is-invalid:hover > td.el-table__cell) {
  background: var(--dc-surface-muted);
  color: var(--dc-text-muted);
  cursor: not-allowed;
}
.is-invalid-status {
  color: var(--el-color-danger);
}
.datapoint-picker__table-scroll :deep(.el-table__row.is-selected > td.el-table__cell),
.datapoint-picker__table-scroll :deep(.el-table__row.is-selected:hover > td.el-table__cell) {
  background: var(--dc-primary-soft);
}
.datapoint-picker__footer {
  width: 100%;
  display: flex;
  align-items: center;
  gap: 8px;
}
.datapoint-picker__footer-spacer {
  flex: 1;
}
@media (max-width: 640px) {
  .datapoint-picker__head {
    align-items: stretch;
    flex-direction: column;
  }
  .datapoint-picker__filters {
    width: 100%;
  }
}
</style>
