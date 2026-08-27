;(function () {
  const query = (name) => new URLSearchParams(window.location.search).get(name) || ''
  const sessionId = () => query('sessionId')
  const endpoint = (suffix) => '/api/v1/scene-editor-sessions/' + encodeURIComponent(sessionId()) + suffix

  async function envelope(response) {
    const payload = await response.json().catch(() => null)
    if (!response.ok || !payload || payload.code !== 0) throw new Error(payload && payload.msg ? payload.msg : '操作失败')
    return payload.data
  }

  function selectedData(editor) {
    const view = editor && (editor.displayView || editor.sceneView)
    const dm = (view && view.graphView && view.graphView.dm) || (editor && editor.dm)
    const sm = (editor && editor.sm) || (dm && dm.sm && dm.sm())
    return sm && sm.ld ? sm.ld() : null
  }

  function controlValueType(data) {
    const type = data && data.a && data.a('induforge.control.type')
    if (type === 'checkbox' || type === 'radio' || type === 'switch' || type === 'toggle-button') return 'boolean'
    if (type === 'slider' || type === 'number-input' || type === 'progress') return 'number'
    return 'string'
  }

  function eventTriggers(data) {
    const type = data && data.a && data.a('induforge.control.type')
    if (type === 'button') return [['click', '点击'], ['press', '按下'], ['release', '释放']]
    if (type === 'slider' || type === 'number-input') return [['input', '输入中'], ['change', '确认变化']]
    if (type) return [['change', '值变化']]
    if (data && data.a && data.a('induforge.path.type') === 'pipe') return [['click', '点击'], ['doubleClick', '双击'], ['flowStateChange', '流动状态变化']]
    return [['click', '点击'], ['doubleClick', '双击']]
  }

  function bindingTargets(data) {
    const type = data && data.a && data.a('induforge.control.type')
    if (type === 'checkbox' || type === 'radio' || type === 'switch' || type === 'toggle-button') return [['selected', '选中状态']]
    if (type === 'slider' || type === 'number-input' || type === 'progress' || type === 'select') return [['value', '控件值']]
    if (data && data.a && data.a('induforge.path.type') === 'pipe') return [['pipe.running', '管道运行状态'], ['color', '介质颜色'], ['visible', '可见性']]
    return [['text', '文本'], ['value', '值'], ['visible', '可见性'], ['color', '颜色'], ['state', '对象状态']]
  }

  function formRow(label, control) {
    const row = document.createElement('label')
    row.className = 'if-binding-row'
    const text = document.createElement('span')
    text.textContent = label
    row.append(text, control)
    return row
  }

  function selectControl(options) {
    const select = document.createElement('select')
    for (const option of options) {
      const node = document.createElement('option')
      node.value = option[0]
      node.textContent = option[1]
      select.appendChild(node)
    }
    return select
  }

  class DataBindingPanel {
    constructor(editor) {
      this.editor = editor
      this.root = document.createElement('div')
      this.root.className = 'if-binding-panel'
      this.points = []
      this.page = 1
      this.totalPages = 1
      this.keyword = ''
      this.render()
      this.loadPoints()
    }
    async loadPoints(keyword) {
      try {
        if (keyword != null) this.keyword = keyword
        const params = new URLSearchParams({ page: String(this.page), pageSize: '100', search: this.keyword, sortField: 'path', sortOrder: 'asc' })
        const payload = await envelope(await fetch(endpoint('/datapoints') + '?' + params, { credentials: 'same-origin' }))
        this.points = payload.datapoints || payload.items || []
        this.totalPages = Math.max(1, Number(payload.pagination && payload.pagination.totalPages) || 1)
        this.fillPoints()
        const pageStatus = this.root.querySelector('.if-binding-pages span')
        if (pageStatus) pageStatus.textContent = this.page + ' / ' + this.totalPages
        const previous = this.root.querySelector('[data-points-page="prev"]')
        const next = this.root.querySelector('[data-points-page="next"]')
        if (previous) previous.disabled = this.page <= 1
        if (next) next.disabled = this.page >= this.totalPages
      } catch (error) { this.status(error.message, true) }
    }
    fillPoints() {
      const select = this.root.querySelector('[data-field="pointPath"]')
      if (!select) return
      const current = select.value
      select.replaceChildren()
      for (const point of this.points) {
        const option = document.createElement('option')
        option.value = point.path
        option.textContent = point.path + ' · ' + point.dataType
        option.disabled = point.status !== 'active'
        option.dataset.dataType = point.dataType || ''
        option.dataset.get = String(point.capabilities && point.capabilities.get !== false)
        option.dataset.sub = String(point.capabilities && point.capabilities.sub !== false)
        option.dataset.set = String(point.capabilities && point.capabilities.set === true)
        select.appendChild(option)
      }
      if (current) select.value = current
    }
    render() {
      this.root.innerHTML = '<div class="if-binding-empty">请选择画布对象</div><div class="if-binding-form" hidden></div><div class="if-binding-status"></div>'
      this.refresh()
    }
    refresh() {
      const data = selectedData(this.editor)
      const form = this.root.querySelector('.if-binding-form')
      this.root.querySelector('.if-binding-empty').hidden = Boolean(data)
      form.hidden = !data
      if (!data) return
      form.replaceChildren()
      const search = document.createElement('input')
      search.placeholder = '搜索数据点'
      let timer
      search.value = this.keyword
      search.oninput = () => { clearTimeout(timer); timer = setTimeout(() => { this.page = 1; this.loadPoints(search.value.trim()) }, 250) }
      const point = selectControl([]); point.dataset.field = 'pointPath'
      const target = selectControl(bindingTargets(data)); target.dataset.field = 'target'
      const direction = selectControl([['read', '只读'], ['write', '只写'], ['twoWay', '双向']]); direction.dataset.field = 'direction'
      const readMode = selectControl([['query', '查询一次'], ['subscribe', '实时订阅']]); readMode.dataset.field = 'readMode'
      const writeTrigger = selectControl([['confirm', '确认变化'], ['change', '值变化'], ['click', '明确操作']]); writeTrigger.dataset.field = 'writeTrigger'
      const precision = document.createElement('input'); precision.type = 'number'; precision.min = '0'; precision.max = '12'; precision.dataset.field = 'precision'
      const prefix = document.createElement('input'); prefix.dataset.field = 'prefix'
      const suffix = document.createElement('input'); suffix.dataset.field = 'suffix'
      const nullText = document.createElement('input'); nullText.dataset.field = 'nullText'
      const qualityFallback = document.createElement('input'); qualityFallback.dataset.field = 'qualityFallback'
      const pages = document.createElement('div'); pages.className = 'if-binding-pages'; pages.innerHTML = '<button type="button" data-points-page="prev">上一页</button><span></span><button type="button" data-points-page="next">下一页</button>'
      pages.querySelector('[data-points-page="prev"]').onclick = () => { if (this.page > 1) { this.page -= 1; this.loadPoints() } }
      pages.querySelector('[data-points-page="next"]').onclick = () => { if (this.page < this.totalPages) { this.page += 1; this.loadPoints() } }
      const save = document.createElement('button'); save.type = 'button'; save.textContent = '保存联动'; save.onclick = () => this.save(data)
      const clear = document.createElement('button'); clear.type = 'button'; clear.textContent = '移除联动'; clear.onclick = () => {
        const bindings = Object.assign({}, data.a('induforge.bindings') || {})
        delete bindings[target.value]
        data.a('induforge.bindings', Object.keys(bindings).length ? bindings : null)
        this.refresh()
      }
      const actions = document.createElement('div'); actions.className = 'if-binding-actions'; actions.append(save, clear)
      form.append(search, pages, formRow('数据点', point), formRow('目标属性', target), formRow('方向', direction), formRow('读取方式', readMode), formRow('写入时机', writeTrigger), formRow('数字精度', precision), formRow('前缀', prefix), formRow('后缀', suffix), formRow('空值文案', nullText), formRow('异常质量', qualityFallback), actions)
      this.fillPoints()
      const bindings = data.a('induforge.bindings') || {}
      const names = Object.keys(bindings)
      if (names.length) {
        const binding = bindings[names[0]] || {}
        target.value = names[0]
        for (const name of ['pointPath', 'direction', 'readMode', 'writeTrigger']) if (binding[name] != null) form.querySelector('[data-field="' + name + '"]').value = binding[name]
        const format = binding.format || {}
        for (const name of ['precision', 'prefix', 'suffix', 'nullText', 'qualityFallback']) if (format[name] != null) form.querySelector('[data-field="' + name + '"]').value = format[name]
      }
    }
    save(data) {
      const form = this.root.querySelector('.if-binding-form')
      const value = (name) => form.querySelector('[data-field="' + name + '"]').value
      const target = value('target')
      const point = form.querySelector('[data-field="pointPath"]')
      const option = point.options[point.selectedIndex]
      if (!option || !value('pointPath')) return this.status('请选择数据点', true)
      const direction = value('direction')
      if (direction !== 'write' && option.dataset.get !== 'true') return this.status('该数据点不支持读取', true)
      if (direction !== 'read' && option.dataset.set !== 'true') return this.status('该数据点不支持写入', true)
      if (direction !== 'write' && value('readMode') === 'subscribe' && option.dataset.sub !== 'true') return this.status('该数据点不支持订阅', true)
      const precision = value('precision')
      const bindings = Object.assign({}, data.a('induforge.bindings') || {})
      bindings[target] = {
          pointPath: value('pointPath'), direction: direction, readMode: value('readMode'), writeTrigger: value('writeTrigger'), valueType: option.dataset.dataType,
          format: { precision: precision === '' ? null : Number(precision), prefix: value('prefix'), suffix: value('suffix'), nullText: value('nullText'), qualityFallback: value('qualityFallback') },
      }
      data.a('induforge.bindings', bindings)
      this.status('已保存，场景保存后生效')
    }
    status(message, error) {
      const target = this.root.querySelector('.if-binding-status')
      target.textContent = message || ''
      target.classList.toggle('error', Boolean(error))
    }
  }

  class InteractionPanel {
    constructor(editor) { this.editor = editor; this.root = document.createElement('div'); this.root.className = 'if-binding-panel'; this.refresh() }
    refresh() {
      const data = selectedData(this.editor)
      this.root.replaceChildren()
      if (!data) { const empty = document.createElement('div'); empty.className = 'if-binding-empty'; empty.textContent = '请选择画布对象'; this.root.appendChild(empty); return }
      const interactions = data.a('induforge.interactions') || { events: [], commands: [] }
      const animation = selectControl([['off', '无'], ['blink', '闪烁'], ['pulse', '脉冲'], ['rotate', '旋转']])
      animation.value = data.a('induforge.animation.preset') || 'off'
      const animationSpeed = document.createElement('input'); animationSpeed.type = 'number'; animationSpeed.min = '0.25'; animationSpeed.max = '4'; animationSpeed.step = '0.25'; animationSpeed.value = String(data.a('induforge.animation.speed') || 1)
      const applyAnimation = document.createElement('button'); applyAnimation.type = 'button'; applyAnimation.textContent = '应用动画'
      applyAnimation.onclick = () => {
        data.a('induforge.animation.preset', animation.value)
        data.a('induforge.animation.speed', Math.max(0.25, Math.min(4, Number(animationSpeed.value) || 1)))
        if (window.InduForgeAnimation) window.InduForgeAnimation.apply(data)
      }
      const eventName = document.createElement('input'); eventName.placeholder = '公开事件名'
      const trigger = selectControl(eventTriggers(data))
      const addEvent = document.createElement('button'); addEvent.type = 'button'; addEvent.textContent = '添加事件'
      addEvent.onclick = () => {
        if (!/^[A-Za-z][A-Za-z0-9_.:-]{0,63}$/.test(eventName.value.trim())) return
        interactions.events = (interactions.events || []).filter((item) => item.name !== eventName.value.trim())
        interactions.events.push({ name: eventName.value.trim(), trigger: trigger.value, valueType: controlValueType(data) })
        data.a('induforge.interactions', interactions); this.refresh()
      }
      const commandName = document.createElement('input'); commandName.placeholder = '公开命令名'
      const action = selectControl([['setValue', '设置值'], ['show', '显示'], ['hide', '隐藏'], ['enable', '启用'], ['disable', '禁用'], ['setPipeState', '设置管道状态'], ['focus', '聚焦'], ['highlight', '高亮']])
      const addCommand = document.createElement('button'); addCommand.type = 'button'; addCommand.textContent = '添加命令'
      addCommand.onclick = () => {
        if (!/^[A-Za-z][A-Za-z0-9_.:-]{0,63}$/.test(commandName.value.trim())) return
        interactions.commands = (interactions.commands || []).filter((item) => item.name !== commandName.value.trim())
        interactions.commands.push({ name: commandName.value.trim(), action: action.value, valueType: controlValueType(data) })
        data.a('induforge.interactions', interactions); this.refresh()
      }
      this.root.append(formRow('动画预设', animation), formRow('动画速度', animationSpeed), applyAnimation, formRow('触发器', trigger), formRow('事件名', eventName), addEvent, this.list(interactions.events || [], (name) => { interactions.events = interactions.events.filter((item) => item.name !== name); data.a('induforge.interactions', interactions); this.refresh() }), formRow('对象操作', action), formRow('命令名', commandName), addCommand, this.list(interactions.commands || [], (name) => { interactions.commands = interactions.commands.filter((item) => item.name !== name); data.a('induforge.interactions', interactions); this.refresh() }))
    }
    list(items, remove) {
      const list = document.createElement('div'); list.className = 'if-binding-list'
      for (const item of items) {
        const row = document.createElement('div'); const label = document.createElement('span'); label.textContent = item.name + ' · ' + (item.trigger || item.action)
        const button = document.createElement('button'); button.type = 'button'; button.textContent = '移除'; button.onclick = () => remove(item.name)
        row.append(label, button); list.appendChild(row)
      }
      return list
    }
  }

  function installStyles() {
    if (document.getElementById('induforge-binding-styles')) return
    const style = document.createElement('style'); style.id = 'induforge-binding-styles'
    style.textContent = '.if-binding-panel{height:100%;box-sizing:border-box;overflow:auto;padding:10px;background:#f7f8f6;color:#18201d;font:12px Arial,sans-serif}.if-binding-empty{padding:18px 8px;color:#69736d;text-align:center}.if-binding-form{display:grid;gap:7px}.if-binding-row{display:grid;grid-template-columns:72px minmax(0,1fr);gap:6px;align-items:center}.if-binding-panel input,.if-binding-panel select,.if-binding-panel button{box-sizing:border-box;min-width:0;min-height:28px;border:1px solid #b7c0ba;border-radius:4px;background:#fff;padding:5px;color:#18201d}.if-binding-panel button{cursor:pointer}.if-binding-pages{display:grid;grid-template-columns:64px 1fr 64px;gap:5px;align-items:center;text-align:center}.if-binding-actions{display:grid;grid-template-columns:1fr 1fr;gap:6px}.if-binding-list{display:grid;gap:4px;margin:6px 0 12px}.if-binding-list>div{display:grid;grid-template-columns:minmax(0,1fr) 48px;gap:4px;align-items:center}.if-binding-list button{min-height:24px;padding:2px}.if-binding-status{min-height:18px;margin-top:6px;color:#397357}.if-binding-status.error{color:#a52b1e}'
    document.head.appendChild(style)
  }

  window.InduForgeBindings = {
    mount(editor) {
      if (!sessionId() || !editor || !editor.rightBottomTabView) return
      installStyles()
      const dataPanel = new DataBindingPanel(editor)
      const interactionPanel = new InteractionPanel(editor)
      addTab(editor.rightBottomTabView, '数据联动', dataPanel.root)
      addTab(editor.rightBottomTabView, '交互', interactionPanel.root)
      const view = editor.displayView || editor.sceneView
      const dm = (view && view.graphView && view.graphView.dm) || editor.dm
      const sm = editor.sm || (dm && dm.sm && dm.sm())
      if (window.InduForgeAnimation) window.InduForgeAnimation.applyModel(dm)
      if (sm && sm.addSelectionChangeListener) sm.addSelectionChangeListener(() => { dataPanel.refresh(); interactionPanel.refresh() })
    },
  }

  function addTab(tabView, name, view) {
    if (typeof tabView.add === 'function') return tabView.add(name, view, false)
    const tab = new ht.Tab()
    tab.setName(name)
    tab.setView(view)
    tabView.getTabModel().add(tab)
    return tab
  }
})()
