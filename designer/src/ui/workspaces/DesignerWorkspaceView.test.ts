import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import DesignerWorkspaceView from './DesignerWorkspaceView.vue'
import { contextPackApi } from './code/context-pack-api'
import { resolveWorkspaceState } from './code/workspace-runtime'
import { sceneContractApi } from './scene-contract-api'
import { requestWorkspaceClose, requestWorkspaceOpen } from '@/runtime/wujie-context'
import { getEditorUiStore } from '@/stores/editor-ui-store'

const runtimeContextMock = vi.hoisted(() => ({
  sceneCommittedListener: null as null | (() => void),
  projectId: 'project-1',
}))

const codeWorkspaceApiMock = vi.hoisted(() => ({
  get: vi.fn(),
  rebuild: vi.fn(),
}))

vi.mock('vue-router', () => ({
  useRoute: () => ({ meta: { project: { id: runtimeContextMock.projectId } } }),
}))

vi.mock('@/runtime/wujie-context', () => ({
  getCurrentProjectId: () => runtimeContextMock.projectId,
  requestWorkspaceOpen: vi.fn(() => true),
  requestWorkspaceClose: vi.fn(() => true),
  subscribeSceneCommitted: vi.fn((listener: () => void) => {
    runtimeContextMock.sceneCommittedListener = listener
    return () => {
      runtimeContextMock.sceneCommittedListener = null
    }
  }),
}))

vi.mock('./code/workspace-runtime', () => ({ resolveWorkspaceState: vi.fn() }))

vi.mock('./code/code-workspace-api', () => ({ codeWorkspaceApi: codeWorkspaceApiMock }))

vi.mock('./code/context-pack-api', () => ({
  contextPackApi: { refresh: vi.fn() },
}))

vi.mock('./scene-contract-api', () => ({
  sceneContractApi: { list: vi.fn(), create: vi.fn(), update: vi.fn(), remove: vi.fn() },
}))

const workspace = {
  status: 'running' as const,
  containerName: 'induforge-code-project-1',
  onlineUsers: [],
  services: {
    ai: { url: 'https://ai.workspace.test/', hostPort: null },
    code: { url: 'https://code.workspace.test/', hostPort: null },
    preview: { url: 'https://preview.workspace.test/', hostPort: null },
    previewControl: { url: 'https://control.workspace.test/', hostPort: null },
  },
}

class ResizeObserverStub {
  static workspace: ResizeObserverStub | null = null

  constructor(
    private readonly callback: (
      entries: Array<{ contentRect: { width: number } }>,
      observer: ResizeObserverStub,
    ) => void,
  ) {}

  observe(target: Element) {
    if (target.classList.contains('workbench-stage')) ResizeObserverStub.workspace = this
  }
  unobserve() {}
  disconnect() {}

  emit(width: number) {
    this.callback([{ contentRect: { width } }], this)
  }
}

let workspaceInitializationStatus: 'uninitialized' | 'initialized'
let initializeTemplateId: string | null

