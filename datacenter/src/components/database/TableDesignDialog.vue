<template>
  <DcDialog
    ref="dialogRef"
    :model-value="modelValue"
    :title="dialogTitle"
    width="1120px"
    body-max-height="calc(100vh - 180px)"
    :dirty="isDirty"
    @update:model-value="$emit('update:modelValue', $event)"
    @close="$emit('update:modelValue', false)"
  >
    <div class="table-design">
      <section class="table-design__meta">
        <label>
          <span>{{ ui('表名', 'Table Name') }}</span>
          <el-input
            v-model="draft.name"
            size="small"
            :placeholder="ui('例如 device_data', 'For example, device_data')"
            :disabled="isEditMode"
          />
        </label>
        <label>
          <span>{{ ui('表类型', 'Table Type') }}</span>
          <el-dropdown
            v-if="supportsTableKindDropdown"
            placement="bottom"
            trigger="click"
            @command="selectTableKind"
          >
            <el-button class="table-design__kind-button" size="small">
              {{ tableKindLabel }}
              <IconTablerChevronDown class="table-design__kind-arrow" />
            </el-button>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item
                  v-for="item in tableKindOptions"
                  :key="item.value"
                  :command="item.value"
                >
                  {{ item.label }}
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
          <el-button v-else class="table-design__kind-button" size="small" disabled>
            {{ tableKindLabel }}
          </el-button>
        </label>
      </section>

      <section v-if="props.supportsHypertable && !isEditMode" class="table-design__template">
        <div>
          <strong>{{ ui('采集数据模板', 'Collection Data Template') }}</strong>
          <span>{{ ui('按 ts、point_name、quality、value 创建常见采集数据时序表。', 'Create a common collection time-series table with ts, point_name, quality, and value.') }}</span>
        </div>
        <el-button size="small" @click="applyCollectionTemplate">{{ ui('使用模板', 'Use Template') }}</el-button>
      </section>

      <el-tabs v-model="activeTab" class="table-design__tabs">
        <el-tab-pane :label="ui('字段', 'Columns')" name="columns">
          <div class="table-design__toolbar">
            <button type="button" @click="addColumn">
              <IconTablerPlus />
              <span>{{ ui('添加字段', 'Add Column') }}</span>
            </button>
          </div>
          <div class="table-design__scroll">
            <div class="table-design__grid is-columns">
              <div class="table-design__head">{{ ui('字段名', 'Column Name') }}</div>
              <div class="table-design__head">{{ ui('类型', 'Type') }}</div>
              <div class="table-design__head">{{ ui('长度', 'Length') }}</div>
              <div class="table-design__head">{{ ui('数值精度', 'Numeric Precision') }}</div>
              <div class="table-design__head">{{ ui('标记', 'Flags') }}</div>
              <div class="table-design__head">{{ ui('默认值', 'Default') }}</div>
              <div class="table-design__head">{{ ui('备注', 'Comment') }}</div>
              <div class="table-design__head"></div>

              <template v-for="(column, index) in draft.columns" :key="column.localId">
                <el-input
                  v-model="column.name"
                  size="small"
                  :placeholder="ui('字段名', 'Column name')"
                  :disabled="isEditMode && column.persisted"
                />
                <el-select
                  v-model="column.type"
                  size="small"
                  :disabled="isEditMode && column.persisted"
                >
                  <el-option
                    v-for="item in columnTypeOptions"
                    :key="item.value"
                    :label="item.label"
                    :value="item.value"
                  />
                </el-select>
                <el-input-number
                  v-model="column.length"
                  size="small"
                  :min="1"
                  :max="65535"
                  controls-position="right"
                  placeholder="-"
                  :disabled="!supportsLength(column.type) || (isEditMode && column.persisted)"
                />
                <div class="table-design__precision">
                  <el-input-number
                    v-model="column.precision"
                    size="small"
                    :min="1"
                    :max="65"
                    controls-position="right"
                    :placeholder="ui('总位数', 'Precision')"
                    :disabled="!supportsPrecision(column.type) || (isEditMode && column.persisted)"
                  />
                  <el-input-number
                    v-model="column.scale"
                    size="small"
                    :min="0"
                    :max="30"
                    controls-position="right"
                    :placeholder="ui('小数位', 'Scale')"
                    :disabled="!supportsPrecision(column.type) || (isEditMode && column.persisted)"
                  />
                </div>
                <div class="table-design__checks">
                  <el-checkbox v-model="column.primary" :disabled="isEditMode && column.persisted">
                    {{ ui('主键', 'Primary Key') }}
                  </el-checkbox>
                  <el-checkbox v-model="column.nullable">{{ ui('可空', 'Nullable') }}</el-checkbox>
                  <el-checkbox
                    v-model="column.autoIncrement"
                    :disabled="
                      !supportsAutoIncrement(column.type) || (isEditMode && column.persisted)
                    "
                  >
                    {{ ui('自增', 'Auto Increment') }}
                  </el-checkbox>
                </div>
                <el-input
                  v-model="column.defaultValue"
                  size="small"
                  :placeholder="ui('默认值', 'Default value')"
                  :disabled="isEditMode && column.persisted"
                />
                <el-input v-model="column.comment" size="small" :placeholder="ui('备注', 'Comment')" />
                <button
                  type="button"
                  class="table-design__icon"
                  :disabled="draft.columns.length <= 1"
                  @click="removeColumn(index)"
                >
                  <IconTablerTrash />
                </button>
              </template>
            </div>
          </div>
        </el-tab-pane>

        <el-tab-pane :label="ui('索引', 'Indexes')" name="indexes">
          <div class="table-design__toolbar">
            <button type="button" @click="addIndex">
              <IconTablerPlus />
              <span>{{ ui('添加索引', 'Add Index') }}</span>
            </button>
          </div>
          <div class="table-design__grid is-indexes">
            <div class="table-design__head">{{ ui('索引名', 'Index Name') }}</div>
            <div class="table-design__head">{{ ui('类型', 'Type') }}</div>
            <div class="table-design__head">{{ ui('字段', 'Columns') }}</div>
            <div class="table-design__head"></div>

            <template v-for="(index, rowIndex) in draft.indexes" :key="index.localId">
              <el-input v-model="index.name" size="small" placeholder="idx_name" />
              <el-select v-model="index.type" size="small">
                <el-option :label="ui('普通索引', 'Index')" value="index" />
                <el-option :label="ui('唯一索引', 'Unique Index')" value="unique" />
              </el-select>
              <el-select
                v-model="index.columns"
                size="small"
                multiple
                collapse-tags
                collapse-tags-tooltip
                :placeholder="ui('选择字段', 'Select columns')"
              >
                <el-option
                  v-for="column in selectableColumns"
                  :key="column.name"
                  :label="column.name"
                  :value="column.name"
                />
              </el-select>
              <button type="button" class="table-design__icon" @click="removeIndex(rowIndex)">
                <IconTablerTrash />
              </button>
            </template>
          </div>
          <div v-if="draft.indexes.length === 0" class="table-design__empty">{{ ui('暂无索引', 'No indexes') }}</div>
        </el-tab-pane>

        <el-tab-pane v-if="draft.kind === 'hypertable'" :label="ui('维度字段', 'Dimension Columns')" name="dimensions">
          <div class="table-design__toolbar">
            <button type="button" @click="addTag">
              <IconTablerPlus />
              <span>{{ ui('添加维度字段', 'Add Dimension Column') }}</span>
            </button>
          </div>
          <div class="table-design__grid is-dimensions">
            <div class="table-design__head">{{ ui('字段名', 'Column Name') }}</div>
            <div class="table-design__head">{{ ui('类型', 'Type') }}</div>
            <div class="table-design__head">{{ ui('长度', 'Length') }}</div>
            <div class="table-design__head">{{ ui('备注', 'Comment') }}</div>
            <div class="table-design__head"></div>

            <template v-for="(tag, index) in draft.tags" :key="tag.localId">
              <el-input v-model="tag.name" size="small" :placeholder="ui('例如 device_id', 'For example, device_id')" />
              <el-select v-model="tag.type" size="small">
                <el-option
                  v-for="item in tagTypeOptions"
                  :key="item.value"
                  :label="item.label"
                  :value="item.value"
                />
              </el-select>
              <el-input-number
                v-model="tag.length"
                size="small"
                :min="1"
                :max="65535"
                controls-position="right"
              />
              <el-input v-model="tag.comment" size="small" :placeholder="ui('备注', 'Comment')" />
              <button type="button" class="table-design__icon" @click="removeTag(index)">
                <IconTablerTrash />
              </button>
            </template>
          </div>
          <div v-if="draft.tags.length === 0" class="table-design__empty">{{ ui('暂无维度字段', 'No dimension columns') }}</div>
        </el-tab-pane>

        <el-tab-pane v-if="draft.kind === 'hypertable'" :label="ui('时序策略', 'Time-series Policy')" name="timeseries">
          <div class="table-design__policy">
            <label>
              <span>{{ ui('时间分区字段', 'Time Partition Column') }}</span>
              <el-select v-model="draft.timeColumn" size="small" :placeholder="ui('选择时间字段', 'Select a time column')">
                <el-option
                  v-for="column in timeColumnOptions"
                  :key="column.name"
                  :label="column.name"
                  :value="column.name"
                />
              </el-select>
            </label>
            <label>
              <span class="table-design__label-line">
                {{ ui('Chunk 时间间隔', 'Chunk Interval') }}
                <el-tooltip
                  :content="ui('建议按采集频率选择：高频数据可用 1 hour 或 6 hours，普通趋势推荐 1 day，低频长期数据可用 7 days。', 'Choose by collection frequency: 1 or 6 hours for high-frequency data, 1 day for regular trends, and 7 days for low-frequency long-term data.')"
                  placement="top"
                >
                  <IconTablerInfoCircle />
                </el-tooltip>
              </span>
              <el-input v-model="draft.chunkInterval" size="small" :placeholder="ui('例如 1 day', 'For example, 1 day')" />
            </label>
            <label class="table-design__policy-switch">
              <span class="table-design__label-line">
                <el-checkbox
                  v-model="draft.retentionEnabled"
                  class="table-design__policy-checkbox"
                  :aria-label="ui('启用数据保留策略', 'Enable data retention policy')"
                />
                {{ ui('启用数据保留策略', 'Enable Data Retention Policy') }}
                <el-tooltip
                  :content="ui('开启后 TimescaleDB 会自动清理超过保留天数的历史数据；关闭则长期保留，需要自行清理。', 'When enabled, TimescaleDB automatically removes data older than the retention period; otherwise data is retained until manually removed.')"
                  placement="top"
                >
                  <IconTablerInfoCircle />
                </el-tooltip>
              </span>
            </label>
            <label v-if="draft.retentionEnabled">
              <span>{{ ui('保留天数', 'Retention Days') }}</span>
              <el-input-number
                v-model="draft.retentionDays"
                size="small"
                :min="1"
                :max="3650"
                controls-position="right"
              />
            </label>
          </div>
        </el-tab-pane>
      </el-tabs>
    </div>

    <template #footer>
      <el-button @click="requestClose">{{ ui('取消', 'Cancel') }}</el-button>
      <el-button type="primary" :loading="submitting" @click="submit">{{ submitLabel }}</el-button>
    </template>
  </DcDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import IconTablerChevronDown from '~icons/tabler/chevron-down'
