<template>
  <section class="workspace-tool-frame">
    <iframe ref="frameRef" v-if="frameUrl" :src="frameUrl" :title="frameTitle" />
    <div v-else class="workspace-tool-state">
      <strong>{{ loading ? '正在创建编辑会话' : '编辑器暂不可用' }}</strong>
      <p v-if="error">{{ error }}</p>
      <button v-if="error" type="button" @click="loadSession">重试</button>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import request, { getApiErrorMessage } from '@/utils/request'
import type { ApiResponse } from '@/types/api'
import type {
  WorkspaceSceneCommittedEvent,
  WorkspaceToolProject,
  WorkspaceToolTarget,
} from '@/types/workspace-tool'

const props = defineProps<{
  target: WorkspaceToolTarget
  project: WorkspaceToolProject
  sceneId: string
  sceneName: string
}>()

const emit = defineEmits<{
  sceneCommitted: [event: WorkspaceSceneCommittedEvent]
}>()

interface EditorSessionResponse {
  sessionId: string
  url: string
  expiresAt: string
  sceneName: string
}

const frameUrl = ref('')
const frameRef = ref<HTMLIFrameElement | null>(null)
const loading = ref(false)
const error = ref('')
const frameTitle = computed(() => {
  return props.target === '2d' ? `${props.sceneName} 2D 编辑器` : `${props.sceneName} 3D 编辑器`
})

function handleEditorMessage(event: MessageEvent): void {
  if (event.origin !== window.location.origin || event.source !== frameRef.value?.contentWindow)
    return
  const payload = event.data as Record<string, unknown> | null
  if (
    !payload ||
    payload.source !== 'induforge-scene' ||
    payload.type !== 'committed' ||
    typeof payload.revision !== 'number'
  ) {
    return
  }
  emit('sceneCommitted', {
    projectId: String(props.project.id),
    target: props.target,
    sceneId: props.sceneId,
    revision: payload.revision,
  })
}

async function loadSession(): Promise<void> {
  loading.value = true
  error.value = ''
  frameUrl.value = ''
  try {
    const response = (await request.post<ApiResponse<EditorSessionResponse>>(
      `/projects/${encodeURIComponent(String(props.project.id))}/scenes/${encodeURIComponent(props.sceneId)}/editor-session`,
      undefined,
      { params: { kind: props.target } },
    )) as unknown as ApiResponse<EditorSessionResponse>
    if (!response.data?.url) throw new Error('编辑会话接口未返回 URL')
    frameUrl.value = response.data.url
  } catch (reason) {
    error.value = getApiErrorMessage(reason, '创建编辑会话失败')
  } finally {
    loading.value = false
  }
}

watch(() => [props.project.id, props.target, props.sceneId, props.sceneName], loadSession, {
  immediate: true,
})

onMounted(() => window.addEventListener('message', handleEditorMessage))
onBeforeUnmount(() => window.removeEventListener('message', handleEditorMessage))
</script>

<style scoped>
.workspace-tool-frame,
.workspace-tool-frame iframe {
  width: 100%;
  height: 100%;
}

.workspace-tool-frame {
  overflow: hidden;
  background: #f4f6f5;
}

.workspace-tool-state {
  height: 100%;
  display: grid;
  place-content: center;
  gap: 8px;
  color: #334155;
  text-align: center;
}

.workspace-tool-state p {
  margin: 0;
  color: #b42318;
  font-size: 12px;
}

.workspace-tool-state button {
  justify-self: center;
  padding: 6px 12px;
  border: 1px solid #94a3b8;
  background: #fff;
  cursor: pointer;
}

.workspace-tool-frame iframe {
  display: block;
  border: 0;
  background: #fff;
}
</style>
