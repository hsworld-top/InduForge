<template>
  <div class="mqtt-batch-mapping">
    <header class="mqtt-batch-mapping__header">
      <div class="mqtt-batch-mapping__header-toolbar">
        <div class="mqtt-batch-mapping__header-actions">
          <el-button type="primary" size="small" @click="addMappingRow">
            <IconTablerPlus class="mqtt-batch-mapping__button-icon" />
            新增
          </el-button>
          <el-button size="small" @click="openBatchGenerateDialog">
            <IconTablerWand class="mqtt-batch-mapping__button-icon" />
            批量生成
          </el-button>
          <el-button
            class="mqtt-batch-mapping__live-action"
            :class="{ 'is-connected': previewSessionId }"
            size="small"
            type="primary"
            plain
            :disabled="!previewSessionId"
            @click="$emit('openMonitor')"
          >
            <IconTablerActivity class="mqtt-batch-mapping__button-icon" />
            变量预览/监控
          </el-button>
          <el-button type="primary" size="small" :loading="saving" @click="saveMappings">
            <IconTablerDeviceFloppy class="mqtt-batch-mapping__button-icon" />
            保存映射
          </el-button>
        </div>
        <div class="mqtt-batch-mapping__filters">
          <el-input
            v-model="searchKeyword"
            placeholder="搜索变量名或标识"
            clearable
            size="small"
            @input="handleSearch"
          >
            <template #prefix>
              <IconTablerSearch class="mqtt-batch-mapping__input-icon" />
            </template>
          </el-input>
          <el-popover
            v-model:visible="sortFieldPopoverVisible"
            placement="bottom-start"
            :width="148"
            trigger="click"
          >
            <template #reference>
              <button type="button" class="mqtt-batch-mapping__sort-pill">
                {{ currentSortFieldLabel }}
              </button>
            </template>
            <div class="mqtt-batch-mapping__sort-menu">
              <button
                v-for="option in sortFieldOptions"
                :key="option.value"
                type="button"
                class="mqtt-batch-mapping__sort-item"
                :class="{ 'is-active': sortBy === option.value }"
                @click="changeSortField(option.value)"
              >
                {{ option.label }}
              </button>
            </div>
          </el-popover>
          <button type="button" class="mqtt-batch-mapping__sort-pill" @click="toggleSortOrder">
            <IconTablerSortDescending
              v-if="sortOrder === 'desc'"
              class="mqtt-batch-mapping__sort-icon"
            />
            <IconTablerSortAscending v-else class="mqtt-batch-mapping__sort-icon" />
            {{ currentSortOrderLabel }}
          </button>
          <el-button size="small" :loading="loading" @click="() => loadTags()">
            <IconTablerRefresh class="mqtt-batch-mapping__button-icon" />
            刷新
          </el-button>
        </div>
      </div>
    </header>

    <section
      ref="bodyRef"
      class="mqtt-batch-mapping__body"
      :style="isCompactLayout ? undefined : bodyGridStyle"
    >
      <div class="mqtt-batch-mapping__result">
        <div class="mqtt-batch-mapping__section-title">
          <strong>变量映射清单</strong>
          <div class="mqtt-batch-mapping__result-meta">
            <span>本页 {{ visibleMappings.length }} / 共 {{ mappingDisplayTotal }} 个变量</span>
            <span v-if="runtimeInfoLoading" class="mqtt-batch-mapping__runtime-loading">
              正在补充最近值和状态...
            </span>
            <el-tooltip
              :content="configCollapsed ? '展开样例消息和拆分规则' : '收起样例消息和拆分规则'"
              placement="top"
            >
              <button
                type="button"
                class="mqtt-batch-mapping__drawer-toggle"
                :class="{ 'is-collapsed': configCollapsed }"
                @click="toggleConfigPanel"
              >
                <IconTablerLayoutSidebarRight />
              </button>
            </el-tooltip>
          </div>
        </div>
        <el-table
          ref="mappingTableRef"
          class="mqtt-batch-mapping__table"
          v-loading="loading"
          element-loading-text="加载变量映射..."
          :data="visibleMappings"
          height="100%"
          row-key="matchName"
          empty-text="填写样例消息后点击解析预览，或手动新增变量"
          @select-all="handleMappingSelectAll"
          @selection-change="handleMappingSelectionChange"
        >
          <el-table-column type="selection" width="42" reserve-selection />
          <el-table-column label="变量名" min-width="150">
            <template #default="{ row }">
              <el-input
                v-if="isEditingMappingCell(row, 'name')"
                v-model="row.name"
                size="small"
                @blur="stopEditingMappingCell"
                @keyup.enter="stopEditingMappingCell"
              />
              <button
                v-else
                type="button"
                class="mqtt-batch-mapping__cell-editor"
                @click="startEditingMappingCell(row, 'name')"
              >
                {{ row.name || row.matchName || '-' }}
              </button>
            </template>
          </el-table-column>
          <el-table-column label="数据类型" width="120">
            <template #default="{ row }">
              <el-select
                v-if="isEditingMappingCell(row, 'dataType')"
                v-model="row.dataType"
                size="small"
                @change="stopEditingMappingCell"
                @visible-change="(visible) => !visible && stopEditingMappingCell()"
              >
                <el-option label="字符串" value="string" />
                <el-option label="数值（float64）" value="float64" />
                <el-option label="布尔" value="bool" />
                <el-option label="对象" value="object" />
                <el-option label="数组" value="array" />
              </el-select>
              <button
                v-else
                type="button"
                class="mqtt-batch-mapping__cell-editor"
                @click="startEditingMappingCell(row, 'dataType')"
              >
                {{ getDataTypeLabel(row.dataType) }}
              </button>
            </template>
          </el-table-column>
          <el-table-column label="最近值" min-width="150" show-overflow-tooltip>
            <template #default="{ row }">
              <span class="mqtt-batch-mapping__value">{{ formatLastValue(row) }}</span>
            </template>
          </el-table-column>
          <el-table-column label="质量" width="92" show-overflow-tooltip>
            <template #default="{ row }">
              <span class="mqtt-batch-mapping__mono">{{ formatQualityLabel(row) }}</span>
            </template>
          </el-table-column>
          <el-table-column label="时间" min-width="150" show-overflow-tooltip>
            <template #default="{ row }">
              <span class="mqtt-batch-mapping__mono">{{ formatLastTime(row) }}</span>
            </template>
          </el-table-column>
          <el-table-column label="状态" width="92">
            <template #default="{ row }">
              <span class="mqtt-batch-mapping__status" :class="statusClass(row)">
                {{ statusLabel(row) }}
              </span>
            </template>
          </el-table-column>
          <el-table-column label="创建时间" width="168">
            <template #default="{ row }">
              <span class="mqtt-batch-mapping__mono">{{ formatDisplayTime(row.createdAt) }}</span>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="124" fixed="right">
            <template #default="{ row }">
              <div class="mqtt-batch-mapping__row-actions">
                <el-tooltip content="解析规则" placement="top">
                  <button
                    type="button"
                    class="mqtt-batch-mapping__icon-action"
                    @click="openRuleDialog(row)"
                  >
                    <IconTablerBraces />
                  </button>
                </el-tooltip>
                <el-tooltip content="复制" placement="top">
                  <button
                    type="button"
                    class="mqtt-batch-mapping__icon-action"
                    @click="copyMappingRow(row)"
                  >
                    <IconTablerCopy />
                  </button>
                </el-tooltip>
                <el-tooltip content="删除" placement="top">
                  <button
                    type="button"
                    class="mqtt-batch-mapping__icon-action is-danger"
                    @click="removeMappingRow(row)"
                  >
                    <IconTablerTrash />
                  </button>
                </el-tooltip>
              </div>
            </template>
          </el-table-column>
        </el-table>
        <BulkActionBar :selected-count="selectedMappingCount" @clear="clearAllMappingSelection">
          <button
            type="button"
            class="mqtt-batch-mapping__bulk-action-btn"
            @click="selectCurrentMappingPage"
          >
            当前页
          </button>
          <button
            type="button"
            class="mqtt-batch-mapping__bulk-action-btn"
            @click="selectAllMappingResults"
          >
            全部结果
          </button>
          <button
            type="button"
            class="mqtt-batch-mapping__bulk-action-btn is-danger"
            @click="removeSelectedMappings"
          >
            删除选中
          </button>
        </BulkActionBar>
        <div class="mqtt-batch-mapping__pagination">
          <el-pagination
            v-model:current-page="mappingPagination.page"
            v-model:page-size="mappingPagination.pageSize"
            :page-sizes="[50, 100, 200]"
            :total="mappingDisplayTotal"
            background
            layout="total, sizes, prev, pager, next, jumper"
            small
            @size-change="handleMappingPageSizeChange"
            @current-change="handleMappingPageChange"
          />
        </div>
      </div>
      <div
        v-if="!configCollapsed"
        class="mqtt-batch-mapping__resize-handle"
        @mousedown="startConfigResize"
      />
      <div v-if="!configCollapsed" class="mqtt-batch-mapping__config">
        <div class="mqtt-batch-mapping__sample">
          <div class="mqtt-batch-mapping__section-title">
            <strong>样例消息</strong>
            <el-button text size="small" @click="fillDefaultSample">填入示例</el-button>
          </div>
          <MonacoEditor
            v-model="samplePayload"
            class="mqtt-batch-mapping__sample-input"
            language="json"
            theme="vs"
            height="100%"
            :options="sampleEditorOptions"
          />
        </div>

        <div class="mqtt-batch-mapping__rules">
          <div class="mqtt-batch-mapping__section-title">
            <strong>拆分规则</strong>
          </div>
          <el-form label-position="top" class="mqtt-batch-mapping__form">
            <el-form-item label="变量数组路径">
              <el-input v-model="ruleForm.arrayPath" placeholder="$ 或 $.items" />
            </el-form-item>
            <el-form-item label="变量名字段路径">
              <el-input v-model="ruleForm.namePath" placeholder="N" />
            </el-form-item>
            <el-form-item label="值字段路径">
              <el-input v-model="ruleForm.valuePath" placeholder="V" />
            </el-form-item>
            <el-form-item label="质量字段路径">
              <el-input v-model="ruleForm.qualityPath" placeholder="Q" />
            </el-form-item>
            <el-form-item label="时间字段路径">
              <el-input v-model="ruleForm.timePath" placeholder="T" />
            </el-form-item>
          </el-form>
          <div class="mqtt-batch-mapping__rule-actions">
            <el-tooltip
              content="根据样例消息和拆分规则生成或更新右侧变量映射清单；这里只预览并整理清单，点击保存映射后才会创建或更新变量。"
              placement="top"
            >
              <el-button type="primary" plain size="small" @click="parseSample">
                <IconTablerWand class="mqtt-batch-mapping__button-icon" />
                解析预览
              </el-button>
            </el-tooltip>
            <el-tooltip
              content="保存当前拆分规则为本订阅的默认建点规则；之后新建的变量会按这套规则写入变量解析规则。"
              placement="top"
            >
              <el-button size="small" :loading="defaultRuleSaving" @click="saveDefaultRule">
                <IconTablerDeviceFloppy class="mqtt-batch-mapping__button-icon" />
                保存规则
              </el-button>
            </el-tooltip>
          </div>
        </div>
      </div>
    </section>

    <DcDialog
      v-model="batchGenerateVisible"
      title="批量生成变量"
      width="460px"
      @close="resetBatchGenerateForm"
    >
      <el-form label-width="92px" class="mqtt-batch-mapping__generate-form">
        <el-form-item label="变量前缀">
          <el-input v-model="batchGenerateForm.prefix" placeholder="例如 tag" />
        </el-form-item>
        <el-form-item label="起始序号">
          <el-input-number v-model="batchGenerateForm.start" :min="0" :precision="0" />
        </el-form-item>
        <el-form-item label="结束序号">
          <el-input-number v-model="batchGenerateForm.end" :min="0" :precision="0" />
        </el-form-item>
        <el-form-item label="数据类型">
          <el-select v-model="batchGenerateForm.dataType">
            <el-option label="字符串" value="string" />
            <el-option label="数值（float64）" value="float64" />
            <el-option label="布尔" value="bool" />
            <el-option label="对象" value="object" />
            <el-option label="数组" value="array" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="batchGenerateVisible = false">取消</el-button>
        <el-button type="primary" @click="applyBatchGenerate">生成</el-button>
      </template>
    </DcDialog>

    <DcDialog
      v-model="ruleDialogVisible"
      :title="`编辑解析规则 - ${ruleDialogForm.name || '-'}`"
      width="680px"
      @close="resetRuleDialog"
    >
      <el-form label-position="top" class="mqtt-batch-mapping__rule-dialog-form">
        <div class="mqtt-batch-mapping__rule-dialog-summary">
          <span>变量名：{{ ruleDialogForm.name || '-' }}</span>
          <span>解析类型：批量映射</span>
        </div>
        <div class="mqtt-batch-mapping__rule-dialog-grid">
          <el-form-item label="变量数组路径">
            <el-input v-model="ruleDialogForm.arrayPath" placeholder="$ 或 $.data.data" />
          </el-form-item>
          <el-form-item label="变量名字段路径">
            <el-input v-model="ruleDialogForm.namePath" placeholder="N" />
          </el-form-item>
          <el-form-item label="匹配变量名">
            <el-input v-model="ruleDialogForm.matchName" placeholder="例如 tag1" />
          </el-form-item>
          <el-form-item label="值字段路径">
            <el-input v-model="ruleDialogForm.valuePath" placeholder="V" />
          </el-form-item>
          <el-form-item label="质量字段路径">
            <el-input v-model="ruleDialogForm.qualityPath" placeholder="Q" />
          </el-form-item>
          <el-form-item label="时间字段路径">
            <el-input v-model="ruleDialogForm.timePath" placeholder="T" />
          </el-form-item>
        </div>
      </el-form>
      <template #footer>
        <el-button @click="ruleDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="ruleDialogSaving" @click="saveRuleDialog">
          保存规则
        </el-button>
      </template>
    </DcDialog>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { debugLogger } from '@/utils/debug'
