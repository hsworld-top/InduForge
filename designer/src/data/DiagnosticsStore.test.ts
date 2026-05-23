import { afterEach, describe, expect, it, vi } from 'vitest'
import { DiagnosticsStore } from './DiagnosticsStore'

function buildJsonResponse(body: unknown, status = 200): Response {
  return {
    ok: status >= 200 && status < 300,
    status,
    json: async () => body,
  } as Response
}

describe('DiagnosticsStore HTTP 响应兼容', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
    vi.restoreAllMocks()
  })

  it('兼容旧包络 success=true + data 数组', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        buildJsonResponse({
          success: true,
          data: [
            {
              path: 'device.temp',
              status: 'active',
              dataType: 'number',
            },
          ],
        }),
      ),
    )
    const store = new DiagnosticsStore()

    await (
      store as unknown as { _fetchStatusBatch: (paths: string[]) => Promise<void> }
    )._fetchStatusBatch(['device.temp'])

    expect(store.getPathsByStatus('active')).toContain('device.temp')
    expect(store.getSummary().unknown).toBe(0)
  })

  it('旧包络 success=false 时按失败处理并回填 unknown', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        buildJsonResponse({
          success: false,
          message: '批量查询失败',
        }),
      ),
    )
    const store = new DiagnosticsStore()

    await (
      store as unknown as { _fetchStatusBatch: (paths: string[]) => Promise<void> }
    )._fetchStatusBatch(['device.temp'])

    const issues = store.getIssues()
    expect(issues[0]?.path).toBe('device.temp')
    expect(issues[0]?.status).toBe('unknown')
    expect(String(issues[0]?.statusReason)).toContain('批量查询失败')
  })
})
