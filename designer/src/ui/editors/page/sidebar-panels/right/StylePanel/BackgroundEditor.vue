<!--
  BackgroundEditor - 背景样式编辑器
  编辑背景颜色（含透明度）
-->
<script setup lang="ts">
import { computed } from 'vue'
import FriendlyColorPicker from '@/ui/shared/widgets/base/FriendlyColorPicker.vue'

interface BackgroundStyleModel {
  background?: string
}

const props = withDefaults(defineProps<{ modelValue?: BackgroundStyleModel }>(), {
  modelValue: () => ({}),
})

const emit = defineEmits<{
  (event: 'update:modelValue', value: BackgroundStyleModel): void
}>()

const backgroundColor = computed(() => props.modelValue.background || '')

function handleColorChange(value: string) {
  emit('update:modelValue', { ...props.modelValue, background: value })
}
</script>

<template>
  <div class="background-editor">
    <div class="editor-group-title">背景</div>
    <div class="form-item">
      <label>背景颜色</label>
      <FriendlyColorPicker
        :model-value="backgroundColor"
        :show-alpha="true"
        @update:model-value="handleColorChange"
      />
    </div>
  </div>
</template>

<style scoped>
.background-editor {
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