import { useWindowSize } from '@vueuse/core'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  createMqttTagsBatch,
  deleteMqttTagsByFilter,
  deleteMqttTagsBatch,
  getDataPointStatuses,
  getMqttTagValues,
  getMqttTags,
  updateMqttSubscriptionDefaultBatchParseRule,
  updateMqttTag,
} from '@/api/data.api'
import { useMqttTagSync } from '@/composables/useMqttTagSync'
import { getApiErrorMessage } from '@/utils/request'
import { TIME_FORMAT } from '@/constants'
import DcDialog from '@/components/shared/DcDialog.vue'
import BulkActionBar from '@/components/shared/BulkActionBar.vue'
import MonacoEditor from '@/components/MonacoEditor.vue'
import dayjs from 'dayjs'
import IconTablerActivity from '~icons/tabler/activity'
import IconTablerBraces from '~icons/tabler/braces'
import IconTablerCopy from '~icons/tabler/copy'
import IconTablerDeviceFloppy from '~icons/tabler/device-floppy'
import IconTablerLayoutSidebarRight from '~icons/tabler/layout-sidebar-right'
import IconTablerPlus from '~icons/tabler/plus'
import IconTablerRefresh from '~icons/tabler/refresh'
import IconTablerSearch from '~icons/tabler/search'
import IconTablerSortAscending from '~icons/tabler/sort-ascending'
import IconTablerSortDescending from '~icons/tabler/sort-descending'
import IconTablerTrash from '~icons/tabler/trash'
import IconTablerWand from '~icons/tabler/wand'

type MqttSubscription = {
  id: string
  name?: string
  topic?: string
  defaultBatchParseRule?: BatchParseRule | null
}

type BatchMappingRow = {
  matchName: string
  name: string
  code: string
  dataType: 'string' | 'float64' | 'bool' | 'object' | 'array'
  sampleValue: unknown
  sampleQuality: unknown
  sampleTime: unknown
  currentValue?: any
  datapointPath?: string
  datapointStatus?: string
  createdAt?: string
  localPinned?: boolean
  existingTagId?: string
  customRule?: BatchParseRule
}

type BatchParseRule = {
  arrayPath?: string
  namePath?: string
  matchName?: string
  valuePath?: string
  qualityPath?: string
  timePath?: string
}

type MonitorTagSnapshot = {
  id: string
  name: string
  code: string
  dataType: BatchMappingRow['dataType']
  currentValue?: any
  createdAt?: string
}

const props = defineProps<{
  projectId: string
  subscription: MqttSubscription
  previewSessionId?: string
}>()

const emit = defineEmits<{
  (event: 'openMonitor'): void
  (event: 'subscriptionUpdated', subscription: MqttSubscription): void
}>()

