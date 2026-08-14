import { execFile } from 'node:child_process'
import {
  cp,
  chmod,
  mkdir,
  readFile,
  readdir,
  rename,
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
  let lastError = null

  async function runCommand(command, args, commandOptions = {}) {
    try {
      return await commandRunner(command, args, commandOptions)
    } catch (error) {
      throw new Error(commandErrorMessage(command, args, error), { cause: error })
    }
  }

  const markerPath = path.join(workspaceRoot, '.induforge', 'project.json')
  const catalogPath = path.join(templatesRoot, 'catalog.json')

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
    const marker = await readMarker()
    if (marker) {
      return workspaceState('initialized', marker.templateId, marker.initializedAt)
    }
    const entries = await readdir(workspaceRoot)
    if (entries.length === 0) {
      const error = lastError
      lastError = null
      return workspaceState('uninitialized', null, null, error)
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
      for (const entry of await readdir(stagingRoot)) {
        const target = path.join(workspaceRoot, entry)
        await rename(path.join(stagingRoot, entry), target)
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

    const stagingRoot = path.join(workspaceRoot, `.induforge-initialize-${process.pid}-${Date.now()}`)
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

      await mkdir(path.join(stagingRoot, '.induforge'), { recursive: true })
      await writeFile(
        path.join(stagingRoot, '.induforge', 'project.json'),
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

  return { status, templates, initialize }
}
