<template>
  <div
    class="connection-list w-64 bg-white dark:bg-gray-800 border-r border-gray-200 dark:border-gray-700 flex flex-col overflow-hidden"
  >
    <div class="px-2.5 py-2 flex-1 overflow-y-auto min-h-0">
      <div class="space-y-2">
        <div class="section-block">
          <div class="section-title">{{ t("connection.basicConfig") }}</div>
          <div
            class="section-entry flex items-center gap-2 px-3 py-2 rounded-md cursor-pointer"
            @click="handleOpenDataPoints"
          >
            <IconTablerDatabase class="w-4 h-4 text-indigo-500" />
            <span class="text-sm text-gray-700 dark:text-gray-200">{{
              t("connection.datapoints")
            }}</span>
          </div>
        </div>

        <div class="section-block">
          <div class="section-head">
            <span class="section-title">{{
              t("connection.connectionManagement")
            }}</span>
            <div class="section-actions">
              <el-button
                size="small"
                circle
                @click="handleCreate"
                class="header-btn icon-btn"
                :title="t('actions.createConnection')"
              >
                <IconTablerPlus class="w-4 h-4" />
              </el-button>
              <el-button
                size="small"
                circle
                @click="handleRefresh"
                class="header-btn icon-btn"
                :title="t('actions.refresh')"
              >
                <IconTablerRefresh class="w-4 h-4" />
              </el-button>
            </div>
          </div>
          <div class="section-tools">
            <el-input
              v-model="searchText"
              size="small"
              :placeholder="t('connection.searchPlaceholder')"
              clearable
              class="connection-search flex-1"
            />
          </div>
          <div class="connection-items">
            <ConnectionItem
              v-for="connection in filteredConnections"
              :key="connection.id"
              :connection="connection"
              :is-selected="selectedConnectionId === connection.id"
              :is-expanded="getConnectionState(connection.id).expanded"
              @click="handleSelect"
              @dblclick="handleDblClick"
              @contextmenu="handleContextMenu"
            >
              <template #expanded v-if="connection.type === 'relational'">
                <div class="mt-2 space-y-2" @dblclick.stop>
                  <!-- 查询列表 -->
                  <div
                    class="space-y-1 pb-2 border-b border-gray-200 dark:border-gray-700"
                  >
                    <div
                      class="flex items-center justify-between text-xs font-medium text-gray-600 dark:text-gray-300 bg-blue-50 dark:bg-blue-900/20 px-2 py-1 rounded"
                      @dblclick.stop
                      @contextmenu.prevent.stop="
                        handleQueriesSectionContextMenu($event, connection)
                      "
                    >
                      <div
                        class="flex items-center space-x-2 flex-1 cursor-pointer"
                        @click.stop="toggleSection(connection.id, 'queries')"
                      >
                        <component
                          :is="
                            getConnectionState(connection.id).queriesExpanded
                              ? IconTablerChevronDown
                              : IconTablerChevronRight
                          "
                          class="text-blue-500 w-4 h-4"
                        />
                        <span class="text-blue-700 dark:text-blue-300">{{
                          t("connection.querySection")
                        }}</span>
                      </div>
                      <div class="flex items-center space-x-1">
                        <el-tooltip
                          :content="t('connection.refreshQueryList')"
                          placement="top"
                        >
                          <button
                            class="p-1 hover:bg-blue-200 dark:hover:bg-blue-700 rounded"
                            @click.stop="refreshQueries(connection.id)"
                          >
                            <IconTablerRefresh
                              class="w-3 h-3 text-blue-600 dark:text-blue-400"
                            />
                          </button>
                        </el-tooltip>
                        <span
                          v-if="
                            !getConnectionState(connection.id).loadingQueries
                          "
                          class="text-blue-600 dark:text-blue-400"
                        >
                          {{ getConnectionState(connection.id).queries.length }}
                        </span>
                      </div>
                    </div>
                    <div
                      v-if="getConnectionState(connection.id).loadingQueries"
                      class="pl-6 text-xs text-gray-500"
                    >
                      {{ t("connection.loadingQueries") }}
                    </div>
                    <template
                      v-else-if="
                        getConnectionState(connection.id).queriesExpanded
                      "
                    >
                      <div
                        v-if="
                          getConnectionState(connection.id).queries.length === 0
                        "
                        class="pl-6 text-xs text-gray-500"
                      >
                        {{ t("connection.emptyQueries") }}
                      </div>
                      <ul
                        v-else
                        class="space-y-0.5 pl-6 border-l-2 border-blue-300 dark:border-blue-700"
                      >
                        <li
                          v-for="query in getConnectionState(connection.id)
                            .queries"
                          :key="query.id"
                          class="flex items-center justify-between text-xs text-gray-700 dark:text-gray-300 hover:text-blue-800 dark:hover:text-blue-200 hover:bg-blue-200 dark:hover:bg-blue-800/60 cursor-pointer group px-2 py-1.5 rounded transition-all duration-150 hover:translate-x-0.5 hover:shadow-sm"
                          @dblclick.stop="
                            handleQueryDblClick(connection, query)
                          "
                          @contextmenu.prevent.stop="
                            handleQueryContextMenu($event, connection, query)
                          "
                        >
                          <el-tooltip
                            :content="query.name"
                            placement="top"
                            :show-after="500"
                          >
                            <span class="truncate flex-1 font-medium">{{
                              query.name
                            }}</span>
                          </el-tooltip>
                          <el-tooltip
                            :content="t('actions.delete')"
                            placement="top"
                          >
                            <button
                              class="opacity-0 group-hover:opacity-100 p-0.5 hover:bg-red-100 dark:hover:bg-red-900 rounded transition-opacity"
                              @click.stop="handleDeleteQuery(connection, query)"
                            >
                              <IconTablerTrash class="w-3 h-3 text-red-500" />
                            </button>
                          </el-tooltip>
                        </li>
                      </ul>
                    </template>
                  </div>

                  <!-- 表列表 -->
                  <div class="space-y-1 pt-1">
                    <div
                      class="flex items-center justify-between text-xs font-medium text-gray-600 dark:text-gray-300 bg-green-50 dark:bg-green-900/20 px-2 py-1 rounded"
                      @dblclick.stop
                      @contextmenu.prevent.stop="
                        handleTablesSectionContextMenu($event, connection)
                      "
                    >
                      <div
                        class="flex items-center space-x-2 flex-1 cursor-pointer"
                        @click.stop="toggleSection(connection.id, 'tables')"
                      >
                        <component
                          :is="
                            getConnectionState(connection.id).tablesExpanded
                              ? IconTablerChevronDown
                              : IconTablerChevronRight
                          "
                          class="text-green-500 w-4 h-4"
                        />
                        <span class="text-green-700 dark:text-green-300">{{
                          t("connection.tableSection")
                        }}</span>
                      </div>
                      <div class="flex items-center space-x-1">
                        <el-tooltip
                          :content="t('connection.refreshTableList')"
                          placement="top"
                        >
                          <button
                            class="p-1 hover:bg-green-200 dark:hover:bg-green-700 rounded"
                            @click.stop="refreshTables(connection.id)"
                          >
                            <IconTablerRefresh
                              class="w-3 h-3 text-green-600 dark:text-green-400"
                            />
                          </button>
                        </el-tooltip>
                        <el-tooltip
                          :content="t('connection.viewTableList')"
                          placement="top"
                        >
                          <button
                            class="p-1 hover:bg-green-200 dark:hover:bg-green-700 rounded"
                            @click.stop="handleViewTableList(connection)"
                          >
                            <IconTablerTable
                              class="w-3 h-3 text-green-600 dark:text-green-400"
                            />
                          </button>
                        </el-tooltip>
                        <span
                          v-if="!getConnectionState(connection.id).loading"
                          class="text-green-600 dark:text-green-400"
                        >
                          {{ getConnectionState(connection.id).tables.length }}
                        </span>
                      </div>
                    </div>
                    <div
                      v-if="getConnectionState(connection.id).loading"
                      class="pl-6 text-xs text-gray-500"
                    >
                      {{ t("connection.loadingTables") }}
                    </div>
                    <template
                      v-else-if="
                        getConnectionState(connection.id).tablesExpanded
                      "
                    >
                      <div
                        v-if="
                          getConnectionState(connection.id).tables.length === 0
                        "
                        class="pl-6 text-xs text-gray-500"
                      >
                        {{ t("connection.emptyTables") }}
                      </div>
                      <ul
                        v-else
                        class="space-y-0.5 pl-6 border-l-2 border-green-300 dark:border-green-700"
                      >
                        <li
                          v-for="table in getConnectionState(connection.id)
                            .tables"
                          :key="table.name"
                          class="flex items-center justify-between text-xs text-gray-700 dark:text-gray-300 hover:text-green-700 dark:hover:text-green-300 hover:bg-green-100 dark:hover:bg-green-900/40 cursor-pointer px-2 py-1.5 rounded transition-all duration-150 hover:translate-x-0.5"
                          @dblclick.stop="
                            handleTableDblClick(connection, table)
                          "
                          @contextmenu.prevent.stop="
                            handleTableContextMenu($event, connection, table)
                          "
                        >
                          <el-tooltip
                            :content="table.name"
                            placement="top"
                            :show-after="500"
                          >
                            <span class="truncate font-medium">{{
                              table.name
                            }}</span>
                          </el-tooltip>
                          <span class="text-gray-400 text-[10px]">{{
                            table.rows
                          }}</span>
                        </li>
                      </ul>
                    </template>
                  </div>
                </div>
              </template>
              <template #expanded v-else-if="connection.type === 'mqtt'">
                <div class="mt-2 space-y-2" @dblclick.stop>
                  <div class="space-y-1">
                    <div
                      class="flex items-center justify-between text-xs font-medium text-gray-600 dark:text-gray-300 bg-purple-50 dark:bg-purple-900/20 px-2 py-1 rounded"
                      @dblclick.stop
                    >
                      <div
                        class="flex items-center space-x-2 flex-1 cursor-pointer"
                        @click.stop="
                          toggleSection(connection.id, 'mqttSubscriptions')
                        "
                      >
                        <component
                          :is="
                            getConnectionState(connection.id)
                              .mqttSubscriptionsExpanded
                              ? IconTablerChevronDown
                              : IconTablerChevronRight
                          "
                          class="text-purple-500 w-4 h-4"
                        />
                        <span class="text-purple-700 dark:text-purple-300">{{
                          t("connection.subscriptionSection")
                        }}</span>
                      </div>
                      <div class="flex items-center space-x-1">
                        <el-tooltip
                          :content="t('connection.refreshSubscriptionList')"
                          placement="top"
                        >
                          <button
                            class="p-1 hover:bg-purple-200 dark:hover:bg-purple-700 rounded"
                            @click.stop="
                              refreshMqttSubscriptions(connection.id)
                            "
                          >
                            <IconTablerRefresh
                              class="w-3 h-3 text-purple-600 dark:text-purple-400"
                            />
                          </button>
                        </el-tooltip>
                        <span
                          v-if="
                            !getConnectionState(connection.id)
                              .loadingMqttSubscriptions
                          "
                          class="text-purple-600 dark:text-purple-400"
                        >
                          {{
                            getConnectionState(connection.id).mqttSubscriptions
                              .length
                          }}
                        </span>
                      </div>
                    </div>
                    <div
                      v-if="
                        getConnectionState(connection.id)
                          .loadingMqttSubscriptions
                      "
                      class="pl-6 text-xs text-gray-500"
                    >
                      {{ t("connection.loadingSubscriptions") }}
                    </div>
                    <template
                      v-else-if="
                        getConnectionState(connection.id)
                          .mqttSubscriptionsExpanded
                      "
                    >
                      <div
                        v-if="
                          getConnectionState(connection.id).mqttSubscriptions
                            .length === 0
                        "
                        class="pl-6 text-xs text-gray-500"
                      >
                        {{ t("connection.emptySubscriptions") }}
                      </div>
                      <ul
                        v-else
                        class="space-y-0.5 pl-6 border-l-2 border-purple-300 dark:border-purple-700"
                      >
                        <li
                          v-for="subscription in getConnectionState(
                            connection.id,
                          ).mqttSubscriptions"
                          :key="subscription.id"
                          class="flex items-center justify-between text-xs text-gray-700 dark:text-gray-300 hover:text-purple-700 dark:hover:text-purple-300 hover:bg-purple-100 dark:hover:bg-purple-900/40 cursor-pointer px-2 py-1.5 rounded transition-all duration-150 hover:translate-x-0.5"
                          @dblclick.stop="
                            handleMqttSubscriptionDblClick(
                              connection,
                              subscription,
                            )
                          "
                          @contextmenu.prevent.stop="
                            handleMqttSubscriptionContextMenu(
                              $event,
                              connection,
                              subscription,
                            )
                          "
                        >
                          <el-tooltip
                            :content="subscription.name"
                            placement="top"
                            :show-after="500"
                          >
                            <span class="truncate flex-1 font-medium">{{
                              subscription.name
                            }}</span>
                          </el-tooltip>
                          <span class="text-gray-400 text-[10px] truncate ml-2">
                            {{ subscription.topic }}
                          </span>
                        </li>
                      </ul>
                    </template>
                  </div>
                </div>
              </template>
            </ConnectionItem>
          </div>
        </div>

        <div class="section-block">
          <div class="section-title">{{ t("connection.processingLogic") }}</div>
          <div
            class="section-entry flex items-center gap-2 px-3 py-2 rounded-md cursor-pointer"
            @click="handleOpenCalcUnits"
          >
            <IconTablerCalculator class="w-4 h-4 text-amber-500" />
            <span class="text-sm text-gray-700 dark:text-gray-200">{{
              t("connection.computeUnits")
            }}</span>
          </div>
        </div>

        <div class="section-block">
          <div
            class="section-entry flex items-center gap-2 px-3 py-2 rounded-md cursor-pointer"
            @click="handleOpenAlarmUnits"
          >
            <IconTablerBell class="w-4 h-4 text-rose-500" />
            <span class="text-sm text-gray-700 dark:text-gray-200">{{
              t("connection.alarmUnits")
            }}</span>
          </div>
        </div>

        <!-- 空状态 -->
        <div
          v-if="filteredConnections.length === 0"
          class="text-center py-8 text-gray-500"
        >
          <div class="text-sm">
            {{
              searchText
                ? t("states.noMatchedConnections")
                : t("states.noDataConnections")
            }}
          </div>
          <el-button
            type="primary"
            size="small"
            @click="handleCreate"
            class="mt-2"
          >
            {{ t("actions.createConnection") }}
          </el-button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from "vue";
