import { execFile } from 'node:child_process'
import {
  cp,
  chmod,
  mkdir,
  readFile,
  readdir,
  rename,
  rmdir,
  rm,
  writeFile,
} from 'node:fs/promises'
import path from 'node:path'
import { promisify } from 'node:util'

const execFileAsync = promisify(execFile)

export class WorkspaceInitializationError extends Error {
  constructor(message, statusCode = 400) {
    super(message)
    this.name = 'WorkspaceInitializationError'
    this.statusCode = statusCode
  }
}

function now() {
  return new Date().toISOString()
}

function workspaceState(status, templateId = null, initializedAt = null, message = null) {
  return { status, templateId, initializedAt, message }
}

function validateCatalog(value) {
  if (!value || typeof value !== 'object' || value.version !== 1 || !Array.isArray(value.templates)) {
    throw new Error('工程模板目录不符合契约')
  }
  const ids = new Set()
  const templates = value.templates.map((item) => {
    if (!item || typeof item !== 'object') throw new Error('工程模板项不符合契约')
    const fields = ['id', 'name', 'description', 'framework', 'language', 'directory']
    for (const field of fields) {
      if (typeof item[field] !== 'string' || !item[field].trim()) {
        throw new Error(`工程模板字段 ${field} 不符合契约`)
      }
    }
    if (ids.has(item.id)) throw new Error(`工程模板 ID 重复: ${item.id}`)
    if (item.directory !== path.basename(item.directory)) {
      throw new Error(`工程模板目录非法: ${item.directory}`)
    }
    ids.add(item.id)
    return {
      id: item.id,
      name: item.name,
      description: item.description,
      framework: item.framework,
      language: item.language,
      directory: item.directory,
    }
  })
  return { version: 1, generator: String(value.generator || ''), templates }
}

async function defaultRunCommand(command, args, options = {}) {
  return execFileAsync(command, args, {
    cwd: options.cwd,
    env: options.env,
    encoding: 'utf8',
    maxBuffer: 8 * 1024 * 1024,
  })
}

function commandErrorMessage(command, args, error) {
  const stderr = typeof error?.stderr === 'string' ? error.stderr.trim() : ''
  const stdout = typeof error?.stdout === 'string' ? error.stdout.trim() : ''
  const fallback = error instanceof Error ? error.message.trim() : ''
  const detail = stderr || stdout || fallback || '未知错误'
  return `命令执行失败: ${command} ${args.join(' ')}\n${detail}`
}

async function makeWorkspaceWritable(root) {
  for (const entry of await readdir(root, { withFileTypes: true })) {
    const target = path.join(root, entry.name)
    if (entry.isDirectory()) {
      await chmod(target, 0o755)
      await makeWorkspaceWritable(target)
    } else {
      await chmod(target, 0o644)
    }
  }
  await chmod(root, 0o755)
}

