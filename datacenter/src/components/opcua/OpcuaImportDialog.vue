<template>
  <DcDialog
    ref="dialogRef"
    v-model="visible"
    title="从 OPC UA 导入变量"
    width="min(1280px, calc(100vw - 64px))"
    class="opcua-import-dialog"
    body-max-height="calc(100vh - 132px)"
    :dirty="isDirty"
    :close-disabled="loading"
  >
    <div class="opcua-import">
      <div class="opcua-import__summary">
        <div>
          <strong>{{ candidateCount }}</strong>
          <span>候选变量</span>
        </div>
        <div>
          <strong>{{ step === 'confirm' ? readyRows.length : '-' }}</strong>
          <span>可导入</span>
        </div>
        <div>
          <strong>{{ step === 'confirm' ? skippedRows.length : '-' }}</strong>
          <span>将跳过</span>
        </div>
        <div>
          <strong>{{ step === 'confirm' ? issueRows.length : '-' }}</strong>
          <span>需检查</span>
        </div>
        <em>{{ step === 'pick' ? '选择变量' : '导入确认' }}</em>
      </div>

      <div v-if="step === 'confirm'" class="opcua-import__tools">
        <el-checkbox v-model="onlyIssues" size="small">只看问题</el-checkbox>
        <el-button size="small" :disabled="issueRows.length === 0" @click="downloadIssueReport">
          <IconTablerAlertTriangle />
          错误报告
        </el-button>
      </div>

      <el-tabs v-if="step === 'pick'" v-model="mode">
        <el-tab-pane label="浏览导入" name="browse">
          <section class="opcua-import__browse-section">
            <div class="opcua-import__section-head">
              <strong>服务器节点浏览</strong>
              <span>{{ browseSummary }}</span>
            </div>
            <div class="opcua-import__toolbar">
              <el-input
                v-model="browseKeyword"
                size="small"
                clearable
                placeholder="搜索已加载节点的 NodeId / 节点名称 / 类型"
              />
              <el-button
                size="small"
                :loading="browseLoading"
                :disabled="!connected"
                @click="refreshBrowseTree"
              >
                <IconTablerRefresh />
                刷新浏览
              </el-button>
            </div>
            <div v-if="!connected" class="opcua-import__hint is-warning">
              请先连接 OPC UA 开发态会话，再刷新浏览结果。
            </div>
            <div v-else-if="browseDiagnostics.length > 0" class="opcua-import__hint">
              {{ browseDiagnostics[0] }}
            </div>
            <div class="opcua-import__browse-tree">
              <el-tree
                :key="treeVersion"
                ref="treeRef"
                lazy
                show-checkbox
                check-strictly
                node-key="id"
                :load="loadBrowseTreeNode"
                :props="browseTreeProps"
                :filter-node-method="filterBrowseTreeNode"
                @check="handleBrowseCheck"
              >
                <template #default="{ data }">
                  <div class="opcua-import__tree-node">
                    <span class="opcua-import__tree-title">
                      <IconTablerFolder v-if="data.nodeType === 'folder'" />
                      <IconTablerVariable v-else />
                      <strong>{{ data.name }}</strong>
                      <el-tag size="small" :type="browseStateTagType(data)">
                        {{ browseStateText(data) }}
                      </el-tag>
                      <el-button
                        v-if="data.nodeType === 'folder'"
                        size="small"
                        link
                        :loading="subtreeLoadingNodeId === data.nodeId"
                        @click.stop="importBrowseSubtree(data)"
                      >
                        导入子树变量
                      </el-button>
                    </span>
                    <span class="opcua-import__tree-meta">
                      <span>{{ data.nodeId }}</span>
                      <em v-if="data.dataType">{{ data.dataType }}</em>
                    </span>
                  </div>
                </template>
              </el-tree>
            </div>
          </section>
        </el-tab-pane>
        <el-tab-pane label="批量 NodeId" name="manual">
          <section class="opcua-import__manual-section">
            <div class="opcua-import__section-head">
              <strong>批量 NodeId</strong>
              <span>支持 NodeId、CSV、TSV 和 Excel 粘贴</span>
            </div>
            <el-input
              v-model="rawText"
              type="textarea"
              :rows="9"
              placeholder="每行一个 NodeId；或：ns=2;s=Line1.Motor01.Speed,Double,电机转速；也可粘贴含表头的表格"
            />
          </section>
        </el-tab-pane>
      </el-tabs>

      <section v-if="step === 'confirm'" class="opcua-import__defaults">
        <div class="opcua-import__section-head">
          <strong>批量默认值</strong>
          <span>应用后会重新检查候选行</span>
        </div>
        <div class="opcua-import__default-grid">
          <el-input v-model="defaults.samplingMs" size="small" placeholder="采样周期 ms" />
          <el-select v-model="defaults.accessLevel" size="small">
            <el-option label="Read" value="Read" />
            <el-option label="Write" value="Write" />
            <el-option label="ReadWrite" value="ReadWrite" />
          </el-select>
          <el-input v-model="defaults.deadband" size="small" placeholder="死区" />
          <el-input v-model="defaults.unit" size="small" clearable placeholder="单位" />
          <el-button size="small" @click="applyDefaults">应用到候选行</el-button>
        </div>
      </section>

      <section v-if="step === 'confirm'" class="opcua-import__confirm-section">
        <div class="opcua-import__section-head">
          <strong>导入确认</strong>
          <span>导入前可逐行调整关键字段</span>
        </div>
        <el-table :data="pagedDisplayRows" height="300" size="small" row-key="key" class="opcua-import__confirm-table">
          <el-table-column label="状态" width="74" fixed>
            <template #default="{ row }">
              <el-tag size="small" :type="rowTagType(row.state)">
                {{ rowStateText(row.state) }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="变量名" min-width="160">
            <template #default="{ row }"><el-input v-model="row.name" size="small" /></template>
          </el-table-column>
          <el-table-column prop="code" label="Code" min-width="140" show-overflow-tooltip />
          <el-table-column prop="nodeId" label="NodeId" min-width="180" show-overflow-tooltip />
          <el-table-column label="类型" width="124">
            <template #default="{ row }"><el-input v-model="row.dataType" size="small" /></template>
          </el-table-column>
          <el-table-column label="采样" width="104">
            <template #default="{ row }">
              <el-input v-model="row.samplingMs" size="small" />
            </template>
          </el-table-column>
          <el-table-column label="权限" width="112">
            <template #default="{ row }">
              <el-select v-model="row.accessLevel" size="small">
                <el-option label="Read" value="Read" />
                <el-option label="Write" value="Write" />
                <el-option label="ReadWrite" value="ReadWrite" />
              </el-select>
            </template>
          </el-table-column>
          <el-table-column label="死区" width="92">
            <template #default="{ row }">
              <el-input v-model="row.deadband" size="small" />
            </template>
          </el-table-column>
          <el-table-column label="单位" width="88">
            <template #default="{ row }"><el-input v-model="row.unit" size="small" /></template>
          </el-table-column>
          <el-table-column prop="issue" label="问题" min-width="160" show-overflow-tooltip />
        </el-table>
        <div class="opcua-import__pager">
          <el-pagination
            v-model:current-page="confirmPage"
            v-model:page-size="confirmPageSize"
            small
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
        :disabled="readyRows.length === 0"
        :loading="loading"
        @click="submit"
      >
        导入 {{ readyRows.length }} 个变量
      </el-button>
    </template>
  </DcDialog>
</template>

<script setup lang="ts">
import { computed, nextTick, reactive, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { TreeInstance } from 'element-plus'
import DcDialog from '@/components/shared/DcDialog.vue'
import { downloadCsv, normalizeHeaderRow, parseDelimitedRows, parseTabularText } from '@/utils/tabular-file'
import type { OpcuaBrowseNode } from './types'
import IconTablerAlertTriangle from '~icons/tabler/alert-triangle'
import IconTablerFolder from '~icons/tabler/folder'
import IconTablerRefresh from '~icons/tabler/refresh'
import IconTablerVariable from '~icons/tabler/variable'

type OpcuaImportState = 'ready' | 'existing' | 'duplicate' | 'error'
type OpcuaImportSource = 'browse' | 'manual'

type OpcuaBrowseRow = OpcuaBrowseNode & {
  modeled: boolean
}

type OpcuaImportPreviewRow = {
  key: string
  source: OpcuaImportSource
  rowNo: number
  name: string
  code: string
  nodeId: string
  browseName?: string | null
  displayName?: string | null
  dataType: string
  samplingMs: string
  accessLevel: string
  deadband: string
  unit: string | null
  description: string | null
  state: OpcuaImportState
  issue: string
}

const props = defineProps<{
  modelValue: boolean
  loading?: boolean
  connected?: boolean
  browseLoading?: boolean
  browseNodes?: OpcuaBrowseNode[]
  browseDiagnostics?: string[]
}>()

const emit = defineEmits<{
  (event: 'update:modelValue', value: boolean): void
  (event: 'submit', rows: Array<Record<string, unknown>>): void
  (event: 'browse', nodeId?: string): void
  (event: 'browse-subtree', nodeId: string, done: (nodes: OpcuaBrowseNode[]) => void): void
}>()

const visible = computed({
  get: () => props.modelValue,
  set: (value: boolean) => emit('update:modelValue', value),
})

const dialogRef = ref<InstanceType<typeof DcDialog> | null>(null)
const rawText = ref('')
const mode = ref<'browse' | 'manual'>('browse')
const step = ref<'pick' | 'confirm'>('pick')
const browseKeyword = ref('')
const selectedBrowseIds = ref<string[]>([])
const manualDraftRows = ref<OpcuaImportPreviewRow[]>([])
const browseDraftRows = ref<OpcuaImportPreviewRow[]>([])
const treeRef = ref<TreeInstance | null>(null)
const treeVersion = ref(0)
const pendingBrowseResolvers = new Map<string, (nodes: OpcuaBrowseRow[]) => void>()
const subtreeLoadingNodeId = ref('')
const onlyIssues = ref(false)
const confirmPage = ref(1)
const confirmPageSize = ref(50)
const confirmPreparing = ref(false)
const defaults = reactive({
  samplingMs: '1000',
  accessLevel: 'Read',
  deadband: '',
  unit: '',
})

const isDirty = computed(
  () =>
    props.modelValue &&
    (rawText.value.trim().length > 0 ||
      selectedBrowseIds.value.length > 0 ||
      manualDraftRows.value.length > 0 ||
      browseDraftRows.value.length > 0),
)

watch(
  () => props.modelValue,
  (visible) => {
    if (!visible) return
    rawText.value = ''
    step.value = 'pick'
    browseKeyword.value = ''
    selectedBrowseIds.value = []
    manualDraftRows.value = []
    browseDraftRows.value = []
    subtreeLoadingNodeId.value = ''
    onlyIssues.value = false
    confirmPage.value = 1
    confirmPageSize.value = 50
    confirmPreparing.value = false
    defaults.samplingMs = '1000'
    defaults.accessLevel = 'Read'
    defaults.deadband = ''
    defaults.unit = ''
    mode.value = props.connected ? 'browse' : 'manual'
  },
)

watch(rawText, () => {
  manualDraftRows.value = buildManualRows(rawText.value)
})

watch(browseKeyword, (keyword) => {
  treeRef.value?.filter(keyword)
})

watch(
  () => props.browseNodes,
  () => {
    resolvePendingBrowseNodes()
  },
  { deep: true },
)

watch([onlyIssues, confirmPageSize], () => {
  confirmPage.value = 1
})

const browseDiagnostics = computed(() => props.browseDiagnostics || [])
const existingNodeIds = computed(
  () =>
    new Set(
      (props.browseNodes || [])
        .filter((node) => node.nodeType === 'variable' && node.modeled === true)
        .map((node) => node.nodeId.trim())
        .filter(Boolean),
    ),
)
const browseRows = computed<OpcuaBrowseRow[]>(() =>
  (props.browseNodes || []).map((node) => ({ ...node, modeled: node.modeled === true })),
)
const browseRowsById = computed(() => new Map(browseRows.value.map((node) => [node.id, node])))
const browseSummary = computed(() => {
  const variables = browseRows.value.filter((row) => row.nodeType === 'variable')
  const importable = variables.filter((row) => !row.modeled).length
  return `${browseRows.value.length} 个已加载节点 · ${variables.length} 个变量 · ${importable} 个可导入`
})
const browseTreeProps = {
  label: 'name',
  children: 'children',
  isLeaf: (data: OpcuaBrowseRow) => data.nodeType === 'variable' || data.hasChildren === false,
  disabled: (data: OpcuaBrowseRow) => data.nodeType === 'variable' && !isBrowseRowSelectable(data),
}

const candidateCount = computed(() =>
  mode.value === 'browse' ? selectedBrowseIds.value.length : manualDraftRows.value.length,
)
const draftRows = computed(() => (mode.value === 'browse' ? browseDraftRows.value : manualDraftRows.value))
const previewRows = computed(() => validateRows(draftRows.value))
const displayRows = computed(() =>
  onlyIssues.value ? previewRows.value.filter((row) => row.state !== 'ready') : previewRows.value,
)
const pagedDisplayRows = computed(() => {
  const start = (confirmPage.value - 1) * confirmPageSize.value
  return displayRows.value.slice(start, start + confirmPageSize.value)
})
const readyRows = computed(() => previewRows.value.filter((row) => row.state === 'ready'))
const skippedRows = computed(() => previewRows.value.filter((row) => row.state === 'existing'))
const issueRows = computed(() =>
  previewRows.value.filter((row) => row.state === 'duplicate' || row.state === 'error'),
)

function buildManualRows(value: string): OpcuaImportPreviewRow[] {
  return parseManualSourceRows(value).map((row, index) => {
    const nodeId = row.nodeId.trim()
    const name = row.name.trim() || nameFromNodeId(nodeId) || `变量${index + 1}`
    return createPreviewRow({
      key: `manual-${index}-${nodeId || row.raw}`,
      source: 'manual',
      rowNo: index + 1,
      name,
      nodeId,
      dataType: normalizeOpcuaDataType(row.dataType.trim() || 'Double'),
      unit: row.unit.trim() || null,
      samplingMs: String(toInteger(row.samplingMs, Number(defaults.samplingMs) || 1000)),
      accessLevel: normalizeAccessLevel(row.accessLevel || defaults.accessLevel),
      deadband: normalizeNullableNumberText(row.deadband, defaults.deadband),
      description: row.description.trim() || null,
    })
  })
}

function buildBrowseRows(ids: string[]): OpcuaImportPreviewRow[] {
  return ids
    .map((id, index) => ({ node: browseRowsById.value.get(id), index }))
    .filter((item): item is { node: OpcuaBrowseRow; index: number } => Boolean(item.node))
    .map(({ node, index }) =>
      createPreviewRow({
        key: `browse-${node.id}`,
        source: 'browse',
        rowNo: index + 1,
        name: node.displayName || node.browseName || node.name || nameFromNodeId(node.nodeId),
        nodeId: node.nodeId,
        browseName: node.browseName || node.name,
        displayName: node.displayName || node.name,
        dataType: normalizeOpcuaDataType(node.dataType || 'Double'),
        unit: defaults.unit || null,
        samplingMs: defaults.samplingMs,
        accessLevel: defaults.accessLevel,
        deadband: defaults.deadband,
        description: null,
      }),
    )
}

function mergeBrowseDraftRows(nodes: OpcuaBrowseNode[]) {
  const candidates = nodes
    .filter((node) => node.nodeType === 'variable' && node.modeled !== true)
    .map((node) => node.id)
  selectedBrowseIds.value = Array.from(new Set([...selectedBrowseIds.value, ...candidates]))
}

function createPreviewRow(row: Partial<OpcuaImportPreviewRow> & { source: OpcuaImportSource; rowNo: number; key: string; nodeId: string }) {
  return {
    name: row.name || nameFromNodeId(row.nodeId),
    code: row.code || '',
    browseName: row.browseName || null,
    displayName: row.displayName || null,
    dataType: normalizeOpcuaDataType(row.dataType || 'Double'),
    samplingMs: String(row.samplingMs || defaults.samplingMs),
    accessLevel: normalizeAccessLevel(row.accessLevel || defaults.accessLevel),
    deadband: String(row.deadband ?? ''),
    unit: row.unit || null,
    description: row.description || null,
    state: 'ready' as OpcuaImportState,
    issue: '',
    ...row,
  }
}

function validateRows(rows: OpcuaImportPreviewRow[]) {
  const nodeIdCounts = countBy(rows.map((row) => row.nodeId.trim()).filter(Boolean))
  const usedCodes = new Set<string>()
  return rows.map((row) => {
    const next = row
    const nodeId = next.nodeId.trim()
    const samplingMs = Number(next.samplingMs)
    const deadband = next.deadband === '' ? null : Number(next.deadband)
    const issues = [
      !nodeId ? 'NodeId 为空' : '',
      !next.dataType.trim() ? '数据类型为空' : '',
      !Number.isFinite(samplingMs) || samplingMs <= 0 || !Number.isInteger(samplingMs)
        ? '采样周期必须为正整数'
        : '',
      deadband !== null && (!Number.isFinite(deadband) || deadband < 0) ? '死区必须大于等于 0' : '',
    ].filter(Boolean)
    next.code = allocateUniqueCode(normalizeCode(next.code || next.name || nameFromNodeId(nodeId)), usedCodes)
    usedCodes.add(next.code)
    if (issues.length > 0) {
      next.state = 'error'
      next.issue = issues.join('；')
      return next
    }
    if (existingNodeIds.value.has(nodeId)) {
      next.state = 'existing'
      next.issue = 'NodeId 已建模，本次将跳过'
      return next
    }
    if ((nodeIdCounts.get(nodeId) || 0) > 1) {
      next.state = 'duplicate'
      next.issue = '本批次 NodeId 重复'
      return next
    }
    next.state = 'ready'
    next.issue = ''
    return next
  })
}

function parseManualSourceRows(value: string) {
  const text = value.trim()
  if (!text) return []
  const parsed = parseTabularText(text)
  const headerMatched = parsed.some((row) =>
    Object.keys(row).some((header) =>
      Object.values(headerAliases).some((aliases) =>
        aliases.map((alias) => alias.toLowerCase()).includes(header.trim().toLowerCase()),
      ),
    ),
  )
  if (headerMatched) {
    return parsed.map((row) => normalizeManualRow(normalizeHeaderRow(row, headerAliases)))
  }
  return parseDelimitedRows(text)
    .filter((row) => row.some((cell) => cell.trim() !== ''))
    .map((row) => normalizeHeaderlessRow(row.map((cell) => cell.trim())))
}

const headerAliases = {
  name: ['变量名', '名称', 'name', 'displayName'],
  code: ['Code', '编码', 'code'],
  nodeId: ['NodeId', '节点ID', 'nodeId'],
  dataType: ['数据类型', '类型', 'dataType', 'type'],
  unit: ['单位', 'unit'],
  samplingMs: ['采样周期', '周期', 'samplingMs', 'interval'],
  accessLevel: ['访问级别', '发布能力', 'accessLevel'],
  deadband: ['死区', 'deadband'],
  description: ['描述', '说明', 'description'],
}

function normalizeManualRow(row: Record<string, string>) {
  return {
    raw: Object.values(row).join(','),
    name: row.name || '',
    code: row.code || '',
    nodeId: row.nodeId || '',
    dataType: row.dataType || 'Double',
    unit: row.unit || '',
    samplingMs: row.samplingMs || '',
    accessLevel: row.accessLevel || '',
    deadband: row.deadband || '',
    description: row.description || '',
  }
}

function normalizeHeaderlessRow(cells: string[]) {
  const nodeIdIndex = cells.findIndex((cell) => looksLikeNodeId(cell))
  if (nodeIdIndex >= 0 && nodeIdIndex !== 0) {
    return normalizeManualRow({
      name: cells.slice(0, nodeIdIndex).join(' '),
      nodeId: cells[nodeIdIndex] || '',
      dataType: cells[nodeIdIndex + 1] || 'Double',
    })
  }
  return normalizeManualRow({
    nodeId: cells[0] || '',
    dataType: cells[1] || 'Double',
    name: cells[2] || '',
    code: cells[3] || '',
  })
}

function submit() {
  if (readyRows.value.length === 0) {
    ElMessage.warning('没有可导入的 OPC UA 变量')
    return
  }
  const skipped = previewRows.value.length - readyRows.value.length
  const runSubmit = () => {
    const rows = readyRows.value.map((row) => buildSubmitRow(row))
    if (rows.some((row) => row === null)) {
      ElMessage.warning('请先修正采样周期或死区后再导入')
      return
    }
    emit('submit', rows.filter((row): row is Record<string, unknown> => row !== null))
  }
  if (skipped > 0) {
    void ElMessageBox.confirm(
      `本次将导入 ${readyRows.value.length} 个变量，跳过 ${skipped} 行问题数据。是否继续？`,
      '确认导入',
    ).then(runSubmit)
    return
  }
  runSubmit()
}

function buildSubmitRow(row: OpcuaImportPreviewRow): Record<string, unknown> | null {
  const samplingMs = Number(row.samplingMs)
  const deadbandText = String(row.deadband || '').trim()
  const deadband = deadbandText === '' ? null : Number(deadbandText)
  if (!Number.isFinite(samplingMs) || samplingMs <= 0 || !Number.isInteger(samplingMs)) return null
  if (deadband !== null && (!Number.isFinite(deadband) || deadband < 0)) return null

  // 批量导入接口使用严格 JSON 解码，只提交后端声明支持的字段，避免 UI 状态字段进入请求体。
  return {
    name: row.name,
    code: row.code,
    nodeId: row.nodeId,
    browseName: row.browseName || null,
    displayName: row.displayName || null,
    dataType: row.dataType,
    unit: row.unit || null,
    samplingMs,
    deadband,
    accessLevel: normalizeAccessLevel(row.accessLevel),
    description: row.description || null,
  }
}

async function goConfirm() {
  if (candidateCount.value === 0) {
    ElMessage.warning(mode.value === 'browse' ? '请先选择要导入的 OPC UA 变量' : '请先输入 NodeId')
    return
  }
  confirmPreparing.value = true
  await nextTick()
  await waitForPaint()
  try {
    if (mode.value === 'browse') {
      browseDraftRows.value = buildBrowseRows(selectedBrowseIds.value)
      if (browseDraftRows.value.length === 0) {
        ElMessage.warning('候选变量数据已失效，请重新展开或刷新浏览树')
        return
      }
    }
    confirmPage.value = 1
    step.value = 'confirm'
  } finally {
    confirmPreparing.value = false
  }
}

function waitForPaint() {
  return new Promise<void>((resolve) => {
    window.requestAnimationFrame(() => window.requestAnimationFrame(() => resolve()))
  })
}

function applyDefaults() {
  const target = mode.value === 'browse' ? browseDraftRows.value : manualDraftRows.value
  target.forEach((row) => {
    row.samplingMs = defaults.samplingMs
    row.accessLevel = defaults.accessLevel
    row.deadband = defaults.deadband
    row.unit = defaults.unit || null
  })
}

function downloadIssueReport() {
  downloadCsv(
    'opcua-import-issues.csv',
    ['行号', '来源', '状态', '变量名', 'Code', 'NodeId', '数据类型', '问题'],
    previewRows.value
      .filter((row) => row.state !== 'ready')
      .map((row) => ({
        行号: row.rowNo,
        来源: row.source === 'browse' ? '浏览导入' : '批量NodeId',
        状态: rowStateText(row.state),
        变量名: row.name,
        Code: row.code,
        NodeId: row.nodeId,
        数据类型: row.dataType,
        问题: row.issue,
      })),
  )
}

function requestClose() {
  void dialogRef.value?.requestClose()
}

function closeSilently() {
  dialogRef.value?.closeSilently()
}

function refreshBrowseTree() {
  selectedBrowseIds.value = []
  browseDraftRows.value = []
  pendingBrowseResolvers.clear()
  treeVersion.value += 1
  emit('browse')
}

function importBrowseSubtree(row: OpcuaBrowseRow) {
  if (subtreeLoadingNodeId.value) return
  subtreeLoadingNodeId.value = row.nodeId
  emit('browse-subtree', row.nodeId, (nodes) => {
    subtreeLoadingNodeId.value = ''
    mergeBrowseDraftRows(nodes)
    nextTick(() => treeRef.value?.setCheckedKeys(selectedBrowseIds.value))
    if (nodes.length === 0) {
      ElMessage.info('当前目录下没有可导入变量')
      return
    }
    ElMessage.success(`已加入 ${nodes.filter((node) => node.nodeType === 'variable' && node.modeled !== true).length} 个子树变量`)
  })
}

function handleBrowseCheck(_: OpcuaBrowseRow, checked: { checkedKeys: Array<string | number> }) {
  const ids = checked.checkedKeys.map(String)
  const checkedFolders = ids
    .map((id) => browseRowsById.value.get(id))
    .filter((row): row is OpcuaBrowseRow => Boolean(row) && row.nodeType === 'folder')
  selectedBrowseIds.value = ids.filter((id) => {
    const row = browseRowsById.value.get(id)
    return row && isBrowseRowSelectable(row)
  })
  nextTick(() => treeRef.value?.setCheckedKeys(selectedBrowseIds.value))
  if (checkedFolders.length > 0) {
    importBrowseSubtree(checkedFolders[0])
  }
}

function isBrowseRowSelectable(row: OpcuaBrowseRow) {
  return row.nodeType === 'variable' && row.modeled !== true
}

function loadBrowseTreeNode(node: { level: number; data?: OpcuaBrowseRow }, resolve: (nodes: OpcuaBrowseRow[]) => void) {
  if (!props.connected) {
    resolve([])
    return
  }
  const parentNodeId = node.level === 0 ? '' : node.data?.nodeId || ''
  const cachedChildren = childrenOf(parentNodeId)
  if (cachedChildren.length > 0) {
    resolve(cachedChildren)
    return
  }
  const key = browsePendingKey(parentNodeId)
  pendingBrowseResolvers.set(key, resolve)
  emit('browse', parentNodeId || undefined)
  window.setTimeout(() => {
    if (!pendingBrowseResolvers.has(key)) return
    pendingBrowseResolvers.delete(key)
    resolve(childrenOf(parentNodeId))
  }, 12000)
}

function resolvePendingBrowseNodes() {
  pendingBrowseResolvers.forEach((resolve, key) => {
    const parentNodeId = key === '__root__' ? '' : key
    resolve(childrenOf(parentNodeId))
    pendingBrowseResolvers.delete(key)
  })
  nextTick(() => treeRef.value?.setCheckedKeys(selectedBrowseIds.value))
}

function childrenOf(parentNodeId: string) {
  if (!parentNodeId) {
    return browseRows.value.filter((row) => !row.parentId)
  }
  const parent = browseRows.value.find((row) => row.nodeId === parentNodeId)
  if (!parent) return []
  return browseRows.value.filter((row) => row.parentId === parent.id)
}

function browsePendingKey(parentNodeId: string) {
  return parentNodeId || '__root__'
}

function filterBrowseTreeNode(keyword: string, data: OpcuaBrowseRow) {
  const text = keyword.trim().toLowerCase()
  if (!text) return true
  return [data.name, data.nodeId, data.dataType || '', data.browseName || '', data.displayName || ''].some(
    (value) => String(value).toLowerCase().includes(text),
  )
}

function rowStateText(state: OpcuaImportState) {
  if (state === 'existing') return '已存在'
  if (state === 'duplicate') return '重复'
  if (state === 'error') return '错误'
  return '可导入'
}

function rowTagType(state: OpcuaImportState) {
  if (state === 'ready') return 'success'
  if (state === 'existing') return 'info'
  return 'warning'
}

function browseStateText(row: OpcuaBrowseRow) {
  if (row.nodeType === 'folder') return '目录'
  return row.modeled ? '已建模' : '可导入'
}

function browseStateTagType(row: OpcuaBrowseRow) {
  if (row.nodeType === 'folder') return 'info'
  return row.modeled ? 'info' : 'success'
}

function looksLikeNodeId(value: string) {
  return /(^|\b)ns=\d+;[isgb]=/i.test(value) || /^[isgb]=/i.test(value)
}

function nameFromNodeId(nodeId: string) {
  return (
    nodeId
      .split(/[.;=:/]/)
      .filter(Boolean)
      .at(-1) || nodeId
  )
}

function normalizeCode(value: string) {
  return (
    String(value || '')
      .trim()
      .toLowerCase()
      .replace(/[^a-z0-9_]+/g, '_')
      .replace(/^_+|_+$/g, '') || 'node'
  )
}

function allocateUniqueCode(base: string, used: Set<string>) {
  if (!used.has(base)) return base
  for (let index = 2; index < 10000; index += 1) {
    const candidate = `${base}_${index}`
    if (!used.has(candidate)) return candidate
  }
  return `${base}_${Date.now()}`
}

function countBy(values: string[]) {
  const result = new Map<string, number>()
  values.forEach((value) => result.set(value, (result.get(value) || 0) + 1))
  return result
}

function normalizeAccessLevel(value: string) {
  const text = String(value || '').trim().toLowerCase()
  if (['write', '写'].includes(text)) return 'Write'
  if (['readwrite', 'read_write', '读写'].includes(text)) return 'ReadWrite'
  return 'Read'
}

function toInteger(value: string, fallback: number) {
  if (value === '') return fallback
  const parsed = Number(value)
  return Number.isFinite(parsed) ? Math.trunc(parsed) : fallback
}

function normalizeNullableNumberText(value: string, fallback: string) {
  if (value === '') return fallback
  const parsed = Number(value)
  return Number.isFinite(parsed) ? String(parsed) : fallback
}

function normalizeOpcuaDataType(value: string) {
  const text = String(value || '').trim()
  const aliases: Record<string, string> = {
    'i=1': 'Boolean',
    'i=2': 'SByte',
    'i=3': 'Byte',
    'i=4': 'Int16',
    'i=5': 'UInt16',
    'i=6': 'Int32',
    'i=7': 'UInt32',
    'i=8': 'Int64',
    'i=9': 'UInt64',
    'i=10': 'Float',
    'i=11': 'Double',
    'i=12': 'String',
    'i=13': 'DateTime',
    'i=17': 'NodeID',
    'i=20': 'QualifiedName',
    'i=21': 'LocalizedText',
    'i=24': 'BaseDataType',
    'i=26': 'Number',
    'i=27': 'Integer',
    'i=28': 'UInteger',
    'i=290': 'Duration',
    'i=295': 'LocaleID',
  }
  return aliases[text] || text
}

defineExpose({ closeSilently })
</script>

<style scoped>
.opcua-import {
  display: grid;
  gap: 12px;
  max-height: calc(100vh - 172px);
  overflow: hidden;
  padding-right: 2px;
}

.opcua-import__summary {
  padding: 10px 12px;
  border: 1px solid color-mix(in oklch, var(--dc-primary) 18%, var(--dc-border));
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-subtle);
  display: flex;
  align-items: center;
  gap: 18px;
}

.opcua-import__summary div {
  display: inline-flex;
  align-items: baseline;
  gap: 6px;
}

.opcua-import__summary strong {
  color: var(--dc-primary);
  font-size: 18px;
}

.opcua-import__summary span {
  color: var(--dc-text-secondary);
  font-size: 12px;
  font-weight: 700;
}

.opcua-import__summary em {
  margin-left: auto;
  color: var(--dc-text-muted);
  font-style: normal;
  font-size: 12px;
}

.opcua-import__tools {
  display: flex;
  align-items: center;
  gap: 8px;
}

.opcua-import__tools :deep(.el-button),
.opcua-import__toolbar :deep(.el-button) {
  gap: 5px;
}

.opcua-import__tools svg,
.opcua-import__toolbar svg {
  width: 14px;
  height: 14px;
}

.opcua-import section {
  padding: 10px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-subtle);
}

