<template>
  <div class="collector-discovery">
    <div class="collector-discovery__toolbar">
      <el-input v-model="parentNodeId" placeholder="父节点 NodeId" /><el-button
        data-test="browse-device"
        type="primary"
        :disabled="!enabled"
        :loading="loading"
        @click="browse"
        >浏览设备</el-button
      ><el-button :disabled="!selected.length" @click="emitPoints">保存为点位</el-button>
    </div>
    <el-alert
      v-if="!enabled"
      title="当前未选择兼容 Agent，仍可离线编辑连接和点位"
      type="info"
      :closable="false"
      show-icon
    />
    <el-table :data="nodes" height="100%" @selection-change="selected = $event"
      ><el-table-column type="selection" width="44" /><el-table-column
        prop="displayName"
        label="显示名"
        min-width="180" /><el-table-column
        prop="nodeId"
        label="NodeId"
        min-width="260" /><el-table-column prop="nodeClass" label="类型" width="100"
    /></el-table>
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
  enabled: boolean
}>()
const emit = defineEmits<{ points: [points: Record<string, unknown>[]] }>()
type BrowseNode = {
  nodeId: string
  displayName: string
  browseName: string
  nodeClass: string
  dataType?: string
  hasChildren: boolean
}
const parentNodeId = ref('ns=0;i=85')
const nodes = ref<BrowseNode[]>([])
const selected = ref<BrowseNode[]>([])
const loading = ref(false)
async function waitTask(taskId: string) {
  for (let index = 0; index < 30; index++) {
    const task = await getCollectorTask(props.projectId, taskId)
    if (task.status === 'succeeded') return task
    if (['failed', 'cancelled', 'expired'].includes(task.status))
      throw new Error(task.errorMessage || '设备浏览失败')
    await new Promise((resolve) => setTimeout(resolve, 1000))
  }
  throw new Error('设备浏览超时')
}
async function browse() {
  if (!props.agentId) return
  loading.value = true
  try {
    const task = await createCollectorTask(props.projectId, {
      agentId: props.agentId,
      connectionId: props.connectionId,
      operation: 'device.browse',
      input: { parentNodeId: parentNodeId.value, maxDepth: 1 },
    })
    const completed = await waitTask(task.taskId)
    nodes.value = (completed.result as { nodes?: BrowseNode[] } | null)?.nodes || []
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '设备浏览失败')
  } finally {
    loading.value = false
  }
}
function emitPoints() {
  emit(
    'points',
    selected.value
      .filter((node) => node.nodeClass === 'variable')
      .map((node, index) => ({
        groupId: null,
        code: `node_${Date.now()}_${index + 1}`,
        name: node.displayName || node.browseName,
        address: { nodeId: node.nodeId },
        dataType: node.dataType || 'float64',
        elementCount: 1,
        readOptions: {},
        acquisition: { mode: 'polling', intervalMs: 1000 },
        enabled: true,
        sortOrder: index,
        metadata: {},
      })),
  )
}
</script>

<style scoped>
.collector-discovery {
  display: flex;
  height: 100%;
  min-height: 0;
  flex-direction: column;
  gap: 12px;
}
.collector-discovery__toolbar {
  display: grid;
  grid-template-columns: 1fr auto auto;
  gap: 10px;
}
.collector-discovery :deep(.el-table) {
  flex: 1;
}
</style>
