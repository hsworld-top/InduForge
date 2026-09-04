<template>
  <el-tooltip :content="action.label" placement="top">
    <el-button
      size="small"
      text
      data-testid="project-deploy-action"
      :aria-label="action.label"
      :disabled="action.disabled"
      @click.stop="emit('click')"
    >
      <el-icon :class="{ 'is-loading': action.kind === 'progress' || action.kind === 'loading' }"
        ><component :is="icons[action.icon]"
      /></el-icon>
      <span>{{ action.label }}</span>
    </el-button>
  </el-tooltip>
</template>
<script setup lang="ts">
import { computed } from 'vue'
import { Loading, Setting, UploadFilled, VideoPlay, Warning } from '@element-plus/icons-vue'
import { projectDeploymentAction, type ProjectDeploymentSummary } from './project-deployment-action'
const props = defineProps<{ summary?: ProjectDeploymentSummary | null }>()
const emit = defineEmits<{ click: [] }>()
const action = computed(() => projectDeploymentAction(props.summary))
const icons: Record<string, unknown> = { Loading, Setting, UploadFilled, VideoPlay, Warning }
</script>
