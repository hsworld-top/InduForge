import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import PreviewPanel from './PreviewPanel.vue'

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

describe('PreviewPanel', () => {
  beforeEach(() => {
    vi.stubGlobal('ResizeObserver', ResizeObserverStub)
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, json: async () => runningState }))
  })

  it('设备菜单切换尺寸时不重建预览 iframe', async () => {
    const wrapper = mount(PreviewPanel, {
      props: {
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

  it('新标签直接打开工作空间返回的 Preview URL', async () => {
    const open = vi.spyOn(window, 'open').mockImplementation(() => null)
    const wrapper = mount(PreviewPanel, {
      props: {
        previewUrl: 'https://preview.workspace.test/project/',
        controlUrl: 'https://control.workspace.test/',
        active: true,
      },
    })
    await flushPromises()

    await wrapper.get('button[aria-label="在新窗口打开"]').trigger('click')

    expect(open).toHaveBeenCalledWith(
      'https://preview.workspace.test/project/',
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
})
