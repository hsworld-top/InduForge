<template>
  <div class="mqtt-tag-list">
    <div class="mqtt-tag-list__toolbar">
      <div class="mqtt-tag-list__actions">
        <el-button type="primary" size="small" @click="handleCreateTag">
          <IconTablerPlus class="mqtt-tag-list__button-icon" />
          {{ ui('新建变量', 'New Variable') }}
        </el-button>
        <el-button size="small" :loading="exporting" @click="handleBatchExport">
          <IconTablerDownload class="mqtt-tag-list__button-icon" />
          {{ ui('导出结果', 'Export Results') }}
        </el-button>
        <el-button
          class="mqtt-tag-list__live-action"
          :class="{ 'is-connected': liveActionConnected }"
          size="small"
          type="primary"
          plain
          @click="$emit('openMonitor')"
        >
          <IconTablerActivity class="mqtt-tag-list__button-icon" />
          {{ ui('变量预览/监控', 'Variable Preview / Monitor') }}
        </el-button>
      </div>
      <div class="mqtt-tag-list__filters">
        <el-input
          v-model="searchKeyword"
          :placeholder="ui('搜索变量名称或解析规则', 'Search variable name or parse rule')"
          clearable
          size="small"
          @input="handleSearch"
        >
          <template #prefix>
            <IconTablerSearch class="mqtt-tag-list__input-icon" />
          </template>
        </el-input>
        <el-popover
          v-model:visible="sortFieldPopoverVisible"
          placement="bottom-start"
          :width="148"
          trigger="click"
        >
          <template #reference>
            <button type="button" class="mqtt-tag-list__sort-pill">
              {{ currentSortFieldLabel }}
            </button>
          </template>
          <div class="mqtt-tag-list__sort-menu">
            <button
              v-for="option in sortFieldOptions"
              :key="option.value"
              type="button"
              class="mqtt-tag-list__sort-item"
              :class="{ 'is-active': sortBy === option.value }"
              @click="changeSortField(option.value)"
            >
              {{ option.label }}
            </button>
          </div>
        </el-popover>
        <button type="button" class="mqtt-tag-list__sort-pill" @click="toggleSortOrder">
          <IconTablerSortDescending v-if="sortOrder === 'desc'" class="mqtt-tag-list__sort-icon" />
          <IconTablerSortAscending v-else class="mqtt-tag-list__sort-icon" />
          {{ currentSortOrderLabel }}
        </button>
        <el-button size="small" :loading="loading" @click="handleRefresh">
          <IconTablerRefresh class="mqtt-tag-list__button-icon" />
          {{ ui('刷新', 'Refresh') }}
        </el-button>
      </div>
    </div>

    <div class="mqtt-tag-list__body">
      <el-table
        ref="tagTableRef"
        class="mqtt-tag-list__table"
        v-loading="loading"
        :data="tags"
        height="100%"
        row-key="id"
        :empty-text="ui('暂无变量', 'No variables')"
        @selection-change="handleSelectionChange"
      >
        <el-table-column type="selection" width="42" reserve-selection />
        <el-table-column :label="ui('变量名', 'Variable')" min-width="170" show-overflow-tooltip>
          <template #default="{ row }">
            <div class="mqtt-tag-list__name">
              <strong>{{ row.name }}</strong>
              <span>{{ row.description || row.code }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column :label="ui('类型', 'Type')" width="92">
          <template #default="{ row }">{{ getDataTypeLabel(row.dataType) }}</template>
        </el-table-column>
        <el-table-column :label="ui('解析规则', 'Parse Rule')" min-width="160" show-overflow-tooltip>
          <template #default="{ row }">
            <span class="mqtt-tag-list__mono">{{ formatParseRule(row) }}</span>
          </template>
        </el-table-column>
        <el-table-column :label="ui('数据点', 'Data Point')" min-width="190" show-overflow-tooltip>
          <template #default="{ row }">
            <span class="mqtt-tag-list__mono">{{ row.datapointPath || '-' }}</span>
          </template>
        </el-table-column>
        <el-table-column :label="ui('最近值', 'Latest Value')" min-width="120" show-overflow-tooltip>
          <template #default="{ row }">
            <span class="mqtt-tag-list__value">{{ formatLastValue(row) }}</span>
          </template>
        </el-table-column>
        <el-table-column :label="ui('创建时间', 'Created At')" width="168">
          <template #default="{ row }">{{ formatTime(row.createdAt) }}</template>
        </el-table-column>
        <el-table-column :label="ui('状态', 'Status')" width="96">
          <template #default="{ row }">
            <span class="mqtt-tag-list__status" :class="statusClass(row)">
              {{ statusLabel(row) }}
            </span>
          </template>
        </el-table-column>
        <el-table-column :label="ui('操作', 'Actions')" width="104" fixed="right">
          <template #default="{ row }">
            <div class="mqtt-tag-list__row-actions">
              <el-tooltip :content="ui('查看', 'View')" placement="top">
                <button
                  type="button"
                  class="mqtt-tag-list__icon-action"
                  @click="handleViewTag(row)"
                >
                  <IconTablerEye />
                </button>
              </el-tooltip>
              <el-tooltip :content="ui('编辑', 'Edit')" placement="top">
                <button
                  type="button"
                  class="mqtt-tag-list__icon-action"
                  @click="handleEditTag(row)"
                >
                  <IconTablerEdit />
                </button>
              </el-tooltip>
              <el-tooltip :content="ui('删除', 'Delete')" placement="top">
                <button
                  type="button"
                  class="mqtt-tag-list__icon-action is-danger"
                  @click="handleDeleteTag(row)"
                >
                  <IconTablerTrash />
                </button>
              </el-tooltip>
            </div>
          </template>
        </el-table-column>
      </el-table>
      <BulkActionBar :selected-count="selectedTagCount" @clear="clearTagSelection">
        <button type="button" class="mqtt-tag-list__bulk-action-btn" @click="selectCurrentPage">
          {{ ui('当前页', 'Current Page') }}
        </button>
        <button type="button" class="mqtt-tag-list__bulk-action-btn" @click="selectAllResults">
          {{ ui('全部结果', 'All Results') }}
        </button>
        <button
          type="button"
          class="mqtt-tag-list__bulk-action-btn is-danger"
          @click="handleBatchDeleteTags"
        >
          {{ ui('删除选中', 'Delete Selected') }}
        </button>
      </BulkActionBar>
      <div class="mqtt-tag-list__pagination">
        <el-pagination
          :current-page="pagination.page"
          :page-size="pagination.pageSize"
          :page-sizes="[20, 50, 100]"
          :total="pagination.total"
          background
          layout="total, sizes, prev, pager, next, jumper"
          small
          @current-change="changePage"
          @size-change="changePageSize"
        />
      </div>
    </div>

    <MqttTagDialog
      v-if="tagDialogVisible"
      :visible="tagDialogVisible"
      :tag="currentTag"
      :project-id="projectId"
      :subscription-id="subscriptionId"
      :mode="tagDialogMode"
      @close="tagDialogVisible = false"
      @success="handleTagDialogSuccess"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, ref, watch } from 'vue'
