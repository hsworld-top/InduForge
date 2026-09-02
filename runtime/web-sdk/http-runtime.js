const RUNTIME_ERROR_CODE = 50031
const DEFAULT_TIMEOUT_MS = 10_000

function sdkResult(code, msg, data = null, reqId) {
  const result = { code, msg, data }
  if (reqId) result.reqId = reqId
  return result
}

function unsupported(methodName) {
  return sdkResult(RUNTIME_ERROR_CODE, `当前 Runtime V1 未提供 ${methodName}() 能力`)
}

function normalizeBaseUrl(value) {
  const baseUrl = value ?? '/api/v1/runtime'
  if (typeof baseUrl !== 'string' || !baseUrl.trim()) {
    throw new TypeError('createHttpRuntime(options) 的 baseUrl 必须是非空字符串')
  }
  return baseUrl.replace(/\/+$/, '')
}

function runtimeUrl(baseUrl, path) {
  return `${baseUrl}/${path.replace(/^\/+/, '')}`
}

function pointUrl(baseUrl, path) {
  if (!path) throw new TypeError('数据点 path 不能为空')
  return runtimeUrl(baseUrl, `points/${encodeURIComponent(path)}`)
}

function isEnvelope(value) {
  return (
    value &&
    typeof value === 'object' &&
    !Array.isArray(value) &&
    typeof value.code === 'number' &&
    typeof value.msg === 'string' &&
    Object.prototype.hasOwnProperty.call(value, 'data')
  )
}

function errorMessage(error, fallback) {
  return error instanceof Error && error.message ? error.message : fallback
}

function createHeaders(identity, authorization, extraHeaders) {
  const headers = new Headers(extraHeaders ?? {})
  if (identity?.deploymentId) headers.set('X-InduForge-Deployment-Id', identity.deploymentId)
  if (identity?.projectId) headers.set('X-InduForge-Project-Id', identity.projectId)
  if (authorization)
    headers.set(
      'Authorization',
      authorization.startsWith('Bearer ') ? authorization : `Bearer ${authorization}`,
    )
  return headers
}

function requestFailure(error, didTimeout, wasAborted) {
  if (didTimeout()) return sdkResult(RUNTIME_ERROR_CODE, 'Runtime API 请求超时')
  if (wasAborted()) return sdkResult(RUNTIME_ERROR_CODE, 'Runtime API 请求已取消')
  return sdkResult(RUNTIME_ERROR_CODE, errorMessage(error, 'Runtime API 请求失败'))
}

function combineAbortSignal(signal, timeoutMs) {
  const controller = new AbortController()
  let timedOut = false
  let externallyAborted = false
  const abortFromCaller = () => {
    externallyAborted = true
    controller.abort(signal?.reason)
  }
  if (signal?.aborted) abortFromCaller()
  else signal?.addEventListener?.('abort', abortFromCaller, { once: true })
  const timer = setTimeout(() => {
    timedOut = true
    controller.abort(new Error('Runtime API 请求超时'))
  }, timeoutMs)
  return {
    signal: controller.signal,
    didTimeout: () => timedOut,
    wasAborted: () => externallyAborted,
    dispose() {
      clearTimeout(timer)
      signal?.removeEventListener?.('abort', abortFromCaller)
    },
  }
}

function transformResult(result, transform) {
  if (result.code !== 0) return result
  try {
    return { ...result, data: transform(result.data) }
  } catch (error) {
    return sdkResult(
      RUNTIME_ERROR_CODE,
      errorMessage(error, 'Runtime API 响应格式无效'),
      null,
      result.reqId,
    )
  }
}

