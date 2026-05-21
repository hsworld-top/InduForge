<template>
  <div class="mqtt-message-viewer">
    <WorkbenchStreamToolbar
      v-model:search="searchText"
      v-model:limit="displayLimit"
      v-model:format-json="formatJson"
      v-model:auto-scroll="autoScroll"
      v-model:show-timestamp="showTimestamp"
      :icon="IconTablerRss"
      :title="subscriptionTitle"
      :subtitle="subscription?.topic || ''"
      :status-label="isConnected ? '实时通道已连接' : '等待实时通道'"
      :status-tone="isConnected ? 'success' : 'neutral'"
      :loading="loading"
      @refresh="loadMessages"
      @clear="handleClear"
    />

    <div class="mqtt-message-viewer__body">
      <WorkbenchStreamMessageList
        ref="messageListRef"
        :messages="filteredMessages"
        :loading="loading"
        :format-json="formatJson"
        :show-timestamp="showTimestamp"
        empty-text="暂无 MQTT 消息"
        empty-hint="启动预览连接后，实时消息会显示在这里"
        @select="handleSelectMessage"
        @copy="handleCopyMessage"
      />
    </div>

    <footer class="mqtt-message-viewer__status">
      <span>
        共 {{ messages.length }} 条
        <template v-if="searchText"> · 筛选 {{ filteredMessages.length }} 条</template>
      </span>
      <span v-if="lastMessageTime">最后消息 {{ formatTimestamp(lastMessageTime) }}</span>
      <button type="button" class="mqtt-message-viewer__toggle" @click="handleToggleSubscription">
        {{ subscriptionEnabled ? "禁用订阅" : "启用订阅" }}
      </button>
    </footer>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, ref, watch } from "vue";
import { ElMessage } from "element-plus";
import dayjs from "dayjs";
import IconTablerRss from "~icons/tabler/rss";
import dataAPI from "@/api/data.api";
import { TIME_FORMAT } from "@/constants";
import { getApiErrorMessage } from "@/utils/request";
import WorkbenchStreamMessageList from "@/components/workbench/WorkbenchStreamMessageList.vue";
import WorkbenchStreamToolbar from "@/components/workbench/WorkbenchStreamToolbar.vue";

type MqttMessage = {
  id?: string | number;
  topic?: string;
  payload?: unknown;
  qos?: number;
  timestamp?: string | number | Date;
};

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

const loading = ref(false);
const messages = ref<MqttMessage[]>([]);
const searchText = ref("");
const displayLimit = ref(100);
const autoScroll = ref(true);
const showTimestamp = ref(true);
const formatJson = ref(true);
const isConnected = ref(false);
const subscriptionEnabled = ref(false);
const messageListRef = ref<any>(null);

const subscriptionTitle = computed(
  () => props.subscription?.name || props.subscription?.topic || "消息查看器",
);

const filteredMessages = computed(() => {
  const keyword = searchText.value.trim().toLowerCase();
  let result = messages.value;

  if (keyword) {
    result = result.filter((message) => {
      const topic = String(message.topic || "").toLowerCase();
      const payload =
        typeof message.payload === "string"
          ? message.payload
          : JSON.stringify(message.payload ?? "");
      return topic.includes(keyword) || payload.toLowerCase().includes(keyword);
    });
  }

  return displayLimit.value > 0 ? result.slice(0, displayLimit.value) : result;
});

const lastMessageTime = computed(() => messages.value[0]?.timestamp || null);

const formatTimestamp = (timestamp) => {
  const date = dayjs(timestamp);
  return date.isValid() ? date.format(TIME_FORMAT) : "-";
};

const formatPayload = (payload) => {
  if (payload === null || payload === undefined) return "";
  if (typeof payload === "string") {
    if (!formatJson.value) return payload;
    try {
      return JSON.stringify(JSON.parse(payload), null, 2);
    } catch {
      return payload;
    }
  }
  try {
    return JSON.stringify(payload, null, formatJson.value ? 2 : 0);
  } catch {
    return String(payload);
  }
};

const normalizeIncomingMessage = (message) => {
  const payload = message?.message || message || {};
  return {
    id: payload.id || `${Date.now()}-${Math.random()}`,
    topic: payload.topic || props.subscription?.topic || "",
    payload: payload.payload,
    qos: payload.qos ?? 0,
    timestamp: payload.timestamp || payload.receivedAt || Date.now(),
  };
};

/**
 * 历史消息只作为当前订阅的启动快照；实时消息仍由工作台 socket 追加。
 */
const loadMessages = async () => {
  if (!props.subscription?.id) return;

  loading.value = true;
  try {
    const response = await dataAPI.getMqttSubscriptionMessages(
      props.projectId,
      props.subscription.id,
      { limit: displayLimit.value },
    );
    messages.value = (response.data?.list || []).map(normalizeIncomingMessage);
    await messageListRef.value?.scrollToTop?.();
  } catch (error) {
    ElMessage.error(
      "加载消息失败：" + getApiErrorMessage(error, "加载消息失败"),
    );
  } finally {
    loading.value = false;
  }
};

const addMessage = async (message) => {
  messages.value.unshift(normalizeIncomingMessage(message));
  if (messages.value.length > 1000) {
    messages.value = messages.value.slice(0, 1000);
  }
  if (autoScroll.value) {
    await nextTick();
    await messageListRef.value?.scrollToTop?.();
  }
};

const setConnected = (connected) => {
  isConnected.value = Boolean(connected);
};

const handleSelectMessage = (message) => {
  emit("message-select", message);
};

const handleCopyMessage = async (message) => {
  try {
    await navigator.clipboard.writeText(formatPayload(message.payload));
    ElMessage.success("已复制 Payload");
  } catch {
    ElMessage.error("复制失败");
  }
};

const handleClear = () => {
  messages.value = [];
};

const handleToggleSubscription = async () => {
  if (!props.subscription?.id) return;

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

watch(
  () => props.subscription,
  (subscription) => {
    messages.value = [];
    searchText.value = "";
    isConnected.value = false;
    subscriptionEnabled.value = Boolean(subscription?.isEnabled);
    if (subscription?.id) {
      loadMessages();
    }
  },
  { immediate: true },
);

onMounted(() => {
  subscriptionEnabled.value = Boolean(props.subscription?.isEnabled);
});

defineExpose({
  addMessage,
  setConnected,
  loadMessages,
});
</script>

<style scoped>
.mqtt-message-viewer {
  height: 100%;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  background: var(--dc-surface-raised);
}

.mqtt-message-viewer__body {
  min-height: 0;
  flex: 1;
}

.mqtt-message-viewer__status {
  min-height: 34px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 0 12px;
  border-top: 1px solid var(--dc-border);
  background: var(--dc-surface-subtle);
  color: var(--dc-text-muted);
  font-size: 11px;
}

.mqtt-message-viewer__toggle {
  height: 24px;
  padding: 0 9px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
  color: var(--dc-text-secondary);
  font-size: 11px;
  font-weight: 700;
}

.mqtt-message-viewer__toggle:hover {
  border-color: color-mix(in oklch, var(--dc-primary) 28%, var(--dc-border));
  color: var(--dc-primary);
}
</style>
