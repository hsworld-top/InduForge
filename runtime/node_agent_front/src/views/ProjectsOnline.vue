<template>
  <div class="projects">
    <el-card class="box-card panel-card">
      <template #header>
        <span>{{ t('remote.bindTitle') }}</span>
      </template>
      <el-alert
        v-if="centerOffline"
        class="offline-alert"
        type="warning"
        :closable="false"
        :title="centerStatusMessage || t('remote.centerOfflineSaved')"
      />
      <el-form v-if="showBindForm" :model="centerForm" label-width="110px" class="center-form">
        <el-form-item :label="t('remote.centerUrl')" required>
          <div class="address-row">
            <el-select v-model="centerForm.centerProtocol" style="width: 120px">
              <el-option label="http://" value="http" />
              <el-option label="https://" value="https" />
            </el-select>
            <el-input v-model="centerForm.centerIp" placeholder="127.0.0.1" />
            <el-input v-model="centerForm.centerPort" placeholder="19601" style="width: 140px" />
          </div>
        </el-form-item>
        <el-form-item :label="t('remote.tenantCode')">
          <el-input
            v-model="centerForm.tenantCode"
            :placeholder="locale === 'en-US' ? 'default (optional)' : 'default（可选）'"
          />
        </el-form-item>
        <el-form-item :label="locale === 'en-US' ? 'Username' : '用户名'" required>
          <el-input v-model="centerForm.username" />
        </el-form-item>
        <el-form-item :label="locale === 'en-US' ? 'Password' : '密码'" required>
          <el-input v-model="centerForm.password" type="password" show-password />
        </el-form-item>
        <el-form-item :label="t('remote.nodeName')" required>
          <el-input v-model="centerForm.nodeName" />
        </el-form-item>
        <el-form-item :label="t('remote.nodeId')">
          <el-input :model-value="machineId || t('common.na')" readonly />
        </el-form-item>
        <el-form-item :label="locale === 'en-US' ? 'Node Description' : '节点描述'">
          <el-input v-model="centerForm.nodeDescription" type="textarea" :rows="2" />
        </el-form-item>
      </el-form>
      <el-descriptions v-else :column="1" border>
        <el-descriptions-item :label="t('remote.bindTitle')">
          <el-tag class="status-tag" v-if="approvalPending" type="warning">{{
            t('remote.pendingApproval')
          }}</el-tag>
          <el-tag class="status-tag" v-else type="success">{{ t('remote.bound') }}</el-tag>
        </el-descriptions-item>
        <el-descriptions-item :label="t('remote.centerStatus')">
          <el-tag class="status-tag" :type="centerOffline ? 'danger' : 'success'">
            {{ centerOffline ? t('remote.offline') : t('remote.online') }}
          </el-tag>
        </el-descriptions-item>
        <el-descriptions-item :label="t('remote.centerUrl')">
          {{ bindingSnapshot.centerUrl || t('common.na') }}
        </el-descriptions-item>
        <el-descriptions-item :label="t('remote.tenantCode')">
          {{ bindingSnapshot.tenantCode || t('common.na') }}
        </el-descriptions-item>
        <el-descriptions-item :label="t('remote.nodeName')">
          {{ bindingSnapshot.nodeName || t('common.na') }}
        </el-descriptions-item>
        <el-descriptions-item :label="t('remote.nodeId')">
          {{ machineId || t('common.na') }}
        </el-descriptions-item>
      </el-descriptions>
      <div class="bind-actions">
        <el-button v-if="showBindForm" type="primary" :loading="binding" @click="bindCenter">{{
          t('remote.bindSubmit')
        }}</el-button>
        <el-button
          v-else-if="approvalPending"
          type="primary"
          :loading="checkingApproval"
          @click="checkApproval"
        >
          {{ t('remote.refreshApproval') }}
        </el-button>
        <el-button v-if="centerOffline" @click="reconcileCenterBindingState">
          {{ t('remote.retry') }}
        </el-button>
      </div>
    </el-card>

    <el-card class="box-card panel-card">
      <template #header>
        <span>{{ t('remote.managedTitle') }}</span>
      </template>
      <el-table v-loading="nodeStore.loading" :data="onlineProjects" style="width: 100%">
        <el-table-column prop="id" :label="t('local.projectId')" min-width="170" />
        <el-table-column :label="t('remote.source')" width="120">
          <template #default>
            <el-tag class="status-tag" type="warning">{{ t('remote.centerManaged') }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="currentVersion" :label="t('local.version')" width="120" />
        <el-table-column :label="t('local.status')" width="110">
          <template #default="{ row }">
            <el-tag class="status-tag" :type="getStatusType(row.status)">{{
              formatStatus(row.status)
            }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column :label="t('local.runtimeStatus')" width="140">
          <template #default="{ row }">{{
            formatStatus(row.runtimeStatus?.state || 'stopped')
          }}</template>
        </el-table-column>
        <el-table-column :label="t('local.pid')" width="100">
          <template #default="{ row }">{{ row.runtimeStatus?.pid || t('common.na') }}</template>
        </el-table-column>
        <el-table-column :label="t('local.lastStartedAt')" width="180">
          <template #default="{ row }">{{ formatTime(row.lastStartedAt) }}</template>
        </el-table-column>
        <el-table-column :label="t('local.actions')" min-width="220" fixed="right">
          <template #default="{ row }">
            <el-tag class="status-tag" type="info">{{ t('remote.statusOnly') }}</el-tag>
            <el-button
              type="success"
              size="small"
              style="margin-left: 8px"
              @click="viewLogs(row.id)"
              >{{ t('local.logs') }}</el-button
            >
          </template>
        </el-table-column>
      </el-table>
      <el-empty
        v-if="!onlineProjects.length && !nodeStore.loading"
        :description="t('remote.noManagedProjects')"
      />
    </el-card>
  </div>
