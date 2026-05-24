<template>
  <div class="asw-container">
    <!-- 主体：由协议子壳填充 -->
    <div class="asw-body">
      <component
        :is="resolvedPanel"
        :connection="connection"
        :project-id="projectId"
        @back="$emit('back')"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import SqlWorkbench from './workbench/SqlWorkbench.vue'
import MqttWorkbenchPanel from './workbench/MqttWorkbenchPanel.vue'
import ProtocolWorkbenchPanel from './workbench/ProtocolWorkbenchPanel.vue'
import RedisManagerWorkbench from './workbench/RedisManagerWorkbench.vue'
import IndustrialProtocolWorkbench from './workbench/IndustrialProtocolWorkbench.vue'
import ReadOnlyConfigPanel from './workbench/ReadOnlyConfigPanel.vue'
import BuiltinRelationWorkbench from './workbench/BuiltinRelationWorkbench.vue'
import BuiltinTimeseriesWorkbench from './workbench/BuiltinTimeseriesWorkbench.vue'
import BuiltinRealtimeWorkbench from './workbench/BuiltinRealtimeWorkbench.vue'
import BuiltinMessageWorkbench from './workbench/BuiltinMessageWorkbench.vue'

type AccessSourceConnection = {
  id: string
  name?: string
  type?: string
  status?: string
  relationalConfig?: Record<string, any>
  mqttConfig?: Record<string, any>
  config?: Record<string, any>
}

const props = defineProps<{
  connection: AccessSourceConnection
  projectId: string
}>()

defineEmits<{
  (event: 'back'): void
}>()

/* 按协议类型选对应子壳 */
const resolvedPanel = computed(() => {
  const type = props.connection.type || ''
  if (type === 'builtin.relation') return BuiltinRelationWorkbench
  if (type === 'builtin.timeseries') return BuiltinTimeseriesWorkbench
  if (type === 'builtin.realtime') return BuiltinRealtimeWorkbench
  if (type === 'builtin.message') return BuiltinMessageWorkbench
  if (['relational', 'mysql', 'postgresql', 'sqlserver', 'tdengine'].includes(type)) {
    return SqlWorkbench
  }
  if (type === 'mqtt') {
    return MqttWorkbenchPanel
  }
  if (type === 'redis') {
    return RedisManagerWorkbench
  }
  if (['opcua', 'modbus'].includes(type)) {
    return IndustrialProtocolWorkbench
  }
  if (['kafka', 'http', 'websocket'].includes(type)) {
    return ProtocolWorkbenchPanel
  }
  if (['opcua', 'opcda', 's7', 'modbus'].includes(type)) {
    return ReadOnlyConfigPanel
  }
  return ReadOnlyConfigPanel
})
</script>

<style scoped>
.asw-container {
  height: 100%;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

/* 主体区撑满全部高度 */
.asw-body {
  flex: 1;
  min-height: 0;
  overflow: hidden;
}
</style>
