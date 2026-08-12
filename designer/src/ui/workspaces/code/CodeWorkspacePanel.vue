<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import IconLucidePlay from '~icons/lucide/play'
import IconLucideRedo2 from '~icons/lucide/redo-2'
import { getApiErrorMessage } from '@/utils/request'
import {
  codeWorkspaceApi,
  type CodeWorkspaceState,
  type CodeWorkspaceStatus,
} from './code-workspace-api'
import { contextPackApi } from './context-pack-api'

const props = defineProps<{
  projectId: string
  refreshKey: number
}>()

const emit = defineEmits<{
  'url-change': [url: string | null]
}>()

type WorkspaceAction = 'start' | 'stop' | 'rebuild'

const workspace = ref<CodeWorkspaceState | null>(null)
const loading = ref(true)
const pendingAction = ref<WorkspaceAction | null>(null)
const errorMessage = ref('')
const contextWarning = ref('')
const frameRevision = ref(0)
let requestSequence = 0
let pollTimer: ReturnType<typeof setTimeout> | null = null

const statusLabels: Record<CodeWorkspaceStatus, string> = {
  running: '运行中',
  stopped: '已停止',
  missing: '未创建',
  starting: '启动中',
  error: '异常',
}

const status = computed<CodeWorkspaceStatus>(() => workspace.value?.status ?? 'missing')
const statusLabel = computed(() => statusLabels[status.value])
const workspaceUrl = computed(() => {
  if (workspace.value?.status !== 'running' || !workspace.value.url) return null
  try {
    const url = new URL(workspace.value.url)
    return url.protocol === 'http:' || url.protocol === 'https:' ? url.toString() : null
  } catch {
    return null
  }
})
const isBusy = computed(() => loading.value || pendingAction.value !== null)
const frameKey = computed(() => `${workspaceUrl.value ?? 'empty'}:${frameRevision.value}`)
const onlineUsers = computed(() => workspace.value?.onlineUsers ?? [])
const onlineLabel = computed(() => {
  const others = onlineUsers.value.filter((user) => !user.isCurrent)
  if (others.length === 0) return '当前仅你在线'
  return `${others.length} 位其他成员在线`
})

watch(workspaceUrl, (url) => emit('url-change', url), { immediate: true })

watch(
  () => props.projectId,
  () => {
    frameRevision.value += 1
    void loadWorkspace(true, true)
  },
)

watch(
  () => props.refreshKey,
  (current, previous) => {
    if (current === previous) return
    frameRevision.value += 1
    void loadWorkspace()
  },
)

onMounted(() => void loadWorkspace(true, true))

onBeforeUnmount(() => {
  requestSequence += 1
  clearPollTimer()
  emit('url-change', null)
})

function clearPollTimer(): void {
  if (pollTimer) {
    clearTimeout(pollTimer)
    pollTimer = null
  }
}

function scheduleStatusPoll(): void {
  clearPollTimer()
  if (workspace.value?.status === 'starting') {
    pollTimer = setTimeout(() => void loadWorkspace(false), 1500)
  } else if (workspace.value?.status === 'running') {
    pollTimer = setTimeout(() => void loadWorkspace(false), 20000)
  }
}

async function loadWorkspace(showLoading = true, autoStart = false): Promise<void> {
  const sequence = ++requestSequence
  clearPollTimer()
  if (showLoading) loading.value = true
  errorMessage.value = ''

  try {
    let nextWorkspace = await codeWorkspaceApi.get(props.projectId)
    if (autoStart && (nextWorkspace.status === 'missing' || nextWorkspace.status === 'stopped')) {
      await refreshContextBeforeStart()
      nextWorkspace = await codeWorkspaceApi.start(props.projectId)
    }
    if (sequence !== requestSequence) return
    workspace.value = nextWorkspace
    scheduleStatusPoll()
  } catch (error) {
    if (sequence !== requestSequence) return
    errorMessage.value = getApiErrorMessage(error, '获取代码工作区状态失败')
  } finally {
    if (sequence === requestSequence) loading.value = false
  }
}

async function refreshContextBeforeStart(): Promise<void> {
  contextWarning.value = ''
  try {
    await contextPackApi.refresh(props.projectId)
  } catch (error) {
    // 上下文生成失败不能阻止客户进入已有源码工作区，但必须明确提示其刷新失败。
    contextWarning.value = getApiErrorMessage(error, '工程上下文刷新失败，请稍后手动重试')
  }
}

async function runAction(action: WorkspaceAction): Promise<void> {
  if (isBusy.value) return

  const sequence = ++requestSequence
  clearPollTimer()
  pendingAction.value = action
  errorMessage.value = ''

  try {
    const nextWorkspace = await codeWorkspaceApi[action](props.projectId)
    if (sequence !== requestSequence) return
    workspace.value = nextWorkspace
    frameRevision.value += 1
    scheduleStatusPoll()
  } catch (error) {
    if (sequence !== requestSequence) return
    const actionLabel = action === 'start' ? '启动' : action === 'stop' ? '停止' : '重建'
    errorMessage.value = getApiErrorMessage(error, `${actionLabel}代码工作区失败`)
  } finally {
    if (sequence === requestSequence) pendingAction.value = null
  }
}
</script>