import IconTablerInfoCircle from '~icons/tabler/info-circle'
import IconTablerPlus from '~icons/tabler/plus'
import IconTablerTrash from '~icons/tabler/trash'
import DcDialog from '@/components/shared/DcDialog.vue'
import { createConnectionTable, updateConnectionTableStructure } from '@/api/data.api'
import { getApiErrorMessage } from '@/utils/request'
import { datacenterLocale } from '@/i18n/runtime'

const ui = (zh: string, en: string) => (datacenterLocale.value === 'en' ? en : zh)

type DraftColumn = {
  localId: string
  name: string
  type: string
  length?: number
  precision?: number
  scale?: number
  nullable: boolean
  primary: boolean
  autoIncrement: boolean
  defaultValue: string
  comment: string
  persisted?: boolean
}

type DraftIndex = {
  localId: string
  name: string
  type: 'index' | 'unique'
  columns: string[]
}

const props = defineProps<{
  modelValue: boolean
  projectId: string | number
  connectionId: string
  dbType: string
  supportsHypertable?: boolean
  mode?: 'create' | 'edit'
  tableName?: string
  initialStructure?: Record<string, any> | null
}>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  created: [tableName: string]
  updated: [tableName: string]
}>()

const dialogRef = ref<InstanceType<typeof DcDialog> | null>(null)
const activeTab = ref('columns')
const submitting = ref(false)
const initialSnapshot = ref('')
const draft = ref(createDraft())
const isEditMode = computed(() => props.mode === 'edit')
const dialogTitle = computed(() =>
  isEditMode.value ? ui(`修改表结构 - ${props.tableName || draft.value.name}`, `Edit Table Schema - ${props.tableName || draft.value.name}`) : ui('新建表', 'New Table'),
)
const submitLabel = computed(() => (isEditMode.value ? ui('保存修改', 'Save Changes') : ui('创建', 'Create')))

