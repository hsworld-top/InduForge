<template>
  <section class="kafka-variable-panel">
    <header class="kafka-variable-panel__toolbar">
      <div class="kafka-variable-panel__title">
        <strong>{{ mapping.name || mapping.topic }} 变量管理</strong>
        <span>{{ mapping.topic }}</span>
      </div>
      <div class="kafka-variable-panel__actions">
        <el-button type="primary" size="small" @click="openCreateDialog">
          <IconTablerPlus class="kafka-variable-panel__button-icon" />
          新建变量
        </el-button>
        <el-button size="small" :loading="batchSaving" @click="createFromSamples">
          <IconTablerSparkles class="kafka-variable-panel__button-icon" />
          从样本创建
        </el-button>
        <el-button size="small" :loading="previewing" @click="runVariablePreview">
          <IconTablerActivity class="kafka-variable-panel__button-icon" />
          变量预览
        </el-button>
        <el-button size="small" @click="openMessagePreview">
          <IconTablerMessages class="kafka-variable-panel__button-icon" />
          消息预览
        </el-button>
        <el-button size="small" :loading="loading" @click="reloadAll">
          <IconTablerRefresh class="kafka-variable-panel__button-icon" />
          刷新
        </el-button>
      </div>
    </header>

    <div class="kafka-variable-panel__meta">
      <WorkbenchStatusPill :label="connected ? '已连接' : '未连接'" :tone="connected ? 'success' : 'neutral'" />
      <WorkbenchStatusPill :label="`变量 ${pagination.total}`" tone="info" />
      <span v-if="candidateCount > 0">样本候选 {{ candidateCount }} 个</span>
      <div class="kafka-variable-panel__group-tools">
        <el-select v-model="selectedGroupId" size="small" class="kafka-variable-panel__group-filter">
          <el-option label="全部变量" value="" />
          <el-option label="未分组" value="__ungrouped" />
          <el-option v-for="group in groups" :key="group.id" :label="group.name" :value="group.id" />
        </el-select>
        <el-tooltip content="新建变量组" placement="top">
          <el-button size="small" :icon="IconTablerFolderPlus" @click="openCreateGroup" />
        </el-tooltip>
        <el-tooltip content="编辑当前变量组" placement="top">
          <el-button
            size="small"
            :icon="IconTablerEdit"
            :disabled="!currentGroup"
            @click="currentGroup && openEditGroup(currentGroup)"
          />
        </el-tooltip>
        <el-tooltip content="删除当前变量组" placement="top">
          <el-button
            size="small"
            :icon="IconTablerTrash"
            :disabled="!currentGroup"
            @click="currentGroup && deleteGroup(currentGroup)"
          />
        </el-tooltip>
      </div>
    </div>

    <div class="kafka-variable-panel__body">
      <el-table v-loading="loading" :data="fields" height="100%" empty-text="暂无变量">
        <el-table-column prop="name" label="变量名" min-width="140" show-overflow-tooltip />
        <el-table-column prop="valuePath" label="字段路径" min-width="170" show-overflow-tooltip />
        <el-table-column prop="dataType" label="类型" width="92" />
        <el-table-column prop="dataPointPath" label="数据点" min-width="190" show-overflow-tooltip>
          <template #default="{ row }">
            {{ row.dataPointPath || '-' }}
          </template>
        </el-table-column>
        <el-table-column label="最后值" min-width="140" show-overflow-tooltip>
          <template #default="{ row }">
            <span class="kafka-variable-panel__mono">{{ formatLastValue(row.lastValue) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="质量" width="92">
          <template #default="{ row }">
            <WorkbenchStatusPill :label="qualityLabel(row.quality)" :tone="qualityTone(row.quality)" />
          </template>
        </el-table-column>
        <el-table-column label="更新时间" width="156">
          <template #default="{ row }">
            {{ formatTime(row.lastUpdatedAt) }}
          </template>
        </el-table-column>
        <el-table-column label="状态" width="86">
          <template #default="{ row }">
            <WorkbenchStatusPill :label="row.enabled ? '启用' : '停用'" :tone="row.enabled ? 'success' : 'neutral'" />
          </template>
        </el-table-column>
        <el-table-column label="操作" width="150" fixed="right">
          <template #default="{ row }">
            <div class="kafka-variable-panel__row-actions">
              <el-tooltip content="编辑" placement="top">
                <button type="button" class="kafka-variable-panel__icon-action" @click="openEditDialog(row)">
                  <IconTablerEdit />
                </button>
              </el-tooltip>
              <el-tooltip :content="row.enabled ? '停用' : '启用'" placement="top">
                <button type="button" class="kafka-variable-panel__icon-action" @click="toggleField(row)">
                  <IconTablerPlayerPause v-if="row.enabled" />
                  <IconTablerPlayerPlay v-else />
                </button>
              </el-tooltip>
              <el-tooltip content="删除" placement="top">
                <button type="button" class="kafka-variable-panel__icon-action is-danger" @click="deleteField(row)">
                  <IconTablerTrash />
                </button>
              </el-tooltip>
            </div>
          </template>
        </el-table-column>
      </el-table>
      <div class="kafka-variable-panel__pagination">
        <el-pagination
          :current-page="pagination.page"
          :page-size="pagination.pageSize"
          :page-sizes="[20, 50, 100]"
          :total="pagination.total"
          background
          layout="total, sizes, prev, pager, next, jumper"
          small
          @current-change="changePage"
          @size-change="changePageSize"
        />
      </div>
    </div>

    <DcDialog
      v-model="dialogVisible"
      :title="editingField ? '编辑变量' : '新建变量'"
      width="560px"
      :close-disabled="fieldSaving"
    >
      <el-form class="kafka-variable-panel__form" label-position="top">
        <el-form-item label="变量名">
          <el-input v-model="form.name" placeholder="temperature" />
        </el-form-item>
        <el-form-item label="字段路径">
          <el-input v-model="form.valuePath" placeholder="payload.temperature" />
        </el-form-item>
        <div class="kafka-variable-panel__form-grid">
          <el-form-item label="类型">
            <el-select v-model="form.dataType">
              <el-option label="string" value="string" />
              <el-option label="number" value="number" />
              <el-option label="boolean" value="boolean" />
              <el-option label="object" value="object" />
              <el-option label="array" value="array" />
            </el-select>
          </el-form-item>
          <el-form-item label="状态">
            <el-switch v-model="form.enabled" active-text="启用" inactive-text="停用" />
          </el-form-item>
        </div>
        <el-form-item label="所属分组">
          <el-select v-model="form.groupId" clearable placeholder="未分组">
            <el-option v-for="group in groups" :key="group.id" :label="group.name" :value="group.id" />
          </el-select>
        </el-form-item>
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
      v-model="groupDialogVisible"
      :title="editingGroup ? '编辑变量组' : '新建变量组'"
      width="440px"
      :close-disabled="groupSaving"
    >
      <el-form class="kafka-variable-panel__form" label-position="top">
        <el-form-item label="变量组名称">
          <el-input v-model="groupForm.name" placeholder="遥测变量" />
        </el-form-item>
        <el-form-item label="父级变量组">
          <el-select v-model="groupForm.parentId" clearable placeholder="根目录">
            <el-option
              v-for="group in groupParentOptions"
              :key="group.id"
              :label="group.name"
              :value="group.id"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="说明">
          <el-input v-model="groupForm.description" type="textarea" :rows="3" placeholder="可选" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button :disabled="groupSaving" @click="groupDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="groupSaving" :disabled="!groupForm.name.trim()" @click="saveGroup">
          保存
        </el-button>
      </template>
    </DcDialog>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import dayjs from 'dayjs'
import IconTablerActivity from '~icons/tabler/activity'
import IconTablerEdit from '~icons/tabler/edit'
import IconTablerFolderPlus from '~icons/tabler/folder-plus'
import IconTablerMessages from '~icons/tabler/messages'
import IconTablerPlayerPause from '~icons/tabler/player-pause'
import IconTablerPlayerPlay from '~icons/tabler/player-play'
import IconTablerPlus from '~icons/tabler/plus'
import IconTablerRefresh from '~icons/tabler/refresh'
import IconTablerSparkles from '~icons/tabler/sparkles'
import IconTablerTrash from '~icons/tabler/trash'
import dataAPI from '@/api/data.api'
import DcDialog from '@/components/shared/DcDialog.vue'
import WorkbenchStatusPill from '@/components/workbench/WorkbenchStatusPill.vue'
import { TIME_FORMAT } from '@/constants'
import { getApiErrorMessage } from '@/utils/request'
import type { KafkaField, KafkaFieldGroup, KafkaPreview, KafkaPreviewSample, KafkaTopicMapping } from './types'

type CandidateField = {
  path: string
  dataType: string
  exists: boolean
}

const props = defineProps<{
  projectId: string
  mapping: KafkaTopicMapping
  samples: KafkaPreviewSample[]
  connected: boolean
}>()

const emit = defineEmits<{
  (event: 'openPreview', mapping: KafkaTopicMapping): void
  (
    event: 'samples',
    payload: { mappingId: string; samples: KafkaPreviewSample[]; preview: KafkaPreview },
  ): void
}>()

const fields = ref<KafkaField[]>([])
const groups = ref<KafkaFieldGroup[]>([])
const candidates = ref<CandidateField[]>([])
const loading = ref(false)
const previewing = ref(false)
const batchSaving = ref(false)
const fieldSaving = ref(false)
const groupSaving = ref(false)
const dialogVisible = ref(false)
const groupDialogVisible = ref(false)
const editingField = ref<KafkaField | null>(null)
const editingGroup = ref<KafkaFieldGroup | null>(null)
const selectedGroupId = ref('')
const pagination = ref({ page: 1, pageSize: 20, total: 0, totalPages: 0 })

const form = reactive({
  groupId: '',
  name: '',
  valuePath: '',
  dataType: 'string',
  enabled: true,
  description: '',
})

const groupForm = reactive({
  name: '',
  parentId: '',
  description: '',
})

const canSubmit = computed(() => form.name.trim().length > 0 && form.valuePath.trim().length > 0)
const candidateCount = computed(() => candidates.value.filter((candidate) => !candidate.exists).length)
const currentGroup = computed(() => groups.value.find((group) => group.id === selectedGroupId.value) || null)
const groupParentOptions = computed(() =>
  groups.value.filter((group) => !editingGroup.value || group.id !== editingGroup.value.id),
)

const loadFields = async () => {
  loading.value = true
  try {
    const res = await dataAPI.getKafkaFields(props.projectId, props.mapping.id, {
      page: pagination.value.page,
      pageSize: pagination.value.pageSize,
      groupId: selectedGroupId.value || undefined,
    })
    fields.value = res.list || []
    pagination.value = {
      page: res.pagination?.page || pagination.value.page,
      pageSize: res.pagination?.pageSize || pagination.value.pageSize,
      total: res.pagination?.total || 0,
      totalPages: res.pagination?.totalPages || 0,
    }
    candidates.value = inferCandidateFields(props.samples, fields.value)
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '加载 Kafka 变量失败'))
  } finally {
    loading.value = false
  }
}

