<template>
  <div class="access-source-list">
    <AccessSourceCard
      v-for="connection in connections"
      :key="connection.id"
      :connection="connection"
      :active="selectedConnectionId === connection.id"
      :is-phase2="isPhase2Type(connection.type)"
      @open="$emit('open', $event)"
      @edit="$emit('edit', $event)"
      @delete-connection="$emit('delete-connection', $event)"
    />

    <!-- 空状态：连接列表为空时显示 -->
    <div v-if="connections.length === 0" class="access-source-list__empty-wrap">
      <EmptyState
        icon-name="access-source"
        title="暂无接入源"
        description="使用新增连接卡片创建第一个数据接入。"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import AccessSourceCard from './AccessSourceCard.vue'
import EmptyState from '@/components/shared/EmptyState.vue'

type AccessSourceConnection = {
  id: string
  name?: string
  type?: string
  status?: string
  datapointCount?: number
  dataPointCount?: number
  relationalConfig?: {
    dbType?: string
    host?: string
    port?: number | string
    database?: string
  }
  mqttConfig?: {
    protocol?: string
    brokerUrl?: string
    host?: string
    port?: number | string
    topic?: string
    defaultTopic?: string
  }
  config?: Record<string, unknown>
}

defineProps<{
  connections: AccessSourceConnection[]
  selectedConnectionId?: string | null
}>()

defineEmits<{
  (event: 'open', connection: AccessSourceConnection): void
  (event: 'edit', connection: AccessSourceConnection): void
  (event: 'delete-connection', connection: AccessSourceConnection): void
  (event: 'create'): void
}>()

/* Phase 2 类型：当前仅提供配置，无运行时工作台 */
const PHASE2_TYPES = new Set(['opcda', 's7', 'kafka', 'tdengine'])

const isPhase2Type = (type?: string) => PHASE2_TYPES.has(type || '')
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
