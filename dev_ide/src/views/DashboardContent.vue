<template>
  <div class="dashboard-content ck-workbench-page">
    <!-- 一体化操作栏 (Cockpit-style) -->
    <div class="ck-workbench-toolbar">
      <!-- 左侧：标题与欢迎语 -->
      <div class="ck-toolbar-left">
        <span class="dashboard-welcome-text">
          {{ t('dashboard.welcomeBack', { username }) }}
        </span>
      </div>

      <!-- 右侧：刷新 -->
      <div class="ck-toolbar-right">
        <el-tooltip :content="t('common.refresh')" placement="top">
          <button
            class="ck-icon-button"
            :disabled="dashboardRefreshing"
            @click="handleDashboardRefresh"
          >
            <el-icon :class="{ 'animate-spin': dashboardRefreshing }"><RefreshRight /></el-icon>
          </button>
        </el-tooltip>
      </div>
    </div>

    <!-- 错误提示 -->
    <div v-if="loadError">
      <el-alert :title="loadError" type="warning" show-icon :closable="true" />
    </div>

    <!-- 统计卡片 -->
    <div class="ck-stat-grid dashboard-stat-grid">
      <!-- 用户统计 -->
      <div
        v-if="canAccessUserManagement"
        class="stat-card group cursor-pointer"
        @click="openTab('user-management')"
      >
        <div class="flex items-center justify-between">
          <div>
            <p class="dashboard-stat-label">
              {{ t('dashboard.userCount') }}
            </p>
            <p class="dashboard-stat-value">
              {{ loading ? '--' : stats.users }}
            </p>
          </div>
          <div
            class="dashboard-stat-icon dashboard-stat-icon--users group-hover:scale-110 transition-transform duration-300"
          >
            <el-icon class="w-6 h-6"><UserFilled /></el-icon>
          </div>
        </div>
        <div class="mt-3 flex items-center dashboard-stat-action text-gray-400 dark:text-gray-500">
          <svg
            class="w-3.5 h-3.5 mr-1"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
            stroke-width="2"
          >
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
            <p class="dashboard-stat-label">
              {{ t('dashboard.projectCount') }}
            </p>
            <p class="dashboard-stat-value">
              {{ loading ? '--' : stats.projects }}
            </p>
          </div>
          <div
            class="dashboard-stat-icon dashboard-stat-icon--projects group-hover:scale-110 transition-transform duration-300"
          >
            <el-icon class="w-6 h-6"><FolderOpened /></el-icon>
          </div>
        </div>
        <div class="mt-3 flex items-center dashboard-stat-action text-gray-400 dark:text-gray-500">
          <svg
            class="w-3.5 h-3.5 mr-1"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
            stroke-width="2"
          >
            <path stroke-linecap="round" stroke-linejoin="round" d="M13 7l5 5m0 0l-5 5m5-5H6" />
          </svg>
          {{ t('dashboard.clickToView') }}
        </div>
      </div>

      <!-- 节点统计 -->
      <div
        v-if="canAccessOpsManagement"
        class="stat-card group cursor-pointer"
        @click="openTab('ops-management')"
      >
        <div class="flex items-center justify-between">
          <div>
            <p class="dashboard-stat-label">
              {{ t('dashboard.nodeCount') }}
            </p>
            <p class="dashboard-stat-value">
              {{ loading ? '--' : stats.nodes }}
            </p>
          </div>
          <div
            class="dashboard-stat-icon dashboard-stat-icon--nodes group-hover:scale-110 transition-transform duration-300"
          >
            <el-icon class="w-6 h-6"><Connection /></el-icon>
          </div>
        </div>
        <div class="mt-3 flex items-center dashboard-stat-action text-gray-400 dark:text-gray-500">
          <svg
            class="w-3.5 h-3.5 mr-1"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
            stroke-width="2"
          >
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
            <p class="dashboard-stat-label">
              {{ t('dashboard.tenantCount') }}
            </p>
            <p class="dashboard-stat-value">
              {{ loading ? '--' : stats.tenants }}
            </p>
          </div>
          <div
            class="dashboard-stat-icon dashboard-stat-icon--tenants group-hover:scale-110 transition-transform duration-300"
          >
            <el-icon class="w-6 h-6"><OfficeBuilding /></el-icon>
          </div>
        </div>
        <div class="mt-3 flex items-center dashboard-stat-action text-gray-400 dark:text-gray-500">
          <svg
            class="w-3.5 h-3.5 mr-1"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
            stroke-width="2"
          >
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
            <p class="dashboard-stat-label">
              {{ t('dashboard.systemLogs') }}
            </p>
            <p class="dashboard-stat-value">
              {{ loading ? '--' : stats.logs }}
            </p>
          </div>
          <div
            class="dashboard-stat-icon dashboard-stat-icon--logs group-hover:scale-110 transition-transform duration-300"
          >
            <el-icon class="w-6 h-6"><DocumentCopy /></el-icon>
          </div>
        </div>
        <div class="mt-3 flex items-center dashboard-stat-action text-gray-400 dark:text-gray-500">
          <svg
            class="w-3.5 h-3.5 mr-1"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
            stroke-width="2"
          >
            <path stroke-linecap="round" stroke-linejoin="round" d="M13 7l5 5m0 0l-5 5m5-5H6" />
          </svg>
          {{ t('dashboard.clickToView') }}
        </div>
      </div>
    </div>

    <div class="dashboard-lower-grid">
      <!-- 租户共享便签 -->
      <div class="ck-content-area dashboard-note-panel">
        <div class="dashboard-panel-header">
          <h3 class="dashboard-panel-title">
            <span class="dashboard-panel-title-bar dashboard-panel-title-bar--note"></span>
            <el-icon><Memo /></el-icon>
            {{ t('dashboard.sharedNote') }}
          </h3>
          <span class="dashboard-panel-meta">
            {{ t('dashboard.noteCount', { count: tenantNotes.length }) }}
          </span>
        </div>

        <div class="dashboard-note-body">
          <el-alert
            v-if="noteLoadError"
            :title="noteLoadError"
            type="warning"
            show-icon
            :closable="false"
          />

          <div class="dashboard-note-composer">
            <el-input
              v-model="newNoteDraft"
              type="textarea"
              resize="none"
              :maxlength="2000"
              :rows="3"
              :placeholder="t('dashboard.notePlaceholder')"
              class="dashboard-note-input dashboard-note-input--composer"
              :disabled="noteLoading || noteCreating"
            />
            <div class="dashboard-note-composer-footer">
              <span
                class="dashboard-note-count"
                :class="{ 'is-limit': newNoteDraft.length >= 2000 }"
              >
                {{ t('dashboard.noteChars', { count: newNoteDraft.length, max: 2000 }) }}
              </span>
              <button
                type="button"
                class="ck-toolbar-pill-btn dashboard-note-save"
                :disabled="!canCreateTenantNote"
                @click="createTenantNote"
              >
                <el-icon v-if="noteCreating" class="animate-spin"><RefreshRight /></el-icon>
                <el-icon v-else><Plus /></el-icon>
                {{ t('dashboard.addNote') }}
              </button>
            </div>
          </div>

          <div class="dashboard-note-list">
            <div v-if="noteLoading" class="dashboard-note-state">
              <el-icon class="animate-spin"><RefreshRight /></el-icon>
              {{ t('dashboard.noteLoading') }}
            </div>
            <div v-else-if="tenantNotes.length === 0" class="dashboard-note-state">
              {{ t('dashboard.noNotes') }}
            </div>
            <div v-else v-for="note in tenantNotes" :key="note.id" class="dashboard-note-item">
              <template v-if="editingNoteId === note.id">
                <el-input
                  v-model="editingNoteDraft"
                  type="textarea"
                  resize="none"
                  :maxlength="2000"
                  :rows="3"
                  class="dashboard-note-input dashboard-note-input--edit"
                  :disabled="noteUpdatingId === note.id"
                />
                <div class="dashboard-note-item-footer">
                  <span
                    class="dashboard-note-count"
                    :class="{ 'is-limit': editingNoteDraft.length >= 2000 }"
                  >
                    {{ t('dashboard.noteChars', { count: editingNoteDraft.length, max: 2000 }) }}
                  </span>
                  <div class="dashboard-note-actions">
                    <button
                      type="button"
                      class="ck-icon-button dashboard-note-action-btn"
                      :disabled="noteUpdatingId === note.id"
                      @click="cancelEditTenantNote"
                    >
                      <el-icon><Close /></el-icon>
                    </button>
                    <button
                      type="button"
                      class="ck-icon-button dashboard-note-action-btn dashboard-note-action-btn--primary"
                      :disabled="!canUpdateTenantNote(note)"
                      @click="updateTenantNote(note)"
                    >
                      <el-icon v-if="noteUpdatingId === note.id" class="animate-spin">
                        <RefreshRight />
                      </el-icon>
                      <el-icon v-else><Check /></el-icon>
                    </button>
                  </div>
                </div>
              </template>
              <template v-else>
                <p class="dashboard-note-content">{{ note.content }}</p>
                <div class="dashboard-note-item-footer">
                  <span class="dashboard-note-meta">
                    {{ formatNoteMeta(note) }}
                  </span>
                  <div class="dashboard-note-actions">
                    <button
                      type="button"
                      class="ck-icon-button dashboard-note-action-btn"
                      @click="startEditTenantNote(note)"
                    >
                      <el-icon><EditPen /></el-icon>
                    </button>
                    <button
                      type="button"
                      class="ck-icon-button dashboard-note-action-btn dashboard-note-action-btn--danger"
                      :disabled="noteDeletingId === note.id"
                      @click="deleteTenantNote(note)"
                    >
                      <el-icon v-if="noteDeletingId === note.id" class="animate-spin">
                        <RefreshRight />
                      </el-icon>
                      <el-icon v-else><Delete /></el-icon>
                    </button>
                  </div>
                </div>
              </template>
            </div>
          </div>
        </div>
      </div>

      <!-- 最近活动 -->
      <div class="ck-content-area dashboard-activity-panel">
        <div class="dashboard-panel-header">
          <h3 class="dashboard-panel-title">
            <span class="dashboard-panel-title-bar dashboard-panel-title-bar--activity"></span>
            {{ t('dashboard.recentActivity') }}
          </h3>
          <span class="dashboard-panel-meta">
            {{ t('dashboard.lastUpdated', { time: formatDateTime(lastUpdatedAt) || '-' }) }}
          </span>
        </div>

        <div class="ck-content-scroll dashboard-activity-scroll">
          <div v-if="loading" class="dashboard-activity-state">
            <div class="text-sm text-gray-400 dark:text-gray-500 flex items-center gap-2">
              <el-icon class="animate-spin"><RefreshRight /></el-icon>
              {{ t('dashboard.loadingData') }}
            </div>
          </div>

          <div v-else-if="recentActivities.length === 0" class="dashboard-activity-state">
            <div class="text-center">
              <div
                class="w-16 h-16 mx-auto mb-3 bg-gray-100 dark:bg-gray-700 rounded-2xl flex items-center justify-center"
              >
                <svg
                  class="w-8 h-8 text-gray-300 dark:text-gray-600"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                  stroke-width="1.5"
                >
                  <path
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    d="M19.5 14.25v-2.625a3.375 3.375 0 00-3.375-3.375h-1.5A1.125 1.125 0 0113.5 7.125v-1.5a3.375 3.375 0 00-3.375-3.375H8.25m0 12.75h7.5m-7.5 3H12M10.5 2.25H5.625c-.621 0-1.125.504-1.125 1.125v17.25c0 .621.504 1.125 1.125 1.125h12.75c.621 0 1.125-.504 1.125-1.125V11.25a9 9 0 00-9-9z"
                  />
                </svg>
              </div>
              <p class="text-sm text-gray-400 dark:text-gray-500">
                {{ t('dashboard.noActivity') }}
              </p>
            </div>
          </div>

          <div v-else class="dashboard-activity-list">
            <div
              v-for="(activity, index) in recentActivities"
              :key="activity.id"
              class="dashboard-activity-item"
            >
              <div class="flex-shrink-0 relative">
                <div class="dashboard-activity-index">
                  {{ index + 1 }}
                </div>
              </div>
              <div class="flex-1 min-w-0">
                <p
                  class="text-[13px] text-gray-700 dark:text-gray-300 leading-relaxed line-clamp-2"
                >
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
  </div>
