import assert from 'node:assert/strict'
import { mkdir, mkdtemp, readFile, readdir, writeFile } from 'node:fs/promises'
import os from 'node:os'
import path from 'node:path'
import test from 'node:test'
import { createWorkspaceInitializer, WorkspaceInitializationError } from './workspace-initializer.mjs'

async function fixture() {
  const root = await mkdtemp(path.join(os.tmpdir(), 'induforge-workspace-'))
  const workspaceRoot = path.join(root, 'workspace')
  const templatesRoot = path.join(root, 'templates')
  const templateRoot = path.join(templatesRoot, 'vite-vue-js')
  await mkdir(workspaceRoot)
  await mkdir(templateRoot, { recursive: true })
  await writeFile(
    path.join(templatesRoot, 'catalog.json'),
    JSON.stringify({
      version: 1,
      generator: 'create-vite@test',
      templates: [
        {
          id: 'vite-vue-js',
          name: 'Vue + JavaScript',
          description: 'Vue',
          framework: 'vue',
          language: 'javascript',
          directory: 'vite-vue-js',
        },
      ],
    }),
  )
  await writeFile(path.join(templateRoot, 'package.json'), '{"name":"fixture","private":true}\n')
  await writeFile(path.join(templateRoot, 'pnpm-lock.yaml'), 'lockfileVersion: 9.0\n')
  await writeFile(path.join(templateRoot, '.gitignore'), 'node_modules\n')
  const commands = []
  const initializer = createWorkspaceInitializer({
    workspaceRoot,
    templatesRoot,
    storeDir: path.join(root, 'store'),
    runCommand: async (command, args, options) => {
      commands.push([command, ...args])
      if (command === 'git' && args[0] === 'init') await mkdir(path.join(options.cwd, '.git'))
    },
  })
  return { initializer, workspaceRoot, commands }
}

test('空工作区可以列出模板并完成一次初始化', async () => {
  const { initializer, workspaceRoot, commands } = await fixture()
  assert.equal((await initializer.status()).status, 'uninitialized')
  assert.equal((await initializer.templates()).templates[0].id, 'vite-vue-js')

  const result = await initializer.initialize('vite-vue-js')
  assert.equal(result.status, 'initialized')
  assert.equal((await initializer.status()).templateId, 'vite-vue-js')
  assert.match(await readFile(path.join(workspaceRoot, 'package.json'), 'utf8'), /fixture/)
  assert.equal(JSON.parse(await readFile(path.join(workspaceRoot, '.induforge/project.json'))).version, 1)
  assert.deepEqual(commands[0], [
    'pnpm',
    'install',
    '--offline',
    '--frozen-lockfile',
    '--ignore-workspace',
    '--config.trust-lockfile=true',
    '--store-dir',
    path.join(path.dirname(workspaceRoot), 'store'),
  ])
  assert.deepEqual(commands.at(-1), ['git', 'commit', '-m', 'chore: 初始化工程模板'])
})

test('命令失败时保留 stderr 诊断信息并清理暂存目录', async () => {
  const root = await mkdtemp(path.join(os.tmpdir(), 'induforge-workspace-error-'))
  const workspaceRoot = path.join(root, 'workspace')
  const templatesRoot = path.join(root, 'templates')
  const templateRoot = path.join(templatesRoot, 'vite-vue-js')
  await mkdir(workspaceRoot)
  await mkdir(templateRoot, { recursive: true })
  await writeFile(
    path.join(templatesRoot, 'catalog.json'),
    JSON.stringify({
      version: 1,
      templates: [
        {
          id: 'vite-vue-js',
          name: 'Vue + JavaScript',
          description: 'Vue',
          framework: 'vue',
          language: 'javascript',
          directory: 'vite-vue-js',
        },
      ],
    }),
  )
  await writeFile(path.join(templateRoot, 'package.json'), '{"name":"fixture","private":true}\n')
  await writeFile(path.join(templateRoot, 'pnpm-lock.yaml'), 'lockfileVersion: 9.0\n')

  const initializer = createWorkspaceInitializer({
    workspaceRoot,
    templatesRoot,
    storeDir: path.join(root, 'store'),
    runCommand: async () => {
      const error = new Error('Command failed')
      error.stderr = '[ERR_PNPM_NO_OFFLINE_META] 缺少离线元数据'
      throw error
    },
  })

  await assert.rejects(
    () => initializer.initialize('vite-vue-js'),
    (error) => /ERR_PNPM_NO_OFFLINE_META/.test(error.message),
  )
  assert.deepEqual(await readdir(workspaceRoot), [])
  assert.match((await initializer.status()).message, /ERR_PNPM_NO_OFFLINE_META/)
})

test('已初始化工作区拒绝再次选择模板', async () => {
  const { initializer } = await fixture()
  await initializer.initialize('vite-vue-js')
  await assert.rejects(
    () => initializer.initialize('vite-vue-js'),
    (error) => error instanceof WorkspaceInitializationError && error.statusCode === 409,
  )
})

test('已初始化工作区会按锁文件补齐依赖，供恢复后的预览重新启动', async () => {
  const { initializer, workspaceRoot, commands } = await fixture()
  await initializer.initialize('vite-vue-js')
  commands.length = 0

  await initializer.ensureDependencies()

  assert.deepEqual(commands, [[
    'pnpm',
    'install',
    '--offline',
    '--frozen-lockfile',
    '--ignore-workspace',
    '--config.trust-lockfile=true',
    '--store-dir',
    path.join(path.dirname(workspaceRoot), 'store'),
  ]])
})

test('依赖恢复要求完整的源码与锁文件', async () => {
  const { initializer, workspaceRoot } = await fixture()
  await mkdir(path.join(workspaceRoot, '.induforge'), { recursive: true })
  await writeFile(
    path.join(workspaceRoot, '.induforge', 'project.json'),
    JSON.stringify({ version: 1, templateId: 'vite-vue-js', initializedAt: '2026-09-04T00:00:00.000Z' }),
  )

  await assert.rejects(
    () => initializer.ensureDependencies(),
    (error) => error instanceof WorkspaceInitializationError && /pnpm-lock/.test(error.message),
  )
})

test('仅有平台上下文挂载时仍可初始化，并保留上下文目录', async () => {
  const { initializer, workspaceRoot } = await fixture()
  const contextRoot = path.join(workspaceRoot, '.induforge', 'context')
  await mkdir(contextRoot, { recursive: true })
  await writeFile(path.join(contextRoot, 'runtime.json'), '{"project":"demo"}\n')

  assert.equal((await initializer.status()).status, 'uninitialized')
  await initializer.initialize('vite-vue-js')

  assert.equal((await initializer.status()).status, 'initialized')
  assert.equal(
    await readFile(path.join(contextRoot, 'runtime.json'), 'utf8'),
    '{"project":"demo"}\n',
  )
  assert.equal(JSON.parse(await readFile(path.join(workspaceRoot, '.induforge/project.json'))).version, 1)
})

test('非空且无标记的工作区不会被覆盖', async () => {
  const { initializer, workspaceRoot } = await fixture()
  await writeFile(path.join(workspaceRoot, 'user-file.txt'), 'keep')
  const state = await initializer.status()
  assert.equal(state.status, 'error')
  await assert.rejects(
    () => initializer.initialize('vite-vue-js'),
    (error) => error instanceof WorkspaceInitializationError && error.statusCode === 409,
  )
})
