import { describe, expect, it } from 'vitest'
import { restoreDashboardTabState, serializeDashboardTabState } from '@/utils/dashboardTabState'

describe('dashboardTabState', () => {
  it('可序列化并恢复 Wujie 子应用标签状态', () => {
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
          component: { name: 'WujieMicroApp' },
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
      microAppComponent: { name: 'WujieMicroApp' },
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

  it('工具标签只持久化工程与目标并在恢复时重新创建组件', () => {
    const savedState = serializeDashboardTabState({
      tabs: [
        {
          key: 'p1:2d:overview',
          title: '产线总览 · 2D',
          component: { name: 'WorkspaceToolFrame' },
          icon: 'design',
          props: {
            target: '2d',
            sceneId: 'overview',
            sceneName: '产线总览',
            project: { id: 'p1', name: '项目一', tenantId: 't1' },
            url: 'https://must-not-persist.example.test',
          },
        },
      ],
      activeTab: 'p1:2d:overview',
    })

    expect(JSON.stringify(savedState)).not.toContain('must-not-persist')
    const restored = restoreDashboardTabState(savedState, {
      workspaceToolComponent: { name: 'WorkspaceToolFrame' },
    })
    expect(restored).toMatchObject({
      activeTab: 'p1:2d:overview',
      tabs: [
        {
          key: 'p1:2d:overview',
          title: '产线总览 · 2D',
          props: {
            target: '2d',
            sceneId: 'overview',
            sceneName: '产线总览',
            project: { id: 'p1', name: '项目一', tenantId: 't1' },
          },
        },
      ],
    })
  })

  it('不会恢复已取消的源码顶层标签', () => {
    const restored = restoreDashboardTabState(
      {
        tabs: [
          {
            type: 'workspace-tool',
            key: 'p1:code',
            title: '项目一 · 源码',
            icon: 'code',
            props: { target: 'code', project: { id: 'p1', name: '项目一' } },
          },
        ],
        activeTab: 'p1:code',
      },
      { workspaceToolComponent: { name: 'WorkspaceToolFrame' } },
    )

    expect(restored).toBeNull()
  })
})
