import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import CodeWorkspacePanel from './CodeWorkspacePanel.vue'
import { codeWorkspaceApi } from './code-workspace-api'
import { contextPackApi } from './context-pack-api'

vi.mock('./code-workspace-api', () => ({
  codeWorkspaceApi: {
    get: vi.fn(),
    start: vi.fn(),
    stop: vi.fn(),
    rebuild: vi.fn(),
  },
}))

vi.mock('./context-pack-api', () => ({ contextPackApi: { refresh: vi.fn() } }))

const stoppedWorkspace = {
  status: 'stopped' as const,
  url: null,
  hostPort: 18080,
  containerName: 'induforge-code-project-1',
  onlineUsers: [],
}

const runningWorkspace = {
  status: 'running' as const,
  url: 'http://127.0.0.1:18080/',
  hostPort: 18080,
  containerName: 'induforge-code-project-1',
  onlineUsers: [
    { id: 'current', name: '当前用户', isCurrent: true },
    { id: 'other', name: '协作者', isCurrent: false },
  ],
}

describe('CodeWorkspacePanel', () => {
  beforeEach(() => {
    vi.mocked(codeWorkspaceApi.get).mockReset()
    vi.mocked(codeWorkspaceApi.start).mockReset()
    vi.mocked(codeWorkspaceApi.stop).mockReset()
    vi.mocked(codeWorkspaceApi.rebuild).mockReset()
    vi.mocked(contextPackApi.refresh).mockReset()
    vi.mocked(contextPackApi.refresh).mockResolvedValue({
      contractVersion: 'sha256:test', pointCount: 0, roleCount: 0, updatedAt: '2026-08-06 00:00:00',
    })
  })

  it('进入代码工作区后加载状态并打开运行中的 iframe', async () => {
    vi.mocked(codeWorkspaceApi.get).mockResolvedValue(runningWorkspace)

    const wrapper = mount(CodeWorkspacePanel, {
      props: { projectId: 'project-1', refreshKey: 0 },
    })
    await flushPromises()

    expect(codeWorkspaceApi.get).toHaveBeenCalledWith('project-1')
    expect(wrapper.get('iframe').attributes('src')).toBe('http://127.0.0.1:18080/')
    expect(wrapper.text()).toContain('1 位其他成员在线')
    expect(wrapper.emitted('url-change')?.at(-1)).toEqual(['http://127.0.0.1:18080/'])
  })

  it('停止状态可启动并在成功后打开 iframe', async () => {
    vi.mocked(codeWorkspaceApi.get).mockResolvedValue(stoppedWorkspace)
    vi.mocked(codeWorkspaceApi.start).mockResolvedValue(runningWorkspace)

    const wrapper = mount(CodeWorkspacePanel, {
      props: { projectId: 'project-1', refreshKey: 0 },
    })
    await flushPromises()

    expect(codeWorkspaceApi.start).toHaveBeenCalledWith('project-1')
    expect(contextPackApi.refresh).toHaveBeenCalledWith('project-1')
    expect(wrapper.get('iframe').attributes('src')).toBe('http://127.0.0.1:18080/')
  })

  it('外层刷新指令会重新查询状态', async () => {
    vi.mocked(codeWorkspaceApi.get).mockResolvedValue(stoppedWorkspace)

    const wrapper = mount(CodeWorkspacePanel, {
      props: { projectId: 'project-1', refreshKey: 0 },
    })
    await flushPromises()
    await wrapper.setProps({ refreshKey: 1 })
    await flushPromises()

    expect(codeWorkspaceApi.get).toHaveBeenCalledTimes(2)
    expect(codeWorkspaceApi.start).toHaveBeenCalledTimes(1)
  })
})
