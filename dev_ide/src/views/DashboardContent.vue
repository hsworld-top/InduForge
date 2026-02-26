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
        :class="[
          'card transition-shadow duration-200',
          canAccessUserManagement ? 'cursor-pointer hover:shadow-lg' : 'opacity-60 cursor-not-allowed',
        ]"
        @click="handleCardClick('user-management', canAccessUserManagement)"
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
            <p class="text-2xl font-semibold text-gray-900 dark:text-white">
              {{ loading ? '--' : stats.users }}
            </p>
          </div>
        </div>
      </div>

      <div
        :class="[
          'card transition-shadow duration-200',
          canAccessProjectManagement
            ? 'cursor-pointer hover:shadow-lg'
            : 'opacity-60 cursor-not-allowed',
        ]"
        @click="handleCardClick('project-management', canAccessProjectManagement)"
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
              {{ loading ? '--' : stats.projects }}
            </p>
          </div>
        </div>
      </div>

      <div
        v-if="isSuperAdmin"
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
              {{ loading ? '--' : stats.tenants }}
            </p>
          </div>
        </div>
      </div>

      <div
        :class="[
          'card transition-shadow duration-200',
          canAccessSystemLogs ? 'cursor-pointer hover:shadow-lg' : 'opacity-60 cursor-not-allowed',
        ]"
        @click="handleCardClick('system-logs', canAccessSystemLogs)"
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
            <p class="text-2xl font-semibold text-gray-900 dark:text-white">
              {{ loading ? '--' : stats.logs }}
            </p>
          </div>
        </div>
      </div>
    </div>

    <!-- 最近活动 -->
    <div class="card">
      <div class="flex items-center justify-between mb-4">
        <h3 class="text-lg font-semibold text-gray-900 dark:text-white">最近活动</h3>
        <el-button size="small" :loading="loading" @click="loadDashboardData">刷新数据</el-button>
      </div>
      <div
        v-if="loadError"
        class="mb-4 text-sm px-3 py-2 rounded border border-red-200 bg-red-50 text-red-600 dark:border-red-900/50 dark:bg-red-900/20 dark:text-red-300"
      >
        {{ loadError }}
      </div>
      <div v-if="loading" class="text-sm text-gray-500 dark:text-gray-400">数据加载中...</div>
      <div v-else-if="recentActivities.length === 0" class="text-sm text-gray-500 dark:text-gray-400">
        暂无活动记录
      </div>
      <div v-else class="space-y-4">
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
            <p class="text-xs text-gray-500 dark:text-gray-400">{{ formatDateTime(activity.time) }}</p>
          </div>
        </div>
      </div>
      <p class="mt-4 text-xs text-gray-500 dark:text-gray-400">
        最近更新时间：{{ formatDateTime(lastUpdatedAt) || '-' }}
      </p>
    </div>
  </div>
</template>

<script>
import { ref, computed, onMounted } from 'vue'
import { useAuthStore } from '@/store'
import { userAPI, projectAPI, tenantAPI, logAPI } from '@/api'
import { formatDateTime } from '@/utils/date'
import { ElMessage } from 'element-plus'

