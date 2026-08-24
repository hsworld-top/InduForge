<template>
  <div class="alarm-notifications">
    <section class="alarm-notifications__settings">
      <header class="alarm-notifications__head">
        <h3>工程默认通知</h3>
        <button
          type="button"
          class="alarm-notifications__primary"
          :disabled="saving"
          @click="saveSettings"
        >
          <IconTablerDeviceFloppy />
          保存设置
        </button>
      </header>
      <div v-loading="loading" class="alarm-notifications__form">
        <div class="alarm-notifications__checks">
          <el-checkbox v-model="settings.notifyOnRaise">报警触发时通知</el-checkbox>
          <el-checkbox v-model="settings.notifyOnClear">报警恢复时通知</el-checkbox>
        </div>
        <label>
          <span>重复提醒</span>
          <div class="alarm-notifications__repeat">
            <el-switch v-model="repeatEnabled" />
            <el-input-number
              v-if="repeatEnabled"
              v-model="settings.repeatIntervalSeconds"
              :min="1"
              controls-position="right"
            />
            <span v-if="repeatEnabled">秒</span>
          </div>
        </label>
        <label>
          <span>默认渠道</span>
          <el-select v-model="settings.defaultChannelIds" multiple placeholder="选择通知渠道">
            <el-option
              v-for="channel in enabledChannels"
              :key="channel.id"
              :label="channel.name"
              :value="channel.id"
            />
          </el-select>
        </label>
        <label class="is-wide">
          <span>默认消息模板</span>
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
        <h3>通知渠道</h3>
        <button type="button" class="alarm-notifications__secondary" @click="openCreate">
          <IconTablerPlus />新增渠道
        </button>
      </header>
      <el-table v-loading="channelsLoading" :data="channels" class="alarm-notifications__table">
        <el-table-column label="名称" min-width="180" prop="name" />
        <el-table-column label="类型" width="130">
          <template #default="{ row }">{{ channelTypeLabel(row.channelType) }}</template>
        </el-table-column>
        <el-table-column label="密钥" min-width="150">
          <template #default="{ row }">{{ secretSummary(row) }}</template>
        </el-table-column>
        <el-table-column label="状态" width="100">
          <template #default="{ row }"
            ><StatusBadge
              :tone="row.isEnabled ? 'success' : 'muted'"
              :text="row.isEnabled ? '启用' : '停用'"
          /></template>
        </el-table-column>
        <el-table-column label="操作" width="100" align="right">
          <template #default="{ row }">
            <button
              v-if="row.id !== 'runtime_inapp'"
              type="button"
              class="alarm-notifications__icon"
              title="编辑渠道"
              @click="openEdit(row)"
            >
              <IconTablerEdit />
            </button>
            <button
              v-if="row.id !== 'runtime_inapp'"
              type="button"
              class="alarm-notifications__icon is-danger"
              title="删除渠道"
              @click="removeChannel(row)"
            >
              <IconTablerTrash />
            </button>
          </template>
        </el-table-column>
        <template #empty>
          <el-empty :image-size="48" description="暂无通知渠道" />
        </template>
      </el-table>
    </section>

    <DcDialog
      v-model="dialogVisible"
      :title="editingChannel ? '编辑通知渠道' : '新增通知渠道'"
      width="520px"
    >
      <div class="alarm-channel-form">
        <label
          ><span>名称</span><el-input v-model="channelDraft.name" placeholder="例如：生产值班群"
        /></label>
        <label
          ><span>类型</span
          ><el-select v-model="channelDraft.channelType" :disabled="!!editingChannel"
            ><el-option label="Webhook" value="webhook" /><el-option
              label="钉钉"
              value="dingtalk" /><el-option label="企业微信" value="wecom" /></el-select
        ></label>
        <label class="is-wide"
          ><span>Webhook 地址</span><el-input v-model="webhookUrl" placeholder="https://..."
        /></label>
        <label class="is-wide"
          ><span>签名密钥（留空保持原值）</span
          ><el-input
            v-model="secretValue"
            type="password"
            show-password
            autocomplete="new-password"
        /></label>
        <label><span>启用</span><el-switch v-model="channelDraft.isEnabled" /></label>
      </div>
      <template #footer>
        <button type="button" class="alarm-notifications__secondary" @click="dialogVisible = false">
          取消
        </button>
        <button
          type="button"
          class="alarm-notifications__primary"
          :disabled="channelSaving"
          @click="saveChannel"
        >
          保存
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
} from '@/api/alarm.api'
import type {
  AlarmNotificationChannel,
  AlarmNotificationChannelSave,
} from '@/api/schemas/alarm.schema'
import { getApiErrorMessage } from '@/utils/request'
import { useConfirm } from '@/composables/useConfirm'
import { ensureBuiltinAlarmNotificationChannel } from '@/models/alarm-policy'

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
  defaultMessageTemplate: '{{pointName}} 当前值 {{value}}，触发 {{conditionLabel}}',
  defaultChannelIds: ['runtime_inapp'] as string[],
})
const dialogVisible = ref(false)
const editingChannel = ref<AlarmNotificationChannel | null>(null)
const channelSaving = ref(false)
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
const templateVariables = [
  { label: '点位名称', value: '{{pointName}}' },
  { label: '当前值', value: '{{value}}' },
  { label: '报警条件', value: '{{conditionLabel}}' },
  { label: '报警等级', value: '{{severity}}' },
  { label: '触发时间', value: '{{occurredAt}}' },
]
const enabledChannels = computed(() => channels.value.filter((item) => item.isEnabled))

onMounted(() => void load())

async function load() {
  loading.value = true
  channelsLoading.value = true
  const [settingsResult, channelsResult] = await Promise.allSettled([
    getAlarmSettings(props.projectId),
    listAlarmChannels(props.projectId),
  ])
  if (settingsResult.status === 'fulfilled') {
    const data = settingsResult.value
    Object.assign(settings, data)
    repeatEnabled.value = data.repeatIntervalSeconds != null
  } else {
    ElMessage.error(getApiErrorMessage(settingsResult.reason, '加载通知设置失败'))
  }
  if (channelsResult.status === 'rejected') {
    ElMessage.error(getApiErrorMessage(channelsResult.reason, '加载通知渠道失败'))
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
    ElMessage.success('工程通知设置已保存')
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '保存通知设置失败'))
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
    ElMessage.success('通知渠道已保存')
    await load()
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '保存通知渠道失败'))
  } finally {
    channelSaving.value = false
  }
}

async function removeChannel(channel: AlarmNotificationChannel) {
  if (
    !(await confirm(`确认删除通知渠道「${channel.name}」？`, {
      title: '删除通知渠道',
      confirmText: '删除',
      type: 'warning',
    }))
  )
    return
  try {
    await deleteAlarmChannel(props.projectId, channel.id)
    ElMessage.success('通知渠道已删除')
    await load()
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '删除通知渠道失败'))
  }
}

function channelTypeLabel(value: string) {
  return (
    (
      {
        runtime_inapp: '站内通知',
        webhook: 'Webhook',
        dingtalk: '钉钉',
        wecom: '企业微信',
      } as Record<string, string>
    )[value] || value
  )
}

function secretSummary(channel: AlarmNotificationChannel) {
  const count = Object.values(channel.secretStatus).filter(Boolean).length
  return channel.id === 'runtime_inapp' ? '无需密钥' : count ? `已配置 ${count} 项` : '未配置'
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
