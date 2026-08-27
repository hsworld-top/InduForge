;(function () {
  const roots = new Set(['assets', 'components', 'displays', 'materials', 'models', 'previews', 'scenes', 'symbols'])
  const query = (name) => new URLSearchParams(window.location.search).get(name) || ''
  const sceneSessionId = () => query('sessionId')
  const assetSessionId = () => query('assetSessionId')
  const viewerSessionId = () => query('viewerSessionId')
  const sceneEntryPath = () => query('open')
  const fileEndpoint = (suffix) => {
    if (assetSessionId()) return '/api/v1/scene-asset-editor-sessions/' + encodeURIComponent(assetSessionId()) + suffix
    if (sceneSessionId()) return '/api/v1/scene-editor-sessions/' + encodeURIComponent(sceneSessionId()) + suffix
    if (viewerSessionId()) return '/api/v1/scene-viewer-sessions/' + encodeURIComponent(viewerSessionId()) + suffix
    return ''
  }
  const sceneEndpoint = (suffix) => sceneSessionId()
    ? '/api/v1/scene-editor-sessions/' + encodeURIComponent(sceneSessionId()) + suffix
    : ''
  const explorerFileTypes = {
    displays: { fileType: 'display', fileImage: 'editor.display' },
    scenes: { fileType: 'scene', fileImage: 'editor.scene' },
  }

  function applySceneDisplayName(editor, kind) {
    const sceneName = query('sceneName').trim()
    const entryPath = query('open').trim()
    if (!sceneSessionId() || !sceneName || !entryPath || !editor) return

    document.title = sceneName + ' · ' + String(kind).toUpperCase() + ' 编辑器'
    const renameEntryTab = (tab) => {
      if (!tab || typeof tab.setName !== 'function' || typeof tab.getTag !== 'function') return
      if (tab.getTag() === entryPath) tab.setName(sceneName)
    }

    renameEntryTab(editor.mainTabView && editor.mainTabView.currentTab)
    if (editor.__induforgeSceneDisplayNameInstalled || typeof editor.addEventListener !== 'function') return
    editor.__induforgeSceneDisplayNameInstalled = true
    editor.addEventListener((event) => {
      if (!event) return
      if (event.type === 'tabCreated') renameEntryTab(event.tab)
      if (event.type === 'mainTabSelectionChanged') renameEntryTab(event.newTab)
    })
  }

  function normalizePath(value) {
    if (typeof value !== 'string' || !value.trim() || /^(data|blob|javascript):/i.test(value)) return null
    let resolved
    try { resolved = new URL(value, window.location.href) } catch (_error) { return null }
    if (resolved.origin !== window.location.origin) return null
    if (resolved.pathname.startsWith('/api/')) return null
    const prefix = window.location.pathname.replace(/[^/]*$/, '')
    let filename = decodeURIComponent(resolved.pathname)
    if (filename.startsWith(prefix)) filename = filename.slice(prefix.length)
    filename = filename.replace(/^\/+/, '')
    const segments = filename.split('/')
    const currentSessionId = assetSessionId() || sceneSessionId() || viewerSessionId()
    if (segments[0] === currentSessionId && roots.has(segments[1])) {
      filename = segments.slice(1).join('/')
    }
    if (assetSessionId()) return filename
    return roots.has(filename.split('/')[0]) ? filename : null
  }

  function resourceURL(value) {
    const filename = normalizePath(value)
    return filename && fileEndpoint('/files/content')
      ? fileEndpoint('/files/content') + '?' + new URLSearchParams({ path: filename })
      : value
  }

  if (!window.__induforgeHtResourceInterceptorsInstalled) {
    window.__induforgeHtResourceInterceptorsInstalled = true
    const originalFetch = window.fetch.bind(window)
    window.fetch = function (input, init) {
      const source = input instanceof Request ? input.url : input
      const resolved = resourceURL(source)
      return resolved === source ? originalFetch(input, init) : originalFetch(resolved, { ...(init || {}), credentials: 'same-origin' })
    }
    const originalOpen = XMLHttpRequest.prototype.open
    const originalSend = XMLHttpRequest.prototype.send
    XMLHttpRequest.prototype.open = function (method, url) {
      const args = Array.prototype.slice.call(arguments)
      args[1] = resourceURL(url)
      this.__induforgeSceneRequest = args[1] !== url
      return originalOpen.apply(this, args)
    }
    XMLHttpRequest.prototype.send = function () {
      if (this.__induforgeSceneRequest) this.withCredentials = true
      return originalSend.apply(this, arguments)
    }
  }

  async function envelope(response) {
    const payload = await response.json().catch(() => null)
    if (!response.ok || !payload || payload.code !== 0) {
      const error = new Error(payload && payload.msg ? payload.msg : '场景资源操作失败')
      error.code = payload && Number(payload.code) ? Number(payload.code) : 26003
      error.requestId = payload && payload.reqId ? String(payload.reqId) : ''
      throw error
    }
    return payload.data
  }

  function uploadBlob(content) {
    if (content instanceof Blob) return content
    if (typeof content !== 'string' || !/^data:/i.test(content)) return new Blob([content || ''])
    const separator = content.indexOf(',')
    if (separator < 0) return new Blob([])
    const metadata = content.slice(0, separator)
    const payload = content.slice(separator + 1)
    if (!/;base64/i.test(metadata)) return new Blob([decodeURIComponent(payload)])
    const binary = window.atob(payload)
    const bytes = new Uint8Array(binary.length)
    for (let index = 0; index < binary.length; index += 1) bytes[index] = binary.charCodeAt(index)
    return new Blob([bytes])
  }

  function download(blob) {
    const objectURL = URL.createObjectURL(blob)
    const anchor = document.createElement('a')
    anchor.href = objectURL
    anchor.download = 'induforge-scene.zip'
    anchor.style.display = 'none'
    document.body.appendChild(anchor)
    anchor.click()
    anchor.remove()
    window.setTimeout(() => URL.revokeObjectURL(objectURL), 0)
  }

  window.InduForgeDesignFiles = { resolveUrl: resourceURL }
  window.InduForgeService = class InduForgeService {
    constructor(handler, editor) {
      this.handler = handler
      this.editor = editor
      this.draftVersion = 0
      this.pendingSceneSnapshotPath = ''
      this.pendingSceneCommitTimer = null
      this.ready = this.initialize()
    }
    async initialize() {
      if (!sceneSessionId() && !assetSessionId() && !viewerSessionId()) {
        this.handler({ type: 'disconnected', message: '缺少编辑会话' })
        throw new Error('缺少编辑会话')
      }
      if (!viewerSessionId()) {
        const result = await this.list('', 1)
        this.draftVersion = Number(result.draftVersion || 0)
      }
      this.handler({ type: 'connected', message: window.location.origin })
    }
    request(command, data, callback) {
      this.handler({ type: 'request', message: command, cmd: command, data })
      this.ready.then(() => this.execute(command, data)).then((result) => {
        if (callback) callback(result)
        this.handler({ type: 'response', message: command, cmd: command, data: result })
        if (command === 'upload') this.handler({ type: 'fileChanged', path: data && data.path ? data.path : '' })
      }).catch((error) => {
        const result = command === 'source' ? '' : command === 'explore' ? {} : false
        if (callback) callback(result)
        this.handler({ type: 'response', message: error.message || String(error), cmd: command, data: result })
      })
    }
    async execute(command, data) {
      if (viewerSessionId() && command !== 'source') throw new Error('Viewer 会话为只读模式')
      if (command === 'explore') return this.explore(typeof data === 'string' ? data : '')
      if (command === 'source') return this.source(data)
      if (command === 'upload') return this.upload(data)
      if (command === 'export' && sceneSessionId()) return this.exportArchive()
      throw new Error('当前架构不支持此文件命令: ' + command)
    }
    async list(directory, page) {
      const logicalDirectory = directory ? normalizePath(directory) : ''
      if (directory && logicalDirectory === null) throw new Error('场景目录无效')
      const params = new URLSearchParams({ directory: logicalDirectory, page: String(page), limit: '200', sort: 'name', order: 'asc' })
      return envelope(await fetch(fileEndpoint('/files') + '?' + params, { credentials: 'same-origin' }))
    }
    async explore(directory) {
      const payload = await this.list(directory || '', 1)
      this.draftVersion = Number(payload.draftVersion || this.draftVersion)
      const result = {}
      const logicalDirectory = directory ? normalizePath(directory) : ''
      const fileMetadata = explorerFileTypes[(logicalDirectory || '').split('/')[0]]
      for (const item of payload.items || []) {
        result[item.name] = item.type === 'directory' ? {} : fileMetadata ? { ...fileMetadata } : true
      }
      return result
    }
    async source(data) {
      const input = typeof data === 'string' ? { url: data } : data || {}
      const filename = normalizePath(input.url) || input.url
      const response = await fetch(fileEndpoint('/files/content') + '?' + new URLSearchParams({ path: filename }), { credentials: 'same-origin' })
      if (!response.ok) throw new Error('读取场景资源失败')
      if (String(input.encoding).toLowerCase() === 'base64') {
        const buffer = new Uint8Array(await response.arrayBuffer())
        let binary = ''
        buffer.forEach((value) => { binary += String.fromCharCode(value) })
        return (input.prefix || '') + window.btoa(binary)
      }
      return (input.prefix || '') + await response.text()
    }
    async upload(data) {
      if (!data || !data.path) throw new Error('缺少资源路径')
      const logicalPath = normalizePath(data.path) || data.path
      const isPendingSnapshot = Boolean(
        sceneSessionId() &&
        this.pendingSceneSnapshotPath &&
        logicalPath === this.pendingSceneSnapshotPath
      )
      if (isPendingSnapshot && this.pendingSceneCommitTimer) {
        window.clearTimeout(this.pendingSceneCommitTimer)
        this.pendingSceneCommitTimer = null
      }
      const body = uploadBlob(data.content)
      if (/\.zip$/i.test(data.path) && sceneSessionId()) return this.importArchive(body)
      const params = new URLSearchParams({ path: data.path, baseDraftVersion: String(this.draftVersion) })
      const payload = await envelope(await fetch(fileEndpoint('/files/content') + '?' + params, {
        method: 'PUT', credentials: 'same-origin', headers: { 'Content-Type': body.type || 'application/octet-stream' }, body,
      }))
      this.draftVersion = Number(payload.draftVersion)
      if (assetSessionId() && /\.json$/i.test(logicalPath)) {
        await this.commit()
      } else if (sceneSessionId() && logicalPath === sceneEntryPath()) {
        this.pendingSceneSnapshotPath = logicalPath.replace(/\.json$/i, '.png')
        this.scheduleSceneCommitFallback()
      } else if (isPendingSnapshot) {
        this.pendingSceneSnapshotPath = ''
        await this.commit()
      }
      return true
    }
    scheduleSceneCommitFallback() {
      if (this.pendingSceneCommitTimer) window.clearTimeout(this.pendingSceneCommitTimer)
      this.pendingSceneCommitTimer = window.setTimeout(() => {
        this.pendingSceneCommitTimer = null
        if (!this.pendingSceneSnapshotPath) return
        this.pendingSceneSnapshotPath = ''
        this.commit().catch((error) => {
          this.handler({ type: 'response', message: error.message || String(error), cmd: 'commit', data: false })
        })
      }, 1000)
    }
    async importArchive(body) {
      const params = new URLSearchParams({ baseDraftVersion: String(this.draftVersion) })
      const payload = await envelope(await fetch(sceneEndpoint('/import') + '?' + params, {
        method: 'POST', credentials: 'same-origin', headers: { 'Content-Type': 'application/zip' }, body,
      }))
      this.draftVersion = Number(payload.draftVersion)
      return true
    }
    async exportArchive() {
      const response = await fetch(sceneEndpoint('/export'), { method: 'POST', credentials: 'same-origin' })
      if (!response.ok || !(response.headers.get('Content-Type') || '').includes('application/zip')) {
        const payload = await response.json().catch(() => null)
        throw new Error(payload && payload.msg ? payload.msg : '导出场景资源失败')
      }
      download(await response.blob())
      return true
    }
    async commit() {
      const payload = await envelope(await fetch(fileEndpoint('/commit'), {
        method: 'POST', credentials: 'same-origin', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ baseDraftVersion: this.draftVersion }),
      }))
      this.handler({ type: assetSessionId() ? 'assetCommitted' : 'sceneCommitted', message: '保存成功', data: payload })
      if (window.parent && window.parent !== window) {
        window.parent.postMessage({ source: 'induforge-scene', type: assetSessionId() ? 'asset-committed' : 'committed', revision: payload.revision }, window.location.origin)
      }
    }
  }

  window.InduForgeViewerSession = {
    async bootstrap() {
      if (!viewerSessionId()) return null
      return envelope(await fetch('/api/v1/scene-viewer-sessions/' + encodeURIComponent(viewerSessionId()), { credentials: 'same-origin' }))
    },
    async heartbeat() {
      if (!viewerSessionId()) return null
      return envelope(await fetch('/api/v1/scene-viewer-sessions/' + encodeURIComponent(viewerSessionId()) + '/heartbeat', { method: 'POST', credentials: 'same-origin' }))
    },
  }

  const categories = {
    '2d': [['control', '控件'], ['symbol', '图形模板'], ['component', '业务组件'], ['image', '图片'], ['font', '字体']],
    '3d': [['image', '贴图'], ['font', '字体'], ['model', '模型'], ['material', '材质']],
  }
  const assetTypeLabels = {
    image: '图片',
    font: '字体',
    model: '模型',
    material: '材质',
    symbol: '图形模板',
    component: '业务组件',
  }
  const controls = [
    {
      type: 'button', name: '按钮', constructorName: 'ButtonNode', width: 120, height: 36,
      init(data) { data.s('live.label', '按钮') },
    },
    {
      type: 'toggle-button', name: '切换按钮', constructorName: 'ToggleButtonNode', width: 120, height: 36,
      init(data) { data.s('live.label', '切换按钮') },
    },
    {
      type: 'checkbox', name: '复选框', constructorName: 'CheckboxNode', width: 120, height: 32,
      init(data) { data.s('live.label', '复选框') },
    },
    {
      type: 'radio', name: '单选按钮', constructorName: 'RadioButtonNode', width: 120, height: 32,
      init(data) { data.s('live.label', '单选按钮') },
    },
    {
      type: 'switch', name: '开关', constructorName: 'SwitchNode', width: 92, height: 36,
      init(data) {
        data.s('switch.text.on', '开')
        data.s('switch.text.off', '关')
      },
    },
    {
      type: 'select', name: '下拉选择', constructorName: 'ComboboxNode', width: 140, height: 36,
      init(data) {
        data.setItems([{ label: '选项一', value: 'option-1' }, { label: '选项二', value: 'option-2' }])
        data.setSelectedIndex(0)
      },
    },
    {
      type: 'progress', name: '进度条', constructorName: 'ProgressBarNode', width: 160, height: 28,
      init(data) { data.setValue(50) },
    },
    {
      type: 'slider', name: '滑块', constructorName: 'SliderNode', width: 160, height: 32,
      init(data) { data.setValue(50) },
    },
    { type: 'number-input', name: '数字输入', constructorName: 'SpinnerNode', width: 120, height: 36 },
  ]
  const builtInTemplates = [
    {
      type: 'progress-horizontal', name: '横向进度图', image: 'induforge.template.progress.horizontal', width: 180, height: 32,
      init(data) { data.a('induforge.template.value', 50) },
    },
    {
      type: 'progress-vertical', name: '纵向进度图', image: 'induforge.template.progress.vertical', width: 32, height: 180,
      init(data) { data.a('induforge.template.value', 50) },
    },
    { type: 'table', name: '表格', image: 'induforge.template.table', width: 180, height: 108 },
    {
      type: 'clock', name: '时钟', image: 'induforge.template.clock', width: 120, height: 120,
      init(data) {
        const now = new Date()
        data.a('induforge.clock.hour', now.getHours())
        data.a('induforge.clock.minute', now.getMinutes())
        data.a('induforge.clock.second', now.getSeconds())
      },
    },
  ]

  function ensureBuiltInTemplateImages() {
    const engine = window.ht
    if (window.__induforgeBuiltInTemplateImagesInstalled || !engine || !engine.Default || typeof engine.Default.setImage !== 'function') return
    window.__induforgeBuiltInTemplateImagesInstalled = true
    const valueRatio = (data) => Math.max(0, Math.min(100, Number(data.a('induforge.template.value')) || 0)) / 100
    const progressComps = (vertical) => [
      { type: 'rect', rect: vertical ? [0, 0, 32, 180] : [0, 0, 180, 32], background: '#d8ddda' },
      {
        type: 'rect',
        rect: {
          func(data) {
            const ratio = valueRatio(data)
            return vertical ? [0, 180 * (1 - ratio), 32, 180 * ratio] : [0, 0, 180 * ratio, 32]
          },
        },
        background: '#17845b',
      },
      {
        type: 'text',
        text: { func(data) { return Math.round(valueRatio(data) * 100) + '%' } },
        rect: vertical ? [0, 0, 32, 180] : [0, 0, 180, 32],
        align: 'center',
        color: '#18201d',
        font: '12px Arial',
      },
    ]
    engine.Default.setImage('induforge.template.progress.horizontal', { width: 180, height: 32, comps: progressComps(false) })
    engine.Default.setImage('induforge.template.progress.vertical', { width: 32, height: 180, comps: progressComps(true) })
    engine.Default.setImage('induforge.template.table', {
      width: 180,
      height: 108,
      comps: [
        { type: 'rect', rect: [0, 0, 180, 108], background: '#ffffff', borderWidth: 1, borderColor: '#52615a' },
        { type: 'rect', rect: [0, 0, 180, 27], background: '#dfe8e3' },
        { type: 'shape', points: [0, 27, 180, 27], borderWidth: 1, borderColor: '#7b8781' },
        { type: 'shape', points: [0, 54, 180, 54], borderWidth: 1, borderColor: '#7b8781' },
        { type: 'shape', points: [0, 81, 180, 81], borderWidth: 1, borderColor: '#7b8781' },
        { type: 'shape', points: [60, 0, 60, 108], borderWidth: 1, borderColor: '#7b8781' },
        { type: 'shape', points: [120, 0, 120, 108], borderWidth: 1, borderColor: '#7b8781' },
      ],
    })
    const handPoints = (data, unit, maximum, length) => {
      const centerX = 60
      const centerY = 60
      const angle = (Number(data.a(unit)) || 0) / maximum * Math.PI * 2 - Math.PI / 2
      return [centerX, centerY, centerX + Math.cos(angle) * 120 * length, centerY + Math.sin(angle) * 120 * length]
    }
    engine.Default.setImage('induforge.template.clock', {
      width: 120,
      height: 120,
      comps: [
        { type: 'circle', rect: [0.04, 0.04, 0.92, 0.92], relative: true, background: '#ffffff', borderWidth: 2, borderColor: '#52615a' },
        { type: 'shape', points: { func(data) { return handPoints(data, 'induforge.clock.hour', 12, 0.24) } }, borderWidth: 4, borderColor: '#18201d' },
        { type: 'shape', points: { func(data) { return handPoints(data, 'induforge.clock.minute', 60, 0.34) } }, borderWidth: 3, borderColor: '#17845b' },
        { type: 'shape', points: { func(data) { return handPoints(data, 'induforge.clock.second', 60, 0.39) } }, borderWidth: 1, borderColor: '#c64336' },
        { type: 'circle', rect: [0.46, 0.46, 0.08, 0.08], relative: true, background: '#18201d' },
      ],
    })
    // 旧 revision 可能已固定静态管道模板；继续注册其图形，但不再允许从资源库新建。
    engine.Default.setImage('induforge.template.pipe', {
      width: 180,
      height: 72,
      comps: [
        { type: 'shape', points: [14, 20, 122, 20, 122, 52, 166, 52], borderWidth: 12, borderColor: '#586761', borderCap: 'round', borderJoin: 'round' },
        { type: 'shape', points: [14, 20, 122, 20, 122, 52, 166, 52], borderWidth: 5, borderColor: '#80c9a8', borderCap: 'round', borderJoin: 'round' },
      ],
    })
  }

  // 编辑器和 Viewer 反序列化场景前必须完成注册，否则已保存模板会显示为缺失图片。
  ensureBuiltInTemplateImages()

  class AssetLibrary {
    constructor(editor, kind) {
      this.editor = editor
      this.kind = kind
      this.type = categories[kind][0][0]
      this.page = 1
      this.sort = 'name'
      this.order = 'asc'
      this.dependencyPage = 1
      this.draftVersion = 0
      this.assets = new Map()
      this.root = this.createView()
    }
    createView() {
      const root = document.createElement('div')
      root.className = 'if-asset-library'
      root.innerHTML = '<div class="if-asset-types"></div><div class="if-asset-query"><input aria-label="搜索资源" placeholder="搜索资源"><select aria-label="资源排序"><option value="name:asc">名称升序</option><option value="name:desc">名称降序</option><option value="updatedAt:desc">最近更新</option><option value="createdAt:desc">最近上传</option></select><button type="button" title="上传资源">上传</button><button type="button" class="if-save-selection">保存所选</button><input class="if-asset-upload" type="file" hidden></div><div class="if-asset-status"></div><div class="if-asset-list"></div><div class="if-asset-pages"><button type="button" data-page="prev">上一页</button><span></span><button type="button" data-page="next">下一页</button></div><details><summary>高级依赖诊断</summary><div class="if-dependency-list"></div><div class="if-dependency-pages"><button type="button" data-dependency-page="prev">上一页</button><span></span><button type="button" data-dependency-page="next">下一页</button></div></details>'
      const typeBar = root.querySelector('.if-asset-types')
      categories[this.kind].forEach(([value, label]) => {
        const button = document.createElement('button')
        button.type = 'button'
        button.textContent = label
        button.dataset.type = value
        button.onclick = () => { this.type = value; this.page = 1; this.load() }
        typeBar.appendChild(button)
      })
      const search = root.querySelector('input[aria-label="搜索资源"]')
      let searchTimer
      search.oninput = () => { clearTimeout(searchTimer); searchTimer = setTimeout(() => { this.page = 1; this.load() }, 250) }
      root.querySelector('select[aria-label="资源排序"]').onchange = (event) => {
        ;[this.sort, this.order] = event.target.value.split(':')
        this.page = 1
        this.load()
      }
      const fileInput = root.querySelector('.if-asset-upload')
      root.querySelector('[title="上传资源"]').onclick = () => fileInput.click()
      root.querySelector('.if-save-selection').onclick = () => this.saveSelection()
      fileInput.onchange = () => this.upload(fileInput.files && fileInput.files[0])
      root.querySelector('[data-page="prev"]').onclick = () => { if (this.page > 1) { this.page -= 1; this.load() } }
      root.querySelector('[data-page="next"]').onclick = () => { this.page += 1; this.load() }
      root.querySelector('details').ontoggle = (event) => { if (event.target.open) this.loadDependencies() }
      root.querySelector('[data-dependency-page="prev"]').onclick = () => {
        if (this.dependencyPage > 1) { this.dependencyPage -= 1; this.loadDependencies() }
      }
      root.querySelector('[data-dependency-page="next"]').onclick = () => {
        this.dependencyPage += 1
        this.loadDependencies()
      }
      return root
    }
    async request(suffix, init) {
      return envelope(await fetch(sceneEndpoint(suffix), { credentials: 'same-origin', ...(init || {}) }))
    }
    async load() {
      if (!sceneSessionId()) return
      this.updateMode()
      if (this.type === 'control') {
        this.renderControls()
        return
      }
      const search = this.root.querySelector('input[aria-label="搜索资源"]').value
      try {
        const params = new URLSearchParams({ type: this.type, keyword: search, page: String(this.page), limit: '30', sort: this.sort, order: this.order })
        const payload = await this.request('/assets?' + params)
        this.draftVersion = Number(payload.draftVersion || this.draftVersion)
        this.render(payload)
      } catch (error) {
        this.status(error.message, true)
      }
    }
    render(payload) {
      const list = this.root.querySelector('.if-asset-list')
      list.replaceChildren()
      const builtInCount = this.type === 'symbol' && this.page === 1 ? this.renderBuiltInTemplates(list) : 0
      this.assets = new Map((payload.items || []).map((item) => [item.assetId, item]))
      const bindings = new Map((payload.bindings || []).map((item) => [item.assetId, item]))
      for (const asset of payload.items || []) {
        const binding = bindings.get(asset.assetId)
        const row = document.createElement('div')
        row.className = 'if-asset-row'
        row.draggable = true
        row.ondragstart = (event) => event.dataTransfer.setData('application/x-induforge-asset', asset.assetId)
        const preview = asset.type === 'image' ? '<img src="' + asset.thumbnailUrl + '" alt="">' : '<span class="if-asset-icon">' + asset.type.slice(0, 1).toUpperCase() + '</span>'
        row.innerHTML = preview + '<div><strong></strong><small></small></div><div class="if-asset-actions"></div>'
        row.querySelector('strong').textContent = asset.name
        row.querySelector('small').textContent = binding ? (binding.updateAvailable ? '有更新' : '已添加') : (assetTypeLabels[asset.type] || '资源')
        const actions = row.querySelector('.if-asset-actions')
        actions.appendChild(this.actionButton(binding ? (binding.updateAvailable ? '更新' : '移除') : '添加', async () => {
          await this.assetAction(binding ? (binding.updateAvailable ? 'update' : 'detach') : 'attach', asset)
        }))
        if (asset.type === 'symbol' || asset.type === 'component') actions.appendChild(this.actionButton('编辑', () => this.edit(asset)))
        actions.appendChild(this.actionButton('归档', () => this.archive(asset)))
        row.ondblclick = () => { if (!binding) this.assetAction('attach', asset) }
        list.appendChild(row)
      }
      const pages = Math.max(1, Math.ceil((payload.total || 0) / (payload.limit || 30)))
      this.root.querySelector('.if-asset-pages span').textContent = this.page + ' / ' + pages
      this.root.querySelector('[data-page="prev"]').disabled = this.page <= 1
      this.root.querySelector('[data-page="next"]').disabled = this.page >= pages
      this.root.querySelectorAll('.if-asset-types button').forEach((button) => button.classList.toggle('active', button.dataset.type === this.type))
      this.status(payload.total ? '工程资源 ' + payload.total + ' 项' : builtInCount ? '' : '暂无资源')
    }
    renderBuiltInTemplates(list) {
      ensureBuiltInTemplateImages()
      const keyword = this.root.querySelector('input[aria-label="搜索资源"]').value.trim().toLowerCase()
      const templates = builtInTemplates.filter((template) => !keyword || template.name.toLowerCase().includes(keyword))
      if (!templates.length) return 0
      const grid = document.createElement('div')
      grid.className = 'if-template-grid'
      for (const template of templates) {
        const card = document.createElement('button')
        card.type = 'button'
        card.className = 'if-template-card'
        card.draggable = true
        card.title = '添加' + template.name
        card.innerHTML = '<span class="if-template-preview" data-template="' + template.type + '"></span><strong></strong>'
        card.querySelector('strong').textContent = template.name
        card.ondragstart = (event) => event.dataTransfer.setData('application/x-induforge-template', template.type)
        card.onclick = () => this.insertTemplate(template)
        grid.appendChild(card)
      }
      list.appendChild(grid)
      return templates.length
    }
    updateMode() {
      const isControl = this.type === 'control'
      this.root.classList.toggle('control-mode', isControl)
      this.root.querySelector('.if-asset-query').hidden = isControl
      this.root.querySelector('.if-asset-pages').hidden = isControl
      this.root.querySelector('.if-save-selection').hidden = this.type !== 'symbol' && this.type !== 'component'
      this.root.querySelectorAll('.if-asset-types button').forEach((button) => button.classList.toggle('active', button.dataset.type === this.type))
    }
    renderControls() {
      const list = this.root.querySelector('.if-asset-list')
      list.replaceChildren()
      this.assets.clear()
      for (const control of controls) {
        const card = document.createElement('button')
        card.type = 'button'
        card.className = 'if-control-card'
        card.draggable = true
        card.title = '添加' + control.name
        card.innerHTML = '<span class="if-control-preview" data-control="' + control.type + '"></span><strong></strong>'
        card.querySelector('strong').textContent = control.name
        card.ondragstart = (event) => event.dataTransfer.setData('application/x-induforge-control', control.type)
        card.onclick = () => this.insertControl(control)
        list.appendChild(card)
      }
      this.status('')
    }
    insertControl(control, event) {
      const displayView = this.editor && this.editor.displayView
      const Constructor = window.ht && window.ht[control.constructorName]
      if (!displayView || typeof displayView.addData !== 'function') {
        this.status('当前画布不能添加控件', true)
        return null
      }
      if (typeof Constructor !== 'function') {
        this.status('控件运行模块未加载，请刷新编辑器', true)
        return null
      }
      const data = new Constructor()
      data.setSize(control.width, control.height)
      data.setDisplayName(control.name)
      data.a('induforge.control.type', control.type)
      if (control.init) control.init(data)
      const graphView = displayView.graphView
      if (event && graphView && typeof graphView.getLogicalPoint === 'function' && typeof data.setPosition === 'function') {
        const point = graphView.getLogicalPoint(event)
        if (point) data.setPosition(point.x, point.y)
      }
      displayView.addData(data)
      this.status('')
      return data
    }
    insertTemplate(template, event) {
      ensureBuiltInTemplateImages()
      const displayView = this.editor && this.editor.displayView
      const Constructor = window.ht && window.ht.Node
      if (!displayView || typeof displayView.addData !== 'function' || typeof Constructor !== 'function') {
        this.status('当前画布不能添加图形模板', true)
        return null
      }
      const data = new Constructor()
      data.setImage(template.image)
      data.setSize(template.width, template.height)
      data.setDisplayName(template.name)
      data.a('induforge.template.type', template.type)
      if (template.init) template.init(data)
      const graphView = displayView.graphView
      if (event && graphView && typeof graphView.getLogicalPoint === 'function' && typeof data.setPosition === 'function') {
        const point = graphView.getLogicalPoint(event)
        if (point) data.setPosition(point.x, point.y)
      }
      displayView.addData(data)
      return data
    }
    actionButton(label, action) {
      const button = document.createElement('button')
      button.type = 'button'
      button.textContent = label
      button.onclick = async (event) => { event.stopPropagation(); try { await action() } catch (error) { this.status(error.message, true) } }
      return button
    }
    async assetAction(action, asset) {
      const payload = await this.request('/assets/actions', {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ action, assetId: asset.assetId, baseDraftVersion: this.draftVersion }),
      })
      this.draftVersion = Number(payload.draftVersion)
      await this.load()
      if (action === 'attach') await this.insertAttachedAsset(asset)
    }
    async insertAttachedAsset(asset) {
      const displayView = this.editor && (this.editor.displayView || this.editor.sceneView)
      if (!displayView || !displayView.addData || !['image', 'symbol', 'component', 'model', 'material'].includes(asset.type)) return
      if (asset.type === 'material') {
        this.status('材质已挂载，可在 3D 对象材质属性中选择')
        return
      }
      const root = { symbol: 'symbols', component: 'components', image: 'assets', model: 'models' }[asset.type]
      const resource = root + '/library/' + asset.assetId + '/' + asset.entryFile
      if ((asset.type === 'symbol' || asset.type === 'component') && await this.insertSelectionResource(resource)) {
        this.status('已插入' + asset.name)
        return
      }
      const node = new ht.Node()
      if (asset.type === 'model') node.s('shape3d', resource)
      else node.setImage(resource)
      node.setDisplayName(asset.name)
      displayView.addData(node)
    }
    async insertSelectionResource(resource) {
      const response = await fetch(fileEndpoint('/files/content') + '?' + new URLSearchParams({ path: resource }), { credentials: 'same-origin' })
      if (!response.ok) throw new Error('读取资源内容失败')
      const payload = await response.json().catch(() => null)
      if (!payload || payload.induforgeResourceType !== 'selection' || payload.formatVersion !== 1 || !Array.isArray(payload.datas)) return false
      const displayView = this.editor && this.editor.displayView
      const dm = displayView && displayView.graphView && displayView.graphView.dm
      if (!dm || typeof dm.deserialize !== 'function') throw new Error('当前画布不能还原复合资源')
      const inserted = dm.deserialize({ d: payload.datas }, null, { justDatas: true })
      const selection = dm.sm && dm.sm()
      if (selection && typeof selection.ss === 'function' && inserted) selection.ss(inserted)
      return true
    }
    async saveSelection() {
      const view = this.editor && (this.editor.displayView || this.editor.sceneView)
      const dm = view && view.graphView && view.graphView.dm
      const sm = dm && dm.sm && dm.sm()
      if (!sm) return this.status('当前画布不可读取选中内容', true)
      const datas = []
      if (typeof sm.each === 'function') sm.each((data) => { if (data && typeof data.serialize === 'function') datas.push(data.serialize()) })
      const last = sm.ld && sm.ld()
      if (!datas.length && last && typeof last.serialize === 'function') datas.push(last.serialize())
      if (!datas.length) return this.status('请先选择要保存的画布内容', true)
      const name = window.prompt(this.type === 'symbol' ? '图形模板名称' : '业务组件名称')
      if (!name || !name.trim()) return
      try {
        await this.request('/assets/from-selection', {
          method: 'POST', headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ type: this.type, name: name.trim(), selection: { induforgeResourceType: 'selection', formatVersion: 1, datas } }),
        })
        this.status('已保存到工程资源库')
        await this.load()
      } catch (error) { this.status(error.message, true) }
    }
    async upload(file) {
      if (!file) return
      try {
        const params = new URLSearchParams({ type: this.type, name: file.name.replace(/\.[^.]+$/, ''), filename: file.name })
        await this.request('/assets/import?' + params, { method: 'POST', headers: { 'Content-Type': file.type || 'application/octet-stream' }, body: file })
        this.status('上传成功')
        this.page = 1
        await this.load()
      } catch (error) {
        this.status(error.message, true)
      } finally {
        this.root.querySelector('.if-asset-upload').value = ''
      }
    }
    async edit(asset) {
      const session = await this.request('/assets/' + encodeURIComponent(asset.assetId) + '/editor-session', { method: 'POST' })
      window.open(session.url, 'induforge-asset-' + asset.assetId)
    }
    async archive(asset) {
      await this.request('/assets/' + encodeURIComponent(asset.assetId), { method: 'DELETE' })
      await this.load()
    }
    async loadDependencies() {
      try {
        const payload = await this.request('/dependencies?' + new URLSearchParams({ page: String(this.dependencyPage), limit: '50', sort: 'name', order: 'asc' }))
        const target = this.root.querySelector('.if-dependency-list')
        target.replaceChildren(...(payload.items || []).map((item) => {
          const line = document.createElement('div')
          const source = item.sourceType === 'asset' ? (item.sourceAssetName || '资源库') : '当前场景'
          const status = item.status === 'available' ? '正常' : '缺失'
          line.textContent = item.path + ' · ' + source + ' · ' + status + (item.contentHash ? ' · ' + item.contentHash.slice(0, 8) : '')
          return line
        }))
        const pages = Math.max(1, Math.ceil((payload.total || 0) / (payload.limit || 50)))
        this.root.querySelector('.if-dependency-pages span').textContent = this.dependencyPage + ' / ' + pages
        this.root.querySelector('[data-dependency-page="prev"]').disabled = this.dependencyPage <= 1
        this.root.querySelector('[data-dependency-page="next"]').disabled = this.dependencyPage >= pages
      } catch (error) {
        this.status(error.message, true)
      }
    }
    status(message, error) {
      const target = this.root.querySelector('.if-asset-status')
      target.textContent = message || ''
      target.classList.toggle('error', Boolean(error))
      target.hidden = !message
    }
  }

  function installStyles() {
    if (document.getElementById('induforge-asset-styles')) return
    const style = document.createElement('style')
    style.id = 'induforge-asset-styles'
    style.textContent = '.if-asset-library{height:100%;display:grid;grid-template-rows:auto auto auto minmax(0,1fr) auto auto;background:#f7f8f6;color:#18201d;font:12px Arial,sans-serif}.if-asset-types{display:grid;grid-template-columns:repeat(2,1fr);gap:4px;padding:8px;border-bottom:1px solid #d7ddd8}.if-asset-library button{min-height:28px;border:1px solid #b7c0ba;border-radius:4px;background:#fff;color:#18201d;cursor:pointer}.if-asset-library button.active{border-color:#18201d;background:#e6f55c}.if-asset-library button:disabled{opacity:.45;cursor:default}.if-asset-query{display:grid;grid-template-columns:minmax(0,1fr) 88px 56px;gap:5px;padding:7px 8px}.if-asset-query[hidden],.if-asset-pages[hidden],.if-asset-status[hidden]{display:none}.if-asset-query input,.if-asset-query select{min-width:0;border:1px solid #b7c0ba;border-radius:4px;padding:6px;background:#fff}.if-asset-status{min-height:18px;padding:0 8px;color:#65706a}.if-asset-status.error{color:#a52b1e}.if-asset-list{overflow:auto;padding:0 8px}.if-asset-library.control-mode .if-asset-list{display:grid;grid-template-columns:repeat(auto-fill,minmax(62px,1fr));gap:6px;align-content:start;padding:8px}.if-template-grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:6px;margin-bottom:8px;padding-bottom:8px;border-bottom:1px solid #d7ddd8}.if-asset-row{display:grid;grid-template-columns:36px minmax(0,1fr) auto;gap:7px;align-items:center;min-height:48px;border-bottom:1px solid #dde2de}.if-asset-row img,.if-asset-icon{width:32px;height:32px;object-fit:contain;border:1px solid #d1d7d2;border-radius:4px;background:#fff;display:grid;place-items:center}.if-asset-row strong,.if-asset-row small{display:block;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}.if-asset-row small{margin-top:3px;color:#69736d}.if-asset-actions{display:flex;gap:3px}.if-asset-actions button{min-height:24px;padding:2px 5px}.if-control-card,.if-template-card{min-width:0;min-height:0!important;display:flex;flex-direction:column;justify-content:center;align-items:center;gap:6px;padding:6px;overflow:hidden;text-align:center}.if-control-card{aspect-ratio:1}.if-template-card{aspect-ratio:1.25}.if-control-card strong,.if-template-card strong{display:block;width:100%;overflow:hidden;text-overflow:ellipsis;font-size:12px;font-weight:500}.if-control-card strong{white-space:nowrap}.if-template-card strong{white-space:normal;line-height:16px}.if-control-preview,.if-template-preview{flex:none;width:32px;height:22px;position:relative;border:1px solid #69736d;background:#eef1ef}.if-template-preview{width:48px;height:30px}.if-control-preview:after,.if-template-preview:after{content:"";position:absolute;inset:6px;background:#26a269}.if-control-preview[data-control="checkbox"],.if-control-preview[data-control="radio"]{width:22px;height:22px}.if-control-preview[data-control="checkbox"]:after{inset:5px 4px;border:solid #26a269;border-width:0 0 2px 2px;background:transparent;transform:rotate(-45deg)}.if-control-preview[data-control="radio"]{border-radius:50%}.if-control-preview[data-control="radio"]:after{inset:5px;border-radius:50%}.if-control-preview[data-control="switch"],.if-control-preview[data-control="toggle-button"]{border-radius:12px}.if-control-preview[data-control="switch"]:after,.if-control-preview[data-control="toggle-button"]:after{inset:4px 4px 4px 16px;border-radius:50%}.if-control-preview[data-control="progress"]:after{inset:7px 14px 7px 3px}.if-control-preview[data-control="slider"]:before{content:"";position:absolute;left:4px;right:4px;top:10px;border-top:2px solid #69736d}.if-control-preview[data-control="slider"]:after{inset:5px 12px;border-radius:50%}.if-control-preview[data-control="number-input"]:after{content:"1";inset:2px 10px 2px 3px;background:transparent;color:#18201d}.if-template-preview[data-template="progress-horizontal"]:after{inset:9px 20px 9px 4px}.if-template-preview[data-template="progress-vertical"]:after{inset:12px 20px 4px}.if-template-preview[data-template="table"]{background:repeating-linear-gradient(0deg,#fff 0 8px,#7b8781 9px),repeating-linear-gradient(90deg,transparent 0 14px,#7b8781 15px)}.if-template-preview[data-template="table"]:after{display:none}.if-template-preview[data-template="clock"]{width:30px;height:30px;border-radius:50%}.if-template-preview[data-template="clock"]:after{inset:6px 14px 6px 13px;transform:rotate(-35deg);transform-origin:bottom}.if-asset-pages,.if-dependency-pages{display:grid;grid-template-columns:60px 1fr 60px;gap:5px;align-items:center;padding:7px 8px;text-align:center;border-top:1px solid #d7ddd8}.if-asset-library details{max-height:180px;overflow:auto;border-top:1px solid #d7ddd8;padding:7px 8px}.if-dependency-list{margin-top:6px;font:10px monospace;color:#59635d;overflow-wrap:anywhere}.if-dependency-pages{padding:6px 0 0;margin-top:6px}.if-dependency-pages button{min-height:24px}'
    document.head.appendChild(style)
  }

  window.InduForgeAssets = {
    mount(editor, kind) {
      if (!sceneSessionId() || !editor || !editor.leftTopTabView) return null
      applySceneDisplayName(editor, kind)
      installStyles()
      ;['symbolsTab', 'componentsTab', 'assetsTab', 'modelsTab'].forEach((name) => {
        if (editor[name] && editor[name].setVisible) editor[name].setVisible(false)
      })
      const library = new AssetLibrary(editor, kind)
      const tabView = editor.leftTopTabView
      const originalOnTabChanged = tabView.onTabChanged
      tabView.onTabChanged = function(oldTab, newTab) {
        if (newTab && newTab.getView && newTab.getView() === library.root) return
        if (originalOnTabChanged) return originalOnTabChanged.apply(this, arguments)
      }
      const tab = new ht.Tab()
      tab.setName('资源库')
      tab.setView(library.root)
      tabView.getTabModel().add(tab)
      if (tabView.select) tabView.select(tab)
      const dropView = editor.mainPane && editor.mainPane.getView ? editor.mainPane.getView() : null
      if (dropView) {
        dropView.addEventListener('dragover', (event) => event.preventDefault())
        dropView.addEventListener('drop', async (event) => {
          const controlType = event.dataTransfer.getData('application/x-induforge-control')
          const templateType = event.dataTransfer.getData('application/x-induforge-template')
          const assetId = event.dataTransfer.getData('application/x-induforge-asset')
          if (!controlType && !templateType && !assetId) return
          event.preventDefault()
          if (controlType) {
            const control = controls.find((item) => item.type === controlType)
            if (control) library.insertControl(control, event)
            return
          }
          if (templateType) {
            const template = builtInTemplates.find((item) => item.type === templateType)
            if (template) library.insertTemplate(template, event)
            return
          }
          const asset = library.assets.get(assetId)
          if (asset) await library.assetAction('attach', asset)
        })
      }
      library.load()
      return library
    },
    async openAssetDraft(editor) {
      if (!assetSessionId() || !editor) return
      const payload = await envelope(await fetch(fileEndpoint('/files') + '?' + new URLSearchParams({ page: '1', limit: '200' }), { credentials: 'same-origin' }))
      const entry = (payload.items || []).find((item) => item.path === payload.entryFile)
      if (!entry) return
      const content = await fetch(fileEndpoint('/files/content') + '?' + new URLSearchParams({ path: entry.path }), { credentials: 'same-origin' }).then((response) => response.text())
      const type = payload.assetType === 'component' ? 'component' : 'symbol'
      editor.openByJSON(type, entry.path, entry.name, content)
    },
  }
})()
