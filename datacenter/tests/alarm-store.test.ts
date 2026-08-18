import { beforeEach, describe, expect, test, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import type { AlarmPolicy } from '../src/api/schemas/alarm.schema'

const policy: AlarmPolicy = {
  id: 'policy-1',
  projectId: 'project-1',
  groupId: null,
  groupName: null,
  name: '温度报警',
  description: null,
  mode: 'per_target',
  derivedExpression: '',
  bindings: [],
  conditions: [],
  notification: { mode: 'inherit', channelIds: [], messageTemplate: '' },
  isEnabled: true,
  revision: 1,
  contract: {},
  createdAt: null,
  updatedAt: null,
}
const listAlarmPolicies = vi.fn(async () => ({
  list: [policy],
  pagination: { page: 1, pageSize: 20, total: 1 },
}))
const listAlarmGroupTree = vi.fn(async () => [])
const getAlarmPolicy = vi.fn(async () => policy)
const createAlarmPolicy = vi.fn(async () => ({ ...policy, id: 'policy-2' }))
const updateAlarmPolicy = vi.fn(async () => ({ ...policy, name: '已修改' }))
const setAlarmPolicyEnabled = vi.fn(async () => ({ ...policy, isEnabled: false }))
const deleteAlarmPolicy = vi.fn(async () => undefined)

vi.mock('@/api/alarm.api', () => ({
  listAlarmPolicies,
  listAlarmGroupTree,
  getAlarmPolicy,
  createAlarmPolicy,
  updateAlarmPolicy,
  setAlarmPolicyEnabled,
  deleteAlarmPolicy,
}))

describe('alarm store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  test('分页拉取策略并保留总数', async () => {
    const { useAlarmStore } = await import('../src/stores/alarm.store')
    const store = useAlarmStore()
    await store.fetchList('project-1', { page: 1, search: '温度' })
    expect(store.list[0].id).toBe('policy-1')
    expect(store.total).toBe(1)
  })

  test('读取详情和创建策略', async () => {
    const { useAlarmStore } = await import('../src/stores/alarm.store')
    const store = useAlarmStore()
    await store.fetchDetail('project-1', 'policy-1')
    expect(store.editing?.id).toBe('policy-1')
    const created = await store.save('project-1', {
      name: '温度报警',
      mode: 'per_target',
      bindings: [],
      derivedExpression: '',
      conditions: [],
      notification: { mode: 'inherit', channelIds: [], messageTemplate: '' },
      isEnabled: true,
    })
    expect(created.id).toBe('policy-2')
  })
})
