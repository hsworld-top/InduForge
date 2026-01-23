import { io } from 'socket.io-client'
import { Storage } from './storage'

let socket = null

export const initSocket = (tenantId) => {
  if (socket) return socket

  const token = Storage.getToken()
  const baseURL = import.meta.env.VITE_API_BASE_URL || ''
  
  // 提取 socket.io 的 host
  const socketHost = baseURL.replace('/api/v1', '')

  socket = io(socketHost, {
    auth: { token },
    transports: ['websocket', 'polling'],
  })

  socket.on('connect', () => {
    console.log('[Socket] Connected to server')
    if (tenantId) {
      socket.emit('ops:subscribe', { tenantId })
    }
  })

  socket.on('disconnect', () => {
    console.log('[Socket] Disconnected')
  })

  return socket
}

export const getSocket = () => socket

export const closeSocket = () => {
  if (socket) {
    socket.close()
    socket = null
  }
}
