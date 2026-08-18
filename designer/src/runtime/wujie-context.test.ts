import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { getEditorUiStore } from '@/stores/editor-ui-store'
import { applyMicroAppContext, initializeWujieContext, requestWorkspaceOpen } from './wujie-context'

describe('Wujie 工程工具请求', () => {
  beforeEach(() => applyMicroAppContext({ projectId: 'reset-project' }))

  afterEach(() => {
    delete window.__POWERED_BY_WUJIE__
    delete window.$wujie
  })

  it('只向宿主发送当前工程和受控目标', () => {
    const onOpenWorkspace = vi.fn()
    applyMicroAppContext({
      projectId: 'project-1',
      projectName: '产线监控',
      onOpenWorkspace,
    })

    expect(requestWorkspaceOpen('3d', 'factory')).toBe(true)
    expect(onOpenWorkspace).toHaveBeenCalledWith({
      type: 'WORKSPACE_OPEN_REQUEST',
      projectId: 'project-1',
      target: '3d',
      sceneId: 'factory',
    })
  })

  it('宿主未注入回调时拒绝伪造打开行为', () => {
    applyMicroAppContext({ projectId: 'project-1' })
    expect(requestWorkspaceOpen('2d', 'overview')).toBe(false)
  })

  it('拒绝白名单之外的工具目标', () => {
    const onOpenWorkspace = vi.fn()
    applyMicroAppContext({ projectId: 'project-1', onOpenWorkspace })

    expect(requestWorkspaceOpen('code' as never, 'overview')).toBe(false)
    expect(onOpenWorkspace).not.toHaveBeenCalled()
  })

  it('建立上下文就绪握手并持续应用宿主主题与语言', () => {
    const listeners = new Map<string, (payload: unknown) => void>()
    const emit = vi.fn()
    window.__POWERED_BY_WUJIE__ = true
    window.$wujie = {
      props: {
        instanceName: 'designer-project-1',
        projectId: 'project-1',
        theme: 'light',
        locale: 'zh',
      },
      bus: {
        $on: (event, handler) => listeners.set(event, handler),
        $emit: emit,
      },
    }

    initializeWujieContext()

    expect(emit).toHaveBeenCalledWith('micro-app:designer-project-1:context-ready')
    listeners.get('micro-app:designer-project-1:context')?.({
      instanceName: 'designer-project-1',
      projectId: 'project-1',
      theme: 'dark',
      locale: 'en',
    })
    expect(getEditorUiStore().theme.value).toBe('dark')
    expect(getEditorUiStore().locale.value).toBe('en')
  })
})
