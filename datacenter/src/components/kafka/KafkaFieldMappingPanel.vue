<template>
  <section class="kafka-variable-panel">
    <header class="kafka-variable-panel__toolbar">
      <div class="kafka-variable-panel__title">
        <strong>{{ mapping.name || mapping.topic }} 字段映射</strong>
        <span>{{ mapping.topic }}</span>
      </div>
      <div class="kafka-variable-panel__actions">
        <el-button type="primary" size="small" @click="openCreateDialog">
          <IconTablerPlus class="kafka-variable-panel__button-icon" />
          新建映射
        </el-button>
        <el-button size="small" @click="sampleEditorVisible = !sampleEditorVisible">
          <IconTablerBraces class="kafka-variable-panel__button-icon" />
          样例编辑器
        </el-button>
        <el-button size="small" :loading="loading" @click="reloadAll">
          <IconTablerRefresh class="kafka-variable-panel__button-icon" />
          刷新
        </el-button>
      </div>
    </header>

    <div class="kafka-variable-panel__body" :class="{ 'has-editor': sampleEditorVisible }">
      <div class="kafka-variable-panel__result">
        <el-table v-loading="loading" :data="fields" height="100%" empty-text="暂无字段映射">
          <el-table-column prop="name" label="映射名称" min-width="140" show-overflow-tooltip />
          <el-table-column label="字段路径" min-width="170" show-overflow-tooltip>
            <template #default="{ row }">{{ formatKafkaValuePath(row.valuePath) }}</template>
          </el-table-column>
          <el-table-column prop="dataType" label="类型" width="92" />
          <el-table-column
            prop="dataPointPath"
            label="输出数据点"
            min-width="190"
            show-overflow-tooltip
          >
            <template #default="{ row }">
              {{ row.dataPointPath || '-' }}
            </template>
          </el-table-column>
          <el-table-column label="最近测试值" min-width="140" show-overflow-tooltip>
            <template #default="{ row }">
              <span class="kafka-variable-panel__mono">{{ formatLastValue(row.lastValue) }}</span>
            </template>
          </el-table-column>
          <el-table-column label="质量" width="92">
            <template #default="{ row }">
              <WorkbenchStatusPill
                :label="qualityLabel(row.quality)"
                :tone="qualityTone(row.quality)"
              />
            </template>
          </el-table-column>
          <el-table-column label="更新时间" width="156">
            <template #default="{ row }">
              {{ formatTime(row.lastUpdatedAt) }}
            </template>
          </el-table-column>
          <el-table-column label="状态" width="86">
            <template #default="{ row }">
              <WorkbenchStatusPill
                :label="row.enabled ? '启用' : '停用'"
                :tone="row.enabled ? 'success' : 'neutral'"
              />
            </template>
          </el-table-column>
          <el-table-column label="操作" width="150" fixed="right">
            <template #default="{ row }">
              <div class="kafka-variable-panel__row-actions">
                <el-tooltip content="编辑" placement="top">
                  <button
                    type="button"
                    class="kafka-variable-panel__icon-action"
                    @click="openEditDialog(row)"
                  >
                    <IconTablerEdit />
                  </button>
                </el-tooltip>
                <el-tooltip :content="row.enabled ? '停用' : '启用'" placement="top">
                  <button
                    type="button"
                    class="kafka-variable-panel__icon-action"
                    @click="toggleField(row)"
                  >
                    <IconTablerPlayerPause v-if="row.enabled" />
                    <IconTablerPlayerPlay v-else />
                  </button>
                </el-tooltip>
                <el-tooltip content="删除" placement="top">
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
          fill-latest-text="拉取样本"
          :fill-latest-disabled="previewing"
          :error="sampleParseError"
          @fill-latest="pullSamples"
          @parse="parseEditorFields"
        />
      </div>
    </div>

    <DcDialog
      v-model="dialogVisible"
      :title="editingField ? '编辑映射' : '新建映射'"
      width="560px"
      :close-disabled="fieldSaving"
    >
      <el-form class="kafka-variable-panel__form" label-position="top">
        <el-form-item label="映射名称">
          <el-input v-model="form.name" placeholder="temperature" />
        </el-form-item>
        <el-form-item label="字段路径">
          <el-input :model-value="formatKafkaValuePath(form.valuePath)" readonly />
        </el-form-item>
        <div class="kafka-variable-panel__form-grid">
          <el-form-item label="类型">
            <el-select v-model="form.dataType">
              <el-option label="string" value="string" />
              <el-option label="float64" value="float64" />
              <el-option label="bool" value="bool" />
              <el-option label="object" value="object" />
              <el-option label="array" value="array" />
            </el-select>
          </el-form-item>
          <el-form-item label="状态">
            <el-switch v-model="form.enabled" active-text="启用" inactive-text="停用" />
          </el-form-item>
        </div>
        <el-form-item label="描述">
          <el-input v-model="form.description" type="textarea" :rows="3" placeholder="可选" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button :disabled="fieldSaving" @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="fieldSaving" :disabled="!canSubmit" @click="saveField">
          保存
        </el-button>
      </template>
    </DcDialog>

    <DcDialog
      v-model="candidateDialogVisible"
      title="解析字段候选"
      width="760px"
      :close-disabled="batchSaving"
    >
      <el-table
        class="kafka-variable-panel__candidate-table"
        :data="candidateRows"
        max-height="420"
        row-key="path"
        empty-text="暂无字段候选"
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
        <el-table-column prop="name" label="映射名称" min-width="120" />
        <el-table-column prop="path" label="字段路径" min-width="150" show-overflow-tooltip />
        <el-table-column prop="dataType" label="类型" width="88" />
        <el-table-column label="样例值" min-width="120" show-overflow-tooltip>
          <template #default="{ row }">
            <span class="kafka-variable-panel__mono">{{ formatLastValue(row.sampleValue) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="88">
          <template #default="{ row }">
            <WorkbenchStatusPill
              :label="row.exists ? '已存在' : '候选'"
              :tone="row.exists ? 'neutral' : 'info'"
            />
          </template>
        </el-table-column>
      </el-table>
      <template #footer>
        <el-button :disabled="batchSaving" @click="candidateDialogVisible = false">取消</el-button>
        <el-button
          type="primary"
          :loading="batchSaving"
          :disabled="selectedCandidateCount === 0"
          @click="createFromSamples"
        >
          保存选中字段
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
import {
  formatKafkaValuePath,
  inferKafkaSampleFields,
  normalizeKafkaSampleEditorText,
  type KafkaSampleFieldCandidate,
} from './kafkaSampleFields'
import type { KafkaField, KafkaPreview, KafkaPreviewSample, KafkaTopicMapping } from './types'

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
    ElMessage.error(getApiErrorMessage(error, '加载 Kafka 字段映射失败'))
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
    ElMessage.info('请先填入或拉取 JSON 样例，再从字段候选中选择映射')
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
    ElMessage.success('字段映射已保存')
    candidateDialogVisible.value = false
    await reloadAfterMutation()
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '保存 Kafka 字段映射失败'))
  } finally {
    fieldSaving.value = false
  }
}

