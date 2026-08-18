import { describe, expect, it } from 'vitest'
import {
  isWorkspaceOpenRequest,
  matchesWorkspaceRequestProject,
  workspaceToolTabKey,
  workspaceToolTitle,
} from '@/types/workspace-tool'

describe('workspace tool contract', () => {
  it('只接受固定工程工具目标', () => {
    expect(
      isWorkspaceOpenRequest({
        type: 'WORKSPACE_OPEN_REQUEST',
        projectId: 'project-1',
        target: '2d',
        sceneId: 'overview',
      }),
    ).toBe(true)
    expect(
      isWorkspaceOpenRequest({
        type: 'WORKSPACE_OPEN_REQUEST',
        projectId: 'project-1',
        target: 'code',
        sceneId: 'overview',
      }),
    ).toBe(false)
  })

  it('生成工程隔离的稳定标签键和标题', () => {
    expect(workspaceToolTabKey('project-1', '2d', 'overview')).toBe('project-1:2d:overview')
    expect(workspaceToolTabKey('project-2', '2d', 'overview')).toBe('project-2:2d:overview')
    expect(workspaceToolTitle('3d')).toBe('3D')
  })

  it('固定生成 2D 与 3D 场景入口且校验来源工程', () => {
    const request = {
      type: 'WORKSPACE_OPEN_REQUEST' as const,
      projectId: 'project/1',
      target: '3d' as const,
      sceneId: 'factory',
    }
    expect(matchesWorkspaceRequestProject(request, 'project/1')).toBe(true)
    expect(matchesWorkspaceRequestProject(request, 'project-2')).toBe(false)
  })
})