import { debugLogger } from '@/utils/debug'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  deleteMqttTag,
  deleteMqttTagsByFilter,
  deleteMqttTagsBatch,
  getDataPointStatuses,
  getMqttTagValues,
  getMqttTags,
} from '@/api/data.api'
import { useMqttTagSync } from '@/composables/useMqttTagSync'
import MqttTagDialog from './MqttTagDialog.vue'
import BulkActionBar from '@/components/shared/BulkActionBar.vue'
import IconTablerActivity from '~icons/tabler/activity'
import IconTablerDownload from '~icons/tabler/download'
import IconTablerEdit from '~icons/tabler/edit'
import IconTablerEye from '~icons/tabler/eye'
import IconTablerPlus from '~icons/tabler/plus'
import IconTablerRefresh from '~icons/tabler/refresh'
import IconTablerSearch from '~icons/tabler/search'
import IconTablerSortAscending from '~icons/tabler/sort-ascending'
import IconTablerSortDescending from '~icons/tabler/sort-descending'
import IconTablerTrash from '~icons/tabler/trash'
import dayjs from 'dayjs'
import { TIME_FORMAT } from '@/constants'
import { getApiErrorMessage } from '@/utils/request'
import { downloadCsv } from '@/utils/tabular-file'
import { datacenterLocale } from '@/i18n/runtime'