</template>

<script setup>
import { computed, onMounted, onUnmounted, reactive, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { useNodeStore } from '@/store/nodeStore'
import { checkApprovalStatus, registerNodeWithToken } from '@/api/registerApi'
import { nodeApi } from '@/api/nodeApi'
import { useI18nText } from '@/composables/useI18nText'

const router = useRouter()
const nodeStore = useNodeStore()
const { locale, t } = useI18nText()

const binding = ref(false)
const checkingApproval = ref(false)
const centerBound = ref(false)
const approvalPending = ref(false)
const machineId = ref('')
const runtimeCenterUrl = ref('')
const runtimeNodeId = ref('')
const centerOffline = ref(false)
const centerStatusMessage = ref('')
let reconcileTimer = null
const draftStorageKey = 'node_agent_center_bind_draft'
const snapshotStorageKey = 'node_agent_center_bind_snapshot'

const centerForm = reactive({
  centerProtocol: 'http',
  centerIp: '',
  centerPort: '19601',
  tenantCode: '',
  username: '',
  password: '',
  nodeName: '',
  nodeDescription: '',
  ipAddress: '',
  nodePort: 17601,
  agentVersion: '1.0.0',
})

const bindingSnapshot = reactive({
  submitted: false,
  status: '',
  centerUrl: '',
  tenantCode: '',
  nodeName: '',
  nodeId: '',
})

const pendingRegistration = reactive({
  centerUrl: '',
  nodeId: '',
  registrationToken: '',
})

const onlineProjects = computed(() =>
  nodeStore.projects.filter((project) => project?.source === 'center'),
)
const showBindForm = computed(() => !bindingSnapshot.submitted)

const isValidIPv4 = (value) => {
  const pattern = /^(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}$/
  return pattern.test(value)
}

const isDuplicateNodeNameError = (message) => {
  const msg = String(message || '').toLowerCase()
  return (
    msg.includes('must be unique') ||
    msg.includes('duplicate entry') ||
    msg.includes('节点名称已存在')
  )
}

const buildCenterUrl = () => {
  const protocol = centerForm.centerProtocol
  const ip = centerForm.centerIp.trim()
  const portText = String(centerForm.centerPort).trim()
  const port = Number(portText)

  if (!protocol || !['http', 'https'].includes(protocol)) {
    throw new Error('请选择协议')
  }
  if (!isValidIPv4(ip)) {
    throw new Error('请输入合法的IP地址')
  }
  if (!Number.isInteger(port) || port < 1 || port > 65535) {
    throw new Error('请输入合法端口（1-65535）')
  }

  return `${protocol}://${ip}:${port}`
}

const parseCenterUrl = (url) => {
  try {
    const parsed = new URL(url)
    const protocol = parsed.protocol.replace(':', '')
    if (protocol === 'http' || protocol === 'https') {
      centerForm.centerProtocol = protocol
    }
    centerForm.centerIp = parsed.hostname || centerForm.centerIp
    centerForm.centerPort = parsed.port || centerForm.centerPort
  } catch (_) {
    // 忽略无效历史值，避免影响输入
  }
}

const saveDraft = () => {
  const draft = {
    centerProtocol: centerForm.centerProtocol,
    centerIp: centerForm.centerIp,
    centerPort: centerForm.centerPort,
    tenantCode: centerForm.tenantCode,
    username: centerForm.username,
    nodeName: centerForm.nodeName,
    nodeDescription: centerForm.nodeDescription,
  }
  localStorage.setItem(draftStorageKey, JSON.stringify(draft))
}

const restoreDraft = () => {
  try {
    const raw = localStorage.getItem(draftStorageKey)
    if (!raw) return
    const draft = JSON.parse(raw)
    centerForm.centerProtocol = draft?.centerProtocol || centerForm.centerProtocol
    centerForm.centerIp = draft?.centerIp || centerForm.centerIp
    centerForm.centerPort = draft?.centerPort || centerForm.centerPort
    centerForm.tenantCode = draft?.tenantCode || centerForm.tenantCode
    centerForm.username = draft?.username || centerForm.username
    centerForm.nodeName = draft?.nodeName || centerForm.nodeName
    centerForm.nodeDescription = draft?.nodeDescription || centerForm.nodeDescription
  } catch (_) {}
}

const saveSnapshot = () => {
  localStorage.setItem(snapshotStorageKey, JSON.stringify(bindingSnapshot))
}

const restoreSnapshot = () => {
  try {
    const raw = localStorage.getItem(snapshotStorageKey)
    if (!raw) return
    const snapshot = JSON.parse(raw)
    bindingSnapshot.submitted = !!snapshot?.submitted
    bindingSnapshot.status = snapshot?.status || ''
    bindingSnapshot.centerUrl = snapshot?.centerUrl || ''
    bindingSnapshot.tenantCode = snapshot?.tenantCode || ''
    bindingSnapshot.nodeName = snapshot?.nodeName || ''
    bindingSnapshot.nodeId = snapshot?.nodeId || ''
    approvalPending.value = bindingSnapshot.status === 'pending'
    if (bindingSnapshot.centerUrl) {
      pendingRegistration.centerUrl = bindingSnapshot.centerUrl
      parseCenterUrl(bindingSnapshot.centerUrl)
    }
    if (bindingSnapshot.nodeId) {
      pendingRegistration.nodeId = bindingSnapshot.nodeId
    }
  } catch (_) {}
}

const loadBootstrapStatus = async () => {
  try {
    const bootstrap = await nodeApi.getBootstrapStatus()
    centerBound.value = !!bootstrap?.centerBound
    runtimeCenterUrl.value = bootstrap?.centerUrl || ''
    runtimeNodeId.value = bootstrap?.nodeId || ''
    if (bootstrap?.centerUrl) {
      parseCenterUrl(bootstrap.centerUrl)
    }
    centerForm.nodeName = bootstrap?.nodeName || centerForm.nodeName
    if (centerBound.value) {
      bindingSnapshot.submitted = true
      bindingSnapshot.status = 'approved'
      bindingSnapshot.centerUrl = bootstrap?.centerUrl || bindingSnapshot.centerUrl
      bindingSnapshot.nodeName = bootstrap?.nodeName || bindingSnapshot.nodeName
      bindingSnapshot.nodeId =
        bootstrap?.nodeId || pendingRegistration.nodeId || bindingSnapshot.nodeId
      approvalPending.value = false
      saveSnapshot()
    }
  } catch (error) {
    console.error('加载绑定状态失败:', error)
  }
}

const loadServiceConfig = async () => {
  try {
    const config = await nodeApi.getServiceConfig()
    if (config?.listen?.port) {
      centerForm.nodePort = config.listen.port
    }
  } catch (error) {
    console.error('加载服务配置失败:', error)
  }
}

const loadNodeInfo = async () => {
  try {
    const nodeInfo = await nodeApi.getNodeInfo()
    machineId.value = nodeInfo?.machineId || nodeInfo?.id || ''
  } catch (error) {
    console.error('加载节点信息失败:', error)
  }
}

const resetBindingState = async (silent = false) => {
  const fallbackNodeName = centerForm.nodeName || bindingSnapshot.nodeName || 'node-local'
  centerBound.value = false
  centerOffline.value = false
  centerStatusMessage.value = ''
  approvalPending.value = false
  runtimeCenterUrl.value = ''
  runtimeNodeId.value = ''
  bindingSnapshot.submitted = false
  bindingSnapshot.status = ''
  bindingSnapshot.centerUrl = ''
  bindingSnapshot.nodeId = ''
  localStorage.removeItem(snapshotStorageKey)
  pendingRegistration.centerUrl = ''
  pendingRegistration.nodeId = ''
  pendingRegistration.registrationToken = ''

  try {
    await nodeApi.completeBootstrap({
      mode: 'local',
      nodeName: fallbackNodeName,
    })
  } catch (error) {
    console.error('重置本地绑定状态失败:', error)
  }

  if (!silent) {
    ElMessage.warning(t('remote.centerNodeMissing'))
  }
}

const isNodeMissingMessage = (message) => {
  const msg = String(message || '').toLowerCase()
  return msg.includes('不存在') || msg.includes('not found') || msg.includes('node not found')
}

const reconcileCenterBindingState = async () => {
  if (!centerBound.value) return

  const centerUrl = runtimeCenterUrl.value || bindingSnapshot.centerUrl
  const nodeId = runtimeNodeId.value || bindingSnapshot.nodeId
  if (!centerUrl || !nodeId) return

  try {
    const res = await checkApprovalStatus(centerUrl, nodeId)
    centerOffline.value = false
    centerStatusMessage.value = ''
    if (res?.success === false) {
      const msg = String(res?.error || res?.message || '')
      if (isNodeMissingMessage(msg)) {
        await resetBindingState()
      }
      return
    }
    const status = res?.data?.approvalStatus
    if (status === 'approved') return
    if (status === 'pending') {
      approvalPending.value = true
      bindingSnapshot.status = 'pending'
      saveSnapshot()
      return
    }
    if (status === 'rejected') {
      await resetBindingState(true)
      return
    }
    // 未知状态不做解绑，等待下一次对账。
  } catch (error) {
    const message = String(error?.message || '')
    const isNotFound = error?.status === 404 || isNodeMissingMessage(message)
    if (isNotFound) {
      centerOffline.value = false
      centerStatusMessage.value = ''
      await resetBindingState()
      return
    }
    centerOffline.value = true
    centerStatusMessage.value = t('remote.centerOfflineSavedCurrent')
  }
}

const bindCenter = async () => {
  if (!centerForm.username || !centerForm.password || !centerForm.nodeName) {
    ElMessage.warning(t('remote.fillRequired'))
    return
  }
  let centerUrl = ''
  try {
    centerUrl = buildCenterUrl()
  } catch (error) {
    ElMessage.warning(error.message || t('remote.centerUrlInvalid'))
    return
  }

  binding.value = true
  try {
    const registerPayload = {
      centerUrl,
      tenantCode: centerForm.tenantCode,
      username: centerForm.username,
      password: centerForm.password,
      nodeName: centerForm.nodeName,
      nodeDescription: centerForm.nodeDescription,
      ipAddress: centerForm.ipAddress,
      port: centerForm.nodePort,
      agentVersion: centerForm.agentVersion,
    }
    const registerResult = await registerNodeWithToken(registerPayload)
    pendingRegistration.centerUrl = centerUrl
    pendingRegistration.nodeId = registerResult?.data?.nodeId || ''
    pendingRegistration.registrationToken = registerResult?.data?.registrationToken || ''
    bindingSnapshot.submitted = true
    bindingSnapshot.status = 'pending'
    bindingSnapshot.centerUrl = centerUrl
    bindingSnapshot.tenantCode = centerForm.tenantCode
    bindingSnapshot.nodeName = centerForm.nodeName
    bindingSnapshot.nodeId = pendingRegistration.nodeId
    saveSnapshot()
    if (registerResult?.data?.autoApproved || registerResult?.data?.approvalStatus === 'approved') {
      await nodeApi.completeBootstrap({
        mode: 'online',
        centerUrl: pendingRegistration.centerUrl,
        nodeId: pendingRegistration.nodeId,
        registrationToken: pendingRegistration.registrationToken,
      })
      centerBound.value = true
      centerOffline.value = false
      centerStatusMessage.value = ''
      approvalPending.value = false
      bindingSnapshot.status = 'approved'
      saveSnapshot()
      ElMessage.success(t('remote.bindSuccess'))
      await nodeStore.fetchProjects()
    } else {
      approvalPending.value = true
      bindingSnapshot.status = 'pending'
      saveSnapshot()
      ElMessage.success(t('remote.registerSubmitted'))
    }
  } catch (error) {
    if (isDuplicateNodeNameError(error?.message)) {
      ElMessage.error(t('remote.duplicateNodeName'))
      return
    }
    ElMessage.error(error?.message || t('remote.bindFailed'))
  } finally {
    binding.value = false
  }
}

const checkApproval = async (silent = false) => {
  if (!approvalPending.value) return
  checkingApproval.value = true
  try {
    const res = await checkApprovalStatus(pendingRegistration.centerUrl, pendingRegistration.nodeId)
    centerOffline.value = false
    centerStatusMessage.value = ''
    const status = res?.data?.approvalStatus
    if (status === 'approved') {
      await nodeApi.completeBootstrap({
        mode: 'online',
        centerUrl: pendingRegistration.centerUrl,
        nodeId: pendingRegistration.nodeId,
        registrationToken: res?.data?.registrationToken || pendingRegistration.registrationToken,
      })
      centerBound.value = true
      approvalPending.value = false
      bindingSnapshot.submitted = true
      bindingSnapshot.status = 'approved'
      bindingSnapshot.centerUrl = pendingRegistration.centerUrl
      bindingSnapshot.nodeId = pendingRegistration.nodeId
      saveSnapshot()
      if (!silent) {
        ElMessage.success(t('remote.approvalDone'))
      }
      await nodeStore.fetchProjects()
    } else if (status === 'rejected') {
      approvalPending.value = false
      bindingSnapshot.submitted = false
      bindingSnapshot.status = ''
      bindingSnapshot.nodeId = ''
      localStorage.removeItem(snapshotStorageKey)
      if (!silent) {
        ElMessage.error(t('remote.approvalRejected'))
      }
    } else {
      if (!silent) {
        ElMessage.info(t('remote.approvalPendingToast'))
      }
    }
  } catch (error) {
    if (error?.status === 404 || isNodeMissingMessage(error?.message)) {
      await resetBindingState()
      return
    }
    centerOffline.value = true
    centerStatusMessage.value = t('remote.centerOfflineSavedCurrent')
    if (!silent) {
      ElMessage.error(error?.message || t('remote.queryApprovalFailed'))
    }
  } finally {
    checkingApproval.value = false
  }
}

const viewLogs = (projectId) => router.push(`/logs/${projectId}`)

const getStatusType = (status) => {
  const map = { running: 'success', stopped: 'info', error: 'danger', deploying: 'warning' }
  return map[status] || 'info'
}

const formatStatus = (status) => t(`status.${status || 'unknown'}`)
const formatTime = (time) => (time ? new Date(time).toLocaleString(locale.value) : t('common.na'))
const pendingPollIntervalMs = 3000
const boundPollIntervalMs = 15000

const scheduleReconcile = () => {
  if (reconcileTimer) {
    clearTimeout(reconcileTimer)
    reconcileTimer = null
  }
  const nextDelay = approvalPending.value ? pendingPollIntervalMs : boundPollIntervalMs
  reconcileTimer = setTimeout(async () => {
    if (approvalPending.value) {
      await checkApproval(true)
    } else {
      await reconcileCenterBindingState()
    }
    scheduleReconcile()
  }, nextDelay)
}

watch(
  () => [
    centerForm.centerProtocol,
    centerForm.centerIp,
    centerForm.centerPort,
    centerForm.tenantCode,
    centerForm.username,
    centerForm.nodeName,
    centerForm.nodeDescription,
  ],
  saveDraft,
)

onMounted(async () => {
  restoreDraft()
  restoreSnapshot()
  await loadNodeInfo()
  await nodeStore.fetchProjects()
  await loadBootstrapStatus()
  await loadServiceConfig()
  if (approvalPending.value) {
    await checkApproval(true)
  }
  await reconcileCenterBindingState()
  scheduleReconcile()
})

onUnmounted(() => {
  if (reconcileTimer) {
    clearTimeout(reconcileTimer)
    reconcileTimer = null
  }
})
</script>

<style scoped>
.projects {
  display: grid;
  gap: 16px;
}

.box-card {
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  box-shadow: var(--shadow-sm);
}

.bind-actions {
  margin-top: 10px;
  display: flex;
  gap: 8px;
}

.offline-alert {
  margin-bottom: 12px;
}

.center-form {
  max-width: 760px;
}

.address-row {
  width: 100%;
  display: grid;
  grid-template-columns: 120px 1fr 140px;
  gap: 8px;
}
</style>