import IconTablerPlus from "~icons/tabler/plus";
import IconTablerRefresh from "~icons/tabler/refresh";
import IconTablerChevronRight from "~icons/tabler/chevron-right";
import IconTablerChevronDown from "~icons/tabler/chevron-down";
import IconTablerTable from "~icons/tabler/table";
import IconTablerTrash from "~icons/tabler/trash";
import IconTablerDatabase from "~icons/tabler/database";
import IconTablerCalculator from "~icons/tabler/calculator";
import IconTablerBell from "~icons/tabler/bell";
import ConnectionItem from "./ConnectionItem.vue";
import dataAPI from "@/api/data.api";
import { ElMessage, ElMessageBox } from "element-plus";
import { t } from "@/i18n/runtime";
import { getApiErrorMessage } from "@/utils/request";

const props = defineProps({
  connections: {
    type: Array,
    default: () => [],
  },
  selectedConnectionId: {
    type: String,
    default: "",
  },
  projectId: {
    type: String,
    required: true,
  },
});

const emit = defineEmits([
  "select",
  "dblclick",
  "contextmenu",
  "create",
  "refresh",
  "table-dblclick",
  "table-contextmenu",
  "view-table-list",
  "query-dblclick",
  "query-contextmenu",
  "query-deleted",
  "mqtt-subscription-dblclick",
  "mqtt-subscription-view",
  "mqtt-subscription-manage",
  "mqtt-subscription-edit",
  "mqtt-subscription-delete",
  "datapoint-open",
  "calcunit-open",
  "alarmunit-open",
]);

