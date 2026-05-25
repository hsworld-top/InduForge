import { computed, onBeforeUnmount, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { getApiErrorMessage } from '@/utils/request'
import type { ProtocolDevSession } from '@/components/opcua/types'

type SessionApi = {
  create: () => Promise<unknown>
  close: (sessionId: string) => Promise<unknown>
}

const unwrapData = (value: unknown): Record<string, unknown> => {
  const response = value as { data?: unknown }
  const nested = response?.data as { data?: unknown } | undefined
  const payload = nested?.data ?? response?.data ?? value
  return payload && typeof payload === 'object' ? (payload as Record<string, unknown>) : {}
}

export const useProtocolDevSession = (api: SessionApi) => {
  const status = ref<'idle' | 'connecting' | 'connected' | 'disconnecting' | 'error'>('idle')
  const sessionId = ref('')
  const connectedAt = ref('')
  const endpoint = ref('')
  const diagnostics = ref<string[]>([])

  const connected = computed(() => status.value === 'connected' && Boolean(sessionId.value))

  const applySession = (payload: Record<string, unknown>) => {
    const session = payload as Partial<ProtocolDevSession>
    sessionId.value = typeof session.sessionId === 'string' ? session.sessionId : ''
    connectedAt.value = typeof session.connectedAt === 'string' ? session.connectedAt : ''
    endpoint.value = typeof session.endpoint === 'string' ? session.endpoint : ''
    diagnostics.value = Array.isArray(session.diagnostics) ? session.diagnostics : []
  }

  const connect = async () => {
    if (status.value === 'connecting' || connected.value) return
    status.value = 'connecting'
    try {
      applySession(unwrapData(await api.create()))
      status.value = sessionId.value ? 'connected' : 'error'
      if (!sessionId.value) {
        ElMessage.error('连接失败：未返回会话 ID')
      }
    } catch (error) {
      status.value = 'error'
      ElMessage.error(getApiErrorMessage(error, '连接失败'))
    }
  }

  const disconnect = async () => {
    if (!sessionId.value) {
      status.value = 'idle'
      return
    }
    status.value = 'disconnecting'
    const closingId = sessionId.value
    sessionId.value = ''
    try {
      await api.close(closingId)
      status.value = 'idle'
      connectedAt.value = ''
      endpoint.value = ''
      diagnostics.value = []
    } catch (error) {
      status.value = 'error'
      ElMessage.error(getApiErrorMessage(error, '断开连接失败'))
    }
  }

  onBeforeUnmount(() => {
    if (sessionId.value) {
      void api.close(sessionId.value)
    }
  })

  return { status, sessionId, connectedAt, endpoint, diagnostics, connected, connect, disconnect }
}
