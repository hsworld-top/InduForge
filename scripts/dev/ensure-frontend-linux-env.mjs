import { copyFile, readFile, stat } from 'node:fs/promises'
import { constants } from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { readFrontendLinuxProxyTarget, validateFrontendLinuxProxyTarget } from './frontend-linux-env.mjs'

const scriptDir = path.dirname(fileURLToPath(import.meta.url))
const repoRoot = path.resolve(scriptDir, '../..')
const envFile = path.join(repoRoot, '.env.frontend-linux')
const template = path.join(repoRoot, '.env.frontend-linux.example')

async function exists(file) {
  try {
    await stat(file)
    return true
  } catch (error) {
    if (error?.code === 'ENOENT') return false
    throw error
  }
}

const created = !(await exists(envFile))
if (!created) {
  console.log('使用现有 .env.frontend-linux。')
} else {
  await stat(template)
  await copyFile(template, envFile, constants.COPYFILE_EXCL)
  console.error('已从 .env.frontend-linux.example 创建 .env.frontend-linux。请编辑为真实 Linux 中心地址后重新运行。')
  process.exitCode = 1
}

if (!created) {
  const target = readFrontendLinuxProxyTarget(await readFile(envFile, 'utf8'))
  const validation = validateFrontendLinuxProxyTarget(target)
  if (!validation.valid) {
    console.error(`.env.frontend-linux 无效：${validation.message}`)
    process.exitCode = 1
  }
}
