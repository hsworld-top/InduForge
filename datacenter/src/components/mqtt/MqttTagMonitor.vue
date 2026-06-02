<template>
  <div class="mqtt-tag-monitor">
    <div class="mqtt-tag-monitor__toolbar">
      <div class="mqtt-tag-monitor__title">
        <span>变量实时监控</span>
        <WorkbenchStatusPill
          :label="socketConnected ? '变量已订阅' : '等待订阅'"
          :tone="socketConnected ? 'success' : 'neutral'"
        />
        <WorkbenchStatusPill v-if="tags.length > 0" :label="`${tags.length} 个变量`" tone="info" />
      </div>
      <div class="mqtt-tag-monitor__actions">
        <div class="view-toggle">
          <el-button
            size="small"
            :type="viewMode === 'list' ? 'primary' : 'default'"
            @click="viewMode = 'list'"
          >
            列表显示
          </el-button>
          <el-button
            size="small"
            :type="viewMode === 'card' ? 'primary' : 'default'"
            @click="viewMode = 'card'"
          >
            卡片显示
          </el-button>
        </div>
        <el-tooltip content="重新加载变量配置" placement="top">
          <el-button size="small" @click="handleRefresh">
            <IconTablerRefresh class="mr-1 w-4 h-4" />
            重新加载
          </el-button>
        </el-tooltip>
      </div>
    </div>

    <div class="mqtt-tag-monitor__body">
      <div v-if="tags.length === 0" class="mqtt-tag-monitor__empty">
        <IconTablerFile />
        <div>暂无变量</div>
        <small>请先在变量管理中创建变量</small>
      </div>

      <div v-else-if="viewMode === 'card'" class="tag-card-grid">
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
              :label="getQualityLabel(tag.currentValue?.quality || 'unknown')"
              :tone="getQualityTone(tag.currentValue?.quality || 'unknown')"
            />
          </div>

          <div class="tag-value">
            {{ tag.currentValue ? formatValue(tag.currentValue.parsedValue, tag.dataType) : '-' }}
          </div>

          <div class="tag-card__time">
            <span>更新时间</span>
            <strong>{{ tag.currentValue?.timestamp ? formatTimestamp(tag.currentValue.timestamp) : '-' }}</strong>
          </div>
        </div>
      </div>

      <div v-else class="tag-row-list">
        <div class="tag-row tag-row-header">
          <span>变量名</span>
          <span>类型</span>
          <span>当前值</span>
          <span>时间戳</span>
          <span>质量</span>
        </div>
        <div v-for="tag in tags" :key="tag.id" class="tag-row">
          <span class="truncate" :title="tag.name">{{ tag.name }}</span>
          <span>{{ getDataTypeLabel(tag.dataType) }}</span>
          <span class="truncate">
            {{ tag.currentValue ? formatValue(tag.currentValue.parsedValue, tag.dataType) : '-' }}
          </span>
          <span class="truncate">
            {{ tag.currentValue?.timestamp ? formatTimestamp(tag.currentValue.timestamp) : '-' }}
          </span>
          <span>
            <el-tag :type="getQualityColor(tag.currentValue?.quality || 'unknown')" size="small">
              {{ getQualityLabel(tag.currentValue?.quality || 'unknown') }}
            </el-tag>
          </span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount, watch, toRef } from 'vue'
import { ElMessage } from 'element-plus'
import { getMqttTags } from '@/api/data.api'
import { useMqttSocket } from '@/composables/useMqttSocket'
import { useMqttTagSync } from '@/composables/useMqttTagSync'
import WorkbenchStatusPill from '@/components/workbench/WorkbenchStatusPill.vue'
import IconTablerRefresh from '~icons/tabler/refresh'
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
})

const emit = defineEmits<{
  (event: 'tag-value', value: any): void
}>()

const tags = ref([])
const viewMode = ref('list')

const {
  connected: socketConnected,
  disconnect,
  onMessage,
  subscribeTag,
} = useMqttSocket(toRef(props, 'projectId'), toRef(props, 'previewSessionId'))

const tagSubscriptionCleanups = new Map()

