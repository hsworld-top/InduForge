import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
const fake = vi.hoisted(() => ({
  connected: true,
  listeners: new Map<string, Set<(...args: any[]) => void>>(),
  emit: vi.fn(),
  close: vi.fn(),
}))
vi.mock('@/utils/socket', () => ({
  initSocket: () => ({
    get connected() {
      return fake.connected
    },
    emit: fake.emit,
    close: fake.close,
    on(name: string, handler: (...args: any[]) => void) {
      if (!fake.listeners.has(name)) fake.listeners.set(name, new Set())
      fake.listeners.get(name)!.add(handler)
    },
    off(name: string, handler: (...args: any[]) => void) {
      fake.listeners.get(name)?.delete(handler)
    },
  }),
}))
import { acquireOpsWatch, createOpsRealtimeMonitor } from '@/utils/ops-realtime'
const send = (name: string, data?: unknown) =>
  fake.listeners.get(name)?.forEach((handler) => handler(data))
const flush = async () => {
  for (let i = 0; i < 15; i++) await Promise.resolve()
}
const ready = (topics = ['deployments']) =>
  send('ops:ready', {
    subscriptionId: fake.emit.mock.calls.filter((call) => call[0] === 'ops:watch').at(-1)?.[1]
      .subscriptionId,
    epoch: 'a',
    sequence: 1,
    topics,
  })
const disposables: Array<() => void> = []
beforeEach(() => {
  vi.useFakeTimers()
  fake.connected = true
  fake.emit.mockClear()
  fake.close.mockClear()
})
afterEach(() => {
  disposables.splice(0).forEach((dispose) => dispose())
  fake.listeners.clear()
  vi.useRealTimers()
})

describe('统一运维实时订阅', () => {
  it('引用计数只卸载自己的监听，旧ready/重复乱序事件不交付', () => {
    const first = vi.fn()
    const second = vi.fn()
    const a = acquireOpsWatch({ change: first, connection: vi.fn() })
    const b = acquireOpsWatch({ change: second, connection: vi.fn() })
    disposables.push(
      () => a.release(),
      () => b.release(),
    )
    a.setScope({ topics: ['deployments'] })
    ready()
    b.setScope({ topics: ['nodes'] })
    ready(['nodes'])
    expect(fake.listeners.get('ops:change')?.size).toBe(1)
    send('ops:change', { epoch: 'a', sequence: 4, topics: ['deployments'] })
    send('ops:change', { epoch: 'a', sequence: 3, topics: ['deployments'] })
    expect(first).toHaveBeenCalledTimes(2)
    expect(second).toHaveBeenCalledTimes(1)
    a.setScope(null)
    send('ops:change', { epoch: 'a', sequence: 5, topics: ['nodes'] })
    expect(second).toHaveBeenCalledTimes(2)
    a.release()
    expect(fake.close).not.toHaveBeenCalled()
    expect(fake.listeners.get('ops:change')?.size).toBe(1)
  })
  it('online没有轮询，ready/重连对账，隐藏取消和恢复不泄漏', async () => {
    const snapshot = vi.fn().mockResolvedValue(undefined)
    const monitor = createOpsRealtimeMonitor({ snapshot, status: vi.fn(), random: () => 0.5 })
    disposables.push(monitor.dispose)
    monitor.setScope({ topics: ['deployments'] })
    await flush()
    ready()
    await flush()
    const count = snapshot.mock.calls.length
    await vi.advanceTimersByTimeAsync(120000)
    expect(snapshot).toHaveBeenCalledTimes(count)
    fake.connected = false
    send('disconnect')
    await vi.advanceTimersByTimeAsync(15000)
    expect(snapshot).toHaveBeenCalledTimes(count + 1)
    await vi.advanceTimersByTimeAsync(30000)
    expect(snapshot).toHaveBeenCalledTimes(count + 2)
    monitor.setScope(null)
    const paused = snapshot.mock.calls.length
    await vi.advanceTimersByTimeAsync(120000)
    expect(snapshot).toHaveBeenCalledTimes(paused)
    fake.connected = true
    monitor.setScope({ topics: ['deployments'] })
    await flush()
    ready()
    await flush()
    expect(snapshot.mock.calls.length).toBeGreaterThan(paused)
  })
  it('连接正常但HTTP快照失败仍退避重试；权限裁剪为空不持续请求', async () => {
    const snapshot = vi.fn().mockResolvedValue(undefined)
    const status = vi.fn()
    const monitor = createOpsRealtimeMonitor({ snapshot, status, random: () => 0.5 })
    disposables.push(monitor.dispose)
    monitor.setScope({ topics: ['deployments'] })
    await flush()
    ready()
    await flush()
    snapshot.mockRejectedValueOnce(new Error('offline'))
    send('ops:change', { epoch: 'a', sequence: 2, topics: ['deployments'], terminal: true })
    await flush()
    expect(status).toHaveBeenLastCalledWith('stale')
    const count = snapshot.mock.calls.length
    await vi.advanceTimersByTimeAsync(15000)
    expect(snapshot).toHaveBeenCalledTimes(count + 1)
    expect(status).toHaveBeenLastCalledWith('online')
    monitor.setScope(null)
    monitor.setScope({ topics: ['nodes'] })
    await flush()
    ready([])
    await flush()
    const denied = snapshot.mock.calls.length
    await vi.advanceTimersByTimeAsync(180000)
    expect(snapshot).toHaveBeenCalledTimes(denied)
  })
  it('100事件每秒和慢HTTP不会无限abort或饥饿，普通刷新节拍不超过每秒一次', async () => {
    let lastRead = 0
    let version = 0
    const signals: AbortSignal[] = []
    const snapshot = vi.fn(async (_topics, signal: AbortSignal) => {
      signals.push(signal)
      await new Promise((done) => setTimeout(done, 20))
      lastRead = version
    })
    const monitor = createOpsRealtimeMonitor({ snapshot, status: vi.fn(), random: () => 0.5 })
    disposables.push(monitor.dispose)
    monitor.setScope({ topics: ['deployments'] })
    await vi.advanceTimersByTimeAsync(20)
    ready()
    await vi.advanceTimersByTimeAsync(20)
    const initial = snapshot.mock.calls.length
    for (let i = 0; i < 500; i++) {
      version++
      send('ops:change', {
        epoch: 'a',
        sequence: i + 2,
        topics: ['deployments'],
        entityIds: ['d1'],
      })
      await vi.advanceTimersByTimeAsync(10)
    }
    await vi.advanceTimersByTimeAsync(1100)
    expect(snapshot.mock.calls.length - initial).toBeLessThanOrEqual(6)
    expect(snapshot.mock.calls.length - initial).toBeGreaterThanOrEqual(4)
    expect(signals.every((signal) => !signal.aborted)).toBe(true)
    expect(lastRead).toBe(version)
  })
})
