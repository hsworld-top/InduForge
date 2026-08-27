import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import PreviewPanel from './PreviewPanel.vue'

const { viewerSessionMock, ioMock, socketEmitMock, socketHandlers } = vi.hoisted(() => ({
  viewerSessionMock: vi.fn(),
  ioMock: vi.fn(),
  socketEmitMock: vi.fn(),
  socketHandlers: new Map<string, (...args: unknown[]) => void>(),
}))
vi.mock('./scene-contract-api', () => ({
  sceneContractApi: { viewerSession: viewerSessionMock },
}))
vi.mock('socket.io-client', () => ({ io: ioMock }))

class ResizeObserverStub {
  observe() {}
  disconnect() {}
}

const runningState = {
  code: 0,
  msg: 'ok',
  data: {
    status: 'running',
    ownership: 'managed',
    port: 5173,
    updatedAt: '2026-08-13T12:00:00Z',
    message: null,
  },
}

function installFrameWindow(frame: HTMLIFrameElement) {
  const postMessage = vi.fn()
  const contentWindow = { postMessage } as unknown as Window
  Object.defineProperty(frame, 'contentWindow', { configurable: true, value: contentWindow })
  return { contentWindow, postMessage }
}

describe('PreviewPanel', () => {
  beforeEach(() => {
    vi.stubGlobal('ResizeObserver', ResizeObserverStub)
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, json: async () => runningState }))
    viewerSessionMock.mockReset()
    viewerSessionMock.mockResolvedValue({
      sessionId: 'viewer-1',
      sceneId: 'scene-1',
      kind: '2d',
      revision: 3,
      contract: {},
      expiresAt: '2026-08-18T08:00:00Z',
      url: '/designer/scene-studio/display.html?viewerSessionId=viewer-1',
    })
    socketHandlers.clear()
    socketEmitMock.mockReset()
    socketEmitMock.mockImplementation((event: string, payload: { requestId?: string; path?: string }) => {
      if (!event.startsWith('datapoint:')) return
      queueMicrotask(() => socketHandlers.get('response')?.({
        requestId: payload.requestId,
        success: true,
        result: { path: payload.path },
      }))
    })
    ioMock.mockReset()
    ioMock.mockReturnValue({
      connected: true,
      on: vi.fn((event: string, handler: (...args: unknown[]) => void) => {
        socketHandlers.set(event, handler)
      }),
      once: vi.fn((event: string, handler: (...args: unknown[]) => void) => {
        if (event === 'connect') queueMicrotask(handler)
      }),
      emit: socketEmitMock,
      disconnect: vi.fn(),
    })
  })

  it('设备菜单切换尺寸时不重建预览 iframe', async () => {
    const wrapper = mount(PreviewPanel, {
      props: {
        projectId: 'project-1',
        previewUrl: 'https://preview.workspace.test/',
        controlUrl: 'https://control.workspace.test/',
        active: true,
      },
    })
    await flushPromises()
    const frame = wrapper.get('iframe[title="Vite 实时预览"]').element

    await wrapper.get('button[aria-label="切换预览设备"]').trigger('click')
    expect(wrapper.findAll('button[role="menuitemradio"]').map((item) => item.text())).toEqual([
      '网页填满可用区域',
      '平板768 × 1024',
      '手机390 × 844',
    ])
    await wrapper.get('button[role="menuitemradio"]:nth-child(3)').trigger('click')

    expect(wrapper.get('iframe[title="Vite 实时预览"]').element).toBe(frame)
    expect(wrapper.get('.preview-canvas').classes()).toContain('device-mobile')
  })

  it('刷新只重建预览 iframe', async () => {
    const wrapper = mount(PreviewPanel, {
      props: {
        projectId: 'project-1',
        previewUrl: 'https://preview.workspace.test/',
        controlUrl: 'https://control.workspace.test/',
        active: true,
      },
    })
    await flushPromises()
    const frame = wrapper.get('iframe[title="Vite 实时预览"]').element

    await wrapper.get('button[aria-label="刷新预览"]').trigger('click')

    expect(wrapper.get('iframe[title="Vite 实时预览"]').element).not.toBe(frame)
  })

  it('新标签打开受控 Designer 预览宿主', async () => {
    const open = vi.spyOn(window, 'open').mockImplementation(() => null)
    const wrapper = mount(PreviewPanel, {
      props: {
        projectId: 'project-1',
        previewUrl: 'https://preview.workspace.test/project/',
        controlUrl: 'https://control.workspace.test/',
        active: true,
      },
    })
    await flushPromises()

    await wrapper.get('button[aria-label="在新窗口打开"]').trigger('click')

    expect(open).toHaveBeenCalledWith(
      'http://127.0.0.1:3000/designer/preview?projectId=project-1',
      '_blank',
      'noopener,noreferrer',
    )
  })

  it('停止操作调用控制接口并展示停止状态', async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce({ ok: true, json: async () => runningState })
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          ...runningState,
          data: { ...runningState.data, status: 'stopped', ownership: null },
        }),
      })
    vi.stubGlobal('fetch', fetchMock)
    const wrapper = mount(PreviewPanel, {
      props: {
        projectId: 'project-1',
        previewUrl: 'https://preview.workspace.test/',
        controlUrl: 'https://control.workspace.test/',
        active: true,
      },
    })
    await flushPromises()

    await wrapper.get('.command-button').trigger('click')
    await flushPromises()

    expect(fetchMock).toHaveBeenNthCalledWith(
      2,
      new URL('https://control.workspace.test/api/v1/preview/stop'),
      expect.objectContaining({ method: 'POST' }),
    )
    expect(wrapper.text()).toContain('预览服务已停止')
  })

  it('只接受 Vite iframe 的同源版本化 Viewer 请求并忽略页面工程参数', async () => {
    const wrapper = mount(PreviewPanel, {
      props: {
        projectId: 'project-1',
        previewUrl: 'https://preview.workspace.test/',
        controlUrl: 'https://control.workspace.test/',
        active: true,
      },
    })
    await flushPromises()
    const frame = wrapper.get('iframe[title="Vite 实时预览"]').element as HTMLIFrameElement
    const { contentWindow, postMessage } = installFrameWindow(frame)

    window.dispatchEvent(new MessageEvent('message', {
      source: contentWindow,
      origin: 'https://preview.workspace.test',
      data: {
        channel: 'induforge-preview-runtime',
        version: 1,
        type: 'CREATE_SCENE_VIEWER',
        requestId: 'request-1',
        projectId: 'forged-project',
        sceneId: 'scene-1',
        kind: '2d',
      },
    }))
    await flushPromises()

    expect(viewerSessionMock).toHaveBeenCalledWith('project-1', '2d', 'scene-1')
    expect(postMessage).toHaveBeenCalledWith(
      expect.objectContaining({
        type: 'SCENE_VIEWER_RESULT',
        requestId: 'request-1',
        data: expect.objectContaining({
          url: 'http://127.0.0.1:3000/designer/scene-studio/display.html?viewerSessionId=viewer-1',
        }),
      }),
      'https://preview.workspace.test',
    )
  })

  it('拒绝伪造 source、origin 和空请求 ID', async () => {
    const wrapper = mount(PreviewPanel, {
      props: {
        projectId: 'project-1',
        previewUrl: 'https://preview.workspace.test/',
        controlUrl: 'https://control.workspace.test/',
        active: true,
      },
    })
    await flushPromises()
    const frame = wrapper.get('iframe[title="Vite 实时预览"]').element as HTMLIFrameElement
    const { contentWindow } = installFrameWindow(frame)
    const base = {
      channel: 'induforge-preview-runtime',
      version: 1,
      type: 'CREATE_SCENE_VIEWER',
      requestId: 'request-1',
      sceneId: 'scene-1',
      kind: '2d',
    }

    window.dispatchEvent(new MessageEvent('message', { source: window, origin: 'https://preview.workspace.test', data: base }))
    window.dispatchEvent(new MessageEvent('message', { source: contentWindow, origin: 'https://evil.test', data: base }))
    window.dispatchEvent(new MessageEvent('message', { source: contentWindow, origin: 'https://preview.workspace.test', data: { ...base, requestId: '' } }))
    await flushPromises()

    expect(viewerSessionMock).not.toHaveBeenCalled()
  })

  it('向页面保留 Viewer API 的统一错误码和请求 ID', async () => {
    const error = Object.assign(new Error('场景尚无已提交版本'), {
      isBusinessError: true,
      code: 26001,
      reqId: 'backend-request-1',
    })
    viewerSessionMock.mockRejectedValue(error)
    const wrapper = mount(PreviewPanel, {
      props: {
        projectId: 'project-1',
        previewUrl: 'https://preview.workspace.test/',
        controlUrl: 'https://control.workspace.test/',
        active: true,
      },
    })
    await flushPromises()
    const frame = wrapper.get('iframe[title="Vite 实时预览"]').element as HTMLIFrameElement
    const { contentWindow, postMessage } = installFrameWindow(frame)
    window.dispatchEvent(new MessageEvent('message', {
      source: contentWindow,
      origin: 'https://preview.workspace.test',
      data: {
        channel: 'induforge-preview-runtime',
        version: 1,
        type: 'CREATE_SCENE_VIEWER',
        requestId: 'request-2',
        sceneId: 'scene-1',
        kind: '2d',
      },
    }))
    await flushPromises()

    expect(postMessage).toHaveBeenCalledWith(
      expect.objectContaining({
        error: { code: 26001, msg: '场景尚无已提交版本', reqId: 'backend-request-1' },
      }),
      'https://preview.workspace.test',
    )
  })

  it('通过页面运行时桥接读取数据点当前值', async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce({ ok: true, json: async () => runningState })
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          code: 0,
          msg: 'ok',
          data: {
            values: {
              'db.IF关系库.demo.temperature': { value: 26.5, quality: 'good' },
            },
          },
        }),
      })
    vi.stubGlobal('fetch', fetchMock)
    const wrapper = mount(PreviewPanel, {
      props: {
        projectId: 'project-1',
        previewUrl: 'https://preview.workspace.test/',
        controlUrl: 'https://control.workspace.test/',
        active: true,
      },
    })
    await flushPromises()
    const frame = wrapper.get('iframe[title="Vite 实时预览"]').element as HTMLIFrameElement
    const { contentWindow, postMessage } = installFrameWindow(frame)

    window.dispatchEvent(new MessageEvent('message', {
      source: contentWindow,
      origin: 'https://preview.workspace.test',
      data: {
        channel: 'induforge-page-runtime',
        version: 1,
        type: 'REQUEST',
        requestId: 'runtime-read-1',
        domain: 'point',
        operation: 'read',
        path: 'db.IF关系库.demo.temperature',
      },
    }))
    await flushPromises()

    expect(fetchMock).toHaveBeenNthCalledWith(
      2,
      '/api/v1/data/projects/project-1/datapoints/values',
      expect.objectContaining({ method: 'POST' }),
    )
    expect(postMessage).toHaveBeenCalledWith(
      expect.objectContaining({
        type: 'RESULT',
        requestId: 'runtime-read-1',
        result: { code: 0, msg: 'ok', data: { value: 26.5, quality: 'good' } },
      }),
      'https://preview.workspace.test',
    )
  })

  it('页面订阅复用数据预览 Socket 并转发变化事件', async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce({ ok: true, json: async () => runningState })
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({ code: 0, msg: 'ok', data: { sessionId: 'preview-session-1' } }),
      })
    vi.stubGlobal('fetch', fetchMock)
    const wrapper = mount(PreviewPanel, {
      props: {
        projectId: 'project-1',
        previewUrl: 'https://preview.workspace.test/',
        controlUrl: 'https://control.workspace.test/',
        active: true,
      },
    })
    await flushPromises()
    const frame = wrapper.get('iframe[title="Vite 实时预览"]').element as HTMLIFrameElement
    const { contentWindow, postMessage } = installFrameWindow(frame)

    window.dispatchEvent(new MessageEvent('message', {
      source: contentWindow,
      origin: 'https://preview.workspace.test',
      data: {
        channel: 'induforge-page-runtime',
        version: 1,
        type: 'REQUEST',
        requestId: 'runtime-sub-1',
        domain: 'point',
        operation: 'subscribe',
        path: 'mqtt.IF消息库.demo_line_events',
        subscriptionId: 'point-subscription-1',
      },
    }))
    await flushPromises()

    expect(ioMock).toHaveBeenCalledWith(
      window.location.origin,
      expect.objectContaining({
        auth: { projectId: 'project-1', previewSessionId: 'preview-session-1' },
      }),
    )
    expect(socketEmitMock).toHaveBeenCalledWith(
      'datapoint:subscribe',
      expect.objectContaining({ path: 'mqtt.IF消息库.demo_line_events' }),
    )

    socketHandlers.get('datapoint:value')?.({
      path: 'mqtt.IF消息库.demo_line_events',
      value: { event: 'operator_test' },
    })

    expect(postMessage).toHaveBeenCalledWith(
      expect.objectContaining({
        type: 'EVENT',
        subscriptionId: 'point-subscription-1',
        data: expect.objectContaining({ value: { event: 'operator_test' } }),
      }),
      'https://preview.workspace.test',
    )
  })
})
