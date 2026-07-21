<template>
  <el-dialog
    :model-value="modelValue"
    :title="title"
    width="680px"
    @close="emit('update:modelValue', false)"
  >
    <el-alert
      :title="`成功 ${successCount} 个，失败 ${failed.length} 个`"
      :type="successCount > 0 ? 'warning' : 'error'"
      show-icon
      :closable="false"
    />
    <el-table :data="failed" max-height="360" class="collector-batch-result__table">
      <el-table-column prop="index" label="序号" width="72">
        <template #default="scope">{{ scope.row.index + 1 }}</template>
      </el-table-column>
      <el-table-column prop="name" label="变量名称" min-width="160" show-overflow-tooltip />
      <el-table-column prop="message" label="失败原因" min-width="280" show-overflow-tooltip />
    </el-table>
    <template #footer>
      <el-button type="primary" @click="emit('update:modelValue', false)">知道了</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import type { CollectorPointBatchFailure } from '@/api/schemas/collector.schema'

withDefaults(
  defineProps<{
    modelValue: boolean
    title?: string
    successCount: number
    failed: CollectorPointBatchFailure[]
  }>(),
  { title: '批量新增结果' },
)
const emit = defineEmits<{ 'update:modelValue': [value: boolean] }>()
</script>

<style scoped>
.collector-batch-result__table {
  margin-top: 16px;
}
</style>