const tableKindOptions = computed(() => {
  const options = [{ label: ui('普通表', 'Table'), value: 'table' }]
  if (props.supportsHypertable && !isEditMode.value) {
    options.push({ label: ui('时序超表', 'Time-series Hypertable'), value: 'hypertable' })
  }
  return options
})
const supportsTableKindDropdown = computed(() => props.supportsHypertable && !isEditMode.value)
const tableKindLabel = computed(
  () => tableKindOptions.value.find((item) => item.value === draft.value.kind)?.label || ui('普通表', 'Table'),
)

const columnTypeOptions = computed(() => {
  if (props.dbType === 'tdengine') {
    return [
      { label: 'TIMESTAMP', value: 'timestamp' },
      { label: 'INT', value: 'int' },
      { label: 'BIGINT', value: 'bigint' },
      { label: 'DOUBLE', value: 'double' },
      { label: 'BOOL', value: 'boolean' },
      { label: 'VARCHAR', value: 'varchar' },
      { label: 'NCHAR/TEXT', value: 'text' },
    ]
  }
  return [
    { label: 'VARCHAR', value: 'varchar' },
    { label: 'TEXT', value: 'text' },
    { label: 'INT', value: 'int' },
    { label: 'BIGINT', value: 'bigint' },
    { label: 'DOUBLE', value: 'double' },
    { label: 'DECIMAL', value: 'decimal' },
    { label: 'BOOLEAN', value: 'boolean' },
    { label: 'TIMESTAMP', value: 'timestamp' },
    { label: 'JSON', value: 'json' },
  ]
})
const tagTypeOptions = computed(() => [
  { label: 'VARCHAR', value: 'varchar' },
  { label: 'INT', value: 'int' },
  { label: 'BIGINT', value: 'bigint' },
  { label: 'DOUBLE', value: 'double' },
  { label: 'BOOLEAN', value: 'boolean' },
])

