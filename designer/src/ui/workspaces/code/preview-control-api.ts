export type PreviewProcessStatus = 'starting' | 'running' | 'stopping' | 'stopped' | 'error'
export type PreviewProcessOwnership = 'managed' | 'external' | null

export interface PreviewControlState {
  status: PreviewProcessStatus
  ownership: PreviewProcessOwnership
  port: number
  updatedAt: string
  message: string | null
}

interface PreviewControlEnvelope {
  code: number
  msg: string
  data?: unknown
  reqId?: string
}

function parsePreviewControlState(value: unknown): PreviewControlState {
  if (!value || typeof value !== 'object') throw new Error('预览进程状态不符合契约')
  const input = value as Record<string, unknown>
  if (!['starting', 'running', 'stopping', 'stopped', 'error'].includes(String(input.status))) {
    throw new Error('预览进程 status 不符合契约')
  }
  if (![null, 'managed', 'external'].includes(input.ownership as null | string)) {
    throw new Error('预览进程 ownership 不符合契约')
  }
  if (!Number.isInteger(input.port) || Number(input.port) <= 0 || Number(input.port) > 65535) {
    throw new Error('预览进程 port 不符合契约')
  }
  if (typeof input.updatedAt !== 'string' || !input.updatedAt) {
    throw new Error('预览进程 updatedAt 不符合契约')
  }
  if (input.message !== null && typeof input.message !== 'string') {
    throw new Error('预览进程 message 不符合契约')
  }
  return {
    status: input.status as PreviewProcessStatus,
    ownership: input.ownership as PreviewProcessOwnership,
    port: Number(input.port),
    updatedAt: input.updatedAt,
    message: input.message as string | null,
  }
}

async function requestPreviewControl(
  controlUrl: string,
  path: string,
  method: 'GET' | 'POST',
): Promise<PreviewControlState> {
  const target = new URL(path, controlUrl)
  const response = await fetch(target, {
    method,
    credentials: 'omit',
    headers: { Accept: 'application/json' },
  })
  const payload = (await response.json()) as PreviewControlEnvelope
  if (!response.ok || payload.code !== 0) {
    throw new Error(payload.msg || `预览控制请求失败 (${response.status})`)
  }
  return parsePreviewControlState(payload.data)
}

export const previewControlApi = {
  status(controlUrl: string): Promise<PreviewControlState> {
    return requestPreviewControl(controlUrl, '/api/v1/preview/status', 'GET')
  },
  start(controlUrl: string): Promise<PreviewControlState> {
    return requestPreviewControl(controlUrl, '/api/v1/preview/start', 'POST')
  },
  stop(controlUrl: string): Promise<PreviewControlState> {
    return requestPreviewControl(controlUrl, '/api/v1/preview/stop', 'POST')
  },
  restart(controlUrl: string): Promise<PreviewControlState> {
    return requestPreviewControl(controlUrl, '/api/v1/preview/restart', 'POST')
  },
}
