<template>
  <MqttWorkbench
    :project-id="projectId"
    :connection="connection"
    mode="builtin-message"
    @back="$emit('back')"
  />
</template>

<script setup lang="ts">
import MqttWorkbench from '@/components/mqtt/MqttWorkbench.vue'

defineProps<{
  projectId: string
  connection: {
    id: string
    name?: string
    type?: string
    config?: Record<string, any>
    mqttConfig?: Record<string, any>
  }
}>()

defineEmits<{
  (event: 'back'): void
}>()
</script>
