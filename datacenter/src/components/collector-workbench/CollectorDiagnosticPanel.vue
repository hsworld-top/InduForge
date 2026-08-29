<template>
  <section class="collector-diagnostic" v-loading="loading">
    <header class="collector-diagnostic__head">
      <div>
        <h3>{{ ui('连接诊断', 'Connection Diagnostics') }}</h3>
        <p>{{ ui('只展示开发态代理和最近调试摘要，不代表节点运行状态。', 'Shows only the development agent and recent debug summary, not the node runtime state.') }}</p>
      </div>
      <div>
        <el-button @click="load">{{ ui('刷新', 'Refresh') }}</el-button>
        <el-button :loading="exporting" @click="exportCsv">{{ ui('导出失败点 CSV', 'Export Failed Points') }}</el-button>
      </div>
    </header>
    <el-alert
      v-if="diagnostic && !diagnostic.agent.ready"
      :title="agentReason(diagnostic.agent.reason)"
      type="warning"
      show-icon
      :closable="false"
    />
    <el-descriptions v-if="diagnostic" :column="3" border>
      <el-descriptions-item :label="ui('代理', 'Agent')">{{
        diagnostic.agent.name || ui('未选择', 'Not Selected')
      }}</el-descriptions-item>
      <el-descriptions-item :label="ui('最近测试', 'Latest Test')">{{ statusLabel(diagnostic.lastTest.status) }}</el-descriptions-item>
      <el-descriptions-item :label="ui('往返耗时', 'Round-trip Time')">{{
        diagnostic.lastTest.durationMs ? `${diagnostic.lastTest.durationMs} ms` : '—'
      }}</el-descriptions-item>
      <el-descriptions-item :label="ui('点位数', 'Points')">{{ diagnostic.pointCount }}</el-descriptions-item>
      <el-descriptions-item :label="ui('近期尝试', 'Recent Attempts')">{{
        diagnostic.attemptedPointCount
      }}</el-descriptions-item>
      <el-descriptions-item :label="ui('读取成功率', 'Read Success Rate')">{{ successRate }}</el-descriptions-item>
      <el-descriptions-item :label="ui('最近错误', 'Latest Error')" :span="3">{{
        diagnostic.lastTest.message || ui('无', 'None')
      }}</el-descriptions-item>
    </el-descriptions>
    <el-table
      v-if="diagnostic?.failedPoints.length"
      :data="diagnostic.failedPoints"
      class="collector-diagnostic__table"
    >
      <el-table-column prop="name" :label="ui('失败点位', 'Failed Point')" min-width="160" show-overflow-tooltip />
      <el-table-column prop="addressText" :label="ui('地址', 'Address')" min-width="180" show-overflow-tooltip />
      <el-table-column prop="errorMessage" :label="ui('错误', 'Error')" min-width="260" show-overflow-tooltip />
      <el-table-column prop="attemptedAt" :label="ui('最近尝试', 'Latest Attempt')" width="190" />
    </el-table>
    <el-empty v-else-if="diagnostic" :description="ui('近期没有失败点位', 'No recent failed points')" />
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import {
  exportCollectorConnectionDiagnostic,
  getCollectorConnectionDiagnostic,
} from '@/api/collector.api'
import type { CollectorConnectionDiagnostic } from '@/api/schemas/collector.schema'
import { getApiErrorMessage } from '@/utils/request'
import { datacenterLocale } from '@/i18n/runtime'

const ui = (zh: string, en: string) => (datacenterLocale.value === 'en' ? en : zh)

const props = defineProps<{ projectId: string; connectionId: string }>()
const loading = ref(false)
const exporting = ref(false)
const diagnostic = ref<CollectorConnectionDiagnostic | null>(null)
const successRate = computed(() => {
  const attempted = diagnostic.value?.attemptedPointCount || 0
  if (!attempted) return ui('暂无样本', 'No Samples')
  return `${Math.round(((diagnostic.value?.succeededPointCount || 0) / attempted) * 100)}%`
})
async function load() {
  loading.value = true
  try {
    diagnostic.value = await getCollectorConnectionDiagnostic(props.projectId, props.connectionId)
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, ui('读取连接诊断失败', 'Failed to load connection diagnostics')))
  } finally {
    loading.value = false
  }
}
async function exportCsv() {
  exporting.value = true
  try {
    const blob = await exportCollectorConnectionDiagnostic(props.projectId, props.connectionId)
    const url = URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = `collector-diagnostic-${props.connectionId}.csv`
    link.click()
    URL.revokeObjectURL(url)
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, ui('导出诊断失败', 'Failed to export diagnostics')))
  } finally {
    exporting.value = false
  }
}
function statusLabel(status: string) {
  const labels: Record<string, [string, string]> = {
    not_tested: ['未测试', 'Not Tested'],
    succeeded: ['成功', 'Succeeded'],
    failed: ['失败', 'Failed'],
  }
  const value = labels[status]
  return value ? ui(value[0], value[1]) : status || '—'
}
function agentReason(reason?: string | null) {
  const value = reason?.trim() || ''
  if (!value) return ui('调试代理不可用', 'Debug agent unavailable')
  const known: Record<string, string> = {
    未选择调试代理: 'No debug agent selected',
    调试代理离线: 'The debug agent is offline',
    调试代理不可用: 'The debug agent is unavailable',
  }
  return datacenterLocale.value === 'en' ? known[value] || value : value
}
watch(() => props.connectionId, load)
onMounted(load)
</script>

<style scoped>
.collector-diagnostic {
  padding: 8px 4px 24px;
}
.collector-diagnostic__head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 20px;
  margin-bottom: 18px;
}
.collector-diagnostic__head h3 {
  margin: 0;
  font-size: 16px;
}
.collector-diagnostic__head p {
  margin: 5px 0 0;
  color: var(--dc-text-muted);
  font-size: 12px;
}
.collector-diagnostic__table {
  margin-top: 18px;
}
</style>
