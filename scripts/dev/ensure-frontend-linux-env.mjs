import { copyFile, readFile, stat } from 'node:fs/promises'
import { constants } from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import {
  readFrontendLinuxProxyTarget,
  validateFrontendLinuxProxyTarget,
} from './frontend-linux-env.mjs'
import { validateFrontendWorkspaceProxySuffix } from './frontend-workspace-proxy.mjs'
import { resolveFrontendWorkspaceProxyPort } from './frontend-workspace-proxy.mjs'

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
  console.error(
    '已从 .env.frontend-linux.example 创建 .env.frontend-linux。请编辑为真实 Linux 中心地址后重新运行。',
  )
  process.exitCode = 1
}

if (!created) {
  const content = await readFile(envFile, 'utf8')
  const target = readFrontendLinuxProxyTarget(content)
  const validation = validateFrontendLinuxProxyTarget(target)
  if (!validation.valid) {
    console.error(`.env.frontend-linux 无效：${validation.message}`)
    process.exitCode = 1
  }
  const suffix =
    content.match(/^\s*IF_FRONTEND_WORKSPACE_PROXY_SUFFIX\s*=\s*(.*?)\s*(?:#.*)?$/m)?.[1] || ''
  const suffixValidation = validateFrontendWorkspaceProxySuffix(
    suffix.replace(/^['"]|['"]$/g, '').trim(),
  )
  if (!suffixValidation.valid) {
    console.error(`.env.frontend-linux 无效：${suffixValidation.message}`)
    process.exitCode = 1
  }
  const workspaceProxyPort =
    content.match(/^\s*IF_FRONTEND_WORKSPACE_PROXY_PORT\s*=\s*(.*?)\s*(?:#.*)?$/m)?.[1] || ''
  try {
    resolveFrontendWorkspaceProxyPort(workspaceProxyPort.replace(/^['"]|['"]$/g, '').trim())
  } catch (error) {
    console.error(`.env.frontend-linux 无效：${error.message}`)
    process.exitCode = 1
  }
}