const selectableColumns = computed(() =>
  allDraftColumns.value.filter((column) => column.name.trim()),
)
const allDraftColumns = computed(() => [...draft.value.columns, ...draft.value.tags])
const timeColumnOptions = computed(() =>
  draft.value.columns.filter((column) =>
    ['timestamp', 'timestamptz'].includes(String(column.type || '').toLowerCase()),
  ),
)
const snapshot = computed(() => JSON.stringify(draft.value))
const isDirty = computed(() => props.modelValue && snapshot.value !== initialSnapshot.value)

function createDraft() {
  return {
    name: '',
    kind: 'table',
    columns: [
      createColumn({
        name: props?.supportsHypertable ? 'ts' : 'id',
        type: props?.supportsHypertable ? 'timestamptz' : 'bigint',
        primary: !props?.supportsHypertable,
        autoIncrement: !props?.supportsHypertable,
        nullable: false,
      }),
    ],
    indexes: [] as DraftIndex[],
    tags: [] as DraftColumn[],
    timeColumn: props?.supportsHypertable ? 'ts' : '',
    chunkInterval: '1 day',
    retentionEnabled: false,
    retentionDays: 30,
  }
}

function createDraftFromStructure() {
  const structure = props.initialStructure || {}
  return {
    name: props.tableName || '',
    kind: 'table',
    columns: (structure.columns || []).map((column: Record<string, any>) =>
      createColumn({
        name: column.name || '',
        type: normalizeStructureColumnType(column),
        length: column.maxLength || undefined,
        precision: column.numericPrecision || undefined,
        scale: column.numericScale ?? undefined,
        nullable: Boolean(column.nullable),
        primary: Boolean(column.isPrimary),
        autoIncrement: Boolean(column.autoIncrement),
        defaultValue: column.defaultValue || '',
        comment: column.comment || '',
        persisted: true,
      }),
    ),
    indexes: (structure.indexes || [])
      .filter((index: Record<string, any>) => String(index.type || '').toUpperCase() !== 'PRIMARY')
      .map((index: Record<string, any>) => ({
        localId: `index-${Date.now()}-${Math.random().toString(36).slice(2)}`,
        name: index.name || '',
        type: String(index.type || '').toUpperCase() === 'UNIQUE' ? 'unique' : 'index',
        columns: Array.isArray(index.columns) ? [...index.columns] : [],
      })),
    tags: [] as DraftColumn[],
    timeColumn: structure.timeseries?.timeColumn || '',
    chunkInterval: structure.timeseries?.chunkInterval || '1 day',
    retentionEnabled: Boolean(structure.timeseries?.retentionDays),
    retentionDays: structure.timeseries?.retentionDays || 30,
  }
}

