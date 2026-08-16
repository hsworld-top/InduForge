<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import IconLucideBox from '~icons/lucide/box'
import IconLucideBraces from '~icons/lucide/braces'
import IconLucideCode2 from '~icons/lucide/code-2'
import IconLucideFileCode2 from '~icons/lucide/file-code-2'
import IconLucideMonitorPlay from '~icons/lucide/monitor-play'
import IconLucideRefreshCw from '~icons/lucide/refresh-cw'
import IconLucideRoute from '~icons/lucide/route'
import { useRoute } from 'vue-router'
import {
  getCurrentProjectId,
  requestWorkspaceOpen,
  type WorkspaceOpenTarget,
} from '@/runtime/wujie-context'
import { getApiErrorMessage } from '@/utils/request'
import { getEditorUiStore } from '@/stores/editor-ui-store'
import type { CodeWorkspaceState } from './code/code-workspace-api'
import { contextPackApi } from './code/context-pack-api'
import {
  previewControlApi,
  type WorkspaceInitializationState,
  type WorkspaceTemplate,
} from './code/preview-control-api'
import { resolveWorkspaceState } from './code/workspace-runtime'
import PreviewPanel from './PreviewPanel.vue'
import SceneArtifactsPanel from './SceneArtifactsPanel.vue'
import { sceneContractApi, type SceneContract } from './scene-contract-api'
import {
  clampExpandedAiPaneWidth,
  getCollapsedAiPaneWidth,
  MIN_AI_PANE_WIDTH,
  resolveAiPaneLayout,
  shouldUseCompactLayout,
  SPLIT_HANDLE_WIDTH,
  WORKBENCH_MENU_WIDTH,
} from './workspace-layout'

type VisiblePane = 'ai' | 'workbench'
type ContextStatus = 'syncing' | 'synced' | 'error'
type WorkbenchView = 'page' | '2d' | '3d' | 'editor'

interface PersistedWorkspaceLayout {
  version: 1
  aiPaneWidth: number
  lastExpandedAiPaneWidth: number
  workbenchCollapsed: boolean
}

const route = useRoute()
const editorUi = getEditorUiStore()
const stageRef = ref<HTMLElement | null>(null)
const aiFrameRef = ref<HTMLIFrameElement | null>(null)
const workspace = ref<CodeWorkspaceState | null>(null)
const workspaceLoading = ref(true)
const workspaceError = ref('')
const projectWorkspace = ref<WorkspaceInitializationState | null>(null)
const projectWorkspaceLoading = ref(true)
const projectWorkspaceError = ref('')
const workspaceTemplates = ref<WorkspaceTemplate[]>([])
const selectedTemplateId = ref('')
const templateInitializing = ref(false)
const contextStatus = ref<ContextStatus>('syncing')
const contextError = ref('')
const contextVersion = ref('—')
const contextUpdatedAt = ref('')
const pointCount = ref<number | null>(null)
const scene2dCount = ref<number | null>(null)
const scene3dCount = ref<number | null>(null)
const sceneContracts = ref<SceneContract[]>([])
const compactLayout = ref(false)
const visiblePane = ref<VisiblePane>('ai')
const activeWorkbenchView = ref<WorkbenchView>('page')
const aiPaneWidth = ref(MIN_AI_PANE_WIDTH)
const lastExpandedAiPaneWidth = ref(MIN_AI_PANE_WIDTH)
const workbenchCollapsed = ref(false)
const stageWidth = ref(0)
const aiFrameLoaded = ref(false)
const codeFrameLoaded = ref(false)
const toolError = ref('')
let resizeObserver: ResizeObserver | null = null
let workspacePollTimer: ReturnType<typeof setTimeout> | null = null
let workspaceRequestSequence = 0

