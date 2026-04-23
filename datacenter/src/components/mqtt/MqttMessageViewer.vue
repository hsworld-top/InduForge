<template>
  <div
    class="mqtt-message-viewer h-full flex flex-col bg-white dark:bg-gray-800"
  >
    <!-- 工具栏 -->
    <div
      class="toolbar flex items-center justify-between px-4 py-2 border-b border-gray-200 dark:border-gray-700"
    >
      <div class="flex items-center space-x-3">
        <div class="text-sm font-medium text-gray-700 dark:text-gray-300">
          <IconTablerRss class="inline w-4 h-4 mr-1" />
          {{ subscription?.name || "消息查看器" }}
        </div>
        <el-tag v-if="subscription" size="small" type="info">
          {{ subscription.topic }}
        </el-tag>
        <el-tag :type="isConnected ? 'success' : 'info'" size="small">
          {{ isConnected ? "已连接" : "未连接" }}
        </el-tag>
      </div>

      <div class="flex items-center space-x-2">
        <el-button
          :type="subscriptionEnabled ? 'success' : 'info'"
          size="small"
          @click="handleToggleSubscription"
        >
          <IconTablerPower class="mr-1 w-4 h-4" />
          {{ subscriptionEnabled ? "已启用" : "已禁用" }}
        </el-button>

        <el-input
          v-model="searchText"
          placeholder="搜索消息..."
          size="small"
          style="width: 200px"
          clearable
        >
          <template #prefix>
            <IconTablerSearch class="w-4 h-4" />
          </template>
        </el-input>

        <el-input-number
          v-model="displayLimit"
          :min="10"
          :max="1000"
          :step="10"
          size="small"
          style="width: 120px"
          controls-position="right"
        />
        <span class="text-xs text-gray-500">条消息</span>

        <el-button size="small" @click="handleClear">
          <IconTablerTrash class="mr-1 w-4 h-4" />
          清空
        </el-button>

        <el-dropdown trigger="click" @command="handleFormatCommand">
          <el-button size="small">
            <IconTablerSettings class="mr-1 w-4 h-4" />
            设置
          </el-button>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item command="auto-scroll">
                <el-checkbox v-model="autoScroll" @click.stop>
                  自动滚动
                </el-checkbox>
              </el-dropdown-item>
              <el-dropdown-item command="show-timestamp">
                <el-checkbox v-model="showTimestamp" @click.stop>
                  显示时间戳
                </el-checkbox>
              </el-dropdown-item>
              <el-dropdown-item command="format-json">
                <el-checkbox v-model="formatJson" @click.stop>
                  格式化JSON
                </el-checkbox>
              </el-dropdown-item>
              <el-dropdown-item divided command="export">
                <IconTablerDownload class="mr-2 w-4 h-4" />
                导出消息
              </el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
      </div>
    </div>

    <!-- 消息列表 -->
    <div ref="messagesContainer" class="flex-1 overflow-y-auto relative">
      <el-scrollbar>
        <div v-if="loading" class="p-4 text-center text-gray-500">
          <el-icon class="is-loading"><IconTablerLoader /></el-icon>
          <span class="ml-2">加载消息...</span>
        </div>

        <div
          v-else-if="filteredMessages.length === 0"
          class="p-8 text-center text-gray-400"
        >
          <IconTablerInbox class="mx-auto mb-2 w-12 h-12 opacity-50" />
          <p>暂无消息</p>
          <p class="text-sm mt-1">等待接收MQTT消息...</p>
        </div>

        <div v-else class="divide-y divide-gray-100 dark:divide-gray-700">
          <div
            v-for="(message, index) in filteredMessages"
            :key="message.id || index"
            class="message-item p-3 hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors"
            @click="handleSelectMessage(message)"
          >
            <div class="flex items-start justify-between">
              <div class="flex-1 min-w-0">
                <!-- 消息头部 -->
                <div class="flex items-center space-x-2 mb-2">
                  <el-tag size="small" type="primary">
                    QoS {{ message.qos || 0 }}
                  </el-tag>
                  <span class="text-xs text-gray-500 dark:text-gray-400">
                    {{ message.topic }}
                  </span>
                  <span v-if="showTimestamp" class="text-xs text-gray-400">
                    {{ formatTimestamp(message.timestamp) }}
                  </span>
                </div>

                <!-- 消息内容 -->
                <div
                  class="message-content text-sm text-gray-800 dark:text-gray-200 font-mono bg-gray-50 dark:bg-gray-900 p-2 rounded overflow-x-auto"
                  :class="{ 'whitespace-pre-wrap': formatJson }"
                >
                  {{ formatPayload(message.payload) }}
                </div>
              </div>

              <!-- 操作按钮 -->
              <div class="ml-2">
                <el-button
                  type="primary"
                  size="small"
                  circle
                  @click.stop="handleCopyMessage(message)"
                >
                  <IconTablerCopy class="w-4 h-4" />
                </el-button>
              </div>
            </div>
          </div>
        </div>
      </el-scrollbar>

      <!-- 回到顶部按钮 -->
      <transition name="el-fade-in">
        <el-button
          v-show="showBackToTop"
          type="primary"
          circle
          size="large"
          class="back-to-top-btn"
          @click="smoothScrollToTop"
        >
          <IconTablerArrowUp class="w-5 h-5" />
        </el-button>
      </transition>
    </div>

    <!-- 底部状态栏 -->
    <div
      class="status-bar flex items-center justify-between px-4 py-2 border-t border-gray-200 dark:border-gray-700 text-xs text-gray-500"
    >
      <div>
        共 {{ messages.length }} 条消息
        <span v-if="searchText">
          (筛选后 {{ filteredMessages.length }} 条)</span
        >
      </div>
      <div v-if="lastMessageTime">
        最后消息: {{ formatTimestamp(lastMessageTime) }}
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import {
  ref,
  computed,
  watch,
  nextTick,
  onMounted,
  onBeforeUnmount,
} from "vue";
import { ElMessage } from "element-plus";
import IconTablerRss from "~icons/tabler/rss";
import IconTablerSearch from "~icons/tabler/search";
import IconTablerTrash from "~icons/tabler/trash";
import IconTablerSettings from "~icons/tabler/settings";
import IconTablerDownload from "~icons/tabler/download";
import IconTablerLoader from "~icons/tabler/loader";
import IconTablerInbox from "~icons/tabler/inbox";
import IconTablerCopy from "~icons/tabler/copy";
import IconTablerPower from "~icons/tabler/power";
import IconTablerArrowUp from "~icons/tabler/arrow-up";
import dataAPI from "@/api/data.api";
import dayjs from "dayjs";
import { TIME_FORMAT } from "@/constants";
import { getApiErrorMessage } from "@/utils/request";

