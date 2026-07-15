<template>
  <el-dialog
    :model-value="modelValue"
    title="新增工业采集连接"
    width="720px"
    destroy-on-close
    @close="emit('update:modelValue', false)"
  >
    <el-steps :active="step" finish-status="success" align-center>
      <el-step title="协议族" />
      <el-step title="驱动" />
      <el-step title="连接配置" />
    </el-steps>

    <div class="collector-wizard__body">
      <div v-if="step === 0" class="collector-wizard__choices">
        <button
          v-for="family in protocolFamilies"
          :key="family"
          type="button"
          :class="['collector-wizard__choice', { 'is-active': family === protocolFamily }]"
          @click="protocolFamily = family"
        >
          <strong>{{ family.toUpperCase() }}</strong
          ><span>选择该协议族的认证驱动</span>
        </button>
      </div>
      <div v-else-if="step === 1" class="collector-wizard__choices">
        <button
          v-for="driver in filteredDrivers"
          :key="driver.driverId"
          type="button"
          :class="['collector-wizard__choice', { 'is-active': driver.driverId === driverId }]"
          @click="selectDriver(driver.driverId)"
        >
          <strong>{{ driver.displayName }}</strong
          ><span>{{ driver.driverId }} · {{ driver.driverVersion }}</span>
        </button>
      </div>
      <el-form v-else label-position="top">
        <el-form-item label="连接名称" required><el-input v-model="name" /></el-form-item>
        <CollectorSchemaForm
          v-if="driverDetail"
          v-model="values"
          :schema="driverDetail.connectionSchema"
          :ui-schema="driverDetail.uiSchema"
        />
      </el-form>
    </div>

    <template #footer>
      <el-button @click="emit('update:modelValue', false)">取消</el-button>
      <el-button v-if="step > 0" @click="step--">上一步</el-button>
      <el-button v-if="step < 2" type="primary" :disabled="!canContinue" @click="step++"
        >下一步</el-button
      >
      <el-button
        v-else
        data-test="save-connection"
        type="primary"
        :loading="saving"
        :disabled="!name.trim() || !driverDetail"
        @click="save"
        >创建连接</el-button
      >
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import {
  createCollectorConnection,
  getCollectorDriver,
  listCollectorDrivers,
} from '@/api/collector.api'
import {
  splitCollectorFormValues,
  type CollectorDriverDetail,
  type CollectorDriverSummary,
} from '@/api/schemas/collector.schema'
import CollectorSchemaForm from './CollectorSchemaForm.vue'

const props = defineProps<{ modelValue: boolean; projectId: string }>()
const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  created: [connectionId: string]
}>()
const step = ref(0)
const drivers = ref<CollectorDriverSummary[]>([])
const protocolFamily = ref('')
const driverId = ref('')
const driverDetail = ref<CollectorDriverDetail | null>(null)
const name = ref('')
const values = ref<Record<string, unknown>>({})
const saving = ref(false)

const protocolFamilies = computed(() => [
  ...new Set(drivers.value.map((driver) => driver.protocolFamily)),
])
const filteredDrivers = computed(() =>
  drivers.value.filter((driver) => driver.protocolFamily === protocolFamily.value),
)
const canContinue = computed(() =>
  step.value === 0 ? Boolean(protocolFamily.value) : Boolean(driverId.value),
)

watch(
  () => props.modelValue,
  async (visible) => {
    if (!visible) return
    step.value = 0
    protocolFamily.value = ''
    driverId.value = ''
    driverDetail.value = null
    name.value = ''
    values.value = {}
    drivers.value = (await listCollectorDrivers({ page: 1, pageSize: 100 })).list
  },
)

async function selectDriver(value: string) {
  driverId.value = value
  driverDetail.value = await getCollectorDriver(value)
  values.value = Object.fromEntries(
    Object.entries(driverDetail.value.connectionSchema.properties || {})
      .filter(([, property]) => property.default !== undefined)
      .map(([key, property]) => [key, property.default]),
  )
}

async function save() {
  if (!driverDetail.value) return
  saving.value = true
  try {
    const payload = splitCollectorFormValues(driverDetail.value.connectionSchema, values.value)
    const connection = await createCollectorConnection(props.projectId, {
      name: name.value.trim(),
      driverId: driverDetail.value.driverId,
      ...payload,
      metadata: {},
    })
    emit('created', connection.id)
    emit('update:modelValue', false)
    ElMessage.success('工业采集连接已创建')
  } finally {
    saving.value = false
  }
}
</script>

<style scoped>
.collector-wizard__body {
  min-height: 320px;
  padding: 32px 4px 4px;
}
.collector-wizard__choices {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 14px;
}
.collector-wizard__choice {
  display: flex;
  min-height: 96px;
  flex-direction: column;
  align-items: flex-start;
  justify-content: center;
  gap: 8px;
  padding: 18px;
  border: 1px solid #dce3e8;
  border-radius: 10px;
  background: #fff;
  color: #24313c;
  cursor: pointer;
  text-align: left;
  transition: 160ms ease;
}
.collector-wizard__choice:hover,
.collector-wizard__choice.is-active {
  border-color: #1677a5;
  box-shadow: 0 8px 24px rgb(21 88 120 / 12%);
  transform: translateY(-1px);
}
.collector-wizard__choice span {
  color: #788691;
  font-size: 12px;
}
</style>