const projectId = computed(() => {
  const fromRoute = route.meta.project?.id
  return fromRoute ? String(fromRoute) : getCurrentProjectId() || ''
})
const aiUrl = computed(() =>
  workspace.value?.status === 'running' ? workspace.value.services.ai.url : null,
)
const aiFrameUrl = computed(() => {
  if (!aiUrl.value || !projectId.value) return null
  const url = new URL(aiUrl.value)
  url.searchParams.set('induforgeProjectId', projectId.value)
  return url.toString()
})
const aiOrigin = computed(() => (aiFrameUrl.value ? new URL(aiFrameUrl.value).origin : null))
const codeUrl = computed(() =>
  workspace.value?.status === 'running' ? workspace.value.services.code.url : null,
)
const previewUrl = computed(() =>
  workspace.value?.status === 'running' ? workspace.value.services.preview.url : null,
)
const previewControlUrl = computed(() =>
  workspace.value?.status === 'running' ? workspace.value.services.previewControl.url : null,
)
const workspaceInitialized = computed(() => projectWorkspace.value?.status === 'initialized')
const workspaceSetupMessage = computed(() => {
  if (projectWorkspaceError.value) return projectWorkspaceError.value
  if (projectWorkspace.value?.status === 'error') {
    return projectWorkspace.value.message || '工程工作区当前不可初始化'
  }
  return projectWorkspace.value?.message || ''
})
const scene2dContracts = computed(() =>
  sceneContracts.value.filter((contract) => contract.kind === '2d'),
)
const scene3dContracts = computed(() =>
  sceneContracts.value.filter((contract) => contract.kind === '3d'),
)
const splitStyle = computed(() => ({
  gridTemplateColumns: `${aiPaneWidth.value}px ${SPLIT_HANDLE_WIDTH}px minmax(${WORKBENCH_MENU_WIDTH}px, 1fr)`,
  '--workbench-menu-width': `${WORKBENCH_MENU_WIDTH}px`,
}))
const aiPaneMaximum = computed(() =>
  stageWidth.value > 0 ? getCollapsedAiPaneWidth(stageWidth.value) : MIN_AI_PANE_WIDTH,
)
const contextStatusLabel = computed(() => {
  if (contextStatus.value === 'syncing') return '同步中'
  if (contextStatus.value === 'error') return '同步失败'
  return contextUpdatedAt.value ? `已同步 ${formatTime(contextUpdatedAt.value)}` : '已同步'
})
const workspaceStateLabel = computed(() => {
  if (workspaceLoading.value) return '正在读取开发环境'
  if (workspaceError.value) return '开发环境连接失败'
  if (workspace.value?.status === 'starting') return '正在启动开发环境'
  if (workspace.value?.status === 'missing' || workspace.value?.status === 'stopped') {
    return '开发环境尚未运行'
  }
  if (workspace.value?.status === 'error') return '开发环境异常'
  return ''
})

onMounted(() => {
  restoreAiPaneWidth()
  resizeObserver = new ResizeObserver(([entry]) => {
    const width = entry?.contentRect.width ?? 0
    stageWidth.value = width
    compactLayout.value = shouldUseCompactLayout(width)
    if (!compactLayout.value) {
      aiPaneWidth.value = workbenchCollapsed.value
        ? getCollapsedAiPaneWidth(width)
        : clampExpandedAiPaneWidth(aiPaneWidth.value, width)
      if (!workbenchCollapsed.value) lastExpandedAiPaneWidth.value = aiPaneWidth.value
    }
  })
  window.addEventListener('message', handlePiMessage)
  void Promise.allSettled([loadWorkspace(true), runContextRefresh()])
})

onBeforeUnmount(() => {
  resizeObserver?.disconnect()
  clearWorkspacePoll()
  workspaceRequestSequence += 1
  window.removeEventListener('message', handlePiMessage)
})

watch([editorUi.theme, editorUi.locale, aiFrameUrl, projectId], () => postPiContext(), {
  flush: 'post',
})

watch(stageRef, (stage) => {
  resizeObserver?.disconnect()
  if (stage) resizeObserver?.observe(stage)
})

function postPiContext(): void {
  const target = aiFrameRef.value?.contentWindow
  if (!target || !aiFrameLoaded.value || !aiOrigin.value || !projectId.value) return
  target.postMessage(
    {
      type: 'INDUFORGE_PI_CONTEXT',
      version: 1,
      projectId: projectId.value,
      workspaceRoot: '/workspace',
      locale: editorUi.locale.value,
      theme: editorUi.theme.value,
    },
    aiOrigin.value,
  )
}

function handleAiFrameLoad(): void {
  aiFrameLoaded.value = true
  postPiContext()
}

function handlePiMessage(event: MessageEvent): void {
  const target = aiFrameRef.value?.contentWindow
  if (!target || event.source !== target || event.origin !== aiOrigin.value) return
  if (!event.data || typeof event.data !== 'object') return
  const message = event.data as Record<string, unknown>
  if (
    message.type !== 'INDUFORGE_PI_READY' ||
    message.version !== 1 ||
    message.projectId !== projectId.value
  ) {
    return
  }
  postPiContext()
}

function formatTime(value: string): string {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return new Intl.DateTimeFormat('zh-CN', {
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
  }).format(date)
}

function restoreAiPaneWidth(): void {
  if (!projectId.value) return
  try {
    const stored = JSON.parse(
      localStorage.getItem(`designer:workspace-layout:${projectId.value}`) || 'null',
    ) as PersistedWorkspaceLayout | null
    if (stored?.version !== 1) return
    aiPaneWidth.value = Math.max(stored.aiPaneWidth, MIN_AI_PANE_WIDTH)
    lastExpandedAiPaneWidth.value = Math.max(stored.lastExpandedAiPaneWidth, MIN_AI_PANE_WIDTH)
    workbenchCollapsed.value = stored.workbenchCollapsed === true
  } catch {
    // 损坏的本地布局状态直接忽略，避免阻止工作台进入。
  }
}

function persistAiPaneWidth(): void {
  if (projectId.value) {
    localStorage.setItem(
      `designer:workspace-layout:${projectId.value}`,
      JSON.stringify({
        version: 1,
        aiPaneWidth: Math.round(aiPaneWidth.value),
        lastExpandedAiPaneWidth: Math.round(lastExpandedAiPaneWidth.value),
        workbenchCollapsed: workbenchCollapsed.value,
      } satisfies PersistedWorkspaceLayout),
    )
  }
}

