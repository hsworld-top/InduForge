<template>
  <DcDialog v-model="visible" :title="title" width="760px" class="alarm-point-picker">
    <section class="alarm-point-picker__panel">
      <header>
        <div>
          <strong>可选数据点</strong><span>共 {{ total }} 条</span>
          <span v-if="selected.length" class="alarm-point-picker__selected-count"
            >已选择 {{ selected.length }} 个</span
          >
        </div>
        <div class="alarm-point-picker__filters">
          <el-input v-model="search" clearable placeholder="搜索名称或路径" @keyup.enter="reload">
            <template #prefix><IconTablerSearch /></template>
          </el-input>
          <el-select v-model="status" clearable placeholder="全部状态" @change="reload">
            <el-option label="正常" value="active" /><el-option label="失效" value="inactive" />
          </el-select>
        </div>
      </header>
      <div v-loading="loading" class="alarm-point-picker__table-wrap">
        <el-table
          :data="options"
          row-key="id"
          height="360"
          :row-class-name="rowClassName"
          @row-click="togglePoint"
        >
          <el-table-column width="44">
            <template #default="{ row }"
              ><el-checkbox
                :model-value="selectedIds.has(String(row.id))"
                @click.stop
                @change="togglePoint(row)"
            /></template>
          </el-table-column>
          <el-table-column label="名称" min-width="260" show-overflow-tooltip>
            <template #default="{ row }"
              ><div class="alarm-point-picker__name">
                <strong>{{ row.name || row.path }}</strong
                ><span>{{ row.path }}</span>
              </div></template
            >
          </el-table-column>
          <el-table-column prop="dataType" label="类型" width="110" />
          <el-table-column prop="status" label="状态" width="90" />
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
      <div class="alarm-point-picker__footer">
        <span v-if="incompatible" class="alarm-point-picker__warning">所选数据点类型不兼容</span>
        <span v-else></span>
        <button type="button" class="dc-button" @click="visible = false">取消</button>
        <button
          type="button"
          class="dc-button dc-button--primary"
          :disabled="!selected.length || incompatible"
          @click="apply"
        >
          确定{{ selected.length ? `（${selected.length}）` : '' }}
        </button>
      </div>
    </template>
  </DcDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import IconTablerSearch from '~icons/tabler/search'
import DcDialog from '@/components/shared/DcDialog.vue'
import DataCenterPagination from '@/components/shared/DataCenterPagination.vue'
import { getDatapoints } from '@/api/datapoint.api'
import type { Datapoint } from '@/api/schemas/datapoint.schema'
import type { AlarmItemMode } from '@/api/schemas/alarm.schema'
import {
  compatibleAlarmPoints,
  datapointToAlarmPoint,
  type AlarmPointSelection,
} from '@/models/alarm-policy'
import { ElMessage } from 'element-plus'
import { getApiErrorMessage } from '@/utils/request'

const props = withDefaults(
  defineProps<{
    modelValue: boolean
    projectId: string
    mode?: AlarmItemMode
    points?: AlarmPointSelection[]
  }>(),
  { mode: 'point', points: () => [] },
)
const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  apply: [points: AlarmPointSelection[]]
}>()
const visible = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value),
})
const title = computed(() => (props.mode === 'derived' ? '管理组合输入点' : '选择报警数据点'))
const selected = ref<AlarmPointSelection[]>([])
const options = ref<Datapoint[]>([])
const loading = ref(false)
const search = ref('')
const status = ref('')
const page = ref(1)
const pageSize = ref(12)
const total = ref(0)
const totalPages = computed(() => (total.value > 0 ? Math.ceil(total.value / pageSize.value) : 0))
const selectedIds = computed(() => new Set(selected.value.map((point) => point.datapointId)))
const incompatible = computed(
  () => props.mode === 'point' && !compatibleAlarmPoints(selected.value),
)

watch(
  () => props.modelValue,
  (opened) => {
    if (!opened) return
    selected.value = props.points.map((point) => ({ ...point }))
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
function togglePoint(datapoint: Datapoint) {
  const id = String(datapoint.id)
  const index = selected.value.findIndex((point) => point.datapointId === id)
  if (index >= 0) selected.value.splice(index, 1)
  else selected.value.push(datapointToAlarmPoint(datapoint, props.mode, selected.value.length))
}
function rowClassName({ row }: { row: Datapoint }) {
  return selectedIds.value.has(String(row.id)) ? 'is-selected' : ''
}
function apply() {
  if (!selected.value.length || incompatible.value) return
  emit(
    'apply',
    selected.value.map((point) => ({ ...point })),
  )
  visible.value = false
}
</script>

<style scoped>
.alarm-point-picker__panel {
  min-width: 0;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-md);
  overflow: hidden;
  background: var(--dc-surface);
}
.alarm-point-picker__panel > header {
  min-height: 58px;
  padding: 10px 12px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  border-bottom: 1px solid var(--dc-border);
}
.alarm-point-picker__panel > header div:first-child {
  display: flex;
  align-items: baseline;
  gap: 8px;
  white-space: nowrap;
}
.alarm-point-picker__panel > header span {
  color: var(--dc-text-secondary);
  font-size: 12px;
}
.alarm-point-picker__selected-count {
  color: var(--dc-primary) !important;
  font-weight: 700;
}
.alarm-point-picker__filters {
  display: grid;
  grid-template-columns: minmax(150px, 1fr) 110px;
  gap: 8px;
  width: min(390px, 70%);
}
.alarm-point-picker__name {
  display: grid;
  min-width: 0;
}
.alarm-point-picker__name strong,
.alarm-point-picker__name span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.alarm-point-picker__name span {
  color: var(--dc-text-secondary);
  font-size: 11px;
}
.alarm-point-picker__table-wrap :deep(.el-table__row.is-selected > td.el-table__cell) {
  background: var(--dc-primary-soft);
}
.alarm-point-picker__table-wrap :deep(.el-table__row.is-selected:hover > td.el-table__cell) {
  background: var(--dc-primary-soft);
}
.alarm-point-picker__footer {
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 8px;
}
.alarm-point-picker__warning {
  margin-right: auto;
  color: var(--el-color-warning);
}
@media (max-width: 640px) {
  .alarm-point-picker__panel > header {
    align-items: stretch;
    flex-direction: column;
  }
  .alarm-point-picker__filters {
    width: 100%;
  }
}
</style>
