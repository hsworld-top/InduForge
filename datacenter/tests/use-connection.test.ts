import { test } from 'vitest'
import assert from 'node:assert/strict'

import { ConnectionPageSchema, ConnectionSchema } from '../src/api/schemas/connection.schema'

const connection = {
  id: '11111111-1111-1111-1111-111111111111',
  projectId: '22222222-2222-2222-2222-222222222222',
  tenantId: '33333333-3333-3333-3333-333333333333',
  name: 'test1',
  type: 'relational',
  enabled: true,
  configurationState: 'ready',
  testCapability: { status: 'supported' },
  lastTest: { status: 'not_tested' },
  config: {},
  secretStatus: {},
  displayOrder: 0,
  variableCount: 0,
  createdAt: '2026-08-25T00:00:00Z',
  updatedAt: '2026-08-25T00:00:00Z',
} as const

test('连接模型校验拆分后的启用、完整性和测试状态', () => {
  const payload = ConnectionSchema.parse(connection)
  assert.equal(payload.configurationState, 'ready')
  assert.equal(payload.lastTest.status, 'not_tested')
  assert.equal('status' in payload, false)
})

test('无密钥连接兼容历史 null 并归一为空对象', () => {
  const payload = ConnectionSchema.parse({ ...connection, secretStatus: null })
  assert.deepEqual(payload.secretStatus, {})
})

test('连接分页必须使用唯一读模型', () => {
  const payload = ConnectionPageSchema.parse({
    list: [connection],
    pagination: { page: 1, pageSize: 10, total: 1, totalPages: 1 },
  })
  assert.equal(payload.list.length, 1)
  assert.equal(payload.pagination.total, 1)
})
