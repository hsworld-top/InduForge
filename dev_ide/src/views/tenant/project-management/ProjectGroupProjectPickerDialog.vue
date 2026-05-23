<template>
  <el-dialog
    v-model="dialogVisible"
    :title="dialogTitle"
    width="760px"
    :close-on-click-modal="true"
    class="project-group-project-picker"
  >
    <div class="project-group-project-picker__toolbar">
      <el-input
        v-model="keyword"
        data-testid="project-group-picker-search"
        :placeholder="t('projectManagement.groupPickerSearchPlaceholder')"
        clearable
      />
      <el-select
        v-model="selectedTagIds"
        class="project-group-project-picker__tag-filter"
        data-testid="project-group-picker-tag-filter"
        multiple
        clearable
        :placeholder="t('projectManagement.tagFilter')"
      >
        <el-option
          v-for="tag in normalizedTagOptions"
          :key="tag.id"
          :label="tag.name"
          :value="tag.id"
        />
      </el-select>
      <el-select
        v-model="selectedRuntimeModes"
        data-testid="project-group-picker-runtime-filter"
        multiple
        collapse-tags
        clearable
        :placeholder="t('projectManagement.runtimeMode')"
      >
        <el-option
          v-for="option in runtimeModeOptions"
          :key="option.value"
          :label="option.label"
          :value="option.value"
        />
      </el-select>
      <el-select
        v-model="selectedDeployStatuses"
        data-testid="project-group-picker-deploy-filter"
        multiple
        collapse-tags
        clearable
        :placeholder="t('projectManagement.deployStatus')"
      >
        <el-option
          v-for="option in deployStatusOptions"
          :key="option.value"
          :label="option.label"
          :value="option.value"
        />
      </el-select>
    </div>

    <div class="project-group-project-picker__summary">
      <span>{{
        t('projectManagement.groupPickerMatchSummary', { count: filteredProjects.length })
      }}</span>
      <span>{{ t('projectManagement.selectedCount', { count: selectedProjectIds.length }) }}</span>
    </div>

    <div
      v-if="filteredProjects.length > 0"
      v-loading="loading"
      class="project-group-project-picker__list"
    >
      <label
        v-for="project in filteredProjects"
        :key="project.id"
        class="project-group-project-picker__item"
        :class="{ 'is-selected': selectedProjectIds.includes(project.id) }"
        data-testid="project-group-picker-item"
      >
        <input
          type="checkbox"
          class="project-group-project-picker__checkbox"
          data-testid="project-group-picker-checkbox"
          :checked="selectedProjectIds.includes(project.id)"
          @change="toggleProjectSelection(project.id)"
        />
        <span class="project-group-project-picker__item-body">
          <span class="project-group-project-picker__item-name">{{ project.name }}</span>
          <span v-if="project.description" class="project-group-project-picker__item-description">
            {{ project.description }}
          </span>
          <span v-if="project.tags.length > 0" class="project-group-project-picker__tags">
            <el-tag
              v-for="tag in project.tags"
              :key="tag.id"
              size="small"
              effect="plain"
              type="info"
            >
              {{ tag.name }}
            </el-tag>
          </span>
        </span>
      </label>
    </div>
    <el-empty v-else :description="t('projectManagement.groupPickerNoMatchedProjects')" />

    <template #footer>
      <div class="project-group-project-picker__footer">
        <el-button
          data-testid="project-group-picker-select-visible"
          :disabled="filteredProjects.length === 0"
          @click="toggleAllFilteredProjects"
        >
          {{
            t(
              isAllFilteredSelected
                ? 'projectManagement.clearSelection'
                : 'projectManagement.groupPickerSelectVisible',
            )
          }}
        </el-button>
        <div class="project-group-project-picker__footer-actions">
          <el-button @click="dialogVisible = false">{{ t('projectManagement.cancel') }}</el-button>
          <el-button
            type="primary"
            data-testid="project-group-picker-submit"
            :disabled="selectedProjectIds.length === 0"
            @click="submitSelectedProjects"
          >
            {{
              t('projectManagement.groupPickerAddSelected', { count: selectedProjectIds.length })
            }}
          </el-button>
        </div>
      </div>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { ProjectDeployStatus, ProjectRuntimeMode } from '@/api/project.api'
import type { ProjectOverviewItem, ProjectOverviewTag } from './project-overview.types'

type OptionItem = {
  label: string
  value: string
}

type NormalizedProjectCandidate = {
  id: string
  name: string
  description: string
  tags: ProjectOverviewTag[]
  runtimeModes: string[]
  deployStatuses: string[]
}

const props = withDefaults(
  defineProps<{
    visible: boolean
    groupName?: string
    projects?: ProjectOverviewItem[]
    groupId?: string
    tagOptions?: ProjectOverviewTag[]
    runtimeModeOptions?: OptionItem[]
    deployStatusOptions?: OptionItem[]
    loading?: boolean
  }>(),
  {
    groupName: '',
    projects: () => [],
    groupId: '',
    tagOptions: () => [],
    runtimeModeOptions: () => [],
    deployStatusOptions: () => [],
    loading: false,
  },
)

