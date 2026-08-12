import { describe, expect, it } from 'vitest'
import {
  parseRuntimeSettings,
  resolveRuntimeRouteSyncPlan,
  resolveScopedLocaleForPath,
  shouldSyncEditorUiForPath,
} from './runtime-settings'

describe('runtime-settings', () => {
  it('解析 URL 时会消费 locale 参数', () => {
    const url = new URL('http://localhost/designer/?theme=dark&locale=en&pid=p1')

    const result = parseRuntimeSettings(url, { theme: 'light', locale: 'zh' })

    expect(result.theme).toBe('dark')
    expect(result.locale).toBe('en')
  })

  it('preview 路由不参与编辑器 UI 同步', () => {
    expect(shouldSyncEditorUiForPath('/')).toBe(true)
    expect(shouldSyncEditorUiForPath('/preview')).toBe(false)
    expect(shouldSyncEditorUiForPath('/designer/preview')).toBe(false)
  })

  it('preview 路由下 Element Plus locale 会回退到应用级默认值', () => {
    expect(resolveScopedLocaleForPath('/', 'editor-locale', 'fallback-locale')).toBe(
      'editor-locale',
    )
    expect(resolveScopedLocaleForPath('/preview', 'editor-locale', 'fallback-locale')).toBe(
      'fallback-locale',
    )
  })

  it('路由接线计划会消费并清理 theme/locale 查询参数', () => {
    const url = new URL(
      'http://localhost/designer/?token=t1&refreshToken=rt1&theme=dark&locale=en&pid=p1&tenant=t1',
    )

    const plan = resolveRuntimeRouteSyncPlan(url, '/', { theme: 'light', locale: 'zh' })

    expect(plan.shouldSyncEditorUi).toBe(true)
    expect(plan.runtimeSettings).toEqual({ theme: 'dark', locale: 'en' })
    expect(plan.cleanedUrl).toBe('/designer/?pid=p1&tenant=t1')
    expect(plan.shouldReplaceUrl).toBe(true)
    expect(plan.tokenFromUrl).toBe('t1')
    expect(plan.refreshTokenFromUrl).toBe('rt1')
  })

  it('preview 路由接线计划不会同步 editorUi，并会清理运行时查询参数', () => {
    const url = new URL('http://localhost/designer/preview?theme=dark&locale=en&id=p1')

    const plan = resolveRuntimeRouteSyncPlan(url, '/preview', { theme: 'light', locale: 'zh' })

    expect(plan.shouldSyncEditorUi).toBe(false)
    expect(plan.runtimeSettings).toEqual({ theme: 'dark', locale: 'en' })
    expect(plan.cleanedUrl).toBe('/designer/preview?id=p1')
    expect(plan.shouldReplaceUrl).toBe(true)
  })
})
