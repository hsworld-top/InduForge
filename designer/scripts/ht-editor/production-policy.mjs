import {
  copyFileSync,
  existsSync,
  mkdirSync,
  readFileSync,
  readdirSync,
  rmSync,
  statSync,
  writeFileSync,
} from 'node:fs'
import { createRequire } from 'node:module'
import path from 'node:path'
import ts from 'typescript'

const require = createRequire(import.meta.url)
let esbuildTransformSync

const sourceDirectoryName = 'ht-editor'
const publicDirectoryName = 'scene-studio'

const runtimeAliases = new Map([
  ['config/engine.js', 'custom/configs/engine.js'],
  ['runtime/core.js', 'libs/core/ht.js'],
  ['runtime/ui.js', 'libs/core/ht-ui.js'],
  ['runtime/studio-2d.js', 'libs/client.js'],
  ['runtime/studio-3d.js', 'libs/client3d.js'],
  ['runtime/extensions/animation.js', 'libs/plugin/ht-animation.js'],
  ['runtime/extensions/auto-layout.js', 'libs/plugin/ht-autolayout.js'],
  ['runtime/extensions/context-menu.js', 'libs/plugin/ht-contextmenu.js'],
  ['runtime/extensions/css-animation.js', 'libs/plugin/ht-cssanimation.js'],
  ['runtime/extensions/dialog.js', 'libs/plugin/ht-dialog.js'],
  ['runtime/extensions/edge-type.js', 'libs/plugin/ht-edgetype.js'],
  ['runtime/extensions/fbx-loader.js', 'libs/plugin/ht-fbx.js'],
  ['runtime/extensions/form.js', 'libs/plugin/ht-form.js'],
  ['runtime/extensions/gltf-loader.js', 'libs/plugin/ht-gltf.js'],
  ['runtime/extensions/history.js', 'libs/plugin/ht-historymanager.js'],
  ['runtime/extensions/modeling.js', 'libs/plugin/ht-modeling.js'],
  ['runtime/extensions/object-model.js', 'libs/plugin/ht-obj.js'],
  ['runtime/extensions/overview.js', 'libs/plugin/ht-overview.js'],
  ['runtime/extensions/texture-loader.js', 'libs/plugin/ht-textureLoader.js'],
  ['runtime/extensions/vector.js', 'libs/plugin/ht-vector.js'],
])

const directFiles = new Set([
  'index.html',
  'index3d.html',
  'display.html',
  'scene.html',
  'symbol.html',
  'custom/configs/buckle.js',
  'custom/configs/config-customProperties.js',
  'custom/configs/config-dataBindings.js',
  'custom/configs/config-service.js',
  'custom/configs/config-transform.js',
  'custom/configs/config-valueTypes.js',
  'custom/configs/2d/config.js',
  'custom/configs/2d/config-connectActions.js',
  'custom/configs/2d/config-dataBindingsForSymbol.js',
  'custom/configs/2d/config-handleEvent.js',
  'custom/configs/2d/config-inspectorFilter.js',
  'custom/configs/2d/config-inspectorTab.js',
  'custom/configs/2d/config-onEditorCreated.js',
  'custom/configs/2d/config-onMainToolbarCreated.js',
  'custom/configs/2d/config-onRightToolbarCreated.js',
  'custom/configs/2d/config-utils.js',
  'custom/configs/3d/config.js',
  'custom/configs/3d/config-onEditor3dCreated.js',
  'custom/images/clock.json',
  'custom/images/fullscreen.json',
  'custom/images/hprogressbar.json',
  'custom/images/induforge.svg',
  'custom/images/pipe.json',
  'custom/images/table.json',
  'custom/images/vprogressbar.json',
  'custom/libs/InduForgeService.js',
  'custom/libs/echarts.js',
  'custom/locales/en.js',
  'custom/locales/zh.js',
  'locales/en.js',
  'locales/zh.js',
])

