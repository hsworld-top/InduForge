import { createApp, defineComponent, h, nextTick, reactive, type App } from 'vue'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
const mocks = vi.hoisted(() => ({ runs: vi.fn(), events: vi.fn(), run: vi.fn() }))
vi.mock('@/api/ops.api', () => ({
  opsAPI: {
    listProjectDeploymentRuns: mocks.runs,
    listDeploymentRunEventsPage: mocks.events,
    getDeploymentRun: mocks.run,
  },
}))
import DeploymentRunHistory from '@/views/tenant/components/DeploymentRunHistory.vue'
import {
  deploymentEnginePresentation,
  runDurationLabel,
  runOperationLabel,
  runResultLabel,
} from '@/views/tenant/utils/deployment-details'
const run = (id = 'r1') => ({
  id,
  operation: 'stop',
  observedStatus: 'stopped',
  actorDisplayName: '操作员',
  startedAt: '2026-09-03T00:00:00Z',
  completedAt: '2026-09-03T00:00:48Z',
  durationMs: 48000,
})
const flush = async () => {
  for (let i = 0; i < 15; i++) await Promise.resolve()
  await nextTick()
}
let app: App, host: HTMLElement
async function mount() {
  const props = reactive({ deploymentId: 'd1', latestRunId: 'r1', revision: 0, active: true })
  host = document.createElement('div')
  document.body.appendChild(host)
  app = createApp(defineComponent({ setup: () => () => h(DeploymentRunHistory, props) }))
  app.config.warnHandler = () => {}
  app.mount(host)
  await flush()
  return { props, state: (app._instance!.subTree.component as any).setupState }
}
beforeEach(() => {
  mocks.runs.mockReset().mockResolvedValue({ items: [run()], total: 35 })
  mocks.events.mockReset().mockResolvedValue({
    items: [
      {
        id: 'ev1',
        stage: 'observed',
        message: 'base：工程引擎已就绪',
        createdAt: '2026-09-03T00:00:00Z',
      },
    ],
    total: 71,
  })
  mocks.run.mockReset().mockResolvedValue(run())
})
afterEach(() => {
  app?.unmount()
  host?.remove()
})
describe('部署历史真实 SFC', () => {
  it('共享事件展开组件在慢请求期间合并实时刷新，终态后停止重复加载', async () => {
    mocks.runs.mockResolvedValue({ items: [{ ...run(), completedAt: undefined }], total: 1 })
    const { state, props } = await mount()
    let resolve!: (value: any) => void
    mocks.events.mockImplementationOnce(
      () =>
        new Promise((done) => {
          resolve = done
        }),
    )
    state.expand('r1')
    await flush()
    const signal = mocks.events.mock.calls[0][3]
    for (let i = 0; i < 10; i++) {
      props.revision++
      await flush()
    }
    expect(mocks.events).toHaveBeenCalledTimes(1)
    expect(signal.aborted).toBe(false)
    resolve({ items: [], total: 0 })
    await flush()
    expect(mocks.events).toHaveBeenCalledTimes(2)
    mocks.runs.mockResolvedValue({ items: [run()], total: 1 })
    props.revision++
    await flush()
    const completedCount = mocks.events.mock.calls.length
    props.revision++
    await flush()
    expect(mocks.events).toHaveBeenCalledTimes(completedCount)
  })
  it('按需加载真实分页，停止摘要使用实际操作和服务端耗时', async () => {
    const { state } = await mount()
    expect(host.textContent).toContain('最近操作 · 停止 · 已完成 · 48秒')
    expect(mocks.events).not.toHaveBeenCalled()
    state.expand('r1')
    await flush()
    expect(mocks.events).toHaveBeenLastCalledWith('r1', 1, 10, expect.any(AbortSignal))
    expect(host.textContent).toContain('状态上报')
    ;(
      host.querySelector('.run-events .pagination-nav-btn:not([disabled])') as HTMLButtonElement
    ).click()
    await flush()
    expect(mocks.events).toHaveBeenLastCalledWith('r1', 2, 10, expect.any(AbortSignal))
    state.page = 2
    await flush()
    expect(mocks.runs).toHaveBeenLastCalledWith('d1', 2, 10, expect.any(AbortSignal))
    expect(state.expandedId).toBe('')
    expect(host.textContent).toContain('最近操作 · 停止')
  })
  it('切部署和收起取消旧事件，迟到内容不会串入新任务', async () => {
    const { state, props } = await mount()
    let resolve!: (value: any) => void
    mocks.events.mockImplementationOnce(
      () =>
        new Promise((done) => {
          resolve = done
        }),
    )
    state.expand('r1')
    await flush()
    const signal = mocks.events.mock.calls[0][3]
    props.deploymentId = 'd2'
    props.latestRunId = 'r2'
    mocks.runs.mockResolvedValue({ items: [run('r2')], total: 1 })
    await flush()
    resolve({ items: [{ id: 'late', message: '不应显示' }], total: 1 })
    await flush()
    expect(signal.aborted).toBe(true)
    expect(state.expandedId).toBe('')
    expect(host.textContent).not.toContain('不应显示')
  })
  it('高频revision不取消慢请求，合并补刷并保持第二页total正确', async () => {
    const { state, props } = await mount()
    state.page = 2
    await flush()
    let resolve!: (value: any) => void
    mocks.runs.mockImplementationOnce(
      () =>
        new Promise((done) => {
          resolve = done
        }),
    )
    props.revision++
    await flush()
    const count = mocks.runs.mock.calls.length
    const signal = mocks.runs.mock.calls.at(-1)![3]
    for (let i = 0; i < 20; i++) {
      props.revision++
      await flush()
    }
    expect(mocks.runs).toHaveBeenCalledTimes(count)
    expect(signal.aborted).toBe(false)
    mocks.runs.mockResolvedValue({ items: [run('older')], total: 36 })
    resolve({ items: [run('older')], total: 35 })
    await flush()
    expect(mocks.runs).toHaveBeenCalledTimes(count + 1)
    expect(state.total).toBe(36)
    expect(state.page).toBe(2)
  })
  it('隐藏暂停、恢复对账，失败保留内容，末页收敛', async () => {
    const { state, props } = await mount()
    props.active = false
    await flush()
    const count = mocks.runs.mock.calls.length
    props.revision++
    await flush()
    expect(mocks.runs).toHaveBeenCalledTimes(count)
    props.active = true
    await flush()
    expect(mocks.runs).toHaveBeenCalledTimes(count + 1)
    mocks.runs.mockRejectedValueOnce(new Error('offline'))
    props.revision++
    await flush()
    expect(host.textContent).toContain('记录更新中断')
    expect(state.runs).toHaveLength(1)
    mocks.runs.mockResolvedValue({ items: [run()], total: 1 })
    state.page = 4
    await flush()
    expect(state.page).toBe(1)
    expect(state.error).toBe(false)
  })
})
describe('部署详情真实状态呈现', () => {
  it('只在有效相等代次及期望状态匹配时判就绪/停止', () => {
    const service: any = {
      observedStatus: 'running',
      desiredStatus: 'running',
      observedGeneration: 2,
      desiredGeneration: 2,
      lastMessage: 'Kubernetes原文',
    }
    expect(deploymentEnginePresentation(service).technical).toBe('')
    expect(deploymentEnginePresentation({ ...service, observedGeneration: 3 }).state).toBe(
      'pending',
    )
    expect(
      deploymentEnginePresentation({
        ...service,
        observedGeneration: undefined,
        desiredGeneration: undefined,
      }).state,
    ).toBe('pending')
    expect(deploymentEnginePresentation({ ...service, observedStatus: 'stopped' }).state).toBe(
      'pending',
    )
    expect(
      deploymentEnginePresentation({
        ...service,
        desiredStatus: 'stopped',
        observedStatus: 'stopped',
      }).state,
    ).toBe('stopped')
    expect(
      deploymentEnginePresentation({
        ...service,
        observedStatus: 'failed',
        lastMessage: 'CrashLoopBackOff',
      }).summary,
    ).toBe('引擎反复启动失败')
  })
  it('失败/停止/未完成不伪装部署成功或制造倒计时', () => {
    expect(runOperationLabel(run())).toBe('停止')
    expect(runResultLabel({ ...run(), observedStatus: 'failed' })).toBe('失败')
    expect(runDurationLabel({ ...run(), durationMs: null, completedAt: undefined })).toBe('—')
  })
})
