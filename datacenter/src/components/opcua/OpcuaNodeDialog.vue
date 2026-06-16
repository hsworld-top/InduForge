<template>
  <DcDialog
    ref="dialogRef"
    v-model="visible"
    :title="mode === 'edit' ? '编辑 OPC UA 变量' : '新建 OPC UA 变量'"
    width="760px"
    body-max-height="calc(100vh - 180px)"
    :dirty="isDirty"
    :close-disabled="loading"
  >
    <el-form class="opcua-node-form" label-position="top" @submit.prevent>
      <section class="opcua-node-form__section">
        <div class="opcua-node-form__section-head">
          <strong>变量信息</strong>
        </div>
        <div class="opcua-node-form__grid">
          <el-form-item label="变量名称" required>
            <el-input
              v-model="form.name"
              maxlength="100"
              show-word-limit
              placeholder="例如：1 号线电机转速"
              @input="sanitizeName"
            />
          </el-form-item>

          <el-form-item label="所属变量组">
            <el-select v-model="form.groupId" clearable filterable placeholder="未分组">
              <el-option
                v-for="group in groupOptions"
                :key="group.id"
                :label="group.label"
                :value="group.id"
              >
                <span
                  class="opcua-node-form__group-option"
                  :class="{ 'is-child': group.depth > 0 }"
                  :style="{ paddingLeft: `${group.depth * 14}px` }"
                >
                  <span v-if="group.depth > 0" class="opcua-node-form__group-guide" />
                  {{ group.label }}
                </span>
              </el-option>
            </el-select>
          </el-form-item>

          <el-form-item label="单位">
            <el-input v-model="form.unit" maxlength="20" placeholder="例如：rpm、MPa、℃" />
          </el-form-item>
        </div>
      </section>

      <section class="opcua-node-form__section">
        <div class="opcua-node-form__section-head">
          <strong>OPC UA 节点</strong>
        </div>
        <el-form-item required>
          <template #label>
            <span class="opcua-node-form__label">
              NodeId
              <el-tooltip content="服务器节点地址" placement="top">
                <IconTablerInfoCircle />
              </el-tooltip>
            </span>
          </template>
          <el-input
            v-model="form.nodeId"
            maxlength="1000"
            placeholder="例如：ns=2;s=Line1.Motor01.Speed"
            @blur="fillNameAndCodeFromNodeId"
          />
        </el-form-item>
        <div class="opcua-node-form__grid">
          <el-form-item label="BrowseName">
            <el-input v-model="form.browseName" placeholder="可选，服务器浏览名" />
          </el-form-item>

          <el-form-item label="数据类型" required>
            <el-select
              v-model="form.dataType"
              allow-create
              filterable
              placeholder="选择或输入数据类型"
            >
              <el-option v-for="type in dataTypes" :key="type" :label="type" :value="type" />
            </el-select>
          </el-form-item>
        </div>
      </section>

      <section class="opcua-node-form__section">
        <div class="opcua-node-form__section-head">
          <strong>采集与权限</strong>
        </div>
        <div class="opcua-node-form__grid">
          <el-form-item>
            <template #label>
              <span class="opcua-node-form__label">
                采集周期
                <el-tooltip content="单位 ms" placement="top">
                  <IconTablerInfoCircle />
                </el-tooltip>
              </span>
            </template>
            <el-input-number v-model="form.samplingMs" :min="1" :step="100" />
          </el-form-item>

          <el-form-item>
            <template #label>
              <span class="opcua-node-form__label">
                变化死区
                <el-tooltip content="0 表示不启用" placement="top">
                  <IconTablerInfoCircle />
                </el-tooltip>
              </span>
            </template>
            <el-input-number v-model="form.deadband" :min="0" :precision="3" :step="0.1" />
          </el-form-item>
        </div>

        <el-form-item>
          <template #label>
            <span class="opcua-node-form__label">
              运行权限
              <el-tooltip content="仅配置权限" placement="top">
                <IconTablerInfoCircle />
              </el-tooltip>
            </span>
          </template>
          <el-segmented v-model="form.accessLevel" :options="accessLevelOptions" />
        </el-form-item>
      </section>

      <section class="opcua-node-form__section">
        <div class="opcua-node-form__section-head">
          <strong>备注</strong>
        </div>
        <el-form-item>
          <el-input
            v-model="form.description"
            type="textarea"
            :rows="3"
            placeholder="例如：来自 1 号线 PLC，OPC UA 浏览导入后手工确认"
          />
        </el-form-item>
      </section>
    </el-form>

    <template #footer>
      <div class="opcua-node-form__footer">
        <el-button @click="requestClose">取消</el-button>
        <el-button type="primary" :loading="loading" @click="submit">保存变量</el-button>
      </div>
    </template>
  </DcDialog>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import IconTablerInfoCircle from '~icons/tabler/info-circle'
