<template>
  <el-dialog
    :model-value="modelValue"
    :title="title"
    width="680px"
    @close="emit('update:modelValue', false)"
  >
    <el-alert
      :title="ui(`成功 ${successCount} 个，失败 ${failed.length} 个`, `${successCount} succeeded, ${failed.length} failed`)"
      :type="successCount > 0 ? 'warning' : 'error'"
      show-icon
      :closable="false"
    />
    <el-table :data="failed" max-height="360" class="collector-batch-result__table">
      <el-table-column prop="index" :label="ui('序号', 'No.')" width="72">
        <template #default="scope">{{ scope.row.index + 1 }}</template>
      </el-table-column>
      <el-table-column prop="name" :label="ui('变量名称', 'Point Name')" min-width="160" show-overflow-tooltip />
      <el-table-column prop="message" :label="ui('失败原因', 'Failure Reason')" min-width="280" show-overflow-tooltip />
    </el-table>
    <template #footer>
      <el-button type="primary" @click="emit('update:modelValue', false)">{{ ui('知道了', 'OK') }}</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import type { CollectorPointBatchFailure } from '@/api/schemas/collector.schema'
import { computed } from 'vue'
import { datacenterLocale } from '@/i18n/runtime'

const props = defineProps<{
    modelValue: boolean
    title?: string
    successCount: number
    failed: CollectorPointBatchFailure[]
  }>()
const ui = (zh: string, en: string) => (datacenterLocale.value === 'en' ? en : zh)
const title = computed(() => props.title || ui('批量新增结果', 'Batch Create Results'))
const emit = defineEmits<{ 'update:modelValue': [value: boolean] }>()
</script>

<style scoped>
.collector-batch-result__table {
  margin-top: 16px;
}
</style>
