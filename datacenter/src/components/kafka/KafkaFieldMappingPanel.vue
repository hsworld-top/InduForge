<template>
  <section class="kafka-field-panel">
    <header class="kafka-field-panel__toolbar">
      <div class="kafka-field-panel__title">
        <strong>{{ mapping.name || mapping.topic }} 字段映射</strong>
        <span>{{ mapping.topic }}</span>
      </div>
      <div class="kafka-field-panel__actions">
        <el-button size="small" :loading="loading" @click="loadFields">刷新</el-button>
        <el-button
          type="primary"
          size="small"
          :disabled="selectedCandidates.length === 0"
          :loading="saving"
          @click="createSelectedFields"
        >
          创建数据点
        </el-button>
      </div>
    </header>

    <div class="kafka-field-panel__body">
      <section class="kafka-field-panel__section">
        <div class="kafka-field-panel__section-head">
          <h3>样本字段</h3>
          <WorkbenchStatusPill :label="`候选 ${candidates.length}`" tone="info" />
        </div>
        <el-table :data="candidates" height="100%" empty-text="暂无样本字段">
          <el-table-column width="46">
            <template #default="{ row }">
              <el-checkbox v-model="row.selected" :disabled="row.exists" />
            </template>
          </el-table-column>
          <el-table-column prop="path" label="字段路径" min-width="180" show-overflow-tooltip />
          <el-table-column prop="dataType" label="类型" width="100" />
          <el-table-column label="状态" width="110">
            <template #default="{ row }">
              <WorkbenchStatusPill :label="row.exists ? '已映射' : '候选'" :tone="row.exists ? 'success' : 'info'" />
            </template>
          </el-table-column>
        </el-table>
      </section>

      <section class="kafka-field-panel__section">
        <div class="kafka-field-panel__section-head">
          <h3>已建字段</h3>
          <WorkbenchStatusPill :label="`字段 ${fields.length}`" tone="success" />
        </div>
        <el-table v-loading="loading" :data="fields" height="100%" empty-text="暂无字段映射">
          <el-table-column prop="name" label="名称" min-width="130" show-overflow-tooltip />
          <el-table-column prop="valuePath" label="字段路径" min-width="150" show-overflow-tooltip />
          <el-table-column prop="dataType" label="类型" width="90" />
          <el-table-column prop="dataPointPath" label="数据点" min-width="180" show-overflow-tooltip>
            <template #default="{ row }">
              {{ row.dataPointPath || '-' }}
            </template>
          </el-table-column>
        </el-table>
      </section>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import dataAPI from '@/api/data.api'
import WorkbenchStatusPill from '@/components/workbench/WorkbenchStatusPill.vue'
import { getApiErrorMessage } from '@/utils/request'
import type { KafkaField, KafkaPreviewSample, KafkaTopicMapping } from './types'

type CandidateField = {
  path: string
  dataType: string
  selected: boolean
  exists: boolean
}

const props = defineProps<{
  projectId: string
  mapping: KafkaTopicMapping
  samples: KafkaPreviewSample[]
}>()

const fields = ref<KafkaField[]>([])
const candidates = ref<CandidateField[]>([])
const loading = ref(false)
const saving = ref(false)
const selectedCandidates = computed(() =>
  candidates.value.filter((candidate) => candidate.selected && !candidate.exists),
)

const loadFields = async () => {
  loading.value = true
  try {
    const res = await dataAPI.getKafkaFields(props.projectId, props.mapping.id)
    fields.value = res.list || []
    candidates.value = inferCandidateFields(props.samples, fields.value)
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '加载 Kafka 字段失败'))
  } finally {
    loading.value = false
  }
}

const createSelectedFields = async () => {
  saving.value = true
  try {
    await dataAPI.createKafkaFieldsBatch(
      props.projectId,
      props.mapping.id,
      selectedCandidates.value.map((candidate) => ({
        name: candidate.path.split('.').pop() || candidate.path,
        valuePath: candidate.path,
        dataType: candidate.dataType,
        enabled: true,
      })),
    )
    ElMessage.success('字段映射已创建')
    await loadFields()
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '创建字段映射失败'))
  } finally {
    saving.value = false
  }
}

const inferCandidateFields = (samples: KafkaPreviewSample[], existing: KafkaField[]) => {
  const seen = new Map<string, CandidateField>()
  const existingPaths = new Set(existing.map((field) => field.valuePath))
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
      selected: false,
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
  return 'json'
}

onMounted(loadFields)
watch(
  () => [props.mapping.id, props.samples] as const,
  () => {
    void loadFields()
  },
)
</script>

<style scoped>
.kafka-field-panel {
  height: 100%;
  min-height: 0;
  display: flex;
  flex-direction: column;
  background: var(--dc-surface-raised);
}

.kafka-field-panel__toolbar {
  min-height: 48px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 8px 12px;
  border-bottom: 1px solid var(--dc-border);
  background: var(--dc-surface-subtle);
}

.kafka-field-panel__title {
  min-width: 0;
  display: grid;
  gap: 3px;
}

.kafka-field-panel__title strong,
.kafka-field-panel__title span {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.kafka-field-panel__title strong {
  color: var(--dc-text);
  font-size: 13px;
}

.kafka-field-panel__title span {
  color: var(--dc-text-muted);
  font-family: var(--dc-font-mono, ui-monospace, SFMono-Regular, Menlo, Consolas, monospace);
  font-size: 11px;
}

.kafka-field-panel__actions {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}

.kafka-field-panel__body {
  min-height: 0;
  flex: 1;
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
}

.kafka-field-panel__section {
  min-height: 0;
  display: flex;
  flex-direction: column;
  border-right: 1px solid var(--dc-border);
}

.kafka-field-panel__section:last-child {
  border-right: 0;
}

.kafka-field-panel__section-head {
  min-height: 42px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 0 12px;
  border-bottom: 1px solid var(--dc-border);
}

.kafka-field-panel__section h3 {
  margin: 0;
  color: var(--dc-text);
  font-size: 12px;
}

@media (max-width: 980px) {
  .kafka-field-panel__toolbar {
    align-items: flex-start;
    flex-direction: column;
  }

  .kafka-field-panel__body {
    grid-template-columns: 1fr;
  }

  .kafka-field-panel__section {
    min-height: 320px;
    border-right: 0;
    border-bottom: 1px solid var(--dc-border);
  }
}
</style>
