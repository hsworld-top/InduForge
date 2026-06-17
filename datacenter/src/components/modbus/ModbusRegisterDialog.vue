<template>
  <DcDialog
    ref="dialogRef"
    v-model="visible"
    :title="mode === 'edit' ? '编辑 Modbus 变量' : '新建 Modbus 变量'"
    width="760px"
    body-max-height="calc(100vh - 180px)"
    :dirty="isDirty"
    :close-disabled="loading"
  >
    <el-form class="modbus-register-dialog" label-position="top">
      <section>
        <h3><span>01</span>常用配置</h3>
        <div class="modbus-register-dialog__grid">
          <el-form-item label="变量名" required>
            <el-input
              v-model="form.name"
              maxlength="100"
              show-word-limit
              placeholder="例如：1 号线电机转速"
              @input="sanitizeName"
            />
          </el-form-item>
          <el-form-item label="从站">
            <el-select v-model="form.unitId" filterable @change="applySelectedSlaveDefaults">
              <el-option
                v-for="slave in slaves"
                :key="slave.id"
                :label="`${slave.name} (${slave.unitId})`"
                :value="slave.unitId"
              />
            </el-select>
          </el-form-item>
          <el-form-item label="变量组">
            <el-select v-model="form.groupId" clearable filterable placeholder="未分组">
              <el-option
                v-for="group in groupOptions"
                :key="group.id"
                :label="group.label"
                :value="group.id"
              >
                <span
                  class="modbus-register-dialog__group-option"
                  :style="{ paddingLeft: `${group.depth * 14}px` }"
                >
                  <span v-if="group.depth > 0" class="modbus-register-dialog__group-guide" />
                  {{ group.label }}
                </span>
              </el-option>
            </el-select>
          </el-form-item>
          <el-form-item>
            <template #label>
              <span class="modbus-register-dialog__label">
                区域
                <el-tooltip
                  content="选择 Modbus 数据区：线圈/保持寄存器通常可读写；离散输入/输入寄存器通常只读。"
                  placement="top"
                >
                  <IconTablerInfoCircle />
                </el-tooltip>
              </span>
            </template>
            <el-select v-model="form.area" @change="applyAreaDefaults">
              <el-option
                v-for="option in areaOptions"
                :key="option.value"
                :label="option.label"
                :value="option.value"
              >
                <span class="modbus-register-dialog__option">
                  <span>{{ option.label }}</span>
                  <em>{{ option.hint }}</em>
                </span>
              </el-option>
            </el-select>
          </el-form-item>
          <el-form-item label="用户地址"
            ><el-input-number v-model="form.address" :min="0"
          /></el-form-item>
          <el-form-item label="数据类型">
            <el-select v-model="form.dataType" @change="applyDataTypeDefaults">
              <el-option label="bool" value="bool" />
              <el-option label="uint16" value="uint16" />
              <el-option label="int16" value="int16" />
              <el-option label="uint32" value="uint32" />
              <el-option label="int32" value="int32" />
              <el-option label="float32" value="float32" />
              <el-option label="float64" value="float64" />
            </el-select>
          </el-form-item>
          <el-form-item label="倍率"><el-input-number v-model="form.scale" /></el-form-item>
          <el-form-item label="单位"><el-input v-model="form.unit" /></el-form-item>
          <el-form-item label="轮询周期(ms)"
            ><el-input-number v-model="form.pollIntervalMs" :min="100"
          /></el-form-item>
          <el-form-item label="读写权限">
            <el-select v-model="form.accessLevel">
              <el-option label="只读" value="Read" />
              <el-option label="读写" value="ReadWrite" />
            </el-select>
          </el-form-item>
          <el-form-item label="状态">
            <el-select v-model="form.status">
              <el-option label="启用" value="active" />
              <el-option label="停用" value="inactive" />
            </el-select>
          </el-form-item>
        </div>
      </section>
      <el-collapse v-model="advancedPanels" class="modbus-register-dialog__advanced">
        <el-collapse-item title="高级配置" name="advanced">
          <div class="modbus-register-dialog__grid">
            <el-form-item>
              <template #label>
                <span class="modbus-register-dialog__label">
                  地址基准
                  <el-tooltip
                    content="用于把用户看到的 40001/30001 或 1-based 地址转换为 Modbus 协议里的 0-based 地址。"
                    placement="top"
                  >
                    <IconTablerInfoCircle />
                  </el-tooltip>
                </span>
              </template>
              <el-select v-model="form.addressBase">
                <el-option
                  v-for="option in addressBaseOptions"
                  :key="option.value"
                  :label="option.label"
                  :value="option.value"
                >
                  <span class="modbus-register-dialog__option">
                    <span>{{ option.label }}</span>
                    <em>{{ option.hint }}</em>
                  </span>
                </el-option>
              </el-select>
            </el-form-item>
            <el-form-item label="寄存器数量">
              <el-input-number v-model="form.quantity" :min="1" />
            </el-form-item>
            <el-form-item>
              <template #label>
                <span class="modbus-register-dialog__label">
                  Bit 位
                  <el-tooltip
                    content="仅用于从保持寄存器/输入寄存器的某一位解析 bool；线圈和离散输入不需要填写。"
                    placement="top"
                  >
                    <IconTablerInfoCircle />
                  </el-tooltip>
                </span>
              </template>
              <el-input-number
                v-model="form.bitIndex"
                :min="0"
                :max="15"
                :disabled="bitIndexDisabled"
                :placeholder="bitIndexPlaceholder"
              />
            </el-form-item>
            <el-form-item label="字节序">
              <el-select v-model="form.byteOrder">
                <el-option
                  v-for="option in byteOrderOptions"
                  :key="option.value"
                  :label="option.label"
                  :value="option.value"
                />
              </el-select>
            </el-form-item>
            <el-form-item>
              <template #label>
                <span class="modbus-register-dialog__label">
                  字序
                  <el-tooltip
                    content="32/64 位数据跨多个寄存器时，高字在前或低字在前；16 位和 bool 通常无需关注。"
                    placement="top"
                  >
                    <IconTablerInfoCircle />
                  </el-tooltip>
                </span>
              </template>
              <el-select v-model="form.wordOrder">
                <el-option label="高字在前（默认）" value="high_first" />
                <el-option label="低字在前" value="low_first" />
              </el-select>
            </el-form-item>
            <el-form-item label="偏移"><el-input-number v-model="form.offset" /></el-form-item>
            <el-form-item>
              <template #label>
                <span class="modbus-register-dialog__label">
                  超时(ms)
                  <el-tooltip
                    content="留空表示跟随从站请求超时配置；只有单变量需要覆盖时填写。"
                    placement="top"
                  >
                    <IconTablerInfoCircle />
                  </el-tooltip>
                </span>
              </template>
              <el-input-number v-model="form.timeoutMs" :min="1" placeholder="默认跟随从站" />
            </el-form-item>
            <el-form-item>
              <template #label>
                <span class="modbus-register-dialog__label">
                  重试次数
                  <el-tooltip
                    content="留空表示跟随从站重试配置；只有单变量需要覆盖时填写。"
                    placement="top"
                  >
                    <IconTablerInfoCircle />
                  </el-tooltip>
                </span>
              </template>
              <el-input-number v-model="form.retryCount" :min="0" placeholder="默认跟随从站" />
            </el-form-item>
          </div>
          <el-form-item label="说明">
            <el-input v-model="form.description" type="textarea" :rows="2" resize="none" />
          </el-form-item>
        </el-collapse-item>
      </el-collapse>
    </el-form>
    <template #footer>
      <el-button @click="requestClose">取消</el-button>
      <el-button type="primary" :loading="loading" @click="submit">保存</el-button>
    </template>
  </DcDialog>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import DcDialog from '@/components/shared/DcDialog.vue'
