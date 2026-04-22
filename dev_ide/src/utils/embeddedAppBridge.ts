import { Storage } from './storage'

const HANDOFF_STORAGE_PREFIX = 'embedded_app_handoff:'
const DEFAULT_HANDOFF_TTL_MS = 10 * 60 * 1000
const HANDOFF_RECORD_FIELDS = [
  'handoffId',
  'appType',
  'projectId',
  'tenantId',
  'tabKey',
  'pageId',
  'issuedAt',
  'expiresAt',
] as const
const SUPPORTED_APP_TYPES = new Set(['designer', 'datacenter'])

type SupportedAppType = 'designer' | 'datacenter'

interface HostState {
  appType?: unknown
  projectId?: unknown
  pid?: unknown
  tenantId?: unknown
  tabKey?: unknown
  pageId?: unknown
  [key: string]: unknown
}

interface HandoffRecord {
  handoffId: string
  appType?: unknown
  projectId?: unknown
  tenantId?: unknown
  tabKey?: unknown
  pageId?: unknown
  issuedAt?: unknown
  expiresAt?: unknown
}

interface PersistedHandoffRecord {
  handoffId: string
  appType: SupportedAppType
  projectId?: unknown
  tenantId?: unknown
  tabKey?: unknown
  pageId?: unknown
  issuedAt: number
  expiresAt: number
}

interface CreateRecordOptions {
  now?: number
  ttlMs?: number
  handoffId?: string
}

export interface BootstrapResponse extends Record<string, unknown> {
  url: string
  handoffId: string
}

const isPlainObject = (value: unknown): value is Record<string, unknown> =>
  value !== null && typeof value === 'object' && !Array.isArray(value)

/**
 * 生成一次性 handoff 标识。
 * 这个值只用于定位宿主态记录，不携带任何业务凭据。
 * @returns {string} opaque handoff id
 */
export const createHandoffId = (): string => {
  const timestamp = Date.now().toString(36)
  const randomPart = Math.random().toString(36).slice(2, 10)
  return `handoff_${timestamp}_${randomPart}`
}

/**
 * 从输入态里提取第一阶段 handoff 需要的恢复索引。
 * 这里明确不保存 token/refreshToken，避免把敏感上下文落入持久化票据。
 * @param {object} hostState - 宿主态输入
 * @returns {object} 规范化后的 handoff 恢复索引
 */
const normalizeHandoffState = (hostState: HostState = {}): Omit<HandoffRecord, 'handoffId'> => ({
  appType: hostState.appType ?? null,
  projectId: hostState.projectId ?? hostState.pid ?? null,
  tenantId: hostState.tenantId ?? null,
  tabKey: hostState.tabKey ?? null,
  pageId: hostState.pageId ?? null,
})

/**
 * 仅保留 handoff 票据允许持久化的字段。
 * 任何未纳入白名单的字段都会被丢弃，包括 token/refreshToken。
 * @param {object} record - 原始记录
 * @returns {object|null} 去敏后的记录
 */
const sanitizeHandoffRecord = (record: unknown): HandoffRecord | null => {
  if (!isPlainObject(record)) {
    return null
  }

  if (typeof record.handoffId !== 'string' || !record.handoffId) {
    return null
  }

  const sanitized: HandoffRecord = {
    handoffId: record.handoffId,
  }

  for (const field of HANDOFF_RECORD_FIELDS) {
    if (field === 'handoffId') {
      continue
    }
    const value = record[field]
    if (value !== undefined && value !== null) {
      sanitized[field] = value
    }
  }

  return sanitized
}

/**
 * 校验 appType 是否属于当前支持的嵌入类型。
 * 这条白名单会在保存、读取、恢复三个环节复用，避免 unknown-app 被误恢复。
 * @param {string} appType - 应用类型
 * @returns {boolean} 是否允许
 */
const isSupportedAppType = (appType: unknown): appType is SupportedAppType =>
  typeof appType === 'string' && SUPPORTED_APP_TYPES.has(appType)

/**
 * 判断 handoff 记录是否已经过期。
 * @param {object} record - 已去敏的 handoff 记录
 * @returns {boolean} 是否过期
 */
const isExpiredHandoffRecord = (record: HandoffRecord): boolean =>
  typeof record.expiresAt === 'number' && record.expiresAt <= Date.now()

