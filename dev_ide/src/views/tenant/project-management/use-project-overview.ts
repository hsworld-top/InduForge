import { computed, reactive, ref } from 'vue'
import { projectAPI } from '@/api/project.api'
import type {
  ProjectOverviewFiltersState,
  ProjectOverviewGroup,
  ProjectOverviewGroupContext,
  ProjectOverviewItem,
  ProjectOverviewPaginationState,
  ProjectOverviewQueryParams,
  ProjectOverviewRuntimeSummary,
  ProjectOverviewStateTree,
  ProjectOverviewTag,
  ProjectOverviewViewMode,
} from './project-overview.types'
import {
  buildProjectOverviewQueryParams,
  createDefaultProjectOverviewFilters,
  createEmptyProjectOverviewGroupContext,
  normalizeProjectOverviewFilters,
  normalizeProjectOverviewGroupContext,
} from './use-project-filters'

type FetchProjectsApi = (query: ProjectOverviewQueryParams) => Promise<unknown>

type PaginationSummaryFormatter = (input: { start: number; end: number; total: number }) => string

type UseProjectOverviewOptions = {
  fetchProjectsApi?: FetchProjectsApi
  initialFilters?: Partial<ProjectOverviewFiltersState>
  initialPagination?: Partial<ProjectOverviewPaginationState>
  initialGroupContext?: Partial<ProjectOverviewGroupContext>
  initialViewMode?: ProjectOverviewViewMode
  summaryFormatter?: PaginationSummaryFormatter
}

const DEFAULT_PAGE = 1
const DEFAULT_LIMIT = 20

const EMPTY_STATUS_COUNTS = {
  pending: 0,
  deploying: 0,
  running: 0,
  stopped: 0,
  error: 0,
  rollback: 0,
}

const EMPTY_MODE_COUNTS = {
  DEV: 0,
  RELEASE: 0,
}

const normalizeRuntimeModeValue = (value: unknown): 'DEV' | 'RELEASE' | '' => {
  const normalized = normalizeTextValue(value).toUpperCase()
  if (normalized === 'DEV' || normalized === 'DEVELOPMENT') {
    return 'DEV'
  }
  if (normalized === 'RELEASE' || normalized === 'PROD' || normalized === 'PRODUCTION') {
    return 'RELEASE'
  }
  return ''
}

const normalizeTextValue = (value: unknown): string => {
  if (typeof value === 'string') {
    return value.trim()
  }
  if (typeof value === 'number' && Number.isFinite(value)) {
    return String(value)
  }
  return ''
}

const normalizeIdValue = (value: unknown): string => {
  if (typeof value === 'string') {
    return value.trim()
  }
  if (typeof value === 'number' && Number.isFinite(value)) {
    return String(value)
  }
  return ''
}

const toPositiveInteger = (value: unknown, fallback: number): number => {
  const parsed = Number.parseInt(String(value), 10)
  return Number.isInteger(parsed) && parsed > 0 ? parsed : fallback
}

const toNonNegativeInteger = (value: unknown, fallback: number): number => {
  const parsed = Number.parseInt(String(value), 10)
  return Number.isInteger(parsed) && parsed >= 0 ? parsed : fallback
}

const asRecord = (value: unknown): Record<string, unknown> => {
  if (value && typeof value === 'object' && !Array.isArray(value)) {
    return value as Record<string, unknown>
  }
  return {}
}

const isApiEnvelope = (value: Record<string, unknown>): boolean => {
  return 'code' in value && 'msg' in value && 'data' in value
}

const pickArray = (candidates: unknown[]): unknown[] => {
  for (const candidate of candidates) {
    if (Array.isArray(candidate)) {
      return candidate
    }
  }
  return []
}

const pickRecord = (candidates: unknown[]): Record<string, unknown> => {
  for (const candidate of candidates) {
    const normalized = asRecord(candidate)
    if (Object.keys(normalized).length > 0) {
      return normalized
    }
  }
  return {}
}

const unwrapBusinessData = (response: unknown): Record<string, unknown> => {
  const root = asRecord(response)
  if (isApiEnvelope(root)) {
    return asRecord(root.data)
  }

  const rootData = asRecord(root.data)
  if (isApiEnvelope(rootData)) {
    return asRecord(rootData.data)
  }

  return Object.keys(rootData).length > 0 ? rootData : root
}

