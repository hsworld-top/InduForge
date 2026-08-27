<template>
  <section class="source-output-editor">
    <header class="source-output-editor__head">
      <div>
        <strong>{{ title }}</strong>
        <span v-if="progressive && !advancedMode">
          默认将完整结果作为一个稳定数据集；需要独立数据点时再提取单个字段。
        </span>
        <span v-else-if="progressive">
          每个查询只选择一个输出；数据点属性请在数据点列表维护。
        </span>
        <span v-else>每项输出生成一个可被报警、计算和历史存储引用的数据点。</span>
      </div>
      <div class="source-output-editor__head-actions">
        <el-button
          v-if="progressive && !advancedMode"
          size="small"
          type="primary"
          plain
          :disabled="!columns.length"
          :title="columns.length ? '从查询结果中选择一个字段' : '请先运行 SQL 获取字段列表'"
          @click="enableFieldExtraction"
        >
          提取单个字段
        </el-button>
        <template v-else>
          <el-button v-if="progressive" size="small" @click="resetToWholeDataset">
            恢复完整结果集
          </el-button>
          <el-button v-if="!progressive" size="small" type="primary" plain @click="addOutput">
            添加输出
          </el-button>
        </template>
        <el-button
          v-if="progressive"
          :type="generationButtonType"
          :plain="datapointsGenerated"
          :loading="generating"
          :disabled="!canGenerate || (datapointsGenerated && !outputModified)"
          size="small"
          :title="generationButtonTitle"
          @click="$emit('generate')"
        >
          {{ generationButtonText }}
        </el-button>
      </div>
    </header>

    <div v-if="progressive && !advancedMode" class="source-output-editor__dataset">
      <div class="source-output-editor__dataset-main">
        <strong>完整查询结果</strong>
        <span>数据集对象</span>
      </div>
      <p>包含 fields、rows 和 rowCount；数据库总量由查询自行返回。</p>
      <code v-if="outputs[0]?.datapointPath">{{ outputs[0].datapointPath }}</code>
      <code v-else>主动生成后创建路径 · 输出 Key：result</code>
    </div>

    <template v-else-if="progressive">
      <div
        v-for="(output, index) in outputs"
        :key="output.id || `new-${index}`"
        class="source-output-editor__item"
      >
        <div class="source-output-editor__item-head">
          <div>
            <strong>提取结果</strong>
            <el-tag size="small" effect="plain">{{ selectorSummary(output) }}</el-tag>
          </div>
        </div>
        <div class="source-output-editor__row is-extraction">
          <label class="source-output-editor__field">
            <span>选择字段</span>
            <el-select
              :model-value="output.selector.column"
              filterable
              placeholder="选择查询结果字段"
              @update:model-value="setColumn(output, $event)"
            >
              <el-option v-for="column in columns" :key="column" :label="column" :value="column" />
            </el-select>
          </label>
        </div>
        <small v-if="output.datapointPath" class="source-output-editor__datapoint"
          >数据点：{{ output.datapointPath }}</small
        >
      </div>
    </template>

    <template v-else>
      <div
        v-for="(output, index) in outputs"
        :key="output.id || `new-${index}`"
        class="source-output-editor__item"
      >
        <div class="source-output-editor__row is-primary">
          <label class="source-output-editor__field">
            <span>输出 Key</span>
            <el-input v-model="output.key" placeholder="例如 result" @input="emitChange" />
          </label>
          <label class="source-output-editor__field">
            <span>显示名称</span>
            <el-input
              v-model="output.displayName"
              placeholder="用于数据点展示"
              @input="emitChange"
            />
          </label>
          <label class="source-output-editor__field">
            <span>数据类型</span>
            <el-select v-model="output.dataType" placeholder="请选择" @change="emitChange">
              <el-option
                v-for="option in dataTypeOptions"
                :key="option.value"
                :label="option.label"
                :value="option.value"
              />
            </el-select>
          </label>
          <el-button
            class="source-output-editor__remove"
            text
            type="danger"
            :disabled="outputs.length <= 1"
            @click="removeOutput(index)"
            >删除</el-button
          >
        </div>
        <div class="source-output-editor__row">
          <label class="source-output-editor__field">
            <span>取值方式</span>
            <el-select v-model="output.selector.kind" @change="resetSelector(output)">
              <el-option label="完整结果" value="whole" />
              <el-option
                v-if="columns.length || output.selector.kind === 'column'"
                label="结果列"
                value="column"
              />
              <el-option
                v-if="samplePaths.length || output.selector.kind === 'path'"
                label="响应字段"
                value="path"
              />
            </el-select>
          </label>
          <label v-if="output.selector.kind === 'column'" class="source-output-editor__field">
            <span>结果列</span>
            <el-select
              :model-value="output.selector.column"
              @update:model-value="setColumn(output, $event)"
            >
              <el-option v-for="column in columns" :key="column" :label="column" :value="column" />
            </el-select>
          </label>
          <label v-else-if="output.selector.kind === 'path'" class="source-output-editor__field">
            <span>响应字段</span>
            <el-select
              :model-value="pathKey(output.selector.segments)"
              @update:model-value="setPath(output, $event)"
            >
              <el-option
                v-for="path in samplePaths"
                :key="path.key"
                :label="path.label"
                :value="path.key"
              />
            </el-select>
          </label>
          <label class="source-output-editor__field">
            <span>单位</span>
            <el-input v-model="output.unit" placeholder="可选" @input="emitChange" />
          </label>
          <label class="source-output-editor__field">
            <span>精度</span>
            <el-input-number v-model="output.precisionNum" :min="0" @change="emitChange" />
          </label>
        </div>
      </div>
    </template>
  </section>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { ElMessageBox } from 'element-plus'
