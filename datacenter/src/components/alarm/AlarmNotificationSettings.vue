<template>
  <div class="alarm-notifications">
    <section class="alarm-notifications__settings">
      <header class="alarm-notifications__head">
        <h3>{{ t('alarmNotifications.defaults') }}</h3>
        <button
          type="button"
          class="alarm-notifications__primary"
          :disabled="saving"
          @click="saveSettings"
        >
          <IconTablerDeviceFloppy />
          {{ t('alarmNotifications.saveSettings') }}
        </button>
      </header>
      <div v-loading="loading" class="alarm-notifications__form">
        <div class="alarm-notifications__checks">
          <el-checkbox v-model="settings.notifyOnRaise">{{ t('alarmNotifications.notifyRaise') }}</el-checkbox>
          <el-checkbox v-model="settings.notifyOnClear">{{ t('alarmNotifications.notifyClear') }}</el-checkbox>
        </div>
        <label>
          <span>{{ t('alarmNotifications.repeat') }}</span>
          <div class="alarm-notifications__repeat">
            <el-switch v-model="repeatEnabled" />
            <el-input-number
              v-if="repeatEnabled"
              v-model="settings.repeatIntervalSeconds"
              :min="1"
              controls-position="right"
            />
            <span v-if="repeatEnabled">{{ t('common.seconds') }}</span>
          </div>
        </label>
        <label>
          <span>{{ t('alarmNotifications.defaultChannels') }}</span>
          <el-select v-model="settings.defaultChannelIds" multiple :placeholder="t('alarmNotifications.selectChannels')">
            <el-option
              v-for="channel in enabledChannels"
              :key="channel.id"
              :label="channelName(channel)"
              :value="channel.id"
            />
          </el-select>
        </label>
        <label class="is-wide">
          <span>{{ t('alarmNotifications.defaultTemplate') }}</span>
          <el-input v-model="settings.defaultMessageTemplate" type="textarea" :rows="3" />
          <div class="alarm-notifications__variables">
            <button
              v-for="variable in templateVariables"
              :key="variable.value"
              type="button"
              @click="insertVariable(variable.value)"
            >
              {{ variable.label }}
            </button>
          </div>
        </label>
      </div>
    </section>

    <section class="alarm-notifications__channels">
      <header class="alarm-notifications__head">
        <h3>{{ t('alarmNotifications.channels') }}</h3>
        <button type="button" class="alarm-notifications__secondary" @click="openCreate">
          <IconTablerPlus />{{ t('alarmNotifications.addChannel') }}
        </button>
      </header>
      <el-table v-loading="channelsLoading" :data="channels" class="alarm-notifications__table">
        <el-table-column :label="t('alarmNotifications.name')" min-width="180">
          <template #default="{ row }">{{ channelName(row) }}</template>
        </el-table-column>
        <el-table-column :label="t('alarmNotifications.type')" width="130">
          <template #default="{ row }">{{ channelTypeLabel(row.channelType) }}</template>
        </el-table-column>
        <el-table-column :label="t('alarmNotifications.secret')" min-width="150">
          <template #default="{ row }">{{ secretSummary(row) }}</template>
        </el-table-column>
        <el-table-column :label="t('alarmNotifications.lastTest')" min-width="180">
          <template #default="{ row }">
            <el-tooltip
              :content="row.lastTestMessage || testStatusText(row.lastTestStatus)"
              placement="top"
            >
              <span
                ><StatusBadge
                  :tone="testStatusTone(row.lastTestStatus)"
                  :text="testStatusText(row.lastTestStatus)"
              /></span>
            </el-tooltip>
          </template>
        </el-table-column>
        <el-table-column :label="t('alarmNotifications.status')" width="100">
          <template #default="{ row }"
            ><StatusBadge
              :tone="row.isEnabled ? 'success' : 'muted'"
              :text="row.isEnabled ? t('alarmNotifications.enabled') : t('alarmNotifications.disabled')"
          /></template>
        </el-table-column>
        <el-table-column :label="t('alarmNotifications.actions')" width="132" align="right">
          <template #default="{ row }">
            <button
              v-if="row.id !== 'runtime_inapp'"
              type="button"
              class="alarm-notifications__icon"
              :title="t('alarmNotifications.sendTest')"
              :disabled="channelTestingId === row.id"
              @click="runChannelTest(row)"
            >
              <IconTablerSend />
            </button>
            <button
              v-if="row.id !== 'runtime_inapp'"
              type="button"
              class="alarm-notifications__icon"
              :title="t('alarmNotifications.editChannel')"
              @click="openEdit(row)"
            >
              <IconTablerEdit />
            </button>
            <button
              v-if="row.id !== 'runtime_inapp'"
              type="button"
              class="alarm-notifications__icon is-danger"
              :title="t('alarmNotifications.deleteChannel')"
              @click="removeChannel(row)"
            >
              <IconTablerTrash />
            </button>
          </template>
        </el-table-column>
        <template #empty>
          <el-empty :image-size="48" :description="t('alarmNotifications.emptyChannels')" />
        </template>
      </el-table>
    </section>

    <DcDialog
      v-model="dialogVisible"
      :title="editingChannel ? t('alarmNotifications.editChannel') : t('alarmNotifications.createChannel')"
      width="520px"
    >
      <div class="alarm-channel-form">
        <label
          ><span>{{ t('alarmNotifications.name') }}</span><el-input v-model="channelDraft.name" :placeholder="t('alarmNotifications.nameExample')"
        /></label>
        <label
          ><span>{{ t('alarmNotifications.type') }}</span
          ><el-select v-model="channelDraft.channelType" :disabled="!!editingChannel"
            ><el-option label="Webhook" value="webhook" /><el-option
              :label="t('alarmNotifications.channelTypes.dingtalk')"
              value="dingtalk" /><el-option :label="t('alarmNotifications.channelTypes.wecom')" value="wecom" /></el-select
        ></label>
        <label class="is-wide"
          ><span>{{ t('alarmNotifications.webhookUrl') }}</span><el-input v-model="webhookUrl" placeholder="https://..."
        /></label>
        <label class="is-wide"
          ><span>{{ t('alarmNotifications.signingSecret') }}</span
          ><el-input
            v-model="secretValue"
            type="password"
            show-password
            autocomplete="new-password"
        /></label>
        <label><span>{{ t('alarmNotifications.enabled') }}</span><el-switch v-model="channelDraft.isEnabled" /></label>
      </div>
      <template #footer>
        <button type="button" class="alarm-notifications__secondary" @click="dialogVisible = false">
          {{ t('alarmNotifications.cancel') }}
        </button>
        <button
          type="button"
          class="alarm-notifications__primary"
          :disabled="channelSaving"
          @click="saveChannel"
        >
          {{ t('alarmNotifications.save') }}
        </button>
      </template>
    </DcDialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import IconTablerDeviceFloppy from '~icons/tabler/device-floppy'
