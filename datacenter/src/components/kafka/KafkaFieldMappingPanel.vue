<template>
  <section class="kafka-variable-panel">
    <header class="kafka-variable-panel__toolbar">
      <div class="kafka-variable-panel__title">
        <strong>{{ mapping.name || mapping.topic }} · {{ ui('字段映射', 'Field Mappings') }}</strong>
        <span>{{ mapping.topic }}</span>
      </div>
      <div class="kafka-variable-panel__actions">
        <el-button type="primary" size="small" @click="openCreateDialog">
          <IconTablerPlus class="kafka-variable-panel__button-icon" />
          {{ ui('新建映射', 'New Mapping') }}
        </el-button>
        <el-button size="small" @click="sampleEditorVisible = !sampleEditorVisible">
          <IconTablerBraces class="kafka-variable-panel__button-icon" />
          {{ ui('样例编辑器', 'Sample Editor') }}
        </el-button>
        <el-button size="small" :loading="loading" @click="reloadAll">
          <IconTablerRefresh class="kafka-variable-panel__button-icon" />
          {{ ui('刷新', 'Refresh') }}
        </el-button>
      </div>
    </header>

    <div class="kafka-variable-panel__body" :class="{ 'has-editor': sampleEditorVisible }">
      <div class="kafka-variable-panel__result">
        <el-table v-loading="loading" :data="fields" height="100%" :empty-text="ui('暂无字段映射', 'No field mappings')">
          <el-table-column prop="name" :label="ui('映射名称', 'Mapping Name')" min-width="140" show-overflow-tooltip />
          <el-table-column :label="ui('字段路径', 'Field Path')" min-width="170" show-overflow-tooltip>
            <template #default="{ row }">{{ formatKafkaValuePath(row.valuePath) }}</template>
          </el-table-column>
          <el-table-column prop="dataType" :label="ui('类型', 'Type')" width="92" />
          <el-table-column
            prop="dataPointPath"
            :label="ui('输出数据点', 'Output Data Point')"
            min-width="190"
            show-overflow-tooltip
          >
            <template #default="{ row }">
              {{ row.dataPointPath || '-' }}
            </template>
          </el-table-column>
          <el-table-column :label="ui('最近测试值', 'Latest Test Value')" min-width="140" show-overflow-tooltip>
            <template #default="{ row }">
              <span class="kafka-variable-panel__mono">{{ formatLastValue(row.lastValue) }}</span>
            </template>
          </el-table-column>
          <el-table-column :label="ui('质量', 'Quality')" width="92">
            <template #default="{ row }">
              <WorkbenchStatusPill
                :label="qualityLabel(row.quality)"
                :tone="qualityTone(row.quality)"
              />
            </template>
          </el-table-column>
          <el-table-column :label="ui('更新时间', 'Updated At')" width="156">
            <template #default="{ row }">
              {{ formatTime(row.lastUpdatedAt) }}
            </template>
          </el-table-column>
          <el-table-column :label="ui('状态', 'Status')" width="86">
            <template #default="{ row }">
              <WorkbenchStatusPill
                :label="row.enabled ? ui('启用', 'Enabled') : ui('停用', 'Disabled')"
                :tone="row.enabled ? 'success' : 'neutral'"
              />
            </template>
          </el-table-column>
          <el-table-column :label="ui('操作', 'Actions')" width="150" fixed="right">
            <template #default="{ row }">
              <div class="kafka-variable-panel__row-actions">
                <el-tooltip :content="ui('编辑', 'Edit')" placement="top">
                  <button
                    type="button"
                    class="kafka-variable-panel__icon-action"
                    @click="openEditDialog(row)"
                  >
                    <IconTablerEdit />
                  </button>
                </el-tooltip>
                <el-tooltip :content="row.enabled ? ui('停用', 'Disable') : ui('启用', 'Enable')" placement="top">
                  <button
                    type="button"
                    class="kafka-variable-panel__icon-action"
                    @click="toggleField(row)"
                  >
                    <IconTablerPlayerPause v-if="row.enabled" />
                    <IconTablerPlayerPlay v-else />
                  </button>
                </el-tooltip>
                <el-tooltip :content="ui('删除', 'Delete')" placement="top">
                  <button
                    type="button"
                    class="kafka-variable-panel__icon-action is-danger"
                    @click="deleteField(row)"
                  >
                    <IconTablerTrash />
                  </button>
                </el-tooltip>
              </div>
            </template>
          </el-table-column>
        </el-table>
        <DataCenterPagination
          :page="pagination.page"
          :page-size="pagination.pageSize"
          :total="pagination.total"
          :total-pages="pagination.totalPages"
          @change="handlePageChange"
        />
      </div>

      <div v-if="sampleEditorVisible" class="kafka-variable-panel__editor">
        <WorkbenchJsonSampleEditor
          ref="sampleEditorRef"
          v-model="sampleEditorText"
          :fill-latest-text="ui('拉取样本', 'Pull Sample')"
          :fill-latest-disabled="previewing"
          :error="sampleParseError"
          @fill-latest="pullSamples"
          @parse="parseEditorFields"
        />
      </div>
    </div>

    <DcDialog
      v-model="dialogVisible"
      :title="editingField ? ui('编辑映射', 'Edit Mapping') : ui('新建映射', 'New Mapping')"
      width="560px"
      :close-disabled="fieldSaving"
    >
      <el-form class="kafka-variable-panel__form" label-position="top">
        <el-form-item :label="ui('映射名称', 'Mapping Name')">
          <el-input v-model="form.name" placeholder="temperature" />
        </el-form-item>
        <el-form-item :label="ui('字段路径', 'Field Path')">
          <el-input :model-value="formatKafkaValuePath(form.valuePath)" readonly />
        </el-form-item>
        <div class="kafka-variable-panel__form-grid">
          <el-form-item :label="ui('类型', 'Type')">
            <el-select v-model="form.dataType">
              <el-option label="string" value="string" />
              <el-option label="float64" value="float64" />
              <el-option label="bool" value="bool" />
              <el-option label="object" value="object" />
              <el-option label="array" value="array" />
            </el-select>
          </el-form-item>
          <el-form-item :label="ui('状态', 'Status')">
            <el-switch v-model="form.enabled" :active-text="ui('启用', 'Enabled')" :inactive-text="ui('停用', 'Disabled')" />
          </el-form-item>
        </div>
        <el-form-item :label="ui('描述', 'Description')">
          <el-input v-model="form.description" type="textarea" :rows="3" :placeholder="ui('可选', 'Optional')" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button :disabled="fieldSaving" @click="dialogVisible = false">{{ ui('取消', 'Cancel') }}</el-button>
        <el-button type="primary" :loading="fieldSaving" :disabled="!canSubmit" @click="saveField">
          {{ ui('保存', 'Save') }}
        </el-button>
      </template>
    </DcDialog>

    <DcDialog
      v-model="candidateDialogVisible"
      :title="ui('解析字段候选', 'Parsed Field Candidates')"
      width="760px"
      :close-disabled="batchSaving"
    >
      <el-table
        class="kafka-variable-panel__candidate-table"
        :data="candidateRows"
        max-height="420"
        row-key="path"
        :empty-text="ui('暂无字段候选', 'No field candidates')"
      >
        <el-table-column label="" width="42">
          <template #default="{ row }">
            <el-checkbox
              :model-value="selectedCandidatePaths.has(row.path)"
              :disabled="row.exists"
              @update:model-value="toggleCandidateSelection(row, Boolean($event))"
            />
          </template>
        </el-table-column>
        <el-table-column prop="name" :label="ui('映射名称', 'Mapping Name')" min-width="120" />
        <el-table-column prop="path" :label="ui('字段路径', 'Field Path')" min-width="150" show-overflow-tooltip />
        <el-table-column prop="dataType" :label="ui('类型', 'Type')" width="88" />
        <el-table-column :label="ui('样例值', 'Sample Value')" min-width="120" show-overflow-tooltip>
          <template #default="{ row }">
            <span class="kafka-variable-panel__mono">{{ formatLastValue(row.sampleValue) }}</span>
          </template>
        </el-table-column>
        <el-table-column :label="ui('状态', 'Status')" width="88">
          <template #default="{ row }">
            <WorkbenchStatusPill
              :label="row.exists ? ui('已存在', 'Exists') : ui('候选', 'Candidate')"
              :tone="row.exists ? 'neutral' : 'info'"
            />
          </template>
        </el-table-column>
      </el-table>
      <template #footer>
        <el-button :disabled="batchSaving" @click="candidateDialogVisible = false">{{ ui('取消', 'Cancel') }}</el-button>
        <el-button
          type="primary"
          :loading="batchSaving"
          :disabled="selectedCandidateCount === 0"
          @click="createFromSamples"
        >
          {{ ui('保存选中字段', 'Save Selected Fields') }}
        </el-button>
      </template>
    </DcDialog>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import dayjs from 'dayjs'
