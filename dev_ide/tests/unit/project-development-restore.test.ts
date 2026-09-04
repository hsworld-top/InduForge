import { describe, expect, it } from 'vitest'
import {
  isRestoreTerminal,
  restoreStage,
  restoreView,
  versionRestoreAvailability,
} from '@/views/tenant/project-management/project-development-restore'

describe('工程开发态恢复状态机', () => {
  const version = {
    id: 'v1',
    projectId: 'p1',
    version: '1.0.3',
    status: 'ready',
    restorable: true,
    authoringProjectRevision: 19,
  }

  it('只允许包含完整快照和工程修订的成功版本', () => {
    expect(versionRestoreAvailability(version)).toEqual({ allowed: true })
    expect(versionRestoreAvailability({ ...version, status: 'success' }).allowed).toBe(false)
    expect(versionRestoreAvailability({ ...version, status: 'building' })).toEqual({
      allowed: false,
      reasonKey: 'projectManagement.restoreRequiresReadyVersion',
    })
    expect(versionRestoreAvailability({ ...version, restorable: false })).toEqual({
      allowed: false,
      reasonKey: 'projectManagement.restoreSnapshotUnavailable',
    })
    expect(
      versionRestoreAvailability({
        ...version,
        restorable: false,
        restoreUnavailableReason: '快照校验失败',
      }),
    ).toEqual({ allowed: false, reason: '快照校验失败' })
    expect(versionRestoreAvailability({ ...version, authoringProjectRevision: undefined }).allowed).toBe(true)
  })

  it('只按服务端真实状态映射备份、恢复和校验阶段', () => {
    expect(restoreStage('staging')).toEqual({
      active: 0,
      labelKey: 'projectManagement.restoreBackingUp',
    })
    expect(restoreStage('restoring_data')).toEqual({
      active: 1,
      labelKey: 'projectManagement.restoreRestoring',
    })
    expect(restoreStage('compensating').labelKey).toBe(
      'projectManagement.restoreCompensating',
    )
    expect(restoreStage('finalizing')).toEqual({
      active: 2,
      labelKey: 'projectManagement.restoreFinalizing',
    })
    expect(restoreStage('succeeded')).toEqual({
      active: 2,
      labelKey: 'projectManagement.restoreVerified',
    })
  })

  it('成功和失败才解除任务锁，并进入对应结果页', () => {
    expect(isRestoreTerminal({ taskId: 't1', state: 'restoring_scenes' })).toBe(false)
    expect(restoreView({ taskId: 't1', state: 'succeeded' })).toBe('success')
    expect(restoreView({ taskId: 't1', state: 'failed', rolledBack: true })).toBe('failure')
  })
})
