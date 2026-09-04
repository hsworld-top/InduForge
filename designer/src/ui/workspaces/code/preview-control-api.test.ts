import { afterEach, describe, expect, it, vi } from 'vitest'

import { previewControlApi } from './preview-control-api'

describe('previewControlApi workspace gateway', () => {
  afterEach(() => vi.unstubAllGlobals())

  it('forwards the one-time gateway ticket and includes the host-only session cookie', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(
        JSON.stringify({
          code: 0,
          msg: 'ok',
          data: {
            status: 'stopped',
            ownership: null,
            port: 5173,
            updatedAt: '2026-09-04T00:00:00Z',
            message: null,
          },
        }),
        { status: 200, headers: { 'Content-Type': 'application/json' } },
      ),
    )
    vi.stubGlobal('fetch', fetchMock)

    await previewControlApi.status(
      'https://preview-control-project.workspace.test/?__if_workspace_ticket=one-time',
    )

    expect(fetchMock).toHaveBeenCalledTimes(1)
    const [target, options] = fetchMock.mock.calls[0] as [URL, RequestInit]
    expect(target.toString()).toBe(
      'https://preview-control-project.workspace.test/api/v1/preview/status?__if_workspace_ticket=one-time',
    )
    expect(options.credentials).toBe('include')
    expect(options.headers).not.toHaveProperty('Authorization')
  })
})
