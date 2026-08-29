<template>
  <section class="collector-agent-page">
    <div class="collector-agent-workspace">
      <header class="collector-agent-header">
        <div class="collector-agent-header__identity">
          <h1>{{ ui('采集调试代理', 'Collector Debug Agents') }}</h1>
        </div>

        <div class="collector-agent-header__actions">
          <div v-if="agents.length > 0" class="collector-agent-view-switcher">
            <el-tooltip :content="ui('卡片视图', 'Card View')" placement="top">
              <button
                type="button"
                :class="{ 'is-active': viewMode === 'card' }"
                :aria-label="ui('切换到卡片视图', 'Switch to card view')"
                @click="viewMode = 'card'"
              >
                <IconTablerLayoutGrid />
              </button>
            </el-tooltip>
            <el-tooltip :content="ui('列表视图', 'List View')" placement="top">
              <button
                type="button"
                :class="{ 'is-active': viewMode === 'list' }"
                :aria-label="ui('切换到列表视图', 'Switch to list view')"
                @click="viewMode = 'list'"
              >
                <IconTablerListDetails />
              </button>
            </el-tooltip>
          </div>
          <button
            type="button"
            class="collector-agent-action collector-agent-action--secondary"
            :disabled="loading"
            :aria-label="ui('刷新采集调试代理', 'Refresh collector debug agents')"
            @click="loadAgents()"
          >
            <IconTablerRefresh :class="{ 'is-spinning': loading }" />
            <span>{{ ui('刷新', 'Refresh') }}</span>
          </button>
          <button
            v-if="canManage"
            type="button"
            class="collector-agent-action collector-agent-action--primary"
            @click="createRegistrationCode"
          >
            <IconTablerKey />
            <span>{{ ui('生成注册码', 'Generate Registration Code') }}</span>
          </button>
        </div>
      </header>

      <section class="collector-agent-content-panel">
        <div v-if="!canManage" class="collector-agent-notice">
          <IconTablerInfoCircle />
          <span>{{ ui('当前账号可查看代理状态；生成注册码和移除代理需要系统管理员权限。', 'This account can view agent status. Generating registration codes and removing agents require system administrator permission.') }}</span>
        </div>

        <div v-if="loadError" class="collector-agent-error" role="alert">
          <div class="collector-agent-error__icon">
            <IconTablerAlertTriangle />
          </div>
          <div class="collector-agent-error__content">
            <strong>{{ ui('代理状态加载失败', 'Failed to Load Agent Status') }}</strong>
            <span>{{ loadError }}</span>
          </div>
          <button type="button" class="collector-agent-error__retry" @click="loadAgents()">
            {{ ui('重新加载', 'Reload') }}
          </button>
        </div>

        <main v-loading="loading" class="collector-agent-content">
          <section
            v-if="!loading && agents.length === 0 && !loadError"
            class="collector-agent-empty"
          >
            <div class="collector-agent-empty__intro">
              <div class="collector-agent-empty__visual">
                <IconTablerDeviceDesktop />
                <span><IconTablerPlugConnected /></span>
              </div>
              <div>
                <p class="collector-agent-empty__eyebrow">{{ ui('尚未连接调试代理', 'No Debug Agent Connected') }}</p>
                <h2>{{ ui('将采集节点接入中心', 'Connect a Collector Node') }}</h2>
                <p>{{ ui('采集调试代理作为独立进程运行，注册完成后即可承载开发态工业协议调试。', 'The collector debug agent runs as a separate process and provides development-time industrial protocol debugging after registration.') }}</p>
              </div>
            </div>

            <ol class="collector-agent-steps">
              <li>
                <span class="collector-agent-step__number">1</span>
                <div>
                  <strong>{{ ui('生成一次性注册码', 'Generate a One-time Registration Code') }}</strong>
                  <p>{{ ui('注册码仅用于首次注册，并在短时间后自动失效。', 'The code is only used for initial registration and expires shortly afterward.') }}</p>
                </div>
              </li>
              <li>
                <span class="collector-agent-step__number">2</span>
                <div>
                  <strong>{{ ui('启动采集调试代理', 'Start the Collector Debug Agent') }}</strong>
                  <p>{{ ui('在可访问设备网络的节点上运行代理程序。', 'Run the agent on a node that can access the device network.') }}</p>
                </div>
              </li>
              <li>
                <span class="collector-agent-step__number">3</span>
                <div>
                  <strong>{{ ui('连接当前中心', 'Connect to This Center') }}</strong>
                  <p>{{ ui('填写中心地址和注册码，完成后代理会自动上报状态。', 'Enter the center address and registration code. The agent will report its status automatically.') }}</p>
                </div>
              </li>
            </ol>

            <div v-if="canManage" class="collector-agent-empty__action">
              <button
                type="button"
                class="collector-agent-action collector-agent-action--primary"
                @click="createRegistrationCode"
              >
                <IconTablerKey />
                <span>{{ ui('生成首个注册码', 'Generate First Registration Code') }}</span>
              </button>
            </div>
          </section>

          <section
            v-else-if="agents.length > 0 && viewMode === 'card'"
            class="collector-agent-grid"
          >
            <article
              v-for="agent in agents"
              :key="agent.id"
              class="collector-agent-card"
              :class="`is-${agent.status}`"
            >
              <header class="collector-agent-card__header">
                <div class="collector-agent-card__machine">
                  <div class="collector-agent-card__device">
                    <IconTablerDeviceDesktop />
                    <span :class="`is-${agent.status}`" />
                  </div>
                  <div>
                    <h2>{{ agent.name }}</h2>
                    <p :title="`${agent.os} / ${agent.arch} · v${agent.version || '-'}`">
                      {{ agent.os }} / {{ agent.arch }} · v{{ agent.version || '-' }}
                    </p>
                  </div>
                </div>
                <span class="collector-agent-card__status" :class="`is-${agent.status}`">
                  {{ agentStatusText(agent.status) }}
                </span>
              </header>

              <dl class="collector-agent-card__details">
                <div>
                  <dt><IconTablerFingerprint />{{ ui('代理标识', 'Agent ID') }}</dt>
                  <dd>{{ agent.id }}</dd>
                </div>
                <div>
                  <dt><IconTablerNetwork />{{ ui('IP 地址', 'IP Address') }}</dt>
                  <dd>{{ agent.ipAddress || ui('尚未上报', 'Not Reported') }}</dd>
                </div>
                <div>
                  <dt><IconTablerClock />{{ ui('最后心跳', 'Last Heartbeat') }}</dt>
                  <dd>{{ agent.lastSeenAt || ui('尚未上报', 'Not Reported') }}</dd>
                </div>
              </dl>

              <div class="collector-agent-card__capabilities">
                <div class="collector-agent-card__section-title">
                  <span>{{ ui('协议能力', 'Protocol Capabilities') }}</span>
                  <small>{{ ui(`${agent.capabilities.length} 项`, `${agent.capabilities.length}`) }}</small>
                </div>
                <div v-if="agent.capabilities.length > 0" class="collector-agent-capability-list">
                  <el-tooltip
                    v-for="capability in agent.capabilities"
                    :key="`${agent.id}-${capability.driverId}`"
                    :content="capability.operations.join(datacenterLocale === 'en' ? ', ' : '、') || ui('未上报操作能力', 'No operations reported')"
                    placement="top"
                  >
                    <span class="collector-agent-capability">
                      {{ capability.driverId.toUpperCase() }}
                      <small>{{ capability.operations.length }}</small>
                    </span>
                  </el-tooltip>
                </div>
                <span v-else class="collector-agent-card__empty-capability">{{ ui('暂未上报协议能力', 'No protocol capabilities reported') }}</span>
              </div>

              <footer
                v-if="canManage && agent.status !== 'invalid'"
                class="collector-agent-card__footer"
              >
                <button
                  type="button"
                  class="collector-agent-card__remove"
                  @click="removeAgent(agent)"
                >
                  <IconTablerTrash />
                  <span>{{ ui('移除代理', 'Remove Agent') }}</span>
                </button>
              </footer>
              <footer
                v-else-if="agent.status === 'invalid'"
                class="collector-agent-card__footer collector-agent-card__invalid-note"
              >
                {{ ui('需要使用新注册码重新激活', 'A new registration code is required to reactivate') }}
              </footer>
            </article>
          </section>

          <section v-else-if="agents.length > 0" class="collector-agent-table-wrap">
            <el-table :data="agents" row-key="id" class="collector-agent-table">
              <el-table-column :label="ui('代理', 'Agent')" min-width="210">
                <template #default="{ row }">
                  <div class="collector-agent-table__machine">
                    <div class="collector-agent-table__device">
                      <IconTablerDeviceDesktop />
                      <span :class="`is-${row.status}`" />
                    </div>
                    <div>
                      <strong>{{ row.name }}</strong>
                      <small>{{ row.id }}</small>
                    </div>
                  </div>
                </template>
              </el-table-column>
              <el-table-column :label="ui('状态', 'Status')" width="88" align="center">
                <template #default="{ row }">
                  <span class="collector-agent-card__status" :class="`is-${row.status}`">
                    {{ agentStatusText(row.status) }}
                  </span>
                </template>
              </el-table-column>
              <el-table-column :label="ui('系统', 'System')" min-width="130">
                <template #default="{ row }">{{ row.os }} / {{ row.arch }}</template>
              </el-table-column>
              <el-table-column prop="ipAddress" :label="ui('IP 地址', 'IP Address')" width="140" />
              <el-table-column prop="version" :label="ui('版本', 'Version')" min-width="180" show-overflow-tooltip />
              <el-table-column :label="ui('协议能力', 'Capabilities')" min-width="160">
                <template #default="{ row }">
                  <div class="collector-agent-table__protocols">
                    <span v-for="capability in row.capabilities" :key="capability.driverId">
                      {{ capability.driverId.toUpperCase() }}
                    </span>
                    <small v-if="row.capabilities.length === 0">{{ ui('未上报', 'Not Reported') }}</small>
                  </div>
                </template>
              </el-table-column>
              <el-table-column :label="ui('最后心跳', 'Last Heartbeat')" width="170">
                <template #default="{ row }">{{ row.lastSeenAt || ui('尚未上报', 'Not Reported') }}</template>
              </el-table-column>
              <el-table-column v-if="canManage" :label="ui('操作', 'Actions')" width="90" align="right" fixed="right">
                <template #default="{ row }">
                  <button
                    v-if="row.status !== 'invalid'"
                    type="button"
                    class="collector-agent-table__remove"
                    @click="removeAgent(row)"
                  >
                    {{ ui('移除', 'Remove') }}
                  </button>
                  <span v-else class="collector-agent-table__invalid">{{ ui('需重新注册', 'Registration Required') }}</span>
                </template>
              </el-table-column>
            </el-table>
          </section>
        </main>

        <DataCenterPagination
          v-if="pagination.total > 0"
          :page="pagination.page"
          :page-size="pagination.pageSize"
          :total="pagination.total"
          :total-pages="pagination.totalPages"
          @change="handlePaginationChange"
        />
      </section>
    </div>

    <el-dialog
      v-model="registrationDialogVisible"
      class="collector-registration-dialog"
      width="520px"
      :close-on-click-modal="false"
    >
      <template #header>
        <div class="registration-dialog-title">
          <span><IconTablerShieldLock /></span>
          <div>
            <strong>{{ ui('一次性注册码', 'One-time Registration Code') }}</strong>
            <small>{{ ui('用于采集调试代理首次连接当前租户', 'Used for the collector debug agent’s first connection to this tenant') }}</small>
          </div>
        </div>
      </template>

      <div class="registration-dialog-notice">
        <IconTablerAlertTriangle />
        <span>{{ ui('注册码只展示本次，请勿发送给无关人员。', 'This code is shown only once. Do not share it with unauthorized people.') }}</span>
      </div>
      <div class="registration-code-box">
        <code>{{ registrationCode?.code }}</code>
        <button type="button" @click="copyRegistrationCode">
          <IconTablerCopy />
          <span>{{ ui('复制', 'Copy') }}</span>
        </button>
      </div>
      <p class="registration-code-expiry">
        <IconTablerClock />
        <span>{{ ui('有效期至', 'Expires at') }} {{ registrationCode?.expiresAt }}</span>
      </p>
      <template #footer>
        <button
          type="button"
          class="collector-agent-action collector-agent-action--primary"
          @click="registrationDialogVisible = false"
        >
          {{ ui('我已保存', 'I Have Saved It') }}
        </button>
      </template>
    </el-dialog>
  </section>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import IconTablerAlertTriangle from '~icons/tabler/alert-triangle'
