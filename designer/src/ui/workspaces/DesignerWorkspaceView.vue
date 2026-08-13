<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import IconLucideBox from '~icons/lucide/box'
import IconLucideCode2 from '~icons/lucide/code-2'
import IconLucideExternalLink from '~icons/lucide/external-link'
import IconLucideLayers3 from '~icons/lucide/layers-3'
import IconLucideRefreshCw from '~icons/lucide/refresh-cw'
import IconLucideRoute from '~icons/lucide/route'
import { getApiErrorMessage } from '@/utils/request'
import { getCurrentProjectId } from '@/runtime/wujie-context'
import CodeWorkspacePanel from './code/CodeWorkspacePanel.vue'
import { contextPackApi } from './code/context-pack-api'
import SceneContractPanel from './SceneContractPanel.vue'
import {
  buildHtWorkspaceUrl,
  resolveWorkspaceKey,
  type DesignerWorkspaceKey,
} from './workspace-links'

type WorkspaceItem = {
  key: DesignerWorkspaceKey
  label: string
  description: string
  icon: typeof IconLucideCode2
}

const route = useRoute()
const router = useRouter()
const refreshKey = ref(0)
const codeWorkspaceUrl = ref<string | null>(null)
const contextRefreshing = ref(false)
const contextMessage = ref('')
const sceneContractOpen = ref(false)

const workspaces: WorkspaceItem[] = [
  {
    key: 'code',
    label: '页面工程',
    description: '共享 code-server 工作空间',
    icon: IconLucideCode2,
  },
  {
    key: '2d',
    label: '2D 编辑器',
    description: 'HMI、大屏与工艺流程',
    icon: IconLucideRoute,
  },
  {
    key: '3d-scene',
    label: '3D 场景',
    description: '数字孪生场景编排',
    icon: IconLucideLayers3,
  },
  {
    key: '3d-model',
    label: '3D 模型',
    description: '模型与材质资源维护',
    icon: IconLucideBox,
  },
]

const activeWorkspace = ref(resolveWorkspaceKey(route.query.workspace))
const projectId = computed(() => {
  const fromRoute = route.meta.project?.id
  if (fromRoute) return String(fromRoute)

  const fromMicroApp = getCurrentProjectId()
  if (fromMicroApp) return fromMicroApp

  return ''
})
const currentWorkspace = computed<WorkspaceItem>(
  () => workspaces.find((item) => item.key === activeWorkspace.value) ?? workspaces[0]!,
)
const currentFrameUrl = computed(() => {
  if (!projectId.value) return null
  if (activeWorkspace.value === 'code') return codeWorkspaceUrl.value
  return buildHtWorkspaceUrl(import.meta.env.BASE_URL, projectId.value, activeWorkspace.value)
})
const frameKey = computed(
  () => `${activeWorkspace.value}:${currentFrameUrl.value ?? 'empty'}:${refreshKey.value}`,
)

watch(
  () => route.query.workspace,
  (value) => {
    activeWorkspace.value = resolveWorkspaceKey(value)
  },
)

onMounted(() => window.addEventListener('message', handleHtContextSync))
onUnmounted(() => window.removeEventListener('message', handleHtContextSync))

function handleHtContextSync(event: MessageEvent<unknown>): void {
  if (event.origin !== window.location.origin || !event.data || typeof event.data !== 'object') return
  const payload = event.data as { source?: string; type?: string; status?: string; message?: string }
  if (payload.source !== 'induforge-ht' || payload.type !== 'context-sync') return
  contextMessage.value =
    payload.status === 'updated' ? '场景已保存，工程上下文已同步' : payload.message || '场景已保存，工程上下文待刷新'
}

function selectWorkspace(key: DesignerWorkspaceKey): void {
  activeWorkspace.value = key
  if (key === 'code') sceneContractOpen.value = false
  void router.replace({
    query: {
      ...route.query,
      workspace: key,
    },
  })
}

function refreshWorkspace(): void {
  refreshKey.value += 1
}

function sceneKind(): '2d' | '3d' {
  return activeWorkspace.value === '2d' ? '2d' : '3d'
}

