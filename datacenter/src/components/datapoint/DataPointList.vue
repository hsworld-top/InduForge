<template>
  <div class="datapoint-list h-full flex flex-col">
    <div
      v-if="showToolbar"
      class="toolbar flex items-center justify-between px-4 py-3 border-b border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800"
    >
      <div class="text-base font-semibold text-gray-800 dark:text-gray-100">
        {{ t("datapoints.title") }}
      </div>
      <div class="flex items-center gap-2">
        <el-input
          v-model="searchText"
          size="small"
          clearable
          :placeholder="t('datapoints.searchPlaceholder')"
          class="w-52"
        />
        <el-select
          v-model="typeFilter"
          size="small"
          :placeholder="t('datapoints.typePlaceholder')"
          class="w-36"
        >
          <el-option :label="t('datapoints.allTypes')" value="" />
          <el-option :label="t('datapoints.types.db.query')" value="db.query" />
          <el-option :label="t('datapoints.types.mqtt.tag')" value="mqtt.tag" />
          <el-option
            :label="t('datapoints.types.mqtt.subscription')"
            value="mqtt.subscription"
          />
          <el-option
            :label="t('datapoints.types.calc.output')"
            value="calc.output"
          />
        </el-select>
        <el-select
          v-model="statusFilter"
          size="small"
          :placeholder="t('datapoints.statusPlaceholder')"
          class="w-28"
        >
          <el-option :label="t('datapoints.allStatuses')" value="" />
          <el-option :label="t('common.active')" value="active" />
          <el-option :label="t('common.invalid')" value="invalid" />
        </el-select>
        <el-button size="small" @click="handleRefresh">{{
          t("actions.refresh")
        }}</el-button>
      </div>
    </div>

    <div
      class="hint px-4 py-2 text-xs text-gray-500 bg-gray-50 dark:bg-gray-900"
    >
      {{ t("datapoints.hint") }}
    </div>

    <div class="flex-1 overflow-y-auto p-4">
      <div v-if="loading" class="py-8 text-center text-gray-400">
        {{ t("common.loading") }}
      </div>

      <div v-else-if="groupedDataPoints.length === 0" class="py-8">
        <el-empty :description="t('datapoints.empty')" />
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
              {{
                t("datapoints.selectedInvalid", {
                  count: invalidSelection.length,
                })
              }}
            </div>
            <el-button
              size="small"
              type="danger"
              :disabled="invalidSelection.length === 0"
              @click="handleBatchDelete"
            >
              {{ t("actions.batchClean") }}
            </el-button>
          </div>
          <el-table
            :data="group.items"
            size="small"
            stripe
            row-key="id"
            class="group-table"
            @row-click="(row) => emit('select', row)"
            @selection-change="
              group.allowClean ? handleInvalidSelection($event) : null
            "
          >
            <el-table-column
              v-if="group.allowClean"
              type="selection"
              width="42"
            />
            <el-table-column :label="t('datapoints.path')" min-width="280">
              <template #default="{ row }">
                <span class="text-xs text-gray-600 dark:text-gray-300 truncate">
                  {{ row.path }}
                </span>
              </template>
            </el-table-column>
            <el-table-column
              prop="name"
              :label="t('datapoints.name')"
              min-width="140"
            />
            <el-table-column
              prop="dataType"
              :label="t('datapoints.dataType')"
              width="100"
            />
            <el-table-column :label="t('datapoints.status')" width="90">
              <template #default="{ row }">
                <el-tag
                  size="small"
                  :type="row.status === 'invalid' ? 'info' : 'success'"
                >
                  {{
                    row.status === "invalid"
                      ? t("common.invalid")
                      : t("common.active")
                  }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column
              :label="t('datapoints.runtimePermission')"
              min-width="200"
            >
              <template #default="{ row }">
                <el-button
                  link
                  size="small"
                  class="permission-entry"
                  @click.stop="openPermissionDialog(row)"
                >
                  {{ summarizeRuntimeGrant(getWriteRuntimeGrant(row)) }}
                </el-button>
              </template>
            </el-table-column>
            <el-table-column :label="t('datapoints.updatedAt')" width="160">
              <template #default="{ row }">
                <span class="text-xs text-gray-500 dark:text-gray-400">
                  {{ formatTime(getUpdatedAt(row)) }}
                </span>
              </template>
            </el-table-column>
            <el-table-column
              v-if="group.allowClean"
              :label="t('subscription.operations')"
              width="100"
            >
              <template #default="{ row }">
                <el-button
                  link
                  size="small"
                  class="text-red-500"
                  @click="handleDelete(row)"
                >
                  {{ t("actions.clean") }}
                </el-button>
              </template>
            </el-table-column>
          </el-table>
        </div>
      </div>
    </div>

    <el-dialog
      v-model="permissionDialogVisible"
      :title="
        t('datapoints.runtimePermissionDialogTitle', {
          name: currentPermissionDatapoint?.name || '-',
        })
      "
      width="520px"
      destroy-on-close
    >
      <div class="permission-dialog">
        <div class="permission-dialog__hint">
          {{ t("datapoints.runtimePermissionHint") }}
        </div>

        <el-form label-position="top">
          <el-form-item :label="t('datapoints.runtimePermissionSummaryLabel')">
            <el-tag type="info">
              {{ summarizeRuntimeGrant(draftWriteRuntimeGrant) }}
            </el-tag>
          </el-form-item>

          <el-form-item :label="t('datapoints.runtimePermissionInherit')">
            <el-switch v-model="permissionForm.inherit" />
          </el-form-item>

          <el-form-item :label="t('datapoints.runtimePermissionAllowRoles')">
            <el-input
              v-model="allowRolesInput"
              type="textarea"
              :rows="4"
              :placeholder="t('datapoints.runtimePermissionRolesPlaceholder')"
            />
          </el-form-item>

          <el-form-item :label="t('datapoints.runtimePermissionDenyRoles')">
            <el-input
              v-model="denyRolesInput"
              type="textarea"
              :rows="4"
              :placeholder="t('datapoints.runtimePermissionRolesPlaceholder')"
            />
          </el-form-item>
        </el-form>
      </div>

      <template #footer>
        <div class="dialog-footer">
          <el-button @click="permissionDialogVisible = false">
            {{ t("actions.cancel") }}
          </el-button>
          <el-button
            type="primary"
            :loading="permissionSaving"
            @click="handlePermissionSave"
          >
            {{ t("actions.save") }}
          </el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue";
import { ElMessage, ElMessageBox } from "element-plus";
import dayjs from "dayjs";
import { TIME_FORMAT } from "@/constants";
import dataAPI from "@/api/data.api";
import { t } from "@/i18n/runtime";
import {
  normalizeRuntimeGrantPayload,
  summarizeRuntimeGrant,
} from "@/utils/runtime-permission-grants";
import { getApiErrorMessage } from "@/utils/request";

const props = defineProps({
  projectId: {
    type: String,
    required: true,
  },
  sourceType: {
    type: String,
    default: undefined,
  },
  status: {
    type: String,
    default: undefined,
  },
  search: {
    type: String,
    default: undefined,
  },
  showToolbar: {
    type: Boolean,
    default: true,
  },
});

const emit = defineEmits(["select"]);

const loading = ref(false);
const datapoints = ref([]);
const searchText = ref("");
const typeFilter = ref("");
const statusFilter = ref("");
const debounceTimer = ref(null);
const invalidSelection = ref([]);
const permissionDialogVisible = ref(false);
const permissionSaving = ref(false);
const currentPermissionDatapoint = ref(null);
const permissionForm = ref(normalizeRuntimeGrantPayload());
const allowRolesInput = ref("");
const denyRolesInput = ref("");

const typeLabels = {
  "db.query": t("datapoints.types.db.query"),
  "mqtt.tag": t("datapoints.types.mqtt.tag"),
  "mqtt.subscription": t("datapoints.types.mqtt.subscription"),
  "calc.output": t("datapoints.types.calc.output"),
  "static.var": t("datapoints.types.static.var"),
};

/**
 * 外部工作区和列表内置工具栏共用同一个请求构造逻辑。
 * 仅把有值筛选写入 params，避免向旧后端传入 undefined 或空数组。
 */
const buildDataPointQueryParams = () => {
  const selectedSourceType = props.sourceType ?? typeFilter.value;
  const selectedStatus = props.status ?? statusFilter.value;
  const selectedSearch = props.search ?? searchText.value;
  const params: Record<string, unknown> = {
    page: 1,
    pageSize: 200,
  };

  if (selectedSourceType) {
    params.type = selectedSourceType;
  }
  if (selectedStatus) {
    params.status = selectedStatus;
  }
  if (selectedSearch) {
    params.search = selectedSearch;
  }

  return params;
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
      ...buildDataPointQueryParams(),
    });
    datapoints.value = response.data?.datapoints || [];
  } catch (error) {
    ElMessage.error(
      t("datapoints.loadFailed", {
        message: getApiErrorMessage(error, "加载数据点失败"),
      }),
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
  ElMessage.success(t("datapoints.refreshSuccess"));
};

/**
 * 删除失效数据点
 * @param {object} datapoint - 数据点对象
 * @returns {Promise<void>}
 */
const handleDelete = async (datapoint) => {
  try {
    await ElMessageBox.confirm(
      t("datapoints.deleteConfirm", { name: datapoint.name }),
      t("datapoints.deleteConfirmTitle"),
      {
        confirmButtonText: t("actions.clean"),
        cancelButtonText: t("actions.cancel"),
        type: "warning",
      },
    );
    await dataAPI.deleteDataPoint(props.projectId, datapoint.id);
    ElMessage.success(t("datapoints.deleteSuccess"));
    await loadDataPoints();
  } catch (error) {
    if (error !== "cancel") {
      ElMessage.error(
        t("datapoints.deleteFailed", {
          message: getApiErrorMessage(error, "删除数据点失败"),
        }),
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
      t("datapoints.batchDeleteConfirm", {
        count: invalidSelection.value.length,
      }),
      t("datapoints.batchDeleteConfirmTitle"),
      {
        confirmButtonText: t("actions.batchClean"),
        cancelButtonText: t("actions.cancel"),
        type: "warning",
      },
    );
    const response = await dataAPI.deleteDataPointsBatch(
      props.projectId,
      invalidSelection.value.map((item) => item.id),
    );
    const deletedCount = response?.data?.deletedCount ?? 0;
    ElMessage.success(
      t("datapoints.batchDeleteSuccess", { count: deletedCount }),
    );
    invalidSelection.value = [];
    await loadDataPoints();
  } catch (error) {
    if (error !== "cancel") {
      ElMessage.error(
        t("datapoints.batchDeleteFailed", {
          message: getApiErrorMessage(error, "批量删除数据点失败"),
        }),
      );
    }
  }
};

/**
 * 从文本框解析角色列表。
 * 这里故意只接受换行和中英文逗号分隔，避免在轻量弹窗里引入复杂选择器。
 * @param {string} value - 原始输入
 * @returns {string[]}
 */
const parseRoleInput = (value) => {
  return String(value || "")
    .split(/[\n,，]/)
    .map((item) => item.trim())
    .filter(Boolean);
};

/**
 * 兼容不同后端返回字段，统一提取数据点 write 权限。
 * datacenter 侧只负责前端编辑模型兼容，不在这里推导真实继承结果。
 * @param {object} row - 数据点对象
 * @returns {{allowRoles: string[], denyRoles: string[], inherit: boolean}}
 */
const getWriteRuntimeGrant = (row) => {
  return normalizeRuntimeGrantPayload(
    row?.runtimePermissions?.write ||
      row?.runtimePermissionGrants?.write ||
      row?.writePermission ||
      {},
  );
};

/**
 * 计算弹窗当前输入对应的待保存权限。
 * 摘要和最终提交都基于这份归一化结果，避免界面展示与实际 payload 不一致。
 */
const draftWriteRuntimeGrant = computed(() =>
  normalizeRuntimeGrantPayload({
    allowRoles: parseRoleInput(allowRolesInput.value),
    denyRoles: parseRoleInput(denyRolesInput.value),
    inherit: permissionForm.value.inherit,
  }),
);

/**
 * 打开运行态写权限弹窗，并将当前数据点权限快照写入表单。
 * @param {object} row - 数据点对象
 */
const openPermissionDialog = (row) => {
  const currentGrant = getWriteRuntimeGrant(row);
  currentPermissionDatapoint.value = row;
  permissionForm.value = currentGrant;
  allowRolesInput.value = currentGrant.allowRoles.join("\n");
  denyRolesInput.value = currentGrant.denyRoles.join("\n");
  permissionDialogVisible.value = true;
};

/**
 * 保存当前数据点的写权限。
 * 边界条件：只提交 write 节点，避免前端误覆盖未来可能扩展的其他动作权限。
 * 异常分支：接口失败时保留当前弹窗内容，便于用户调整后重试。
 */
const handlePermissionSave = async () => {
  if (!props.projectId || !currentPermissionDatapoint.value?.id) {
    return;
  }

  const writeGrant = draftWriteRuntimeGrant.value;

  permissionSaving.value = true;
  try {
    await dataAPI.updateDatapointRuntimePermissions(
      props.projectId,
      currentPermissionDatapoint.value.id,
      { write: writeGrant },
    );
    permissionForm.value = writeGrant;
    ElMessage.success(t("datapoints.runtimePermissionSaveSuccess"));
    permissionDialogVisible.value = false;
    await loadDataPoints();
  } catch (error) {
    ElMessage.error(
      t("datapoints.runtimePermissionSaveFailed", {
        message: getApiErrorMessage(
          error,
          "保存运行态权限失败",
        ),
      }),
    );
  } finally {
    permissionSaving.value = false;
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
    row?.updatedAt || row?.updated_at || row?.createdAt || row?.created_at || ""
  );
};

const groupedDataPoints = computed(() => {
  const items = datapoints.value || [];
  const selectedStatus = props.status ?? statusFilter.value;
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

  if (selectedStatus !== "active") {
    if (invalidItems.length > 0) {
      result.push({
        key: "invalid",
        label: t("datapoints.invalidGroup"),
        items: invalidItems,
        allowClean: true,
      });
    }
  }

  return result;
});

watch([searchText, typeFilter, statusFilter], () => {
  if (debounceTimer.value) {
    window.clearTimeout(debounceTimer.value);
  }
  debounceTimer.value = setTimeout(() => {
    loadDataPoints();
  }, 300);
});

watch(
  () => [props.sourceType, props.status, props.search],
  () => {
    if (debounceTimer.value) {
      window.clearTimeout(debounceTimer.value);
    }
    debounceTimer.value = setTimeout(() => {
      loadDataPoints();
    }, 300);
  },
);

onMounted(() => {
  loadDataPoints();
});

onBeforeUnmount(() => {
  if (debounceTimer.value) {
    window.clearTimeout(debounceTimer.value);
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

.permission-entry {
  padding: 0;
  white-space: normal;
  text-align: left;
}

.permission-dialog {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.permission-dialog__hint {
  font-size: 12px;
  line-height: 1.6;
  color: #6b7280;
}
</style>
