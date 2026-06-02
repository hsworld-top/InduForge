<template>
  <div class="mqtt-batch-mapping">
    <header class="mqtt-batch-mapping__header">
      <div>
        <strong>批量变量映射</strong>
        <span>{{ subscription.name || subscription.topic }}</span>
      </div>
      <div class="mqtt-batch-mapping__header-actions">
        <el-button
          size="small"
          type="primary"
          plain
          :disabled="!previewSessionId"
          @click="$emit('openMonitor')"
        >
          <IconTablerActivity class="mqtt-batch-mapping__button-icon" />
          变量预览/监控
        </el-button>
        <el-button size="small" :loading="loading" @click="loadTags">
          <IconTablerRefresh class="mqtt-batch-mapping__button-icon" />
          刷新
        </el-button>
        <el-button type="primary" size="small" :loading="saving" @click="saveMappings">
          <IconTablerDeviceFloppy class="mqtt-batch-mapping__button-icon" />
          保存映射
        </el-button>
      </div>
    </header>

    <section class="mqtt-batch-mapping__body">
      <div class="mqtt-batch-mapping__config">
        <div class="mqtt-batch-mapping__sample">
          <div class="mqtt-batch-mapping__section-title">
            <strong>样例消息</strong>
            <el-button text size="small" @click="fillDefaultSample">填入示例</el-button>
          </div>
          <MonacoEditor
            v-model="samplePayload"
            class="mqtt-batch-mapping__sample-input"
            language="json"
            theme="vs"
            height="100%"
            :options="sampleEditorOptions"
          />
        </div>

        <div class="mqtt-batch-mapping__rules">
          <div class="mqtt-batch-mapping__section-title">
            <strong>拆分规则</strong>
          </div>
          <el-form label-position="top" class="mqtt-batch-mapping__form">
            <el-form-item label="变量数组路径">
              <el-input v-model="ruleForm.arrayPath" placeholder="$ 或 $.items" />
            </el-form-item>
            <el-form-item label="变量名字段路径">
              <el-input v-model="ruleForm.namePath" placeholder="N" />
            </el-form-item>
            <el-form-item label="值字段路径">
              <el-input v-model="ruleForm.valuePath" placeholder="V" />
            </el-form-item>
            <el-form-item label="质量字段路径">
              <el-input v-model="ruleForm.qualityPath" placeholder="Q" />
            </el-form-item>
            <el-form-item label="时间字段路径">
              <el-input v-model="ruleForm.timePath" placeholder="T" />
            </el-form-item>
          </el-form>
          <el-button type="primary" plain size="small" @click="parseSample">
            <IconTablerWand class="mqtt-batch-mapping__button-icon" />
            解析预览
          </el-button>
        </div>
      </div>

      <div class="mqtt-batch-mapping__result">
        <div class="mqtt-batch-mapping__section-title">
          <strong>映射结果</strong>
          <span>{{ mappings.length }} 个变量</span>
        </div>
        <el-table
          class="mqtt-batch-mapping__table"
          :data="mappings"
          height="100%"
          row-key="matchName"
          empty-text="填写样例消息后点击解析预览"
        >
          <el-table-column label="变量名" min-width="150">
            <template #default="{ row }">
              <el-input v-model="row.name" size="small" />
            </template>
          </el-table-column>
          <el-table-column label="数据类型" width="120">
            <template #default="{ row }">
              <el-select v-model="row.dataType" size="small">
                <el-option label="字符串" value="string" />
                <el-option label="数值" value="number" />
                <el-option label="布尔" value="boolean" />
                <el-option label="对象" value="object" />
                <el-option label="数组" value="array" />
              </el-select>
            </template>
          </el-table-column>
          <el-table-column label="样例值" min-width="150" show-overflow-tooltip>
            <template #default="{ row }">
              <span class="mqtt-batch-mapping__mono">{{ formatValue(row.sampleValue) }}</span>
            </template>
          </el-table-column>
          <el-table-column label="质量" width="92" show-overflow-tooltip>
            <template #default="{ row }">
              <span class="mqtt-batch-mapping__mono">{{ formatValue(row.sampleQuality) }}</span>
            </template>
          </el-table-column>
          <el-table-column label="时间" min-width="150" show-overflow-tooltip>
            <template #default="{ row }">
              <span class="mqtt-batch-mapping__mono">{{ formatValue(row.sampleTime) }}</span>
            </template>
          </el-table-column>
          <el-table-column label="状态" width="92">
            <template #default="{ row }">
              <span
                class="mqtt-batch-mapping__status"
                :class="{ 'is-existing': row.existingTagId }"
              >
                {{ row.existingTagId ? '已存在' : '待创建' }}
              </span>
            </template>
          </el-table-column>
        </el-table>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { createMqttTagsBatch, getMqttTags, updateMqttTag } from '@/api/data.api'
