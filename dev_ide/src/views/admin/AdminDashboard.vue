<template>
  <div class="admin-dashboard">
    <!-- 页面头部 -->
    <div
      class="bg-white dark:bg-gray-800 shadow-sm border-b border-gray-200 dark:border-gray-700 px-6 py-4"
    >
      <div class="flex items-center justify-between">
        <!-- Logo 图片 -->
        <div class="flex items-center">
          <img :src="logoUrl" alt="Logo" class="h-8 w-auto mr-4" />
          <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">超级管理员控制台</h1>
        </div>
        <div class="flex items-center space-x-4">
          <!-- 主题切换 -->
          <button
            @click="toggleTheme"
            class="p-2 rounded-lg text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-700"
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

          <!-- 用户菜单 -->
          <div class="relative">
            <button
              @click="showUserMenu = !showUserMenu"
              class="flex items-center space-x-2 p-2 rounded-lg text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700"
            >
              <div class="w-8 h-8 bg-blue-500 rounded-full flex items-center justify-center">
                <span class="text-white text-sm font-medium">
                  {{ userInfo?.username?.charAt(0)?.toUpperCase() || 'A' }}
                </span>
              </div>
              <span class="text-sm">{{ userInfo?.username || '管理员' }}</span>
              <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  stroke-width="2"
                  d="M19 9l-7 7-7-7"
                />
              </svg>
            </button>

            <!-- 下拉菜单 -->
            <div
              v-if="showUserMenu"
              class="absolute right-0 mt-2 w-48 bg-white dark:bg-gray-800 rounded-lg shadow-lg border border-gray-200 dark:border-gray-700 py-1 z-50"
            >
              <router-link
                to="/profile"
                class="block px-4 py-2 text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700"
              >
                个人资料
              </router-link>
              <button
                @click="handleLogout"
                class="block w-full text-left px-4 py-2 text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700"
              >
                登出
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 主要内容区域 -->
    <div class="flex flex-1">
      <!-- 左侧菜单栏 -->
      <div class="w-64 bg-white dark:bg-gray-800 border-r border-gray-200 dark:border-gray-700">
        <div class="p-4">
          <nav class="space-y-2">
            <!-- 租户管理 -->
            <router-link
              to="/admin/tenants"
              :class="[
                'w-full flex items-center px-4 py-3 text-left rounded-lg transition-colors',
                $route.path === '/admin/tenants'
                  ? 'bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300'
                  : 'text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700',
              ]"
            >
              <svg class="w-5 h-5 mr-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  stroke-width="2"
                  d="M19 21V5a2 2 0 00-2-2H7a2 2 0 00-2 2v16m14 0h2m-2 0h-5m-9 0H3m2 0h5M9 7h1m-1 4h1m4-4h1m-1 4h1m-5 10v-5a1 1 0 011-1h2a1 1 0 011 1v5m-4 0h4"
                />
              </svg>
              租户管理
            </router-link>
          </nav>
        </div>
      </div>

      <!-- 右侧内容区域 -->
      <div class="flex-1 p-6">
        <router-view />
      </div>
    </div>
  </div>
</template>

<script>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore, useAppStore } from '@/store'

export default {
  name: 'AdminDashboard',
  setup() {
    const router = useRouter()
    const authStore = useAuthStore()
    const appStore = useAppStore()
    const showUserMenu = ref(false)

    const userInfo = computed(() => authStore.userInfo)
    const isDark = computed(() => appStore.isDark)
    const logoUrl = computed(() => appStore.config?.logoUrl || '/images/logo.png')

    const toggleTheme = () => {
      appStore.setTheme(isDark.value ? 'light' : 'dark')
    }

    const handleLogout = async () => {
      try {
        await authStore.logout()
        router.push({ name: 'login' })
      } catch (error) {
        console.error('登出失败:', error)
      }
    }

    // 点击其他地方关闭用户菜单
    const handleClickOutside = (event) => {
      const userMenu = document.querySelector('.relative')
      if (userMenu && !userMenu.contains(event.target)) {
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
      userInfo,
      isDark,
      logoUrl,
      toggleTheme,
      handleLogout,
    }
  },
}
</script>

<style scoped>
.admin-dashboard {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
}
</style>
