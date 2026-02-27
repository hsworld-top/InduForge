<template>
  <div class="dashboard-content p-6">
    <!-- 欢迎信息 -->
    <div class="mb-6">
      <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">
        {{ t('dashboard.welcomeBack', { username }) }}
      </h1>
    </div>

    <!-- 统计卡片 -->
    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
      <div
        v-if="canAccessUserManagement"
        :class="[
          'card transition-shadow duration-200',
          'cursor-pointer hover:shadow-lg',
        ]"
        @click="openTab('user-management')"
      >
        <div class="flex items-center">
          <div class="p-3 rounded-lg bg-green-100 dark:bg-green-900">
            <el-icon class="w-6 h-6 text-green-600 dark:text-green-400"><UserFilled /></el-icon>
          </div>
          <div class="ml-4">
            <p class="text-sm font-medium text-gray-600 dark:text-gray-400">{{ t('dashboard.userCount') }}</p>
            <p class="text-2xl font-semibold text-gray-900 dark:text-white">
              {{ loading ? '--' : stats.users }}
            </p>
          </div>
        </div>
      </div>

      <div
        v-if="canAccessProjectManagement"
        :class="[
          'card transition-shadow duration-200',
          'cursor-pointer hover:shadow-lg',
        ]"
        @click="openTab('project-management')"
      >
        <div class="flex items-center">
          <div class="p-3 rounded-lg bg-blue-100 dark:bg-blue-900">
            <el-icon class="w-6 h-6 text-blue-600 dark:text-blue-400"><FolderOpened /></el-icon>
          </div>
          <div class="ml-4">
            <p class="text-sm font-medium text-gray-600 dark:text-gray-400">{{ t('dashboard.projectCount') }}</p>
            <p class="text-2xl font-semibold text-gray-900 dark:text-white">
              {{ loading ? '--' : stats.projects }}
            </p>
          </div>
        </div>
      </div>

      <div
        v-if="canLoadTenantStats"
        class="card cursor-pointer hover:shadow-lg transition-shadow duration-200"
        @click="openTab('tenant-management')"
      >
        <div class="flex items-center">
          <div class="p-3 rounded-lg bg-yellow-100 dark:bg-yellow-900">
            <el-icon class="w-6 h-6 text-yellow-600 dark:text-yellow-400"><OfficeBuilding /></el-icon>
          </div>
          <div class="ml-4">
            <p class="text-sm font-medium text-gray-600 dark:text-gray-400">{{ t('dashboard.tenantCount') }}</p>
            <p class="text-2xl font-semibold text-gray-900 dark:text-white">
              {{ loading ? '--' : stats.tenants }}
            </p>
          </div>
        </div>
      </div>

      <div
        v-if="canAccessSystemLogs"
        :class="[
          'card transition-shadow duration-200',
          'cursor-pointer hover:shadow-lg',
        ]"
        @click="openTab('system-logs')"
      >
        <div class="flex items-center">
          <div class="p-3 rounded-lg bg-red-100 dark:bg-red-900">
            <el-icon class="w-6 h-6 text-red-600 dark:text-red-400"><DocumentCopy /></el-icon>
          </div>
          <div class="ml-4">
            <p class="text-sm font-medium text-gray-600 dark:text-gray-400">{{ t('dashboard.systemLogs') }}</p>
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
        <h3 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('dashboard.recentActivity') }}</h3>
        <el-button size="small" :loading="loading" @click="loadDashboardData">{{ t('common.refresh') }}</el-button>
      </div>
      <div
        v-if="loadError"
        class="mb-4 text-sm px-3 py-2 rounded border border-red-200 bg-red-50 text-red-600 dark:border-red-900/50 dark:bg-red-900/20 dark:text-red-300"
      >
        {{ loadError }}
      </div>
      <div v-if="loading" class="text-sm text-gray-500 dark:text-gray-400">{{ t('dashboard.loadingData') }}</div>
      <div v-else-if="recentActivities.length === 0" class="text-sm text-gray-500 dark:text-gray-400">
        {{ t('dashboard.noActivity') }}
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
              <el-icon class="w-4 h-4 text-blue-600 dark:text-blue-400"><InfoFilled /></el-icon>
            </div>
          </div>
          <div class="flex-1 min-w-0">
            <p class="text-sm text-gray-900 dark:text-white">{{ activity.description }}</p>
            <p class="text-xs text-gray-500 dark:text-gray-400">{{ formatDateTime(activity.time) }}</p>
          </div>
        </div>
      </div>
      <p class="mt-4 text-xs text-gray-500 dark:text-gray-400">
        {{ t('dashboard.lastUpdated', { time: formatDateTime(lastUpdatedAt) || '-' }) }}
      </p>
    </div>
  </div>
