import { describe, expect, it } from 'vitest'
import {
  healthPresentation,
  isDeployableRuntimeCluster,
  isFreshNodeHeartbeat,
  isSchedulableCollectorNode,
  isDeploymentRunActive,
  lifecycleTagType,
  runEventMessagePresentation,
  runEventStagePresentation,
} from '@/views/tenant/utils/ops-presentation'

describe('ops presentation', () => {
  it('仅对仍在进行的异步任务保持轮询', () => {
    expect(isDeploymentRunActive('pending')).toBe(true)
    expect(isDeploymentRunActive('starting')).toBe(true)
    expect(isDeploymentRunActive('running')).toBe(false)
    expect(isDeploymentRunActive('pending', '2026-08-30T00:00:00Z')).toBe(false)
    expect(isDeploymentRunActive('completed')).toBe(false)
    expect(isDeploymentRunActive('failed')).toBe(false)
  })

  it('只允许已审批、启用、在线且心跳新鲜的采集节点进入发布选择器', () => {
    const now = new Date('2026-08-30T08:00:00Z').getTime()
    const candidate = {
      id: 'collector-1',
      name: '采集节点 01',
      role: 'collector_linux' as const,
      desiredStatus: 'active' as const,
      observedStatus: 'online' as const,
      health: 'healthy' as const,
      approvedAt: '2026-08-30T07:00:00Z',
      lastHeartbeatAt: '2026-08-30T07:59:30Z',
    }
    expect(isFreshNodeHeartbeat(candidate.lastHeartbeatAt, now)).toBe(true)
    expect(isSchedulableCollectorNode(candidate, now)).toBe(true)
    expect(
      isSchedulableCollectorNode({ ...candidate, lastHeartbeatAt: '2026-08-30T07:59:00Z' }, now),
    ).toBe(false)
    expect(isSchedulableCollectorNode({ ...candidate, approvedAt: undefined }, now)).toBe(false)
    expect(isSchedulableCollectorNode({ ...candidate, observedStatus: 'offline' }, now)).toBe(false)
  })

  it('发布目标只接受已就绪的单节点运行集群', () => {
    const cluster = {
      id: 'cluster-1',
      name: '开发运行集群',
      topology: 'single_node' as const,
      desiredStatus: 'ready' as const,
    }
    expect(isDeployableRuntimeCluster(cluster)).toBe(true)
    expect(isDeployableRuntimeCluster({ ...cluster, topology: 'high_availability' })).toBe(false)
    expect(isDeployableRuntimeCluster({ ...cluster, desiredStatus: 'maintenance' })).toBe(false)
  })

  it('未知健康状态降级展示而不是中断页面渲染', () => {
    expect(healthPresentation('healthy')).toEqual({ label: '健康', type: 'success' })
    expect(healthPresentation('pending')).toEqual({ label: '待确认', type: 'info' })
    expect(healthPresentation(undefined)).toEqual({ label: '待确认', type: 'info' })
  })

  it('工作负载状态使用明确的标签颜色', () => {
    expect(lifecycleTagType('running')).toBe('success')
    expect(lifecycleTagType('pending')).toBe('warning')
    expect(lifecycleTagType('failed')).toBe('danger')
    expect(lifecycleTagType('stopped')).toBe('info')
  })

  it('发布任务的内部阶段和固定消息转为用户可读文案', () => {
    expect(runEventStagePresentation('dispatched')).toBe('命令已下发')
    expect(runEventMessagePresentation('agent observed desired state')).toBe('节点已上报目标状态')
    expect(runEventMessagePresentation('自定义错误')).toBe('自定义错误')
  })
})
