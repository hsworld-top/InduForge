<template>
  <section
    class="project-overview-toolbar"
  >
    <div class="project-overview-toolbar__left" data-testid="overview-toolbar-left">
      <el-input
        v-model="searchModel"
        size="small"
        clearable
        class="!w-44"
        data-testid="overview-search-input"
        :placeholder="resolvedSearchPlaceholder"
        :prefix-icon="Search"
      />

      <div class="project-overview-toolbar__view-switcher">
        <el-tooltip :content="t('projectManagement.cardView')" placement="top">
          <button
            type="button"
            class="project-overview-toolbar__view-btn"
            data-testid="overview-view-card"
            :class="{ 'is-active': viewModeModel === 'card' }"
            @click="viewModeModel = 'card'"
          >
            <el-icon><Grid /></el-icon>
          </button>
        </el-tooltip>

        <el-tooltip :content="t('projectManagement.listView')" placement="top">
          <button
            type="button"
            class="project-overview-toolbar__view-btn"
            data-testid="overview-view-list"
            :class="{ 'is-active': viewModeModel === 'list' }"
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
          </button>
        </el-tooltip>
      </div>

      <ProjectFilterDropdown
        v-model="compositeModel"
        :runtime-mode-options="runtimeModeOptions"
        :deploy-status-options="deployStatusOptions"
      />

      <ProjectTagFilterPopover
        v-model="tagIdsModel"
        :options="tagOptions"
        :deletable="canManageProjects"
        :max-selected="tagMaxSelected"
        @delete-tag="emit('delete-tag', $event)"
        @limit="emit('tag-limit', $event)"
      />

      <!-- 排序字段：pill 按钮 + popover 菜单 -->
      <el-popover
        placement="bottom-start"
        :width="160"
        trigger="click"
      >
        <template #reference>
          <button
            type="button"
            class="project-overview-toolbar__pill-btn"
            data-testid="overview-sort-field"
          >
            <span>{{ currentSortFieldLabel }}</span>
          </button>
        </template>
        <div class="sort-popover-menu">
          <button
            v-for="option in resolvedSortFieldOptions"
            :key="option.value"
            type="button"
            class="sort-popover-item"
            :class="{ 'is-active': sortByModel === option.value }"
            @click="sortByModel = option.value"
          >
            {{ option.label }}
          </button>
        </div>
      </el-popover>

      <!-- 排序方向：toggle pill 按钮 -->
      <button
        type="button"
        class="project-overview-toolbar__pill-btn"
        data-testid="overview-sort-order"
        @click="toggleSortOrder"
      >
        <el-icon class="project-overview-toolbar__sort-icon">
          <SortDown v-if="sortOrderModel === 'DESC'" />
          <SortUp v-else />
        </el-icon>
        <span>{{ currentSortOrderLabel }}</span>
      </button>
    </div>

    <div class="project-overview-toolbar__right">
      <template v-if="selectedCount > 0">
        <el-tooltip v-if="canExportProjects" :content="t('projectManagement.batchExport')" placement="top">
          <button
            type="button"
            data-testid="project-batch-export-trigger"
            :aria-label="t('projectManagement.batchExport')"
            class="project-overview-toolbar__icon-button text-emerald-600 dark:text-emerald-300"
            @click="emit('batch-export')"
          >
            <el-icon><Upload /></el-icon>
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
          <el-icon><Download /></el-icon>
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
  SortDown,
  SortUp,
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
    tagMaxSelected?: number
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
      visibility: [],
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
    tagMaxSelected: 10,
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
  (event: 'delete-tag', value: ProjectOverviewTag): void
  (event: 'tag-limit', value: number): void
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

/** 当前排序字段的显示文本 */
const currentSortFieldLabel = computed(() => {
  const found = resolvedSortFieldOptions.value.find(
    (opt) => opt.value === sortByModel.value,
  )
  return found ? found.label : sortByModel.value
})

/** 当前排序方向的显示文本 */
const currentSortOrderLabel = computed(() => {
  const found = resolvedSortOrderOptions.value.find(
    (opt) => opt.value === sortOrderModel.value,
  )
  return found ? found.label : sortOrderModel.value
})

