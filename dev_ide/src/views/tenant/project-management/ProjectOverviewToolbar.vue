<template>
  <section
    class="project-overview-toolbar rounded-xl border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 p-3 flex items-center justify-between gap-3"
  >
    <div class="project-overview-toolbar__left flex items-center flex-wrap gap-2" data-testid="overview-toolbar-left">
      <el-input
        v-model="searchModel"
        size="small"
        clearable
        class="!w-64"
        data-testid="overview-search-input"
        :placeholder="resolvedSearchPlaceholder"
        :prefix-icon="Search"
      />

      <div
        class="flex items-center rounded-lg border border-gray-200 dark:border-gray-700 p-0.5 bg-gray-50 dark:bg-gray-900"
      >
        <el-tooltip :content="t('projectManagement.cardView')" placement="top">
          <el-button
            size="small"
            text
            class="!w-8 !h-7 !px-0"
            data-testid="overview-view-card"
            :class="viewModeModel === 'card' ? '!bg-white dark:!bg-gray-700 !text-blue-600' : ''"
            @click="viewModeModel = 'card'"
          >
            <el-icon><Grid /></el-icon>
          </el-button>
        </el-tooltip>

        <el-tooltip :content="t('projectManagement.listView')" placement="top">
          <el-button
            size="small"
            text
            class="!w-8 !h-7 !px-0"
            data-testid="overview-view-list"
            :class="viewModeModel === 'list' ? '!bg-white dark:!bg-gray-700 !text-blue-600' : ''"
            @click="viewModeModel = 'list'"
          >
            <svg
              class="w-4 h-4"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
              stroke-linecap="round"
              stroke-linejoin="round"
            >
              <line x1="8" y1="6" x2="21" y2="6"></line>
              <line x1="8" y1="12" x2="21" y2="12"></line>
              <line x1="8" y1="18" x2="21" y2="18"></line>
              <line x1="3" y1="6" x2="3.01" y2="6"></line>
              <line x1="3" y1="12" x2="3.01" y2="12"></line>
              <line x1="3" y1="18" x2="3.01" y2="18"></line>
            </svg>
          </el-button>
        </el-tooltip>
      </div>

      <ProjectFilterDropdown
        v-model="compositeModel"
        :runtime-mode-options="runtimeModeOptions"
        :deploy-status-options="deployStatusOptions"
      />

      <ProjectTagFilterPopover v-model="tagIdsModel" :options="tagOptions" />

      <el-select
        v-model="sortByModel"
        size="small"
        class="!w-36"
        data-testid="overview-sort-field"
      >
        <el-option
          v-for="option in resolvedSortFieldOptions"
          :key="option.value"
          :label="option.label"
          :value="option.value"
        />
      </el-select>

      <el-select
        v-model="sortOrderModel"
        size="small"
        class="!w-28"
        data-testid="overview-sort-order"
      >
        <el-option
          v-for="option in resolvedSortOrderOptions"
          :key="option.value"
          :label="option.label"
          :value="option.value"
        />
      </el-select>
    </div>

    <div class="project-overview-toolbar__right flex items-center gap-2">
      <template v-if="selectedCount > 0">
        <el-tooltip v-if="canExportProjects" :content="t('projectManagement.batchExport')" placement="top">
          <button
            type="button"
            data-testid="project-batch-export-trigger"
            :aria-label="t('projectManagement.batchExport')"
            class="project-overview-toolbar__icon-button text-emerald-600 dark:text-emerald-300"
            @click="emit('batch-export')"
          >
            <el-icon><Download /></el-icon>
          </button>
        </el-tooltip>
        <el-tooltip v-if="canDeleteProjects" :content="t('projectManagement.batchDelete')" placement="top">
          <button
            type="button"
            data-testid="project-batch-delete-trigger"
            :aria-label="t('projectManagement.batchDelete')"
            class="project-overview-toolbar__icon-button text-red-500 dark:text-red-300"
            @click="emit('batch-delete')"
          >
            <el-icon><Delete /></el-icon>
          </button>
        </el-tooltip>
      </template>

      <el-tooltip v-if="canManageProjects" :content="t('projectManagement.addProject')" placement="top">
        <button
          type="button"
          data-testid="project-add-trigger"
          :aria-label="t('projectManagement.addProject')"
          class="project-overview-toolbar__primary-button"
          @click="emit('create-project')"
        >
          <el-icon><Plus /></el-icon>
        </button>
      </el-tooltip>
      <el-tooltip :content="t('common.refresh')" placement="top">
        <button
          type="button"
          data-testid="project-refresh-trigger"
          :aria-label="t('common.refresh')"
          class="project-overview-toolbar__icon-button"
          @click="emit('refresh')"
        >
          <el-icon><RefreshRight /></el-icon>
        </button>
      </el-tooltip>
      <el-tooltip v-if="canManageProjects" :content="t('projectManagement.groupManagement')" placement="top">
        <button
          type="button"
          data-testid="project-group-manage-trigger"
          :aria-label="t('projectManagement.groupManagement')"
          class="project-overview-toolbar__icon-button"
          @click="emit('open-group-manager')"
        >
          <el-icon><FolderOpened /></el-icon>
        </button>
      </el-tooltip>
      <el-tooltip v-if="canManageProjects" :content="t('projectManagement.importProject')" placement="top">
        <button
          type="button"
          data-testid="project-import-trigger"
          :aria-label="t('projectManagement.importProject')"
          class="project-overview-toolbar__icon-button"
          @click="emit('import-project')"
        >
          <el-icon><Upload /></el-icon>
        </button>
      </el-tooltip>
      <el-tooltip :content="t('projectManagement.settings')" placement="top">
        <button
          type="button"
          data-testid="project-settings-trigger"
          :aria-label="t('projectManagement.settings')"
          class="project-overview-toolbar__icon-button"
          @click="emit('open-settings')"
        >
          <el-icon><Setting /></el-icon>
        </button>
      </el-tooltip>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import {
  Delete,
  Download,
  FolderOpened,
  Grid,
  Plus,
  RefreshRight,
  Search,
  Setting,
  Upload,
} from '@element-plus/icons-vue'
import { useI18n } from 'vue-i18n'
import type {
  ProjectDeployStatus,
  ProjectOverviewSortField,
  ProjectOverviewSortOrder,
  ProjectRuntimeMode,
} from '@/api/project.api'
import type {
  ProjectOverviewCompositeFilters,
  ProjectOverviewTag,
  ProjectOverviewViewMode,
} from './project-overview.types'
import ProjectFilterDropdown from './ProjectFilterDropdown.vue'
import ProjectTagFilterPopover from './ProjectTagFilterPopover.vue'