function createRequest(options, baseUrl) {
  const fetchImpl = options.fetch ?? globalThis.fetch
  if (typeof fetchImpl !== 'function')
    throw new Error('当前环境未提供 fetch，请通过 createHttpRuntime({ fetch }) 注入')
  const timeoutMs = options.timeoutMs ?? DEFAULT_TIMEOUT_MS
  if (!Number.isFinite(timeoutMs) || timeoutMs < 1)
    throw new TypeError('timeoutMs 必须是大于 0 的数字')

  return async (path, requestOptions = {}) => {
    const abort = combineAbortSignal(requestOptions.signal, requestOptions.timeoutMs ?? timeoutMs)
    try {
      const headers = createHeaders(
        options.identity,
        requestOptions.authorization,
        requestOptions.headers,
      )
      const body =
        requestOptions.json === undefined
          ? requestOptions.body
          : JSON.stringify(requestOptions.json)
      if (requestOptions.json !== undefined && !headers.has('Content-Type'))
        headers.set('Content-Type', 'application/json')
      const response = await fetchImpl(runtimeUrl(baseUrl, path), {
        method: requestOptions.method ?? 'GET',
        headers,
        body,
        credentials: requestOptions.credentials ?? options.credentials ?? 'include',
        signal: abort.signal,
      })
      let payload
      try {
        payload = await response.json()
      } catch (error) {
        return sdkResult(
          RUNTIME_ERROR_CODE,
          response.ok
            ? 'Runtime API 返回了无效 JSON'
            : `Runtime API 请求失败（HTTP ${response.status}）`,
        )
      }
      if (isEnvelope(payload)) return payload
      return sdkResult(RUNTIME_ERROR_CODE, `Runtime API 返回了无效响应（HTTP ${response.status}）`)
    } catch (error) {
      return requestFailure(error, abort.didTimeout, abort.wasAborted)
    } finally {
      abort.dispose()
    }
  }
}

function queryString(query) {
  if (!query || typeof query !== 'object') return ''
  const params = new URLSearchParams()
  for (const [key, value] of Object.entries(query)) {
    if (value !== undefined && value !== null) params.set(key, String(value))
  }
  const value = params.toString()
  return value ? `?${value}` : ''
}

function splitQueryAndRequestOptions(query) {
  if (!query || typeof query !== 'object') return { query, requestOptions: undefined }
  const { signal, timeoutMs, headers, credentials, ...params } = query
  const requestOptions =
    signal || timeoutMs || headers || credentials
      ? { signal, timeoutMs, headers, credentials }
      : undefined
  return { query: params, requestOptions }
}

