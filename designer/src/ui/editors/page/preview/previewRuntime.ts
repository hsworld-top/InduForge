/**
 * 预览运行时（previewRuntime）
 *
 * 职责：
 * - 初始化预览页面的数据绑定系统（DataService、MQTT、变量）
 * - 管理组件引用（componentRefsByPage/ByName）供脚本调用
 * - 数据点订阅、MQTT 订阅、变量映射、查询缓存
 * - 提供 initPreviewRuntime、getComponentRef 等 API
 */

import type { Socket } from 'socket.io-client'
import type {
  PreviewComponentRefInfo,
  PreviewGlobalScriptsShape,
  PreviewLifecycleHandler,
  PreviewPageLifecycleShape,
  PreviewRuntimeHandle,
  PreviewRuntimeInitOptions,
  PreviewScriptItem,
  ScriptSectionWithItems,
} from './preview-runtime.types'
import type { DataService } from '@/data'
import { io } from 'socket.io-client'
import { normalizeGlobalValue } from '@/editor-core/utils/variable-utils'
import { datacenterApi } from '@/services'
import { unwrapApiData } from '@/types/api'
import { Storage } from '@/utils/storage'
import {
  extractDatapointValue,
  getQueryExecuteData,
  requireConnectionsPayload,
  requireDatapointsPagePayload,
  requireQueriesPayload,
} from '@/utils/datapoint-payload'
import { buildComponentStub } from './preview-runtime-component-stub'
import {
  applyPendingCalls,
  buildDatapointCacheKey,
  getComponentAlias,
} from './preview-runtime-helpers'

const API_BASE_TRAILING_SLASH_RE = /\/$/
const PARAM_NAME_RE = /^[A-Z_$][\w$]*$/i
const PREVIEW_SOCKET_PATH = '/socket.io'
const PREVIEW_SESSION_HEARTBEAT_INTERVAL = 5 * 60 * 1000

interface DatapointMetaCacheEntry {
  id?: string
  path?: string
  sourceType?: string
  sourceId?: string
  dataType?: string
}

/** 数据点 path 等：仅接受 string / number / boolean，避免把对象误当 path */
function coerceDataCenterPath(value: unknown): string {
  if (typeof value === 'string') return value
  if (typeof value === 'number' || typeof value === 'boolean') return String(value)
  return ''
}

function coerceMqttMapKey(value: unknown): string | null {
  if (typeof value === 'string' && value) return value
  if (typeof value === 'number') return String(value)
  return null
}

function datapointMetaEntryFromRow(row: Record<string, unknown>): DatapointMetaCacheEntry {
  const entry: DatapointMetaCacheEntry = {
    path: String(row.path),
    sourceType: String(row.sourceType ?? ''),
    sourceId: String(row.sourceId ?? ''),
    dataType: String(row.dataType ?? row.type ?? ''),
  }
  if (row.id != null) entry.id = String(row.id)
  return entry
}

/**
 * 仅接受规范形态 { items: T[] }（与 normalizeGlobalScripts / 页面生命周期约定一致）。
 * 顶层数组等旧格式视为非法，返回空列表。
 */
function itemsFromScriptSection(section: unknown): PreviewScriptItem[] {
  if (!section || typeof section !== 'object' || Array.isArray(section)) return []
  const items = (section as ScriptSectionWithItems).items
  return Array.isArray(items) ? (items as PreviewScriptItem[]) : []
}

/** 当前预览运行时实例（单例） */
let runtimeInstance: PreviewRuntimeHandle | null = null
const componentRefsByPage = new Map<string, Map<string, PreviewComponentRefInfo>>()
const componentRefsByName = new Map<string, PreviewComponentRefInfo>()
const previewDataServiceState: {
  service: DataService | null
  connectPromise: Promise<unknown> | null
  projectId: string | null
  subscribed: Set<string>
  pending: Map<string, { promise: Promise<unknown>; resolve: ((v: unknown) => void) | null }>
} = {
  service: null,
  connectPromise: null,
  projectId: null,
  subscribed: new Set(),
  pending: new Map(),
}
const previewMqttState: {
  socket: Socket | null
  connectPromise: Promise<unknown> | null
  projectId: string | null
  sessionId: string | null
  tagValues: Map<string, unknown>
  subscriptionValues: Map<string, unknown>
  datapointValues: Map<string, unknown>
  datapointSubscribed: Set<string>
  tagSubscribed: Set<string>
  subscriptionSubscribed: Set<string>
  tagIdToProps: Map<string, Set<string>>
  subscriptionIdToProps: Map<string, Set<string>>
  subscribePending: Set<string>
  emitDedup: Map<string, number>
  onValueUpdate: ((type: string, id: string, value: unknown, raw?: unknown) => void) | null
} = {
  socket: null,
  connectPromise: null,
  projectId: null,
  sessionId: null,
  tagValues: new Map(),
  subscriptionValues: new Map(),
  datapointValues: new Map(),
  datapointSubscribed: new Set(),
  tagSubscribed: new Set(),
  subscriptionSubscribed: new Set(),
  tagIdToProps: new Map(),
  subscriptionIdToProps: new Map(),
  subscribePending: new Set(),
  emitDedup: new Map(),
  onValueUpdate: null,
}
const previewSessionState: {
  sessionId: string | null
  projectId: string | null
  createPromise: Promise<string | null> | null
  deletePromise: Promise<void> | null
  deleteSessionId: string | null
  heartbeatTimer: ReturnType<typeof setInterval> | null
  heartbeatInFlight: boolean
  generation: number
  createFailed: boolean
} = {
  sessionId: null,
  projectId: null,
  createPromise: null,
  deletePromise: null,
  deleteSessionId: null,
  heartbeatTimer: null,
  heartbeatInFlight: false,
  generation: 0,
  createFailed: false,
}
const pendingComponentCalls = new Map<string, { method: string; args: unknown[] }[]>()

