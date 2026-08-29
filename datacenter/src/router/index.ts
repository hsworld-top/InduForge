import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import { watch } from 'vue'
import { debugProjectAPI } from '@/api/debug-project.api'
import { Storage } from '@/utils/storage'
import { datacenterLocale, getDatacenterRouteTitle } from '@/i18n/runtime'
// import DataCenter from "../views/DataCenter.vue"; // 原版本
import DataCenter from '../views/DataCenterNew.vue' // 重构版本
import { resolveDatacenterDebugProjectMeta } from './debug-project'
import { createDatacenterRoutes } from './route-config'
import { getMicroAppContext, isWujieMicroApp } from '../runtime/wujie-context'
import { applyDatacenterTheme } from '@/theme/runtime'

const routes = createDatacenterRoutes({
  DataCenterComponent: DataCenter,
})

const router = createRouter({
  history: createWebHistory('/datacenter/'),
  routes: routes as RouteRecordRaw[],
})

export function registerDatacenterBeforeEachGuard(
  targetRouter,
  {
    navigateToUrl = (url) => {
      window.location.href = url
    },
    resolveDefaultDebugProject = () => debugProjectAPI.resolveDefaultProjectByName(),
  } = {},
) {
  return targetRouter.beforeEach(async (to, from, next) => {
    document.title = `${getDatacenterRouteTitle(to.name)} - ProjectIDE`
    applyDatacenterTheme(getMicroAppContext()?.theme ?? Storage.getTheme())

    const isDebugRoute = to.meta.requiresAuth === false

    if (isDebugRoute) {
      to.meta.project = await resolveDatacenterDebugProjectMeta({
        targetUrl: window.location.href,
        resolveDefaultDebugProject,
        getStoredProjectId: () => Storage.getProjectId(),
        getStoredTenantId: () => Storage.getTenantId(),
        setProjectId: (value) => Storage.setProjectId(value),
        setTenantId: (value) => Storage.setTenantId(value),
      })
      next()
      return
    }

    const context = getMicroAppContext()
    if (!isWujieMicroApp() || !context?.projectId) {
      next(false)
      navigateToUrl('/dashboard')
      return
    }

    to.meta.project = {
      id: context.projectId,
      tenantId: context.tenantId ?? null,
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