import { useMqttTagSync } from '@/composables/useMqttTagSync'
import { getApiErrorMessage } from '@/utils/request'
import MonacoEditor from '@/components/MonacoEditor.vue'
import IconTablerActivity from '~icons/tabler/activity'
import IconTablerDeviceFloppy from '~icons/tabler/device-floppy'
import IconTablerRefresh from '~icons/tabler/refresh'
import IconTablerWand from '~icons/tabler/wand'

type MqttSubscription = {
  id: string
  name?: string
  topic?: string
}

type BatchMappingRow = {
  matchName: string
  name: string
  code: string
  dataType: 'string' | 'number' | 'boolean' | 'object' | 'array'
  sampleValue: unknown
  sampleQuality: unknown
  sampleTime: unknown
  existingTagId?: string
}

const props = defineProps<{
  projectId: string
  subscription: MqttSubscription
  previewSessionId?: string
}>()

defineEmits<{
  (event: 'openMonitor'): void
}>()

const loading = ref(false)
const saving = ref(false)
const existingTags = ref<any[]>([])
const mappings = ref<BatchMappingRow[]>([])
const samplePayload = ref('')
const ruleForm = reactive({
  arrayPath: '$',
  namePath: 'N',
  valuePath: 'V',
  qualityPath: 'Q',
  timePath: 'T',
})
const sampleEditorOptions = {
  minimap: { enabled: false },
  fontSize: 12,
  lineHeight: 20,
  tabSize: 2,
  wordWrap: 'off',
  scrollBeyondLastLine: false,
  automaticLayout: true,
}
const { refresh: notifyTagRefresh } = useMqttTagSync(props.subscription.id)

const defaultSample = computed(() =>
  JSON.stringify(
    [
      { N: 'temperature', V: 23.5, T: '2026-06-02 10:00:00', Q: 192 },
      { N: 'pressure', V: 0.82, T: '2026-06-02 10:00:00', Q: 192 },
    ],
    null,
    2,
  ),
)

const fillDefaultSample = () => {
  samplePayload.value = defaultSample.value
}

const loadTags = async () => {
  loading.value = true
  try {
    const response = await getMqttTags(props.projectId, props.subscription.id, {
      page: 1,
      pageSize: 200,
      sortBy: 'createdAt',
      sortOrder: 'desc',
    })
    existingTags.value = response.data?.list || []
    if (mappings.value.length === 0) {
      restoreMappingsFromExistingTags()
    }
    markExistingMappings()
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '加载变量映射失败'))
  } finally {
    loading.value = false
  }
}

const parseSample = () => {
  let parsed: unknown
  try {
    parsed = JSON.parse(samplePayload.value)
  } catch {
    ElMessage.error('样例消息不是合法 JSON')
    return
  }

  const arrayValue = resolvePath(parsed, ruleForm.arrayPath)
  if (!Array.isArray(arrayValue)) {
    ElMessage.error('变量数组路径没有命中数组')
    return
  }

  const rows = arrayValue
    .map((item) => {
      const rawName = resolvePath(item, ruleForm.namePath)
      const matchName = String(rawName ?? '').trim()
      if (!matchName) return null
      const sampleValue = resolvePath(item, ruleForm.valuePath)
      return {
        matchName,
        name: matchName,
        code: normalizeTagCode(matchName),
        dataType: inferDataType(sampleValue),
        sampleValue,
        sampleQuality: resolvePath(item, ruleForm.qualityPath),
        sampleTime: resolvePath(item, ruleForm.timePath),
      }
    })
    .filter(Boolean) as BatchMappingRow[]

  const uniqueRows = Array.from(new Map(rows.map((row) => [row.matchName, row])).values())
  mappings.value = uniqueRows
  markExistingMappings()
}

