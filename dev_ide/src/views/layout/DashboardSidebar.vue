<template>
  <!-- 遮罩层 -->
  <Transition name="fade">
    <div 
      v-if="!sidebarCollapsed" 
      class="fixed inset-0 bg-gray-900/50 dark:bg-black/50 z-40 backdrop-blur-sm transition-opacity"
      @click="closeSidebar"
    ></div>
  </Transition>

  <!-- 侧边栏抽屉 -->
  <div 
    :class="[
      'fixed inset-y-0 left-0 z-50 w-60 flex flex-col transition-transform duration-300 ease-[cubic-bezier(0.22,1,0.36,1)] shadow-2xl',
      isDark ? 'bg-gray-900 border-r border-gray-800' : 'bg-white border-r border-gray-200',
      sidebarCollapsed ? '-translate-x-full' : 'translate-x-0'
    ]"
  >
    <!-- 顶部：Logo 与 收起按钮 -->
    <div class="h-14 px-4 flex items-center justify-between border-b" :class="isDark ? 'border-gray-800' : 'border-gray-100'">
      <div class="flex items-center overflow-hidden">
        <img :src="tenantLogoUrl" :alt="currentTenant?.name || 'Logo'" class="h-7 w-auto mr-2 shrink-0" />
        <h1 class="text-base font-semibold text-gray-800 dark:text-white truncate">
          {{ currentTenant?.name || 'InduForge' }}
        </h1>
      </div>
      <button 
        @click="closeSidebar"
        class="w-8 h-8 flex items-center justify-center rounded-md text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-800 shrink-0 transition-colors"
      >
        <el-icon class="text-[18px]">
          <Fold />
        </el-icon>
      </button>
    </div>

    <!-- 中部：导航菜单 -->
    <div class="flex-1 overflow-y-auto py-4 px-3 space-y-1">
      <nav>
        <button v-if="hasTabPermission('dashboard')" @click="handleOpenTab('dashboard')" :class="[
          'w-full h-10 flex items-center px-3 gap-3 rounded-lg transition-colors mb-1',
          activeTab === 'dashboard'
            ? 'bg-blue-50 text-blue-600 dark:bg-blue-900/30 dark:text-blue-400'
            : 'text-gray-600 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-gray-800'
        ]">
          <svg class="w-5 h-5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 5a2 2 0 012-2h4a2 2 0 012 2v4a2 2 0 01-2 2H6a2 2 0 01-2-2V5z" />
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 5a2 2 0 012-2h4a2 2 0 012 2v4a2 2 0 01-2 2h-4a2 2 0 01-2-2V5zM4 15a2 2 0 012-2h4a2 2 0 012 2v4a2 2 0 01-2 2H6a2 2 0 01-2-2v-4zM12 15a2 2 0 012-2h4a2 2 0 012 2v4a2 2 0 01-2 2h-4a2 2 0 01-2-2v-4z" />
          </svg>
          <span class="text-sm font-medium">{{ t('dashboard.title') }}</span>
        </button>

        <button v-if="hasTabPermission('tenant-management')" @click="handleOpenTab('tenant-management')" :class="[
          'w-full h-10 flex items-center px-3 gap-3 rounded-lg transition-colors mb-1',
          activeTab === 'tenant-management'
            ? 'bg-blue-50 text-blue-600 dark:bg-blue-900/30 dark:text-blue-400'
            : 'text-gray-600 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-gray-800'
        ]">
          <svg class="w-5 h-5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 21V5a2 2 0 00-2-2H7a2 2 0 00-2 2v16m14 0h2m-2 0h-5m-9 0H3m2 0h5M9 7h1m-1 4h1m4-4h1m-1 4h1m-5 10v-5a1 1 0 011-1h2a1 1 0 011 1v5m-4 0h4" />
          </svg>
          <span class="text-sm font-medium">{{ t('dashboard.menuTenant') }}</span>
        </button>

        <button v-if="hasTabPermission('user-management')" @click="handleOpenTab('user-management')" :class="[
          'w-full h-10 flex items-center px-3 gap-3 rounded-lg transition-colors mb-1',
          activeTab === 'user-management'
            ? 'bg-blue-50 text-blue-600 dark:bg-blue-900/30 dark:text-blue-400'
            : 'text-gray-600 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-gray-800'
        ]">
          <el-icon class="w-5 h-5 shrink-0"><UserFilled /></el-icon>
          <span class="text-sm font-medium">{{ t('dashboard.menuUser') }}</span>
        </button>

        <button v-if="hasTabPermission('project-management')" @click="handleOpenTab('project-management')" :class="[
          'w-full h-10 flex items-center px-3 gap-3 rounded-lg transition-colors mb-1',
          activeTab === 'project-management'
            ? 'bg-blue-50 text-blue-600 dark:bg-blue-900/30 dark:text-blue-400'
            : 'text-gray-600 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-gray-800'
        ]">
          <svg class="w-5 h-5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 7a2 2 0 012-2h4l2 2h8a2 2 0 012 2v8a2 2 0 01-2 2H5a2 2 0 01-2-2V7z" />
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 11h18" />
          </svg>
          <span class="text-sm font-medium">{{ t('dashboard.menuProject') }}</span>
        </button>

        <button v-if="hasTabPermission('ops-management')" @click="handleOpenTab('ops-management')" :class="[
          'w-full h-10 flex items-center px-3 gap-3 rounded-lg transition-colors mb-1',
          activeTab === 'ops-management'
            ? 'bg-blue-50 text-blue-600 dark:bg-blue-900/30 dark:text-blue-400'
            : 'text-gray-600 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-gray-800'
        ]">
          <svg class="w-5 h-5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
          </svg>
          <span class="text-sm font-medium">{{ t('dashboard.menuOps') }}</span>
        </button>

        <button v-if="hasTabPermission('system-logs')" @click="handleOpenTab('system-logs')" :class="[
          'w-full h-10 flex items-center px-3 gap-3 rounded-lg transition-colors mb-1',
          activeTab === 'system-logs'
            ? 'bg-blue-50 text-blue-600 dark:bg-blue-900/30 dark:text-blue-400'
            : 'text-gray-600 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-gray-800'
        ]">
          <svg class="w-5 h-5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2" />
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 3h6a1 1 0 011 1v2H8V4a1 1 0 011-1zM9 11h6M9 15h6" />
          </svg>
          <span class="text-sm font-medium">{{ t('dashboard.menuLogs') }}</span>
        </button>
      </nav>
    </div>

    <!-- 底部：统一用户菜单 -->
    <div class="p-3" :class="isDark ? 'border-gray-800' : 'border-gray-100'">
      <el-dropdown trigger="click" placement="top-start" @command="handleUserMenuCommand" class="w-full">
        <div class="flex items-center gap-2.5 h-[44px] px-2 w-full rounded-lg cursor-pointer hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors border border-transparent hover:border-gray-200 dark:hover:border-gray-700">
          <img v-if="userAvatarUrl" :src="userAvatarUrl" alt="avatar" class="w-7 h-7 rounded-full object-cover shadow-sm shrink-0" />
          <div v-else class="w-7 h-7 bg-blue-600 rounded-full flex items-center justify-center text-white text-xs font-medium shadow-sm shrink-0">
            {{ userInitials }}
          </div>
          <div class="flex-1 min-w-0">
            <p class="text-[13px] leading-tight font-medium text-gray-900 dark:text-gray-100 truncate">{{ username }}</p>
            <p class="text-[11px] leading-tight mt-0.5 text-gray-500 dark:text-gray-400 truncate">{{ authStore.userInfo?.role }}</p>
          </div>
          <svg class="w-3.5 h-3.5 text-gray-400 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 15l7-7 7 7" />
          </svg>
        </div>
        <template #dropdown>
          <el-dropdown-menu class="w-[220px] shadow-xl rounded-xl border border-gray-100 dark:border-gray-700">
            <el-dropdown-item command="profile">
              <span class="flex items-center gap-2 text-[13px]">
                <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" /></svg>
                {{ t('dashboard.profile') }}
              </span>
            </el-dropdown-item>
            <el-dropdown-item v-if="isSystemAdmin" command="settings">
              <span class="flex items-center gap-2 text-[13px]">
                <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" /><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" /></svg>
                {{ t('dashboard.menuSettings') }}
              </span>
            </el-dropdown-item>
            
            <el-dropdown-item divided command="toggleTheme">
              <span class="flex items-center justify-between w-full text-gray-600 dark:text-gray-300 text-[13px]">
                <span class="flex items-center gap-2">
                  <svg v-if="isDark" class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 3v1m0 16v1m9-9h-1M4 12H3m15.364 6.364l-.707-.707M6.343 6.343l-.707-.707m12.728 0l-.707.707M6.343 17.657l-.707.707M16 12a4 4 0 11-8 0 4 4 0 018 0z" /></svg>
                  <svg v-else class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20.354 15.354A9 9 0 018.646 3.646 9.003 9.003 0 0012 21a9.003 9.003 0 008.354-5.646z" /></svg>
                  {{ isDark ? (currentLanguage === 'zh' ? '浅色模式' : 'Light Mode') : (currentLanguage === 'zh' ? '暗色模式' : 'Dark Mode') }}
                </span>
              </span>
            </el-dropdown-item>
            
            <!-- 多语言二级菜单 (Popover 悬浮方案) -->
            <el-dropdown-item class="!p-0" divided @click.prevent.stop>
              <el-popover placement="right-start" trigger="hover" width="140" popper-class="!p-1.5 rounded-xl shadow-lg border border-gray-100 dark:border-gray-700" :show-arrow="false" :offset="12">
                <template #reference>
                  <div class="flex items-center justify-between w-full px-4 py-2 hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors">
                    <span class="flex items-center gap-2 text-gray-600 dark:text-gray-300 text-[13px]">
                      <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 5h12M9 3v2m1.048 9.5A18.022 18.022 0 016.412 9m6.088 9h7M11 21l5-10 5 10M12.751 5C11.783 10.77 8.07 15.61 3 18.129" /></svg>
                      {{ currentLanguage === 'zh' ? '语言 / Language' : 'Language / 语言' }}
                    </span>
                    <svg class="w-3 h-3 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" /></svg>
                  </div>
                </template>
                <div class="flex flex-col gap-0.5">
                  <button @click="handleUserMenuCommand('lang-zh')" :class="['w-full text-left px-3 py-2 rounded-lg text-[13px] hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors', currentLanguage === 'zh' ? 'text-blue-600 bg-blue-50 dark:bg-blue-900/20 font-medium' : 'text-gray-700 dark:text-gray-200']">简体中文</button>
                  <button @click="handleUserMenuCommand('lang-en')" :class="['w-full text-left px-3 py-2 rounded-lg text-[13px] hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors', currentLanguage === 'en' ? 'text-blue-600 bg-blue-50 dark:bg-blue-900/20 font-medium' : 'text-gray-700 dark:text-gray-200']">English</button>
                </div>
              </el-popover>
            </el-dropdown-item>

            <el-dropdown-item divided command="logout" class="!text-red-600 dark:!text-red-400">
              <span class="flex items-center gap-2 text-[13px]">
                <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1" /></svg>
                {{ t('auth.logout') }}
              </span>
            </el-dropdown-item>
          </el-dropdown-menu>
        </template>
      </el-dropdown>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { ElMessageBox } from 'element-plus'
