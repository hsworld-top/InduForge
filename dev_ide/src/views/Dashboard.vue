<template>
  <div class="dashboard" :class="{ 'dashboard-maximized': isTabMaximized }">
    <!-- 页面头部 -->
    <div
      v-if="!isTabMaximized"
      class="bg-white dark:bg-gray-800 shadow-sm border-b border-gray-200 dark:border-gray-700 h-12 px-4"
    >
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
            <button
              @click="showLanguageMenu = !showLanguageMenu"
              class="h-8 w-8 flex items-center justify-center rounded-md text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-700"
            >
              <svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  stroke-width="2"
                  d="M3 5h12M9 3v2m1.048 9.5A18.022 18.022 0 016.412 9m6.088 9h7M11 21l5-10 5 10M12.751 5C11.783 10.77 8.07 15.61 3 18.129"
                />
              </svg>
            </button>

            <!-- 语言选择下拉菜单 -->
            <div
              v-if="showLanguageMenu"
              class="absolute right-0 mt-2 w-32 bg-white dark:bg-gray-800 rounded-lg shadow-lg border border-gray-200 dark:border-gray-700 py-1 z-10"
            >
              <button
                @click="changeLanguage('zh')"
                :class="[
                  'block w-full text-left px-4 py-2 text-sm hover:bg-gray-100 dark:hover:bg-gray-700',
                  currentLanguage === 'zh'
                    ? 'text-blue-600 dark:text-blue-400 bg-blue-50 dark:bg-blue-900/20'
                    : 'text-gray-700 dark:text-gray-300',
                ]"
              >
                中文
              </button>
              <button
                @click="changeLanguage('en')"
                :class="[
                  'block w-full text-left px-4 py-2 text-sm hover:bg-gray-100 dark:hover:bg-gray-700',
                  currentLanguage === 'en'
                    ? 'text-blue-600 dark:text-blue-400 bg-blue-50 dark:bg-blue-900/20'
                    : 'text-gray-700 dark:text-gray-300',
                ]"
              >
                English
              </button>
            </div>
          </div>

          <!-- 主题切换 -->
          <button
            @click="toggleTheme"
            class="h-8 w-8 flex items-center justify-center rounded-md text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-700"
          >
            <svg
              v-if="isDark"
              class="w-5 h-5"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
            >
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                stroke-width="2"
                d="M12 3v1m0 16v1m9-9h-1M4 12H3m15.364 6.364l-.707-.707M6.343 6.343l-.707-.707m12.728 0l-.707.707M6.343 17.657l-.707.707M16 12a4 4 0 11-8 0 4 4 0 018 0z"
              />
            </svg>
            <svg v-else class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                stroke-width="2"
                d="M20.354 15.354A9 9 0 018.646 3.646 9.003 9.003 0 0012 21a9.003 9.003 0 008.354-5.646z"
              />
            </svg>
          </button>

          <div class="h-4 w-px bg-gray-200 dark:bg-gray-700"></div>

          <!-- 用户菜单 -->
          <div class="relative user-menu">
            <button
              @click="showUserMenu = !showUserMenu"
              class="flex items-center gap-2 h-8 px-2 rounded-md text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700"
            >
              <img
                v-if="userAvatarUrl"
                :src="userAvatarUrl"
                alt="avatar"
                class="w-6 h-6 rounded-full object-cover"
              />
              <div
                v-else
                class="w-6 h-6 bg-blue-600 rounded-full flex items-center justify-center text-white text-xs font-medium"
              >
                {{ userInitials }}
              </div>
              <span class="text-xs text-gray-600 dark:text-gray-400">{{ username }}</span>
            </button>

            <!-- 下拉菜单 -->
            <div
              v-if="showUserMenu"
              class="absolute right-0 mt-2 w-48 bg-white dark:bg-gray-800 rounded-lg shadow-lg border border-gray-200 dark:border-gray-700 py-1 z-10"
            >
              <button
                @click="openProfileDialog"
                class="block w-full text-left px-4 py-2 text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700"
              >
                {{ t('dashboard.profile') }}
              </button>
              <button
                v-if="isSystemAdmin"
                @click="openSystemSettingsDialog"
                class="block w-full text-left px-4 py-2 text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700"
              >
                {{ t('dashboard.menuSettings') }}
              </button>
              <button
                @click="handleLogout"
                class="block w-full text-left px-4 py-2 text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700"
              >
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
      <div
        v-if="!isTabMaximized"
        :class="[
          'border-r transition-all duration-200 flex flex-col',
          isDark ? 'bg-gray-900 border-gray-800' : 'bg-white border-gray-200',
          sidebarCollapsed ? 'w-12' : 'w-44',
        ]"
      >
        <div :class="[sidebarCollapsed ? 'p-2' : 'p-3', 'flex-1 overflow-y-auto']">
          <nav class="space-y-2">
            <el-tooltip
              v-if="isSystemAdmin"
              :content="t('dashboard.title')"
              placement="right"
              :show-after="500"
              :disabled="!sidebarCollapsed"
            >
              <button
                @click="openTab('dashboard')"
                :class="[
                  'w-full h-11 flex items-center rounded-lg transition-colors',
                  sidebarCollapsed ? 'justify-center' : 'px-3 gap-3 justify-start',
                ]"
              >
                <span
                  :class="[
                    'w-9 h-9 rounded-lg flex items-center justify-center transition-colors',
                    activeTab === 'dashboard'
                      ? 'bg-blue-600 text-white'
                      : isDark
                        ? 'text-gray-400/80 hover:text-white hover:bg-gray-800'
                        : 'text-gray-500 hover:text-gray-700 hover:bg-gray-100',
                  ]"
                >
                  <svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path
                      stroke-linecap="round"
                      stroke-linejoin="round"
                      stroke-width="2"
                      d="M4 5a2 2 0 012-2h4a2 2 0 012 2v4a2 2 0 01-2 2H6a2 2 0 01-2-2V5z"
                    />
                    <path
                      stroke-linecap="round"
                      stroke-linejoin="round"
                      stroke-width="2"
                      d="M12 5a2 2 0 012-2h4a2 2 0 012 2v4a2 2 0 01-2 2h-4a2 2 0 01-2-2V5zM4 15a2 2 0 012-2h4a2 2 0 012 2v4a2 2 0 01-2 2H6a2 2 0 01-2-2v-4zM12 15a2 2 0 012-2h4a2 2 0 012 2v4a2 2 0 01-2 2h-4a2 2 0 01-2-2v-4z"
                    />
                  </svg>
                </span>
                <span
                  v-if="!sidebarCollapsed"
                  :class="[
                    'text-sm font-medium',
                    activeTab === 'dashboard' ? 'text-blue-600' : (isDark ? 'text-gray-200' : 'text-gray-700'),
                  ]"
                >
                  {{ t('dashboard.title') }}
                </span>
              </button>
            </el-tooltip>

            <el-tooltip
              v-if="isSuperAdmin"
              :content="t('dashboard.menuTenant')"
              placement="right"
              :show-after="500"
              :disabled="!sidebarCollapsed"
            >
              <button
                @click="openTab('tenant-management')"
                :class="[
                  'w-full h-11 flex items-center rounded-lg transition-colors',
                  sidebarCollapsed ? 'justify-center' : 'px-3 gap-3 justify-start',
                ]"
              >
                <span
                  :class="[
                    'w-9 h-9 rounded-lg flex items-center justify-center transition-colors',
                    activeTab === 'tenant-management'
                      ? 'bg-blue-600 text-white'
                      : isDark
                        ? 'text-gray-400/80 hover:text-white hover:bg-gray-800'
                        : 'text-gray-500 hover:text-gray-700 hover:bg-gray-100',
                  ]"
                >
                  <svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path
                      stroke-linecap="round"
                      stroke-linejoin="round"
                      stroke-width="2"
                      d="M19 21V5a2 2 0 00-2-2H7a2 2 0 00-2 2v16m14 0h2m-2 0h-5m-9 0H3m2 0h5M9 7h1m-1 4h1m4-4h1m-1 4h1m-5 10v-5a1 1 0 011-1h2a1 1 0 011 1v5m-4 0h4"
                    />
                  </svg>
                </span>
                <span
                  v-if="!sidebarCollapsed"
                  :class="[
                    'text-sm font-medium',
                    activeTab === 'tenant-management' ? 'text-blue-600' : (isDark ? 'text-gray-200' : 'text-gray-700'),
                  ]"
                >
                  {{ t('dashboard.menuTenant') }}
                </span>
              </button>
            </el-tooltip>

            <el-tooltip
              v-if="isSuperAdmin || isSystemAdmin || isUserAdmin"
              :content="t('dashboard.menuUser')"
              placement="right"
              :show-after="500"
              :disabled="!sidebarCollapsed"
            >
              <button
                @click="openTab('user-management')"
                :class="[
                  'w-full h-11 flex items-center rounded-lg transition-colors',
                  sidebarCollapsed ? 'justify-center' : 'px-3 gap-3 justify-start',
                ]"
              >
                <span
                  :class="[
                    'w-9 h-9 rounded-lg flex items-center justify-center transition-colors',
                    activeTab === 'user-management'
                      ? 'bg-blue-600 text-white'
                      : isDark
                        ? 'text-gray-400/80 hover:text-white hover:bg-gray-800'
                        : 'text-gray-500 hover:text-gray-700 hover:bg-gray-100',
                  ]"
                >
                  <el-icon class="w-5 h-5"><UserFilled /></el-icon>
                </span>
                <span
                  v-if="!sidebarCollapsed"
                  :class="[
                    'text-sm font-medium',
                    activeTab === 'user-management' ? 'text-blue-600' : (isDark ? 'text-gray-200' : 'text-gray-700'),
                  ]"
                >
                  {{ t('dashboard.menuUser') }}
                </span>
              </button>
            </el-tooltip>

            <el-tooltip
              v-if="isSuperAdmin || isSystemAdmin || isProjectAdmin"
              :content="t('dashboard.menuProject')"
              placement="right"
              :show-after="500"
              :disabled="!sidebarCollapsed"
            >
              <button
                @click="openTab('project-management')"
                :class="[
                  'w-full h-11 flex items-center rounded-lg transition-colors',
                  sidebarCollapsed ? 'justify-center' : 'px-3 gap-3 justify-start',
                ]"
              >
                <span
                  :class="[
                    'w-9 h-9 rounded-lg flex items-center justify-center transition-colors',
                    activeTab === 'project-management'
                      ? 'bg-blue-600 text-white'
                      : isDark
                        ? 'text-gray-400/80 hover:text-white hover:bg-gray-800'
                        : 'text-gray-500 hover:text-gray-700 hover:bg-gray-100',
                  ]"
                >
                  <svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path
                      stroke-linecap="round"
                      stroke-linejoin="round"
                      stroke-width="2"
                      d="M3 7a2 2 0 012-2h4l2 2h8a2 2 0 012 2v8a2 2 0 01-2 2H5a2 2 0 01-2-2V7z"
                    />
                    <path
                      stroke-linecap="round"
                      stroke-linejoin="round"
                      stroke-width="2"
                      d="M3 11h18"
                    />
                  </svg>
                </span>
                <span
                  v-if="!sidebarCollapsed"
                  :class="[
                    'text-sm font-medium',
                    activeTab === 'project-management' ? 'text-blue-600' : (isDark ? 'text-gray-200' : 'text-gray-700'),
                  ]"
                >
                  {{ t('dashboard.menuProject') }}
                </span>
              </button>
            </el-tooltip>

            <el-tooltip
              v-if="isSuperAdmin || isSystemAdmin || isOpsAdmin"
              :content="t('dashboard.menuOps')"
              placement="right"
              :show-after="500"
              :disabled="!sidebarCollapsed"
            >
              <button
                @click="openTab('ops-management')"
                :class="[
                  'w-full h-11 flex items-center rounded-lg transition-colors',
                  sidebarCollapsed ? 'justify-center' : 'px-3 gap-3 justify-start',
                ]"
              >
                <span
                  :class="[
                    'w-9 h-9 rounded-lg flex items-center justify-center transition-colors',
                    activeTab === 'ops-management'
                      ? 'bg-blue-600 text-white'
                      : isDark
                        ? 'text-gray-400/80 hover:text-white hover:bg-gray-800'
                        : 'text-gray-500 hover:text-gray-700 hover:bg-gray-100',
                  ]"
                >
                  <svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path
                      stroke-linecap="round"
                      stroke-linejoin="round"
                      stroke-width="2"
                      d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"
                    />
                    <path
                      stroke-linecap="round"
                      stroke-linejoin="round"
                      stroke-width="2"
                      d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"
                    />
                  </svg>
                </span>
                <span
                  v-if="!sidebarCollapsed"
                  :class="[
                    'text-sm font-medium',
                    activeTab === 'ops-management' ? 'text-blue-600' : (isDark ? 'text-gray-200' : 'text-gray-700'),
                  ]"
                >
                  {{ t('dashboard.menuOps') }}
                </span>
              </button>
            </el-tooltip>

            <el-tooltip
              v-if="isSuperAdmin || isSystemAdmin || isOpsAdmin"
              :content="t('dashboard.menuLogs')"
              placement="right"
              :show-after="500"
              :disabled="!sidebarCollapsed"
            >
              <button
                @click="openTab('system-logs')"
                :class="[
                  'w-full h-11 flex items-center rounded-lg transition-colors',
                  sidebarCollapsed ? 'justify-center' : 'px-3 gap-3 justify-start',
                ]"
              >
                <span
                  :class="[
                    'w-9 h-9 rounded-lg flex items-center justify-center transition-colors',
                    activeTab === 'system-logs'
                      ? 'bg-blue-600 text-white'
                      : isDark
                        ? 'text-gray-400/80 hover:text-white hover:bg-gray-800'
                        : 'text-gray-500 hover:text-gray-700 hover:bg-gray-100',
                  ]"
                >
                  <svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path
                      stroke-linecap="round"
                      stroke-linejoin="round"
                      stroke-width="2"
                      d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2"
                    />
                    <path
                      stroke-linecap="round"
                      stroke-linejoin="round"
                      stroke-width="2"
                      d="M9 3h6a1 1 0 011 1v2H8V4a1 1 0 011-1zM9 11h6M9 15h6"
                    />
                  </svg>
                </span>
                <span
                  v-if="!sidebarCollapsed"
                  :class="[
                    'text-sm font-medium',
                    activeTab === 'system-logs' ? 'text-blue-600' : (isDark ? 'text-gray-200' : 'text-gray-700'),
                  ]"
                >
                  {{ t('dashboard.menuLogs') }}
                </span>
              </button>
            </el-tooltip>

          </nav>
        </div>
        <div class="px-2 py-2">
          <div :class="['h-px', isDark ? 'bg-gray-800/50' : 'bg-gray-200/60']"></div>
          <button
            @click="toggleSidebar"
            :class="[
              'mt-2 mx-auto h-8 w-8 rounded-md flex items-center justify-center transition-colors',
              isDark
                ? 'text-gray-400 hover:text-white hover:bg-gray-800'
                : 'text-gray-500 hover:text-gray-800 hover:bg-gray-100',
            ]"
          >
            <svg
              :class="['w-4 h-4 transition-transform', sidebarCollapsed ? '' : 'rotate-180']"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
            >
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                stroke-width="2"
                d="M11 19l-7-7 7-7m8 14l-7-7 7-7"
              />
            </svg>
          </button>
        </div>
      </div>

      <!-- 右侧标签页区域 -->
      <div :class="['flex-1 overflow-hidden', isTabMaximized ? '' : 'pl-2']">
        <div v-if="tabs.length > 0" class="h-full">
          <!-- 最大化时的工具栏 -->
          <div v-if="isTabMaximized" class="bg-white dark:bg-gray-800 border-b border-gray-200 dark:border-gray-700 px-4 py-2 flex items-center justify-between">
            <div class="flex items-center space-x-2">
              <span class="text-sm font-medium text-gray-700 dark:text-gray-300">{{ getCurrentTabTitle() }}</span>
            </div>
            <el-button size="small" circle @click="restoreTab">
              <el-icon><FullScreen /></el-icon>
            </el-button>
          </div>
          <el-tabs
            v-model="activeTab"
            type="card"
            :closable="(tab) => tab.key !== 'dashboard'"
            @tab-remove="closeTab"
            :class="['dashboard-tabs h-full', isTabMaximized ? 'dashboard-tabs-maximized' : '']"
          >
            <el-tab-pane
              v-for="tab in tabs"
              :key="tab.key"
              :name="tab.key"
              :v-show="isTabVisible(tab.key)"
            >
              <template #label>
                <div class="flex items-center space-x-2">
                  <span>{{ getTabTitle(tab) }}</span>
                  <el-button
                    v-if="tab.props?.appType"
                    size="small"
                    text
                    circle
                    class="!p-0 !w-4 !h-4"
                    @click.stop="openExternalTab(tab)"
                  >
                    <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path
                        stroke-linecap="round"
                        stroke-linejoin="round"
                        stroke-width="2"
                        d="M14 3h7v7m0-7L10 14m-4 7h11a2 2 0 002-2V8"
                      />
                    </svg>
                  </el-button>
                  <el-button
                    v-if="!isTabMaximized && tab.key !== 'dashboard'"
                    size="small"
                    text
                    circle
                    class="!p-0 !w-4 !h-4"
                    @click.stop="maximizeTab(tab.key)"
                  >
                    <el-icon class="text-xs"><FullScreen /></el-icon>
                  </el-button>
                </div>
              </template>
              <div class="h-full overflow-hidden">
                <component :is="tab.component" @open-tab="openTab" v-bind="tab.props" />
              </div>
            </el-tab-pane>
          </el-tabs>
        </div>
      </div>
    </div>

    <el-dialog
      v-model="showProfileDialog"
      width="760px"
      destroy-on-close
      append-to-body
      class="profile-dialog"
      :title="t('profile.title')"
    >
      <Profile :embedded="true" />
    </el-dialog>

    <el-dialog
      v-model="showSystemSettingsDialog"
      width="960px"
      destroy-on-close
      append-to-body
      :title="t('dashboard.menuSettings')"
    >
      <SystemSettings />
    </el-dialog>
  </div>
