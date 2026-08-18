<template>
  <DcDialog
    v-model="visible"
    class="alarm-point-manager-dialog"
    :title="title"
    width="980px"
    body-max-height="680px"
  >
    <div class="alarm-point-manager">
      <section class="alarm-point-manager__available">
        <header class="alarm-point-manager__section-head">
          <div>
            <h3>可选数据点</h3>
            <span v-if="lockedCategory">已限定为{{ categoryLabel(lockedCategory) }}</span>
          </div>
          <span>共 {{ total }} 条</span>
        </header>

        <div class="alarm-point-manager__toolbar">
          <el-input
            v-model="search"
            size="small"
            clearable
            placeholder="搜索名称或路径"
            :prefix-icon="Search"
            @keyup.enter="reload"
          />
          <el-select v-model="status" size="small" placeholder="全部状态" @change="reload">
            <el-option label="全部状态" value="" />
            <el-option label="正常" value="active" />
            <el-option label="停用" value="inactive" />
            <el-option label="异常" value="error" />
            <el-option label="未知" value="unknown" />
          </el-select>
          <button type="button" class="alarm-point-manager__search" @click="reload">
            <IconTablerSearch />搜索
          </button>
        </div>

        <div v-loading="loading" class="alarm-point-manager__table-wrap">
          <el-table
            :data="options"
            row-key="id"
            height="100%"
            class="alarm-point-manager__table"
            @row-dblclick="toggleDatapoint"
          >
            <el-table-column width="46" align="center">
              <template #default="{ row }">
                <el-checkbox
                  :model-value="isSelected(row)"
                  :disabled="!isSelected(row) && !isCompatible(row)"
                  :aria-label="`${isSelected(row) ? '移除' : '选择'} ${row.name}`"
                  @change="toggleDatapoint(row)"
                />
              </template>
            </el-table-column>
            <el-table-column label="名称" min-width="150">
              <template #default="{ row }">
                <div class="alarm-point-manager__identity">
                  <strong>{{ row.name }}</strong>
                  <small>{{ row.path }}</small>
                </div>
              </template>
            </el-table-column>
            <el-table-column label="类型" width="86">
              <template #default="{ row }">{{ row.dataType || '-' }}</template>
            </el-table-column>
            <el-table-column label="状态" width="78">
              <template #default="{ row }">
                <StatusBadge
                  :tone="row.status === 'active' ? 'success' : 'muted'"
                  :text="statusLabel(row.status)"
                />
              </template>
            </el-table-column>
          </el-table>
        </div>

        <div class="alarm-point-manager__pager">
          <span>{{ page }} / {{ totalPages }}</span>
          <button type="button" :disabled="page <= 1 || loading" @click="changePage(page - 1)">
            <IconTablerChevronLeft />
          </button>
          <button
            type="button"
            :disabled="page >= totalPages || loading"
            @click="changePage(page + 1)"
          >
            <IconTablerChevronRight />
          </button>
        </div>
      </section>

      <section class="alarm-point-manager__selected">
        <header class="alarm-point-manager__section-head">
          <div>
            <h3>已选数据点</h3>
            <span>{{ draftBindings.length }} 个</span>
          </div>
          <button
            v-if="draftBindings.length"
            type="button"
            class="alarm-point-manager__clear"
            @click="clearSelection"
          >
            清空
          </button>
        </header>

        <el-input
          v-model="selectedSearch"
          size="small"
          clearable
          placeholder="筛选已选数据点"
          :prefix-icon="Search"
        />

        <div v-if="selectedPageItems.length" class="alarm-point-manager__selected-list">
          <article v-for="binding in selectedPageItems" :key="binding.datapointId">
            <div class="alarm-point-manager__identity">
              <strong>{{ binding.name || binding.path }}</strong>
              <small>{{ binding.path }}</small>
            </div>
            <input
              v-if="mode === 'derived'"
              v-model="binding.inputKey"
              aria-label="输入变量名"
              placeholder="变量名"
            />
            <button
              type="button"
              :aria-label="`移除 ${binding.name || binding.path}`"
              title="移除数据点"
              @click="removeBinding(binding.datapointId)"
            >
              <IconTablerX />
            </button>
          </article>
        </div>
        <el-empty v-else :image-size="44" description="暂无已选数据点" />

        <div v-if="selectedTotalPages > 1" class="alarm-point-manager__pager">
          <span>{{ selectedPage }} / {{ selectedTotalPages }}</span>
          <button type="button" :disabled="selectedPage <= 1" @click="selectedPage -= 1">
            <IconTablerChevronLeft />
          </button>
          <button
            type="button"
            :disabled="selectedPage >= selectedTotalPages"
            @click="selectedPage += 1"
          >
            <IconTablerChevronRight />
          </button>
        </div>
      </section>
    </div>

    <template #footer>
      <div class="alarm-point-manager__footer-actions">
        <button type="button" class="alarm-point-manager__cancel" @click="visible = false">
          取消
        </button>
        <button type="button" class="alarm-point-manager__apply" @click="applySelection">
          <IconTablerCheck />应用选择（{{ draftBindings.length }}）
        </button>
      </div>
    </template>
  </DcDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { Search } from '@element-plus/icons-vue'