import IconTablerClock from '~icons/tabler/clock'
import IconTablerCopy from '~icons/tabler/copy'
import IconTablerDeviceDesktop from '~icons/tabler/device-desktop'
import IconTablerFingerprint from '~icons/tabler/fingerprint'
import IconTablerInfoCircle from '~icons/tabler/info-circle'
import IconTablerKey from '~icons/tabler/key'
import IconTablerLayoutGrid from '~icons/tabler/layout-grid'
import IconTablerListDetails from '~icons/tabler/list-details'
import IconTablerNetwork from '~icons/tabler/network'
import IconTablerPlugConnected from '~icons/tabler/plug-connected'
import IconTablerRefresh from '~icons/tabler/refresh'
import IconTablerShieldLock from '~icons/tabler/shield-lock'
import IconTablerTrash from '~icons/tabler/trash'
import {
  createCollectorRegistrationCode,
  deleteCollectorAgent,
  getCollectorAgents,
} from '@/api/collector-dev.api'
import type { CollectorAgent, CollectorRegistrationCode } from '@/api/schemas/collector-dev.schema'
import DataCenterPagination from '@/components/shared/DataCenterPagination.vue'
import { Storage } from '@/utils/storage'
import { getApiErrorMessage } from '@/utils/request'
import { datacenterLocale } from '@/i18n/runtime'

