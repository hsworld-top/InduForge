<template>
  <div class="datapoint-list h-full flex flex-col">
    <div
      class="toolbar flex items-center justify-between px-4 py-3 border-b border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800"
    >
      <div class="text-base font-semibold text-gray-800 dark:text-gray-100">
        数据点管理
      </div>
      <div class="flex items-center gap-2">
        <el-input
          v-model="searchText"
          size="small"
          clearable
          placeholder="搜索路径或名称"
          class="w-52"
        />
        <el-select v-model="typeFilter" size="small" placeholder="类型" class="w-36">
          <el-option label="全部类型" value="" />
          <el-option label="数据库查询" value="db.query" />
          <el-option label="MQTT 变量" value="mqtt.tag" />
          <el-option label="MQTT 订阅" value="mqtt.subscription" />
          <el-option label="计算输出" value="calc.output" />
        </el-select>
        <el-select v-model="statusFilter" size="small" placeholder="状态" class="w-28">
          <el-option label="全部状态" value="" />
          <el-option label="活跃" value="active" />
          <el-option label="失效" value="invalid" />
        </el-select>
        <el-button size="small" @click="handleRefresh">刷新</el-button>
      </div>
    </div>

    <div class="hint px-4 py-2 text-xs text-gray-500 bg-gray-50 dark:bg-gray-900">
      数据点由查询、变量、计算单元自动生成，可直接在设计器中使用。
    </div>

    <div class="flex-1 overflow-y-auto p-4">
      <div v-if="loading" class="py-8 text-center text-gray-400">
        加载中...
      </div>

      <div v-else-if="groupedDataPoints.length === 0" class="py-8">
        <el-empty description="暂无数据点" />
      </div>

      <div v-else class="space-y-6">
        <div v-for="group in groupedDataPoints" :key="group.key">
          <div class="group-header flex items-center justify-between">
            <div class="flex items-center gap-2">
              <span class="group-title">{{ group.label }}</span>
              <span class="group-count">{{ group.items.length }}</span>
            </div>
          </div>
          <div v-if="group.allowClean" class="group-actions">
            <div class="text-xs text-gray-500">
              已选择 {{ invalidSelection.length }} 个失效数据点
            </div>
            <el-button
              size="small"
              type="danger"
              :disabled="invalidSelection.length === 0"
              @click="handleBatchDelete"
            >
              批量清理
            </el-button>
          </div>
          <el-table
            :data="group.items"
            size="small"
            stripe
            row-key="id"
            class="group-table"
            @selection-change="
              group.allowClean ? handleInvalidSelection($event) : null
            "
          >
            <el-table-column
              v-if="group.allowClean"
              type="selection"
              width="42"
            />
            <el-table-column label="路径" min-width="280">
              <template #default="{ row }">
                <span class="text-xs text-gray-600 dark:text-gray-300 truncate">
                  {{ row.path }}
                </span>
              </template>
            </el-table-column>
            <el-table-column prop="name" label="名称" min-width="140" />
            <el-table-column prop="dataType" label="数据类型" width="100" />
            <el-table-column label="状态" width="90">
              <template #default="{ row }">
                <el-tag
                  size="small"
                  :type="row.status === 'invalid' ? 'info' : 'success'"
                >
                  {{ row.status === "invalid" ? "失效" : "活跃" }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="更新时间" width="160">
              <template #default="{ row }">
                <span class="text-xs text-gray-500 dark:text-gray-400">
                  {{ formatTime(getUpdatedAt(row)) }}
                </span>
              </template>
            </el-table-column>
            <el-table-column v-if="group.allowClean" label="操作" width="100">
              <template #default="{ row }">
                <el-button
                  link
                  size="small"
                  class="text-red-500"
                  @click="handleDelete(row)"
                >
                  清理
                </el-button>
              </template>
            </el-table-column>
          </el-table>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue";
import { ElMessage, ElMessageBox } from "element-plus";
import dayjs from "dayjs";
import { TIME_FORMAT } from "@/constants";
import dataAPI from "@/api/data.api";

const props = defineProps({
  projectId: {
    type: String,
    required: true,
  },
});

const loading = ref(false);
const datapoints = ref([]);
const searchText = ref("");
const typeFilter = ref("");
const statusFilter = ref("");
const debounceTimer = ref(null);
const invalidSelection = ref([]);

const typeLabels = {
  "db.query": "数据库查询",
  "mqtt.tag": "MQTT 变量",
  "mqtt.subscription": "MQTT 订阅",
  "calc.output": "计算输出",
  "static.var": "静态变量",
};

/**
 * 加载数据点列表
 * @returns {Promise<void>}
 */
const loadDataPoints = async () => {
  if (!props.projectId) return;
  loading.value = true;
  try {
    const response = await dataAPI.getDataPoints(props.projectId, {
      page: 1,
      pageSize: 200,
      type: typeFilter.value || undefined,
      status: statusFilter.value || undefined,
      search: searchText.value || undefined,
    });
    if (response.success) {
      datapoints.value = response.data?.datapoints || [];
    }
  } catch (error) {
    ElMessage.error(
      "加载数据点失败：" + (error.response?.data?.message || error.message),
    );
  } finally {
    loading.value = false;
  }
};

/**
 * 刷新列表
 * @returns {Promise<void>}
 */
const handleRefresh = async () => {
  await loadDataPoints();
  ElMessage.success("数据点列表已刷新");
};

/**
 * 删除失效数据点
 * @param {object} datapoint - 数据点对象
 * @returns {Promise<void>}
 */
const handleDelete = async (datapoint) => {
  try {
    await ElMessageBox.confirm(
      `确定要清理数据点 "${datapoint.name}" 吗？`,
      "清理确认",
      {
        confirmButtonText: "确定",
        cancelButtonText: "取消",
        type: "warning",
      },
    );
    await dataAPI.deleteDataPoint(props.projectId, datapoint.id);
    ElMessage.success("数据点已清理");
    await loadDataPoints();
  } catch (error) {
    if (error !== "cancel") {
      ElMessage.error(
        "清理失败：" + (error.response?.data?.message || error.message),
      );
    }
  }
};

/**
 * 处理失效数据点选择
 * @param {object[]} selection - 选中项
 */
const handleInvalidSelection = (selection) => {
  invalidSelection.value = selection || [];
};

/**
 * 批量清理失效数据点
 */
const handleBatchDelete = async () => {
  if (invalidSelection.value.length === 0) return;
  try {
    await ElMessageBox.confirm(
      `确定要清理 ${invalidSelection.value.length} 个失效数据点吗？`,
      "批量清理确认",
      {
        confirmButtonText: "确定",
        cancelButtonText: "取消",
        type: "warning",
      },
    );
    const response = await dataAPI.deleteDataPointsBatch(
      props.projectId,
      invalidSelection.value.map((item) => item.id),
    );
    const deletedCount = response?.data?.deletedCount ?? 0;
    ElMessage.success(`已清理 ${deletedCount} 个失效数据点`);
    invalidSelection.value = [];
    await loadDataPoints();
  } catch (error) {
    if (error !== "cancel") {
      ElMessage.error(
        "批量清理失败：" + (error.response?.data?.message || error.message),
      );
    }
  }
};

/**
 * 时间格式化
 * @param {string} value - 时间
 * @returns {string}
 */
const formatTime = (value) => {
  if (!value) return "-";
  return dayjs(value).format(TIME_FORMAT);
};

/**
 * 获取更新时间
 * @param {object} row - 数据点
 * @returns {string}
 */
const getUpdatedAt = (row) => {
  return (
    row?.updatedAt ||
    row?.updated_at ||
    row?.createdAt ||
    row?.created_at ||
    ""
  );
};

const groupedDataPoints = computed(() => {
  const items = datapoints.value || [];
  const invalidItems = items.filter((item) => item.status === "invalid");
  const activeItems = items.filter((item) => item.status !== "invalid");

  const groups = new Map();

  activeItems.forEach((item) => {
    const key = item.sourceType || "other";
    if (!groups.has(key)) {
      groups.set(key, []);
    }
    groups.get(key).push(item);
  });

  const result = Array.from(groups.entries()).map(([key, list]) => ({
    key,
    label: typeLabels[key] || key,
    items: list,
    allowClean: false,
  }));

  if (statusFilter.value !== "active") {
    if (invalidItems.length > 0) {
      result.push({
        key: "invalid",
        label: "已失效",
        items: invalidItems,
        allowClean: true,
      });
    }
  }

  return result;
});

watch([searchText, typeFilter, statusFilter], () => {
  if (debounceTimer.value) {
    clearTimeout(debounceTimer.value);
  }
  debounceTimer.value = setTimeout(() => {
    loadDataPoints();
  }, 300);
});

onMounted(() => {
  loadDataPoints();
});

onBeforeUnmount(() => {
  if (debounceTimer.value) {
    clearTimeout(debounceTimer.value);
  }
});

defineExpose({
  refresh: loadDataPoints,
});
</script>

<style scoped>
.group-header {
  padding: 6px 4px;
}

.group-actions {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 6px 4px 4px;
}

.group-title {
  font-size: 13px;
  font-weight: 600;
  color: #111827;
}

.group-count {
  padding: 0 6px;
  border-radius: 999px;
  background: #f3f4f6;
  color: #6b7280;
  font-size: 11px;
}

.dark .group-title {
  color: #e5e7eb;
}

.dark .group-count {
  background: #1f2937;
  color: #9ca3af;
}
</style>