import IconTablerBraces from '~icons/tabler/braces'
import IconTablerEdit from '~icons/tabler/edit'
import IconTablerPlayerPause from '~icons/tabler/player-pause'
import IconTablerPlayerPlay from '~icons/tabler/player-play'
import IconTablerPlus from '~icons/tabler/plus'
import IconTablerRefresh from '~icons/tabler/refresh'
import IconTablerTrash from '~icons/tabler/trash'
import dataAPI from '@/api/data.api'
import DcDialog from '@/components/shared/DcDialog.vue'
import DataCenterPagination from '@/components/shared/DataCenterPagination.vue'
import WorkbenchJsonSampleEditor from '@/components/workbench/WorkbenchJsonSampleEditor.vue'
import WorkbenchStatusPill from '@/components/workbench/WorkbenchStatusPill.vue'
import { TIME_FORMAT } from '@/constants'
import { getApiErrorMessage } from '@/utils/request'
import { datacenterLocale } from '@/i18n/runtime'
import {
  formatKafkaValuePath,
  inferKafkaSampleFields,
  normalizeKafkaSampleEditorText,
  type KafkaSampleFieldCandidate,
} from './kafkaSampleFields'
import type { KafkaField, KafkaPreview, KafkaPreviewSample, KafkaTopicMapping } from './types'

