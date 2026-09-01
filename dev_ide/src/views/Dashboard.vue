<template>
  <div
    class="dashboard h-screen flex flex-col bg-gray-50 dark:bg-[#121212]"
    :class="{ 'dashboard-maximized': isTabMaximized }"
    @mousemove="handleMaximizedMouseMove"
    @mouseleave="handleMaximizedMouseLeave"
  >
    <!-- 全新的侧边栏抽屉组件（全屏时隐藏） -->
    <DashboardSidebar
      v-if="!isTabMaximized"
      :active-tab="activeTab"
      @open-tab="openTab"
      @open-profile="openProfileDialog"
      @open-settings="openSystemSettingsDialog"
    />

    <!-- 主要内容区域 -->
    <div class="flex flex-1 overflow-hidden w-full relative">
      <DashboardTabsArea
        :tabs="tabs"
        v-model:activeTab="activeTab"
        :is-tab-maximized="isTabMaximized"
        :show-maximize-restore-button="showMaximizeRestoreButton"
        :is-tab-visible="isTabVisible"
        :get-tab-title="getTabTitle"
        @close-tab="closeTab"
        @open-external-tab="openExternalTab"
        @maximize-tab="maximizeTab"
        @restore-tab="restoreTab"
        @open-tab="openTab"
        @open-workspace="handleWorkspaceOpenRequest"
        @close-workspace="handleWorkspaceCloseRequest"
        @scene-committed="handleWorkspaceSceneCommitted"
      />
    </div>

    <!-- 对话框 -->
    <el-dialog
      v-model="showProfileDialog"
      width="860px"
      top="8vh"
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

<script lang="ts">
// @ts-nocheck
import { ref, computed, onMounted, onUnmounted, watch, defineAsyncComponent, markRaw } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAuthStore, useAppStore, useTenantStore } from '@/store'
import { Storage } from '@/utils/storage'
import { ElMessage, ElMessageBox, ElNotification } from 'element-plus'
import WujieVue from 'wujie-vue3'
import { resolveDashboardTabTitle } from '@/utils/dashboardTabTitle'
import { restoreDashboardTabState, serializeDashboardTabState } from '@/utils/dashboardTabState'
import { resolveTenantBrandLogo } from '@/utils/tenantBrand'
import { can, canAccessTab, getTabAccessDeniedMessage } from '@/permissions'
import { ROLES, STORAGE_KEYS } from '@/constants'
import { initSocket, getSocket } from '@/utils/socket'
import request from '@/utils/request'
import {
  isWorkspaceCloseRequest,
  isWorkspaceOpenRequest,
  matchesWorkspaceRequestProject,
  workspaceToolTabKey,
  workspaceToolTitle,
} from '@/types/workspace-tool'

// 标签页组件懒加载，提升首次加载速度
const DashboardContent = markRaw(defineAsyncComponent(() => import('@/views/DashboardContent.vue')))
const TenantManagement = markRaw(
  defineAsyncComponent(() => import('@/views/admin/TenantManagement.vue')),
)
const UserManagement = markRaw(
  defineAsyncComponent(() => import('@/views/tenant/UserManagement.vue')),
)
const ProjectManagement = markRaw(
  defineAsyncComponent(() => import('@/views/tenant/ProjectManagement.vue')),
)
const OpsManagement = markRaw(
  defineAsyncComponent(() => import('@/views/tenant/OpsManagementConsole.vue')),
)
const SystemLogs = markRaw(defineAsyncComponent(() => import('@/views/tenant/SystemLogs.vue')))
const SystemSettings = markRaw(
  defineAsyncComponent(() => import('@/views/tenant/SystemSettings.vue')),
)
const Profile = markRaw(defineAsyncComponent(() => import('@/views/profile/Profile.vue')))
const WujieMicroApp = markRaw(defineAsyncComponent(() => import('@/components/WujieMicroApp.vue')))
const WorkspaceToolFrame = markRaw(
  defineAsyncComponent(() => import('@/components/WorkspaceToolFrame.vue')),
)

