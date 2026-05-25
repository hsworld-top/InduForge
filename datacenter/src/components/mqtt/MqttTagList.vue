<template>
  <div class="mqtt-tag-list">
    <div class="mqtt-tag-list__toolbar">
      <div class="mqtt-tag-list__actions">
        <el-button type="primary" size="small" @click="handleCreateTag">
          <IconTablerPlus class="mr-1 w-4 h-4" />
          新建变量
        </el-button>
        <el-button type="success" size="small" @click="handleCreateGroup">
          <IconTablerFolderAdd class="mr-1 w-4 h-4" />
          新建分组
        </el-button>
        <el-button size="small" @click="handleBatchCreate">
          <IconTablerDocumentAdd class="mr-1 w-4 h-4" />
          批量导入
        </el-button>
        <el-button size="small" @click="handleBatchExport">
          <IconTablerDownload class="mr-1 w-4 h-4" />
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
          <IconTablerActivity class="mr-1 w-4 h-4" />
          变量预览/监控
        </el-button>
        <el-button
          class="mqtt-tag-list__live-action"
          :class="{ 'is-connected': liveActionConnected }"
          size="small"
          type="primary"
          plain
          @click="$emit('openPublish')"
        >
          <IconTablerSend class="mr-1 w-4 h-4" />
          发布测试
        </el-button>
      </div>
      <div class="mqtt-tag-list__filters">
        <el-input
          v-model="searchKeyword"
          placeholder="搜索变量名称或标识符"
          clearable
          size="small"
          @input="handleSearch"
        >
          <template #prefix>
            <IconTablerSearch class="w-4 h-4" />
          </template>
        </el-input>
        <el-button size="small" @click="handleRefresh">
          <IconTablerRefresh class="w-4 h-4" />
          刷新
        </el-button>
      </div>
    </div>

    <div class="mqtt-tag-list__body" v-loading="loading">
      <div v-if="tags.length > 0 || groups.length > 0" class="tag-table">
        <div class="tag-table__head tag-table__row">
          <span>变量 / 分组名称</span>
          <span>标识符</span>
          <span>类型</span>
          <span>解析规则</span>
          <span>数据点</span>
          <span>最近值</span>
          <span>状态</span>
          <span>操作</span>
        </div>

        <template v-if="ungroupedTags.length > 0">
          <button
            type="button"
            class="tag-table__row tag-table__row--group"
            @click="toggleGroup('ungrouped')"
          >
            <span class="tag-table__name-cell">
              <component
                :is="activeGroups.includes('ungrouped') ? IconTablerChevronDown : IconTablerChevronRight"
                class="tag-table__chevron"
              />
              <IconTablerFolderOpened class="tag-table__folder" />
              <strong>未分组</strong>
              <em>{{ ungroupedTags.length }} 个变量</em>
            </span>
            <span>—</span>
            <span>—</span>
            <span>—</span>
            <span>—</span>
            <span>—</span>
            <span>
              <span class="tag-table__status is-neutral">GROUP</span>
            </span>
            <span class="tag-table__actions"></span>
          </button>

          <div
            v-for="tag in ungroupedTags"
            v-show="activeGroups.includes('ungrouped')"
            :key="tag.id"
            class="tag-table__row tag-table__row--tag"
          >
            <span class="tag-table__name-cell is-child">
              <IconTablerCodeDots class="tag-table__tag-icon" />
              <span class="tag-table__tag-title">
                <strong>{{ tag.name }}</strong>
                <em>{{ tag.description || tag.code }}</em>
              </span>
            </span>
            <span class="tag-table__mono" :title="tag.code">{{ tag.code }}</span>
            <span>{{ getDataTypeLabel(tag.dataType) }}</span>
            <span class="tag-table__mono" :title="formatParseRule(tag)">
              {{ formatParseRule(tag) }}
            </span>
            <span class="tag-table__mono" :title="tag.datapointPath || '-'">
              {{ tag.datapointPath || '-' }}
            </span>
            <span class="tag-table__value" :title="formatLastValue(tag, true)">
              {{ formatLastValue(tag) }}
            </span>
            <span>
              <span class="tag-table__status" :class="statusClass(tag)">
                {{ statusLabel(tag) }}
              </span>
            </span>
            <span class="tag-table__actions">
              <el-tooltip content="查看" placement="top">
                <button type="button" class="tag-table__action" @click="handleViewTag(tag)">
                  <IconTablerEye />
                </button>
              </el-tooltip>
              <el-tooltip content="编辑" placement="top">
                <button type="button" class="tag-table__action" @click="handleEditTag(tag)">
                  <IconTablerEdit />
                </button>
              </el-tooltip>
              <el-tooltip content="删除" placement="top">
                <button
                  type="button"
                  class="tag-table__action is-danger"
                  @click="handleDeleteTag(tag)"
                >
                  <IconTablerTrash />
                </button>
              </el-tooltip>
            </span>
          </div>
        </template>

        <template v-for="group in sortedGroups" :key="group.id">
          <button
            v-if="shouldRenderGroup(group)"
            type="button"
            class="tag-table__row tag-table__row--group"
            :style="getGroupStyle(group)"
            @click="toggleGroup(group.id)"
          >
            <span class="tag-table__name-cell">
              <component
                :is="activeGroups.includes(group.id) ? IconTablerChevronDown : IconTablerChevronRight"
                class="tag-table__chevron"
              />
              <IconTablerFolder class="tag-table__folder" />
              <strong>{{ group.name }}</strong>
              <em>{{ getGroupTagCount(group.id) }} 个变量</em>
            </span>
            <span class="tag-table__mono" :title="group.code || '-'">{{ group.code || '-' }}</span>
            <span>—</span>
            <span>—</span>
            <span>—</span>
            <span>—</span>
            <span>
              <span class="tag-table__status is-neutral">GROUP</span>
            </span>
            <span class="tag-table__actions" @click.stop>
              <el-tooltip content="编辑分组" placement="top">
                <button type="button" class="tag-table__action" @click="handleEditGroup(group)">
                  <IconTablerEdit />
                </button>
              </el-tooltip>
              <el-tooltip content="删除分组" placement="top">
                <button
                  type="button"
                  class="tag-table__action is-danger"
                  @click="handleDeleteGroup(group)"
                >
                  <IconTablerTrash />
                </button>
              </el-tooltip>
            </span>
          </button>

          <template v-if="activeGroups.includes(group.id)">
            <div
              v-for="tag in getGroupTags(group.id)"
              :key="tag.id"
              class="tag-table__row tag-table__row--tag"
            >
              <span class="tag-table__name-cell is-child">
                <IconTablerCodeDots class="tag-table__tag-icon" />
                <span class="tag-table__tag-title">
                  <strong>{{ tag.name }}</strong>
                  <em>{{ tag.description || tag.code }}</em>
                </span>
              </span>
              <span class="tag-table__mono" :title="tag.code">{{ tag.code }}</span>
              <span>{{ getDataTypeLabel(tag.dataType) }}</span>
              <span class="tag-table__mono" :title="formatParseRule(tag)">
                {{ formatParseRule(tag) }}
              </span>
              <span class="tag-table__mono" :title="tag.datapointPath || '-'">
                {{ tag.datapointPath || '-' }}
              </span>
              <span class="tag-table__value" :title="formatLastValue(tag, true)">
                {{ formatLastValue(tag) }}
              </span>
              <span>
                <span class="tag-table__status" :class="statusClass(tag)">
                  {{ statusLabel(tag) }}
                </span>
              </span>
              <span class="tag-table__actions">
                <el-tooltip content="查看" placement="top">
                  <button type="button" class="tag-table__action" @click="handleViewTag(tag)">
                    <IconTablerEye />
                  </button>
                </el-tooltip>
                <el-tooltip content="编辑" placement="top">
                  <button type="button" class="tag-table__action" @click="handleEditTag(tag)">
                    <IconTablerEdit />
                  </button>
                </el-tooltip>
                <el-tooltip content="删除" placement="top">
                  <button
                    type="button"
                    class="tag-table__action is-danger"
                    @click="handleDeleteTag(tag)"
                  >
                    <IconTablerTrash />
                  </button>
                </el-tooltip>
              </span>
            </div>
          </template>
        </template>
      </div>

      <div v-if="tags.length === 0 && groups.length === 0 && !loading" class="empty-state">
        <IconTablerFile />
        <p>暂无变量</p>
        <small>点击“新建变量”开始创建</small>
      </div>
    </div>

    <MqttTagDialog
      v-if="tagDialogVisible"
      :visible="tagDialogVisible"
      :tag="currentTag"
      :project-id="projectId"
      :subscription-id="subscriptionId"
      :groups="groups"
      :mode="tagDialogMode"
      @close="tagDialogVisible = false"
      @success="handleTagDialogSuccess"
    />

    <MqttTagGroupDialog
      v-if="groupDialogVisible"
      :visible="groupDialogVisible"
      :group="currentGroup"
      :project-id="projectId"
      :subscription-id="subscriptionId"
      :mode="groupDialogMode"
      @close="groupDialogVisible = false"
      @success="handleGroupDialogSuccess"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  getMqttTags,
  deleteMqttTag,
  getMqttTagGroups,
  deleteMqttTagGroup,
  getDataPoints,
  getMqttTagValues,
} from '@/api/data.api'
import { useMqttTagSync } from '@/composables/useMqttTagSync'
import MqttTagDialog from './MqttTagDialog.vue'
import MqttTagGroupDialog from './MqttTagGroupDialog.vue'
import IconTablerPlus from '~icons/tabler/plus'
import IconTablerFolderAdd from '~icons/tabler/folder-plus'
import IconTablerDocumentAdd from '~icons/tabler/file-plus'
import IconTablerRefresh from '~icons/tabler/refresh'
import IconTablerSearch from '~icons/tabler/search'
import IconTablerFolderOpened from '~icons/tabler/folder-open'
import IconTablerFolder from '~icons/tabler/folder'
import IconTablerEdit from '~icons/tabler/edit'
import IconTablerTrash from '~icons/tabler/trash'
import IconTablerEye from '~icons/tabler/eye'
import IconTablerFile from '~icons/tabler/file'
import IconTablerChevronRight from '~icons/tabler/chevron-right'
import IconTablerChevronDown from '~icons/tabler/chevron-down'
import IconTablerDownload from '~icons/tabler/download'
import IconTablerActivity from '~icons/tabler/activity'
import IconTablerSend from '~icons/tabler/send'
import IconTablerCodeDots from '~icons/tabler/code-dots'
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