const ui = (zh: string, en: string) => (datacenterLocale.value === 'en' ? en : zh)

const agents = ref<CollectorAgent[]>([])
const viewMode = ref<'card' | 'list'>('card')
const loading = ref(false)
const pagination = ref({ page: 1, pageSize: 10, total: 0, totalPages: 0 })
const loadError = ref('')
const registrationCode = ref<CollectorRegistrationCode | null>(null)
const registrationDialogVisible = ref(false)
const userInfo = Storage.getUserInfo() as { role?: string } | null
const canManage = computed(() => ['SYSTEM_ADMIN', 'SUPER_ADMIN'].includes(userInfo?.role ?? ''))

let refreshTimer: number | undefined
let refreshRunning = false

const agentStatusText = (status: CollectorAgent['status']) =>
  datacenterLocale.value === 'en'
    ? ({ online: 'Online', offline: 'Offline', invalid: 'Invalid' })[status]
    : ({ online: '在线', offline: '离线', invalid: '无效' })[status]

const loadAgents = async (silent = false) => {
  if (refreshRunning) return
  refreshRunning = true
  if (!silent) loading.value = true
  if (!silent) loadError.value = ''
  try {
    let result = await getCollectorAgents({
      page: pagination.value.page,
      pageSize: pagination.value.pageSize,
    })
    if (result.pagination.totalPages > 0 && result.pagination.page > result.pagination.totalPages) {
      pagination.value.page = result.pagination.totalPages
      result = await getCollectorAgents({
        page: pagination.value.page,
        pageSize: pagination.value.pageSize,
      })
    }
    loadError.value = ''
    agents.value = result.list
    pagination.value = result.pagination
  } catch (error) {
    if (!silent) {
      loadError.value = getApiErrorMessage(error, ui('无法连接数据服务，请确认服务已启动后重试。', 'Unable to connect to the data service. Confirm that it is running and try again.'))
    }
  } finally {
    refreshRunning = false
    if (!silent) loading.value = false
  }
}