const loading = ref(false)
const saving = ref(false)
const runtimeInfoLoading = ref(false)
const loadRequestSeq = ref(0)
const existingTags = ref<any[]>([])
const mappings = ref<BatchMappingRow[]>([])
const selectedMappingKeys = ref<Set<string>>(new Set())
const allMappingResultsSelected = ref(false)
const pendingDeletedTagIds = ref<Set<string>>(new Set())
const mappingTableRef = ref()
const syncingMappingSelection = ref(false)
const configPanelWidth = ref(420)
const configCollapsed = ref(false)
const resizingConfig = ref(false)
const bodyRef = ref<HTMLElement | null>(null)
const searchKeyword = ref('')
const sortBy = ref<'createdAt' | 'name'>('createdAt')
const sortOrder = ref<'asc' | 'desc'>('desc')
const sortFieldPopoverVisible = ref(false)
const editingMappingCell = ref<{ key: string; field: 'name' | 'dataType' } | null>(null)
const samplePayload = ref('')
const batchGenerateVisible = ref(false)
const ruleDialogVisible = ref(false)
const ruleDialogSaving = ref(false)
const defaultRuleSaving = ref(false)
const savedDefaultRule = ref<BatchParseRule | null>(null)
const mappingPagination = reactive({
  page: 1,
  pageSize: 50,
  total: 0,
  totalPages: 0,
})
const batchGenerateForm = reactive({
  prefix: 'tag',
  start: 1,
  end: 100,
  dataType: 'float64' as BatchMappingRow['dataType'],
})
const ruleForm = reactive({
  arrayPath: '$',
  namePath: 'N',
  valuePath: 'V',
  qualityPath: 'Q',
  timePath: 'T',
})
const runtimeRuleKeys: Array<keyof BatchParseRule> = [
  'arrayPath',
  'namePath',
  'valuePath',
  'qualityPath',
  'timePath',
]
const ruleDialogForm = reactive({
  rowKey: '',
  tagId: '',
  name: '',
  code: '',
  dataType: 'float64' as BatchMappingRow['dataType'],
  arrayPath: '$',
  namePath: 'N',
  matchName: '',
  valuePath: 'V',
  qualityPath: 'Q',
  timePath: 'T',
})
const DATAPOINT_STATUS_QUERY_BATCH_SIZE = 100
const LOADING_DELAY_MS = 180
const SAMPLE_PAYLOAD_STORAGE_PREFIX = 'mqtt-batch-sample'
const sampleEditorOptions = {
  minimap: { enabled: false },
  fontSize: 12,
  lineHeight: 20,
  tabSize: 2,
  wordWrap: 'off',
  scrollBeyondLastLine: false,
  automaticLayout: true,
}
const { refresh: notifyTagRefresh } = useMqttTagSync(props.subscription.id)
const { width: windowWidth } = useWindowSize()
const samplePayloadStorageKey = computed(
  () => `${SAMPLE_PAYLOAD_STORAGE_PREFIX}:${props.projectId}:${props.subscription.id}`,
)
const CONFIG_PANEL_MIN_WIDTH = 320
const CONFIG_PANEL_MAX_WIDTH = 680
const bodyGridStyle = computed(() => ({
  gridTemplateColumns: configCollapsed.value
    ? 'minmax(420px, 1fr)'
    : `minmax(420px, 1fr) 6px ${configPanelWidth.value}px`,
}))
const isCompactLayout = computed(() => windowWidth.value <= 1180)
const sortFieldOptions = [
  { label: '创建时间', value: 'createdAt' },
  { label: '名称', value: 'name' },
] as const
const currentSortFieldLabel = computed(
  () => sortFieldOptions.find((option) => option.value === sortBy.value)?.label || '创建时间',
)
const currentSortOrderLabel = computed(() => (sortOrder.value === 'desc' ? '降序' : '升序'))
const pendingDeletedCount = computed(() => pendingDeletedTagIds.value.size)
const hasUnsavedMappingChanges = computed(
  () =>
    pendingDeletedCount.value > 0 ||
    mappings.value.some((row, index) => !row.existingTagId || shouldUpdateMappingRow(row, index)),
)
const localFilteredMappings = computed(() => {
  const keyword = searchKeyword.value.trim().toLowerCase()
  if (!keyword) return mappings.value
  return mappings.value.filter((row) =>
    [row.name, row.matchName, row.code].some((value) =>
      String(value || '')
        .toLowerCase()
        .includes(keyword),
    ),
  )
})
const localOrderedMappings = computed(() => {
  return [...localFilteredMappings.value].sort((left, right) => {
    const pinnedDiff = Number(Boolean(right.localPinned)) - Number(Boolean(left.localPinned))
    if (pinnedDiff !== 0) return pinnedDiff
    if (sortBy.value === 'name') {
      return compareText(left.name || left.matchName, right.name || right.matchName)
    }
    return (
      compareTime(left.createdAt, right.createdAt) ||
      compareText(left.name || left.matchName, right.name || right.matchName)
    )
  })
})
const mappingDisplayTotal = computed(() =>
  hasUnsavedMappingChanges.value
    ? localOrderedMappings.value.length
    : Math.max(0, mappingPagination.total - pendingDeletedCount.value),
)
const visibleMappings = computed(() => {
  if (!hasUnsavedMappingChanges.value) return mappings.value
  const start = (mappingPagination.page - 1) * mappingPagination.pageSize
  return localOrderedMappings.value.slice(start, start + mappingPagination.pageSize)
})
const selectedMappingCount = computed(() =>
  allMappingResultsSelected.value ? mappingDisplayTotal.value : selectedMappingKeys.value.size,
)

const toMonitorTagSnapshot = (row: BatchMappingRow): MonitorTagSnapshot => ({
  id: row.existingTagId || '',
  name: row.name || row.matchName,
  code: row.code,
  dataType: row.dataType,
  currentValue: row.currentValue,
  createdAt: row.createdAt,
})

const getMonitorSnapshot = () => ({
  mode: 'batch',
  page: mappingPagination.page,
  pageSize: mappingPagination.pageSize,
  total: mappingDisplayTotal.value,
  q: searchKeyword.value.trim(),
  sortBy: sortBy.value,
  sortOrder: sortOrder.value,
  rows: visibleMappings.value.map(toMonitorTagSnapshot),
})

const startConfigResize = (event: MouseEvent) => {
  event.preventDefault()
  resizingConfig.value = true
  document.body.classList.add('mqtt-batch-mapping--resizing')
  window.addEventListener('mousemove', handleConfigResize)
  window.addEventListener('mouseup', stopConfigResize)
}

const handleConfigResize = (event: MouseEvent) => {
  if (!resizingConfig.value) return
  const right = bodyRef.value?.getBoundingClientRect().right || window.innerWidth
  const nextWidth = right - event.clientX
  configPanelWidth.value = Math.min(
    CONFIG_PANEL_MAX_WIDTH,
    Math.max(CONFIG_PANEL_MIN_WIDTH, nextWidth),
  )
}

const stopConfigResize = () => {
  resizingConfig.value = false
  document.body.classList.remove('mqtt-batch-mapping--resizing')
  window.removeEventListener('mousemove', handleConfigResize)
  window.removeEventListener('mouseup', stopConfigResize)
}

const toggleConfigPanel = () => {
  configCollapsed.value = !configCollapsed.value
  configPanelWidth.value = Math.max(configPanelWidth.value, CONFIG_PANEL_MIN_WIDTH)
}

const compareText = (left: string, right: string) => {
  const result = String(left || '').localeCompare(String(right || ''), 'zh-CN', {
    numeric: true,
    sensitivity: 'base',
  })
  return sortOrder.value === 'desc' ? -result : result
}

const compareTime = (left?: string, right?: string) => {
  const leftTime = left ? new Date(left).getTime() : 0
  const rightTime = right ? new Date(right).getTime() : 0
  const result = leftTime - rightTime
  return sortOrder.value === 'desc' ? -result : result
}

const handleSearch = () => {
  mappingPagination.page = 1
  clearAllMappingSelection()
  if (hasUnsavedMappingChanges.value) {
    return
  }
  void loadTags()
}

const changeSortField = (value: 'createdAt' | 'name') => {
  sortBy.value = value
  sortFieldPopoverVisible.value = false
  mappingPagination.page = 1
  clearAllMappingSelection()
  if (hasUnsavedMappingChanges.value) {
    return
  }
  void loadTags()
}

const toggleSortOrder = () => {
  sortOrder.value = sortOrder.value === 'desc' ? 'asc' : 'desc'
  mappingPagination.page = 1
  clearAllMappingSelection()
  if (hasUnsavedMappingChanges.value) {
    return
  }
  void loadTags()
}

const defaultSample = computed(() =>
  JSON.stringify(
    [
      { N: 'temperature', V: 23.5, T: '2026-06-02 10:00:00', Q: 192 },
      { N: 'pressure', V: 0.82, T: '2026-06-02 10:00:00', Q: 192 },
    ],
    null,
    2,
  ),
)

const fillDefaultSample = () => {
  samplePayload.value = defaultSample.value
}

const loadSamplePayloadDraft = () => {
  samplePayload.value = window.localStorage.getItem(samplePayloadStorageKey.value) || ''
}

const saveSamplePayloadDraft = (value: string) => {
  if (value) {
    window.localStorage.setItem(samplePayloadStorageKey.value, value)
  } else {
    window.localStorage.removeItem(samplePayloadStorageKey.value)
  }
}

const buildDefaultRuleFromForm = (): BatchParseRule => ({
  arrayPath: ruleForm.arrayPath.trim() || '$',
  namePath: ruleForm.namePath.trim() || 'N',
  valuePath: ruleForm.valuePath.trim() || 'V',
  qualityPath: ruleForm.qualityPath.trim() || 'Q',
  timePath: ruleForm.timePath.trim() || 'T',
})

const applyDefaultRuleToForm = (rule: BatchParseRule) => {
  ruleForm.arrayPath = rule.arrayPath || '$'
  ruleForm.namePath = rule.namePath || 'N'
  ruleForm.valuePath = rule.valuePath || 'V'
  ruleForm.qualityPath = rule.qualityPath || 'Q'
  ruleForm.timePath = rule.timePath || 'T'
}

