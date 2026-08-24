import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import type { AlarmItem } from '../src/api/schemas/alarm.schema'

const item: AlarmItem = {
  id: 'alarm-1',
  projectId: 'project',
  datapointId: 'dp-1',
  path: 'line.temperature',
  datapointName: '温度',
  dataType: 'float64',
  groupId: null,
  groupName: null,
  displayName: '越限报警',
  description: null,
  mode: 'point',
  alarmType: 'threshold',
  evaluationMode: 'single',
  derivedExpression: '',
  inputs: [],
  conditions: [
    {
      id: 'c',
      kind: 'threshold',
      operator: 'gt',
      label: '高',
      severity: 'warning',
      params: { threshold: 80 },
      triggerDelayMs: 0,
      clearDelayMs: 0,
      deadband: 0,
    },
  ],
  notification: { mode: 'inherit', channelIds: [], messageTemplate: '' },
  isEnabled: true,
  revision: 1,
  contract: {},
  createdAt: '2026-08-24T00:00:00Z',
  updatedAt: '2026-08-24T00:00:00Z',
}
const mocks = vi.hoisted(() => ({
  listAlarmItems: vi.fn(),
  getAlarmItem: vi.fn(),
  createAlarmItem: vi.fn(),
  updateAlarmItem: vi.fn(),
  setAlarmItemEnabled: vi.fn(),
  deleteAlarmItem: vi.fn(),
}))
vi.mock('../src/api/alarm.api', () => ({ ...mocks, listAlarmGroupTree: vi.fn(async () => []) }))
import { useAlarmStore } from '../src/stores/alarm.store'

describe('alarm store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
    mocks.listAlarmItems.mockResolvedValue({
      list: [item],
      pagination: { page: 1, pageSize: 20, total: 1 },
    })
    mocks.getAlarmItem.mockResolvedValue(item)
    mocks.createAlarmItem.mockResolvedValue({ ...item, id: 'alarm-2' })
    mocks.updateAlarmItem.mockResolvedValue({ ...item, displayName: '已修改' })
    mocks.setAlarmItemEnabled.mockResolvedValue({ ...item, isEnabled: false })
    mocks.deleteAlarmItem.mockResolvedValue(undefined)
  })
  it('loads, toggles and removes independent alarm items', async () => {
    const store = useAlarmStore()
    await store.fetchList('project')
    expect(store.list).toHaveLength(1)
    await store.toggle('project', 'alarm-1', false)
    expect(store.list[0]?.isEnabled).toBe(false)
    await store.remove('project', 'alarm-1')
    expect(store.list).toHaveLength(0)
  })
  it('saves one item payload', async () => {
    const store = useAlarmStore()
    await store.save('project', {
      datapointId: 'dp-1',
      displayName: '',
      mode: 'point',
      evaluationMode: 'single',
      inputs: [],
      derivedExpression: '',
      conditions: item.conditions,
      notification: item.notification,
      isEnabled: true,
      revision: 0,
      acknowledgedWarningKeys: [],
    })
    expect(mocks.createAlarmItem).toHaveBeenCalledOnce()
  })
})