</template>

<script lang="ts">
// @ts-nocheck
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  UserFilled,
  FolderOpened,
  OfficeBuilding,
  DocumentCopy,
  InfoFilled,
  RefreshRight,
  Connection,
  Memo,
  Check,
  Plus,
  EditPen,
  Delete,
  Close,
} from '@element-plus/icons-vue'
import { useAuthStore } from '@/store'
import { userAPI, projectAPI, tenantAPI, logAPI } from '@/api'
import request from '@/utils/request'
import { formatDateTime } from '@/utils/date'
import { getApiErrorMessage } from '@/utils/request'
import { formatDashboardActivity } from '@/utils/dashboard-activity'
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
    const canAccessOpsManagement = computed(() => canAccessTab('ops-management', role.value))
    const loading = ref(false)
    const noteLoading = ref(false)
    const noteCreating = ref(false)
    const noteUpdatingId = ref('')
    const noteDeletingId = ref('')
    const loadError = ref('')
    const noteLoadError = ref('')
    const lastUpdatedAt = ref('')
    const stats = ref({
      projects: 0,
      users: 0,
      tenants: 0,
      logs: 0,
      nodes: 0,
    })

    const recentActivities = ref([])
    const tenantNotes = ref([])
    const newNoteDraft = ref('')
    const editingNoteId = ref('')
    const editingNoteDraft = ref('')
    const dashboardRefreshing = computed(
      () => loading.value || noteLoading.value || noteCreating.value || !!noteUpdatingId.value,
    )
    const canCreateTenantNote = computed(
      () =>
        !noteLoading.value &&
        !noteCreating.value &&
        newNoteDraft.value.trim().length > 0 &&
        newNoteDraft.value.length <= 2000,
    )

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
          requestEntries.push(['logs', logAPI.getLogs({ page: 1, limit: 1 })])
        }
        requestEntries.push(['recentActivities', logAPI.getRecentActivities({ limit: 5 })])
        if (canAccessOpsManagement.value) {
          requestEntries.push([
            'nodes',
            request.get('/nodes', {
              params: { page: 1, pageSize: 1, approvalStatus: 'approved' },
            }),
          ])
        }
        if (canLoadTenantStats.value) {
          requestEntries.push(['tenants', tenantAPI.getTenants({ page: 1, limit: 1 })])
        }

        if (requestEntries.length === 0) {
          stats.value = { projects: 0, users: 0, tenants: 0, logs: 0, nodes: 0 }
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
        const nodesResult = resultMap.nodes
        const recentActivitiesResult = resultMap.recentActivities
        const tenantsResult = resultMap.tenants

        const getPaginationTotal = (result) =>
          result?.data?.pagination?.total || result?.pagination?.total || 0
        const getNodeTotal = (result) => result?.data?.total || result?.total || 0

        stats.value.users =
          usersResult?.status === 'fulfilled' ? getPaginationTotal(usersResult.value) : 0
        stats.value.projects =
          projectsResult?.status === 'fulfilled' ? getPaginationTotal(projectsResult.value) : 0
        stats.value.logs =
          logsResult?.status === 'fulfilled' ? getPaginationTotal(logsResult.value) : 0
        stats.value.nodes =
          nodesResult?.status === 'fulfilled' ? getNodeTotal(nodesResult.value) : 0
        stats.value.tenants =
          canLoadTenantStats.value && tenantsResult?.status === 'fulfilled'
            ? getPaginationTotal(tenantsResult.value)
            : 0

        if (recentActivitiesResult?.status === 'fulfilled') {
          const activities =
            recentActivitiesResult.value?.data?.list?.activities ||
            recentActivitiesResult.value?.data?.activities ||
            []
          recentActivities.value = activities.slice(0, 5).map((activity, index) => ({
            id: activity.id || `${activity.createdAt}-${index}`,
            description: formatDashboardActivity(activity, t),
            time: activity.createdAt || '',
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
            nodes: t('dashboard.nodeCount'),
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
          nodes: 0,
        }
        recentActivities.value = []
        loadError.value = t('dashboard.loadFailed')
      } finally {
        loading.value = false
      }
    }

    const normalizeTenantNote = (note = {}) => ({
      id: note.id || '',
      content: typeof note.content === 'string' ? note.content : '',
      createdAt: note.createdAt || '',
      createdByName: note.createdByName || '',
      updatedAt: note.updatedAt || '',
      updatedByName: note.updatedByName || '',
    })

    const applyTenantNotes = (notes = []) => {
      tenantNotes.value = Array.isArray(notes) ? notes.map(normalizeTenantNote) : []
    }

    const loadTenantNote = async () => {
      noteLoading.value = true
      noteLoadError.value = ''
      try {
        const response = await tenantAPI.getDashboardNotes()
        applyTenantNotes(response?.data?.notes || [])
      } catch (error) {
        noteLoadError.value = getApiErrorMessage(error, t('dashboard.noteLoadFailed'))
      } finally {
        noteLoading.value = false
      }
    }

    const createTenantNote = async () => {
      if (!canCreateTenantNote.value) return
      noteCreating.value = true
      try {
        const response = await tenantAPI.createDashboardNote(newNoteDraft.value.trim())
        const note = normalizeTenantNote(response?.data?.note || {})
        tenantNotes.value = [note, ...tenantNotes.value]
        newNoteDraft.value = ''
        ElMessage.success(t('dashboard.noteSaved'))
      } catch (error) {
        ElMessage.error(getApiErrorMessage(error, t('dashboard.noteSaveFailed')))
      } finally {
        noteCreating.value = false
      }
    }

    const startEditTenantNote = (note) => {
      editingNoteId.value = note.id
      editingNoteDraft.value = note.content
    }

    const cancelEditTenantNote = () => {
      editingNoteId.value = ''
      editingNoteDraft.value = ''
    }

    const canUpdateTenantNote = (note) =>
      noteUpdatingId.value !== note.id &&
      editingNoteId.value === note.id &&
      editingNoteDraft.value.trim().length > 0 &&
      editingNoteDraft.value.length <= 2000 &&
      editingNoteDraft.value !== note.content

    const updateTenantNote = async (note) => {
      if (!canUpdateTenantNote(note)) return
      noteUpdatingId.value = note.id
      try {
        const response = await tenantAPI.updateDashboardNote(note.id, editingNoteDraft.value.trim())
        const updatedNote = normalizeTenantNote(response?.data?.note || {})
        tenantNotes.value = tenantNotes.value.map((item) =>
          item.id === updatedNote.id
            ? {
                ...item,
                ...updatedNote,
                createdByName: updatedNote.createdByName || item.createdByName,
              }
            : item,
        )
        cancelEditTenantNote()
        ElMessage.success(t('dashboard.noteSaved'))
      } catch (error) {
        ElMessage.error(getApiErrorMessage(error, t('dashboard.noteSaveFailed')))
      } finally {
        noteUpdatingId.value = ''
      }
    }

    const deleteTenantNote = async (note) => {
      try {
        await ElMessageBox.confirm(t('dashboard.noteDeleteConfirm'), t('common.tip'), {
          confirmButtonText: t('common.confirm'),
          cancelButtonText: t('common.cancel'),
          type: 'warning',
        })
      } catch {
        return
      }

      noteDeletingId.value = note.id
      try {
        await tenantAPI.deleteDashboardNote(note.id)
        tenantNotes.value = tenantNotes.value.filter((item) => item.id !== note.id)
        if (editingNoteId.value === note.id) {
          cancelEditTenantNote()
        }
        ElMessage.success(t('dashboard.noteDeleted'))
      } catch (error) {
        ElMessage.error(getApiErrorMessage(error, t('dashboard.noteDeleteFailed')))
      } finally {
        noteDeletingId.value = ''
      }
    }

    const formatNoteMeta = (note) => {
      const updater = note.updatedByName || note.createdByName || t('dashboard.systemAction')
      const time = formatDateTime(note.updatedAt || note.createdAt) || '-'
      return t('dashboard.noteUpdatedMeta', { user: updater, time })
    }

    const handleDashboardRefresh = async () => {
      await Promise.all([loadDashboardData(), loadTenantNote()])
    }

    // 点击卡片打开对应标签页
    const openTab = (tabKey) => {
      emit('open-tab', tabKey)
    }

    onMounted(() => {
      handleDashboardRefresh()
    })

    return {
      username,
      canLoadTenantStats,
      canAccessUserManagement,
      canAccessProjectManagement,
      canAccessSystemLogs,
      canAccessOpsManagement,
      loading,
      noteLoading,
      noteCreating,
      noteUpdatingId,
      noteDeletingId,
      dashboardRefreshing,
      loadError,
      noteLoadError,
      lastUpdatedAt,
      stats,
      recentActivities,
      tenantNotes,
      newNoteDraft,
      editingNoteId,
      editingNoteDraft,
      canCreateTenantNote,
      t,
      UserFilled,
      FolderOpened,
      OfficeBuilding,
      DocumentCopy,
      InfoFilled,
      RefreshRight,
      Connection,
      Memo,
      Check,
      Plus,
      EditPen,
      Delete,
      Close,
      openTab,
      loadDashboardData,
      loadTenantNote,
      createTenantNote,
      startEditTenantNote,
      cancelEditTenantNote,
      canUpdateTenantNote,
      updateTenantNote,
      deleteTenantNote,
      formatNoteMeta,
      handleDashboardRefresh,
      formatDateTime,
    }
  },
}
</script>