const searchText = ref("");

// 连接状态管理
const connectionStates = reactive({});

const getConnectionState = (connectionId) => {
  if (!connectionStates[connectionId]) {
    connectionStates[connectionId] = {
      expanded: false,
      loading: false,
      tables: [],
      tablesExpanded: true,
      loadingQueries: false,
      queries: [],
      queriesExpanded: true,
      loadingMqttSubscriptions: false,
      mqttSubscriptions: [],
      mqttSubscriptionsExpanded: true,
    };
  }
  return connectionStates[connectionId];
};

const filteredConnections = computed(() => {
  const keyword = searchText.value.trim().toLowerCase();
  if (!keyword) return props.connections;
  return props.connections.filter((connection) => {
    const name = String(connection.name || "").toLowerCase();
    const type = String(connection.type || "").toLowerCase();
    return name.includes(keyword) || type.includes(keyword);
  });
});

const handleSelect = (connection) => {
  emit("select", connection);
};

const handleDblClick = (connection) => {
  emit("dblclick", connection);
};

const handleContextMenu = (event, connection) => {
  emit("contextmenu", event, connection);
};

const handleCreate = () => {
  emit("create");
};

const handleRefresh = () => {
  emit("refresh");
};

/**
 * 打开数据点列表
 */
const handleOpenDataPoints = () => {
  emit("datapoint-open");
};