/** 切换排序方向 */
const toggleSortOrder = () => {
  sortOrderModel.value = sortOrderModel.value === 'DESC' ? 'ASC' : 'DESC'
}
</script>

<style scoped>
/* ─── 工具栏：毛玻璃浮动面板 ─── */
.project-overview-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 12px;
  padding: 14px 16px;
  background: var(--ck-bg-card);
  border-radius: var(--ck-radius-lg);
  border: 1px solid var(--ck-border);
  box-shadow: var(--ck-shadow-sm);
  backdrop-filter: blur(10px);
  -webkit-backdrop-filter: blur(10px);
  animation: ck-fadeUp 0.55s ease both;
  position: relative;
  z-index: 5;
}

.project-overview-toolbar__left {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 12px;
  flex: 1;
}

.project-overview-toolbar__right {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-left: auto;
  flex-wrap: wrap;
}

/* ─── 搜索框覆写 ─── */
.project-overview-toolbar :deep(.el-input) {
  --el-input-bg-color: var(--ck-bg-tertiary);
  --el-input-border-color: var(--ck-border);
  --el-input-hover-border-color: var(--ck-primary);
  --el-input-focus-border-color: var(--ck-primary);
}

.project-overview-toolbar :deep(.el-input__wrapper) {
  border-radius: var(--ck-radius-md);
  padding: 6px 12px;
  box-shadow: none;
  border: 1px solid var(--ck-border);
  background: var(--ck-bg-tertiary);
  transition: all 0.2s;
}

.project-overview-toolbar :deep(.el-input__wrapper:focus-within) {
  border-color: var(--ck-primary);
  box-shadow: 0 0 0 3px var(--ck-primary-light);
}

/* ─── 视图切换 pill 容器 ─── */
.project-overview-toolbar__view-switcher {
  display: flex;
  background: rgba(0, 0, 0, 0.04);
  border-radius: 10px;
  padding: 3px;
  border: none;
}

html.dark .project-overview-toolbar__view-switcher,
[data-theme='dark'] .project-overview-toolbar__view-switcher {
  background: rgba(255, 255, 255, 0.06);
}

.project-overview-toolbar__view-btn {
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: transparent;
  border: none;
  border-radius: 8px;
  color: var(--ck-text-muted);
  cursor: pointer;
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
}

.project-overview-toolbar__view-btn:hover {
  color: var(--ck-text-primary);
  background: rgba(255, 255, 255, 0.5);
}

.project-overview-toolbar__view-btn.is-active {
  background: #ffffff;
  color: var(--ck-primary);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.08);
}

html.dark .project-overview-toolbar__view-btn:hover,
[data-theme='dark'] .project-overview-toolbar__view-btn:hover {
  background: rgba(255, 255, 255, 0.08);
}

html.dark .project-overview-toolbar__view-btn.is-active,
[data-theme='dark'] .project-overview-toolbar__view-btn.is-active {
  background: #334155;
  color: #fff;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.2);
}

/* ─── 右侧按钮 ─── */
.project-overview-toolbar__primary-button {
  width: 32px;
  height: 32px;
  border-radius: var(--ck-radius-md);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  background: var(--ck-gradient-primary);
  color: white;
  border: 1px solid rgba(255, 255, 255, 0.35);
  box-shadow: 0 12px 20px rgba(29, 78, 216, 0.24);
  cursor: pointer;
  transition: all 0.2s;
}

.project-overview-toolbar__primary-button:hover {
  transform: translateY(-1px);
  box-shadow: 0 16px 28px rgba(29, 78, 216, 0.32);
}

.project-overview-toolbar__icon-button {
  width: 32px;
  height: 32px;
  border-radius: var(--ck-radius-md);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  background: var(--ck-bg-secondary);
  color: var(--ck-text-secondary);
  border: 1px solid var(--ck-border);
  box-shadow: var(--ck-shadow-sm);
  cursor: pointer;
  transition: all 0.2s;
}

