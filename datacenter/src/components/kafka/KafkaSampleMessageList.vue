<template>
  <div ref="rootRef" class="kafka-sample-list">
    <el-scrollbar ref="scrollbarRef" class="kafka-sample-list__scroll" @scroll="handleScroll">
      <div v-if="loading" class="kafka-sample-list__state">
        <IconTablerLoader2 class="kafka-sample-list__spinner" />
        <span>正在拉取 Kafka 样本...</span>
      </div>

      <div v-else-if="messages.length === 0" class="kafka-sample-list__state">
        <IconTablerInbox />
        <span>{{ emptyText }}</span>
        <small>{{ emptyHint }}</small>
      </div>

      <div v-else class="kafka-sample-list__items" :style="{ height: `${virtualHeight}px` }">
        <div
          class="kafka-sample-list__window"
          :style="{ transform: `translateY(${windowOffset}px)` }"
        >
          <article
            v-for="item in visibleItems"
            :key="item.message.id || `${item.message.topic || 'kafka'}-${item.index}`"
            :data-stream-index="item.index"
            class="kafka-sample-list__item"
          >
            <header class="kafka-sample-list__item-head">
              <span class="kafka-sample-list__topic" :title="item.message.topic || '-'">
                {{ item.message.topic || '-' }}
              </span>
              <span class="kafka-sample-list__meta">
                p{{ item.message.partition ?? '-' }} / o{{ item.message.offset ?? '-' }}
              </span>
              <span v-if="item.message.key" class="kafka-sample-list__key">
                key {{ item.message.key }}
              </span>
              <time v-if="showTimestamp">{{ formatTimestamp(item.message.timestamp) }}</time>
              <el-tooltip content="复制 Payload" placement="top">
                <button
                  type="button"
                  class="kafka-sample-list__copy"
                  @click.stop="$emit('copy', item.message)"
                >
                  <IconTablerCopy />
                </button>
              </el-tooltip>
            </header>

            <pre class="kafka-sample-list__payload">{{ formatPayload(item.message.payload) }}</pre>
          </article>
        </div>
      </div>
    </el-scrollbar>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import dayjs from 'dayjs'
import { TIME_FORMAT } from '@/constants'
import IconTablerCopy from '~icons/tabler/copy'
import IconTablerInbox from '~icons/tabler/inbox'
import IconTablerLoader2 from '~icons/tabler/loader-2'

export type KafkaSampleMessage = {
  id?: string | number
  topic?: string
  partition?: number | string | null
  offset?: number | string | null
  key?: string | null
  payload?: unknown
  timestamp?: string | number | Date
}

const props = withDefaults(
  defineProps<{
    messages: KafkaSampleMessage[]
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
    emptyText: '暂无 Kafka 样本',
    emptyHint: '设置拉取条数后点击拉取样本',
  },
)

defineEmits<{
  (event: 'copy', message: KafkaSampleMessage): void
}>()

const rootRef = ref<HTMLElement | null>(null)
const scrollbarRef = ref<any>(null)
const scrollTop = ref(0)
const viewportHeight = ref(0)
const itemHeights = ref<number[]>([])
let resizeObserver: ResizeObserver | null = null

const estimatedItemHeight = 156
const bufferItems = 6
// 样本 payload 可能很长，记录真实高度能让虚拟滚动在格式化前后保持准确。
const itemOffsets = computed(() => {
  const offsets: number[] = []
  let nextOffset = 0
  for (let index = 0; index < props.messages.length; index += 1) {
    offsets[index] = nextOffset
    nextOffset += itemHeights.value[index] || estimatedItemHeight
  }
  return offsets
})
const virtualHeight = computed(() => {
  if (props.messages.length === 0) return 0
  const lastIndex = props.messages.length - 1
  return itemOffsets.value[lastIndex] + (itemHeights.value[lastIndex] || estimatedItemHeight)
})
const visibleRange = computed(() => {
  if (props.messages.length === 0) return { start: 0, end: 0 }
  const offsets = itemOffsets.value
  const viewportBottom = scrollTop.value + viewportHeight.value
  let start = 0
  while (
    start < props.messages.length &&
    offsets[start] + (itemHeights.value[start] || estimatedItemHeight) < scrollTop.value
  ) {
    start += 1
  }
  let end = start
  while (end < props.messages.length && offsets[end] < viewportBottom) {
    end += 1
  }
  return {
    start: Math.max(0, start - bufferItems),
    end: Math.min(props.messages.length, end + bufferItems),
  }
})
const visibleItems = computed(() =>
  props.messages.slice(visibleRange.value.start, visibleRange.value.end).map((message, offset) => ({
    message,
    index: visibleRange.value.start + offset,
  })),
)
const windowOffset = computed(() => itemOffsets.value[visibleRange.value.start] || 0)