</template>

<script>
import { ref, computed, onMounted, onUnmounted, watch, defineAsyncComponent } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAuthStore, useAppStore, useTenantStore } from '@/store'
import { ElMessage, ElMessageBox } from 'element-plus'
import { buildAppUrl } from '@/utils/appUrl'
import { canAccessTab, getTabAccessDeniedMessage } from '@/permissions'
import { ROLES } from '@/constants'

// 标签页组件懒加载，提升首次加载速度
const DashboardContent = defineAsyncComponent(() => import('@/views/DashboardContent.vue'))
const TenantManagement = defineAsyncComponent(() => import('@/views/admin/TenantManagement.vue'))
const UserManagement = defineAsyncComponent(() => import('@/views/tenant/UserManagement.vue'))
const ProjectManagement = defineAsyncComponent(() => import('@/views/tenant/ProjectManagement.vue'))
const OpsManagement = defineAsyncComponent(() => import('@/views/tenant/OpsManagement.vue'))
const SystemLogs = defineAsyncComponent(() => import('@/views/tenant/SystemLogs.vue'))
const SystemSettings = defineAsyncComponent(() => import('@/views/tenant/SystemSettings.vue'))
const Profile = defineAsyncComponent(() => import('@/views/profile/Profile.vue'))

// 导入默认Logo图片
import defaultLogo from '@/assets/images/demo.png'