function normalizeStructureColumnType(column: Record<string, any>) {
  const type = String(column.type || '').toLowerCase()
  if (type === 'character varying') return 'varchar'
  if (type === 'integer') return 'int'
  if (type === 'double precision') return 'double'
  if (type === 'numeric') return 'decimal'
  if (type === 'timestamp without time zone') return 'timestamp'
  if (type === 'timestamp with time zone') return 'timestamptz'
  if (type === 'jsonb') return 'json'
  return type || 'varchar'
}

function createColumn(patch: Partial<DraftColumn> = {}): DraftColumn {
  return {
    localId: `column-${Date.now()}-${Math.random().toString(36).slice(2)}`,
    name: '',
    type: 'varchar',
    length: 255,
    precision: undefined,
    scale: undefined,
    nullable: true,
    primary: false,
    autoIncrement: false,
    defaultValue: '',
    comment: '',
    ...patch,
  }
}

function createIndex(): DraftIndex {
  return {
    localId: `index-${Date.now()}-${Math.random().toString(36).slice(2)}`,
    name: '',
    type: 'index',
    columns: [],
  }
}

function isDefaultCreateDraft() {
  if (draft.value.name.trim() || draft.value.tags.length > 0 || draft.value.indexes.length > 0) {
    return false
  }
  if (draft.value.columns.length !== 1) return false
  const column = draft.value.columns[0]
  return (
    column.name === 'ts' &&
    column.type === 'timestamptz' &&
    !column.nullable &&
    draft.value.kind === 'table' &&
    draft.value.timeColumn === 'ts' &&
    draft.value.chunkInterval === '1 day' &&
    !draft.value.retentionEnabled &&
    draft.value.retentionDays === 30
  )
}

