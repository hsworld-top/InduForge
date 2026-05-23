<script setup lang="ts">
import { storeToRefs } from 'pinia'
import { computed } from 'vue'
import { useEditorStore } from '@/stores/editor-store'

const props = defineProps<{
  resolvedProps?: Record<string, unknown>
}>()

const editorStore = useEditorStore()
const { projectI18n, projectRuntimeLocale } = storeToRefs(editorStore)

const mode = computed(() => String(props.resolvedProps?.mode || 'select'))
const display = computed(() => String(props.resolvedProps?.display || 'name'))
const size = computed(() => String(props.resolvedProps?.size || 'default'))
const disabled = computed(() => Boolean(props.resolvedProps?.disabled))
const locales = computed(() => projectI18n.value.locales.filter((item) => item.enabled))
const model = computed({
  get: () => projectRuntimeLocale.value || projectI18n.value.defaultLocale,
  set: (value: string) => {
    editorStore.setProjectRuntimeLocale(value)
  },
})

function localeLabel(locale: { code: string; name: string }): string {
  if (display.value === 'code') return locale.code
  if (display.value === 'name-code') return `${locale.name} ${locale.code}`
  return locale.name
}
</script>

<template>
  <div class="language-switcher">
    <el-radio-group
      v-if="mode === 'button-group'"
      v-model="model"
      :size="size as never"
      :disabled="disabled"
    >
      <el-radio-button
        v-for="localeItem in locales"
        :key="localeItem.code"
        :label="localeItem.code"
      >
        {{ localeLabel(localeItem) }}
      </el-radio-button>
    </el-radio-group>
    <el-select v-else v-model="model" :size="size as never" :disabled="disabled">
      <el-option
        v-for="localeItem in locales"
        :key="localeItem.code"
        :label="localeLabel(localeItem)"
        :value="localeItem.code"
      />
    </el-select>
  </div>
</template>

<style scoped>
.language-switcher {
  display: inline-flex;
  align-items: center;
  min-width: 120px;
}

.language-switcher :deep(.el-select) {
  width: 100%;
}
</style>
