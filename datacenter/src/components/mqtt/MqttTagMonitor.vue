<template>
  <div class="mqtt-tag-monitor h-full flex flex-col">
    <div class="flex items-center justify-between p-3 border-b bg-gray-50">
      <div class="flex items-center gap-2">
        <span class="text-sm font-semibold">Tag实时监控</span>
        <el-tag v-if="enabledTags.length > 0" size="small" type="info">
          {{ enabledTags.length }} 个启用变量
        </el-tag>
        <el-tag
          v-if="tags.length > enabledTags.length"
          size="small"
          type="warning"
        >
          {{ tags.length - enabledTags.length }} 个已禁用
        </el-tag>
        <el-tag :type="socketConnected ? 'success' : 'danger'" size="small">
          {{ socketConnected ? "已连接" : "未连接" }}
        </el-tag>
      </div>
      <div class="flex items-center gap-2">
        <div class="view-toggle">
          <el-button
            size="small"
            :type="viewMode === 'list' ? 'primary' : 'default'"
            @click="viewMode = 'list'"
          >
            列表显示
          </el-button>
          <el-button
            size="small"
            :type="viewMode === 'card' ? 'primary' : 'default'"
            @click="viewMode = 'card'"
          >
            卡片显示
          </el-button>
        </div>
        <el-tooltip content="重新加载变量配置" placement="top">
          <el-button size="small" @click="handleRefresh">
            <IconTablerRefresh class="mr-1 w-4 h-4" />
            重新加载
          </el-button>
        </el-tooltip>
      </div>
    </div>

    <div class="flex-1 overflow-auto p-3">
      <div
        v-if="enabledTags.length === 0"
        class="text-center text-gray-400 py-20"
      >
        <IconTablerFile class="text-4xl mb-2 w-10 h-10" />
        <div v-if="tags.length === 0">暂无变量</div>
        <div v-else>暂无启用的变量</div>
        <div class="text-xs mt-1">
          {{
            tags.length === 0
              ? "请先在左侧创建变量"
              : "请在左侧启用变量以开始监控"
          }}
        </div>
      </div>

      <div
        v-else-if="viewMode === 'card'"
        class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3"
      >
        <div
          v-for="tag in enabledTags"
          :key="tag.id"
          class="tag-card border-2 rounded-lg p-3 bg-white shadow-sm hover:shadow-md transition-all duration-300"
          :class="{
            'border-green-400 bg-green-50':
              tag.currentValue?.quality === 'good',
            'border-red-400 bg-red-50': tag.currentValue?.quality === 'bad',
            'border-yellow-400 bg-yellow-50':
              tag.currentValue?.quality === 'uncertain',
            'border-gray-300': !tag.currentValue,
          }"
        >
          <div class="flex items-start justify-between mb-2">
            <div class="flex-1 min-w-0">
              <div class="text-sm font-semibold text-gray-800 truncate">
                {{ tag.name }}
              </div>
              <div class="text-xs text-gray-500 font-mono mt-1">
                {{ tag.code }}
              </div>
            </div>
            <div class="flex items-center gap-1 ml-2">
              <el-tag
                :type="getQualityColor(tag.currentValue?.quality)"
                size="small"
                effect="dark"
              >
                {{ getQualityLabel(tag.currentValue?.quality || "unknown") }}
              </el-tag>
            </div>
          </div>

          <div
            class="tag-value mt-2 p-3 bg-white rounded-lg border border-gray-200"
          >
            <div
              v-if="tag.currentValue"
              class="flex items-baseline justify-between"
            >
              <span class="text-2xl font-bold text-gray-900 truncate">
                {{ formatValue(tag.currentValue.parsedValue, tag.dataType) }}
              </span>
              <span
                v-if="tag.unit"
                class="text-sm text-gray-600 ml-2 font-medium"
              >
                {{ tag.unit }}
              </span>
            </div>
            <div v-else class="text-gray-400 text-center py-2">
              <IconTablerLoader class="mr-1 w-4 h-4 animate-spin" />
              等待数据...
            </div>
          </div>

          <div class="mt-2 text-xs text-gray-500 space-y-1">
            <div class="flex justify-between">
              <span>数据类型:</span>
              <span>{{ getDataTypeLabel(tag.dataType) }}</span>
            </div>
            <div class="flex justify-between">
              <span>解析类型:</span>
              <span>{{ getParseTypeLabel(tag.parseType) }}</span>
            </div>
            <div v-if="tag.currentValue" class="flex justify-between">
              <span>数据质量:</span>
              <el-tag
                :type="getQualityColor(tag.currentValue.quality)"
                size="small"
              >
                {{ getQualityLabel(tag.currentValue.quality) }}
              </el-tag>
            </div>
            <div
              v-if="tag.currentValue?.timestamp"
              class="flex justify-between"
            >
              <span>更新时间:</span>
              <span>{{ formatTimestamp(tag.currentValue.timestamp) }}</span>
            </div>
            <div v-if="tag.currentValue?.error" class="mt-2 text-red-500">
              <IconTablerAlertTriangle class="w-4 h-4" />
              {{ tag.currentValue.error }}
            </div>
          </div>
        </div>
      </div>

      <div v-else class="tag-row-list">
        <div class="tag-row tag-row-header">
          <span>变量名</span>
          <span>类型</span>
          <span>当前值</span>
          <span>时间戳</span>
          <span>质量</span>
        </div>
        <div v-for="tag in enabledTags" :key="tag.id" class="tag-row">
          <span class="truncate" :title="tag.name">{{ tag.name }}</span>
          <span>{{ getDataTypeLabel(tag.dataType) }}</span>
          <span class="truncate">
            {{
              tag.currentValue
                ? formatValue(tag.currentValue.parsedValue, tag.dataType)
                : "-"
            }}
          </span>
          <span class="truncate">
            {{
              tag.currentValue?.timestamp
                ? formatTimestamp(tag.currentValue.timestamp)
                : "-"
            }}
          </span>
          <span>
            <el-tag
              :type="getQualityColor(tag.currentValue?.quality || 'unknown')"
              size="small"
            >
              {{ getQualityLabel(tag.currentValue?.quality || "unknown") }}
            </el-tag>
          </span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount, watch, toRef } from "vue";
