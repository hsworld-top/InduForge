import axios from 'axios'

const api = axios.create({
  baseURL: '/api/v1',
  timeout: 10000,
})

// 节点信息
export const nodeApi = {
  // 获取节点信息
  getNodeInfo() {
    return api.get('/node/info').then((res) => res.data)
  },

  // 获取节点状态
  getNodeStatus() {
    return api.get('/node/status').then((res) => res.data)
  },

  // 获取项目列表
  getProjects() {
    return api.get('/projects').then((res) => res.data)
  },

  // 获取项目详情
  getProject(projectId) {
    return api.get(`/projects/${projectId}`).then((res) => res.data)
  },

  // 部署项目
  deployProject(projectId, data) {
    return api.post(`/projects/${projectId}/deploy`, data).then((res) => res.data)
  },

  // 上传 IFP 并自动解析工程信息部署
  deployProjectByIfp(file, autoStart = true) {
    const formData = new FormData()
    formData.append('ifpFile', file)
    formData.append('autoStart', String(autoStart))
    return api.post('/projects/deploy-ifp', formData).then((res) => res.data)
  },

  // 启动项目
  startProject(projectId) {
    return api.post(`/projects/${projectId}/start`).then((res) => res.data)
  },

  // 停止项目
  stopProject(projectId) {
    return api.post(`/projects/${projectId}/stop`).then((res) => res.data)
  },

  // 重启项目
  restartProject(projectId) {
    return api.post(`/projects/${projectId}/restart`).then((res) => res.data)
  },

  // 回滚项目
  rollbackProject(projectId, version) {
    return api.post(`/projects/${projectId}/rollback?version=${version}`).then((res) => res.data)
  },

  // 获取项目日志
  getProjectLogs(projectId) {
    return api.get(`/projects/${projectId}/logs`).then((res) => res.data)
  },

  // 获取连接配置
  getProfile(projectId) {
    return api.get(`/profile?projectId=${projectId}`).then((res) => res.data)
  },

  // 保存连接配置
  saveProfile(projectId, data) {
    return api.post(`/profile?projectId=${projectId}`, data).then((res) => res.data)
  },

  // 保存节点配置
  saveConfig(data) {
    return api.post('/config/save', data).then((res) => res.data)
  },

  // 获取服务配置
  getServiceConfig() {
    return api.get('/config/service').then((res) => res.data)
  },

  // 获取初始化状态
  getBootstrapStatus() {
    return api.get('/bootstrap/status').then((res) => res.data)
  },

  // 完成初始化
  completeBootstrap(data) {
    return api.post('/bootstrap/complete', data).then((res) => res.data)
  },

  // 设置开机自启动
  saveBootstrapAutostart(enabled) {
    return api.post('/bootstrap/autostart', { enabled }).then((res) => res.data)
  },
}
