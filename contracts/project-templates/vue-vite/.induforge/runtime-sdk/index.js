const POINT_OPERATIONS = new Set(['get', 'set', 'sub', 'pub'])

let configuredRuntime = null

function readBrowserRuntime() {
  if (typeof window === 'undefined') {
    return {}
  }

  return window.__INDUFORGE_RUNTIME__ ?? {}
}

function resolveRuntime(runtime) {
  return runtime ?? configuredRuntime ?? readBrowserRuntime()
}

function requireMethod(target, methodName, adapterName) {
  const method = target?.[methodName]
  if (typeof method !== 'function') {
    throw new Error(`${adapterName} 缺少 ${methodName}() 方法`)
  }

  return method.bind(target)
}

function createPointOperation(runtimeProvider, path, operation) {
  if (!path) {
    throw new Error(`points.${operation}() 必须在具体数据点路径上调用`)
  }

  return (...args) => {
    const runtime = resolveRuntime(runtimeProvider())
    const adapter = runtime.adapter

    switch (operation) {
      case 'get':
        return requireMethod(adapter, 'get', 'runtime adapter')(path, args[0])
      case 'set':
        return requireMethod(adapter, 'set', 'runtime adapter')(path, args[0])
      case 'sub': {
        const handler = args[0]
        if (typeof handler !== 'function') {
          throw new TypeError('points.<path>.sub(handler) 的 handler 必须是函数')
        }
        return requireMethod(adapter, 'subscribe', 'runtime adapter')(path, handler)
      }
      case 'pub':
        return requireMethod(adapter, 'publish', 'runtime adapter')(path, args[0])
      default:
        throw new Error(`不支持的数据点操作：${operation}`)
    }
  }
}

function createPoints(runtimeProvider, path = '') {
  return new Proxy(Object.create(null), {
    get(_target, property) {
      if (property === Symbol.toStringTag) {
        return 'InduForgePoints'
      }
      if (property === 'then') {
        return undefined
      }
      if (typeof property !== 'string') {
        return undefined
      }
      if (POINT_OPERATIONS.has(property)) {
        return createPointOperation(runtimeProvider, path, property)
      }

      const nextPath = path ? `${path}.${property}` : property
      return createPoints(runtimeProvider, nextPath)
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
      if (!Array.isArray(roles)) {
        throw new TypeError('access.hasAnyRole(roles) 的 roles 必须是数组')
      }
      const currentRoles = getRoles(runtimeProvider)
      return roles.some((role) => currentRoles.has(role))
    },
  })
}

function createScenes(runtimeProvider) {
  return Object.freeze({
    open2D(sceneId, options) {
      const navigation = resolveRuntime(runtimeProvider()).navigation
      return requireMethod(navigation, 'open2D', 'navigation adapter')(sceneId, options)
    },
    open3D(sceneId, options) {
      const navigation = resolveRuntime(runtimeProvider()).navigation
      return requireMethod(navigation, 'open3D', 'navigation adapter')(sceneId, options)
    },
  })
}

function buildRuntimeClient(runtimeProvider) {
  return Object.freeze({
    points: createPoints(runtimeProvider),
    access: createAccess(runtimeProvider),
    scenes: createScenes(runtimeProvider),
  })
}

const defaultRuntimeProvider = () => null
const defaultClient = buildRuntimeClient(defaultRuntimeProvider)

export const points = defaultClient.points
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