import { ElMessage } from "element-plus";
import { getMqttTags } from "@/api/data.api";
import { useMqttSocket } from "@/composables/useMqttSocket";
import { useMqttTagSync } from "@/composables/useMqttTagSync";
import IconTablerRefresh from "~icons/tabler/refresh";
import IconTablerFile from "~icons/tabler/file";
import IconTablerLoader from "~icons/tabler/loader";
import IconTablerAlertTriangle from "~icons/tabler/alert-triangle";
import dayjs from "dayjs";
import { TIME_FORMAT } from "@/constants";

const props = defineProps({
  projectId: {
    type: String,
    required: true,
  },
  subscriptionId: {
    type: String,
    required: true,
  },
  previewSessionId: {
    type: String,
    default: "",
  },
});

const tags = ref([]);
const viewMode = ref("list");

const enabledTags = computed(() => {
  return tags.value.filter((tag) => tag.isEnabled);
});

const {
  connected: socketConnected,
  disconnect,
  onMessage,
  subscribeTag,
} = useMqttSocket(toRef(props, "projectId"), toRef(props, "previewSessionId"));

const tagSubscriptionCleanups = new Map();

const { subscribe: subscribeTagSync, unsubscribe: unsubscribeTagSync } =
  useMqttTagSync(props.subscriptionId);

const fetchTags = async () => {
  try {
    const res = await getMqttTags(props.projectId, props.subscriptionId);
    tags.value = res.data || [];
  } catch (error) {
    ElMessage.error(`获取变量列表失败: ${error.message}`);
  }
};

const handleRefresh = () => {
  fetchTags();
};

const formatValue = (value, dataType) => {
  if (value === null || value === undefined) return "-";

  try {
    if (dataType === "object" || dataType === "array") {
      const parsed = JSON.parse(value);
      return JSON.stringify(parsed, null, 2);
    }
    if (dataType === "number") {
      const num = parseFloat(value);
      return Number.isNaN(num) ? value : num.toFixed(2);
    }
    return value;
  } catch {
    return value;
  }
};

const formatTimestamp = (timestamp) => {
  const date = dayjs(timestamp);
  return date.isValid() ? date.format(TIME_FORMAT) : "-";
};

const getDataTypeLabel = (dataType) => {
  const labels = {
    string: "字符串",
    number: "数值",
    boolean: "布尔",
    object: "对象",
    array: "数组",
  };
  return labels[dataType] || dataType;
};

const getParseTypeLabel = (parseType) => {
  const labels = {
    jsonpath: "JSONPath",
    regex: "正则",
    script: "脚本",
    fixed: "固定值",
  };
  return labels[parseType] || parseType;
};

const getQualityLabel = (quality) => {
  const labels = {
    good: "良好",
    bad: "错误",
    uncertain: "不确定",
    unknown: "未知",
  };
  return labels[quality] || quality;
};

const getQualityColor = (quality) => {
  const colors = {
    good: "success",
    bad: "danger",
    uncertain: "warning",
    unknown: "info",
  };
  return colors[quality] || "info";
};

const handleTagValueUpdate = (data) => {
  const tag = tags.value.find((item) => item.id === data.tagId);
  if (!tag) {
    return;
  }

  tag.currentValue = {
    parsedValue: data.value,
    quality: data.quality,
    timestamp: data.timestamp,
    error: data.error,
  };
};

