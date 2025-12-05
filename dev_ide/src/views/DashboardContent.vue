<template>
  <div class="dashboard-content">
    <!-- 欢迎信息 -->
    <div class="mb-8">
      <h2 class="text-xl font-semibold text-gray-900 dark:text-white mb-2">
        欢迎回来，{{ username }}！
      </h2>
      <p class="text-gray-600 dark:text-gray-400">这是您的多租户管理系统仪表板</p>
    </div>

    <!-- 统计卡片 -->
    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
      <div
        class="card cursor-pointer hover:shadow-lg transition-shadow duration-200"
        @click="openTab('user-management')"
      >
        <div class="flex items-center">
          <div class="p-3 rounded-lg bg-green-100 dark:bg-green-900">
            <svg
              class="w-6 h-6 text-green-600 dark:text-green-400"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
            >
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                stroke-width="2"
                d="M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197m13.5-9a2.5 2.5 0 11-5 0 2.5 2.5 0 015 0z"
              />
            </svg>
          </div>
          <div class="ml-4">
            <p class="text-sm font-medium text-gray-600 dark:text-gray-400">用户数量</p>
            <p class="text-2xl font-semibold text-gray-900 dark:text-white">{{ stats.users }}</p>
          </div>
        </div>
      </div>

      <div
        class="card cursor-pointer hover:shadow-lg transition-shadow duration-200"
        @click="openTab('project-management')"
      >
        <div class="flex items-center">
          <div class="p-3 rounded-lg bg-blue-100 dark:bg-blue-900">
            <svg
              class="w-6 h-6 text-blue-600 dark:text-blue-400"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
            >
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                stroke-width="2"
                d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10"
              />
            </svg>
          </div>
          <div class="ml-4">
            <p class="text-sm font-medium text-gray-600 dark:text-gray-400">工程数量</p>
            <p class="text-2xl font-semibold text-gray-900 dark:text-white">
              {{ stats.projects }}
            </p>
          </div>
        </div>
      </div>

      <div
        class="card cursor-pointer hover:shadow-lg transition-shadow duration-200"
        @click="openTab('tenant-management')"
      >
        <div class="flex items-center">
          <div class="p-3 rounded-lg bg-yellow-100 dark:bg-yellow-900">
            <svg
              class="w-6 h-6 text-yellow-600 dark:text-yellow-400"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
            >
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                stroke-width="2"
                d="M19 21V5a2 2 0 00-2-2H7a2 2 0 00-2 2v16m14 0h2m-2 0h-5m-9 0H3m2 0h5M9 7h1m-1 4h1m4-4h1m-1 4h1m-5 10v-5a1 1 0 011-1h2a1 1 0 011 1v5m-4 0h4"
              />
            </svg>
          </div>
          <div class="ml-4">
            <p class="text-sm font-medium text-gray-600 dark:text-gray-400">租户数量</p>
            <p class="text-2xl font-semibold text-gray-900 dark:text-white">
              {{ stats.tenants }}
            </p>
          </div>
        </div>
      </div>

      <div
        class="card cursor-pointer hover:shadow-lg transition-shadow duration-200"
        @click="openTab('system-logs')"
      >
        <div class="flex items-center">
          <div class="p-3 rounded-lg bg-red-100 dark:bg-red-900">
            <svg
              class="w-6 h-6 text-red-600 dark:text-red-400"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
            >
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                stroke-width="2"
                d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L4.082 16.5c-.77.833.192 2.5 1.732 2.5z"
              />
            </svg>
          </div>
          <div class="ml-4">
            <p class="text-sm font-medium text-gray-600 dark:text-gray-400">系统日志</p>
            <p class="text-2xl font-semibold text-gray-900 dark:text-white">{{ stats.logs }}</p>
          </div>
        </div>
      </div>
    </div>

    <!-- 最近活动 -->
    <div class="card">
      <h3 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">最近活动</h3>
      <div class="space-y-4">
        <div
          v-for="activity in recentActivities"
          :key="activity.id"
          class="flex items-start space-x-3"
        >
          <div class="flex-shrink-0">
            <div
              class="w-8 h-8 bg-blue-100 dark:bg-blue-900 rounded-full flex items-center justify-center"
            >
              <svg
                class="w-4 h-4 text-blue-600 dark:text-blue-400"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
              >
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  stroke-width="2"
                  d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
                />
              </svg>
            </div>
          </div>
          <div class="flex-1 min-w-0">
            <p class="text-sm text-gray-900 dark:text-white">{{ activity.description }}</p>
            <p class="text-xs text-gray-500 dark:text-gray-400">{{ activity.time }}</p>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import { ref, computed } from 'vue'
import { useAuthStore } from '@/store'

export default {
  name: 'DashboardContent',
  emits: ['open-tab'],
  setup(props, { emit }) {
    const authStore = useAuthStore()

    const username = computed(() => authStore.username)
    const stats = ref({
      projects: 0,
      users: 0,
      tenants: 0,
      logs: 0,
    })

    const recentActivities = ref([
      {
        id: 1,
        description: '用户张三创建了新工程项目',
        time: '2小时前',
      },
      {
        id: 2,
        description: '系统管理员更新了租户配置',
        time: '4小时前',
      },
      {
        id: 3,
        description: '工程项目"网站重构"状态变更为进行中',
        time: '1天前',
      },
    ])

    // 加载统计数据
    const loadStats = () => {
      // 模拟加载统计数据
      stats.value = {
        projects: 12,
        users: 156,
        tenants: 8,
        logs: 2340,
      }
    }

    // 点击卡片打开对应标签页
    const openTab = (tabKey) => {
      emit('open-tab', tabKey)
    }

    loadStats()

    return {
      username,
      stats,
      recentActivities,
      openTab,
    }
  },
}
</script>

<style scoped>
.card {
  @apply bg-white dark:bg-gray-800 rounded-lg shadow-sm border border-gray-200 dark:border-gray-700 p-6;
}
</style>