<template>
  <section class="code-workspace-panel">
    <iframe
      v-if="workspaceUrl"
      :key="frameKey"
      class="code-workspace-frame"
      :src="workspaceUrl"
      title="页面工程"
    />

    <div
      v-if="workspaceUrl"
      class="online-indicator"
      :title="onlineUsers.map((user) => user.name).join('、')"
    >
      <span aria-hidden="true" />
      {{ onlineLabel }}
    </div>

    <div v-else class="workspace-state-card" :class="`status-${status}`">
      <div class="state-heading">
        <span class="status-dot" aria-hidden="true" />
        <span>CODE-SERVER · {{ statusLabel }}</span>
      </div>

      <template v-if="loading">
        <h3>正在读取工程工作区</h3>
        <p>正在向控制面查询共享 code-server 容器状态。</p>
      </template>

      <template v-else>
        <h3>{{ status === 'starting' ? '代码工作区正在启动' : '代码工作区尚未运行' }}</h3>
        <p v-if="status === 'missing'">首次启动会为当前工程创建共享 code-server 容器。</p>
        <p v-else-if="status === 'stopped'">容器已停止，启动后可继续使用原有源码与配置。</p>
        <p v-else-if="status === 'starting'">启动完成后将自动载入编辑器，无需手动刷新。</p>
        <p v-else-if="status === 'error'">容器状态异常，可尝试重建工作区。</p>
        <p v-else-if="status === 'running'">后端未返回有效的 HTTP 编辑器地址。</p>

        <dl v-if="workspace" class="workspace-meta">
          <div>
            <dt>容器</dt>
            <dd>{{ workspace.containerName || '—' }}</dd>
          </div>
          <div>
            <dt>端口</dt>
            <dd>{{ workspace.hostPort || '—' }}</dd>
          </div>
        </dl>

        <p v-if="errorMessage" class="error-message" role="alert">{{ errorMessage }}</p>
        <p v-if="contextWarning" class="warning-message" role="alert">{{ contextWarning }}</p>

        <div class="workspace-actions">
          <button
            v-if="status !== 'running' && status !== 'starting'"
            type="button"
            :disabled="isBusy"
            @click="runAction('start')"
          >
            <IconLucidePlay />
            {{ pendingAction === 'start' ? '启动中' : '启动工作区' }}
          </button>
          <button
            v-if="status === 'running'"
            type="button"
            :disabled="isBusy"
            @click="runAction('stop')"
          >
            <span class="stop-icon" aria-hidden="true" />
            {{ pendingAction === 'stop' ? '停止中' : '停止' }}
          </button>
          <button
            v-if="status !== 'starting'"
            type="button"
            class="secondary"
            :disabled="isBusy"
            @click="runAction('rebuild')"
          >
            <IconLucideRedo2 />
            {{ pendingAction === 'rebuild' ? '重建中' : '重建' }}
          </button>
          <button type="button" class="secondary" :disabled="isBusy" @click="loadWorkspace()">
            <IconLucideRedo2 />
            重新查询
          </button>
        </div>
      </template>
    </div>
  </section>
</template>

<style scoped>
.code-workspace-panel,
.code-workspace-frame {
  width: 100%;
  height: 100%;
}

.online-indicator {
  position: absolute;
  z-index: 3;
  top: 10px;
  right: 18px;
  height: 28px;
  display: inline-flex;
  align-items: center;
  gap: 7px;
  padding: 0 10px;
  border: 1px solid #4d5653;
  background: rgba(17, 27, 30, 0.92);
  color: #dce3df;
  font-size: 11px;
  pointer-events: none;
}

.online-indicator span {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: #52c878;
}

.code-workspace-panel {
  position: relative;
  background: #f8faf6;
}

.code-workspace-frame {
  display: block;
  border: 0;
  background: #1e1e1e;
}

.workspace-state-card {
  width: min(560px, 80%);
  position: absolute;
  top: 50%;
  left: 50%;
  padding: 34px 36px;
  border-left: 5px solid #d8ff36;
  background: #111b1e;
  color: #fff;
  transform: translate(-50%, -50%);
}

.workspace-state-card.status-error {
  border-left-color: #ff684f;
}

.state-heading {
  display: flex;
  align-items: center;
  gap: 9px;
  color: #aeb8b4;
  font:
    600 10px 'Bahnschrift',
    sans-serif;
  letter-spacing: 0.14em;
}

.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: #d8ff36;
  box-shadow: 0 0 0 3px rgba(216, 255, 54, 0.12);
}

.status-error .status-dot {
  background: #ff684f;
}

h3,
p {
  margin: 0;
}

h3 {
  margin: 12px 0 8px;
  font-size: 22px;
  font-weight: 620;
}

p {
  color: #bec7c3;
  font-size: 13px;
  line-height: 1.7;
}

.workspace-meta {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 1px;
  margin: 20px 0 0;
  background: #34403d;
}

.workspace-meta div {
  min-width: 0;
  padding: 10px 12px;
  background: #1a2527;
}

.workspace-meta dt {
  color: #7f8b87;
  font-size: 10px;
}

.workspace-meta dd {
  overflow: hidden;
  margin: 4px 0 0;
  color: #e7ece9;
  font:
    12px Consolas,
    monospace;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.error-message {
  margin-top: 16px;
  color: #ff9c8c;
}

.warning-message {
  margin: 12px 0 0;
  color: #ffd66d;
  font-size: 12px;
  line-height: 1.5;
}

.workspace-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 22px;
}

.workspace-actions button {
  height: 34px;
  display: inline-flex;
  align-items: center;
  gap: 7px;
  padding: 0 13px;
  border: 1px solid #d8ff36;
  background: #d8ff36;
  color: #172000;
  font-size: 12px;
  cursor: pointer;
}

.workspace-actions button.secondary {
  border-color: #697471;
  background: transparent;
  color: #e7ece9;
}

.workspace-actions button:hover:not(:disabled) {
  border-color: #fff;
}

.workspace-actions button:disabled {
  opacity: 0.45;
  cursor: not-allowed;
}

.workspace-actions svg {
  width: 14px;
  height: 14px;
}

.stop-icon {
  width: 10px;
  height: 10px;
  border: 1px solid currentColor;
}
</style>
