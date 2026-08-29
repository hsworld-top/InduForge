<template>
  <DcDialog v-model="visible" :title="t('alarmImport.title')" width="720px">
    <el-steps :active="step" finish-status="success" simple
      ><el-step :title="t('alarmImport.selectFile')" /><el-step :title="t('alarmImport.validatePreview')" /><el-step :title="t('alarmImport.confirmImport')"
    /></el-steps>
    <section class="alarm-import">
      <template v-if="step === 0">
        <input ref="fileInput" type="file" accept=".xlsx" @change="selectFile" />
        <p>{{ t('alarmImport.scopeHint') }}</p>
      </template>
      <template v-else-if="preview">
        <div class="alarm-import__summary">
          <span
            >{{ t('alarmImport.created') }} <strong>{{ preview.createCount }}</strong></span
          ><span
            >{{ t('alarmImport.updated') }} <strong>{{ preview.updateCount }}</strong></span
          ><span
            >{{ t('alarmImport.unchanged') }} <strong>{{ preview.unchangedCount }}</strong></span
          ><span
            >{{ t('alarmImport.errors') }} <strong>{{ preview.errorCount }}</strong></span
          ><span
            >{{ t('alarmImport.warnings') }} <strong>{{ preview.warningCount }}</strong></span
          >
        </div>
        <el-table
          v-if="preview.issues.length"
          :data="preview.issues"
          height="230"
          :row-class-name="issueRowClass"
          ><el-table-column prop="sheet" :label="t('alarmImport.sheet')" width="110" /><el-table-column
            prop="row"
            :label="t('alarmImport.row')"
            width="64" /><el-table-column :label="t('alarmImport.type')" width="80"
            ><template #default="{ row }"
              ><el-tag :type="row.type === 'error' ? 'danger' : 'warning'" size="small">{{
                row.type === 'error' ? t('alarmImport.errors') : t('alarmImport.warnings')
              }}</el-tag></template
            ></el-table-column
          ><el-table-column prop="message" :label="t('alarmImport.issue')" min-width="300"
        /></el-table>
        <el-checkbox
          v-if="preview.warningCount > 0"
          v-model="warningsConfirmed"
          class="alarm-import__warning-confirm"
        >
          {{ t('alarmImport.confirmWarnings') }}
        </el-checkbox>
        <button
          v-if="preview.issues.length"
          type="button"
          class="alarm-import__error-download"
          @click="downloadErrors"
        >
          {{ t('alarmImport.downloadIssues') }}
        </button>
        <el-empty v-else :image-size="52" :description="t('alarmImport.valid')" />
      </template>
    </section>
    <template #footer
      ><button type="button" class="dc-button" @click="visible = false">{{ t('alarmImport.cancel') }}</button
      ><button
        v-if="step === 0"
        type="button"
        class="dc-button dc-button--primary"
        :disabled="!file || loading"
        @click="previewFile"
      >
        {{ t('alarmImport.validate') }}</button
      ><button
        v-else
        type="button"
        class="dc-button dc-button--primary"
        :disabled="
          !preview ||
          preview.errorCount > 0 ||
          (preview.warningCount > 0 && !warningsConfirmed) ||
          loading
        "
        @click="apply"
      >
        {{ t('alarmImport.confirm') }}
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
import { t } from '@/i18n/runtime'
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
const warningsConfirmed = ref(false)
watch(
  () => props.modelValue,
  (opened) => {
    if (opened) {
      step.value = 0
      file.value = null
      preview.value = null
      warningsConfirmed.value = false
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
    ElMessage.error(getApiErrorMessage(error, t('alarmImport.validationFailed')))
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
      warningsConfirmed.value ? preview.value.warningKeys : [],
    )
    step.value = 2
    ElMessage.success(t('alarmImport.imported', { count: result.affectedCount }))
    emit('imported', result.affectedCount)
    visible.value = false
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, t('alarmImport.importFailed')))
  } finally {
    loading.value = false
  }
}
function issueRowClass({ row }: { row: { type: string } }) {
  return row.type === 'error' ? 'is-error' : 'is-warning'
}
async function downloadErrors() {
  if (!file.value) return
  try {
    const blob = await downloadAlarmImportErrors(props.projectId, file.value)
    const url = URL.createObjectURL(blob)
    const anchor = document.createElement('a')
    anchor.href = url
    anchor.download = t('alarmImport.issueFilename')
    anchor.click()
    URL.revokeObjectURL(url)
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, t('alarmImport.downloadFailed')))
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
.alarm-import__warning-confirm {
  margin-top: 12px;
}
:deep(.el-table__row.is-error) {
  --el-table-tr-bg-color: color-mix(in srgb, var(--el-color-danger) 7%, transparent);
}
:deep(.el-table__row.is-warning) {
  --el-table-tr-bg-color: color-mix(in srgb, var(--el-color-warning) 9%, transparent);
}
</style>
