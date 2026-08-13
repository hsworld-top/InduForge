/**
 * 设计器路由配置。
 *
 * 正式入口只允许两种启动方式：
 * - 被 IDE 作为 Wujie 子应用加载，并由 props 注入工程上下文；
 * - `/designer/debug` 独立调试路由，使用固定默认 UI（light + zh）。
 */

import type { NavigationGuardNext, RouteLocationNormalized, Router } from 'vue-router'
import { createRouter, createWebHistory } from 'vue-router'
import type { EditorUiStore } from '@/stores/editor-ui-store'
import { getEditorUiStore } from '@/stores/editor-ui-store'
import { debugProjectApi } from '@/services/debugProjectApi'
import { isDesignerDebugRouteEnabled } from '@/runtime/debug-route'
import { Storage } from '@/utils/storage'
import { getMicroAppContext, isWujieMicroApp } from '@/runtime/wujie-context'
import { shouldSyncEditorUiForPath } from './runtime-settings'

declare module 'vue-router' {
  interface RouteMeta {
    title?: string
    requiresAuth?: boolean
    project?: {
      id: string | null | undefined
      tenantId: string | null | undefined
    }
  }
}

type DesignerDebugProjectMeta = {
  id: string | null
  tenantId: string | null
}

type ResolveDefaultDebugProject = () => Promise<DesignerDebugProjectMeta | null>

export function createDesignerRoutes(enableDebugRoute?: boolean) {
  const routes = [
    {
      path: '/',
      name: 'Designer',
      component: () => import('@/ui/workspaces/DesignerWorkspaceView.vue'),
      meta: {
        title: '设计器',
        requiresAuth: true,
      },
    },
  ]

  if (isDesignerDebugRouteEnabled(enableDebugRoute)) {
    routes.push({
      path: '/debug',
      name: 'DesignerDebug',
      component: () => import('@/ui/workspaces/DesignerWorkspaceView.vue'),
      meta: {
        title: '设计器调试',
        requiresAuth: false,
      },
    })
  }

  return routes
}

const routes = createDesignerRoutes()

const router = createRouter({
  history: createWebHistory('/designer/'),
  routes,
})

function clearEditorUiThemeEffects(): void {
  document.documentElement.classList.remove('dark')
  document.documentElement.removeAttribute('data-theme')
}

type RuntimeRouteEffectsDependencies = {
  editorUi: EditorUiStore
}

type DesignerBeforeEachGuardDependencies = {
  navigateToUrl?: (url: string) => void
  resolveDefaultDebugProject?: ResolveDefaultDebugProject
}

function asNonEmptyString(value: unknown): string | null {
  return typeof value === 'string' && value.trim().length > 0 ? value.trim() : null
}

function asFirstNonEmptyString(value: unknown): string | null {
  if (Array.isArray(value)) {
    return asFirstNonEmptyString(value[0])
  }
  return asNonEmptyString(value)
}

async function resolveDesignerDebugProjectMeta(
  targetUrl: string,
  resolveDefaultDebugProject: ResolveDefaultDebugProject,
): Promise<DesignerDebugProjectMeta> {
  const url = new URL(targetUrl)
  const projectIdFromUrl =
    asNonEmptyString(url.searchParams.get('pid')) ?? asNonEmptyString(url.searchParams.get('id'))
  const tenantIdFromUrl = asNonEmptyString(url.searchParams.get('tenant'))

  if (projectIdFromUrl) {
    Storage.setProjectId(projectIdFromUrl)
    if (tenantIdFromUrl) {
      Storage.setTenantId(tenantIdFromUrl)
    }

    return {
      id: projectIdFromUrl,
      tenantId: tenantIdFromUrl ?? Storage.getTenantId(),
    }
  }

  const defaultDebugProject = await resolveDefaultDebugProject()
  if (defaultDebugProject?.id) {
    Storage.setProjectId(defaultDebugProject.id)
    if (defaultDebugProject.tenantId) {
      Storage.setTenantId(defaultDebugProject.tenantId)
    }

    return {
      id: defaultDebugProject.id,
      tenantId: defaultDebugProject.tenantId ?? Storage.getTenantId(),
    }
  }

  return {
    id: Storage.getProjectId(),
    tenantId: Storage.getTenantId(),
  }
}

export function applyRuntimeRouteEffects(
  routePath: string,
  _currentUrl: string,
  { editorUi }: RuntimeRouteEffectsDependencies,
): void {
  if (!shouldSyncEditorUiForPath(routePath)) {
    clearEditorUiThemeEffects()
    return
  }

  // 设计态路由只把当前 UI 状态重新同步到 DOM，避免路由切换时覆盖宿主已经下发的主题/语言。
  editorUi.applyThemeToDom()
}

function syncRuntimeSettings(routePath = window.location.pathname): void {
  applyRuntimeRouteEffects(routePath, window.location.href, {
    editorUi: getEditorUiStore(),
  })
}

export function registerDesignerBeforeEachGuard(
  targetRouter: Router,
  dependencies: DesignerBeforeEachGuardDependencies = {},
): () => void {
  const navigateToUrl =
    dependencies.navigateToUrl ?? ((url: string) => window.location.replace(url))
  const resolveDefaultDebugProject =
    dependencies.resolveDefaultDebugProject ?? (() => debugProjectApi.resolveDefaultProjectByName())
  return targetRouter.beforeEach(
    async (to: RouteLocationNormalized, _from, next: NavigationGuardNext) => {
      document.title = `${to.meta.title || '设计器'} - InduForge`

      syncRuntimeSettings(to.path)

      if (to.name === 'DesignerDebug') {
        to.meta.project = await resolveDesignerDebugProjectMeta(
          window.location.href,
          resolveDefaultDebugProject,
        )
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
    },
  )
}

registerDesignerBeforeEachGuard(router)

export default router
