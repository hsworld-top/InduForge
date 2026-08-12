import type { EditorLocale, EditorTheme } from '@/stores/editor-ui-store'

export interface RuntimeSettingsFallback {
  theme: EditorTheme
  locale: EditorLocale
}

export interface RuntimeSettingsResult {
  theme: EditorTheme
  locale: EditorLocale
}

export interface RuntimeRouteSyncPlan {
  tokenFromUrl: string | null
  refreshTokenFromUrl: string | null
  projectIdFromUrl: string | null
  tenantIdFromUrl: string | null
  runtimeSettings: RuntimeSettingsResult
  shouldSyncEditorUi: boolean
  shouldReplaceUrl: boolean
  cleanedUrl: string
}

const isEditorTheme = (value: unknown): value is EditorTheme =>
  value === 'light' || value === 'dark'

const isEditorLocale = (value: unknown): value is EditorLocale => value === 'zh' || value === 'en'

/**
 * 解析运行时 URL 中的编辑器主题与语言，非法值回退到传入兜底值。
 */
export function parseRuntimeSettings(
  url: URL,
  fallback: RuntimeSettingsFallback,
): RuntimeSettingsResult {
  const themeParam = url.searchParams.get('theme')
  const localeParam = url.searchParams.get('locale')

  return {
    theme: isEditorTheme(themeParam) ? themeParam : fallback.theme,
    locale: isEditorLocale(localeParam) ? localeParam : fallback.locale,
  }
}

/**
 * 统一解析路由进入时的运行时设置消费结果，便于路由守卫和测试共享。
 */
export function resolveRuntimeRouteSyncPlan(
  url: URL,
  routePath: string,
  fallback: RuntimeSettingsFallback,
): RuntimeRouteSyncPlan {
  const tokenFromUrl = url.searchParams.get('token')
  const refreshTokenFromUrl = url.searchParams.get('refreshToken')
  const projectIdFromUrl = url.searchParams.get('pid') || url.searchParams.get('id')
  const tenantIdFromUrl = url.searchParams.get('tenant')
  const runtimeSettings = parseRuntimeSettings(url, fallback)
  const cleanedUrl = new URL(url.toString())
  let shouldReplaceUrl = false

  ;['token', 'refreshToken', 'theme', 'locale'].forEach((key) => {
    if (cleanedUrl.searchParams.has(key)) {
      cleanedUrl.searchParams.delete(key)
      shouldReplaceUrl = true
    }
  })

  const nextQuery = cleanedUrl.searchParams.toString()

  return {
    tokenFromUrl,
    refreshTokenFromUrl,
    projectIdFromUrl,
    tenantIdFromUrl,
    runtimeSettings,
    shouldSyncEditorUi: shouldSyncEditorUiForPath(routePath),
    shouldReplaceUrl,
    cleanedUrl: nextQuery ? `${cleanedUrl.pathname}?${nextQuery}` : cleanedUrl.pathname,
  }
}

/**
 * 仅设计器编辑路由允许消费编辑器 UI 的主题/语言同步。
 */
export function shouldSyncEditorUiForPath(pathname: string): boolean {
  return !pathname.includes('/preview')
}

/**
 * 预览路由下 Element Plus locale 使用应用级兜底值，避免继承编辑器壳层状态。
 */
export function resolveScopedLocaleForPath<T>(
  pathname: string,
  editorLocale: T,
  fallbackLocale: T,
): T {
  return shouldSyncEditorUiForPath(pathname) ? editorLocale : fallbackLocale
}