<style scoped>
.dashboard-content {
  background: transparent;
  overflow: hidden;
}

.dashboard-welcome-text {
  font-size: 20px;
  line-height: 28px;
  font-weight: 700;
  color: var(--ck-text-primary);
}

/* 统计卡片 */
.stat-card {
  @apply bg-white dark:bg-gray-800 rounded-2xl p-6 border border-gray-200/80 dark:border-gray-700 shadow-[0_2px_12px_rgba(0,0,0,0.04)] hover:shadow-[0_8px_24px_rgba(0,0,0,0.08)] hover:-translate-y-0.5 transition-all duration-300;
}

.dashboard-stat-label {
  margin-bottom: 6px;
  font-size: 14px;
  line-height: 20px;
  font-weight: 700;
  color: var(--ck-text-secondary);
}

.dashboard-stat-value {
  font-size: 32px;
  line-height: 38px;
  font-weight: 800;
  color: var(--ck-text-primary);
}

.dashboard-stat-action {
  font-size: 13px;
  line-height: 18px;
  font-weight: 600;
}

.dashboard-stat-icon {
  width: 48px;
  height: 48px;
  border-radius: 14px;
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 8px 18px rgba(15, 23, 42, 0.08);
}

.dashboard-stat-icon--users {
  color: #047857;
  background: linear-gradient(135deg, rgba(16, 185, 129, 0.16), rgba(5, 150, 105, 0.24));
}