import IconTablerEdit from '~icons/tabler/edit'
import IconTablerPlus from '~icons/tabler/plus'
import IconTablerSend from '~icons/tabler/send'
import IconTablerTrash from '~icons/tabler/trash'
import DcDialog from '@/components/shared/DcDialog.vue'
import StatusBadge from '@/components/shared/StatusBadge.vue'
import {
  createAlarmChannel,
  deleteAlarmChannel,
  getAlarmSettings,
  listAlarmChannels,
  saveAlarmSettings,
  updateAlarmChannel,
  testAlarmChannel,
} from '@/api/alarm.api'
import type {
  AlarmNotificationChannel,
  AlarmNotificationChannelSave,
} from '@/api/schemas/alarm.schema'
import { getApiErrorMessage } from '@/utils/request'
import { useConfirm } from '@/composables/useConfirm'
import { ensureBuiltinAlarmNotificationChannel } from '@/models/alarm-item'
import { datacenterLocale, t } from '@/i18n/runtime'

const props = defineProps<{ projectId: string }>()
const emit = defineEmits<{ changed: [channels: AlarmNotificationChannel[]] }>()
const { confirm } = useConfirm()
const loading = ref(false)
const saving = ref(false)
const channelsLoading = ref(false)
const channels = ref<AlarmNotificationChannel[]>([])
const repeatEnabled = ref(false)
const settings = reactive({
  notifyOnRaise: true,
  notifyOnClear: true,
  repeatIntervalSeconds: 300 as number | null,
  defaultMessageTemplate:
    datacenterLocale.value === 'en'
      ? '{{pointName}} current value {{value}}, triggered {{conditionLabel}}'
      : '{{pointName}} 当前值 {{value}}，触发 {{conditionLabel}}',
  defaultChannelIds: ['runtime_inapp'] as string[],
})
const dialogVisible = ref(false)
const editingChannel = ref<AlarmNotificationChannel | null>(null)
const channelSaving = ref(false)
const channelTestingId = ref('')
const channelDraft = reactive<AlarmNotificationChannelSave>({
  name: '',
  channelType: 'webhook',
  config: {},
  secrets: {},
  deleteSecretKeys: [],
  isEnabled: true,
})
const webhookUrl = ref('')
const secretValue = ref('')
const templateVariables = computed(() => [
  { label: t('alarmNotifications.variables.pointName'), value: '{{pointName}}' },
  { label: t('alarmNotifications.variables.value'), value: '{{value}}' },
  { label: t('alarmNotifications.variables.condition'), value: '{{conditionLabel}}' },
  { label: t('alarmNotifications.variables.severity'), value: '{{severity}}' },
  { label: t('alarmNotifications.variables.occurredAt'), value: '{{occurredAt}}' },
])
const enabledChannels = computed(() => channels.value.filter((item) => item.isEnabled))
const channelName = (channel: AlarmNotificationChannel) =>
  channel.id === 'runtime_inapp' ? t('alarmNotifications.builtinChannelName') : channel.name