defineEmits(['openMonitor', 'openPublish'])

const loading = ref(false)
const tags = ref([])
const groups = ref([])
const searchKeyword = ref('')
const activeGroups = ref(['ungrouped'])

const tagDialogVisible = ref(false)
const tagDialogMode = ref('create')
const currentTag = ref(null)

const groupDialogVisible = ref(false)
const groupDialogMode = ref('create')
const currentGroup = ref(null)

const { notify: notifyTagChange } = useMqttTagSync(props.subscriptionId)
const liveActionConnected = computed(() => Boolean(props.previewSessionId))

const sortedGroups = computed(() => {
  return [...groups.value].sort((a, b) => a.order - b.order)
})

const filteredTags = computed(() => {
  if (!searchKeyword.value) {
    return tags.value
  }
  const keyword = searchKeyword.value.toLowerCase()
  return tags.value.filter(
    (tag) => tag.name.toLowerCase().includes(keyword) || tag.code.toLowerCase().includes(keyword),
  )
})

const ungroupedTags = computed(() => {
  return filteredTags.value.filter((tag) => !tag.groupId)
})

const getGroupTags = (groupId) => {
  return filteredTags.value.filter((tag) => tag.groupId === groupId)
}

const getGroupTagCount = (groupId) => {
  return tags.value.filter((tag) => tag.groupId === groupId).length
}