const normalizeTag = (value: unknown): ProjectOverviewTag | null => {
  const record = asRecord(value)
  const id = normalizeIdValue(record.id)
  if (!id) {
    return null
  }

  return {
    id,
    name: normalizeTextValue(record.name),
    description: normalizeTextValue(record.description) || null,
    sortOrder: toNonNegativeInteger(record.sortOrder, 0),
  }
}

const normalizeGroup = (value: unknown): ProjectOverviewGroup | null => {
  const record = asRecord(value)
  const id = normalizeIdValue(record.id)
  if (!id) {
    return null
  }

  return {
    id,
    name: normalizeTextValue(record.name),
    description: normalizeTextValue(record.description) || null,
    sortOrder: toNonNegativeInteger(record.sortOrder, 0),
  }
}

const normalizeRuntimeSummary = (value: unknown): ProjectOverviewRuntimeSummary => {
  const record = asRecord(value)
  const statusCounts = asRecord(record.statusCounts)
  const modeCounts = asRecord(record.modeCounts)
  const runtimeMode = normalizeRuntimeModeValue(record.runtimeMode) || 'DEV'

  return {
    runtimeStatus: normalizeTextValue(record.runtimeStatus) || 'UNKNOWN',
    runtimeMode,
    deploymentCount: toNonNegativeInteger(record.deploymentCount, 0),
    runningCount: toNonNegativeInteger(record.runningCount, 0),
    statusCounts: {
      pending: toNonNegativeInteger(statusCounts.pending, EMPTY_STATUS_COUNTS.pending),
      deploying: toNonNegativeInteger(statusCounts.deploying, EMPTY_STATUS_COUNTS.deploying),
      running: toNonNegativeInteger(statusCounts.running, EMPTY_STATUS_COUNTS.running),
      stopped: toNonNegativeInteger(statusCounts.stopped, EMPTY_STATUS_COUNTS.stopped),
      error: toNonNegativeInteger(statusCounts.error, EMPTY_STATUS_COUNTS.error),
      rollback: toNonNegativeInteger(statusCounts.rollback, EMPTY_STATUS_COUNTS.rollback),
    },
    modeCounts: {
      DEV: toNonNegativeInteger(modeCounts.DEV, EMPTY_MODE_COUNTS.DEV),
      RELEASE: toNonNegativeInteger(modeCounts.RELEASE, EMPTY_MODE_COUNTS.RELEASE),
    },
    lastDeployedAt: normalizeTextValue(record.lastDeployedAt) || null,
  }
}

const normalizeProjectOverviewItem = (value: unknown): ProjectOverviewItem => {
  const record = asRecord(value)
  const tags = pickArray([record.tags]).map(normalizeTag).filter(Boolean) as ProjectOverviewTag[]
  const runtimeSummary = normalizeRuntimeSummary(record.runtimeSummary)

  return {
    ...record,
    id: normalizeIdValue(record.id),
    name: normalizeTextValue(record.name),
    description: normalizeTextValue(record.description) || null,
    colorTag: normalizeTextValue(record.colorTag) || null,
    createdBy: normalizeIdValue(record.createdBy),
    updatedBy: normalizeIdValue(record.updatedBy),
    createdByName: normalizeTextValue(record.createdByName) || null,
    updatedByName: normalizeTextValue(record.updatedByName) || null,
    createdAt: normalizeTextValue(record.createdAt) || null,
    updatedAt: normalizeTextValue(record.updatedAt) || null,
    lastDeployedAt: normalizeTextValue(record.lastDeployedAt) || runtimeSummary.lastDeployedAt,
    group: normalizeGroup(record.group),
    tags,
    runtimeSummary,
  }
}

export const buildProjectPaginationSummary = ({
  page,
  limit,
  total,
}: {
  page: number
  limit: number
  total: number
}): string => {
  const normalizedTotal = toNonNegativeInteger(total, 0)
  const normalizedPage = toPositiveInteger(page, DEFAULT_PAGE)
  const normalizedLimit = toPositiveInteger(limit, DEFAULT_LIMIT)

  if (normalizedTotal <= 0) {
    return '0 / 0'
  }

  const totalPages = Math.max(1, Math.ceil(normalizedTotal / normalizedLimit))
  const safePage = Math.min(normalizedPage, totalPages)
  const start = (safePage - 1) * normalizedLimit + 1
  const end = Math.min(safePage * normalizedLimit, normalizedTotal)
  return `${start}-${end} / ${normalizedTotal}`
}