.dashboard-stat-icon--projects {
  color: #2563eb;
  background: linear-gradient(135deg, rgba(59, 130, 246, 0.16), rgba(37, 99, 235, 0.24));
}

.dashboard-stat-icon--nodes {
  color: #0891b2;
  background: linear-gradient(135deg, rgba(6, 182, 212, 0.16), rgba(8, 145, 178, 0.24));
}

.dashboard-stat-icon--tenants {
  color: #b45309;
  background: linear-gradient(135deg, rgba(245, 158, 11, 0.16), rgba(217, 119, 6, 0.24));
}

.dashboard-stat-icon--logs {
  color: #be123c;
  background: linear-gradient(135deg, rgba(244, 63, 94, 0.14), rgba(225, 29, 72, 0.22));
}

.dashboard-stat-grid {
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
}

.dashboard-lower-grid {
  flex: 1;
  display: grid;
  grid-template-columns: minmax(0, 1.25fr) minmax(360px, 0.75fr);
  gap: 16px;
  min-height: 0;
  overflow: hidden;
}

.dashboard-note-panel,
.dashboard-activity-panel {
  padding: 0;
  height: 100%;
  min-height: 0;
  overflow: hidden;
}

.dashboard-note-panel,
.dashboard-activity-panel {
  display: flex;
  flex-direction: column;
}

