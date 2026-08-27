<template>
  <div class="mqtt-tag-monitor">
    <div class="mqtt-tag-monitor__body">
      <div v-if="totalTags === 0" class="mqtt-tag-monitor__empty">
        <IconTablerFile />
        <div>暂无变量</div>
        <small>请先在订阅的变量配置中创建变量</small>
      </div>

      <div v-else-if="viewMode === 'card'" v-loading="loading" class="tag-card-grid">
        <div
          v-for="tag in tags"
          :key="tag.id"
          class="tag-card"
          :class="`is-${tag.currentValue?.quality || 'unknown'}`"
        >
          <div class="tag-card__header">
            <div class="tag-card__name" :title="tag.name">
              {{ tag.name }}
            </div>
            <WorkbenchStatusPill
              v-if="showQualityFields"
              :label="formatQualityLabel(tag.currentValue)"
              :tone="getQualityTone(tag.currentValue?.quality || 'unknown')"
            />
          </div>

          <div class="tag-value">
            {{ tag.currentValue ? formatTagValue(tag.currentValue, tag.dataType) : '-' }}
          </div>

          <div v-if="showQualityFields" class="tag-card__time">
            <span>更新时间</span>
            <strong>{{
              tag.currentValue?.timestamp ? formatTimestamp(tag.currentValue.timestamp) : '-'
            }}</strong>
          </div>
        </div>
      </div>

      <div v-else v-loading="loading" class="tag-row-list">
        <div class="tag-row tag-row-header" :class="{ 'is-single': !showQualityFields }">
          <span>变量名</span>
          <span>类型</span>
          <span>当前值</span>
          <span v-if="showQualityFields">时间戳</span>
          <span v-if="showQualityFields">质量</span>
        </div>
        <div
          v-for="tag in tags"
          :key="tag.id"
          class="tag-row"
          :class="{ 'is-single': !showQualityFields }"
        >
          <span class="truncate" :title="tag.name">{{ tag.name }}</span>
          <span>{{ getDataTypeLabel(tag.dataType) }}</span>
          <span class="truncate">
            {{ tag.currentValue ? formatTagValue(tag.currentValue, tag.dataType) : '-' }}
          </span>
          <span v-if="showQualityFields" class="truncate">
            {{ tag.currentValue?.timestamp ? formatTimestamp(tag.currentValue.timestamp) : '-' }}
          </span>
          <span v-if="showQualityFields">
            <el-tag :type="getQualityColor(tag.currentValue?.quality || 'unknown')" size="small">
              {{ formatQualityLabel(tag.currentValue) }}
            </el-tag>
          </span>
        </div>
      </div>
    </div>

    <div v-if="totalTags > 0" class="mqtt-tag-monitor__pagination">
      <el-pagination
        v-model:current-page="pagination.page"
        v-model:page-size="pagination.pageSize"
        :page-sizes="pageSizes"
        :total="totalTags"
        background
        layout="total, sizes, prev, pager, next, jumper"
        small
        @size-change="handlePageSizeChange"
        @current-change="handlePageChange"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref, onMounted, onBeforeUnmount, watch, toRef } from 'vue'
import { ElMessage } from 'element-plus'
import { getMqttTags } from '@/api/data.api'
import { useMqttSocket } from '@/composables/useMqttSocket'
import { useMqttTagSync } from '@/composables/useMqttTagSync'
import WorkbenchStatusPill from '@/components/workbench/WorkbenchStatusPill.vue'
import IconTablerFile from '~icons/tabler/file'
import dayjs from 'dayjs'
import { TIME_FORMAT } from '@/constants'

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
  snapshot: {
    type: Object,
    default: null,
  },
})

const emit = defineEmits<{
  (event: 'tag-value', value: any): void
  (event: 'latest-values', values: any[]): void
}>()

