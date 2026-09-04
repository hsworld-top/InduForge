import { initSocket } from './socket'
import type { Socket } from 'socket.io-client'

export type OpsTopic = 'nodes' | 'environments' | 'deployments' | 'events'
export interface OpsScope {
  topics: OpsTopic[]
  entityIds?: string[]
}
export interface OpsChange {
  epoch: string
  sequence: number
  topics: OpsTopic[]
  entityIds?: string[]
  timestamp?: string
  resync?: boolean
  terminal?: boolean
}
export interface OpsRefreshHint {
  full: boolean
  entityIds: string[]
}
interface Watcher {
  id: string
  scope: OpsScope | null
  epoch: string
  sequence: number
  ready: boolean
  change: (event: OpsChange) => void
  connection: (ready: boolean, topics?: OpsTopic[]) => void
}

const watchers = new Set<Watcher>()
let shared: Socket | undefined
let nextId = 0
const watch = (item: Watcher) => {
  if (shared?.connected && item.scope)
    shared.emit('ops:watch', { subscriptionId: item.id, ...item.scope })
}
const connected = () => {
  for (const item of watchers) {
    item.ready = false
    item.connection(false)
    watch(item)
  }
}
const disconnected = () => {
  for (const item of watchers) {
    item.ready = false
    if (item.scope) item.connection(false)
  }
}
const denied = (value: unknown) => {
  const event = value as { code?: string; data?: { code?: string } } | undefined
  if ((event?.code || event?.data?.code) !== 'AUTH_FORBIDDEN') return
  for (const item of watchers) {
    item.ready = false
    if (item.scope) item.connection(false, [])
  }
}
const ready = (event: {
  subscriptionId?: string
  epoch?: string
  sequence?: number
  topics?: OpsTopic[]
}) => {
  for (const item of watchers) {
    if (
      item.id !== event.subscriptionId ||
      !item.scope ||
      typeof event.epoch !== 'string' ||
      typeof event.sequence !== 'number'
    )
      continue
    item.epoch = event.epoch
    item.sequence = event.sequence
    item.ready = Boolean(event.topics?.length)
    item.scope = {
      ...item.scope,
      topics: (event.topics || []).filter((topic) => item.scope?.topics.includes(topic)),
    }
    item.connection(item.ready, item.scope.topics)
    if (!item.ready) continue
    item.change({
      epoch: event.epoch,
      sequence: event.sequence,
      topics: item.scope.topics,
      resync: true,
    })
  }
}
const changed = (event: OpsChange) => {
  if (
    !event ||
    typeof event.epoch !== 'string' ||
    !Number.isFinite(event.sequence) ||
    !Array.isArray(event.topics)
  )
    return
  for (const item of watchers) {
    if (!item.scope || !item.ready) continue
    if (item.epoch === event.epoch && event.sequence <= item.sequence) continue
    const epochChanged = item.epoch !== event.epoch
    item.epoch = event.epoch
    item.sequence = event.sequence
    if (
      !epochChanged &&
      !event.resync &&
      !event.topics.some((topic) => item.scope?.topics.includes(topic))
    )
      continue
    if (
      !epochChanged &&
      !event.resync &&
      item.scope.entityIds?.length &&
      event.entityIds?.length &&
      !event.entityIds.some((id) => item.scope?.entityIds?.includes(id))
    )
      continue
    item.change(
      epochChanged || event.resync ? { ...event, topics: item.scope.topics, resync: true } : event,
    )
  }
}

/** 运维订阅引用计数仅管理自己的监听器，不能断开工程管理等消费者共用的Socket。 */
export function acquireOpsWatch(callbacks: Pick<Watcher, 'change' | 'connection'>) {
  if (!shared) {
    shared = initSocket()
    shared.on('connect', connected)
    shared.on('disconnect', disconnected)
    shared.on('ops:ready', ready)
    shared.on('ops:change', changed)
    shared.on('ops:auth', denied)
    shared.on('connect_error', denied)
  }
  const item: Watcher = {
    id: `ide-ops-${++nextId}`,
    scope: null,
    epoch: '',
    sequence: -1,
    ready: false,
    ...callbacks,
  }
  watchers.add(item)
  return {
    setScope(scope: OpsScope | null) {
      if (item.scope && shared?.connected) shared.emit('ops:unwatch', { subscriptionId: item.id })
      // 每次恢复使用新ID，旧ready无法激活已隐藏或被替换的订阅。
      item.id = `ide-ops-${++nextId}`
      item.scope = scope
      item.ready = false
      item.epoch = ''
      item.sequence = -1
      callbacks.connection(false)
      watch(item)
    },
    release() {
      if (item.scope && shared?.connected) shared.emit('ops:unwatch', { subscriptionId: item.id })
      watchers.delete(item)
      if (!watchers.size && shared) {
        shared.off('connect', connected)
        shared.off('disconnect', disconnected)
        shared.off('ops:ready', ready)
        shared.off('ops:change', changed)
        shared.off('ops:auth', denied)
        shared.off('connect_error', denied)
        shared = undefined
      }
    },
  }
}

