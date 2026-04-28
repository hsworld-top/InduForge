<template>
  <div class="access-source-workspace">
    <section class="access-source-workspace__panel">
      <header class="access-source-workspace__head">
        <div>
          <div class="access-source-workspace__eyebrow">连接管理</div>
          <h2>接入源</h2>
          <p>通过卡片打开工作台、编辑配置或新增数据接入。</p>
        </div>
        <div class="access-source-workspace__head-actions">
          <button
            type="button"
            class="access-source-workspace__ghost"
            @click="$emit('refresh')"
          >
            <IconTablerRefresh class="access-source-workspace__action-icon" />
            <span>刷新</span>
          </button>
          <button
            type="button"
            class="access-source-workspace__primary"
            @click="$emit('create')"
          >
            <IconTablerPlus class="access-source-workspace__action-icon" />
            <span>新增连接</span>
          </button>
        </div>
      </header>

      <div class="access-source-workspace__toolbar">
        <div class="access-source-workspace__categories" aria-label="接入分类">
          <button
            v-for="category in categories"
            :key="category.value"
            type="button"
            class="access-source-workspace__category"
            :class="{ 'is-active': activeCategory === category.value }"
            @click="activeCategory = category.value"
          >
            <span>{{ category.label }}</span>
            <small>{{ countByCategory(category.value) }}</small>
          </button>
        </div>
      </div>

      <AccessSourceList
        :connections="filteredConnections"
        :selected-connection-id="selectedConnection?.id"
        @select="handleSelect"
        @open="handleOpen"
        @edit="handleEdit"
        @create="$emit('create')"
      />
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from "vue";
import IconTablerPlus from "~icons/tabler/plus";
import IconTablerRefresh from "~icons/tabler/refresh";
import dataAPI from "@/api/data.api";
import AccessSourceList from "./AccessSourceList.vue";

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
  config?: Record<string, any>;
};

const props = defineProps<{
  connections: AccessSourceConnection[];
  selectedConnectionId?: string | null;
  projectId?: string | number | null;
}>();

const emit = defineEmits<{
  (event: "create"): void;
  (event: "refresh"): void;
  (event: "open", connection: AccessSourceConnection): void;
  (event: "edit", connection: AccessSourceConnection): void;
}>();

const activeCategory = ref("all");
const localSelectedId = ref<string | null>(props.selectedConnectionId || null);
const localDatapointCounts = ref<Record<string, number>>({});

const categories = [
  { label: "全部", value: "all", hint: "所有接入" },
  { label: "数据库/时序库", value: "database", hint: "SQL 与时序源" },
  { label: "消息/流", value: "stream", hint: "MQTT 与流式主题" },
  { label: "工业协议", value: "industrial", hint: "OPC UA、S7、Modbus" },
];

const filteredConnections = computed(() => {
  const enrichedConnections = props.connections.map((connection) => {
    const count = localDatapointCounts.value[connection.id];
    if (typeof count !== "number") return connection;
    return {
      ...connection,
      datapointCount: count,
      dataPointCount: count,
    };
  });
  if (activeCategory.value === "all") return enrichedConnections;
  return enrichedConnections.filter(
    (connection) => resolveCategory(connection) === activeCategory.value,
  );
});

const selectedConnection = computed(() => {
  const byLocalId = filteredConnections.value.find(
    (connection) => connection.id === localSelectedId.value,
  );
  return byLocalId || filteredConnections.value[0] || null;
});

watch(
  () => props.selectedConnectionId,
  (selectedConnectionId) => {
    if (selectedConnectionId) {
      localSelectedId.value = selectedConnectionId;
    }
  },
);

watch(
  filteredConnections,
  (nextConnections) => {
    if (
      nextConnections.length > 0 &&
      !nextConnections.some(
        (connection) => connection.id === localSelectedId.value,
      )
    ) {
      localSelectedId.value = nextConnections[0].id;
    }
  },
  { immediate: true },
);

const handleSelect = (connection: AccessSourceConnection) => {
  localSelectedId.value = connection.id;
};

const handleOpen = (connection: AccessSourceConnection) => {
  localSelectedId.value = connection.id;
  emit("open", connection);
};

const handleEdit = (connection: AccessSourceConnection) => {
  localSelectedId.value = connection.id;
  emit("edit", connection);
};

const resolveCategory = (connection: AccessSourceConnection) => {
  if (connection.type === "relational") return "database";
  if (["mqtt", "kafka", "websocket", "http"].includes(connection.type || "")) {
    return "stream";
  }
  if (["redis", "tdengine"].includes(connection.type || "")) return "database";
  if (["opcua", "s7", "modbus"].includes(connection.type || "")) {
    return "industrial";
  }
  if (connection.type?.includes("opc") || connection.type?.includes("modbus")) {
    return "industrial";
  }
  return "all";
};

const countByCategory = (category: string) => {
  if (category === "all") return props.connections.length;
  return props.connections.filter(
    (connection) => resolveCategory(connection) === category,
  ).length;
};

const isSqlConnection = (connection: AccessSourceConnection) => {
  if (connection.type === "relational") return true;
  return ["mysql", "postgresql", "sqlserver", "tdengine"].includes(
    connection.type || "",
  );
};