const handlePaginationChange = async (value: { page: number; pageSize: number }) => {
  pagination.value.page = value.page
  pagination.value.pageSize = value.pageSize
  await loadAgents()
}

const createRegistrationCode = async () => {
  try {
    registrationCode.value = await createCollectorRegistrationCode()
    registrationDialogVisible.value = true
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, ui('生成注册码失败', 'Failed to generate registration code')))
  }
}

const copyRegistrationCode = async () => {
  if (!registrationCode.value) return
  await navigator.clipboard.writeText(registrationCode.value.code)
  ElMessage.success(ui('注册码已复制', 'Registration code copied'))
}

const removeAgent = async (agent: CollectorAgent) => {
  await ElMessageBox.confirm(
    ui(`移除“${agent.name}”后，其本地 Token 将立即失效。`, `Removing “${agent.name}” immediately invalidates its local token.`),
    ui('移除采集调试代理', 'Remove Collector Debug Agent'),
    { type: 'warning', confirmButtonText: ui('移除', 'Remove'), cancelButtonText: ui('取消', 'Cancel') },
  )
  try {
    await deleteCollectorAgent(agent.id)
    ElMessage.success(ui('代理已移除', 'Agent removed'))
    await loadAgents()
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, ui('移除代理失败', 'Failed to remove agent')))
  }
}

const handleVisibilityChange = () => {
  if (document.visibilityState === 'visible') void loadAgents(true)
}

onMounted(() => {
  void loadAgents()
  refreshTimer = window.setInterval(() => {
    if (document.visibilityState === 'visible') void loadAgents(true)
  }, 5000)
  document.addEventListener('visibilitychange', handleVisibilityChange)
})
onBeforeUnmount(() => {
  if (refreshTimer !== undefined) window.clearInterval(refreshTimer)
  document.removeEventListener('visibilitychange', handleVisibilityChange)
})
</script>

<style scoped>
.collector-agent-page {
  height: 100%;
  min-height: 0;
  overflow: hidden;
  padding: 20px;
  background: var(--dc-surface-subtle);
}

.collector-agent-workspace {
  height: 100%;
  min-height: 0;
  display: flex;
  flex-direction: column;
  gap: 14px;
  overflow: visible;
}

.collector-agent-header {
  flex: 0 0 auto;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 24px;
  padding: 16px 20px;
  border: 1px solid var(--dc-border);
  border-radius: 14px;
  background: var(--dc-surface-raised);
  box-shadow:
    0 8px 24px rgba(15, 23, 42, 0.06),
    0 1px 2px rgba(15, 23, 42, 0.04);
}

