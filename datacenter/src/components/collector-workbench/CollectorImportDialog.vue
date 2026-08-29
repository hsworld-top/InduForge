<template>
  <el-dialog
    :model-value="modelValue"
    :title="ui('批量导入采集点', 'Import Collection Points')"
    width="820px"
    @close="emit('update:modelValue', false)"
  >
    <div class="collector-import__intro">
      <span>{{ ui('先下载当前驱动模板，填写后再导入。', 'Download the driver template, complete it, and then import the file.') }}</span>
      <el-dropdown split-button :loading="downloading" @click="downloadTemplate('xlsx')">
        {{ ui('下载 XLSX 模板', 'Download XLSX Template') }}
        <template #dropdown>
          <el-dropdown-menu>
            <el-dropdown-item @click="downloadTemplate('csv')">{{ ui('下载 CSV 模板', 'Download CSV Template') }}</el-dropdown-item>
          </el-dropdown-menu>
        </template>
      </el-dropdown>
    </div>
    <el-upload drag :auto-upload="false" :limit="1" :on-change="onFileChange"
      ><div>{{ ui('拖入 CSV、TSV 或 XLSX 文件', 'Drop a CSV, TSV, or XLSX file here') }}</div>
      <small>{{ ui('最大 20MB，最多 10000 行', 'Up to 20 MB and 10,000 rows') }}</small></el-upload
    >
    <div class="collector-import__actions">
      <el-button type="primary" :disabled="!file" :loading="loading" @click="preview"
        >{{ ui('生成预览', 'Generate Preview') }}</el-button
      >
    </div>
    <template v-if="result"
      ><el-alert
        :title="ui(`共 ${result.totalRows} 行，可导入 ${result.validRows} 行`, `${result.totalRows} rows, ${result.validRows} ready to import`)"
        :type="result.errors.length ? 'warning' : 'success'"
        show-icon /><el-table :data="result.errors" max-height="260"
        ><el-table-column prop="row" :label="ui('行', 'Row')" width="70" /><el-table-column
          prop="field"
          :label="ui('字段', 'Field')"
          width="130" /><el-table-column prop="message" :label="ui('错误', 'Error')" /></el-table
    ></template>
    <template #footer
      ><el-button @click="emit('update:modelValue', false)">{{ ui('取消', 'Cancel') }}</el-button
      ><el-button
        type="primary"
        :disabled="!result || result.errors.length > 0"
        :loading="committing"
        @click="commit"
        >{{ ui('确认导入', 'Import') }}</el-button
      ></template
    >
  </el-dialog>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { ElMessage } from 'element-plus'
import type { UploadFile } from 'element-plus'
import {
  commitCollectorPointImport,
  downloadCollectorPointImportTemplate,
  previewCollectorPointImport,
} from '@/api/collector.api'
import type { CollectorImportPreview } from '@/api/schemas/collector.schema'
import { datacenterLocale } from '@/i18n/runtime'

const ui = (zh: string, en: string) => (datacenterLocale.value === 'en' ? en : zh)
const props = defineProps<{
  modelValue: boolean
  projectId: string
  connectionId: string
  driverId: string
}>()
const emit = defineEmits<{ 'update:modelValue': [value: boolean]; committed: [] }>()
const file = ref<File | null>(null)
const result = ref<CollectorImportPreview | null>(null)
const loading = ref(false)
const committing = ref(false)
const downloading = ref(false)
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
async function downloadTemplate(format: 'csv' | 'xlsx') {
  downloading.value = true
  try {
    const blob = await downloadCollectorPointImportTemplate(props.driverId, format)
    const url = URL.createObjectURL(blob)
    const anchor = document.createElement('a')
    anchor.href = url
    anchor.download = `${props.driverId}-points.${format}`
    anchor.click()
    URL.revokeObjectURL(url)
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : ui('模板下载失败', 'Failed to download the template'))
  } finally {
    downloading.value = false
  }
}
async function commit() {
  if (!result.value) return
  committing.value = true
  try {
    await commitCollectorPointImport(props.projectId, props.connectionId, result.value.importId)
    ElMessage.success(ui('点位导入完成', 'Points imported'))
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
.collector-import__intro {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 14px;
  color: var(--dc-text-secondary);
  font-size: 13px;
}
</style>