const connectionCache = new Map<string, unknown>()
const queryCache = new Map<string, unknown[]>()
const mappedValueCache = new Map<string, unknown>()
const mappedValuePending = new Map<string, boolean>()
const mappedDetails = new Map<string, Record<string, unknown>>()
const datapointMetaCache = new Map<string, DatapointMetaCacheEntry>()
const datapointMetaPending = new Map<string, Promise<unknown>>()

function getApiBase() {
  if (typeof __VITE_API_URL__ !== 'undefined' && __VITE_API_URL__) {
    return __VITE_API_URL__
  }
  if (typeof import.meta !== 'undefined' && import.meta.env?.VITE_API_URL) {
    return import.meta.env.VITE_API_URL
  }
  return 'http://localhost:19601'
}

function extractPreviewSessionId(payload: unknown): string | null {
  if (!payload || typeof payload !== 'object') return null
  const record = payload as Record<string, unknown>
  const candidates = [
    record.sessionId,
    record.id,
    (record.data as Record<string, unknown> | undefined)?.sessionId,
    (record.data as Record<string, unknown> | undefined)?.id,
  ]
  for (const candidate of candidates) {
    if (typeof candidate === 'string' && candidate) return candidate
  }
  return null
}

function stopPreviewSessionHeartbeat() {
  if (previewSessionState.heartbeatTimer) {
    clearInterval(previewSessionState.heartbeatTimer)
    previewSessionState.heartbeatTimer = null
  }
  previewSessionState.heartbeatInFlight = false
}

async function sendPreviewSessionHeartbeat(sessionId: string) {
  if (!sessionId) return
  try {
    await datacenterApi.heartbeatPreviewSession(sessionId)
  } catch {
    // 预览会话心跳是保活信号，失败时不打断现有回退读取链路。
  }
}

function startPreviewSessionHeartbeat(sessionId: string) {
  stopPreviewSessionHeartbeat()
  if (!sessionId) return
  previewSessionState.heartbeatTimer = setInterval(() => {
    if (!previewSessionState.sessionId || previewSessionState.sessionId !== sessionId) return
    if (previewSessionState.heartbeatInFlight) return
    previewSessionState.heartbeatInFlight = true
    void sendPreviewSessionHeartbeat(sessionId).finally(() => {
      previewSessionState.heartbeatInFlight = false
    })
  }, PREVIEW_SESSION_HEARTBEAT_INTERVAL)
  void sendPreviewSessionHeartbeat(sessionId)
}

function resetPreviewMqttSocketState() {
  if (previewMqttState.socket) {
    previewMqttState.socket.disconnect()
  }
  previewMqttState.socket = null
  previewMqttState.connectPromise = null
  previewMqttState.projectId = null
  previewMqttState.sessionId = null
}

function resetPreviewSessionLocalState() {
  stopPreviewSessionHeartbeat()
  previewSessionState.sessionId = null
  previewSessionState.projectId = null
}

async function deletePreviewSessionRemote(sessionId: string) {
  if (!sessionId) return
  if (previewSessionState.deletePromise && previewSessionState.deleteSessionId === sessionId) {
    return previewSessionState.deletePromise
  }

  const deletePromise = Promise.resolve(datacenterApi.deletePreviewSession(sessionId))
    .then(() => undefined)
    .catch(() => {
      // 删除失败只记录为最佳努力清理，避免影响编辑器退出和重复清理。
      return undefined
    })
    .finally(() => {
      if (previewSessionState.deleteSessionId === sessionId) {
        previewSessionState.deletePromise = null
        previewSessionState.deleteSessionId = null
      }
    }) as Promise<void>

  previewSessionState.deletePromise = deletePromise
  previewSessionState.deleteSessionId = sessionId
  return deletePromise
}

async function ensurePreviewSession(projectId: string | null | undefined) {
  if (!projectId) return null
  if (previewSessionState.sessionId && previewSessionState.projectId === projectId) {
    return previewSessionState.sessionId
  }
  if (
    previewSessionState.createPromise &&
    previewSessionState.projectId === projectId &&
    !previewSessionState.sessionId
  ) {
    return previewSessionState.createPromise
  }

  const token = Storage.getToken()
  if (!token) {
    previewSessionState.createFailed = true
    return null
  }

  const requestGeneration = ++previewSessionState.generation
  previewSessionState.projectId = projectId

  const createPromise: Promise<string | null> = (async () => {
    try {
      const result = await datacenterApi.createPreviewSession(projectId)
      const sessionId = extractPreviewSessionId(unwrapApiData(result))
      if (!sessionId) {
        previewSessionState.createFailed = true
        return null
      }

      if (previewSessionState.generation !== requestGeneration) {
        void deletePreviewSessionRemote(sessionId)
        return null
      }

      previewSessionState.sessionId = sessionId
      previewSessionState.projectId = projectId
      previewSessionState.createFailed = false
      startPreviewSessionHeartbeat(sessionId)
      return sessionId
    } catch {
      previewSessionState.createFailed = true
      return null
    }
  })()

  previewSessionState.createPromise = createPromise
  createPromise.finally(() => {
    if (previewSessionState.createPromise === createPromise) {
      previewSessionState.createPromise = null
    }
  })
  return createPromise
}

