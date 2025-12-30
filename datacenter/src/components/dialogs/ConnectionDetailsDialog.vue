<template>
  <el-dialog v-model="visible" title="连接详情" width="600px">
    <el-descriptions :column="1" border v-if="connection">
      <el-descriptions-item label="连接名称">{{ connection.name }}</el-descriptions-item>
      <el-descriptions-item label="连接类型">{{ getConnectionTypeLabel(connection.type) }}</el-descriptions-item>
      <el-descriptions-item label="连接状态">
        <el-tag
          :type="
            connection.status === 'connected' ? 'success' :
            connection.status === 'error' ? 'danger' :
            connection.status === 'disconnected' ? 'warning' : 'info'
          "
        >
          {{
            connection.status === 'connected' ? '已连接' :
            connection.status === 'error' ? '连接错误' :
            connection.status === 'disconnected' ? '已断开' : '未知状态'
          }}
        </el-tag>
      </el-descriptions-item>
      <template v-if="connection.type === 'relational' && connection.relationalConfig">
        <el-descriptions-item label="数据库类型">{{ connection.relationalConfig.dbType }}</el-descriptions-item>
        <el-descriptions-item label="主机地址">{{ connection.relationalConfig.host }}</el-descriptions-item>
        <el-descriptions-item label="端口">{{ connection.relationalConfig.port }}</el-descriptions-item>
        <el-descriptions-item label="数据库名">{{ connection.relationalConfig.database }}</el-descriptions-item>
        <el-descriptions-item label="用户名">{{ connection.relationalConfig.username }}</el-descriptions-item>
      </template>
      <template v-if="connection.type === 'mqtt' && connection.mqttConfig">
        <el-descriptions-item label="协议">{{ connection.mqttConfig.protocol }}</el-descriptions-item>
        <el-descriptions-item label="Broker 地址">{{ connection.mqttConfig.brokerUrl }}</el-descriptions-item>
        <el-descriptions-item label="端口">{{ connection.mqttConfig.port }}</el-descriptions-item>
        <el-descriptions-item label="客户端 ID">{{ connection.mqttConfig.clientId || '自动生成' }}</el-descriptions-item>
        <el-descriptions-item label="用户名">{{ connection.mqttConfig.username || '-' }}</el-descriptions-item>
        <el-descriptions-item label="QoS">{{ connection.mqttConfig.qos }}</el-descriptions-item>
        <el-descriptions-item label="保持连接">{{ connection.mqttConfig.keepalive }}秒</el-descriptions-item>
        <el-descriptions-item label="清除会话">{{ connection.mqttConfig.cleanSession ? '是' : '否' }}</el-descriptions-item>
      </template>
      <el-descriptions-item label="创建时间">{{ formatDate(connection.createdAt) }}</el-descriptions-item>
      <el-descriptions-item label="更新时间">{{ formatDate(connection.updatedAt) }}</el-descriptions-item>
    </el-descriptions>
    <template #footer>
      <el-button @click="visible = false">关闭</el-button>
    </template>
  </el-dialog>
</template>

<script setup>
import { computed } from 'vue'
import { getConnectionTypeConfig } from '@/config/connectionTypes'

const props = defineProps({
  modelValue: {
    type: Boolean,
    default: false
  },
  connection: {
    type: Object,
    default: null
  }
})

const emit = defineEmits(['update:modelValue'])

const visible = computed({
  get: () => props.modelValue,
  set: (val) => emit('update:modelValue', val)
})

const getConnectionTypeLabel = (type) => {
  if (type === 'relational' && props.connection?.relationalConfig) {
    const dbType = props.connection.relationalConfig.dbType
    const config = getConnectionTypeConfig(dbType)
    return config ? config.label : dbType
  }
  const labels = {
    relational: '关系数据库',
    mqtt: 'MQTT',
    websocket: 'WebSocket',
    opcua: 'OPC UA',
    http: 'HTTP'
  }
  return labels[type] || type
}

const formatDate = (dateString) => {
  if (!dateString) return '-'
  const date = new Date(dateString)
  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit'
  })
}
</script>
