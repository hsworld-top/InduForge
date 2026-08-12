;(function () {
  const workspaceRoots = new Set([
    'assets',
    'components',
    'displays',
    'materials',
    'models',
    'previews',
    'scenes',
    'symbols',
  ])

  function readStoredValue(key) {
    const value = window.localStorage.getItem(key)
    if (!value) return ''
    try {
      return JSON.parse(value)
    } catch (_error) {
      return value
    }
  }

  function projectIdFromWindow(targetWindow) {
    try {
      const params = new URLSearchParams(targetWindow.location.search)
      return params.get('projectId') || params.get('pid') || ''
    } catch (_error) {
      return ''
    }
  }

  function resolveProjectId() {
    const current = projectIdFromWindow(window)
    if (current) return current
    if (window.parent && window.parent !== window) {
      const parent = projectIdFromWindow(window.parent)
      if (parent) return parent
    }
    if (window.opener) {
      const opener = projectIdFromWindow(window.opener)
      if (opener) return opener
    }
    return readStoredValue('project_id')
  }

  function resolveApiBase() {
    return new URLSearchParams(window.location.search).get('apiBase') || '/api/v1'
  }

  function readAccessToken() {
    return window.localStorage.getItem('auth_token') || ''
  }

  function authorizationHeaders(headers) {
    const result = new Headers(headers || undefined)
    const token = readAccessToken()
    if (token) result.set('Authorization', 'Bearer ' + token)
    return result
  }

  function normalizeWorkspacePath(value) {
    if (typeof value !== 'string' || value.trim() === '') return null
    if (/^(data|blob|javascript):/i.test(value)) return null

    let resolved
    try {
      resolved = new URL(value, window.location.href)
    } catch (_error) {
      return null
    }
    if (resolved.origin !== window.location.origin) return null

    const editorPrefix = window.location.pathname.replace(/[^/]*$/, '')
    let path = decodeURIComponent(resolved.pathname)
    if (path.startsWith(editorPrefix)) path = path.slice(editorPrefix.length)
    path = path.replace(/^\/+/, '')
    const root = path.split('/')[0]
    return workspaceRoots.has(root) ? path : null
  }

  function resolveDesignFileUrl(value) {
    const path = normalizeWorkspacePath(value)
    const projectId = resolveProjectId()
    if (!path || !projectId) return value

    const apiBase = resolveApiBase().replace(/\/$/, '')
    const params = new URLSearchParams({ path })
    return (
      apiBase +
      '/projects/' +
      encodeURIComponent(projectId) +
      '/design-files/content?' +
      params.toString()
    )
  }

  function installResourceInterceptors() {
    if (window.__induforgeHtResourceInterceptorsInstalled) return
    window.__induforgeHtResourceInterceptorsInstalled = true

    const originalFetch = window.fetch.bind(window)
    window.fetch = function (input, init) {
      const sourceUrl = input instanceof Request ? input.url : input
      const resolvedUrl = resolveDesignFileUrl(sourceUrl)
      if (resolvedUrl === sourceUrl) return originalFetch(input, init)

      const requestInit = { ...(init || {}), headers: authorizationHeaders(init && init.headers) }
      return originalFetch(resolvedUrl, requestInit)
    }

    const originalOpen = XMLHttpRequest.prototype.open
    const originalSend = XMLHttpRequest.prototype.send
    XMLHttpRequest.prototype.open = function (method, url) {
      const resolvedUrl = resolveDesignFileUrl(url)
      this.__induforgeDesignFileRequest = resolvedUrl !== url
      const args = Array.prototype.slice.call(arguments)
      args[1] = resolvedUrl
      return originalOpen.apply(this, args)
    }
    XMLHttpRequest.prototype.send = function () {
      if (this.__induforgeDesignFileRequest) {
        const token = readAccessToken()
        if (token) this.setRequestHeader('Authorization', 'Bearer ' + token)
      }
      return originalSend.apply(this, arguments)
    }

    const descriptor = Object.getOwnPropertyDescriptor(HTMLImageElement.prototype, 'src')
    if (descriptor && descriptor.get && descriptor.set) {
      Object.defineProperty(HTMLImageElement.prototype, 'src', {
        configurable: descriptor.configurable,
        enumerable: descriptor.enumerable,
        get: descriptor.get,
        set(value) {
          const resolvedUrl = resolveDesignFileUrl(value)
          if (resolvedUrl === value) {
            descriptor.set.call(this, value)
            return
          }

          originalFetch(resolvedUrl, { headers: authorizationHeaders() })
            .then((response) => {
              if (!response.ok) throw new Error('设计资源读取失败')
              return response.blob()
            })
            .then((blob) => {
              const objectUrl = URL.createObjectURL(blob)
              this.addEventListener('load', () => URL.revokeObjectURL(objectUrl), { once: true })
              this.addEventListener('error', () => URL.revokeObjectURL(objectUrl), { once: true })
              descriptor.set.call(this, objectUrl)
            })
            .catch(() => descriptor.set.call(this, value))
        },
      })
    }
  }

  function fallbackResult(command) {
    if (command === 'source') return ''
    if (command === 'explore') return {}
    return false
  }

  function downloadBlob(blob, filename) {
    const objectUrl = URL.createObjectURL(blob)
    const anchor = document.createElement('a')
    anchor.href = objectUrl
    anchor.download = filename || 'induforge-design.zip'
    anchor.style.display = 'none'
    document.body.appendChild(anchor)
    anchor.click()
    anchor.remove()
    window.setTimeout(() => URL.revokeObjectURL(objectUrl), 0)
  }

  function downloadFilename(response) {
    const disposition = response.headers && response.headers.get('Content-Disposition')
    if (!disposition) return 'induforge-design.zip'
    const encoded = disposition.match(/filename\*=UTF-8''([^;]+)/i)
    if (encoded) return decodeURIComponent(encoded[1])
    const plain = disposition.match(/filename="?([^";]+)"?/i)
    return plain ? plain[1] : 'induforge-design.zip'
  }

  installResourceInterceptors()

  window.InduForgeDesignFiles = { resolveUrl: resolveDesignFileUrl }
  window.InduForgeService = class InduForgeService {
    constructor(handler, editor) {
      this.handler = handler
      this.editor = editor
      this.projectId = resolveProjectId()
      this.apiBase = resolveApiBase()

      window.setTimeout(() => {
        if (!this.projectId) {
          this.handler({ type: 'disconnected', message: '缺少工程 ID' })
          return
        }
        this.handler({ type: 'connected', message: window.location.origin })
      }, 0)
    }

    request(command, data, callback) {
      let message = command
      if (typeof data === 'string') message += ': ' + data
      else if (data && data.path) message += ': ' + data.path
      this.handler({ type: 'request', message: message, cmd: command, data: data })

      if (command === 'export') {
        this.requestExport(data, callback)
        return
      }

      window
        .fetch(
          this.apiBase +
            '/projects/' +
            encodeURIComponent(this.projectId) +
            '/design-files/actions',
          {
            method: 'POST',
            credentials: 'same-origin',
            headers: authorizationHeaders({ 'Content-Type': 'application/json' }),
            body: JSON.stringify({ command: command, data: data }),
          },
        )
        .then((response) => response.json())
        .then((payload) => {
          if (!payload || payload.code !== 0) {
            throw new Error(payload && payload.msg ? payload.msg : '设计文件操作失败')
          }
          const result = payload.data ? payload.data.result : undefined
          this.notifyContextSync(payload.data && payload.data.contextSync)
          if (callback) callback(result)
          this.handler({ type: 'response', message: command, cmd: command, data: result })
          if (command === 'upload' && result && typeof result === 'object') {
            if (result.importPending) {
              this.handler({ type: 'confirm', path: result.path, datas: result.conflicts || [] })
              return
            }
            this.notifyChangedRoots(result)
            return
          }
          if (command === 'import' && result && typeof result === 'object') {
            if (data && data.move) this.notifyChangedRoots(result)
            return
          }
          if (['upload', 'remove', 'rename', 'mkdir', 'paste'].indexOf(command) >= 0) {
            this.handler({ type: 'fileChanged', path: this.resolveChangedPath(command, data) })
          }
        })
        .catch((error) => {
          const result = fallbackResult(command)
          if (callback) callback(result)
          this.handler({
            type: 'response',
            message: error instanceof Error ? error.message : String(error),
            cmd: command,
            data: result,
          })
        })
    }

    requestExport(paths, callback) {
      window
        .fetch(
          this.apiBase + '/projects/' + encodeURIComponent(this.projectId) + '/design-files/export',
          {
            method: 'POST',
            credentials: 'same-origin',
            headers: authorizationHeaders({ 'Content-Type': 'application/json' }),
            body: JSON.stringify({ paths: Array.isArray(paths) ? paths : [] }),
          },
        )
        .then((response) => {
          const contentType = response.headers && response.headers.get('Content-Type')
          if (!response.ok || !contentType || contentType.indexOf('application/zip') < 0) {
            return response
              .json()
              .catch(() => null)
              .then((payload) => {
                throw new Error(payload && payload.msg ? payload.msg : '设计资源导出失败')
              })
          }
          return response
            .blob()
            .then((blob) => ({ blob: blob, filename: downloadFilename(response) }))
        })
        .then((result) => {
          downloadBlob(result.blob, result.filename)
          if (callback) callback(true)
          this.handler({ type: 'response', message: 'export', cmd: 'export', data: true })
        })
        .catch((error) => {
          if (callback) callback(false)
          this.handler({
            type: 'response',
            message: error instanceof Error ? error.message : String(error),
            cmd: 'export',
            data: false,
          })
        })
    }

    notifyChangedRoots(result) {
      if (!result || !Array.isArray(result.roots)) return
      result.roots.forEach((root) => {
        if (workspaceRoots.has(root)) this.handler({ type: 'fileChanged', path: root + '/' })
      })
    }

    // 将后端上下文同步结果发给外层 Designer；保存成功不应被上下文镜像失败掩盖。
    notifyContextSync(contextSync) {
      if (!contextSync || typeof contextSync !== 'object' || !contextSync.status) return
      const message =
        contextSync.status === 'updated'
          ? '工程上下文已同步'
          : contextSync.message || '场景已保存，工程上下文待刷新'
      this.handler({ type: 'contextSync', message: message, data: contextSync })
      if (window.parent && window.parent !== window) {
        window.parent.postMessage(
          { source: 'induforge-ht', type: 'context-sync', status: contextSync.status, message: message },
          window.location.origin,
        )
      }
    }

    resolveChangedPath(command, data) {
      if (typeof data === 'string') return data
      if (!data) return ''
      if (command === 'rename') return data.new || data.old || ''
      return data.path || data.destDir || ''
    }
  }
})()