const shouldRenderGroup = (group) => {
  if (!searchKeyword.value) {
    return true
  }
  return getGroupTags(group.id).length > 0
}

const getGroupStyle = (group) => {
  const baseColor = group?.color || '#3b82f6'
  return {
    '--group-color': baseColor,
    '--group-bg-color': buildGroupBackgroundColor(baseColor),
  }
}

const buildGroupBackgroundColor = (color) => {
  const normalized = String(color || '').trim()
  if (!normalized) {
    return '#f0f9ff'
  }

  const hexMatch = normalized.match(/^#([0-9a-fA-F]{6})([0-9a-fA-F]{2})?$/)
  if (hexMatch) {
    const hex = hexMatch[1]
    const r = parseInt(hex.slice(0, 2), 16)
    const g = parseInt(hex.slice(2, 4), 16)
    const b = parseInt(hex.slice(4, 6), 16)
    return `rgba(${r}, ${g}, ${b}, 0.12)`
  }

  const rgbMatch = normalized.match(/^rgba?\(\s*(\d+)\s*,\s*(\d+)\s*,\s*(\d+)/i)
  if (rgbMatch) {
    const r = Number(rgbMatch[1])
    const g = Number(rgbMatch[2])
    const b = Number(rgbMatch[3])
    return `rgba(${r}, ${g}, ${b}, 0.12)`
  }

  return '#f0f9ff'
}

const toggleGroup = (groupId) => {
  const index = activeGroups.value.indexOf(groupId)
  if (index > -1) {
    activeGroups.value.splice(index, 1)
  } else {
    activeGroups.value.push(groupId)
  }
}

const loadGroups = async () => {
  try {
    const response = await getMqttTagGroups(props.projectId, props.subscriptionId)
    groups.value = response.data?.list || []
    activeGroups.value = ['ungrouped', ...groups.value.map((group) => group.id)]
  } catch (error) {
    console.error('Failed to load tag groups:', error)
    ElMessage.error(getApiErrorMessage(error, '加载变量组失败'))
  }
}

const loadTags = async () => {
  try {
    loading.value = true
    const response = await getMqttTags(props.projectId, props.subscriptionId)
    tags.value = response.data?.list || []
    await Promise.all([loadTagDatapoints(tags.value), loadTagValues(tags.value)])
  } catch (error) {
    console.error('Failed to load tags:', error)
    ElMessage.error(getApiErrorMessage(error, '加载变量失败'))
  } finally {
    loading.value = false
  }
}

const loadTagValues = async (tagList) => {
  const ids = (tagList || []).map((tag) => tag.id).filter(Boolean)
  if (ids.length === 0) {
    return
  }

  try {
    const response = await getMqttTagValues(props.projectId, ids)
    const list = Array.isArray(response.data) ? response.data : response.data?.list || []
    const valueMap = new Map(list.map((item) => [item.tagId, item]))
    tags.value.forEach((tag) => {
      const value = valueMap.get(tag.id)
      if (value) {
        tag.currentValue = normalizeTagValue(value)
      }
    })
  } catch (error) {
    console.error('Failed to load tag values:', error)
  }
}

const loadTagDatapoints = async (tagList) => {
  const ids = (tagList || []).map((tag) => tag.id).filter(Boolean)
  if (ids.length === 0) {
    tags.value.forEach((tag) => {
      tag.datapointPath = ''
      tag.datapointStatus = ''
    })
    return
  }

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
  await Promise.all([loadGroups(), loadTags()])
  ElMessage.success('刷新成功')
}

const handleBatchExport = () => {
  ElMessage.info('批量导出功能待实现')
}

const handleCreateGroup = () => {
  currentGroup.value = null
  groupDialogMode.value = 'create'
  groupDialogVisible.value = true
}

const handleEditGroup = (group) => {
  currentGroup.value = { ...group }
  groupDialogMode.value = 'edit'
  groupDialogVisible.value = true
}

const handleDeleteGroup = async (group) => {
  try {
    await ElMessageBox.confirm(
      `确定要删除分组“${group.name}”吗？该分组下的变量将移至未分组。`,
      '删除确认',
      {
        type: 'warning',
        confirmButtonText: '删除',
        cancelButtonText: '取消',
      },
    )

    await deleteMqttTagGroup(props.projectId, group.id)
    ElMessage.success('删除成功')
    await handleRefresh()
  } catch (error) {
    if (error !== 'cancel') {
      console.error('Failed to delete group:', error)
      ElMessage.error('删除失败')
    }
  }
}

const handleGroupDialogSuccess = async () => {
  groupDialogVisible.value = false
  await loadGroups()
  ElMessage.success(groupDialogMode.value === 'create' ? '创建成功' : '更新成功')
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
    await loadTags()
    notifyTagChange('deleted', { tagId: tag.id })
  } catch (error) {
    if (error !== 'cancel') {
      console.error('Failed to delete tag:', error)
      ElMessage.error('删除失败')
    }
  }
}

const handleTagDialogSuccess = async () => {
  tagDialogVisible.value = false
  await loadTags()
  const isCreate = tagDialogMode.value === 'create'
  ElMessage.success(isCreate ? '创建成功' : '更新成功')
  notifyTagChange(isCreate ? 'created' : 'updated', {
    tagId: currentTag.value?.id,
  })
}

const handleBatchCreate = () => {
  ElMessage.info('批量导入功能开发中...')
}

const handleSearch = () => {
  // 搜索逻辑由 computed 自动处理
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
  if (!tagId) {
    return
  }
  const tag = tags.value.find((item) => item.id === tagId)
  if (!tag) {
    return
  }
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

const formatLastValue = (tag, full = false) => {
  const raw = tag.currentValue?.parsedValue ?? tag.currentValue?.value
  if (raw === null || raw === undefined || raw === '') {
    return '-'
  }

  let text = ''
  if (typeof raw === 'object') {
    text = JSON.stringify(raw)
  } else if (tag.dataType === 'number') {
    const num = Number(raw)
    text = Number.isFinite(num) ? String(Number(num.toFixed(4))) : String(raw)
  } else {
    text = String(raw)
  }

  if (tag.unit && text !== '-') {
    text = `${text} ${tag.unit}`
  }
  if (full || text.length <= 36) {
    return text
  }
  return `${text.slice(0, 35)}...`
}

const statusLabel = (tag) => {
  if (tag.currentValue?.quality === 'bad') {
    return '解析异常'
  }
  if (tag.datapointStatus === 'invalid') {
    return '失效'
  }
  if (tag.datapointPath) {
    return '活跃'
  }
  return '未生成'
}

const statusClass = (tag) => {
  if (tag.currentValue?.quality === 'bad') {
    return 'is-danger'
  }
  if (tag.datapointStatus === 'invalid') {
    return 'is-muted'
  }
  if (tag.datapointPath) {
    return 'is-success'
  }
  return 'is-warning'
}

onMounted(async () => {
  await Promise.all([loadGroups(), loadTags()])
})

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

.mqtt-tag-list__filters :deep(.el-input) {
  width: 220px;
}

.mqtt-tag-list__actions :deep(.mqtt-tag-list__live-action.el-button) {
  border-color: var(--dc-border);
  background: var(--dc-surface-raised);
  color: var(--dc-text-secondary);
}

.mqtt-tag-list__actions :deep(.mqtt-tag-list__live-action.el-button:hover),
.mqtt-tag-list__actions :deep(.mqtt-tag-list__live-action.el-button:focus) {
  border-color: var(--dc-border);
  background: var(--dc-surface-raised);
  color: var(--dc-text-secondary);
}

.mqtt-tag-list__actions :deep(.mqtt-tag-list__live-action.el-button.is-connected) {
  border-color: var(--el-color-primary);
  background: var(--el-color-primary);
  color: var(--el-color-white);
}

.mqtt-tag-list__actions :deep(.mqtt-tag-list__live-action.el-button.is-connected:hover),
.mqtt-tag-list__actions :deep(.mqtt-tag-list__live-action.el-button.is-connected:focus) {
  border-color: var(--el-color-primary);
  background: var(--el-color-primary);
  color: var(--el-color-white);
}

.mqtt-tag-list__body {
  min-height: 0;
  flex: 1;
  overflow: auto;
  padding: 10px;
}

.tag-table {
  min-width: 1040px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
  overflow: hidden;
}

.tag-table__row {
  display: grid;
  grid-template-columns:
    minmax(230px, 1.8fr) minmax(130px, 1fr) minmax(72px, 0.48fr)
    minmax(140px, 1fr) minmax(160px, 1.35fr) minmax(110px, 0.8fr)
    minmax(82px, 0.48fr) 94px;
  align-items: center;
  column-gap: 14px;
  min-height: 44px;
  padding: 0 14px;
  border-bottom: 1px solid var(--dc-border);
  color: var(--dc-text-secondary);
  font-size: 12px;
}

.tag-table__row:last-child {
  border-bottom: none;
}

.tag-table__row > span {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.tag-table__head {
  min-height: 42px;
  background: color-mix(in oklch, var(--dc-primary) 5%, var(--dc-surface-subtle));
  color: var(--dc-text);
  font-size: 12px;
  font-weight: 800;
}

.tag-table__row--group {
  width: 100%;
  border-right: 0;
  border-left: 0;
  border-top: 0;
  background: color-mix(in oklch, var(--group-bg-color, var(--dc-surface-muted)) 55%, var(--dc-surface-raised));
  color: var(--dc-text-secondary);
  text-align: left;
  cursor: pointer;
}

.tag-table__row--group:hover,
.tag-table__row--tag:hover {
  background: var(--dc-primary-soft);
}

.tag-table__row--tag {
  background: var(--dc-surface-raised);
}

.tag-table__name-cell {
  min-width: 0;
  display: inline-flex;
  align-items: center;
  gap: 8px;
}

.tag-table__name-cell strong {
  min-width: 0;
  overflow: hidden;
  color: var(--dc-text);
  font-size: 13px;
  font-weight: 800;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.tag-table__name-cell em {
  flex: 0 0 auto;
  color: var(--dc-text-muted);
  font-size: 11px;
  font-style: normal;
}

.tag-table__name-cell.is-child {
  padding-left: 32px;
  position: relative;
}

.tag-table__name-cell.is-child::before {
  position: absolute;
  left: 12px;
  top: -16px;
  bottom: 50%;
  width: 1px;
  background: var(--dc-border);
  content: '';
}

.tag-table__name-cell.is-child::after {
  position: absolute;
  left: 12px;
  top: 50%;
  width: 12px;
  height: 1px;
  background: var(--dc-border);
  content: '';
}

.tag-table__chevron,
.tag-table__folder,
.tag-table__tag-icon {
  width: 15px;
  height: 15px;
  flex: 0 0 auto;
}

.tag-table__chevron {
  color: var(--dc-text-muted);
}

.tag-table__folder {
  color: var(--group-color, var(--dc-primary));
}

.tag-table__tag-icon {
  color: var(--dc-primary);
}

.tag-table__tag-title {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.tag-table__tag-title em {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.tag-table__mono,
.tag-table__value {
  font-family: var(--dc-font-mono, ui-monospace, SFMono-Regular, Menlo, Consolas, monospace);
}

.tag-table__value {
  color: var(--dc-text);
  font-weight: 700;
}

.tag-table__status {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 58px;
  height: 20px;
  padding: 0 6px;
  border-radius: 3px;
  background: var(--dc-surface-muted);
  color: var(--dc-text-muted);
  font-size: 11px;
  font-weight: 800;
}

.tag-table__status::before {
  width: 6px;
  height: 6px;
  margin-right: 5px;
  border-radius: 999px;
  background: currentColor;
  content: '';
}

.tag-table__status.is-success {
  background: color-mix(in oklch, var(--dc-success) 12%, var(--dc-surface-raised));
  color: var(--dc-success);
}

.tag-table__status.is-warning {
  background: rgba(245, 158, 11, 0.12);
  color: #b45309;
}

.tag-table__status.is-danger {
  background: var(--dc-danger-soft);
  color: var(--dc-danger);
}

.tag-table__status.is-muted,
.tag-table__status.is-neutral {
  background: var(--dc-surface-muted);
  color: var(--dc-text-muted);
}

.tag-table__actions {
  display: inline-flex;
  align-items: center;
  justify-content: flex-end;
  gap: 4px;
}

.tag-table__action {
  width: 26px;
  height: 26px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 1px solid transparent;
  border-radius: var(--dc-radius-sm);
  background: transparent;
  color: var(--dc-text-muted);
}

.tag-table__action:hover {
  border-color: var(--dc-border);
  background: var(--dc-surface-raised);
  color: var(--dc-primary);
}

.tag-table__action.is-danger:hover {
  color: var(--dc-danger);
}

.tag-table__action svg {
  width: 14px;
  height: 14px;
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 260px;
  color: var(--dc-text-muted);
  text-align: center;
}

.empty-state svg {
  width: 42px;
  height: 42px;
  margin-bottom: 10px;
  opacity: 0.72;
}

.empty-state p {
  margin: 0;
  color: var(--dc-text-secondary);
  font-size: 14px;
  font-weight: 700;
}

.empty-state small {
  margin-top: 4px;
  font-size: 12px;
}

@media (max-width: 980px) {
  .mqtt-tag-list__toolbar {
    align-items: flex-start;
    flex-direction: column;
  }
}
</style>
