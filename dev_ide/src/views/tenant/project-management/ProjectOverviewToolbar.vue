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
      <slot name="actions" />
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { Grid, Search } from '@element-plus/icons-vue'
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
  },
)

const emit = defineEmits<{
  (event: 'update:search', value: string): void
  (event: 'update:viewMode', value: ProjectOverviewViewMode): void
  (event: 'update:sortBy', value: ProjectOverviewSortField): void
  (event: 'update:sortOrder', value: 'ASC' | 'DESC'): void
  (event: 'update:compositeFilters', value: ProjectOverviewCompositeFilters): void
  (event: 'update:tagIds', value: string[]): void
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