const copiedDirectories = ['vs']
const browserIdentifierAliases = new Map([
  ['ht', 'InduSceneEngine'],
  ['htconfig', 'InduSceneEngineConfig'],
  ['hteditor', 'InduSceneStudio'],
  ['hteditor3d', 'InduSceneStudio3D'],
  ['hteditor_config', 'InduSceneStudioConfig'],
  ['__ht__', '__if__'],
  ['__ht__list', '__if__list'],
  ['htComp', 'ifComp'],
  ['htExpired', 'engineExpired'],
  ['htWillExpire', 'engineWillExpire'],
  ['HTView', 'SceneView'],
  ['HTDataInspector', 'SceneDataInspector'],
  ['HTNodeInspector', 'SceneNodeInspector'],
  ['HTTextInspector', 'SceneTextInspector'],
  ['HTShapeInspector', 'SceneShapeInspector'],
  ['HTGroupInspector', 'SceneGroupInspector'],
  ['HTBlockInspector', 'SceneBlockInspector'],
  ['HTRefGraphInspector', 'SceneRefGraphInspector'],
  ['HTEdgeInspector', 'SceneEdgeInspector'],
  ['htDataInspector', 'sceneDataInspector'],
  ['htNodeInspector', 'sceneNodeInspector'],
  ['htTextInspector', 'sceneTextInspector'],
  ['htShapeInspector', 'sceneShapeInspector'],
  ['htGroupInspector', 'sceneGroupInspector'],
  ['htBlockInspector', 'sceneBlockInspector'],
  ['htRefGraphInspector', 'sceneRefGraphInspector'],
  ['htEdgeInspector', 'sceneEdgeInspector'],
  ['_htDataInspector', '_sceneDataInspector'],
  ['_htNodeInspector', '_sceneNodeInspector'],
  ['_htTextInspector', '_sceneTextInspector'],
  ['_htShapeInspector', '_sceneShapeInspector'],
  ['_htGroupInspector', '_sceneGroupInspector'],
  ['_htBlockInspector', '_sceneBlockInspector'],
  ['_htRefGraphInspector', '_sceneRefGraphInspector'],
  ['_htEdgeInspector', '_sceneEdgeInspector'],
  ['locate', 'unsupportedFileAction'],
])

