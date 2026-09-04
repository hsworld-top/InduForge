<template>
  <span class="ops-business-message"
    >{{ display || '—' }}
    <details
      v-if="text && display !== text"
      @toggle="expanded = ($event.target as HTMLDetailsElement).open"
    >
      <summary>技术详情</summary>
      <pre v-if="expanded">{{ text }}</pre>
    </details></span
  >
</template>
<script setup lang="ts">
import { computed, ref } from 'vue'
import { opsMessageSummary } from '../utils/ops-business'
const props = defineProps<{
  text?: string
  summary?: string
  kind?: 'error' | 'progress' | 'info'
}>()
const expanded = ref(false)
const display = computed(() => opsMessageSummary(props.summary || props.text, props.kind))
</script>
<style scoped>
.ops-business-message {
  overflow-wrap: anywhere;
}
summary {
  color: var(--el-color-primary);
  cursor: pointer;
  font-size: 12px;
}
pre {
  white-space: pre-wrap;
  overflow-wrap: anywhere;
  font-size: 12px;
  margin: 6px 0;
}
</style>