const loadGroups = async () => {
  try {
    const res = await dataAPI.getKafkaFieldGroups(props.projectId, props.mapping.id)
    groups.value = res.list || []
    if (selectedGroupId.value && selectedGroupId.value !== '__ungrouped' && !currentGroup.value) {
      selectedGroupId.value = ''
    }
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '加载 Kafka 变量组失败'))
  }
}

const reloadAll = async () => {
  await Promise.all([loadGroups(), loadFields()])
}

const reloadFirstPage = async () => {
  pagination.value.page = 1
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
  await reloadFirstPage()
}

const openCreateDialog = () => {
  editingField.value = null
  Object.assign(form, {
    groupId: selectedGroupId.value && selectedGroupId.value !== '__ungrouped' ? selectedGroupId.value : '',
    name: '',
    valuePath: '',
    dataType: 'string',
    enabled: true,
    description: '',
  })
  dialogVisible.value = true
}

const openEditDialog = (field: KafkaField) => {
  editingField.value = field
  Object.assign(form, {
    groupId: field.groupId || '',
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
      groupId: form.groupId || null,
      name: form.name.trim(),
      valuePath: form.valuePath.trim(),
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
    ElMessage.success('变量已保存')
    await reloadAfterMutation()
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '保存 Kafka 变量失败'))
  } finally {
    fieldSaving.value = false
  }
}

