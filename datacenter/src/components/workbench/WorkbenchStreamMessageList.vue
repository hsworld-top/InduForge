<template>
  <div ref="rootRef" class="workbench-stream-list">
    <el-scrollbar ref="scrollbarRef" class="workbench-stream-list__scroll">
      <div v-if="loading" class="workbench-stream-list__state">
        <IconTablerLoader2 class="workbench-stream-list__spinner" />
        <span>加载消息...</span>
      </div>

      <div v-else-if="messages.length === 0" class="workbench-stream-list__state">
        <IconTablerInbox />
        <span>{{ emptyText }}</span>
        <small>{{ emptyHint }}</small>
      </div>

      <div v-else class="workbench-stream-list__items">
        <article
          v-for="(message, index) in messages"
          :key="message.id || `${message.topic || 'message'}-${index}`"
          class="workbench-stream-list__item"
          @click="$emit('select', message)"
        >
          <header class="workbench-stream-list__item-head">
            <WorkbenchStatusPill :label="`QoS ${message.qos ?? 0}`" tone="info" />
            <span class="workbench-stream-list__topic" :title="message.topic || '-'">
              {{ message.topic || '-' }}
            </span>
            <time v-if="showTimestamp">
              {{ formatTimestamp(message.timestamp) }}
            </time>
            <el-tooltip content="复制 Payload" placement="top">
              <button
                type="button"
                class="workbench-stream-list__copy"
                @click.stop="$emit('copy', message)"
              >
                <IconTablerCopy />
              </button>
            </el-tooltip>
          </header>

          <pre class="workbench-stream-list__payload">{{ formatPayload(message.payload) }}</pre>
        </article>
      </div>
    </el-scrollbar>
  </div>
</template>

<script setup lang="ts">
import { nextTick, ref } from 'vue'
import dayjs from 'dayjs'
import { TIME_FORMAT } from '@/constants'
import WorkbenchStatusPill from './WorkbenchStatusPill.vue'
import IconTablerCopy from '~icons/tabler/copy'
import IconTablerInbox from '~icons/tabler/inbox'
import IconTablerLoader2 from '~icons/tabler/loader-2'

type StreamMessage = {
  id?: string | number
  topic?: string
  payload?: unknown
  qos?: number
  timestamp?: string | number | Date
}

const props = withDefaults(
  defineProps<{
    messages: StreamMessage[]
    loading?: boolean
    showTimestamp?: boolean
    formatJson?: boolean
    emptyText?: string
    emptyHint?: string
  }>(),
  {
    loading: false,
    showTimestamp: true,
    formatJson: true,
    emptyText: '暂无消息',
    emptyHint: '等待接收实时数据',
  },
)

defineEmits<{
  (event: 'select', message: StreamMessage): void
  (event: 'copy', message: StreamMessage): void
}>()

const rootRef = ref<HTMLElement | null>(null)
const scrollbarRef = ref<any>(null)

const formatTimestamp = (timestamp?: string | number | Date) => {
  if (!timestamp) return '-'
  const date = dayjs(timestamp)
  return date.isValid() ? date.format(TIME_FORMAT) : '-'
}

const formatPayload = (payload: unknown) => {
  if (payload === null || payload === undefined) {
    return ''
  }
  if (typeof payload === 'string') {
    if (!props.formatJson) return payload
    try {
      return JSON.stringify(JSON.parse(payload), null, 2)
    } catch {
      return payload
    }
  }
  try {
    return JSON.stringify(payload, null, props.formatJson ? 2 : 0)
  } catch {
    return String(payload)
  }
}

const scrollToTop = async () => {
  await nextTick()
  scrollbarRef.value?.setScrollTop?.(0)
}

defineExpose({
  scrollToTop,
})
</script>

<style scoped>
.workbench-stream-list {
  height: 100%;
  min-height: 0;
  background: var(--dc-surface-raised, #fff);
}

.workbench-stream-list__scroll {
  height: 100%;
}

.workbench-stream-list__items {
  display: flex;
  flex-direction: column;
}

.workbench-stream-list__item {
  padding: 10px 12px;
  border-bottom: 1px solid var(--dc-border, #e2e8f0);
  cursor: pointer;
}

.workbench-stream-list__item:hover {
  background: var(--dc-primary-soft, #eff6ff);
}

.workbench-stream-list__item-head {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto auto;
  align-items: center;
  gap: 8px;
  margin-bottom: 7px;
}

.workbench-stream-list__topic {
  overflow: hidden;
  color: var(--dc-text-secondary, #334155);
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.workbench-stream-list__item-head time {
  color: var(--dc-text-muted, #64748b);
  font-size: 11px;
  white-space: nowrap;
}

.workbench-stream-list__copy {
  width: 24px;
  height: 24px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--dc-border, #e2e8f0);
  border-radius: var(--dc-radius-sm, 6px);
  background: var(--dc-surface-raised, #fff);
  color: var(--dc-text-muted, #64748b);
}

.workbench-stream-list__copy:hover {
  color: var(--dc-primary, #2563eb);
}

.workbench-stream-list__copy svg {
  width: 14px;
  height: 14px;
}

.workbench-stream-list__payload {
  max-height: 280px;
  margin: 0;
  padding: 9px 10px;
  overflow: auto;
  border: 1px solid color-mix(in oklch, var(--dc-border, #e2e8f0) 78%, transparent);
  border-radius: var(--dc-radius-sm, 6px);
  background: var(--dc-surface-subtle, #f8fafc);
  color: var(--dc-text, #0f172a);
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: 12px;
  line-height: 1.55;
  white-space: pre-wrap;
  word-break: break-word;
}

.workbench-stream-list__state {
  height: 100%;
  min-height: 260px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  color: var(--dc-text-muted, #64748b);
  font-size: 13px;
}

.workbench-stream-list__state svg {
  width: 34px;
  height: 34px;
  opacity: 0.75;
}

.workbench-stream-list__state small {
  font-size: 12px;
}

.workbench-stream-list__spinner {
  animation: workbench-stream-spin 0.9s linear infinite;
}

@keyframes workbench-stream-spin {
  to {
    transform: rotate(360deg);
  }
}
</style>