const handleTagSyncEvent = async (event) => {
  console.log("[MqttTagMonitor] Received sync event:", event);

  switch (event.type) {
    case "created":
    case "updated":
    case "refresh":
      await fetchTags();
      break;
    case "deleted":
      tags.value = tags.value.filter((tag) => tag.id !== event.data.tagId);
      break;
    case "toggled": {
      const tag = tags.value.find((item) => item.id === event.data.tagId);
      if (tag) {
        tag.isEnabled = event.data.isEnabled;
        if (socketConnected.value) {
          syncSubscriptions();
        }
      }
      break;
    }
  }
};

const syncSubscriptions = () => {
  if (!socketConnected.value) {
    return;
  }

  const desiredIds = new Set(enabledTags.value.map((tag) => tag.id));

  desiredIds.forEach((tagId) => {
    if (!tagSubscriptionCleanups.has(tagId)) {
      tagSubscriptionCleanups.set(tagId, subscribeTag(tagId));
    }
  });

  Array.from(tagSubscriptionCleanups.entries()).forEach(([tagId, cleanup]) => {
    if (desiredIds.has(tagId)) {
      return;
    }
    cleanup?.();
    tagSubscriptionCleanups.delete(tagId);
  });
};

let stopSocketWatch = null;
let unsubscribeMessage = null;

onMounted(async () => {
  await fetchTags();
  subscribeTagSync(handleTagSyncEvent);

  unsubscribeMessage = onMessage((data) => {
    console.log("[MqttTagMonitor] Received message:", data);
    if (data?.tagId) {
      handleTagValueUpdate(data);
    }
  });

  stopSocketWatch = watch(
    () => [
      socketConnected.value,
      enabledTags.value.map((tag) => tag.id).join(","),
    ],
    () => {
      if (!socketConnected.value) {
        return;
      }
      syncSubscriptions();
    },
    { immediate: true },
  );
});

onBeforeUnmount(() => {
  stopSocketWatch?.();
  unsubscribeMessage?.();
  unsubscribeTagSync(handleTagSyncEvent);
  Array.from(tagSubscriptionCleanups.values()).forEach((cleanup) => {
    cleanup?.();
  });
  tagSubscriptionCleanups.clear();
  disconnect();
});

defineExpose({
  refresh: fetchTags,
});
</script>

<style scoped>
.mqtt-tag-monitor {
  background: linear-gradient(to bottom, #f5f7fa, #e9ecef);
}

.view-toggle {
  display: inline-flex;
  gap: 6px;
}

.tag-row-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.tag-row {
  display: grid;
  grid-template-columns: 1.4fr 0.6fr 1fr 1.2fr 0.6fr;
  gap: 12px;
  align-items: center;
  padding: 8px 10px;
  background: #fff;
  border: 1px solid #e4e7ed;
  border-radius: 6px;
  font-size: 12px;
  color: #374151;
}

.tag-row-header {
  background: #f9fafb;
  font-weight: 600;
  color: #6b7280;
}

.tag-row .truncate {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.tag-card {
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  position: relative;
  overflow: hidden;
}

.tag-card::before {
  content: "";
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 4px;
  background: linear-gradient(90deg, transparent, currentColor, transparent);
  opacity: 0;
  transition: opacity 0.3s;
}

.tag-card:hover {
  transform: translateY(-2px) scale(1.01);
}

.tag-card:hover::before {
  opacity: 0.3;
}

.tag-value {
  min-height: 64px;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.3s;
}

.tag-value:hover {
  background: #f8f9fa;
}

.tag-card.border-green-400 {
  animation: pulse-green 2s infinite;
}

.tag-card.border-red-400 {
  animation: pulse-red 2s infinite;
}

.tag-card.border-yellow-400 {
  animation: pulse-yellow 2s infinite;
}

@keyframes pulse-green {
  0%,
  100% {
    box-shadow: 0 0 0 0 rgba(52, 211, 153, 0.4);
  }
  50% {
    box-shadow: 0 0 0 8px rgba(52, 211, 153, 0);
  }
}

@keyframes pulse-red {
  0%,
  100% {
    box-shadow: 0 0 0 0 rgba(248, 113, 113, 0.4);
  }
  50% {
    box-shadow: 0 0 0 8px rgba(248, 113, 113, 0);
  }
}

@keyframes pulse-yellow {
  0%,
  100% {
    box-shadow: 0 0 0 0 rgba(251, 191, 36, 0.4);
  }
  50% {
    box-shadow: 0 0 0 8px rgba(251, 191, 36, 0);
  }
}

@media (min-width: 1920px) {
  .grid {
    grid-template-columns: repeat(4, minmax(0, 1fr));
  }
}
</style>