async function refreshProjectContext(): Promise<void> {
  if (!projectId.value || contextRefreshing.value) return
  contextRefreshing.value = true
  contextMessage.value = ''
  try {
    const result = await contextPackApi.refresh(projectId.value)
    contextMessage.value = `上下文已刷新：${result.pointCount} 个数据点、${result.roleCount} 个角色`
  } catch (error) {
    contextMessage.value = getApiErrorMessage(error, '工程上下文刷新失败')
  } finally {
    contextRefreshing.value = false
  }
}

function openWorkspaceInNewWindow(): void {
  if (currentFrameUrl.value) {
    window.open(currentFrameUrl.value, '_blank', 'noopener,noreferrer')
  }
}
</script>

<template>
  <main class="workspace-shell">
    <header class="workspace-header">
      <div class="brand-block">
        <span class="brand-mark" aria-hidden="true">IF</span>
        <div>
          <p class="eyebrow">INDUSTRIAL APPLICATION STUDIO</p>
          <h1>设计中心</h1>
        </div>
      </div>

      <div class="header-meta">
        <span class="shared-status"><i />多人共享 · 最后保存生效</span>
        <span v-if="projectId" class="project-id" :title="projectId">{{ projectId }}</span>
      </div>
    </header>

    <section class="workspace-body">
      <nav class="workspace-nav" aria-label="设计工作区">
        <div class="nav-heading">
          <span>WORKSPACES</span>
          <strong>04</strong>
        </div>
        <button
          v-for="(item, index) in workspaces"
          :key="item.key"
          type="button"
          class="workspace-nav-item"
          :class="{ active: activeWorkspace === item.key }"
          @click="selectWorkspace(item.key)"
        >
          <span class="workspace-index">0{{ index + 1 }}</span>
          <component :is="item.icon" class="workspace-icon" />
          <span class="workspace-copy">
            <strong>{{ item.label }}</strong>
            <small>{{ item.description }}</small>
          </span>
        </button>

        <div class="nav-note">
          <span>工程事实</span>
          <p>Vue 源码与 HT 场景文件统一保存在工程工作空间。</p>
        </div>
      </nav>

      <section class="workspace-stage">
        <div class="stage-toolbar">
          <div>
            <p>{{ currentWorkspace.description }}</p>
            <h2>{{ currentWorkspace.label }}</h2>
          </div>
          <div class="stage-actions">
            <button
              v-if="activeWorkspace !== 'code'"
              type="button"
              title="维护场景提供给 Vue 页面和运行时的公开接口"
              @click="sceneContractOpen = !sceneContractOpen"
            >
              {{ sceneContractOpen ? '关闭场景接口' : '场景公开接口' }}
            </button>
            <button
              type="button"
              :disabled="!projectId || contextRefreshing"
              title="刷新代码工作区中的工程上下文包"
              @click="refreshProjectContext"
            >
              <IconLucideRefreshCw />
              {{ contextRefreshing ? '刷新上下文中' : '刷新工程上下文' }}
            </button>
            <button type="button" title="刷新当前工作区" @click="refreshWorkspace">
              <IconLucideRefreshCw />
              刷新
            </button>
            <button
              type="button"
              title="在新窗口打开"
              :disabled="!currentFrameUrl"
              @click="openWorkspaceInNewWindow"
            >
              <IconLucideExternalLink />
              新窗口
            </button>
          </div>
        </div>

        <div class="stage-content">
          <p v-if="contextMessage" class="context-message">{{ contextMessage }}</p>
          <CodeWorkspacePanel
            v-if="projectId && activeWorkspace === 'code'"
            :project-id="projectId"
            :refresh-key="refreshKey"
            @url-change="codeWorkspaceUrl = $event"
          />

          <iframe
            v-else-if="currentFrameUrl"
            :key="frameKey"
            class="workspace-frame"
            :src="currentFrameUrl"
            :title="currentWorkspace.label"
          />

          <SceneContractPanel
            v-if="projectId && activeWorkspace !== 'code' && sceneContractOpen"
            :project-id="projectId"
            :kind="sceneKind()"
            @close="sceneContractOpen = false"
            @synced="contextMessage = $event"
          />

          <div v-else-if="!projectId" class="empty-state danger-state">
            <span>PROJECT CONTEXT MISSING</span>
            <h3>未获取到工程上下文</h3>
            <p>请从工程管理入口重新进入 Designer。</p>
          </div>
        </div>
      </section>
    </section>
  </main>