.opcua-import__section-head {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 8px;
}

.opcua-import__section-head strong {
  color: var(--dc-text);
  font-size: 13px;
}

.opcua-import__section-head span {
  color: var(--dc-text-muted);
  font-size: 12px;
}

.opcua-import__toolbar {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}

.opcua-import__hint {
  margin-bottom: 8px;
  padding: 7px 9px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
  color: var(--dc-text-muted);
  font-size: 12px;
  line-height: 18px;
}

.opcua-import__hint.is-warning {
  border-color: color-mix(in oklch, var(--el-color-warning) 28%, var(--dc-border));
  color: var(--el-color-warning);
}

.opcua-import__browse-tree {
  height: 360px;
  overflow: auto;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface);
}

.opcua-import__browse-tree :deep(.el-tree) {
  min-width: 760px;
  padding: 6px 0;
  background: transparent;
}

.opcua-import__browse-tree :deep(.el-tree-node__content) {
  height: 44px;
  align-items: flex-start;
  padding-top: 4px;
}

.opcua-import__tree-node {
  min-width: 0;
  display: grid;
  gap: 3px;
  width: 100%;
  padding-right: 10px;
}

.opcua-import__tree-title {
  min-width: 0;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  overflow: hidden;
}

.opcua-import__tree-title strong {
  min-width: 0;
  color: var(--dc-text);
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
  overflow: hidden;
}

