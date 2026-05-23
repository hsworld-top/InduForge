// @ts-nocheck
import assert from 'node:assert/strict'
import { test } from 'vitest'

import {
  normalizeRuntimeGrantPayload,
  summarizeRuntimeGrant,
} from '../src/utils/runtime-permission-grants'

test('normalizeRuntimeGrantPayload 会输出 allow/deny/inherit 结构', () => {
  assert.deepEqual(
    normalizeRuntimeGrantPayload({
      allowRoles: ['OPERATOR'],
      denyRoles: ['VIEWER'],
      inherit: false,
    }),
    {
      allowRoles: ['OPERATOR'],
      denyRoles: ['VIEWER'],
      inherit: false,
    },
  )
})

test('summarizeRuntimeGrant 会返回中文摘要', () => {
  assert.equal(
    summarizeRuntimeGrant({
      allowRoles: ['OPERATOR', 'ADMIN'],
      denyRoles: [],
      inherit: true,
    }),
    '允许 2 / 拒绝 0 / 继承',
  )
})

test('normalizeRuntimeGrantPayload 会保留 allow/deny 原始交集，仅做去重和结构归一化', () => {
  assert.deepEqual(
    normalizeRuntimeGrantPayload({
      allowRoles: ['OPERATOR', 'VIEWER', 'ADMIN'],
      denyRoles: ['VIEWER', 'GUEST'],
      inherit: true,
    }),
    {
      allowRoles: ['OPERATOR', 'VIEWER', 'ADMIN'],
      denyRoles: ['VIEWER', 'GUEST'],
      inherit: true,
    },
  )
})
