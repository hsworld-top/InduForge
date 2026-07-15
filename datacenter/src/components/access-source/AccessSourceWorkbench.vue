<template>
  <div class="asw-container">
    <!-- 主体：由协议子壳填充 -->
    <div class="asw-body">
      <component
        :is="resolvedPanel"
        :connection="connection"
        :project-id="projectId"
        @back="$emit('back')"
        @update-connection="$emit('update-connection')"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import SqlWorkbench from './workbench/SqlWorkbench.vue'
import MqttWorkbenchPanel from './workbench/MqttWorkbenchPanel.vue'
import KafkaWorkbenchPanel from './workbench/KafkaWorkbenchPanel.vue'
import HttpWorkbenchPanel from './workbench/http/HttpWorkbenchPanel.vue'
import WebSocketWorkbenchPanel from './workbench/websocket/WebSocketWorkbenchPanel.vue'
import RealtimeStoreWorkbench from './workbench/realtime/RealtimeStoreWorkbench.vue'
import IndustrialCollectorWorkbench from '@/components/collector-workbench/IndustrialCollectorWorkbench.vue'
import ReadOnlyConfigPanel from './workbench/ReadOnlyConfigPanel.vue'
import BuiltinRelationWorkbench from './workbench/BuiltinRelationWorkbench.vue'
import BuiltinTimeseriesWorkbench from './workbench/BuiltinTimeseriesWorkbench.vue'
import BuiltinMessageWorkbench from './workbench/BuiltinMessageWorkbench.vue'

type AccessSourceConnection = {
  id: string
  name?: string
  type?: string
  status?: string
  relationalConfig?: Record<string, any>
  mqttConfig?: Record<string, any>
  config?: Record<string, any>
  category?: string
}

const props = defineProps<{
  connection: AccessSourceConnection
  projectId: string
}>()

defineEmits<{
  (event: 'back'): void
  (event: 'update-connection'): void
}>()

/* 按协议类型选对应子壳 */
const resolvedPanel = computed(() => {
  const type = props.connection.type || ''
  if (type === 'collector' && props.connection.category === 'industrial')
    return IndustrialCollectorWorkbench
  if (type === 'builtin.relation') return BuiltinRelationWorkbench
  if (type === 'builtin.timeseries') return BuiltinTimeseriesWorkbench
  if (type === 'builtin.realtime') return RealtimeStoreWorkbench
  if (type === 'builtin.message') return BuiltinMessageWorkbench
  if (['relational', 'mysql', 'postgresql', 'sqlserver', 'tdengine'].includes(type)) {
    return SqlWorkbench
  }
  if (type === 'mqtt') {
    return MqttWorkbenchPanel
  }
  if (type === 'kafka') {
    return KafkaWorkbenchPanel
  }
  if (type === 'redis') {
    return RealtimeStoreWorkbench
  }
  if (type === 'http') {
    return HttpWorkbenchPanel
  }
  if (type === 'websocket') {
    return WebSocketWorkbenchPanel
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
