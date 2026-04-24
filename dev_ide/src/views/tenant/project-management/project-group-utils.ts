import type {
  ProjectGroupCardViewModel,
  ProjectGroupProjectPreview,
  ProjectOverviewGroup,
  ProjectStatusMetric,
} from './project-overview.types'

type ProjectGroupSource =
  | (Omit<
      Partial<ProjectOverviewGroup>,
      'id' | 'name' | 'description' | 'sortOrder' | 'projectCount'
    > & {
      id?: unknown
      name?: unknown
      description?: unknown
      sortOrder?: unknown
      projectCount?: unknown
    })
  | null
  | undefined
type ProjectPreviewSource =
  | {
      id?: unknown
      name?: unknown
      group?: { id?: unknown; name?: unknown; [key: string]: unknown } | null
      projectId?: unknown
      groupId?: unknown
      [key: string]: unknown
    }
  | null
  | undefined

const normalizeTextValue = (value: unknown): string => {
  if (value === null || value === undefined) {
    return ''
  }

  return String(value).trim()
}

const normalizeSortOrder = (value: unknown): number => {
  const parsed = Number(value)
  return Number.isFinite(parsed) ? parsed : 0
}

const normalizeProjectCount = (value: unknown): number | null => {
  if (value === null || value === undefined || value === '') {
    return null
  }

  const parsed = Number.parseInt(String(value), 10)
  return Number.isInteger(parsed) && parsed >= 0 ? parsed : null
}

const resolveProjectId = (project: ProjectPreviewSource): string => {
  return normalizeTextValue(project?.id) || normalizeTextValue(project?.projectId)
}

const resolveProjectGroupId = (project: ProjectPreviewSource): string =>
  normalizeTextValue(project?.group?.id) || normalizeTextValue(project?.groupId)

/**
 * 构建分组卡片视图数据，统一处理接口分组与当前页工程预览的松散字段。
 */
export const buildProjectGroupCardItems = (
  groups: readonly ProjectGroupSource[] = [],
  projects: readonly ProjectPreviewSource[] = [],
): ProjectGroupCardViewModel[] => {
  const projectPreviewMap = new Map<string, ProjectGroupProjectPreview[]>()

  projects.forEach((project) => {
    const groupId = resolveProjectGroupId(project)
    const projectId = resolveProjectId(project)
    const name = normalizeTextValue(project?.name)
    if (!groupId || !projectId || !name) {
      return
    }

    const previews = projectPreviewMap.get(groupId) || []
    previews.push({ id: projectId, name })
    projectPreviewMap.set(groupId, previews)
  })

  const result: ProjectGroupCardViewModel[] = []

  groups.forEach((group) => {
    const id = normalizeTextValue(group?.id)
    if (!id) {
      return
    }

    const projectsInGroup = projectPreviewMap.get(id) || []
    const explicitProjectCount = normalizeProjectCount(group?.projectCount)

    result.push({
      id,
      name: normalizeTextValue(group?.name),
      description:
        group?.description === null || group?.description === undefined
          ? null
          : normalizeTextValue(group.description),
      sortOrder: normalizeSortOrder(group?.sortOrder),
      projectCount: explicitProjectCount ?? projectsInGroup.length,
      projects: projectsInGroup,
    })
  })

  return result.sort((left, right) => {
    const sortDelta = (left.sortOrder ?? 0) - (right.sortOrder ?? 0)
    if (sortDelta !== 0) {
      return sortDelta
    }

    return left.name.localeCompare(right.name, 'zh-Hans-CN')
  })
}

export const resolveProjectStatusMetric = (status: unknown): ProjectStatusMetric => {
  switch (normalizeTextValue(status).toLowerCase()) {
    case 'running':
      return { labelKey: 'projectManagement.deployStatusRunning', percent: 100, level: 'good' }
    case 'deploying':
      return { labelKey: 'projectManagement.deployStatusDeploying', percent: 72, level: 'warn' }
    case 'pending':
      return { labelKey: 'projectManagement.deployStatusPending', percent: 48, level: 'warn' }
    case 'stopped':
      return { labelKey: 'projectManagement.deployStatusStopped', percent: 36, level: 'muted' }
    case 'rollback':
      return { labelKey: 'projectManagement.deployStatusRollback', percent: 28, level: 'bad' }
    case 'error':
      return { labelKey: 'projectManagement.deployStatusError', percent: 20, level: 'bad' }
    default:
      return { labelKey: 'projectManagement.deployStatusUnknown', percent: 0, level: 'muted' }
  }
}