onMounted(() => void load())
let loadRequestSeq = 0

async function load() {
  const seq = ++loadRequestSeq
  loading.value = true
  channelsLoading.value = true
  const [settingsResult, channelsResult] = await Promise.allSettled([
    getAlarmSettings(props.projectId),
    listAlarmChannels(props.projectId),
  ])
  if (seq !== loadRequestSeq) return
  if (settingsResult.status === 'fulfilled') {
    const data = settingsResult.value
    Object.assign(settings, data)
    repeatEnabled.value = data.repeatIntervalSeconds != null
  } else {
    ElMessage.error(getApiErrorMessage(settingsResult.reason, t('alarmNotifications.loadSettingsFailed')))
  }
  if (channelsResult.status === 'rejected') {
    ElMessage.error(getApiErrorMessage(channelsResult.reason, t('alarmNotifications.loadChannelsFailed')))
  }
  channels.value = ensureBuiltinAlarmNotificationChannel(
    props.projectId,
    channelsResult.status === 'fulfilled' ? channelsResult.value : [],
  )
  emit('changed', channels.value)
  loading.value = false
  channelsLoading.value = false
}

async function saveSettings() {
  saving.value = true
  try {
    const saved = await saveAlarmSettings(props.projectId, {
      notifyOnRaise: settings.notifyOnRaise,
      notifyOnClear: settings.notifyOnClear,
      repeatIntervalSeconds: repeatEnabled.value ? settings.repeatIntervalSeconds : null,
      defaultMessageTemplate: settings.defaultMessageTemplate,
      defaultChannelIds: settings.defaultChannelIds,
    })
    Object.assign(settings, saved)
    ElMessage.success(t('alarmNotifications.settingsSaved'))
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, t('alarmNotifications.saveSettingsFailed')))
  } finally {
    saving.value = false
  }
}

function insertVariable(value: string) {
  settings.defaultMessageTemplate += value
}

function resetChannelDraft() {
  Object.assign(channelDraft, {
    name: '',
    channelType: 'webhook',
    config: {},
    secrets: {},
    deleteSecretKeys: [],
    isEnabled: true,
  })
  webhookUrl.value = ''
  secretValue.value = ''
}

function openCreate() {
  editingChannel.value = null
  resetChannelDraft()
  dialogVisible.value = true
}

function openEdit(channel: AlarmNotificationChannel) {
  editingChannel.value = channel
  Object.assign(channelDraft, {
    name: channel.name,
    channelType: channel.channelType,
    config: { ...channel.config },
    secrets: {},
    deleteSecretKeys: [],
    isEnabled: channel.isEnabled,
  })
  webhookUrl.value = String(channel.config.webhookUrl || '')
  secretValue.value = ''
  dialogVisible.value = true
}

async function saveChannel() {
  channelSaving.value = true
  try {
    const payload: AlarmNotificationChannelSave = {
      ...channelDraft,
      config: { webhookUrl: webhookUrl.value.trim() },
      secrets: secretValue.value.trim() ? { signingSecret: secretValue.value.trim() } : {},
      deleteSecretKeys: [],
    }
    if (editingChannel.value)
      await updateAlarmChannel(props.projectId, editingChannel.value.id, payload)
    else await createAlarmChannel(props.projectId, payload)
    dialogVisible.value = false
    ElMessage.success(t('alarmNotifications.channelSaved'))
    await load()
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, t('alarmNotifications.saveChannelFailed')))
  } finally {
    channelSaving.value = false
  }
}