async function cleanupPreviewSession(options?: { awaitRemote?: boolean }) {
  const sessionId = previewSessionState.sessionId
  previewSessionState.generation += 1
  previewSessionState.createPromise = null
  resetPreviewSessionLocalState()
  previewSessionState.createFailed = false
  const deletePromise = sessionId ? deletePreviewSessionRemote(sessionId) : null
  if (options?.awaitRemote && deletePromise) {
    await deletePromise
  }
}

async function resolveConnection(projectId: string | null | undefined, name: string) {
  if (connectionCache.has(name)) return connectionCache.get(name)
  if (!projectId) return null
  try {
    const result = await datacenterApi.getConnections(projectId, {
      page: 1,
      limit: 200,
    })
    const body = unwrapApiData(result)
    const connections = requireConnectionsPayload(body)
    const found = connections.find((item: unknown) => (item as { name?: string }).name === name)
    if (found) {
      connectionCache.set(name, found)
    }
    return found || null
  } catch {
    return null
  }
}

async function resolveQuery(
  projectId: string,
  connectionId: string | undefined,

  queryName: string,
) {
  if (!projectId) return null
  const cacheKey = `${projectId}:${connectionId || 'all'}`
  let queries = queryCache.get(cacheKey)
  if (!queries) {
    try {
      const result = await datacenterApi.getQueries(projectId, {
        connectionId,
        page: 1,
        limit: 200,
      })
      const body = unwrapApiData(result)
      queries = requireQueriesPayload(body) as unknown[]
      queryCache.set(cacheKey, queries)
    } catch {
      queries = []
      queryCache.set(cacheKey, queries)
    }
  }
  return (
    (queries as Array<Record<string, unknown>>).find(
      (item) => item.name === queryName || item.id === queryName,
    ) || null
  )
}

async function executeQueryByPath(projectId: string | null | undefined, path: unknown) {
  const [sourceName, ...rest] = String(path || '').split('.')
  const field = rest.join('.')
  if (!sourceName || !field) return undefined
  const connection = (await resolveConnection(projectId, sourceName)) as Record<
    string,
    unknown
  > | null
  if (!connection || connection.type !== 'relational') return undefined
  const query = (await resolveQuery(projectId as string, String(connection.id), field)) as {
    id?: string
  } | null
  if (!query?.id) return undefined
  const result = await datacenterApi.executeQuery(query.id)
  return getQueryExecuteData(unwrapApiData(result))
}

async function ensurePreviewMqttSocket(projectId: string | null | undefined) {
  const sessionId = await ensurePreviewSession(projectId)
  if (!projectId || !sessionId) return null

  const apiBase = getApiBase().replace(API_BASE_TRAILING_SLASH_RE, '')

  if (
    previewMqttState.socket &&
    previewMqttState.projectId === projectId &&
    previewMqttState.sessionId === sessionId
  ) {
    if (!previewMqttState.connectPromise) {
      const sock = previewMqttState.socket
      previewMqttState.connectPromise = new Promise((resolve, reject) => {
        sock.once('connect', () => resolve(undefined))
        sock.once('connect_error', reject)
      })
    }
    try {
      await previewMqttState.connectPromise
      return previewMqttState.socket
    } catch {
      resetPreviewMqttSocketState()
      return null
    }
  }

  resetPreviewMqttSocketState()

  const socketOpts: NonNullable<Parameters<typeof io>[1]> = {
    path: PREVIEW_SOCKET_PATH,
    transports: ['websocket'],
    reconnection: true,
    reconnectionDelay: 1000,
    reconnectionDelayMax: 5000,
    reconnectionAttempts: Infinity,
    auth: {
      token: Storage.getToken() || '',
      projectId,
      previewSessionId: sessionId,
    },
  }
  const socket = io(apiBase, socketOpts)

  previewMqttState.socket = socket
  previewMqttState.projectId = projectId ?? null
  previewMqttState.sessionId = sessionId
  previewMqttState.connectPromise = new Promise((resolve, reject) => {
    socket.once('connect', () => resolve(undefined))
    socket.once('connect_error', reject)
  })

  socket.on('mqtt:tag:value', (data) => {
    if (!data?.tagId) return
    const value = data.value ?? data.parsedValue ?? data.payload
    previewMqttState.tagValues.set(data.tagId, value)
    previewMqttState.onValueUpdate?.('tag', data.tagId, value, data)
  })

  socket.on('connect', () => {
    previewMqttState.tagSubscribed.forEach((tagId) => {
      socket.emit('mqtt:tag:subscribe', { tagId })
    })
    previewMqttState.subscriptionSubscribed.forEach((subscriptionId) => {
      socket.emit('mqtt:subscribe', { subscriptionId })
    })
  })

  socket.on('mqtt:message', (data) => {
    if (!data?.subscriptionId) return
    const value = data.payload ?? data.message ?? data.value ?? data
    previewMqttState.subscriptionValues.set(data.subscriptionId, value)
    previewMqttState.onValueUpdate?.('subscription', data.subscriptionId, value, data)
  })

  socket.on('datapoint:value', (data) => {
    if (!data?.path) return
    const value = data.value ?? data.payload ?? data
    previewMqttState.datapointValues.set(data.path, value)
    previewMqttState.onValueUpdate?.('datapoint', data.path, value, data)
  })

  try {
    await previewMqttState.connectPromise
    return socket
  } catch {
    resetPreviewMqttSocketState()
    return null
  }
}

