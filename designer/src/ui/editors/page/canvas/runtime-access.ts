import type {
  ComponentNode,
  PageRuntimeAccessConfig,
  RuntimeRoleRef,
} from '@/editor-core/document/types'
import type { RuntimeUserRecord } from '@/stores/editor/project-runtime-role-actions'

export interface RuntimeAccessContext {
  enabled: boolean
  schemes: PageRuntimeAccessConfig['schemes']
  user: RuntimeUserRecord | null
  canViewPage: boolean
  evaluateScheme: (schemeId?: string) => boolean
  isNodeVisible: (node: ComponentNode | Record<string, any>) => boolean
  isNodeOperable: (node: ComponentNode | Record<string, any>) => boolean
}

const normalizeId = (value: unknown): string => String(value ?? '').trim()

function collectUserRoleIds(user: RuntimeUserRecord | null): Set<string> {
  const ids = new Set<string>()
  if (!user) return ids
  user.roleIds.forEach((roleId) => {
    const normalized = normalizeId(roleId)
    if (normalized) ids.add(normalized)
  })
  user.roles.forEach((role) => {
    if (role.id) ids.add(role.id)
    if (role.roleId) ids.add(role.roleId)
  })
  return ids
}

function hasAnyRole(roleRefs: RuntimeRoleRef[] | undefined, roleIds: Set<string>): boolean {
  const refs = Array.isArray(roleRefs) ? roleRefs : []
  if (refs.length === 0) return true
  for (const ref of refs) {
    if (roleIds.has(ref.roleId)) return true
  }
  return false
}

export function createRuntimeAccessContext(input: {
  config?: PageRuntimeAccessConfig | null
  user?: RuntimeUserRecord | null
}): RuntimeAccessContext {
  const config = input.config || null
  const enabled = Boolean(config?.enabled)
  const schemes = Array.isArray(config?.schemes) ? config.schemes : []
  const user = input.user || null
  const userRoleIds = collectUserRoleIds(user)
  const canViewPage = !enabled || hasAnyRole(config?.allowedRoles, userRoleIds)

  const evaluateScheme = (schemeId?: string): boolean => {
    if (!enabled) return true
    const normalizedSchemeId = normalizeId(schemeId)
    if (!normalizedSchemeId) return true
    const scheme = schemes.find((item) => item.id === normalizedSchemeId)
    if (!scheme) return false
    return hasAnyRole(scheme.roleRefs, userRoleIds)
  }

  return {
    enabled,
    schemes,
    user,
    canViewPage,
    evaluateScheme,
    isNodeVisible(node) {
      return evaluateScheme(node?.permissions?.runtimeAccess?.visibleSchemeId)
    },
    isNodeOperable(node) {
      return evaluateScheme(node?.permissions?.runtimeAccess?.operableSchemeId)
    },
  }
}
