<template>
  <section class="collector-diagnostics">
    <header class="collector-diagnostics__header">
      <div>
        <strong>连接诊断</strong>
        <p>通过当前采集调试代理验证设备地址、协议握手和驱动配置。</p>
      </div>
      <el-button type="primary" :disabled="!enabled" :loading="loading" @click="emit('run')">
        测试连接
      </el-button>
    </header>

    <el-alert
      v-if="!enabled"
      title="请选择支持 connection.test 的在线 Agent"
      type="info"
      :closable="false"
      show-icon
    />

    <div class="collector-diagnostics__result" :class="`is-${resultTone}`">
      <div class="collector-diagnostics__result-head">
        <span><i />{{ resultTitle }}</span>
        <time v-if="task?.finishedAt || task?.createdAt">{{
          task.finishedAt || task.createdAt
        }}</time>
      </div>
      <p>{{ resultMessage }}</p>
      <pre v-if="task?.result">{{ JSON.stringify(task.result, null, 2) }}</pre>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { CollectorTask } from '@/api/schemas/collector.schema'

const props = defineProps<{
  enabled: boolean
  loading: boolean
  task: CollectorTask | null
}>()
const emit = defineEmits<{ run: [] }>()
const resultTone = computed(() => {
  if (!props.task) return 'idle'
  if (props.task.status === 'succeeded') return 'success'
  if (['failed', 'cancelled', 'expired'].includes(props.task.status)) return 'danger'
  return 'running'
})
const resultTitle = computed(() => {
  if (!props.task) return '尚未执行连接测试'
  if (props.task.status === 'succeeded') return '连接测试通过'
  if (['failed', 'cancelled', 'expired'].includes(props.task.status)) return '连接测试失败'
  return '正在执行连接测试'
})
const resultMessage = computed(() => {
  if (!props.task) return '执行后将在这里显示驱动返回的连接耗时和服务端信息。'
  if (props.task.status === 'succeeded') return '当前配置已通过采集调试代理验证。'
  return props.task.errorMessage || '任务已提交，正在等待采集调试代理返回结果。'
})
</script>

<style scoped>
.collector-diagnostics {
  display: flex;
  height: 100%;
  flex-direction: column;
  gap: 16px;
}
.collector-diagnostics__header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 24px;
  padding: 18px 20px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-md);
  background: var(--dc-surface-raised);
}
.collector-diagnostics__header strong {
  color: var(--dc-text);
  font-size: 15px;
}
.collector-diagnostics__header p,
.collector-diagnostics__result p {
  margin: 6px 0 0;
  color: var(--dc-text-muted);
  font-size: 12px;
}
.collector-diagnostics__result {
  min-height: 180px;
  padding: 20px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-md);
  background: var(--dc-surface-raised);
}
.collector-diagnostics__result.is-success {
  border-color: color-mix(in srgb, var(--dc-success) 35%, var(--dc-border));
}
.collector-diagnostics__result.is-danger {
  border-color: color-mix(in srgb, var(--dc-danger) 35%, var(--dc-border));
}
.collector-diagnostics__result-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}
.collector-diagnostics__result-head span {
  display: flex;
  align-items: center;
  gap: 8px;
  color: var(--dc-text-secondary);
  font-size: 14px;
  font-weight: 700;
}
.collector-diagnostics__result-head i {
  width: 9px;
  height: 9px;
  border-radius: 50%;
  background: var(--dc-border-strong);
}
.is-running .collector-diagnostics__result-head i {
  background: var(--dc-primary);
}
.is-success .collector-diagnostics__result-head i {
  background: var(--dc-success);
}
.is-danger .collector-diagnostics__result-head i {
  background: var(--dc-danger);
}
.collector-diagnostics__result-head time {
  color: var(--dc-text-muted);
  font-size: 11px;
}
.collector-diagnostics pre {
  max-height: 320px;
  overflow: auto;
  margin: 18px 0 0;
  padding: 16px;
  border-radius: var(--dc-radius-sm);
  background: #10191f;
  color: #b9e5d4;
  font: 12px/1.65 var(--dc-font-mono);
}
</style>
