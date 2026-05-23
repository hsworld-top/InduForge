import { beforeEach, describe, expect, test, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import type { AlarmPolicy } from '../src/api/schemas/alarm.schema'

const alarmPolicy: AlarmPolicy = {
  id: 'policy-1',
  projectId: 'project-1',
  groupId: null,
  name: '温度策略',
  description: '',
  mode: 'per_target',
  targets: [{ datapointId: 'dp-1', path: 'metrics.temperature', dataType: 'number' }],
  inputs: [],
  derivedExpression: '',
  conditions: [
    {
      id: 'c-h',
      type: 'H',
      name: '高限',
      isEnabled: true,
      severity: 'major',
      params: { limit: 80 },
    },
  ],
  isEnabled: true,
  effectiveEnabled: true,
  suppression: { enabled: false },
  messageTemplate: '',
  contract: { version: 1 },
  createdAt: '',
  updatedAt: '',
}

const getAlarmPolicyGroups = vi.fn(async () => [])
const getAlarmPolicyTree = vi.fn(async () => ({
  groups: [],
  rootPolicies: [alarmPolicy],
  policies: [alarmPolicy],
  matchedPolicyCount: 1,
  totalPolicyCount: 1,
}))
const getAlarmPolicies = vi.fn(async () => ({
  list: [alarmPolicy],
  pagination: { total: 1 },
}))
const getAlarmPolicy = vi.fn(async () => alarmPolicy)
const createAlarmPolicy = vi.fn(async () => ({
  ...alarmPolicy,
  id: 'policy-2',
}))
const updateAlarmPolicy = vi.fn(async () => ({
  ...alarmPolicy,
  name: '温度策略2',
}))
const deleteAlarmPolicy = vi.fn(async () => undefined)
const toggleAlarmPolicy = vi.fn(async () => ({
  ...alarmPolicy,
  isEnabled: false,
}))
const testAlarmPolicy = vi.fn(async () => ({
  triggered: true,
  state: 'triggered',
  triggeredConditions: alarmPolicy.conditions,
  diagnostics: {},
  conditionResults: [],
}))
const getAlarmPolicyContract = vi.fn(async () => ({
  version: 1,
  policyId: 'policy-1',
}))
const validateAlarmPolicyDraft = vi.fn(async () => ({
  valid: true,
  errors: [],
  contract: { version: 1 },
}))
const batchEnableAlarmPolicies = vi.fn(async () => undefined)
const batchDisableAlarmPolicies = vi.fn(async () => undefined)
const batchMoveAlarmPolicies = vi.fn(async () => undefined)
const batchApplyAlarmConditions = vi.fn(async () => undefined)

vi.mock('@/api/alarm.api', () => ({
  getAlarmPolicyGroups,
  getAlarmPolicyTree,
  getAlarmPolicies,
  getAlarmPolicy,
  createAlarmPolicy,
  updateAlarmPolicy,
  deleteAlarmPolicy,
  toggleAlarmPolicy,
  testAlarmPolicy,
  getAlarmPolicyContract,
  validateAlarmPolicyDraft,
  batchEnableAlarmPolicies,
  batchDisableAlarmPolicies,
  batchMoveAlarmPolicies,
  batchApplyAlarmConditions,
}))

describe('alarm store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  test('拉取策略树并保留根目录策略', async () => {
    const { useAlarmStore } = await import('../src/stores/alarm.store')
    const store = useAlarmStore()

    await store.fetchTree('project-1', { search: '温度' })

    expect(getAlarmPolicyTree).toHaveBeenCalledWith('project-1', {
      search: '温度',
    })
    expect(store.tree.rootPolicies[0].id).toBe('policy-1')
    expect(store.total).toBe(1)
  })

  test('筛选后全选使用筛选快照', async () => {
    const { useAlarmStore } = await import('../src/stores/alarm.store')
    const store = useAlarmStore()

    store.selectFiltered({ search: '温度' })
    store.selectPolicy('policy-2', false)

    expect(store.selection.mode).toBe('filtered')
    if (store.selection.mode === 'filtered') {
      expect(store.selection.excludePolicyIds).toContain('policy-2')
    }
  })

  test('打开详情并写入 editing', async () => {
    const { useAlarmStore } = await import('../src/stores/alarm.store')
    const store = useAlarmStore()

    const detail = await store.openPolicy('project-1', 'policy-1')

    expect(getAlarmPolicy).toHaveBeenCalledWith('project-1', 'policy-1')
    expect(detail.id).toBe('policy-1')
    expect(store.editing?.id).toBe('policy-1')
  })

  test('创建策略后插入列表并设为当前编辑', async () => {
    const { useAlarmStore } = await import('../src/stores/alarm.store')
    const store = useAlarmStore()

    const created = await store.createPolicy('project-1', {
      name: '温度策略',
      mode: 'per_target',
      targets: alarmPolicy.targets,
      inputs: [],
      derivedExpression: '',
      conditions: alarmPolicy.conditions,
      isEnabled: true,
      suppression: { enabled: false },
      messageTemplate: '',
    })

    expect(createAlarmPolicy).toHaveBeenCalled()
    expect(created.id).toBe('policy-2')
    expect(store.list[0].id).toBe('policy-2')
    expect(store.editing?.id).toBe('policy-2')
  })

  test('试算、契约和批量操作写入独立状态', async () => {
    const { useAlarmStore } = await import('../src/stores/alarm.store')
    const store = useAlarmStore()
    store.selectFiltered({ search: '温度' })

    await store.runTrial('project-1', 'policy-1', { value: 90 })
    await store.fetchContract('project-1', 'policy-1')
    await store.batchEnable('project-1')

    expect(testAlarmPolicy).toHaveBeenCalled()
    expect(store.trial.result?.state).toBe('triggered')
    expect(store.contract.data?.policyId).toBe('policy-1')
    expect(batchEnableAlarmPolicies).toHaveBeenCalledWith('project-1', store.selection)
  })
})