const buildInitialPagination = (
  input: Partial<ProjectOverviewPaginationState> | undefined,
): ProjectOverviewPaginationState => {
  const page = toPositiveInteger(input?.page, DEFAULT_PAGE)
  const limit = toPositiveInteger(input?.limit, DEFAULT_LIMIT)
  const total = toNonNegativeInteger(input?.total, 0)
  const totalPages =
    toNonNegativeInteger(input?.totalPages, 0) || (total > 0 ? Math.ceil(total / limit) : 0)

  return {
    page,
    limit,
    total,
    totalPages,
    summary: '',
  }
}

const resolveSummaryText = ({
  pagination,
  summaryFormatter,
}: {
  pagination: Pick<ProjectOverviewPaginationState, 'page' | 'limit' | 'total'>
  summaryFormatter?: PaginationSummaryFormatter
}): string => {
  if (!summaryFormatter) {
    return buildProjectPaginationSummary(pagination)
  }

  const total = toNonNegativeInteger(pagination.total, 0)
  if (total <= 0) {
    return summaryFormatter({ start: 0, end: 0, total: 0 })
  }

  const page = toPositiveInteger(pagination.page, DEFAULT_PAGE)
  const limit = toPositiveInteger(pagination.limit, DEFAULT_LIMIT)
  const start = (page - 1) * limit + 1
  const end = Math.min(page * limit, total)

  return summaryFormatter({ start, end, total })
}

/**
 * 兼容 request 拦截器与旧调用方式，稳定提取 projects 与 pagination。
 */
const extractProjectOverviewResponse = (
  response: unknown,
): {
  projects: unknown[]
  pagination: Record<string, unknown>
} => {
  const root = asRecord(response)
  const rootData = asRecord(root.data)
  const rootDataData = asRecord(rootData.data)
  const businessData = unwrapBusinessData(response)
  const businessList = asRecord(businessData.list)
  const rootList = asRecord(root.list)
  const rootDataList = asRecord(rootData.list)
  const rootDataDataList = asRecord(rootDataData.list)

  const projects = pickArray([
    businessList.projects,
    businessData.projects,
    businessList.items,
    businessData.items,
    Array.isArray(businessData.list) ? businessData.list : null,
    rootDataDataList.projects,
    rootDataData.projects,
    rootDataDataList.items,
    rootDataData.items,
    Array.isArray(rootDataData.list) ? rootDataData.list : null,
    rootDataList.projects,
    rootData.projects,
    rootDataList.items,
    rootData.items,
    Array.isArray(rootData.list) ? rootData.list : null,
    rootList.projects,
    root.projects,
    rootList.items,
    root.items,
    Array.isArray(root.list) ? root.list : null,
  ])

  const pagination = pickRecord([
    businessData.pagination,
    businessList.pagination,
    rootDataData.pagination,
    rootDataDataList.pagination,
    rootData.pagination,
    rootDataList.pagination,
    root.pagination,
    rootList.pagination,
  ])
  return { projects, pagination }
}

