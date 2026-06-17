<template>
  <DcDialog
    ref="dialogRef"
    v-model="visible"
    title="导入 Modbus 变量"
    width="min(1180px, calc(100vw - 64px))"
    body-max-height="calc(100vh - 132px)"
    class="modbus-import-dialog"
    :dirty="isDirty"
    :close-disabled="loading || confirmPreparing"
  >
    <div class="modbus-import">
      <div class="modbus-import__summary">
        <div>
          <strong>{{ candidateCount }}</strong>
          <span>候选变量</span>
        </div>
        <div>
          <strong>{{ step === 'confirm' ? validRows.length : '-' }}</strong>
          <span>可导入</span>
        </div>
        <div>
          <strong>{{ step === 'confirm' ? issueRows.length : '-' }}</strong>
          <span>需检查</span>
        </div>
        <em>{{ step === 'pick' ? '准备数据' : '导入确认' }}</em>
      </div>

      <div v-if="step === 'confirm'" class="modbus-import__tools">
        <el-checkbox v-model="onlyIssues" size="small">只看问题</el-checkbox>
        <el-button size="small" :disabled="issueRows.length === 0" @click="downloadIssueReport">
          <IconTablerAlertTriangle />
          错误报告
        </el-button>
      </div>

      <el-tabs v-if="step === 'pick'" v-model="mode" class="modbus-import__tabs">
        <el-tab-pane label="粘贴表格" name="paste">
          <section class="modbus-import__section">
            <div class="modbus-import__section-head">
              <strong>粘贴表格 / 文件导入</strong>
              <span>{{ pasteSourceSummary }}</span>
            </div>
            <div class="modbus-import__toolbar">
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
              <span v-if="fileName" class="modbus-import__file" :title="fileName">
                {{ fileName }}
              </span>
            </div>
            <el-form label-position="top" class="modbus-import__form">
              <el-form-item label="表格内容">
                <el-input
                  v-model="pasteText"
                  type="textarea"
                  :rows="12"
                  placeholder="变量名,从站地址,数据区,地址,地址基准,类型,字节序,字序,倍率,偏移,单位,采集周期,读写权限,描述"
                  @input="clearFileSource"
                />
              </el-form-item>
            </el-form>
          </section>
        </el-tab-pane>

        <el-tab-pane label="地址段生成" name="range">
          <section class="modbus-import__section">
            <div class="modbus-import__section-head">
              <strong>地址段生成</strong>
              <span>{{ range.count }} 个候选变量</span>
            </div>
            <el-form label-position="top" class="modbus-import__range-form">
              <el-form-item label="从站地址">
                <el-input-number v-model="range.unitId" :min="0" :max="247" />
              </el-form-item>
              <el-form-item label="Modbus 数据区">
                <el-select v-model="range.area">
                  <el-option
                    v-for="option in areaOptions"
                    :key="option.value"
                    :label="option.label"
                    :value="option.value"
                  />
                </el-select>
              </el-form-item>
              <el-form-item label="地址基准">
                <el-select v-model="range.addressBase">
                  <el-option label="Modicon 地址（40001/30001 等）" value="modicon" />
                  <el-option label="从 1 开始" value="one_based" />
                  <el-option label="从 0 开始" value="zero_based" />
                </el-select>
              </el-form-item>
              <el-form-item label="起始地址">
                <el-input-number v-model="range.startAddress" :min="0" />
              </el-form-item>
              <el-form-item label="变量数量">
                <el-input-number v-model="range.count" :min="1" :max="500" />
              </el-form-item>
              <el-form-item label="数据类型">
                <el-select v-model="range.dataType">
                  <el-option label="uint16" value="uint16" />
                  <el-option label="int16" value="int16" />
                  <el-option label="uint32" value="uint32" />
                  <el-option label="int32" value="int32" />
                  <el-option label="float32" value="float32" />
                  <el-option label="bool" value="bool" />
                </el-select>
              </el-form-item>
              <el-form-item label="变量名前缀">
                <el-input v-model="range.prefix" />
              </el-form-item>
              <el-form-item label="采集周期(ms)">
                <el-input-number v-model="range.pollIntervalMs" :min="100" :step="100" />
              </el-form-item>
            </el-form>
          </section>
        </el-tab-pane>
      </el-tabs>

      <section v-if="step === 'confirm'" class="modbus-import__defaults">
        <div class="modbus-import__section-head">
          <strong>批量默认值</strong>
          <span>应用后会重新检查候选行</span>
        </div>
        <el-form label-position="top" class="modbus-import__default-form">
          <el-form-item label="导入到分组">
            <el-select v-model="selectedDefaultGroupId" clearable filterable placeholder="未分组">
              <el-option label="未分组" value="" />
              <el-option
                v-for="group in groups"
                :key="group.id"
                :label="group.name"
                :value="group.id"
              />
            </el-select>
          </el-form-item>
          <el-form-item label="默认从站">
            <el-select v-model="selectedDefaultUnitId" filterable>
              <el-option
                v-for="slave in slaves"
                :key="slave.id"
                :label="`${slave.name} (${slave.unitId})`"
                :value="slave.unitId"
              />
            </el-select>
          </el-form-item>
          <el-form-item label="默认字节序">
            <el-select v-model="defaultByteOrder">
              <el-option label="ABCD" value="ABCD" />
              <el-option label="BADC" value="BADC" />
              <el-option label="CDAB" value="CDAB" />
              <el-option label="DCBA" value="DCBA" />
            </el-select>
          </el-form-item>
          <el-form-item label="默认采集周期(ms)">
            <el-input-number v-model="defaultPollIntervalMs" :min="100" :step="100" />
          </el-form-item>
          <el-form-item label=" ">
            <el-button @click="applyDefaults">应用到候选行</el-button>
          </el-form-item>
        </el-form>
      </section>

      <section v-if="step === 'confirm'" class="modbus-import__confirm-section">
        <div class="modbus-import__section-head">
          <strong>导入确认</strong>
          <span>导入前可逐行调整关键字段</span>
        </div>
        <el-table
          :data="pagedDisplayRows"
          height="320"
          row-key="rowNo"
          size="small"
          class="modbus-import__confirm-table"
        >
          <el-table-column label="状态" width="78" fixed>
            <template #default="{ row }">
              <el-tag size="small" :type="rowIssue(row) ? 'warning' : 'success'">
                {{ rowIssue(row) ? '检查' : '可导入' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="变量名" min-width="150">
            <template #default="{ row }">
              <el-input v-model="row.name" size="small" />
            </template>
          </el-table-column>
          <el-table-column prop="code" label="Code" min-width="130" show-overflow-tooltip />
          <el-table-column label="从站" width="92">
            <template #default="{ row }">
              <el-input v-model="row.unitId" size="small" />
            </template>
          </el-table-column>
          <el-table-column label="数据区" min-width="168">
            <template #default="{ row }">
              <el-select v-model="row.area" size="small">
                <el-option
                  v-for="option in areaOptions"
                  :key="option.value"
                  :label="option.label"
                  :value="option.value"
                />
              </el-select>
            </template>
          </el-table-column>
          <el-table-column label="地址" width="102">
            <template #default="{ row }">
              <el-input v-model="row.address" size="small" />
            </template>
          </el-table-column>
          <el-table-column label="地址基准" width="128">
            <template #default="{ row }">
              <el-select v-model="row.addressBase" size="small">
                <el-option label="Modicon" value="modicon" />
                <el-option label="从 1 开始" value="one_based" />
                <el-option label="从 0 开始" value="zero_based" />
              </el-select>
            </template>
          </el-table-column>
          <el-table-column label="协议地址" width="92">
            <template #default="{ row }">{{ rowProtocolAddress(row) }}</template>
          </el-table-column>
          <el-table-column label="类型" width="112">
            <template #default="{ row }">
              <el-select v-model="row.dataType" size="small">
                <el-option
                  v-for="option in dataTypeOptions"
                  :key="option.value"
                  :label="option.label"
                  :value="option.value"
                />
              </el-select>
            </template>
          </el-table-column>
          <el-table-column label="采集周期" width="110">
            <template #default="{ row }">
              <el-input v-model="row.pollIntervalMs" size="small" />
            </template>
          </el-table-column>
          <el-table-column label="问题" min-width="160" show-overflow-tooltip>
            <template #default="{ row }">{{ rowIssue(row) || '-' }}</template>
          </el-table-column>
        </el-table>
        <div class="modbus-import__pager">
          <el-pagination
            v-model:current-page="confirmPage"
            v-model:page-size="confirmPageSize"
            size="small"
            layout="total, sizes, prev, pager, next"
            :page-sizes="[50, 100, 200]"
            :total="displayRows.length"
          />
        </div>
      </section>
    </div>

    <template #footer>
      <el-button @click="requestClose">取消</el-button>
      <el-button v-if="step === 'confirm'" @click="step = 'pick'">上一步</el-button>
      <el-button
        v-if="step === 'pick'"
        type="primary"
        :disabled="candidateCount === 0"
        :loading="confirmPreparing"
        @click="goConfirm"
      >
        {{ confirmPreparing ? '准备确认...' : `下一步，确认 ${candidateCount} 个候选变量` }}
      </el-button>
      <el-button
        v-else
        type="primary"
        :disabled="validRows.length === 0"
        :loading="loading"
        @click="submit"
      >
        导入 {{ validRows.length }} 个变量
      </el-button>
    </template>
  </DcDialog>
</template>

<script setup lang="ts">
import { computed, nextTick, reactive, ref, watch } from 'vue'
import type { UploadFile } from 'element-plus'
import { ElMessage, ElMessageBox } from 'element-plus'
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

type ModbusImportMode = 'paste' | 'range'
type ModbusImportStep = 'pick' | 'confirm'
type ModbusImportSubmitPayload = {
  groupId: string | null
  registers: Array<Record<string, unknown>>
}
type ModbusImportPreviewRow = {
  rowNo: number
  name: string
  code: string
  unitId: number | string
  area: string
  address: number | string
  addressBase: string
  protocolAddress: number
  dataType: string
  byteOrder: string
  wordOrder: string
  scale: number | string
  offset: number | string
  unit: string | null
  pollIntervalMs: number | string
  accessLevel: string
  description: string | null
  issue: string
}

const props = defineProps<{
  modelValue: boolean
  loading?: boolean
  groups?: ModbusRegisterGroup[]
  slaves?: ModbusSlaveDevice[]
  existingCodes?: string[]
  defaultGroupId?: string
  defaultUnitId?: number
}>()
const emit = defineEmits<{
  (event: 'update:modelValue', value: boolean): void
  (event: 'submit', payload: ModbusImportSubmitPayload): void
}>()

const visible = computed({
  get: () => props.modelValue,
  set: (value: boolean) => emit('update:modelValue', value),
})
const dialogRef = ref<InstanceType<typeof DcDialog> | null>(null)
const mode = ref<ModbusImportMode>('paste')
const step = ref<ModbusImportStep>('pick')
const pasteText = ref('')
const fileRows = ref<Record<string, string>[]>([])
const fileName = ref('')
const onlyIssues = ref(false)
const confirmPage = ref(1)
const confirmPageSize = ref(50)
const confirmPreparing = ref(false)
const confirmRows = ref<ModbusImportPreviewRow[]>([])
const selectedDefaultGroupId = ref('')
const selectedDefaultUnitId = ref(1)
const defaultByteOrder = ref('ABCD')
const defaultPollIntervalMs = ref(1000)
const invalidVariableNamePattern = /[^\u4e00-\u9fa5A-Za-z0-9_$#%@+()[\]&-]/g
const validVariableNamePattern = /^[\u4e00-\u9fa5A-Za-z0-9_$#%@+()[\]&-]+$/
const areaOptions = [
  { label: '读写数值寄存器（保持寄存器 4x）', value: 'holding_register' },
  { label: '只读数值寄存器（输入寄存器 3x）', value: 'input_register' },
  { label: '开关量输出（线圈 0x）', value: 'coil' },
  { label: '开关量输入（离散输入 1x）', value: 'discrete_input' },
]
const dataTypeOptions = [
  { label: 'bool', value: 'bool' },
  { label: 'uint16', value: 'uint16' },
  { label: 'int16', value: 'int16' },
  { label: 'uint32', value: 'uint32' },
  { label: 'int32', value: 'int32' },
  { label: 'float32', value: 'float32' },
  { label: 'float64', value: 'float64' },
]
const range = reactive({
  unitId: 1,
  area: 'holding_register',
  addressBase: 'modicon',
  startAddress: 40001,
  count: 20,
  dataType: 'uint16',
  prefix: 'modbus_reg_',
  pollIntervalMs: 1000,
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
      confirmRows.value.length > 0 ||
      (mode.value === 'range' && JSON.stringify({ ...range }) !== initialRangeSnapshot)),
)
const sourceRows = computed(() => {
  if (fileRows.value.length > 0) return fileRows.value
  return parsePasteRows(pasteText.value)
})
const candidateCount = computed(() =>
  mode.value === 'range' ? Number(range.count || 0) : sourceRows.value.length,
)
const pasteSourceSummary = computed(() =>
  fileRows.value.length > 0 ? `${fileRows.value.length} 行文件数据` : `${sourceRows.value.length} 行粘贴数据`,
)
const validatedRows = computed(() => validateRows(confirmRows.value))
const validatedRowMap = computed(
  () => new Map(validatedRows.value.map((row) => [row.rowNo, row])),
)
const displayRows = computed(() =>
  onlyIssues.value ? confirmRows.value.filter((row) => rowIssue(row)) : confirmRows.value,
)
const pagedDisplayRows = computed(() => {
  const start = (confirmPage.value - 1) * confirmPageSize.value
  return displayRows.value.slice(start, start + confirmPageSize.value)
})
const issueRows = computed(() => validatedRows.value.filter((row) => row.issue))
const validRows = computed(() =>
  validatedRows.value
    .filter((row) => !row.issue)
    .map((row) => ({
      name: row.name,
      code: row.code,
      unitId: Number(row.unitId),
      area: row.area,
      address: Number(row.address),
      addressBase: row.addressBase,
      dataType: normalizeDataType(row.dataType),
      byteOrder: row.byteOrder,
      wordOrder: row.wordOrder,
      scale: Number(row.scale),
      offset: Number(row.offset),
      unit: row.unit || null,
      pollIntervalMs: Number(row.pollIntervalMs),
      accessLevel: normalizeAccessLevel(row.accessLevel),
      description: row.description || null,
    })),
)

const headerAliases = {
  name: ['变量名', '名称', 'name'],
  unitId: ['从站地址', '从站', 'unitId', 'slaveId'],
  area: ['数据区', '区域', '寄存器区', 'area'],
  address: ['地址', '用户地址', 'address'],
  addressBase: ['地址基准', 'addressBase'],
  dataType: ['数据类型', '类型', 'dataType', 'type'],
  byteOrder: ['字节序', 'byteOrder'],
  wordOrder: ['字序', 'wordOrder'],
  scale: ['倍率', 'scale'],
  offset: ['偏移', 'offset'],
  unit: ['单位', 'unit'],
  pollIntervalMs: ['采集周期', '周期', 'pollIntervalMs'],
  accessLevel: ['发布能力', '读写能力', '读写权限', 'accessLevel'],
  description: ['描述', '说明', 'description'],
}
const templateHeaders = [
  '变量名',
  '从站地址',
  '数据区',
  '地址',
  '地址基准',
  '数据类型',
  '字节序',
  '字序',
  '倍率',
  '偏移',
  '单位',
  '采集周期',
  '读写权限',
  '描述',
]
const templateRows = [
  {
    变量名: '电机转速',
    从站地址: 1,
    数据区: '读写数值寄存器（保持寄存器 4x）',
    地址: 40001,
    地址基准: 'modicon',
    数据类型: 'float32',
    字节序: 'ABCD',
    字序: 'high_first',
    倍率: 1,
    偏移: 0,
    单位: 'rpm',
    采集周期: 1000,
    读写权限: 'read',
    描述: '主电机转速',
  },
  {
    变量名: '运行状态',
    从站地址: 1,
    数据区: '开关量输出（线圈 0x）',
    地址: 1,
    地址基准: 'modicon',
    数据类型: 'bool',
    字节序: 'ABCD',
    字序: 'high_first',
    倍率: 1,
    偏移: 0,
    单位: '',
    采集周期: 500,
    读写权限: 'readwrite',
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
    step.value = 'pick'
    onlyIssues.value = false
    confirmPage.value = 1
    confirmPageSize.value = 50
    confirmPreparing.value = false
    confirmRows.value = []
    selectedDefaultGroupId.value = props.defaultGroupId || ''
    const defaultSlave =
      slaves.value.find((slave) => slave.unitId === props.defaultUnitId) || slaves.value[0]
    selectedDefaultUnitId.value = defaultSlave?.unitId ?? 1
    defaultByteOrder.value = defaultSlave?.defaultByteOrder || 'ABCD'
    defaultPollIntervalMs.value = defaultSlave?.defaultPollIntervalMs || 1000
    range.unitId = selectedDefaultUnitId.value
    range.pollIntervalMs = defaultPollIntervalMs.value
  },
)

watch([onlyIssues, confirmPageSize], () => {
  confirmPage.value = 1
})

watch([mode, pasteText, fileRows, range], () => {
  if (step.value !== 'pick') return
  confirmRows.value = []
})

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

function buildRangeRows() {
  return Array.from({ length: Number(range.count || 0) }, (_, index) => {
    const address = Number(range.startAddress) + index
    const name = `${range.prefix}${index + 1}`
    return createPreviewRow({
      rowNo: index + 1,
      name,
      code: toVariableCode(name, `range_${address}`),
      unitId: range.unitId,
      area: range.area,
      address,
      addressBase: range.addressBase,
      protocolAddress: normalizeProtocolAddress(range.area, range.addressBase, address),
      dataType: range.dataType,
      byteOrder: defaultByteOrder.value,
      wordOrder: 'high_first',
      scale: 1,
      offset: 0,
      unit: null,
      pollIntervalMs: range.pollIntervalMs,
      accessLevel: defaultAccessLevel(range.area),
      description: null,
    })
  })
}

function buildPreviewRow(row: Record<string, string>, index: number) {
  const normalized = normalizeHeaderRow(row, headerAliases)
  const area = normalizeArea(normalized.area)
  const addressBase = normalizeAddressBase(normalized.addressBase)
  const address = toInteger(normalized.address, Number.NaN)
  const protocolAddress = normalizeProtocolAddress(area, addressBase, address)
  const dataType =
    normalized.dataType || (area === 'coil' || area === 'discrete_input' ? 'bool' : 'uint16')
  const name = sanitizeVariableName(normalized.name || `变量${index + 1}`)
  return createPreviewRow({
    rowNo: index + 1,
    name,
    code: toVariableCode(name, `${area}_${address}_${index + 1}`),
    unitId: toInteger(normalized.unitId, selectedDefaultUnitId.value),
    area,
    address,
    addressBase,
    protocolAddress,
    dataType,
    byteOrder: normalized.byteOrder || defaultByteOrder.value,
    wordOrder: normalized.wordOrder || 'high_first',
    scale: toNumber(normalized.scale, 1),
    offset: toNumber(normalized.offset, 0),
    unit: normalized.unit || null,
    pollIntervalMs: toInteger(normalized.pollIntervalMs, defaultPollIntervalMs.value),
    accessLevel: normalizeAccessLevel(normalized.accessLevel || defaultAccessLevel(area)),
    description: normalized.description || null,
  })
}

function createPreviewRow(row: Omit<ModbusImportPreviewRow, 'issue'>): ModbusImportPreviewRow {
  return { ...row, issue: '' }
}

function validateRows(rows: ModbusImportPreviewRow[]) {
  const usedCodes = new Set((props.existingCodes || []).filter(Boolean))
  return rows.map((row) => {
    const area = normalizeArea(row.area)
    const addressBase = normalizeAddressBase(row.addressBase)
    const unitId = toInteger(String(row.unitId), Number.NaN)
    const address = toInteger(String(row.address), Number.NaN)
    const protocolAddress = normalizeProtocolAddress(area, addressBase, address)
    const scale = toNumber(String(row.scale), Number.NaN)
    const offset = toNumber(String(row.offset), Number.NaN)
    const pollIntervalMs = toInteger(String(row.pollIntervalMs), Number.NaN)
    const dataType = normalizeDataType(row.dataType)
    const name = sanitizeVariableName(row.name)
    const issue = firstIssue([
      !name ? '变量名为空' : '',
      !isValidVariableName(name) ? '变量名称包含不支持的字符' : '',
      Number.isNaN(unitId) || unitId < 0 || unitId > 247 ? '从站地址必须在 0-247' : '',
      Number.isNaN(address) ? '地址不是数字' : '',
      Number.isNaN(protocolAddress) || protocolAddress < 0 ? '地址与 Modbus 数据区不匹配' : '',
      !dataType ? '数据类型为空' : '',
      !isSupportedDataType(dataType) ? '数据类型不支持' : '',
      ['coil', 'discrete_input'].includes(area) &&
      !['bool', 'boolean'].includes(dataType.toLowerCase())
        ? '开关量数据区默认只支持 bool'
        : '',
      Number.isNaN(scale) ? '倍率不是数字' : '',
      Number.isNaN(offset) ? '偏移不是数字' : '',
      Number.isNaN(pollIntervalMs) || pollIntervalMs <= 0 ? '采集周期必须大于 0' : '',
    ])
    const code = issue
      ? row.code
      : allocateUniqueCode(row.code || toVariableCode(row.name, 'register'), usedCodes)
    if (!issue) usedCodes.add(code)
    return {
      ...row,
      name,
      code,
      area,
      addressBase,
      unitId: Number.isNaN(unitId) ? row.unitId : unitId,
      address: Number.isNaN(address) ? row.address : address,
      protocolAddress,
      dataType,
      scale: Number.isNaN(scale) ? row.scale : scale,
      offset: Number.isNaN(offset) ? row.offset : offset,
      pollIntervalMs: Number.isNaN(pollIntervalMs) ? row.pollIntervalMs : pollIntervalMs,
      accessLevel: normalizeAccessLevel(row.accessLevel),
      issue,
    }
  })
}

function currentGroupId() {
  return selectedDefaultGroupId.value || null
}

function validatedRow(row: ModbusImportPreviewRow) {
  return validatedRowMap.value.get(row.rowNo) || row
}

function rowIssue(row: ModbusImportPreviewRow) {
  return validatedRow(row).issue
}

function rowProtocolAddress(row: ModbusImportPreviewRow) {
  const value = validatedRow(row).protocolAddress
  return Number.isNaN(value) ? '-' : value
}

async function goConfirm() {
  if (candidateCount.value === 0) {
    ElMessage.warning(mode.value === 'range' ? '请先设置地址段数量' : '请先粘贴或选择导入文件')
    return
  }
  confirmPreparing.value = true
  await nextTick()
  await waitForPaint()
  try {
    confirmRows.value =
      mode.value === 'range'
        ? buildRangeRows()
        : sourceRows.value.map((row, index) => buildPreviewRow(row, index))
    confirmPage.value = 1
    onlyIssues.value = false
    step.value = 'confirm'
  } finally {
    confirmPreparing.value = false
  }
}

function applyDefaults() {
  confirmRows.value.forEach((row) => {
    row.unitId = selectedDefaultUnitId.value
    row.byteOrder = defaultByteOrder.value
    row.pollIntervalMs = defaultPollIntervalMs.value
  })
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
    ['行号', '变量名', '数据区', '地址', '问题'],
    issueRows.value.map((row) => ({
      行号: row.rowNo,
      变量名: row.name,
      数据区: formatAreaLabel(row.area),
      地址: row.address,
      问题: row.issue,
    })),
  )
}

function formatAreaLabel(area: string) {
  const map: Record<string, string> = {
    coil: '开关量输出（线圈 0x）',
    discrete_input: '开关量输入（离散输入 1x）',
    input_register: '只读数值寄存器（输入寄存器 3x）',
    holding_register: '读写数值寄存器（保持寄存器 4x）',
  }
  return map[area] || area
}

function submit() {
  if (validRows.value.length === 0) {
    ElMessage.warning('没有可导入的 Modbus 变量')
    return
  }
  const skipped = confirmRows.value.length - validRows.value.length
  const runSubmit = () => emit('submit', { groupId: currentGroupId(), registers: validRows.value })
  if (skipped > 0) {
    void ElMessageBox.confirm(
      `本次将导入 ${validRows.value.length} 个变量，跳过 ${skipped} 行问题数据。是否继续？`,
      '确认导入',
    ).then(runSubmit)
    return
  }
  runSubmit()
}

function requestClose() {
  void dialogRef.value?.requestClose()
}

function closeSilently() {
  dialogRef.value?.closeSilently()
}

function waitForPaint() {
  return new Promise<void>((resolve) => {
    window.requestAnimationFrame(() => window.requestAnimationFrame(() => resolve()))
  })
}

function normalizeArea(value: string) {
  const text = String(value || '').trim().toLowerCase()
  if (
    [
      'coil',
      'coils',
      '0x',
      '00001',
      '线圈',
      '开关量输出',
      '输出开关',
      '开关量输出（线圈 0x）',
    ].includes(text)
  )
    return 'coil'
  if (
    [
      'discrete_input',
      'discrete input',
      'input',
      '1x',
      '10001',
      '离散输入',
      '离散量输入',
      '开关量输入',
      '输入开关',
      '开关量输入（离散输入 1x）',
    ].includes(text)
  )
    return 'discrete_input'
  if (
    [
      'input_register',
      'input register',
      '3x',
      '30001',
      '输入寄存器',
      '只读寄存器',
      '只读数值寄存器',
      '只读数值寄存器（输入寄存器 3x）',
    ].includes(text)
  )
    return 'input_register'
  return 'holding_register'
}

function normalizeAddressBase(value: string) {
  const text = String(value || '').trim().toLowerCase()
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
  const text = String(value || '').trim().toLowerCase()
  if (['write', '写', 'readwrite', 'read_write', '读写'].includes(text)) return 'readwrite'
  return 'read'
}

function normalizeDataType(value: string) {
  return String(value || '').trim().toLowerCase()
}

function isSupportedDataType(value: string) {
  const normalized = normalizeDataType(value)
  return dataTypeOptions.some((option) => option.value === normalized)
}

function sanitizeVariableName(value: string) {
  return String(value || '').trim().replace(invalidVariableNamePattern, '')
}

function isValidVariableName(value: string) {
  return validVariableNamePattern.test(value)
}

function toVariableCode(value: string, fallback: string) {
  const normalized = String(value || fallback)
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9_]+/g, '_')
    .replace(/^_+|_+$/g, '')
  return normalized || 'register'
}

function allocateUniqueCode(baseCode: string, usedCodes: Set<string>) {
  const base = toVariableCode(baseCode, 'register')
  if (!usedCodes.has(base)) return base
  for (let index = 2; index < 10000; index += 1) {
    const candidate = `${base}_${index}`
    if (!usedCodes.has(candidate)) return candidate
  }
  return `${base}_${Date.now()}`
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
.modbus-import {
  min-height: 0;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.modbus-import__summary {
  min-height: 54px;
  padding: 10px 12px;
  border: 1px solid color-mix(in oklch, var(--dc-primary) 18%, var(--dc-border));
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-subtle);
  display: grid;
  grid-template-columns: repeat(3, minmax(92px, 120px)) 1fr;
  align-items: center;
  gap: 10px;
}

.modbus-import__summary div {
  display: grid;
  gap: 2px;
}

.modbus-import__summary strong {
  color: var(--dc-primary);
  font-size: 18px;
  line-height: 20px;
}

.modbus-import__summary span {
  color: var(--dc-text-secondary);
  font-size: 12px;
  font-weight: 700;
}

.modbus-import__summary em {
  justify-self: end;
  font-style: normal;
  color: var(--dc-text-muted);
  font-size: 12px;
}

.modbus-import__tools,
.modbus-import__toolbar {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.modbus-import__tools :deep(.el-button),
.modbus-import__toolbar :deep(.el-button) {
  gap: 5px;
}

.modbus-import__tools svg,
.modbus-import__toolbar svg {
  width: 14px;
  height: 14px;
}

.modbus-import__tabs {
  min-height: 0;
}

.modbus-import__section,
.modbus-import__defaults,
.modbus-import__confirm-section {
  min-height: 0;
  padding: 12px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface);
}

.modbus-import__section-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 10px;
}

.modbus-import__section-head strong {
  color: var(--dc-text);
  font-size: 13px;
}

.modbus-import__section-head span {
  min-width: 0;
  overflow: hidden;
  color: var(--dc-text-muted);
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.modbus-import__file {
  min-width: 0;
  overflow: hidden;
  color: var(--dc-text-muted);
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 12px;
}

.modbus-import__form {
  margin-top: 10px;
}

.modbus-import__range-form,
.modbus-import__default-form {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 10px 12px;
}

.modbus-import__default-form {
  grid-template-columns: minmax(160px, 1.2fr) repeat(3, minmax(130px, 1fr)) auto;
}

.modbus-import__range-form :deep(.el-select),
.modbus-import__range-form :deep(.el-input),
.modbus-import__range-form :deep(.el-input-number),
.modbus-import__default-form :deep(.el-select),
.modbus-import__default-form :deep(.el-input-number) {
  width: 100%;
}

.modbus-import__confirm-table {
  width: 100%;
}

.modbus-import__confirm-table :deep(.el-table__inner-wrapper) {
  min-width: 1080px;
}

.modbus-import__confirm-table :deep(.cell) {
  padding-left: 7px;
  padding-right: 7px;
  white-space: nowrap;
}

.modbus-import__confirm-table :deep(.el-input__wrapper),
.modbus-import__confirm-table :deep(.el-select__wrapper) {
  min-height: 30px;
  box-shadow: 0 0 0 1px var(--dc-border) inset;
}

.modbus-import__confirm-table :deep(.el-table__body-wrapper) {
  overflow: auto;
}

.modbus-import__pager {
  display: flex;
  justify-content: flex-end;
  padding-top: 8px;
  margin-top: 8px;
  border-top: 1px solid var(--dc-border);
}

.modbus-import :deep(.el-form-item) {
  margin-bottom: 0;
}

.modbus-import :deep(.el-form-item__label) {
  color: var(--dc-text-secondary);
  font-size: 12px;
  font-weight: 700;
}

.modbus-import :deep(.el-table) {
  --el-table-header-bg-color: var(--dc-surface-raised);
  --el-table-border-color: var(--dc-border);
}
</style>

<style>
.modbus-import-dialog .el-dialog__body,
.modbus-import-dialog .dc-dialog__body {
  overflow: hidden;
}
</style>
