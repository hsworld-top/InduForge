<template>
  <el-select
    :model-value="modelValue"
    clearable
    placeholder="选择调试代理"
    style="width: 240px"
    @update:model-value="selectAgent"
  >
    <el-option v-for="agent in agents" :key="agent.id" :value="agent.id" :disabled="!agent.online">
      <span>{{ agent.name }}</span
      ><span class="collector-agent-option">{{ agent.online ? '在线' : '离线' }}</span>
    </el-option>
  </el-select>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { getCollectorAgents } from '@/api/collector-dev.api'
import type { CollectorAgent } from '@/api/schemas/collector-dev.schema'
import { collectorAgentStorageKey } from './collector-workbench-model'

const props = defineProps<{ modelValue?: string; projectId: string }>()
const emit = defineEmits<{
  'update:modelValue': [id: string]
  change: [agent: CollectorAgent | undefined]
  loaded: [agents: CollectorAgent[]]
}>()
const agents = ref<CollectorAgent[]>([])

async function load() {
  agents.value = (await getCollectorAgents({ page: 1, pageSize: 100 })).list
  emit('loaded', agents.value)
  const remembered = localStorage.getItem(collectorAgentStorageKey(props.projectId)) || ''
  const selected = props.modelValue || remembered
  if (selected && agents.value.some((agent) => agent.id === selected)) selectAgent(selected)
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
onMounted(load)
</script>

<style scoped>
.collector-agent-option {
  float: right;
  margin-left: 20px;
  color: #87929a;
  font-size: 12px;
}
</style>