function emitWithDedup(
  socket: Socket,
  event: string,
  key: string,
  payload: unknown,

  ttl = 800,
) {
  if (!socket || !event || !key) return
  const now = Date.now()
  const cacheKey = `${event}:${key}`
  const last = previewMqttState.emitDedup.get(cacheKey)
  if (last && now - last < ttl) return
  previewMqttState.emitDedup.set(cacheKey, now)
  socket.emit(event, payload)
}
function cacheDatapointMetaList(projectId: string | null | undefined, items: unknown) {
  if (!projectId || !Array.isArray(items)) return
  items.forEach((raw: unknown) => {
    const item = raw as Record<string, unknown>
    if (!item?.path) return
    const key = buildDatapointCacheKey(projectId, item.path)
    if (datapointMetaCache.has(key)) return
    datapointMetaCache.set(key, datapointMetaEntryFromRow(item))
  })
}

async function resolveDatapointMeta(
  projectId: string | null | undefined,
  path: string | null | undefined,
) {
  if (!projectId || !path) return null
  const key = buildDatapointCacheKey(projectId, path)
  if (datapointMetaCache.has(key)) return datapointMetaCache.get(key)
  if (datapointMetaPending.has(key)) return datapointMetaPending.get(key)

  const pending = (async () => {
    let page = 1
    const pageSize = 300
    let totalPages = 1
    while (page <= totalPages) {
      const result = await datacenterApi.getDataPoints(projectId, {
        page,
        pageSize,
      })
      let data
      try {
        data = requireDatapointsPagePayload(unwrapApiData(result))
      } catch {
        break
      }
      const items = data.datapoints as unknown[]
      cacheDatapointMetaList(projectId, items)
      const hit = items.find((row: unknown) => (row as { path?: string })?.path === path) as
        | Record<string, unknown>
        | undefined
      if (hit) {
        const meta = datapointMetaEntryFromRow(hit)
        datapointMetaCache.set(key, meta)
        return meta
      }
      const pagination = data.pagination
      if (pagination.totalPages) {
        totalPages = Number(pagination.totalPages) || 1
      } else if (items.length < pageSize) {
        break
      } else {
        totalPages = Math.max(totalPages, page + 1)
      }
      page += 1
    }
    return null
  })()

  datapointMetaPending.set(key, pending)
  const result = await pending
  datapointMetaPending.delete(key)
  return result
}

function trackMqttProp(prop: string, sourceType: string, sourceId: string) {
  if (!prop || !sourceId) return
  if (sourceType.includes('tag')) {
    if (!previewMqttState.tagIdToProps.has(sourceId)) {
      previewMqttState.tagIdToProps.set(sourceId, new Set())
    }
    previewMqttState.tagIdToProps.get(sourceId)!.add(prop)
  } else if (sourceType.includes('subscription')) {
    if (!previewMqttState.subscriptionIdToProps.has(sourceId)) {
      previewMqttState.subscriptionIdToProps.set(sourceId, new Set())
    }
    previewMqttState.subscriptionIdToProps.get(sourceId)!.add(prop)
  }
}

async function resolveSourceInfo(
  projectId: string | null | undefined,
  detail: Record<string, unknown>,
) {
  const source = (detail?.source as Record<string, unknown>) || {}
  const path = coerceDataCenterPath(source.path)
  let sourceType = String(source.sourceType || '')
  let sourceId = String(source.sourceId || '')
  let datapointId = String(source.datapointId || '')
  if (projectId && path) {
    const meta = (await resolveDatapointMeta(projectId, path)) as {
      sourceType?: string
      sourceId?: string
      id?: string
    } | null
    if (meta) {
      if (!sourceType || sourceType !== meta.sourceType) {
        sourceType = meta.sourceType || sourceType
      }
      if (!sourceId || sourceId !== meta.sourceId) {
        sourceId = meta.sourceId || sourceId
      }
      if (!datapointId || datapointId !== meta.id) {
        datapointId = meta.id || datapointId
      }
    }
  }
  return { path, sourceType, sourceId, datapointId }
}

function registerMqttMapping(prop: string, detail: Record<string, unknown>) {
  const src = detail?.source as Record<string, unknown> | undefined
  if (!detail?.mapped || src?.type !== 'dataCenter') return
  mappedDetails.set(prop, detail)
  const srcMeta = detail.source as Record<string, unknown> | undefined
  const sourceType = String(srcMeta?.sourceType || '')
  const sourceId = srcMeta?.sourceId
  if (!sourceId || typeof sourceId !== 'string') return
  trackMqttProp(prop, sourceType, sourceId)
}

