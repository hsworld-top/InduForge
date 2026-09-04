import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import path from 'node:path'
import test from 'node:test'
import { fileURLToPath } from 'node:url'

const workspaceRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../..')

test('四类新工程模板显式配置同源 HTTP Runtime', async () => {
  const entries = [
    'vite-vue-js/src/main.js',
    'vite-vue-ts/src/main.ts',
    'vite-react-js/src/main.jsx',
    'vite-react-ts/src/main.tsx',
  ]
  for (const entry of entries) {
    const source = await readFile(path.join(workspaceRoot, 'contracts/project-templates', entry), 'utf8')
    assert.match(source, /import \{ configureRuntime, createBrowserRuntime \} from '@induforge\/runtime-sdk'/)
	assert.match(source, /configureRuntime\(createBrowserRuntime\(\)\)/)
  }
})

test('已运行的预览不会因重复启动请求重装依赖', async () => {
  const source = await readFile(path.join(workspaceRoot, 'designer/code-workspace/preview-control.mjs'), 'utf8')
  const start = source.indexOf('async function startManagedProcess()')
  const body = source.slice(start, source.indexOf('\nasync function serializeOperation', start))
  const listenerCheck = body.indexOf('const pids = await listenPids()')
  const dependencyInstall = body.indexOf('await workspaceInitializer.ensureDependencies()')
  assert.ok(listenerCheck >= 0, '启动流程必须先检查已有 5173 监听进程')
  assert.ok(dependencyInstall > listenerCheck, '仅在需要新启 Vite 时才允许恢复依赖')
  assert.match(body.slice(listenerCheck, dependencyInstall), /if \(pids\.length > 0\)[\s\S]*return updateState\('running'/)
})
