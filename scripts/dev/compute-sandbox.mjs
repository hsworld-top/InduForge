import { readFile } from 'node:fs/promises'
import path from 'node:path'
import { spawnSync } from 'node:child_process'
import { fileURLToPath } from 'node:url'

const scriptDir = path.dirname(fileURLToPath(import.meta.url))
const repoRoot = path.resolve(scriptDir, '../..')
const envFile = path.join(repoRoot, '.env')
const imageName = 'induforge/compute-sandbox:dev'
const containerName = 'induforge-compute-sandbox-dev'
const healthURL = 'http://127.0.0.1:18103/health'
const capabilitiesURL = 'http://127.0.0.1:18103/v1/capabilities'

function runDocker(args, options = {}) {
  const result = spawnSync('docker', args, {
    cwd: repoRoot,
    encoding: 'utf8',
    stdio: options.capture ? 'pipe' : 'inherit',
    env: options.env || process.env,
  })
  if (result.error) throw result.error
  if (!options.allowFailure && result.status !== 0) {
    process.exit(result.status || 1)
  }
  return result
}

function inspectContainer() {
  const result = runDocker(['inspect', '-f', '{{.State.Running}}', containerName], {
    capture: true,
    allowFailure: true,
  })
  if (result.status !== 0) return { exists: false, running: false }
  return { exists: true, running: result.stdout.trim() === 'true' }
}

async function readSandboxToken() {
  const content = await readFile(envFile, 'utf8')
  const line = content
    .split(/\r?\n/)
    .find((item) => item.trimStart().startsWith('DATA_SERVICE_COMPUTE_SANDBOX_TOKEN='))
  let value = line?.slice(line.indexOf('=') + 1).trim() || ''
  if (
    value.length >= 2 &&
    ((value.startsWith('"') && value.endsWith('"')) ||
      (value.startsWith("'") && value.endsWith("'")))
  ) {
    value = value.slice(1, -1)
  }
  if (value.length < 24) {
    throw new Error('根目录 .env 的 DATA_SERVICE_COMPUTE_SANDBOX_TOKEN 未配置或少于 24 个字符。')
  }
  return value
}

async function waitUntilReady(token) {
  let lastError
  for (let attempt = 0; attempt < 30; attempt += 1) {
    try {
      const health = await fetch(healthURL, { signal: AbortSignal.timeout(1000) })
      if (!health.ok) throw new Error(`健康检查返回 HTTP ${health.status}`)
      const response = await fetch(capabilitiesURL, {
        headers: { Authorization: `Bearer ${token}` },
        signal: AbortSignal.timeout(2000),
      })
      if (!response.ok) throw new Error(`能力接口返回 HTTP ${response.status}`)
      const capabilities = await response.json()
      if (!capabilities.available) throw new Error('容器已启动，但 Linux 隔离能力未就绪')
      const languages = (capabilities.languages || [])
        .map((item) => `${item.language} ${item.version}`)
        .join('、')
      console.log(`计算沙箱可用：${languages || '未声明语言版本'}；地址 ${healthURL}`)
      return
    } catch (error) {
      lastError = error
      await new Promise((resolve) => setTimeout(resolve, 250))
    }
  }
  throw lastError || new Error('计算沙箱启动超时')
}

async function start({ rebuild = false } = {}) {
  const token = await readSandboxToken()
  const current = inspectContainer()

  if (rebuild && current.exists) {
    runDocker(['rm', '-f', containerName])
  } else if (current.running) {
    console.log(`容器 ${containerName} 已在运行。`)
    await waitUntilReady(token)
    return
  } else if (current.exists) {
    runDocker(['start', containerName])
    await waitUntilReady(token)
    return
  }

  runDocker(['build', '-t', imageName, '-f', 'compute_sandbox/Dockerfile', 'compute_sandbox'])
  runDocker(
    [
      'run',
      '-d',
      '--name',
      containerName,
      '--restart',
      'unless-stopped',
      '-p',
      '127.0.0.1:18103:18103',
      '-e',
      'COMPUTE_SANDBOX_ADDR=:18103',
      '-e',
      'COMPUTE_SANDBOX_TOKEN',
      '--read-only',
      '--tmpfs',
      '/tmp:size=64m,mode=1777',
      '--mount',
      'type=volume,source=induforge-compute-dependencies-dev,target=/dependencies',
      '--cap-drop',
      'ALL',
      '--cap-add',
      'SYS_ADMIN',
      '--cap-add',
      'NET_ADMIN',
      '--cap-add',
      'SETUID',
      '--cap-add',
      'SETGID',
      '--security-opt',
      'no-new-privileges=true',
      '--security-opt',
      'seccomp=unconfined',
      '--pids-limit',
      '64',
      '--memory',
      '384m',
      imageName,
    ],
    { env: { ...process.env, COMPUTE_SANDBOX_TOKEN: token } },
  )
  await waitUntilReady(token)
}

async function status() {
  const current = inspectContainer()
  if (!current.exists) {
    console.log(`容器 ${containerName} 尚未创建。`)
    return
  }
  runDocker([
    'ps',
    '-a',
    '--filter',
    `name=^/${containerName}$`,
    '--format',
    'table {{.Names}}\t{{.Status}}\t{{.Ports}}',
  ])
  if (current.running) await waitUntilReady(await readSandboxToken())
}

const command = process.argv[2] || 'start'

try {
  if (command === 'start') await start()
  else if (command === 'rebuild') await start({ rebuild: true })
  else if (command === 'stop') {
    const current = inspectContainer()
    if (current.running) runDocker(['stop', containerName])
    else console.log(`容器 ${containerName} 当前未运行。`)
  } else if (command === 'status') await status()
  else if (command === 'logs') runDocker(['logs', '-f', containerName])
  else throw new Error(`未知命令：${command}`)
} catch (error) {
  console.error(error instanceof Error ? error.message : String(error))
  process.exit(1)
}