.dashboard-panel-header,
.dashboard-activity-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 16px;
  border-bottom: 1px solid var(--ck-border-light);
}

.dashboard-panel-title {
  display: flex;
  align-items: center;
  gap: 8px;
  color: var(--ck-text-primary);
  font-size: 15px;
  font-weight: 700;
}

.dashboard-panel-title-bar {
  width: 6px;
  height: 20px;
  border-radius: 999px;
}

.dashboard-panel-title-bar--note {
  background: #64748b;
}

.dashboard-panel-title-bar--activity {
  background: #3b82f6;
}

.dashboard-panel-meta {
  color: var(--ck-text-muted);
  font-size: 11px;
  font-family:
    ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', 'Courier New',
    monospace;
}

.dashboard-note-body {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 14px 16px 10px;
  min-height: 0;
  overflow: hidden;
}

.dashboard-note-input {
  width: 100%;
}

.dashboard-note-input :deep(.el-textarea__inner) {
  border: 0;
  border-radius: var(--ck-radius-md);
  background: rgba(248, 250, 252, 0.72);
  box-shadow: none;
  color: var(--ck-text-primary);
  font-size: 14px;
  line-height: 22px;
  padding: 10px 12px;
}

.dashboard-note-input--composer :deep(.el-textarea__inner),
.dashboard-note-input--edit :deep(.el-textarea__inner) {
  min-height: 82px !important;
}

