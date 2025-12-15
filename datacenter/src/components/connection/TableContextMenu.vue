<!--
  表右键菜单组件
  显示表相关的操作选项
-->
<template>
  <Teleport to="body">
    <div
      v-if="visible"
      class="fixed inset-0 z-50"
      @click="handleClose"
      @contextmenu.prevent
    >
      <div
        class="absolute bg-white dark:bg-gray-800 rounded-lg shadow-lg border border-gray-200 dark:border-gray-700 py-1 min-w-[160px]"
        :style="{ left: position.x + 'px', top: position.y + 'px' }"
        @click.stop
      >
        <div
          class="px-4 py-2 hover:bg-gray-100 dark:hover:bg-gray-700 cursor-pointer flex items-center space-x-2 text-sm"
          @click="handleViewStructure"
        >
          <IconTablerTable class="w-4 h-4 text-blue-500" />
          <span>查看表结构</span>
        </div>
        <div
          class="px-4 py-2 hover:bg-gray-100 dark:hover:bg-gray-700 cursor-pointer flex items-center space-x-2 text-sm"
          @click="handleQueryTable"
        >
          <IconTablerCode class="w-4 h-4 text-green-500" />
          <span>查询表数据</span>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup>
import { computed } from 'vue'
import IconTablerTable from '~icons/tabler/table'
import IconTablerCode from '~icons/tabler/code'

const props = defineProps({
  visible: {
    type: Boolean,
    default: false
  },
  position: {
    type: Object,
    default: () => ({ x: 0, y: 0 })
  },
  connection: {
    type: Object,
    default: null
  },
  table: {
    type: Object,
    default: null
  }
})

const emit = defineEmits(['update:visible', 'view-structure', 'query-table'])

const handleClose = () => {
  emit('update:visible', false)
}

const handleViewStructure = () => {
  emit('view-structure', props.connection, props.table)
  handleClose()
}

const handleQueryTable = () => {
  emit('query-table', props.connection, props.table)
  handleClose()
}
</script>

<style scoped>
/* 菜单动画 */
.fixed {
  animation: fadeIn 0.15s ease-out;
}

@keyframes fadeIn {
  from {
    opacity: 0;
    transform: scale(0.95);
  }
  to {
    opacity: 1;
    transform: scale(1);
  }
}
</style>