/**
 * 统一处理 handoff 记录的去敏和过期判断。
 * 这个方法既给 localStorage 读取分支复用，也给对象态恢复分支复用，确保规则一致。
 * @param {object} record - 原始或已解析的 handoff 记录
 * @returns {object|null} 可恢复的去敏记录
 */
const normalizeHandoffRecordForRestore = (record: unknown): PersistedHandoffRecord | null => {
  const sanitized = sanitizeHandoffRecord(record)
  if (!sanitized) {
    return null
  }

  if (!isSupportedAppType(sanitized.appType)) {
    return null
  }

  if (typeof sanitized.issuedAt !== 'number' || typeof sanitized.expiresAt !== 'number') {
    return null
  }

  if (isExpiredHandoffRecord(sanitized)) {
    return null
  }

  return sanitized as PersistedHandoffRecord
}

/**
 * 校验 handoff 记录是否满足持久化与恢复要求。
 * 保存阶段必须先通过这道门，避免写入后永远无法恢复的脏票据。
 * @param {object} record - 待保存的 handoff 记录
 * @returns {object|null} 可持久化的记录
 */
const validateHandoffRecordForPersistence = (record: unknown): PersistedHandoffRecord | null => {
  const sanitized = sanitizeHandoffRecord(record)
  if (!sanitized) {
    return null
  }

  if (!isSupportedAppType(sanitized.appType)) {
    return null
  }

  if (typeof sanitized.issuedAt !== 'number') {
    return null
  }

  if (typeof sanitized.expiresAt !== 'number' || sanitized.expiresAt <= Date.now()) {
    return null
  }

  return sanitized as PersistedHandoffRecord
}

/**
 * 解析支持的嵌入应用类型。
 * 当前仅允许 designer 和 datacenter，未知类型必须显式失败。
 * @param {string} appType - 应用类型
 * @returns {string} 规范化后的 base path
 */
const resolveAppBasePath = (appType: string): string => {
  if (appType === 'designer') {
    return '/designer/'
  }

  if (appType === 'datacenter') {
    return '/datacenter/'
  }

  throw new Error(`未知的应用类型: ${appType}`)
}

/**
 * 生成 handoff 记录。
 * 记录会带上过期时间，避免旧宿主态被错误恢复。
 * @param {object} hostState - 宿主态
 * @param {object} options - 可选参数
 * @param {number} [options.now] - 用于测试或回放的时间戳
 * @param {number} [options.ttlMs] - 记录有效期
 * @param {string} [options.handoffId] - 外部注入的 handoff id
 * @returns {object} handoff 记录
 */
export const createHandoffRecord = (hostState: HostState = {}, options: CreateRecordOptions = {}): HandoffRecord => {
  const now = options.now ?? Date.now()
  const ttlMs = Number.isFinite(options.ttlMs) && (options.ttlMs as number) > 0 ? (options.ttlMs as number) : DEFAULT_HANDOFF_TTL_MS
  const handoffId = options.handoffId ?? createHandoffId()

  return {
    handoffId,
    ...normalizeHandoffState(hostState),
    issuedAt: now,
    expiresAt: now + ttlMs,
  }
}

/**
 * 构建嵌入应用地址。
 * 这里只下发 opaque handoff id，不下发 token、pid 等敏感字段。
 * @param {string} appType - 应用类型
 * @param {string} handoffId - handoff 标识
 * @returns {string} 可直接打开的嵌入地址
 */
export const buildEmbeddedAppUrl = (appType: string, handoffId?: string | null): string => {
  const params = new URLSearchParams()
  if (handoffId) {
    params.set('handoffId', handoffId)
  }

  const basePath = resolveAppBasePath(appType)
  const query = params.toString()
  return query ? `${basePath}?${query}` : basePath
}

/**
 * 保存 handoff 记录到 localStorage。
 * @param {object} record - handoff 记录
 * @returns {object|null} 保存成功返回记录，失败返回 null
 */
export const saveHandoffRecord = (record: unknown): PersistedHandoffRecord => {
  const sanitizedRecord = validateHandoffRecordForPersistence(record)
  if (!sanitizedRecord) {
    throw new Error('handoff 票据无效，必须包含 appType、issuedAt 和未过期的 expiresAt')
  }

  try {
    localStorage.setItem(`${HANDOFF_STORAGE_PREFIX}${sanitizedRecord.handoffId}`, JSON.stringify(sanitizedRecord))
    return sanitizedRecord
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error)
    const wrappedError = new Error(`保存 handoff 票据失败: ${message}`)
    ;(wrappedError as Error & { cause?: unknown }).cause = error
    console.warn('Save handoff record error:', wrappedError)
    throw wrappedError
  }
}

