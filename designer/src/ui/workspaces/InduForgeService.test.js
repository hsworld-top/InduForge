import { beforeAll, beforeEach, describe, expect, it, vi } from 'vitest'

const fetchMock = vi.fn()

describe('InduForgeService', () => {
  beforeAll(async () => {
    window.fetch = fetchMock
    await import('../../../scripts/ht-editor/InduForgeService.js')
  })

  beforeEach(() => {
    fetchMock.mockReset()
    window.history.replaceState({}, '', '/ht-editor/index.html?sessionId=session-1')
    window.localStorage.clear()
    URL.createObjectURL = vi.fn(() => 'blob:design-export')
    URL.revokeObjectURL = vi.fn()
  })

  it('将 HT 工程资源地址改写为受控读取接口', () => {
    expect(window.InduForgeDesignFiles.resolveUrl('assets/pump.png')).toBe(
      '/api/v1/scene-editor-sessions/session-1/files/content?path=assets%2Fpump.png',
    )
    expect(window.InduForgeDesignFiles.resolveUrl('/session-1/displays/main.json')).toBe(
      '/api/v1/scene-editor-sessions/session-1/files/content?path=displays%2Fmain.json',
    )
    expect(window.InduForgeDesignFiles.resolveUrl('/other-session/displays/main.json')).toBe(
      '/other-session/displays/main.json',
    )
    expect(window.InduForgeDesignFiles.resolveUrl('custom/images/favicon.ico')).toBe(
      'custom/images/favicon.ico',
    )
  })

  it('没有编辑会话时不读取本地工程上下文', () => {
    window.history.replaceState({}, '', '/ht-editor/display.html')
    window.localStorage.setItem('project_id', JSON.stringify('project-2'))

    expect(window.InduForgeDesignFiles.resolveUrl('scenes/main.json')).toBe('scenes/main.json')
  })

  it('将 HT 原生根目录转换为后端逻辑目录并返回固定入口', async () => {
    fetchMock.mockResolvedValue({
      ok: true,
      json: async () => ({
        code: 0,
        data: {
          items: [{ name: 'main.json', path: 'displays/main.json', type: 'file' }],
          total: 1,
          draftVersion: 0,
        },
      }),
    })
    const service = new window.InduForgeService(vi.fn(), {})
    await service.ready
    fetchMock.mockClear()

    await expect(service.explore('/displays')).resolves.toEqual({
      'main.json': { fileType: 'display', fileImage: 'editor.display' },
    })

    const requestURL = new URL(fetchMock.mock.calls[0][0], window.location.origin)
    expect(requestURL.searchParams.get('directory')).toBe('displays')
  })

  it('通过 REST 命令接口保存设计文件', async () => {
    fetchMock.mockImplementation(async (url) => ({
      ok: true,
      json: async () =>
        url.includes('/files?')
          ? { code: 0, data: { items: [], total: 0, page: 1, limit: 200, draftVersion: 0 } }
          : url.includes('/commit')
            ? { code: 0, data: { revision: 1 } }
            : { code: 0, data: { draftVersion: 1 } },
    }))
    const handler = vi.fn()
    const callback = vi.fn()
    const service = new window.InduForgeService(handler, {})

    service.request('upload', { path: 'displays/main.json', content: '{}' }, callback)
    await vi.waitFor(() => expect(callback).toHaveBeenCalledWith(true))

    expect(fetchMock).toHaveBeenCalledWith(
      expect.stringContaining('/api/v1/scene-editor-sessions/session-1/files/content?'),
      expect.objectContaining({
        method: 'PUT',
      }),
    )
    const request = fetchMock.mock.calls[0][1]
    expect(request.credentials).toBe('same-origin')
    expect(handler).toHaveBeenCalledWith(expect.objectContaining({ type: 'sceneCommitted' }))
  })

  it('通过独立二进制接口导出 HT 资源', async () => {
    const click = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {})
    fetchMock.mockImplementation(async (url) =>
      url.includes('/files?')
        ? {
            ok: true,
            json: async () => ({
              code: 0,
              data: { items: [], total: 0, page: 1, limit: 200, draftVersion: 0 },
            }),
          }
        : {
            ok: true,
            headers: new Headers({ 'Content-Type': 'application/zip' }),
            blob: async () => new Blob(['zip']),
          },
    )
    const callback = vi.fn()
    const service = new window.InduForgeService(vi.fn(), {})

    service.request('export', ['scenes/main.json'], callback)
    await vi.waitFor(() => expect(callback).toHaveBeenCalledWith(true))

    expect(fetchMock).toHaveBeenCalledWith(
      '/api/v1/scene-editor-sessions/session-1/export',
      expect.objectContaining({
        method: 'POST',
      }),
    )
    expect(click).toHaveBeenCalled()
    click.mockRestore()
  })

  it('ZIP 导入冲突时触发 HT 原生确认事件', async () => {
    fetchMock.mockImplementation(async (url) => ({
      ok: true,
      json: async () =>
        url.includes('/files?')
          ? { code: 0, data: { items: [], total: 0, page: 1, limit: 200, draftVersion: 0 } }
          : { code: 0, data: { draftVersion: 1 } },
    }))
    const handler = vi.fn()
    const service = new window.InduForgeService(handler, {})

    service.request('upload', {
      path: 'imports/design.zip',
      content: 'data:application/zip;base64,AA==',
    })
    await vi.waitFor(() =>
      expect(fetchMock).toHaveBeenCalledWith(
        expect.stringContaining('/scene-editor-sessions/session-1/import?'),
        expect.objectContaining({ method: 'POST' }),
      ),
    )
  })

  it('导出业务失败时不下载 JSON 错误响应', async () => {
    const click = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {})
    fetchMock.mockImplementation(async (url) =>
      url.includes('/files?')
        ? {
            ok: true,
            json: async () => ({
              code: 0,
              data: { items: [], total: 0, page: 1, limit: 200, draftVersion: 0 },
            }),
          }
        : {
            ok: true,
            headers: new Headers({ 'Content-Type': 'application/json' }),
            json: async () => ({ code: 3001, msg: '设计文件不存在' }),
          },
    )
    const callback = vi.fn()
    const handler = vi.fn()
    const service = new window.InduForgeService(handler, {})

    service.request('export', ['scenes/missing.json'], callback)
    await vi.waitFor(() => expect(callback).toHaveBeenCalledWith(false))

    expect(click).not.toHaveBeenCalled()
    expect(handler).toHaveBeenCalledWith(
      expect.objectContaining({ type: 'response', cmd: 'export', message: '设计文件不存在' }),
    )
    click.mockRestore()
  })

  it('资源库按当前分类分页加载并使用乐观锁挂载资源', async () => {
    const asset = { assetId: 'asset-1', name: '泵', type: 'image', entryFile: 'pump.png' }
    fetchMock.mockImplementation(async (url, init) => {
      if (url.includes('/assets/actions')) {
        return { ok: true, json: async () => ({ code: 0, data: { draftVersion: 4 } }) }
      }
      return {
        ok: true,
        json: async () => ({
          code: 0,
          data: { items: [asset], bindings: [], total: 1, limit: 30, draftVersion: 3 },
        }),
      }
    })
    window.ht = {
      Tab: class {
        setName() {}
        setView(view) {
          this.view = view
        }
        getView() {
          return this.view
        }
      },
    }
    const tabs = []
    const nativeTabChanged = vi.fn()
    const selectTab = vi.fn()
    const tabView = {
      onTabChanged: nativeTabChanged,
      select: selectTab,
      getTabModel: () => ({
        add: (tab) => {
          tabs.push(tab)
          tabView.onTabChanged(null, tab)
        },
      }),
    }
    const editor = { leftTopTabView: tabView }

    const library = window.InduForgeAssets.mount(editor, '2d')
    await vi.waitFor(() =>
      expect(fetchMock).toHaveBeenCalledWith(
        expect.stringMatching(/\/assets\?.*type=image.*limit=30/),
        expect.objectContaining({ credentials: 'same-origin' }),
      ),
    )
    await vi.waitFor(() => expect(library.draftVersion).toBe(3))
    await library.assetAction('attach', asset)

    const actionRequest = fetchMock.mock.calls.find(([url]) => url.includes('/assets/actions'))
    expect(JSON.parse(actionRequest[1].body)).toEqual({
      action: 'attach',
      assetId: 'asset-1',
      baseDraftVersion: 3,
    })
    expect(tabs).toHaveLength(1)
    expect(selectTab).toHaveBeenCalledWith(tabs[0])
    expect(nativeTabChanged).not.toHaveBeenCalled()
  })

  it('Symbol 工作副本保存后提交新的内部代次', async () => {
    window.history.replaceState({}, '', '/ht-editor/index.html?assetSessionId=asset-session-1')
    fetchMock.mockImplementation(async (url) => {
      if (url.includes('/files?')) {
        return {
          ok: true,
          json: async () => ({ code: 0, data: { items: [], total: 0, draftVersion: 1 } }),
        }
      }
      if (url.includes('/commit')) {
        return {
          ok: true,
          json: async () => ({ code: 0, data: { assetId: 'asset-1', draftVersion: 2 } }),
        }
      }
      return { ok: true, json: async () => ({ code: 0, data: { draftVersion: 2 } }) }
    })
    const handler = vi.fn()
    const callback = vi.fn()
    const service = new window.InduForgeService(handler, {})

    service.request('upload', { path: 'symbols/pump.json', content: '{}' }, callback)
    await vi.waitFor(() => expect(callback).toHaveBeenCalledWith(true))

    expect(fetchMock).toHaveBeenCalledWith(
      expect.stringContaining('/api/v1/scene-asset-editor-sessions/asset-session-1/files/content?'),
      expect.objectContaining({ method: 'PUT' }),
    )
    expect(fetchMock).toHaveBeenCalledWith(
      '/api/v1/scene-asset-editor-sessions/asset-session-1/commit',
      expect.objectContaining({ method: 'POST' }),
    )
    expect(handler).toHaveBeenCalledWith(expect.objectContaining({ type: 'assetCommitted' }))
  })
})
