<template>
  <div class="h-full flex flex-col relative dashboard-tabs-area" :class="{ 'has-hamburger': sidebarCollapsed && !isTabMaximized }">
    <!-- 汉堡菜单按钮 (绝对定位在 Tabs 左侧) -->
    <div 
      v-if="sidebarCollapsed && !isTabMaximized" 
      class="absolute left-0 top-0 h-[40px] w-[48px] flex items-center justify-center z-10"
    >
      <button 
        @click="openSidebar" 
        class="p-1.5 rounded-md text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors"
      >
        <svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h16M4 18h16" />
        </svg>
      </button>
    </div>

    <el-tabs 
      v-model="internalActiveTab" 
      type="card" 
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
          <div class="flex items-center space-x-2">
            <span>{{ getTabTitle(tab) }}</span>
            <el-button v-if="tab.props?.appType" size="small" text circle class="!p-0 !w-4 !h-4"
              @click.stop="$emit('open-external-tab', tab)">
              <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                  d="M14 3h7v7m0-7L10 14m-4 7h11a2 2 0 002-2V8" />
              </svg>
            </el-button>
            <el-button v-if="!isTabMaximized && tab.key !== 'dashboard'" size="small" text circle
              class="!p-0 !w-4 !h-4" @click.stop="$emit('maximize-tab', tab.key)">
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
import { FullScreen, Close } from '@element-plus/icons-vue'

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
/* 当带有汉堡菜单时，给 el-tabs 的 header 部分增加左内边距，以免标签被遮挡 */
.dashboard-tabs-area.has-hamburger .el-tabs__header {
  padding-left: 48px;
}
/* 移除 el-tabs 的底部 margin 和强制边框，使其更干净 */
.dashboard-tabs-area .el-tabs__header {
  margin-bottom: 0 !important;
  background-color: var(--el-bg-color);
}
</style>
