<template>
  <div class="logs">
    <div class="toolbar">
      <div class="toolbar-left">
        <el-button @click="$router.back()">
          <el-icon><ArrowLeft /></el-icon>
          返回
        </el-button>
      </div>
      <div class="toolbar-actions">
        <el-button @click="refreshLogs">
          <el-icon><Refresh /></el-icon>
          刷新
        </el-button>
        <el-button @click="downloadLogs">
          <el-icon><Download /></el-icon>
          下载
        </el-button>
      </div>
    </div>

    <!-- 日志控制 -->
    <el-card class="box-card">
      <div class="log-controls">
        <el-select v-model="logLevel" placeholder="选择日志级别" style="width: 150px;">
          <el-option label="全部" value="" />
          <el-option label="DEBUG" value="debug" />
          <el-option label="INFO" value="info" />
          <el-option label="WARN" value="warn" />
          <el-option label="ERROR" value="error" />
        </el-select>

        <el-input
          v-model="searchKeyword"
          placeholder="搜索日志内容"
          style="width: 300px; margin-left: 10px;"
          clearable
        >
          <template #prefix>
            <el-icon><Search /></el-icon>
          </template>
        </el-input>

        <el-switch
          v-model="autoRefresh"
          active-text="自动刷新"
          style="margin-left: 10px;"
        />
      </div>
    </el-card>

    <!-- 日志内容 -->
    <el-card class="box-card log-content">
      <div v-loading="loading" class="log-viewer">
        <div v-if="filteredLogs.length === 0 && !loading" class="empty-state">
          <el-empty description="暂无日志数据" />
        </div>
        <div v-else class="log-list">
          <div
            v-for="(log, index) in filteredLogs"
            :key="index"
            class="log-item"
            :class="`log-${log.level?.toLowerCase() || 'info'}`"
          >
            <span class="log-time">{{ formatLogTime(log.time) }}</span>
            <span class="log-level">{{ log.level }}</span>
            <span class="log-message">{{ log.message }}</span>
          </div>
        </div>
      </div>
    </el-card>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useNodeStore } from '@/store/nodeStore'
import { ElMessage } from 'element-plus'
import { ArrowLeft, Refresh, Download, Search } from '@element-plus/icons-vue'

const route = useRoute()
const nodeStore = useNodeStore()

const projectId = ref(route.params.projectId)
const logs = ref([])
const loading = ref(false)
const logLevel = ref('')
const searchKeyword = ref('')
const autoRefresh = ref(true)
let refreshTimer = null

const filteredLogs = computed(() => {
  let filtered = logs.value

  // 按级别过滤
  if (logLevel.value) {
    filtered = filtered.filter(log =>
      log.level?.toLowerCase() === logLevel.value.toLowerCase()
    )
  }

  // 按关键词搜索
  if (searchKeyword.value) {
    filtered = filtered.filter(log =>
      log.message?.toLowerCase().includes(searchKeyword.value.toLowerCase())
    )
  }

  return filtered
})

const loadLogs = async () => {
  if (!projectId.value) return

  loading.value = true
  try {
    const data = await nodeStore.getProjectLogs(projectId.value)
    logs.value = data || []
  } catch (error) {
    console.error('获取日志失败:', error)
    ElMessage.error('获取日志失败')
  } finally {
    loading.value = false
  }
}

const refreshLogs = async () => {
  await loadLogs()
  ElMessage.success('日志刷新成功')
}

const downloadLogs = () => {
  const content = filteredLogs.value
    .map(log => `[${log.time}] ${log.level}: ${log.message}`)
    .join('\n')

  const blob = new Blob([content], { type: 'text/plain' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `${projectId.value}_logs_${new Date().toISOString().split('T')[0]}.txt`
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)

  ElMessage.success('日志下载成功')
}

const formatLogTime = (time) => {
  if (!time) return ''
  return new Date(time).toLocaleString()
}

// 自动刷新
watch(autoRefresh, (newVal) => {
  if (newVal) {
    refreshTimer = setInterval(loadLogs, 5000)
  } else {
    if (refreshTimer) {
      clearInterval(refreshTimer)
      refreshTimer = null
    }
  }
})

onMounted(async () => {
  await loadLogs()
  if (autoRefresh.value) {
    refreshTimer = setInterval(loadLogs, 5000)
  }
})

onUnmounted(() => {
  if (refreshTimer) {
    clearInterval(refreshTimer)
  }
})
</script>

<style scoped>
.logs {
  display: grid;
  gap: 16px;
}

.toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.toolbar-left {
  display: flex;
  align-items: center;
}

.toolbar-actions {
  display: flex;
  gap: 10px;
}

.box-card {
  margin-bottom: 0;
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  box-shadow: var(--shadow-sm);
}

.log-controls {
  display: flex;
  align-items: center;
  gap: 10px;
}

.log-content {
  min-height: 460px;
}

.log-viewer {
  height: 100%;
  overflow-y: auto;
  background-color: #1e1e1e;
  color: #d4d4d4;
  font-family: 'Courier New', monospace;
  font-size: 13px;
  padding: 15px;
  border-radius: 4px;
}

.log-list {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.log-item {
  display: flex;
  gap: 10px;
  padding: 2px 0;
  line-height: 1.5;
}

.log-time {
  color: #858585;
  white-space: nowrap;
  min-width: 160px;
}

.log-level {
  font-weight: bold;
  min-width: 60px;
  text-transform: uppercase;
}

.log-message {
  flex: 1;
  word-break: break-all;
}

.log-debug .log-level {
  color: #808080;
}

.log-info .log-level {
  color: #4fc3f7;
}

.log-warn .log-level {
  color: #ffb74d;
}

.log-error .log-level {
  color: #f06292;
}

.empty-state {
  display: flex;
  justify-content: center;
  align-items: center;
  height: 100%;
}
</style>