const tags = ref([])
const loading = ref(false)
const viewMode = ref('list')
const pagination = reactive({
  page: Number(props.snapshot?.page) || 1,
  pageSize: Number(props.snapshot?.pageSize) || 50,
  total: Number(props.snapshot?.total) || 0,
})
const latestValueMap = new Map()
const totalTags = computed(() => pagination.total)
const showQualityFields = computed(() => props.snapshot?.mode === 'batch')
const pageSizes = computed(() =>
  props.snapshot?.mode === 'single' ? [20, 50, 100] : [50, 100, 200],
)
const subscriptionWindowTags = computed(() => {
  // 监控弹窗按后端分页加载当前页；只订阅当前页变量，避免一次性订阅全部变量造成实时通道压力。
  return tags.value
})

const {
  connected: socketConnected,
  disconnect,
  onMessage,
  subscribeTag,
} = useMqttSocket(toRef(props, 'projectId'), toRef(props, 'previewSessionId'))

const tagSubscriptionCleanups = new Map()
const activeSubscribedTagIds = ref(new Set<string>())

const { subscribe: subscribeTagSync, unsubscribe: unsubscribeTagSync } = useMqttTagSync(
  props.subscriptionId,
)

const fetchTags = async () => {
  loading.value = true
  try {
    const res = await getMqttTags(props.projectId, props.subscriptionId, {
      page: pagination.page,
      pageSize: pagination.pageSize,
      q: props.snapshot?.q || undefined,
      sortBy: props.snapshot?.sortBy,
      sortOrder: props.snapshot?.sortOrder,
    })
    tags.value = mergeTagRows(tags.value, res.data?.list || [])
    const pageInfo = res.data?.pagination || {}
    pagination.page = Number(pageInfo.page) || pagination.page
    pagination.pageSize = Number(pageInfo.pageSize) || pagination.pageSize
    pagination.total = Number(pageInfo.total) || 0
    normalizePage()
  } catch (error) {
    ElMessage.error(`获取变量列表失败: ${error.message}`)
  } finally {
    loading.value = false
  }
}

const mergeTagRows = (currentRows, nextRows) => {
  const currentValueMap = new Map(
    (currentRows || []).map((tag) => [tag.id, tag.currentValue]).filter(([, value]) => value),
  )
  return (nextRows || []).map((tag) => ({
    ...tag,
    currentValue: latestValueMap.get(tag.id) || tag.currentValue || currentValueMap.get(tag.id),
  }))
}

const setViewMode = (mode: 'list' | 'card') => {
  viewMode.value = mode
}

const normalizePage = () => {
  const totalPages = Math.max(1, Math.ceil(totalTags.value / pagination.pageSize))
  if (pagination.page > totalPages) {
    pagination.page = totalPages
  }
}

const handlePageSizeChange = async () => {
  pagination.page = 1
  releaseAllTagSubscriptions()
  await fetchTags()
  normalizePage()
  syncSubscriptions()
}

const handlePageChange = async () => {
  releaseAllTagSubscriptions()
  await fetchTags()
  syncSubscriptions()
}

const formatValue = (value, dataType) => {
  if (value === null || value === undefined) return '-'

  try {
    if (dataType === 'object' || dataType === 'array') {
      const parsed = typeof value === 'string' ? JSON.parse(value) : value
      return JSON.stringify(parsed, null, 2)
    }
    if (dataType === 'float64') {
      const num = parseFloat(value)
      return Number.isNaN(num) ? value : num.toFixed(2)
    }
    return value
  } catch {
    return value
  }
}

const formatTagValue = (currentValue, dataType) => {
  if (!hasReceivedTagValue(currentValue)) return '-'
  return formatValue(currentValue?.parsedValue ?? currentValue?.value, dataType)
}

const hasReceivedTagValue = (currentValue) => {
  if (!currentValue) return false
  const raw = currentValue.parsedValue ?? currentValue.value
  return raw !== null && raw !== undefined && raw !== ''
}

const formatTimestamp = (timestamp) => {
  const date = dayjs(timestamp)
  return date.isValid() ? date.format(TIME_FORMAT) : '-'
}

const getDataTypeLabel = (dataType) => {
  const labels = {
    string: '字符串',
    float64: '数值',
    bool: '布尔',
    object: '对象',
    array: '数组',
  }
  return labels[dataType] || dataType
}

const getQualityLabel = (quality) => {
  const labels = {
    good: '良好',
    bad: '错误',
    uncertain: '不确定',
    unknown: '未知',
  }
  return labels[quality] || quality
}

