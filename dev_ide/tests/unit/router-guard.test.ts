import { describe, expect, it } from 'vitest'
import type { RouteRecordNormalized } from 'vue-router'
import router from '@/router'

describe('router', () => {
  it('包含 login 路由', () => {
    const route = router
      .getRoutes()
      .find((item: RouteRecordNormalized) => item.path === '/login')
    expect(route).toBeTruthy()
  })
})