function getStageWidth(): number {
  return stageWidth.value || stageRef.value?.clientWidth || 0
}

function beginResize(event: PointerEvent): void {
  if (compactLayout.value || !stageRef.value) return
  const startX = event.clientX
  const startWidth = aiPaneWidth.value
  const containerWidth = getStageWidth()
  const startedCollapsed = workbenchCollapsed.value
  const target = event.currentTarget as HTMLElement
  target.setPointerCapture?.(event.pointerId)

  const move = (moveEvent: PointerEvent) => {
    const delta = moveEvent.clientX - startX
    if (startedCollapsed) {
      if (delta >= -8) return
      workbenchCollapsed.value = false
      aiPaneWidth.value = clampExpandedAiPaneWidth(
        lastExpandedAiPaneWidth.value + delta + 8,
        containerWidth,
      )
      return
    }

    const layout = resolveAiPaneLayout(startWidth + delta, containerWidth)
    if (layout.workbenchCollapsed && !workbenchCollapsed.value) {
      lastExpandedAiPaneWidth.value = startWidth
    }
    workbenchCollapsed.value = layout.workbenchCollapsed
    aiPaneWidth.value = layout.width
  }
  const end = () => {
    if (!workbenchCollapsed.value) lastExpandedAiPaneWidth.value = aiPaneWidth.value
    persistAiPaneWidth()
    target.removeEventListener('pointermove', move)
    target.removeEventListener('pointerup', end)
    target.removeEventListener('pointercancel', end)
  }
  target.addEventListener('pointermove', move)
  target.addEventListener('pointerup', end)
  target.addEventListener('pointercancel', end)
}

function resizeWithKeyboard(event: KeyboardEvent): void {
  if (!stageRef.value || !['ArrowLeft', 'ArrowRight'].includes(event.key)) return
  event.preventDefault()
  if (workbenchCollapsed.value) {
    if (event.key === 'ArrowLeft') restoreWorkbench()
    return
  }
  const delta = event.key === 'ArrowLeft' ? -24 : 24
  const layout = resolveAiPaneLayout(aiPaneWidth.value + delta, getStageWidth())
  if (layout.workbenchCollapsed) lastExpandedAiPaneWidth.value = aiPaneWidth.value
  workbenchCollapsed.value = layout.workbenchCollapsed
  aiPaneWidth.value = layout.width
  if (!workbenchCollapsed.value) lastExpandedAiPaneWidth.value = aiPaneWidth.value
  persistAiPaneWidth()
}

function restoreWorkbench(): void {
  if (!stageRef.value) return
  workbenchCollapsed.value = false
  aiPaneWidth.value = clampExpandedAiPaneWidth(lastExpandedAiPaneWidth.value, getStageWidth())
  lastExpandedAiPaneWidth.value = aiPaneWidth.value
  persistAiPaneWidth()
}

function selectWorkbenchView(view: WorkbenchView): void {
  activeWorkbenchView.value = view
  if (workbenchCollapsed.value && !compactLayout.value) restoreWorkbench()
}

function clearWorkspacePoll(): void {
  if (workspacePollTimer) clearTimeout(workspacePollTimer)
  workspacePollTimer = null
}

function scheduleWorkspacePoll(): void {
  clearWorkspacePoll()
  if (workspace.value?.status === 'starting') {
    workspacePollTimer = setTimeout(() => void loadWorkspace(false), 1500)
  }
}

async function loadWorkspace(autoStart: boolean): Promise<void> {
  if (!projectId.value) return
  const sequence = ++workspaceRequestSequence
  clearWorkspacePoll()
  workspaceLoading.value = true
  workspaceError.value = ''
  try {
    const nextWorkspace = await resolveWorkspaceState(projectId.value, autoStart)
    if (sequence !== workspaceRequestSequence) return
    if (workspace.value?.services.ai.url !== nextWorkspace.services.ai.url)
      aiFrameLoaded.value = false
    if (workspace.value?.services.code.url !== nextWorkspace.services.code.url) {
      codeFrameLoaded.value = false
    }
    workspace.value = nextWorkspace
    await loadProjectWorkspace(nextWorkspace.services.previewControl.url)
    scheduleWorkspacePoll()
  } catch (error) {
    if (sequence !== workspaceRequestSequence) return
    workspaceError.value = getApiErrorMessage(error, '读取工程开发环境失败')
    projectWorkspaceLoading.value = false
  } finally {
    if (sequence === workspaceRequestSequence) workspaceLoading.value = false
  }
}

async function loadProjectWorkspace(controlUrl: string | null): Promise<void> {
  projectWorkspaceLoading.value = true
  projectWorkspaceError.value = ''
  if (!controlUrl) {
    projectWorkspace.value = null
    projectWorkspaceError.value = '工程初始化服务尚未接入'
    projectWorkspaceLoading.value = false
    return
  }
  try {
    const nextState = await previewControlApi.workspaceStatus(controlUrl)
    projectWorkspace.value = nextState
    if (nextState.status === 'uninitialized') {
      const catalog = await previewControlApi.templates(controlUrl)
      workspaceTemplates.value = catalog.templates
      if (!catalog.templates.some((template) => template.id === selectedTemplateId.value)) {
        selectedTemplateId.value = catalog.templates[0]?.id || ''
      }
    }
  } catch (error) {
    projectWorkspaceError.value = getApiErrorMessage(error, '读取工程初始化状态失败')
  } finally {
    projectWorkspaceLoading.value = false
  }
}