export default {
  name: 'DashboardContent',
  emits: ['open-tab'],
  setup(props, { emit }) {
    const authStore = useAuthStore()

    const username = computed(() => authStore.userInfo?.username || '')
    const role = computed(() => authStore.userInfo?.role)
    const isSuperAdmin = computed(() => role.value === 'SUPER_ADMIN')
    const canAccessUserManagement = computed(
      () => role.value === 'SUPER_ADMIN' || role.value === 'SYSTEM_ADMIN'
    )
    const canAccessProjectManagement = computed(
      () =>
        role.value === 'SUPER_ADMIN' ||
        role.value === 'SYSTEM_ADMIN' ||
        role.value === 'PROJECT_ADMIN'
    )
    const canAccessSystemLogs = computed(
      () => role.value === 'SUPER_ADMIN' || role.value === 'SYSTEM_ADMIN' || role.value === 'OPS_ADMIN'
    )
    const loading = ref(false)
    const loadError = ref('')
    const lastUpdatedAt = ref('')
    const stats = ref({
      projects: 0,
      users: 0,
      tenants: 0,
      logs: 0,
    })

    const recentActivities = ref([])

    const loadDashboardData = async () => {
      loading.value = true
      loadError.value = ''
      try {
        const requestEntries = []
        if (canAccessUserManagement.value) {
          requestEntries.push(['users', userAPI.getUsers({ page: 1, limit: 1 })])
        }
        if (canAccessProjectManagement.value) {
          requestEntries.push(['projects', projectAPI.getProjects({ page: 1, limit: 1 })])
        }
        if (canAccessSystemLogs.value) {
          requestEntries.push(['logs', logAPI.getLogs({ page: 1, limit: 5 })])
        }
        if (isSuperAdmin.value) {
          requestEntries.push(['tenants', tenantAPI.getTenants({ page: 1, limit: 1 })])
        }

        if (requestEntries.length === 0) {
          stats.value = { projects: 0, users: 0, tenants: 0, logs: 0 }
          recentActivities.value = []
          lastUpdatedAt.value = new Date().toISOString()
          return
        }

        const results = await Promise.allSettled(requestEntries.map((entry) => entry[1]))
        const resultMap = {}
        requestEntries.forEach(([key], index) => {
          resultMap[key] = results[index]
        })

        const usersResult = resultMap.users
        const projectsResult = resultMap.projects
        const logsResult = resultMap.logs
        const tenantsResult = resultMap.tenants

        stats.value.users =
          usersResult?.status === 'fulfilled' ? usersResult.value?.pagination?.total || 0 : 0
        stats.value.projects =
          projectsResult?.status === 'fulfilled'
            ? projectsResult.value?.pagination?.total || 0
            : 0
        stats.value.logs =
          logsResult?.status === 'fulfilled' ? logsResult.value?.pagination?.total || 0 : 0
        stats.value.tenants =
          isSuperAdmin.value && tenantsResult?.status === 'fulfilled'
            ? tenantsResult.value?.pagination?.total || 0
            : 0

        if (logsResult?.status === 'fulfilled') {
          const logs = logsResult.value?.data?.logs || []
          recentActivities.value = logs.slice(0, 5).map((log, index) => ({
            id: log.id || `${log.createdAt}-${index}`,
            description: log.message || `${log.action || '系统'} 操作`,
            time: log.createdAt || '',
          }))
        } else {
          recentActivities.value = []
        }

        const failedCount = results.filter((item) => item.status === 'rejected').length
        if (failedCount > 0) {
          loadError.value = `部分数据加载失败（${failedCount}/${results.length}），已展示可用数据。`
        }
        lastUpdatedAt.value = new Date().toISOString()
      } catch {
        stats.value = {
          projects: 0,
          users: 0,
          tenants: 0,
          logs: 0,
        }
        recentActivities.value = []
        loadError.value = '仪表盘数据加载失败，请稍后重试。'
      } finally {
        loading.value = false
      }
    }

    // 点击卡片打开对应标签页
    const openTab = (tabKey) => {
      emit('open-tab', tabKey)
    }

    const handleCardClick = (tabKey, canAccess) => {
      if (!canAccess) {
        ElMessage.warning('您当前角色无权访问该功能')
        return
      }
      openTab(tabKey)
    }

    onMounted(() => {
      loadDashboardData()
    })

    return {
      username,
      isSuperAdmin,
      canAccessUserManagement,
      canAccessProjectManagement,
      canAccessSystemLogs,
      loading,
      loadError,
      lastUpdatedAt,
      stats,
      recentActivities,
      openTab,
      loadDashboardData,
      handleCardClick,
      formatDateTime,
    }
  },
}
</script>

<style scoped>
.card {
  @apply bg-white dark:bg-gray-800 rounded-lg shadow-sm border border-gray-200 dark:border-gray-700 p-6;
}
</style>