/** 事件只携带失效提示；串行、可取消的HTTP快照保证权限/分页契约仍是唯一数据来源。 */
export function createOpsRealtimeMonitor(options: {
  snapshot: (topics: OpsTopic[], signal: AbortSignal, hint: OpsRefreshHint) => Promise<void>
  status: (state: 'connecting' | 'online' | 'offline' | 'stale' | 'forbidden') => void
  random?: () => number
}) {
  let scope: OpsScope | null = null
  let online = false
  let disposed = false
  let fallbackAttempt = 0
  let snapshotFailed = false
  let timer: ReturnType<typeof setTimeout> | undefined
  let batch: ReturnType<typeof setTimeout> | undefined
  let controller: AbortController | undefined
  let inFlight: Promise<void> | undefined
  let nextAllowedAt = 0
  const pending = new Set<OpsTopic>()
  const changedIds = new Set<string>()
  let fullRefresh = false
  const clear = () => {
    if (timer) clearTimeout(timer)
    if (batch) clearTimeout(batch)
    timer = undefined
    batch = undefined
  }
  const scheduleFallback = () => {
    if (timer) return
    if (!scope || (online && !snapshotFailed) || disposed) return
    const base = Math.min(15000 * 2 ** Math.min(fallbackAttempt++, 2), 60000)
    timer = setTimeout(
      () => {
        timer = undefined
        if (!online) options.status('offline')
        void refresh()
      },
      base * (0.8 + (options.random || Math.random)() * 0.4),
    )
  }
  async function drain(): Promise<void> {
    if (!scope || disposed) return
    if (inFlight) {
      await inFlight
      if (pending.size) await drain()
      return
    }
    if (!pending.size) return
    if (Date.now() < nextAllowedAt) {
      if (batch) clearTimeout(batch)
      batch = setTimeout(() => {
        batch = undefined
        void drain()
      }, nextAllowedAt - Date.now())
      return
    }
    const topics = [...pending]
    pending.clear()
    const hint = { full: fullRefresh, entityIds: [...changedIds] }
    fullRefresh = false
    changedIds.clear()
    nextAllowedAt = Date.now() + 1000
    const current = new AbortController()
    controller = current
    inFlight = options
      .snapshot(topics, current.signal, hint)
      .then(() => {
        if (!current.signal.aborted) {
          snapshotFailed = false
          if (online && timer) {
            clearTimeout(timer)
            timer = undefined
            fallbackAttempt = 0
          }
          options.status(online ? 'online' : 'offline')
        }
      })
      .catch(() => {
        if (!current.signal.aborted) {
          snapshotFailed = true
          options.status('stale')
        }
      })
      .finally(() => {
        inFlight = undefined
        if (controller === current) controller = undefined
      })
    await inFlight
    if (pending.size) await drain()
    else scheduleFallback()
  }
  function enqueue(topics: OpsTopic[], immediate: boolean, event?: OpsChange) {
    if (!scope || disposed) return
    for (const topic of topics) if (scope.topics.includes(topic)) pending.add(topic)
    if (!pending.size) return
    if (!event || event.resync || !event.entityIds?.length) fullRefresh = true
    else
      for (const id of event.entityIds) {
        if (changedIds.size < 100) changedIds.add(id)
        else {
          fullRefresh = true
          changedIds.clear()
          break
        }
      }
    // 普通事件只标脏：高频心跳不能不断取消慢请求导致快照饥饿。
    if (immediate) {
      nextAllowedAt = 0
      return drain()
    }
    if (!batch)
      batch = setTimeout(() => {
        batch = undefined
        void drain()
      }, 120)
  }
  async function refresh(topics = scope?.topics || []) {
    await enqueue(topics, true)
  }
  const subscription = acquireOpsWatch({
    connection(ready, topics) {
      if (topics && scope) {
        scope = { ...scope, topics }
        if (!topics.length) {
          scope = null
          clear()
          pending.clear()
          controller?.abort()
          options.status('forbidden')
          return
        }
      }
      online = ready
      if (!scope || disposed) return
      options.status(ready ? 'online' : 'connecting')
      if (ready) {
        fallbackAttempt = 0
        if (timer) clearTimeout(timer)
        timer = undefined
      } else scheduleFallback()
    },
    change(event) {
      void enqueue(
        event.resync ? scope?.topics || [] : event.topics,
        Boolean(event.resync || event.terminal),
        event,
      )
    },
  })
  return {
    refresh,
    setScope(value: OpsScope | null) {
      if (JSON.stringify(value) === JSON.stringify(scope)) return
      clear()
      controller?.abort()
      pending.clear()
      changedIds.clear()
      fullRefresh = false
      scope = value
      online = false
      fallbackAttempt = 0
      subscription.setScope(value)
      if (value) void refresh()
    },
    dispose() {
      disposed = true
      scope = null
      clear()
      pending.clear()
      controller?.abort()
      subscription.release()
    },
  }
}