import DcDialog from '@/components/shared/DcDialog.vue'
import type { OpcuaNode, OpcuaNodeGroup } from './types'

type FormState = {
  name: string
  code: string
  groupId: string
  nodeId: string
  browseName: string
  dataType: string
  unit: string
  samplingMs: number
  deadband: number | null
  accessLevel: string
  description: string
}

type OpcuaGroupOptionNode = OpcuaNodeGroup & {
  children: OpcuaGroupOptionNode[]
}

const props = defineProps<{
  modelValue: boolean
  mode: 'create' | 'edit'
  groups: OpcuaNodeGroup[]
  node?: OpcuaNode | null
  defaultGroupId?: string
  loading?: boolean
}>()

const emit = defineEmits<{
  (event: 'update:modelValue', value: boolean): void
  (event: 'submit', value: Record<string, unknown>): void
}>()

const visible = computed({
  get: () => props.modelValue,
  set: (value: boolean) => emit('update:modelValue', value),
})

const dialogRef = ref<InstanceType<typeof DcDialog> | null>(null)
const initialSnapshot = ref('')
const dataTypes = [
  'Boolean',
  'Int16',
  'UInt16',
  'Int32',
  'UInt32',
  'Int64',
  'Float',
  'Double',
  'String',
  'DateTime',
]
const accessLevelOptions = [
  { label: '只读', value: 'Read' },
  { label: '读写', value: 'ReadWrite' },
]
const invalidVariableNamePattern = /[^\u4e00-\u9fa5A-Za-z0-9_$#%@+()[\]&-]/g
const validVariableNamePattern = /^[\u4e00-\u9fa5A-Za-z0-9_$#%@+()[\]&-]+$/

const form = reactive<FormState>({
  name: '',
  code: '',
  groupId: '',
  nodeId: '',
  browseName: '',
  dataType: 'Double',
  unit: '',
  samplingMs: 1000,
  deadband: 0,
  accessLevel: 'Read',
  description: '',
})

const groupOptions = computed(() => {
  const nodes = new Map<string, OpcuaGroupOptionNode>()
  props.groups.forEach((group) => {
    nodes.set(group.id, { ...group, children: [] })
  })
  const roots: OpcuaGroupOptionNode[] = []
  nodes.forEach((node) => {
    const parent = node.parentId ? nodes.get(node.parentId) : null
    if (parent) parent.children.push(node)
    else roots.push(node)
  })
  const sortNodes = (items: OpcuaGroupOptionNode[]) => {
    items.sort(
      (left, right) =>
        (left.sortOrder || 0) - (right.sortOrder || 0) || left.name.localeCompare(right.name),
    )
    items.forEach((item) => sortNodes(item.children))
  }
  sortNodes(roots)
  const visit = (
    group: OpcuaGroupOptionNode,
    depth: number,
  ): Array<{
    id: string
    label: string
    depth: number
  }> => [
    { id: group.id, label: group.name, depth },
    ...group.children.flatMap((child) => visit(child, depth + 1)),
  ]
  return roots.flatMap((group) => visit(group, 0))
})

const isDirty = computed(() => props.modelValue && snapshotForm() !== initialSnapshot.value)

watch(
  () => [props.modelValue, props.node, props.defaultGroupId] as const,
  () => {
    if (!props.modelValue) return
    resetForm()
  },
  { immediate: true },
)

function resetForm() {
  form.name = props.node?.name || ''
  form.code = props.node?.code || ''
  form.groupId = props.node?.groupId || props.defaultGroupId || ''
  form.nodeId = props.node?.nodeId || ''
  form.browseName = props.node?.browseName || ''
  form.dataType = props.node?.dataType || 'Double'
  form.unit = props.node?.unit || ''
  form.samplingMs = props.node?.samplingMs || 1000
  form.deadband = props.node?.deadband ?? 0
  form.accessLevel = normalizeFormAccessLevel(props.node?.accessLevel)
  form.description = props.node?.description || ''
  initialSnapshot.value = snapshotForm()
}

function submit() {
  sanitizeName()
  if (!form.name.trim() || !form.nodeId.trim() || !form.dataType.trim()) {
    ElMessage.warning('请填写变量名称、NodeId 和数据类型')
    return
  }
  if (!isValidVariableName(form.name)) {
    ElMessage.warning('变量名称包含不支持的字符')
    return
  }
  const code = form.code.trim() || toVariableCode(nodeIdTail(form.nodeId))
  emit('submit', {
    name: form.name.trim(),
    code,
    groupId: form.groupId || null,
    hasGroupId: true,
    nodeId: form.nodeId.trim(),
    browseName: form.browseName.trim() || null,
    dataType: form.dataType.trim(),
    unit: form.unit.trim() || null,
    samplingMs: form.samplingMs,
    deadband: form.deadband,
    accessLevel: form.accessLevel,
    description: form.description.trim() || null,
  })
}

function requestClose() {
  void dialogRef.value?.requestClose()
}

function closeSilently() {
  dialogRef.value?.closeSilently()
}

function sanitizeName() {
  const next = form.name.replace(invalidVariableNamePattern, '')
  if (next !== form.name) form.name = next
}

function isValidVariableName(value: string) {
  return validVariableNamePattern.test(value)
}

function fillNameAndCodeFromNodeId() {
  if (!form.nodeId.trim()) return
  const fallback = nodeIdTail(form.nodeId)
  if (!form.name.trim()) form.name = fallback
  if (props.mode !== 'edit' && !form.code.trim()) form.code = toVariableCode(fallback)
}

function snapshotForm() {
  return JSON.stringify({ ...form })
}

function nodeIdTail(nodeId: string) {
  return (
    nodeId
      .split(/[.;=:/]/)
      .filter(Boolean)
      .at(-1) || nodeId.trim()
  )
}

function toVariableCode(value: string) {
  const normalized = value
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9_]+/g, '_')
    .replace(/^_+|_+$/g, '')
  return normalized || 'node'
}

