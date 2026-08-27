<template>
  <section class="collector-diagnostic" v-loading="loading">
    <header class="collector-diagnostic__head">
      <div>
        <h3>连接诊断</h3>
        <p>只展示开发态代理和最近调试摘要，不代表节点运行状态。</p>
      </div>
      <div>
        <el-button @click="load">刷新</el-button>
        <el-button :loading="exporting" @click="exportCsv">导出失败点 CSV</el-button>
      </div>
    </header>
    <el-alert
      v-if="diagnostic && !diagnostic.agent.ready"
      :title="diagnostic.agent.reason || '调试代理不可用'"
      type="warning"
      show-icon
      :closable="false"
    />
    <el-descriptions v-if="diagnostic" :column="3" border>
      <el-descriptions-item label="代理">{{
        diagnostic.agent.name || '未选择'
      }}</el-descriptions-item>
      <el-descriptions-item label="最近测试">{{ diagnostic.lastTest.status }}</el-descriptions-item>
      <el-descriptions-item label="往返耗时">{{
        diagnostic.lastTest.durationMs ? `${diagnostic.lastTest.durationMs} ms` : '—'
      }}</el-descriptions-item>
      <el-descriptions-item label="点位数">{{ diagnostic.pointCount }}</el-descriptions-item>
      <el-descriptions-item label="近期尝试">{{
        diagnostic.attemptedPointCount
      }}</el-descriptions-item>
      <el-descriptions-item label="读取成功率">{{ successRate }}</el-descriptions-item>
      <el-descriptions-item label="最近错误" :span="3">{{
        diagnostic.lastTest.message || '无'
      }}</el-descriptions-item>
    </el-descriptions>
    <el-table
      v-if="diagnostic?.failedPoints.length"
      :data="diagnostic.failedPoints"
      class="collector-diagnostic__table"
    >
      <el-table-column prop="name" label="失败点位" min-width="160" show-overflow-tooltip />
      <el-table-column prop="addressText" label="地址" min-width="180" show-overflow-tooltip />
      <el-table-column prop="errorMessage" label="错误" min-width="260" show-overflow-tooltip />
      <el-table-column prop="attemptedAt" label="最近尝试" width="190" />
    </el-table>
    <el-empty v-else-if="diagnostic" description="近期没有失败点位" />
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

const props = defineProps<{ projectId: string; connectionId: string }>()
const loading = ref(false)
const exporting = ref(false)
const diagnostic = ref<CollectorConnectionDiagnostic | null>(null)
const successRate = computed(() => {
  const attempted = diagnostic.value?.attemptedPointCount || 0
  if (!attempted) return '暂无样本'
  return `${Math.round(((diagnostic.value?.succeededPointCount || 0) / attempted) * 100)}%`
})
async function load() {
  loading.value = true
  try {
    diagnostic.value = await getCollectorConnectionDiagnostic(props.projectId, props.connectionId)
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '读取连接诊断失败'))
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
    ElMessage.error(getApiErrorMessage(error, '导出诊断失败'))
  } finally {
    exporting.value = false
  }
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
