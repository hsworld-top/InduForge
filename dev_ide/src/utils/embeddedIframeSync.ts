import { createBootstrapResponse } from './embeddedAppBridge'

const EMBEDDED_IFRAME_SELECTOR = 'iframe.embedded-iframe'
const SUPPORTED_EMBEDDED_MESSAGE_TYPES = new Set([
  'APP_BOOTSTRAP_REQUEST',
  'AUTH_REFRESHED',
  'AUTH_EXPIRED',
])

type EmbeddedMessageType = 'APP_BOOTSTRAP_REQUEST' | 'AUTH_REFRESHED' | 'AUTH_EXPIRED'
type EmbeddedAppType = 'designer' | 'datacenter'

interface MessageWindowLike {
  postMessage?: (message: unknown, targetOrigin: string) => void
}

interface IframeLike {
  src?: string
  contentWindow?: MessageWindowLike | null
  getAttribute?: (name: string) => string | null | undefined
}

interface EmbeddedRegistryEntry {
  iframe: IframeLike
  origin: string
  appType: EmbeddedAppType
  project: Record<string, unknown> | null
  tabKey: unknown
  pageId: unknown
}

interface MessageEventLike {
  data?: Record<string, any>
  source?: unknown
  origin?: string
}

interface HandleEmbeddedMessageOptions {
  event?: MessageEventLike
  registry?: Map<unknown, EmbeddedRegistryEntry> | null
  resolveBootstrapState?: (
    entry: EmbeddedRegistryEntry,
    payload: Record<string, unknown>,
  ) => Record<string, unknown> | null | undefined
  onAuthRefreshed?: (payload: Record<string, unknown>, entry: EmbeddedRegistryEntry) => void
  onAuthExpired?: (payload: Record<string, unknown>, entry: EmbeddedRegistryEntry) => void
  postMessage?: (target: EmbeddedRegistryEntry, message: Record<string, unknown>) => unknown
}

interface DocumentLike {
  querySelectorAll?: (selector: string) => Iterable<unknown> | null
}

type EmbeddedTarget = EmbeddedRegistryEntry | IframeLike | Record<string, any>

/**
 * 获取当前宿主页面 origin。
 * 当调用方传入相对 URL 但没有显式 origin 时，只允许回落到真实宿主 origin，不能伪造 http://localhost。
 * @returns {string|null} 当前宿主 origin
 */
const resolveCurrentOrigin = (): string | null => {
  const locationLike = globalThis.window?.location ?? globalThis.location
  return typeof locationLike?.origin === 'string' && locationLike.origin
    ? locationLike.origin
    : null
}

/**
 * 构造嵌入应用更新消息。
 * @param {string} type - 消息类型
 * @param {string} key - 消息字段名
 * @param {string} value - 消息值
 * @returns {object} 消息对象
 */
export const createEmbeddedUpdateMessage = (
  type: string,
  key: string,
  value: string,
): Record<string, string> => ({
  type,
  [key]: value,
})

/**
 * 将输入统一转换为可遍历的嵌入目标列表。
 * 这里同时兼容注册表 values、数组、NodeList，以及旧的 document 查询链路。
 * @param {Iterable<object>|object} targets - 目标集合或 document-like 对象
 * @returns {Array<object>} 归一化后的目标数组
 */
const normalizeEmbeddedTargets = (targets: unknown): EmbeddedTarget[] => {
  if (!targets) return []

  const documentLike = targets as DocumentLike
  if (documentLike?.querySelectorAll) {
    return Array.from(
      documentLike.querySelectorAll(EMBEDDED_IFRAME_SELECTOR) || [],
    ) as EmbeddedTarget[]
  }

  if (typeof (targets as Record<PropertyKey, unknown>)[Symbol.iterator] === 'function') {
    return Array.from(targets as Iterable<EmbeddedTarget>)
  }

  const forEachLike = targets as {
    forEach?: (handler: (item: EmbeddedTarget) => void) => void
  }
  if (typeof forEachLike.forEach === 'function') {
    const items: EmbeddedTarget[] = []
    forEachLike.forEach((item: EmbeddedTarget) => items.push(item))
    return items
  }

  return []
}

