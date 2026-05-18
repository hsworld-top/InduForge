<template>
  <article
    class="access-source-card"
    :class="{ 'is-active': active }"
    @click="handleCardClick"
  >
    <div class="access-source-card__top">
      <span
        class="access-source-card__icon"
        :class="`is-${resolveConnectionVisual(connection).category}`"
      >
        <component :is="resolveConnectionVisual(connection).icon" />
      </span>
      <div class="access-source-card__badges">
        <!-- Phase 2 仅配置角标 -->
        <StatusBadge v-if="isPhase2" tone="warning" text="仅配置" />
        <span
          class="access-source-card__status"
          :class="`is-${resolveStatusKey(connection.status)}`"
        >
          <span class="access-source-card__status-dot"></span>
          {{ resolveStatusText(connection.status) }}
        </span>
      </div>
    </div>

    <div class="access-source-card__body">
      <div class="access-source-card__name" :title="connection.name">
        {{ connection.name || "未命名连接" }}
      </div>
      <div class="access-source-card__type">
        <component
          :is="resolveConnectionVisual(connection).miniIcon"
          class="access-source-card__type-icon"
        />
        <span>{{ resolveConnectionType(connection) }}</span>
        <span class="access-source-card__divider">·</span>
        <span class="access-source-card__endpoint">
          {{ resolveConnectionEndpoint(connection) }}
        </span>
      </div>
    </div>

    <div class="access-source-card__divider-line"></div>

    <!-- 底部按钮区：click.stop 阻止冒泡到卡片的 handleCardClick -->
    <div class="access-source-card__bottom" @click.stop>
      <button
        type="button"
        class="access-source-card__open"
        @click="$emit('open', connection)"
      >
        <span>打开工作台</span>
        <IconTablerArrowRight class="access-source-card__open-icon" />
      </button>
      <button
        type="button"
        class="access-source-card__edit"
        :aria-label="`编辑连接 ${connection.name || ''}`"
        :title="`编辑连接 ${connection.name || ''}`"
        @click="$emit('edit', connection)"
      >
        <IconTablerSettings />
      </button>
    </div>
  </article>
</template>

<script setup lang="ts">
import IconTablerArrowRight from "~icons/tabler/arrow-right";
import IconTablerBuildingFactory2 from "~icons/tabler/building-factory-2";
import IconTablerDatabase from "~icons/tabler/database";
import IconTablerMessages from "~icons/tabler/messages";
import IconTablerSettings from "~icons/tabler/settings";
import StatusBadge from "@/components/shared/StatusBadge.vue";

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

const props = defineProps<{
  connection: AccessSourceConnection;
  active: boolean;
  /** 工业协议等 Phase 2 类型，显示「仅配置」角标 */
  isPhase2?: boolean;
}>();

const emit = defineEmits<{
  /** 单击卡片空白区域：A2 抽屉将监听，A1 仅写 URL 高亮 */
  (event: "open-detail", connection: AccessSourceConnection): void;
  /** 单击「打开工作台」按钮 */
  (event: "open", connection: AccessSourceConnection): void;
  /** 单击编辑按钮 */
  (event: "edit", connection: AccessSourceConnection): void;
}>();

function handleCardClick() {
  emit("open-detail", props.connection);
}

const databaseTypes = new Set([
  "relational",
  "mysql",
  "postgresql",
  "sqlserver",
  "tdengine",
  "redis",
]);
const streamTypes = new Set(["mqtt", "kafka", "websocket", "http"]);
const industrialTypes = new Set(["opcua", "opcda", "s7", "modbus"]);

const resolveConnectionCategory = (connection: AccessSourceConnection) => {
  const type = connection.type || "";
  if (databaseTypes.has(type)) return "database";
  if (streamTypes.has(type)) return "stream";
  if (industrialTypes.has(type)) return "industrial";
  if (type.includes("opc") || type.includes("modbus")) return "industrial";
  return "database";
};