const props = defineProps({
  subscription: {
    type: Object,
    default: null,
  },
  projectId: {
    type: String,
    required: true,
  },
  connectionId: {
    type: String,
    required: true,
  },
});

const emit = defineEmits(["message-select"]);

// 状态
const loading = ref(false);
const messages = ref([]);
const searchText = ref("");
const displayLimit = ref(100); // 显示消息数量限制，默认100条
const autoScroll = ref(true);
const showTimestamp = ref(true);
const formatJson = ref(true);
const isConnected = ref(false);
const subscriptionEnabled = ref(false);
const messagesContainer = ref(null);
const isUserAtTop = ref(true); // 用户是否在顶部（用于控制自动滚动）
const showBackToTop = ref(false); // 是否显示回到顶部按钮

// 计算属性
const filteredMessages = computed(() => {
  let filtered = messages.value;

  // 应用搜索过滤
  if (searchText.value) {
    const search = searchText.value.toLowerCase();
    filtered = filtered.filter((msg) => {
      const payload =
        typeof msg.payload === "string"
          ? msg.payload
          : JSON.stringify(msg.payload);
      return (
        msg.topic?.toLowerCase().includes(search) ||
        payload.toLowerCase().includes(search)
      );
    });
  }

  // 应用数量限制（保留最新的N条）
  if (displayLimit.value && filtered.length > displayLimit.value) {
    return filtered.slice(0, displayLimit.value);
  }

  return filtered;
});

