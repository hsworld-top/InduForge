<template>
  <DcDialog
    ref="dialogRef"
    v-model="visible"
    title="导入 S7 地址表"
    width="860px"
    body-max-height="calc(100vh - 180px)"
    :dirty="isDirty"
    :close-disabled="loading"
  >
    <div class="s7-import-dialog__summary">
      <strong>{{ rows.length }}</strong>
      <span>待导入变量</span>
      <em>{{ importModeText }}</em>
    </div>
    <div class="s7-import-dialog__tools">
      <el-upload
        :auto-upload="false"
        :show-file-list="false"
        accept=".csv,.tsv,.xlsx,.xls"
        :on-change="handleFileChange"
      >
        <el-button size="small">
          <IconTablerFileImport />
          选择文件
        </el-button>
      </el-upload>
      <el-button size="small" @click="downloadTemplate('csv')">
        <IconTablerFileTypeCsv />
        CSV模板
      </el-button>
      <el-button size="small" @click="downloadTemplate('xlsx')">
        <IconTablerFileSpreadsheet />
        XLSX模板
      </el-button>
      <el-checkbox v-model="onlyIssues" size="small">只看错误</el-checkbox>
      <el-button size="small" :disabled="issueRows.length === 0" @click="downloadIssueReport">
        <IconTablerAlertTriangle />
        错误报告
      </el-button>
      <span v-if="fileName" class="s7-import-dialog__file" :title="fileName">{{ fileName }}</span>
    </div>
    <el-input
      v-model="text"
      type="textarea"
      :rows="6"
      placeholder="变量名,Code,分组,地址,数据类型,单位,倍率,偏移,采集周期,发布能力,描述"
      @input="clearFileSource"
    />
    <el-table
      class="s7-import-dialog__table"
      :data="displayRows"
      height="260"
      empty-text="选择文件或粘贴表格文本后预览"
    >
      <el-table-column label="状态" width="76"
        ><template #default="{ row }"
          ><el-tag size="small" :type="row.issue ? 'warning' : 'success'">{{
            row.issue ? '检查' : '可导入'
          }}</el-tag></template
        ></el-table-column
      >
      <el-table-column prop="name" label="变量名" min-width="130" show-overflow-tooltip />
      <el-table-column prop="code" label="Code" min-width="120" show-overflow-tooltip />
      <el-table-column prop="addressText" label="原始地址" min-width="130" show-overflow-tooltip />
      <el-table-column
        prop="normalizedAddress"
        label="标准地址"
        min-width="130"
        show-overflow-tooltip
      />
      <el-table-column prop="area" label="区域" width="72" />
      <el-table-column prop="byteRange" label="字节范围" width="96" />
      <el-table-column prop="dataType" label="类型" width="90" />
      <el-table-column prop="accessLevel" label="权限" width="92" />
      <el-table-column prop="issue" label="问题" min-width="160" show-overflow-tooltip />
    </el-table>
    <template #footer>
      <el-button @click="requestClose">取消</el-button>
      <el-button
        type="primary"
        :disabled="validRows.length === 0"
        :loading="loading"
        @click="submit"
        >导入 {{ validRows.length }} 个变量</el-button
      >
    </template>
  </DcDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import type { UploadFile } from 'element-plus'
import { ElMessage } from 'element-plus'
import DcDialog from '@/components/shared/DcDialog.vue'
import {
  downloadCsv,
  downloadXlsx,
  normalizeHeaderRow,
  parseDelimitedRows,
  parseTabularText,
  readTabularFile,
} from '@/utils/tabular-file'
import IconTablerAlertTriangle from '~icons/tabler/alert-triangle'
import IconTablerFileImport from '~icons/tabler/file-import'
import IconTablerFileSpreadsheet from '~icons/tabler/file-spreadsheet'
import IconTablerFileTypeCsv from '~icons/tabler/file-type-csv'