const createFromSamples = async () => {
  const sourceCandidates = candidateRows.value
  const selectedPaths = selectedCandidatePaths.value
  if (selectedPaths.size === 0) {
    ElMessage.warning('请先选择要保存的字段候选')
    return
  }
  const nextCandidates = sourceCandidates.filter(
    (candidate) => !candidate.exists && selectedPaths.has(candidate.path),
  )
  if (nextCandidates.length === 0) {
    ElMessage.warning('暂无可保存的字段候选，请先解析 JSON 样例')
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
    ElMessage.success('字段映射已保存')
    candidateDialogVisible.value = false
    await reloadAfterMutation()
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '保存字段映射失败'))
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
    ElMessage.success(firstSample ? '样本已拉取并填入编辑器' : '本次未拉取到样本')
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, 'Kafka 样本拉取失败'))
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
    ElMessage.success(`已解析 ${candidateRows.value.length} 个字段候选`)
  } catch {
    sampleParseError.value = 'JSON 样例格式无效'
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
    ElMessage.success(field.enabled ? '映射已停用' : '映射已启用')
    await loadFields()
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '更新映射状态失败'))
  }
}

const deleteField = async (field: KafkaField) => {
  try {
    await ElMessageBox.confirm(`删除映射“${field.name}”？对应数据点将标记为失效。`, '删除映射', {
      confirmButtonText: '删除',
      cancelButtonText: '取消',
      type: 'warning',
    })
    await dataAPI.deleteKafkaField(props.projectId, field.id)
    ElMessage.success('映射已删除')
    await reloadAfterMutation()
  } catch (error) {
    if (error === 'cancel' || error === 'close') return
    ElMessage.error(getApiErrorMessage(error, '删除 Kafka 字段映射失败'))
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
  if (quality === 'good') return '良好'
  if (quality === 'bad') return '异常'
  return '未知'
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
