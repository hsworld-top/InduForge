import { describe, expect, it, vi } from 'vitest'
import request from '@/utils/request'
import { sceneContractApi } from './scene-contract-api'

vi.mock('@/utils/request', () => ({
  default: { get: vi.fn(), post: vi.fn(), put: vi.fn(), delete: vi.fn() },
}))

describe('sceneContractApi', () => {
  it('使用工程场景分页接口生成公开契约快照', async () => {
    vi.mocked(request.get).mockResolvedValue({
      data: {
        items: [
          {
            sceneId: 'main',
            kind: '2d',
            name: '主画面',
            publicContract: { description: '', parameters: [], events: [], commands: [] },
            datapointRefs: [],
            currentRevision: 3,
            draftVersion: 4,
            committedDraftVersion: 3,
          },
        ],
        total: 1,
      },
    })
    await expect(sceneContractApi.list('project/a')).resolves.toEqual({
      contractVersion: '3',
      contracts: [expect.objectContaining({ id: 'main', kind: '2d', contractVersion: '3' })],
    })
    expect(request.get).toHaveBeenCalledWith('/projects/project%2Fa/scenes', {
      params: { page: 1, limit: 200, sort: 'name', order: 'asc' },
    })
  })

  it('通过场景集合接口显式创建场景', async () => {
    vi.mocked(request.post).mockResolvedValue({
      data: {
        sceneId: 'factory-main',
        kind: '3d',
        name: '主厂区',
        publicContract: { description: '', parameters: [], events: [], commands: [] },
        datapointRefs: [],
        currentRevision: 0,
        draftVersion: 1,
        committedDraftVersion: 0,
      },
    })

    await expect(
      sceneContractApi.create('project/a', {
        kind: '3d',
        name: '主厂区',
      }),
    ).resolves.toEqual(
      expect.objectContaining({
        id: 'factory-main',
        kind: '3d',
        name: '主厂区',
      }),
    )
    expect(request.post).toHaveBeenCalledWith('/projects/project%2Fa/scenes', {
      kind: '3d',
      name: '主厂区',
      publicContract: { description: '', parameters: [], events: [], commands: [] },
    })
  })
})
