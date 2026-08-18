;(function () {
  const roots = new Set(['assets', 'components', 'displays', 'materials', 'models', 'previews', 'scenes', 'symbols'])
  const query = (name) => new URLSearchParams(window.location.search).get(name) || ''
  const sceneSessionId = () => query('sessionId')
  const assetSessionId = () => query('assetSessionId')
  const fileEndpoint = (suffix) => {
    if (assetSessionId()) return '/api/v1/scene-asset-editor-sessions/' + encodeURIComponent(assetSessionId()) + suffix
    if (sceneSessionId()) return '/api/v1/scene-editor-sessions/' + encodeURIComponent(sceneSessionId()) + suffix
    return ''
  }
  const sceneEndpoint = (suffix) => sceneSessionId()
    ? '/api/v1/scene-editor-sessions/' + encodeURIComponent(sceneSessionId()) + suffix
    : ''
  const explorerFileTypes = {
    displays: { fileType: 'display', fileImage: 'editor.display' },
    scenes: { fileType: 'scene', fileImage: 'editor.scene' },
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
    const currentSessionId = assetSessionId() || sceneSessionId()
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
    if (!response.ok || !payload || payload.code !== 0) throw new Error(payload && payload.msg ? payload.msg : '场景资源操作失败')
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
      this.ready = this.initialize()
    }
    async initialize() {
      if (!sceneSessionId() && !assetSessionId()) {
        this.handler({ type: 'disconnected', message: '缺少编辑会话' })
        throw new Error('缺少编辑会话')
      }
      const result = await this.list('', 1)
      this.draftVersion = Number(result.draftVersion || 0)
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
      const body = uploadBlob(data.content)
      if (/\.zip$/i.test(data.path) && sceneSessionId()) return this.importArchive(body)
      const params = new URLSearchParams({ path: data.path, baseDraftVersion: String(this.draftVersion) })
      const payload = await envelope(await fetch(fileEndpoint('/files/content') + '?' + params, {
        method: 'PUT', credentials: 'same-origin', headers: { 'Content-Type': body.type || 'application/octet-stream' }, body,
      }))
      this.draftVersion = Number(payload.draftVersion)
      if (/\.json$/i.test(data.path)) await this.commit()
      return true
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

  const categories = {
    '2d': [['image', '图片'], ['font', '字体'], ['symbol', 'Symbol'], ['component', 'Component']],
    '3d': [['image', '贴图'], ['font', '字体'], ['model', '模型'], ['material', '材质']],
  }

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
      root.innerHTML = '<div class="if-asset-types"></div><div class="if-asset-query"><input aria-label="搜索资源" placeholder="搜索资源"><select aria-label="资源排序"><option value="name:asc">名称升序</option><option value="name:desc">名称降序</option><option value="updatedAt:desc">最近更新</option><option value="createdAt:desc">最近上传</option></select><button type="button" title="上传资源">上传</button><input class="if-asset-upload" type="file" hidden></div><div class="if-asset-status"></div><div class="if-asset-list"></div><div class="if-asset-pages"><button type="button" data-page="prev">上一页</button><span></span><button type="button" data-page="next">下一页</button></div><details><summary>高级依赖诊断</summary><div class="if-dependency-list"></div><div class="if-dependency-pages"><button type="button" data-dependency-page="prev">上一页</button><span></span><button type="button" data-dependency-page="next">下一页</button></div></details>'
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
        row.querySelector('small').textContent = binding ? (binding.updateAvailable ? '有更新' : '已添加') : asset.type
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
      this.status(payload.total ? '共 ' + payload.total + ' 项' : '暂无资源')
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
      if (action === 'attach') this.insertAttachedAsset(asset)
    }
    insertAttachedAsset(asset) {
      const displayView = this.editor && this.editor.displayView
      if (!displayView || !displayView.addData || (asset.type !== 'image' && asset.type !== 'symbol')) return
      const node = new ht.Node()
      const root = asset.type === 'symbol' ? 'symbols' : 'assets'
      node.setImage(root + '/library/' + asset.assetId + '/' + asset.entryFile)
      node.setDisplayName(asset.name)
      displayView.addData(node)
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
    }
  }

  function installStyles() {
    if (document.getElementById('induforge-asset-styles')) return
    const style = document.createElement('style')
    style.id = 'induforge-asset-styles'
    style.textContent = '.if-asset-library{height:100%;display:grid;grid-template-rows:auto auto auto minmax(0,1fr) auto auto;background:#f7f8f6;color:#18201d;font:12px Arial,sans-serif}.if-asset-types{display:grid;grid-template-columns:repeat(2,1fr);gap:4px;padding:8px;border-bottom:1px solid #d7ddd8}.if-asset-library button{min-height:28px;border:1px solid #b7c0ba;border-radius:4px;background:#fff;color:#18201d;cursor:pointer}.if-asset-library button.active{border-color:#18201d;background:#e6f55c}.if-asset-library button:disabled{opacity:.45;cursor:default}.if-asset-query{display:grid;grid-template-columns:minmax(0,1fr) 88px 56px;gap:5px;padding:7px 8px}.if-asset-query input,.if-asset-query select{min-width:0;border:1px solid #b7c0ba;border-radius:4px;padding:6px;background:#fff}.if-asset-status{min-height:18px;padding:0 8px;color:#65706a}.if-asset-status.error{color:#a52b1e}.if-asset-list{overflow:auto;padding:0 8px}.if-asset-row{display:grid;grid-template-columns:36px minmax(0,1fr) auto;gap:7px;align-items:center;min-height:48px;border-bottom:1px solid #dde2de}.if-asset-row img,.if-asset-icon{width:32px;height:32px;object-fit:contain;border:1px solid #d1d7d2;border-radius:4px;background:#fff;display:grid;place-items:center}.if-asset-row strong,.if-asset-row small{display:block;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}.if-asset-row small{margin-top:3px;color:#69736d}.if-asset-actions{display:flex;gap:3px}.if-asset-actions button{min-height:24px;padding:2px 5px}.if-asset-pages,.if-dependency-pages{display:grid;grid-template-columns:60px 1fr 60px;gap:5px;align-items:center;padding:7px 8px;text-align:center;border-top:1px solid #d7ddd8}.if-asset-library details{max-height:180px;overflow:auto;border-top:1px solid #d7ddd8;padding:7px 8px}.if-dependency-list{margin-top:6px;font:10px monospace;color:#59635d;overflow-wrap:anywhere}.if-dependency-pages{padding:6px 0 0;margin-top:6px}.if-dependency-pages button{min-height:24px}'
    document.head.appendChild(style)
  }

  window.InduForgeAssets = {
    mount(editor, kind) {
      if (!sceneSessionId() || !editor || !editor.leftTopTabView) return null
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
          const assetId = event.dataTransfer.getData('application/x-induforge-asset')
          if (!assetId) return
          event.preventDefault()
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