const props = defineProps<{ modelValue: boolean; loading?: boolean }>()
const emit = defineEmits<{
  (event: 'update:modelValue', value: boolean): void
  (event: 'submit', rows: Array<Record<string, unknown>>): void
}>()
const visible = computed({
  get: () => props.modelValue,
  set: (value: boolean) => emit('update:modelValue', value),
})
const dialogRef = ref<InstanceType<typeof DcDialog> | null>(null)
const text = ref('')
const fileRows = ref<Record<string, string>[]>([])
const fileName = ref('')
const onlyIssues = ref(false)
watch(
  () => props.modelValue,
  (visible) => {
    if (visible) {
      text.value = ''
      fileRows.value = []
      fileName.value = ''
      onlyIssues.value = false
    }
  },
)

type S7ImportPreviewRow = {
  name: string
  code: string
  groupPath: string
  addressText: string
  normalizedAddress: string
  area: string
  dbNumber: number | null
  byteRange: string
  bitOffset: number | null
  dataType: string
  unit: string | null
  scale: number
  offset: number
  pollIntervalMs: number
  accessLevel: string
  description: string | null
  metadata: Record<string, unknown>
  sortOrder: number
  issue: string
}

const headerAliases = {
  name: ['变量名', '名称', 'name', 'variableName'],
  code: ['Code', '编码', '变量编码', 'code'],
  groupPath: ['分组', '分组路径', 'group', 'groupPath'],
  addressText: ['地址', 'S7地址', 'address', 'addressText'],
  dataType: ['数据类型', '类型', 'dataType', 'type'],
  unit: ['单位', 'unit'],
  scale: ['倍率', '系数', 'scale'],
  offset: ['偏移', 'offset'],
  pollIntervalMs: ['采集周期', '周期', 'pollIntervalMs', 'interval'],
  accessLevel: ['发布能力', '读写能力', 'accessLevel', 'publishAccess'],
  description: ['描述', '说明', 'description'],
}

const templateHeaders = [
  '变量名',
  'Code',
  '分组',
  '地址',
  '数据类型',
  '单位',
  '倍率',
  '偏移',
  '采集周期',
  '发布能力',
  '描述',
]
const templateRows = [
  {
    变量名: '电机转速',
    Code: 'motor_speed',
    分组: '一号线/电机',
    地址: 'DB1.DBD0',
    数据类型: 'Real',
    单位: 'rpm',
    倍率: 1,
    偏移: 0,
    采集周期: 1000,
    发布能力: 'Read',
    描述: '主电机转速',
  },
  {
    变量名: '急停状态',
    Code: 'emergency_stop',
    分组: '一号线/安全',
    地址: 'I0.0',
    数据类型: 'Bool',
    单位: '',
    倍率: 1,
    偏移: 0,
    采集周期: 500,
    发布能力: 'Read',
    描述: '现场急停输入',
  },
]

const importModeText = computed(() => (fileName.value ? '文件地址表' : '粘贴地址表'))
const isDirty = computed(
  () =>
    props.modelValue &&
    (text.value.trim().length > 0 || fileRows.value.length > 0 || Boolean(fileName.value)),
)
const sourceRows = computed(() => {
  if (fileRows.value.length > 0) return fileRows.value
  return parsePasteRows(text.value)
})
const rows = computed<S7ImportPreviewRow[]>(() =>
  sourceRows.value.map((row, index) => buildPreviewRow(row, index)),
)
const displayRows = computed(() =>
  onlyIssues.value ? rows.value.filter((row) => row.issue) : rows.value,
)
const issueRows = computed(() => rows.value.filter((row) => row.issue))
const validRows = computed(() =>
  rows.value
    .filter((row) => !row.issue)
    .map(
      ({
        issue: _issue,
        normalizedAddress,
        area,
        dbNumber,
        byteRange,
        bitOffset,
        groupPath,
        ...row
      }) => ({
        ...row,
        metadata: {
          ...row.metadata,
          importGroupPath: groupPath || undefined,
          normalizedPreview: normalizedAddress,
          addressPreview: {
            area,
            dbNumber,
            byteRange,
            bitOffset,
          },
        },
      }),
    ),
)

const handleFileChange = async (uploadFile: UploadFile) => {
  const raw = uploadFile.raw
  if (!raw) return
  try {
    fileRows.value = await readTabularFile(raw)
    fileName.value = raw.name
    text.value = ''
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '读取导入文件失败')
  }
}

const clearFileSource = () => {
  if (fileRows.value.length === 0) return
  fileRows.value = []
  fileName.value = ''
}