.dashboard-note-input :deep(.el-textarea__inner:focus) {
  box-shadow: 0 0 0 3px var(--ck-primary-light);
}

.dashboard-note-composer {
  padding: 10px;
  border-radius: var(--ck-radius-md);
  background: linear-gradient(135deg, rgba(226, 232, 240, 0.68), rgba(248, 250, 252, 0.9));
}

.dashboard-note-composer-footer,
.dashboard-note-item-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-top: 8px;
}

.dashboard-note-count {
  color: var(--ck-text-muted);
  font-size: 12px;
}

.dashboard-note-count.is-limit {
  color: var(--ck-danger);
}

.dashboard-note-save {
  color: #fff;
  background: var(--ck-primary);
}

.dashboard-note-save:hover:not(:disabled) {
  color: #fff;
  background: var(--ck-primary-hover);
}

.dashboard-note-save:disabled {
  cursor: not-allowed;
  opacity: 0.48;
}

.dashboard-note-list {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
  overflow-y: auto;
  overscroll-behavior: contain;
  scrollbar-gutter: stable;
  padding-right: 2px;
}

.dashboard-note-state {
  min-height: 120px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  color: var(--ck-text-muted);
  font-size: 13px;
}

.dashboard-note-item {
  padding: 10px;
  border-radius: var(--ck-radius-md);
  background: rgba(248, 250, 252, 0.74);
  transition: background 0.2s ease;
}

