import type { DeploymentRun, DeploymentService } from '@/api/ops.api'

export const runOperationLabel = (run?: DeploymentRun) => {
  if (run?.operation === 'deploy' && run.message?.startsWith('重新部署')) return '重新部署'
  return (
    (
      {
        deploy: '部署更新',
        start: '启动',
        stop: '停止',
        restart: '重启',
        redeploy: '重新部署',
        delete: '删除',
      } as Record<string, string>
    )[run?.operation || ''] || '部署操作'
  )
}
export const runResultLabel = (run: DeploymentRun) => {
  const status = run.status || run.observedStatus
  if (status === 'failed') return '失败'
  if (run.completedAt) return '已完成'
  return '执行中'
}
export const runDurationLabel = (run: DeploymentRun) => {
  if (!run.completedAt) return '—'
  const elapsed =
    run.durationMs ??
    (run.startedAt ? Date.parse(run.completedAt) - Date.parse(run.startedAt) : NaN)
  if (!Number.isFinite(elapsed) || elapsed < 0) return '—'
  if (elapsed < 1000) return '<1秒'
  const seconds = Math.floor(elapsed / 1000)
  return seconds < 60 ? `${seconds}秒` : `${Math.floor(seconds / 60)}分${seconds % 60}秒`
}
/** 正常行只展示运行语义；编排原文留在异常的按需技术详情中。 */
export function deploymentEnginePresentation(service: DeploymentService) {
  const status = service.observedStatus
  const converged =
    service.observedGeneration != null &&
    service.desiredGeneration != null &&
    service.observedGeneration === service.desiredGeneration
  if (status === 'running' && converged && (service.desiredStatus || 'running') === 'running')
    return { state: 'ready', label: '已就绪', summary: '运行正常', technical: '' }
  if (status === 'stopped' && service.desiredStatus === 'stopped' && converged)
    return { state: 'stopped', label: '已停止', summary: '未运行', technical: '' }
  if (status === 'failed') {
    const message = service.lastMessage || ''
    const summary = /CrashLoopBackOff/i.test(message)
      ? '引擎反复启动失败'
      : /ImagePull|ErrImagePull/i.test(message)
        ? '运行组件获取失败'
        : /timeout|deadline|超时/i.test(message)
          ? '引擎启动超时'
          : /PersistentVolume|PVC/i.test(message)
            ? '运行存储尚未就绪'
            : '引擎运行异常'
    return { state: 'failed', label: '异常', summary, technical: message }
  }
  return {
    state: 'pending',
    label: '执行中',
    summary: service.desiredStatus === 'stopped' ? '正在停止' : '等待引擎就绪',
    technical: '',
  }
}
