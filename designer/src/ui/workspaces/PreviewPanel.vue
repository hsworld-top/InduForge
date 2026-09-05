<script setup lang="ts">
import { computed, nextTick, onBeforeMount, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { io, type Socket } from 'socket.io-client'
import IconLucideCheck from '~icons/lucide/check'
import IconLucideExternalLink from '~icons/lucide/external-link'
import IconLucideMaximize2 from '~icons/lucide/maximize-2'
import IconLucideMonitorSmartphone from '~icons/lucide/monitor-smartphone'
import IconLucidePower from '~icons/lucide/power'
import IconLucideRefreshCw from '~icons/lucide/refresh-cw'
import IconLucideRotateCcw from '~icons/lucide/rotate-ccw'
import request, { getApiErrorMessage, resolveApiError } from '@/utils/request'
import { sceneContractApi, type SceneKind } from './scene-contract-api'
import { previewControlApi, type PreviewControlState } from './code/preview-control-api'

type DeviceMode = 'web' | 'tablet' | 'mobile'
type PreviewOperation = 'start' | 'stop' | 'restart'

interface DevicePreset {
  id: DeviceMode
  label: string
  width: number | null
  height: number | null
}

interface PreviewHostBridge {
  onRegisterWindowMessageListener?: (listener: (event: MessageEvent) => void) => () => void
  onPostWindowMessage?: (target: Window, data: unknown, targetOrigin: string) => void
}

const props = defineProps<{
  projectId: string
  previewUrl: string | null
  controlUrl: string | null
  active: boolean
}>()

const devices: DevicePreset[] = [
  { id: 'web', label: '网页', width: null, height: null },
  { id: 'tablet', label: '平板', width: 768, height: 1024 },
  { id: 'mobile', label: '手机', width: 390, height: 844 },
]

const panelRef = ref<HTMLElement | null>(null)
const canvasRef = ref<HTMLElement | null>(null)
const previewFrameRef = ref<HTMLIFrameElement | null>(null)
const selectedDevice = ref<DeviceMode>('web')
const deviceMenuOpen = ref(false)
const frameLoaded = ref(false)
const frameKey = ref(0)
const scale = ref(1)
const processState = ref<PreviewControlState | null>(null)
const processError = ref('')
const operation = ref<PreviewOperation | null>(null)
let pollTimer: ReturnType<typeof setTimeout> | null = null
let resizeObserver: ResizeObserver | null = null
let revisionChannel: BroadcastChannel | null = null
let unregisterHostMessageListener: (() => void) | null = null
let dataSocket: Socket | null = null
let dataSocketPromise: Promise<Socket> | null = null
let dataPreviewSessionId = ''
let dataHeartbeatTimer: number | null = null
const viewerSessions = new Map<string, { datapointRefs: Set<string> }>()
const dataSubscriptions = new Map<string, Set<string>>()
const pageDataSubscriptions = new Map<string, Set<string>>()
const dataSocketRequests = new Map<
  string,
  { resolve: (value: unknown) => void; reject: (error: Error) => void; timer: number }
>()

const PREVIEW_CHANNEL = 'induforge-preview-runtime'
const PAGE_RUNTIME_CHANNEL = 'induforge-page-runtime'

const currentDevice = computed(
  () => devices.find((device) => device.id === selectedDevice.value) ?? devices[0]!,
)
const processRunning = computed(() => processState.value?.status === 'running')
const processBusy = computed(() =>
  ['starting', 'stopping'].includes(processState.value?.status || ''),
)
const processLabel = computed(() => {
  if (processState.value?.status === 'starting') return '正在启动'
  if (processState.value?.status === 'stopping') return '正在停止'
  if (processState.value?.status === 'running') {
    return processState.value.ownership === 'external' ? '外部进程' : '运行中'
  }
  if (processState.value?.status === 'error') return '异常'
  return '已停止'
})
const primaryActionLabel = computed(() => {
  if (operation.value === 'start') return '启动中'
  if (operation.value === 'stop') return '停止中'
  return processRunning.value ? '停止' : '启动'
})
const deviceFrameStyle = computed(() => {
  const device = currentDevice.value
  if (!device.width || !device.height) return undefined
  return {
    width: `${device.width}px`,
    height: `${device.height}px`,
    transform: `scale(${scale.value})`,
  }
})
const deviceStageStyle = computed(() => {
  const device = currentDevice.value
  if (!device.width || !device.height) return undefined
  return {
    width: `${device.width * scale.value}px`,
    height: `${device.height * scale.value}px`,
  }
})

onBeforeMount(() => {
  window.addEventListener('message', handleRuntimeMessage)
  unregisterHostMessageListener =
    getPreviewHostBridge()?.onRegisterWindowMessageListener?.(handleRuntimeMessage) || null
})

onMounted(() => {
  resizeObserver = new ResizeObserver(() => updateScale())
  if (canvasRef.value) resizeObserver.observe(canvasRef.value)
  void refreshProcessState()
  if (props.projectId && typeof BroadcastChannel !== 'undefined') {
    revisionChannel = new BroadcastChannel(`induforge-scene-revisions:${props.projectId}`)
    revisionChannel.addEventListener('message', handleRevisionBroadcast)
  }
})

onBeforeUnmount(() => {
  resizeObserver?.disconnect()
  clearPoll()
  window.removeEventListener('message', handleRuntimeMessage)
  unregisterHostMessageListener?.()
  unregisterHostMessageListener = null
  revisionChannel?.close()
  revisionChannel = null
  dataSocket?.disconnect()
  dataSocket = null
  dataSocketPromise = null
  for (const pending of dataSocketRequests.values()) {
    window.clearTimeout(pending.timer)
    pending.reject(new Error('场景数据连接已关闭'))
  }
  dataSocketRequests.clear()
  if (dataHeartbeatTimer) clearInterval(dataHeartbeatTimer)
  dataHeartbeatTimer = null
  viewerSessions.clear()
  dataSubscriptions.clear()
  pageDataSubscriptions.clear()
})

watch(
  () => props.controlUrl,
  () => void refreshProcessState(),
)
watch(
  () => props.active,
  () => schedulePoll(),
)
watch(selectedDevice, () => void nextTick(updateScale))

function updateScale(): void {
  const device = currentDevice.value
  const canvas = canvasRef.value
  if (!canvas || !device.width || !device.height) {
    scale.value = 1
    return
  }
  const horizontalPadding = 36
  const verticalPadding = 36
  scale.value = Math.min(
    1,
    Math.max(0.2, (canvas.clientWidth - horizontalPadding) / device.width),
    Math.max(0.2, (canvas.clientHeight - verticalPadding) / device.height),
  )
}

function clearPoll(): void {
  if (pollTimer) clearTimeout(pollTimer)
  pollTimer = null
}

function schedulePoll(fast = false): void {
  clearPoll()
  if (!props.controlUrl) return
  const unstable = ['starting', 'stopping'].includes(processState.value?.status || '')
  const delay = fast || unstable ? 1000 : props.active ? 3000 : 10000
  pollTimer = setTimeout(() => void refreshProcessState(), delay)
}

async function refreshProcessState(): Promise<void> {
  clearPoll()
  if (!props.controlUrl) {
    processState.value = null
    processError.value = '预览控制服务尚未接入'
    return
  }
  try {
    processState.value = await previewControlApi.status(props.controlUrl)
    processError.value = ''
  } catch (error) {
    processError.value = getApiErrorMessage(error, '读取预览进程状态失败')
  } finally {
    schedulePoll()
  }
}

async function runOperation(target: PreviewOperation): Promise<void> {
  if (!props.controlUrl || operation.value || processBusy.value) return
  operation.value = target
  processError.value = ''
  releasePageDatapointSubscriptions()
  try {
    processState.value = await previewControlApi[target](props.controlUrl)
    if (target !== 'stop') frameLoaded.value = false
  } catch (error) {
    processError.value = getApiErrorMessage(
      error,
      `预览服务${target === 'stop' ? '停止' : '启动'}失败`,
    )
  } finally {
    operation.value = null
    schedulePoll(true)
  }
}

function toggleProcess(): void {
  void runOperation(processRunning.value ? 'stop' : 'start')
}

function selectDevice(device: DeviceMode): void {
  selectedDevice.value = device
  deviceMenuOpen.value = false
}

function reloadPreview(): void {
  if (!props.previewUrl || !processRunning.value) return
  releasePageDatapointSubscriptions()
  frameLoaded.value = false
  frameKey.value += 1
}

function openPreviewWindow(): void {
  if (!props.projectId) return
  const url = new URL('/designer/preview', window.location.origin)
  url.searchParams.set('projectId', props.projectId)
  window.open(url.toString(), '_blank', 'noopener,noreferrer')
}

async function handleRuntimeMessage(event: MessageEvent): Promise<void> {
  const frame = previewFrameRef.value
  if (!frame || event.source !== frame.contentWindow || !props.previewUrl) return
  if (event.origin !== new URL(props.previewUrl).origin) return
  const message = event.data as Record<string, unknown>
  if (
    message?.channel === PAGE_RUNTIME_CHANNEL &&
    message.version === 1 &&
    message.type === 'RELEASE_ALL'
  ) {
    releasePageDatapointSubscriptions()
    return
  }
  if (
    message?.channel === PAGE_RUNTIME_CHANNEL &&
    message.version === 1 &&
    message.type === 'REQUEST'
  ) {
    await handlePageRuntimeRequest(message, event.origin, event.source as Window)
    return
  }
  if (
    message?.channel === PREVIEW_CHANNEL &&
    message.version === 1 &&
    message.type === 'SCENE_DATA_REQUEST'
  ) {
    await handleSceneDataRequest(message, event.origin)
    return
  }
  if (
    message?.channel === PREVIEW_CHANNEL &&
    message.version === 1 &&
    message.type === 'RELEASE_SCENE_VIEWER' &&
    typeof message.viewerSessionId === 'string'
  ) {
    releaseViewerSession(message.viewerSessionId)
    return
  }
  if (
    !message ||
    message.channel !== PREVIEW_CHANNEL ||
    message.version !== 1 ||
    message.type !== 'CREATE_SCENE_VIEWER' ||
    typeof message.requestId !== 'string' ||
    !message.requestId.trim() ||
    typeof message.sceneId !== 'string' ||
    !message.sceneId.trim() ||
    !['2d', '3d'].includes(String(message.kind))
  ) return
  try {
    const session = await sceneContractApi.viewerSession(
      props.projectId,
      message.kind as SceneKind,
      message.sceneId,
    )
    viewerSessions.set(session.sessionId, {
      datapointRefs: new Set(Array.isArray(session.datapointRefs) ? session.datapointRefs : []),
    })
    frame.contentWindow?.postMessage(
      {
        channel: PREVIEW_CHANNEL,
        version: 1,
        type: 'SCENE_VIEWER_RESULT',
        requestId: message.requestId,
        data: { ...session, url: new URL(session.url, window.location.origin).toString() },
      },
      event.origin,
    )
  } catch (error) {
    const resolved = resolveApiError(error, '创建场景 Viewer 失败')
    frame.contentWindow?.postMessage(
      {
        channel: PREVIEW_CHANNEL,
        version: 1,
        type: 'SCENE_VIEWER_RESULT',
        requestId: message.requestId,
        error: { code: resolved.code || 26003, msg: resolved.msg, reqId: resolved.reqId || '' },
      },
      event.origin,
    )
  }
}

// 预览数据同样属于工程开发态。统一请求封装负责租户、登录续期和开发代次，
// 避免恢复工程后缺失代次的 POST 被写栅栏拒绝。
async function requestPreviewData(path: string, body?: unknown): Promise<unknown> {
  const response = await request.post<{ data?: unknown }>(path, body, {
    authoringProjectId: props.projectId,
  })
  return response.data
}

async function ensureDataSocket(): Promise<Socket> {
  if (dataSocket?.connected) return dataSocket
  if (dataSocketPromise) return dataSocketPromise
  dataSocketPromise = connectDataSocket()
  try {
    return await dataSocketPromise
  } finally {
    dataSocketPromise = null
  }
}

async function connectDataSocket(): Promise<Socket> {
  if (!dataPreviewSessionId) {
    const data = (await requestPreviewData(
      `/data/projects/${encodeURIComponent(props.projectId)}/preview/sessions`,
      { meta: { consumer: 'designer-preview' } },
    )) as { id?: string; sessionId?: string }
    dataPreviewSessionId = String(data.sessionId || data.id || '')
    if (!dataPreviewSessionId) throw new Error('数据预览会话响应无效')
    dataHeartbeatTimer = window.setInterval(() => {
      void requestPreviewData(
        `/data/preview/sessions/${encodeURIComponent(dataPreviewSessionId)}/heartbeat`,
      ).catch(() => {})
    }, 10 * 60 * 1000)
  }
  dataSocket = io(window.location.origin, {
    path: '/socket.io',
    transports: ['websocket', 'polling'],
    withCredentials: true,
    auth: { projectId: props.projectId, previewSessionId: dataPreviewSessionId },
  })
  dataSocket.on('datapoint:value', (payload: { path?: string }) => forwardDatapointValue(payload))
  dataSocket.on('datapoint:values', (values: Array<{ path?: string }>) => values.forEach(forwardDatapointValue))
  dataSocket.on('connect', () => {
    // 数据服务开发态重载或短暂断线后，Socket.IO 会重连，但服务端订阅状态需重新声明。
    for (const path of new Set([...dataSubscriptions.keys(), ...pageDataSubscriptions.keys()])) {
      void requestDataSocket(dataSocket!, 'datapoint:subscribe', path).catch(() => {})
    }
  })
  dataSocket.on('response', (payload: { requestId?: string; success?: boolean; result?: unknown; error?: string }) => {
    const pending = dataSocketRequests.get(String(payload.requestId || ''))
    if (!pending) return
    window.clearTimeout(pending.timer)
    dataSocketRequests.delete(String(payload.requestId))
    if (payload.success) pending.resolve(payload.result)
    else pending.reject(new Error(payload.error || '数据订阅操作失败'))
  })
  dataSocket.on('disconnect', () => {
    for (const pending of dataSocketRequests.values()) {
      window.clearTimeout(pending.timer)
      pending.reject(new Error('数据订阅连接已断开'))
    }
    dataSocketRequests.clear()
  })
  if (!dataSocket.connected) {
    await new Promise<void>((resolve, reject) => {
      const timer = window.setTimeout(() => reject(new Error('数据订阅连接超时')), 8000)
      dataSocket?.once('connect', () => { clearTimeout(timer); resolve() })
      dataSocket?.once('connect_error', (error) => { clearTimeout(timer); reject(error) })
    })
  }
  return dataSocket
}

function requestDataSocket(socket: Socket, event: string, path: string): Promise<unknown> {
  const requestId = window.crypto?.randomUUID?.() || `${Date.now()}-${Math.random()}`
  return new Promise((resolve, reject) => {
    const timer = window.setTimeout(() => {
      dataSocketRequests.delete(requestId)
      reject(new Error('数据订阅操作超时'))
    }, 8000)
    dataSocketRequests.set(requestId, { resolve, reject, timer })
    socket.emit(event, { requestId, path })
  })
}

function releaseViewerSession(viewerSessionId: string): void {
  if (!viewerSessionId) return
  viewerSessions.delete(viewerSessionId)
  for (const [path, subscribers] of dataSubscriptions) {
    subscribers.delete(viewerSessionId)
    if (subscribers.size) continue
    dataSubscriptions.delete(path)
    if (!hasDatapointSubscribers(path) && dataSocket?.connected) {
      void requestDataSocket(dataSocket, 'datapoint:unsubscribe', path).catch(() => {})
    }
  }
}

function forwardDatapointValue(payload: { path?: string; [key: string]: unknown }): void {
  const path = String(payload.path || '')
  for (const viewerSessionId of dataSubscriptions.get(path) || []) {
    previewFrameRef.value?.contentWindow?.postMessage(
      { channel: PREVIEW_CHANNEL, version: 1, type: 'SCENE_DATA_PUSH', viewerSessionId, path, data: payload },
      props.previewUrl ? new URL(props.previewUrl).origin : '*',
    )
  }
  for (const subscriptionId of pageDataSubscriptions.get(path) || []) {
    postPageRuntimeMessage(
      {
        channel: PAGE_RUNTIME_CHANNEL,
        version: 1,
        type: 'EVENT',
        subscriptionId,
        path,
        data: payload,
      },
      props.previewUrl ? new URL(props.previewUrl).origin : '*',
    )
  }
}

function postPageRuntimeMessage(data: unknown, targetOrigin: string, targetWindow?: Window): void {
  const target = targetWindow || previewFrameRef.value?.contentWindow
  if (!target) return
  const hostPostMessage = getPreviewHostBridge()?.onPostWindowMessage
  if (hostPostMessage) {
    hostPostMessage(target, data, targetOrigin)
    return
  }
  target.postMessage(data, targetOrigin)
}

function getPreviewHostBridge(): PreviewHostBridge | null {
  const bridge = window.$wujie?.props
  return bridge && typeof bridge === 'object' ? (bridge as PreviewHostBridge) : null
}

function hasDatapointSubscribers(path: string): boolean {
  return Boolean(dataSubscriptions.get(path)?.size || pageDataSubscriptions.get(path)?.size)
}

function pageRuntimeResult(code: number, msg: string, data: unknown, reqId?: string) {
  return { code, msg, data, ...(reqId ? { reqId } : {}) }
}

async function readPageDatapoint(path: string): Promise<unknown> {
  const data = (await requestPreviewData(
    `/data/projects/${encodeURIComponent(props.projectId)}/datapoints/values`,
    { paths: [path] },
  )) as { values?: Record<string, unknown> }
  return data.values?.[path] ?? null
}

async function writePageDatapoint(path: string, value: unknown): Promise<unknown> {
  return requestPreviewData(
    `/data/projects/${encodeURIComponent(props.projectId)}/datapoints/write-by-path`,
    { path, value },
  )
}

async function subscribePageDatapoint(path: string, subscriptionId: string): Promise<void> {
  const socket = await ensureDataSocket()
  const subscriptions = pageDataSubscriptions.get(path) || new Set<string>()
  if (subscriptions.has(subscriptionId)) return
  const firstSubscriber = !hasDatapointSubscribers(path)
  subscriptions.add(subscriptionId)
  pageDataSubscriptions.set(path, subscriptions)
  try {
    if (firstSubscriber) await requestDataSocket(socket, 'datapoint:subscribe', path)
  } catch (error) {
    subscriptions.delete(subscriptionId)
    if (!subscriptions.size) pageDataSubscriptions.delete(path)
    throw error
  }
}

async function unsubscribePageDatapoint(path: string, subscriptionId: string): Promise<void> {
  const subscriptions = pageDataSubscriptions.get(path)
  subscriptions?.delete(subscriptionId)
  if (subscriptions && !subscriptions.size) pageDataSubscriptions.delete(path)
  if (!hasDatapointSubscribers(path) && dataSocket?.connected) {
    await requestDataSocket(dataSocket, 'datapoint:unsubscribe', path)
  }
}

async function handlePageRuntimeRequest(
  message: Record<string, unknown>,
  targetOrigin: string,
  targetWindow: Window,
): Promise<void> {
  const requestId = String(message.requestId || '')
  const domain = String(message.domain || '')
  const operation = String(message.operation || '')
  const path = String(message.path || '').trim()
  const args = Array.isArray(message.args) ? message.args : []
  const subscriptionId = String(message.subscriptionId || '')
  if (!requestId) return

  const respond = (result: ReturnType<typeof pageRuntimeResult>) => {
    postPageRuntimeMessage(
      { channel: PAGE_RUNTIME_CHANNEL, version: 1, type: 'RESULT', requestId, result },
      targetOrigin,
      targetWindow,
    )
  }

  if (domain !== 'point') {
    respond(pageRuntimeResult(50031, `开发态预览尚未提供 ${domain}.${operation}() 能力`, null))
    return
  }
  if (!path) {
    respond(pageRuntimeResult(26007, '数据点路径不能为空', null))
    return
  }

  try {
    if (['get', 'read', 'peek', 'refresh'].includes(operation)) {
      const value = await readPageDatapoint(path)
      respond(pageRuntimeResult(0, 'ok', value))
      return
    }
    if (operation === 'set' || operation === 'publish') {
      const result = await writePageDatapoint(path, args[0])
      // 开发态页面自身写入后立即回推给同页订阅者；节点运行态由数据总线负责广播。
      forwardDatapointValue({ path, data: args[0], value: args[0] })
      respond(pageRuntimeResult(0, 'ok', result))
      return
    }
    if (operation === 'subscribe') {
      if (!subscriptionId) {
        respond(pageRuntimeResult(26009, '数据点订阅标识不能为空', null))
        return
      }
      await subscribePageDatapoint(path, subscriptionId)
      respond(pageRuntimeResult(0, 'ok', { path, subscribed: true }))
      return
    }
    if (operation === 'unsubscribe') {
      await unsubscribePageDatapoint(path, subscriptionId)
      respond(pageRuntimeResult(0, 'ok', { path, subscribed: false }))
      return
    }
    respond(pageRuntimeResult(50031, `开发态预览尚未提供 point.${operation}() 能力`, null))
  } catch (error) {
    const resolved = resolveApiError(error, '预览数据操作失败')
    respond(pageRuntimeResult(resolved.code || 50031, resolved.msg, null, resolved.reqId))
  }
}

async function handleSceneDataRequest(message: Record<string, unknown>, targetOrigin: string): Promise<void> {
  const requestId = String(message.requestId || '')
  const viewerSessionId = String(message.viewerSessionId || '')
  const path = String(message.path || '').trim()
  const operation = String(message.operation || '')
  const access = viewerSessions.get(viewerSessionId)
  const respond = (data?: unknown, error?: { code: number; msg: string }) => {
    previewFrameRef.value?.contentWindow?.postMessage(
      { channel: PREVIEW_CHANNEL, version: 1, type: 'SCENE_DATA_RESULT', requestId, viewerSessionId, path, data, error },
      targetOrigin,
    )
  }
  if (!requestId || !access || !path || !access.datapointRefs.has(path)) {
    respond(undefined, { code: 26007, msg: '数据点未在当前场景 revision 中声明' })
    return
  }
  try {
    if (operation === 'get') {
      respond({ value: await readPageDatapoint(path) })
      return
    }
    if (operation === 'set') {
      respond(await writePageDatapoint(path, message.value))
      return
    }
    const socket = await ensureDataSocket()
    const subscriptions = dataSubscriptions.get(path) || new Set<string>()
    if (operation === 'sub') {
      if (subscriptions.has(viewerSessionId)) {
        respond({ subscribed: true })
        return
      }
      const firstSubscriber = !hasDatapointSubscribers(path)
      subscriptions.add(viewerSessionId)
      dataSubscriptions.set(path, subscriptions)
      try {
        if (firstSubscriber) await requestDataSocket(socket, 'datapoint:subscribe', path)
      } catch (error) {
        subscriptions.delete(viewerSessionId)
        if (!subscriptions.size) dataSubscriptions.delete(path)
        throw error
      }
      respond({ subscribed: true })
      return
    }
    if (operation === 'unsub') {
      subscriptions.delete(viewerSessionId)
      if (!subscriptions.size) {
        dataSubscriptions.delete(path)
        if (!hasDatapointSubscribers(path)) {
          await requestDataSocket(socket, 'datapoint:unsubscribe', path)
        }
      }
      respond({ subscribed: false })
      return
    }
    respond(undefined, { code: 26008, msg: '不支持的数据操作' })
  } catch (error) {
    const status = (error as { code?: number }).code
    const code = operation === 'set' ? (status === 403 || status === 11002 ? 26010 : 26011) : operation === 'sub' ? 26009 : 26008
    respond(undefined, { code, msg: error instanceof Error ? error.message : String(error) })
  }
}

function postRevisionChanged(value: unknown): void {
  if (!value || typeof value !== 'object') return
  const event = value as Record<string, unknown>
  if (event.projectId && event.projectId !== props.projectId) return
  if (typeof event.sceneId !== 'string' || !['2d', '3d'].includes(String(event.target))) return
  previewFrameRef.value?.contentWindow?.postMessage(
    {
      channel: PREVIEW_CHANNEL,
      version: 1,
      type: 'SCENE_REVISION_CHANGED',
      sceneId: event.sceneId,
      kind: event.target,
      revision: event.revision,
    },
    props.previewUrl ? new URL(props.previewUrl).origin : '*',
  )
}

function handleRevisionBroadcast(event: MessageEvent): void {
  postRevisionChanged(event.data)
}

function notifySceneRevision(event: unknown): void {
  postRevisionChanged(event)
  revisionChannel?.postMessage(event)
}

function handlePreviewFrameLoad(): void {
  frameLoaded.value = true
}

function releasePageDatapointSubscriptions(): void {
  // 页面卸载或预览重启时回收宿主侧订阅，避免后台继续轮询不可达的回调。
  const paths = [...pageDataSubscriptions.keys()]
  pageDataSubscriptions.clear()
  for (const path of paths) {
    if (!dataSubscriptions.get(path)?.size && dataSocket?.connected) {
      void requestDataSocket(dataSocket, 'datapoint:unsubscribe', path).catch(() => {})
    }
  }
}

defineExpose({ notifySceneRevision })

async function enterFullscreen(): Promise<void> {
  await panelRef.value?.requestFullscreen?.()
}
</script>

<template>
  <section ref="panelRef" class="preview-panel">
    <header class="preview-toolbar">
      <div class="preview-toolbar-left">
        <div class="device-picker">
          <button
            type="button"
            class="icon-button device-button"
            aria-label="切换预览设备"
            :aria-expanded="deviceMenuOpen"
            title="切换预览设备"
            @click="deviceMenuOpen = !deviceMenuOpen"
          >
            <IconLucideMonitorSmartphone />
          </button>
          <div v-if="deviceMenuOpen" class="device-menu" role="menu">
            <button
              v-for="device in devices"
              :key="device.id"
              type="button"
              role="menuitemradio"
              :aria-checked="selectedDevice === device.id"
              @click="selectDevice(device.id)"
            >
              <span>
                <strong>{{ device.label }}</strong>
                <small v-if="device.width">{{ device.width }} × {{ device.height }}</small>
                <small v-else>填满可用区域</small>
              </span>
              <IconLucideCheck v-if="selectedDevice === device.id" />
            </button>
          </div>
        </div>
        <span class="device-label">{{ currentDevice.label }}</span>
        <span class="process-status" :class="`status-${processState?.status || 'unknown'}`">
          <i />{{ processLabel }}
        </span>
      </div>

      <div class="preview-toolbar-actions">
        <button
          type="button"
          class="command-button"
          :disabled="!controlUrl || processBusy || Boolean(operation)"
          @click="toggleProcess"
        >
          <IconLucidePower />
          {{ primaryActionLabel }}
        </button>
        <button
          type="button"
          class="icon-button"
          title="重启预览服务"
          aria-label="重启预览服务"
          :disabled="!controlUrl || processBusy || Boolean(operation)"
          @click="runOperation('restart')"
        >
          <IconLucideRotateCcw :class="{ spinning: operation === 'restart' }" />
        </button>
        <span class="toolbar-divider" />
        <button
          type="button"
          class="icon-button"
          title="刷新预览"
          aria-label="刷新预览"
          :disabled="!previewUrl || !processRunning"
          @click="reloadPreview"
        >
          <IconLucideRefreshCw />
        </button>
        <button
          type="button"
          class="icon-button"
          title="在新窗口打开"
          aria-label="在新窗口打开"
          :disabled="!previewUrl"
          @click="openPreviewWindow"
        >
          <IconLucideExternalLink />
        </button>
        <button
          type="button"
          class="icon-button"
          title="全屏预览"
          aria-label="全屏预览"
          @click="enterFullscreen"
        >
          <IconLucideMaximize2 />
        </button>
      </div>
    </header>

    <div ref="canvasRef" class="preview-canvas" :class="`device-${selectedDevice}`">
      <div class="device-stage" :style="deviceStageStyle">
        <div class="preview-frame-shell" :style="deviceFrameStyle">
          <iframe
            v-if="previewUrl"
            ref="previewFrameRef"
            :key="frameKey"
            class="preview-frame"
            :class="{ loaded: frameLoaded }"
            :src="previewUrl"
            title="Vite 实时预览"
            @load="handlePreviewFrameLoad"
          />
          <div v-if="previewUrl && processRunning && !frameLoaded" class="preview-state subtle">
            <span class="loading-line" />
            <strong>正在连接 Vite 预览</strong>
          </div>
          <div v-else-if="!previewUrl" class="preview-state">
            <strong>预览服务尚未接入</strong>
          </div>
          <div v-else-if="!processRunning" class="preview-state stopped-state">
            <strong>预览服务已停止</strong>
            <p>{{ processError || processState?.message || '启动 Vite 后将在这里显示页面。' }}</p>
            <button
              type="button"
              :disabled="!controlUrl || processBusy || Boolean(operation)"
              @click="runOperation('start')"
            >
              启动预览
            </button>
          </div>
        </div>
      </div>
      <p v-if="processError && processRunning" class="preview-error" role="alert">
        {{ processError }}
      </p>
    </div>
  </section>
</template>

<style scoped>
.preview-panel {
  width: 100%;
  height: 100%;
  min-width: 0;
  min-height: 0;
  display: grid;
  grid-template-rows: 40px minmax(0, 1fr);
  color: #273142;
  background: #eef1f5;
}

.preview-toolbar,
.preview-toolbar-left,
.preview-toolbar-actions,
.command-button,
.icon-button,
.process-status {
  display: flex;
  align-items: center;
}

.preview-toolbar {
  position: relative;
  z-index: 6;
  justify-content: space-between;
  gap: 12px;
  padding: 0 8px 0 10px;
  border-bottom: 1px solid #d9dee7;
  background: #fff;
}

.preview-toolbar-left,
.preview-toolbar-actions {
  gap: 5px;
}

.device-picker {
  position: relative;
}

.icon-button,
.command-button {
  height: 28px;
  justify-content: center;
  border: 1px solid transparent;
  border-radius: 5px;
  color: #596579;
  background: transparent;
  cursor: pointer;
}

.icon-button {
  width: 30px;
  padding: 0;
}

.command-button {
  gap: 5px;
  padding: 0 8px;
  font-size: 11px;
}

.icon-button:hover:not(:disabled),
.command-button:hover:not(:disabled) {
  border-color: #ccd5e2;
  color: #245fc2;
  background: #f0f5ff;
}

.icon-button:disabled,
.command-button:disabled {
  opacity: 0.42;
  cursor: default;
}

.icon-button svg,
.command-button svg {
  width: 15px;
  height: 15px;
}

.device-label {
  color: #526074;
  font-size: 11px;
  font-weight: 600;
}

.device-menu {
  position: absolute;
  z-index: 20;
  top: 34px;
  left: 0;
  width: 210px;
  padding: 5px;
  border: 1px solid #d6dce6;
  border-radius: 6px;
  background: #fff;
  box-shadow: 0 12px 28px rgba(20, 31, 51, 0.16);
}

.device-menu button {
  width: 100%;
  min-height: 44px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 6px 8px;
  border: 0;
  border-radius: 4px;
  color: #273142;
  background: transparent;
  text-align: left;
  cursor: pointer;
}

.device-menu button:hover,
.device-menu button[aria-checked='true'] {
  background: #f0f4fa;
}

.device-menu button span {
  display: grid;
  gap: 2px;
}

.device-menu strong {
  font-size: 12px;
  font-weight: 600;
}

.device-menu small {
  color: #7a8596;
  font-size: 10px;
}

.device-menu svg {
  width: 14px;
  height: 14px;
  color: #2563eb;
}

.process-status {
  gap: 5px;
  margin-left: 5px;
  color: #6d7889;
  font-size: 10px;
}

.process-status i {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #98a2b3;
}

.process-status.status-running i {
  background: #22a559;
}

.process-status.status-starting i,
.process-status.status-stopping i {
  background: #e69925;
}

.process-status.status-error i {
  background: #dc3d45;
}

.toolbar-divider {
  width: 1px;
  height: 18px;
  margin: 0 2px;
  background: #dce1e9;
}

.preview-canvas {
  position: relative;
  min-width: 0;
  min-height: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: auto;
  background: #e8ebf0;
}

.preview-canvas.device-web {
  display: block;
  padding: 0;
}

.device-stage {
  position: relative;
  flex: 0 0 auto;
}

.device-web .device-stage {
  width: 100%;
  height: 100%;
}

.preview-frame-shell {
  position: relative;
  width: 100%;
  height: 100%;
  overflow: hidden;
  background: #fff;
  box-shadow: 0 4px 18px rgba(20, 31, 51, 0.13);
  transform-origin: top left;
}

.device-web .preview-frame-shell {
  box-shadow: none;
}

.preview-frame {
  width: 100%;
  height: 100%;
  display: block;
  border: 0;
  opacity: 0;
  background: #fff;
}

.preview-frame.loaded {
  opacity: 1;
}

.preview-state {
  position: absolute;
  z-index: 3;
  inset: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 28px;
  color: #697586;
  text-align: center;
  background: #f7f8fa;
}

.preview-state strong {
  color: #263248;
  font-size: 14px;
}

.preview-state p {
  max-width: 380px;
  margin: 7px 0 0;
  font-size: 11px;
  line-height: 1.55;
}

.preview-state button {
  height: 30px;
  margin-top: 14px;
  padding: 0 12px;
  border: 1px solid #b9c9e5;
  border-radius: 5px;
  color: #245fc2;
  background: #eef4ff;
  cursor: pointer;
}

.preview-state.subtle {
  background: #fff;
}

.loading-line {
  width: 112px;
  height: 3px;
  margin-bottom: 10px;
  overflow: hidden;
  border-radius: 2px;
  background: #e1e6ee;
}

.loading-line::after {
  content: '';
  display: block;
  width: 38px;
  height: 100%;
  background: #2563eb;
  animation: loading-line 1.1s ease-in-out infinite;
}

.preview-error {
  position: absolute;
  right: 12px;
  bottom: 12px;
  max-width: 420px;
  margin: 0;
  padding: 7px 9px;
  border: 1px solid #fecaca;
  border-radius: 5px;
  color: #b4232c;
  background: #fff;
  font-size: 10px;
  box-shadow: 0 7px 18px rgba(20, 31, 51, 0.12);
}

.spinning {
  animation: spin 0.9s linear infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

@keyframes loading-line {
  from {
    transform: translateX(-38px);
  }
  to {
    transform: translateX(112px);
  }
}

@media (max-width: 760px) {
  .device-label,
  .process-status {
    display: none;
  }

  .command-button {
    width: 30px;
    padding: 0;
    overflow: hidden;
    color: transparent;
    gap: 0;
  }

  .command-button svg {
    flex: 0 0 auto;
    color: #596579;
  }
}
</style>