</template>

<script>
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  UserFilled,
  FolderOpened,
  OfficeBuilding,
  DocumentCopy,
  InfoFilled,
} from '@element-plus/icons-vue'
import { useAuthStore } from '@/store'
import { userAPI, projectAPI, tenantAPI, logAPI } from '@/api'
import { formatDateTime } from '@/utils/date'
import { canAccessTab, canRequestTenantStats } from '@/permissions'

export default {
  name: 'DashboardContent',
  emits: ['open-tab'],
  setup(props, { emit }) {
    const authStore = useAuthStore()
    const { t } = useI18n()

    const username = computed(() => authStore.userInfo?.username || '')
    const role = computed(() => authStore.userInfo?.role)
    const canLoadTenantStats = computed(() => canRequestTenantStats(role.value))
    const canAccessUserManagement = computed(() => canAccessTab('user-management', role.value))
    const canAccessProjectManagement = computed(() => canAccessTab('project-management', role.value))
    const canAccessSystemLogs = computed(() => canAccessTab('system-logs', role.value))
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
        if (canLoadTenantStats.value) {
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
          canLoadTenantStats.value && tenantsResult?.status === 'fulfilled'
            ? tenantsResult.value?.pagination?.total || 0
            : 0

        if (logsResult?.status === 'fulfilled') {
          const logs = logsResult.value?.data?.logs || []
          recentActivities.value = logs.slice(0, 5).map((log, index) => ({
            id: log.id || `${log.createdAt}-${index}`,
            description:
              log.message ||
              t('dashboard.activityFallback', {
                action: log.action || t('dashboard.systemAction'),
              }),
            time: log.createdAt || '',
          }))
        } else {
          recentActivities.value = []
        }

        const failedEntries = requestEntries
          .map(([key], index) => ({ key, result: results[index] }))
          .filter((item) => item.result.status === 'rejected')
        const failedCount = failedEntries.length
        if (failedCount > 0) {
          const sourceLabelMap = {
            users: t('dashboard.userCount'),
            projects: t('dashboard.projectCount'),
            logs: t('dashboard.systemLogs'),
            tenants: t('dashboard.tenantCount'),
          }
          const failedSources = failedEntries.map((item) => sourceLabelMap[item.key] || item.key)
          loadError.value = t('dashboard.partialLoadFailed', {
            failed: failedCount,
            total: results.length,
            sources: failedSources.join('、'),
          })
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
        loadError.value = t('dashboard.loadFailed')
      } finally {
        loading.value = false
      }
    }

    // 点击卡片打开对应标签页
    const openTab = (tabKey) => {
      emit('open-tab', tabKey)
    }

    onMounted(() => {
      loadDashboardData()
    })

    return {
      username,
      canLoadTenantStats,
      canAccessUserManagement,
      canAccessProjectManagement,
      canAccessSystemLogs,
      loading,
      loadError,
      lastUpdatedAt,
      stats,
      recentActivities,
      t,
      UserFilled,
      FolderOpened,
      OfficeBuilding,
      DocumentCopy,
      InfoFilled,
      openTab,
      loadDashboardData,
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