async function applyCollectionTemplate() {
  if (!isDefaultCreateDraft()) {
    try {
      await ElMessageBox.confirm(
        ui('采集数据模板会替换当前字段、维度字段和时序策略，是否继续？', 'The collection template replaces current columns, dimension columns, and time-series policy. Continue?'),
        ui('使用采集数据模板', 'Use Collection Template'),
        {
          confirmButtonText: ui('使用模板', 'Use Template'),
          cancelButtonText: ui('取消', 'Cancel'),
          type: 'warning',
        },
      )
    } catch {
      return
    }
  }
  const tableName = draft.value.name.trim() || 'point_samples'
  draft.value = {
    ...draft.value,
    name: tableName,
    kind: 'hypertable',
    columns: [
      createColumn({ name: 'ts', type: 'timestamptz', nullable: false }),
      createColumn({
        name: 'quality',
        type: 'int',
        nullable: false,
        comment: ui('质量码，常见约定 192=GOOD', 'Quality code; 192 commonly means GOOD'),
      }),
      createColumn({ name: 'value', type: 'double', nullable: true, comment: ui('采集值', 'Collected value') }),
    ],
    indexes: [
      {
        localId: `index-${Date.now()}-${Math.random().toString(36).slice(2)}`,
        name: 'idx_point_samples_point_time',
        type: 'index',
        columns: ['point_name', 'ts'],
      },
    ],
    tags: [
      createColumn({
        name: 'point_name',
        type: 'varchar',
        length: 128,
        nullable: false,
        comment: ui('变量名或点位标识', 'Variable name or point identifier'),
      }),
    ],
    timeColumn: 'ts',
    chunkInterval: '1 day',
    retentionEnabled: false,
    retentionDays: 30,
  }
  activeTab.value = 'columns'
}

function resetDraft() {
  draft.value = isEditMode.value ? createDraftFromStructure() : createDraft()
  activeTab.value = 'columns'
  initialSnapshot.value = snapshot.value
}

function selectTableKind(kind: string | number | object) {
  draft.value.kind = String(kind)
  if (draft.value.kind === 'hypertable') {
    ensureHypertableDefaults()
  }
  if (draft.value.kind !== 'hypertable' && ['dimensions', 'timeseries'].includes(activeTab.value)) {
    activeTab.value = 'columns'
  }
}

function ensureHypertableDefaults() {
  if (!draft.value.timeColumn) {
    const timestampColumn = timeColumnOptions.value[0]
    draft.value.timeColumn = timestampColumn?.name || ''
  }
  draft.value.columns.forEach((column) => {
    if (column.name === draft.value.timeColumn) {
      column.primary = false
      column.autoIncrement = false
      column.nullable = false
    }
  })
}

function supportsLength(type: string) {
  return ['varchar', 'text'].includes(type)
}

function supportsPrecision(type: string) {
  return type === 'decimal'
}

function supportsAutoIncrement(type: string) {
  return ['int', 'bigint'].includes(type)
}

function addColumn() {
  draft.value.columns.push(createColumn())
}

function removeColumn(index: number) {
  const [removed] = draft.value.columns.splice(index, 1)
  if (draft.value.timeColumn === removed?.name) {
    draft.value.timeColumn = timeColumnOptions.value[0]?.name || ''
  }
  if (!removed?.name) return
  draft.value.indexes.forEach((item) => {
    item.columns = item.columns.filter((column) => column !== removed.name)
  })
}

function addIndex() {
  draft.value.indexes.push(createIndex())
}

function removeIndex(index: number) {
  draft.value.indexes.splice(index, 1)
}

function addTag() {
  draft.value.tags.push(createColumn({ type: 'varchar', length: 64, nullable: false }))
}

function removeTag(index: number) {
  draft.value.tags.splice(index, 1)
}