async function removeChannel(channel: AlarmNotificationChannel) {
  if (
    !(await confirm(t('alarmNotifications.deleteConfirm', { name: channel.name }), {
      title: t('alarmNotifications.deleteTitle'),
      confirmText: t('alarm.delete'),
      type: 'warning',
    }))
  )
    return
  try {
    await deleteAlarmChannel(props.projectId, channel.id)
    ElMessage.success(t('alarmNotifications.channelDeleted'))
    await load()
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, t('alarmNotifications.deleteFailed')))
  }
}

async function runChannelTest(channel: AlarmNotificationChannel) {
  channelTestingId.value = channel.id
  try {
    await testAlarmChannel(props.projectId, channel.id)
    ElMessage.success(t('alarmNotifications.testSent'))
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, t('alarmNotifications.testFailed')))
  } finally {
    channelTestingId.value = ''
    await load()
  }
}

function testStatusText(status: string) {
  return (
    ({ not_tested: t('alarmNotifications.testStatuses.not_tested'), succeeded: t('alarmNotifications.testStatuses.succeeded'), failed: t('alarmNotifications.testStatuses.failed') } as Record<string, string>)[
      status
    ] || t('alarmNotifications.testStatuses.not_tested')
  )
}

function testStatusTone(status: string): 'success' | 'danger' | 'muted' {
  if (status === 'succeeded') return 'success'
  if (status === 'failed') return 'danger'
  return 'muted'
}

function channelTypeLabel(value: string) {
  return (
    (
      {
        runtime_inapp: t('alarmNotifications.channelTypes.runtime_inapp'),
        webhook: 'Webhook',
        dingtalk: t('alarmNotifications.channelTypes.dingtalk'),
        wecom: t('alarmNotifications.channelTypes.wecom'),
      } as Record<string, string>
    )[value] || value
  )
}

function secretSummary(channel: AlarmNotificationChannel) {
  const count = Object.values(channel.secretStatus).filter(Boolean).length
  return channel.id === 'runtime_inapp'
    ? t('alarmNotifications.noSecret')
    : count
      ? t('alarmNotifications.configuredSecrets', { count })
      : t('alarmNotifications.notConfigured')
}
</script>

