import { describe, expect, it, vi } from 'vitest'
import request from '@/utils/request'
import { sceneContractApi } from './scene-contract-api'

vi.mock('@/utils/request', () => ({
  default: { get: vi.fn(), put: vi.fn(), delete: vi.fn() },
}))

describe('sceneContractApi', () => {
  it('uses the project-scoped scene contract endpoint', async () => {
    vi.mocked(request.get).mockResolvedValue({ data: { contractVersion: 'v1', contracts: [] } })
    await expect(sceneContractApi.list('project/a')).resolves.toEqual({ contractVersion: 'v1', contracts: [] })
    expect(request.get).toHaveBeenCalledWith('/projects/project%2Fa/scene-contracts')
  })
})
