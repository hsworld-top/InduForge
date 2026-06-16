<template>
  <DcDialog
    ref="dialogRef"
    v-model="visible"
    title="从 OPC UA 导入变量"
    width="940px"
    class="opcua-import-dialog"
    body-max-height="calc(100vh - 180px)"
    :dirty="isDirty"
    :close-disabled="loading"
  >
    <div class="opcua-import">
      <el-tabs v-model="mode">
        <el-tab-pane label="浏览导入" name="browse">
          <section>
            <div class="opcua-import__section-head">
              <strong>服务器节点浏览</strong>
              <span>{{ browseSummary }}</span>
            </div>
            <div class="opcua-import__toolbar">
              <el-input
                v-model="browseKeyword"
                size="small"
                clearable
                placeholder="搜索 NodeId / BrowseName / 路径"
              />
              <el-button
                size="small"
                :loading="browseLoading"
                :disabled="!connected"
                @click="$emit('browse')"
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
            <el-table
              :data="filteredBrowseRows"
              height="300"
              size="small"
              row-key="id"
              @selection-change="handleBrowseSelection"
            >
              <el-table-column type="selection" width="42" :selectable="isBrowseRowSelectable" />
              <el-table-column label="节点" min-width="180" show-overflow-tooltip>
                <template #default="{ row }">
                  <div class="opcua-import__node-name">
                    <strong>{{ row.name }}</strong>
                    <span>{{ row.path }}</span>
                  </div>
                </template>
              </el-table-column>
              <el-table-column prop="nodeId" label="NodeId" min-width="250" show-overflow-tooltip />
              <el-table-column prop="dataType" label="类型" width="110" />
              <el-table-column label="状态" width="104">
                <template #default="{ row }">
                  <el-tag size="small" :type="row.modeled ? 'info' : 'success'">
                    {{ row.modeled ? '已建模' : '可导入' }}
                  </el-tag>
                </template>
              </el-table-column>
            </el-table>
          </section>
        </el-tab-pane>
        <el-tab-pane label="批量 NodeId" name="manual">
          <section>
            <div class="opcua-import__section-head">
              <strong>批量 NodeId</strong>
              <span>每行一个变量，导入前可改变量名</span>
            </div>
            <el-input
              v-model="rawText"
              type="textarea"
              :rows="8"
              placeholder="每行一个 NodeId，可用逗号补充类型与变量名：ns=2;s=Line1.Motor01.Speed,Double,电机转速"
            />
          </section>
        </el-tab-pane>
      </el-tabs>

      <section>
        <div class="opcua-import__section-head">
          <strong>导入确认</strong>
          <span>{{ rows.length }} 个候选变量</span>
        </div>
        <el-table :data="rows" height="220" size="small">
          <el-table-column label="变量名" min-width="150">
            <template #default="{ row }"><el-input v-model="row.name" size="small" /></template>
          </el-table-column>
          <el-table-column prop="nodeId" label="NodeId" min-width="230" show-overflow-tooltip />
          <el-table-column label="类型" width="120">
            <template #default="{ row }"><el-input v-model="row.dataType" size="small" /></template>
          </el-table-column>
          <el-table-column label="采样" width="110">
            <template #default="{ row }"
              ><el-input-number v-model="row.samplingMs" size="small" :min="1"
            /></template>
          </el-table-column>
        </el-table>
      </section>
    </div>
    <template #footer>
      <el-button @click="requestClose">取消</el-button>
      <el-button type="primary" :loading="loading" @click="submit">导入</el-button>
    </template>
  </DcDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import DcDialog from '@/components/shared/DcDialog.vue'
import type { OpcuaBrowseNode } from './types'
import IconTablerRefresh from '~icons/tabler/refresh'

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
  (event: 'browse'): void
}>()

const visible = computed({
  get: () => props.modelValue,
  set: (value: boolean) => emit('update:modelValue', value),
})

const dialogRef = ref<InstanceType<typeof DcDialog> | null>(null)
const rawText = ref('')
const mode = ref<'browse' | 'manual'>('browse')
const browseKeyword = ref('')
const selectedBrowseIds = ref<string[]>([])
const isDirty = computed(
  () => props.modelValue && (rawText.value.trim().length > 0 || selectedBrowseIds.value.length > 0),
)

watch(
  () => props.modelValue,
  (visible) => {
    if (!visible) return
    rawText.value = ''
    browseKeyword.value = ''
    selectedBrowseIds.value = []
    mode.value = props.connected ? 'browse' : 'manual'
  },
)