import { UserFilled, Fold } from '@element-plus/icons-vue'
import { useAuthStore, useAppStore, useTenantStore } from '@/store'
import { canAccessTab } from '@/permissions'
import { ROLES } from '@/constants'
import defaultLogo from '@/assets/images/default-logo.svg'

const props = defineProps({
  activeTab: {
    type: String,
    required: true
  }
})

const emit = defineEmits(['open-tab', 'open-profile', 'open-settings'])

const router = useRouter()
const { locale, t } = useI18n()
const authStore = useAuthStore()
const appStore = useAppStore()
const tenantStore = useTenantStore()

const isDark = computed(() => appStore.isDark)
const sidebarCollapsed = computed(() => appStore.sidebarCollapsed)

const closeSidebar = () => {
  appStore.setSidebarCollapsed(true)
}

const handleOpenTab = (tabKey) => {
  emit('open-tab', tabKey)
  // 移动端/窄屏模式下，点击菜单后自动收起侧边栏
  // 这里作为通用交互，点击导航后关闭抽屉更符合专注模式
  closeSidebar()
}

// 用户信息
const username = computed(() => authStore.userInfo?.username || '')
const userInitials = computed(() => {
  const name = username.value || 'U'
  return name.charAt(0).toUpperCase()
})
const userAvatarUrl = computed(() => authStore.userInfo?.avatarUrl || '')
const isSystemAdmin = computed(() => authStore.userInfo?.role === ROLES.SYSTEM_ADMIN)

// 租户信息
const currentTenant = computed(() => tenantStore.currentTenant)
const tenantLogoUrl = computed(() => {
  const tenantLogo = currentTenant.value?.logoUrl
  if (tenantLogo) {
    return tenantLogo.startsWith('http') ? tenantLogo : tenantLogo
  }
  return defaultLogo
})

// 权限检查
const hasTabPermission = (tabKey) => canAccessTab(tabKey, authStore.userInfo?.role)

// 操作
const toggleTheme = () => {
  appStore.setTheme(isDark.value ? 'light' : 'dark')
}

const changeLanguage = (lang) => {
  locale.value = lang
  appStore.setLanguage(lang)
}
const currentLanguage = computed(() => appStore.language || 'zh')

const handleUserMenuCommand = async (command) => {
  if (command === 'profile') {
    emit('open-profile')
  } else if (command === 'settings') {
    emit('open-settings')
  } else if (command === 'toggleTheme') {
    toggleTheme()
  } else if (command.startsWith('lang-')) {
    const lang = command.replace('lang-', '')
    changeLanguage(lang)
  } else if (command === 'logout') {
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
}
</script>

<style scoped>
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.3s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