type RuntimeModeOption = {
  label: string
  value: ProjectRuntimeMode
}

type DeployStatusOption = {
  label: string
  value: ProjectDeployStatus
}

type SortFieldOption = {
  label: string
  value: ProjectOverviewSortField
}

type SortOrderOption = {
  label: string
  value: 'ASC' | 'DESC'
}

const props = withDefaults(
  defineProps<{
    search?: string
    searchPlaceholder?: string
    viewMode?: ProjectOverviewViewMode
    sortBy?: ProjectOverviewSortField
    sortOrder?: ProjectOverviewSortOrder
    compositeFilters?: ProjectOverviewCompositeFilters
    tagIds?: string[]
    tagOptions?: ProjectOverviewTag[]
    runtimeModeOptions?: RuntimeModeOption[]
    deployStatusOptions?: DeployStatusOption[]
    sortFieldOptions?: SortFieldOption[]
    sortOrderOptions?: SortOrderOption[]
    selectedCount?: number
    canManageProjects?: boolean
    canExportProjects?: boolean
    canDeleteProjects?: boolean
  }>(),
  {
    search: '',
    searchPlaceholder: '',
    viewMode: 'card',
    sortBy: 'createdAt',
    sortOrder: 'DESC',
    compositeFilters: () => ({
      runtimeModes: [],
      deployStatuses: [],
      createdBy: '',
    }),
    tagIds: () => [],
    tagOptions: () => [],
    runtimeModeOptions: () => [],
    deployStatusOptions: () => [],
    sortFieldOptions: () => [],
    sortOrderOptions: () => [],
    selectedCount: 0,
    canManageProjects: true,
    canExportProjects: true,
    canDeleteProjects: true,
  },
)

