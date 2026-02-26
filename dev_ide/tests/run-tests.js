import assert from 'node:assert/strict'

import { ROLES } from '../src/constants/index.js'
import {
  hasRole,
  canAccessTab,
  canManageUsers,
  canApproveNodes,
  getTabAccessDeniedMessage,
} from '../src/permissions/rules.js'
import {
  buildDeployStatusSummary,
  getDeployStatusType,
  getDeployLabel,
  isFailedDeploy,
  getDeployFailureReason,
} from '../src/views/tenant/utils/ops-status.js'

const run = (name, fn) => {
  try {
    fn()
    console.log(`PASS ${name}`)
  } catch (error) {
    console.error(`FAIL ${name}`)
    console.error(error)
    process.exitCode = 1
  }
}

run('权限规则：hasRole', () => {
  assert.equal(hasRole(ROLES.SYSTEM_ADMIN, [ROLES.SYSTEM_ADMIN, ROLES.OPS_ADMIN]), true)
  assert.equal(hasRole(ROLES.USER, [ROLES.SYSTEM_ADMIN]), false)
  assert.equal(hasRole('', [ROLES.SYSTEM_ADMIN]), false)
})

run('权限规则：canAccessTab', () => {
  assert.equal(canAccessTab('tenant-management', ROLES.SUPER_ADMIN), true)
  assert.equal(canAccessTab('tenant-management', ROLES.SYSTEM_ADMIN), false)
  assert.equal(canAccessTab('dashboard', ROLES.USER), true)
})

run('权限规则：业务判定函数', () => {
  assert.equal(canManageUsers(ROLES.USER_ADMIN), true)
  assert.equal(canManageUsers(ROLES.OPS_ADMIN), false)
  assert.equal(canApproveNodes(ROLES.SYSTEM_ADMIN), true)
  assert.equal(canApproveNodes(ROLES.USER_ADMIN), false)
  assert.equal(getTabAccessDeniedMessage('system-settings'), '只有系统管理员才能访问系统设置')
})

run('运维状态：聚合统计', () => {
  const summary = buildDeployStatusSummary([
    { deployments: [{ status: 'running' }, { status: 'deploying' }, { status: 'error' }] },
    { deployments: [{ status: 'pending' }, { status: 'stopped' }, { status: 'failed' }] },
  ])
  assert.deepEqual(summary, { running: 1, deploying: 2, stopped: 1, failed: 2 })
})

run('运维状态：映射与失败判定', () => {
  assert.equal(getDeployStatusType('running'), 'success')
  assert.equal(getDeployStatusType('failed'), 'danger')
  assert.equal(getDeployLabel('pending'), '等待中')
  assert.equal(isFailedDeploy({ status: 'error' }), true)
  assert.equal(getDeployFailureReason({ errorMessage: 'network error' }), 'network error')
})

if (process.exitCode !== 1) {
  console.log('ALL TESTS PASSED')
}
