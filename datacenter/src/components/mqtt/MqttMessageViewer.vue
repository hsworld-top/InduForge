<template>
  <div class="mqtt-message-viewer">
    <WorkbenchStreamToolbar
      v-model:search="searchText"
      v-model:limit="displayLimit"
      v-model:format-json="formatJson"
      v-model:auto-scroll="autoScroll"
      v-model:show-timestamp="showTimestamp"
      :icon="IconTablerRss"
      :title="subscriptionTitle"
      :subtitle="subscription?.topic || ''"
      :status-label="isSubscribed ? '已订阅' : '未订阅'"
      :status-tone="isSubscribed ? 'success' : 'neutral'"
      :status-clickable="true"
      :status-action-loading="subscriptionChanging"
      :loading="loading"
      :max-limit="MESSAGE_WINDOW_LIMIT"
      @status-click="handleToggleSubscription"
      @refresh="loadMessages"
      @clear="handleClear"
    />

    <div class="mqtt-message-viewer__body">
      <WorkbenchStreamMessageList
        ref="messageListRef"
        :messages="filteredMessages"
        :loading="loading"
        :format-json="formatJson"
        :show-timestamp="showTimestamp"
        :empty-text="emptyText"
        :empty-hint="emptyHint"
        @select="handleSelectMessage"
        @copy="handleCopyMessage"
      />
    </div>

    <footer class="mqtt-message-viewer__status">
      <span>
        共 {{ messages.length }} 条
        <template v-if="searchText"> · 筛选 {{ filteredMessages.length }} 条</template>
      </span>
      <span v-if="lastMessageTime">最后消息 {{ formatTimestamp(lastMessageTime) }}</span>
      <span>{{ isSubscribed ? '实时订阅已启用' : '实时订阅已停止' }}</span>
    </footer>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import dayjs from 'dayjs'
import IconTablerRss from '~icons/tabler/rss'
import dataAPI from '@/api/data.api'
import { TIME_FORMAT } from '@/constants'
import { getApiErrorMessage } from '@/utils/request'
import WorkbenchStreamMessageList from '@/components/workbench/WorkbenchStreamMessageList.vue'
import WorkbenchStreamToolbar from '@/components/workbench/WorkbenchStreamToolbar.vue'

type MqttMessage = {
  id?: string | number
  topic?: string
  payload?: unknown
  qos?: number
  timestamp?: string | number | Date
}

const props = defineProps({
  subscription: {
    type: Object,
    default: null,
  },
  projectId: {
    type: String,
    required: true,
  },
  connectionId: {
    type: String,
    required: true,
  },
  source: {
    type: String,
    default: 'mqtt',
  },
})

const emit = defineEmits(['message-select', 'subscribe', 'unsubscribe'])

const MESSAGE_WINDOW_LIMIT = 5000

const loading = ref(false)
const clearing = ref(false)
const messages = ref<MqttMessage[]>([])
const searchText = ref('')
const displayLimit = ref(500)
const autoScroll = ref(true)
const showTimestamp = ref(true)
const formatJson = ref(true)
const isSubscribed = ref(false)
const subscriptionChanging = ref(false)
const messageListRef = ref<any>(null)

const subscriptionTitle = computed(
  () => props.subscription?.name || props.subscription?.topic || '消息查看器',
)
const isBuiltinMessage = computed(() => props.source === 'builtin-message')
const emptyText = computed(() => (isBuiltinMessage.value ? '暂无 IF消息' : '暂无 MQTT 消息'))
const emptyHint = computed(() =>
  isBuiltinMessage.value ? '发布或接收消息后会显示在这里' : '启动预览连接后，实时消息会显示在这里',
)

const filteredMessages = computed(() => {
  const keyword = searchText.value.trim().toLowerCase()
  let result = messages.value

  if (keyword) {
    result = result.filter((message) => {
      const topic = String(message.topic || '').toLowerCase()
      const payload =
        typeof message.payload === 'string'
          ? message.payload
          : JSON.stringify(message.payload ?? '')
      return topic.includes(keyword) || payload.toLowerCase().includes(keyword)
    })
  }

  const limit = Math.min(displayLimit.value > 0 ? displayLimit.value : MESSAGE_WINDOW_LIMIT, MESSAGE_WINDOW_LIMIT)
  return result.slice(0, limit)
})

const lastMessageTime = computed(() => messages.value[0]?.timestamp || null)

const formatTimestamp = (timestamp) => {
  const date = dayjs(timestamp)
  return date.isValid() ? date.format(TIME_FORMAT) : '-'
}

