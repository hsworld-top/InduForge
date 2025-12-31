<template>
  <div
    class="connection-list w-64 bg-white dark:bg-gray-800 border-r border-gray-200 dark:border-gray-700 flex flex-col overflow-hidden"
  >
    <div class="p-4 flex-1 overflow-y-auto min-h-0">
      <div class="space-y-2">
        <!-- 标题和操作按钮 -->
        <div class="flex items-center justify-between mb-3">
          <h3 class="text-sm font-medium text-gray-900 dark:text-white">
            数据连接
          </h3>
          <div class="flex items-center space-x-2">
            <el-button type="primary" size="small" @click="handleCreate">
              <IconTablerPlus class="mr-1 w-4 h-4" />
              新建
            </el-button>
            <el-button
              size="small"
              circle
              @click="handleRefresh"
              class="!border-gray-300 dark:!border-gray-600 hover:!bg-gray-50 dark:hover:!bg-gray-700"
            >
              <IconTablerRefresh class="w-4 h-4" />
            </el-button>
          </div>
        </div>

        <!-- 连接列表 -->
        <div class="connection-items">
          <ConnectionItem
            v-for="connection in connections"
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
                      <span class="text-blue-700 dark:text-blue-300">查询</span>
                    </div>
                    <div class="flex items-center space-x-1">
                      <el-tooltip content="刷新查询列表" placement="top">
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
                        v-if="!getConnectionState(connection.id).loadingQueries"
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
                    加载查询列表...
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
                      暂无查询
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
                        @dblclick.stop="handleQueryDblClick(connection, query)"
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
                        <el-tooltip content="删除查询" placement="top">
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
                      <span class="text-green-700 dark:text-green-300">表</span>
                    </div>
                    <div class="flex items-center space-x-1">
                      <el-tooltip content="刷新表列表" placement="top">
                        <button
                          class="p-1 hover:bg-green-200 dark:hover:bg-green-700 rounded"
                          @click.stop="refreshTables(connection.id)"
                        >
                          <IconTablerRefresh
                            class="w-3 h-3 text-green-600 dark:text-green-400"
                          />
                        </button>
                      </el-tooltip>
                      <el-tooltip content="查看表列表" placement="top">
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
                    加载表列表...
                  </div>
                  <template
                    v-else-if="getConnectionState(connection.id).tablesExpanded"
                  >
                    <div
                      v-if="
                        getConnectionState(connection.id).tables.length === 0
                      "
                      class="pl-6 text-xs text-gray-500"
                    >
                      暂无表
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
                        @dblclick.stop="handleTableDblClick(connection, table)"
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
          </ConnectionItem>
        </div>

        <!-- 空状态 -->
        <div
          v-if="connections.length === 0"
          class="text-center py-8 text-gray-500"
        >
          <div class="text-sm">暂无数据连接</div>
          <el-button
            type="primary"
            size="small"
            @click="handleCreate"
            class="mt-2"
          >
            创建连接
          </el-button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { reactive } from "vue";
import IconTablerPlus from "~icons/tabler/plus";
import IconTablerRefresh from "~icons/tabler/refresh";
import IconTablerChevronRight from "~icons/tabler/chevron-right";
import IconTablerChevronDown from "~icons/tabler/chevron-down";
import IconTablerTable from "~icons/tabler/table";
import IconTablerTrash from "~icons/tabler/trash";
import ConnectionItem from "./ConnectionItem.vue";
import dataAPI from "@/api/data.api";
import { ElMessage, ElMessageBox } from "element-plus";

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
]);

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
    };
  }
  return connectionStates[connectionId];
};

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
    if (response.success) {
      state.tables = response.data.tables || [];
    }
  } catch (error) {
    ElMessage.error(
      "加载表列表失败：" + (error.response?.data?.message || error.message),
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
    if (response.success) {
      state.queries = response.data.queries || [];
    }
  } catch (error) {
    ElMessage.error(
      "加载查询列表失败：" + (error.response?.data?.message || error.message),
    );
  } finally {
    state.loadingQueries = false;
  }
};

const handleQueryDblClick = (connection, query) => {
  emit("query-dblclick", connection, query);
};

const handleQueryContextMenu = (event, connection, query) => {
  emit("query-contextmenu", event, connection, query);
};

const handleDeleteQuery = async (connection, query) => {
  try {
    await ElMessageBox.confirm(
      `确定要删除查询 "${query.name}" 吗？`,
      "删除确认",
      {
        confirmButtonText: "确定",
        cancelButtonText: "取消",
        type: "warning",
      },
    );

    await dataAPI.deleteQuery(query.id);
    ElMessage.success("查询已删除");

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
        "删除查询失败：" + (error.response?.data?.message || error.message),
      );
    }
  }
};

/**
 * 刷新表列表
 */
const refreshTables = async (connectionId) => {
  await loadTables(connectionId);
  ElMessage.success("表列表已刷新");
};

/**
 * 刷新查询列表
 */
const refreshQueries = async (connectionId) => {
  await loadQueries(connectionId);
  ElMessage.success("查询列表已刷新");
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
  refreshItem.innerHTML = '<span class="mr-2">🔄</span>刷新查询列表';
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
  refreshItem.innerHTML = '<span class="mr-2">🔄</span>刷新表列表';
  refreshItem.onclick = () => {
    refreshTables(connection.id);
    document.body.removeChild(menu);
  };

  const viewItem = document.createElement("div");
  viewItem.className =
    "px-4 py-2 text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 cursor-pointer flex items-center";
  viewItem.innerHTML = '<span class="mr-2">📋</span>查看表列表';
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
});
</script>
