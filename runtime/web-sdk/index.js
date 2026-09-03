import { createPreviewBridgeRuntime } from './preview-bridge.js'
import { createHttpRuntime } from './http-runtime.js'

export { createHttpRuntime } from './http-runtime.js'

const POINT_OPERATIONS = new Set([
  'get',
  'read',
  'peek',
  'set',
  'subscribe',
  'history',
  'refresh',
  'run',
  'execute',
  'publish',
  'sub',
  'pub',
])
const POINT_METADATA = new Set([
  'id',
  'ref',
  'path',
  'name',
  'displayName',
  'dataType',
  'schema',
  'source',
  'status',
  'unit',
  'precision',
  'min',
  'max',
  'defaultValue',
  'tags',
  'attributes',
  'capabilities',
])

let configuredRuntime = null

function readBrowserRuntime() {
  if (typeof window === 'undefined') return {}
  // Designer 预览始终优先使用宿主桥接；发布页没有父宿主时回退到工程入口
  // 的同源 Runtime API。Bearer 凭据只由 project-gateway 在服务端注入。
  return window.__INDUFORGE_RUNTIME__ ?? createPreviewBridgeRuntime() ?? createHttpRuntime()
}

function resolveRuntime(runtime) {
  return runtime ?? configuredRuntime ?? readBrowserRuntime()
}

function sdkResult(code, msg, data, reqId) {
  const result = { code, msg, data }
  if (reqId) result.reqId = reqId
  return result
}

function normalizeResult(value) {
  if (
    value &&
    typeof value === 'object' &&
    !Array.isArray(value) &&
    typeof value.code === 'number' &&
    typeof value.msg === 'string' &&
    Object.prototype.hasOwnProperty.call(value, 'data')
  )
    return value
  return sdkResult(0, 'ok', value ?? null)
}

function unsupported(adapterName, methodName) {
  return sdkResult(50031, `${adapterName} 未提供 ${methodName}() 能力`, null)
}

async function callAdapter(target, methodName, adapterName, args) {
  const method = target?.[methodName]
  if (typeof method !== 'function') return unsupported(adapterName, methodName)
  try {
    return normalizeResult(await method.apply(target, args))
  } catch (error) {
    return sdkResult(50031, error instanceof Error ? error.message : String(error), null)
  }
}

function requireMethod(target, methodName, adapterName) {
  const method = target?.[methodName]
  if (typeof method !== 'function') throw new Error(`${adapterName} 缺少 ${methodName}() 方法`)
  return method.bind(target)
}

function pointContracts(runtimeProvider) {
  const runtime = resolveRuntime(runtimeProvider())
  return runtime.pointContracts ?? runtime.points?.contracts ?? {}
}

function pointContract(runtimeProvider, path) {
  const contracts = pointContracts(runtimeProvider)
  return contracts && typeof contracts === 'object' ? (contracts[path] ?? null) : null
}

function pointMetadata(runtimeProvider, path, property) {
  const contract = pointContract(runtimeProvider, path) ?? {}
  switch (property) {
    case 'ref':
    case 'path':
      return path
    case 'displayName':
      return contract.displayName ?? contract.name ?? null
    case 'source':
      return Object.freeze({
        type: contract.source?.type ?? contract.sourceType ?? null,
        id: contract.source?.id ?? contract.sourceId ?? null,
      })
    case 'tags':
      return Object.freeze(Array.isArray(contract.tags) ? [...contract.tags] : [])
    case 'attributes':
      return Object.freeze({ ...(contract.attributes ?? contract.attributeDefaults ?? {}) })
    case 'capabilities':
      return Object.freeze({ ...(contract.capabilities ?? {}) })
    default:
      return contract[property] ?? null
  }
}

function createPointOperation(runtimeProvider, path, operation) {
  if (!path) throw new Error(`points.${operation}() 必须在具体数据点路径上调用`)
  const adapterMethod =
    operation === 'sub' ? 'subscribe' : operation === 'pub' ? 'publish' : operation
  return (...args) => {
    if (adapterMethod === 'subscribe' && typeof args[0] !== 'function') {
      throw new TypeError('point.subscribe(handler) 的 handler 必须是函数')
    }
    const runtime = resolveRuntime(runtimeProvider())
    return callAdapter(runtime.adapter, adapterMethod, 'runtime adapter', [path, ...args])
  }
}

function createPoint(runtimeProvider, path = '') {
  return new Proxy(Object.create(null), {
    get(_target, property) {
      if (property === Symbol.toStringTag) return 'InduForgeDataPoint'
      if (property === 'then') return undefined
      if (typeof property !== 'string') return undefined
      if (!path && (property === 'byPath' || property === 'resolve')) {
        return (targetPath) => createPoint(runtimeProvider, String(targetPath || ''))
      }
      if (POINT_OPERATIONS.has(property))
        return createPointOperation(runtimeProvider, path, property)
      if (path && POINT_METADATA.has(property) && pointContract(runtimeProvider, path)) {
        return pointMetadata(runtimeProvider, path, property)
      }
      const nextPath = path ? `${path}.${property}` : property
      return createPoint(runtimeProvider, nextPath)
    },
  })
}

