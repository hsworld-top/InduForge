<template>
  <div class="access-source-list">
    <button
      v-for="connection in connections"
      :key="connection.id"
      type="button"
      class="access-source-card"
      :class="{ 'is-active': selectedConnectionId === connection.id }"
      @click="$emit('select', connection)"
      @dblclick="$emit('open', connection)"
    >
      <div class="access-source-card__top">
        <div>
          <div class="access-source-card__name">{{ connection.name }}</div>
          <div class="access-source-card__type">
            {{ resolveConnectionType(connection) }}
          </div>
        </div>
        <span
          class="access-source-card__status"
          :class="`is-${connection.status || 'unknown'}`"
        >
          {{ resolveStatusText(connection.status) }}
        </span>
      </div>

      <div class="access-source-card__meta">
        {{ resolveConnectionEndpoint(connection) }}
      </div>

      <div class="access-source-card__bottom">
        <span>数据点 {{ resolveDatapointCount(connection) }}</span>
        <span>双击打开旧工作台</span>
      </div>
    </button>

    <div v-if="connections.length === 0" class="access-source-list__empty">
      <div class="access-source-list__empty-mark" />
      <strong>暂无接入源</strong>
      <span>点击左侧“新增接入”创建数据库、MQTT 或工业协议连接。</span>
    </div>
  </div>
</template>

<script setup lang="ts">
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
};

defineProps<{
  connections: AccessSourceConnection[];
  selectedConnectionId?: string | null;
}>();

defineEmits<{
  (event: "select", connection: AccessSourceConnection): void;
  (event: "open", connection: AccessSourceConnection): void;
}>();

const dbTypeLabels: Record<string, string> = {
  mysql: "MySQL",
  postgresql: "PostgreSQL",
  sqlserver: "SQL Server",
};

const statusLabels: Record<string, string> = {
  connected: "在线",
  disconnected: "离线",
  error: "异常",
  unknown: "未知",
};

const resolveConnectionType = (connection: AccessSourceConnection) => {
  if (connection.type === "relational") {
    const dbType = connection.relationalConfig?.dbType || "";
    return dbTypeLabels[dbType] || dbType || "数据库";
  }

  if (connection.type === "mqtt") {
    return "MQTT Broker";
  }

  return connection.type || "未知类型";
};

const resolveConnectionEndpoint = (connection: AccessSourceConnection) => {
  if (connection.type === "relational") {
    const config = connection.relationalConfig;
    if (!config) return "未配置数据库地址";
    const host = [config.host, config.port].filter(Boolean).join(":");
    return [host, config.database].filter(Boolean).join(" / ") || "未配置数据库";
  }

  if (connection.type === "mqtt") {
    const config = connection.mqttConfig;
    if (!config) return "未配置 Broker";
    const host = config.brokerUrl || config.host;
    const endpoint = [host, config.port].filter(Boolean).join(":");
    const topic = config.topic || config.defaultTopic;
    return [endpoint, topic].filter(Boolean).join(" / ") || "未配置 Topic";
  }

  return "等待接入配置";
};

const resolveDatapointCount = (connection: AccessSourceConnection) => {
  const count = connection.datapointCount ?? connection.dataPointCount;
  return typeof count === "number" ? count : "待映射";
};

const resolveStatusText = (status?: string) =>
  statusLabels[status || "unknown"] || status || "未知";
</script>

<style scoped>
.access-source-list {
  min-height: 0;
  overflow-y: auto;
  padding: 12px;
}

.access-source-card {
  width: 100%;
  display: block;
  margin-bottom: 10px;
  padding: 12px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-md);
  background: var(--dc-surface-raised);
  color: var(--dc-text);
  text-align: left;
  cursor: pointer;
  transition:
    transform 0.18s ease,
    border-color 0.18s ease,
    box-shadow 0.18s ease;
}

.access-source-card:hover,
.access-source-card.is-active {
  transform: translateY(-1px);
  border-color: rgba(29, 78, 216, 0.42);
  box-shadow: var(--dc-shadow-surface);
}

.access-source-card.is-active {
  background: var(--dc-primary-soft);
}

.access-source-card__top,
.access-source-card__bottom {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 10px;
}

.access-source-card__name {
  color: var(--dc-text);
  font-size: 15px;
  font-weight: 700;
}

.access-source-card__type,
.access-source-card__meta,
.access-source-card__bottom {
  color: var(--dc-text-secondary);
  font-size: 12px;
}

.access-source-card__meta {
  margin-top: 12px;
  padding: 9px 10px;
  overflow: hidden;
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-muted);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.access-source-card__bottom {
  margin-top: 12px;
  color: var(--dc-primary);
  font-weight: 600;
}

.access-source-card__status {
  flex: 0 0 auto;
  padding: 3px 8px;
  border-radius: var(--dc-radius-xs);
  background: var(--dc-surface-muted);
  color: var(--dc-text-secondary);
  font-size: 11px;
  font-weight: 700;
}

.access-source-card__status.is-connected {
  background: var(--dc-success-soft);
  color: var(--dc-success);
}

.access-source-card__status.is-error {
  background: var(--dc-danger-soft);
  color: var(--dc-danger);
}

.access-source-list__empty {
  min-height: 320px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  color: var(--dc-text-secondary);
  text-align: center;
}

.access-source-list__empty strong {
  color: var(--dc-text);
}

.access-source-list__empty span {
  max-width: 240px;
  font-size: 12px;
  line-height: 1.6;
}

.access-source-list__empty-mark {
  width: 62px;
  height: 62px;
  border: 1px solid rgba(29, 78, 216, 0.2);
  border-radius: var(--dc-radius-md);
  background: var(--dc-primary-soft);
}
</style>