import type { ModbusRegister, ModbusRegisterGroup, ModbusSlaveDevice } from './types'
import IconTablerInfoCircle from '~icons/tabler/info-circle'

const props = defineProps<{
  modelValue: boolean
  mode: 'create' | 'edit'
  groups: ModbusRegisterGroup[]
  slaves?: ModbusSlaveDevice[]
  register?: ModbusRegister | null
  existingCodes?: string[]
  defaultGroupId?: string
  defaultUnitId?: number
  loading?: boolean
}>()

const emit = defineEmits<{
  (event: 'update:modelValue', value: boolean): void
  (event: 'submit', payload: Record<string, unknown>): void
}>()

const visible = computed({
  get: () => props.modelValue,
  set: (value: boolean) => emit('update:modelValue', value),
})
const dialogRef = ref<InstanceType<typeof DcDialog> | null>(null)
const initialSnapshot = ref('')
const advancedPanels = ref<string[]>([])
const invalidVariableNamePattern = /[^\u4e00-\u9fa5A-Za-z0-9_$#%@+()[\]&-]/g
const validVariableNamePattern = /^[\u4e00-\u9fa5A-Za-z0-9_$#%@+()[\]&-]+$/
const addressBaseOptions = [
  { label: 'Modicon 地址（40001/30001）', value: 'modicon', hint: '常见 PLC 文档写法' },
  { label: '从 1 开始', value: 'one_based', hint: '用户地址 1 对应协议地址 0' },
  { label: '从 0 开始', value: 'zero_based', hint: '用户地址就是协议地址' },
]
const areaOptions = [
  { label: '保持寄存器', value: 'holding_register', hint: 'Holding Register，可读写数值' },
  { label: '输入寄存器', value: 'input_register', hint: 'Input Register，只读数值' },
  { label: '线圈', value: 'coil', hint: 'Coil，可读写开关量' },
  { label: '离散输入', value: 'discrete_input', hint: 'Discrete Input，只读开关量' },
]
const byteOrderOptions = [
  { label: 'ABCD（默认）', value: 'ABCD' },
  { label: 'BADC（字节交换）', value: 'BADC' },
  { label: 'CDAB（字交换）', value: 'CDAB' },
  { label: 'DCBA（字节和字都交换）', value: 'DCBA' },
]
type ModbusGroupOptionNode = ModbusRegisterGroup & {
  children: ModbusGroupOptionNode[]
}

const form = reactive({
  groupId: '',
  name: '',
  code: '',
  unitId: 1,
  area: 'holding_register',
  address: 40001,
  addressBase: 'modicon',
  quantity: 1,
  dataType: 'uint16',
  byteOrder: 'ABCD',
  wordOrder: 'high_first',
  bitIndex: null as number | null,
  scale: 1,
  offset: 0,
  unit: '',
  pollIntervalMs: 1000,
  timeoutMs: null as number | null,
  retryCount: null as number | null,
  accessLevel: 'Read',
  description: '',
  status: 'active',
})

const groupOptions = computed(() => {
  const nodes = new Map<string, ModbusGroupOptionNode>()
  props.groups.forEach((group) => {
    nodes.set(group.id, { ...group, children: [] })
  })
  const roots: ModbusGroupOptionNode[] = []
  nodes.forEach((node) => {
    const parent = node.parentId ? nodes.get(node.parentId) : null
    if (parent) parent.children.push(node)
    else roots.push(node)
  })
  const sortNodes = (items: ModbusGroupOptionNode[]) => {
    items.sort(
      (left, right) =>
        (left.sortOrder || 0) - (right.sortOrder || 0) || left.name.localeCompare(right.name),
    )
    items.forEach((item) => sortNodes(item.children))
  }
  sortNodes(roots)
  const visit = (group: ModbusGroupOptionNode, depth: number) => [
    { id: group.id, label: group.name, depth },
    ...group.children.flatMap((child) => visit(child, depth + 1)),
  ]
  return roots.flatMap((group) => visit(group, 0))
})

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

watch(
  () => [props.modelValue, props.register, props.defaultGroupId],
  () => {
    if (!props.modelValue) return
    const item = props.register
    form.groupId = item?.groupId || props.defaultGroupId || ''
    form.name = item?.name || ''
    form.code = item?.code || ''
    const defaultSlave =
      slaves.value.find((slave) => slave.unitId === props.defaultUnitId) || slaves.value[0]
    form.unitId = item?.unitId ?? defaultSlave?.unitId ?? 1
    form.area = item?.area || 'holding_register'
    form.address = item?.address ?? 40001
    form.addressBase = item?.addressBase || 'modicon'
    form.quantity = item?.quantity ?? 1
    form.dataType = item?.dataType || 'uint16'
    form.byteOrder = item?.byteOrder || defaultSlave?.defaultByteOrder || 'ABCD'
    form.wordOrder = item?.wordOrder || defaultSlave?.defaultWordOrder || 'high_first'
    form.bitIndex = item?.bitIndex ?? null
    form.scale = item?.scale ?? 1
    form.offset = item?.offset ?? 0
    form.unit = item?.unit || ''
    form.pollIntervalMs = item?.pollIntervalMs ?? defaultSlave?.defaultPollIntervalMs ?? 1000
    form.timeoutMs = item?.timeoutMs ?? null
    form.retryCount = item?.retryCount ?? null
    form.accessLevel = normalizeAccessLevel(item?.accessLevel)
    form.description = item?.description || ''
    form.status = item?.status || 'active'
    form.bitIndex = isRegisterBitVariable(form.area, form.dataType) ? (form.bitIndex ?? 0) : null
    advancedPanels.value = []
    initialSnapshot.value = snapshotForm()
  },
  { immediate: true },
)

const isDirty = computed(() => props.modelValue && snapshotForm() !== initialSnapshot.value)
const bitIndexDisabled = computed(() => !isRegisterBitVariable(form.area, form.dataType))
const bitIndexPlaceholder = computed(() =>
  bitIndexDisabled.value ? '当前类型无需填写' : '默认 0，可填 0-15',
)

const submit = () => {
  sanitizeName()
  if (!form.name.trim()) {
    ElMessage.warning('请填写变量名')
    return
  }
  if (!isValidVariableName(form.name)) {
    ElMessage.warning('变量名称包含不支持的字符')
    return
  }
  emit('submit', {
    ...form,
    code: form.code.trim() || allocateVariableCode(form.name),
    groupId: form.groupId || null,
    unit: form.unit.trim() || null,
    description: form.description.trim() || null,
    hasGroupId: true,
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

function normalizeAccessLevel(value?: string | null) {
  return value === 'ReadWrite' || value === 'readwrite' || value === 'Write' || value === 'write'
    ? 'ReadWrite'
    : 'Read'
}

function applySelectedSlaveDefaults() {
  const slave = slaves.value.find((item) => item.unitId === form.unitId)
  if (!slave) return
  form.pollIntervalMs = slave.defaultPollIntervalMs || form.pollIntervalMs || 1000
  form.byteOrder = slave.defaultByteOrder || form.byteOrder || 'ABCD'
  form.wordOrder = slave.defaultWordOrder || form.wordOrder || 'high_first'
}

function applyAreaDefaults() {
  const isBitArea = form.area === 'coil' || form.area === 'discrete_input'
  form.dataType = isBitArea ? 'bool' : form.dataType === 'bool' ? 'uint16' : form.dataType
  form.quantity = quantityForDataType(form.dataType)
  form.bitIndex = isRegisterBitVariable(form.area, form.dataType) ? (form.bitIndex ?? 0) : null
  form.accessLevel =
    form.area === 'input_register' || form.area === 'discrete_input' ? 'Read' : form.accessLevel
}

function applyDataTypeDefaults() {
  form.quantity = quantityForDataType(form.dataType)
  form.bitIndex = isRegisterBitVariable(form.area, form.dataType) ? (form.bitIndex ?? 0) : null
}

function quantityForDataType(dataType: string) {
  const normalized = dataType.toLowerCase()
  if (['float64', 'double', 'int64', 'uint64'].includes(normalized)) return 4
  if (['float32', 'float', 'int32', 'uint32'].includes(normalized)) return 2
  return 1
}

function isRegisterBitVariable(area: string, dataType: string) {
  return (
    (area === 'holding_register' || area === 'input_register') &&
    ['bool', 'boolean'].includes(dataType.toLowerCase())
  )
}

function toVariableCode(value: string) {
  const normalized = value
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9_]+/g, '_')
    .replace(/^_+|_+$/g, '')
  return normalized || 'register'
}

function allocateVariableCode(value: string) {
  // code 是内部唯一标识，界面不开放编辑；这里基于当前列表预先避让重复，最终唯一性仍由后端约束兜底。
  const base = toVariableCode(value)
  const ignoredCode = props.mode === 'edit' ? props.register?.code : ''
  const usedCodes = new Set(
    (props.existingCodes || []).filter((code) => code && code !== ignoredCode),
  )
  if (!usedCodes.has(base)) return base
  for (let index = 2; index < 10000; index += 1) {
    const candidate = `${base}_${index}`
    if (!usedCodes.has(candidate)) return candidate
  }
  return `${base}_${Date.now()}`
}

function snapshotForm() {
  return JSON.stringify({ ...form })
}

defineExpose({ closeSilently })
</script>

<style scoped>
.modbus-register-dialog {
  display: grid;
  gap: 10px;
}

.modbus-register-dialog section {
  padding: 12px 12px 4px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: color-mix(in oklch, var(--dc-surface-subtle) 82%, transparent);
}

.modbus-register-dialog h3 {
  margin: 0 0 10px;
  display: flex;
  align-items: center;
  gap: 8px;
  color: var(--dc-text);
  font-size: 13px;
}

.modbus-register-dialog h3 span {
  width: 24px;
  height: 18px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 999px;
  background: color-mix(in oklch, var(--dc-primary) 14%, var(--dc-surface-raised));
  color: var(--dc-primary);
  font-size: 10px;
  font-weight: 800;
}

.modbus-register-dialog__grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 6px 12px;
}

.modbus-register-dialog__label,
.modbus-register-dialog__option {
  min-width: 0;
  display: inline-flex;
  align-items: center;
  gap: 5px;
}

.modbus-register-dialog__label svg {
  width: 14px;
  height: 14px;
  color: var(--dc-text-muted);
}

.modbus-register-dialog__option {
  width: 100%;
  justify-content: space-between;
}

.modbus-register-dialog__option em {
  min-width: 0;
  overflow: hidden;
  color: var(--dc-text-muted);
  font-size: 12px;
  font-style: normal;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.modbus-register-dialog :deep(.el-select),
.modbus-register-dialog :deep(.el-input-number) {
  width: 100%;
}
.modbus-register-dialog__group-option {
  min-width: 0;
  display: inline-flex;
  align-items: center;
  gap: 6px;
}
.modbus-register-dialog__group-guide {
  width: 10px;
  height: 1px;
  flex: 0 0 auto;
  background: var(--dc-border);
}

.modbus-register-dialog__advanced {
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
  overflow: hidden;
}

.modbus-register-dialog__advanced :deep(.el-collapse-item__header) {
  height: 38px;
  padding: 0 12px;
  border-bottom: 1px solid var(--dc-border);
  background: var(--dc-surface-subtle);
  color: var(--dc-text);
  font-size: 13px;
  font-weight: 700;
}

.modbus-register-dialog__advanced :deep(.el-collapse-item__wrap) {
  border-bottom: 0;
}

.modbus-register-dialog__advanced :deep(.el-collapse-item__content) {
  padding: 12px 12px 4px;
}

@media (max-width: 760px) {
  .modbus-register-dialog__grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
</style>
