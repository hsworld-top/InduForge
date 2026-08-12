import { describe, expect, it } from 'vitest'
import { buildHtWorkspaceUrl, resolveWorkspaceKey, sanitizeCodeServerUrl } from './workspace-links'

describe('workspace links', () => {
  it('builds source-based HT entry links', () => {
    expect(buildHtWorkspaceUrl('/designer/', 'project-1', '2d')).toBe(
      '/designer/ht-editor/index.html?projectId=project-1',
    )
    expect(buildHtWorkspaceUrl('/designer/', 'project-1', '3d-scene')).toBe(
      '/designer/ht-editor/index3d.html?projectId=project-1&workspace=scene',
    )
    expect(buildHtWorkspaceUrl('/designer/', 'project-1', '3d-model')).toBe(
      '/designer/ht-editor/index3d.html?projectId=project-1&workspace=model',
    )
  })

  it('falls back to code workspace for unknown values', () => {
    expect(resolveWorkspaceKey('unknown')).toBe('code')
    expect(resolveWorkspaceKey('2d')).toBe('2d')
  })

  it('accepts only http code-server URLs', () => {
    expect(sanitizeCodeServerUrl('http://localhost:18080')).toBe('http://localhost:18080/')
    expect(sanitizeCodeServerUrl('javascript:alert(1)')).toBeNull()
  })
})