.dashboard-note-item:hover {
  background: var(--ck-bg-hover);
}

.dashboard-note-content {
  color: var(--ck-text-primary);
  font-size: 13px;
  line-height: 21px;
  white-space: pre-wrap;
  word-break: break-word;
}

.dashboard-note-meta {
  min-width: 0;
  color: var(--ck-text-muted);
  font-size: 11px;
}

.dashboard-note-actions {
  display: flex;
  flex-shrink: 0;
  align-items: center;
  gap: 6px;
}

.dashboard-note-action-btn {
  width: 28px;
  height: 28px;
  border-radius: 8px;
}

.dashboard-note-action-btn--primary {
  color: var(--ck-primary);
}

.dashboard-note-action-btn--danger {
  color: var(--ck-danger);
}

.dashboard-activity-scroll {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow-y: auto;
  overscroll-behavior: contain;
  scrollbar-gutter: stable;
}

.dashboard-activity-state {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 220px;
}

.dashboard-activity-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.dashboard-activity-item {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 9px 10px;
  border-radius: var(--ck-radius-md);
  background: rgba(248, 250, 252, 0.72);
  transition: background 0.2s ease;
}

.dashboard-activity-item:hover {
  background: var(--ck-bg-hover);
}

.dashboard-activity-index {
  width: 26px;
  height: 26px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--ck-text-secondary);
  background: var(--ck-bg-tertiary);
  font-size: 12px;
  font-weight: 700;
}

@media (max-width: 900px) {
  .dashboard-stat-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .dashboard-lower-grid {
    grid-template-columns: 1fr;
    flex: initial;
    overflow: visible;
  }
}

@media (max-width: 640px) {
  .dashboard-stat-grid {
    grid-template-columns: 1fr;
  }
}
</style>
