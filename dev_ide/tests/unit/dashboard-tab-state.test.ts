import { describe, expect, it } from 'vitest'
import { restoreDashboardTabState, serializeDashboardTabState } from '@/utils/dashboardTabState'

describe('dashboardTabState', () => {
  it('可序列化并恢复 embedded tab 状态', () => {
    const tabConfigMap = {
      dashboard: {
        titleKey: 'dashboard.title',
        component: { name: 'DashboardContent' },
        icon: 'home',
      },
    }

    const savedState = serializeDashboardTabState({
      tabs: [
        {
          key: 'dashboard',
          titleKey: 'dashboard.title',
          component: tabConfigMap.dashboard.component,
          icon: 'home',
          props: null,
        },
        {
          key: 'data-center-p2',
          titleKey: 'projectManagement.dataCenter',
          titlePrefix: '项目二',
          component: { name: 'EmbeddedApp' },
          icon: 'database',
          props: {
            appType: 'datacenter',
            project: {
              id: 'p2',
              tenantId: 't2',
              name: '项目二',
            },
          },
        },
      ],
      activeTab: 'data-center-p2',
      tabConfigMap,
      hasTabPermission: () => true,
    })

    const restored = restoreDashboardTabState(savedState, {
      tabConfigMap,
      hasTabPermission: () => true,
      embeddedComponent: { name: 'EmbeddedApp' },
      translate(key: string) {
        if (key === 'dashboard.title') return '仪表盘'
        if (key === 'projectManagement.dataCenter') return '数据中心'
        return key
      },
    })

    expect(restored?.activeTab).toBe('data-center-p2')
    expect(restored?.tabs).toHaveLength(2)
    expect(restored?.tabs[1]).toMatchObject({
      key: 'data-center-p2',
      title: '项目二 - 数据中心',
      props: {
        appType: 'datacenter',
        project: {
          id: 'p2',
          tenantId: 't2',
          name: '项目二',
        },
      },
    })
  })
})
