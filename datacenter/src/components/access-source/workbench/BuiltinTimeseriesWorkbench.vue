<template>
  <SqlQueryWorkbench :project-id="projectId" :connection="sqlConnection" @back="$emit('back')" />
</template>

<script setup lang="ts">
import { computed } from 'vue'
import SqlQueryWorkbench from '@/components/database/SqlQueryWorkbench.vue'

const props = defineProps<{
  connection: Record<string, any>
  projectId: string
}>()

defineEmits<{
  (event: 'back'): void
}>()

const sqlConnection = computed(() => ({
  ...props.connection,
  type: 'builtin.timeseries',
  relationalConfig: {
    dbType: 'postgresql',
    database: props.connection.name || 'IF时序库',
    schema: props.connection.config?.runtimeKey || 'system',
  },
}))
</script>