async function initializeProjectWorkspace(): Promise<void> {
  if (!previewControlUrl.value || !selectedTemplateId.value || templateInitializing.value) return
  templateInitializing.value = true
  projectWorkspaceError.value = ''
  projectWorkspace.value = {
    status: 'initializing',
    templateId: null,
    initializedAt: null,
    message: null,
  }
  try {
    const result = await previewControlApi.initialize(
      previewControlUrl.value,
      selectedTemplateId.value,
    )
    projectWorkspace.value = result.workspace
    aiFrameLoaded.value = false
    codeFrameLoaded.value = false
  } catch (error) {
    projectWorkspaceError.value = getApiErrorMessage(error, '工程初始化失败')
    await loadProjectWorkspace(previewControlUrl.value)
  } finally {
    templateInitializing.value = false
  }
}

async function runContextRefresh(): Promise<void> {
  if (!projectId.value) return
  contextStatus.value = 'syncing'
  contextError.value = ''
  const [contextResult, sceneResult] = await Promise.allSettled([
    contextPackApi.refresh(projectId.value),
    sceneContractApi.list(projectId.value),
  ])

  const errors: string[] = []
  if (contextResult.status === 'fulfilled') {
    contextVersion.value = contextResult.value.contractVersion || '—'
    contextUpdatedAt.value = contextResult.value.updatedAt
    pointCount.value = contextResult.value.pointCount
  } else {
    errors.push(getApiErrorMessage(contextResult.reason, '上下文刷新失败'))
  }

  if (sceneResult.status === 'fulfilled') {
    sceneContracts.value = sceneResult.value.contracts
    scene2dCount.value = scene2dContracts.value.length
    scene3dCount.value = scene3dContracts.value.length
  } else {
    errors.push(getApiErrorMessage(sceneResult.reason, '场景摘要读取失败'))
  }

  contextError.value = errors.join('；')
  contextStatus.value = errors.length > 0 ? 'error' : 'synced'
}

async function retryContext(): Promise<void> {
  if (contextStatus.value === 'syncing') return
  await runContextRefresh()
}

function openWorkspace(target: WorkspaceOpenTarget): void {
  toolError.value = ''
  if (!requestWorkspaceOpen(target)) {
    toolError.value = '当前未连接工程工具宿主'
    window.setTimeout(() => {
      toolError.value = ''
    }, 2400)
  }
}

async function retryWorkspace(): Promise<void> {
  await nextTick()
  await loadWorkspace(true)
}
</script>

