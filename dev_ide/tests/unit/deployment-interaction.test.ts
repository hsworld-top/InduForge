import { afterEach, describe, expect, it, vi } from 'vitest'
import type { ProjectDeployment } from '@/api/ops.api'
import {
  canOperateDeployment,
  createDeploymentRefresh,
  deploymentAccessUrl,
  deploymentBusy,
  deploymentProgressLabel,
  deploymentMode,
  deploymentReplacementMessage,
} from '@/views/tenant/utils/deployment-interaction'

const deployment = (extra: Partial<ProjectDeployment> = {}): ProjectDeployment => ({
  id: 'd1',
  projectId: 'p1',
  projectName: '工程',
  environmentId: 'e1',
  accessPort: 17800,
  observedStatus: 'running',
  desiredStatus: 'running',
  entryStatus: 'running',
  ...extra,
})
const flush = async () => {
  await Promise.resolve()
  await Promise.resolve()
  await Promise.resolve()
}
afterEach(() => vi.useRealTimers())

describe('工程部署动作和来源边界', () => {
  it('并行进度使用真实就绪数，失败与终态不继续播放启动状态', () => {
    const running = {
      serviceType: 'base' as const,
      observedStatus: 'running',
      desiredGeneration: 2,
      observedGeneration: 2,
    }
    const pending = {
      serviceType: 'compute' as const,
      observedStatus: 'pending',
      desiredGeneration: 2,
      observedGeneration: 1,
    }
    expect(
      deploymentProgressLabel(
        deployment({ observedStatus: 'pending', services: [running, pending] }),
      ),
    ).toBe('正在启动计算引擎 · 1/2 已就绪')
    expect(deploymentProgressLabel(deployment({ services: [running] }))).toBe('运行中')
    expect(
      deploymentProgressLabel(
        deployment({
          observedStatus: 'failed',
          services: [{ ...pending, observedStatus: 'failed' }],
        }),
      ),
    ).toBe('执行失败 · 计算引擎')
  })
  it('稳定运行允许停止重启，停止允许启动，执行中和权限不足禁用', () => {
    expect(canOperateDeployment(deployment(), 'stop', true)).toBe(true)
    expect(canOperateDeployment(deployment(), 'start', true)).toBe(false)
    expect(
      canOperateDeployment(
        deployment({ observedStatus: 'stopped', desiredStatus: 'stopped' }),
        'start',
        true,
      ),
    ).toBe(true)
    for (const action of ['start', 'stop', 'restart', 'redeploy', 'delete'] as const) {
      expect(canOperateDeployment(deployment(), action, false)).toBe(false)
      expect(canOperateDeployment(deployment(), action, true, true)).toBe(false)
      expect(canOperateDeployment(deployment({ observedStatus: 'pending' }), action, true)).toBe(
        false,
      )
    }
    expect(
      deploymentBusy(
        deployment({
          services: [{ serviceType: 'base', desiredGeneration: 2, observedGeneration: 1 }],
        }),
      ),
    ).toBe(true)
    expect(canOperateDeployment(deployment({ observedStatus: 'failed' }), 'redeploy', true)).toBe(
      true,
    )
  })
  it('模式切换明确替换唯一部署，同模式更新也提示替换当前制品', () => {
    expect(deploymentMode(deployment({ version: '__DEV__' }))).toBe('development')
    expect(
      deploymentReplacementMessage(deployment({ mode: 'development' }), 'production'),
    ).toContain('切换部署模式将替换当前部署')
    expect(
      deploymentReplacementMessage(deployment({ mode: 'production' }), 'production'),
    ).toContain('更新将替换当前部署制品')
    expect(deploymentReplacementMessage(deployment(), 'development')).toContain(
      '不会创建第二套运行环境或隔离数据',
    )
  })
  it('仅允许就绪工程的安全HTTP外链', () => {
    expect(deploymentAccessUrl(deployment({ accessUrl: 'http://172.16.125.129:17800/' }))).toBe(
      'http://172.16.125.129:17800/',
    )
    for (const accessUrl of ['javascript:alert(1)', 'http://user:pass@host/', '/unsafe'])
      expect(deploymentAccessUrl(deployment({ accessUrl }))).toBe('')
    expect(
      deploymentAccessUrl(deployment({ accessUrl: 'https://host/', accessAvailable: false })),
    ).toBe('')
  })
})

describe('静默刷新生命周期', () => {
  it('激活立即刷新，稳定5秒/执行中2秒；暂停与恢复及卸载不会遗留请求', async () => {
    vi.useFakeTimers()
    let busy = false
    const load = vi.fn().mockResolvedValue(undefined)
    const poll = createDeploymentRefresh({ load, busy: () => busy, error: vi.fn() })
    poll.setActive(true)
    await flush()
    expect(load).toHaveBeenCalledTimes(1)
    await vi.advanceTimersByTimeAsync(4999)
    expect(load).toHaveBeenCalledTimes(1)
    busy = true
    await vi.advanceTimersByTimeAsync(1)
    expect(load).toHaveBeenCalledTimes(2)
    await vi.advanceTimersByTimeAsync(2000)
    expect(load).toHaveBeenCalledTimes(3)
    poll.setActive(false)
    await vi.advanceTimersByTimeAsync(10000)
    expect(load).toHaveBeenCalledTimes(3)
    poll.setActive(true)
    await flush()
    expect(load).toHaveBeenCalledTimes(4)
    poll.dispose()
    await vi.advanceTimersByTimeAsync(10000)
    expect(load).toHaveBeenCalledTimes(4)
  })
  it('快速刷新合并为一次后续请求，隐藏取消后不接受过期响应', async () => {
    vi.useFakeTimers()
    let resolve: () => void = () => {}
    let accepted = 0
    const signals: AbortSignal[] = []
    const load = vi.fn((signal: AbortSignal) => {
      signals.push(signal)
      return new Promise<void>((done) => {
        resolve = () => {
          if (!signal.aborted) accepted++
          done()
        }
      })
    })
    const poll = createDeploymentRefresh({ load, busy: () => false, error: vi.fn() })
    poll.setActive(true)
    void poll.refresh()
    void poll.refresh()
    expect(load).toHaveBeenCalledTimes(1)
    poll.setActive(false)
    expect(signals[0].aborted).toBe(true)
    poll.setActive(true)
    resolve()
    await flush()
    expect(accepted).toBe(0)
    expect(load).toHaveBeenCalledTimes(2)
    resolve()
    await flush()
    expect(accepted).toBe(1)
    poll.dispose()
  })
  it('失败通过状态回调报告且仍会恢复刷新，取消不报错', async () => {
    vi.useFakeTimers()
    const error = vi.fn()
    const load = vi.fn().mockRejectedValueOnce(new Error('offline')).mockResolvedValue(undefined)
    const poll = createDeploymentRefresh({ load, busy: () => false, error })
    poll.setActive(true)
    await flush()
    expect(error).toHaveBeenCalledTimes(1)
    await vi.advanceTimersByTimeAsync(5000)
    expect(load).toHaveBeenCalledTimes(2)
    poll.dispose()
  })
})
