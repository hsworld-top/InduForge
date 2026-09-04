import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { getEditorUiStore } from '@/stores/editor-ui-store'
import {
  applyMicroAppContext,
  getCurrentAuthoringEpoch,
  initializeWujieContext,
  reportAuthoringStale,
  requestWorkspaceClose,
  requestWorkspaceOpen,
  subscribeSceneCommitted,
} from './wujie-context'

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

    expect(requestWorkspaceOpen('3d', 'factory', '主厂区')).toBe(true)
    expect(onOpenWorkspace).toHaveBeenCalledWith({
      type: 'WORKSPACE_OPEN_REQUEST',
      projectId: 'project-1',
      target: '3d',
      sceneId: 'factory',
      sceneName: '主厂区',
    })
  })

  it('宿主未注入回调时拒绝伪造打开行为', () => {
    applyMicroAppContext({ projectId: 'project-1' })
    expect(requestWorkspaceOpen('2d', 'overview', '产线总览')).toBe(false)
  })

  it('拒绝白名单之外的工具目标', () => {
    const onOpenWorkspace = vi.fn()
    applyMicroAppContext({ projectId: 'project-1', onOpenWorkspace })

    expect(requestWorkspaceOpen('code' as never, 'overview', '产线总览')).toBe(false)
    expect(onOpenWorkspace).not.toHaveBeenCalled()
  })

  it('删除场景后请求宿主关闭对应工具标签', () => {
    const onCloseWorkspace = vi.fn()
    applyMicroAppContext({ projectId: 'project-1', onCloseWorkspace })

    expect(requestWorkspaceClose('2d', 'overview')).toBe(true)
    expect(onCloseWorkspace).toHaveBeenCalledWith({
      type: 'WORKSPACE_CLOSE_REQUEST',
      projectId: 'project-1',
      target: '2d',
      sceneId: 'overview',
    })
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

    const sceneCommitted = vi.fn()
    const unsubscribe = subscribeSceneCommitted(sceneCommitted)
    listeners.get('micro-app:designer-project-1:scene-committed')?.({
      projectId: 'project-1',
      target: '2d',
      sceneId: 'overview',
      revision: 3,
    })
    expect(sceneCommitted).toHaveBeenCalledWith({
      projectId: 'project-1',
      target: '2d',
      sceneId: 'overview',
      revision: 3,
    })
    unsubscribe()
  })

  it('保存宿主注入的编辑代际并将失效事件交还宿主处理', () => {
    const onAuthoringStale = vi.fn()
    applyMicroAppContext({
      projectId: 'project-1',
      authoringEpoch: ' epoch-7 ',
      onAuthoringStale,
    })

    expect(getCurrentAuthoringEpoch()).toBe('epoch-7')
    reportAuthoringStale({ projectId: 'project-1', currentAuthoringEpoch: 'epoch-8' })
    expect(onAuthoringStale).toHaveBeenCalledWith({
      projectId: 'project-1',
      currentAuthoringEpoch: 'epoch-8',
    })
  })
})