const formatQualityLabel = (currentValue) => {
  if (!hasReceivedTagValue(currentValue)) return '-'
  const quality = currentValue?.quality || 'unknown'
  const label = getQualityLabel(quality)
  const qualityCode = currentValue?.qualityCode
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

const getQualityColor = (quality) => {
  const colors = {
    good: 'success',
    bad: 'danger',
    uncertain: 'warning',
    unknown: 'info',
  }
  return colors[quality] || 'info'
}

const getQualityTone = (quality) => {
  const tones = {
    good: 'success',
    bad: 'danger',
    uncertain: 'warning',
    unknown: 'neutral',
  }
  return tones[quality] || 'neutral'
}

const handleTagValueUpdate = (data) => {
  const tag = tags.value.find((item) => item.id === data.tagId)
  if (!tag) {
    return
  }

  tag.currentValue = {
    parsedValue: data.parsedValue ?? data.value,
    value: data.value ?? data.parsedValue,
    quality: data.quality,
    qualityCode: data.qualityCode,
    timestamp: data.timestamp,
    error: data.error,
  }
  latestValueMap.set(tag.id, tag.currentValue)
  emit('tag-value', data)
}

const handleTagSyncEvent = async (event) => {
  switch (event.type) {
    case 'created':
    case 'updated':
    case 'refresh':
      await fetchTags()
      break
    case 'deleted':
      tags.value = tags.value.filter((tag) => tag.id !== event.data.tagId)
      break
  }
}

const syncSubscriptions = () => {
  if (!socketConnected.value) {
    return
  }

  // 批量变量很多时，只订阅当前页变量，翻页时释放上一页订阅，避免实时通道持续推送无关变量。
  const desiredIds = new Set(subscriptionWindowTags.value.map((tag) => tag.id).filter(Boolean))

  // 先释放已经不在当前页的订阅，避免翻页期间旧页变量继续推送到当前弹窗。
  Array.from(tagSubscriptionCleanups.entries()).forEach(([tagId, cleanup]) => {
    if (desiredIds.has(tagId)) {
      return
    }
    cleanup?.()
    tagSubscriptionCleanups.delete(tagId)
  })

  desiredIds.forEach((tagId) => {
    if (!tagSubscriptionCleanups.has(tagId)) {
      tagSubscriptionCleanups.set(tagId, subscribeTag(tagId))
    }
  })

  activeSubscribedTagIds.value = new Set(tagSubscriptionCleanups.keys())
}

const releaseAllTagSubscriptions = () => {
  Array.from(tagSubscriptionCleanups.values()).forEach((cleanup) => {
    cleanup?.()
  })
  tagSubscriptionCleanups.clear()
  activeSubscribedTagIds.value = new Set()
}

const getLatestValues = () =>
  Array.from(latestValueMap.entries()).map(([tagId, currentValue]) => ({
    tagId,
    subscriptionId: props.subscriptionId,
    ...currentValue,
  }))

let stopSocketWatch = null
let unsubscribeMessage = null

onMounted(async () => {
  await fetchTags()
  subscribeTagSync(handleTagSyncEvent)

  unsubscribeMessage = onMessage((data) => {
    if (data?.tagId && activeSubscribedTagIds.value.has(data.tagId)) {
      handleTagValueUpdate(data)
    }
  })

  stopSocketWatch = watch(
    () => [socketConnected.value, subscriptionWindowTags.value.map((tag) => tag.id).join(',')],
    () => {
      if (!socketConnected.value) {
        return
      }
      syncSubscriptions()
    },
    { immediate: true },
  )
})

watch(
  () => props.snapshot,
  async () => {
    pagination.page = Number(props.snapshot?.page) || 1
    pagination.pageSize = Number(props.snapshot?.pageSize) || 50
    pagination.total = Number(props.snapshot?.total) || 0
    if (Array.isArray(props.snapshot?.rows)) {
      tags.value = mergeTagRows(tags.value, props.snapshot.rows)
    }
    releaseAllTagSubscriptions()
    await fetchTags()
    syncSubscriptions()
  },
)

onBeforeUnmount(() => {
  emit('latest-values', getLatestValues())
  stopSocketWatch?.()
  unsubscribeMessage?.()
  unsubscribeTagSync(handleTagSyncEvent)
  releaseAllTagSubscriptions()
  disconnect()
})

defineExpose({
  refresh: fetchTags,
  viewMode,
  setViewMode,
  getLatestValues,
})
</script>

<style scoped>
.mqtt-tag-monitor {
  height: 100%;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  background: var(--dc-surface-raised);
}

.mqtt-tag-monitor__body {
  min-height: 0;
  flex: 1;
  overflow: auto;
  padding: 10px;
  overscroll-behavior: contain;
}

.mqtt-tag-monitor__pagination {
  flex: none;
  min-height: 48px;
  display: flex;
  align-items: center;
  justify-content: flex-end;
  padding: 8px 12px;
  border-top: 1px solid var(--dc-border);
  background: var(--dc-surface-raised);
}

.mqtt-tag-monitor__empty {
  min-height: 280px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 7px;
  color: var(--dc-text-muted);
  text-align: center;
}

.mqtt-tag-monitor__empty svg {
  width: 40px;
  height: 40px;
  opacity: 0.72;
}

.mqtt-tag-monitor__empty div {
  color: var(--dc-text-secondary);
  font-size: 14px;
  font-weight: 700;
}

.mqtt-tag-monitor__empty small {
  font-size: 12px;
}

.tag-card-grid {
  min-height: max-content;
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 8px;
}

.tag-row-list {
  min-height: max-content;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.tag-row {
  display: grid;
  grid-template-columns: 1.4fr 0.6fr 1fr 1.2fr 0.6fr;
  gap: 12px;
  align-items: center;
  padding: 8px 10px;
  background: var(--dc-surface-raised);
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  font-size: 12px;
  color: var(--dc-text-secondary);
}

.tag-row.is-single {
  grid-template-columns: 1.4fr 0.6fr 2fr;
}

.tag-row-header {
  background: var(--dc-surface-subtle);
  font-weight: 600;
  color: var(--dc-text-muted);
}

.tag-row .truncate {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.tag-card {
  padding: 9px 10px;
  border: 1px solid var(--dc-border);
  border-left: 3px solid var(--quality-color, var(--dc-border));
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
  box-shadow: var(--dc-shadow-surface);
  transition:
    border-color 0.16s ease,
    box-shadow 0.16s ease;
  position: relative;
  overflow: hidden;
}

.tag-card:hover {
  border-color: color-mix(in oklch, var(--quality-color, var(--dc-primary)) 45%, var(--dc-border));
}

.tag-card__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.tag-value {
  min-height: 38px;
  display: flex;
  align-items: center;
  margin-top: 8px;
  padding: 6px 8px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-subtle);
  color: var(--dc-text);
  font-family: var(--dc-font-mono, ui-monospace, SFMono-Regular, Menlo, Consolas, monospace);
  font-size: 16px;
  font-weight: 800;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.tag-card__name {
  min-width: 0;
  flex: 1;
  overflow: hidden;
  color: var(--dc-text);
  font-size: 13px;
  font-weight: 700;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.tag-card__time {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-top: 7px;
  color: var(--dc-text-muted);
  font-size: 11px;
}

.tag-card__time strong {
  min-width: 0;
  overflow: hidden;
  color: var(--dc-text-secondary);
  font-weight: 700;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.tag-card.is-good {
  --quality-color: #10b981;
}

.tag-card.is-bad {
  --quality-color: #ef4444;
}

.tag-card.is-uncertain {
  --quality-color: #f59e0b;
}

.tag-card.is-unknown {
  --quality-color: var(--dc-border);
}

@media (min-width: 1920px) {
  .tag-card-grid {
    grid-template-columns: repeat(5, minmax(0, 1fr));
  }
}

@media (max-width: 1180px) {
  .tag-card-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 760px) {
  .mqtt-tag-monitor__toolbar {
    align-items: flex-start;
    flex-direction: column;
  }

  .tag-card-grid {
    grid-template-columns: 1fr;
  }
}
</style>
