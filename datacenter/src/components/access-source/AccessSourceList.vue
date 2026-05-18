<template>
  <div class="access-source-list">
    <AccessSourceCard
      v-for="connection in connections"
      :key="connection.id"
      :connection="connection"
      :active="selectedConnectionId === connection.id"
      :is-phase2="isPhase2Type(connection.type)"
      @open-detail="$emit('open-detail', $event)"
      @open="$emit('open', $event)"
      @edit="$emit('edit', $event)"
    />

    <!-- 新增连接占位卡 -->
    <button
      type="button"
      class="access-source-card access-source-card--create"
      @click="$emit('create')"
    >
      <span class="access-source-card__create-icon">
        <IconTablerPlus />
      </span>
      <div>
        <strong>新增连接</strong>
        <span>连接数据库、MQTT、HTTP 或工业协议源。</span>
      </div>
    </button>

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
import IconTablerPlus from "~icons/tabler/plus";
import AccessSourceCard from "./AccessSourceCard.vue";
import EmptyState from "@/components/shared/EmptyState.vue";

type AccessSourceConnection = {
  id: string;
  name?: string;
  type?: string;
  status?: string;
  datapointCount?: number;
  dataPointCount?: number;
  relationalConfig?: {
    dbType?: string;
    host?: string;
    port?: number | string;
    database?: string;
  };
  mqttConfig?: {
    protocol?: string;
    brokerUrl?: string;
    host?: string;
    port?: number | string;
    topic?: string;
    defaultTopic?: string;
  };
  config?: Record<string, unknown>;
};

defineProps<{
  connections: AccessSourceConnection[];
  selectedConnectionId?: string | null;
}>();

defineEmits<{
  (event: "open-detail", connection: AccessSourceConnection): void;
  (event: "open", connection: AccessSourceConnection): void;
  (event: "edit", connection: AccessSourceConnection): void;
  (event: "create"): void;
}>();

/* Phase 2 类型：当前仅提供配置，无运行时工作台 */
const PHASE2_TYPES = new Set([
  "opcua",
  "opcda",
  "s7",
  "modbus",
  "kafka",
  "http",
  "websocket",
  "redis",
  "tdengine",
]);

const isPhase2Type = (type?: string) => PHASE2_TYPES.has(type || "");
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

/* 新增占位卡样式（从旧 List 保留） */
.access-source-card--create {
  min-height: 190px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 16px;
  padding: 18px;
  border: 2px dashed var(--dc-connection-card-create-border);
  border-radius: var(--dc-radius-md);
  background: rgba(255, 255, 255, 0.46);
  box-shadow: none;
  color: var(--dc-text);
  text-align: center;
  transition:
    border-color 0.18s ease,
    background-color 0.18s ease;
}

.access-source-card--create:hover {
  border-color: color-mix(in oklch, var(--dc-primary) 34%, var(--dc-border));
  background: var(--dc-surface-raised);
}

.access-source-card__create-icon {
  width: 54px;
  height: 54px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: var(--dc-radius-md);
  background: var(--dc-surface-raised);
  box-shadow: var(--dc-shadow-surface);
  color: var(--dc-primary);
}

.access-source-card__create-icon svg {
  width: 28px;
  height: 28px;
}

.access-source-card--create strong,
.access-source-card--create span {
  display: block;
}

.access-source-card--create strong {
  color: var(--dc-text);
  font-size: 18px;
  font-weight: 800;
}

.access-source-card--create span {
  max-width: 230px;
  margin-top: 8px;
  color: var(--dc-text-secondary);
  font-size: 13px;
  line-height: 1.6;
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

  .access-source-card--create {
    min-height: 178px;
    padding: 15px;
  }
}
</style>
