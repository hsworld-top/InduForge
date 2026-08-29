<template>
  <div
    ref="listRef"
    class="access-source-list"
    @dragover.prevent="handleListDragOver"
    @drop.prevent="handleDrop"
  >
    <template v-for="item in displayItems" :key="item.key">
      <div v-if="item.kind === 'placeholder'" class="access-source-list__placeholder">
        <span>{{ ui('放到这里', 'Drop here') }}</span>
      </div>
      <AccessSourceCard
        v-else
        :connection="item.connection"
        :active="selectedConnectionId === item.connection.id"
        :testing="testingConnectionId === item.connection.id"
        :class="{ 'is-dragging': draggable && draggingConnectionId === item.connection.id }"
        @dragover.prevent="handleListDragOver"
        @drop.prevent="handleDrop"
        @dragend="clearDragState"
        @open="$emit('open', $event)"
        @edit="$emit('edit', $event)"
        @test="$emit('test', $event)"
        @delete-connection="$emit('delete-connection', $event)"
      >
        <template #top-actions>
          <button
            v-if="draggable"
            type="button"
            class="access-source-list__drag-handle"
            draggable="true"
            :title="ui('拖拽调整位置', 'Drag to reorder')"
            :aria-label="ui('拖拽调整位置', 'Drag to reorder')"
            @dragstart="handleDragStart(item.connection.id, $event)"
            @dragend="clearDragState"
            @click.stop
          >
            <IconTablerGripVertical />
          </button>
        </template>
      </AccessSourceCard>
    </template>

    <!-- 空状态：连接列表为空时显示 -->
    <div v-if="connections.length === 0" class="access-source-list__empty-wrap">
      <EmptyState
        icon-name="access-source"
        :title="ui('暂无接入源', 'No access sources')"
        :description="ui('使用新增连接卡片创建第一个数据接入。', 'Create the first data connection with New Connection.')"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import AccessSourceCard from './AccessSourceCard.vue'
import EmptyState from '@/components/shared/EmptyState.vue'
import IconTablerGripVertical from '~icons/tabler/grip-vertical'
import type { Connection as AccessSourceConnection } from '@/api/schemas/connection.schema'
import { datacenterLocale } from '@/i18n/runtime'

const ui = (zh: string, en: string) => (datacenterLocale.value === 'en' ? en : zh)

const emit = defineEmits<{
  (event: 'open', connection: AccessSourceConnection): void
  (event: 'edit', connection: AccessSourceConnection): void
  (event: 'test', connection: AccessSourceConnection): void
  (event: 'delete-connection', connection: AccessSourceConnection): void
  (event: 'create'): void
  (event: 'reorder', connectionIds: string[]): void
}>()

const props = defineProps<{
  connections: AccessSourceConnection[]
  selectedConnectionId?: string | null
  draggable?: boolean
  testingConnectionId?: string
}>()

const draggingConnectionId = ref<string | null>(null)
const placeholderIndex = ref<number | null>(null)
const listRef = ref<HTMLElement | null>(null)

const displayItems = computed(() => {
  const items = props.connections.map((connection) => ({
    kind: 'connection' as const,
    key: connection.id,
    connection,
  }))
  if (!props.draggable || !draggingConnectionId.value || placeholderIndex.value === null) {
    return items
  }
  const index = Math.min(Math.max(placeholderIndex.value, 0), items.length)
  return [
    ...items.slice(0, index),
    { kind: 'placeholder' as const, key: 'drop-placeholder' },
    ...items.slice(index),
  ]
})

const clearDragState = () => {
  draggingConnectionId.value = null
  placeholderIndex.value = null
}

const handleDragStart = (connectionId: string, event: DragEvent) => {
  if (!props.draggable) return
  draggingConnectionId.value = connectionId
  placeholderIndex.value = null
  event.dataTransfer?.setData('text/plain', connectionId)
  if (event.dataTransfer) {
    event.dataTransfer.effectAllowed = 'move'
  }
}