const createFromSamples = async () => {
  const nextCandidates = inferCandidateFields(props.samples, fields.value).filter(
    (candidate) => !candidate.exists,
  )
  if (nextCandidates.length === 0) {
    ElMessage.warning(props.samples.length > 0 ? '暂无可创建的样本变量' : '请先连接后执行变量预览')
    return
  }
  batchSaving.value = true
  try {
    await dataAPI.createKafkaFieldsBatch(
      props.projectId,
      props.mapping.id,
      nextCandidates.map((candidate) => ({
        name: candidate.path.split('.').pop() || candidate.path,
        valuePath: candidate.path,
        dataType: candidate.dataType,
        enabled: true,
        groupId: selectedGroupId.value && selectedGroupId.value !== '__ungrouped' ? selectedGroupId.value : null,
      })),
    )
    ElMessage.success('样本变量已创建')
    await reloadAfterMutation()
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '从样本创建变量失败'))
  } finally {
    batchSaving.value = false
  }
}

const runVariablePreview = async () => {
  if (!props.connected) {
    ElMessage.warning('请先连接后再预览变量')
    return
  }
  previewing.value = true
  try {
    const preview = await dataAPI.previewKafkaTopicMapping(props.projectId, props.mapping.id, {
      limit: props.mapping.sampleLimit,
      timeoutMs: props.mapping.timeoutMs,
      decode: props.mapping.decode,
    })
    emit('samples', {
      mappingId: String(props.mapping.id),
      samples: preview.samples || [],
      preview,
    })
    ElMessage.success('变量预览已刷新')
    await loadFields()
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, 'Kafka 变量预览失败'))
  } finally {
    previewing.value = false
  }
}