.collector-agent-content-panel {
  min-width: 0;
  min-height: 0;
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  border: 1px solid var(--dc-border);
  border-radius: 14px;
  background: var(--dc-surface-raised);
  box-shadow:
    0 8px 24px rgba(15, 23, 42, 0.055),
    0 1px 2px rgba(15, 23, 42, 0.04);
}

.collector-agent-header__identity {
  min-width: 0;
  min-height: 34px;
  display: flex;
  align-items: center;
}

.collector-agent-header h1 {
  margin: 0;
  color: var(--dc-text);
  font-size: 22px;
  font-weight: 700;
  line-height: 1.2;
}

.collector-agent-header__actions,
.collector-agent-action {
  display: inline-flex;
  align-items: center;
}

.collector-agent-header__actions {
  flex: 0 0 auto;
  gap: 8px;
}

.collector-agent-view-switcher {
  height: 34px;
  display: inline-flex;
  align-items: center;
  gap: 2px;
  padding: 2px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-muted);
}

.collector-agent-view-switcher button {
  width: 28px;
  height: 28px;
  display: grid;
  place-items: center;
  border: none;
  border-radius: 5px;
  background: transparent;
  color: var(--dc-text-muted);
}

.collector-agent-view-switcher button.is-active {
  background: var(--dc-surface-raised);
  color: var(--dc-primary);
  box-shadow: 0 1px 3px rgba(15, 23, 42, 0.1);
}

.collector-agent-view-switcher svg {
  width: 15px;
  height: 15px;
}

.collector-agent-action {
  min-height: 34px;
  justify-content: center;
  gap: 7px;
  padding: 0 13px;
  border-radius: var(--dc-radius-sm);
  font-size: 13px;
  font-weight: 700;
  transition:
    border-color 0.18s ease,
    background-color 0.18s ease,
    color 0.18s ease,
    transform 0.18s ease;
}

.collector-agent-action svg {
  width: 16px;
  height: 16px;
}

.collector-agent-action:not(:disabled):hover {
  transform: translateY(-1px);
}

.collector-agent-action:disabled {
  cursor: not-allowed;
  opacity: 0.6;
}

.collector-agent-action--secondary {
  border: 1px solid var(--dc-border);
  background: var(--dc-surface-raised);
  color: var(--dc-text-secondary);
}

.collector-agent-action--secondary:not(:disabled):hover {
  border-color: color-mix(in srgb, var(--dc-primary) 28%, var(--dc-border));
  color: var(--dc-primary);
}

.collector-agent-action--primary {
  border: 1px solid var(--dc-primary);
  background: var(--dc-primary);
  color: #fff;
}

.collector-agent-action--primary:not(:disabled):hover {
  border-color: var(--dc-primary-hover);
  background: var(--dc-primary-hover);
}

.collector-agent-notice,
.collector-agent-error {
  display: flex;
  align-items: center;
  margin: 16px 24px 0;
  border-radius: var(--dc-radius-sm);
}

.collector-agent-notice {
  gap: 9px;
  padding: 10px 12px;
  border: 1px solid color-mix(in srgb, var(--dc-primary) 28%, var(--dc-border));
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
  font-size: 12px;
}

.collector-agent-notice svg {
  flex: 0 0 16px;
  width: 16px;
  height: 16px;
}

.collector-agent-error {
  gap: 12px;
  padding: 12px 14px;
  border: 1px solid color-mix(in srgb, var(--dc-danger) 28%, var(--dc-border));
  background: var(--dc-danger-soft);
}

.collector-agent-error__icon {
  flex: 0 0 34px;
  width: 34px;
  height: 34px;
  display: grid;
  place-items: center;
  border-radius: 9px;
  background: var(--dc-danger-soft);
  color: var(--dc-danger);
}

.collector-agent-error__icon svg {
  width: 18px;
  height: 18px;
}

.collector-agent-error__content {
  min-width: 0;
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: 2px;
}

.collector-agent-error__content strong {
  color: #991b1b;
  font-size: 13px;
}

.collector-agent-error__content span {
  color: #b91c1c;
  font-size: 12px;
  overflow-wrap: anywhere;
}

.collector-agent-error__retry {
  flex: 0 0 auto;
  padding: 6px 11px;
  border: 1px solid #fca5a5;
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
  color: #b91c1c;
  font-size: 12px;
  font-weight: 700;
}

.collector-agent-content {
  min-height: 0;
  flex: 1;
  overflow: auto;
  padding: 20px 24px 24px;
}

.collector-agent-empty {
  overflow: hidden;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-md);
  background: linear-gradient(
    180deg,
    var(--dc-surface-raised) 0%,
    var(--dc-surface-subtle) 100%
  );
}

.collector-agent-empty__intro {
  display: flex;
  align-items: center;
  gap: 20px;
  padding: 26px 28px 22px;
}

