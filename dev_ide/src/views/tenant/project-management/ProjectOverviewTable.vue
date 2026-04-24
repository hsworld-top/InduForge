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

    <el-table-column :label="t('projectManagement.projectName')" min-width="220">
      <template #default="{ row }">
        <div
          v-if="isGroupRow(row)"
          class="flex items-center gap-2 text-xs font-semibold text-gray-700 dark:text-gray-200"
        >
          <el-icon><FolderOpened /></el-icon>
          <span>{{ row.groupName }}</span>
          <el-tag size="small" effect="plain" type="info">
            {{ t('projectManagement.projectCountUnit', { count: row.projectCount }) }}
          </el-tag>
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

    <el-table-column v-if="showDescriptionColumn" :label="t('projectManagement.description')" min-width="220" show-overflow-tooltip>
      <template #default="{ row }">
        <span v-if="isItemRow(row)">{{ row.project.description || '--' }}</span>
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

    <el-table-column v-if="showRuntimeStatusColumn" :label="t('projectManagement.deployStatus')" width="120" align="center">
      <template #default="{ row }">
        <el-tag v-if="isItemRow(row)" size="small" :type="resolveRuntimeTag(row.project.runtimeSummary.runtimeStatus)">
          {{ resolveRuntimeText(row.project.runtimeSummary.runtimeStatus) }}
        </el-tag>
      </template>
    </el-table-column>

    <el-table-column v-if="showCreatorColumn" :label="t('projectManagement.createdBy')" width="140" show-overflow-tooltip>
      <template #default="{ row }">
        <span v-if="isItemRow(row)">{{ row.project.createdByName || row.project.createdBy || '--' }}</span>
      </template>
    </el-table-column>

    <el-table-column v-if="showUpdatedAtColumn" :label="t('projectManagement.sortFieldUpdatedAt')" width="170">
      <template #default="{ row }">
        <span v-if="isItemRow(row)">{{ resolveDisplayTime(row.project.updatedAt) }}</span>
      </template>
    </el-table-column>

    <el-table-column v-if="showActionColumn" :label="t('projectManagement.actions')" width="240" fixed="right">
      <template #default="{ row }">
        <div v-if="isItemRow(row)" class="flex items-center justify-end gap-1">
          <slot name="actions" :project="row.project">
            <el-tooltip v-if="showMemberAction" :content="t('projectManagement.memberAndPermission')" placement="top">
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

            <el-tooltip v-if="showDeployAction" :content="t('projectManagement.publishAndDeploy')" placement="top">
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

            <el-tooltip v-if="showExportAction" :content="t('projectManagement.exportProject')" placement="top">
              <el-button
                size="small"
                text
                circle
                class="!w-7 !h-7"
                @click.stop="emit('export', row.project)"
              >
                <el-icon><Download /></el-icon>
              </el-button>
            </el-tooltip>

            <el-tooltip v-if="showDeleteAction" :content="t('projectManagement.delete')" placement="top">
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
      </template>
    </el-table-column>
  </el-table>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import dayjs from 'dayjs'
import { Delete, Download, FolderOpened, UploadFilled, User } from '@element-plus/icons-vue'
import type { TableColumnCtx, TagProps } from 'element-plus'
import { useI18n } from 'vue-i18n'
import type { ProjectOverviewItem } from './project-overview.types'

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
    loading?: boolean
    grouped?: boolean
    emptyDescription?: string
    selectedIds?: string[]
    showSelection?: boolean
    showDescriptionColumn?: boolean
    showTagColumn?: boolean
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
    loading: false,
    grouped: true,
    emptyDescription: '',
    selectedIds: () => [],
    showSelection: false,
    showDescriptionColumn: true,
    showTagColumn: true,
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
}>()

const UNGROUPED_ID = '__ungrouped__'
const { t } = useI18n()
const selectedIdSet = computed(() => new Set(props.selectedIds))
const resolvedEmptyDescription = computed(
  () => props.emptyDescription || t('projectManagement.emptyProjects'),
)
const tableColumnCount = computed(() => {
  let count = 1
  if (props.showDescriptionColumn) {
    count += 1
  }
  if (props.showTagColumn) {
    count += 1
  }
  if (props.showRuntimeStatusColumn) {
    count += 1
  }
  if (props.showCreatorColumn) {
    count += 1
  }
  if (props.showUpdatedAtColumn) {
    count += 1
  }
  if (props.showActionColumn) {
    count += 1
  }
  if (props.showSelection) {
    count += 1
  }
  return count
})
const groupStartColumnIndex = computed(() => (props.showSelection ? 1 : 0))

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
  return t('projectManagement.deployStatusUnknown')
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
  columnIndex,
}: {
  row: ProjectOverviewTableRow
  column: TableColumnCtx<ProjectOverviewTableRow>
  rowIndex: number
  columnIndex: number
}) => {
  if (!isGroupRow(row)) {
    return [1, 1]
  }
  if (columnIndex < groupStartColumnIndex.value) {
    return [0, 0]
  }
  if (columnIndex === groupStartColumnIndex.value) {
    return [1, tableColumnCount.value - groupStartColumnIndex.value]
  }
  return [0, 0]
}

const resolveRowClassName = ({ row }: { row: ProjectOverviewTableRow }) =>
  isGroupRow(row) ? 'project-overview-table__group-row' : ''

const handleRowClick = (row: ProjectOverviewTableRow) => {
  if (isItemRow(row)) {
    emit('open-project', row.project)
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
  background-color: var(--el-fill-color-light);
}

:deep(.project-overview-table__group-row:hover > td) {
  background-color: var(--el-fill-color-light);
}

:deep(.el-table__row td) {
  padding: 18px 0;
}

:deep(.el-table__header th) {
  background: #f4f6f9;
  color: #475569;
  font-weight: 700;
}
</style>
