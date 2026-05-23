<template>
  <div class="aws-sql">
    <SqlQueryWorkbench :project-id="projectId" :connection="connection" @back="$emit('back')" />
  </div>
</template>

<script setup lang="ts">
import SqlQueryWorkbench from '@/components/database/SqlQueryWorkbench.vue'

defineProps<{
  connection: Record<string, any>
  projectId: string
}>()

defineEmits<{
  (event: 'back'): void
}>()
</script>

<style scoped>
.aws-sql {
  height: 100%;
  min-height: 0;
  overflow: hidden;
}
</style>