.collector-agent-empty__visual {
  position: relative;
  flex: 0 0 76px;
  width: 76px;
  height: 76px;
  display: grid;
  place-items: center;
  border: 1px solid color-mix(in srgb, var(--dc-primary) 28%, var(--dc-border));
  border-radius: 20px;
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
}

.collector-agent-empty__visual > svg {
  width: 36px;
  height: 36px;
}

.collector-agent-empty__visual span {
  position: absolute;
  right: -7px;
  bottom: -7px;
  width: 30px;
  height: 30px;
  display: grid;
  place-items: center;
  border: 3px solid var(--dc-surface-subtle);
  border-radius: 50%;
  background: var(--dc-primary);
  color: #fff;
}

.collector-agent-empty__visual span svg {
  width: 15px;
  height: 15px;
}

.collector-agent-empty__eyebrow {
  margin: 0 0 4px;
  color: var(--dc-primary);
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.08em;
}

.collector-agent-empty h2 {
  margin: 0;
  color: var(--dc-text);
  font-size: 18px;
}

.collector-agent-empty__intro p:last-child {
  max-width: 660px;
  margin: 7px 0 0;
  color: var(--dc-text-secondary);
  font-size: 13px;
  line-height: 1.6;
}

.collector-agent-steps {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 0;
  margin: 0;
  padding: 0 28px 24px;
  list-style: none;
}

.collector-agent-steps li {
  position: relative;
  min-width: 0;
  display: flex;
  gap: 11px;
  padding: 17px 20px 17px 0;
  border-top: 1px solid var(--dc-border);
}

.collector-agent-steps li:not(:last-child)::after {
  position: absolute;
  top: 17px;
  right: 12px;
  width: 1px;
  height: calc(100% - 34px);
  background: var(--dc-border);
  content: '';
}

.collector-agent-step__number {
  flex: 0 0 26px;
  width: 26px;
  height: 26px;
  display: grid;
  place-items: center;
  border: 1px solid color-mix(in srgb, var(--dc-primary) 28%, var(--dc-border));
  border-radius: 50%;
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
  font-size: 12px;
  font-weight: 800;
}

.collector-agent-steps strong {
  color: var(--dc-text);
  font-size: 13px;
}

.collector-agent-steps p {
  margin: 5px 0 0;
  color: var(--dc-text-secondary);
  font-size: 12px;
  line-height: 1.55;
}

.collector-agent-empty__action {
  display: flex;
  justify-content: flex-end;
  padding: 14px 28px;
  border-top: 1px solid var(--dc-border);
  background: color-mix(in srgb, var(--dc-surface-raised) 92%, transparent);
}

.collector-agent-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 380px));
  gap: 14px;
  justify-content: start;
}

.collector-agent-card {
  overflow: hidden;
  border: 1px solid var(--dc-border);
  border-top: 3px solid #cbd5e1;
  border-radius: var(--dc-radius-md);
  background: var(--dc-surface-raised);
  transition:
    border-color 0.18s ease,
    box-shadow 0.18s ease,
    transform 0.18s ease;
}

.collector-agent-card:hover {
  transform: translateY(-1px);
  border-color: color-mix(in srgb, var(--dc-primary) 20%, var(--dc-border));
  box-shadow: 0 10px 24px rgba(15, 23, 42, 0.07);
}

.collector-agent-card.is-online {
  border-top-color: #22c55e;
}

.collector-agent-card__header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 14px;
  padding: 17px 18px 15px;
  border-bottom: 1px solid var(--dc-border);
}

.collector-agent-card__machine {
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 11px;
}

.collector-agent-card__device {
  position: relative;
  flex: 0 0 38px;
  width: 38px;
  height: 38px;
  display: grid;
  place-items: center;
  border-radius: 10px;
  background: var(--dc-surface-muted);
  color: var(--dc-text-secondary);
}

.collector-agent-card__device svg {
  width: 20px;
  height: 20px;
}

.collector-agent-card__device span {
  position: absolute;
  right: -2px;
  bottom: -2px;
  width: 10px;
  height: 10px;
  border: 2px solid var(--dc-surface-raised);
  border-radius: 50%;
  background: #94a3b8;
}

.collector-agent-card__device span.is-online {
  background: #22c55e;
}

