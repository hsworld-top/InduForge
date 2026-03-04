import { defineStore } from 'pinia'
import { ref } from 'vue'
import { ElMessage } from 'element-plus'
import { nodeApi } from '@/api/nodeApi'

export const useNodeStore = defineStore('node', () => {
  const nodeInfo = ref(null)
  const projects = ref([])
  const loading = ref(false)

  /**
   * 获取当前语言。
   * @returns {'zh-CN'|'en-US'}
   */
  const getLocale = () => (localStorage.getItem('node_agent_locale') === 'en-US' ? 'en-US' : 'zh-CN')

  /**
   * 返回中英文文案。
   * @param {string} zh 中文文案
   * @param {string} en 英文文案
   * @returns {string}
   */
  const text = (zh, en) => (getLocale() === 'en-US' ? en : zh)

  /**
   * 统一错误文案。
   * @param {unknown} error 错误对象
   * @param {string} fallback 默认文案
   * @returns {string}
   */
  const getErrorMessage = (error, fallback) => {
    const code = error?.response?.data?.code
    if (code === 'PROJECT_MANAGED_BY_CENTER') {
      return text(
        '该工程由运维中心托管，请在运维中心执行状态变更',
        'This project is managed by the center. Please change its status in the center.',
      )
    }
    return error?.response?.data?.error || fallback
  }

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
      ElMessage.error(text('获取项目列表失败', 'Failed to fetch project list'))
    } finally {
      loading.value = false
    }
  }

  // 部署项目
  const deployProject = async (projectId, data) => {
    try {
      await nodeApi.deployProject(projectId, data)
      ElMessage.success(text('部署成功', 'Deployment succeeded'))
      await fetchProjects()
      return true
    } catch (error) {
      console.error('部署失败:', error)
      ElMessage.error(getErrorMessage(error, text('部署失败', 'Deployment failed')))
      return false
    }
  }

  // 通过 IFP 文件部署项目（项目ID/版本由后端解析）
  const deployProjectByIfp = async (file, autoStart) => {
    try {
      await nodeApi.deployProjectByIfp(file, autoStart)
      ElMessage.success(text('部署成功', 'Deployment succeeded'))
      await fetchProjects()
      return true
    } catch (error) {
      console.error('部署失败:', error)
      ElMessage.error(getErrorMessage(error, text('部署失败', 'Deployment failed')))
      return false
    }
  }

  // 启动项目
  const startProject = async (projectId) => {
    try {
      await nodeApi.startProject(projectId)
      ElMessage.success(text('启动成功', 'Start succeeded'))
      await fetchProjects()
      return true
    } catch (error) {
      console.error('启动失败:', error)
      ElMessage.error(getErrorMessage(error, text('启动失败', 'Start failed')))
      return false
    }
  }

  // 停止项目
  const stopProject = async (projectId) => {
    try {
      await nodeApi.stopProject(projectId)
      ElMessage.success(text('停止成功', 'Stop succeeded'))
      await fetchProjects()
      return true
    } catch (error) {
      console.error('停止失败:', error)
      ElMessage.error(getErrorMessage(error, text('停止失败', 'Stop failed')))
      return false
    }
  }

  // 重启项目
  const restartProject = async (projectId) => {
    try {
      await nodeApi.restartProject(projectId)
      ElMessage.success(text('重启成功', 'Restart succeeded'))
      await fetchProjects()
      return true
    } catch (error) {
      console.error('重启失败:', error)
      ElMessage.error(getErrorMessage(error, text('重启失败', 'Restart failed')))
      return false
    }
  }

  // 回滚项目
  const rollbackProject = async (projectId, version) => {
    try {
      await nodeApi.rollbackProject(projectId, version)
      ElMessage.success(text('回滚成功', 'Rollback succeeded'))
      await fetchProjects()
      return true
    } catch (error) {
      console.error('回滚失败:', error)
      ElMessage.error(getErrorMessage(error, text('回滚失败', 'Rollback failed')))
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
    deployProjectByIfp,
    startProject,
    stopProject,
    restartProject,
    rollbackProject,
    getProjectLogs,
  }
})
