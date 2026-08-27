<template>
  <el-form
    v-if="connection && driver"
    v-loading="loading"
    label-position="top"
    class="collector-editor"
  >
    <div class="collector-editor__basic">
      <el-form-item label="连接名称">
        <el-input v-model="name" maxlength="50" placeholder="仅支持文字、数字和空格" />
      </el-form-item>
      <el-form-item label="连接编码">
        <el-input :model-value="connection.code" disabled />
        <span class="collector-editor__hint">创建时自动生成，修改连接名称不会改变编码</span>
      </el-form-item>
    </div>
    <CollectorSchemaForm
      v-model="values"
      :schema="driver.connectionSchema"
      :ui-schema="driver.uiSchema"
      :string-field-options="stringFieldOptions"
      @refresh-options="emit('refresh-agent')"
    />
    <section class="collector-editor__acquisition">
      <div class="collector-editor__section-title">
        <div>
          <strong>默认采集参数</strong>
          <p>新增变量自动继承；变量可以只覆盖有差异的字段。</p>
        </div>
        <el-switch v-model="isEnabled" active-text="启用连接" />
      </div>
      <div class="collector-editor__acquisition-grid">
        <el-form-item label="采集周期 (ms)"
          ><el-input-number v-model="defaultAcquisition.intervalMs" :min="1"
        /></el-form-item>
        <el-form-item label="读取超时 (ms)"
          ><el-input-number v-model="defaultAcquisition.timeoutMs" :min="1"
        /></el-form-item>
        <el-form-item label="失败重试"
          ><el-input-number v-model="defaultAcquisition.retryCount" :min="0"
        /></el-form-item>
        <el-form-item label="数值死区"
          ><el-input-number v-model="defaultAcquisition.deadband" :min="0"
        /></el-form-item>
        <el-form-item label="优先级"
          ><el-input-number v-model="defaultAcquisition.priority" :min="0"
        /></el-form-item>
        <el-form-item label="仅变化时采集"
          ><el-switch v-model="defaultAcquisition.changeOnly"
        /></el-form-item>
      </div>
    </section>
    <div class="collector-editor__actions">
      <el-button data-test="save-connection" type="primary" :loading="saving" @click="save"
        >保存配置</el-button
      >
    </div>
  </el-form>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { getCollectorDriver, updateCollectorConnection } from '@/api/collector.api'
import { getApiErrorMessage } from '@/utils/request'
import {
  splitCollectorFormValues,
  type CollectorConnection,
  type CollectorDriverDetail,
} from '@/api/schemas/collector.schema'
import type { CollectorAgent } from '@/api/schemas/collector-dev.schema'
import CollectorSchemaForm from './CollectorSchemaForm.vue'
import { validateCollectorConnectionName } from './collector-workbench-model'

const props = defineProps<{
  projectId: string
  connection: CollectorConnection | null
  agent?: CollectorAgent
}>()
const emit = defineEmits<{
  saved: [connection: CollectorConnection]
  'refresh-agent': []
}>()
const driver = ref<CollectorDriverDetail | null>(null)
const name = ref('')
const values = ref<Record<string, unknown>>({})
const loading = ref(false)
const saving = ref(false)
const isEnabled = ref(true)
const defaultAcquisition = ref({
  intervalMs: 1000,
  timeoutMs: 3000,
  retryCount: 0,
  deadband: 0,
  changeOnly: false,
  priority: 0,
})
const stringFieldOptions = computed<Record<string, string[]>>(() => {
  if (!driver.value?.connectionSchema.properties?.portName) return {}
  const capability = props.agent?.capabilities.find(
    (item) =>
      item.driverId === driver.value?.driverId &&
      item.driverVersion === driver.value?.driverVersion &&
      item.schemaVersions.includes(driver.value?.schemaVersion || 0),
  )
  return { portName: capability?.resources?.serialPorts || [] }
})
watch(
  () => props.connection,
  async (connection) => {
    if (!connection) return
    loading.value = true
    try {
      driver.value = await getCollectorDriver(connection.driverId)
      name.value = connection.name
      values.value = { ...connection.config }
      isEnabled.value = connection.isEnabled
      defaultAcquisition.value = {
        intervalMs: Number(connection.defaultAcquisition.intervalMs) || 1000,
        timeoutMs: Number(connection.defaultAcquisition.timeoutMs) || 3000,
        retryCount: Number(connection.defaultAcquisition.retryCount) || 0,
        deadband: Number(connection.defaultAcquisition.deadband) || 0,
        changeOnly: Boolean(connection.defaultAcquisition.changeOnly),
        priority: Number(connection.defaultAcquisition.priority) || 0,
      }
    } finally {
      loading.value = false
    }
  },
  { immediate: true },
)
async function save() {
  if (!props.connection || !driver.value) return
  const validatedName = validateCollectorConnectionName(name.value)
  if (validatedName.error) {
    ElMessage.warning(validatedName.error)
    return
  }
  saving.value = true
  try {
    const payload = splitCollectorFormValues(driver.value.connectionSchema, values.value)
    const result = await updateCollectorConnection(props.projectId, props.connection.id, {
      name: validatedName.name,
      isEnabled: isEnabled.value,
      defaultAcquisition: defaultAcquisition.value,
      ...payload,
    })
    emit('saved', result)
    ElMessage.success('连接配置已保存')
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '连接配置保存失败'))
  } finally {
    saving.value = false
  }
}
</script>

<style scoped>
.collector-editor {
  width: min(100%, 1180px);
  margin: 0 auto;
  padding: 6px 4px 24px;
}
.collector-editor__basic {
  width: min(100%, 560px);
  margin-bottom: 4px;
}
.collector-editor__hint {
  margin-top: 6px;
  color: var(--dc-text-muted);
  font-size: 12px;
}
.collector-editor__actions {
  position: sticky;
  bottom: -16px;
  z-index: 2;
  display: flex;
  justify-content: flex-end;
  margin: 8px -4px -24px;
  padding: 14px 4px 16px;
  border-top: 1px solid var(--dc-border);
  background: color-mix(in srgb, var(--dc-surface-raised) 94%, transparent);
  backdrop-filter: blur(8px);
}
.collector-editor__acquisition {
  margin-top: 20px;
  padding-top: 18px;
  border-top: 1px solid var(--dc-border);
}
.collector-editor__section-title {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 20px;
  margin-bottom: 14px;
}
.collector-editor__section-title p {
  margin: 4px 0 0;
  color: var(--dc-text-muted);
  font-size: 12px;
}
.collector-editor__acquisition-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 0 16px;
}
.collector-editor__acquisition-grid :deep(.el-input-number) {
  width: 100%;
}
@media (max-width: 820px) {
  .collector-editor__acquisition-grid {
    grid-template-columns: 1fr 1fr;
  }
}
</style>
