<template>
  <div class="admin-dashboard h-screen flex flex-col bg-gray-50 dark:bg-[#121212]">
    <!-- 页面头部 -->
    <div
      class="bg-white dark:bg-gray-800 shadow-sm border-b border-gray-200 dark:border-gray-700 h-12 px-4 flex-none z-10"
    >
      <div class="flex items-center justify-between h-full">
        <!-- Logo 图片 -->
        <div class="flex items-center">
          <img :src="logoUrl" alt="Logo" class="h-7 w-auto mr-2" />
          <h1 class="text-base font-semibold text-gray-800 dark:text-white hidden sm:block">
            {{ t('adminDashboard.title') }}
          </h1>
        </div>
        <div class="flex items-center space-x-2">
          <!-- 主题切换 -->
          <button
            @click="toggleTheme"
            class="h-8 w-8 flex items-center justify-center rounded-md text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
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
          <div class="relative">
            <button
              @click="showUserMenu = !showUserMenu"
              class="flex items-center gap-2 h-8 px-2 rounded-md text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
            >
              <div class="w-6 h-6 bg-blue-500 rounded-full flex items-center justify-center">
                <span class="text-white text-xs font-medium">
                  {{ userInfo?.username?.charAt(0)?.toUpperCase() || 'A' }}
                </span>
              </div>
              <span class="text-xs font-medium text-gray-600 dark:text-gray-400">{{
                userInfo?.username || t('adminDashboard.admin')
              }}</span>
            </button>

            <!-- 下拉菜单 -->
            <transition
              enter-active-class="transition ease-out duration-100"
              enter-from-class="transform opacity-0 scale-95"
              enter-to-class="transform opacity-100 scale-100"
              leave-active-class="transition ease-in duration-75"
              leave-from-class="transform opacity-100 scale-100"
              leave-to-class="transform opacity-0 scale-95"
            >
              <div
                v-if="showUserMenu"
                class="absolute right-0 mt-2 w-48 bg-white dark:bg-gray-800 rounded-lg shadow-lg border border-gray-200 dark:border-gray-700 py-1 z-50 overflow-hidden"
              >
                <button
                  @click="openProfileDialog"
                  class="block w-full text-left px-4 py-2 text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
                >
                  {{ t('adminDashboard.profile') }}
                </button>
                <button
                  @click="handleLogout"
                  class="block w-full text-left px-4 py-2 text-sm text-red-600 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20 transition-colors"
                >
                  {{ t('auth.logout') }}
                </button>
              </div>
            </transition>
          </div>
        </div>
      </div>
    </div>

    <!-- 主要内容区域 -->
    <div class="flex-1 overflow-auto w-full relative">
      <router-view />
    </div>

    <!-- 个人资料弹窗 -->
    <el-dialog
      v-model="showProfileDialog"
      width="600px"
      top="8vh"
      destroy-on-close
      append-to-body
      class="admin-profile-dialog"
      :title="t('adminDashboard.profile')"
    >
      <AdminProfile />
    </el-dialog>
  </div>
</template>

<script lang="ts">
import { ref, computed, onMounted, onUnmounted, defineComponent, defineAsyncComponent } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAuthStore, useAppStore } from '@/store'
import defaultLogoUrl from '@/assets/images/default-logo.svg'

const AdminProfile = defineAsyncComponent(() => import('@/views/admin/components/AdminProfile.vue'))

export default defineComponent({
  name: 'AdminDashboard',
  components: {
    AdminProfile,
  },
  setup() {
    const { t } = useI18n()
    const router = useRouter()
    const authStore = useAuthStore()
    const appStore = useAppStore()
    const showUserMenu = ref(false)
    const showProfileDialog = ref(false)

    const userInfo = computed(() => authStore.userInfo)
    const isDark = computed(() => appStore.isDark)
    const logoUrl = computed(() => appStore.config?.logoUrl || defaultLogoUrl)

    const toggleTheme = () => {
      appStore.setTheme(isDark.value ? 'light' : 'dark')
    }

    const openProfileDialog = () => {
      showUserMenu.value = false
      showProfileDialog.value = true
    }

    const handleLogout = async () => {
      try {
        await authStore.logout()
        router.push({ name: 'login' })
      } catch (error) {
        console.error(t('adminDashboard.logoutFailed'), error)
      }
    }

    // 点击其他地方关闭用户菜单
    const handleClickOutside = (event: MouseEvent) => {
      const userMenu = document.querySelector('.relative')
      if (userMenu && event.target instanceof Node && !userMenu.contains(event.target)) {
        showUserMenu.value = false
      }
    }

    onMounted(() => {
      document.addEventListener('click', handleClickOutside)
    })

    onUnmounted(() => {
      document.removeEventListener('click', handleClickOutside)
    })

    return {
      showUserMenu,
      showProfileDialog,
      openProfileDialog,
      t,
      userInfo,
      isDark,
      logoUrl,
      toggleTheme,
      handleLogout,
    }
  },
})
</script>

<style scoped>
.admin-dashboard {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
}

:deep(.admin-profile-dialog) {
  border-radius: 1.5rem;
  overflow: hidden;
}

:deep(.admin-profile-dialog .el-dialog__header) {
  margin-right: 0;
  padding: 1.25rem 1.5rem 0.5rem;
  font-weight: 700;
}

:deep(.admin-profile-dialog .el-dialog__body) {
  padding: 1rem 1.5rem 1.5rem;
}
</style>
