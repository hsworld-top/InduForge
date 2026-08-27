<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { getApiErrorMessage } from '@/utils/request'
import { resolveWorkspaceState } from './code/workspace-runtime'
import type { CodeWorkspaceState } from './code/code-workspace-api'
import PreviewPanel from './PreviewPanel.vue'

const route = useRoute()
const projectId = computed(() => String(route.meta.project?.id || '').trim())
const workspace = ref<CodeWorkspaceState | null>(null)
const error = ref('')

onMounted(async () => {
  if (!projectId.value) {
    error.value = '缺少工程标识'
    return
  }
  try {
    workspace.value = await resolveWorkspaceState(projectId.value, false)
  } catch (reason) {
    error.value = getApiErrorMessage(reason, '读取工程预览地址失败')
  }
})
</script>

<template>
  <main class="standalone-preview">
    <PreviewPanel
      v-if="workspace"
      :project-id="projectId"
      :preview-url="workspace.services.preview.url"
      :control-url="workspace.services.previewControl.url"
      :active="true"
    />
    <p v-else>{{ error || '正在加载工程预览' }}</p>
  </main>
</template>

<style scoped>
.standalone-preview { width: 100vw; height: 100vh; margin: 0; overflow: hidden; }
.standalone-preview > p { display: grid; height: 100%; margin: 0; place-items: center; color: #52605b; }
</style>