const ui = (zh: string, en: string) => (datacenterLocale.value === 'en' ? en : zh)

const props = defineProps({
  projectId: {
    type: String,
    required: true,
  },
  subscriptionId: {
    type: String,
    required: true,
  },
  previewSessionId: {
    type: String,
    default: '',
  },
})

defineEmits(['openMonitor'])

const loading = ref(false)
const exporting = ref(false)
const tags = ref<any[]>([])
const selectedTags = ref<any[]>([])
const allTagResultsSelected = ref(false)
const tagTableRef = ref()
const syncingTagSelection = ref(false)
const searchKeyword = ref('')
const pagination = ref({ page: 1, pageSize: 20, total: 0, totalPages: 0 })
const sortBy = ref<'createdAt' | 'name'>('createdAt')
const sortOrder = ref<'asc' | 'desc'>('desc')
const sortFieldPopoverVisible = ref(false)

const tagDialogVisible = ref(false)
const tagDialogMode = ref('create')
const currentTag = ref<any | null>(null)

const { notify: notifyTagChange } = useMqttTagSync(props.subscriptionId)
const liveActionConnected = computed(() => Boolean(props.previewSessionId))
const sortFieldOptions = computed(() => [
  { label: ui('创建时间', 'Created At'), value: 'createdAt' },
  { label: ui('名称', 'Name'), value: 'name' },
] as const)
const LOADING_DELAY_MS = 180
const currentSortFieldLabel = computed(
  () => sortFieldOptions.value.find((option) => option.value === sortBy.value)?.label || ui('创建时间', 'Created At'),
)
const currentSortOrderLabel = computed(() => (sortOrder.value === 'desc' ? ui('降序', 'Descending') : ui('升序', 'Ascending')))
const selectedTagCount = computed(() =>
  allTagResultsSelected.value ? pagination.value.total : selectedTags.value.length,
)

const loadTags = async (options: { silent?: boolean } = {}) => {
  let loadingTimer: ReturnType<typeof window.setTimeout> | null = null
  try {
    if (!options.silent) {
      loadingTimer = window.setTimeout(() => {
        loading.value = true
      }, LOADING_DELAY_MS)
    }
    const params = {
      page: pagination.value.page,
      pageSize: pagination.value.pageSize,
      q: searchKeyword.value.trim() || undefined,
      sortBy: sortBy.value,
      sortOrder: sortOrder.value,
    }
    const response = await getMqttTags(props.projectId, props.subscriptionId, params)
    tags.value = response.data?.list || []
    const pageInfo = response.data?.pagination || {}
    pagination.value = {
      page: pageInfo.page || pagination.value.page,
      pageSize: pageInfo.pageSize || pagination.value.pageSize,
      total: pageInfo.total || 0,
      totalPages: pageInfo.totalPages || 0,
    }
    await Promise.all([loadTagDatapoints(tags.value), loadTagValues(tags.value)])
    await syncTagSelection()
  } catch (error) {
    debugLogger.error('Failed to load tags:', error)
    ElMessage.error(getApiErrorMessage(error, ui('加载变量失败', 'Failed to load variables')))
  } finally {
    if (loadingTimer) {
      window.clearTimeout(loadingTimer)
    }
    if (!options.silent) {
      loading.value = false
    }
  }
}

const reloadAll = async () => {
  await loadTags()
}

const reloadQuietly = async () => {
  await loadTags({ silent: true })
}

const reloadFirstPage = async () => {
  pagination.value.page = 1
  await loadTags()
}

