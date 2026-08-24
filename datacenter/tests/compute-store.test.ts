import { beforeEach, describe, expect, test, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

const getComputeUnits = vi.fn()

vi.mock('@/api/compute.api', () => ({
  getComputeUnits,
  getComputeUnit: vi.fn(),
  getComputeFolders: vi.fn(),
  createComputeUnit: vi.fn(),
  createComputeFolder: vi.fn(),
  updateComputeFolder: vi.fn(),
  updateComputeUnit: vi.fn(),
  deleteComputeFolder: vi.fn(),
  deleteComputeUnit: vi.fn(),
  toggleComputeUnit: vi.fn(),
  getComputeDependencies: vi.fn(),
}))

describe('compute store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    getComputeUnits.mockReset()
  })

  test('快速搜索时忽略较晚返回的旧响应', async () => {
    const pending: Array<{
      resolve: (value: unknown) => void
      search: unknown
    }> = []
    getComputeUnits.mockImplementation(
      (_projectId: string, params: Record<string, unknown>) =>
        new Promise((resolve) => pending.push({ resolve, search: params.search })),
    )

    const { useComputeStore } = await import('../src/stores/compute.store')
    const store = useComputeStore()
    const oldRequest = store.fetchList('project-1', { page: 1, pageSize: 50, search: '旧' })
    const latestRequest = store.fetchList('project-1', { page: 1, pageSize: 50, search: '新' })

    expect(pending.map((item) => item.search)).toEqual(['旧', '新'])
    pending[1].resolve({
      list: [{ id: 'new', name: '新结果' }],
      pagination: { page: 1, pageSize: 50, total: 1 },
    })
    await latestRequest
    pending[0].resolve({
      list: [{ id: 'old', name: '旧结果' }],
      pagination: { page: 1, pageSize: 50, total: 1 },
    })
    await oldRequest

    expect(store.list.map((item) => item.id)).toEqual(['new'])
    expect(store.total).toBe(1)
    expect(store.loading).toBe(false)
  })

  test('追加分页时按 ID 合并并更新分页位置', async () => {
    getComputeUnits
      .mockResolvedValueOnce({
        list: [{ id: 'unit-1', name: '单元 1' }],
        pagination: { page: 1, pageSize: 1, total: 2 },
      })
      .mockResolvedValueOnce({
        list: [
          { id: 'unit-1', name: '单元 1（更新）' },
          { id: 'unit-2', name: '单元 2' },
        ],
        pagination: { page: 2, pageSize: 1, total: 2 },
      })

    const { useComputeStore } = await import('../src/stores/compute.store')
    const store = useComputeStore()
    await store.fetchList('project-1', { page: 1, pageSize: 1 })
    await store.fetchList('project-1', { page: 2, pageSize: 1 }, { append: true })

    expect(store.list.map((item) => item.id)).toEqual(['unit-1', 'unit-2'])
    expect(store.list[0].name).toBe('单元 1（更新）')
    expect(store.page).toBe(2)
  })
})
