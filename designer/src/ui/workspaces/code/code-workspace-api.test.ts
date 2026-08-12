import { beforeEach, describe, expect, it, vi } from 'vitest'
import request from '@/utils/request'
import { codeWorkspaceApi } from './code-workspace-api'

vi.mock('@/utils/request', () => ({
  default: {
    get: vi.fn(),
    post: vi.fn(),
  },
}))

const workspace = {
  status: 'running' as const,
  url: 'http://127.0.0.1:18080/',
  hostPort: 18080,
  containerName: 'induforge-code-project-1',
  onlineUsers: [],
}

describe('codeWorkspaceApi', () => {
  beforeEach(() => {
    vi.mocked(request.get).mockReset()
    vi.mocked(request.post).mockReset()
  })

  it('使用工程级状态接口并解包 data', async () => {
    vi.mocked(request.get).mockResolvedValue({ code: 0, msg: 'ok', data: workspace })

    await expect(codeWorkspaceApi.get('project/1')).resolves.toEqual(workspace)
    expect(request.get).toHaveBeenCalledWith('/projects/project%2F1/code-workspace')
  })

  it.each(['start', 'stop', 'rebuild'] as const)('调用 %s 操作接口', async (action) => {
    vi.mocked(request.post).mockResolvedValue({ code: 0, msg: 'ok', data: workspace })

    await expect(codeWorkspaceApi[action]('project-1')).resolves.toEqual(workspace)
    expect(request.post).toHaveBeenCalledWith(`/projects/project-1/code-workspace/${action}`)
  })

  it('缺少 data 时拒绝继续渲染', async () => {
    vi.mocked(request.get).mockResolvedValue({ code: 0, msg: 'ok' })

    await expect(codeWorkspaceApi.get('project-1')).rejects.toThrow('代码工作区接口未返回 data')
  })
})
