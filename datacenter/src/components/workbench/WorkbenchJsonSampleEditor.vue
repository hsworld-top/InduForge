<template>
  <section class="workbench-json-sample-editor">
    <header class="workbench-json-sample-editor__header">
      <strong>{{ title }}</strong>
      <div class="workbench-json-sample-editor__actions">
        <el-button size="small" :disabled="fillLatestDisabled" @click="$emit('fillLatest')">
          {{ fillLatestText }}
        </el-button>
        <el-button size="small" @click="formatEditor">格式化</el-button>
        <el-button type="primary" size="small" @click="$emit('parse')">
          {{ parseText }}
        </el-button>
      </div>
    </header>
    <MonacoEditor
      ref="editorRef"
      class="workbench-json-sample-editor__monaco"
      :model-value="modelValue"
      language="json"
      theme="vs"
      height="100%"
      :options="editorOptions"
      @update:model-value="$emit('update:modelValue', $event)"
    />
    <p v-if="error" class="workbench-json-sample-editor__error">{{ error }}</p>
  </section>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import MonacoEditor from '@/components/MonacoEditor.vue'

withDefaults(
  defineProps<{
    modelValue: string
    title?: string
    fillLatestText?: string
    fillLatestDisabled?: boolean
    parseText?: string
    error?: string
  }>(),
  {
    title: 'JSON 样例',
    fillLatestText: '填入最近样本',
    fillLatestDisabled: false,
    parseText: '解析字段',
    error: '',
  },
)

defineEmits<{
  (event: 'update:modelValue', value: string): void
  (event: 'fillLatest'): void
  (event: 'parse'): void
}>()

const editorRef = ref<InstanceType<typeof MonacoEditor> | null>(null)
const editorOptions = {
  minimap: { enabled: false },
  lineNumbers: 'on',
  wordWrap: 'on',
  scrollBeyondLastLine: false,
}

const formatEditor = async () => {
  await editorRef.value?.format?.()
}

defineExpose({
  format: formatEditor,
  focus: () => editorRef.value?.focus?.(),
})
</script>

<style scoped>
.workbench-json-sample-editor {
  min-height: 0;
  min-width: 0;
  display: grid;
  grid-template-rows: auto minmax(220px, 1fr) auto;
  gap: 8px;
}

.workbench-json-sample-editor__header,
.workbench-json-sample-editor__actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.workbench-json-sample-editor__header {
  justify-content: space-between;
}

.workbench-json-sample-editor__header strong {
  min-width: 0;
  overflow: hidden;
  color: var(--dc-text);
  font-size: 13px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.workbench-json-sample-editor__actions {
  flex-shrink: 0;
  flex-wrap: wrap;
  justify-content: flex-end;
}

.workbench-json-sample-editor__monaco {
  min-height: 0;
  overflow: hidden;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
}

.workbench-json-sample-editor__error {
  margin: 0;
  color: var(--dc-danger);
  font-size: 12px;
  line-height: 1.5;
}
</style>
