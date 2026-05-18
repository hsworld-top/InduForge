<template>
  <div class="access-source-workspace">
    <section class="access-source-workspace__panel">
      <!-- 头部：标题 + 右侧 toolbar -->
      <header class="access-source-workspace__head">
        <div class="access-source-workspace__head-title">
          <h2>接入源</h2>
          <p>通过卡片打开工作台或查看详情。</p>
        </div>

        <div class="access-source-workspace__toolbar">
          <!-- 搜索框 -->
          <el-input
            v-model="searchInputValue"
            class="access-source-workspace__search"
            size="small"
            clearable
            placeholder="搜索名称"
            :prefix-icon="SearchIcon"
            @input="handleSearchInput"
            @clear="handleSearchClear"
          />

          <!-- 类型筛选 pill -->
          <el-popover
            trigger="click"
            placement="bottom-start"
            :width="160"
            popper-class="access-source-workspace__popover"
          >
            <template #reference>
              <PillButton :active="filterType !== 'all'">
                {{ typeLabel }}
              </PillButton>
            </template>
            <div class="access-source-workspace__pop-list">
              <button
                v-for="item in typeOptions"
                :key="item.value"
                type="button"
                class="access-source-workspace__pop-item"
                :class="{ 'is-active': filterType === item.value }"
                @click="selectType(item.value)"
              >
                {{ item.label }}
              </button>
            </div>
          </el-popover>

          <!-- 状态筛选 pill -->
          <el-popover
            trigger="click"
            placement="bottom-start"
            :width="130"
            popper-class="access-source-workspace__popover"
          >
            <template #reference>
              <PillButton :active="filterStatus !== 'all'">
                {{ statusLabel }}
              </PillButton>
            </template>
            <div class="access-source-workspace__pop-list">
              <button
                v-for="item in statusOptions"
                :key="item.value"
                type="button"
                class="access-source-workspace__pop-item"
                :class="{ 'is-active': filterStatus === item.value }"
                @click="selectStatus(item.value)"
              >
                {{ item.label }}
              </button>
            </div>
          </el-popover>

          <!-- 刷新 -->
          <button
            type="button"
            class="access-source-workspace__icon-btn"
            title="刷新"
            @click="$emit('refresh')"
          >
            <IconTablerRefresh class="access-source-workspace__icon-btn-icon" />
          </button>

          <!-- 新增连接 -->
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

      <AccessSourceList
        :connections="filteredConnections"
        :selected-connection-id="activeConnectionId"
        @open-detail="handleOpenDetail"
        @open="handleOpen"
        @edit="handleEdit"
        @create="$emit('create')"
      />
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import { Search } from "@element-plus/icons-vue";
import IconTablerPlus from "~icons/tabler/plus";
import IconTablerRefresh from "~icons/tabler/refresh";
import dataAPI from "@/api/data.api";
import AccessSourceList from "./AccessSourceList.vue";
import PillButton from "@/components/shared/PillButton.vue";

/* Search 图标赋值给变量，传给 el-input prefix-icon */
const SearchIcon = Search;

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

const route = useRoute();
const router = useRouter();

/* ── URL 同步：从 query 读取初始筛选值 ── */
const filterQ = ref(String(route.query.q || ""));
const filterType = ref(String(route.query.type || "all"));
const filterStatus = ref(String(route.query.status || "all"));

/* 搜索框双向绑定值（防抖前的输入缓存） */
const searchInputValue = ref(filterQ.value);

/* 300ms 防抖 timer */
let searchDebounceTimer: ReturnType<typeof setTimeout> | null = null;

const handleSearchInput = (val: string) => {
  if (searchDebounceTimer) clearTimeout(searchDebounceTimer);
  searchDebounceTimer = setTimeout(() => {
    filterQ.value = val.trim();
    syncQuery();
  }, 300);
};

const handleSearchClear = () => {
  if (searchDebounceTimer) clearTimeout(searchDebounceTimer);
  filterQ.value = "";
  searchInputValue.value = "";
  syncQuery();
};