const loadDefaultRuleDraft = () => {
  const rule = normalizeBatchRuleObject(props.subscription.defaultBatchParseRule)
  savedDefaultRule.value = rule && Object.keys(rule).length > 0 ? rule : null
  if (savedDefaultRule.value) {
    applyDefaultRuleToForm(savedDefaultRule.value)
  }
}

const saveDefaultRule = async () => {
  const rule = buildDefaultRuleFromForm()
  defaultRuleSaving.value = true
  try {
    const response = await updateMqttSubscriptionDefaultBatchParseRule(
      props.projectId,
      props.subscription.id,
      rule,
    )
    const updatedSubscription = response.data || {
      ...props.subscription,
      defaultBatchParseRule: rule,
    }
    savedDefaultRule.value = updatedSubscription.defaultBatchParseRule || rule
    emit('subscriptionUpdated', updatedSubscription)
    ElMessage.success('拆分规则已保存，之后新建变量会使用这套规则')
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '保存拆分规则失败'))
  } finally {
    defaultRuleSaving.value = false
  }
}

const resolveNewRowRule = (matchName: string): BatchParseRule => ({
  ...(savedDefaultRule.value || buildDefaultRuleFromForm()),
  matchName,
})

const normalizeBatchRuleObject = (
  value?: Record<string, unknown> | BatchParseRule | null,
): BatchParseRule => ({
  arrayPath: String(value?.arrayPath || '').trim() || undefined,
  namePath: String(value?.namePath || '').trim() || undefined,
  valuePath: String(value?.valuePath || '').trim() || undefined,
  qualityPath: String(value?.qualityPath || '').trim() || undefined,
  timePath: String(value?.timePath || '').trim() || undefined,
  matchName: String(value?.matchName || '').trim() || undefined,
})

const getDataTypeLabel = (dataType?: BatchMappingRow['dataType']) => {
  const labels = {
    string: '字符串',
    float64: '数值',
    bool: '布尔',
    object: '对象',
    array: '数组',
  }
  return dataType ? labels[dataType] || dataType : '-'
}

const isEditingMappingCell = (row: BatchMappingRow, field: 'name' | 'dataType') =>
  editingMappingCell.value?.key === row.matchName && editingMappingCell.value.field === field

const startEditingMappingCell = (row: BatchMappingRow, field: 'name' | 'dataType') => {
  editingMappingCell.value = { key: row.matchName, field }
}

const stopEditingMappingCell = () => {
  editingMappingCell.value = null
}

const loadTags = async (options: { loadValues?: boolean; silent?: boolean } = {}) => {
  const requestSeq = loadRequestSeq.value + 1
  loadRequestSeq.value = requestSeq
  let loadingTimer: ReturnType<typeof window.setTimeout> | null = null
  if (!options.silent) {
    loadingTimer = window.setTimeout(() => {
      if (requestSeq === loadRequestSeq.value) {
        loading.value = true
      }
    }, LOADING_DELAY_MS)
  }
  try {
    // 搜索输入会即时触发分页请求，旧请求晚返回时必须丢弃，避免覆盖用户最新筛选结果。
    const response = await getMqttTags(props.projectId, props.subscription.id, {
      page: mappingPagination.page,
      pageSize: mappingPagination.pageSize,
      q: searchKeyword.value.trim() || undefined,
      sortBy: sortBy.value,
      sortOrder: sortOrder.value,
    })
    if (requestSeq !== loadRequestSeq.value) return
    existingTags.value = response.data?.list || []
    const pageInfo = response.data?.pagination || {}
    mappingPagination.page = pageInfo.page || mappingPagination.page
    mappingPagination.pageSize = pageInfo.pageSize || mappingPagination.pageSize
    mappingPagination.total = pageInfo.total || 0
    mappingPagination.totalPages = pageInfo.totalPages || 0
    restoreMappingsFromExistingTags()
    if (requestSeq === loadRequestSeq.value) {
      await syncVisibleMappingSelection()
    }
    // 变量清单是首屏核心内容；最近值和数据点状态属于补充信息，放到后台加载，避免 1000+ 变量时被分片请求卡住首屏。
    void loadMappingRuntimeInfo(options, requestSeq)
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '加载变量映射失败'))
  } finally {
    if (loadingTimer) {
      window.clearTimeout(loadingTimer)
    }
    if (!options.silent && requestSeq === loadRequestSeq.value) {
      loading.value = false
    }
  }
}

const reloadQuietly = async () => {
  await loadTags({ silent: true })
}

const parseSample = () => {
  let parsed: unknown
  try {
    parsed = JSON.parse(samplePayload.value)
  } catch {
    ElMessage.error('样例消息不是合法 JSON')
    return
  }

  const arrayValue = resolvePath(parsed, ruleForm.arrayPath)
  if (!Array.isArray(arrayValue)) {
    ElMessage.error('变量数组路径没有命中数组')
    return
  }

  const rows = arrayValue
    .map((item) => {
      const rawName = resolvePath(item, ruleForm.namePath)
      const matchName = String(rawName ?? '').trim()
      if (!matchName) return null
      const sampleValue = resolvePath(item, ruleForm.valuePath)
      return {
        matchName,
        name: matchName,
        code: normalizeTagCode(matchName),
        dataType: inferDataType(sampleValue),
        sampleValue,
        sampleQuality: resolvePath(item, ruleForm.qualityPath),
        sampleTime: resolvePath(item, ruleForm.timePath),
        createdAt: new Date().toISOString(),
        localPinned: true,
        customRule: resolveNewRowRule(matchName),
      }
    })
    .filter(Boolean) as BatchMappingRow[]

  const uniqueRows = Array.from(new Map(rows.map((row) => [row.matchName, row])).values())
  mergeSampleRows(uniqueRows)
  markExistingMappings()
  normalizeMappingPage()
}

const mergeSampleRows = (rows: BatchMappingRow[]) => {
  const rowMap = new Map(mappings.value.map((row) => [row.matchName, row]))
  const previousCount = mappings.value.length
  const parsedRows = rows.map((row) => {
    const current = rowMap.get(row.matchName)
    return {
      ...(current || row),
      sampleValue: row.sampleValue,
      sampleQuality: row.sampleQuality,
      sampleTime: row.sampleTime,
      dataType: current?.dataType || row.dataType,
      name: current?.name || row.name,
      code: current?.code || row.code,
      createdAt: current?.createdAt || row.createdAt,
      localPinned: current?.localPinned ?? row.localPinned,
      customRule: current?.customRule || row.customRule,
    }
  })
  const parsedKeys = new Set(parsedRows.map((row) => row.matchName))
  const remainingRows = mappings.value.filter((row) => !parsedKeys.has(row.matchName))
  mappings.value = [...parsedRows, ...remainingRows]
  mappingPagination.page = 1
  mappingPagination.total = Math.max(mappingPagination.total, mappings.value.length, previousCount)
  normalizeMappingPage()
}

const saveMappings = async () => {
  const deletedTagIds = Array.from(pendingDeletedTagIds.value)
  if (mappings.value.length === 0 && deletedTagIds.length === 0) {
    ElMessage.info('请先解析出变量映射')
    return
  }

  const createRows = mappings.value.filter((row) => !row.existingTagId)
  const updateRows = mappings.value.filter((row, index) => shouldUpdateMappingRow(row, index))
  saving.value = true
  try {
    // 后端分页后，前端只持有当前页；删除必须只处理用户显式移除的行，不能再用“当前清单外即删除”的全量语义。
    if (deletedTagIds.length > 0) {
      await deleteMqttTagsBatch(props.projectId, deletedTagIds)
    }
    if (createRows.length > 0) {
      await createMqttTagsBatch(
        props.projectId,
        props.subscription.id,
        createRows.map((row, index) => buildTagPayload(row, index)),
      )
    }
    await Promise.all(
      updateRows.map((row, index) =>
        updateMqttTag(props.projectId, row.existingTagId, buildTagPayload(row, index)),
      ),
    )
    ElMessage.success(
      `映射已保存：新建 ${createRows.length} 个，更新 ${updateRows.length} 个，删除 ${deletedTagIds.length} 个`,
    )
    pendingDeletedTagIds.value = new Set()
    await loadTags({ loadValues: false })
    clearAllMappingSelection()
    notifyTagRefresh()
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '保存批量变量映射失败'))
  } finally {
    saving.value = false
  }
}

const buildTagPayload = (row: BatchMappingRow, index: number) => ({
  name: row.name,
  code: row.code,
  dataType: row.dataType,
  parseType: 'batch_jsonpath',
  parseRule: JSON.stringify(resolveRowRule(row)),
  order: index,
})

