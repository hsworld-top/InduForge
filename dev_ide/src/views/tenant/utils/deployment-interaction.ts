import type { DeploymentMode, OpsDeploymentAction, ProjectDeployment } from '@/api/ops.api'

/** 运行意图尚未收敛时禁止重复操作，失败态仍允许显式重试。 */
export function deploymentBusy(item: ProjectDeployment) {
  if (item.observedStatus === 'failed') return false
  return (
    !['running', 'stopped'].includes(item.observedStatus || '') ||
    Boolean(item.desiredStatus && item.desiredStatus !== item.observedStatus) ||
    Boolean(
      item.services?.some(
        (service) =>
          service.observedStatus !== 'failed' &&
          (service.desiredGeneration || 0) > (service.observedGeneration || 0),
      ),
    )
  )
}

/** 并行引擎只汇总已观察到的就绪数；名称并列表示同时执行，不暗示启动顺序。 */
export function deploymentProgressLabel(item: ProjectDeployment) {
  const names: Record<string, string> = {
    base: '基础引擎',
    compute: '计算引擎',
    alarm: '报警引擎',
    collector: '采集引擎',
  }
  const services = item.services || []
  const failed = services.filter((service) => service.observedStatus === 'failed')
  if (item.observedStatus === 'failed' || failed.length)
    return `执行失败${failed.length ? ` · ${failed.map((service) => names[service.serviceType] || service.serviceType).join('、')}` : ''}`
  const required = services.filter((service) => (service.desiredStatus || 'running') === 'running')
  const ready = required.filter(
    (service) =>
      service.observedStatus === 'running' &&
      (service.observedGeneration || 0) >= (service.desiredGeneration || 0),
  )
  if (item.desiredStatus === 'stopped')
    return item.observedStatus === 'stopped' && !deploymentBusy(item)
      ? '已停止'
      : '正在停止运行引擎'
  if (!deploymentBusy(item)) return item.observedStatus === 'running' ? '运行中' : '已停止'
  const pending = required.filter((service) => !ready.includes(service))
  return `正在启动${pending.length ? pending.map((service) => names[service.serviceType] || service.serviceType).join('、') : '运行引擎'}${required.length ? ` · ${ready.length}/${required.length} 已就绪` : ''}`
}

export function canOperateDeployment(
  item: ProjectDeployment,
  action: OpsDeploymentAction | 'delete',
  permitted: boolean,
  busy = false,
) {
  if (!permitted || busy || deploymentBusy(item)) return false
  if (action === 'start') return ['stopped', 'failed'].includes(item.observedStatus || '')
  if (action === 'stop' || action === 'restart') return item.observedStatus === 'running'
  return ['running', 'stopped', 'failed'].includes(item.observedStatus || '')
}

export const deploymentMode = (item: Pick<ProjectDeployment, 'mode' | 'version'>): DeploymentMode =>
  item.mode || (item.version === '__DEV__' || item.version === 'dev' ? 'development' : 'production')

export function deploymentReplacementMessage(item: ProjectDeployment, mode: DeploymentMode) {
  const switching = deploymentMode(item) !== mode
  return `${switching ? '切换部署模式将替换当前部署' : '更新将替换当前部署制品'}。同一工程在同一环境只保留一套部署，不会创建第二套运行环境或隔离数据。${mode === 'development' ? '将构建当前工程快照' : '将使用所选发布版本'}，期间可能暂时不可用。是否继续？`
}

/** 外链仅接受后端给出的 HTTP(S) 地址，未就绪时不提供打开入口。 */
export function deploymentAccessUrl(item: ProjectDeployment) {
  if (
    item.accessAvailable === false ||
    item.observedStatus !== 'running' ||
    item.entryStatus !== 'running'
  )
    return ''
  try {
    const url = new URL(item.accessUrl || '')
    return ['http:', 'https:'].includes(url.protocol) && !url.username && !url.password
      ? url.href
      : ''
  } catch {
    return ''
  }
}

/** 串行轮询；失活取消请求，响应通过 signal 防止覆盖新页面，恢复立即刷新。 */
export function createDeploymentRefresh(options: {
  load: (signal: AbortSignal) => Promise<void>
  busy: () => boolean
  error: () => void
}) {
  let active = false
  let timer: ReturnType<typeof setTimeout> | undefined
  let controller: AbortController | undefined
  let pending = false
  let inFlight: Promise<void> | undefined
  const schedule = () => {
    if (active) timer = setTimeout(() => void refresh(), options.busy() ? 2000 : 5000)
  }
  async function refresh(): Promise<void> {
    if (!active) return
    if (timer) clearTimeout(timer)
    if (inFlight) {
      pending = true
      return inFlight
    }
    const current = new AbortController()
    controller = current
    inFlight = options
      .load(current.signal)
      .catch(() => {
        if (!current.signal.aborted) options.error()
      })
      .finally(() => {
        inFlight = undefined
        if (controller === current) controller = undefined
        if (pending && active) {
          pending = false
          void refresh()
        } else schedule()
      })
    return inFlight
  }
  return {
    refresh,
    setActive(value: boolean) {
      if (value === active) return
      active = value
      if (timer) clearTimeout(timer)
      if (!active) {
        pending = false
        controller?.abort()
      } else void refresh()
    },
    dispose() {
      active = false
      pending = false
      if (timer) clearTimeout(timer)
      controller?.abort()
    },
  }
}
