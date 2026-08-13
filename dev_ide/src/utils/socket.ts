import { io, type Socket } from 'socket.io-client'

let socket: Socket | null = null

export const initSocket = (tenantId?: string | null): Socket => {
  if (socket) return socket

  const apiURL = (import.meta.env.VITE_API_BASE_URL || import.meta.env.VITE_API_URL || '').trim()
  // 提取 socket.io 的 host，兼容 /api 或 /api/v1 结尾
  const socketHost = apiURL
    .replace(/\/api\/v1\/?$/i, '')
    .replace(/\/api\/?$/i, '')
    .replace(/\/$/, '')

  console.log('[Socket] init host:', socketHost || '(same-origin)')

  socket = io(socketHost, {
	withCredentials: true,
	path: '/control-socket.io',
    transports: ['websocket', 'polling'],
    timeout: 8000,
  })

  socket.on('connect', () => {
    console.log('[Socket] Connected to server')
    if (tenantId) {
      socket?.emit('ops:subscribe', { tenantId })
    }
  })

  socket.on('disconnect', () => {
    console.log('[Socket] Disconnected')
  })

  return socket
}

export const getSocket = (): Socket | null => socket

export const closeSocket = (): void => {
  if (socket) {
    socket.close()
    socket = null
  }
}