const resolveRowRule = (row: BatchMappingRow): BatchParseRule => {
  if (row.customRule && Object.keys(row.customRule).length > 0) {
    return {
      arrayPath: row.customRule.arrayPath || ruleForm.arrayPath,
      namePath: row.customRule.namePath || ruleForm.namePath,
      matchName: row.customRule.matchName || row.matchName,
      valuePath: row.customRule.valuePath || ruleForm.valuePath,
      qualityPath: row.customRule.qualityPath || ruleForm.qualityPath,
      timePath: row.customRule.timePath || ruleForm.timePath,
    }
  }

  if (row.existingTagId) {
    const current = existingTags.value.find((tag) => tag.id === row.existingTagId)
    const currentRule = parseBatchRule(current?.parseRule || '')
    if (currentRule && Object.keys(currentRule).length > 0) {
      return {
        arrayPath: currentRule.arrayPath || ruleForm.arrayPath,
        namePath: currentRule.namePath || ruleForm.namePath,
        matchName: currentRule.matchName || row.matchName,
        valuePath: currentRule.valuePath || ruleForm.valuePath,
        qualityPath: currentRule.qualityPath || ruleForm.qualityPath,
        timePath: currentRule.timePath || ruleForm.timePath,
      }
    }
  }

  return {
    arrayPath: ruleForm.arrayPath,
    namePath: ruleForm.namePath,
    matchName: row.matchName,
    valuePath: ruleForm.valuePath,
    qualityPath: ruleForm.qualityPath,
    timePath: ruleForm.timePath,
  }
}

const shouldUpdateMappingRow = (row: BatchMappingRow, index: number) => {
  if (!row.existingTagId) return false
  const current = existingTags.value.find((tag) => tag.id === row.existingTagId)
  if (!current) return false

  if (current.name !== row.name || current.code !== row.code || current.dataType !== row.dataType) {
    return true
  }

  const currentRule = parseBatchRule(current.parseRule)
  const nextRule = parseBatchRule(buildTagPayload(row, index).parseRule)
  if (currentRule.matchName !== nextRule.matchName) return true
  if (isRuntimeRuleChanged(currentRule, nextRule)) return true

  return false
}

const isRuntimeRuleChanged = (currentRule: BatchParseRule, nextRule: BatchParseRule) =>
  runtimeRuleKeys.some((key) => String(currentRule?.[key] || '') !== String(nextRule?.[key] || ''))

const openRuleDialog = (row: BatchMappingRow) => {
  const current = row.existingTagId
    ? existingTags.value.find((tag) => tag.id === row.existingTagId)
    : null
  const rule = parseBatchRule(current?.parseRule || JSON.stringify(resolveRowRule(row)))
  ruleDialogForm.rowKey = row.matchName
  ruleDialogForm.tagId = row.existingTagId || ''
  ruleDialogForm.name = row.name || row.matchName
  ruleDialogForm.code = row.code
  ruleDialogForm.dataType = row.dataType
  ruleDialogForm.arrayPath = rule.arrayPath || '$'
  ruleDialogForm.namePath = rule.namePath || 'N'
  ruleDialogForm.matchName = rule.matchName || row.matchName
  ruleDialogForm.valuePath = rule.valuePath || 'V'
  ruleDialogForm.qualityPath = rule.qualityPath || 'Q'
  ruleDialogForm.timePath = rule.timePath || 'T'
  ruleDialogVisible.value = true
}

const resetRuleDialog = () => {
  ruleDialogForm.rowKey = ''
  ruleDialogForm.tagId = ''
  ruleDialogForm.name = ''
  ruleDialogForm.code = ''
  ruleDialogForm.dataType = 'float64'
  ruleDialogForm.arrayPath = '$'
  ruleDialogForm.namePath = 'N'
  ruleDialogForm.matchName = ''
  ruleDialogForm.valuePath = 'V'
  ruleDialogForm.qualityPath = 'Q'
  ruleDialogForm.timePath = 'T'
}

const buildRuleDialogParseRule = (): BatchParseRule => ({
  arrayPath: ruleDialogForm.arrayPath.trim() || '$',
  namePath: ruleDialogForm.namePath.trim() || 'N',
  matchName: ruleDialogForm.matchName.trim(),
  valuePath: ruleDialogForm.valuePath.trim() || 'V',
  qualityPath: ruleDialogForm.qualityPath.trim() || 'Q',
  timePath: ruleDialogForm.timePath.trim() || 'T',
})

const saveRuleDialog = async () => {
  const nextRule = buildRuleDialogParseRule()
  if (!nextRule.matchName) {
    ElMessage.warning('匹配变量名不能为空')
    return
  }

  const duplicated = mappings.value.some(
    (item) => item.matchName !== ruleDialogForm.rowKey && item.matchName === nextRule.matchName,
  )
  if (duplicated) {
    ElMessage.warning(`变量 ${nextRule.matchName} 已存在，请更换匹配变量名`)
    return
  }

  const row = mappings.value.find((item) => item.matchName === ruleDialogForm.rowKey)
  if (!row) {
    ElMessage.warning('变量行不存在，请刷新后重试')
    return
  }

  const parseRule = JSON.stringify(nextRule)
  if (!ruleDialogForm.tagId) {
    row.matchName = nextRule.matchName
    row.name = ruleDialogForm.name || nextRule.matchName
    row.customRule = nextRule
    ruleDialogVisible.value = false
    ElMessage.success('解析规则已更新，保存映射后生效')
    return
  }

  ruleDialogSaving.value = true
  try {
    await updateMqttTag(props.projectId, ruleDialogForm.tagId, {
      name: ruleDialogForm.name,
      code: ruleDialogForm.code,
      dataType: ruleDialogForm.dataType,
      parseType: 'batch_jsonpath',
      parseRule,
      order:
        existingTags.value.find((tag) => tag.id === ruleDialogForm.tagId)?.order ??
        visibleMappings.value.findIndex((item) => item.existingTagId === ruleDialogForm.tagId),
    })
    const current = existingTags.value.find((tag) => tag.id === ruleDialogForm.tagId)
    if (current) {
      current.parseRule = parseRule
      current.name = ruleDialogForm.name
      current.code = ruleDialogForm.code
      current.dataType = ruleDialogForm.dataType
    }
    row.matchName = nextRule.matchName
    row.name = ruleDialogForm.name || nextRule.matchName
    row.code = ruleDialogForm.code
    row.dataType = ruleDialogForm.dataType
    row.customRule = nextRule
    ruleDialogVisible.value = false
    ElMessage.success('解析规则已保存')
    notifyTagRefresh()
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '保存解析规则失败'))
  } finally {
    ruleDialogSaving.value = false
  }
}

const createMappingRow = (
  matchName: string,
  dataType: BatchMappingRow['dataType'] = 'float64',
  createdAt = new Date().toISOString(),
): BatchMappingRow => ({
  matchName,
  name: matchName,
  code: normalizeTagCode(matchName),
  dataType,
  sampleValue: undefined,
  sampleQuality: undefined,
  sampleTime: undefined,
  createdAt,
  localPinned: true,
  customRule: resolveNewRowRule(matchName),
})

const addMappingRow = () => {
  const nextIndex = resolveNextGeneratedMappingIndex()
  const row = createMappingRow(`tag${nextIndex}`)
  mappings.value = [row, ...mappings.value]
  markExistingMappings()
  mappingPagination.page = 1
  mappingPagination.total = Math.max(mappingPagination.total, mappings.value.length)
}

const resolveNextGeneratedMappingIndex = () => {
  const maxLoadedIndex = mappings.value.reduce((max, row) => {
    const match = String(row.name || row.matchName || '').match(/^tag(\d+)$/)
    return match ? Math.max(max, Number(match[1])) : max
  }, 0)
  return Math.max(maxLoadedIndex, mappingDisplayTotal.value) + 1
}

const copyMappingRow = (row: BatchMappingRow) => {
  const nextName = nextCopyName(row.name || row.matchName)
  mappings.value = [
    {
      ...row,
      matchName: nextName,
      name: nextName,
      code: normalizeTagCode(nextName),
      existingTagId: '',
      sampleValue: undefined,
      sampleQuality: undefined,
      sampleTime: undefined,
      currentValue: undefined,
      datapointPath: '',
      datapointStatus: '',
      createdAt: new Date().toISOString(),
      localPinned: true,
      customRule: {
        ...resolveRowRule(row),
        matchName: nextName,
      },
    },
    ...mappings.value,
  ]
  markExistingMappings()
  mappingPagination.page = 1
  mappingPagination.total = Math.max(mappingPagination.total, mappings.value.length)
}

const removeMappingRow = (row: BatchMappingRow) => {
  if (row.existingTagId) {
    pendingDeletedTagIds.value.add(row.existingTagId)
  }
  mappings.value = mappings.value.filter((item) => item !== row)
  removeSelectedMappingKeys([row.matchName])
  normalizeMappingPage()
}

const handleMappingSelectionChange = (rows: BatchMappingRow[]) => {
  if (syncingMappingSelection.value) return
  allMappingResultsSelected.value = false
  const visibleKeys = new Set(visibleMappings.value.map((row) => row.matchName))
  const nextKeys = new Set(selectedMappingKeys.value)
  visibleKeys.forEach((key) => nextKeys.delete(key))
  rows.forEach((row) => nextKeys.add(row.matchName))
  selectedMappingKeys.value = nextKeys
}

const handleMappingSelectAll = async (rows: BatchMappingRow[]) => {
  allMappingResultsSelected.value = false
  const visibleKeys = visibleMappings.value.map((row) => row.matchName)
  if (rows.length === 0) {
    removeSelectedMappingKeys(visibleKeys)
    return
  }

  const nextKeys = new Set(selectedMappingKeys.value)
  visibleKeys.forEach((key) => nextKeys.add(key))
  selectedMappingKeys.value = nextKeys
  await syncVisibleMappingSelection()
}