/**
 * 打开计算单元
 */
const handleOpenCalcUnits = () => {
  emit("calcunit-open");
};

/**
 * 打开报警单元
 */
const handleOpenAlarmUnits = () => {
  emit("alarmunit-open");
};

const handleTableDblClick = (connection, table) => {
  emit("table-dblclick", connection, table);
};

const handleTableContextMenu = (event, connection, table) => {
  emit("table-contextmenu", event, connection, table);
};

const handleViewTableList = (connection) => {
  emit("view-table-list", connection);
};

const toggleSection = async (connectionId, section) => {
  const state = getConnectionState(connectionId);

  if (section === "tables") {
    state.tablesExpanded = !state.tablesExpanded;

    // 如果展开且还没加载表列表，则加载
    if (state.tablesExpanded && state.tables.length === 0 && !state.loading) {
      await loadTables(connectionId);
    }
  } else if (section === "queries") {
    state.queriesExpanded = !state.queriesExpanded;

    // 如果展开且还没加载查询列表，则加载
    if (
      state.queriesExpanded &&
      state.queries.length === 0 &&
      !state.loadingQueries
    ) {
      await loadQueries(connectionId);
    }
  } else if (section === "mqttSubscriptions") {
    state.mqttSubscriptionsExpanded = !state.mqttSubscriptionsExpanded;

    if (
      state.mqttSubscriptionsExpanded &&
      state.mqttSubscriptions.length === 0 &&
      !state.loadingMqttSubscriptions
    ) {
      await loadMqttSubscriptions(connectionId);
    }
  }
};