<template>
  <main class="ai-workbench" :class="{ 'setup-mode': !workspaceInitialized }">
    <section
      v-if="workspaceLoading || projectWorkspaceLoading"
      class="workspace-setup workspace-setup-loading"
    >
      <span class="loading-line" />
      <strong>正在准备工程工作空间</strong>
      <p>正在读取可用模板和初始化状态。</p>
    </section>

    <section v-else-if="!workspaceInitialized" class="workspace-setup">
      <div class="workspace-setup-content">
        <span class="workspace-setup-kicker">PROJECT TEMPLATE</span>
        <h1>选择工程技术栈</h1>
        <p class="workspace-setup-description">
          工程只在首次创建时选择模板，初始化完成后由 AI 和编辑器共同维护源码。
        </p>

        <div
          v-if="projectWorkspace?.status === 'uninitialized' && workspaceTemplates.length > 0"
          class="template-grid"
          role="radiogroup"
          aria-label="工程模板"
        >
          <button
            v-for="template in workspaceTemplates"
            :key="template.id"
            type="button"
            class="template-option"
            :class="{ selected: selectedTemplateId === template.id }"
            role="radio"
            :aria-checked="selectedTemplateId === template.id"
            @click="selectedTemplateId = template.id"
          >
            <span class="template-icon" :class="`framework-${template.framework}`">
              <IconLucideBraces v-if="template.framework === 'react'" />
              <IconLucideFileCode2 v-else />
            </span>
            <span class="template-copy">
              <strong>{{ template.name }}</strong>
              <small>{{ template.description }}</small>
            </span>
            <span class="template-language">{{
              template.language === 'typescript' ? 'TS' : 'JS'
            }}</span>
          </button>
        </div>

        <div v-else class="workspace-setup-state">
          <span v-if="projectWorkspace?.status === 'initializing'" class="loading-line" />
          <strong>
            {{
              projectWorkspace?.status === 'initializing'
                ? '正在创建工程'
                : workspaceError
                  ? '开发环境连接失败'
                  : '工程工作区不可初始化'
            }}
          </strong>
          <p>{{ workspaceSetupMessage || workspaceError || '没有可用的工程模板。' }}</p>
        </div>

        <p v-if="workspaceSetupMessage" class="workspace-setup-error" role="alert">
          {{ workspaceSetupMessage }}
        </p>
        <div class="workspace-setup-actions">
          <button
            v-if="projectWorkspace?.status === 'uninitialized' && workspaceTemplates.length > 0"
            type="button"
            class="initialize-button"
            :disabled="!selectedTemplateId || templateInitializing"
            @click="initializeProjectWorkspace"
          >
            {{ templateInitializing ? '正在初始化' : '创建工程' }}
          </button>
          <button
            v-else-if="projectWorkspace?.status !== 'initializing'"
            type="button"
            class="retry-setup-button"
            @click="retryWorkspace"
          >
            重新检查
          </button>
        </div>
      </div>
    </section>

    <section
      v-else
      ref="stageRef"
      class="workbench-stage"
      :class="{ compact: compactLayout }"
      :style="compactLayout ? undefined : splitStyle"
    >
      <div class="left-workspace">
        <section
          class="service-pane ai-pane"
          :class="{ hidden: compactLayout && visiblePane !== 'ai' }"
        >
          <iframe
            v-if="aiFrameUrl"
            ref="aiFrameRef"
            class="service-frame"
            :class="{ loaded: aiFrameLoaded }"
            :src="aiFrameUrl"
            title="Pi Web AI 对话"
            @load="handleAiFrameLoad"
          />
          <div v-if="aiFrameUrl && !aiFrameLoaded" class="pane-state subtle-state">
            <span class="loading-line" />
            <strong>正在连接 Pi Web</strong>
          </div>
          <div v-else-if="!aiUrl" class="pane-state">
            <span class="state-kicker">PI WEB</span>
            <strong>{{ workspaceStateLabel || 'AI 服务尚未接入' }}</strong>
            <p>{{ workspaceError || '控制面返回 Pi Web 受控地址后，对话将在此处加载。' }}</p>
            <button
              v-if="workspaceError || workspace?.status === 'error'"
              type="button"
              @click="retryWorkspace"
            >
              重新连接
            </button>
          </div>
        </section>
      </div>

      <nav v-if="compactLayout" class="compact-tabs" aria-label="工作台视图">
        <button type="button" :class="{ active: visiblePane === 'ai' }" @click="visiblePane = 'ai'">
          AI
        </button>
        <button
          type="button"
          :class="{ active: visiblePane === 'workbench' }"
          @click="visiblePane = 'workbench'"
        >
          工作台
        </button>
      </nav>

      <div
        v-if="!compactLayout"
        class="split-handle"
        role="separator"
        aria-label="调整 AI 与工作台区域宽度"
        aria-orientation="vertical"
        :aria-valuemin="MIN_AI_PANE_WIDTH"
        :aria-valuemax="Math.round(aiPaneMaximum)"
        :aria-valuenow="Math.round(aiPaneWidth)"
        tabindex="0"
        @pointerdown="beginResize"
        @keydown="resizeWithKeyboard"
      >
        <span />
      </div>

      <section
        class="workbench-pane"
        :class="{
          hidden: compactLayout && visiblePane !== 'workbench',
          collapsed: workbenchCollapsed && !compactLayout,
        }"
      >
        <nav class="workbench-menu" aria-label="工程开发工具">
          <div class="workbench-menu-main">
            <button
              type="button"
              :class="{ active: activeWorkbenchView === 'page' }"
              title="页面"
              aria-label="页面"
              @click="selectWorkbenchView('page')"
            >
              <IconLucideMonitorPlay />
              <span>页面</span>
            </button>
            <button
              type="button"
              :class="{ active: activeWorkbenchView === '2d' }"
              title="2D 产物"
              aria-label="2D"
              @click="selectWorkbenchView('2d')"
            >
              <IconLucideRoute />
              <span>2D</span>
            </button>
            <button
              type="button"
              :class="{ active: activeWorkbenchView === '3d' }"
              title="3D 产物"
              aria-label="3D"
              @click="selectWorkbenchView('3d')"
            >
              <IconLucideBox />
              <span>3D</span>
            </button>
            <button
              type="button"
              :class="{ active: activeWorkbenchView === 'editor' }"
              :disabled="!codeUrl"
              title="编辑器"
              aria-label="编辑器"
              @click="selectWorkbenchView('editor')"
            >
              <IconLucideCode2 />
              <span>编辑器</span>
            </button>
          </div>
        </nav>

        <p v-if="toolError" class="tool-error" role="alert">{{ toolError }}</p>

        <div class="workbench-content">
          <div
            class="workbench-view preview-workbench-view"
            :class="{ active: activeWorkbenchView === 'page' }"
          >
            <PreviewPanel
              :preview-url="previewUrl"
              :control-url="previewControlUrl"
              :active="activeWorkbenchView === 'page'"
            />
          </div>
          <div class="workbench-view" :class="{ active: activeWorkbenchView === '2d' }">
            <SceneArtifactsPanel
              kind="2d"
              :contracts="scene2dContracts"
              @open-editor="openWorkspace('2d')"
            />
          </div>
          <div class="workbench-view" :class="{ active: activeWorkbenchView === '3d' }">
            <SceneArtifactsPanel
              kind="3d"
              :contracts="scene3dContracts"
              @open-editor="openWorkspace('3d')"
            />
          </div>
          <div
            class="workbench-view code-server-shell"
            :class="{ active: activeWorkbenchView === 'editor' }"
          >
            <iframe
              v-if="codeUrl"
              class="service-frame code-server-frame"
              :class="{ loaded: codeFrameLoaded }"
              :src="codeUrl"
              title="工程开发工作台"
              @load="codeFrameLoaded = true"
            />
            <div v-if="codeUrl && !codeFrameLoaded" class="pane-state subtle-state">
              <span class="loading-line" />
              <strong>正在连接工程工作台</strong>
            </div>
            <div v-else-if="!codeUrl" class="pane-state">
              <span class="state-kicker">CODE-SERVER</span>
              <strong>{{ workspaceStateLabel || '工程工作台尚未接入' }}</strong>
              <p>
                {{
                  workspaceError || '控制面返回 code-server 受控地址后，源码开发环境将在此处加载。'
                }}
              </p>
              <button
                v-if="workspaceError || workspace?.status === 'error'"
                type="button"
                @click="retryWorkspace"
              >
                重新连接
              </button>
            </div>
          </div>
        </div>
      </section>
    </section>

    <footer v-if="workspaceInitialized" class="context-summary" :title="contextError">
      <span><b>上下文</b> {{ contextVersion }}</span>
      <i />
      <span><b>2D</b> {{ scene2dCount ?? '—' }} 个</span>
      <i />
      <span><b>3D</b> {{ scene3dCount ?? '—' }} 个</span>
      <i />
      <span><b>数据点位</b> {{ pointCount ?? '—' }} 个</span>
      <span class="summary-time">{{
        contextUpdatedAt ? `更新于 ${formatTime(contextUpdatedAt)}` : ''
      }}</span>
      <span v-if="contextStatus === 'error'" class="summary-error">部分摘要更新失败</span>
      <button
        type="button"
        class="summary-sync"
        :class="`status-${contextStatus}`"
        :title="contextError || '刷新 AI 使用的工程上下文'"
        :disabled="contextStatus === 'syncing'"
        @click="retryContext"
      >
        <IconLucideRefreshCw :class="{ spinning: contextStatus === 'syncing' }" />
        {{ contextStatusLabel }}
      </button>
    </footer>
  </main>
