<template>
  <div class="mqtt-tag-list">
    <div class="mqtt-tag-list__toolbar">
      <div class="mqtt-tag-list__actions">
        <el-button type="primary" size="small" @click="handleCreateTag">
          <IconTablerPlus class="mqtt-tag-list__button-icon" />
          新建变量
        </el-button>
        <el-button size="small" @click="handleBatchCreate">
          <IconTablerDocumentAdd class="mqtt-tag-list__button-icon" />
          批量导入
        </el-button>
        <el-button size="small" @click="handleBatchExport">
          <IconTablerDownload class="mqtt-tag-list__button-icon" />
          批量导出
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
          变量预览/监控
        </el-button>
      </div>
      <div class="mqtt-tag-list__filters">
        <el-input
          v-model="searchKeyword"
          placeholder="搜索变量名称或解析规则"
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
          <IconTablerSortDescending
            v-if="sortOrder === 'desc'"
            class="mqtt-tag-list__sort-icon"
          />
          <IconTablerSortAscending v-else class="mqtt-tag-list__sort-icon" />
          {{ currentSortOrderLabel }}
        </button>
        <el-button size="small" :loading="loading" @click="handleRefresh">
          <IconTablerRefresh class="mqtt-tag-list__button-icon" />
          刷新
        </el-button>
      </div>
    </div>

    <div class="mqtt-tag-list__body">
      <el-table
        class="mqtt-tag-list__table"
        v-loading="loading"
        :data="tags"
        height="100%"
        row-key="id"
        empty-text="暂无变量"
      >
        <el-table-column label="变量名" min-width="170" show-overflow-tooltip>
          <template #default="{ row }">
            <div class="mqtt-tag-list__name">
              <strong>{{ row.name }}</strong>
              <span>{{ row.description || row.code }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="类型" width="92">
          <template #default="{ row }">{{ getDataTypeLabel(row.dataType) }}</template>
        </el-table-column>
        <el-table-column label="解析规则" min-width="160" show-overflow-tooltip>
          <template #default="{ row }">
            <span class="mqtt-tag-list__mono">{{ formatParseRule(row) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="数据点" min-width="190" show-overflow-tooltip>
          <template #default="{ row }">
            <span class="mqtt-tag-list__mono">{{ row.datapointPath || '-' }}</span>
          </template>
        </el-table-column>
        <el-table-column label="最近值" min-width="120" show-overflow-tooltip>
          <template #default="{ row }">
            <span class="mqtt-tag-list__value">{{ formatLastValue(row) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="创建时间" width="168">
          <template #default="{ row }">{{ formatTime(row.createdAt) }}</template>
        </el-table-column>
        <el-table-column label="状态" width="96">
          <template #default="{ row }">
            <span class="mqtt-tag-list__status" :class="statusClass(row)">
              {{ statusLabel(row) }}
            </span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="104" fixed="right">
          <template #default="{ row }">
            <div class="mqtt-tag-list__row-actions">
              <el-tooltip content="查看" placement="top">
                <button type="button" class="mqtt-tag-list__icon-action" @click="handleViewTag(row)">
                  <IconTablerEye />
                </button>
              </el-tooltip>
              <el-tooltip content="编辑" placement="top">
                <button type="button" class="mqtt-tag-list__icon-action" @click="handleEditTag(row)">
                  <IconTablerEdit />
                </button>
              </el-tooltip>
              <el-tooltip content="删除" placement="top">
                <button type="button" class="mqtt-tag-list__icon-action is-danger" @click="handleDeleteTag(row)">
                  <IconTablerTrash />
                </button>
              </el-tooltip>
            </div>
          </template>
        </el-table-column>
      </el-table>
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
import { computed, onMounted, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  deleteMqttTag,
  getDataPoints,
  getMqttTagValues,
  getMqttTags,
} from '@/api/data.api'
import { useMqttTagSync } from '@/composables/useMqttTagSync'
import MqttTagDialog from './MqttTagDialog.vue'
import IconTablerActivity from '~icons/tabler/activity'
import IconTablerDocumentAdd from '~icons/tabler/file-plus'
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
const tags = ref<any[]>([])
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
const sortFieldOptions = [
  { label: '创建时间', value: 'createdAt' },
  { label: '名称', value: 'name' },
] as const
const currentSortFieldLabel = computed(
  () => sortFieldOptions.find((option) => option.value === sortBy.value)?.label || '创建时间',
)
const currentSortOrderLabel = computed(() => (sortOrder.value === 'desc' ? '降序' : '升序'))

const loadTags = async () => {
  try {
    loading.value = true
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
  } catch (error) {
    console.error('Failed to load tags:', error)
    ElMessage.error(getApiErrorMessage(error, '加载变量失败'))
  } finally {
    loading.value = false
  }
}

const reloadAll = async () => {
  await loadTags()
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
    const response = await getMqttTagValues(props.projectId, ids)
    const list = Array.isArray(response.data) ? response.data : response.data?.list || []
    const valueMap = new Map(list.map((item) => [item.tagId, item]))
    tags.value.forEach((tag) => {
      const value = valueMap.get(tag.id)
      if (value) tag.currentValue = normalizeTagValue(value)
    })
  } catch (error) {
    console.error('Failed to load tag values:', error)
  }
}

const loadTagDatapoints = async (tagList) => {
  const ids = (tagList || []).map((tag) => tag.id).filter(Boolean)
  if (ids.length === 0) return

  try {
    const response = await getDataPoints(props.projectId, {
      type: 'mqtt.tag',
      sourceIds: ids.join(','),
      page: 1,
      pageSize: 200,
    })
    const list = response.data?.datapoints || []
    const map = new Map(list.map((item) => [item.sourceId, item]))
    tags.value.forEach((tag) => {
      const datapoint = map.get(tag.id)
      tag.datapointPath = datapoint?.path || ''
      tag.datapointStatus = datapoint?.status || ''
    })
  } catch (error) {
    console.error('Failed to load datapoints:', error)
  }
}

const handleRefresh = async () => {
  await reloadAll()
  ElMessage.success('刷新成功')
}

const handleSearch = () => {
  void reloadFirstPage()
}

const changeSortField = (value: 'createdAt' | 'name') => {
  sortBy.value = value
  sortFieldPopoverVisible.value = false
  void reloadFirstPage()
}

const toggleSortOrder = () => {
  sortOrder.value = sortOrder.value === 'desc' ? 'asc' : 'desc'
  void reloadFirstPage()
}

const changePage = async (page) => {
  pagination.value.page = page
  await loadTags()
}

const changePageSize = async (pageSize) => {
  pagination.value.pageSize = pageSize
  await reloadFirstPage()
}

const handleBatchExport = () => {
  ElMessage.info('批量导出功能待实现')
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
    await ElMessageBox.confirm(`确定要删除变量“${tag.name}”吗？`, '删除确认', {
      type: 'warning',
      confirmButtonText: '删除',
      cancelButtonText: '取消',
    })

    await deleteMqttTag(props.projectId, tag.id)
    ElMessage.success('删除成功')
    await reloadAfterMutation()
    notifyTagChange('deleted', { tagId: tag.id })
  } catch (error) {
    if (error !== 'cancel') {
      console.error('Failed to delete tag:', error)
      ElMessage.error(getApiErrorMessage(error, '删除失败'))
    }
  }
}

const handleTagDialogSuccess = async () => {
  tagDialogVisible.value = false
  await reloadAfterMutation()
  const isCreate = tagDialogMode.value === 'create'
  ElMessage.success(isCreate ? '创建成功' : '更新成功')
  notifyTagChange(isCreate ? 'created' : 'updated', {
    tagId: currentTag.value?.id,
  })
}

const handleBatchCreate = () => {
  ElMessage.info('批量导入功能开发中...')
}

const normalizeTagValue = (value) => ({
  parsedValue: value?.parsedValue ?? value?.value,
  value: value?.value ?? value?.parsedValue,
  quality: value?.quality || 'unknown',
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
    string: '字符串',
    number: '数值',
    boolean: '布尔',
    object: '对象',
    array: '数组',
  }
  return labels[dataType] || dataType || '-'
}

const getParseTypeLabel = (parseType) => {
  const labels = {
    jsonpath: 'JSONPath',
    regex: '正则',
    script: '脚本',
    fixed: '固定值',
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
  } else if (tag.dataType === 'number') {
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
  if (tag.currentValue?.quality === 'bad') return '解析异常'
  if (tag.datapointStatus === 'invalid') return '失效'
  if (tag.datapointPath) return '活跃'
  return '未生成'
}

const statusClass = (tag) => {
  if (tag.currentValue?.quality === 'bad') return 'is-danger'
  if (tag.datapointStatus === 'invalid') return 'is-muted'
  if (tag.datapointPath) return 'is-success'
  return 'is-warning'
}

watch(
  () => props.subscriptionId,
  async () => {
    searchKeyword.value = ''
    pagination.value.page = 1
    await reloadAll()
  },
)

onMounted(reloadAll)

defineExpose({
  applyTagValueUpdate,
  refresh: handleRefresh,
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
  height: 0;
  min-height: 0;
  flex: 1;
  display: flex;
  flex-direction: column;
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
  background: rgba(245, 158, 11, 0.12);
  color: #b45309;
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

@media (max-width: 980px) {
  .mqtt-tag-list__toolbar {
    align-items: flex-start;
    flex-direction: column;
  }
}
</style>
