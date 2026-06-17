<template>
  <DcDialog
    ref="dialogRef"
    v-model="visible"
    title="导入 Modbus 变量"
    width="860px"
    body-max-height="calc(100vh - 180px)"
    :dirty="isDirty"
    :close-disabled="loading"
  >
    <div class="modbus-import-dialog__summary">
      <strong>{{ previewRows.length }}</strong>
      <span>待导入变量</span>
      <em>{{ mode === 'paste' ? '粘贴表格' : '地址段生成' }}</em>
    </div>
    <div class="modbus-import-dialog__tools">
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
      <span v-if="fileName" class="modbus-import-dialog__file" :title="fileName">{{
        fileName
      }}</span>
    </div>
    <div class="modbus-import-dialog__defaults">
      <el-select v-model="selectedDefaultGroupId" clearable filterable placeholder="默认分组">
        <el-option label="未分组" value="" />
        <el-option v-for="group in groups" :key="group.id" :label="group.name" :value="group.id" />
      </el-select>
      <el-select v-model="selectedDefaultUnitId" filterable placeholder="默认从站">
        <el-option
          v-for="slave in slaves"
          :key="slave.id"
          :label="`${slave.name} (${slave.unitId})`"
          :value="slave.unitId"
        />
      </el-select>
      <el-select v-model="defaultByteOrder" placeholder="默认字节序">
        <el-option label="ABCD" value="ABCD" />
        <el-option label="BADC" value="BADC" />
        <el-option label="CDAB" value="CDAB" />
        <el-option label="DCBA" value="DCBA" />
      </el-select>
      <el-input-number v-model="defaultPollIntervalMs" :min="100" :step="100" />
    </div>
    <el-tabs v-model="mode">
      <el-tab-pane label="粘贴表格" name="paste">
        <el-input
          v-model="pasteText"
          type="textarea"
          :rows="9"
          placeholder="变量名,Code,从站地址,区域,地址,地址基准,类型,字节序,字序,倍率,偏移,单位,采集周期,发布能力,描述"
          @input="clearFileSource"
        />
      </el-tab-pane>
      <el-tab-pane label="地址段生成" name="range">
        <div class="modbus-import-dialog__range">
          <el-input-number v-model="range.unitId" :min="0" :max="247" />
          <el-select v-model="range.area">
            <el-option label="Holding Register" value="holding_register" />
            <el-option label="Input Register" value="input_register" />
            <el-option label="Coil" value="coil" />
            <el-option label="Discrete Input" value="discrete_input" />
          </el-select>
          <el-input-number v-model="range.startAddress" :min="0" />
          <el-input-number v-model="range.count" :min="1" :max="500" />
          <el-select v-model="range.dataType">
            <el-option label="uint16" value="uint16" />
            <el-option label="bool" value="bool" />
            <el-option label="float32" value="float32" />
          </el-select>
          <el-input v-model="range.prefix" />
        </div>
      </el-tab-pane>
    </el-tabs>
    <el-table :data="displayRows" height="230px" row-key="rowNo">
      <el-table-column label="状态" width="74">
        <template #default="{ row }">
          <el-tag size="small" :type="row.issue ? 'warning' : 'success'">
            {{ row.issue ? '检查' : '可导入' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="name" label="变量名" />
      <el-table-column prop="code" label="Code" />
      <el-table-column prop="unitId" label="从站" width="70" />
      <el-table-column prop="area" label="区域" width="130" />
      <el-table-column prop="address" label="地址" width="90" />
      <el-table-column prop="protocolAddress" label="协议地址" width="92" />
      <el-table-column prop="dataType" label="类型" width="90" />
      <el-table-column prop="issue" label="问题" min-width="150" show-overflow-tooltip />
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
import { computed, reactive, ref, watch } from 'vue'
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
import type { ModbusRegisterGroup, ModbusSlaveDevice } from './types'
import IconTablerAlertTriangle from '~icons/tabler/alert-triangle'
import IconTablerFileImport from '~icons/tabler/file-import'
import IconTablerFileSpreadsheet from '~icons/tabler/file-spreadsheet'
import IconTablerFileTypeCsv from '~icons/tabler/file-type-csv'

const props = defineProps<{
  modelValue: boolean
  loading?: boolean
  groups?: ModbusRegisterGroup[]
  slaves?: ModbusSlaveDevice[]
  defaultGroupId?: string
  defaultUnitId?: number
}>()
const emit = defineEmits<{
  (event: 'update:modelValue', value: boolean): void
  (event: 'submit', rows: Array<Record<string, unknown>>): void
}>()

const visible = computed({
  get: () => props.modelValue,
  set: (value: boolean) => emit('update:modelValue', value),
})
const dialogRef = ref<InstanceType<typeof DcDialog> | null>(null)
const mode = ref('paste')
const pasteText = ref('')
const fileRows = ref<Record<string, string>[]>([])
const fileName = ref('')
const onlyIssues = ref(false)
const selectedDefaultGroupId = ref('')
const selectedDefaultUnitId = ref(1)
const defaultByteOrder = ref('ABCD')
const defaultPollIntervalMs = ref(1000)
const range = reactive({
  unitId: 1,
  area: 'holding_register',
  startAddress: 40001,
  count: 20,
  dataType: 'uint16',
  prefix: 'modbus_reg_',
})

const groups = computed(() => props.groups || [])
const slaves = computed(() =>
  (props.slaves || []).length > 0
    ? props.slaves || []
    : [
        {
          id: 'default',
          unitId: props.defaultUnitId ?? 1,
          name: `从站 ${props.defaultUnitId ?? 1}`,
          enabled: true,
          defaultPollIntervalMs: 1000,
          defaultByteOrder: 'ABCD',
          defaultWordOrder: 'high_first',
        },
      ],
)

const initialRangeSnapshot = JSON.stringify({ ...range })
const isDirty = computed(
  () =>
    props.modelValue &&
    (pasteText.value.trim().length > 0 ||
      fileRows.value.length > 0 ||
      Boolean(fileName.value) ||
      (mode.value === 'range' && JSON.stringify({ ...range }) !== initialRangeSnapshot)),
)

const previewRows = computed(() => {
  if (mode.value === 'range') {
    return Array.from({ length: range.count }, (_, index) => ({
      rowNo: index + 1,
      name: `${range.prefix}${index + 1}`,
      code: `${range.prefix}${index + 1}`,
      unitId: range.unitId,
      area: range.area,
      address: range.startAddress + index,
      addressBase: 'modicon',
      protocolAddress: normalizeProtocolAddress(range.area, 'modicon', range.startAddress + index),
      dataType: range.dataType,
      byteOrder: defaultByteOrder.value,
      wordOrder: 'high_first',
      scale: 1,
      offset: 0,
      pollIntervalMs: defaultPollIntervalMs.value,
      accessLevel: defaultAccessLevel(range.area),
      issue: '',
    }))
  }
  return sourceRows.value.map((row, index) => buildPreviewRow(row, index))
})

const displayRows = computed(() =>
  onlyIssues.value ? previewRows.value.filter((row) => row.issue) : previewRows.value,
)
const issueRows = computed(() => previewRows.value.filter((row) => row.issue))
const validRows = computed(() =>
  previewRows.value
    .filter((row) => !row.issue)
    .map(({ issue: _issue, rowNo: _rowNo, protocolAddress: _protocolAddress, ...row }) => ({
      ...row,
      groupId: selectedDefaultGroupId.value || null,
    })),
)
const sourceRows = computed(() => {
  if (fileRows.value.length > 0) return fileRows.value
  return parsePasteRows(pasteText.value)
})

const headerAliases = {
  name: ['变量名', '名称', 'name'],
  code: ['Code', '编码', 'code'],
  unitId: ['从站地址', '从站', 'unitId', 'slaveId'],
  area: ['区域', '寄存器区', 'area'],
  address: ['地址', '用户地址', 'address'],
  addressBase: ['地址基准', 'addressBase'],
  dataType: ['数据类型', '类型', 'dataType', 'type'],
  byteOrder: ['字节序', 'byteOrder'],
  wordOrder: ['字序', 'wordOrder'],
  scale: ['倍率', 'scale'],
  offset: ['偏移', 'offset'],
  unit: ['单位', 'unit'],
  pollIntervalMs: ['采集周期', '周期', 'pollIntervalMs'],
  accessLevel: ['发布能力', '读写能力', 'accessLevel'],
  description: ['描述', '说明', 'description'],
}
const templateHeaders = [
  '变量名',
  'Code',
  '从站地址',
  '区域',
  '地址',
  '地址基准',
  '数据类型',
  '字节序',
  '字序',
  '倍率',
  '偏移',
  '单位',
  '采集周期',
  '发布能力',
  '描述',
]
const templateRows = [
  {
    变量名: '电机转速',
    Code: 'motor_speed',
    从站地址: 1,
    区域: 'holding_register',
    地址: 40001,
    地址基准: 'modicon',
    数据类型: 'float32',
    字节序: 'ABCD',
    字序: 'high_first',
    倍率: 1,
    偏移: 0,
    单位: 'rpm',
    采集周期: 1000,
    发布能力: 'read',
    描述: '主电机转速',
  },
  {
    变量名: '运行状态',
    Code: 'running_state',
    从站地址: 1,
    区域: 'coil',
    地址: 1,
    地址基准: 'modicon',
    数据类型: 'bool',
    字节序: 'ABCD',
    字序: 'high_first',
    倍率: 1,
    偏移: 0,
    单位: '',
    采集周期: 500,
    发布能力: 'readwrite',
    描述: '线圈状态',
  },
]

watch(
  () => props.modelValue,
  (visible) => {
    if (!visible) return
    pasteText.value = ''
    fileRows.value = []
    fileName.value = ''
    onlyIssues.value = false
    selectedDefaultGroupId.value = props.defaultGroupId || ''
    const defaultSlave =
      slaves.value.find((slave) => slave.unitId === props.defaultUnitId) || slaves.value[0]
    selectedDefaultUnitId.value = defaultSlave?.unitId ?? 1
    defaultByteOrder.value = defaultSlave?.defaultByteOrder || 'ABCD'
    defaultPollIntervalMs.value = defaultSlave?.defaultPollIntervalMs || 1000
    range.unitId = selectedDefaultUnitId.value
  },
)

async function handleFileChange(uploadFile: UploadFile) {
  const raw = uploadFile.raw
  if (!raw) return
  try {
    fileRows.value = await readTabularFile(raw)
    fileName.value = raw.name
    pasteText.value = ''
    mode.value = 'paste'
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '读取导入文件失败')
  }
}

function clearFileSource() {
  if (fileRows.value.length === 0) return
  fileRows.value = []
  fileName.value = ''
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

function buildPreviewRow(row: Record<string, string>, index: number) {
  const normalized = normalizeHeaderRow(row, headerAliases)
  const area = normalizeArea(normalized.area)
  const addressBase = normalizeAddressBase(normalized.addressBase)
  const address = toInteger(normalized.address, Number.NaN)
  const protocolAddress = normalizeProtocolAddress(area, addressBase, address)
  const dataType =
    normalized.dataType || (area === 'coil' || area === 'discrete_input' ? 'bool' : 'uint16')
  const unitId = toInteger(normalized.unitId, selectedDefaultUnitId.value)
  const scale = toNumber(normalized.scale, 1)
  const offset = toNumber(normalized.offset, 0)
  const pollIntervalMs = toInteger(normalized.pollIntervalMs, defaultPollIntervalMs.value)
  const issue = firstIssue([
    !normalized.name ? '变量名为空' : '',
    Number.isNaN(unitId) || unitId < 0 || unitId > 247 ? '从站地址必须在 0-247' : '',
    Number.isNaN(address) ? '地址不是数字' : '',
    Number.isNaN(protocolAddress) || protocolAddress < 0 ? '地址与寄存器区域不匹配' : '',
    !dataType ? '数据类型为空' : '',
    ['coil', 'discrete_input'].includes(area) &&
    !['bool', 'boolean'].includes(dataType.toLowerCase())
      ? 'Coil / Discrete Input 默认只支持 bool'
      : '',
    Number.isNaN(scale) ? '倍率不是数字' : '',
    Number.isNaN(offset) ? '偏移不是数字' : '',
    Number.isNaN(pollIntervalMs) || pollIntervalMs <= 0 ? '采集周期必须大于 0' : '',
  ])
  return {
    rowNo: index + 1,
    name: normalized.name || `变量${index + 1}`,
    code: normalized.code,
    unitId,
    area,
    address,
    addressBase,
    protocolAddress,
    dataType,
    byteOrder: normalized.byteOrder || defaultByteOrder.value,
    wordOrder: normalized.wordOrder || 'high_first',
    scale,
    offset,
    unit: normalized.unit || null,
    pollIntervalMs,
    accessLevel: normalizeAccessLevel(normalized.accessLevel || defaultAccessLevel(area)),
    description: normalized.description || null,
    issue,
  }
}

function downloadTemplate(type: 'csv' | 'xlsx') {
  if (type === 'csv') {
    downloadCsv('modbus-variable-import-template.csv', templateHeaders, templateRows)
    return
  }
  downloadXlsx('modbus-variable-import-template.xlsx', [
    { name: 'Modbus变量模板', headers: templateHeaders, rows: templateRows },
  ])
}

function downloadIssueReport() {
  downloadCsv(
    'modbus-import-issues.csv',
    ['行号', '变量名', 'Code', '区域', '地址', '问题'],
    issueRows.value.map((row) => ({
      行号: row.rowNo,
      变量名: row.name,
      Code: row.code,
      区域: row.area,
      地址: row.address,
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

function normalizeArea(value: string) {
  const text = value.trim().toLowerCase()
  if (['coil', 'coils', '0x', '00001'].includes(text)) return 'coil'
  if (['discrete_input', 'discrete input', 'input', '1x', '10001'].includes(text))
    return 'discrete_input'
  if (['input_register', 'input register', '3x', '30001'].includes(text)) return 'input_register'
  return 'holding_register'
}

function normalizeAddressBase(value: string) {
  const text = value.trim().toLowerCase()
  if (['zero_based', '0', '0-based'].includes(text)) return 'zero_based'
  if (['one_based', '1', '1-based'].includes(text)) return 'one_based'
  return 'modicon'
}

function normalizeProtocolAddress(area: string, addressBase: string, address: number) {
  if (Number.isNaN(address)) return Number.NaN
  if (addressBase === 'zero_based') return address
  if (addressBase === 'one_based') return address - 1
  return address - modiconPrefix(area) - 1
}

function modiconPrefix(area: string) {
  if (area === 'coil') return 0
  if (area === 'discrete_input') return 10000
  if (area === 'input_register') return 30000
  return 40000
}

function defaultAccessLevel(area: string) {
  return ['coil', 'holding_register'].includes(area) ? 'read' : 'read'
}

function normalizeAccessLevel(value: string) {
  const text = value.trim().toLowerCase()
  if (['write', '写', 'readwrite', 'read_write', '读写'].includes(text)) return 'readwrite'
  return 'read'
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
.modbus-import-dialog__summary {
  margin-bottom: 10px;
  padding: 10px 12px;
  border: 1px solid color-mix(in oklch, var(--dc-primary) 18%, var(--dc-border));
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-subtle);
  display: flex;
  align-items: center;
  gap: 8px;
}

.modbus-import-dialog__summary strong {
  color: var(--dc-primary);
  font-size: 18px;
}

.modbus-import-dialog__summary span {
  color: var(--dc-text-secondary);
  font-size: 12px;
  font-weight: 700;
}

.modbus-import-dialog__summary em {
  margin-left: auto;
  font-style: normal;
  color: var(--dc-text-muted);
  font-size: 12px;
}

.modbus-import-dialog__tools {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 10px;
  min-width: 0;
}

.modbus-import-dialog__tools :deep(.el-button) {
  gap: 5px;
}

.modbus-import-dialog__tools svg {
  width: 14px;
  height: 14px;
}

.modbus-import-dialog__file {
  min-width: 0;
  overflow: hidden;
  color: var(--dc-text-muted);
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 12px;
}

.modbus-import-dialog__range {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 10px;
  margin-bottom: 12px;
}

.modbus-import-dialog__range :deep(.el-select),
.modbus-import-dialog__range :deep(.el-input-number) {
  width: 100%;
}
</style>