async function subscribeMqttSource(
  projectId: string | null | undefined,
  detail: Record<string, unknown>,
) {
  const src0 = detail?.source as Record<string, unknown> | undefined
  if (!detail?.mapped || src0?.type !== 'dataCenter') return
  const resolved = await resolveSourceInfo(projectId, detail)
  const sourceType = String(resolved.sourceType || '')
  const sourceId = resolved.sourceId
  const path = coerceDataCenterPath(resolved.path)
  const socket = await ensurePreviewMqttSocket(projectId)
  const pendingKey = sourceId ? `${sourceType}:${sourceId}` : path ? `datapoint:${path}` : ''
  if (pendingKey) {
    if (previewMqttState.subscribePending.has(pendingKey)) return
    previewMqttState.subscribePending.add(pendingKey)
  }
  try {
    if (sourceId) {
      if (sourceType.includes('tag')) {
        if (!previewMqttState.tagSubscribed.has(sourceId)) {
          previewMqttState.tagSubscribed.add(sourceId)
          if (socket?.connected) {
            emitWithDedup(socket!, 'mqtt:tag:subscribe', sourceId, {
              tagId: sourceId,
            })
          }
        }
      } else if (sourceType.includes('subscription')) {
        if (!previewMqttState.subscriptionSubscribed.has(sourceId)) {
          previewMqttState.subscriptionSubscribed.add(sourceId)
          if (socket?.connected) {
            emitWithDedup(socket!, 'mqtt:subscribe', sourceId, {
              subscriptionId: sourceId,
            })
          }
        }
      }
    }
    if (path && socket && !previewMqttState.datapointSubscribed.has(path)) {
      emitWithDedup(socket, 'datapoint:subscribe', `${projectId}:${path}`, {
        projectId,
        paths: [path],
      })
      previewMqttState.datapointSubscribed.add(path)
    }
  } finally {
    if (pendingKey) previewMqttState.subscribePending.delete(pendingKey)
  }
}
async function resolveMappedGlobalValue(
  projectId: string | null | undefined,
  detail: Record<string, unknown>,
) {
  const source = detail?.source as Record<string, unknown> | undefined
  if (!source || source.type !== 'dataCenter' || !source.path) {
    return normalizeGlobalValue(detail)
  }
  if (!projectId) return normalizeGlobalValue(detail)
  const fallbackValue = normalizeGlobalValue(detail)

  if (source.datapointId || source.sourceType || source.sourceId || source.path) {
    const resolved = await resolveSourceInfo(projectId, detail)
    const sourceType = String(resolved.sourceType || '')
    if (sourceType.includes('query') && (resolved.sourceId || source.sourceId)) {
      try {
        const result = await datacenterApi.executeQuery(
          String(resolved.sourceId || source.sourceId || ''),
        )
        const value = getQueryExecuteData(unwrapApiData(result))
        return value ?? fallbackValue
      } catch {
        try {
          const fallbackResult = await executeQueryByPath(
            projectId,
            coerceDataCenterPath(source.path),
          )
          if (fallbackResult !== undefined) return fallbackResult
        } catch {
          // ignore
        }
        return fallbackValue
      }
    }
    if (sourceType.includes('subscription') || sourceType.includes('tag')) {
      await subscribeMqttSource(projectId, detail)
      if (sourceType.includes('tag')) {
        const tagKey = coerceMqttMapKey(resolved.sourceId) ?? coerceMqttMapKey(source.sourceId)
        if (tagKey) {
          const value = previewMqttState.tagValues.get(tagKey)
          if (value !== undefined) return value
        }
      }
      if (sourceType.includes('subscription')) {
        const subKey = coerceMqttMapKey(resolved.sourceId) ?? coerceMqttMapKey(source.sourceId)
        if (subKey) {
          const value = previewMqttState.subscriptionValues.get(subKey)
          if (value !== undefined) return value
        }
      }
      const dpPath = coerceDataCenterPath(resolved.path)
      if (dpPath) {
        const pathValue = previewMqttState.datapointValues.get(dpPath)
        return pathValue ?? fallbackValue
      }
      return fallbackValue
    }
    const dpId = coerceMqttMapKey(resolved.datapointId) ?? coerceMqttMapKey(source.datapointId)
    if (dpId) {
      try {
        const result = await datacenterApi.getDatapointValues(projectId, [dpId])
        const payload = unwrapApiData(result)
        const picked = extractDatapointValue(payload, dpId)
        return picked ?? fallbackValue
      } catch {
        return fallbackValue
      }
    }
  }

  if (String(source.path).startsWith('mqtt.')) {
    await subscribeMqttSource(projectId, detail)
    const pathValue = previewMqttState.datapointValues.get(String(source.path))
    return pathValue ?? fallbackValue
  }

  const [sourceName, ...rest] = String(source.path).split('.')
  const field = rest.join('.')
  if (!sourceName || !field) return normalizeGlobalValue(detail)

  const connection = (await resolveConnection(projectId, sourceName)) as Record<
    string,
    unknown
  > | null
  if (!connection) return normalizeGlobalValue(detail)

  if (connection.type === 'relational') {
    const query = (await resolveQuery(projectId, String(connection.id), field)) as {
      id?: string
    } | null
    if (!query?.id) return normalizeGlobalValue(detail)
    try {
      const result = await datacenterApi.executeQuery(query.id)
      return getQueryExecuteData(unwrapApiData(result)) ?? fallbackValue
    } catch {
      return fallbackValue
    }
  }

  return fallbackValue
}

function parseParamNames(value: unknown) {
  if (!value || typeof value !== 'string') return []
  return value
    .split(',')
    .map((name) => name.trim())
    .filter((name) => PARAM_NAME_RE.test(name))
}