const emit = defineEmits<{
  (event: 'update:visible', value: boolean): void
  (event: 'select', projectIds: string[]): void
  (
    event: 'query-change',
    payload: {
      search: string
      tagIds: string[]
      runtimeModes: ProjectRuntimeMode[]
      deployStatuses: ProjectDeployStatus[]
    },
  ): void
}>()

const { t } = useI18n()
const keyword = ref('')
const selectedTagIds = ref<string[]>([])
const selectedRuntimeModes = ref<ProjectRuntimeMode[]>([])
const selectedDeployStatuses = ref<ProjectDeployStatus[]>([])
const selectedProjectIds = ref<string[]>([])

const dialogVisible = computed({
  get: () => props.visible,
  set: (value: boolean) => emit('update:visible', value),
})

const dialogTitle = computed(() => `${t('projectManagement.addProject')} - ${props.groupName}`)

const normalizeText = (value: unknown) => String(value || '').trim()

const normalizeStringList = (values: unknown): string[] => {
  if (!Array.isArray(values)) {
    return []
  }
  return [...new Set(values.map((item) => normalizeText(item)).filter(Boolean))]
}

const normalizeRuntimeModeValue = (value: unknown): ProjectRuntimeMode | '' => {
  const normalized = normalizeText(value).toUpperCase()
  if (normalized === 'DEV' || normalized === 'DEVELOPMENT') {
    return 'DEV'
  }
  if (normalized === 'RELEASE' || normalized === 'PROD' || normalized === 'PRODUCTION') {
    return 'RELEASE'
  }
  return ''
}

const normalizeTags = (tags: unknown): ProjectOverviewTag[] => {
  if (!Array.isArray(tags)) {
    return []
  }
  return tags
    .map((tag) => {
      const id = normalizeText((tag as ProjectOverviewTag)?.id)
      const name = normalizeText((tag as ProjectOverviewTag)?.name)
      return id ? { ...(tag as ProjectOverviewTag), id, name: name || id } : null
    })
    .filter(Boolean) as ProjectOverviewTag[]
}

const normalizeRuntimeModes = (project: ProjectOverviewItem): string[] => {
  const modeCounts = project.runtimeSummary?.modeCounts || {}
  const modes = ['DEV', 'RELEASE'].filter((mode) =>
    Object.entries(modeCounts).some(
      ([key, value]) => normalizeRuntimeModeValue(key) === mode && Number(value || 0) > 0,
    ),
  )
  const fallbackModes = normalizeStringList([project.runtimeSummary?.runtimeMode])
    .map((mode) => normalizeRuntimeModeValue(mode))
    .filter(Boolean)
  return [...new Set([...modes, ...fallbackModes])]
}

const normalizeDeployStatuses = (project: ProjectOverviewItem): string[] => {
  const statusCounts = project.runtimeSummary?.statusCounts || {}
  const statuses = Object.entries(statusCounts)
    .filter(([, count]) => Number(count || 0) > 0)
    .map(([status]) => status.toLowerCase())
  const runtimeStatus = normalizeText(project.runtimeSummary?.runtimeStatus).toLowerCase()
  return statuses.length > 0 ? statuses : normalizeStringList([runtimeStatus])
}

const normalizedTagOptions = computed(() => normalizeTags(props.tagOptions))

const availableProjects = computed(() =>
  [...props.projects]
    .filter((project) => {
      const projectId = String(project?.id || '').trim()
      const projectGroupId = String(project?.group?.id || '').trim()
      return projectId && projectGroupId !== props.groupId
    })
    .map<NormalizedProjectCandidate>((project) => ({
      id: String(project.id),
      name: normalizeText(project.name) || String(project.id),
      description: normalizeText(project.description),
      tags: normalizeTags(project.tags),
      runtimeModes: normalizeRuntimeModes(project),
      deployStatuses: normalizeDeployStatuses(project),
    })),
)

const filteredProjects = computed(() => {
  const normalizedKeyword = keyword.value.trim().toLowerCase()
  const tagIdSet = new Set(selectedTagIds.value)
  const runtimeModeSet = new Set(selectedRuntimeModes.value)
  const deployStatusSet = new Set(selectedDeployStatuses.value)

  return availableProjects.value.filter((project) => {
    const matchesKeyword =
      !normalizedKeyword ||
      project.name.toLowerCase().includes(normalizedKeyword) ||
      project.description.toLowerCase().includes(normalizedKeyword)
    const matchesTag = tagIdSet.size === 0 || project.tags.some((tag) => tagIdSet.has(tag.id))
    const matchesRuntimeMode =
      runtimeModeSet.size === 0 ||
      project.runtimeModes.some((mode) => runtimeModeSet.has(mode as ProjectRuntimeMode))
    const matchesDeployStatus =
      deployStatusSet.size === 0 ||
      project.deployStatuses.some((status) => deployStatusSet.has(status as ProjectDeployStatus))

    return matchesKeyword && matchesTag && matchesRuntimeMode && matchesDeployStatus
  })
})

