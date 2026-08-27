import { test } from 'vitest'
import assert from 'node:assert/strict'

import { createDatacenterRoutes, resolveDatacenterRouteBase } from '../src/router/route-config'

test('模块切换保留 debug 路由前缀', () => {
  assert.equal(resolveDatacenterRouteBase('/debug/access-source/source-1/workbench'), '/debug')
  assert.equal(resolveDatacenterRouteBase('/access-source/source-1/workbench'), '')
})

test('生产态不会注册 datacenter debug 路由', () => {
  const DataCenterComponent = { name: 'DataCenterStub' }
  const routes = createDatacenterRoutes({
    DataCenterComponent,
    enableDebugRoute: false,
  })

  // 路由结构：根 redirect + 正式模块路由
  assert.equal(routes.length, 2)

  // 根路径 redirect
  assert.equal(routes[0].path, '/')
  assert.equal(typeof routes[0].redirect, 'function')
  const redirect = routes[0].redirect
  assert.equal(typeof redirect, 'function')
  assert.deepEqual((redirect as Function)({ query: { keyword: 'demo' } }), {
    path: '/datapoint',
    query: { keyword: 'demo' },
  })

  // 正式模块路由
  assert.deepEqual(routes[1], {
    path: '/:module(datapoint|access-source|industrial-collector|history-storage|compute|alarm)/:objectId?/:tab?',
    name: 'datacenter',
    component: DataCenterComponent,
    meta: {
      titleKey: 'route.datacenter',
      requiresAuth: true,
    },
  })
})
