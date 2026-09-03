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
