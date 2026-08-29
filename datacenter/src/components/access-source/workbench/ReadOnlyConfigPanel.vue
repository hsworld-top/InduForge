<template>
  <div class="aws-readonly-config">
    <EmptyState
      :title="ui('工业协议仅配置', 'Industrial Protocol Configuration Only')"
      :description="ui('该协议在开发态仅持有配置契约，真实采集由节点侧执行。', 'Development mode stores only the configuration contract; actual collection runs on the node.')"
      icon-name="access-source"
    >
      <el-button size="small" @click="$emit('back')">{{ ui('返回接入源列表', 'Back to Access Sources') }}</el-button>
    </EmptyState>
  </div>
</template>

<script setup lang="ts">
import EmptyState from '@/components/shared/EmptyState.vue'
import { datacenterLocale } from '@/i18n/runtime'

const ui = (zh: string, en: string) => (datacenterLocale.value === 'en' ? en : zh)

defineProps<{
  connection: Record<string, any>
  projectId: string
}>()

defineEmits<{
  (event: 'back'): void
}>()
</script>

<style scoped>
.aws-readonly-config {
  height: 100%;
  min-height: 0;
  display: flex;
  align-items: center;
  justify-content: center;
}
</style>
