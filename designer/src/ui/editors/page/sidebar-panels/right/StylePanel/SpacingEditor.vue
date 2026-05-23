<!--
  SpacingEditor - 间距编辑器
  编辑 padding 或 margin（上右下左）
-->
<script setup lang="ts">
import { computed } from 'vue'

interface SpacingStyleModel {
  [key: string]: number | string | undefined
}

const props = withDefaults(
  defineProps<{
    title?: string
    modelValue?: SpacingStyleModel
    prefix: string
  }>(),
  {
    title: '间距',
    modelValue: () => ({}),
  },
)

const emit = defineEmits<{
  (event: 'update:modelValue', value: SpacingStyleModel): void
}>()

function parseValue(value: unknown): number {
  if (!value) return 0
  const num = Number.parseInt(String(value), 10)
  return Number.isNaN(num) ? 0 : num
}

const top = computed(() => parseValue(props.modelValue[`${props.prefix}Top`]))
const right = computed(() => parseValue(props.modelValue[`${props.prefix}Right`]))
const bottom = computed(() => parseValue(props.modelValue[`${props.prefix}Bottom`]))
const left = computed(() => parseValue(props.modelValue[`${props.prefix}Left`]))

function handleTopChange(value: number) {
  emit('update:modelValue', {
    ...props.modelValue,
    [`${props.prefix}Top`]: value,
  })
}

function handleRightChange(value: number) {
  emit('update:modelValue', {
    ...props.modelValue,
    [`${props.prefix}Right`]: value,
  })
}

function handleBottomChange(value: number) {
  emit('update:modelValue', {
    ...props.modelValue,
    [`${props.prefix}Bottom`]: value,
  })
}

function handleLeftChange(value: number) {
  emit('update:modelValue', {
    ...props.modelValue,
    [`${props.prefix}Left`]: value,
  })
}
</script>

<template>
  <div class="spacing-editor">
    <div class="editor-group-title">{{ title }}</div>
    <div class="spacing-grid">
      <div class="form-item">
        <label>上</label>
        <el-input-number
          :model-value="top"
          size="small"
          :min="0"
          :step="1"
          controls-position="right"
          @update:model-value="handleTopChange"
        />
      </div>
      <div class="form-item">
        <label>右</label>
        <el-input-number
          :model-value="right"
          size="small"
          :min="0"
          :step="1"
          controls-position="right"
          @update:model-value="handleRightChange"
        />
      </div>
      <div class="form-item">
        <label>下</label>
        <el-input-number
          :model-value="bottom"
          size="small"
          :min="0"
          :step="1"
          controls-position="right"
          @update:model-value="handleBottomChange"
        />
      </div>
      <div class="form-item">
        <label>左</label>
        <el-input-number
          :model-value="left"
          size="small"
          :min="0"
          :step="1"
          controls-position="right"
          @update:model-value="handleLeftChange"
        />
      </div>
    </div>
  </div>
</template>

<style scoped>
.spacing-editor {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.editor-group-title {
  font-size: 12px;
  font-weight: 500;
  color: var(--el-text-color-secondary);
  padding-bottom: 4px;
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.spacing-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 8px;
}

.form-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.form-item label {
  font-size: 12px;
  color: var(--el-text-color-regular);
}
</style>