export function initPreviewRuntime(
  options?: PreviewRuntimeInitOptions | null,
): PreviewRuntimeHandle {
  const { projectId, projectVariables, globalScripts, pageLifecycle, pageVariables } = options || {}
  const overrides = new Map<string, unknown>()
  const timerIds = new Set<ReturnType<typeof setInterval>>()
  const pageTimerIds = new Set<ReturnType<typeof setInterval>>()
  const lifecycleConfig: PreviewPageLifecycleShape =
    (pageLifecycle as PreviewPageLifecycleShape | undefined) || {}
  const pageVarOverrides = new Map<string, unknown>()
  const pageVarDefs: Record<string, unknown> =
    pageVariables && typeof pageVariables === 'object'
      ? (pageVariables as Record<string, unknown>)
      : {}

  const updateMappedValue = (prop: string, nextValue: unknown, detail: unknown) => {
    const fallbackValue = normalizeGlobalValue(detail)
    const resolvedValue = nextValue ?? fallbackValue
    const previous = mappedValueCache.has(prop) ? mappedValueCache.get(prop) : fallbackValue
    mappedValueCache.set(prop, resolvedValue)
    if (previous !== resolvedValue) {
      triggerVariableChange(prop, resolvedValue, previous)
    }
  }

  const preloadMappedGlobals = async () => {
    const entries = Object.entries(
      (projectVariables || {}) as Record<string, Record<string, unknown>>,
    )
    const tasks: Promise<unknown>[] = []
    entries.forEach(([name, detail]) => {
      const src = detail?.source as Record<string, unknown> | undefined
      if (!detail?.mapped || src?.type !== 'dataCenter') return
      if (mappedValueCache.has(name) || mappedValuePending.has(name)) return
      const sourceType = String(src?.sourceType || '')
      if (!sourceType.includes('query')) return
      mappedValuePending.set(name, true)
      tasks.push(
        resolveMappedGlobalValue(projectId, detail)
          .then((value) => {
            updateMappedValue(name, value, detail)
          })
          .finally(() => {
            mappedValuePending.delete(name)
          }),
      )
    })
    if (tasks.length > 0) {
      await Promise.allSettled(tasks)
    }
  }

  previewMqttState.onValueUpdate = (type, id, value, _raw?) => {
    if (type === 'datapoint') {
      mappedDetails.forEach((detail, name) => {
        const src = detail?.source as Record<string, unknown> | undefined
        const path = src?.path
        if (path && path === id) {
          updateMappedValue(name, value, detail)
        }
      })
      return
    }
    const detailMap =
      type === 'tag' ? previewMqttState.tagIdToProps : previewMqttState.subscriptionIdToProps
    const props = detailMap.get(id)
    if (!props || props.size === 0) return
    props.forEach((prop) => {
      const pv = projectVariables as Record<string, unknown> | undefined
      const detail = pv?.[prop] || mappedDetails.get(prop)
      updateMappedValue(prop, value, detail)
    })
  }

  const globalsProxy = new Proxy(
    {},
    {
      get(_target, prop) {
        if (typeof prop !== 'string') return undefined
        if (overrides.has(prop)) return overrides.get(prop)
        const pv = projectVariables as Record<string, Record<string, unknown>>
        const detail = pv?.[prop]
        if (!detail) return undefined
        const gsrc = detail.source as Record<string, unknown> | undefined
        if (detail?.mapped && gsrc?.type === 'dataCenter') {
          registerMqttMapping(prop, detail)
          const sourceType = String(gsrc?.sourceType || '')
          const sourceId = gsrc?.sourceId
          const path = gsrc?.path
          if ((!sourceId || !sourceType) && path) {
            resolveSourceInfo(projectId, detail).then((resolved) => {
              if (resolved?.sourceId && resolved?.sourceType) {
                trackMqttProp(prop, resolved.sourceType, String(resolved.sourceId))
              }
            })
          }
          if (sourceId) {
            if (sourceType.includes('tag')) {
              const liveValue = previewMqttState.tagValues.get(String(sourceId))
              if (liveValue !== undefined) {
                updateMappedValue(prop, liveValue, detail)
                return liveValue
              }
            }
            if (sourceType.includes('subscription')) {
              const liveValue = previewMqttState.subscriptionValues.get(String(sourceId))
              if (liveValue !== undefined) {
                updateMappedValue(prop, liveValue, detail)
                return liveValue
              }
            }
          }
          if (path) {
            const pathValue = previewMqttState.datapointValues.get(String(path))
            if (pathValue !== undefined) {
              updateMappedValue(prop, pathValue, detail)
              return pathValue
            }
          }
          if (!mappedValuePending.has(prop)) {
            mappedValuePending.set(prop, true)
            resolveMappedGlobalValue(projectId, detail as Record<string, unknown>)
              .then((value) => {
                updateMappedValue(prop, value ?? normalizeGlobalValue(detail), detail)
              })
              .finally(() => {
                mappedValuePending.delete(prop)
              })
          }
          if (mappedValueCache.has(prop)) {
            return mappedValueCache.get(prop)
          }
          return normalizeGlobalValue(detail)
        }
        return normalizeGlobalValue(detail)
      },
      set(_target, prop, value) {
        if (typeof prop !== 'string') return false
        const pvSet = projectVariables as Record<string, { default?: unknown }> | undefined
        const prev = overrides.has(prop) ? overrides.get(prop) : pvSet?.[prop]?.default
        overrides.set(prop, value)
        if (prev !== value) {
          triggerVariableChange(prop, value, prev)
        }
        return true
      },
    },
  )

  const normalizePageValue = (detail: unknown) => {
    if (!detail || typeof detail !== 'object') return detail ?? null
    const d = detail as Record<string, unknown>
    if (d.default === undefined && d.defaultValue !== undefined) {
      return normalizeGlobalValue({ ...d, default: d.defaultValue })
    }
    return normalizeGlobalValue(detail)
  }

  const triggerPageVariableChange = async (name: string, value: unknown, previous: unknown) => {
    const items = itemsFromScriptSection(lifecycleConfig.variableChanges)
    const hits = items.filter((item) => (item.variable || item.name) === name && item.code)
    for (const item of hits) {
      if (item.enabled === false) continue
      await runCode(item.code, { name, value, previous }, null, options?.pageId)
    }
  }

  const pageVarsProxy = new Proxy(
    {},
    {
      get(_target, prop) {
        if (typeof prop !== 'string') return undefined
        if (pageVarOverrides.has(prop)) return pageVarOverrides.get(prop)
        const detail = pageVarDefs?.[prop]
        if (!detail) return undefined
        return normalizePageValue(detail)
      },
      set(_target, prop, value) {
        if (typeof prop !== 'string') return false
        const detail = pageVarDefs?.[prop]
        const prev = pageVarOverrides.has(prop)
          ? pageVarOverrides.get(prop)
          : detail
            ? normalizePageValue(detail)
            : undefined
        pageVarOverrides.set(prop, value)
        if (prev !== value) {
          triggerPageVariableChange(prop, value, prev)
        }
        return true
      },
    },
  )

  const buildCustomScripts = () => {
    const items = itemsFromScriptSection(
      (globalScripts as PreviewGlobalScriptsShape | undefined)?.custom,
    )
    const handlers: Record<string, (...args: unknown[]) => Promise<unknown>> = {}
    items.forEach((item) => {
      if (!item.name) return
      const paramNames = parseParamNames(item.params || item.args)
      handlers[String(item.name)] = async (...args: unknown[]) => {
        const code = String(item.code || '')
        if (!code.trim()) return undefined
        const scope = {
          $global: globalsProxy,
          $vars: pageVarsProxy,
          customScripts: handlers,
          console,
          $event: undefined,
        }
        const localKeys = [...paramNames, ...Object.keys(scope)]
        const localValues = [...paramNames.map((_, index) => args[index]), ...Object.values(scope)]
        try {
          const runner = new Function(
            ...localKeys,
            `"use strict";\nreturn (async function() {\n${code}\n}).call(this);`,
          )
          return await runner(...localValues)
        } catch (error) {
          console.error(`[Preview] customScripts.${item.name} error:`, error)
          return undefined
        }
      }
    })
    return handlers
  }

  const customScripts = buildCustomScripts()

  const buildPageProxy = () =>
    new Proxy(
      {},
      {
        get(_target, prop) {
          if (typeof prop !== 'string') return undefined
          const pageMap = componentRefsByPage.get(prop)
          if (!pageMap) {
            return new Proxy(
              {},
              {
                get(_subTarget, name) {
                  if (typeof name !== 'string') return undefined
                  return buildComponentStub(pendingComponentCalls, prop, name)
                },
              },
            )
          }
          return new Proxy(
            {},
            {
              get(_subTarget, name) {
                if (typeof name !== 'string') return undefined
                return pageMap.get(name) || buildComponentStub(pendingComponentCalls, prop, name)
              },
            },
          )
        },
      },
    )

  const getComponentsProxy = (pageId: string | null | undefined) =>
    new Proxy(
      {},
      {
        get(_target, prop) {
          if (typeof prop !== 'string') return undefined
          if (prop === 'pages') return buildPageProxy()
          const pageMap = componentRefsByPage.get(pageId ?? '')
          if (pageMap && pageMap.has(prop)) return pageMap.get(prop)
          if (componentRefsByName.has(prop)) return componentRefsByName.get(prop)
          return buildComponentStub(pendingComponentCalls, pageId, prop)
        },
      },
    )

  async function runCode(
    code: unknown,
    event: unknown,
    thisArg: unknown,
    pageId: string | null | undefined,
  ) {
    if (!code || !String(code).trim()) return
    await preloadMappedGlobals()
    const components = getComponentsProxy(pageId || options?.pageId)
    const context = {
      $event: event,
      $global: globalsProxy,
      $vars: pageVarsProxy,
      customScripts,
      components,
      console,
    }
    try {
      const runner = new Function(
        ...Object.keys(context),
        `"use strict";\nreturn (async function() {\n${String(code)}\n}).call(this);`,
      )
      return await runner.call(thisArg || null, ...Object.values(context))
    } catch (error) {
      console.error('[Preview] Script error:', error)
    }
  }

  async function triggerVariableChange(name: string, value: unknown, previous: unknown) {
    const items = itemsFromScriptSection(
      (globalScripts as PreviewGlobalScriptsShape | undefined)?.variableChanges,
    )
    const hits = items.filter((item) => (item.variable || item.name) === name && item.code)
    for (const item of hits) {
      if (item.enabled === false) continue
      await runCode(item.code, { name, value, previous }, null, options?.pageId ?? null)
    }
  }

  /**
   * 启动页面定时器
   * @returns {void}
   */
  const startPageTimers = () => {
    const timers = itemsFromScriptSection(lifecycleConfig.timers)
    timers.forEach((item) => {
      if (!item.code || item.enabled === false) return
      const interval = Number(item.interval || item.time || 1000)
      const id = setInterval(
        () => {
          void runCode(item.code, { type: 'timer', name: item.name }, null, options?.pageId)
        },
        Math.max(100, interval),
      )
      pageTimerIds.add(id)
    })
  }

  /**
   * 执行页面生命周期脚本
   * @param {string} key - 生命周期 key
   * @returns {Promise<void>} 执行结果
   */
  const runPageLifecycleHandlers = async (key: string) => {
    if (!key) return
    const handlersRaw = lifecycleConfig[key]
    const handlers = Array.isArray(handlersRaw) ? (handlersRaw as PreviewLifecycleHandler[]) : []
    for (const handler of handlers) {
      if (!handler) continue
      if (typeof handler === 'object' && (handler as PreviewScriptItem).enabled === false) {
        continue
      }
      const code = typeof handler === 'string' ? handler : (handler as PreviewScriptItem).code
      if (!code || !String(code).trim()) continue
      await runCode(code, { type: 'lifecycle', name: key }, null, options?.pageId)
    }
  }

  const start = async () => {
    const gs = globalScripts as PreviewGlobalScriptsShape | undefined
    const system = gs?.system
    const startup = system?.startup
    const systemCode = startup?.code
    previewSessionState.createFailed = false
    await ensurePreviewSession(projectId ?? null)
    await runCode(systemCode, { type: 'startup' }, null, options?.pageId ?? null)
    await runPageLifecycleHandlers('onMounted')
    const globalTimers = itemsFromScriptSection(gs?.timers)
    globalTimers.forEach((item) => {
      const interval = Number(item.interval || item.time || 1000)
      if (!item.code) return
      const id = setInterval(
        () => {
          void runCode(item.code, { type: 'timer', name: item.name }, null, options?.pageId ?? null)
        },
        Math.max(100, interval),
      )
      timerIds.add(id)
    })
    startPageTimers()
  }

  const stop = async () => {
    timerIds.forEach((id) => clearInterval(id))
    timerIds.clear()
    pageTimerIds.forEach((id) => clearInterval(id))
    pageTimerIds.clear()
    await runPageLifecycleHandlers('onUnmounted')
    const gs = globalScripts as PreviewGlobalScriptsShape | undefined
    const system = gs?.system
    const shutdown = system?.shutdown
    const shutdownCode = shutdown?.code
    await runCode(shutdownCode, { type: 'shutdown' }, null, options?.pageId ?? null)
    resetPreviewMqttSocketState()
    await cleanupPreviewSession({ awaitRemote: true })
  }

  const registerComponentRef = (pageIdValue: string, name: string, refInfo: unknown) => {
    if (!pageIdValue || !name || !refInfo) return
    const refObj = refInfo as PreviewComponentRefInfo
    if (!componentRefsByPage.has(pageIdValue)) {
      componentRefsByPage.set(pageIdValue, new Map())
    }
    componentRefsByPage.get(pageIdValue)!.set(name, refObj)
    componentRefsByName.set(name, refObj)
    applyPendingCalls(pendingComponentCalls, pageIdValue, name, refObj)
    const alias = getComponentAlias(name)
    if (alias && alias !== name) {
      const pageMap = componentRefsByPage.get(pageIdValue)
      if (pageMap && !pageMap.has(alias)) {
        pageMap.set(alias, refObj)
      }
      if (!componentRefsByName.has(alias)) {
        componentRefsByName.set(alias, refObj)
      }
      applyPendingCalls(pendingComponentCalls, pageIdValue, alias, refObj)
    }
  }

  const unregisterComponentRef = (pageIdValue: string, name: string, refInfo: unknown) => {
    if (!pageIdValue || !name) return
    const pageMap = componentRefsByPage.get(pageIdValue)
    if (pageMap && pageMap.get(name) === refInfo) {
      pageMap.delete(name)
    }
    if (componentRefsByName.get(name) === refInfo) {
      componentRefsByName.delete(name)
    }
    const alias = getComponentAlias(name)
    if (alias && alias !== name) {
      if (pageMap && pageMap.get(alias) === refInfo) {
        pageMap.delete(alias)
      }
      if (componentRefsByName.get(alias) === refInfo) {
        componentRefsByName.delete(alias)
      }
    }
  }

  runtimeInstance = {
    globals: globalsProxy,
    customScripts,
    runCode,
    start,
    stop,
    registerComponentRef,
    unregisterComponentRef,
  }

  return runtimeInstance
}