const downloadTemplate = (type: 'csv' | 'xlsx') => {
  if (type === 'csv') {
    downloadCsv('s7-variable-import-template.csv', templateHeaders, templateRows)
    return
  }
  downloadXlsx('s7-variable-import-template.xlsx', [
    { name: 'S7变量模板', headers: templateHeaders, rows: templateRows },
  ])
}

const downloadIssueReport = () => {
  downloadCsv(
    's7-import-issues.csv',
    ['行号', '变量名', 'Code', '分组', '原始地址', '标准地址', '类型', '问题'],
    issueRows.value.map((row) => ({
      行号: row.sortOrder + 1,
      变量名: row.name,
      Code: row.code,
      分组: row.groupPath,
      原始地址: row.addressText,
      标准地址: row.normalizedAddress,
      类型: row.dataType,
      问题: row.issue,
    })),
  )
}

function submit() {
  emit('submit', validRows.value)
}

function requestClose() {
  void dialogRef.value?.requestClose()
}

function closeSilently() {
  dialogRef.value?.closeSilently()
}

function parsePasteRows(value: string) {
  const parsed = parseTabularText(value)
  const headerMatched = parsed.some((row) =>
    Object.keys(row).some((header) =>
      Object.values(headerAliases).some((aliases) =>
        aliases.map((alias) => alias.toLowerCase()).includes(header.trim().toLowerCase()),
      ),
    ),
  )
  if (headerMatched || value.trim() === '') return parsed
  const headerlessRows = parseDelimitedRows(value).filter((row) =>
    row.some((cell) => cell.trim() !== ''),
  )
  return headerlessRows.map((row) =>
    Object.fromEntries(templateHeaders.map((header, index) => [header, row[index]?.trim() || ''])),
  )
}

function buildPreviewRow(row: Record<string, string>, index: number): S7ImportPreviewRow {
  const normalized = normalizeHeaderRow(row, headerAliases)
  const dataType = normalizeDataType(normalized.dataType)
  const addressText = normalized.addressText
  const preview = parseS7AddressPreview(addressText, dataType)
  const issue = firstIssue([
    !addressText ? '地址为空' : '',
    preview.issue,
    Number.isNaN(toNumber(normalized.scale, 1)) ? '倍率不是数字' : '',
    Number.isNaN(toNumber(normalized.offset, 0)) ? '偏移不是数字' : '',
    Number.isNaN(toNumber(normalized.pollIntervalMs, 1000)) ? '采集周期不是数字' : '',
  ])
  return {
    name: normalized.name || addressText || `变量${index + 1}`,
    code: normalized.code,
    groupPath: normalized.groupPath,
    addressText,
    normalizedAddress: preview.normalizedAddress,
    area: preview.area,
    dbNumber: preview.dbNumber,
    byteRange: preview.byteRange,
    bitOffset: preview.bitOffset,
    dataType,
    unit: normalized.unit || null,
    scale: toNumber(normalized.scale, 1),
    offset: toNumber(normalized.offset, 0),
    pollIntervalMs: toInteger(normalized.pollIntervalMs, 1000),
    accessLevel: normalizeAccessLevel(normalized.accessLevel),
    description: normalized.description || null,
    metadata: { importSource: fileName.value ? 'file' : 'paste' },
    sortOrder: index,
    issue,
  }
}

function parseS7AddressPreview(addressText: string, dataType: string) {
  const text = addressText.trim().toUpperCase()
  if (!text) return emptyAddressPreview('地址为空')
  const dbMatch = text.match(/^DB(\d+)\.DB([XBWDL])(\d+)(?:\.(\d))?$/i)
  if (dbMatch) {
    const dbNumber = Number(dbMatch[1])
    const addressType = `DB${dbMatch[2].toUpperCase()}`
    const byteOffset = Number(dbMatch[3])
    const bitOffset = dbMatch[4] === undefined ? null : Number(dbMatch[4])
    return normalizeAddressPreview('DB', dbNumber, addressType, byteOffset, bitOffset, dataType)
  }
  const areaMatch = text.match(/^([MIQ])([BWD]?)(\d+)(?:\.(\d))?$/i)
  if (areaMatch) {
    const area = areaMatch[1].toUpperCase()
    const width = (areaMatch[2] || (areaMatch[4] !== undefined ? 'X' : '')).toUpperCase()
    const byteOffset = Number(areaMatch[3])
    const bitOffset = areaMatch[4] === undefined ? null : Number(areaMatch[4])
    return normalizeAddressPreview(area, null, `${area}${width}`, byteOffset, bitOffset, dataType)
  }
  return emptyAddressPreview('S7 地址格式不支持')
}