const ui = (zh: string, en: string) => (datacenterLocale.value === 'en' ? en : zh)

const props = defineProps<{
  projectId: string
  mapping: KafkaTopicMapping
  samples: KafkaPreviewSample[]
}>()

const emit = defineEmits<{
  (
    event: 'samples',
    payload: { mappingId: string; samples: KafkaPreviewSample[]; preview: KafkaPreview },
  ): void
}>()

const fields = ref<KafkaField[]>([])
const loading = ref(false)
const previewing = ref(false)
const batchSaving = ref(false)
const fieldSaving = ref(false)
const dialogVisible = ref(false)
const candidateDialogVisible = ref(false)
const editingField = ref<KafkaField | null>(null)
const pagination = ref({ page: 1, pageSize: 20, total: 0, totalPages: 0 })
const sampleEditorRef = ref<InstanceType<typeof WorkbenchJsonSampleEditor> | null>(null)
const sampleEditorVisible = ref(true)
const sampleEditorText = ref('')
const sampleParseError = ref('')
const selectedCandidatePaths = ref<Set<string>>(new Set())
const candidateRows = ref<KafkaSampleFieldCandidate[]>([])

const form = reactive({
  name: '',
  valuePath: [] as Array<string | number>,
  dataType: 'string',
  enabled: true,
  description: '',
})

const canSubmit = computed(() => form.name.trim().length > 0 && form.valuePath.length > 0)
const selectedCandidateCount = computed(() => selectedCandidatePaths.value.size)
const sampleEditorStorageKey = computed(
  () => `datacenter:kafka-field-sample:${props.projectId}:${props.mapping.id}`,
)

