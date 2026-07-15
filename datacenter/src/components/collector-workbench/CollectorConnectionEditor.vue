<template>
  <el-form
    v-if="connection && driver"
    v-loading="loading"
    label-position="top"
    class="collector-editor"
  >
    <div class="collector-editor__grid">
      <el-form-item label="连接名称"><el-input v-model="name" /></el-form-item>
      <el-form-item label="启用采集"><el-switch v-model="enabled" /></el-form-item>
    </div>
    <div class="collector-editor__driver">
      <span>驱动</span><strong>{{ driver.displayName }}</strong
      ><code>{{ connection.driverId }}@{{ connection.driverVersion }}</code>
    </div>
    <CollectorSchemaForm
      v-model="values"
      :schema="driver.connectionSchema"
      :ui-schema="driver.uiSchema"
    />
    <div class="collector-editor__actions">
      <el-button data-test="save-connection" type="primary" :loading="saving" @click="save"
        >保存配置</el-button
      >
    </div>
  </el-form>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { getCollectorDriver, updateCollectorConnection } from '@/api/collector.api'
import {
  splitCollectorFormValues,
  type CollectorConnection,
  type CollectorDriverDetail,
} from '@/api/schemas/collector.schema'
import CollectorSchemaForm from './CollectorSchemaForm.vue'

const props = defineProps<{ projectId: string; connection: CollectorConnection | null }>()
const emit = defineEmits<{ saved: [connection: CollectorConnection] }>()
const driver = ref<CollectorDriverDetail | null>(null)
const name = ref('')
const enabled = ref(true)
const values = ref<Record<string, unknown>>({})
const loading = ref(false)
const saving = ref(false)
watch(
  () => props.connection,
  async (connection) => {
    if (!connection) return
    loading.value = true
    try {
      driver.value = await getCollectorDriver(connection.driverId)
      name.value = connection.name
      enabled.value = connection.enabled
      values.value = { ...connection.config }
    } finally {
      loading.value = false
    }
  },
  { immediate: true },
)
async function save() {
  if (!props.connection || !driver.value) return
  saving.value = true
  try {
    const payload = splitCollectorFormValues(driver.value.connectionSchema, values.value)
    const result = await updateCollectorConnection(props.projectId, props.connection.id, {
      name: name.value,
      enabled: enabled.value,
      ...payload,
    })
    emit('saved', result)
    ElMessage.success('连接配置已保存')
  } finally {
    saving.value = false
  }
}
</script>

<style scoped>
.collector-editor {
  max-width: 760px;
  padding: 6px 4px 24px;
}
.collector-editor__grid {
  display: grid;
  grid-template-columns: 1fr 160px;
  gap: 18px;
}
.collector-editor__driver {
  display: grid;
  grid-template-columns: 80px 1fr auto;
  align-items: center;
  margin-bottom: 20px;
  padding: 14px 16px;
  border: 1px solid #dfe7eb;
  border-radius: 8px;
  background: #f7f9fa;
}
.collector-editor__driver span,
.collector-editor__driver code {
  color: #74818a;
  font-size: 12px;
}
.collector-editor__actions {
  display: flex;
  justify-content: flex-end;
  padding-top: 18px;
}
</style>
