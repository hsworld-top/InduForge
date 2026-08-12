import type {
  ProjectDeployStatus,
  ProjectOverviewSortField,
  ProjectOverviewSortOrder,
  ProjectRuntimeMode,
  ProjectVisibility,
} from '@/api/project.api'
import type {
  ProjectOverviewCompositeFilters,
  ProjectOverviewFiltersState,
  ProjectOverviewGroupContext,
  ProjectOverviewPaginationState,
  ProjectOverviewQueryParams,
} from './project-overview.types'

interface ProjectOverviewFiltersInput {
  search?: unknown
  composite?: Partial<Record<keyof ProjectOverviewCompositeFilters, unknown>> | null
  tagIds?: unknown
  sortBy?: unknown
  sortOrder?: unknown
}

const DEFAULT_PAGE = 1
const DEFAULT_LIMIT = 20
const DEFAULT_SORT_BY: ProjectOverviewSortField = 'createdAt'
const DEFAULT_SORT_ORDER: ProjectOverviewSortOrder = 'DESC'

const SORT_FIELD_SET = new Set<ProjectOverviewSortField>([
  'createdAt',
  'updatedAt',
  'lastDeployedAt',
  'runtimeStatus',
])
const RUNTIME_MODE_SET = new Set<ProjectRuntimeMode>(['DEV', 'RELEASE'])
const VISIBILITY_SET = new Set<ProjectVisibility>(['private', 'internal'])
const DEPLOY_STATUS_SET = new Set<ProjectDeployStatus>([
  'pending',
  'deploying',
  'running',
  'stopped',
  'error',
  'rollback',
])

const normalizeTextValue = (value: unknown): string => {
  if (typeof value === 'string') {
    return value.trim()
  }
  if (typeof value === 'number' && Number.isFinite(value)) {
    return String(value)
  }
  return ''
}

const normalizePositiveInteger = (value: unknown, fallback: number): number => {
  const parsed = Number.parseInt(String(value), 10)
  return Number.isInteger(parsed) && parsed > 0 ? parsed : fallback
}

const uniqueStringList = (values: unknown): string[] => {
  if (!Array.isArray(values)) {
    return []
  }

  const normalized = values.map((item) => normalizeTextValue(item)).filter(Boolean)
  return [...new Set(normalized)]
}

const normalizeSortField = (value: unknown): ProjectOverviewSortField => {
  const candidate = normalizeTextValue(value) as ProjectOverviewSortField
  return SORT_FIELD_SET.has(candidate) ? candidate : DEFAULT_SORT_BY
}

const normalizeSortOrder = (value: unknown): ProjectOverviewSortOrder => {
  const candidate = normalizeTextValue(value).toUpperCase()
  return candidate === 'ASC' || candidate === 'DESC' ? candidate : DEFAULT_SORT_ORDER
}

const normalizeRuntimeModes = (values: unknown): ProjectRuntimeMode[] => {
  const normalized = uniqueStringList(values)
    .map((item) => item.toUpperCase())
    .filter((item): item is ProjectRuntimeMode => RUNTIME_MODE_SET.has(item as ProjectRuntimeMode))

  return [...new Set(normalized)]
}

const normalizeDeployStatuses = (values: unknown): ProjectDeployStatus[] => {
  const normalized = uniqueStringList(values)
    .map((item) => item.toLowerCase())
    .filter((item): item is ProjectDeployStatus =>
      DEPLOY_STATUS_SET.has(item as ProjectDeployStatus),
    )

  return [...new Set(normalized)]
}

const normalizeVisibility = (values: unknown): ProjectVisibility[] => {
  const normalized = uniqueStringList(values)
    .map((item) => item.toLowerCase())
    .filter((item): item is ProjectVisibility => VISIBILITY_SET.has(item as ProjectVisibility))

  return [...new Set(normalized)]
}

export const createDefaultProjectOverviewFilters = (): ProjectOverviewFiltersState => ({
  search: '',
  composite: {
    runtimeModes: [],
    deployStatuses: [],
    visibility: [],
    createdBy: '',
  },
  tagIds: [],
  sortBy: DEFAULT_SORT_BY,
  sortOrder: DEFAULT_SORT_ORDER,
})

export const createEmptyProjectOverviewGroupContext = (): ProjectOverviewGroupContext => ({
  groupId: '',
  groupName: '',
})

export const normalizeProjectOverviewGroupContext = (
  value?: Partial<ProjectOverviewGroupContext> | null,
): ProjectOverviewGroupContext => ({
  groupId: normalizeTextValue(value?.groupId),
  groupName: normalizeTextValue(value?.groupName),
})

export const normalizeProjectOverviewFilters = (
  candidate: ProjectOverviewFiltersInput = {},
): ProjectOverviewFiltersState => {
  const composite = candidate.composite ?? {}

  return {
    search: normalizeTextValue(candidate.search),
    composite: {
      runtimeModes: normalizeRuntimeModes(composite.runtimeModes),
      deployStatuses: normalizeDeployStatuses(composite.deployStatuses),
      visibility: normalizeVisibility(composite.visibility),
      createdBy: normalizeTextValue(composite.createdBy),
    },
    tagIds: uniqueStringList(candidate.tagIds),
    sortBy: normalizeSortField(candidate.sortBy),
    sortOrder: normalizeSortOrder(candidate.sortOrder),
  }
}

/**
 * 统一构造总览查询对象，避免页面层拼参数时遗漏筛选字段。
 */
export const buildProjectOverviewQueryParams = ({
  filters,
  pagination,
  groupContext,
}: {
  filters: ProjectOverviewFiltersInput
  pagination: Pick<ProjectOverviewPaginationState, 'page' | 'limit'>
  groupContext?: Partial<ProjectOverviewGroupContext> | null
}): ProjectOverviewQueryParams => {
  const normalizedFilters = normalizeProjectOverviewFilters(filters)
  const normalizedGroupContext = normalizeProjectOverviewGroupContext(groupContext)
  const query: ProjectOverviewQueryParams = {
    page: normalizePositiveInteger(pagination.page, DEFAULT_PAGE),
    limit: normalizePositiveInteger(pagination.limit, DEFAULT_LIMIT),
    sortBy: normalizedFilters.sortBy,
    sortOrder: normalizedFilters.sortOrder,
  }

  if (normalizedFilters.search) {
    query.name = normalizedFilters.search
  }

  if (normalizedFilters.tagIds.length > 0) {
    query.tagId = normalizedFilters.tagIds
  }

  if (normalizedFilters.composite.runtimeModes.length > 0) {
    query.runtimeMode = normalizedFilters.composite.runtimeModes
  }

  if (normalizedFilters.composite.deployStatuses.length > 0) {
    query.deployStatus = normalizedFilters.composite.deployStatuses
  }

  if (normalizedFilters.composite.visibility.length > 0) {
    query.visibility = normalizedFilters.composite.visibility
  }

  if (normalizedFilters.composite.createdBy) {
    query.createdBy = normalizedFilters.composite.createdBy
  }

  if (normalizedGroupContext.groupId) {
    query.groupId = normalizedGroupContext.groupId
  }

  return query
}
