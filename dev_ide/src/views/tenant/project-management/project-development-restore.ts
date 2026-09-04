import type {
  ApplicationVersion,
  DevelopmentRestoreTask,
  DevelopmentRestoreTaskState,
} from '@/api/ops.api'

export type DevelopmentRestoreView = 'confirm' | 'progress' | 'success' | 'failure'

export function versionRestoreAvailability(version: ApplicationVersion): {
  allowed: boolean
  reason?: string
  reasonKey?: string
} {
  if (version.status !== 'ready') {
    return { allowed: false, reasonKey: 'projectManagement.restoreRequiresReadyVersion' }
  }
  if (!version.restorable) {
    if (version.restoreUnavailableReason) {
      return { allowed: false, reason: version.restoreUnavailableReason }
    }
    return { allowed: false, reasonKey: 'projectManagement.restoreSnapshotUnavailable' }
  }
  return { allowed: true }
}

export function restoreView(task: DevelopmentRestoreTask): DevelopmentRestoreView {
  if (task.state === 'succeeded') return 'success'
  if (task.state === 'failed') return 'failure'
  return 'progress'
}

export function restoreStage(state: DevelopmentRestoreTaskState): {
  active: number
  labelKey: string
} {
  if (state === 'queued' || state === 'staging') {
    return { active: 0, labelKey: 'projectManagement.restoreBackingUp' }
  }
  if (
    state === 'restoring_workspace' ||
    state === 'restoring_scenes' ||
    state === 'restoring_data'
  ) {
    return { active: 1, labelKey: 'projectManagement.restoreRestoring' }
  }
  if (state === 'compensating') {
    return { active: 1, labelKey: 'projectManagement.restoreCompensating' }
  }
  if (state === 'finalizing') {
    return { active: 2, labelKey: 'projectManagement.restoreFinalizing' }
  }
  if (state === 'succeeded') return { active: 2, labelKey: 'projectManagement.restoreVerified' }
  return { active: 2, labelKey: 'projectManagement.restoreVerifyFailed' }
}

export function isRestoreTerminal(task: DevelopmentRestoreTask | null): boolean {
  return task?.state === 'succeeded' || task?.state === 'failed'
}