import { createWholeSourceOutput } from '@/api/schemas/source-output.schema'
import type {
  CanonicalDataPointType,
  SourceOutputInput,
  SourceOutputSelector,
} from '@/api/schemas/source-output.schema'

type EditableOutput = SourceOutputInput & { datapointPath?: string }
type SamplePath = { key: string; label: string; segments: Array<string | number>; type: string }

const props = withDefaults(
  defineProps<{
    modelValue: EditableOutput[]
    title?: string
    columns?: string[]
    columnTypes?: Partial<Record<string, CanonicalDataPointType>>
    sample?: unknown
    wholeDataType?: CanonicalDataPointType
    progressive?: boolean
    defaultOutputName?: string
    datapointsGenerated?: boolean
    outputModified?: boolean
    generating?: boolean
    canGenerate?: boolean
  }>(),
  {
    title: '输出数据点',
    columns: () => [],
    columnTypes: () => ({}),
    sample: undefined,
    wholeDataType: 'object',
    progressive: false,
    defaultOutputName: '完整结果',
    datapointsGenerated: false,
    outputModified: false,
    generating: false,
    canGenerate: true,
  },
)

const emit = defineEmits<{
  (event: 'update:modelValue', value: EditableOutput[]): void
  (event: 'change'): void
  (event: 'generate'): void
}>()

const outputs = computed(() => props.modelValue)
const isDefaultWholeDataset = computed(
  () =>
    outputs.value.length === 1 &&
    outputs.value[0]?.key === 'result' &&
    outputs.value[0]?.selector.kind === 'whole' &&
    outputs.value[0]?.dataType === props.wholeDataType,
)
const advancedMode = ref(!props.progressive || !isDefaultWholeDataset.value)
const generationButtonText = computed(() => {
  if (!props.canGenerate) return '保存查询后生成数据点'
  if (!props.datapointsGenerated) return '保存并生成数据点'
  if (props.outputModified) return '保存并更新数据点'
  return '数据点已生成'
})
const generationButtonType = computed(() =>
  props.datapointsGenerated && !props.outputModified ? 'success' : 'primary',
)
const generationButtonTitle = computed(() => {
  if (!props.canGenerate) return '请先保存查询，再主动生成数据点'
  if (props.datapointsGenerated && !props.outputModified) return '提取配置和数据点已经同步保存'
  return '保存当前提取配置，并创建或更新对应数据点'
})