</template>

<style scoped>
.workspace-shell {
  --ink: #111b1e;
  --paper: #edf0eb;
  --panel: #f8faf6;
  --line: #c8cec6;
  --signal: #d8ff36;
  --signal-ink: #172000;
  min-width: 1180px;
  min-height: 720px;
  height: 100vh;
  color: var(--ink);
  background:
    linear-gradient(rgba(17, 27, 30, 0.035) 1px, transparent 1px),
    linear-gradient(90deg, rgba(17, 27, 30, 0.035) 1px, transparent 1px), var(--paper);
  background-size: 24px 24px;
  overflow: hidden;
}

.workspace-header {
  height: 76px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 24px;
  border-bottom: 1px solid var(--line);
  background: rgba(248, 250, 246, 0.94);
}

.brand-block,
.header-meta,
.stage-actions {
  display: flex;
  align-items: center;
}

.brand-block {
  gap: 14px;
}

.brand-mark {
  width: 42px;
  height: 42px;
  display: grid;
  place-items: center;
  border: 1px solid var(--ink);
  background: var(--ink);
  color: var(--signal);
  font-family: 'Bahnschrift', 'DIN Alternate', sans-serif;
  font-size: 15px;
  font-weight: 800;
  letter-spacing: -0.04em;
}

.eyebrow,
.nav-heading span,
.empty-state > span {
  margin: 0;
  font-family: 'Bahnschrift', 'DIN Alternate', sans-serif;
  font-size: 10px;
  letter-spacing: 0.18em;
  color: #66716f;
}

h1,
h2,
h3,
p {
  margin: 0;
}

h1 {
  margin-top: 2px;
  font-size: 22px;
  font-weight: 650;
  letter-spacing: 0.02em;
}

.header-meta {
  gap: 12px;
  font-size: 12px;
}

.shared-status,
.project-id {
  height: 30px;
  display: inline-flex;
  align-items: center;
  padding: 0 10px;
  border: 1px solid var(--line);
  background: #fff;
}

.shared-status i {
  width: 7px;
  height: 7px;
  margin-right: 8px;
  border-radius: 50%;
  background: #33a457;
  box-shadow: 0 0 0 3px rgba(51, 164, 87, 0.14);
}