export function createWorkspaceInitializer(options = {}) {
  const workspaceRoot = path.resolve(options.workspaceRoot || '/workspace')
  const templatesRoot = path.resolve(options.templatesRoot || '/opt/induforge/templates')
  const storeDir = path.resolve(options.storeDir || '/cache/pnpm-store')
  const commandRunner = options.runCommand || defaultRunCommand
  let initialization = null
  let dependencyInstallation = null
  let lastError = null

  async function runCommand(command, args, commandOptions = {}) {
    try {
      return await commandRunner(command, args, commandOptions)
    } catch (error) {
      throw new Error(commandErrorMessage(command, args, error), { cause: error })
    }
  }

  const metadataRoot = path.join(workspaceRoot, '.workspace')
  const legacyMetadataRoot = path.join(workspaceRoot, '.induforge')
  const markerPath = path.join(metadataRoot, 'project.json')
  const catalogPath = path.join(templatesRoot, 'catalog.json')

  async function migrateLegacyMetadata() {
    let legacyEntries
    try {
      legacyEntries = await readdir(legacyMetadataRoot, { withFileTypes: true })
    } catch (error) {
      if (error?.code === 'ENOENT') return
      throw error
    }
    let currentEntries
    try {
      currentEntries = await readdir(metadataRoot, { withFileTypes: true })
    } catch (error) {
      if (error?.code === 'ENOENT') {
        await rename(legacyMetadataRoot, metadataRoot)
        return
      }
      throw error
    }
    const currentNames = new Set(currentEntries.map((entry) => entry.name))
    for (const entry of legacyEntries) {
      if (currentNames.has(entry.name)) {
        if (entry.name === 'context' && entry.isDirectory()) {
          await rm(path.join(legacyMetadataRoot, entry.name), { recursive: true, force: true })
        }
        continue
      }
      await rename(path.join(legacyMetadataRoot, entry.name), path.join(metadataRoot, entry.name))
    }
    try {
      await rmdir(legacyMetadataRoot)
    } catch (error) {
      if (error?.code !== 'ENOTEMPTY') throw error
    }
  }

  async function readCatalog() {
    return validateCatalog(JSON.parse(await readFile(catalogPath, 'utf8')))
  }

  async function readMarker() {
    try {
      const marker = JSON.parse(await readFile(markerPath, 'utf8'))
      if (
        marker?.version !== 1 ||
        typeof marker.templateId !== 'string' ||
        typeof marker.initializedAt !== 'string'
      ) {
        throw new Error('工程初始化标记不符合契约')
      }
      return marker
    } catch (error) {
      if (error && typeof error === 'object' && 'code' in error && error.code === 'ENOENT') return null
      throw error
    }
  }

  async function status() {
    if (initialization) return workspaceState('initializing')
    await migrateLegacyMetadata()
    const marker = await readMarker()
    if (marker) {
      return workspaceState('initialized', marker.templateId, marker.initializedAt)
    }
    const entries = await readdir(workspaceRoot, { withFileTypes: true })
    if (entries.length === 0) {
      const error = lastError
      lastError = null
      return workspaceState('uninitialized', null, null, error)
    }
    // Kubernetes 会把平台上下文单独挂载到工作区的 .workspace/context。
    // 它不是用户工程文件；仅这一固定目录存在时仍允许首次模板初始化。
    if (
      entries.length === 1 &&
      entries[0].name === '.workspace' &&
      entries[0].isDirectory()
    ) {
      const metadataEntries = await readdir(path.join(workspaceRoot, '.workspace'), {
        withFileTypes: true,
      })
      if (
        metadataEntries.length === 1 &&
        metadataEntries[0].name === 'context' &&
        metadataEntries[0].isDirectory()
      ) {
        const error = lastError
        lastError = null
        return workspaceState('uninitialized', null, null, error)
      }
    }
    return workspaceState(
      'error',
      null,
      null,
      '工作区存在未识别文件，必须使用新的空工作区卷进行初始化',
    )
  }

  async function templates() {
    const catalog = await readCatalog()
    return {
      version: catalog.version,
      generator: catalog.generator,
      templates: catalog.templates.map(({ directory: _directory, ...template }) => template),
    }
  }

  async function moveStagedWorkspace(stagingRoot) {
    const moved = []
    try {
      for (const entry of await readdir(stagingRoot, { withFileTypes: true })) {
        // .workspace/context 可能是平台的独立挂载点。初始化标记与其共用父目录，
        // 因此只移动本次生成的标记，绝不覆盖或移动平台上下文。
        if (entry.name === '.workspace' && entry.isDirectory()) {
          const metadataRoot = path.join(workspaceRoot, '.workspace')
          await mkdir(metadataRoot, { recursive: true })
          for (const metadataEntry of await readdir(path.join(stagingRoot, entry.name))) {
            const target = path.join(metadataRoot, metadataEntry)
            await rename(path.join(stagingRoot, entry.name, metadataEntry), target)
            moved.push(target)
          }
          continue
        }
        const target = path.join(workspaceRoot, entry.name)
        await rename(path.join(stagingRoot, entry.name), target)
        moved.push(target)
      }
    } catch (error) {
      await Promise.all(moved.map((target) => rm(target, { recursive: true, force: true })))
      throw error
    } finally {
      await rm(stagingRoot, { recursive: true, force: true })
    }
  }

  async function performInitialization(templateId) {
    const current = await status()
    if (current.status !== 'uninitialized') {
      throw new WorkspaceInitializationError(
        current.status === 'initialized' ? '工程已经初始化' : current.message || '工程当前不可初始化',
        409,
      )
    }

    const catalog = await readCatalog()
    const template = catalog.templates.find((item) => item.id === templateId)
    if (!template) throw new WorkspaceInitializationError('工程模板不存在', 400)

    const templateRoot = path.resolve(templatesRoot, template.directory)
    if (path.dirname(templateRoot) !== templatesRoot) {
      throw new WorkspaceInitializationError('工程模板目录非法', 400)
    }

    const stagingRoot = path.join(workspaceRoot, `.workspace-initialize-${process.pid}-${Date.now()}`)
    const initializedAt = now()
    try {
      await mkdir(stagingRoot, { recursive: false })
      await cp(templateRoot, stagingRoot, { recursive: true, force: false })
      await makeWorkspaceWritable(stagingRoot)
      try {
        await readFile(path.join(stagingRoot, 'pnpm-lock.yaml'), 'utf8')
      } catch {
        throw new Error('工程模板缺少镜像构建生成的 pnpm-lock.yaml')
      }
      await runCommand(
        'pnpm',
        [
          'install',
          '--offline',
          '--frozen-lockfile',
          '--ignore-workspace',
          '--config.trust-lockfile=true',
          '--store-dir',
          storeDir,
        ],
        { cwd: stagingRoot, env: { ...process.env, CI: 'true' } },
      )

      await mkdir(path.join(stagingRoot, '.workspace'), { recursive: true })
      await writeFile(
        path.join(stagingRoot, '.workspace', 'project.json'),
        `${JSON.stringify({ version: 1, templateId, initializedAt }, null, 2)}\n`,
        'utf8',
      )
      await runCommand('git', ['init', '--initial-branch=main'], { cwd: stagingRoot, env: process.env })
      await runCommand('git', ['config', '--local', 'user.name', 'InduForge'], {
        cwd: stagingRoot,
        env: process.env,
      })
      await runCommand('git', ['config', '--local', 'user.email', 'workspace@induforge.local'], {
        cwd: stagingRoot,
        env: process.env,
      })
      await runCommand('git', ['add', '.'], { cwd: stagingRoot, env: process.env })
      await runCommand('git', ['commit', '-m', 'chore: 初始化工程模板'], {
        cwd: stagingRoot,
        env: process.env,
      })
      await moveStagedWorkspace(stagingRoot)
      lastError = null
      return workspaceState('initialized', templateId, initializedAt)
    } catch (error) {
      await rm(stagingRoot, { recursive: true, force: true })
      lastError = error instanceof Error ? error.message : '工程初始化失败'
      throw error
    }
  }

  async function initialize(templateId) {
    if (typeof templateId !== 'string' || !templateId.trim()) {
      throw new WorkspaceInitializationError('templateId 不能为空', 400)
    }
    if (initialization) {
      throw new WorkspaceInitializationError('工程正在初始化', 409)
    }
    const task = performInitialization(templateId.trim())
    initialization = task
    try {
      return await task
    } finally {
      initialization = null
    }
  }

  // authoring-snapshot 有意不保存 node_modules。恢复后的工作区仍带有初始化标记，
  // 因此不能把“已初始化”误当成“依赖已就绪”；每次启动 Vite 前按锁文件补齐依赖。
  // pnpm 的 frozen 模式不会改写 package.json 或锁文件，只重建 node_modules 中缺失或损坏的依赖。
  async function ensureDependencies() {
    if (dependencyInstallation) return dependencyInstallation
    const task = (async () => {
      const current = await status()
      if (current.status !== 'initialized') {
        throw new WorkspaceInitializationError(current.message || '工程尚未初始化', 409)
      }
      try {
        await readFile(path.join(workspaceRoot, 'package.json'), 'utf8')
        await readFile(path.join(workspaceRoot, 'pnpm-lock.yaml'), 'utf8')
      } catch {
        throw new WorkspaceInitializationError('工程源码缺少 package.json 或 pnpm-lock.yaml，无法恢复依赖', 409)
      }
      await runCommand(
        'pnpm',
        [
          'install',
          '--offline',
          '--frozen-lockfile',
          '--ignore-workspace',
          '--config.trust-lockfile=true',
          '--store-dir',
          storeDir,
        ],
        { cwd: workspaceRoot, env: { ...process.env, CI: 'true' } },
      )
    })()
    dependencyInstallation = task
    try {
      await task
    } finally {
      dependencyInstallation = null
    }
  }

  return { status, templates, initialize, ensureDependencies }
}