const reloadAfterMutation = async () => {
  await loadTags()
  if (pagination.value.page > 1 && tags.value.length === 0 && pagination.value.total > 0) {
    pagination.value.page -= 1
    await loadTags()
  }
}

const loadTagValues = async (tagList) => {
  const ids = (tagList || []).map((tag) => tag.id).filter(Boolean)
  if (ids.length === 0) return

  try {
    const response = await getMqttTagValues(props.projectId, ids, { compact: true })
    const list = Array.isArray(response.data) ? response.data : response.data?.list || []
    const valueMap = new Map(list.map((item) => [item.tagId, item]))
    tags.value.forEach((tag) => {
      const value = valueMap.get(tag.id)
      if (value) tag.currentValue = normalizeTagValue(value)
    })
  } catch (error) {
    debugLogger.error('Failed to load tag values:', error)
  }
}

const loadTagDatapoints = async (tagList) => {
  const ids = (tagList || []).map((tag) => tag.id).filter(Boolean)
  if (ids.length === 0) return

  try {
    const response = await getDataPointStatuses(props.projectId, {
      sourceIds: ids,
    })
    const list = response.data?.datapoints || []
    const map = new Map(list.map((item) => [item.sourceId, item]))
    tags.value.forEach((tag) => {
      const datapoint = map.get(tag.id)
      tag.datapointPath = datapoint?.path || ''
      tag.datapointStatus = datapoint?.status || ''
    })
  } catch (error) {
    debugLogger.error('Failed to load datapoints:', error)
  }
}

const handleRefresh = async () => {
  await reloadAll()
  ElMessage.success(ui('刷新成功', 'Refreshed successfully'))
}

const handleSearch = () => {
  clearTagSelection()
  void reloadFirstPage()
}

const changeSortField = (value: 'createdAt' | 'name') => {
  sortBy.value = value
  sortFieldPopoverVisible.value = false
  clearTagSelection()
  void reloadFirstPage()
}

const toggleSortOrder = () => {
  sortOrder.value = sortOrder.value === 'desc' ? 'asc' : 'desc'
  clearTagSelection()
  void reloadFirstPage()
}

const changePage = async (page) => {
  pagination.value.page = page
  await loadTags()
}

const changePageSize = async (pageSize) => {
  pagination.value.pageSize = pageSize
  if (!allTagResultsSelected.value) {
    clearTagSelection()
  }
  await reloadFirstPage()
}

const handleBatchExport = async () => {
  if (exporting.value) return
  exporting.value = true
  try {
    const pageSize = 100
    const firstResponse = await getMqttTags(props.projectId, props.subscriptionId, {
      page: 1,
      pageSize,
      q: searchKeyword.value.trim() || undefined,
      sortBy: sortBy.value,
      sortOrder: sortOrder.value,
    })
    const firstPage = firstResponse.data?.list || []
    const totalPages = Number(firstResponse.data?.pagination?.totalPages || 1)
    const remainingResponses = []
    for (let page = 2; page <= totalPages; page += 1) {
      remainingResponses.push(
        await getMqttTags(props.projectId, props.subscriptionId, {
          page,
          pageSize,
          q: searchKeyword.value.trim() || undefined,
          sortBy: sortBy.value,
          sortOrder: sortOrder.value,
        }),
      )
    }
    const exportedTags = [
      ...firstPage,
      ...remainingResponses.flatMap((response) => response.data?.list || []),
    ]
    const statusMap = new Map<string, any>()
    for (let offset = 0; offset < exportedTags.length; offset += 500) {
      const sourceIds = exportedTags.slice(offset, offset + 500).map((tag) => tag.id)
      const statusResponse = await getDataPointStatuses(props.projectId, { sourceIds })
      ;(statusResponse.data?.datapoints || []).forEach((item) => {
        statusMap.set(item.sourceId, item)
      })
    }
    const columns = {
      name: ui('变量名', 'Variable Name'),
      code: ui('编码', 'Code'),
      description: ui('描述', 'Description'),
      dataType: ui('数据类型', 'Data Type'),
      parseType: ui('解析类型', 'Parse Type'),
      parseRule: ui('解析规则', 'Parse Rule'),
      unit: ui('单位', 'Unit'),
      status: ui('状态', 'Status'),
      datapointPath: ui('数据点路径', 'Data Point Path'),
      createdAt: ui('创建时间', 'Created At'),
    }
    const headers = Object.values(columns)
    const rows = exportedTags.map((tag) => {
      const datapoint = statusMap.get(tag.id)
      return {
        [columns.name]: tag.name,
        [columns.code]: tag.code,
        [columns.description]: tag.description || '',
        [columns.dataType]: getDataTypeLabel(tag.dataType),
        [columns.parseType]: getParseTypeLabel(tag.parseType),
        [columns.parseRule]: tag.parseRule || '',
        [columns.unit]: tag.unit || '',
        [columns.status]: datapoint?.status === 'invalid' ? ui('失效', 'Invalid') : datapoint?.path ? ui('活跃', 'Active') : ui('未生成', 'Not Generated'),
        [columns.datapointPath]: datapoint?.path || '',
        [columns.createdAt]: formatTime(tag.createdAt),
      }
    })
    downloadCsv(`mqtt-variables-${dayjs().format('YYYYMMDD-HHmmss')}.csv`, headers, rows)
    ElMessage.success(ui(`已导出 ${rows.length} 条变量`, `Exported ${rows.length} variables`))
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, ui('导出变量失败', 'Failed to export variables')))
  } finally {
    exporting.value = false
  }
}

