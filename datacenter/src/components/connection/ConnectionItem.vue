<template>
  <el-tooltip :content="detailText" placement="right" :disabled="!detailText">
    <div
      :class="[
        'connection-item transition-colors',
        isSelected ? 'is-selected' : 'hover:bg-gray-100 dark:hover:bg-gray-700',
        isExpanded ? 'is-expanded' : '',
      ]"
    >
      <!-- 连接图标和信息（可点击区域） -->
      <div
        class="flex items-center justify-between cursor-pointer"
        @click="handleClick"
        @dblclick.stop="handleDblClick"
        @contextmenu.prevent="handleContextMenu"
      >
        <div class="flex items-center flex-1 min-w-0">
          <div class="w-5 h-5 mr-2 flex-shrink-0">
            <IconTablerDatabase
              v-if="connection.type === 'relational'"
              class="w-5 h-5 text-blue-500"
            />
            <IconTablerCloudDataConnection
              v-else-if="connection.type === 'mqtt'"
              class="w-5 h-5 text-green-500"
            />
            <IconTablerDatabase v-else class="w-5 h-5 text-gray-500" />
          </div>

          <!-- 连接信息 -->
          <div class="flex-1 min-w-0">
            <div
              :class="[
                'text-sm truncate',
                isSelected
                  ? 'font-semibold text-indigo-700 dark:text-indigo-200'
                  : 'font-medium text-gray-900 dark:text-white',
              ]"
            >
              {{ connection.name }}
            </div>
            <div class="text-xs text-gray-500 dark:text-gray-400 truncate">
              {{ typeLabel }}
            </div>
          </div>
        </div>

        <!-- 状态指示器 -->
        <div class="flex-shrink-0 flex items-center space-x-1">
          <StatusIndicator :status="connection.status" />
          <component
            v-if="
              connection.type === 'relational' || connection.type === 'mqtt'
            "
            :is="isExpanded ? IconTablerChevronDown : IconTablerChevronRight"
            class="expand-icon text-gray-400 w-4 h-4"
          />
        </div>
      </div>

      <!-- 展开内容（表和查询列表） - 不响应点击事件 -->
      <slot name="expanded" v-if="isExpanded"></slot>
    </div>
  </el-tooltip>
</template>

<script setup>
import { computed } from "vue";
import IconTablerDatabase from "~icons/tabler/database";
import IconTablerCloudDataConnection from "~icons/tabler/cloud-data-connection";
import IconTablerChevronRight from "~icons/tabler/chevron-right";
import IconTablerChevronDown from "~icons/tabler/chevron-down";
import StatusIndicator from "@/components/shared/StatusIndicator.vue";
import { getConnectionTypeConfig } from "@/config/connectionTypes";

const props = defineProps({
  connection: {
    type: Object,
    required: true,
  },
  isSelected: {
    type: Boolean,
    default: false,
  },
  isExpanded: {
    type: Boolean,
    default: false,
  },
});

const emit = defineEmits(["click", "dblclick", "contextmenu"]);

const typeLabel = computed(() => {
  if (
    props.connection.type === "relational" &&
    props.connection.relationalConfig
  ) {
    const dbType = props.connection.relationalConfig.dbType;
    const config = getConnectionTypeConfig(dbType);
    return config ? config.label : dbType;
  } else if (props.connection.type === "mqtt") {
    const config = getConnectionTypeConfig("mqtt");
    return config ? config.label : "MQTT";
  }
  return props.connection.type;
});

const detailText = computed(() => {
  if (
    props.connection.type === "relational" &&
    props.connection.relationalConfig
  ) {
    const config = props.connection.relationalConfig;
    return `${config.dbType} - ${config.host}:${config.port}`;
  }
  if (props.connection.type === "mqtt" && props.connection.mqttConfig) {
    const config = props.connection.mqttConfig;
    return `${config.protocol}://${config.brokerUrl}:${config.port || 1883}`;
  }
  return "";
});

const handleClick = () => {
  emit("click", props.connection);
};

const handleDblClick = () => {
  emit("dblclick", props.connection);
};

const handleContextMenu = (event) => {
  emit("contextmenu", event, props.connection);
};
</script>

<style scoped>
.connection-item {
  padding: 8px 10px;
  border-radius: 10px;
  transition: background-color 0.2s ease;
}

.connection-item.is-selected {
  background: #e0e7ff;
}

.dark .connection-item.is-selected {
  background: rgba(79, 70, 229, 0.28);
}

.expand-icon {
  opacity: 0;
  transition: opacity 0.2s ease;
}

.connection-item:hover .expand-icon,
.connection-item.is-expanded .expand-icon {
  opacity: 1;
}
</style>