const openMessagePreview = () => {
  if (!props.connected) {
    ElMessage.warning('请先连接后再预览消息')
    return
  }
  emit('openPreview', props.mapping)
}

const toggleField = async (field: KafkaField) => {
  try {
    await dataAPI.toggleKafkaField(props.projectId, field.id, !field.enabled)
    ElMessage.success(field.enabled ? '变量已停用' : '变量已启用')
    await loadFields()
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '更新变量状态失败'))
  }
}

const deleteField = async (field: KafkaField) => {
  try {
    await ElMessageBox.confirm(`删除变量“${field.name}”？对应数据点将标记为失效。`, '删除变量', {
      confirmButtonText: '删除',
      cancelButtonText: '取消',
      type: 'warning',
    })
    await dataAPI.deleteKafkaField(props.projectId, field.id)
    ElMessage.success('变量已删除')
    await reloadAfterMutation()
  } catch (error) {
    if (error === 'cancel' || error === 'close') return
    ElMessage.error(getApiErrorMessage(error, '删除 Kafka 变量失败'))
  }
}

const openCreateGroup = () => {
  editingGroup.value = null
  Object.assign(groupForm, { name: '', parentId: '', description: '' })
  groupDialogVisible.value = true
}

const openEditGroup = (group: KafkaFieldGroup) => {
  editingGroup.value = group
  Object.assign(groupForm, {
    name: group.name,
    parentId: group.parentId || '',
    description: group.description || '',
  })
  groupDialogVisible.value = true
}

