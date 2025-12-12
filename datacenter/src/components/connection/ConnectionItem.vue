<template>
  <div
    :class="[
      'connection-item p-3 rounded-lg transition-colors',
      isSelected
        ? 'bg-blue-100 dark:bg-blue-900/30 border-blue-300 dark:border-blue-600'
        : 'hover:bg-gray-100 dark:hover:bg-gray-700'
    ]"
  >
    <!-- 连接图标和信息（可点击区域） -->
    <div 
      class="flex items-center justify-between cursor-pointer"
      @click="handleClick"
      @dblclick.stop="handleDblClick"
      @contextmenu.prevent="handleContextMenu"
    >
      <div class="flex items-center flex-1">
        <div class="w-6 h-6 mr-3 flex-shrink-0">
          <IconTablerDatabase class="w-6 h-6 text-blue-500" />
        </div>

        <!-- 连接信息 -->
        <div class="flex-1 min-w-0">
          <div class="text-sm font-medium text-gray-900 dark:text-white truncate">
            {{ connection.name }}
          </div>
          <div class="text-xs text-gray-500 dark:text-gray-400">
            {{ typeLabel }}
          </div>
        </div>
      </div>

      <!-- 状态指示器 -->
      <div class="flex-shrink-0 flex items-center space-x-1">
        <StatusIndicator :status="connection.status" />
        <component 
          v-if="connection.type === 'relational'" 
          :is="isExpanded ? IconTablerChevronDown : IconTablerChevronRight" 
          class="text-gray-400 w-4 h-4"
        />
      </div>
    </div>

    <!-- 连接详情 -->
    <div
      v-if="connection.type === 'relational' && connection.relationalConfig"
      class="mt-2 text-xs text-gray-500 dark:text-gray-400 cursor-pointer"
      @click="handleClick"
      @dblclick.stop="handleDblClick"
    >
      {{ connection.relationalConfig.dbType }} - {{ connection.relationalConfig.host }}:{{
        connection.relationalConfig.port
      }}
    </div>

    <!-- 展开内容（表和查询列表） - 不响应点击事件 -->
    <slot name="expanded" v-if="isExpanded"></slot>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import IconTablerDatabase from '~icons/tabler/database'
import IconTablerChevronRight from '~icons/tabler/chevron-right'
import IconTablerChevronDown from '~icons/tabler/chevron-down'
import StatusIndicator from '@/components/shared/StatusIndicator.vue'
import { getConnectionTypeConfig } from '@/config/connectionTypes'

const props = defineProps({
  connection: {
    type: Object,
    required: true
  },
  isSelected: {
    type: Boolean,
    default: false
  },
  isExpanded: {
    type: Boolean,
    default: false
  }
})

const emit = defineEmits(['click', 'dblclick', 'contextmenu'])

const typeLabel = computed(() => {
  if (props.connection.type === 'relational' && props.connection.relationalConfig) {
    const dbType = props.connection.relationalConfig.dbType
    const config = getConnectionTypeConfig(dbType)
    return config ? config.label : dbType
  }
  return props.connection.type
})

const handleClick = () => {
  emit('click', props.connection)
}

const handleDblClick = () => {
  emit('dblclick', props.connection)
}

const handleContextMenu = (event) => {
  emit('contextmenu', event, props.connection)
}
</script>

<style scoped>
.connection-item {
  border-radius: 8px;
  transition: all 0.2s ease;
}

.connection-item:hover {
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
}
</style>
