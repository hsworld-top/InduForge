<template>
  <div class="aws-mqtt">
    <MqttWorkbench :project-id="projectId" :connection="connection" @back="$emit('back')" />
  </div>
</template>

<script setup lang="ts">
import MqttWorkbench from '@/components/mqtt/MqttWorkbench.vue'

defineProps<{
  connection: Record<string, any>
  projectId: string
}>()

defineEmits<{
  (event: 'back'): void
}>()
</script>

<style scoped>
.aws-mqtt {
  height: 100%;
  min-height: 0;
  overflow: hidden;
}
</style>