const loadTables = async (connectionId) => {
  const state = getConnectionState(connectionId);
  state.loading = true;

  try {
    const response = await dataAPI.getConnectionTables(
      props.projectId,
      connectionId,
    );
    state.tables = response.data?.tables || [];
  } catch (error) {
    ElMessage.error(
      `${t("connection.loadingTables")} ${getApiErrorMessage(error, "加载表列表失败")}`,
    );
  } finally {
    state.loading = false;
  }
};

const loadQueries = async (connectionId) => {
  const state = getConnectionState(connectionId);
  state.loadingQueries = true;

  try {
    const response = await dataAPI.getQueries(props.projectId, {
      connectionId,
    });
    state.queries = response.data?.queries || [];
  } catch (error) {
    ElMessage.error(
      `${t("connection.loadingQueries")} ${getApiErrorMessage(error, "加载查询失败")}`,
    );
  } finally {
    state.loadingQueries = false;
  }
};

/**
 * 加载 MQTT 订阅列表
 * @param {string} connectionId - 连接ID
 * @returns {Promise<void>}
 */
const loadMqttSubscriptions = async (connectionId) => {
  const state = getConnectionState(connectionId);
  state.loadingMqttSubscriptions = true;

  try {
    const response = await dataAPI.getMqttSubscriptions(
      props.projectId,
      connectionId,
    );
    state.mqttSubscriptions = response.data || [];
  } catch (error) {
    ElMessage.error(
      t("subscription.loadFailed", {
        message: getApiErrorMessage(error, "加载订阅失败"),
      }),
    );
  } finally {
    state.loadingMqttSubscriptions = false;
  }
};

const handleQueryDblClick = (connection, query) => {
  emit("query-dblclick", connection, query);
};

const handleQueryContextMenu = (event, connection, query) => {
  emit("query-contextmenu", event, connection, query);
};

/**
 * 双击 MQTT 订阅
 * @param {object} connection - 连接信息
 * @param {object} subscription - 订阅信息
 * @returns {void}
 */
const handleMqttSubscriptionDblClick = (connection, subscription) => {
  emit("mqtt-subscription-dblclick", connection, subscription);
};

/**
 * MQTT 订阅右键菜单
 * @param {MouseEvent} event - 鼠标事件
 * @param {object} connection - 连接信息
 * @param {object} subscription - 订阅信息
 * @returns {void}
 */