.opcua-import__tree-title svg {
  width: 13px;
  height: 13px;
  flex: 0 0 auto;
  color: var(--dc-text-muted);
}

.opcua-import__tree-meta {
  min-width: 0;
  display: inline-flex;
  gap: 10px;
  overflow: hidden;
  color: var(--dc-text-muted);
  font-size: 11px;
  line-height: 14px;
}

.opcua-import__tree-meta span {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.opcua-import__tree-meta em {
  flex: 0 0 auto;
  font-style: normal;
  color: var(--dc-text-secondary);
}

.opcua-import__default-grid {
  display: grid;
  grid-template-columns: minmax(110px, 140px) minmax(110px, 140px) minmax(90px, 120px) minmax(90px, 1fr) auto;
  align-items: center;
  gap: 8px;
}

.opcua-import__default-grid :deep(.el-select),
.opcua-import__default-grid :deep(.el-input) {
  width: 100%;
}

.opcua-import :deep(.el-table) {
  --el-table-header-bg-color: var(--dc-surface-raised);
  --el-table-border-color: var(--dc-border);
}

.opcua-import__confirm-table {
  width: 100%;
}

.opcua-import__confirm-table :deep(.el-table__inner-wrapper) {
  min-width: 1220px;
}

.opcua-import__confirm-table :deep(.cell) {
  padding-left: 7px;
  padding-right: 7px;
}

.opcua-import__confirm-table :deep(.el-input__wrapper),
.opcua-import__confirm-table :deep(.el-select__wrapper) {
  min-height: 30px;
  box-shadow: 0 0 0 1px var(--dc-border) inset;
}

.opcua-import__confirm-table :deep(.el-table__body-wrapper) {
  overflow: auto;
}

.opcua-import__pager {
  display: flex;
  justify-content: flex-end;
  padding-top: 8px;
  border-top: 1px solid var(--dc-border);
  margin-top: 8px;
}
</style>

<style>
.opcua-import-dialog .el-dialog__body {
  overflow: hidden;
}

.opcua-import-dialog .dc-dialog__body {
  overflow: hidden;
}
</style>