<style scoped>
.alarm-notifications {
  width: 100%;
  height: 100%;
  min-height: 0;
  display: grid;
  grid-template-columns: minmax(320px, 380px) minmax(0, 1fr);
}
.alarm-notifications__settings,
.alarm-notifications__channels {
  min-width: 0;
  min-height: 0;
  padding: 20px;
  background: var(--dc-surface-raised);
}
.alarm-notifications__settings {
  overflow: auto;
  border-right: 1px solid var(--dc-border);
  scrollbar-color: var(--dc-border-strong, #cbd5e1) transparent;
  scrollbar-width: thin;
}
.alarm-notifications__settings::-webkit-scrollbar {
  width: 7px;
}
.alarm-notifications__settings::-webkit-scrollbar-thumb {
  border-radius: 7px;
  background: var(--dc-border-strong, #cbd5e1);
}
.alarm-notifications__channels {
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
.alarm-notifications__head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  min-height: 34px;
  margin-bottom: 16px;
}
.alarm-notifications__head h3 {
  margin: 0;
  font-size: 15px;
  letter-spacing: 0;
}
.alarm-notifications__form {
  display: grid;
  grid-template-columns: 1fr;
  gap: 16px;
}
.alarm-notifications__form label,
.alarm-channel-form label {
  display: grid;
  gap: 6px;
  color: var(--dc-text-secondary);
  font-size: 12px;
}
.alarm-notifications__checks {
  min-height: 32px;
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  align-items: center;
  gap: 8px;
}
.alarm-notifications__checks :deep(.el-checkbox) {
  width: 100%;
  height: 32px;
  display: inline-flex;
  align-items: center;
  margin-right: 0;
  line-height: 1;
}
.alarm-notifications__checks :deep(.el-checkbox__input) {
  display: inline-flex;
  align-items: center;
}
.alarm-notifications__checks :deep(.el-checkbox__label) {
  min-width: 0;
  padding-left: 8px;
  overflow: hidden;
  font-size: 13px;
  line-height: 20px;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.alarm-notifications__repeat {
  display: flex;
  align-items: center;
  gap: 8px;
}
.alarm-notifications__variables {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.alarm-notifications__variables button {
  padding: 3px 7px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-muted);
  color: var(--dc-text-secondary);
  font-size: 11px;
  cursor: pointer;
}
.alarm-notifications__variables button:hover {
  border-color: color-mix(in oklch, var(--dc-primary) 24%, var(--dc-border));
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
}
.alarm-notifications__primary,
.alarm-notifications__secondary,
.alarm-notifications__icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  border-radius: var(--dc-radius-sm);
  cursor: pointer;
  font-family: inherit;
}
.alarm-notifications__primary,
.alarm-notifications__secondary {
  height: 32px;
  padding: 0 12px;
}
.alarm-notifications__primary {
  border: 1px solid var(--dc-primary);
  background: var(--dc-primary);
  color: white;
}
.alarm-notifications__secondary {
  border: 1px solid var(--dc-border);
  background: var(--dc-surface-raised);
  color: var(--dc-text-secondary);
}
.alarm-notifications__secondary svg {
  width: 15px;
}
.alarm-notifications__primary svg {
  width: 15px;
}
.alarm-notifications__primary:hover {
  background: color-mix(in srgb, var(--dc-primary) 88%, black);
}
.alarm-notifications__secondary:hover,
.alarm-notifications__icon:hover {
  border-color: color-mix(in oklch, var(--dc-primary) 24%, var(--dc-border));
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
}
.alarm-notifications__primary:focus-visible,
.alarm-notifications__secondary:focus-visible,
.alarm-notifications__icon:focus-visible,
.alarm-notifications__variables button:focus-visible {
  outline: none;
  border-color: var(--dc-primary);
  box-shadow: 0 0 0 3px rgba(29, 78, 216, 0.12);
}
.alarm-notifications__icon {
  width: 30px;
  height: 30px;
  border: 1px solid var(--dc-border);
  background: var(--dc-surface-raised);
  color: var(--dc-text-secondary);
}
.alarm-notifications__icon.is-danger {
  margin-left: 4px;
}
.alarm-notifications__icon.is-danger:hover {
  border-color: color-mix(in oklch, var(--el-color-danger) 28%, var(--dc-border));
  background: color-mix(in srgb, var(--el-color-danger) 8%, white);
  color: var(--el-color-danger);
}
.alarm-notifications__table {
  min-height: 0;
  flex: 1;
}
.alarm-notifications__table :deep(.el-table__inner-wrapper::before) {
  display: none;
}
.alarm-notifications__table :deep(.el-table__header th) {
  height: 50px;
  background: var(--dc-surface-raised);
  color: var(--dc-text);
  font-size: 13px;
  font-weight: 700;
  letter-spacing: 0;
}
.alarm-notifications__table :deep(.el-table__cell) {
  padding: 8px 0;
  color: var(--dc-text-secondary);
  font-size: 13px;
}
.alarm-notifications__table :deep(.el-table__row) {
  height: 60px;
}
.alarm-notifications__table :deep(.el-table__row:hover > td.el-table__cell) {
  background: var(--dc-surface-muted);
}
.alarm-channel-form {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 14px;
}
.alarm-channel-form .is-wide {
  grid-column: 1 / -1;
}
@container alarm-workspace (max-width: 1100px) {
  .alarm-notifications {
    grid-template-columns: minmax(300px, 340px) minmax(0, 1fr);
  }
}
@container alarm-workspace (max-width: 860px) {
  .alarm-notifications {
    grid-template-columns: minmax(0, 1fr);
    grid-template-rows: max-content minmax(280px, max-content);
    align-content: start;
    overflow: auto;
  }
  .alarm-notifications__settings,
  .alarm-notifications__channels {
    padding: 16px;
  }
  .alarm-notifications__settings {
    overflow: visible;
    border-right: 0;
    border-bottom: 1px solid var(--dc-border);
  }
  .alarm-notifications__channels {
    min-height: 280px;
    overflow: visible;
  }
  .alarm-notifications__table {
    min-height: 220px;
    flex: 0 0 auto;
  }
}
@container alarm-workspace (max-width: 520px) {
  .alarm-notifications__head {
    align-items: stretch;
    flex-direction: column;
  }
  .alarm-notifications__checks {
    grid-template-columns: minmax(0, 1fr);
  }
  .alarm-notifications__primary,
  .alarm-notifications__secondary {
    align-self: flex-start;
  }
}
</style>
