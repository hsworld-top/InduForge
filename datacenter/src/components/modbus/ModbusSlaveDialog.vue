<template>
  <el-dialog
    v-model="visible"
    :title="mode === 'edit' ? '编辑 Modbus 从站' : '新建 Modbus 从站'"
    width="560px"
    append-to-body
    destroy-on-close
  >
    <el-form class="modbus-slave-dialog" label-position="top">
      <div class="modbus-slave-dialog__grid is-three">
        <el-form-item label="从站名称">
          <el-input v-model="form.name" placeholder="例如：电表 A" />
        </el-form-item>
        <el-form-item label="从站地址">
          <el-input-number v-model="form.unitId" :min="0" :max="247" class="w-full" />
        </el-form-item>
      </div>
      <div class="modbus-slave-dialog__grid">
        <el-form-item label="启用状态">
          <el-switch v-model="form.enabled" active-text="启用" inactive-text="停用" />
        </el-form-item>
        <el-form-item label="默认轮询周期">
          <el-input-number
            v-model="form.defaultPollIntervalMs"
            :min="100"
            :step="100"
            class="w-full"
          />
        </el-form-item>
      </div>
      <div class="modbus-slave-dialog__grid">
        <el-form-item label="默认字节序">
          <el-select v-model="form.defaultByteOrder" class="w-full">
            <el-option label="ABCD" value="ABCD" />
            <el-option label="BADC" value="BADC" />
            <el-option label="CDAB" value="CDAB" />
            <el-option label="DCBA" value="DCBA" />
          </el-select>
        </el-form-item>
        <el-form-item label="默认字序">
          <el-select v-model="form.defaultWordOrder" class="w-full">
            <el-option label="高字在前" value="high_first" />
            <el-option label="低字在前" value="low_first" />
          </el-select>
        </el-form-item>
      </div>
      <div class="modbus-slave-dialog__grid">
        <el-form-item label="请求间隔">
          <el-input-number
            v-model="form.requestIntervalMs"
            :min="0"
            :step="50"
            placeholder="默认不额外等待"
            class="w-full"
          />
        </el-form-item>
        <el-form-item label="请求超时">
          <el-input-number
            v-model="form.timeoutMs"
            :min="1"
            :step="1000"
            placeholder="默认跟随连接"
            class="w-full"
          />
        </el-form-item>
        <el-form-item label="重试次数">
          <el-input-number
            v-model="form.retryCount"
            :min="0"
            :max="10"
            placeholder="默认不重试"
            class="w-full"
          />
        </el-form-item>
      </div>
      <el-form-item label="描述">
        <el-input v-model="form.description" type="textarea" :rows="3" resize="none" />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="visible = false">取消</el-button>
      <el-button type="primary" :loading="loading" @click="submit">保存</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { reactive, watch } from 'vue'
import { ElMessage } from 'element-plus'
import type { ModbusSlaveDevice } from './types'

const visible = defineModel<boolean>({ default: false })
const props = defineProps<{
  mode: 'create' | 'edit'
  slave?: ModbusSlaveDevice | null
  loading?: boolean
  defaultUnitId?: number
}>()

const emit = defineEmits<{ (event: 'submit', payload: Record<string, unknown>): void }>()

const form = reactive({
  name: '',
  unitId: 1,
  enabled: true,
  defaultPollIntervalMs: 1000,
  defaultByteOrder: 'ABCD',
  defaultWordOrder: 'high_first',
  requestIntervalMs: null as number | null,
  timeoutMs: null as number | null,
  retryCount: null as number | null,
  description: '',
  sortOrder: 0,
})

const reset = () => {
  const slave = props.slave
  form.name = slave?.name || ''
  form.unitId = slave?.unitId ?? props.defaultUnitId ?? 1
  form.enabled = slave?.enabled ?? true
  form.defaultPollIntervalMs = slave?.defaultPollIntervalMs ?? 1000
  form.defaultByteOrder = slave?.defaultByteOrder || 'ABCD'
  form.defaultWordOrder = slave?.defaultWordOrder || 'high_first'
  form.requestIntervalMs = slave?.requestIntervalMs ?? null
  form.timeoutMs = slave?.timeoutMs ?? null
  form.retryCount = slave?.retryCount ?? null
  form.description = slave?.description || ''
  form.sortOrder = slave?.sortOrder ?? form.unitId
}

const submit = () => {
  if (!String(form.name || '').trim()) {
    ElMessage.warning('请填写从站名称')
    return
  }
  emit('submit', {
    name: form.name.trim(),
    unitId: form.unitId,
    enabled: form.enabled,
    defaultPollIntervalMs: form.defaultPollIntervalMs,
    defaultByteOrder: form.defaultByteOrder,
    defaultWordOrder: form.defaultWordOrder,
    requestIntervalMs: form.requestIntervalMs || null,
    timeoutMs: form.timeoutMs || null,
    retryCount: form.retryCount || null,
    description: form.description.trim() || null,
    sortOrder: form.sortOrder,
  })
}

watch(visible, (next) => {
  if (next) reset()
})
</script>

<style scoped>
.modbus-slave-dialog {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.modbus-slave-dialog__grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}

.modbus-slave-dialog__grid.is-three {
  grid-template-columns: repeat(3, minmax(0, 1fr));
}

.modbus-slave-dialog :deep(.el-select),
.modbus-slave-dialog :deep(.el-input-number) {
  width: 100%;
}
</style>
