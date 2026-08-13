import { execFile, spawn } from 'node:child_process'
import { createServer } from 'node:http'
import process from 'node:process'
import { promisify } from 'node:util'

const execFileAsync = promisify(execFile)
const controlPort = Number.parseInt(process.env.PREVIEW_CONTROL_PORT || '5174', 10)
const previewPort = Number.parseInt(process.env.VITE_PORT || '5173', 10)
const workspaceRoot = process.env.WORKSPACE_ROOT || '/workspace'
const allowedOrigins = new Set(
  (process.env.PREVIEW_CONTROL_ALLOWED_ORIGINS || '')
    .split(',')
    .map((value) => value.trim())
    .filter(Boolean),
)

let managedProcess = null
let operation = Promise.resolve()
let state = createState('stopped', null, null)

function createState(status, ownership, message) {
  return {
    status,
    ownership,
    port: previewPort,
    updatedAt: new Date().toISOString(),
    message,
  }
}

function updateState(status, ownership, message = null) {
  state = createState(status, ownership, message)
  return state
}

function requestId() {
  return `preview-${Date.now()}-${Math.random().toString(16).slice(2, 8)}`
}

function envelope(data, msg = '操作成功', code = 0) {
  return { code, msg, data, reqId: requestId() }
}

function applyCors(request, response) {
  const origin = request.headers.origin
  if (!origin) return true
  if (allowedOrigins.size > 0 && !allowedOrigins.has(origin)) return false
  response.setHeader('Access-Control-Allow-Origin', origin)
  response.setHeader('Vary', 'Origin')
  response.setHeader('Access-Control-Allow-Methods', 'GET,POST,OPTIONS')
  response.setHeader('Access-Control-Allow-Headers', 'Content-Type')
  return true
}

function sendJson(response, statusCode, body) {
  response.writeHead(statusCode, { 'Content-Type': 'application/json; charset=utf-8' })
  response.end(JSON.stringify(body))
}

async function listenPids() {
  try {
    const { stdout } = await execFileAsync('fuser', ['-n', 'tcp', String(previewPort)], {
      encoding: 'utf8',
    })
    return [...new Set(stdout.match(/\d+/g)?.map(Number) || [])]
  } catch (error) {
    if (error && typeof error === 'object' && 'code' in error && error.code === 1) return []
    throw error
  }
}

function isManagedAlive() {
  return Boolean(managedProcess && managedProcess.exitCode === null && !managedProcess.killed)
}

async function processGroupId(pid) {
  const { stdout } = await execFileAsync('ps', ['-o', 'pgid=', '-p', String(pid)], {
    encoding: 'utf8',
  })
  const groupId = Number.parseInt(stdout.trim(), 10)
  return Number.isInteger(groupId) ? groupId : null
}

async function isManagedListener(pids) {
  if (!isManagedAlive() || !managedProcess) return false
  const groups = await Promise.all(pids.map(processGroupId))
  return groups.includes(managedProcess.pid)
}

async function resolveState() {
  const pids = await listenPids()
  if (pids.length === 0) {
    if (!['starting', 'stopping', 'error'].includes(state.status)) {
      updateState('stopped', null)
    }
    return state
  }
  if (await isManagedListener(pids)) {
    updateState('running', 'managed')
    return state
  }
  updateState('running', 'external')
  return state
}

async function waitForPort(expectedOpen, timeoutMs) {
  const deadline = Date.now() + timeoutMs
  while (Date.now() < deadline) {
    const open = (await listenPids()).length > 0
    if (open === expectedOpen) return true
    await new Promise((resolve) => setTimeout(resolve, 150))
  }
  return false
}

async function ensureCoderProcess(pid) {
  const { stdout } = await execFileAsync('ps', ['-o', 'user=', '-p', String(pid)], {
    encoding: 'utf8',
  })
  if (stdout.trim() !== process.env.USER) {
    throw new Error(`拒绝终止非当前用户进程 ${pid}`)
  }
}

async function signalPid(pid, signal) {
  await ensureCoderProcess(pid)
  try {
    process.kill(pid, signal)
  } catch (error) {
    if (error && typeof error === 'object' && 'code' in error && error.code === 'ESRCH') return
    throw error
  }
}