/**
 * 提取广播或匹配时真正的 iframe 节点。
 * 注册表项会持有 iframe，旧链路则直接传 iframe。
 * @param {object} target - 注册表项或 iframe 节点
 * @returns {object|null} iframe 节点
 */
const resolveIframeFromTarget = (target: unknown): IframeLike | null => {
  if (!target || typeof target !== 'object') {
    return null
  }
  const record = target as Record<string, unknown>
  const iframe = (record.iframe ?? target) as IframeLike | null
  return iframe ?? null
}

/**
 * 从 iframe src 或注册表项里解析 origin。
 * 主题/语言广播必须显式携带 origin，不能再使用 "*"。
 * @param {object} target - 注册表项或 iframe 节点
 * @returns {string|null} origin
 */
const resolveOriginFromTarget = (target: unknown): string | null => {
  if (target && typeof target === 'object') {
    const record = target as Record<string, unknown>
    if (typeof record.origin === 'string' && record.origin) {
      return record.origin
    }
  }

  const iframe = resolveIframeFromTarget(target)
  const src = iframe?.src || iframe?.getAttribute?.('src') || ''
  if (!src) return null

  try {
    if (/^[a-zA-Z][a-zA-Z\d+\-.]*:/.test(src)) {
      return new URL(src).origin
    }

    const currentOrigin = resolveCurrentOrigin()
    return currentOrigin ? new URL(src, currentOrigin).origin : null
  } catch (_error) {
    return null
  }
}

/**
 * 从 iframe src 或注册表项里解析嵌入应用类型。
 * @param {object} target - 注册表项或 iframe 节点
 * @returns {string|null} 应用类型
 */
const resolveAppTypeFromTarget = (target: unknown): EmbeddedAppType | null => {
  if (target && typeof target === 'object') {
    const record = target as Record<string, unknown>
    if (record.appType === 'designer' || record.appType === 'datacenter') {
      return record.appType
    }
  }

  const iframe = resolveIframeFromTarget(target)
  const src = iframe?.src || iframe?.getAttribute?.('src') || ''
  if (!src) return null

  try {
    const currentOrigin = resolveCurrentOrigin()
    // 这里仅为了解析 pathname 判定 appType；即使没有真实 origin，也不能影响广播或消息匹配用到的 origin。
    const url = /^[a-zA-Z][a-zA-Z\d+\-.]*:/.test(src)
      ? new URL(src)
      : currentOrigin
        ? new URL(src, currentOrigin)
        : new URL(src, 'http://embedded.invalid')

    if (url.pathname === '/designer' || url.pathname.startsWith('/designer/')) {
      return 'designer'
    }
    if (url.pathname === '/datacenter' || url.pathname.startsWith('/datacenter/')) {
      return 'datacenter'
    }
    return null
  } catch (_error) {
    return null
  }
}

/**
 * 发送消息到嵌入目标。
 * @param {object} target - 注册表项或 iframe 节点
 * @param {object} message - 消息对象
 * @returns {boolean} 是否已发送
 */
export const postMessageToEmbeddedTarget = (
  target: unknown,
  message: Record<string, unknown>,
): boolean => {
  const iframe = resolveIframeFromTarget(target)
  const origin = resolveOriginFromTarget(target)
  if (!iframe?.contentWindow?.postMessage || !origin) {
    return false
  }

  iframe.contentWindow.postMessage(message, origin)
  return true
}

/**
 * 基于 iframe 节点和上下文构建注册表项。
 * Dashboard 会在 iframe load 后调用它，后续 message 匹配统一走这份注册表。
 * @param {object} payload - 注册信息
 * @returns {object|null} 可用注册表项
 */