const handleCreateTag = () => {
  currentTag.value = null
  tagDialogMode.value = 'create'
  tagDialogVisible.value = true
}

const handleEditTag = (tag) => {
  currentTag.value = { ...tag }
  tagDialogMode.value = 'edit'
  tagDialogVisible.value = true
}

const handleViewTag = (tag) => {
  currentTag.value = { ...tag }
  tagDialogMode.value = 'view'
  tagDialogVisible.value = true
}

const handleDeleteTag = async (tag) => {
  try {
    await ElMessageBox.confirm(ui(`确定要删除变量“${tag.name}”吗？`, `Delete variable “${tag.name}”?`), ui('删除确认', 'Confirm Delete'), {
      type: 'warning',
      confirmButtonText: ui('删除', 'Delete'),
      cancelButtonText: ui('取消', 'Cancel'),
    })

    await deleteMqttTag(props.projectId, tag.id)
    ElMessage.success(ui('删除成功', 'Deleted successfully'))
    await reloadAfterMutation()
    notifyTagChange('deleted', { tagId: tag.id })
  } catch (error) {
    if (error !== 'cancel') {
      debugLogger.error('Failed to delete tag:', error)
      ElMessage.error(getApiErrorMessage(error, ui('删除失败', 'Failed to delete variable')))
    }
  }
}

const handleSelectionChange = (rows) => {
  if (syncingTagSelection.value) return
  allTagResultsSelected.value = false
  const visibleIds = new Set(tags.value.map((tag) => tag.id))
  const retainedRows = selectedTags.value.filter((tag) => !visibleIds.has(tag.id))
  selectedTags.value = [...retainedRows, ...(rows || [])]
}

const clearTagSelection = () => {
  allTagResultsSelected.value = false
  selectedTags.value = []
  void syncTagSelection()
}

const selectCurrentPage = async () => {
  allTagResultsSelected.value = false
  selectedTags.value = [...tags.value]
  await syncTagSelection()
}

const selectAllResults = async () => {
  allTagResultsSelected.value = true
  selectedTags.value = [...tags.value]
  await syncTagSelection()
}

const syncTagSelection = async () => {
  await nextTick()
  const table = tagTableRef.value
  if (!table) return
  syncingTagSelection.value = true
  table.clearSelection()
  const selectedIds = new Set(selectedTags.value.map((tag) => tag.id))
  tags.value.forEach((tag) => {
    if (allTagResultsSelected.value || selectedIds.has(tag.id)) {
      table.toggleRowSelection(tag, true)
    }
  })
  await nextTick()
  syncingTagSelection.value = false
}

