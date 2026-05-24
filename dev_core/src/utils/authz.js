const ROLE_CAPABILITIES = {
  SUPER_ADMIN: ['tenant:manage'],
  SYSTEM_ADMIN: ['*'],
  PROJECT_ADMIN: [
    'project:read',
    'project:write',
    'release:publish',
    'deploy:execute',
    'runtime:operate',
    'node:read',
  ],
  OPS_ADMIN: [
    'project:read',
    'release:publish',
    'deploy:execute',
    'runtime:operate',
    'node:read',
    'node:approve',
  ],
  USER_ADMIN: ['user:write'],
  USER: ['project:read'],
}

const GLOBAL_PROJECT_ACCESS_ROLES = new Set(['SUPER_ADMIN', 'SYSTEM_ADMIN'])

function normalizeProjectIds(projectIds, role) {
  if (GLOBAL_PROJECT_ACCESS_ROLES.has(role)) {
    return ['*']
  }

  if (!Array.isArray(projectIds)) {
    return []
  }

  return [...new Set(projectIds.map((item) => String(item || '').trim()).filter(Boolean))]
}

function listCapabilitiesForRole(role) {
  const normalizedRole = String(role || '').trim()
  return [...(ROLE_CAPABILITIES[normalizedRole] || [])]
}

function hasCapability(userRole, capability) {
  if (!userRole || !capability) return false
  const caps = listCapabilitiesForRole(userRole)
  return caps.includes('*') || caps.includes(capability)
}

async function listAccessibleProjectIds(ProjectModel, { tenantId, role }) {
  const normalizedRole = String(role || '').trim()
  if (GLOBAL_PROJECT_ACCESS_ROLES.has(normalizedRole)) {
    return ['*']
  }

  const normalizedTenantId = String(tenantId || '').trim()
  if (!normalizedTenantId || !ProjectModel || typeof ProjectModel.findAll !== 'function') {
    return []
  }

  const projects = await ProjectModel.findAll({
    where: { tenantId: normalizedTenantId },
    attributes: ['id'],
  })

  return normalizeProjectIds(
    projects.map((item) => item.id),
    normalizedRole,
  )
}

function buildAccessTokenPayload({ userId, username, role, tenantId, projectIds }) {
  const normalizedRole = String(role || '').trim()

  return {
    userId,
    username,
    role: normalizedRole,
    tenantId,
    capabilities: listCapabilitiesForRole(normalizedRole),
    projectIds: normalizeProjectIds(projectIds, normalizedRole),
  }
}

module.exports = {
  ROLE_CAPABILITIES,
  buildAccessTokenPayload,
  hasCapability,
  listAccessibleProjectIds,
  listCapabilitiesForRole,
}
