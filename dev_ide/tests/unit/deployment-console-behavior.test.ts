import { createApp, defineComponent, h, nextTick, ref, type App } from 'vue'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
const mocks = vi.hoisted(() => ({
  list: vi.fn(),
  detail: vi.fn(),
  action: vi.fn(),
  create: vi.fn(),
  confirm: vi.fn(),
  success: vi.fn(),
  error: vi.fn(),
  envDetail: vi.fn(),
  overview: vi.fn(),
  envNodes: vi.fn(),
  envList: vi.fn(),
  isOpsAdmin: false,
  socketEmit: vi.fn(),
  socketHandlers: new Map<string, (...args: any[]) => void>(),
}))
vi.mock('@/utils/socket', () => ({
  initSocket: () => ({
    connected: true,
    on: (name: string, handler: (...args: any[]) => void) =>
      mocks.socketHandlers.set(name, handler),
    off: (name: string) => mocks.socketHandlers.delete(name),
    emit: mocks.socketEmit,
  }),
}))
vi.mock('@/api/ops.api', () => ({
  opsAPI: {
    listProjectDeployments: mocks.list,
    getProjectDeployment: mocks.detail,
    operateProjectDeployment: mocks.action,
    createProjectDeployment: mocks.create,
    getRuntimeEnvironment: mocks.envDetail,
    getRuntimeEnvironmentOverview: mocks.overview,
    listRuntimeEnvironmentNodes: mocks.envNodes,
    listRuntimeEnvironmentServices: vi.fn().mockResolvedValue({ items: [], total: 0 }),
    listRuntimeEnvironmentEvents: vi.fn().mockResolvedValue({ items: [], total: 0 }),
    listRecords: vi.fn().mockResolvedValue({ items: [], total: 0 }),
    listDeploymentRunEvents: vi.fn().mockResolvedValue({ items: [], total: 0 }),
    listProjectDeploymentRuns: vi.fn().mockResolvedValue({ items: [], total: 0 }),
    listDeploymentRunEventsPage: vi.fn().mockResolvedValue({ items: [], total: 0 }),
    getDeploymentRun: vi.fn().mockResolvedValue({ id: 'r1' }),
    listRuntimeEnvironments: mocks.envList,
    listNodes: vi.fn().mockResolvedValue({ items: [], total: 0 }),
    listEnrollments: vi.fn().mockResolvedValue({ items: [], total: 0 }),
    listNodePackages: vi.fn().mockResolvedValue({ items: [], total: 0 }),
  },
}))
vi.mock('@/store', () => ({ useAuthStore: () => ({ userInfo: { role: 'PROJECT_ADMIN' } }) }))
vi.mock('@/permissions', () => ({
  can: (_role: string, capability: string) =>
    capability === 'deploy:execute' || (capability === 'runtime:operate' && mocks.isOpsAdmin),
}))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key, locale: ref('zh-CN') }) }))
vi.mock('element-plus', () => ({
  ElMessage: { success: mocks.success, error: mocks.error, warning: vi.fn() },
  ElMessageBox: { confirm: mocks.confirm },
}))
import OpsManagementConsole from '@/views/tenant/OpsManagementConsole.vue'