const lastMessageTime = computed(() => {
  if (messages.value.length === 0) return null;
  return messages.value[messages.value.length - 1].timestamp;
});

/**
 * 加载历史消息
 */
const loadMessages = async () => {
  if (!props.subscription) return;

  loading.value = true;
  try {
    const response = await dataAPI.getMqttSubscriptionMessages(
      props.projectId,
      props.subscription.id,
    );
    messages.value = response.data || [];
    isUserAtTop.value = true; // 加载后默认在顶部
    await scrollToTop(); // 滚动到顶部显示最新消息
  } catch (error) {
    ElMessage.error(
      "加载消息失败：" + getApiErrorMessage(error, "加载消息失败"),
    );
  } finally {
    loading.value = false;
  }
};

/**
 * 添加新消息
 */
const addMessage = async (message) => {
  // 只有在订阅启用时才添加消息
  if (!subscriptionEnabled.value) {
    return;
  }

  // 添加到数组开头，最新消息在上面
  messages.value.unshift({
    id: Date.now() + Math.random(),
    topic: message.topic,
    payload: message.payload,
    qos: message.qos || 0,
    timestamp: message.timestamp || Date.now(),
  });

  // 限制消息数量（保留最新的1000条）
  if (messages.value.length > 1000) {
    messages.value = messages.value.slice(0, 1000);
  }

  // 只有当自动滚动开启且用户在顶部时，才自动滚动
  if (autoScroll.value && isUserAtTop.value) {
    await scrollToTop();
  }
};

/**
 * 格式化时间戳
 */
const formatTimestamp = (timestamp) => {
  if (!timestamp) return "";
  const date = dayjs(timestamp);
  return date.isValid() ? date.format(TIME_FORMAT) : "";
};

/**
 * 格式化消息内容
 */
const formatPayload = (payload) => {
  if (!payload) return "";

  try {
    // 如果是字符串，尝试解析为JSON
    if (typeof payload === "string") {
      try {
        const json = JSON.parse(payload);
        return formatJson.value ? JSON.stringify(json, null, 2) : payload;
      } catch {
        return payload;
      }
    }

    // 如果是对象，直接序列化
    if (typeof payload === "object") {
      return formatJson.value
        ? JSON.stringify(payload, null, 2)
        : JSON.stringify(payload);
    }

    return String(payload);
  } catch (error) {
    return String(payload);
  }
};

/**
 * 滚动到顶部
 */
const scrollToTop = async () => {
  await nextTick();
  if (messagesContainer.value) {
    const scrollElement = messagesContainer.value.querySelector(
      ".el-scrollbar__wrap",
    );
    if (scrollElement) {
      scrollElement.scrollTop = 0;
    }
  }
};

/**
 * 平滑滚动到顶部（用于回到顶部按钮）
 */
const smoothScrollToTop = async () => {
  await nextTick();
  if (messagesContainer.value) {
    const scrollElement = messagesContainer.value.querySelector(
      ".el-scrollbar__wrap",
    );
    if (scrollElement) {
      scrollElement.scrollTo({
        top: 0,
        behavior: "smooth",
      });
    }
  }
};

/**
 * 滚动到底部
 */
const scrollToBottom = async () => {
  await nextTick();
  if (messagesContainer.value) {
    const scrollElement = messagesContainer.value.querySelector(
      ".el-scrollbar__wrap",
    );
    if (scrollElement) {
      scrollElement.scrollTop = scrollElement.scrollHeight;
    }
  }
};

/**
 * 选择消息
 */
const handleSelectMessage = (message) => {
  emit("message-select", message);
};

/**
 * 复制消息
 */
const handleCopyMessage = async (message) => {
  try {
    const text = formatPayload(message.payload);
    await navigator.clipboard.writeText(text);
    ElMessage.success("已复制到剪贴板");
  } catch (error) {
    ElMessage.error("复制失败");
  }
};

/**
 * 清空消息
 */
const handleClear = () => {
  messages.value = [];
  isUserAtTop.value = true; // 清空后恢复到顶部状态
  ElMessage.success("消息已清空");
};