export const createEmbeddedRegistryEntry = (
  payload: Record<string, unknown> = {},
): EmbeddedRegistryEntry | null => {
  const iframe = resolveIframeFromTarget(payload)
  const origin = resolveOriginFromTarget(payload)
  const appType = resolveAppTypeFromTarget(payload)

  if (!iframe || !origin || !appType) {
    return null
  }

  return {
    iframe,
    origin,
    appType,
    project: (payload.project as Record<string, unknown>) ?? null,
    tabKey: payload.tabKey ?? null,
    pageId: payload.pageId ?? null,
  }
}

/**
 * 将嵌入应用登记到宿主注册表。
 * 注册失败时直接返回 null，调用方据此跳过后续逻辑。
 * @param {Map<object, object>} registry - 嵌入注册表
 * @param {object} payload - 注册信息
 * @returns {object|null} 最终登记的注册表项
 */
export const registerEmbeddedIframe = (
  registry: Map<unknown, EmbeddedRegistryEntry> | null | undefined,
  payload: Record<string, unknown> = {},
): EmbeddedRegistryEntry | null => {
  if (!registry?.set) return null

  const entry = createEmbeddedRegistryEntry(payload)
  if (!entry) return null

  registry.set(entry.iframe, entry)
  return entry
}

/**
 * 从宿主注册表移除嵌入应用。
 * @param {Map<object, object>} registry - 嵌入注册表
 * @param {object} iframe - iframe 节点
 */
export const unregisterEmbeddedIframe = (
  registry: Map<unknown, EmbeddedRegistryEntry> | null | undefined,
  iframe: unknown,
): void => {
  registry?.delete?.(iframe)
}

/**
 * 按 message source + origin 匹配已登记的嵌入应用。
 * 只有完全匹配的 iframe 才允许参与 bootstrap/auth 同步，避免未知窗口冒充。
 * @param {Map<object, object>} registry - 嵌入注册表
 * @param {object} source - message source
 * @param {string} origin - message origin
 * @returns {object|null} 匹配到的注册表项
 */
export const findEmbeddedEntryByMessageSource = (
  registry: Map<unknown, EmbeddedRegistryEntry> | null | undefined,
  source: unknown,
  origin: string | undefined,
): EmbeddedRegistryEntry | null => {
  if (!registry?.values || !source || !origin) {
    return null
  }

  for (const entry of registry.values()) {
    if (entry?.iframe?.contentWindow === source && entry.origin === origin) {
      return entry
    }
  }

  return null
}

/**
 * 判断 iframe 是否属于设计中心。
 * @param {object} iframe - iframe 节点
 * @returns {boolean} 是否为设计中心 iframe
 */
export const isDesignerEmbeddedIframe = (iframe: unknown): boolean =>
  resolveAppTypeFromTarget(iframe) === 'designer'

/**
 * 向嵌入 iframe 广播消息。
 * @param {Iterable<object>|object} targets - 注册表项集合、NodeList 或 document-like 对象
 * @param {object} message - 要广播的消息
 */
export const broadcastToEmbeddedIframes = (
  targets: unknown,
  message: Record<string, unknown>,
): void => {
  for (const target of normalizeEmbeddedTargets(targets)) {
    postMessageToEmbeddedTarget(target, message)
  }
}

/**
 * 仅向设计中心 iframe 广播消息。
 * @param {Iterable<object>|object} targets - 注册表项集合、NodeList 或 document-like 对象
 * @param {object} message - 要广播的消息
 */
export const broadcastToDesignerEmbeddedIframes = (
  targets: unknown,
  message: Record<string, unknown>,
): void => {
  for (const target of normalizeEmbeddedTargets(targets)) {
    if (resolveAppTypeFromTarget(target) !== 'designer') continue
    postMessageToEmbeddedTarget(target, message)
  }
}

/**
 * 从文档中筛选嵌入 iframe。
 * @param {object} documentLike - 类文档对象
 * @returns {Array<object>} iframe 列表
 */
export const getEmbeddedIframes = (documentLike: DocumentLike): EmbeddedTarget[] =>
  normalizeEmbeddedTargets(documentLike)

