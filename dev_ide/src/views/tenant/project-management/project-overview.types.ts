import type {
  ProjectDeployStatus,
  ProjectListQueryParams,
  ProjectOverviewSortField,
  ProjectOverviewSortOrder,
  ProjectRuntimeMode,
  ProjectVisibility,
} from '@/api/project.api'

export type ProjectOverviewViewMode = 'card' | 'list'

export interface ProjectOverviewTag {
  id: string
  name: string
  description?: string | null
  sortOrder?: number
}

export interface ProjectOverviewGroup {
  id: string
  name: string
  description?: string | null
  sortOrder?: number
  projectCount?: number
}

export interface ProjectGroupProjectPreview {
  id: string
  name: string
}

export interface ProjectGroupCardViewModel extends ProjectOverviewGroup {
  projectCount: number
  projects: ProjectGroupProjectPreview[]
}

export type ProjectStatusMetricLevel = 'good' | 'warn' | 'bad' | 'muted'

export interface ProjectStatusMetric {
  labelKey: string
  percent: number
  level: ProjectStatusMetricLevel
}

export interface ProjectOverviewRuntimeStatusCounts {
  pending: number
  deploying: number
  running: number
  stopped: number
  error: number
  rollback: number
}

export interface ProjectOverviewRuntimeModeCounts {
  DEV: number
  RELEASE: number
}

export interface ProjectOverviewRuntimeNode {
  id: string
  name: string
  ipAddress?: string | null
  nodeStatus?: string | null
  deployStatus?: string | null
  mode?: string | null
}

export interface ProjectOverviewRuntimeSummary {
  runtimeStatus: string
  runtimeMode?: ProjectRuntimeMode | string | null
  deploymentCount: number
  runningCount: number
  statusCounts: ProjectOverviewRuntimeStatusCounts
  modeCounts: ProjectOverviewRuntimeModeCounts
  nodes?: ProjectOverviewRuntimeNode[]
  lastDeployedAt: string | null
}

export interface ProjectOverviewItem {
  id: string
  name: string
  description?: string | null
  colorTag?: string | null
  visibility?: ProjectVisibility | string | null
  createdBy?: string
  updatedBy?: string
  createdByName?: string | null
  updatedByName?: string | null
  createdAt?: string | null
  updatedAt?: string | null
  lastDeployedAt?: string | null
  tags: ProjectOverviewTag[]
  group: ProjectOverviewGroup | null
  runtimeSummary: ProjectOverviewRuntimeSummary
  [key: string]: unknown
}

export interface ProjectOverviewCompositeFilters {
  runtimeModes: ProjectRuntimeMode[]
  deployStatuses: ProjectDeployStatus[]
  visibility: ProjectVisibility[]
  createdBy: string
}

export interface ProjectOverviewFiltersState {
  search: string
  composite: ProjectOverviewCompositeFilters
  tagIds: string[]
  sortBy: ProjectOverviewSortField
  sortOrder: ProjectOverviewSortOrder
}

export interface ProjectOverviewPaginationState {
  page: number
  limit: number
  total: number
  totalPages: number
  summary: string
}

export interface ProjectOverviewGroupContext {
  groupId: string
  groupName: string
}

export type ProjectOverviewQueryParams = ProjectListQueryParams

export interface ProjectOverviewStateTree {
  filters: ProjectOverviewFiltersState
  pagination: ProjectOverviewPaginationState
  groupContext: ProjectOverviewGroupContext
  viewMode: ProjectOverviewViewMode
}