import IconTablerCheck from '~icons/tabler/check'
import IconTablerChevronLeft from '~icons/tabler/chevron-left'
import IconTablerChevronRight from '~icons/tabler/chevron-right'
import IconTablerSearch from '~icons/tabler/search'
import IconTablerX from '~icons/tabler/x'
import DcDialog from '@/components/shared/DcDialog.vue'
import StatusBadge from '@/components/shared/StatusBadge.vue'
import { getDatapoints } from '@/api/datapoint.api'
import type { Datapoint } from '@/api/schemas/datapoint.schema'
import type { AlarmBinding, AlarmPolicyMode } from '@/api/schemas/alarm.schema'
import {
  alarmPointCategory,
  datapointToAlarmBinding,
  type AlarmPointCategory,
} from '@/models/alarm-policy'
import { getApiErrorMessage } from '@/utils/request'

const props = withDefaults(
  defineProps<{
    modelValue: boolean
    projectId: string
    mode?: AlarmPolicyMode
    bindings?: AlarmBinding[]
  }>(),
  { mode: 'per_target', bindings: () => [] },
)

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  apply: [bindings: AlarmBinding[]]
}>()

const visible = computed({
  get: () => props.modelValue,
  set: (value: boolean) => emit('update:modelValue', value),
})
const title = computed(() => (props.mode === 'derived' ? '管理组合输入点' : '管理报警数据点'))
const draftBindings = ref<AlarmBinding[]>([])
const options = ref<Datapoint[]>([])
const loading = ref(false)
const search = ref('')
const status = ref('')
const page = ref(1)
const pageSize = 12
const total = ref(0)
const selectedSearch = ref('')
const selectedPage = ref(1)
const selectedPageSize = 8

const totalPages = computed(() => Math.max(1, Math.ceil(total.value / pageSize)))
const selectedIds = computed(() => new Set(draftBindings.value.map((item) => item.datapointId)))
const lockedCategory = computed<AlarmPointCategory | null>(() => {
  if (props.mode === 'derived' || !draftBindings.value.length) return null
  return alarmPointCategory(draftBindings.value[0]?.dataType)
})
const filteredSelected = computed(() => {
  const keyword = selectedSearch.value.trim().toLowerCase()
  if (!keyword) return draftBindings.value
  return draftBindings.value.filter((item) =>
    `${item.name} ${item.path}`.toLowerCase().includes(keyword),
  )
})
const selectedTotalPages = computed(() =>
  Math.max(1, Math.ceil(filteredSelected.value.length / selectedPageSize)),
)
const selectedPageItems = computed(() => {
  const offset = (selectedPage.value - 1) * selectedPageSize
  return filteredSelected.value.slice(offset, offset + selectedPageSize)
})

watch(
  () => props.modelValue,
  async (opened) => {
    if (!opened) return
    draftBindings.value = props.bindings.map((item) => ({ ...item }))
    selectedSearch.value = ''
    selectedPage.value = 1
    await reload()
  },
)
watch([selectedSearch, () => draftBindings.value.length], () => {
  selectedPage.value = Math.min(selectedPage.value, selectedTotalPages.value)
})

async function reload() {
  page.value = 1
  await loadDatapoints()
}

async function loadDatapoints() {
  if (!props.projectId) return
  loading.value = true
  try {
    const result = await getDatapoints(props.projectId, {
      page: page.value,
      pageSize,
      search: search.value.trim(),
      status: status.value || undefined,
    })
    options.value = result.list
    total.value = Number(result.pagination?.total ?? result.list.length)
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '加载数据点失败'))
  } finally {
    loading.value = false
  }
}

