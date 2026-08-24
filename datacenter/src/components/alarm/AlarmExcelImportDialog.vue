<template>
  <DcDialog v-model="visible" title="Excel 导入报警项" width="720px">
    <el-steps :active="step" finish-status="success" simple
      ><el-step title="选择文件" /><el-step title="校验预览" /><el-step title="确认导入"
    /></el-steps>
    <section class="alarm-import">
      <template v-if="step === 0">
        <input ref="fileInput" type="file" accept=".xlsx" @change="selectFile" />
        <p>仅新增和更新普通报警。缺失行不会删除报警，组合报警不参与 Excel 导入。</p>
      </template>
      <template v-else-if="preview">
        <div class="alarm-import__summary">
          <span
            >新增 <strong>{{ preview.createCount }}</strong></span
          ><span
            >更新 <strong>{{ preview.updateCount }}</strong></span
          ><span
            >无变化 <strong>{{ preview.unchangedCount }}</strong></span
          ><span
            >错误 <strong>{{ preview.errorCount }}</strong></span
          ><span
            >警告 <strong>{{ preview.warningCount }}</strong></span
          >
        </div>
        <el-table v-if="preview.issues.length" :data="preview.issues" height="230"
          ><el-table-column prop="sheet" label="工作表" width="110" /><el-table-column
            prop="row"
            label="行"
            width="64" /><el-table-column prop="message" label="问题" min-width="360"
        /></el-table>
        <button
          v-if="preview.issues.length"
          type="button"
          class="alarm-import__error-download"
          @click="downloadErrors"
        >
          下载带问题说明的工作簿
        </button>
        <el-empty v-else :image-size="52" description="校验通过，可以导入" />
      </template>
    </section>
    <template #footer
      ><button type="button" class="dc-button" @click="visible = false">取消</button
      ><button
        v-if="step === 0"
        type="button"
        class="dc-button dc-button--primary"
        :disabled="!file || loading"
        @click="previewFile"
      >
        校验文件</button
      ><button
        v-else
        type="button"
        class="dc-button dc-button--primary"
        :disabled="!preview || preview.errorCount > 0 || loading"
        @click="apply"
      >
        确认导入
      </button></template
    >
  </DcDialog>
</template>
<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import DcDialog from '@/components/shared/DcDialog.vue'
import { applyAlarmImport, downloadAlarmImportErrors, previewAlarmImport } from '@/api/alarm.api'
import type { AlarmExcelPreview } from '@/api/schemas/alarm.schema'
import { getApiErrorMessage } from '@/utils/request'
const props = defineProps<{ modelValue: boolean; projectId: string }>()
const emit = defineEmits<{ 'update:modelValue': [value: boolean]; imported: [count: number] }>()
const visible = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value),
})
const step = ref(0)
const file = ref<File | null>(null)
const preview = ref<AlarmExcelPreview | null>(null)
const loading = ref(false)
watch(
  () => props.modelValue,
  (opened) => {
    if (opened) {
      step.value = 0
      file.value = null
      preview.value = null
    }
  },
)
function selectFile(event: Event) {
  file.value = (event.target as HTMLInputElement).files?.[0] || null
}
async function previewFile() {
  if (!file.value) return
  loading.value = true
  try {
    preview.value = await previewAlarmImport(props.projectId, file.value)
    step.value = 1
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '校验导入文件失败'))
  } finally {
    loading.value = false
  }
}
async function apply() {
  if (!file.value || !preview.value) return
  loading.value = true
  try {
    const result = await applyAlarmImport(
      props.projectId,
      file.value,
      preview.value.digest,
      preview.value.warningKeys,
    )
    step.value = 2
    ElMessage.success(`已导入 ${result.affectedCount} 条报警项`)
    emit('imported', result.affectedCount)
    visible.value = false
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '导入报警项失败'))
  } finally {
    loading.value = false
  }
}
async function downloadErrors() {
  if (!file.value) return
  try {
    const blob = await downloadAlarmImportErrors(props.projectId, file.value)
    const url = URL.createObjectURL(blob)
    const anchor = document.createElement('a')
    anchor.href = url
    anchor.download = '报警项-导入问题.xlsx'
    anchor.click()
    URL.revokeObjectURL(url)
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '下载问题工作簿失败'))
  }
}
</script>
<style scoped>
.alarm-import {
  min-height: 300px;
  padding: 24px 4px 0;
}
.alarm-import p {
  color: var(--dc-text-secondary);
}
.alarm-import__summary {
  display: grid;
  grid-template-columns: repeat(5, minmax(0, 1fr));
  gap: 8px;
  margin-bottom: 12px;
}
.alarm-import__summary span {
  padding: 12px;
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-muted);
  text-align: center;
}
.alarm-import__error-download {
  margin-top: 10px;
  border: 0;
  background: transparent;
  color: var(--dc-primary);
}
</style>
