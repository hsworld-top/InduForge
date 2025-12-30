<template>
  <div class="mqtt-subscription-list h-full flex flex-col">
    <!-- 工具栏 -->
    <div
      class="toolbar flex items-center justify-between px-4 py-2 border-b border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800"
    >
      <div class="text-sm font-medium text-gray-700 dark:text-gray-300">
        订阅列表
      </div>
      <el-button type="primary" size="small" @click="handleCreate">
        <IconTablerPlus class="mr-1 w-4 h-4" />
        新建订阅
      </el-button>
    </div>

    <!-- 订阅列表 -->
    <div class="flex-1 overflow-y-auto">
      <el-scrollbar>
        <div v-if="loading" class="p-4 text-center text-gray-500">
          <el-icon class="is-loading"><IconTablerLoader /></el-icon>
          <span class="ml-2">加载中...</span>
        </div>

        <div
          v-else-if="subscriptions.length === 0"
          class="p-8 text-center text-gray-400"
        >
          <IconTablerInbox class="mx-auto mb-2 w-12 h-12 opacity-50" />
          <p>暂无订阅</p>
          <p class="text-sm mt-1">点击上方按钮创建新订阅</p>
        </div>

        <div v-else class="divide-y divide-gray-200 dark:divide-gray-700">
          <div
            v-for="subscription in subscriptions"
            :key="subscription.id"
            class="subscription-item p-3 hover:bg-gray-50 dark:hover:bg-gray-700 cursor-pointer transition-colors"
            @click="handleSelect(subscription)"
            @dblclick="handleView(subscription)"
            @contextmenu.prevent="handleContextMenu($event, subscription)"
          >
            <div class="flex items-start justify-between">
              <div class="flex-1 min-w-0">
                <div class="flex items-center space-x-2">
                  <el-tag
                    :type="subscription.isEnabled ? 'success' : 'info'"
                    size="small"
                  >
                    {{ subscription.isEnabled ? "启用" : "禁用" }}
                  </el-tag>
                  <span
                    class="font-medium text-gray-900 dark:text-gray-100 truncate"
                  >
                    {{ subscription.name }}
                  </span>
                </div>
                <div
                  class="mt-1 text-sm text-gray-500 dark:text-gray-400 truncate"
                >
                  <IconTablerRss class="inline w-3 h-3 mr-1" />
                  {{ subscription.topic }}
                </div>
                <div
                  class="mt-1 flex items-center space-x-3 text-xs text-gray-400"
                >
                  <span>QoS: {{ subscription.qos }}</span>
                  <span
                    v-if="subscription.description"
                    class="truncate max-w-[200px]"
                  >
                    {{ subscription.description }}
                  </span>
                </div>
              </div>
              <div class="flex items-center space-x-1 ml-2">
                <el-tooltip content="查看消息" placement="top">
                  <el-button
                    type="primary"
                    size="small"
                    circle
                    @click.stop="handleView(subscription)"
                  >
                    <IconTablerEye class="w-4 h-4" />
                  </el-button>
                </el-tooltip>
                <el-tooltip
                  :content="subscription.isEnabled ? '禁用' : '启用'"
                  placement="top"
                >
                  <el-button
                    :type="subscription.isEnabled ? 'warning' : 'success'"
                    size="small"
                    circle
                    @click.stop="handleToggle(subscription)"
                  >
                    <IconTablerPower class="w-4 h-4" />
                  </el-button>
                </el-tooltip>
                <el-dropdown
                  trigger="click"
                  @command="(cmd) => handleCommand(cmd, subscription)"
                >
                  <el-button size="small" circle @click.stop>
                    <IconTablerDots class="w-4 h-4" />
                  </el-button>
                  <template #dropdown>
                    <el-dropdown-menu>
                      <el-dropdown-item command="edit">
                        <IconTablerEdit class="mr-2 w-4 h-4" />
                        编辑
                      </el-dropdown-item>
                      <el-dropdown-item command="delete" divided>
                        <IconTablerTrash class="mr-2 w-4 h-4 text-red-500" />
                        <span class="text-red-500">删除</span>
                      </el-dropdown-item>
                    </el-dropdown-menu>
                  </template>
                </el-dropdown>
              </div>
            </div>
          </div>
        </div>
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

<script setup>
import { ref, onMounted, watch } from "vue";
import { ElMessage, ElMessageBox } from "element-plus";
import IconTablerPlus from "~icons/tabler/plus";
import IconTablerLoader from "~icons/tabler/loader";
import IconTablerInbox from "~icons/tabler/inbox";
import IconTablerRss from "~icons/tabler/rss";
import IconTablerEye from "~icons/tabler/eye";
import IconTablerPower from "~icons/tabler/power";
import IconTablerDots from "~icons/tabler/dots";
import IconTablerEdit from "~icons/tabler/edit";
import IconTablerTrash from "~icons/tabler/trash";
import MqttSubscriptionDialog from "./MqttSubscriptionDialog.vue";
import dataAPI from "@/api/data.api";

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

const emit = defineEmits(["view-messages", "subscription-select"]);

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
      props.connectionId
    );
    if (response.success) {
      subscriptions.value = response.data || [];
    }
  } catch (error) {
    ElMessage.error(
      "加载订阅列表失败：" + (error.response?.data?.message || error.message)
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
 * 切换启用状态
 */
const handleToggle = async (subscription) => {
  try {
    const response = await dataAPI.toggleMqttSubscription(
      props.projectId,
      subscription.id
    );
    if (response.success) {
      subscription.isEnabled = response.data.isEnabled;
      ElMessage.success(`订阅已${subscription.isEnabled ? "启用" : "禁用"}`);
    }
  } catch (error) {
    ElMessage.error(
      "操作失败：" + (error.response?.data?.message || error.message)
    );
  }
};

/**
 * 处理命令
 */
const handleCommand = (command, subscription) => {
  if (command === "edit") {
    handleEdit(subscription);
  } else if (command === "delete") {
    handleDelete(subscription);
  }
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
      `确定要删除订阅 "${subscription.name}" 吗？`,
      "删除确认",
      {
        confirmButtonText: "确定",
        cancelButtonText: "取消",
        type: "warning",
      }
    );

    await dataAPI.deleteMqttSubscription(props.projectId, subscription.id);
    ElMessage.success("订阅已删除");
    await loadSubscriptions();
  } catch (error) {
    if (error !== "cancel") {
      ElMessage.error(
        "删除失败：" + (error.response?.data?.message || error.message)
      );
    }
  }
};

/**
 * 右键菜单
 */
const handleContextMenu = (event, subscription) => {
  // TODO: 实现右键菜单
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
  { immediate: true }
);

// 组件挂载
onMounted(() => {
  loadSubscriptions();
});

// 暴露方法
defineExpose({
  loadSubscriptions,
});
</script>

<style scoped>
.mqtt-subscription-list {
  background: white;
}

.dark .mqtt-subscription-list {
  background: #1f2937;
}

.subscription-item {
  transition: background-color 0.2s;
}
</style>