const selectCurrentMappingPage = async () => {
  allMappingResultsSelected.value = false
  selectedMappingKeys.value = new Set(visibleMappings.value.map((row) => row.matchName))
  await syncVisibleMappingSelection()
}

const selectAllMappingResults = async () => {
  allMappingResultsSelected.value = true
  selectedMappingKeys.value = new Set(visibleMappings.value.map((row) => row.matchName))
  await syncVisibleMappingSelection()
}

const syncVisibleMappingSelection = async () => {
  await nextTick()
  const table = mappingTableRef.value
  if (!table) return
  syncingMappingSelection.value = true
  table.clearSelection()
  visibleMappings.value.forEach((row) => {
    if (allMappingResultsSelected.value || selectedMappingKeys.value.has(row.matchName)) {
      table.toggleRowSelection(row, true)
    }
  })
  await nextTick()
  syncingMappingSelection.value = false
}

const removeSelectedMappings = async () => {
  if (allMappingResultsSelected.value) {
    if (hasUnsavedMappingChanges.value) {
      const selectedKeys = new Set(localFilteredMappings.value.map((row) => row.matchName))
      const removedCount = selectedKeys.size
      if (removedCount === 0) return
      mappings.value.forEach((row) => {
        if (selectedKeys.has(row.matchName) && row.existingTagId) {
          pendingDeletedTagIds.value.add(row.existingTagId)
        }
      })
      mappings.value = mappings.value.filter((row) => !selectedKeys.has(row.matchName))
      clearAllMappingSelection()
      normalizeMappingPage()
      ElMessage.success(`已从清单移除 ${removedCount} 个变量，保存后生效`)
      return
    }

    const deleteCount = mappingDisplayTotal.value
    if (deleteCount === 0) return
    try {
      await ElMessageBox.confirm(
        `确定要删除当前筛选结果中的 ${deleteCount} 个变量吗？`,
        '批量删除确认',
        {
          type: 'warning',
          confirmButtonText: '删除',
          cancelButtonText: '取消',
        },
      )
      const response = await deleteMqttTagsByFilter(props.projectId, props.subscription.id, {
        search: searchKeyword.value.trim(),
      })
      const deletedCount = response?.data?.deletedCount ?? 0
      ElMessage.success(`已删除 ${deletedCount} 个变量`)
      clearAllMappingSelection()
      await loadTags({ loadValues: false })
      notifyTagRefresh()
    } catch (error) {
      if (error !== 'cancel') {
        ElMessage.error(getApiErrorMessage(error, '批量删除失败'))
      }
    }
    return
  }
  const selectedKeys = selectedMappingKeys.value
  if (selectedKeys.size === 0) return

  const removedCount = selectedKeys.size
  mappings.value.forEach((row) => {
    if (selectedKeys.has(row.matchName) && row.existingTagId) {
      pendingDeletedTagIds.value.add(row.existingTagId)
    }
  })
  mappings.value = mappings.value.filter((row) => !selectedKeys.has(row.matchName))
  clearAllMappingSelection()
  normalizeMappingPage()
  ElMessage.success(`已从清单移除 ${removedCount} 个变量，保存后生效`)
}

const removeSelectedMappingKeys = (keys: string[]) => {
  const nextKeys = new Set(selectedMappingKeys.value)
  keys.forEach((key) => nextKeys.delete(key))
  selectedMappingKeys.value = nextKeys
}

const clearAllMappingSelection = () => {
  allMappingResultsSelected.value = false
  selectedMappingKeys.value = new Set()
  void syncVisibleMappingSelection()
}

const handleMappingPageSizeChange = async () => {
  mappingPagination.page = 1
  if (!allMappingResultsSelected.value) {
    clearAllMappingSelection()
  }
  if (hasUnsavedMappingChanges.value) {
    await syncVisibleMappingSelection()
    return
  }
  await loadTags()
}

const handleMappingPageChange = async () => {
  if (!allMappingResultsSelected.value) {
    clearAllMappingSelection()
  }
  if (hasUnsavedMappingChanges.value) {
    await syncVisibleMappingSelection()
    return
  }
  await loadTags()
}

const normalizeMappingPage = () => {
  const total = mappingDisplayTotal.value
  const totalPages = Math.max(1, Math.ceil(total / mappingPagination.pageSize))
  if (mappingPagination.page > totalPages) {
    mappingPagination.page = totalPages
  }
}

const nextCopyName = (name: string) => {
  const normalizedName = String(name || 'tag').trim() || 'tag'
  const matched = normalizedName.match(/^(.*?)(\d+)$/)
  const names = new Set(mappings.value.flatMap((row) => [row.matchName, row.name]))

  if (matched) {
    const prefix = matched[1]
    const rawNumber = matched[2]
    let nextNumber = Number(rawNumber) + 1
    let candidate = `${prefix}${String(nextNumber).padStart(rawNumber.length, '0')}`
    while (names.has(candidate)) {
      nextNumber += 1
      candidate = `${prefix}${String(nextNumber).padStart(rawNumber.length, '0')}`
    }
    return candidate
  }

  const base = `${normalizedName}_copy`
  let copyIndex = 1
  let candidate = base
  while (names.has(candidate)) {
    copyIndex += 1
    candidate = `${base}${copyIndex}`
  }
  return candidate
}

const openBatchGenerateDialog = () => {
  batchGenerateVisible.value = true
}

const resetBatchGenerateForm = () => {
  batchGenerateForm.prefix = 'tag'
  batchGenerateForm.start = 1
  batchGenerateForm.end = 100
  batchGenerateForm.dataType = 'float64'
}

const applyBatchGenerate = () => {
  const prefix = batchGenerateForm.prefix.trim()
  if (!prefix) {
    ElMessage.warning('变量前缀不能为空')
    return
  }
  if (batchGenerateForm.end < batchGenerateForm.start) {
    ElMessage.warning('结束序号不能小于起始序号')
    return
  }
  const count = batchGenerateForm.end - batchGenerateForm.start + 1
  if (count > 1000) {
    ElMessage.warning('单次最多生成 1000 个变量')
    return
  }

  const rowMap = new Map(mappings.value.map((row) => [row.matchName, row]))
  const baseTime = Date.now()
  for (let index = batchGenerateForm.start; index <= batchGenerateForm.end; index += 1) {
    const matchName = `${prefix}${index}`
    if (!rowMap.has(matchName)) {
      rowMap.set(
        matchName,
        createMappingRow(
          matchName,
          batchGenerateForm.dataType,
          new Date(baseTime + index - batchGenerateForm.start).toISOString(),
        ),
      )
    }
  }
  mappings.value = Array.from(rowMap.values())
  markExistingMappings()
  mappingPagination.page = 1
  mappingPagination.total = Math.max(mappingPagination.total, mappings.value.length)
  sortBy.value = 'createdAt'
  sortOrder.value = 'desc'
  normalizeMappingPage()
  batchGenerateVisible.value = false
}

const markExistingMappings = () => {
  const codeMap = new Map(existingTags.value.map((tag) => [tag.code, tag]))
  mappings.value.forEach((row) => {
    const tag = codeMap.get(row.code)
    row.existingTagId = tag?.id || ''
    if (tag) {
      row.customRule = undefined
    }
    if (!tag) {
      row.currentValue = undefined
      row.datapointPath = ''
      row.datapointStatus = ''
    }
  })
}

const loadMappingRuntimeInfo = async (
  options: { loadValues?: boolean } = {},
  requestSeq = loadRequestSeq.value,
) => {
  const tagIDs = existingTags.value.map((tag) => tag.id).filter(Boolean)
  if (tagIDs.length === 0) {
    if (requestSeq === loadRequestSeq.value) {
      runtimeInfoLoading.value = false
    }
    return
  }

  runtimeInfoLoading.value = true
  const tasks = [loadMappingDatapoints(tagIDs, requestSeq)]
  if (options.loadValues !== false) {
    tasks.push(loadMappingValues(tagIDs, requestSeq))
  }
  await Promise.all(tasks)
  if (requestSeq === loadRequestSeq.value) {
    runtimeInfoLoading.value = false
  }
}

const loadMappingValues = async (tagIDs: string[], requestSeq = loadRequestSeq.value) => {
  try {
    const response = await getMqttTagValues(props.projectId, tagIDs, { compact: true })
    if (requestSeq !== loadRequestSeq.value) return
    const list = Array.isArray(response.data) ? response.data : response.data?.list || []
    const valueMap = new Map(list.map((item) => [item.tagId, normalizeTagValue(item)]))
    mappings.value.forEach((row) => {
      if (row.existingTagId) {
        row.currentValue = valueMap.get(row.existingTagId)
      }
    })
  } catch (error) {
    debugLogger.error('Failed to load batch mapping values:', error)
  }
}