const { subscribe: subscribeTagSync, unsubscribe: unsubscribeTagSync } = useMqttTagSync(
  props.subscriptionId,
)

const fetchTags = async () => {
  try {
    const res = await getMqttTags(props.projectId, props.subscriptionId)
    tags.value = res.data?.list || []
  } catch (error) {
    ElMessage.error(`获取变量列表失败: ${error.message}`)
  }
}

const handleRefresh = () => {
  fetchTags()
}

const formatValue = (value, dataType) => {
  if (value === null || value === undefined) return '-'

  try {
    if (dataType === 'object' || dataType === 'array') {
      const parsed = JSON.parse(value)
      return JSON.stringify(parsed, null, 2)
    }
    if (dataType === 'number') {
      const num = parseFloat(value)
      return Number.isNaN(num) ? value : num.toFixed(2)
    }
    return value
  } catch {
    return value
  }
}

const formatTimestamp = (timestamp) => {
  const date = dayjs(timestamp)
  return date.isValid() ? date.format(TIME_FORMAT) : '-'
}

const getDataTypeLabel = (dataType) => {
  const labels = {
    string: '字符串',
    number: '数值',
    boolean: '布尔',
    object: '对象',
    array: '数组',
  }
  return labels[dataType] || dataType
}

const getParseTypeLabel = (parseType) => {
  const labels = {
    jsonpath: 'JSONPath',
    regex: '正则',
    script: '脚本',
    fixed: '固定值',
  }
  return labels[parseType] || parseType
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
    timestamp: data.timestamp,
    error: data.error,
  }
  emit('tag-value', data)
}

const handleTagSyncEvent = async (event) => {
  console.log('[MqttTagMonitor] Received sync event:', event)

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

  const desiredIds = new Set(tags.value.map((tag) => tag.id))

  desiredIds.forEach((tagId) => {
    if (!tagSubscriptionCleanups.has(tagId)) {
      tagSubscriptionCleanups.set(tagId, subscribeTag(tagId))
    }
  })

  Array.from(tagSubscriptionCleanups.entries()).forEach(([tagId, cleanup]) => {
    if (desiredIds.has(tagId)) {
      return
    }
    cleanup?.()
    tagSubscriptionCleanups.delete(tagId)
  })
}

let stopSocketWatch = null
let unsubscribeMessage = null

onMounted(async () => {
  await fetchTags()
  subscribeTagSync(handleTagSyncEvent)

  unsubscribeMessage = onMessage((data) => {
    console.log('[MqttTagMonitor] Received message:', data)
    if (data?.tagId) {
      handleTagValueUpdate(data)
    }
  })

  stopSocketWatch = watch(
    () => [socketConnected.value, tags.value.map((tag) => tag.id).join(',')],
    () => {
      if (!socketConnected.value) {
        return
      }
      syncSubscriptions()
    },
    { immediate: true },
  )
})

onBeforeUnmount(() => {
  stopSocketWatch?.()
  unsubscribeMessage?.()
  unsubscribeTagSync(handleTagSyncEvent)
  Array.from(tagSubscriptionCleanups.values()).forEach((cleanup) => {
    cleanup?.()
  })
  tagSubscriptionCleanups.clear()
  disconnect()
})

defineExpose({
  refresh: fetchTags,
})
</script>

<style scoped>
.mqtt-tag-monitor {
  height: 100%;
  min-height: 0;
  display: flex;
  flex-direction: column;
  background: var(--dc-surface-raised);
}

.mqtt-tag-monitor__toolbar {
  min-height: 48px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 9px 12px;
  border-bottom: 1px solid var(--dc-border);
  background: var(--dc-surface-subtle);
}

.mqtt-tag-monitor__title,
.mqtt-tag-monitor__actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.mqtt-tag-monitor__title > span:first-child {
  color: var(--dc-text);
  font-size: 13px;
  font-weight: 700;
}

.mqtt-tag-monitor__body {
  min-height: 0;
  flex: 1;
  overflow: auto;
  padding: 10px;
}

.view-toggle {
  display: inline-flex;
  gap: 6px;
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
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 8px;
}

.tag-row-list {
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