const loadFields = async () => {
  loading.value = true
  try {
    const res = await dataAPI.getKafkaFields(props.projectId, props.mapping.id, {
      page: pagination.value.page,
      pageSize: pagination.value.pageSize,
    })
    fields.value = res.list || []
    pagination.value = {
      page: res.pagination?.page || pagination.value.page,
      pageSize: res.pagination?.pageSize || pagination.value.pageSize,
      total: res.pagination?.total || 0,
      totalPages: res.pagination?.totalPages || 0,
    }
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, ui('加载 Kafka 字段映射失败', 'Failed to load Kafka field mappings')))
  } finally {
    loading.value = false
  }
}

const reloadAll = async () => {
  await loadFields()
}

const reloadAfterMutation = async () => {
  await loadFields()
  if (pagination.value.page > 1 && fields.value.length === 0 && pagination.value.total > 0) {
    pagination.value.page -= 1
    await loadFields()
  }
}

const changePage = async (page: number) => {
  pagination.value.page = page
  await loadFields()
}

const changePageSize = async (pageSize: number) => {
  pagination.value.pageSize = pageSize
  pagination.value.page = 1
  await loadFields()
}

const handlePageChange = async (value: { page: number; pageSize: number }) => {
  if (value.pageSize !== pagination.value.pageSize) {
    await changePageSize(value.pageSize)
    return
  }
  await changePage(value.page)
}

const openCreateDialog = () => {
  if (!sampleEditorText.value.trim()) {
    sampleEditorVisible.value = true
    ElMessage.info(ui('请先填入或拉取 JSON 样例，再从字段候选中选择映射', 'Enter or pull a JSON sample, then select fields to map'))
    return
  }
  parseEditorFields()
}

const openEditDialog = (field: KafkaField) => {
  editingField.value = field
  Object.assign(form, {
    name: field.name,
    valuePath: field.valuePath,
    dataType: field.dataType || 'string',
    enabled: field.enabled,
    description: field.description || '',
  })
  dialogVisible.value = true
}

const saveField = async () => {
  if (!canSubmit.value) return
  fieldSaving.value = true
  try {
    const payload = {
      groupId: null,
      name: form.name.trim(),
      valuePath: [...form.valuePath],
      dataType: form.dataType,
      enabled: form.enabled,
      description: form.description.trim(),
    }
    if (editingField.value) {
      await dataAPI.updateKafkaField(props.projectId, editingField.value.id, payload)
    } else {
      await dataAPI.createKafkaField(props.projectId, props.mapping.id, payload)
    }
    fieldSaving.value = false
    dialogVisible.value = false
    ElMessage.success(ui('字段映射已保存', 'Field mapping saved'))
    candidateDialogVisible.value = false
    await reloadAfterMutation()
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, ui('保存 Kafka 字段映射失败', 'Failed to save Kafka field mapping')))
  } finally {
    fieldSaving.value = false
  }
}

const createFromSamples = async () => {
  const sourceCandidates = candidateRows.value
  const selectedPaths = selectedCandidatePaths.value
  if (selectedPaths.size === 0) {
    ElMessage.warning(ui('请先选择要保存的字段候选', 'Select field candidates to save'))
    return
  }
  const nextCandidates = sourceCandidates.filter(
    (candidate) => !candidate.exists && selectedPaths.has(candidate.path),
  )
  if (nextCandidates.length === 0) {
    ElMessage.warning(ui('暂无可保存的字段候选，请先解析 JSON 样例', 'No field candidates are available. Parse a JSON sample first.'))
    return
  }
  batchSaving.value = true
  try {
    await dataAPI.createKafkaFieldsBatch(
      props.projectId,
      props.mapping.id,
      nextCandidates.map((candidate) => ({
        name: candidate.name,
        valuePath: candidate.segments,
        dataType: candidate.dataType,
        enabled: true,
        groupId: null,
      })),
    )
    ElMessage.success(ui('字段映射已保存', 'Field mappings saved'))
    candidateDialogVisible.value = false
    await reloadAfterMutation()
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, ui('保存字段映射失败', 'Failed to save field mappings')))
  } finally {
    batchSaving.value = false
  }
}

