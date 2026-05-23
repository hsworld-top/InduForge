/**
 * 设计器主入口。
 *
 * 关键职责：
 * - 初始化 Vue 应用、Pinia、Router、Element Plus；
 * - 为正式入口建立宿主 bootstrap 握手；
 * - 统一处理来自宿主的认证、主题、语言同步消息。
 */

import ElementPlus from 'element-plus'
import { createPinia } from 'pinia'
import { createApp } from 'vue'
import { i18n } from './i18n'
import App from './App.vue'
import { registerAllDescriptors } from './materials'
import * as descriptorRegistry from './editor-core/descriptors/registry'
import { initDescriptorRegistry } from './editor-core/document/factory'
import { registerBuiltinComponents } from './editor-core/registry/builtin-manifests'
import router from './router'
import {
  getTrustedHostOriginSet,
  getTrustedHostSources,
  handleAuthRefreshedMessage,
  handleBootstrapResponseMessage,
  initializeDesignerHostBootstrap,
  isTrustedHostMessage,
  postAppBootstrapRequest,
  resolveDesignerIdeOriginFromRuntime,
  type TrustedMessageSource,
} from './runtime/host-bootstrap'
import { getEditorUiStore } from './stores/editor-ui-store'
import { shouldSyncEditorUiForPath } from './router/runtime-settings'
import 'element-plus/dist/index.css'
import './assets/styles/main.css'

function isThemeUpdatePayload(data: unknown): data is {
  type: string
  theme: 'light' | 'dark'
} {
  if (data === null || typeof data !== 'object') return false
  const o = data as Record<string, unknown>
  if (o.type !== 'THEME_UPDATE') return false
  const theme = o.theme
  return theme === 'light' || theme === 'dark'
}

function isLocaleUpdatePayload(data: unknown): data is {
  type: string
  locale: 'zh' | 'en'
} {
  if (data === null || typeof data !== 'object') return false
  const o = data as Record<string, unknown>
  if (o.type !== 'LOCALE_UPDATE') return false
  const locale = o.locale
  return locale === 'zh' || locale === 'en'
}

type RuntimeMessageHandlerDependencies = {
  editorUi: ReturnType<typeof getEditorUiStore>
  i18n: typeof i18n
  getPathname?: () => string
  getTrustedOriginSet?: () => Set<string>
  getTrustedSources?: () => TrustedMessageSource[]
}

export function resolveTrustedMessageSources({
  locationOrigin,
  referrer,
}: {
  locationOrigin: string
  referrer: string
}): Set<string> {
  const trustedOrigins = new Set<string>([locationOrigin])

  if (!referrer) {
    return trustedOrigins
  }

  try {
    trustedOrigins.add(new URL(referrer).origin)
  } catch {
    // referrer 不可解析时回退到当前 origin，避免把异常输入放大成信任边界。
  }

  return trustedOrigins
}

export function createRuntimeMessageHandler({
  editorUi,
  i18n,
  getPathname = () => window.location.pathname,
  getTrustedOriginSet = () => getTrustedHostOriginSet(),
  getTrustedSources = () => getTrustedHostSources(),
}: RuntimeMessageHandlerDependencies): (event: MessageEvent) => void {
  return (event: MessageEvent) => {
    if (
      !isTrustedHostMessage(event, {
        trustedOrigins: getTrustedOriginSet(),
        trustedSources: getTrustedSources(),
      })
    ) {
      return
    }

    const data = event.data

    if (
      handleBootstrapResponseMessage(data, {
        allowUiSync: shouldSyncEditorUiForPath(getPathname()),
        editorUi,
      })
    ) {
      return
    }

    if (handleAuthRefreshedMessage(data)) {
      return
    }

    if (!shouldSyncEditorUiForPath(getPathname())) {
      return
    }

    if (isThemeUpdatePayload(data)) {
      editorUi.setTheme(data.theme)
      return
    }

    if (isLocaleUpdatePayload(data)) {
      editorUi.setLocale(data.locale)
      i18n.global.locale.value = data.locale
      registerBuiltinComponents()
      return
    }
  }
}

export function bootstrapDesignerApp(): void {
  const hostBootstrap = initializeDesignerHostBootstrap({
    ideOrigin: resolveDesignerIdeOriginFromRuntime(),
  })
  if (hostBootstrap.plan.shouldRedirectToIde && hostBootstrap.plan.ideRedirectUrl) {
    window.location.replace(hostBootstrap.plan.ideRedirectUrl)
    return
  }

  const app = createApp(App)
  const pinia = createPinia()

  app.use(pinia)

  const editorUi = getEditorUiStore()

  registerBuiltinComponents()
  registerAllDescriptors()
  initDescriptorRegistry(descriptorRegistry)

  window.addEventListener(
    'message',
    createRuntimeMessageHandler({
      editorUi,
      i18n,
      getPathname: () => window.location.pathname,
      getTrustedOriginSet: () => getTrustedHostOriginSet(),
      getTrustedSources: () => getTrustedHostSources(),
    }),
  )

  app.use(router)
  app.use(i18n)
  app.use(ElementPlus)
  app.provide('editorUi', editorUi)

  if (hostBootstrap.plan.shouldWaitForBootstrap) {
    postAppBootstrapRequest()
  }

  app.mount('#app')

  const initialLoading = document.getElementById('app-loading')
  if (initialLoading) {
    if (window.location.pathname.includes('/preview')) {
      initialLoading.remove()
    } else {
      requestAnimationFrame(() => {
        initialLoading.remove()
      })
    }
  }
}

if (!import.meta.env.VITEST) {
  bootstrapDesignerApp()
}
