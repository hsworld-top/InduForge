import { mkdtempSync, mkdirSync, readFileSync, writeFileSync } from 'node:fs'
import { tmpdir } from 'node:os'
import path from 'node:path'
import { runInNewContext } from 'node:vm'
import { describe, expect, it, vi } from 'vitest'
import {
  enforceHTSingleEntryPolicy,
  pruneHTProduction,
  resolveSceneStudioAsset,
  rewriteSceneStudioIdentifiers,
  scanHTProduction,
} from '../../../scripts/ht-editor/production-policy.mjs'

describe('HT 生产文件策略', () => {
  it('只保留正式根入口并移除示例与旧配置', () => {
    const root = mkdtempSync(path.join(tmpdir(), 'induforge-ht-policy-'))
    mkdirSync(path.join(root, 'custom/previews'), { recursive: true })
    mkdirSync(path.join(root, 'custom/configs-customstyle'), { recursive: true })
    writeFileSync(path.join(root, 'index.html'), '<title>InduForge 2D 编辑器</title>')
    writeFileSync(path.join(root, 'display-pump.html'), '<title>demo</title>')
    writeFileSync(path.join(root, 'custom/previews/display.html'), 'demo')
    writeFileSync(path.join(root, 'custom/configs-customstyle/config.js'), 'demo')

    pruneHTProduction(root)

    expect(readFileSync(path.join(root, 'index.html'), 'utf8')).toContain('InduForge')
    expect(() => readFileSync(path.join(root, 'display-pump.html'))).toThrow()
    expect(() => readFileSync(path.join(root, 'custom/previews/display.html'))).toThrow()
  })

  it('拒绝正式入口中的厂商链接和旧字符串消息', () => {
    const root = mkdtempSync(path.join(tmpdir(), 'induforge-ht-scan-'))
    writeFileSync(path.join(root, 'index.html'), `<script>window.postMessage('legacy')</script>`)
    expect(() => scanHTProduction(root)).toThrow(/旧字符串消息协议/)
  })

  it('拒绝重新暴露 HT 原生作品管理标签', () => {
    const root = mkdtempSync(path.join(tmpdir(), 'induforge-ht-native-work-'))
    mkdirSync(path.join(root, 'custom/configs/2d'), { recursive: true })
    writeFileSync(
      path.join(root, 'custom/configs/2d/config-onEditorCreated.js'),
      `editor.displaysTab.setName('当前场景')`,
    )

    expect(() => scanHTProduction(root)).toThrow(/原生作品管理标签/)
  })

  it('正式 HT 配置保持一个平台场景对应一个根入口', () => {
    const root = path.resolve(process.cwd(), 'public/ht-editor')
    expect(() => enforceHTSingleEntryPolicy(root)).not.toThrow()
    const toolbarSource = readFileSync(
      path.join(root, 'custom/configs/2d/config-onMainToolbarCreated.js'),
      'utf8',
    )
    expect(toolbarSource).not.toContain('InduForgeSwitch')
    expect(toolbarSource).not.toMatch(/id\s*=\s*'(?:Node|Group|SubGraph)'/)
    expect(toolbarSource).not.toMatch(/symbols\/basic\/(?:node|group|subgraph)\.json/)
    expect(toolbarSource).not.toMatch(/StraightPath|PolylinePath|CurvePath/)
  })

  it('新建管道默认包含管壁、介质和流动参数', () => {
    const root = path.resolve(process.cwd(), 'public/ht-editor')
    const toolbarSource = readFileSync(
      path.join(root, 'custom/configs/2d/config-onMainToolbarCreated.js'),
      'utf8',
    )
    const items = []
    const mainToolbar = {
      setItemVisible() {},
      removeItemById() {},
      addItem(item) {
        items.push(item)
      },
    }
    const Shape = class {}
    const editor = {
      mainToolbar,
      displayView: {},
      createDisplayItem: (id, toolTip, icon, type, initData) => ({
        id,
        toolTip,
        icon,
        type,
        initData,
      }),
      createSymbolItem: (id) => ({ id }),
    }
    const context = {
      window: { hteditor_config: {} },
      hteditor: { getString: (name) => name },
      ht: { Edge: class {}, Shape },
    }
    runInNewContext(toolbarSource, context)
    context.window.hteditor_config.onMainToolbarCreated(editor)

    const pipeItem = items.find((item) => item.id === 'Pipe')
    const autoRouteItem = items.find((item) => item.id === 'AutoRoute')
    expect(autoRouteItem.icon).toBe('editor.changepath')
    expect(pipeItem.icon).toBe('custom/images/pipe.json')
    const styles = {}
    const attrs = {}
    pipeItem.initData({
      setDisplayName() {},
      a(name, value) {
        if (name && typeof name === 'object') Object.assign(attrs, name)
        else attrs[name] = value
      },
      s(values) {
        Object.assign(styles, values)
      },
    })

    expect(pipeItem.type).toBe(Shape)
    expect(attrs['induforge.path.type']).toBe('pipe')
    expect(styles).toMatchObject({
      'shape.border.width': 12,
      'shape.border.color': '#56615C',
      'shape.dash': true,
      'shape.dash.pattern': [100000, 0],
      'shape.dash.width': 6,
      'shape.dash.color': '#47CFA0',
      'shape.dash.flow': false,
      'shape.dash.flow.reverse': false,
      'shape.dash.flow.step': 2,
    })
    expect(attrs).toMatchObject({
      'induforge.pipe.flowMode': 'continuous',
      'induforge.pipe.running': true,
      'induforge.pipe.reverse': false,
      'induforge.pipe.speed': 2,
    })
  })

  it('路径选择和自动绕障使用 GraphView 的正式 API', () => {
    const root = path.resolve(process.cwd(), 'public/ht-editor')
    const editorCreatedSource = readFileSync(
      path.join(root, 'custom/configs/2d/config-onEditorCreated.js'),
      'utf8',
    )
    const toolbarSource = readFileSync(
      path.join(root, 'custom/configs/2d/config-onMainToolbarCreated.js'),
      'utf8',
    )

    expect(editorCreatedSource).toContain('graphView.dm && graphView.dm()')
    expect(editorCreatedSource).toContain("property === 'points' || property === 'segments'")
    expect(editorCreatedSource).toContain('refreshShapeBounds(graphView, event.data, true)')
    expect(editorCreatedSource).toContain('graphView.__induforgeShapeHitToleranceInstalled')
    expect(editorCreatedSource).toContain('candidate instanceof ht.Shape')
    expect(editorCreatedSource).toContain('editInteractor.pointsEditingMode = true')
    expect(editorCreatedSource).toContain('editInteractor.pointsEditingMode = false')
    expect(editorCreatedSource).toContain("event.kind === 'doubleClickData'")
    expect(editorCreatedSource).toContain("graphView.setEditStyle('anchorVisible', false)")
    expect(editorCreatedSource).toContain("graphView.setEditStyle('moveDummyThreshold', true)")
    expect(toolbarSource).toContain('graphView.sm && graphView.sm().ld()')
  })

  it('开放路径使用扩展命中范围并在点集变化后刷新边界', () => {
    const root = path.resolve(process.cwd(), 'public/ht-editor')
    const source = readFileSync(
      path.join(root, 'custom/configs/2d/config-onEditorCreated.js'),
      'utf8',
    )
    const Shape = class {
      rect = { x: 0, y: 0, width: 30, height: 30 }
      fireShapeChange = vi.fn()
      setRect = vi.fn((rect) => {
        this.rect = rect
      })
      getRect() {
        return this.rect
      }
      getPoints() {
        return { toArray: () => [{ x: 10, y: 20 }, { x: 80, y: 60 }] }
      }
    }
    const shape = new Shape()
    const hitTolerances = []
    let propertyListener
    let selectionListener
    let interactorListener
    let editorListener
    let modelListener
    const deferredCallbacks = []
    const selectionModel = {
      addSelectionChangeListener(listener) {
        selectionListener = listener
      },
      ld: () => shape,
    }
    const dataModel = {
      each(callback) {
        callback(shape)
      },
      addDataPropertyChangeListener(listener) {
        propertyListener = listener
      },
      addDataModelChangeListener(listener) {
        modelListener = listener
      },
      sm: () => selectionModel,
    }
    const graphView = {
      dm: vi.fn(() => dataModel),
      enableDashFlow: vi.fn(),
      enableFlow: vi.fn(),
      setEditStyle: vi.fn(),
      sm: () => selectionModel,
      addInteractorListener(listener) {
        interactorListener = listener
      },
      getEditInteractor: vi.fn(() => editInteractor),
      invalidateData: vi.fn(),
      invalidateSelection: vi.fn(),
      getDataAt(_point, filter, tolerance) {
        hitTolerances.push(tolerance)
        return tolerance === 8 && filter(shape) ? shape : undefined
      },
    }
    const originalAutoScroll = vi.fn(() => ({ x: 3, y: 4 }))
    const moveHandle = {
      startEdit: vi.fn(),
      _46O: vi.fn(),
    }
    const editInteractor = {
      pointsEditingMode: false,
      setCursor: vi.fn(),
      getSubModule: vi.fn((catalog) => (catalog === 'MoveDummy' ? moveHandle : undefined)),
    }
    graphView.autoScroll = originalAutoScroll
    const editor = {
      displayView: { graphView },
      addEventListener(listener) {
        editorListener = listener
      },
      displays: {},
      mainMenu: { setItems() {} },
      mainToolbar: { setItemVisible() {} },
    }
    const context = {
      window: {
        hteditor_config: {},
        setTimeout(callback, delay) {
          deferredCallbacks.push({ callback, delay })
          return deferredCallbacks.length
        },
      },
      ht: { Shape },
    }

    runInNewContext(source, context)
    context.window.hteditor_config.onEditorCreated(editor)

    expect(graphView.dm).toHaveBeenCalledOnce()
    expect(graphView.setEditStyle).toHaveBeenCalledWith('anchorVisible', false)
    expect(graphView.setEditStyle).toHaveBeenCalledWith('moveDummyThreshold', true)
    expect(graphView.setEditStyle).toHaveBeenCalledWith('moveDummySensitivity', 18)
    expect(graphView.setEditStyle).toHaveBeenCalledWith('moveDummyPosition', [0, 0, -20, -20])
    moveHandle.startEdit(editInteractor, {})
    expect(editInteractor.setCursor).toHaveBeenCalledWith('move')
    expect(graphView.autoScroll()).toEqual({ x: 0, y: 0 })
    expect(originalAutoScroll).not.toHaveBeenCalled()
    moveHandle._46O({})
    expect(graphView.autoScroll).toBe(originalAutoScroll)
    expect(graphView.autoScroll()).toEqual({ x: 3, y: 4 })
    expect(shape.fireShapeChange).toHaveBeenCalledOnce()
    expect(graphView.invalidateData).toHaveBeenCalledWith(shape)
    expect(deferredCallbacks).toHaveLength(2)
    deferredCallbacks.shift().callback()
    expect(shape.fireShapeChange).toHaveBeenCalledTimes(2)
    deferredCallbacks.shift().callback()
    expect(shape.fireShapeChange).toHaveBeenCalledTimes(3)
    expect(graphView.dm).toHaveBeenCalledTimes(2)
    expect(graphView.getDataAt({ x: 10, y: 10 })).toBe(shape)
    expect(hitTolerances).toEqual([undefined, 8])
    propertyListener({ data: shape, property: 'points' })
    expect(shape.fireShapeChange).toHaveBeenCalledTimes(4)
    expect(graphView.invalidateData).toHaveBeenCalledWith(shape)
    deferredCallbacks.shift().callback()
    expect(shape.fireShapeChange).toHaveBeenCalledTimes(5)
    modelListener({ data: shape, kind: 'add' })
    expect(shape.fireShapeChange).toHaveBeenCalledTimes(6)
    deferredCallbacks.shift().callback()
    expect(shape.fireShapeChange).toHaveBeenCalledTimes(7)
    selectionListener({ data: shape })
    expect(shape.fireShapeChange).toHaveBeenCalledTimes(8)
    expect(editInteractor.pointsEditingMode).toBe(false)
    expect(deferredCallbacks).toHaveLength(1)
    expect(deferredCallbacks.map((item) => item.delay)).toEqual([0])
    interactorListener({ kind: 'clickData', data: shape })
    expect(editInteractor.pointsEditingMode).toBe(false)
    interactorListener({ kind: 'doubleClickData', data: shape })
    expect(editInteractor.pointsEditingMode).toBe(true)
    expect(graphView.invalidateSelection).not.toHaveBeenCalled()
    interactorListener({ kind: 'clickData', data: shape })
    expect(editInteractor.pointsEditingMode).toBe(false)
    expect(graphView.invalidateSelection).toHaveBeenCalledOnce()
    deferredCallbacks.shift().callback()
    expect(shape.fireShapeChange).toHaveBeenCalledTimes(9)
    interactorListener({ kind: 'betweenEditPoint' })
    expect(shape.fireShapeChange).toHaveBeenCalledTimes(10)
    interactorListener({ kind: 'selectPoint' })
    expect(shape.fireShapeChange).toHaveBeenCalledTimes(11)
    deferredCallbacks.shift().callback()
    expect(shape.fireShapeChange).toHaveBeenCalledTimes(12)
    expect(graphView.getDataAt({ x: 10, y: 10 }, undefined, 3)).toBe(shape)
    expect(hitTolerances.slice(-2)).toEqual([3, 8])

    const reopenedGraphView = {
      dm: () => dataModel,
      getDataAt: () => undefined,
    }
    editorListener({
      type: 'displayViewOpened',
      params: { displayView: { graphView: reopenedGraphView } },
    })
    expect(reopenedGraphView.__induforgeShapeEditingInstalled).toBe(true)
  })

  it('作品视图延迟创建后仍会安装路径编辑能力且不会无限探测', () => {
    const root = path.resolve(process.cwd(), 'public/ht-editor')
    const source = readFileSync(
      path.join(root, 'custom/configs/2d/config-onEditorCreated.js'),
      'utf8',
    )
    const deferredCallbacks = []
    const Shape = class {}
    const graphView = {
      dm: () => ({
        each() {},
      }),
      getDataAt: () => undefined,
    }
    const editor = {
      displayView: undefined,
      addEventListener() {},
      displays: {},
      mainMenu: { setItems() {} },
      mainToolbar: { setItemVisible() {} },
    }
    const context = {
      window: {
        hteditor_config: {},
        setTimeout(callback, delay) {
          deferredCallbacks.push({ callback, delay })
          return deferredCallbacks.length
        },
      },
      ht: { Shape },
    }

    runInNewContext(source, context)
    context.window.hteditor_config.onEditorCreated(editor)

    expect(deferredCallbacks.map((item) => item.delay)).toEqual([0])
    deferredCallbacks.shift().callback()
    expect(deferredCallbacks.map((item) => item.delay)).toEqual([50])
    editor.displayView = { graphView }
    deferredCallbacks.shift().callback()
    expect(graphView.__induforgeShapeEditingInstalled).toBe(true)
    expect(deferredCallbacks).toHaveLength(0)
  })

  it('HT 抑制原生边界同步时仍按 Shape 点集更新外接矩形', () => {
    const root = path.resolve(process.cwd(), 'public/ht-editor')
    const source = readFileSync(
      path.join(root, 'custom/configs/2d/config-onEditorCreated.js'),
      'utf8',
    )
    const Shape = class {
      rect = { x: -15, y: -15, width: 30, height: 30 }
      points = [{ x: 100, y: 80 }, { x: 280, y: 170 }]
      fireShapeChange() {}
      getPoints() {
        return { toArray: () => this.points }
      }
      getRect() {
        return this.rect
      }
      setRect(rect) {
        this.rect = rect
      }
    }
    const context = {
      window: { hteditor_config: {}, setTimeout() {} },
      ht: { Shape },
    }
    const editor = {
      displays: {},
      mainMenu: { setItems() {} },
      mainToolbar: { setItemVisible() {} },
    }

    runInNewContext(source, context)
    context.window.hteditor_config.onEditorCreated(editor)
    const shape = new Shape()
    shape.fireShapeChange()

    expect(shape.getRect()).toEqual({ x: 100, y: 80, width: 180, height: 90 })
    expect(shape._55I).toBeUndefined()
  })

  it('HT 只移动 Shape 外框时同步平移路径点集', () => {
    const root = path.resolve(process.cwd(), 'public/ht-editor')
    const source = readFileSync(
      path.join(root, 'custom/configs/2d/config-onEditorCreated.js'),
      'utf8',
    )
    const Shape = class {
      position = { x: 50, y: 30 }
      points = [
        { x: 10, y: 20 },
        { x: 90, y: 60 },
      ]
      fireShapeChange() {}
      getPosition() {
        return this.position
      }
      setPosition(position) {
        this.position = position
      }
      getPoints() {
        return { toArray: () => this.points }
      }
      shiftPoints(offsetX, offsetY) {
        this.points = this.points.map((point) => ({
          x: point.x + offsetX,
          y: point.y + offsetY,
        }))
      }
    }
    const context = {
      window: { hteditor_config: {}, setTimeout() {} },
      ht: { Shape },
    }
    const editor = {
      displays: {},
      mainMenu: { setItems() {} },
      mainToolbar: { setItemVisible() {} },
    }

    runInNewContext(source, context)
    context.window.hteditor_config.onEditorCreated(editor)
    const shape = new Shape()
    shape.setPosition({ x: 80, y: 70 })

    expect(shape.getPoints().toArray()).toEqual([
      { x: 40, y: 60 },
      { x: 120, y: 100 },
    ])
  })

  it('隐藏平台管理的 2D 预览和快照地址并在保存前清理', () => {
    const root = path.resolve(process.cwd(), 'public/ht-editor')
    const filterSource = readFileSync(
      path.join(root, 'custom/configs/2d/config-inspectorFilter.js'),
      'utf8',
    )
    const handleEventSource = readFileSync(
      path.join(root, 'custom/configs/2d/config-handleEvent.js'),
      'utf8',
    )
    const filterContext = {
      window: { hteditor_config: {} },
      ht: { Shape: class {} },
    }
    runInNewContext(filterSource, filterContext)

    expect(filterContext.window.hteditor_config.detailFilter.isDisplayPropertyVisible({}, 'previewURL')).toBe(false)
    expect(filterContext.window.hteditor_config.detailFilter.isDisplayPropertyVisible({}, 'snapshotURL')).toBe(false)
    expect(filterContext.window.hteditor_config.compactFilter.isDisplayPropertyVisible({}, 'previewURL')).toBe(false)
    expect(filterContext.window.hteditor_config.compactFilter.isDisplayPropertyVisible({}, 'snapshotURL')).toBe(false)
    expect(
      filterContext.window.hteditor_config.detailFilter.isSymbolPropertyVisible(
        {},
        'previewURL.snapshotURL',
      ),
    ).toBe(false)

    const handleEventContext = {
      window: { hteditor_config: {} },
      hteditor: { getString: (name) => name },
    }
    runInNewContext(handleEventSource, handleEventContext)
    const attributes = { previewURL: 'external.html', snapshotURL: 'external.png' }
    const dataModel = {
      a(name, value) {
        attributes[name] = value
      },
    }
    handleEventContext.window.hteditor_config.handleEvent({}, 'displayViewSaving', {
      displayView: { dm: dataModel },
    })

    expect(attributes).toEqual({ previewURL: undefined, snapshotURL: undefined })
  })

  it('隐藏平台管理的 3D 预览地址并固定使用平台 Viewer', () => {
    const root = path.resolve(process.cwd(), 'public/ht-editor')
    const createdSource = readFileSync(
      path.join(root, 'custom/configs/3d/config-onEditor3dCreated.js'),
      'utf8',
    )
    const entrySource = readFileSync(path.join(root, 'index3d.html'), 'utf8')
    const inspector = {
      isPropertyVisible: vi.fn(() => true),
      filterProperties: vi.fn(),
    }
    const attributes = { previewURL: 'external.html', snapshotURL: 'external.png' }
    let editorListener
    const editor = {
      sceneInspector: inspector,
      dm: {
        a(name, value) {
          attributes[name] = value
        },
      },
      addEventListener(listener) {
        editorListener = listener
      },
    }
    const context = { window: { hteditor_config: {} } }
    runInNewContext(createdSource, context)
    context.window.hteditor_config.onEditor3dCreated(editor)

    expect(inspector.isPropertyVisible({ keys: { name: 'previewURL' } })).toBe(false)
    expect(inspector.isPropertyVisible({ keys: { name: 'camera' } })).toBe(true)
    expect(inspector.filterProperties).toHaveBeenCalledOnce()
    editorListener({ type: 'sceneSaving' })
    expect(attributes).toEqual({ previewURL: undefined, snapshotURL: undefined })
    expect(entrySource).toContain('var previewURL = "scene.html"')
    expect(entrySource).not.toContain('this.dm.a("previewURL")')
  })

  it('对外只解析白标资源名并拒绝原始引擎路径', () => {
    expect(resolveSceneStudioAsset('runtime/core.js')).toBe('libs/core/ht.js')
    expect(resolveSceneStudioAsset('runtime/extensions/live-controls.js')).toBe(
      'libs/plugin/ht-live.js',
    )
    expect(resolveSceneStudioAsset('runtime/extensions/dash-flow.js')).toBe(
      'libs/plugin/ht-dashflow.js',
    )
    expect(resolveSceneStudioAsset('config/engine.js')).toBe('custom/configs/engine.js')
    expect(resolveSceneStudioAsset('libs/core/ht.js')).toBeNull()
    expect(resolveSceneStudioAsset('../ht-editor/index.html')).toBeNull()
  })

  it('白标核心库保留同名局部变量语义', () => {
    const transformed = rewriteSceneStudioIdentifiers(
      `!function(root){var ht={List:function(){}};var key='ht';root.ht=ht;root[key]=ht;root.def('ht.List')}(window);`,
      'runtime/core.js',
    )

    expect(transformed).toContain('IF')
    expect(transformed).toContain('IF.List')
    expect(transformed).not.toMatch(/["']ht["']/)
    expect(transformed).not.toContain('ht.List')
    expect(transformed).not.toContain('.ht')
  })

  it('白标核心库将两字符全局探测同步到 IF', () => {
    const source = readFileSync(
      path.resolve(process.cwd(), 'public/ht-editor/libs/core/ht.js'),
      'utf8',
    )
    const transformed = rewriteSceneStudioIdentifiers(source, 'runtime/core.js')

    const namespaceProbe = parseInt('IF', 32)
    expect(namespaceProbe).toBe(591)
    expect(transformed).toMatch(
      new RegExp(`\\b${namespaceProbe}\\s*===\\s*[\\w$]+\\([\\w$]+,\\s*32\\)`),
    )
    expect(transformed).not.toMatch(/\b573\s*===\s*[\w$]+\([\w$]+,\s*32\)/)
  })

  it('白标运行模块使用新全局命名空间', () => {
    const transformed = rewriteSceneStudioIdentifiers(
      `!function(root){var namespace='ht';var engine=root[namespace];engine.ui={}}(window);`,
      'runtime/ui.js',
    )

    expect(transformed).toContain('IF')
    expect(transformed).not.toMatch(/["']ht["']/)
  })

  it('白标运行模块移除可识别的引擎内部名称', () => {
    const transformed = rewriteSceneStudioIdentifiers(
      String.raw`window.htComp = value; window.htExpired = text; studio.HTNodeInspector = Type;
       this.htNodeInspector = this._htNodeInspector; var inspectorKey = 'htNodeInspector';
       studio.HTView = View; var marker = '__ht__function'; var easterEgg = '$$HTISFUCKINGAWESOME$$';
       var roots = ['/ht', '/htdesign']; var binary = '.htb'; var binaryPattern = /\.htb$/i;
       console.log('HT Editor v5.0.0 powered by HT for Web v' + ht.Default.getVersion());`,
      'runtime/studio-2d.js',
    )

    expect(transformed).toContain('ifComp')
    expect(transformed).toContain('engineExpired')
    expect(transformed).toContain('SceneNodeInspector')
    expect(transformed).toContain('sceneNodeInspector')
    expect(transformed).toContain('_sceneNodeInspector')
    expect(transformed).toContain('SceneView')
    expect(transformed).toContain('__if__function')
    expect(transformed).toContain('INDUFORGE_SCENE_ENGINE$$')
    expect(transformed).toContain('/scene-design')
    expect(transformed).toContain('.ifb')
    expect(transformed).toContain('/\\.ifb$/i')
    expect(transformed).toContain('1.0.0')
    expect(transformed).not.toMatch(
      /__ht__|HTISFUCKINGAWESOME|htNodeInspector|HTNodeInspector|HTView|htdesign|\.htb|5\.0\.0/,
    )
  })

  it('拒绝恢复 HT 原生多图纸管理', () => {
    const root = mkdtempSync(path.join(tmpdir(), 'induforge-ht-single-entry-'))
    const configs = path.join(root, 'custom/configs')
    mkdirSync(path.join(configs, '2d'), { recursive: true })
    mkdirSync(path.join(configs, '3d'), { recursive: true })
    writeFileSync(path.join(root, 'index.html'), 'createEditor({ open: entryPath })')
    writeFileSync(path.join(root, 'index3d.html'), 'createEditor3d({ open: entryPath })')
    writeFileSync(path.join(configs, '2d/config.js'), 'displaysEditable: true')
    writeFileSync(path.join(configs, '2d/config-onEditorCreated.js'), '')
    writeFileSync(path.join(configs, '3d/config.js'), '')
    writeFileSync(path.join(configs, '3d/config-onEditor3dCreated.js'), '')

    expect(() => enforceHTSingleEntryPolicy(root)).toThrow(/2D 图纸树只读/)
  })
})