const handleBatchDeleteTags = async () => {
  if (allTagResultsSelected.value) {
    const deleteCount = pagination.value.total
    if (deleteCount === 0) return
    try {
      await ElMessageBox.confirm(
        ui(`确定要删除当前筛选结果中的 ${deleteCount} 个变量吗？`, `Delete all ${deleteCount} variables in the current filtered results?`),
        ui('批量删除确认', 'Confirm Bulk Delete'),
        {
          type: 'warning',
          confirmButtonText: ui('删除', 'Delete'),
          cancelButtonText: ui('取消', 'Cancel'),
        },
      )
      const response = await deleteMqttTagsByFilter(props.projectId, props.subscriptionId, {
        search: searchKeyword.value.trim(),
      })
      const deletedCount = response?.data?.deletedCount ?? 0
      ElMessage.success(ui(`已删除 ${deletedCount} 个变量`, `Deleted ${deletedCount} variables`))
      clearTagSelection()
      await reloadAfterMutation()
      notifyTagChange('deleted', { filtered: true, deletedCount })
    } catch (error) {
      if (error !== 'cancel') {
        debugLogger.error('Failed to delete filtered tags:', error)
        ElMessage.error(getApiErrorMessage(error, ui('批量删除失败', 'Bulk delete failed')))
      }
    }
    return
  }
  const tagIds = selectedTags.value.map((tag) => tag.id).filter(Boolean)
  if (tagIds.length === 0) return

  try {
    await ElMessageBox.confirm(ui(`确定要删除选中的 ${tagIds.length} 个变量吗？`, `Delete the ${tagIds.length} selected variables?`), ui('批量删除确认', 'Confirm Bulk Delete'), {
      type: 'warning',
      confirmButtonText: ui('删除', 'Delete'),
      cancelButtonText: ui('取消', 'Cancel'),
    })

    await deleteMqttTagsBatch(props.projectId, tagIds)
    ElMessage.success(ui('批量删除成功', 'Variables deleted successfully'))
    clearTagSelection()
    await reloadAfterMutation()
    notifyTagChange('deleted', { tagIds })
  } catch (error) {
    if (error !== 'cancel') {
      debugLogger.error('Failed to delete selected tags:', error)
      ElMessage.error(getApiErrorMessage(error, ui('批量删除失败', 'Bulk delete failed')))
    }
  }
}

const handleTagDialogSuccess = async () => {
  tagDialogVisible.value = false
  await reloadAfterMutation()
  const isCreate = tagDialogMode.value === 'create'
  ElMessage.success(isCreate ? ui('创建成功', 'Created successfully') : ui('更新成功', 'Updated successfully'))
  notifyTagChange(isCreate ? 'created' : 'updated', {
    tagId: currentTag.value?.id,
  })
}

const normalizeTagValue = (value) => ({
  parsedValue: value?.parsedValue ?? value?.value,
  value: value?.value ?? value?.parsedValue,
  quality: value?.quality || 'unknown',
  qualityCode: value?.qualityCode,
  timestamp: value?.timestamp || value?.receivedAt || '',
  error: value?.error || '',
})

const applyTagValueUpdate = (value) => {
  const tagId = value?.tagId
  if (!tagId) return
  const tag = tags.value.find((item) => item.id === tagId)
  if (!tag) return
  tag.currentValue = normalizeTagValue(value)
}

const getDataTypeLabel = (dataType) => {
  const labels = {
    string: ui('字符串', 'String'),
    float64: ui('数值', 'Number'),
    bool: ui('布尔', 'Boolean'),
    object: ui('对象', 'Object'),
    array: ui('数组', 'Array'),
  }
  return labels[dataType] || dataType || '-'
}

