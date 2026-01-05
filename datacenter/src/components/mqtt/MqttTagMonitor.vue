<template>
  <div class="mqtt-tag-monitor h-full flex flex-col">
    <!-- 头部工具栏 -->
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
            行展示
          </el-button>
          <el-button
            size="small"
            :type="viewMode === 'card' ? 'primary' : 'default'"
            @click="viewMode = 'card'"
          >
            卡片展示
          </el-button>
        </div>
        <el-tooltip content="重新加载变量配置" placement="top">
          <el-button size="small" @click="handleRefresh">
            <IconTablerRefresh class="mr-1 w-4 h-4" />
            重载配置
          </el-button>
        </el-tooltip>
      </div>
    </div>

    <!-- Tag监控面板 -->
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
          <!-- Tag名称和标识符 -->
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

          <!-- Tag值显示 -->
          <div class="tag-value mt-2 p-3 bg-white rounded-lg border border-gray-200">
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

          <!-- Tag元数据 -->
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
        <div
          v-for="tag in enabledTags"
          :key="tag.id"
          class="tag-row"
        >
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

<script setup>
import { ref, computed, onMounted, onBeforeUnmount, watch } from "vue";
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
});

const tags = ref([]);
const viewMode = ref("list");

// 只显示启用的Tag
const enabledTags = computed(() => {
  return tags.value.filter((tag) => tag.isEnabled);
});

const {
  socket,
  connected: socketConnected,
  connect,
  disconnect,
  onMessage,
} = useMqttSocket(props.projectId);

const subscribedTagIds = ref(new Set());

// Tag 同步管理（监听左侧变化）
const { subscribe: subscribeTagSync, unsubscribe: unsubscribeTagSync } =
  useMqttTagSync(props.subscriptionId);

// 获取Tag列表
const fetchTags = async () => {
  try {
    const res = await getMqttTags(props.projectId, props.subscriptionId);
    // 后端返回: { success: true, data: tags数组, pagination }
    tags.value = res.data || [];
  } catch (error) {
    ElMessage.error("获取变量列表失败: " + error.message);
  }
};

// 刷新
const handleRefresh = () => {
  fetchTags();
};

// 格式化值
const formatValue = (value, dataType) => {
  if (value === null || value === undefined) return "-";

  try {
    if (dataType === "object" || dataType === "array") {
      const parsed = JSON.parse(value);
      return JSON.stringify(parsed, null, 2);
    }
    if (dataType === "number") {
      const num = parseFloat(value);
      return isNaN(num) ? value : num.toFixed(2);
    }
    return value;
  } catch {
    return value;
  }
};

// 格式化时间戳
const formatTimestamp = (timestamp) => {
  const date = dayjs(timestamp);
  return date.isValid() ? date.format(TIME_FORMAT) : "-";
};

// 数据类型标签
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

// 解析类型标签
const getParseTypeLabel = (parseType) => {
  const labels = {
    jsonpath: "JSONPath",
    regex: "正则",
    script: "脚本",
    fixed: "固定值",
  };
  return labels[parseType] || parseType;
};

// 数据质量标签
const getQualityLabel = (quality) => {
  const labels = {
    good: "良好",
    bad: "错误",
    uncertain: "不确定",
    unknown: "未知",
  };
  return labels[quality] || quality;
};

// 数据质量颜色
const getQualityColor = (quality) => {
  const colors = {
    good: "success",
    bad: "danger",
    uncertain: "warning",
    unknown: "info",
  };
  return colors[quality] || "info";
};

// 处理Tag值更新
const handleTagValueUpdate = (data) => {
  const tag = tags.value.find((t) => t.id === data.tagId);
  if (tag) {
    tag.currentValue = {
      parsedValue: data.value,
      quality: data.quality,
      timestamp: data.timestamp,
      error: data.error,
    };
  }
};

// 处理左侧Tag变化事件
const handleTagSyncEvent = async (event) => {
  console.log("[MqttTagMonitor] Received sync event:", event);

  switch (event.type) {
    case "created":
      // 新建Tag，重新加载列表
      await fetchTags();
      break;

    case "updated":
      // 更新Tag，重新加载列表
      await fetchTags();
      break;

    case "deleted":
      // 删除Tag，从列表中移除
      tags.value = tags.value.filter((t) => t.id !== event.data.tagId);
      break;

    case "toggled": {
      // 切换启用状态
      const tag = tags.value.find((t) => t.id === event.data.tagId);
      if (tag) {
        tag.isEnabled = event.data.isEnabled;

        // 如果禁用了Tag，取消订阅Socket.IO
        if (!tag.isEnabled && socket.value) {
          socket.value.emit("mqtt:tag:unsubscribe", { tagId: tag.id });
        }
        // 如果启用了Tag，订阅Socket.IO
        else if (tag.isEnabled && socket.value && socketConnected.value) {
          syncSubscriptions();
        }
      }
      break;
    }

    case "refresh":
      // 批量刷新
      await fetchTags();
      break;
  }
};

/**
 * 同步订阅列表，避免重复订阅
 */
const syncSubscriptions = () => {
  if (!socket.value || !socketConnected.value) {
    return;
  }

  const desiredIds = new Set(enabledTags.value.map((tag) => tag.id));

  desiredIds.forEach((tagId) => {
    if (!subscribedTagIds.value.has(tagId)) {
      socket.value.emit("mqtt:tag:subscribe", { tagId });
      subscribedTagIds.value.add(tagId);
    }
  });

  subscribedTagIds.value.forEach((tagId) => {
    if (!desiredIds.has(tagId)) {
      socket.value.emit("mqtt:tag:unsubscribe", { tagId });
      subscribedTagIds.value.delete(tagId);
    }
  });
};

onMounted(async () => {
  await fetchTags();

  // 订阅左侧Tag变化事件
  subscribeTagSync(handleTagSyncEvent);

  // 连接Socket.IO
  connect(props.projectId);

  // 监听Tag值更新消息
  const unsubscribeMessage = onMessage((data) => {
    console.log("[MqttTagMonitor] Received message:", data);
    if (data.tagId) {
      // 这是Tag值更新消息
      console.log("[MqttTagMonitor] Processing tag value update:", data);
      handleTagValueUpdate(data);
    }
  });

  const stopWatch = watch(
    () => [socketConnected.value, enabledTags.value.map((tag) => tag.id).join(",")],
    () => {
      if (!socketConnected.value) {
        subscribedTagIds.value.clear();
        return;
      }
      syncSubscriptions();
    },
    { immediate: true }
  );

  const cleanup = () => {
    stopWatch();
    unsubscribeMessage();
  };

  onBeforeUnmount(cleanup);
});

onBeforeUnmount(() => {
  // 取消订阅左侧Tag变化事件
  unsubscribeTagSync(handleTagSyncEvent);

  // 取消订阅Socket.IO
  if (socket.value && socketConnected.value) {
    subscribedTagIds.value.forEach((tagId) => {
      socket.value.emit("mqtt:tag:unsubscribe", { tagId });
    });
  }

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

/* 质量状态边框颜色动画 */
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

/* 响应式网格布局优化 */
@media (min-width: 1920px) {
  .grid {
    grid-template-columns: repeat(4, minmax(0, 1fr));
  }
}
</style>