function validateDraft() {
  if (!draft.value.name.trim()) return ui('请输入表名', 'Enter a table name')
  const names = new Set<string>()
  for (const column of draft.value.columns) {
    const name = column.name.trim()
    if (!name) return ui('字段名不能为空', 'Column name is required')
    if (names.has(name)) return ui(`字段名重复：${name}`, `Duplicate column name: ${name}`)
    names.add(name)
  }
  if (!draft.value.columns.some((column) => column.primary) && draft.value.kind === 'table') {
    return ui('普通表至少需要一个主键字段', 'A table requires at least one primary key column')
  }
  for (const index of draft.value.indexes) {
    if (!index.name.trim()) return ui('索引名不能为空', 'Index name is required')
    if (index.columns.length === 0) return ui(`索引 ${index.name || ''} 需要选择字段`, `Select columns for index ${index.name || ''}`)
  }
  if (draft.value.kind === 'hypertable') {
    if (!draft.value.timeColumn) return ui('请选择时间分区字段', 'Select a time partition column')
    if (!draft.value.chunkInterval.trim()) return ui('请输入 Chunk 时间间隔', 'Enter a chunk interval')
    const tagNames = new Set<string>()
    for (const tag of draft.value.tags) {
      const name = tag.name.trim()
      if (!name) return ui('维度字段名不能为空', 'Dimension column name is required')
      if (names.has(name)) return ui(`维度字段不能与数据字段重名：${name}`, `Dimension column duplicates a data column: ${name}`)
      if (tagNames.has(name)) return ui(`维度字段名重复：${name}`, `Duplicate dimension column: ${name}`)
      tagNames.add(name)
    }
  }
  return ''
}

function buildPayload() {
  return {
    name: draft.value.name.trim(),
    kind: draft.value.kind,
    columns: draft.value.columns.map((column) => ({
      name: column.name.trim(),
      type: column.type,
      length: supportsLength(column.type) ? column.length || undefined : undefined,
      precision: supportsPrecision(column.type) ? column.precision || undefined : undefined,
      scale: supportsPrecision(column.type) ? (column.scale ?? undefined) : undefined,
      nullable: Boolean(column.nullable),
      primary: Boolean(column.primary),
      autoIncrement: supportsAutoIncrement(column.type) && Boolean(column.autoIncrement),
      defaultValue: column.defaultValue.trim(),
      comment: column.comment.trim(),
    })),
    indexes: draft.value.indexes.map((index) => ({
      name: index.name.trim(),
      type: index.type,
      columns: [...index.columns],
    })),
    timeseries:
      draft.value.kind === 'hypertable'
        ? {
            timeColumn: draft.value.timeColumn,
            dimensionColumns: draft.value.tags.map((tag) => ({
              name: tag.name.trim(),
              type: tag.type,
              length: tag.length || undefined,
              nullable: false,
              primary: false,
              autoIncrement: false,
              defaultValue: '',
              comment: tag.comment.trim(),
            })),
            chunkInterval: draft.value.chunkInterval.trim(),
            retentionDays: draft.value.retentionEnabled
              ? draft.value.retentionDays || undefined
              : undefined,
          }
        : undefined,
  }
}

async function submit() {
  const message = validateDraft()
  if (message) {
    ElMessage.warning(message)
    return
  }
  submitting.value = true
  try {
    const payload = buildPayload()
    if (isEditMode.value) {
      await updateConnectionTableStructure(props.projectId, props.connectionId, payload.name, {
        columns: payload.columns,
        indexes: payload.indexes,
      })
      ElMessage.success(ui('表结构已修改', 'Table schema updated'))
      initialSnapshot.value = snapshot.value
      emit('updated', payload.name)
      dialogRef.value?.closeSilently()
      return
    }
    await createConnectionTable(props.projectId, props.connectionId, payload)
    ElMessage.success(ui('表已创建', 'Table created'))
    initialSnapshot.value = snapshot.value
    emit('created', payload.name)
    dialogRef.value?.closeSilently()
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, ui('创建表失败', 'Failed to create table')))
  } finally {
    submitting.value = false
  }
}

