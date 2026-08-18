// @ts-nocheck
import { test } from 'vitest'
import assert from 'node:assert/strict'

import { createDatacenterRoutes } from '../src/router/route-config'

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
  assert.deepEqual(routes[0].redirect({ query: { keyword: 'demo' } }), {
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
