<template>
  <section class="builtin-workbench">
    <WorkbenchSourceHeader
      :title="connection.name || 'IF实时库'"
      fallback-title="IF实时库"
      status-label="开发态测试"
      status-tone="info"
      :meta="sourceMetaRows"
      @back="$emit('back')"
    />
    <div class="builtin-workbench__body">
      <div class="builtin-workbench__grid">
        <el-input v-model="form.key" placeholder="current/temp" />
        <el-input v-model="form.value" placeholder="36.5" />
        <el-input-number v-model="form.ttlSeconds" :min="0" :max="86400" />
      </div>
      <div class="builtin-workbench__actions">
        <el-button type="primary" :loading="loading" @click="setKey">写入</el-button>
        <el-button :loading="loading" @click="getKey">读取</el-button>
        <el-button :loading="loading" @click="deleteKey">删除</el-button>
      </div>
      <el-table v-if="keys.length" :data="keys" size="small" border height="220">
        <el-table-column prop="key" label="Key" min-width="180" show-overflow-tooltip />
        <el-table-column prop="valueType" label="类型" width="100" />
        <el-table-column prop="defaultTtlSeconds" label="默认 TTL" width="110" />
        <el-table-column prop="description" label="说明" min-width="160" show-overflow-tooltip />
      </el-table>
      <el-alert v-if="errorMessage" type="error" :closable="false" :title="errorMessage" />
      <pre v-if="currentValue" class="builtin-workbench__pre">{{ currentValue }}</pre>
      <el-empty v-else description="暂无当前值" :image-size="80" />
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import dataAPI from '@/api/data.api'
import { getApiErrorMessage } from '@/utils/request'
import WorkbenchSourceHeader from '@/components/workbench/WorkbenchSourceHeader.vue'

const props = defineProps<{
  connection: Record<string, any>
  projectId: string
}>()

defineEmits<{
  (event: 'back'): void
}>()

const form = reactive({ key: 'current/temp', value: '36.5', ttlSeconds: 300 })
const loading = ref(false)
const errorMessage = ref('')
const currentValue = ref('')
const keys = ref<any[]>([])
const sourceMetaRows = computed(() => [
  { label: '类型', value: 'IF实时库' },
  {
    label: '标识',
    value: String(props.connection.config?.runtimeKey || props.connection.id || '-'),
  },
])

const loadKeys = async () => {
  try {
    const response = await dataAPI.getBuiltinRealtimeKeys(props.projectId, props.connection.id)
    keys.value = response?.data?.list || []
  } catch {
    keys.value = []
  }
}

const setKey = async () => {
  await run(async () => {
    const response = await dataAPI.setBuiltinRealtimeKey(props.projectId, props.connection.id, {
      key: form.key,
      value: form.value,
      ttlSeconds: form.ttlSeconds,
    })
    currentValue.value = JSON.stringify(response?.data || {}, null, 2)
    await loadKeys()
  }, '写入 IF实时库失败')
}

const getKey = async () => {
  await run(async () => {
    const response = await dataAPI.getBuiltinRealtimeKey(
      props.projectId,
      props.connection.id,
      form.key,
    )
    currentValue.value = JSON.stringify(response?.data || {}, null, 2)
  }, '读取 IF实时库失败')
}

const deleteKey = async () => {
  await run(async () => {
    await dataAPI.deleteBuiltinRealtimeKey(props.projectId, props.connection.id, form.key)
    currentValue.value = ''
    await loadKeys()
  }, '删除 IF实时库失败')
}

const run = async (fn: () => Promise<void>, fallback: string) => {
  loading.value = true
  errorMessage.value = ''
  try {
    await fn()
  } catch (error) {
    errorMessage.value = getApiErrorMessage(error, fallback)
  } finally {
    loading.value = false
  }
}

onMounted(loadKeys)
</script>

<style scoped>
@import './builtin-workbench.css';
</style>
