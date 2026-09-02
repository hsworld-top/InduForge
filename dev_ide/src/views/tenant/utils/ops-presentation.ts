import type {
  ApplicationVersion,
  DeploymentService,
  OpsCapability,
  OpsNode,
  OpsPlatform,
  OpsServiceType,
  ProjectDeployment,
} from '@/api/ops.api'

export const NODE_HEARTBEAT_FRESH_MS = 45_000
export const capabilityLabel: Record<OpsCapability, string> = {
  project_entry: '工程入口',
  data_runtime: '数据运行',
  collector: '数据采集',
}
export const serviceLabel: Record<OpsServiceType | OpsCapability, string> = {
  base: '基础引擎',
  compute: '计算引擎',
  alarm: '报警引擎',
  collector: '数据采集',
  project_entry: '工程入口',
  data_runtime: '数据运行',
}
export const platformLabel = (platform?: string) =>
  platform === 'windows' ? 'Windows' : platform === 'linux' ? 'Linux' : '未知平台'

const isRecord = (value: unknown): value is Record<string, unknown> =>
  Boolean(value) && typeof value === 'object' && !Array.isArray(value)

/** 仅在下拉中展示具备正式 Release 基本特征的版本，部署时仍由后端完整验证。 */
export const isDeployableReleaseVersion = (
  version: ApplicationVersion,
  requireCollector = false,
) => {
  const artifacts = version.manifest?.artifacts
  return (
    ['ready', 'success'].includes(String(version.status || '').toLowerCase()) &&
    /^[0-9a-fA-F]{64}$/.test(version.artifactHash || '') &&
    version.manifest?.schemaVersion === '2.0' &&
    isRecord(artifacts?.client) &&
    isRecord(artifacts?.runtime) &&
    (!requireCollector || isRecord(artifacts?.collector))
  )
}

/** 节点用途由安装平台派生，避免用户拼出无法运行的能力组合。 */
export const enrollmentCapabilities = (
  platform: OpsPlatform,
  enableCollector = false,
): OpsCapability[] => {
  if (platform === 'windows') return ['collector']
  return enableCollector
    ? ['project_entry', 'data_runtime', 'collector']
    : ['project_entry', 'data_runtime']
}

const installValue = (value: string, field: string) => {
  const normalized = value.trim()
  if (!normalized || /[\r\n]/.test(normalized)) {
    throw new TypeError(`${field} 必须是非空单行文本`)
  }
  return normalized
}
const bashQuote = (value: string) => `'${value.replaceAll("'", `'"'"'`)}'`
const powershellQuote = (value: string) => `'${value.replaceAll("'", "''")}'`

export interface EnrollmentServerUrlValidation {
  valid: boolean
  normalized: string
  loopback: boolean
  error: string
}

const invalidServerUrl = (error: string): EnrollmentServerUrlValidation => ({
  valid: false,
  normalized: '',
  loopback: false,
  error,
})

const isLoopbackHostname = (hostname: string) =>
  hostname === 'localhost' || /^127(?:\.\d{1,3}){3}$/.test(hostname) || hostname === '[::1]'

/** 节点安装只接受安全、无敏感参数的中心根地址。 */
export const validateEnrollmentServerUrl = (value: string): EnrollmentServerUrlValidation => {
  const input = value.trim()
  if (!input || /[\r\n]/.test(input)) {
    return invalidServerUrl('请输入非空单行的节点可访问中心地址')
  }

  let url: URL
  try {
    url = new URL(input)
  } catch {
    return invalidServerUrl('请输入完整的 HTTP/HTTPS 绝对地址')
  }
  if (url.protocol !== 'https:' && url.protocol !== 'http:') {
    return invalidServerUrl('中心地址仅支持 HTTPS，仅 loopback 地址可使用 HTTP')
  }
  if (url.username || url.password) {
    return invalidServerUrl('中心地址不能包含用户名或密码')
  }
  if (url.pathname !== '/') {
    return invalidServerUrl('中心地址必须使用根路径，不能包含子路径')
  }
  const suffix = url.href.slice(url.origin.length)
  if (suffix.includes('?')) {
    return invalidServerUrl('中心地址不能包含查询参数')
  }
  if (suffix.includes('#')) {
    return invalidServerUrl('中心地址不能包含片段')
  }

  const loopback = isLoopbackHostname(url.hostname.toLowerCase())
  if (url.protocol === 'http:' && !loopback) {
    return invalidServerUrl('远程物理节点的中心地址必须使用 HTTPS')
  }
  return { valid: true, normalized: url.origin, loopback, error: '' }
}

/** 仅生成供用户复制的安装命令，不执行、不记录接入码。 */
export const enrollmentInstallCommand = (input: {
  platform: OpsPlatform
  serverUrl: string
  enrollmentCode: string
  enableCollector?: boolean
}) => {
  const serverUrlValidation = validateEnrollmentServerUrl(input.serverUrl)
  if (!serverUrlValidation.valid) throw new TypeError(serverUrlValidation.error)
  const serverUrl = serverUrlValidation.normalized
  const enrollmentCode = installValue(input.enrollmentCode, '接入码')
  if (input.platform === 'windows') {
    return `& .\\install.ps1 -ServerUrl ${powershellQuote(serverUrl)} -EnrollmentCode ${powershellQuote(enrollmentCode)}`
  }
  const collectorOption = input.enableCollector ? ' --enable-collector' : ''
  return `sudo ./install.sh${collectorOption} --server-url ${bashQuote(serverUrl)} --enrollment-code ${bashQuote(enrollmentCode)}`
}

