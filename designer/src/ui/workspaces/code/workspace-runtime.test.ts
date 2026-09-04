import { describe, expect, it, vi } from 'vitest'
import { createDevelopmentWorkspaceState, resolveWorkspaceState } from './workspace-runtime'

const stoppedWorkspace = {
  status: 'stopped' as const,
  containerName: 'production-workspace',
  onlineUsers: [],
  services: {
    ai: { url: null, hostPort: null },
    code: { url: null, hostPort: null },
    preview: { url: null, hostPort: null },
    previewControl: { url: null, hostPort: null },
  },
}

const runningWorkspace = {
  ...stoppedWorkspace,
  status: 'running' as const,
  services: {
    ai: { url: 'https://ai.workspace.test/', hostPort: null },
    code: { url: 'https://code.workspace.test/', hostPort: null },
    preview: { url: 'https://preview.workspace.test/', hostPort: null },
    previewControl: { url: 'https://control.workspace.test/', hostPort: null },
  },
}

describe('Designer 工作空间运行模式', () => {
  it('开发模式直接使用 WSL 映射地址且不调用后端', async () => {
    const get = vi.fn()
    const start = vi.fn()

    await expect(
      resolveWorkspaceState(
        'project-1',
        true,
        {
          DEV: true,
          MODE: 'development',
          VITE_DESIGNER_AI_URL: 'http://127.0.0.1:33141',
          VITE_DESIGNER_CODE_URL: 'http://127.0.0.1:33000',
          VITE_DESIGNER_PREVIEW_URL: 'http://127.0.0.1:35173',
          VITE_DESIGNER_PREVIEW_CONTROL_URL: 'http://127.0.0.1:35174',
          VITE_DESIGNER_WORKSPACE_INSTANCE_ID: 'induforge-piweb-smoke',
        },
        { get, start },
      ),
    ).resolves.toEqual({
      status: 'running',
      containerName: 'induforge-piweb-smoke',
      onlineUsers: [],
      services: {
        ai: { url: 'http://127.0.0.1:33141/', hostPort: 33141 },
        code: { url: 'http://127.0.0.1:33000/', hostPort: 33000 },
        preview: { url: 'http://127.0.0.1:35173/', hostPort: 35173 },
        previewControl: { url: 'http://127.0.0.1:35174/', hostPort: 35174 },
      },
    })
    expect(get).not.toHaveBeenCalled()
    expect(start).not.toHaveBeenCalled()
  })

  it('生产模式通过 dev_core 查询并启动工程工作空间', async () => {
    const get = vi.fn().mockResolvedValue(stoppedWorkspace)
    const start = vi.fn().mockResolvedValue(runningWorkspace)

    await expect(
      resolveWorkspaceState('project-1', true, { DEV: false }, { get, start }),
    ).resolves.toEqual(runningWorkspace)
    expect(get).toHaveBeenCalledWith('project-1')
    expect(start).toHaveBeenCalledWith('project-1')
  })

  it('frontend-linux 即使是 Vite 开发态也通过中心 API 查询并启动工程工作空间', async () => {
    const get = vi.fn().mockResolvedValue(stoppedWorkspace)
    const start = vi.fn().mockResolvedValue(runningWorkspace)

    await expect(
      resolveWorkspaceState(
        'project-1',
        true,
        {
          DEV: true,
          MODE: 'frontend-linux',
          // 这些本机地址即使存在，也不能在 Linux 中心模式使用。
          VITE_DESIGNER_AI_URL: 'http://127.0.0.1:33141',
        },
        { get, start },
      ),
    ).resolves.toEqual(runningWorkspace)
    expect(get).toHaveBeenCalledWith('project-1')
    expect(start).toHaveBeenCalledWith('project-1')
  })

  it('开发模式缺少必需地址时立即失败', () => {
    expect(() =>
      createDevelopmentWorkspaceState({
        DEV: true,
        VITE_DESIGNER_AI_URL: 'http://127.0.0.1:33141',
      }),
    ).toThrow('开发环境必须配置')
  })
})
