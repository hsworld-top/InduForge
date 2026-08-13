<template>
  <section class="workspace-tool-frame">
    <iframe :src="frameUrl" :title="frameTitle" />
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { WorkspaceToolProject, WorkspaceToolTarget } from '@/types/workspace-tool'
import { buildWorkspaceToolUrl } from '@/types/workspace-tool'

const props = defineProps<{
  target: WorkspaceToolTarget
  project: WorkspaceToolProject
}>()

const frameUrl = computed(() => buildWorkspaceToolUrl(props.target, String(props.project.id)))
const frameTitle = computed(() => {
  const projectName = props.project.name || '工程'
  return props.target === '2d' ? `${projectName} 2D 编辑器` : `${projectName} 3D 编辑器`
})
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

.workspace-tool-frame iframe {
  display: block;
  border: 0;
  background: #fff;
}
</style>