const handleMqttSubscriptionContextMenu = (event, connection, subscription) => {
  const menu = document.createElement("div");
  menu.className =
    "fixed bg-white dark:bg-gray-800 rounded-lg shadow-lg border border-gray-200 dark:border-gray-700 py-1 min-w-[140px]";
  menu.style.cssText = `position: fixed; left: ${event.clientX}px; top: ${event.clientY}px; z-index: 9999;`;

  const viewItem = document.createElement("div");
  viewItem.className =
    "px-4 py-2 text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 cursor-pointer";
  viewItem.textContent = t("actions.viewMessages");
  viewItem.onclick = () => {
    emit("mqtt-subscription-view", connection, subscription);
    document.body.removeChild(menu);
  };

  const manageItem = document.createElement("div");
  manageItem.className =
    "px-4 py-2 text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 cursor-pointer";
  manageItem.textContent = t("actions.manageVariables");
  manageItem.onclick = () => {
    emit("mqtt-subscription-manage", connection, subscription);
    document.body.removeChild(menu);
  };

  const editItem = document.createElement("div");
  editItem.className =
    "px-4 py-2 text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 cursor-pointer";
  editItem.textContent = t("subscription.edit");
  editItem.onclick = () => {
    emit("mqtt-subscription-edit", connection, subscription);
    document.body.removeChild(menu);
  };

  const deleteItem = document.createElement("div");
  deleteItem.className =
    "px-4 py-2 text-sm text-red-600 hover:bg-red-50 dark:hover:bg-red-900/30 cursor-pointer";
  deleteItem.textContent = t("actions.delete");
  deleteItem.onclick = () => {
    emit("mqtt-subscription-delete", connection, subscription);
    document.body.removeChild(menu);
  };

  menu.appendChild(viewItem);
  menu.appendChild(manageItem);
  menu.appendChild(editItem);
  menu.appendChild(deleteItem);
  document.body.appendChild(menu);

  const closeMenu = (e) => {
    if (!menu.contains(e.target)) {
      if (document.body.contains(menu)) {
        document.body.removeChild(menu);
      }
      document.removeEventListener("click", closeMenu);
    }
  };
  setTimeout(() => {
    document.addEventListener("click", closeMenu);
  }, 0);
};

const handleDeleteQuery = async (connection, query) => {
  try {
    await ElMessageBox.confirm(
      t("query.deleteQueryConfirm", { name: query.name }),
      t("query.deleteConnectionTitle"),
      {
        confirmButtonText: t("actions.delete"),
        cancelButtonText: t("actions.cancel"),
        type: "warning",
      },
    );

    await dataAPI.deleteQuery(query.id);
    ElMessage.success(t("query.deleted"));

    // 从列表中移除
    const state = getConnectionState(connection.id);
    const index = state.queries.findIndex((q) => q.id === query.id);
    if (index > -1) {
      state.queries.splice(index, 1);
    }

    emit("query-deleted", query);
  } catch (error) {
    if (error !== "cancel") {
      ElMessage.error(
        t("query.deleteQueryFailed", {
          message: getApiErrorMessage(error, "删除查询失败"),
        }),
      );
    }
  }
};

/**
 * 刷新表列表
 */
const refreshTables = async (connectionId) => {
  await loadTables(connectionId);
  ElMessage.success(t("connection.refreshTableList"));
};

/**
 * 刷新查询列表
 */
const refreshQueries = async (connectionId) => {
  await loadQueries(connectionId);
  ElMessage.success(t("connection.refreshQueryList"));
};

/**
 * 刷新订阅列表
 * @param {string} connectionId - 连接ID
 * @returns {Promise<void>}
 */
const refreshMqttSubscriptions = async (connectionId) => {
  await loadMqttSubscriptions(connectionId);
  ElMessage.success(t("connection.refreshSubscriptionList"));
};

/**
 * 查询列表标题栏右键菜单
 */