/**
 * 从 localStorage 读取并解析 handoff 记录。
 * @param {string} storageKey - 存储键
 * @returns {object|null} 记录或 null
 */
const readHandoffRecord = (storageKey: string): PersistedHandoffRecord | null => {
  try {
    const rawValue = localStorage.getItem(storageKey)
    if (!rawValue) {
      return null
    }

    const record = sanitizeHandoffRecord(JSON.parse(rawValue))
    if (!record) {
      return null
    }

    if (isExpiredHandoffRecord(record)) {
      // 过期票据必须从存储里清掉，避免后续前缀扫描再次捞到同一条脏数据。
      localStorage.removeItem(storageKey)
      return null
    }

    if (!isSupportedAppType(record.appType)) {
      return null
    }

    if (typeof record.issuedAt !== 'number' || typeof record.expiresAt !== 'number') {
      return null
    }

    return record as PersistedHandoffRecord
  } catch (error) {
    console.warn(`Read handoff record error for key "${storageKey}":`, error)
    return null
  }
}

/**
 * 读取 handoff 记录。
 * 先按固定 key 直取，失败后再按前缀扫描，作为兼容恢复路径，不是主路径。
 * 过期记录会被视为不可恢复，并在命中时清理。
 * @param {string} handoffId - handoff 标识
 * @returns {object|null} 记录不存在或已过期时返回 null
 */
export const loadHandoffRecord = (handoffId?: string | null): PersistedHandoffRecord | null => {
  if (!handoffId) {
    return null
  }

  const storageKey = `${HANDOFF_STORAGE_PREFIX}${handoffId}`
  const directRecord = readHandoffRecord(storageKey)
  if (directRecord) {
    return directRecord
  }

  const keys = Storage.getKeysByPrefix(HANDOFF_STORAGE_PREFIX)
  for (const key of keys) {
    const record = readHandoffRecord(key)
    if (record?.handoffId === handoffId) {
      return record
    }
  }

  return null
}

/**
 * 将已保存的手递交记录转换成恢复载荷。
 * 这里不返回内部存储字段，避免下游误把元数据当作宿主态使用。
 * @param {string|object} handoff - handoff id 或已加载记录
 * @returns {object|null} 恢复载荷
 */
export const resolveRestorePayload = (handoff: string | Record<string, unknown>): Record<string, unknown> | null => {
  const record =
    typeof handoff === 'string' ? loadHandoffRecord(handoff) : normalizeHandoffRecordForRestore(handoff)
  if (!record?.handoffId) {
    return null
  }

  const payload: Record<string, unknown> = {
    handoffId: record.handoffId,
    appType: record.appType,
  }

  if (record.projectId !== undefined && record.projectId !== null) {
    payload.projectId = record.projectId
  }

  if (record.tenantId !== undefined && record.tenantId !== null) {
    payload.tenantId = record.tenantId
  }

  if (record.tabKey !== undefined && record.tabKey !== null) {
    payload.tabKey = record.tabKey
  }

  if (record.pageId !== undefined && record.pageId !== null) {
    payload.pageId = record.pageId
  }

  return payload
}

/**
 * 生成 bootstrap 响应。
 * 响应对象既包含可直接使用的嵌入地址，也保留完整宿主态字段，便于上层直接透传。
 * @param {string} appType - 应用类型
 * @param {object} hostState - 宿主态
 * @param {object} options - 可选参数
 * @returns {object} bootstrap 响应
 */
export const createBootstrapResponse = (
  appType: string,
  hostState: Record<string, unknown> = {},
  options: CreateRecordOptions = {}
): BootstrapResponse => {
  const { pid, projectId: inputProjectId, ...restHostState } = hostState
  const bootstrapState: Record<string, unknown> = {
    ...restHostState,
    appType,
    projectId: inputProjectId ?? pid ?? null,
  }
  const record = createHandoffRecord(bootstrapState, options)
  const url = buildEmbeddedAppUrl(appType, record.handoffId)
  const savedRecord = saveHandoffRecord(record)

  return {
    ...bootstrapState,
    url,
    handoffId: savedRecord.handoffId,
  }
}
