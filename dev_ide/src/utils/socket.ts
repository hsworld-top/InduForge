import { io, type Socket } from 'socket.io-client'
import { clearAuthAndRedirectToLogin, refreshSession } from './request'

let socket: Socket | null = null
let legacyTenantId: string | null = null
let authRecovery: Promise<void> | null = null
let authAttempts = 0
let authWindow = 0
let authForbidden = false
let reconnectTimer: ReturnType<typeof setTimeout> | undefined

/** 仅已识别的会话过期（或服务端主动断开）续租；普通网络错误交给Socket重连。 */
function recoverSocketSession(target: Socket) {
  if (authForbidden || authRecovery || reconnectTimer || socket !== target) return
  if (Date.now() - authWindow > 60000) {
    authWindow = Date.now()
    authAttempts = 0
  }
  if (authAttempts >= 2) return
  authAttempts++
  authRecovery = refreshSession()
    .then((valid) => {
      if (socket !== target || authForbidden) return
      if (!valid) {
        authForbidden = true
        target.disconnect()
        clearAuthAndRedirectToLogin()
        return
      }
      reconnectTimer = setTimeout(() => {
        reconnectTimer = undefined
        if (socket === target && !authForbidden) target.connect()
      }, 500 * authAttempts)
    })
    .finally(() => {
      authRecovery = null
    })
}

export const initSocket = (tenantId?: string | null): Socket => {
  if (tenantId) legacyTenantId = tenantId
  if (socket) {
    // 运维可能先建立共享连接；旧消费者后挂载时仍须补订阅，重连也保留该登记。
    if (tenantId && socket.connected) socket.emit('ops:subscribe', { tenantId })
    return socket
  }

  // 浏览器只能连当前 origin；Vite 负责把控制面 Socket 代理到本机或远程中心。
  // 这样远程中心无需为每台开发机的 Vite 端口放开 CORS。
  const socketHost = window.location.origin
  console.log('[Socket] init host:', socketHost)

  socket = io(socketHost, {
    withCredentials: true,
    path: '/control-socket.io',
    transports: ['websocket', 'polling'],
    timeout: 8000,
    // 网络中断允许长期退避恢复；鉴权拒绝则由下方明确disconnect停止重连。
    reconnectionAttempts: Infinity,
    reconnectionDelayMax: 30000,
  })
  const target = socket
  authForbidden = false
  const authFailure = (code?: string) => {
    if (code === 'AUTH_EXPIRED') recoverSocketSession(target)
    if (code === 'AUTH_FORBIDDEN') {
      authForbidden = true
      if (reconnectTimer) clearTimeout(reconnectTimer)
      target.disconnect()
    }
  }
  socket.on('ops:auth', (event: { code?: string }) => authFailure(event?.code))
  socket.on('connect_error', (error: Error & { data?: { code?: string } }) =>
    authFailure(error.data?.code),
  )

  socket.on('connect', () => {
    console.log('[Socket] Connected to server')
    if (legacyTenantId) {
      socket?.emit('ops:subscribe', { tenantId: legacyTenantId })
    }
  })

  socket.on('disconnect', (reason) => {
    console.log('[Socket] Disconnected')
    if (reason === 'io server disconnect') recoverSocketSession(target)
  })

  return socket
}

export const getSocket = (): Socket | null => socket

export const closeSocket = (): void => {
  legacyTenantId = null
  if (reconnectTimer) clearTimeout(reconnectTimer)
  reconnectTimer = undefined
  authForbidden = true
  authAttempts = 0
  authWindow = 0
  if (socket) {
    socket.close()
    socket = null
  }
}
