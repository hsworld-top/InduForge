import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import DesignerWorkspaceView from './DesignerWorkspaceView.vue'
import { contextPackApi } from './code/context-pack-api'
import { resolveWorkspaceState } from './code/workspace-runtime'
import { sceneContractApi } from './scene-contract-api'
import { requestWorkspaceOpen } from '@/runtime/wujie-context'

vi.mock('vue-router', () => ({
  useRoute: () => ({ meta: { project: { id: 'project-1' } } }),
}))

vi.mock('@/runtime/wujie-context', () => ({
  getCurrentProjectId: () => 'project-1',
  requestWorkspaceOpen: vi.fn(() => true),
}))

vi.mock('./code/workspace-runtime', () => ({ resolveWorkspaceState: vi.fn() }))

vi.mock('./code/context-pack-api', () => ({
  contextPackApi: { refresh: vi.fn() },
}))

vi.mock('./scene-contract-api', () => ({
  sceneContractApi: { list: vi.fn() },
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
  observe() {}
  disconnect() {}
}

describe('DesignerWorkspaceView', () => {
  beforeEach(() => {
    vi.stubGlobal('ResizeObserver', ResizeObserverStub)
    vi.mocked(resolveWorkspaceState).mockReset()
    vi.mocked(contextPackApi.refresh).mockReset()
    vi.mocked(sceneContractApi.list).mockReset()
    vi.mocked(requestWorkspaceOpen).mockReset()
    vi.mocked(requestWorkspaceOpen).mockReturnValue(true)
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: true,
        json: async () => ({
          code: 0,
          msg: 'ok',
          data: {
            status: 'running',
            ownership: 'managed',
            port: 5173,
            updatedAt: '2026-08-13T12:00:00Z',
            message: null,
          },
        }),
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
          route: '/line-overview',
          embedMode: 'both',
          inputs: [],
          events: [],
          commands: [],
          publicObjects: [],
          datapointRefs: ['line.speed'],
          contractVersion: '2d-contract-version',
        },
        {
          id: 'scene-3d',
          kind: '3d',
          name: '厂区场景',
          embedMode: 'embedded',
          contractVersion: '3d-contract-version',
        },
      ],
      contractVersion: 'scene-snapshot-version',
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
    expect(wrapper.get('iframe[title="工程开发工作台"]').attributes('src')).toBe(
      'https://code.workspace.test/',
    )
    expect(wrapper.get('iframe[title="Vite 实时预览"]').attributes('src')).toBe(
      'https://preview.workspace.test/',
    )
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

  it('2D 和 3D 页面展示场景契约卡片，并通过宿主打开对应编辑器', async () => {
    const wrapper = mount(DesignerWorkspaceView)
    await flushPromises()

    await wrapper.get('button[aria-label="2D"]').trigger('click')
    expect(wrapper.get('.workbench-view.active').text()).toContain('产线总览')
    expect(wrapper.get('.workbench-view.active').text()).toContain('/line-overview')
    await wrapper.get('.workbench-view.active .scene-editor-button').trigger('click')
    expect(requestWorkspaceOpen).toHaveBeenCalledWith('2d')

    await wrapper.get('button[aria-label="3D"]').trigger('click')
    expect(wrapper.get('.workbench-view.active').text()).toContain('厂区场景')
    await wrapper.get('.workbench-view.active .scene-editor-button').trigger('click')
    expect(requestWorkspaceOpen).toHaveBeenCalledWith('3d')
  })
})
