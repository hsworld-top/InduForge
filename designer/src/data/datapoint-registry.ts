/**
 * 数据点注册表
 */

import type { DatapointStatus } from './types.ts'
import { EventEmitter } from '../editor-core/utils/EventEmitter.ts'

export type { DatapointStatus }

export interface DatapointMeta {
  id: string
  path: string
  provider?: string
  dataType?: string
  unit?: string
  description?: string
  status?: DatapointStatus
  lastValue?: unknown
  lastUpdated?: number
}

export const DatapointRegistryEvents = {
  REGISTERED: 'registered',
  UNREGISTERED: 'unregistered',
  STATUS_CHANGED: 'statusChanged',
  VALUE_UPDATED: 'valueUpdated',
} as const

export class DatapointRegistry extends EventEmitter {
  private _datapoints: Map<string, DatapointMeta>
  private _byProvider: Map<string, Set<string>>
  private _subscribers: Map<string, Set<string>>

  constructor() {
    super()
    this._datapoints = new Map()
    this._byProvider = new Map()
    this._subscribers = new Map()
  }

  register(meta: DatapointMeta): void {
    if (!meta.id || !meta.path) {
      throw new Error('数据点元信息缺少必要字段 (id, path)')
    }

    this._datapoints.set(meta.id, {
      ...meta,
      status: meta.status || 'unknown',
      lastUpdated: Date.now(),
    })

    const provider = meta.provider || 'default'
    if (!this._byProvider.has(provider)) {
      this._byProvider.set(provider, new Set())
    }
    this._byProvider.get(provider)!.add(meta.id)

    this.emit(DatapointRegistryEvents.REGISTERED, meta)
  }

  registerAll(metas: DatapointMeta[]): void {
    for (const meta of metas) {
      this.register(meta)
    }
  }

  unregister(id: string): void {
    const meta = this._datapoints.get(id)
    if (meta) {
      const providerSet = this._byProvider.get(meta.provider || 'default')
      if (providerSet) {
        providerSet.delete(id)
      }

      this._datapoints.delete(id)
      this._subscribers.delete(id)

      this.emit(DatapointRegistryEvents.UNREGISTERED, { id })
    }
  }

  get(id: string): DatapointMeta | undefined {
    return this._datapoints.get(id)
  }

  getByPath(path: string): DatapointMeta | undefined {
    for (const meta of this._datapoints.values()) {
      if (meta.path === path) {
        return meta
      }
    }
    return undefined
  }

  has(id: string): boolean {
    return this._datapoints.has(id)
  }

  getAll(): DatapointMeta[] {
    return Array.from(this._datapoints.values())
  }

  getByProvider(provider: string): DatapointMeta[] {
    const ids = this._byProvider.get(provider)
    if (!ids) return []

    return Array.from(ids)
      .map((pid) => this._datapoints.get(pid))
      .filter((m): m is DatapointMeta => Boolean(m))
  }

  updateStatus(id: string, status: DatapointStatus, reason?: string): void {
    const meta = this._datapoints.get(id)
    if (meta) {
      const oldStatus = meta.status
      meta.status = status
      meta.lastUpdated = Date.now()

      if (oldStatus !== status) {
        this.emit(DatapointRegistryEvents.STATUS_CHANGED, {
          id,
          path: meta.path,
          oldStatus,
          newStatus: status,
          reason,
        })
      }
    }
  }

  updateValue(id: string, value: unknown): void {
    const meta = this._datapoints.get(id)
    if (meta) {
      meta.lastValue = value
      meta.lastUpdated = Date.now()

      if (meta.status === 'unknown') {
        meta.status = 'active'
      }

      this.emit(DatapointRegistryEvents.VALUE_UPDATED, {
        id,
        path: meta.path,
        value,
      })
    }
  }

  subscribe(datapointId: string, componentId: string): void {
    if (!this._subscribers.has(datapointId)) {
      this._subscribers.set(datapointId, new Set())
    }
    this._subscribers.get(datapointId)!.add(componentId)
  }

  unsubscribe(datapointId: string, componentId: string): void {
    const subs = this._subscribers.get(datapointId)
    if (subs) {
      subs.delete(componentId)
    }
  }

  getSubscribers(datapointId: string): string[] {
    const subs = this._subscribers.get(datapointId)
    return subs ? Array.from(subs) : []
  }

  getComponentSubscriptions(componentId: string): string[] {
    const result: string[] = []
    for (const [datapointId, subs] of this._subscribers) {
      if (subs.has(componentId)) {
        result.push(datapointId)
      }
    }
    return result
  }

  clearComponentSubscriptions(componentId: string): void {
    for (const subs of this._subscribers.values()) {
      subs.delete(componentId)
    }
  }

  search(keyword: string): DatapointMeta[] {
    const lowerKeyword = keyword.toLowerCase()
    return this.getAll().filter(
      (m) =>
        m.path.toLowerCase().includes(lowerKeyword) ||
        m.description?.toLowerCase().includes(lowerKeyword),
    )
  }

  getInvalidDatapoints(): DatapointMeta[] {
    return this.getAll().filter((m) => m.status === 'invalid')
  }

  getUnknownDatapoints(): DatapointMeta[] {
    return this.getAll().filter((m) => m.status === 'unknown')
  }

  clear(): void {
    this._datapoints.clear()
    this._byProvider.clear()
    this._subscribers.clear()
  }
}

export const datapointRegistry = new DatapointRegistry()

export default DatapointRegistry