</template>

<style scoped>
.ai-workbench {
  --surface: #ffffff;
  --surface-subtle: #f6f7f9;
  --surface-hover: #eef2f7;
  --ink: #172033;
  --muted: #64748b;
  --muted-light: #94a3b8;
  --line: #d8dee8;
  --line-light: #e8ecf2;
  --primary: #2563eb;
  --primary-hover: #1d4ed8;
  --primary-light: #eaf1ff;
  --success: #16803a;
  --danger: #dc2626;
  --workbench-menu-width: 46px;
  width: 100%;
  height: 100%;
  min-height: 0;
  min-width: 0;
  display: grid;
  grid-template-rows: minmax(0, 1fr) 32px;
  overflow: hidden;
  color: var(--ink);
  background: #edf0f4;
}

.ai-workbench.setup-mode {
  grid-template-rows: minmax(0, 1fr);
}

.workspace-setup {
  min-width: 0;
  min-height: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: auto;
  padding: 36px;
  background: var(--surface-subtle);
}

.workspace-setup-content {
  width: min(760px, 100%);
}

.workspace-setup-loading,
.workspace-setup-state {
  flex-direction: column;
  color: var(--muted);
  text-align: center;
}

.workspace-setup-loading strong,
.workspace-setup-state strong {
  margin-top: 12px;
  color: var(--ink);
  font-size: 15px;
}

.workspace-setup-loading p,
.workspace-setup-state p {
  margin: 7px 0 0;
  font-size: 12px;
}

.workspace-setup-kicker {
  color: var(--primary);
  font-size: 10px;
  font-weight: 700;
  letter-spacing: 0.12em;
}

.workspace-setup h1 {
  margin: 8px 0 0;
  color: var(--ink);
  font-size: 24px;
  font-weight: 680;
  letter-spacing: 0;
}

.workspace-setup-description {
  margin: 8px 0 24px;
  color: var(--muted);
  font-size: 13px;
  line-height: 1.6;
}

.template-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
}

.template-option {
  position: relative;
  min-width: 0;
  min-height: 92px;
  display: grid;
  grid-template-columns: 42px minmax(0, 1fr) auto;
  align-items: center;
  gap: 12px;
  padding: 15px;
  border: 1px solid var(--line);
  border-radius: 6px;
  color: var(--ink);
  background: var(--surface);
  text-align: left;
  cursor: pointer;
}

.template-option:hover {
  border-color: #aebbd0;
  background: var(--surface-hover);
}

.template-option.selected {
  border-color: var(--primary);
  box-shadow: 0 0 0 1px var(--primary);
}

.template-icon {
  width: 42px;
  height: 42px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 6px;
  background: #e9f8f2;
  color: #087f5b;
}

.template-icon.framework-react {
  color: #087ea4;
  background: #e7f7fb;
}

