<template>
  <div
    class="dashboard"
    :class="{ 'dashboard-maximized': isTabMaximized }"
    @mousemove="handleMaximizedMouseMove"
    @mouseleave="handleMaximizedMouseLeave"
  >
    <!-- 页面头部 -->
    <div v-if="!isTabMaximized"
      class="bg-white dark:bg-gray-800 shadow-sm border-b border-gray-200 dark:border-gray-700 h-12 px-4">
      <div class="flex items-center justify-between h-full">
        <!-- Logo 图片 -->
        <div class="flex items-center">
          <img :src="tenantLogoUrl" :alt="currentTenant?.name || 'Logo'" class="h-7 w-auto mr-2" />
          <h1 class="text-base font-semibold text-gray-800 dark:text-white hidden sm:block">
            {{ currentTenant?.name || 'InduForge' }}
          </h1>
        </div>
        <div class="flex items-center space-x-2">
          <!-- 语言切换 -->
          <div class="relative language-menu">
            <button @click="showLanguageMenu = !showLanguageMenu"
              class="h-8 w-8 flex items-center justify-center rounded-md text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-700">
              <svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                  d="M3 5h12M9 3v2m1.048 9.5A18.022 18.022 0 016.412 9m6.088 9h7M11 21l5-10 5 10M12.751 5C11.783 10.77 8.07 15.61 3 18.129" />
              </svg>
            </button>

            <!-- 语言选择下拉菜单 -->
            <div v-if="showLanguageMenu"
              class="absolute right-0 mt-2 w-32 bg-white dark:bg-gray-800 rounded-lg shadow-lg border border-gray-200 dark:border-gray-700 py-1 z-10">
              <button @click="changeLanguage('zh')" :class="[
                'block w-full text-left px-4 py-2 text-sm hover:bg-gray-100 dark:hover:bg-gray-700',
                currentLanguage === 'zh'
                  ? 'text-blue-600 dark:text-blue-400 bg-blue-50 dark:bg-blue-900/20'
                  : 'text-gray-700 dark:text-gray-300',
              ]">
                {{ t('system.languageZh') }}
              </button>
              <button @click="changeLanguage('en')" :class="[
                'block w-full text-left px-4 py-2 text-sm hover:bg-gray-100 dark:hover:bg-gray-700',
                currentLanguage === 'en'
                  ? 'text-blue-600 dark:text-blue-400 bg-blue-50 dark:bg-blue-900/20'
                  : 'text-gray-700 dark:text-gray-300',
              ]">
                {{ t('system.languageEn') }}
              </button>
            </div>
          </div>

          <!-- 主题切换 -->
          <button @click="toggleTheme"
            class="h-8 w-8 flex items-center justify-center rounded-md text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-700">
            <svg v-if="isDark" class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                d="M12 3v1m0 16v1m9-9h-1M4 12H3m15.364 6.364l-.707-.707M6.343 6.343l-.707-.707m12.728 0l-.707.707M6.343 17.657l-.707.707M16 12a4 4 0 11-8 0 4 4 0 018 0z" />
            </svg>
            <svg v-else class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                d="M20.354 15.354A9 9 0 018.646 3.646 9.003 9.003 0 0012 21a9.003 9.003 0 008.354-5.646z" />
            </svg>
          </button>

          <div class="h-4 w-px bg-gray-200 dark:bg-gray-700"></div>

          <!-- 用户菜单 -->
          <div class="relative user-menu">
            <button @click="showUserMenu = !showUserMenu"
              class="flex items-center gap-2 h-8 px-2 rounded-md text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700">
              <img v-if="userAvatarUrl" :src="userAvatarUrl" alt="avatar" class="w-6 h-6 rounded-full object-cover" />
              <div v-else
                class="w-6 h-6 bg-blue-600 rounded-full flex items-center justify-center text-white text-xs font-medium">
                {{ userInitials }}
              </div>
              <span class="text-xs text-gray-600 dark:text-gray-400">{{ username }}</span>
            </button>

            <!-- 下拉菜单 -->
            <div v-if="showUserMenu"
              class="absolute right-0 mt-2 w-48 bg-white dark:bg-gray-800 rounded-lg shadow-lg border border-gray-200 dark:border-gray-700 py-1 z-10">
              <button @click="openProfileDialog"
                class="block w-full text-left px-4 py-2 text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700">
                {{ t('dashboard.profile') }}
              </button>
              <button v-if="isSystemAdmin" @click="openSystemSettingsDialog"
                class="block w-full text-left px-4 py-2 text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700">
                {{ t('dashboard.menuSettings') }}
              </button>
              <button @click="handleLogout"
                class="block w-full text-left px-4 py-2 text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700">
                {{ t('auth.logout') }}
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 主要内容区域 -->
    <div class="flex flex-1 overflow-hidden">
      <!-- 左侧菜单栏 -->
      <div v-if="!isTabMaximized" :class="[
        'sidebar-panel relative border-r transition-all duration-300 ease-[cubic-bezier(0.22,1,0.36,1)] flex flex-col',
        isDark ? 'bg-gray-900 border-gray-800' : 'bg-white border-gray-200',
        sidebarCollapsed ? 'w-10' : 'w-40',
      ]">
        <div :class="[sidebarCollapsed ? 'p-1.5' : 'p-2', 'flex-1 overflow-y-auto']">
          <nav class="space-y-2">
            <el-tooltip v-if="hasTabPermission('dashboard')" :content="t('dashboard.title')" placement="right" :show-after="500"
              :disabled="!sidebarCollapsed">
              <button @click="openTab('dashboard')" :class="[
                'w-full h-11 flex items-center rounded-lg transition-colors',
                sidebarCollapsed ? 'justify-center' : 'px-2 gap-2 justify-start',
              ]">
                <span :class="[
                  'w-9 h-9 rounded-lg flex items-center justify-center transition-colors',
                  activeTab === 'dashboard'
                    ? 'bg-blue-600 text-white'
                    : isDark
                      ? 'text-gray-400/80 hover:text-white hover:bg-gray-800'
                      : 'text-gray-500 hover:text-gray-700 hover:bg-gray-100',
                ]">
                  <svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                      d="M4 5a2 2 0 012-2h4a2 2 0 012 2v4a2 2 0 01-2 2H6a2 2 0 01-2-2V5z" />
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                      d="M12 5a2 2 0 012-2h4a2 2 0 012 2v4a2 2 0 01-2 2h-4a2 2 0 01-2-2V5zM4 15a2 2 0 012-2h4a2 2 0 012 2v4a2 2 0 01-2 2H6a2 2 0 01-2-2v-4zM12 15a2 2 0 012-2h4a2 2 0 012 2v4a2 2 0 01-2 2h-4a2 2 0 01-2-2v-4z" />
                  </svg>
                </span>
                <span :class="[
                  'sidebar-menu-label text-sm font-medium',
                  sidebarCollapsed ? 'sidebar-menu-label-collapsed' : 'sidebar-menu-label-expanded',
                  activeTab === 'dashboard' ? 'text-blue-600' : (isDark ? 'text-gray-200' : 'text-gray-700'),
                ]">
                  {{ t('dashboard.title') }}
                </span>
              </button>
            </el-tooltip>

            <el-tooltip v-if="hasTabPermission('tenant-management')" :content="t('dashboard.menuTenant')" placement="right" :show-after="500"
              :disabled="!sidebarCollapsed">
              <button @click="openTab('tenant-management')" :class="[
                'w-full h-11 flex items-center rounded-lg transition-colors',
                sidebarCollapsed ? 'justify-center' : 'px-2 gap-2 justify-start',
              ]">
                <span :class="[
                  'w-9 h-9 rounded-lg flex items-center justify-center transition-colors',
                  activeTab === 'tenant-management'
                    ? 'bg-blue-600 text-white'
                    : isDark
                      ? 'text-gray-400/80 hover:text-white hover:bg-gray-800'
                      : 'text-gray-500 hover:text-gray-700 hover:bg-gray-100',
                ]">
                  <svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                      d="M19 21V5a2 2 0 00-2-2H7a2 2 0 00-2 2v16m14 0h2m-2 0h-5m-9 0H3m2 0h5M9 7h1m-1 4h1m4-4h1m-1 4h1m-5 10v-5a1 1 0 011-1h2a1 1 0 011 1v5m-4 0h4" />
                  </svg>
                </span>
                <span :class="[
                  'sidebar-menu-label text-sm font-medium',
                  sidebarCollapsed ? 'sidebar-menu-label-collapsed' : 'sidebar-menu-label-expanded',
                  activeTab === 'tenant-management' ? 'text-blue-600' : (isDark ? 'text-gray-200' : 'text-gray-700'),
                ]">
                  {{ t('dashboard.menuTenant') }}
                </span>
              </button>
            </el-tooltip>

            <el-tooltip v-if="hasTabPermission('user-management')" :content="t('dashboard.menuUser')"
              placement="right" :show-after="500" :disabled="!sidebarCollapsed">
              <button @click="openTab('user-management')" :class="[
                'w-full h-11 flex items-center rounded-lg transition-colors',
                sidebarCollapsed ? 'justify-center' : 'px-2 gap-2 justify-start',
              ]">
                <span :class="[
                  'w-9 h-9 rounded-lg flex items-center justify-center transition-colors',
                  activeTab === 'user-management'
                    ? 'bg-blue-600 text-white'
                    : isDark
                      ? 'text-gray-400/80 hover:text-white hover:bg-gray-800'
                      : 'text-gray-500 hover:text-gray-700 hover:bg-gray-100',
                ]">
                  <el-icon class="w-5 h-5">
                    <UserFilled />
                  </el-icon>
                </span>
                <span :class="[
                  'sidebar-menu-label text-sm font-medium',
                  sidebarCollapsed ? 'sidebar-menu-label-collapsed' : 'sidebar-menu-label-expanded',
                  activeTab === 'user-management' ? 'text-blue-600' : (isDark ? 'text-gray-200' : 'text-gray-700'),
                ]">
                  {{ t('dashboard.menuUser') }}
                </span>
              </button>
            </el-tooltip>

            <el-tooltip v-if="hasTabPermission('project-management')" :content="t('dashboard.menuProject')"
              placement="right" :show-after="500" :disabled="!sidebarCollapsed">
              <button @click="openTab('project-management')" :class="[
                'w-full h-11 flex items-center rounded-lg transition-colors',
                sidebarCollapsed ? 'justify-center' : 'px-2 gap-2 justify-start',
              ]">
                <span :class="[
                  'w-9 h-9 rounded-lg flex items-center justify-center transition-colors',
                  activeTab === 'project-management'
                    ? 'bg-blue-600 text-white'
                    : isDark
                      ? 'text-gray-400/80 hover:text-white hover:bg-gray-800'
                      : 'text-gray-500 hover:text-gray-700 hover:bg-gray-100',
                ]">
                  <svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                      d="M3 7a2 2 0 012-2h4l2 2h8a2 2 0 012 2v8a2 2 0 01-2 2H5a2 2 0 01-2-2V7z" />
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 11h18" />
                  </svg>
                </span>
                <span :class="[
                  'sidebar-menu-label text-sm font-medium',
                  sidebarCollapsed ? 'sidebar-menu-label-collapsed' : 'sidebar-menu-label-expanded',
                  activeTab === 'project-management' ? 'text-blue-600' : (isDark ? 'text-gray-200' : 'text-gray-700'),
                ]">
                  {{ t('dashboard.menuProject') }}
                </span>
              </button>
            </el-tooltip>

            <el-tooltip v-if="hasTabPermission('ops-management')" :content="t('dashboard.menuOps')"
              placement="right" :show-after="500" :disabled="!sidebarCollapsed">
              <button @click="openTab('ops-management')" :class="[
                'w-full h-11 flex items-center rounded-lg transition-colors',
                sidebarCollapsed ? 'justify-center' : 'px-2 gap-2 justify-start',
              ]">
                <span :class="[
                  'w-9 h-9 rounded-lg flex items-center justify-center transition-colors',
                  activeTab === 'ops-management'
                    ? 'bg-blue-600 text-white'
                    : isDark
                      ? 'text-gray-400/80 hover:text-white hover:bg-gray-800'
                      : 'text-gray-500 hover:text-gray-700 hover:bg-gray-100',
                ]">
                  <svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                      d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                      d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                  </svg>
                </span>
                <span :class="[
                  'sidebar-menu-label text-sm font-medium',
                  sidebarCollapsed ? 'sidebar-menu-label-collapsed' : 'sidebar-menu-label-expanded',
                  activeTab === 'ops-management' ? 'text-blue-600' : (isDark ? 'text-gray-200' : 'text-gray-700'),
                ]">
                  {{ t('dashboard.menuOps') }}
                </span>
              </button>
            </el-tooltip>

            <el-tooltip v-if="hasTabPermission('system-logs')" :content="t('dashboard.menuLogs')"
              placement="right" :show-after="500" :disabled="!sidebarCollapsed">
              <button @click="openTab('system-logs')" :class="[
                'w-full h-11 flex items-center rounded-lg transition-colors',
                sidebarCollapsed ? 'justify-center' : 'px-2 gap-2 justify-start',
              ]">
                <span :class="[
                  'w-9 h-9 rounded-lg flex items-center justify-center transition-colors',
                  activeTab === 'system-logs'
                    ? 'bg-blue-600 text-white'
                    : isDark
                      ? 'text-gray-400/80 hover:text-white hover:bg-gray-800'
                      : 'text-gray-500 hover:text-gray-700 hover:bg-gray-100',
                ]">
                  <svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                      d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2" />
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                      d="M9 3h6a1 1 0 011 1v2H8V4a1 1 0 011-1zM9 11h6M9 15h6" />
                  </svg>
                </span>
                <span :class="[
                  'sidebar-menu-label text-sm font-medium',
                  sidebarCollapsed ? 'sidebar-menu-label-collapsed' : 'sidebar-menu-label-expanded',
                  activeTab === 'system-logs' ? 'text-blue-600' : (isDark ? 'text-gray-200' : 'text-gray-700'),
                ]">
                  {{ t('dashboard.menuLogs') }}
                </span>
              </button>
            </el-tooltip>

          </nav>
        </div>
        <button @click="toggleSidebar" :class="[
          'sidebar-toggle-button absolute z-30 top-1/2 -right-3 -translate-y-1/2 h-9 w-6 rounded-full border shadow-sm flex items-center justify-center transition-all duration-200',
          isDark
            ? 'bg-gray-900 border-gray-700 text-gray-400 hover:text-white hover:bg-gray-800 hover:border-gray-600'
            : 'bg-white border-gray-200 text-gray-500 hover:text-gray-800 hover:bg-gray-50 hover:border-gray-300',
        ]" aria-label="Toggle sidebar">
          <svg :class="[
            'w-3.5 h-3.5 transition-transform duration-200 ease-out',
            sidebarCollapsed ? 'rotate-180' : '',
          ]" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 6l-6 6 6 6" />
          </svg>
        </button>
      </div>

      <!-- 右侧标签页区域 -->
      <div :class="['flex-1 overflow-hidden relative', isTabMaximized ? '' : 'pl-3']">
        <div v-if="tabs.length > 0" class="h-full flex flex-col">
          <el-tabs v-model="activeTab" type="card" :closable="(tab) => tab.key !== 'dashboard'" @tab-remove="closeTab"
            :class="['dashboard-tabs h-full flex-1 min-h-0', isTabMaximized ? 'dashboard-tabs-maximized' : '']">
            <el-tab-pane v-for="tab in tabs" :key="tab.key" :name="tab.key" :v-show="isTabVisible(tab.key)">
              <template #label>
                <div class="flex items-center space-x-2">
                  <span>{{ getTabTitle(tab) }}</span>
                  <el-button v-if="tab.props?.appType" size="small" text circle class="!p-0 !w-4 !h-4"
                    @click.stop="openExternalTab(tab)">
                    <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                        d="M14 3h7v7m0-7L10 14m-4 7h11a2 2 0 002-2V8" />
                    </svg>
                  </el-button>
                  <el-button v-if="!isTabMaximized && tab.key !== 'dashboard'" size="small" text circle
                    class="!p-0 !w-4 !h-4" @click.stop="maximizeTab(tab.key)">
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
                  @open-tab="openTab"
                  @embedded-register="registerEmbeddedApp"
                  @embedded-unregister="unregisterEmbeddedApp"
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
                @click="restoreTab"
              >
                <el-icon>
                  <Close />
                </el-icon>
              </el-button>
            </div>
          </Transition>
        </div>
      </div>
    </div>

    <el-dialog v-model="showProfileDialog" width="760px" destroy-on-close append-to-body class="profile-dialog"
      :title="t('profile.title')">
      <Profile :embedded="true" />
    </el-dialog>

    <el-dialog v-model="showSystemSettingsDialog" width="960px" destroy-on-close append-to-body
      :title="t('dashboard.menuSettings')">
      <SystemSettings />
    </el-dialog>
  </div>