const pullSamples = async () => {
  previewing.value = true
  try {
    const preview = await dataAPI.previewKafkaTopicMapping(props.projectId, props.mapping.id, {
      limit: props.mapping.sampleLimit,
      timeoutMs: props.mapping.timeoutMs,
      decode: props.mapping.decode,
    })
    const samples = preview.samples || []
    const firstSample = samples[0]
    if (firstSample) {
      sampleEditorText.value = normalizeKafkaSampleEditorText(firstSample)
      saveSampleEditorDraft()
    }
    emit('samples', {
      mappingId: String(props.mapping.id),
      samples,
      preview,
    })
    ElMessage.success(firstSample ? ui('样本已拉取并填入编辑器', 'Sample pulled into the editor') : ui('本次未拉取到样本', 'No sample was returned'))
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, ui('Kafka 样本拉取失败', 'Failed to pull Kafka sample')))
  } finally {
    previewing.value = false
  }
}

const parseEditorFields = () => {
  sampleParseError.value = ''
  saveSampleEditorDraft()
  try {
    const parsed = JSON.parse(sampleEditorText.value)
    candidateRows.value = inferKafkaSampleFields(
      [{ value: parsed }],
      fields.value.map((field) => field.valuePath),
    )
    selectedCandidatePaths.value = new Set(
      candidateRows.value
        .filter((candidate) => !candidate.exists)
        .map((candidate) => candidate.path),
    )
    candidateDialogVisible.value = true
    ElMessage.success(ui(`已解析 ${candidateRows.value.length} 个字段候选`, `Parsed ${candidateRows.value.length} field candidate${candidateRows.value.length === 1 ? '' : 's'}`))
  } catch {
    sampleParseError.value = ui('JSON 样例格式无效', 'Invalid JSON sample')
  }
}

const toggleCandidateSelection = (candidate: KafkaSampleFieldCandidate, checked: boolean) => {
  const next = new Set(selectedCandidatePaths.value)
  if (checked) {
    next.add(candidate.path)
  } else {
    next.delete(candidate.path)
  }
  selectedCandidatePaths.value = next
}

const loadSampleEditorDraft = () => {
  sampleEditorText.value = window.localStorage.getItem(sampleEditorStorageKey.value) || ''
}

const saveSampleEditorDraft = () => {
  const value = sampleEditorText.value
  if (value) {
    window.localStorage.setItem(sampleEditorStorageKey.value, value)
  } else {
    window.localStorage.removeItem(sampleEditorStorageKey.value)
  }
}

const toggleField = async (field: KafkaField) => {
  try {
    await dataAPI.toggleKafkaField(props.projectId, field.id, !field.enabled)
    ElMessage.success(field.enabled ? ui('映射已停用', 'Mapping disabled') : ui('映射已启用', 'Mapping enabled'))
    await loadFields()
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, ui('更新映射状态失败', 'Failed to update mapping status')))
  }
}

const deleteField = async (field: KafkaField) => {
  try {
    await ElMessageBox.confirm(ui(`删除映射“${field.name}”？对应数据点将标记为失效。`, `Delete mapping “${field.name}”? Its data point will be marked invalid.`), ui('删除映射', 'Delete Mapping'), {
      confirmButtonText: ui('删除', 'Delete'),
      cancelButtonText: ui('取消', 'Cancel'),
      type: 'warning',
    })
    await dataAPI.deleteKafkaField(props.projectId, field.id)
    ElMessage.success(ui('映射已删除', 'Mapping deleted'))
    await reloadAfterMutation()
  } catch (error) {
    if (error === 'cancel' || error === 'close') return
    ElMessage.error(getApiErrorMessage(error, ui('删除 Kafka 字段映射失败', 'Failed to delete Kafka field mapping')))
  }
}

const formatLastValue = (value: unknown) => {
  if (value === null || value === undefined) return '-'
  if (typeof value === 'string') return value
  try {
    return JSON.stringify(value)
  } catch {
    return String(value)
  }
}

const formatTime = (value?: string | null) => {
  if (!value) return '-'
  const parsed = dayjs(value)
  return parsed.isValid() ? parsed.format(TIME_FORMAT) : '-'
}

