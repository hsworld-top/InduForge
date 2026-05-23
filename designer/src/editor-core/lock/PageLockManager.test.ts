import { describe, expect, it, vi } from 'vitest'
import { PageLockManager } from './PageLockManager'

describe('PageLockManager', () => {
  it('旧释放请求完成后不会清掉新获取的页面锁', async () => {
    let resolveDelete!: () => void
    const pendingDelete = new Promise<void>((resolve) => {
      resolveDelete = resolve
    })

    const api = {
      get: vi.fn(),
      post: vi.fn().mockImplementation(async (path: string) => ({
        code: 0,
        msg: 'ok',
        data: {
          locked: true,
          lockedBy: 'user-1',
          lockedByName: `锁-${path}`,
          lockedAt: Date.now(),
        },
      })),
      delete: vi.fn().mockImplementation(async () => {
        await pendingDelete
        return { code: 0, msg: 'ok', data: null }
      }),
    }

    const manager = new PageLockManager({
      api,
      currentUserId: 'user-1',
      heartbeatInterval: 60_000,
    })

    await manager.acquireLock('page-a')

    const releasePromise = manager.releaseLock()
    await Promise.resolve()

    await manager.acquireLock('page-b')
    resolveDelete()
    await releasePromise

    expect(manager.getCurrentPageId()).toBe('page-b')
    expect(manager.getLockState()).toEqual(
      expect.objectContaining({
        pageId: 'page-b',
        locked: true,
        isOwner: true,
      }),
    )
  })

  it('获取页面锁遇到业务失败时返回 locked 结果', async () => {
    const api = {
      get: vi.fn(),
      post: vi.fn().mockRejectedValue({
        isBusinessError: true,
        code: 25003,
        message: '页面已被其他用户锁定',
        data: {
          locked: true,
          lockedBy: 'user-2',
          lockedByName: '用户二',
          lockedAt: Date.now(),
        },
      }),
      delete: vi.fn(),
    }

    const manager = new PageLockManager({
      api,
      currentUserId: 'user-1',
      heartbeatInterval: 60_000,
    })

    const result = await manager.acquireLock('page-a')

    expect(result).toEqual({
      success: false,
      reason: 'locked',
      lockedByName: '用户二',
    })
    expect(manager.getLockState()).toEqual(
      expect.objectContaining({
        pageId: 'page-a',
        locked: true,
        lockedBy: 'user-2',
        isOwner: false,
      }),
    )
  })

  it('非锁冲突业务失败返回 error 并保留业务 code/message', async () => {
    const api = {
      get: vi.fn(),
      post: vi.fn().mockResolvedValue({
        code: 25001,
        msg: '页面参数不合法',
        data: { field: 'pageId' },
      }),
      delete: vi.fn(),
    }

    const manager = new PageLockManager({
      api,
      currentUserId: 'user-1',
      heartbeatInterval: 60_000,
    })

    const result = await manager.acquireLock('page-a')
    expect(result.success).toBe(false)
    if (result.success) {
      return
    }

    expect(result.reason).toBe('error')
    expect(result.error).toMatchObject({
      message: '页面参数不合法',
      code: 25001,
      isBusinessError: true,
    })
  })

  it('acquireLock 遇到旧包络 success=false 时按失败处理', async () => {
    const api = {
      get: vi.fn(),
      post: vi.fn().mockResolvedValue({
        success: false,
        message: '页面已被其他用户锁定',
        data: {
          locked: true,
          lockedBy: 'user-2',
          lockedByName: '用户二',
          lockedAt: Date.now(),
        },
      }),
      delete: vi.fn(),
    }

    const manager = new PageLockManager({
      api,
      currentUserId: 'user-1',
      heartbeatInterval: 60_000,
    })

    const result = await manager.acquireLock('page-a')
    expect(result).toEqual({
      success: false,
      reason: 'locked',
      lockedByName: '用户二',
    })
    expect(manager.getLockState()).toEqual(
      expect.objectContaining({
        pageId: 'page-a',
        locked: true,
        lockedBy: 'user-2',
        isOwner: false,
      }),
    )
  })

  it('queryLockStatus 遇到旧包络 success=false 时抛业务错误', async () => {
    const api = {
      get: vi.fn().mockResolvedValue({
        success: false,
        message: '查询失败',
        data: {
          reason: 'forbidden',
        },
      }),
      post: vi.fn(),
      delete: vi.fn(),
    }

    const manager = new PageLockManager({
      api,
      currentUserId: 'user-1',
      heartbeatInterval: 60_000,
    })

    await expect(manager.queryLockStatus('page-a')).rejects.toMatchObject({
      name: 'PageLockBusinessError',
      message: '查询失败',
      isBusinessError: true,
    })
  })

  it('释放挂起期间仅丢失当前锁时仍会清理残留状态', async () => {
    let resolveDelete!: () => void
    const pendingDelete = new Promise<void>((resolve) => {
      resolveDelete = resolve
    })

    const api = {
      get: vi.fn(),
      post: vi.fn().mockResolvedValue({
        code: 0,
        msg: 'ok',
        data: {
          locked: true,
          lockedBy: 'user-1',
          lockedByName: '当前锁',
          lockedAt: Date.now(),
        },
      }),
      delete: vi.fn().mockImplementation(async () => {
        await pendingDelete
        return { code: 0, msg: 'ok', data: null }
      }),
    }

    const manager = new PageLockManager({
      api,
      currentUserId: 'user-1',
      heartbeatInterval: 60_000,
    })

    await manager.acquireLock('page-a')

    const releasePromise = manager.releaseLock()
    await Promise.resolve()

    manager._handleLockLost()
    resolveDelete()
    await releasePromise

    expect(manager.getCurrentPageId()).toBeNull()
    expect(manager.getLockState()).toBeNull()
  })
})
