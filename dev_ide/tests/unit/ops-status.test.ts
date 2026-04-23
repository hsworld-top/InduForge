import { describe, expect, it } from 'vitest'

import {
  buildDeployStatusSummary,
  getDeployFailureReason,
  getDeployLabel,
  getDeployStatusType,
  isFailedDeploy,
} from '@/views/tenant/utils/ops-status'

describe('views/tenant/utils/ops-status', () => {
  it('buildDeployStatusSummary 能正确聚合各类部署状态', () => {
    const summary = buildDeployStatusSummary([
      {
        deployments: [{ status: 'running' }, { status: 'deploying' }, { status: 'pending' }],
      },
      {
        deployments: [{ status: 'stopped' }, { status: 'error' }, { status: 'failed' }],
      },
      {
        deployments: [{ status: 'unknown' }],
      },
      {
        deployments: null,
      },
    ])

    expect(summary).toEqual({
      running: 1,
      deploying: 2,
      stopped: 1,
      failed: 2,
    })
  })

  it('状态映射与失败判定保持预期', () => {
    expect(getDeployStatusType('running')).toBe('success')
    expect(getDeployStatusType('failed')).toBe('danger')
    expect(getDeployStatusType('unknown')).toBe('info')

    expect(getDeployLabel('deploying')).toBe('部署中')
    expect(getDeployLabel('pending')).toBe('等待中')
    expect(getDeployLabel('unknown')).toBe('unknown')

    expect(isFailedDeploy({ status: 'error' })).toBe(true)
    expect(isFailedDeploy({ status: 'failed' })).toBe(true)
    expect(isFailedDeploy({ status: 'running' })).toBe(false)
    expect(isFailedDeploy(undefined)).toBe(false)
  })

  it('getDeployFailureReason 按优先级返回失败原因', () => {
    expect(getDeployFailureReason({ errorMessage: '主错误' })).toBe('主错误')
    expect(getDeployFailureReason({ lastError: '次错误' })).toBe('次错误')
    expect(getDeployFailureReason({ message: '通用错误' })).toBe('通用错误')
    expect(getDeployFailureReason({})).toBe('未提供失败原因')
    expect(getDeployFailureReason(undefined)).toBe('未提供失败原因')
  })
})