.project-id {
  max-width: 220px;
  overflow: hidden;
  color: #5c6765;
  font-family: Consolas, monospace;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.workspace-body {
  height: calc(100vh - 76px);
  display: grid;
  grid-template-columns: 292px minmax(0, 1fr);
}

.workspace-nav {
  position: relative;
  padding: 22px 16px;
  border-right: 1px solid var(--line);
  background: rgba(238, 242, 235, 0.96);
}

.nav-heading {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  padding: 0 8px 18px;
}

.nav-heading strong {
  color: #7d8784;
  font:
    500 12px Consolas,
    monospace;
}

.workspace-nav-item {
  width: 100%;
  min-height: 76px;
  display: grid;
  grid-template-columns: 26px 28px 1fr;
  align-items: center;
  gap: 10px;
  margin-bottom: 8px;
  padding: 10px 12px;
  border: 1px solid transparent;
  color: inherit;
  background: transparent;
  text-align: left;
  cursor: pointer;
  transition:
    transform 150ms ease,
    border-color 150ms ease,
    background 150ms ease;
}

.workspace-nav-item:hover {
  transform: translateX(3px);
  border-color: #abb4ae;
  background: rgba(255, 255, 255, 0.58);
}

.workspace-nav-item.active {
  border-color: var(--ink);
  background: var(--ink);
  color: #fff;
  box-shadow: 5px 5px 0 var(--signal);
}

.workspace-index {
  align-self: start;
  padding-top: 3px;
  color: #818b88;
  font:
    500 10px Consolas,
    monospace;
}

.active .workspace-index {
  color: var(--signal);
}

.workspace-icon {
  width: 20px;
  height: 20px;
}

.workspace-copy {
  min-width: 0;
  display: grid;
  gap: 5px;
}

.workspace-copy strong {
  font-size: 14px;
  font-weight: 650;
}

.workspace-copy small {
  overflow: hidden;
  color: #74807c;
  font-size: 11px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.active .workspace-copy small {
  color: #b8c0bc;
}

.nav-note {
  position: absolute;
  right: 22px;
  bottom: 22px;
  left: 22px;
  padding-top: 14px;
  border-top: 1px solid var(--line);
}

.nav-note span {
  color: #66716f;
  font:
    600 10px 'Bahnschrift',
    sans-serif;
  letter-spacing: 0.12em;
}

.nav-note p {
  margin-top: 7px;
  color: #66716f;
  font-size: 11px;
  line-height: 1.55;
}

.workspace-stage {
  min-width: 0;
  display: grid;
  grid-template-rows: 64px minmax(0, 1fr);
  padding: 14px 16px 16px;
}

.stage-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 4px;
}

.stage-toolbar p {
  color: #6d7774;
  font-size: 11px;
}

.stage-toolbar h2 {
  margin-top: 3px;
  font-size: 20px;
  font-weight: 650;
}

.stage-actions {
  gap: 8px;
}

.stage-actions button {
  height: 32px;
  display: inline-flex;
  align-items: center;
  gap: 7px;
  padding: 0 11px;
  border: 1px solid #aeb6b1;
  color: var(--ink);
  background: #fff;
  font-size: 12px;
  cursor: pointer;
}

.stage-actions button:hover:not(:disabled) {
  border-color: var(--ink);
  background: var(--signal);
}

.stage-actions button:disabled {
  opacity: 0.42;
  cursor: not-allowed;
}

.stage-actions svg {
  width: 14px;
  height: 14px;
}

.stage-content {
  min-height: 0;
  position: relative;
  border: 1px solid var(--ink);
  background: var(--panel);
  box-shadow: 8px 8px 0 rgba(17, 27, 30, 0.1);
  overflow: hidden;
}

.stage-content::before {
  content: '';
  position: absolute;
  z-index: 2;
  top: 0;
  right: 0;
  width: 18px;
  height: 18px;
  background: linear-gradient(135deg, transparent 48%, var(--signal) 49%);
  pointer-events: none;
}

.context-message {
  position: absolute;
  z-index: 4;
  right: 18px;
  bottom: 14px;
  max-width: min(580px, calc(100% - 36px));
  margin: 0;
  padding: 8px 10px;
  border: 1px solid #aeb6b1;
  color: #34403c;
  background: rgba(255, 255, 255, 0.94);
  font-size: 11px;
}

.workspace-frame {
  width: 100%;
  height: 100%;
  display: block;
  border: 0;
  background: #fff;
}

.empty-state {
  width: min(520px, 80%);
  position: absolute;
  top: 50%;
  left: 50%;
  padding: 34px 36px;
  border-left: 5px solid var(--signal);
  background: #111b1e;
  color: #fff;
  transform: translate(-50%, -50%);
}

.empty-state h3 {
  margin: 10px 0 8px;
  font-size: 22px;
  font-weight: 620;
}

.empty-state p {
  color: #bec7c3;
  font-size: 13px;
  line-height: 1.7;
}

.empty-state small {
  display: block;
  margin-top: 18px;
  color: var(--signal);
  font-family: Consolas, monospace;
  font-size: 11px;
}

.danger-state {
  border-left-color: #ff684f;
}

@media (max-width: 1300px) {
  .workspace-body {
    grid-template-columns: 252px minmax(0, 1fr);
  }

  .workspace-copy small,
  .nav-note {
    display: none;
  }
}
</style>
