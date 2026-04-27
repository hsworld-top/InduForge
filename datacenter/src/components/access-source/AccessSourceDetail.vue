<template>
  <aside class="access-source-detail">
    <template v-if="connection">
      <div class="access-source-detail__hero">
        <div class="access-source-detail__eyebrow">接入源详情</div>
        <h3>{{ connection.name || "-" }}</h3>
        <p>{{ typeLabel }}</p>
        <el-button type="primary" size="small" @click="$emit('open', connection)">
          打开工作台
        </el-button>
      </div>

      <el-tabs v-model="activeTab" class="access-source-detail__tabs">
        <el-tab-pane
          v-for="tab in tabs"
          :key="tab.value"
          :label="tab.label"
          :name="tab.value"
        />
      </el-tabs>

      <div class="access-source-detail__body">
        <template v-if="activeTab === 'overview'">
          <dl class="access-source-detail__list">
            <div>
              <dt>状态</dt>
              <dd>{{ statusText }}</dd>
            </div>
            <div>
              <dt>地址</dt>
              <dd>{{ endpointText }}</dd>
            </div>
            <div>
              <dt>数据点</dt>
              <dd>{{ datapointText }}</dd>
            </div>
          </dl>
        </template>

        <template v-else-if="activeTab === 'config'">
          <dl class="access-source-detail__list">
            <div v-for="item in configRows" :key="item.label">
              <dt>{{ item.label }}</dt>
              <dd>{{ item.value }}</dd>
            </div>
          </dl>
        </template>

        <template v-else>
          <div class="access-source-detail__placeholder">
            <strong>{{ currentTabLabel }}</strong>
            <span>
              初版先保留工作区占位，点击“打开工作台”进入旧表查询或 MQTT
              管理流程。
            </span>
          </div>
        </template>
      </div>
    </template>

    <div v-else class="access-source-detail__empty">
      <div class="access-source-detail__empty-orb" />
      <strong>选择一个接入源</strong>
      <span>右侧会展示概览、配置、解析映射、预览、数据点和记录入口。</span>
    </div>
  </aside>
</template>

<script setup lang="ts">
import { computed, ref, watch } from "vue";

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

const props = defineProps<{
  connection?: AccessSourceConnection | null;
}>();

defineEmits<{
  (event: "open", connection: AccessSourceConnection): void;
}>();

const activeTab = ref("overview");

const tabs = [
  { label: "概览", value: "overview" },
  { label: "配置", value: "config" },
  { label: "解析或映射", value: "mapping" },
  { label: "预览", value: "preview" },
  { label: "数据点", value: "datapoints" },
  { label: "记录", value: "logs" },
];

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

watch(
  () => props.connection?.id,
  () => {
    activeTab.value = "overview";
  },
);

const typeLabel = computed(() => {
  const connection = props.connection;
  if (!connection) return "-";

  if (connection.type === "relational") {
    const dbType = connection.relationalConfig?.dbType || "";
    return dbTypeLabels[dbType] || dbType || "数据库";
  }

  if (connection.type === "mqtt") return "MQTT Broker";

  return connection.type || "未知类型";
});

const endpointText = computed(() => {
  const connection = props.connection;
  if (!connection) return "-";

  if (connection.type === "relational") {
    const config = connection.relationalConfig;
    if (!config) return "未配置";
    const host = [config.host, config.port].filter(Boolean).join(":");
    return [host, config.database].filter(Boolean).join(" / ") || "未配置";
  }

  if (connection.type === "mqtt") {
    const config = connection.mqttConfig;
    if (!config) return "未配置";
    const endpoint = [config.brokerUrl || config.host, config.port]
      .filter(Boolean)
      .join(":");
    return [endpoint, config.topic || config.defaultTopic]
      .filter(Boolean)
      .join(" / ") || "未配置";
  }

  return "未配置";
});

const statusText = computed(
  () =>
    statusLabels[props.connection?.status || "unknown"] ||
    props.connection?.status ||
    "未知",
);

const datapointText = computed(() => {
  const connection = props.connection;
  const count = connection?.datapointCount ?? connection?.dataPointCount;
  return typeof count === "number" ? `${count} 个` : "待映射";
});

const configRows = computed(() => {
  const connection = props.connection;
  if (!connection) return [];

  if (connection.type === "relational") {
    const config = connection.relationalConfig || {};
    return [
      { label: "数据库类型", value: typeLabel.value },
      { label: "Host", value: config.host || "-" },
      { label: "Port", value: config.port || "-" },
      { label: "Database", value: config.database || "-" },
    ];
  }

  if (connection.type === "mqtt") {
    const config = connection.mqttConfig || {};
    return [
      { label: "协议", value: config.protocol || "mqtt" },
      { label: "Broker", value: config.brokerUrl || config.host || "-" },
      { label: "Port", value: config.port || 1883 },
      { label: "Topic", value: config.topic || config.defaultTopic || "-" },
    ];
  }

  return [{ label: "类型", value: connection.type || "-" }];
});

const currentTabLabel = computed(
  () => tabs.find((tab) => tab.value === activeTab.value)?.label || "详情",
);
</script>

<style scoped>
.access-source-detail {
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-md);
  background: var(--dc-surface-raised);
  box-shadow: var(--dc-shadow-surface);
}

.access-source-detail__hero {
  margin: 12px;
  padding: 14px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-md);
  background: var(--dc-surface-subtle);
  color: var(--dc-text);
}

.access-source-detail__eyebrow {
  color: var(--dc-primary);
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0;
}

.access-source-detail__hero h3 {
  margin: 8px 0 4px;
  font-size: 17px;
  font-weight: 700;
}

.access-source-detail__hero p {
  margin: 0 0 14px;
  color: var(--dc-text-secondary);
  font-size: 12px;
}

.access-source-detail__tabs {
  padding: 0 12px;
}

.access-source-detail__tabs :deep(.el-tabs__header) {
  margin: 0;
}

.access-source-detail__tabs :deep(.el-tabs__item) {
  padding: 0 8px;
  color: var(--dc-text-secondary);
  font-size: 12px;
}

.access-source-detail__tabs :deep(.el-tabs__item.is-active) {
  color: var(--dc-primary);
  font-weight: 700;
}

.access-source-detail__tabs :deep(.el-tabs__active-bar) {
  background: var(--dc-primary);
}

.access-source-detail__body {
  min-height: 0;
  flex: 1;
  overflow-y: auto;
  padding: 12px;
}

.access-source-detail__list {
  margin: 0;
}

.access-source-detail__list div {
  display: flex;
  justify-content: space-between;
  gap: 14px;
  padding: 12px 0;
  border-bottom: 1px solid var(--dc-border);
}

.access-source-detail__list dt {
  color: var(--dc-text-secondary);
  font-size: 12px;
}

.access-source-detail__list dd {
  min-width: 0;
  margin: 0;
  color: var(--dc-text);
  font-size: 12px;
  font-weight: 700;
  text-align: right;
  word-break: break-all;
}

.access-source-detail__placeholder,
.access-source-detail__empty {
  height: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  color: var(--dc-text-secondary);
  text-align: center;
}

.access-source-detail__placeholder strong,
.access-source-detail__empty strong {
  color: var(--dc-text);
}

.access-source-detail__placeholder span,
.access-source-detail__empty span {
  max-width: 220px;
  font-size: 12px;
  line-height: 1.7;
}

.access-source-detail__empty-orb {
  width: 58px;
  height: 58px;
  border: 1px solid rgba(29, 78, 216, 0.2);
  border-radius: var(--dc-radius-md);
  background: var(--dc-primary-soft);
}
</style>
