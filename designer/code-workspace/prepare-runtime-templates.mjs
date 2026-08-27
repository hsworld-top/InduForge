import { cp, mkdir, readFile, writeFile } from 'node:fs/promises'
import path from 'node:path'
import process from 'node:process'
import { execFile } from 'node:child_process'
import { promisify } from 'node:util'

const execFileAsync = promisify(execFile)
const [templatesRoot, sdkArchive, storeDir] = process.argv.slice(2).map((value) => path.resolve(value))
if (!templatesRoot || !sdkArchive || !storeDir) throw new Error('缺少模板目录、SDK 包或 pnpm store 参数')

for (const directory of ['vite-vue-js', 'vite-vue-ts', 'vite-react-js', 'vite-react-ts']) {
  const root = path.join(templatesRoot, directory)
  const packageDir = path.join(root, '.induforge', 'packages')
  const packagePath = path.join(root, 'package.json')
  await mkdir(packageDir, { recursive: true })
  await cp(sdkArchive, path.join(packageDir, 'runtime-sdk.tgz'))
  const manifest = JSON.parse(await readFile(packagePath, 'utf8'))
  manifest.dependencies['@induforge/runtime-sdk'] = 'file:.induforge/packages/runtime-sdk.tgz'
  await writeFile(packagePath, `${JSON.stringify(manifest, null, 2)}\n`, 'utf8')
  await execFileAsync('pnpm', [
    'install', '--lockfile-only', '--offline', '--ignore-workspace',
    '--config.trust-lockfile=true', '--store-dir', storeDir,
  ], { cwd: root, env: { ...process.env, CI: 'true' } })
}
