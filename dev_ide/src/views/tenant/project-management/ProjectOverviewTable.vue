<template>
  <el-table
    :data="tableRows"
    row-key="id"
    :span-method="handleSpanMethod"
    :row-class-name="resolveRowClassName"
    :empty-text="resolvedEmptyDescription"
    v-loading="loading"
    @row-click="handleRowClick"
  >
    <el-table-column v-if="showSelection" width="56" align="center" fixed="left">
      <template #default="{ row }">
        <el-checkbox
          v-if="isItemRow(row)"
          :model-value="isSelected(row.project.id)"
          @click.stop
          @change="(value) => handleSelectionChange(row.project.id, value)"
        />
      </template>
    </el-table-column>

    <el-table-column :label="t('projectManagement.projectName')" min-width="160">
      <template #default="{ row }">
        <div v-if="isGroupRow(row)" class="project-overview-table__group-header">
          <div class="project-overview-table__group-main">
            <span class="project-overview-table__group-icon">
              <el-icon><FolderOpened /></el-icon>
            </span>
            <div class="project-overview-table__group-copy">
              <span class="project-overview-table__group-name">{{ row.groupName }}</span>
              <span class="project-overview-table__group-count">
                {{ t('projectManagement.projectCountUnit', { count: row.projectCount }) }}
              </span>
            </div>
          </div>
        </div>
        <button
          v-else
          type="button"
          class="text-left text-sm font-medium text-blue-600 hover:text-blue-700 truncate max-w-full"
          @click.stop="emit('open-project', row.project)"
        >
          {{ row.project.name }}
        </button>
      </template>
    </el-table-column>

    <el-table-column v-if="showTagColumn" :label="t('projectManagement.tagFilter')" min-width="180">
      <template #default="{ row }">
        <div v-if="isItemRow(row)" class="flex flex-wrap gap-1">
          <el-tag
            v-for="tag in row.project.tags"
            :key="tag.id"
            size="small"
            effect="plain"
            type="info"
            class="!mr-0"
          >
            {{ tag.name }}
          </el-tag>
          <span v-if="row.project.tags.length === 0" class="text-xs text-gray-400">--</span>
        </div>
      </template>
    </el-table-column>

    <el-table-column
      v-if="showCreatorColumn"
      :label="t('projectManagement.createdBy')"
      width="140"
      show-overflow-tooltip
    >
      <template #default="{ row }">
        <span v-if="isItemRow(row)">{{
          row.project.createdByName || row.project.createdBy || '--'
        }}</span>
      </template>
    </el-table-column>

    <el-table-column
      v-if="showRuntimeModeColumn"
      :label="t('projectManagement.runtimeMode')"
      width="110"
      align="center"
    >
      <template #default="{ row }">
        <el-tag v-if="isItemRow(row)" size="small" type="info" effect="plain">
          {{ resolveRuntimeModeText(row.project) }}
        </el-tag>
      </template>
    </el-table-column>

    <el-table-column
      v-if="showRuntimeStatusColumn"
      :label="t('projectManagement.deployStatus')"
      width="120"
      align="center"
    >
      <template #default="{ row }">
        <el-tag
          v-if="isItemRow(row)"
          size="small"
          :type="resolveRuntimeTag(row.project.runtimeSummary.runtimeStatus)"
        >
          {{ resolveRuntimeText(row.project.runtimeSummary.runtimeStatus) }}
        </el-tag>
      </template>
    </el-table-column>

    <el-table-column
      v-if="showDescriptionColumn"
      :label="t('projectManagement.description')"
      min-width="280"
      show-overflow-tooltip
    >
      <template #default="{ row }">
        <span v-if="isItemRow(row)" class="project-overview-table__description-cell">
          {{ row.project.description || '--' }}
        </span>
      </template>
    </el-table-column>

    <el-table-column
      v-if="showUpdatedAtColumn"
      :label="t('projectManagement.sortFieldUpdatedAt')"
      width="170"
    >
      <template #default="{ row }">
        <span v-if="isItemRow(row)">{{ resolveDisplayTime(row.project.updatedAt) }}</span>
      </template>
    </el-table-column>

    <el-table-column
      v-if="showActionColumn"
      prop="actions"
      :label="t('projectManagement.actions')"
      width="430"
      fixed="right"
    >
      <template #default="{ row }">
        <div v-if="isItemRow(row)" class="project-overview-table__project-actions" @click.stop>
          <slot name="actions" :project="row.project">
            <el-tooltip
              v-if="showMemberAction"
              :content="t('projectManagement.memberAndPermission')"
              placement="top"
            >
              <el-button
                size="small"
                text
                circle
                class="!w-7 !h-7"
                @click.stop="emit('open-runtime-access', row.project)"
              >
                <el-icon><User /></el-icon>
              </el-button>
            </el-tooltip>

            <el-tooltip
              v-if="showDeployAction"
              :content="t('projectManagement.publishAndDeploy')"
              placement="top"
            >
              <el-button
                size="small"
                text
                circle
                class="!w-7 !h-7"
                @click.stop="emit('deploy', row.project)"
              >
                <el-icon><UploadFilled /></el-icon>
              </el-button>
            </el-tooltip>

            <el-tooltip
              v-if="showExportAction"
              :content="t('projectManagement.exportProject')"
              placement="top"
            >
              <el-button
                size="small"
                text
                circle
                class="!w-7 !h-7"
                @click.stop="emit('export', row.project)"
              >
                <el-icon><Upload /></el-icon>
              </el-button>
            </el-tooltip>

            <el-tooltip
              v-if="showDeleteAction"
              :content="t('projectManagement.delete')"
              placement="top"
            >
              <el-button
                size="small"
                text
                circle
                class="!w-7 !h-7 !text-red-500"
                @click.stop="emit('delete', row.project)"
              >
                <el-icon><Delete /></el-icon>
              </el-button>
            </el-tooltip>
          </slot>
        </div>
        <div
          v-else-if="isEditableGroupRow(row)"
          class="project-overview-table__group-actions project-overview-table__group-actions--fixed"
          @click.stop
        >
          <el-tooltip :content="t('projectManagement.addProjectToGroup')" placement="top">
            <el-button
              size="small"
              text
              circle
              class="project-overview-table__group-action"
              @click="emit('group-add-project', row)"
            >
              <el-icon :size="16"><Plus /></el-icon>
            </el-button>
          </el-tooltip>
          <el-tooltip :content="t('projectManagement.editGroup')" placement="top">
            <el-button
              size="small"
              text
              circle
              class="project-overview-table__group-action"
              @click="emit('group-edit', row)"
            >
              <el-icon :size="16"><Edit /></el-icon>
            </el-button>
          </el-tooltip>
          <el-tooltip :content="t('projectManagement.deleteGroup')" placement="top">
            <el-button
              size="small"
              text
              circle
              class="project-overview-table__group-action project-overview-table__group-action--danger"
              @click="emit('group-delete', row)"
            >
              <el-icon :size="16"><Delete /></el-icon>
            </el-button>
          </el-tooltip>
        </div>
      </template>
    </el-table-column>
  </el-table>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import dayjs from 'dayjs'
