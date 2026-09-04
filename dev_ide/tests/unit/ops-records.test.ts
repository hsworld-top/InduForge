import { createApp, defineComponent, h, nextTick, reactive, type App } from 'vue'
import { beforeEach, afterEach, describe, it, expect, vi } from 'vitest'
const mocks = vi.hoisted(() => ({
  list: vi.fn(),
  events: vi.fn(),
  emit: vi.fn(),
  handlers: new Map<string, (...args: any[]) => void>(),
}))
vi.mock('@/api/ops.api', () => ({
  opsAPI: { listRecords: mocks.list, listDeploymentRunEventsPage: mocks.events },
}))
vi.mock('@/utils/socket', () => ({
  initSocket: () => ({
    connected: true,
    on: (event: string, handler: (...args: any[]) => void) => mocks.handlers.set(event, handler),
    off: (event: string) => mocks.handlers.delete(event),
    emit: mocks.emit,
  }),
}))
import OpsRecords from '@/views/tenant/components/OpsRecords.vue'
const row = (id = '1', extra = {}) => ({
  id,
  sourceKind: 'deployment_run',
  recordType: 'operation',
  objectType: 'deployment',
  objectId: 'd1',
  objectName: '演示工程',
  environmentId: 'e1',
  title: '启动',
  status: 'success',
  actorDisplayName: '管理员',
  time: '2026-09-03T00:00:00Z',
  completedAt: '2026-09-03T00:00:48Z',
  durationMs: 48000,
  message: '已完成',
  taskRef: { runId: 'r1', deploymentId: 'd1' },
  detailUnavailableReason: '',
  ...extra,
})
const flush = async () => {
  for (let i = 0; i < 20; i++) await Promise.resolve()
  await nextTick()
}
let app: App, host: HTMLElement
async function mount(environmentId = '') {
  const props = reactive({
    active: true,
    environmentId,
    objectId: '',
    objectType: '',
    objectName: '',
    environments: [{ id: 'e1', name: '默认环境' }],
  })
  host = document.createElement('div')
  document.body.appendChild(host)
  app = createApp(defineComponent({ setup: () => () => h(OpsRecords, props) }))
  app.config.warnHandler = () => {}
  app.mount(host)
  await flush()
  return { props, state: (app._instance!.subTree.component as any).setupState }
}
beforeEach(() => {
  vi.useFakeTimers()
  mocks.list.mockReset().mockResolvedValue({ items: [row()], total: 35 })
  mocks.events.mockReset().mockResolvedValue({ items: [], total: 0 })
  mocks.emit.mockClear()
  mocks.handlers.clear()
})
afterEach(() => {
  app?.unmount()
  host?.remove()
  vi.useRealTimers()
})
describe('统一运维记录实际SFC', () => {
  it('主工具栏与五组筛选分层，分页位于滚动区外，展开前不加载详情', async () => {
    await mount()
    const primary = host.querySelector('.records-primary')!
    expect(primary.querySelector('.records-search')).not.toBeNull()
    expect(primary.querySelector('.records-actions')).not.toBeNull()
    expect(primary.querySelector('.records-filters')).toBeNull()
    expect(host.querySelectorAll('.records-filters .records-filter')).toHaveLength(5)
    const scroll = host.querySelector('.records-scroll')!
    expect(scroll.querySelector('table[aria-label="运维记录"]')).not.toBeNull()
    expect(scroll.querySelector('.ck-pagination-bar')).toBeNull()
    expect(host.querySelector('.records-content > .ck-pagination-bar')).not.toBeNull()
    expect(mocks.events).not.toHaveBeenCalled()
    const expand = host.querySelector<HTMLButtonElement>('.record-expand')!
    expect(expand.getAttribute('aria-expanded')).toBe('false')
    expand.click()
    await flush()
    expect(expand.getAttribute('aria-expanded')).toBe('true')
    expect(host.querySelector('.record-detail-heading')?.textContent).toContain('执行明细')
    expect(mocks.events).toHaveBeenCalledTimes(1)
  })
  it('已打开记录页切到节点入口更新精确对象条件', async () => {
    const { props } = await mount('e1')
    props.environmentId = ''
    props.objectId = 'n1'
    props.objectType = 'node'
    props.objectName = '节点一'
    await flush()
    expect(mocks.list.mock.calls.at(-1)![0]).toMatchObject({ objectId: 'n1', objectType: 'node' })
    expect(mocks.list.mock.calls.at(-1)![0].environmentId).toBeUndefined()
    expect(host.textContent).toContain('节点一')
  })
  it('后端分页/类型对象时间状态筛选，无逐行预取事件', async () => {
    const { state } = await mount('e1')
    expect(mocks.list.mock.calls[0][0]).toMatchObject({ page: 1, limit: 10, environmentId: 'e1' })
    expect(mocks.events).not.toHaveBeenCalled()
    state.page = 2
    await flush()
    expect(mocks.list.mock.calls.at(-1)![0].page).toBe(2)
    state.filters.recordType = 'operation'
    state.filters.objectType = 'foundation'
    state.filters.status = 'accepted'
    state.range = ['2026-09-01T00:00:00+08:00', '2026-09-03T00:00:00+08:00']
    await flush()
    expect(mocks.list.mock.calls.at(-1)![0]).toMatchObject({
      page: 1,
      recordType: 'operation',
      objectType: 'foundation',
      status: 'accepted',
      from: '2026-09-01T00:00:00+08:00',
    })
    state.expandedId = '1'
    await flush()
    expect(mocks.events).toHaveBeenCalledTimes(1)
    state.expandedId = ''
    await flush()
    expect(mocks.events).toHaveBeenCalledTimes(1)
  })
  it('搜索切页取消旧请求，迟到结果不覆盖新条件', async () => {
    const { state } = await mount()
    let resolve!: (value: any) => void
    mocks.list.mockImplementationOnce(
      () =>
        new Promise((done) => {
          resolve = done
        }),
    )
    state.page = 2
    await flush()
    const signal = mocks.list.mock.calls.at(-1)![1]
    state.filters.search = '新的对象'
    await flush()
    expect(signal.aborted).toBe(true)
    mocks.list.mockResolvedValue({ items: [row('new')], total: 1 })
    resolve({ items: [row('late')], total: 35 })
    await flush()
    expect(state.rows[0].id).toBe('new')
    expect(state.page).toBe(1)
  })
  it('实时变更仅串行对账当前页，隐藏退订，恢复快照且失败保留行', async () => {
    const { props, state } = await mount()
    const subscriptionId = mocks.emit.mock.calls.find((call) => call[0] === 'ops:watch')![1]
      .subscriptionId
    mocks.handlers.get('ops:ready')?.({
      subscriptionId,
      epoch: 'e',
      sequence: 1,
      topics: ['deployments', 'events'],
    })
    await flush()
    state.page = 2
    await flush()
    mocks.handlers.get('ops:change')?.({
      epoch: 'e',
      sequence: 2,
      topics: ['events'],
      terminal: true,
    })
    await flush()
    expect(mocks.list.mock.calls.at(-1)![0].page).toBe(2)
    props.active = false
    await flush()
    const count = mocks.list.mock.calls.length
    expect(mocks.emit.mock.calls.some((call) => call[0] === 'ops:unwatch')).toBe(true)
    await vi.advanceTimersByTimeAsync(60000)
    expect(mocks.list).toHaveBeenCalledTimes(count)
    props.active = true
    await flush()
    expect(mocks.list.mock.calls.length).toBeGreaterThan(count)
    mocks.list.mockRejectedValueOnce(new Error('network'))
    await state.monitor.refresh()
    await flush()
    expect(state.rows).toHaveLength(1)
    expect(state.connection).toBe('stale')
    expect(host.querySelector('.records-notice')?.textContent).toContain('保留上次记录')
    expect(host.querySelector('.records-notice button')?.textContent).toBe('重试')
  })
  it('系统事件不冒充任务、已删除对象展示摘要原因且不请求events', async () => {
    mocks.list.mockResolvedValue({
      items: [
        row('event', {
          sourceKind: 'cluster_event',
          recordType: 'event',
          objectType: 'node',
          status: 'warning',
          taskRef: null,
          title: '节点离线',
        }),
        row('deleted', { taskRef: null, detailUnavailableReason: '对象已删除，保留操作摘要' }),
      ],
      total: 2,
    })
    const { state } = await mount()
    expect(host.textContent).toContain('系统事件')
    expect(host.textContent).toContain('异常')
    state.expandedId = 'deleted'
    await flush()
    expect(host.textContent).toContain('对象已删除，保留操作摘要')
    expect(mocks.events).not.toHaveBeenCalled()
  })
})