const getParseTypeLabel = (parseType) => {
  const labels = {
    jsonpath: 'JSONPath',
    regex: ui('正则', 'Regex'),
    script: ui('脚本', 'Script'),
    fixed: ui('固定值', 'Fixed Value'),
    batch_jsonpath: ui('批量映射', 'Batch Mapping'),
  }
  return labels[parseType] || parseType || '-'
}

const formatParseRule = (tag) => {
  const type = getParseTypeLabel(tag.parseType)
  const rule = String(tag.parseRule || '').trim()
  return rule ? `${type}: ${rule}` : type
}

const formatLastValue = (tag) => {
  const raw = tag.currentValue?.parsedValue ?? tag.currentValue?.value
  if (raw === null || raw === undefined || raw === '') return '-'
  let text = ''
  if (typeof raw === 'object') {
    text = JSON.stringify(raw)
  } else if (tag.dataType === 'float64') {
    const num = Number(raw)
    text = Number.isFinite(num) ? String(Number(num.toFixed(4))) : String(raw)
  } else {
    text = String(raw)
  }
  return tag.unit && text !== '-' ? `${text} ${tag.unit}` : text
}

const formatTime = (value) => {
  if (!value) return '-'
  const time = dayjs(value)
  return time.isValid() ? time.format(TIME_FORMAT) : '-'
}

const statusLabel = (tag) => {
  if (tag.datapointStatus === 'invalid') return ui('失效', 'Invalid')
  if (tag.datapointPath) return ui('活跃', 'Active')
  return ui('未生成', 'Not Generated')
}

const statusClass = (tag) => {
  if (tag.datapointStatus === 'invalid') return 'is-muted'
  if (tag.datapointPath) return 'is-success'
  return 'is-warning'
}

const toMonitorTagSnapshot = (tag) => ({
  id: tag.id,
  name: tag.name,
  code: tag.code,
  dataType: tag.dataType,
  currentValue: tag.currentValue,
  createdAt: tag.createdAt,
})

const getMonitorSnapshot = () => ({
  mode: 'single',
  page: pagination.value.page,
  pageSize: pagination.value.pageSize,
  total: pagination.value.total,
  q: searchKeyword.value.trim(),
  sortBy: sortBy.value,
  sortOrder: sortOrder.value,
  rows: tags.value.map(toMonitorTagSnapshot),
})

watch(
  () => props.subscriptionId,
  async () => {
    searchKeyword.value = ''
    pagination.value.page = 1
    clearTagSelection()
    await reloadAll()
  },
)

onMounted(reloadAll)

defineExpose({
  applyTagValueUpdate,
  getMonitorSnapshot,
  refresh: handleRefresh,
  refreshQuietly: reloadQuietly,
})
</script>

<style scoped>
.mqtt-tag-list {
  height: 100%;
  min-height: 0;
  display: flex;
  flex-direction: column;
  background: var(--dc-surface-raised);
}

.mqtt-tag-list__toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 10px 12px;
  border-bottom: 1px solid var(--dc-border);
  background: var(--dc-surface-subtle);
}

.mqtt-tag-list__actions,
.mqtt-tag-list__filters {
  display: flex;
  align-items: center;
  gap: 7px;
  flex-wrap: wrap;
}

.mqtt-tag-list__button-icon,
.mqtt-tag-list__input-icon {
  width: 14px;
  height: 14px;
  margin-right: 4px;
}

.mqtt-tag-list__input-icon {
  margin-right: 0;
}

.mqtt-tag-list__filters :deep(.el-input) {
  width: 220px;
}