const loadMappingDatapoints = async (tagIDs: string[], requestSeq = loadRequestSeq.value) => {
  try {
    const responses = await Promise.all(
      chunkArray(tagIDs, DATAPOINT_STATUS_QUERY_BATCH_SIZE).map((batchIDs) =>
        getDataPointStatuses(props.projectId, {
          sourceIds: batchIDs,
        }),
      ),
    )
    if (requestSeq !== loadRequestSeq.value) return
    const list = responses.flatMap((response) => response.data?.datapoints || [])
    const datapointMap = new Map(list.map((item) => [item.sourceId, item]))
    mappings.value.forEach((row) => {
      if (!row.existingTagId) return
      const datapoint = datapointMap.get(row.existingTagId)
      row.datapointPath = datapoint?.path || ''
      row.datapointStatus = datapoint?.status || ''
    })
  } catch (error) {
    debugLogger.error('Failed to load batch mapping datapoints:', error)
  }
}

const chunkArray = <T,>(items: T[], size: number): T[][] => {
  const chunks: T[][] = []
  for (let index = 0; index < items.length; index += size) {
    chunks.push(items.slice(index, index + size))
  }
  return chunks
}

const normalizeTagValue = (value) => ({
  parsedValue: value?.parsedValue ?? value?.value,
  value: value?.value ?? value?.parsedValue,
  quality: value?.quality || 'unknown',
  qualityCode: value?.qualityCode,
  timestamp: value?.timestamp || value?.receivedAt || '',
  error: value?.error || '',
})

const hasReceivedTagValue = (value) => {
  if (!value) return false
  const raw = value.parsedValue ?? value.value
  return raw !== null && raw !== undefined && raw !== ''
}

const applyTagValueUpdate = (value) => {
  const tagId = value?.tagId
  if (!tagId) return
  const row = mappings.value.find((item) => item.existingTagId === tagId)
  if (!row) return
  row.currentValue = normalizeTagValue(value)
}

const applyTagValueUpdates = (values: any[]) => {
  ;(values || []).forEach((value) => applyTagValueUpdate(value))
}

const restoreMappingsFromExistingTags = () => {
  const batchTags = existingTags.value.filter(
    (tag) => tag.parseType === 'batch_jsonpath' && !pendingDeletedTagIds.value.has(tag.id),
  )
  if (batchTags.length === 0) {
    mappings.value = []
    return
  }

  const rows = batchTags
    .map((tag) => {
      const rule = parseBatchRule(tag.parseRule)
      const matchName = String(rule.matchName || '').trim()
      if (!matchName) return null
      return {
        matchName,
        name: tag.name || matchName,
        code: tag.code,
        dataType: tag.dataType || 'string',
        sampleValue: undefined,
        sampleQuality: undefined,
        sampleTime: undefined,
        createdAt: tag.createdAt,
        localPinned: false,
        existingTagId: tag.id,
      }
    })
    .filter(Boolean) as BatchMappingRow[]

  const firstRule = parseBatchRule(batchTags[0]?.parseRule)
  if (!savedDefaultRule.value) {
    applyDefaultRuleToForm(firstRule)
  }
  mappings.value = rows
  normalizeMappingPage()
}

const parseBatchRule = (ruleText?: string): BatchParseRule => {
  try {
    return JSON.parse(ruleText || '{}')
  } catch {
    return {}
  }
}

const normalizeTagCode = (name: string) => {
  const prefix = props.subscription.name || props.subscription.topic || props.subscription.id
  return `${normalizeCode(prefix)}_${hashText(name)}`
}

const normalizeCode = (value: string) => {
  const code = String(value || '')
    .toLowerCase()
    .replace(/\s+/g, '_')
    .replace(/[^a-z0-9_]/g, '_')
    .replace(/^_+|_+$/g, '')
    .replace(/_+/g, '_')
  return code || `tag_${hashText(value)}`
}

const hashText = (value: string) => {
  let hash = 0
  for (const char of String(value || '')) {
    hash = (hash * 31 + char.charCodeAt(0)) >>> 0
  }
  return hash.toString(36)
}

const inferDataType = (value: unknown): BatchMappingRow['dataType'] => {
  if (Array.isArray(value)) return 'array'
  if (value !== null && typeof value === 'object') return 'object'
  if (typeof value === 'number') return 'float64'
  if (typeof value === 'boolean') return 'bool'
  return 'string'
}

const resolvePath = (source: unknown, path: string) => {
  const normalized = normalizePath(path)
  if (normalized.length === 0) return source
  return normalized.reduce((current, segment) => {
    if (current === null || current === undefined) return undefined
    if (Array.isArray(current) && /^\d+$/.test(segment)) {
      return current[Number(segment)]
    }
    return current?.[segment]
  }, source as any)
}

const normalizePath = (path: string) =>
  String(path || '')
    .trim()
    .replace(/^\$/, '')
    .replace(/^\./, '')
    .replace(/\[(\d+)\]/g, '.$1')
    .split('.')
    .map((item) => item.trim())
    .filter(Boolean)

const formatLastValue = (row: BatchMappingRow) => {
  const raw = row.currentValue?.parsedValue ?? row.currentValue?.value
  if (raw === null || raw === undefined || raw === '') return '-'
  if (typeof raw === 'object') return JSON.stringify(raw)
  if (row.dataType === 'float64') {
    const num = Number(raw)
    return Number.isFinite(num) ? String(Number(num.toFixed(4))) : String(raw)
  }
  return String(raw)
}

const formatQualityLabel = (row: BatchMappingRow) => {
  if (!hasReceivedTagValue(row.currentValue)) return '-'
  const quality = row.currentValue?.quality || 'unknown'
  const labels = {
    good: '良好',
    bad: '错误',
    uncertain: '不确定',
    unknown: '未知',
  }
  const label = labels[quality] || quality
  const qualityCode = row.currentValue?.qualityCode
  if (
    quality === 'bad' &&
    qualityCode !== null &&
    qualityCode !== undefined &&
    qualityCode !== ''
  ) {
    return `${label} ${qualityCode}`
  }
  return label
}

const formatLastTime = (row: BatchMappingRow) => {
  return hasReceivedTagValue(row.currentValue)
    ? formatDisplayTime(row.currentValue?.timestamp)
    : '-'
}

const formatDisplayTime = (value: unknown) => {
  if (value === null || value === undefined || value === '') return '-'
  const normalized = normalizeTimestampValue(value)
  const time = dayjs(normalized)
  return time.isValid() ? time.format(TIME_FORMAT) : '-'
}

const normalizeTimestampValue = (value: unknown) => {
  if (typeof value === 'number') {
    return normalizeNumericTimestamp(value)
  }

  const text = String(value).trim()
  if (/^\d+$/.test(text)) {
    return normalizeNumericTimestamp(Number(text))
  }
  return text
}

const normalizeNumericTimestamp = (value: number) => {
  const length = String(Math.trunc(Math.abs(value))).length
  if (length <= 10) return value * 1000
  if (length <= 13) return value
  if (length <= 16) return Math.floor(value / 1000)
  return Math.floor(value / 1000000)
}

const statusLabel = (row: BatchMappingRow) => {
  if (row.datapointStatus === 'invalid') return '失效'
  if (row.datapointPath) return '活跃'
  return '未生成'
}

const statusClass = (row: BatchMappingRow) => {
  if (row.datapointStatus === 'invalid') return 'is-muted'
  if (row.datapointPath) return 'is-success'
  return 'is-warning'
}

watch(
  () => props.subscription.id,
  async () => {
    mappings.value = []
    pendingDeletedTagIds.value = new Set()
    clearAllMappingSelection()
    mappingPagination.page = 1
    mappingPagination.total = 0
    mappingPagination.totalPages = 0
    loadSamplePayloadDraft()
    loadDefaultRuleDraft()
    await loadTags()
  },
)

watch(
  () => props.subscription.defaultBatchParseRule,
  () => {
    loadDefaultRuleDraft()
  },
)

watch(samplePayload, (value) => {
  saveSamplePayloadDraft(value)
})

onMounted(() => {
  loadSamplePayloadDraft()
  loadDefaultRuleDraft()
  void loadTags()
})

onBeforeUnmount(stopConfigResize)

defineExpose({
  applyTagValueUpdate,
  applyTagValueUpdates,
  getMonitorSnapshot,
  refreshQuietly: reloadQuietly,
})
</script>

<style scoped>
.mqtt-batch-mapping {
  height: 100%;
  min-height: 0;
  display: flex;
  flex-direction: column;
  background: var(--dc-surface-raised);
}

.mqtt-batch-mapping__header {
  min-height: 52px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 10px 14px;
  border-bottom: 1px solid var(--dc-border);
  background: var(--dc-surface-subtle);
}

.mqtt-batch-mapping__header-toolbar {
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}

.mqtt-batch-mapping__header-actions,
.mqtt-batch-mapping__filters {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  flex-wrap: wrap;
}

.mqtt-batch-mapping__filters :deep(.el-input) {
  width: 220px;
}

.mqtt-batch-mapping__input-icon {
  width: 14px;
  height: 14px;
}

.mqtt-batch-mapping__sort-pill {
  height: 24px;
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 0 9px;
  border: 1px solid var(--dc-border);
  border-radius: 999px;
  background: var(--dc-surface-raised);
  color: var(--dc-text-secondary);
  font-size: 12px;
  font-weight: 700;
  white-space: nowrap;
}