async function changePage(next: number) {
  const target = Math.min(Math.max(1, next), totalPages.value)
  if (target === page.value) return
  page.value = target
  await loadDatapoints()
}

function isSelected(datapoint: Datapoint) {
  return selectedIds.value.has(String(datapoint.id))
}

function isCompatible(datapoint: Datapoint) {
  return !lockedCategory.value || alarmPointCategory(datapoint.dataType) === lockedCategory.value
}

function toggleDatapoint(datapoint: Datapoint) {
  const id = String(datapoint.id)
  if (selectedIds.value.has(id)) {
    removeBinding(id)
    return
  }
  if (!isCompatible(datapoint)) {
    ElMessage.warning(`普通报警当前只能选择${categoryLabel(lockedCategory.value)}数据点`)
    return
  }
  draftBindings.value.push(
    datapointToAlarmBinding(datapoint, props.mode, draftBindings.value.length),
  )
  selectedPage.value = selectedTotalPages.value
}

function removeBinding(datapointId: string) {
  draftBindings.value = draftBindings.value.filter((item) => item.datapointId !== datapointId)
}

function clearSelection() {
  draftBindings.value = []
}

function applySelection() {
  emit(
    'apply',
    draftBindings.value.map((item, index) => ({
      ...item,
      role: props.mode === 'derived' ? 'input' : 'target',
      inputKey: props.mode === 'derived' ? item.inputKey?.trim() || `v${index + 1}` : null,
    })),
  )
  visible.value = false
}

function categoryLabel(category: AlarmPointCategory | null) {
  return (
    (
      { number: '数值类型', boolean: '布尔类型', text: '文本类型', structured: '结构类型' } as const
    )[category || 'text'] || '同类'
  )
}

function statusLabel(value?: string) {
  return (
    { active: '正常', inactive: '停用', error: '异常', unknown: '未知' } as Record<string, string>
  )[value || 'unknown']
}
</script>

<style scoped>
:global(.alarm-point-manager-dialog .el-dialog__body) {
  overflow: hidden;
}
.alarm-point-manager {
  height: min(500px, calc(100vh - 220px));
  min-height: 360px;
  display: grid;
  grid-template-columns: minmax(0, 1.25fr) minmax(300px, 0.75fr);
  overflow: hidden;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
}
.alarm-point-manager__available,
.alarm-point-manager__selected {
  min-width: 0;
  min-height: 0;
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 16px;
}
.alarm-point-manager__selected {
  border-left: 1px solid var(--dc-border);
  background: var(--dc-surface-muted);
}
.alarm-point-manager__section-head {
  min-height: 32px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}
