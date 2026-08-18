import { mkdtempSync, mkdirSync, readFileSync, writeFileSync } from 'node:fs'
import { tmpdir } from 'node:os'
import path from 'node:path'
import { describe, expect, it } from 'vitest'
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
  })

  it('对外只解析白标资源名并拒绝原始引擎路径', () => {
    expect(resolveSceneStudioAsset('runtime/core.js')).toBe('libs/core/ht.js')
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
