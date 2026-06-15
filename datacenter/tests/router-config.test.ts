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

  // v2 路由结构：根 redirect + 模块路由 + 旧 ID 兼容 redirect
  assert.equal(routes.length, 3)

  // 根路径 redirect
  assert.equal(routes[0].path, '/')
  assert.equal(typeof routes[0].redirect, 'function')
  assert.deepEqual(routes[0].redirect({ query: { handoff: 'handoff-1' } }), {
    path: '/datapoint',
    query: { handoff: 'handoff-1' },
  })

  // 正式模块路由
  assert.deepEqual(routes[1], {
    path: '/:module(datapoint|access-source|storage-policy|compute|alarm)/:objectId?/:tab?',
    name: 'datacenter',
    component: DataCenterComponent,
    meta: {
      titleKey: 'route.datacenter',
      requiresAuth: true,
    },
  })

  // 旧模块 ID 兼容 redirect（函数类型，只检查路径）
  assert.equal(
    routes[2].path,
    '/:legacyModule(datapoints|access-sources|storage-policies|compute-units|alarm-units)/:rest(.*)?',
  )
  assert.equal(typeof routes[2].redirect, 'function')
})