export const isFreshNodeHeartbeat = (value?: string, now = Date.now()) =>
  Boolean(value) && Math.abs(now - new Date(value!).getTime()) <= NODE_HEARTBEAT_FRESH_MS
export const isDeployableNode = (node: OpsNode, now = Date.now()) =>
  !node.assignedDeploymentId &&
  node.approvedAt != null &&
  node.desiredStatus === 'active' &&
  node.observedStatus === 'online' &&
  isFreshNodeHeartbeat(node.lastHeartbeatAt, now) &&
  node.capabilities.includes('project_entry') &&
  node.capabilities.includes('data_runtime')
export const lifecyclePresentation = (status?: string) =>
  ({
    running: '运行中',
    stopped: '已停止',
    failed: '失败',
    pending: '等待中',
    online: '在线',
    offline: '未连接',
    pending_approval: '待确认',
    created: '等待领取',
    claimed: '待确认',
    approved: '已接入',
    rejected: '已拒绝',
    expired: '已过期',
    active: '已启用',
    revoked: '已撤销',
  })[String(status || '').toLowerCase()] || '状态待更新'
export const lifecycleTagType = (status?: string) =>
  ['running', 'online', 'active', 'approved'].includes(String(status).toLowerCase())
    ? ('success' as const)
    : ['failed', 'offline', 'revoked', 'rejected', 'expired'].includes(String(status).toLowerCase())
      ? ('danger' as const)
      : ['pending', 'degraded', 'pending_approval', 'claimed'].includes(
            String(status).toLowerCase(),
          )
        ? ('warning' as const)
        : ('info' as const)
export const serviceStatePresentation = (service: DeploymentService) => ({
  label: lifecyclePresentation(service.observedStatus),
  detail: service.lastMessage || '等待状态更新',
  type: lifecycleTagType(service.observedStatus),
})
export const runEventPresentation = (stage?: string, message?: string) => {
  const normalizedMessage = String(message || '').toLowerCase()
  const queuedMessages: Record<string, string> = {
    'deployment queued': '部署任务已创建，等待目标节点执行',
    'start queued': '启动任务已创建，等待目标节点执行',
    'stop queued': '停止任务已创建，等待目标节点执行',
    'restart queued': '重启任务已创建，等待目标节点执行',
  }
  if (stage === 'dispatched') {
    return { stage: '已下发', message: '工作负载已下发，等待运行服务就绪' }
  }
  if (stage === 'ready' || stage === 'succeeded') {
    return { stage: '已就绪', message: message || '全部运行服务已通过健康检查' }
  }
  return {
    stage: stage === 'queued' ? '已受理' : stage || '状态更新',
    message: queuedMessages[normalizedMessage] || message || '状态已更新',
  }
}

/**
 * 部署详情只能由后端实际观察状态推进。下发 Kubernetes 工作负载不等于容器已启动，
 * 更不能代表健康检查成功；只有入口与所有运行服务均已 running 才展示完成。
 */
export const deploymentDetailPresentation = (deployment?: {
  observedStatus?: string
  entryStatus?: string
  services?: Array<Pick<DeploymentService, 'observedStatus'>>
}) => {
  const services = deployment?.services || []
  const failed =
    deployment?.observedStatus === 'failed' ||
    deployment?.entryStatus === 'failed' ||
    services.some((service) => service.observedStatus === 'failed')
  const ready =
    !failed &&
    deployment?.observedStatus === 'running' &&
    deployment?.entryStatus === 'running' &&
    services.length > 0 &&
    services.every((service) => service.observedStatus === 'running')
  if (ready) {
    return {
      active: 4,
      processStatus: 'process' as const,
      prepareDescription: '运行资源已准备',
      serviceDescription: '全部运行服务已启动',
      healthDescription: '全部运行服务已通过健康检查',
    }
  }
  return {
    active: 1,
    processStatus: failed ? ('error' as const) : ('process' as const),
    prepareDescription: failed ? '运行资源准备失败，请查看服务状态' : '工作负载已下发，等待 Kubernetes 就绪',
    serviceDescription: failed ? '运行服务未能启动' : '等待运行服务就绪',
    healthDescription: failed ? '健康检查未通过' : '等待全部运行服务通过健康检查',
  }
}
export const deploymentStatePresentation = (
  deployment: Pick<ProjectDeployment, 'observedStatus' | 'entryStatus' | 'health'>,
) =>
  deployment.observedStatus === 'failed' || deployment.entryStatus === 'failed'
    ? { label: '运行失败', type: 'danger' as const }
    : deployment.observedStatus === 'running' && deployment.entryStatus === 'running'
      ? { label: '运行中', type: 'success' as const }
      : {
          label: lifecyclePresentation(deployment.observedStatus),
          type: lifecycleTagType(deployment.observedStatus),
        }
