<template>
  <el-form
    v-if="connection && driver"
    v-loading="loading"
    label-position="top"
    class="collector-editor"
  >
    <div class="collector-editor__basic">
      <el-form-item :label="ui('连接名称', 'Connection Name')">
        <el-input v-model="name" maxlength="50" :placeholder="ui('例如 1号产线 OPC UA', 'For example, Line 1 OPC UA')" />
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
          <strong>{{ ui('默认采集参数', 'Default Acquisition') }}</strong>
          <p>{{ ui('作为该连接下变量的默认采集参数。', 'Applied as the default acquisition settings for points in this connection.') }}</p>
        </div>
        <el-switch v-model="isEnabled" :active-text="ui('启用连接', 'Enable Connection')" />
      </div>
      <div class="collector-editor__acquisition-grid">
        <el-form-item :label="ui('采集周期（毫秒）', 'Interval (ms)')"
          ><el-input-number v-model="defaultAcquisition.intervalMs" :min="1"
        /></el-form-item>
        <el-form-item :label="ui('数值死区', 'Numeric Deadband')"
          ><el-input-number v-model="defaultAcquisition.deadband" :min="0"
        /></el-form-item>
        <el-form-item :label="ui('仅变化时上报', 'Report Changes Only')"
          ><el-switch v-model="defaultAcquisition.changeOnly"
        /></el-form-item>
      </div>
    </section>
    <div class="collector-editor__actions">
      <el-button data-test="save-connection" type="primary" :loading="saving" @click="save"
        >{{ ui('保存配置', 'Save Configuration') }}</el-button
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
import { datacenterLocale } from '@/i18n/runtime'

const ui = (zh: string, en: string) => (datacenterLocale.value === 'en' ? en : zh)

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
  deadband: 0,
  changeOnly: false,
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
        deadband: Number(connection.defaultAcquisition.deadband) || 0,
        changeOnly: Boolean(connection.defaultAcquisition.changeOnly),
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
    ElMessage.success(ui('连接配置已保存', 'Connection configuration saved'))
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, ui('连接配置保存失败', 'Failed to save the connection configuration')))
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
