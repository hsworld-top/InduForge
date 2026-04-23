<template>
  <div class="dashboard-content h-full flex flex-col">
    <!-- 一体化操作栏 (Cockpit-style) -->
    <div
      class="flex items-center justify-between bg-white dark:bg-gray-800 rounded-2xl shadow-sm border border-gray-100 dark:border-gray-700 p-3 mb-6"
    >
      <!-- 左侧：标题与欢迎语 -->
      <div class="flex items-center space-x-4">
        <h1
          class="text-[16px] font-semibold text-gray-800 dark:text-gray-100 ml-2 whitespace-nowrap"
        >
          {{ t('dashboard.title') }}
        </h1>
        <div class="h-5 w-px bg-gray-200 dark:bg-gray-700 mx-1 hidden md:block"></div>
        <span class="text-sm text-gray-500 dark:text-gray-400 hidden md:inline">
          {{ t('dashboard.welcomeBack', { username }) }}
        </span>
      </div>

      <!-- 右侧：刷新 -->
      <div class="flex items-center space-x-3">
        <el-tooltip :content="t('common.refresh')" placement="top">
          <button
            class="w-9 h-9 rounded-full bg-gray-50 dark:bg-gray-700 border border-gray-200 dark:border-gray-600 flex items-center justify-center text-gray-600 dark:text-gray-300 hover:bg-white dark:hover:bg-gray-600 transition-colors shadow-sm"
            :disabled="loading"
            @click="loadDashboardData"
          >
            <el-icon :class="{ 'animate-spin': loading }"><RefreshRight /></el-icon>
          </button>
        </el-tooltip>
      </div>
    </div>

    <!-- 错误提示 -->
    <div v-if="loadError" class="mb-4">
      <el-alert :title="loadError" type="warning" show-icon :closable="true" />
    </div>

    <!-- 统计卡片 -->
    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-5 mb-6">
      <!-- 用户统计 -->
      <div
        v-if="canAccessUserManagement"
        class="stat-card group cursor-pointer"
        @click="openTab('user-management')"
      >
        <div class="flex items-center justify-between">
          <div>
            <p class="text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider mb-1">
              {{ t('dashboard.userCount') }}
            </p>
            <p class="text-2xl font-bold text-gray-900 dark:text-white">
              {{ loading ? '--' : stats.users }}
            </p>
          </div>
          <div
            class="w-12 h-12 rounded-2xl bg-gradient-to-br from-emerald-400 to-emerald-600 flex items-center justify-center shadow-lg shadow-emerald-500/20 group-hover:scale-110 transition-transform duration-300"
          >
            <el-icon class="w-6 h-6 text-white"><UserFilled /></el-icon>
          </div>
        </div>
        <div class="mt-3 flex items-center text-xs text-gray-400 dark:text-gray-500">
          <svg class="w-3.5 h-3.5 mr-1" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M13 7l5 5m0 0l-5 5m5-5H6" />
          </svg>
          {{ t('dashboard.clickToView') }}
        </div>
      </div>

      <!-- 工程统计 -->
      <div
        v-if="canAccessProjectManagement"
        class="stat-card group cursor-pointer"
        @click="openTab('project-management')"
      >
        <div class="flex items-center justify-between">
          <div>
            <p class="text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider mb-1">
              {{ t('dashboard.projectCount') }}
            </p>
            <p class="text-2xl font-bold text-gray-900 dark:text-white">
              {{ loading ? '--' : stats.projects }}
            </p>
          </div>
          <div
            class="w-12 h-12 rounded-2xl bg-gradient-to-br from-blue-400 to-blue-600 flex items-center justify-center shadow-lg shadow-blue-500/20 group-hover:scale-110 transition-transform duration-300"
          >
            <el-icon class="w-6 h-6 text-white"><FolderOpened /></el-icon>
          </div>
        </div>
        <div class="mt-3 flex items-center text-xs text-gray-400 dark:text-gray-500">
          <svg class="w-3.5 h-3.5 mr-1" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M13 7l5 5m0 0l-5 5m5-5H6" />
          </svg>
          {{ t('dashboard.clickToView') }}
        </div>
      </div>

      <!-- 租户统计 -->
      <div
        v-if="canLoadTenantStats"
        class="stat-card group cursor-pointer"
        @click="openTab('tenant-management')"
      >
        <div class="flex items-center justify-between">
          <div>
            <p class="text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider mb-1">
              {{ t('dashboard.tenantCount') }}
            </p>
            <p class="text-2xl font-bold text-gray-900 dark:text-white">
              {{ loading ? '--' : stats.tenants }}
            </p>
          </div>
          <div
            class="w-12 h-12 rounded-2xl bg-gradient-to-br from-amber-400 to-amber-600 flex items-center justify-center shadow-lg shadow-amber-500/20 group-hover:scale-110 transition-transform duration-300"
          >
            <el-icon class="w-6 h-6 text-white"><OfficeBuilding /></el-icon>
          </div>
        </div>
        <div class="mt-3 flex items-center text-xs text-gray-400 dark:text-gray-500">
          <svg class="w-3.5 h-3.5 mr-1" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M13 7l5 5m0 0l-5 5m5-5H6" />
          </svg>
          {{ t('dashboard.clickToView') }}
        </div>
      </div>

      <!-- 系统日志统计 -->
      <div
        v-if="canAccessSystemLogs"
        class="stat-card group cursor-pointer"
        @click="openTab('system-logs')"
      >
        <div class="flex items-center justify-between">
          <div>
            <p class="text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider mb-1">
              {{ t('dashboard.systemLogs') }}
            </p>
            <p class="text-2xl font-bold text-gray-900 dark:text-white">
              {{ loading ? '--' : stats.logs }}
            </p>
          </div>
          <div
            class="w-12 h-12 rounded-2xl bg-gradient-to-br from-rose-400 to-rose-600 flex items-center justify-center shadow-lg shadow-rose-500/20 group-hover:scale-110 transition-transform duration-300"
          >
            <el-icon class="w-6 h-6 text-white"><DocumentCopy /></el-icon>
          </div>
        </div>
        <div class="mt-3 flex items-center text-xs text-gray-400 dark:text-gray-500">
          <svg class="w-3.5 h-3.5 mr-1" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M13 7l5 5m0 0l-5 5m5-5H6" />
          </svg>
          {{ t('dashboard.clickToView') }}
        </div>
      </div>
    </div>

    <!-- 最近活动 -->
    <div class="flex-1 min-h-0">
      <div
        class="bg-white dark:bg-gray-800 rounded-2xl shadow-sm border border-gray-100 dark:border-gray-700 p-5 h-full flex flex-col"
      >
        <div class="flex items-center justify-between mb-5">
          <h3 class="text-[15px] font-semibold text-gray-800 dark:text-gray-100 flex items-center gap-2">
            <div class="w-1.5 h-5 bg-blue-500 rounded-full"></div>
            {{ t('dashboard.recentActivity') }}
          </h3>
          <span class="text-[11px] text-gray-400 dark:text-gray-500 font-mono">
            {{ t('dashboard.lastUpdated', { time: formatDateTime(lastUpdatedAt) || '-' }) }}
          </span>
        </div>

        <div v-if="loading" class="flex-1 flex items-center justify-center">
          <div class="text-sm text-gray-400 dark:text-gray-500 flex items-center gap-2">
            <el-icon class="animate-spin"><RefreshRight /></el-icon>
            {{ t('dashboard.loadingData') }}
          </div>
        </div>

        <div
          v-else-if="recentActivities.length === 0"
          class="flex-1 flex items-center justify-center"
        >
          <div class="text-center">
            <div class="w-16 h-16 mx-auto mb-3 bg-gray-100 dark:bg-gray-700 rounded-2xl flex items-center justify-center">
              <svg class="w-8 h-8 text-gray-300 dark:text-gray-600" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
                <path stroke-linecap="round" stroke-linejoin="round" d="M19.5 14.25v-2.625a3.375 3.375 0 00-3.375-3.375h-1.5A1.125 1.125 0 0113.5 7.125v-1.5a3.375 3.375 0 00-3.375-3.375H8.25m0 12.75h7.5m-7.5 3H12M10.5 2.25H5.625c-.621 0-1.125.504-1.125 1.125v17.25c0 .621.504 1.125 1.125 1.125h12.75c.621 0 1.125-.504 1.125-1.125V11.25a9 9 0 00-9-9z" />
              </svg>
            </div>
            <p class="text-sm text-gray-400 dark:text-gray-500">{{ t('dashboard.noActivity') }}</p>
          </div>
        </div>

        <div v-else class="flex-1 overflow-y-auto space-y-3">
          <div
            v-for="(activity, index) in recentActivities"
            :key="activity.id"
            class="flex items-start gap-3 p-3 rounded-xl bg-gray-50/60 dark:bg-gray-900/30 border border-gray-100/80 dark:border-gray-800/60 hover:border-gray-200 dark:hover:border-gray-700 transition-colors"
          >
            <div class="flex-shrink-0 relative">
              <div
                class="w-8 h-8 rounded-lg flex items-center justify-center text-xs font-bold"
                :class="getActivityIconClass(index)"
              >
                {{ index + 1 }}
              </div>
            </div>
            <div class="flex-1 min-w-0">
              <p class="text-[13px] text-gray-700 dark:text-gray-300 leading-relaxed line-clamp-2">
                {{ activity.description }}
              </p>
              <p class="text-[11px] text-gray-400 dark:text-gray-500 mt-1 font-mono">
                {{ formatDateTime(activity.time) }}
              </p>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts">
