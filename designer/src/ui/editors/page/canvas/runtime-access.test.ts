import { describe, expect, it } from 'vitest'
import type { ComponentNode, PageRuntimeAccessConfig } from '@/editor-core/document/types'
import type { RuntimeUserRecord } from '@/stores/editor/project-runtime-role-actions'
import { createRuntimeAccessContext } from './runtime-access'

const role = (roleId: string, roleCode = roleId) => ({
  roleId,
  roleCode,
  roleName: roleCode,
})

const user = (roleIds: string[]): RuntimeUserRecord => ({
  id: 'user-1',
  username: 'operator',
  displayName: 'operator',
  status: 'active',
  roleIds,
  roles: [],
})

const node = (runtimeAccess: Record<string, string>): ComponentNode =>
  ({
    id: 'node-1',
    type: 'Button',
    permissions: {
      runtimeAccess,
    },
  }) as unknown as ComponentNode

describe('runtime-access', () => {
  it('does not restrict page or components when switch is disabled', () => {
    const context = createRuntimeAccessContext({
      config: {
        enabled: false,
        allowedRoles: [role('admin')],
        schemes: [{ id: 'admin-only', name: 'Admin', roleRefs: [role('admin')] }],
      },
      user: user(['viewer']),
    })

    expect(context.canViewPage).toBe(true)
    expect(context.isNodeVisible(node({ visibleSchemeId: 'missing' }))).toBe(true)
    expect(context.isNodeOperable(node({ operableSchemeId: 'missing' }))).toBe(true)
  })

  it('allows page access when enabled but no page roles are selected', () => {
    const context = createRuntimeAccessContext({
      config: {
        enabled: true,
        allowedRoles: [],
        schemes: [],
      },
      user: null,
    })

    expect(context.canViewPage).toBe(true)
  })

  it('evaluates schemes by runtime role id and denies missing schemes', () => {
    const config: PageRuntimeAccessConfig = {
      enabled: true,
      allowedRoles: [role('viewer')],
      schemes: [
        { id: 'viewer-visible', name: 'Viewer visible', roleRefs: [role('viewer')] },
        { id: 'admin-operable', name: 'Admin operable', roleRefs: [role('admin')] },
      ],
    }
    const context = createRuntimeAccessContext({
      config,
      user: user(['viewer']),
    })

    expect(context.canViewPage).toBe(true)
    expect(context.isNodeVisible(node({ visibleSchemeId: 'viewer-visible' }))).toBe(true)
    expect(context.isNodeOperable(node({ operableSchemeId: 'admin-operable' }))).toBe(false)
    expect(context.isNodeVisible(node({ visibleSchemeId: 'deleted-scheme' }))).toBe(false)
  })
})
