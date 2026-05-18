<template>
  <div class="mqtt-subscription-list h-full flex flex-col">
    <!-- 工具栏 -->
    <div
      class="toolbar flex items-center gap-2 px-4 py-2 border-b border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800"
    >
      <el-button
        class="new-subscription-btn"
        type="primary"
        plain
        size="small"
        @click="handleCreate"
      >
        <IconTablerPlus class="mr-1 w-4 h-4" />
        {{ t("subscription.create") }}
      </el-button>
    </div>

    <!-- 订阅列表 -->
    <div class="flex-1 overflow-y-auto">
      <el-scrollbar>
        <div v-if="loading" class="p-4 text-center text-gray-500">
          <el-icon class="is-loading"><IconTablerLoader /></el-icon>
          <span class="ml-2">{{ t("subscription.loading") }}</span>
        </div>

        <div
          v-else-if="subscriptions.length === 0"
          class="p-8 text-center text-gray-400"
        >
          <IconTablerInbox class="mx-auto mb-2 w-12 h-12 opacity-50" />
          <p>{{ t("subscription.empty") }}</p>
          <p class="text-sm mt-1">{{ t("subscription.emptyHint") }}</p>
        </div>

        <el-table
          v-else
          :data="subscriptions"
          :table-layout="'auto'"
          size="small"
          row-key="id"
          stripe
          class="subscription-table"
          @row-click="handleSelect"
          @row-dblclick="handleManageTags"
          @row-contextmenu="
            (row, column, event) => handleContextMenu(event, row)
          "
        >
          <el-table-column :label="t('subscription.name')" min-width="120">
            <template #default="{ row }">
              <span
                class="text-sm font-medium text-gray-900 dark:text-gray-100 truncate"
              >
                {{ row.name || "-" }}
              </span>
            </template>
          </el-table-column>
          <el-table-column :label="t('subscription.topic')" min-width="160">
            <template #default="{ row }">
              <span class="text-xs text-gray-500 dark:text-gray-400 truncate">
                {{ row.topic }}
              </span>
            </template>
          </el-table-column>
          <el-table-column prop="qos" label="QoS" width="70" />
          <el-table-column :label="t('subscription.status')" width="90">
            <template #default="{ row }">
              <el-tag
                size="small"
                :type="row.isEnabled === false ? 'info' : 'success'"
              >
                {{
                  row.isEnabled === false
                    ? t("common.disabled")
                    : t("common.enabled")
                }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column :label="t('subscription.remark')" min-width="120">
            <template #default="{ row }">
              <span class="text-xs text-gray-500 dark:text-gray-400 truncate">
                {{ row.description || "-" }}
              </span>
            </template>
          </el-table-column>
          <el-table-column :label="t('subscription.operations')" width="280">
            <template #default="{ row }">
              <div class="action-cell flex items-center gap-2">
                <el-tooltip
                  :content="t('actions.viewMessages')"
                  placement="top"
                >
                  <el-button
                    class="action-link"
                    link
                    size="small"
                    @click.stop="handleView(row)"
                  >
                    {{ t("actions.viewMessages") }}
                  </el-button>
                </el-tooltip>
                <el-tooltip
                  :content="t('actions.manageVariables')"
                  placement="top"
                >
                  <el-button
                    class="action-link"
                    link
                    size="small"
                    @click.stop="handleManageTags(row)"
                  >
                    {{ t("actions.manageVariables") }}
                  </el-button>
                </el-tooltip>
                <el-tooltip :content="t('actions.edit')" placement="top">
                  <el-button
                    class="action-link"
                    link
                    size="small"
                    @click.stop="handleEdit(row)"
                  >
                    {{ t("actions.edit") }}
                  </el-button>
                </el-tooltip>
                <el-tooltip :content="t('actions.delete')" placement="top">
                  <el-button
                    class="action-link"
                    link
                    size="small"
                    @click.stop="handleDelete(row)"
                  >
                    {{ t("actions.delete") }}
                  </el-button>
                </el-tooltip>
              </div>
            </template>
          </el-table-column>
        </el-table>
      </el-scrollbar>
    </div>

    <!-- 订阅管理对话框 -->
    <MqttSubscriptionDialog
      v-model="showDialog"
      :connection-id="connectionId"
      :subscription="currentSubscription"
      :mode="dialogMode"
      @success="handleDialogSuccess"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, watch } from "vue";
import { ElMessage, ElMessageBox } from "element-plus";
import IconTablerPlus from "~icons/tabler/plus";
import IconTablerLoader from "~icons/tabler/loader";
import IconTablerInbox from "~icons/tabler/inbox";
import MqttSubscriptionDialog from "./MqttSubscriptionDialog.vue";
import dataAPI from "@/api/data.api";
import { t } from "@/i18n/runtime";
import { getApiErrorMessage } from "@/utils/request";

const props = defineProps({
  connectionId: {
    type: String,
    required: true,
  },
  projectId: {
    type: String,
    required: true,
  },
});

const emit = defineEmits([
  "view-messages",
  "subscription-select",
  "manage-tags",
  "subscription-deleted",
]);

const loading = ref(false);
const subscriptions = ref([]);
const showDialog = ref(false);
const dialogMode = ref("create");
const currentSubscription = ref(null);

/**
 * 加载订阅列表
 */