watch(isDefaultWholeDataset, (isDefault) => {
  if (props.progressive && !isDefault) advancedMode.value = true
})
const dataTypeOptions: Array<{ value: CanonicalDataPointType; label: string }> = [
  { value: 'bool', label: '布尔' },
  { value: 'int8', label: 'int8' },
  { value: 'uint8', label: 'uint8' },
  { value: 'int16', label: 'int16' },
  { value: 'uint16', label: 'uint16' },
  { value: 'int32', label: 'int32' },
  { value: 'uint32', label: 'uint32' },
  { value: 'int64', label: 'int64' },
  { value: 'uint64', label: 'uint64' },
  { value: 'float32', label: 'float32' },
  { value: 'float64', label: 'float64' },
  { value: 'decimal', label: '高精度数值' },
  { value: 'string', label: '文本' },
  { value: 'bytes', label: '字节' },
  { value: 'datetime', label: '日期时间' },
  { value: 'object', label: '对象' },
  { value: 'array', label: '数组' },
]

const valueType = (value: unknown) => {
  if (Array.isArray(value)) return 'array'
  if (value === null) return 'null'
  return typeof value
}

const collectPaths = (
  value: unknown,
  segments: Array<string | number> = [],
  depth = 0,
): SamplePath[] => {
  if (depth > 8) return []
  if (Array.isArray(value)) {
    return value
      .slice(0, 20)
      .flatMap((entry, index) => collectPaths(entry, [...segments, index], depth + 1))
  }
  if (value && typeof value === 'object') {
    return Object.entries(value as Record<string, unknown>).flatMap(([key, entry]) =>
      collectPaths(entry, [...segments, key], depth + 1),
    )
  }
  if (!segments.length) return []
  const key = JSON.stringify(segments)
  const label = segments
    .map((segment) =>
      typeof segment === 'number'
        ? `[${segment}]`
        : segments.indexOf(segment) === 0
          ? segment
          : `.${segment}`,
    )
    .join('')
    .replace('.[', '[')
  return [{ key, label, segments, type: valueType(value) }]
}

const samplePaths = computed(() =>
  collectPaths(props.sample).filter(
    (path, index, all) => all.findIndex((item) => item.key === path.key) === index,
  ),
)
const pathKey = (segments?: Array<string | number>) => JSON.stringify(segments || [])

const valueAtSegments = (root: unknown, segments: Array<string | number>) => {
  let current = root
  for (const segment of segments) {
    if (current === null || current === undefined || typeof current !== 'object') return undefined
    current = (current as Record<string | number, unknown>)[segment]
  }
  return current
}

const selectorSummary = (output: EditableOutput) => {
  if (output.selector.kind === 'whole') return '完整数据集'
  if (output.selector.kind === 'column') return output.selector.column || '未选择字段'
  return (
    samplePaths.value.find((path) => path.key === pathKey(output.selector.segments))?.label ||
    '嵌套字段'
  )
}

const suggestedDataType = (
  value: unknown,
  databaseType?: CanonicalDataPointType,
): CanonicalDataPointType => {
  if (databaseType) return databaseType
  if (value === null || value === undefined) return 'string'
  if (typeof value === 'boolean') return 'bool'
  if (typeof value === 'number') return Number.isInteger(value) ? 'int64' : 'float64'
  if (typeof value === 'string') return 'string'
  if (Array.isArray(value)) return 'array'
  return 'object'
}

const normalizeOutputKey = (value: string, fallback: string) => {
  const normalized = value
    .trim()
    .replace(/[^a-zA-Z0-9_-]+/g, '_')
    .replace(/^_+|_+$/g, '')
  return normalized || fallback
}