</template>

<script>
import { ref, computed, onMounted, onUnmounted, watch, defineAsyncComponent } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAuthStore, useAppStore, useTenantStore } from '@/store'
import { Storage } from '@/utils/storage'
import { ElMessage, ElMessageBox, ElNotification } from 'element-plus'
import { buildAppUrl } from '@/utils/appUrl'
import { resolveRestorePayload } from '@/utils/embeddedAppBridge'
import {
  createRestoredEmbeddedTab,
  EMBEDDED_APP_COMPONENT,
  extractDashboardHandoffId,
  stripDashboardHandoffQuery,
} from '@/utils/dashboardEntryHandoff'
import {
  createEmbeddedUpdateMessage,
  broadcastToEmbeddedIframes,
  handleEmbeddedWindowMessage,
  registerEmbeddedIframe,
  syncLocaleToEmbeddedIframes,
  unregisterEmbeddedIframe,
} from '@/utils/embeddedIframeSync'
import { resolveDashboardTabTitle } from '@/utils/dashboardTabTitle'
import { restoreDashboardTabState, serializeDashboardTabState } from '@/utils/dashboardTabState'
import { canAccessTab, getTabAccessDeniedMessage } from '@/permissions'
import { ROLES, STORAGE_KEYS } from '@/constants'
import { initSocket, getSocket } from '@/utils/socket'
import request from '@/utils/request'

