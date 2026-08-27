;(function () {
  var CHANNEL = 'induforge-scene-runtime'
  var VERSION = 1
  var params = {}
  var sceneId = ''
  var handlers = new Map()
  var paramListeners = new Set()
  var dataRequests = new Map()
  var dataSubscriptions = new Map()
  var graphViewCleanups = new Map()
  var parentOrigin = '*'
  try {
    if (document.referrer) parentOrigin = new URL(document.referrer).origin
  } catch (_error) {}

  function post(type, data) {
    if (!window.parent || window.parent === window) return
    window.parent.postMessage(Object.assign({ channel: CHANNEL, version: VERSION, type: type, sceneId: sceneId }, data || {}), parentOrigin)
  }

  function requestData(operation, path, value) {
    path = String(path || '').trim()
    if (!path) return Promise.reject(new Error('数据点 path 不能为空'))
    var requestId = (window.crypto && window.crypto.randomUUID) ? window.crypto.randomUUID() : Date.now() + '-' + Math.random()
    return new Promise(function (resolve, reject) {
      var timer = window.setTimeout(function () {
        dataRequests.delete(requestId)
        reject(new Error('数据点操作超时'))
      }, 10000)
      dataRequests.set(requestId, { resolve: resolve, reject: reject, timer: timer })
      post('DATA_REQUEST', { requestId: requestId, operation: operation, path: path, value: value })
    })
  }

  function pointAPI(path) {
    return Object.freeze({
      get: function () { return requestData('get', path).then(function (result) { return result && result.value }) },
      set: function (value) { return requestData('set', path, value) },
      sub: function (listener) {
        if (typeof listener !== 'function') return Promise.reject(new TypeError('订阅监听器必须是函数'))
        var listeners = dataSubscriptions.get(path)
        if (!listeners) { listeners = new Set(); dataSubscriptions.set(path, listeners) }
        listeners.add(listener)
        return requestData('sub', path).then(function () {
          return function () {
            var current = dataSubscriptions.get(path)
            if (!current) return
            current.delete(listener)
            if (!current.size) { dataSubscriptions.delete(path); requestData('unsub', path).catch(function () {}) }
          }
        })
      }
    })
  }

  window.addEventListener('message', function (event) {
    if (event.source !== window.parent || (parentOrigin !== '*' && event.origin !== parentOrigin)) return
    var message = event.data
    if (!message || message.channel !== CHANNEL || message.version !== VERSION || message.sceneId !== sceneId) return
    if (message.type === 'INITIALIZE' || message.type === 'PARAMS_CHANGED') {
      params = message.params && typeof message.params === 'object' ? message.params : {}
      paramListeners.forEach(function (listener) {
        try { listener(params) } catch (error) { post('ERROR', { code: 26003, msg: error.message || String(error) }) }
      })
      return
    }
    if (message.type === 'DATA_RESULT') {
      var pending = dataRequests.get(message.requestId)
      if (!pending) return
      window.clearTimeout(pending.timer)
      dataRequests.delete(message.requestId)
      if (message.error) {
        var dataError = new Error(message.error.msg || '数据点操作失败')
        dataError.code = Number(message.error.code) || 26008
        pending.reject(dataError)
      } else pending.resolve(message.data)
      return
    }
    if (message.type === 'DATA_PUSH') {
      var listeners = dataSubscriptions.get(message.path)
      if (listeners) listeners.forEach(function (listener) {
        try { listener(message.data) } catch (error) { post('ERROR', { code: 26008, msg: error.message || String(error) }) }
      })
      return
    }
    if (message.type !== 'COMMAND_REQUEST') return
    var handler = handlers.get(message.name)
    if (!handler) {
      post('COMMAND_RESULT', { requestId: message.requestId, error: { code: 26005, msg: '场景未注册命令 ' + message.name } })
      return
    }
    Promise.resolve().then(function () { return handler(message.payload) }).then(function (result) {
      post('COMMAND_RESULT', { requestId: message.requestId, result: result })
    }).catch(function (error) {
      post('COMMAND_RESULT', { requestId: message.requestId, error: { code: 26003, msg: error.message || String(error) } })
    })
  })

  window.InduForgeScene = Object.freeze({
    configure: function (input) { sceneId = input && input.sceneId ? String(input.sceneId) : '' },
    getParams: function () { return params },
    onParamsChange: function (listener) {
      if (typeof listener !== 'function') throw new TypeError('参数监听器必须是函数')
      paramListeners.add(listener)
      return function () { paramListeners.delete(listener) }
    },
    emit: function (name, payload) { post('EVENT', { name: String(name || ''), payload: payload }) },
    registerCommand: function (name, handler) {
      name = String(name || '')
      if (!name || typeof handler !== 'function') throw new TypeError('命令名称和处理函数不能为空')
      if (handlers.has(name)) throw new Error('命令 ' + name + ' 已注册')
      handlers.set(name, handler)
      return function () { if (handlers.get(name) === handler) handlers.delete(name) }
    },
    points: new Proxy({}, { get: function (_target, path) { return pointAPI(String(path)) } }),
    point: pointAPI,
    markReady: function () { post('READY') },
    reportError: function (error) {
      post('ERROR', {
        code: error && Number(error.code) ? Number(error.code) : 26003,
        msg: error && error.message ? error.message : String(error),
        requestId: error && error.requestId ? String(error.requestId) : '',
      })
    },
  })

  function formatValue(value, format) {
    if (value == null) return format.nullText || ''
    if (typeof value === 'number' && Number.isInteger(format.precision) && format.precision >= 0) value = value.toFixed(format.precision)
    return String(format.prefix || '') + String(value) + String(format.suffix || '')
  }

  function applyBindingValue(data, target, payload, binding) {
    var value = payload && Object.prototype.hasOwnProperty.call(payload, 'value') ? payload.value : payload
    if (payload && payload.quality && payload.quality !== 'good' && binding.format && binding.format.qualityFallback) value = binding.format.qualityFallback
    if (!data.__induforgeConfirmedValues) data.__induforgeConfirmedValues = {}
    data.__induforgeConfirmedValues[target] = value
    data.__induforgeDatapointUpdate = true
    try {
      if (target === 'text') data.s('label', formatValue(value, binding.format || {}))
      else if (target === 'visible') data.s('2d.visible', Boolean(value))
      else if (target === 'color') data.s(data.a('induforge.path.type') === 'pipe' ? 'shape.dash.color' : 'body.color', String(value))
      else if (target === 'selected' && typeof data.setSelected === 'function') data.setSelected(Boolean(value))
      else if (target === 'value' && typeof data.setValue === 'function') data.setValue(value)
      else if (target === 'pipe.running') { data.a('induforge.pipe.running', Boolean(value)); if (window.InduForgePipe) window.InduForgePipe.apply(data) }
      else data.a('induforge.runtime.' + target, value)
    } finally {
      window.setTimeout(function () { data.__induforgeDatapointUpdate = false }, 0)
    }
  }

  function objectValue(data, target) {
    if (target === 'selected' && typeof data.isSelected === 'function') return data.isSelected()
    if (target === 'value' && typeof data.getValue === 'function') return data.getValue()
    if (target === 'pipe.running') return data.a('induforge.pipe.running') !== false
    return data.a('induforge.runtime.' + target)
  }

  function restoreConfirmedValue(data, target, binding) {
    var confirmed = data.__induforgeConfirmedValues
    if (!confirmed || !Object.prototype.hasOwnProperty.call(confirmed, target)) return
    applyBindingValue(data, target, { value: confirmed[target], quality: 'good' }, binding)
  }

  function writeBindingValue(data, target, binding) {
    var submitted = objectValue(data, target)
    return window.InduForgeScene.point(binding.pointPath).set(submitted).then(function (result) {
      var confirmed = result && Object.prototype.hasOwnProperty.call(result, 'value') ? result.value : submitted
      applyBindingValue(data, target, { value: confirmed, quality: 'good' }, binding)
      return result
    }).catch(function (error) {
      restoreConfirmedValue(data, target, binding)
      window.InduForgeScene.reportError(error)
    })
  }

  window.InduForgeSceneRuntime = Object.freeze({
    bindGraphView: function (graphView) {
      if (!graphView || graphView.__induforgeRuntimeBound) return
      graphView.__induforgeRuntimeBound = true
      var dm = graphView.dm
      var cleanups = []
      dm.each(function (data) {
        var bindings = data.a && data.a('induforge.bindings')
        Object.keys(bindings || {}).forEach(function (target) {
          var binding = bindings[target] || {}
          if (!binding.pointPath || binding.direction === 'write') return
          var update = function (payload) { applyBindingValue(data, target, payload, binding) }
          if (binding.readMode === 'subscribe') window.InduForgeScene.point(binding.pointPath).sub(update).then(function (unsubscribe) {
            if (typeof unsubscribe === 'function') cleanups.push(unsubscribe)
          }).catch(window.InduForgeScene.reportError)
          else window.InduForgeScene.point(binding.pointPath).get().then(function (value) { update({ value: value, quality: 'good' }) }).catch(window.InduForgeScene.reportError)
        })
        var interactions = data.a && data.a('induforge.interactions')
        ;(interactions && interactions.commands || []).forEach(function (mapping) {
          try {
            var unregister = window.InduForgeScene.registerCommand(mapping.name, function (input) {
              if (mapping.action === 'setValue' && typeof data.setValue === 'function') data.setValue(input.value)
              else if (mapping.action === 'show') data.s('2d.visible', true)
              else if (mapping.action === 'hide') data.s('2d.visible', false)
              else if (mapping.action === 'enable') {
                if (typeof data.setEnabled === 'function') data.setEnabled(true)
                else data.a('induforge.runtime.enabled', true)
              } else if (mapping.action === 'disable') {
                if (typeof data.setEnabled === 'function') data.setEnabled(false)
                else data.a('induforge.runtime.enabled', false)
              }
              else if (mapping.action === 'setPipeState') { Object.keys(input || {}).forEach(function (key) { data.a('induforge.pipe.' + key, input[key]) }); if (window.InduForgePipe) window.InduForgePipe.apply(data) }
              else if (mapping.action === 'focus') { dm.sm().ss(data); graphView.fitData(data, true, 40) }
              else if (mapping.action === 'highlight') data.s('shadow', true)
              return null
            })
            cleanups.push(unregister)
          } catch (error) { window.InduForgeScene.reportError(error) }
        })
      })
      if (dm.addDataPropertyChangeListener) dm.addDataPropertyChangeListener(function (event) {
        var data = event.data
        if (!data || data.__induforgeDatapointUpdate) return
        var interactions = data.a && data.a('induforge.interactions')
        ;(interactions && interactions.events || []).forEach(function (mapping) {
          var changed = mapping.trigger === 'change' && (event.property === 'value' || event.property === 'selected' || event.property === 'selectedItem' || event.property === 'selectedIndex')
          var input = mapping.trigger === 'input' && event.property === 'value'
          var flow = mapping.trigger === 'flowStateChange' && String(event.property || '').indexOf('induforge.pipe.') === 0
          if (changed || input) window.InduForgeScene.emit(mapping.name, { value: objectValue(data, data.a('induforge.control.type') === 'switch' ? 'selected' : 'value') })
          if (flow) window.InduForgeScene.emit(mapping.name, {
            mode: data.a('induforge.pipe.flowMode') || 'continuous', running: data.a('induforge.pipe.running') !== false,
            reverse: data.a('induforge.pipe.reverse') === true, speed: Number(data.a('induforge.pipe.speed') || 2)
          })
        })
        var bindings = data.a && data.a('induforge.bindings')
        Object.keys(bindings || {}).forEach(function (target) {
          var binding = bindings[target]
          if (binding.direction === 'read' || !binding.pointPath) return
          var properties = { value: ['value'], selected: ['selected'], 'pipe.running': ['induforge.pipe.running'] }[target] || []
          if (properties.indexOf(event.property) < 0) return
          writeBindingValue(data, target, binding)
        })
      })
      if (graphView.addInteractorListener) graphView.addInteractorListener(function (event) {
        if (event.kind !== 'clickData' && event.kind !== 'doubleClickData' && event.kind !== 'beginData' && event.kind !== 'endData') return
        var data = event.data
        var interactions = data && data.a && data.a('induforge.interactions')
        ;(interactions && interactions.events || []).forEach(function (mapping) {
          if ((mapping.trigger === 'click' && event.kind === 'clickData') ||
              (mapping.trigger === 'doubleClick' && event.kind === 'doubleClickData') ||
              (mapping.trigger === 'press' && event.kind === 'beginData') ||
              (mapping.trigger === 'release' && event.kind === 'endData')) window.InduForgeScene.emit(mapping.name, {})
        })
      })
      graphViewCleanups.set(graphView, cleanups)
    },
    unbindGraphView: function (graphView) {
      var cleanups = graphViewCleanups.get(graphView) || []
      graphViewCleanups.delete(graphView)
      cleanups.forEach(function (cleanup) {
        try { cleanup() } catch (_error) {}
      })
      if (graphView) graphView.__induforgeRuntimeBound = false
    }
  })

  window.addEventListener('beforeunload', function () {
    graphViewCleanups.forEach(function (cleanups) {
      cleanups.forEach(function (cleanup) { try { cleanup() } catch (_error) {} })
    })
    graphViewCleanups.clear()
    dataRequests.forEach(function (pending) {
      window.clearTimeout(pending.timer)
      pending.reject(new Error('Viewer 已关闭'))
    })
    dataRequests.clear()
  })
})()