const forbiddenVisiblePatterns = [
  { name: '厂商站点链接', pattern: /hightopo\.com/i },
  { name: '厂商可见品牌', pattern: /HT for Web|HIGHTOPO is AWESOME/i },
  { name: '厂商编辑器横幅', pattern: /\bHT (?:3D )?Editor\b/i },
  { name: '旧编辑器命名空间', pattern: /\bhteditor3?d?\b/i },
  { name: '旧引擎全局变量', pattern: /\bwindow\s*\.\s*ht\b/i },
  { name: '旧引擎字符串命名空间', pattern: /["']ht(?:\.|["'])/ },
  { name: '旧引擎 DOM 前缀', pattern: /["'`.]ht[-_](?=[a-z])/i },
  { name: '旧引擎私有序列化前缀', pattern: /__ht__/i },
  { name: '旧引擎授权变量', pattern: /\bht(?:Expired|WillExpire)\b/i },
  { name: '旧引擎二进制扩展名', pattern: /\.htb\b/i },
  {
    name: '旧引擎检查器类型',
    pattern:
      /\bHT(?:Data|Node|Text|Shape|Group|Block|RefGraph|Edge)Inspector\b|\bHTView\b/i,
  },
  { name: '旧引擎隐藏标记', pattern: /HTISFUCKINGAWESOME/i },
  { name: '旧引擎设计目录', pattern: /\bhtdesign\b/i },
  { name: '上游运行版本指纹', pattern: /(?:5\.0\.0|5\.0\.2-dev1|7\.7\.2-dev3)/ },
  { name: '旧公开资源路径', pattern: /\bht-editor\b/i },
  { name: '原编辑器产品线标识', pattern: /\bisSaveInKP\b|\bK[FP]\d+(?:\.\d+)?\b/ },
  { name: '高炉演示', pattern: /炼铁高炉|display-pudding|gaolu-modeling/i },
  { name: 'locate 文件命令', pattern: /case\s+['"]locate['"]|request\s*\(\s*['"]locate['"]/i },
  { name: '旧字符串消息协议', pattern: /postMessage\s*\(\s*['"]/i },
  { name: '原生作品管理标签', pattern: /setName\s*\(\s*['"]当前场景['"]\s*\)/i },
]

const singleEntryRules = [
  {
    file: 'index.html',
    checks: [
      ['2D 使用白标运行资源', /runtime\/core\.js/],
      ['2D 受控入口传入编辑器', /createEditor\(\{\s*open:\s*entryPath/],
      ['2D 预览传入固定入口', /open:\s*entryPath/],
    ],
  },
  {
    file: 'index3d.html',
    checks: [
      ['3D 使用白标运行资源', /runtime\/core\.js/],
      ['3D 受控入口传入编辑器', /createEditor3d\(\{\s*open:\s*entryPath/],
      ['3D 预览传入固定入口', /open:\s*entryPath/],
    ],
  },
  {
    file: 'custom/configs/2d/config.js',
    checks: [
      ['2D 图纸树只读', /displaysEditable\s*:\s*false/],
      ['2D Symbol 树只读', /symbolsEditable\s*:\s*false/],
      ['2D Component 树只读', /componentsEditable\s*:\s*false/],
      ['2D 禁止缺失入口时新建', /newIfFailToOpen\s*:\s*false/],
    ],
  },
  {
    file: 'custom/configs/2d/config-onEditorCreated.js',
    checks: [
      ['2D 隐藏原生图纸页', /displaysTab\.setVisible\(false\)/],
      ['2D 隐藏原生作品菜单', /mainMenu\.setItems\(\[\]\)/],
      ['2D 清空图纸文件菜单', /view\.menu\.setItems\(\[\]\)/],
    ],
  },
  {
    file: 'custom/configs/3d/config.js',
    checks: [['3D 禁止缺失入口时新建', /newIfFailToOpen\s*:\s*false/]],
  },
  {
    file: 'custom/configs/3d/config-onEditor3dCreated.js',
    checks: [
      ['3D 隐藏原生作品菜单', /mainMenu\.setItems\(\[\]\)/],
      ['3D 隐藏原生作品和资源页', /tab\.setVisible\(false\)/],
      ['3D 清空文件菜单', /view\.menu\.setItems\(\[\]\)/],
    ],
  },
]

export function resolveSceneStudioAsset(relativePath) {
  const decoded = decodeURIComponent(String(relativePath || '')).replace(/\\/g, '/')
  const normalized = path.posix.normalize('/' + decoded).slice(1)
  if (!normalized || normalized.startsWith('../') || normalized.includes('\0')) return null
  if (runtimeAliases.has(normalized)) return runtimeAliases.get(normalized)
  if (directFiles.has(normalized)) return normalized
  if (
    copiedDirectories.some(
      (directory) => normalized === directory || normalized.startsWith(directory + '/'),
    )
  )
    return normalized
  return null
}

function sanitizeEngineText(source) {
  return source
    .replace(/(?:5\.0\.0|5\.0\.2-dev1|7\.7\.2-dev3)/g, '1.0.0')
    .replace(/\bht\.(?=[A-Za-z_$]|$)/g, 'IF.')
    .replace(
      /\bht(Data|Node|Text|Shape|Group|Block|RefGraph|Edge)Inspector\b/g,
      'scene$1Inspector',
    )
    .replace(/\bht(?=[A-Z])/g, 'if')
    .replace(/\bht-(?=[a-z])/gi, 'if-')
    .replace(/\bht_(?=[a-z])/gi, 'if_')
    .replace(/\bhtComp\b/g, 'ifComp')
    .replace(/__ht__/gi, '__if__')
    .replace(/\bhtExpired\b/g, 'engineExpired')
    .replace(/\bhtWillExpire\b/g, 'engineWillExpire')
    .replace(/\bHTView\b/g, 'SceneView')
    .replace(
      /\bHT(Data|Node|Text|Shape|Group|Block|RefGraph|Edge)Inspector\b/g,
      'Scene$1Inspector',
    )
    .replace(/HTISFUCKINGAWESOME\$\$/g, () => 'INDUFORGE_SCENE_ENGINE$$')
    .replace(/(^|\/)htdesign(?=\/|$)/gi, '$1scene-design')
    .replace(/(^|\/)ht(?=\/|$)/gi, '$1scene')
    .replace(/\bhtb\b/gi, 'ifb')
    .replace(/\/(?:ht)(?=\.(?:json|png|svg)\b)/gi, '/induforge')
}

function sanitizeBrandText(source, sanitizeEngineTokens = true) {
  const branded = source
    .replace(/https?:\/\/(?:www\.)?hightopo\.com[^'"\s)]*/gi, 'about:blank')
    .replace(/(?:www\.)?hightopo\.com/gi, 'InduForge')
    .replace(/HT for Web/gi, 'InduForge Scene Engine')
    .replace(/HT 3D Editor/gi, 'InduForge 3D Studio')
    .replace(/HT Editor/gi, 'InduForge Scene Studio')
    .replace(/HIGHTOPO/gi, 'InduForge')
    .replace(/hteditor3d/g, 'InduSceneStudio3D')
    .replace(/hteditor_config/g, 'InduSceneStudioConfig')
    .replace(/hteditor/g, 'InduSceneStudio')
    .replace(/ht-editor/gi, 'scene-studio')
    .replace(/htoverview/g, 'scene-overview')
    .replace(/locateFile/g, 'unsupportedFileAction')
    .replace(/(['"])locate\1/g, '$1unsupportedFileAction$1')
  return sanitizeEngineTokens ? sanitizeEngineText(branded) : branded
}

function rewriteEngineNamespaceProbe(source) {
  // 核心库用 parseInt(namespace, 32) 的固定值寻找两字符全局对象，白标后需同步指向 IF。
  return source.replace(
    /\b2\s*===\s*([A-Za-z_$][\w$]*)\.length\s*&&\s*573\s*===\s*([A-Za-z_$][\w$]*)\(\1\s*,\s*32\s*\)/g,
    '2 === $1.length && 591 === $2($1, 32)',
  )
}

export function rewriteSceneStudioIdentifiers(source, filename = 'scene-studio.js') {
  const isEngineCore = filename === 'runtime/core.js'
  const isRuntimeModule = filename.startsWith('runtime/')
  const preparedSource = isEngineCore ? rewriteEngineNamespaceProbe(source) : source
  const sourceFile = ts.createSourceFile(
    filename,
    preparedSource,
    ts.ScriptTarget.Latest,
    true,
    ts.ScriptKind.JS,
  )
  const transformed = ts.transform(sourceFile, [
    (context) => {
      const visit = (node) => {
        if (ts.isStringLiteralLike(node)) {
          if (isRuntimeModule && node.text === 'ht') {
            return context.factory.createStringLiteral('IF')
          }
          const rewrittenText = sanitizeEngineText(node.text)
          if (rewrittenText !== node.text) {
            return context.factory.createStringLiteral(rewrittenText)
          }
        }
        if (ts.isIdentifier(node) && browserIdentifierAliases.has(node.text)) {
          // 核心库内部可能存在同名局部变量，只改写它挂到 window 上的属性名。
          if (
            isEngineCore &&
            node.text === 'ht' &&
            !(ts.isPropertyAccessExpression(node.parent) && node.parent.name === node)
          ) {
            return node
          }
          if (isEngineCore && node.text === 'ht') {
            return context.factory.createIdentifier('IF')
          }
          return context.factory.createIdentifier(browserIdentifierAliases.get(node.text))
        }
        return ts.visitEachChild(node, visit, context)
      }
      return (root) => ts.visitNode(root, visit)
    },
  ])
  try {
    return ts
      .createPrinter({ removeComments: true })
      .printFile(transformed.transformed[0])
      .replace(/\\\.htb\b/gi, '\\.ifb')
  } finally {
    transformed.dispose()
  }
}

export function sanitizeSceneStudioJavaScript(source, filename = 'scene-studio.js') {
  const rewritten = rewriteSceneStudioIdentifiers(source, filename)
  esbuildTransformSync ||= require('esbuild').transformSync
  const transformed = esbuildTransformSync(sanitizeBrandText(rewritten, false), {
    loader: 'js',
    minify: true,
    legalComments: 'none',
    target: 'es2018',
  }).code
  return filename === 'runtime/core.js'
    ? `${transformed};window.InduSceneEngine=window.IF;\n`
    : transformed
}

function shouldTransformJavaScript(relativePath) {
  return (
    runtimeAliases.has(relativePath) ||
    relativePath.startsWith('custom/configs/') ||
    relativePath.startsWith('custom/locales/') ||
    relativePath.startsWith('locales/') ||
    relativePath === 'custom/libs/InduForgeService.js'
  )
}

function readPublicAsset(sourceRoot, publicRelativePath) {
  const sourceRelativePath = resolveSceneStudioAsset(publicRelativePath)
  if (!sourceRelativePath) return null
  const filename = path.join(sourceRoot, sourceRelativePath)
  if (!existsSync(filename) || !statSync(filename).isFile()) return null
  if (publicRelativePath.endsWith('.js') && shouldTransformJavaScript(publicRelativePath)) {
    return Buffer.from(
      sanitizeSceneStudioJavaScript(readFileSync(filename, 'utf8'), publicRelativePath),
    )
  }
  if (/\.(?:html|json|css|svg)$/.test(publicRelativePath)) {
    return Buffer.from(sanitizeBrandText(readFileSync(filename, 'utf8')))
  }
  return readFileSync(filename)
}

function contentType(filename) {
  const extension = path.extname(filename).toLowerCase()
  return (
    {
      '.css': 'text/css; charset=utf-8',
      '.html': 'text/html; charset=utf-8',
      '.js': 'text/javascript; charset=utf-8',
      '.json': 'application/json; charset=utf-8',
      '.svg': 'image/svg+xml',
      '.ttf': 'font/ttf',
      '.woff': 'font/woff',
      '.woff2': 'font/woff2',
    }[extension] || 'application/octet-stream'
  )
}

function installDevelopmentMiddleware(server, sourceRoot) {
  const cache = new Map()
  server.middlewares.use((request, response, next) => {
    const pathname = new URL(request.url || '/', 'http://localhost').pathname
    const oldMarker = `/${sourceDirectoryName}/`
    if (pathname.includes(oldMarker)) {
      response.statusCode = 404
      response.end()
      return
    }
    const marker = `/${publicDirectoryName}/`
    const markerIndex = pathname.indexOf(marker)
    if (markerIndex < 0) {
      next()
      return
    }
    const relativePath = pathname.slice(markerIndex + marker.length)
    if (!resolveSceneStudioAsset(relativePath)) {
      response.statusCode = 404
      response.end()
      return
    }
    try {
      let body = cache.get(relativePath)
      if (!body) {
        body = readPublicAsset(sourceRoot, relativePath)
        if (!body) throw new Error('资源不存在')
        cache.set(relativePath, body)
      }
      response.statusCode = 200
      response.setHeader('Content-Type', contentType(relativePath))
      response.setHeader('Cache-Control', 'no-store')
      response.end(body)
    } catch (_error) {
      response.statusCode = 404
      response.end()
    }
  })
}

export function pruneHTProduction(root) {
  if (!existsSync(root)) return
  for (const entry of readdirSync(root, { withFileTypes: true })) {
    if (entry.isFile() && entry.name.endsWith('.html') && !directFiles.has(entry.name)) {
      rmSync(path.join(root, entry.name))
    }
  }
  for (const relative of [
    'custom/previews',
    'custom/configs-customstyle',
    'custom/configs/configs-old',
  ]) {
    rmSync(path.join(root, relative), { recursive: true, force: true })
  }
}

function copyPublicFile(sourceRoot, targetRoot, publicRelativePath) {
  const body = readPublicAsset(sourceRoot, publicRelativePath)
  if (!body) throw new Error(`场景工作室资源不存在: ${publicRelativePath}`)
  const target = path.join(targetRoot, publicRelativePath)
  mkdirSync(path.dirname(target), { recursive: true })
  writeFileSync(target, body)
}

function copyDirectory(sourceRoot, targetRoot, relativePath) {
  const source = path.join(sourceRoot, relativePath)
  for (const filename of listFiles(source)) {
    const child = path.relative(sourceRoot, filename)
    const target = path.join(targetRoot, child)
    mkdirSync(path.dirname(target), { recursive: true })
    copyFileSync(filename, target)
  }
}

export function buildSceneStudioProduction(distRoot) {
  const sourceRoot = path.join(distRoot, sourceDirectoryName)
  const targetRoot = path.join(distRoot, publicDirectoryName)
  if (!existsSync(sourceRoot)) throw new Error('缺少场景工作室源资源')
  enforceHTSingleEntryPolicy(sourceRoot)
  rmSync(targetRoot, { recursive: true, force: true })
  mkdirSync(targetRoot, { recursive: true })
  for (const relativePath of [...directFiles, ...runtimeAliases.keys()]) {
    copyPublicFile(sourceRoot, targetRoot, relativePath)
  }
  for (const relativePath of copiedDirectories) copyDirectory(sourceRoot, targetRoot, relativePath)
  rmSync(sourceRoot, { recursive: true, force: true })
  scanHTProduction(targetRoot)
  return targetRoot
}

export function scanHTProduction(root) {
  const violations = []
  if (!existsSync(root)) throw new Error('HT 生产文件扫描失败: 缺少生产目录')
  for (const filename of listTextFiles(root)) {
    const relative = path.relative(root, filename).replace(/\\/g, '/')
    if (/(?:^|\/)ht(?:[-.]|$)|hteditor/i.test(relative)) {
      violations.push(`${relative}: 旧厂商资源文件名`)
    }
    const content = readFileSync(filename, 'utf8')
    for (const rule of forbiddenVisiblePatterns) {
      if (rule.pattern.test(content)) violations.push(`${relative}: ${rule.name}`)
    }
  }
  if (violations.length > 0) {
    throw new Error(`HT 生产文件扫描失败:\n${violations.join('\n')}`)
  }
}

export function enforceHTSingleEntryPolicy(root) {
  const violations = []
  for (const rule of singleEntryRules) {
    const filename = path.join(root, rule.file)
    if (!existsSync(filename)) {
      violations.push(`${rule.file}: 缺少单入口配置`)
      continue
    }
    const content = readFileSync(filename, 'utf8')
    for (const [name, pattern] of rule.checks) {
      if (!pattern.test(content)) violations.push(`${rule.file}: ${name}`)
    }
  }
  if (violations.length > 0) {
    throw new Error(`HT 单入口策略扫描失败:\n${violations.join('\n')}`)
  }
}

export function htProductionPolicy(rootDir) {
  const sourceRoot = path.join(rootDir, 'public', sourceDirectoryName)
  return {
    name: 'induforge-scene-studio-policy',
    configureServer(server) {
      installDevelopmentMiddleware(server, sourceRoot)
    },
    closeBundle() {
      buildSceneStudioProduction(path.join(rootDir, 'dist'))
    },
  }
}

function listFiles(target) {
  if (statSync(target).isFile()) return [target]
  const result = []
  for (const entry of readdirSync(target, { withFileTypes: true })) {
    const filename = path.join(target, entry.name)
    if (entry.isDirectory()) result.push(...listFiles(filename))
    else result.push(filename)
  }
  return result
}

function listTextFiles(target) {
  return listFiles(target).filter((filename) => /\.(?:css|html|js|json|svg)$/.test(filename))
}