export default {
  name: 'Dashboard',
  components: {
    Profile,
    SystemSettings,
  },
  setup() {
    const router = useRouter()
    const { locale, t } = useI18n()
    const authStore = useAuthStore()
    const appStore = useAppStore()
    const tenantStore = useTenantStore()

    const showUserMenu = ref(false)
    const showLanguageMenu = ref(false)
    const showProfileDialog = ref(false)
    const showSystemSettingsDialog = ref(false)

    // 侧边栏折叠状态（持久化）
    const sidebarCollapsed = computed({
      get: () => appStore.sidebarCollapsed,
      set: (value) => appStore.setSidebarCollapsed(value),
    })

    // 标签页状态
    const tabs = ref([])
    const activeTab = ref('')
    const isTabMaximized = ref(false)

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
      const iframes = document.querySelectorAll('iframe.embedded-iframe')
      iframes.forEach((iframe) => {
        iframe.contentWindow?.postMessage({ type: 'THEME_UPDATE', theme }, '*')
      })
    }

    // 租户相关计算属性
    const currentTenant = computed(() => tenantStore.currentTenant)
    const tenantLogoUrl = computed(() => {
      // 优先使用租户的logo，然后使用默认的demo.png
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

    // 打开标签页
    const openTab = (tabData) => {
      // 支持两种调用方式：字符串key或对象
      const isObject = typeof tabData === 'object'
      const tabKey = isObject ? tabData.key : tabData
      const customTitle = isObject ? tabData.title : null
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
        title: config.titleKey ? t(config.titleKey) : config.title,
        titleKey: config.titleKey || null,
        component: config.component,
        icon: config.icon,
        props: customProps,
      }

      tabs.value.push(newTab)

      // 激活新标签页
      activeTab.value = tabKey
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

      // 根据用户角色决定默认打开的标签页
      // 系统管理员和超级管理员默认打开仪表盘
      // 其他用户默认打开他们有权限访问的功能
      if (isSuperAdmin.value || isSystemAdmin.value) {
        openTab('dashboard')
      } else if (isUserAdmin.value) {
        openTab('user-management')
      } else if (isProjectAdmin.value) {
        openTab('project-management')
      } else if (isOpsAdmin.value) {
        openTab('system-logs')
      } else {
        // 如果没有任何权限，打开仪表盘（虽然看不到菜单，但至少有内容）
        openTab('dashboard')
      }

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
    })

    // 最大化标签页
    const maximizeTab = (tabKey) => {
      if (tabKey === 'dashboard') return
      isTabMaximized.value = true
      activeTab.value = tabKey
    }

    // 还原标签页
    const restoreTab = () => {
      isTabMaximized.value = false
    }

    // 获取当前标签页标题
    const getTabTitle = (tab) => {
      if (!tab) return ''
      if (tab.titleKey) return t(tab.titleKey)
      return tab.title || ''
    }

    const getCurrentTabTitle = () => {
      const tab = tabs.value.find(t => t.key === activeTab.value)
      return getTabTitle(tab)
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
        if (tab.titleKey) {
          return {
            ...tab,
            title: t(tab.titleKey),
          }
        }
        return tab
      })
    })

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
      openTab,
      closeTab,
      maximizeTab,
      restoreTab,
      getTabTitle,
      getCurrentTabTitle,
      openExternalTab,
      toggleSidebar,
      isTabMaximized,
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

/* 标签页样式 */
.dashboard-tabs :deep(.el-tabs__header) {
  margin: 0;
  padding: 4px 6px 0 6px;
}

.dashboard-tabs :deep(.el-tabs__nav-wrap) {
  margin-bottom: 0;
}

.dashboard-tabs :deep(.el-tabs__nav) {
  border-radius: 6px;
}

.dashboard-tabs :deep(.el-tabs__item) {
  border-radius: 4px 4px 0 0;
  margin-right: 4px;
  color: rgb(55 65 81);
  padding: 6px 14px;
  height: 36px;
  line-height: 22px;
  font-size: 14px;
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

.dashboard-tabs :deep(.dark .el-tabs__item) {
  color: rgb(209 213 219);
}

.dashboard-tabs :deep(.dark .el-tabs__item:hover) {
  color: rgb(147 197 253);
  background-color: rgb(30 41 59);
}

.dashboard-tabs :deep(.dark .el-tabs__item.is-active) {
  color: rgb(191 219 254);
  background-color: rgb(30 41 59);
  border-bottom-color: rgb(30 41 59);
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
</style>
