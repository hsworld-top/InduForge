<template>
  <el-dialog
    :model-value="modelValue"
    title="批量导入采集点"
    width="820px"
    @close="emit('update:modelValue', false)"
  >
    <el-upload drag :auto-upload="false" :limit="1" :on-change="onFileChange"
      ><div>拖入 CSV、TSV 或 XLSX 文件</div>
      <small>最大 20MB，最多 10000 行</small></el-upload
    >
    <div class="collector-import__actions">
      <el-button type="primary" :disabled="!file" :loading="loading" @click="preview"
        >生成预览</el-button
      >
    </div>
    <template v-if="result"
      ><el-alert
        :title="`共 ${result.totalRows} 行，可导入 ${result.validRows} 行`"
        :type="result.errors.length ? 'warning' : 'success'"
        show-icon /><el-table :data="result.errors" max-height="260"
        ><el-table-column prop="row" label="行" width="70" /><el-table-column
          prop="field"
          label="字段"
          width="130" /><el-table-column prop="message" label="错误" /></el-table
    ></template>
    <template #footer
      ><el-button @click="emit('update:modelValue', false)">取消</el-button
      ><el-button
        type="primary"
        :disabled="!result || result.errors.length > 0"
        :loading="committing"
        @click="commit"
        >确认导入</el-button
      ></template
    >
  </el-dialog>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { ElMessage } from 'element-plus'
import type { UploadFile } from 'element-plus'
import { commitCollectorPointImport, previewCollectorPointImport } from '@/api/collector.api'
import type { CollectorImportPreview } from '@/api/schemas/collector.schema'
const props = defineProps<{ modelValue: boolean; projectId: string; connectionId: string }>()
const emit = defineEmits<{ 'update:modelValue': [value: boolean]; committed: [] }>()
const file = ref<File | null>(null)
const result = ref<CollectorImportPreview | null>(null)
const loading = ref(false)
const committing = ref(false)
function onFileChange(uploadFile: UploadFile) {
  file.value = uploadFile.raw || null
  result.value = null
}
async function preview() {
  if (!file.value) return
  loading.value = true
  try {
    result.value = await previewCollectorPointImport(
      props.projectId,
      props.connectionId,
      file.value,
    )
  } finally {
    loading.value = false
  }
}
async function commit() {
  if (!result.value) return
  committing.value = true
  try {
    await commitCollectorPointImport(props.projectId, props.connectionId, result.value.importId)
    ElMessage.success('点位导入完成')
    emit('committed')
    emit('update:modelValue', false)
  } finally {
    committing.value = false
  }
}
</script>

<style scoped>
.collector-import__actions {
  display: flex;
  justify-content: flex-end;
  margin: 14px 0;
}
</style>