const saveMappings = async () => {
  if (mappings.value.length === 0) {
    ElMessage.info('请先解析出变量映射')
    return
  }

  const createRows = mappings.value.filter((row) => !row.existingTagId)
  const updateRows = mappings.value.filter((row) => row.existingTagId)
  saving.value = true
  try {
    if (createRows.length > 0) {
      await createMqttTagsBatch(
        props.projectId,
        props.subscription.id,
        createRows.map((row, index) => buildTagPayload(row, index)),
      )
    }
    await Promise.all(
      updateRows.map((row, index) =>
        updateMqttTag(props.projectId, row.existingTagId, buildTagPayload(row, index)),
      ),
    )
    ElMessage.success(`映射已保存：新建 ${createRows.length} 个，更新 ${updateRows.length} 个`)
    await loadTags()
    notifyTagRefresh()
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '保存批量变量映射失败'))
  } finally {
    saving.value = false
  }
}

const buildTagPayload = (row: BatchMappingRow, index: number) => ({
  name: row.name,
  code: row.code,
  dataType: row.dataType,
  parseType: 'batch_jsonpath',
  parseRule: JSON.stringify({
    arrayPath: ruleForm.arrayPath,
    namePath: ruleForm.namePath,
    matchName: row.matchName,
    valuePath: ruleForm.valuePath,
    qualityPath: ruleForm.qualityPath,
    timePath: ruleForm.timePath,
  }),
  order: index,
})

const markExistingMappings = () => {
  const codeMap = new Map(existingTags.value.map((tag) => [tag.code, tag]))
  mappings.value.forEach((row) => {
    row.existingTagId = codeMap.get(row.code)?.id || ''
  })
}

const restoreMappingsFromExistingTags = () => {
  const batchTags = existingTags.value.filter((tag) => tag.parseType === 'batch_jsonpath')
  if (batchTags.length === 0) return

  const rows = batchTags
    .map((tag) => {
      const rule = parseBatchRule(tag.parseRule)
      const matchName = String(rule.matchName || '').trim()
      if (!matchName) return null
      return {
        matchName,
        name: tag.name || matchName,
        code: tag.code,
        dataType: tag.dataType || 'string',
        sampleValue: undefined,
        sampleQuality: undefined,
        sampleTime: undefined,
        existingTagId: tag.id,
      }
    })
    .filter(Boolean) as BatchMappingRow[]

  const firstRule = parseBatchRule(batchTags[0]?.parseRule)
  ruleForm.arrayPath = firstRule.arrayPath || '$'
  ruleForm.namePath = firstRule.namePath || 'N'
  ruleForm.valuePath = firstRule.valuePath || 'V'
  ruleForm.qualityPath = firstRule.qualityPath || 'Q'
  ruleForm.timePath = firstRule.timePath || 'T'
  mappings.value = rows
}

const parseBatchRule = (ruleText: string) => {
  try {
    return JSON.parse(ruleText || '{}')
  } catch {
    return {}
  }
}

const normalizeTagCode = (name: string) => {
  const prefix = props.subscription.name || props.subscription.topic || props.subscription.id
  return `${normalizeCode(prefix)}_${hashText(name)}`
}

const normalizeCode = (value: string) => {
  const code = String(value || '')
    .toLowerCase()
    .replace(/\s+/g, '_')
    .replace(/[^a-z0-9_]/g, '_')
    .replace(/^_+|_+$/g, '')
    .replace(/_+/g, '_')
  return code || `tag_${hashText(value)}`
}

const hashText = (value: string) => {
  let hash = 0
  for (const char of String(value || '')) {
    hash = (hash * 31 + char.charCodeAt(0)) >>> 0
  }
  return hash.toString(36)
}

const inferDataType = (value: unknown): BatchMappingRow['dataType'] => {
  if (Array.isArray(value)) return 'array'
  if (value !== null && typeof value === 'object') return 'object'
  if (typeof value === 'number') return 'number'
  if (typeof value === 'boolean') return 'boolean'
  return 'string'
}

const resolvePath = (source: unknown, path: string) => {
  const normalized = normalizePath(path)
  if (normalized.length === 0) return source
  return normalized.reduce((current, segment) => {
    if (current === null || current === undefined) return undefined
    if (Array.isArray(current) && /^\d+$/.test(segment)) {
      return current[Number(segment)]
    }
    return current?.[segment]
  }, source as any)
}