.collector-agent-card__header h2 {
  max-width: 260px;
  margin: 0;
  overflow: hidden;
  color: var(--dc-text);
  font-size: 15px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.collector-agent-card__header p {
  max-width: 270px;
  margin: 5px 0 0;
  overflow: hidden;
  color: var(--dc-text-secondary);
  font-size: 11px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.collector-agent-table-wrap {
  width: 100%;
  min-width: 0;
  overflow: hidden;
  background: var(--dc-surface-raised);
}

.collector-agent-table__machine {
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 10px;
}

.collector-agent-table__device {
  position: relative;
  flex: 0 0 32px;
  width: 32px;
  height: 32px;
  display: grid;
  place-items: center;
  border-radius: 8px;
  background: var(--dc-surface-muted);
  color: var(--dc-text-secondary);
}

.collector-agent-table__device svg {
  width: 17px;
  height: 17px;
}

.collector-agent-table__device span {
  position: absolute;
  right: -1px;
  bottom: -1px;
  width: 9px;
  height: 9px;
  border: 2px solid var(--dc-surface-raised);
  border-radius: 50%;
  background: #94a3b8;
}

.collector-agent-table__device span.is-online {
  background: #22c55e;
}

.collector-agent-table__machine div:last-child {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 3px;
}

.collector-agent-table__machine strong {
  color: var(--dc-primary);
  font-size: 13px;
  font-weight: 700;
}

.collector-agent-table__machine small {
  overflow: hidden;
  color: var(--dc-text-muted);
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: 10px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.collector-agent-table__protocols {
  display: flex;
  flex-wrap: wrap;
  gap: 5px;
}

.collector-agent-table__protocols span {
  padding: 3px 6px;
  border: 1px solid color-mix(in srgb, var(--dc-primary) 28%, var(--dc-border));
  border-radius: 5px;
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
  font-size: 10px;
  font-weight: 700;
}

.collector-agent-table__protocols small {
  color: var(--dc-text-muted);
}

.collector-agent-table :deep(.el-table__header th) {
  height: 50px;
  background: var(--dc-surface-raised);
  color: var(--dc-text);
  font-size: 13px;
  font-weight: 700;
}

.collector-agent-table :deep(.el-table__cell) {
  padding: 8px 0;
  color: var(--dc-text-secondary);
  font-size: 13px;
}

.collector-agent-table :deep(.el-table__inner-wrapper::before) {
  display: none;
}

.collector-agent-table :deep(.el-table__row) {
  height: 60px;
}

.collector-agent-table :deep(.el-table__row td) {
  border-bottom-color: color-mix(in oklch, var(--dc-border) 66%, transparent);
  transition: background-color 0.16s ease;
}

.collector-agent-table :deep(.el-table__row:hover > td.el-table__cell) {
  background: color-mix(in oklch, var(--dc-primary-soft) 28%, var(--dc-surface-raised));
}

.collector-agent-table :deep(.el-table-fixed-column--right) {
  background: var(--dc-surface-raised);
}

.collector-agent-table :deep(th.el-table-fixed-column--right) {
  background: var(--dc-surface-raised);
}

.collector-agent-table :deep(.el-table__row:hover > td.el-table-fixed-column--right) {
  background: color-mix(in oklch, var(--dc-primary-soft) 28%, var(--dc-surface-raised));
}

.collector-agent-table__remove {
  height: 28px;
  padding: 0 8px;
  border: 0;
  border-radius: 6px;
  background: transparent;
  color: var(--dc-text-muted);
  font-size: 12px;
  font-weight: 600;
}

.collector-agent-table__remove:hover {
  background: var(--dc-danger-soft);
  color: var(--dc-danger);
}

.collector-agent-card__status {
  flex: 0 0 auto;
  padding: 4px 8px;
  border-radius: 999px;
  font-size: 11px;
  font-weight: 700;
}

.collector-agent-card__status.is-online {
  background: var(--dc-success-soft);
  color: var(--dc-success);
}

.collector-agent-card__status.is-offline {
  background: var(--dc-surface-muted);
  color: var(--dc-text-muted);
}

.collector-agent-card__status.is-invalid {
  background: var(--dc-danger-soft);
  color: var(--dc-danger);
}

.collector-agent-card.is-invalid {
  border-color: #fecdd3;
}

.collector-agent-card__device span.is-invalid,
.collector-agent-table__device span.is-invalid {
  background: #e11d48;
}

.collector-agent-card__invalid-note,
.collector-agent-table__invalid {
  color: #be123c;
  font-size: 12px;
  font-weight: 600;
}

.collector-agent-card__details {
  display: grid;
  gap: 11px;
  margin: 0;
  padding: 15px 18px;
}

.collector-agent-card__details div {
  display: grid;
  grid-template-columns: 96px minmax(0, 1fr);
  gap: 12px;
}

.collector-agent-card__details dt {
  display: flex;
  align-items: center;
  gap: 6px;
  color: var(--dc-text-secondary);
  font-size: 11px;
}

.collector-agent-card__details dt svg {
  width: 14px;
  height: 14px;
}

.collector-agent-card__details dd {
  margin: 0;
  overflow-wrap: anywhere;
  color: var(--dc-text);
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: 11px;
  text-align: right;
}

.collector-agent-card__capabilities {
  padding: 13px 18px 16px;
  border-top: 1px solid var(--dc-border);
  background: var(--dc-surface-subtle);
}

.collector-agent-card__section-title {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 9px;
}

.collector-agent-card__section-title span {
  color: var(--dc-text);
  font-size: 12px;
  font-weight: 700;
}

.collector-agent-card__section-title small {
  color: var(--dc-text-muted);
  font-size: 10px;
}

.collector-agent-capability-list {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.collector-agent-capability {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 5px 8px;
  border: 1px solid color-mix(in srgb, var(--dc-primary) 28%, var(--dc-border));
  border-radius: 6px;
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
  font-size: 10px;
  font-weight: 700;
  cursor: default;
}

.collector-agent-capability small {
  min-width: 16px;
  padding: 1px 4px;
  border-radius: 999px;
  background: color-mix(in srgb, var(--dc-primary) 12%, transparent);
  text-align: center;
}

.collector-agent-card__empty-capability {
  color: var(--dc-text-muted);
  font-size: 11px;
}

.collector-agent-card__footer {
  display: flex;
  justify-content: flex-end;
  padding: 10px 18px;
  border-top: 1px solid var(--dc-border);
}

.collector-agent-card__remove {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 5px 8px;
  border: none;
  border-radius: 6px;
  background: transparent;
  color: var(--dc-text-muted);
  font-size: 11px;
}

.collector-agent-card__remove:hover {
  background: var(--dc-danger-soft);
  color: var(--dc-danger);
}

.collector-agent-card__remove svg {
  width: 14px;
  height: 14px;
}

.registration-dialog-title {
  display: flex;
  align-items: center;
  gap: 11px;
}

.registration-dialog-title > span {
  width: 36px;
  height: 36px;
  display: grid;
  place-items: center;
  border-radius: 10px;
  background: var(--dc-primary-soft);
  color: var(--dc-primary);
}

.registration-dialog-title svg {
  width: 19px;
  height: 19px;
}

.registration-dialog-title div {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.registration-dialog-title strong {
  color: var(--dc-text);
  font-size: 16px;
}

.registration-dialog-title small {
  color: var(--dc-text-secondary);
  font-size: 11px;
}

.registration-dialog-notice {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 12px;
  border: 1px solid color-mix(in srgb, var(--dc-warning) 28%, var(--dc-border));
  border-radius: var(--dc-radius-sm);
  background: var(--dc-warning-soft);
  color: var(--dc-warning);
  font-size: 12px;
}

.registration-dialog-notice svg {
  flex: 0 0 16px;
  width: 16px;
  height: 16px;
}

.registration-code-box {
  display: flex;
  align-items: stretch;
  gap: 10px;
  margin-top: 16px;
}

.registration-code-box code {
  min-width: 0;
  flex: 1;
  padding: 14px;
  overflow-wrap: anywhere;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-muted);
  color: var(--dc-text);
  font-size: 14px;
  line-height: 1.5;
}

.registration-code-box button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 0 13px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-raised);
  color: var(--dc-text-secondary);
  font-size: 12px;
  font-weight: 700;
}

.registration-code-box button:hover {
  border-color: color-mix(in srgb, var(--dc-primary) 28%, var(--dc-border));
  color: var(--dc-primary);
}

.registration-code-box button svg {
  width: 15px;
  height: 15px;
}

.registration-code-expiry {
  display: flex;
  align-items: center;
  gap: 6px;
  margin: 12px 0 0;
  color: var(--dc-text-secondary);
  font-size: 11px;
}

.registration-code-expiry svg {
  width: 14px;
  height: 14px;
}

.is-spinning {
  animation: collector-agent-spin 0.8s linear infinite;
}

@keyframes collector-agent-spin {
  to {
    transform: rotate(360deg);
  }
}

@media (max-width: 980px) {
  .collector-agent-steps {
    grid-template-columns: 1fr;
  }

  .collector-agent-steps li:not(:last-child)::after {
    display: none;
  }
}

@media (max-width: 680px) {
  .collector-agent-page {
    padding: 12px;
  }

  .collector-agent-header {
    flex-direction: column;
    padding: 18px;
  }

  .collector-agent-header__actions {
    width: 100%;
  }

  .collector-agent-header__actions .collector-agent-action {
    flex: 1;
  }

  .collector-agent-content {
    padding: 16px 18px 20px;
  }

  .collector-agent-empty__intro {
    align-items: flex-start;
    flex-direction: column;
    padding: 22px;
  }

  .collector-agent-steps {
    padding: 0 22px 18px;
  }

  .collector-agent-empty__action {
    padding: 14px 22px;
  }

  .collector-agent-grid {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