/**
 * 同步设计中心语言到嵌入 iframe。
 * @param {Iterable<object>|object} targets - 注册表项集合、NodeList 或 document-like 对象
 * @param {string} locale - 语言
 */
export const syncDesignerLocaleToEmbeddedIframes = (targets: unknown, locale: string): void => {
  broadcastToDesignerEmbeddedIframes(
    targets,
    createEmbeddedUpdateMessage('LOCALE_UPDATE', 'locale', locale),
  )
}

/**
 * 同步语言到全部嵌入 iframe。
 * 设计中心与数据中心都依赖宿主语言，不能再只广播给 designer。
 * @param {Iterable<object>|object} targets - 注册表项集合、NodeList 或 document-like 对象
 * @param {string} locale - 语言
 */
export const syncLocaleToEmbeddedIframes = (targets: unknown, locale: string): void => {
  broadcastToEmbeddedIframes(
    targets,
    createEmbeddedUpdateMessage('LOCALE_UPDATE', 'locale', locale),
  )
}

/**
 * 统一提取 message 载荷。
 * 历史调用方可能把业务字段放在根层，也可能放在 payload 里，这里统一兼容。
 * @param {object} data - message data
 * @returns {object} 业务载荷
 */
const resolveEmbeddedMessagePayload = (
  data: Record<string, unknown> = {},
): Record<string, unknown> => {
  if (data?.payload && typeof data.payload === 'object' && !Array.isArray(data.payload)) {
    return data.payload as Record<string, unknown>
  }
  return data
}

/**
 * 处理来自嵌入应用的 window message。
 * 这里不直接依赖 Vue/Pinia，只做匹配与分发，便于 Node 环境测试。
 * @param {object} options - 处理参数
 * @returns {object} 处理结果
 */
export const handleEmbeddedWindowMessage = (
  options: HandleEmbeddedMessageOptions = {},
):
  | { handled: false; reason: 'unsupported-message' | 'unregistered-source' }
  | {
      handled: true
      type: EmbeddedMessageType
      entry: EmbeddedRegistryEntry
      responseMessage?: Record<string, unknown>
    } => {
  const { event, registry, resolveBootstrapState, onAuthRefreshed, onAuthExpired } = options
  const data = event?.data
  const type = data?.type

  if (typeof type !== 'string' || !SUPPORTED_EMBEDDED_MESSAGE_TYPES.has(type)) {
    return { handled: false, reason: 'unsupported-message' }
  }

  const entry = findEmbeddedEntryByMessageSource(registry, event?.source, event?.origin)
  if (!entry) {
    return { handled: false, reason: 'unregistered-source' }
  }

  const payload = resolveEmbeddedMessagePayload(data)
  const messageType = type as EmbeddedMessageType
  if (messageType === 'APP_BOOTSTRAP_REQUEST') {
    const hostState = resolveBootstrapState?.(entry, payload) ?? {}
    const project = entry.project
    const bootstrapPayload = createBootstrapResponse(entry.appType, {
      ...hostState,
      projectId: hostState.projectId ?? project?.id ?? project?.projectId ?? null,
      tenantId: hostState.tenantId ?? project?.tenantId ?? null,
      tabKey: hostState.tabKey ?? entry.tabKey ?? null,
      pageId: hostState.pageId ?? entry.pageId ?? null,
    })

    const responseMessage: Record<string, unknown> = {
      type: 'APP_BOOTSTRAP_RESPONSE',
      payload: bootstrapPayload,
    }

    if (payload?.requestId) {
      responseMessage.requestId = payload.requestId
    }

    const postMessage = options.postMessage ?? postMessageToEmbeddedTarget
    postMessage(entry, responseMessage)

    return {
      handled: true,
      type: messageType,
      entry,
      responseMessage,
    }
  }

  if (messageType === 'AUTH_REFRESHED') {
    onAuthRefreshed?.(payload, entry)
    return {
      handled: true,
      type: messageType,
      entry,
    }
  }

  onAuthExpired?.(payload, entry)
  return {
    handled: true,
    type: messageType,
    entry,
  }
}
