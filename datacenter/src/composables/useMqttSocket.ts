import { ref, watch, unref, onBeforeUnmount } from 'vue'
import { debugLogger } from '@/utils/debug'
import { io } from 'socket.io-client'
import { ElMessage } from 'element-plus'
import { buildMqttSocketSharedKey, createMqttSocketSharedRegistry } from './mqtt-socket-shared'

const sharedRegistry = createMqttSocketSharedRegistry({
  ioFactory: io,
  getApiUrl: () => window.location.origin,
  notifier: ({ type, message }) => {
    ElMessage({
      type,
      message,
      duration: 2000,
      offset: 60,
    })
  },
  logger: console,
})

let nextHandlerId = 1
let nextSubscriptionHandleId = 1
let nextTagHandleId = 1

/**
 * MQTT Socket.IO Composable
 * 负责在 datacenter 内复用 preview session 对应的共享 socket：
 * 1. 同一 projectId + previewSessionId 只建立一条底层连接
 * 2. 多个组件共享 mqtt 订阅与 tag 订阅时做引用计数，避免互相提前解绑
 * 3. 对组件暴露的 API 尽量保持不变，降低页面改造成本
 *
 * @param {import("vue").MaybeRefOrGetter<string | null | undefined>} projectIdSource
 * @param {import("vue").MaybeRefOrGetter<string | null | undefined>} previewSessionIdSource
 */