const isAllFilteredSelected = computed(
  () =>
    filteredProjects.value.length > 0 &&
    filteredProjects.value.every((project) => selectedProjectIds.value.includes(project.id)),
)

const emitQueryChange = () => {
  emit('query-change', {
    search: keyword.value.trim(),
    tagIds: [...selectedTagIds.value],
    runtimeModes: [...selectedRuntimeModes.value],
    deployStatuses: [...selectedDeployStatuses.value],
  })
}

const toggleProjectSelection = (projectId: string) => {
  if (selectedProjectIds.value.includes(projectId)) {
    selectedProjectIds.value = selectedProjectIds.value.filter((id) => id !== projectId)
    return
  }
  selectedProjectIds.value = [...selectedProjectIds.value, projectId]
}

const toggleAllFilteredProjects = () => {
  const visibleIds = filteredProjects.value.map((project) => project.id)
  if (isAllFilteredSelected.value) {
    selectedProjectIds.value = selectedProjectIds.value.filter((id) => !visibleIds.includes(id))
    return
  }
  selectedProjectIds.value = [...new Set([...selectedProjectIds.value, ...visibleIds])]
}

const submitSelectedProjects = () => {
  if (selectedProjectIds.value.length === 0) {
    return
  }
  emit('select', [...selectedProjectIds.value])
}

watch([keyword, selectedTagIds, selectedRuntimeModes, selectedDeployStatuses], emitQueryChange, {
  deep: true,
})

watch(
  () => props.visible,
  (visible) => {
    if (visible) {
      keyword.value = ''
      selectedTagIds.value = []
      selectedRuntimeModes.value = []
      selectedDeployStatuses.value = []
      selectedProjectIds.value = []
      return
    }
    selectedProjectIds.value = []
  },
)

watch(
  availableProjects,
  (projects) => {
    const availableIdSet = new Set(projects.map((project) => project.id))
    selectedProjectIds.value = selectedProjectIds.value.filter((id) => availableIdSet.has(id))
  },
  { immediate: true },
)
</script>

<style scoped>
.project-group-project-picker__toolbar {
  display: grid;
  grid-template-columns: minmax(180px, 1.4fr) repeat(3, minmax(130px, 1fr));
  gap: 10px;
  margin-bottom: 12px;
}

.project-group-project-picker__tag-filter :deep(.el-select__wrapper) {
  height: auto;
  min-height: 32px;
  align-items: flex-start;
  padding-top: 4px;
  padding-bottom: 4px;
}

.project-group-project-picker__tag-filter :deep(.el-select__selection) {
  flex-wrap: wrap;
  align-items: center;
  gap: 4px;
  max-height: 72px;
  overflow-y: auto;
}

.project-group-project-picker__tag-filter :deep(.el-select__selected-item) {
  max-width: 100%;
}

.project-group-project-picker__summary {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 10px;
  color: #64748b;
  font-size: 12px;
}

.project-group-project-picker__list {
  display: flex;
  max-height: 420px;
  flex-direction: column;
  gap: 8px;
  overflow-y: auto;
}

.project-group-project-picker__item {
  width: 100%;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  background: #ffffff;
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 12px;
  color: #1f2937;
  cursor: pointer;
  text-align: left;
  transition:
    border-color 0.2s ease,
    background 0.2s ease;
}

.project-group-project-picker__item:hover,
.project-group-project-picker__item.is-selected {
  border-color: #93c5fd;
  background: #f8fbff;
}

.project-group-project-picker__checkbox {
  margin-top: 3px;
  flex: 0 0 auto;
}

.project-group-project-picker__item-body {
  display: grid;
  min-width: 0;
  gap: 5px;
}

.project-group-project-picker__item-name {
  overflow: hidden;
  color: #111827;
  font-size: 14px;
  font-weight: 600;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.project-group-project-picker__item-description {
  overflow: hidden;
  color: #64748b;
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.project-group-project-picker__tags {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}

.project-group-project-picker__footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.project-group-project-picker__footer-actions {
  display: inline-flex;
  align-items: center;
  gap: 8px;
}

html.dark .project-group-project-picker__item,
[data-theme='dark'] .project-group-project-picker__item {
  border-color: #374151;
  background: #111827;
  color: #e5e7eb;
}

html.dark .project-group-project-picker__item-name,
[data-theme='dark'] .project-group-project-picker__item-name {
  color: #e5e7eb;
}

html.dark .project-group-project-picker__item-description,
[data-theme='dark'] .project-group-project-picker__item-description,
html.dark .project-group-project-picker__summary,
[data-theme='dark'] .project-group-project-picker__summary {
  color: #94a3b8;
}

@media (max-width: 760px) {
  .project-group-project-picker__toolbar {
    grid-template-columns: 1fr;
  }

  .project-group-project-picker__footer {
    align-items: stretch;
    flex-direction: column;
  }

  .project-group-project-picker__footer-actions {
    justify-content: flex-end;
  }
}
</style>
