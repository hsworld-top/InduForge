<template>
  <div class="h-full w-full flex-1 flex flex-col relative dashboard-tabs-area" :class="{ 'has-hamburger': sidebarCollapsed && !isTabMaximized }">
    <!-- 汉堡菜单按钮 (绝对定位在 Tabs 左侧) -->
    <div 
      v-if="sidebarCollapsed && !isTabMaximized" 
      class="absolute left-0 top-0 h-[32px] w-[44px] flex items-center justify-center z-10 border-r border-transparent"
    >
      <button 
        @click="openSidebar" 
        class="p-1.5 rounded-lg text-gray-500 hover:text-gray-900 dark:text-gray-400 dark:hover:text-gray-100 hover:bg-gray-200/50 dark:hover:bg-gray-700/50 transition-all cursor-pointer"
      >
        <el-icon class="text-[18px]">
          <Expand />
        </el-icon>
      </button>
    </div>

    <el-tabs 
      v-model="internalActiveTab" 
      @tab-remove="handleCloseTab"
      :class="['dashboard-tabs h-full flex-1 min-h-0', isTabMaximized ? 'dashboard-tabs-maximized' : '']"
    >
      <el-tab-pane
        v-for="tab in tabs"
        :key="tab.key"
        :name="tab.key"
        :v-show="isTabVisible(tab.key)"
        :closable="tab.key !== 'dashboard'"
      >
        <template #label>
          <div class="flex items-center space-x-2 tab-label-content">
            <span class="truncate max-w-[150px]">{{ getTabTitle(tab) }}</span>
            <el-button v-if="tab.props?.appType" size="small" text circle class="!p-0 !w-5 !h-5 !ml-2 opacity-60 hover:opacity-100 transition-opacity"
              @click.stop="$emit('open-external-tab', tab)">
              <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                  d="M14 3h7v7m0-7L10 14m-4 7h11a2 2 0 002-2V8" />
              </svg>
            </el-button>
            <el-button v-if="!isTabMaximized && tab.key !== 'dashboard'" size="small" text circle
              class="!p-0 !w-5 !h-5 !ml-1 opacity-60 hover:opacity-100 transition-opacity" @click.stop="$emit('maximize-tab', tab.key)">
              <el-icon class="text-xs">
                <FullScreen />
              </el-icon>
            </el-button>
          </div>
        </template>
        <div class="h-full overflow-hidden">
          <component
            :is="tab.component"
            :tab-key="tab.key"
            @open-tab="$emit('open-tab', $event)"
            @embedded-register="$emit('embedded-register', $event)"
            @embedded-unregister="$emit('embedded-unregister', $event)"
            v-bind="tab.props"
          />
        </div>
      </el-tab-pane>
    </el-tabs>

    <Transition name="drop-down">
      <div
        v-if="isTabMaximized && showMaximizeRestoreButton"
        class="maximize-restore-anchor"
      >
        <el-button
          class="maximize-restore-floating-button"
          circle
          @click="$emit('restore-tab')"
        >
          <el-icon>
            <Close />
          </el-icon>
        </el-button>
      </div>
    </Transition>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { useAppStore } from '@/store'
import { FullScreen, Close, Expand } from '@element-plus/icons-vue'

const props = defineProps({
  tabs: {
    type: Array,
    required: true
  },
  activeTab: {
    type: String,
    required: true
  },
  isTabMaximized: {
    type: Boolean,
    default: false
  },
  showMaximizeRestoreButton: {
    type: Boolean,
    default: false
  },
  isTabVisible: {
    type: Function,
    required: true
  },
  getTabTitle: {
    type: Function,
    required: true
  }
})

const emit = defineEmits([
  'update:activeTab',
  'close-tab',
  'open-external-tab',
  'maximize-tab',
  'restore-tab',
  'open-tab',
  'embedded-register',
  'embedded-unregister'
])

const appStore = useAppStore()
const sidebarCollapsed = computed(() => appStore.sidebarCollapsed)

const internalActiveTab = computed({
  get: () => props.activeTab,
  set: (val) => emit('update:activeTab', val)
})

const openSidebar = () => {
  appStore.setSidebarCollapsed(false)
}

const handleCloseTab = (targetName) => {
  emit('close-tab', targetName)
}
</script>

<style>
/* Dashboard Tabs 极简浮岛风格优化 */

/* 为汉堡菜单预留空间 */
.dashboard-tabs-area.has-hamburger .dashboard-tabs .el-tabs__header {
  padding-left: 44px;
}

/* 顶部标签区域背景：轻微降低层级感，高度更紧凑 */
.dashboard-tabs .el-tabs__header {
  margin-bottom: 0 !important;
  padding: 2px 6px;
  background-color: var(--el-bg-color-page);
  border-bottom: 1px solid var(--el-border-color-light) !important;
}

/* 修复 Element Plus Tabs 内容区域无法撑满高度的问题 */
.dashboard-tabs {
  display: flex !important;
  flex-direction: column !important;
}
.dashboard-tabs .el-tabs__content {
  flex: 1 !important;
  min-height: 0 !important;
  padding: 0 !important;
}
.dashboard-tabs .el-tab-pane {
  height: 100% !important;
}

/* 去除默认底线和游标 */
.dashboard-tabs .el-tabs__nav-wrap::after,
.dashboard-tabs .el-tabs__active-bar {
  display: none !important;
}

/* 标签列表排版 */
.dashboard-tabs .el-tabs__nav {
  gap: 6px;
  border: none !important;
}

/* 默认状态：透明无边框的圆角按钮 */
.dashboard-tabs .el-tabs__item {
  height: 28px !important;
  line-height: 28px !important;
  padding: 0 10px !important;
  border-radius: 6px !important;
  color: var(--el-text-color-regular) !important;
  font-weight: 500 !important;
  font-size: 12px !important;
  transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1) !important;
  border: 1px solid transparent !important;
}

/* 悬停状态：微妙背景加深 */
.dashboard-tabs .el-tabs__item:hover {
  background-color: var(--el-fill-color) !important;
  color: var(--el-text-color-primary) !important;
}

/* 激活状态：变成一个立体悬浮的卡片 (Pill) */
.dashboard-tabs .el-tabs__item.is-active {
  background-color: var(--el-bg-color) !important;
  color: var(--el-color-primary) !important;
  box-shadow: 0 1px 3px rgba(0,0,0,0.04), 0 1px 2px rgba(0,0,0,0.02) !important;
  border-color: var(--el-border-color-lighter) !important;
}

/* 内置组件：调整关闭按钮样式 */
.dashboard-tabs .el-tabs__item .is-icon-close {
  width: 14px !important;
  height: 14px !important;
  line-height: 14px !important;
  margin-left: 6px !important;
  margin-right: -2px !important;
  transition: all 0.2s;
  border-radius: 4px;
}
.dashboard-tabs .el-tabs__item .is-icon-close:hover {
  background-color: var(--el-fill-color-dark);
  color: var(--el-text-color-primary);
}

/* Flex对齐辅助 */
.dashboard-tabs .tab-label-content {
  display: inline-flex;
  align-items: center;
}
</style>
