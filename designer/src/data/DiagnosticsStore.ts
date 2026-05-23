/**
 * DiagnosticsStore - 数据点状态追踪
 * 管理和追踪所有数据点的状态
 */

import { ApiRequestError, normalizeApiEnvelope } from './types.ts'
import type {
  DatapointStatus,
  DatapointStatusInfo,
  DiagnosticInfo,
  DiagnosticsSummary,
} from './types.ts'
import { EventEmitter } from '../editor-core/utils/EventEmitter.ts'

function isRecord(value: unknown): value is Record<string, unknown> {
  return value !== null && typeof value === 'object'
}

interface DiagnosticsStoreOptions {
  apiBaseUrl?: string
  cacheExpireTime?: number
  batchSize?: number
}

interface CachedStatus {
  info: DatapointStatusInfo
  timestamp: number
}

function buildDiagnosticInfo(
  path: string,
  info: DatapointStatusInfo,
  bindingCount: number,
): DiagnosticInfo {
  const diagnostic: DiagnosticInfo = {
    path,
    status: info.status,
    dataType: info.dataType,
    bindingCount,
  }
  if (info.statusReason) {
    diagnostic.statusReason = info.statusReason
  }
  return diagnostic
}

/**
 * 诊断存储类
 */
export class DiagnosticsStore extends EventEmitter {
  private _apiBaseUrl: string

  private _cacheExpireTime: number

  private _batchSize: number

  private _cache: Map<string, CachedStatus>

  private _nodeBindings: Map<string, Set<string>>

  private _datapointNodes: Map<string, Set<string>>

  private _pendingPaths: Set<string>

  private _batchTimer: ReturnType<typeof setTimeout> | null

  private _batchDelay: number

  /**
   * 创建诊断存储
   * @param {object} [options] - 配置选项
   * @param {string} [options.apiBaseUrl] - API 基础路径
   * @param {number} [options.cacheExpireTime] - 缓存过期时间（毫秒）
   * @param {number} [options.batchSize] - 批量查询大小
   */
  constructor(options: DiagnosticsStoreOptions = {}) {
    super()

    this._apiBaseUrl = options.apiBaseUrl || '/api/v1'
    this._cacheExpireTime = options.cacheExpireTime ?? 30000
    this._batchSize = options.batchSize ?? 50
    this._cache = new Map()
    this._nodeBindings = new Map()
    this._datapointNodes = new Map()
    this._pendingPaths = new Set()
    this._batchTimer = null
    this._batchDelay = 100
  }

  /**
   * 获取数据点状态
   * @param {string} path - 数据点路径
   * @param {boolean} [forceRefresh] - 是否强制刷新
   * @returns {Promise<DatapointStatusInfo>}
   */
  async getStatus(path: string, forceRefresh = false): Promise<DatapointStatusInfo> {
    if (!forceRefresh) {
      const cached = this._getFromCache(path)
      if (cached) return cached
    }

    this._addToPending(path)

    return new Promise((resolve) => {
      const handler = (...args: unknown[]) => {
        const data = args[0] as DatapointStatusInfo | undefined
        if (!data) return
        if (data.path === path) {
          this.off('statusUpdated', handler)
          resolve(data)
        }
      }
      this.on('statusUpdated', handler)

      setTimeout(() => {
        this.off('statusUpdated', handler)
        resolve(this._getFromCache(path) || this._createUnknownStatus(path))
      }, 5000)
    })
  }

  /**
   * 批量获取数据点状态
   * @param {string[]} paths - 数据点路径数组
   * @param {boolean} [forceRefresh] - 是否强制刷新
   * @returns {Promise<Map<string, DatapointStatusInfo>>}
   */
  async getStatusBatch(
    paths: string[],
    forceRefresh = false,
  ): Promise<Map<string, DatapointStatusInfo>> {
    const results = new Map<string, DatapointStatusInfo>()
    const needFetch: string[] = []

    for (const path of paths) {
      if (!forceRefresh) {
        const cached = this._getFromCache(path)
        if (cached) {
          results.set(path, cached)
          continue
        }
      }
      needFetch.push(path)
    }

    if (needFetch.length > 0) {
      await this._fetchStatusBatch(needFetch)

      for (const path of needFetch) {
        const cached = this._getFromCache(path)
        results.set(path, cached || this._createUnknownStatus(path))
      }
    }

    return results
  }

  /**
   * 从缓存获取状态
   * @param {string} path - 数据点路径
   * @returns {DatapointStatusInfo | null}
   * @private
   */
  private _getFromCache(path: string): DatapointStatusInfo | null {
    const cached = this._cache.get(path)
    if (!cached) return null

    if (Date.now() - cached.timestamp > this._cacheExpireTime) {
      this._cache.delete(path)
      return null
    }

    return cached.info
  }

