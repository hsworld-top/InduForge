<template>
  <el-select
    :model-value="modelValue"
    clearable
    placeholder="选择调试代理"
    style="width: 240px"
    @update:model-value="selectAgent"
  >
    <el-option
      v-for="agent in agents"
      :key="agent.id"
      :label="formatCollectorAgentName(agent)"
      :value="agent.id"
      :disabled="agent.status !== 'online'"
    >
      <span>{{ formatCollectorAgentName(agent) }}</span>
      <span class="collector-agent-option">{{ agent.status === 'online' ? '在线' : '离线' }}</span>
    </el-option>
  </el-select>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { getCollectorAgents } from '@/api/collector-dev.api'
import type { CollectorAgent } from '@/api/schemas/collector-dev.schema'
import { collectorAgentStorageKey, formatCollectorAgentName } from './collector-workbench-model'

const props = defineProps<{ modelValue?: string; projectId: string }>()
const emit = defineEmits<{
  'update:modelValue': [id: string]
  change: [agent: CollectorAgent | undefined]
  loaded: [agents: CollectorAgent[]]
}>()
const agents = ref<CollectorAgent[]>([])
let refreshTimer: number | undefined
let loading = false

async function load() {
  if (loading || document.visibilityState !== 'visible') return
  loading = true
  try {
    const result = await getCollectorAgents({ page: 1, pageSize: 100 })
    agents.value = result.list.filter((agent) => agent.status !== 'invalid')
    emit('loaded', agents.value)
    const remembered = localStorage.getItem(collectorAgentStorageKey(props.projectId)) || ''
    const selected = props.modelValue || remembered
    if (!selected) return
    const selectedAgent = agents.value.find((agent) => agent.id === selected)
    if (!selectedAgent) {
      selectAgent('')
      return
    }
    emit('update:modelValue', selected)
    emit('change', selectedAgent)
  } catch {
    // 静默刷新失败时保留当前选择，等待下一次轮询恢复。
  } finally {
    loading = false
  }
}
function selectAgent(id: string) {
  if (id) localStorage.setItem(collectorAgentStorageKey(props.projectId), id)
  else localStorage.removeItem(collectorAgentStorageKey(props.projectId))
  emit('update:modelValue', id)
  emit(
    'change',
    agents.value.find((agent) => agent.id === id),
  )
}
function handleVisibilityChange() {
  if (document.visibilityState === 'visible') void load()
}
onMounted(() => {
  void load()
  refreshTimer = window.setInterval(() => void load(), 5000)
  document.addEventListener('visibilitychange', handleVisibilityChange)
})
onBeforeUnmount(() => {
  if (refreshTimer !== undefined) window.clearInterval(refreshTimer)
  document.removeEventListener('visibilitychange', handleVisibilityChange)
})
defineExpose({ reload: load })
</script>

<style scoped>
.collector-agent-option {
  float: right;
  margin-left: 20px;
  color: var(--dc-text-muted);
  font-size: 12px;
}
</style>
