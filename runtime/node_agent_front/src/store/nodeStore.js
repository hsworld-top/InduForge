import { defineStore } from 'pinia'
import { ref } from 'vue'
import { nodeApi } from '@/api/nodeApi'

export const useNodeStore = defineStore('node', () => {
  const nodeInfo = ref(null)
  const projects = ref([])
  const loading = ref(false)

  // 获取节点信息
  const fetchNodeInfo = async () => {
    try {
      const data = await nodeApi.getNodeInfo()
      nodeInfo.value = data
    } catch (error) {
      console.error('获取节点信息失败:', error)
    }
  }

  // 获取项目列表
  const fetchProjects = async () => {
    loading.value = true
    try {
      const data = await nodeApi.getProjects()
      projects.value = data
    } catch (error) {
      console.error('获取项目列表失败:', error)
      ElMessage.error('获取项目列表失败')
    } finally {
      loading.value = false
    }
  }

  // 部署项目
  const deployProject = async (projectId, data) => {
    try {
      await nodeApi.deployProject(projectId, data)
      ElMessage.success('部署成功')
      await fetchProjects()
      return true
    } catch (error) {
      console.error('部署失败:', error)
      ElMessage.error('部署失败')
      return false
    }
  }

  // 启动项目
  const startProject = async (projectId) => {
    try {
      await nodeApi.startProject(projectId)
      ElMessage.success('启动成功')
      await fetchProjects()
      return true
    } catch (error) {
      console.error('启动失败:', error)
      ElMessage.error('启动失败')
      return false
    }
  }

  // 停止项目
  const stopProject = async (projectId) => {
    try {
      await nodeApi.stopProject(projectId)
      ElMessage.success('停止成功')
      await fetchProjects()
      return true
    } catch (error) {
      console.error('停止失败:', error)
      ElMessage.error('停止失败')
      return false
    }
  }

  // 重启项目
  const restartProject = async (projectId) => {
    try {
      await nodeApi.restartProject(projectId)
      ElMessage.success('重启成功')
      await fetchProjects()
      return true
    } catch (error) {
      console.error('重启失败:', error)
      ElMessage.error('重启失败')
      return false
    }
  }

  // 回滚项目
  const rollbackProject = async (projectId, version) => {
    try {
      await nodeApi.rollbackProject(projectId, version)
      ElMessage.success('回滚成功')
      await fetchProjects()
      return true
    } catch (error) {
      console.error('回滚失败:', error)
      ElMessage.error('回滚失败')
      return false
    }
  }

  // 获取项目日志
  const getProjectLogs = async (projectId) => {
    try {
      return await nodeApi.getProjectLogs(projectId)
    } catch (error) {
      console.error('获取日志失败:', error)
      return []
    }
  }

  return {
    nodeInfo,
    projects,
    loading,
    fetchNodeInfo,
    fetchProjects,
    deployProject,
    startProject,
    stopProject,
    restartProject,
    rollbackProject,
    getProjectLogs,
  }
})