const browseDiagnostics = computed(() => props.browseDiagnostics || [])
const browseNodeById = computed(
  () => new Map((props.browseNodes || []).map((node) => [node.id, node])),
)
const browseRows = computed(() =>
  (props.browseNodes || [])
    .filter((node) => node.nodeType === 'variable')
    .map((node) => ({
      ...node,
      path: browsePathOf(node),
      modeled: node.modeled === true,
    })),
)
const filteredBrowseRows = computed(() => {
  const text = browseKeyword.value.trim().toLowerCase()
  if (!text) return browseRows.value
  return browseRows.value.filter((row) =>
    [row.name, row.nodeId, row.dataType || '', row.path].some((value) =>
      String(value).toLowerCase().includes(text),
    ),
  )
})
const browseSummary = computed(() => {
  const importable = browseRows.value.filter((row) => !row.modeled).length
  return `${browseRows.value.length} 个变量节点 · ${importable} 个可导入`
})

const manualRows = computed(() =>
  rawText.value
    .split(/\r?\n/)
    .map((line) => line.trim())
    .filter(Boolean)
    .map((line) => {
      const [nodeId, dataType = 'Double', name = ''] = line.split(',').map((part) => part.trim())
      return {
        name:
          name ||
          nodeId
            .split(/[.;=:/]/)
            .filter(Boolean)
            .at(-1) ||
          nodeId,
        code: '',
        nodeId,
        dataType,
        samplingMs: 1000,
      }
    }),
)

const browseImportRows = computed(() =>
  selectedBrowseIds.value
    .map((id) => browseRows.value.find((row) => row.id === id))
    .filter((row): row is (typeof browseRows.value)[number] => Boolean(row) && !row.modeled)
    .map((row) => ({
      name: row.name,
      code: '',
      nodeId: row.nodeId,
      browseName: row.name,
      displayName: row.name,
      dataType: row.dataType || 'Double',
      samplingMs: 1000,
    })),
)

const rows = computed(() => (mode.value === 'browse' ? browseImportRows.value : manualRows.value))

const submit = () => {
  if (rows.value.length === 0) {
    ElMessage.warning(
      mode.value === 'browse' ? '请选择可导入的 OPC UA 变量节点' : '请至少输入一个 NodeId',
    )
    return
  }
  emit('submit', rows.value)
}

function requestClose() {
  void dialogRef.value?.requestClose()
}

function closeSilently() {
  dialogRef.value?.closeSilently()
}

function handleBrowseSelection(selection: Array<{ id: string }>) {
  selectedBrowseIds.value = selection.map((row) => row.id)
}

function isBrowseRowSelectable(row: { modeled?: boolean }) {
  return row.modeled !== true
}

function browsePathOf(node: OpcuaBrowseNode) {
  const segments: string[] = [node.name]
  const visited = new Set<string>([node.id])
  let parentId = node.parentId || ''
  while (parentId && !visited.has(parentId)) {
    visited.add(parentId)
    const parent = browseNodeById.value.get(parentId)
    if (!parent) break
    segments.unshift(parent.name)
    parentId = parent.parentId || ''
  }
  return segments.join('/')
}

defineExpose({ closeSilently })
</script>

<style scoped>
.opcua-import {
  display: grid;
  gap: 12px;
  max-height: calc(100vh - 220px);
  overflow-y: auto;
  padding-right: 2px;
}

.opcua-import :deep(.el-tabs) {
  min-height: 0;
}

.opcua-import :deep(.el-tabs__content) {
  min-height: 0;
}

.opcua-import :deep(.el-tab-pane) {
  min-height: 0;
}

.opcua-import section {
  padding: 10px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-subtle);
}

.opcua-import__browse-section,
.opcua-import__manual-section,
.opcua-import__confirm-section {
  min-height: 0;
  overflow: hidden;
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

.opcua-import__toolbar :deep(.el-button) {
  gap: 5px;
}

.opcua-import__toolbar svg {
  width: 14px;
  height: 14px;
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

.opcua-import__node-name {
  min-width: 0;
  display: grid;
  gap: 2px;
}

.opcua-import__node-name strong,
.opcua-import__node-name span {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.opcua-import__node-name strong {
  color: var(--dc-text);
  font-size: 12px;
}

.opcua-import__node-name span {
  color: var(--dc-text-muted);
  font-size: 11px;
}

.opcua-import :deep(.el-table) {
  --el-table-header-bg-color: var(--dc-surface-raised);
  --el-table-border-color: var(--dc-border);
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