describe('DesignerWorkspaceView', () => {
  beforeEach(() => {
    vi.stubGlobal('ResizeObserver', ResizeObserverStub)
    ResizeObserverStub.workspace = null
    localStorage.clear()
    vi.mocked(resolveWorkspaceState).mockReset()
    codeWorkspaceApiMock.get.mockReset()
    codeWorkspaceApiMock.rebuild.mockReset()
    vi.mocked(contextPackApi.refresh).mockReset()
    vi.mocked(sceneContractApi.list).mockReset()
    vi.mocked(sceneContractApi.create).mockReset()
    vi.mocked(sceneContractApi.remove).mockReset()
    vi.mocked(requestWorkspaceOpen).mockReset()
    vi.mocked(requestWorkspaceOpen).mockReturnValue(true)
    vi.mocked(requestWorkspaceClose).mockReset()
    vi.mocked(requestWorkspaceClose).mockReturnValue(true)
    runtimeContextMock.sceneCommittedListener = null
    runtimeContextMock.projectId = 'project-1'
    getEditorUiStore().initFromRuntime({ theme: 'light', locale: 'zh' })
    workspaceInitializationStatus = 'initialized'
    initializeTemplateId = null
    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: RequestInfo | URL, options?: RequestInit) => {
        const url = new URL(String(input))
        let data: unknown
        if (url.pathname === '/api/v1/workspace/status') {
          data = {
            status: workspaceInitializationStatus,
            templateId: workspaceInitializationStatus === 'initialized' ? 'vite-vue-ts' : null,
            initializedAt:
              workspaceInitializationStatus === 'initialized' ? '2026-08-14T02:00:00Z' : null,
            message: null,
          }
        } else if (url.pathname === '/api/v1/workspace/templates') {
          data = {
            version: 1,
            generator: 'create-vite@9.1.2',
            templates: [
              {
                id: 'vite-vue-js',
                name: 'Vue + JavaScript',
                description: 'Vite 官方 Vue JavaScript 模板',
                framework: 'vue',
                language: 'javascript',
              },
              {
                id: 'vite-vue-ts',
                name: 'Vue + TypeScript',
                description: 'Vite 官方 Vue TypeScript 模板',
                framework: 'vue',
                language: 'typescript',
              },
              {
                id: 'vite-react-js',
                name: 'React + JavaScript',
                description: 'Vite 官方 React JavaScript 模板',
                framework: 'react',
                language: 'javascript',
              },
              {
                id: 'vite-react-ts',
                name: 'React + TypeScript',
                description: 'Vite 官方 React TypeScript 模板',
                framework: 'react',
                language: 'typescript',
              },
            ],
          }
        } else if (url.pathname === '/api/v1/workspace/initialize') {
          initializeTemplateId = JSON.parse(String(options?.body)).templateId
          workspaceInitializationStatus = 'initialized'
          data = {
            workspace: {
              status: 'initialized',
              templateId: initializeTemplateId,
              initializedAt: '2026-08-14T02:00:00Z',
              message: null,
            },
            preview: {
              status: 'running',
              ownership: 'managed',
              port: 5173,
              updatedAt: '2026-08-14T02:00:00Z',
              message: null,
            },
          }
        } else {
          data = {
            status: 'running',
            ownership: 'managed',
            port: 5173,
            updatedAt: '2026-08-13T12:00:00Z',
            message: null,
          }
        }
        return {
          ok: true,
          status: 200,
          json: async () => ({ code: 0, msg: 'ok', data }),
        } as Response
      }),
    )
    vi.mocked(resolveWorkspaceState).mockResolvedValue(workspace)
    vi.mocked(contextPackApi.refresh).mockResolvedValue({
      contractVersion: 'context-v1',
      pointCount: 12,
      roleCount: 0,
      updatedAt: '2026-08-13T12:00:00Z',
    })
    vi.mocked(sceneContractApi.list).mockResolvedValue({
      contracts: [
        {
          id: 'scene-2d',
          kind: '2d',
          name: '产线总览',
          description: '生产线实时状态与关键数据点。',
          parameters: [],
          events: [],
          commands: [],
          datapointRefs: ['line.speed'],
          contractVersion: '2d-contract-version',
        },
        {
          id: 'scene-3d',
          kind: '3d',
          name: '厂区场景',
          parameters: [],
          events: [],
          commands: [],
          contractVersion: '3d-contract-version',
        },
      ],
      contractVersion: 'scene-snapshot-version',
    })
    vi.mocked(sceneContractApi.create).mockResolvedValue({
      id: 'new-scene',
      kind: '2d',
      name: '新场景',
      parameters: [],
      events: [],
      commands: [],
      contractVersion: '0',
    })
  })

  it('中间菜单提供页面、2D、3D 和编辑器，页面与编辑器 iframe 常驻', async () => {
    const wrapper = mount(DesignerWorkspaceView)
    await flushPromises()

    const menuLabels = wrapper
      .findAll('.workbench-menu button')
      .map((button) => button.text().trim())
    expect(menuLabels).toEqual(['页面', '2D', '3D', '编辑器'])
    expect(wrapper.text()).not.toContain('源码')
    expect(wrapper.findAll('iframe')).toHaveLength(3)
    expect(wrapper.get('iframe[title="Pi Web AI 对话"]').attributes('src')).toBe(
      'https://ai.workspace.test/?induforgeProjectId=project-1',
    )
    expect(wrapper.get('iframe[title="工程开发工作台"]').attributes('src')).toBe(
      'https://code.workspace.test/',
    )
    expect(wrapper.get('iframe[title="Vite 实时预览"]').attributes('src')).toBe(
      'https://preview.workspace.test/',
    )
    expect(wrapper.find('.rebuild-workspace-button').exists()).toBe(false)
  })

  it('空工作区先选择官方模板，初始化完成后再创建工作台 iframe', async () => {
    workspaceInitializationStatus = 'uninitialized'
    const wrapper = mount(DesignerWorkspaceView)
    await flushPromises()

    expect(wrapper.findAll('iframe')).toHaveLength(0)
    expect(wrapper.findAll('.template-option')).toHaveLength(4)
    expect(wrapper.text()).toContain('Vue + JavaScript')
    expect(wrapper.text()).toContain('React + TypeScript')

    await wrapper
      .findAll('.template-option')
      .find((option) => option.text().includes('React + TypeScript'))!
      .trigger('click')
    await wrapper.get('.initialize-button').trigger('click')
    await flushPromises()

    expect(initializeTemplateId).toBe('vite-react-ts')
    expect(wrapper.findAll('iframe')).toHaveLength(3)
  })

  it('内置教程工程首次打开时自动装载 Vue JavaScript 示例', async () => {
    runtimeContextMock.projectId = '00000000-0000-4000-8000-000000000001'
    workspaceInitializationStatus = 'uninitialized'

    const wrapper = mount(DesignerWorkspaceView)
    await flushPromises()

    expect(initializeTemplateId).toBe('vite-vue-js')
    expect(wrapper.findAll('iframe')).toHaveLength(3)
    expect(wrapper.text()).not.toContain('选择工程模板')
  })

  it('仅工作区异常时显示重新构建入口，确认后轮询运行态并进入工作台', async () => {
    const brokenWorkspace = {
      ...workspace,
      status: 'error' as const,
      services: {
        ai: { url: null, hostPort: null },
        code: { url: null, hostPort: null },
        preview: { url: null, hostPort: null },
        previewControl: { url: null, hostPort: null },
      },
    }
    vi.mocked(resolveWorkspaceState)
      .mockResolvedValueOnce(brokenWorkspace)
      .mockResolvedValue(workspace)
    let finishRebuild: (value: typeof workspace) => void = () => {}
    codeWorkspaceApiMock.rebuild.mockImplementation(
      () =>
        new Promise<typeof workspace>((resolve) => {
          finishRebuild = resolve
        }),
    )
    vi.spyOn(window, 'confirm').mockReturnValue(true)

    const wrapper = mount(DesignerWorkspaceView)
    await flushPromises()

    expect(wrapper.get('.rebuild-workspace-button').text()).toBe('重新构建工作区')

    const rebuildClick = wrapper.get('.rebuild-workspace-button').trigger('click')
    await flushPromises()

    expect(wrapper.get('.rebuild-workspace-button').text()).toBe('正在重新构建')
    expect((wrapper.get('.rebuild-workspace-button').element as HTMLButtonElement).disabled).toBe(
      true,
    )
    finishRebuild(workspace)
    await rebuildClick
    await flushPromises()

    expect(window.confirm).toHaveBeenCalledWith(
      '重新构建开发工作区会停止并重新创建开发服务，不会删除工程设计、页面源码或已发布版本。是否继续？',
    )
    expect(codeWorkspaceApiMock.rebuild).toHaveBeenCalledWith('project-1')
    expect(wrapper.find('.rebuild-workspace-button').exists()).toBe(false)
    expect(wrapper.findAll('iframe')).toHaveLength(3)
  })

  it('Pi iframe 加载、READY 与 IDE 设置变化时发送严格宿主上下文', async () => {
    const wrapper = mount(DesignerWorkspaceView)
    await flushPromises()
    const frame = wrapper.get('iframe[title="Pi Web AI 对话"]')
    const postMessage = vi.fn()
    const frameWindow = { postMessage } as unknown as Window
    Object.defineProperty(frame.element, 'contentWindow', {
      configurable: true,
      value: frameWindow,
    })

    await frame.trigger('load')
    expect(postMessage).toHaveBeenLastCalledWith(
      {
        type: 'INDUFORGE_PI_CONTEXT',
        version: 1,
        projectId: 'project-1',
        workspaceRoot: '/workspace',
        locale: 'zh',
        theme: 'light',
      },
      'https://ai.workspace.test',
    )

    postMessage.mockClear()
    window.dispatchEvent(
      new MessageEvent('message', {
        data: { type: 'INDUFORGE_PI_READY', version: 1, projectId: 'project-1' },
        origin: 'https://ai.workspace.test',
        source: frameWindow,
      }),
    )
    expect(postMessage).toHaveBeenCalledTimes(1)

    postMessage.mockClear()
    getEditorUiStore().setTheme('dark')
    getEditorUiStore().setLocale('en')
    await flushPromises()
    expect(postMessage).toHaveBeenLastCalledWith(
      expect.objectContaining({ theme: 'dark', locale: 'en' }),
      'https://ai.workspace.test',
    )
  })

  it('忽略伪造来源、错误工程和错误消息结构', async () => {
    const wrapper = mount(DesignerWorkspaceView)
    await flushPromises()
    const frame = wrapper.get('iframe[title="Pi Web AI 对话"]')
    const postMessage = vi.fn()
    const frameWindow = { postMessage } as unknown as Window
    Object.defineProperty(frame.element, 'contentWindow', {
      configurable: true,
      value: frameWindow,
    })
    await frame.trigger('load')
    postMessage.mockClear()

    for (const message of [
      new MessageEvent('message', {
        data: { type: 'INDUFORGE_PI_READY', version: 1, projectId: 'project-1' },
        origin: 'https://forged.test',
        source: frameWindow,
      }),
      new MessageEvent('message', {
        data: { type: 'INDUFORGE_PI_READY', version: 1, projectId: 'other-project' },
        origin: 'https://ai.workspace.test',
        source: frameWindow,
      }),
      new MessageEvent('message', {
        data: { type: 'INDUFORGE_PI_READY', version: 2, projectId: 'project-1' },
        origin: 'https://ai.workspace.test',
        source: frameWindow,
      }),
    ]) {
      window.dispatchEvent(message)
    }

    expect(postMessage).not.toHaveBeenCalled()
  })

  it('预览和编辑器切换不销毁或修改 iframe', async () => {
    const wrapper = mount(DesignerWorkspaceView)
    await flushPromises()
    const preview = wrapper.get('iframe[title="Vite 实时预览"]').element
    const editor = wrapper.get('iframe[title="工程开发工作台"]').element

    await wrapper.get('button[aria-label="编辑器"]').trigger('click')

    expect(wrapper.get('iframe[title="Vite 实时预览"]').element).toBe(preview)
    expect(wrapper.get('iframe[title="工程开发工作台"]').element).toBe(editor)
    expect(wrapper.get('.code-server-shell').classes()).toContain('active')
  })

  it('折叠后只保留窄菜单，点击菜单恢复工作台且 iframe 不销毁', async () => {
    localStorage.setItem(
      'designer:workspace-layout:project-1',
      JSON.stringify({
        version: 1,
        aiPaneWidth: 1348,
        lastExpandedAiPaneWidth: 720,
        workbenchCollapsed: true,
      }),
    )
    const wrapper = mount(DesignerWorkspaceView)
    await flushPromises()
    ResizeObserverStub.workspace?.emit(1400)
    await wrapper.vm.$nextTick()

    const preview = wrapper.get('iframe[title="Vite 实时预览"]').element
    expect(wrapper.get('.workbench-pane').classes()).toContain('collapsed')
    expect(wrapper.get('.workbench-stage').attributes('style')).toContain('1348px')

    await wrapper.get('button[aria-label="2D"]').trigger('click')

    expect(wrapper.get('.workbench-pane').classes()).not.toContain('collapsed')
    expect(wrapper.get('.workbench-stage').attributes('style')).toContain('720px')
    expect(wrapper.get('iframe[title="Vite 实时预览"]').element).toBe(preview)
    expect(wrapper.get('.workbench-view.active').text()).toContain('产线总览')
  })

  it('2D 和 3D 页面展示场景契约卡片，并通过宿主打开对应编辑器', async () => {
    const wrapper = mount(DesignerWorkspaceView)
    await flushPromises()

    await wrapper.get('button[aria-label="2D"]').trigger('click')
    expect(wrapper.get('.workbench-view.active').text()).toContain('产线总览')
    expect(wrapper.get('.workbench-view.active').text()).not.toContain('公开能力')
    expect(wrapper.get('.workbench-view.active').text()).not.toContain('独立 / 嵌入')
    expect(wrapper.get('.workbench-view.active').text()).toContain('关联数据点 1')
    expect(wrapper.find('button[aria-label="预览已提交版本"]').exists()).toBe(false)
    await wrapper.get('.workbench-view.active button[aria-label="打开编辑器"]').trigger('click')
    expect(requestWorkspaceOpen).toHaveBeenCalledWith('2d', 'scene-2d', '产线总览')

    await wrapper.get('button[aria-label="3D"]').trigger('click')
    expect(wrapper.get('.workbench-view.active').text()).toContain('厂区场景')
    await wrapper.get('.workbench-view.active button[aria-label="打开编辑器"]').trigger('click')
    expect(requestWorkspaceOpen).toHaveBeenCalledWith('3d', 'scene-3d', '厂区场景')
  })

  it('场景设置展示结构化交互和 Provider 提取的数据点', async () => {
    const wrapper = mount(DesignerWorkspaceView)
    await flushPromises()

    await wrapper.get('button[aria-label="2D"]').trigger('click')
    await wrapper.get('.workbench-view.active button[aria-label="场景设置"]').trigger('click')
    await flushPromises()

    const panel = wrapper.get('.scene-contract-panel')
    expect(panel.get('h3').text()).toBe('2D 场景设置')
    expect((panel.get('input[placeholder="产线总览"]').element as HTMLInputElement).value).toBe(
      '产线总览',
    )
    expect(panel.text()).toContain('关联数据点')
    expect(panel.text()).toContain('参数')
    expect(panel.text()).toContain('事件')
    expect(panel.text()).toContain('命令')
    expect(panel.text()).toContain('line.speed')
    expect(panel.text()).not.toContain('页面集成')
    expect(panel.text()).not.toContain('权限引用')
    expect(panel.get('footer > span').text()).toBe('')
  })

  it('新建场景使用精简表单，创建成功后直接打开对应 HT 编辑器', async () => {
    vi.mocked(sceneContractApi.create).mockResolvedValue({
      id: 'new-scene',
      kind: '2d',
      name: '主产线',
      parameters: [],
      events: [],
      commands: [],
      contractVersion: '0',
    })
    const wrapper = mount(DesignerWorkspaceView)
    await flushPromises()

    await wrapper.get('button[aria-label="2D"]').trigger('click')
    await wrapper.get('.workbench-view.active .scene-editor-button').trigger('click')

    expect(wrapper.get('[role="dialog"]').text()).toContain('新建2D 画面')
    expect(wrapper.get('[role="dialog"]').text()).not.toContain('场景 ID')
    expect(wrapper.find('.scene-contract-panel').exists()).toBe(false)
    expect(wrapper.text()).not.toContain('+ 新建接口')

    await wrapper.get('input[placeholder="产线总览"]').setValue('主产线')
    await wrapper.get('.scene-create-dialog form').trigger('submit')
    await flushPromises()

    expect(sceneContractApi.create).toHaveBeenCalledWith('project-1', {
      name: '主产线',
      kind: '2d',
    })
    expect(requestWorkspaceOpen).toHaveBeenCalledWith('2d', 'new-scene', '主产线')
    expect(wrapper.find('[role="dialog"]').exists()).toBe(false)
  })

  it('场景名称为空时留在新建表单且不请求后端', async () => {
    const wrapper = mount(DesignerWorkspaceView)
    await flushPromises()

    await wrapper.get('button[aria-label="3D"]').trigger('click')
    await wrapper.get('.workbench-view.active .scene-editor-button').trigger('click')
    await wrapper.get('.scene-create-dialog form').trigger('submit')

    expect(wrapper.get('[role="alert"]').text()).toContain('场景名称')
    expect(sceneContractApi.create).not.toHaveBeenCalled()
  })

  it('删除场景成功后请求宿主关闭对应编辑器标签', async () => {
    vi.spyOn(window, 'confirm').mockReturnValue(true)
    vi.mocked(sceneContractApi.remove).mockResolvedValue({
      deleted: true,
      contextSync: { status: 'updated' },
    })
    const wrapper = mount(DesignerWorkspaceView)
    await flushPromises()

    await wrapper.get('button[aria-label="2D"]').trigger('click')
    await wrapper.get('.workbench-view.active button[aria-label="删除场景"]').trigger('click')
    await flushPromises()

    expect(sceneContractApi.remove).toHaveBeenCalledWith('project-1', '2d', 'scene-2d')
    expect(requestWorkspaceClose).toHaveBeenCalledWith('2d', 'scene-2d')
  })

  it('HT 提交 revision 后自动刷新场景状态', async () => {
    mount(DesignerWorkspaceView)
    await flushPromises()
    vi.mocked(sceneContractApi.list).mockClear()

    runtimeContextMock.sceneCommittedListener?.()
    await flushPromises()

    expect(sceneContractApi.list).toHaveBeenCalledWith('project-1')
  })
})