/* 当前激活的连接 id：优先 URL params，其次 props，最后 null */
const activeConnectionId = computed<string | null>(() => {
  const paramId = String(route.params.objectId || "");
  if (paramId) return paramId;
  return props.selectedConnectionId || null;
});

/* 把当前筛选写入 URL query（仅写入非空字段）*/
function syncQuery() {
  const query: Record<string, string> = {};
  if (filterQ.value) query.q = filterQ.value;
  if (filterType.value && filterType.value !== "all") query.type = filterType.value;
  if (filterStatus.value && filterStatus.value !== "all")
    query.status = filterStatus.value;
  void router.replace({ query });
}

/* ── 类型筛选选项 ── */
const typeOptions = [
  { label: "全部", value: "all" },
  { label: "数据库", value: "database" },
  { label: "消息/流", value: "stream" },
  { label: "工业协议", value: "industrial" },
];

const typeLabel = computed(() => {
  const found = typeOptions.find((o) => o.value === filterType.value);
  return found ? (filterType.value === "all" ? "类型" : found.label) : "类型";
});

const selectType = (val: string) => {
  filterType.value = val;
  syncQuery();
};

/* ── 状态筛选选项 ── */
const statusOptions = [
  { label: "全部", value: "all" },
  { label: "在线", value: "connected" },
  { label: "离线", value: "disconnected" },
  { label: "异常", value: "error" },
  { label: "未知", value: "unknown" },
];

const statusLabel = computed(() => {
  const found = statusOptions.find((o) => o.value === filterStatus.value);
  return found
    ? filterStatus.value === "all"
      ? "状态"
      : found.label
    : "状态";
});

const selectStatus = (val: string) => {
  filterStatus.value = val;
  syncQuery();
};

/* ── 分类判断（与旧 resolveCategory 逻辑一致）── */
const resolveCategory = (connection: AccessSourceConnection) => {
  const type = connection.type || "";
  if (
    ["relational", "mysql", "postgresql", "sqlserver", "tdengine", "redis"].includes(
      type,
    )
  )
    return "database";
  if (["mqtt", "kafka", "websocket", "http"].includes(type)) return "stream";
  if (["opcua", "opcda", "s7", "modbus"].includes(type)) return "industrial";
  if (type.includes("opc") || type.includes("modbus")) return "industrial";
  return "all";
};

/* 状态归一：connected→connected, disconnected→disconnected, error/degraded→error, 其他→unknown */
const resolveStatusKey = (status?: string) => {
  if (status === "connected") return "connected";
  if (status === "disconnected") return "disconnected";
  if (status === "error" || status === "degraded") return "error";
  return "unknown";
};

/* ── SQL 数据点数量本地加载（过渡期保留） ── */
/* @deprecated A1 临时保留，后续随 A2/A3 移到 store */
const localDatapointCounts = ref<Record<string, number>>({});

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
        const queries =
          queryResponse.data?.queries || queryResponse.data || [];
        const sourceIds = queries
          .map((query: { id?: string }) => query.id)
          .filter(Boolean);
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
        nextCounts[connection.id] =
          pointResponse.data?.datapoints?.length || 0;
      } catch {
        const fallback =
          connection.datapointCount ?? connection.dataPointCount;
        if (typeof fallback === "number") nextCounts[connection.id] = fallback;
      }
    }),
  );
  localDatapointCounts.value = nextCounts;
};

watch(
  () => [props.projectId, props.connections.map((c) => c.id).join(",")],
  () => {
    void loadSqlDatapointCounts();
  },
  { immediate: true },
);

/* ── 前端过滤（基于 props.connections）── */
const filteredConnections = computed(() => {
  let list = props.connections.map((connection) => {
    const count = localDatapointCounts.value[connection.id];
    if (typeof count !== "number") return connection;
    return { ...connection, datapointCount: count, dataPointCount: count };
  });

  /* 名称搜索（大小写不敏感前缀模糊匹配）*/
  if (filterQ.value) {
    const lower = filterQ.value.toLowerCase();
    list = list.filter((c) => (c.name || "").toLowerCase().includes(lower));
  }

  /* 类型筛选 */
  if (filterType.value !== "all") {
    list = list.filter((c) => resolveCategory(c) === filterType.value);
  }

  /* 状态筛选 */
  if (filterStatus.value !== "all") {
    list = list.filter(
      (c) => resolveStatusKey(c.status) === filterStatus.value,
    );
  }

  return list;
});