function resolveWebSocketUrl(options, baseUrl) {
  if (options.wsUrl) return options.wsUrl
  if (/^https?:\/\//i.test(baseUrl))
    return baseUrl.replace(/^http/i, 'ws').replace(/\/api\/v1\/runtime\/?$/, '/ws/v1/points')
  if (typeof location !== 'undefined' && location.href)
    return new URL('/ws/v1/points', location.href).toString()
  return '/ws/v1/points'
}

function websocketError(message, details) {
  const error = new Error(message)
  Object.assign(error, details)
  return error
}

function createPointSubscription(options, baseUrl, path, handler, subscriptionOptions = {}) {
  const WebSocketImpl = options.WebSocket ?? globalThis.WebSocket
  if (typeof WebSocketImpl !== 'function')
    return Promise.resolve(
      sdkResult(
        RUNTIME_ERROR_CODE,
        '当前环境未提供 WebSocket，请通过 createHttpRuntime({ WebSocket }) 注入',
      ),
    )
  if (typeof handler !== 'function')
    throw new TypeError('point.subscribe(handler) 的 handler 必须是函数')
  const timeoutMs = subscriptionOptions.timeoutMs ?? options.timeoutMs ?? DEFAULT_TIMEOUT_MS
  if (!Number.isFinite(timeoutMs) || timeoutMs < 1)
    throw new TypeError('timeoutMs 必须是大于 0 的数字')

  return new Promise((resolve) => {
    let socket
    let settled = false
    let closedByClient = false
    let errorReported = false
    let timer
    const cleanupSignal = () => subscriptionOptions.signal?.removeEventListener?.('abort', abort)
    const reportError = (error) => {
      if (errorReported) return
      errorReported = true
      subscriptionOptions.onError?.(error)
    }
    const failBeforeSubscription = (message) => {
      if (settled) return
      settled = true
      clearTimeout(timer)
      cleanupSignal()
      resolve(sdkResult(RUNTIME_ERROR_CODE, message))
    }
    const abort = () => {
      closedByClient = true
      failBeforeSubscription('WebSocket 订阅已取消')
      socket?.close(1000, 'subscription cancelled')
    }
    const close = () => {
      if (closedByClient) return
      closedByClient = true
      socket?.close(1000, 'subscription closed')
    }
    try {
      socket = new WebSocketImpl(resolveWebSocketUrl(options, baseUrl))
    } catch (error) {
      failBeforeSubscription(errorMessage(error, '无法建立 WebSocket 连接'))
      return
    }
    timer = setTimeout(() => {
      closedByClient = true
      failBeforeSubscription('WebSocket 订阅超时')
      socket.close(1000, 'subscription timeout')
    }, timeoutMs)
    if (subscriptionOptions.signal?.aborted) {
      abort()
      return
    }
    subscriptionOptions.signal?.addEventListener?.('abort', abort, { once: true })
    socket.onopen = () => {
      try {
        socket.send(JSON.stringify({ action: 'subscribe', paths: [path] }))
      } catch (error) {
        failBeforeSubscription(errorMessage(error, '无法发送 WebSocket 订阅请求'))
      }
    }
    socket.onmessage = (event) => {
      if (typeof event.data !== 'string') return
      let message
      try {
        message = JSON.parse(event.data)
      } catch {
        reportError(websocketError('Runtime API 返回了无效 WebSocket 消息'))
        return
      }
      if (message.type === 'subscribed') {
        if (settled) return
        settled = true
        clearTimeout(timer)
        resolve(sdkResult(0, 'ok', close))
        return
      }
      if (message.type === 'point') {
        try {
          handler(message.data)
        } catch (error) {
          reportError(error instanceof Error ? error : websocketError(String(error)))
        }
        return
      }
      if (message.type === 'error') {
        const error = websocketError(message.msg || 'Runtime API 拒绝 WebSocket 订阅', {
          code: message.code,
        })
        if (!settled) failBeforeSubscription(error.message)
        else reportError(error)
        closedByClient = true
        socket.close()
      }
    }
    socket.onerror = () => {
      const error = websocketError('Runtime API WebSocket 连接错误')
      if (!settled) failBeforeSubscription(error.message)
      else if (!closedByClient) reportError(error)
    }
    socket.onclose = (event) => {
      clearTimeout(timer)
      cleanupSignal()
      const detail = { code: event.code, reason: event.reason, wasClean: event.wasClean }
      if (!settled) failBeforeSubscription(event.reason || 'Runtime API WebSocket 连接已关闭')
      else {
        if (!closedByClient && event.code !== 1000)
          reportError(websocketError(event.reason || 'Runtime API WebSocket 连接已关闭', detail))
        subscriptionOptions.onClose?.(detail)
      }
    }
  })
}

/**
 * 为发布后的工程创建 Runtime API 浏览器适配器。
 * 默认使用同源 /api/v1/runtime 与 /ws/v1/points，Gateway 负责注入工程身份头。
 */
export function createHttpRuntime(options = {}) {
  if (options == null || typeof options !== 'object' || Array.isArray(options)) {
    throw new TypeError('createHttpRuntime(options) 的 options 必须是对象')
  }
  const baseUrl = normalizeBaseUrl(options.baseUrl)
  const request = createRequest(options, baseUrl)
  const pointContracts = Object.create(null)
  const roles = []
  const cacheCatalog = (catalog) => {
    for (const key of Object.keys(pointContracts)) delete pointContracts[key]
    for (const point of catalog?.points ?? []) {
      if (point?.path) pointContracts[point.path] = point
    }
    return catalog
  }
  const getCatalog = async (requestOptions) =>
    transformResult(await request('catalog', requestOptions), cacheCatalog)
  const cacheSession = (session) => {
    roles.splice(0, roles.length, ...(Array.isArray(session?.roles) ? session.roles : []))
    return session
  }
  const currentPoint = async (path, requestOptions) =>
    request(`points/${encodeURIComponent(path)}`, requestOptions)
  const findCompute = async (ref, requestOptions) =>
    transformResult(await getCatalog(requestOptions), (catalog) => {
      const compute = (catalog.computes ?? []).find(
        (item) => item?.id === ref || item?.name === ref,
      )
      if (!compute) throw new Error('计算单元不存在')
      return compute
    })

  return Object.freeze({
    pointContracts,
    roles,
    catalog: Object.freeze({ get: getCatalog }),
    session: Object.freeze({
      establish: async (accessToken, requestOptions = {}) =>
        transformResult(
          await request('session', {
            ...requestOptions,
            method: 'POST',
            authorization: accessToken ?? options.accessToken,
          }),
          cacheSession,
        ),
      query: async (requestOptions) =>
        transformResult(await request('session', requestOptions), cacheSession),
      exit: async (requestOptions) => {
        const result = await request('session', { ...requestOptions, method: 'DELETE' })
        if (result.code === 0) roles.splice(0)
        return result
      },
    }),
    adapter: Object.freeze({
      get: async (path, requestOptions) =>
        transformResult(await currentPoint(path, requestOptions), (item) => item.value),
      read: currentPoint,
      peek: currentPoint,
      history: async (path, query) => {
        const parts = splitQueryAndRequestOptions(query)
        return transformResult(
          await request(
            `points/${encodeURIComponent(path)}/history${queryString(parts.query)}`,
            parts.requestOptions,
          ),
          (data) => data.items ?? [],
        )
      },
      subscribe: (path, handler, subscriptionOptions) =>
        createPointSubscription(options, baseUrl, path, handler, subscriptionOptions),
      set: (path, value, requestOptions) =>
        request(`points/${encodeURIComponent(path)}/write`, {
          ...requestOptions,
          method: 'POST',
          json: { value },
        }),
      refresh: () => unsupported('points.refresh'),
      run: () => unsupported('points.run'),
      execute: () => unsupported('points.execute'),
      publish: () => unsupported('points.publish'),
    }),
    alarmAdapter: Object.freeze({
      listCurrent: (query) => {
        const parts = splitQueryAndRequestOptions(query)
        return request(`alarms/current${queryString(parts.query)}`, parts.requestOptions)
      },
      getCurrent: () => unsupported('alarms.current.get'),
      listItems: () => unsupported('alarms.items.list'),
      getItem: () => unsupported('alarms.items.get'),
      getSettings: () => unsupported('alarms.settings.get'),
      updateSettings: () => unsupported('alarms.settings.update'),
      subscribeChanges: () => unsupported('alarms.changes.subscribe'),
      acknowledge: (id, input = {}) => {
        if (typeof id !== 'string' || !id.trim())
          throw new TypeError('alarms.actions.acknowledge(id, input) 的 id 必须是非空字符串')
        if (!input || typeof input !== 'object' || Array.isArray(input))
          throw new TypeError('alarms.actions.acknowledge(id, input) 的 input 必须是对象')
        const { requestOptions, ...body } = input
        return request(`alarms/${encodeURIComponent(id)}/acknowledge`, {
          ...(requestOptions ?? {}),
          method: 'POST',
          json: body,
        })
      },
      unacknowledge: () => unsupported('alarms.actions.unacknowledge'),
      forceClear: () => unsupported('alarms.actions.forceClear'),
      shelve: () => unsupported('alarms.actions.shelve'),
      unshelve: () => unsupported('alarms.actions.unshelve'),
      listHistory: () => unsupported('alarms.history.list'),
      getHistory: () => unsupported('alarms.history.get'),
    }),
    computeAdapter: Object.freeze({
      describe: findCompute,
      run: (ref, input, requestOptions) =>
        request(`computes/${encodeURIComponent(ref)}/run`, {
          ...requestOptions,
          method: 'POST',
          json: { input },
        }),
    }),
  })
}