.mqtt-batch-mapping__sort-pill:hover {
  border-color: color-mix(in oklch, var(--dc-primary) 34%, var(--dc-border));
  color: var(--dc-primary);
}

.mqtt-batch-mapping__sort-icon {
  width: 13px;
  height: 13px;
}

.mqtt-batch-mapping__sort-menu {
  display: grid;
  gap: 4px;
}

.mqtt-batch-mapping__sort-item {
  width: 100%;
  height: 28px;
  padding: 0 9px;
  border: 0;
  border-radius: var(--dc-radius-sm);
  background: transparent;
  color: var(--dc-text-secondary);
  font-size: 12px;
  font-weight: 700;
  text-align: left;
}

.mqtt-batch-mapping__sort-item:hover,
.mqtt-batch-mapping__sort-item.is-active {
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
}

.mqtt-batch-mapping__header-actions :deep(.mqtt-batch-mapping__live-action.el-button) {
  border-color: var(--dc-border);
  background: var(--dc-surface-raised);
  color: var(--dc-text-secondary);
}

.mqtt-batch-mapping__header-actions :deep(.mqtt-batch-mapping__live-action.el-button.is-connected) {
  border-color: var(--el-color-primary);
  background: var(--el-color-primary);
  color: var(--el-color-white);
}

.mqtt-batch-mapping__body {
  height: 0;
  min-height: 0;
  flex: 1;
  display: grid;
  gap: 0;
}

.mqtt-batch-mapping__config {
  width: 100%;
  min-height: 0;
  min-width: 0;
  container-type: inline-size;
  display: grid;
  grid-template-rows: minmax(220px, 1fr) auto;
}

.mqtt-batch-mapping__resize-handle {
  width: 6px;
  min-height: 0;
  cursor: col-resize;
  background: var(--dc-surface-subtle);
  border-left: 1px solid var(--dc-border);
  border-right: 1px solid var(--dc-border);
}

.mqtt-batch-mapping__resize-handle:hover {
  background: color-mix(in oklch, var(--dc-primary) 12%, var(--dc-surface-subtle));
}

.mqtt-batch-mapping__sample,
.mqtt-batch-mapping__rules,
.mqtt-batch-mapping__result {
  min-height: 0;
  min-width: 0;
  padding: 12px;
}

.mqtt-batch-mapping__sample {
  display: flex;
  flex-direction: column;
  border-bottom: 1px solid var(--dc-border);
}

.mqtt-batch-mapping__sample-input {
  flex: 1;
  min-height: 0;
  overflow: hidden;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
}

.mqtt-batch-mapping__rules {
  display: grid;
  gap: 10px;
  align-content: start;
}

.mqtt-batch-mapping__rule-actions {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
}

.mqtt-batch-mapping__rule-actions :deep(.el-tooltip__trigger) {
  width: 100%;
}

.mqtt-batch-mapping__rule-actions :deep(.el-button) {
  width: 100%;
  justify-content: center;
  margin-left: 0;
}

.mqtt-batch-mapping__section-title {
  min-height: 28px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 8px;
}

.mqtt-batch-mapping__section-title strong {
  min-width: 0;
  overflow: hidden;
  color: var(--dc-text);
  font-size: 13px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.mqtt-batch-mapping__section-title span {
  color: var(--dc-text-muted);
  font-size: 12px;
}

.mqtt-batch-mapping__result-meta {
  display: inline-flex;
  align-items: center;
  gap: 8px;
}

.mqtt-batch-mapping__runtime-loading {
  color: var(--dc-text-muted);
  font-size: 12px;
}

.mqtt-batch-mapping__drawer-toggle {
  width: 26px;
  height: 24px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
  color: var(--dc-text-secondary);
}

.mqtt-batch-mapping__drawer-toggle:hover {
  border-color: var(--dc-primary);
  color: var(--dc-primary);
}

.mqtt-batch-mapping__drawer-toggle.is-collapsed {
  border-color: var(--dc-primary);
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
}

.mqtt-batch-mapping__drawer-toggle svg {
  width: 15px;
  height: 15px;
}

.mqtt-batch-mapping__form {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
  gap: 8px 10px;
}

.mqtt-batch-mapping__form :deep(.el-form-item) {
  margin-bottom: 0;
}

.mqtt-batch-mapping__form :deep(.el-form-item:first-child) {
  grid-column: 1 / -1;
}

.mqtt-batch-mapping__form :deep(.el-form-item__label) {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.mqtt-batch-mapping__result {
  position: relative;
  display: flex;
  flex-direction: column;
}

.mqtt-batch-mapping__table {
  flex: 1;
  min-height: 0;
}

.mqtt-batch-mapping__pagination {
  min-height: 42px;
  display: flex;
  align-items: center;
  justify-content: flex-end;
  padding-top: 8px;
  border-top: 1px solid var(--dc-border);
}

.mqtt-batch-mapping__result :deep(.dc-bulk-action-bar) {
  bottom: 58px;
}

.mqtt-batch-mapping__bulk-action-btn {
  max-width: 92px;
  height: 28px;
  min-width: 0;
  overflow: hidden;
  padding: 0 12px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
  color: var(--dc-text);
  cursor: pointer;
  font-family: inherit;
  font-size: 13px;
  font-weight: 600;
  text-align: center;
  text-overflow: ellipsis;
  white-space: nowrap;
  transition:
    background 0.18s ease,
    border-color 0.18s ease,
    color 0.18s ease;
}

.mqtt-batch-mapping__bulk-action-btn:hover {
  border-color: rgba(29, 78, 216, 0.26);
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
}

.mqtt-batch-mapping__bulk-action-btn.is-danger:hover {
  border-color: rgba(220, 38, 38, 0.26);
  background: rgba(220, 38, 38, 0.08);
  color: var(--dc-danger);
}

.mqtt-batch-mapping__mono {
  font-family: var(--dc-font-mono, ui-monospace, SFMono-Regular, Menlo, Consolas, monospace);
  color: var(--dc-text-secondary);
  font-size: 12px;
}

.mqtt-batch-mapping__value {
  font-family: var(--dc-font-mono, ui-monospace, SFMono-Regular, Menlo, Consolas, monospace);
  color: var(--dc-text);
  font-size: 12px;
  font-weight: 700;
}

.mqtt-batch-mapping__status {
  height: 22px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 0 7px;
  border-radius: var(--dc-radius-sm);
  background: rgba(245, 158, 11, 0.12);
  color: #b45309;
  font-size: 11px;
  font-weight: 800;
}

.mqtt-batch-mapping__status.is-success {
  background: color-mix(in oklch, var(--dc-success) 12%, var(--dc-surface-raised));
  color: var(--dc-success);
}

.mqtt-batch-mapping__status.is-warning {
  background: rgba(245, 158, 11, 0.12);
  color: #b45309;
}

.mqtt-batch-mapping__status.is-danger {
  background: var(--dc-danger-soft);
  color: var(--dc-danger);
}

.mqtt-batch-mapping__status.is-muted {
  background: var(--dc-surface-muted);
  color: var(--dc-text-muted);
}

.mqtt-batch-mapping__row-actions {
  display: inline-flex;
  align-items: center;
  gap: 4px;
}

.mqtt-batch-mapping__icon-action {
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

.mqtt-batch-mapping__icon-action:hover {
  border-color: var(--dc-primary);
  color: var(--dc-primary);
}

.mqtt-batch-mapping__icon-action.is-danger:hover {
  border-color: var(--dc-danger);
  color: var(--dc-danger);
}

.mqtt-batch-mapping__icon-action svg {
  width: 14px;
  height: 14px;
}

.mqtt-batch-mapping__generate-form :deep(.el-input-number),
.mqtt-batch-mapping__generate-form :deep(.el-select) {
  width: 100%;
}

.mqtt-batch-mapping__rule-dialog-form {
  display: grid;
  gap: 14px;
}

.mqtt-batch-mapping__rule-dialog-summary {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 9px 12px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-subtle);
  color: var(--dc-text-secondary);
  font-size: 12px;
  font-weight: 700;
}

.mqtt-batch-mapping__rule-dialog-summary span {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.mqtt-batch-mapping__rule-dialog-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px 12px;
}

.mqtt-batch-mapping__rule-dialog-grid :deep(.el-form-item) {
  margin-bottom: 0;
}

.mqtt-batch-mapping__rule-dialog-grid :deep(.el-form-item__label) {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.mqtt-batch-mapping__button-icon {
  width: 14px;
  height: 14px;
  margin-right: 4px;
}

@media (max-width: 1180px) {
  .mqtt-batch-mapping__body {
    grid-template-columns: 1fr;
    overflow: auto;
  }

  .mqtt-batch-mapping__config {
    border-right: 0;
    border-bottom: 1px solid var(--dc-border);
  }

  .mqtt-batch-mapping__resize-handle {
    display: none;
  }
}

:global(.mqtt-batch-mapping--resizing) {
  cursor: col-resize;
  user-select: none;
}
</style>