// 标签页组件懒加载，提升首次加载速度
const DashboardContent = defineAsyncComponent(() => import('@/views/DashboardContent.vue'))
const TenantManagement = defineAsyncComponent(() => import('@/views/admin/TenantManagement.vue'))
const UserManagement = defineAsyncComponent(() => import('@/views/tenant/UserManagement.vue'))
const ProjectManagement = defineAsyncComponent(() => import('@/views/tenant/ProjectManagement.vue'))
const OpsManagement = defineAsyncComponent(() => import('@/views/tenant/OpsManagement.vue'))
const SystemLogs = defineAsyncComponent(() => import('@/views/tenant/SystemLogs.vue'))
const SystemSettings = defineAsyncComponent(() => import('@/views/tenant/SystemSettings.vue'))
const Profile = defineAsyncComponent(() => import('@/views/profile/Profile.vue'))
const EmbeddedApp = defineAsyncComponent(() => import('@/components/EmbeddedApp.vue'))

// 导入默认Logo图片
import defaultLogo from '@/assets/images/default-logo.svg'

export default {
  name: 'Dashboard',
  components: {
    Profile,
    SystemSettings,
  },
  setup() {
    const route = useRoute()
    const router = useRouter()
    const { locale, t } = useI18n()
    const authStore = useAuthStore()
    const appStore = useAppStore()
    const tenantStore = useTenantStore()

    const showUserMenu = ref(false)
    const showLanguageMenu = ref(false)
    const showProfileDialog = ref(false)
    const showSystemSettingsDialog = ref(false)
    const pendingRequestNotifications = new Map()
    const pendingPollingTimer = ref(null)
    const lastPendingCount = ref(0)
    const opsSocket = ref(null)
    const wsConnectCheckTimer = ref(null)
    const embeddedRegistry = new Map()

    // 侧边栏折叠状态（持久化）
    const sidebarCollapsed = computed({
      get: () => appStore.sidebarCollapsed,
      set: (value) => appStore.setSidebarCollapsed(value),
    })

    // 标签页状态
    const tabs = ref([])
    const activeTab = ref('')
    const isTabMaximized = ref(false)
    const showMaximizeRestoreButton = ref(false)
    const maximizeRestoreHideTimer = ref(null)
    const tabsInitialized = ref(false)

    // 标签页配置
    const getTabConfigMap = () => ({
      dashboard: {
        titleKey: 'dashboard.title',
        component: DashboardContent,
        icon: 'dashboard',
      },
      'tenant-management': {
        titleKey: 'dashboard.menuTenant',
        component: TenantManagement,
        icon: 'building',
      },
      'user-management': {
        titleKey: 'dashboard.menuUser',
        component: UserManagement,
        icon: 'users',
      },
      'project-management': {
        titleKey: 'dashboard.menuProject',
        component: ProjectManagement,
        icon: 'folder',
      },
      'ops-management': {
        titleKey: 'dashboard.menuOps',
        component: OpsManagement,
        icon: 'cog',
      },
      'system-logs': {
        titleKey: 'dashboard.menuLogs',
        component: SystemLogs,
        icon: 'clipboard',
      },
      'system-settings': {
        titleKey: 'dashboard.menuSettings',
        component: SystemSettings,
        icon: 'cog',
      },
    })

    const isDark = computed(() => appStore.isDark)
    const username = computed(() => authStore.userInfo?.username || '')
    const userInitials = computed(() => {
      const name = username.value || 'U'
      return name.charAt(0).toUpperCase()
    })
    const userAvatarUrl = computed(() => authStore.userInfo?.avatarUrl || '')
    const isAdmin = computed(() => authStore.isAdmin)
    const isSuperAdmin = computed(() => authStore.userInfo?.role === ROLES.SUPER_ADMIN)
    const isSystemAdmin = computed(() => authStore.userInfo?.role === ROLES.SYSTEM_ADMIN)
    const isOpsAdmin = computed(() => authStore.userInfo?.role === ROLES.OPS_ADMIN)
    const canReceiveNodePendingAlerts = computed(() => isSystemAdmin.value || isOpsAdmin.value)
    const isProjectAdmin = computed(() => authStore.userInfo?.role === ROLES.PROJECT_ADMIN)
    const isUserAdmin = computed(() => authStore.userInfo?.role === ROLES.USER_ADMIN)

    /**
     * 检查标签页访问权限。
     * @param {string} tabKey - 标签页 key
     * @returns {boolean} 是否允许访问
     */
    const hasTabPermission = (tabKey) => canAccessTab(tabKey, authStore.userInfo?.role)

    // 检查标签页是否可见
    const isTabVisible = (tabKey) => hasTabPermission(tabKey)

    // 监听超级管理员状态变化，如果不是超级管理员且当前激活的是租户管理，切换到dashboard
    watch(isSuperAdmin, (newVal) => {
      if (!newVal && activeTab.value === 'tenant-management') {
        activeTab.value = 'dashboard'
      }
    })

    /**
     * 通知嵌入应用更新主题。
     * @param {string} theme - 主题
     */
    const syncEmbeddedTheme = (theme) => {
      broadcastToEmbeddedIframes(
        embeddedRegistry.values(),
        createEmbeddedUpdateMessage('THEME_UPDATE', 'theme', theme)
      )
    }

    /**
     * 通知嵌入应用更新语言。
     * @param {string} localeValue - 语言
     */
    const syncEmbeddedLocale = (localeValue) => {
      syncLocaleToEmbeddedIframes(embeddedRegistry.values(), localeValue)
    }

    // 租户相关计算属性
    const currentTenant = computed(() => tenantStore.currentTenant)
    const tenantLogoUrl = computed(() => {
      // 优先使用租户的logo，然后使用平台默认logo
      const tenantLogo = currentTenant.value?.logoUrl
      if (tenantLogo) {
        // 如果是完整URL，直接使用；如果是相对路径，拼接public路径
        return tenantLogo.startsWith('http') ? tenantLogo : tenantLogo
      }
      return defaultLogo
    })

    const toggleTheme = () => {
      appStore.setTheme(isDark.value ? 'light' : 'dark')
    }

    const changeLanguage = (lang) => {
      locale.value = lang
      appStore.setLanguage(lang)
      showLanguageMenu.value = false
    }

    const currentLanguage = computed(() => appStore.language || 'zh')

    /**
     * 打开个人资料弹窗。
     */
    const openProfileDialog = () => {
      showProfileDialog.value = true
      showUserMenu.value = false
    }

    /**
     * 打开系统设置弹窗。
     */
    const openSystemSettingsDialog = () => {
      if (!isSystemAdmin.value) return
      showSystemSettingsDialog.value = true
      showUserMenu.value = false
    }

    const handleLogout = async () => {
      try {
        await ElMessageBox.confirm(t('common.logoutConfirm'), t('common.tip'), {
          confirmButtonText: t('common.confirm'),
          cancelButtonText: t('common.cancel'),
          type: 'warning',
        })

        authStore.logout()
        router.push({ name: 'login' })
      } catch {
        // 用户取消操作
      }
    }

    /**
     * 在 iframe load 后登记嵌入应用，供宿主后续按 source + origin 匹配 bootstrap 请求。
     * 这里故意按最新 load 结果覆盖，兼容同一标签页内 iframe 重载或并行改动后的 src 更新。
     * @param {object} payload - 注册信息
     */
    const registerEmbeddedApp = (payload) => {
      registerEmbeddedIframe(embeddedRegistry, payload)
    }

    /**
     * 标签页销毁时移除嵌入登记，避免旧 contentWindow 继续命中宿主匹配。
     * @param {object} payload - 卸载信息
     */
    const unregisterEmbeddedApp = (payload = {}) => {
      if (!payload?.iframe) return
      unregisterEmbeddedIframe(embeddedRegistry, payload.iframe)
    }

    /**
     * 基于当前宿主态生成 bootstrap 载荷。
     * 敏感信息只在已登记 iframe 的定向响应里返回，不放进正式入口 URL 查询串。
     * @param {object} entry - 已匹配的嵌入注册项
     * @returns {object} bootstrap 宿主态
     */
    const resolveEmbeddedBootstrapState = (entry) => ({
      token: Storage.getToken(),
      refreshToken: Storage.getRefreshToken(),
      theme: appStore.theme,
      locale: appStore.language,
      tenantId: entry?.project?.tenantId ?? Storage.getTenantId(),
    })

    /**
     * 接收嵌入应用刷新后的 token，并同步回宿主缓存。
     * 这里只更新当前前端会直接读取的 token/refreshToken，避免重做整套登录流程。
     * @param {object} payload - 认证刷新载荷
     */
    const handleEmbeddedAuthRefreshed = (payload = {}) => {
      const nextToken = payload.token ?? payload.accessToken
      const nextRefreshToken = payload.refreshToken

      if (!nextToken) return

      Storage.setToken(nextToken)
      authStore.token = nextToken
      authStore.isAuthenticated = true

      if (nextRefreshToken) {
        Storage.setRefreshToken(nextRefreshToken)
        authStore.refreshToken = nextRefreshToken
      }
    }

    /**
     * 嵌入应用认证失效后，沿用宿主已有登出与路由清理流程。
     * 这里不再弹确认框，避免已失效会话把用户卡在不可用页面。
     */
    const handleEmbeddedAuthExpired = () => {
      authStore.logout()
      router.push({ name: 'login' })
    }

    /**
     * 处理来自嵌入应用的消息。
     * 只有 source + origin 能匹配已登记 iframe 时才会处理，避免未知窗口冒充宿主协议。
     * @param {MessageEvent} event - 浏览器消息事件
     */
    const handleEmbeddedMessage = (event) => {
      handleEmbeddedWindowMessage({
        event,
        registry: embeddedRegistry,
        resolveBootstrapState: resolveEmbeddedBootstrapState,
        onAuthRefreshed: handleEmbeddedAuthRefreshed,
        onAuthExpired: handleEmbeddedAuthExpired,
      })
    }

    // 打开标签页
    const openTab = (tabData) => {
      // 支持两种调用方式：字符串key或对象
      const isObject = typeof tabData === 'object'
      const tabKey = isObject ? tabData.key : tabData
      const customTitle = isObject ? tabData.title : null
      const customTitlePrefix = isObject ? tabData.titlePrefix : null
      const customTitleParams = isObject ? tabData.titleParams : null
      const customComponent = isObject ? tabData.component : null
      const customProps = isObject ? tabData.props : null

      let config
      if (isObject && customComponent) {
        // 自定义标签页配置
        // 如果组件是函数（可能是动态导入），使用 defineAsyncComponent 包装以确保正确处理
        const component = typeof customComponent === 'function'
          ? defineAsyncComponent(customComponent)
          : customComponent
        config = {
          title: customTitle,
          component: component,
          icon: tabData.icon || 'folder',
          titleKey: tabData.titleKey || null,
          titlePrefix: customTitlePrefix,
          titleParams: customTitleParams,
        }
      } else {
        // 标准标签页配置
        const tabConfigs = getTabConfigMap()
        config = tabConfigs[tabKey]
        if (!config) {
          console.warn('Dashboard: No config found for tab:', tabKey)
          return
        }
      }

      // 权限检查（仅对标准标签页）
      if (!isObject) {
        if (!hasTabPermission(tabKey)) {
          ElMessage.warning(getTabAccessDeniedMessage(tabKey))
          return
        }
      }

      // 检查标签页是否已存在
      const existingTab = tabs.value.find((tab) => tab.key === tabKey)
      if (existingTab) {
        // 如果已存在，激活它
        activeTab.value = tabKey
        return
      }

      // 添加新标签页
      const newTab = {
        key: tabKey,
        title: resolveDashboardTabTitle(config, t),
        titleKey: config.titleKey || null,
        titlePrefix: config.titlePrefix || null,
        titleParams: config.titleParams || null,
        component: config.component,
        icon: config.icon,
        props: customProps,
      }

      tabs.value.push(newTab)

      // 激活新标签页
      activeTab.value = tabKey
    }

    /**
     * 打开运维待审核申请界面。
     */
    const openOpsPendingRequests = () => {
      if (!canReceiveNodePendingAlerts.value) return
      openTab('ops-management')
      // 通过短时重试确保 OpsManagement 完成挂载后再打开弹窗。
      ;[80, 220, 420].forEach((delay) => {
        window.setTimeout(() => {
          window.dispatchEvent(new window.CustomEvent('ops:open-pending-requests'))
        }, delay)
      })
    }

    /**
     * 显示节点待审核的全局通知（常驻，手动关闭）。
     * @param {object} payload - 节点申请事件载荷
     */
    const showNodePendingNotification = (payload = {}) => {
      const nodeId = payload.nodeId || payload.id || `${Date.now()}`
      if (pendingRequestNotifications.has(nodeId)) return

      const nodeName = payload.nodeName || '-'
      const applicantName = payload.applicant?.username || payload.username || '-'
      let notificationRef = null
      notificationRef = ElNotification({
        title: t('dashboard.pendingNodeTitle'),
        message: t('dashboard.pendingNodeMessage', { nodeName, applicant: applicantName }),
        type: 'warning',
        duration: 0,
        showClose: true,
        onClick: () => {
          openOpsPendingRequests()
          notificationRef?.close?.()
        },
        onClose: () => {
          pendingRequestNotifications.delete(nodeId)
        },
      })

      pendingRequestNotifications.set(nodeId, notificationRef)
    }

    /**
     * 页面初始化时检查是否存在未处理的待审核申请，并主动提醒一次。
     */
    const notifyExistingPendingRequests = async () => {
      if (!canReceiveNodePendingAlerts.value) return
      try {
        const res = await request.get('/nodes', {
          params: {
            page: 1,
            pageSize: 5,
            approvalStatus: 'pending',
          },
        })
        const items = res?.data?.items || []
        if (!Array.isArray(items) || items.length === 0) return

        const first = items[0]
        showNodePendingNotification({
          nodeId: `pending-summary-${Date.now()}`,
          nodeName: first?.name || '-',
          applicant: { username: first?.registrant?.username || '-' },
        })
      } catch (error) {
        console.warn('[OpsPending] 初始化待审核提醒失败:', error)
      }
    }

    /**
     * 订阅运维事件并处理全局通知。
     */
    const setupOpsPendingSubscription = () => {
      if (!canReceiveNodePendingAlerts.value) return
      const tenantId = Storage.getTenantId()
      if (!tenantId) return
      const socket = initSocket(tenantId)
      opsSocket.value = socket

      socket.on('connect', handleOpsSocketConnect)
      socket.on('disconnect', handleOpsSocketDisconnect)
      socket.on('connect_error', handleOpsSocketConnectError)
      socket.on('ops:node:pending', handleOpsNodePending)

      if (wsConnectCheckTimer.value) {
        window.clearTimeout(wsConnectCheckTimer.value)
      }
      wsConnectCheckTimer.value = window.setTimeout(() => {
        if (!socket.connected) {
          console.warn('[OpsPending][WS] 连接超时，启用轮询兜底')
          startPendingPolling('connect-timeout')
        }
      }, 6000)
    }

    /**
     * 兜底：轮询待审核申请数量，避免 WebSocket 异常时无法实时提示。
     */
    const startPendingPolling = async (reason = 'unknown') => {
      if (!canReceiveNodePendingAlerts.value) return
      if (pendingPollingTimer.value) return
      console.warn(`[OpsPending][Polling] 已启用，原因: ${reason}`)

      const pollPendingCount = async (notifyOnIncrease = false) => {
        try {
          const res = await request.get('/nodes', { params: { pageSize: 1, approvalStatus: 'pending' } })
          const total = Number(res?.data?.total || 0)
          if (notifyOnIncrease && total > lastPendingCount.value) {
            console.log(`[OpsPending][Polling] 检测到待审核新增: ${lastPendingCount.value} -> ${total}`)
            ElNotification({
              title: t('dashboard.pendingNodeTitle'),
              message: t('dashboard.pendingNodeMessage', { nodeName: '-', applicant: '-' }),
              type: 'warning',
              duration: 5000,
              onClick: () => {
                openOpsPendingRequests()
              },
            })
          }
          lastPendingCount.value = total
        } catch (error) {
          console.warn('轮询待审核申请失败:', error)
        }
      }

      await pollPendingCount(false)
      pendingPollingTimer.value = window.setInterval(() => {
        pollPendingCount(true)
      }, 8000)
    }

    /**
     * 停止兜底轮询。
     */
    const stopPendingPolling = () => {
      if (pendingPollingTimer.value) {
        window.clearInterval(pendingPollingTimer.value)
        pendingPollingTimer.value = null
        console.log('[OpsPending][Polling] 已停止')
      }
    }

    /**
     * WebSocket 连接成功回调。
     */
    const handleOpsSocketConnect = () => {
      console.log('[OpsPending][WS] 已连接')
      stopPendingPolling()
    }

    /**
     * WebSocket 连接断开回调。
     * @param {string} reason - 断开原因
     */
    const handleOpsSocketDisconnect = (reason) => {
      console.warn('[OpsPending][WS] 连接断开:', reason)
      startPendingPolling(`disconnect:${reason || 'unknown'}`)
    }

    /**
     * WebSocket 连接错误回调。
     * @param {Error} error - 连接错误
     */
    const handleOpsSocketConnectError = (error) => {
      console.warn('[OpsPending][WS] 连接失败:', error?.message || error)
      startPendingPolling(`connect_error:${error?.message || 'unknown'}`)
    }

    /**
     * 收到待审核事件回调。
     * @param {object} payload - 节点申请事件载荷
     */
    const handleOpsNodePending = (payload = {}) => {
      console.log('[OpsPending][WS] 收到待审核事件:', payload)
      showNodePendingNotification(payload)
    }

    /**
     * 获取当前用户对应的标签持久化键。
     * @returns {string} 本地存储键
     */
    const getTabStateStorageKey = () => {
      const userId = authStore.userInfo?.id || 'anonymous'
      return `${STORAGE_KEYS.DASHBOARD_TAB_STATE}_${userId}`
    }

    /**
     * 持久化当前标签状态。
     */
    const persistTabState = () => {
      if (!tabsInitialized.value) return
      Storage.set(
        getTabStateStorageKey(),
        serializeDashboardTabState({
          tabs: tabs.value,
          activeTab: activeTab.value,
          tabConfigMap: getTabConfigMap(),
          hasTabPermission,
        })
      )
    }

    /**
     * 从本地存储恢复标签状态。
     * @returns {boolean} 是否恢复成功
     */
    const restoreTabState = () => {
      const restoredState = restoreDashboardTabState(Storage.get(getTabStateStorageKey(), null), {
        tabConfigMap: getTabConfigMap(),
        hasTabPermission,
        embeddedComponent: EmbeddedApp,
        translate: t,
      })
      if (!restoredState) return false

      tabs.value = restoredState.tabs
      activeTab.value = restoredState.activeTab

      return true
    }

    /**
     * 消费地址栏中的 handoffId，并恢复独立标签页对应的嵌入应用标签。
     * 这里在 Dashboard 挂载后执行，确保设计中心/数据中心独立打开时能落到正确标签，而不是停留在 IDE 首页。
     */
    const restoreEmbeddedTabFromRoute = async () => {
      const handoffId = extractDashboardHandoffId(route.query)
      if (!handoffId) return

      const restoredPayload = resolveRestorePayload(handoffId)
      const restoredTab = createRestoredEmbeddedTab(restoredPayload)

      if (restoredTab) {
        openTab({
          ...restoredTab,
          component:
            restoredTab.component === EMBEDDED_APP_COMPONENT ? EmbeddedApp : restoredTab.component,
        })
      }

      await router.replace({
        path: route.path || '/dashboard',
        query: stripDashboardHandoffQuery(route.query),
        hash: route.hash,
      })
    }

    // 关闭标签页
    const closeTab = (tabKey) => {
      const index = tabs.value.findIndex((tab) => tab.key === tabKey)
      if (index === -1) return

      tabs.value.splice(index, 1)

      // 如果关闭的是当前激活的标签页，选择其他标签页
      if (activeTab.value === tabKey) {
        if (tabs.value.length > 0) {
          // 优先选择有权限的标签页
          let newTabKey = null
          for (let i = tabs.value.length - 1; i >= 0; i--) {
            if (isTabVisible(tabs.value[i].key)) {
              newTabKey = tabs.value[i].key
              break
            }
          }

          // 如果没有找到有权限的标签页，则选择相邻的标签页（虽然可能没有权限）
          if (!newTabKey) {
            const newIndex = Math.min(index, tabs.value.length - 1)
            newTabKey = tabs.value[newIndex].key
          }

          activeTab.value = newTabKey
        } else {
          // 如果没有标签页了，清空激活状态
          activeTab.value = ''
        }
      }
    }

    // 点击外部关闭菜单
    const handleClickOutside = (event) => {
      const userMenu = event.target.closest('.user-menu')
      const languageMenu = event.target.closest('.language-menu')

      if (!userMenu) {
        showUserMenu.value = false
      }
      if (!languageMenu) {
        showLanguageMenu.value = false
      }
    }

    onMounted(async () => {
      // 添加全局点击事件监听
      document.addEventListener('click', handleClickOutside)
      window.addEventListener('message', handleEmbeddedMessage)

      // 优先恢复历史标签状态，未恢复成功时按角色打开默认标签
      const restored = restoreTabState()
      if (!restored) {
        if (isSuperAdmin.value) {
          openTab('tenant-management')
        } else {
          openTab('dashboard')
        }
      }
      await restoreEmbeddedTabFromRoute()
      tabsInitialized.value = true
      persistTabState()
      setupOpsPendingSubscription()
      notifyExistingPendingRequests()

      // 仅超级管理员按需获取租户详情，避免非超级管理员触发租户接口请求
      if (isSuperAdmin.value && authStore.userInfo?.tenantId) {
        try {
          const { tenantAPI } = await import('@/api')
          const response = await tenantAPI.getTenantById(authStore.userInfo.tenantId)
          const currentTenantData = response?.data?.tenant || response?.data || null
          if (currentTenantData) {
            tenantStore.setCurrentTenant(currentTenantData)
          }
        } catch (error) {
          console.error('Failed to fetch current tenant:', error)
        }
      }
    })

    // 组件卸载时移除事件监听
    onUnmounted(() => {
      document.removeEventListener('click', handleClickOutside)
      window.removeEventListener('message', handleEmbeddedMessage)
      embeddedRegistry.clear()
      const socket = getSocket()
      if (socket) {
        socket.off('connect', handleOpsSocketConnect)
        socket.off('disconnect', handleOpsSocketDisconnect)
        socket.off('connect_error', handleOpsSocketConnectError)
        socket.off('ops:node:pending', handleOpsNodePending)
      }
      pendingRequestNotifications.forEach((notification) => notification?.close?.())
      pendingRequestNotifications.clear()
      stopPendingPolling()
      if (wsConnectCheckTimer.value) {
        window.clearTimeout(wsConnectCheckTimer.value)
        wsConnectCheckTimer.value = null
      }
      if (maximizeRestoreHideTimer.value) {
        window.clearTimeout(maximizeRestoreHideTimer.value)
        maximizeRestoreHideTimer.value = null
      }
    })

    // 最大化标签页
    const maximizeTab = (tabKey) => {
      if (tabKey === 'dashboard') return
      isTabMaximized.value = true
      activeTab.value = tabKey
      showMaximizeRestoreButton.value = true
      if (maximizeRestoreHideTimer.value) {
        window.clearTimeout(maximizeRestoreHideTimer.value)
      }
      maximizeRestoreHideTimer.value = window.setTimeout(() => {
        showMaximizeRestoreButton.value = false
      }, 1200)
    }

    // 还原标签页
    const restoreTab = () => {
      isTabMaximized.value = false
      showMaximizeRestoreButton.value = false
      if (maximizeRestoreHideTimer.value) {
        window.clearTimeout(maximizeRestoreHideTimer.value)
        maximizeRestoreHideTimer.value = null
      }
    }

    const handleMaximizedMouseMove = (event) => {
      if (!isTabMaximized.value) return
      if (event?.clientY <= 56) {
        showMaximizeRestoreButton.value = true
        if (maximizeRestoreHideTimer.value) {
          window.clearTimeout(maximizeRestoreHideTimer.value)
          maximizeRestoreHideTimer.value = null
        }
        return
      }
      if (showMaximizeRestoreButton.value && !maximizeRestoreHideTimer.value) {
        maximizeRestoreHideTimer.value = window.setTimeout(() => {
          showMaximizeRestoreButton.value = false
          maximizeRestoreHideTimer.value = null
        }, 220)
      }
    }

    const handleMaximizedMouseLeave = () => {
      if (!isTabMaximized.value) return
      showMaximizeRestoreButton.value = false
      if (maximizeRestoreHideTimer.value) {
        window.clearTimeout(maximizeRestoreHideTimer.value)
        maximizeRestoreHideTimer.value = null
      }
    }

    // 获取当前标签页标题
    const getTabTitle = (tab) => {
      return resolveDashboardTabTitle(tab, t)
    }

    /**
     * 在新标签页打开设计中心或数据中心。
     * @param {object} tab - 标签页配置
     */
    const openExternalTab = (tab) => {
      if (!tab?.props?.appType) return
      const url = buildAppUrl(tab.props.appType, tab.props.project)
      const newWindow = window.open(url, '_blank', 'noopener')
      if (newWindow) newWindow.opener = null
    }

    /**
     * 切换侧边栏折叠状态。
     */
    const toggleSidebar = () => {
      appStore.toggleSidebar()
    }

    watch(isDark, (nextIsDark) => {
      const theme = nextIsDark ? 'dark' : 'light'
      syncEmbeddedTheme(theme)
    })

    watch(locale, () => {
      tabs.value = tabs.value.map((tab) => {
        return {
          ...tab,
          title: resolveDashboardTabTitle(tab, t),
        }
      })
      syncEmbeddedLocale(locale.value)
    })

    watch(
      [tabs, activeTab],
      () => {
        persistTabState()
      },
      { deep: true }
    )

    return {
      showUserMenu,
      showLanguageMenu,
      sidebarCollapsed,
      tabs,
      activeTab,
      isDark,
      username,
      userInitials,
      userAvatarUrl,
      t,
      isAdmin,
      isSuperAdmin,
      isSystemAdmin,
      isOpsAdmin,
      isProjectAdmin,
      isUserAdmin,
      hasTabPermission,
      isTabVisible,
      currentTenant,
      tenantLogoUrl,
      toggleTheme,
      changeLanguage,
      currentLanguage,
      showProfileDialog,
      showSystemSettingsDialog,
      openProfileDialog,
      openSystemSettingsDialog,
      handleLogout,
      registerEmbeddedApp,
      unregisterEmbeddedApp,
      openTab,
      closeTab,
      maximizeTab,
      restoreTab,
      handleMaximizedMouseMove,
      handleMaximizedMouseLeave,
      getTabTitle,
      openExternalTab,
      toggleSidebar,
      isTabMaximized,
      showMaximizeRestoreButton,
    }
  },
}
</script>