const loadSqlDatapointCounts = async () => {
  if (!props.projectId) {
    localDatapointCounts.value = {};
    return;
  }

  const sqlConnections = props.connections.filter(isSqlConnection);
  const nextCounts: Record<string, number> = {};
  await Promise.all(
    sqlConnections.map(async (connection) => {
      try {
        const queryResponse = await dataAPI.getQueries(props.projectId, {
          connectionId: connection.id,
          queryType: "sql",
          page: 1,
          pageSize: 100,
        });
        const queries = queryResponse.data?.queries || queryResponse.data || [];
        const sourceIds = queries.map((query) => query.id).filter(Boolean);
        if (sourceIds.length === 0) {
          nextCounts[connection.id] = 0;
          return;
        }
        const pointResponse = await dataAPI.getDataPoints(props.projectId, {
          type: "db.query",
          sourceIds: sourceIds.join(","),
          page: 1,
          pageSize: 200,
        });
        nextCounts[connection.id] = pointResponse.data?.datapoints?.length || 0;
      } catch {
        const fallback = connection.datapointCount ?? connection.dataPointCount;
        if (typeof fallback === "number") nextCounts[connection.id] = fallback;
      }
    }),
  );
  localDatapointCounts.value = nextCounts;
};

watch(
  () => [props.projectId, props.connections.map((item) => item.id).join(",")],
  () => {
    void loadSqlDatapointCounts();
  },
  { immediate: true },
);
</script>

<style scoped>
.access-source-workspace {
  height: 100%;
  min-height: 0;
  display: flex;
}

.access-source-workspace__panel {
  min-width: 0;
  min-height: 0;
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-md);
  background: linear-gradient(180deg, #fbfcff 0%, var(--dc-surface) 100%);
  box-shadow: var(--dc-shadow-surface);
}

.access-source-workspace__head {
  flex: 0 0 auto;
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 18px;
  padding: 18px 20px 14px;
  border-bottom: 1px solid var(--dc-border);
  background: var(--dc-surface-raised);
}

.access-source-workspace__eyebrow {
  margin-bottom: 5px;
  color: var(--dc-primary);
  font-size: 12px;
  font-weight: 700;
}

.access-source-workspace__head h2 {
  margin: 0;
  color: var(--dc-text);
  font-size: 20px;
  font-weight: 700;
  line-height: 1.3;
}

.access-source-workspace__head p {
  margin: 4px 0 0;
  color: var(--dc-text-secondary);
  font-size: 13px;
}

.access-source-workspace__head-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 0 0 auto;
}

.access-source-workspace__toolbar {
  flex: 0 0 auto;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 12px 20px 0;
}

.access-source-workspace__categories {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
}

.access-source-workspace__category {
  min-height: 32px;
  display: flex;
  align-items: center;
  gap: 7px;
  padding: 6px 10px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
  color: var(--dc-text-secondary);
  text-align: left;
  font-size: 13px;
  font-weight: 700;
  transition:
    border-color 0.18s ease,
    background-color 0.18s ease,
    color 0.18s ease;
}

.access-source-workspace__category small {
  min-width: 20px;
  padding: 1px 6px;
  border-radius: 999px;
  background: var(--dc-surface-muted);
  color: var(--dc-text-secondary);
  font-size: 11px;
  font-weight: 700;
  text-align: center;
}

.access-source-workspace__category.is-active {
  border-color: rgba(29, 78, 216, 0.34);
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
  font-weight: 700;
}

.access-source-workspace__primary,
.access-source-workspace__ghost {
  min-height: 32px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 7px;
  padding: 0 12px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  font-size: 13px;
  font-weight: 700;
  transition:
    background-color 0.18s ease,
    border-color 0.18s ease,
    color 0.18s ease,
    transform 0.18s ease;
}

.access-source-workspace__primary {
  border: 1px solid var(--dc-primary);
  background: var(--dc-primary);
  color: var(--dc-surface-raised);
}

.access-source-workspace__action-icon {
  width: 16px;
  height: 16px;
}

.access-source-workspace__ghost {
  background: var(--dc-surface-raised);
  color: var(--dc-text-secondary);
}

.access-source-workspace__primary:hover,
.access-source-workspace__ghost:hover {
  transform: translateY(-1px);
}

.access-source-workspace__primary:hover {
  background: var(--dc-primary-hover);
  border-color: var(--dc-primary-hover);
}

.access-source-workspace__ghost:hover {
  border-color: color-mix(in oklch, var(--dc-primary) 24%, var(--dc-border));
  color: var(--dc-primary);
}

@media (max-width: 1120px) {
  .access-source-workspace__head {
    align-items: stretch;
    flex-direction: column;
  }
}

@media (max-width: 760px) {
  .access-source-workspace__head,
  .access-source-workspace__toolbar {
    padding-left: 14px;
    padding-right: 14px;
  }

  .access-source-workspace__head-actions {
    width: 100%;
  }

  .access-source-workspace__primary,
  .access-source-workspace__ghost {
    flex: 1;
  }
}
</style>
