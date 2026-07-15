<template>
  <div class="collector-live-debug">
    <div class="collector-live-debug__toolbar">
      <span>已选择 {{ pointIds.length }} 个已保存点位</span
      ><el-button
        type="primary"
        :disabled="!enabled || !pointIds.length"
        :loading="loading"
        @click="read"
        >读取实时值</el-button
      >
    </div>
    <el-alert
      v-if="!enabled"
      title="请选择支持 point.read 的在线 Agent"
      type="info"
      :closable="false"
    />
    <pre>{{ output }}</pre>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { ElMessage } from 'element-plus'
import { createCollectorTask, getCollectorTask } from '@/api/collector.api'
const props = defineProps<{
  projectId: string
  connectionId: string
  agentId?: string
  pointIds: string[]
  enabled: boolean
}>()
const loading = ref(false)
const output = ref('请选择点位后执行读取')
async function read() {
  if (!props.agentId) return
  loading.value = true
  try {
    const task = await createCollectorTask(props.projectId, {
      agentId: props.agentId,
      connectionId: props.connectionId,
      operation: 'point.read',
      input: { pointIds: props.pointIds },
    })
    for (let index = 0; index < 30; index++) {
      const current = await getCollectorTask(props.projectId, task.taskId)
      if (current.status === 'succeeded') {
        output.value = JSON.stringify(current.result, null, 2)
        return
      }
      if (['failed', 'cancelled', 'expired'].includes(current.status))
        throw new Error(current.errorMessage || '读取失败')
      await new Promise((resolve) => setTimeout(resolve, 1000))
    }
    throw new Error('读取任务超时')
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '读取失败')
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.collector-live-debug {
  display: flex;
  height: 100%;
  flex-direction: column;
  gap: 14px;
}
.collector-live-debug__toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.collector-live-debug pre {
  flex: 1;
  overflow: auto;
  margin: 0;
  padding: 18px;
  border: 1px solid #dfe6ea;
  border-radius: 8px;
  background: #10191f;
  color: #b9e5d4;
  font:
    12px/1.65 Consolas,
    monospace;
}
</style>
