import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
const mocks = vi.hoisted(() => ({
  handlers: new Map<string, (...args: any[]) => void>(),
  refresh: vi.fn(),
  redirect: vi.fn(),
  connect: vi.fn(),
  disconnect: vi.fn(),
  emit: vi.fn(),
  host: '',
  options: {} as Record<string, unknown>,
}))
vi.mock('socket.io-client', () => ({
  io: (_host: string, options: Record<string, unknown>) => {
    mocks.host = _host
    mocks.options = options
    return {
      connected: true,
      on: (name: string, handler: (...args: any[]) => void) => mocks.handlers.set(name, handler),
      connect: mocks.connect,
      disconnect: mocks.disconnect,
      close: vi.fn(),
      emit: mocks.emit,
    }
  },
}))
vi.mock('@/utils/request', () => ({
  refreshSession: mocks.refresh,
  clearAuthAndRedirectToLogin: mocks.redirect,
}))
import { closeSocket, initSocket } from '@/utils/socket'
const flush = async () => {
  for (let i = 0; i < 8; i++) await Promise.resolve()
}
beforeEach(() => {
  vi.useFakeTimers()
  vi.setSystemTime(new Date('2026-09-03T00:00:00Z'))
  mocks.handlers.clear()
  mocks.refresh.mockReset().mockResolvedValue(true)
  mocks.redirect.mockReset()
  mocks.connect.mockReset()
  mocks.disconnect.mockReset()
  mocks.emit.mockReset()
  mocks.host = ''
})
afterEach(() => {
  closeSocket()
  vi.useRealTimers()
})
describe('Socket HttpOnly会话恢复', () => {
  it('控制面 Socket 固定连接浏览器当前 origin，不读取后端绝对地址', () => {
    initSocket()
    expect(mocks.host).toBe(window.location.origin)
    expect(mocks.options.path).toBe('/control-socket.io')
  })

  it('运维先建立连接时旧消费者补订阅，旧消费者先建立时登记不被覆盖，重连均恢复', () => {
    const first = initSocket()
    expect(initSocket('tenant-1')).toBe(first)
    expect(mocks.emit).toHaveBeenLastCalledWith('ops:subscribe', { tenantId: 'tenant-1' })
    mocks.handlers.get('connect')?.()
    expect(mocks.emit).toHaveBeenCalledTimes(2)
    closeSocket()
    mocks.emit.mockClear()
    const legacy = initSocket('tenant-2')
    expect(initSocket()).toBe(legacy)
    mocks.handlers.get('connect')?.()
    expect(mocks.emit).toHaveBeenLastCalledWith('ops:subscribe', { tenantId: 'tenant-2' })
  })
  it('长期网络中断不耗尽自动重连，间隔封顶且恢复继续旧消费者订阅', async () => {
    initSocket('tenant-1')
    expect(mocks.options.reconnectionAttempts).toBe(Infinity)
    expect(mocks.options.reconnectionDelayMax).toBe(30000)
    mocks.handlers.get('disconnect')?.('transport close')
    await vi.advanceTimersByTimeAsync(600000)
    expect(mocks.refresh).not.toHaveBeenCalled()
    mocks.handlers.get('connect')?.()
    expect(mocks.emit).toHaveBeenLastCalledWith('ops:subscribe', { tenantId: 'tenant-1' })
  })
  it('到期与主动断开合并为一次续租，有限退避重连携带新cookie', async () => {
    initSocket()
    mocks.handlers.get('ops:auth')?.({ code: 'AUTH_EXPIRED' })
    mocks.handlers.get('disconnect')?.('io server disconnect')
    await flush()
    expect(mocks.refresh).toHaveBeenCalledTimes(1)
    await vi.advanceTimersByTimeAsync(250)
    mocks.handlers.get('disconnect')?.('io server disconnect')
    expect(mocks.refresh).toHaveBeenCalledTimes(1)
    expect(mocks.connect).not.toHaveBeenCalled()
    await vi.advanceTimersByTimeAsync(250)
    expect(mocks.connect).toHaveBeenCalledTimes(1)
  })
  it('续租失败结束重试并交统一登录清理，撤权不续租', async () => {
    mocks.refresh.mockResolvedValueOnce(false)
    initSocket()
    mocks.handlers.get('connect_error')?.({ data: { code: 'AUTH_EXPIRED' } })
    await flush()
    expect(mocks.redirect).toHaveBeenCalledTimes(1)
    await vi.advanceTimersByTimeAsync(60000)
    expect(mocks.connect).not.toHaveBeenCalled()
    closeSocket()
    mocks.refresh.mockClear()
    initSocket()
    mocks.handlers.get('ops:auth')?.({ code: 'AUTH_FORBIDDEN' })
    mocks.handlers.get('disconnect')?.('io server disconnect')
    await flush()
    expect(mocks.refresh).not.toHaveBeenCalled()
  })
  it('普通网络错误不会轮换cookie，关闭连接取消延迟重连', async () => {
    initSocket()
    mocks.handlers.get('connect_error')?.(new Error('network'))
    mocks.handlers.get('disconnect')?.('transport close')
    expect(mocks.refresh).not.toHaveBeenCalled()
    mocks.handlers.get('ops:auth')?.({ code: 'AUTH_EXPIRED' })
    await flush()
    closeSocket()
    await vi.advanceTimersByTimeAsync(60000)
    expect(mocks.connect).not.toHaveBeenCalled()
  })
})
