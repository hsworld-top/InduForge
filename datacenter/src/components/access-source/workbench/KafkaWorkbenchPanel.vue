<template>
  <div class="aws-kafka">
    <KafkaWorkbench :project-id="projectId" :connection="connection" @back="$emit('back')" />
  </div>
</template>

<script setup lang="ts">
import KafkaWorkbench from '@/components/kafka/KafkaWorkbench.vue'

defineProps<{
  connection: Record<string, any>
  projectId: string
}>()

defineEmits<{
  (event: 'back'): void
}>()
</script>

<style scoped>
.aws-kafka {
  height: 100%;
  min-height: 0;
  overflow: hidden;
}
</style>