.project-overview-toolbar__icon-button:hover {
  background: var(--ck-bg-tertiary);
  color: var(--ck-text-primary);
}

html.dark .project-overview-toolbar__icon-button,
[data-theme='dark'] .project-overview-toolbar__icon-button {
  background: var(--ck-bg-secondary);
  color: var(--ck-text-secondary);
  border-color: var(--ck-border);
}

html.dark .project-overview-toolbar__icon-button:hover,
[data-theme='dark'] .project-overview-toolbar__icon-button:hover {
  background: var(--ck-bg-hover);
  color: var(--ck-text-primary);
}

/* ─── 下拉选择框覆写（创建时间 / 升降序） ─── */
.project-overview-toolbar :deep(.el-select .el-input__wrapper) {
  border-radius: 10px;
  box-shadow: none;
  border: none;
  background: rgba(0, 0, 0, 0.04);
  padding: 4px 10px;
  font-size: 13px;
  height: 32px;
}

.project-overview-toolbar :deep(.el-select .el-input__wrapper:hover) {
  background: rgba(0, 0, 0, 0.08);
}

html.dark .project-overview-toolbar :deep(.el-select .el-input__wrapper),
[data-theme='dark'] .project-overview-toolbar :deep(.el-select .el-input__wrapper) {
  background: rgba(255, 255, 255, 0.06);
}

html.dark .project-overview-toolbar :deep(.el-select .el-input__wrapper:hover),
[data-theme='dark'] .project-overview-toolbar :deep(.el-select .el-input__wrapper:hover) {
  background: rgba(255, 255, 255, 0.1);
}

/* ─── 筛选按钮覆写（综合筛选 / 标签筛选） ─── */
.project-overview-toolbar :deep(.el-button) {
  border-radius: 10px;
  border: none;
  background: rgba(0, 0, 0, 0.04);
  color: var(--ck-text-secondary);
  font-size: 13px;
  font-weight: 400;
  padding: 0 12px;
  height: 32px;
  box-shadow: none;
  transition: all 0.2s;
}

.project-overview-toolbar :deep(.el-button:hover) {
  background: rgba(0, 0, 0, 0.08);
  color: var(--ck-text-primary);
}

.project-overview-toolbar :deep(.el-button:focus) {
  background: rgba(0, 0, 0, 0.06);
  color: var(--ck-text-primary);
}

html.dark .project-overview-toolbar :deep(.el-button),
[data-theme='dark'] .project-overview-toolbar :deep(.el-button) {
  background: rgba(255, 255, 255, 0.06);
  color: var(--ck-text-secondary);
}

html.dark .project-overview-toolbar :deep(.el-button:hover),
[data-theme='dark'] .project-overview-toolbar :deep(.el-button:hover) {
  background: rgba(255, 255, 255, 0.1);
  color: var(--ck-text-primary);
}

/* ─── pill 按钮（排序字段 / 排序方向） ─── */
.project-overview-toolbar__pill-btn {
  display: inline-flex;
  align-items: center;
  gap: 2px;
  padding: 0 12px;
  height: 32px;
  border-radius: 10px;
  border: none;
  background: rgba(0, 0, 0, 0.04);
  color: var(--ck-text-secondary);
  font-size: 13px;
  font-weight: 400;
  cursor: pointer;
  transition: all 0.2s;
  white-space: nowrap;
  font-family: inherit;
}

.project-overview-toolbar__pill-btn:hover {
  background: rgba(0, 0, 0, 0.08);
  color: var(--ck-text-primary);
}

.project-overview-toolbar__sort-icon {
  margin-right: 2px;
  font-size: 14px;
}

html.dark .project-overview-toolbar__pill-btn,
[data-theme='dark'] .project-overview-toolbar__pill-btn {
  background: rgba(255, 255, 255, 0.06);
  color: var(--ck-text-secondary);
}

html.dark .project-overview-toolbar__pill-btn:hover,
[data-theme='dark'] .project-overview-toolbar__pill-btn:hover {
  background: rgba(255, 255, 255, 0.1);
  color: var(--ck-text-primary);
}
</style>

