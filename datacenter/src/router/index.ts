// @ts-nocheck
import { createRouter, createWebHistory } from 'vue-router'
import { watch } from 'vue'
import { debugProjectAPI } from '@/api/debug-project.api'
import { STORAGE_KEYS } from '@/constants/index'
import { Storage } from '@/utils/storage'
import { datacenterLocale, getDatacenterRouteTitle } from '@/i18n/runtime'
// import DataCenter from "../views/DataCenter.vue"; // 原版本
import DataCenter from '../views/DataCenterNew.vue' // 重构版本
import { buildRouteRuntimeUrl } from './entrypoint-url'
import { resolveDatacenterDebugProjectMeta } from './debug-project'
import { createDatacenterRoutes } from './route-config'
import {
  applyThemeToDocument,
  buildIdeLoginUrl,
  buildIdeRestoreUrl,
  hasReusableTopLevelSession,
  resolveIdeOriginFromRuntime,
  restoreTopLevelHandoffRecord,
  shouldRedirectTopLevelToIde,
  shouldUseDebugMode,
  waitForHostBootstrap,
} from '../runtime/host-bootstrap'

const routes = createDatacenterRoutes({
  DataCenterComponent: DataCenter,
})

const router = createRouter({
  history: createWebHistory('/datacenter/'),
  routes,
})

export function registerDatacenterBeforeEachGuard(
  targetRouter,
  {
    getCurrentUrl = () => window.location.href,
    getIdeOrigin = (currentUrl) =>
      resolveIdeOriginFromRuntime({
        currentUrl,
        referrer: document.referrer,
      }),
    isTopLevelWindow = () => window.parent === window,
    navigateToUrl = (url) => {
      window.location.href = url
    },
    resolveDefaultDebugProject = () => debugProjectAPI.resolveDefaultProjectByName(),
    waitForBootstrap = waitForHostBootstrap,
  } = {},
) {
  return targetRouter.beforeEach(async (to, from, next) => {
    document.title = `${getDatacenterRouteTitle(to.name)} - ProjectIDE`
    applyThemeToDocument(Storage.getTheme())

    const runtimeUrl = buildRouteRuntimeUrl(getCurrentUrl(), to.path)
    const ideOrigin = getIdeOrigin(runtimeUrl.toString())
    const handoff = runtimeUrl.searchParams.get('handoff')
    const isDebugRoute = shouldUseDebugMode(runtimeUrl.pathname) || to.meta.requiresAuth === false
    const isTopLevel = isTopLevelWindow()
    const restoredTopLevelHandoff =
      isTopLevel && Boolean(handoff) && restoreTopLevelHandoffRecord(handoff)
    const allowReusableSession = hasReusableTopLevelSession({
      handoff: restoredTopLevelHandoff ? null : handoff,
    })

    if (shouldRedirectTopLevelToIde(runtimeUrl.pathname, isTopLevel, allowReusableSession)) {
      navigateToUrl(buildIdeRestoreUrl(handoff, ideOrigin))
      return
    }

    if (isDebugRoute) {
      to.meta.project = await resolveDatacenterDebugProjectMeta({
        targetUrl: runtimeUrl.toString(),
        resolveDefaultDebugProject,
        getStoredProjectId: () => Storage.getProjectId(),
        getStoredTenantId: () => Storage.getTenantId(),
        setProjectId: (value) => Storage.setProjectId(value),
        setTenantId: (value) => Storage.setTenantId(value),
      })
      next()
      return
    }

    let token = Storage.getToken()
    let bootstrapReady = true
    if (handoff && !isTopLevel) {
      /**
       * handoff 表示宿主要求恢复新的工程上下文。
       * 等待 bootstrap 前先清掉旧工程与租户，避免超时时继续读到上一工程残留。
       * 顶层独立打开时，main.ts 已经尝试从同源 handoff 票据恢复工程上下文，
       * 这里不能再清理，否则会再次触发回 IDE 恢复。
       */
      Storage.removeProjectId()
      Storage.remove(STORAGE_KEYS.TENANT_ID)
    }

    if (!token || handoff) {
      bootstrapReady = await waitForBootstrap()
      token = Storage.getToken()
    }

    if (!token) {
      navigateToUrl(buildIdeLoginUrl(runtimeUrl.toString(), ideOrigin))
      return
    }

    if (!bootstrapReady) {
      // handoff 超时只负责解除等待；后续由 projectId 缺失分支回到 IDE 恢复，不再误导到登录页。
    }

    const projectId = Storage.getProjectId()
    const tenantId = Storage.getTenantId()

    if (!projectId) {
      navigateToUrl(buildIdeRestoreUrl(handoff, ideOrigin))
      return
    }

    to.meta.project = {
      id: projectId,
      tenantId,
    }

    next()
  })
}

registerDatacenterBeforeEachGuard(router)

watch(
  datacenterLocale,
  () => {
    const currentRoute = router.currentRoute.value
    document.title = `${getDatacenterRouteTitle(currentRoute.name)} - ProjectIDE`
  },
  { immediate: true },
)

export default router