const loadSubscriptions = async () => {
  loading.value = true;
  try {
    const response = await dataAPI.getMqttSubscriptions(
      props.projectId,
      props.connectionId,
    );
    /* 后端返回结构：{ data: { pagination, subscriptions } } */
    subscriptions.value = response.data?.subscriptions || [];
  } catch (error) {
    ElMessage.error(
      t("subscription.loadFailed", {
        message: getApiErrorMessage(error, "加载订阅失败"),
      }),
    );
  } finally {
    loading.value = false;
  }
};

/**
 * 创建订阅
 */
const handleCreate = () => {
  dialogMode.value = "create";
  currentSubscription.value = null;
  showDialog.value = true;
};

/**
 * 选择订阅
 */
const handleSelect = (subscription) => {
  emit("subscription-select", subscription);
};

/**
 * 查看消息
 */
const handleView = (subscription) => {
  emit("view-messages", subscription);
};

/**
 * 管理变量
 */
const handleManageTags = (subscription) => {
  emit("manage-tags", subscription);
};

/**
 * 编辑订阅
 */
const handleEdit = (subscription) => {
  dialogMode.value = "edit";
  currentSubscription.value = subscription;
  showDialog.value = true;
};

/**
 * 删除订阅
 */
const handleDelete = async (subscription) => {
  try {
    await ElMessageBox.confirm(
      t("subscription.deleteConfirm", { name: subscription.name }),
      t("subscription.deleteConfirmTitle"),
      {
        confirmButtonText: t("actions.delete"),
        cancelButtonText: t("actions.cancel"),
        type: "warning",
      },
    );

    await dataAPI.deleteMqttSubscription(props.projectId, subscription.id);
    ElMessage.success(t("subscription.deleted"));
    await loadSubscriptions();
    emit("subscription-deleted", subscription);
  } catch (error) {
    if (error !== "cancel") {
      ElMessage.error(
        t("subscription.deleteFailed", {
          message: getApiErrorMessage(error, "删除订阅失败"),
        }),
      );
    }
  }
};

/**
 * 右键菜单
 */
const handleContextMenu = (event, subscription) => {
  if (event?.preventDefault) {
    event.preventDefault();
  }
  const menu = document.createElement("div");
  menu.className =
    "fixed bg-white dark:bg-gray-800 rounded-lg shadow-lg border border-gray-200 dark:border-gray-700 py-1 min-w-[140px]";
  menu.style.cssText = `position: fixed; left: ${event.clientX}px; top: ${event.clientY}px; z-index: 9999;`;

  const openItem = document.createElement("div");
  openItem.className =
    "px-4 py-2 text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 cursor-pointer";
  openItem.textContent = t("actions.viewMessages");
  openItem.onclick = () => {
    handleView(subscription);
    document.body.removeChild(menu);
  };

  const manageItem = document.createElement("div");
  manageItem.className =
    "px-4 py-2 text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 cursor-pointer";
  manageItem.textContent = t("actions.manageVariables");
  manageItem.onclick = () => {
    handleManageTags(subscription);
    document.body.removeChild(menu);
  };

  const editItem = document.createElement("div");
  editItem.className =
    "px-4 py-2 text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 cursor-pointer";
  editItem.textContent = t("subscription.edit");
  editItem.onclick = () => {
    handleEdit(subscription);
    document.body.removeChild(menu);
  };

  const deleteItem = document.createElement("div");
  deleteItem.className =
    "px-4 py-2 text-sm text-red-600 hover:bg-red-50 dark:hover:bg-red-900/30 cursor-pointer";
  deleteItem.textContent = t("actions.delete");
  deleteItem.onclick = async () => {
    document.body.removeChild(menu);
    await handleDelete(subscription);
  };

  menu.appendChild(openItem);
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

/**
 * 对话框成功回调
 */
const handleDialogSuccess = () => {
  loadSubscriptions();
};

// 监听连接ID变化
watch(
  () => props.connectionId,
  () => {
    if (props.connectionId) {
      loadSubscriptions();
    }
  },
  { immediate: true },
);

// 组件挂载
onMounted(() => {
  loadSubscriptions();
});

// 暴露方法
defineExpose({
  loadSubscriptions,
  openEditDialog: (subscription) => {
    if (!subscription) return;
    handleEdit(subscription);
  },
});
</script>

<style scoped>
.subscription-table :deep(.el-table__row) {
  cursor: pointer;
}

.action-cell {
  flex-wrap: wrap;
  row-gap: 4px;
}

.new-subscription-btn {
  --el-button-text-color: #1f2937;
  --el-button-border-color: #e5e7eb;
  --el-button-bg-color: #f9fafb;
  --el-button-hover-text-color: #111827;
  --el-button-hover-border-color: #d1d5db;
  --el-button-hover-bg-color: #f3f4f6;
}

.action-link {
  --el-button-text-color: #4b5563;
  --el-button-hover-text-color: #111827;
}
</style>

<style scoped>
.mqtt-subscription-list {
  background: white;
}

.dark .mqtt-subscription-list {
  background: #1f2937;
}

.dark .new-subscription-btn {
  --el-button-text-color: #e5e7eb;
  --el-button-border-color: #374151;
  --el-button-bg-color: #111827;
  --el-button-hover-text-color: #ffffff;
  --el-button-hover-border-color: #4b5563;
  --el-button-hover-bg-color: #1f2937;
}

.dark .action-link {
  --el-button-text-color: #d1d5db;
  --el-button-hover-text-color: #ffffff;
}

.subscription-item {
  transition: background-color 0.2s;
}
</style>