  /**
   * 创建未知状态
   * @param {string} path - 数据点路径
   * @returns {DatapointStatusInfo}
   * @private
   */
  private _createUnknownStatus(path: string): DatapointStatusInfo {
    return {
      path,
      status: 'unknown',
      statusReason: '数据点未找到或未配置',
      dataType: 'unknown',
    }
  }

  /**
   * 添加到待查询队列
   * @param {string} path - 数据点路径
   * @private
   */
  private _addToPending(path: string): void {
    this._pendingPaths.add(path)
    this._scheduleBatchFetch()
  }

  /**
   * 调度批量查询
   * @private
   */
  private _scheduleBatchFetch(): void {
    if (this._batchTimer) return

    this._batchTimer = setTimeout(async () => {
      this._batchTimer = null

      if (this._pendingPaths.size === 0) return

      const paths = Array.from(this._pendingPaths)
      this._pendingPaths.clear()

      for (let i = 0; i < paths.length; i += this._batchSize) {
        const batch = paths.slice(i, i + this._batchSize)
        await this._fetchStatusBatch(batch)
      }
    }, this._batchDelay)
  }

  /**
   * 批量查询状态（API 调用）
   * @param {string[]} paths - 数据点路径数组
   * @private
   */
  private async _fetchStatusBatch(paths: string[]): Promise<void> {
    try {
      const response = await fetch(`${this._apiBaseUrl}/datapoints/status`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ paths }),
      })

      let payload: unknown = null
      try {
        payload = await response.json()
      } catch {
        payload = null
      }

      const envelope = normalizeApiEnvelope<DatapointStatusInfo[]>(payload)

      if (!response.ok) {
        throw new ApiRequestError({
          code: envelope?.code,
          msg: envelope?.msg || `HTTP ${response.status}`,
          reqId: envelope?.reqId,
          status: response.status,
          data: envelope?.data,
          isBusinessError: false,
        })
      }

      if (envelope && envelope.code !== 0) {
        throw new ApiRequestError({
          code: envelope.code,
          msg: envelope.msg || '查询数据点状态失败',
          reqId: envelope.reqId,
          status: response.status,
          data: envelope.data,
          isBusinessError: true,
        })
      }

      let statusList: DatapointStatusInfo[] = []
      if (envelope) {
        statusList = Array.isArray(envelope.data) ? envelope.data : []
      } else if (Array.isArray(payload)) {
        // 兼容极旧链路：接口直接返回状态数组。
        statusList = payload as DatapointStatusInfo[]
      } else if (isRecord(payload) && typeof payload.success === 'boolean') {
        // 兼容旧包络：{ success, data, message }
        if (!payload.success) {
          throw new ApiRequestError({
            msg: typeof payload.message === 'string' ? payload.message : '查询数据点状态失败',
            status: response.status,
            data: payload.data,
            isBusinessError: true,
          })
        }

        if (!Array.isArray(payload.data)) {
          throw new ApiRequestError({
            msg: '查询数据点状态失败: data 不是数组',
            status: response.status,
            data: payload.data,
            isBusinessError: false,
          })
        }
        statusList = payload.data as DatapointStatusInfo[]
      } else {
        throw new ApiRequestError({
          msg: '查询数据点状态失败: 响应格式不受支持',
          status: response.status,
          data: payload,
          isBusinessError: false,
        })
      }

      for (const info of statusList) {
        this._updateCache(info)
      }
    } catch (error) {
      console.error('Failed to fetch datapoint status:', error)

      for (const path of paths) {
        this._updateCache({
          path,
          status: 'unknown',
          statusReason: `查询失败: ${error instanceof Error ? error.message : String(error)}`,
          dataType: 'unknown',
        })
      }
    }
  }

  /**
   * 更新缓存
   * @param {DatapointStatusInfo} info - 状态信息
   * @private
   */
  private _updateCache(info: DatapointStatusInfo): void {
    const oldInfo = this._cache.get(info.path)?.info

    this._cache.set(info.path, {
      info,
      timestamp: Date.now(),
    })

    if (!oldInfo || oldInfo.status !== info.status) {
      this.emit('statusUpdated', info)
      this.emit('statusChange', {
        path: info.path,
        oldStatus: oldInfo?.status,
        newStatus: info.status,
        info,
      })
    }
  }

  /**
   * 注册节点绑定
   * @param {string} nodeId - 节点 ID
   * @param {string[]} datapointPaths - 数据点路径数组
   */
  registerNodeBindings(nodeId: string, datapointPaths: string[]): void {
    this.unregisterNodeBindings(nodeId)
    this._nodeBindings.set(nodeId, new Set(datapointPaths))

    for (const path of datapointPaths) {
      if (!this._datapointNodes.has(path)) {
        this._datapointNodes.set(path, new Set())
      }
      this._datapointNodes.get(path)?.add(nodeId)
    }
  }

  /**
   * 取消注册节点绑定
   * @param {string} nodeId - 节点 ID
   */
  unregisterNodeBindings(nodeId: string): void {
    const paths = this._nodeBindings.get(nodeId)
    if (!paths) return

    for (const path of paths) {
      this._datapointNodes.get(path)?.delete(nodeId)
    }

    this._nodeBindings.delete(nodeId)
  }

  /**
   * 获取节点绑定的数据点
   * @param {string} nodeId - 节点 ID
   * @returns {string[]}
   */
  getNodeBindings(nodeId: string): string[] {
    const paths = this._nodeBindings.get(nodeId)
    return paths ? Array.from(paths) : []
  }

  /**
   * 获取使用某数据点的节点
   * @param {string} path - 数据点路径
   * @returns {string[]}
   */
  getDatapointNodes(path: string): string[] {
    const nodes = this._datapointNodes.get(path)
    return nodes ? Array.from(nodes) : []
  }

  /**
   * 获取诊断摘要
   * @returns {DiagnosticsSummary}
   */
  getSummary(): DiagnosticsSummary & { issues: DiagnosticInfo[] } {
    const issues: DiagnosticInfo[] = []
    let active = 0
    let invalid = 0
    let unknown = 0

    for (const [path, cached] of this._cache.entries()) {
      const { info } = cached
      switch (info.status) {
        case 'active':
          active++
          break
        case 'invalid':
          invalid++
          issues.push(buildDiagnosticInfo(path, info, this.getDatapointNodes(path).length))
          break
        case 'unknown':
          unknown++
          issues.push(buildDiagnosticInfo(path, info, this.getDatapointNodes(path).length))
          break
      }
    }

    return {
      total: this._cache.size,
      active,
      invalid,
      unknown,
      issues,
    }
  }

  /**
   * 获取所有问题数据点
   * @returns {DiagnosticInfo[]}
   */
  getIssues(): DiagnosticInfo[] {
    const issues: DiagnosticInfo[] = []

    for (const [path, cached] of this._cache.entries()) {
      const { info } = cached
      if (info.status !== 'active') {
        issues.push(buildDiagnosticInfo(path, info, this.getDatapointNodes(path).length))
      }
    }

    return issues
  }

  /**
   * 获取特定状态的数据点
   * @param {DatapointStatus} status - 状态
   * @returns {string[]}
   */
  getPathsByStatus(status: DatapointStatus): string[] {
    const paths: string[] = []
    for (const [path, cached] of this._cache.entries()) {
      if (cached.info.status === status) {
        paths.push(path)
      }
    }
    return paths
  }

  /**
   * 手动设置状态（用于 WebSocket 推送）
   * @param {DatapointStatusInfo} info - 状态信息
   */
  setStatus(info: DatapointStatusInfo): void {
    this._updateCache(info)
  }

  /**
   * 批量设置状态
   * @param {DatapointStatusInfo[]} infos - 状态信息数组
   */
  setStatusBatch(infos: DatapointStatusInfo[]): void {
    for (const info of infos) {
      this._updateCache(info)
    }
  }

  /**
   * 标记数据点为失效
   * @param {string} path - 数据点路径
   * @param {string} [reason] - 原因
   */
  markInvalid(path: string, reason?: string): void {
    this._updateCache({
      path,
      status: 'invalid',
      statusReason: reason || '手动标记为失效',
      dataType: 'unknown',
    })
  }

  /**
   * 标记数据点为活跃
   * @param {string} path - 数据点路径
   */
  markActive(path: string): void {
    this._updateCache({
      path,
      status: 'active',
      dataType: 'unknown',
    })
  }

  /**
   * 清除缓存
   */
  clearCache(): void {
    this._cache.clear()
    this.emit('cacheCleared')
  }

  /**
   * 清除过期缓存
   */
  clearExpiredCache(): void {
    const now = Date.now()
    for (const [path, cached] of this._cache.entries()) {
      if (now - cached.timestamp > this._cacheExpireTime) {
        this._cache.delete(path)
      }
    }
  }

  /**
   * 刷新所有缓存
   * @returns {Promise<void>}
   */
  async refreshAll(): Promise<void> {
    const paths = Array.from(this._cache.keys())
    if (paths.length === 0) return

    for (let i = 0; i < paths.length; i += this._batchSize) {
      const batch = paths.slice(i, i + this._batchSize)
      await this._fetchStatusBatch(batch)
    }
  }

  /**
   * 销毁
   */
  destroy(): void {
    if (this._batchTimer) {
      clearTimeout(this._batchTimer)
      this._batchTimer = null
    }

    this._cache.clear()
    this._nodeBindings.clear()
    this._datapointNodes.clear()
    this._pendingPaths.clear()

    this.removeAllListeners()
  }
}

/**
 * 创建诊断存储实例
 * @param {object} [options] - 配置选项
 * @returns {DiagnosticsStore}
 */
export function createDiagnosticsStore(options?: DiagnosticsStoreOptions): DiagnosticsStore {
  return new DiagnosticsStore(options)
}

export default DiagnosticsStore