const qualityLabel = (quality?: string) => {
  if (quality === 'good') return ui('良好', 'Good')
  if (quality === 'bad') return ui('异常', 'Bad')
  return ui('未知', 'Unknown')
}

const qualityTone = (quality?: string) => {
  if (quality === 'good') return 'success'
  if (quality === 'bad') return 'danger'
  return 'neutral'
}

onMounted(() => {
  loadSampleEditorDraft()
  void reloadAll()
})

watch(
  () => props.mapping.id,
  () => {
    pagination.value.page = 1
    loadSampleEditorDraft()
    void reloadAll()
  },
)
</script>

<style scoped>
.kafka-variable-panel {
  height: 100%;
  min-height: 0;
  display: flex;
  flex-direction: column;
  background: var(--dc-surface-raised);
}

.kafka-variable-panel__toolbar {
  min-height: 48px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 8px 12px;
  border-bottom: 1px solid var(--dc-border);
  background: var(--dc-surface-subtle);
}

.kafka-variable-panel__title {
  min-width: 0;
  display: grid;
  gap: 3px;
}

.kafka-variable-panel__title strong,
.kafka-variable-panel__title span {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.kafka-variable-panel__title strong {
  color: var(--dc-text);
  font-size: 13px;
}

.kafka-variable-panel__title span,
.kafka-variable-panel__mono {
  color: var(--dc-text-muted);
  font-family: var(--dc-font-mono, ui-monospace, SFMono-Regular, Menlo, Consolas, monospace);
  font-size: 11px;
}

.kafka-variable-panel__actions {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}

.kafka-variable-panel__button-icon {
  width: 14px;
  height: 14px;
  margin-right: 4px;
}

.kafka-variable-panel__body {
  min-height: 0;
  flex: 1;
  display: grid;
  grid-template-columns: minmax(0, 1fr);
}

.kafka-variable-panel__body.has-editor {
  grid-template-columns: minmax(0, 1fr) minmax(340px, 420px);
}

.kafka-variable-panel__result,
.kafka-variable-panel__editor {
  min-height: 0;
  min-width: 0;
}

.kafka-variable-panel__result {
  position: relative;
  display: flex;
  flex-direction: column;
}

.kafka-variable-panel__result > :deep(.el-table) {
  flex: 1;
  min-height: 0;
}

.kafka-variable-panel__editor {
  display: grid;
  grid-template-rows: minmax(220px, 1fr);
  gap: 8px;
  padding: 10px;
  border-left: 1px solid var(--dc-border);
  background: var(--dc-surface-subtle);
}

.kafka-variable-panel__row-actions {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.kafka-variable-panel__icon-action {
  width: 26px;
  height: 26px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
  color: var(--dc-text-secondary);
}

.kafka-variable-panel__icon-action:hover {
  border-color: var(--dc-primary);
  color: var(--dc-primary);
}

.kafka-variable-panel__icon-action.is-danger:hover {
  border-color: var(--dc-danger);
  color: var(--dc-danger);
}

.kafka-variable-panel__icon-action svg {
  width: 14px;
  height: 14px;
}

.kafka-variable-panel__form {
  display: grid;
  gap: 2px;
}

.kafka-variable-panel__form :deep(.el-select) {
  width: 100%;
}

.kafka-variable-panel__form-grid {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 160px;
  gap: 12px;
}

.kafka-variable-panel__pagination {
  min-height: 42px;
  display: flex;
  align-items: center;
  justify-content: flex-end;
  padding: 6px 12px;
  border-top: 1px solid var(--dc-border);
  background: var(--dc-surface-raised);
}

@media (max-width: 900px) {
  .kafka-variable-panel__toolbar {
    align-items: flex-start;
    flex-direction: column;
  }

  .kafka-variable-panel__actions {
    flex-wrap: wrap;
  }

  .kafka-variable-panel__body.has-editor {
    grid-template-columns: 1fr;
  }

  .kafka-variable-panel__editor {
    min-height: 360px;
    border-left: 0;
    border-top: 1px solid var(--dc-border);
  }
}
</style>