function requestClose() {
  void dialogRef.value?.requestClose()
}

watch(
  () => props.modelValue,
  (visible) => {
    if (visible) resetDraft()
  },
)
</script>

<style scoped>
.table-design {
  display: grid;
  gap: 12px;
}

.table-design__meta {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 260px;
  gap: 12px;
}

.table-design__meta label {
  display: grid;
  gap: 6px;
  color: var(--dc-text-secondary);
  font-size: 12px;
  font-weight: 700;
}

.table-design__template {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 10px 12px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-muted);
}

.table-design__template div {
  display: grid;
  gap: 3px;
}

.table-design__template strong {
  color: var(--dc-text);
  font-size: 13px;
}

.table-design__template span {
  color: var(--dc-text-muted);
  font-size: 12px;
}

.table-design__kind-button {
  width: 100%;
  justify-content: space-between;
}

.table-design__kind-arrow {
  width: 14px;
  height: 14px;
}

.table-design__tabs {
  min-height: 360px;
}

.table-design__toolbar {
  display: flex;
  justify-content: flex-end;
  margin-bottom: 8px;
}

.table-design__toolbar button,
.table-design__icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
  color: var(--dc-text-secondary);
}

.table-design__toolbar button {
  height: 30px;
  gap: 6px;
  padding: 0 10px;
  font-size: 12px;
  font-weight: 700;
}

.table-design__toolbar svg,
.table-design__icon svg {
  width: 15px;
  height: 15px;
}

.table-design__grid {
  display: grid;
  align-items: center;
  gap: 6px;
}

.table-design__scroll {
  max-width: 100%;
  overflow-x: auto;
  padding-bottom: 4px;
}

.table-design__grid.is-columns {
  min-width: 1040px;
  grid-template-columns: 160px 140px 100px 136px 150px 150px 160px 30px;
}

.table-design__grid.is-indexes {
  grid-template-columns: 180px 140px minmax(220px, 1fr) 30px;
}

.table-design__grid.is-dimensions {
  grid-template-columns: 180px 150px 120px minmax(180px, 1fr) 30px;
}

.table-design__policy {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}

.table-design__policy label {
  display: grid;
  gap: 6px;
  color: var(--dc-text-secondary);
  font-size: 12px;
  font-weight: 700;
}

.table-design__policy-switch {
  align-content: start;
  padding-top: 22px;
}

.table-design__label-line {
  display: inline-flex;
  align-items: center;
  gap: 5px;
}

.table-design__label-line svg {
  width: 14px;
  height: 14px;
  color: var(--dc-text-muted);
  cursor: help;
}

.table-design__label-line svg:hover {
  color: var(--dc-primary);
}

.table-design__policy-checkbox {
  height: 16px;
}

.table-design__policy-checkbox :deep(.el-checkbox__label) {
  display: none;
}

.table-design__head {
  color: var(--dc-text-muted);
  font-size: 11px;
  font-weight: 700;
}

.table-design__precision {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 4px;
}

.table-design__checks {
  display: flex;
  align-items: center;
  gap: 8px;
  white-space: nowrap;
}

.table-design__checks :deep(.el-checkbox) {
  margin-right: 0;
  height: 28px;
}

.table-design__checks :deep(.el-checkbox__label) {
  padding-left: 4px;
  font-size: 12px;
}

.table-design :deep(.el-input-number) {
  width: 100%;
}

.table-design :deep(.el-input-number .el-input__inner) {
  text-align: left;
}

.table-design__icon {
  width: 30px;
  height: 30px;
}

.table-design__icon:disabled {
  cursor: not-allowed;
  opacity: 0.45;
}

.table-design__empty {
  padding: 24px;
  color: var(--dc-text-muted);
  font-size: 12px;
  text-align: center;
}
</style>