<style scoped>
.dashboard {
  height: 100vh;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.sidebar-menu-label {
  display: inline-block;
  white-space: nowrap;
  overflow: hidden;
  transform-origin: left center;
  transition:
    max-width 260ms cubic-bezier(0.22, 1, 0.36, 1),
    opacity 180ms ease,
    transform 220ms cubic-bezier(0.22, 1, 0.36, 1);
}

.sidebar-menu-label-expanded {
  max-width: 120px;
  opacity: 1;
  transform: translateX(0);
}

.sidebar-menu-label-collapsed {
  max-width: 0;
  opacity: 0;
  transform: translateX(-4px);
}

.sidebar-toggle-button {
  opacity: 0;
  pointer-events: none;
}

.sidebar-panel:hover .sidebar-toggle-button,
.sidebar-panel:focus-within .sidebar-toggle-button {
  opacity: 1;
  pointer-events: auto;
}

/* 标签页样式 */
.dashboard-tabs :deep(.el-tabs__header) {
  margin: 0;
  padding: 0;
}

.dashboard-tabs :deep(.el-tabs__nav-wrap) {
  margin-bottom: 0;
}

.dashboard-tabs :deep(.el-tabs__nav-wrap::after) {
  display: none;
}

.dashboard-tabs :deep(.el-tabs__nav) {
  border-radius: 6px;
  border-bottom: 1px solid rgb(229 231 235);
  overflow: hidden;
}

html.dark .dashboard-tabs :deep(.el-tabs__nav),
[data-theme="dark"] .dashboard-tabs :deep(.el-tabs__nav) {
  border-bottom-color: rgb(55 65 81);
}

.dashboard-tabs :deep(.el-tabs__item) {
  border-radius: 4px 4px 0 0;
  color: rgb(55 65 81);
  padding: 6px 12px;
  height: 36px;
  line-height: 22px;
  font-size: 14px;
}

.dashboard-tabs :deep(.el-tabs__item:first-child) {
  padding-left: 10px;
}

.dashboard-tabs :deep(.el-tabs__item:hover) {
  color: rgb(59 130 246);
  background-color: rgb(239 246 255);
}

.dashboard-tabs :deep(.el-tabs__item.is-active) {
  color: rgb(59 130 246);
  background-color: rgb(239 246 255);
  border-bottom-color: rgb(239 246 255);
}

html.dark .dashboard-tabs :deep(.el-tabs__item),
[data-theme="dark"] .dashboard-tabs :deep(.el-tabs__item) {
  color: rgb(209 213 219);
}

html.dark .dashboard-tabs :deep(.el-tabs__item:hover),
[data-theme="dark"] .dashboard-tabs :deep(.el-tabs__item:hover) {
  color: rgb(147 197 253);
  background-color: rgb(31 41 55);
}

html.dark .dashboard-tabs :deep(.el-tabs__item.is-active),
[data-theme="dark"] .dashboard-tabs :deep(.el-tabs__item.is-active) {
  color: rgb(191 219 254);
  background-color: rgb(30 58 138 / 0.28);
  border-bottom-color: rgb(30 58 138 / 0.28);
}

.dashboard-tabs :deep(.el-tabs__content) {
  padding: 0;
  height: 100%;
  overflow: hidden;
}

.dashboard-tabs :deep(.el-tab-pane) {
  height: 100%;
  overflow: hidden;
}

/* 最大化状态下的样式 */
.dashboard-maximized {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  z-index: 1000;
}

.dashboard-tabs-maximized :deep(.el-tabs__header) {
  display: none;
}

.dashboard-tabs-maximized :deep(.el-tabs__content) {
  height: 100%;
}

.maximize-restore-floating-button {
  pointer-events: auto;
  z-index: 1100;
  width: 36px;
  height: 36px;
  border: 1px solid rgb(31 41 55 / 45%);
  background: rgb(17 24 39 / 88%);
  color: #fff;
  box-shadow: 0 6px 16px rgb(0 0 0 / 28%);
}

.maximize-restore-floating-button:hover {
  background: rgb(17 24 39 / 96%);
  border-color: rgb(31 41 55 / 70%);
}

.maximize-restore-floating-button :deep(.el-icon) {
  font-size: 16px;
  color: #fff;
}

.maximize-restore-anchor {
  position: fixed;
  top: 6px;
  left: 50%;
  transform: translateX(-50%);
  pointer-events: none;
  z-index: 1100;
}

html.dark .maximize-restore-floating-button,
[data-theme="dark"] .maximize-restore-floating-button {
  border-color: rgb(148 163 184 / 45%);
  background: rgb(15 23 42 / 88%);
}

.drop-down-enter-active,
.drop-down-leave-active {
  transition: transform 0.18s ease, opacity 0.18s ease;
}

.drop-down-enter-from,
.drop-down-leave-to {
  transform: translate(-50%, -14px);
  opacity: 0;
}

.drop-down-enter-to,
.drop-down-leave-from {
  transform: translate(-50%, 0);
  opacity: 1;
}
</style>