const resolveConnectionVisual = (connection: AccessSourceConnection) => {
  const category = resolveConnectionCategory(connection);
  if (category === "stream") {
    return { category, icon: IconTablerMessages, miniIcon: IconTablerMessages };
  }
  if (category === "industrial") {
    return {
      category,
      icon: IconTablerBuildingFactory2,
      miniIcon: IconTablerBuildingFactory2,
    };
  }
  return { category, icon: IconTablerDatabase, miniIcon: IconTablerDatabase };
};

const resolveConnectionType = (connection: AccessSourceConnection) => {
  if (connection.type === "relational") {
    const dbType = connection.relationalConfig?.dbType || "";
    const dbTypeLabels: Record<string, string> = {
      mysql: "MySQL",
      postgresql: "PostgreSQL",
      sqlserver: "SQL Server",
    };
    return dbTypeLabels[dbType] || dbType || "数据库/时序库";
  }
  if (connection.type === "mqtt") return "MQTT Broker";
  const protocolLabels: Record<string, string> = {
    kafka: "Kafka Topic",
    http: "HTTP Source",
    websocket: "WebSocket",
    redis: "Redis",
    opcua: "OPC UA",
    opcda: "OPC DA",
    s7: "Siemens S7",
    modbus: "Modbus",
    tdengine: "TDengine",
  };
  if (connection.type && protocolLabels[connection.type]) {
    return protocolLabels[connection.type];
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
  const config = connection.config || {};
  if (connection.type === "kafka") {
    return (
      [config["brokers"], config["topic"]].filter(Boolean).join(" / ") ||
      "未配置 Topic"
    );
  }
  if (connection.type === "http") {
    return (
      [config["method"] || "GET", config["baseUrl"]].filter(Boolean).join(" ") ||
      "未配置 URL"
    );
  }
  if (connection.type === "websocket") {
    return (
      [config["url"], config["topic"]].filter(Boolean).join(" / ") ||
      "未配置 WebSocket"
    );
  }
  if (connection.type === "redis") {
    return (
      [config["address"], config["keyPattern"] || "*"].filter(Boolean).join(" / ") ||
      "未配置 Redis"
    );
  }
  if (connection.type === "opcua") {
    return (
      [config["endpoint"], config["securityMode"] || "none"]
        .filter(Boolean)
        .join(" / ") || "未配置 OPC UA"
    );
  }
  if (connection.type === "opcda") {
    return String(config["serverProgId"] || config["host"] || "未配置 OPC DA");
  }
  if (connection.type === "s7") {
    return [config["host"], config["port"]].filter(Boolean).join(":") || "未配置 S7";
  }
  if (connection.type === "modbus") {
    if (config["mode"] === "rtu") return "RTU / 串口配置";
    return (
      [config["host"], config["port"]].filter(Boolean).join(":") || "未配置 Modbus"
    );
  }
  if (connection.type === "tdengine") {
    return (
      [config["dsn"], config["database"]].filter(Boolean).join(" / ") ||
      "未配置 TDengine"
    );
  }
  return "等待接入配置";
};

const resolveStatusKey = (status?: string) => {
  if (status === "connected") return "connected";
  if (status === "disconnected") return "disconnected";
  if (status === "error" || status === "degraded") return "degraded";
  return "unknown";
};

const statusLabels: Record<string, string> = {
  connected: "在线",
  disconnected: "离线",
  error: "异常",
  degraded: "降级",
  unknown: "未知",
};

const resolveStatusText = (status?: string) =>
  statusLabels[status || "unknown"] || status || "未知";
</script>

<style scoped>
.access-source-card {
  min-height: 190px;
  display: flex;
  flex-direction: column;
  padding: 18px;
  border: 1px solid var(--dc-connection-card-border);
  border-radius: var(--dc-radius-md);
  background: var(--dc-connection-card-bg);
  box-shadow: var(--dc-connection-card-shadow);
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
  transform: translateY(-2px);
  border-color: color-mix(in oklch, var(--dc-primary) 28%, var(--dc-border));
  box-shadow: var(--dc-connection-card-hover-shadow);
}

.access-source-card.is-active {
  outline: 3px solid rgba(29, 78, 216, 0.1);
}

.access-source-card__top {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}

/* 右上角角标区：竖向排列或横向排列均可，间距 6px */
.access-source-card__badges {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
  justify-content: flex-end;
}

.access-source-card__icon {
  width: 48px;
  height: 48px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: var(--dc-radius-sm);
}

.access-source-card__icon svg {
  width: 28px;
  height: 28px;
}

.access-source-card__icon.is-database {
  background: var(--dc-connection-icon-database-bg);
  color: var(--dc-connection-icon-database-text);
}

.access-source-card__icon.is-stream {
  background: var(--dc-connection-icon-stream-bg);
  color: var(--dc-connection-icon-stream-text);
}

.access-source-card__icon.is-industrial {
  background: var(--dc-connection-icon-industrial-bg);
  color: var(--dc-connection-icon-industrial-text);
}

.access-source-card__status {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-height: 24px;
  padding: 3px 10px;
  border-radius: 999px;
  background: var(--dc-connection-status-unknown-bg);
  color: var(--dc-connection-status-unknown-text);
  font-size: 12px;
  font-weight: 700;
  white-space: nowrap;
}

.access-source-card__status-dot {
  width: 7px;
  height: 7px;
  border-radius: 999px;
  background: currentColor;
}

.access-source-card__status.is-connected {
  background: var(--dc-connection-status-online-bg);
  color: var(--dc-connection-status-online-text);
}

.access-source-card__status.is-disconnected {
  background: var(--dc-connection-status-offline-bg);
  color: var(--dc-connection-status-offline-text);
}

.access-source-card__status.is-degraded {
  background: var(--dc-connection-status-degraded-bg);
  color: var(--dc-connection-status-degraded-text);
}

.access-source-card__body {
  min-width: 0;
  margin-top: 28px;
}

.access-source-card__name {
  overflow: hidden;
  color: var(--dc-text);
  font-size: 20px;
  font-weight: 800;
  line-height: 1.25;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.access-source-card__type {
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 7px;
  margin-top: 10px;
  color: var(--dc-text-secondary);
  font-size: 14px;
  line-height: 1.45;
}

.access-source-card__type-icon {
  width: 16px;
  height: 16px;
  flex: 0 0 auto;
  color: var(--dc-text-secondary);
}

.access-source-card__divider,
.access-source-card__endpoint {
  color: var(--dc-text-secondary);
}

.access-source-card__endpoint {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.access-source-card__divider-line {
  height: 1px;
  margin: auto 0 16px;
  background: var(--dc-border);
  opacity: 0.6;
}

.access-source-card__bottom {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.access-source-card__open {
  min-width: 0;
  min-height: 36px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 0 15px;
  border: 1px solid var(--dc-primary);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-primary);
  color: var(--dc-surface-raised);
  font-size: 13px;
  font-weight: 800;
  transition:
    background-color 0.18s ease,
    border-color 0.18s ease,
    transform 0.18s ease;
}

.access-source-card__open span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.access-source-card__open:hover {
  transform: translateY(-1px);
  border-color: var(--dc-primary-hover);
  background: var(--dc-primary-hover);
}

.access-source-card__open-icon {
  width: 17px;
  height: 17px;
  flex: 0 0 auto;
}

.access-source-card__edit {
  width: 34px;
  height: 34px;
  flex: 0 0 auto;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 1px solid transparent;
  border-radius: var(--dc-radius-sm);
  background: transparent;
  color: #8a9aaf;
  transition:
    background-color 0.18s ease,
    color 0.18s ease;
}

.access-source-card__edit svg {
  width: 22px;
  height: 22px;
}

.access-source-card__edit:hover {
  background: var(--dc-surface-muted);
  color: var(--dc-primary);
}

@media (max-width: 760px) {
  .access-source-card {
    min-height: 178px;
    padding: 15px;
  }

  .access-source-card__name {
    font-size: 18px;
  }

  .access-source-card__type {
    font-size: 13px;
  }
}
</style>