function normalizeAddressPreview(
  area: string,
  dbNumber: number | null,
  addressType: string,
  byteOffset: number,
  bitOffset: number | null,
  dataType: string,
) {
  if (bitOffset !== null && (bitOffset < 0 || bitOffset > 7))
    return emptyAddressPreview('bit 位必须在 0-7')
  const isBoolAddress = addressType.endsWith('X') || dataType.toLowerCase() === 'bool'
  if (isBoolAddress && bitOffset === null) return emptyAddressPreview('Bool 地址必须包含 bit 位')
  const normalizedAddress =
    area === 'DB' && dbNumber !== null
      ? isBoolAddress
        ? `DB${dbNumber}.DBX${byteOffset}.${bitOffset}`
        : `DB${dbNumber}.${addressType}${byteOffset}`
      : isBoolAddress
        ? `${area}${byteOffset}.${bitOffset}`
        : `${addressType}${byteOffset}`
  return {
    normalizedAddress,
    area,
    dbNumber,
    byteRange: `${byteOffset}-${byteOffset + estimateDataTypeBytes(dataType) - 1}`,
    bitOffset,
    issue: '',
  }
}

function emptyAddressPreview(issue: string) {
  return { normalizedAddress: '', area: '', dbNumber: null, byteRange: '', bitOffset: null, issue }
}

function normalizeDataType(value: string) {
  const matched = [
    'Bool',
    'Byte',
    'Word',
    'DWord',
    'Int',
    'DInt',
    'Real',
    'DateTime',
    'String',
  ].find((item) => item.toLowerCase() === value.trim().toLowerCase())
  return matched || 'Real'
}

function estimateDataTypeBytes(dataType: string) {
  const map: Record<string, number> = {
    Bool: 1,
    Byte: 1,
    Word: 2,
    Int: 2,
    DWord: 4,
    DInt: 4,
    Real: 4,
    DateTime: 8,
    String: 254,
  }
  return map[dataType] || 1
}

function normalizeAccessLevel(value: string) {
  const text = value.trim().toLowerCase()
  if (['write', '写', 'readwrite', 'read_write', '读写'].includes(text)) return 'ReadWrite'
  return 'Read'
}

function toNumber(value: string, fallback: number) {
  if (value === '') return fallback
  return Number(value)
}

function toInteger(value: string, fallback: number) {
  const parsed = toNumber(value, fallback)
  return Number.isNaN(parsed) ? parsed : Math.trunc(parsed)
}

function firstIssue(issues: string[]) {
  return issues.find(Boolean) || ''
}

defineExpose({ closeSilently })
</script>

<style scoped>
.s7-import-dialog__summary {
  margin-bottom: 10px;
  padding: 10px 12px;
  border: 1px solid color-mix(in oklch, var(--dc-primary) 18%, var(--dc-border));
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-subtle);
  display: flex;
  align-items: center;
  gap: 8px;
}
.s7-import-dialog__summary strong {
  color: var(--dc-primary);
  font-size: 18px;
}
.s7-import-dialog__summary span {
  color: var(--dc-text-secondary);
  font-size: 12px;
  font-weight: 700;
}
.s7-import-dialog__summary em {
  margin-left: auto;
  color: var(--dc-text-muted);
  font-style: normal;
  font-size: 12px;
}
.s7-import-dialog__tools {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 10px;
}
.s7-import-dialog__tools :deep(.el-button) {
  gap: 5px;
}
.s7-import-dialog__tools svg {
  width: 14px;
  height: 14px;
}
.s7-import-dialog__file {
  min-width: 0;
  overflow: hidden;
  color: var(--dc-text-muted);
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 12px;
}
.s7-import-dialog__table {
  margin-top: 12px;
}
</style>
