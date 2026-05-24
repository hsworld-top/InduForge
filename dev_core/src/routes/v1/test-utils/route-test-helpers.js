/**
 * 直接执行 Express Router 的路由栈，避免测试环境因本地监听能力受限而触发 supertest 绑定端口失败。
 * 这里只覆盖当前路由契约测试需要的最小响应面：status / json / setHeader / send / end。
 */
function createMockResponse(req) {
  const state = {
    statusCode: 200,
    payload: null,
    headers: {},
    sent: false,
  }

  const res = {
    req,
    locals: {},
    setHeader(key, value) {
      state.headers[key] = value
    },
    status(code) {
      state.statusCode = code
      return res
    },
    json(payload) {
      state.payload = payload
      state.sent = true
      return res
    },
    send(payload) {
      state.payload = payload
      state.sent = true
      return res
    },
    end() {
      state.sent = true
      return res
    },
  }

  return { res, state }
}

function getRouteHandlers(router, path, method) {
  const routeLayer = router.stack.find(
    (layer) => layer.route && layer.route.path === path && layer.route.methods[method],
  )

  if (!routeLayer) {
    throw new Error(`未找到路由: ${method.toUpperCase()} ${path}`)
  }

  return routeLayer.route.stack
}

async function runHandlersSequentially(handlers, req, res, state, index = 0) {
  if (index >= handlers.length || state.sent) return

  const layer = handlers[index]

  await new Promise((resolve, reject) => {
    let nextCalled = false
    let settled = false

    const finish = () => {
      if (settled) return
      settled = true
      resolve()
    }

    const next = (error) => {
      nextCalled = true
      if (error) {
        reject(error)
        return
      }

      Promise.resolve(runHandlersSequentially(handlers, req, res, state, index + 1))
        .then(finish)
        .catch(reject)
    }

    try {
      const returned = layer.handle(req, res, next)

      if (returned && typeof returned.then === 'function') {
        returned
          .then(() => {
            if (!nextCalled) finish()
          })
          .catch(reject)
        return
      }

      queueMicrotask(() => {
        if (!nextCalled) finish()
      })
    } catch (error) {
      reject(error)
    }
  })
}

async function invokeRoute(router, path, method, overrides = {}) {
  const handlers = getRouteHandlers(router, path, method)
  const headers = overrides.headers || {}
  const req = {
    method: method.toUpperCase(),
    url: overrides.url || path,
    originalUrl: overrides.originalUrl || overrides.url || path,
    path: overrides.path || path,
    params: overrides.params || {},
    query: overrides.query || {},
    body: overrides.body || {},
    headers,
    language: overrides.language || 'zh-CN',
    requestId: overrides.requestId || 'req-route-test',
    ip: overrides.ip || '127.0.0.1',
    get: overrides.get || ((header) => headers[String(header).toLowerCase()] || ''),
  }

  const { res, state } = createMockResponse(req)
  await runHandlersSequentially(handlers, req, res, state)

  return {
    status: state.statusCode,
    body: state.payload,
    headers: state.headers,
    req,
    res,
  }
}

module.exports = {
  createMockResponse,
  getRouteHandlers,
  invokeRoute,
}
