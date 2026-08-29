<template>
  <div class="retention-field">
    <span>{{ ui('保留时间', 'Retention') }}</span>
    <div class="retention-field__control">
      <el-input-number
        v-if="modelValue !== null"
        :model-value="modelValue"
        :min="1"
        :controls="false"
        @update:model-value="updateDays"
      />
      <span v-if="modelValue !== null">{{ ui('天', 'days') }}</span>
      <el-checkbox :model-value="modelValue === null" @change="toggleForever">{{ ui('永久', 'Forever') }}</el-checkbox>
    </div>
  </div>
</template>

<script setup lang="ts">
import { datacenterLocale } from '@/i18n/runtime'

const ui = (zh: string, en: string) => (datacenterLocale.value === 'en' ? en : zh)
const props = defineProps<{ modelValue: number | null }>()
const emit = defineEmits<{ (event: 'update:modelValue', value: number | null): void }>()
function updateDays(value: number | undefined) {
  emit('update:modelValue', Math.max(1, value || 1))
}
function toggleForever(value: boolean | string | number) {
  emit('update:modelValue', value ? null : props.modelValue || 30)
}
</script>

<style scoped>
.retention-field {
  min-width: 0;
  display: grid;
  gap: 6px;
  color: var(--dc-text-secondary);
  font-size: 12px;
}
.retention-field__control {
  min-height: 32px;
  display: grid;
  grid-template-columns: minmax(70px, 1fr) auto auto;
  align-items: center;
  gap: 6px;
}
.retention-field :deep(.el-input-number) {
  width: 100%;
}
</style>
