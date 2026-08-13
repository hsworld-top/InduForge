import { beforeEach, describe, expect, it, vi } from 'vitest'
import { applyMicroAppContext, requestWorkspaceOpen } from './wujie-context'

describe('Wujie 工程工具请求', () => {
  beforeEach(() => applyMicroAppContext({ projectId: 'reset-project' }))

  it('只向宿主发送当前工程和受控目标', () => {
    const onOpenWorkspace = vi.fn()
    applyMicroAppContext({
      projectId: 'project-1',
      projectName: '产线监控',
      onOpenWorkspace,
    })

    expect(requestWorkspaceOpen('3d')).toBe(true)
    expect(onOpenWorkspace).toHaveBeenCalledWith({
      type: 'WORKSPACE_OPEN_REQUEST',
      projectId: 'project-1',
      target: '3d',
    })
  })

  it('宿主未注入回调时拒绝伪造打开行为', () => {
    applyMicroAppContext({ projectId: 'project-1' })
    expect(requestWorkspaceOpen('2d')).toBe(false)
  })

  it('拒绝白名单之外的工具目标', () => {
    const onOpenWorkspace = vi.fn()
    applyMicroAppContext({ projectId: 'project-1', onOpenWorkspace })

    expect(requestWorkspaceOpen('code' as never)).toBe(false)
    expect(onOpenWorkspace).not.toHaveBeenCalled()
  })
})