import {
  Delete,
  Edit,
  FolderOpened,
  Plus,
  Upload,
  UploadFilled,
  User,
} from '@element-plus/icons-vue'
import type { TableColumnCtx, TagProps } from 'element-plus'
import { useI18n } from 'vue-i18n'
import type { ProjectGroupCardViewModel, ProjectOverviewItem } from './project-overview.types'

type ProjectGroupRow = {
  id: string
  rowType: 'group'
  groupId: string
  groupName: string
  projectCount: number
}

type ProjectItemRow = {
  id: string
  rowType: 'item'
  groupId: string
  groupName: string
  project: ProjectOverviewItem
}

type ProjectOverviewTableRow = ProjectGroupRow | ProjectItemRow

const props = withDefaults(
  defineProps<{
    projects?: ProjectOverviewItem[]
    groupCards?: ProjectGroupCardViewModel[]
    loading?: boolean
    grouped?: boolean
    showGroupedProjectItems?: boolean
    emptyDescription?: string
    selectedIds?: string[]
    showSelection?: boolean
    showDescriptionColumn?: boolean
    showTagColumn?: boolean
    showRuntimeModeColumn?: boolean
    showRuntimeStatusColumn?: boolean
    showCreatorColumn?: boolean
    showUpdatedAtColumn?: boolean
    showActionColumn?: boolean
    showMemberAction?: boolean
    showDeployAction?: boolean
    showExportAction?: boolean
    showDeleteAction?: boolean
  }>(),
  {
    projects: () => [],
    groupCards: () => [],
    loading: false,
    grouped: true,
    showGroupedProjectItems: true,
    emptyDescription: '',
    selectedIds: () => [],
    showSelection: false,
    showDescriptionColumn: true,
    showTagColumn: true,
    showRuntimeModeColumn: true,
    showRuntimeStatusColumn: true,
    showCreatorColumn: true,
    showUpdatedAtColumn: true,
    showActionColumn: true,
    showMemberAction: true,
    showDeployAction: true,
    showExportAction: true,
    showDeleteAction: true,
  },
)