let app: App | undefined
let host: HTMLElement
const overviewData = () => ({
  environmentId: 'e1',
  snapshotAt: '2026-09-03T00:00:00Z',
  thresholds: { freshnessSeconds: 45, capacityAttentionPercent: 85, capacityCriticalPercent: 95 },
  deployments: { total: 1, running: 0, stopped: 1, failed: 0, pending: 0, stale: 0, unknown: 0 },
  nodes: {
    total: 1,
    online: 1,
    offline: 0,
    fault: 0,
    staleMetrics: 0,
    unknownMetrics: 0,
    capacityAttention: 0,
    capacityCritical: 0,
    maxDiskUsagePercent: null,
  },
  riskNodes: [],
})
const item = (id = 'd1') => ({
  id,
  projectId: 'p1',
  projectName: '工程',
  environmentId: 'e1',
  mode: 'development',
  version: '__DEV__',
  accessPort: 17800,
  observedStatus: 'running',
  desiredStatus: 'running',
  entryStatus: 'running',
})
const flush = async () => {
  for (let i = 0; i < 15; i++) await Promise.resolve()
  await nextTick()
}
async function mount() {
  host = document.createElement('div')
  document.body.appendChild(host)
  const active = ref(true)
  app = createApp(
    defineComponent({ setup: () => () => h(OpsManagementConsole, { isActive: active.value }) }),
  )
  app.config.globalProperties.$t = (key: string, params?: unknown) =>
    key === 'opsConsole.environments.runningCount'
      ? `${(params as { count?: number })?.count}个运行中`
      : key
  app.component('ElTableColumn', defineComponent({ setup: () => () => null }))
  app.config.warnHandler = () => {}
  app.mount(host)
  await flush()
  // 访问真实SFC setup状态，测试请求与动作行为，不靠源码字符串推断。
  const state = (app._instance!.subTree.component as any).setupState
  return { state, active }
}
beforeEach(() => {
  mocks.envNodes.mockReset().mockResolvedValue({ items: [], total: 0 })
  mocks.overview.mockReset().mockResolvedValue(overviewData())
  mocks.isOpsAdmin = false
  vi.useFakeTimers()
  mocks.list.mockReset().mockResolvedValue({ items: [item()], total: 1 })
  mocks.detail.mockReset().mockResolvedValue(item())
  mocks.action
    .mockReset()
    .mockResolvedValue({ deployment: { ...item(), observedStatus: 'pending' }, run: { id: 'r1' } })
  mocks.confirm.mockReset().mockResolvedValue('confirm')
  mocks.create.mockReset().mockResolvedValue(item())
  mocks.success.mockReset()
  mocks.error.mockReset()
  mocks.envDetail.mockReset().mockResolvedValue({ id: 'e1', name: '环境一', status: 'ready' })
  mocks.envList.mockReset().mockResolvedValue({ items: [], total: 0 })
  mocks.socketEmit.mockClear()
})
afterEach(() => {
  app?.unmount()
  app = undefined
  host?.remove()
  vi.useRealTimers()
})

