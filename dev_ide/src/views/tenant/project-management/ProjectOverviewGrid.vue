<template>
  <section class="project-overview-grid">
    <div v-if="loading" class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-4 gap-4">
      <el-skeleton v-for="index in 4" :key="index" animated class="rounded-xl border border-gray-100 p-4">
        <template #template>
          <el-skeleton-item variant="h3" class="!w-2/3 !h-5" />
          <el-skeleton-item variant="text" class="!w-full !h-3 mt-3" />
          <el-skeleton-item variant="text" class="!w-5/6 !h-3 mt-2" />
          <div class="mt-6 flex justify-between items-center">
            <el-skeleton-item variant="text" class="!w-1/3 !h-3" />
            <div class="flex gap-2">
              <el-skeleton-item variant="circle" class="!w-6 !h-6" />
              <el-skeleton-item variant="circle" class="!w-6 !h-6" />
              <el-skeleton-item variant="circle" class="!w-6 !h-6" />
            </div>
          </div>
        </template>
      </el-skeleton>
    </div>

    <el-empty v-else-if="projects.length === 0" :description="resolvedEmptyDescription" />

    <div v-else class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-4 gap-4">
      <article
        v-for="project in projects"
        :key="project.id"
        class="rounded-xl border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 p-4 flex flex-col shadow-sm hover:shadow-md transition-shadow"
      >
        <header class="flex items-start justify-between gap-2">
          <div class="flex items-center gap-2 min-w-0">
            <el-checkbox
              v-if="showSelection"
              :model-value="isSelected(project.id)"
              @change="(value) => handleSelectionChange(project.id, value)"
            />
            <button
              type="button"
              class="text-left min-w-0 text-sm font-semibold text-gray-800 dark:text-gray-100 truncate hover:text-blue-600"
              :title="project.name"
              @click="emit('open-project', project)"
            >
              {{ project.name }}
            </button>
          </div>
          <el-tag size="small" :type="resolveRuntimeTag(project.runtimeSummary.runtimeStatus)">
            {{ resolveRuntimeText(project.runtimeSummary.runtimeStatus) }}
          </el-tag>
        </header>

        <p class="text-xs text-gray-500 dark:text-gray-400 mt-3 line-clamp-2 min-h-[32px]">
          {{ project.description || t('projectManagement.noDescription') }}
        </p>

        <div class="mt-3 flex flex-wrap gap-1.5 min-h-[22px]">
          <el-tag
            v-for="tag in project.tags"
            :key="tag.id"
            size="small"
            effect="plain"
            class="!mr-0"
            type="info"
          >
            {{ tag.name }}
          </el-tag>
          <span
            v-if="project.tags.length === 0"
            class="text-[11px] text-gray-400 dark:text-gray-500 leading-5"
          >
            {{ t('projectManagement.noTags') }}
          </span>
        </div>

        <div
          class="mt-4 pt-3 border-t border-gray-100 dark:border-gray-700 flex items-center justify-between gap-2"
        >
          <span class="text-[11px] text-gray-400 dark:text-gray-500">
            {{ resolveDisplayTime(project) }}
          </span>

          <div class="flex items-center gap-1">
            <slot name="actions" :project="project">
              <el-tooltip v-if="showMemberAction" :content="t('projectManagement.memberAndPermission')" placement="top">
                <el-button
                  size="small"
                  text
                  circle
                  class="!w-7 !h-7"
                  @click.stop="emit('open-runtime-access', project)"
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
                  @click.stop="emit('deploy', project)"
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
                  @click.stop="emit('export', project)"
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
                  @click.stop="emit('delete', project)"
                >
                  <el-icon><Delete /></el-icon>
                </el-button>
              </el-tooltip>
            </slot>
          </div>
        </div>
      </article>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import dayjs from 'dayjs'
import { Delete, Download, UploadFilled, User } from '@element-plus/icons-vue'
import type { TagProps } from 'element-plus'
import { useI18n } from 'vue-i18n'
import type { ProjectOverviewItem } from './project-overview.types'

const props = withDefaults(
  defineProps<{
    projects?: ProjectOverviewItem[]
    loading?: boolean
    emptyDescription?: string
    selectedIds?: string[]
    showSelection?: boolean
    showMemberAction?: boolean
    showDeployAction?: boolean
    showExportAction?: boolean
    showDeleteAction?: boolean
  }>(),
  {
    projects: () => [],
    loading: false,
    emptyDescription: '',
    selectedIds: () => [],
    showSelection: false,
    showMemberAction: true,
    showDeployAction: true,
    showExportAction: true,
    showDeleteAction: true,
  },
)

const { t } = useI18n()

const resolvedEmptyDescription = computed(
  () => props.emptyDescription || t('projectManagement.emptyProjects'),
)

const emit = defineEmits<{
  (event: 'open-project', payload: ProjectOverviewItem): void
  (event: 'selection-change', payload: { projectId: string; selected: boolean }): void
  (event: 'open-runtime-access', payload: ProjectOverviewItem): void
  (event: 'deploy', payload: ProjectOverviewItem): void
  (event: 'export', payload: ProjectOverviewItem): void
  (event: 'delete', payload: ProjectOverviewItem): void
}>()

const selectedIdSet = computed(() => new Set(props.selectedIds))

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

const resolveDisplayTime = (project: ProjectOverviewItem) => {
  const candidate =
    project.lastDeployedAt || project.updatedAt || project.createdAt || project.runtimeSummary.lastDeployedAt
  if (!candidate) {
    return '--'
  }
  const parsed = dayjs(candidate)
  return parsed.isValid() ? parsed.format('YYYY-MM-DD HH:mm:ss') : '--'
}

const isSelected = (projectId: string) => selectedIdSet.value.has(projectId)

const handleSelectionChange = (projectId: string, value: string | number | boolean) => {
  emit('selection-change', {
    projectId,
    selected: Boolean(value),
  })
}
</script>