export function getPreviewRuntime(): PreviewRuntimeHandle | null {
  return runtimeInstance
}

export function clearPreviewRuntime() {
  runtimeInstance = null
  mappedValueCache.clear()
  mappedValuePending.clear()
  mappedDetails.clear()
  componentRefsByPage.clear()
  componentRefsByName.clear()
  pendingComponentCalls.clear()
  if (previewDataServiceState.service) {
    previewDataServiceState.service.destroy()
  }
  previewDataServiceState.service = null
  previewDataServiceState.connectPromise = null
  previewDataServiceState.projectId = null
  previewDataServiceState.subscribed.clear()
  previewDataServiceState.pending.clear()
  resetPreviewMqttSocketState()
  previewMqttState.tagValues.clear()
  previewMqttState.subscriptionValues.clear()
  previewMqttState.datapointValues.clear()
  previewMqttState.datapointSubscribed.clear()
  previewMqttState.tagSubscribed.clear()
  previewMqttState.subscriptionSubscribed.clear()
  previewMqttState.tagIdToProps.clear()
  previewMqttState.subscriptionIdToProps.clear()
  previewMqttState.subscribePending.clear()
  previewMqttState.emitDedup.clear()
  previewMqttState.onValueUpdate = null
  void cleanupPreviewSession()
}