const applySelectionDefaults = (output: EditableOutput, label: string, value: unknown) => {
  const index = outputs.value.indexOf(output) + 1
  if (props.progressive || /^(result|output\d+)$/i.test(output.key)) {
    output.key = normalizeOutputKey(label, `output${index}`)
  }
  if (props.progressive || /^(完整结果|查询结果|输出 \d+)$/u.test(output.displayName)) {
    output.displayName = label
  }
  output.dataType = suggestedDataType(value, props.columnTypes[label])
}

const emitChange = () => {
  emit(
    'update:modelValue',
    outputs.value.map((output, index) => ({ ...output, sortOrder: index })),
  )
  emit('change')
}

const enableFieldExtraction = () => {
  const output = outputs.value[0]
  const column = props.columns[0]
  if (!output || !column) return
  output.selector = { kind: 'column', column }
  applySelectionDefaults(output, column, valueAtSegments(props.sample, [column]))
  advancedMode.value = true
  emitChange()
}

const addOutput = () => {
  const index = outputs.value.length + 1
  const column =
    props.columns.find(
      (item) =>
        !outputs.value.some(
          (output) => output.selector.kind === 'column' && output.selector.column === item,
        ),
    ) || props.columns[0]
  const sampleRow =
    props.sample && typeof props.sample === 'object'
      ? (props.sample as Record<string, unknown>)
      : {}
  emit('update:modelValue', [
    ...outputs.value,
    {
      key: column ? normalizeOutputKey(column, `output${index}`) : `output${index}`,
      displayName: column || `输出 ${index}`,
      selector: column ? { kind: 'column', column } : { kind: 'whole' },
      dataType: column
        ? suggestedDataType(sampleRow[column], props.columnTypes[column])
        : props.wholeDataType,
      unit: null,
      precisionNum: null,
      sortOrder: outputs.value.length,
    },
  ])
  emit('change')
}

const removeOutput = (index: number) => {
  emit(
    'update:modelValue',
    outputs.value
      .filter((_, itemIndex) => itemIndex !== index)
      .map((output, sortOrder) => ({ ...output, sortOrder })),
  )
  emit('change')
}

const resetToWholeDataset = async () => {
  if (!isDefaultWholeDataset.value) {
    try {
      await ElMessageBox.confirm(
        '恢复后只保留完整结果集，当前列/字段提取配置将被移除。',
        '恢复完整结果集',
        {
          confirmButtonText: '确认恢复',
          cancelButtonText: '取消',
          type: 'warning',
        },
      )
    } catch {
      return
    }
  }
  const wholeOutput = createWholeSourceOutput(
    'result',
    props.defaultOutputName,
    props.wholeDataType,
  )
  const currentOutput = outputs.value.length === 1 ? outputs.value[0] : undefined
  emit('update:modelValue', [
    currentOutput
      ? {
          ...currentOutput,
          ...wholeOutput,
          id: currentOutput.id,
          datapointPath: currentOutput.datapointPath,
        }
      : wholeOutput,
  ])
  emit('change')
  advancedMode.value = false
}

const resetSelector = (output: EditableOutput) => {
  if (output.selector.kind === 'column') {
    output.selector = { kind: 'column', column: props.columns[0] || '' }
    applySelectionDefaults(
      output,
      props.columns[0] || '字段',
      valueAtSegments(props.sample, [props.columns[0]]),
    )
  } else if (output.selector.kind === 'path') {
    output.selector = { kind: 'path', segments: samplePaths.value[0]?.segments || [] }
    const selected = samplePaths.value[0]
    applySelectionDefaults(
      output,
      selected?.label || '字段',
      valueAtSegments(props.sample, selected?.segments || []),
    )
  } else {
    output.selector = { kind: 'whole' }
    output.dataType = props.wholeDataType
    if (props.progressive) {
      output.key = 'result'
      output.displayName = props.defaultOutputName
    }
  }
  emitChange()
}

const setColumn = (output: EditableOutput, column: string) => {
  output.selector = { kind: 'column', column }
  applySelectionDefaults(output, column, valueAtSegments(props.sample, [column]))
  emitChange()
}

