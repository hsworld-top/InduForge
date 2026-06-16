<template>
  <DcDialog
    ref="dialogRef"
    v-model="visible"
    :title="mode === 'edit' ? '编辑 Modbus 变量' : '新建 Modbus 变量'"
    width="820px"
    body-max-height="calc(100vh - 180px)"
    :dirty="isDirty"
    :close-disabled="loading"
  >
    <el-form class="modbus-register-dialog" label-position="top">
      <section>
        <h3><span>01</span>基础信息</h3>
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
        </div>
      </section>
      <section>
        <h3><span>02</span>Modbus 地址</h3>
        <div class="modbus-register-dialog__grid">
          <el-form-item label="从站地址"
            ><el-input-number v-model="form.unitId" :min="0" :max="247"
          /></el-form-item>
          <el-form-item label="区域">
            <el-select v-model="form.area">
              <el-option label="Holding Register" value="holding_register" />
              <el-option label="Input Register" value="input_register" />
              <el-option label="Coil" value="coil" />
              <el-option label="Discrete Input" value="discrete_input" />
            </el-select>
          </el-form-item>
          <el-form-item label="用户地址"
            ><el-input-number v-model="form.address" :min="0"
          /></el-form-item>
          <el-form-item label="地址基准">
            <el-select v-model="form.addressBase">
              <el-option label="Modicon 4xxxx/3xxxx" value="modicon" />
              <el-option label="1-based" value="one_based" />
              <el-option label="0-based" value="zero_based" />
            </el-select>
          </el-form-item>
        </div>
      </section>
      <section>
        <h3><span>03</span>解析规则</h3>
        <div class="modbus-register-dialog__grid">
          <el-form-item label="数据类型">
            <el-select v-model="form.dataType">
              <el-option label="bool" value="bool" />
              <el-option label="uint16" value="uint16" />
              <el-option label="int16" value="int16" />
              <el-option label="uint32" value="uint32" />
              <el-option label="int32" value="int32" />
              <el-option label="float32" value="float32" />
              <el-option label="float64" value="float64" />
            </el-select>
          </el-form-item>
          <el-form-item label="寄存器数量"
            ><el-input-number v-model="form.quantity" :min="1"
          /></el-form-item>
          <el-form-item label="字节序">
            <el-select v-model="form.byteOrder">
              <el-option label="ABCD" value="ABCD" />
              <el-option label="BADC" value="BADC" />
              <el-option label="CDAB" value="CDAB" />
              <el-option label="DCBA" value="DCBA" />
            </el-select>
          </el-form-item>
          <el-form-item label="字序">
            <el-select v-model="form.wordOrder">
              <el-option label="High Word First" value="high_first" />
              <el-option label="Low Word First" value="low_first" />
            </el-select>
          </el-form-item>
          <el-form-item label="Bit 位"
            ><el-input-number v-model="form.bitIndex" :min="0" :max="15"
          /></el-form-item>
          <el-form-item label="倍率"><el-input-number v-model="form.scale" /></el-form-item>
          <el-form-item label="偏移"><el-input-number v-model="form.offset" /></el-form-item>
          <el-form-item label="单位"><el-input v-model="form.unit" /></el-form-item>
        </div>
      </section>
      <section>
        <h3><span>04</span>采样与权限</h3>
        <div class="modbus-register-dialog__grid">
          <el-form-item label="轮询周期(ms)"
            ><el-input-number v-model="form.pollIntervalMs" :min="100"
          /></el-form-item>
          <el-form-item label="超时(ms)"
            ><el-input-number v-model="form.timeoutMs" :min="1"
          /></el-form-item>
          <el-form-item label="重试次数"
            ><el-input-number v-model="form.retryCount" :min="0"
          /></el-form-item>
          <el-form-item label="读写权限">
            <el-select v-model="form.accessLevel">
              <el-option label="只读" value="Read" />
              <el-option label="读写" value="ReadWrite" />
            </el-select>
          </el-form-item>
        </div>
      </section>
      <el-form-item label="说明"
        ><el-input v-model="form.description" type="textarea" :rows="2"
      /></el-form-item>
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
import type { ModbusRegister, ModbusRegisterGroup } from './types'

const props = defineProps<{
  modelValue: boolean
  mode: 'create' | 'edit'
  groups: ModbusRegisterGroup[]
  register?: ModbusRegister | null
  defaultGroupId?: string
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
const invalidVariableNamePattern = /[^\u4e00-\u9fa5A-Za-z0-9_$#%@+()[\]&-]/g
const validVariableNamePattern = /^[\u4e00-\u9fa5A-Za-z0-9_$#%@+()[\]&-]+$/
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

watch(
  () => [props.modelValue, props.register, props.defaultGroupId],
  () => {
    if (!props.modelValue) return
    const item = props.register
    form.groupId = item?.groupId || props.defaultGroupId || ''
    form.name = item?.name || ''
    form.code = item?.code || ''
    form.unitId = item?.unitId ?? 1
    form.area = item?.area || 'holding_register'
    form.address = item?.address ?? 40001
    form.addressBase = item?.addressBase || 'modicon'
    form.quantity = item?.quantity ?? 1
    form.dataType = item?.dataType || 'uint16'
    form.byteOrder = item?.byteOrder || 'ABCD'
    form.wordOrder = item?.wordOrder || 'high_first'
    form.bitIndex = item?.bitIndex ?? null
    form.scale = item?.scale ?? 1
    form.offset = item?.offset ?? 0
    form.unit = item?.unit || ''
    form.pollIntervalMs = item?.pollIntervalMs ?? 1000
    form.timeoutMs = item?.timeoutMs ?? null
    form.retryCount = item?.retryCount ?? null
    form.accessLevel = normalizeAccessLevel(item?.accessLevel)
    form.description = item?.description || ''
    form.status = item?.status || 'active'
    initialSnapshot.value = snapshotForm()
  },
  { immediate: true },
)

const isDirty = computed(() => props.modelValue && snapshotForm() !== initialSnapshot.value)

const submit = () => {
  sanitizeName()
  if (!form.name.trim()) {
    ElMessage.warning('请填写变量名')
    return
  }
  if (!validVariableNamePattern.test(form.name)) {
    ElMessage.warning('变量名称包含不支持的字符')
    return
  }
  emit('submit', {
    ...form,
    code: form.code.trim() || toVariableCode(form.name),
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

function normalizeAccessLevel(value?: string | null) {
  return value === 'ReadWrite' || value === 'readwrite' || value === 'Write' || value === 'write'
    ? 'ReadWrite'
    : 'Read'
}

function toVariableCode(value: string) {
  const normalized = value
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9_]+/g, '_')
    .replace(/^_+|_+$/g, '')
  return normalized || 'register'
}

function snapshotForm() {
  return JSON.stringify({ ...form })
}

defineExpose({ closeSilently })
</script>

<style scoped>
.modbus-register-dialog {
  display: grid;
  gap: 12px;
}

.modbus-register-dialog section {
  padding: 10px 12px 4px;
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
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 6px 12px;
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
</style>