.template-icon svg {
  width: 21px;
  height: 21px;
}

.template-copy {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 5px;
}

.template-copy strong {
  overflow: hidden;
  font-size: 13px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.template-copy small {
  overflow: hidden;
  color: var(--muted);
  font-size: 11px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.template-language {
  align-self: start;
  padding: 3px 5px;
  border: 1px solid var(--line-light);
  border-radius: 4px;
  color: var(--muted);
  background: var(--surface-subtle);
  font-size: 9px;
  font-weight: 700;
}

.workspace-setup-state {
  min-height: 164px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--line);
  border-radius: 6px;
  background: var(--surface);
}

.workspace-setup-error {
  margin: 12px 0 0;
  color: var(--danger);
  font-size: 12px;
}

.workspace-setup-actions {
  min-height: 34px;
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
}

.initialize-button,
.retry-setup-button {
  height: 34px;
  padding: 0 16px;
  border-radius: 5px;
  font-size: 12px;
  font-weight: 650;
  cursor: pointer;
}

.initialize-button {
  border: 1px solid var(--primary);
  color: #fff;
  background: var(--primary);
}

.initialize-button:hover:not(:disabled) {
  border-color: var(--primary-hover);
  background: var(--primary-hover);
}

.initialize-button:disabled {
  opacity: 0.55;
  cursor: default;
}

.retry-setup-button {
  border: 1px solid var(--line);
  color: var(--ink);
  background: var(--surface);
}

.workbench-stage {
  min-width: 0;
  min-height: 0;
  display: grid;
  overflow: hidden;
  border-bottom: 1px solid var(--line);
  background: var(--surface);
}

.left-workspace {
  min-width: 0;
  min-height: 0;
  display: block;
}

.left-workspace .service-pane {
  width: 100%;
  height: 100%;
}

.context-summary,
.summary-sync {
  display: flex;
  align-items: center;
}

.summary-sync svg {
  width: 14px;
  height: 14px;
}

.tool-error {
  position: absolute;
  z-index: 8;
  margin: 0;
  padding: 7px 10px;
  border: 1px solid #fecaca;
  border-radius: 5px;
  color: var(--danger);
  background: #fff;
  box-shadow: 0 6px 18px rgba(15, 23, 42, 0.12);
  font-size: 11px;
}

.tool-error {
  top: 8px;
  left: 62px;
}

.service-pane,
.workbench-content,
.code-server-shell {
  position: relative;
  min-width: 0;
  min-height: 0;
  overflow: hidden;
  background: var(--surface);
}

.service-pane.hidden,
.workbench-pane.hidden {
  display: none;
}

.service-frame {
  width: 100%;
  height: 100%;
  display: block;
  border: 0;
  opacity: 0;
  background: var(--surface);
}

.service-frame.loaded {
  opacity: 1;
}

.split-handle {
  position: relative;
  z-index: 2;
  cursor: col-resize;
  touch-action: none;
  background: #e6eaf0;
  outline: none;
}

.split-handle span {
  position: absolute;
  top: 50%;
  left: 2px;
  width: 2px;
  height: 34px;
  border-radius: 2px;
  background: #aeb8c7;
  transform: translateY(-50%);
}

.split-handle:hover span,
.split-handle:focus span {
  background: var(--primary);
}

.workbench-pane {
  position: relative;
  min-width: 0;
  min-height: 0;
  display: grid;
  grid-template-columns: var(--workbench-menu-width) minmax(0, 1fr);
  background: #1f1f1f;
}

.workbench-pane.collapsed {
  grid-template-columns: var(--workbench-menu-width) 0;
}

.workbench-pane.collapsed .workbench-content {
  visibility: hidden;
  pointer-events: none;
}

.workbench-menu {
  min-height: 0;
  display: flex;
  flex-direction: column;
  justify-content: flex-start;
  border-right: 1px solid #d7dde6;
  background: #f7f8fa;
}

.workbench-menu-main {
  display: flex;
  flex-direction: column;
}

.workbench-menu button {
  width: calc(var(--workbench-menu-width) - 1px);
  min-height: 48px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 4px;
  padding: 5px 2px;
  border: 0;
  border-left: 2px solid transparent;
  color: #667085;
  background: transparent;
  font-size: 9px;
  cursor: pointer;
}

.workbench-menu button svg {
  width: 18px;
  height: 18px;
}

.workbench-menu button:hover:not(:disabled) {
  color: #26344d;
  background: #e9edf3;
}

.workbench-menu button.active {
  border-left-color: var(--primary);
  color: var(--primary);
  background: #eaf1ff;
}

.workbench-menu button:disabled {
  color: #b2bac7;
  cursor: default;
}

.workbench-content {
  position: relative;
  min-width: 0;
  min-height: 0;
  overflow: hidden;
}

.workbench-view {
  position: absolute;
  inset: 0;
  visibility: hidden;
  opacity: 0;
  pointer-events: none;
}

.workbench-view.active {
  visibility: visible;
  opacity: 1;
  pointer-events: auto;
}

.preview-workbench-view {
  background: #eef1f5;
}

.code-server-frame {
  background: #1f1f1f;
}

.pane-state {
  position: absolute;
  inset: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 32px;
  color: var(--muted);
  text-align: center;
  background: var(--surface-subtle);
}

.pane-state strong {
  margin-top: 8px;
  color: var(--ink);
  font-size: 15px;
  font-weight: 650;
}

.pane-state p {
  max-width: 420px;
  margin: 7px 0 0;
  font-size: 12px;
  line-height: 1.6;
}

.pane-state button {
  margin-top: 16px;
  height: 32px;
  padding: 0 13px;
  border: 1px solid #bfd0ee;
  border-radius: 5px;
  color: var(--primary);
  background: var(--primary-light);
  font-weight: 600;
  cursor: pointer;
}

.state-kicker {
  color: var(--muted-light);
  font:
    600 10px Inter,
    'Helvetica Neue',
    sans-serif;
  letter-spacing: 0.12em;
}

.subtle-state {
  z-index: 1;
  background: var(--surface);
}

.loading-line {
  width: 120px;
  height: 3px;
  overflow: hidden;
  border-radius: 2px;
  background: #e2e8f0;
}

.loading-line::after {
  content: '';
  display: block;
  width: 42px;
  height: 100%;
  background: var(--primary);
  animation: loading-line 1.1s ease-in-out infinite;
}

.compact-tabs {
  height: 36px;
  display: grid;
  grid-template-columns: 1fr 1fr;
  border-bottom: 1px solid var(--line-light);
  background: var(--surface);
}

.compact-tabs button {
  border: 0;
  border-bottom: 2px solid transparent;
  color: var(--muted);
  background: transparent;
  font-size: 12px;
  cursor: pointer;
}

.compact-tabs button.active {
  border-bottom-color: var(--primary);
  color: var(--primary);
  font-weight: 650;
}

.workbench-stage.compact {
  grid-template-columns: minmax(0, 1fr) !important;
  grid-template-rows: 36px minmax(0, 1fr);
}

.workbench-stage.compact .left-workspace {
  display: contents;
}

.workbench-stage.compact .compact-tabs {
  grid-row: 1;
}

.workbench-stage.compact .service-pane,
.workbench-stage.compact .workbench-pane {
  grid-row: 2;
}

.context-summary {
  min-width: 0;
  gap: 10px;
  padding: 0 12px;
  color: var(--muted);
  background: #f8f9fb;
  font-size: 11px;
  white-space: nowrap;
}

.context-summary b {
  color: var(--ink);
  font-weight: 600;
}

.context-summary i {
  width: 1px;
  height: 12px;
  background: var(--line);
}

.summary-time {
  margin-left: auto;
}

.summary-error {
  color: var(--danger);
}

.summary-sync {
  height: 25px;
  gap: 5px;
  padding: 0 7px;
  border: 0;
  border-radius: 4px;
  color: var(--muted);
  background: transparent;
  font: inherit;
  cursor: pointer;
}

.summary-sync:hover:not(:disabled) {
  color: var(--primary);
  background: var(--primary-light);
}

.summary-sync.status-synced {
  color: var(--success);
}

.summary-sync.status-error {
  color: var(--danger);
}

.spinning {
  animation: spin 0.9s linear infinite;
}

:global(html.dark) .ai-workbench,
:global([data-theme='dark']) .ai-workbench {
  --surface: #20242b;
  --surface-subtle: #171a20;
  --surface-hover: #2d333d;
  --ink: #eef2f7;
  --muted: #a8b0bd;
  --muted-light: #788292;
  --line: #3c434e;
  --line-light: #303640;
  --primary: #75a7ff;
  --primary-hover: #9abfff;
  --primary-light: #263958;
  background: #171a20;
}

:global(html.dark) .workbench-menu,
:global([data-theme='dark']) .workbench-menu,
:global(html.dark) .context-summary,
:global([data-theme='dark']) .context-summary,
:global(html.dark) .template-option,
:global([data-theme='dark']) .template-option,
:global(html.dark) .workspace-setup-state,
:global([data-theme='dark']) .workspace-setup-state {
  border-color: #3c434e;
  background: #242930;
}

:global(html.dark) .workbench-menu button:hover:not(:disabled),
:global([data-theme='dark']) .workbench-menu button:hover:not(:disabled) {
  color: #e8edf5;
  background: #303640;
}

:global(html.dark) .workbench-menu button.active,
:global([data-theme='dark']) .workbench-menu button.active {
  color: #8db5ff;
  background: #263958;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

@keyframes loading-line {
  from {
    transform: translateX(-42px);
  }
  to {
    transform: translateX(120px);
  }
}

@media (max-width: 660px) {
  .workspace-setup {
    align-items: flex-start;
    padding: 24px 16px;
  }

  .template-grid {
    grid-template-columns: minmax(0, 1fr);
  }

  .template-option {
    min-height: 82px;
  }

  .context-summary {
    gap: 7px;
    overflow-x: auto;
  }

  .summary-time {
    margin-left: 0;
  }

  .summary-sync {
    position: sticky;
    right: 0;
    flex: 0 0 auto;
    background: var(--surface);
  }
}
</style>