const formatPayload = (payload) => {
  if (payload === null || payload === undefined) return ''
  if (typeof payload === 'string') {
    if (!formatJson.value) return payload
    try {
      return JSON.stringify(JSON.parse(payload), null, 2)
    } catch {
      return payload
    }
  }
  try {
    return JSON.stringify(payload, null, formatJson.value ? 2 : 0)
  } catch {
    return String(payload)
  }
}

const normalizeIncomingMessage = (message) => {
  const payload = message?.message || message || {}
  return {
    id: payload.id || `${Date.now()}-${Math.random()}`,
    topic: payload.topic || props.subscription?.topic || '',
    payload: payload.payload,
    qos: payload.qos ?? 0,
    timestamp: payload.timestamp || payload.receivedAt || Date.now(),
  }
}

const messageFingerprint = (message: MqttMessage) => [
  message.topic || '',
  String(message.qos ?? 0),
  String(message.timestamp || ''),
  typeof message.payload === 'string' ? message.payload : JSON.stringify(message.payload ?? ''),
].join('\u0001')

// 订阅成功时后端会补发最近消息快照；它可能和历史列表最新一条相同，需要去重避免刷新后数量跳变。
const uniqueMessages = (items: MqttMessage[]) => {
  const seen = new Set<string>()
  const result: MqttMessage[] = []
  for (const item of items) {
    const key = messageFingerprint(item)
    if (seen.has(key)) continue
    seen.add(key)
    result.push(item)
  }
  return result
}

/**
 * 历史消息只作为当前订阅的启动快照；实时消息仍由工作台 socket 追加。
 */
const loadMessages = async () => {
  if (!props.subscription?.id) return

  loading.value = true
  try {
    const response = await dataAPI.getMqttSubscriptionMessages(
      props.projectId,
      props.subscription.id,
      { limit: displayLimit.value },
    )
    messages.value = uniqueMessages((response.data?.list || []).map(normalizeIncomingMessage))
      .slice(0, MESSAGE_WINDOW_LIMIT)
    await messageListRef.value?.scrollToTop?.()
  } catch (error) {
    ElMessage.error('加载消息失败：' + getApiErrorMessage(error, '加载消息失败'))
  } finally {
    loading.value = false
  }
}

const addMessage = async (message) => {
  messages.value = uniqueMessages([normalizeIncomingMessage(message), ...messages.value])
  if (messages.value.length > MESSAGE_WINDOW_LIMIT) {
    messages.value = messages.value.slice(0, MESSAGE_WINDOW_LIMIT)
  }
  if (autoScroll.value) {
    await nextTick()
    await messageListRef.value?.scrollToTop?.()
  }
}

const setSubscribed = (subscribed) => {
  isSubscribed.value = Boolean(subscribed)
}

const handleToggleSubscription = async () => {
  if (!props.subscription?.id || subscriptionChanging.value) return
  subscriptionChanging.value = true
  try {
    emit(isSubscribed.value ? 'unsubscribe' : 'subscribe', props.subscription)
  } finally {
    subscriptionChanging.value = false
  }
}

const handleSelectMessage = (message) => {
  emit('message-select', message)
}

const handleCopyMessage = async (message) => {
  try {
    await navigator.clipboard.writeText(formatPayload(message.payload))
    ElMessage.success('已复制 Payload')
  } catch {
    ElMessage.error('复制失败')
  }
}

const handleClear = async () => {
  if (clearing.value) return
  if (!props.subscription?.id) return

  clearing.value = true
  try {
    await dataAPI.clearMqttSubscriptionMessages(props.projectId, props.subscription.id)
    messages.value = []
    ElMessage.success('消息已清空')
  } catch (error) {
    ElMessage.error('清空消息失败：' + getApiErrorMessage(error, '清空消息失败'))
  } finally {
    clearing.value = false
  }
}

watch(
  () => props.subscription,
  (subscription) => {
    messages.value = []
    searchText.value = ''
    isSubscribed.value = false
    if (subscription?.id) {
      loadMessages()
    }
  },
  { immediate: true },
)

defineExpose({
  addMessage,
  setSubscribed,
  loadMessages,
})
</script>

<style scoped>
.mqtt-message-viewer {
  height: 100%;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  background: var(--dc-surface-raised);
}

.mqtt-message-viewer__body {
  min-height: 0;
  flex: 1;
}

.mqtt-message-viewer__status {
  min-height: 34px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 0 12px;
  border-top: 1px solid var(--dc-border);
  background: var(--dc-surface-subtle);
  color: var(--dc-text-muted);
  font-size: 11px;
}
</style>