const emit = defineEmits<{
  (event: 'open-project', payload: ProjectOverviewItem): void
  (event: 'selection-change', payload: { projectId: string; selected: boolean }): void
  (event: 'open-runtime-access', payload: ProjectOverviewItem): void
  (event: 'deploy', payload: ProjectOverviewItem): void
  (event: 'export', payload: ProjectOverviewItem): void
  (event: 'delete', payload: ProjectOverviewItem): void
  (event: 'group-select', payload: { groupId: string; groupName: string }): void
  (event: 'group-add-project', payload: ProjectGroupRow): void
  (event: 'group-edit', payload: ProjectGroupRow): void
  (event: 'group-delete', payload: ProjectGroupRow): void
}>()

const UNGROUPED_ID = '__ungrouped__'
const { t } = useI18n()
const selectedIdSet = computed(() => new Set(props.selectedIds))
const resolvedEmptyDescription = computed(
  () => props.emptyDescription || t('projectManagement.emptyProjects'),
)

const tableRows = computed<ProjectOverviewTableRow[]>(() => {
  if (!props.grouped) {
    return props.projects.map((project) => {
      const groupId = project.group?.id || UNGROUPED_ID
      const groupName = project.group?.name || t('projectManagement.unknownGroup')
      return {
        id: `item-${project.id}`,
        rowType: 'item',
        groupId,
        groupName,
        project,
      }
    })
  }

  if (!props.showGroupedProjectItems) {
    const result: ProjectOverviewTableRow[] = []
    const knownGroupIds = new Set<string>()
    const fallbackGroups = new Map<
      string,
      { groupName: string; count: number; explicitCount?: number }
    >()
    const ungroupedProjects: ProjectOverviewItem[] = []

    props.groupCards.forEach((group) => {
      const groupId = String(group.id)
      knownGroupIds.add(groupId)
      result.push({
        id: `group-${groupId}`,
        rowType: 'group',
        groupId,
        groupName: group.name,
        projectCount: group.projectCount,
      })
    })

    for (const project of props.projects) {
      const groupId = project.group?.id
      if (groupId) {
        if (!knownGroupIds.has(groupId)) {
          const existing = fallbackGroups.get(groupId)
          fallbackGroups.set(groupId, {
            groupName: project.group?.name || t('projectManagement.unknownGroup'),
            count: (existing?.count || 0) + 1,
            explicitCount: project.group?.projectCount ?? existing?.explicitCount,
          })
        }
        continue
      }

      ungroupedProjects.push(project)
    }

    fallbackGroups.forEach((group, groupId) => {
      result.push({
        id: `group-${groupId}`,
        rowType: 'group',
        groupId,
        groupName: group.groupName,
        projectCount: group.explicitCount ?? group.count,
      })
    })

    ungroupedProjects.forEach((project) => {
      result.push({
        id: `item-${project.id}`,
        rowType: 'item',
        groupId: UNGROUPED_ID,
        groupName: t('projectManagement.unknownGroup'),
        project,
      })
    })

    return result
  }

  const groups = new Map<string, { groupName: string; projects: ProjectOverviewItem[] }>()
  for (const project of props.projects) {
    const groupId = project.group?.id || UNGROUPED_ID
    const groupName = project.group?.name || t('projectManagement.unknownGroup')
    const existing = groups.get(groupId)
    if (existing) {
      existing.projects.push(project)
      continue
    }
    groups.set(groupId, {
      groupName,
      projects: [project],
    })
  }

  const result: ProjectOverviewTableRow[] = []
  groups.forEach((value, groupId) => {
    result.push({
      id: `group-${groupId}`,
      rowType: 'group',
      groupId,
      groupName: value.groupName,
      projectCount: value.projects.length,
    })
    value.projects.forEach((project) => {
      result.push({
        id: `item-${project.id}`,
        rowType: 'item',
        groupId,
        groupName: value.groupName,
        project,
      })
    })
  })

  return result
})

const isGroupRow = (row: ProjectOverviewTableRow): row is ProjectGroupRow => row.rowType === 'group'

const isItemRow = (row: ProjectOverviewTableRow): row is ProjectItemRow => row.rowType === 'item'

const isEditableGroupRow = (row: ProjectOverviewTableRow): row is ProjectGroupRow =>
  isGroupRow(row) && row.groupId !== UNGROUPED_ID

const resolveRuntimeTag = (status: string): TagProps['type'] => {
  const normalized = status.trim().toLowerCase()
  if (normalized === 'running') {
    return 'success'
  }
  if (normalized === 'deploying' || normalized === 'pending') {
    return 'warning'
  }
  if (normalized === 'error' || normalized === 'rollback') {
    return 'danger'
  }
  return 'info'
}

