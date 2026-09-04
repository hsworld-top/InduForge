import { describe, expect, it } from 'vitest'
import {
  projectDeploymentAction,
  publishAccessPort,
  publishActionContextMatches,
  type ProjectDeploymentSummary,
} from '@/views/tenant/project-management/project-deployment-action'
import {
  deploymentNavigation,
  requestDeploymentNavigation,
} from '@/views/tenant/project-management/project-deployment-navigation'

const summary = (overrides: Partial<ProjectDeploymentSummary> = {}): ProjectDeploymentSummary => ({
  deploymentCount: 1,
  environmentCount: 1,
  operationInProgress: false,
  primarySelection: 'unique',
  updatedAt: null,
  primaryDeployment: {
    id: 'd1',
    environmentId: 'e1',
    environmentName: '默认环境',
    mode: 'development',
    desiredStatus: 'running',
    observedStatus: 'running',
    operationInProgress: false,
    placements: { base: 'n1' },
  },
  ...overrides,
})

describe('工程发布入口状态', () => {
  it('切换环境或模式后不继续沿用原更新动作标题', () => {
    const initial = { mode: 'development' as const, environmentId: 'env-1' }
    expect(publishActionContextMatches(initial, 'DEV', 'env-1')).toBe(true)
    expect(publishActionContextMatches(initial, 'RELEASE', 'env-1')).toBe(false)
    expect(publishActionContextMatches(initial, 'DEV', 'env-2')).toBe(false)
  })
  it('更新保留当前部署自定义端口，仅新建采用默认端口', () => {
    expect(publishAccessPort({ accessPort: 18888 })).toBe(18888)
    expect(publishAccessPort(undefined)).toBe(17800)
  })
  it('未初始化不伪造无部署，0部署才允许发布', () => {
    expect(projectDeploymentAction(undefined)).toMatchObject({
      label: '读取部署状态',
      disabled: true,
    })
    expect(
      projectDeploymentAction(
        summary({
          deploymentCount: 0,
          environmentCount: 0,
          primarySelection: 'none',
          primaryDeployment: null,
        }),
      ),
    ).toMatchObject({ label: '发布工程', kind: 'publish' })
  })
  it('唯一部署按模式、执行、异常、停止状态映射', () => {
    expect(projectDeploymentAction(summary()).label).toBe('更新开发版')
    expect(
      projectDeploymentAction(
        summary({ primaryDeployment: { ...summary().primaryDeployment!, mode: 'production' } }),
      ).label,
    ).toBe('发布新版本')
    expect(projectDeploymentAction(summary({ operationInProgress: true })).label).toBe('查看进度')
    expect(
      projectDeploymentAction(
        summary({
          operationInProgress: false,
          primaryDeployment: {
            ...summary().primaryDeployment!,
            operationInProgress: false,
            updating: true,
          },
        }),
      ).label,
    ).toBe('查看进度')
    expect(
      projectDeploymentAction(summary({ primaryDeployment: {
        ...summary().primaryDeployment!, observedStatus: 'failed', operationInProgress: false, updating: true,
      } })).label,
    ).toBe('处理异常')
    expect(
      projectDeploymentAction(
        summary({
          primaryDeployment: { ...summary().primaryDeployment!, observedStatus: 'failed' },
        }),
      ).label,
    ).toBe('处理异常')
    expect(
      projectDeploymentAction(
        summary({
          primaryDeployment: {
            ...summary().primaryDeployment!,
            observedStatus: 'stopped',
            desiredStatus: 'stopped',
          },
        }),
      ).label,
    ).toBe('更新并启动')
  })
  it('多环境或多部署只进入管理，不猜主部署', () => {
    expect(
      projectDeploymentAction(
        summary({
          deploymentCount: 2,
          environmentCount: 2,
          primarySelection: 'multiple',
          primaryDeployment: null,
        }),
      ),
    ).toMatchObject({ label: '管理部署', kind: 'manage' })
  })
  it('导航请求在运维懒加载前保留最新明确目标', () => {
    requestDeploymentNavigation('p1', 'd1', '工程一')
    const first = deploymentNavigation.value
    requestDeploymentNavigation('p2', undefined, '工程二')
    expect(deploymentNavigation.value).toMatchObject({ projectId: 'p2', projectName: '工程二' })
    expect(deploymentNavigation.value!.requestId).toBeGreaterThan(first!.requestId)
  })
})