.mqtt-tag-list__sort-pill {
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

.mqtt-tag-list__sort-pill:hover {
  border-color: color-mix(in oklch, var(--dc-primary) 34%, var(--dc-border));
  color: var(--dc-primary);
}

.mqtt-tag-list__sort-icon {
  width: 13px;
  height: 13px;
}

.mqtt-tag-list__sort-menu {
  display: grid;
  gap: 4px;
}

.mqtt-tag-list__sort-item {
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

.mqtt-tag-list__sort-item:hover,
.mqtt-tag-list__sort-item.is-active {
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
}

.mqtt-tag-list__actions :deep(.mqtt-tag-list__live-action.el-button) {
  border-color: var(--dc-border);
  background: var(--dc-surface-raised);
  color: var(--dc-text-secondary);
}

.mqtt-tag-list__actions :deep(.mqtt-tag-list__live-action.el-button.is-connected) {
  border-color: var(--el-color-primary);
  background: var(--el-color-primary);
  color: var(--el-color-white);
}

.mqtt-tag-list__body {
  position: relative;
  height: 0;
  min-height: 0;
  flex: 1;
  display: flex;
  flex-direction: column;
}

.mqtt-tag-list__body :deep(.dc-bulk-action-bar) {
  bottom: 58px;
}

.mqtt-tag-list__table {
  flex: 1;
  min-height: 0;
}

.mqtt-tag-list__name {
  min-width: 0;
  display: grid;
  gap: 2px;
}

.mqtt-tag-list__name strong {
  overflow: hidden;
  color: var(--dc-text);
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.mqtt-tag-list__name span,
.mqtt-tag-list__mono {
  overflow: hidden;
  color: var(--dc-text-muted);
  font-size: 11px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.mqtt-tag-list__mono,
.mqtt-tag-list__value {
  font-family: var(--dc-font-mono, ui-monospace, SFMono-Regular, Menlo, Consolas, monospace);
}

.mqtt-tag-list__value {
  color: var(--dc-text);
  font-weight: 700;
}

.mqtt-tag-list__status {
  min-width: 58px;
  height: 22px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 0 7px;
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-muted);
  color: var(--dc-text-muted);
  font-size: 11px;
  font-weight: 800;
}

.mqtt-tag-list__status.is-success {
  background: color-mix(in oklch, var(--dc-success) 12%, var(--dc-surface-raised));
  color: var(--dc-success);
}

.mqtt-tag-list__status.is-warning {
  background: color-mix(in oklch, var(--dc-warning) 14%, var(--dc-surface-raised));
  color: var(--dc-warning);
}

.mqtt-tag-list__status.is-danger {
  background: var(--dc-danger-soft);
  color: var(--dc-danger);
}

.mqtt-tag-list__status.is-muted {
  background: var(--dc-surface-muted);
  color: var(--dc-text-muted);
}

.mqtt-tag-list__row-actions {
  display: inline-flex;
  align-items: center;
  gap: 4px;
}

.mqtt-tag-list__icon-action {
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

.mqtt-tag-list__icon-action:hover {
  border-color: var(--dc-primary);
  color: var(--dc-primary);
}

.mqtt-tag-list__icon-action.is-danger:hover {
  border-color: var(--dc-danger);
  color: var(--dc-danger);
}

.mqtt-tag-list__icon-action svg {
  width: 14px;
  height: 14px;
}

.mqtt-tag-list__pagination {
  min-height: 42px;
  display: flex;
  align-items: center;
  justify-content: flex-end;
  padding: 6px 12px;
  border-top: 1px solid var(--dc-border);
  background: var(--dc-surface-raised);
}

.mqtt-tag-list__bulk-action-btn {
  max-width: 92px;
  height: 28px;
  min-width: 0;
  overflow: hidden;
  padding: 0 10px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
  color: var(--dc-text);
  cursor: pointer;
  font-family: inherit;
  font-size: 12px;
  font-weight: 700;
  text-align: center;
  text-overflow: ellipsis;
  white-space: nowrap;
  transition:
    background 0.18s ease,
    border-color 0.18s ease,
    color 0.18s ease;
}

.mqtt-tag-list__bulk-action-btn:hover {
  border-color: rgba(29, 78, 216, 0.26);
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
}

.mqtt-tag-list__bulk-action-btn.is-danger:hover {
  border-color: rgba(220, 38, 38, 0.26);
  background: rgba(220, 38, 38, 0.08);
  color: var(--dc-danger);
}

@media (max-width: 980px) {
  .mqtt-tag-list__toolbar {
    align-items: flex-start;
    flex-direction: column;
  }
}
</style>