const handleListDragOver = (event: DragEvent) => {
  if (!props.draggable || !draggingConnectionId.value || !listRef.value) return

  const list = listRef.value
  const card = list.querySelector('.access-source-card') as HTMLElement | null
  if (!card) return

  const listStyle = window.getComputedStyle(list)
  const listRect = list.getBoundingClientRect()
  const cardRect = card.getBoundingClientRect()
  const paddingLeft = Number.parseFloat(listStyle.paddingLeft) || 0
  const paddingTop = Number.parseFloat(listStyle.paddingTop) || 0
  const paddingRight = Number.parseFloat(listStyle.paddingRight) || 0
  const columnGap = Number.parseFloat(listStyle.columnGap) || 0
  const rowGap = Number.parseFloat(listStyle.rowGap) || columnGap
  const contentWidth = list.clientWidth - paddingLeft - paddingRight
  const columnWidth = cardRect.width + columnGap
  const rowHeight = cardRect.height + rowGap
  const columnCount = Math.max(1, Math.floor((contentWidth + columnGap) / columnWidth))
  const localX = event.clientX - listRect.left - paddingLeft + list.scrollLeft
  const localY = event.clientY - listRect.top - paddingTop + list.scrollTop
  const column = Math.min(Math.max(Math.floor(localX / columnWidth), 0), columnCount - 1)
  const row = Math.max(Math.floor(localY / rowHeight), 0)
  const xInCard = localX - column * columnWidth
  const yInCard = localY - row * rowHeight
  const insertAfter = xInCard > cardRect.width / 2 || yInCard > cardRect.height * 0.62
  const nextIndex = row * columnCount + column + (insertAfter ? 1 : 0)
  placeholderIndex.value = Math.min(Math.max(nextIndex, 0), props.connections.length)
}

const handleDrop = () => {
  if (!props.draggable || !draggingConnectionId.value) {
    clearDragState()
    return
  }
  const sourceConnectionId = draggingConnectionId.value
  const targetIndex = placeholderIndex.value
  clearDragState()
  if (targetIndex === null) return

  const sourceConnection = props.connections.find(
    (connection) => connection.id === sourceConnectionId,
  )
  if (!sourceConnection) return

  const boundedTargetIndex = Math.min(Math.max(targetIndex, 0), props.connections.length)
  const sourceIndex = props.connections.findIndex(
    (connection) => connection.id === sourceConnectionId,
  )
  const insertionIndex =
    sourceIndex >= 0 && sourceIndex < boundedTargetIndex
      ? boundedTargetIndex - 1
      : boundedTargetIndex
  const nextConnections = props.connections.filter(
    (connection) => connection.id !== sourceConnectionId,
  )
  nextConnections.splice(
    Math.min(Math.max(insertionIndex, 0), nextConnections.length),
    0,
    sourceConnection,
  )
  emit(
    'reorder',
    nextConnections.map((connection) => connection.id),
  )
}
</script>

<style scoped>
.access-source-list {
  min-height: 0;
  flex: 1;
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(310px, 1fr));
  align-content: start;
  gap: 18px;
  overflow-y: auto;
  padding: 20px;
}

.access-source-list :deep(.access-source-card.is-dragging) {
  opacity: 0.58;
  transform: scale(0.985);
}

.access-source-list__drag-handle {
  width: 28px;
  height: 28px;
  flex: 0 0 auto;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 0;
  border-radius: var(--dc-radius-sm);
  background: transparent;
  color: var(--dc-text-muted);
  cursor: grab;
  transition:
    background-color 0.18s ease,
    color 0.18s ease;
}

.access-source-list__drag-handle:hover {
  background: var(--dc-surface-muted);
  color: var(--dc-primary);
}

.access-source-list__drag-handle:active {
  cursor: grabbing;
}

.access-source-list__drag-handle svg {
  width: 16px;
  height: 16px;
}

.access-source-list__placeholder {
  min-height: 168px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 1px dashed color-mix(in oklch, var(--dc-primary) 52%, var(--dc-border));
  border-radius: var(--dc-radius-md);
  background: linear-gradient(
    135deg,
    color-mix(in oklch, var(--dc-primary-soft) 70%, transparent),
    color-mix(in oklch, var(--dc-surface-raised) 86%, transparent)
  );
  color: var(--dc-primary);
  font-size: 13px;
  font-weight: 700;
  box-shadow: inset 0 0 0 2px color-mix(in oklch, var(--dc-primary) 10%, transparent);
  pointer-events: none;
}

.access-source-list__placeholder span {
  padding: 4px 10px;
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
  box-shadow: var(--dc-shadow-surface);
}

/* 空状态占满整行 */
.access-source-list__empty-wrap {
  grid-column: 1 / -1;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 32px 0;
}

@media (max-width: 760px) {
  .access-source-list {
    grid-template-columns: 1fr;
    gap: 12px;
    padding: 14px;
  }
}
</style>
