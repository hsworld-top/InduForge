import { afterEach, describe, expect, it } from 'vitest'
import { i18n } from '@/i18n'
import { canEditRuntimeConstraint, getPageInspectorSections } from './page-inspector-sections'

describe('page-inspector-sections', () => {
  afterEach(() => {
    i18n.global.locale.value = 'zh'
  })

  it('returns localized section titles', () => {
    expect(getPageInspectorSections().map((section) => section.key)).toEqual([
      'identity',
      'route',
      'viewport',
      'visual',
      'runtime',
      'runtimeAccess',
    ])
    expect(getPageInspectorSections().map((section) => section.title)).toEqual([
      '页面身份',
      '路由与入口',
      '布局与适配',
      '视觉',
      '运行控制',
      '运行态权限控制',
    ])

    i18n.global.locale.value = 'en'

    expect(getPageInspectorSections().map((section) => section.title)).toEqual([
      'Identity',
      'Route & Entry',
      'Layout & Adaptation',
      'Visual',
      'Runtime',
      '运行态权限控制',
    ])
  })

  it('contains required runtime fields', () => {
    const runtimeSection = getPageInspectorSections().find((section) => section.key === 'runtime')
    expect(runtimeSection?.fields).toEqual(['openMode', 'popup', 'cacheMode', 'preloadMode'])
  })

  it('runtime constraint editability follows autoFit', () => {
    expect(canEditRuntimeConstraint(true)).toBe(true)
    expect(canEditRuntimeConstraint(false)).toBe(false)
  })
})