import DashboardSidebar from './layout/DashboardSidebar.vue'
import DashboardTabsArea from './layout/DashboardTabsArea.vue'

export default {
  name: 'Dashboard',
  components: {
    Profile,
    SystemSettings,
    DashboardSidebar,
    DashboardTabsArea,
  },
  setup() {
    const route = useRoute()
    const router = useRouter()
    const { locale, t } = useI18n()
    const authStore = useAuthStore()
    const appStore = useAppStore()
    const tenantStore = useTenantStore()

    const showUserMenu = ref(false)
    const showLanguageMenu = ref(false)
    const showProfileDialog = ref(false)
    const showSystemSettingsDialog = ref(false)
    const pendingRequestNotifications = new Map()
    const pendingPollingTimer = ref(null)
    const lastPendingCount = ref(0)
    const opsSocket = ref(null)
    const wsConnectCheckTimer = ref(null)

    // 侧边栏折叠状态（持久化）
    const sidebarCollapsed = computed({
      get: () => appStore.sidebarCollapsed,
      set: (value) => appStore.setSidebarCollapsed(value),
    })

    // 标签页状态
    const tabs = ref([])
    const activeTab = ref('')
    const isTabMaximized = ref(false)
    const showMaximizeRestoreButton = ref(false)
    const maximizeRestoreHideTimer = ref(null)
    const tabsInitialized = ref(false)

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
    const canReceiveNodePendingAlerts = computed(() => isSystemAdmin.value || isOpsAdmin.value)
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

    // 租户相关计算属性
    const currentTenant = computed(() => tenantStore.currentTenant)
    const tenantLogoUrl = computed(() => {
      return resolveTenantBrandLogo(currentTenant.value)
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

        await authStore.logout()
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
      const customTitlePrefix = isObject ? tabData.titlePrefix : null
      const customTitleParams = isObject ? tabData.titleParams : null
      const customComponent = isObject ? tabData.component : null
      const customProps = isObject ? tabData.props : null

      let config
      if (isObject && customComponent) {
        // 自定义标签页配置
        // 如果组件是函数（可能是动态导入），使用 defineAsyncComponent 包装以确保正确处理
        const component =
          typeof customComponent === 'function'
            ? markRaw(defineAsyncComponent(customComponent))
            : markRaw(customComponent)
        config = {
          title: customTitle,
          component: component,
          icon: tabData.icon || 'folder',
          titleKey: tabData.titleKey || null,
          titlePrefix: customTitlePrefix,
          titleParams: customTitleParams,
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
        title: resolveDashboardTabTitle(config, t),
        titleKey: config.titleKey || null,
        titlePrefix: config.titlePrefix || null,
        titleParams: config.titleParams || null,
        component: config.component,
        icon: config.icon,
        props: customProps,
      }

      tabs.value.push(newTab)

      // 激活新标签页
      activeTab.value = tabKey
    }

    /**
     * 只接受当前 Designer 实例为自身工程发起的受控工具打开请求。
     */
    const handleWorkspaceOpenRequest = ({ tab, request }) => {
      if (!isWorkspaceOpenRequest(request) || tab?.props?.appType !== 'designer') return
      if (!can(authStore.userInfo?.role, 'project:write')) {
        ElMessage.error('当前用户没有工程工具访问权限')
        return
      }
      const project = tab.props.project
      const projectId = String(project?.id || '')
      if (!matchesWorkspaceRequestProject(request, projectId)) {
        ElMessage.error('工程工具请求与当前工程不一致')
        return
      }

      openTab({
        key: workspaceToolTabKey(projectId, request.target, request.sceneId),
        title: workspaceToolTitle(request.target, request.sceneName),
        component: WorkspaceToolFrame,
        props: {
          target: request.target,
          sceneId: request.sceneId,
          sceneName: request.sceneName,
          project: {
            id: projectId,
            name: project.name,
            tenantId: project.tenantId,
          },
        },
        icon: 'design',
      })
    }

    /**
     * 打开运维待审核申请界面。
     */
    const openOpsPendingRequests = () => {
      if (!canReceiveNodePendingAlerts.value) return
      openTab('ops-management')
      // 通过短时重试确保 OpsManagement 完成挂载后再打开弹窗。
      ;[80, 220, 420].forEach((delay) => {
        window.setTimeout(() => {
          window.dispatchEvent(new window.CustomEvent('ops:open-pending-requests'))
        }, delay)
      })
    }

    /**
     * 显示节点待审核的全局通知（常驻，手动关闭）。
     * @param {object} payload - 节点申请事件载荷
     */
    const showNodePendingNotification = (payload = {}) => {
      const nodeId = payload.nodeId || payload.id || `${Date.now()}`
      if (pendingRequestNotifications.has(nodeId)) return

      const nodeName = payload.nodeName || '-'
      const applicantName = payload.applicant?.username || payload.username || '-'
      let notificationRef = null
      notificationRef = ElNotification({
        title: t('dashboard.pendingNodeTitle'),
        message: t('dashboard.pendingNodeMessage', {
          nodeName,
          applicant: applicantName,
        }),
        type: 'warning',
        duration: 0,
        showClose: true,
        onClick: () => {
          openOpsPendingRequests()
          notificationRef?.close?.()
        },
        onClose: () => {
          pendingRequestNotifications.delete(nodeId)
        },
      })

      pendingRequestNotifications.set(nodeId, notificationRef)
    }

    /**
     * 页面初始化时检查是否存在未处理的待审核申请，并主动提醒一次。
     */
    const notifyExistingPendingRequests = async () => {
      if (!canReceiveNodePendingAlerts.value) return
      try {
        const res = await request.get('/nodes', {
          params: {
            page: 1,
            pageSize: 5,
            approvalStatus: 'pending',
          },
        })
        const items = res?.data?.items || []
        if (!Array.isArray(items) || items.length === 0) return

        const first = items[0]
        showNodePendingNotification({
          nodeId: `pending-summary-${Date.now()}`,
          nodeName: first?.name || '-',
          applicant: { username: first?.registrant?.username || '-' },
        })
      } catch (error) {
        console.warn('[OpsPending] 初始化待审核提醒失败:', error)
      }
    }

    /**
     * 订阅运维事件并处理全局通知。
     */
    const setupOpsPendingSubscription = () => {
      if (!canReceiveNodePendingAlerts.value) return
      const tenantId = Storage.getTenantId()
      if (!tenantId) return
      const socket = initSocket(tenantId)
      opsSocket.value = socket

      socket.on('connect', handleOpsSocketConnect)
      socket.on('disconnect', handleOpsSocketDisconnect)
      socket.on('connect_error', handleOpsSocketConnectError)
      socket.on('ops:node:pending', handleOpsNodePending)

      if (wsConnectCheckTimer.value) {
        window.clearTimeout(wsConnectCheckTimer.value)
      }
      wsConnectCheckTimer.value = window.setTimeout(() => {
        if (!socket.connected) {
          console.warn('[OpsPending][WS] 连接超时，启用轮询兜底')
          startPendingPolling('connect-timeout')
        }
      }, 6000)
    }

    /**
     * 兜底：轮询待审核申请数量，避免 WebSocket 异常时无法实时提示。
     */
    const startPendingPolling = async (reason = 'unknown') => {
      if (!canReceiveNodePendingAlerts.value) return
      if (pendingPollingTimer.value) return
      console.warn(`[OpsPending][Polling] 已启用，原因: ${reason}`)

      const pollPendingCount = async (notifyOnIncrease = false) => {
        try {
          const res = await request.get('/nodes', {
            params: { pageSize: 1, approvalStatus: 'pending' },
          })
          const total = Number(res?.data?.total || 0)
          if (notifyOnIncrease && total > lastPendingCount.value) {
            console.log(
              `[OpsPending][Polling] 检测到待审核新增: ${lastPendingCount.value} -> ${total}`,
            )
            ElNotification({
              title: t('dashboard.pendingNodeTitle'),
              message: t('dashboard.pendingNodeMessage', {
                nodeName: '-',
                applicant: '-',
              }),
              type: 'warning',
              duration: 5000,
              onClick: () => {
                openOpsPendingRequests()
              },
            })
          }
          lastPendingCount.value = total
        } catch (error) {
          console.warn('轮询待审核申请失败:', error)
        }
      }

      await pollPendingCount(false)
      pendingPollingTimer.value = window.setInterval(() => {
        pollPendingCount(true)
      }, 8000)
    }

    /**
     * 停止兜底轮询。
     */
    const stopPendingPolling = () => {
      if (pendingPollingTimer.value) {
        window.clearInterval(pendingPollingTimer.value)
        pendingPollingTimer.value = null
        console.log('[OpsPending][Polling] 已停止')
      }
    }

    /**
     * WebSocket 连接成功回调。
     */
    const handleOpsSocketConnect = () => {
      console.log('[OpsPending][WS] 已连接')
      stopPendingPolling()
    }

    /**
     * WebSocket 连接断开回调。
     * @param {string} reason - 断开原因
     */
    const handleOpsSocketDisconnect = (reason) => {
      console.warn('[OpsPending][WS] 连接断开:', reason)
      startPendingPolling(`disconnect:${reason || 'unknown'}`)
    }

    /**
     * WebSocket 连接错误回调。
     * @param {Error} error - 连接错误
     */
    const handleOpsSocketConnectError = (error) => {
      console.warn('[OpsPending][WS] 连接失败:', error?.message || error)
      startPendingPolling(`connect_error:${error?.message || 'unknown'}`)
    }

    /**
     * 收到待审核事件回调。
     * @param {object} payload - 节点申请事件载荷
     */
    const handleOpsNodePending = (payload = {}) => {
      console.log('[OpsPending][WS] 收到待审核事件:', payload)
      showNodePendingNotification(payload)
    }

    /**
     * 获取当前用户对应的标签持久化键。
     * @returns {string} 本地存储键
     */
    const getTabStateStorageKey = () => {
      const userId = authStore.userInfo?.id || 'anonymous'
      return `${STORAGE_KEYS.DASHBOARD_TAB_STATE}_${userId}`
    }

    /**
     * 持久化当前标签状态。
     */
    const persistTabState = () => {
      if (!tabsInitialized.value) return
      Storage.set(
        getTabStateStorageKey(),
        serializeDashboardTabState({
          tabs: tabs.value,
          activeTab: activeTab.value,
          tabConfigMap: getTabConfigMap(),
          hasTabPermission,
        }),
      )
    }

    /**
     * 从本地存储恢复标签状态。
     * @returns {boolean} 是否恢复成功
     */
    const restoreTabState = () => {
      const restoredState = restoreDashboardTabState(Storage.get(getTabStateStorageKey(), null), {
        tabConfigMap: getTabConfigMap(),
        hasTabPermission,
        microAppComponent: WujieMicroApp,
        workspaceToolComponent: WorkspaceToolFrame,
        translate: t,
      })
      if (!restoredState) return false

      tabs.value = restoredState.tabs
      activeTab.value = restoredState.activeTab

      return true
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

    /**
     * 场景删除成功后，只关闭同工程、同类型、同场景的工具标签。
     */
    const handleWorkspaceCloseRequest = ({ tab, request }) => {
      if (!isWorkspaceCloseRequest(request) || tab?.props?.appType !== 'designer') return
      const projectId = String(tab.props.project?.id || '')
      if (!matchesWorkspaceRequestProject(request, projectId)) return
      closeTab(workspaceToolTabKey(projectId, request.target, request.sceneId))
    }

    /**
     * HT revision 提交成功后通知同工程 Designer 刷新场景摘要。
     */
    const handleWorkspaceSceneCommitted = ({ tab, event }) => {
      if (tab?.component !== WorkspaceToolFrame || !event) return
      const projectId = String(tab.props?.project?.id || '')
      if (
        !projectId ||
        event.projectId !== projectId ||
        event.target !== tab.props?.target ||
        event.sceneId !== tab.props?.sceneId
      ) {
        return
      }
      WujieVue.bus.$emit(`micro-app:designer-${projectId}:scene-committed`, event)
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
      window.addEventListener('keydown', handleKeyDown)

      // 优先恢复历史标签状态，未恢复成功时按角色打开默认标签
      const restored = restoreTabState()
      if (!restored) {
        if (isSuperAdmin.value) {
          openTab('tenant-management')
        } else {
          openTab('dashboard')
        }
      }
      tabsInitialized.value = true
      persistTabState()
      setupOpsPendingSubscription()
      notifyExistingPendingRequests()

      // 当前租户品牌用于菜单栏和浏览器标签，所有登录角色都需要读取。
      if (authStore.userInfo?.tenantId) {
        try {
          await tenantStore.fetchCurrentTenant()
        } catch (error) {
          console.error('Failed to fetch current tenant:', error)
        }
      }
    })

    // 组件卸载时移除事件监听
    onUnmounted(() => {
      document.removeEventListener('click', handleClickOutside)
      window.removeEventListener('keydown', handleKeyDown)
      const socket = getSocket()
      if (socket) {
        socket.off('connect', handleOpsSocketConnect)
        socket.off('disconnect', handleOpsSocketDisconnect)
        socket.off('connect_error', handleOpsSocketConnectError)
        socket.off('ops:node:pending', handleOpsNodePending)
      }
      pendingRequestNotifications.forEach((notification) => notification?.close?.())
      pendingRequestNotifications.clear()
      stopPendingPolling()
      if (wsConnectCheckTimer.value) {
        window.clearTimeout(wsConnectCheckTimer.value)
        wsConnectCheckTimer.value = null
      }
      if (maximizeRestoreHideTimer.value) {
        window.clearTimeout(maximizeRestoreHideTimer.value)
        maximizeRestoreHideTimer.value = null
      }
    })

    // 最大化标签页
    const maximizeTab = (tabKey) => {
      if (tabKey === 'dashboard') return
      isTabMaximized.value = true
      activeTab.value = tabKey
      showMaximizeRestoreButton.value = true
      // 全屏后自动收起侧边栏
      appStore.setSidebarCollapsed(true)
      if (maximizeRestoreHideTimer.value) {
        window.clearTimeout(maximizeRestoreHideTimer.value)
      }
      maximizeRestoreHideTimer.value = window.setTimeout(() => {
        showMaximizeRestoreButton.value = false
      }, 2000)
    }

    // 还原标签页
    const restoreTab = () => {
      isTabMaximized.value = false
      showMaximizeRestoreButton.value = false
      if (maximizeRestoreHideTimer.value) {
        window.clearTimeout(maximizeRestoreHideTimer.value)
        maximizeRestoreHideTimer.value = null
      }
    }

    // ESC 退出全屏
    const handleKeyDown = (event) => {
      if (event.key === 'Escape' && isTabMaximized.value) {
        restoreTab()
      }
    }

    const handleMaximizedMouseMove = (event) => {
      if (!isTabMaximized.value) return
      if (event?.clientY <= 56) {
        showMaximizeRestoreButton.value = true
        if (maximizeRestoreHideTimer.value) {
          window.clearTimeout(maximizeRestoreHideTimer.value)
          maximizeRestoreHideTimer.value = null
        }
        return
      }
      if (showMaximizeRestoreButton.value && !maximizeRestoreHideTimer.value) {
        maximizeRestoreHideTimer.value = window.setTimeout(() => {
          showMaximizeRestoreButton.value = false
          maximizeRestoreHideTimer.value = null
        }, 220)
      }
    }

    const handleMaximizedMouseLeave = () => {
      if (!isTabMaximized.value) return
      showMaximizeRestoreButton.value = false
      if (maximizeRestoreHideTimer.value) {
        window.clearTimeout(maximizeRestoreHideTimer.value)
        maximizeRestoreHideTimer.value = null
      }
    }

    // 获取当前标签页标题
    const getTabTitle = (tab) => {
      return resolveDashboardTabTitle(tab, t)
    }

    /**
     * 在新标签页打开设计中心或数据中心。
     * @param {object} tab - 标签页配置
     */
    const openExternalTab = (tab) => {
      if (!tab?.props?.appType) return
      const url = router.resolve({
        name: 'workspace-micro-app',
        params: {
          projectId: String(tab.props.project.id),
          appType: tab.props.appType,
        },
      }).href
      const newWindow = window.open(url, '_blank', 'noopener')
      if (newWindow) newWindow.opener = null
    }

    /**
     * 切换侧边栏折叠状态。
     */
    const toggleSidebar = () => {
      appStore.toggleSidebar()
    }

    watch(locale, () => {
      tabs.value = tabs.value.map((tab) => {
        return {
          ...tab,
          title: resolveDashboardTabTitle(tab, t),
        }
      })
    })

    watch(
      [tabs, activeTab],
      () => {
        persistTabState()
      },
      { deep: true },
    )

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
      hasTabPermission,
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
      handleWorkspaceOpenRequest,
      handleWorkspaceCloseRequest,
      handleWorkspaceSceneCommitted,
      closeTab,
      maximizeTab,
      restoreTab,
      handleMaximizedMouseMove,
      handleMaximizedMouseLeave,
      getTabTitle,
      openExternalTab,
      toggleSidebar,
      isTabMaximized,
      showMaximizeRestoreButton,
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

.sidebar-menu-label {
  display: inline-block;
  white-space: nowrap;
  overflow: hidden;
  transform-origin: left center;
  transition:
    max-width 260ms cubic-bezier(0.22, 1, 0.36, 1),
    opacity 180ms ease,
    transform 220ms cubic-bezier(0.22, 1, 0.36, 1);
}

.sidebar-menu-label-expanded {
  max-width: 120px;
  opacity: 1;
  transform: translateX(0);
}

.sidebar-menu-label-collapsed {
  max-width: 0;
  opacity: 0;
  transform: translateX(-4px);
}

.sidebar-toggle-button {
  opacity: 0;
  pointer-events: none;
}

.sidebar-panel:hover .sidebar-toggle-button,
.sidebar-panel:focus-within .sidebar-toggle-button {
  opacity: 1;
  pointer-events: auto;
}

/* 标签页样式 */
.dashboard-tabs :deep(.el-tabs__header) {
  margin: 0;
  padding: 0;
}

.dashboard-tabs :deep(.el-tabs__nav-wrap) {
  margin-bottom: 0;
}

.dashboard-tabs :deep(.el-tabs__nav-wrap::after) {
  display: none;
}

.dashboard-tabs :deep(.el-tabs__nav) {
  border-radius: 6px;
  border-bottom: 1px solid rgb(229 231 235);
  overflow: hidden;
}

html.dark .dashboard-tabs :deep(.el-tabs__nav),
[data-theme='dark'] .dashboard-tabs :deep(.el-tabs__nav) {
  border-bottom-color: rgb(55 65 81);
}

.dashboard-tabs :deep(.el-tabs__item) {
  border-radius: 4px 4px 0 0;
  color: rgb(55 65 81);
  padding: 6px 12px;
  height: 36px;
  line-height: 22px;
  font-size: 14px;
}

.dashboard-tabs :deep(.el-tabs__item:first-child) {
  padding-left: 10px;
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

html.dark .dashboard-tabs :deep(.el-tabs__item),
[data-theme='dark'] .dashboard-tabs :deep(.el-tabs__item) {
  color: rgb(209 213 219);
}

html.dark .dashboard-tabs :deep(.el-tabs__item:hover),
[data-theme='dark'] .dashboard-tabs :deep(.el-tabs__item:hover) {
  color: rgb(147 197 253);
  background-color: rgb(31 41 55);
}

html.dark .dashboard-tabs :deep(.el-tabs__item.is-active),
[data-theme='dark'] .dashboard-tabs :deep(.el-tabs__item.is-active) {
  color: rgb(191 219 254);
  background-color: rgb(30 58 138 / 0.28);
  border-bottom-color: rgb(30 58 138 / 0.28);
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
</style>