/* ── 事件处理 ── */
/* open-detail：写 URL，A2 抽屉将监听 objectId；A1 仅高亮 */
const handleOpenDetail = (connection: AccessSourceConnection) => {
  void router.replace({
    params: { ...route.params, objectId: connection.id },
    query: route.query,
  });
};

const handleOpen = (connection: AccessSourceConnection) => {
  /* 不动 URL，上抛给 DataCenterNew 走旧 SqlWorkbench/MqttWorkbench */
  emit("open", connection);
};

const handleEdit = (connection: AccessSourceConnection) => {
  emit("edit", connection);
};
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

/* 头部：标题在左，toolbar 在右 */
.access-source-workspace__head {
  flex: 0 0 auto;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 18px;
  padding: 18px 20px 14px;
  border-bottom: 1px solid var(--dc-border);
  background: var(--dc-surface-raised);
}

.access-source-workspace__head-title h2 {
  margin: 0;
  color: var(--dc-text);
  font-size: 20px;
  font-weight: 700;
  line-height: 1.3;
}

.access-source-workspace__head-title p {
  margin: 4px 0 0;
  color: var(--dc-text-secondary);
  font-size: 13px;
}

/* toolbar：搜索 + pill + 刷新 + 新增 */
.access-source-workspace__toolbar {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  flex: 0 0 auto;
}

.access-source-workspace__search {
  width: 200px;
}

/* 刷新图标按钮 */
.access-source-workspace__icon-btn {
  width: 32px;
  height: 32px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
  color: var(--dc-text-secondary);
  transition:
    border-color 0.18s ease,
    color 0.18s ease;
}

.access-source-workspace__icon-btn:hover {
  border-color: color-mix(in oklch, var(--dc-primary) 24%, var(--dc-border));
  color: var(--dc-primary);
}

.access-source-workspace__icon-btn-icon {
  width: 16px;
  height: 16px;
}

.access-source-workspace__primary {
  min-height: 32px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 7px;
  padding: 0 12px;
  border: 1px solid var(--dc-primary);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-primary);
  color: var(--dc-surface-raised);
  font-size: 13px;
  font-weight: 700;
  transition:
    background-color 0.18s ease,
    border-color 0.18s ease,
    transform 0.18s ease;
}

.access-source-workspace__primary:hover {
  transform: translateY(-1px);
  background: var(--dc-primary-hover);
  border-color: var(--dc-primary-hover);
}

.access-source-workspace__action-icon {
  width: 16px;
  height: 16px;
}

/* popover 内选项列表 */
.access-source-workspace__pop-list {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 4px 0;
}

.access-source-workspace__pop-item {
  width: 100%;
  padding: 7px 12px;
  border: none;
  border-radius: var(--dc-radius-sm);
  background: transparent;
  color: var(--dc-text-secondary);
  font-size: 13px;
  text-align: left;
  cursor: pointer;
  transition: background-color 0.14s ease, color 0.14s ease;
}

.access-source-workspace__pop-item:hover {
  background: var(--dc-surface-muted);
  color: var(--dc-text);
}

.access-source-workspace__pop-item.is-active {
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
  font-weight: 700;
}

@media (max-width: 1120px) {
  .access-source-workspace__head {
    align-items: flex-start;
    flex-direction: column;
  }
}

@media (max-width: 760px) {
  .access-source-workspace__head {
    padding-left: 14px;
    padding-right: 14px;
  }

  .access-source-workspace__toolbar {
    width: 100%;
  }

  .access-source-workspace__search {
    width: 100%;
  }

  .access-source-workspace__primary {
    flex: 1;
  }
}
</style>