const setPath = (output: EditableOutput, key: string) => {
  const selected = samplePaths.value.find((path) => path.key === key)
  if (selected) {
    output.selector = { kind: 'path', segments: selected.segments } as SourceOutputSelector
    applySelectionDefaults(output, selected.label, valueAtSegments(props.sample, selected.segments))
  }
  emitChange()
}
</script>

<style scoped>
.source-output-editor {
  display: grid;
  gap: 10px;
  min-width: 0;
  container-type: inline-size;
}
.source-output-editor__head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
}
.source-output-editor__head > div {
  min-width: 0;
  display: grid;
  gap: 3px;
}
.source-output-editor__head > .source-output-editor__head-actions {
  flex-shrink: 0;
  display: flex;
  justify-content: flex-end;
  gap: 6px;
}
.source-output-editor__head span,
.source-output-editor__datapoint {
  color: var(--el-text-color-secondary);
  font-size: 12px;
}
.source-output-editor__item {
  min-width: 0;
  display: grid;
  gap: 8px;
  padding: 10px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 6px;
  background: var(--el-fill-color-blank);
}
.source-output-editor__item-head,
.source-output-editor__item-head > div {
  display: flex;
  align-items: center;
  gap: 8px;
}
.source-output-editor__item-head {
  justify-content: space-between;
}
.source-output-editor__item-head strong {
  color: var(--dc-text);
  font-size: 12px;
}
.source-output-editor__dataset {
  min-width: 0;
  display: grid;
  gap: 8px;
  padding: 12px;
  border: 1px solid color-mix(in oklch, var(--dc-primary) 24%, var(--dc-border));
  border-radius: var(--dc-radius-sm);
  background: var(--dc-primary-soft);
}
.source-output-editor__dataset-main {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}
.source-output-editor__dataset-main strong {
  color: var(--dc-text);
  font-size: 13px;
}
.source-output-editor__dataset-main span {
  flex-shrink: 0;
  color: var(--dc-primary);
  font-size: 11px;
  font-weight: 700;
}
.source-output-editor__dataset p {
  margin: 0;
  color: var(--dc-text-secondary);
  font-size: 12px;
  line-height: 1.6;
}
.source-output-editor__dataset code {
  min-width: 0;
  overflow: hidden;
  color: var(--dc-primary);
  font-family: var(--dc-font-mono);
  font-size: 11px;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.source-output-editor__row {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 8px;
  align-items: end;
}
.source-output-editor__row.is-primary {
  grid-template-columns: minmax(0, 1fr) minmax(0, 1.35fr) minmax(0, 1fr) auto;
}
.source-output-editor__row.is-extraction {
  grid-template-columns: minmax(0, 1fr);
}
.source-output-editor__whole-hint {
  min-height: 32px;
  display: flex;
  align-items: center;
  padding: 0 9px;
  border-radius: 4px;
  background: var(--dc-surface-muted);
  color: var(--dc-text-secondary);
  font-size: 11px;
  line-height: 1.4;
}
.source-output-editor__field {
  min-width: 0;
  display: grid;
  gap: 4px;
}
.source-output-editor__field > span {
  color: var(--el-text-color-secondary);
  font-size: 11px;
  font-weight: 600;
}
.source-output-editor__field :deep(.el-select),
.source-output-editor__field :deep(.el-input-number) {
  width: 100%;
}
.source-output-editor__remove {
  justify-self: end;
}
.source-output-editor__path-type {
  float: right;
  margin-left: 14px;
  color: var(--el-text-color-secondary);
}
@container (max-width: 720px) {
  .source-output-editor__row,
  .source-output-editor__row.is-primary,
  .source-output-editor__row.is-extraction {
    grid-template-columns: 1fr 1fr;
  }

  .source-output-editor__remove {
    align-self: end;
  }
}

@container (max-width: 420px) {
  .source-output-editor__head {
    align-items: stretch;
    flex-direction: column;
    gap: 8px;
  }

  .source-output-editor__head > .source-output-editor__head-actions {
    flex-wrap: wrap;
  }

  .source-output-editor__row,
  .source-output-editor__row.is-primary,
  .source-output-editor__row.is-extraction {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