const normalizePath = (path: string) =>
  String(path || '')
    .trim()
    .replace(/^\$/, '')
    .replace(/^\./, '')
    .replace(/\[(\d+)\]/g, '.$1')
    .split('.')
    .map((item) => item.trim())
    .filter(Boolean)

const formatValue = (value: unknown) => {
  if (value === null || value === undefined || value === '') return '-'
  if (typeof value === 'object') return JSON.stringify(value)
  return String(value)
}

watch(
  () => props.subscription.id,
  async () => {
    mappings.value = []
    samplePayload.value = ''
    await loadTags()
  },
)

onMounted(loadTags)
</script>

<style scoped>
.mqtt-batch-mapping {
  height: 100%;
  min-height: 0;
  display: flex;
  flex-direction: column;
  background: var(--dc-surface-raised);
}

.mqtt-batch-mapping__header {
  min-height: 52px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 10px 14px;
  border-bottom: 1px solid var(--dc-border);
  background: var(--dc-surface-subtle);
}

.mqtt-batch-mapping__header > div:first-child {
  min-width: 0;
  display: grid;
  gap: 2px;
}

.mqtt-batch-mapping__header strong {
  color: var(--dc-text);
  font-size: 14px;
}

.mqtt-batch-mapping__header span {
  overflow: hidden;
  color: var(--dc-text-muted);
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.mqtt-batch-mapping__header-actions {
  display: inline-flex;
  align-items: center;
  gap: 8px;
}

.mqtt-batch-mapping__body {
  height: 0;
  min-height: 0;
  flex: 1;
  display: grid;
  grid-template-columns: minmax(360px, 0.42fr) minmax(420px, 1fr);
  gap: 0;
}

.mqtt-batch-mapping__config {
  min-height: 0;
  display: grid;
  grid-template-rows: minmax(220px, 1fr) auto;
  border-right: 1px solid var(--dc-border);
}

.mqtt-batch-mapping__sample,
.mqtt-batch-mapping__rules,
.mqtt-batch-mapping__result {
  min-height: 0;
  padding: 12px;
}

.mqtt-batch-mapping__sample {
  display: flex;
  flex-direction: column;
  border-bottom: 1px solid var(--dc-border);
}

.mqtt-batch-mapping__sample-input {
  flex: 1;
  min-height: 0;
  overflow: hidden;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
}

.mqtt-batch-mapping__rules {
  display: grid;
  gap: 10px;
}

.mqtt-batch-mapping__section-title {
  min-height: 28px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 8px;
}

.mqtt-batch-mapping__section-title strong {
  color: var(--dc-text);
  font-size: 13px;
}

.mqtt-batch-mapping__section-title span {
  color: var(--dc-text-muted);
  font-size: 12px;
}

.mqtt-batch-mapping__form {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px 10px;
}

.mqtt-batch-mapping__form :deep(.el-form-item) {
  margin-bottom: 0;
}

.mqtt-batch-mapping__form :deep(.el-form-item:first-child) {
  grid-column: 1 / -1;
}

.mqtt-batch-mapping__result {
  display: flex;
  flex-direction: column;
}

.mqtt-batch-mapping__table {
  flex: 1;
  min-height: 0;
}

.mqtt-batch-mapping__mono {
  font-family: var(--dc-font-mono, ui-monospace, SFMono-Regular, Menlo, Consolas, monospace);
  color: var(--dc-text-secondary);
  font-size: 12px;
}

.mqtt-batch-mapping__status {
  height: 22px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 0 7px;
  border-radius: var(--dc-radius-sm);
  background: rgba(245, 158, 11, 0.12);
  color: #b45309;
  font-size: 11px;
  font-weight: 800;
}

.mqtt-batch-mapping__status.is-existing {
  background: color-mix(in oklch, var(--dc-success) 12%, var(--dc-surface-raised));
  color: var(--dc-success);
}

.mqtt-batch-mapping__button-icon {
  width: 14px;
  height: 14px;
  margin-right: 4px;
}

@media (max-width: 1180px) {
  .mqtt-batch-mapping__body {
    grid-template-columns: 1fr;
    overflow: auto;
  }

  .mqtt-batch-mapping__config {
    border-right: 0;
    border-bottom: 1px solid var(--dc-border);
  }
}
</style>
