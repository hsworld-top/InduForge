<template>
  <div class="mqtt-publish-tester">
    <header class="mqtt-publish-tester__header">
      <div class="mqtt-publish-tester__title">
        <IconTablerSend />
        <div>
          <strong>{{ ui('发布测试', 'Publish Test') }}</strong>
          <span>{{ subscription?.name || subscription?.topic || '-' }}</span>
        </div>
      </div>
      <WorkbenchStatusPill
        :label="connected ? ui('可发布', 'Ready') : ui('未连接', 'Disconnected')"
        :tone="connected ? 'success' : 'neutral'"
      />
    </header>

    <div class="mqtt-publish-tester__body">
      <el-alert
        v-if="!connected"
        type="warning"
        :closable="false"
        :title="ui('请先连接后再发布测试消息', 'Connect before publishing a test message')"
        show-icon
      />

      <section class="mqtt-publish-tester__form">
        <label>
          <span>Topic</span>
          <el-input v-model="topic" disabled placeholder="device/demo/1" />
        </label>

        <label>
          <span>QoS</span>
          <el-input-number v-model="qos" :min="0" :max="2" />
        </label>

        <label>
          <span>Payload JSON</span>
          <el-input v-model="payloadText" type="textarea" :rows="14" spellcheck="false" />
        </label>

        <el-alert
          v-if="errorMessage"
          type="error"
          :closable="false"
          :title="errorMessage"
          show-icon
        />

        <div class="mqtt-publish-tester__actions">
          <el-button :disabled="publishing" @click="resetPayload">{{ ui('重置', 'Reset') }}</el-button>
          <el-button
            type="primary"
            :loading="publishing"
            :disabled="!connected || !topic.trim()"
            @click="publishMessage"
          >
            {{ ui('发布', 'Publish') }}
          </el-button>
        </div>
      </section>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import dataAPI from '@/api/data.api'
import { getApiErrorMessage } from '@/utils/request'
import WorkbenchStatusPill from '@/components/workbench/WorkbenchStatusPill.vue'
import IconTablerSend from '~icons/tabler/send'
import { datacenterLocale } from '@/i18n/runtime'

const ui = (zh: string, en: string) => (datacenterLocale.value === 'en' ? en : zh)

const defaultPayload = () => JSON.stringify({ value: 1 }, null, 2)

const props = defineProps<{
  projectId: string
  connectionId: string
  subscription: {
    id: string
    name?: string
    topic?: string
    qos?: number
  }
  source: 'mqtt' | 'builtin-message'
  connected: boolean
}>()

const topic = ref('')
const qos = ref(0)
const payloadText = ref(defaultPayload())
const publishing = ref(false)
const errorMessage = ref('')

const resetForm = () => {
  topic.value = props.subscription?.topic || ''
  qos.value = props.subscription?.qos ?? 0
  payloadText.value = defaultPayload()
  errorMessage.value = ''
}

const resetPayload = () => {
  payloadText.value = defaultPayload()
  errorMessage.value = ''
}

const publishMessage = async () => {
  let payload: unknown
  try {
    payload = JSON.parse(payloadText.value)
  } catch {
    errorMessage.value = ui('Payload 必须是合法 JSON', 'Payload must be valid JSON')
    return
  }

  if (!props.connected) {
    errorMessage.value = ui('请先连接后再发布测试消息', 'Connect before publishing a test message')
    return
  }

  publishing.value = true
  errorMessage.value = ''
  try {
    const input = {
      topic: String(props.subscription?.topic || '').trim(),
      qos: qos.value,
      payload,
    }
    await dataAPI.publishMqttMessage(props.projectId, props.connectionId, input)
    ElMessage.success(ui('消息已发布', 'Message published'))
  } catch (error) {
    errorMessage.value = getApiErrorMessage(
      error,
      props.source === 'builtin-message' ? ui('发布 IF消息库消息失败', 'Failed to publish IF Message Store message') : ui('发布 MQTT 消息失败', 'Failed to publish MQTT message'),
    )
  } finally {
    publishing.value = false
  }
}

watch(
  () => [props.subscription?.id, props.subscription?.topic, props.subscription?.qos],
  resetForm,
  { immediate: true },
)
</script>

<style scoped>
.mqtt-publish-tester {
  height: 100%;
  min-height: 0;
  display: flex;
  flex-direction: column;
  background: var(--dc-surface-raised);
}

.mqtt-publish-tester__header {
  min-height: 48px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 8px 12px;
  border-bottom: 1px solid var(--dc-border);
  background: var(--dc-surface-subtle);
}

.mqtt-publish-tester__title {
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 9px;
}

.mqtt-publish-tester__title > svg {
  width: 17px;
  height: 17px;
  color: var(--dc-primary);
  flex-shrink: 0;
}

.mqtt-publish-tester__title div {
  min-width: 0;
  display: grid;
  gap: 2px;
}

.mqtt-publish-tester__title strong,
.mqtt-publish-tester__title span {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.mqtt-publish-tester__title strong {
  color: var(--dc-text);
  font-size: 13px;
}

.mqtt-publish-tester__title span {
  color: var(--dc-text-muted);
  font-family: var(--dc-font-mono, monospace);
  font-size: 11px;
}

.mqtt-publish-tester__body {
  min-height: 0;
  flex: 1;
  overflow: auto;
  padding: 12px;
}

.mqtt-publish-tester__form {
  max-width: 760px;
  display: grid;
  gap: 12px;
}

.mqtt-publish-tester__form label {
  display: grid;
  gap: 6px;
  color: var(--dc-text-secondary);
  font-size: 12px;
  font-weight: 700;
}

.mqtt-publish-tester__actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}
</style>
