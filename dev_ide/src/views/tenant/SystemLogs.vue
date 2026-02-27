<template>
  <div class="system-logs">
    <div class="flex justify-between items-center mb-6">
      <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">{{ t('systemLogs.title') }}</h1>
      <el-button @click="handleRefresh" :loading="loading">
        <el-icon><Refresh /></el-icon>
        {{ t('systemLogs.refresh') }}
      </el-button>
    </div>

    <div class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-4 gap-4 mb-6">
      <div class="stat-card">
        <div class="stat-label">{{ t('systemLogs.levelError') }}</div>
        <div class="stat-value text-red-600 dark:text-red-400">{{ levelStats.error || 0 }}</div>
      </div>
      <div class="stat-card">
        <div class="stat-label">{{ t('systemLogs.levelWarning') }}</div>
        <div class="stat-value text-orange-600 dark:text-orange-400">
          {{ levelStats.warning || 0 }}
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-label">{{ t('systemLogs.levelInfo') }}</div>
        <div class="stat-value text-blue-600 dark:text-blue-400">{{ levelStats.info || 0 }}</div>
      </div>
      <div class="stat-card">
        <div class="stat-label">{{ t('systemLogs.levelDebug') }}</div>
        <div class="stat-value text-gray-700 dark:text-gray-300">{{ levelStats.debug || 0 }}</div>
      </div>
    </div>

    <div class="panel mb-6">
      <el-form :inline="true" :model="filters" class="flex flex-wrap gap-3">
        <el-form-item :label="t('systemLogs.level')">
          <el-select v-model="filters.level" :placeholder="t('systemLogs.allLevel')" clearable style="width: 140px">
            <el-option :label="t('systemLogs.levelError')" value="error" />
            <el-option :label="t('systemLogs.levelWarning')" value="warning" />
            <el-option :label="t('systemLogs.levelInfo')" value="info" />
            <el-option :label="t('systemLogs.levelDebug')" value="debug" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('systemLogs.action')">
          <el-input
            v-model="filters.action"
            :placeholder="t('systemLogs.actionPlaceholder')"
            clearable
            style="width: 180px"
            @keyup.enter="handleSearch"
          />
        </el-form-item>
        <el-form-item :label="t('systemLogs.resource')">
          <el-input
            v-model="filters.resource"
            :placeholder="t('systemLogs.resourcePlaceholder')"
            clearable
            style="width: 180px"
            @keyup.enter="handleSearch"
          />
        </el-form-item>
        <el-form-item :label="t('systemLogs.timeRange')">
          <el-date-picker
            v-model="filters.dateRange"
            type="datetimerange"
            value-format="YYYY-MM-DD HH:mm:ss"
            :range-separator="t('systemLogs.rangeTo')"
            :start-placeholder="t('systemLogs.startTime')"
            :end-placeholder="t('systemLogs.endTime')"
            style="width: 360px"
          />
        </el-form-item>
        <el-form-item>
          <el-button text @click="applyQuickRange('today')">{{ t('systemLogs.today') }}</el-button>
          <el-button text @click="applyQuickRange('last7')">{{ t('systemLogs.last7Days') }}</el-button>
          <el-button text @click="applyQuickRange('last30')">{{ t('systemLogs.last30Days') }}</el-button>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="handleSearch">{{ t('systemLogs.query') }}</el-button>
          <el-button @click="resetFilters">{{ t('systemLogs.reset') }}</el-button>
        </el-form-item>
        <el-form-item :label="t('systemLogs.filterView')">
          <el-select
            v-model="selectedViewId"
            :placeholder="t('systemLogs.selectSavedView')"
            clearable
            style="width: 220px"
            @change="handleApplySavedView"
          >
            <el-option
              v-for="view in sortedSavedViews"
              :key="view.id"
              :label="view.name"
              :value="view.id"
            />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button type="success" plain @click="saveCurrentView">{{ t('systemLogs.saveCurrentFilter') }}</el-button>
          <el-button type="danger" plain :disabled="!selectedViewId" @click="removeSelectedView">
            {{ t('systemLogs.deleteSelectedView') }}
          </el-button>
          <el-button type="primary" plain @click="handleExportCurrent">{{ t('systemLogs.exportCurrentResult') }}</el-button>
        </el-form-item>
      </el-form>
    </div>

    <div class="panel">
      <div v-if="loadError" class="px-4 pt-4">
        <el-alert :title="loadError" type="error" show-icon :closable="false">
          <template #default>
            <el-button text type="primary" @click="handleRefresh">{{ t('systemLogs.reload') }}</el-button>
          </template>
        </el-alert>
      </div>
      <el-table
        :data="logs"
        v-loading="loading"
        style="width: 100%"
        :header-cell-style="{ background: '#f9fafb', color: '#374151' }"
      >
        <el-table-column prop="createdAt" :label="t('systemLogs.time')" width="180">
          <template #default="scope">
            {{ formatDateTime(scope.row.createdAt) }}
          </template>
        </el-table-column>
        <el-table-column prop="level" :label="t('systemLogs.level')" width="90">
          <template #default="scope">
            <el-tag :type="getLevelTagType(scope.row.level)" size="small">
              {{ getLevelLabel(scope.row.level) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="action" :label="t('systemLogs.action')" width="180" show-overflow-tooltip />
        <el-table-column prop="resource" :label="t('systemLogs.resource')" width="180" show-overflow-tooltip />
        <el-table-column prop="message" :label="t('systemLogs.logContent')" min-width="280" show-overflow-tooltip />
        <el-table-column prop="ip" :label="t('systemLogs.ip')" width="140" />
        <el-table-column :label="t('systemLogs.user')" width="140">
          <template #default="scope">
            {{ scope.row.user?.fullName || scope.row.user?.username || '-' }}
          </template>
        </el-table-column>
        <el-table-column :label="t('systemLogs.tenant')" width="140">
          <template #default="scope">
            {{ scope.row.tenant?.name || '-' }}
          </template>
        </el-table-column>
      </el-table>

      <div class="flex justify-end items-center p-4 border-t border-gray-200 dark:border-gray-700">
        <el-pagination
          v-model:current-page="pagination.page"
          v-model:page-size="pagination.limit"
          :page-sizes="systemLogPageSizeOptions"
          :total="pagination.total"
          layout="total, sizes, prev, pager, next, jumper"
          @size-change="handleSizeChange"
          @current-change="handleCurrentChange"
        />
      </div>
    </div>
  </div>
</template>

<script>
import { ref, reactive, onMounted, computed } from 'vue'
import dayjs from 'dayjs'
import { useI18n } from 'vue-i18n'
import { ElMessage, ElMessageBox } from 'element-plus'
import { logAPI } from '@/api'
import { formatDateTime } from '@/utils/date'
import { Storage } from '@/utils/storage'
import { STORAGE_KEYS, SYSTEM_LOG_PAGE_SIZE_OPTIONS } from '@/constants'

export default {
  name: 'SystemLogs',
  setup() {
    const { t } = useI18n()
    const loading = ref(false)
    const loadError = ref('')
    const logs = ref([])
    const levelStats = ref({})

    const filters = reactive({
      level: '',
      action: '',
      resource: '',
      dateRange: [],
    })
    const savedViews = ref(Storage.get(STORAGE_KEYS.SYSTEM_LOG_SAVED_VIEWS, []))
    const selectedViewId = ref('')
    const sortedSavedViews = computed(() =>
      [...savedViews.value].sort(
        (a, b) => new Date(b.updatedAt || 0) - new Date(a.updatedAt || 0)
      )
    )

    const pagination = reactive({
      page: 1,
      limit: Storage.get('system_log_page_size', 20),
      total: 0,
      totalPages: 0,
    })

    const getQueryParams = () => {
      const [startDate, endDate] = Array.isArray(filters.dateRange) ? filters.dateRange : []
      const params = {
        page: pagination.page,
        limit: pagination.limit,
        level: filters.level || undefined,
        action: filters.action || undefined,
        resource: filters.resource || undefined,
        startDate: startDate || undefined,
        endDate: endDate || undefined,
      }
      Object.keys(params).forEach((key) => {
        if (params[key] === undefined) delete params[key]
      })
      return params
    }

    const fetchLogs = async () => {
      loading.value = true
      loadError.value = ''
      try {
        const response = await logAPI.getLogs(getQueryParams())
        logs.value = response.data?.logs || []
        pagination.total = response.pagination?.total || 0
        pagination.totalPages = response.pagination?.totalPages || 0
      } catch (error) {
        loadError.value = error.response?.data?.message || t('systemLogs.fetchFailedRetry')
        ElMessage.error(
          t('systemLogs.fetchFailed', {
            message: error.response?.data?.message || error.message,
          })
        )
      } finally {
        loading.value = false
      }
    }

    const fetchStats = async () => {
      try {
        const [startDate, endDate] = Array.isArray(filters.dateRange) ? filters.dateRange : []
        const response = await logAPI.getLogStats({
          startDate: startDate || undefined,
          endDate: endDate || undefined,
        })
        levelStats.value = response.data?.levelStats || {}
      } catch {
        levelStats.value = {}
      }
    }

    const handleSearch = async () => {
      pagination.page = 1
      await Promise.all([fetchLogs(), fetchStats()])
    }

    const resetFilters = async () => {
      filters.level = ''
      filters.action = ''
      filters.resource = ''
      filters.dateRange = []
      pagination.page = 1
      await Promise.all([fetchLogs(), fetchStats()])
    }

    const handleRefresh = async () => {
      await Promise.all([fetchLogs(), fetchStats()])
    }

    const applyQuickRange = async (type) => {
      const now = dayjs()
      if (type === 'today') {
        filters.dateRange = [
          now.startOf('day').format('YYYY-MM-DD HH:mm:ss'),
          now.endOf('day').format('YYYY-MM-DD HH:mm:ss'),
        ]
      } else if (type === 'last7') {
        filters.dateRange = [
          now.subtract(7, 'day').startOf('day').format('YYYY-MM-DD HH:mm:ss'),
          now.endOf('day').format('YYYY-MM-DD HH:mm:ss'),
        ]
      } else if (type === 'last30') {
        filters.dateRange = [
          now.subtract(30, 'day').startOf('day').format('YYYY-MM-DD HH:mm:ss'),
          now.endOf('day').format('YYYY-MM-DD HH:mm:ss'),
        ]
      }
      await handleSearch()
    }

    const handleExportCurrent = async () => {
      try {
        const params = getQueryParams()
        delete params.page
        delete params.limit

        const response = await logAPI.exportLogs(params)
        const blob =
          response instanceof window.Blob
            ? response
            : new window.Blob([response], { type: 'text/csv;charset=utf-8;' })
        const url = window.URL.createObjectURL(blob)
        const link = document.createElement('a')
        link.href = url
        link.download = `system-logs-${Date.now()}.csv`
        link.click()
        window.URL.revokeObjectURL(url)
        ElMessage.success(t('systemLogs.exportSuccess'))
      } catch (error) {
        ElMessage.error(
          t('systemLogs.exportFailed', {
            message: error.response?.data?.message || error.message,
          })
        )
      }
    }

    const saveCurrentView = async () => {
      const { value: name } = await ElMessageBox.prompt(
        t('systemLogs.promptViewName'),
        t('systemLogs.promptViewTitle'),
        {
          confirmButtonText: t('systemLogs.save'),
          cancelButtonText: t('systemLogs.cancel'),
        inputPattern: /^.{1,20}$/,
          inputErrorMessage: t('systemLogs.nameLengthError'),
        }
      ).catch(() => ({ value: '' }))

      if (!name) return

      const currentFilters = {
        level: filters.level,
        action: filters.action,
        resource: filters.resource,
        dateRange: Array.isArray(filters.dateRange) ? [...filters.dateRange] : [],
      }

      const existingIndex = savedViews.value.findIndex((item) => item.name === name)
      const newView = {
        id: existingIndex > -1 ? savedViews.value[existingIndex].id : `view_${Date.now()}`,
        name,
        filters: currentFilters,
        updatedAt: new Date().toISOString(),
      }

      if (existingIndex > -1) {
        savedViews.value.splice(existingIndex, 1, newView)
      } else {
        savedViews.value.push(newView)
        if (savedViews.value.length > 20) {
          savedViews.value = savedViews.value
            .sort((a, b) => new Date(b.updatedAt || 0) - new Date(a.updatedAt || 0))
            .slice(0, 20)
          ElMessage.warning(t('systemLogs.limitViewsWarning'))
        }
      }

      Storage.set(STORAGE_KEYS.SYSTEM_LOG_SAVED_VIEWS, savedViews.value)
      selectedViewId.value = newView.id
      ElMessage.success(existingIndex > -1 ? t('systemLogs.viewUpdated') : t('systemLogs.viewSaved'))
    }

    const handleApplySavedView = async (viewId, silent = false) => {
      if (!viewId) {
        Storage.remove(STORAGE_KEYS.SYSTEM_LOG_LAST_VIEW_ID)
        return
      }
      const targetViewIndex = savedViews.value.findIndex((item) => item.id === viewId)
      const targetView = targetViewIndex > -1 ? savedViews.value[targetViewIndex] : null
      if (!targetView) return

      filters.level = targetView.filters.level || ''
      filters.action = targetView.filters.action || ''
      filters.resource = targetView.filters.resource || ''
      filters.dateRange = Array.isArray(targetView.filters.dateRange)
        ? [...targetView.filters.dateRange]
        : []
      savedViews.value[targetViewIndex] = {
        ...targetView,
        updatedAt: new Date().toISOString(),
      }
      Storage.set(STORAGE_KEYS.SYSTEM_LOG_SAVED_VIEWS, savedViews.value)
      pagination.page = 1
      Storage.set(STORAGE_KEYS.SYSTEM_LOG_LAST_VIEW_ID, viewId)

      await Promise.all([fetchLogs(), fetchStats()])
      if (!silent) {
        ElMessage.success(t('systemLogs.viewApplied', { name: targetView.name }))
      }
    }

    const removeSelectedView = async () => {
      const targetView = savedViews.value.find((item) => item.id === selectedViewId.value)
      if (!targetView) return

      try {
        await ElMessageBox.confirm(
          t('systemLogs.removeViewConfirm', { name: targetView.name }),
          t('systemLogs.removeViewTitle'),
          {
          confirmButtonText: t('systemLogs.remove'),
          cancelButtonText: t('systemLogs.cancel'),
          type: 'warning',
          }
        )

        savedViews.value = savedViews.value.filter((item) => item.id !== selectedViewId.value)
        Storage.set(STORAGE_KEYS.SYSTEM_LOG_SAVED_VIEWS, savedViews.value)
        Storage.remove(STORAGE_KEYS.SYSTEM_LOG_LAST_VIEW_ID)
        selectedViewId.value = ''
        ElMessage.success(t('systemLogs.viewDeleted'))
      } catch (error) {
        if (error !== 'cancel') {
          ElMessage.error(t('systemLogs.viewDeleteFailed'))
        }
      }
    }

    const handleSizeChange = async (size) => {
      pagination.limit = size
      Storage.set('system_log_page_size', size)
      pagination.page = 1
      await fetchLogs()
    }

    const handleCurrentChange = async (page) => {
      pagination.page = page
      await fetchLogs()
    }

    const getLevelTagType = (level) => {
      const map = {
        error: 'danger',
        warning: 'warning',
        info: 'primary',
        debug: 'info',
      }
      return map[level] || 'info'
    }

    const getLevelLabel = (level) => {
      const map = {
        error: t('systemLogs.levelError'),
        warning: t('systemLogs.levelWarning'),
        info: t('systemLogs.levelInfo'),
        debug: t('systemLogs.levelDebug'),
      }
      return map[level] || level || '-'
    }

    onMounted(async () => {
      await Promise.all([fetchLogs(), fetchStats()])
      const lastViewId = Storage.get(STORAGE_KEYS.SYSTEM_LOG_LAST_VIEW_ID, '')
      if (lastViewId && savedViews.value.some((item) => item.id === lastViewId)) {
        selectedViewId.value = lastViewId
        await handleApplySavedView(lastViewId, true)
      }
    })

    return {
      loading,
      t,
      loadError,
      logs,
      filters,
      pagination,
      levelStats,
      fetchLogs,
      handleSearch,
      resetFilters,
      handleRefresh,
      applyQuickRange,
      handleExportCurrent,
      saveCurrentView,
      handleApplySavedView,
      removeSelectedView,
      handleSizeChange,
      handleCurrentChange,
      getLevelTagType,
      getLevelLabel,
      formatDateTime,
      savedViews,
      sortedSavedViews,
      selectedViewId,
      systemLogPageSizeOptions: SYSTEM_LOG_PAGE_SIZE_OPTIONS,
    }
  },
}
</script>

<style scoped>
.system-logs {
  padding: 20px;
}

.panel {
  @apply bg-white dark:bg-gray-800 rounded-lg shadow-sm border border-gray-200 dark:border-gray-700;
}

.stat-card {
  @apply bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-4;
}

.stat-label {
  @apply text-sm text-gray-500 dark:text-gray-400;
}

.stat-value {
  @apply text-2xl font-semibold mt-2;
}
</style>

