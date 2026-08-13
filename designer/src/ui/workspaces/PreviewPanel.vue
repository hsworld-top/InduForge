<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import IconLucideCheck from '~icons/lucide/check'
import IconLucideExternalLink from '~icons/lucide/external-link'
import IconLucideMaximize2 from '~icons/lucide/maximize-2'
import IconLucideMonitorSmartphone from '~icons/lucide/monitor-smartphone'
import IconLucidePower from '~icons/lucide/power'
import IconLucideRefreshCw from '~icons/lucide/refresh-cw'
import IconLucideRotateCcw from '~icons/lucide/rotate-ccw'
import IconLucideTerminalSquare from '~icons/lucide/square-terminal'
import { getApiErrorMessage } from '@/utils/request'
import { previewControlApi, type PreviewControlState } from './code/preview-control-api'

type DeviceMode = 'web' | 'tablet' | 'mobile'
type PreviewOperation = 'start' | 'stop' | 'restart'

interface DevicePreset {
  id: DeviceMode
  label: string
  width: number | null
  height: number | null
}

const props = defineProps<{
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

onMounted(() => {
  resizeObserver = new ResizeObserver(() => updateScale())
  if (canvasRef.value) resizeObserver.observe(canvasRef.value)
  void refreshProcessState()
})

onBeforeUnmount(() => {
  resizeObserver?.disconnect()
  clearPoll()
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
  frameLoaded.value = false
  frameKey.value += 1
}

function openPreviewWindow(): void {
  if (props.previewUrl) window.open(props.previewUrl, '_blank', 'noopener,noreferrer')
}

async function enterFullscreen(): Promise<void> {
  await panelRef.value?.requestFullscreen?.()
}

function openDevtools(): void {
  const target = previewFrameRef.value?.contentWindow
  if (!target || !props.previewUrl || !processRunning.value) return
  target.postMessage(
    {
      source: 'induforge-designer',
      type: 'PREVIEW_DEVTOOLS_COMMAND',
      command: 'toggle',
    },
    new URL(props.previewUrl).origin,
  )
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
            :key="frameKey"
            ref="previewFrameRef"
            class="preview-frame"
            :class="{ loaded: frameLoaded }"
            :src="previewUrl"
            title="Vite 实时预览"
            @load="frameLoaded = true"
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

    <footer class="preview-footer">
      <button type="button" :disabled="!previewUrl || !processRunning" @click="openDevtools">
        <IconLucideTerminalSquare />
        控制台
      </button>
    </footer>
  </section>
</template>

<style scoped>
.preview-panel {
  width: 100%;
  height: 100%;
  min-width: 0;
  min-height: 0;
  display: grid;
  grid-template-rows: 40px minmax(0, 1fr) 30px;
  color: #273142;
  background: #eef1f5;
}

.preview-toolbar,
.preview-toolbar-left,
.preview-toolbar-actions,
.command-button,
.icon-button,
.process-status,
.preview-footer button {
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
.command-button:disabled,
.preview-footer button:disabled {
  opacity: 0.42;
  cursor: default;
}

.icon-button svg,
.command-button svg,
.preview-footer svg {
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

.preview-footer {
  display: flex;
  align-items: center;
  border-top: 1px solid #d8dee7;
  background: #f7f8fa;
}

.preview-footer button {
  height: 29px;
  gap: 6px;
  padding: 0 12px;
  border: 0;
  border-bottom: 2px solid #2563eb;
  color: #344258;
  background: #fff;
  font-size: 11px;
  cursor: pointer;
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
