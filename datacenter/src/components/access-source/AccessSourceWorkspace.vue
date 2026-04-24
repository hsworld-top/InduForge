<template>
  <div class="access-source-workspace">
    <aside class="access-source-workspace__filters">
      <div class="access-source-workspace__panel-title">接入分类</div>

      <button
        v-for="category in categories"
        :key="category.value"
        type="button"
        class="access-source-workspace__category"
        :class="{ 'is-active': activeCategory === category.value }"
        @click="activeCategory = category.value"
      >
        <span>{{ category.label }}</span>
        <small>{{ category.hint }}</small>
      </button>

      <button
        type="button"
        class="access-source-workspace__create"
        @click="$emit('create')"
      >
        <IconTablerPlus class="access-source-workspace__create-icon" />
        <span>新增接入</span>
      </button>

      <button
        type="button"
        class="access-source-workspace__ghost"
        @click="$emit('refresh')"
      >
        刷新接入源
      </button>
    </aside>

    <section class="access-source-workspace__list-panel">
      <div class="access-source-workspace__list-head">
        <div>
          <div class="access-source-workspace__panel-title">接入源</div>
          <p>单击查看详情，双击进入旧表查询或 MQTT 管理工作台。</p>
        </div>
        <el-button
          size="small"
          :disabled="!selectedConnection"
          @click="openSelectedConnection"
        >
          打开旧工作台
        </el-button>
      </div>

      <AccessSourceList
        :connections="filteredConnections"
        :selected-connection-id="selectedConnection?.id"
        @select="handleSelect"
        @open="handleOpen"
      />
    </section>

    <AccessSourceDetail
      :connection="selectedConnection"
      @open="handleOpen"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from "vue";
import IconTablerPlus from "~icons/tabler/plus";
import AccessSourceDetail from "./AccessSourceDetail.vue";
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
};

const props = defineProps<{
  connections: AccessSourceConnection[];
  selectedConnectionId?: string | null;
  projectId?: string | null;
}>();

const emit = defineEmits<{
  (event: "create"): void;
  (event: "refresh"): void;
  (event: "open", connection: AccessSourceConnection): void;
}>();

const activeCategory = ref("all");
const localSelectedId = ref<string | null>(props.selectedConnectionId || null);

const categories = [
  { label: "全部", value: "all", hint: "所有接入" },
  { label: "数据库/时序库", value: "database", hint: "SQL 与时序源" },
  { label: "消息/流", value: "stream", hint: "MQTT 与流式主题" },
  { label: "工业协议", value: "industrial", hint: "OPC UA、Modbus" },
  { label: "设备模板", value: "template", hint: "设备模型复用" },
];

const filteredConnections = computed(() => {
  if (activeCategory.value === "all") return props.connections;
  return props.connections.filter(
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
      !nextConnections.some((connection) => connection.id === localSelectedId.value)
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

const openSelectedConnection = () => {
  if (selectedConnection.value) {
    emit("open", selectedConnection.value);
  }
};

const resolveCategory = (connection: AccessSourceConnection) => {
  if (connection.type === "relational") return "database";
  if (connection.type === "mqtt") return "stream";
  if (connection.type?.includes("template")) return "template";
  if (connection.type?.includes("opc") || connection.type?.includes("modbus")) {
    return "industrial";
  }
  return "all";
};
</script>

<style scoped>
.access-source-workspace {
  height: 100%;
  display: grid;
  grid-template-columns: minmax(210px, 260px) minmax(0, 1fr) minmax(280px, 340px);
  gap: 12px;
}

.access-source-workspace__filters,
.access-source-workspace__list-panel {
  min-height: 0;
  overflow: hidden;
  border: 1px solid var(--dc-line, #d6cebf);
  border-radius: 8px;
  background: rgba(255, 253, 247, 0.94);
  box-shadow: var(--dc-shadow, 0 14px 34px rgba(48, 42, 32, 0.11));
}

.access-source-workspace__filters {
  padding: 12px;
  background: #fbfaf4;
}

.access-source-workspace__panel-title {
  margin-bottom: 12px;
  color: var(--dc-ink, #20231f);
  font-size: 14px;
  font-weight: 800;
}

.access-source-workspace__category {
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  margin-bottom: 8px;
  padding: 10px 12px;
  border: 1px solid #ded5c6;
  border-radius: 7px;
  background: #f1eee5;
  color: #465047;
  text-align: left;
}

.access-source-workspace__category small {
  color: #7c8078;
  font-size: 11px;
}

.access-source-workspace__category.is-active {
  border-color: #cfe1dc;
  background: var(--dc-soft, #e8f0ed);
  color: #245b60;
  font-weight: 900;
}

.access-source-workspace__create,
.access-source-workspace__ghost {
  width: 100%;
  margin-top: 14px;
  border-radius: 7px;
  font-size: 13px;
  font-weight: 800;
}

.access-source-workspace__create {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 12px;
  border: 0;
  border: 1px solid var(--dc-accent, #197278);
  background: var(--dc-accent, #197278);
  color: #fffdf7;
}

.access-source-workspace__create-icon {
  width: 16px;
  height: 16px;
}

.access-source-workspace__ghost {
  padding: 11px;
  border: 1px solid #bdd9d2;
  background: var(--dc-soft, #e8f0ed);
  color: #31565a;
}

.access-source-workspace__list-panel {
  display: flex;
  flex-direction: column;
}

.access-source-workspace__list-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
  padding: 13px 14px 11px;
  border-bottom: 1px solid var(--dc-line, #d6cebf);
  background: var(--dc-paper, #fffdf7);
}

.access-source-workspace__list-head p {
  margin: 2px 0 0;
  color: var(--dc-muted, #687066);
  font-size: 12px;
}

@media (max-width: 1120px) {
  .access-source-workspace {
    grid-template-columns: 220px minmax(0, 1fr);
  }
}

@media (max-width: 760px) {
  .access-source-workspace {
    grid-template-columns: 1fr;
    overflow-y: auto;
  }

  .access-source-workspace__filters,
  .access-source-workspace__list-panel {
    min-height: 320px;
  }
}
</style>
