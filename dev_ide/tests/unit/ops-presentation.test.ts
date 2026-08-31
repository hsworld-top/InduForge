import { describe, expect, it } from 'vitest'
import {
  deploymentCollectorNodeId,
  deploymentHasCollectorNode,
  deploymentNeedsAttention,
  deploymentStatePresentation,
  healthPresentation,
  isDeployableRuntimeCluster,
  isFreshNodeHeartbeat,
  isSchedulableCollectorNode,
  isDeploymentRunActive,
  lifecyclePresentation,
  lifecycleTagType,
  nodeHealthPresentation,
  nodeLocationPresentation,
  nodeServicePresentation,
  runEventMessagePresentation,
  runEventStagePresentation,
  workloadStatePresentation,
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

  it('以用户可读文案区分运行服务与采集服务', () => {
    expect(
      nodeLocationPresentation({
        role: 'runtime_linux',
        runtimeClusterId: 'cluster-1',
        runtimeClusterName: '产线 A 资源池',
      }),
    ).toBe('产线 A 资源池')
    expect(nodeLocationPresentation({ role: 'runtime_linux', runtimeClusterId: 'cluster-1' })).toBe(
      '运行资源池',
    )
    expect(nodeLocationPresentation({ role: 'collector_linux', runtimeClusterId: null })).toBe(
      '该服务器',
    )
    expect(nodeLocationPresentation({ role: 'runtime_linux', runtimeClusterId: null })).toBe(
      '未分配运行位置',
    )
    expect(nodeServicePresentation({ role: 'runtime_linux' })).toBe('运行工程服务')
    expect(nodeServicePresentation({ role: 'collector_windows' })).toBe('运行采集服务')
  })

  it('未知健康状态降级展示而不是中断页面渲染', () => {
    expect(healthPresentation('healthy')).toEqual({ label: '健康', type: 'success' })
    expect(healthPresentation('pending')).toEqual({ label: '状态待更新', type: 'info' })
    expect(healthPresentation(undefined)).toEqual({ label: '状态待更新', type: 'info' })
  })

  it('节点未上报时展示连接状态而不是推断服务器故障', () => {
    expect(nodeHealthPresentation('unavailable')).toEqual({ label: '不可用', type: 'danger' })
    expect(nodeHealthPresentation('unavailable', 'offline')).toEqual({
      label: '未连接',
      type: 'danger',
    })
    expect(nodeHealthPresentation('unavailable', 'revoked')).toEqual({
      label: '已撤销',
      type: 'danger',
    })
    expect(nodeHealthPresentation('healthy')).toEqual({ label: '健康', type: 'success' })
  })

  it('采集节点未配置时不把工程服务展示为健康运行', () => {
    const incomplete = {
      workloads: [{ role: 'collector' as const, hostNodeId: '' }],
      health: 'healthy' as const,
      observedStatus: 'running' as const,
    }
    expect(deploymentHasCollectorNode(incomplete)).toBe(false)
    expect(deploymentCollectorNodeId(incomplete)).toBe('')
    expect(deploymentNeedsAttention(incomplete)).toBe(true)
    expect(deploymentStatePresentation(incomplete)).toEqual({
      label: '配置不完整',
      detail: '采集服务不可用',
      type: 'warning',
    })
    expect(
      deploymentStatePresentation({
        workloads: [{ role: 'collector', hostNodeId: 'node-1' }],
        health: 'healthy',
        observedStatus: 'running',
      }),
    ).toEqual({ label: '运行中', detail: '状态正常', type: 'success' })
    expect(
      deploymentStatePresentation({
        workloads: [{ role: 'collector', hostNodeId: 'node-1' }],
        health: 'unavailable',
        observedStatus: 'failed',
      }),
    ).toEqual({ label: '运行失败', detail: '请查看操作记录', type: 'danger' })
    const configured = {
      workloads: [{ role: 'collector' as const, hostNodeId: 'node-1' }],
      health: 'healthy' as const,
      observedStatus: 'running' as const,
    }
    expect(deploymentCollectorNodeId(configured)).toBe('node-1')
    expect(deploymentNeedsAttention(configured, { health: 'unavailable' })).toBe(true)
    expect(deploymentStatePresentation(configured, { health: 'unavailable' })).toEqual({
      label: '需关注',
      detail: '采集节点未连接',
      type: 'danger',
    })
    expect(
      deploymentStatePresentation(configured, { health: 'unavailable' }, { health: 'unavailable' }),
    ).toEqual({ label: '需关注', detail: '运行节点和采集节点未连接', type: 'danger' })
  })

  it('工作负载状态使用明确的标签颜色', () => {
    expect(lifecycleTagType('running')).toBe('success')
    expect(lifecycleTagType('online')).toBe('success')
    expect(lifecycleTagType('pending')).toBe('warning')
    expect(lifecycleTagType('pending_approval')).toBe('warning')
    expect(lifecycleTagType('failed')).toBe('danger')
    expect(lifecycleTagType('offline')).toBe('danger')
    expect(lifecycleTagType('stopped')).toBe('info')
    expect(lifecyclePresentation('online')).toBe('在线')
    expect(lifecyclePresentation('pending_approval')).toBe('待确认')
    expect(lifecyclePresentation('unexpected')).toBe('状态待更新')
    expect(lifecycleTagType('degraded')).toBe('warning')
  })

  it('服务状态优先反映实际运行节点的连通性', () => {
    const workload = {
      role: 'compute' as const,
      health: 'healthy' as const,
      observedStatus: 'running' as const,
      lastMessage: 'agent observed desired state',
    }
    expect(workloadStatePresentation(workload, 'unavailable')).toEqual({
      label: '未连接',
      detail: '运行节点未连接',
      type: 'danger',
    })
    expect(workloadStatePresentation({ ...workload, role: 'collector' }, 'degraded')).toEqual({
      label: '需关注',
      detail: '采集节点状态异常',
      type: 'warning',
    })
    expect(workloadStatePresentation(workload, 'healthy')).toEqual({
      label: '运行中',
      detail: '服务状态已更新',
      type: 'success',
    })
    expect(
      workloadStatePresentation(
        { ...workload, observedStatus: 'failed', lastMessage: null },
        'healthy',
      ),
    ).toEqual({
      label: '运行异常',
      detail: '服务运行异常，可尝试重新启动；如仍失败，请联系管理员。',
      type: 'danger',
    })
  })

  it('发布任务的内部阶段和固定消息转为用户可读文案', () => {
    expect(runEventStagePresentation('dispatched')).toBe('正在执行')
    expect(runEventMessagePresentation('agent observed desired state')).toBe('服务状态已更新')
    expect(runEventMessagePresentation('自定义错误')).toBe('状态已更新')
  })
})