export const useProjectOverviewState = (options: UseProjectOverviewOptions = {}) => {
  const fetchProjectsApi: FetchProjectsApi = options.fetchProjectsApi || projectAPI.getProjects
  const projects = ref<ProjectOverviewItem[]>([])
  const loading = ref(false)
  const lastQuery = ref<ProjectOverviewQueryParams>({})

  const state = reactive<ProjectOverviewStateTree>({
    filters: normalizeProjectOverviewFilters({
      ...createDefaultProjectOverviewFilters(),
      ...(options.initialFilters || {}),
      composite: {
        ...createDefaultProjectOverviewFilters().composite,
        ...(options.initialFilters?.composite || {}),
      },
    }),
    pagination: buildInitialPagination(options.initialPagination),
    groupContext: normalizeProjectOverviewGroupContext(
      options.initialGroupContext || createEmptyProjectOverviewGroupContext(),
    ),
    viewMode: options.initialViewMode === 'list' ? 'list' : 'card',
  })

  const refreshPaginationSummary = () => {
    state.pagination.summary = resolveSummaryText({
      pagination: state.pagination,
      summaryFormatter: options.summaryFormatter,
    })
  }

  const patchFilters = (patch: Partial<ProjectOverviewFiltersState>) => {
    const merged: Partial<ProjectOverviewFiltersState> = {
      ...state.filters,
      ...patch,
      composite: {
        ...state.filters.composite,
        ...(patch.composite || {}),
      },
    }
    Object.assign(state.filters, normalizeProjectOverviewFilters(merged))
    state.pagination.page = DEFAULT_PAGE
    refreshPaginationSummary()
  }

  const setSort = (
    sortBy: ProjectOverviewFiltersState['sortBy'],
    sortOrder: ProjectOverviewFiltersState['sortOrder'],
  ) => {
    patchFilters({ sortBy, sortOrder })
  }

  const setGroupContext = (context: Partial<ProjectOverviewGroupContext>) => {
    const patch: Partial<ProjectOverviewGroupContext> = {}
    if (Object.prototype.hasOwnProperty.call(context, 'groupId')) {
      patch.groupId = normalizeTextValue(context.groupId)
    }
    if (Object.prototype.hasOwnProperty.call(context, 'groupName')) {
      patch.groupName = normalizeTextValue(context.groupName)
    }

    Object.assign(state.groupContext, patch)
    state.pagination.page = DEFAULT_PAGE
    refreshPaginationSummary()
  }

  const clearGroupContext = () => {
    Object.assign(state.groupContext, createEmptyProjectOverviewGroupContext())
    state.pagination.page = DEFAULT_PAGE
    refreshPaginationSummary()
  }

  const setPage = (page: number) => {
    state.pagination.page = toPositiveInteger(page, DEFAULT_PAGE)
    refreshPaginationSummary()
  }

  const setLimit = (limit: number) => {
    state.pagination.limit = toPositiveInteger(limit, DEFAULT_LIMIT)
    state.pagination.page = DEFAULT_PAGE
    refreshPaginationSummary()
  }

  const setViewMode = (mode: ProjectOverviewViewMode) => {
    state.viewMode = mode === 'list' ? 'list' : 'card'
  }

  const currentQuery = computed<ProjectOverviewQueryParams>(() =>
    buildProjectOverviewQueryParams({
      filters: state.filters,
      pagination: state.pagination,
      groupContext: state.groupContext,
    }),
  )

  const applyPagination = (pagination: Record<string, unknown>) => {
    const rawPage = toPositiveInteger(
      pagination.page,
      toPositiveInteger(currentQuery.value.page, state.pagination.page),
    )
    const rawLimit = toPositiveInteger(
      pagination.limit,
      toPositiveInteger(currentQuery.value.limit, state.pagination.limit),
    )
    const total = toNonNegativeInteger(pagination.total, 0)
    const totalPages =
      toNonNegativeInteger(pagination.totalPages, 0) ||
      (total > 0 ? Math.ceil(total / rawLimit) : 0)
    const page = totalPages > 0 ? Math.min(rawPage, totalPages) : DEFAULT_PAGE

    Object.assign(state.pagination, {
      page,
      limit: rawLimit,
      total,
      totalPages,
    })
    refreshPaginationSummary()
  }

  const fetchProjects = async (extraQuery: Partial<ProjectOverviewQueryParams> = {}) => {
    loading.value = true
    try {
      const query = {
        ...currentQuery.value,
        ...extraQuery,
      }
      lastQuery.value = query

      const response = await fetchProjectsApi(query)
      const { projects: rawProjects, pagination } = extractProjectOverviewResponse(response)
      projects.value = rawProjects.map((item) => normalizeProjectOverviewItem(item))
      applyPagination(pagination)
      return projects.value
    } finally {
      loading.value = false
    }
  }

  refreshPaginationSummary()

  const viewMode = computed<ProjectOverviewViewMode>({
    get: () => state.viewMode,
    set: (mode) => setViewMode(mode),
  })

  return {
    state,
    projects,
    loading,
    lastQuery,
    filters: state.filters,
    pagination: state.pagination,
    groupContext: state.groupContext,
    viewMode,
    currentQuery,
    patchFilters,
    setSort,
    setGroupContext,
    clearGroupContext,
    setPage,
    setLimit,
    setViewMode,
    refreshPaginationSummary,
    fetchProjects,
  }
}
