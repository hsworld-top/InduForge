const CHANNEL = 'induforge-page-runtime'
const VERSION = 1
const REQUEST_TIMEOUT_MS = 10_000

let requestSequence = 0
let subscriptionSequence = 0
let listeningWindow = null
let unloadWindow = null
const pendingRequests = new Map()
const subscriptionHandlers = new Map()

function currentWindow() {
  return typeof window === 'undefined' ? null : window
}

function parentOrigin(target) {
  const referrer = target?.document?.referrer
  if (!referrer) return '*'
  try {
    return new URL(referrer).origin
  } catch {
    return '*'
  }
}

function nextId(prefix) {
  requestSequence += 1
  return `${prefix}-${Date.now()}-${requestSequence}`
}

function handleMessage(event) {
  const target = currentWindow()
  if (!target || event.source !== target.parent) return
  const message = event.data
  if (!message || message.channel !== CHANNEL || message.version !== VERSION) return

  if (message.type === 'RESULT' && typeof message.requestId === 'string') {
    const pending = pendingRequests.get(message.requestId)
    if (!pending) return
    clearTimeout(pending.timer)
    pendingRequests.delete(message.requestId)
    pending.resolve(message.result)
    return
  }

  if (message.type === 'EVENT' && typeof message.subscriptionId === 'string') {
    subscriptionHandlers.get(message.subscriptionId)?.(message.data)
  }
}

function ensureListener(target) {
  if (listeningWindow === target) return
  listeningWindow?.removeEventListener?.('message', handleMessage)
  target.addEventListener('message', handleMessage)
  listeningWindow = target
  if (unloadWindow !== target) {
    target.addEventListener('beforeunload', () => {
      target.parent?.postMessage(
        { channel: CHANNEL, version: VERSION, type: 'RELEASE_ALL' },
        parentOrigin(target),
      )
      subscriptionHandlers.clear()
    })
    unloadWindow = target
  }
}

function request(domain, operation, payload = {}) {
  const target = currentWindow()
  if (!target || !target.parent || target.parent === target) {
    return Promise.reject(new Error('当前页面未接入 InduForge 预览宿主'))
  }
  ensureListener(target)
  const requestId = nextId('runtime')
  return new Promise((resolve, reject) => {
    const timer = setTimeout(() => {
      pendingRequests.delete(requestId)
      reject(new Error(`${domain}.${operation}() 请求预览宿主超时`))
    }, REQUEST_TIMEOUT_MS)
    pendingRequests.set(requestId, { resolve, reject, timer })
    try {
      target.parent.postMessage(
        { channel: CHANNEL, version: VERSION, type: 'REQUEST', requestId, domain, operation, ...payload },
        parentOrigin(target),
      )
    } catch (error) {
      clearTimeout(timer)
      pendingRequests.delete(requestId)
      reject(error)
    }
  })
}

function subscribeDomain(domain, operation, handler, options) {
  subscriptionSequence += 1
  const subscriptionId = `${domain}-${Date.now()}-${subscriptionSequence}`
  subscriptionHandlers.set(subscriptionId, handler)
  return request(domain, operation, { args: [options], subscriptionId }).then((result) => {
    if (!result || result.code !== 0) {
      subscriptionHandlers.delete(subscriptionId)
      return result
    }
    return {
      ...result,
      data: () => {
        subscriptionHandlers.delete(subscriptionId)
        void request(domain, 'unsubscribe', { subscriptionId }).catch(() => {})
      },
    }
  })
}

function subscribePoint(path, handler, options) {
  subscriptionSequence += 1
  const subscriptionId = `point-${Date.now()}-${subscriptionSequence}`
  subscriptionHandlers.set(subscriptionId, handler)
  return pointRequest('subscribe', path, [options], subscriptionId).then((result) => {
    if (!result || result.code !== 0) {
      subscriptionHandlers.delete(subscriptionId)
      return result
    }
    return {
      ...result,
      data: () => {
        subscriptionHandlers.delete(subscriptionId)
        void request('point', 'unsubscribe', { path, subscriptionId }).catch(() => {})
      },
    }
  })
}

function pointRequest(operation, path, args, subscriptionId) {
  return request('point', operation, { path, args, subscriptionId })
}

function createPointAdapter() {
  return Object.freeze({
    get: (path, ...args) => pointRequest('get', path, args),
    read: (path, ...args) => pointRequest('read', path, args),
    peek: (path, ...args) => pointRequest('peek', path, args),
    set: (path, ...args) => pointRequest('set', path, args),
    subscribe: (path, handler, options) => subscribePoint(path, handler, options),
    history: (path, ...args) => pointRequest('history', path, args),
    refresh: (path, ...args) => pointRequest('refresh', path, args),
    run: (path, ...args) => pointRequest('run', path, args),
    execute: (path, ...args) => pointRequest('execute', path, args),
    publish: (path, ...args) => pointRequest('publish', path, args),
  })
}

function createAlarmAdapter() {
  const invoke = (operation, args) => request('alarm', operation, { args })
  return Object.freeze({
    listItems: (...args) => invoke('listItems', args),
    getItem: (...args) => invoke('getItem', args),
    getSettings: (...args) => invoke('getSettings', args),
    updateSettings: (...args) => invoke('updateSettings', args),
    listCurrent: (...args) => invoke('listCurrent', args),
    getCurrent: (...args) => invoke('getCurrent', args),
    subscribeChanges: (handler, options) => subscribeDomain('alarm', 'subscribeChanges', handler, options),
    acknowledge: (...args) => invoke('acknowledge', args),
    unacknowledge: (...args) => invoke('unacknowledge', args),
    forceClear: (...args) => invoke('forceClear', args),
    shelve: (...args) => invoke('shelve', args),
    unshelve: (...args) => invoke('unshelve', args),
    listHistory: (...args) => invoke('listHistory', args),
    getHistory: (...args) => invoke('getHistory', args),
  })
}

function createComputeAdapter() {
  return Object.freeze({
    run: (ref, input) => request('compute', 'run', { ref, args: [input] }),
    describe: (ref) => request('compute', 'describe', { ref, args: [] }),
  })
}

export function createPreviewBridgeRuntime() {
  const target = currentWindow()
  if (!target || !target.parent || target.parent === target) return null
  ensureListener(target)
  return Object.freeze({
    adapter: createPointAdapter(),
    alarmAdapter: createAlarmAdapter(),
    computeAdapter: createComputeAdapter(),
  })
}