// @ts-nocheck
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  UserFilled,
  FolderOpened,
  OfficeBuilding,
  DocumentCopy,
  InfoFilled,
  RefreshRight,
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
    const canAccessProjectManagement = computed(() =>
      canAccessTab('project-management', role.value),
    )
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

    /**
     * 活动条目图标颜色，按序号循环使用不同颜色。
     */
    const getActivityIconClass = (index) => {
      const colors = [
        'bg-blue-100 text-blue-600 dark:bg-blue-900/40 dark:text-blue-400',
        'bg-emerald-100 text-emerald-600 dark:bg-emerald-900/40 dark:text-emerald-400',
        'bg-amber-100 text-amber-600 dark:bg-amber-900/40 dark:text-amber-400',
        'bg-rose-100 text-rose-600 dark:bg-rose-900/40 dark:text-rose-400',
        'bg-purple-100 text-purple-600 dark:bg-purple-900/40 dark:text-purple-400',
      ]
      return colors[index % colors.length]
    }

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
        } else {
          requestEntries.push(['recentActivities', logAPI.getRecentActivities({ limit: 5 })])
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
        const recentActivitiesResult = resultMap.recentActivities
        const tenantsResult = resultMap.tenants

        const getPaginationTotal = (result) =>
          result?.data?.pagination?.total || result?.pagination?.total || 0

        stats.value.users = usersResult?.status === 'fulfilled' ? getPaginationTotal(usersResult.value) : 0
        stats.value.projects =
          projectsResult?.status === 'fulfilled' ? getPaginationTotal(projectsResult.value) : 0
        stats.value.logs = logsResult?.status === 'fulfilled' ? getPaginationTotal(logsResult.value) : 0
        stats.value.tenants =
          canLoadTenantStats.value && tenantsResult?.status === 'fulfilled'
            ? getPaginationTotal(tenantsResult.value)
            : 0

        if (logsResult?.status === 'fulfilled') {
          const logs = logsResult.value?.data?.list?.logs || logsResult.value?.data?.logs || []
          recentActivities.value = logs.slice(0, 5).map((log, index) => ({
            id: log.id || `${log.createdAt}-${index}`,
            description:
              log.message ||
              t('dashboard.activityFallback', {
                action: log.action || t('dashboard.systemAction'),
              }),
            time: log.createdAt || '',
          }))
        } else if (recentActivitiesResult?.status === 'fulfilled') {
          const activities =
            recentActivitiesResult.value?.data?.list?.activities ||
            recentActivitiesResult.value?.data?.activities ||
            []
          recentActivities.value = activities.slice(0, 5).map((log, index) => ({
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
            recentActivities: t('dashboard.recentActivity'),
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
      RefreshRight,
      openTab,
      loadDashboardData,
      formatDateTime,
      getActivityIconClass,
    }
  },
}
</script>

<style scoped>
.dashboard-content {
  padding: 20px;
  background-color: #f8fafc;
}
.dark .dashboard-content {
  background-color: #0f172a;
}

/* 统计卡片 */
.stat-card {
  @apply bg-white dark:bg-gray-800 rounded-2xl p-5 border border-gray-200/80 dark:border-gray-700 shadow-[0_2px_12px_rgba(0,0,0,0.04)] hover:shadow-[0_8px_24px_rgba(0,0,0,0.08)] hover:-translate-y-0.5 transition-all duration-300;
}
</style>
