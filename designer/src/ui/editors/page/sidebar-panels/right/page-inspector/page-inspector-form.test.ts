import { describe, expect, it } from 'vitest'
import { hydratePageInspectorForm } from '../page-inspector-config'

describe('page-inspector-form', () => {
  it('hydrates form state from legacy flat config fields', () => {
    const form = hydratePageInspectorForm({
      name: '旧页面',
      role: 'normal',
      routePath: '/legacy',
      config: {
        description: 'legacy',
        width: 1920,
        height: 1080,
        autoFit: true,
        lockAspectRatio: true,
        enableMinSize: true,
        windowStyle: 'normal',
        permissionDesc: '2item',
        background: { kind: 'color', value: '#000000' },
      },
    })

    expect(form.description).toBe('legacy')
    expect(form.routePath).toBe('/legacy')
    expect(form.openMode).toBe('normal')
    expect(form.minWidth).toBe(0)
    expect(form.minHeight).toBe(0)
    expect(form.permissionSummary).toBe('2item')
    expect(form.backgroundValue).toBe('#000000')
  })

  it('prefers grouped config fields over legacy fallback fields', () => {
    const form = hydratePageInspectorForm({
      name: '新页面',
      role: 'normal',
      routePath: '/legacy-path',
      parentRoutePath: '/group',
      config: {
        description: 'legacy',
        width: 1920,
        height: 1080,
        windowStyle: 'cover',
        permissionDesc: '1item',
        meta: {
          title: '运行标题',
          description: 'grouped',
        },
        route: {
          mode: 'manual',
          path: '/group/custom-page',
          slug: 'custom-page',
        },
        viewport: {
          preset: 'tablet',
          width: 1024,
          height: 768,
          autoFit: false,
          lockAspectRatio: false,
          minWidth: 800,
          minHeight: 600,
          overflowMode: 'hidden',
        },
        runtime: {
          openMode: 'popup',
          permission: {
            summary: '3item',
          },
          popup: {
            width: 800,
            height: 480,
            center: false,
            maskClosable: false,
          },
          cacheMode: 'cache',
          preloadMode: 'eager',
        },
      },
    })

    expect(form.title).toBe('运行标题')
    expect(form.description).toBe('grouped')
    expect(form.routeMode).toBe('manual')
    expect(form.routePath).toBe('/group/custom-page')
    expect(form.routeSlug).toBe('custom-page')
    expect(form.width).toBe(1024)
    expect(form.height).toBe(768)
    expect(form.overflowMode).toBe('hidden')
    expect(form.openMode).toBe('popup')
    expect(form.permissionSummary).toBe('3item')
    expect(form.cacheMode).toBe('cache')
    expect(form.preloadMode).toBe('eager')
  })
})