const formatTimestamp = (timestamp?: string | number | Date) => {
  if (!timestamp) return '-'
  const date = dayjs(timestamp)
  return date.isValid() ? date.format(TIME_FORMAT) : '-'
}

const formatPayload = (payload: unknown) => {
  if (payload === null || payload === undefined) return ''
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

const handleScroll = ({ scrollTop: nextScrollTop }: { scrollTop: number }) => {
  scrollTop.value = Number(nextScrollTop || 0)
}

const updateViewportHeight = () => {
  viewportHeight.value = rootRef.value?.clientHeight || 0
}

const measureVisibleItems = async () => {
  await nextTick()
  const container = rootRef.value
  if (!container) return

  let changed = false
  const nextHeights = itemHeights.value.slice(0, props.messages.length)
  container
    .querySelectorAll<HTMLElement>('.kafka-sample-list__item[data-stream-index]')
    .forEach((element) => {
      const index = Number(element.dataset.streamIndex)
      if (!Number.isInteger(index) || index < 0) return
      const height = Math.ceil(element.getBoundingClientRect().height)
      if (height > 0 && nextHeights[index] !== height) {
        nextHeights[index] = height
        changed = true
      }
    })

  if (changed) {
    itemHeights.value = nextHeights
  }
}

onMounted(() => {
  updateViewportHeight()
  if (rootRef.value && typeof ResizeObserver !== 'undefined') {
    resizeObserver = new ResizeObserver(() => {
      updateViewportHeight()
      measureVisibleItems()
    })
    resizeObserver.observe(rootRef.value)
  }
  measureVisibleItems()
})

onBeforeUnmount(() => {
  resizeObserver?.disconnect()
  resizeObserver = null
})

watch(
  () => props.messages,
  () => {
    itemHeights.value = []
    updateViewportHeight()
    measureVisibleItems()
  },
)

watch(
  [visibleItems, () => props.formatJson],
  () => {
    measureVisibleItems()
  },
  { flush: 'post' },
)
</script>

<style scoped>
.kafka-sample-list {
  height: 100%;
  min-height: 0;
  background: var(--dc-surface-raised, #fff);
}

.kafka-sample-list__scroll {
  height: 100%;
}

.kafka-sample-list__items {
  position: relative;
}

.kafka-sample-list__window {
  position: absolute;
  inset: 0 0 auto 0;
  will-change: transform;
}

.kafka-sample-list__item {
  min-height: 156px;
  padding: 11px 12px 12px;
  border-bottom: 1px solid var(--dc-border, #e2e8f0);
}

.kafka-sample-list__item:hover {
  background: var(--dc-surface-subtle, #f8fafc);
}

.kafka-sample-list__item-head {
  display: grid;
  grid-template-columns: minmax(160px, 1fr) auto auto auto auto;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}

.kafka-sample-list__topic,
.kafka-sample-list__meta,
.kafka-sample-list__key {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.kafka-sample-list__topic {
  color: var(--dc-text-secondary, #334155);
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: 12px;
  font-weight: 700;
}

.kafka-sample-list__meta,
.kafka-sample-list__key {
  height: 22px;
  display: inline-flex;
  align-items: center;
  padding: 0 7px;
  border: 1px solid var(--dc-border, #e2e8f0);
  border-radius: 999px;
  background: var(--dc-surface-muted, #f1f5f9);
  color: var(--dc-text-secondary, #334155);
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: 11px;
}

.kafka-sample-list__item-head time {
  color: var(--dc-text-muted, #64748b);
  font-size: 11px;
  white-space: nowrap;
}

.kafka-sample-list__copy {
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

.kafka-sample-list__copy:hover {
  color: var(--dc-primary, #2563eb);
}

.kafka-sample-list__copy svg {
  width: 14px;
  height: 14px;
}

.kafka-sample-list__payload {
  margin: 0;
  padding: 10px 11px;
  overflow: visible;
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

.kafka-sample-list__state {
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

.kafka-sample-list__state svg {
  width: 34px;
  height: 34px;
  opacity: 0.75;
}

.kafka-sample-list__state small {
  font-size: 12px;
}

.kafka-sample-list__spinner {
  animation: kafka-sample-spin 0.9s linear infinite;
}

@media (max-width: 960px) {
  .kafka-sample-list__item-head {
    grid-template-columns: minmax(0, 1fr) auto auto;
  }

  .kafka-sample-list__key,
  .kafka-sample-list__item-head time {
    display: none;
  }
}

@keyframes kafka-sample-spin {
  to {
    transform: rotate(360deg);
  }
}
</style>