/**
 * 处理格式化命令
 */
const handleFormatCommand = (command) => {
  if (command === "export") {
    exportMessages();
  }
};

/**
 * 导出消息
 */
const exportMessages = () => {
  try {
    const data = messages.value.map((msg) => ({
      topic: msg.topic,
      payload: formatPayload(msg.payload),
      qos: msg.qos,
      timestamp: formatTimestamp(msg.timestamp),
    }));

    const json = JSON.stringify(data, null, 2);
    const blob = new Blob([json], { type: "application/json" });
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = `mqtt-messages-${Date.now()}.json`;
    link.click();
    URL.revokeObjectURL(url);

    ElMessage.success("导出成功");
  } catch (error) {
    ElMessage.error("导出失败");
  }
};

/**
 * 设置连接状态
 */
const setConnected = (connected) => {
  console.log(
    "[MqttMessageViewer] Setting connected status:",
    connected,
    "for subscription:",
    props.subscription?.id,
  );
  isConnected.value = connected;
};

/**
 * 切换订阅启用/禁用
 */
const handleToggleSubscription = async () => {
  if (!props.subscription) return;

  try {
    const response = await dataAPI.toggleMqttSubscription(
      props.projectId,
      props.subscription.id,
    );

    subscriptionEnabled.value = Boolean(response.data?.isEnabled);
    ElMessage.success(`订阅已${subscriptionEnabled.value ? "启用" : "禁用"}`);
  } catch (error) {
    ElMessage.error("操作失败：" + getApiErrorMessage(error, "操作失败"));
  }
};

// 监听订阅变化
watch(
  () => props.subscription,
  (newSub) => {
    if (newSub) {
      messages.value = [];
      subscriptionEnabled.value = newSub.isEnabled || false;
      loadMessages();
    }
  },
  { immediate: true },
);

/**
 * 处理滚动事件
 */
const handleScroll = () => {
  if (!messagesContainer.value) return;

  const scrollElement = messagesContainer.value.querySelector(
    ".el-scrollbar__wrap",
  );
  if (scrollElement) {
    const scrollTop = scrollElement.scrollTop;
    // 如果滚动位置在顶部附近（小于50px），认为用户在顶部
    isUserAtTop.value = scrollTop < 50;
    // 如果滚动距离超过200px，显示回到顶部按钮
    showBackToTop.value = scrollTop > 200;
  }
};

// 组件挂载
onMounted(() => {
  if (props.subscription) {
    subscriptionEnabled.value = props.subscription.isEnabled || false;
    loadMessages();
  }

  // 添加滚动监听
  nextTick(() => {
    if (messagesContainer.value) {
      const scrollElement = messagesContainer.value.querySelector(
        ".el-scrollbar__wrap",
      );
      if (scrollElement) {
        scrollElement.addEventListener("scroll", handleScroll);
      }
    }
  });
});

// 组件卸载时清理
onBeforeUnmount(() => {
  if (messagesContainer.value) {
    const scrollElement = messagesContainer.value.querySelector(
      ".el-scrollbar__wrap",
    );
    if (scrollElement) {
      scrollElement.removeEventListener("scroll", handleScroll);
    }
  }
});

// 暴露方法
defineExpose({
  addMessage,
  setConnected,
  loadMessages,
});
</script>

<style scoped>
.mqtt-message-viewer {
  font-family:
    -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue",
    Arial, sans-serif;
}

.message-item {
  transition: background-color 0.2s;
}

.message-content {
  max-height: 300px;
  overflow: auto;
  word-break: break-all;
}

.status-bar {
  background-color: #fafafa;
}

.dark .status-bar {
  background-color: #1f2937;
}

/* 回到顶部按钮 */
.back-to-top-btn {
  position: absolute;
  bottom: 24px;
  right: 24px;
  z-index: 10;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
  transition: all 0.3s ease;
}

.back-to-top-btn:hover {
  transform: translateY(-2px);
  box-shadow: 0 6px 16px rgba(0, 0, 0, 0.2);
}
</style>
