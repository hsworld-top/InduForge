import { describe, expect, it } from 'vitest'
import { formatDashboardActivity } from '@/utils/dashboard-activity'

const messages: Record<string, string> = {
  'dashboard.systemAction': '系统',
  'dashboard.activityAction.create': '创建了',
  'dashboard.activityAction.update': '更新了',
  'dashboard.activityAction.delete': '删除了',
  'dashboard.activityAction.approve': '审批通过了',
  'dashboard.activityAction.deploy': '部署了',
  'dashboard.activityAction.operate': '操作了',
  'dashboard.activityResource.project': '工程',
  'dashboard.activityResource.note': '共享便签',
  'dashboard.activityResource.node': '运行节点',
  'dashboard.activityResource.deployment': '部署',
}

const t = (key: string, params: Record<string, unknown> = {}) => {
  if (key === 'dashboard.businessActivity') {
    return `${params.actor}${params.action}${params.resource}`
  }
  return messages[key] || key
}

describe('仪表盘最近业务活动', () => {
  it('使用操作者、写操作和业务资源生成描述', () => {
    expect(
      formatDashboardActivity(
        { action: 'create', resource: 'projects', user: { fullName: '张三' } },
        t,
      ),
    ).toBe('张三创建了工程')
  })

  it('优先根据路径识别共享便签', () => {
    expect(
      formatDashboardActivity(
        {
          action: 'update',
          resource: 'tenants',
          path: '/api/v1/tenants/current/dashboard-notes/note-1',
          user: { username: 'admin' },
        },
        t,
      ),
    ).toBe('admin更新了共享便签')
  })

  it('识别审批和部署等非 CRUD 业务动作', () => {
    expect(
      formatDashboardActivity(
        {
          action: 'update',
          resource: 'nodes',
          path: '/api/v1/nodes/node-1/approve',
          user: { username: 'admin' },
        },
        t,
      ),
    ).toBe('admin审批通过了运行节点')

    expect(
      formatDashboardActivity(
        {
          action: 'create',
          resource: 'deployments',
          path: '/api/v1/deployments/version-1/deploy',
          user: { username: 'admin' },
        },
        t,
      ),
    ).toBe('admin部署了部署')
  })
})