async function stopCurrentProcess() {
  updateState('stopping', state.ownership)
  const pids = await listenPids()
  if (pids.length === 0) {
    managedProcess = null
    return updateState('stopped', null)
  }

  if (isManagedAlive() && managedProcess) {
    try {
      process.kill(-managedProcess.pid, 'SIGTERM')
    } catch (error) {
      if (!error || typeof error !== 'object' || !('code' in error) || error.code !== 'ESRCH') {
        throw error
      }
    }
  }

  for (const pid of pids) await signalPid(pid, 'SIGTERM')
  if (!(await waitForPort(false, 5000))) {
    for (const pid of await listenPids()) await signalPid(pid, 'SIGKILL')
    if (!(await waitForPort(false, 2000))) throw new Error('5173 端口进程未能停止')
  }
  managedProcess = null
  return updateState('stopped', null)
}

async function startManagedProcess() {
  const pids = await listenPids()
  if (pids.length > 0) {
    return updateState('running', (await isManagedListener(pids)) ? 'managed' : 'external')
  }

  updateState('starting', 'managed')
  const child = spawn('pnpm', ['dev'], {
    cwd: workspaceRoot,
    env: process.env,
    detached: true,
    stdio: ['ignore', 'inherit', 'inherit'],
  })
  managedProcess = child
  child.once('error', (error) => {
    if (managedProcess !== child) return
    managedProcess = null
    updateState('error', null, error.message)
  })
  child.once('exit', (code, signal) => {
    if (managedProcess !== child) return
    managedProcess = null
    if (state.status === 'stopping') return
    updateState(
      code === 0 ? 'stopped' : 'error',
      null,
      code === 0 ? null : `Vite 退出: code=${code ?? 'null'}, signal=${signal ?? 'null'}`,
    )
  })

  if (!(await waitForPort(true, 15000))) {
    await stopCurrentProcess().catch(() => undefined)
    throw new Error('Vite 启动超时')
  }
  return updateState('running', 'managed')
}

function serializeOperation(task) {
  operation = operation.then(task, task)
  return operation
}

async function handleOperation(pathname) {
  if (pathname.endsWith('/start')) return serializeOperation(startManagedProcess)
  if (pathname.endsWith('/stop')) return serializeOperation(stopCurrentProcess)
  if (pathname.endsWith('/restart')) {
    return serializeOperation(async () => {
      await stopCurrentProcess()
      return startManagedProcess()
    })
  }
  throw new Error('未知预览控制操作')
}

const server = createServer(async (request, response) => {
  if (!applyCors(request, response)) {
    sendJson(response, 403, envelope(null, '来源不允许', 10002))
    return
  }
  if (request.method === 'OPTIONS') {
    response.writeHead(204)
    response.end()
    return
  }

  const url = new URL(request.url || '/', `http://${request.headers.host || 'localhost'}`)
  try {
    if (request.method === 'GET' && url.pathname === '/health') {
      sendJson(response, 200, envelope({ status: 'ok' }))
      return
    }
    if (request.method === 'GET' && url.pathname === '/api/v1/preview/status') {
      sendJson(response, 200, envelope(await resolveState()))
      return
    }
    if (request.method === 'POST' && /^\/api\/v1\/preview\/(start|stop|restart)$/.test(url.pathname)) {
      sendJson(response, 200, envelope(await handleOperation(url.pathname)))
      return
    }
    sendJson(response, 404, envelope(null, '接口不存在', 10003))
  } catch (error) {
    const message = error instanceof Error ? error.message : '预览控制失败'
    updateState('error', null, message)
    sendJson(response, 500, envelope(state, message, 30000))
  }
})

server.listen(controlPort, '0.0.0.0', async () => {
  console.log(`Preview Control listening on 0.0.0.0:${controlPort}`)
  try {
    await serializeOperation(startManagedProcess)
  } catch (error) {
    console.error(error)
  }
})

async function shutdown() {
  server.close()
  await serializeOperation(stopCurrentProcess).catch(() => undefined)
  process.exit(0)
}

process.on('SIGTERM', shutdown)
process.on('SIGINT', shutdown)