export function useMqttSocket(projectIdSource, previewSessionIdSource = null) {
  const socket = ref(null)
  const connected = ref(false)

  const localMessageHandlers = new Map()
  const localSubscriptionHandles = new Map()
  const localTagHandles = new Map()

  let currentConnectionKey = ''
  let stopStateWatcher = null

  const readSource = (source) => {
    if (typeof source === 'function') {
      return source()
    }
    return unref(source)
  }

  const readProjectId = () => readSource(projectIdSource) || ''
  const readPreviewSessionId = () => readSource(previewSessionIdSource) || ''
  const resolveDesiredConnectionKey = () => {
    const projectId = String(readProjectId() || '').trim()
    const previewSessionId = String(readPreviewSessionId() || '').trim()

    if (!projectId || !previewSessionId) {
      return ''
    }

    return buildMqttSocketSharedKey({
      projectId,
      previewSessionId,
    })
  }

  const bindSharedState = () => {
    if (typeof stopStateWatcher === 'function') {
      stopStateWatcher()
      stopStateWatcher = null
    }

    if (!currentConnectionKey) {
      socket.value = null
      connected.value = false
      return
    }

    stopStateWatcher = sharedRegistry.watchState(
      currentConnectionKey,
      ({ socket: sharedSocket, connected: sharedConnected }) => {
        socket.value = sharedSocket ?? null
        connected.value = Boolean(sharedConnected)
      },
    )
  }

  const attachLocalResources = (connectionKey) => {
    if (!connectionKey) {
      return
    }

    localMessageHandlers.forEach((handler, handlerId) => {
      sharedRegistry.addMessageHandler(connectionKey, handlerId, handler)
    })

    localSubscriptionHandles.forEach((subscriptionId) => {
      sharedRegistry.subscribeSubscription(connectionKey, subscriptionId)
    })

    localTagHandles.forEach((tagId) => {
      sharedRegistry.subscribeTag(connectionKey, tagId)
    })
  }

  const detachLocalResources = (connectionKey) => {
    if (!connectionKey) {
      return
    }

    localMessageHandlers.forEach((_, handlerId) => {
      sharedRegistry.removeMessageHandler(connectionKey, handlerId)
    })

    localSubscriptionHandles.forEach((subscriptionId) => {
      sharedRegistry.unsubscribeSubscription(connectionKey, subscriptionId)
    })

    localTagHandles.forEach((tagId) => {
      sharedRegistry.unsubscribeTag(connectionKey, tagId)
    })
  }

  const releaseCurrentConnection = () => {
    if (!currentConnectionKey) {
      socket.value = null
      connected.value = false
      return
    }

    detachLocalResources(currentConnectionKey)
    if (typeof stopStateWatcher === 'function') {
      stopStateWatcher()
      stopStateWatcher = null
    }
    sharedRegistry.release(currentConnectionKey)

    currentConnectionKey = ''
    socket.value = null
    connected.value = false
  }

  const connect = () => {
    const desiredConnectionKey = resolveDesiredConnectionKey()
    if (!desiredConnectionKey) {
      releaseCurrentConnection()
      return null
    }

    if (desiredConnectionKey === currentConnectionKey) {
      bindSharedState()
      return socket.value
    }

    const result = sharedRegistry.acquire({
      projectId: readProjectId(),
      previewSessionId: readPreviewSessionId(),
    })

    if (!result) {
      releaseCurrentConnection()
      return null
    }

    if (result.key === currentConnectionKey) {
      bindSharedState()
      return result.entry.socket
    }

    if (currentConnectionKey) {
      detachLocalResources(currentConnectionKey)
      if (typeof stopStateWatcher === 'function') {
        stopStateWatcher()
        stopStateWatcher = null
      }
      sharedRegistry.release(currentConnectionKey)
    }

    currentConnectionKey = result.key
    bindSharedState()
    attachLocalResources(currentConnectionKey)
    return result.entry.socket
  }

  const disconnect = () => {
    releaseCurrentConnection()
  }

  const onMessage = (handler) => {
    if (typeof handler !== 'function') {
      return () => {}
    }

    const handlerId = `global-${nextHandlerId++}`
    localMessageHandlers.set(handlerId, handler)
    if (currentConnectionKey) {
      sharedRegistry.addMessageHandler(currentConnectionKey, handlerId, handler)
    }

    return () => {
      if (currentConnectionKey) {
        sharedRegistry.removeMessageHandler(currentConnectionKey, handlerId)
      }
      localMessageHandlers.delete(handlerId)
    }
  }

  const subscribeMessages = (subscriptionId, handler) => {
    const normalizedSubscriptionId = String(subscriptionId || '').trim()
    if (!normalizedSubscriptionId) {
      return () => {}
    }

    const handlerId = `subscription-${nextHandlerId++}`
    const subscriptionHandleId = `sub-${nextSubscriptionHandleId++}`
    const wrappedHandler = (data) => {
      if (typeof handler === 'function' && data?.subscriptionId === normalizedSubscriptionId) {
        handler(data)
      }
    }

    localMessageHandlers.set(handlerId, wrappedHandler)
    localSubscriptionHandles.set(subscriptionHandleId, normalizedSubscriptionId)

    if (currentConnectionKey) {
      sharedRegistry.addMessageHandler(currentConnectionKey, handlerId, wrappedHandler)
      sharedRegistry.subscribeSubscription(currentConnectionKey, normalizedSubscriptionId)
    }

    return () => {
      if (currentConnectionKey) {
        sharedRegistry.removeMessageHandler(currentConnectionKey, handlerId)
        sharedRegistry.unsubscribeSubscription(currentConnectionKey, normalizedSubscriptionId)
      }

      localMessageHandlers.delete(handlerId)
      localSubscriptionHandles.delete(subscriptionHandleId)
    }
  }

  const subscribeTag = (tagId) => {
    const normalizedTagId = String(tagId || '').trim()
    if (!normalizedTagId) {
      return () => {}
    }

    const tagHandleId = `tag-${nextTagHandleId++}`
    localTagHandles.set(tagHandleId, normalizedTagId)

    if (currentConnectionKey) {
      sharedRegistry.subscribeTag(currentConnectionKey, normalizedTagId)
    }

    return () => {
      if (currentConnectionKey) {
        sharedRegistry.unsubscribeTag(currentConnectionKey, normalizedTagId)
      }
      localTagHandles.delete(tagHandleId)
    }
  }

  const emit = (event, data) => {
    if (!currentConnectionKey) {
      debugLogger.warn('[MqttSocket] Not connected, cannot emit:', event)
      return false
    }

    return sharedRegistry.emit(currentConnectionKey, event, data)
  }

  watch(
    () => [readProjectId(), readPreviewSessionId()],
    () => {
      connect()
    },
    { immediate: true },
  )

  onBeforeUnmount(() => {
    disconnect()
  })

  return {
    socket,
    connected,
    connect,
    disconnect,
    subscribeMessages,
    subscribeTag,
    onMessage,
    emit,
  }
}