function normalizeFormAccessLevel(value?: string | null) {
  return value === 'ReadWrite' || value === 'Write' ? 'ReadWrite' : 'Read'
}

defineExpose({ closeSilently })
</script>

<style scoped>
.opcua-node-form {
  display: grid;
  gap: 12px;
}

.opcua-node-form__section {
  padding: 12px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-subtle);
}

.opcua-node-form__section-head {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 10px;
}

.opcua-node-form__section-head strong {
  color: var(--dc-text);
  font-size: 13px;
}

.opcua-node-form__section-head span {
  color: var(--dc-text-muted);
  font-size: 12px;
}

.opcua-node-form__grid {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
  gap: 12px 14px;
}

.opcua-node-form__footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}

.opcua-node-form :deep(.el-form-item) {
  margin-bottom: 0;
}

.opcua-node-form__label {
  display: inline-flex;
  align-items: center;
  gap: 5px;
}

.opcua-node-form__label svg {
  width: 14px;
  height: 14px;
  color: var(--dc-text-muted);
}

.opcua-node-form__group-option {
  min-width: 0;
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.opcua-node-form__group-guide {
  width: 10px;
  height: 1px;
  flex: 0 0 auto;
  background: var(--dc-border);
}

.opcua-node-form :deep(.el-select),
.opcua-node-form :deep(.el-input-number),
.opcua-node-form :deep(.el-segmented) {
  width: 100%;
}

@media (max-width: 720px) {
  .opcua-node-form__grid {
    grid-template-columns: 1fr;
  }
}
</style>