const handleQueriesSectionContextMenu = (event, connection) => {
  // 创建简单的右键菜单
  const menu = document.createElement("div");
  menu.className =
    "fixed bg-white dark:bg-gray-800 rounded-lg shadow-lg border border-gray-200 dark:border-gray-700 py-1 min-w-[150px]";
  menu.style.cssText = `position: fixed; left: ${event.clientX}px; top: ${event.clientY}px; z-index: 9999;`;

  const refreshItem = document.createElement("div");
  refreshItem.className =
    "px-4 py-2 text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 cursor-pointer flex items-center";
  refreshItem.innerHTML = `<span class="mr-2">🔄</span>${t("connection.refreshQueryList")}`;
  refreshItem.onclick = () => {
    refreshQueries(connection.id);
    document.body.removeChild(menu);
  };

  menu.appendChild(refreshItem);
  document.body.appendChild(menu);

  // 点击外部关闭菜单
  const closeMenu = (e) => {
    if (!menu.contains(e.target)) {
      if (document.body.contains(menu)) {
        document.body.removeChild(menu);
      }
      document.removeEventListener("click", closeMenu);
    }
  };
  setTimeout(() => {
    document.addEventListener("click", closeMenu);
  }, 0);
};

/**
 * 表列表标题栏右键菜单
 */
const handleTablesSectionContextMenu = (event, connection) => {
  // 创建简单的右键菜单
  const menu = document.createElement("div");
  menu.className =
    "fixed bg-white dark:bg-gray-800 rounded-lg shadow-lg border border-gray-200 dark:border-gray-700 py-1 min-w-[150px]";
  menu.style.cssText = `position: fixed; left: ${event.clientX}px; top: ${event.clientY}px; z-index: 9999;`;

  const refreshItem = document.createElement("div");
  refreshItem.className =
    "px-4 py-2 text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 cursor-pointer flex items-center";
  refreshItem.innerHTML = `<span class="mr-2">🔄</span>${t("connection.refreshTableList")}`;
  refreshItem.onclick = () => {
    refreshTables(connection.id);
    document.body.removeChild(menu);
  };

  const viewItem = document.createElement("div");
  viewItem.className =
    "px-4 py-2 text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 cursor-pointer flex items-center";
  viewItem.innerHTML = `<span class="mr-2">📋</span>${t("connection.viewTableList")}`;
  viewItem.onclick = () => {
    handleViewTableList(connection);
    document.body.removeChild(menu);
  };

  menu.appendChild(refreshItem);
  menu.appendChild(viewItem);
  document.body.appendChild(menu);

  // 点击外部关闭菜单
  const closeMenu = (e) => {
    if (!menu.contains(e.target)) {
      if (document.body.contains(menu)) {
        document.body.removeChild(menu);
      }
      document.removeEventListener("click", closeMenu);
    }
  };
  setTimeout(() => {
    document.addEventListener("click", closeMenu);
  }, 0);
};

// 暴露方法给父组件
defineExpose({
  getConnectionState,
  loadTables,
  loadQueries,
  loadMqttSubscriptions,
});
</script>

<style scoped>
.connection-search :deep(.el-input__wrapper) {
  border-radius: 8px;
  box-shadow: none;
  background: #f8fafc;
  border: 1px solid #e5e7eb;
}

.dark .connection-search :deep(.el-input__wrapper) {
  background: #0f172a;
  border-color: #1f2937;
}

.connection-search :deep(.el-input__wrapper.is-focus) {
  box-shadow: 0 0 0 1px rgba(59, 130, 246, 0.35);
  border-color: #93c5fd;
}

.header-btn {
  width: 28px;
  height: 28px;
  padding: 0;
}

.icon-btn {
  border-color: transparent;
  color: #6b7280;
  background: transparent;
}

.icon-btn:hover {
  color: #2563eb;
  background: rgba(243, 244, 246, 0.9);
}

.section-block {
  padding: 6px 2px 10px;
}

.section-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 6px 6px;
}

.section-title {
  font-size: 12px;
  color: #6b7280;
}

.section-actions {
  display: flex;
  align-items: center;
  gap: 4px;
}

.section-tools {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 0 6px 10px;
}

.section-entry {
  background: #eef2ff;
}

.section-entry:hover {
  background: #e0e7ff;
}

.dark .section-title {
  color: #6b7280;
}

.dark .section-entry {
  background: rgba(79, 70, 229, 0.18);
}

.dark .section-entry:hover {
  background: rgba(79, 70, 229, 0.28);
}

.section-entry .text-amber-500 {
  color: #f59e0b;
}

.section-entry .text-rose-500 {
  color: #f43f5e;
}

.connection-items {
  padding-left: 8px;
}
</style>
