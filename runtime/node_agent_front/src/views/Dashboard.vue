<template>
  <div class="dashboard">
    <!-- 头部 -->
    <div class="header">
      <h1>NodeAgent 管理控制台</h1>
      <div class="header-actions">
        <el-button type="primary" @click="refresh">
          <el-icon><Refresh /></el-icon>
          刷新
        </el-button>
      </div>
    </div>

    <!-- 节点信息卡片 -->
    <el-card class="box-card">
      <template #header>
        <div class="card-header">
          <span>节点信息</span>
        </div>
      </template>
      <div class="node-info">
        <div class="info-item">
          <span class="label">节点ID:</span>
          <span class="value">{{ nodeStore.nodeInfo?.id || '-' }}</span>
        </div>
        <div class="info-item">
          <span class="label">版本:</span>
          <span class="value">{{ nodeStore.nodeInfo?.version || '-' }}</span>
        </div>
        <div class="info-item">
          <span class="label">执行器类型:</span>
          <span class="value">{{ nodeStore.nodeInfo?.executorType || '-' }}</span>
        </div>
        <div class="info-item">
          <span class="label">工作目录:</span>
          <span class="value">{{ nodeStore.nodeInfo?.workDir || '-' }}</span>
        </div>
      </div>
    </el-card>

    <!-- 项目列表 -->
    <el-card class="box-card">
      <template #header>
        <div class="card-header">
          <span>运行中的项目 ({{ nodeStore.projects.length }})</span>
        </div>
      </template>
      <div v-loading="nodeStore.loading" class="projects-list">
        <div v-if="nodeStore.projects.length === 0" class="empty-state">
          <el-empty description="暂无运行项目" />
        </div>
        <div v-else class="projects-grid">
          <div v-for="project in nodeStore.projects" :key="project.id" class="project-card">
            <div class="project-header">
              <h3>{{ project.id }}</h3>
              <el-tag :type="getStatusType(project.status)">
                {{ project.status }}
              </el-tag>
            </div>
            <div class="project-info">
              <p><strong>版本:</strong> {{ project.currentVersion || '-' }}</p>
              <p><strong>状态:</strong> {{ project.runtimeStatus?.state || 'stopped' }}</p>
              <p v-if="project.runtimeStatus?.pid"><strong>PID:</strong> {{ project.runtimeStatus.pid }}</p>
            </div>
            <div class="project-actions">
              <el-button
                v-if="project.status !== 'running'"
                type="primary"
                size="small"
                @click="startProject(project.id)"
              >
                启动
              </el-button>
              <el-button
                v-else
                type="warning"
                size="small"
                @click="stopProject(project.id)"
              >
                停止
              </el-button>
              <el-button
                type="info"
                size="small"
                @click="restartProject(project.id)"
              >
                重启
              </el-button>
              <el-button
                type="success"
                size="small"
                @click="viewLogs(project.id)"
              >
                日志
              </el-button>
            </div>
          </div>
        </div>
      </div>
    </el-card>
  </div>
</template>

<script setup>
import { onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useNodeStore } from '@/store/nodeStore'
import { ElMessage } from 'element-plus'
import { Refresh } from '@element-plus/icons-vue'

const router = useRouter()
const nodeStore = useNodeStore()

onMounted(async () => {
  await nodeStore.fetchNodeInfo()
  await nodeStore.fetchProjects()
})

const refresh = async () => {
  await nodeStore.fetchNodeInfo()
  await nodeStore.fetchProjects()
  ElMessage.success('刷新成功')
}

const startProject = async (projectId) => {
  await nodeStore.startProject(projectId)
}

const stopProject = async (projectId) => {
  await nodeStore.stopProject(projectId)
}

const restartProject = async (projectId) => {
  await nodeStore.restartProject(projectId)
}

const viewLogs = (projectId) => {
  router.push(`/logs/${projectId}`)
}

const getStatusType = (status) => {
  const map = {
    running: 'success',
    stopped: 'info',
    error: 'danger',
    deploying: 'warning',
  }
  return map[status] || 'info'
}
</script>

<style scoped>
.dashboard {
  min-height: 100vh;
  padding: 20px;
  background-color: #f5f7fa;
}

.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.header h1 {
  font-size: 24px;
  color: #303133;
}

.box-card {
  margin-bottom: 20px;
}

.card-header {
  font-weight: 600;
  color: #303133;
}

.node-info {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
  gap: 20px;
}

.info-item {
  display: flex;
  flex-direction: column;
}

.info-item .label {
  font-size: 14px;
  color: #909399;
  margin-bottom: 4px;
}

.info-item .value {
  font-size: 16px;
  color: #303133;
  font-weight: 500;
}

.projects-list {
  min-height: 200px;
}

.empty-state {
  padding: 40px 0;
}

.projects-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: 16px;
}

.project-card {
  border: 1px solid #ebeef5;
  border-radius: 8px;
  padding: 16px;
  background-color: #fff;
  transition: all 0.3s;
}

.project-card:hover {
  box-shadow: 0 2px 12px 0 rgba(0, 0, 0, 0.1);
}

.project-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.project-header h3 {
  font-size: 16px;
  color: #303133;
  margin: 0;
}

.project-info {
  margin-bottom: 12px;
}

.project-info p {
  margin: 4px 0;
  font-size: 14px;
  color: #606266;
}

.project-actions {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}
</style>