const resolveRuntimeText = (status: string) => {
  const normalized = status.trim().toLowerCase()
  if (normalized === 'running') {
    return t('projectManagement.deployStatusRunning')
  }
  if (normalized === 'deploying') {
    return t('projectManagement.deployStatusDeploying')
  }
  if (normalized === 'pending') {
    return t('projectManagement.deployStatusPending')
  }
  if (normalized === 'stopped') {
    return t('projectManagement.deployStatusStopped')
  }
  if (normalized === 'error') {
    return t('projectManagement.deployStatusError')
  }
  if (normalized === 'rollback') {
    return t('projectManagement.deployStatusRollback')
  }
  if (normalized === 'not_deployed') {
    return t('projectManagement.notDeployed')
  }
  return t('projectManagement.deployStatusUnknown')
}

const normalizeRuntimeMode = (value: unknown): 'DEV' | 'RELEASE' | '' => {
  const normalized = String(value || '')
    .trim()
    .toUpperCase()
  if (normalized === 'DEV') {
    return 'DEV'
  }
  if (normalized === 'RELEASE') {
    return 'RELEASE'
  }
  return ''
}

const resolveRuntimeModeText = (project: ProjectOverviewItem) => {
  const mode = normalizeRuntimeMode(project.runtimeSummary.runtimeMode)
  if (mode === 'DEV') {
    return t('projectManagement.modeDisplayDev')
  }
  if (mode === 'RELEASE') {
    return t('projectManagement.modeDisplayRelease')
  }
  return '--'
}

const resolveDisplayTime = (value: string | null | undefined) => {
  if (!value) {
    return '--'
  }
  const parsed = dayjs(value)
  return parsed.isValid() ? parsed.format('YYYY-MM-DD HH:mm:ss') : '--'
}

const handleSpanMethod = ({
  row,
}: {
  row: ProjectOverviewTableRow
  column: TableColumnCtx<ProjectOverviewTableRow>
  rowIndex: number
  columnIndex: number
}) => {
  if (!isGroupRow(row)) {
    return [1, 1]
  }
  return [1, 1]
}

const resolveRowClassName = ({ row }: { row: ProjectOverviewTableRow }) =>
  isGroupRow(row) ? 'project-overview-table__group-row' : ''

const handleRowClick = (row: ProjectOverviewTableRow) => {
  if (isItemRow(row)) {
    emit('open-project', row.project)
    return
  }
  if (isEditableGroupRow(row)) {
    emit('group-select', {
      groupId: row.groupId,
      groupName: row.groupName,
    })
  }
}

const isSelected = (projectId: string) => selectedIdSet.value.has(projectId)

const handleSelectionChange = (projectId: string, value: string | number | boolean) => {
  emit('selection-change', {
    projectId,
    selected: Boolean(value),
  })
}
</script>

<style scoped>
:deep(.project-overview-table__group-row td) {
  background-color: color-mix(in srgb, var(--el-fill-color-light) 82%, #ffffff);
  padding: 10px 0;
  cursor: pointer;
}

:deep(.project-overview-table__group-row:hover > td) {
  background-color: color-mix(in srgb, var(--el-fill-color-light) 78%, #ffffff);
}

:deep(.el-table__row td) {
  padding: 18px 0;
}

:deep(.el-table__header th) {
  background: #f4f6f9;
  color: #475569;
  font-weight: 700;
}

.project-overview-table__group-header {
  display: flex;
  width: 100%;
  min-width: 0;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}

.project-overview-table__group-main {
  display: inline-flex;
  min-width: 0;
  align-items: center;
  gap: 10px;
}

.project-overview-table__group-icon {
  display: inline-flex;
  width: 24px;
  height: 24px;
  flex: 0 0 auto;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 6px;
  background: var(--el-bg-color);
  color: var(--el-color-primary);
}

.project-overview-table__group-copy {
  display: flex;
  min-width: 0;
  align-items: baseline;
  gap: 8px;
}

.project-overview-table__group-name {
  min-width: 0;
  overflow: hidden;
  color: var(--el-text-color-primary);
  font-size: 13px;
  font-weight: 700;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.project-overview-table__group-count {
  flex: 0 0 auto;
  color: var(--el-text-color-secondary);
  font-size: 12px;
  font-weight: 500;
}

.project-overview-table__group-actions {
  display: inline-flex;
  flex: 0 0 auto;
  align-items: center;
  gap: 2px;
}

.project-overview-table__project-actions {
  display: inline-flex;
  width: 100%;
  align-items: center;
  justify-content: flex-start;
  flex-wrap: nowrap;
  gap: 2px;
}

.project-overview-table__description-cell {
  display: block;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.project-overview-table__group-actions--fixed {
  width: 100%;
  justify-content: flex-end;
}

.project-overview-table__group-action {
  width: 28px !important;
  height: 28px !important;
  min-width: 28px !important;
  margin: 0 !important;
  padding: 0 !important;
}

.project-overview-table__group-action--danger {
  color: var(--el-color-danger) !important;
}
</style>
