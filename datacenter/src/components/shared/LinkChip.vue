<template>
  <button type="button" class="dc-link-chip" :title="label" @click="$emit('click', payload)">
    <component :is="moduleIcon" class="dc-link-chip__icon" aria-hidden="true" />
    <span class="dc-link-chip__label">{{ label }}</span>
    <ArrowRight class="dc-link-chip__arrow" aria-hidden="true" />
  </button>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { ArrowRight, Bell, Connection, Cpu, DataLine } from '@element-plus/icons-vue'

type LinkChipModule = 'datapoint' | 'access-source' | 'compute' | 'alarm'

const props = defineProps<{
  module: LinkChipModule
  objectId: string
  label: string
}>()

defineEmits<{
  (event: 'click', payload: { module: LinkChipModule; objectId: string; label: string }): void
}>()

const moduleIconMap = {
  datapoint: DataLine,
  'access-source': Connection,
  compute: Cpu,
  alarm: Bell,
}

const moduleIcon = computed(() => moduleIconMap[props.module])
const payload = computed(() => ({
  module: props.module,
  objectId: props.objectId,
  label: props.label,
}))
</script>

<style scoped>
.dc-link-chip {
  height: 24px;
  max-width: 100%;
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 0 8px;
  border: 1px solid var(--dc-border);
  border-radius: 8px;
  background: var(--dc-surface-raised);
  color: var(--dc-text-secondary);
  cursor: pointer;
  font-family: inherit;
  font-size: 12px;
  font-weight: 600;
  letter-spacing: 0;
  line-height: 1;
  transition:
    background-color 0.18s ease,
    border-color 0.18s ease,
    color 0.18s ease;
}

.dc-link-chip:hover {
  border-color: rgba(29, 78, 216, 0.22);
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
}

.dc-link-chip:focus-visible {
  outline: none;
  border-color: var(--dc-primary);
  box-shadow: 0 0 0 3px rgba(29, 78, 216, 0.12);
}

.dc-link-chip__icon,
.dc-link-chip__arrow {
  width: 14px;
  height: 14px;
  flex: 0 0 14px;
}

.dc-link-chip__label {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.dc-link-chip__arrow {
  color: var(--dc-text-muted);
}
</style>