const saveGroup = async () => {
  groupSaving.value = true
  try {
    const payload = {
      name: groupForm.name.trim(),
      parentId: groupForm.parentId || null,
      description: groupForm.description.trim() || null,
      sortOrder: editingGroup.value?.sortOrder || 0,
    }
    if (editingGroup.value) {
      await dataAPI.updateKafkaFieldGroup(props.projectId, editingGroup.value.id, payload)
    } else {
      await dataAPI.createKafkaFieldGroup(props.projectId, props.mapping.id, payload)
    }
    groupDialogVisible.value = false
    ElMessage.success(editingGroup.value ? '变量组已更新' : '变量组已创建')
    await loadGroups()
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '保存 Kafka 变量组失败'))
  } finally {
    groupSaving.value = false
  }
}

const deleteGroup = async (group: KafkaFieldGroup) => {
  try {
    await ElMessageBox.confirm(`删除变量组“${group.name}”？组内变量会移动到未分组。`, '删除变量组', {
      confirmButtonText: '删除',
      cancelButtonText: '取消',
      type: 'warning',
    })
    await dataAPI.deleteKafkaFieldGroup(props.projectId, group.id)
    if (selectedGroupId.value === group.id) selectedGroupId.value = ''
    ElMessage.success('变量组已删除')
    await reloadAll()
  } catch (error) {
    if (error === 'cancel' || error === 'close') return
    ElMessage.error(getApiErrorMessage(error, '删除 Kafka 变量组失败'))
  }
}

const inferCandidateFields = (samples: KafkaPreviewSample[], existing: KafkaField[]) => {
  const seen = new Map<string, CandidateField>()
  const existingPaths = new Set(existing.map((field) => field.valuePath))

  // 样本推断只辅助创建变量，真实变量仍以用户保存的字段映射为准。
  const visit = (prefix: string, value: unknown) => {
    if (value && typeof value === 'object' && !Array.isArray(value)) {
      Object.entries(value as Record<string, unknown>).forEach(([key, child]) => {
        visit(prefix ? `${prefix}.${key}` : key, child)
      })
      return
    }
    if (!prefix || seen.has(prefix)) return
    seen.set(prefix, {
      path: prefix,
      dataType: resolveDataType(value),
      exists: existingPaths.has(prefix),
    })
  }

  samples.forEach((sample) => visit('', normalizeSampleValue(sample.value)))
  return Array.from(seen.values())
}

const normalizeSampleValue = (value: unknown) => {
  if (typeof value !== 'string') return value
  try {
    return JSON.parse(value)
  } catch {
    return value
  }
}

const resolveDataType = (value: unknown) => {
  if (typeof value === 'number') return 'number'
  if (typeof value === 'boolean') return 'boolean'
  if (typeof value === 'string') return 'string'
  if (Array.isArray(value)) return 'array'
  return 'object'
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

onMounted(reloadAll)

watch(selectedGroupId, () => {
  void reloadFirstPage()
})

watch(
  () => props.mapping.id,
  () => {
    selectedGroupId.value = ''
    pagination.value.page = 1
    void reloadAll()
  },
)

watch(
  () => props.samples,
  () => {
    void loadFields()
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

.kafka-variable-panel__meta {
  min-height: 34px;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 12px;
  border-bottom: 1px solid var(--dc-border);
  color: var(--dc-text-muted);
  font-size: 12px;
}

.kafka-variable-panel__group-tools {
  margin-left: auto;
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.kafka-variable-panel__group-filter {
  width: 160px;
}

.kafka-variable-panel__body {
  min-height: 0;
  flex: 1;
  display: flex;
  flex-direction: column;
}

.kafka-variable-panel__body :deep(.el-table) {
  flex: 1;
  min-height: 0;
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

  .kafka-variable-panel__meta {
    align-items: flex-start;
    flex-direction: column;
  }

  .kafka-variable-panel__group-tools {
    margin-left: 0;
    flex-wrap: wrap;
  }
}
</style>