function createAlarmDomain(runtimeProvider) {
  const invoke = (methodName, ...args) => {
    const adapter = resolveRuntime(runtimeProvider()).alarmAdapter
    return callAdapter(adapter, methodName, 'alarm adapter', args)
  }
  return Object.freeze({
    items: Object.freeze({
      list: (query) => invoke('listItems', query),
      get: (id) => invoke('getItem', id),
    }),
    settings: Object.freeze({
      get: () => invoke('getSettings'),
      update: (patch) => invoke('updateSettings', patch),
    }),
    current: Object.freeze({
      list: (query) => invoke('listCurrent', query),
      get: (id) => invoke('getCurrent', id),
    }),
    changes: Object.freeze({
      subscribe(handler, options) {
        if (typeof handler !== 'function')
          throw new TypeError('alarms.changes.subscribe(handler) 的 handler 必须是函数')
        return invoke('subscribeChanges', handler, options)
      },
    }),
    actions: Object.freeze({
      acknowledge: (id, input) => invoke('acknowledge', id, input),
      unacknowledge: (id, input) => invoke('unacknowledge', id, input),
      forceClear: (id, input) => invoke('forceClear', id, input),
      shelve: (id, input) => invoke('shelve', id, input),
      unshelve: (id, input) => invoke('unshelve', id, input),
    }),
    history: Object.freeze({
      list: (query) => invoke('listHistory', query),
      get: (id) => invoke('getHistory', id),
    }),
  })
}

function createComputeOperation(runtimeProvider, ref, methodName) {
  return (input) => {
    const adapter = resolveRuntime(runtimeProvider()).computeAdapter
    return callAdapter(adapter, methodName, 'compute adapter', [ref, input])
  }
}

function createComputes(runtimeProvider, ref = '') {
  return new Proxy(Object.create(null), {
    get(_target, property) {
      if (property === Symbol.toStringTag) return 'InduForgeComputes'
      if (property === 'then') return undefined
      if (typeof property !== 'string') return undefined
      if (!ref && (property === 'byRef' || property === 'resolve')) {
        return (targetRef) => createComputes(runtimeProvider, String(targetRef || ''))
      }
      if (property === 'run' || property === 'describe') {
        if (!ref) throw new Error(`computes.${property}() 必须在具体计算单元上调用`)
        return createComputeOperation(runtimeProvider, ref, property)
      }
      const nextRef = ref ? `${ref}.${property}` : property
      return createComputes(runtimeProvider, nextRef)
    },
  })
}

function getRoles(runtimeProvider) {
  const runtime = resolveRuntime(runtimeProvider())
  const roles = runtime.roles ?? runtime.access?.roles ?? []
  return new Set(Array.isArray(roles) ? roles : [])
}

function createAccess(runtimeProvider) {
  return Object.freeze({
    hasRole(role) {
      return getRoles(runtimeProvider).has(role)
    },
    hasAnyRole(roles) {
      if (!Array.isArray(roles)) throw new TypeError('access.hasAnyRole(roles) 的 roles 必须是数组')
      const currentRoles = getRoles(runtimeProvider)
      return roles.some((role) => currentRoles.has(role))
    },
  })
}

function createScenes(runtimeProvider) {
  return Object.freeze({
    open2D(sceneId, options) {
      return requireMethod(
        resolveRuntime(runtimeProvider()).navigation,
        'open2D',
        'navigation adapter',
      )(sceneId, options)
    },
    open3D(sceneId, options) {
      return requireMethod(
        resolveRuntime(runtimeProvider()).navigation,
        'open3D',
        'navigation adapter',
      )(sceneId, options)
    },
  })
}

function buildRuntimeClient(runtimeProvider) {
  return Object.freeze({
    points: createPoint(runtimeProvider),
    alarms: createAlarmDomain(runtimeProvider),
    computes: createComputes(runtimeProvider),
    access: createAccess(runtimeProvider),
    scenes: createScenes(runtimeProvider),
  })
}

const defaultClient = buildRuntimeClient(() => null)

export const points = defaultClient.points
export const alarms = defaultClient.alarms
export const computes = defaultClient.computes
export const access = defaultClient.access
export const scenes = defaultClient.scenes

export function configureRuntime(runtime) {
  if (runtime == null || typeof runtime !== 'object' || Array.isArray(runtime)) {
    throw new TypeError('configureRuntime(runtime) 的 runtime 必须是对象')
  }
  configuredRuntime = runtime
  return defaultClient
}

export function createRuntimeClient(runtime) {
  if (runtime != null && (typeof runtime !== 'object' || Array.isArray(runtime))) {
    throw new TypeError('createRuntimeClient(runtime) 的 runtime 必须是对象')
  }
  return buildRuntimeClient(() => runtime)
}
