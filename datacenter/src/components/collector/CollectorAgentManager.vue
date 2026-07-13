<template>
  <section class="collector-agent-page">
    <header class="collector-agent-page__header">
      <div>
        <p class="collector-agent-page__eyebrow">开发态设备网络接入</p>
        <h1>采集调试代理</h1>
        <p>Windows 调试代理注册到当前租户后，可由工程调试 API 复用。该页面不承载接入源配置。</p>
      </div>
      <div class="collector-agent-page__actions">
        <el-button @click="loadAgents">刷新</el-button>
        <el-button v-if="canManage" type="primary" @click="createRegistrationCode"
          >生成注册码</el-button
        >
      </div>
    </header>

    <el-alert v-if="!canManage" type="info" :closable="false" show-icon>
      当前账号可查看代理状态；生成注册码和删除代理需要系统管理员权限。
    </el-alert>

    <div v-loading="loading" class="collector-agent-page__content">
      <el-empty v-if="!loading && agents.length === 0" description="当前租户尚未注册采集调试代理">
        <el-button v-if="canManage" type="primary" @click="createRegistrationCode"
          >生成首个注册码</el-button
        >
      </el-empty>

      <div v-else class="collector-agent-grid">
        <article v-for="agent in agents" :key="agent.id" class="collector-agent-card">
          <div class="collector-agent-card__title">
            <div>
              <h2>{{ agent.name }}</h2>
              <span>{{ agent.os }} / {{ agent.arch }} · v{{ agent.version || '-' }}</span>
            </div>
            <el-tag :type="agent.online ? 'success' : 'info'" effect="light">
              {{ agent.online ? '在线' : '离线' }}
            </el-tag>
          </div>
          <dl>
            <div>
              <dt>Agent ID</dt>
              <dd>{{ agent.id }}</dd>
            </div>
            <div>
              <dt>最后心跳</dt>
              <dd>{{ agent.lastSeenAt || '尚未上报' }}</dd>
            </div>
            <div>
              <dt>协议能力</dt>
              <dd>{{ formatCapabilities(agent) }}</dd>
            </div>
          </dl>
          <footer v-if="canManage">
            <el-button type="danger" plain @click="removeAgent(agent)">移除代理</el-button>
          </footer>
        </article>
      </div>
    </div>

    <el-dialog
      v-model="registrationDialogVisible"
      title="一次性注册码"
      width="520px"
      :close-on-click-modal="false"
    >
      <el-alert
        type="warning"
        :closable="false"
        show-icon
        title="注册码只展示本次，请勿发送给无关人员。"
      />
      <div class="registration-code-box">
        <code>{{ registrationCode?.code }}</code>
        <el-button @click="copyRegistrationCode">复制</el-button>
      </div>
      <p class="registration-code-expiry">有效期至：{{ registrationCode?.expiresAt }}</p>
      <template #footer
        ><el-button type="primary" @click="registrationDialogVisible = false"
          >我已保存</el-button
        ></template
      >
    </el-dialog>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  createCollectorRegistrationCode,
  deleteCollectorAgent,
  getCollectorAgents,
} from '@/api/collector-dev.api'
import type { CollectorAgent, CollectorRegistrationCode } from '@/api/schemas/collector-dev.schema'
import { Storage } from '@/utils/storage'
import { getApiErrorMessage } from '@/utils/request'

const agents = ref<CollectorAgent[]>([])
const loading = ref(false)
const registrationCode = ref<CollectorRegistrationCode | null>(null)
const registrationDialogVisible = ref(false)
const userInfo = Storage.getUserInfo() as { role?: string } | null
const canManage = computed(() => ['SYSTEM_ADMIN', 'SUPER_ADMIN'].includes(userInfo?.role ?? ''))

const loadAgents = async () => {
  loading.value = true
  try {
    agents.value = await getCollectorAgents()
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '加载采集调试代理失败'))
  } finally {
    loading.value = false
  }
}

const createRegistrationCode = async () => {
  try {
    registrationCode.value = await createCollectorRegistrationCode()
    registrationDialogVisible.value = true
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '生成注册码失败'))
  }
}

const copyRegistrationCode = async () => {
  if (!registrationCode.value) return
  await navigator.clipboard.writeText(registrationCode.value.code)
  ElMessage.success('注册码已复制')
}

const removeAgent = async (agent: CollectorAgent) => {
  await ElMessageBox.confirm(
    `移除“${agent.name}”后，其本地 Token 将立即失效。`,
    '移除采集调试代理',
    { type: 'warning', confirmButtonText: '移除', cancelButtonText: '取消' },
  )
  try {
    await deleteCollectorAgent(agent.id)
    ElMessage.success('代理已移除')
    await loadAgents()
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '移除代理失败'))
  }
}

const formatCapabilities = (agent: CollectorAgent) =>
  agent.capabilities
    .map((item) => `${item.protocolType.toUpperCase()}：${item.operations.join('、')}`)
    .join('；') || '未上报'

onMounted(loadAgents)
</script>

<style scoped>
.collector-agent-page {
  height: 100%;
  overflow: auto;
  padding: 28px;
  background: var(--dc-surface-subtle);
}
.collector-agent-page__header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 24px;
  margin-bottom: 20px;
}
.collector-agent-page__header h1 {
  margin: 4px 0 8px;
  color: var(--dc-text);
  font-size: 24px;
}
.collector-agent-page__header p {
  margin: 0;
  color: var(--dc-text-secondary);
}
.collector-agent-page__eyebrow {
  color: var(--dc-primary) !important;
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0.08em;
}
.collector-agent-page__actions {
  display: flex;
  gap: 8px;
}
.collector-agent-page__content {
  min-height: 260px;
  margin-top: 18px;
}
.collector-agent-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(340px, 1fr));
  gap: 16px;
}
.collector-agent-card {
  padding: 20px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-md);
  background: var(--dc-surface-raised);
  box-shadow: var(--dc-shadow-surface);
}
.collector-agent-card__title {
  display: flex;
  justify-content: space-between;
  gap: 16px;
}
.collector-agent-card__title h2 {
  margin: 0 0 6px;
  color: var(--dc-text);
  font-size: 17px;
}
.collector-agent-card__title span {
  color: var(--dc-text-secondary);
  font-size: 12px;
}
.collector-agent-card dl {
  display: grid;
  gap: 12px;
  margin: 18px 0;
}
.collector-agent-card dl div {
  display: grid;
  grid-template-columns: 88px minmax(0, 1fr);
  gap: 12px;
}
.collector-agent-card dt {
  color: var(--dc-text-secondary);
  font-size: 12px;
}
.collector-agent-card dd {
  margin: 0;
  overflow-wrap: anywhere;
  color: var(--dc-text);
  font-size: 13px;
}
.collector-agent-card footer {
  display: flex;
  justify-content: flex-end;
  padding-top: 14px;
  border-top: 1px solid var(--dc-border);
}
.registration-code-box {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-top: 18px;
  padding: 14px;
  border-radius: var(--dc-radius-sm);
  background: var(--dc-surface-muted);
}
.registration-code-box code {
  flex: 1;
  overflow-wrap: anywhere;
  color: var(--dc-text);
  font-size: 15px;
}
.registration-code-expiry {
  color: var(--dc-text-secondary);
  font-size: 13px;
}
</style>
