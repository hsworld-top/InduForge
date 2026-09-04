import { beforeEach, describe, expect, it, vi } from 'vitest'
import request from '@/utils/request'
import { contextPackApi } from './context-pack-api'

vi.mock('@/utils/request', () => ({ default: { post: vi.fn() } }))

describe('contextPackApi', () => {
  beforeEach(() => vi.mocked(request.post).mockReset())

  it('刷新工程上下文并解包结果', async () => {
    const result = { contractVersion: 'sha256:test', pointCount: 2, roleCount: 1, updatedAt: '2026-08-06 00:00:00' }
    vi.mocked(request.post).mockResolvedValue({ code: 0, msg: 'ok', data: result })

    await expect(contextPackApi.refresh('project/1')).resolves.toEqual(result)
    expect(request.post).toHaveBeenCalledWith(
      '/projects/project%2F1/workspace-context/refresh',
      undefined,
      { authoringProjectId: 'project/1' },
    )
  })
})