const emit = defineEmits<{
  (event: 'update:search', value: string): void
  (event: 'update:viewMode', value: ProjectOverviewViewMode): void
  (event: 'update:sortBy', value: ProjectOverviewSortField): void
  (event: 'update:sortOrder', value: 'ASC' | 'DESC'): void
  (event: 'update:compositeFilters', value: ProjectOverviewCompositeFilters): void
  (event: 'update:tagIds', value: string[]): void
  (event: 'create-project'): void
  (event: 'refresh'): void
  (event: 'open-group-manager'): void
  (event: 'import-project'): void
  (event: 'batch-export'): void
  (event: 'batch-delete'): void
  (event: 'open-settings'): void
}>()

const { t } = useI18n()

const resolvedSearchPlaceholder = computed(
  () => props.searchPlaceholder || t('projectManagement.searchPlaceholder'),
)

const resolvedSortFieldOptions = computed<SortFieldOption[]>(() =>
  props.sortFieldOptions.length > 0
    ? props.sortFieldOptions
    : [
        { label: t('projectManagement.sortFieldCreatedAt'), value: 'createdAt' },
        { label: t('projectManagement.sortFieldUpdatedAt'), value: 'updatedAt' },
        { label: t('projectManagement.sortFieldLastDeployedAt'), value: 'lastDeployedAt' },
        { label: t('projectManagement.sortFieldRuntimeStatus'), value: 'runtimeStatus' },
      ],
)

const resolvedSortOrderOptions = computed<SortOrderOption[]>(() =>
  props.sortOrderOptions.length > 0
    ? props.sortOrderOptions
    : [
        { label: t('projectManagement.sortOrderDesc'), value: 'DESC' },
        { label: t('projectManagement.sortOrderAsc'), value: 'ASC' },
      ],
)

const searchModel = computed({
  get: () => props.search,
  set: (value: string) => emit('update:search', value),
})

const viewModeModel = computed<ProjectOverviewViewMode>({
  get: () => (props.viewMode === 'list' ? 'list' : 'card'),
  set: (value) => emit('update:viewMode', value),
})

const sortByModel = computed<ProjectOverviewSortField>({
  get: () => props.sortBy,
  set: (value) => emit('update:sortBy', value),
})

const sortOrderModel = computed<'ASC' | 'DESC'>({
  get: () => (String(props.sortOrder).toUpperCase() === 'ASC' ? 'ASC' : 'DESC'),
  set: (value) => emit('update:sortOrder', value),
})

const compositeModel = computed<ProjectOverviewCompositeFilters>({
  get: () => props.compositeFilters,
  set: (value) => emit('update:compositeFilters', value),
})

const tagIdsModel = computed<string[]>({
  get: () => props.tagIds,
  set: (value) => emit('update:tagIds', value),
})
</script>

<style scoped>
.project-overview-toolbar__primary-button,
.project-overview-toolbar__icon-button {
  width: 36px;
  height: 36px;
  border-radius: 9999px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  transition:
    color 0.2s ease,
    background-color 0.2s ease,
    border-color 0.2s ease;
}

.project-overview-toolbar__primary-button {
  border: 1px solid #2563eb;
  background: #3b82f6;
  color: #ffffff;
  box-shadow: 0 1px 2px rgb(15 23 42 / 0.08);
}

.project-overview-toolbar__primary-button:hover {
  background: #2563eb;
}

.project-overview-toolbar__icon-button {
  border: 1px solid #e5e7eb;
  background: #f9fafb;
  color: #4b5563;
  box-shadow: 0 1px 2px rgb(15 23 42 / 0.04);
}

.project-overview-toolbar__icon-button:hover {
  background: #ffffff;
  color: #2563eb;
}

html.dark .project-overview-toolbar__icon-button,
[data-theme='dark'] .project-overview-toolbar__icon-button {
  border-color: #4b5563;
  background: #374151;
  color: #d1d5db;
}

html.dark .project-overview-toolbar__icon-button:hover,
[data-theme='dark'] .project-overview-toolbar__icon-button:hover {
  background: #4b5563;
  color: #93c5fd;
}
</style>
