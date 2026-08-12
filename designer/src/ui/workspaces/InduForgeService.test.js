import { beforeAll, beforeEach, describe, expect, it, vi } from 'vitest'

const fetchMock = vi.fn()

describe('InduForgeService', () => {
  beforeAll(async () => {
    window.fetch = fetchMock
    await import('../../../scripts/ht-editor/InduForgeService.js')
  })

  beforeEach(() => {
    fetchMock.mockReset()
    window.history.replaceState({}, '', '/ht-editor/index.html?projectId=project-1')
    window.localStorage.clear()
    window.localStorage.setItem('auth_token', 'token-1')
    URL.createObjectURL = vi.fn(() => 'blob:design-export')
    URL.revokeObjectURL = vi.fn()
  })

  it('将 HT 工程资源地址改写为受控读取接口', () => {
    expect(window.InduForgeDesignFiles.resolveUrl('assets/pump.png')).toBe(
      '/api/v1/projects/project-1/design-files/content?path=assets%2Fpump.png',
    )
    expect(window.InduForgeDesignFiles.resolveUrl('custom/images/favicon.ico')).toBe(
      'custom/images/favicon.ico',
    )
  })

  it('没有查询参数时从父窗口或本地工程上下文读取工程 ID', () => {
    window.history.replaceState({}, '', '/ht-editor/display.html')
    window.localStorage.setItem('project_id', JSON.stringify('project-2'))

    expect(window.InduForgeDesignFiles.resolveUrl('scenes/main.json')).toBe(
      '/api/v1/projects/project-2/design-files/content?path=scenes%2Fmain.json',
    )
  })

  it('通过 REST 命令接口保存设计文件', async () => {
    fetchMock.mockResolvedValue({
      json: async () => ({ code: 0, data: { result: true, contextSync: { status: 'updated' } } }),
    })
    const handler = vi.fn()
    const callback = vi.fn()
    const service = new window.InduForgeService(handler, {})

    service.request('upload', { path: 'displays/main.json', content: '{}' }, callback)
    await vi.waitFor(() => expect(callback).toHaveBeenCalledWith(true))

    expect(fetchMock).toHaveBeenCalledWith(
      '/api/v1/projects/project-1/design-files/actions',
      expect.objectContaining({
        method: 'POST',
        headers: expect.any(Headers),
      }),
    )
    const request = fetchMock.mock.calls[0][1]
    expect(request.headers.get('Authorization')).toBe('Bearer token-1')
    expect(handler).toHaveBeenCalledWith(
      expect.objectContaining({ type: 'contextSync', message: '工程上下文已同步' }),
    )
  })

  it('通过独立二进制接口导出 HT 资源', async () => {
    const click = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {})
    fetchMock.mockResolvedValue({
      ok: true,
      headers: new Headers({
        'Content-Disposition': 'attachment; filename="scene.zip"',
        'Content-Type': 'application/zip',
      }),
      blob: async () => new Blob(['zip']),
    })
    const callback = vi.fn()
    const service = new window.InduForgeService(vi.fn(), {})

    service.request('export', ['scenes/main.json'], callback)
    await vi.waitFor(() => expect(callback).toHaveBeenCalledWith(true))

    expect(fetchMock).toHaveBeenCalledWith(
      '/api/v1/projects/project-1/design-files/export',
      expect.objectContaining({
        method: 'POST',
        body: JSON.stringify({ paths: ['scenes/main.json'] }),
      }),
    )
    expect(click).toHaveBeenCalled()
    click.mockRestore()
  })

  it('ZIP 导入冲突时触发 HT 原生确认事件', async () => {
    fetchMock.mockResolvedValue({
      json: async () => ({
        code: 0,
        data: {
          result: {
            importPending: true,
            path: 'import-token',
            conflicts: ['scenes/main.json'],
            roots: ['scenes'],
          },
        },
      }),
    })
    const handler = vi.fn()
    const service = new window.InduForgeService(handler, {})

    service.request('upload', {
      path: 'imports/design.zip',
      content: 'data:application/zip;base64,AA==',
    })
    await vi.waitFor(() =>
      expect(handler).toHaveBeenCalledWith({
        type: 'confirm',
        path: 'import-token',
        datas: ['scenes/main.json'],
      }),
    )
  })

  it('导出业务失败时不下载 JSON 错误响应', async () => {
    const click = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {})
    fetchMock.mockResolvedValue({
      ok: true,
      headers: new Headers({ 'Content-Type': 'application/json' }),
      json: async () => ({ code: 3001, msg: '设计文件不存在' }),
    })
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
})