.alarm-point-manager__section-head > div {
  min-width: 0;
  display: flex;
  align-items: baseline;
  gap: 8px;
}
.alarm-point-manager__section-head h3 {
  margin: 0;
  font-size: 14px;
  letter-spacing: 0;
}
.alarm-point-manager__section-head span {
  color: var(--dc-text-muted);
  font-size: 11px;
  white-space: nowrap;
}
.alarm-point-manager__toolbar {
  display: grid;
  grid-template-columns: minmax(160px, 1fr) 110px auto;
  align-items: center;
  gap: 8px;
}
.alarm-point-manager__toolbar :deep(.el-input__wrapper),
.alarm-point-manager__toolbar :deep(.el-select__wrapper) {
  min-height: 32px;
  border-radius: var(--dc-radius-sm);
  box-sizing: border-box;
}
.alarm-point-manager__search,
.alarm-point-manager__clear,
.alarm-point-manager__pager button,
.alarm-point-manager__selected-list article > button,
.alarm-point-manager__cancel,
.alarm-point-manager__apply {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 5px;
  border-radius: var(--dc-radius-sm);
  font-family: inherit;
}
.alarm-point-manager__search {
  height: 32px;
  min-width: 68px;
  padding: 0 10px;
  border: 1px solid var(--dc-border);
  background: var(--dc-surface-raised);
  color: var(--dc-text-secondary);
  font-size: 12px;
  font-weight: 600;
  line-height: 1;
}
.alarm-point-manager__search:hover {
  border-color: color-mix(in oklch, var(--dc-primary) 24%, var(--dc-border));
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
}
.alarm-point-manager__search svg {
  width: 14px;
  height: 14px;
  flex: 0 0 14px;
}
.alarm-point-manager__table-wrap {
  min-height: 0;
  flex: 1;
  overflow: hidden;
}
.alarm-point-manager__table {
  height: 100%;
}
.alarm-point-manager__table :deep(.el-table__inner-wrapper::before) {
  display: none;
}
.alarm-point-manager__table :deep(.el-table__header th) {
  height: 42px;
  background: var(--dc-surface-raised);
  color: var(--dc-text);
  font-size: 12px;
  font-weight: 700;
}
.alarm-point-manager__table :deep(.el-table__cell) {
  padding: 7px 0;
}
.alarm-point-manager__identity {
  min-width: 0;
  display: grid;
  gap: 3px;
}
.alarm-point-manager__identity strong,
.alarm-point-manager__identity small {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.alarm-point-manager__identity strong {
  color: var(--dc-text);
  font-size: 12px;
  letter-spacing: 0;
}
.alarm-point-manager__identity small {
  color: var(--dc-text-muted);
  font-size: 10px;
}
.alarm-point-manager__pager {
  min-height: 32px;
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 6px;
}
.alarm-point-manager__pager span {
  margin-right: 2px;
  color: var(--dc-text-muted);
  font-size: 11px;
  line-height: 1;
}
.alarm-point-manager__pager button,
.alarm-point-manager__selected-list article > button {
  width: 30px;
  height: 30px;
  border: 1px solid var(--dc-border);
  background: var(--dc-surface-raised);
  color: var(--dc-text-secondary);
  line-height: 1;
  vertical-align: middle;
}
.alarm-point-manager__pager button svg,
.alarm-point-manager__selected-list article > button svg {
  width: 15px;
  height: 15px;
}
.alarm-point-manager__pager button:disabled {
  opacity: 0.4;
}
.alarm-point-manager__selected-list {
  min-height: 0;
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 7px;
}
.alarm-point-manager__selected-list article {
  min-height: 48px;
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: center;
  gap: 8px;
  padding: 7px 8px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
}
.alarm-point-manager__selected-list article:has(input) {
  grid-template-columns: minmax(0, 1fr) 92px auto;
}
.alarm-point-manager__selected-list input {
  width: 92px;
  height: 28px;
  padding: 0 7px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
  color: var(--dc-text);
}
.alarm-point-manager__selected-list article > button:hover,
.alarm-point-manager__pager button:not(:disabled):hover {
  border-color: color-mix(in oklch, var(--dc-primary) 24%, var(--dc-border));
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
}
.alarm-point-manager__clear {
  height: 28px;
  padding: 0 8px;
  border: 0;
  background: transparent;
  color: var(--dc-text-muted);
  font-size: 12px;
}
.alarm-point-manager__clear:hover {
  color: var(--el-color-danger);
}
.alarm-point-manager__cancel,
.alarm-point-manager__apply {
  height: 34px;
  min-width: 72px;
  padding: 0 14px;
  box-sizing: border-box;
  font-size: 13px;
  font-weight: 600;
  line-height: 1;
  vertical-align: middle;
}
.alarm-point-manager__cancel {
  border: 1px solid var(--dc-border);
  background: var(--dc-surface-raised);
  color: var(--dc-text-secondary);
}
.alarm-point-manager__apply {
  min-width: 144px;
  border: 1px solid var(--dc-primary);
  background: var(--dc-primary);
  color: white;
}
.alarm-point-manager__apply svg {
  width: 15px;
  height: 15px;
  flex: 0 0 15px;
}
.alarm-point-manager__footer-actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 8px;
}
.alarm-point-manager button:focus-visible,
.alarm-point-manager input:focus-visible,
.alarm-point-manager__cancel:focus-visible,
.alarm-point-manager__apply:focus-visible {
  outline: none;
  box-shadow: 0 0 0 3px rgba(29, 78, 216, 0.12);
}
@media (max-width: 760px) {
  :global(.alarm-point-manager-dialog .el-dialog__body) {
    overflow-y: auto;
  }
  .alarm-point-manager {
    height: auto;
    min-height: 620px;
    grid-template-columns: 1fr;
    overflow: auto;
  }
  .alarm-point-manager__available,
  .alarm-point-manager__selected {
    min-height: 480px;
  }
  .alarm-point-manager__selected {
    border-top: 1px solid var(--dc-border);
    border-left: 0;
  }
}
</style>