describe('工程部署真实页面交互', () => {
  it('首次从概览进入基础服务按需取得节点依赖，无需先访问节点页', async () => {
    mocks.isOpsAdmin = true
    const environment = {
      id: 'e1',
      name: '环境一',
      status: 'available',
      nodeCount: 1,
      onlineNodeCount: 1,
      foundationTotal: 8,
      foundationHealthy: 8,
      projectCount: 1,
      runningDeploymentCount: 1,
    }
    mocks.envList.mockResolvedValue({ items: [environment], total: 1 })
    mocks.envDetail.mockResolvedValue(environment)
    mocks.envNodes.mockResolvedValue({
      items: [
        {
          id: 'n1',
          name: '节点一',
          capabilities: [],
          observedStatus: 'online',
          clusterStatus: 'ready',
        },
      ],
      total: 1,
    })
    const { state } = await mount()
    expect(state.environmentDetailTab).toBe('overview')
    expect(state.environmentNodes).toHaveLength(0)
    state.environmentDetailTab = 'services'
    await flush()
    expect(mocks.envNodes).toHaveBeenCalledWith(
      'e1',
      { page: 1, pageSize: 10 },
      expect.any(AbortSignal),
    )
    expect(state.environmentNodes).toHaveLength(1)
    expect(state.foundationDeployDisabled).toBe(false)
    state.environmentNodes[0].clusterStatus = 'failed'
    await nextTick()
    expect(state.foundationDeployDisabled).toBe(true)
  })
  it('磁盘保留两位不跨容量阈值，风险仍只信任后端，CPU与内存保持整数', async () => {
    const { state } = await mount()
    const node = {
      observedStatus: 'online',
      metricsStale: false,
      diskPercent: 84.65,
      cpuPercent: 42.65,
      memoryPercent: 31.65,
      health: 'healthy',
    }
    expect(state.overviewMetric(node, 'diskPercent')).toBe(84.65)
    expect(state.overviewNodeHealth(node)).toBe('在线')
    expect(state.overviewMetric(node, 'cpuPercent')).toBe(43)
    expect(state.overviewMetric(node, 'memoryPercent')).toBe(32)
    expect(
      state.overviewMetric(
        { ...node, diskPercent: 85, health: 'capacity_attention' },
        'diskPercent',
      ),
    ).toBe(85)
    expect(
      state.overviewNodeHealth({ ...node, diskPercent: 85, health: 'capacity_attention' }),
    ).toBe('容量关注')
    expect(state.overviewDiskPercent(84.999)).toBe(84.99)
    expect(state.overviewDiskPercent(86.65)).toBe(86.65)
  })
  it('概览容量汇总与风险节点来自全局快照，空指标不显示零且不改用户节点名', async () => {
    mocks.isOpsAdmin = true
    const environment = {
      id: 'e1',
      name: '环境一',
      status: 'available',
      nodeCount: 12,
      onlineNodeCount: 12,
      foundationTotal: 8,
      foundationHealthy: 8,
      projectCount: 1,
      runningDeploymentCount: 1,
    }
    mocks.envList.mockResolvedValue({ items: [environment], total: 1 })
    mocks.envDetail.mockResolvedValue(environment)
    const snapshot = {
      ...overviewData(),
      nodes: {
        ...overviewData().nodes,
        total: 12,
        online: 12,
        capacityAttention: 1,
        maxDiskUsagePercent: 87,
      },
      riskNodes: [
        {
          id: 'n1',
          name: 'nginx-node',
          observedStatus: 'online',
          clusterStatus: 'ready',
          metricsStale: false,
          cpuPercent: null,
          memoryPercent: 42,
          diskPercent: 87,
          health: 'capacity_attention',
        },
      ],
    }
    mocks.overview.mockResolvedValue(snapshot)
    const { state } = await mount()
    expect(host.textContent).toContain('容量提醒')
    expect(host.textContent).toContain('最高 87%')
    expect(host.textContent).not.toContain('运行故障')
    expect(host.textContent).toContain('nginx-node')
    expect(host.querySelector('.overview-node-table')?.textContent).toContain('—')
    state.environmentNodes = [
      {
        id: 'unrelated',
        name: '页内普通节点',
        observedStatus: 'online',
        resourceSummary: { disk: { usedPercent: 1 } },
      },
    ]
    await nextTick()
    expect(host.textContent).toContain('最高 87%')
    state.openOverviewNodes()
    await flush()
    expect(state.environmentDetailTab).toBe('nodes')
    expect(host.querySelector('.ops-toolbar-summaries')?.textContent).toContain('12/12 在线')
  })
  it('概览故障/过期独立呈现，后台失败保留最近数据且切环境清空', async () => {
    mocks.isOpsAdmin = true
    const environment = {
      id: 'e1',
      name: '环境一',
      status: 'attention',
      nodeCount: 1,
      onlineNodeCount: 0,
      foundationTotal: 0,
      foundationHealthy: 0,
      projectCount: 1,
      runningDeploymentCount: 0,
    }
    mocks.envList.mockResolvedValue({ items: [environment], total: 1 })
    mocks.envDetail.mockResolvedValue(environment)
    mocks.overview.mockResolvedValue({
      ...overviewData(),
      deployments: { ...overviewData().deployments, failed: 1, stopped: 0 },
      nodes: { ...overviewData().nodes, offline: 1, online: 0, staleMetrics: 1 },
      riskNodes: [
        {
          id: 'n1',
          name: '旧节点',
          observedStatus: 'offline',
          metricsStale: true,
          cpuPercent: 70,
          memoryPercent: 70,
          diskPercent: 99,
          health: 'offline',
        },
      ],
    })
    const { state } = await mount()
    expect(host.textContent).toContain('运行故障')
    expect(host.textContent).toContain('指标过期')
    expect(host.querySelector('.overview-node-table')?.textContent).not.toContain('99%')
    mocks.overview.mockRejectedValue(new Error('offline'))
    await state.realtime.refresh()
    await flush()
    expect(state.overviewError).toBe(true)
    expect(host.textContent).toContain('概览更新中断')
    expect(state.overview.nodes.offline).toBe(1)
    state.selectedEnvironmentId = 'e2'
    await nextTick()
    expect(state.overview).toBeNull()
  })
  it('环境查看全部和节点记录入口携带明确对象条件，不保留无关环境筛选', async () => {
    const { state } = await mount()
    state.selectedEnvironmentId = 'e1'
    state.openEnvironmentRecords()
    await flush()
    expect(state.activeTab).toBe('records')
    expect(state.recordEnvironmentId).toBe('e1')
    state.openNodeRecords({ id: 'n1', displayName: '节点一' })
    await flush()
    expect(state.recordEnvironmentId).toBe('')
    expect(state.recordObject).toMatchObject({ id: 'n1', type: 'node', name: '节点一' })
  })
  it('概览独立使用服务端运行数，总部署1在停止时显示0运行、运行时显示1运行', async () => {
    mocks.isOpsAdmin = true
    const environment = {
      id: 'e1',
      name: '环境一',
      status: 'available',
      nodeCount: 1,
      onlineNodeCount: 1,
      foundationTotal: 8,
      foundationHealthy: 8,
      projectCount: 1,
      runningDeploymentCount: 0,
    }
    mocks.envList.mockResolvedValue({ items: [environment], total: 1 })
    mocks.envDetail.mockResolvedValue(environment)
    const { state } = await mount()
    expect(state.selectedEnvironment.projectCount).toBe(1)
    expect(state.selectedEnvironment.runningDeploymentCount).toBe(0)
    expect(host.querySelector('.ops-toolbar-summaries')?.textContent).toContain('0/1 运行中')
    mocks.envDetail.mockResolvedValue({ ...environment, runningDeploymentCount: 1 })
    mocks.overview.mockResolvedValue({
      ...overviewData(),
      deployments: { ...overviewData().deployments, running: 1, stopped: 0 },
    })
    await state.realtime.refresh()
    await nextTick()
    expect(state.selectedEnvironment.projectCount).toBe(1)
    expect(host.querySelector('.ops-toolbar-summaries')?.textContent).toContain('1/1 运行中')
  })
  it('基础服务只信任中心权威freshness，过期不显示绿色且不计入组健康数', async () => {
    const { state } = await mount()
    state.foundationServices = [
      {
        serviceType: 'if_realtime',
        observedStatus: 'running',
        observedStale: true,
        observedAt: '2099-01-01T00:00:00Z',
      },
    ]
    const row = () => state.foundationRows.find((service: any) => service.type === 'if_realtime')
    expect(row()).toMatchObject({ state: 'attention', label: '观测已过期' })
    expect(state.foundationServiceGroups[0].label).toMatch(/^0 \/ /)
    // 浏览器日期无论领先还是落后，都不反转数据库权威的过期结论。
    vi.setSystemTime(new Date('2000-01-01T00:00:00Z'))
    expect(row().state).toBe('attention')
    state.foundationServices = [
      {
        serviceType: 'if_realtime',
        observedStatus: 'running',
        observedStale: false,
        observedAt: '1999-01-01T00:00:00Z',
      },
    ]
    expect(row().state).toBe('available')
    expect(state.foundationServiceGroups[0].label).toMatch(/^1 \/ /)
  })
  it('首加载失败保留skeleton，不把未加载数据显示成空环境', async () => {
    mocks.list.mockRejectedValueOnce(new Error('offline'))
    const { state } = await mount()
    expect(state.initialSnapshotComplete).toBe(false)
    expect(host.querySelector('[aria-busy="true"]')).not.toBeNull()
    await state.loadDeployments()
    expect(state.initialSnapshotComplete).toBe(true)
  })
  it('其他环境事件不改变当前详情，已知节点指标仍刷新当前环境', async () => {
    const { state } = await mount()
    state.environmentOptions = [{ id: 'e1', name: '环境一' }]
    state.selectedEnvironmentId = 'e1'
    state.activeTab = 'environments'
    await flush()
    state.environmentNodes = [{ id: 'n1', name: '节点一' }]
    const subscriptionId = mocks.socketEmit.mock.calls
      .filter((call) => call[0] === 'ops:watch')
      .at(-1)![1].subscriptionId
    mocks.socketHandlers.get('ops:ready')?.({
      subscriptionId,
      epoch: 'a',
      sequence: 1,
      topics: ['nodes', 'environments', 'deployments', 'events'],
    })
    await flush()
    state.environmentNodes = [{ id: 'n1', name: '节点一' }]
    mocks.envDetail.mockClear()
    mocks.socketHandlers.get('ops:change')?.({
      epoch: 'a',
      sequence: 2,
      topics: ['environments'],
      entityIds: ['e2'],
    })
    await vi.advanceTimersByTimeAsync(1200)
    expect(state.environmentManagementMode).toBe(false)
    expect(state.selectedEnvironmentId).toBe('e1')
    expect(mocks.envDetail).not.toHaveBeenCalled()
    mocks.socketHandlers.get('ops:change')?.({
      epoch: 'a',
      sequence: 3,
      topics: ['nodes'],
      entityIds: ['n1'],
    })
    await vi.advanceTimersByTimeAsync(1200)
    expect(mocks.envDetail).toHaveBeenCalledWith('e1', expect.any(AbortSignal))
  })
  it('任务提交后loading贯穿pending，其他操作者后续任务终态也释放旧锁', async () => {
    const { state } = await mount()
    const pending = { ...item(), observedStatus: 'pending', latestRunId: 'r1' }
    mocks.list.mockResolvedValue({ items: [pending], total: 1 })
    await state.operateDeploymentRow(state.deploymentRows[0], 'restart')
    expect(state.deploymentOperations.d1).toBe('restart')
    mocks.list.mockResolvedValue({
      items: [{ ...item(), latestRunId: 'another-user-run' }],
      total: 1,
    })
    state.deploymentPage.page = 2
    await state.loadDeployments()
    expect(state.deploymentOperations.d1).toBeUndefined()
  })
  it('跨模式更新先查询目标唯一部署并确认，取消不提交且确认期间固定原请求', async () => {
    const { state } = await mount()
    state.projects = [{ id: 'p1', name: '工程' }]
    state.versions = [{ id: 'v1', version: '1.0.0', status: 'ready' }]
    Object.assign(state.deployForm, {
      projectId: 'p1',
      environmentId: 'e1',
      mode: 'production',
      applicationVersionId: 'v1',
    })
    state.deployForm.placements.base = 'n1'
    mocks.confirm.mockRejectedValueOnce('cancel')
    await state.createDeployment()
    expect(mocks.list).toHaveBeenLastCalledWith({
      projectId: 'p1',
      environmentId: 'e1',
      page: 1,
      pageSize: 1,
    })
    expect(mocks.confirm.mock.calls[0][0]).toContain('切换部署模式将替换当前部署')
    expect(mocks.create).not.toHaveBeenCalled()
    let confirm!: () => void
    mocks.confirm.mockImplementationOnce(
      () =>
        new Promise<void>((resolve) => {
          confirm = resolve
        }),
    )
    const pending = state.createDeployment()
    await flush()
    state.deployForm.environmentId = 'e2'
    confirm()
    await pending
    expect(mocks.create).toHaveBeenCalledExactlyOnceWith(
      expect.objectContaining({
        projectId: 'p1',
        environmentId: 'e1',
        mode: 'production',
        applicationVersionId: 'v1',
      }),
    )
  })
  it('同部署确认期间锁定且重新部署只调用正式action，另一个部署不被锁定', async () => {
    const { state } = await mount()
    let confirm!: () => void
    mocks.confirm.mockImplementationOnce(
      () =>
        new Promise<void>((resolve) => {
          confirm = resolve
        }),
    )
    const row = state.deploymentRows[0]
    const pending = state.operateDeploymentRow(row, 'redeploy')
    expect(state.canRunAction(row, 'restart')).toBe(false)
    expect(state.canRunAction({ ...row, id: 'd2', deployment: item('d2') }, 'restart')).toBe(true)
    await state.operateDeploymentRow(row, 'redeploy')
    expect(mocks.confirm).toHaveBeenCalledTimes(1)
    expect(mocks.action).not.toHaveBeenCalled()
    confirm()
    await pending
    expect(mocks.action).toHaveBeenCalledExactlyOnceWith('d1', 'redeploy')
    expect(mocks.success.mock.calls[0][0]).toContain('已提交')
  })
  it('旧分页迟到不能覆盖新查询，抽屉按ID刷新且不依赖当前页', async () => {
    const { state } = await mount()
    state.openDeploymentRow(state.deploymentRows[0])
    await flush()
    let resolve!: (value: unknown) => void
    mocks.list.mockImplementationOnce(
      () =>
        new Promise((done) => {
          resolve = done
        }),
    )
    const request = state.loadDeployments()
    await flush()
    state.deploymentPage.page = 2
    mocks.list.mockResolvedValue({ items: [item('page2')], total: 20 })
    void state.loadDeployments()
    resolve({ items: [item('stale')], total: 999 })
    await request
    await flush()
    expect(state.deployments[0].id).toBe('page2')
    expect(state.deploymentPage.total).toBe(20)
    expect(state.selectedDeploymentRow.id).toBe('d1')
    expect(mocks.detail).toHaveBeenCalledWith('d1', expect.any(AbortSignal))
  })
  it('IDE切走暂停，恢复立即刷新，静默失败保留数据并显示过期，恢复清除', async () => {
    const { state, active } = await mount()
    active.value = false
    await nextTick()
    const before = mocks.list.mock.calls.length
    await vi.advanceTimersByTimeAsync(10000)
    expect(mocks.list).toHaveBeenCalledTimes(before)
    mocks.list.mockRejectedValueOnce(new Error('offline'))
    active.value = true
    await flush()
    expect(state.deploymentStale).toBe(true)
    expect(state.deployments[0].id).toBe('d1')
    expect(mocks.error).not.toHaveBeenCalled()
    await vi.advanceTimersByTimeAsync(20000)
    await flush()
    expect(state.deploymentStale).toBe(false)
  })
})
