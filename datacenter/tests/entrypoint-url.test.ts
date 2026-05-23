import { describe, expect, it } from 'vitest'
import { buildRouteRuntimeUrl } from '@/router/entrypoint-url'

describe('entrypoint-url', () => {
  it('内部模块切换不继承当前地址栏残留 handoff', () => {
    const url = buildRouteRuntimeUrl(
      'http://datacenter.example/datacenter/datapoint?handoff=handoff-old',
      '/access-source',
    )

    expect(url.toString()).toBe('http://datacenter.example/datacenter/access-source')
  })

  it('根路径入口仍保持 datacenter 正式入口路径', () => {
    const url = buildRouteRuntimeUrl('http://datacenter.example/datacenter/?handoff=handoff-1', '/')

    expect(url.toString()).toBe('http://datacenter.example/datacenter/')
  })
})
